package game

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Online lobby (Mayhem UI, phase 5), shared by the host and the guests.
//
// Tabs: SESSION (host steps: host started -> invite link ready with a big
// COPY LINK -> waiting for players; a guest sees its connection), MATCH and
// MAPS (the local Custom Game builders; guests get them read-only through
// uiCtx.readOnly) and PLAYERS (online slot cards; EDIT LOADOUT opens the
// phase-4 loadout screen).
//
// Permissions are the existing ones: match edits go through
// customLocalActivate -> activateCustomV2Hit (customModifierEditingAllowed),
// character edits through activateCustomPlayerEditorHit
// (onlineMayEditCustomCharacterSlot), and a guest's activations are filtered
// by onlineClientActivate to its own slot exactly like the old client click
// path; the host still validates every received config (netwire_secure.go).

const (
	onlineLobbyTabSession = iota
	onlineLobbyTabMatch
	onlineLobbyTabMaps
	onlineLobbyTabPlayers
)

var (
	onlineLobbyTabLabels  = []string{"SESSION", "MATCH", "MAPS", "PLAYERS"}
	onlineLobbyTabBarRect = Rect{X: 380, Y: 30, W: 500, H: 40}

	onlineSessionStatusPanel  = Rect{X: uiMargin, Y: uiContentTop, W: 540, H: uiContentBottom - uiContentTop}
	onlineSessionPlayersPanel = Rect{X: 576, Y: uiContentTop, W: 304, H: uiContentBottom - uiContentTop}
	onlineSessionCodeRect     = Rect{X: 34, Y: 318, W: 512, H: 62}
	onlineSessionCopyRect     = Rect{X: 34, Y: 390, W: 512, H: 52}
	onlineLobbyCopyRect       = Rect{X: 232, Y: uiActionTop, W: 180, H: uiButtonH}
	onlineLobbyConfirmRect    = Rect{X: 250, Y: 200, W: 400, H: 190}
	onlineRulesModalRect      = Rect{X: 30, Y: 26, W: 840, H: 548}
)

func (g *Game) onlineLobbyIsHost() bool {
	return g != nil && g.customOnlineMode && g.netplay != nil && g.netplay.mode == netplayHost
}

func (g *Game) onlineLobbyIsClient() bool {
	return g != nil && g.customOnlineMode && g.netplay != nil && g.netplay.mode == netplayClient
}

// onlineClientActivate is the guest's only write path: an editor code for
// its own slot, then the config is sent to the host (old client click path).
func (g *Game) onlineClientActivate(code int) {
	slot := g.onlineLocalPlayerID() - 1
	if slot < 1 || slot >= len(g.customPlayers) {
		return
	}
	target, ok := customPlayerEditorHitSlot(code)
	if !ok || target != slot {
		return
	}
	if g.activateCustomPlayerEditorHit(code) {
		g.customPlayers[slot].Type = 1
		g.sendOnlineLocalConfigIfChanged()
	}
}

func (g *Game) onlineLobbyTab() int {
	if g.onlineLobbyIsClient() {
		if !g.onlineClientTabSet {
			return onlineLobbyTabPlayers
		}
		return g.onlineClientTab
	}
	if g.onlineSessionTab {
		return onlineLobbyTabSession
	}
	switch g.customPage {
	case customPageMaps:
		return onlineLobbyTabMaps
	case customPagePlayers:
		return onlineLobbyTabPlayers
	}
	return onlineLobbyTabMatch
}

// setOnlineLobbyTab switches the visible tab. A guest's tab is view-only
// state (its page is pinned to PLAYERS by the host snapshot); the host's
// MATCH / MAPS / PLAYERS tabs are the existing wizard page codes.
func (g *Game) setOnlineLobbyTab(tab int) {
	g.closeCustomLoadout()
	g.customLivesFocus = false
	if g.onlineLobbyIsClient() {
		g.onlineClientTab = tab
		g.onlineClientTabSet = true
		return
	}
	if tab == onlineLobbyTabSession {
		g.onlineSessionTab = true
		return
	}
	g.onlineSessionTab = false
	g.customLocalActivate(customLocalTabCodes[tab-1])
}

// ---------------------------------------------------------------------------
// Update entry points

// updateOnlineLobby is the host's lobby frame (updateMenu).
func (g *Game) updateOnlineLobby() {
	g.updateCustomWarning()
	textFocus := g.customLocalTextFocus()
	if g.customPage == customPageGame && !g.onlineSessionTab {
		g.updateCustomLivesInput()
	}
	if g.customPage == customPagePlayers {
		g.customMenuX = 0
		g.updateCustomPlayerCards()
		g.updateCustomNameInput()
	}
	if g.onlineLobbyTab() == onlineLobbyTabMaps {
		g.preloadCustomMapThumbs()
	}
	in := g.readUIInput(&g.customUI)
	if textFocus {
		in = in.pointerOnly()
	}
	g.runOnlineLobbyUpdate(in)
}

// updateOnlineClientCustomGame is the guest's lobby frame (updateNetClient).
func (g *Game) updateOnlineClientCustomGame() {
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	id := g.onlineLocalPlayerID()
	if id < 2 || id > maxOnlinePlayers {
		return
	}
	g.onlineEnterScreen(onlineScreenLobby)
	textFocus := g.customLocalTextFocus()
	if g.customPage == customPagePlayers {
		g.updateCustomPlayerCards()
		g.updateCustomNameInput()
	}
	if g.onlineLobbyTab() == onlineLobbyTabMaps {
		g.preloadCustomMapThumbs()
	}
	in := g.readUIInput(&g.customUI)
	if textFocus {
		in = in.pointerOnly()
	}
	g.runOnlineLobbyUpdate(in)
}

// runOnlineLobbyUpdate is one update pass for a frame of input (tests drive it).
func (g *Game) runOnlineLobbyUpdate(in uiInput) {
	ctx := g.customUI.begin(uiPassUpdate, nil, in)
	g.buildOnlineLobby(ctx)
	ctx.end()
	if msg := g.multiplayerMessage; msg != g.onlineLastMessage {
		g.onlineLastMessage = msg
		if strings.TrimSpace(msg) != "" && g.onlineScreenKind() == onlineScreenLobby {
			g.customUI.pushToast(msg, onlineMessageKind(msg))
		}
	}
}

// onlineLeave leaves the session: a host closes it for everyone, a guest
// disconnects (both land on the server browser, as before).
func (g *Game) onlineLeave() {
	g.onlineLeaveConfirm = false
	g.closeCustomLoadout()
	if g.onlineLobbyIsClient() {
		_ = g.CloseNetplay()
		g.returnToOnlineServerBrowser()
		g.multiplayerMessage = "DISCONNECTED"
		return
	}
	g.returnToOnlineServerBrowser()
}

// onlineLobbyStart is the host's START MATCH (the old SLOTS-step checks).
func (g *Game) onlineLobbyStart() {
	g.closeCustomLoadout()
	g.customLivesFocus = false
	if g.netplay == nil {
		g.returnToOnlineServerBrowser()
		return
	}
	if !g.onlineLobbyIsHost() {
		return
	}
	if g.onlineZombie {
		// Zombie co-op: the host may start alone; guests join in progress.
		g.startZombieMatch()
		return
	}
	if g.onlineActiveParticipantCount() < 2 {
		g.multiplayerMessage = "WAITING FOR ANOTHER PLAYER OR AI BOT"
		return
	}
	if modeHasLives(g.customMode) && g.customLives <= 0 {
		g.playCustomWarning(1)
		g.onlineSessionTab = false
		g.setCustomV2Page(customPageGame)
		g.customUI.pushToast("LIVES MUST BE AT LEAST 1", uiToastWarning)
		return
	}
	g.startCustomGame()
}

func (g *Game) onlineCopyLink() {
	if g.copyHostInviteCode() {
		g.customUI.pushToast("LINK COPIED - SEND IT TO YOUR FRIENDS", uiToastOK)
		return
	}
	g.customUI.pushToast(g.inviteCopyFeedback, uiToastWarning)
}

// ---------------------------------------------------------------------------
// Screen

func (g *Game) onlineLobbyBreadcrumb() string {
	name := strings.TrimSpace(g.multiplayerSessionName)
	parts := []string{"ONLINE"}
	if g.onlineLobbyIsHost() {
		if name == "" {
			name = defaultPublicServerName()
		}
		parts = append(parts, strings.ToUpper(name), "HOST")
	} else {
		parts = append(parts, fmt.Sprintf("GUEST P%d", g.onlineLocalPlayerID()))
	}
	// The tab bar starts at x=380: keep the trail short enough to fit.
	return uiFit(strings.Join(parts, "  /  "), false, uiSizeHint, onlineLobbyTabBarRect.X-uiMargin-16)
}

func (g *Game) buildOnlineLobby(ctx *uiCtx) {
	ctx.Page(0.92)
	if g.customLoadoutSlot > 0 {
		if g.loadoutOpen() && !g.customRulesOpen {
			g.buildCustomLoadout(ctx)
			return
		}
		g.customLoadoutSlot = 0
	}
	host := g.onlineLobbyIsHost()
	client := g.onlineLobbyIsClient()
	ctx.Header("LOBBY", g.onlineLobbyBreadcrumb())
	ctx.Badge(Rect{X: uiMargin + uiMeasure(true, uiSizeTitle, "LOBBY") + 14, Y: 31, W: 44, H: 22}, "NET", uiNetBlue)
	if g.onlineZombie {
		ctx.Badge(Rect{X: uiMargin + uiMeasure(true, uiSizeTitle, "LOBBY") + 64, Y: 31, W: 70, H: 22}, "ZOMBIE", zombieTagColor)
	}

	tab := g.onlineLobbyTab()
	if t := ctx.TabBar("on.tabs", onlineLobbyTabBarRect, onlineLobbyTabLabels, tab); t != tab {
		g.setOnlineLobbyTab(t)
		tab = g.onlineLobbyTab()
	}

	readOnly := !g.customModifierEditingAllowed()
	switch tab {
	case onlineLobbyTabSession:
		g.buildOnlineSessionTab(ctx)
	case onlineLobbyTabMatch, onlineLobbyTabMaps:
		ctx.readOnly = readOnly
		switch {
		case g.onlineZombie:
			g.buildZombieOnlineSettings(ctx)
		case tab == onlineLobbyTabMatch:
			g.buildCustomLocalMatch(ctx, true)
		default:
			g.buildCustomLocalMaps(ctx)
		}
		ctx.readOnly = false
	default:
		g.buildOnlinePlayers(ctx)
	}

	leaveLabel := "LEAVE SESSION"
	leaveInfo := "Close the session for every player and return to the server browser."
	if client {
		leaveLabel, leaveInfo = "LEAVE", "Disconnect from the host and return to the server browser."
	}
	if ctx.Button("on.leave", uiActionBackRect(), leaveLabel, uiButtonOpts{Kind: uiButtonDanger, InfoTitle: leaveLabel, InfoBody: leaveInfo}) {
		g.onlineLeaveConfirm = true
	}
	if host && tab != onlineLobbyTabSession {
		code, ready := g.currentHostDisplayJoinCode()
		invite := g.onlineSessionVisibility == onlineSessionInviteOnly
		copyInfo := "Copy the invite link to the clipboard (also CTRL+C)."
		if !invite {
			copyInfo = "Open sessions are listed in the server browser; there is no link to copy."
		} else if !ready {
			copyInfo = "The invite link is not ready yet."
		}
		if ctx.Button("on.copy", onlineLobbyCopyRect, "COPY LINK", uiButtonOpts{Disabled: !ready || code == "", InfoTitle: "COPY LINK", InfoBody: copyInfo}) {
			g.onlineCopyLink()
		}
	}
	if host {
		startInfo := "Start the match with everyone in the lobby."
		waiting := g.onlineActiveParticipantCount() < 2 && !g.onlineZombie
		if waiting {
			startInfo = "Waiting for another player or an AI bot."
		}
		if ctx.Button("on.start", uiActionPrimaryRect(220), "START MATCH", uiButtonOpts{Kind: uiButtonPrimary, Disabled: waiting, InfoTitle: "START MATCH", InfoBody: startInfo}) {
			g.onlineLobbyStart()
			return
		}
		if waiting && ctx.drawing() && tab != onlineLobbyTabSession {
			uiText(ctx.dst, "WAITING FOR A PLAYER / AI BOT", true, uiSizeHint, uiTextMuted, uiActionPrimaryRect(220).X-12, uiActionTop+uiButtonH/2, uiAlignRight)
		}
	} else {
		ctx.Button("on.start", uiActionPrimaryRect(220), "WAITING FOR HOST", uiButtonOpts{Kind: uiButtonPrimary, Disabled: true,
			InfoTitle: "WAITING FOR HOST", InfoBody: "The host starts the match."})
		g.drawOnlineGuestNote(ctx, readOnly && (tab == onlineLobbyTabMatch || tab == onlineLobbyTabMaps))
	}

	hints := []uiKeyHint{{"ESC", "Leave"}, {"UP/DOWN", "Move"}, {"LEFT/RIGHT", "Change"}, {"ENTER", "Select"}, {"Q", "Prev tab"}, {"E", "Next tab"}}
	if host {
		hints = append(hints, uiKeyHint{"CTRL+C", "Copy link"})
	}
	ctx.KeyHintBar(hints)
	ctx.Toasts()

	switch {
	case g.onlineLeaveConfirm:
		g.buildOnlineLeaveConfirm(ctx)
		return
	case host && g.customRulesOpen:
		if g.customChaosOpen {
			g.buildCustomLocalChaos(ctx)
		} else {
			g.buildCustomLocalModifiers(ctx)
		}
		return
	case host && g.onlineVisualPicker != onlineVisualNone:
		g.buildOnlinePickerModal(ctx)
		return
	case host && g.onlinePerkBlocklistOpen:
		g.buildOnlinePerkBlocklistModal(ctx)
		return
	case host && g.onlineBlocklistOpen:
		g.buildOnlineWeaponBlocklistModal(ctx)
		return
	case host && g.onlineRulesOpen:
		g.buildOnlineRulesModal(ctx)
		return
	}
	if ctx.Back() {
		g.onlineLeaveConfirm = true
	}
}

// drawOnlineGuestNote fills the guest's action bar: the match settings are
// the host's (read-only on MATCH / MAPS) and the host starts the match.
func (g *Game) drawOnlineGuestNote(ctx *uiCtx, readOnlyTab bool) {
	if !ctx.drawing() {
		return
	}
	text, fill := "THE HOST STARTS THE MATCH", uiInk700
	if readOnlyTab {
		text, fill = "READ ONLY - THE HOST SETS THE MATCH", uiNetBlue
	}
	w := uiMeasure(true, uiSizeHint, text) + 24
	r := Rect{X: 232, Y: uiActionTop + 11, W: w, H: 22}
	uiFillRound(ctx.dst, r, 4, fill)
	uiTextIn(ctx.dst, text, true, uiSizeHint, uiTextLight, r, uiAlignCenter, 6)
}

func (g *Game) buildOnlineLeaveConfirm(ctx *uiCtx) {
	m := onlineLobbyConfirmRect
	host := g.onlineLobbyIsHost()
	title, body, action := "LEAVE?", "You will be disconnected from the host.", "LEAVE"
	if host {
		title, body, action = "CLOSE SESSION?", "Every connected player is disconnected and the invite link stops working.", "CLOSE SESSION"
		if g.netplay != nil && g.netplay.connectedCount() == 0 {
			body = "Nobody has joined yet. The invite link stops working."
		}
	}
	ctx.BeginModal(m, title)
	if ctx.drawing() {
		uiParagraph(ctx.dst, body, false, uiSizeBody, uiTextMutedDk, m.X+14, m.Y+60, m.W-28, 20, 3)
	}
	if ctx.Button("on.confirm.stay", Rect{X: m.X + 14, Y: m.Y + m.H - 58, W: 170, H: 44}, "STAY", uiButtonOpts{Kind: uiButtonPaper}) {
		g.onlineLeaveConfirm = false
	}
	if ctx.Button("on.confirm.leave", Rect{X: m.X + m.W - 14 - 200, Y: m.Y + m.H - 58, W: 200, H: 44}, action, uiButtonOpts{Kind: uiButtonDanger}) {
		g.onlineLeave()
		ctx.EndModal()
		return
	}
	ctx.EndModal()
	if ctx.Back() {
		g.onlineLeaveConfirm = false
	}
}

// ---------------------------------------------------------------------------
// SESSION tab

type onlineStepState int

const (
	onlineStepPending onlineStepState = iota
	onlineStepActive
	onlineStepDone
	onlineStepError
)

type onlineHostStep struct {
	title  string
	detail string
	state  onlineStepState
}

func (g *Game) onlineSessionScopeText() string {
	scope := "LAN"
	if g.onlineSessionScope == onlineSessionScopeInternet {
		scope = "INTERNET + LAN"
	}
	access := "OPEN"
	if g.onlineSessionVisibility == onlineSessionInviteOnly {
		access = "INVITE CODE"
	}
	return scope + " / " + access
}

// onlineHostTunnelError is the tunnel's user-visible error, if any.
func (g *Game) onlineHostTunnelError() error {
	if g.onlineSessionScope != onlineSessionScopeInternet || g.cloudflareTunnel == nil {
		return nil
	}
	_, _, _, err := g.cloudflareTunnel.Status()
	return err
}

// onlineHostSteps derives the host flow from the existing session status:
// 1 host started, 2 invite link (tunnel) or browser visibility, 3 players.
func (g *Game) onlineHostSteps() [3]onlineHostStep {
	var steps [3]onlineHostStep
	steps[0] = onlineHostStep{title: "HOST STARTED", detail: g.onlineSessionScopeText() + " SESSION", state: onlineStepDone}
	label := g.currentHostDisplayJoinCodeLabel()
	if g.onlineSessionVisibility == onlineSessionInviteOnly {
		code, ready := g.currentHostDisplayJoinCode()
		switch {
		case ready && code != "":
			steps[1] = onlineHostStep{title: "INVITE LINK READY", detail: "Copy the link and send it to your friends.", state: onlineStepDone}
		case g.onlineHostTunnelError() != nil:
			steps[1] = onlineHostStep{title: "TUNNEL ERROR", detail: strings.TrimPrefix(label, "INVITE CODE: "), state: onlineStepError}
		case g.onlineSessionScope == onlineSessionScopeInternet:
			steps[1] = onlineHostStep{title: "STARTING TUNNEL", detail: strings.TrimPrefix(label, "INVITE CODE: "), state: onlineStepActive}
		default:
			steps[1] = onlineHostStep{title: "PREPARING INVITE CODE", detail: strings.TrimPrefix(label, "INVITE CODE: "), state: onlineStepActive}
		}
	} else {
		steps[1] = onlineHostStep{title: "LISTED IN THE SERVER BROWSER", detail: label, state: onlineStepDone}
	}
	joined := 0
	if g.netplay != nil {
		joined = g.netplay.connectedCount()
	}
	switch {
	case joined == 1:
		steps[2] = onlineHostStep{title: "1 PLAYER JOINED", detail: "Add AI bots or start the match when you are ready.", state: onlineStepDone}
	case joined > 1:
		steps[2] = onlineHostStep{title: fmt.Sprintf("%d PLAYERS JOINED", joined), detail: "Add AI bots or start the match when you are ready.", state: onlineStepDone}
	case steps[1].state == onlineStepDone:
		steps[2] = onlineHostStep{title: "WAITING FOR PLAYERS", detail: "Players appear here as soon as they join. Open slots on the PLAYERS tab.", state: onlineStepActive}
	default:
		steps[2] = onlineHostStep{title: "WAITING FOR PLAYERS", detail: "Players can join once the link is ready.", state: onlineStepPending}
	}
	return steps
}

func drawOnlineStep(ctx *uiCtx, r Rect, number int, step onlineHostStep) {
	if !ctx.drawing() {
		return
	}
	uiFillRound(ctx.dst, r, 5, uiWithAlpha(uiInk600, 0.55))
	circle := Rect{X: r.X + 12, Y: r.Y + r.H/2 - 15, W: 30, H: 30}
	fill, text, mark := uiInk500, uiTextMuted, fmt.Sprint(number)
	switch step.state {
	case onlineStepActive:
		fill, text = uiOrange, uiTextDark
		mark = []string{".", "..", "..."}[(ctx.st.frame/15)%3]
	case onlineStepDone:
		fill, text, mark = uiOK, uiTextLight, "OK"
	case onlineStepError:
		fill, text, mark = uiDanger, uiTextLight, "!"
	}
	uiFillRound(ctx.dst, circle, 15, fill)
	uiTextIn(ctx.dst, mark, true, uiSizeHint, text, circle, uiAlignCenter, 2)
	titleClr := uiTextLight
	if step.state == onlineStepPending {
		titleClr = uiTextMuted
	}
	uiText(ctx.dst, step.title, true, uiSizeButton, titleClr, r.X+54, r.Y+17, uiAlignLeft)
	uiText(ctx.dst, uiFit(step.detail, false, uiSizeHint, r.W-66), false, uiSizeHint, uiTextMuted, r.X+54, r.Y+37, uiAlignLeft)
}

func (g *Game) buildOnlineSessionTab(ctx *uiCtx) {
	if g.onlineLobbyIsClient() {
		g.buildOnlineClientSessionTab(ctx)
	} else {
		g.buildOnlineHostSessionTab(ctx)
	}
	g.buildOnlineSessionPlayers(ctx)
}

func (g *Game) buildOnlineHostSessionTab(ctx *uiCtx) {
	p := onlineSessionStatusPanel
	ctx.Panel(p, "HOST STATUS")
	steps := g.onlineHostSteps()
	for i, step := range steps {
		drawOnlineStep(ctx, Rect{X: p.X + 14, Y: p.Y + 48 + float64(i)*58, W: p.W - 28, H: 50}, i+1, step)
	}

	invite := g.onlineSessionVisibility == onlineSessionInviteOnly
	code, ready := g.currentHostDisplayJoinCode()
	tunnelErr := g.onlineHostTunnelError()
	box := onlineSessionCodeRect
	if ctx.drawing() {
		uiFillRound(ctx.dst, box, 5, uiInk900)
		switch {
		case tunnelErr != nil && !ready:
			uiFillRect(ctx.dst, Rect{X: box.X, Y: box.Y + 6, W: 4, H: box.H - 12}, uiDanger)
			uiParagraph(ctx.dst, "TUNNEL ERROR: "+tunnelErr.Error(), false, uiSizeHint, uiTextLight, box.X+14, box.Y+16, box.W-26, 15, 3)
		case invite:
			uiText(ctx.dst, "INVITE LINK", true, uiSizeHint, uiTextMuted, box.X+14, box.Y+16, uiAlignLeft)
			shown, clr := code, uiTextLight
			if !ready || code == "" {
				shown, clr = strings.TrimPrefix(g.currentHostDisplayJoinCodeLabel(), "INVITE CODE: "), uiTextMuted
			}
			uiText(ctx.dst, uiFit(shown, true, uiSizeButton, box.W-28), true, uiSizeButton, clr, box.X+14, box.Y+42, uiAlignLeft)
		default:
			uiText(ctx.dst, "OPEN SESSION", true, uiSizeHint, uiTextMuted, box.X+14, box.Y+16, uiAlignLeft)
			uiText(ctx.dst, uiFit(g.currentHostDisplayJoinCodeLabel(), true, uiSizeBody, box.W-28), true, uiSizeBody, uiTextLight, box.X+14, box.Y+42, uiAlignLeft)
		}
	}

	if tunnelErr != nil && !ready {
		if ctx.Button("on.restart", onlineSessionCopyRect, "RESTART SESSION", uiButtonOpts{Kind: uiButtonPrimary, Size: 22,
			InfoTitle: "RESTART SESSION", InfoBody: "Start the host and the tunnel again with the same settings."}) {
			g.createConfiguredOnlineSession()
			g.onlineSessionTab = true
		}
	} else {
		info := "Copy the invite link to the clipboard (also CTRL+C)."
		if !invite {
			info = "Open sessions are listed in the server browser; there is no link to copy."
		} else if !ready {
			info = "The link appears as soon as the tunnel is ready."
		}
		if ctx.Button("on.copybig", onlineSessionCopyRect, "COPY LINK", uiButtonOpts{Kind: uiButtonPrimary, Size: 24, Disabled: !ready || code == "",
			InfoTitle: "COPY LINK", InfoBody: info}) {
			g.onlineCopyLink()
		}
	}
	if ctx.drawing() {
		hint := "Friends paste the link in ONLINE > JOIN WITH AN INVITE CODE and press JOIN."
		clr := uiTextMuted
		switch {
		case tunnelErr != nil && !ready:
			hint = "Cloudflare could not open the tunnel. RESTART SESSION tries again (the full error is in network.log)."
		case !invite:
			hint = "Players on your network see this session in their server browser."
		}
		if g.inviteCopyFeedback != "" && time.Now().Before(g.inviteCopyFeedbackUntil) {
			hint, clr = g.inviteCopyFeedback, uiOK
		}
		uiParagraph(ctx.dst, hint, true, uiSizeBody, clr, p.X+16, 462, p.W-32, 19, 2)
	}
}

func (g *Game) buildOnlineClientSessionTab(ctx *uiCtx) {
	p := onlineSessionStatusPanel
	ctx.Panel(p, "CONNECTION")
	id := g.onlineLocalPlayerID()
	ping := "MEASURING..."
	if g.netplay != nil {
		if rtt, ok := g.netplay.hostRTTMillis(); ok {
			ping = fmt.Sprintf("%d ms", rtt)
		}
	}
	steps := []onlineHostStep{
		{title: "CONNECTED TO THE HOST", detail: fmt.Sprintf("You are PLAYER %d.  %s", id, g.customSessionStatus()), state: onlineStepDone},
		{title: "PING " + ping, detail: "Round trip to the host.", state: onlineStepDone},
		{title: "WAITING FOR THE HOST", detail: "The host picks the mode and maps and starts the match.", state: onlineStepActive},
	}
	for i, step := range steps {
		drawOnlineStep(ctx, Rect{X: p.X + 14, Y: p.Y + 48 + float64(i)*58, W: p.W - 28, H: 50}, i+1, step)
	}
	if ctx.drawing() {
		uiParagraph(ctx.dst, "Set up your character, weapon, perk, gear and throwable while you wait. MATCH and MAPS show the host's settings.",
			false, uiSizeBody, uiTextMuted, p.X+16, 336, p.W-32, 20, 3)
	}
	slot := id - 1
	if ctx.Button("on.myloadout", onlineSessionCopyRect, "EDIT MY LOADOUT", uiButtonOpts{Kind: uiButtonPrimary, Size: 22,
		Disabled:  slot < 1 || slot >= len(g.customPlayers) || g.customPlayers[slot].Type == 0,
		InfoTitle: "EDIT MY LOADOUT", InfoBody: "Open your loadout screen."}) {
		g.onlineClientTab = onlineLobbyTabPlayers
		g.onlineClientTabSet = true
		g.openCustomLoadout(slot)
	}
}

// onlineSlotStatus describes a slot for the lobby: label, colour, ping text.
func (g *Game) onlineSlotStatus(slot int) (string, color.NRGBA, string) {
	cfg := g.customPlayers[slot]
	host := g.onlineLobbyIsHost()
	ping := ""
	switch {
	case slot == 0:
		if !host && g.netplay != nil {
			if rtt, ok := g.netplay.hostRTTMillis(); ok {
				ping = fmt.Sprintf("%d ms", rtt)
			}
		}
		return "HOST", uiNetBlue, ping
	case cfg.Type == 2:
		return "AI BOT", uiInk500, ""
	case cfg.Type == 1 && g.onlinePlayerSlots[slot]:
		if host && g.netplay != nil {
			if rtt, ok := g.netplay.peerRTTMillis(slot + 1); ok {
				ping = fmt.Sprintf("%d ms", rtt)
			} else {
				ping = "..."
			}
		}
		return "CONNECTED", uiOK, ping
	case cfg.Type == 1:
		return "WAITING FOR PLAYER", uiOrangePress, ""
	}
	return "EMPTY", uiInk500, ""
}

func onlinePingColor(ping string) color.NRGBA {
	var ms int
	if _, err := fmt.Sscanf(ping, "%d ms", &ms); err != nil {
		return uiTextMuted
	}
	switch {
	case ms < 90:
		return uiOK
	case ms < 180:
		return uiOrange
	}
	return uiDanger
}

func (g *Game) onlineSlotName(slot int) string {
	name := strings.TrimSpace(g.customPlayers[slot].Name)
	if name == "" {
		name = fmt.Sprintf("Player %d", slot+1)
	}
	return strings.ToUpper(name)
}

func (g *Game) buildOnlineSessionPlayers(ctx *uiCtx) {
	p := onlineSessionPlayersPanel
	ctx.Panel(p, "PLAYERS")
	if ctx.drawing() {
		localID := g.onlineLocalPlayerID()
		for slot := range g.customPlayers {
			r := Rect{X: p.X + 14, Y: p.Y + 48 + float64(slot)*58, W: p.W - 28, H: 50}
			uiFillRound(ctx.dst, r, 5, uiWithAlpha(uiInk600, 0.55))
			ctx.Badge(Rect{X: r.X + 10, Y: r.Y + 13, W: 30, H: 24}, fmt.Sprintf("P%d", slot+1), uiInk900)
			status, clr, ping := g.onlineSlotStatus(slot)
			name := g.onlineSlotName(slot)
			if g.customPlayers[slot].Type == 0 || status == "WAITING FOR PLAYER" {
				name = "-"
			}
			if localID == slot+1 {
				name += "  (YOU)"
			}
			uiText(ctx.dst, uiFit(name, true, uiSizeBody, r.W-120), true, uiSizeBody, uiTextLight, r.X+50, r.Y+17, uiAlignLeft)
			uiFillRound(ctx.dst, Rect{X: r.X + 50, Y: r.Y + 31, W: 8, H: 8}, 4, clr)
			uiText(ctx.dst, status, true, uiSizeHint, uiTextMuted, r.X+64, r.Y+36, uiAlignLeft)
			if ping != "" {
				uiText(ctx.dst, ping, true, uiSizeBody, onlinePingColor(ping), r.X+r.W-10, r.Y+r.H/2, uiAlignRight)
			}
		}
	}
	if g.onlineLobbyIsHost() {
		if ctx.Button("on.settings", Rect{X: p.X + 14, Y: p.Y + p.H - 56, W: p.W - 28, H: 42}, "SESSION SETTINGS", uiButtonOpts{Size: uiSizeBody,
			InfoTitle: "SESSION SETTINGS", InfoBody: "Name, reach and access of the session (APPLY restarts the host)."}) {
			g.closeCustomLoadout()
			g.customLocalActivate(menuHitCustomV2TabSession)
		}
	}
}

// ---------------------------------------------------------------------------
// MATCH: online rules row

func (g *Game) onlineRulesShortSummary() string {
	weapon := onlineWeaponModeLabel(g.onlineWeaponMode)
	switch g.onlineWeaponMode {
	case onlineWeaponsOnlyOne:
		weapon = "ONLY " + strings.ToUpper(weaponDisplayName(g.onlineOnlyWeapon))
	case onlineWeaponsBlocklist:
		weapon = fmt.Sprintf("%d WEAPONS BLOCKED", g.blockedOnlineWeaponCount())
	}
	perks := "PERKS ON"
	if !g.onlinePerksEnabled {
		perks = "PERKS OFF"
	}
	return weapon + "  /  " + perks
}

func (g *Game) buildOnlineRulesRow(ctx *uiCtx, row Rect) {
	ctrl := ctx.SettingRow(row, "ONLINE RULES", "on.rules", 190)
	ctx.Label(uiFit(g.onlineRulesShortSummary(), true, uiSizeBody, ctrl.W-160), true, uiSizeBody, uiTextLight, ctrl.X+4, ctrl.Y+ctrl.H/2, uiAlignLeft)
	if ctx.Button("on.rules", Rect{X: ctrl.X + ctrl.W - 140, Y: ctrl.Y, W: 140, H: ctrl.H}, "CONFIGURE", uiButtonOpts{Size: uiSizeBody,
		InfoTitle: "ONLINE RULES", InfoBody: "Map rotation, weapon rule and blocklists, perks and power-ups for every player."}) {
		g.customLivesFocus = false
		g.customLocalActivate(menuHitCustomSessionRules)
	}
}

// ---------------------------------------------------------------------------
// PLAYERS (online slot cards)

func (g *Game) buildOnlinePlayers(ctx *uiCtx) {
	for slot := range g.customPlayers {
		g.buildOnlinePlayerCard(ctx, slot)
	}
	if ret := g.customLoadoutReturn; ret > 0 && ctx.pass == uiPassUpdate {
		g.customLoadoutReturn = 0
		ctx.st.focus = fmt.Sprintf("on.p%d.edit", ret-1)
	}
}

func (g *Game) buildOnlinePlayerCard(ctx *uiCtx, slot int) {
	cfg := g.customPlayers[slot]
	r := customLocalCardRect(slot)
	x, w := r.X+12, r.W-24
	host := g.onlineLobbyIsHost()
	mine := g.onlineLocalPlayerID() == slot+1
	editable := g.onlineMayEditCustomCharacterSlot(slot) && cfg.Type != 0
	status, statusClr, ping := g.onlineSlotStatus(slot)
	waiting := cfg.Type == 1 && slot > 0 && !g.onlinePlayerSlots[slot]
	empty := cfg.Type == 0

	if ctx.drawing() {
		ctx.Card(r, "")
		strip := uiInk500
		if !empty && !waiting {
			strip = developedPlayerColorSwatch(cfg.Color)
			strip.A = 0xff
		}
		uiFillRound(ctx.dst, Rect{X: r.X, Y: r.Y, W: r.W, H: 8}, 4, strip)
		ctx.Badge(Rect{X: x, Y: r.Y + 16, W: 34, H: 24}, fmt.Sprintf("P%d", slot+1), uiInk900)
		if mine {
			ctx.Badge(Rect{X: x + 40, Y: r.Y + 16, W: 40, H: 24}, "YOU", uiOrangePress)
		}
		if ping != "" {
			uiText(ctx.dst, ping, true, uiSizeHint, onlinePingColor(ping), r.X+r.W-12, r.Y+28, uiAlignRight)
		}
		// Status pill.
		pill := Rect{X: x, Y: r.Y + 88, W: w, H: 28}
		uiFillRound(ctx.dst, pill, 4, uiPaperShade)
		uiFillRound(ctx.dst, Rect{X: pill.X + 10, Y: pill.Y + 10, W: 8, H: 8}, 4, statusClr)
		uiTextIn(ctx.dst, status, true, uiSizeHint, uiTextDark, Rect{X: pill.X + 24, Y: pill.Y, W: pill.W - 28, H: pill.H}, uiAlignLeft, 0)
	}

	nameID := fmt.Sprintf("on.p%d.name", slot)
	focused := g.customNameFocus == slot+1
	placeholder := fmt.Sprintf("Player %d", slot+1)
	if waiting {
		placeholder = "Waiting for a player..."
	}
	if ctx.TextField(nameID, Rect{X: x, Y: r.Y + 48, W: w, H: 32}, cfg.Name, placeholder, focused, !editable,
		"NAME", "Click to type a name. ENTER or ESC finishes.") {
		g.customLocalActivate(menuHitPlayerColorBase + 500 + slot)
	}
	if focused && ctx.PressedOutside(nameID) {
		g.customNameFocus = 0
	}

	if empty {
		if ctx.drawing() {
			uiText(ctx.dst, "SLOT EMPTY", true, uiSizeSection, uiTextDark, x, r.Y+160, uiAlignLeft)
			body := "The host can open this slot for an online player or add an AI bot."
			if host {
				body = "Open the slot for a player who joins online, or add an AI bot."
			}
			uiParagraph(ctx.dst, body, false, uiSizeBody, uiTextMutedDk, x, r.Y+192, w, 19, 4)
		}
		if host && slot > 0 {
			if ctx.Button(fmt.Sprintf("on.p%d.open", slot), Rect{X: x, Y: r.Y + 318, W: w, H: 40}, "OPEN ONLINE SLOT", uiButtonOpts{Kind: uiButtonPrimary, Size: uiSizeBody,
				InfoTitle: "OPEN ONLINE SLOT", InfoBody: "A player who joins with the link takes this slot."}) {
				g.customLocalActivate(playerSlotHit(slot, playerSlotActionHuman))
			}
			if ctx.Button(fmt.Sprintf("on.p%d.addai", slot), Rect{X: x, Y: r.Y + 366, W: w, H: 40}, "ADD AI BOT", uiButtonOpts{Size: uiSizeBody,
				InfoTitle: "ADD AI BOT", InfoBody: "A bot plays this slot, run by the host."}) {
				g.customLocalActivate(playerSlotHit(slot, playerSlotActionAI))
			}
		}
		return
	}

	if ctx.drawing() {
		preview := Rect{X: x, Y: r.Y + 124, W: w, H: 100}
		uiFillRound(ctx.dst, preview, 6, uiPaperShade)
		if waiting {
			uiParagraph(ctx.dst, "Waiting for a player to join with the invite link...", true, uiSizeBody, uiTextMutedDk, preview.X+12, preview.Y+34, preview.W-24, 19, 3)
		} else {
			g.drawSetupPreviewIn(ctx.dst, cfg, uiInset(preview, 6), 1)
			gear := migrateLegacyGearPerk(cfg)
			rows := [][2]string{
				{"GUN", weaponDisplayName(cfg.Gun)},
				{"PERK", onlinePerkName(gear.Perk)},
				{"ABILITY", abilityName(gear.Ability)},
				{"GRENADE", strings.ToUpper(throwableName(cfg.Throwable))},
			}
			for i, row := range rows {
				mid := r.Y + 240 + float64(i)*18
				uiText(ctx.dst, row[0], true, uiSizeHint, uiTextMutedDk, x, mid, uiAlignLeft)
				uiText(ctx.dst, uiFit(strings.ToUpper(row[1]), true, uiSizeHint, w-62), true, uiSizeHint, uiTextDark, x+62, mid, uiAlignLeft)
			}
		}
	}

	if g.customMode == SourceGameModeTeams && !waiting {
		team := 0
		if cfg.Team == 2 {
			team = 1
		}
		if v := ctx.Segmented(fmt.Sprintf("on.p%d.team", slot), Rect{X: x, Y: r.Y + 310, W: w, H: 28}, []string{"TEAM A", "TEAM B"}, team, !editable,
			"TEAM", "Team Battle: players on the same team fight together."); v != team {
			g.customLocalActivate(customTeamHit(slot, v+1))
		}
	} else if ctx.drawing() && !waiting {
		hint := "READ ONLY"
		switch {
		case editable && mine:
			hint = "YOUR CHARACTER"
		case editable:
			hint = "YOU CONTROL THIS LOADOUT"
		case cfg.Type == 2:
			hint = "BOT LOADOUT SET BY THE HOST"
		}
		uiText(ctx.dst, hint, true, uiSizeHint, uiTextMutedDk, x, r.Y+324, uiAlignLeft)
	}

	editInfo := "Character, colour, starting weapon, perk, ability / gear and throwable."
	if !editable {
		editInfo = "Only this player can change this loadout."
	}
	if ctx.Button(fmt.Sprintf("on.p%d.edit", slot), Rect{X: x, Y: r.Y + 346, W: w, H: 34}, "EDIT LOADOUT", uiButtonOpts{Size: uiSizeBody, Disabled: !editable || waiting,
		InfoTitle: "EDIT LOADOUT", InfoBody: editInfo}) {
		g.openCustomLoadout(slot)
	}
	if host && slot > 0 {
		label, info := "CLOSE SLOT", "Close this online slot."
		switch {
		case cfg.Type == 2:
			label, info = "REMOVE BOT", "Remove the AI bot from this slot."
		case g.onlinePlayerSlots[slot]:
			label, info = "KICK PLAYER", "Disconnect this player and close the slot."
		}
		if ctx.Button(fmt.Sprintf("on.p%d.remove", slot), Rect{X: x, Y: r.Y + 386, W: w, H: 32}, label, uiButtonOpts{Kind: uiButtonDanger, Size: uiSizeBody,
			InfoTitle: label, InfoBody: info}) {
			g.closeCustomLoadout()
			g.customLocalActivate(menuHitCustomV2SlotRemoveBase + slot)
		}
	}
}

// ---------------------------------------------------------------------------
// Online rules modals (host). Every control sends the old overlay's code
// through activateOnlineLobbyHit.

// onlineBlockedShade tints a blocklist tile's art red when it is blocked.
func onlineBlockedShade(dst *ebiten.Image, r Rect, blocked bool) {
	if blocked {
		uiFillRound(dst, r, 4, uiWithAlpha(uiDanger, 0.22))
	}
}

func onlineBoolIndex(v bool) int {
	if v {
		return 1
	}
	return 0
}

func onlineFixedMapLabel(mapValue int) string {
	if mapValue == 0 {
		return "RANDOM"
	}
	return strings.ToUpper(onlineMapNumberLabel(mapValue))
}

var onlineMapRotationLabels = []string{"FIXED MAP", "RANDOM GM1", "RANDOM GM2", "RANDOM ALL"}
var onlineWeaponModeLabels = []string{"ALL WEAPONS", "ONLY ONE", "BLOCKLIST"}

func (g *Game) buildOnlineRulesModal(ctx *uiCtx) {
	m := onlineRulesModalRect
	ctx.BeginModal(m, "ONLINE RULES")
	if ctx.drawing() {
		uiText(ctx.dst, "Rules the host enforces for every player. Changes are mirrored live.", false, uiSizeBody, uiTextMutedDk, m.X+14, m.Y+52, uiAlignLeft)
	}
	labelX, ctrlX := m.X+20, m.X+220
	row := func(i int) float64 { return m.Y + 78 + float64(i)*56 }
	label := func(i int, s string) {
		if ctx.drawing() {
			uiText(ctx.dst, s, true, uiSizeButton, uiTextDark, labelX, row(i)+18, uiAlignLeft)
		}
	}

	label(0, "MAP ROTATION")
	if v := ctx.Segmented("on.rules.maprot", Rect{X: ctrlX, Y: row(0), W: 440, H: 36}, onlineMapRotationLabels, g.onlineMapRotation, false,
		"MAP ROTATION", "Which arena each round uses when no map playlist is set on the MAPS tab."); v != g.onlineMapRotation {
		g.activateOnlineLobbyHit(menuHitOnlineRuleMapChoiceBase + v)
	}
	if g.onlineMapRotation == onlineMapFixed {
		if ctx.Button("on.rules.fixedmap", Rect{X: ctrlX + 452, Y: row(0), W: m.X + m.W - 20 - (ctrlX + 452), H: 36}, "MAP: "+onlineFixedMapLabel(g.customMap), uiButtonOpts{Size: uiSizeBody,
			InfoTitle: "FIXED MAP", InfoBody: "Pick the arena for every round."}) {
			g.activateOnlineLobbyHit(menuHitOnlineRuleFixedMapPicker)
		}
	}
	if ctx.drawing() && g.customSeriesPlaylistConfigured {
		uiText(ctx.dst, "The map playlist on the MAPS tab is set, so it decides the maps.", false, uiSizeHint, uiTextMutedDk, ctrlX, row(0)+46, uiAlignLeft)
	}

	label(1, "WEAPON RULE")
	if v := ctx.Segmented("on.rules.weapon", Rect{X: ctrlX, Y: row(1), W: 440, H: 36}, onlineWeaponModeLabels, g.onlineWeaponMode, false,
		"WEAPON RULE", "All weapons, a single weapon for everyone, or a custom blocklist."); v != g.onlineWeaponMode {
		g.activateOnlineLobbyHit(menuHitOnlineRuleWeaponChoiceBase + v)
	}
	sideR := Rect{X: ctrlX + 452, Y: row(1), W: m.X + m.W - 20 - (ctrlX + 452), H: 36}
	switch g.onlineWeaponMode {
	case onlineWeaponsOnlyOne:
		if ctx.Button("on.rules.onlyweapon", sideR, strings.ToUpper(weaponDisplayName(g.onlineOnlyWeapon)), uiButtonOpts{Size: uiSizeBody,
			InfoTitle: "ONLY WEAPON", InfoBody: "Pick the weapon every player uses."}) {
			g.activateOnlineLobbyHit(menuHitOnlineRuleOnlyWeaponPicker)
		}
	case onlineWeaponsBlocklist:
		if ctx.Button("on.rules.blocklist", sideR, fmt.Sprintf("BLOCKLIST - %d BLOCKED", g.blockedOnlineWeaponCount()), uiButtonOpts{Size: uiSizeBody,
			InfoTitle: "WEAPON BLOCKLIST", InfoBody: "Choose weapons that never appear."}) {
			g.activateOnlineLobbyHit(menuHitOnlineRuleBlocklist)
		}
	}

	label(2, "PERKS + ABILITIES")
	// Segmented OFF / ON: the Toggle pill is made for ink, the modal is paper.
	if v := ctx.Segmented("on.rules.perks", Rect{X: ctrlX, Y: row(2), W: 200, H: 36}, []string{"OFF", "ON"}, onlineBoolIndex(g.onlinePerksEnabled), false,
		"PERKS + ABILITIES", "Allow players' perks and abilities / gear in online matches."); v != onlineBoolIndex(g.onlinePerksEnabled) {
		g.activateOnlineLobbyHit(menuHitOnlineRulePerks)
	}
	if ctx.Button("on.rules.perkblock", Rect{X: ctrlX + 220, Y: row(2), W: 300, H: 36},
		fmt.Sprintf("PERK / ABILITY BLOCKLIST - %d", g.blockedOnlinePerkCount()+g.blockedOnlineAbilityCount()), uiButtonOpts{Size: uiSizeBody,
			InfoTitle: "PERK / ABILITY BLOCKLIST", InfoBody: "Blocked perks and abilities cannot be used by any player."}) {
		g.activateOnlineLobbyHit(menuHitOnlineRulePerkBlocklist)
	}

	label(3, "POWER-UPS")
	if v := ctx.Segmented("on.rules.powerups", Rect{X: ctrlX, Y: row(3), W: 200, H: 36}, []string{"OFF", "ON"}, onlineBoolIndex(g.onlinePowerupsEnabled), false,
		"POWER-UPS", "Power-up pickups in online matches."); v != onlineBoolIndex(g.onlinePowerupsEnabled) {
		g.activateOnlineLobbyHit(menuHitOnlineRulePowerups)
	}

	if ctx.drawing() {
		title, body := ctx.FocusInfo()
		if body == "" {
			title, body = "ONLINE RULES", "Hover or focus a rule to see what it does."
		}
		uiText(ctx.dst, title, true, uiSizeBody, uiTextDark, m.X+20, m.Y+m.H-82, uiAlignLeft)
		uiParagraph(ctx.dst, body, false, uiSizeBody, uiTextMutedDk, m.X+20, m.Y+m.H-60, m.W-40-220, 19, 2)
	}
	if ctx.Button("on.rules.done", Rect{X: m.X + m.W - 220, Y: m.Y + m.H - 64, W: 200, H: 44}, "DONE", uiButtonOpts{Kind: uiButtonPrimary}) {
		g.activateOnlineLobbyHit(menuHitOnlineOverlayBack)
	}
	ctx.EndModal()
	if ctx.Back() {
		g.activateOnlineLobbyHit(menuHitOnlineOverlayBack)
	}
}

// buildOnlinePagedGrid lays one page of tiles out and wires PREV / NEXT.
func buildOnlinePagedGrid(ctx *uiCtx, m Rect, count, page, maxPage, cols, rows int, prevCode, nextCode int, act func(code int), tile func(i int, r Rect)) {
	area := Rect{X: m.X + 20, Y: m.Y + 70, W: m.W - 40, H: m.H - 70 - 80}
	const gap = 8.0
	w := (area.W - float64(cols-1)*gap) / float64(cols)
	h := (area.H - float64(rows-1)*gap) / float64(rows)
	for i := 0; i < count; i++ {
		tile(i, Rect{X: area.X + float64(i%cols)*(w+gap), Y: area.Y + float64(i/cols)*(h+gap), W: w, H: h})
	}
	if maxPage <= 0 {
		return
	}
	y := m.Y + m.H - 64
	if ctx.Button("on.page.prev", Rect{X: m.X + 20, Y: y, W: 150, H: 44}, "PREV PAGE", uiButtonOpts{Size: uiSizeBody, Disabled: page <= 0}) {
		act(prevCode)
	}
	if ctx.drawing() {
		uiText(ctx.dst, fmt.Sprintf("PAGE %d / %d", page+1, maxPage+1), true, uiSizeBody, uiTextDark, m.X+250, y+22, uiAlignCenter)
	}
	if ctx.Button("on.page.next", Rect{X: m.X + 330, Y: y, W: 150, H: 44}, "NEXT PAGE", uiButtonOpts{Size: uiSizeBody, Disabled: page >= maxPage}) {
		act(nextCode)
	}
}

func (g *Game) buildOnlinePickerModal(ctx *uiCtx) {
	m := onlineRulesModalRect
	picker := g.onlineVisualPicker
	ctx.BeginModal(m, onlineVisualPickerTitle(picker))
	items := g.onlineVisualPagedItems()
	pageSize := g.onlineVisualPageSize()
	start := g.onlineVisualPage * pageSize
	count := len(items) - start
	if count > pageSize {
		count = pageSize
	}
	if count < 0 {
		count = 0
	}
	cols, rows := 6, 4
	if picker == onlineVisualFixedMap {
		cols, rows = 4, 3
	}
	buildOnlinePagedGrid(ctx, m, count, g.onlineVisualPage, g.onlineVisualMaxPage(), cols, rows, menuHitOnlineVisualPrevPage, menuHitOnlineVisualNextPage, g.activateOnlineLobbyHit,
		func(i int, r Rect) {
			n := items[start+i]
			label := strings.ToUpper(weaponDisplayName(n))
			selected := g.onlineOnlyWeapon == n
			art := func(dst *ebiten.Image, inner Rect) { g.drawOnlineWeaponThumb(dst, n, inner) }
			if picker == onlineVisualFixedMap {
				label = strings.ToUpper(onlineMapNumberLabel(n))
				selected = g.customMap == n
				art = func(dst *ebiten.Image, inner Rect) { g.drawOnlineMapThumb(dst, n, inner) }
			}
			if ctx.Tile(fmt.Sprintf("on.pick.%d", n), r, label, uiTileOpts{Selected: selected, InfoTitle: label, Art: art}) {
				g.activateOnlineLobbyHit(onlineVisualChoiceHit(i))
			}
		})
	if ctx.Button("on.pick.back", Rect{X: m.X + m.W - 220, Y: m.Y + m.H - 64, W: 200, H: 44}, "BACK TO RULES", uiButtonOpts{Kind: uiButtonPrimary}) {
		g.activateOnlineLobbyHit(menuHitOnlineOverlayBack)
	}
	ctx.EndModal()
	if ctx.Back() {
		g.activateOnlineLobbyHit(menuHitOnlineOverlayBack)
	}
}

func (g *Game) buildOnlineWeaponBlocklistModal(ctx *uiCtx) {
	m := onlineRulesModalRect
	ctx.BeginModal(m, fmt.Sprintf("WEAPON BLOCKLIST - %d BLOCKED", g.blockedOnlineWeaponCount()))
	if ctx.drawing() {
		uiText(ctx.dst, "Click a weapon to block or allow it. Blocked weapons are marked with X.", false, uiSizeBody, uiTextMutedDk, m.X+14, m.Y+52, uiAlignLeft)
	}
	weapons := onlineAllWeaponNumbers()
	start := g.onlineBlocklistPage * 24
	count := len(weapons) - start
	if count > 24 {
		count = 24
	}
	if count < 0 {
		count = 0
	}
	buildOnlinePagedGrid(ctx, m, count, g.onlineBlocklistPage, g.onlineBlocklistMaxPage(), 6, 4, menuHitOnlineBlockPrevPage, menuHitOnlineBlockNextPage, g.activateOnlineLobbyHit,
		func(i int, r Rect) {
			n := weapons[start+i]
			blocked := n > 0 && n < len(g.onlineBlockedWeapons) && g.onlineBlockedWeapons[n]
			label := strings.ToUpper(weaponDisplayName(n))
			info := "Allowed. Click to block."
			if blocked {
				info = "Blocked. Click to allow."
			}
			if ctx.Tile(fmt.Sprintf("on.block.%d", n), r, label, uiTileOpts{Locked: blocked, InfoTitle: label, InfoBody: info,
				Art: func(dst *ebiten.Image, inner Rect) {
					g.drawOnlineWeaponThumb(dst, n, inner)
					onlineBlockedShade(dst, inner, blocked)
				}}) {
				g.activateOnlineLobbyHit(menuHitOnlineBlockBase + start + i)
			}
		})
	if ctx.Button("on.block.back", Rect{X: m.X + m.W - 220, Y: m.Y + m.H - 64, W: 200, H: 44}, "BACK TO RULES", uiButtonOpts{Kind: uiButtonPrimary}) {
		g.activateOnlineLobbyHit(menuHitOnlineOverlayBack)
	}
	ctx.EndModal()
	if ctx.Back() {
		g.activateOnlineLobbyHit(menuHitOnlineOverlayBack)
	}
}

func (g *Game) buildOnlinePerkBlocklistModal(ctx *uiCtx) {
	m := onlineRulesModalRect
	ctx.BeginModal(m, fmt.Sprintf("PERK / ABILITY BLOCKLIST - %d BLOCKED", g.blockedOnlinePerkCount()+g.blockedOnlineAbilityCount()))
	if ctx.drawing() {
		uiText(ctx.dst, "Blocked perks and abilities (marked X) cannot be used by any player.", false, uiSizeBody, uiTextMutedDk, m.X+14, m.Y+52, uiAlignLeft)
		uiText(ctx.dst, "PERKS", true, uiSizeButton, uiTextDark, m.X+20, m.Y+82, uiAlignLeft)
	}
	const gap = 8.0
	perks := selectablePerkNumbers()
	cols := 5
	w := (m.W - 40 - float64(cols-1)*gap) / float64(cols)
	for i, perk := range perks {
		r := Rect{X: m.X + 20 + float64(i%cols)*(w+gap), Y: m.Y + 98 + float64(i/cols)*(72+gap), W: w, H: 72}
		blocked := perk < len(g.onlineBlockedPerks) && g.onlineBlockedPerks[perk]
		label := strings.ToUpper(onlinePerkName(perk))
		if ctx.Tile(fmt.Sprintf("on.perkblock.%d", perk), r, label, uiTileOpts{Locked: blocked, InfoTitle: label, InfoBody: loadoutPerkInfo[perk],
			Art: func(dst *ebiten.Image, inner Rect) {
				g.drawLoadoutPerkArt(dst, perk, inner)
				onlineBlockedShade(dst, inner, blocked)
			}}) {
			g.activateOnlineLobbyHit(menuHitOnlinePerkBase + perk)
		}
	}
	perkRows := (len(perks) + cols - 1) / cols
	y := m.Y + 98 + float64(perkRows)*(72+gap) + 8
	if ctx.drawing() {
		uiText(ctx.dst, "ABILITIES / GEAR", true, uiSizeButton, uiTextDark, m.X+20, y, uiAlignLeft)
	}
	abilities := realAbilityNumbers()
	acols := 7
	aw := (m.W - 40 - float64(acols-1)*gap) / float64(acols)
	for i, ability := range abilities {
		r := Rect{X: m.X + 20 + float64(i%acols)*(aw+gap), Y: y + 16 + float64(i/acols)*(56+gap), W: aw, H: 56}
		blocked := ability < len(g.onlineBlockedAbilities) && g.onlineBlockedAbilities[ability]
		label := strings.ToUpper(abilityName(ability))
		if ctx.Tile(fmt.Sprintf("on.abilityblock.%d", ability), r, label, uiTileOpts{Locked: blocked, InfoTitle: label,
			Art: func(dst *ebiten.Image, inner Rect) {
				drawAbilityIcon(dst, ability, inner.X+inner.W/2, inner.Y+inner.H/2, 10, 1)
				onlineBlockedShade(dst, inner, blocked)
			}}) {
			g.activateOnlineLobbyHit(menuHitOnlineAbilityBase + ability)
		}
	}
	if ctx.Button("on.perkblock.back", Rect{X: m.X + m.W - 220, Y: m.Y + m.H - 64, W: 200, H: 44}, "BACK TO RULES", uiButtonOpts{Kind: uiButtonPrimary}) {
		g.activateOnlineLobbyHit(menuHitOnlineOverlayBack)
	}
	ctx.EndModal()
	if ctx.Back() {
		g.activateOnlineLobbyHit(menuHitOnlineOverlayBack)
	}
}
