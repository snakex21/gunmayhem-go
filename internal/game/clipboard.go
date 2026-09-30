package game

import (
	"errors"
	"strings"
	"unicode/utf16"
)

// Clipboard input is untrusted. Bound it before decoding or trimming, then
// replace the field atomically rather than filtering/truncating an invite.
const maxClipboardTextBytes = 4096

var errClipboardUnsupported = errors.New("CLIPBOARD REQUIRES WINDOWS")

func normalizePastedMultiplayerAddress(value string) (string, error) {
	if len(value) > maxClipboardTextBytes {
		return "", errors.New("TEXT TOO LONG")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("NO TEXT")
	}
	if len(value) > maxMultiplayerAddressChars {
		return "", errors.New("ADDRESS TOO LONG")
	}
	for _, r := range value {
		if !validNetAddressRune(r) {
			return "", errors.New("USE ONE CODE OR IP:PORT")
		}
	}
	return value, nil
}

func decodeClipboardTextUTF16(units []uint16) (string, error) {
	if len(units) == 0 || len(units) > maxClipboardTextBytes/2 {
		return "", errors.New("INVALID CLIPBOARD SIZE")
	}
	for i, u := range units {
		if u == 0 {
			// Invalid UTF-16 decodes to a replacement rune, which the address
			// validator rejects together with every other non-ASCII character.
			return string(utf16.Decode(units[:i])), nil
		}
	}
	return "", errors.New("INVALID CLIPBOARD TEXT")
}

func (g *Game) pasteMultiplayerAddress(read func() (string, error)) {
	if g == nil || !g.multiplayerFocus {
		return
	}
	value, err := read()
	if err != nil {
		// Do not echo clipboard contents or arbitrary backend error details.
		g.multiplayerMessage = "PASTE FAILED - RETRY OR TYPE THE CODE"
		if errors.Is(err, errClipboardUnsupported) {
			g.multiplayerMessage = "PASTE UNAVAILABLE - CLIPBOARD REQUIRES WINDOWS"
		}
		return
	}
	value, err = normalizePastedMultiplayerAddress(value)
	if err != nil {
		g.multiplayerMessage = "PASTE FAILED: " + err.Error()
		return
	}
	g.multiplayerAddress = value
	g.multiplayerMessage = "PASTED - CHECK THE CODE, THEN CLICK JOIN"
}

func (g *Game) canCopyHostInvite() bool {
	if g == nil || !g.customOnlineMode || g.customPage != customPagePlayers || g.netplay == nil || g.netplay.mode != netplayHost || g.onlineSessionVisibility != onlineSessionInviteOnly {
		return false
	}
	code, ready := g.currentHostDisplayJoinCode()
	return ready && code != ""
}

func (g *Game) copyHostInvite(write func(string) error) {
	if !g.canCopyHostInvite() {
		return
	}
	code, _ := g.currentHostDisplayJoinCode()
	if err := write(code); err != nil {
		g.multiplayerMessage = "COPY FAILED - RETRY"
		if errors.Is(err, errClipboardUnsupported) {
			g.multiplayerMessage = "COPY REQUIRES WINDOWS"
		}
		return
	}
	g.multiplayerMessage = "INVITE COPIED"
}

func (g *Game) handleCopyInviteHit(hit int, write func(string) error) bool {
	if hit != menuHitCustomV2CopyInvite {
		return false
	}
	g.copyHostInvite(write)
	return true
}

// Internet labels follow the explicit opt-in and configured catalog; ordinary
// launches continue to describe the default LAN-only flow.
func customNetworkLabels() (tab, heading, address, placeholder string) {
	if experimentalInternetEnabled() {
		if configuredPublicDirectoryURL() != "" {
			return "ONLINE", "LAN + INTERNET ROOMS", "JOIN CODE / IP:PORT (CTRL+V ON WINDOWS)", "ENTER JOIN CODE OR IP:PORT"
		}
		return "ONLINE", "LAN + PRIVATE INTERNET", "JOIN CODE / IP:PORT (CTRL+V ON WINDOWS)", "ENTER JOIN CODE OR IP:PORT"
	}
	return "LAN", "LAN SERVER BROWSER", "JOIN CODE / LAN IP:PORT (CTRL+V ON WINDOWS)", "ENTER LAN JOIN CODE OR IP:PORT"
}
