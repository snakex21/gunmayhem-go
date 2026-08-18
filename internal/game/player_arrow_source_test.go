package game

import (
	"math"
	"testing"
)

func TestPlayerArrowUsesSourceOffscreenBounds(t *testing.T) {
	g := &Game{cameraX: 0, cameraY: 0}
	p := &Player{Active: true, X: 450, Y: 300}
	if visible, _, _ := g.sourcePlayerArrowState(p); visible {
		t.Fatal("center-screen player unexpectedly shows source player_arrow")
	}

	p.Y = 49
	visible, angle, distance := g.sourcePlayerArrowState(p)
	if !visible {
		t.Fatal("player above source y<=50 boundary did not show player_arrow")
	}
	if math.Abs(angle+math.Pi/2) > 0.001 {
		t.Fatalf("upper player arrow angle=%f want -pi/2", angle)
	}
	if distance != 251 {
		t.Fatalf("upper player arrow distance=%d want251", distance)
	}

	p.X, p.Y = 909, 300
	if visible, _, _ := g.sourcePlayerArrowState(p); visible {
		t.Fatal("source right edge is x>=910; x=909 must remain hidden")
	}
	p.X = 910
	if visible, _, _ := g.sourcePlayerArrowState(p); !visible {
		t.Fatal("source x>=910 boundary did not show player_arrow")
	}
}
