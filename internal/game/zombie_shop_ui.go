package game

import (
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
)

// zombie_shop_ui.go: the shop panels (Mayhem UI kit). On a shared local
// screen every human gets an own panel (1 big, 2 side by side, 3-4 in a
// grid) driven by that player's own keys: UP/DOWN pick an item, LEFT/RIGHT
// switch the tab, SHOOT buys, GRENADE closes the shop (READY). The mouse
// works on every panel. Online each guest sees only its own panel; its
// purchases are requests the host validates (zombie_net.go).
//
// While the shop is open the player's game input is masked
// (updateZombiePreInput). During the break a closed shop reopens with the
// ABILITY key.

type zombieShopUI struct {
	st      uiState
	tab     int
	sel     int
	scroll  int
	prev    PlayerInputState
	wasOpen bool
	// holdKeys: the shop just closed; SHOOT / GRENADE / ABILITY stay masked
	// until released so the closing key does not also throw or dash.
	holdKeys bool
}

// holdingKeys reports (and ends) the post-close key hold.
func (ui *zombieShopUI) holdingKeys(cur PlayerInputState) bool {
	if ui == nil || !ui.holdKeys {
		return false
	}
	if cur.Shoot || cur.Grenade || cur.Ability {
		return true
	}
	ui.holdKeys = false
	return false
}

// zombieShopKeys are this frame's shop key edges of one player.
type zombieShopKeys struct {
	up, down, left, right, buy, close, open bool
}

func zombieShopEdges(cur, prev PlayerInputState) zombieShopKeys {
	return zombieShopKeys{
		up: cur.Up && !prev.Up, down: cur.Down && !prev.Down,
		left: cur.Left && !prev.Left, right: cur.Right && !prev.Right,
		buy: cur.Shoot && !prev.Shoot, close: cur.Grenade && !prev.Grenade,
		open: cur.Ability && !prev.Ability,
	}
}

// zombieShopRects lays out n panels.
func zombieShopRects(n int) []Rect {
	switch {
	case n <= 1:
		return []Rect{{X: 170, Y: 88, W: 560, H: 410}}
	case n == 2:
		return []Rect{{X: 14, Y: 88, W: 430, H: 410}, {X: 456, Y: 88, W: 430, H: 410}}
	}
	return []Rect{
		{X: 14, Y: 66, W: 430, H: 226}, {X: 456, Y: 66, W: 430, H: 226},
		{X: 14, Y: 300, W: 430, H: 226}, {X: 456, Y: 300, W: 430, H: 226},
	}
}

// zombieLocalShopPlayers: the humans whose shop is shown on this screen.
func (g *Game) zombieLocalShopPlayers() []*Player {
	m := g.zombie
	if m == nil {
		return nil
	}
	out := make([]*Player, 0, 4)
	if !m.host {
		if p := g.playerByID(g.onlineLocalPlayerID()); p != nil {
			out = append(out, p)
		}
		return out
	}
	for _, p := range g.players {
		if p == nil || p.IsDouble || p.AI || p.ID < 1 || p.ID > 4 {
			continue
		}
		if hp := m.hostPlayer(p.ID); hp != nil && hp.remote {
			continue
		}
		out = append(out, p)
	}
	return out
}

// zombieShopAction applies (local / host) or requests (guest) a shop action.
func (g *Game) zombieShopAction(p *Player, action, item int) {
	if g.zombie == nil || p == nil {
		return
	}
	if !g.zombie.host {
		g.queueZombieRequest(action, item)
		return
	}
	switch action {
	case netZombieReqBuy:
		g.zombieBuy(p, item)
	case netZombieReqClose:
		g.zombieSetShopOpen(p, false)
	case netZombieReqOpen:
		g.zombieSetShopOpen(p, true)
	}
}

// updateZombieShopUI runs the local panels (local match and the host's own
// player) once per gameplay tick.
func (g *Game) updateZombieShopUI() {
	if !g.zombieSimulating() {
		return
	}
	players := g.zombieLocalShopPlayers()
	if len(players) == 0 {
		return
	}
	pointer := g.readUIInput(&g.zombieUI).pointerOnly()
	rects := zombieShopRects(g.zombieOpenShopCount(players))
	slot := 0
	for _, p := range players {
		cur := capturePlayerInput(p.Controls)
		open := g.zombieShopIsOpen(p)
		r := rects[min(slot, len(rects)-1)]
		if open {
			slot++
		}
		g.runZombieShopPanel(p, cur, false, pointer, r)
	}
}

// updateZombieGuestShop runs a guest's own panel; true while it is open.
func (g *Game) updateZombieGuestShop() bool {
	m := g.zombie
	if m == nil || m.host {
		return false
	}
	p := g.playerByID(g.onlineLocalPlayerID())
	if p == nil {
		return false
	}
	cur := capturePlayerInput(g.controlConfigs[0])
	pointer := g.readUIInput(&g.zombieUI)
	esc := pointer.Esc
	pointer = pointer.pointerOnly()
	g.runZombieShopPanel(p, cur, esc, pointer, zombieShopRects(1)[0])
	return g.zombieShopIsOpen(p)
}

// zombieGuestInputMask clears the keys a guest's closed shop still holds.
func (g *Game) zombieGuestInputMask(in PlayerInputState) PlayerInputState {
	m := g.zombie
	if m == nil || m.host {
		return in
	}
	id := g.onlineLocalPlayerID()
	if id < 1 || id > 4 || !m.shop[id-1].holdingKeys(in) {
		return in
	}
	in.Shoot, in.Grenade, in.Ability = false, false, false
	return in
}

func (g *Game) zombieShopIsOpen(p *Player) bool {
	zp := g.zombie.playerState(p.ID)
	return zp != nil && zp.ShopOpen && zp.State == zombiePlayerAlive && g.zombie.hud.Phase == zombiePhaseBreak
}

func (g *Game) zombieOpenShopCount(players []*Player) int {
	n := 0
	for _, p := range players {
		if g.zombieShopIsOpen(p) {
			n++
		}
	}
	return n
}

func (g *Game) runZombieShopPanel(p *Player, cur PlayerInputState, esc bool, pointer uiInput, r Rect) {
	m := g.zombie
	ui := &m.shop[p.ID-1]
	zp := m.playerState(p.ID)
	if zp == nil {
		return
	}
	open := g.zombieShopIsOpen(p)
	if !open && ui.wasOpen {
		ui.holdKeys = true
	}
	if open && !ui.wasOpen {
		ui.prev = cur // keys held when the shop opened do not buy
		ui.st.reset()
	}
	keys := zombieShopEdges(cur, ui.prev)
	ui.prev = cur
	ui.wasOpen = open
	if !open {
		if keys.open && m.hud.Phase == zombiePhaseBreak && zp.State == zombiePlayerAlive {
			g.zombieShopAction(p, netZombieReqOpen, 0)
		}
		return
	}
	items := zombieShopTab(ui.tab)
	switch {
	case keys.left:
		ui.tab = (ui.tab + zombieTabCount - 1) % zombieTabCount
		ui.sel, ui.scroll = 0, 0
	case keys.right:
		ui.tab = (ui.tab + 1) % zombieTabCount
		ui.sel, ui.scroll = 0, 0
	case keys.up:
		ui.sel = max(0, ui.sel-1)
	case keys.down:
		ui.sel = min(len(items)-1, ui.sel+1)
	case keys.buy && ui.sel >= 0 && ui.sel < len(items):
		g.zombieShopAction(p, netZombieReqBuy, items[ui.sel].ID)
	case keys.close || esc:
		g.zombieShopAction(p, netZombieReqClose, 0)
		return
	}
	ctx := ui.st.begin(uiPassUpdate, nil, pointer)
	g.buildZombieShopPanel(ctx, p, zp, ui, r)
	ctx.end()
}

// drawZombieShops draws the open panels of this screen.
func (g *Game) drawZombieShops(screen *ebiten.Image) {
	if !g.zombieActive() {
		return
	}
	players := g.zombieLocalShopPlayers()
	rects := zombieShopRects(g.zombieOpenShopCount(players))
	slot := 0
	for _, p := range players {
		if !g.zombieShopIsOpen(p) {
			continue
		}
		r := rects[min(slot, len(rects)-1)]
		slot++
		ui := &g.zombie.shop[p.ID-1]
		ctx := ui.st.begin(uiPassDraw, screen, uiInput{X: g.uiLastCursorX, Y: g.uiLastCursorY})
		g.buildZombieShopPanel(ctx, p, g.zombie.playerState(p.ID), ui, r)
		ctx.end()
	}
}

// zombieShopFooterH: big panels show a key-hint line, the small 2x2 ones
// only the info line next to READY.
func zombieShopFooterH(r Rect) float64 {
	if r.H < 300 {
		return 44
	}
	return 62
}

func zombieShopRowsVisible(r Rect) int {
	listH := r.H - 74 - zombieShopFooterH(r)
	return max(1, int(listH/30))
}

func (g *Game) buildZombieShopPanel(ctx *uiCtx, p *Player, zp *ZombiePlayerState, ui *zombieShopUI, r Rect) {
	if zp == nil {
		return
	}
	prefix := "zs.p" + strconv.Itoa(p.ID)
	col := developedPlayerColorSwatch(p.PlayerColor)
	if ctx.drawing() {
		uiFillRound(ctx.dst, Rect{X: r.X + 3, Y: r.Y + 4, W: r.W, H: r.H}, 8, color.NRGBA{A: 0x70})
		uiFillRound(ctx.dst, r, 8, uiWithAlpha(uiInk900, 0.97))
		uiFillRound(ctx.dst, Rect{X: r.X, Y: r.Y, W: r.W, H: 34}, 8, uiInk700)
		uiFillRect(ctx.dst, Rect{X: r.X + 10, Y: r.Y + 8, W: 5, H: 18}, col)
		uiText(ctx.dst, uiFit("SHOP  -  "+p.Name, true, uiSizeButton, r.W-170), true, uiSizeButton, uiTextLight, r.X+22, r.Y+17, uiAlignLeft)
		uiText(ctx.dst, strconv.Itoa(zp.Points), true, 20, color.NRGBA{R: 0xff, G: 0xd8, B: 0x40, A: 0xff}, r.X+r.W-44, r.Y+17, uiAlignRight)
		uiText(ctx.dst, "PTS", false, uiSizeHint, uiTextMuted, r.X+r.W-12, r.Y+18, uiAlignRight)
	}
	ctx.Block(prefix+".bg", r)
	if t := ctx.TabBar(prefix+".tabs", Rect{X: r.X + 10, Y: r.Y + 40, W: r.W - 20, H: 28}, zombieTabLabels, ui.tab); t != ui.tab {
		ui.tab, ui.sel, ui.scroll = t, 0, 0
	}
	items := zombieShopTab(ui.tab)
	if ui.sel >= len(items) {
		ui.sel = max(0, len(items)-1)
	}
	visible := zombieShopRowsVisible(r)
	if ctx.pass == uiPassUpdate && ctx.in.Wheel != 0 && r.Contains(ctx.in.X, ctx.in.Y) {
		if ctx.in.Wheel > 0 {
			ui.scroll--
		} else {
			ui.scroll++
		}
	}
	if ui.sel < ui.scroll {
		ui.scroll = ui.sel
	}
	if ui.sel >= ui.scroll+visible && ctx.in.Wheel == 0 {
		ui.scroll = ui.sel - visible + 1
	}
	ui.scroll = max(0, min(ui.scroll, len(items)-visible))
	listY := r.Y + 74
	for row := 0; row < visible; row++ {
		i := ui.scroll + row
		if i >= len(items) {
			break
		}
		it := items[i]
		price := it.priceFor(zp)
		status := g.zombieCheckBuy(p, it)
		rr := Rect{X: r.X + 10, Y: listY + float64(row)*30, W: r.W - 20, H: 27}
		id := prefix + ".item." + strconv.Itoa(it.ID)
		selected := i == ui.sel
		if ctx.Row(id, rr, uiRowOpts{Selected: selected, InfoTitle: it.Name, InfoBody: it.Info,
			Draw: func(dst *ebiten.Image, r Rect, fg color.NRGBA, alpha float64) {
				uiText(dst, uiFit(it.Name, true, uiSizeBody, r.W-150), true, uiSizeBody, uiWithAlpha(fg, alpha), r.X+10, r.Y+r.H/2, uiAlignLeft)
				priceText := strconv.Itoa(price)
				priceCol := color.NRGBA{R: 0xff, G: 0xd8, B: 0x40, A: 0xff}
				if selected {
					priceCol = uiTextDark
				}
				switch status {
				case zombieMsgOwned:
					priceText, priceCol = "OWNED", uiOK
				case zombieMsgFull:
					priceText, priceCol = "FULL", uiTextMuted
				case zombieMsgNoGun:
					priceText, priceCol = "NO GUN", uiTextMuted
				case zombieMsgNoPoints:
					priceCol = uiDanger
				}
				uiText(dst, priceText, true, uiSizeBody, uiWithAlpha(priceCol, alpha), r.X+r.W-10, r.Y+r.H/2, uiAlignRight)
			}}) {
			ui.sel = i
			g.zombieShopAction(p, netZombieReqBuy, it.ID)
		}
	}
	if ctx.drawing() && len(items) > visible {
		// Scroll bar.
		track := Rect{X: r.X + r.W - 6, Y: listY, W: 3, H: float64(visible) * 30}
		uiFillRect(ctx.dst, track, uiInk600)
		h := track.H * float64(visible) / float64(len(items))
		y := track.Y + (track.H-h)*float64(ui.scroll)/float64(max(1, len(items)-visible))
		uiFillRect(ctx.dst, Rect{X: track.X, Y: y, W: 3, H: h}, uiOrange)
	}

	// Footer: item info or the last purchase result, then READY.
	small := r.H < 300
	footY := r.Y + r.H - zombieShopFooterH(r) + 4
	secs := (g.zombie.hud.PhaseTimer + TicksPerSecond - 1) / TicksPerSecond
	readyRect := Rect{X: r.X + r.W - 128, Y: footY + 14, W: 118, H: 36}
	if small {
		readyRect = Rect{X: r.X + r.W - 128, Y: footY, W: 118, H: 32}
	}
	if ctx.drawing() {
		info := ""
		infoCol := uiTextMuted
		if zp.MsgTimer > 0 && zp.Msg != zombieMsgNone {
			info = zombieMsgText[zp.Msg]
			infoCol = uiDanger
			if zp.Msg == zombieMsgBought || zp.Msg == zombieMsgTaken {
				infoCol = uiOK
			}
		} else if ui.sel >= 0 && ui.sel < len(items) {
			info = items[ui.sel].Info
		}
		if small {
			uiText(ctx.dst, uiFit(info, false, uiSizeHint, r.W-150), false, uiSizeHint, infoCol, r.X+12, footY+16, uiAlignLeft)
		} else {
			uiText(ctx.dst, uiFit(info, false, uiSizeHint, r.W-150), false, uiSizeHint, infoCol, r.X+12, footY+10, uiAlignLeft)
			hint := "UP/DOWN pick  LEFT/RIGHT tab  SHOOT buy  GRENADE ready"
			if !g.zombie.host {
				hint = "UP/DOWN pick  LEFT/RIGHT tab  SHOOT buy  GRENADE/ESC ready"
			}
			uiText(ctx.dst, uiFit(hint, false, uiSizeHint, r.W-150), false, uiSizeHint, uiTextMuted, r.X+12, footY+34, uiAlignLeft)
		}
	}
	if ctx.Button(prefix+".ready", readyRect, "READY  "+strconv.Itoa(secs)+"s", uiButtonOpts{Kind: uiButtonPrimary,
		InfoTitle: "READY", InfoBody: "Close the shop. The next wave starts when everybody is ready."}) {
		g.zombieShopAction(p, netZombieReqClose, 0)
	}
}
