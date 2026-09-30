package game

import (
	"math"
	"math/rand"
)

const (
	ThrowableDynamite = iota
	ThrowableMk2
	ThrowableF1
	ThrowableStielhand
	ThrowableRGD5
	ThrowableMolotov
	throwableTypeCount
)

type throwableProfile struct {
	Name               string
	Fuse               int
	Radius             float64
	HorizontalPush     float64
	VerticalPush       float64
	EdgeHorizontalPush float64
	EdgeVerticalPush   float64
}

var throwableProfiles = [...]throwableProfile{
	// Dynamite keeps the original GM1 impulse exactly: at distance d<=200 the
	// source uses (350-d)/5 horizontally and (350-d)/10 vertically. That means
	// the edge of the blast is still dangerous instead of fading almost to zero.
	{Name: "DYNAMITE", Fuse: 50, Radius: 200, HorizontalPush: 70, VerticalPush: 35, EdgeHorizontalPush: 30, EdgeVerticalPush: 15},
	// Added grenades use the same readable Gun Mayhem-style blast curve, but each
	// keeps a distinct personality: F-1 reaches furthest, M24 hits hardest, RGD-5
	// is balanced and Mk2 sits between them.
	{Name: "US MK 2", Fuse: 58, Radius: 185, HorizontalPush: 61, VerticalPush: 39, EdgeHorizontalPush: 23, EdgeVerticalPush: 17},
	{Name: "SOVIET F-1", Fuse: 70, Radius: 220, HorizontalPush: 56, VerticalPush: 34, EdgeHorizontalPush: 24, EdgeVerticalPush: 15},
	{Name: "GERMAN M24", Fuse: 44, Radius: 170, HorizontalPush: 82, VerticalPush: 45, EdgeHorizontalPush: 31, EdgeVerticalPush: 19},
	{Name: "SOVIET RGD-5", Fuse: 54, Radius: 180, HorizontalPush: 58, VerticalPush: 38, EdgeHorizontalPush: 22, EdgeVerticalPush: 16},
	{Name: "MOLOTOV", Fuse: 90, Radius: 100, HorizontalPush: 7, VerticalPush: 4, EdgeHorizontalPush: 2, EdgeVerticalPush: 1},
}

func normalizedThrowableType(kind int) int {
	if kind < 0 || kind >= throwableTypeCount {
		return ThrowableDynamite
	}
	return kind
}

func throwableName(kind int) string {
	return throwableProfiles[normalizedThrowableType(kind)].Name
}

type Grenade struct {
	X, Y           float64
	VX, VY         float64
	Rotation       float64
	RotSpeed       float64
	Fuse           int
	OwnerID        int
	Type           int
	PortalCooldown int
	FreePass       bool
	HitGround      bool
	Dead           bool
}

type FirePatch struct {
	X, Y    float64
	OwnerID int
	Life    int
	Frame   int
	Dead    bool
}

type Explosion struct {
	X, Y float64
	Life int
}

type DynamiteEffect struct {
	X, Y     float64
	VX, VY   float64
	Rotation float64
	RotSpeed float64
	Scale    float64
	Alpha    float64
	Fresh    bool
	Dead     bool
}

func updateGrenadeInput(p *Player, grenades *[]Grenade) {
	if !p.Active || p.Grenades <= 0 {
		return
	}
	// Source wraps offhand input in gamemode != 2, while Gun Game constructor
	// sets offhandnumber=-1. Both modes therefore have no grenade input.
	if p.GameMode == SourceGameModeInstagib || p.GameMode == SourceGameModeGunGame {
		return
	}

	pressed := p.grenadePressed()
	if pressed && !p.GrenadeHeld {
		p.GrenadeHeld = true
		p.GrenadePower = 1
	} else if !pressed && p.GrenadeHeld {
		p.GrenadeHeld = false
		rot := rand.Float64()*10 + 5
		if rand.Intn(2) == 0 {
			rot = -rot
		}
		*grenades = append(*grenades, Grenade{
			X:        p.X + 30*float64(p.Facing),
			Y:        p.Y - 35,
			VX:       float64(p.Facing) * p.GrenadePower,
			VY:       -p.GrenadePower,
			Rotation: float64(p.Facing) * 50,
			RotSpeed: rot,
			OwnerID:  p.scoreOwnerID(),
			Type:     normalizedThrowableType(p.ThrowableType),
		})
		p.Grenades--
	}

	if pressed && p.GrenadePower < 14 {
		p.GrenadePower += 0.5
		if p.GrenadePower > 14 {
			p.GrenadePower = 14
		}
	}
}

func updateGrenades(arena Map, grenades []Grenade, players []*Player, explosions *[]Explosion, dynamite *[]DynamiteEffect, blast *[]BlastEffect, fire *[]FirePatch) {
	for i := range grenades {
		g := &grenades[i]
		if g.Dead {
			continue
		}

		if !g.HitGround {
			g.Rotation += g.RotSpeed
		} else if math.Abs(90-g.Rotation) < math.Abs(-90-g.Rotation) {
			g.Rotation += (90 - g.Rotation) / 5
		} else {
			g.Rotation += (-90 - g.Rotation) / 5
		}

		if g.VY < 24 {
			g.VY += arena.Gravity + 0.12
		}
		// Source spawns fx_dynamite at the grenade's tip before moving the
		// grenade itself this frame.
		dirX := math.Cos((g.Rotation-90)*math.Pi/180) * 25
		dirY := math.Sin((g.Rotation-90)*math.Pi/180) * 25
		if normalizedThrowableType(g.Type) == ThrowableDynamite {
			*dynamite = append(*dynamite, newDynamiteEffect(g.X+dirX, g.Y+dirY))
		}
		previousX, previousY := g.X, g.Y
		g.X += g.VX
		g.Y += g.VY

		if !g.FreePass && g.VY > 0 {
			for _, r := range arena.Platforms {
				if !r.ContainsX(g.X) || g.Y+8 < r.Y || g.Y+8-g.VY > r.Y {
					continue
				}
				g.Y -= g.VY
				for step := 1; step <= 5; step++ {
					y := g.Y + 8 + float64(step)*(g.VY/5)
					if r.Contains(g.X, y) {
						g.Y += g.VY / 5 * float64(step-1)
						g.RotSpeed *= 0.5
						g.VX *= 0.6
						g.VY *= -0.4
						g.HitGround = true
						break
					}
				}
				break
			}
		}

		insideY := false
		insideY10 := false
		for _, r := range arena.Platforms {
			if r.Contains(g.X, g.Y) {
				insideY = true
			}
			if r.Contains(g.X, g.Y+10) {
				insideY10 = true
			}
			if insideY && insideY10 {
				break
			}
		}
		// Exact source order:
		// 1) clear freepass only when neither _Y nor _Y+10 hits ground,
		// 2) set freepass only when _Y itself hits ground.
		// Using (_Y || _Y+10) for step 2 lets the grenade enter a platform
		// several pixels too far before collisions resume.
		if !insideY && !insideY10 {
			g.FreePass = false
		}
		if insideY && !g.FreePass {
			g.FreePass = true
		}

		g.Fuse++
		profile := throwableProfiles[normalizedThrowableType(g.Type)]
		molotovImpact := false
		if g.Type == ThrowableMolotov {
			molotovImpact = g.HitGround
			// A fast Molotov should not tunnel through a thin platform between
			// frames. Break it on the first platform top crossed while descending.
			if !molotovImpact {
				if impactX, impactY, ok := molotovTopImpact(arena, previousX, previousY, g.X, g.Y); ok {
					g.X, g.Y = impactX, impactY
					g.HitGround = true
					molotovImpact = true
				}
			}
		}
		if g.Fuse > profile.Fuse || molotovImpact {
			if g.Type == ThrowableMolotov {
				spawnMolotovFire(arena, g, players, fire)
				g.Dead = true
				continue
			}
			explodeGrenade(g, players)
			*blast = append(*blast, spawnGrenadeBlastEffects(g.X, g.Y-20)...)
			*explosions = append(*explosions, Explosion{X: g.X, Y: g.Y - 20, Life: 18})
			g.Dead = true
			continue
		}

		if g.Y >= 900 || g.X < -500 || g.X > 1400 {
			g.Dead = true
		}
	}
}

func grenadeBlastAxis(center, edge, distance, radius float64) float64 {
	if radius <= 0 || distance >= radius {
		return edge
	}
	t := distance / radius
	return center + (edge-center)*t
}

func explodeGrenade(g *Grenade, players []*Player) {
	profile := throwableProfiles[normalizedThrowableType(g.Type)]
	for _, p := range players {
		if !p.Active {
			continue
		}
		dx := p.X - g.X
		dy := p.Y - 30 - g.Y
		distance := math.Round(math.Sqrt(dx*dx + dy*dy))
		if distance > profile.Radius {
			continue
		}
		radians := math.Atan2(dy, dx)
		pushXMag := grenadeBlastAxis(profile.HorizontalPush, profile.EdgeHorizontalPush, distance, profile.Radius)
		pushYMag := grenadeBlastAxis(profile.VerticalPush, profile.EdgeVerticalPush, distance, profile.Radius)
		pushX := math.Cos(radians) * pushXMag
		pushY := math.Sin(radians) * pushYMag
		if p.PerkNumber == 6 || p.ShieldTime > 0 {
			// Source switches from /5,/10 to /25,/50: exactly one fifth
			// of the normal grenade impulse on each axis.
			pushX /= 5
			pushY /= 5
		}
		p.applyCombatImpulse(pushX, pushY)
		p.LastHitBy = g.OwnerID
		p.HitByGrenade = true
	}
}

func molotovTopImpact(arena Map, x0, y0, x1, y1 float64) (float64, float64, bool) {
	if y1 <= y0 {
		return 0, 0, false
	}
	bestT := 2.0
	bestX, bestY := 0.0, 0.0
	for _, r := range arena.Platforms {
		if y0 > r.Y || y1 < r.Y {
			continue
		}
		t := (r.Y - y0) / (y1 - y0)
		if t < 0 || t > 1 || t >= bestT {
			continue
		}
		x := x0 + (x1-x0)*t
		if x < r.X || x > r.X+r.W {
			continue
		}
		bestT, bestX, bestY = t, x, r.Y
	}
	return bestX, bestY, bestT <= 1
}

func fireSurfaceY(arena Map, x, referenceY float64) (float64, bool) {
	bestDistance := 70.0
	bestY := 0.0
	found := false
	for _, r := range arena.Platforms {
		if x < r.X || x > r.X+r.W {
			continue
		}
		d := math.Abs(r.Y - referenceY)
		if d <= bestDistance {
			bestDistance = d
			bestY = r.Y
			found = true
		}
	}
	return bestY, found
}

func spawnMolotovFire(arena Map, g *Grenade, players []*Player, patches *[]FirePatch) {
	if patches != nil {
		before := len(*patches)
		// Spread only across real platform surface. The old implementation placed
		// seven patches at a fixed Y even across gaps, which made fire hover in air.
		for i := -3; i <= 3; i++ {
			x := g.X + float64(i)*28
			y, ok := fireSurfaceY(arena, x, g.Y)
			if !ok {
				continue
			}
			*patches = append(*patches, FirePatch{
				X: x, Y: y, OwnerID: g.OwnerID,
				Life: 300 - int(math.Abs(float64(i)))*20,
			})
		}
		// Test arenas or unusual geometry can lack a platform list. Keep one
		// center flame rather than silently deleting the Molotov effect entirely.
		if len(*patches) == before {
			*patches = append(*patches, FirePatch{X: g.X, Y: g.Y, OwnerID: g.OwnerID, Life: 300})
		}
	}
	for _, p := range players {
		if p == nil || !p.Active || math.Hypot(p.X-g.X, p.Y-30-g.Y) > 100 {
			continue
		}
		dir := 1
		if p.X < g.X {
			dir = -1
		}
		p.igniteWithPower(g.OwnerID, dir, 165, 20)
		// Molotov is primarily an ignition weapon. Give the bottle impact a small
		// readable shove without turning it into another high-explosive grenade.
		p.applyCombatImpulse(float64(dir)*5.5, -2.8)
		p.LastHitBy = g.OwnerID
		p.HitByGrenade = true
	}
}

func updateFirePatches(patches []FirePatch, players []*Player) {
	for i := range patches {
		patch := &patches[i]
		if patch.Dead {
			continue
		}
		patch.Frame++
		patch.Life--
		if patch.Life <= 0 {
			patch.Dead = true
			continue
		}
		if patch.Frame%6 != 0 {
			continue
		}
		for _, p := range players {
			if p == nil || !p.Active || math.Abs(p.X-patch.X) > 34 {
				continue
			}
			// Player Y is effectively the feet/ground contact. Fire rises from the
			// platform, so it can catch somebody standing in it or jumping just over
			// it, but no longer burns players a full platform-height away.
			vertical := p.Y - patch.Y
			if vertical < -52 || vertical > 16 {
				continue
			}
			dir := 1
			if p.X < patch.X {
				dir = -1
			}
			p.igniteWithPower(patch.OwnerID, dir, 125, 18)
			// Standing in burning Molotov fuel must physically matter in a
			// knockback game. Contact gives a readable pop away from the flame,
			// while a short per-player cooldown prevents seven overlapping fire
			// patches from stacking the impulse in the same frame.
			p.applyFireContactKnockback(patch.OwnerID, dir, 5.5, 4.2, 8)
			p.LastHitBy = patch.OwnerID
		}
	}
}

func compactFirePatches(in []FirePatch) []FirePatch {
	out := in[:0]
	for _, patch := range in {
		if !patch.Dead {
			out = append(out, patch)
		}
	}
	return out
}

func newDynamiteEffect(x, y float64) DynamiteEffect {
	vx := rand.Float64()*10 - 5
	vy := rand.Float64()*4 - 2
	rot := rand.Float64()*10 + 5
	if rand.Intn(2) == 0 {
		rot *= -1
	}
	// DefineSprite_343 frame script moves the new clip once immediately.
	return DynamiteEffect{
		X: x + vx, Y: y + vy,
		VX: vx, VY: vy, RotSpeed: rot,
		Scale: 1, Alpha: 1, Fresh: true,
	}
}

func updateDynamiteEffects(effects []DynamiteEffect) {
	for i := range effects {
		e := &effects[i]
		if e.Dead {
			continue
		}
		if e.Fresh {
			e.Fresh = false
			continue
		}
		e.X += e.VX
		e.VX += e.VX * -0.8
		e.Y += e.VY
		e.Scale -= 0.05
		e.Rotation += e.RotSpeed
		e.Alpha -= 0.20
		if e.Alpha <= 0.01 {
			e.Dead = true
		}
	}
}

func compactDynamiteEffects(in []DynamiteEffect) []DynamiteEffect {
	out := in[:0]
	for _, e := range in {
		if !e.Dead {
			out = append(out, e)
		}
	}
	return out
}

func updateExplosions(explosions []Explosion) {
	for i := range explosions {
		if explosions[i].Life > 0 {
			explosions[i].Life--
		}
	}
}

func compactGrenades(in []Grenade) []Grenade {
	out := in[:0]
	for _, v := range in {
		if !v.Dead {
			out = append(out, v)
		}
	}
	return out
}

func compactExplosions(in []Explosion) []Explosion {
	out := in[:0]
	for _, v := range in {
		if v.Life > 0 {
			out = append(out, v)
		}
	}
	return out
}
