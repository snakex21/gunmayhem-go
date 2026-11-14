package game

type WeaponAnimSource struct {
	Blowback   float64
	Pushback   float64
	IdleRotate float64
}

// Source frame-1 weapon animation values, transcribed from the original
// ActionScript. Keeping these as Go data means the runtime no longer needs the
// decompiled scripts tree; source/ remains the reference used to verify them.
var weaponAnimCatalog = map[int]WeaponAnimSource{
	1:  {Blowback: 30, Pushback: 0, IdleRotate: 40},
	2:  {Blowback: 30, Pushback: 0, IdleRotate: 40},
	3:  {Blowback: 20, Pushback: 0, IdleRotate: 40},
	4:  {Blowback: 30, Pushback: 10, IdleRotate: 40},
	5:  {Blowback: 30, Pushback: 0, IdleRotate: 40},
	6:  {Blowback: 40, Pushback: 20, IdleRotate: -60},
	8:  {Blowback: 30, Pushback: 0, IdleRotate: 40},
	9:  {Blowback: 20, Pushback: 40, IdleRotate: -80},
	10: {Blowback: 7, Pushback: 10, IdleRotate: -60},
	11: {Blowback: 10, Pushback: 30, IdleRotate: -60},
	12: {Blowback: 10, Pushback: 3, IdleRotate: 40},
	13: {Blowback: 20, Pushback: 30, IdleRotate: -80},
	14: {Blowback: 8, Pushback: 15, IdleRotate: -70},
	15: {Blowback: 10, Pushback: 20, IdleRotate: -60},
	16: {Blowback: 10, Pushback: 30, IdleRotate: -70},
	17: {Blowback: 8, Pushback: 12, IdleRotate: -60},
	18: {Blowback: 10, Pushback: 20, IdleRotate: -80},
	19: {Blowback: 5, Pushback: 3, IdleRotate: 40},
	20: {Blowback: 10, Pushback: 10, IdleRotate: 30},
	21: {Blowback: 10, Pushback: 3, IdleRotate: 30},
	22: {Blowback: 7, Pushback: 8, IdleRotate: -70},
	23: {Blowback: 10, Pushback: 3, IdleRotate: 30},
	24: {Blowback: 10, Pushback: 3, IdleRotate: 40},
	25: {Blowback: 10, Pushback: 3, IdleRotate: 30},
	26: {Blowback: 10, Pushback: 3, IdleRotate: 40},
	27: {Blowback: 10, Pushback: 3, IdleRotate: -60},
	28: {Blowback: 10, Pushback: 3, IdleRotate: 40},
	29: {Blowback: 10, Pushback: 5, IdleRotate: 30},
	30: {Blowback: 5, Pushback: 3, IdleRotate: 20},
	31: {Blowback: 10, Pushback: 3, IdleRotate: 40},
	32: {Blowback: 10, Pushback: 3, IdleRotate: 30},
	33: {Blowback: 10, Pushback: 30, IdleRotate: -70},
	34: {Blowback: 10, Pushback: 30, IdleRotate: -60},
	35: {Blowback: 10, Pushback: 30, IdleRotate: 25},
	36: {Blowback: 10, Pushback: 30, IdleRotate: 30},
	37: {Blowback: 10, Pushback: 30, IdleRotate: -70},
	38: {Blowback: 10, Pushback: 30, IdleRotate: -70},
	39: {Blowback: 8, Pushback: 15, IdleRotate: -70},
	40: {Blowback: 8, Pushback: 15, IdleRotate: 30},
	41: {Blowback: 8, Pushback: 15, IdleRotate: -70},
	42: {Blowback: 8, Pushback: 15, IdleRotate: -60},
	43: {Blowback: 10, Pushback: 20, IdleRotate: -80},
	44: {Blowback: 4, Pushback: 30, IdleRotate: -80},
	45: {Blowback: 20, Pushback: 30, IdleRotate: 30},
	46: {Blowback: 8, Pushback: 15, IdleRotate: -60},
	47: {Blowback: 8, Pushback: 15, IdleRotate: 30},
	48: {Blowback: 20, Pushback: 30, IdleRotate: -80},
	49: {Blowback: 20, Pushback: 30, IdleRotate: -80},
	50: {Blowback: 20, Pushback: 40, IdleRotate: -80},
	51: {Blowback: 10, Pushback: 20, IdleRotate: -70},
	52: {Blowback: 10, Pushback: 20, IdleRotate: -70},
	53: {Blowback: 20, Pushback: 40, IdleRotate: -80},
	54: {Blowback: 20, Pushback: 20, IdleRotate: -70},
	55: {Blowback: 0, Pushback: 10, IdleRotate: 30},
	56: {Blowback: 7, Pushback: 10, IdleRotate: -70},
	57: {Blowback: 7, Pushback: 10, IdleRotate: -60},
	58: {Blowback: 7, Pushback: 10, IdleRotate: -70},
	59: {Blowback: 7, Pushback: 10, IdleRotate: -60},
	60: {Blowback: 7, Pushback: 10, IdleRotate: -70},
	61: {Blowback: 7, Pushback: 10, IdleRotate: -60},
	62: {Blowback: 3, Pushback: 10, IdleRotate: 30},
	63: {Blowback: 7, Pushback: 10, IdleRotate: -60},
	64: {Blowback: 7, Pushback: 10, IdleRotate: -60},
	65: {Blowback: 7, Pushback: 10, IdleRotate: -70},
	66: {Blowback: 7, Pushback: 10, IdleRotate: -60},
	67: {Blowback: 10, Pushback: 3, IdleRotate: 30},
	68: {Blowback: 10, Pushback: 3, IdleRotate: 30},
	69: {Blowback: 10, Pushback: 3, IdleRotate: 40},
	70: {Blowback: 7, Pushback: 10, IdleRotate: -70},
	71: {Blowback: 10, Pushback: 3, IdleRotate: 30},
	72: {Blowback: 7, Pushback: 10, IdleRotate: -60},
	73: {Blowback: 7, Pushback: 10, IdleRotate: -60},
	74: {Blowback: 10, Pushback: 10, IdleRotate: 30},
	75: {Blowback: 7, Pushback: 10, IdleRotate: -70},
	76: {Blowback: 7, Pushback: 10, IdleRotate: -70},
	77: {Blowback: 8, Pushback: 15, IdleRotate: -60},
	78: {Blowback: 8, Pushback: 15, IdleRotate: 30},
	79: {Blowback: 10, Pushback: 30, IdleRotate: -70},
	80: {Blowback: 10, Pushback: 30, IdleRotate: 30},
	81: {Blowback: 10, Pushback: 30, IdleRotate: -60},
	82: {Blowback: 10, Pushback: 20, IdleRotate: -60},
	83: {Blowback: 8, Pushback: 15, IdleRotate: 30},
	84: {Blowback: 20, Pushback: 40, IdleRotate: -80},
	85: {Blowback: 8, Pushback: 15, IdleRotate: 30},
	86: {Blowback: 10, Pushback: 20, IdleRotate: 25},
}

func sourceWeaponAnim(number int) WeaponAnimSource {
	if anim, ok := weaponAnimCatalog[number]; ok {
		return anim
	}
	if spec, ok := developedWeaponSpec(number); ok {
		anim := developedWeaponKick(number, spec)
		anim.IdleRotate = developedWeaponIdleRotate(number, spec)
		return anim
	}
	return WeaponAnimSource{}
}

// developedWeaponIdleRotate: generated 87+ sprites are authored horizontally
// around their grip. Source weapons have different baked-in registration
// angles, so copying their large +/-60 degree values would stand these
// sprites almost upright. Keep the same idle idea with a small, natural tilt.
func developedWeaponIdleRotate(number int, spec DevelopedWeaponSpec) float64 {
	switch spec.Category {
	case DevelopedCategoryPistol:
		return 12
	case DevelopedCategorySMG:
		return 8
	case DevelopedCategoryRifle:
		return -8
	case DevelopedCategoryLMG:
		return -6
	case DevelopedCategorySniper:
		return -7
	case DevelopedCategoryShotgun:
		return -8
	case DevelopedCategoryLauncher:
		return -4
	case DevelopedCategorySpecial:
		switch number {
		case 178: // the bow is already vertical in its source sprite
			return 0
		case 180, 181: // nail gun and flare gun are held like handguns
			return 10
		default:
			return -5
		}
	}
	return 0
}

// developedWeaponKick is the shot kick of a developed weapon: Blowback is the
// muzzle rise (gun rotation) and Pushback pulls both hands back, exactly the
// fields fireWeaponRound applies to source guns; updateSourceVisualState then
// eases the gun back. Unlike the idle tilt these are deltas from the shooting
// pose (rotation 0 for every weapon), so the source values carry over
// directly. Values are the per-class medians of weaponAnimCatalog 1-86:
//
//	handguns 1-3,5,8: 30/0 (Glock 3: 20/0), revolvers 4,6: 30/10, 40/20
//	SMGs 12,19,21-28,30-32,67-71: 10/3
//	assault rifles 10,20,29,56,58-63,72-76: 7/10; semi mini14 (17): 8/12
//	machine guns 57,64-66: 7/10
//	bolt snipers 11,16,33-38,79-81: 10/30; semi snipers 14,39-42,77,78: 8/15
//	pump/lever shotguns 13,45,48-50,53,84: 20/30; semi/auto 15,46,47,51,52,82,83,85,86: 8-10/15-20
//
// Launchers and most specials have no source class; their values are
// estimates in the same range (heavy tubes push back more than they rise).
// The visual fields are ordinary synced Player state, so online clients see
// the kick without any protocol change.
func developedWeaponKick(number int, spec DevelopedWeaponSpec) WeaponAnimSource {
	switch number {
	case 111, 112, 231, 190: // magnum revolvers, Howdah: SNAKE-like
		return WeaponAnimSource{Blowback: 40, Pushback: 20}
	case 99: // Glock 18C machine pistol: SINE PISTOL (Glock)
		return WeaponAnimSource{Blowback: 20}
	case 177: // crossbow: semi sniper class
		return WeaponAnimSource{Blowback: 8, Pushback: 15}
	case 178: // compound bow: string release, no powder kick
		return WeaponAnimSource{Pushback: 6}
	case 179, 182: // harpoon gun, musket: single heavy shot, bolt sniper class
		return WeaponAnimSource{Blowback: 10, Pushback: 30}
	case 180, 189: // nail gun, portal gun: light SMG-like tap
		return WeaponAnimSource{Blowback: 10, Pushback: 3}
	case 181: // flare gun: handgun
		return WeaponAnimSource{Blowback: 30}
	case 185: // energy rifle (full auto): assault rifle class
		return WeaponAnimSource{Blowback: 7, Pushback: 10}
	case 187, 188:
		// Flamethrowers fire every frame, so any rise would be a constant
		// tilt; a small steady brace reads as the stream's push instead.
		return WeaponAnimSource{Pushback: 2}
	}
	switch spec.Category {
	case DevelopedCategoryPistol:
		return WeaponAnimSource{Blowback: 30}
	case DevelopedCategorySMG:
		return WeaponAnimSource{Blowback: 10, Pushback: 3}
	case DevelopedCategoryRifle:
		if spec.FireMode == DevelopedFireSemi {
			return WeaponAnimSource{Blowback: 8, Pushback: 12}
		}
		return WeaponAnimSource{Blowback: 7, Pushback: 10}
	case DevelopedCategoryLMG:
		return WeaponAnimSource{Blowback: 7, Pushback: 10}
	case DevelopedCategorySniper:
		if developedManualAction(number) {
			return WeaponAnimSource{Blowback: 10, Pushback: 30}
		}
		return WeaponAnimSource{Blowback: 8, Pushback: 15}
	case DevelopedCategoryShotgun:
		if developedManualAction(number) {
			return WeaponAnimSource{Blowback: 20, Pushback: 30}
		}
		return WeaponAnimSource{Blowback: 10, Pushback: 20}
	case DevelopedCategoryLauncher:
		if spec.Projectile == DevelopedProjectileGrenade {
			return WeaponAnimSource{Blowback: 12, Pushback: 20}
		}
		return WeaponAnimSource{Blowback: 5, Pushback: 25}
	}
	return WeaponAnimSource{}
}

// developedManualAction marks bolt/lever/pump/break-action guns: in the
// source those are the slow (ROF >= 18) snipers and shotguns.
func developedManualAction(number int) bool {
	def, ok := WeaponByNumber(number)
	return ok && def.ROF >= 18
}
