package game

import "testing"

func TestChaosIceFloorChangesRuntimeArenaOnlyDuringEffect(t *testing.T) {
	g := &Game{arena: OriginalMap1(), customMatchActive: true, customChaosEnabled: true, chaosCurrentEffect: chaosEffectIceFloor}
	if !g.customRuntimeArena().LowFriction {
		t.Fatal("ice floor did not enable low friction")
	}
	g.chaosCurrentEffect = chaosEffectNone
	if g.customRuntimeArena().LowFriction != g.arena.LowFriction {
		t.Fatal("ice floor persisted after chaos effect ended")
	}
}

func TestChaosZeroGravityOverridesArenaGravity(t *testing.T) {
	g := &Game{arena: OriginalMap1(), customMatchActive: true, customChaosEnabled: true, chaosCurrentEffect: chaosEffectZeroGravity}
	if got := g.customRuntimeArena().Gravity; got != 0.08 {
		t.Fatalf("zero gravity=%v want 0.08", got)
	}
}

func TestChaosCategoryPoolContainsRequestedCategory(t *testing.T) {
	g := &Game{}
	pool := g.chaosWeaponCategoryPool(DevelopedCategorySniper)
	if len(pool) == 0 {
		t.Fatal("sniper chaos pool is empty")
	}
	for _, number := range pool {
		category := sourceWeaponCategory(number)
		if spec, ok := developedWeaponSpec(number); ok {
			category = spec.Category
		}
		if category != DevelopedCategorySniper {
			t.Fatalf("weapon %d category=%d want sniper", number, category)
		}
	}
}

func TestChaosFireFloorCreatesNeutralPatches(t *testing.T) {
	g := &Game{arena: OriginalMap1()}
	g.spawnChaosFireFloor(8)
	if len(g.firePatches) == 0 {
		t.Fatal("fire floor created no patches")
	}
	for _, patch := range g.firePatches {
		if patch.OwnerID != 0 || patch.Life <= 0 {
			t.Fatalf("invalid fire floor patch %+v", patch)
		}
	}
}

func TestChaosGrenadeRefillAndRandomThrowables(t *testing.T) {
	arena := OriginalMap1()
	p := NewPlayer(1, arena)
	p.Grenades = 0
	g := &Game{arena: arena, players: []*Player{p}, customChaosInterval: 30}
	g.activateChaosEffect(chaosEffectGrenadeRefill)
	if p.Grenades != 30 {
		t.Fatalf("grenade refill=%d want30", p.Grenades)
	}
	p.Grenades = 0
	g.activateChaosEffect(chaosEffectRandomThrowables)
	if p.Grenades < 12 || p.ThrowableType < 0 || p.ThrowableType >= throwableTypeCount {
		t.Fatalf("random throwables grenades=%d type=%d", p.Grenades, p.ThrowableType)
	}
}
