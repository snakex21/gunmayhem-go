package game

import "testing"

func TestHUDLastLifeSourcePlayhead(t *testing.T) {
	m := OriginalMap1()
	p := NewPlayer(1, m)
	p.Lives = 2
	g := &Game{players: []*Player{p}, GameMode: SourceGameModeNormal}
	g.resetHUDLastLife()

	if g.hudLastLifeFrame[0] != 0 || g.hudLastLifePlaying[0] {
		t.Fatalf("initial lastlife frame=%d playing=%v", g.hudLastLifeFrame[0], g.hudLastLifePlaying[0])
	}
	if g.hudLastLifeShouldDraw(0) {
		t.Fatal("stopped source frame1 must not leak LAST LIFE while player has more than one life")
	}
	p.Lives = 1
	g.updateHUDLastLife()
	if g.hudLastLifeFrame[0] != 0 || !g.hudLastLifePlaying[0] {
		t.Fatalf("one-life trigger frame=%d playing=%v, want source frame1 playing", g.hudLastLifeFrame[0], g.hudLastLifePlaying[0])
	}
	if g.hudLastLifeShouldDraw(0) {
		t.Fatal("source frame1 is the hidden resting frame; visible animation begins after it advances")
	}
	g.updateHUDLastLife()
	if !g.hudLastLifeShouldDraw(0) || g.hudLastLifeFrame[0] != 1 {
		t.Fatalf("LAST LIFE first visible frame=%d draw=%v, want frame2 visible", g.hudLastLifeFrame[0], g.hudLastLifeShouldDraw(0))
	}
	for i := 0; i < 88; i++ {
		g.updateHUDLastLife()
	}
	if g.hudLastLifeFrame[0] != 89 || g.hudLastLifePlaying[0] {
		t.Fatalf("LAST LIFE stop frame=%d playing=%v, want source frame90 stopped", g.hudLastLifeFrame[0], g.hudLastLifePlaying[0])
	}
	if g.hudLastLifeShouldDraw(0) {
		t.Fatal("source frame90 has returned behind the HUD card and must no longer be visibly exposed")
	}
	// Further HUD/gameplay ticks at one life freeze the same hidden frame90;
	// they must neither restart nor expose the warning again.
	for i := 0; i < 20; i++ {
		g.updateHUDLastLife()
	}
	if g.hudLastLifeFrame[0] != 89 || g.hudLastLifePlaying[0] || g.hudLastLifeShouldDraw(0) {
		t.Fatalf("LAST LIFE changed after source stop: frame=%d playing=%v draw=%v",
			g.hudLastLifeFrame[0], g.hudLastLifePlaying[0], g.hudLastLifeShouldDraw(0))
	}

	p.Lives = 0
	g.updateHUDLastLife()
	if g.hudLastLifeFrame[0] != 89 || !g.hudLastLifePlaying[0] {
		t.Fatalf("game-over trigger frame=%d playing=%v, want hidden source frame90 resumed", g.hudLastLifeFrame[0], g.hudLastLifePlaying[0])
	}
	if g.hudLastLifeShouldDraw(0) {
		t.Fatal("source frame90 must stay hidden while HUD already displays 0 lives")
	}
	g.updateHUDLastLife()
	if g.hudLastLifeFrame[0] != 90 || !g.hudLastLifeShouldDraw(0) {
		t.Fatalf("GAME OVER first frame=%d draw=%v, want frame91 visible", g.hudLastLifeFrame[0], g.hudLastLifeShouldDraw(0))
	}
	for i := 0; i < 89; i++ {
		g.updateHUDLastLife()
	}
	if g.hudLastLifeFrame[0] != 179 || g.hudLastLifePlaying[0] {
		t.Fatalf("GAME OVER stop frame=%d playing=%v, want source frame180 stopped", g.hudLastLifeFrame[0], g.hudLastLifePlaying[0])
	}
}

func TestHUDLastLifeFinalDeathSkipsUnfinishedWarning(t *testing.T) {
	m := OriginalMap1()
	p := NewPlayer(1, m)
	p.Lives = 1
	g := &Game{players: []*Player{p}, GameMode: SourceGameModeNormal}
	g.resetHUDLastLife()
	g.updateHUDLastLife()
	for i := 0; i < 10; i++ {
		g.updateHUDLastLife()
	}
	if g.hudLastLifeFrame[0] <= 0 || g.hudLastLifeFrame[0] >= 89 {
		t.Fatalf("fixture is not inside LAST LIFE animation: frame=%d", g.hudLastLifeFrame[0])
	}
	p.Lives = 0
	g.updateHUDLastLife()
	if g.hudLastLifeFrame[0] != 89 || !g.hudLastLifePlaying[0] || g.hudLastLifeShouldDraw(0) {
		t.Fatalf("0 lives must hide unfinished LAST LIFE: frame=%d playing=%v draw=%v",
			g.hudLastLifeFrame[0], g.hudLastLifePlaying[0], g.hudLastLifeShouldDraw(0))
	}
}

func TestHUDLastLifeDrawRequiresExactlyOneLife(t *testing.T) {
	m := OriginalMap1()
	p := NewPlayer(1, m)
	g := &Game{players: []*Player{p}, GameMode: SourceGameModeNormal}
	g.hudLastLifeFrame[0] = 20
	g.hudLastLifePlaying[0] = true

	for _, lives := range []int{0, 2, 3, 10} {
		p.Lives = lives
		if g.hudLastLifeShouldDraw(0) {
			t.Fatalf("LAST LIFE leaked with Lives=%d", lives)
		}
	}
	p.Lives = 1
	if !g.hudLastLifeShouldDraw(0) {
		t.Fatal("LAST LIFE should draw when Lives==1")
	}
}

func TestHUDLastLifeDoesNotLeakIntoGunGame(t *testing.T) {
	g := &Game{GameMode: SourceGameModeGunGame}
	for _, frame := range []int{1, 20, 88, 89, 90, 179} {
		g.hudLastLifeFrame[0] = frame
		g.hudLastLifePlaying[0] = true
		if g.hudLastLifeShouldDraw(0) {
			t.Fatalf("Gun Game leaked LAST LIFE/GAME OVER frame %d", frame+1)
		}
	}
	g.hudLastLifeFrame[0] = 180
	if !g.hudLastLifeShouldDraw(0) {
		t.Fatal("Gun Game LEVEL UP frame181 should remain drawable")
	}
}

func TestHUDLastLifeExportFramesExist(t *testing.T) {
	a := LoadAssets()
	for _, frame := range []int{0, 1, 88, 89, 90, 179, 180, 269} {
		r := a.HUDLastLifeFrame(frame)
		if r == nil || r.Image == nil {
			t.Fatalf("missing Symbol1456 source frame %d", frame+1)
		}
	}
	// Static-text registration is negative in the XFL. A (0,0) registration
	// reproduces the old bug where LIFE leaked below the HUD card.
	r := a.HUDLastLifeFrame(1)
	if r.Bounds.X > -60 || r.Bounds.Y > -20 {
		t.Fatalf("LAST LIFE registration=%+v, want negative source registration", r.Bounds)
	}
}
