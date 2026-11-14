package game

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Options, rebuilt with the Mayhem UI kit (ui_kit_*.go): tabs GAME / DISPLAY /
// AUDIO / CONTROLS. One build function serves the update and draw passes, so
// every control's drawn rect is also its hit rect.

const (
	optionsTabGame = iota
	optionsTabDisplay
	optionsTabAudio
	optionsTabControls
	optionsTabCount
)

var optionsTabLabels = []string{"GAME", "DISPLAY", "AUDIO", "CONTROLS"}

var (
	optionsTabBarRect = Rect{X: 440, Y: 30, W: 440, H: 40}
	optionsInfoRect   = Rect{X: 640, Y: uiContentTop + 4, W: 240, H: 408}
	optionsModalRect  = Rect{X: 250, Y: 205, W: 400, H: 190}
)

// optionsRowRect is setting row i (label left, control right) on the
// GAME / DISPLAY / AUDIO tabs.
func optionsRowRect(i int) Rect {
	return Rect{X: uiMargin, Y: uiContentTop + 4 + float64(i)*46, W: 604, H: 40}
}

func optionsControlRect(i int) Rect {
	row := optionsRowRect(i)
	return Rect{X: 232, Y: row.Y + 3, W: 380, H: 34}
}

var optionControlLabels = [...]string{"JUMP", "LEFT", "DOWN", "RIGHT", "SHOOT", "BOMB", "ABILITY"}

func optionsPlayerCardRect(player int) Rect {
	return Rect{X: uiMargin + float64(player)*217, Y: uiContentTop + 4, W: 209, H: 408}
}

// optionsControlKeyRect is the rebinding button of one player's key.
func optionsControlKeyRect(player, index int) Rect {
	card := optionsPlayerCardRect(player)
	return Rect{X: card.X + 104, Y: card.Y + 52 + float64(index)*48, W: 92, H: 40}
}

func (g *Game) updateOptionsMenu() {
	in := g.readUIInput(&g.optionsUI)
	if g.optionsKeyCapturePlayer > 0 {
		// The capture modal swallows the keyboard: ESC cancels, any valid key is
		// bound (Enter/Space/arrows included, as in the original Symbol965).
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.cancelOptionsKeyCapture()
			return
		}
		for _, key := range inpututil.AppendJustPressedKeys(nil) {
			vk, ok := ebitenKeyToSourceVK(key)
			if !ok || !sourceOptionValidVK(vk) {
				continue
			}
			g.assignOptionsKey(vk)
			return
		}
		in = uiInput{X: in.X, Y: in.Y, Moved: in.Moved, Down: in.Down, Pressed: in.Pressed, Released: in.Released}
	}
	g.runOptionsUpdate(in)
}

// runOptionsUpdate is the update pass for one frame of input (tests drive it
// directly).
func (g *Game) runOptionsUpdate(in uiInput) {
	ctx := g.optionsUI.begin(uiPassUpdate, nil, in)
	g.buildOptions(ctx)
	ctx.end()
}

func (g *Game) drawOptionsMenu(screen *ebiten.Image) {
	ctx := g.optionsUI.begin(uiPassDraw, screen, uiInput{X: g.uiLastCursorX, Y: g.uiLastCursorY})
	g.buildOptions(ctx)
	ctx.end()
}

func (g *Game) leaveOptionsMenu() {
	// DefineSprite_991.btn_back is the only Options control that plays menu.wav.
	g.playSourceSFX("menu.wav", false)
	g.optionsKeyCapturePlayer = 0
	g.optionsKeyCaptureIndex = 0
	g.optionsUI.reset()
	g.beginFade(screenMainMenu, fadePurposeScreen)
}

func (g *Game) cancelOptionsKeyCapture() {
	g.optionsKeyCapturePlayer = 0
	g.optionsKeyCaptureIndex = 0
}

func (g *Game) assignOptionsKey(vk int) {
	player := g.optionsKeyCapturePlayer - 1
	index := g.optionsKeyCaptureIndex
	if player < 0 || player >= len(g.controlConfigs) || index < 0 || index >= len(optionControlLabels) {
		g.cancelOptionsKeyCapture()
		return
	}
	setControlByIndex(&g.controlConfigs[player], index, vk)
	if player < len(g.players) && g.players[player] != nil {
		g.players[player].Controls = g.controlConfigs[player]
	}
	_ = g.saveConfig()
	g.optionsUI.pushToast(fmt.Sprintf("PLAYER %d %s = %s", player+1, optionControlLabels[index], optionsKeyName(vk)), uiToastOK)
	g.cancelOptionsKeyCapture()
}

func (g *Game) buildOptions(ctx *uiCtx) {
	ctx.Page(0.9)
	ctx.Header("OPTIONS", "MAIN MENU  /  OPTIONS")
	if tab := ctx.TabBar("opt.tabs", optionsTabBarRect, optionsTabLabels, g.optionsTab); tab != g.optionsTab {
		g.optionsTab = tab
	}

	switch g.optionsTab {
	case optionsTabDisplay:
		g.buildOptionsDisplay(ctx)
	case optionsTabAudio:
		g.buildOptionsAudio(ctx)
	case optionsTabControls:
		g.buildOptionsControls(ctx)
	default:
		g.buildOptionsGame(ctx)
	}

	if ctx.Button("opt.back", uiActionBackRect(), "BACK", uiButtonOpts{InfoTitle: "BACK", InfoBody: "Return to the main menu. Settings are saved immediately."}) {
		g.leaveOptionsMenu()
		return
	}
	resetLabel, resetInfo := "RESET ALL SETTINGS", "Restore every setting except key bindings to its default."
	switch g.optionsTab {
	case optionsTabDisplay:
		resetLabel, resetInfo = "RESET DISPLAY", "Restore window size, scaling, filter, quality and the other display settings."
	case optionsTabAudio:
		resetLabel, resetInfo = "RESET AUDIO", "Music and sound effects back to 100%, loudness EQ off."
	case optionsTabControls:
		resetLabel, resetInfo = "RESET CONTROLS", "Restore the original keys of all four players."
	}
	if ctx.Button("opt.reset", uiActionPrimaryRect(230), resetLabel, uiButtonOpts{Kind: uiButtonDanger, InfoTitle: resetLabel, InfoBody: resetInfo}) {
		switch g.optionsTab {
		case optionsTabDisplay:
			g.resetDisplaySettings()
			g.optionsUI.pushToast("DISPLAY SETTINGS RESET", uiToastOK)
		case optionsTabAudio:
			g.resetAudioSettings()
			g.optionsUI.pushToast("AUDIO SETTINGS RESET", uiToastOK)
		case optionsTabControls:
			g.resetOptionControls()
			g.optionsUI.pushToast("CONTROLS RESET", uiToastOK)
		default:
			g.resetOptionSettings()
			g.optionsUI.pushToast("ALL SETTINGS RESET", uiToastOK)
		}
		_ = g.saveConfig()
	}
	ctx.KeyHintBar([]uiKeyHint{
		{"ESC", "Back"}, {"UP/DOWN", "Move"}, {"LEFT/RIGHT", "Change"}, {"ENTER", "Select"},
		{"Q", "Prev tab"}, {"E", "Next tab"}, {"F11", "Fullscreen"},
	})
	ctx.Toasts()

	if g.optionsKeyCapturePlayer > 0 {
		g.buildOptionsKeyCapture(ctx)
		return
	}
	if ctx.Back() {
		g.leaveOptionsMenu()
	}
}

// optionsRow draws a setting row's background and label; the row whose
// control is hovered or keyboard-focused gets the orange marker.
func (g *Game) optionsRow(ctx *uiCtx, i int, label, controlID string) Rect {
	row := optionsRowRect(i)
	if ctx.drawing() {
		fill := uiWithAlpha(uiInk700, 0.75)
		active := controlID != "" && (ctx.st.hot == controlID || (ctx.st.focus == controlID && ctx.st.keyboard))
		if active {
			fill = uiInk700
		}
		uiFillRound(ctx.dst, row, 5, fill)
		if active {
			uiFillRect(ctx.dst, Rect{X: row.X, Y: row.Y + 6, W: 4, H: row.H - 12}, uiOrange)
		}
		uiText(ctx.dst, label, true, uiSizeButton, uiTextLight, row.X+16, row.Y+row.H/2, uiAlignLeft)
	}
	return optionsControlRect(i)
}

func (g *Game) optionsInfo(ctx *uiCtx, fallbackTitle, fallbackBody string) {
	title, body := ctx.FocusInfo()
	if title == "" {
		title, body = fallbackTitle, fallbackBody
	}
	ctx.InfoPanel(optionsInfoRect, title, body)
}

func (g *Game) buildOptionsGame(ctx *uiCtx) {
	r := g.optionsRow(ctx, 0, "ALLOW CHEATS", "opt.cheats")
	if on := ctx.Toggle("opt.cheats", r, g.allowCheats, false, "ALLOW CHEATS",
		"Enables the developer keys: F1 collisions, F2/F3 change map, R resets the arena, F4 toggles AI for P2. Off by default."); on != g.allowCheats {
		g.allowCheats = on
		if !on {
			g.showCollisions = false
			g.menuDebug = false
		}
		_ = g.saveConfig()
	}

	r = g.optionsRow(ctx, 1, "GAME VERSION", "")
	ctx.Label(currentUpdateVersionLabel(), true, uiSizeButton, uiOrange, r.X+4, r.Y+r.H/2, uiAlignLeft)

	r = g.optionsRow(ctx, 2, "UPDATES", "opt.check")
	checkLabel := "CHECK FOR UPDATES"
	switch g.updateRuntime.phase {
	case updatePhaseChecking:
		checkLabel = "CHECKING..."
	case updatePhaseDownloading:
		checkLabel = "DOWNLOADING..."
	}
	busy := g.updateRuntime.phase == updatePhaseChecking || g.updateRuntime.phase == updatePhaseDownloading
	if ctx.Button("opt.check", Rect{X: r.X, Y: r.Y, W: 186, H: r.H}, checkLabel, uiButtonOpts{Disabled: busy, Size: uiSizeBody,
		InfoTitle: "CHECK FOR UPDATES", InfoBody: "Ask the release server for a newer version. Tagged releases also check automatically at start."}) {
		g.beginManualUpdateCheck()
	}
	available := g.settingsUpdateActionAvailable()
	updateLabel := "UPDATE NOW"
	if g.updateRuntime.phase == updatePhaseDownloading {
		updateLabel = "INSTALLING..."
	}
	if ctx.Button("opt.update", Rect{X: r.X + 194, Y: r.Y, W: 186, H: r.H}, updateLabel, uiButtonOpts{Kind: uiButtonPrimary, Disabled: !available, Size: uiSizeBody,
		InfoTitle: "UPDATE NOW", InfoBody: "Download and install the newer version, then restart. Available after a successful check."}) {
		g.beginUpdateDownload()
	}

	row := optionsRowRect(3)
	statusRow := Rect{X: row.X, Y: row.Y, W: row.W, H: 86}
	if ctx.drawing() {
		uiFillRound(ctx.dst, statusRow, 5, uiWithAlpha(uiInk700, 0.75))
		uiText(ctx.dst, "UPDATE STATUS", true, uiSizeHint, uiTextMuted, row.X+16, row.Y+18, uiAlignLeft)
		uiParagraph(ctx.dst, g.settingsUpdateStatus(), false, uiSizeBody, uiTextLight, row.X+16, row.Y+42, statusRow.W-32, 20, 2)
	}

	g.optionsInfo(ctx, "GAME", "General settings and updates. Use the tabs for display, audio and key bindings.")
}

func (g *Game) buildOptionsDisplay(ctx *uiCtx) {
	fullscreen := ebiten.IsFullscreen()
	r := g.optionsRow(ctx, 0, "DISPLAY MODE", "opt.mode")
	mode := 0
	if fullscreen {
		mode = 1
	}
	if v := ctx.Segmented("opt.mode", r, []string{"WINDOWED", "FULLSCREEN"}, mode, false, "DISPLAY MODE",
		"FULLSCREEN is borderless at the desktop resolution (the monitor mode is never changed). F11 toggles it anywhere."); v != mode {
		g.setFullscreen(v == 1)
		_ = g.saveConfig()
	}

	// Window size presets that fit on this monitor, plus CUSTOM.
	var labels []string
	var presetOf []int
	selected := -1
	current := g.displayPresetIndex()
	for i, p := range displayWindowPresets {
		if !g.displayPresetAvailable(i) {
			continue
		}
		if i == current {
			selected = len(labels)
		}
		labels = append(labels, p.label)
		presetOf = append(presetOf, i)
	}
	labels = append(labels, "CUSTOM")
	presetOf = append(presetOf, -1)
	if selected < 0 {
		selected = len(labels) - 1
	}
	sizeInfo := "Window " + g.displaySizeLabel(g.windowedWidth, g.windowedHeight) + ". Presets larger than this monitor are hidden; drag the window edge for a CUSTOM size."
	if fullscreen {
		sizeInfo = "Switch to WINDOWED to change the window size. " + sizeInfo
	}
	r = g.optionsRow(ctx, 1, "WINDOW SIZE", "opt.size")
	if v := ctx.Segmented("opt.size", r, labels, selected, fullscreen, "WINDOW SIZE", sizeInfo); v != selected {
		if p := presetOf[v]; p >= 0 {
			w, h := displayPresetSize(p)
			g.applyWindowSize(w, h)
		} else if g.displayCustomW > 0 {
			g.applyWindowSize(g.displayCustomW, g.displayCustomH)
		}
		_ = g.saveConfig()
	}

	r = g.optionsRow(ctx, 2, "SCALING", "opt.scaling")
	if v := ctx.Segmented("opt.scaling", r, displayScalingLabels, g.displayScaling, false, "SCALING",
		"FIT keeps the 3:2 picture with black bars. INTEGER zooms by whole numbers only (pixel-perfect, larger bars). STRETCH fills the window and distorts the picture."); v != g.displayScaling {
		g.displayScaling = v
		_ = g.saveConfig()
	}

	r = g.optionsRow(ctx, 3, "SCALING FILTER", "opt.filter")
	if v := ctx.Segmented("opt.filter", r, displayFilterLabels, g.displayFilter, false, "SCALING FILTER",
		"How the 900x600 picture is enlarged to the window. AUTO follows QUALITY. SHARP = nearest pixel, PIXEL = sharp with even pixels at any zoom, SMOOTH = soft, Flash-like."); v != g.displayFilter {
		g.displayFilter = v
		_ = g.saveConfig()
	}

	r = g.optionsRow(ctx, 4, "QUALITY", "opt.quality")
	if v := ctx.Segmented("opt.quality", r, displayQualityLabels, g.quality-1, false, "QUALITY",
		"LOW: sharp sprites and no extra explosion particles, like the original LOW. MEDIUM: smooth sprites. HIGH: smooth upscale and vector-drawn player bodies. Only affects this computer."); v != g.quality-1 {
		g.setQuality(v + 1)
		_ = g.saveConfig()
	}

	r = g.optionsRow(ctx, 5, "SCREEN SHAKE", "opt.shake")
	if on := ctx.Toggle("opt.shake", r, g.displayScreenShake, false, "SCREEN SHAKE",
		"Explosions jolt the screen like in the original Flash game. Turn off if it is uncomfortable. Only affects this computer."); on != g.displayScreenShake {
		g.displayScreenShake = on
		_ = g.saveConfig()
	}

	r = g.optionsRow(ctx, 6, "VSYNC", "opt.vsync")
	if on := ctx.Toggle("opt.vsync", r, g.displayVSync, false, "VSYNC",
		"Draw in step with the monitor refresh. OFF may tear. The game itself always runs at 30 ticks per second."); on != g.displayVSync {
		g.setVSync(on)
		_ = g.saveConfig()
	}

	r = g.optionsRow(ctx, 7, "POWER SAVE", "opt.power")
	if on := ctx.Toggle("opt.power", r, g.displayPowerSave, false, "POWER SAVE",
		"Redraw only when the game has advanced (30 frames per second). Saves battery; looks the same because the game runs at 30 ticks."); on != g.displayPowerSave {
		g.setPowerSave(on)
		_ = g.saveConfig()
	}

	monitors := g.displayMonitors()
	r = g.optionsRow(ctx, 8, "MONITOR", "opt.monitor")
	monitorLabel := "PRIMARY"
	if len(monitors) > 0 {
		i := g.displayMonitor
		if i < 0 || i >= len(monitors) {
			i = 0
		}
		w, h := monitors[i].Size()
		monitorLabel = fmt.Sprintf("%d / %d   %d x %d", i+1, len(monitors), w, h)
	}
	if d := ctx.Stepper("opt.monitor", r, monitorLabel, g.displayMonitor > 0, g.displayMonitor < len(monitors)-1, len(monitors) < 2,
		"MONITOR", "Which monitor the game window and fullscreen use."); d != 0 {
		g.setDisplayMonitor(g.displayMonitor + d)
		_ = g.saveConfig()
	}

	g.optionsInfo(ctx, "DISPLAY", "Window "+g.displaySizeLabel(g.windowedWidth, g.windowedHeight)+
		". Display settings only change this computer; online matches and the simulation are not affected.")
}

func (g *Game) buildOptionsAudio(ctx *uiCtx) {
	r := g.optionsRow(ctx, 0, "MUSIC", "opt.music")
	slider := Rect{X: r.X, Y: r.Y, W: r.W - 70, H: r.H}
	if v, changed, done := ctx.Slider("opt.music", slider, g.musicVolume, false, "MUSIC VOLUME", "Drag or use LEFT / RIGHT (5% steps)."); changed || done {
		g.setMusicVolume(v)
		if done {
			_ = g.saveConfig()
		}
	}
	ctx.Label(fmt.Sprintf("%d%%", int(g.musicVolume*100+0.5)), true, uiSizeButton, uiTextLight, r.X+r.W, r.Y+r.H/2, uiAlignRight)

	r = g.optionsRow(ctx, 1, "SOUND EFFECTS", "opt.sfx")
	slider = Rect{X: r.X, Y: r.Y, W: r.W - 70, H: r.H}
	if v, changed, done := ctx.Slider("opt.sfx", slider, g.soundVolume, false, "SOUND EFFECTS VOLUME", "Drag or use LEFT / RIGHT (5% steps)."); changed || done {
		g.setSoundVolume(v)
		if done {
			_ = g.saveConfig()
		}
	}
	ctx.Label(fmt.Sprintf("%d%%", int(g.soundVolume*100+0.5)), true, uiSizeButton, uiTextLight, r.X+r.W, r.Y+r.H/2, uiAlignRight)

	r = g.optionsRow(ctx, 2, "LOUDNESS EQ", "opt.eq")
	if on := ctx.Toggle("opt.eq", r, g.loudnessEQ, false, "LOUDNESS EQ",
		"Narrows the dynamic range of sound effects: quiet shots become louder and explosions softer."); on != g.loudnessEQ {
		g.loudnessEQ = on
		_ = g.saveConfig()
	}

	g.optionsInfo(ctx, "AUDIO", "Volumes are saved when you release the slider.")
}

func (g *Game) buildOptionsControls(ctx *uiCtx) {
	uses := map[int]int{}
	for p := range g.controlConfigs {
		for i := range optionControlLabels {
			uses[controlByIndex(g.controlConfigs[p], i)]++
		}
	}
	for p := 0; p < 4; p++ {
		card := optionsPlayerCardRect(p)
		ctx.Card(card, fmt.Sprintf("PLAYER %d", p+1))
		for i, label := range optionControlLabels {
			kr := optionsControlKeyRect(p, i)
			ctx.Label(label, true, uiSizeBody, uiTextDark, card.X+16, kr.Y+kr.H/2, uiAlignLeft)
			vk := controlByIndex(g.controlConfigs[p], i)
			id := fmt.Sprintf("opt.key.%d.%d", p, i)
			info := fmt.Sprintf("Player %d %s: %s. Click or press ENTER, then press the new key.", p+1, strings.ToLower(label), optionsKeyName(vk))
			if uses[vk] > 1 {
				info += " This key is also used by another binding."
			}
			if ctx.Button(id, kr, "", uiButtonOpts{InfoTitle: fmt.Sprintf("P%d %s", p+1, label), InfoBody: info}) {
				g.optionsKeyCapturePlayer = p + 1
				g.optionsKeyCaptureIndex = i
			}
			if ctx.drawing() {
				clr := uiTextLight
				if uses[vk] > 1 {
					clr = uiOrange
				}
				drawOptionsKeyGlyph(ctx.dst, kr, vk, clr)
			}
		}
	}
}

func (g *Game) buildOptionsKeyCapture(ctx *uiCtx) {
	p, i := g.optionsKeyCapturePlayer-1, g.optionsKeyCaptureIndex
	if p < 0 || p >= len(g.controlConfigs) || i < 0 || i >= len(optionControlLabels) {
		g.cancelOptionsKeyCapture()
		return
	}
	m := optionsModalRect
	ctx.BeginModal(m, fmt.Sprintf("PLAYER %d  -  %s", p+1, optionControlLabels[i]))
	ctx.Label("Press the new key.", false, uiSizeButton, uiTextDark, m.X+16, m.Y+66, uiAlignLeft)
	ctx.Label("Current: "+optionsKeyName(controlByIndex(g.controlConfigs[p], i))+"     ESC cancels", false, uiSizeBody, uiTextMutedDk, m.X+16, m.Y+96, uiAlignLeft)
	if ctx.Button("opt.capture.cancel", Rect{X: m.X + m.W - 156, Y: m.Y + m.H - 56, W: 140, H: 40}, "CANCEL", uiButtonOpts{}) {
		g.cancelOptionsKeyCapture()
	}
	ctx.EndModal()
}

// optionsKeyName is the display name of a Flash key code.
func optionsKeyName(vk int) string {
	switch vk {
	case 37:
		return "LEFT"
	case 38:
		return "UP"
	case 39:
		return "RIGHT"
	case 40:
		return "DOWN"
	}
	name := strings.ToUpper(sourceVKName(vk))
	if name == "" {
		return "?"
	}
	return name
}

// drawOptionsKeyGlyph draws a key name, using arrow glyphs for arrow keys.
func drawOptionsKeyGlyph(dst *ebiten.Image, r Rect, vk int, clr color.NRGBA) {
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	switch vk {
	case 37:
		uiDrawArrow(dst, cx, cy, 18, uiArrowLeft, clr)
	case 38:
		uiDrawArrow(dst, cx, cy, 18, uiArrowUp, clr)
	case 39:
		uiDrawArrow(dst, cx, cy, 18, uiArrowRight, clr)
	case 40:
		uiDrawArrow(dst, cx, cy, 18, uiArrowDown, clr)
	default:
		uiTextIn(dst, optionsKeyName(vk), true, uiSizeButton, clr, r, uiAlignCenter, 6)
	}
}

func clampOptionVolume(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func optionSliderValueFromX(r Rect, x float64) float64 {
	if r.W <= 0 {
		return 0
	}
	return clampOptionVolume((x - r.X) / r.W)
}

func (g *Game) setMusicVolume(v float64) {
	g.musicVolume = clampOptionVolume(v)
	g.musicOn = g.musicVolume > 0
	if g.audioStarted {
		g.syncSourceMusic()
	}
}

func (g *Game) setSoundVolume(v float64) {
	g.soundVolume = clampOptionVolume(v)
	g.soundOn = g.soundVolume > 0
}

func (g *Game) toggleOptionsFullscreen() {
	if ebiten.IsFullscreen() {
		ebiten.SetFullscreen(false)
		if g.windowedWidth >= displayMinWindowW && g.windowedHeight >= displayMinWindowH {
			ebiten.SetWindowSize(g.windowedWidth, g.windowedHeight)
		}
		if g.windowPositionSaved {
			ebiten.SetWindowPosition(g.windowedX, g.windowedY)
		}
		return
	}
	g.captureWindowedState()
	ebiten.SetFullscreen(true)
}

func (g *Game) resetOptionControls() {
	for i := range g.controlConfigs {
		g.controlConfigs[i] = OriginalControls(i + 1)
	}
	for i, p := range g.players {
		if p != nil && i < len(g.controlConfigs) {
			p.Controls = g.controlConfigs[i]
		}
	}
	g.optionsKeyCapturePlayer = 0
	g.optionsKeyCaptureIndex = 0
}

func (g *Game) resetAudioSettings() {
	defaults := DefaultAppConfig()
	g.loudnessEQ = defaults.LoudnessEQ
	g.setSoundVolume(defaults.SoundVolume)
	g.setMusicVolume(defaults.MusicVolume)
}

func (g *Game) resetDisplaySettings() {
	defaults := DefaultAppConfig()
	g.setQuality(defaults.Quality)
	g.displayScaling = defaults.ScalingMode
	g.displayFilter = defaults.ScalingFilter
	g.displayScreenShake = defaults.ScreenShake
	g.setVSync(defaults.VSync)
	g.setPowerSave(defaults.PowerSave)
	if g.persistent {
		ebiten.SetFullscreen(false)
		ebiten.SetWindowSize(defaults.WindowWidth, defaults.WindowHeight)
	}
	g.windowedWidth = defaults.WindowWidth
	g.windowedHeight = defaults.WindowHeight
	g.windowPositionSaved = false
}

// resetOptionSettings restores every setting except key bindings.
func (g *Game) resetOptionSettings() {
	defaults := DefaultAppConfig()
	g.allowCheats = defaults.AllowCheats
	g.showCollisions = false
	g.menuDebug = false
	g.resetAudioSettings()
	g.resetDisplaySettings()
}

func controlByIndex(c Controls, index int) int {
	switch index {
	case 0:
		return c.Up
	case 1:
		return c.Left
	case 2:
		return c.Down
	case 3:
		return c.Right
	case 4:
		return c.Shoot
	case 5:
		return c.Grenade
	case 6:
		return c.Ability
	}
	return 0
}

func setControlByIndex(c *Controls, index, value int) {
	switch index {
	case 0:
		c.Up = value
	case 1:
		c.Left = value
	case 2:
		c.Down = value
	case 3:
		c.Right = value
	case 4:
		c.Shoot = value
	case 5:
		c.Grenade = value
	case 6:
		c.Ability = value
	}
}

func sourceOptionValidVK(v int) bool {
	if (v >= 65 && v <= 90) || (v >= 48 && v <= 57) {
		return true
	}
	switch v {
	case 8, 13, 16, 17, 20, 33, 34, 35, 36, 37, 38, 39, 40, 45, 46, 145,
		96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 109, 110, 111,
		113, 115, 118, 119, 120, 121, 123, 186, 187, 188, 189, 190, 191, 192, 219, 220, 221, 222:
		return true
	}
	return false
}

func ebitenKeyToSourceVK(k ebiten.Key) (int, bool) {
	if k >= ebiten.KeyA && k <= ebiten.KeyZ {
		return 65 + int(k-ebiten.KeyA), true
	}
	if k >= ebiten.KeyDigit0 && k <= ebiten.KeyDigit9 {
		return 48 + int(k-ebiten.KeyDigit0), true
	}
	switch k {
	case ebiten.KeyBackspace:
		return 8, true
	case ebiten.KeyEnter:
		return 13, true
	case ebiten.KeyShiftLeft, ebiten.KeyShiftRight:
		return 16, true
	case ebiten.KeyControlLeft, ebiten.KeyControlRight:
		return 17, true
	case ebiten.KeyCapsLock:
		return 20, true
	case ebiten.KeyPageUp:
		return 33, true
	case ebiten.KeyPageDown:
		return 34, true
	case ebiten.KeyEnd:
		return 35, true
	case ebiten.KeyHome:
		return 36, true
	case ebiten.KeyArrowLeft:
		return 37, true
	case ebiten.KeyArrowUp:
		return 38, true
	case ebiten.KeyArrowRight:
		return 39, true
	case ebiten.KeyArrowDown:
		return 40, true
	case ebiten.KeyInsert:
		return 45, true
	case ebiten.KeyDelete:
		return 46, true
	case ebiten.KeyScrollLock:
		return 145, true
	case ebiten.KeyNumpad0:
		return 96, true
	case ebiten.KeyNumpad1:
		return 97, true
	case ebiten.KeyNumpad2:
		return 98, true
	case ebiten.KeyNumpad3:
		return 99, true
	case ebiten.KeyNumpad4:
		return 100, true
	case ebiten.KeyNumpad5:
		return 101, true
	case ebiten.KeyNumpad6:
		return 102, true
	case ebiten.KeyNumpad7:
		return 103, true
	case ebiten.KeyNumpad8:
		return 104, true
	case ebiten.KeyNumpad9:
		return 105, true
	case ebiten.KeyNumpadMultiply:
		return 106, true
	case ebiten.KeyNumpadAdd:
		return 107, true
	case ebiten.KeyNumpadSubtract:
		return 109, true
	case ebiten.KeyNumpadDecimal:
		return 110, true
	case ebiten.KeyNumpadDivide:
		return 111, true
	case ebiten.KeyF2:
		return 113, true
	case ebiten.KeyF4:
		return 115, true
	case ebiten.KeyF7:
		return 118, true
	case ebiten.KeyF8:
		return 119, true
	case ebiten.KeyF9:
		return 120, true
	case ebiten.KeyF10:
		return 121, true
	case ebiten.KeyF12:
		return 123, true
	case ebiten.KeySemicolon:
		return 186, true
	case ebiten.KeyEqual:
		return 187, true
	case ebiten.KeyComma:
		return 188, true
	case ebiten.KeyMinus:
		return 189, true
	case ebiten.KeyPeriod:
		return 190, true
	case ebiten.KeySlash:
		return 191, true
	case ebiten.KeyBackquote:
		return 192, true
	case ebiten.KeyBracketLeft:
		return 219, true
	case ebiten.KeyBackslash:
		return 220, true
	case ebiten.KeyBracketRight:
		return 221, true
	case ebiten.KeyQuote:
		return 222, true
	}
	return 0, false
}

func sourceVKName(v int) string {
	if (v >= 65 && v <= 90) || (v >= 48 && v <= 57) {
		return string(rune(v))
	}
	names := map[int]string{
		8: "BS", 13: "Entr", 16: "Shft", 17: "Ctrl", 20: "Cpsl", 33: "PgU", 34: "PgD", 35: "End", 36: "Hom",
		37: "←", 38: "↑", 39: "→", 40: "↓", 45: "Ins", 46: "Del", 145: "Scrl",
		96: "np0", 97: "np1", 98: "np2", 99: "np3", 100: "np4", 101: "np5", 102: "np6", 103: "np7", 104: "np8", 105: "np9",
		106: "np*", 107: "np+", 109: "np-", 110: "np.", 111: "np/", 113: "F2", 115: "F4", 118: "F7", 119: "F8", 120: "F9", 121: "F10", 123: "F12",
		186: ";", 187: "=", 188: ",", 189: "-", 190: ".", 191: "/", 192: "`", 219: "[", 220: "\\", 221: "]", 222: "'",
	}
	return names[v]
}
