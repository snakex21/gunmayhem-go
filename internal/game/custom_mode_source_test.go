package game

import "testing"

func TestCustomModeSourceControlsByMode(t *testing.T) {
	if !modeHasCrateToggle(SourceGameModeNormal) || !modeHasPickupToggle(SourceGameModeNormal) || !modeHasLives(SourceGameModeNormal) { t.Fatal("normal source controls missing") }
	if !modeHasCrateToggle(SourceGameModeInstagib) || modeHasPickupToggle(SourceGameModeInstagib) || !modeHasLives(SourceGameModeInstagib) { t.Fatal("instagib source controls mismatch") }
	if !modeHasCrateToggle(SourceGameModeTeams) || !modeHasPickupToggle(SourceGameModeTeams) || !modeHasLives(SourceGameModeTeams) { t.Fatal("team source controls mismatch") }
	if modeHasCrateToggle(SourceGameModeGunGame) || modeHasPickupToggle(SourceGameModeGunGame) || modeHasLives(SourceGameModeGunGame) { t.Fatal("gun game exposed source-absent options") }
	if modeHasCrateToggle(SourceGameModeSurvival) || modeHasPickupToggle(SourceGameModeSurvival) || modeHasLives(SourceGameModeSurvival) { t.Fatal("survival exposed source-absent options") }
}

func TestCustomModeCheckboxHitsMatchSymbol1273(t *testing.T) {
	g := &Game{customMode: SourceGameModeNormal}
	if got := g.customModeOptionsHitAt(-1125, 405); got != menuHitModeCrates { t.Fatalf("crate hit=%d", got) }
	if got := g.customModeOptionsHitAt(-1125, 452); got != menuHitModePickups { t.Fatalf("pickup hit=%d", got) }
	if got := g.customModeOptionsHitAt(-1300, 405); got != menuHitModeLives { t.Fatalf("lives hit=%d", got) }
}

func TestCustomContinuePlaysSourceLivesWarning(t *testing.T) {
	g := &Game{customMode: SourceGameModeNormal, customLives: 0, customPage: customPageGame}
	g.activateCustomMenuHit(menuHitGameContinue)
	if g.customPage != customPageGame || g.customWarning != 1 || g.customWarningFrame != 2 {
		t.Fatalf("warning1 source flow page=%d kind=%d frame=%d", g.customPage, g.customWarning, g.customWarningFrame)
	}
}
