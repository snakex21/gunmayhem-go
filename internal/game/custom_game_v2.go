package game

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	menuHitCustomV2TabSession = 11000 + iota
	menuHitCustomV2TabMatch
	menuHitCustomV2TabPlaylist
	menuHitCustomV2TabPlayers
	menuHitCustomV2Back
	menuHitCustomV2Start
	menuHitCustomV2PlaylistGM1
	menuHitCustomV2PlaylistGM2
	menuHitCustomV2ApplyAll
	menuHitCustomV2RoundBase      = 11100
	menuHitCustomV2MapBase        = 11200
	menuHitCustomV2SlotRemoveBase = 11300
)

const customV2PlaylistVisibleRows = 9

var (
	// LOCAL header tabs start after the title; keeping them separate from the
	// online step rects prevents LOCAL CUSTOM GAME from running underneath step 1.
	customV2TabMatchRect    = Rect{X: 280, Y: 48, W: 120, H: 34}
	customV2TabPlaylistRect = Rect{X: 408, Y: 48, W: 105, H: 34}
	customV2TabPlayersRect  = Rect{X: 521, Y: 48, W: 90, H: 34}
	customV2BackRect        = Rect{X: 20, Y: 528, W: 190, H: 52}
	customV2StartRect       = Rect{X: 660, Y: 528, W: 220, H: 52}

	customV2LivesMinusRect  = Rect{X: 365, Y: 283, W: 44, H: 38}
	customV2LivesPlusRect   = Rect{X: 505, Y: 283, W: 44, H: 38}
	customV2RoundsMinusRect = Rect{X: 615, Y: 283, W: 44, H: 38}
	customV2RoundsPlusRect  = Rect{X: 775, Y: 283, W: 44, H: 38}
	customV2CratesRect      = Rect{X: 325, Y: 345, W: 250, H: 68}
	customV2PickupsRect     = Rect{X: 595, Y: 345, W: 250, H: 68}
	customV2ModifiersRect   = Rect{X: 325, Y: 431, W: 520, H: 58}

	customV2PlaylistListRect = Rect{X: 25, Y: 145, W: 315, H: 350}
	customV2PlaylistGM1Rect  = Rect{X: 380, Y: 157, W: 105, H: 34}
	customV2PlaylistGM2Rect  = Rect{X: 492, Y: 157, W: 105, H: 34}
	customV2ApplyAllRect     = Rect{X: 640, Y: 151, W: 215, H: 40}

	customV2OnlineStepRects = [4]Rect{
		{X: 300, Y: 42, W: 108, H: 34},
		{X: 414, Y: 42, W: 108, H: 34},
		{X: 528, Y: 42, W: 108, H: 34},
		{X: 642, Y: 42, W: 108, H: 34},
	}
)

var customV2ModeRows = [...]struct {
	mode  int
	rect  Rect
	label string
}{
	{SourceGameModeNormal, Rect{X: 35, Y: 150, W: 245, H: 54}, "LAST MAN STANDING"},
	{SourceGameModeTeams, Rect{X: 35, Y: 213, W: 245, H: 54}, "TEAM BATTLE"},
	{SourceGameModeSurvival, Rect{X: 35, Y: 276, W: 245, H: 54}, "DUCK SURVIVAL"},
	{SourceGameModeGunGame, Rect{X: 35, Y: 339, W: 245, H: 54}, "GUN GAME"},
	{SourceGameModeInstagib, Rect{X: 35, Y: 402, W: 245, H: 54}, "1 HIT 1 KILL"},
}

type customV2MapChoice struct {
	value int
	label string
}

func customV2MapChoices(gm2 bool) []customV2MapChoice {
	if gm2 {
		out := []customV2MapChoice{{value: customGM2RandomNew, label: "RANDOM GM2"}}
		for n := 1; n <= 9; n++ {
			out = append(out, customV2MapChoice{value: gm2MapID(n), label: fmt.Sprintf("MAP %d - %s", n, gm2MapDisplayNames[n])})
		}
		return out
	}
	out := []customV2MapChoice{{value: 0, label: "RANDOM GM1"}}
	for n := 1; n <= 12; n++ {
		label := sourceMapDisplayNames[n]
		if label == "" {
			label = fmt.Sprintf("MAP %d", n)
		}
		out = append(out, customV2MapChoice{value: n, label: fmt.Sprintf("MAP %d - %s", n, label)})
	}
	return out
}

func customV2MapChoiceRect(index int) Rect {
	col := index % 2
	row := index / 2
	return Rect{X: 380 + float64(col)*242, Y: 207 + float64(row)*43, W: 225, H: 36}
}

func customV2RoundRect(visibleRow int) Rect {
	return Rect{X: 35, Y: 158 + float64(visibleRow)*37, W: 295, H: 33}
}

func customV2OnlineSlotRemoveRect(slot int) Rect {
	return Rect{X: 20 + float64(slot)*220 + 22, Y: 466, W: 176, H: 28}
}

func (g *Game) customV2OnlineSlotRemoveHitAt(x, y float64) int {
	if g == nil || !g.customOnlineMode || g.customPage != customPagePlayers || g.netplay == nil || g.netplay.mode != netplayHost {
		return menuHitNone
	}
	for slot := 1; slot < len(g.customPlayers); slot++ {
		if g.customPlayers[slot].Type == 0 {
			continue
		}
		if customV2OnlineSlotRemoveRect(slot).Contains(x, y) {
			return menuHitCustomV2SlotRemoveBase + slot
		}
	}
	return menuHitNone
}

func (g *Game) returnToOnlineLobbyFromCustomSetup() {
	if g == nil || g.netplay == nil {
		return
	}
	g.customLivesFocus = false
	g.openOnlineCustomSlots("SESSION ACTIVE - WAITING FOR PLAYER / ADD AI")
}

func (g *Game) backToOnlineSessionSetupFromWizard() {
	if g == nil {
		return
	}
	// Before a host exists, step 1 is simply the normal create-session screen.
	// If a host already exists, opening SESSION must not silently destroy it;
	// changes take effect only after APPLY & RESTART SESSION is pressed.
	g.customOnlineMode = true
	g.customNetworkOpen = true
	g.customHostSetupOpen = true
	g.multiplayerFocus = false
	g.multiplayerNameFocus = false
	g.customLivesFocus = false
	g.customRulesOpen = false
	g.customChaosOpen = false
	g.menuPressed = menuHitNone
	if g.netplay != nil && g.netplay.mode == netplayHost {
		g.multiplayerMessage = "SESSION SETTINGS - APPLYING RESTARTS THE HOST"
	} else {
		g.multiplayerMessage = "CONFIGURE THE ONLINE SESSION"
	}
}

func (g *Game) setCustomV2Page(page int) {
	if g == nil {
		return
	}
	switch page {
	case customPageGame:
		g.customPage = customPageGame
		g.customMenuX = customPageTargetX(customPageGame) // keeps modifier overlay coordinates valid
	case customPageMaps:
		g.customPage = customPageMaps
		g.customMenuX = customPageTargetX(customPageMaps)
		g.ensureCustomPlaylistState()
	case customPagePlayers:
		g.customPage = customPagePlayers
		g.customMenuX = customPageTargetX(customPagePlayers) // player-card hitboxes are already screen-local here
	}
	g.customLivesFocus = false
	g.menuPressed = menuHitNone
}

func (g *Game) customV2HitAt(x, y float64) int {
	// Both Local and Online are step-based flows. The step headers are real
	// navigation controls, not decorative labels.
	if !g.customOnlineMode {
		switch {
		case customSessionLocalRect.Contains(x, y):
			return menuHitCustomSessionLocal
		case customSessionOnlineRect.Contains(x, y):
			return menuHitCustomSessionOnline
		case customV2TabMatchRect.Contains(x, y):
			return menuHitCustomV2TabMatch
		case customV2TabPlaylistRect.Contains(x, y):
			return menuHitCustomV2TabPlaylist
		case customV2TabPlayersRect.Contains(x, y):
			return menuHitCustomV2TabPlayers
		}
	} else if g.netplay != nil && g.netplay.mode == netplayHost {
		for i, r := range customV2OnlineStepRects {
			if !r.Contains(x, y) {
				continue
			}
			switch i {
			case 0:
				return menuHitCustomV2TabSession
			case 1:
				return menuHitCustomV2TabMatch
			case 2:
				return menuHitCustomV2TabPlaylist
			case 3:
				return menuHitCustomV2TabPlayers
			}
		}
	}
	switch {
	case customV2BackRect.Contains(x, y):
		return menuHitCustomV2Back
	case customV2StartRect.Contains(x, y):
		return menuHitCustomV2Start
	}

	switch g.customPage {
	case customPageGame:
		for _, row := range customV2ModeRows {
			if row.rect.Contains(x, y) {
				return menuHitModeBase + row.mode
			}
		}
		if modeHasLives(g.customMode) {
			switch {
			case customV2LivesMinusRect.Contains(x, y):
				return menuHitModeLivesMinus
			case Rect{X: 414, Y: 283, W: 84, H: 38}.Contains(x, y):
				return menuHitModeLives
			case customV2LivesPlusRect.Contains(x, y):
				return menuHitModeLivesPlus
			}
		}
		if customV2RoundsMinusRect.Contains(x, y) {
			return menuHitModeRoundsMinus
		}
		if customV2RoundsPlusRect.Contains(x, y) {
			return menuHitModeRoundsPlus
		}
		if modeHasCrateToggle(g.customMode) && customV2CratesRect.Contains(x, y) {
			return menuHitModeCrates
		}
		if modeHasPickupToggle(g.customMode) && customV2PickupsRect.Contains(x, y) {
			return menuHitModePickups
		}
		if g.customModifierEditingAllowed() && customV2ModifiersRect.Contains(x, y) {
			return menuHitCustomModifiersOpen
		}
	case customPageMaps:
		if customV2PlaylistGM1Rect.Contains(x, y) {
			return menuHitCustomV2PlaylistGM1
		}
		if customV2PlaylistGM2Rect.Contains(x, y) {
			return menuHitCustomV2PlaylistGM2
		}
		if customV2ApplyAllRect.Contains(x, y) {
			return menuHitCustomV2ApplyAll
		}
		for row := 0; row < customV2PlaylistVisibleRows; row++ {
			round := g.customPlaylistScroll + row + 1
			if round > g.customSeriesRounds {
				break
			}
			if customV2RoundRect(row).Contains(x, y) {
				return menuHitCustomV2RoundBase + round
			}
		}
		for i := range customV2MapChoices(g.customPlaylistGM2) {
			if customV2MapChoiceRect(i).Contains(x, y) {
				return menuHitCustomV2MapBase + i
			}
		}
	case customPagePlayers:
		if removeHit := g.customV2OnlineSlotRemoveHitAt(x, y); removeHit != menuHitNone {
			return removeHit
		}
		if editorHit := g.customPlayerEditorHitAt(x, y); editorHit != menuHitNone {
			if slot, ok := customPlayerEditorHitSlot(editorHit); !g.customOnlineMode || !ok || g.onlineHostMayEditCustomSlot(slot) {
				return editorHit
			}
		}
		if playerHit := g.customPlayerSetupHitAt(x, y); playerHit != menuHitNone {
			if !g.customOnlineMode {
				return playerHit
			}
			// In Online the old source CLEAR SLOT hitbox is disabled entirely.
			// Removing a bot/slot/player is only possible through the explicit
			// REMOVE BOT / CLOSE SLOT / KICK PLAYER button below the card. This
			// prevents a click anywhere on a remote card from ejecting somebody.
			slot, action, ok := decodePlayerSlotHit(playerHit)
			if ok && g.netplay != nil && g.netplay.mode == netplayHost && slot > 0 && g.customPlayers[slot].Type == 0 &&
				(action == playerSlotActionHuman || action == playerSlotActionAI) {
				return playerHit
			}
		}
	}
	return menuHitNone
}

func (g *Game) activateCustomV2Hit(hit int) {
	if hit == menuHitNone {
		return
	}
	if g.activateCustomModeOption(hit) {
		g.ensureCustomPlaylistState()
		return
	}
	if g.activateCustomPlayerEditorHit(hit) {
		return
	}
	if g.activateCustomPlayerSlot(hit) {
		if g.customOnlineMode && g.netplay != nil && g.netplay.mode == netplayHost {
			g.syncOnlineHostState()
		}
		return
	}
	if hit >= menuHitModeBase+SourceGameModeNormal && hit <= menuHitModeBase+SourceGameModeSurvival {
		g.customMode = hit - menuHitModeBase
		g.customLivesFocus = false
		return
	}
	switch {
	case hit == menuHitCustomSessionLocal:
		g.customOnlineMode = false
	case hit == menuHitCustomSessionOnline:
		g.customOnlineMode = true
		g.returnToOnlineServerBrowser()
	case hit == menuHitCustomV2TabSession:
		if g.customOnlineMode && g.netplay != nil && g.netplay.mode == netplayHost {
			g.backToOnlineSessionSetupFromWizard()
		}
	case hit == menuHitCustomV2TabMatch:
		g.setCustomV2Page(customPageGame)
		if g.customOnlineMode && g.netplay != nil && g.netplay.mode == netplayHost {
			g.syncOnlineHostState()
		}
	case hit == menuHitCustomV2TabPlaylist:
		g.setCustomV2Page(customPageMaps)
		if g.customOnlineMode && g.netplay != nil && g.netplay.mode == netplayHost {
			g.syncOnlineHostState()
		}
	case hit == menuHitCustomV2TabPlayers:
		g.setCustomV2Page(customPagePlayers)
		if g.customOnlineMode && g.netplay != nil && g.netplay.mode == netplayHost {
			g.syncOnlineHostState()
		}
	case hit == menuHitCustomV2Back:
		if g.customOnlineMode {
			if g.netplay != nil && g.netplay.mode == netplayClient {
				_ = g.CloseNetplay()
				g.returnToOnlineServerBrowser()
				return
			}
			switch g.customPage {
			case customPagePlayers:
				g.setCustomV2Page(customPageMaps)
			case customPageMaps:
				g.setCustomV2Page(customPageGame)
			default:
				g.backToOnlineSessionSetupFromWizard()
			}
			return
		}
		switch g.customPage {
		case customPagePlayers:
			g.setCustomV2Page(customPageMaps)
		case customPageMaps:
			g.setCustomV2Page(customPageGame)
		default:
			g.beginFade(screenMainMenu, fadePurposeScreen)
		}
	case hit == menuHitCustomV2Start:
		if !g.customOnlineMode {
			switch g.customPage {
			case customPageGame:
				if modeHasLives(g.customMode) && g.customLives <= 0 {
					g.playCustomWarning(1)
					return
				}
				g.setCustomV2Page(customPageMaps)
				return
			case customPageMaps:
				g.setCustomV2Page(customPagePlayers)
				return
			case customPagePlayers:
				if g.activeCustomPlayerCount() < 2 {
					g.playCustomWarning(2)
					return
				}
			}
		}
		if g.customOnlineMode {
			if g.netplay == nil {
				g.returnToOnlineServerBrowser()
				return
			}
			if g.netplay.mode == netplayClient {
				return
			}
			switch g.customPage {
			case customPageGame:
				if modeHasLives(g.customMode) && g.customLives <= 0 {
					g.playCustomWarning(1)
					return
				}
				g.setCustomV2Page(customPageMaps)
				g.syncOnlineHostState()
				return
			case customPageMaps:
				g.setCustomV2Page(customPagePlayers)
				g.syncOnlineHostState()
				return
			case customPagePlayers:
				if g.onlineActiveParticipantCount() < 2 {
					g.multiplayerMessage = "WAITING FOR ANOTHER PLAYER OR AI BOT"
					return
				}
			}
		}
		if modeHasLives(g.customMode) && g.customLives <= 0 {
			g.playCustomWarning(1)
			g.setCustomV2Page(customPageGame)
			return
		}
		g.startCustomGame()
	case hit == menuHitCustomV2PlaylistGM1:
		g.customPlaylistGM2 = false
		g.customMapThumbPreloadCursor = 0
	case hit == menuHitCustomV2PlaylistGM2:
		if gm2MapAssetsAvailable() {
			g.customPlaylistGM2 = true
			g.customMapThumbPreloadCursor = 0
		}
	case hit == menuHitCustomV2ApplyAll:
		g.ensureCustomPlaylistState()
		mapValue := g.customSeriesMapForRound(g.customPlaylistSelectedRound)
		g.setCustomSeriesMapForAll(mapValue)
	case hit >= menuHitCustomV2RoundBase+1 && hit <= menuHitCustomV2RoundBase+customSeriesMaxRounds:
		round := hit - menuHitCustomV2RoundBase
		if round >= 1 && round <= g.customSeriesRounds {
			g.customPlaylistSelectedRound = round
			mapValue := g.customSeriesMapForRound(round)
			g.customPlaylistGM2 = isGM2MapID(mapValue) || mapValue == customGM2RandomNew || mapValue == customGM2RandomAll
		}
	case hit >= menuHitCustomV2SlotRemoveBase && hit < menuHitCustomV2SlotRemoveBase+4:
		g.removeOnlineCustomSlot(hit - menuHitCustomV2SlotRemoveBase)
	case hit >= menuHitCustomV2MapBase:
		choices := customV2MapChoices(g.customPlaylistGM2)
		index := hit - menuHitCustomV2MapBase
		if index >= 0 && index < len(choices) {
			g.setCustomSeriesMap(g.customPlaylistSelectedRound, choices[index].value)
		}
	}
}

func (g *Game) updateCustomV2PlaylistScroll(x, y float64) {
	if g == nil || g.customPage != customPageMaps || !customV2PlaylistListRect.Contains(x, y) {
		return
	}
	_, wheelY := ebiten.Wheel()
	if wheelY == 0 {
		return
	}
	if wheelY > 0 {
		g.customPlaylistScroll--
	} else {
		g.customPlaylistScroll++
	}
	g.ensureCustomPlaylistState()
}

func (g *Game) updateCustomV2MapThumbPreload() {
	if g == nil || g.customPage != customPageMaps {
		return
	}
	choices := customV2MapChoices(g.customPlaylistGM2)
	if len(choices) == 0 {
		return
	}
	// Decode at most one real map thumbnail per update. Switching to GM2 no
	// longer blocks on loading all nine arenas at once; their thumbnails fill
	// in over the next few frames and then stay cached for the lifetime of game.
	for checked := 0; checked < len(choices); checked++ {
		i := (g.customMapThumbPreloadCursor + checked) % len(choices)
		mapNumber := choices[i].value
		if mapNumber <= 0 {
			continue
		}
		if g.customMapThumbCache != nil && g.customMapThumbCache[mapNumber] != nil {
			continue
		}
		g.ensureOnlineMapThumb(mapNumber)
		g.customMapThumbPreloadCursor = (i + 1) % len(choices)
		return
	}
}

func (g *Game) updateCustomGameV2() {
	g.updateCustomWarning()
	mx, my := ebiten.CursorPosition()
	x, y := float64(mx), float64(my)

	if g.customNetworkOpen {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if g.customHostSetupOpen {
				if g.netplay != nil && g.netplay.mode == netplayHost {
					// ESC is an unconditional escape hatch from a running host. The
					// visible CANCEL button still returns to the wizard without closing.
					g.returnToOnlineServerBrowser()
				} else {
					g.customHostSetupOpen = false
					g.multiplayerNameFocus = false
					g.menuPressed = menuHitNone
					g.multiplayerMessage = "CHOOSE A SESSION OR CREATE ONE"
				}
			} else {
				g.leaveCustomNetworkPanel()
				g.menuPressed = menuHitNone
			}
			return
		}
		g.updateCustomSessionInput(x, y)
		return
	}
	if g.customOnlineRulesOverlayActive() {
		g.updateMultiplayerLobby()
		return
	}
	if g.customRulesOpen {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if g.customChaosOpen {
				g.customChaosOpen = false
			} else {
				g.customRulesOpen = false
			}
			g.menuPressed = menuHitNone
			return
		}
		g.customMenuX = customPageTargetX(customPageGame)
		hit := g.customModifierHitAt(x-g.customMenuX, y)
		activated, ok := g.resolveMenuRelease(hit)
		if ok {
			g.activateCustomModifierHit(activated)
		}
		return
	}

	if g.customPage == customPageGame {
		g.updateCustomLivesInput()
	}
	if g.customPage == customPagePlayers {
		g.customMenuX = 0
		g.updateCustomPlayerCards()
		g.updateCustomNameInput()
	}
	g.updateCustomV2PlaylistScroll(x, y)
	g.updateCustomV2MapThumbPreload()

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.menuPressed = menuHitNone
		if g.customOnlineMode {
			if g.netplay != nil && g.netplay.mode == netplayClient {
				_ = g.CloseNetplay()
				g.returnToOnlineServerBrowser()
				return
			}
			switch g.customPage {
			case customPagePlayers:
				g.setCustomV2Page(customPageMaps)
			case customPageMaps:
				g.setCustomV2Page(customPageGame)
			default:
				g.backToOnlineSessionSetupFromWizard()
			}
			return
		}
		switch g.customPage {
		case customPagePlayers:
			g.setCustomV2Page(customPageMaps)
		case customPageMaps:
			g.setCustomV2Page(customPageGame)
		default:
			g.beginFade(screenMainMenu, fadePurposeScreen)
		}
		return
	}

	hit := g.customV2HitAt(x, y)
	activated, ok := g.resolveMenuRelease(hit)
	if !ok {
		return
	}
	g.activateCustomV2Hit(activated)
}

func drawCustomV2Tab(screen *ebiten.Image, r Rect, label string, selected bool) {
	fill := color.NRGBA{R: 0xd8, G: 0xd8, B: 0xd8, A: 0xff}
	text := color.Black
	if selected {
		fill = color.NRGBA{R: 0xff, G: 0x92, B: 0x00, A: 0xff}
	}
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, fill)
	drawSourceMenuText(screen, label, menuFontCondensedExtraBold, 14, text, r.X+12, r.Y+8)
}

func drawCustomLocalStep(screen *ebiten.Image, r Rect, number, label string, selected bool) {
	fill := color.NRGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff}
	text := color.NRGBA{R: 0x23, G: 0x23, B: 0x23, A: 0xff}
	if selected {
		fill = color.NRGBA{R: 0xff, G: 0x92, B: 0x00, A: 0xff}
	}
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, fill)
	drawSourceMenuText(screen, number, menuFontCondensedExtraBold, 12, text, r.X+10, r.Y+5)
	drawSourceMenuText(screen, label, menuFontTwCen, 15, text, r.X+29, r.Y+7)
}

func drawCustomOnlineTab(screen *ebiten.Image, r Rect, label string, selected bool) {
	fill := color.NRGBA{R: 0x2a, G: 0x31, B: 0x3c, A: 0xff}
	text := color.NRGBA{R: 0xd9, G: 0xdf, B: 0xe8, A: 0xff}
	if selected {
		fill = color.NRGBA{R: 0x06, G: 0x68, B: 0xe8, A: 0xff}
		text = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	}
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, fill)
	drawSourceMenuText(screen, label, menuFontCondensedExtraBold, 13.5, text, r.X+11, r.Y+8)
}

func (g *Game) drawCustomV2Shell(screen *ebiten.Image) {
	if g.customOnlineMode {
		g.drawCustomV2OnlineShell(screen)
		return
	}
	g.drawCustomV2LocalShell(screen)
}

func (g *Game) drawCustomV2LocalShell(screen *ebiten.Image) {
	// Local Custom Game is intentionally a couch-game setup, not the online UI
	// with networking labels removed. The shared match data stays the same, but
	// the presentation is deliberately simpler and warmer.
	screen.Fill(color.NRGBA{R: 0xe7, G: 0xe7, B: 0xe7, A: 0xff})
	ebitenutil.DrawRect(screen, 15, 14, 870, 78, color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xff})
	drawSourceMenuText(screen, "LOCAL CUSTOM GAME", menuFontCondensedExtraBold, 31, color.White, 30, 23)
	drawSourceMenuText(screen, "ONE COMPUTER  /  LOCAL PLAYERS + AI BOTS", menuFontCondensedExtraBold, 10.5,
		color.NRGBA{R: 0xff, G: 0xb0, B: 0x45, A: 0xff}, 31, 60)

	drawCustomLocalStep(screen, customV2TabMatchRect, "1", "MATCH", g.customPage == customPageGame)
	drawCustomLocalStep(screen, customV2TabPlaylistRect, "2", "MAPS", g.customPage == customPageMaps)
	drawCustomLocalStep(screen, customV2TabPlayersRect, "3", "PLAYERS", g.customPage == customPagePlayers)

	ebitenutil.DrawRect(screen, customV2BackRect.X, customV2BackRect.Y, customV2BackRect.W, customV2BackRect.H, color.White)
	backLabel := "MAIN MENU"
	switch g.customPage {
	case customPageMaps:
		backLabel = "BACK TO MATCH"
	case customPagePlayers:
		backLabel = "BACK TO MAPS"
	}
	drawSourceMenuText(screen, backLabel, menuFontTwCen, 22, color.Black, customV2BackRect.X+14, customV2BackRect.Y+14)

	startLabel := "NEXT: MAPS"
	if g.customPage == customPageMaps {
		startLabel = "NEXT: PLAYERS"
	} else if g.customPage == customPagePlayers {
		startLabel = "START LOCAL GAME"
	}
	ebitenutil.DrawRect(screen, customV2StartRect.X, customV2StartRect.Y, customV2StartRect.W, customV2StartRect.H,
		color.NRGBA{R: 0xff, G: 0x92, B: 0x00, A: 0xff})
	drawSourceMenuText(screen, startLabel, menuFontTwCen, 19, color.Black, customV2StartRect.X+17, customV2StartRect.Y+15)

	g.drawCustomSessionControls(screen)
}

func drawOnlineWizardStep(screen *ebiten.Image, r Rect, number, label string, state int) {
	fill := color.NRGBA{R: 0x2a, G: 0x31, B: 0x3c, A: 0xff}
	text := color.NRGBA{R: 0xb7, G: 0xc0, B: 0xcc, A: 0xff}
	if state == 1 {
		fill = color.NRGBA{R: 0x06, G: 0x68, B: 0xe8, A: 0xff}
		text = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	} else if state == 2 {
		fill = color.NRGBA{R: 0x24, G: 0x46, B: 0x3b, A: 0xff}
		text = color.NRGBA{R: 0xb8, G: 0xef, B: 0xd2, A: 0xff}
	}
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, fill)
	drawSourceMenuText(screen, number, menuFontCondensedExtraBold, 10, text, r.X+7, r.Y+9)
	drawSourceMenuText(screen, label, menuFontCondensedExtraBold, 11.5, text, r.X+24, r.Y+8)
}

func (g *Game) drawCustomV2OnlineShell(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 0x15, G: 0x1b, B: 0x24, A: 0xff})
	ebitenutil.DrawRect(screen, 15, 14, 870, 78, color.NRGBA{R: 0x1f, G: 0x28, B: 0x35, A: 0xff})
	ebitenutil.DrawRect(screen, 15, 14, 6, 78, color.NRGBA{R: 0x06, G: 0x68, B: 0xe8, A: 0xff})
	drawSourceMenuText(screen, "ONLINE CUSTOM GAME", menuFontCondensedExtraBold, 27, color.White, 31, 21)
	status := g.customSessionStatus()
	drawSourceMenuText(screen, status, menuFontCondensedExtraBold, 10,
		color.NRGBA{R: 0x74, G: 0xb6, B: 0xff, A: 0xff}, 32, 57)

	current := 1
	if g.customPage == customPageMaps {
		current = 2
	}
	if g.customPage == customPagePlayers {
		current = 3
	}
	labels := [4]string{"SESSION", "MODE", "MAPS", "SLOTS"}
	for i := 0; i < 4; i++ {
		state := 0
		if i < current {
			state = 2
		}
		if i == current {
			state = 1
		}
		drawOnlineWizardStep(screen, customV2OnlineStepRects[i], fmt.Sprintf("%d", i+1), labels[i], state)
	}

	ebitenutil.DrawRect(screen, customV2BackRect.X, customV2BackRect.Y, customV2BackRect.W, customV2BackRect.H,
		color.NRGBA{R: 0x2b, G: 0x31, B: 0x3a, A: 0xff})
	backLabel := "BACK TO SESSION"
	if g.netplay != nil && g.netplay.mode == netplayClient {
		backLabel = "LEAVE SERVER"
	} else if g.customPage == customPageMaps {
		backLabel = "BACK TO MODE"
	} else if g.customPage == customPagePlayers {
		backLabel = "BACK TO MAPS"
	}
	drawSourceMenuText(screen, backLabel, menuFontTwCen, 18, color.White, customV2BackRect.X+14, customV2BackRect.Y+16)

	startFill := color.NRGBA{R: 0x06, G: 0x68, B: 0xe8, A: 0xff}
	startLabel := "NEXT: MAPS"
	if g.customPage == customPageMaps {
		startLabel = "NEXT: SLOTS"
	} else if g.customPage == customPagePlayers {
		startLabel = "START ONLINE MATCH"
		if g.netplay != nil && g.netplay.mode == netplayClient {
			startFill = color.NRGBA{R: 0x3c, G: 0x43, B: 0x4d, A: 0xff}
			startLabel = "WAIT FOR HOST"
		} else if g.onlineActiveParticipantCount() < 2 {
			startFill = color.NRGBA{R: 0x3c, G: 0x43, B: 0x4d, A: 0xff}
			startLabel = "WAIT FOR PLAYER / ADD AI"
		}
	}
	ebitenutil.DrawRect(screen, customV2StartRect.X, customV2StartRect.Y, customV2StartRect.W, customV2StartRect.H, startFill)
	drawSourceMenuText(screen, startLabel, menuFontTwCen, 17, color.White, customV2StartRect.X+14, customV2StartRect.Y+17)
}

func (g *Game) drawCustomV2Match(screen *ebiten.Image) {
	drawSetupPanel(screen, Rect{X: 20, Y: 105, W: 275, H: 405})
	drawSetupPanel(screen, Rect{X: 310, Y: 105, W: 570, H: 405})
	modeHeading := "LOCAL GAME MODE"
	settingsHeading := "LOCAL MATCH SETTINGS"
	if g.customOnlineMode {
		modeHeading = "ONLINE GAME MODE"
		settingsHeading = "HOST MATCH SETTINGS"
		ebitenutil.DrawRect(screen, 20, 105, 275, 3, color.NRGBA{R: 0x06, G: 0x68, B: 0xe8, A: 0xff})
		ebitenutil.DrawRect(screen, 310, 105, 570, 3, color.NRGBA{R: 0x06, G: 0x68, B: 0xe8, A: 0xff})
	}
	drawSourceMenuText(screen, modeHeading, menuFontCondensedExtraBold, 20, color.White, 35, 116)
	for _, row := range customV2ModeRows {
		fill := color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}
		if g.customMode == row.mode {
			fill = color.NRGBA{R: 0xff, G: 0x92, B: 0x00, A: 0xff}
		}
		ebitenutil.DrawRect(screen, row.rect.X, row.rect.Y, row.rect.W, row.rect.H, fill)
		drawSourceMenuText(screen, row.label, menuFontTwCen, 18, color.Black, row.rect.X+12, row.rect.Y+15)
	}

	drawSourceMenuText(screen, developedModeTitle(g.customMode), menuFontCondensedExtraBold, 29, color.White, 325, 120)
	for i, line := range developedModeDescription(g.customMode) {
		drawSourceMenuText(screen, line, menuFontCondensed, 14, color.NRGBA{R: 0xc9, G: 0xcd, B: 0xd3, A: 0xff}, 325, 158+float64(i)*22)
	}
	drawSourceMenuText(screen, settingsHeading, menuFontCondensedExtraBold, 18, color.White, 325, 224)

	ebitenutil.DrawRect(screen, 325, 255, 235, 80, color.NRGBA{R: 0x27, G: 0x2b, B: 0x32, A: 0xff})
	drawSourceMenuText(screen, "LIVES", menuFontCondensedExtraBold, 14, color.White, 340, 265)
	if modeHasLives(g.customMode) {
		drawSetupSmallButton(screen, customV2LivesMinusRect, "-", g.customLives > 1)
		drawSetupSmallButton(screen, customV2LivesPlusRect, "+", true)
		value := fmt.Sprintf("%d", g.customLives)
		if g.customLivesFocus {
			value += "_"
		}
		drawSourceMenuText(screen, value, menuFontTwCen, 23, color.White, 430, 290)
	} else {
		drawSourceMenuText(screen, "MODE RULE", menuFontTwCen, 18, color.NRGBA{R: 0xa8, G: 0xae, B: 0xb6, A: 0xff}, 380, 292)
	}

	ebitenutil.DrawRect(screen, 590, 255, 255, 80, color.NRGBA{R: 0x27, G: 0x2b, B: 0x32, A: 0xff})
	drawSourceMenuText(screen, "ROUNDS", menuFontCondensedExtraBold, 14, color.White, 605, 265)
	drawSetupSmallButton(screen, customV2RoundsMinusRect, "-", g.customSeriesRounds > 1)
	drawSetupSmallButton(screen, customV2RoundsPlusRect, "+", g.customSeriesRounds < customSeriesMaxRounds)
	drawSourceMenuText(screen, fmt.Sprintf("%d", g.customSeriesRounds), menuFontTwCen, 23, color.White, 700, 290)
	drawSourceMenuText(screen, "Each round can use a different map in PLAYLIST.", menuFontCondensed, 10.5, color.NRGBA{R: 0xa9, G: 0xaf, B: 0xb8, A: 0xff}, 605, 321)

	crateState := "MODE RULE"
	crateActive := false
	crateEnabled := modeHasCrateToggle(g.customMode)
	if crateEnabled {
		crateActive = g.CrateON
		if g.CrateON {
			crateState = "ON"
		} else {
			crateState = "OFF"
		}
	}
	drawSetupToggleCard(screen, customV2CratesRect, "CRATES", crateState, crateActive, crateEnabled)
	pickupState := "MODE RULE"
	pickupActive := false
	pickupEnabled := modeHasPickupToggle(g.customMode)
	if pickupEnabled {
		pickupActive = g.PowerON
		if g.PowerON {
			pickupState = "ON"
		} else {
			pickupState = "OFF"
		}
	}
	drawSetupToggleCard(screen, customV2PickupsRect, "PICKUPS", pickupState, pickupActive, pickupEnabled)

	ebitenutil.DrawRect(screen, customV2ModifiersRect.X, customV2ModifiersRect.Y, customV2ModifiersRect.W, customV2ModifiersRect.H, color.NRGBA{R: 0x31, G: 0x35, B: 0x3d, A: 0xff})
	drawSourceMenuText(screen, "MODIFIERS / CHAOS", menuFontCondensedExtraBold, 14, color.White, 340, 442)
	drawSourceMenuText(screen, g.developedModifierSummary(), menuFontCondensed, 12.5, color.NRGBA{R: 0xc8, G: 0xcc, B: 0xd3, A: 0xff}, 495, 443)
	drawSourceMenuText(screen, "CLICK TO CONFIGURE", menuFontCondensedExtraBold, 10.5, color.NRGBA{R: 0xff, G: 0xa7, B: 0x34, A: 0xff}, 340, 469)

	if g.customWarningFrame > 0 && g.customWarning == 1 {
		ebitenutil.DrawRect(screen, 325, 496, 520, 24, color.NRGBA{R: 0x8e, G: 0x32, B: 0x21, A: 0xff})
		drawSourceMenuText(screen, "LIVES MUST BE AT LEAST 1", menuFontCondensedExtraBold, 12, color.White, 340, 501)
	}
}

func (g *Game) drawCustomV2Playlist(screen *ebiten.Image) {
	g.ensureCustomPlaylistState()
	drawSetupPanel(screen, Rect{X: 20, Y: 105, W: 325, H: 405})
	drawSetupPanel(screen, Rect{X: 360, Y: 105, W: 520, H: 405})
	playlistHeading := "LOCAL MAP SERIES"
	if g.customOnlineMode {
		playlistHeading = "ONLINE MAP PLAYLIST"
		ebitenutil.DrawRect(screen, 20, 105, 325, 3, color.NRGBA{R: 0x06, G: 0x68, B: 0xe8, A: 0xff})
		ebitenutil.DrawRect(screen, 360, 105, 520, 3, color.NRGBA{R: 0x06, G: 0x68, B: 0xe8, A: 0xff})
	}
	drawSourceMenuText(screen, playlistHeading, menuFontCondensedExtraBold, 22, color.White, 35, 118)
	drawSourceMenuText(screen, fmt.Sprintf("%d ROUNDS - EACH ROUND HAS ITS OWN MAP", g.customSeriesRounds), menuFontCondensed, 11.5, color.NRGBA{R: 0xb7, G: 0xbc, B: 0xc4, A: 0xff}, 35, 140)

	for row := 0; row < customV2PlaylistVisibleRows; row++ {
		round := g.customPlaylistScroll + row + 1
		if round > g.customSeriesRounds {
			break
		}
		r := customV2RoundRect(row)
		selected := round == g.customPlaylistSelectedRound
		fill := color.NRGBA{R: 0x2c, G: 0x30, B: 0x37, A: 0xff}
		if selected {
			fill = color.NRGBA{R: 0xff, G: 0x92, B: 0x00, A: 0xff}
		}
		ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, fill)
		text := color.White
		if selected {
			text = color.Black
		}
		drawSourceMenuText(screen, fmt.Sprintf("ROUND %02d", round), menuFontCondensedExtraBold, 12, text, r.X+10, r.Y+8)
		label := customSeriesMapLabel(g.customSeriesMapForRound(round))
		if len(label) > 26 {
			label = label[:26]
		}
		drawSourceMenuText(screen, label, menuFontCondensed, 10.5, text, r.X+92, r.Y+9)
	}
	if g.customSeriesRounds > customV2PlaylistVisibleRows {
		drawSourceMenuText(screen, "MOUSE WHEEL TO SCROLL", menuFontCondensedExtraBold, 10.5, color.NRGBA{R: 0x9e, G: 0xa4, B: 0xad, A: 0xff}, 45, 493)
	}

	round := g.customPlaylistSelectedRound
	drawSourceMenuText(screen, fmt.Sprintf("CHOOSE MAP FOR ROUND %d", round), menuFontCondensedExtraBold, 20, color.White, 380, 119)
	drawCustomV2Tab(screen, customV2PlaylistGM1Rect, "GM1", !g.customPlaylistGM2)
	gm2Enabled := gm2MapAssetsAvailable()
	if gm2Enabled {
		drawCustomV2Tab(screen, customV2PlaylistGM2Rect, "GM2", g.customPlaylistGM2)
	} else {
		ebitenutil.DrawRect(screen, customV2PlaylistGM2Rect.X, customV2PlaylistGM2Rect.Y, customV2PlaylistGM2Rect.W, customV2PlaylistGM2Rect.H, color.NRGBA{R: 0x3d, G: 0x40, B: 0x46, A: 0xff})
		drawSourceMenuText(screen, "GM2 N/A", menuFontCondensedExtraBold, 12, color.NRGBA{R: 0x85, G: 0x89, B: 0x90, A: 0xff}, customV2PlaylistGM2Rect.X+12, customV2PlaylistGM2Rect.Y+9)
	}
	drawModifierButton(screen, customV2ApplyAllRect, "APPLY THIS MAP TO ALL", false)

	selectedMap := g.customSeriesMapForRound(round)
	for i, choice := range customV2MapChoices(g.customPlaylistGM2) {
		r := customV2MapChoiceRect(i)
		selected := selectedMap == choice.value
		fill := color.NRGBA{R: 0xe8, G: 0xe8, B: 0xe8, A: 0xff}
		if selected {
			fill = color.NRGBA{R: 0xff, G: 0x92, B: 0x00, A: 0xff}
		}
		ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, fill)
		// Restore the graphical map thumbnails in both Local and Online. These
		// use the real map preview/scene assets already shipped with the game.
		thumb := Rect{X: r.X + 3, Y: r.Y + 2, W: 55, H: r.H - 4}
		g.drawOnlineMapThumbIfReady(screen, choice.value, thumb)
		label := choice.label
		if len(label) > 20 {
			label = label[:20]
		}
		drawSourceMenuText(screen, label, menuFontTwCen, 10.8, color.Black, r.X+64, r.Y+10)
	}
}

func (g *Game) drawCustomV2Players(screen *ebiten.Image) {
	g.customMenuX = 0
	for i, cfg := range g.customPlayers {
		cardX := 20 + float64(i)*220
		g.drawPlayerSetupCardMasked(screen, cfg, g.customEditor[i], i, cardX, g.customCardY[i], 1, g.customColorMask[i], false)
	}
	g.drawCustomOnlineSlotOverlay(screen)

	if g.customOnlineMode && g.netplay != nil && g.netplay.mode == netplayHost {
		for slot := 1; slot < len(g.customPlayers); slot++ {
			cfg := g.customPlayers[slot]
			if cfg.Type == 0 {
				continue
			}
			label := "CLOSE SLOT"
			if cfg.Type == 2 {
				label = "REMOVE BOT"
			} else if g.onlinePlayerSlots[slot] {
				label = "KICK PLAYER"
			}
			r := customV2OnlineSlotRemoveRect(slot)
			ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, color.NRGBA{R: 0x5b, G: 0x2d, B: 0x2d, A: 0xff})
			drawSourceMenuText(screen, label, menuFontCondensedExtraBold, 11.5, color.White, r.X+14, r.Y+7)
		}
	}

	panelFill := color.NRGBA{R: 0x2b, G: 0x2b, B: 0x2b, A: 0xff}
	accent := color.NRGBA{R: 0xff, G: 0x92, B: 0x00, A: 0xff}
	label := "LOCAL: PLAYERS ON THIS COMPUTER + AI BOTS"
	if g.customOnlineMode {
		panelFill = color.NRGBA{R: 0x1d, G: 0x26, B: 0x32, A: 0xff}
		accent = color.NRGBA{R: 0x06, G: 0x68, B: 0xe8, A: 0xff}
		label = "ONLINE: OPEN NETWORK SLOTS OR ADD AI BOTS"
		if g.netplay != nil && g.netplay.mode == netplayHost {
			label = g.currentHostDisplayJoinCodeLabel()
		}
	}
	ebitenutil.DrawRect(screen, 20, 501, 860, 20, panelFill)
	ebitenutil.DrawRect(screen, 20, 501, 5, 20, accent)
	fontSize := 10.5
	if strings.HasPrefix(label, "JOIN CODE: ") {
		fontSize = 9.5
	}
	drawSourceMenuText(screen, label, menuFontCondensedExtraBold, fontSize, color.White, 34, 505)
}

func (g *Game) drawCustomGameV2(screen *ebiten.Image) {
	if g.customNetworkOpen {
		g.drawCustomNetworkPanel(screen)
		return
	}
	g.drawCustomV2Shell(screen)
	switch g.customPage {
	case customPageMaps:
		g.drawCustomV2Playlist(screen)
	case customPagePlayers:
		g.drawCustomV2Players(screen)
	default:
		g.drawCustomV2Match(screen)
	}
	if g.customRulesOpen {
		g.customMenuX = customPageTargetX(customPageGame)
		g.drawCustomModifiersOverlay(screen)
	}
}
