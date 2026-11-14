package game

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	menuHitMultiplayerLocal = 6000 + iota
	menuHitMultiplayerHost
	menuHitMultiplayerRefresh
	menuHitMultiplayerDirectJoin
	menuHitMultiplayerBack
)

const (
	menuHitMultiplayerServerBase = 6100
	multiplayerServerVisibleRows = 5
	multiplayerServerRowH        = 30.0
)

var (
	multiplayerLocalRect      = Rect{X: 180, Y: 132, W: 250, H: 52}
	multiplayerHostRect       = Rect{X: 470, Y: 132, W: 250, H: 52}
	multiplayerRefreshRect    = Rect{X: 550, Y: 180, W: 170, H: 44}
	multiplayerServerListRect = Rect{X: 180, Y: 230, W: 540, H: multiplayerServerRowH * multiplayerServerVisibleRows}
	multiplayerAddressRect    = Rect{X: 180, Y: 414, W: 355, H: 44}
	multiplayerDirectJoinRect = Rect{X: 550, Y: 414, W: 170, H: 44}
	multiplayerBackRect       = Rect{X: 180, Y: 492, W: 540, H: 50}
)

func multiplayerMenuHitAt(x, y float64) int {
	switch {
	case multiplayerLocalRect.Contains(x, y):
		return menuHitMultiplayerLocal
	case multiplayerHostRect.Contains(x, y):
		return menuHitMultiplayerHost
	case multiplayerRefreshRect.Contains(x, y):
		return menuHitMultiplayerRefresh
	case multiplayerDirectJoinRect.Contains(x, y):
		return menuHitMultiplayerDirectJoin
	case multiplayerBackRect.Contains(x, y):
		return menuHitMultiplayerBack
	default:
		return menuHitNone
	}
}

func validNetAddressRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(".:-_[]", r)
}

func (g *Game) updateMultiplayerSessionNameInput() {
	if !g.multiplayerNameFocus || g.netplay != nil {
		return
	}
	if textEraseRepeated() {
		r := []rune(g.multiplayerSessionName)
		if len(r) > 0 {
			g.multiplayerSessionName = string(r[:len(r)-1])
		}
	}
	for _, ch := range ebiten.AppendInputChars(nil) {
		if len([]rune(g.multiplayerSessionName)) >= 24 || ch < 32 || ch == 127 {
			continue
		}
		g.multiplayerSessionName += string(ch)
	}
}

func (g *Game) updateMultiplayerAddressInput() {
	// Ctrl+V / Shift+Insert works whenever the join field is on screen; long
	// GMT1 invite codes are impractical to type.
	if isPasteShortcutJustPressed() {
		g.pasteIntoMultiplayerAddress()
		return
	}
	if !g.multiplayerFocus {
		return
	}
	if textEraseRepeated() {
		r := []rune(g.multiplayerAddress)
		if len(r) > 0 {
			g.multiplayerAddress = string(r[:len(r)-1])
		}
	}
	for _, ch := range ebiten.AppendInputChars(nil) {
		if !validNetAddressRune(ch) || len([]rune(g.multiplayerAddress)) >= cloudflareTunnelURLMaxLen {
			continue
		}
		g.multiplayerAddress += string(ch)
	}
}

func (g *Game) maxMultiplayerServerScroll() int {
	max := len(g.multiplayerServers) - multiplayerServerVisibleRows
	if max < 0 {
		return 0
	}
	return max
}

func (g *Game) multiplayerServerHitAtRect(listRect Rect, x, y float64) int {
	if !listRect.Contains(x, y) {
		return menuHitNone
	}
	row := int((y - listRect.Y) / multiplayerServerRowH)
	if row < 0 || row >= multiplayerServerVisibleRows {
		return menuHitNone
	}
	index := g.multiplayerServerScroll + row
	if index < 0 || index >= len(g.multiplayerServers) {
		return menuHitNone
	}
	return menuHitMultiplayerServerBase + index
}

func (g *Game) multiplayerServerHitAt(x, y float64) int {
	return g.multiplayerServerHitAtRect(multiplayerServerListRect, x, y)
}

func (g *Game) updateMultiplayerServerScroll(x, y float64) {
	if !multiplayerServerListRect.Contains(x, y) {
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
	if g.multiplayerServerScroll < 0 {
		g.multiplayerServerScroll = 0
	}
	if max := g.maxMultiplayerServerScroll(); g.multiplayerServerScroll > max {
		g.multiplayerServerScroll = max
	}
}

func (g *Game) updateMultiplayerMenu() {
	g.pollMultiplayerServerBrowser()
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		_ = g.CloseNetplay()
		g.multiplayerFocus = false
		g.menuPressed = menuHitNone
		g.beginFade(screenMainMenu, fadePurposeScreen)
		return
	}

	mx, my := g.cursorPos()
	x, y := float64(mx), float64(my)
	g.updateMultiplayerServerScroll(x, y)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.multiplayerFocus = multiplayerAddressRect.Contains(x, y)
	}
	g.updateMultiplayerAddressInput()

	hit := multiplayerMenuHitAt(x, y)
	if hit == menuHitNone {
		hit = g.multiplayerServerHitAt(x, y)
	}
	activated, ok := g.resolveMenuRelease(hit)
	if !ok {
		if g.multiplayerFocus && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.joinFromMultiplayerMenu()
		}
		return
	}

	switch {
	case activated == menuHitMultiplayerLocal:
		_ = g.CloseNetplay()
		g.multiplayerFocus = false
		g.multiplayerMessage = "LOCAL GAME"
		g.customPage = customPageGame
		g.customMenuX = 1800
		g.beginFade(screenCustomGame, fadePurposeScreen)
	case activated == menuHitMultiplayerHost:
		if err := g.StartNetHost(":7777"); err != nil {
			g.reportNetError("HOST ERROR", err)
			return
		}
		g.multiplayerFocus = false
		g.clearDisconnectedOnlineHumanSlots()
		g.openOnlineCustomSlots("HOSTING - WAITING FOR PLAYER / ADD AI")
	case activated == menuHitMultiplayerRefresh:
		g.refreshMultiplayerServers()
	case activated == menuHitMultiplayerDirectJoin:
		g.joinFromMultiplayerMenu()
	case activated == menuHitMultiplayerBack:
		_ = g.CloseNetplay()
		g.multiplayerFocus = false
		g.beginFade(screenMainMenu, fadePurposeScreen)
	case activated >= menuHitMultiplayerServerBase:
		index := activated - menuHitMultiplayerServerBase
		if index < 0 || index >= len(g.multiplayerServers) {
			return
		}
		server := g.multiplayerServers[index]
		if server.Full() {
			g.multiplayerMessage = "SERVER FULL: " + server.Name
			return
		}
		g.multiplayerAddress = server.Address
		g.beginAsyncNetJoin(server.Name, func(scratch *Game) error {
			return scratch.StartNetClientServer(server)
		}, func() {
			g.multiplayerFocus = false
			g.openOnlineCustomSlots("CONNECTED - CHOOSE YOUR CHARACTER")
		})
	}
}

func (g *Game) joinFromMultiplayerMenu() {
	address := strings.TrimSpace(g.multiplayerAddress)
	if address == "" {
		g.multiplayerMessage = "ENTER A JOIN CODE OR IP:PORT"
		return
	}
	g.beginAsyncNetJoin(address, func(scratch *Game) error {
		return scratch.startNetClientFromInput(address)
	}, func() {
		g.multiplayerFocus = false
		g.openOnlineCustomSlots("CONNECTED - CHOOSE YOUR CHARACTER")
	})
}

func drawDevelopedMenuButton(screen *ebiten.Image, r Rect, label string) {
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, color.NRGBA{R: 0x24, G: 0x27, B: 0x2d, A: 0xee})
	ebitenutil.DrawRect(screen, r.X+2, r.Y+2, r.W-4, r.H-4, color.NRGBA{R: 0xf2, G: 0x8c, B: 0x00, A: 0xff})
	drawSourceMenuText(screen, label, menuFontTwCen, 22, color.NRGBA{A: 0xff}, r.X+14, r.Y+10)
}

func (g *Game) drawMultiplayerServerList(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, multiplayerServerListRect.X, multiplayerServerListRect.Y, multiplayerServerListRect.W, multiplayerServerListRect.H,
		color.NRGBA{R: 0x24, G: 0x27, B: 0x2d, A: 0xff})
	if len(g.multiplayerServers) == 0 {
		label := "SEARCHING..."
		if g.multiplayerDiscoveryDone && g.multiplayerDiscoveryCh == nil {
			label = "NO OPEN SERVERS - USE JOIN CODE"
		}
		drawSourceMenuText(screen, label, menuFontTwCen, 18, color.NRGBA{R: 0xbb, G: 0xbb, B: 0xbb, A: 0xff},
			multiplayerServerListRect.X+15, multiplayerServerListRect.Y+58)
		return
	}

	for row := 0; row < multiplayerServerVisibleRows; row++ {
		index := g.multiplayerServerScroll + row
		if index >= len(g.multiplayerServers) {
			break
		}
		server := g.multiplayerServers[index]
		y := multiplayerServerListRect.Y + float64(row)*multiplayerServerRowH
		c := color.NRGBA{R: 0x36, G: 0x3b, B: 0x44, A: 0xff}
		if server.Full() {
			c = color.NRGBA{R: 0x45, G: 0x2f, B: 0x2f, A: 0xff}
		}
		ebitenutil.DrawRect(screen, multiplayerServerListRect.X+2, y+1, multiplayerServerListRect.W-4, multiplayerServerRowH-2, c)
		drawSourceMenuText(screen, server.Name, menuFontTwCen, 16, color.NRGBA{A: 0xff}, multiplayerServerListRect.X+10, y+6)
		source := strings.TrimSpace(server.Sources)
		if source == "" {
			source = "NETWORK"
		}
		drawSourceMenuText(screen, source, menuFontCondensedExtraBold, 12.5, color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}, multiplayerServerListRect.X+300, y+8)
		players := "?/" + strconv.Itoa(server.MaxPlayers)
		if server.Players >= 0 {
			players = strconv.Itoa(server.Players) + "/" + strconv.Itoa(server.MaxPlayers)
		}
		drawSourceMenuText(screen, players, menuFontCondensedExtraBold, 14, color.NRGBA{A: 0xff}, multiplayerServerListRect.X+490, y+7)
	}
}

func (g *Game) drawMultiplayerMenu(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, 140, 55, 620, 510, color.NRGBA{R: 0x18, G: 0x1c, B: 0x22, A: 0xf2})
	drawSourceMenuText(screen, "MULTIPLAYER", menuFontTwCen, 38, color.NRGBA{A: 0xff}, 180, 72)
	drawSourceMenuText(screen, "LOCAL OR PLAYER-HOSTED NETWORK GAME", menuFontCondensed, 15, color.NRGBA{A: 0xff}, 180, 112)

	drawDevelopedMenuButton(screen, multiplayerLocalRect, "LOCAL GAME")
	drawDevelopedMenuButton(screen, multiplayerHostRect, "HOST GAME")

	drawSourceMenuText(screen, "SERVER LIST", menuFontCondensedExtraBold, 17, color.NRGBA{A: 0xff}, 180, 199)
	drawDevelopedMenuButton(screen, multiplayerRefreshRect, "REFRESH")
	g.drawMultiplayerServerList(screen)

	drawSourceMenuText(screen, "DIRECT JOIN - IP:PORT", menuFontCondensedExtraBold, 16, color.NRGBA{A: 0xff}, 180, 389)
	fieldColor := color.NRGBA{R: 0x38, G: 0x3e, B: 0x48, A: 0xff}
	if g.multiplayerFocus {
		fieldColor = color.NRGBA{R: 0x4b, G: 0x54, B: 0x61, A: 0xff}
	}
	ebitenutil.DrawRect(screen, multiplayerAddressRect.X, multiplayerAddressRect.Y, multiplayerAddressRect.W, multiplayerAddressRect.H, fieldColor)
	address := g.multiplayerAddress
	if g.multiplayerFocus {
		address += "_"
	}
	drawSourceMenuText(screen, address, menuFontTwCen, 17, color.NRGBA{A: 0xff}, multiplayerAddressRect.X+10, multiplayerAddressRect.Y+11)
	drawDevelopedMenuButton(screen, multiplayerDirectJoinRect, "JOIN")
	drawDevelopedMenuButton(screen, multiplayerBackRect, "BACK")

	status := g.multiplayerMessage
	if g.netplay != nil {
		status = g.NetplayStatus()
	}
	if status != "" {
		// Full text, wrapped (errors used to be cut off). Extra lines get a
		// backdrop because they extend below the panel.
		var backdrop color.Color
		if wrappedLineCount(status, menuFontCondensed, 13.5, 540) > 1 {
			backdrop = color.NRGBA{R: 0x18, G: 0x1c, B: 0x22, A: 0xf2}
		}
		drawWrappedMenuText(screen, status, menuFontCondensed, 13.5, color.NRGBA{A: 0xff}, 180, 548, 540, 16, 3, backdrop)
	}
}
