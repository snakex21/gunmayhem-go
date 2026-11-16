package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Campaign player codes: the campaign briefing (P2 ADD / REMOVE) and the
// campaign Loadout (campaign_ui.go, loadoutBinding) write P1 / P2 through
// activateCampaignPlayerHit. Every family owns a disjoint range (see
// TestCampaignPlayerSetupHitCodesAreUnique).
const (
	campaignHitNone            = 0
	campaignPlayerHitHumanBase = 9000 // + slot
	campaignPlayerHitClearBase = 9010 // + slot
	campaignPlayerHitNameBase  = 9900 // + slot
	campaignPlayerHitCustomGun = 12000
	campaignPlayerHitThrowable = 13000
	// campaignPlayerHitSelectBase + slot*campaignSelectSlotStride +
	// editor*campaignSelectEditorStride + item number.
	campaignPlayerHitSelectBase = 20000
	campaignSelectSlotStride    = 1000
	campaignSelectEditorStride  = 100
)

func campaignSelectHit(slot, editor, number int) int {
	return campaignPlayerHitSelectBase + slot*campaignSelectSlotStride + editor*campaignSelectEditorStride + number
}

func campaignThrowableChoiceHit(slot, kind int) int {
	if kind < 0 || kind >= throwableTypeCount {
		return campaignHitNone
	}
	return campaignPlayerHitThrowable + slot*throwableHitStride + kind + 1
}

func campaignDevelopedGunHit(slot, number int) int {
	return campaignPlayerHitCustomGun + slot*developedWeaponSlotCount + number - 98
}

func decodeCampaignDevelopedGunHit(hit int) (slot, number int, ok bool) {
	v := hit - campaignPlayerHitCustomGun
	if v < 0 || v >= 2*developedWeaponSlotCount {
		return 0, 0, false
	}
	slot, number = v/developedWeaponSlotCount, 98+v%developedWeaponSlotCount
	_, exists := developedWeaponSpec(number)
	return slot, number, slot >= 0 && slot < 2 && exists && isStarterHandgunNumber(number)
}

func (g *Game) initCampaignPlayerSetup() {
	g.campaignPlayers = [2]CustomPlayerConfig{
		{Name: "Player 1", Color: 2, Shirt: 1, Hat: 1, Eyes: 1, Gun: 1, Perk: 7, Type: 1, Team: 1},
		{Name: "Player 2", Color: 5, Shirt: 1, Hat: 1, Eyes: 1, Gun: 1, Perk: 7, Type: 0, Team: 1},
	}
}

func (g *Game) activateCampaignPlayerHit(hit int) bool {
	if hit >= campaignPlayerHitThrowable && hit < campaignPlayerHitThrowable+2*throwableHitStride {
		v := hit - campaignPlayerHitThrowable
		slot, action := v/throwableHitStride, v%throwableHitStride
		kind := action - 1
		if slot < 2 && kind >= 0 && kind < throwableTypeCount {
			g.campaignPlayers[slot].Throwable = kind
			return true
		}
	}
	if slot, number, ok := decodeCampaignDevelopedGunHit(hit); ok {
		g.campaignPlayers[slot].Gun = number
		return true
	}
	if hit >= campaignPlayerHitHumanBase && hit < campaignPlayerHitHumanBase+2 {
		slot := hit - campaignPlayerHitHumanBase
		g.campaignPlayers[slot].Type = 1
		return true
	}
	if hit >= campaignPlayerHitClearBase && hit < campaignPlayerHitClearBase+2 {
		slot := hit - campaignPlayerHitClearBase
		if slot == 1 {
			g.campaignPlayers[slot].Type = 0
		}
		return true
	}
	if hit >= campaignPlayerHitNameBase && hit < campaignPlayerHitNameBase+2 {
		g.campaignNameFocus = hit - campaignPlayerHitNameBase + 1
		return true
	}
	if hit >= campaignPlayerHitSelectBase && hit < campaignPlayerHitSelectBase+2*campaignSelectSlotStride {
		v := hit - campaignPlayerHitSelectBase
		slot := v / campaignSelectSlotStride
		v %= campaignSelectSlotStride
		editor, n := v/campaignSelectEditorStride, v%campaignSelectEditorStride
		if slot < 2 && ((editor >= 1 && editor <= 5) || editor == customEditEyes || editor == customEditThrowable || editor == customEditColor || editor == customEditAbility) && n > 0 {
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
			case customEditEyes:
				cfg.Eyes = n
			case customEditColor:
				if n >= 1 && n <= maxDevelopedColorValue() {
					cfg.Color = n
				}
			case customEditThrowable:
				kind := n - 1
				if kind >= 0 && kind < throwableTypeCount {
					cfg.Throwable = kind
				}
			case customEditAbility:
				if ability, ok := abilityFromMenuNumber(n); ok {
					cfg.Ability = ability
				}
			}
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
	if textEraseRepeated() {
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

func (g *Game) campaignSelectionUnlocked(editor, number int) bool {
	if g.campaignSetExtra && editor == customEditPerk {
		// EXTRA is the expanded campaign: every real perk tile (1..9) is
		// available from its player setup, independently of GM1 progress.
		return isSelectablePerk(number)
	}
	return g.customSelectionUnlocked(editor, number)
}
