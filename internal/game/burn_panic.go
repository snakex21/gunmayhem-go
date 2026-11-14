package game

import "math"

// BURNING PANIC
//
// Fire is the "loss of control" status, distinct from the gases (tear gas =
// slow + spread, shock gas = steady outward push):
//
//   - Lateral jolts: every burnJoltIntervalMin..Max frames the burning player
//     is jolted left or right. The direction is random but biased away from
//     the fire that lit them (BurnDirection) with burnAwayBias.
//   - Panic runs: short windows where horizontal input is overridden and the
//     character keeps running in its own panic direction (same bias).
//   - Weapon shake: shots fired while burning get burnAimShakeDegrees spread.
//   - Counterplay: each jump pats the fire down (burnPatDownFraction of the
//     remaining burn), limited per ignition and by a cooldown and a floor.
//   - BLAST SHIELD halves burn duration and jolt strength.
//   - Flamethrower ignitions burn for burnFlamethrowerScale of the weapon's
//     IgniteFrames so a stream cannot keep a victim panicking for long.
//
// Everything runs on the host simulation. Randomness comes from a small
// per-player xorshift state (BurnRNG) seeded at ignition from synced game
// state, never from math/rand, so a burn plays out identically for the same
// starting state (tests, replays) and clients only render the synced result.

const (
	// Jolts.
	burnJoltIntervalMin = 10   // frames between jolts (inclusive range)
	burnJoltIntervalMax = 15   //
	burnAwayBias        = 0.65 // chance a jolt / panic run goes away from the fire
	burnJoltPowerScale  = 0.18 // jolt = BurnPower * scale, clamped below
	burnJoltMin         = 3.0  // horizontal jolt impulse (px/frame)
	burnJoltMax         = 4.2  //
	burnJoltLiftScale   = 0.30 // vertical hop per jolt as a fraction of the push

	// Panic runs (horizontal input override).
	burnPanicFirstMin  = 6  // frames after ignition before the first panic run
	burnPanicFirstMax  = 14 //
	burnPanicWindowMin = 6  // panic run length
	burnPanicWindowMax = 10 //
	burnPanicGapMin    = 20 // controllable frames between panic runs
	burnPanicGapMax    = 36 //

	// Weapon shake while burning (degrees, uniform +-).
	burnAimShakeDegrees = 8.0

	// Pat-down counterplay (jumping while burning).
	burnPatDownFraction = 0.20 // remaining burn removed per pat-down
	burnPatDownMax      = 3    // pat-downs per ignition (0.8^3 ~ 51% left)
	burnPatDownCooldown = 18   // frames; a double jump counts once
	burnPatDownFloor    = 12   // a pat-down never cuts the burn below this

	// Sources.
	burnBlastShieldScale  = 0.5 // BLAST SHIELD (perk 6): burn time and jolts
	burnFlamethrowerScale = 0.5 // flamethrower IgniteFrames multiplier

	// Flamethrower stream contact push (combat.go hitPlayerWithFlame). A pulse
	// every 3 ticks (was 7) gives ~2.3-2.6 push/tick so a flamethrower can
	// ring a target out (~1.0 s from 60 px, ~1.8 s from 150 px) while its
	// burn panic stays shorter than a Molotov's.
	flamethrowerContactPush     = 6.0
	flamethrowerContactLift     = 2.2
	flamethrowerContactCooldown = 3

	// Campaign enemies.
	droneBurnMaxFrames = 30 // drones have no panic; they just smoke briefly

	// Bots.
	aiBurnReactSkill    = 0.30 // below this a burning bot just panics
	aiBurnEdgeSkill     = 0.50 // from this skill bots steer away from edges
	aiBurnEdgeLook      = 70.0 // px ahead checked for ground while burning
	aiBurnPatDelaySlow  = 30   // frames before patting down at react skill
	aiBurnPatDelayQuick = 4    // ... and at skill 1
)

// burnRand advances the player's burn xorshift32 state.
func (p *Player) burnRand() uint32 {
	x := p.BurnRNG
	if x == 0 {
		x = 0x6D2B79F5
	}
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	p.BurnRNG = x
	return x
}

// burnRandRange returns an int in [lo, hi].
func (p *Player) burnRandRange(lo, hi int) int {
	if hi <= lo {
		return lo
	}
	return lo + int(p.burnRand()%uint32(hi-lo+1))
}

// burnRandChance is true with probability chance.
func (p *Player) burnRandChance(chance float64) bool {
	return float64(p.burnRand()&0xFFFF)/65536.0 < chance
}

// burnSeed derives a fresh RNG state from synced, deterministic state.
func burnSeed(p *Player) uint32 {
	h := uint32(p.ID)*0x9E3779B1 ^ uint32(p.BurnIgnitions)*0x85EBCA6B ^
		uint32(p.DeathSerial)*0xC2B2AE35 ^ uint32(int32(math.Round(p.X)))*0x27D4EB2F
	h ^= h >> 16
	h *= 0x7FEB352D
	h ^= h >> 15
	h *= 0x846CA68B
	h ^= h >> 16
	if h == 0 {
		h = 0x6D2B79F5
	}
	return h
}

// burnBiasedDir picks left/right, favouring BurnDirection (away from the fire).
func (p *Player) burnBiasedDir() int {
	away := p.BurnDirection
	if away == 0 {
		away = 1
	}
	if p.burnRandChance(burnAwayBias) {
		return away
	}
	return -away
}

// burnFramesFor applies BLAST SHIELD to an ignition length.
func burnFramesFor(p *Player, frames int) int {
	if p != nil && p.PerkNumber == 6 {
		frames = int(math.Round(float64(frames) * burnBlastShieldScale))
		if frames < 1 {
			frames = 1
		}
	}
	return frames
}

// flamethrowerIgniteFrames is the burn a flamethrower particle applies.
func flamethrowerIgniteFrames(igniteFrames int) int {
	if igniteFrames <= 0 {
		return 0
	}
	return int(math.Round(float64(igniteFrames) * burnFlamethrowerScale))
}

// startBurnPanic resets the panic state on a fresh ignition.
func (p *Player) startBurnPanic() {
	p.BurnIgnitions++
	p.BurnRNG = burnSeed(p)
	p.BurnJoltTimer = p.burnRandRange(burnJoltIntervalMin, burnJoltIntervalMax)
	p.BurnPanicTimer = p.burnRandRange(burnPanicFirstMin, burnPanicFirstMax)
	p.BurnPanicTime = 0
	p.BurnPanicDir = 0
	p.BurnPatDowns = 0
	p.BurnPatCooldown = 0
}

// clearBurn removes every burn / panic field (death, respawn, burn end).
func (p *Player) clearBurn() {
	p.BurnTime = 0
	p.BurnOwnerID = 0
	p.BurnDirection = 0
	p.BurnTick = 0
	p.BurnPower = 0
	p.BurnMax = 0
	p.BurnPanicTime = 0
	p.BurnPanicDir = 0
	p.BurnJoltTimer = 0
	p.BurnPanicTimer = 0
	p.BurnPatDowns = 0
	p.BurnPatCooldown = 0
}

// endBurnPanic stops the panic once the fire is out (owner/tick are kept
// like before so the last hit credit is unchanged).
func (p *Player) endBurnPanic() {
	p.BurnTime = 0
	p.BurnPower = 0
	p.BurnMax = 0
	p.BurnPanicTime = 0
	p.BurnPanicDir = 0
	p.BurnJoltTimer = 0
	p.BurnPanicTimer = 0
}

// burnPanicInput overrides horizontal input during a panic run.
func (p *Player) burnPanicInput(left, right bool) (bool, bool) {
	if p.BurnTime <= 0 || p.BurnPanicTime <= 0 || p.BurnPanicDir == 0 {
		return left, right
	}
	return p.BurnPanicDir < 0, p.BurnPanicDir > 0
}

// burnPatDown is called on every executed jump. It reports whether it cut
// the burn.
func (p *Player) burnPatDown() bool {
	if p.BurnTime <= burnPatDownFloor || p.BurnPatCooldown > 0 || p.BurnPatDowns >= burnPatDownMax {
		return false
	}
	remove := int(math.Round(float64(p.BurnTime) * burnPatDownFraction))
	if p.BurnTime-remove < burnPatDownFloor {
		remove = p.BurnTime - burnPatDownFloor
	}
	if remove <= 0 {
		return false
	}
	p.BurnTime -= remove
	p.BurnPatDowns++
	p.BurnPatCooldown = burnPatDownCooldown
	return true
}

// burnJoltStrength is the horizontal impulse of one jolt before ability scaling.
func burnJoltStrength(p *Player) float64 {
	power := math.Max(burnJoltMin, math.Min(burnJoltMax, p.BurnPower*burnJoltPowerScale))
	if p.DamageMulti > 0 {
		power *= p.DamageMulti
	}
	if p.PerkNumber == 6 {
		power *= burnBlastShieldScale
	}
	return power
}

// updateBurning runs once per player update (and per drone update).
func (p *Player) updateBurning() {
	if p == nil || p.BurnTime <= 0 || !p.Active {
		return
	}
	if p.BurnPatCooldown > 0 {
		p.BurnPatCooldown--
	}
	if p.ShieldTime > 0 {
		// Shield powerup smothers the fire quickly and stops the panic.
		p.BurnPanicTime = 0
		p.BurnTime -= 3
		if p.BurnTime <= 0 {
			p.endBurnPanic()
		}
		return
	}
	p.BurnTime--
	if p.BurnTime <= 0 {
		p.endBurnPanic()
		return
	}
	p.BurnTick++

	// Panic run windows.
	if p.BurnPanicTime > 0 {
		p.BurnPanicTime--
		if p.BurnPanicTime == 0 {
			p.BurnPanicTimer = p.burnRandRange(burnPanicGapMin, burnPanicGapMax)
		}
	} else {
		p.BurnPanicTimer--
		if p.BurnPanicTimer <= 0 {
			p.BurnPanicTime = p.burnRandRange(burnPanicWindowMin, burnPanicWindowMax)
			p.BurnPanicDir = p.burnBiasedDir()
		}
	}

	// Lateral jolts.
	p.BurnJoltTimer--
	if p.BurnJoltTimer > 0 {
		return
	}
	p.BurnJoltTimer = p.burnRandRange(burnJoltIntervalMin, burnJoltIntervalMax)
	dir := p.burnBiasedDir()
	scale := abilityBurnTickScale(p)
	push := burnJoltStrength(p) * scale
	p.VX += push * float64(dir)
	p.VY -= push * burnJoltLiftScale
	p.HitNumber++
	p.HitTimer = 0
	p.LastHitBy = p.BurnOwnerID
	p.HitByGrenade = false
}

// burnPanicSnapshot / restoreBurnPanic let the ability SHIELD window undo an
// ignition it blocked.
type burnPanicSnapshot struct {
	max, panicTime, panicDir, joltTimer, panicTimer, patDowns, patCooldown, ignitions int
	rng                                                                              uint32
}

func (p *Player) burnPanicSnapshot() burnPanicSnapshot {
	return burnPanicSnapshot{
		max: p.BurnMax, panicTime: p.BurnPanicTime, panicDir: p.BurnPanicDir, joltTimer: p.BurnJoltTimer,
		panicTimer: p.BurnPanicTimer, patDowns: p.BurnPatDowns, patCooldown: p.BurnPatCooldown,
		ignitions: p.BurnIgnitions, rng: p.BurnRNG,
	}
}

func (p *Player) restoreBurnPanic(s burnPanicSnapshot) {
	p.BurnMax, p.BurnPanicTime, p.BurnPanicDir, p.BurnJoltTimer = s.max, s.panicTime, s.panicDir, s.joltTimer
	p.BurnPanicTimer, p.BurnPatDowns, p.BurnPatCooldown = s.panicTimer, s.patDowns, s.patCooldown
	p.BurnIgnitions, p.BurnRNG = s.ignitions, s.rng
}

// ---------------------------------------------------------------------------
// Visual state (host side, synced through the Visual* / Leg* fields)

// burnFlailPose raises and waves the free hand while burning. Called at the
// end of updateSourceVisualState.
func (p *Player) burnFlailPose() {
	if p.BurnTime <= 0 {
		return
	}
	wave := math.Sin(float64(p.BurnTick) * 0.9)
	p.VisualHand2Y += (-52 + wave*7 - p.VisualHand2Y) / 2
	p.VisualHand2ChildX += (8 + wave*10 - p.VisualHand2ChildX) / 2
	p.VisualBodyY += math.Sin(float64(p.BurnTick)*1.3) * 0.8
}

// burnFastLegs doubles the leg cycle while running on fire.
func (p *Player) burnFastLegs() bool {
	return p.BurnTime > 0
}

// ---------------------------------------------------------------------------
// Bots

// aiHandleBurning: skilled bots pat the fire down by jumping and steer away
// from platform edges while burning. The panic itself still applies to them.
func aiHandleBurning(p *Player, g *Game) {
	if p == nil || g == nil || !p.Active || p.BurnTime <= 0 {
		return
	}
	skill := aiThrowSkill(p, g)
	if skill < aiBurnReactSkill {
		return
	}
	grounded := p.JumpNum == 2
	if skill >= aiBurnEdgeSkill && grounded {
		leftGround := pointHitsPlatforms(g.arena.Platforms, p.X-aiBurnEdgeLook, p.Y+16)
		rightGround := pointHitsPlatforms(g.arena.Platforms, p.X+aiBurnEdgeLook, p.Y+16)
		if !leftGround && rightGround {
			p.AILeft, p.AIRight = false, true
			p.AILockLeft = 0
		} else if !rightGround && leftGround {
			p.AILeft, p.AIRight = true, false
			p.AILockRight = 0
		}
	}
	if aiShouldPatDown(p, skill) {
		p.AIUp = true
	}
}

// aiShouldPatDown: grounded, a pat-down available, and the bot has reacted.
func aiShouldPatDown(p *Player, skill float64) bool {
	if p.JumpNum != 2 || p.JumpHeld || p.BurnPatCooldown > 0 || p.BurnPatDowns >= burnPatDownMax {
		return false
	}
	if p.BurnTime <= burnPatDownFloor+8 {
		return false
	}
	t := clamp01((skill - aiBurnReactSkill) / (1 - aiBurnReactSkill))
	delay := int(math.Round(lerpSkill(aiBurnPatDelaySlow, aiBurnPatDelayQuick, t)))
	return p.BurnTick >= delay
}
