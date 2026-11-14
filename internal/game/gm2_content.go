package game

// gm2WeaponAssetsAvailable reports whether the runtime can actually render the
// appended GM2 arsenal. assets/gm2 is preferred by the resolver; during local
// import work the separately unpacked `gun mayhem 2` tree is a read-only
// fallback. This prevents fresh clones without GM2 assets from spawning
// invisible weapons just because the 67..86 definitions exist in Go.
func gm2WeaponAssetsAvailable() bool {
	checks := [][]string{
		{"sprites", "DefineSprite_573_gun_A1", "1.png"},
		{"sprites", "DefineSprite_613_gun_D5", "1.png"},
		{"fla", "LIBRARY", "Symbol 573.xml"},
		{"fla", "LIBRARY", "Symbol 613.xml"},
	}
	for _, parts := range checks {
		if _, err := findOriginalPathIn("gm2", parts...); err != nil {
			return false
		}
	}
	return true
}

func developedCrateWeapons() []int {
	var out []int
	if gm2WeaponAssetsAvailable() {
		out = append(out, GM2CrateWeapons()...)
	} else {
		out = append(out, DefaultCrateWeapons()...)
	}
	// Custom 87+ weapons are intentionally dormant until their first PNG is
	// present under assets/custom/weapons/<asset-dir>/1.png.
	for _, n := range availableCustomWeaponNumbers() {
		if developedCrateEligible(n) {
			out = append(out, n)
		}
	}
	return out
}

// developedCrateEligible keeps added handguns and the Portal Gun out of
// random crates. Source crates never hold handguns (random(77)+10), and the
// added pistols reload in place, so a crate pistol would be a permanent
// replacement for the starter. They stay available as starter handguns, in
// the pistols-only modifier, the gun library and campaign loadouts.
func developedCrateEligible(number int) bool {
	spec, ok := developedWeaponSpec(number)
	if !ok {
		return false
	}
	if spec.Category == DevelopedCategoryPistol || spec.Projectile == DevelopedProjectilePortal {
		return false
	}
	return true
}
