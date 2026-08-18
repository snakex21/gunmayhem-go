package game

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	customEditShirt = 1
	customEditHat1  = 2
	customEditGun   = 3
	customEditPerk  = 4
	customEditHat2  = 5
)

const (
	menuHitPlayerEditBase       = 1000
	menuHitPlayerEditorBackBase = 1100
	menuHitPlayerSelectBase     = 2000
	menuHitPlayerHatPageBase    = 7000
	menuHitPlayerTeamBase       = 7100
	menuHitPlayerColorBase      = 7200
)

type customSelectionItem struct {
	Editor int
	Number int
	Rect   Rect
}

var (
	customSelectionOnce sync.Once
	customSelectionData map[int][]customSelectionItem
)

func customEditHit(slot, editor int) int { return menuHitPlayerEditBase + slot*10 + editor }
func customEditorBackHit(slot int) int   { return menuHitPlayerEditorBackBase + slot }
func customSelectHit(slot, editor, number int) int {
	return menuHitPlayerSelectBase + slot*1000 + editor*100 + number
}
func customHatPageHit(slot int) int       { return menuHitPlayerHatPageBase + slot }
func customTeamHit(slot, team int) int    { return menuHitPlayerTeamBase + slot*10 + team }
func customColorHit(slot, number int) int { return menuHitPlayerColorBase + slot*100 + number }

func decodeCustomSelectHit(hit int) (slot, editor, number int, ok bool) {
	v := hit - menuHitPlayerSelectBase
	if v < 0 || v >= 4000 {
		return 0, 0, 0, false
	}
	slot = v / 1000
	v %= 1000
	editor = v / 100
	number = v % 100
	if slot < 0 || slot >= 4 || editor < 1 || editor > 5 || number <= 0 {
		return 0, 0, 0, false
	}
	return slot, editor, number, true
}

func sourceCustomSelectionItems() map[int][]customSelectionItem {
	customSelectionOnce.Do(func() {
		customSelectionData = map[int][]customSelectionItem{}
		doc, err := loadXFLVectorDoc("Symbol 835")
		if err != nil {
			return
		}
		expected := map[int]string{
			customEditShirt: "Symbol 811",
			customEditHat1:  "Symbol 813",
			customEditGun:   "Symbol 817",
			customEditPerk:  "Symbol 831",
			customEditHat2:  "Symbol 833",
		}
		for editor := 1; editor <= 5; editor++ {
			lib := expected[editor]
			bounds, err := sourceFrameVisualBounds(lib, 0)
			if err != nil {
				continue
			}
			for _, layer := range doc.Timeline.Layers {
				frame, ok := activeVectorFrame(layer.Frames, editor-1)
				if !ok {
					continue
				}
				for _, inst := range frame.Elements.Instances {
					if inst.Library != lib {
						continue
					}
					m := matrixFromVector(inst.Matrix.Value)
					n := sourceCustomizerNumber(editor, m.TX, m.TY)
					if n <= 0 {
						continue
					}
					customSelectionData[editor] = append(customSelectionData[editor], customSelectionItem{
						Editor: editor,
						Number: n,
						Rect:   transformRect(bounds, m),
					})
				}
			}
		}
	})
	return customSelectionData
}

func sourceCustomizerNumber(editor int, x, y float64) int {
	xx := 0
	if x < 140 {
		xx = 3
	}
	if x < 80 {
		xx = 2
	}
	if x < 15 {
		xx = 1
	}
	if xx == 0 {
		return 0
	}

	yy := 0
	switch editor {
	case customEditShirt, customEditHat1, customEditHat2:
		if y < 660 {
			yy = 4
		}
		if y < 600 {
			yy = 3
		}
		if y < 540 {
			yy = 2
		}
		if y < 480 {
			yy = 1
		}
		if y < 420 {
			yy = 0
		}
	case customEditGun, customEditPerk:
		if y < 660 {
			yy = 3
		}
		if y < 600 {
			yy = 2
		}
		if y < 540 {
			yy = 1
		}
		if y < 480 {
			yy = 0
		}
	}
	n := yy*3 + xx
	if editor == customEditHat2 {
		n += 12
	}
	return n
}

func (g *Game) customPlayerEditorHitAt(x, y float64) int {
	for slot := range g.customPlayers {
		if g.customPlayers[slot].Type == 0 {
			continue
		}
		cardX := 20.0 + float64(slot)*220
		cardY := g.customCardY[slot]
		editor := g.customEditor[slot]
		if editor == 0 {
			// Four exact Symbol806 EDIT hit areas from Symbol1302 frame3.
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
				if sourceMenuHitRect(customPlayerTypeButtonLocal, cardX+e.x, cardY+e.y, e.sx, 1).Contains(x, y) {
					return customEditHit(slot, e.editor)
				}
			}

			// Name field is a real DOMInputText. Give focus on mouse press/release
			// by returning a dedicated hit id in the same onRelease pipeline.
			if (Rect{X: cardX + 32.5, Y: cardY + 294.35, W: 139.55, H: 17.9}).Contains(x, y) {
				return menuHitPlayerColorBase + 500 + slot // name focus marker
			}

			if number, ok := sourceColorNumberAt(cardX, cardY, x, y); ok {
				return customColorHit(slot, number)
			}

			if g.customMode == SourceGameModeTeams {
				// Symbol804 at card local (22,360.25); Symbol800 teamA/B children.
				teamA := sourceMenuHitRect(Rect{X: 0, Y: 0, W: 148.5, H: 24}, cardX+22, cardY+360.25, 0.496795654296875, 1)
				teamB := sourceMenuHitRect(Rect{X: 0, Y: 0, W: 148.5, H: 24}, cardX+22+82.05, cardY+360.25, 0.496795654296875, 1)
				if teamA.Contains(x, y) {
					return customTeamHit(slot, 1)
				}
				if teamB.Contains(x, y) {
					return customTeamHit(slot, 2)
				}
			}
			continue
		}

		// During slide=2 the entire Symbol1302 card sits at originY-400, while
		// Symbol835 itself stays at local (0,0). Use parsed source child bounds.
		for _, item := range sourceCustomSelectionItems()[editor] {
			r := item.Rect
			r.X += cardX
			r.Y += cardY
			if r.Contains(x, y) && g.customSelectionUnlocked(editor, item.Number) {
				return customSelectHit(slot, editor, item.Number)
			}
		}

		// Hat page switch: Symbol806 at Symbol835 local (27.25,671.7).
		if editor == customEditHat1 || editor == customEditHat2 {
			if sourceMenuHitRect(customPlayerTypeButtonLocal, cardX+27.25, cardY+671.7, 1, 1).Contains(x, y) {
				return customHatPageHit(slot)
			}
		}
		// Symbol1302.btn_back is local (27.5,759.45).
		if sourceMenuHitRect(customPlayerTypeButtonLocal, cardX+27.5, cardY+759.45, 1, 1).Contains(x, y) {
			return customEditorBackHit(slot)
		}
	}
	return menuHitNone
}

func sourceColorNumberAt(cardX, cardY, x, y float64) (int, bool) {
	// Use the very same source matrix as rendering. Keeping hit-testing and
	// drawing on one transform prevents the visible palette drifting away from
	// its clickable cells after window/UI changes.
	m := sourceColorTransform(cardX, cardY)
	if !m.IsInvertible() {
		return 0, false
	}
	m.Invert()
	lx, ly := m.Apply(x, y)
	if lx < 0 || lx >= 50 || ly < 0 || ly >= 125 {
		return 0, false
	}
	n := int(math.Floor(lx/25))*5 + int(math.Floor(ly/25)) + 1
	return n, n >= 1 && n <= 10
}

func (g *Game) customSelectionUnlocked(editor, number int) bool {
	if editor != customEditPerk {
		return true
	}
	// Symbol831: perk3 requires level2 completed, perk6 level5, perk9 level6.
	switch number {
	case 3:
		return g.campaignLevels[1] == 2
	case 6:
		return g.campaignLevels[4] == 2
	case 9:
		return g.campaignLevels[5] == 2
	}
	return number != 10
}

func (g *Game) activateCustomPlayerEditorHit(hit int) bool {
	if hit >= menuHitPlayerEditBase && hit < menuHitPlayerEditBase+40 {
		v := hit - menuHitPlayerEditBase
		slot, editor := v/10, v%10
		if slot >= 0 && slot < 4 && editor >= 1 && editor <= 4 {
			for i := range g.customEditor {
				g.customEditor[i] = 0
			}
			if editor == customEditHat1 && g.customPlayers[slot].Hat > 12 {
				editor = customEditHat2
			}
			g.customEditor[slot] = editor
			g.customNameFocus = 0
			return true
		}
	}
	if hit >= menuHitPlayerEditorBackBase && hit < menuHitPlayerEditorBackBase+4 {
		slot := hit - menuHitPlayerEditorBackBase
		g.customEditor[slot] = 0
		return true
	}
	if slot, editor, number, ok := decodeCustomSelectHit(hit); ok {
		cfg := &g.customPlayers[slot]
		switch editor {
		case customEditShirt:
			cfg.Shirt = number
		case customEditHat1, customEditHat2:
			cfg.Hat = number
		case customEditGun:
			cfg.Gun = number
		case customEditPerk:
			cfg.Perk = number
		}
		g.customEditor[slot] = 0
		return true
	}
	if hit >= menuHitPlayerHatPageBase && hit < menuHitPlayerHatPageBase+4 {
		slot := hit - menuHitPlayerHatPageBase
		if g.customEditor[slot] == customEditHat1 {
			g.customEditor[slot] = customEditHat2
		} else {
			g.customEditor[slot] = customEditHat1
		}
		return true
	}
	if hit >= menuHitPlayerTeamBase && hit < menuHitPlayerTeamBase+40 {
		v := hit - menuHitPlayerTeamBase
		slot, team := v/10, v%10
		if slot >= 0 && slot < 4 && (team == 1 || team == 2) {
			g.customPlayers[slot].Team = team
			return true
		}
	}
	if hit >= menuHitPlayerColorBase && hit < menuHitPlayerColorBase+400 {
		v := hit - menuHitPlayerColorBase
		slot, number := v/100, v%100
		if slot >= 0 && slot < 4 && number >= 1 && number <= 10 {
			g.customPlayers[slot].Color = number
			return true
		}
	}
	// Dedicated name focus ids are base+500+slot.
	if hit >= menuHitPlayerColorBase+500 && hit < menuHitPlayerColorBase+504 {
		g.customNameFocus = hit - (menuHitPlayerColorBase + 500) + 1
		return true
	}
	return false
}

func (g *Game) updateCustomNameInput() {
	if g.customNameFocus <= 0 || g.customNameFocus > 4 || g.customPage != customPagePlayers {
		return
	}
	slot := g.customNameFocus - 1
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.customNameFocus = 0
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		r := []rune(g.customPlayers[slot].Name)
		if len(r) > 0 {
			g.customPlayers[slot].Name = string(r[:len(r)-1])
		}
	}
	chars := ebiten.AppendInputChars(nil)
	if len(chars) > 0 {
		r := []rune(g.customPlayers[slot].Name)
		for _, ch := range chars {
			if ch >= 32 && ch != 127 && len(r) < 20 {
				r = append(r, ch)
			}
		}
		g.customPlayers[slot].Name = string(r)
	}
}
