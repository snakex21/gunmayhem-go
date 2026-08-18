package game

import (
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	menuHitModeCrates  = 8001
	menuHitModePickups = 8002
	menuHitModeLives   = 8003
)

var modeCheckLocal = Rect{X: -1.5, Y: -7.35, W: 34.85, H: 38.85} // Symbol1265

func modeHasLives(mode int) bool {
	return mode == SourceGameModeNormal || mode == SourceGameModeInstagib || mode == SourceGameModeTeams
}

func modeHasCrateToggle(mode int) bool {
	return mode == SourceGameModeNormal || mode == SourceGameModeInstagib || mode == SourceGameModeTeams
}

func modeHasPickupToggle(mode int) bool {
	return mode == SourceGameModeNormal || mode == SourceGameModeTeams
}

func (g *Game) customModeOptionsHitAt(x, y float64) int {
	if modeHasCrateToggle(g.customMode) {
		if sourceMenuHitRect(modeCheckLocal, -1137.75, 393.5, 1, 1).Contains(x, y) {
			return menuHitModeCrates
		}
	}
	if modeHasPickupToggle(g.customMode) {
		if sourceMenuHitRect(modeCheckLocal, -1137.75, 440.55, 1, 1).Contains(x, y) {
			return menuHitModePickups
		}
	}
	if modeHasLives(g.customMode) {
		if (Rect{X: -1320.65, Y: 394, W: 59.2, H: 22.35}).Contains(x, y) {
			return menuHitModeLives
		}
	}
	return menuHitNone
}

func (g *Game) activateCustomModeOption(hit int) bool {
	switch hit {
	case menuHitModeCrates:
		if modeHasCrateToggle(g.customMode) {
			g.CrateON = !g.CrateON
		}
		return true
	case menuHitModePickups:
		if modeHasPickupToggle(g.customMode) {
			g.PowerON = !g.PowerON
		}
		return true
	case menuHitModeLives:
		if modeHasLives(g.customMode) {
			g.customLivesFocus = true
		}
		return true
	}
	return false
}

func (g *Game) updateCustomLivesInput() {
	if !g.customLivesFocus || g.customPage != customPageGame || !modeHasLives(g.customMode) {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.customLivesFocus = false
		return
	}
	value := strconv.Itoa(g.customLives)
	if g.customLives == 0 {
		value = ""
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(value) > 0 {
		value = value[:len(value)-1]
	}
	for _, ch := range ebiten.AppendInputChars(nil) {
		if ch >= '0' && ch <= '9' && len(value) < 5 {
			value += string(ch)
		}
	}
	if value == "" {
		g.customLives = 0
	} else if n, err := strconv.Atoi(value); err == nil {
		g.customLives = n
	}
}

func drawModeTextLines(screen *ebiten.Image, lines []string, x, y float64) {
	for i, line := range lines {
		drawSourceMenuText(screen, line, menuFontCondensed, 21.2, color.Black, x, y+float64(i)*25)
	}
}

func (g *Game) drawCustomModeButtons(screen *ebiten.Image) {
	// Symbol1282 source state: idle = white Symbol47 strip, hover = raw
	// Symbol1199 #999999, chosen = the same Symbol1199 color-transformed to
	// #ff6600. FFDec flattened the default chosen layer into every button, so
	// rebuild the five visible rows at runtime instead of drawing a second raw
	// gray 'chosen' raster over the text.
	rows := [...]struct {
		mode      int
		ty, textY float64
		label     string
	}{
		{SourceGameModeNormal, 160.0, 162.0, "Last Man Standing"},
		{SourceGameModeTeams, 225.7, 227.7, "Last Man Standing (Team mode)"},
		{SourceGameModeSurvival, 291.5, 293.5, "Duck Survival"},
		{SourceGameModeGunGame, 357.25, 359.25, "Gun Game"},
		{SourceGameModeInstagib, 423.1, 425.1, "1 Hit 1 Kill"},
	}
	mx, my := ebiten.CursorPosition()
	lx := float64(mx) - g.customMenuX
	ly := float64(my)
	for _, row := range rows {
		r := sourceMenuHitRect(modeButtonLocal, -1770, row.ty, 1.6749725341796875, 2.99078369140625)
		clr := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
		if row.mode == g.customMode {
			clr = color.NRGBA{R: 0xff, G: 0x66, A: 0xff}
		} else if r.Contains(lx, ly) {
			clr = color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}
		}
		ebitenutil.DrawRect(screen, g.customMenuX+r.X, r.Y, r.W, r.H, clr)
		drawSourceMenuText(screen, row.label, menuFontTwCen, 22.15, color.Black,
			g.customMenuX-1763, row.textY)
	}
}

func (g *Game) drawCustomModeOptions(screen *ebiten.Image) {
	mode := g.customMode
	if mode < SourceGameModeNormal || mode > SourceGameModeSurvival {
		mode = SourceGameModeNormal
	}

	// FFDec exports all six Symbol1273 timeline frames as the same Catch the
	// Duck raster. Rebuild the runtime frame from its XFL coordinates instead.
	// The flattened Symbol1309 underneath still supplies the panel drop shadow.
	x := g.customMenuX - 1400
	y := 100.0
	ebitenutil.DrawRect(screen, x, y, 480, 400, color.White)

	title := "LAST MAN STANDING"
	desc := []string{
		"Free for all. Last player alive wins the game. Weapons are",
		"picked up in crates.",
	}
	switch mode {
	case SourceGameModeInstagib:
		title = "1 Hit 1 Kill"
		desc = []string{
			"Given the most powerful gun in the game, eliminate your",
			"opponents. Be mindful of ammo consumption, as you are only",
			"given 5 shots per life.",
		}
	case SourceGameModeTeams:
		title = "Last Man Standing"
		desc = []string{"Same as last man standing, except players battle in teams."}
	case SourceGameModeGunGame:
		title = "Gun Game"
		desc = []string{
			"Kill your opponents to level up your weapon. First player to",
			"get a kill with all the weapons wins the game.",
		}
	case SourceGameModeSurvival:
		title = "Duck Survival"
		desc = []string{
			"Play by yourself or with a friend. Survive for as long as",
			"possible. Random weapons can obtained through crates or the",
			"random weapon box.",
		}
	}

	drawSourceMenuText(screen, title, menuFontCondensed, 46.8, color.Black, x+9, y+6.05)
	if mode == SourceGameModeTeams {
		drawSourceMenuText(screen, "(Team Mode)", menuFontCondensed, 32.7, color.Black, x+312.05, y+19.05)
	}
	ebitenutil.DrawRect(screen, x+12, y+61.15, 438.75, 1.5, color.Black)
	drawModeTextLines(screen, desc, x+14, y+70)

	// Static pause notice from Symbol1273 layers 2/3.
	ebitenutil.DrawRect(screen, x+271.1, y+148, 200, 60, color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff})
	drawSourceMenuText(screen, "You can pause the game at any", menuFontCondensed, 17.65,
		color.NRGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xff}, x+278, y+154.75)
	drawSourceMenuText(screen, "time by pressing ESC or SPACE", menuFontCondensed, 17.65,
		color.NRGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xff}, x+278, y+179)

	drawSourceMenuText(screen, "Game Options", menuFontCondensed, 26.5, color.Black, x+11.95, y+240.85)
	ebitenutil.DrawRect(screen, x+8.95, y+273.45, 438.75, 1.5, color.Black)

	if modeHasLives(mode) {
		drawSourceMenuText(screen, "Lives:", menuFontCondensed, 26.5, color.Black, x+9, y+288.5)
		// Symbol1273.inputlives is Arial 20, not Tw Cen MT.
		drawSourceMenuText(screen, strconv.Itoa(g.customLives), menuFontArial, 20,
			color.Black, x+79.35, y+294)
	} else if mode == SourceGameModeSurvival {
		drawSourceMenuText(screen, "Lives:", menuFontCondensed, 26.5, color.Black, x+9, y+288.5)
		drawSourceMenuText(screen, "10", menuFontArialBold, 23.75, color.Black, x+67.05, y+291.5)
	}

	if modeHasCrateToggle(mode) {
		drawSourceMenuText(screen, "Crates", menuFontCondensed, 26.5, color.Black, x+159, y+293.5)
		frame := 1
		if !g.CrateON {
			frame = 2
		}
		drawSourceRaster(screen, g.assets.ModeCheckFrames[frame], x+262.25, y+293.5, 1, 1, 1)
	}
	if modeHasPickupToggle(mode) {
		drawSourceMenuText(screen, "Pickups", menuFontCondensed, 26.5, color.Black, x+159, y+340.5)
		frame := 1
		if !g.PowerON {
			frame = 2
		}
		drawSourceRaster(screen, g.assets.ModeCheckFrames[frame], x+262.25, y+340.55, 1, 1, 1)
	}
}
