package game

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNormalizeCloudflareTunnelURL(t *testing.T) {
	got, err := normalizeCloudflareTunnelURL("https://quiet-river-123.trycloudflare.com")
	if err != nil {
		t.Fatal(err)
	}
	want := "wss://quiet-river-123.trycloudflare.com" + cloudflareNetplayPath
	if got != want {
		t.Fatalf("url=%q want %q", got, want)
	}
	for _, bad := range []string{
		"http://quiet-river-123.trycloudflare.com",
		"https://example.com",
		"https://../trycloudflare.com",
		"wss://quiet-river-123.trycloudflare.com:8443/gm/netplay",
	} {
		if _, err := normalizeCloudflareTunnelURL(bad); err == nil {
			t.Fatalf("accepted invalid URL %q", bad)
		}
	}
}

func TestCloudflareInviteCodeRoundTrip(t *testing.T) {
	key, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	code, err := encodeCloudflareInviteCode("https://winter-moon-77.trycloudflare.com", key)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(code, cloudflareInviteCodePrefix) {
		t.Fatalf("code=%q", code)
	}
	wsURL, decodedKey, err := decodeCloudflareInviteCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if wsURL != "wss://winter-moon-77.trycloudflare.com"+cloudflareNetplayPath {
		t.Fatalf("wsURL=%q", wsURL)
	}
	if decodedKey != key {
		t.Fatalf("decoded key mismatch: %q != %q", decodedKey, key)
	}
}

func TestCloudflareQuickTunnelLiveWebSocket(t *testing.T) {
	if os.Getenv("GUNMAYHEM_CLOUDFLARE_LIVE_TEST") != "1" {
		t.Skip("set GUNMAYHEM_CLOUDFLARE_LIVE_TEST=1 to create a real Quick Tunnel")
	}
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	host.inviteOnly = false
	host.expectedSessionKey = ""
	host.setAllowedRemotePlayers([5]bool{false, true, true, true, true})

	tunnel, err := startCloudflareQuickTunnel(host)
	if err != nil {
		t.Fatal(err)
	}
	defer tunnel.Close()
	wsURL, err := tunnel.WaitReady(30 * time.Second)
	if err != nil {
		t.Fatal(err)
	}

	client, err := newClientNetplayCloudflare(wsURL, "", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := waitForClientHandshake(client, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if got := client.assignedPlayer(); got != 2 {
		t.Fatalf("assigned player=%d want 2", got)
	}
}

type typedNilRemoteAddrConn struct {
	net.Conn
}

func (typedNilRemoteAddrConn) RemoteAddr() net.Addr {
	var addr *net.TCPAddr
	return addr
}

func TestCloudflareMonitorStartupDoesNotBlockCaller(t *testing.T) {
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer health.Close()

	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	tunnel := &cloudflareQuickTunnel{
		session:   host,
		publicURL: health.URL,
		wsURL:     "wss://unit-test.trycloudflare.com" + cloudflareNetplayPath,
		ready:     make(chan struct{}),
	}
	tunnel.running.Store(true)
	tunnel.readyOnce.Do(func() { close(tunnel.ready) })

	started := time.Now()
	tunnel.monitorStartup()
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("monitorStartup blocked caller for %v", elapsed)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ready, publicURL, wsURL, err := tunnel.Status()
		if ready && publicURL == health.URL && wsURL == tunnel.wsURL {
			if got := host.publicProbeInfo().TunnelAddress; got != wsURL {
				t.Fatalf("published tunnel address=%q want %q", got, wsURL)
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("asynchronous Cloudflare startup did not become ready")
}

func TestTrackedWebSocketConnUsesSafeRemoteAddr(t *testing.T) {
	tracked := newTrackedWebSocketConn(typedNilRemoteAddrConn{})
	addr := tracked.RemoteAddr()
	if addr == nil || addr.Network() != "cloudflare-wss" || addr.String() != "cloudflare-wss" {
		t.Fatalf("unexpected fallback address: %#v", addr)
	}
}

func TestLoopbackEphemeralInternetHostDoesNotRequireCloudflared(t *testing.T) {
	if shouldUseCloudflareInternetTunnel("127.0.0.1:0") {
		t.Fatal("loopback ephemeral test host unexpectedly requires Cloudflare")
	}
	if !shouldUseCloudflareInternetTunnel(":7777") {
		t.Fatal("production Internet host did not require Cloudflare")
	}
}
