package game

// realWeaponNames is used by the developed main branch for player-facing
// labels. WeaponDef.Name intentionally remains the original Flash name (for
// example "COOL PISTOL") so source-parity tests and reference data stay intact.
//
// Most labels are directly corroborated by the original linkage identifiers
// (gun_m1911, gun_deagle, gun_ak47, gun_p90, ...). A few authoring names are
// shorthand; for those we use the real-world counterpart represented by the
// artwork rather than rewriting the source definition itself.
var realWeaponNames = map[int]string{
	1:  "Colt M1911A1",
	2:  "Desert Eagle",
	3:  "Glock 18C",
	4:  "Taurus Raging Bull M444",
	5:  "FN Forty-Nine",
	6:  "Colt Python",
	8:  "Colt M1911A1 (Infinite Ammo)",
	9:  "De-Materializer",
	10: "AK-47",
	11: "H&K PSG-1",
	12: "H&K MP5K-PDW",
	13: "Winchester Model 1200 Sawed-Off",
	14: "FN SCAR SSR",
	15: "USAS-12",
	16: "Mauser 86 SR",
	17: "Ruger Mini-14",
	18: "Winchester Model 1887 Sawed-Off",
	19: "FN P90 Silenced",
	20: "Bushmaster M17S",
	21: "PP-19 Bizon",
	22: "Thompson SMG",
	23: "H&K UMP45",
	24: "Steyr AUG A3 Para",
	25: "H&K MP5A3",
	26: "LUSA A2",
	27: "Beretta M12",
	28: "MAS-38",
	29: "AKS-74U",
	30: "KRISS Vector",
	31: "B&T MP9",
	32: "MAS-48",
	33: "Accuracy International AWM-F",
	34: "PGM Hecate II",
	35: "Barrett M95",
	36: "DSR-1",
	37: "Steyr SSG 69",
	38: "Blaser R93 LRS2",
	39: "Galatz",
	40: "VSS Vintorez",
	41: "Zastava M76",
	42: "KAC SR-25",
	43: "Winchester Model 1873",
	44: "Ithaca Flues Double-Barrel Sawed-Off",
	45: "Kel-Tec KSG-12",
	46: "AA-12",
	47: "Pancor Jackhammer MK3A2",
	48: "PM-5-350 (PM-12)",
	49: "Remington 870 Marine Magnum",
	50: "Franchi SPAS-12",
	51: "Saiga-12",
	52: "Saiga-20",
	53: "Fallout Combat Shotgun (SPAS-97)",
	54: "Mossberg 500",
	55: "M134 Minigun",
	56: "H&K G36C",
	57: "RPK-74",
	58: "SIG SG 552",
	59: "IMI Galil ARM",
	60: "Colt M4A1",
	61: "H&K HK33",
	62: "FN F2000",
	63: "FN SCAR-H",
	64: "Stoner 63A",
	65: "FN M249 SAW",
	66: "Steyr AUG HBAR",

	// GM2 additions. The game itself uses descriptive labels (GREY SMG,
	// STREET SWEEPER, ...); these are the real-world designs represented by
	// the artwork. Where the reference identifies only a family/model class,
	// keep that conservative name instead of inventing a more specific variant.
	67: "CZ Scorpion EVO 3 A1",
	68: "MAC-10",
	69: "ST Kinetics CPW",
	70: "Daewoo K7",
	71: "B&T APC",
	72: "CETME Rifle",
	73: "StG 44",
	74: "OTs-14 Groza",
	75: "AR-15",
	76: "Stoner 63",
	77: "Serbu BFG-50A",
	78: "Gepard Anti-Materiel Rifle",
	79: "ArmaLite AR-50",
	80: "VSSK Vykhlop",
	81: "AMR-2",
	82: "Origin-12",
	83: "SRM Arms Model 1216",
	84: "Benelli M3 Super 90",
	85: "Armsel Striker",
	86: "Browning Auto-5",
}

func (w WeaponDef) DisplayName() string {
	if name := realWeaponNames[w.Number]; name != "" {
		return name
	}
	return w.Name
}

func weaponDisplayName(number int) string {
	if def, ok := WeaponByNumber(number); ok {
		return def.DisplayName()
	}
	return "Nothing Selected"
}
