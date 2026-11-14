package game

import ()

const (
	menuHitCustomHostBack = 9950 + iota
	menuHitCustomHostCreate
	menuHitCustomHostScopeLAN
	menuHitCustomHostScopeInternet
	menuHitCustomHostAccessOpen
	menuHitCustomHostAccessInvite
	menuHitCustomHostLeave
)

var (
	customHostNameRect          = Rect{X: 180, Y: 142, W: 540, H: 42}
	customHostScopeLANRect      = Rect{X: 180, Y: 238, W: 255, H: 78}
	customHostScopeInternetRect = Rect{X: 465, Y: 238, W: 255, H: 78}
	customHostAccessOpenRect    = Rect{X: 180, Y: 360, W: 255, H: 78}
	customHostAccessInviteRect  = Rect{X: 465, Y: 360, W: 255, H: 78}
	customHostBackRect          = Rect{X: 180, Y: 500, W: 250, H: 50}
	customHostCreateRect        = Rect{X: 470, Y: 500, W: 250, H: 50}
	customHostLeaveRect         = Rect{X: 80, Y: 500, W: 220, H: 50}
	customHostCancelEditRect    = Rect{X: 340, Y: 500, W: 220, H: 50}
	customHostApplyEditRect     = Rect{X: 600, Y: 500, W: 220, H: 50}
)

func (g *Game) customHostSetupHitAt(x, y float64) int {
	if g != nil && g.netplay != nil && g.netplay.mode == netplayHost {
		switch {
		case customHostLeaveRect.Contains(x, y):
			return menuHitCustomHostLeave
		case customHostCancelEditRect.Contains(x, y):
			return menuHitCustomHostBack
		case customHostApplyEditRect.Contains(x, y):
			return menuHitCustomHostCreate
		}
		return menuHitNone
	}
	switch {
	case customHostBackRect.Contains(x, y):
		return menuHitCustomHostBack
	case customHostCreateRect.Contains(x, y):
		return menuHitCustomHostCreate
	case customHostScopeLANRect.Contains(x, y):
		return menuHitCustomHostScopeLAN
	case customHostScopeInternetRect.Contains(x, y):
		return menuHitCustomHostScopeInternet
	case customHostAccessOpenRect.Contains(x, y):
		return menuHitCustomHostAccessOpen
	case customHostAccessInviteRect.Contains(x, y):
		return menuHitCustomHostAccessInvite
	default:
		return menuHitNone
	}
}

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
