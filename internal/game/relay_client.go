package game

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
	"golang.org/x/net/websocket"

	"gunmayhem-go/internal/masterapi"
)

const relayReconnectDelay = 2 * time.Second

type relayHostTunnel struct {
	address string
	id      string
	secret  string
	session *netplaySession

	closed atomic.Bool
	ready  atomic.Bool
	stop   chan struct{}
	done   chan struct{}

	mu      sync.Mutex
	conn    net.Conn
	mux     *yamux.Session
	lastErr error
}

func gameRelayYamuxConfig() *yamux.Config {
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

func startRelayHostTunnel(address, id, secret string, session *netplaySession) *relayHostTunnel {
	t := &relayHostTunnel{
		address: strings.TrimSpace(address),
		id:      strings.TrimSpace(id),
		secret:  strings.TrimSpace(secret),
		session: session,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go t.loop()
	return t
}

func (t *relayHostTunnel) loop() {
	defer close(t.done)
	for !t.closed.Load() {
		if err := t.runOnce(); err != nil && !t.closed.Load() {
			t.setError(err)
		}
		if t.closed.Load() {
			return
		}
		timer := time.NewTimer(relayReconnectDelay)
		select {
		case <-timer.C:
		case <-t.stop:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		}
	}
}

func (t *relayHostTunnel) runOnce() error {
	if t == nil || t.session == nil || t.address == "" || t.id == "" || t.secret == "" {
		return errors.New("incomplete relay host configuration")
	}
	conn, err := dialRelayTransport(t.address, 5*time.Second)
	if err != nil {
		return err
	}
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetNoDelay(true)
	}
	if err := masterapi.WriteRelayHandshake(conn, masterapi.RelayHandshake{
		Role:     masterapi.RelayRoleHost,
		ID:       t.id,
		Protocol: netplayProtocolVersion,
	}); err != nil {
		_ = conn.Close()
		return err
	}
	challengeAck, err := masterapi.ReadRelayAck(conn)
	if err != nil {
		_ = conn.Close()
		return err
	}
	if !challengeAck.OK || strings.TrimSpace(challengeAck.Challenge) == "" {
		_ = conn.Close()
		if challengeAck.Error == "" {
			challengeAck.Error = "relay rejected host"
		}
		return errors.New(challengeAck.Error)
	}
	proof := masterapi.RelayProofForSecret(t.secret, challengeAck.Challenge)
	if proof == "" {
		_ = conn.Close()
		return errors.New("cannot create relay authentication proof")
	}
	if err := masterapi.WriteRelayProof(conn, masterapi.RelayProof{Proof: proof}); err != nil {
		_ = conn.Close()
		return err
	}
	ack, err := masterapi.ReadRelayAck(conn)
	if err != nil {
		_ = conn.Close()
		return err
	}
	if !ack.OK {
		_ = conn.Close()
		if ack.Error == "" {
			ack.Error = "relay rejected host proof"
		}
		return errors.New(ack.Error)
	}
	mux, err := yamux.Client(conn, gameRelayYamuxConfig())
	if err != nil {
		_ = conn.Close()
		return err
	}

	t.mu.Lock()
	if t.closed.Load() {
		t.mu.Unlock()
		_ = mux.Close()
		return net.ErrClosed
	}
	t.conn = conn
	t.mux = mux
	t.lastErr = nil
	t.mu.Unlock()
	t.ready.Store(true)
	defer func() {
		t.ready.Store(false)
		t.mu.Lock()
		if t.mux == mux {
			t.mux = nil
		}
		if t.conn == conn {
			t.conn = nil
		}
		t.mu.Unlock()
		_ = mux.Close()
		_ = conn.Close()
	}()

	for !t.closed.Load() {
		stream, err := mux.Accept()
		if err != nil {
			return err
		}
		if !t.session.attachIncomingConn(stream) {
			_ = stream.Close()
		}
	}
	return nil
}

func (t *relayHostTunnel) setError(err error) {
	if t == nil || err == nil {
		return
	}
	t.mu.Lock()
	t.lastErr = err
	t.mu.Unlock()
}

func (t *relayHostTunnel) Status() (ready bool, err error) {
	if t == nil {
		return false, nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ready.Load(), t.lastErr
}

func (t *relayHostTunnel) Close() error {
	if t == nil || t.closed.Swap(true) {
		return nil
	}
	close(t.stop)
	t.mu.Lock()
	mux := t.mux
	conn := t.conn
	t.mu.Unlock()
	if mux != nil {
		_ = mux.Close()
	}
	if conn != nil {
		_ = conn.Close()
	}
	return nil
}

func dialRelayNetplay(relayAddress, serverID string) (net.Conn, error) {
	relayAddress = strings.TrimSpace(relayAddress)
	serverID = strings.TrimSpace(serverID)
	if relayAddress == "" || serverID == "" || len(serverID) > 96 {
		return nil, errors.New("invalid relay server data")
	}
	if normalizeRelayAddress(relayAddress) == "" {
		return nil, fmt.Errorf("invalid relay address")
	}
	conn, err := dialRelayTransport(relayAddress, 5*time.Second)
	if err != nil {
		return nil, err
	}
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetNoDelay(true)
	}
	if err := masterapi.WriteRelayHandshake(conn, masterapi.RelayHandshake{
		Role:     masterapi.RelayRoleClient,
		ID:       serverID,
		Protocol: netplayProtocolVersion,
	}); err != nil {
		_ = conn.Close()
		return nil, err
	}
	ack, err := masterapi.ReadRelayAck(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if !ack.OK {
		_ = conn.Close()
		if ack.Error == "" {
			ack.Error = "relay rejected client"
		}
		return nil, errors.New(ack.Error)
	}
	return conn, nil
}

func normalizeRelayAddress(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" || (u.Scheme != "wss" && u.Scheme != "ws") {
			return ""
		}
		if u.Scheme == "ws" {
			host := strings.TrimSpace(u.Hostname())
			ip := net.ParseIP(host)
			if !strings.EqualFold(host, "localhost") && (ip == nil || (!ip.IsLoopback() && !ip.IsPrivate())) {
				return ""
			}
		}
		return u.String()
	}
	if _, _, err := net.SplitHostPort(raw); err != nil {
		return ""
	}
	return raw
}

func dialRelayTransport(address string, timeout time.Duration) (net.Conn, error) {
	address = normalizeRelayAddress(address)
	if address == "" {
		return nil, errors.New("invalid relay address")
	}
	if !strings.Contains(address, "://") {
		return net.DialTimeout("tcp", address, timeout)
	}
	config, err := websocket.NewConfig(address, "https://gunmayhem.invalid/")
	if err != nil {
		return nil, err
	}
	config.Protocol = []string{"gunmayhem-relay-v1"}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := config.DialContext(ctx)
	if err != nil {
		return nil, err
	}
	conn.PayloadType = websocket.BinaryFrame
	return conn, nil
}
