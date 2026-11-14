package game

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	ebitentext "github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
)

// Procedural art of the second added arsenal. Everything is drawn from
// synced Player / Bullet state; no image files.

var (
	arsenal2ArtMu      sync.Mutex
	arsenal2WeaponArt  = map[int]*SourceRaster{}
	arsenal2PowerupArt = map[int]*SourceRaster{}
	arsenal2NameArt    = map[int]*SourceRaster{}
	harpoonBoltImage   *ebiten.Image
	thermalShimmerBuf  *ebiten.Image
	thermalOutlineBuf  *ebiten.Image
)

// Grip (hand) pixel of the generated weapon canvases. The fit tables in
// developed_weapon_fit.go use the same values as fractions.
const (
	batArtW, batArtH         = 80, 16
	batGripX, batGripY       = 14.0, 8.0
	blowerArtW, blowerArtH   = 78, 34
	blowerGripX, blowerGripY = 17.5, 22.0
)

func proceduralWeaponRaster(number int) *SourceRaster {
	arsenal2ArtMu.Lock()
	defer arsenal2ArtMu.Unlock()
	if r, ok := arsenal2WeaponArt[number]; ok {
		return r
	}
	var r *SourceRaster
	switch number {
	case weaponBaseballBat:
		r = &SourceRaster{Image: makeBaseballBatArt(), Bounds: Rect{X: -batGripX, Y: -batGripY, W: batArtW, H: batArtH}}
	case weaponLeafBlower:
		r = &SourceRaster{Image: makeLeafBlowerArt(), Bounds: Rect{X: -blowerGripX, Y: -blowerGripY, W: blowerArtW, H: blowerArtH}}
	}
	arsenal2WeaponArt[number] = r
	return r
}

func strokePolygon(dst *ebiten.Image, pts [][2]float64, width float32, clr color.NRGBA) {
	for i := range pts {
		j := (i + 1) % len(pts)
		vector.StrokeLine(dst, float32(pts[i][0]), float32(pts[i][1]), float32(pts[j][0]), float32(pts[j][1]), width, clr, true)
	}
}

// makeBaseballBatArt: a wooden bat, knob left, barrel right, taped handle.
func makeBaseballBatArt() *ebiten.Image {
	img := ebiten.NewImage(batArtW, batArtH)
	cy := float64(batArtH) / 2
	outline := color.NRGBA{R: 0x3b, G: 0x24, B: 0x12, A: 0xff}
	wood := color.NRGBA{R: 0xd8, G: 0xa1, B: 0x5c, A: 0xff}
	shine := color.NRGBA{R: 0xf2, G: 0xcd, B: 0x92, A: 0xff}
	grain := color.NRGBA{R: 0xb0, G: 0x74, B: 0x3a, A: 0xff}
	tape := color.NRGBA{R: 0x2b, G: 0x2b, B: 0x2e, A: 0xff}
	body := func(grow float64) [][2]float64 {
		top := [][2]float64{{7, cy - 2.4 - grow}, {30, cy - 2.6 - grow}, {56, cy - 5.0 - grow}, {71, cy - 5.6 - grow}}
		pts := append([][2]float64(nil), top...)
		for i := 1; i < 8; i++ { // round cap
			a := -math.Pi/2 + float64(i)/8*math.Pi
			pts = append(pts, [2]float64{71 + math.Cos(a)*(5.6+grow), cy + math.Sin(a)*(5.6+grow)})
		}
		for i := len(top) - 1; i >= 0; i-- {
			pts = append(pts, [2]float64{top[i][0], 2*cy - top[i][1]})
		}
		return pts
	}
	fillPolygon(img, body(1.2), outline)
	vector.DrawFilledCircle(img, 5.2, float32(cy), 4.4, outline, true)
	fillPolygon(img, body(0), wood)
	vector.DrawFilledCircle(img, 5.2, float32(cy), 3.2, wood, true)
	// Grain along the barrel and a highlight stripe on top.
	vector.StrokeLine(img, 34, float32(cy+1.2), 68, float32(cy+2.4), 1, grain, true)
	vector.StrokeLine(img, 44, float32(cy-0.6), 70, float32(cy-0.4), 0.8, grain, true)
	vector.StrokeLine(img, 33, float32(cy-2.0), 70, float32(cy-4.2), 1.2, shine, true)
	// Grip tape.
	vector.DrawFilledRect(img, 8, float32(cy-2.4), 18, 4.8, tape, true)
	for x := 9.0; x < 26; x += 3.5 {
		vector.StrokeLine(img, float32(x), float32(cy+2.2), float32(x+2.2), float32(cy-2.2), 0.8, color.NRGBA{R: 0x5a, G: 0x5a, B: 0x60, A: 0xff}, true)
	}
	return img
}

// makeLeafBlowerArt: a handheld blower held like a gun: orange fan housing
// over a pistol grip, fuel tank behind, a straight tube with a flared nozzle.
func makeLeafBlowerArt() *ebiten.Image {
	img := ebiten.NewImage(blowerArtW, blowerArtH)
	outline := color.NRGBA{R: 0x1d, G: 0x1d, B: 0x20, A: 0xff}
	orange := color.NRGBA{R: 0xf0, G: 0x84, B: 0x1e, A: 0xff}
	orangeDark := color.NRGBA{R: 0xa8, G: 0x52, B: 0x0c, A: 0xff}
	tube := color.NRGBA{R: 0x4b, G: 0x4f, B: 0x55, A: 0xff}
	tubeShine := color.NRGBA{R: 0x8a, G: 0x90, B: 0x98, A: 0xff}
	// Fuel tank behind the housing.
	vector.DrawFilledRect(img, 0.5, 7.5, 13, 15, outline, true)
	vector.DrawFilledRect(img, 1.5, 8.5, 11, 13, color.NRGBA{R: 0x33, G: 0x34, B: 0x38, A: 0xff}, true)
	vector.DrawFilledRect(img, 3, 6, 5, 3, orange, true)
	// Pistol grip.
	fillPolygon(img, [][2]float64{{13.5, 19}, {22, 19}, {20.5, 33}, {14, 33}}, outline)
	fillPolygon(img, [][2]float64{{14.8, 20}, {20.8, 20}, {19.6, 32}, {15.2, 32}}, color.NRGBA{R: 0x2e, G: 0x2e, B: 0x31, A: 0xff})
	vector.StrokeLine(img, 21, 22, 24, 26, 1.4, outline, true) // trigger
	// Tube with a joint ring, then the flared nozzle.
	vector.DrawFilledRect(img, 26, 8.5, 41, 9, outline, true)
	vector.DrawFilledRect(img, 27, 9.5, 39, 7, tube, true)
	vector.StrokeLine(img, 28, 11, 66, 11, 1.2, tubeShine, true)
	vector.DrawFilledRect(img, 27, 8, 3.5, 10, outline, true)
	nozzle := [][2]float64{{64, 8}, {76.5, 5.5}, {76.5, 20.5}, {64, 18}}
	fillPolygon(img, nozzle, orange)
	strokePolygon(img, nozzle, 1.2, outline)
	// Fan housing.
	vector.DrawFilledCircle(img, 19, 13, 12, outline, true)
	vector.DrawFilledCircle(img, 19, 13, 10.8, orange, true)
	vector.StrokeCircle(img, 19, 13, 7, 1.4, orangeDark, true)
	vector.DrawFilledCircle(img, 19, 13, 2.6, outline, true)
	vector.StrokeLine(img, 13, 6, 25, 6, 1.1, color.NRGBA{R: 0xff, G: 0xc0, B: 0x7a, A: 0xff}, true)
	return img
}

// ---------------------------------------------------------------------------
// Player-attached art

// meleeDrawRotation is the bat's angle (degrees, facing right): raised at
// rest, a fast sweep through the front after a swing.
func meleeDrawRotation(p *Player) float64 {
	const rest = -55.0
	if p.MeleeSwing <= 0 {
		return rest
	}
	t := float64(batSwingFrames-p.MeleeSwing) / batSwingFrames
	if t < 0 {
		t = 0
	}
	if t > 0.72 {
		// Recover to the raised rest pose over the last frames.
		k := (t - 0.72) / 0.28
		return 52 + (rest-52)*k
	}
	e := 1 - math.Pow(1-t/0.72, 3)
	return -115 + (52+115)*e
}

// gunDrawRotation is the rotation drawGun applies.
func gunDrawRotation(p *Player) float64 {
	if isMeleeWeapon(p.Weapon.Def.Number) {
		return meleeDrawRotation(p)
	}
	return p.VisualGunRotation
}

// akimboOffHandLocal is the player-local position of the off-hand gun (and
// the off hand holding it) during AKIMBO.
func akimboOffHandLocal(p *Player) (gunX, gunY, handX, handY float64) {
	bx, by := akimboOffHandBack/0.8, akimboOffHandDrop/0.8
	gunX = p.VisualHand1X + p.VisualGunX - bx
	gunY = p.VisualHand1Y + p.VisualGunY + by
	handX = p.VisualHand1X + p.VisualHand1ChildX - bx
	handY = p.VisualHand1Y + p.VisualHand1ChildY + by
	return
}

// drawAkimboOffHandGun draws the second handgun at the off hand.
func (g *Game) drawAkimboOffHandGun(screen *ebiten.Image, p *Player) {
	weaponColor := p.PlayerColor
	if source, useGM2 := resolvedPlayerColor(p.PlayerColor, p.UseGM2Cosmetics); useGM2 {
		weaponColor = -source
	}
	gun := g.assets.WeaponFrameForPlayer(p.Weapon.Def.Number, p.Weapon.Frame, weaponColor)
	if gun == nil {
		return
	}
	gx, gy, _, _ := akimboOffHandLocal(p)
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(gun.Bounds.X, gun.Bounds.Y)
	op.GeoM.Rotate(p.VisualGunRotation * math.Pi / 180)
	op.GeoM.Translate(gx, gy)
	op.GeoM.Scale(float64(p.Facing), 1)
	scale := playerRenderScale(p)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(p.X, p.Y)
	op.ColorScale.Scale(0.86, 0.86, 0.86, 1) // a touch darker: it is behind
	if a := p.Alpha * p.Weapon.Alpha; a < 1 {
		op.ColorScale.ScaleAlpha(float32(a))
	}
	screen.DrawImage(gun.Image, op)
}

// Ballistic shield plate, player-local (facing right, before the 0.8 scale).
const (
	bShieldFrontX0, bShieldFrontX1 = 17.0, 31.0
	bShieldTop, bShieldBottom      = -78.0, -8.0
)

func ballisticShieldColors(alpha float64) (plate, rim, slit, shine color.NRGBA) {
	a := func(v uint8) uint8 { return uint8(float64(v) * math.Max(0, math.Min(1, alpha))) }
	return color.NRGBA{R: 0x4c, G: 0x58, B: 0x66, A: a(0xf2)},
		color.NRGBA{R: 0x1a, G: 0x1e, B: 0x24, A: a(0xff)},
		color.NRGBA{R: 0x9c, G: 0xc8, B: 0xe8, A: a(0xb0)},
		color.NRGBA{R: 0xb0, G: 0xc8, B: 0xe0, A: a(0x90)}
}

// drawBallisticShieldFront: the plate held in the off hand, in front.
func (g *Game) drawBallisticShieldFront(screen *ebiten.Image, p *Player) {
	plate, rim, slit, shine := ballisticShieldColors(p.Alpha)
	pt := func(lx, ly float64) [2]float64 {
		x, y := playerPartWorld(p, lx, ly)
		return [2]float64{float64(x), float64(y)}
	}
	bob := (p.VisualHand2Y + 25) * 0.4
	x0, x1 := bShieldFrontX0, bShieldFrontX1
	pts := [][2]float64{pt(x0, bShieldTop+bob), pt(x1, bShieldTop+4+bob), pt(x1, bShieldBottom-4+bob), pt(x0, bShieldBottom+bob)}
	fillPolygon(screen, pts, plate)
	strokePolygon(screen, pts, 1.6, rim)
	// Viewport slit near the top and a glint stripe.
	s0, s1 := pt(x0+2, bShieldTop+16+bob), pt(x1-2, bShieldTop+17+bob)
	vector.StrokeLine(screen, float32(s0[0]), float32(s0[1]), float32(s1[0]), float32(s1[1]), 3, slit, true)
	m0, m1 := pt((x0+x1)/2, bShieldTop+26+bob), pt((x0+x1)/2, bShieldBottom-10+bob)
	vector.StrokeLine(screen, float32(m0[0]), float32(m0[1]), float32(m1[0]), float32(m1[1]), 1.2, shine, true)
	if p.ShieldGearSpark > 0 {
		face := pt(x1+2, (bShieldTop+bShieldBottom)/2+bob)
		g.drawBallisticShieldSpark(screen, p, face[0], face[1], float64(p.Facing))
	}
}

// drawBallisticShieldBack: the plate slung on the back (behind the body).
func (g *Game) drawBallisticShieldBack(screen *ebiten.Image, p *Player) {
	plate, rim, _, shine := ballisticShieldColors(p.Alpha)
	pt := func(lx, ly float64) [2]float64 {
		x, y := playerPartWorld(p, lx, ly)
		return [2]float64{float64(x), float64(y)}
	}
	body := p.VisualBodyY + 60
	pts := [][2]float64{pt(-30, -76+body), pt(-15, -79+body), pt(-11, -14+body), pt(-26, -11+body)}
	fillPolygon(screen, pts, plate)
	strokePolygon(screen, pts, 1.6, rim)
	m0, m1 := pt(-20, -66+body), pt(-17, -22+body)
	vector.StrokeLine(screen, float32(m0[0]), float32(m0[1]), float32(m1[0]), float32(m1[1]), 1.2, shine, true)
	// Sling strap across the torso.
	s0, s1 := pt(-14, -70+body), pt(8, -30+body)
	vector.StrokeLine(screen, float32(s0[0]), float32(s0[1]), float32(s1[0]), float32(s1[1]), 2.2, rim, true)
	if p.ShieldGearSpark > 0 {
		face := pt(-31, -44+body)
		g.drawBallisticShieldSpark(screen, p, face[0], face[1], -float64(p.Facing))
	}
}

func (g *Game) drawBallisticShieldSpark(screen *ebiten.Image, p *Player, fx, fy, dir float64) {
	k := float64(p.ShieldGearSpark) / ballisticShieldSparkFrames
	sy := fy + float64((p.ShieldGearSpark*13+p.ID*7)%24) - 12
	for i := -1; i <= 1; i++ {
		vector.StrokeLine(screen, float32(fx), float32(sy), float32(fx+dir*11*k), float32(sy+float64(i)*7*k), 1.6,
			color.NRGBA{R: 0xff, G: 0xe0, B: 0x60, A: uint8(255 * k)}, true)
	}
	vector.DrawFilledCircle(screen, float32(fx), float32(sy), float32(3*k+1), color.NRGBA{R: 0xff, G: 0xff, B: 0xc0, A: uint8(220 * k)}, true)
}

// drawLeafBlowerGust: streaks of air and a few leaves through the cone,
// phased by the shot counter so it animates identically online.
func (g *Game) drawLeafBlowerGust(screen *ebiten.Image, p *Player) {
	if p.BlowerTime <= 0 || !isLeafBlower(p.Weapon.Def.Number) {
		return
	}
	// p is the screen-space draw copy: the nozzle is in screen coordinates.
	ox, oy, f := blowerNozzle(p)
	fade := float64(p.BlowerTime) / blowerVisualTicks
	phase := p.ShotsFired
	for i := 0; i < 11; i++ {
		d := float64((phase*17 + i*41) % int(blowerRange))
		spread := math.Sin(float64(i)*1.7+0.4) * (blowerNearHalfWidth*0.7 + d*blowerHalfAngleTan*0.8)
		x0 := ox + d*f
		y0 := oy + spread
		ln := 14 + 8*(1-d/blowerRange)
		a := uint8(150 * (1 - d/blowerRange) * fade)
		vector.StrokeLine(screen, float32(x0), float32(y0), float32(x0+ln*f), float32(y0+spread*0.06), 1.4,
			color.NRGBA{R: 0xe8, G: 0xf2, B: 0xff, A: a}, true)
	}
	leaf := []color.NRGBA{{R: 0xd8, G: 0x8a, B: 0x2a, A: 0xff}, {R: 0x8a, G: 0xb0, B: 0x3a, A: 0xff}, {R: 0xc0, G: 0x4a, B: 0x22, A: 0xff}}
	for i := 0; i < 4; i++ {
		d := float64((phase*11 + i*53) % int(blowerRange))
		spread := math.Sin(float64(i)*2.3+float64(phase)*0.21) * (blowerNearHalfWidth*0.5 + d*blowerHalfAngleTan*0.7)
		cx, cy := ox+d*f, oy+spread
		ang := float64(phase*(i+1)) * 0.5
		c := leaf[i%len(leaf)]
		c.A = uint8(230 * (1 - d/blowerRange) * fade)
		pts := make([][2]float64, 0, 8)
		for k := 0; k < 8; k++ {
			a := float64(k) / 8 * 2 * math.Pi
			px, py := math.Cos(a)*4, math.Sin(a)*2
			pts = append(pts, [2]float64{cx + px*math.Cos(ang) - py*math.Sin(ang), cy + px*math.Sin(ang) + py*math.Cos(ang)})
		}
		fillPolygon(screen, pts, c)
	}
}

// drawHarpoonRopes: the rope from the shooter's gun to a harpoon in flight,
// and the taut rope to a hooked victim while it is being reeled in.
func (g *Game) drawHarpoonRopes(screen *ebiten.Image) {
	rope := color.NRGBA{R: 0x8a, G: 0x6a, B: 0x3c, A: 0xff}
	dark := color.NRGBA{R: 0x3a, G: 0x2a, B: 0x16, A: 0xff}
	for i := range g.bullets {
		b := &g.bullets[i]
		if b.Dead || b.DevelopedProjectile != DevelopedProjectileHarpoon {
			continue
		}
		shooter := g.playerByID(b.ShooterID)
		if shooter == nil || !shooter.Active {
			continue
		}
		sx, sy := harpoonRopeAnchor(shooter)
		tail := float64(b.Facing)
		if tail == 0 {
			tail = 1
		}
		drawRopeCurve(screen, g.worldX(sx), g.worldY(sy), g.worldX(b.X-16*tail), g.worldY(b.Y), 0.06, rope, dark, 1)
	}
	for _, t := range g.players {
		if t == nil || !t.Active || t.HarpoonTether <= 0 {
			continue
		}
		shooter := g.playerByID(t.HarpoonTetherID)
		if shooter == nil || !shooter.Active {
			continue
		}
		sx, sy := harpoonRopeAnchor(shooter)
		tx, ty := t.X, t.Y-36*playerRenderScale(t)/0.8
		a := math.Min(1, float64(t.HarpoonTether)/4)
		drawRopeCurve(screen, g.worldX(sx), g.worldY(sy), g.worldX(tx), g.worldY(ty), 0.015, rope, dark, a)
		// The barbed head stuck in the victim.
		dir := sign(sx - tx)
		hx, hy := float32(g.worldX(tx)), float32(g.worldY(ty))
		vector.StrokeLine(screen, hx, hy, hx+float32(dir*9), hy, 2.4, color.NRGBA{R: 0x40, G: 0x44, B: 0x48, A: uint8(255 * a)}, true)
		vector.StrokeLine(screen, hx+float32(dir*3), hy, hx+float32(dir*7), hy-4, 1.6, color.NRGBA{R: 0x70, G: 0x74, B: 0x78, A: uint8(255 * a)}, true)
		vector.StrokeLine(screen, hx+float32(dir*3), hy, hx+float32(dir*7), hy+4, 1.6, color.NRGBA{R: 0x70, G: 0x74, B: 0x78, A: uint8(255 * a)}, true)
	}
}

// harpoonRopeAnchor is the world point the rope leaves the shooter's gun.
func harpoonRopeAnchor(p *Player) (float64, float64) {
	s := playerRenderScale(p) / 0.8
	f := float64(p.Facing)
	if f == 0 {
		f = 1
	}
	return p.X + 34*s*f, p.Y - 38*s
}

// drawRopeCurve draws a slightly sagging rope (sag = fraction of length).
func drawRopeCurve(screen *ebiten.Image, x0, y0, x1, y1, sag float64, rope, dark color.NRGBA, alpha float64) {
	length := math.Hypot(x1-x0, y1-y0)
	if length < 1 {
		return
	}
	rope.A = uint8(float64(rope.A) * alpha)
	dark.A = uint8(float64(dark.A) * alpha)
	const segs = 12
	px, py := x0, y0
	for i := 1; i <= segs; i++ {
		t := float64(i) / segs
		x := x0 + (x1-x0)*t
		y := y0 + (y1-y0)*t + math.Sin(t*math.Pi)*length*sag
		vector.StrokeLine(screen, float32(px), float32(py+0.8), float32(x), float32(y+0.8), 2.4, dark, true)
		vector.StrokeLine(screen, float32(px), float32(py), float32(x), float32(y), 1.4, rope, true)
		px, py = x, y
	}
}

func makeHarpoonBolt() *ebiten.Image {
	img := ebiten.NewImage(44, 14)
	shaft := color.NRGBA{R: 0x50, G: 0x54, B: 0x58, A: 0xff}
	dark := color.NRGBA{R: 0x1a, G: 0x1a, B: 0x1c, A: 0xff}
	vector.DrawFilledRect(img, 2, 5.5, 32, 3, dark, true)
	vector.DrawFilledRect(img, 3, 6, 30, 1.6, shaft, true)
	head := [][2]float64{{31, 2}, {43, 7}, {31, 12}, {34, 7}}
	fillPolygon(img, head, color.NRGBA{R: 0x9a, G: 0xa0, B: 0xa6, A: 0xff})
	strokePolygon(img, head, 1, dark)
	vector.StrokeLine(img, 27, 7, 31, 2.5, 1.4, dark, true) // barbs
	vector.StrokeLine(img, 27, 7, 31, 11.5, 1.4, dark, true)
	vector.DrawFilledCircle(img, 3.5, 7, 2.2, color.NRGBA{R: 0x8a, G: 0x6a, B: 0x3c, A: 0xff}, true) // rope eye
	return img
}

func harpoonProjectileImage() *ebiten.Image {
	arsenal2ArtMu.Lock()
	defer arsenal2ArtMu.Unlock()
	if harpoonBoltImage == nil {
		harpoonBoltImage = makeHarpoonBolt()
	}
	return harpoonBoltImage
}

// ---------------------------------------------------------------------------
// Thermal goggles vs invisibility

// thermalViewer returns the screen's viewer when it wears thermal goggles.
func (g *Game) thermalViewer() *Player {
	v := g.playerByID(g.viewerPlayerID())
	if v == nil || !v.Active || v.AbilityNumber != abilityThermalGoggles {
		return nil
	}
	return v
}

// thermalSeesInvisible: on this screen, p (invisible) shows as a heat
// shimmer. Online and single-human screens are per viewer.
func (g *Game) thermalSeesInvisible(p *Player) bool {
	if p == nil || !p.Active || p.InvisibleTime <= 0 {
		return false
	}
	v := g.thermalViewer()
	return v != nil && v.ID != p.ID && v.scoreOwnerID() != p.scoreOwnerID()
}

// drawThermalInvisibleShimmer draws invisible players as a faint, wavering
// orange heat outline for a thermal-goggles viewer. On a shared local screen
// (no single viewer) a thermal wearer instead marks invisible enemies with a
// small heat ripple, like the smoke chevrons.
func (g *Game) drawThermalInvisibleShimmer(screen *ebiten.Image) {
	if g.thermalViewer() != nil {
		for _, p := range g.players {
			if !g.thermalSeesInvisible(p) {
				continue
			}
			drawP := g.worldDrawPlayer(p)
			drawP.Alpha = 1
			g.drawHeatOutline(screen, &drawP)
		}
		return
	}
	if g.viewerPlayerID() != 0 {
		return // a single viewer without goggles sees nothing
	}
	for _, wearer := range g.players {
		if wearer == nil || !wearer.Active || wearer.AbilityNumber != abilityThermalGoggles {
			continue
		}
		clr := developedPlayerColorSwatch(wearer.PlayerColor)
		for _, p := range g.players {
			if p == nil || !p.Active || p.InvisibleTime <= 0 || p.scoreOwnerID() == wearer.scoreOwnerID() {
				continue
			}
			mx := float32(g.worldX(p.X))
			my := float32(g.worldY(p.Y - 40*playerRenderScale(p)/0.8))
			for k := 0; k < 3; k++ {
				r := float32(6+k*5) + float32((p.InvisibleTime+k*7)%10)*0.6
				c := clr
				c.A = uint8(110 - k*30)
				vector.StrokeCircle(screen, mx, my, r, 1.6, c, true)
			}
		}
	}
}

// drawHeatOutline renders drawP's silhouette as an outline only (the body
// interior is cut out), tinted heat orange and jittered sideways.
func (g *Game) drawHeatOutline(screen *ebiten.Image, drawP *Player) {
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	if thermalShimmerBuf == nil || thermalShimmerBuf.Bounds().Dx() != w || thermalShimmerBuf.Bounds().Dy() != h {
		thermalShimmerBuf = ebiten.NewImage(w, h)
		thermalOutlineBuf = ebiten.NewImage(w, h)
	}
	thermalShimmerBuf.Clear()
	thermalOutlineBuf.Clear()
	g.drawPlayer(thermalShimmerBuf, drawP)
	// Dilate: four offset copies, then cut the original out of the middle.
	for _, o := range [4][2]float64{{-2, 0}, {2, 0}, {0, -2}, {0, 2}} {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(o[0], o[1])
		thermalOutlineBuf.DrawImage(thermalShimmerBuf, op)
	}
	cut := &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationOut}
	thermalOutlineBuf.DrawImage(thermalShimmerBuf, cut)
	wave := math.Sin(float64(drawP.InvisibleTime)*0.7) * 1.5
	for i, a := range []float64{0.55, 0.22} {
		var cm colorm.ColorM
		cm.Scale(0, 0, 0, a)
		cm.Translate(1, 0.5, 0.12, 0)
		op := &colorm.DrawImageOptions{}
		op.GeoM.Translate(wave*float64(1-2*i), float64(i))
		colorm.DrawImage(screen, thermalOutlineBuf, cm, op)
	}
}

// ---------------------------------------------------------------------------
// Powerups and menu icons

// arsenal2PowerupIcon is the floating icon of the AKIMBO / BALLISTIC SHIELD
// powerups, in the source icons' yellow (#ffcc00), centred like Symbol 730.
func arsenal2PowerupIcon(kind int) *SourceRaster {
	if kind != powerupAkimbo && kind != powerupBallisticShield {
		return nil
	}
	arsenal2ArtMu.Lock()
	defer arsenal2ArtMu.Unlock()
	if r, ok := arsenal2PowerupArt[kind]; ok {
		return r
	}
	img := ebiten.NewImage(48, 48)
	yellow := color.NRGBA{R: 0xff, G: 0xcc, A: 0xff}
	dark := color.NRGBA{R: 0x6a, G: 0x44, A: 0xff}
	if kind == powerupAkimbo {
		// Two crossed pistols.
		for _, side := range []float64{-1, 1} {
			ang := side * 0.42
			cx, cy := 24.0, 22.0
			pt := func(x, y float64) [2]float64 {
				rx, ry := rotPt(0, 0, x*side, y, ang)
				return [2]float64{cx + rx, cy + ry}
			}
			slide := [][2]float64{pt(-15, -7), pt(13, -7), pt(13, -1), pt(-15, -1)}
			grip := [][2]float64{pt(-15, -1), pt(-6, -1), pt(-9, 13), pt(-16, 13)}
			fillPolygon(img, grip, dark)
			fillPolygon(img, slide, yellow)
			strokePolygon(img, slide, 1.2, dark)
			fillPolygon(img, [][2]float64{pt(-14, -1), pt(-7, -1), pt(-9.5, 11), pt(-15, 11)}, yellow)
		}
	} else {
		pts := [][2]float64{{14, 6}, {34, 6}, {37, 12}, {37, 34}, {24, 44}, {11, 34}, {11, 12}}
		fillPolygon(img, pts, yellow)
		strokePolygon(img, pts, 2, dark)
		vector.StrokeLine(img, 16, 15, 32, 15, 3, dark, true)
		vector.StrokeLine(img, 24, 20, 24, 38, 1.4, dark, true)
	}
	r := &SourceRaster{Image: img, Bounds: Rect{X: -24, Y: -24, W: 48, H: 48}}
	arsenal2PowerupArt[kind] = r
	return r
}

// arsenal2PowerupNameRaster mirrors fx_powerupname for the new kinds.
func arsenal2PowerupNameRaster(kind int) *SourceRaster {
	if kind != powerupAkimbo && kind != powerupBallisticShield {
		return nil
	}
	arsenal2ArtMu.Lock()
	defer arsenal2ArtMu.Unlock()
	if r, ok := arsenal2NameArt[kind]; ok {
		return r
	}
	label := powerupName(kind)
	face, err := sourceHUDFaceForText(22, label)
	if err != nil || face == nil {
		arsenal2NameArt[kind] = nil
		return nil
	}
	w := font.MeasureString(face, label).Ceil() + 8
	h := face.Metrics().Height.Ceil() + 8
	img := ebiten.NewImage(w, h)
	base := 4 + face.Metrics().Ascent.Ceil()
	for _, o := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {1, 1}} {
		ebitentext.Draw(img, label, face, 4+o[0], base+o[1], color.NRGBA{A: 0xff})
	}
	ebitentext.Draw(img, label, face, 4, base, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	r := &SourceRaster{Image: img, Bounds: Rect{X: -float64(w) / 2, Y: -float64(h) / 2, W: float64(w), H: float64(h)}}
	arsenal2NameArt[kind] = r
	return r
}

// drawBallisticShieldIcon is the gear icon (s ~ 10 for menu tiles).
func drawBallisticShieldIcon(dst *ebiten.Image, cx, cy, s, alpha float64) {
	k := s / 10
	plate := withAlpha(color.NRGBA{R: 0x6a, G: 0x7a, B: 0x8c, A: 0xff}, alpha)
	rim := withAlpha(color.NRGBA{R: 0x1a, G: 0x1e, B: 0x24, A: 0xff}, alpha)
	slit := withAlpha(color.NRGBA{R: 0x9c, G: 0xc8, B: 0xe8, A: 0xff}, alpha)
	pts := [][2]float64{{cx - 6*k, cy - 10*k}, {cx + 6*k, cy - 10*k}, {cx + 7*k, cy - 7*k}, {cx + 7*k, cy + 8*k}, {cx, cy + 11*k}, {cx - 7*k, cy + 8*k}, {cx - 7*k, cy - 7*k}}
	fillPolygon(dst, pts, plate)
	strokePolygon(dst, pts, float32(1.5*k), rim)
	vector.StrokeLine(dst, float32(cx-4.5*k), float32(cy-5*k), float32(cx+4.5*k), float32(cy-5*k), float32(2.2*k), slit, true)
}

// countBallisticShieldSparks counts fresh shield ricochets (sound cue).
func countBallisticShieldSparks(players []*Player) int {
	n := 0
	for _, p := range players {
		if p != nil && p.ShieldGearSpark == ballisticShieldSparkFrames {
			n++
		}
	}
	return n
}
