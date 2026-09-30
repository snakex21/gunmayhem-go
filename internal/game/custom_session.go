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
	menuHitCustomSessionLocal = 9700 + iota
	menuHitCustomSessionOnline
	menuHitCustomSessionNetwork
	menuHitCustomSessionRules
	menuHitCustomNetworkHostOpen
	menuHitCustomNetworkHostInvite
	menuHitCustomNetworkRefresh
	menuHitCustomNetworkJoin
	menuHitCustomNetworkDisconnect
	menuHitCustomNetworkClose
	menuHitCustomNetworkMainMenu
	menuHitCustomNetworkCreateSetup
	menuHitCustomNetworkJoinSelected
)

const menuHitCustomNetworkServerBase = 9800

var (
	customSessionLocalRect        = Rect{X: 646, Y: 16, W: 96, H: 30}
	customSessionOnlineRect       = Rect{X: 748, Y: 16, W: 106, H: 30}
	customSessionRulesRect        = Rect{X: 760, Y: 45, W: 105, H: 28}
	customSessionNetworkRect      = Rect{X: 646, Y: 50, W: 208, H: 28}
	customNetworkCreateRect       = Rect{X: 55, Y: 135, W: 300, H: 45}
	customNetworkRefreshRect      = Rect{X: 370, Y: 135, W: 150, H: 45}
	customNetworkServerListRect   = Rect{X: 55, Y: 205, W: 385, H: 230}
	customNetworkPreviewRect      = Rect{X: 455, Y: 205, W: 390, H: 230}
	customNetworkJoinSelectedRect = Rect{X: 620, Y: 392, W: 205, H: 34}
	customNetworkAddressRect      = Rect{X: 55, Y: 468, W: 470, H: 40}
	customNetworkDirectJoinRect   = Rect{X: 540, Y: 468, W: 140, H: 40}
	customNetworkMainMenuRect     = Rect{X: 55, Y: 530, W: 250, H: 45}
)

func (g *Game) customSessionHitAt(x, y float64) int {
	// LOCAL/ONLINE toggles exist on the local setup and browser, but not as
	// invisible hot-zones inside the sequential online wizard.
	if g.customNetworkOpen || !g.customOnlineMode {
		if customSessionLocalRect.Contains(x, y) {
			return menuHitCustomSessionLocal
		}
		if customSessionOnlineRect.Contains(x, y) {
			return menuHitCustomSessionOnline
		}
	}
	if g.customOnlineMode && !g.customNetworkOpen && g.netplay != nil && g.netplay.mode == netplayHost && customSessionRulesRect.Contains(x, y) {
		return menuHitCustomSessionRules
	}
	return menuHitNone
}

func (g *Game) customBrowserServerHitAt(x, y float64) int {
	if !customNetworkServerListRect.Contains(x, y) {
		return menuHitNone
	}
	const rowH = 34.0
	row := int((y - customNetworkServerListRect.Y) / rowH)
	if row < 0 || row >= 6 {
		return menuHitNone
	}
	index := g.multiplayerServerScroll + row
	if index < 0 || index >= len(g.multiplayerServers) {
		return menuHitNone
	}
	return menuHitCustomNetworkServerBase + index
}

func (g *Game) updateCustomBrowserServerScroll(x, y float64) {
	if !customNetworkServerListRect.Contains(x, y) {
		return
	}
	_, wheelY := ebiten.Wheel()
	if wheelY == 0 {
		return
	}
	if wheelY > 0 {
		g.multiplayerServerScroll--
	} else {
		g.multiplayerServerScroll++
	}
	maxScroll := len(g.multiplayerServers) - 6
	if maxScroll < 0 {
		maxScroll = 0
	}
	if g.multiplayerServerScroll < 0 {
		g.multiplayerServerScroll = 0
	}
	if g.multiplayerServerScroll > maxScroll {
		g.multiplayerServerScroll = maxScroll
	}
}

func (g *Game) customNetworkHitAt(x, y float64) int {
	if customNetworkMainMenuRect.Contains(x, y) {
		return menuHitCustomNetworkMainMenu
	}
	if g.netplay == nil && customNetworkCreateRect.Contains(x, y) {
		return menuHitCustomNetworkCreateSetup
	}
	if g.netplay == nil && customNetworkRefreshRect.Contains(x, y) {
		return menuHitCustomNetworkRefresh
	}
	if customNetworkDirectJoinRect.Contains(x, y) {
		return menuHitCustomNetworkJoin
	}
	if g.multiplayerSelectedServer >= 0 && g.multiplayerSelectedServer < len(g.multiplayerServers) && customNetworkJoinSelectedRect.Contains(x, y) {
		return menuHitCustomNetworkJoinSelected
	}
	if hit := g.customBrowserServerHitAt(x, y); hit >= menuHitCustomNetworkServerBase {
		return hit
	}
	return menuHitNone
}

func (g *Game) updateCustomSessionInput(x, y float64) bool {
	if g.customOnlineMode {
		g.pollMultiplayerServerBrowser()
	}
	if g.customNetworkOpen {
		if g.customHostSetupOpen {
			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				g.multiplayerNameFocus = customHostNameRect.Contains(x, y)
				g.multiplayerFocus = false
			}
			g.updateMultiplayerSessionNameInput()
			hit := g.customHostSetupHitAt(x, y)
			activated, ok := g.resolveMenuRelease(hit)
			if !ok {
				return true
			}
			g.activateCustomHostSetupHit(activated)
			return true
		}

		g.multiplayerNameFocus = false
		g.updateCustomBrowserServerScroll(x, y)
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.multiplayerFocus = customNetworkAddressRect.Contains(x, y)
		}
		g.updateMultiplayerAddressInput()

		// LOCAL / ONLINE remain real controls on the full network page.
		if sessionHit := g.customSessionHitAt(x, y); sessionHit == menuHitCustomSessionLocal || sessionHit == menuHitCustomSessionOnline {
			activated, ok := g.resolveMenuRelease(sessionHit)
			if !ok {
				return true
			}
			switch activated {
			case menuHitCustomSessionLocal:
				_ = g.CloseNetplay()
				g.customOnlineMode = false
				g.customNetworkOpen = false
				g.customHostSetupOpen = false
				g.onlineMatchSetup = false
				g.multiplayerFocus = false
				g.multiplayerMessage = "LOCAL CUSTOM GAME"
			case menuHitCustomSessionOnline:
				g.customOnlineMode = true
			}
			return true
		}

		hit := g.customNetworkHitAt(x, y)
		activated, ok := g.resolveMenuRelease(hit)
		if !ok {
			return true
		}
		g.activateCustomNetworkHit(activated)
		return true
	}

	hit := g.customSessionHitAt(x, y)
	if hit == menuHitNone {
		return false
	}
	activated, ok := g.resolveMenuRelease(hit)
	if !ok {
		return true
	}
	switch activated {
	case menuHitCustomSessionLocal:
		_ = g.CloseNetplay()
		g.customOnlineMode = false
		g.customNetworkOpen = false
		g.onlineMatchSetup = false
		g.multiplayerMessage = "LOCAL CUSTOM GAME"
	case menuHitCustomSessionOnline:
		// ONLINE starts from the server browser/session creation flow. A host then
		// configures the match sequentially: session -> game mode -> maps -> slots.
		g.customOnlineMode = true
		g.returnToOnlineServerBrowser()
	case menuHitCustomSessionNetwork:
		// While a session exists the status bar is a route back to the lobby,
		// never a destructive leave. Without a session it opens the server
		// browser/create flow.
		if g.netplay != nil {
			g.returnToOnlineLobbyFromCustomSetup()
		} else {
			g.returnToOnlineServerBrowser()
		}
	case menuHitCustomSessionRules:
		if g.netplay != nil && g.netplay.mode == netplayHost {
			g.onlineRulesOpen = true
		}
	}
	return true
}

func (g *Game) returnToOnlineServerBrowser() {
	if g == nil {
		return
	}
	_ = g.CloseNetplay()
	g.customOnlineMode = true
	g.onlineMatchSetup = false
	g.customNetworkOpen = true
	g.customHostSetupOpen = false
	g.multiplayerFocus = false
	g.multiplayerNameFocus = false
	g.multiplayerServers = nil
	g.multiplayerServerScroll = 0
	g.multiplayerSelectedServer = -1
	// Invalidate discovery that may still contain the host being closed.
	g.multiplayerDiscoveryCh = nil
	g.multiplayerDiscoveryDone = false
	g.multiplayerMessage = "CHOOSE A SESSION OR CREATE ONE"
	g.beginMultiplayerServerBrowser()
}

func (g *Game) leaveCustomNetworkPanel() {
	if g == nil {
		return
	}
	_ = g.CloseNetplay()
	g.onlineMatchSetup = false
	g.customNetworkOpen = false
	g.customHostSetupOpen = false
	g.multiplayerFocus = false
	g.multiplayerNameFocus = false
	g.multiplayerServers = nil
	g.multiplayerServerScroll = 0
	g.multiplayerSelectedServer = -1
	g.multiplayerDiscoveryCh = nil
	g.multiplayerDiscoveryDone = false
	g.multiplayerMessage = "ONLINE SETTINGS KEPT"
}

func (g *Game) activateCustomNetworkHit(hit int) {
	switch {
	case hit == menuHitCustomNetworkCreateSetup:
		g.customHostSetupOpen = true
		g.multiplayerFocus = false
		g.multiplayerNameFocus = false
		g.multiplayerMessage = "CONFIGURE THE NEW SESSION"
	case hit == menuHitCustomNetworkRefresh:
		g.multiplayerSelectedServer = -1
		g.refreshMultiplayerServers()
	case hit == menuHitCustomNetworkJoin:
		g.joinFromCustomSession()
	case hit == menuHitCustomNetworkJoinSelected:
		g.joinSelectedCustomServer()
	case hit == menuHitCustomNetworkClose:
		g.leaveCustomNetworkPanel()
	case hit == menuHitCustomNetworkMainMenu:
		_ = g.CloseNetplay()
		g.customOnlineMode = false
		g.customNetworkOpen = false
		g.customHostSetupOpen = false
		g.onlineMatchSetup = false
		g.multiplayerFocus = false
		g.multiplayerNameFocus = false
		g.menuPressed = menuHitNone
		g.beginFade(screenMainMenu, fadePurposeScreen)
	case hit >= menuHitCustomNetworkServerBase:
		index := hit - menuHitCustomNetworkServerBase
		if index < 0 || index >= len(g.multiplayerServers) {
			return
		}
		g.multiplayerSelectedServer = index
		g.multiplayerFocus = false
		server := g.multiplayerServers[index]
		g.multiplayerMessage = "SELECTED: " + server.Name
	}
}

func (g *Game) joinSelectedCustomServer() {
	if g == nil || g.multiplayerSelectedServer < 0 || g.multiplayerSelectedServer >= len(g.multiplayerServers) {
		return
	}
	server := g.multiplayerServers[g.multiplayerSelectedServer]
	if server.Full() {
		g.multiplayerMessage = "SERVER FULL: " + server.Name
		return
	}
	g.multiplayerAddress = server.Address
	if err := g.StartNetClientServer(server); err != nil {
		g.multiplayerMessage = "JOIN ERROR: " + err.Error()
		return
	}
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.customNetworkOpen = false
	g.customHostSetupOpen = false
	g.multiplayerFocus = false
	// A client joins the final wizard step. Host-owned MATCH/MAPS are visible
	// through the browser preview and synchronized snapshots; the client mainly
	// chooses their own character in the SLOTS step.
	g.setCustomV2Page(customPagePlayers)
	g.multiplayerMessage = "CONNECTED - CHOOSE YOUR CHARACTER"
	g.screen = screenCustomGame
}

func (g *Game) joinFromCustomSession() {
	address := strings.TrimSpace(g.multiplayerAddress)
	if address == "" {
		g.multiplayerMessage = "ENTER A JOIN CODE OR IP:PORT"
		return
	}
	var err error
	upperAddress := strings.ToUpper(address)
	if strings.HasPrefix(upperAddress, cloudflareInviteCodePrefix) {
		err = g.StartNetClientCloudflareInvite(address)
	} else if strings.HasPrefix(upperAddress, inviteCodePrefix) {
		err = g.StartNetClientInvite(address)
	} else if strings.HasPrefix(upperAddress, shortOpenCodePrefix) {
		err = g.StartNetClientShortOpenCode(address)
	} else {
		err = g.StartNetClient(address)
	}
	if err != nil {
		g.multiplayerMessage = "JOIN ERROR: " + err.Error()
		return
	}
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.customNetworkOpen = false
	g.customHostSetupOpen = false
	g.multiplayerFocus = false
	g.setCustomV2Page(customPagePlayers)
	g.multiplayerMessage = "CONNECTED - CHOOSE YOUR CHARACTER"
	g.screen = screenCustomGame
}

func (g *Game) configuredOnlineBotCount() int {
	count := 0
	for slot := 1; slot < len(g.customPlayers); slot++ {
		if g.customPlayers[slot].Type == 2 {
			count++
		}
	}
	return count
}

func (g *Game) configuredOnlineHumanCapacity() int {
	capacity := 1
	for slot := 1; slot < len(g.customPlayers); slot++ {
		if g.customPlayers[slot].Type == 1 {
			capacity++
		}
	}
	return capacity
}

func (g *Game) customSessionStatus() string {
	if !g.customOnlineMode {
		return "LOCAL"
	}
	if g.netplay == nil {
		return "SERVER BROWSER - OFFLINE"
	}
	bots := g.configuredOnlineBotCount()
	capacity := g.configuredOnlineHumanCapacity()
	if g.netplay.mode == netplayHost {
		scope := "LAN"
		if g.onlineSessionScope == onlineSessionScopeInternet {
			scope = "WORLD"
		}
		access := "OPEN"
		if g.onlineSessionVisibility == onlineSessionInviteOnly {
			access = "INVITE"
		}
		return fmt.Sprintf("%s %s  H%d/%d  AI %d", scope, access, 1+g.netplay.connectedCount(), capacity, bots)
	}
	id := g.netplay.assignedPlayer()
	if id > 0 {
		connected := 0
		for _, present := range g.onlinePlayerSlots {
			if present {
				connected++
			}
		}
		if connected == 0 {
			connected = 1
		}
		return fmt.Sprintf("P%d  H%d/%d  AI %d", id, connected, capacity, bots)
	}
	return "ONLINE - CONNECTING"
}

func drawCustomSessionButton(dst *ebiten.Image, r Rect, label string, selected bool, online bool) {
	border := color.NRGBA{R: 0x24, G: 0x27, B: 0x2d, A: 0xee}
	fill := color.NRGBA{R: 0xdd, G: 0xdd, B: 0xdd, A: 0xff}
	text := color.NRGBA{A: 0xff}
	if selected {
		if online {
			fill = color.NRGBA{R: 0x00, G: 0x66, B: 0xff, A: 0xff}
			text = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
		} else {
			fill = color.NRGBA{R: 0xff, G: 0x99, A: 0xff}
		}
	}
	ebitenutil.DrawRect(dst, r.X, r.Y, r.W, r.H, border)
	ebitenutil.DrawRect(dst, r.X+2, r.Y+2, r.W-4, r.H-4, fill)
	drawSourceMenuText(dst, label, menuFontTwCen, 14.5, text, r.X+10, r.Y+6)
}

func (g *Game) drawCustomSessionControls(screen *ebiten.Image) {
	drawCustomSessionButton(screen, customSessionLocalRect, "LOCAL", !g.customOnlineMode, false)
	drawCustomSessionButton(screen, customSessionOnlineRect, "ONLINE", g.customOnlineMode, true)
	if g.customOnlineMode {
		if g.netplay != nil && g.netplay.mode == netplayHost {
			drawCustomSessionButton(screen, customSessionRulesRect, "RULES", false, true)
		}
		fill := color.NRGBA{R: 0x28, G: 0x32, B: 0x42, A: 0xf4}
		ebitenutil.DrawRect(screen, customSessionNetworkRect.X, customSessionNetworkRect.Y, customSessionNetworkRect.W, customSessionNetworkRect.H, fill)
		status := g.customSessionStatus()
		if len(status) > 27 {
			status = status[:27]
		}
		drawSourceMenuText(screen, status, menuFontCondensedExtraBold, 11.5, color.White, customSessionNetworkRect.X+8, customSessionNetworkRect.Y+6)
		if g.netplay != nil {
			mapSummary := onlineMapRotationLabel(g.onlineMapRotation)
			if g.customSeriesPlaylistConfigured {
				mapSummary = fmt.Sprintf("PLAYLIST %d ROUNDS", clampCustomSeriesRounds(g.customSeriesRounds))
			}
			summary := mapSummary + " | " + g.onlineRulesSummary()
			if len(summary) > 57 {
				summary = summary[:57]
			}
			ebitenutil.DrawRect(screen, 540, 81, 314, 17, color.NRGBA{R: 0x1d, G: 0x25, B: 0x32, A: 0xe8})
			drawSourceMenuText(screen, summary, menuFontCondensed, 9.5, color.White, 546, 84)
		}
	}
	if g.customNetworkOpen {
		g.drawCustomNetworkPanel(screen)
		return
	}
	if g.customOnlineRulesOverlayActive() {
		g.drawMultiplayerLobby(screen)
	}
}

func (g *Game) customOnlineRulesOverlayActive() bool {
	return g.onlineRulesOpen || g.onlineBlocklistOpen || g.onlinePerkBlocklistOpen || g.onlineVisualPicker != onlineVisualNone
}

func (g *Game) drawCustomBrowserServerList(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, customNetworkServerListRect.X, customNetworkServerListRect.Y, customNetworkServerListRect.W, customNetworkServerListRect.H,
		color.NRGBA{R: 0x22, G: 0x27, B: 0x30, A: 0xff})
	if len(g.multiplayerServers) == 0 {
		label := g.multiplayerServerListEmptyLabel()
		drawSourceMenuText(screen, label, menuFontTwCen, 18, color.NRGBA{R: 0xaa, G: 0xb0, B: 0xb8, A: 0xff}, customNetworkServerListRect.X+18, customNetworkServerListRect.Y+85)
		return
	}
	const rowH = 34.0
	for row := 0; row < 6; row++ {
		index := g.multiplayerServerScroll + row
		if index >= len(g.multiplayerServers) {
			break
		}
		server := g.multiplayerServers[index]
		y := customNetworkServerListRect.Y + float64(row)*rowH
		fill := color.NRGBA{R: 0x34, G: 0x3a, B: 0x45, A: 0xff}
		if server.Full() {
			fill = color.NRGBA{R: 0x49, G: 0x31, B: 0x34, A: 0xff}
		}
		if index == g.multiplayerSelectedServer {
			fill = color.NRGBA{R: 0x08, G: 0x63, B: 0xc9, A: 0xff}
		}
		ebitenutil.DrawRect(screen, customNetworkServerListRect.X+2, y+1, customNetworkServerListRect.W-4, rowH-2, fill)
		name := server.Name
		if len([]rune(name)) > 24 {
			name = string([]rune(name)[:24])
		}
		drawSourceMenuText(screen, name, menuFontTwCen, 14.5, color.White, customNetworkServerListRect.X+10, y+7)
		source := strings.TrimSpace(server.Sources)
		if source == "" {
			source = "NETWORK"
		}
		drawSourceMenuText(screen, source, menuFontCondensedExtraBold, 10.5, color.NRGBA{R: 0xc9, G: 0xd0, B: 0xda, A: 0xff}, customNetworkServerListRect.X+245, y+9)
		players := fmt.Sprintf("%d/%d", server.Players, server.MaxPlayers)
		if server.Players < 0 {
			players = fmt.Sprintf("?/%d", server.MaxPlayers)
		}
		drawSourceMenuText(screen, players, menuFontCondensedExtraBold, 12, color.White, customNetworkServerListRect.X+335, y+8)
	}
}

func (g *Game) drawCustomServerPreview(screen *ebiten.Image) {
	r := customNetworkPreviewRect
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, color.NRGBA{R: 0x22, G: 0x27, B: 0x30, A: 0xff})
	if g.multiplayerSelectedServer < 0 || g.multiplayerSelectedServer >= len(g.multiplayerServers) {
		drawSourceMenuText(screen, "SELECT A SERVER", menuFontCondensedExtraBold, 19, color.White, r.X+18, r.Y+18)
		drawSourceMenuText(screen, "Click a server to preview its maps and rules before joining.", menuFontCondensed, 11.5,
			color.NRGBA{R: 0xac, G: 0xb4, B: 0xbf, A: 0xff}, r.X+18, r.Y+52)
		return
	}
	server := g.multiplayerServers[g.multiplayerSelectedServer]
	name := server.Name
	if len([]rune(name)) > 30 {
		name = string([]rune(name)[:30])
	}
	drawSourceMenuText(screen, name, menuFontCondensedExtraBold, 19, color.White, r.X+16, r.Y+12)
	players := fmt.Sprintf("%d/%d", server.Players, server.MaxPlayers)
	if server.Players < 0 {
		players = fmt.Sprintf("?/%d", server.MaxPlayers)
	}
	drawSourceMenuText(screen, fmt.Sprintf("%s   PLAYERS %s", strings.TrimSpace(server.Sources), players), menuFontCondensedExtraBold, 10.5,
		color.NRGBA{R: 0x77, G: 0xb7, B: 0xff, A: 0xff}, r.X+16, r.Y+39)

	if !server.HasRulePreview() {
		label := "MATCH DETAILS UNAVAILABLE"
		detail := "The host did not provide preview metadata."
		if server.Players < 0 {
			label = "UNVERIFIED DHT SESSION"
			detail = "Join to verify the host and retrieve maps and rules."
		}
		drawSourceMenuText(screen, label, menuFontCondensedExtraBold, 14, color.NRGBA{R: 0xd2, G: 0xd6, B: 0xdc, A: 0xff}, r.X+16, r.Y+80)
		drawSourceMenuText(screen, detail, menuFontCondensed, 11, color.NRGBA{R: 0x9f, G: 0xa7, B: 0xb1, A: 0xff}, r.X+16, r.Y+108)
	} else {
		drawSourceMenuText(screen, developedModeTitle(server.GameMode), menuFontCondensedExtraBold, 15.5, color.White, r.X+16, r.Y+61)
		drawSourceMenuText(screen, fmt.Sprintf("LIVES %d   ROUNDS %d   CRATES %s   PICKUPS %s", server.Lives, server.Rounds, onOff(server.Crates), onOff(server.Pickups)), menuFontCondensed, 10.5,
			color.NRGBA{R: 0xca, G: 0xd0, B: 0xd8, A: 0xff}, r.X+16, r.Y+84)
		rules := strings.TrimSpace(server.RuleSummary)
		if rules == "" {
			rules = "NO MODIFIERS"
		}
		if len([]rune(rules)) > 46 {
			rules = string([]rune(rules)[:46])
		}
		drawSourceMenuText(screen, rules, menuFontCondensedExtraBold, 10.5, color.NRGBA{R: 0xff, G: 0xae, B: 0x3c, A: 0xff}, r.X+16, r.Y+102)

		thumbY := r.Y + 124
		shown := server.Rounds
		if shown > 3 {
			shown = 3
		}
		for i := 0; i < shown; i++ {
			mapValue := server.Maps[i]
			thumb := Rect{X: r.X + 16 + float64(i)*105, Y: thumbY, W: 95, H: 57}
			g.drawOnlineMapThumb(screen, mapValue, thumb)
			drawSourceMenuText(screen, fmt.Sprintf("R%d", i+1), menuFontCondensedExtraBold, 9.5, color.White, thumb.X+4, thumb.Y+42)
		}
		if server.Rounds > 3 {
			drawSourceMenuText(screen, fmt.Sprintf("+ %d MORE", server.Rounds-3), menuFontCondensedExtraBold, 10, color.NRGBA{R: 0xb5, G: 0xbc, B: 0xc6, A: 0xff}, r.X+332, thumbY+20)
		}
	}

	if server.Full() {
		ebitenutil.DrawRect(screen, customNetworkJoinSelectedRect.X, customNetworkJoinSelectedRect.Y, customNetworkJoinSelectedRect.W, customNetworkJoinSelectedRect.H, color.NRGBA{R: 0x4b, G: 0x3a, B: 0x3a, A: 0xff})
		drawSourceMenuText(screen, "SERVER FULL", menuFontTwCen, 14, color.NRGBA{R: 0xc8, G: 0xb5, B: 0xb5, A: 0xff}, customNetworkJoinSelectedRect.X+45, customNetworkJoinSelectedRect.Y+8)
	} else {
		ebitenutil.DrawRect(screen, customNetworkJoinSelectedRect.X, customNetworkJoinSelectedRect.Y, customNetworkJoinSelectedRect.W, customNetworkJoinSelectedRect.H, color.NRGBA{R: 0x06, G: 0x68, B: 0xe8, A: 0xff})
		drawSourceMenuText(screen, "JOIN SESSION", menuFontTwCen, 14.5, color.White, customNetworkJoinSelectedRect.X+34, customNetworkJoinSelectedRect.Y+8)
	}
}

func onOff(v bool) string {
	if v {
		return "ON"
	}
	return "OFF"
}

func (g *Game) drawCustomNetworkPanel(screen *ebiten.Image) {
	if g.customHostSetupOpen {
		g.drawCustomHostSetupPanel(screen)
		return
	}

	// Dedicated server browser. Session naming and visibility live on the
	// separate CREATE SESSION page so this screen stays focused on discovery.
	ebitenutil.DrawRect(screen, 0, 0, 900, 600, color.NRGBA{R: 0x18, G: 0x1c, B: 0x22, A: 0xff})
	ebitenutil.DrawRect(screen, 25, 20, 850, 70, color.NRGBA{R: 0xf5, G: 0xf5, B: 0xf5, A: 0xff})
	drawSourceMenuText(screen, "CUSTOM GAME", menuFontTwCen, 34, color.Black, 45, 30)
	tab, heading, addressLabel, placeholder := customNetworkLabels()
	drawCustomSessionButton(screen, customSessionLocalRect, "LOCAL", false, false)
	drawCustomSessionButton(screen, customSessionOnlineRect, tab, true, true)

	drawSourceMenuText(screen, heading, menuFontTwCen, 27, color.White, 55, 101)
	drawDevelopedMenuButton(screen, customNetworkCreateRect, "CREATE SESSION")
	drawDevelopedMenuButton(screen, customNetworkRefreshRect, "REFRESH")

	drawSourceMenuText(screen, "OPEN SESSIONS", menuFontCondensedExtraBold, 14, color.White, customNetworkServerListRect.X, 187)
	drawSourceMenuText(screen, "MAPS + RULES PREVIEW", menuFontCondensedExtraBold, 14, color.White, customNetworkPreviewRect.X, 187)
	g.drawCustomBrowserServerList(screen)
	g.drawCustomServerPreview(screen)

	drawSourceMenuText(screen, addressLabel, menuFontCondensedExtraBold, 12.5, color.White, customNetworkAddressRect.X, 448)
	ebitenutil.DrawRect(screen, customNetworkAddressRect.X, customNetworkAddressRect.Y, customNetworkAddressRect.W, customNetworkAddressRect.H, color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff})
	address := g.multiplayerAddress
	if address == "" {
		address = placeholder
	}
	if g.multiplayerFocus {
		address += "_"
	}
	drawSourceMenuText(screen, address, menuFontTwCen, 15, color.Black, customNetworkAddressRect.X+10, customNetworkAddressRect.Y+9)
	drawDevelopedMenuButton(screen, customNetworkDirectJoinRect, "JOIN")
	if g.multiplayerMessage != "" {
		msg := g.multiplayerMessage
		if len(msg) > 82 {
			msg = msg[:82]
		}
		drawSourceMenuText(screen, msg, menuFontCondensed, 10.5, color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}, 55, 512)
	}
	drawDevelopedMenuButton(screen, customNetworkMainMenuRect, "MAIN MENU")
}

func customPlayerEditorHitSlot(hit int) (int, bool) {
	if slot, _, ok := decodeCustomDevelopedGunHit(hit); ok {
		return slot, true
	}
	if hit >= menuHitPlayerThrowableBase && hit < menuHitPlayerThrowableBase+40 {
		return (hit - menuHitPlayerThrowableBase) / 10, true
	}
	if hit >= menuHitPlayerEditBase && hit < menuHitPlayerEditBase+40 {
		return (hit - menuHitPlayerEditBase) / 10, true
	}
	if hit >= menuHitPlayerEditorBackBase && hit < menuHitPlayerEditorBackBase+4 {
		return hit - menuHitPlayerEditorBackBase, true
	}
	if slot, _, _, ok := decodeCustomSelectHit(hit); ok {
		return slot, true
	}
	if hit >= menuHitPlayerHatPageBase && hit < menuHitPlayerHatPageBase+4 {
		return hit - menuHitPlayerHatPageBase, true
	}
	if hit >= menuHitPlayerTeamBase && hit < menuHitPlayerTeamBase+40 {
		return (hit - menuHitPlayerTeamBase) / 10, true
	}
	if hit >= menuHitPlayerColorBase && hit < menuHitPlayerColorBase+400 {
		return (hit - menuHitPlayerColorBase) / 100, true
	}
	if hit >= menuHitPlayerColorBase+500 && hit < menuHitPlayerColorBase+504 {
		return hit - (menuHitPlayerColorBase + 500), true
	}
	return 0, false
}

func (g *Game) onlineMayEditCustomCharacterSlot(slot int) bool {
	if g == nil || slot < 0 || slot >= len(g.customPlayers) {
		return false
	}
	if !g.customOnlineMode {
		return true
	}
	if g.netplay == nil {
		// Before a network session exists there is no remote character to edit.
		// Keep only the local player's character and AI bot loadouts editable.
		return slot == 0 || g.customPlayers[slot].Type == 2
	}
	switch g.netplay.mode {
	case netplayHost:
		// The host owns Player 1 and may configure AI bots. Human network slots
		// are owned by the corresponding clients, whether they are already
		// connected or merely waiting for a player.
		return slot == 0 || g.customPlayers[slot].Type == 2
	case netplayClient:
		return g.onlineLocalPlayerID() == slot+1
	default:
		return false
	}
}

func (g *Game) onlineHostMayEditCustomSlot(slot int) bool {
	return g.onlineMayEditCustomCharacterSlot(slot)
}

func (g *Game) removeOnlineCustomSlot(slot int) {
	if g == nil || slot <= 0 || slot >= len(g.customPlayers) || !g.customOnlineMode || g.netplay == nil || g.netplay.mode != netplayHost {
		return
	}
	kind := g.customPlayers[slot].Type
	if kind == 0 {
		return
	}
	wasConnected := g.onlinePlayerSlots[slot]
	g.customPlayers[slot].Type = 0
	g.customEditor[slot] = 0
	if g.customNameFocus == slot+1 {
		g.customNameFocus = 0
	}
	g.syncOnlineHostState()
	g.onlinePlayerSlots[slot] = false
	switch {
	case kind == 2:
		g.multiplayerMessage = fmt.Sprintf("BOT REMOVED FROM SLOT %d", slot+1)
	case wasConnected:
		g.multiplayerMessage = fmt.Sprintf("PLAYER %d KICKED", slot+1)
	default:
		g.multiplayerMessage = fmt.Sprintf("ONLINE SLOT %d CLOSED", slot+1)
	}
}

func (g *Game) updateOnlineClientCustomGame() {
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	mx, my := ebiten.CursorPosition()
	id := g.onlineLocalPlayerID()
	if id < 2 || id > maxOnlinePlayers {
		return
	}
	slot := id - 1

	if g.customPage == customPagePlayers {
		g.updateCustomPlayerCards()
		g.updateCustomNameInput()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.customPage == customPagePlayers && g.customEditor[slot] != 0 {
			g.customEditor[slot] = 0
			g.customNameFocus = 0
			return
		}
		_ = g.CloseNetplay()
		g.returnToOnlineServerBrowser()
		g.multiplayerMessage = "DISCONNECTED"
		return
	}

	hit := menuHitNone
	if customV2BackRect.Contains(float64(mx), float64(my)) {
		hit = menuHitCustomV2Back
	} else if g.customPage == customPagePlayers {
		// The original NAME field on the player's card is the only nickname
		// field; there is no second overlay competing with the color palette.
		hit = g.customPlayerEditorHitAt(float64(mx), float64(my))
		if target, ok := customPlayerEditorHitSlot(hit); !ok || target != slot {
			hit = menuHitNone
		}
	}
	activated, ok := g.resolveMenuRelease(hit)
	if !ok {
		return
	}
	if activated == menuHitCustomV2Back {
		_ = g.CloseNetplay()
		g.returnToOnlineServerBrowser()
		g.multiplayerMessage = "DISCONNECTED"
		return
	}
	if g.activateCustomPlayerEditorHit(activated) {
		g.customPlayers[slot].Type = 1
		g.sendOnlineLocalConfigIfChanged()
	}
}

func (g *Game) drawCustomOnlineSlotOverlay(screen *ebiten.Image) {
	if !g.customOnlineMode || g.customPage != customPagePlayers {
		return
	}
	localID := g.onlineLocalPlayerID()
	for slot, cfg := range g.customPlayers {
		cardX := g.customMenuX + 20 + float64(slot)*220
		cardY := g.customCardY[slot]
		label := ""
		switch {
		case slot == 0 && g.netplay != nil && g.netplay.mode == netplayHost:
			label = "HOST"
		case slot == 0 && cfg.Type == 1:
			label = "HOST - READ ONLY"
		case cfg.Type == 2:
			label = "AI BOT"
		case cfg.Type == 1 && slot > 0:
			if g.onlinePlayerSlots[slot] {
				if localID == slot+1 {
					label = "ONLINE"
				} else {
					label = "ONLINE - READ ONLY"
				}
			} else {
				label = "WAITING ONLINE"
			}
		case cfg.Type == 0:
			label = "EMPTY"
		}
		if localID == slot+1 {
			label += "  YOU"
		}
		if label != "" && cfg.Type != 0 {
			// Fully opaque: the old source CLEAR SLOT text underneath must not bleed
			// through the status badge (HOST / YOU / READ ONLY).
			ebitenutil.DrawRect(screen, cardX+18, cardY+7, 164, 22, color.NRGBA{R: 0x00, G: 0x66, B: 0xff, A: 0xff})
			drawSourceMenuText(screen, strings.TrimSpace(label), menuFontCondensedExtraBold, 11.5, color.White, cardX+25, cardY+11)
		}
		if cfg.Type == 0 && g.netplay != nil && g.netplay.mode == netplayHost && slot > 0 {
			// Cover the source HUMAN label so in online Custom Game it is clear
			// that the human button opens a network slot rather than a local keyboard player.
			r := sourceMenuHitRect(customPlayerTypeButtonLocal, cardX+20, cardY-254.95, 1.0775146484375, 1)
			ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff})
			drawSourceMenuText(screen, "ONLINE PLAYER", menuFontTwCen, 12, color.Black, r.X+12, r.Y+5)
		}
	}
}
