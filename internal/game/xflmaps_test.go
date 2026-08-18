package game

import "testing"

func TestLoadOriginalMaps(t *testing.T) {
	maps, err := LoadOriginalMaps()
	if err != nil {
		t.Fatal(err)
	}
	if len(maps) != 13 {
		t.Fatalf("expected 13 maps, got %d", len(maps))
	}
	if got := len(maps[1].Platforms); got != 6 {
		t.Fatalf("map 1: expected 6 platform rectangles, got %d", got)
	}
	if maps[4].Gravity != 0.6 {
		t.Fatalf("map 4: expected gravity 0.6, got %v", maps[4].Gravity)
	}
	for n := 1; n <= 13; n++ {
		m, ok := maps[n]
		if !ok {
			t.Fatalf("missing map %d", n)
		}
		if len(m.Platforms) == 0 || m.SpawnMaxX <= m.SpawnMinX || m.CrateMaxX <= m.CrateMinX {
			t.Fatalf("map %d has invalid extracted geometry: %+v", n, m)
		}
	}
}
