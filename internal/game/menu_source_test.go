package game

import "testing"

func TestCustomGameModeButtonsMatchSymbol1309(t *testing.T) {
	g := &Game{customMode: SourceGameModeNormal, screen: screenCustomGame, customPage: customPageGame}
	tests := []struct {
		y    float64
		want int
	}{
		{170, SourceGameModeNormal},
		{235, SourceGameModeTeams},
		{300, SourceGameModeSurvival},
		{365, SourceGameModeGunGame},
		{430, SourceGameModeInstagib},
	}
	for _, tc := range tests {
		hit := customMenuHitAt(customPageGame, -1700, tc.y)
		g.activateCustomMenuHit(hit)
		if g.customMode != tc.want {
			t.Fatalf("mode row y=%v selected %d want %d", tc.y, g.customMode, tc.want)
		}
	}
}

func TestCustomMapButtonsReverseSourceMapOrder(t *testing.T) {
	g := &Game{}
	g.activateCustomMenuHit(customMenuHitAt(customPageMaps, -780, 105))
	if g.customMap != 0 {
		t.Fatalf("RANDOM selected map %d want 0", g.customMap)
	}
	g.activateCustomMenuHit(customMenuHitAt(customPageMaps, -780, 145))
	if g.customMap != 12 {
		t.Fatalf("mapbtn1 selected map %d want 12", g.customMap)
	}
	g.activateCustomMenuHit(customMenuHitAt(customPageMaps, -780, 465))
	if g.customMap != 1 {
		t.Fatalf("mapbtn12 selected map %d want 1", g.customMap)
	}
}

func TestCustomPageSourceXPositions(t *testing.T) {
	if customPageTargetX(customPageGame) != 1800 || customPageTargetX(customPageMaps) != 900 || customPageTargetX(customPagePlayers) != 0 {
		t.Fatal("Symbol1309 source page positions changed")
	}
}
