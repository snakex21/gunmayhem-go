package game

import "testing"

func TestSourceCameraEquation(t *testing.T) {
	g := &Game{players: []*Player{
		{Active: true, X: 100, Y: 100},
		{Active: true, X: 300, Y: 300},
	}}
	g.updateSourceCamera()
	// X target = (100+300)/-2 + 450 = 250; 0+(250/8)=31.25 -> round 31.
	if g.cameraX != 31 {
		t.Fatalf("source cameraX=%v want31", g.cameraX)
	}
	// Y target = (100+300)/-2 + 280 = 80; 0+(80/8)=10.
	if g.cameraY != 10 {
		t.Fatalf("source cameraY=%v want10", g.cameraY)
	}
}

func TestSourceCameraClamp(t *testing.T) {
	g := &Game{players: []*Player{{Active: true, X: -500, Y: -500}, {Active: true, X: 1500, Y: 900}}}
	g.updateSourceCamera()
	// Source clamps left/right to -100/1000 and high/low to 50/500.
	if g.cameraX != 0 { // ((-100+1000)/-2+450)==0
		t.Fatalf("clamped source cameraX=%v want0", g.cameraX)
	}
	if g.cameraY != 1 { // ((50+500)/-2+280)=5; 5/8=.625 -> round1
		t.Fatalf("clamped source cameraY=%v want1", g.cameraY)
	}
}
