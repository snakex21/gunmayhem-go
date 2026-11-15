package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Procedural art of the Stage B enemies (mech walker, gunship) in the style
// of the drone and the shield-bearer: flat steel shapes with dark rims,
// everything drawn from synced CampaignEnemy state (no sprite files).

var (
	bigSteel     = color.NRGBA{R: 0x3a, G: 0x40, B: 0x48, A: 0xff}
	bigSteelHi   = color.NRGBA{R: 0x56, G: 0x5e, B: 0x68, A: 0xff}
	bigSteelDark = color.NRGBA{R: 0x26, G: 0x2a, B: 0x30, A: 0xff}
	bigRim       = color.NRGBA{R: 0x12, G: 0x14, B: 0x18, A: 0xff}
	bigOlive     = color.NRGBA{R: 0x3e, G: 0x47, B: 0x38, A: 0xff}
	bigOliveDark = color.NRGBA{R: 0x2a, G: 0x30, B: 0x26, A: 0xff}
	bigFlash     = color.NRGBA{R: 0xe0, G: 0xe0, B: 0xe0, A: 0xff}
	bigLaser     = color.NRGBA{R: 0xff, G: 0x30, B: 0x20, A: 0xff}
	bigWarn      = color.NRGBA{R: 0xff, G: 0x90, B: 0x20, A: 0xff}
)

// localPoly maps facing-local points (x forward) around (cx, cy), rotated.
func localPoly(cx, cy, f, rot float64, pts [][2]float64) [][2]float64 {
	out := make([][2]float64, len(pts))
	for i, p := range pts {
		x, y := rotPt(cx, cy, cx+p[0]*f, cy+p[1], rot)
		out[i] = [2]float64{x, y}
	}
	return out
}

func strokeClosed(dst *ebiten.Image, pts [][2]float64, w float32, clr color.NRGBA) {
	for i := range pts {
		j := (i + 1) % len(pts)
		vector.StrokeLine(dst, float32(pts[i][0]), float32(pts[i][1]), float32(pts[j][0]), float32(pts[j][1]), w, clr, true)
	}
}

func flashColor(e *CampaignEnemy, base color.NRGBA) color.NRGBA {
	if e.HitFlash > 0 && e.HitFlash%2 == 0 {
		return bigFlash
	}
	return base
}

// --- mech ---------------------------------------------------------------------

func (g *Game) drawMech(screen *ebiten.Image, e *CampaignEnemy) {
	b := &e.Body
	x, y := g.worldX(b.X), g.worldY(b.Y)
	if x < -200 || x > ScreenWidth+200 || y < -300 || y > ScreenHeight+300 {
		return
	}
	f := float64(b.Facing)
	if f == 0 {
		f = 1
	}
	g.drawMechTelegraph(screen, e)
	melt := e.State == mechStateMeltdown
	if melt {
		x += math.Sin(float64(e.Frame)*2.3) * 2
		y += math.Cos(float64(e.Frame)*1.9) * 1
	}
	// Body-local points (x forward) scaled to the hitbox size.
	const s = mechDrawScale
	P := func(lx, ly float64) (float64, float64) { return x + f*lx*s, y + ly*s }
	poly := func(pts ...[2]float64) [][2]float64 {
		out := make([][2]float64, len(pts))
		for i, p := range pts {
			out[i][0], out[i][1] = P(p[0], p[1])
		}
		return out
	}
	airborne := b.JumpNum != 2
	moving := !airborne && math.Abs(b.VX) > 0.25
	phase := b.X * 0.05
	hull := flashColor(e, bigSteel)
	if melt {
		hull = color.NRGBA{R: 0x5a, G: 0x34, B: 0x2c, A: 0xff}
	}
	// Legs: far leg first (darker), then near leg. Chicken-walker: the knee
	// points back.
	for i, side := range []float64{-1, 1} {
		swing, lift := 0.0, 0.0
		if moving {
			swing = math.Sin(phase+side*math.Pi/2) * 8
			lift = math.Max(0, math.Cos(phase+side*math.Pi/2)) * 5
		}
		if e.State == mechStateStompWind && side > 0 {
			k := math.Min(1, float64(e.StateTimer)/mechStompWind)
			lift = 22 * k
			swing = 5 * k
		}
		if airborne {
			lift, swing = 6, -3*side
		}
		clr := bigSteelDark
		if i == 1 {
			clr = flashColor(e, bigSteelHi)
		}
		hipX, hipY := P(-3+side*4, -58)
		kneeX, kneeY := P(-15+swing*0.5+side*3, -34-lift*0.5)
		ankX, ankY := P(3+swing+side*2, -9-lift)
		w := float32(s)
		vector.StrokeLine(screen, float32(hipX), float32(hipY), float32(kneeX), float32(kneeY), 11*w, bigRim, true)
		vector.StrokeLine(screen, float32(hipX), float32(hipY), float32(kneeX), float32(kneeY), 8*w, clr, true)
		vector.StrokeLine(screen, float32(kneeX), float32(kneeY), float32(ankX), float32(ankY), 9*w, bigRim, true)
		vector.StrokeLine(screen, float32(kneeX), float32(kneeY), float32(ankX), float32(ankY), 6*w, clr, true)
		vector.DrawFilledCircle(screen, float32(kneeX), float32(kneeY), 6*w, bigRim, true)
		vector.DrawFilledCircle(screen, float32(kneeX), float32(kneeY), 3.5*w, clr, true)
		// Foot: a wide clawed plate from the ankle down to the ground.
		ax, ay := ankX, ankY
		foot := [][2]float64{{ax - f*9*s, ay - 2*s}, {ax + f*4*s, ay - 3*s}, {ax + f*17*s, ay + 7*s}, {ax + f*17*s, ay + 9*s}, {ax - f*12*s, ay + 9*s}}
		fillPolygon(screen, foot, clr)
		strokeClosed(screen, foot, 1.3, bigRim)
	}
	// Jump jets while rising.
	if airborne && b.VY < 0 {
		for _, sd := range []float64{-1, 1} {
			jx, jy := P(-24+sd*4, -50)
			drawPuff(screen, jx, jy, 10, 1, 0.6, 0.2, 0.5)
		}
	}
	// Torso.
	torso := poly([2]float64{-32, -96}, [2]float64{20, -104}, [2]float64{34, -86}, [2]float64{30, -60}, [2]float64{-24, -56}, [2]float64{-36, -72})
	fillPolygon(screen, torso, hull)
	strokeClosed(screen, torso, 2, bigRim)
	l0x, l0y := P(-28, -80)
	l1x, l1y := P(28, -82)
	vector.StrokeLine(screen, float32(l0x), float32(l0y), float32(l1x), float32(l1y), 1.3, bigSteelDark, true)
	for _, rv := range [][2]float64{{-20, -66}, {-6, -67}, {8, -68}, {20, -69}} {
		rx, ry := P(rv[0], rv[1])
		vector.DrawFilledCircle(screen, float32(rx), float32(ry), 1.6, bigRim, true)
	}
	// Cockpit glass (red eye while attacking).
	glass := poly([2]float64{14, -99}, [2]float64{30, -87}, [2]float64{27, -78}, [2]float64{12, -84})
	fillPolygon(screen, glass, color.NRGBA{R: 0xd8, G: 0x9a, B: 0x30, A: 0xd0})
	strokeClosed(screen, glass, 1.2, bigRim)
	if e.State == mechStateAim || e.State == mechStateFire || e.State == mechStateRocketWind {
		ex, ey := P(22, -88)
		vector.DrawFilledCircle(screen, float32(ex), float32(ey), 3, bigLaser, true)
	}
	// Exposed reactor on the back: the weak spot.
	pulse := 0.6 + 0.4*math.Sin(float64(e.Frame)*0.25)
	rx0, ry0 := P(-37, -92)
	rx1, ry1 := P(-27, -66)
	if rx0 > rx1 {
		rx0, rx1 = rx1, rx0
	}
	vector.DrawFilledRect(screen, float32(rx0), float32(ry0), float32(rx1-rx0), float32(ry1-ry0), color.NRGBA{R: 0x30, G: 0x10, B: 0x08, A: 0xff}, true)
	vector.DrawFilledRect(screen, float32(rx0+2), float32(ry0+3), float32(rx1-rx0-4), float32(ry1-ry0-6), withAlpha(color.NRGBA{R: 0xff, G: 0x88, B: 0x20, A: 0xff}, pulse), true)
	for i := 1; i < 5; i++ {
		yy := float32(ry0 + (ry1-ry0)*float64(i)/5)
		vector.StrokeLine(screen, float32(rx0), yy, float32(rx1), yy, 1.3, bigRim, true)
	}
	vector.StrokeRect(screen, float32(rx0), float32(ry0), float32(rx1-rx0), float32(ry1-ry0), 1.5, bigRim, true)
	if e.ShieldSpark > 0 {
		k := float64(e.ShieldSpark) / 8
		sx, sy := P(-38, -80)
		for i := -1; i <= 1; i++ {
			vector.StrokeLine(screen, float32(sx), float32(sy), float32(sx-f*14*k), float32(sy+float64(i)*9*k), 2, color.NRGBA{R: 0xff, G: 0xe0, B: 0x60, A: uint8(255 * k)}, true)
		}
		vector.DrawFilledCircle(screen, float32(sx), float32(sy), float32(5*k+1), color.NRGBA{R: 0xff, G: 0xff, B: 0xc0, A: uint8(220 * k)}, true)
	}
	// Shoulder mortar pod.
	if e.Mortar {
		pod := poly([2]float64{-26, -118}, [2]float64{-2, -118}, [2]float64{-2, -101}, [2]float64{-26, -99})
		fillPolygon(screen, pod, bigSteelDark)
		strokeClosed(screen, pod, 1.4, bigRim)
		glow := color.NRGBA{R: 0x18, G: 0x18, B: 0x1c, A: 0xff}
		if e.State == mechStateRocketWind && (e.StateTimer/3)%2 == 0 {
			glow = bigWarn
		}
		for _, ox := range []float64{-19, -9} {
			tx, ty := P(ox, -110)
			vector.DrawFilledCircle(screen, float32(tx), float32(ty), float32(3.6*s), glow, true)
		}
	}
	// Twin machine-gun pod at a player's chest height (where rounds leave).
	gy := y - mechMuzzleY
	ax0, _ := P(6, 0)
	ax1, _ := P(30, 0)
	arm := [][2]float64{{ax0, gy - 11}, {ax1, gy - 9}, {ax1, gy + 9}, {ax0, gy + 7}}
	sx, sy := P(10, -62)
	vector.StrokeLine(screen, float32(sx), float32(sy), float32(ax0+f*6), float32(gy), 6, bigRim, true)
	fillPolygon(screen, arm, flashColor(e, bigSteelHi))
	strokeClosed(screen, arm, 1.4, bigRim)
	for _, sd := range []float64{-1, 1} {
		by := gy + sd*mechBarrelSeparateY
		vector.StrokeLine(screen, float32(ax1-f*2), float32(by), float32(x+f*(mechMuzzleX*s+6)), float32(by), 3.2, bigRim, true)
	}
	if melt {
		for i := 0; i < 3; i++ {
			a := float64(e.Frame+i*9) * 0.08
			px, py := P(-10, -110)
			drawPuff(screen, px+math.Sin(a)*16, py-float64((e.Frame*2+i*14)%40), 13, 0.25, 0.25, 0.27, 0.45)
		}
	}
	if !e.Elite {
		_, hy := P(0, -128)
		drawEnemyHPPips(screen, e, x, hy)
	}
}

func (g *Game) drawMechTelegraph(screen *ebiten.Image, e *CampaignEnemy) {
	switch e.State {
	case mechStateAim:
		k := 1 - float64(e.StateTimer)/mechAimFrames
		x0, x1, y := g.worldX(e.AimX0), g.worldX(e.AimX1), g.worldY(e.AimY)
		a := 0.35 + 0.65*k
		if (e.StateTimer/2)%2 == 0 {
			a *= 0.7
		}
		vector.StrokeLine(screen, float32(x0), float32(y), float32(x1), float32(y), float32(1+2*k), withAlpha(bigLaser, a), true)
		vector.DrawFilledCircle(screen, float32(x0), float32(y), float32(3+3*k), withAlpha(bigLaser, a), true)
	case mechStateStompWind:
		cx, cy := g.worldX(e.AimX0), g.worldY(e.AimY)
		r := e.AimX1
		k := math.Min(1, float64(e.StateTimer)/mechStompWind)
		pts := make([][2]float64, 0, 24)
		for i := 0; i < 24; i++ {
			a := float64(i) / 24 * 2 * math.Pi
			pts = append(pts, [2]float64{cx + math.Cos(a)*r, cy - 2 + math.Sin(a)*9})
		}
		fillPolygon(screen, pts, withAlpha(bigWarn, 0.12+0.18*k))
		strokeClosed(screen, pts, 2.2, withAlpha(bigWarn, 0.5+0.5*k))
		// The full reach, faint.
		full := make([][2]float64, 0, 24)
		for i := 0; i < 24; i++ {
			a := float64(i) / 24 * 2 * math.Pi
			full = append(full, [2]float64{cx + math.Cos(a)*mechStompRadius, cy - 2 + math.Sin(a)*10})
		}
		strokeClosed(screen, full, 1, withAlpha(bigWarn, 0.35))
	}
}

// --- gunship --------------------------------------------------------------------

func (g *Game) drawHeli(screen *ebiten.Image, e *CampaignEnemy) {
	b := &e.Body
	wx, wy := heliCenter(b)
	cx, cy := g.worldX(wx), g.worldY(wy)
	if cx < -300 || cx > ScreenWidth+300 || cy < -300 || cy > ScreenHeight+300 {
		return
	}
	f := float64(b.Facing)
	if f == 0 {
		f = 1
	}
	rot := e.Tilt
	crash := e.State == heliStateCrash
	g.drawHeliStrafeLine(screen, e)
	// Searchlight cone.
	if !crash {
		nx, ny := rotPt(cx, cy, cx+f*56, cy+8, rot)
		gx, gy := nx+f*80, ny+240
		if e.State == heliStateStrafeLine || e.State == heliStateStrafe {
			gx, gy = g.worldX(e.AimX0+(e.AimX1-e.AimX0)*0.5), g.worldY(e.AimY)
		}
		dx, dy := gx-nx, gy-ny
		d := math.Hypot(dx, dy)
		if d > 1 {
			px, py := -dy/d*46, dx/d*46
			fillPolygon(screen, [][2]float64{{nx, ny}, {gx + px, gy + py}, {gx - px, gy - py}}, color.NRGBA{R: 0xff, G: 0xf4, B: 0xc8, A: 0x30})
			spot := make([][2]float64, 0, 16)
			for i := 0; i < 16; i++ {
				a := float64(i) / 16 * 2 * math.Pi
				spot = append(spot, [2]float64{gx + math.Cos(a)*48, gy + math.Sin(a)*9})
			}
			fillPolygon(screen, spot, color.NRGBA{R: 0xff, G: 0xf8, B: 0xd8, A: 0x40})
		}
	}
	hull := flashColor(e, bigOlive)
	if crash {
		hull = color.NRGBA{R: 0x40, G: 0x34, B: 0x2c, A: 0xff}
	}
	// Tail boom, fin and tail rotor.
	boom := localPoly(cx, cy, f, rot, [][2]float64{{-36, -10}, {-112, -16}, {-112, -9}, {-36, 4}})
	fillPolygon(screen, boom, hull)
	strokeClosed(screen, boom, 1.4, bigRim)
	fin := localPoly(cx, cy, f, rot, [][2]float64{{-104, -14}, {-122, -34}, {-114, -34}, {-98, -12}})
	fillPolygon(screen, fin, bigOliveDark)
	strokeClosed(screen, fin, 1.2, bigRim)
	tx, ty := rotPt(cx, cy, cx-f*116, cy-18, rot)
	tr := 13.0
	tblur := uint8(0x60 + (e.Frame*41)%0x50)
	if crash {
		tblur = 0xc0
		tr = 10
	}
	vector.DrawFilledCircle(screen, float32(tx), float32(ty), float32(tr), color.NRGBA{R: 0xb8, G: 0xc4, B: 0xd0, A: tblur}, true)
	vector.DrawFilledCircle(screen, float32(tx), float32(ty), 2.5, bigRim, true)
	// Skids.
	for _, sx := range []float64{-22, 22} {
		a0x, a0y := rotPt(cx, cy, cx+f*sx, cy+16, rot)
		a1x, a1y := rotPt(cx, cy, cx+f*(sx-4), cy+30, rot)
		vector.StrokeLine(screen, float32(a0x), float32(a0y), float32(a1x), float32(a1y), 2.4, bigRim, true)
	}
	s0x, s0y := rotPt(cx, cy, cx-f*44, cy+30, rot)
	s1x, s1y := rotPt(cx, cy, cx+f*36, cy+30, rot)
	vector.StrokeLine(screen, float32(s0x), float32(s0y), float32(s1x), float32(s1y), 3, bigRim, true)
	// Fuselage.
	body := localPoly(cx, cy, f, rot, [][2]float64{{-50, -18}, {18, -24}, {46, -14}, {62, 2}, {52, 18}, {-38, 20}, {-56, 4}})
	fillPolygon(screen, body, hull)
	strokeClosed(screen, body, 2, bigRim)
	stripe := localPoly(cx, cy, f, rot, [][2]float64{{-50, 6}, {48, 6}, {50, 10}, {-50, 11}})
	fillPolygon(screen, stripe, bigOliveDark)
	// Side door (open while dropping troops).
	door := localPoly(cx, cy, f, rot, [][2]float64{{-24, -12}, {-4, -13}, {-4, 14}, {-24, 15}})
	if e.State == heliStateDrop {
		fillPolygon(screen, door, color.NRGBA{R: 0x0c, G: 0x0c, B: 0x0e, A: 0xff})
	} else {
		strokeClosed(screen, door, 1.1, bigRim)
	}
	// Cockpit glass.
	glass := localPoly(cx, cy, f, rot, [][2]float64{{20, -20}, {44, -11}, {56, 2}, {30, 2}, {18, -6}})
	fillPolygon(screen, glass, color.NRGBA{R: 0x70, G: 0xa0, B: 0xc8, A: 0xc0})
	strokeClosed(screen, glass, 1.2, bigRim)
	// Rocket pod under the stub wing; glows before a salvo.
	pod := localPoly(cx, cy, f, rot, [][2]float64{{-14, 18}, {20, 18}, {24, 24}, {-14, 26}})
	podClr := bigSteelDark
	if e.State == heliStateSalvoWind && (e.StateTimer/3)%2 == 0 {
		podClr = bigWarn
	}
	fillPolygon(screen, pod, podClr)
	strokeClosed(screen, pod, 1.2, bigRim)
	// Nose gun.
	g0x, g0y := rotPt(cx, cy, cx+f*54, cy+14, rot)
	g1x, g1y := rotPt(cx, cy, cx+f*70, cy+22, rot)
	vector.StrokeLine(screen, float32(g0x), float32(g0y), float32(g1x), float32(g1y), 3.4, bigRim, true)
	vector.DrawFilledCircle(screen, float32(g0x), float32(g0y), 4, bigSteelDark, true)
	// Main rotor: mast, a flat blur disc and two foreshortened blades.
	m0x, m0y := rotPt(cx, cy, cx, cy-22, rot)
	m1x, m1y := rotPt(cx, cy, cx, cy-31, rot)
	vector.StrokeLine(screen, float32(m0x), float32(m0y), float32(m1x), float32(m1y), 4, bigRim, true)
	spin := float64(e.Frame) * 0.9
	if crash {
		spin = float64(e.Frame) * 0.35
	}
	disc := make([][2]float64, 0, 20)
	for i := 0; i < 20; i++ {
		a := float64(i) / 20 * 2 * math.Pi
		px, py := rotPt(cx, cy, m1x+math.Cos(a)*104, m1y+math.Sin(a)*4.5, rot)
		disc = append(disc, [2]float64{px, py})
	}
	blur := uint8(0x38 + (e.Frame*29)%0x30)
	fillPolygon(screen, disc, color.NRGBA{R: 0xb8, G: 0xc4, B: 0xd0, A: blur})
	for k := 0; k < 2; k++ {
		l := 104 * math.Cos(spin+float64(k)*math.Pi/2)
		b0x, b0y := rotPt(cx, cy, m1x-l, m1y, rot)
		b1x, b1y := rotPt(cx, cy, m1x+l, m1y, rot)
		vector.StrokeLine(screen, float32(b0x), float32(b0y), float32(b1x), float32(b1y), 2.2, withAlpha(bigRim, 0.75), true)
	}
	vector.DrawFilledCircle(screen, float32(m1x), float32(m1y), 3.5, bigSteelDark, true)
	// Blinking beacon.
	if (e.Frame/10)%2 == 0 {
		lx, ly := rotPt(cx, cy, cx-f*112, cy-16, rot)
		vector.DrawFilledCircle(screen, float32(lx), float32(ly), 2.6, color.NRGBA{R: 0xff, G: 0x30, B: 0x20, A: 0xff}, true)
		vector.DrawFilledCircle(screen, float32(lx), float32(ly), 6, color.NRGBA{R: 0xff, G: 0x30, B: 0x20, A: 0x40}, true)
	}
	if e.Hooked > 0 {
		// Straining against the rope: rotor wash puffs under the skids.
		for i := 0; i < 2; i++ {
			drawPuff(screen, cx+float64((e.Frame*7+i*40)%80-40), cy+44, 10, 0.75, 0.75, 0.72, 0.25)
		}
	}
	if crash || e.HP*4 < e.MaxHP {
		// Damaged engine: smoke (and fire while going down).
		n := 2
		if crash {
			n = 4
		}
		for i := 0; i < n; i++ {
			a := float64(e.Frame*3+i*17) * 0.05
			sx, sy := rotPt(cx, cy, cx-f*20, cy-26, rot)
			drawPuff(screen, sx-f*float64(i)*16+math.Sin(a)*6, sy-float64(i)*10, 12+float64(i)*4, 0.16, 0.16, 0.18, 0.7)
		}
		if crash {
			fx, fy := rotPt(cx, cy, cx-f*14, cy-20, rot)
			drawPuff(screen, fx, fy, 12, 1, 0.55, 0.15, 0.7)
		}
	}
}

// drawHeliStrafeLine: the red line along the platform that the next (or
// current) strafing run covers, with chevrons in the run direction.
func (g *Game) drawHeliStrafeLine(screen *ebiten.Image, e *CampaignEnemy) {
	if e.State != heliStateStrafeLine && e.State != heliStateStrafe {
		return
	}
	f := float64(e.Body.Facing)
	if f == 0 {
		f = 1
	}
	x0, x1 := e.AimX0, e.AimX1
	if e.State == heliStateStrafe {
		// Only what is still ahead of the gunship.
		head := e.Body.X + f*60
		if (head-x0)*f > 0 {
			x0 = head
		}
		if (x1-x0)*f <= 0 {
			return
		}
	}
	y := g.worldY(e.AimY) - 3
	pulse := 0.55 + 0.45*math.Sin(float64(e.Frame)*0.5)
	a := 0.6 + 0.4*pulse
	sx0, sx1 := g.worldX(x0), g.worldX(x1)
	lo, hi := math.Min(sx0, sx1), math.Max(sx0, sx1)
	// A danger band on the walkway, then bold dashes and moving chevrons.
	vector.DrawFilledRect(screen, float32(lo), float32(y-16), float32(hi-lo), 16, withAlpha(bigLaser, 0.18*a), true)
	step := 18.0
	n := int(math.Abs(sx1-sx0) / step)
	for i := 0; i < n; i += 2 {
		p0 := sx0 + f*float64(i)*step
		p1 := sx0 + f*float64(i+1)*step
		vector.StrokeLine(screen, float32(p0), float32(y), float32(p1), float32(y), 6, withAlpha(bigRim, 0.6*a), true)
		vector.StrokeLine(screen, float32(p0), float32(y), float32(p1), float32(y), 4, withAlpha(bigLaser, a), true)
	}
	for i := 1; i < n; i += 4 {
		px := sx0 + f*float64(i)*step + f*float64(e.Frame%8)
		vector.StrokeLine(screen, float32(px-f*9), float32(y-24), float32(px), float32(y-15), 3, withAlpha(bigLaser, a), true)
		vector.StrokeLine(screen, float32(px-f*9), float32(y-6), float32(px), float32(y-15), 3, withAlpha(bigLaser, a), true)
	}
}

// --- wrecks -------------------------------------------------------------------------

// drawBigWreck: plates and rotor pieces thrown out of the explosion, falling
// and fading over campaignBigWreckFrames.
func (g *Game) drawBigWreck(screen *ebiten.Image, e *CampaignEnemy) {
	k := 1 - float64(e.DeadFrames)/campaignBigWreckFrames
	if k <= 0 {
		return
	}
	t := float64(e.DeadFrames)
	x := g.worldX(e.Body.LastDeathX)
	y := g.worldY(e.Body.LastDeathY - 40)
	heli := campaignEnemyKind(e.Kind) == campaignEnemyHelicopter
	clr := bigSteel
	if heli {
		clr = bigOlive
	}
	for i := 0; i < 8; i++ {
		a := float64(i)*0.83 + float64(e.Body.ID%7)
		vx := math.Cos(a) * (2.2 + float64(i%3))
		vy := -3.5 - float64(i%4)
		px := x + vx*t
		py := y + vy*t + 0.16*t*t
		rot := t * (0.1 + float64(i%3)*0.05)
		sz := 5.0 + float64(i%3)*3
		pts := [][2]float64{{px - sz, py - sz*0.5}, {px + sz, py - sz*0.6}, {px + sz*0.7, py + sz*0.5}, {px - sz*0.8, py + sz*0.4}}
		for j := range pts {
			pts[j][0], pts[j][1] = rotPt(px, py, pts[j][0], pts[j][1], rot)
		}
		fillPolygon(screen, pts, withAlpha(clr, k))
		strokeClosed(screen, pts, 1, withAlpha(bigRim, k))
	}
	if heli {
		// A broken rotor blade cartwheeling away.
		px, py := x+t*3.2, y-t*2+0.14*t*t
		l := 46.0
		a := t * 0.4
		vector.StrokeLine(screen, float32(px-math.Cos(a)*l), float32(py-math.Sin(a)*l), float32(px+math.Cos(a)*l), float32(py+math.Sin(a)*l), 3, withAlpha(bigRim, k), true)
	}
	// Burning hulk and smoke where it went off.
	drawPuff(screen, x, y+20-t*0.6, 26*k+8, 1, 0.5, 0.15, float32(0.5*k))
	for i := 0; i < 3; i++ {
		drawPuff(screen, x+float64(i-1)*18, y-t*1.2-float64(i)*10, 14+t*0.3, 0.2, 0.2, 0.22, float32(0.45*k))
	}
}

// --- HUD ----------------------------------------------------------------------------

// drawCampaignBigEnemyBars draws the boss bar of every elite HP enemy below
// the bot boss bars; it returns the next free y.
func (g *Game) drawCampaignBigEnemyBars(screen *ebiten.Image, y float64) float64 {
	for _, e := range g.campaignEnemies {
		if e == nil || !e.Elite || e.MaxHP <= 0 {
			continue
		}
		if e.Dead && e.DeadFrames > 30 {
			continue
		}
		var ticks []float64
		if campaignEnemyKind(e.Kind) == campaignEnemyHelicopter {
			ticks = []float64{2.0 / 3, 1.0 / 3}
		}
		frac := float64(e.HP) / float64(e.MaxHP)
		drawBossBar(screen, e.Body.Name, e.Title, frac, ticks, e.Phase, e.alive() && e.HP > 0, y)
		y += 44
	}
	return y
}
