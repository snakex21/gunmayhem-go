package game

// openOnlineCustomSlots is the single online waiting/setup screen. The old
// standalone ONLINE LOBBY screen is retained only as a protocol-era enum value;
// normal gameplay must always return to the Custom Game SLOTS page instead.
func (g *Game) openOnlineCustomSlots(message string) {
	if g == nil {
		return
	}
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.customNetworkOpen = false
	g.customHostSetupOpen = false
	g.customRulesOpen = false
	g.customChaosOpen = false
	g.onlineLobbyCustomize = false
	g.onlineRulesOpen = false
	g.onlineBlocklistOpen = false
	g.onlinePerkBlocklistOpen = false
	g.onlineVisualPicker = onlineVisualNone
	g.onlineVisualPage = 0
	g.menuPressed = menuHitNone
	g.pausePressed = pauseHitNone
	g.setCustomV2Page(customPagePlayers)
	g.customMenuX = 0
	g.screen = screenCustomGame
	g.multiplayerMessage = message
}

// clearDisconnectedOnlineHumanSlots converts remote human slots that no longer
// have a peer into real EMPTY slots. Waiting slots from a finished/aborted match
// should not masquerade as connected players when the host returns to setup.
func (g *Game) clearDisconnectedOnlineHumanSlots() {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost {
		return
	}
	connected := [5]bool{false, true, false, false, false}
	for _, id := range g.netplay.connectedPlayerIDs() {
		if id >= 2 && id <= maxOnlinePlayers {
			connected[id] = true
		}
	}
	for id := 2; id <= maxOnlinePlayers; id++ {
		slot := id - 1
		if g.customPlayers[slot].Type != 1 || connected[id] {
			continue
		}
		g.customPlayers[slot].Type = 0
		g.customEditor[slot] = 0
		g.onlinePlayerSlots[slot] = false
		if g.customNameFocus == slot+1 {
			g.customNameFocus = 0
		}
	}
	g.syncOnlineHostState()
}

// returnOnlineHostToLobbyIfAlone keeps the hosted session alive but aborts the
// current online match when fewer than two active participants remain. It sends
// the host back to the unified Custom Game SLOTS page, not the obsolete lobby.
func (g *Game) returnOnlineHostToLobbyIfAlone() bool {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost {
		return false
	}
	if g.screen != screenGameplay || !g.onlineMatchSetup || g.onlineActiveParticipantCount() >= 2 {
		return false
	}

	g.paused = false
	g.fadeActive = false
	g.fadeFrame = 0
	g.GameWin = false
	g.teamGameWin = false
	g.matchWinCountdown = 0
	g.soloWinFrame = 0
	g.winnerPlayerID = 0
	g.winnerAnimFrame = 0
	g.teamWinAnimFrame = 0
	g.customSeriesRound = 0
	g.customSeriesRecordedRound = 0
	g.customSeriesStartingNext = false
	g.customSeriesStats = [4]customSeriesPlayerStats{}
	g.resetOnlinePostGameReady()
	g.resetChaosRuntime()
	g.clearDisconnectedOnlineHumanSlots()
	g.openOnlineCustomSlots("PLAYER LEFT - WAITING FOR PLAYER / ADD AI")
	networkDiagnostic("HOST match aborted: fewer than two active participants; returned to Custom Game SLOTS")
	return true
}
