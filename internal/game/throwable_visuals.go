package game

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	throwableVisualOnce       sync.Once
	throwableImages           [throwableTypeCount]*ebiten.Image
	throwableDynamiteSetupArt *SourceRaster
)

func ensureThrowableVisuals() {
	throwableVisualOnce.Do(func() {
		throwableDynamiteSetupArt = loadSourceRaster("Symbol 665", 0, "sprites", "DefineSprite_665_wep_grenade", "1.png")
		// Dynamite keeps the original source art. The added throwables use polished
		// generated sprites; compact vector drawings remain as a portable fallback
		// if an optional custom asset is missing from an incomplete installation.
		throwableImages[ThrowableMk2] = loadGeneratedThrowableImage("mk2.png")
		throwableImages[ThrowableF1] = loadGeneratedThrowableImage("f1.png")
		throwableImages[ThrowableStielhand] = loadGeneratedThrowableImage("m24.png")
		throwableImages[ThrowableRGD5] = loadGeneratedThrowableImage("rgd5.png")
		throwableImages[ThrowableMolotov] = loadGeneratedThrowableImage("molotov.png")

		if throwableImages[ThrowableMk2] == nil {
			mk2 := ebiten.NewImage(24, 32)
			vector.DrawFilledCircle(mk2, 12, 17, 10, color.NRGBA{R: 0x22, G: 0x28, B: 0x18, A: 0xff}, true)
			vector.DrawFilledRect(mk2, 8, 3, 8, 7, color.NRGBA{R: 0x49, G: 0x4f, B: 0x38, A: 0xff}, true)
			for y := float32(10); y < 27; y += 5 {
				vector.StrokeLine(mk2, 4, y, 20, y, 1.2, color.NRGBA{R: 0x70, G: 0x78, B: 0x52, A: 0xff}, true)
			}
			throwableImages[ThrowableMk2] = mk2
		}

		if throwableImages[ThrowableF1] == nil {
			f1 := ebiten.NewImage(24, 32)
			vector.DrawFilledCircle(f1, 12, 17, 10, color.NRGBA{R: 0x28, G: 0x32, B: 0x1b, A: 0xff}, true)
			vector.DrawFilledRect(f1, 8, 3, 8, 7, color.NRGBA{R: 0x35, G: 0x3a, B: 0x2b, A: 0xff}, true)
			vector.StrokeCircle(f1, 12, 17, 7, 1.5, color.NRGBA{R: 0x75, G: 0x82, B: 0x4c, A: 0xff}, true)
			throwableImages[ThrowableF1] = f1
		}

		if throwableImages[ThrowableStielhand] == nil {
			m24 := ebiten.NewImage(42, 16)
			vector.DrawFilledRect(m24, 2, 5, 26, 7, color.NRGBA{R: 0x8a, G: 0x63, B: 0x32, A: 0xff}, true)
			vector.DrawFilledRect(m24, 26, 2, 13, 13, color.NRGBA{R: 0x4c, G: 0x54, B: 0x35, A: 0xff}, true)
			vector.StrokeRect(m24, 26, 2, 13, 13, 1.5, color.NRGBA{R: 0x12, G: 0x12, B: 0x10, A: 0xff}, true)
			throwableImages[ThrowableStielhand] = m24
		}

		if throwableImages[ThrowableRGD5] == nil {
			rgd := ebiten.NewImage(24, 30)
			vector.DrawFilledCircle(rgd, 12, 17, 9, color.NRGBA{R: 0x55, G: 0x63, B: 0x31, A: 0xff}, true)
			vector.DrawFilledRect(rgd, 8, 4, 8, 7, color.NRGBA{R: 0x38, G: 0x3c, B: 0x2c, A: 0xff}, true)
			throwableImages[ThrowableRGD5] = rgd
		}

		if throwableImages[ThrowableMolotov] == nil {
			molotov := ebiten.NewImage(18, 42)
			vector.DrawFilledRect(molotov, 6, 3, 6, 11, color.NRGBA{R: 0x4c, G: 0x28, B: 0x18, A: 0xff}, true)
			vector.DrawFilledCircle(molotov, 9, 27, 8, color.NRGBA{R: 0x8a, G: 0xb6, B: 0xa7, A: 0xc8}, true)
			vector.DrawFilledRect(molotov, 3, 20, 12, 10, color.NRGBA{R: 0xb5, G: 0x46, B: 0x19, A: 0xd0}, true)
			vector.DrawFilledCircle(molotov, 9, 2, 4, color.NRGBA{R: 0xff, G: 0x70, B: 0x0c, A: 0xff}, true)
			throwableImages[ThrowableMolotov] = molotov
		}
	})
}

func loadGeneratedThrowableImage(filename string) *ebiten.Image {
	src := decodeOriginalPNGIn("custom", "throwables", filename)
	if src == nil {
		return nil
	}
	bounds, ok := alphaBounds(src)
	if !ok || bounds.Empty() {
		return nil
	}
	// Strip the transparent generation canvas before uploading the texture.
	// This keeps memory use small and makes all later fit calculations operate
	// on the visible sprite rather than its original square padding.
	trimmed := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(trimmed, trimmed.Bounds(), src, bounds.Min, draw.Src)
	return ebiten.NewImageFromImage(trimmed)
}

func throwableSetupChoiceRect(cardX, cardY float64, kind int) Rect {
	if kind < 0 || kind >= throwableTypeCount {
		return Rect{}
	}
	// Six visual choices fit exactly in the old 165px throwable selector row.
	return Rect{X: cardX + 17.5 + float64(kind)*27.5, Y: cardY + 245, W: 25, H: 22}
}

func (g *Game) drawThrowableSetupChoice(dst *ebiten.Image, kind int, r Rect, selected bool) {
	if dst == nil || r.W <= 0 || r.H <= 0 || kind < 0 || kind >= throwableTypeCount {
		return
	}
	border := color.NRGBA{R: 0xa7, G: 0xa7, B: 0xa7, A: 0xff}
	fill := color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}
	if selected {
		border = color.NRGBA{R: 0xff, G: 0x91, B: 0x00, A: 0xff}
		fill = color.NRGBA{R: 0xff, G: 0xdd, B: 0xa8, A: 0xff}
	}
	ebitenutil.DrawRect(dst, r.X, r.Y, r.W, r.H, border)
	ebitenutil.DrawRect(dst, r.X+2, r.Y+2, r.W-4, r.H-4, fill)
	// Reserve a dedicated strip at the bottom for the throwable name. Keeping
	// the sprite out of that strip prevents long names from being printed over
	// the artwork, especially on the tall F-1/RGD-5/Molotov sprites.
	innerH := r.H - 27
	if innerH < 10 {
		innerH = r.H - 6
	}
	inner := Rect{X: r.X + 4, Y: r.Y + 4, W: r.W - 8, H: innerH}
	ensureThrowableVisuals()
	if kind == ThrowableDynamite {
		art := throwableDynamiteSetupArt
		if g != nil && g.assets != nil && g.assets.Grenade != nil {
			art = g.assets.Grenade
		}
		if art != nil {
			drawSetupThumbFit(dst, art, inner, 1)
			return
		}
		drawSourceMenuText(dst, "T", menuFontCondensedExtraBold, 12, color.Black, inner.X+6, inner.Y+3)
		return
	}
	img := throwableImages[kind]
	if img == nil {
		return
	}
	w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	if w <= 0 || h <= 0 {
		return
	}
	s := math.Min(inner.W/w, inner.H/h)
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(inner.X+(inner.W-w*s)/2, inner.Y+(inner.H-h*s)/2)
	dst.DrawImage(img, op)
}

func throwableGameplaySize(kind int) (float64, float64) {
	switch normalizedThrowableType(kind) {
	case ThrowableMk2, ThrowableF1:
		return 34, 44
	case ThrowableStielhand:
		return 62, 28
	case ThrowableRGD5:
		return 34, 42
	case ThrowableMolotov:
		return 31, 58
	default:
		return 40, 48
	}
}

func (g *Game) drawThrowable(screen *ebiten.Image, gr Grenade) {
	kind := normalizedThrowableType(gr.Type)
	if kind == ThrowableDynamite && g.assets.Grenade != nil {
		drawSourceRasterRot(screen, g.assets.Grenade, g.worldX(gr.X), g.worldY(gr.Y), .8, .8, gr.Rotation, 1)
		return
	}
	ensureThrowableVisuals()
	img := throwableImages[kind]
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	op.GeoM.Translate(-w/2, -h/2)
	maxW, maxH := throwableGameplaySize(kind)
	scale := math.Min(maxW/w, maxH/h)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Rotate(gr.Rotation * math.Pi / 180)
	op.GeoM.Translate(g.worldX(gr.X), g.worldY(gr.Y))
	screen.DrawImage(img, op)
}

func (g *Game) drawFirePatches(screen *ebiten.Image) {
	for _, patch := range g.firePatches {
		if patch.Dead || patch.Life <= 0 {
			continue
		}
		x, y := float32(g.worldX(patch.X)), float32(g.worldY(patch.Y))
		pulse := float32((patch.Frame/3)%3) * 1.4
		vector.DrawFilledCircle(screen, x, y-7-pulse, 12+pulse, color.NRGBA{R: 0xff, G: 0x35, B: 0x08, A: 0x88}, true)
		vector.DrawFilledCircle(screen, x+3, y-8-pulse, 7+pulse*.5, color.NRGBA{R: 0xff, G: 0x9b, B: 0x12, A: 0xe0}, true)
		vector.DrawFilledCircle(screen, x+5, y-10-pulse, 3.2, color.NRGBA{R: 0xff, G: 0xef, B: 0x74, A: 0xff}, true)
	}
}
