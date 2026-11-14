package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Grenade/Molotov presentation: throw-charge preview, HUD throwable count and
// the animated Molotov fire. Everything here is derived from state that is
// already in the online snapshot (Player.GrenadeHeld/GrenadePower/Grenades,
// FirePatch.Frame/Life), so clients render it without protocol changes.

const grenadeMaxPower = 14.0

// grenadePreviewPoint is one sampled position of the predicted throw.
type grenadePreviewPoint struct{ X, Y float64 }

// grenadeThrowPreview predicts the flight of a grenade released now. It is
// the shared throw simulator (simulateThrow: the real spawn point, launch
// velocity, gravity, bounce and fuse code of updateGrenades) cut at the first
// platform contact (landed=true), at detonation, or after maxSteps frames.
func grenadeThrowPreview(arena Map, p *Player, maxSteps int) ([]grenadePreviewPoint, bool) {
	res := simulateGrenade(arena, throwSpawnGrenade(p.X, p.Y, p.Facing, p.GrenadePower, p.ThrowableType), true)
	points := res.Path
	landed := false
	if res.LandIndex >= 0 && res.LandIndex < maxSteps {
		points = points[:res.LandIndex+1]
		landed = true
	} else if len(points) > maxSteps {
		points = points[:maxSteps]
	}
	return points, landed
}

func (g *Game) drawGrenadeChargeIndicators(screen *ebiten.Image) {
	arena := g.customRuntimeArena()
	for _, p := range g.players {
		if p == nil || !p.Active || !p.GrenadeHeld || p.Grenades <= 0 {
			continue
		}
		frac := (p.GrenadePower - 1) / (grenadeMaxPower - 1)
		frac = math.Max(0, math.Min(1, frac))
		// Charge colour: green (short lob, still dangerous to the thrower)
		// through yellow to red at full power.
		r := uint8(80 + 175*frac)
		gr := uint8(230 - 120*frac)
		charge := color.NRGBA{R: r, G: gr, B: 40, A: 230}

		// Dotted trajectory preview.
		points, landed := grenadeThrowPreview(arena, p, 70)
		for i, pt := range points {
			if i%3 != 2 && i != len(points)-1 {
				continue
			}
			alpha := uint8(220 - 150*float64(i)/float64(len(points)+1))
			vector.DrawFilledCircle(screen, float32(g.worldX(pt.X)), float32(g.worldY(pt.Y)), 2.4,
				color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: alpha}, true)
			vector.StrokeCircle(screen, float32(g.worldX(pt.X)), float32(g.worldY(pt.Y)), 2.6, 1,
				color.NRGBA{A: alpha / 2}, true)
		}
		if landed && len(points) > 0 {
			end := points[len(points)-1]
			ex, ey := float32(g.worldX(end.X)), float32(g.worldY(end.Y+8))
			vector.StrokeLine(screen, ex-6, ey-6, ex+6, ey+6, 2, charge, true)
			vector.StrokeLine(screen, ex-6, ey+6, ex+6, ey-6, 2, charge, true)
		}

		// Charge meter above the head.
		scale := playerRenderScale(p)
		cx := float32(g.worldX(p.X))
		cy := float32(g.worldY(p.Y - 110*scale/0.8))
		const w, h = 40, 6
		vector.DrawFilledRect(screen, cx-w/2-1, cy-1, w+2, h+2, color.NRGBA{A: 190}, false)
		vector.DrawFilledRect(screen, cx-w/2, cy, float32(w*frac), h, charge, false)
		if p.GrenadePower >= grenadeMaxPower {
			vector.StrokeRect(screen, cx-w/2-1, cy-1, w+2, h+2, 1, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, false)
		}
	}
}

// drawHUDThrowableCount draws "<icon> xN" for the player's remaining
// throwables to the right of the ammo counter on the HUD card.
func (g *Game) drawHUDThrowableCount(screen *ebiten.Image, p *Player, x, y float64) {
	if p == nil || p.GameMode == SourceGameModeInstagib || p.GameMode == SourceGameModeGunGame {
		return
	}
	iconX, iconY := x+62, y-1.5
	kind := normalizedThrowableType(p.ThrowableType)
	alpha := 1.0
	if p.Grenades <= 0 {
		alpha = 0.35
	}
	ensureThrowableVisuals()
	if kind == ThrowableDynamite && g.assets != nil && g.assets.Grenade != nil {
		drawSourceRasterRot(screen, g.assets.Grenade, iconX, iconY, .32, .32, 25, alpha)
	} else if img := throwableImages[kind]; img != nil {
		w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
		s := math.Min(16/w, 18/h)
		op := &ebiten.DrawImageOptions{}
		applySourceRenderQuality(op)
		op.GeoM.Translate(-w/2, -h/2)
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(iconX, iconY)
		op.ColorScale.ScaleAlpha(float32(alpha))
		screen.DrawImage(img, op)
	}
	clr := color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xff}
	if p.Grenades <= 0 {
		clr = color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}
	}
	drawSourceHUDText(screen, fmt.Sprintf("x%d", p.Grenades), 12, clr, iconX+8, y-9)
}

// fireHash gives each patch a stable per-position seed so neighbouring patches
// flicker out of phase. It only depends on snapshot data.
func fireHash(x float64, i int) int {
	h := int(math.Abs(x))*73856093 ^ (i+1)*19349663
	if h < 0 {
		h = -h
	}
	return h
}

func (g *Game) drawFirePatches(screen *ebiten.Image) {
	for _, patch := range g.firePatches {
		if patch.Dead || patch.Life <= 0 {
			continue
		}
		x, y := float32(g.worldX(patch.X)), float32(g.worldY(patch.Y))
		fade := float32(1)
		if patch.Life < 40 {
			fade = float32(patch.Life) / 40
		}
		grow := float32(1)
		if patch.Frame < 8 {
			grow = .35 + .65*float32(patch.Frame)/8
		}

		// Impact burst: a quick bright ring as the fuel ignites.
		if patch.Frame < 12 {
			t := float32(patch.Frame) / 12
			vector.StrokeCircle(screen, x, y-8, 8+26*t, 3*(1-t)+1,
				color.NRGBA{R: 0xff, G: 0xd0, B: 0x50, A: uint8(220 * (1 - t))}, true)
		}

		// Base glow on the floor.
		vector.DrawFilledRect(screen, x-24*grow, y-4, 48*grow, 4,
			color.NRGBA{R: 0xff, G: 0x44, B: 0x08, A: uint8(150 * fade)}, true)
		vector.DrawFilledCircle(screen, x, y-6, 15*grow,
			color.NRGBA{R: 0xff, G: 0x30, B: 0x06, A: uint8(70 * fade)}, true)

		// Flame tongues: deterministic rising/shrinking particles. Each tongue
		// loops over `period` frames with its own phase and lateral offset.
		const tongues = 6
		for i := 0; i < tongues; i++ {
			seed := fireHash(patch.X, i)
			period := 16 + seed%9
			phase := float32((patch.Frame+seed%period)%period) / float32(period) // 0..1 rising
			lateral := float32(seed%31-15) * .9 * grow
			sway := float32(math.Sin(float64(patch.Frame+i*7)*.35)) * 2.5 * phase
			px := x + lateral + sway
			py := y - 4 - phase*30*grow
			radius := (7.5 - 5.5*phase) * grow
			var c color.NRGBA
			switch {
			case phase < .3:
				c = color.NRGBA{R: 0xff, G: 0xe8, B: 0x70, A: 235}
			case phase < .6:
				c = color.NRGBA{R: 0xff, G: 0x9a, B: 0x14, A: 210}
			default:
				c = color.NRGBA{R: 0xe8, G: 0x3a, B: 0x0a, A: uint8(200 * (1 - phase) / .4)}
			}
			c.A = uint8(float32(c.A) * fade)
			vector.DrawFilledCircle(screen, px, py, radius, c, true)
		}
		// Hot core.
		vector.DrawFilledCircle(screen, x, y-6, 4.5*grow,
			color.NRGBA{R: 0xff, G: 0xf4, B: 0xb0, A: uint8(230 * fade)}, true)
		// Occasional ember drifting up.
		if seed := fireHash(patch.X, patch.Frame/20); seed%3 == 0 {
			t := float32(patch.Frame%20) / 20
			vector.DrawFilledCircle(screen, x+float32(seed%21-10), y-12-t*34, 1.4,
				color.NRGBA{R: 0xff, G: 0xc0, B: 0x40, A: uint8(220 * (1 - t) * fade)}, true)
		}
	}
}
