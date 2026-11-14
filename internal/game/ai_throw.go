package game

import (
	"math"
	"math/rand"
)

// Bot throwables. Neither the original playerAI nor the earlier port ever
// pressed KEYNADE; this is a developed (user-approved) behaviour.
//
// The bot throws exactly like a human: it holds AIGrenade for as many frames
// as the chosen charge needs and then lets go, so updateGrenadeInput spawns,
// counts and syncs the throwable through the normal path (host and online
// identical, clients only see the resulting grenade). Aiming uses the shared
// throw simulator (simulateThrow), i.e. the real flight, bounce and fuse code.
//
// Charge: updateGrenadeInput starts a held throw at power 1 and adds 0.5 on
// every pressed frame (the press frame included), so after k pressed frames
// the release throws with power 1+0.5*k (k = 1..26 -> 1.5..14). Power 1.0
// itself is not reachable by any player.

const (
	aiThrowMinPower     = 1.5
	aiThrowPowerStep    = 0.5
	aiThrowSelfMargin   = 40.0 // effect radius + this around the bot = unsafe
	aiThrowEvalInterval = 10   // frames between throw evaluations
	aiThrowRetryFrames  = 30   // after a candidate was rejected by the act roll
	aiThrowCampRadius   = 45.0
	aiThrowCampFrames   = 150 // 2.5 s standing within aiThrowCampRadius
	aiThrowEdgeDistance = 110.0
	aiThrowPushRange    = 60.0
)

// aiThrowSkill is the bot's throwing skill 0..1. There is no custom bot
// difficulty setting, so it follows the campaign progress (first mission 0,
// last 1), the boss is always 1, and versus/custom bots sit at 0.6.
func aiThrowSkill(p *Player, g *Game) float64 {
	if p != nil && p.AISpecial == sourceAISpecialBoss {
		return 1
	}
	if g == nil || !g.campaignMode {
		return 0.6
	}
	maxLevel := 10
	if g.campaignSetGM2 {
		maxLevel = 16
	} else if g.campaignSetExtra {
		maxLevel = extraCampaignMissionCount
	}
	if maxLevel <= 1 {
		return 1
	}
	return clamp01(float64(g.campaignLevel-1) / float64(maxLevel-1))
}

func clamp01(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}

func lerpSkill(low, high, skill float64) float64 {
	return low + (high-low)*clamp01(skill)
}

// Skill-dependent knobs (spec: 0.15->0.6 act chance, +-1.5->+-0.25 power
// error, target lead off->on, a decreasing minimum score).
func aiThrowActChance(skill float64) float64  { return lerpSkill(0.15, 0.6, skill) }
func aiThrowPowerError(skill float64) float64 { return lerpSkill(1.5, 0.25, skill) }
func aiThrowMinScore(skill float64) float64   { return lerpSkill(0.8, 0.45, skill) }
func aiThrowLeads(skill float64) bool         { return skill >= 0.4 }
func aiThrowCooldownFrames(skill float64) int {
	return int(math.Round(lerpSkill(240, 90, skill)))
}

// aiThrowAllowed covers the hard limits: modes without throwables, nothing
// left (low skill keeps one in reserve), blind, anchored, airborne (only
// skilled bots throw from the air).
func aiThrowAllowed(p *Player, skill float64) bool {
	if p == nil || !p.AI || !p.Active {
		return false
	}
	if p.GameMode == SourceGameModeInstagib || p.GameMode == SourceGameModeGunGame {
		return false
	}
	reserve := 0
	if skill < 0.35 {
		reserve = 1
	}
	if p.Grenades <= reserve || p.BlindTime > 0 || p.anchored() {
		return false
	}
	if p.JumpNum != 2 && skill < 0.75 {
		return false
	}
	return true
}

// aiThrowTarget is one enemy as the bot sees it.
type aiThrowTarget struct {
	P       *Player
	X, Y    float64 // current (or last seen) feet position
	VX      float64
	Hidden  bool // known only from the last seen position (smoke)
	Primary bool
}

// aiThrowPlan is the chosen throw.
type aiThrowPlan struct {
	Power    float64
	Score    float64
	Result   throwSimResult
	TargetID int
}

// aiDecideThrow runs in decideAIController just before aiApplyFlashAvoid.
func aiDecideThrow(p *Player, g *Game) {
	if p == nil || g == nil || !p.AI || !p.Active {
		if p != nil {
			p.AIThrowPower = 0
		}
		return
	}
	arena := g.customRuntimeArena()
	aiThrowFlashFollowUp(p, g, arena)
	aiTrackCamper(p, g)
	if p.AIThrowCooldown > 0 {
		p.AIThrowCooldown--
	}
	skill := aiThrowSkill(p, g)
	if p.AIThrowPower > 0 {
		aiContinueThrow(p, g, arena, skill)
		return
	}
	if p.AIThrowCooldown > 0 || p.AIFlashAvoid > 0 || !aiThrowAllowed(p, skill) {
		return
	}
	if p.AIThrowCheck > 0 {
		p.AIThrowCheck--
		return
	}
	p.AIThrowCheck = aiThrowEvalInterval
	plan, ok := aiBestThrow(p, g, arena, skill)
	if !ok {
		return
	}
	if rand.Float64() >= aiThrowActChance(skill) {
		p.AIThrowCheck = aiThrowRetryFrames
		return
	}
	power := plan.Power + (rand.Float64()*2-1)*aiThrowPowerError(skill)
	power = math.Round(power/aiThrowPowerStep) * aiThrowPowerStep
	p.AIThrowPower = math.Max(aiThrowMinPower, math.Min(grenadeMaxPower, power))
	p.AIThrowTargetID = 0
	if throwableProfileFor(p.ThrowableType).Behavior == throwableBehaviorFlash {
		p.AIThrowTargetID = plan.TargetID
	}
	p.AIGrenade = true
	aiHoldStillForThrow(p)
}

// aiContinueThrow keeps the throw key down until the planned charge is
// reached, then releases. A release that would now catch the bot or a
// teammate keeps charging instead (a longer throw lands further away).
func aiContinueThrow(p *Player, g *Game, arena Map, skill float64) {
	if !p.GrenadeHeld || p.BlindTime > 0 || p.Grenades <= 0 ||
		p.GameMode == SourceGameModeInstagib || p.GameMode == SourceGameModeGunGame {
		// The wind-up was dropped (blinded, respawned) or throwing is off.
		p.AIThrowPower = 0
		return
	}
	p.AIGrenade = true
	if p.GrenadePower+1e-9 >= p.AIThrowPower {
		safe := aiThrowReleaseSafe(p, g, arena, p.X, p.Y, p.Facing, p.GrenadePower)
		if safe || p.GrenadePower >= grenadeMaxPower {
			p.AIGrenade = false
			p.AIThrowPower = 0
			p.AIThrowCooldown = aiThrowCooldownFrames(skill) + rand.Intn(31)
			if p.AIThrowCooldown > 240 {
				p.AIThrowCooldown = 240
			}
			return
		}
	}
	aiHoldStillForThrow(p)
}

// aiHoldStillForThrow stops walking while charging so the release point
// matches the plan. Airborne recovery steering and fire escape are untouched.
func aiHoldStillForThrow(p *Player) {
	if p.JumpNum != 2 || p.BurnTime > 0 {
		return
	}
	p.AILeft, p.AIRight = false, false
	p.AILockLeft, p.AILockRight = 0, 0
}

// aiThrowFlashFollowUp: turn away from the bot's own flashbang right before
// it pops, then push in on the blinded victim.
func aiThrowFlashFollowUp(p *Player, g *Game, arena Map) {
	ownFlashLive := false
	for i := range g.grenades {
		gr := &g.grenades[i]
		profile := throwableProfileFor(gr.Type)
		if gr.Dead || profile.Behavior != throwableBehaviorFlash || gr.OwnerID != p.scoreOwnerID() {
			continue
		}
		ownFlashLive = true
		if gr.Fuse >= profile.Fuse-flashAIWarningFrames && p.AIFlashAvoid == 0 {
			aiTurnAwayFromFlash(arena, p, gr.X, gr.Y-flashPointLift)
		}
	}
	if p.AIThrowTargetID == 0 || ownFlashLive {
		return
	}
	target := g.playerByID(p.AIThrowTargetID)
	if target == nil || !target.Active || target.BlindTime <= 0 || p.BlindTime > 0 {
		p.AIThrowTargetID = 0
		return
	}
	if p.AIFlashAvoid > 0 || p.JumpNum != 2 || math.Abs(target.Y-p.Y) > 60 {
		return
	}
	dir := 1
	if target.X < p.X {
		dir = -1
	}
	if math.Abs(target.X-p.X) > aiThrowPushRange &&
		pointHitsPlatforms(arena.Platforms, p.X+64*float64(dir), p.Y+16) {
		p.AILeft, p.AIRight = dir < 0, dir > 0
		p.AILockLeft, p.AILockRight = 0, 0
	}
	if p.Facing == dir {
		p.AIShoot = true
	}
}

// aiTrackCamper measures how long the current target stays in one spot.
func aiTrackCamper(p *Player, g *Game) {
	target := sourceAITargetPlayer(p, g)
	if target == nil {
		p.AIThrowCampID, p.AIThrowCampFrames = 0, 0
		return
	}
	if target.ID != p.AIThrowCampID || math.Hypot(target.X-p.AIThrowCampX, target.Y-p.AIThrowCampY) > aiThrowCampRadius {
		p.AIThrowCampID = target.ID
		p.AIThrowCampX, p.AIThrowCampY = target.X, target.Y
		p.AIThrowCampFrames = 0
		return
	}
	p.AIThrowCampFrames++
}

func aiThrowIsTeammate(p, other *Player) bool {
	return other != nil && other != p && other.Active &&
		(other.Team == p.Team || other.scoreOwnerID() == p.scoreOwnerID())
}

// aiThrowTargets lists the enemies the bot knows about: visible ones and the
// one it is searching for (last seen position, e.g. inside smoke).
func aiThrowTargets(p *Player, g *Game) []aiThrowTarget {
	primary := sourceAITargetPlayer(p, g)
	var out []aiThrowTarget
	for _, other := range g.players {
		if other == nil || other == p || !other.Active || other.Team == p.Team || other.scoreOwnerID() == p.scoreOwnerID() {
			continue
		}
		if g.aiCanDetect(p, other) {
			out = append(out, aiThrowTarget{P: other, X: other.X, Y: other.Y, VX: other.VX, Primary: other == primary})
			continue
		}
		if p.AISearchTime > 0 && p.AISearchPlayerID == other.ID && other.InvisibleTime <= 0 {
			out = append(out, aiThrowTarget{P: other, X: p.AISearchX, Y: p.AISearchY, Hidden: true, Primary: primary == nil})
		}
	}
	return out
}

// aiThrowLeadDistance is how far a player drifting at vx travels in t frames
// under ground friction.
func aiThrowLeadDistance(vx float64, frames int) float64 {
	return vx * (1 - math.Pow(Friction, float64(frames))) / (1 - Friction)
}

// aiThrowReleasePoint predicts where the bot stands after holding the key
// for `frames` more frames (standing still: only its current slide).
func aiThrowReleasePoint(p *Player, frames int) (float64, float64) {
	if p.JumpNum == 2 {
		return p.X + aiThrowLeadDistance(p.VX, frames), p.Y
	}
	return p.X + p.VX*float64(frames), p.Y + p.VY*float64(frames)
}

// aiBestThrow tries every reachable charge in the bot's facing direction and
// returns the best scoring throw. It is deterministic (no rolls).
func aiBestThrow(p *Player, g *Game, arena Map, skill float64) (aiThrowPlan, bool) {
	kind := normalizedThrowableType(p.ThrowableType)
	profile := throwableProfileFor(kind)
	targets := aiThrowTargets(p, g)
	if len(targets) == 0 {
		return aiThrowPlan{}, false
	}
	left, _, right, _, ok := platformBounds(arena.Platforms)
	if !ok {
		return aiThrowPlan{}, false
	}
	facing := p.Facing
	if facing == 0 {
		facing = 1
	}
	best := aiThrowPlan{}
	found := false
	minScore := aiThrowMinScore(skill)
	for k := 1; float64(k)*aiThrowPowerStep+1 <= grenadeMaxPower+1e-9; k++ {
		if p.JumpNum != 2 && k > 4 {
			break // airborne: only quick throws, the release point drifts
		}
		power := 1 + aiThrowPowerStep*float64(k)
		rx, ry := aiThrowReleasePoint(p, k)
		res := simulateThrow(arena, rx, ry, facing, power, kind)
		if !res.OK || res.X < left-40 || res.X > right+40 {
			continue
		}
		if !aiThrowSafeAt(p, g, arena, res, kind, rx, ry) {
			continue
		}
		score, targetID := aiScoreThrow(p, g, arena, res, profile, targets, k, skill, rx, ry)
		if score < minScore {
			continue
		}
		if !found || score > best.Score {
			best = aiThrowPlan{Power: power, Score: score, Result: res, TargetID: targetID}
			found = true
		}
	}
	return best, found
}

// aiThrowReleaseSafe re-checks the self/teammate safety for a release now.
func aiThrowReleaseSafe(p *Player, g *Game, arena Map, x, y float64, facing int, power float64) bool {
	kind := normalizedThrowableType(p.ThrowableType)
	res := simulateThrow(arena, x, y, facing, power, kind)
	if !res.OK {
		return true // lost off the map: wasted, but harmless
	}
	return aiThrowSafeAt(p, g, arena, res, kind, x, y)
}

// aiThrowHarmRadius is the reach of a throwable's harmful effect around its
// detonation point, and the vertical offset of that effect's centre from
// the grenade (all effects measure against the player's body at feet-30).
func aiThrowHarmRadius(kind int) (radius, centreLift float64, harmful bool) {
	profile := throwableProfileFor(kind)
	switch profile.Behavior {
	case throwableBehaviorExplode:
		return profile.Radius, 0, true
	case throwableBehaviorShock:
		return gasCloudRadius, 22, true // spawnGasCloud: centre at g.Y-22
	case throwableBehaviorTearGas:
		return tearGasCloudRadius, 34, true // spawnTearGasCloud: centre at g.Y-34
	case throwableBehaviorIgnite:
		return profile.Radius, 0, true
	}
	return 0, 0, false
}

// aiThrowSafeAt rejects throws that would catch the bot itself (effect radius
// + aiThrowSelfMargin at its release point) or a teammate, and flashbangs
// that would blind the bot longer than a turned-away glance.
func aiThrowSafeAt(p *Player, g *Game, arena Map, res throwSimResult, kind int, selfX, selfY float64) bool {
	profile := throwableProfileFor(kind)
	if profile.Behavior == throwableBehaviorFlash {
		fx, fy := res.X, res.Y-flashPointLift
		probe := *p
		probe.X, probe.Y = selfX, selfY
		// The bot turns its back right before the pop (aiThrowFlashFollowUp)
		// when grounded; airborne it cannot be sure to, so assume it faces it.
		probe.Facing = 1
		if fx < selfX {
			probe.Facing = -1
		}
		if p.JumpNum == 2 {
			probe.Facing = -probe.Facing
		}
		if frames, _ := flashbangBlindFor(arena, fx, fy, &probe); frames > flashAwayBlindFrames || (p.JumpNum != 2 && frames > 0) {
			return false
		}
		for _, mate := range g.players {
			if !aiThrowIsTeammate(p, mate) {
				continue
			}
			if frames, _ := flashbangBlindFor(arena, fx, fy, mate); frames > flashAwayBlindFrames {
				return false
			}
		}
		return true
	}
	radius, lift, harmful := aiThrowHarmRadius(kind)
	if !harmful {
		return true
	}
	cx, cy := res.X, res.Y-lift
	reach := radius + aiThrowSelfMargin
	// The Molotov thrower never burns in their own fire (fireOwnerImmune);
	// everyone else, teammates included, does.
	if profile.Behavior != throwableBehaviorIgnite && math.Hypot(selfX-cx, selfY-30-cy) <= reach {
		return false
	}
	for _, mate := range g.players {
		if !aiThrowIsTeammate(p, mate) {
			continue
		}
		if profile.Behavior == throwableBehaviorIgnite && fireOwnerImmune(mate, p.scoreOwnerID()) {
			continue
		}
		if math.Hypot(mate.X-cx, mate.Y-30-cy) <= reach {
			return false
		}
		if profile.Behavior == throwableBehaviorIgnite && aiThrowOnFireStrip(arena, res, mate.X, mate.Y, aiThrowSelfMargin) {
			return false
		}
	}
	return true
}

// aiThrowOnFireStrip reports whether a player at (x, y) would stand in the
// Molotov fire strip spawned at the predicted break point (+margin).
func aiThrowOnFireStrip(arena Map, res throwSimResult, x, y, margin float64) bool {
	surfaceY, ok := fireSurfaceY(arena, res.X, res.Y)
	if !ok {
		return false
	}
	vertical := y - surfaceY
	return vertical >= -52 && vertical <= 16 && math.Abs(x-res.X) <= 3*28+molotovFireHalfWidth+margin
}

// aiThrowPlatformEdge returns the distance from (x, y) to the nearer edge of
// the platform under it and the side (-1 left, +1 right) of that edge.
func aiThrowPlatformEdge(arena Map, x, y float64) (float64, int, bool) {
	for _, r := range arena.Platforms {
		if !r.ContainsX(x) || r.Y < y-6 || r.Y > y+30 {
			continue
		}
		left, right := x-r.X, r.X+r.W-x
		if left < right {
			return left, -1, true
		}
		return right, 1, true
	}
	return 0, 0, false
}

// aiThrowBulletsBlocked: the bot's bullets cannot reach this target (other
// level, behind a platform, hidden in smoke).
func aiThrowBulletsBlocked(p *Player, g *Game, arena Map, t aiThrowTarget) bool {
	if t.Hidden {
		return true
	}
	if t.Y >= p.Y+20 || t.Y <= p.Y-80 {
		return true // decideAIController only shoots inside this band
	}
	if flashOccluded(arena, p.X, p.Y-38, t.X, t.Y-35) {
		return true
	}
	return g != nil && aiSmokeHides(g.areaEffects, p, t.P)
}

// aiThrowEffectValue is the 0..1 effect of a throw going off at res on a
// player at (x, y) (feet). Values are relative to each type's full effect.
func aiThrowEffectValue(arena Map, res throwSimResult, profile throwableProfile, target *Player, x, y float64) float64 {
	switch profile.Behavior {
	case throwableBehaviorExplode:
		d := math.Hypot(x-res.X, y-30-res.Y)
		if d > profile.Radius || profile.HorizontalPush <= 0 {
			return 0
		}
		v := grenadeBlastAxis(profile.HorizontalPush, profile.EdgeHorizontalPush, d, profile.Radius) / profile.HorizontalPush
		if target.PerkNumber == 6 || target.ShieldTime > 0 {
			v /= 5
		}
		return v
	case throwableBehaviorIgnite:
		if d := math.Hypot(x-res.X, y-30-res.Y); d <= profile.Radius {
			return 0.85 + 0.15*(1-d/profile.Radius) // prefer the target's feet
		}
		if aiThrowOnFireStrip(arena, res, x, y, 0) {
			return 0.6
		}
		return 0
	case throwableBehaviorShock:
		d := math.Hypot(x-res.X, y-30-(res.Y-22))
		if d >= gasCloudRadius {
			return 0
		}
		v := 0.3 + 0.5*(1-d/gasCloudRadius)
		if target.PerkNumber == 6 || target.ShieldTime > 0 {
			v *= gasBlastShieldScale
		}
		return v
	case throwableBehaviorTearGas:
		if target.AbilityNumber == abilityGasMask {
			return 0
		}
		if d := math.Hypot(x-res.X, y-30-(res.Y-34)); d <= tearGasCloudRadius*0.9 {
			return 0.45 + 0.15*(1-d/(tearGasCloudRadius*0.9)) // prefer the cloud core
		}
		return 0
	case throwableBehaviorFlash:
		probe := *target
		probe.X, probe.Y = x, y
		frames, _ := flashbangBlindFor(arena, res.X, res.Y-flashPointLift, &probe)
		return float64(frames) / flashFullBlindMaxFrames
	}
	return 0
}

// aiScoreThrow scores a simulated throw against the known enemies. A throw
// only counts when one of the reasons to throw applies: two or more enemies
// caught, a knock-off near a platform edge, a target bullets cannot reach, a
// camper, or the bot's own gun being empty. Smoke is scored defensively.
func aiScoreThrow(p *Player, g *Game, arena Map, res throwSimResult, profile throwableProfile,
	targets []aiThrowTarget, holdFrames int, skill, selfX, selfY float64) (float64, int) {
	frames := holdFrames + res.Frames
	lead := aiThrowLeads(skill)
	weaponDown := p.Weapon.Bullets <= 0
	if profile.Behavior == throwableBehaviorSmoke {
		return aiScoreSmoke(p, g, res, targets, weaponDown, selfX, selfY)
	}
	total, affected := 0.0, 0
	triggered := false
	bestID, bestValue := 0, 0.0
	for _, t := range targets {
		x, y := t.X, t.Y
		if lead && !t.Hidden {
			x += aiThrowLeadDistance(t.VX, frames) * skill
		}
		v := aiThrowEffectValue(arena, res, profile, t.P, x, y)
		if v <= 0 {
			continue
		}
		if v >= 0.25 {
			affected++
		}
		weight := 0.8
		if t.Primary {
			weight = 1
		}
		reason := false
		if profile.Behavior == throwableBehaviorExplode || profile.Behavior == throwableBehaviorShock {
			if dist, side, ok := aiThrowPlatformEdge(arena, x, y); ok && dist < aiThrowEdgeDistance {
				pushSide := 1
				if x < res.X {
					pushSide = -1
				}
				if pushSide == side && v >= 0.3 {
					reason = true
					v += 0.4
				}
			}
		}
		if profile.Behavior == throwableBehaviorShock || profile.Behavior == throwableBehaviorIgnite {
			// Campers and players who cannot leave their spot fast are what
			// lingering effects punish.
			if t.P.ID == p.AIThrowCampID && p.AIThrowCampFrames >= aiThrowCampFrames {
				v += 0.2
			}
		}
		if aiThrowBulletsBlocked(p, g, arena, t) {
			reason = true
		}
		if t.P.ID == p.AIThrowCampID && p.AIThrowCampFrames >= aiThrowCampFrames {
			reason = true
		}
		if weaponDown {
			reason = true
		}
		if reason {
			triggered = true
		}
		total += v * weight
		if v > bestValue {
			bestValue, bestID = v, t.P.ID
		}
	}
	if affected >= 2 {
		triggered = true
		total += 0.2
	}
	if !triggered {
		return 0, 0
	}
	return total, bestID
}

// aiScoreSmoke: smoke is defensive. It is worth throwing when an enemy who
// can shoot the bot (same level, facing it) would have its line of sight
// cut by the cloud, more so when the bot cannot shoot back.
func aiScoreSmoke(p *Player, g *Game, res throwSimResult, targets []aiThrowTarget, weaponDown bool, selfX, selfY float64) (float64, int) {
	cx, cy := res.X, res.Y-40
	core := smokeCloudRadius * smokeCoreFactor
	for _, t := range targets {
		if t.Hidden || !t.Primary {
			continue
		}
		if math.Abs(t.Y-selfY) > 80 {
			continue
		}
		facingBot := (selfX-t.X)*float64(t.P.Facing) > 0
		if !facingBot {
			continue
		}
		ex, ey := t.X, t.Y-35
		bx, by := selfX, selfY-45
		if g != nil && smokeBlocksSegment(g.areaEffects, ex, ey, bx, by) {
			return 0, 0 // already covered
		}
		if pointSegmentDistance(cx, cy, ex, ey, bx, by) > core*0.8 {
			continue
		}
		score := 0.65
		if weaponDown {
			score += 0.4
		}
		if p.AbilityNumber == abilityThermalGoggles {
			score += 0.3 // the bot still sees through it
		}
		return score, t.P.ID
	}
	return 0, 0
}
