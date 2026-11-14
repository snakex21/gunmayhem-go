package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Burning panic presentation. Procedural, derived only from synced fields
// (BurnTime, BurnMax, BurnTick, BurnPanicTime, Facing, X/Y), so online
// clients draw exactly what the host simulates. The base fire blobs at the
// feet stay in drawPlayerBurn (game.go); this adds body flames licking up the
// torso and arms and a smoke trail above the head.

// drawBurningPanicAll draws the burn overlay for players and burning
// campaign shield-bearers (drones only smoke briefly through drawPlayerBurn).
func (g *Game) drawBurningPanicAll(screen *ebiten.Image) {
	for _, p := range g.players {
		if p != nil && p.Active && p.BurnTime > 0 {
			g.drawBurningPanic(screen, p)
		}
	}
	for _, e := range g.campaignEnemies {
		if e == nil || e.Dead || !e.Body.Active || e.Body.BurnTime <= 0 {
			continue
		}
		if campaignEnemyKind(e.Kind) == campaignEnemyShieldBearer {
			g.drawBurningPanic(screen, &e.Body)
		}
	}
}

func (g *Game) drawBurningPanic(screen *ebiten.Image, p *Player) {
	scale := playerRenderScale(p) / 0.8
	x, feet := g.worldX(p.X), g.worldY(p.Y)
	tick := float64(p.BurnTick)
	fade := 1.0
	if p.BurnTime < 20 {
		fade = float64(p.BurnTime) / 20 // dying flames
	}
	alpha := float32(fade)
	if p.Alpha < 1 {
		alpha *= float32(math.Max(0.25, p.Alpha))
	}

	// Body flames: tongues rise from the torso, hands and head and loop.
	type anchor struct{ lx, ly, size float64 }
	anchors := [...]anchor{
		{-6, -22, 1.0}, {6, -26, 1.0}, {0, -40, 1.15}, {-8, -50, 0.9},
		{8, -48, 0.9}, {0, -62, 0.85}, {-14, -70, 0.7}, {14, -66, 0.7},
	}
	if p.BurnPanicTime > 0 {
		// Flailing: flames flare higher around the raised arms.
		anchors[6].ly -= 8
		anchors[7].ly -= 8
	}
	facing := float64(p.Facing)
	if facing == 0 {
		facing = 1
	}
	for i, a := range anchors {
		phase := math.Mod(tick*0.11+float64(i)*0.37, 1) // 0 at base -> 1 at tip
		sway := math.Sin(tick*0.45+float64(i)*1.7) * 3
		fx := x + (a.lx*facing+sway)*scale
		fy := feet + (a.ly-phase*14)*scale
		r := (5.5 - phase*3.5) * a.size * scale
		drawPuff(screen, fx, fy, r*2.2, 1, 0.32, 0.05, 0.55*alpha*float32(1-phase))
		drawPuff(screen, fx, fy+1, r*1.2, 1, 0.82, 0.25, 0.85*alpha*float32(1-phase))
	}

	// Smoke puffs drifting up from above the head, leaning against the run.
	drift := -facing
	if p.BurnPanicTime > 0 && p.BurnPanicDir != 0 {
		drift = -float64(p.BurnPanicDir)
	}
	for k := 0; k < 4; k++ {
		t := math.Mod(tick/30+float64(k)*0.25, 1)
		sx := x + (drift*t*18+math.Sin(tick*0.07+float64(k))*3)*scale
		sy := feet - (78+t*34)*scale
		drawPuff(screen, sx, sy, (5+t*11)*scale, 0.32, 0.30, 0.30, 0.55*alpha*float32(1-t))
	}
}

// drawHUDBurning is the HUD card burn indicator: an orange remaining-time bar
// (same layout as the blind / cough bars) with a flicker while panicking.
func drawHUDBurning(screen *ebiten.Image, p *Player, x, y float64) {
	if p == nil || p.BurnTime <= 0 {
		return
	}
	barX, barW := x-36.0, 132.0
	ref := p.BurnMax
	if ref < p.BurnTime {
		ref = p.BurnTime
	}
	frac := float64(p.BurnTime) / float64(ref)
	if p.BurnPanicTime > 0 && (p.BurnTick/3)%2 == 0 {
		ebitenutil.DrawRect(screen, x-40, y-34, 140, 58, color.NRGBA{R: 0xff, G: 0x60, B: 0x10, A: 0x50})
	}
	ebitenutil.DrawRect(screen, barX, y+26, barW, 9, color.NRGBA{A: 0xb0})
	ebitenutil.DrawRect(screen, barX+1, y+27, (barW-2)*frac, 7, color.NRGBA{R: 0xff, G: 0x7a, B: 0x1a, A: 0xff})
	drawSourceHUDText(screen, "BURNING", 9, color.NRGBA{R: 0x2a, G: 0x0c, B: 0x00, A: 0xff}, barX+3, y+25)
}
