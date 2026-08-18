package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type CustomPlayerConfig struct {
	Name  string
	Color int
	Shirt int
	Hat   int
	Gun   int
	Perk  int
	Type  int // source playertype: 0 empty, 1 human, 2 AI
	Team  int
}

func sourceDefaultCustomPlayers() [4]CustomPlayerConfig {
	return [4]CustomPlayerConfig{
		{Name: "Player 1", Color: 2, Shirt: 1, Hat: 1, Gun: 1, Perk: 7, Type: 1, Team: 1},
		{Name: "Player 2", Color: 5, Shirt: 1, Hat: 1, Gun: 1, Perk: 7, Type: 1, Team: 2},
		{Name: "Player 3", Color: 8, Shirt: 1, Hat: 1, Gun: 1, Perk: 7, Type: 0, Team: 1},
		{Name: "Player 4", Color: 10, Shirt: 1, Hat: 1, Gun: 1, Perk: 7, Type: 0, Team: 2},
	}
}

const (
	menuHitPlayerSlotBase = 300
	playerSlotActionHuman = 1
	playerSlotActionAI    = 2
	playerSlotActionClear = 3
)

var customPlayerTypeButtonLocal = Rect{X: 0, Y: 0, W: 148.5, H: 24} // Symbol806

func playerSlotHit(slot, action int) int {
	return menuHitPlayerSlotBase + slot*10 + action
}

func decodePlayerSlotHit(hit int) (slot, action int, ok bool) {
	if hit < menuHitPlayerSlotBase || hit >= menuHitPlayerSlotBase+40 {
		return 0, 0, false
	}
	v := hit - menuHitPlayerSlotBase
	slot, action = v/10, v%10
	if slot < 0 || slot >= 4 || action < playerSlotActionHuman || action > playerSlotActionClear {
		return 0, 0, false
	}
	return slot, action, true
}

func (g *Game) initCustomPlayerSetup() {
	g.customPlayers = sourceDefaultCustomPlayers()
	for i := range g.customPlayers {
		g.customColorMask[i] = 100
		if g.customPlayers[i].Type == 0 {
			g.customCardY[i] = 500
		} else {
			g.customCardY[i] = 100
		}
	}
}

// Symbol1302.onEnterFrame: slide0 -> originy+400 (500), slide1 -> originy
// (100), each with half-distance easing and integer rounding.
func (g *Game) updateCustomPlayerCards() {
	mx, my := ebiten.CursorPosition()
	mouseX, mouseY := float64(mx), float64(my)
	for i := range g.customPlayers {
		target := 100.0
		if g.customPlayers[i].Type == 0 {
			target = 500
		} else if g.customEditor[i] != 0 {
			// Symbol1302.slide=2 -> originy-400 = -300.
			target = -300
		}
		g.customCardY[i] += (target - g.customCardY[i]) / 2
		if math.Abs(g.customCardY[i]-target) < 0.5 {
			g.customCardY[i] = target
		} else if math.Mod(g.customCardY[i], 1) != 0 {
			g.customCardY[i] = math.Round(g.customCardY[i])
		}
		hovered := false
		if g.customPage == customPagePlayers && g.customPlayers[i].Type != 0 && g.customEditor[i] == 0 {
			cardX := g.customMenuX + 20 + float64(i)*220
			hovered = sourceColorSelectorRect(cardX, g.customCardY[i]).Contains(mouseX, mouseY)
		}
		g.customColorMask[i] = advanceSourceColorMask(g.customColorMask[i], hovered)
	}
}

func (g *Game) updateCustomWarning() {
	if g.customWarningFrame <= 0 {
		return
	}
	g.customWarningFrame++
	limit := 25
	if g.customWarning == 1 {
		limit = 15
	}
	if g.customWarningFrame > limit {
		g.customWarning = 0
		g.customWarningFrame = 0
	}
}

func (g *Game) playCustomWarning(kind int) {
	g.customWarning = kind
	// warning.play() starts a clip stopped on frame1; its first visible source
	// frame on the following tick is frame2.
	g.customWarningFrame = 2
}

func (g *Game) customPlayerSetupHitAt(x, y float64) int {
	for i := range g.customPlayers {
		cardX := 20.0 + float64(i)*220
		cardY := g.customCardY[i]
		if g.customPlayers[i].Type == 0 {
			if sourceMenuHitRect(customPlayerTypeButtonLocal, cardX+20, cardY-254.95, 1.0775146484375, 1).Contains(x, y) {
				return playerSlotHit(i, playerSlotActionHuman)
			}
			if sourceMenuHitRect(customPlayerTypeButtonLocal, cardX+20, cardY-220.95, 1.0775146484375, 1).Contains(x, y) {
				return playerSlotHit(i, playerSlotActionAI)
			}
		} else {
			if sourceMenuHitRect(customPlayerTypeButtonLocal, cardX+20, cardY+9.15, 1.0775146484375, 1).Contains(x, y) {
				return playerSlotHit(i, playerSlotActionClear)
			}
		}
	}
	return menuHitNone
}

func (g *Game) activateCustomPlayerSlot(hit int) bool {
	slot, action, ok := decodePlayerSlotHit(hit)
	if !ok {
		return false
	}
	switch action {
	case playerSlotActionHuman:
		g.customPlayers[slot].Type = 1
	case playerSlotActionAI:
		g.customPlayers[slot].Type = 2
	case playerSlotActionClear:
		g.customPlayers[slot].Type = 0
		g.customEditor[slot] = 0
		if g.customNameFocus == slot+1 {
			g.customNameFocus = 0
		}
	}
	return true
}

func (g *Game) activeCustomPlayerCount() int {
	n := 0
	for _, cfg := range g.customPlayers {
		if cfg.Type > 0 {
			n++
		}
	}
	return n
}

func (g *Game) drawCustomGameInteractions(screen *ebiten.Image) {
	if g.customPage == customPageGame {
		g.drawCustomModeButtons(screen)
		g.drawCustomModeOptions(screen)
	}
	if g.customPage == customPageMaps {
		g.drawCustomMapButtons(screen)
		g.drawCustomMapPreview(screen)
	}
	if g.customPage == customPagePlayers {
		for i, cfg := range g.customPlayers {
			cardX := g.customMenuX + 20 + float64(i)*220
			g.drawPlayerSetupCardMasked(screen, cfg, g.customEditor[i], i, cardX, g.customCardY[i], 1, g.customColorMask[i], false)
		}
	}

	if g.customWarningFrame <= 0 {
		return
	}
	switch g.customWarning {
	case 1:
		// Symbol1309.warning1 at local (-1397.35,388.1).
		drawSourceRaster(screen, g.assets.CustomWarningLives[g.customWarningFrame],
			g.customMenuX-1397.35, 388.1, 1, 1, 1)
	case 2:
		// Symbol1309.warning2 at local (312.6,249.3).
		drawSourceRaster(screen, g.assets.CustomWarningPlayers[g.customWarningFrame],
			g.customMenuX+312.6, 249.3, 1, 1, 1)
	}
}
