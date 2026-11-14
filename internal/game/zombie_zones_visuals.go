package game

import (
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// zombie_zones_visuals.go: doors (barricade / gate drawn in code, price
// sign, opening animation), the shared-screen gather prompt and the zone
// name on the HUD. No image files.

var (
	zombieDoorWood     = rgb(0x8a, 0x5a, 0x2e)
	zombieDoorWoodDark = rgb(0x5c, 0x3a, 0x1c)
	zombieDoorMetal    = rgb(0x8c, 0x92, 0x96)
	zombieDoorGlow     = color.NRGBA{R: 0x7c, G: 0xff, B: 0x6a, A: 0xff}
)

const (
	zombieDoorHalfW = 30.0
	zombieDoorH     = 86.0
)

// drawZombieDoors draws every door side that lies in the zone on screen.
func (g *Game) drawZombieDoors(screen *ebiten.Image) {
	m := g.zombie
	if !m.multiZone() {
		return
	}
	view := g.zombieViewZone()
	for di := range m.arena.Doors {
		if di >= zombieMaxDoors {
			break
		}
		d := &m.arena.Doors[di]
		here, toZone, _, ok := d.end(view)
		if !ok {
			continue
		}
		g.drawZombieDoor(screen, di, d, here, toZone)
	}
}

func (g *Game) drawZombieDoor(screen *ebiten.Image, di int, d *zombieDoorDef, here zombiePoint, toZone int) {
	m := g.zombie
	st := m.hud.Doors[di]
	x, y := g.worldX(here.X), g.worldY(here.Y)
	if x < -120 || x > ScreenWidth+120 || y < -40 || y > ScreenHeight+140 {
		return
	}
	top := y - zombieDoorH
	left, right := x-zombieDoorHalfW, x+zombieDoorHalfW
	dest := ""
	if z := m.arena.zone(toZone); z != nil {
		dest = z.Name
	}
	pulse := 0.5 + 0.5*math.Sin(float64(g.mapFXFrame+m.hud.Kills*3+di*17)*0.12)

	// Doorway: dark inside, brighter when open.
	inside := color.NRGBA{R: 0x10, G: 0x0c, B: 0x0a, A: 0xe6}
	vector.DrawFilledRect(screen, float32(left), float32(top), float32(2*zombieDoorHalfW), float32(zombieDoorH), inside, true)
	if st.Open {
		open := 1.0
		if st.Anim > 0 {
			open = 1 - float64(st.Anim)/zombieDoorOpenFrames
		}
		glow := zombieDoorGlow
		glow.A = uint8(40 + 50*pulse*open)
		vector.DrawFilledRect(screen, float32(left+4), float32(top+4), float32(2*zombieDoorHalfW-8), float32(zombieDoorH-4), glow, true)
		// Chevrons pointing through the door.
		for i := 0; i < 3; i++ {
			cy := top + 26 + float64(i)*16
			a := withAlpha(zombieDoorGlow, open*(0.45+0.4*math.Sin(float64(g.mapFXFrame)*0.2-float64(i))))
			vector.StrokeLine(screen, float32(x-8), float32(cy-6), float32(x+2), float32(cy), 3, a, true)
			vector.StrokeLine(screen, float32(x+2), float32(cy), float32(x-8), float32(cy+6), 3, a, true)
		}
	}
	// Frame: two posts and a lintel.
	for _, px := range []float64{left - 6, right} {
		vector.DrawFilledRect(screen, float32(px-2), float32(top-8), 10, float32(zombieDoorH+8), zombieInk, true)
		vector.DrawFilledRect(screen, float32(px), float32(top-6), 6, float32(zombieDoorH+6), zombieDoorWoodDark, true)
	}
	vector.DrawFilledRect(screen, float32(left-10), float32(top-16), float32(2*zombieDoorHalfW+20), 12, zombieInk, true)
	vector.DrawFilledRect(screen, float32(left-8), float32(top-14), float32(2*zombieDoorHalfW+16), 8, zombieDoorWood, true)

	// Barricade boards: they fly apart and fade while the door opens.
	if !st.Open || st.Anim > 0 {
		k := 0.0
		if st.Open {
			k = 1 - float64(st.Anim)/zombieDoorOpenFrames
		}
		alpha := 1 - k
		boards := [4][4]float64{ // y offset, tilt, x drift sign, fall speed
			{18, -0.18, -1, 1.0},
			{38, 0.14, 1, 1.3},
			{58, -0.1, -1, 0.8},
			{74, 0.2, 1, 1.1},
		}
		for _, bd := range boards {
			cx := x + bd[2]*k*70
			cy := top + bd[0] + k*k*90*bd[3]
			tilt := bd[1] + bd[2]*k*1.4
			g.drawDoorBoard(screen, cx, cy, 2*zombieDoorHalfW+16, 11, tilt, alpha)
		}
		if !st.Open {
			// Padlock on the middle board.
			lx, ly := x, top+44
			vector.DrawFilledRect(screen, float32(lx-8), float32(ly-2), 16, 13, zombieInk, true)
			vector.DrawFilledRect(screen, float32(lx-6), float32(ly), 12, 9, rgb(0xd0, 0xa8, 0x30), true)
			var shackle vector.Path
			shackle.Arc(float32(lx), float32(ly-2), 5, math.Pi, 2*math.Pi, vector.Clockwise)
			strokePathColor(screen, &shackle, zombieDoorMetal, 2.5)
		}
	}

	// Sign above the door: price while closed, the destination when open.
	sign := strconv.Itoa(d.Cost)
	signClr := color.NRGBA{R: 0xff, G: 0xd8, B: 0x40, A: 0xff}
	if st.Open {
		sign = dest
		signClr = zombieDoorGlow
	}
	w := uiMeasure(true, 15, sign) + 18
	r := Rect{X: x - w/2, Y: top - 44, W: w, H: 24}
	uiFillRound(screen, r, 4, color.NRGBA{R: 0x10, G: 0x12, B: 0x14, A: 0xe0})
	uiFillRect(screen, Rect{X: r.X, Y: r.Y + r.H - 3, W: r.W, H: 3}, withAlpha(signClr, 0.8))
	uiTextIn(screen, sign, true, 15, signClr, r, uiAlignCenter, 4)

	// Prompt for a local player at the door.
	for _, p := range g.players {
		if p == nil || !p.Active || p.AI || !g.zombieLocalViewer(p) || !m.doorInReach(p, di) {
			continue
		}
		zp := m.playerState(p.ID)
		text := ""
		switch {
		case !st.Open && d.OpenFrom > 0 && m.hud.Wave < d.OpenFrom:
			text = "LOCKED UNTIL WAVE " + strconv.Itoa(d.OpenFrom)
		case !st.Open && zp != nil && zp.Points < d.Cost:
			text = "OPEN " + d.Name + "  " + strconv.Itoa(d.Cost) + " - NOT ENOUGH POINTS"
		case !st.Open:
			text = "DOWN: OPEN " + d.Name + "  " + strconv.Itoa(d.Cost)
		case m.host && g.zombieSharedScreen():
			text = "DOWN: ALL PLAYERS TO " + dest
		default:
			text = "DOWN: GO TO " + dest
		}
		drawZombieTag(screen, text, x, top-66)
		break
	}
}

func (g *Game) drawDoorBoard(screen *ebiten.Image, cx, cy, w, h, tilt, alpha float64) {
	if alpha <= 0.01 {
		return
	}
	c, s := math.Cos(tilt), math.Sin(tilt)
	corner := func(dx, dy float64) [2]float64 { return [2]float64{cx + dx*c - dy*s, cy + dx*s + dy*c} }
	outline := [][2]float64{corner(-w/2-2, -h/2-2), corner(w/2+2, -h/2-2), corner(w/2+2, h/2+2), corner(-w/2-2, h/2+2)}
	fillPolygon(screen, outline, withAlpha(zombieInk, alpha))
	board := [][2]float64{corner(-w/2, -h/2), corner(w/2, -h/2), corner(w/2, h/2), corner(-w/2, h/2)}
	fillPolygon(screen, board, withAlpha(zombieDoorWood, alpha))
	// Grain line and nails.
	a, b := corner(-w/2+4, 0), corner(w/2-4, 0)
	vector.StrokeLine(screen, float32(a[0]), float32(a[1]), float32(b[0]), float32(b[1]), 1, withAlpha(zombieDoorWoodDark, alpha), true)
	for _, nx := range []float64{-w/2 + 6, w/2 - 6} {
		n := corner(nx, 0)
		vector.DrawFilledCircle(screen, float32(n[0]), float32(n[1]), 1.8, withAlpha(zombieDoorMetal, alpha), true)
	}
}

// drawZombieZoneHUD: the zone name under the wave panel and the gather
// prompt of a shared screen.
func (g *Game) drawZombieZoneHUD(screen *ebiten.Image) {
	m := g.zombie
	if !m.multiZone() {
		return
	}
	if z := m.arena.zone(g.zombieViewZone()); z != nil {
		// Left of the wave panel (the boss bar sits under it).
		uiText(screen, z.Name, true, uiSizeButton, zombieDoorGlow, 340, 22, uiAlignRight)
	}
	gs := m.hud.Gather
	if gs.Door == 0 || gs.Need == 0 {
		return
	}
	dest := ""
	if di := gs.Door - 1; di < len(m.arena.Doors) {
		if _, to, _, ok := m.arena.Doors[di].end(g.zombieViewZone()); ok {
			if z := m.arena.zone(to); z != nil {
				dest = z.Name
			}
		}
	}
	text := "ALL PLAYERS TO THE DOOR  " + strconv.Itoa(gs.At) + "/" + strconv.Itoa(gs.Need)
	if dest != "" {
		text += "  -  " + dest
	}
	w := math.Max(300, uiMeasure(true, uiSizeButton, text)+40)
	r := Rect{X: ScreenWidth/2 - w/2, Y: 96, W: w, H: 40}
	uiFillRound(screen, r, 6, color.NRGBA{R: 0x10, G: 0x12, B: 0x14, A: 0xd8})
	uiText(screen, text, true, uiSizeButton, uiTextLight, r.X+r.W/2, r.Y+15, uiAlignCenter)
	bar := Rect{X: r.X + 12, Y: r.Y + r.H - 10, W: r.W - 24, H: 5}
	uiFillRound(screen, bar, 2, color.NRGBA{R: 0x30, G: 0x34, B: 0x38, A: 0xff})
	frac := float64(gs.Progress) / zombieGatherFrames
	if gs.At < gs.Need {
		frac = float64(gs.At) / float64(gs.Need) * 0.3
	}
	if frac > 0 {
		uiFillRound(screen, Rect{X: bar.X, Y: bar.Y, W: bar.W * math.Min(1, frac), H: bar.H}, 2, zombieDoorGlow)
	}
}
