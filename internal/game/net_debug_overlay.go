package game

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// net_debug_overlay.go: F6 toggles a small network/smoothness overlay (any
// screen, local or online). Ctrl+F6 switches the guest's own-player
// prediction off/on for A/B comparison. F6 is not a bindable game key.

func (g *Game) updateNetDebugKeys() {
	if g == nil || !inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		return
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		g.netPredict.disabled = !g.netPredict.disabled
		g.netPredict.reset()
		g.netDebugOverlay = true
		return
	}
	g.netDebugOverlay = !g.netDebugOverlay
}

// netDebugLines describes the current render and network state.
func (g *Game) netDebugLines(now time.Time) []string {
	smooth := "OFF"
	if g.renderInterpolationEnabled() {
		smooth = "ON"
	}
	fps := "VSYNC"
	if c := displayFrameRateCap(g.displayFrameRate); c > 0 {
		fps = fmt.Sprintf("CAP %d", c)
	} else if g.displayFrameRate == displayFrameRateUnlimited {
		fps = "UNLIMITED"
	}
	lines := []string{fmt.Sprintf("FPS %.0f (%s)  TPS %.0f  SMOOTH %s  alpha %.2f",
		ebiten.ActualFPS(), fps, ebiten.ActualTPS(), smooth, g.renderInterp.lastAlpha)}
	s := g.netplay
	if s == nil {
		return lines
	}
	switch s.mode {
	case netplayClient:
		rate, jitter, gap := s.snapshotMeter.stats(now)
		buffered, delay := s.playout.stats()
		ping := "..."
		if rtt, ok := s.hostRTTMillis(); ok {
			ping = fmt.Sprintf("%d ms", rtt)
		}
		lines = append(lines,
			fmt.Sprintf("SNAP %.0f/s  jitter %.0f ms  max gap %.0f ms  PING %s", rate, jitter, gap, ping),
			fmt.Sprintf("BUFFER %d ahead  playout delay %d ticks", buffered, delay))
		st := g.netPredict.stats
		pred := "ON"
		if g.netPredict.disabled {
			pred = "OFF (Ctrl+F6)"
		} else if !g.netPredict.lastAppliedOK {
			pred = "IDLE"
		}
		lines = append(lines, fmt.Sprintf("PREDICT %s  unacked %d  err %.1f px  max %.1f  blend %.1f  snaps %d",
			pred, st.unacked, st.lastError, st.maxError, st.offset, st.snaps))
	case netplayHost:
		s.peersMu.RLock()
		for _, id := range sortedPeerIDs(s.peers) {
			p := s.peers[id]
			ping := "..."
			if rtt, ok := loadNetRTT(&p.rttMillis); ok {
				ping = fmt.Sprintf("%d ms", rtt)
			}
			p.inputMu.Lock()
			queued := len(p.input.queue)
			p.inputMu.Unlock()
			lines = append(lines, fmt.Sprintf("P%d  PING %s  in flight %d ticks  input queue %d", id, ping, p.inflightTicks(), queued))
		}
		s.peersMu.RUnlock()
	}
	return lines
}

func sortedPeerIDs(peers map[int]*netplayPeer) []int {
	var ids []int
	for id := 1; id <= maxOnlinePlayers; id++ {
		if peers[id] != nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func (g *Game) drawNetDebugOverlay(screen *ebiten.Image) {
	if g == nil || !g.netDebugOverlay || screen == nil {
		return
	}
	lines := g.netDebugLines(time.Now())
	w := 0.0
	for _, l := range lines {
		w = max(w, uiMeasure(false, uiSizeHint, l))
	}
	r := Rect{X: 6, Y: ScreenHeight - 10 - float64(len(lines))*17, W: w + 16, H: float64(len(lines))*17 + 6}
	uiFillRound(screen, r, 4, uiWithAlpha(uiInk900, 0.85))
	for i, l := range lines {
		uiText(screen, l, false, uiSizeHint, uiTextLight, r.X+8, r.Y+11+float64(i)*17, uiAlignLeft)
	}
}
