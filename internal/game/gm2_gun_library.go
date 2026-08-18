package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	gunLibraryHitSetGM1 = -10
	gunLibraryHitSetGM2 = -11
)

func gunLibrarySetRects() (Rect, Rect) {
	return Rect{X: 475, Y: 24, W: 86, H: 30}, Rect{X: 570, Y: 24, W: 96, H: 30}
}

func gunLibrarySetHitAt(x, y float64) int {
	gm1, gm2 := gunLibrarySetRects()
	if gm1.Contains(x, y) {
		return gunLibraryHitSetGM1
	}
	if gm2.Contains(x, y) {
		return gunLibraryHitSetGM2
	}
	return 0
}

func gm2GunLibraryRect(number int) Rect {
	index := number - 67
	if index < 0 || index >= 20 {
		return Rect{}
	}
	col := index % 5
	row := index / 5
	return Rect{X: 28 + float64(col)*170, Y: 94 + float64(row)*78, W: 156, H: 66}
}

func gm2GunLibraryHitAt(x, y float64) int {
	for number := 67; number <= 86; number++ {
		if gm2GunLibraryRect(number).Contains(x, y) {
			return number
		}
	}
	return 0
}

func (g *Game) drawGunLibrarySetTabs(screen *ebiten.Image) {
	gm1, gm2 := gunLibrarySetRects()
	for _, tab := range []struct {
		r Rect
		label string
		active bool
		enabled bool
	}{
		{gm1, "GM1", !g.gunLibraryGM2, true},
		{gm2, "GM2", g.gunLibraryGM2, gm2WeaponAssetsAvailable()},
	} {
		c := color.NRGBA{R: 0xd0, G: 0xd0, B: 0xd0, A: 0xff}
		if !tab.enabled {
			c = color.NRGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xff}
		} else if tab.active {
			c = color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
		}
		ebitenutil.DrawRect(screen, tab.r.X, tab.r.Y, tab.r.W, tab.r.H, c)
		drawSourceMenuText(screen, tab.label, menuFontCondensedExtraBold, 17, color.Black, tab.r.X+25, tab.r.Y+4)
	}
}

func gm2WeaponCategory(number int) string {
	switch {
	case number >= 67 && number <= 71:
		return "SMGs"
	case number >= 72 && number <= 76:
		return "ARs / LMGs"
	case number >= 77 && number <= 81:
		return "SNIPERS"
	case number >= 82 && number <= 86:
		return "SHOTGUNS"
	default:
		return ""
	}
}

func (g *Game) drawGM2GunLibrary(screen *ebiten.Image) {
	// Cover the frozen GM1 card grid but retain the source header/back/test area.
	ebitenutil.DrawRect(screen, 18, 72, 864, 348, color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	drawSourceMenuText(screen, "GUN MAYHEM 2 - NEW WEAPONS", menuFontCondensedExtraBold, 25, color.Black, 28, 67)

	for number := 67; number <= 86; number++ {
		r := gm2GunLibraryRect(number)
		c := color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}
		if g.gunLibrarySelected == number {
			c = color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
		}
		ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, c)
		if thumb := g.assets.GunLibraryThumbs[number]; thumb != nil {
			drawGunLibraryCleanThumb(screen, thumb, r.X+29, r.Y+33, 48, 36, 1, -12)
		}
		def, _ := WeaponByNumber(number)
		drawSourceMenuText(screen, fmt.Sprintf("%d  %s", number, def.Name), menuFontTwCen, 11.4, color.Black, r.X+55, r.Y+8)
		drawSourceMenuText(screen, gm2WeaponCategory(number), menuFontTwCen, 10.5, color.NRGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xff}, r.X+55, r.Y+29)
	}

	g.drawGM2WeaponDetails(screen)
}

func drawGM2StatBar(screen *ebiten.Image, label string, value, max float64, x, y float64) {
	drawSourceMenuText(screen, label, menuFontCondensed, 14, color.Black, x, y-3)
	ebitenutil.DrawRect(screen, x+100, y, 180, 7, color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff})
	if max <= 0 {
		return
	}
	f := math.Max(0, math.Min(1, value/max))
	ebitenutil.DrawRect(screen, x+100, y, 180*f, 7, color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff})
}

func (g *Game) drawGM2WeaponDetails(screen *ebiten.Image) {
	n := g.gunLibrarySelected
	if n < 67 || n > 86 {
		n = 67
	}
	def, ok := WeaponByNumber(n)
	if !ok {
		return
	}
	ebitenutil.DrawRect(screen, 25, 433, 600, 145, color.NRGBA{R: 0xb8, G: 0xb8, B: 0xb8, A: 0xff})
	drawSourceMenuText(screen, def.DisplayName(), menuFontCondensedExtraBold, 22, color.Black, 36, 439)
	drawSourceMenuText(screen, def.Name, menuFontTwCen, 12, color.NRGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xff}, 36, 468)
	if thumb := g.assets.GunLibraryThumbs[n]; thumb != nil {
		drawGunLibraryCleanThumb(screen, thumb, 555, 500, 120, 80, 1, -12)
	}
	drawGM2StatBar(screen, "Damage", def.Firepower, 70, 36, 493)
	// Lower ROF means faster in the Flash source; invert it for the visual bar.
	rate := 1.0 / math.Max(1, float64(def.ROF))
	drawGM2StatBar(screen, "Rate of fire", rate, 0.5, 36, 512)
	drawGM2StatBar(screen, "Ammo", float64(def.Bullets), 40, 36, 531)
	drawGM2StatBar(screen, "Recoil", def.Recoil, 10, 36, 550)
}
