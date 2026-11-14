package game

import (
	"fmt"
	"testing"
)

// Legacy screens still use numeric hit codes. Every code a screen can produce
// must mean exactly one thing on that screen (extends the campaign pattern in
// campaign_hitcodes_test.go). Regression B1: the card editor's developed gun
// codes (8000-8927) overlapped the mode options, modifiers and chaos setup.

type hitCodeSet struct {
	t     *testing.T
	name  string
	codes map[int]string
}

func newHitCodeSet(t *testing.T, name string) *hitCodeSet {
	return &hitCodeSet{t: t, name: name, codes: map[int]string{}}
}

func (s *hitCodeSet) add(code int, label string) {
	s.t.Helper()
	if code == menuHitNone {
		s.t.Fatalf("%s: %s produced menuHitNone", s.name, label)
	}
	if prev, dup := s.codes[code]; dup {
		s.t.Fatalf("%s: hit code %d is both %q and %q", s.name, code, prev, label)
	}
	s.codes[code] = label
}

// customPlayerEditorCodes enumerates every code the four custom player cards
// and their editors can return.
func customPlayerEditorCodes() map[int]string {
	out := map[int]string{}
	add := func(code int, label string) { out[code] = label }
	editors := []int{customEditShirt, customEditHat1, customEditGun, customEditPerk, customEditEyes, customEditThrowable, customEditColor, customEditAbility}
	for slot := 0; slot < 4; slot++ {
		for action := playerSlotActionHuman; action <= playerSlotActionClear; action++ {
			add(playerSlotHit(slot, action), fmt.Sprintf("slot%d action%d", slot, action))
		}
		add(customEditorBackHit(slot), fmt.Sprintf("editor back slot%d", slot))
		add(customHatPageHit(slot), fmt.Sprintf("hat page slot%d", slot))
		add(menuHitPlayerColorBase+500+slot, fmt.Sprintf("name slot%d", slot))
		add(customTeamHit(slot, 1), fmt.Sprintf("team A slot%d", slot))
		add(customTeamHit(slot, 2), fmt.Sprintf("team B slot%d", slot))
		for _, editor := range editors {
			add(customEditHit(slot, editor), fmt.Sprintf("open editor %d slot%d", editor, slot))
			for _, item := range developedCustomSelectionItems(editor) {
				n := item.Number
				switch {
				case editor == customEditGun && n >= 98:
					add(customDevelopedGunHit(slot, n), fmt.Sprintf("developed gun %d slot%d", n, slot))
				case editor == customEditThrowable:
					add(customThrowableChoiceHit(slot, n-1), fmt.Sprintf("throwable %d slot%d", n-1, slot))
				default:
					add(customSelectHit(slot, editor, n), fmt.Sprintf("editor %d item %d slot%d", editor, n, slot))
				}
			}
		}
	}
	return out
}

func TestCustomGameHitCodesAreUnique(t *testing.T) {
	s := newHitCodeSet(t, "custom game")
	for code := menuHitCustomV2TabSession; code <= menuHitCustomV2ApplyAll; code++ {
		s.add(code, fmt.Sprintf("v2 control %d", code))
	}
	for round := 1; round <= customSeriesMaxRounds; round++ {
		s.add(menuHitCustomV2RoundBase+round, fmt.Sprintf("round %d", round))
	}
	maps := len(customV2MapChoices(false))
	if gm2 := len(customV2MapChoices(true)); gm2 > maps {
		maps = gm2
	}
	for i := 0; i < maps; i++ {
		s.add(menuHitCustomV2MapBase+i, fmt.Sprintf("map choice %d", i))
	}
	for slot := 1; slot < 4; slot++ {
		s.add(menuHitCustomV2SlotRemoveBase+slot, fmt.Sprintf("remove slot %d", slot))
	}
	s.add(menuHitCustomSessionLocal, "LOCAL")
	s.add(menuHitCustomSessionOnline, "ONLINE")
	s.add(menuHitCustomSessionNetwork, "session status")
	s.add(menuHitCustomSessionRules, "RULES")
	for _, row := range customV2ModeRows {
		s.add(menuHitModeBase+row.mode, "mode "+row.label)
	}
	for code := menuHitModeCrates; code <= menuHitModeLivesPlus; code++ {
		s.add(code, fmt.Sprintf("mode option %d", code))
	}
	for code := menuHitCustomModifiersOpen; code <= menuHitCustomModifiersRoundNext; code++ {
		s.add(code, fmt.Sprintf("modifier control %d", code))
	}
	for _, spec := range customModifierSpecs {
		s.add(menuHitCustomModifierBase+spec.ID, "modifier "+spec.Label)
	}
	for code := menuHitChaosSetup; code <= menuHitChaosClear; code++ {
		s.add(code, fmt.Sprintf("chaos control %d", code))
	}
	for _, spec := range chaosEffectSpecs {
		s.add(menuHitChaosEffectBase+spec.ID, "chaos "+spec.Label)
	}
	editorCodes := customPlayerEditorCodes()
	for code, label := range editorCodes {
		s.add(code, label)
	}
	if len(editorCodes) < 400 {
		t.Fatalf("suspiciously few player editor codes: %d", len(editorCodes))
	}
}

// B1 regression: no card-editor code may be consumed by the mode options
// (checked first by activateCustomV2Hit), and each developed pistol code
// selects that pistol for its own slot.
func TestCustomPlayerEditorCodesReachTheEditor(t *testing.T) {
	for code, label := range customPlayerEditorCodes() {
		g := New()
		before := fmt.Sprintf("%v %v %v %d %d", g.CrateON, g.PowerON, g.customLivesFocus, g.customLives, g.customSeriesRounds)
		if g.activateCustomModeOption(code) {
			t.Fatalf("%s (code %d) was consumed by the mode options", label, code)
		}
		if after := fmt.Sprintf("%v %v %v %d %d", g.CrateON, g.PowerON, g.customLivesFocus, g.customLives, g.customSeriesRounds); after != before {
			t.Fatalf("%s (code %d) changed match settings", label, code)
		}
	}
	checked := 0
	for slot := 0; slot < 4; slot++ {
		for _, item := range developedCustomSelectionItems(customEditGun) {
			if item.Number < 98 {
				continue
			}
			g := New()
			g.customPage = customPagePlayers
			g.customPlayers[slot].Type = 1
			g.customEditor[slot] = customEditGun
			g.activateCustomV2Hit(customDevelopedGunHit(slot, item.Number))
			if g.customPlayers[slot].Gun != item.Number {
				t.Fatalf("slot %d pistol %d: gun=%d", slot, item.Number, g.customPlayers[slot].Gun)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no developed pistols enumerated")
	}
}

func TestCustomNetworkBrowserHitCodesAreUnique(t *testing.T) {
	s := newHitCodeSet(t, "network browser")
	for code := menuHitCustomSessionLocal; code <= menuHitCustomNetworkJoinSelected; code++ {
		s.add(code, fmt.Sprintf("session control %d", code))
	}
	for i := 0; i < menuHitCustomHostBack-menuHitCustomNetworkServerBase; i++ {
		s.add(menuHitCustomNetworkServerBase+i, fmt.Sprintf("server row %d", i))
	}
	for code := menuHitCustomHostBack; code <= menuHitCustomHostLeave; code++ {
		s.add(code, fmt.Sprintf("host setup control %d", code))
	}
}

func TestOnlineOverlayHitCodesAreUnique(t *testing.T) {
	s := newHitCodeSet(t, "online overlays")
	for code := menuHitMultiplayerLobbySetup; code <= menuHitOnlineBlockNextPage; code++ {
		s.add(code, fmt.Sprintf("lobby control %d", code))
	}
	for field := 0; field < onlineCharFieldCount; field++ {
		s.add(onlineCharacterHit(field, -1), fmt.Sprintf("character %d -", field))
		s.add(onlineCharacterHit(field, 1), fmt.Sprintf("character %d +", field))
		s.add(onlineCharacterVisualFieldHit(field), fmt.Sprintf("character picture %d", field))
	}
	for i := range onlineAllWeaponNumbers() {
		s.add(menuHitOnlineBlockBase+i, fmt.Sprintf("weapon block %d", i))
	}
	for perk := 1; perk < onlinePerkTableSize; perk++ {
		s.add(menuHitOnlinePerkBase+perk, fmt.Sprintf("perk block %d", perk))
	}
	for ability := 1; ability < abilityCount; ability++ {
		s.add(menuHitOnlineAbilityBase+ability, fmt.Sprintf("ability block %d", ability))
	}
	for i := 0; i < 100; i++ {
		s.add(onlineVisualChoiceHit(i), fmt.Sprintf("picker choice %d", i))
	}
	s.add(menuHitOnlineVisualPrevPage, "picker prev")
	s.add(menuHitOnlineVisualNextPage, "picker next")
	for i := 0; i < 4; i++ {
		s.add(menuHitOnlineRuleMapChoiceBase+i, fmt.Sprintf("map rule %d", i))
	}
	for i := 0; i < 3; i++ {
		s.add(menuHitOnlineRuleWeaponChoiceBase+i, fmt.Sprintf("weapon rule %d", i))
	}
	s.add(menuHitOnlineRuleOnlyWeaponPicker, "only weapon picker")
	s.add(menuHitOnlineRuleFixedMapPicker, "fixed map picker")
}
