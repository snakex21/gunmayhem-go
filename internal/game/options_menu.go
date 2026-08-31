package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	optionHitNone = iota
	optionHitMusicOn
	optionHitMusicOff
	optionHitSoundOn
	optionHitSoundOff
	optionHitQualityLow
	optionHitQualityMedium
	optionHitQualityHigh
	optionHitBack
	optionHitLoudnessEQ
	optionHitFullscreen
	optionHitResetSettings
	optionHitResetControls
	optionHitAllowCheats
	optionHitCheckUpdates
	optionHitUpdateNow
	optionHitControlBase = 100
)

const (
	optionSliderNone = iota
	optionSliderMusic
	optionSliderSFX
)

var optionQualityButtons = [...]struct {
	hit   int
	label string
	rect  Rect
}{
	{optionHitQualityLow, "LOW", Rect{X: 445, Y: 486, W: 72, H: 28}},
	{optionHitQualityMedium, "MEDIUM", Rect{X: 522, Y: 486, W: 72, H: 28}},
	{optionHitQualityHigh, "HIGH", Rect{X: 599, Y: 486, W: 72, H: 28}},
}

var (
	optionMusicSliderRect   = Rect{X: 115, Y: 452, W: 220, H: 12}
	optionSFXSliderRect     = Rect{X: 115, Y: 485, W: 220, H: 12}
	optionLoudnessRect      = Rect{X: 385, Y: 451, W: 140, H: 28}
	optionFullscreenRect    = Rect{X: 535, Y: 451, W: 150, H: 28}
	optionResetSettingsRect = Rect{X: 705, Y: 451, W: 150, H: 28}
	optionResetControlsRect = Rect{X: 705, Y: 486, W: 150, H: 28}
	optionAllowCheatsRect   = Rect{X: 535, Y: 521, W: 150, H: 28}
	optionBackRect          = Rect{X: 705, Y: 521, W: 150, H: 28}
	optionCheckUpdatesRect  = Rect{X: 535, Y: 558, W: 150, H: 28}
	optionUpdateNowRect     = Rect{X: 705, Y: 558, W: 150, H: 28}
)

func optionsHitAt(x, y float64) int {
	for _, button := range optionQualityButtons {
		if button.rect.Contains(x, y) {
			return button.hit
		}
	}
	switch {
	case optionLoudnessRect.Contains(x, y):
		return optionHitLoudnessEQ
	case optionFullscreenRect.Contains(x, y):
		return optionHitFullscreen
	case optionResetSettingsRect.Contains(x, y):
		return optionHitResetSettings
	case optionResetControlsRect.Contains(x, y):
		return optionHitResetControls
	case optionAllowCheatsRect.Contains(x, y):
		return optionHitAllowCheats
	case optionBackRect.Contains(x, y):
		return optionHitBack
	case optionCheckUpdatesRect.Contains(x, y):
		return optionHitCheckUpdates
	case optionUpdateNowRect.Contains(x, y):
		return optionHitUpdateNow
	}
	return optionHitNone
}

func optionSliderHitAt(x, y float64) int {
	if optionMusicSliderRect.Contains(x, y) {
		return optionSliderMusic
	}
	if optionSFXSliderRect.Contains(x, y) {
		return optionSliderSFX
	}
	return optionSliderNone
}

func (g *Game) updateOptionsMenu() {
	if g.optionsKeyCapturePlayer > 0 {
		// Developed UX: ESC cancels key capture instead of becoming a binding or
		// trapping the user in the original Flash lockup.
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.optionsKeyCapturePlayer = 0
			g.optionsKeyCaptureIndex = 0
			g.optionsPressed = optionHitNone
			return
		}
		for _, key := range inpututil.AppendJustPressedKeys(nil) {
			vk, ok := ebitenKeyToSourceVK(key)
			if !ok || !sourceOptionValidVK(vk) {
				continue
			}
			player := g.optionsKeyCapturePlayer - 1
			index := g.optionsKeyCaptureIndex
			setControlByIndex(&g.controlConfigs[player], index, vk)
			_ = g.saveConfig()
			g.optionsKeyCapturePlayer = 0
			g.optionsKeyCaptureIndex = 0
			break
		}
		return
	}

	mx, my := ebiten.CursorPosition()
	x, y := float64(mx), float64(my)
	if g.optionsDraggingSlider != optionSliderNone {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			g.setOptionSliderFromX(g.optionsDraggingSlider, x)
			return
		}
		g.setOptionSliderFromX(g.optionsDraggingSlider, x)
		g.optionsDraggingSlider = optionSliderNone
		_ = g.saveConfig()
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.optionsPressed = optionHitNone
		g.beginFade(screenMainMenu, fadePurposeScreen)
		return
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if slider := optionSliderHitAt(x, y); slider != optionSliderNone {
			g.optionsDraggingSlider = slider
			g.setOptionSliderFromX(slider, x)
			g.optionsPressed = optionHitNone
			return
		}
		if player, index, ok := optionsControlHitAt(x, y); ok {
			// Symbol965 uses onPress and immediately disables every other button
			// until keyListener receives a valid Key.getCode().
			g.optionsKeyCapturePlayer = player + 1
			g.optionsKeyCaptureIndex = index
			g.optionsPressed = optionHitNone
			return
		}
	}
	hit := optionsHitAt(x, y)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.optionsPressed = hit
	}
	if !inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		return
	}
	pressed := g.optionsPressed
	g.optionsPressed = optionHitNone
	if pressed == optionHitNone || pressed != hit {
		return
	}
	switch pressed {
	case optionHitLoudnessEQ:
		g.loudnessEQ = !g.loudnessEQ
		_ = g.saveConfig()
	case optionHitFullscreen:
		g.toggleOptionsFullscreen()
		_ = g.saveConfig()
	case optionHitResetSettings:
		g.resetOptionSettings()
		_ = g.saveConfig()
	case optionHitResetControls:
		g.resetOptionControls()
		_ = g.saveConfig()
	case optionHitAllowCheats:
		g.allowCheats = !g.allowCheats
		if !g.allowCheats {
			g.showCollisions = false
			g.menuDebug = false
		}
		_ = g.saveConfig()
	case optionHitCheckUpdates:
		g.beginManualUpdateCheck()
	case optionHitUpdateNow:
		if g.settingsUpdateActionAvailable() {
			g.beginUpdateDownload()
		}
	case optionHitQualityLow:
		g.quality = 1
		setSourceRenderQuality(1)
		g.assets.invalidateQualityDependentPlayerRasters()
		ebiten.SetScreenFilterEnabled(false)
		_ = g.saveConfig()
	case optionHitQualityMedium:
		g.quality = 2
		setSourceRenderQuality(2)
		g.assets.invalidateQualityDependentPlayerRasters()
		ebiten.SetScreenFilterEnabled(true)
		_ = g.saveConfig()
	case optionHitQualityHigh:
		g.quality = 3
		setSourceRenderQuality(3)
		g.assets.invalidateQualityDependentPlayerRasters()
		ebiten.SetScreenFilterEnabled(true)
		_ = g.saveConfig()
	case optionHitBack:
		// DefineSprite_991.btn_back is the only Options control that plays menu.wav.
		g.playSourceSFX("menu.wav", false)
		g.beginFade(screenMainMenu, fadePurposeScreen)
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

func (g *Game) setOptionSliderFromX(slider int, x float64) {
	switch slider {
	case optionSliderMusic:
		g.musicVolume = optionSliderValueFromX(optionMusicSliderRect, x)
		g.musicOn = g.musicVolume > 0
		if g.audioStarted {
			g.syncSourceMusic()
		}
	case optionSliderSFX:
		g.soundVolume = optionSliderValueFromX(optionSFXSliderRect, x)
		g.soundOn = g.soundVolume > 0
	}
}

func (g *Game) toggleOptionsFullscreen() {
	if ebiten.IsFullscreen() {
		ebiten.SetFullscreen(false)
		if g.windowedWidth >= 480 && g.windowedHeight >= 270 {
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

func (g *Game) resetOptionSettings() {
	defaults := DefaultAppConfig()
	g.musicVolume = defaults.MusicVolume
	g.soundVolume = defaults.SoundVolume
	g.musicOn = true
	g.soundOn = true
	g.loudnessEQ = defaults.LoudnessEQ
	g.allowCheats = defaults.AllowCheats
	g.quality = defaults.Quality
	setSourceRenderQuality(g.quality)
	g.assets.invalidateQualityDependentPlayerRasters()
	ebiten.SetScreenFilterEnabled(true)
	ebiten.SetFullscreen(false)
	ebiten.SetWindowSize(defaults.WindowWidth, defaults.WindowHeight)
	g.windowedWidth = defaults.WindowWidth
	g.windowedHeight = defaults.WindowHeight
	g.windowPositionSaved = false
	if g.audioStarted {
		g.syncSourceMusic()
	}
}

var optionControlPanelX = [...]float64{50, 270, 490, 700}

var optionControlKeyPositions = [...]struct{ x, y float64 }{
	{55.15, 101.7}, // up
	{16.65, 140.2}, // left
	{55.15, 140.2}, // down
	{93.65, 140.2}, // right
	{40.35, 247.2}, // shoot
	{78.85, 247.2}, // bomb
}

func optionsControlHitAt(x, y float64) (player, index int, ok bool) {
	for p, px := range optionControlPanelX {
		for i, pos := range optionControlKeyPositions {
			r := Rect{X: px + pos.x, Y: 83.85 + pos.y, W: 33, H: 33} // Symbol965 exact bounds
			if r.Contains(x, y) {
				return p, i, true
			}
		}
	}
	return 0, 0, false
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

func (g *Game) drawOptionControlPanels(screen *ebiten.Image) {
	mx, my := ebiten.CursorPosition()
	hoverPlayer, hoverIndex, hoverOK := optionsControlHitAt(float64(mx), float64(my))
	labels := [...]struct {
		text string
		x, y float64
	}{
		{"JUMP", 55.0, 78.35},
		{"LEFT", 10.5, 117.7},
		{"RIGHT", 99.85, 117.7},
		{"DOWN", 52.5, 177.35},
		{"SHOOT", 31.95, 223.85},
		{"BOMB", 83.6, 283.7},
	}
	for p, px := range optionControlPanelX {
		panelY := 83.85
		if g.assets.OptionPanelBase != nil {
			drawSourceRaster(screen, g.assets.OptionPanelBase, px, panelY, 1, 1, 1)
		}
		drawSourceMenuText(screen, "Player "+string(rune('1'+p)), menuFontCondensed, 26.5,
			color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xff}, px+5, panelY+20)
		for _, label := range labels {
			drawSourceMenuText(screen, label.text, menuFontCondensed, 17.65, color.Black,
				px+label.x, panelY+label.y)
		}
		for i, pos := range optionControlKeyPositions {
			frame := 1
			if g.optionsKeyCapturePlayer == 0 && hoverOK && hoverPlayer == p && hoverIndex == i {
				frame = 2
			}
			keyX := px + pos.x
			keyY := panelY + pos.y
			if keyFrame := g.assets.OptionKeyFrames[frame]; keyFrame != nil {
				drawSourceRaster(screen, keyFrame, keyX, keyY, 1, 1, 1)
			}
			vk := controlByIndex(g.controlConfigs[p], i)
			if glyph := g.assets.OptionArrowGlyphs[vk]; glyph != nil {
				// Symbol965 arrow graphics are Symbols961..964 at 5% scale.
				drawSourceRaster(screen, glyph, keyX, keyY, 0.05, 0.05, 1)
				continue
			}
			name := sourceVKName(vk)
			// Dynamic keytext is a 32.55 px centered Tw Cen MT Condensed Extra Bold field.
			textX := keyX + 16.5 - float64(len([]rune(name)))*3.6
			drawSourceMenuText(screen, name, menuFontCondensedExtraBold, 19, color.Black, textX, keyY+5.95)
		}
	}
}

func drawOptionSlider(screen *ebiten.Image, r Rect, value float64) {
	value = clampOptionVolume(value)
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, color.NRGBA{R: 0x52, G: 0x58, B: 0x63, A: 0xff})
	ebitenutil.DrawRect(screen, r.X+2, r.Y+2, r.W-4, r.H-4, color.NRGBA{R: 0x20, G: 0x24, B: 0x2b, A: 0xff})
	fillW := (r.W - 4) * value
	if fillW > 0 {
		ebitenutil.DrawRect(screen, r.X+2, r.Y+2, fillW, r.H-4, color.NRGBA{R: 0x00, G: 0x66, B: 0xff, A: 0xff})
	}
	knobX := r.X + 2 + (r.W-4)*value
	ebitenutil.DrawRect(screen, knobX-3, r.Y-4, 6, r.H+8, color.NRGBA{R: 0xf4, G: 0xf4, B: 0xf4, A: 0xff})
}

func drawOptionActionButton(screen *ebiten.Image, r Rect, label string, active bool) {
	border := color.NRGBA{R: 0x44, G: 0x49, B: 0x54, A: 0xff}
	fill := color.NRGBA{R: 0x2d, G: 0x32, B: 0x3a, A: 0xff}
	if active {
		fill = color.NRGBA{R: 0xd8, G: 0x5a, B: 0x00, A: 0xff}
	}
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, border)
	ebitenutil.DrawRect(screen, r.X+2, r.Y+2, r.W-4, r.H-4, fill)
	drawSourceMenuText(screen, label, menuFontCondensedExtraBold, 12.5, color.White, r.X+9, r.Y+7)
}

func drawOptionQualityButton(screen *ebiten.Image, r Rect, label string, active bool) {
	border := color.NRGBA{R: 0x44, G: 0x49, B: 0x54, A: 0xff}
	fill := color.NRGBA{R: 0x3a, G: 0x3d, B: 0x43, A: 0xff}
	textColor := color.White
	if active {
		fill = color.NRGBA{R: 0xff, G: 0x7b, B: 0x00, A: 0xff}
		textColor = color.Black
	}
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, border)
	ebitenutil.DrawRect(screen, r.X+2, r.Y+2, r.W-4, r.H-4, fill)
	textX := r.X + r.W/2 - float64(len([]rune(label)))*3.3
	drawSourceMenuText(screen, label, menuFontCondensedExtraBold, 12.5, textColor, textX, r.Y+7)
}

func (g *Game) drawDevelopedOptionSettings(screen *ebiten.Image) {
	panel := Rect{X: 40, Y: 430, W: 820, H: 164}
	ebitenutil.DrawRect(screen, panel.X, panel.Y, panel.W, panel.H, color.NRGBA{R: 0x17, G: 0x19, B: 0x1d, A: 0xfa})
	ebitenutil.DrawRect(screen, panel.X, panel.Y, panel.W, 2, color.NRGBA{R: 0x4d, G: 0x4f, B: 0x54, A: 0xff})

	drawSourceMenuText(screen, "AUDIO", menuFontCondensedExtraBold, 13.5, color.White, 55, 437)
	drawSourceMenuText(screen, "MUSIC", menuFontCondensedExtraBold, 12.5, color.White, 55, 451)
	drawOptionSlider(screen, optionMusicSliderRect, g.musicVolume)
	drawSourceMenuText(screen, fmt.Sprintf("%d%%", int(g.musicVolume*100+0.5)), menuFontCondensedExtraBold, 11.5, color.White, 345, 451)
	drawSourceMenuText(screen, "SFX", menuFontCondensedExtraBold, 12.5, color.White, 55, 484)
	drawOptionSlider(screen, optionSFXSliderRect, g.soundVolume)
	drawSourceMenuText(screen, fmt.Sprintf("%d%%", int(g.soundVolume*100+0.5)), menuFontCondensedExtraBold, 11.5, color.White, 345, 484)

	drawSourceMenuText(screen, "DISPLAY", menuFontCondensedExtraBold, 13.5, color.White, 385, 437)
	drawOptionActionButton(screen, optionLoudnessRect, "LOUDNESS EQ", g.loudnessEQ)
	fullscreenLabel := "FULLSCREEN OFF"
	if ebiten.IsFullscreen() {
		fullscreenLabel = "FULLSCREEN ON"
	}
	drawOptionActionButton(screen, optionFullscreenRect, fullscreenLabel, ebiten.IsFullscreen())
	drawOptionActionButton(screen, optionResetSettingsRect, "RESET SETTINGS", false)

	drawSourceMenuText(screen, "QUALITY", menuFontCondensedExtraBold, 12.5, color.White, 385, 490)
	for _, button := range optionQualityButtons {
		active := (button.hit == optionHitQualityLow && g.quality == 1) ||
			(button.hit == optionHitQualityMedium && g.quality == 2) ||
			(button.hit == optionHitQualityHigh && g.quality == 3)
		drawOptionQualityButton(screen, button.rect, button.label, active)
	}
	drawOptionActionButton(screen, optionResetControlsRect, "RESET CONTROLS", false)
	cheatsLabel := "ALLOW CHEATS OFF"
	if g.allowCheats {
		cheatsLabel = "ALLOW CHEATS ON"
	}
	drawOptionActionButton(screen, optionAllowCheatsRect, cheatsLabel, g.allowCheats)
	drawOptionActionButton(screen, optionBackRect, "BACK", false)

	drawSourceMenuText(screen, "UPDATES", menuFontCondensedExtraBold, 13.5, color.White, 55, 558)
	drawSourceMenuText(screen, "CURRENT: "+currentUpdateVersionLabel(), menuFontCondensedExtraBold, 11.5, color.White, 120, 558)
	drawSourceMenuText(screen, g.settingsUpdateStatus(), menuFontCondensed, 11.5,
		color.NRGBA{R: 0xc7, G: 0xcb, B: 0xd1, A: 0xff}, 120, 575)
	checkLabel := "CHECK FOR UPDATES"
	if g.updateRuntime.phase == updatePhaseChecking {
		checkLabel = "CHECKING..."
	} else if g.updateRuntime.phase == updatePhaseDownloading {
		checkLabel = "DOWNLOADING..."
	}
	drawOptionActionButton(screen, optionCheckUpdatesRect, checkLabel, g.updateRuntime.phase == updatePhaseChecking)
	actionLabel := "NO UPDATE"
	if g.settingsUpdateActionAvailable() {
		actionLabel = "UPDATE NOW"
	} else if g.updateRuntime.phase == updatePhaseDownloading {
		actionLabel = "INSTALLING"
	} else if currentUpdateVersionLabel() == "DEV BUILD" {
		actionLabel = "DEV BUILD"
	}
	drawOptionActionButton(screen, optionUpdateNowRect, actionLabel, g.settingsUpdateActionAvailable())
}

func (g *Game) drawOptionsInteractions(screen *ebiten.Image) {
	if g.assets.OptionToggleFrames[3] == nil {
		return
	}
	// FFDec flattened Symbol990's hidden key-capture popup into OptionsMenu.
	// In the source lockup._alpha starts at 0, so this central 300x150 area is
	// plain black until a key button is actually pressed. Clear the stale copy
	// before rebuilding the live controls; the real popup is drawn last below.
	ebitenutil.DrawRect(screen, 300, 218.1, 300, 150, color.Black)
	// Wipe the entire original lower Options area, not only the new panel bounds.
	// The flattened Flash raster still contains the old MUSIC/SOUND/QUALITY/BACK
	// artwork all the way to the bottom edge, so leaving even a small uncovered
	// strip makes those remnants visible underneath the redesigned controls.
	ebitenutil.DrawRect(screen, 0, 425, 900, 175, color.NRGBA{R: 0x12, G: 0x12, B: 0x12, A: 0xff})
	g.drawDevelopedOptionSettings(screen)

	// Symbol972 is runtime-driven (player number + six dynamic key clips).
	// The flattened FFDec OptionsMenu bakes Player1/W into all four panels, so
	// cover those panels with their source base and rebuild the live controls.
	g.drawOptionControlPanels(screen)
	if g.optionsKeyCapturePlayer > 0 {
		if g.assets.OptionLockupBase != nil {
			drawSourceRaster(screen, g.assets.OptionLockupBase, 0, 0, 1, 1, 1)
			drawSourceMenuText(screen, "Press a Valid Key to Assign", menuFontTwCen, 21.25,
				color.Black, 321.05, 268.65)
			drawSourceMenuText(screen, "ESC = CANCEL", menuFontCondensedExtraBold, 14,
				color.NRGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xff}, 401, 302)
		} else {
			drawSourceRaster(screen, g.assets.OptionLockup, 0, 0, 1, 1, 1)
		}
	}
}
