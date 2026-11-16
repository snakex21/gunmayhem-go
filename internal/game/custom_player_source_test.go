package game

import "testing"

func TestCustomPlayerDefaultsAndSlotHits(t *testing.T) {
	g := &Game{}
	g.initCustomPlayerSetup()
	if g.activeCustomPlayerCount() != 2 {
		t.Fatalf("default active=%d want2", g.activeCustomPlayerCount())
	}
	// An empty slot accepts the HUMAN action code.
	if !g.activateCustomPlayerSlot(playerSlotHit(2, playerSlotActionHuman)) || g.customPlayers[2].Type != 1 {
		t.Fatalf("empty slot HUMAN action: type=%d", g.customPlayers[2].Type)
	}
}
