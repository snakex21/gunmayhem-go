package game

import "testing"

func TestCustomModeSourceControlsByMode(t *testing.T) {
	if !modeHasCrateToggle(SourceGameModeNormal) || !modeHasPickupToggle(SourceGameModeNormal) || !modeHasLives(SourceGameModeNormal) { t.Fatal("normal source controls missing") }
	if !modeHasCrateToggle(SourceGameModeInstagib) || modeHasPickupToggle(SourceGameModeInstagib) || !modeHasLives(SourceGameModeInstagib) { t.Fatal("instagib source controls mismatch") }
	if !modeHasCrateToggle(SourceGameModeTeams) || !modeHasPickupToggle(SourceGameModeTeams) || !modeHasLives(SourceGameModeTeams) { t.Fatal("team source controls mismatch") }
	if modeHasCrateToggle(SourceGameModeGunGame) || modeHasPickupToggle(SourceGameModeGunGame) || modeHasLives(SourceGameModeGunGame) { t.Fatal("gun game exposed source-absent options") }
	if modeHasCrateToggle(SourceGameModeSurvival) || modeHasPickupToggle(SourceGameModeSurvival) || modeHasLives(SourceGameModeSurvival) { t.Fatal("survival exposed source-absent options") }
}

// Mode options are kit controls on the MATCH tab (custom_local_menu.go); each
// one is live exactly where it is drawn and maps to its source option code.
func TestDevelopedCustomModeOptionHits(t *testing.T) {
	g := New()
	g.screen = screenCustomGame
	g.customMode = SourceGameModeNormal
	g.runCustomLocalUpdate(uiInput{})
	for id, row := range map[string]int{"cust.lives": 0, "cust.rounds": 1, "cust.crates": 2, "cust.pickups": 3, "cust.modifiers": 4} {
		it, ok := g.customUI.item(id)
		if !ok {
			t.Fatalf("%s missing", id)
		}
		if r := customLocalSettingRow(row); !(it.rect.Y >= r.Y && it.rect.Y+it.rect.H <= r.Y+r.H) {
			t.Fatalf("%s %+v is not inside its row %+v", id, it.rect, r)
		}
	}
	g.customMode = SourceGameModeInstagib
	g.runCustomLocalUpdate(uiInput{})
	if _, ok := g.customUI.item("cust.pickups"); ok {
		t.Fatal("1 HIT 1 KILL exposes the source-absent pickups toggle")
	}
	if _, ok := g.customUI.item("cust.crates"); !ok {
		t.Fatal("1 HIT 1 KILL lost its crates toggle")
	}
}

func TestCustomContinuePlaysSourceLivesWarning(t *testing.T) {
	g := &Game{customMode: SourceGameModeNormal, customLives: 0, customPage: customPageGame}
	g.activateCustomV2Hit(menuHitCustomV2Start)
	if g.customPage != customPageGame || g.customWarning != 1 || g.customWarningFrame != 2 {
		t.Fatalf("warning1 source flow page=%d kind=%d frame=%d", g.customPage, g.customWarning, g.customWarningFrame)
	}
}
