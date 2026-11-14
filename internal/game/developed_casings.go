package game

import (
	"math"
	"math/rand"
)

// Spent casings and flame nozzle of the developed arsenal (87-231).
//
// Source guns eject casings from their timeline scripts (ejectshell* /
// ejectshot*, see enterWeaponTimelineFrame); developed weapons have no
// timeline, so fireWeaponRound ejects one per round here. The particle kinds
// are the source shell clips, chosen like the source chooses them per class:
//
//	1 ejectshell   pistols, SMGs, rifles (M1911, AK, MP5K, ...)
//	2 ejectshell2  snipers (HK, SCAR, AWM, Galatz, ...)
//	4 ejectshell4  belt-fed machine guns (Stoner, M249, F2000)
//	5 ejectshot    shotguns
//	6 ejectshot3   bottom-ejecting bullpup shotguns (KSG, Jackhammer)
//
// Casings are ordinary Shell entries in the snapshot, so online clients see
// exactly the host's casings.

// developedCasingKind returns the shell particle a developed weapon ejects per
// round, or 0 for none: launchers, bows, crossbow, harpoon, nail gun, flare
// gun, musket, energy, flamethrower and portal gun (no cartridge), revolvers
// and break-action guns (they keep the case until reloading) and the caseless
// G11.
func developedCasingKind(number int, spec DevelopedWeaponSpec) int {
	switch number {
	case 111, 112, 231, // revolvers
		190,      // Howdah: break action
		183, 229, // single-shot sawed-off, Model 1900 double: break action
		213: // H&K G11: caseless ammunition
		return 0
	case 94, 95, 140, 141, 142, 143, 144, 199, 200: // belt-fed machine guns
		return 4
	case 166: // DP-12: bullpup, ejects downward
		return 6
	}
	if spec.Projectile != DevelopedProjectileBullet {
		return 0
	}
	switch spec.Category {
	case DevelopedCategoryPistol, DevelopedCategorySMG, DevelopedCategoryRifle, DevelopedCategoryLMG:
		return 1
	case DevelopedCategorySniper:
		return 2
	case DevelopedCategoryShotgun:
		return 5
	}
	return 0 // specials (the musket fires a ball) and launchers
}

// developedBullpup lists the bullpup layouts: their ejection port sits behind
// the grip, like the source M17 / AUG SMG / HBAR (ShellX -3..0).
var developedBullpup = map[int]bool{133: true, 134: true, 135: true, 136: true, 166: true}

// developedCasingPort is the ejection port in world px (facing right, before
// the mini multiplier), derived from the measured fit (developed_weapon_fit.go):
// a little ahead of the grip hand (pose.ShootX) toward the muzzle (FlashX), on
// the bore line. The fractions match the source ShellX values: M1911 25 with
// grip 25 / muzzle 45, AK 20 with 12 / 75, HK sniper 20 with 10 / 70.
func developedCasingPort(number int, spec DevelopedWeaponSpec, kind int) (float64, float64) {
	pose := poseForWeapon(number)
	def, _ := WeaponByNumber(number)
	y := developedFlashY(number)
	if kind == 3 || kind == 6 {
		y = -27 // source ejectshell3 / ejectshot3 height (bottom eject)
	}
	if developedBullpup[number] {
		return pose.ShootX - 24, y
	}
	frac := 0.15
	if spec.Category == DevelopedCategoryPistol {
		frac = 0.08
	}
	return pose.ShootX + frac*(def.FlashX-pose.ShootX), y
}

// spawnDevelopedCasing ejects one casing for a developed round. Source shells
// fly at a random +-5 px; these go up and backward relative to the facing
// (bottom-eject shells just drop).
func spawnDevelopedCasing(p *Player, w *WeaponState, spec DevelopedWeaponSpec, shells *[]Shell) {
	if shells == nil {
		return
	}
	kind := developedCasingKind(w.Def.Number, spec)
	if kind == 0 {
		return
	}
	mini := p.MiniMulti
	if mini <= 0 {
		mini = 1
	}
	facing := p.Facing
	if facing == 0 {
		facing = 1
	}
	f := float64(facing)
	portX, portY := developedCasingPort(w.Def.Number, spec, kind)
	rot := rand.Float64()*10 + 5
	if rand.Intn(2) == 0 {
		rot = -rot
	}
	s := Shell{
		X:        p.X + portX*mini*f,
		Y:        p.Y + portY*mini,
		VX:       -f * (0.5 + rand.Float64()*3.5),
		VY:       rand.Float64()*4 - 9,
		RotSpeed: rot,
		ScaleX:   f,
		Kind:     kind,
	}
	if kind == 6 {
		s.VX = f * (rand.Float64()*2 - 1)
		s.VY = rand.Float64() * 2
	}
	*shells = append(*shells, s)
}

// developedFlameStart places a flamethrower's flame at the measured nozzle
// (developedWeaponMuzzles) instead of the source bullet origin (+23 px), and
// returns the lifetime that keeps the stream's reach where it was: the flame
// starts (FlashX-23)*mini px further out, so the frames it would spend
// covering that distance are taken off the tail of its life. Point-blank
// hits are kept by sweeping back to the old origin on the first step
// (Bullet.FlameSweepBack).
func developedFlameStart(p *Player, w *WeaponState, spec DevelopedWeaponSpec, mini float64) (x, y float64, lifetime int, sweepBack float64) {
	facing := float64(p.Facing)
	if facing == 0 {
		facing = 1
	}
	nozzle := w.Def.FlashX
	if nozzle < 23 {
		nozzle = 23
	}
	offset := (nozzle - 23) * mini
	return p.X + nozzle*mini*facing, p.Y + developedFlashY(w.Def.Number)*mini,
		developedFlameLifetime(spec.ProjectileSpeed, spec.Lifetime, offset), offset
}

// developedFlameTravel is how far a flame moves horizontally over a lifetime
// (updateDevelopedProjectile: VX *= 0.988 before each move, it moves on Time
// 2..lifetime).
func developedFlameTravel(speed float64, lifetime int) float64 {
	total, vx := 0.0, speed
	for t := 2; t <= lifetime; t++ {
		vx *= 0.988
		total += vx
	}
	return total
}

// developedFlameLifetime returns the lifetime whose end point from a start
// `offset` px further out is closest to the original end point.
func developedFlameLifetime(speed float64, lifetime int, offset float64) int {
	if offset <= 0 || speed <= 0 || lifetime <= 2 {
		return lifetime
	}
	want := developedFlameTravel(speed, lifetime)
	best, bestErr := lifetime, math.Inf(1)
	for l := 2; l <= lifetime; l++ {
		if e := math.Abs(offset + developedFlameTravel(speed, l) - want); e < bestErr {
			best, bestErr = l, e
		}
	}
	return best
}
