package game

import "testing"

func TestCustomModifierConflicts(t *testing.T) {
	pairs := [][2]int{
		{customModifierShotgunsOnly, customModifierPistolsOnly},
		{customModifierPistolsOnly, customModifierExplosivesOnly},
		{customModifierLowGravity, customModifierHighGravity},
		{customModifierNoReload, customModifierInfiniteAmmo},
		{customModifierNoCrates, customModifierCratesEverywhere},
		{customModifierRandomWeaponEachLife, customModifierSameWeapon},
	}
	for _, pair := range pairs {
		if !customModifiersConflict(pair[0], pair[1]) || !customModifiersConflict(pair[1], pair[0]) {
			t.Fatalf("expected conflict %d <-> %d", pair[0], pair[1])
		}
	}
}

func TestCustomShotgunAndPistolPools(t *testing.T) {
	g := &Game{customMatchActive: true, matchModifiers: map[int]bool{customModifierShotgunsOnly: true}}
	shotguns := g.modifierWeaponPool()
	if len(shotguns) == 0 {
		t.Fatal("shotgun pool empty")
	}
	for _, n := range shotguns {
		def, ok := WeaponByNumber(n)
		if !ok || def.Shotgun <= 0 {
			t.Fatalf("non-shotgun %d in shotgun pool", n)
		}
	}
	g.matchModifiers = map[int]bool{customModifierPistolsOnly: true}
	pistols := g.modifierWeaponPool()
	if len(pistols) != 27 {
		t.Fatalf("pistol pool len=%d want27 (6 source + 21 custom)", len(pistols))
	}
	for _, n := range pistols {
		if n >= 1 && n <= 6 {
			continue
		}
		spec, ok := developedWeaponSpec(n)
		if !ok || spec.Category != DevelopedCategoryPistol {
			t.Fatalf("non-pistol %d in pistol pool", n)
		}
	}
}

func TestCustomGravityAndCrateRules(t *testing.T) {
	g := &Game{arena: OriginalMap1(), CrateON: true, customMatchActive: true, matchModifiers: map[int]bool{customModifierLowGravity: true}}
	if got := g.customRuntimeArena().Gravity; got >= g.arena.Gravity {
		t.Fatalf("low gravity=%v base=%v", got, g.arena.Gravity)
	}
	g.matchModifiers = map[int]bool{customModifierHighGravity: true}
	if got := g.customRuntimeArena().Gravity; got <= g.arena.Gravity {
		t.Fatalf("high gravity=%v base=%v", got, g.arena.Gravity)
	}
	g.matchModifiers = map[int]bool{customModifierNoCrates: true}
	if g.customCratesEnabled() {
		t.Fatal("no crates rule still enables crates")
	}
	g.matchModifiers = map[int]bool{customModifierCratesEverywhere: true}
	if !g.customCratesEnabled() || g.customCrateInterval() >= 400 {
		t.Fatal("crates everywhere rule not applied")
	}
}

func TestCustomSameWeaponSetup(t *testing.T) {
	arena := OriginalMap1()
	p1 := NewPlayer(1, arena)
	p2 := NewPlayer(2, arena)
	g := &Game{
		players:           []*Player{p1, p2},
		customMatchActive: true,
		matchModifiers:    map[int]bool{customModifierSameWeapon: true, customModifierShotgunsOnly: true},
		crateWeapons:      developedCrateWeapons(),
	}
	g.applyCustomModifierSetup()
	if p1.DefaultWeapon != p2.DefaultWeapon {
		t.Fatalf("same weapon rule gave %d and %d", p1.DefaultWeapon, p2.DefaultWeapon)
	}
	def, ok := WeaponByNumber(p1.DefaultWeapon)
	if !ok || def.Shotgun <= 0 {
		t.Fatalf("same weapon with shotgun rule picked %d", p1.DefaultWeapon)
	}
	if len(g.crateWeapons) != 1 || g.crateWeapons[0] != p1.DefaultWeapon {
		t.Fatalf("crate pool=%v want only %d", g.crateWeapons, p1.DefaultWeapon)
	}
}
