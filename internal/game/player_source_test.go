package game

import "testing"

func TestPlayerFrameHitboxMatchesSymbol691(t *testing.T) {
	p := &Player{X: 100, Y: 200, PlayerScale: 0.8}
	h := p.Hitbox()
	if h.X != 80 || h.Y != 144 || h.W != 40 || h.H != 56 {
		t.Fatalf("normal source frame hitbox=%+v want x80 y144 w40 h56", h)
	}
	p.PlayerScale = 0.5
	h = p.Hitbox()
	if h.X != 87.5 || h.Y != 165 || h.W != 25 || h.H != 35 {
		t.Fatalf("mini source frame hitbox=%+v want x87.5 y165 w25 h35", h)
	}
}

func TestStandingPlayerDoesNotSinkThroughPlatform(t *testing.T) {
	m := OriginalMap1()
	p := &Player{
		X:           m.Platforms[0].X + 100,
		Y:           m.Platforms[0].Y,
		VY:          0,
		JumpNum:     2,
		Grounded:    true,
		Active:      true,
		Weight:      1,
		PlayerScale: 0.8,
		Alpha:       1,
		SpeedTime:   -1,
		MiniTime:    -1,
		MiniMulti:   1,
	}
	wantY := p.Y
	for i := 0; i < 2000; i++ {
		p.Update(m, nil, nil)
		if !p.Active {
			t.Fatalf("player died while standing on platform at tick %d", i)
		}
	}
	if p.Y != wantY {
		t.Fatalf("standing player drifted vertically: got %.6f want %.6f", p.Y, wantY)
	}
	if p.FreePass {
		t.Fatal("standing player incorrectly entered freepass state")
	}
	if p.JumpNum != 2 || !p.Grounded {
		t.Fatalf("standing player lost grounded state: jumpnum=%d grounded=%v", p.JumpNum, p.Grounded)
	}
}

func TestMiniUsesSeparateSourceMultiAndPlayerScale(t *testing.T) {
	p := &Player{MiniTime: 261, MiniMulti: 0.72, PlayerScale: 0.56, Alpha: 1}
	p.updatePowerupTimers()
	if p.MiniTime != 260 || p.MiniMulti != 0.6 || p.PlayerScale != 0.5 {
		t.Fatalf("MINI source snap at 260: time=%d minimulti=%v playerScale=%v", p.MiniTime, p.MiniMulti, p.PlayerScale)
	}
}
