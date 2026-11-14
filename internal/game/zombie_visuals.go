package game

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// zombie_visuals.go: procedural zombie art, the mystery box, down markers
// and the in-game zombie HUD. No image files: every zombie pose is drawn
// once with vector shapes into a cached image (zombieSprite) in the Gun
// Mayhem look - big round head, thick black outline, small body, floating
// hands - and then blitted, so 48 zombies cost 48 image draws per frame.
// Kinds are colour-coded and have distinct silhouettes.

var zombieTagColor = color.NRGBA{R: 0x3f, G: 0x8f, B: 0x3a, A: 0xff}

type zombiePalette struct {
	skin, skinDark, shirt, eye color.NRGBA
}

var zombiePalettes = [zombieKindCount]zombiePalette{
	zombieWalker:   {skin: rgb(0x86, 0xb0, 0x5a), skinDark: rgb(0x5c, 0x80, 0x3c), shirt: rgb(0x5e, 0x4d, 0x3c), eye: rgb(0xff, 0xe0, 0x40)},
	zombieRunner:   {skin: rgb(0xc8, 0xcc, 0x6e), skinDark: rgb(0x9a, 0xa0, 0x48), shirt: rgb(0xb0, 0x30, 0x2c), eye: rgb(0xff, 0x30, 0x20)},
	zombieBrute:    {skin: rgb(0x86, 0x66, 0x9c), skinDark: rgb(0x5e, 0x44, 0x72), shirt: rgb(0x4a, 0x3a, 0x30), eye: rgb(0xff, 0x9a, 0x20)},
	zombieExploder: {skin: rgb(0xd0, 0xa8, 0x48), skinDark: rgb(0xa0, 0x78, 0x2c), shirt: rgb(0xe8, 0x80, 0x20), eye: rgb(0xff, 0xf0, 0x80)},
	zombieLeaper:   {skin: rgb(0x62, 0x98, 0xb4), skinDark: rgb(0x40, 0x6c, 0x86), shirt: rgb(0x2c, 0x34, 0x44), eye: rgb(0x90, 0xff, 0xff)},
	zombieBoss:     {skin: rgb(0xb0, 0x38, 0x30), skinDark: rgb(0x7a, 0x20, 0x1c), shirt: rgb(0x30, 0x24, 0x22), eye: rgb(0xff, 0xf0, 0x40)},
}

func rgb(r, g, b uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: 0xff} }

var zombieInk = color.NRGBA{R: 0x14, G: 0x12, B: 0x12, A: 0xff}

// Poses of the sprite cache.
const (
	zombiePoseWalk   = 0 // + frame 0..7
	zombiePoseAttack = 8 // + frame 0..2 (wind-up, strike, recover)
	zombiePoseAir    = 11
	zombiePoseCount  = 12
)

const zombieWalkFrames = 8

// zombieFigureScale: figure units to pixels at Scale 1, so the drawn zombie
// (head top to feet) fills the 0.8-scale player hitbox like a GM character.
const zombieFigureScale = 0.8 * 1.18

// zombieSprite is one cached pose.
type zombieSprite struct {
	img    *ebiten.Image
	ox, oy float64 // feet position inside img (pixels)
	res    float64 // pixels per world unit
}

var zombieSprites [zombieKindCount][zombiePoseCount]*zombieSprite

func zombieSpriteRes(k zombieKind) float64 {
	if k == zombieBoss {
		return 1.25
	}
	return 2
}

// zombieSpriteFor renders (once) and returns the sprite of a kind and pose.
func zombieSpriteFor(k zombieKind, pose int) *zombieSprite {
	if k <= zombieKindNone || k >= zombieKindCount {
		k = zombieWalker
	}
	if pose < 0 || pose >= zombiePoseCount {
		pose = 0
	}
	if s := zombieSprites[k][pose]; s != nil {
		return s
	}
	scale := k.stats().Scale * zombieFigureScale
	res := zombieSpriteRes(k)
	w := int(math.Ceil(150 * scale * res))
	h := int(math.Ceil(130 * scale * res))
	s := &zombieSprite{img: ebiten.NewImage(w, h), ox: float64(w) / 2, oy: float64(h) - 6*res, res: res}
	drawZombieFigure(s.img, s.ox, s.oy, scale*res, k, pose)
	zombieSprites[k][pose] = s
	return s
}

// drawZombieFigure draws a zombie facing right with its feet at (ox, oy);
// u is pixels per figure unit.
func drawZombieFigure(dst *ebiten.Image, ox, oy, u float64, k zombieKind, pose int) {
	pal := zombiePalettes[k]
	P := func(x, y float64) (float32, float32) { return float32(ox + x*u), float32(oy + y*u) }
	circle := func(x, y, r float64, c color.NRGBA) {
		cx, cy := P(x, y)
		vector.DrawFilledCircle(dst, cx, cy, float32(r*u), c, true)
	}
	outlined := func(x, y, r float64, c color.NRGBA) {
		circle(x, y, r+2.2, zombieInk)
		circle(x, y, r, c)
	}
	line := func(x0, y0, x1, y1, w float64, c color.NRGBA) {
		ax, ay := P(x0, y0)
		bx, by := P(x1, y1)
		vector.StrokeLine(dst, ax, ay, bx, by, float32(w*u), c, true)
		vector.DrawFilledCircle(dst, ax, ay, float32(w*u/2), c, true)
		vector.DrawFilledCircle(dst, bx, by, float32(w*u/2), c, true)
	}
	limb := func(x0, y0, x1, y1, w float64, c color.NRGBA) {
		line(x0, y0, x1, y1, w+4.2, zombieInk)
		line(x0, y0, x1, y1, w, c)
	}
	poly := func(c color.NRGBA, pts ...float64) {
		ps := make([][2]float64, 0, len(pts)/2)
		for i := 0; i+1 < len(pts); i += 2 {
			x, y := P(pts[i], pts[i+1])
			ps = append(ps, [2]float64{float64(x), float64(y)})
		}
		fillPolygon(dst, ps, c)
	}

	// Pose parameters.
	walk := 0.0
	attack := -1
	air := false
	switch {
	case pose < zombiePoseAttack:
		walk = float64(pose) / zombieWalkFrames * 2 * math.Pi
	case pose < zombiePoseAir:
		attack = pose - zombiePoseAttack
	default:
		air = true
	}
	stride := 0.55
	lean := 3.0 // head / torso forward
	switch k {
	case zombieRunner:
		stride, lean = 0.85, 8
	case zombieBrute, zombieBoss:
		stride, lean = 0.4, 2
	case zombieLeaper:
		stride, lean = 0.6, 9
	case zombieExploder:
		stride, lean = 0.4, 0
	}
	bob := math.Abs(math.Sin(walk)) * 1.6
	legA := math.Sin(walk) * stride
	legB := -legA
	if air {
		legA, legB = 0.7, -0.2
	}
	hipY := -15.0
	legLen := 15.0
	if k == zombieLeaper {
		hipY, legLen = -13, 14
	}
	torsoW, torsoH := 15.0, 18.0
	headR := 14.5
	headY := -44.0
	switch k {
	case zombieRunner:
		torsoW, headR = 12, 13
	case zombieBrute, zombieBoss:
		torsoW, torsoH, headR, headY = 26, 22, 11.5, -46
	case zombieExploder:
		torsoW, torsoH = 24, 22
	case zombieLeaper:
		headR, headY = 13, -38
	}
	shoulderY := hipY - torsoH + 4 - bob

	// Legs (back leg first, darker).
	footA := [2]float64{math.Sin(legA) * legLen, hipY + math.Cos(legA)*legLen}
	footB := [2]float64{math.Sin(legB) * legLen, hipY + math.Cos(legB)*legLen}
	if k == zombieLeaper && !air {
		// Bent legs: knees forward.
		limb(-2, hipY-bob, 6+footB[0]*0.4, hipY+7, 4.5, pal.skinDark)
		limb(6+footB[0]*0.4, hipY+7, footB[0]-2, 0, 4.5, pal.skinDark)
		limb(2, hipY-bob, 9+footA[0]*0.4, hipY+6, 4.5, pal.skin)
		limb(9+footA[0]*0.4, hipY+6, footA[0]+1, 0, 4.5, pal.skin)
	} else {
		lw := 5.0
		if k == zombieBrute || k == zombieBoss {
			lw = 8
		}
		limb(-2, hipY-bob, footB[0]-2, footB[1], lw, pal.shirt)
		limb(2, hipY-bob, footA[0]+1, footA[1], lw, pal.shirt)
	}

	// Back arm (behind the body).
	armReach := 22.0
	armY := shoulderY + 3
	handBack := [2]float64{lean + armReach - 4 + math.Sin(walk+1.3)*2.5, armY + 2 + math.Cos(walk)*1.5}
	handFront := [2]float64{lean + armReach + math.Sin(walk)*2.5, armY + 4 - math.Cos(walk)*1.5}
	switch attack {
	case 0: // wind-up: arms raised back and up
		handBack = [2]float64{lean - 4, shoulderY - 16}
		handFront = [2]float64{lean + 2, shoulderY - 20}
	case 1: // strike: arms thrust forward and down
		handBack = [2]float64{lean + armReach + 8, armY + 6}
		handFront = [2]float64{lean + armReach + 12, armY + 10}
	case 2:
		handBack = [2]float64{lean + armReach, armY + 6}
		handFront = [2]float64{lean + armReach + 4, armY + 8}
	}
	if air {
		handBack = [2]float64{lean + 16, shoulderY - 12}
		handFront = [2]float64{lean + 22, shoulderY - 8}
	}
	if k == zombieExploder {
		// Short arms hugging the belly.
		handBack = [2]float64{lean + 12, armY + 6 + math.Sin(walk)}
		handFront = [2]float64{lean + 15, armY + 9 - math.Sin(walk)}
		if attack >= 0 {
			handBack = [2]float64{lean + 2, shoulderY - 14}
			handFront = [2]float64{lean + 8, shoulderY - 16}
		}
	}
	handR := 4.6
	armW := 4.0
	switch k {
	case zombieBrute, zombieBoss:
		handR, armW = 8, 7
	case zombieRunner:
		handR, armW = 4, 3.4
	}
	limb(lean*0.4, shoulderY+2, handBack[0], handBack[1], armW, pal.skinDark)
	outlined(handBack[0], handBack[1], handR, pal.skinDark)

	// Torso.
	tx := lean * 0.4
	switch k {
	case zombieExploder:
		cy := hipY - torsoH/2 - bob
		outlined(tx+2, cy, torsoW/2+3, pal.shirt)
		// Pustules and a glow spot.
		circle(tx+8, cy-5, 3.2, color.NRGBA{R: 0xff, G: 0xd0, B: 0x60, A: 0xff})
		circle(tx-4, cy+4, 2.6, color.NRGBA{R: 0xb0, G: 0x50, B: 0x10, A: 0xff})
		circle(tx+6, cy+7, 2.2, color.NRGBA{R: 0xb0, G: 0x50, B: 0x10, A: 0xff})
		circle(tx-6, cy-6, 1.8, color.NRGBA{R: 0xff, G: 0xe8, B: 0xa0, A: 0xff})
	case zombieBrute, zombieBoss:
		top := shoulderY - 2
		poly(zombieInk, tx-torsoW/2-3, top-3, tx+torsoW/2+5, top-3, tx+torsoW/2-1, hipY+3-bob, tx-torsoW/2+1, hipY+3-bob)
		poly(pal.skin, tx-torsoW/2, top, tx+torsoW/2+2, top, tx+torsoW/2-3, hipY-bob, tx-torsoW/2+3, hipY-bob)
		// Scars / belt.
		line(tx-torsoW/2+4, hipY-4-bob, tx+torsoW/2-4, hipY-4-bob, 3, pal.shirt)
		line(tx-3, top+5, tx+6, top+11, 1.6, pal.skinDark)
		line(tx+4, top+4, tx+1, top+12, 1.6, pal.skinDark)
		if k == zombieBoss {
			// Bone spikes on the shoulders.
			poly(zombieInk, tx-torsoW/2-2, top+1, tx-torsoW/2+4, top-14, tx-torsoW/2+9, top+1)
			poly(rgb(0xe8, 0xe0, 0xc8), tx-torsoW/2, top, tx-torsoW/2+4, top-11, tx-torsoW/2+7, top)
			poly(zombieInk, tx+torsoW/2-6, top+1, tx+torsoW/2, top-15, tx+torsoW/2+5, top+1)
			poly(rgb(0xe8, 0xe0, 0xc8), tx+torsoW/2-4, top, tx+torsoW/2, top-12, tx+torsoW/2+3, top)
		}
	default:
		top := shoulderY - 1
		poly(zombieInk, tx-torsoW/2-2.2, top-2.2, tx+torsoW/2+2.2, top-2.2, tx+torsoW/2+1.2, hipY+2.2-bob, tx-torsoW/2-1.2, hipY+2.2-bob)
		poly(pal.shirt, tx-torsoW/2, top, tx+torsoW/2, top, tx+torsoW/2-1, hipY-bob, tx-torsoW/2+1, hipY-bob)
		// Torn hem.
		poly(pal.skinDark, tx-torsoW/2+1, hipY-bob, tx-2, hipY-bob-4, tx+3, hipY-bob, tx+torsoW/2-1, hipY-bob-3, tx+torsoW/2-1, hipY-bob)
	}

	// Head.
	hx := lean + 1
	hy := headY - bob
	if k == zombieBrute || k == zombieBoss {
		hy = shoulderY - headR + 2
		hx = lean + torsoW*0.25
	}
	outlined(hx, hy, headR, pal.skin)
	// Rot patch and stitches.
	circle(hx-headR*0.45, hy-headR*0.35, headR*0.28, pal.skinDark)
	line(hx-headR*0.7, hy+headR*0.15, hx-headR*0.25, hy+headR*0.2, 1.2, zombieInk)
	for i := 0; i < 3; i++ {
		sx := hx - headR*0.62 + float64(i)*headR*0.18
		line(sx, hy+headR*0.08, sx, hy+headR*0.3, 0.9, zombieInk)
	}
	// Eyes: dark sockets with a glowing dot, looking forward.
	eyeY := hy - headR*0.12
	e1, e2 := hx+headR*0.28, hx+headR*0.68
	circle(e1, eyeY, headR*0.2, zombieInk)
	circle(e2, eyeY+0.6, headR*0.17, zombieInk)
	circle(e1+0.6, eyeY, headR*0.09, pal.eye)
	circle(e2+0.5, eyeY+0.6, headR*0.08, pal.eye)
	// Jagged mouth.
	my := hy + headR*0.45
	line(hx+headR*0.15, my, hx+headR*0.35, my-1.5, 1.4, zombieInk)
	line(hx+headR*0.35, my-1.5, hx+headR*0.55, my+0.5, 1.4, zombieInk)
	line(hx+headR*0.55, my+0.5, hx+headR*0.78, my-1, 1.4, zombieInk)
	if attack == 1 || k == zombieRunner || k == zombieBoss {
		circle(hx+headR*0.5, my+1.6, headR*0.16, color.NRGBA{R: 0x60, G: 0x10, B: 0x10, A: 0xff})
	}
	if k == zombieBoss {
		// Horns.
		poly(zombieInk, hx-headR*0.6, hy-headR*0.6, hx-headR*1.3, hy-headR*1.7, hx-headR*0.1, hy-headR*0.95)
		poly(rgb(0xe8, 0xe0, 0xc8), hx-headR*0.55, hy-headR*0.7, hx-headR*1.15, hy-headR*1.5, hx-headR*0.2, hy-headR*0.92)
		poly(zombieInk, hx+headR*0.2, hy-headR*0.95, hx+headR*0.9, hy-headR*1.9, hx+headR*0.75, hy-headR*0.6)
		poly(rgb(0xe8, 0xe0, 0xc8), hx+headR*0.28, hy-headR*0.92, hx+headR*0.85, hy-headR*1.7, hx+headR*0.68, hy-headR*0.68)
	}

	// Front arm and hand.
	limb(lean*0.4+2, shoulderY+3, handFront[0], handFront[1], armW, pal.skin)
	outlined(handFront[0], handFront[1], handR, pal.skin)
	if k == zombieLeaper {
		// Claws.
		for i := -1; i <= 1; i++ {
			line(handFront[0]+handR*0.6, handFront[1]+float64(i)*2.2, handFront[0]+handR*1.9, handFront[1]+float64(i)*3.2, 1.3, rgb(0xf0, 0xf0, 0xe0))
		}
	}
	if k == zombieBrute || k == zombieBoss {
		// Knuckles.
		for i := -1; i <= 1; i++ {
			circle(handFront[0]+handR*0.55, handFront[1]+float64(i)*handR*0.45, handR*0.22, pal.skinDark)
		}
	}
}

func (z *Zombie) pose() int {
	switch z.State {
	case zombieStateAttack:
		if z.StateTimer > z.kind().stats().Windup/3 {
			return zombiePoseAttack
		}
		return zombiePoseAttack + 1
	case zombieStateLeap, zombieStateSlam:
		return zombiePoseAir
	}
	if z.Airborne {
		return zombiePoseAir
	}
	speed := math.Abs(z.VX)
	if speed < 0.4 {
		return zombiePoseWalk
	}
	rate := 3
	if z.kind() == zombieRunner || speed > 4 {
		rate = 2
	}
	return zombiePoseWalk + (z.Anim/rate)%zombieWalkFrames
}

// --- world ---------------------------------------------------------------------------

// drawZombieWorld draws the mystery box, downed players, zombies and the
// points popups (below the players).
func (g *Game) drawZombieWorld(screen *ebiten.Image) {
	if !g.zombieActive() {
		return
	}
	m := g.zombie
	g.drawZombieBox(screen)
	g.drawZombieDownMarkers(screen)
	for _, z := range m.zombies {
		if z != nil {
			g.drawZombie(screen, z)
		}
	}
	for _, p := range m.pops {
		g.drawZombiePop(screen, p)
	}
}

func (g *Game) drawZombie(screen *ebiten.Image, z *Zombie) {
	k := z.kind()
	if z.Dead && z.RingOut {
		return
	}
	x, y := g.worldX(z.X), g.worldY(z.Y)
	if x < -200 || x > ScreenWidth+200 || y < -300 || y > ScreenHeight+300 {
		return
	}
	if z.State == zombieStateRising {
		g.drawZombieRising(screen, z, x, y)
		return
	}
	s := zombieSpriteFor(k, z.pose())
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(-s.ox, -s.oy)
	f := float64(z.Facing)
	if f == 0 {
		f = 1
	}
	op.GeoM.Scale(f/s.res, 1/s.res)
	if z.Dead {
		// Topple backwards and fade.
		t := math.Min(1, float64(z.DeadFrames)/8)
		op.GeoM.Rotate(-f * t * math.Pi / 2)
		a := 1.0
		if z.DeadFrames > 10 {
			a = math.Max(0, 1-float64(z.DeadFrames-10)/float64(zombieDeathFrames-10))
		}
		op.ColorScale.Scale(0.6, 0.6, 0.6, 1)
		op.ColorScale.ScaleAlpha(float32(a))
	}
	op.GeoM.Translate(x, y)
	if z.HitFlash > 0 && z.HitFlash%2 == 1 && !z.Dead {
		op.ColorScale.Scale(1.9, 1.9, 1.9, 1)
	}
	screen.DrawImage(s.img, op)
	if z.Dead {
		return
	}
	if k == zombieExploder {
		g.drawExploderGlow(screen, z, x, y)
	}
	if z.HP < z.MaxHP && k != zombieBoss {
		scale := k.stats().Scale
		w := 28 * math.Max(1, scale*0.8)
		bar := Rect{X: x - w/2, Y: y - 70*scale*0.8 - 12, W: w, H: 4}
		vector.DrawFilledRect(screen, float32(bar.X-1), float32(bar.Y-1), float32(bar.W+2), float32(bar.H+2), color.NRGBA{A: 0xb0}, false)
		frac := math.Max(0, float64(z.HP)/float64(max(z.MaxHP, 1)))
		vector.DrawFilledRect(screen, float32(bar.X), float32(bar.Y), float32(bar.W*frac), float32(bar.H), color.NRGBA{R: 0xe0, G: 0x40, B: 0x30, A: 0xff}, false)
	}
}

func (g *Game) drawExploderGlow(screen *ebiten.Image, z *Zombie, x, y float64) {
	scale := zombieStats[zombieExploder].Scale * zombieFigureScale
	cy := y - 26*scale
	pulse := 0.5 + 0.5*math.Sin(float64(z.Anim)*0.25)
	a := 0.22 + 0.18*pulse
	r := 16 * scale
	if z.State == zombieStateFuse {
		k := 1 - float64(z.StateTimer)/float64(zombieExploderFuse)
		a = 0.4 + 0.5*k
		r *= 1 + 0.5*k
		if (z.StateTimer/2)%2 == 0 {
			vector.DrawFilledCircle(screen, float32(x), float32(cy), float32(r*0.8), color.NRGBA{R: 0xff, G: 0xff, B: 0xc0, A: 0x90}, true)
		}
	}
	vector.DrawFilledCircle(screen, float32(x), float32(cy), float32(r), color.NRGBA{R: 0xff, G: 0x90, B: 0x20, A: uint8(255 * a)}, true)
}

// drawZombieRising: a glowing crack first, then the zombie climbs out.
func (g *Game) drawZombieRising(screen *ebiten.Image, z *Zombie, x, y float64) {
	k := z.kind()
	total := zombieRiseFrames
	tele := zombieRiseTelegraph
	if k == zombieBoss {
		total *= 2
		tele *= 2
	}
	elapsed := total - z.StateTimer
	scale := k.stats().Scale
	pulse := 0.5 + 0.5*math.Sin(float64(elapsed)*0.45)
	glow := color.NRGBA{R: 0x80, G: 0xff, B: 0x50, A: uint8(80 + 100*pulse)}
	if k == zombieBoss {
		glow = color.NRGBA{R: 0xff, G: 0x40, B: 0x30, A: uint8(90 + 120*pulse)}
	}
	rw := 22 * scale
	pts := make([][2]float64, 0, 16)
	for i := 0; i < 16; i++ {
		a := float64(i) / 16 * 2 * math.Pi
		pts = append(pts, [2]float64{x + math.Cos(a)*rw, y - 1 + math.Sin(a)*rw*0.22})
	}
	fillPolygon(screen, pts, glow)
	vector.StrokeLine(screen, float32(x-rw*0.7), float32(y-1), float32(x+rw*0.6), float32(y-2), 2, zombieInk, true)
	vector.StrokeLine(screen, float32(x-rw*0.2), float32(y-3), float32(x+rw*0.1), float32(y+1), 1.6, zombieInk, true)
	if elapsed < tele {
		// Warning chevron above the spot.
		cy := y - 26 - 4*pulse
		vector.StrokeLine(screen, float32(x-7), float32(cy-6), float32(x), float32(cy), 2.5, glow, true)
		vector.StrokeLine(screen, float32(x+7), float32(cy-6), float32(x), float32(cy), 2.5, glow, true)
		return
	}
	f := float64(elapsed-tele) / float64(total-tele)
	s := zombieSpriteFor(k, zombiePoseAttack)
	// Show the top f of the figure with its lower edge on the ground line.
	visEnd := int(math.Round(f * s.oy))
	if visEnd <= 0 {
		return
	}
	part := s.img.SubImage(image.Rect(0, 0, s.img.Bounds().Dx(), min(visEnd, s.img.Bounds().Dy()))).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(-s.ox, -float64(visEnd))
	f2 := float64(z.Facing)
	if f2 == 0 {
		f2 = 1
	}
	op.GeoM.Scale(f2/s.res, 1/s.res)
	op.GeoM.Translate(x, y)
	op.ColorScale.Scale(0.85, 0.85, 0.85, 1)
	screen.DrawImage(part, op)
	// Dirt flying up.
	for i := 0; i < 4; i++ {
		a := float64(i)*1.9 + float64(elapsed)*0.2
		dx := math.Cos(a) * rw * 0.8
		dy := -math.Abs(math.Sin(a*1.3)) * 14 * scale
		vector.DrawFilledCircle(screen, float32(x+dx), float32(y+dy-2), float32(2.2*scale), rgb(0x5a, 0x40, 0x2a), true)
	}
}

func (g *Game) drawZombiePop(screen *ebiten.Image, p ZombiePop) {
	x, y := g.worldX(p.X), g.worldY(p.Y)
	a := math.Min(1, float64(p.Life)/12)
	var text string
	clr := uiTextLight
	size := 14.0
	switch p.Kind {
	case zombiePopHit:
		text = "+" + strconv.Itoa(p.Amount)
		clr = color.NRGBA{R: 0xf0, G: 0xf0, B: 0xe0, A: 0xff}
		size = 13
	case zombiePopKill:
		text = "+" + strconv.Itoa(p.Amount)
		clr = color.NRGBA{R: 0xff, G: 0xd8, B: 0x40, A: 0xff}
		size = 17
	case zombiePopHeadshot:
		text = "HEADSHOT +" + strconv.Itoa(p.Amount)
		clr = uiOrange
		size = 17
	case zombiePopRingOut:
		text = "RING-OUT +" + strconv.Itoa(p.Amount)
		clr = color.NRGBA{R: 0x60, G: 0xd8, B: 0xff, A: 0xff}
		size = 17
	case zombiePopRevive:
		text = "REVIVE +" + strconv.Itoa(p.Amount)
		clr = uiOK
		size = 17
	case zombiePopSpend:
		text = strconv.Itoa(p.Amount)
		clr = color.NRGBA{R: 0xff, G: 0x60, B: 0x50, A: 0xff}
		size = 15
	}
	uiText(screen, text, true, size, uiWithAlpha(color.NRGBA{A: 0xff}, a*0.7), x+1.5, y+1.5, uiAlignCenter)
	uiText(screen, text, true, size, uiWithAlpha(clr, a), x, y, uiAlignCenter)
}

// --- mystery box --------------------------------------------------------------------

func (g *Game) drawZombieBox(screen *ebiten.Image) {
	m := g.zombie
	b := m.hud.Box
	if b.X == 0 && b.Y == 0 {
		return
	}
	x, y := g.worldX(b.X), g.worldY(b.Y)
	lift := 0.0
	alpha := 1.0
	if b.State == zombieBoxLeaving {
		k := 1 - float64(b.Timer)/zombieBoxLeaveFrames
		lift = k * 60
		alpha = 1 - k
	}
	pulse := 0.5 + 0.5*math.Sin(float64(m.hud.Box.Timer+m.hud.Kills+int(g.mapFXFrame))*0.2)
	if b.State == zombieBoxRolling || b.State == zombieBoxOffer {
		// Light beam.
		fillPolygon(screen, [][2]float64{{x - 22, y - 30}, {x + 22, y - 30}, {x + 40, y - 170}, {x - 40, y - 170}},
			color.NRGBA{R: 0xff, G: 0xf0, B: 0xa0, A: uint8(40 + 40*pulse)})
	}
	w, h := 64.0, 30.0
	bx, by := x-w/2, y-h-lift
	ink := withAlpha(zombieInk, alpha)
	vector.DrawFilledRect(screen, float32(bx-2), float32(by-2), float32(w+4), float32(h+4), ink, true)
	vector.DrawFilledRect(screen, float32(bx), float32(by), float32(w), float32(h), withAlpha(rgb(0x7a, 0x4e, 0x28), alpha), true)
	for i := 1; i < 3; i++ {
		ly := by + float64(i)*h/3
		vector.StrokeLine(screen, float32(bx), float32(ly), float32(bx+w), float32(ly), 1.2, withAlpha(rgb(0x55, 0x34, 0x18), alpha), true)
	}
	// Lid (open while rolling / offering).
	lidOpen := b.State == zombieBoxRolling || b.State == zombieBoxOffer
	if lidOpen {
		fillPolygon(screen, [][2]float64{{bx - 2, by - 2}, {bx + w + 2, by - 2}, {bx + w - 6, by - 16}, {bx + 4, by - 16}}, withAlpha(rgb(0x5a, 0x38, 0x1c), alpha))
	} else {
		vector.DrawFilledRect(screen, float32(bx-3), float32(by-6), float32(w+6), 8, ink, true)
		vector.DrawFilledRect(screen, float32(bx-1), float32(by-5), float32(w+2), 5, withAlpha(rgb(0x96, 0x62, 0x34), alpha), true)
	}
	// Metal corners.
	for _, cx := range []float64{bx + 3, bx + w - 9} {
		vector.DrawFilledRect(screen, float32(cx), float32(by+3), 6, float32(h-6), withAlpha(rgb(0xb8, 0xb0, 0x90), alpha), true)
	}
	// Glowing "?".
	q := color.NRGBA{R: 0xff, G: 0xe0, B: 0x50, A: uint8(255 * alpha)}
	vector.DrawFilledCircle(screen, float32(x), float32(by+h/2), 11, withAlpha(color.NRGBA{R: 0xff, G: 0xd0, B: 0x40, A: 0x50}, alpha*(0.5+0.5*pulse)), true)
	uiText(screen, "?", true, 22, q, x, by+h/2, uiAlignCenter)

	if (b.State == zombieBoxRolling || b.State == zombieBoxOffer) && b.Weapon != 0 && g.assets != nil {
		float := math.Sin(float64(b.Timer)*0.15) * 3
		r := Rect{X: x - 40, Y: by - 64 + float, W: 80, H: 40}
		drawSetupThumbFit(screen, g.assets.WeaponFrame(b.Weapon, 0), r, 1)
		if b.State == zombieBoxOffer {
			uiText(screen, uiFit(weaponDisplayName(b.Weapon), true, 13, 180), true, 13, uiTextLight, x, by-74+float, uiAlignCenter)
		}
	}
	if b.State == zombieBoxLeaving {
		uiText(screen, "BYE BYE!", true, 18, withAlpha(color.NRGBA{R: 0xff, G: 0x70, B: 0x60, A: 0xff}, alpha), x, by-40, uiAlignCenter)
	}
	// Prompt for nearby local players.
	for _, p := range g.players {
		if p == nil || !p.Active || p.AI || !g.zombieLocalViewer(p) || !m.boxInReach(p) {
			continue
		}
		zp := m.playerState(p.ID)
		text := ""
		switch b.State {
		case zombieBoxIdle:
			text = "DOWN: MYSTERY BOX  " + strconv.Itoa(zombieBoxPrice)
			if zp != nil && zp.Points < zombieBoxPrice {
				text = "MYSTERY BOX  " + strconv.Itoa(zombieBoxPrice) + " - NOT ENOUGH POINTS"
			}
		case zombieBoxOffer:
			if b.OwnerID == p.ID {
				text = "DOWN: TAKE " + weaponDisplayName(b.Weapon)
			}
		}
		if text != "" {
			drawZombieTag(screen, text, x, by-(30+lift)-50)
		}
		break
	}
}

// zombieLocalViewer: p is controlled on this machine (prompts are only for
// the local players; online each guest sees its own).
func (g *Game) zombieLocalViewer(p *Player) bool {
	if g.netplay == nil {
		return true
	}
	return p.ID == g.onlineLocalPlayerID()
}

func drawZombieTag(screen *ebiten.Image, text string, cx, cy float64) {
	w := uiMeasure(true, uiSizeHint, text) + 16
	r := Rect{X: cx - w/2, Y: cy - 11, W: w, H: 22}
	uiFillRound(screen, r, 4, color.NRGBA{R: 0x10, G: 0x12, B: 0x14, A: 0xd8})
	uiTextIn(screen, text, true, uiSizeHint, uiTextLight, r, uiAlignCenter, 4)
}

// --- downed players ---------------------------------------------------------------------

func (g *Game) drawZombieDownMarkers(screen *ebiten.Image) {
	m := g.zombie
	for i := range m.hud.Players {
		zp := &m.hud.Players[i]
		if zp.ID == 0 || (zp.State != zombiePlayerDown && zp.State != zombiePlayerDead) {
			continue
		}
		p := g.playerByID(zp.ID)
		x, y := g.worldX(zp.DownX), g.worldY(zp.DownY)
		col := uiOrange
		if p != nil {
			col = developedPlayerColorSwatch(p.PlayerColor)
		}
		if zp.State == zombiePlayerDead {
			// Gravestone until the wave ends.
			vector.DrawFilledRect(screen, float32(x-13), float32(y-30), 26, 30, zombieInk, true)
			vector.DrawFilledCircle(screen, float32(x), float32(y-30), 13, zombieInk, true)
			vector.DrawFilledRect(screen, float32(x-11), float32(y-29), 22, 29, rgb(0x9a, 0x9a, 0x94), true)
			vector.DrawFilledCircle(screen, float32(x), float32(y-29), 11, rgb(0x9a, 0x9a, 0x94), true)
			uiText(screen, "RIP", true, 13, zombieInk, x, y-24, uiAlignCenter)
			uiText(screen, fmt.Sprintf("P%d", zp.ID), true, 13, col, x, y-48, uiAlignCenter)
			continue
		}
		// Lying figure: a head and a body in the player's colour.
		vector.DrawFilledRect(screen, float32(x-18), float32(y-11), 30, 11, zombieInk, true)
		vector.DrawFilledRect(screen, float32(x-16), float32(y-9), 26, 7, col, true)
		vector.DrawFilledCircle(screen, float32(x+17), float32(y-10), 10, zombieInk, true)
		vector.DrawFilledCircle(screen, float32(x+17), float32(y-10), 8, rgb(0xf0, 0xe0, 0xc8), true)
		// Revive ring and bleed-out timer.
		cy := y - 46
		vector.DrawFilledCircle(screen, float32(x), float32(cy), 17, color.NRGBA{R: 0x10, G: 0x12, B: 0x14, A: 0xc0}, true)
		frac := float64(zp.ReviveProgress) / zombieReviveFrames
		if frac > 0 {
			var arc vector.Path
			arc.Arc(float32(x), float32(cy), 15, -math.Pi/2, float32(-math.Pi/2+2*math.Pi*frac), vector.Clockwise)
			strokePathColor(screen, &arc, uiOK, 4)
		} else {
			var ring vector.Path
			ring.Arc(float32(x), float32(cy), 15, 0, 2*math.Pi, vector.Clockwise)
			blink := (zp.DownTimer/10)%2 == 0
			c := color.NRGBA{R: 0xff, G: 0x50, B: 0x40, A: 0xff}
			if !blink && zp.DownTimer < 300 {
				c.A = 0x60
			}
			strokePathColor(screen, &ring, c, 2.5)
		}
		secs := (zp.DownTimer + TicksPerSecond - 1) / TicksPerSecond
		uiText(screen, strconv.Itoa(secs), true, 15, uiTextLight, x, cy, uiAlignCenter)
		label := fmt.Sprintf("P%d DOWN", zp.ID)
		if zp.SelfRevives > 0 {
			label = fmt.Sprintf("P%d GETTING UP", zp.ID)
		}
		uiText(screen, label, true, 13, col, x, cy-25, uiAlignCenter)
	}
}

// --- main menu tile art ----------------------------------------------------------------

func (g *Game) drawMainZombieArt(dst *ebiten.Image, r Rect, alpha float64) {
	ground := r.Y + r.H - 62
	draw := func(k zombieKind, pose int, x, scale float64, flip bool) {
		s := zombieSpriteFor(k, pose)
		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterLinear
		op.GeoM.Translate(-s.ox, -s.oy)
		f := scale / s.res
		if flip {
			op.GeoM.Scale(-f, f)
		} else {
			op.GeoM.Scale(f, f)
		}
		op.GeoM.Translate(x, ground)
		op.ColorScale.ScaleAlpha(float32(alpha))
		dst.DrawImage(s.img, op)
	}
	draw(zombieBrute, zombiePoseWalk+2, r.X+r.W*0.68, 0.62, true)
	draw(zombieWalker, zombiePoseWalk+5, r.X+r.W*0.86, 0.8, true)
	draw(zombieRunner, zombiePoseWalk+1, r.X+r.W*0.48, 0.8, true)
}
