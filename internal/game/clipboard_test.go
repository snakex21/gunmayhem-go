package game

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestClipboardPasteAddressValidation(t *testing.T) {
	for _, value := range []string{"192.168.1.2:7777", "[::1]:7777", "GMT1-HOST.ABCDEFG234", strings.Repeat("A", maxMultiplayerAddressChars)} {
		got, err := normalizePastedMultiplayerAddress(" \r\n" + value + "\t ")
		if err != nil || got != value {
			t.Fatalf("valid address rejected: len=%d err=%v", len(value), err)
		}
	}
	for _, value := range []string{"", " \n", "FIRST\nSECOND", "FIRST\rSECOND", "FIRST\tSECOND", "FIRST SECOND", "https://example.com/", "code;command", "code\x00hidden", "code\u202e", "日本", strings.Repeat("A", maxMultiplayerAddressChars+1), strings.Repeat(" ", maxClipboardTextBytes+1)} {
		if _, err := normalizePastedMultiplayerAddress(value); err == nil {
			t.Fatalf("invalid address accepted: len=%d", len(value))
		}
	}
}

func TestClipboardUTF16BoundsAndValidation(t *testing.T) {
	value := "GMT1-HOST.ABC234"
	units := append(utf16.Encode([]rune(value)), 0)
	got, err := decodeClipboardTextUTF16(units)
	if err != nil || got != value {
		t.Fatalf("decode failed: %v", err)
	}
	for _, input := range [][]uint16{nil, {65}, make([]uint16, maxClipboardTextBytes/2+1)} {
		if _, err := decodeClipboardTextUTF16(input); err == nil {
			t.Fatal("invalid buffer accepted")
		}
	}
	got, err = decodeClipboardTextUTF16([]uint16{0xd800, 0})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = normalizePastedMultiplayerAddress(got); err == nil {
		t.Fatal("unpaired surrogate accepted")
	}
}

func TestClipboardPasteReplacesAtomicallyAndNeverReadsUnfocused(t *testing.T) {
	g := &Game{multiplayerAddress: "127.0.0.1:7777"}
	calls := 0
	read := func() (string, error) { calls++; return "192.168.1.2:7777", nil }
	g.pasteMultiplayerAddress(read)
	if calls != 0 {
		t.Fatal("unfocused field read clipboard")
	}
	g.multiplayerFocus = true
	g.pasteMultiplayerAddress(read)
	if calls != 1 || g.multiplayerAddress != "192.168.1.2:7777" {
		t.Fatal("paste did not replace default")
	}
	previous := g.multiplayerAddress
	for _, badRead := range []func() (string, error){
		func() (string, error) { return "SECRET\nSECOND", nil },
		func() (string, error) { return strings.Repeat("A", 129), nil },
		func() (string, error) { return "", errors.New("PRIVATE CLIPBOARD CONTENT") },
		func() (string, error) { return "", errClipboardUnsupported },
	} {
		g.pasteMultiplayerAddress(badRead)
		if g.multiplayerAddress != previous {
			t.Fatal("failed paste changed address")
		}
		if !strings.HasPrefix(g.multiplayerMessage, "PASTE ") || strings.Contains(g.multiplayerMessage, "SECRET") || strings.Contains(g.multiplayerMessage, "PRIVATE") {
			t.Fatal("bad paste status")
		}
	}
	if !strings.Contains(g.multiplayerMessage, "WINDOWS") {
		t.Fatal("unsupported platform not explained")
	}
}

func TestClipboardLongestInternetInviteRoundtrip(t *testing.T) {
	key, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	code, err := encodeCloudflareInviteCode("https://"+strings.Repeat("a", 63)+cloudflareTunnelHostSuffix, key)
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 95 {
		t.Fatalf("invite length=%d", len(code))
	}
	g := &Game{multiplayerFocus: true}
	g.pasteMultiplayerAddress(func() (string, error) { return code, nil })
	_, got, err := decodeCloudflareInviteCode(g.multiplayerAddress)
	if err != nil || got != key {
		t.Fatal("paste changed invitation")
	}
	host := &Game{customOnlineMode: true, customPage: customPagePlayers, onlineSessionVisibility: onlineSessionInviteOnly,
		onlineSessionScope: onlineSessionScopeInternet, onlineSessionKey: key, netplay: &netplaySession{mode: netplayHost},
		cloudflareTunnel: &cloudflareQuickTunnel{wsURL: "wss://" + strings.Repeat("a", 63) + cloudflareTunnelHostSuffix + "/ws"}}
	host.cloudflareTunnel.running.Store(true)
	copied := ""
	host.copyHostInvite(func(value string) error { copied = value; return nil })
	if copied != code {
		t.Fatal("copy lost part of the full Internet invite")
	}
	host.cloudflareTunnel.running.Store(false)
	host.copyHostInvite(func(string) error { t.Fatal("copied unavailable tunnel invite"); return nil })
}

func TestClipboardCopyInviteScopeAndFeedback(t *testing.T) {
	g := &Game{customOnlineMode: true, customPage: customPagePlayers, onlineSessionVisibility: onlineSessionInviteOnly, netplay: &netplaySession{mode: netplayHost, shortJoinCode: "ABC234"}}
	if !g.canCopyHostInvite() {
		t.Fatal("private ready host cannot copy")
	}
	x, y := customV2CopyInviteRect.X+2, customV2CopyInviteRect.Y+2
	if g.customV2HitAt(x, y) != menuHitCustomV2CopyInvite {
		t.Fatal("copy button not clickable")
	}
	calls := 0
	var copied string
	write := func(value string) error { calls++; copied = value; return nil }
	if !g.handleCopyInviteHit(g.customV2HitAt(x, y), write) {
		t.Fatal("copy hit not dispatched")
	}
	if calls != 1 || copied != "GM-ABC234" || g.multiplayerMessage != "INVITE COPIED" {
		t.Fatal("copy result incorrect")
	}
	if g.handleCopyInviteHit(menuHitNone, write) || calls != 1 {
		t.Fatal("unrelated hit copied invitation")
	}
	g.copyHostInvite(func(string) error { return errors.New("PRIVATE DATA") })
	if g.multiplayerMessage != "COPY FAILED - RETRY" {
		t.Fatal("copy error leaked backend details")
	}
	g.copyHostInvite(func(string) error { return errClipboardUnsupported })
	if g.multiplayerMessage != "COPY REQUIRES WINDOWS" {
		t.Fatal("copy unsupported not explained")
	}
	for _, change := range []func(){
		func() { g.onlineSessionVisibility = onlineSessionOpen },
		func() { g.onlineSessionVisibility = onlineSessionInviteOnly; g.netplay.mode = netplayClient },
		func() { g.netplay.mode = netplayHost; g.customPage = customPageMaps },
		func() { g.customPage = customPagePlayers; g.netplay = nil },
	} {
		change()
		g.copyHostInvite(write)
		if calls != 1 || g.canCopyHostInvite() || g.customV2HitAt(x, y) == menuHitCustomV2CopyInvite {
			t.Fatal("copy available outside private host slots")
		}
	}
}

func TestClipboardInternetLabelsRequireOptIn(t *testing.T) {
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "")
	tab, heading, address, _ := customNetworkLabels()
	if tab != "LAN" || heading != "LAN SERVER BROWSER" || !strings.Contains(address, "LAN IP:PORT") {
		t.Fatal("default labels changed reach")
	}
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "1")
	tab, heading, address, _ = customNetworkLabels()
	if tab != "ONLINE" || heading != "LAN + PRIVATE INTERNET" || strings.Contains(address, "LAN IP:PORT") {
		t.Fatal("opt-in labels misleading")
	}
}
