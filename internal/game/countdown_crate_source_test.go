package game

import "testing"

func TestEquipWeaponUsesGunFrameAdjustROFStartup(t *testing.T) {
	p := NewPlayer(1, OriginalMap1())
	p.EquipWeapon(10)
	want := p.Weapon.Def.ROF - 2
	if p.Weapon.WaitTime != want {
		t.Fatalf("equipped gun waittime=%d want%d from source gun frame1 adjustrof()", p.Weapon.WaitTime, want)
	}
}

func TestWinCountdownUsesThreeDistinctXFLDigitTimelines(t *testing.T) {
	checks := []struct {
		digit  int
		symbol string
		frame  int
	}{
		{3, "Symbol 1482", 5},
		{2, "Symbol 1484", 22},
		{1, "Symbol 1486", 38},
	}
	for _, tc := range checks {
		timeline, err := loadChildTransformTimeline("Symbol 1487", tc.symbol)
		if err != nil {
			t.Fatalf("digit %d timeline: %v", tc.digit, err)
		}
		if tc.frame >= len(timeline) || !timeline[tc.frame].Valid {
			t.Fatalf("digit %d missing at XFL frame %d", tc.digit, tc.frame)
		}
		if timeline[tc.frame].Alpha <= 0 {
			t.Fatalf("digit %d alpha=%v at XFL frame %d", tc.digit, timeline[tc.frame].Alpha, tc.frame)
		}
	}
}

func TestNonFinalDeathDoesNotStartSourceSoloCountdown(t *testing.T) {
	m := OriginalMap1()
	p1 := NewPlayer(1, m)
	p2 := NewPlayer(2, m)
	p1.Lives, p2.Lives = 3, 3
	p1.TotalLives, p2.TotalLives = 3, 3
	g := &Game{players: []*Player{p1, p2}, arena: m, GameMode: SourceGameModeNormal}
	p2.Kill(m) // respawns with two lives and remains in activeplayers in source
	g.updateMatchInteractions()
	if g.soloWinFrame != 0 || g.GameWin {
		t.Fatalf("non-final death started countdown: frame=%d gamewin=%v", g.soloWinFrame, g.GameWin)
	}
}
