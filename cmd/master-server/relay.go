package main

import (
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"

	"gunmayhem-go/internal/masterapi"
)

const (
	maxRelayClientsPerServer  = 4
	maxRelayPendingHandshakes = 256
)

type relayHub struct {
	registry *serverRegistry
	closed   atomic.Bool

	mu       sync.RWMutex
	hosts    map[string]*yamux.Session
	active   map[string]int
	pending  chan struct{}
	listener net.Listener
}

func newRelayHub(registry *serverRegistry) *relayHub {
	return &relayHub{
		registry: registry,
		hosts:    make(map[string]*yamux.Session),
		active:   make(map[string]int),
		pending:  make(chan struct{}, maxRelayPendingHandshakes),
	}
}

func relayYamuxConfig() *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.AcceptBacklog = 8
	cfg.EnableKeepAlive = true
	cfg.KeepAliveInterval = 15 * time.Second
	cfg.ConnectionWriteTimeout = 5 * time.Second
	cfg.MaxStreamWindowSize = 256 * 1024
	cfg.StreamOpenTimeout = 5 * time.Second
	cfg.StreamCloseTimeout = 5 * time.Second
	cfg.LogOutput = io.Discard
	return cfg
}

func (h *relayHub) Serve(addr string) error {
	if h == nil {
		return errors.New("nil relay hub")
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return h.serveListener(ln)
}

func (h *relayHub) serveListener(ln net.Listener) error {
	if h == nil || ln == nil {
		return errors.New("invalid relay listener")
	}
	h.listener = ln
	for !h.closed.Load() {
		conn, err := ln.Accept()
		if err != nil {
			if h.closed.Load() {
				return nil
			}
			return err
		}
		if tcp, ok := conn.(*net.TCPConn); ok {
			_ = tcp.SetNoDelay(true)
		}
		select {
		case h.pending <- struct{}{}:
			go func() {
				defer func() { <-h.pending }()
				h.handle(conn)
			}()
		default:
			_ = conn.Close()
		}
	}
	return nil
}

func (h *relayHub) handle(conn net.Conn) {
	hs, err := masterapi.ReadRelayHandshake(conn)
	if err != nil {
		_ = conn.Close()
		return
	}
	if len(hs.ID) == 0 || len(hs.ID) > 96 || hs.Protocol <= 0 {
		_ = masterapi.WriteRelayAck(conn, masterapi.RelayAck{OK: false, Error: "invalid relay request"})
		_ = conn.Close()
		return
	}
	switch hs.Role {
	case masterapi.RelayRoleHost:
		h.handleHost(conn, hs)
	case masterapi.RelayRoleClient:
		h.handleClient(conn, hs)
	default:
		_ = masterapi.WriteRelayAck(conn, masterapi.RelayAck{OK: false, Error: "invalid relay role"})
		_ = conn.Close()
	}
}

func (h *relayHub) handleWebSocket(conn net.Conn) {
	if h == nil || conn == nil || h.closed.Load() {
		if conn != nil {
			_ = conn.Close()
		}
		return
	}
	select {
	case h.pending <- struct{}{}:
		defer func() { <-h.pending }()
		h.handle(conn)
	default:
		_ = conn.Close()
	}
}

func (h *relayHub) handleHost(conn net.Conn, hs masterapi.RelayHandshake) {
	if h.registry == nil {
		_ = masterapi.WriteRelayAck(conn, masterapi.RelayAck{OK: false, Error: "relay authentication failed"})
		_ = conn.Close()
		return
	}
	secret, ok := h.registry.relayHostSecret(hs.ID, hs.Protocol)
	if !ok {
		_ = masterapi.WriteRelayAck(conn, masterapi.RelayAck{OK: false, Error: "relay authentication failed"})
		_ = conn.Close()
		return
	}
	challenge, err := masterapi.NewRelayChallenge()
	if err != nil {
		_ = masterapi.WriteRelayAck(conn, masterapi.RelayAck{OK: false, Error: "relay authentication unavailable"})
		_ = conn.Close()
		return
	}
	if err := masterapi.WriteRelayAck(conn, masterapi.RelayAck{OK: true, Challenge: challenge}); err != nil {
		_ = conn.Close()
		return
	}
	proof, err := masterapi.ReadRelayProof(conn)
	if err != nil || !masterapi.VerifyRelayProof(secret, challenge, proof.Proof) {
		_ = masterapi.WriteRelayAck(conn, masterapi.RelayAck{OK: false, Error: "relay authentication failed"})
		_ = conn.Close()
		return
	}
	if err := masterapi.WriteRelayAck(conn, masterapi.RelayAck{OK: true}); err != nil {
		_ = conn.Close()
		return
	}
	session, err := yamux.Server(conn, relayYamuxConfig())
	if err != nil {
		_ = conn.Close()
		return
	}

	h.mu.Lock()
	old := h.hosts[hs.ID]
	h.hosts[hs.ID] = session
	h.mu.Unlock()
	if old != nil && old != session {
		_ = old.Close()
	}

	<-session.CloseChan()
	h.mu.Lock()
	if h.hosts[hs.ID] == session {
		delete(h.hosts, hs.ID)
	}
	h.mu.Unlock()
}

func (h *relayHub) handleClient(conn net.Conn, hs masterapi.RelayHandshake) {
	if h.registry == nil || !h.registry.relayClientAllowed(hs.ID, hs.Protocol) {
		_ = masterapi.WriteRelayAck(conn, masterapi.RelayAck{OK: false, Error: "server unavailable"})
		_ = conn.Close()
		return
	}

	h.mu.Lock()
	session := h.hosts[hs.ID]
	if session == nil || h.active[hs.ID] >= maxRelayClientsPerServer {
		h.mu.Unlock()
		_ = masterapi.WriteRelayAck(conn, masterapi.RelayAck{OK: false, Error: "relay host unavailable"})
		_ = conn.Close()
		return
	}
	h.active[hs.ID]++
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		if h.active[hs.ID] > 1 {
			h.active[hs.ID]--
		} else {
			delete(h.active, hs.ID)
		}
		h.mu.Unlock()
	}()

	stream, err := session.Open()
	if err != nil {
		_ = masterapi.WriteRelayAck(conn, masterapi.RelayAck{OK: false, Error: "relay stream unavailable"})
		_ = conn.Close()
		return
	}
	if err := masterapi.WriteRelayAck(conn, masterapi.RelayAck{OK: true}); err != nil {
		_ = stream.Close()
		_ = conn.Close()
		return
	}
	proxyRelay(conn, stream)
}

func proxyRelay(a, b net.Conn) {
	done := make(chan struct{}, 2)
	copyOne := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		if tcp, ok := dst.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
		done <- struct{}{}
	}
	go copyOne(a, b)
	go copyOne(b, a)
	<-done
	_ = a.Close()
	_ = b.Close()
	<-done
}

func (h *relayHub) dropHost(id string) {
	if h == nil || strings.TrimSpace(id) == "" {
		return
	}
	h.mu.Lock()
	session := h.hosts[id]
	delete(h.hosts, id)
	delete(h.active, id)
	h.mu.Unlock()
	if session != nil {
		_ = session.Close()
	}
}

func (h *relayHub) Close() error {
	if h == nil || h.closed.Swap(true) {
		return nil
	}
	if h.listener != nil {
		_ = h.listener.Close()
	}
	h.mu.Lock()
	hosts := make([]*yamux.Session, 0, len(h.hosts))
	for _, session := range h.hosts {
		hosts = append(hosts, session)
	}
	h.hosts = make(map[string]*yamux.Session)
	h.active = make(map[string]int)
	h.mu.Unlock()
	for _, session := range hosts {
		_ = session.Close()
	}
	log.Printf("relay hub stopped")
	return nil
}
