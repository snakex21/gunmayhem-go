package game

import "testing"

func TestOriginalLegTimelines(t *testing.T) {
	leg1, err := loadChildTransformTimeline("Symbol 282", "Symbol 188")
	if err != nil {
		t.Fatal(err)
	}
	leg2, err := loadChildTransformTimeline("Symbol 189", "Symbol 188")
	if err != nil {
		t.Fatal(err)
	}
	if len(leg1) != 18 {
		t.Fatalf("Symbol 282 source timeline: got %d frames, want 18", len(leg1))
	}
	if len(leg2) != 19 {
		t.Fatalf("Symbol 189 source timeline: got %d frames, want 19", len(leg2))
	}
	// Frame 0/1 of both source timelines are held keyframes. This catches the
	// old bug where PNG color variants were advanced instead of XFL matrices.
	if leg1[0].Matrix != leg1[1].Matrix {
		t.Fatal("Symbol 282 frame 0 duration hold was not preserved")
	}
	if leg2[0].Matrix != leg2[1].Matrix {
		t.Fatal("Symbol 189 frame 0 duration hold was not preserved")
	}
}

func TestSourceDefaultPlayerColors(t *testing.T) {
	want := []int{2, 5, 8, 10}
	for i, color := range want {
		if got := sourceDefaultPlayerColor(i + 1); got != color {
			t.Fatalf("P%d color=%d want %d", i+1, got, color)
		}
	}
}
