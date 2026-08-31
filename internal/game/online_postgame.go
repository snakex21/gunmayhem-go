package game

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const postGameReadyCountdownFrames = 180

func (g *Game) resetOnlinePostGameReady() {
	if g == nil {
		return
	}
	g.postGameReady = [4]bool{}
	g.postGameReadyCountdown = 0
	if g.netplay != nil && g.netplay.mode == netplayHost {
		g.netplay.resetPostGameReady()
	}
}

func (g *Game) onlinePostGameReadySnapshot() [4]bool {
	if g == nil {
		return [4]bool{}
	}
	if g.netplay == nil || g.netplay.mode != netplayHost {
		return g.postGameReady
	}
	ready := [4]bool{}
	ready[0] = g.postGameReady[0]
	for _, id := range g.netplay.connectedPlayerIDs() {
		if id >= 2 && id <= maxOnlinePlayers {
			ready[id-1] = g.netplay.postGameReadyFor(id)
		}
	}
	return ready
}

func (g *Game) onlinePostGameReadyCounts() (readyCount, total int) {
	if g == nil || !g.onlineMatchSetup || g.netplay == nil {
		return 0, 0
	}
	ready := g.postGameReady
	present := g.onlinePlayerSlots
	if g.netplay.mode == netplayHost {
		ready = g.onlinePostGameReadySnapshot()
		present = [4]bool{true, false, false, false}
		for _, id := range g.netplay.connectedPlayerIDs() {
			if id >= 1 && id <= len(present) {
				present[id-1] = true
			}
		}
	}
	for i, isPresent := range present {
		if !isPresent {
			continue
		}
		total++
		if ready[i] {
			readyCount++
		}
	}
	return readyCount, total
}

func (g *Game) localPostGameReady() bool {
	if g == nil || g.netplay == nil {
		return false
	}
	id := 1
	if g.netplay.mode == netplayClient {
		id = g.netplay.assignedPlayer()
	}
	return id >= 1 && id <= len(g.postGameReady) && g.postGameReady[id-1]
}

func (g *Game) markLocalPostGameReady() {
	if g == nil || g.netplay == nil || !g.onlineMatchSetup {
		return
	}
	if g.netplay.mode == netplayHost {
		g.postGameReady[0] = true
		return
	}
	id := g.netplay.assignedPlayer()
	if id < 2 || id > maxOnlinePlayers {
		return
	}
	g.postGameReady[id-1] = true
	g.netplay.queuePostGameReady(true)
}

func (g *Game) updateOnlineHostPostGame() {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost || !g.onlineMatchSetup {
		return
	}

	mx, my := ebiten.CursorPosition()
	r := sourceMenuHitRect(campaignBackLocal, 638.15, 532.1, 1.86492919921875, 1.0372772216796875)
	hit := menuHitNone
	if r.Contains(float64(mx), float64(my)) && !g.localPostGameReady() {
		hit = menuHitGameContinue
	}
	activated, ok := g.resolveMenuRelease(hit)
	if ok && activated == menuHitGameContinue {
		g.playSourceSFX("menu.wav", false)
		g.markLocalPostGameReady()
	}

	g.postGameReady = g.onlinePostGameReadySnapshot()
	ready, total := g.onlinePostGameReadyCounts()
	if total > 0 && ready == total {
		if g.postGameReadyCountdown <= 0 {
			g.postGameReadyCountdown = postGameReadyCountdownFrames
		} else {
			g.postGameReadyCountdown--
			if g.postGameReadyCountdown <= 0 {
				if g.customSeriesRound > 0 && !g.customSeriesIsFinalRound() {
					g.startNextCustomSeriesRound()
					return
				}
				if g.customSeriesRound > 0 {
					g.finishCustomSeriesToSummary()
					return
				}
				g.customOnlineMode = true
				g.customPage = customPagePlayers
				g.customMenuX = 0
				g.beginFade(screenCustomGame, fadePurposeScreen)
			}
		}
	} else {
		g.postGameReadyCountdown = 0
	}
}

func (g *Game) updateOnlineClientPostGame() {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayClient || !g.onlineMatchSetup {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		_ = g.CloseNetplay()
		g.screen = screenCustomGame
		g.customOnlineMode = true
		g.onlineMatchSetup = false
		g.customNetworkOpen = true
		g.multiplayerMessage = "DISCONNECTED"
		g.beginMultiplayerServerBrowser()
		return
	}
	mx, my := ebiten.CursorPosition()
	r := sourceMenuHitRect(campaignBackLocal, 638.15, 532.1, 1.86492919921875, 1.0372772216796875)
	hit := menuHitNone
	if r.Contains(float64(mx), float64(my)) && !g.localPostGameReady() {
		hit = menuHitGameContinue
	}
	activated, ok := g.resolveMenuRelease(hit)
	if ok && activated == menuHitGameContinue {
		g.playSourceSFX("menu.wav", false)
		g.markLocalPostGameReady()
	}
}

func (g *Game) onlinePostGameStatusText() string {
	ready, total := g.onlinePostGameReadyCounts()
	if total <= 0 {
		return ""
	}
	if g.postGameReadyCountdown > 0 {
		seconds := (g.postGameReadyCountdown + 59) / 60
		return fmt.Sprintf("READY %d/%d   NEXT ROUND IN %d", ready, total, seconds)
	}
	if g.localPostGameReady() {
		return fmt.Sprintf("READY %d/%d   WAITING FOR PLAYERS", ready, total)
	}
	return fmt.Sprintf("READY %d/%d", ready, total)
}
