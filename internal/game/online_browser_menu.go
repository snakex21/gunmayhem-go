package game

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Online screens, rebuilt with the Mayhem UI kit (phase 5): the server
// browser, the host setup ("HOST A SESSION" / SESSION SETTINGS) and, in
// online_lobby_menu.go, the lobby shared by host and guests.
//
// The screens only draw and route input. Every action calls the existing
// online functions with their existing codes (activateCustomNetworkHit,
// activateCustomHostSetupHit, activateCustomV2Hit, activateOnlineLobbyHit,
// joinFromCustomSession, pasteIntoMultiplayerAddress, copyHostInviteCode...),
// so netcode, protocol, tunnel and validation behaviour is unchanged.

const (
	onlineScreenBrowser = iota
	onlineScreenHostSetup
	onlineScreenLobby
)

var (
	onlineBrowserRefreshRect  = Rect{X: 760, Y: 30, W: 120, H: 40}
	onlineBrowserListPanel    = Rect{X: uiMargin, Y: uiContentTop, W: 452, H: 288}
	onlineBrowserPreviewPanel = Rect{X: 484, Y: uiContentTop, W: 396, H: 288}
	onlineBrowserJoinPanel    = Rect{X: uiMargin, Y: 392, W: 860, H: 124}
	onlineBrowserAddressRect  = Rect{X: 34, Y: 436, W: 520, H: 36}
	onlineBrowserPasteRect    = Rect{X: 564, Y: 436, W: 120, H: 36}
	onlineBrowserJoinRect     = Rect{X: 694, Y: 436, W: 172, H: 36}
	onlineBrowserLocalRect    = Rect{X: 232, Y: uiActionTop, W: 190, H: uiButtonH}

	onlineHostPanel      = Rect{X: uiMargin, Y: uiContentTop, W: 568, H: uiContentBottom - uiContentTop}
	onlineHostInfoRect   = Rect{X: 604, Y: uiContentTop, W: 276, H: uiContentBottom - uiContentTop}
	onlineHostNameRect   = Rect{X: 34, Y: 132, W: 540, H: 38}
	onlineHostReachRect  = [2]Rect{{X: 34, Y: 214, W: 265, H: 88}, {X: 309, Y: 214, W: 265, H: 88}}
	onlineHostAccessRect = [2]Rect{{X: 34, Y: 340, W: 265, H: 88}, {X: 309, Y: 340, W: 265, H: 88}}
	onlineHostCancelRect = Rect{X: 232, Y: uiActionTop, W: 190, H: uiButtonH}
)

const onlineBrowserRowH = 34.0

func onlineBrowserVisibleRows() int {
	return int((onlineBrowserListPanel.H - 62) / onlineBrowserRowH)
}

func onlineBrowserRowRect(row int) Rect {
	p := onlineBrowserListPanel
	return Rect{X: p.X + 10, Y: p.Y + 56 + float64(row)*onlineBrowserRowH, W: p.W - 20, H: onlineBrowserRowH - 4}
}

func onlineBrowserRowID(index int) string { return fmt.Sprintf("ob.srv.%d", index) }

// ---------------------------------------------------------------------------
// Routing

func (g *Game) onlineScreenKind() int {
	if g.customNetworkOpen {
		if g.customHostSetupOpen {
			return onlineScreenHostSetup
		}
		return onlineScreenBrowser
	}
	return onlineScreenLobby
}

// onlineEnterScreen resets the kit state whenever the online sub-screen
// changes, so focus / hover never leak from one screen into the next.
func (g *Game) onlineEnterScreen(kind int) {
	if g.onlineUIScreen == kind+1 {
		return
	}
	g.onlineUIScreen = kind + 1
	g.onlineUI.reset()
	g.onlineLeaveConfirm = false
	g.onlineLastMessage = ""
	if kind == onlineScreenLobby {
		g.customUI.reset()
		g.customLoadoutSlot = 0
	}
}

// updateOnlineCustomGame is the host / offline side (updateMenu). Guests run
// the lobby from updateNetClient -> updateOnlineClientCustomGame.
func (g *Game) updateOnlineCustomGame() {
	kind := g.onlineScreenKind()
	if kind == onlineScreenLobby && g.netplay == nil {
		// No session behind the lobby (it can only be left over from an old
		// flow): the server browser is the online entry point.
		g.returnToOnlineServerBrowser()
		kind = g.onlineScreenKind()
	}
	g.onlineEnterScreen(kind)
	switch kind {
	case onlineScreenBrowser:
		g.updateOnlineBrowser()
	case onlineScreenHostSetup:
		g.updateOnlineHostSetup()
	default:
		g.updateOnlineLobby()
	}
}

func (g *Game) drawOnlineCustomGame(screen *ebiten.Image) {
	in := uiInput{X: g.uiLastCursorX, Y: g.uiLastCursorY}
	switch g.onlineScreenKind() {
	case onlineScreenBrowser:
		ctx := g.onlineUI.begin(uiPassDraw, screen, in)
		g.buildOnlineBrowser(ctx)
		ctx.end()
	case onlineScreenHostSetup:
		ctx := g.onlineUI.begin(uiPassDraw, screen, in)
		g.buildOnlineHostSetup(ctx)
		ctx.end()
	default:
		ctx := g.customUI.begin(uiPassDraw, screen, in)
		g.buildOnlineLobby(ctx)
		ctx.end()
	}
}

// onlineMessageKind classifies a status / error line for its colour.
func onlineMessageKind(msg string) uiToastKind {
	upper := strings.ToUpper(msg)
	for _, bad := range []string{"ERROR", "FAILED", "DISCONNECTED", "MISMATCH", "FULL", "CANNOT", "UNABLE", "TIMEOUT", "REFUSED"} {
		if strings.Contains(upper, bad) {
			return uiToastError
		}
	}
	for _, good := range []string{"COPIED", "PASTED", "CONNECTED", "HOSTING", "JOINED"} {
		if strings.Contains(upper, good) {
			return uiToastOK
		}
	}
	return uiToastInfo
}

func uiToastAccent(kind uiToastKind) color.NRGBA {
	switch kind {
	case uiToastOK:
		return uiOK
	case uiToastWarning:
		return uiOrange
	case uiToastError:
		return uiDanger
	}
	return uiNetBlue
}

// drawOnlineStatusStrip draws a status / error message with a coloured
// accent, wrapped to at most maxLines lines.
func drawOnlineStatusStrip(dst *ebiten.Image, r Rect, msg string, maxLines int) {
	if dst == nil || strings.TrimSpace(msg) == "" {
		return
	}
	uiFillRound(dst, r, 4, uiInk900)
	uiFillRect(dst, Rect{X: r.X, Y: r.Y + 4, W: 4, H: r.H - 8}, uiToastAccent(onlineMessageKind(msg)))
	lineH := 15.0
	lines := len(wrapTextToWidth(msg, r.W-24, func(v string) float64 { return uiMeasure(false, uiSizeHint, v) }))
	if lines > maxLines {
		lines = maxLines
	}
	first := r.Y + r.H/2 - float64(lines-1)*lineH/2
	uiParagraph(dst, msg, false, uiSizeHint, uiTextLight, r.X+14, first, r.W-24, lineH, maxLines)
}

// ---------------------------------------------------------------------------
// Server browser

func (g *Game) updateOnlineBrowser() {
	g.pollMultiplayerServerBrowser()
	// Typing and Ctrl+V / Shift+Insert paste into the join field.
	g.updateMultiplayerAddressInput()
	in := g.readUIInput(&g.onlineUI)
	if g.multiplayerFocus {
		// The field owns the keyboard: ENTER joins, ESC only leaves the field.
		enter, esc := in.Enter, in.Esc
		in = in.pointerOnly()
		g.runOnlineBrowserUpdate(in)
		switch {
		case enter && !inpututil.IsKeyJustPressed(ebiten.KeySpace):
			g.activateCustomNetworkHit(menuHitCustomNetworkJoin)
		case esc:
			g.multiplayerFocus = false
		}
		return
	}
	g.runOnlineBrowserUpdate(in)
}

// runOnlineBrowserUpdate is one update pass for a frame of input (tests drive it).
func (g *Game) runOnlineBrowserUpdate(in uiInput) {
	ctx := g.onlineUI.begin(uiPassUpdate, nil, in)
	g.buildOnlineBrowser(ctx)
	ctx.end()
	g.onlineBrowserLastFocus = g.onlineUI.focus
}

func (g *Game) onlineBrowserStatusText() string {
	switch {
	case len(g.multiplayerServers) > 0:
		n := len(g.multiplayerServers)
		if n == 1 {
			return "1 SESSION FOUND"
		}
		return fmt.Sprintf("%d SESSIONS FOUND", n)
	case g.multiplayerDiscoveryDone && g.multiplayerDiscoveryCh == nil:
		return "NO OPEN SESSIONS"
	}
	return "SEARCHING..."
}

// onlineGoLocal is the old LOCAL toggle of the network page: close any
// session and show the local Custom Game.
func (g *Game) onlineGoLocal() {
	_ = g.CloseNetplay()
	g.customOnlineMode = false
	g.customNetworkOpen = false
	g.customHostSetupOpen = false
	g.onlineMatchSetup = false
	g.multiplayerFocus = false
	g.multiplayerMessage = "LOCAL CUSTOM GAME"
	g.customUI.reset()
	g.onlineUIScreen = 0
}

func (g *Game) buildOnlineBrowser(ctx *uiCtx) {
	ctx.Page(0.94)
	ctx.Header("ONLINE", "MAIN MENU  /  ONLINE  /  SERVER BROWSER")
	ctx.Badge(Rect{X: uiMargin + uiMeasure(true, uiSizeTitle, "ONLINE") + 14, Y: 31, W: 44, H: 22}, "NET", uiNetBlue)
	ctx.Label(g.onlineBrowserStatusText(), true, uiSizeBody, uiTextMuted, onlineBrowserRefreshRect.X-16, 50, uiAlignRight)
	if ctx.Button("ob.refresh", onlineBrowserRefreshRect, "REFRESH", uiButtonOpts{Size: uiSizeBody,
		InfoTitle: "REFRESH", InfoBody: "Search the LAN and the Internet server list (DHT) again."}) {
		g.activateCustomNetworkHit(menuHitCustomNetworkRefresh)
	}

	g.buildOnlineBrowserList(ctx)
	g.buildOnlineBrowserPreview(ctx)
	g.buildOnlineBrowserJoin(ctx)

	if ctx.Button("ob.back", uiActionBackRect(), "MAIN MENU", uiButtonOpts{InfoTitle: "MAIN MENU"}) {
		g.activateCustomNetworkHit(menuHitCustomNetworkMainMenu)
		return
	}
	if ctx.Button("ob.local", onlineBrowserLocalRect, "LOCAL VERSUS", uiButtonOpts{
		InfoTitle: "LOCAL VERSUS", InfoBody: "Leave online and set up a match on this computer."}) {
		g.onlineGoLocal()
		return
	}
	if ctx.Button("ob.host", uiActionPrimaryRect(220), "HOST SESSION", uiButtonOpts{Kind: uiButtonPrimary,
		InfoTitle: "HOST SESSION", InfoBody: "Create your own session on the LAN or the internet."}) {
		g.activateCustomNetworkHit(menuHitCustomNetworkCreateSetup)
		return
	}
	ctx.KeyHintBar([]uiKeyHint{{"ESC", "Main menu"}, {"UP/DOWN", "Move"}, {"ENTER", "Select / join"}, {"CTRL+V", "Paste code"}})
	ctx.Toasts()

	if g.pendingNetJoin != nil {
		g.buildOnlineConnectingModal(ctx)
		return
	}
	if ctx.Back() {
		g.activateCustomNetworkHit(menuHitCustomNetworkMainMenu)
	}
}

func (g *Game) clampOnlineBrowserScroll() {
	maxScroll := len(g.multiplayerServers) - onlineBrowserVisibleRows()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if g.multiplayerServerScroll > maxScroll {
		g.multiplayerServerScroll = maxScroll
	}
	if g.multiplayerServerScroll < 0 {
		g.multiplayerServerScroll = 0
	}
}

func onlineServerPlayers(s multiplayerServerInfo) string {
	if s.Players < 0 {
		return fmt.Sprintf("?/%d", s.MaxPlayers)
	}
	return fmt.Sprintf("%d/%d", s.Players, s.MaxPlayers)
}

func onlineServerSource(s multiplayerServerInfo) string {
	source := strings.ToUpper(strings.TrimSpace(s.Sources))
	if source == "" {
		source = "NETWORK"
	}
	return source
}

func (g *Game) buildOnlineBrowserList(ctx *uiCtx) {
	p := onlineBrowserListPanel
	ctx.Panel(p, "OPEN SESSIONS")
	ctx.Block("ob.list", p)
	g.clampOnlineBrowserScroll()
	visible := onlineBrowserVisibleRows()

	// Keyboard scrolling: Up / Down past the visible rows moves the window.
	// The kit has already moved focus for this frame, so decide from the row
	// that was focused before the key (onlineBrowserLastFocus).
	if ctx.pass == uiPassUpdate && ctx.live("ob.list") {
		was := g.onlineBrowserLastFocus
		if ctx.in.Up && was == onlineBrowserRowID(g.multiplayerServerScroll) && g.multiplayerServerScroll > 0 {
			g.multiplayerServerScroll--
			ctx.st.focus = onlineBrowserRowID(g.multiplayerServerScroll)
		}
		last := g.multiplayerServerScroll + visible - 1
		if ctx.in.DownKey && was == onlineBrowserRowID(last) && last+1 < len(g.multiplayerServers) {
			g.multiplayerServerScroll++
			ctx.st.focus = onlineBrowserRowID(last + 1)
		}
		hot := ctx.st.hot
		if ctx.in.Wheel != 0 && (hot == "ob.list" || strings.HasPrefix(hot, "ob.srv.")) {
			if ctx.in.Wheel > 0 {
				g.multiplayerServerScroll--
			} else {
				g.multiplayerServerScroll++
			}
			g.clampOnlineBrowserScroll()
		}
	}

	if len(g.multiplayerServers) == 0 {
		if ctx.drawing() {
			title, body := "SEARCHING FOR SESSIONS...", "Looking on your LAN and the Internet server list (DHT)."
			if g.multiplayerDiscoveryDone && g.multiplayerDiscoveryCh == nil {
				title, body = "NO OPEN SESSIONS", "Nobody is hosting an open session right now. Join with an invite code below, or host your own session."
			}
			uiText(ctx.dst, title, true, uiSizeButton, uiTextLight, p.X+18, p.Y+92, uiAlignLeft)
			uiParagraph(ctx.dst, body, false, uiSizeBody, uiTextMuted, p.X+18, p.Y+122, p.W-36, 20, 3)
		}
		return
	}
	if ctx.drawing() {
		uiText(ctx.dst, "NAME", true, uiSizeHint, uiTextMuted, p.X+22, p.Y+46, uiAlignLeft)
		uiText(ctx.dst, "SOURCE", true, uiSizeHint, uiTextMuted, p.X+p.W-150, p.Y+46, uiAlignLeft)
		uiText(ctx.dst, "PLAYERS", true, uiSizeHint, uiTextMuted, p.X+p.W-22, p.Y+46, uiAlignRight)
	}
	for row := 0; row < visible; row++ {
		index := g.multiplayerServerScroll + row
		if index >= len(g.multiplayerServers) {
			break
		}
		server := g.multiplayerServers[index]
		id := onlineBrowserRowID(index)
		selected := index == g.multiplayerSelectedServer
		info := fmt.Sprintf("%s - %s, players %s.", server.Name, onlineServerSource(server), onlineServerPlayers(server))
		if server.Full() {
			info += " The session is full."
		}
		keyboardJoin := ctx.pass == uiPassUpdate && ctx.activate == id && selected
		if ctx.Row(id, onlineBrowserRowRect(row), uiRowOpts{Selected: selected, InfoTitle: server.Name, InfoBody: info + " ENTER again joins.",
			Draw: func(dst *ebiten.Image, r Rect, fg color.NRGBA, alpha float64) {
				name := server.Name
				nameClr := fg
				if server.Full() && !selected {
					nameClr = uiTextMuted
				}
				uiTextIn(dst, name, true, uiSizeBody, uiWithAlpha(nameClr, alpha), Rect{X: r.X, Y: r.Y, W: r.W - 160, H: r.H}, uiAlignLeft, 12)
				badge := Rect{X: r.X + r.W - 140, Y: r.Y + 6, W: 70, H: r.H - 12}
				badgeFill := uiInk900
				if strings.Contains(onlineServerSource(server), "LAN") {
					badgeFill = uiNetBlue
				}
				uiFillRound(dst, badge, 3, uiWithAlpha(badgeFill, alpha))
				uiTextIn(dst, onlineServerSource(server), true, uiSizeHint, uiWithAlpha(uiTextLight, alpha), badge, uiAlignCenter, 4)
				players := onlineServerPlayers(server)
				if server.Full() {
					players = "FULL"
				}
				uiText(dst, players, true, uiSizeBody, uiWithAlpha(fg, alpha), r.X+r.W-12, r.Y+r.H/2, uiAlignRight)
			}}) {
			if keyboardJoin {
				g.activateCustomNetworkHit(menuHitCustomNetworkJoinSelected)
			} else {
				g.activateCustomNetworkHit(menuHitCustomNetworkServerBase + index)
			}
		}
	}
	if ctx.drawing() && len(g.multiplayerServers) > visible {
		maxScroll := len(g.multiplayerServers) - visible
		track := Rect{X: p.X + p.W - 7, Y: p.Y + 56, W: 3, H: float64(visible) * onlineBrowserRowH}
		barH := track.H * float64(visible) / float64(len(g.multiplayerServers))
		barY := track.Y + (track.H-barH)*float64(g.multiplayerServerScroll)/float64(maxScroll)
		uiFillRound(ctx.dst, Rect{X: track.X, Y: barY, W: 3, H: barH}, 1.5, uiInk500)
	}
}

func (g *Game) onlineSelectedServer() (multiplayerServerInfo, bool) {
	if g.multiplayerSelectedServer < 0 || g.multiplayerSelectedServer >= len(g.multiplayerServers) {
		return multiplayerServerInfo{}, false
	}
	return g.multiplayerServers[g.multiplayerSelectedServer], true
}

func (g *Game) buildOnlineBrowserPreview(ctx *uiCtx) {
	p := onlineBrowserPreviewPanel
	ctx.Panel(p, "SESSION DETAILS")
	server, ok := g.onlineSelectedServer()
	joinRect := Rect{X: p.X + 14, Y: p.Y + p.H - 56, W: p.W - 28, H: 44}
	if !ok {
		if ctx.drawing() {
			uiText(ctx.dst, "SELECT A SESSION", true, uiSizeButton, uiTextLight, p.X+14, p.Y+62, uiAlignLeft)
			uiParagraph(ctx.dst, "Click a session on the left to see its mode, maps and rules before you join.", false, uiSizeBody, uiTextMuted, p.X+14, p.Y+92, p.W-28, 20, 4)
		}
		ctx.Button("ob.joinsel", joinRect, "JOIN SESSION", uiButtonOpts{Kind: uiButtonPrimary, Disabled: true,
			InfoTitle: "JOIN SESSION", InfoBody: "Select a session first."})
		return
	}
	if ctx.drawing() {
		x, w := p.X+14, p.W-28
		uiText(ctx.dst, uiFit(server.Name, true, uiSizeButton, w), true, uiSizeButton, uiTextLight, x, p.Y+60, uiAlignLeft)
		uiText(ctx.dst, onlineServerSource(server)+"   /   PLAYERS "+onlineServerPlayers(server), true, uiSizeHint, uiNetBlue, x, p.Y+82, uiAlignLeft)
		if !server.HasRulePreview() {
			label, detail := "MATCH DETAILS UNAVAILABLE", "The host did not publish its mode, maps and rules."
			if server.Players < 0 {
				label, detail = "UNVERIFIED DHT SESSION", "Join to verify the host and see its maps and rules."
			}
			uiText(ctx.dst, label, true, uiSizeBody, uiTextLight, x, p.Y+116, uiAlignLeft)
			uiParagraph(ctx.dst, detail, false, uiSizeBody, uiTextMuted, x, p.Y+140, w, 20, 3)
		} else {
			uiText(ctx.dst, developedModeTitle(server.GameMode), true, uiSizeButton, uiTextLight, x, p.Y+110, uiAlignLeft)
			stats := fmt.Sprintf("LIVES %d   /   ROUNDS %d   /   CRATES %s   /   PICKUPS %s", server.Lives, server.Rounds, onOff(server.Crates), onOff(server.Pickups))
			uiText(ctx.dst, uiFit(stats, false, uiSizeHint, w), false, uiSizeHint, uiTextMuted, x, p.Y+132, uiAlignLeft)
			rules := strings.TrimSpace(server.RuleSummary)
			if rules == "" {
				rules = "NO MODIFIERS"
			}
			uiText(ctx.dst, uiFit(strings.ToUpper(rules), true, uiSizeHint, w), true, uiSizeHint, uiOrange, x, p.Y+152, uiAlignLeft)
			shown := server.Rounds
			if shown > 3 {
				shown = 3
			}
			for i := 0; i < shown; i++ {
				thumb := Rect{X: x + float64(i)*118, Y: p.Y + 166, W: 110, H: 52}
				uiFillRound(ctx.dst, thumb, 4, uiInk900)
				g.drawOnlineMapThumb(ctx.dst, server.Maps[i], uiInset(thumb, 2))
				ctx.Badge(Rect{X: thumb.X + 4, Y: thumb.Y + thumb.H - 20, W: 26, H: 16}, fmt.Sprintf("R%d", i+1), uiInk900)
			}
			if server.Rounds > 3 {
				uiText(ctx.dst, fmt.Sprintf("+%d", server.Rounds-3), true, uiSizeBody, uiTextMuted, x+w, p.Y+192, uiAlignRight)
			}
		}
	}
	label, info := "JOIN SESSION", "Connect to "+server.Name+"."
	if server.Full() {
		label, info = "SESSION FULL", "Every player slot of this session is taken."
	}
	if ctx.Button("ob.joinsel", joinRect, label, uiButtonOpts{Kind: uiButtonPrimary, Disabled: server.Full(), InfoTitle: "JOIN SESSION", InfoBody: info}) {
		g.activateCustomNetworkHit(menuHitCustomNetworkJoinSelected)
	}
}

func (g *Game) buildOnlineBrowserJoin(ctx *uiCtx) {
	p := onlineBrowserJoinPanel
	ctx.Panel(p, "JOIN WITH AN INVITE CODE, LINK OR IP:PORT")
	if ctx.TextField("ob.addr", onlineBrowserAddressRect, g.multiplayerAddress, "Paste an invite code or link (CTRL+V)", g.multiplayerFocus, false,
		"INVITE CODE", "Paste the code or link the host sent you, or type an IP:PORT. ENTER joins, ESC leaves the field.") {
		g.multiplayerFocus = true
		g.multiplayerNameFocus = false
	}
	if g.multiplayerFocus && ctx.PressedOutside("ob.addr") {
		g.multiplayerFocus = false
	}
	if ctx.Button("ob.paste", onlineBrowserPasteRect, "PASTE", uiButtonOpts{Size: uiSizeBody,
		InfoTitle: "PASTE", InfoBody: "Paste the invite code or link from the clipboard."}) {
		g.pasteIntoMultiplayerAddress()
	}
	if ctx.Button("ob.join", onlineBrowserJoinRect, "JOIN", uiButtonOpts{Disabled: strings.TrimSpace(g.multiplayerAddress) == "",
		InfoTitle: "JOIN", InfoBody: "Connect with the code, link or address in the field."}) {
		g.activateCustomNetworkHit(menuHitCustomNetworkJoin)
	}
	if ctx.drawing() {
		msg := g.multiplayerMessage
		if msg == "" {
			msg = "Friends who host send you an invite code or link. Open sessions on your LAN show up in the list."
		}
		drawOnlineStatusStrip(ctx.dst, Rect{X: p.X + 14, Y: p.Y + 86, W: p.W - 28, H: 32}, msg, 2)
	}
}

func (g *Game) buildOnlineConnectingModal(ctx *uiCtx) {
	m := Rect{X: 230, Y: 210, W: 440, H: 170}
	ctx.BeginModal(m, "CONNECTING")
	if ctx.drawing() {
		target := ""
		elapsed := 0
		if job := g.pendingNetJoin; job != nil {
			target = job.target
			elapsed = int(time.Since(job.started).Seconds())
		}
		uiText(ctx.dst, uiFit(strings.ToUpper(target), true, uiSizeButton, m.W-28), true, uiSizeButton, uiTextDark, m.X+14, m.Y+60, uiAlignLeft)
		dots := strings.Repeat(".", 1+(ctx.st.frame/20)%3)
		uiText(ctx.dst, fmt.Sprintf("Contacting the host%s  %ds", dots, elapsed), false, uiSizeBody, uiTextMutedDk, m.X+14, m.Y+90, uiAlignLeft)
		uiText(ctx.dst, "Press ESC to cancel.", true, uiSizeBody, uiTextDark, m.X+14, m.Y+130, uiAlignLeft)
	}
	ctx.EndModal()
}

// ---------------------------------------------------------------------------
// Host setup (create a session / edit the running one)

func (g *Game) onlineHostEditing() bool {
	return g.netplay != nil && g.netplay.mode == netplayHost
}

func (g *Game) updateOnlineHostSetup() {
	in := g.readUIInput(&g.onlineUI)
	if g.multiplayerNameFocus {
		g.updateMultiplayerSessionNameInput()
		enterOrEsc := in.Esc || (in.Enter && !inpututil.IsKeyJustPressed(ebiten.KeySpace))
		in = in.pointerOnly()
		if enterOrEsc {
			g.multiplayerNameFocus = false
		}
	}
	g.runOnlineHostSetupUpdate(in)
}

// runOnlineHostSetupUpdate is one update pass for a frame of input (tests drive it).
func (g *Game) runOnlineHostSetupUpdate(in uiInput) {
	ctx := g.onlineUI.begin(uiPassUpdate, nil, in)
	g.buildOnlineHostSetup(ctx)
	ctx.end()
}

// onlineHostSetupAction runs a host setup code; afterwards a running host is
// shown its lobby's SESSION tab.
func (g *Game) onlineHostSetupAction(code int) {
	g.activateCustomHostSetupHit(code)
	if g.onlineHostEditing() && !g.customNetworkOpen {
		g.onlineSessionTab = true
	}
}

func onlineOptionArt(title, subtitle string, disabled bool) func(dst *ebiten.Image, inner Rect) {
	return func(dst *ebiten.Image, inner Rect) {
		uiText(dst, title, true, uiSizeButton, uiTextDark, inner.X+10, inner.Y+20, uiAlignLeft)
		uiParagraph(dst, subtitle, false, uiSizeHint, uiTextMutedDk, inner.X+10, inner.Y+44, inner.W-20, 16, 2)
	}
}

func (g *Game) buildOnlineHostSetup(ctx *uiCtx) {
	editing := g.onlineHostEditing()
	title := "HOST A SESSION"
	if editing {
		title = "SESSION SETTINGS"
	}
	ctx.Page(0.94)
	ctx.Header(title, "MAIN MENU  /  ONLINE  /  HOST")

	p := onlineHostPanel
	ctx.Panel(p, "")
	ctx.Label("SESSION NAME", true, uiSizeBody, uiTextLight, p.X+14, 116, uiAlignLeft)
	if ctx.TextField("oh.name", onlineHostNameRect, g.multiplayerSessionName, defaultPublicServerName(), g.multiplayerNameFocus, editing,
		"SESSION NAME", "The name other players see in the server browser.") {
		g.multiplayerNameFocus = true
		g.multiplayerFocus = false
	}
	if g.multiplayerNameFocus && ctx.PressedOutside("oh.name") {
		g.multiplayerNameFocus = false
	}

	reachInfo := "Fixed while the session runs. LEAVE SESSION and host again to change it."
	ctx.Label("NETWORK REACH", true, uiSizeBody, uiTextLight, p.X+14, 200, uiAlignLeft)
	reach := []struct {
		code, scope     int
		title, subtitle string
		info            string
	}{
		{menuHitCustomHostScopeLAN, onlineSessionScopeLAN, "LAN ONLY", "Players on your local network.", "Direct connection to players on the same network."},
		{menuHitCustomHostScopeInternet, onlineSessionScopeInternet, "INTERNET", "Cloudflare WSS tunnel + LAN.", "A temporary Cloudflare tunnel lets friends anywhere join. Use an INVITE CODE for the most reliable connection."},
	}
	for i, o := range reach {
		info := o.info
		if editing {
			info = reachInfo
		}
		if ctx.Tile(fmt.Sprintf("oh.reach.%d", i), onlineHostReachRect[i], "", uiTileOpts{Selected: g.customHostScope == o.scope, Disabled: editing,
			InfoTitle: o.title, InfoBody: info, Art: onlineOptionArt(o.title, o.subtitle, editing)}) {
			g.onlineHostSetupAction(o.code)
		}
	}
	ctx.Label("ACCESS", true, uiSizeBody, uiTextLight, p.X+14, 326, uiAlignLeft)
	access := []struct {
		code, access    int
		title, subtitle string
		info            string
	}{
		{menuHitCustomHostAccessOpen, onlineSessionOpen, "OPEN", "Listed in the server browser.", "Anyone who can see the session in the server browser may join."},
		{menuHitCustomHostAccessInvite, onlineSessionInviteOnly, "INVITE CODE", "Hidden; only players with the code join.", "The session is not listed. You get a link / code to send to your friends."},
	}
	for i, o := range access {
		info := o.info
		if editing {
			info = reachInfo
		}
		if ctx.Tile(fmt.Sprintf("oh.access.%d", i), onlineHostAccessRect[i], "", uiTileOpts{Selected: g.customHostAccess == o.access, Disabled: editing,
			InfoTitle: o.title, InfoBody: info, Art: onlineOptionArt(o.title, o.subtitle, editing)}) {
			g.onlineHostSetupAction(o.code)
		}
	}
	if ctx.drawing() {
		note := "LAN stays direct. INTERNET uses a temporary Cloudflare tunnel; INVITE CODE is the most reliable access mode."
		if editing {
			note = "APPLY & RESTART restarts the host with these settings: connected players have to join again."
		}
		uiParagraph(ctx.dst, note, false, uiSizeHint, uiTextMuted, p.X+14, 466, p.W-28, 17, 2)
	}

	title2, body := ctx.FocusInfo()
	if title2 == "" {
		title2, body = "HOSTING", "Pick who can reach your session and how they find it, then create it. You configure the mode, maps and players in the lobby."
	}
	info := onlineHostInfoRect
	errMsg := ""
	if strings.HasPrefix(g.multiplayerMessage, "HOST ERROR") {
		errMsg = g.multiplayerMessage
		info.H -= 132
	}
	ctx.InfoPanel(info, title2, body)
	if ctx.drawing() && errMsg != "" {
		r := Rect{X: info.X, Y: info.Y + info.H + 8, W: info.W, H: 124}
		uiFillRound(ctx.dst, r, 6, uiInk700)
		uiFillRect(ctx.dst, Rect{X: r.X, Y: r.Y + 8, W: 4, H: r.H - 16}, uiDanger)
		uiParagraph(ctx.dst, errMsg, false, uiSizeHint, uiTextLight, r.X+16, r.Y+18, r.W-28, 16, 6)
	}

	if editing {
		if ctx.Button("oh.leave", uiActionBackRect(), "LEAVE SESSION", uiButtonOpts{Kind: uiButtonDanger,
			InfoTitle: "LEAVE SESSION", InfoBody: "Close the session for everyone and return to the server browser."}) {
			g.onlineHostSetupAction(menuHitCustomHostLeave)
			return
		}
		if ctx.Button("oh.cancel", onlineHostCancelRect, "CANCEL", uiButtonOpts{InfoTitle: "CANCEL", InfoBody: "Back to the lobby; the session keeps running."}) {
			g.onlineHostSetupAction(menuHitCustomHostBack)
			return
		}
		if ctx.Button("oh.create", uiActionPrimaryRect(220), "APPLY & RESTART", uiButtonOpts{Kind: uiButtonPrimary,
			InfoTitle: "APPLY & RESTART", InfoBody: "Restart the host with these settings."}) {
			g.onlineHostSetupAction(menuHitCustomHostCreate)
			return
		}
	} else {
		if ctx.Button("oh.back", uiActionBackRect(), "BACK TO SERVERS", uiButtonOpts{InfoTitle: "BACK TO SERVERS"}) {
			g.onlineHostSetupAction(menuHitCustomHostBack)
			return
		}
		if ctx.Button("oh.create", uiActionPrimaryRect(220), "CREATE SESSION", uiButtonOpts{Kind: uiButtonPrimary,
			InfoTitle: "CREATE SESSION", InfoBody: "Start hosting and open the lobby."}) {
			g.onlineHostSetupAction(menuHitCustomHostCreate)
			return
		}
	}
	ctx.KeyHintBar([]uiKeyHint{{"ESC", "Back"}, {"UP/DOWN", "Move"}, {"LEFT/RIGHT", "Move"}, {"ENTER", "Select"}})
	ctx.Toasts()
	if ctx.Back() {
		// ESC never closes a running host here: it is CANCEL while editing.
		g.onlineHostSetupAction(menuHitCustomHostBack)
	}
}

// Text input for the browser join field and the host session name.

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
