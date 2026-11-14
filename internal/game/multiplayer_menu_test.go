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

func TestMultiplayerMenuHitboxes(t *testing.T) {
	checks := []struct {
		r    Rect
		want int
	}{
		{multiplayerLocalRect, menuHitMultiplayerLocal},
		{multiplayerHostRect, menuHitMultiplayerHost},
		{multiplayerRefreshRect, menuHitMultiplayerRefresh},
		{multiplayerDirectJoinRect, menuHitMultiplayerDirectJoin},
		{multiplayerBackRect, menuHitMultiplayerBack},
	}
	for _, tc := range checks {
		if got := multiplayerMenuHitAt(tc.r.X+tc.r.W/2, tc.r.Y+tc.r.H/2); got != tc.want {
			t.Errorf("hit=%d want %d for rect %+v", got, tc.want, tc.r)
		}
	}
}

func TestMultiplayerServerRowsUseAbsoluteServerIndex(t *testing.T) {
	g := &Game{
		multiplayerServers: []multiplayerServerInfo{
			{Name: "A", Address: "10.0.0.1:7777"},
			{Name: "B", Address: "10.0.0.2:7777"},
			{Name: "C", Address: "10.0.0.3:7777"},
			{Name: "D", Address: "10.0.0.4:7777"},
			{Name: "E", Address: "10.0.0.5:7777"},
			{Name: "F", Address: "10.0.0.6:7777"},
		},
		multiplayerServerScroll: 1,
	}
	x := multiplayerServerListRect.X + 20
	y := multiplayerServerListRect.Y + multiplayerServerRowH/2
	if got := g.multiplayerServerHitAt(x, y); got != menuHitMultiplayerServerBase+1 {
		t.Fatalf("server row hit=%d want=%d", got, menuHitMultiplayerServerBase+1)
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
