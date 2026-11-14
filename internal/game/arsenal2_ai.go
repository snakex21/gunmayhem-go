package game

import "math"

// Bot behaviour for the second added arsenal. decideAIController calls
// aiArsenal2Decide after its source movement / trigger decisions; the source
// playerAI is unchanged for every other weapon.

// aiPreferredRange is the horizontal distance a bot keeps from its target
// (the source playerAI uses 150 px for every gun).
func aiPreferredRange(p *Player) float64 {
	n := p.Weapon.Def.Number
	switch {
	case isMeleeWeapon(n):
		return 30
	case isLeafBlower(n):
		return 95
	case n == weaponHarpoonGun:
		return 230
	}
	if ballisticShieldInFront(p) {
		// Behind a raised shield a bot can afford to close in.
		return 100
	}
	return 150
}

// aiEdgeLimit is how close to the arena edge a bot still walks toward a
// target (source: 200 px). Melee and the blower need to reach people near
// edges, which is exactly where they are dangerous.
func aiEdgeLimit(p *Player) float64 {
	n := p.Weapon.Def.Number
	if isMeleeWeapon(n) || isLeafBlower(n) {
		return 90
	}
	return 200
}

// aiShotIntoShield reports whether the bot's bullets would hit target's
// ballistic shield (front raised toward the bot, or a slung shield on a back
// turned to it).
func aiShotIntoShield(p, target *Player) bool {
	if p == nil || target == nil {
		return false
	}
	return ballisticShieldCovers(target, sign(target.X-p.X), 0)
}

// aiBulletWeapon: weapons whose push a ballistic shield soaks up.
func aiBulletWeapon(number int) bool {
	if isMeleeWeapon(number) || isLeafBlower(number) || number == weaponHarpoonGun {
		return false
	}
	if spec, ok := developedWeaponSpec(number); ok {
		return spec.ExplosionRadius <= 0 && spec.Projectile != DevelopedProjectileFlame && spec.Projectile != DevelopedProjectilePortal
	}
	return true
}

func aiArsenal2Decide(p *Player, g *Game, target *Player) {
	if p == nil || g == nil || !p.AI || !p.Active {
		return
	}
	n := p.Weapon.Def.Number
	switch {
	case isMeleeWeapon(n):
		aiDecideMelee(p, target)
	case isLeafBlower(n):
		aiDecideBlower(p, g, target)
	case n == weaponHarpoonGun:
		aiDecideHarpoon(p, g, target)
	}
	if target != nil && aiShotIntoShield(p, target) && aiBulletWeapon(n) {
		aiFlankShield(p, g, target)
	}
	if ballisticShieldInFront(p) {
		aiFaceThreatWithShield(p, g)
	}
}

// aiDecideMelee swings only when the target is inside the bat's reach.
func aiDecideMelee(p *Player, target *Player) {
	if target == nil {
		p.AIShoot = false
		return
	}
	dx := target.X - p.X
	facing := (dx >= 0 && p.Facing == 1) || (dx < 0 && p.Facing == -1)
	reach := meleeHitbox(p)
	reach.X -= 6
	reach.W += 12
	p.AIShoot = facing && rectsOverlap(reach, target.Hitbox())
	if !facing && math.Abs(dx) < 50 && math.Abs(target.Y-p.Y) < 50 {
		// Turn around on a target right behind.
		p.AILeft, p.AIRight = dx < 0, dx >= 0
	}
}

// aiDecideBlower blows at targets in the cone, and at incoming grenades /
// rockets even without a target.
func aiDecideBlower(p *Player, g *Game, target *Player) {
	ox, oy, f := blowerNozzle(p)
	shoot := false
	if target != nil {
		s := playerRenderScale(target) / 0.8
		if _, ok := blowerConeFactor(ox, oy, f, target.X, target.Y-35*s, 26*s); ok && arsenal2LineClear(g.arena, ox, oy, target.X, target.Y-35*s) {
			shoot = true
		}
	}
	owner := p.scoreOwnerID()
	for i := range g.grenades {
		gr := &g.grenades[i]
		if gr.Dead || gr.OwnerID == owner {
			continue
		}
		if _, ok := blowerConeFactor(ox, oy, f, gr.X, gr.Y, 6); ok && arsenal2LineClear(g.arena, ox, oy, gr.X, gr.Y) {
			shoot = true
		}
	}
	for i := range g.bullets {
		b := &g.bullets[i]
		if b.Dead || b.OwnerID == owner || !arsenal2Deflectable(b) || b.VX*f >= 0 {
			continue
		}
		if _, ok := blowerConeFactor(ox, oy, f, b.X, b.Y, 10); ok && arsenal2LineClear(g.arena, ox, oy, b.X, b.Y) {
			shoot = true
		}
	}
	p.AIShoot = shoot
}

// aiDecideHarpoon hooks targets that are far enough for the reel to matter,
// preferring ones across a gap (no ground between the bot and the target).
func aiDecideHarpoon(p *Player, g *Game, target *Player) {
	if target == nil || !p.AIShoot {
		return
	}
	dx := math.Abs(target.X - p.X)
	if dx < 120 {
		p.AIShoot = false
		return
	}
	if p.Weapon.Bullets > 1 {
		return
	}
	// Last hook: only across a gap, where the pull is a likely ring-out.
	mid := (target.X + p.X) / 2
	if pointHitsPlatforms(g.arena.Platforms, mid, target.Y+16) {
		p.AIShoot = false
	}
}

// aiFlankShield: bullets into a raised ballistic shield do little, so the
// bot jumps over the bearer to get behind it (throwables score higher too,
// see aiScoreThrow).
func aiFlankShield(p *Player, g *Game, target *Player) {
	dx := target.X - p.X
	if p.JumpNum != 2 || math.Abs(dx) > 170 || math.Abs(target.Y-p.Y) > 40 {
		return
	}
	// Do not leap off the arena: only when there is ground past the target.
	past := target.X + sign(dx)*70
	if !pointHitsPlatforms(g.arena.Platforms, past, target.Y+16) {
		return
	}
	p.AIUp = true
	p.AILeft, p.AIRight = dx < 0, dx >= 0
	if dx < 0 {
		p.AILockLeft, p.AILockRight = 8, 0
	} else {
		p.AILockRight, p.AILockLeft = 8, 0
	}
}

// aiFaceThreatWithShield turns a shield-bearing bot toward a projectile about
// to hit its unprotected back (when it is not busy shooting).
func aiFaceThreatWithShield(p *Player, g *Game) {
	if p.AIShoot || p.JumpNum != 2 {
		return
	}
	for i := range g.bullets {
		b := &g.bullets[i]
		if b.Dead || b.OwnerID == p.scoreOwnerID() || b.Y < p.Y-80 || b.Y > p.Y+5 {
			continue
		}
		dx := p.X - b.X
		if math.Abs(dx) > 180 || dx*b.VX <= 0 {
			continue
		}
		// Incoming from the back side?
		if sign(b.X-p.X) != float64(p.Facing) {
			p.AILeft, p.AIRight = b.X < p.X, b.X >= p.X
			return
		}
	}
}

// aiThermalDetectRange: with THERMAL GOGGLES a bot spots invisible players
// three times further away (aiCanDetectPlayer, normally 135 px).
const (
	aiInvisibleDetectRange = 135.0
	aiThermalDetectRange   = 420.0
)
