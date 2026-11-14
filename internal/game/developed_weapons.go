package game

import (
	"image"
	"math"
	"sort"
	"strconv"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	xdraw "golang.org/x/image/draw"
)

type DevelopedWeaponCategory int

const (
	DevelopedCategoryPistol DevelopedWeaponCategory = iota + 1
	DevelopedCategorySMG
	DevelopedCategoryRifle
	DevelopedCategoryLMG
	DevelopedCategorySniper
	DevelopedCategoryShotgun
	DevelopedCategoryLauncher
	DevelopedCategorySpecial
)

type DevelopedFireMode int

const (
	DevelopedFireAuto DevelopedFireMode = iota
	DevelopedFireSemi
	DevelopedFireBurst2
	DevelopedFireBurst3
)

type DevelopedProjectileKind int

const (
	DevelopedProjectileBullet DevelopedProjectileKind = iota
	DevelopedProjectileRocket
	DevelopedProjectileArcRocket
	DevelopedProjectileTopAttack
	DevelopedProjectileGrenade
	DevelopedProjectileStrike
	DevelopedProjectileBolt
	DevelopedProjectileFlare
	DevelopedProjectileEnergy
	DevelopedProjectileFlame
	DevelopedProjectilePortal
)

type DevelopedWeaponSpec struct {
	Number            int
	AssetDir          string
	Category          DevelopedWeaponCategory
	FireMode          DevelopedFireMode
	Projectile        DevelopedProjectileKind
	ProjectileSpeed   float64
	ProjectileGravity float64
	Spread            float64
	ExplosionRadius   float64
	ExplosionPower    float64
	Lifetime          int
	IgniteFrames      int
	// PelletFanStep > 0 replaces the source shotgun fan (2 degrees per pellet,
	// slightly biased downward) with a symmetric fan of this many degrees per
	// pellet. Used for wide-choke heavy shotguns that must fall off with range.
	PelletFanStep float64
	// BurstGap is the number of frames between the rounds of a Burst2/Burst3
	// weapon (derived from the real burst cyclic rate, never below 2). The
	// weapon's Def.ROF is the pause after the last round of the burst.
	BurstGap int
	// SemiBurst makes a burst weapon fire one burst per trigger press, like a
	// semi-auto (AF2011-A1: both barrels per pull).
	SemiBurst bool
}

// developedBurstRounds returns the rounds per trigger of a developed weapon
// (1 for everything that is not a burst weapon).
func developedBurstRounds(spec DevelopedWeaponSpec) int {
	switch spec.FireMode {
	case DevelopedFireBurst2:
		return 2
	case DevelopedFireBurst3:
		return 3
	}
	return 1
}

// developedBurstGap returns the frames between two rounds of a burst. Rounds
// must leave on separate frames so they are seen and heard as a burst.
func developedBurstGap(spec DevelopedWeaponSpec) int {
	if spec.BurstGap < 2 {
		return 2
	}
	return spec.BurstGap
}

// developedTriggerResetRequired reports whether the trigger must be released
// between shots (semi-auto, bolt, pump, lever, single-shot and launchers, plus
// per-pull bursts).
func developedTriggerResetRequired(spec DevelopedWeaponSpec) bool {
	return spec.FireMode == DevelopedFireSemi || spec.SemiBurst
}

// setDevelopedBurst configures a burst weapon after it has been added.
func setDevelopedBurst(number, gap int, spread float64, semi bool) {
	spec := developedWeaponSpecs[number]
	spec.BurstGap = gap
	spec.Spread = spread
	spec.SemiBurst = semi
	developedWeaponSpecs[number] = spec
}

var developedWeaponCatalog = map[int]WeaponDef{}
var developedWeaponSpecs = map[int]DevelopedWeaponSpec{}
var customWeaponAssetsOnce sync.Once
var customWeaponAssets = map[int]bool{}

// Highest developed weapon ID is 231; index zero stays unused in the runtime
// selection/blocklist arrays.
const developedWeaponSlotCount = 232

func init() {
	// Recoil (the shooter's own push per round) never exceeds the strongest
	// source weapon of the same class: held fire averages at most 0.267
	// px/frame for SMGs and machine guns (RPK/M249 0.8 every 3 frames, the
	// minigun 0.25), 0.286 for rifles, 0.227 for pistols (Snake), 0.4 for
	// snipers, shotguns and launchers, and one shot kicks at most 10 (source
	// snipers / Anti Aircraft Gun). Heavy guns sit at the top of their class.
	// See developed_recoil_test.go.
	//
	// Launchers / special heavy weapons.
	// Pickups carry exactly one magazine (no reload): disposable/heavy rockets
	// get a single round, reloadable tubes two. ExplosionPower is the full
	// impulse of a direct hit; RPG-7 (120) is deliberately above every sniper
	// (originals max 70+3, Tankgewehr 92) and the KS-23 point-blank load (112).
	addLauncher(87, "RPG-7", "rpg7", 28, 42, 10, 1, DevelopedProjectileRocket, 17, 0, 190, 120)
	addLauncher(88, "PIAT", "piat", 60, 44, 10, 2, DevelopedProjectileArcRocket, 12, 0.55, 205, 95)
	addLauncher(89, "FGM-148 Javelin", "javelin", 45, 48, 10, 1, DevelopedProjectileTopAttack, 14, 0, 225, 90)
	addLauncher(90, "HIMARS Strike", "himars_strike", 70, 55, 2, 1, DevelopedProjectileStrike, himarsStrikeFallSpeed, 0, 285, 110)

	// Modern rifles requested for the expansion.
	addRifle(91, "Remington ACR", "remington_acr", 4, 23, .65, 30, DevelopedFireAuto)
	addRifle(92, "FN SCAR-L", "scar_l", 5, 24, .7, 30, DevelopedFireAuto)
	addRifle(93, "M14 EBR", "m14_ebr", 10, 38, 2.4, 20, DevelopedFireSemi)

	// Machine guns requested up front.
	addLMG(94, "PKP Pecheneg", "pkp_pecheneg", 3, 20, .8, 60)
	addLMG(95, "RPD", "rpd", 3, 20, .75, 60)

	// WWII SMGs requested up front.
	addSMG(96, "PPSh-41", "ppsh41", 3, 12, .55, 71, DevelopedFireAuto)
	addSMG(97, "MP 40", "mp40", 5, 19, .75, 32, DevelopedFireAuto)

	// Pistols. ID 3 remains source-faithful internally but is displayed as Glock 17.
	addPistol(98, "FN Five-seveN", "fn_fiveseven", 7, 16, .55, 20, DevelopedFireSemi)
	addPistol(99, "Glock 18C", "glock18c", 2, 8, .45, 17, DevelopedFireAuto)
	addPistol(100, "Beretta 92FS", "beretta_92fs", 8, 19, .8, 15, DevelopedFireSemi)
	addPistol(101, "SIG Sauer P226", "sig_p226", 8, 19, .8, 15, DevelopedFireSemi)
	addPistol(102, "CZ 75", "cz75", 8, 18, .75, 16, DevelopedFireSemi)
	addPistol(103, "H&K USP .45", "hk_usp45", 9, 22, 1.0, 12, DevelopedFireSemi)
	addPistol(104, "H&K Mark 23", "hk_mark23", 10, 24, 1.1, 12, DevelopedFireSemi)
	addPistol(105, "Walther P99", "walther_p99", 7, 17, .7, 16, DevelopedFireSemi)
	addPistol(106, "Browning Hi-Power", "browning_hi_power", 9, 20, .85, 13, DevelopedFireSemi)
	addPistol(107, "Tokarev TT-33", "tt33", 9, 22, .9, 8, DevelopedFireSemi)
	addPistol(108, "Makarov PM", "makarov_pm", 9, 17, .65, 8, DevelopedFireSemi)
	addPistol(109, "Luger P08", "luger_p08", 10, 21, .9, 8, DevelopedFireSemi)
	addPistol(110, "Mauser C96", "mauser_c96", 9, 20, .85, 10, DevelopedFireSemi)
	addPistol(111, "S&W Model 29", "sw_model29", 17, 34, 3.8, 6, DevelopedFireSemi)
	addPistol(112, "Mateba 6 Unica", "mateba_6_unica", 13, 31, 2.9, 6, DevelopedFireSemi)

	// SMG / PDW expansion.
	addSMG(113, "H&K MP7A1", "hk_mp7", 3, 15, .45, 40, DevelopedFireAuto)
	addSMG(114, "Uzi", "uzi", 5, 19, .8, 32, DevelopedFireAuto)
	addSMG(115, "Micro Uzi", "micro_uzi", 2, 12, .5, 32, DevelopedFireAuto)
	addSMG(116, "MAC-11", "mac11", 2, 11, .5, 32, DevelopedFireAuto)
	addSMG(117, "Skorpion vz. 61", "vz61", 3, 13, .5, 20, DevelopedFireAuto)
	addSMG(118, "PP-2000", "pp2000", 4, 16, .5, 20, DevelopedFireAuto)
	addSMG(119, "SIG MPX", "sig_mpx", 3, 17, .45, 30, DevelopedFireAuto)
	addSMG(120, "PP-19 Vityaz", "pp19_vityaz", 4, 18, .6, 30, DevelopedFireAuto)
	addSMG(121, "Sterling L2A3", "sterling_l2a3", 5, 19, .7, 34, DevelopedFireAuto)
	addSMG(122, "Sten Mk II", "sten_mk2", 5, 19, .8, 32, DevelopedFireAuto)
	addSMG(123, "PPS-43", "pps43", 5, 19, .7, 35, DevelopedFireAuto)

	// Assault rifles / battle rifles.
	addRifle(124, "H&K HK416", "hk416", 3, 18, .65, 30, DevelopedFireAuto)
	addRifle(125, "H&K HK417", "hk417", 8, 36, 1.5, 20, DevelopedFireAuto)
	addRifle(126, "H&K G3", "hk_g3", 9, 36, 2.0, 20, DevelopedFireAuto)
	addRifle(127, "FN FAL", "fn_fal", 7, 34, 2.0, 20, DevelopedFireAuto)
	addRifle(128, "AN-94", "an94", 6, 20, .65, 30, DevelopedFireBurst2)
	addRifle(129, "AEK-971", "aek971", 3, 18, .6, 30, DevelopedFireAuto)
	addRifle(130, "AK-12", "ak12", 4, 23, .7, 30, DevelopedFireAuto)
	addRifle(131, "AK-74", "ak74", 5, 22, .8, 30, DevelopedFireAuto)
	addRifle(132, "AKM", "akm", 5, 25, 1.0, 30, DevelopedFireAuto)
	addRifle(133, "QBZ-95", "qbz95", 5, 22, .55, 30, DevelopedFireAuto)
	addRifle(134, "IWI Tavor TAR-21", "tavor_tar21", 3, 18, .55, 30, DevelopedFireAuto)
	addRifle(135, "FAMAS F1", "famas_f1", 6, 20, .6, 25, DevelopedFireBurst3)
	addRifle(136, "L85A2", "l85a2", 4, 23, .65, 30, DevelopedFireAuto)
	addRifle(137, "SIG SG 550", "sig_sg550", 4, 24, .65, 30, DevelopedFireAuto)
	addRifle(138, "H&K XM8", "hk_xm8", 4, 22, .55, 30, DevelopedFireAuto)
	addRifle(139, "AS Val", "as_val", 3, 18, .45, 20, DevelopedFireAuto)

	// Machine-gun family.
	addLMG(140, "PKM", "pkm", 3, 20, .8, 60)
	addLMG(141, "MG42", "mg42", 2, 13, .35, 50)
	addLMG(142, "MG3", "mg3", 2, 13, .45, 50)
	addLMG(143, "M60", "m60", 4, 24, 1.0, 60)
	addLMG(144, "M240B", "m240b", 3, 20, .8, 60)
	addLMG(145, "M1918A2 BAR", "m1918_bar", 5, 30, 1.3, 20)
	addLMG(146, "DP-28", "dp28", 4, 25, 1.0, 47)
	addLMG(147, "Bren Mk II", "bren_mk2", 4, 25, 1.0, 30)
	addLMG(148, "Lewis Gun", "lewis_gun", 4, 24, 1.0, 47)
	addLMG(149, "Ultimax 100", "ultimax100", 4, 24, .6, 60)

	// WWII / marksman / sniper expansion.
	addSniper(150, "Karabiner 98k", "kar98k", 25, 56, 7.0, 5, DevelopedFireSemi)
	addSniper(151, "Mosin-Nagant M91/30", "mosin_nagant", 25, 56, 7.0, 5, DevelopedFireSemi)
	addSniper(152, "Lee-Enfield No.4 Mk I", "lee_enfield", 20, 53, 6.0, 10, DevelopedFireSemi)
	addSniper(153, "M1 Garand", "m1_garand", 10, 37, 2.4, 8, DevelopedFireSemi)
	addRifle(154, "M1 Carbine", "m1_carbine", 7, 27, 1.1, 15, DevelopedFireSemi)
	addSniper(155, "M24 SWS", "m24_sws", 27, 61, 8.0, 5, DevelopedFireSemi)
	addSniper(156, "M40A5", "m40a5", 27, 62, 8.0, 5, DevelopedFireSemi)
	addSniper(157, "M14", "m14", 10, 37, 2.3, 20, DevelopedFireSemi)
	addSniper(158, "Dragunov SVD", "svd", 12, 49, 2.0, 10, DevelopedFireSemi)
	addSniper(159, "Barrett M82A1", "barrett_m82a1", 18, 68, 7.0, 10, DevelopedFireSemi)
	addSniper(160, "CheyTac M200", "cheytac_m200", 31, 72, 10.0, 7, DevelopedFireSemi)
	addSniper(161, "Walther WA 2000", "wa2000", 13, 54, 2.0, 6, DevelopedFireSemi)

	// Shotguns.
	addShotgun(162, "Benelli M4", "benelli_m4", 12, 7.5, 2.5, 7, 7)
	addShotgun(163, "Winchester Model 1897", "winchester_1897", 20, 8.5, 5.0, 5, 7)
	addShotgun(164, "Winchester Model 1887", "winchester_1887", 18, 8.0, 5.0, 5, 7)
	// KS-23: heaviest pump shotgun. 7x16 point blank (112) stays below the
	// RPG-7 direct hit; the slow pump, 3 shells and a wide 5-degree fan make it
	// a close-range knockout that thins out quickly with distance.
	addShotgun(165, "KS-23", "ks23", 32, 16.0, 10.0, 3, 7)
	ks23 := developedWeaponSpecs[165]
	ks23.PelletFanStep = 5
	developedWeaponSpecs[165] = ks23
	addShotgun(166, "DP-12", "dp12", 10, 7.5, 4.0, 14, 7)
	addShotgun(167, "M26 MASS", "m26_mass", 18, 8.0, 4.0, 5, 7)

	// More launchers.
	addLauncher(168, "M72 LAW", "m72_law", 35, 43, 9, 1, DevelopedProjectileRocket, 18, 0, 185, 100)
	addLauncher(169, "AT4", "at4", 40, 47, 10, 1, DevelopedProjectileRocket, 19, 0, 205, 105)
	addLauncher(170, "Carl Gustaf M4", "carl_gustaf_m4", 60, 46, 10, 2, DevelopedProjectileRocket, 18, 0, 200, 90)
	addLauncher(171, "Panzerfaust", "panzerfaust", 42, 48, 10, 1, DevelopedProjectileArcRocket, 13, .35, 215, 105)
	addLauncher(172, "Panzerschreck", "panzerschreck", 60, 48, 10, 2, DevelopedProjectileRocket, 15, 0, 215, 95)
	addGrenadeLauncher(173, "M79 Grenade Launcher", "m79", 40, 36, 7, 2, 13)
	addGrenadeLauncher(174, "M203 Grenade Launcher", "m203", 40, 34, 6, 2, 14)
	addGrenadeLauncher(175, "M32 MGL", "m32_mgl", 12, 31, 4, 6, 14)
	addGrenadeLauncher(176, "China Lake Grenade Launcher", "china_lake", 18, 34, 5, 4, 14)

	// Deliberately unusual weapons: useful for making the expansion feel like
	// Gun Mayhem instead of just a catalogue of near-identical rifles.
	addSpecialProjectile(177, "Crossbow", "crossbow", 18, 40, 3, 6, DevelopedProjectileBolt, 20, .16)
	addSpecialProjectile(178, "Compound Bow", "compound_bow", 16, 37, 2, 8, DevelopedProjectileBolt, 18, .28)
	addSpecialProjectile(179, "Harpoon Gun", "harpoon_gun", 28, 78, 8, 1, DevelopedProjectileBolt, 22, .06)
	addSpecialProjectile(180, "Nail Gun", "nail_gun", 3, 12, .25, 40, DevelopedProjectileBolt, 23, .03)
	addSpecialProjectile(181, "Flare Gun", "flare_gun", 30, 30, 3, 4, DevelopedProjectileFlare, 14, .20)
	addSpecialProjectile(182, "Musket", "musket", 38, 64, 10, 1, DevelopedProjectileBullet, 30, 0)
	addShotgun(183, "Single-Shot Sawed-Off", "single_sawedoff", 44, 11.0, 10.0, 2, 8)
	addGrenadeLauncher(184, "Improvised Grenade Launcher", "improvised_launcher", 45, 38, 9, 2, 11)
	addSpecialProjectile(185, "Energy Rifle", "energy_rifle", 5, 26, .2, 24, DevelopedProjectileEnergy, 34, 0)
	addSniper(186, "Carcano M91/38", "carcano_m91_38", 25, 55, 6.5, 6, DevelopedFireSemi)

	// Developed special weapons. Flamethrowers use short-lived flame particles;
	// the Portal Gun alternates a blue/orange surface portal for its owner.
	// A flame particle is emitted every tick, so its direct push must be much
	// lower than a conventional bullet. Most of the readable impact comes from
	// the short lingering burn instead of stacking full-strength hits every
	// 30 Hz tick.
	addFlamethrower(187, "M2 Flamethrower", "m2_flamethrower", 1, 1.45, .18, 220, 13, 100)
	addFlamethrower(188, "Flammenwerfer 35", "flammenwerfer35", 1, 1.8, .22, 190, 12, 115)
	addPortalGun(189, "Portal Gun", "portal_gun")

	// WWI / early automatic-arms collection and requested historical/modern
	// follow-up. Existing M249 SAW (65), UMP45 (23) and Browning Auto-5 (86)
	// remain the original source weapons and are intentionally not duplicated.
	// The Lancaster is a four-barrel Howdah pistol, not a miniature shotgun.
	// Each trigger pull empties one chamber with a single heavy projectile.
	addDevelopedWeapon(developedDef(190, "Lancaster Howdah Pistol", "lancaster_howdah", 10, 32, 2.2, 4, 0, .95), DevelopedWeaponSpec{
		AssetDir: "lancaster_howdah", Category: DevelopedCategoryPistol, FireMode: DevelopedFireSemi,
		Projectile: DevelopedProjectileBullet, ProjectileSpeed: 22, Spread: 2.5, Lifetime: 80,
	})
	addSniper(191, "Gewehr 98", "gewehr98", 25, 58, 7, 5, DevelopedFireSemi)
	addSniper(192, "Tankgewehr M1918", "tankgewehr_m1918", 60, 92, 10, 1, DevelopedFireSemi)
	addSniper(193, "Martini-Henry", "martini_henry", 32, 68, 10, 3, DevelopedFireSemi)
	addSMG(194, "MP 18", "mp18", 6, 23, .8, 32, DevelopedFireAuto)
	addSMG(195, "MP 28", "mp28", 5, 19, .75, 32, DevelopedFireAuto)
	addSMG(196, "Erma EMP", "erma_emp", 5, 19, .7, 32, DevelopedFireAuto)
	addSMG(197, "Automatico M1918", "automatico_m1918", 3, 12, .55, 25, DevelopedFireAuto)
	addDevelopedWeapon(developedDef(198, "Villar Perosa M1915", "villar_perosa", 4, 15, .5, 50, 0, .62), DevelopedWeaponSpec{
		AssetDir: "villar_perosa", Category: DevelopedCategoryLMG, FireMode: DevelopedFireBurst2,
		Projectile: DevelopedProjectileBullet, ProjectileSpeed: 25, Spread: 4.5, Lifetime: 85,
	})
	addLMG(199, "Maxim MG 08/15", "maxim_mg0815", 4, 24, 1.0, 60)
	addLMG(200, "MG 15 n.A.", "mg15_na", 4, 24, 1.0, 60)
	addLMG(201, "Chauchat M1915", "chauchat_m1915", 9, 36, 1.4, 20)
	addSMG(202, "Hellriegel 1915", "hellriegel_1915", 5, 19, .65, 60, DevelopedFireAuto)
	addLMG(203, "Browning Automatic Rifle M1918", "bar_m1918", 4, 26, 1.0, 20)
	addShotgun(204, "Sjögren Inertial", "sjogren_inertial", 12, 6.8, 2.0, 5, 7)
	addSniper(205, "Springfield M1903", "springfield_m1903", 25, 58, 7, 5, DevelopedFireSemi)
	addPistol(206, "Colt Model 1903 Hammerless", "colt_1903_hammerless", 9, 18, .7, 8, DevelopedFireSemi)
	addPistol(207, "Frommer Stop", "frommer_stop", 8, 16, .5, 8, DevelopedFireSemi)
	addRifle(208, "Gewehr 43", "gewehr43", 9, 31, 1.4, 10, DevelopedFireSemi)
	addRifle(209, "SKS", "sks", 7, 29, 1.1, 10, DevelopedFireSemi)
	addRifle(210, "SVT-40", "svt40", 8, 32, 1.4, 10, DevelopedFireSemi)
	addRifle(211, "Fedorov Avtomat", "fedorov_avtomat", 7, 28, .9, 25, DevelopedFireAuto)
	addSMG(212, "H&K UMP9", "ump9", 4, 18, .65, 30, DevelopedFireAuto)
	addRifle(213, "H&K G11", "hk_g11", 6, 16, .35, 45, DevelopedFireBurst3)
	addSMG(214, "M3 Grease Gun", "m3_grease_gun", 6, 23, .8, 30, DevelopedFireAuto)
	addSniper(215, "Type 99 Arisaka", "type99_arisaka", 25, 58, 7, 5, DevelopedFireSemi)
	addPistol(216, "Nambu Type 14", "nambu_type14", 9, 16, .65, 8, DevelopedFireSemi)
	addSMG(217, "Type 100", "type100", 6, 23, .65, 30, DevelopedFireAuto)
	addRifle(218, "M1941 Johnson Rifle", "m1941_johnson", 9, 31, 1.2, 10, DevelopedFireSemi)
	addPistol(219, "AF2011-A1 Double Barrel", "af2011a1", 10, 13, 1.3, 16, DevelopedFireBurst2)

	// Particularly recognizable BF1 omissions suggested alongside the request.
	addLMG(220, "Huot Automatic Rifle", "huot_automatic", 4, 24, 1.0, 25)
	addLMG(221, "Madsen MG", "madsen_mg", 5, 30, 1.1, 30)
	addRifle(222, "Ribeyrolles 1918", "ribeyrolles_1918", 7, 31, .8, 25, DevelopedFireAuto)
	addSMG(223, "SMG 08/18", "smg_0818", 4, 16, .65, 50, DevelopedFireAuto)
	addRifle(224, "RSC 1917", "rsc_1917", 11, 35, 2.0, 6, DevelopedFireSemi)
	addRifle(225, "Mondragón M1908", "mondragon_m1908", 9, 31, 1.3, 10, DevelopedFireSemi)
	addRifle(226, "General Liu Rifle", "general_liu", 10, 32, 1.4, 6, DevelopedFireSemi)
	addSniper(227, "Ross Mk III", "ross_mk3", 20, 55, 6.5, 5, DevelopedFireSemi)
	addSniper(228, "Russian 1895", "russian_1895", 20, 54, 5.5, 5, DevelopedFireSemi)
	addShotgun(229, "Model 1900", "model_1900", 28, 9.5, 7.0, 2, 8)
	addSMG(230, "Annihilator", "annihilator", 2, 12, .5, 20, DevelopedFireAuto)
	addPistol(231, "Peacekeeper", "peacekeeper", 22, 36, 5.0, 6, DevelopedFireSemi)

	// Burst weapons: rounds leave BurstGap frames apart (from the real burst
	// cyclic rate, see developed_fire_rates.go), Def.ROF above is the pause
	// after the burst, per-round knockback sits below a full-auto rifle of the
	// same cartridge and the spread is tighter. Average push over a burst
	// cycle is about 5 px/frame, inside the original assault-rifle band.
	setDevelopedBurst(128, 2, 1.0, false) // AN-94: 2 rounds at 1800 rpm
	setDevelopedBurst(135, 3, 1.0, false) // FAMAS F1: 3 rounds at ~1000 rpm
	setDevelopedBurst(213, 2, 1.0, false) // G11: 3-round hyperburst at ~2100 rpm
	setDevelopedBurst(198, 2, 3.0, false) // Villar Perosa: twin barrels
	setDevelopedBurst(219, 2, 1.5, true)  // AF2011-A1: both barrels per pull

	// 145 and 203 were the same rifle. 145 (bipod sprite) is now the WWII
	// M1918A2 on its slow-rate setting: slower, harder-hitting per shot and
	// steadier off the bipod. 203 stays the original WWI M1918.
	bar := developedWeaponSpecs[145]
	bar.Spread = 2.5
	developedWeaponSpecs[145] = bar
}

func addDevelopedWeapon(def WeaponDef, spec DevelopedWeaponSpec) {
	if muzzle, ok := developedWeaponMuzzles[def.Number]; ok {
		def.FlashX = muzzle[0]
	}
	developedWeaponCatalog[def.Number] = def
	spec.Number = def.Number
	developedWeaponSpecs[def.Number] = spec
}

func developedDef(number int, name, asset string, rof int, firepower, recoil float64, bullets, shotgun int, weight float64) WeaponDef {
	return WeaponDef{
		Number: number, Name: name, ROF: rof, Firepower: firepower, Recoil: recoil,
		Bullets: bullets, Shotgun: shotgun, Weight: weight, ShellX: 20, FlashX: 58,
		SpriteDir: asset,
	}
}

func addPistol(number int, name, asset string, rof int, firepower, recoil float64, bullets int, mode DevelopedFireMode) {
	addDevelopedWeapon(developedDef(number, name, asset, rof, firepower, recoil, bullets, 0, .95), DevelopedWeaponSpec{
		AssetDir: asset, Category: DevelopedCategoryPistol, FireMode: mode,
		Projectile: DevelopedProjectileBullet, ProjectileSpeed: 25, Spread: 2.5, Lifetime: 90,
	})
}

func addSMG(number int, name, asset string, rof int, firepower, recoil float64, bullets int, mode DevelopedFireMode) {
	addDevelopedWeapon(developedDef(number, name, asset, rof, firepower, recoil, bullets, 0, .9), DevelopedWeaponSpec{
		AssetDir: asset, Category: DevelopedCategorySMG, FireMode: mode,
		Projectile: DevelopedProjectileBullet, ProjectileSpeed: 25, Spread: 3.5, Lifetime: 90,
	})
}

func addRifle(number int, name, asset string, rof int, firepower, recoil float64, bullets int, mode DevelopedFireMode) {
	addDevelopedWeapon(developedDef(number, name, asset, rof, firepower, recoil, bullets, 0, .8), DevelopedWeaponSpec{
		AssetDir: asset, Category: DevelopedCategoryRifle, FireMode: mode,
		Projectile: DevelopedProjectileBullet, ProjectileSpeed: 27, Spread: 2.5, Lifetime: 100,
	})
}

func addLMG(number int, name, asset string, rof int, firepower, recoil float64, bullets int) {
	addDevelopedWeapon(developedDef(number, name, asset, rof, firepower, recoil, bullets, 0, .7), DevelopedWeaponSpec{
		AssetDir: asset, Category: DevelopedCategoryLMG, FireMode: DevelopedFireAuto,
		Projectile: DevelopedProjectileBullet, ProjectileSpeed: 26, Spread: 4, Lifetime: 100,
	})
}

func addSniper(number int, name, asset string, rof int, firepower, recoil float64, bullets int, mode DevelopedFireMode) {
	addDevelopedWeapon(developedDef(number, name, asset, rof, firepower, recoil, bullets, 0, .72), DevelopedWeaponSpec{
		AssetDir: asset, Category: DevelopedCategorySniper, FireMode: mode,
		Projectile: DevelopedProjectileBullet, ProjectileSpeed: 32, Spread: 1.2, Lifetime: 120,
	})
}

func addShotgun(number int, name, asset string, rof int, firepower, recoil float64, bullets, pellets int) {
	addDevelopedWeapon(developedDef(number, name, asset, rof, firepower, recoil, bullets, pellets, .78), DevelopedWeaponSpec{
		AssetDir: asset, Category: DevelopedCategoryShotgun, FireMode: DevelopedFireSemi,
		Projectile: DevelopedProjectileBullet, ProjectileSpeed: 25, Spread: 8, Lifetime: 20,
	})
}

func addLauncher(number int, name, asset string, rof int, firepower, recoil float64, bullets int, projectile DevelopedProjectileKind, speed, gravity, radius, power float64) {
	addDevelopedWeapon(developedDef(number, name, asset, rof, firepower, recoil, bullets, 0, .55), DevelopedWeaponSpec{
		AssetDir: asset, Category: DevelopedCategoryLauncher, FireMode: DevelopedFireSemi,
		Projectile: projectile, ProjectileSpeed: speed, ProjectileGravity: gravity, Spread: .5,
		ExplosionRadius: radius, ExplosionPower: power, Lifetime: 180,
	})
}

func addGrenadeLauncher(number int, name, asset string, rof int, firepower, recoil float64, bullets int, speed float64) {
	addDevelopedWeapon(developedDef(number, name, asset, rof, firepower, recoil, bullets, 0, .65), DevelopedWeaponSpec{
		AssetDir: asset, Category: DevelopedCategoryLauncher, FireMode: DevelopedFireSemi,
		Projectile: DevelopedProjectileGrenade, ProjectileSpeed: speed, ProjectileGravity: .58, Spread: 1,
		ExplosionRadius: 165, ExplosionPower: 70, Lifetime: 95,
	})
}

func addSpecialProjectile(number int, name, asset string, rof int, firepower, recoil float64, bullets int, projectile DevelopedProjectileKind, speed, gravity float64) {
	mode := DevelopedFireSemi
	if number == 180 || number == 185 {
		mode = DevelopedFireAuto
	}
	spec := DevelopedWeaponSpec{
		AssetDir: asset, Category: DevelopedCategorySpecial, FireMode: mode,
		Projectile: projectile, ProjectileSpeed: speed, ProjectileGravity: gravity, Spread: 1.5, Lifetime: 140,
	}
	if projectile == DevelopedProjectileFlare {
		spec.IgniteFrames = 90
	}
	addDevelopedWeapon(developedDef(number, name, asset, rof, firepower, recoil, bullets, 0, .85), spec)
}

func addFlamethrower(number int, name, asset string, rof int, firepower, recoil float64, bullets int, speed float64, igniteFrames int) {
	addDevelopedWeapon(developedDef(number, name, asset, rof, firepower, recoil, bullets, 0, .72), DevelopedWeaponSpec{
		AssetDir: asset, Category: DevelopedCategorySpecial, FireMode: DevelopedFireAuto,
		Projectile: DevelopedProjectileFlame, ProjectileSpeed: speed, Spread: 5.5,
		Lifetime: 20, IgniteFrames: igniteFrames,
	})
}

func addPortalGun(number int, name, asset string) {
	addDevelopedWeapon(developedDef(number, name, asset, 9, 0, .15, 24, 0, .80), DevelopedWeaponSpec{
		AssetDir: asset, Category: DevelopedCategorySpecial, FireMode: DevelopedFireSemi,
		Projectile: DevelopedProjectilePortal, ProjectileSpeed: 28, Lifetime: 80,
	})
}

func developedWeaponSpec(number int) (DevelopedWeaponSpec, bool) {
	spec, ok := developedWeaponSpecs[number]
	return spec, ok
}

func allDevelopedWeaponNumbers() []int {
	out := make([]int, 0, len(developedWeaponCatalog))
	for n := range developedWeaponCatalog {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

// starterHandgunNumbers is the shared pre-match handgun roster. The six
// source handguns keep their original order, followed by every defined
// developed pistol. Availability must not depend on a runtime os.Stat check:
// portable builds already package the weapon assets and a transient path issue
// must never silently remove choices such as Peacekeeper from the picker.
func starterHandgunNumbers() []int {
	out := []int{1, 2, 3, 4, 5, 6}
	for _, number := range allDevelopedWeaponNumbers() {
		spec, ok := developedWeaponSpec(number)
		if ok && spec.Category == DevelopedCategoryPistol {
			out = append(out, number)
		}
	}
	return out
}

func isStarterHandgunNumber(number int) bool {
	for _, candidate := range starterHandgunNumbers() {
		if candidate == number {
			return true
		}
	}
	return false
}

func stepStarterHandgun(number, delta int) int {
	handguns := starterHandgunNumbers()
	if len(handguns) == 0 {
		return 1
	}
	index := -1
	for i, candidate := range handguns {
		if candidate == number {
			index = i
			break
		}
	}
	if index < 0 {
		return handguns[0]
	}
	index = (index + delta%len(handguns) + len(handguns)) % len(handguns)
	return handguns[index]
}

func customWeaponAssetAvailable(number int) bool {
	customWeaponAssetsOnce.Do(func() {
		for n, spec := range developedWeaponSpecs {
			_, err := findOriginalPathIn("custom", "weapons", spec.AssetDir, "1.png")
			customWeaponAssets[n] = err == nil
		}
	})
	return customWeaponAssets[number]
}

func availableCustomWeaponNumbers() []int {
	out := make([]int, 0, len(developedWeaponCatalog))
	for _, n := range allDevelopedWeaponNumbers() {
		if customWeaponAssetAvailable(n) {
			out = append(out, n)
		}
	}
	return out
}

func loadCustomWeaponRaster(number, frame int) *SourceRaster {
	spec, ok := developedWeaponSpecs[number]
	if !ok {
		return nil
	}
	filename := strconv.Itoa(frame+1) + ".png"
	img := decodeOriginalPNGIn("custom", "weapons", spec.AssetDir, filename)
	if img == nil && frame != 0 {
		img = decodeOriginalPNGIn("custom", "weapons", spec.AssetDir, "1.png")
	}
	if img == nil {
		return nil
	}
	img = normalizeCustomWeaponScale(img, number, spec.Category)
	originalBounds := img.Bounds()
	originX, originY := customWeaponOrigin(number, spec.Category)
	originPixelX := float64(originalBounds.Dx()) * originX
	originPixelY := float64(originalBounds.Dy()) * originY
	b := img.Bounds()
	return &SourceRaster{
		Image: ebiten.NewImageFromImage(img),
		// Generated custom sprites do not have XFL registration metadata. Anchor
		// them at the category's typical trigger/grip position so stocks extend
		// behind the hand and barrels extend in front, just like source weapons.
		Bounds: Rect{
			X: -originPixelX,
			Y: -originPixelY,
			W: float64(b.Dx()), H: float64(b.Dy()),
		},
	}
}

func normalizeCustomWeaponScale(img image.Image, number int, category DevelopedWeaponCategory) image.Image {
	if img == nil {
		return img
	}
	// Generated sidearms were authored at almost rifle scale. The harpoon sprite
	// also has a deliberately bulky silhouette, so normalize that one weapon to
	// player scale without touching any original weapon (1..86).
	scale := 1.0
	if override, ok := customWeaponScaleOverrides[number]; ok {
		scale = override
	} else if number >= 190 && number <= 231 {
		// The WWI/follow-up batch was authored on uniform 148 px canvases,
		// whereas the source game's shoulder weapons are normally about
		// 88..116 px wide. Normalize this batch by physical weapon class.
		switch category {
		case DevelopedCategoryPistol:
			scale = .42
		case DevelopedCategorySMG:
			scale = .65
		case DevelopedCategoryRifle, DevelopedCategorySniper, DevelopedCategoryShotgun:
			scale = .76
		case DevelopedCategoryLMG:
			scale = .76
		}
	} else if category == DevelopedCategoryPistol {
		scale = .72
	} else if number == 179 {
		scale = .66
	} else if number == 183 {
		scale = .72
	}
	// ImageGen source files remain high-resolution portable assets. Cap only
	// unusually large rasters; the existing compact 87..186 sprites stay exact.
	b := img.Bounds()
	const maxW = 148.0
	const maxH = 78.0
	fit := math.Min(maxW/float64(b.Dx()), maxH/float64(b.Dy()))
	if fit < scale {
		scale = fit
	}
	if scale == 1 {
		return img
	}
	w := int(float64(b.Dx())*scale + .5)
	h := int(float64(b.Dy())*scale + .5)
	if w < 1 || h < 1 {
		return img
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	return dst
}

// These late additions were generated on much larger canvases than the older
// developed weapons. Per-weapon normalization keeps their silhouettes readable
// without changing the size of any original weapon or the earlier additions.
var customWeaponScaleOverrides = map[int]float64{
	190: .42, // Lancaster Howdah: compact multi-barrel pistol
	192: .84, // Tankgewehr remains visibly longer than an ordinary rifle
	198: .72, // Villar Perosa is tall, but should not cover the whole torso
	206: .42, // Colt Model 1903 pocket pistol
	207: .38, // Frommer Stop
	213: .70, // G11 compact bullpup body
	216: .38, // Nambu Type 14
	219: .42, // AF2011-A1 double 1911
	230: .50, // Annihilator: shoulder-fired SMG, larger than a sidearm
	231: .42, // Peacekeeper long-barrel revolver
	93:  1.2, // M14 EBR canvas has wide padding; match other battle rifles
	165: 1.2, // KS-23 is a full-size pump gun, not a carbine
	180: .75, // nail gun is a handheld tool
	181: .70, // flare gun is pistol-sized
}

func customWeaponOrigin(number int, category DevelopedWeaponCategory) (x, y float64) {
	if grip, ok := customWeaponGripOrigins[number]; ok {
		return grip[0], grip[1]
	}
	switch category {
	case DevelopedCategoryPistol:
		return .46, .57
	case DevelopedCategorySMG:
		return .44, .59
	case DevelopedCategoryLauncher:
		return .42, .59
	case DevelopedCategoryLMG, DevelopedCategorySniper:
		return .38, .58
	case DevelopedCategoryRifle, DevelopedCategoryShotgun:
		return .40, .60
	default:
		return .42, .60
	}
}
