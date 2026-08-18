package game

import "testing"

func TestSourceRoundStartsCrateTimerAt200(t *testing.T) {
	g := &Game{}
	g.resetArenaState()
	if g.crateTimer != 200 {
		t.Fatalf("source cratetime start=%d want200", g.crateTimer)
	}
	if g.powerupTimer != 0 {
		t.Fatalf("source poweruptime start=%d want0", g.powerupTimer)
	}
}
