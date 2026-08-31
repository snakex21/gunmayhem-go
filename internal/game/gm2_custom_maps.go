package game

import (
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	customGM2RandomNew = -101
	customGM2RandomAll = -102

	menuHitMapSetGM1    = 490
	menuHitMapSetGM2    = 491
	menuHitGM2MapBase   = 500
	menuHitGM2RandomNew = 522
	menuHitGM2RandomAll = 523
)

type gm2MapRow struct {
	value int
	label string
}

func gm2CustomMapRows() []gm2MapRow {
	// The combined build keeps only one GM2 random option. Put it first, just
	// like GM1's RANDOM row, then show the nine genuinely new GM2 arenas.
	rows := make([]gm2MapRow, 0, 10)
	rows = append(rows, gm2MapRow{value: customGM2RandomAll, label: "Random GM2"})
	for n := 1; n <= 9; n++ {
		rows = append(rows, gm2MapRow{value: gm2MapID(n), label: gm2MapDisplayNames[n]})
	}
	return rows
}

var (
	gm2MapAssetsOnce sync.Once
	gm2MapAssetsOK   bool
)

func gm2MapAssetsAvailable() bool {
	gm2MapAssetsOnce.Do(func() {
		gm2MapAssetsOK = true
		checks := [][]string{
			{"fla", "LIBRARY", "Symbol 1940.xml"},
			{"sprites", "DefineSprite_1642", "1.png"},
			{"sprites", "DefineSprite_1642", "21.png"},
			{"sprites", "DefineSprite_1835", "21.png"},
		}
		for _, parts := range checks {
			if _, err := findOriginalPathIn("gm2", parts...); err != nil {
				gm2MapAssetsOK = false
				return
			}
		}
	})
	return gm2MapAssetsOK
}

func gm2MapSetButtonRects() (Rect, Rect) {
	return Rect{X: -883, Y: 69, W: 96, H: 22}, Rect{X: -783, Y: 69, W: 96, H: 22}
}

func gm2MapRowRect(index int) Rect {
	// Ten rows fill the same vertical strip as the denser GM1 selector instead
	// of leaving a large empty block below the nine GM2 maps.
	const startY = 99.3
	const rowH = 36.0
	return Rect{X: -883, Y: startY + float64(index)*rowH, W: 200, H: rowH - 0.6}
}

func gm2MapListClearRect() Rect {
	// Cover the full source GM1 map-button strip. The CustomMenu raster has the
	// old GM1 rows baked into it, so when GM2 is selected we must erase the
	// entire strip before drawing the shorter 9-new-maps list.
	return Rect{X: -883, Y: 92, W: 200, H: 402}
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
	clear := gm2MapListClearRect()
	ebitenutil.DrawRect(screen, g.customMenuX+clear.X, clear.Y, clear.W, clear.H,
		color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xff})

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
		drawSourceMenuText(screen, row.label, menuFontTwCen, 14.2, color.Black,
			g.customMenuX+r.X+5, r.Y+8)
	}
}

func (g *Game) drawGM2MapPreview(screen *ebiten.Image) {
	x := g.customMenuX - 640
	y := 120.0

	// Some GM2 scene rasters have visual bounds that extend far outside the
	// nominal 900x600 stage. Drawing them straight onto the menu lets those
	// oversized layers bleed over the map list and lower controls. Render the
	// preview into its own 600x350 image first so Ebiten clips every layer to
	// the intended preview rectangle.
	preview := ebiten.NewImage(600, 350)
	preview.Fill(color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xff})

	if g.customMap == customGM2RandomNew || g.customMap == customGM2RandomAll {
		label := "RANDOM GM2"
		if g.customMap == customGM2RandomNew {
			label = "RANDOM NEW MAPS"
		}
		drawSourceMenuText(preview, "?", menuFontCondensedExtraBold, 145, color.NRGBA{R: 0x99, A: 0xff}, 258, 75)
		drawSourceMenuText(preview, label, menuFontCondensedExtraBold, 38, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, 10, 300)
	} else if isGM2MapID(g.customMap) {
		if sourcePreview := g.assets.MapPreviewImages[g.customMap]; sourcePreview != nil {
			// Symbol 1236 already contains a dedicated bitmap for every GM2 map,
			// exactly like GM1. Use that original preview directly instead of
			// constructing the whole arena scene just for this menu.
			drawSetupThumbFit(preview, sourcePreview, Rect{X: 0, Y: 0, W: 600, H: 350}, 1)
		} else {
			g.assets.EnsureScene(g.customMap)
			const s = 350.0 / 600.0
			ox := (600 - 900*s) / 2
			drawSourceRaster(preview, g.assets.SceneBack[g.customMap], ox, 0, s, s, 1)
			drawSourceRaster(preview, g.assets.SceneMid[g.customMap], ox, 0, s, s, 1)
			drawSourceRaster(preview, g.assets.SceneFront[g.customMap], ox+51.1*s, 192.6*s, s, s, 1)
		}
		name := gm2MapDisplayNames[gm2SourceMapNumber(g.customMap)]
		ebitenutil.DrawRect(preview, 0, 0, 600, 38, color.NRGBA{A: 0xb0})
		drawSourceMenuText(preview, name, menuFontCondensedExtraBold, 30, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, 8, 3)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(preview, op)
}
