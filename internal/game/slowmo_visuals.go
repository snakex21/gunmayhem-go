package game

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// slowmo_visuals.go: SLOW-MO feedback — the hourglass powerup icon, a cool
// desaturating tint with a blue vignette over the arena, and the top-centre
// timer bar.

var (
	slowMoVignetteOnce sync.Once
	slowMoVignette     *ebiten.Image
)

// drawSlowMoHourglass draws an hourglass centred on (cx, cy); s = 1 fits a
// 48 px powerup icon.
func drawSlowMoHourglass(dst *ebiten.Image, cx, cy, s float64, fill, ink color.NRGBA) {
	pt := func(x, y float64) [2]float64 { return [2]float64{cx + x*s, cy + y*s} }
	// Glass: two triangles meeting at the waist.
	glass := [][2]float64{pt(-10, -15), pt(10, -15), pt(1.6, -1.5), pt(1.6, 1.5), pt(10, 15), pt(-10, 15), pt(-1.6, 1.5), pt(-1.6, -1.5)}
	fillPolygon(dst, glass, fill)
	strokePolygon(dst, glass, float32(1.6*s), ink)
	// Sand: what is left on top, the falling stream and the pile below.
	fillPolygon(dst, [][2]float64{pt(-5.5, -9), pt(5.5, -9), pt(1, -3), pt(-1, -3)}, ink)
	vector.StrokeLine(dst, float32(cx), float32(cy-3*s), float32(cx), float32(cy+10*s), float32(1.2*s), ink, true)
	fillPolygon(dst, [][2]float64{pt(0, 7), pt(7.5, 14), pt(-7.5, 14)}, ink)
	// End caps.
	for _, y := range []float64{-19, 15} {
		caps := [][2]float64{pt(-13, y), pt(13, y), pt(13, y+4), pt(-13, y+4)}
		fillPolygon(dst, caps, fill)
		strokePolygon(dst, caps, float32(1.4*s), ink)
	}
}

// slowMoVignetteImage is a screen-sized radial darkening in deep blue,
// transparent in the middle.
func slowMoVignetteImage() *ebiten.Image {
	slowMoVignetteOnce.Do(func() {
		const w, h = ScreenWidth, ScreenHeight
		pix := make([]byte, w*h*4)
		for y := 0; y < h; y++ {
			ny := (float64(y)+0.5)/h*2 - 1
			for x := 0; x < w; x++ {
				nx := (float64(x)+0.5)/w*2 - 1
				d := math.Sqrt(nx*nx*0.85 + ny*ny)
				a := (d - 0.62) / 0.6
				if a <= 0 {
					continue
				}
				if a > 1 {
					a = 1
				}
				a = a * a * 0.62
				i := (y*w + x) * 4
				// Premultiplied RGBA.
				pix[i] = byte(8 * a)
				pix[i+1] = byte(22 * a)
				pix[i+2] = byte(64 * a)
				pix[i+3] = byte(255 * a)
			}
		}
		img := ebiten.NewImage(w, h)
		img.WritePixels(pix)
		slowMoVignette = img
	})
	return slowMoVignette
}

// drawSlowMoOverlay tints the arena while the match runs slowed: a grey-blue
// wash (it pulls the colours towards grey) and the blue vignette.
func (g *Game) drawSlowMoOverlay(screen *ebiten.Image) {
	a := g.slowMoFeedbackAlpha()
	if a <= 0 {
		return
	}
	ebitenutil.DrawRect(screen, 0, 0, ScreenWidth, ScreenHeight, color.NRGBA{R: 104, G: 124, B: 160, A: uint8(70 * a)})
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(float32(a))
	screen.DrawImage(slowMoVignetteImage(), op)
}

// slowMoBarRect is the top-centre timer bar (below the Chaos banner when
// that is on screen).
func (g *Game) slowMoBarRect() Rect {
	const w, h = 236.0, 30.0
	y := 70.0
	if g.customChaosEnabled && g.chaosCurrentEffect != chaosEffectNone {
		y = 158
	}
	return Rect{X: (ScreenWidth - w) / 2, Y: y, W: w, H: h}
}

// drawSlowMoBar draws "SLOW-MO" with the time left as a draining bar.
func (g *Game) drawSlowMoBar(screen *ebiten.Image) {
	a := g.slowMoFeedbackAlpha()
	if a <= 0 {
		return
	}
	r := g.slowMoBarRect()
	panel := withAlpha(color.NRGBA{R: 0x12, G: 0x1a, B: 0x2c, A: 0xff}, 0.86*a)
	accent := withAlpha(color.NRGBA{R: 0x7c, G: 0xd4, B: 0xff, A: 0xff}, a)
	track := withAlpha(color.NRGBA{R: 0x2a, G: 0x3a, B: 0x55, A: 0xff}, a)
	ink := withAlpha(color.NRGBA{R: 0x12, G: 0x1a, B: 0x2c, A: 0xff}, a)
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, panel)
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, 3, accent)
	drawSlowMoHourglass(screen, r.X+17, r.Y+16, 0.48, accent, ink)
	drawSourceMenuText(screen, "SLOW-MO", menuFontCondensedExtraBold, 15, withAlpha(color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, a), r.X+32, r.Y+7)
	barX, barW := r.X+110, r.W-122
	ebitenutil.DrawRect(screen, barX, r.Y+12, barW, 8, track)
	left := float64(g.slowMo.left) / slowMoRealTicks
	ebitenutil.DrawRect(screen, barX, r.Y+12, barW*math.Max(0, math.Min(1, left)), 8, accent)
}
