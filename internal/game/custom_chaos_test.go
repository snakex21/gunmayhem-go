package game

import "testing"

func TestChaosDefaultsSelectWholePool(t *testing.T) {
	g := &Game{customChaosInterval: 30}
	g.initCustomChaosDefaults()
	if got, want := g.chaosSelectedCount(), len(chaosEffectSpecs); got != want {
		t.Fatalf("selected chaos effects=%d want %d", got, want)
	}
}

func TestChaosPickerUsesOnlyManualPool(t *testing.T) {
	g := &Game{}
	g.customChaosSelected[chaosEffectFireRain] = true
	for i := 0; i < 20; i++ {
		if got := g.pickNextChaosEffect(); got != chaosEffectFireRain {
			t.Fatalf("picked effect=%d want fire rain=%d", got, chaosEffectFireRain)
		}
	}
}

func TestChaosTemporaryModifierParticipatesInRuntimeRules(t *testing.T) {
	p := NewPlayer(1, OriginalMap1())
	g := &Game{
		players:            []*Player{p},
		customMatchActive:  true,
		customChaosEnabled: true,
		chaosCurrentEffect: chaosEffectJetpacks,
		matchModifiers:     map[int]bool{},
	}
	if !g.customModifierActive(customModifierJetpacks) {
		t.Fatal("chaos jetpack effect did not expose runtime modifier")
	}
	g.syncRuntimePlayerRuleModifiers()
	if !p.RuleInfiniteJetpack {
		t.Fatal("chaos jetpack effect did not enable infinite jetpack")
	}
	g.chaosCurrentEffect = chaosEffectNone
	g.syncRuntimePlayerRuleModifiers()
	if p.RuleInfiniteJetpack {
		t.Fatal("jetpack rule remained active after chaos effect ended")
	}
}

func TestChaosFireRainSpawnsNeutralMolotov(t *testing.T) {
	g := &Game{arena: OriginalMap1()}
	g.spawnChaosFallingThrowable(ThrowableMolotov)
	if len(g.grenades) != 1 {
		t.Fatalf("grenades=%d want1", len(g.grenades))
	}
	grenade := g.grenades[0]
	if grenade.Type != ThrowableMolotov || grenade.OwnerID != 0 || grenade.Y >= 0 {
		t.Fatalf("fire rain projectile=%+v", grenade)
	}
}

func TestChaosShockPulsePushesActivePlayers(t *testing.T) {
	arena := OriginalMap1()
	left := NewPlayer(1, arena)
	right := NewPlayer(2, arena)
	left.X = arena.CrateMinX
	right.X = arena.CrateMaxX
	left.VX, right.VX = 0, 0
	g := &Game{arena: arena, players: []*Player{left, right}}
	g.fireChaosShockPulse()
	if left.VX >= 0 || right.VX <= 0 || left.VY >= 0 || right.VY >= 0 {
		t.Fatalf("pulse velocities left=(%v,%v) right=(%v,%v)", left.VX, left.VY, right.VX, right.VY)
	}
}
