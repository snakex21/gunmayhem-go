package game

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	developedProjectileVisualOnce sync.Once
	developedRocketFrames         []*ebiten.Image
	developedFlareFrames          []*ebiten.Image
	developedEnergyFrames         []*ebiten.Image
	developedFlameFrames          []*ebiten.Image
	developedPortalFrames         []*ebiten.Image
	developedGrenadeImage         *ebiten.Image
	developedBoltImage            *ebiten.Image
)

func ensureDevelopedProjectileVisuals() {
	developedProjectileVisualOnce.Do(func() {
		for frame := 0; frame < 4; frame++ {
			developedRocketFrames = append(developedRocketFrames, makeDevelopedRocketFrame(frame))
			developedFlareFrames = append(developedFlareFrames, makeDevelopedFlareFrame(frame))
			developedEnergyFrames = append(developedEnergyFrames, makeDevelopedEnergyFrame(frame))
			developedFlameFrames = append(developedFlameFrames, makeDevelopedFlameFrame(frame))
			developedPortalFrames = append(developedPortalFrames, makeDevelopedPortalFrame(frame))
		}
		developedGrenadeImage = makeDevelopedGrenadeRound()
		developedBoltImage = makeDevelopedBolt()
	})
}

func filledRect(dst *ebiten.Image, x, y, width, height float32, clr color.Color) {
	vector.DrawFilledRect(dst, x, y, width, height, clr, false)
}

func filledCircle(dst *ebiten.Image, x, y, radius float32, clr color.Color) {
	vector.DrawFilledCircle(dst, x, y, radius, clr, false)
}

func makeDevelopedRocketFrame(frame int) *ebiten.Image {
	img := ebiten.NewImage(42, 22)
	flameLength := []float32{7, 11, 8, 13}[frame&3]
	filledCircle(img, 11-flameLength*.48, 11, flameLength*.48, color.NRGBA{R: 0xff, G: 0x42, B: 0x0b, A: 0xe8})
	filledCircle(img, 11-flameLength*.25, 11, flameLength*.30, color.NRGBA{R: 0xff, G: 0xd2, B: 0x37, A: 0xff})

	// Black source-style silhouette with a compact gray rocket body.
	filledRect(img, 10, 6, 23, 10, color.NRGBA{R: 0x0d, G: 0x0d, B: 0x0d, A: 0xff})
	filledRect(img, 11, 7, 21, 8, color.NRGBA{R: 0x65, G: 0x69, B: 0x68, A: 0xff})
	filledRect(img, 10, 3, 8, 5, color.NRGBA{R: 0x12, G: 0x12, B: 0x12, A: 0xff})
	filledRect(img, 10, 14, 8, 5, color.NRGBA{R: 0x12, G: 0x12, B: 0x12, A: 0xff})
	filledCircle(img, 33, 11, 6, color.NRGBA{R: 0x0d, G: 0x0d, B: 0x0d, A: 0xff})
	filledCircle(img, 33, 11, 4.5, color.NRGBA{R: 0xb1, G: 0x3f, B: 0x25, A: 0xff})
	filledRect(img, 18, 7, 3, 8, color.NRGBA{R: 0x2b, G: 0x2d, B: 0x2d, A: 0xff})
	return img
}

func makeDevelopedGrenadeRound() *ebiten.Image {
	img := ebiten.NewImage(24, 14)
	filledCircle(img, 5, 7, 5, color.NRGBA{R: 0x0b, G: 0x0d, B: 0x0b, A: 0xff})
	filledCircle(img, 19, 7, 5, color.NRGBA{R: 0x0b, G: 0x0d, B: 0x0b, A: 0xff})
	filledRect(img, 5, 2, 14, 10, color.NRGBA{R: 0x0b, G: 0x0d, B: 0x0b, A: 0xff})
	filledCircle(img, 5, 7, 3.5, color.NRGBA{R: 0x58, G: 0x64, B: 0x38, A: 0xff})
	filledCircle(img, 19, 7, 3.5, color.NRGBA{R: 0x58, G: 0x64, B: 0x38, A: 0xff})
	filledRect(img, 5, 3.5, 14, 7, color.NRGBA{R: 0x58, G: 0x64, B: 0x38, A: 0xff})
	filledRect(img, 9, 3, 2, 8, color.NRGBA{R: 0xc1, G: 0xa4, B: 0x45, A: 0xff})
	filledRect(img, 14, 3, 2, 8, color.NRGBA{R: 0x27, G: 0x2b, B: 0x22, A: 0xff})
	return img
}

func makeDevelopedBolt() *ebiten.Image {
	img := ebiten.NewImage(36, 12)
	filledRect(img, 4, 5, 26, 2, color.NRGBA{R: 0x1a, G: 0x1a, B: 0x19, A: 0xff})
	filledRect(img, 5, 5, 24, 1, color.NRGBA{R: 0xb6, G: 0x82, B: 0x45, A: 0xff})
	filledRect(img, 2, 2, 5, 4, color.NRGBA{R: 0x63, G: 0x28, B: 0x20, A: 0xff})
	filledRect(img, 2, 6, 5, 4, color.NRGBA{R: 0x63, G: 0x28, B: 0x20, A: 0xff})
	filledCircle(img, 30, 6, 4, color.NRGBA{R: 0x12, G: 0x12, B: 0x12, A: 0xff})
	filledCircle(img, 31, 6, 2.4, color.NRGBA{R: 0xa9, G: 0xae, B: 0xac, A: 0xff})
	return img
}

func makeDevelopedFlareFrame(frame int) *ebiten.Image {
	img := ebiten.NewImage(28, 18)
	radius := float32(5 + frame%3)
	filledCircle(img, 18, 9, radius, color.NRGBA{R: 0xff, G: 0x74, B: 0x16, A: 0x88})
	filledCircle(img, 18, 9, 3.5, color.NRGBA{R: 0xff, G: 0xe1, B: 0x51, A: 0xff})
	filledRect(img, 6, 5, 12, 8, color.NRGBA{R: 0x15, G: 0x15, B: 0x15, A: 0xff})
	filledRect(img, 7, 6, 10, 6, color.NRGBA{R: 0xd8, G: 0x35, B: 0x1d, A: 0xff})
	return img
}

func makeDevelopedEnergyFrame(frame int) *ebiten.Image {
	img := ebiten.NewImage(32, 18)
	radius := float32(5 + frame%2)
	filledCircle(img, 16, 9, radius+3, color.NRGBA{R: 0x22, G: 0xd8, B: 0xff, A: 0x54})
	filledRect(img, 6, 5, 20, 8, color.NRGBA{R: 0x0b, G: 0x34, B: 0x43, A: 0xff})
	filledRect(img, 8, 7, 16, 4, color.NRGBA{R: 0x65, G: 0xec, B: 0xff, A: 0xff})
	filledCircle(img, 24, 9, radius, color.NRGBA{R: 0xb6, G: 0xf8, B: 0xff, A: 0xff})
	return img
}

func makeDevelopedFlameFrame(frame int) *ebiten.Image {
	img := ebiten.NewImage(38, 26)
	pulse := float32(frame&1) * 2
	filledCircle(img, 18, 13, 11+pulse, color.NRGBA{R: 0xff, G: 0x38, B: 0x08, A: 0x80})
	filledCircle(img, 21, 13, 7+pulse*.5, color.NRGBA{R: 0xff, G: 0x87, B: 0x12, A: 0xee})
	filledCircle(img, 25, 13, 3.6, color.NRGBA{R: 0xff, G: 0xef, B: 0x72, A: 0xff})
	filledCircle(img, 8, 13, 5, color.NRGBA{R: 0xd9, G: 0x24, B: 0x06, A: 0xb0})
	return img
}

func makeDevelopedPortalFrame(frame int) *ebiten.Image {
	img := ebiten.NewImage(26, 26)
	outer := color.NRGBA{R: 0x18, G: 0xa9, B: 0xff, A: 0x82}
	inner := color.NRGBA{R: 0xa8, G: 0xef, B: 0xff, A: 0xff}
	if frame&1 == 1 {
		outer = color.NRGBA{R: 0xff, G: 0x79, B: 0x13, A: 0x82}
		inner = color.NRGBA{R: 0xff, G: 0xdc, B: 0x65, A: 0xff}
	}
	filledCircle(img, 13, 13, 12, outer)
	filledCircle(img, 13, 13, 6, inner)
	return img
}

func (g *Game) drawDevelopedProjectile(screen *ebiten.Image, b *Bullet) bool {
	if b == nil || b.DevelopedProjectile == DevelopedProjectileBullet {
		return false
	}
	ensureDevelopedProjectileVisuals()

	angle := math.Atan2(b.VY, b.VX)
	var img *ebiten.Image
	scale := 1.0
	switch b.DevelopedProjectile {
	case DevelopedProjectileRocket, DevelopedProjectileArcRocket, DevelopedProjectileTopAttack, DevelopedProjectileStrike:
		if b.DevelopedProjectile == DevelopedProjectileStrike {
			g.drawAirstrikeWarning(screen, b)
		}
		img = developedRocketFrames[b.Time%len(developedRocketFrames)]
		// A short animated smoke train makes speed/direction readable even when
		// the camera is moving. Flame length changes every tick in the sprite.
		speed := math.Hypot(b.VX, b.VY)
		if speed > 0 {
			ux, uy := b.VX/speed, b.VY/speed
			for i := 1; i <= 3; i++ {
				distance := float64(7 + i*6)
				x := g.worldX(b.X - ux*distance)
				y := g.worldY(b.Y - uy*distance)
				radius := float32(4-i) + float32((b.Time+i)&1)
				filledCircle(screen, float32(x), float32(y), radius, color.NRGBA{R: 0x8d, G: 0x8d, B: 0x86, A: uint8(105 - i*22)})
			}
		}
	case DevelopedProjectileGrenade:
		img = developedGrenadeImage
		angle += float64(b.Time*b.Facing) * 0.38
		scale = .95
	case DevelopedProjectileBolt:
		img = developedBoltImage
	case DevelopedProjectileFlare:
		img = developedFlareFrames[b.Time%len(developedFlareFrames)]
	case DevelopedProjectileEnergy:
		img = developedEnergyFrames[b.Time%len(developedEnergyFrames)]
		scale = .9 + .08*math.Sin(float64(b.Time)*.8)
	case DevelopedProjectileFlame:
		img = developedFlameFrames[b.Time%len(developedFlameFrames)]
		scale = b.VisualScale
		if scale <= 0 {
			scale = 1
		}
	case DevelopedProjectilePortal:
		frame := 0
		if b.PortalColor == PortalOrange {
			frame = 1
		}
		img = developedPortalFrames[frame]
		scale = .85 + .12*math.Sin(float64(b.Time)*.9)
	default:
		return false
	}

	op := &ebiten.DrawImageOptions{}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Rotate(angle)
	op.GeoM.Translate(g.worldX(b.X), g.worldY(b.Y))
	if b.Alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(b.Alpha))
	}
	screen.DrawImage(img, op)
	return true
}

func (g *Game) drawAirstrikeWarning(screen *ebiten.Image, b *Bullet) {
	if g == nil || screen == nil || b == nil {
		return
	}
	impactY := g.arena.LowestY
	found := false
	for _, platform := range g.arena.Platforms {
		if !platform.ContainsX(b.X) || platform.Y < b.Y {
			continue
		}
		if !found || platform.Y < impactY {
			impactY = platform.Y
			found = true
		}
	}
	x, y := float32(g.worldX(b.X)), float32(g.worldY(impactY))
	pulse := float32(12 + (b.Time%12)*2)
	alpha := uint8(120 + (b.Time%6)*20)
	red := color.NRGBA{R: 0xff, G: 0x28, B: 0x12, A: alpha}
	vector.StrokeCircle(screen, x, y, pulse, 2.5, red, true)
	vector.StrokeLine(screen, x-13, y, x+13, y, 2, red, true)
	vector.StrokeLine(screen, x, y-13, x, y+4, 2, red, true)
}
