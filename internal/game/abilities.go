package game

import (
	"math"
	"math/rand"
	"strconv"
)

// Second loadout slot: ABILITY / GEAR. Every player picks one original perk
// (1-9, Player.PerkNumber) AND one ability or piece of gear
// (Player.AbilityNumber). Gameplay is host-authoritative; the runtime fields
// below are ordinary exported Player fields, so the snapshot codec syncs them
// to online clients for the HUD and the visuals.
//
// Ability ids are stored in configs and sent online as ints: append new ids
// (and raise abilityCount), never reorder.
const (
	abilityNone           = 0
	abilityDash           = 1  // active: horizontal burst (ability key)
	abilityAnchor         = 2  // active: hold to ignore knockback (ability key)
	abilityTripleJump     = 3  // passive: one extra, weaker mid-air jump
	abilityHeavy          = 4  // passive: -20% knockback, slower and lower
	abilityLight          = 5  // passive: +20% knockback, faster and higher
	abilityShield         = 6  // passive: blocks one hit per life
	abilityGrenadier      = 7  // passive: +2 throwables at spawn
	abilityScavenger      = 8  // passive: sees crate contents, +1/3 crate ammo
	abilityFlashGoggles   = 9  // gear: flashbang immunity
	abilityThermalGoggles = 10 // gear: sees through smoke
	abilityGasMask        = 11 // gear: shock gas / tear gas immunity
	abilityKevlar         = 12 // gear: -25% bullet knockback, a bit slower
	// gear: riot shield, always up (arsenal2_defs.go)
	abilityBallisticShield = 13
	// abilityCount bounds per-ability tables (index = ability id).
	abilityCount = 14
	// abilityRandom is a config-only choice. It is resolved to a real ability
	// when the player spawns into a match and never stored on a Player. It is
	// far above abilityCount so new abilities can be appended.
	abilityRandom = 50
)

// Legacy perk numbers of the gear that used to live in the perk slot. Old
// configs carrying them are migrated by migrateLegacyGearPerk.
const (
	legacyPerkFlashGoggles   = 14
	legacyPerkThermalGoggles = 15
	legacyPerkGasMask        = 16
	legacyPerkKevlar         = 17
)

// Tunables. Frames are simulation ticks (30 TPS, main.go SetTPS(30)).
const (
	dashSpeed          = 17.0 // horizontal velocity set by a dash (run top speed is ~9)
	dashAirLift        = 2.0  // an air dash cancels falling and lifts slightly
	dashTrailFrames    = 12   // frames of dash after-image trail
	dashCooldownFrames = 240  // 8 s

	anchorMaxFrames      = 90  // 3 s per use
	anchorCooldownFrames = 210 // 7 s, starts when the anchor is released
	anchorBrake          = 0.55

	heavyKnockbackScale = 0.80
	heavyMoveScale      = 0.88
	heavyJumpScale      = 0.90

	lightKnockbackScale = 1.20
	lightMoveScale      = 1.12
	lightJumpScale      = 1.10

	tripleJumpExtraJumps = 1
	tripleJumpScale      = 0.70 // of the first jump; the second jump is 0.83

	grenadierBonusThrowables = 2

	// SCAVENGER: +1/3 of the weapon's base ammo on every crate pickup. It is
	// computed from the base magazine like EXTRA AMMO (perk 7) and simply
	// added, so both together give +2/3 (each rounded up).
	scavengerAmmoScale = 0.333

	shieldBreakFrames = 24
	// shieldMinImpulse ignores tiny environmental nudges (gas drift, wind)
	// so the one-hit shield is spent on a real hit.
	shieldMinImpulse = 1.5
)

// selectableAbilityNumbers lists every config choice in menu order.
func selectableAbilityNumbers() []int {
	return []int{
		abilityNone, abilityDash, abilityAnchor, abilityTripleJump,
		abilityHeavy, abilityLight, abilityShield,
		abilityGrenadier, abilityScavenger, abilityFlashGoggles,
		abilityThermalGoggles, abilityGasMask, abilityKevlar,
		abilityBallisticShield,
		abilityRandom,
	}
}

// realAbilityNumbers are the abilities a player can actually have (no NONE,
// no RANDOM).
func realAbilityNumbers() []int {
	out := make([]int, 0, abilityCount-1)
	for n := 1; n < abilityCount; n++ {
		out = append(out, n)
	}
	return out
}

func isSelectableAbility(n int) bool {
	return (n >= abilityNone && n < abilityCount) || n == abilityRandom
}

// abilityMenuIndex is the position of an ability in selectableAbilityNumbers;
// menu hit codes use index+1 so they stay small.
func abilityMenuIndex(n int) int {
	for i, a := range selectableAbilityNumbers() {
		if a == n {
			return i
		}
	}
	return -1
}

func abilityFromMenuNumber(number int) (int, bool) {
	list := selectableAbilityNumbers()
	if number < 1 || number > len(list) {
		return 0, false
	}
	return list[number-1], true
}

func abilityName(n int) string {
	switch n {
	case abilityNone:
		return "NONE"
	case abilityDash:
		return "DASH"
	case abilityAnchor:
		return "ANCHOR"
	case abilityTripleJump:
		return "TRIPLE JUMP"
	case abilityHeavy:
		return "HEAVY"
	case abilityLight:
		return "LIGHT"
	case abilityShield:
		return "SHIELD"
	case abilityGrenadier:
		return "GRENADIER"
	case abilityScavenger:
		return "SCAVENGER"
	case abilityFlashGoggles:
		return "FLASH GOGGLES"
	case abilityThermalGoggles:
		return "THERMAL GOGGLES"
	case abilityGasMask:
		return "GAS MASK"
	case abilityKevlar:
		return "KEVLAR"
	case abilityBallisticShield:
		return "BALLISTIC SHIELD"
	case abilityRandom:
		return "RANDOM"
	}
	return "ABILITY"
}

// abilityShortName fits the HUD badge.
func abilityShortName(n int) string {
	switch n {
	case abilityTripleJump:
		return "3-JUMP"
	case abilityFlashGoggles:
		return "FLASH G."
	case abilityThermalGoggles:
		return "THERMAL"
	case abilityBallisticShield:
		return "B.SHIELD"
	}
	return abilityName(n)
}

// framesAsSeconds formats a tick count as real seconds ("8", "1.5").
func framesAsSeconds(frames int) string {
	return strconv.FormatFloat(float64(frames)/TicksPerSecond, 'f', -1, 64)
}

func abilityDescription(n int) string {
	switch n {
	case abilityNone:
		return "No ability."
	case abilityDash:
		return "Ability key: quick dash (also once in the air). " + framesAsSeconds(dashCooldownFrames) + " s cooldown."
	case abilityAnchor:
		return "Hold ability key: no knockback, but no moving or shooting. Max " + framesAsSeconds(anchorMaxFrames) + " s."
	case abilityTripleJump:
		return "A third, weaker mid-air jump."
	case abilityHeavy:
		return "-20% knockback taken. Slower run, lower jump."
	case abilityLight:
		return "Faster run, higher jump. +20% knockback taken."
	case abilityShield:
		return "Blocks one hit (bullet or blast) per life."
	case abilityGrenadier:
		return "+2 throwables at every spawn."
	case abilityScavenger:
		return "See what crates hold. Crate weapons get +1/3 ammo."
	case abilityFlashGoggles:
		return "Immune to flashbangs."
	case abilityThermalGoggles:
		return "See players through smoke and spot invisible players as a heat shimmer."
	case abilityGasMask:
		return "Immune to shock gas and tear gas."
	case abilityKevlar:
		return "-25% bullet knockback. A bit slower."
	case abilityBallisticShield:
		return "Riot shield, always up: front bullets push only 22%. Two-handed guns or akimbo move it to your back. Slower, lower jump."
	case abilityRandom:
		return "A random ability every match."
	}
	return ""
}

// abilityUsesKey reports whether the ability needs the ability key.
func abilityUsesKey(n int) bool {
	return n == abilityDash || n == abilityAnchor
}

// migrateLegacyGearPerk moves gear that an old config stored in the perk
// slot (14-17) into the ability slot; the perk becomes NO PERKS. A config
// that already has an ability keeps it (the stale gear perk is just
// dropped).
func migrateLegacyGearPerk(cfg CustomPlayerConfig) CustomPlayerConfig {
	gear := 0
	switch cfg.Perk {
	case legacyPerkFlashGoggles:
		gear = abilityFlashGoggles
	case legacyPerkThermalGoggles:
		gear = abilityThermalGoggles
	case legacyPerkGasMask:
		gear = abilityGasMask
	case legacyPerkKevlar:
		gear = abilityKevlar
	}
	if gear != 0 {
		cfg.Perk = 1
		if cfg.Ability == abilityNone {
			cfg.Ability = gear
		}
	}
	if !isSelectableAbility(cfg.Ability) {
		cfg.Ability = abilityNone
	}
	return cfg
}

// resolveConfigAbility turns a config choice into the ability a spawned
// player gets (RANDOM picks one of the allowed real abilities).
func resolveConfigAbility(choice int, allowed func(int) bool) int {
	if choice == abilityRandom {
		pool := make([]int, 0, abilityCount)
		for _, n := range realAbilityNumbers() {
			if allowed == nil || allowed(n) {
				pool = append(pool, n)
			}
		}
		if len(pool) == 0 {
			return abilityNone
		}
		return pool[rand.Intn(len(pool))]
	}
	if choice <= abilityNone || choice >= abilityCount {
		return abilityNone
	}
	if allowed != nil && !allowed(choice) {
		return abilityNone
	}
	return choice
}

// ---------------------------------------------------------------------------
// Passive modifiers

func playerAbility(p *Player) int {
	if p == nil {
		return abilityNone
	}
	return p.AbilityNumber
}

// abilityKnockbackScale is the HEAVY / LIGHT scale applied to every external
// push (bullets, explosions, fire, gas, hazards). Anchor and shield are
// handled separately in the knockback window.
func abilityKnockbackScale(p *Player) float64 {
	scale := 1.0
	switch playerAbility(p) {
	case abilityHeavy:
		scale = heavyKnockbackScale
	case abilityLight:
		scale = lightKnockbackScale
	}
	if p != nil && p.KnockbackScale > 0 {
		// Campaign bosses and non-player enemies (campaign_enemies.go) use the
		// same window, so their resistance covers every push source too.
		scale *= p.KnockbackScale
	}
	return scale
}

func abilityMoveMultiplier(p *Player) float64 {
	switch playerAbility(p) {
	case abilityHeavy:
		return heavyMoveScale
	case abilityLight:
		return lightMoveScale
	case abilityKevlar:
		return kevlarMoveScale
	}
	return 1
}

func abilityJumpMultiplier(p *Player) float64 {
	switch playerAbility(p) {
	case abilityHeavy:
		return heavyJumpScale
	case abilityLight:
		return lightJumpScale
	case abilityKevlar:
		return kevlarJumpScale
	}
	return 1
}

func (p *Player) anchored() bool {
	return p != nil && p.AnchorTime > 0
}

// abilityShieldReady: the SHIELD ability still has its one block this life.
func abilityShieldReady(p *Player) bool {
	return p != nil && p.Active && p.AbilityNumber == abilityShield && !p.AbilityShieldUsed
}

// resetAbilityForLife runs on every respawn.
func (p *Player) resetAbilityForLife() {
	p.AbilityShieldUsed = false
	p.AbilityShieldBreak = 0
	p.AnchorTime = 0
	p.DashTime = 0
	p.DashAirUsed = false
	p.AbilityAirJumps = 0
	if p.AbilityNumber == abilityTripleJump {
		p.AbilityAirJumps = tripleJumpExtraJumps
	}
	p.AbilityKeyHeld = false
}

// refillAbilityOnLanding runs when the player lands on a platform.
func (p *Player) refillAbilityOnLanding() {
	p.DashAirUsed = false
	if p.AbilityNumber == abilityTripleJump {
		p.AbilityAirJumps = tripleJumpExtraJumps
	}
}

func (p *Player) abilityPressed() bool {
	if p.RemoteControlled {
		return p.RemoteInput.Ability || p.slowMoLatch&netInputBitAbility != 0
	}
	if p.AI {
		return p.AIAbility
	}
	return sourceKeyDown(p.Controls.Ability) || p.slowMoLatch&netInputBitAbility != 0
}

// ---------------------------------------------------------------------------
// Per-frame update (called from UpdateWithFire)

// updateAbilityTimers ticks the cooldown and FX counters once per frame.
func (p *Player) updateAbilityTimers() {
	if p.AbilityCooldown > 0 {
		p.AbilityCooldown--
		if p.AbilityCooldown == 0 {
			p.AbilityCooldownMax = 0
		}
	}
	if p.DashTime > 0 {
		p.DashTime--
	}
	if p.AbilityShieldBreak > 0 {
		p.AbilityShieldBreak--
	}
}

// updateAnchor runs before movement input. It returns true while the player
// is anchored, in which case all movement input is ignored this frame.
func (p *Player) updateAnchor(pressed, edge bool) bool {
	if p.AbilityNumber != abilityAnchor {
		p.AnchorTime = 0
		return false
	}
	grounded := p.JumpNum == 2
	if p.AnchorTime > 0 {
		if !pressed || !grounded || p.AnchorTime >= anchorMaxFrames {
			p.AnchorTime = 0
			p.AbilityCooldown = anchorCooldownFrames
			p.AbilityCooldownMax = anchorCooldownFrames
			return false
		}
		p.AnchorTime++
	} else if edge && p.AbilityCooldown == 0 && grounded {
		p.AnchorTime = 1
		p.PendingSounds = append(p.PendingSounds, "drop2.wav") // heavy clunk
	}
	if p.AnchorTime > 0 {
		p.VX *= anchorBrake
		if math.Abs(p.VX) < 0.1 {
			p.VX = 0
		}
		return true
	}
	return false
}

// tryDash runs after movement input (so it uses this frame's facing).
func (p *Player) tryDash(edge bool) bool {
	if p.AbilityNumber != abilityDash || !edge || p.AbilityCooldown > 0 {
		return false
	}
	grounded := p.JumpNum == 2
	if !grounded && p.DashAirUsed {
		return false
	}
	facing := float64(p.Facing)
	if facing == 0 {
		facing = 1
	}
	p.VX = facing * dashSpeed
	if !grounded {
		p.DashAirUsed = true
		if p.VY > -dashAirLift {
			p.VY = -dashAirLift
		}
	}
	p.DashTime = dashTrailFrames
	p.AbilityCooldown = dashCooldownFrames
	p.AbilityCooldownMax = dashCooldownFrames
	p.PendingSounds = append(p.PendingSounds, "jetpack.wav")
	return true
}

// tryTripleJump is the extra mid-air jump of TRIPLE JUMP. It only fires
// after both normal jumps (and a JUMP+ perk jump) have been used.
func (p *Player) tryTripleJump(up bool, jumpPower float64) bool {
	if p.AbilityNumber != abilityTripleJump || !up || p.JumpHeld || p.JumpNum != 0 || p.AbilityAirJumps <= 0 {
		return false
	}
	p.AbilityAirJumps--
	p.JumpHeld = true
	p.VY = -jumpPower * tripleJumpScale
	p.Y--
	return true
}

// ---------------------------------------------------------------------------
// Knockback window
//
// Knockback in this game is a velocity change applied by many systems
// (bullets, explosions, Molotov fire, gas, chaos and campaign hazards). The
// ability rules (HEAVY / LIGHT scale, ANCHOR immunity, the one-hit SHIELD)
// are applied once around each of those update blocks instead of being
// repeated in every push site, so new knockback sources inherit them.

type abilityKnockbackState struct {
	vx, vy          float64
	hitNumber       int
	lastHitBy       int
	hitByGrenade    bool
	burnTime        int
	burnPower       float64
	burnOwnerID     int
	burnDirection   int
	burnTick        int
	instagibFrame   int
	instagibPlaying bool
	burnPanic       burnPanicSnapshot
}

// abilityWindowNeeded skips the bookkeeping when nobody has an ability that
// changes knockback.
func abilityWindowNeeded(players []*Player) bool {
	for _, p := range players {
		if p == nil || !p.Active {
			continue
		}
		switch p.AbilityNumber {
		case abilityHeavy, abilityLight, abilityShield, abilityAnchor:
			return true
		}
		if p.KnockbackScale > 0 && p.KnockbackScale != 1 {
			return true
		}
	}
	return false
}

func beginAbilityKnockbackWindow(players []*Player) []abilityKnockbackState {
	if !abilityWindowNeeded(players) {
		return nil
	}
	out := make([]abilityKnockbackState, len(players))
	for i, p := range players {
		if p == nil {
			continue
		}
		out[i] = abilityKnockbackState{
			vx: p.VX, vy: p.VY, hitNumber: p.HitNumber, lastHitBy: p.LastHitBy, hitByGrenade: p.HitByGrenade,
			burnTime: p.BurnTime, burnPower: p.BurnPower, burnOwnerID: p.BurnOwnerID, burnDirection: p.BurnDirection,
			burnTick: p.BurnTick, instagibFrame: p.InstagibFrame, instagibPlaying: p.InstagibPlaying,
			burnPanic: p.burnPanicSnapshot(),
		}
	}
	return out
}

// endAbilityKnockbackWindow applies the ability rules to every velocity
// change since the matching begin call. shieldBlocks=false is used for
// environmental windows (chaos events) that must not spend the shield.
func endAbilityKnockbackWindow(players []*Player, before []abilityKnockbackState, shieldBlocks bool) (shieldBroke bool) {
	if before == nil {
		return false
	}
	for i, p := range players {
		if p == nil || i >= len(before) || !p.Active {
			continue
		}
		b := before[i]
		dvx, dvy := p.VX-b.vx, p.VY-b.vy
		if dvx == 0 && dvy == 0 && p.HitNumber == b.hitNumber && p.InstagibPlaying == b.instagibPlaying {
			continue
		}
		switch {
		case p.AnchorTime > 0:
			// Planted: no push at all. Hit credit (LastHitBy) stays.
			p.VX, p.VY = b.vx, b.vy
		case shieldBlocks && p.AbilityNumber == abilityShield && p.ShieldTime <= 0 && shieldShouldBlock(p, b, dvx, dvy):
			restoreAbilityKnockbackState(p, b)
			if !p.AbilityShieldUsed {
				p.AbilityShieldUsed = true
				p.AbilityShieldBreak = shieldBreakFrames
				shieldBroke = true
			}
		default:
			if scale := abilityKnockbackScale(p); scale != 1 {
				p.VX = b.vx + dvx*scale
				p.VY = b.vy + dvy*scale
			}
		}
	}
	return shieldBroke
}

// shieldShouldBlock: the shield is still up (or was spent earlier this very
// frame, which keeps the rest of that frame's hits off too) and this is a
// real hit by somebody else.
func shieldShouldBlock(p *Player, b abilityKnockbackState, dvx, dvy float64) bool {
	justBroke := p.AbilityShieldUsed && p.AbilityShieldBreak == shieldBreakFrames
	if p.AbilityShieldUsed && !justBroke {
		return false
	}
	hit := p.HitNumber != b.hitNumber || (p.InstagibPlaying && !b.instagibPlaying)
	if !hit && math.Hypot(dvx, dvy) < shieldMinImpulse {
		return false
	}
	// Your own rocket / grenade (rocket jumps) never spends the shield.
	if p.HitNumber == b.hitNumber && p.LastHitBy != 0 && p.LastHitBy == p.scoreOwnerID() {
		return false
	}
	return true
}

func restoreAbilityKnockbackState(p *Player, b abilityKnockbackState) {
	p.VX, p.VY = b.vx, b.vy
	p.HitNumber = b.hitNumber
	p.LastHitBy = b.lastHitBy
	p.HitByGrenade = b.hitByGrenade
	p.BurnTime, p.BurnPower, p.BurnOwnerID = b.burnTime, b.burnPower, b.burnOwnerID
	p.BurnDirection, p.BurnTick = b.burnDirection, b.burnTick
	p.restoreBurnPanic(b.burnPanic)
	p.InstagibFrame, p.InstagibPlaying = b.instagibFrame, b.instagibPlaying
}

// abilityBurnTickScale applies to the per-tick fire push in updateBurning,
// which runs inside the player update rather than in a knockback window.
func abilityBurnTickScale(p *Player) float64 {
	if p.anchored() {
		return 0
	}
	return abilityKnockbackScale(p)
}

// ---------------------------------------------------------------------------
// Crates (SCAVENGER)

// scavengerBonusAmmo is the extra ammo a SCAVENGER gets for a crate weapon.
func scavengerBonusAmmo(p *Player, number int) int {
	if p == nil || p.AbilityNumber != abilityScavenger {
		return 0
	}
	base := NewWeapon(number).Bullets
	if base <= 0 || base > 1000 { // infinite / special-ammo weapons
		return 0
	}
	return int(math.Ceil(float64(base) * scavengerAmmoScale))
}

// ---------------------------------------------------------------------------
// Bots

// aiUseAbility decides the ability key for a bot. It runs at the end of
// decideAIController, after the movement decisions.
func aiUseAbility(p *Player, g *Game) {
	if p == nil || g == nil || !p.AI || !p.Active {
		return
	}
	switch p.AbilityNumber {
	case abilityDash:
		aiDecideDash(p, g)
	case abilityAnchor:
		aiDecideAnchor(p, g)
	}
}

func aiPlatformSpan(g *Game) (left, right float64) {
	left, _, right, _, ok := platformBounds(g.arena.Platforms)
	if !ok {
		return 0, ScreenWidth
	}
	return left, right
}

// aiDecideDash: recover toward the arena after being knocked off, and dash
// away from an edge when a shot is about to arrive.
func aiDecideDash(p *Player, g *Game) {
	if p.AbilityCooldown > 0 {
		return
	}
	left, right := aiPlatformSpan(g)
	middle := (left + right) / 2
	grounded := p.JumpNum == 2
	toward := 1
	if p.X > middle {
		toward = -1
	}
	if !grounded && !p.DashAirUsed {
		off := p.X < left+10 || p.X > right-10
		flungOut := (p.VX > 9 && p.X > middle) || (p.VX < -9 && p.X < middle)
		// Only dash once the normal jumps are spent (or while falling), so
		// the dash extends the recovery instead of replacing it.
		if (off || flungOut) && (p.JumpNum <= 0 || p.VY > 2) {
			aiFaceAndPressAbility(p, toward)
		}
		return
	}
	if grounded && aiIncomingThreat(p, g, 170) {
		nearEdge := p.X < left+130 || p.X > right-130
		if nearEdge {
			aiFaceAndPressAbility(p, toward)
		}
	}
}

// aiDecideAnchor holds the anchor near an edge when a projectile or a
// grenade blast is about to push the bot off.
func aiDecideAnchor(p *Player, g *Game) {
	if p.JumpNum != 2 {
		return
	}
	if p.AnchorTime == 0 && p.AbilityCooldown > 0 {
		return
	}
	left, right := aiPlatformSpan(g)
	nearEdge := p.X < left+150 || p.X > right-150
	threat := aiIncomingThreat(p, g, 220) || aiGrenadeAboutToBlow(p, g)
	if threat && (nearEdge || aiGrenadeAboutToBlow(p, g)) {
		p.AIAbility = true
		// Standing still is the price of the anchor; drop movement this frame
		// so the bot does not fight its own decision.
		p.AILeft, p.AIRight, p.AIUp, p.AIDown, p.AIShoot = false, false, false, false, false
	}
}

func aiFaceAndPressAbility(p *Player, dir int) {
	p.AILeft = dir < 0
	p.AIRight = dir > 0
	p.AIAbility = true
}

// aiIncomingThreat: an enemy projectile at body height flying toward the
// bot within reach px.
func aiIncomingThreat(p *Player, g *Game, reach float64) bool {
	for i := range g.bullets {
		b := &g.bullets[i]
		if b.Dead || b.OwnerID == p.scoreOwnerID() {
			continue
		}
		if b.Y < p.Y-80 || b.Y > p.Y+5 {
			continue
		}
		dx := p.X - b.X
		if math.Abs(dx) > reach {
			continue
		}
		vx := b.VX
		if vx == 0 {
			vx = float64(b.Facing)
		}
		if dx*vx > 0 {
			return true
		}
	}
	return false
}

func aiGrenadeAboutToBlow(p *Player, g *Game) bool {
	for i := range g.grenades {
		gr := &g.grenades[i]
		if gr.Dead {
			continue
		}
		profile := throwableProfileFor(gr.Type)
		if profile.Behavior != throwableBehaviorExplode || gr.Fuse < profile.Fuse-15 {
			continue
		}
		if math.Hypot(p.X-gr.X, p.Y-30-gr.Y) < profile.Radius*0.8 {
			return true
		}
	}
	return false
}

const campaignAbilityRing = 12

// campaignEnemyAbility gives campaign bots a fixed, mission-appropriate
// ability so a mission plays the same every time. set: 1 GM1, 2 GM2,
// 3 EXTRA; index is the enemy's order in the mission.
func campaignEnemyAbility(set, level, index int) int {
	var table []int
	switch set {
	case 1:
		table = []int{abilityNone, abilityDash, abilityHeavy, abilityTripleJump, abilityAnchor,
			abilityShield, abilityDash, abilityLight, abilityThermalGoggles, abilityHeavy}
	case 2:
		table = []int{abilityNone, abilityDash, abilityTripleJump, abilityShield, abilityHeavy,
			abilityThermalGoggles, abilityLight, abilityAnchor, abilityDash, abilityKevlar,
			abilityLight, abilityShield, abilityAnchor, abilityTripleJump, abilityHeavy, abilityDash}
	default:
		table = []int{abilityDash, abilityShield, abilityAnchor, abilityTripleJump, abilityHeavy,
			abilityGasMask, abilityDash, abilityKevlar, abilityLight, abilityAnchor,
			abilityFlashGoggles, abilityShield, abilityHeavy, abilityDash, abilityTripleJump,
			abilityAnchor, abilityScavenger, abilityShield, abilityHeavy, abilityDash}
	}
	if level < 1 || len(table) == 0 {
		return abilityNone
	}
	n := table[(level-1)%len(table)]
	if index > 0 && n != abilityNone {
		// A second enemy in the same mission gets a different, still fixed
		// ability so pairs do not mirror each other.
		// Fixed 12-ability ring (the abilities that existed when the
		// missions were authored) so later abilities never reshuffle them.
		n = realAbilityNumbers()[(n-1+index*3)%campaignAbilityRing]
	}
	return n
}
