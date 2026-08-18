package game

type WeaponDef struct {
	Number    int
	Name      string
	ROF       int
	Firepower float64
	Recoil    float64
	Bullets   int
	Shotgun   int
	Weight    float64
	ShellX    float64
	FlashX    float64
	SpriteDir string
}

var weaponCatalog = map[int]WeaponDef{
	1:  {1, "COOL PISTOL", 10, 20, 1, 9, 0, 1, 25, 45, "DefineSprite_416_gun_m1911"},
	2:  {2, "SAND HAWK", 13, 24, 2, 7, 0, .95, 20, 47, "DefineSprite_422_gun_deagle"},
	3:  {3, "SINE PISTOL", 6, 16, .7, 17, 0, 1, 25, 45, "DefineSprite_434_gun_glock"},
	4:  {4, "ANGRY COW", 16, 30, 3, 6, 0, .95, 25, 45, "DefineSprite_442_gun_bull"},
	5:  {5, "FIFTY EIGHT", 8, 18, 1, 16, 0, 1, 25, 45, "DefineSprite_451_gun_49"},
	6:  {6, "SNAKE", 22, 36, 5, 6, 0, .95, 25, 50, "DefineSprite_457_gun_python"},
	8:  {8, "COOL PISTOL INFINITY", 10, 20, 1, 9999, 0, 1, 25, 45, "DefineSprite_606_gun_m1911weak"},
	9:  {9, "DE- MATERIALIZER", 12, 25, 7, 5, 0, 1, 20, 65, "DefineSprite_461_gun_instagib"},
	10: {10, "CLASSIC ASSAULT RIFLE", 4, 22, .8, 30, 0, .8, 20, 75, "DefineSprite_463_gun_ak47"},
	11: {11, "COOL SNIPER", 31, 60, 10, 5, 0, .7, 20, 70, "DefineSprite_465_gun_hk"},
	12: {12, "SHORT SMG", 4, 17, .8, 30, 0, .95, 20, 45, "DefineSprite_467_gun_mp5k"},
	13: {13, "SAWED OFF SHOTGUN", 25, 8, 5, 7, 7, 0, 20, 55, "DefineSprite_471_gun_ithaca"},
	14: {14, "MILITARY SNIPER", 13, 58, 1.5, 8, 0, .7, 20, 72, "DefineSprite_473_gun_scar"},
	15: {15, "FAT SHOTGUN", 6, 5, .8, 12, 7, .75, 20, 60, "DefineSprite_475_gun_usas"},
	16: {16, "CLASSIC SNIPER", 25, 56, 8, 6, 0, .75, 20, 85, "DefineSprite_477_gun_501"},
	17: {17, "COMPACT RIFLE", 5, 30, .8, 15, 0, .85, 20, 75, "DefineSprite_114_gun_mini14"},
	18: {18, "LEVER SHOTGUN", 18, 8, 5, 7, 7, .8, 20, 55, "DefineSprite_481_gun_lever"},
	19: {19, "TACTICAL SMG", 3, 14, .3, 50, 0, .9, 5, 65, "DefineSprite_483_gun_p90"},
	20: {20, "BULLPUP ASSAULT RIFLE", 4, 22, .5, 30, 0, .8, -3, 58, "DefineSprite_485_gun_m17"},
	21: {21, "COMPACT SMG", 3, 14, .8, 48, 0, .9, 20, 50, "DefineSprite_487_gun_bizon"},
	22: {22, "MAFIA SMG", 4, 20, .8, 25, 0, .8, 20, 60, "DefineSprite_489_gun_tommy"},
	23: {23, "SLICK SMG", 4, 21, .8, 25, 0, .9, 25, 50, "DefineSprite_491_gun_ump"},
	24: {24, "BULLPUP SMG", 4, 20, .8, 30, 0, .9, 0, 53, "DefineSprite_493_gun_augsmg"},
	25: {25, "POLICE SMG", 4, 18, .8, 30, 0, .9, 20, 55, "DefineSprite_495_gun_mp5a3"},
	26: {26, "LOSER SMG", 4, 4, .2, 30, 0, .6, 25, 50, "DefineSprite_497_gun_lusa"},
	27: {27, "LIGHT SMG", 4, 15, .8, 32, 0, .9, 25, 50, "DefineSprite_499_gun_m12"},
	28: {28, "ANTIQUE SMG", 3, 15, .8, 32, 0, .9, 25, 55, "DefineSprite_501_gun_mas38"},
	29: {29, "COMPACT ASSAULT RIFLE", 4, 22, 1, 25, 0, .85, 25, 60, "DefineSprite_503_gun_aksu"},
	30: {30, "ADVANCED SMG", 3, 16, .2, 28, 0, .9, 25, 53, "DefineSprite_505_gun_kriss"},
	31: {31, "MICRO SMG", 3, 15, .5, 30, 0, .95, 17, 45, "DefineSprite_507_gun_mp9"},
	32: {32, "OLD SMG", 3, 15, .8, 32, 0, .9, 25, 55, "DefineSprite_509_gun_mas"},
	33: {33, "MODERN SNIPER", 31, 63, 10, 5, 0, .7, 25, 75, "DefineSprite_511_gun_awm"},
	34: {34, "HI-POWER SNIPER", 31, 70, 10, 5, 0, .65, 20, 85, "DefineSprite_513_gun_hecate"},
	35: {35, "BULLPUP SNIPER", 31, 63, 10, 5, 0, .7, -5, 85, "DefineSprite_515_gun_m95"},
	36: {36, "TACTICAL SNIPER", 31, 60, 10, 5, 0, .75, -5, 75, "DefineSprite_517_gun_dsr"},
	37: {37, "RELIABLE SNIPER", 25, 60, 10, 5, 0, .8, 25, 75, "DefineSprite_519_gun_ssg69"},
	38: {38, "ASSASSIN SNIPER", 25, 60, 10, 5, 0, .9, 25, 75, "DefineSprite_521_gun_lrs2"},
	39: {39, "RAPID SNIPER", 15, 51, 1.2, 7, 0, .8, 20, 77, "DefineSprite_523_gun_galatz"},
	40: {40, "STEALTH SNIPER", 15, 56, 1.2, 10, 0, .8, 20, 77, "DefineSprite_525_gun_vss"},
	41: {41, "RUGGED SNIPER", 12, 55, 1.2, 10, 0, .8, 20, 79, "DefineSprite_527_gun_m76"},
	42: {42, "AMERICAN SNIPER", 12, 55, 1.5, 8, 0, .7, 20, 79, "DefineSprite_529_gun_sr25"},
	43: {43, "LEVER SNIPER", 18, 60, 5, 7, 0, .8, 25, 60, "DefineSprite_531_gun_leversniper"},
	44: {44, "GANGSTER SHOTGUN", 40, 10, 8, 4, 7, .75, 20, 40, "DefineSprite_538_gun_sawnoff"},
	45: {45, "TACTICAL SHOTGUN", 24, 8, 5, 7, 7, .8, 0, 50, "DefineSprite_542_gun_ksg"},
	46: {46, "FULL AUTO SHOTGUN", 6, 6, .2, 12, 7, .75, 20, 60, "DefineSprite_544_gun_aa12"},
	47: {47, "PROTOTYPE SHOTGUN", 6, 8, .8, 10, 7, .7, -1, 62, "DefineSprite_546_gun_jackhammer"},
	48: {48, "AVERAGE SHOTGUN", 25, 7, 5, 10, 7, .8, 20, 55, "DefineSprite_550_gun_pm12"},
	49: {49, "CHROME SHOTGUN", 25, 8, 5, 7, 7, .8, 20, 50, "DefineSprite_554_gun_870"},
	50: {50, "ANTI AIRCRAFT GUN", 25, 9, 10, 5, 7, .7, 20, 58, "DefineSprite_558_gun_spas12"},
	51: {51, "ASSAULT SHOTGUN", 12, 7, 3, 8, 7, .75, 20, 63, "DefineSprite_560_gun_saiga12"},
	52: {52, "HUNTING SHOTGUN", 12, 6, 2, 7, 7, .75, 20, 63, "DefineSprite_562_gun_saiga20"},
	53: {53, "FUTURISTIC SHOTGUN", 25, 9, 5, 6, 7, .7, 20, 53, "DefineSprite_567_gun_spas97"},
	54: {54, "STUBBY SHOTGUN", 18, 6, 3, 5, 7, .9, 20, 38, "DefineSprite_112_gun_moss500"},
	55: {55, "MINIGUN", 2, 20, .5, 150, 0, .7, 20, 71, "DefineSprite_599_gun_mini"},
	56: {56, "SHORT ASSAULT RIFLE", 4, 20, .8, 30, 0, .8, 25, 57, "DefineSprite_573_gun_g36c"},
	57: {57, "CLASSIC MACHINE GUN", 3, 20, .8, 40, 0, .75, 15, 70, "DefineSprite_575_gun_rpk"},
	58: {58, "COVERT ASSAULT RIFLE", 5, 22, .8, 30, 0, .8, 21, 64, "DefineSprite_577_gun_sig552"},
	59: {59, "GUERILLA ASSAULT RIFLE", 5, 25, .8, 35, 0, .8, 20, 72, "DefineSprite_579_gun_galil"},
	60: {60, "ASSAULT CARBINE", 4, 20, .8, 30, 0, .85, 21, 62, "DefineSprite_582_gun_m4"},
	61: {61, "LONG RANGE RIFLE", 7, 32, 2, 20, 0, .75, 15, 66, "DefineSprite_584_gun_hk33"},
	62: {62, "FUTURISTIC ASSAULT RIFLE", 4, 25, .2, 30, 0, .8, 40, 56, "DefineSprite_586_gun_f2000"},
	63: {63, "HI-POWER RIFLE", 7, 34, 2, 20, 0, .75, 17, 65, "DefineSprite_588_gun_scarh"},
	64: {64, "TOP-LOAD MACHINE GUN", 3, 20, .8, 50, 0, .7, 19, 70, "DefineSprite_590_gun_stoner"},
	65: {65, "MILITARY MACHINE GUN", 3, 20, .8, 60, 0, .7, 19, 68, "DefineSprite_592_gun_m249"},
	66: {66, "BULLPUP MACHINE GUN", 3, 18, .8, 42, 0, .8, 0, 70, "DefineSprite_594_gun_hbar"},

	// Gun Mayhem 2 additions. GM2 deliberately keeps the original 1..66
	// numbering and appends these twenty weapons as 67..86, so they can share
	// the same gameplay catalog without remapping any GM1 save/runtime state.
	67: {67, "GREY SMG", 3, 15, .4, 30, 0, .9, 20, 55, "DefineSprite_573_gun_A1"},
	68: {68, "CHEAP SMG", 2, 12, .3, 32, 0, .9, 20, 55, "DefineSprite_575_gun_A2"},
	69: {69, "MINI SMG", 3, 16, .5, 30, 0, .95, 17, 45, "DefineSprite_577_gun_A3"},
	70: {70, "CHROME SMG", 4, 20, .6, 30, 0, .8, 21, 64, "DefineSprite_579_gun_A4"},
	71: {71, "FANCY SMG", 4, 21, .6, 30, 0, .9, 25, 50, "DefineSprite_581_gun_A5"},
	72: {72, "RELIABLE RIFLE", 5, 28, 1, 20, 0, .75, 17, 65, "DefineSprite_583_gun_B1"},
	73: {73, "ANTIQUE RIFLE", 5, 28, 1, 30, 0, .8, 20, 72, "DefineSprite_585_gun_B2"},
	74: {74, "BULLPUP STEALTH RIFLE", 4, 24, .4, 30, 0, .8, -3, 58, "DefineSprite_587_gun_B3"},
	75: {75, "PRECISION CARBINE", 5, 26, .4, 20, 0, .85, 21, 62, "DefineSprite_589_gun_B4"},
	76: {76, "OLD ASSAULT RIFLE", 3, 19, .6, 40, 0, .8, 21, 64, "DefineSprite_591_gun_B5"},
	77: {77, ".50 SNIPER", 18, 62, 1.5, 5, 0, .7, 20, 79, "DefineSprite_593_gun_C1"},
	78: {78, "STEADY SNIPER", 12, 48, 1.2, 10, 0, .8, -10, 77, "DefineSprite_595_gun_C2"},
	79: {79, "SINGLE SHOT SNIPER", 31, 70, 10, 1, 0, .7, 25, 75, "DefineSprite_597_gun_C3"},
	80: {80, "TACTICAL STEALTH SNIPER", 25, 55, 5, 5, 0, .75, -5, 75, "DefineSprite_599_gun_C4"},
	81: {81, "LIGHT SNIPER", 31, 50, 7, 5, 0, .8, 20, 70, "DefineSprite_601_gun_C5"},
	82: {82, "AUTOMATIC SHOTGUN", 6, 7, .8, 10, 7, .8, 20, 60, "DefineSprite_603_gun_D1"},
	83: {83, "HI-CAP SHOTGUN", 12, 7, .6, 12, 7, .75, 20, 60, "DefineSprite_605_gun_D2"},
	84: {84, "STANDARD SHOTGUN", 25, 8, 5, 5, 7, .8, 20, 58, "DefineSprite_609_gun_D3"},
	85: {85, "STREET SWEEPER", 10, 7, .5, 12, 7, .85, 20, 60, "DefineSprite_611_gun_D4"},
	86: {86, "4 ROUND SHOTGUN", 12, 8, 2, 4, 7, .85, 20, 63, "DefineSprite_613_gun_D5"},
}

func WeaponByNumber(number int) (WeaponDef, bool) {
	w, ok := weaponCatalog[number]
	return w, ok
}

func NewWeapon(number int) WeaponState {
	def, ok := WeaponByNumber(number)
	if !ok {
		def = weaponCatalog[1]
	}
	wait := def.ROF - 2
	if def.Number < 10 {
		wait = -10
	}
	timeline := weaponTimeline(def.Number)
	return WeaponState{Def: def, Bullets: def.Bullets, WaitTime: wait, Frame: timeline.RestFrame, Alpha: 1}
}

// Flash starts with all crate guns unlocked except four five-weapon groups.
func DefaultCrateWeapons() []int {
	locked := map[int]bool{}
	for _, index := range []int{18, 19, 20, 21, 22, 29, 30, 31, 32, 33, 40, 41, 42, 43, 44, 52, 53, 54, 55, 56} {
		locked[index+10] = true
	}
	out := make([]int, 0, 37)
	for n := 10; n <= 66; n++ {
		if !locked[n] {
			out = append(out, n)
		}
	}
	return out
}

// GM2CrateWeapons mirrors DefineSprite_784_crate: random(77)+10, i.e. every
// normal crate weapon from 10 through 86 inclusive. Keep this separate from
// DefaultCrateWeapons so the preserved GM1 campaign can retain its original
// unlock-driven crate pool while developed Custom Game uses the combined GM2
// arsenal.
func GM2CrateWeapons() []int {
	out := make([]int, 0, 77)
	for n := 10; n <= 86; n++ {
		out = append(out, n)
	}
	return out
}
