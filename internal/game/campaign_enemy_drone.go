package game

import "math"

// DRONE: a small quad-rotor that hovers (no gravity, no platform collision),
// keeps its distance at the target's chest height and taps a pistol. It can
// not fall off a platform, but it is light: bullets, shotguns, blasts and
// especially shock clouds (which stall its rotors) throw it beyond the arena,
// where it is destroyed.

const (
	droneHitboxScale    = 0.45 // Player.Hitbox: 22 x 31 px
	droneMuzzleScale    = 0.42 // bullets leave at the drone centre
	droneKnockbackScale = 1.15 // a little lighter than a player
	droneDrag           = 0.92 // per-frame velocity retention (air drag)
	droneMaxAccel       = 0.45
	droneStallAccel     = 0.0
	droneKeepDistance   = 215.0
	droneShotRange      = 430.0
	droneShotHeight     = 34.0 // max vertical misalignment to pull the trigger
	droneWallClearance  = 34.0
	droneStallFrames    = 50
	droneStallGravity   = 0.42
	droneMaxSpeed       = 6.0
)

// droneCenter is the visual / physical centre of the drone.
func droneCenter(b *Player) (float64, float64) {
	return b.X, b.Y - 70*droneHitboxScale/2
}

func (g *Game) droneInShockCloud(b *Player) bool {
	cx, cy := droneCenter(b)
	for i := range g.areaEffects {
		a := &g.areaEffects[i]
		if a.Dead || a.Kind != areaEffectGas {
			continue
		}
		if math.Hypot(cx-a.X, cy-a.Y) < a.Radius {
			return true
		}
	}
	return false
}

func clampAbs(v, limit float64) float64 {
	if v > limit {
		return limit
	}
	if v < -limit {
		return -limit
	}
	return v
}

func (g *Game) updateDrone(e *CampaignEnemy, arena Map) {
	b := &e.Body
	if !b.Active {
		return
	}
	// Timers a player would tick in its own update.
	if b.BlindTime > 0 {
		b.BlindTime--
		if b.BlindTime == 0 {
			clearBlind(b)
		}
	}
	b.TearGasTime = 0 // no lungs: tear gas does nothing to a drone
	if b.HitTimer <= 40 {
		b.HitTimer++
	} else {
		b.HitNumber = 0
	}
	if b.FireContactCooldown > 0 {
		b.FireContactCooldown--
	}
	// Drones have no panic (no input to take over): fire only scorches them
	// briefly, so the burn is capped and its jolts rattle the rotors.
	if b.BurnTime > droneBurnMaxFrames {
		b.BurnTime = droneBurnMaxFrames
	}
	b.BurnPanicTime = 0
	b.updateBurning()
	b.advanceInstagibTimeline()
	if b.KillSelf {
		b.KillSelf = false
		g.destroyCampaignEnemy(e)
		return
	}
	if g.droneInShockCloud(b) {
		// A shock cloud shorts the rotors: the drone drops and drifts.
		e.Stall = droneStallFrames
	} else if e.Stall > 0 {
		e.Stall--
	}

	minX, maxX, top, _ := campaignArenaBounds(arena)
	cx, cy := droneCenter(b)
	target := g.campaignEnemyTarget(e)

	desiredX, desiredY := e.HomeX, e.HomeY
	if target != nil {
		side := 1.0
		if b.X < target.X {
			side = -1
		}
		desiredX = target.X + side*droneKeepDistance
		if desiredX < minX+30 || desiredX > maxX-30 {
			// Never park beyond an edge: swap to the other side of the target.
			desiredX = target.X - side*droneKeepDistance
		}
		// Bullets leave at the drone centre; line it up with the target chest.
		desiredY = target.Y - 35 + 70*droneHitboxScale/2
		e.HomeX, e.HomeY = desiredX, math.Min(desiredY, top-10)
	} else {
		// Patrol: drift slowly across the home point.
		desiredX += math.Sin(float64(e.Frame)*0.012+float64(b.ID)) * 120
	}
	desiredX = math.Max(minX+20, math.Min(maxX-20, desiredX))
	desiredY = math.Max(-60, desiredY)
	// Gentle bobbing keeps it alive-looking and makes it a moving target.
	desiredY += math.Sin(float64(e.Frame)*0.08+float64(b.ID)*1.7) * 7

	accel := droneMaxAccel
	if b.BlindTime > 0 {
		// Blinded optics: hold position and wobble.
		desiredX, desiredY = b.X+math.Sin(float64(e.Frame)*0.2)*30, b.Y
		accel *= 0.5
	}
	ax, ay := 0.0, 0.0
	if e.Stall > 0 {
		accel = droneStallAccel
		ay += droneStallGravity
	} else {
		ax = clampAbs((desiredX-b.X)*0.012-b.VX*0.10, accel)
		ay = clampAbs((desiredY-b.Y)*0.014-b.VY*0.10, accel)
		// Wall avoidance: keep the rotor disc clear of every platform.
		for _, r := range arena.Platforms {
			px := math.Max(r.X, math.Min(cx, r.X+r.W))
			py := math.Max(r.Y, math.Min(cy, r.Y+r.H))
			dx, dy := cx-px, cy-py
			d := math.Hypot(dx, dy)
			if d >= droneWallClearance {
				continue
			}
			if d < 0.001 {
				dx, dy, d = 0, -1, 1 // inside a platform: climb out
			}
			push := (droneWallClearance - d) * 0.05
			ax += dx / d * push
			ay += dy / d * push
		}
	}
	// Steering never exceeds a cruise speed, but knockback can (and steering
	// may still brake it).
	if math.Abs(b.VX) < droneMaxSpeed {
		b.VX = clampAbs(b.VX+ax, droneMaxSpeed)
	} else if b.VX*ax < 0 {
		b.VX += ax
	}
	b.VY += ay
	b.X += b.VX
	b.Y += b.VY
	b.VX *= droneDrag
	b.VY *= droneDrag
	e.Tilt += (clampAbs(b.VX*0.06, 0.55) - e.Tilt) * 0.25

	if campaignEnemyOutOfBounds(arena, b.X, b.Y) {
		g.destroyCampaignEnemy(e)
		return
	}

	// Face and shoot.
	pull := false
	if target != nil {
		if target.X >= b.X {
			b.Facing = 1
		} else {
			b.Facing = -1
		}
		dx := math.Abs(target.X - b.X)
		muzzleY := b.Y - 38*droneMuzzleScale
		aligned := math.Abs(muzzleY-(target.Y-35)) < droneShotHeight
		if e.Stall == 0 && b.BlindTime <= 0 && aligned && dx < droneShotRange && e.ShotTimer == 0 {
			pull = true
			e.ShotTimer = e.ShotDelay
		}
	}
	g.fireCampaignEnemyWeapon(e, pull)
	if b.Weapon.Bullets <= 0 && b.Weapon.Def.Number >= 9 {
		b.EquipWeapon(b.DefaultWeapon)
	}
}
