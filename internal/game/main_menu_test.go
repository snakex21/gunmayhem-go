package game

import "testing"

func newMainMenuGame() *Game {
	g := New()
	g.screen = screenMainMenu
	g.runMainMenuUpdate(uiInput{})
	return g
}

func mainMenuClick(g *Game, x, y float64) {
	g.runMainMenuUpdate(uiInput{X: x, Y: y, Moved: true})
	g.runMainMenuUpdate(uiInput{X: x, Y: y, Pressed: true, Down: true})
	g.runMainMenuUpdate(uiInput{X: x, Y: y, Released: true})
	// One idle frame so the registry reflects what the click opened.
	g.runMainMenuUpdate(uiInput{X: x, Y: y})
}

func mainMenuKey(g *Game, in uiInput) {
	g.runMainMenuUpdate(in)
	g.runMainMenuUpdate(uiInput{})
}

func TestMainMenuTilesOpenTheirScreens(t *testing.T) {
	cases := []struct {
		name   string
		r      Rect
		target gameScreen
		check  func(g *Game) bool
	}{
		{"versus", mainTileVersusRect, screenCustomGame, func(g *Game) bool {
			return !g.customOnlineMode && !g.customNetworkOpen && g.customPage == customPageGame
		}},
		{"campaign", mainTileCampaignRect, screenCampaign, nil},
		{"online", mainTileOnlineRect, screenCustomGame, func(g *Game) bool {
			return g.customOnlineMode && g.customNetworkOpen && !g.customHostSetupOpen && g.netplay == nil
		}},
		{"library", mainTileLibraryRect, screenGunLibrary, nil},
		{"options", mainOptionsRect, screenOptions, nil},
		{"credits", mainCreditsRect, screenCredits, nil},
	}
	for _, tc := range cases {
		g := newMainMenuGame()
		g.customOnlineMode = true // VERSUS must reset a previous online visit
		mainMenuClick(g, tc.r.X+tc.r.W/2, tc.r.Y+tc.r.H/2)
		if !g.fadeActive || g.fadeTarget != tc.target {
			t.Fatalf("%s: fade=%v target=%d want %d", tc.name, g.fadeActive, g.fadeTarget, tc.target)
		}
		if tc.check != nil && !tc.check(g) {
			t.Fatalf("%s: wrong custom-game state online=%v network=%v page=%d", tc.name, g.customOnlineMode, g.customNetworkOpen, g.customPage)
		}
		_ = g.CloseNetplay()
	}
}

func TestMainMenuZombieTileOpensSetup(t *testing.T) {
	g := newMainMenuGame()
	it, ok := g.mainMenuUI.item("main.zombie")
	if !ok || it.disabled || !it.focusable {
		t.Fatalf("zombie tile missing or disabled: %+v ok=%v", it, ok)
	}
	g.onlineZombie = true
	mainMenuClick(g, mainTileZombieRect.X+40, mainTileZombieRect.Y+40)
	if !g.fadeActive || g.fadeTarget != screenZombieSetup {
		t.Fatalf("ZOMBIE tile: fade=%v target=%d", g.fadeActive, g.fadeTarget)
	}
	if g.onlineZombie || g.zombieSetup.Slots[0] != 1 {
		t.Fatalf("setup not reset: online=%v slots=%v", g.onlineZombie, g.zombieSetup.Slots)
	}
}

func TestMainMenuKeyboardNavigation(t *testing.T) {
	g := newMainMenuGame()
	mainMenuKey(g, uiInput{DownKey: true})
	if g.mainMenuUI.focus != "main.versus" {
		t.Fatalf("first focus=%q want main.versus", g.mainMenuUI.focus)
	}
	mainMenuKey(g, uiInput{Right: true})
	if g.mainMenuUI.focus != "main.campaign" {
		t.Fatalf("RIGHT from versus=%q", g.mainMenuUI.focus)
	}
	mainMenuKey(g, uiInput{Right: true})
	if g.mainMenuUI.focus != "main.online" {
		t.Fatalf("RIGHT from campaign=%q", g.mainMenuUI.focus)
	}
	mainMenuKey(g, uiInput{Left: true})
	mainMenuKey(g, uiInput{DownKey: true})
	if g.mainMenuUI.focus != "main.library" {
		t.Fatalf("DOWN from campaign=%q", g.mainMenuUI.focus)
	}
	// The ZOMBIE tile is right of GUN LIBRARY and reachable by keyboard.
	mainMenuKey(g, uiInput{Right: true})
	if g.mainMenuUI.focus != "main.zombie" {
		t.Fatalf("RIGHT from library=%q want main.zombie", g.mainMenuUI.focus)
	}
	for g.mainMenuUI.focus != "main.library" {
		mainMenuKey(g, uiInput{Tab: true})
	}
	mainMenuKey(g, uiInput{Enter: true})
	if !g.fadeActive || g.fadeTarget != screenGunLibrary {
		t.Fatal("ENTER on GUN LIBRARY did not open it")
	}
}

func TestMainMenuQuitConfirmation(t *testing.T) {
	g := newMainMenuGame()
	mainMenuKey(g, uiInput{Esc: true})
	if !g.mainMenuQuitOpen {
		t.Fatal("ESC did not ask to quit")
	}
	// The modal blocks the tiles.
	mainMenuClick(g, mainTileVersusRect.X+30, mainTileVersusRect.Y+30)
	if g.fadeActive {
		t.Fatal("tile clicked through the quit modal")
	}
	mainMenuKey(g, uiInput{Esc: true})
	if g.mainMenuQuitOpen || g.quitRequested {
		t.Fatal("ESC must cancel the quit modal")
	}
	mainMenuClick(g, mainQuitRect.X+20, mainQuitRect.Y+10)
	if !g.mainMenuQuitOpen {
		t.Fatal("QUIT did not open the confirmation")
	}
	yes, ok := g.mainMenuUI.item("main.quit.yes")
	if !ok {
		t.Fatal("no QUIT button in the modal")
	}
	mainMenuClick(g, yes.rect.X+10, yes.rect.Y+10)
	if !g.quitRequested {
		t.Fatal("QUIT in the modal did not request exit")
	}
}

func TestMainMenuUpdateNotice(t *testing.T) {
	g := newMainMenuGame()
	if _, ok := g.mainMenuUI.item("main.update"); ok {
		t.Fatal("update notice shown without an update")
	}
	g.updateRuntime.phase = updatePhaseAvailable
	g.updateRuntime.release.Tag = "v9.9.9"
	g.runMainMenuUpdate(uiInput{})
	it, ok := g.mainMenuUI.item("main.update")
	if !ok || it.disabled || it.rect != mainUpdateRect {
		t.Fatalf("update notice: %+v ok=%v", it, ok)
	}
	g.updateRuntime.phase = updatePhaseDownloading
	g.runMainMenuUpdate(uiInput{})
	if it, _ := g.mainMenuUI.item("main.update"); !it.disabled {
		t.Fatal("update notice must be inert while downloading")
	}
}

func TestMainMenuLayoutFitsTheGrid(t *testing.T) {
	g := newMainMenuGame()
	for _, it := range g.mainMenuUI.items {
		r := it.rect
		if r.X < uiMargin-0.5 || r.X+r.W > ScreenWidth-uiMargin+0.5 || r.Y < 0 || r.Y+r.H > uiActionBottom+0.5 {
			t.Fatalf("%s outside the layout grid: %+v", it.id, r)
		}
		for _, other := range g.mainMenuUI.items {
			if other.id != it.id && rectsOverlap(other.rect, r) {
				t.Fatalf("%s overlaps %s", it.id, other.id)
			}
		}
	}
}
