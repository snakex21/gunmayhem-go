package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var sourceMapDisplayNames = map[int]string{
	1:  "NO NAME",
	2:  "DESSERT DUEL",
	3:  "UNDERWATER SLAUGHTER",
	4:  "SOLAR SHOOTOUT",
	5:  "GREAT WALL BRAWL",
	6:  "MAGIC MUSHROOM MOUNTAIN MELEE",
	7:  "DESERT DESTRUCTION",
	8:  "HOVERING HOUSES",
	9:  "MIDNIGHT WOOD",
	10: "POLAR PWNAGE",
	11: "GRIM CITY",
	12: "SAFARI SHOWDOWN",
}

func (g *Game) drawCustomMapButtons(screen *ebiten.Image) {
	rows := [...]struct {
		mapNumber int
		ty        float64
		label     string
	}{
		{0, 99.3, "RANDOM"},
		{12, 138.9, "Safari Showdown"},
		{11, 167.95, "Grim City"},
		{10, 197.0, "Polar Pwnage"},
		{9, 226.0, "Midnight Wood"},
		{8, 255.05, "Hovering Houses"},
		{7, 284.1, "Desert Destruction"},
		{6, 313.15, "Magic Mushroom Mountain Melee"},
		{5, 342.2, "Great Wall Brawl"},
		{4, 371.2, "Solar Shootout"},
		{3, 400.25, "Underwater Slaughter"},
		{2, 429.3, "Dessert Duel"},
		{1, 458.35, "No Name"},
	}
	mx, my := ebiten.CursorPosition()
	lx := float64(mx) - g.customMenuX
	ly := float64(my)
	for _, row := range rows {
		r := sourceMenuHitRect(mapButtonLocal, -883, row.ty, 1, 1.3199920654296875)
		clr := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
		if row.mapNumber == g.customMap {
			clr = color.NRGBA{R: 0xff, G: 0x66, A: 0xff}
		} else if r.Contains(lx, ly) {
			clr = color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}
		}
		ebitenutil.DrawRect(screen, g.customMenuX+r.X, r.Y, r.W, r.H, clr)
		drawSourceMenuText(screen, row.label, menuFontTwCen, 14.2, color.Black,
			g.customMenuX-878, row.ty+4)
	}
}

// drawCustomMapPreview reconstructs Symbol1230's current frame. FFDec emitted
// identical PNGs for all 13 frames, while XFL switches the 600x350 source
// bitmap on every map selection.
func (g *Game) drawCustomMapPreview(screen *ebiten.Image) {
	x := g.customMenuX - 640
	y := 120.0

	if g.customMap == 0 {
		// Symbol1230 frame13 is the RANDOM question-mark frame.
		ebitenutil.DrawRect(screen, x, y, 600, 350, color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xff})
		drawSourceMenuText(screen, "?", menuFontCondensedExtraBold, 150, color.NRGBA{R: 0x99, A: 0xff}, x+255, y+80)
		drawSourceMenuText(screen, "RANDOM", menuFontCondensedExtraBold, 60.15, color.Black, x+5, y-2.9)
		return
	}

	if preview := g.assets.MapPreviewImages[g.customMap]; preview != nil {
		drawSourceRaster(screen, preview, x, y, 1, 1, 1)
	}
	if name := sourceMapDisplayNames[g.customMap]; name != "" {
		size := 60.15
		if g.customMap == 3 {
			size = 56.4 // source UNDERWATER SLAUGHTER frame
		}
		drawSourceMenuText(screen, name, menuFontCondensedExtraBold, size, color.Black, x+5, y-2.9)
	}
}
