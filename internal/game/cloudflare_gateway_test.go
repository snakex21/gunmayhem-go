package game

import (
	"context"
	"encoding/binary"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func testCloudflareGateway(t *testing.T) (*netplaySession, *cloudflareQuickTunnel, string) {
	t.Helper()
	key, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	host, err := newConfiguredHostNetplay("127.0.0.1:0", nil, func(s *netplaySession) { s.inviteOnly = true; s.expectedSessionKey = key })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { host.Close() })
	gateway, err := startCloudflareGateway(host)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { gateway.Close() })
	return host, gateway, key
}

func dialTestGateway(t *testing.T, gateway *cloudflareQuickTunnel) *websocket.Conn {
	t.Helper()
	config, err := websocket.NewConfig("ws://"+gateway.listener.Addr().String()+cloudflareNetplayPath, "https://gunmayhem.invalid/")
	if err != nil {
		t.Fatal(err)
	}
	config.Protocol = []string{cloudflareNetplayProtocol}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := config.DialContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	conn.PayloadType = websocket.BinaryFrame
	return conn
}

func TestCloudflareGatewayWSSAuthenticatedInputAndSnapshots(t *testing.T) {
	host, gateway, key := testCloudflareGateway(t)
	// Real local TLS + WebSocket upgrade + the production gateway handler.
	// Trust only the test server's generated certificate, without InsecureSkipVerify.
	tlsServer := httptest.NewTLSServer(gateway.server.Handler)
	defer tlsServer.Close()
	config, err := websocket.NewConfig("wss"+strings.TrimPrefix(tlsServer.URL, "https")+cloudflareNetplayPath, "https://gunmayhem.invalid/")
	if err != nil {
		t.Fatal(err)
	}
	config.Protocol = []string{cloudflareNetplayProtocol}
	config.TlsConfig = tlsServer.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	conn, err := config.DialContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	conn.PayloadType = websocket.BinaryFrame
	client, err := newClientNetplayFromConnWithKey(newTrackedWebSocketConn(conn), "local TLS gateway", key)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	waitFor(t, "WSS handshake", client.connected)
	client.queueInput(PlayerInputState{Right: true, Grenade: true})
	waitFor(t, "WSS input", func() bool { return host.latestInputFor(2).Right && host.latestInputFor(2).Grenade })
	host.queueState(netSnapshot{TotalLives: 11, Players: []Player{{ID: 1, Name: "Host", X: 44}, {ID: 2, Name: "Guest", X: 88}}})
	waitFor(t, "WSS authoritative snapshot", func() bool {
		s, ok := client.latestState()
		return ok && s.TotalLives == 11 && len(s.Players) == 2 && s.Players[1].X == 88
	})
	if err := gateway.Close(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "WSS disconnect on gateway close", func() bool { return !client.connected() && host.connectedCount() == 0 })
}

func TestCloudflareGatewayRejectsWrongSecretAndProtocol(t *testing.T) {
	host, gateway, _ := testCloudflareGateway(t)
	conn := dialTestGateway(t, gateway)
	wrong, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	client, err := newClientNetplayFromConnWithKey(newTrackedWebSocketConn(conn), "wrong secret", wrong)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	waitFor(t, "wrong secret rejection", func() bool { return client.LastError() != nil })
	if host.connectedCount() != 0 {
		t.Fatal("wrong secret consumed player slot")
	}
	config, err := websocket.NewConfig("ws://"+gateway.listener.Addr().String()+cloudflareNetplayPath, "https://gunmayhem.invalid/")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if unexpected, err := config.DialContext(ctx); err == nil {
		unexpected.Close()
		t.Fatal("missing application subprotocol accepted")
	}
}

func TestCloudflareGatewayCloseCancelsUnauthenticatedConnection(t *testing.T) {
	host, gateway, _ := testCloudflareGateway(t)
	conn := dialTestGateway(t, gateway)
	defer conn.Close()
	var msg netWireMessage
	if err := readNetMessage(conn, maxServerWireFrame, &msg); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "gateway pending handshake", func() bool { return host.pendingHandshakes.Load() == 1 })
	if err := gateway.Close(); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	var b [1]byte
	if _, err := conn.Read(b[:]); err == nil {
		t.Fatal("upgrade survived gateway shutdown")
	}
	waitFor(t, "gateway handshake cleanup", func() bool { return host.pendingHandshakes.Load() == 0 })
	// Handler cleanup is asynchronous after the socket has closed; the
	// handshake goroutine finishing does not order the HTTP handler's defer.
	waitFor(t, "gateway upgraded connection cleanup", func() bool {
		gateway.connectionsMu.Lock()
		defer gateway.connectionsMu.Unlock()
		return len(gateway.connections) == 0
	})
}

func TestCloudflareGatewayBoundsWireFrames(t *testing.T) {
	host, gateway, _ := testCloudflareGateway(t)
	conn := dialTestGateway(t, gateway)
	defer conn.Close()
	var msg netWireMessage
	if err := readNetMessage(conn, maxServerWireFrame, &msg); err != nil {
		t.Fatal(err)
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], maxClientWireFrame+1)
	if _, err := conn.Write(header[:]); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := conn.Read(header[:]); err == nil {
		t.Fatal("oversized payload accepted over WebSocket")
	}
	waitFor(t, "oversized frame cleanup", func() bool { return host.pendingHandshakes.Load() == 0 })
	if host.connectedCount() != 0 {
		t.Fatal("oversized frame consumed a slot")
	}
}

func TestCloudflareGatewayDisconnectAndRejoin(t *testing.T) {
	host, gateway, key := testCloudflareGateway(t)
	for attempt := 0; attempt < 3; attempt++ {
		conn := dialTestGateway(t, gateway)
		client, err := newClientNetplayFromConnWithKey(newTrackedWebSocketConn(conn), "rejoin test", key)
		if err != nil {
			t.Fatal(err)
		}
		waitFor(t, "WebSocket join", client.connected)
		if client.assignedPlayer() != 2 {
			client.Close()
			t.Fatal("vacated slot not reused")
		}
		client.Close()
		waitFor(t, "WebSocket slot cleanup", func() bool { return host.connectedCount() == 0 })
	}
}

func TestTrackedWebSocketCloseUnblocksPendingWriter(t *testing.T) {
	left, right := net.Pipe()
	defer right.Close()
	tracked := newTrackedWebSocketConn(left)
	writing := make(chan struct{})
	written := make(chan struct{})
	go func() { close(writing); _, _ = tracked.Write([]byte("blocked")); close(written) }()
	<-writing
	tracked.Close()
	select {
	case <-written:
	case <-time.After(time.Second):
		t.Fatal("close left writer blocked")
	}
}

func TestCloudflareGatewayRejectsForeignAndMissingOrigin(t *testing.T) {
	_, gateway, _ := testCloudflareGateway(t)
	for _, origin := range []string{"https://attacker.example/", "http://gunmayhem.invalid/", "https://gunmayhem.invalid.evil.example/", "https://gunmayhem.invalid/path"} {
		t.Run(origin, func(t *testing.T) {
			config, err := websocket.NewConfig("ws://"+gateway.listener.Addr().String()+cloudflareNetplayPath, origin)
			if err != nil {
				t.Fatal(err)
			}
			config.Protocol = []string{cloudflareNetplayProtocol}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if conn, err := config.DialContext(ctx); err == nil {
				conn.Close()
				t.Fatal("foreign origin accepted")
			}
		})
	}
	request, err := http.NewRequest(http.MethodGet, "http://"+gateway.listener.Addr().String()+cloudflareNetplayPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "websocket")
	request.Header.Set("Sec-WebSocket-Version", "13")
	request.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	request.Header.Set("Sec-WebSocket-Protocol", cloudflareNetplayProtocol)
	client := &http.Client{Timeout: time.Second}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("missing origin status=%d", response.StatusCode)
	}
}

func TestCloudflareLongestInviteFitsAddressField(t *testing.T) {
	key, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	code, err := encodeCloudflareInviteCode("https://"+strings.Repeat("a", 63)+cloudflareTunnelHostSuffix, key)
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 95 {
		t.Fatalf("max invite length=%d want95", len(code))
	}
	got := appendMultiplayerAddressInput("", []rune(code))
	if got != code {
		t.Fatalf("invite truncated: %d to %d", len(code), len(got))
	}
	_, gotKey, err := decodeCloudflareInviteCode(got)
	if err != nil || gotKey != key {
		t.Fatal("entered invite did not roundtrip")
	}
	if len(appendMultiplayerAddressInput("", []rune(strings.Repeat("a", 1024)))) != maxMultiplayerAddressChars {
		t.Fatal("input lost its size bound")
	}
	if got := appendMultiplayerAddressInput("GM-", []rune("A\n B/")); got != "GM-AB" {
		t.Fatalf("invalid input accepted: %q", got)
	}
}
