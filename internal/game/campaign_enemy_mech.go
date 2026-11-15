package game

import "math"

// MECH: a big bipedal walker (Stage B). It walks the platforms with the
// ordinary player physics (gravity, drop-through, ring-out) and the heavy
// zombie platform graph (no long jumps), shrugs off most push
// (mechKnockbackScale) and has an HP pool fed by the raw push it takes
// (campaign_enemies_big.go). Its back is the weak spot: the exposed reactor
// takes mechBackMult times the push and the damage, and the mech turns
// slowly (mechTurnFrames), so flanking pays. Enough push still rings it out.
//
// Attacks, each with a readable telegraph:
//   - twin machine guns: a red laser at chest height for mechAimFrames,
//     then a burst of mechBurstRounds (jump over it);
//   - stomp: one leg rises and an orange ring grows on the ground for
//     mechStompWind; then everyone standing on that surface within
//     mechStompRadius is thrown away (be in the air to dodge);
//   - shoulder mortar (Rockets units only): pods glow, then two strikes are
//     marked on the ground mechRocketWarn ticks before they fall.
//
// A fresh harpoon hook jams a leg: it cancels the current wind-up.

const (
	mechGun            = 21 // COMPACT SMG feeding the twin barrels
	mechHitW           = 70.0
	mechHitH           = 120.0
	mechDrawScale      = 1.25 // procedural art units -> px (fills the hitbox)
	mechKnockbackScale = 0.12
	mechMoveScale      = 0.34
	mechJumpScale      = 0.92
	mechFirepowerMulti = 0.85
	mechBackMult       = 2.5
	mechBurnMaxFrames  = 30

	mechTurnFrames      = 26
	mechAimFrames       = 26
	mechBurstRounds     = 6
	mechMGRange         = 520.0
	mechMGHeight        = 50.0 // max vertical misalignment to open fire
	mechStompTrigger    = 140.0
	mechStompRadius     = 175.0
	mechStompWind       = 30
	mechStompRecover    = 26
	mechStompPush       = 12.5
	mechStompLift       = 5.5
	mechRocketWind      = 32
	mechRocketWarn      = 50
	mechRocketCooldown  = 210
	mechRocketSpread    = 46.0
	mechAttackCooldown  = 45
	mechMeltdownFrames  = 40
	mechHoldMin         = 140.0
	mechHoldMax         = 330.0
	mechStuckFrames     = 75
	mechMuzzleX         = 40.0
	mechMuzzleY         = 38.0
	mechBarrelSeparateY = 4.0
)

const (
	mechStateIdle = iota
	mechStateTurn
	mechStateAim
	mechStateFire
	mechStateStompWind
	mechStateStompRecover
	mechStateRocketWind
	mechStateMeltdown
)

// setupMechBody configures a freshly spawned mech: it drops in near one
// side of the spawn area.
func setupMechBody(body *Player, e *CampaignEnemy, arena Map, side int) {
	minX, maxX, _, _ := campaignArenaBounds(arena)
	span := arena.SpawnMaxX - arena.SpawnMinX
	center := (arena.SpawnMinX + arena.SpawnMaxX) / 2
	if span <= 0 {
		span = maxX - minX
		center = (minX + maxX) / 2
	}
	body.KnockbackScale = mechKnockbackScale
	body.CampaignMoveScale = mechMoveScale
	body.RuleJumpMultiplier = mechJumpScale
	body.FirepowerMulti = mechFirepowerMulti
	body.EquipWeapon(body.DefaultWeapon)
	body.hitW, body.hitH = mechHitW, mechHitH
	body.HatNumber = 1
	body.X = center + float64(side)*span*0.38
	body.Y = -220
	body.JumpNum = 1
	body.Facing = -side
	e.HomeX = body.X
	e.nav = zombieNavState{cur: -1, lastPlat: -1}
	e.attackCD = mechAttackCooldown
}

// mechBackHit: a push along the mech's facing came from behind.
func mechBackHit(e *CampaignEnemy, dvx float64) bool {
	f := float64(e.Body.Facing)
	if f == 0 {
		f = 1
	}
	return dvx*f > 0.05
}

func (g *Game) mechNavFor(arena Map) *zombieNav {
	if g.mechNav == nil || g.mechNav.mapNumber != arena.Number || len(g.mechNav.plats) != len(arena.Platforms) {
		g.mechNav = buildZombieNav(arena, zombieNavHeavy)
	}
	return g.mechNav
}

// mechWindup: a telegraph that a harpoon hook can cancel.
func mechWindup(state int) bool {
	return state == mechStateAim || state == mechStateStompWind || state == mechStateRocketWind
}

func (g *Game) updateMech(e *CampaignEnemy, arena Map) {
	b := &e.Body
	if !b.Active {
		return
	}
	// A sealed cockpit: no tear gas, no burn panic, short scorching.
	b.TearGasTime = 0
	b.BurnPanicTime = 0
	if b.BurnTime > mechBurnMaxFrames {
		b.BurnTime = mechBurnMaxFrames
	}
	if e.attackCD > 0 {
		e.attackCD--
	}
	if e.salvo > 0 {
		e.salvo-- // mortar reload
	}
	b.clearAIInput()
	if e.State == mechStateMeltdown {
		e.StateTimer++
		b.UpdateWithFire(arena, nil, nil, nil)
		if e.StateTimer%5 == 0 && b.Active {
			spawnMolotovSparks(&g.dynamiteFX, b.X+float64((e.StateTimer*37)%50-25), b.Y-60, 2)
		}
		if b.Active && e.StateTimer >= mechMeltdownFrames {
			g.finishBigEnemy(e)
		}
		return
	}
	// A fresh harpoon hook jams a leg: the wind-up is lost.
	if b.HarpoonTether > 0 && e.hookPrev == 0 && mechWindup(e.State) {
		e.State, e.StateTimer = mechStateIdle, 0
		e.attackCD = mechAttackCooldown
	}
	e.hookPrev = b.HarpoonTether

	target := g.campaignEnemyTarget(e)
	nav := g.mechNavFor(arena)
	grounded := b.JumpNum == 2
	pull := false
	f := float64(b.Facing)
	if f == 0 {
		f = 1
	}

	switch e.State {
	case mechStateTurn:
		e.StateTimer--
		if e.StateTimer <= 0 {
			b.Facing = -b.Facing
			if b.Facing == 0 {
				b.Facing = 1
			}
			e.State = mechStateIdle
		}
	case mechStateAim:
		e.StateTimer--
		e.AimY = b.Y - mechMuzzleY
		e.AimX0 = b.X + f*mechMuzzleX*mechDrawScale
		e.AimX1 = b.X + f*mechMGRange
		if e.StateTimer <= 0 {
			e.State = mechStateFire
			e.burst = mechBurstRounds
			if b.Weapon.Bullets < mechBurstRounds || b.Weapon.Def.Number != b.DefaultWeapon {
				b.EquipWeapon(b.DefaultWeapon)
			}
			b.Weapon.WaitTime = b.Weapon.Def.ROF
		}
	case mechStateFire:
		pull = e.burst > 0 && b.BlindTime <= 0
		if !pull {
			e.State = mechStateIdle
			e.attackCD = mechAttackCooldown
		}
	case mechStateStompWind:
		e.StateTimer++
		e.AimX0, e.AimY = b.X, b.Y
		e.AimX1 = mechStompRadius * math.Min(1, float64(e.StateTimer)/mechStompWind)
		if e.StateTimer >= mechStompWind {
			if grounded {
				g.mechStomp(e)
			}
			e.State, e.StateTimer = mechStateStompRecover, 0
		}
	case mechStateStompRecover:
		e.StateTimer++
		if e.StateTimer >= mechStompRecover {
			e.State = mechStateIdle
			e.attackCD = mechAttackCooldown
		}
	case mechStateRocketWind:
		e.StateTimer++
		if e.StateTimer >= mechRocketWind {
			g.mechLaunchMortar(e, target, arena)
			e.State = mechStateIdle
			e.attackCD = mechAttackCooldown
			e.salvo = mechRocketCooldown
		}
	default:
		g.mechIdle(e, target, nav, grounded)
	}

	fired := false
	fire := func() {
		fired = true
		g.mechFire(e, pull)
	}
	b.UpdateWithFire(arena, nil, nil, fire)
	if !fired && b.Active {
		g.mechFire(e, false)
	}
	if b.Active {
		g.advanceBodyVisuals(b)
	}
}

// mechIdle picks the next attack or walks toward the target.
func (g *Game) mechIdle(e *CampaignEnemy, target *Player, nav *zombieNav, grounded bool) {
	b := &e.Body
	if target == nil {
		e.stuckFrames = 0
		return
	}
	dx := target.X - b.X
	dy := target.Y - b.Y
	adx := math.Abs(dx)
	dir := 1
	if dx < 0 {
		dir = -1
	}
	behind := dir != b.Facing && adx > 24
	samePlat := grounded && nav.platformUnder(b.X, b.Y) >= 0 && nav.platformUnder(b.X, b.Y) == nav.platformNear(target.X, target.Y)
	if grounded && e.attackCD == 0 && b.BlindTime <= 0 {
		switch {
		case adx < mechStompTrigger && math.Abs(dy) < 40:
			e.State, e.StateTimer = mechStateStompWind, 0
			e.AimX0, e.AimY, e.AimX1 = b.X, b.Y, 0
			g.playSourceSFX("jetpack.wav", false)
			return
		case !behind && math.Abs(dy) < mechMGHeight && adx < mechMGRange:
			e.State, e.StateTimer = mechStateAim, mechAimFrames
			return
		case e.Mortar && e.salvo == 0 && (adx > 220 || math.Abs(dy) > 60):
			e.State, e.StateTimer = mechStateRocketWind, 0
			return
		}
	}
	if behind && grounded {
		e.State, e.StateTimer = mechStateTurn, mechTurnFrames
		return
	}
	if samePlat && adx <= mechHoldMax {
		e.stuckFrames = 0
		return // in range: plant and wait for the guns / the stomp
	}
	navSteer(b, &e.nav, e.Frame, target.X, target.Y, nav)
	// Never spin on the spot: a key against the facing becomes a slow turn.
	if grounded && ((b.AILeft && b.Facing > 0) || (b.AIRight && b.Facing < 0)) {
		b.AILeft, b.AIRight, b.AIUp, b.AIDown = false, false, false, false
		e.State, e.StateTimer = mechStateTurn, mechTurnFrames
		return
	}
	g.mechTrackStuck(e)
}

// mechTrackStuck: walking but not moving for a while -> hop and turn.
func (g *Game) mechTrackStuck(e *CampaignEnemy) {
	b := &e.Body
	if !(b.AILeft || b.AIRight) {
		e.stuckFrames = 0
		return
	}
	if math.Abs(b.X-e.stuckX) > 3 {
		e.stuckX = b.X
		e.stuckFrames = 0
		return
	}
	e.stuckFrames++
	if e.stuckFrames >= mechStuckFrames {
		e.stuckFrames = 0
		e.nav = zombieNavState{cur: -1, lastPlat: -1}
		b.AIUp = true
	}
}

// mechFire runs the twin guns for one tick; rounds alternate barrels.
func (g *Game) mechFire(e *CampaignEnemy, pull bool) {
	b := &e.Body
	before := len(g.bullets)
	g.fireCampaignEnemyWeapon(e, pull)
	for i := before; i < len(g.bullets); i++ {
		bl := &g.bullets[i]
		if bl.OwnerID != b.ID {
			continue
		}
		if e.burst%2 == 0 {
			bl.Y -= mechBarrelSeparateY
		} else {
			bl.Y += mechBarrelSeparateY
		}
		e.burst--
	}
}

// mechStomp throws everyone standing on the mech's surface away from it.
// It runs inside the players' knockback window (HEAVY / ANCHOR / SHIELD).
func (g *Game) mechStomp(e *CampaignEnemy) {
	b := &e.Body
	for _, p := range g.players {
		if p == nil || !p.Active || p.Team == b.Team || p.JumpNum != 2 {
			continue
		}
		if math.Abs(p.Y-b.Y) > 14 {
			continue
		}
		dx := p.X - b.X
		if math.Abs(dx) > mechStompRadius {
			continue
		}
		dir := sign(dx)
		if dir == 0 {
			dir = float64(b.Facing)
		}
		push := mechStompPush * (1 - 0.5*math.Abs(dx)/mechStompRadius)
		lift := mechStompLift
		if p.PerkNumber == 6 || p.ShieldTime > 0 {
			push *= 0.4
			lift *= 0.4
		}
		p.VX += dir * push
		p.VY -= lift
		p.HitNumber++
		p.HitTimer = 0
		p.LastHitBy = b.ID
		p.HitByGrenade = false
	}
	for _, s := range []float64{-1, 1} {
		spawnMolotovSparks(&g.dynamiteFX, b.X+s*30, b.Y-4, 3)
	}
	g.explosions = append(g.explosions, Explosion{X: b.X, Y: b.Y - 6, Life: 10})
	g.playSourceSFX("drop2.wav", true)
	g.playRandomHitSound()
}

// mechLaunchMortar marks two strikes around the target.
func (g *Game) mechLaunchMortar(e *CampaignEnemy, target *Player, arena Map) {
	if target == nil {
		return
	}
	b := &e.Body
	minX, maxX, _, _ := campaignArenaBounds(arena)
	dir := 1
	if target.X < b.X {
		dir = -1
	}
	for _, off := range []float64{-mechRocketSpread, mechRocketSpread} {
		x := math.Max(minX+10, math.Min(maxX-10, target.X+off))
		g.queueCampaignStrike(arena, x, target.Y, mechRocketWarn, b.ID, dir)
	}
	g.playSourceSFX("dev_javelin.wav", false)
}
