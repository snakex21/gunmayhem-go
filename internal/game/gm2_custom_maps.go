package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	customGM2RandomNew = -101
	customGM2RandomAll = -102

	menuHitMapSetGM1  = 490
	menuHitMapSetGM2  = 491
	menuHitGM2MapBase = 500
	menuHitGM2RandomNew = 522
	menuHitGM2RandomAll = 523
)

type gm2MapRow struct {
	value int
	label string
}

func gm2CustomMapRows() []gm2MapRow {
	rows := make([]gm2MapRow, 0, 23)
	for n := 1; n <= 21; n++ {
		rows = append(rows, gm2MapRow{value: gm2MapID(n), label: gm2MapDisplayNames[n]})
	}
	rows = append(rows,
		gm2MapRow{value: customGM2RandomNew, label: "Random (new maps)"},
		gm2MapRow{value: customGM2RandomAll, label: "Random (all maps)"},
	)
	return rows
}

func gm2MapAssetsAvailable() bool {
	checks := [][]string{
		{"fla", "LIBRARY", "Symbol 1940.xml"},
		{"sprites", "DefineSprite_1642", "1.png"},
		{"sprites", "DefineSprite_1642", "21.png"},
		{"sprites", "DefineSprite_1835", "21.png"},
	}
	for _, parts := range checks {
		if _, err := findOriginalPathIn("gm2", parts...); err != nil {
			return false
		}
	}
	return true
}

func gm2MapSetButtonRects() (Rect, Rect) {
	return Rect{X: -883, Y: 69, W: 96, H: 22}, Rect{X: -783, Y: 69, W: 96, H: 22}
}

func gm2MapRowRect(index int) Rect {
	const startY = 94.0
	const rowH = 17.3
	return Rect{X: -883, Y: startY + float64(index)*rowH, W: 200, H: rowH - 0.6}
}

// gm2MapMenuHitAt overlays the developed GM1/GM2 source selector on the map
// page. The bool reports whether the GM2 overlay owns this point so the old GM1
// rows beneath it cannot accidentally fire while the GM2 list is visible.
func (g *Game) gm2MapMenuHitAt(x, y float64) (int, bool) {
	gm1, gm2 := gm2MapSetButtonRects()
	if gm1.Contains(x, y) {
		return menuHitMapSetGM1, true
	}
	if gm2.Contains(x, y) {
		if gm2MapAssetsAvailable() {
			return menuHitMapSetGM2, true
		}
		return menuHitNone, true
	}
	if !g.customMapSetGM2 {
		return menuHitNone, false
	}
	for i, row := range gm2CustomMapRows() {
		if gm2MapRowRect(i).Contains(x, y) {
			switch row.value {
			case customGM2RandomNew:
				return menuHitGM2RandomNew, true
			case customGM2RandomAll:
				return menuHitGM2RandomAll, true
			default:
				return menuHitGM2MapBase + gm2SourceMapNumber(row.value), true
			}
		}
	}
	// Suppress the original GM1 map rows under the developed GM2 list.
	if x >= -883 && x <= -683 && y >= 92 && y <= 494 {
		return menuHitNone, true
	}
	return menuHitNone, false
}

func (g *Game) drawMapSetButtons(screen *ebiten.Image) {
	gm1, gm2 := gm2MapSetButtonRects()
	buttons := []struct {
		r       Rect
		label   string
		active  bool
		enabled bool
	}{
		{gm1, "GM1", !g.customMapSetGM2, true},
		{gm2, "GM2", g.customMapSetGM2, gm2MapAssetsAvailable()},
	}
	for _, b := range buttons {
		c := color.NRGBA{R: 0xdd, G: 0xdd, B: 0xdd, A: 0xff}
		if b.active {
			c = color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
		} else if !b.enabled {
			c = color.NRGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xff}
		}
		ebitenutil.DrawRect(screen, g.customMenuX+b.r.X, b.r.Y, b.r.W, b.r.H, c)
		drawSourceMenuText(screen, b.label, menuFontTwCen, 14, color.Black,
			g.customMenuX+b.r.X+31, b.r.Y+3)
	}
}

func (g *Game) drawGM2CustomMapButtons(screen *ebiten.Image) {
	mx, my := ebiten.CursorPosition()
	lx := float64(mx) - g.customMenuX
	ly := float64(my)
	for i, row := range gm2CustomMapRows() {
		r := gm2MapRowRect(i)
		c := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
		if row.value == g.customMap {
			c = color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
		} else if r.Contains(lx, ly) {
			c = color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}
		}
		ebitenutil.DrawRect(screen, g.customMenuX+r.X, r.Y, r.W, r.H, c)
		drawSourceMenuText(screen, row.label, menuFontTwCen, 11.5, color.Black,
			g.customMenuX+r.X+4, r.Y+1.5)
	}
}

func (g *Game) drawGM2MapPreview(screen *ebiten.Image) {
	x := g.customMenuX - 640
	y := 120.0
	ebitenutil.DrawRect(screen, x, y, 600, 350, color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xff})
	if g.customMap == customGM2RandomNew || g.customMap == customGM2RandomAll {
		label := "RANDOM GM2"
		if g.customMap == customGM2RandomNew {
			label = "RANDOM NEW MAPS"
		}
		drawSourceMenuText(screen, "?", menuFontCondensedExtraBold, 145, color.NRGBA{R: 0x99, A: 0xff}, x+258, y+75)
		drawSourceMenuText(screen, label, menuFontCondensedExtraBold, 38, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, x+10, y+300)
		return
	}
	if !isGM2MapID(g.customMap) {
		return
	}
	g.assets.EnsureScene(g.customMap)
	const s = 350.0 / 600.0
	ox := x + (600-900*s)/2
	drawSourceRaster(screen, g.assets.SceneBack[g.customMap], ox, y, s, s, 1)
	drawSourceRaster(screen, g.assets.SceneMid[g.customMap], ox, y, s, s, 1)
	drawSourceRaster(screen, g.assets.SceneFront[g.customMap], ox+51.1*s, y+192.6*s, s, s, 1)
	name := gm2MapDisplayNames[gm2SourceMapNumber(g.customMap)]
	ebitenutil.DrawRect(screen, x, y, 600, 38, color.NRGBA{A: 0xb0})
	drawSourceMenuText(screen, name, menuFontCondensedExtraBold, 30, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, x+8, y+3)
}
