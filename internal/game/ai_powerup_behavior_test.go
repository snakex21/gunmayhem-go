package game

import "testing"

func TestAIPowerupTargetWalksIntoSmallPickupFrame(t *testing.T) {
	ai := &Player{
		ID: 1, Active: true, AI: true, Team: 1,
		X: 400, Y: 350, Facing: 1, JumpNum: 2, PlayerScale: .8,
		AITargetValid: true, AITargetKind: aiTargetPowerup, AITargetSerial: 77,
		AITargetX: 430, AITargetY: 310,
		Weapon: NewWeapon(1),
	}
	g := &Game{
		arena: Map{Platforms: []Rect{{X: 0, Y: 350, W: 900, H: 120}}, LowestY: 600},
		players: []*Player{ai},
		powerups: []Powerup{{Serial: 77, X: 430, Y: 310, Type: 6, Frame: 19}},
	}
	decideAIController(ai, g)
	if !ai.AIRight {
		t.Fatal("AI stopped outside the 10x10 MINI pickup instead of walking into it")
	}

	// Once the bot reaches the pickup centre it must receive the exact same MINI
	// effect as a human player.
	ai.X = 430
	updatePowerups(g.powerups, g.players)
	if ai.MiniTime != 400 {
		t.Fatalf("AI MINI pickup did not apply: MiniTime=%d", ai.MiniTime)
	}
	ai.updatePowerupTimers()
	if ai.PlayerScale >= .8 {
		t.Fatalf("AI MINI pickup did not start shrinking: scale=%v", ai.PlayerScale)
	}
}

func TestAIInvisiblePowerupActuallyFadesBot(t *testing.T) {
	ai := &Player{ID: 2, Active: true, AI: true, Alpha: 1, PlayerScale: .8}
	applyPowerup(ai, 1)
	for i := 0; i < 12; i++ {
		ai.updatePowerupTimers()
	}
	if ai.InvisibleTime <= 0 || ai.Alpha > .01 {
		t.Fatalf("AI invisibility did not hide bot: time=%d alpha=%v", ai.InvisibleTime, ai.Alpha)
	}
	g := &Game{players: []*Player{ai}}
	if g.shouldDrawInvisibleGhost(ai) {
		t.Fatal("AI invisibility incorrectly draws the local-human ghost guide")
	}
}

func TestAIFullRetargetStillSearchesPlayerWhoJustTurnedInvisible(t *testing.T) {
	ai := &Player{
		ID: 1, Active: true, AI: true, Team: 1,
		X: 100, Y: 300, Facing: 1, JumpNum: 2,
		AITargetTimer: 39,
		AITargetValid: true, AITargetKind: aiTargetPlayer, AITargetPlayerID: 2,
		AITargetX: 260, AITargetY: 300,
		Weapon: NewWeapon(1),
	}
	target := &Player{ID: 2, Active: true, Team: 2, X: 700, Y: 300, InvisibleTime: 150}
	g := &Game{arena: Map{Platforms: []Rect{{X: 0, Y: 350, W: 900, H: 120}}, LowestY: 600}, players: []*Player{ai, target}}
	prepareAITarget(ai, g)
	if !ai.AITargetValid || ai.AITargetKind != aiTargetSearch || ai.AISearchPlayerID != target.ID {
		t.Fatalf("AI forgot invisible player on full retarget boundary: %+v", ai)
	}
	if ai.AISearchX != 260 || ai.AISearchY != 300 {
		t.Fatalf("AI search leaked current invisible coordinates: search=(%v,%v)", ai.AISearchX, ai.AISearchY)
	}

	shots := 0
	for i := 0; i < 120; i++ {
		decideAIController(ai, g)
		if ai.AIShoot {
			shots++
		}
	}
	if shots == 0 {
		t.Fatal("AI did not blind-fire while searching for invisible opponent")
	}
}
