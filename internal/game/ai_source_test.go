package game

import "testing"

func TestAISourceTargetTimerAndPowerupPriority(t *testing.T) {
	arena := OriginalMap1()
	ai := NewPlayer(2, arena)
	ai.AI = true
	ai.Team = 2
	ai.X, ai.Y = 400, 300
	enemy := NewPlayer(1, arena)
	enemy.Team = 1
	enemy.X, enemy.Y = 410, 300
	g := &Game{
		players: []*Player{enemy, ai},
		arena:   arena,
		powerups: []Powerup{
			{Serial: 1, X: 800, Y: 300, Type: 0},
			{Serial: 2, X: 850, Y: 300, Type: 3},
		},
	}

	for i := 0; i < 39; i++ {
		prepareAITarget(ai, g)
	}
	if ai.AITargetValid {
		t.Fatal("playerAI selected target before source targettime reached 40")
	}
	prepareAITarget(ai, g)
	// Non-double source always breaks on life powerup 0 even though enemy is
	// much closer and jetpack powerup 3 appears later in cratearray.
	if !ai.AITargetValid || ai.AITargetKind != aiTargetPowerup || ai.AITargetSerial != 1 {
		t.Fatalf("source powerup priority mismatch: kind=%d serial=%d", ai.AITargetKind, ai.AITargetSerial)
	}

	ai.IsDouble = true
	ai.AITargetTimer = 39
	prepareAITarget(ai, g)
	// Exact source precedence means Double skips life(0) but still breaks on 3.
	if ai.AITargetKind != aiTargetPowerup || ai.AITargetSerial != 2 {
		t.Fatalf("double source priority mismatch: kind=%d serial=%d want jetpack serial2", ai.AITargetKind, ai.AITargetSerial)
	}
}

func TestAIStaysOnMapWithoutSelfFalling(t *testing.T) {
	arena := OriginalMap1()
	enemy := NewPlayer(1, arena)
	enemy.Team = 1
	enemy.X, enemy.Y = 450, arena.Platforms[0].Y
	enemy.VX, enemy.VY = 0, 0
	enemy.JumpNum = 2
	enemy.Grounded = true

	ai := NewPlayer(2, arena)
	ai.AI = true
	ai.Team = 2
	ai.X, ai.Y = 650, arena.Platforms[0].Y
	ai.VX, ai.VY = 0, 0
	ai.JumpNum = 2
	ai.Grounded = true
	ai.AITargetTimer = 40

	g := &Game{players: []*Player{enemy, ai}, arena: arena}
	startDeaths := ai.DeathSerial
	for tick := 0; tick < 5000; tick++ {
		prepareAITarget(ai, g)
		ai.Update(arena, nil, func() { decideAIController(ai, g) })
		if ai.DeathSerial != startDeaths {
			t.Fatalf("AI self-fell by tick %d at x=%.2f y=%.2f", tick, ai.LastDeathX, ai.LastDeathY)
		}
	}
}

func TestAIDoesNotFollowTargetOffMap(t *testing.T) {
	arena := OriginalMap1()
	enemy := NewPlayer(1, arena)
	enemy.Team = 1
	enemy.X, enemy.Y = 1050, 650
	enemy.VX, enemy.VY = 0, 0
	enemy.JumpNum = 1

	ai := NewPlayer(2, arena)
	ai.AI = true
	ai.Team = 2
	ai.X, ai.Y = 650, arena.Platforms[0].Y
	ai.VX, ai.VY = 0, 0
	ai.JumpNum = 2
	ai.Grounded = true
	ai.AITargetTimer = 40

	g := &Game{players: []*Player{enemy, ai}, arena: arena}
	startDeaths := ai.DeathSerial
	for tick := 0; tick < 3000; tick++ {
		prepareAITarget(ai, g)
		ai.Update(arena, nil, func() { decideAIController(ai, g) })
		if ai.DeathSerial != startDeaths {
			t.Fatalf("AI followed an off-map target to death by tick %d at x=%.2f y=%.2f", tick, ai.LastDeathX, ai.LastDeathY)
		}
	}
}

func TestAINavigatesDownToLowerPlatformsWithoutFallingOut(t *testing.T) {
	arena := OriginalMap1()
	cases := []struct {
		name string
		x, y float64
	}{
		{"left-middle", 250, arena.Platforms[4].Y},
		{"right-middle", 650, arena.Platforms[1].Y},
		{"left-low", 250, arena.Platforms[5].Y},
		{"right-low", 650, arena.Platforms[3].Y},
		{"bottom", 450, arena.Platforms[2].Y},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			enemy := NewPlayer(1, arena)
			enemy.Team = 1
			enemy.X, enemy.Y = tc.x, tc.y
			enemy.VX, enemy.VY = 0, 0
			enemy.JumpNum = 2
			enemy.Grounded = true

			ai := NewPlayer(2, arena)
			ai.AI = true
			ai.Team = 2
			ai.X, ai.Y = 650, arena.Platforms[0].Y
			ai.VX, ai.VY = 0, 0
			ai.JumpNum = 2
			ai.Grounded = true
			ai.AITargetTimer = 40

			g := &Game{players: []*Player{enemy, ai}, arena: arena}
			startDeaths := ai.DeathSerial
			for tick := 0; tick < 5000; tick++ {
				prepareAITarget(ai, g)
				ai.Update(arena, nil, func() { decideAIController(ai, g) })
				if ai.DeathSerial != startDeaths {
					t.Fatalf("AI fell out by tick %d chasing target at %.1f,%.1f; death at %.2f,%.2f", tick, tc.x, tc.y, ai.LastDeathX, ai.LastDeathY)
				}
			}
		})
	}
}

func TestSourceCrateArrayUsesAttachmentSerial(t *testing.T) {
	g := &Game{
		crates:   []Crate{{Serial: 2, X: 2}, {Serial: 4, X: 4}},
		powerups: []Powerup{{Serial: 1, X: 1}, {Serial: 3, X: 3}},
	}
	refs := sourceCrateArray(g)
	if len(refs) != 4 {
		t.Fatalf("cratearray len=%d want4", len(refs))
	}
	for i, want := range []int{1, 2, 3, 4} {
		if refs[i].Serial != want {
			t.Fatalf("cratearray[%d].serial=%d want%d", i, refs[i].Serial, want)
		}
	}
}
