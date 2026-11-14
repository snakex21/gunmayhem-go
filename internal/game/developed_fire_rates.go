package game

import "math"

// Real-world rate of fire for the developed weapons 87-231.
//
// The 84 source weapons (1-86) keep their ActionScript values. The added
// firearms take their fire delay (Def.ROF, frames at 30 TPS) from the real
// weapon's rate of fire, mapped onto the game scale so that they land inside
// the band the source weapons occupy:
//
//	delay = round(K / RPM), never below 2 frames (the source minigun's floor)
//
// Delay is proportional to 1/RPM, so the real ratios between weapons are kept
// up to whole-frame rounding. K is calibrated per class as the geometric mean
// of ROF x real cyclic RPM over the source weapons of that class:
//
//	SMG          K 2800  (MP5K, MP5A3, P90, Bizon, Thompson, UMP45, AUG Para,
//	                      M12, MAS-38, Vector, MP9, Scorpion EVO, MAC-10, CPW,
//	                      K7, APC)
//	Rifle        K 2950  (AK-47, AKS-74U, G36C, SG 552, Galil, M4A1, F2000,
//	                      CETME, StG 44, Groza, Stoner 63)
//	Battle rifle K 4700  (HK33 and SCAR-H, the slow "long range" rifles)
//	Machine gun  K 2150  (RPK-74, Stoner 63A, M249, AUG HBAR)
//	Practical    K 1000  semi-auto, bolt, lever, pump and break actions use a
//	                     practical aimed rate instead of a cyclic one. 1000
//	                     puts a 40 rpm bolt action on 25 frames (Mauser 86,
//	                     SSG 69, R93 in the source), an 85 rpm marksman rifle
//	                     on 12 (SR-25, M76) and a 100 rpm .45 pistol on 10
//	                     (M1911).
//
// Burst weapons use the burst's cyclic rate for the gap between rounds
// (BurstRPM -> DevelopedWeaponSpec.BurstGap); their Def.ROF is the designed
// pause after the burst and is not RPM-derived unless RPM is set.
//
// Launchers, grenade launchers, flamethrowers, the Portal Gun, bows,
// crossbow, harpoon, nail/flare guns, energy rifle, musket, the one-round
// Tankgewehr/Martini-Henry/Single-Shot Sawed-Off and the KS-23 are not mapped:
// their pace is set by blast or single-round handling, not by a rate of fire.
type developedRateBasis int

const (
	rateCyclicSMG developedRateBasis = iota + 1
	rateCyclicRifle
	rateCyclicBattleRifle
	rateCyclicMG
	ratePractical
)

var developedRateConstant = map[developedRateBasis]float64{
	rateCyclicSMG:         2800,
	rateCyclicRifle:       2950,
	rateCyclicBattleRifle: 4700,
	rateCyclicMG:          2150,
	ratePractical:         1000,
}

// developedMinFireDelay is the fastest cadence in the game: the source M134
// minigun (55) fires every 2 frames.
const developedMinFireDelay = 2

func developedFireDelayForRPM(basis developedRateBasis, rpm float64) int {
	k, ok := developedRateConstant[basis]
	if !ok || rpm <= 0 {
		return 0
	}
	delay := int(math.Round(k / rpm))
	if delay < developedMinFireDelay {
		delay = developedMinFireDelay
	}
	return delay
}

type developedFireRate struct {
	Basis    developedRateBasis
	RPM      float64 // cyclic (automatic) or practical (manual) rate -> Def.ROF
	BurstRPM float64 // burst cyclic rate -> BurstGap (burst weapons only)
}

// developedFireRates lists the rate used for every mapped added weapon.
// Cyclic rates are the commonly published figures (middle of the range where
// sources give one); practical rates are aimed shots per minute by action.
var developedFireRates = map[int]developedFireRate{
	// Rifles (automatic, cyclic).
	91:  {Basis: rateCyclicRifle, RPM: 700},       // Remington ACR
	92:  {Basis: rateCyclicRifle, RPM: 625},       // FN SCAR-L (550-650)
	124: {Basis: rateCyclicRifle, RPM: 850},       // HK416 (700-900)
	129: {Basis: rateCyclicRifle, RPM: 900},       // AEK-971
	130: {Basis: rateCyclicRifle, RPM: 700},       // AK-12
	131: {Basis: rateCyclicRifle, RPM: 650},       // AK-74 (600-650)
	132: {Basis: rateCyclicRifle, RPM: 600},       // AKM
	133: {Basis: rateCyclicRifle, RPM: 650},       // QBZ-95
	134: {Basis: rateCyclicRifle, RPM: 850},       // Tavor TAR-21 (750-950)
	136: {Basis: rateCyclicRifle, RPM: 700},       // L85A2 (610-775)
	137: {Basis: rateCyclicRifle, RPM: 700},       // SIG SG 550
	138: {Basis: rateCyclicRifle, RPM: 750},       // XM8
	139: {Basis: rateCyclicRifle, RPM: 900},       // AS Val
	211: {Basis: rateCyclicRifle, RPM: 400},       // Fedorov Avtomat
	222: {Basis: rateCyclicRifle, RPM: 450},       // Ribeyrolles 1918 (approx.)
	125: {Basis: rateCyclicBattleRifle, RPM: 600}, // HK417
	126: {Basis: rateCyclicBattleRifle, RPM: 550}, // G3 (500-600)
	127: {Basis: rateCyclicBattleRifle, RPM: 650}, // FN FAL (650-700)

	// Burst rifles: BurstRPM -> gap between rounds.
	128: {Basis: rateCyclicRifle, BurstRPM: 1800}, // AN-94 2-round burst
	135: {Basis: rateCyclicRifle, BurstRPM: 1000}, // FAMAS F1 3-round burst
	213: {Basis: rateCyclicRifle, BurstRPM: 2100}, // G11 hyperburst

	// SMGs and machine pistols (cyclic).
	96:  {Basis: rateCyclicSMG, RPM: 1000}, // PPSh-41 (900-1250)
	97:  {Basis: rateCyclicSMG, RPM: 550},  // MP 40
	99:  {Basis: rateCyclicSMG, RPM: 1200}, // Glock 18C (1100-1200)
	113: {Basis: rateCyclicSMG, RPM: 950},  // MP7A1
	114: {Basis: rateCyclicSMG, RPM: 600},  // Uzi
	115: {Basis: rateCyclicSMG, RPM: 1250}, // Micro Uzi
	116: {Basis: rateCyclicSMG, RPM: 1200}, // MAC-11 (.380)
	117: {Basis: rateCyclicSMG, RPM: 850},  // Skorpion vz. 61
	118: {Basis: rateCyclicSMG, RPM: 700},  // PP-2000 (600-800)
	119: {Basis: rateCyclicSMG, RPM: 850},  // SIG MPX
	120: {Basis: rateCyclicSMG, RPM: 750},  // PP-19 Vityaz (700-800)
	121: {Basis: rateCyclicSMG, RPM: 550},  // Sterling L2A3
	122: {Basis: rateCyclicSMG, RPM: 550},  // Sten Mk II (500-600)
	123: {Basis: rateCyclicSMG, RPM: 600},  // PPS-43
	194: {Basis: rateCyclicSMG, RPM: 450},  // MP 18 (350-500)
	195: {Basis: rateCyclicSMG, RPM: 600},  // MP 28
	196: {Basis: rateCyclicSMG, RPM: 550},  // Erma EMP
	197: {Basis: rateCyclicSMG, RPM: 900},  // Automatico M1918 (OVP)
	202: {Basis: rateCyclicSMG, RPM: 600},  // Hellriegel 1915 (approx.)
	212: {Basis: rateCyclicSMG, RPM: 650},  // UMP9
	214: {Basis: rateCyclicSMG, RPM: 450},  // M3 Grease Gun
	217: {Basis: rateCyclicSMG, RPM: 450},  // Type 100 (early)
	223: {Basis: rateCyclicSMG, RPM: 750},  // SMG 08/18 (fictional, approx.)
	230: {Basis: rateCyclicSMG, RPM: 1500}, // Thompson "Annihilator" prototype

	// Machine guns and automatic rifles (cyclic).
	94:  {Basis: rateCyclicMG, RPM: 650},       // PKP Pecheneg
	95:  {Basis: rateCyclicMG, RPM: 700},       // RPD (650-750)
	140: {Basis: rateCyclicMG, RPM: 650},       // PKM
	141: {Basis: rateCyclicMG, RPM: 1200},      // MG42
	142: {Basis: rateCyclicMG, RPM: 1100},      // MG3 (1000-1300)
	143: {Basis: rateCyclicMG, RPM: 550},       // M60
	144: {Basis: rateCyclicMG, RPM: 750},       // M240B (650-950)
	145: {Basis: rateCyclicMG, RPM: 400},       // M1918A2 BAR, slow-rate setting
	146: {Basis: rateCyclicMG, RPM: 550},       // DP-28
	147: {Basis: rateCyclicMG, RPM: 520},       // Bren Mk II
	148: {Basis: rateCyclicMG, RPM: 550},       // Lewis Gun
	149: {Basis: rateCyclicMG, RPM: 600},       // Ultimax 100
	199: {Basis: rateCyclicMG, RPM: 500},       // Maxim MG 08/15
	200: {Basis: rateCyclicMG, RPM: 500},       // MG 15 n.A.
	201: {Basis: rateCyclicMG, RPM: 240},       // Chauchat
	203: {Basis: rateCyclicMG, RPM: 550},       // M1918 BAR (500-650)
	220: {Basis: rateCyclicMG, RPM: 500},       // Huot (475-600)
	221: {Basis: rateCyclicMG, RPM: 450},       // Madsen
	198: {Basis: rateCyclicMG, BurstRPM: 1500}, // Villar Perosa twin barrels

	// Pistols (practical aimed rate).
	98:  {Basis: ratePractical, RPM: 140}, // FN Five-seveN
	100: {Basis: ratePractical, RPM: 125}, // Beretta 92FS
	101: {Basis: ratePractical, RPM: 125}, // SIG P226
	102: {Basis: ratePractical, RPM: 125}, // CZ 75
	103: {Basis: ratePractical, RPM: 110}, // USP .45
	104: {Basis: ratePractical, RPM: 100}, // Mark 23
	105: {Basis: ratePractical, RPM: 140}, // Walther P99
	106: {Basis: ratePractical, RPM: 110}, // Browning Hi-Power
	107: {Basis: ratePractical, RPM: 110}, // Tokarev TT-33
	108: {Basis: ratePractical, RPM: 110}, // Makarov PM
	109: {Basis: ratePractical, RPM: 100}, // Luger P08
	110: {Basis: ratePractical, RPM: 110}, // Mauser C96
	111: {Basis: ratePractical, RPM: 60},  // S&W Model 29 (.44 Magnum)
	112: {Basis: ratePractical, RPM: 80},  // Mateba 6 Unica
	190: {Basis: ratePractical, RPM: 100}, // Lancaster Howdah
	206: {Basis: ratePractical, RPM: 110}, // Colt Model 1903
	207: {Basis: ratePractical, RPM: 125}, // Frommer Stop
	216: {Basis: ratePractical, RPM: 110}, // Nambu Type 14
	219: {Basis: ratePractical, RPM: 100}, // AF2011-A1: pulls per minute; both barrels fire together in reality
	231: {Basis: ratePractical, RPM: 45},  // Peacekeeper single-action revolver

	// Semi-auto rifles (practical).
	93:  {Basis: ratePractical, RPM: 100}, // M14 EBR
	153: {Basis: ratePractical, RPM: 100}, // M1 Garand
	154: {Basis: ratePractical, RPM: 140}, // M1 Carbine
	157: {Basis: ratePractical, RPM: 100}, // M14
	158: {Basis: ratePractical, RPM: 85},  // Dragunov SVD
	159: {Basis: ratePractical, RPM: 55},  // Barrett M82A1
	161: {Basis: ratePractical, RPM: 75},  // Walther WA 2000
	208: {Basis: ratePractical, RPM: 110}, // Gewehr 43
	209: {Basis: ratePractical, RPM: 140}, // SKS
	210: {Basis: ratePractical, RPM: 120}, // SVT-40
	218: {Basis: ratePractical, RPM: 110}, // M1941 Johnson
	224: {Basis: ratePractical, RPM: 90},  // RSC 1917
	225: {Basis: ratePractical, RPM: 110}, // Mondragon M1908
	226: {Basis: ratePractical, RPM: 100}, // General Liu

	// Bolt and lever actions (practical).
	150: {Basis: ratePractical, RPM: 40}, // Karabiner 98k
	151: {Basis: ratePractical, RPM: 40}, // Mosin-Nagant
	152: {Basis: ratePractical, RPM: 50}, // Lee-Enfield (cock-on-close bolt)
	155: {Basis: ratePractical, RPM: 37}, // M24 SWS
	156: {Basis: ratePractical, RPM: 37}, // M40A5
	160: {Basis: ratePractical, RPM: 32}, // CheyTac M200
	186: {Basis: ratePractical, RPM: 40}, // Carcano
	191: {Basis: ratePractical, RPM: 40}, // Gewehr 98
	205: {Basis: ratePractical, RPM: 40}, // Springfield M1903
	215: {Basis: ratePractical, RPM: 40}, // Type 99 Arisaka
	227: {Basis: ratePractical, RPM: 50}, // Ross Mk III (straight pull)
	228: {Basis: ratePractical, RPM: 50}, // Russian 1895 (lever)

	// Shotguns (practical).
	162: {Basis: ratePractical, RPM: 85},  // Benelli M4 (semi-auto)
	163: {Basis: ratePractical, RPM: 50},  // Winchester 1897 (pump)
	164: {Basis: ratePractical, RPM: 55},  // Winchester 1887 (lever)
	166: {Basis: ratePractical, RPM: 100}, // DP-12 (two shots per pump)
	167: {Basis: ratePractical, RPM: 55},  // M26 MASS (straight-pull)
	204: {Basis: ratePractical, RPM: 85},  // Sjogren inertial (semi-auto)
	229: {Basis: ratePractical, RPM: 36},  // Model 1900 double barrel
}
