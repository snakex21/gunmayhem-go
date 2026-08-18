package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	campaignPlayerHitHumanBase   = 9000
	campaignPlayerHitClearBase   = 9010
	campaignPlayerHitEditBase    = 9020
	campaignPlayerHitBackBase    = 9050
	campaignPlayerHitSelectBase  = 9100
	campaignPlayerHitHatPageBase = 9700
	campaignPlayerHitColorBase   = 9800
	campaignPlayerHitNameBase    = 9900
)

func (g *Game) initCampaignPlayerSetup() {
	g.campaignPlayers = [2]CustomPlayerConfig{
		{Name: "Player 1", Color: 2, Shirt: 1, Hat: 1, Gun: 1, Perk: 7, Type: 1, Team: 1},
		{Name: "Player 2", Color: 5, Shirt: 1, Hat: 1, Gun: 1, Perk: 7, Type: 0, Team: 1},
	}
	g.campaignCardY[0] = 100
	g.campaignCardY[1] = 500
	g.campaignColorMask[0] = 100
	g.campaignColorMask[1] = 100
}

func (g *Game) updateCampaignPlayerCards() {
	mx, my := ebiten.CursorPosition()
	mouseX, mouseY := float64(mx), float64(my)
	for i := range g.campaignPlayers {
		target := 100.0
		if g.campaignPlayers[i].Type == 0 {
			target = 500
		} else if g.campaignEditor[i] != 0 {
			target = -300
		}
		g.campaignCardY[i] += (target - g.campaignCardY[i]) / 2
		if math.Abs(g.campaignCardY[i]-target) < 0.5 {
			g.campaignCardY[i] = target
		} else if math.Mod(g.campaignCardY[i], 1) != 0 {
			g.campaignCardY[i] = math.Round(g.campaignCardY[i])
		}
		hovered := g.campaignPlayers[i].Type != 0 && g.campaignEditor[i] == 0 &&
			sourceColorSelectorRect(campaignCardX(i), g.campaignCardY[i]).Contains(mouseX, mouseY)
		g.campaignColorMask[i] = advanceSourceColorMask(g.campaignColorMask[i], hovered)
	}
}

func campaignCardX(slot int) float64 { return 27 + float64(slot)*220 }

func (g *Game) campaignPlayerHitAt(x, y float64) int {
	for slot := 0; slot < 2; slot++ {
		cfg := g.campaignPlayers[slot]
		cx, cy := campaignCardX(slot), g.campaignCardY[slot]
		editor := g.campaignEditor[slot]
		if editor == 0 {
			if cfg.Type == 0 {
				// Campaign Symbol857 exposes only HUMAN PLAYER for an empty slot.
				if sourceMenuHitRect(customPlayerTypeButtonLocal, cx+20, cy-254.95, 1.0775146484375, 1).Contains(x, y) {
					return campaignPlayerHitHumanBase + slot
				}
				continue
			}
			// Symbol857 hides/moves btn_clear for menu1; only optional P2 can be
			// cleared in the source campaign setup.
			if slot == 1 && sourceMenuHitRect(customPlayerTypeButtonLocal, cx+20, cy+9.15, 1.0775146484375, 1).Contains(x, y) {
				return campaignPlayerHitClearBase + slot
			}
			edits := [...]struct {
				editor   int
				x, y, sx float64
			}{
				{customEditShirt, 17.5, 127.5, 0.538726806640625},
				{customEditHat1, 17.5, 93.0, 0.538726806640625},
				{customEditGun, 17.5, 161.7, 0.5387115478515625},
				{customEditPerk, 17.5, 220.75, 0.53875732421875},
			}
			for _, e := range edits {
				if sourceMenuHitRect(customPlayerTypeButtonLocal, cx+e.x, cy+e.y, e.sx, 1).Contains(x, y) {
					return campaignPlayerHitEditBase + slot*10 + e.editor
				}
			}
			if (Rect{X: cx + 32.5, Y: cy + 294.35, W: 139.55, H: 17.9}).Contains(x, y) {
				return campaignPlayerHitNameBase + slot
			}
			if n, ok := sourceColorNumberAt(cx, cy, x, y); ok {
				return campaignPlayerHitColorBase + slot*100 + n
			}
			continue
		}

		for _, item := range sourceCustomSelectionItems()[editor] {
			r := item.Rect
			r.X += cx
			r.Y += cy
			if r.Contains(x, y) && g.customSelectionUnlocked(editor, item.Number) {
				return campaignPlayerHitSelectBase + slot*1000 + editor*100 + item.Number
			}
		}
		if (editor == customEditHat1 || editor == customEditHat2) &&
			sourceMenuHitRect(customPlayerTypeButtonLocal, cx+27.25, cy+671.7, 1, 1).Contains(x, y) {
			return campaignPlayerHitHatPageBase + slot
		}
		if sourceMenuHitRect(customPlayerTypeButtonLocal, cx+27.5, cy+759.45, 1, 1).Contains(x, y) {
			return campaignPlayerHitBackBase + slot
		}
	}
	return campaignHitNone
}

func (g *Game) activateCampaignPlayerHit(hit int) bool {
	if hit >= campaignPlayerHitHumanBase && hit < campaignPlayerHitHumanBase+2 {
		slot := hit - campaignPlayerHitHumanBase
		g.campaignPlayers[slot].Type = 1
		return true
	}
	if hit >= campaignPlayerHitClearBase && hit < campaignPlayerHitClearBase+2 {
		slot := hit - campaignPlayerHitClearBase
		if slot == 1 {
			g.campaignPlayers[slot].Type = 0
			g.campaignEditor[slot] = 0
		}
		return true
	}
	if hit >= campaignPlayerHitEditBase && hit < campaignPlayerHitEditBase+20 {
		v := hit - campaignPlayerHitEditBase
		slot, editor := v/10, v%10
		if slot < 2 && editor >= 1 && editor <= 4 {
			for i := range g.campaignEditor {
				g.campaignEditor[i] = 0
			}
			if editor == customEditHat1 && g.campaignPlayers[slot].Hat > 12 {
				editor = customEditHat2
			}
			g.campaignEditor[slot] = editor
			g.campaignNameFocus = 0
			return true
		}
	}
	if hit >= campaignPlayerHitBackBase && hit < campaignPlayerHitBackBase+2 {
		g.campaignEditor[hit-campaignPlayerHitBackBase] = 0
		return true
	}
	if hit >= campaignPlayerHitHatPageBase && hit < campaignPlayerHitHatPageBase+2 {
		slot := hit - campaignPlayerHitHatPageBase
		if g.campaignEditor[slot] == customEditHat1 {
			g.campaignEditor[slot] = customEditHat2
		} else {
			g.campaignEditor[slot] = customEditHat1
		}
		return true
	}
	if hit >= campaignPlayerHitColorBase && hit < campaignPlayerHitColorBase+200 {
		v := hit - campaignPlayerHitColorBase
		slot, n := v/100, v%100
		if slot < 2 && n >= 1 && n <= 10 {
			g.campaignPlayers[slot].Color = n
			return true
		}
	}
	if hit >= campaignPlayerHitNameBase && hit < campaignPlayerHitNameBase+2 {
		g.campaignNameFocus = hit - campaignPlayerHitNameBase + 1
		return true
	}
	if hit >= campaignPlayerHitSelectBase && hit < campaignPlayerHitSelectBase+2000 {
		v := hit - campaignPlayerHitSelectBase
		slot := v / 1000
		v %= 1000
		editor, n := v/100, v%100
		if slot < 2 && editor >= 1 && editor <= 5 && n > 0 {
			cfg := &g.campaignPlayers[slot]
			switch editor {
			case customEditShirt:
				cfg.Shirt = n
			case customEditHat1, customEditHat2:
				cfg.Hat = n
			case customEditGun:
				cfg.Gun = n
			case customEditPerk:
				cfg.Perk = n
			}
			g.campaignEditor[slot] = 0
			return true
		}
	}
	return false
}

func (g *Game) updateCampaignNameInput() {
	if g.campaignNameFocus <= 0 || g.campaignNameFocus > 2 || g.screen != screenCampaign {
		return
	}
	slot := g.campaignNameFocus - 1
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.campaignNameFocus = 0
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		r := []rune(g.campaignPlayers[slot].Name)
		if len(r) > 0 {
			g.campaignPlayers[slot].Name = string(r[:len(r)-1])
		}
	}
	for _, ch := range ebiten.AppendInputChars(nil) {
		r := []rune(g.campaignPlayers[slot].Name)
		if ch >= 32 && ch != 127 && len(r) < 20 {
			g.campaignPlayers[slot].Name += string(ch)
		}
	}
}

func (g *Game) drawCampaignPlayerCards(screen *ebiten.Image, alpha float64) {
	for slot := 0; slot < 2; slot++ {
		cfg := g.campaignPlayers[slot]
		g.drawPlayerSetupCardMasked(screen, cfg, g.campaignEditor[slot], slot, campaignCardX(slot), g.campaignCardY[slot], alpha, g.campaignColorMask[slot], true)
	}
}
