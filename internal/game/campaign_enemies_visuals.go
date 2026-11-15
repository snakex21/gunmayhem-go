package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Procedural art for the non-player campaign enemies and hazard markers.
// Everything is drawn from synced state, no sprite files.

func fillPolygon(dst *ebiten.Image, pts [][2]float64, clr color.NRGBA) {
	if len(pts) < 3 {
		return
	}
	var path vector.Path
	path.MoveTo(float32(pts[0][0]), float32(pts[0][1]))
	for _, pt := range pts[1:] {
		path.LineTo(float32(pt[0]), float32(pt[1]))
	}
	path.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, &path, &vector.FillOptions{}, op)
}

// rotPt rotates (x, y) around (cx, cy).
func rotPt(cx, cy, x, y, angle float64) (float64, float64) {
	s, c := math.Sincos(angle)
	dx, dy := x-cx, y-cy
	return cx + dx*c - dy*s, cy + dx*s + dy*c
}

func (g *Game) drawCampaignEnemies(screen *ebiten.Image) {
	for _, e := range g.campaignEnemies {
		if e == nil {
			continue
		}
		if e.Dead {
			if campaignEnemyKind(e.Kind).bigEnemy() {
				g.drawBigWreck(screen, e)
			} else {
				g.drawCampaignEnemyWreck(screen, e)
			}
			continue
		}
		switch campaignEnemyKind(e.Kind) {
		case campaignEnemyDrone:
			g.drawDrone(screen, e)
		case campaignEnemyShieldBearer:
			g.drawShieldBearer(screen, e)
		case campaignEnemyMech:
			g.drawMech(screen, e)
		case campaignEnemyHelicopter:
			g.drawHeli(screen, e)
		}
	}
}

func (g *Game) drawDrone(screen *ebiten.Image, e *CampaignEnemy) {
	b := &e.Body
	wx, wy := droneCenter(b)
	cx, cy := g.worldX(wx), g.worldY(wy)
	tilt := e.Tilt
	if e.Stall > 0 {
		tilt += math.Sin(float64(e.Frame)*0.9) * 0.25
	}
	f := float64(b.Facing)
	if f == 0 {
		f = 1
	}
	hull := color.NRGBA{R: 0x3a, G: 0x40, B: 0x48, A: 0xff}
	edge := color.NRGBA{R: 0x12, G: 0x14, B: 0x18, A: 0xff}
	if e.HitFlash > 0 && e.HitFlash%2 == 0 {
		hull = color.NRGBA{R: 0xd8, G: 0xd8, B: 0xd8, A: 0xff}
	}
	// Arms to the two visible rotor hubs.
	for _, side := range []float64{-1, 1} {
		ax, ay := rotPt(cx, cy, cx+side*17, cy-6, tilt)
		vector.StrokeLine(screen, float32(cx), float32(cy-1), float32(ax), float32(ay), 3, edge, true)
		// Rotor: a flat, flickering blur ellipse unless stalled.
		blur := uint8(0x70 + (e.Frame*37+int(side*11))%0x50)
		rw := 11.0
		if e.Stall > 0 {
			blur = 0xe0
			rw = 9
		}
		pts := make([][2]float64, 0, 12)
		for i := 0; i < 12; i++ {
			a := float64(i) / 12 * 2 * math.Pi
			px, py := rotPt(cx, cy, ax+math.Cos(a)*rw, ay-3+math.Sin(a)*2.2, tilt)
			pts = append(pts, [2]float64{px, py})
		}
		fillPolygon(screen, pts, color.NRGBA{R: 0xb8, G: 0xc4, B: 0xd0, A: blur})
		if e.Stall > 0 {
			// Stalled: visible blades instead of the blur.
			bx0, by0 := rotPt(cx, cy, ax-rw, ay-3, tilt+float64(e.Frame%7)*0.05)
			bx1, by1 := rotPt(cx, cy, ax+rw, ay-3, tilt+float64(e.Frame%7)*0.05)
			vector.StrokeLine(screen, float32(bx0), float32(by0), float32(bx1), float32(by1), 1.5, edge, true)
		}
		vector.DrawFilledCircle(screen, float32(ax), float32(ay), 2.2, edge, true)
	}
	// Hull.
	hullPts := [][2]float64{{cx - 12, cy - 4}, {cx + 12, cy - 4}, {cx + 9, cy + 5}, {cx - 9, cy + 5}}
	for i := range hullPts {
		hullPts[i][0], hullPts[i][1] = rotPt(cx, cy, hullPts[i][0], hullPts[i][1], tilt)
	}
	fillPolygon(screen, hullPts, hull)
	for i := range hullPts {
		j := (i + 1) % len(hullPts)
		vector.StrokeLine(screen, float32(hullPts[i][0]), float32(hullPts[i][1]), float32(hullPts[j][0]), float32(hullPts[j][1]), 1.2, edge, true)
	}
	// Gun pod under the nose, pointing where it shoots.
	gx0, gy0 := rotPt(cx, cy, cx+f*4, cy+6, tilt)
	gx1, gy1 := rotPt(cx, cy, cx+f*15, cy+6, tilt)
	vector.StrokeLine(screen, float32(gx0), float32(gy0), float32(gx1), float32(gy1), 2.6, edge, true)
	// Camera eye: red while it has a target lock, amber when blinded.
	ex, ey := rotPt(cx, cy, cx+f*7, cy, tilt)
	eye := color.NRGBA{R: 0xff, G: 0x30, B: 0x20, A: 0xff}
	if b.BlindTime > 0 {
		eye = color.NRGBA{R: 0xff, G: 0xc0, B: 0x30, A: 0xff}
	}
	vector.DrawFilledCircle(screen, float32(ex), float32(ey), 2.4, eye, true)
	// Blinking beacon on top.
	if (e.Frame/12)%2 == 0 {
		lx, ly := rotPt(cx, cy, cx-f*5, cy-6, tilt)
		vector.DrawFilledCircle(screen, float32(lx), float32(ly), 2.6, color.NRGBA{R: 0x40, G: 0xff, B: 0x60, A: 0xff}, true)
		vector.DrawFilledCircle(screen, float32(lx), float32(ly), 5, color.NRGBA{R: 0x40, G: 0xff, B: 0x60, A: 0x40}, true)
	}
	drawEnemyHPPips(screen, e, cx, cy-22)
}

func (g *Game) drawShieldBearer(screen *ebiten.Image, e *CampaignEnemy) {
	b := e.Body
	drawP := b
	drawP.X = g.worldX(b.X)
	drawP.Y = g.worldY(b.Y)
	if g.assets != nil {
		g.drawPlayer(screen, &drawP)
	}
	f := float64(b.Facing)
	if f == 0 {
		f = 1
	}
	x, y := drawP.X, drawP.Y
	// Procedural riot helmet: a dark dome with a tinted visor.
	hx, hy := x+f*2, y-51+(b.VisualEyesY+46)*0.8
	dome := make([][2]float64, 0, 14)
	for i := 0; i <= 12; i++ {
		a := math.Pi + float64(i)/12*math.Pi
		dome = append(dome, [2]float64{hx + math.Cos(a)*17, hy + math.Sin(a)*13})
	}
	fillPolygon(screen, dome, color.NRGBA{R: 0x22, G: 0x2a, B: 0x36, A: 0xff})
	vector.StrokeLine(screen, float32(hx-17), float32(hy), float32(hx+17), float32(hy), 2.5, color.NRGBA{R: 0x10, G: 0x12, B: 0x16, A: 0xff}, true)
	fillPolygon(screen, [][2]float64{{hx - f*2, hy}, {hx + f*17, hy}, {hx + f*15, hy + 8}, {hx - f*2, hy + 8}}, color.NRGBA{R: 0x70, G: 0xa0, B: 0xc8, A: 0x80})

	plate := color.NRGBA{R: 0x4c, G: 0x58, B: 0x66, A: 0xf0}
	rim := color.NRGBA{R: 0x1a, G: 0x1e, B: 0x24, A: 0xff}
	shine := color.NRGBA{R: 0xb0, G: 0xc8, B: 0xe0, A: 0x90}
	var pts [][2]float64
	if e.ShieldDown > 0 {
		// Lowered: the plate hangs at the hip, angled away.
		pts = [][2]float64{
			{x + f*8, y - 30}, {x + f*18, y - 34}, {x + f*30, y + 2}, {x + f*20, y + 4},
		}
	} else {
		bash := 0.0
		if e.StateTimer > 0 {
			bash = 6 * math.Sin(float64(12-e.StateTimer)/12*math.Pi)
		}
		x0, x1 := x+f*(shieldLocalX0+bash), x+f*(shieldLocalX1+bash)
		pts = [][2]float64{
			{x0, y + shieldLocalY0}, {x1, y + shieldLocalY0 + 3}, {x1, y + shieldLocalY1 - 3}, {x0, y + shieldLocalY1},
		}
	}
	fillPolygon(screen, pts, plate)
	for i := range pts {
		j := (i + 1) % len(pts)
		vector.StrokeLine(screen, float32(pts[i][0]), float32(pts[i][1]), float32(pts[j][0]), float32(pts[j][1]), 1.6, rim, true)
	}
	if e.ShieldDown <= 0 {
		// Visor slit near the top, and a glint stripe.
		sx0, sx1 := pts[0][0], pts[1][0]
		vector.StrokeLine(screen, float32(sx0+f*1.5), float32(y+shieldLocalY0+14), float32(sx1-f*1.5), float32(y+shieldLocalY0+15), 2.2, color.NRGBA{R: 0x08, G: 0x0a, B: 0x0c, A: 0xff}, true)
		mid := (sx0 + sx1) / 2
		vector.StrokeLine(screen, float32(mid), float32(y+shieldLocalY0+22), float32(mid), float32(y+shieldLocalY1-8), 1.2, shine, true)
		if e.ShieldSpark > 0 {
			sparkY := y + shieldLocalY0 + 30 + float64((e.Frame*13)%20) - 10
			fx := x + f*(shieldLocalX1+2)
			k := float64(e.ShieldSpark) / 8
			for i := -1; i <= 1; i++ {
				vector.StrokeLine(screen, float32(fx), float32(sparkY), float32(fx+f*10*k), float32(sparkY+float64(i)*7*k), 1.6,
					color.NRGBA{R: 0xff, G: 0xe0, B: 0x60, A: uint8(255 * k)}, true)
			}
			vector.DrawFilledCircle(screen, float32(fx), float32(sparkY), float32(3*k+1), color.NRGBA{R: 0xff, G: 0xff, B: 0xc0, A: uint8(220 * k)}, true)
		}
	}
	drawEnemyHPPips(screen, e, x, y-78)
}

// drawEnemyHPPips shows remaining HP for enemies that have an HP pool.
func drawEnemyHPPips(screen *ebiten.Image, e *CampaignEnemy, cx, y float64) {
	if e.MaxHP <= 0 || e.HP >= e.MaxHP {
		return
	}
	const w = 26.0
	frac := float64(e.HP) / float64(e.MaxHP)
	vector.DrawFilledRect(screen, float32(cx-w/2-1), float32(y-1), w+2, 5, color.NRGBA{A: 0xb0}, false)
	vector.DrawFilledRect(screen, float32(cx-w/2), float32(y), float32(w*frac), 3, color.NRGBA{R: 0xff, G: 0x50, B: 0x30, A: 0xff}, false)
}

func (g *Game) drawCampaignEnemyWreck(screen *ebiten.Image, e *CampaignEnemy) {
	if campaignEnemyKind(e.Kind) != campaignEnemyDrone {
		return
	}
	k := 1 - float64(e.DeadFrames)/campaignEnemyDeathFXFrames
	if k <= 0 {
		return
	}
	x := g.worldX(e.Body.LastDeathX)
	y := g.worldY(e.Body.LastDeathY-16) + float64(e.DeadFrames)*float64(e.DeadFrames)*0.12
	for i := 0; i < 4; i++ {
		a := float64(i)*1.7 + float64(e.DeadFrames)*0.3
		dx := math.Cos(a) * float64(e.DeadFrames) * 1.4
		vector.StrokeLine(screen, float32(x+dx), float32(y), float32(x+dx+math.Cos(a)*6), float32(y+math.Sin(a)*6), 2,
			color.NRGBA{R: 0x30, G: 0x34, B: 0x3a, A: uint8(255 * k)}, true)
	}
}

// drawHazardMarkers: a pulsing target ring where the shell will land and a
// dashed drop line, red once the payload is falling.
func (g *Game) drawHazardMarkers(screen *ebiten.Image) {
	for _, m := range g.extraHazardMarkers {
		if m.Dead || m.Life <= 0 {
			continue
		}
		x, y := g.worldX(m.X), g.worldY(m.Y)
		dropping := m.Life <= m.Drop
		age := m.MaxLife - m.Life
		pulse := 0.5 + 0.5*math.Sin(float64(age)*0.35)
		clr := color.NRGBA{R: 0xff, G: 0xd0, B: 0x30, A: 0xff}
		if dropping {
			clr = color.NRGBA{R: 0xff, G: 0x38, B: 0x20, A: 0xff}
		}
		alpha := math.Min(1, float64(age)/8)
		if m.Life < 10 {
			alpha *= float64(m.Life) / 10
		}
		for yy := 0.0; yy < y-14; yy += 16 {
			vector.StrokeLine(screen, float32(x), float32(yy), float32(x), float32(math.Min(yy+8, y-14)), 1.5, withAlpha(clr, alpha*0.55), true)
		}
		r := 16 + 6*pulse
		var ring vector.Path
		ring.Arc(float32(x), float32(y-3), float32(r), 0, 2*math.Pi, vector.Clockwise)
		strokePathColor(screen, &ring, withAlpha(clr, alpha), 2.5)
		fillPolygon(screen, [][2]float64{{x - r, y - 3}, {x + r, y - 3}, {x + r*0.7, y}, {x - r*0.7, y}}, withAlpha(clr, alpha*0.35))
		// Chevron pointing down at the impact point.
		vector.StrokeLine(screen, float32(x-7), float32(y-30-4*pulse), float32(x), float32(y-22-4*pulse), 2.5, withAlpha(clr, alpha), true)
		vector.StrokeLine(screen, float32(x+7), float32(y-30-4*pulse), float32(x), float32(y-22-4*pulse), 2.5, withAlpha(clr, alpha), true)
	}
}

// drawCampaignEnemyArrows reuses the off-screen player arrow for enemies,
// tinted with the enemy-side colour.
func (g *Game) drawCampaignEnemyArrows(screen *ebiten.Image) {
	if g.assets == nil {
		return
	}
	for _, e := range g.campaignEnemies {
		if !e.alive() {
			continue
		}
		visible, angle, distance := g.sourcePlayerArrowState(&e.Body)
		if !visible {
			continue
		}
		colorIndex, _ := resolvedPlayerColor(e.Body.PlayerColor, false)
		g.assets.EnsurePlayerColor(colorIndex)
		arrow := g.assets.PlayerArrowFrames[colorIndex]
		if arrow == nil {
			arrow = g.assets.PlayerArrowFrames[0]
		}
		drawPlayerArrowRaster(screen, arrow, angle)
		drawPlayerArrowDistance(screen, distance, angle)
	}
}

// campaignEnemyRopePoint is where a harpoon hook sticks in an enemy (world).
func campaignEnemyRopePoint(e *CampaignEnemy) (float64, float64) {
	b := &e.Body
	switch campaignEnemyKind(e.Kind) {
	case campaignEnemyDrone:
		return droneCenter(b)
	case campaignEnemyHelicopter:
		return b.X, b.Y - 8 // the belly
	case campaignEnemyMech:
		return b.X, b.Y - 90
	}
	return b.X, b.Y - 36*playerRenderScale(b)/0.8
}
