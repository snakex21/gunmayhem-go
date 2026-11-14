package game

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
)

// Player-facing text for netplay version checks, disconnect reasons and the
// ping overlay.

const netDisconnectNoticeDuration = 20 * time.Second

func netVersionMismatchMessage(hostVersion, clientVersion string) string {
	return fmt.Sprintf("version mismatch: Host has version %s, you have %s. Both players need the same game version.", hostVersion, clientVersion)
}

func netRemoteVersionLabel(protocol int, gameVersion string) string {
	gameVersion = sanitizeNetReason(gameVersion)
	if gameVersion != "" {
		return gameVersion
	}
	if protocol > 0 && protocol < netplayProtocolVersion {
		return fmt.Sprintf("an older version (network protocol %d)", protocol)
	}
	return fmt.Sprintf("an unknown version (network protocol %d)", protocol)
}

// sanitizeNetReason bounds text received from the other side before it is
// shown or logged.
func sanitizeNetReason(reason string) string {
	reason = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.TrimSpace(reason))
	if len([]rune(reason)) > 160 {
		reason = string([]rune(reason)[:160]) + "..."
	}
	return reason
}

// friendlyNetError turns a session error into a short reason for the player.
func friendlyNetError(err error) string {
	switch {
	case err == nil:
		return "connection closed"
	case errors.Is(err, errNetPeerTimeout):
		return "no data from the host for 10 s - the connection was lost"
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return "the host closed the connection"
	case errors.Is(err, net.ErrClosed):
		return "connection closed"
	}
	return err.Error()
}

// setNetDisconnectNotice shows why the session ended. The server browser
// refresh that follows a disconnect keeps it visible instead of overwriting
// it with "SEARCHING NETWORK...".
func (g *Game) setNetDisconnectNotice(text string) {
	g.netDisconnectNotice = text
	g.netDisconnectNoticeUntil = time.Now().Add(netDisconnectNoticeDuration)
	g.multiplayerMessage = text
}

func (g *Game) setMultiplayerBrowserStatus(text string) {
	if g.netDisconnectNotice != "" && time.Now().Before(g.netDisconnectNoticeUntil) {
		g.multiplayerMessage = g.netDisconnectNotice + " | " + text
		return
	}
	g.netDisconnectNotice = ""
	g.multiplayerMessage = text
}

// netPingLabel is the short ping text for the online overlay ("" offline).
func (g *Game) netPingLabel() string {
	if g == nil || g.netplay == nil {
		return ""
	}
	s := g.netplay
	switch s.mode {
	case netplayClient:
		if !s.connected() {
			return ""
		}
		if rtt, ok := s.hostRTTMillis(); ok {
			return fmt.Sprintf("PING %d ms", rtt)
		}
		return "PING ..."
	case netplayHost:
		var parts []string
		for _, id := range s.connectedPlayerIDs() {
			if rtt, ok := s.peerRTTMillis(id); ok {
				parts = append(parts, fmt.Sprintf("P%d %d ms", id, rtt))
			} else {
				parts = append(parts, fmt.Sprintf("P%d ...", id))
			}
		}
		if len(parts) == 0 {
			return ""
		}
		return "PING " + strings.Join(parts, "  ")
	}
	return ""
}

// drawNetPingOverlay draws the ping chip in the top-right corner during
// online sessions (lobby, setup and gameplay).
func (g *Game) drawNetPingOverlay(screen *ebiten.Image) {
	label := g.netPingLabel()
	if label == "" || screen == nil {
		return
	}
	w := uiMeasure(true, uiSizeHint, label) + 26
	r := Rect{X: float64(screen.Bounds().Dx()) - w - 6, Y: 4, W: w, H: 20}
	uiFillRound(screen, r, 4, uiWithAlpha(uiInk900, 0.85))
	uiFillRound(screen, Rect{X: r.X + 7, Y: r.Y + 6, W: 8, H: 8}, 4, uiNetBlue)
	uiTextIn(screen, label, true, uiSizeHint, uiTextLight, Rect{X: r.X + 20, Y: r.Y, W: r.W - 22, H: r.H}, uiAlignLeft, 0)
}

// drawNetClientWaitingScreen is the guest's fallback view before the first
// snapshot of a match (or on a screen guests have no menu for).
func (g *Game) drawNetClientWaitingScreen(screen *ebiten.Image) {
	ctx := (&uiState{}).begin(uiPassDraw, screen, uiInput{X: -1, Y: -1})
	ctx.Page(1)
	ctx.Header("ONLINE", "ONLINE  /  "+strings.ToUpper(g.NetplayStatus()))
	ctx.Badge(Rect{X: uiMargin + uiMeasure(true, uiSizeTitle, "ONLINE") + 14, Y: 31, W: 44, H: 22}, "NET", uiNetBlue)
	ctx.Panel(Rect{X: 250, Y: 230, W: 400, H: 110}, "")
	ctx.Label("WAITING FOR THE HOST", true, uiSizeSection, uiTextLight, ScreenWidth/2, 270, uiAlignCenter)
	ctx.Label("The match starts as soon as the host's game arrives.", false, uiSizeBody, uiTextMuted, ScreenWidth/2, 304, uiAlignCenter)
}
