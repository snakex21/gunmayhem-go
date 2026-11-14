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
