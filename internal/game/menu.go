package game

import (
	"fmt"
	"image/color"
	"math/rand"

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
	screenSeriesSummary
	screenGameplay
	screenMultiplayer
	screenMultiplayerLobby
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
	if g.allowCheats && inpututil.IsKeyJustPressed(ebiten.KeyF1) {
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
	case screenSeriesSummary:
		g.updateSeriesSummaryMenu()
	case screenOptions:
		g.updateOptionsMenu()
	case screenGunLibrary:
		g.updateGunLibraryMenu()
	case screenCredits:
		g.updateCreditsMenu()
	case screenMultiplayer:
		g.updateMultiplayerMenu()
	case screenMultiplayerLobby:
		// Retired UI compatibility: never show the standalone lobby again.
		g.openOnlineCustomSlots("SESSION ACTIVE - WAITING FOR PLAYER / ADD AI")
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
	menuHitMainMultiplayer
	menuHitMainUpdate
	menuHitGameBack
	menuHitGameContinue
	menuHitMapBack
	menuHitMapContinue
	menuHitPlayerBack
	menuHitPlayerStart
	menuHitModeBase = 100
	menuHitMapBase  = 200
)

// Exact frame-1 visual bounds of Symbol 1048 from the source XFL (the Gun
// Library BACK button still uses the Flash hit shape).
var wideButtonLocal = Rect{X: 0, Y: -0.01953125, W: 639.9993896484375, H: 60.05859375}

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

func (g *Game) updateSecondaryMenu() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.menuPressed = menuHitNone
		g.beginFade(screenMainMenu, fadePurposeScreen)
		return
	}
	mx, my := g.cursorPos()
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
	if g.onlineMatchSetup && g.netplay != nil && g.netplay.mode == netplayHost {
		g.updateOnlineHostPostGame()
		return
	}

	mx, my := g.cursorPos()
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
		if g.customSeriesRound > 0 {
			if !g.customSeriesIsFinalRound() {
				g.startNextCustomSeriesRound()
				return
			}
			g.finishCustomSeriesToSummary()
			return
		}
		g.customPage = customPageGame
		g.customMenuX = 1800
		g.beginFade(screenCustomGame, fadePurposeScreen)
	}
}

func (g *Game) updateCustomGameMenu() {
	// The online browser, host setup and online wizard keep their own screens
	// until the online phase of the Mayhem UI migration; local Custom Game is
	// the kit screen in custom_local_menu.go.
	if g.customOnlineMode || g.customNetworkOpen {
		g.updateCustomGameV2()
		return
	}
	g.updateCustomLocalMenu()
}

func (g *Game) startCustomGame() {
	if !g.customSeriesStartingNext {
		g.beginCustomSeries()
	}
	if g.onlineMatchSetup {
		if g.netplay == nil || g.netplay.mode != netplayHost {
			g.multiplayerMessage = "ONLY THE HOST CAN START"
			return
		}
		g.syncOnlineHostState()
		if g.onlineActiveParticipantCount() < 2 {
			g.multiplayerMessage = "ADD AN ONLINE PLAYER OR AI BOT"
			g.playCustomWarning(2)
			return
		}
	}

	// The host is authoritative in online Custom Game, so modifier physics and
	// weapon rules are applied there as well. Clients render the host snapshot
	// instead of simulating a second copy of the match.
	g.prepareCustomMatchModifiers()
	g.resetChaosRuntime()

	mapSelection := g.customSeriesMapForRound(g.customSeriesRound)
	// Preserve the older online rotation setting only for sessions that have
	// never configured the new playlist. Once PLAYLIST is used, its per-round
	// map choices are authoritative both offline and online.
	if g.onlineMatchSetup && !g.customSeriesPlaylistConfigured {
		switch g.onlineMapRotation {
		case onlineMapRandomGM1:
			mapSelection = 0
		case onlineMapRandomGM2:
			mapSelection = customGM2RandomNew
		case onlineMapRandomAll:
			pick := rand.Intn(21)
			if pick < 12 {
				mapSelection = pick + 1
			} else {
				mapSelection = gm2MapID(pick - 11)
			}
		}
	}
	g.customMap = mapSelection
	g.customMapSetGM2 = isGM2MapID(mapSelection) || mapSelection == customGM2RandomNew || mapSelection == customGM2RandomAll
	mapNumber := resolveCustomSeriesMapValue(mapSelection)
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

	// Source btn_start refuses to launch unless at least two participant slots
	// are active. Online counts connected humans plus AI bots; waiting ONLINE
	// slots do not count until somebody actually joins them.
	if !g.onlineMatchSetup && g.activeCustomPlayerCount() < 2 {
		g.playCustomWarning(2)
		return
	}

	players := make([]*Player, 0, 4)
	for slot, cfg := range g.customPlayers {
		if cfg.Type == 0 {
			continue
		}
		if g.onlineMatchSetup && slot > 0 && cfg.Type == 1 && !g.onlinePlayerSlots[slot] {
			// ONLINE slot is open for join-in-progress, but no human occupies it yet.
			continue
		}
		id := slot + 1 // preserve PLAYERNUMBER/control slot exactly
		p := NewPlayer(id, arena)
		p.Controls = g.controlConfigs[id-1]
		p.Name = cfg.Name
		p.PlayerColor = cfg.Color
		p.ShirtNumber = cfg.Shirt
		p.HatNumber = cfg.Hat
		p.EyeNumber = cfg.Eyes
		p.ThrowableType = normalizedThrowableType(cfg.Throwable)
		cfg = migrateLegacyGearPerk(cfg)
		if g.onlineMatchSetup {
			p.DefaultWeapon = g.onlineStartingWeapon(cfg.Gun)
			p.PerkNumber = g.onlineAllowedPerk(cfg.Perk)
			p.AbilityNumber = g.onlineAllowedAbility(cfg.Ability, abilityNone)
			p.AI = cfg.Type == 2
			p.RemoteControlled = cfg.Type == 1 && id != 1
		} else {
			p.DefaultWeapon = cfg.Gun
			p.PerkNumber = cfg.Perk
			p.AbilityNumber = resolveConfigAbility(cfg.Ability, nil)
			p.AI = cfg.Type == 2
		}
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
	// Developed Custom Game uses the complete GM2 crate pool (10..86) when
	// the GM2 runtime assets are actually available. Fresh clones that do not
	// yet contain assets/gm2 safely retain the GM1 pool instead of spawning an
	// invisible appended weapon. GM1 campaign always resets to its legacy pool.
	if g.onlineMatchSetup {
		g.crateWeapons = g.onlineCrateWeaponPool()
	} else {
		g.crateWeapons = developedCrateWeapons()
	}
	g.applyCustomModifierSetup()
	g.GameMode = mode
	g.TotalLives = lives
	g.TeamGame = mode == SourceGameModeTeams
	if g.onlineMatchSetup {
		g.PowerON = g.onlinePowerupsEnabled
	}
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
		g.drawMainMenu(screen)
	case screenCustomGame:
		g.drawCustomGameMenu(screen)
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
		// Mayhem UI Options (options_menu.go) replaces the flattened Flash
		// OptionsMenu raster and its patched-over controls.
		g.drawOptionsMenu(screen)
	case screenCredits:
		if g.assets.CreditsMenu != nil {
			drawSourceRaster(screen, g.assets.CreditsMenu, 0, 0, 1, 1, 1)
		} else {
			g.drawSecondaryMenuFallback(screen, "CREDITS")
		}
	case screenMultiplayer:
		g.drawMultiplayerMenu(screen)
	case screenMultiplayerLobby:
		g.drawCustomGameMenu(screen)
	case screenSeriesSummary:
		g.drawSeriesSummary(screen)
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
	mx, my := g.cursorPos()
	var st *uiState
	switch g.screen {
	case screenMainMenu:
		st = &g.mainMenuUI
	case screenOptions:
		st = &g.optionsUI
	case screenCustomGame:
		if !g.customOnlineMode && !g.customNetworkOpen {
			st = &g.customUI
		}
	}
	hot := ""
	if st != nil {
		for _, it := range st.items {
			drawMenuDebugRect(screen, it.rect)
		}
		hot = st.hot
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("MENU DEBUG  mouse=%d,%d  hot=%s  pressed=%d  page=%d", mx, my, hot, g.menuPressed, g.customPage), 8, 8)
}
