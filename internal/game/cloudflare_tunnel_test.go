package game

import (
	"strings"
	"testing"
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

func TestLoopbackEphemeralInternetHostDoesNotRequireCloudflared(t *testing.T) {
	if shouldUseCloudflareInternetTunnel("127.0.0.1:0") {
		t.Fatal("loopback ephemeral test host unexpectedly requires Cloudflare")
	}
	if !shouldUseCloudflareInternetTunnel(":7777") {
		t.Fatal("production Internet host did not require Cloudflare")
	}
}
