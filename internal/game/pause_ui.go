package game

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// Pause menu rebuilt with the Mayhem UI kit (phase 7), drawn over the frozen
// match: RESUME, RESTART, OPTIONS (volumes, fullscreen, screen shake) and the
// mode's quit action. ESC resumes (or leaves the OPTIONS view).
//
// Online: the host pauses the whole session as before (netSnapshot.Paused)
// and may restart a Custom Game round; a guest cannot pause or restart. ESC
// opens the guest's own menu (RESUME / OPTIONS / LEAVE SESSION) while the
// match keeps running, and a host pause is shown as a PAUSED BY HOST banner.

var (
	pauseCardRect   = Rect{X: 290, Y: 120, W: 320, H: 372}
	pauseOptionRect = Rect{X: 230, Y: 120, W: 440, H: 372}
)

func (g *Game) pauseIsGuest() bool { return g.netplay != nil && g.netplay.mode == netplayClient }

// pauseRestart reports whether RESTART is available and why not.
func (g *Game) pauseRestart() (bool, string) {
	switch {
	case g.pauseIsGuest():
		return false, "Only the host can restart."
	case g.zombie != nil:
		return false, "A zombie run ends on the results screen: use END RUN."
	case g.onlineMatchSetup && (g.netplay == nil || g.netplay.mode != netplayHost):
		return false, "Only the host can restart."
	}
	return true, "Start this round again from the beginning with the same players and rules."
}

// pauseContext is the line under the PAUSED title.
func (g *Game) pauseContext() string {
	switch {
	case g.gototest:
		return "GUN LIBRARY  /  TEST RANGE"
	case g.zombie != nil:
		return "ZOMBIE"
	case g.campaignMode:
		return fmt.Sprintf("CAMPAIGN  /  %s  /  MISSION %d", campaignSetNames[g.campaignSet()], g.campaignLevel)
	case g.onlineMatchSetup:
		return fmt.Sprintf("ONLINE  /  ROUND %d / %d", g.customSeriesRound, clampCustomSeriesRounds(g.customSeriesRounds))
	case g.customSeriesRound > 0:
		return fmt.Sprintf("VERSUS  /  ROUND %d / %d", g.customSeriesRound, clampCustomSeriesRounds(g.customSeriesRounds))
	}
	return "MATCH"
}

// pauseQuitLabel names the old "exit game" action for the current mode.
func (g *Game) pauseQuitLabel() (string, string) {
	switch {
	case g.zombie != nil:
		return "END RUN", "End the run now and see the results (the record still counts)."
	case g.gototest:
		return "BACK TO LIBRARY", "Leave the test range."
	case g.campaignMode && g.campaignQuitCompletes():
		return "FINISH MISSION", "Every enemy in this mission is down, so leaving now counts as completed."
	case g.campaignMode:
		return "QUIT MISSION", "Back to the mission list. The mission is not completed."
	case g.onlineMatchSetup:
		return "END MATCH", "Abort the match: everybody returns to the lobby, the session stays open."
	case g.customSeriesRound > 0:
		return "QUIT TO SETUP", "Abort the match and return to the Custom Game setup."
	}
	return "QUIT", "Leave the match."
}

// ---------------------------------------------------------------------------
// Actions (existing match flow)

func (g *Game) resumeFromPause() {
	g.paused = false
	g.pauseOptions = false
}

// quitFromPause is the old pause "exit game" button.
func (g *Game) quitFromPause() {
	g.paused = false
	g.pauseOptions = false
	if g.zombieSimulating() {
		// Leaving a zombie match ends it: results (and the record).
		g.finishZombieMatch()
		return
	}
	if !g.campaignMode && !g.gototest && g.customSeriesRound > 0 {
		g.abortCustomSeriesToSetup()
		return
	}
	target := screenPostGame
	if g.gototest {
		target = screenGunLibrary
	} else if g.campaignMode {
		target = screenCampaign
	}
	g.beginFade(target, fadePurposeEndGame)
}

// restartFromPause starts the same round again through the mode's own start
// function: startGunLibraryTest, start*CampaignMission or startCustomGame
// (the current series round, like startNextCustomSeriesRound).
func (g *Game) restartFromPause() {
	if ok, _ := g.pauseRestart(); !ok || g.fadeActive {
		return
	}
	test, campaign := g.gototest, g.campaignMode
	testDisabled, testRespawn, testFrame := g.testGunDisabled, g.testGunRespawn, g.testGunFrame
	g.resetMatchEndState()
	g.pauseOptions = false
	defer func() {
		if !g.fadeActive {
			// The start function refused (preconditions): keep the paused match.
			g.paused = true
			g.gototest = test
			g.testGunDisabled, g.testGunRespawn, g.testGunFrame = testDisabled, testRespawn, testFrame
			g.pauseUI.pushToast("CANNOT RESTART THIS MATCH", uiToastWarning)
		}
	}()
	switch {
	case test:
		g.startGunLibraryTest()
	case campaign:
		g.startSelectedCampaignMission()
	case g.customSeriesRound > 0:
		g.customSeriesStartingNext = true
		g.resetOnlinePostGameReady()
		g.startCustomGame()
		g.customSeriesStartingNext = false
	default:
		g.startCustomGame()
	}
}

// ---------------------------------------------------------------------------
// Update / draw

func (g *Game) updatePause() {
	g.runPauseUpdate(g.readUIInput(&g.pauseUI))
}

// runPauseUpdate is one update pass for a frame of input (tests drive it).
func (g *Game) runPauseUpdate(in uiInput) {
	ctx := g.pauseUI.begin(uiPassUpdate, nil, in)
	g.buildPause(ctx)
	ctx.end()
}

// runGuestMenuUpdate is the guest menu's update pass.
func (g *Game) runGuestMenuUpdate(in uiInput) {
	ctx := g.pauseUI.begin(uiPassUpdate, nil, in)
	g.buildGuestMenu(ctx)
	ctx.end()
}

// drawPauseOverlay draws the pause menu, the guest menu or the host-pause
// banner over the match.
func (g *Game) drawPauseOverlay(screen *ebiten.Image) {
	in := uiInput{X: g.uiLastCursorX, Y: g.uiLastCursorY}
	switch {
	case g.pauseIsGuest() && g.guestMenuOpen:
		ctx := g.pauseUI.begin(uiPassDraw, screen, in)
		g.buildGuestMenu(ctx)
		ctx.end()
	case g.pauseIsGuest() && g.paused:
		drawHostPauseBanner(screen)
	case g.paused:
		ctx := g.pauseUI.begin(uiPassDraw, screen, in)
		g.buildPause(ctx)
		ctx.end()
	}
}

func drawHostPauseBanner(dst *ebiten.Image) {
	uiFillRect(dst, Rect{W: ScreenWidth, H: ScreenHeight}, uiWithAlpha(uiScrim, 0.6))
	r := Rect{X: 300, Y: 250, W: 300, H: 100}
	uiFillRound(dst, r, 8, uiPaper)
	uiFillRect(dst, Rect{X: r.X, Y: r.Y + 46, W: r.W, H: 4}, uiOrange)
	uiText(dst, "PAUSED BY HOST", true, uiSizeSection, uiTextDark, r.X+r.W/2, r.Y+26, uiAlignCenter)
	uiText(dst, "The match continues when the host resumes.", false, uiSizeBody, uiTextMutedDk, r.X+r.W/2, r.Y+74, uiAlignCenter)
}

// buildPauseFrame draws the dimmed match and the menu card: paper for the
// button menus, ink for OPTIONS (its toggles and sliders are ink controls).
func (g *Game) buildPauseFrame(ctx *uiCtx, r Rect, title, sub string, dark bool) {
	if !ctx.drawing() {
		return
	}
	uiFillRect(ctx.dst, Rect{W: ScreenWidth, H: ScreenHeight}, uiScrim)
	titleClr, subClr := uiTextDark, uiTextMutedDk
	if dark {
		uiFillRound(ctx.dst, Rect{X: r.X + 2, Y: r.Y + 3, W: r.W, H: r.H}, 8, uiWithAlpha(uiInk900, 0.6))
		uiFillRound(ctx.dst, r, 8, uiInk700)
		titleClr, subClr = uiTextLight, uiTextMuted
	} else {
		ctx.Card(r, "")
	}
	uiText(ctx.dst, title, true, uiSizeTitle, titleClr, r.X+20, r.Y+34, uiAlignLeft)
	uiText(ctx.dst, uiFit(sub, true, uiSizeHint, r.W-40), true, uiSizeHint, subClr, r.X+21, r.Y+64, uiAlignLeft)
	uiFillRect(ctx.dst, Rect{X: r.X + 20, Y: r.Y + 78, W: r.W - 40, H: 3}, uiOrange)
}

func (g *Game) buildPause(ctx *uiCtx) {
	if g.pauseOptions {
		g.buildPauseOptions(ctx, "pause")
		return
	}
	r := pauseCardRect
	g.buildPauseFrame(ctx, r, "PAUSED", g.pauseContext(), false)
	x, w := r.X+20, r.W-40
	row := func(i int) Rect { return Rect{X: x, Y: r.Y + 100 + float64(i)*56, W: w, H: 46} }
	if ctx.Button("pause.resume", row(0), "RESUME", uiButtonOpts{Kind: uiButtonPrimary, InfoTitle: "RESUME", InfoBody: "Back to the match (ESC)."}) {
		g.resumeFromPause()
		return
	}
	canRestart, restartInfo := g.pauseRestart()
	if ctx.Button("pause.restart", row(1), "RESTART", uiButtonOpts{Disabled: !canRestart, InfoTitle: "RESTART", InfoBody: restartInfo}) {
		g.restartFromPause()
		return
	}
	if ctx.Button("pause.options", row(2), "OPTIONS", uiButtonOpts{InfoTitle: "OPTIONS", InfoBody: "Volumes, fullscreen and screen shake."}) {
		g.pauseOptions = true
		g.pauseUI.focus = ""
		return
	}
	label, info := g.pauseQuitLabel()
	if ctx.Button("pause.quit", row(3), label, uiButtonOpts{Kind: uiButtonDanger, InfoTitle: label, InfoBody: info}) {
		g.quitFromPause()
		return
	}
	if ctx.drawing() {
		title, body := ctx.FocusInfo()
		if title != "" {
			uiParagraph(ctx.dst, body, false, uiSizeHint, uiTextMutedDk, x, r.Y+r.H-36, w, 16, 2)
		}
	}
	ctx.KeyHintBar([]uiKeyHint{{"ESC", "Resume"}, {"UP/DOWN", "Move"}, {"ENTER", "Select"}})
	ctx.Toasts()
	if ctx.Back() {
		g.resumeFromPause()
	}
}

func (g *Game) buildGuestMenu(ctx *uiCtx) {
	if g.pauseOptions {
		g.buildPauseOptions(ctx, "guest")
		return
	}
	r := pauseCardRect
	sub := "ONLINE  /  THE MATCH KEEPS RUNNING"
	if g.paused {
		sub = "ONLINE  /  PAUSED BY HOST"
	}
	g.buildPauseFrame(ctx, r, "MENU", sub, false)
	x, w := r.X+20, r.W-40
	row := func(i int) Rect { return Rect{X: x, Y: r.Y + 100 + float64(i)*56, W: w, H: 46} }
	if ctx.Button("guest.resume", row(0), "RESUME", uiButtonOpts{Kind: uiButtonPrimary, InfoTitle: "RESUME"}) {
		g.closeGuestMenu()
		return
	}
	ctx.Button("guest.restart", row(1), "RESTART", uiButtonOpts{Disabled: true, InfoTitle: "RESTART", InfoBody: "Only the host can restart."})
	if ctx.Button("guest.options", row(2), "OPTIONS", uiButtonOpts{InfoTitle: "OPTIONS", InfoBody: "Volumes, fullscreen and screen shake."}) {
		g.pauseOptions = true
		g.pauseUI.focus = ""
		return
	}
	if ctx.Button("guest.leave", row(3), "LEAVE SESSION", uiButtonOpts{Kind: uiButtonDanger,
		InfoTitle: "LEAVE SESSION", InfoBody: "Disconnect from the host and return to the server browser."}) {
		g.closeGuestMenu()
		g.guestLeaveSession()
		return
	}
	if ctx.drawing() {
		uiParagraph(ctx.dst, "Your player stands still while this menu is open.", false, uiSizeHint, uiTextMutedDk, x, r.Y+r.H-36, w, 16, 2)
	}
	ctx.KeyHintBar([]uiKeyHint{{"ESC", "Resume"}, {"UP/DOWN", "Move"}, {"ENTER", "Select"}})
	if ctx.Back() {
		g.closeGuestMenu()
	}
}

func (g *Game) openGuestMenu() {
	g.guestMenuOpen = true
	g.pauseOptions = false
	g.pauseUI.reset()
}

func (g *Game) closeGuestMenu() {
	g.guestMenuOpen = false
	g.pauseOptions = false
}

// buildPauseOptions is the in-match OPTIONS view; it uses the Options
// screen's setters and saves the config the same way.
func (g *Game) buildPauseOptions(ctx *uiCtx, prefix string) {
	r := pauseOptionRect
	g.buildPauseFrame(ctx, r, "OPTIONS", "AUDIO AND DISPLAY  /  MORE IN THE MAIN MENU OPTIONS", true)
	x, w := r.X+20, r.W-40
	labelW := 150.0
	rowY := func(i int) float64 { return r.Y + 100 + float64(i)*52 }
	label := func(i int, s string) {
		if ctx.drawing() {
			uiText(ctx.dst, s, true, uiSizeButton, uiTextLight, x, rowY(i)+20, uiAlignLeft)
		}
	}
	control := func(i int) Rect { return Rect{X: x + labelW, Y: rowY(i), W: w - labelW, H: 40} }

	label(0, "MUSIC")
	sr := control(0)
	sr.W -= 52
	if v, changed, done := ctx.Slider(prefix+".music", sr, g.musicVolume, false, "MUSIC VOLUME", "Drag or use LEFT / RIGHT (5% steps)."); changed || done {
		g.setMusicVolume(v)
		if done {
			_ = g.saveConfig()
		}
	}
	ctx.Label(fmt.Sprintf("%d%%", int(g.musicVolume*100+0.5)), true, uiSizeButton, uiTextLight, x+w, rowY(0)+20, uiAlignRight)

	label(1, "SOUND EFFECTS")
	sr = control(1)
	sr.W -= 52
	if v, changed, done := ctx.Slider(prefix+".sfx", sr, g.soundVolume, false, "SOUND EFFECTS VOLUME", "Drag or use LEFT / RIGHT (5% steps)."); changed || done {
		g.setSoundVolume(v)
		if done {
			_ = g.saveConfig()
		}
	}
	ctx.Label(fmt.Sprintf("%d%%", int(g.soundVolume*100+0.5)), true, uiSizeButton, uiTextLight, x+w, rowY(1)+20, uiAlignRight)

	label(2, "FULLSCREEN")
	if on := ctx.Toggle(prefix+".full", control(2), ebiten.IsFullscreen(), false, "FULLSCREEN", "F11 toggles it anywhere."); on != ebiten.IsFullscreen() {
		g.setFullscreen(on)
		_ = g.saveConfig()
	}
	label(3, "SCREEN SHAKE")
	if on := ctx.Toggle(prefix+".shake", control(3), g.displayScreenShake, false, "SCREEN SHAKE", "Explosions jolt the screen."); on != g.displayScreenShake {
		g.displayScreenShake = on
		_ = g.saveConfig()
	}
	if ctx.Button(prefix+".optback", Rect{X: x, Y: r.Y + r.H - 64, W: w, H: 44}, "BACK", uiButtonOpts{InfoTitle: "BACK"}) {
		g.pauseOptions = false
		g.pauseUI.focus = ""
		return
	}
	ctx.KeyHintBar([]uiKeyHint{{"ESC", "Back"}, {"UP/DOWN", "Move"}, {"LEFT/RIGHT", "Change"}, {"ENTER", "Select"}})
	if ctx.Back() {
		g.pauseOptions = false
		g.pauseUI.focus = ""
	}
}

// guestZombieShopOpen reports whether the guest's own zombie shop is open.
func (g *Game) guestZombieShopOpen() bool {
	m := g.zombie
	if g.screen != screenGameplay || m == nil || m.host {
		return false
	}
	p := g.playerByID(g.onlineLocalPlayerID())
	return p != nil && g.zombieShopIsOpen(p)
}

// guestMenuMayOpen: ESC opens the guest menu only in a running match, never
// over the winner / fade sequence or on the press that closed the zombie shop.
func (g *Game) guestMenuMayOpen(shopWasOpen bool) bool {
	return g.screen == screenGameplay && !g.GameWin && !g.teamGameWin && !g.fadeActive && !shopWasOpen && !g.guestZombieShopOpen()
}

// campaignQuitCompletes: leaving a campaign match counts as completed only
// when the mission's win condition already holds (the GM2 tutorial has no
// opponents, so FINISH MISSION completes it).
func (g *Game) campaignQuitCompletes() bool {
	return g.campaignWinConditionMet()
}
