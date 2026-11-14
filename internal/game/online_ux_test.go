package game

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNormalizePastedJoinInputFindsCloudflareInvite(t *testing.T) {
	key, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	code, err := encodeCloudflareInviteCode("https://quiet-gm-river-12.trycloudflare.com", key)
	if err != nil {
		t.Fatal(err)
	}
	for _, pasted := range []string{
		code,
		"  " + code + "\r\n",
		"\t" + strings.ToLower(code) + "\n",
		"Join my game: " + code + " thanks!",
		"code=\"" + code + "\".",
		"(" + code + ")",
	} {
		got := normalizePastedJoinInput(pasted)
		if got != code {
			t.Fatalf("paste %q -> %q want %q", pasted, got, code)
		}
		if _, decodedKey, err := decodeCloudflareInviteCode(got); err != nil || decodedKey != key {
			t.Fatalf("pasted code does not decode: %v", err)
		}
	}
}

func TestNormalizePastedJoinInputOtherForms(t *testing.T) {
	cases := map[string]string{
		"  192.168.1.20:7777 \n": "192.168.1.20:7777",
		"gm-abc123":              "GM-ABC123",
		"https://quiet-gm-river-12.trycloudflare.com":          "quiet-gm-river-12.trycloudflare.com",
		"wss://quiet-gm-river-12.trycloudflare.com/gm/netplay": "quiet-gm-river-12.trycloudflare.com",
		"[::1]:7777": "[::1]:7777",
		"   ":        "",
	}
	for in, want := range cases {
		if got := normalizePastedJoinInput(in); got != want {
			t.Fatalf("paste %q -> %q want %q", in, got, want)
		}
	}
	long := strings.Repeat("A", 500)
	if got := normalizePastedJoinInput(long); len(got) != cloudflareTunnelURLMaxLen {
		t.Fatalf("long paste not capped: %d", len(got))
	}
}

func TestWrapTextToWidthKeepsEveryCharacter(t *testing.T) {
	measure := func(s string) float64 { return float64(len([]rune(s))) }
	text := "JOIN ERROR: connect through Cloudflare: dial tcp: lookup quiet-river-12.trycloudflare.com: no such host " + strings.Repeat("X", 45)
	lines := wrapTextToWidth(text, 20, measure)
	if len(lines) < 5 {
		t.Fatalf("expected several lines, got %d", len(lines))
	}
	for _, line := range lines {
		if measure(line) > 20 {
			t.Fatalf("line too wide: %q", line)
		}
	}
	joined := strings.ReplaceAll(strings.Join(lines, ""), " ", "")
	if joined != strings.ReplaceAll(text, " ", "") {
		t.Fatalf("wrapping lost text:\n%q\n%q", joined, text)
	}
}

func TestNetClientInputReleasedWhenUnfocused(t *testing.T) {
	held := PlayerInputState{Right: true, Up: true}
	if got := netClientInputForFocus(held, true); got != held {
		t.Fatalf("focused input changed: %+v", got)
	}
	if got := netClientInputForFocus(held, false); got != (PlayerInputState{}) {
		t.Fatalf("unfocused input not neutral: %+v", got)
	}
}

func waitPendingJoin(t *testing.T, g *Game) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for g.pendingNetJoin != nil {
		if time.Now().After(deadline) {
			t.Fatal("async join did not finish")
		}
		g.updatePendingNetJoin()
		time.Sleep(5 * time.Millisecond)
	}
}

func TestAsyncJoinReportsFullError(t *testing.T) {
	g := &Game{}
	longErr := errors.New("connect through Cloudflare: " + strings.Repeat("very long reason ", 12))
	g.beginAsyncNetJoin("GMT1-TEST", func(*Game) error {
		time.Sleep(20 * time.Millisecond)
		return longErr
	}, func() { t.Fatal("onSuccess called for a failed join") })
	if g.pendingNetJoin == nil || !strings.HasPrefix(g.multiplayerMessage, "CONNECTING TO") {
		t.Fatalf("no pending join state: %q", g.multiplayerMessage)
	}
	waitPendingJoin(t, g)
	if g.multiplayerMessage != "JOIN ERROR: "+longErr.Error() {
		t.Fatalf("error truncated or changed: %q", g.multiplayerMessage)
	}
	if g.netplay != nil {
		t.Fatal("failed join installed a session")
	}
}

func TestAsyncJoinInstallsSessionOnGameThread(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	address := host.listener.Addr().String()

	g := &Game{}
	succeeded := false
	g.beginAsyncNetJoin(address, func(scratch *Game) error {
		return scratch.startNetClientFromInput(address)
	}, func() { succeeded = true })
	waitPendingJoin(t, g)
	defer g.CloseNetplay()
	if !succeeded || g.netplay == nil || g.netplay.mode != netplayClient {
		t.Fatalf("join not installed: ok=%v msg=%q", succeeded, g.multiplayerMessage)
	}
	if got := g.netplay.assignedPlayer(); got < 2 {
		t.Fatalf("assigned player=%d", got)
	}
}

func TestCancelPendingJoinClosesLateSession(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	address := host.listener.Addr().String()

	release := make(chan struct{})
	sessions := make(chan *netplaySession, 1)
	g := &Game{}
	g.beginAsyncNetJoin(address, func(scratch *Game) error {
		<-release
		err := scratch.startNetClientFromInput(address)
		sessions <- scratch.netplay
		return err
	}, func() { t.Fatal("cancelled join reached onSuccess") })
	g.cancelPendingNetJoin()
	if g.pendingNetJoin != nil {
		t.Fatal("pending join not cleared")
	}
	close(release)
	s := <-sessions
	if s == nil {
		t.Fatal("join did not produce a session")
	}
	deadline := time.Now().Add(3 * time.Second)
	for !s.closed.Load() {
		if time.Now().After(deadline) {
			t.Fatal("session from a cancelled join was not closed")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if g.netplay != nil {
		t.Fatal("cancelled join installed a session")
	}
}

// TestClipboardRoundTrip uses the real system clipboard, so it is opt-in. It
// restores the previous text afterwards.
func TestClipboardRoundTrip(t *testing.T) {
	if os.Getenv("GUNMAYHEM_CLIPBOARD_TEST") != "1" {
		t.Skip("set GUNMAYHEM_CLIPBOARD_TEST=1 to touch the system clipboard")
	}
	previous, prevErr := readClipboardText()
	defer func() {
		if prevErr == nil {
			_ = writeClipboardText(previous)
		}
	}()
	want := "GMT1-QUIET-RIVER-12.ABCDEFGHIJKLMNOPQRSTUVWXYZ zażółć"
	if err := writeClipboardText(want); err != nil {
		t.Fatal(err)
	}
	got, err := readClipboardText()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("clipboard=%q want %q", got, want)
	}
}
