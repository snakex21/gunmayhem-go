package game

import "math"

// GUNSHIP: the helicopter boss of the EXTRA finale (Stage B).
//
// It flies its own motion (no gravity, no platforms, no ring-out) above the
// players and has an HP pool fed by the raw push it takes
// (campaign_enemies_big.go). In a game where every gun shoots sideways,
// players hurt it in four ways:
//   - shoot or bat it when it is low: during a strafing run it flies only
//     heliStrafeAbove px over the platform (a jump puts the muzzle on the
//     hull); at hover height it needs a double-jump shot;
//   - send its rockets back: a gunship rocket batted (bat) or blown (leaf
//     blower) back flies home and hits it for heliRocketDamage HP;
//   - harpoon it: a hook drags it down beside the shooter for
//     heliHookFrames (rope drawn; its attacks pause), within reach of guns
//     and bats; then it is immune to hooks for heliHookCooldown;
//   - any explosive (RPG, grenades) hitting the hull counts its full push.
//
// Phases (HP thresholds 2/3 and 1/3, ticks on the boss bar):
//  1. strafing runs: a red line on the target's platform for
//     heliStrafeWarn ticks, then a pass with the nose gun firing rounds
//     down along it (they stop on platforms: hide under one);
//  2. adds rocket salvos: every impact point gets a ground marker before
//     the rocket even leaves the pod;
//  3. drops drones and shield-bearers (at most heliMaxAdds alive and
//     heliMaxDrops in all), faster runs, bigger salvos.
//
// At zero HP it spins down trailing smoke and explodes on impact.

const (
	heliHitW           = 128.0
	heliHitH           = 54.0
	heliKnockbackScale = 0.15
	heliHoverAbove     = 200.0 // hull bottom above the target's feet
	heliStrafeAbove    = 112.0 // hull bottom above the strafed platform
	heliHookedAbove    = 30.0  // hull bottom above a harpoon shooter's feet
	heliHookedSide     = 110.0
	heliMaxSpeed       = 5.5
	heliStrafeSpeed    = 7.0
	heliStrafeSpeedP3  = 8.5
	heliAccel          = 0.55
	heliDrag           = 0.93
	heliStrafeWarn     = 50
	heliStrafeHalf     = 260.0
	heliRoundEvery     = 3
	heliRoundPush      = 7.0
	heliRoundVX        = 6.0
	heliRoundVY        = 15.0
	heliSalvoWind      = 40
	heliSalvoGap       = 9
	heliMaxSalvo       = 4
	heliRocketSpeed    = 7.0
	heliRocketRadius   = 120.0
	heliRocketPower    = 26.0
	heliRocketPush     = 24.0
	heliRocketDamage   = 7.0
	heliReturnSpeed    = 13.0
	heliHookFrames     = 90
	heliHookCooldown   = 120
	heliDropHover      = 34
	heliMaxAdds        = 3
	heliMaxDrops       = 6
	heliCrashFrames    = 80
	heliDropDroneHP    = 6
)

const (
	heliStateEnter = iota
	heliStateHover
	heliStateStrafeLine
	heliStateStrafe
	heliStateSalvoWind
	heliStateSalvo
	heliStateDrop
	heliStateCrash
)

// heliHoverFrames is the pause between attacks per phase.
var heliHoverFrames = [3]int{75, 55, 40}

// heliRocketsPerPhase is the salvo size per phase (phase 1 has none).
var heliRocketsPerPhase = [3]int{0, 3, 4}

// heliCenter is the middle of the hull (world).
func heliCenter(b *Player) (float64, float64) {
	return b.X, b.Y - heliHitH/2
}

func setupHeliBody(body *Player, e *CampaignEnemy, arena Map, side int) {
	minX, maxX, top, _ := campaignArenaBounds(arena)
	body.KnockbackScale = heliKnockbackScale
	body.hitW, body.hitH = heliHitW, heliHitH
	body.Grounded = false
	if side < 0 {
		body.X = minX - 220
	} else {
		body.X = maxX + 220
	}
	body.Y = top - 150
	body.Facing = -side
	e.HomeX = (minX+maxX)/2 + float64(side)*(maxX-minX)*0.2
	e.HomeY = top + heliHoverAbove
	e.State = heliStateEnter
}

// heliPhaseFor: 0 above 2/3 HP, 1 above 1/3, else 2.
func heliPhaseFor(hp, maxHP int) int {
	switch {
	case maxHP <= 0:
		return 0
	case hp*3 <= maxHP:
		return 2
	case hp*3 <= maxHP*2:
		return 1
	}
	return 0
}

var heliPhaseBanners = [3]string{"", "ROCKET PODS ARMED", "MAYDAY - DEPLOYING TROOPS"}

func (g *Game) updateHeli(e *CampaignEnemy, arena Map) {
	b := &e.Body
	if !b.Active {
		return
	}
	// Timers a player would tick in its own update. Fire and gas cannot
	// reach the armored cabin.
	if b.BlindTime > 0 {
		b.BlindTime--
		if b.BlindTime == 0 {
			clearBlind(b)
		}
	}
	b.TearGasTime, b.BurnTime, b.BurnPanicTime = 0, 0, 0
	if b.HitTimer <= 40 {
		b.HitTimer++
	} else {
		b.HitNumber = 0
	}
	b.KillSelf = false
	if e.hookCD > 0 {
		e.hookCD--
	}
	for want := heliPhaseFor(e.HP, e.MaxHP); e.Phase < want && e.State != heliStateCrash; {
		e.Phase++
		g.bossBanner = b.Name + ": " + heliPhaseBanners[e.Phase]
		g.bossBannerFrame = bossBannerFrames
		g.playSourceSFX("emp.wav", true)
	}
	if e.State == heliStateCrash {
		g.updateHeliCrash(e, arena)
		return
	}
	b.tickArsenal2Timers()
	g.updateHeliHook(e)

	minX, maxX, top, bottom := campaignArenaBounds(arena)
	target := g.campaignEnemyTarget(e)
	if target != nil {
		e.HomeY += (target.Y - e.HomeY) * 0.05
	}
	hoverY := math.Max(top-240, math.Min(bottom-160, e.HomeY-heliHoverAbove))
	desiredX, desiredY := e.HomeX, hoverY
	maxSpeed := heliMaxSpeed

	if e.Hooked > 0 {
		if shooter := g.playerByID(e.hookBy); shooter != nil {
			side := sign(b.X - shooter.X)
			if side == 0 {
				side = 1
			}
			desiredX, desiredY = shooter.X+side*heliHookedSide, shooter.Y-heliHookedAbove
			maxSpeed = heliStrafeSpeed
		}
	} else {
		switch e.State {
		case heliStateEnter:
			if math.Abs(b.X-e.HomeX) < 50 {
				e.State, e.StateTimer = heliStateHover, heliHoverFrames[0]
			}
		case heliStateHover:
			desiredY += math.Sin(float64(e.Frame)*0.07) * 8
			e.StateTimer--
			if e.StateTimer <= 0 {
				g.heliNextAttack(e, target, arena)
			}
		case heliStateStrafeLine:
			dir := float64(b.Facing)
			desiredX, desiredY = e.AimX0-dir*60, e.AimY-heliStrafeAbove
			e.StateTimer--
			if (e.StateTimer <= 0 && math.Abs(b.X-desiredX) < 60) || e.StateTimer < -90 {
				e.State, e.StateTimer = heliStateStrafe, 0
				g.playSourceSFX("jetpack.wav", true)
			}
		case heliStateStrafe:
			dir := float64(b.Facing)
			desiredX, desiredY = e.AimX1+dir*120, e.AimY-heliStrafeAbove
			maxSpeed = heliStrafeSpeed
			if e.Phase >= 2 {
				maxSpeed = heliStrafeSpeedP3
			}
			e.StateTimer++
			ahead := (b.X - e.AimX0) * dir
			if ahead > -20 && (b.X-e.AimX1)*dir < 0 && e.StateTimer%heliRoundEvery == 0 && b.BlindTime <= 0 {
				g.heliFireRound(e)
			}
			if (b.X-e.AimX1)*dir > 50 || e.StateTimer > 200 {
				g.heliEndAttack(e)
			}
		case heliStateDrop:
			desiredX, desiredY = e.AimX0, hoverY+40
			if math.Abs(b.X-e.AimX0) < 30 {
				e.StateTimer++
			}
			if e.StateTimer >= heliDropHover {
				g.heliDropAdd(e, arena)
				g.heliEndAttack(e)
			}
		}
	}
	// A salvo keeps its timing even while hooked: its markers are down.
	if e.State == heliStateSalvoWind {
		if e.Hooked == 0 {
			desiredX, desiredY = e.HomeX, e.AimY
		}
		e.StateTimer--
		if e.StateTimer <= 0 {
			e.State, e.StateTimer = heliStateSalvo, 0
		}
	}
	if e.State == heliStateSalvo {
		if e.Hooked == 0 {
			desiredX, desiredY = e.HomeX, e.AimY
		}
		e.StateTimer++
		if e.StateTimer%heliSalvoGap == 1 {
			g.heliLaunchRocket(e)
		}
		if e.salvo <= 0 {
			g.heliEndAttack(e)
		}
	}

	// Steering (like the drone, with a heavier hull).
	ax := clampAbs((desiredX-b.X)*0.02-b.VX*0.12, heliAccel)
	ay := clampAbs((desiredY-b.Y)*0.025-b.VY*0.14, heliAccel)
	if math.Abs(b.VX) < maxSpeed {
		b.VX = clampAbs(b.VX+ax, maxSpeed)
	} else if b.VX*ax < 0 {
		b.VX += ax
	}
	b.VY = clampAbs(b.VY+ay, maxSpeed)
	b.X += b.VX
	b.Y += b.VY
	b.VX *= heliDrag
	b.VY *= heliDrag
	if e.State != heliStateEnter {
		b.X = math.Max(minX-60, math.Min(maxX+60, b.X))
	}
	b.Y = math.Max(-400, math.Min(bottom-60, b.Y))
	e.Tilt += (clampAbs(b.VX*0.05, 0.35) - e.Tilt) * 0.2
	if e.State != heliStateStrafe && e.State != heliStateStrafeLine && target != nil {
		if target.X >= b.X {
			b.Facing = 1
		} else {
			b.Facing = -1
		}
	}
	e.ShotTimer++
	if e.ShotTimer%45 == 0 {
		g.playSourceSFXGain("jetpack.wav", false, 0.35) // rotor thump
	}
}

// updateHeliHook starts or keeps the harpoon drag.
func (g *Game) updateHeliHook(e *CampaignEnemy) {
	b := &e.Body
	if b.HarpoonTether > 0 && e.hookPrev == 0 && e.Hooked == 0 && e.hookCD == 0 && e.State != heliStateEnter {
		e.Hooked = heliHookFrames
		e.hookBy = b.HarpoonTetherID
		// A hook breaks off a strafing run or a drop.
		switch e.State {
		case heliStateStrafeLine, heliStateStrafe, heliStateDrop:
			g.heliEndAttack(e)
		}
	}
	if e.Hooked > 0 {
		shooter := g.playerByID(e.hookBy)
		if shooter == nil || !shooter.Active || b.HarpoonTetherID != e.hookBy {
			e.Hooked = 1 // rope cut (the shooter died)
		}
		e.Hooked--
		if e.Hooked > 0 {
			b.HarpoonTether, b.HarpoonTetherID = harpoonTetherFrames, e.hookBy
		} else {
			e.hookCD = heliHookCooldown
		}
	}
	e.hookPrev = b.HarpoonTether
}

func (g *Game) heliEndAttack(e *CampaignEnemy) {
	e.State = heliStateHover
	e.StateTimer = heliHoverFrames[min(e.Phase, 2)]
	e.salvo = 0
	e.AimX0, e.AimX1 = 0, 0
}

// heliNextAttack cycles the phase's attack pattern.
func (g *Game) heliNextAttack(e *CampaignEnemy, target *Player, arena Map) {
	b := &e.Body
	minX, maxX, _, _ := campaignArenaBounds(arena)
	if target == nil {
		e.StateTimer = 30
		return
	}
	// The next hover spot: the other side of the target.
	side := 1.0
	if e.cycle%2 == 1 {
		side = -1
	}
	nextHome := math.Max(minX+70, math.Min(maxX-70, target.X+side*170))
	pattern := []int{heliStateStrafeLine}
	switch e.Phase {
	case 1:
		pattern = []int{heliStateStrafeLine, heliStateSalvoWind}
	case 2:
		pattern = []int{heliStateDrop, heliStateSalvoWind, heliStateStrafeLine}
	}
	next := pattern[e.cycle%len(pattern)]
	e.cycle++
	if next == heliStateDrop && !g.heliCanDrop(e) {
		next = heliStateSalvoWind
	}
	if next != heliStateSalvoWind {
		e.HomeX = nextHome // a salvo is fired from where it was planned
	}
	switch next {
	case heliStateStrafeLine:
		py := surfaceUnder(arena, target.X, target.Y)
		dir := 1.0
		if b.X > target.X {
			dir = -1
		}
		e.AimX0 = math.Max(minX, math.Min(maxX, target.X-dir*heliStrafeHalf))
		e.AimX1 = math.Max(minX, math.Min(maxX, target.X+dir*heliStrafeHalf))
		e.AimY = py
		b.Facing = int(dir)
		e.State, e.StateTimer = heliStateStrafeLine, heliStrafeWarn
	case heliStateSalvoWind:
		g.heliPlanSalvo(e, arena)
	case heliStateDrop:
		x := target.X - sign(b.X-target.X)*160
		if sign(b.X-target.X) == 0 {
			x = target.X + 160
		}
		x = math.Max(minX+40, math.Min(maxX-40, x))
		if !platformAtX(arena, x) {
			x = target.X
		}
		e.AimX0 = x
		e.State, e.StateTimer = heliStateDrop, 0
	}
}

// surfaceUnder is the top of the platform under (x, y) (the first one at or
// below y), or y itself over a gap.
func surfaceUnder(arena Map, x, y float64) float64 {
	best := math.Inf(1)
	for _, r := range arena.Platforms {
		if r.ContainsX(x) && r.Y >= y-6 && r.Y < best {
			best = r.Y
		}
	}
	if math.IsInf(best, 1) {
		return y
	}
	return best
}

func platformAtX(arena Map, x float64) bool {
	for _, r := range arena.Platforms {
		if r.ContainsX(x) {
			return true
		}
	}
	return false
}

// heliFireRound: one nose-gun round down and ahead along the run.
func (g *Game) heliFireRound(e *CampaignEnemy) {
	b := &e.Body
	dir := float64(b.Facing)
	if dir == 0 {
		dir = 1
	}
	x, y := b.X+dir*48, b.Y-6
	vx, vy := dir*heliRoundVX+b.VX*0.5, heliRoundVY
	g.bullets = append(g.bullets, Bullet{
		Kind: BulletNormal, X: x, Y: y, VX: vx, VY: vy,
		Rotation:  math.Atan2(vy, vx) * 180 / math.Pi,
		Firepower: heliRoundPush, OwnerID: b.ID, Team: campaignEnemyTeam, Facing: int(dir),
		Time: 1, Alpha: 1, TrailWidth: 20, TrailFrame: 1, ShooterID: b.ID, bossShot: bossShotRound,
	})
	appendMuzzleFlash(&g.flashes, x, y, int(dir))
	if e.StateTimer%(2*heliRoundEvery) == 0 {
		g.playSourceSFX("lmg.wav", false)
	}
}

// heliPlanSalvo picks the impact points and marks them on the ground before
// any rocket leaves the pods.
func (g *Game) heliPlanSalvo(e *CampaignEnemy, arena Map) {
	b := &e.Body
	n := heliRocketsPerPhase[min(e.Phase, 2)]
	if n <= 0 {
		g.heliEndAttack(e)
		return
	}
	minX, maxX, _, _ := campaignArenaBounds(arena)
	var aims []*Player
	for _, p := range g.players {
		if p != nil && p.Active && p.Team != b.Team && g.aiCanDetect(b, p) {
			aims = append(aims, p)
		}
	}
	if len(aims) == 0 {
		g.heliEndAttack(e)
		return
	}
	offsets := [heliMaxSalvo]float64{0, -75, 75, 150}
	e.HomeX, e.AimY = b.X, b.Y // hold position through the salvo
	for i := 0; i < n; i++ {
		t := aims[i%len(aims)]
		off := offsets[i/len(aims)%heliMaxSalvo]
		if i%2 == 1 {
			off = -off
		}
		tx := math.Max(minX+10, math.Min(maxX-10, t.X+off))
		lx, ly := heliPod(b, i)
		ix, iy := segmentImpact(arena, lx, ly, tx, hazardLandingY(arena, tx))
		e.rocketPts[i] = [2]float64{ix, iy}
		flight := int(math.Ceil(math.Hypot(ix-lx, iy-ly) / heliRocketSpeed))
		launch := heliSalvoWind + i*heliSalvoGap + 1
		life := launch + flight + campaignStrikeLinger
		g.extraHazardMarkers = append(g.extraHazardMarkers, HazardMarker{
			Kind: hazardMarkerStrike, X: ix, Y: iy, Life: life, MaxLife: life, Drop: flight + campaignStrikeLinger,
		})
	}
	e.salvo = n
	e.burst = 0 // index of the next rocket
	e.State, e.StateTimer = heliStateSalvoWind, heliSalvoWind
}

// segmentImpact walks from (x0, y0) toward (x1, y1) and returns the first
// point inside a platform (the rocket's real impact), else the end point.
func segmentImpact(arena Map, x0, y0, x1, y1 float64) (float64, float64) {
	d := math.Hypot(x1-x0, y1-y0)
	steps := int(d/4) + 1
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x, y := x0+(x1-x0)*t, y0+(y1-y0)*t
		for _, r := range arena.Platforms {
			if x >= r.X && x <= r.X+r.W && y >= r.Y && y <= r.Y+r.H {
				return x, r.Y
			}
		}
	}
	return x1, y1
}

// heliPod is the launch point of rocket i (pods alternate sides).
func heliPod(b *Player, i int) (float64, float64) {
	side := 1.0
	if i%2 == 1 {
		side = -1
	}
	return b.X + side*30, b.Y - 4
}

// heliLaunchRocket fires the next planned rocket from alternating pods.
func (g *Game) heliLaunchRocket(e *CampaignEnemy) {
	b := &e.Body
	i := e.burst
	if i < 0 || i >= heliMaxSalvo || e.salvo <= 0 {
		e.salvo = 0
		return
	}
	lx, ly := heliPod(b, i)
	tx, ty := e.rocketPts[i][0], e.rocketPts[i][1]
	dx, dy := tx-lx, ty-ly
	d := math.Hypot(dx, dy)
	if d < 1 {
		d = 1
	}
	vx, vy := dx/d*heliRocketSpeed, dy/d*heliRocketSpeed
	facing := 1
	if vx < 0 {
		facing = -1
	}
	g.bullets = append(g.bullets, Bullet{
		Kind: BulletNormal, X: lx, Y: ly, VX: vx, VY: vy,
		Firepower: heliRocketPush, OwnerID: b.ID, Team: campaignEnemyTeam, Facing: facing,
		Time: 1, Alpha: 1, TrailWidth: 20, TrailFrame: 1, ShooterID: b.ID,
		DevelopedProjectile: DevelopedProjectileRocket, ExplosionRadius: heliRocketRadius,
		ExplosionPower: heliRocketPower, Lifetime: 240, GuidanceLost: true,
		bossShot: bossShotRocket, bossFrom: b.ID,
	})
	e.burst++
	e.salvo--
	g.playSourceSFX("dev_launcher_rpg.wav", false)
}

// heliCanDrop: room for another add (alive cap and total cap).
func (g *Game) heliCanDrop(e *CampaignEnemy) bool {
	if e.drops >= heliMaxDrops {
		return false
	}
	return g.heliAddsAlive(e) < heliMaxAdds
}

func (g *Game) heliAddsAlive(e *CampaignEnemy) int {
	n := 0
	for _, o := range g.campaignEnemies {
		if o != nil && o.alive() && o.summoner == e.Body.ID {
			n++
		}
	}
	return n
}

// heliDropAdd lowers a shield-bearer (even drops) or releases a drone.
func (g *Game) heliDropAdd(e *CampaignEnemy, arena Map) {
	if !g.heliCanDrop(e) {
		return
	}
	b := &e.Body
	kind := campaignEnemyShieldBearer
	unit := extraCampaignUnit{Kind: kind}
	if e.drops%2 == 1 || !platformAtX(arena, b.X) {
		unit = extraCampaignUnit{Kind: campaignEnemyDrone, HP: heliDropDroneHP}
	}
	add := g.spawnCampaignEnemy(unit, -b.Facing, arena)
	add.summoner = b.ID
	add.Body.X, add.Body.Y = b.X, b.Y+12
	add.Body.VX, add.Body.VY = 0, 0
	add.HomeX, add.HomeY = b.X, b.Y
	e.drops++
	g.playSourceSFX("bolt2.wav", false)
}

// startHeliCrash: zero HP. It keeps counting as standing until it hits the
// ground (the win waits for the explosion).
func (g *Game) startHeliCrash(e *CampaignEnemy) {
	b := &e.Body
	e.State, e.StateTimer = heliStateCrash, 0
	e.Hooked, e.salvo = 0, 0
	b.HarpoonTether, b.HarpoonTetherID = 0, 0
	dir := sign(b.X - (g.arena.CrateMinX+g.arena.CrateMaxX)/2)
	if dir == 0 {
		dir = 1
	}
	b.VX, b.VY = dir*2.2, -1.5
	g.bossBanner = b.Name + ": GOING DOWN!"
	g.bossBannerFrame = bossBannerFrames
	g.playSourceSFX("emp.wav", true)
}

func (g *Game) updateHeliCrash(e *CampaignEnemy, arena Map) {
	b := &e.Body
	_, _, _, bottom := campaignArenaBounds(arena)
	e.StateTimer++
	b.VY += 0.2
	b.VX *= 0.995
	b.X += b.VX
	b.Y += b.VY
	e.Tilt += 0.11 * sign(b.VX)
	cx, cy := heliCenter(b)
	if e.StateTimer%4 == 0 {
		spawnMolotovSparks(&g.dynamiteFX, cx-sign(b.VX)*40, cy, 2)
	}
	if e.StateTimer%16 == 8 {
		g.explosions = append(g.explosions, Explosion{X: cx + float64(e.StateTimer%3-1)*30, Y: cy, Life: 12})
		g.playRandomExplosionSound()
	}
	hitGround := false
	if e.StateTimer > 12 {
		for _, r := range arena.Platforms {
			if b.X >= r.X && b.X <= r.X+r.W && b.Y >= r.Y && b.Y-b.VY <= r.Y+4 {
				hitGround = true
				b.Y = r.Y
				break
			}
		}
	}
	if hitGround || e.StateTimer >= heliCrashFrames || b.Y > bottom+120 {
		g.finishBigEnemy(e)
	}
}
