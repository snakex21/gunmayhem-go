package game

const (
	menuHitCustomHostBack = 9950 + iota
	menuHitCustomHostCreate
	menuHitCustomHostScopeLAN
	menuHitCustomHostScopeInternet
	menuHitCustomHostAccessOpen
	menuHitCustomHostAccessInvite
	menuHitCustomHostLeave
)

func (g *Game) activateCustomHostSetupHit(hit int) {
	switch hit {
	case menuHitCustomHostBack:
		g.customHostSetupOpen = false
		g.multiplayerNameFocus = false
		if g.netplay != nil && g.netplay.mode == netplayHost {
			g.customNetworkOpen = false
			g.setCustomV2Page(customPageGame)
			g.multiplayerMessage = "SESSION SETTINGS UNCHANGED"
		} else {
			g.onlineZombie = false
			g.multiplayerMessage = "CHOOSE A SESSION OR CREATE ONE"
		}
	case menuHitCustomHostScopeLAN:
		g.customHostScope = onlineSessionScopeLAN
	case menuHitCustomHostScopeInternet:
		g.customHostScope = onlineSessionScopeInternet
		g.multiplayerMessage = "INTERNET QUICK TUNNEL - INVITE CODE RECOMMENDED"
	case menuHitCustomHostAccessOpen:
		g.customHostAccess = onlineSessionOpen
	case menuHitCustomHostAccessInvite:
		g.customHostAccess = onlineSessionInviteOnly
	case menuHitCustomHostLeave:
		g.returnToOnlineServerBrowser()
	case menuHitCustomHostCreate:
		g.createConfiguredOnlineSession()
	}
}

func (g *Game) createConfiguredOnlineSession() {
	scope := g.customHostScope
	if scope != onlineSessionScopeLAN && scope != onlineSessionScopeInternet {
		scope = onlineSessionScopeLAN
		g.customHostScope = scope
	}
	access := g.customHostAccess
	if access != onlineSessionOpen && access != onlineSessionInviteOnly {
		access = onlineSessionOpen
		g.customHostAccess = access
	}
	if err := g.StartNetHostConfig(":7777", scope, access); err != nil {
		g.reportNetError("HOST ERROR", err)
		return
	}
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.customNetworkOpen = false
	g.customHostSetupOpen = false
	g.multiplayerFocus = false
	g.multiplayerNameFocus = false

	scopeText := "LAN"
	if scope == onlineSessionScopeInternet {
		scopeText = "INTERNET + LAN"
	}
	accessText := "OPEN"
	if access == onlineSessionInviteOnly {
		accessText = "INVITE CODE"
	}
	g.multiplayerMessage = scopeText + " / " + accessText + " - HOSTING"
	g.syncOnlineHostState()
	// Online Custom Game is a wizard: session/network setup -> game mode ->
	// playlist -> player slots/character. Do not force the host through the old
	// standalone lobby before the match has even been configured.
	g.customPage = customPageGame
	g.customMenuX = customPageTargetX(customPageGame)
	g.screen = screenCustomGame
	g.beginFade(screenCustomGame, fadePurposeScreen)
}
