package game

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	menuHitMultiplayerHost = 6000 + iota
	menuHitMultiplayerJoin
	menuHitMultiplayerBack
)

var (
	multiplayerAddressRect = Rect{X: 225, Y: 188, W: 450, H: 46}
	multiplayerHostRect    = Rect{X: 225, Y: 264, W: 215, H: 55}
	multiplayerJoinRect    = Rect{X: 460, Y: 264, W: 215, H: 55}
	multiplayerBackRect    = Rect{X: 225, Y: 356, W: 450, H: 55}
	mainMultiplayerRect    = Rect{X: 572.05, Y: 205, W: 300, H: 40}
)

func multiplayerMenuHitAt(x, y float64) int {
	switch {
	case multiplayerHostRect.Contains(x, y):
		return menuHitMultiplayerHost
	case multiplayerJoinRect.Contains(x, y):
		return menuHitMultiplayerJoin
	case multiplayerBackRect.Contains(x, y):
		return menuHitMultiplayerBack
	default:
		return menuHitNone
	}
}

func validNetAddressRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(".:-_[]", r)
}

func (g *Game) updateMultiplayerAddressInput() {
	if !g.multiplayerFocus {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		r := []rune(g.multiplayerAddress)
		if len(r) > 0 {
			g.multiplayerAddress = string(r[:len(r)-1])
		}
	}
	for _, ch := range ebiten.AppendInputChars(nil) {
		if !validNetAddressRune(ch) || len([]rune(g.multiplayerAddress)) >= 80 {
			continue
		}
		g.multiplayerAddress += string(ch)
	}
}

func (g *Game) updateMultiplayerMenu() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		_ = g.CloseNetplay()
		g.multiplayerFocus = false
		g.menuPressed = menuHitNone
		g.beginFade(screenMainMenu, fadePurposeScreen)
		return
	}

	mx, my := ebiten.CursorPosition()
	x, y := float64(mx), float64(my)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.multiplayerFocus = multiplayerAddressRect.Contains(x, y)
	}
	g.updateMultiplayerAddressInput()

	hit := multiplayerMenuHitAt(x, y)
	activated, ok := g.resolveMenuRelease(hit)
	if !ok {
		if g.multiplayerFocus && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.joinFromMultiplayerMenu()
		}
		return
	}

	switch activated {
	case menuHitMultiplayerHost:
		_ = g.CloseNetplay()
		if err := g.StartNetHost(":7777"); err != nil {
			g.multiplayerMessage = "HOST ERROR: " + err.Error()
			return
		}
		// Network mode reserves Player 2 for the remote client. The host keeps
		// the normal Custom Game setup flow for map/mode/lives selection.
		g.customPlayers[1].Type = 1
		g.multiplayerMessage = "HOSTING ON TCP PORT 7777"
		g.customPage = customPageGame
		g.customMenuX = 1800
		g.beginFade(screenCustomGame, fadePurposeScreen)
	case menuHitMultiplayerJoin:
		g.joinFromMultiplayerMenu()
	case menuHitMultiplayerBack:
		_ = g.CloseNetplay()
		g.multiplayerFocus = false
		g.beginFade(screenMainMenu, fadePurposeScreen)
	}
}

func (g *Game) joinFromMultiplayerMenu() {
	_ = g.CloseNetplay()
	address := strings.TrimSpace(g.multiplayerAddress)
	if address == "" {
		address = "127.0.0.1:7777"
		g.multiplayerAddress = address
	}
	if err := g.StartNetClient(address); err != nil {
		g.multiplayerMessage = "JOIN ERROR: " + err.Error()
		return
	}
	g.multiplayerMessage = "CONNECTED - WAITING FOR HOST"
}

func drawDevelopedMenuButton(screen *ebiten.Image, r Rect, label string) {
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, color.NRGBA{R: 0x24, G: 0x27, B: 0x2d, A: 0xee})
	ebitenutil.DrawRect(screen, r.X+2, r.Y+2, r.W-4, r.H-4, color.NRGBA{R: 0xf2, G: 0x8c, B: 0x00, A: 0xff})
	drawSourceMenuText(screen, label, menuFontTwCen, 22, color.NRGBA{A: 0xff}, r.X+14, r.Y+10)
}

func (g *Game) drawMainMultiplayerButton(screen *ebiten.Image) {
	drawDevelopedMenuButton(screen, mainMultiplayerRect, "MULTIPLAYER")
}

func (g *Game) drawMultiplayerMenu(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, 175, 100, 550, 360, color.NRGBA{R: 0x18, G: 0x1c, B: 0x22, A: 0xee})
	drawSourceMenuText(screen, "MULTIPLAYER", menuFontTwCen, 38, color.NRGBA{A: 0xff}, 225, 120)
	drawSourceMenuText(screen, "HOST: TCP PORT 7777   /   JOIN: IP:PORT", menuFontCondensed, 16, color.NRGBA{A: 0xff}, 225, 166)

	fieldColor := color.NRGBA{R: 0x38, G: 0x3e, B: 0x48, A: 0xff}
	if g.multiplayerFocus {
		fieldColor = color.NRGBA{R: 0x4b, G: 0x54, B: 0x61, A: 0xff}
	}
	ebitenutil.DrawRect(screen, multiplayerAddressRect.X, multiplayerAddressRect.Y, multiplayerAddressRect.W, multiplayerAddressRect.H, fieldColor)
	address := g.multiplayerAddress
	if g.multiplayerFocus {
		address += "_"
	}
	drawSourceMenuText(screen, address, menuFontTwCen, 19, color.NRGBA{A: 0xff}, multiplayerAddressRect.X+12, multiplayerAddressRect.Y+12)

	drawDevelopedMenuButton(screen, multiplayerHostRect, "HOST GAME")
	drawDevelopedMenuButton(screen, multiplayerJoinRect, "JOIN GAME")
	drawDevelopedMenuButton(screen, multiplayerBackRect, "BACK")

	status := g.multiplayerMessage
	if g.netplay != nil {
		status = g.NetplayStatus()
	}
	if status != "" {
		drawSourceMenuText(screen, status, menuFontCondensed, 15, color.NRGBA{A: 0xff}, 225, 427)
	}
}
