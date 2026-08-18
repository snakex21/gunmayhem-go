package game

import "testing"

func TestGM2MapRuntimeIDsDoNotCollideWithGM1(t *testing.T) {
	seen := map[int]bool{}
	for n := 1; n <= 13; n++ {
		seen[n] = true
	}
	for n := 1; n <= 21; n++ {
		id := gm2MapID(n)
		if seen[id] {
			t.Fatalf("GM2 map %d runtime id %d collides with GM1", n, id)
		}
		if !isGM2MapID(id) || gm2SourceMapNumber(id) != n {
			t.Fatalf("GM2 map id round trip failed: source=%d runtime=%d back=%d", n, id, gm2SourceMapNumber(id))
		}
		seen[id] = true
	}
}

func TestGM2MapSelectorHitboxes(t *testing.T) {
	if !gm2MapAssetsAvailable() {
		t.Skip("local unpacked GM2 map assets not available")
	}
	g := &Game{customMapSetGM2: false, customMap: 1}
	_, gm2Toggle := gm2MapSetButtonRects()
	hit, handled := g.gm2MapMenuHitAt(gm2Toggle.X+10, gm2Toggle.Y+10)
	if !handled || hit != menuHitMapSetGM2 {
		t.Fatalf("GM2 toggle hit=%d handled=%v", hit, handled)
	}
	g.customMapSetGM2 = true
	first := gm2MapRowRect(0)
	hit, handled = g.gm2MapMenuHitAt(first.X+10, first.Y+5)
	if !handled || hit != menuHitGM2MapBase+1 {
		t.Fatalf("first GM2 map hit=%d handled=%v", hit, handled)
	}
	last := gm2MapRowRect(22)
	hit, handled = g.gm2MapMenuHitAt(last.X+10, last.Y+5)
	if !handled || hit != menuHitGM2RandomAll {
		t.Fatalf("GM2 random-all hit=%d handled=%v", hit, handled)
	}
}

func TestLoadAllGM2MapsFromLocalSource(t *testing.T) {
	if _, err := findOriginalPathIn("gm2", "fla", "LIBRARY", "Symbol 1940.xml"); err != nil {
		t.Skip("local unpacked GM2 source not available")
	}
	libraryDir, err := findOriginalPathIn("gm2", "fla", "LIBRARY")
	if err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 21; n++ {
		m, err := LoadGM2Map(n)
		if err != nil {
			t.Fatalf("GM2 map %d (%s): %v", n, gm2MapDisplayNames[n], err)
		}
		if m.Number != gm2MapID(n) {
			t.Fatalf("GM2 map %d runtime number=%d", n, m.Number)
		}
		if len(m.Platforms) == 0 {
			t.Fatalf("GM2 map %d has no platform rectangles", n)
		}
		if m.SpawnMaxX <= m.SpawnMinX || m.CrateMaxX <= m.CrateMinX {
			t.Fatalf("GM2 map %d invalid spawn/crate bounds: %+v", n, m)
		}
		if m.LowestY <= 0 {
			t.Fatalf("GM2 map %d invalid lowest=%v", n, m.LowestY)
		}
		wantGravity := DefaultGravity
		if n == 13 {
			wantGravity = 0.6
		}
		if m.Gravity != wantGravity {
			t.Fatalf("GM2 map %d gravity=%v want=%v", n, m.Gravity, wantGravity)
		}
		for _, symbol := range []string{"Symbol 1642", "Symbol 1690", "Symbol 1835"} {
			if bounds, err := sourceFrameVisualBoundsInDir(libraryDir, symbol, n-1); err != nil {
				t.Fatalf("GM2 map %d scene %s: %v", n, symbol, err)
			} else if bounds.W <= 0 || bounds.H <= 0 {
				t.Fatalf("GM2 map %d scene %s empty bounds: %+v", n, symbol, bounds)
			}
		}
	}
}
