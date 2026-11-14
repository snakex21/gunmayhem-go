package game

// Near-duplicate review of the added arsenal (docs/weapons_roster.md).
//
// Many added guns were near-identical: same class, same fire delay, push
// within 1, the same magazine within 2 and the same recoil. Fire delays of
// real firearms come from their real cyclic rate (developed_fire_rates.go)
// and stay; each duplicate instead gets its own feel on the remaining axes:
// handling weight (run speed while held), accuracy (Spread), muzzle velocity
// (ProjectileSpeed), magazine, recoil and a small push change. Every value
// stays inside its class caps (developed_balance_test.go,
// developed_recoil_test.go). Original weapons 1-86 are never changed.
type developedIdentityTweak struct {
	Firepower float64 // 0: unchanged
	Bullets   int
	Recoil    float64
	Weight    float64
	Spread    float64
	Speed     float64
	FanStep   float64 // shotguns: PelletFanStep
	Why       string
}

// developedIdentityTweaks: number -> new values (before values in
// docs/weapons_roster.md, "Near-duplicate review").
var developedIdentityTweaks = map[int]developedIdentityTweak{
	// Pistols.
	109: {Spread: 1.0, Speed: 29, Recoil: 0.7, Why: "Luger: precise, fast round (vs Colt M1911A1)"},
	231: {Firepower: 34, Recoil: 4.5, Spread: 1.2, Speed: 30, Why: "Peacekeeper: long-barrel accuracy, less push than the Python"},
	101: {Spread: 1.8, Speed: 28, Recoil: 0.85, Why: "P226: tighter and faster than the 92FS"},
	102: {Spread: 2.0, Recoil: 0.6, Why: "CZ 75: soft-shooting"},
	206: {Firepower: 17, Weight: 1, Why: "Colt 1903: pocket pistol, no slowdown"},
	216: {Spread: 3.5, Speed: 22, Why: "Nambu: slow, wandering round"},
	// SMGs.
	114: {Firepower: 18, Spread: 4.2, Weight: 0.95, Why: "Uzi: light, sprays"},
	121: {Spread: 2.2, Why: "Sterling: accurate"},
	122: {Firepower: 18, Spread: 4.8, Why: "Sten: crude"},
	123: {Firepower: 20, Why: "PPS-43: hardest-hitting of the WWII SMGs"},
	195: {Bullets: 50, Weight: 0.86, Why: "MP 28: 50-round magazine, heavier"},
	196: {Firepower: 20, Spread: 3.0, Recoil: 0.6, Why: "Erma EMP: steady"},
	115: {Bullets: 25, Spread: 3.0, Why: "Micro Uzi: shorter magazine, tighter"},
	116: {Spread: 4.8, Why: "MAC-11: pure spray"},
	212: {Spread: 2.5, Weight: 0.88, Why: "UMP9: accurate, heavier than the Vityaz"},
	119: {Spread: 2.4, Recoil: 0.4, Why: "SIG MPX: accurate (vs ST Kinetics CPW)"},
	214: {Recoil: 0.6, Spread: 4.0, Why: "Grease Gun: slow, soft, loose"},
	// Rifles.
	130: {Firepower: 24, Recoil: 0.9, Spread: 3.2, Why: "AK-12: hits harder, kicks and wanders"},
	136: {Firepower: 22, Spread: 1.4, Weight: 0.74, Why: "L85A2: very accurate, heavy"},
	137: {Bullets: 20, Spread: 1.6, Speed: 30, Why: "SG 550: precise, 20-round magazine"},
	138: {Firepower: 21, Recoil: 0.45, Weight: 0.86, Why: "XM8: light carbine"},
	131: {Recoil: 0.6, Spread: 3.2, Why: "AK-74: soft recoil, loose (vs SIG SG 552)"},
	127: {Firepower: 35, Weight: 0.74, Spread: 2.0, Why: "FAL: heavier and harder than the SCAR-H"},
	129: {Recoil: 0.4, Spread: 3.0, Why: "AEK-971: balanced recoil"},
	134: {Firepower: 17, Spread: 2.0, Weight: 0.86, Why: "Tavor: compact bullpup"},
	218: {Firepower: 30, Spread: 1.6, Why: "Johnson: accurate"},
	225: {Bullets: 8, Firepower: 32, Recoil: 1.6, Spread: 3.0, Why: "Mondragon: 8 heavy, kicking rounds"},
	// Machine guns.
	94:  {Spread: 2.5, Why: "PKP: forced-air barrel, accurate"},
	95:  {Firepower: 19, Bullets: 50, Weight: 0.78, Why: "RPD: light, drum of 50"},
	140: {Firepower: 19, Recoil: 0.7, Spread: 4.5, Why: "PKM: loose spray"},
	144: {Recoil: 0.6, Spread: 3.0, Weight: 0.64, Why: "M240B: heavy, steady"},
	142: {Bullets: 60, Spread: 3.0, Why: "MG3: belt of 60, tighter than the MG42"},
	141: {Spread: 4.5, Why: "MG42: wilder"},
	199: {Weight: 0.6, Spread: 2.5, Recoil: 0.8, Why: "Maxim: water-cooled, steady, very heavy"},
	200: {Firepower: 23, Bullets: 50, Spread: 4.5, Weight: 0.76, Why: "MG 15: light aircraft gun"},
	148: {Spread: 3.0, Weight: 0.66, Why: "Lewis: cooling shroud, heavy"},
	// Bolt-action rifles.
	151: {Firepower: 57, Recoil: 8, Spread: 1.6, Weight: 0.7, Why: "Mosin: kicks, heavier"},
	156: {Firepower: 64, Recoil: 8.5, Speed: 34, Why: "M40A5: hotter round than the M24"},
	205: {Spread: 1.0, Speed: 34, Recoil: 7.5, Why: "Springfield: flat and precise"},
	215: {Firepower: 56, Recoil: 6, Weight: 0.76, Why: "Arisaka: mild 6.5/7.7 mm, lighter"},
	// Shotguns.
	164: {FanStep: 1.4, Why: "full-length 1887: tighter pattern than the sawed-off (18)"},
	204: {FanStep: 3.0, Why: "Sjogren: wide pattern (vs Saiga-20)"},
}

// applyDevelopedIdentityTweaks runs at the end of the developed catalog init.
func applyDevelopedIdentityTweaks() {
	for number, tw := range developedIdentityTweaks {
		def, okDef := developedWeaponCatalog[number]
		spec, okSpec := developedWeaponSpecs[number]
		if !okDef || !okSpec {
			continue
		}
		if tw.Firepower > 0 {
			def.Firepower = tw.Firepower
		}
		if tw.Bullets > 0 {
			def.Bullets = tw.Bullets
		}
		if tw.Recoil > 0 {
			def.Recoil = tw.Recoil
		}
		if tw.Weight > 0 {
			def.Weight = tw.Weight
		}
		if tw.Spread > 0 {
			spec.Spread = tw.Spread
		}
		if tw.Speed > 0 {
			spec.ProjectileSpeed = tw.Speed
		}
		if tw.FanStep > 0 {
			spec.PelletFanStep = tw.FanStep
		}
		developedWeaponCatalog[number] = def
		developedWeaponSpecs[number] = spec
	}
}
