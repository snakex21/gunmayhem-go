package game

import (
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
	optionHitControlBase = 100
)

var optionToggleLocal = Rect{X: 0, Y: 0, W: 57, H: 31.5} // Symbol982

var optionPlacements = [...]struct {
	hit  int
	x, y float64
}{
	{optionHitMusicOn, 74.0, 511.85},
	{optionHitMusicOff, 137.0, 511.85},
	{optionHitSoundOn, 252.55, 511.85},
	{optionHitSoundOff, 315.55, 511.85},
	{optionHitQualityLow, 431.1, 511.85},
	{optionHitQualityMedium, 494.1, 511.85},
	{optionHitQualityHigh, 557.1, 512.15},
}

func optionsHitAt(x, y float64) int {
	for _, p := range optionPlacements {
		if sourceMenuHitRect(optionToggleLocal, p.x, p.y, 1, 1).Contains(x, y) {
			return p.hit
		}
	}
	if sourceMenuHitRect(campaignBackLocal, 714.15, 512.15, 1, 1).Contains(x, y) {
		return optionHitBack
	}
	return optionHitNone
}

func (g *Game) updateOptionsMenu() {
	if g.optionsKeyCapturePlayer > 0 {
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
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.optionsPressed = optionHitNone
		g.beginFade(screenMainMenu, fadePurposeScreen)
		return
	}
	mx, my := ebiten.CursorPosition()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if player, index, ok := optionsControlHitAt(float64(mx), float64(my)); ok {
			// Symbol965 uses onPress and immediately disables every other button
			// until keyListener receives a valid Key.getCode().
			g.optionsKeyCapturePlayer = player + 1
			g.optionsKeyCaptureIndex = index
			g.optionsPressed = optionHitNone
			return
		}
	}
	hit := optionsHitAt(float64(mx), float64(my))
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
	case optionHitMusicOn:
		g.musicOn = true
		if g.audioStarted {
			g.syncSourceMusic()
		}
		_ = g.saveConfig()
	case optionHitMusicOff:
		g.musicOn = false
		if g.audioStarted {
			g.syncSourceMusic()
		}
		_ = g.saveConfig()
	case optionHitSoundOn:
		g.soundOn = true
		_ = g.saveConfig()
	case optionHitSoundOff:
		g.soundOn = false
		_ = g.saveConfig()
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

func (g *Game) drawOptionsInteractions(screen *ebiten.Image) {
	if g.assets.OptionToggleFrames[3] == nil {
		return
	}
	// FFDec flattened Symbol990's hidden key-capture popup into OptionsMenu.
	// In the source lockup._alpha starts at 0, so this central 300x150 area is
	// plain black until a key button is actually pressed. Clear the stale copy
	// before rebuilding the live controls; the real popup is drawn last below.
	ebitenutil.DrawRect(screen, 300, 218.1, 300, 150, color.Black)
	selected := map[int]bool{
		optionHitMusicOn:       g.musicOn,
		optionHitMusicOff:      !g.musicOn,
		optionHitSoundOn:       g.soundOn,
		optionHitSoundOff:      !g.soundOn,
		optionHitQualityLow:    g.quality == 1,
		optionHitQualityMedium: g.quality == 2,
		optionHitQualityHigh:   g.quality == 3,
	}
	mx, my := ebiten.CursorPosition()
	hover := optionsHitAt(float64(mx), float64(my))
	for _, p := range optionPlacements {
		frame := 1
		if selected[p.hit] {
			frame = 3
		} else if hover == p.hit {
			frame = 2
		}
		drawSourceRaster(screen, g.assets.OptionToggleFrames[frame], p.x, p.y, 1, 1, 1)
	}
	// Symbol991 keeps the button captions on layers ABOVE Symbol982. Since the
	// runtime state buttons are redrawn here after the flattened base menu, draw
	// those source captions again on top or they'd be covered by the buttons.
	for _, item := range []struct {
		text       string
		x, y, size float64
	}{
		{"ON", 81.35, 514.25, 22.1},
		{"OFF", 144.90, 514.25, 22.1},
		{"ON", 260.45, 514.25, 22.1},
		{"OFF", 324.00, 514.25, 22.1},
		{"LOW", 439.00, 518.75, 15.9},
		{"MEDIUM", 498.80, 518.00, 15.9},
		{"HIGH", 565.00, 518.30, 15.9},
	} {
		drawSourceMenuText(screen, item.text, menuFontCondensed, item.size, color.Black, item.x, item.y)
	}
	// Symbol972 is runtime-driven (player number + six dynamic key clips).
	// The flattened FFDec OptionsMenu bakes Player1/W into all four panels, so
	// cover those panels with their source base and rebuild the live controls.
	g.drawOptionControlPanels(screen)
	if g.optionsKeyCapturePlayer > 0 {
		if g.assets.OptionLockupBase != nil {
			drawSourceRaster(screen, g.assets.OptionLockupBase, 0, 0, 1, 1, 1)
			drawSourceMenuText(screen, "Press a Valid Key to Assign", menuFontTwCen, 21.25,
				color.Black, 321.05, 274.65)
		} else {
			drawSourceRaster(screen, g.assets.OptionLockup, 0, 0, 1, 1, 1)
		}
	}
}
