package game

import (
	"fmt"
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

// Geometry of the online wizard shell (MODE / MAPS / SLOTS). Local Custom
// Game is the kit screen in custom_local_menu.go.
var (
	customV2BackRect  = Rect{X: 20, Y: 528, W: 190, H: 52}
	customV2StartRect = Rect{X: 660, Y: 528, W: 220, H: 52}

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

// customV2HitAt is the online wizard's hit test (local Custom Game uses the
// kit, but shares activateCustomV2Hit). The step headers are real navigation
// controls, not decorative labels.
func (g *Game) customV2HitAt(x, y float64) int {
	if g.customOnlineMode && g.netplay != nil && g.netplay.mode == netplayHost {
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
		// B2: the host's RULES overlay (map rotation, weapon rule, blocklists)
		// is reachable from every wizard step.
		if customSessionRulesRect.Contains(x, y) {
			return menuHitCustomSessionRules
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
	case hit == menuHitCustomSessionRules:
		if g.customOnlineMode && g.netplay != nil && g.netplay.mode == netplayHost {
			g.onlineRulesOpen = true
			g.menuPressed = menuHitNone
		}
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

func (g *Game) updateCustomV2MapThumbPreload() {
	if g == nil || g.customPage != customPageMaps {
		return
	}
	g.preloadCustomMapThumbs()
}

// preloadCustomMapThumbs decodes at most one missing MAPS-tab thumbnail.
func (g *Game) preloadCustomMapThumbs() {
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
