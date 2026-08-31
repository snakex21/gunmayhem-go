package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

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
		g.customHostScope = onlineSessionScopeLAN
		g.multiplayerMessage = "INTERNET MULTIPLAYER IS DISABLED IN THIS RELEASE - LAN ONLY"
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
	scope := onlineSessionScopeLAN
	g.customHostScope = scope
	access := g.customHostAccess
	if access != onlineSessionOpen && access != onlineSessionInviteOnly {
		access = onlineSessionOpen
		g.customHostAccess = access
	}
	if err := g.StartNetHostConfig(":7777", scope, access); err != nil {
		g.multiplayerMessage = "HOST ERROR: " + err.Error()
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

func drawHostTypeCard(screen *ebiten.Image, r Rect, title, subtitle string, selected bool) {
	border := color.NRGBA{R: 0x46, G: 0x4b, B: 0x53, A: 0xff}
	fill := color.NRGBA{R: 0x29, G: 0x2d, B: 0x34, A: 0xff}
	accent := color.NRGBA{R: 0x7d, G: 0x84, B: 0x8e, A: 0xff}
	if selected {
		border = color.NRGBA{R: 0xff, G: 0x91, B: 0x00, A: 0xff}
		fill = color.NRGBA{R: 0x38, G: 0x30, B: 0x25, A: 0xff}
		accent = color.NRGBA{R: 0xff, G: 0x91, B: 0x00, A: 0xff}
	}
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, border)
	ebitenutil.DrawRect(screen, r.X+3, r.Y+3, r.W-6, r.H-6, fill)
	ebitenutil.DrawRect(screen, r.X+3, r.Y+3, r.W-6, 5, accent)
	drawSourceMenuText(screen, title, menuFontCondensedExtraBold, 17, color.White, r.X+14, r.Y+16)
	drawSourceMenuText(screen, subtitle, menuFontCondensed, 11.5, color.NRGBA{R: 0xc8, G: 0xcc, B: 0xd3, A: 0xff}, r.X+14, r.Y+47)
}

func (g *Game) drawCustomHostSetupPanel(screen *ebiten.Image) {
	editingExisting := g.netplay != nil && g.netplay.mode == netplayHost
	ebitenutil.DrawRect(screen, 0, 0, 900, 600, color.NRGBA{R: 0x18, G: 0x1c, B: 0x22, A: 0xff})
	ebitenutil.DrawRect(screen, 25, 20, 850, 70, color.NRGBA{R: 0xf5, G: 0xf5, B: 0xf5, A: 0xff})
	title := "CREATE LAN SESSION"
	if editingExisting {
		title = "LAN SESSION SETTINGS"
	}
	drawSourceMenuText(screen, title, menuFontTwCen, 32, color.Black, 45, 32)

	drawSourceMenuText(screen, "SESSION NAME", menuFontCondensedExtraBold, 13, color.White, customHostNameRect.X, 116)
	field := color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}
	if g.multiplayerNameFocus {
		field = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	}
	ebitenutil.DrawRect(screen, customHostNameRect.X, customHostNameRect.Y, customHostNameRect.W, customHostNameRect.H, field)
	name := g.multiplayerSessionName
	if g.multiplayerNameFocus {
		name += "_"
	}
	drawSourceMenuText(screen, name, menuFontTwCen, 18, color.Black, customHostNameRect.X+10, customHostNameRect.Y+10)

	drawSourceMenuText(screen, "NETWORK REACH", menuFontCondensedExtraBold, 16, color.White, 180, 205)
	drawHostTypeCard(screen, customHostScopeLANRect, "LAN ONLY", "Players on your local network.", g.customHostScope == onlineSessionScopeLAN)
	drawHostTypeCard(screen, customHostScopeInternetRect, "INTERNET", "Disabled in this release.", false)

	drawSourceMenuText(screen, "ACCESS", menuFontCondensedExtraBold, 16, color.White, 180, 332)
	drawHostTypeCard(screen, customHostAccessOpenRect, "OPEN", "Visible in the selected scope.", g.customHostAccess == onlineSessionOpen)
	drawHostTypeCard(screen, customHostAccessInviteRect, "INVITE CODE", "Hidden; only players with code join.", g.customHostAccess == onlineSessionInviteOnly)

	drawSourceMenuText(screen, "This release supports LAN multiplayer only. OPEN and INVITE CODE both work inside the local network.", menuFontCondensed, 11.5,
		color.NRGBA{R: 0xb9, G: 0xbf, B: 0xc8, A: 0xff}, 180, 458)
	info := "Returning to the server browser closes the host but keeps these settings until the game is closed."
	backLabel := "BACK TO SERVERS"
	applyLabel := "CREATE SESSION"
	if editingExisting {
		info = "CANCEL keeps the host. LEAVE SESSION / ESC closes it and opens the server browser."
		backLabel = "CANCEL"
		applyLabel = "APPLY & RESTART"
	}
	drawSourceMenuText(screen, info, menuFontCondensed, 11,
		color.NRGBA{R: 0x9f, G: 0xa6, B: 0xb0, A: 0xff}, 180, 477)

	if editingExisting {
		drawDevelopedMenuButton(screen, customHostLeaveRect, "LEAVE SESSION")
		drawDevelopedMenuButton(screen, customHostCancelEditRect, backLabel)
		drawDevelopedMenuButton(screen, customHostApplyEditRect, applyLabel)
	} else {
		drawDevelopedMenuButton(screen, customHostBackRect, backLabel)
		drawDevelopedMenuButton(screen, customHostCreateRect, applyLabel)
	}
}
