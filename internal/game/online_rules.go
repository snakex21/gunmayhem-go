package game

const (
	onlineMapFixed = iota
	onlineMapRandomGM1
	onlineMapRandomGM2
	onlineMapRandomAll
)

const (
	onlineWeaponsAll = iota
	onlineWeaponsOnlyOne
	onlineWeaponsBlocklist
)

func onlineAllWeaponNumbers() []int {
	out := make([]int, 0, 83+len(developedWeaponCatalog))
	for n := 1; n <= 6; n++ {
		out = append(out, n)
	}
	for n := 10; n <= 86; n++ {
		out = append(out, n)
	}
	out = append(out, availableCustomWeaponNumbers()...)
	return out
}

func onlineMapRotationLabel(v int) string {
	switch v {
	case onlineMapRandomGM1:
		return "RANDOM GM1"
	case onlineMapRandomGM2:
		return "RANDOM GM2"
	case onlineMapRandomAll:
		return "RANDOM ALL"
	default:
		return "FIXED MAP"
	}
}

func onlineWeaponModeLabel(v int) string {
	switch v {
	case onlineWeaponsOnlyOne:
		return "ONLY ONE WEAPON"
	case onlineWeaponsBlocklist:
		return "CUSTOM BLOCKLIST"
	default:
		return "ALL WEAPONS"
	}
}

func onlinePerkName(n int) string {
	names := map[int]string{
		1:                       "NO PERKS",
		2:                       "SPEED+",
		3:                       "JUMP+",
		4:                       "MINI GUN",
		5:                       "NO RECOIL",
		6:                       "BLAST SHIELD",
		7:                       "EXTRA AMMO",
		8:                       "EXTRA NADES",
		9:                       "GUN SPAWN",
		perkKnockbackResistance: "KNOCKBACK -30%",
	}
	if name := names[n]; name != "" {
		return name
	}
	return "PERK"
}

func (g *Game) blockedOnlinePerkCount() int {
	if g == nil {
		return 0
	}
	count := 0
	for n := 1; n < len(g.onlineBlockedPerks); n++ {
		if g.onlineBlockedPerks[n] {
			count++
		}
	}
	return count
}

func (g *Game) blockedOnlineWeaponCount() int {
	if g == nil {
		return 0
	}
	count := 0
	for _, n := range onlineAllWeaponNumbers() {
		if n > 0 && n < len(g.onlineBlockedWeapons) && g.onlineBlockedWeapons[n] {
			count++
		}
	}
	return count
}
