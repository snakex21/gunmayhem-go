package game

import "testing"

// The Flash-strip mode and map selectors (Symbol1309) were replaced by the
// MATCH and MAPS tabs; these pin the same choices on the new screen.
func TestCustomGameModeButtonsSelectSourceModes(t *testing.T) {
	g := newCustomLocalGame(t)
	want := []int{SourceGameModeNormal, SourceGameModeTeams, SourceGameModeSurvival, SourceGameModeGunGame, SourceGameModeInstagib}
	for i, mode := range want {
		r := customLocalModeRect(i)
		customLocalClick(g, r.X+r.W/2, r.Y+r.H/2)
		if g.customMode != mode {
			t.Fatalf("mode row %d selected %d want %d", i, g.customMode, mode)
		}
	}
}

func TestCustomMapTilesUseSourceMapNumbers(t *testing.T) {
	g := newCustomLocalGame(t)
	g.setCustomV2Page(customPageMaps)
	g.runCustomLocalUpdate(uiInput{})
	for tile, want := range map[int]int{0: 0, 1: 1, 12: 12} {
		r := customLocalMapTileRect(tile)
		customLocalClick(g, r.X+r.W/2, r.Y+r.H/2)
		if got := g.customSeriesMapForRound(1); got != want {
			t.Fatalf("tile %d selected map %d want %d", tile, got, want)
		}
	}
	if g.customPlaylistGM2 {
		t.Fatal("GM1 map tile switched to GM2")
	}
}

func TestCustomPageSourceXPositions(t *testing.T) {
	if customPageTargetX(customPageGame) != 1800 || customPageTargetX(customPageMaps) != 900 || customPageTargetX(customPagePlayers) != 0 {
		t.Fatal("Symbol1309 source page positions changed")
	}
}
