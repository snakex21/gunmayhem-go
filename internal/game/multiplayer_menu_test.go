package game

import "testing"

// ONLINE is a main-menu tile now (it used to hide behind Custom Game ->
// ONLINE); VERSUS opens the local Custom Game.
func TestMainMenuHasOnlineAndVersusTiles(t *testing.T) {
	g := newMainMenuGame()
	for _, id := range []string{"main.versus", "main.online"} {
		if _, ok := g.mainMenuUI.item(id); !ok {
			t.Fatalf("main menu has no %s tile", id)
		}
	}
	mainMenuClick(g, mainTileVersusRect.X+60, mainTileVersusRect.Y+60)
	if !g.fadeActive || g.fadeTarget != screenCustomGame || g.customOnlineMode {
		t.Fatalf("VERSUS: fade=%v target=%d online=%v", g.fadeActive, g.fadeTarget, g.customOnlineMode)
	}
}

func TestPublicServerFullState(t *testing.T) {
	if !(multiplayerServerInfo{Players: 4, MaxPlayers: 4}).Full() {
		t.Fatal("4/4 server should be full")
	}
	if (multiplayerServerInfo{Players: 3, MaxPlayers: 4}).Full() {
		t.Fatal("3/4 server should not be full")
	}
}
