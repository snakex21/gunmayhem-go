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
	_ // was screenMultiplayer (legacy menu, removed); keeps wire values stable
	screenMultiplayerLobby
	screenZombieSetup   // zombie_menu.go
	screenZombieResults // zombie_menu.go
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
	case screenMultiplayerLobby:
		// Retired UI compatibility: never show the standalone lobby again.
		g.openOnlineCustomSlots("SESSION ACTIVE - WAITING FOR PLAYER / ADD AI")
	case screenZombieSetup:
		g.updateZombieSetupMenu()
	case screenZombieResults:
		g.updateZombieResultsMenu()
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

func (g *Game) updateCustomGameMenu() {
	// Online (server browser, host setup, lobby) and local Custom Game are
	// Mayhem UI kit screens: online_browser_menu.go / online_lobby_menu.go
	// and custom_local_menu.go. Guests run the lobby from updateNetClient.
	if g.customOnlineMode || g.customNetworkOpen {
		g.updateOnlineCustomGame()
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
		g.drawGunLibraryMenu(screen)
	case screenOptions:
		// Mayhem UI Options (options_menu.go) replaces the flattened Flash
		// OptionsMenu raster and its patched-over controls.
		g.drawOptionsMenu(screen)
	case screenCredits:
		g.drawCreditsMenu(screen)
	case screenMultiplayerLobby:
		g.drawCustomGameMenu(screen)
	case screenSeriesSummary:
		g.drawResultsMenu(screen)
	case screenZombieSetup:
		g.drawZombieSetupMenu(screen)
	case screenZombieResults:
		g.drawZombieResultsMenu(screen)
	case screenPostGame:
		g.drawResultsMenu(screen)
	}
	if g.menuDebug {
		g.drawMenuDebug(screen)
	}
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

// activeKitState is the Mayhem UI state of the screen currently shown (the
// F1 debug overlay outlines its registry), or nil.
func (g *Game) activeKitState() *uiState {
	switch g.screen {
	case screenMainMenu:
		return &g.mainMenuUI
	case screenOptions:
		return &g.optionsUI
	case screenCustomGame, screenMultiplayerLobby:
		if (g.customOnlineMode || g.customNetworkOpen) && g.onlineScreenKind() != onlineScreenLobby {
			return &g.onlineUI
		}
		return &g.customUI
	case screenCampaign:
		return &g.campaignUI
	case screenGunLibrary:
		return &g.libraryUI
	case screenCredits:
		return &g.creditsUI
	case screenPostGame, screenSeriesSummary:
		return &g.resultsUI
	case screenZombieSetup, screenZombieResults:
		return &g.zombieUI
	case screenGameplay:
		if g.paused || g.guestMenuOpen {
			return &g.pauseUI
		}
	}
	return nil
}

func (g *Game) drawMenuDebug(screen *ebiten.Image) {
	mx, my := g.cursorPos()
	st := g.activeKitState()
	hot := ""
	if st != nil {
		for _, it := range st.items {
			drawMenuDebugRect(screen, it.rect)
		}
		hot = st.hot
	}
	focus := ""
	if st != nil {
		focus = st.focus
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("MENU DEBUG  mouse=%d,%d  hot=%s  focus=%s  screen=%d", mx, my, hot, focus, g.screen), 8, 8)
}
