package game

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	customEditShirt     = 1
	customEditHat1      = 2
	customEditGun       = 3
	customEditPerk      = 4
	customEditHat2      = 5
	customEditEyes      = 6
	customEditThrowable = 7
	customEditColor     = 8
	customEditAbility   = 9 // developed ABILITY / GEAR page (abilities.go)
)

const (
	menuHitPlayerSelectBase = 2000
	menuHitPlayerTeamBase   = 7100
	menuHitPlayerColorBase  = 7200
	// B1: was 8000, whose 8000-8927 range overlapped the mode options (8001-8007),
	// modifiers (8200+) and chaos (8700+). Every family now owns a disjoint
	// range, see TestCustomGameHitCodesAreUnique.
	menuHitPlayerCustomGunBase = 30000
	menuHitPlayerThrowableBase = 16000
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

func customSelectHit(slot, editor, number int) int {
	return menuHitPlayerSelectBase + slot*1000 + editor*100 + number
}
func customTeamHit(slot, team int) int    { return menuHitPlayerTeamBase + slot*10 + team }
func customColorHit(slot, number int) int { return menuHitPlayerColorBase + slot*100 + number }
func customThrowableChoiceHit(slot, kind int) int {
	if kind < 0 || kind >= throwableTypeCount {
		return menuHitNone
	}
	return menuHitPlayerThrowableBase + slot*throwableHitStride + kind + 1
}

func customDevelopedGunHit(slot, number int) int {
	return menuHitPlayerCustomGunBase + slot*developedWeaponSlotCount + number - 98
}

func decodeCustomDevelopedGunHit(hit int) (slot, number int, ok bool) {
	v := hit - menuHitPlayerCustomGunBase
	if v < 0 || v >= 4*developedWeaponSlotCount {
		return 0, 0, false
	}
	slot, number = v/developedWeaponSlotCount, 98+v%developedWeaponSlotCount
	_, exists := developedWeaponSpec(number)
	return slot, number, slot >= 0 && slot < 4 && exists && isStarterHandgunNumber(number)
}

func decodeCustomSelectHit(hit int) (slot, editor, number int, ok bool) {
	v := hit - menuHitPlayerSelectBase
	if v < 0 || v >= 4000 {
		return 0, 0, 0, false
	}
	slot = v / 1000
	v %= 1000
	editor = v / 100
	number = v % 100
	if slot < 0 || slot >= 4 || editor < 1 || editor > customEditAbility || number <= 0 {
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

func (g *Game) customSelectionUnlocked(editor, number int) bool {
	if editor != customEditPerk {
		return true
	}
	// Developed build: all real perks are available immediately. The Flash
	// source locked perks 3/6/9 behind GM1 campaign progress, but that made
	// visible choices such as GUN SPAWN appear broken until those missions had
	// been completed. Keep only the real 1..9 perks plus the developed perks
	// 14+ (10..13 are reserved for GM2 perks); frame 10 is not a selectable
	// perk.
	return isSelectablePerk(number)
}

func (g *Game) activateCustomPlayerEditorHit(hit int) bool {
	if hit >= menuHitPlayerThrowableBase && hit < menuHitPlayerThrowableBase+4*throwableHitStride {
		v := hit - menuHitPlayerThrowableBase
		slot, action := v/throwableHitStride, v%throwableHitStride
		kind := action - 1
		if slot >= 0 && slot < 4 && kind >= 0 && kind < throwableTypeCount {
			if !g.onlineMayEditCustomCharacterSlot(slot) {
				return true
			}
			g.customPlayers[slot].Throwable = kind
			return true
		}
	}
	if slot, number, ok := decodeCustomDevelopedGunHit(hit); ok {
		if !g.onlineMayEditCustomCharacterSlot(slot) {
			return true
		}
		g.customPlayers[slot].Gun = number
		return true
	}
	if slot, editor, number, ok := decodeCustomSelectHit(hit); ok {
		if !g.onlineMayEditCustomCharacterSlot(slot) {
			return true
		}
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
		case customEditEyes:
			cfg.Eyes = number
		case customEditColor:
			if number >= 1 && number <= maxDevelopedColorValue() {
				cfg.Color = number
			}
		case customEditThrowable:
			kind := number - 1
			if kind >= 0 && kind < throwableTypeCount {
				cfg.Throwable = kind
			}
		case customEditAbility:
			if ability, ok := abilityFromMenuNumber(number); ok {
				cfg.Ability = ability
			}
		}
		return true
	}
	if hit >= menuHitPlayerTeamBase && hit < menuHitPlayerTeamBase+40 {
		v := hit - menuHitPlayerTeamBase
		slot, team := v/10, v%10
		if slot >= 0 && slot < 4 && (team == 1 || team == 2) {
			if !g.onlineMayEditCustomCharacterSlot(slot) {
				return true
			}
			g.customPlayers[slot].Team = team
			return true
		}
	}
	if hit >= menuHitPlayerColorBase && hit < menuHitPlayerColorBase+400 {
		v := hit - menuHitPlayerColorBase
		slot, number := v/100, v%100
		if slot >= 0 && slot < 4 && number >= 1 && number <= maxDevelopedColorValue() {
			if !g.onlineMayEditCustomCharacterSlot(slot) {
				return true
			}
			g.customPlayers[slot].Color = number
			return true
		}
	}
	// Dedicated name focus ids are base+500+slot.
	if hit >= menuHitPlayerColorBase+500 && hit < menuHitPlayerColorBase+504 {
		slot := hit - (menuHitPlayerColorBase + 500)
		if !g.onlineMayEditCustomCharacterSlot(slot) {
			g.customNameFocus = 0
			return true
		}
		g.customNameFocus = slot + 1
		return true
	}
	return false
}

func (g *Game) updateCustomNameInput() {
	if g.customNameFocus <= 0 || g.customNameFocus > 4 || g.customPage != customPagePlayers {
		return
	}
	slot := g.customNameFocus - 1
	if !g.onlineMayEditCustomCharacterSlot(slot) {
		g.customNameFocus = 0
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.customNameFocus = 0
		return
	}
	if textEraseRepeated() {
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
