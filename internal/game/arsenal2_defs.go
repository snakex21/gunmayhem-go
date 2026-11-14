package game

// arsenal2_*.go: the second added-arsenal package (docs/weapons_roster.md).
//
//   - HARPOON GUN (179): a hooked bolt on a rope. A hit PULLS the victim
//     toward the shooter instead of pushing it away (reverse knockback).
//   - LEAF BLOWER (232): a continuous short cone of weak push. It also blows
//     incoming grenades / throwables away and turns rockets around.
//   - BASEBALL BAT (233): melee. Starter option and crate pickup. One swing
//     every swing cooldown, strong knockback with a small pop-up, bats back
//     grenades.
//   - BALLISTIC SHIELD: gear (ability 13) and a crate powerup (kind 8). It is
//     always up: in front with one-handed weapons (pistols, the bat), on the
//     back with two-handed weapons and during AKIMBO.
//   - AKIMBO: crate powerup (kind 7). Dual pistols: twice the fire rate, the
//     off-hand gun has its own magazine.
//   - THERMAL GOGGLES now also show invisible players as a heat shimmer.
//
// Everything is drawn in code (arsenal2_visuals.go); no image files.

const (
	weaponLeafBlower  = 232
	weaponBaseballBat = 233
	weaponHarpoonGun  = 179
)

// Leaf blower tuning (30 TPS ticks). The push is applied every tick while
// the trigger is held, so it is quoted per tick: 1.25 at the nozzle falling
// to 0.45 at the end of the cone. For comparison a starter pistol sustains
// 1.0-1.6 push per tick, an assault rifle about 5.5 and walking against a
// push gives 0.64 per tick: the blower beats walking up close and slides a
// target about 150-250 px over a full second, without ever being a gun.
const (
	blowerFuelTicks     = 150 // 5 s of air per pickup
	blowerRange         = 200.0
	blowerHalfAngleTan  = 0.42 // ~23 degrees half cone
	blowerNearHalfWidth = 26.0 // the nozzle itself is ~50 px tall
	blowerPushNear      = 1.25
	blowerPushFar       = 0.45
	blowerLift          = 0.10
	blowerVisualTicks   = 3
	// Throwables in the cone get a gust per tick (and float a little).
	blowerGrenadePush = 1.1
	blowerGrenadeLift = 0.35
)

// Baseball bat tuning. 26 push every 15 ticks is 1.7 per tick of sustained
// pressure, the top of the starter pistol band, but only within reach.
const (
	batPush            = 26.0
	batLift            = 4.5
	batSwingCooldown   = 15
	batSwingFrames     = 9
	batReachFront      = 66.0
	batReachBack       = 6.0
	batReachTop        = 80.0
	batPickupSwings    = 20 // crate bats break after this many swings
	batShieldScale     = 0.5
	batZombieBonus     = 16.0 // extra zombie damage per bat hit (one-hit wave-1 walkers)
	batGrenadeHitSpeed = 15.0
	batGrenadeHitLift  = 7.0
)

// Harpoon tuning. The rope reels the victim in: the pull impulse is sized to
// slide the victim to about harpoonStopPx from the shooter (an impulse v
// slides v / (1 - 0.93) = 14.3 v px), clamped to [harpoonPullMin,
// harpoonPullMax]. 23 px/tick is ~330 px, about a strong sniper hit in
// reverse; three hooks per pickup.
const (
	harpoonPullMin      = 7.0
	harpoonPullMax      = 23.0
	harpoonStopPx       = 70.0
	harpoonLift         = 4.5
	harpoonClimbMax     = 6.0
	harpoonTetherFrames = 12
	harpoonDirectPush   = 3.0 // the hook's own (forward) tug, before the reel
)

// Ballistic shield tuning.
const (
	ballisticShieldBulletScale = 0.22 // front bullets push 22%
	ballisticShieldMoveScale   = 0.90
	ballisticShieldJumpScale   = 0.93
	ballisticShieldSparkFrames = 6
	// zombie claws into a raised shield: half damage, 40% push.
	ballisticShieldClawDamage = 0.5
	ballisticShieldClawPush   = 0.4
)

// AKIMBO lasts 12 s (30 TPS), the long end of the requested 10-15 s.
const akimboFrames = 360

// Extra crate powerups (powerup.go). Kinds 0-6 are the source ones; these
// only spawn when the developed arsenal is in the crate pool.
const (
	powerupAkimbo          = 7
	powerupBallisticShield = 8
	powerupKindCount       = 9
)

// addArsenal2Weapons runs at the end of the developed catalog init
// (developed_weapons.go) so it replaces the old plain-bolt harpoon.
func addArsenal2Weapons() {
	// HARPOON GUN: three hooks. Its direct push is tiny; the reel does the
	// work (hitPlayerWithHarpoon).
	addSpecialProjectile(weaponHarpoonGun, "Harpoon Gun", "harpoon_gun", 40, harpoonDirectPush, 2.5, 3, DevelopedProjectileHarpoon, 22, .06)
	// LEAF BLOWER: ROF 1, one "round" is one tick of air.
	addDevelopedWeapon(developedDef(weaponLeafBlower, "Leaf Blower", "", 1, blowerPushNear, .03, blowerFuelTicks, 0, .82), DevelopedWeaponSpec{
		Category: DevelopedCategorySpecial, FireMode: DevelopedFireAuto, Projectile: DevelopedProjectileGust,
	})
	// BASEBALL BAT: crate bats hold batPickupSwings swings; a starter bat
	// never breaks (fireMeleeSwing).
	addDevelopedWeapon(developedDef(weaponBaseballBat, "Baseball Bat", "", batSwingCooldown, batPush, 0, batPickupSwings, 0, 1), DevelopedWeaponSpec{
		Category: DevelopedCategoryMelee, FireMode: DevelopedFireAuto, Projectile: DevelopedProjectileBullet,
	})
}

// proceduralWeapon: the weapon art is generated in code (no PNG).
func proceduralWeapon(number int) bool {
	return number == weaponLeafBlower || number == weaponBaseballBat
}

func isMeleeWeapon(number int) bool {
	spec, ok := developedWeaponSpec(number)
	return ok && spec.Category == DevelopedCategoryMelee
}

func isLeafBlower(number int) bool {
	spec, ok := developedWeaponSpec(number)
	return ok && spec.Projectile == DevelopedProjectileGust
}

// weaponIsPistol: the source handguns 1-6 and 8, every added pistol and the
// flare pistol. These are the guns AKIMBO doubles.
func weaponIsPistol(number int) bool {
	if number >= 1 && number <= 8 && number != 7 {
		return true
	}
	if number == 181 {
		return true
	}
	spec, ok := developedWeaponSpec(number)
	return ok && spec.Category == DevelopedCategoryPistol
}

// weaponOneHanded classifies every weapon for the BALLISTIC SHIELD: only
// pistols and melee weapons leave the off hand free for the shield. Every
// other weapon (SMGs, rifles, machine guns, snipers, shotguns, launchers,
// the De-Materializer, the nail gun, the harpoon, the blower, ...) is
// two-handed. The full table is in docs/weapons_roster.md.
func weaponOneHanded(number int) bool {
	return weaponIsPistol(number) || isMeleeWeapon(number)
}

// akimboActive: the AKIMBO powerup is running and the player holds a pistol.
// With any other weapon the timer keeps running but the effect is suspended
// (it comes back when the player is back on a pistol before it runs out).
func akimboActive(p *Player) bool {
	return p != nil && p.AkimboTime > 0 && weaponIsPistol(p.Weapon.Def.Number)
}

// hasBallisticShield: gear slot or the crate pickup.
func hasBallisticShield(p *Player) bool {
	return p != nil && p.Active && (p.AbilityNumber == abilityBallisticShield || p.ShieldGearPickup)
}

// ballisticShieldInFront: the shield is held in the off hand. Both hands busy
// (a two-handed weapon or AKIMBO) moves it onto the back.
func ballisticShieldInFront(p *Player) bool {
	return hasBallisticShield(p) && weaponOneHanded(p.Weapon.Def.Number) && !akimboActive(p)
}

// ballisticShieldCovers reports whether a hit travelling with velocity
// (vx, vy) strikes the shield: mostly horizontal, and coming at the side the
// shield is on (front when held, back when slung). Steep hits from above,
// explosions, fire and gas are never covered.
func ballisticShieldCovers(p *Player, vx, vy float64) bool {
	if !hasBallisticShield(p) || vx == 0 {
		return false
	}
	if vy*vy > vx*vx*1.44 { // steeper than ~50 degrees
		return false
	}
	f := float64(p.Facing)
	if f == 0 {
		f = 1
	}
	if ballisticShieldInFront(p) {
		return vx*f < 0 // flying into the face
	}
	return vx*f > 0 // flying into the back
}

func ballisticShieldMoveMultiplier(p *Player) float64 {
	if hasBallisticShield(p) {
		return ballisticShieldMoveScale
	}
	return 1
}

func ballisticShieldJumpMultiplier(p *Player) float64 {
	if hasBallisticShield(p) {
		return ballisticShieldJumpScale
	}
	return 1
}

// effectiveWeaponROF is the fire cooldown actually used: AKIMBO halves it
// (rounded up).
func effectiveWeaponROF(p *Player) int {
	rof := p.Weapon.Def.ROF
	if akimboActive(p) && rof > 1 {
		rof = (rof + 1) / 2
	}
	return rof
}

// weaponArsenal2Description is the long text of the added package for the
// loadout and the gun library.
func weaponArsenal2Description(number int) string {
	switch number {
	case weaponHarpoonGun:
		return "Hooks the target on a rope and PULLS it toward you. Drag people off safe platforms over a gap. 3 hooks."
	case weaponLeafBlower:
		return "Hold: a short cone of weak, steady push. Blows grenades away and turns rockets around. 5 s of fuel."
	case weaponBaseballBat:
		return "Melee swing with strong knockback and a pop-up. Bats grenades back. Never breaks as a starter; crate bats last 20 swings. Shield-friendly."
	}
	return ""
}

// clearArsenal2State resets the per-life state of the package (respawn and
// final death).
func (p *Player) clearArsenal2State() {
	p.AkimboTime = 0
	p.AkimboHand = 0
	p.ShieldGearPickup = false
	p.ShieldGearSpark = 0
	p.MeleeSwing = 0
	p.MeleeHitPending = false
	p.BlowerTime = 0
	p.BlowerPulse = false
	p.HarpoonTether = 0
	p.HarpoonTetherID = 0
}

// tickArsenal2Timers runs once per player tick (updatePowerupTimers).
func (p *Player) tickArsenal2Timers() {
	if p.AkimboTime > 0 {
		p.AkimboTime--
		if p.AkimboTime == 0 {
			p.AkimboHand = 0
		}
	}
	if p.ShieldGearSpark > 0 {
		p.ShieldGearSpark--
	}
	if p.MeleeSwing > 0 {
		p.MeleeSwing--
	}
	if p.BlowerTime > 0 {
		p.BlowerTime--
	}
	if p.HarpoonTether > 0 {
		p.HarpoonTether--
		if p.HarpoonTether == 0 {
			p.HarpoonTetherID = 0
		}
	}
}

// extendedPickups: the developed arsenal is in this match's crate pool, so
// the added powerups (AKIMBO, BALLISTIC SHIELD) may spawn too. The GM1
// campaign keeps its source-only pool and therefore the exact source roll.
func (g *Game) extendedPickups() bool {
	if g == nil {
		return false
	}
	for _, n := range g.crateWeapons {
		if n > 86 {
			return true
		}
	}
	return false
}

// validateArsenal2Player bounds the synced arsenal2 Player fields of a host
// snapshot (netwire_secure.go): a hostile host cannot make a guest render a
// stuck rope, a never-ending swing or a negative timer.
func validateArsenal2Player(p *Player) error {
	switch {
	case p.AkimboTime < 0 || p.AkimboTime > akimboFrames:
		return errArsenal2Snapshot("akimbo timer")
	case p.AkimboHand < 0 || p.AkimboHand > 1:
		return errArsenal2Snapshot("akimbo hand")
	case p.ShieldGearSpark < 0 || p.ShieldGearSpark > ballisticShieldSparkFrames:
		return errArsenal2Snapshot("shield spark")
	case p.MeleeSwing < 0 || p.MeleeSwing > batSwingFrames:
		return errArsenal2Snapshot("bat swing")
	case p.BlowerTime < 0 || p.BlowerTime > blowerVisualTicks:
		return errArsenal2Snapshot("blower")
	case p.HarpoonTether < 0 || p.HarpoonTether > harpoonTetherFrames:
		return errArsenal2Snapshot("harpoon rope")
	case p.HarpoonTetherID < 0 || p.HarpoonTetherID > 1<<20:
		return errArsenal2Snapshot("harpoon rope owner")
	}
	return nil
}

type errArsenal2Snapshot string

func (e errArsenal2Snapshot) Error() string {
	return "invalid " + string(e) + " in host snapshot"
}
