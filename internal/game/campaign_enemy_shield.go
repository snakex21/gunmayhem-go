package game

import "math"

// SHIELD-BEARER: a slow riot trooper that walks the platforms with normal
// player physics (gravity, drop-through, ring-out) and carries a front riot
// shield. Bullets that hit the shield from the front are deflected with a
// spark and almost no push. Shots from behind, steep shots from above,
// explosions, shock gas and fire work normally. A flashbang or tear gas makes
// him lower the shield for a moment. Up close he shoves; at range he fires a
// pistol past the shield edge.

const (
	shieldBearerKnockbackScale = 0.85
	shieldBearerMoveScale      = 0.55
	shieldBearerShoveRange     = 62.0
	shieldBearerShovePush      = 10.5
	shieldBearerShoveLift      = 4.0
	shieldBearerShoveCooldown  = 70
	shieldBearerShotMin        = 90.0
	shieldBearerShotMax        = 380.0
	shieldBearerLowerFrames    = 40 // shield stays down after blind / gas ends
	shieldDeflectPushScale     = 0.06
	// Shield plate in body-local coordinates (facing right).
	shieldLocalX0 = 15.0
	shieldLocalX1 = 26.0
	shieldLocalY0 = -64.0
	shieldLocalY1 = -2.0
)

func (e *CampaignEnemy) shieldUp() bool {
	return e != nil && campaignEnemyKind(e.Kind) == campaignEnemyShieldBearer && e.alive() && e.ShieldDown <= 0
}

// shieldRect is the world rectangle of the raised shield.
func (e *CampaignEnemy) shieldRect() Rect {
	b := &e.Body
	f := float64(b.Facing)
	if f == 0 {
		f = 1
	}
	x0, x1 := b.X+shieldLocalX0*f, b.X+shieldLocalX1*f
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	return Rect{X: x0, Y: b.Y + shieldLocalY0, W: x1 - x0, H: shieldLocalY1 - shieldLocalY0}
}

// shieldBlocksBullet decides whether bl, about to move this frame, strikes
// the front of e's raised shield.
func (e *CampaignEnemy) shieldBlocksBullet(bl *Bullet) bool {
	if !e.shieldUp() || bl == nil || bl.Dead || !bulletCanHitPlayer(bl, &e.Body) {
		return false
	}
	switch bl.DevelopedProjectile {
	case DevelopedProjectileFlame, DevelopedProjectilePortal:
		return false // fire licks around the plate; portals are not hits
	}
	f := float64(e.Body.Facing)
	if f == 0 {
		f = 1
	}
	// Must travel toward the bearer's front ...
	if bl.VX*f >= 0 {
		return false
	}
	// ... mostly horizontally (steep shots from above get over the top) ...
	if math.Abs(bl.VY) > math.Abs(bl.VX)*1.2 {
		return false
	}
	// ... and come from in front of the plate.
	if (bl.X-e.Body.X)*f < shieldLocalX0 {
		return false
	}
	nx, ny := bl.X+bl.VX, bl.Y+bl.VY+bl.Gravity
	return segmentIntersectsRect(bl.X, bl.Y, nx, ny, e.shieldRect())
}

// deflectShieldBullets runs before updateBullets. Ordinary rounds that hit a
// raised shield from the front are deflected; explosive rounds detonate on
// the plate and push the bearer like any blast.
func (g *Game) deflectShieldBullets(targets []*Player) int {
	deflected := 0
	for _, e := range g.campaignEnemies {
		if !e.shieldUp() {
			continue
		}
		f := float64(e.Body.Facing)
		if f == 0 {
			f = 1
		}
		face := e.Body.X + shieldLocalX1*f
		for i := range g.bullets {
			bl := &g.bullets[i]
			if !e.shieldBlocksBullet(bl) {
				continue
			}
			if bl.ExplosionRadius > 0 {
				bl.X = face
				explodeDevelopedProjectileWithDirectHit(bl, nil, targets, &g.explosions, &g.blastFX)
				continue
			}
			sy := math.Max(e.Body.Y+shieldLocalY0+4, math.Min(e.Body.Y+shieldLocalY1-4, bl.Y))
			bl.Dead = true
			// Minimal shove through the stance; no hit counted, no credit.
			e.Body.VX -= bl.Firepower * f * shieldDeflectPushScale
			e.ShieldSpark = 8
			spawnMolotovSparks(&g.dynamiteFX, face, sy, 3)
			deflected++
		}
	}
	return deflected
}

func (g *Game) updateShieldBearer(e *CampaignEnemy, arena Map) {
	b := &e.Body
	if !b.Active {
		return
	}
	// Flashbang or tear gas: the bearer lowers the shield to rub his eyes.
	if b.BlindTime > 0 || b.TearGasTime > 0 {
		e.ShieldDown = shieldBearerLowerFrames
	} else if e.ShieldDown > 0 {
		e.ShieldDown--
	}
	if e.StateTimer > 0 {
		e.StateTimer--
	}
	b.clearAIInput()
	target := g.campaignEnemyTarget(e)
	pull := false
	if target != nil && b.BlindTime <= 0 {
		dx := target.X - b.X
		dy := target.Y - b.Y
		dir := 1
		if dx < 0 {
			dir = -1
		}
		grounded := b.JumpNum == 2
		ahead := pointHitsPlatforms(arena.Platforms, b.X+float64(dir)*30, b.Y+16)
		adx := math.Abs(dx)
		facing := shieldBearerFace(e, dir)
		switch {
		case !facing:
			// Still turning the shield around: stand.
		case adx <= shieldBearerShoveRange && math.Abs(dy) < 45:
			if e.ShoveTimer == 0 && grounded {
				g.shieldBearerShove(e, target)
			}
		case ahead || !grounded:
			// Advance behind the shield. Never step off an edge on purpose.
			if dir < 0 {
				b.AILeft = true
			} else {
				b.AIRight = true
			}
		}
		if grounded && dy < -70 && adx < 150 && e.Frame%40 == 0 {
			b.AIUp = true // climb toward a target on a higher platform
		}
		if grounded && dy > 70 && adx < 90 && b.Y < arena.LowestY {
			b.AIDown = true
		}
		inLine := math.Abs(dy) < 45
		if facing && inLine && adx >= shieldBearerShotMin && adx <= shieldBearerShotMax && e.ShotTimer == 0 && e.ShieldDown <= 0 {
			pull = true
			e.ShotTimer = e.ShotDelay
		}
	}
	fired := false
	fire := func() {
		fired = true
		g.fireCampaignEnemyWeapon(e, pull)
	}
	b.UpdateWithFire(arena, nil, nil, fire)
	if !fired && b.Active {
		g.fireCampaignEnemyWeapon(e, false)
	}
	if b.Active && b.Weapon.Bullets <= 0 && b.Weapon.Def.Number >= 9 {
		b.EquipWeapon(b.DefaultWeapon)
	}
	// Ring-outs use the ordinary player bounds inside UpdateWithFire (Kill).
	if b.Active {
		g.advanceBodyVisuals(b)
	}
}

// advanceBodyVisuals ticks the source stick-figure timelines of an enemy body
// exactly like the player loop does.
func (g *Game) advanceBodyVisuals(b *Player) {
	if g.assets == nil {
		b.updateSourceVisualState()
		return
	}
	b.advanceLegTimelines(len(g.assets.Leg1Timeline), len(g.assets.Leg2Timeline))
	b.advanceHeadTimeline(g.assets.HeadTimeline)
	b.updateSourceVisualState()
}

// shieldBearerTurnFrames: turning the heavy shield around takes a moment,
// which is the window for flanking shots.
const shieldBearerTurnFrames = 26

// shieldBearerFace turns toward dir after the turn delay.
func shieldBearerFace(e *CampaignEnemy, dir int) bool {
	if e.Body.Facing == dir {
		e.State = 0
		return true
	}
	e.State++
	if e.State >= shieldBearerTurnFrames {
		e.State = 0
		e.Body.Facing = dir
		return true
	}
	return false
}

// shieldBearerShove is a short shield bash. It runs inside the players'
// knockback window, so HEAVY / ANCHOR / SHIELD apply to it.
func (g *Game) shieldBearerShove(e *CampaignEnemy, target *Player) {
	b := &e.Body
	f := float64(b.Facing)
	if f == 0 {
		f = 1
	}
	push := shieldBearerShovePush
	if target.PerkNumber == 6 || target.ShieldTime > 0 {
		push *= 0.4
	}
	target.VX += push * f
	target.VY -= shieldBearerShoveLift
	target.HitNumber++
	target.HitTimer = 0
	target.LastHitBy = b.ID
	target.HitByGrenade = false
	e.ShoveTimer = shieldBearerShoveCooldown
	e.StateTimer = 12 // bash animation
	g.playRandomHitSound()
}
