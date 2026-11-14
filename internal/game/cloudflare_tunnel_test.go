package game

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
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
	// Use the production startup path (backoff + cloudflared restarts).
	tunnel.monitorStartup()
	var wsURL string
	deadline := time.Now().Add(time.Duration(cloudflareTunnelMaxAttempts) * (cloudflareTunnelStartWindow + 5*time.Second))
	for {
		ready, _, url, statusErr := tunnel.Status()
		if ready {
			wsURL = url
			break
		}
		if statusErr != nil {
			t.Fatal(statusErr)
		}
		if time.Now().After(deadline) {
			t.Fatal("Quick Tunnel did not become ready")
		}
		time.Sleep(250 * time.Millisecond)
	}
	attempt, _ := tunnel.StartupAttempt()
	t.Logf("tunnel ready attempt=%d url=%s", attempt, wsURL)
	if data, readErr := os.ReadFile(networkDiagnosticPath); readErr == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "CLOUDFLARE") {
				t.Log(line)
			}
		}
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
	}
	tunnel.running.Store(true)

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

// TestCloudflareWaitReadyBacksOffHealthProbes guards against the old busy loop
// that fired ~35 /healthz requests per second while the edge returned 530.
func TestCloudflareWaitReadyBacksOffHealthProbes(t *testing.T) {
	var (
		mu       sync.Mutex
		requests []time.Time
	)
	const failures = 4
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, time.Now())
		n := len(requests)
		mu.Unlock()
		if n <= failures {
			w.WriteHeader(530)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer health.Close()

	tunnel := &cloudflareQuickTunnel{
		publicURL:        health.URL,
		wsURL:            "wss://unit-test.trycloudflare.com" + cloudflareNetplayPath,
		healthBackoffMin: 40 * time.Millisecond,
		healthBackoffMax: 120 * time.Millisecond,
	}
	tunnel.running.Store(true)
	wsURL, err := tunnel.WaitReady(5 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if wsURL != tunnel.wsURL {
		t.Fatalf("wsURL=%q", wsURL)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(requests) != failures+1 {
		t.Fatalf("health requests=%d want %d", len(requests), failures+1)
	}
	wantGaps := []time.Duration{40, 80, 120, 120}
	for i := 1; i < len(requests); i++ {
		gap := requests[i].Sub(requests[i-1])
		min := wantGaps[i-1]*time.Millisecond - 10*time.Millisecond
		if gap < min {
			t.Fatalf("probe %d came %v after the previous one; want >= %v", i, gap, min)
		}
	}
}

func TestCloudflareWaitReadyTimesOutWithRealReason(t *testing.T) {
	var count atomic.Int32
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(530)
	}))
	defer health.Close()

	tunnel := &cloudflareQuickTunnel{
		publicURL:        health.URL,
		wsURL:            "wss://unit-test.trycloudflare.com" + cloudflareNetplayPath,
		healthBackoffMin: 50 * time.Millisecond,
		healthBackoffMax: 100 * time.Millisecond,
	}
	tunnel.running.Store(true)
	started := time.Now()
	_, err := tunnel.WaitReady(600 * time.Millisecond)
	if err == nil {
		t.Fatal("WaitReady succeeded against a failing edge")
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("WaitReady overran its timeout: %v", elapsed)
	}
	if !strings.Contains(err.Error(), "HTTP 530") {
		t.Fatalf("error does not carry the real reason: %v", err)
	}
	if got := count.Load(); got > 12 {
		t.Fatalf("too many health probes in 600ms: %d", got)
	}
}

func TestCloudflareWaitReadyStopsOnAttemptError(t *testing.T) {
	tunnel := &cloudflareQuickTunnel{lastCloudflaredErr: "2026-01-01T00:00:00Z ERR failed to request quick Tunnel: 429 Too Many Requests"}
	tunnel.setAttemptError(errors.New("cloudflared stopped: exit status 1"))
	_, err := tunnel.WaitReady(time.Second)
	if err == nil {
		t.Fatal("WaitReady ignored the attempt error")
	}
	if _, _, _, statusErr := tunnel.Status(); statusErr != nil {
		t.Fatalf("a retryable attempt error leaked into Status: %v", statusErr)
	}
	if got := tunnel.withCloudflaredReason(err).Error(); !strings.Contains(got, "429 Too Many Requests") {
		t.Fatalf("cloudflared reason missing: %s", got)
	}
}

func TestCloudflaredErrorLineDetection(t *testing.T) {
	if !isCloudflaredErrorLine("2026-11-14T10:00:00Z ERR Failed to create new quick Tunnel error=\"429\"") {
		t.Fatal("ERR line not detected")
	}
	if isCloudflaredErrorLine("2026-11-14T10:00:00Z INF Requesting new quick Tunnel on trycloudflare.com...") {
		t.Fatal("INF line treated as an error")
	}
}
