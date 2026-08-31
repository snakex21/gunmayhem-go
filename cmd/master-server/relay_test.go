package main

import (
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/yamux"
	"golang.org/x/net/websocket"

	"gunmayhem-go/internal/masterapi"
)

func startTestRelayHub(t *testing.T, reg *serverRegistry) (*relayHub, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	hub := newRelayHub(reg)
	reg.relay = hub
	go func() {
		_ = hub.serveListener(ln)
	}()
	t.Cleanup(func() { _ = hub.Close() })
	return hub, ln.Addr().String()
}

func TestRelayBridgesThroughWebSocket(t *testing.T) {
	reg := newServerRegistry(35 * time.Second)
	const id = "websocket-server"
	const secret = "0123456789abcdef0123456789abcdef"
	reg.entries[id] = registryEntry{
		server: masterapi.Server{ID: id, Protocol: 15},
		seen:   time.Now(), secret: secret,
	}
	hub := newRelayHub(reg)
	reg.relay = hub
	ts := httptest.NewServer(newHandler(reg))
	defer ts.Close()
	defer hub.Close()
	relayURL := "ws" + strings.TrimPrefix(ts.URL, "http") + masterapi.RelayWebSocketPath
	dial := func() *websocket.Conn {
		conn, err := websocket.Dial(relayURL, "gunmayhem-relay-v1", "http://localhost/")
		if err != nil {
			t.Fatal(err)
		}
		conn.PayloadType = websocket.BinaryFrame
		return conn
	}

	hostConn := dial()
	if err := masterapi.WriteRelayHandshake(hostConn, masterapi.RelayHandshake{Role: masterapi.RelayRoleHost, ID: id, Protocol: 15}); err != nil {
		t.Fatal(err)
	}
	challenge, err := masterapi.ReadRelayAck(hostConn)
	if err != nil || !challenge.OK {
		t.Fatalf("challenge=%+v err=%v", challenge, err)
	}
	if err := masterapi.WriteRelayProof(hostConn, masterapi.RelayProof{Proof: masterapi.RelayProofForSecret(secret, challenge.Challenge)}); err != nil {
		t.Fatal(err)
	}
	if ack, err := masterapi.ReadRelayAck(hostConn); err != nil || !ack.OK {
		t.Fatalf("host ack=%+v err=%v", ack, err)
	}
	hostMux, err := yamux.Client(hostConn, relayYamuxConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer hostMux.Close()

	streamCh := make(chan net.Conn, 1)
	go func() {
		stream, _ := hostMux.Accept()
		streamCh <- stream
	}()
	clientConn := dial()
	defer clientConn.Close()
	if err := masterapi.WriteRelayHandshake(clientConn, masterapi.RelayHandshake{Role: masterapi.RelayRoleClient, ID: id, Protocol: 15}); err != nil {
		t.Fatal(err)
	}
	if ack, err := masterapi.ReadRelayAck(clientConn); err != nil || !ack.OK {
		t.Fatalf("client ack=%+v err=%v", ack, err)
	}
	var stream net.Conn
	select {
	case stream = <-streamCh:
	case <-time.After(2 * time.Second):
		t.Fatal("WebSocket relay did not open host stream")
	}
	if stream == nil {
		t.Fatal("WebSocket relay returned nil stream")
	}
	defer stream.Close()
	if _, err := clientConn.Write([]byte("vpn-safe")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len("vpn-safe"))
	if _, err := io.ReadFull(stream, buf); err != nil || string(buf) != "vpn-safe" {
		t.Fatalf("WebSocket relay payload=%q err=%v", buf, err)
	}
}

func TestRelayBridgesAuthenticatedHostAndClient(t *testing.T) {
	reg := newServerRegistry(35 * time.Second)
	const id = "test-server"
	const secret = "0123456789abcdef0123456789abcdef"
	reg.entries[id] = registryEntry{
		server: masterapi.Server{ID: id, Name: "Test", Address: "203.0.113.1:7777", Players: 1, MaxPlayers: 4, Protocol: 4},
		seen:   time.Now(),
		secret: secret,
	}
	_, address := startTestRelayHub(t, reg)

	hostConn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := masterapi.WriteRelayHandshake(hostConn, masterapi.RelayHandshake{Role: masterapi.RelayRoleHost, ID: id, Protocol: 4}); err != nil {
		t.Fatal(err)
	}
	challengeAck, err := masterapi.ReadRelayAck(hostConn)
	if err != nil || !challengeAck.OK || challengeAck.Challenge == "" {
		t.Fatalf("host relay challenge=%+v err=%v", challengeAck, err)
	}
	proof := masterapi.RelayProofForSecret(secret, challengeAck.Challenge)
	if err := masterapi.WriteRelayProof(hostConn, masterapi.RelayProof{Proof: proof}); err != nil {
		t.Fatal(err)
	}
	ack, err := masterapi.ReadRelayAck(hostConn)
	if err != nil || !ack.OK {
		t.Fatalf("host relay ack=%+v err=%v", ack, err)
	}
	hostMux, err := yamux.Client(hostConn, relayYamuxConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer hostMux.Close()

	streamCh := make(chan net.Conn, 1)
	errCh := make(chan error, 1)
	go func() {
		stream, err := hostMux.Accept()
		if err != nil {
			errCh <- err
			return
		}
		streamCh <- stream
	}()

	clientConn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()
	if err := masterapi.WriteRelayHandshake(clientConn, masterapi.RelayHandshake{Role: masterapi.RelayRoleClient, ID: id, Protocol: 4}); err != nil {
		t.Fatal(err)
	}
	clientAck, err := masterapi.ReadRelayAck(clientConn)
	if err != nil || !clientAck.OK {
		t.Fatalf("client relay ack=%+v err=%v", clientAck, err)
	}

	var stream net.Conn
	select {
	case stream = <-streamCh:
	case err := <-errCh:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("relay stream was not opened")
	}
	defer stream.Close()

	if _, err := clientConn.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(stream, buf); err != nil || string(buf) != "ping" {
		t.Fatalf("client -> host got %q err=%v", string(buf), err)
	}
	if _, err := stream.Write([]byte("pong")); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(clientConn, buf); err != nil || string(buf) != "pong" {
		t.Fatalf("host -> client got %q err=%v", string(buf), err)
	}
}

func TestRelayRejectsWrongHostSecret(t *testing.T) {
	reg := newServerRegistry(35 * time.Second)
	reg.entries["server"] = registryEntry{
		server: masterapi.Server{ID: "server", Protocol: 4},
		seen:   time.Now(),
		secret: "correct-secret",
	}
	_, address := startTestRelayHub(t, reg)

	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := masterapi.WriteRelayHandshake(conn, masterapi.RelayHandshake{Role: masterapi.RelayRoleHost, ID: "server", Protocol: 4}); err != nil {
		t.Fatal(err)
	}
	challengeAck, err := masterapi.ReadRelayAck(conn)
	if err != nil || !challengeAck.OK || challengeAck.Challenge == "" {
		t.Fatalf("relay challenge=%+v err=%v", challengeAck, err)
	}
	wrongProof := masterapi.RelayProofForSecret("wrong-secret", challengeAck.Challenge)
	if err := masterapi.WriteRelayProof(conn, masterapi.RelayProof{Proof: wrongProof}); err != nil {
		t.Fatal(err)
	}
	ack, err := masterapi.ReadRelayAck(conn)
	if err != nil {
		t.Fatal(err)
	}
	if ack.OK {
		t.Fatal("relay accepted a host with the wrong secret")
	}
}
