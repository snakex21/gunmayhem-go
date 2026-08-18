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
	if gm2WeaponAssetsAvailable() {
		return GM2CrateWeapons()
	}
	return DefaultCrateWeapons()
}
