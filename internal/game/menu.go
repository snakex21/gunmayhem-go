package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"os/exec"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type gameScreen int

const (
	screenMainMenu gameScreen = iota
	screenCustomGame
	screenCampaign
	screenGunLibrary
	screenOptions
	screenCredits
	screenPostGame
	screenGameplay
)

// Symbol1309 is one 2700px-wide menu strip. The source script moves the whole
// clip between x=1800 (GAME SETUP), x=900 (MAP SELECTION) and x=0
// (PLAYER SETUP), easing by one third every frame.
const (
	customPagePlayers = 1
	customPageMaps    = 2
	customPageGame    = 3
)

func customPageTargetX(page int) float64 {
	switch page {
	case customPagePlayers:
		return 0
	case customPageMaps:
		return 900
	default:
		return 1800
	}
}

func (g *Game) updateMenu() error {
	if g.fadeActive {
		g.updateFade()
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.menuDebug = !g.menuDebug
	}
	switch g.screen {
	case screenMainMenu:
		g.updateMainMenu()
	case screenCustomGame:
		g.updateCustomGameMenu()
	case screenCampaign:
		g.updateCampaignMenu()
	case screenPostGame:
		g.updatePostGameMenu()
	case screenOptions:
		g.updateOptionsMenu()
	case screenGunLibrary:
		g.updateGunLibraryMenu()
	case screenCredits:
		g.updateCreditsMenu()
	}
	return nil
}

const (
	menuHitNone = iota
	menuHitMainCampaign
	menuHitMainCustom
	menuHitMainLibrary
	menuHitMainOptions
	menuHitMainCredits
	menuHitMainMore
	menuHitGameBack
	menuHitGameContinue
	menuHitMapBack
	menuHitMapContinue
	menuHitPlayerBack
	menuHitPlayerStart
	menuHitModeBase = 100
	menuHitMapBase  = 200
)

var (
	// Exact frame-1 visual bounds from the source XFL. Mouse hit-testing in
	// Flash uses these movie-clip shapes after the instance matrix is applied.
	mainButtonLocal = Rect{X: 0, Y: 0, W: 300, H: 45}                                  // Symbol 8
	modeButtonLocal = Rect{X: 0, Y: 0, W: 200, H: 20}                                  // Symbol 1282
	mapButtonLocal  = Rect{X: 0, Y: -0.0150390625, W: 200, H: 20.0150390625}           // Symbol 1200
	wideButtonLocal = Rect{X: 0, Y: -0.01953125, W: 639.9993896484375, H: 60.05859375} // Symbol 1048
)

func sourceMenuHitRect(local Rect, tx, ty, sx, sy float64) Rect {
	return transformedLocalRect(local, tx, ty, sx, sy)
}

func (g *Game) resolveMenuRelease(hit int) (int, bool) {
	// The Flash menus use onRelease, not onPress. Remember which source button
	// received the mouse-down and trigger only if the mouse is released over
	// that same clip. This prevents neighbouring/animating buttons from firing.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.menuPressed = hit
		return menuHitNone, false
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		pressed := g.menuPressed
		g.menuPressed = menuHitNone
		if pressed != menuHitNone && pressed == hit {
			return hit, true
		}
	}
	return menuHitNone, false
}

func mainMenuHitAt(x, y float64) int {
	placements := [...]struct {
		hit    int
		tx, ty float64
	}{
		{menuHitMainCampaign, 572.05, 253.05},
		{menuHitMainCustom, 572.05, 298.85},
		{menuHitMainLibrary, 572.05, 365.75},
		{menuHitMainOptions, 572.05, 433.75},
		{menuHitMainCredits, 572.05, 479.55},
		{menuHitMainMore, 572.05, 525.40},
	}
	for _, p := range placements {
		if sourceMenuHitRect(mainButtonLocal, p.tx, p.ty, 1, 1).Contains(x, y) {
			return p.hit
		}
	}
	return menuHitNone
}

func (g *Game) updateMainMenu() {
	mx, my := ebiten.CursorPosition()
	hit := mainMenuHitAt(float64(mx), float64(my))
	activated, ok := g.resolveMenuRelease(hit)
	if !ok {
		return
	}
	// DefineSprite_28_mainmenu: btn1..btn5 play menu.wav; the Armor Games
	// external-link button (btn6) deliberately does not.
	if activated != menuHitMainMore {
		g.playSourceSFX("menu.wav", false)
	}
	switch activated {
	case menuHitMainCampaign:
		g.beginFade(screenCampaign, fadePurposeScreen)
	case menuHitMainCustom:
		g.customPage = customPageGame
		g.customMenuX = 1800
		g.beginFade(screenCustomGame, fadePurposeScreen)
	case menuHitMainLibrary:
		g.beginFade(screenGunLibrary, fadePurposeScreen)
	case menuHitMainOptions:
		g.beginFade(screenOptions, fadePurposeScreen)
	case menuHitMainCredits:
		g.beginFade(screenCredits, fadePurposeScreen)
	case menuHitMainMore:
		// Exact source action: getURL("http://armorgames.com", _blank).
		_ = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", "http://armorgames.com").Start()
	}
}

func (g *Game) updateSecondaryMenu() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.menuPressed = menuHitNone
		g.beginFade(screenMainMenu, fadePurposeScreen)
		return
	}
	mx, my := ebiten.CursorPosition()
	hit := menuHitNone
	switch g.screen {
	case screenCredits:
		// Symbol1046.btn_back = Symbol792 at (716.15,520.15).
		if sourceMenuHitRect(campaignBackLocal, 716.15, 520.15, 1, 1).Contains(float64(mx), float64(my)) {
			hit = menuHitGameBack
		}
	case screenGunLibrary:
		// Symbol1198.btn_back = Symbol1048 at (679.75,20.05), xscale 31.25%.
		if sourceMenuHitRect(wideButtonLocal, 679.75, 20.05, 0.3125, 0.9991607666015625).Contains(float64(mx), float64(my)) {
			hit = menuHitGameBack
		}
	}
	activated, ok := g.resolveMenuRelease(hit)
	if ok && activated == menuHitGameBack {
		// Symbols1046 (Credits) and 1198 (Gun Library) both play menu.wav on Back.
		g.playSourceSFX("menu.wav", false)
		g.beginFade(screenMainMenu, fadePurposeScreen)
	}
}

func (g *Game) updatePostGameMenu() {
	mx, my := ebiten.CursorPosition()
	r := sourceMenuHitRect(campaignBackLocal, 638.15, 532.1, 1.86492919921875, 1.0372772216796875)
	hit := menuHitNone
	if r.Contains(float64(mx), float64(my)) {
		hit = menuHitGameContinue
	}
	activated, ok := g.resolveMenuRelease(hit)
	if ok && activated == menuHitGameContinue {
		// DefineSprite_1019.btn_back.
		g.playSourceSFX("menu.wav", false)
	}
	if (inpututil.IsKeyJustPressed(ebiten.KeyEscape)) || (ok && activated == menuHitGameContinue) {
		g.customPage = customPageGame
		g.customMenuX = 1800
		g.beginFade(screenCustomGame, fadePurposeScreen)
	}
}

func (g *Game) updateCustomGameMenu() {
	g.updateCustomWarning()
	if g.customPage == customPageGame {
		g.updateCustomLivesInput()
	}
	if g.customPage == customPagePlayers {
		g.updateCustomPlayerCards()
		g.updateCustomNameInput()
	}
	target := customPageTargetX(g.customPage)
	g.customMenuX += (target - g.customMenuX) / 3
	if math.Abs(g.customMenuX-target) < 0.5 {
		g.customMenuX = target
	} else if math.Mod(g.customMenuX, 1) != 0 {
		// Source rounds whenever _X is fractional.
		g.customMenuX = math.Round(g.customMenuX)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.menuPressed = menuHitNone
		if g.customPage == customPageGame {
			g.beginFade(screenMainMenu, fadePurposeScreen)
		} else {
			g.customPage++
		}
		return
	}

	mx, my := ebiten.CursorPosition()
	// Symbol1309's local origin is translated by _X. The SourceRaster padding
	// does not enter this calculation: drawSourceRaster preserves that origin.
	lx := float64(mx) - g.customMenuX
	ly := float64(my)
	hit := customMenuHitAt(g.customPage, lx, ly)
	if g.customPage == customPageGame {
		if optionHit := g.customModeOptionsHitAt(lx, ly); optionHit != menuHitNone {
			hit = optionHit
		}
	}
	if g.customPage == customPagePlayers {
		if editorHit := g.customPlayerEditorHitAt(lx, ly); editorHit != menuHitNone {
			hit = editorHit
		} else if playerHit := g.customPlayerSetupHitAt(lx, ly); playerHit != menuHitNone {
			hit = playerHit
		}
	}
	activated, ok := g.resolveMenuRelease(hit)
	if !ok {
		return
	}
	// Symbol1309 only plays menu.wav on the five navigation buttons. Mode/map
	// choices, player setup fields and START itself are silent in the source.
	switch activated {
	case menuHitGameBack, menuHitGameContinue, menuHitMapBack, menuHitMapContinue, menuHitPlayerBack:
		g.playSourceSFX("menu.wav", false)
	}
	g.activateCustomMenuHit(activated)
}

func customMenuHitAt(page int, x, y float64) int {
	switch page {
	case customPageGame:
		rows := [...]struct {
			mode   int
			tx, ty float64
		}{
			{SourceGameModeNormal, -1770, 160.0},
			{SourceGameModeTeams, -1770, 225.7},
			{SourceGameModeSurvival, -1770, 291.5},
			{SourceGameModeGunGame, -1770, 357.25},
			{SourceGameModeInstagib, -1770, 423.1},
		}
		for _, row := range rows {
			if sourceMenuHitRect(modeButtonLocal, row.tx, row.ty, 1.6749725341796875, 2.99078369140625).Contains(x, y) {
				return menuHitModeBase + row.mode
			}
		}
		if sourceMenuHitRect(wideButtonLocal, -1780, 520.05, 0.3125, 0.9991607666015625).Contains(x, y) {
			return menuHitGameBack
		}
		if sourceMenuHitRect(wideButtonLocal, -1560, 520.05, 1, 0.9991607666015625).Contains(x, y) {
			return menuHitGameContinue
		}

	case customPageMaps:
		mapRows := [...]struct {
			mapNumber int
			ty        float64
		}{
			{0, 99.3},
			{12, 138.9}, {11, 167.95}, {10, 197.0}, {9, 226.0},
			{8, 255.05}, {7, 284.1}, {6, 313.15}, {5, 342.2},
			{4, 371.2}, {3, 400.25}, {2, 429.3}, {1, 458.35},
		}
		for _, row := range mapRows {
			if sourceMenuHitRect(mapButtonLocal, -883, row.ty, 1, 1.3199920654296875).Contains(x, y) {
				return menuHitMapBase + row.mapNumber
			}
		}
		if sourceMenuHitRect(wideButtonLocal, -880, 520.05, 0.3125, 0.9991607666015625).Contains(x, y) {
			return menuHitMapBack
		}
		if sourceMenuHitRect(wideButtonLocal, -660, 520.05, 1, 0.9991607666015625).Contains(x, y) {
			return menuHitMapContinue
		}

	case customPagePlayers:
		if sourceMenuHitRect(wideButtonLocal, 20, 520.05, 0.3125, 0.9991607666015625).Contains(x, y) {
			return menuHitPlayerBack
		}
		if sourceMenuHitRect(wideButtonLocal, 240, 520.05, 1, 0.9991607666015625).Contains(x, y) {
			return menuHitPlayerStart
		}
	}
	return menuHitNone
}

func (g *Game) activateCustomMenuHit(hit int) {
	if g.activateCustomModeOption(hit) {
		return
	}
	if g.activateCustomPlayerEditorHit(hit) {
		return
	}
	if g.activateCustomPlayerSlot(hit) {
		return
	}
	switch {
	case hit >= menuHitModeBase+SourceGameModeNormal && hit <= menuHitModeBase+SourceGameModeSurvival:
		g.customMode = hit - menuHitModeBase
	case hit >= menuHitMapBase && hit <= menuHitMapBase+12:
		g.customMap = hit - menuHitMapBase
	case hit == menuHitGameBack:
		g.beginFade(screenMainMenu, fadePurposeScreen)
	case hit == menuHitGameContinue:
		if modeHasLives(g.customMode) && g.customLives <= 0 {
			g.playCustomWarning(1)
			return
		}
		g.customLivesFocus = false
		g.customPage = customPageMaps
	case hit == menuHitMapBack:
		g.customPage = customPageGame
	case hit == menuHitMapContinue:
		g.customPage = customPagePlayers
	case hit == menuHitPlayerBack:
		g.customPage = customPageMaps
	case hit == menuHitPlayerStart:
		g.startCustomGame()
	}
}

func (g *Game) startCustomGame() {
	mapNumber := g.customMap
	if mapNumber == 0 {
		// Source RANDOM chooses one of the twelve Custom Game maps. Geometry is
		// loaded only after the number is chosen, not all twelve at startup.
		mapNumber = rand.Intn(12) + 1
	}
	arena, ok := g.ensureMap(mapNumber)
	if !ok {
		arena, _ = g.ensureMap(1)
	}
	g.arena = arena

	mode := g.customMode
	if mode < SourceGameModeNormal || mode > SourceGameModeSurvival {
		mode = SourceGameModeNormal
	}
	lives := g.customLives
	if lives <= 0 {
		lives = 10
	}
	if mode == SourceGameModeSurvival {
		lives = 10
	}

	// Source btn_start refuses to launch unless at least two ptype slots are
	// active and plays warning2 instead.
	if g.activeCustomPlayerCount() < 2 {
		g.playCustomWarning(2)
		return
	}

	players := make([]*Player, 0, 4)
	for slot, cfg := range g.customPlayers {
		if cfg.Type == 0 {
			continue
		}
		id := slot + 1 // preserve PLAYERNUMBER/control slot exactly
		p := NewPlayer(id, arena)
		p.Controls = g.controlConfigs[id-1]
		p.Name = cfg.Name
		p.PlayerColor = cfg.Color
		p.ShirtNumber = cfg.Shirt
		p.HatNumber = cfg.Hat
		p.DefaultWeapon = cfg.Gun
		p.PerkNumber = cfg.Perk
		p.AI = cfg.Type == 2
		if mode == SourceGameModeTeams {
			p.Team = cfg.Team
		} else {
			p.Team = id
		}
		if mode == SourceGameModeSurvival {
			p.Team = 0
		}
		p.configureSourceGameMode(mode, lives, arena)
		players = append(players, p)
	}

	g.players = players
	g.GameMode = mode
	g.TotalLives = lives
	g.TeamGame = mode == SourceGameModeTeams
	if mode == SourceGameModeSurvival {
		// frame10 source forces Survival to crates on / pickups off.
		g.CrateON = true
		g.PowerON = false
	}
	// Other custom modes keep the runtime checkbox values from Symbol1273;
	// Gun Game/1 Hit 1 Kill suppress irrelevant pickup types in their own
	// source spawn clips rather than rewriting these root booleans.
	g.GameWin = false
	g.teamGameWin = false
	g.matchWinCountdown = 0
	g.soloWinFrame = 0
	g.winnerPlayerID = 0
	g.campaignMode = false
	g.resetArenaState()
	g.beginFade(screenGameplay, fadePurposeScreen)
}

func (g *Game) drawMenu(screen *ebiten.Image) {
	g.assets.EnsureScene(g.arena.Number)
	// Source menu remains on top of the current arena scene. Keep the map art
	// behind the menu so the transition can later gain the original bot demo.
	drawSourceRaster(screen, g.assets.SceneBack[g.arena.Number], 0, 0, 1, 1, 1)
	drawSourceRaster(screen, g.assets.SceneMid[g.arena.Number], 0, 0, 1, 1, 1)
	drawSourceRaster(screen, g.assets.SceneFront[g.arena.Number], 51.1, 192.6, 1, 1, 1)

	switch g.screen {
	case screenMainMenu:
		drawSourceRaster(screen, g.assets.MainMenu, 0, 0, 1, 1, 1)
	case screenCustomGame:
		drawSourceRaster(screen, g.assets.CustomMenu, g.customMenuX, 0, 1, 1, 1)
		g.drawCustomGameInteractions(screen)
	case screenCampaign:
		g.drawCampaignMenu(screen)
	case screenGunLibrary:
		g.assets.EnsureGunLibrary()
		if g.assets.GunLibraryMenu != nil {
			drawSourceRaster(screen, g.assets.GunLibraryMenu, 0, 0, 1, 1, 1)
			g.drawGunLibraryInteractions(screen)
		} else {
			g.drawSecondaryMenuFallback(screen, "GUN LIBRARY")
		}
	case screenOptions:
		if g.assets.OptionsMenu != nil {
			drawSourceRaster(screen, g.assets.OptionsMenu, 0, 0, 1, 1, 1)
			g.drawOptionsInteractions(screen)
		} else {
			g.drawSecondaryMenuFallback(screen, "OPTIONS")
		}
	case screenCredits:
		if g.assets.CreditsMenu != nil {
			drawSourceRaster(screen, g.assets.CreditsMenu, 0, 0, 1, 1, 1)
		} else {
			g.drawSecondaryMenuFallback(screen, "CREDITS")
		}
	case screenPostGame:
		if !g.drawPostGameScreen(screen) {
			// Flattened Symbol1019 is fallback-only because it contains baked
			// Cool Dude / 89 authoring placeholders in all four player columns.
			if g.assets.PostGameMenu != nil {
				drawSourceRaster(screen, g.assets.PostGameMenu, 0, 0, 1, 1, 1)
			} else {
				g.drawSecondaryMenuFallback(screen, "RESULTS")
			}
		}
	}
	if g.menuDebug {
		g.drawMenuDebug(screen)
	}
}

func (g *Game) drawCustomMenuChosen(screen *ebiten.Image) {
	if g.assets.MenuChosen == nil {
		return
	}
	switch g.customPage {
	case customPageMaps:
		rows := map[int]float64{
			0:  99.3,
			12: 138.9, 11: 167.95, 10: 197.0, 9: 226.0,
			8: 255.05, 7: 284.1, 6: 313.15, 5: 342.2,
			4: 371.2, 3: 400.25, 2: 429.3, 1: 458.35,
		}
		if y, ok := rows[g.customMap]; ok {
			drawSourceRaster(screen, g.assets.MenuChosen, g.customMenuX-883, y, 1, 1.3199920654296875, 1)
		}
	}
}

func (g *Game) drawSecondaryMenuFallback(screen *ebiten.Image, title string) {
	// Exact source screen names; used only if the flattened source raster for
	// that menu was not exported by FFDec. ESC or the bottom BACK area returns
	// to the main menu, matching each source screen's btn_back action.
	drawSourceHUDText(screen, title, 50.75, color.NRGBA{A: 255}, 28, 24)
	drawSourceHUDText(screen, "BACK", 53, color.NRGBA{A: 255}, 28, 520)
}

func drawMenuDebugRect(dst *ebiten.Image, r Rect) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	c := color.RGBA{R: 255, G: 0, B: 0, A: 70}
	ebitenutil.DrawRect(dst, r.X, r.Y, r.W, 1, c)
	ebitenutil.DrawRect(dst, r.X, r.Y+r.H-1, r.W, 1, c)
	ebitenutil.DrawRect(dst, r.X, r.Y, 1, r.H, c)
	ebitenutil.DrawRect(dst, r.X+r.W-1, r.Y, 1, r.H, c)
}

func (g *Game) drawMenuDebug(screen *ebiten.Image) {
	mx, my := ebiten.CursorPosition()
	hit := menuHitNone
	if g.screen == screenMainMenu {
		placements := [...]struct {
			tx, ty float64
		}{
			{572.05, 253.05}, {572.05, 298.85}, {572.05, 365.75},
			{572.05, 433.75}, {572.05, 479.55}, {572.05, 525.40},
		}
		for _, p := range placements {
			drawMenuDebugRect(screen, sourceMenuHitRect(mainButtonLocal, p.tx, p.ty, 1, 1))
		}
		hit = mainMenuHitAt(float64(mx), float64(my))
	} else if g.screen == screenCustomGame {
		lx := float64(mx) - g.customMenuX
		ly := float64(my)
		hit = customMenuHitAt(g.customPage, lx, ly)
		for _, r := range customMenuDebugRects(g.customPage) {
			r.X += g.customMenuX
			drawMenuDebugRect(screen, r)
		}
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("MENU DEBUG  mouse=%d,%d  hit=%d  pressed=%d  page=%d  menuX=%.0f", mx, my, hit, g.menuPressed, g.customPage, g.customMenuX), 8, 8)
}

func customMenuDebugRects(page int) []Rect {
	var out []Rect
	switch page {
	case customPageGame:
		for _, p := range [][2]float64{{-1770, 160.0}, {-1770, 225.7}, {-1770, 291.5}, {-1770, 357.25}, {-1770, 423.1}} {
			out = append(out, sourceMenuHitRect(modeButtonLocal, p[0], p[1], 1.6749725341796875, 2.99078369140625))
		}
		out = append(out,
			sourceMenuHitRect(wideButtonLocal, -1780, 520.05, 0.3125, 0.9991607666015625),
			sourceMenuHitRect(wideButtonLocal, -1560, 520.05, 1, 0.9991607666015625),
		)
	case customPageMaps:
		for _, y := range []float64{99.3, 138.9, 167.95, 197.0, 226.0, 255.05, 284.1, 313.15, 342.2, 371.2, 400.25, 429.3, 458.35} {
			out = append(out, sourceMenuHitRect(mapButtonLocal, -883, y, 1, 1.3199920654296875))
		}
		out = append(out,
			sourceMenuHitRect(wideButtonLocal, -880, 520.05, 0.3125, 0.9991607666015625),
			sourceMenuHitRect(wideButtonLocal, -660, 520.05, 1, 0.9991607666015625),
		)
	case customPagePlayers:
		out = append(out,
			sourceMenuHitRect(wideButtonLocal, 20, 520.05, 0.3125, 0.9991607666015625),
			sourceMenuHitRect(wideButtonLocal, 240, 520.05, 1, 0.9991607666015625),
		)
	}
	return out
}
