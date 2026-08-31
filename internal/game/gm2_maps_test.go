package game

import (
	"math"
	"testing"
)

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
	rows := gm2CustomMapRows()
	if len(rows) != 10 {
		t.Fatalf("combined GM2 selector rows=%d want10 (random + 9 new)", len(rows))
	}
	first := gm2MapRowRect(0)
	hit, handled = g.gm2MapMenuHitAt(first.X+10, first.Y+5)
	if !handled || hit != menuHitGM2RandomAll {
		t.Fatalf("top GM2 random hit=%d handled=%v", hit, handled)
	}
	firstMap := gm2MapRowRect(1)
	hit, handled = g.gm2MapMenuHitAt(firstMap.X+10, firstMap.Y+5)
	if !handled || hit != menuHitGM2MapBase+1 {
		t.Fatalf("first GM2 map hit=%d handled=%v", hit, handled)
	}
	last := gm2MapRowRect(len(rows) - 1)
	if last.Y+last.H < 455 || last.Y+last.H > 495 {
		t.Fatalf("GM2 selector does not fill panel: last bottom=%.1f", last.Y+last.H)
	}
}

func TestGM2ListClearRectRemovesBakedGM1Rows(t *testing.T) {
	clear := gm2MapListClearRect()
	for i := range gm2CustomMapRows() {
		r := gm2MapRowRect(i)
		if !clear.Contains(r.X+r.W/2, r.Y+r.H/2) {
			t.Fatalf("GM2 row %d is outside clear strip: row=%+v clear=%+v", i, r, clear)
		}
	}
	// The last baked GM1 row lives around y=458; it must disappear on GM2.
	if !clear.Contains(-780, 465) {
		t.Fatalf("clear strip does not cover final baked GM1 row: %+v", clear)
	}
	// Tabs and BACK live outside this strip and must remain visible.
	if clear.Contains(-780, 80) || clear.Contains(-780, 540) {
		t.Fatalf("clear strip overlaps GM1/GM2 tabs or BACK area: %+v", clear)
	}
}

func TestGM2SourcePreviewBitmapsLoadLikeGM1(t *testing.T) {
	if _, err := findOriginalPathIn("gm2", "images", "1170.jpg"); err != nil {
		t.Skip("local unpacked GM2 preview bitmaps not available")
	}
	a := LoadAssets()
	a.EnsureInteractions()
	for n := 1; n <= 9; n++ {
		preview := a.MapPreviewImages[gm2MapID(n)]
		if preview == nil || preview.Image == nil {
			t.Fatalf("GM2 map %d source preview was not loaded", n)
		}
		b := preview.Image.Bounds()
		if b.Dx() <= 0 || b.Dy() <= 0 {
			t.Fatalf("GM2 map %d source preview has invalid size %v", n, b)
		}
	}
}

func TestGM2NewMapOrderMatchesSource(t *testing.T) {
	want := []string{"Castle", "Highway", "Sub Base", "Alien Planet", "Venice", "Avalon", "Space Station", "Ski Lift", "Jungle"}
	rows := gm2CustomMapRows()
	if len(rows) != len(want)+1 {
		t.Fatalf("GM2 selector rows=%d want%d", len(rows), len(want)+1)
	}
	if rows[0].value != customGM2RandomAll || rows[0].label != "Random GM2" {
		t.Fatalf("GM2 top row is not Random GM2: %+v", rows[0])
	}
	for i, name := range want {
		row := rows[i+1]
		if got := row.label; got != name {
			t.Fatalf("GM2 new map %d label=%q want=%q", i+1, got, name)
		}
		if row.value != gm2MapID(i+1) {
			t.Fatalf("GM2 new map %d runtime id=%d want=%d", i+1, row.value, gm2MapID(i+1))
		}
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

func TestGM2SteppedPlatformsMatchFlashVectorShapes(t *testing.T) {
	if _, err := findOriginalPathIn("gm2", "fla", "LIBRARY", "Symbol 1940.xml"); err != nil {
		t.Skip("local unpacked GM2 source not available")
	}

	// These are the five GM2 source frames containing non-rectangular platform
	// paths. Each extra rectangle is an exact slab of a step/notch that the old
	// bounding-box parser incorrectly filled with invisible collision.
	wantCounts := map[int]int{14: 6, 15: 16, 17: 8, 19: 6, 21: 7}
	for sourceNumber, want := range wantCounts {
		m, err := LoadGM2Map(sourceNumber)
		if err != nil {
			t.Fatalf("load GM2 map %d: %v", sourceNumber, err)
		}
		if got := len(m.Platforms); got != want {
			t.Errorf("GM2 map %d collision slabs=%d want=%d", sourceNumber, got, want)
		}
	}

	// GM2 map 19 / Symbol 1932 has a 16px-deep lower center step. In Flash,
	// the lower corners are empty. The previous AABB admitted both points.
	m, err := LoadGM2Map(19)
	if err != nil {
		t.Fatal(err)
	}
	if pointHitsPlatforms(m.Platforms, 50, 570) {
		t.Fatal("GM2 map 19 lower-left notch became invisible floor")
	}
	if !pointHitsPlatforms(m.Platforms, 500, 570) {
		t.Fatal("GM2 map 19 lower center step is missing")
	}
}

func TestOrthogonalEdgeRectsPreserveNotch(t *testing.T) {
	rects, ok := orthogonalEdgeRects("! 18322 7200| 0 7200| 0 7680| 2192 7680| 2192 8000| 16775 8000| 16775 7680| 18322 7680| 18322 7200")
	if !ok || len(rects) != 2 {
		t.Fatalf("notched edge produced %d rectangles, ok=%v: %+v", len(rects), ok, rects)
	}
	want := []Rect{{X: 0, Y: 360, W: 916.1, H: 24}, {X: 109.6, Y: 384, W: 729.15, H: 16}}
	for i := range want {
		if math.Abs(rects[i].X-want[i].X) > 1e-9 || math.Abs(rects[i].Y-want[i].Y) > 1e-9 ||
			math.Abs(rects[i].W-want[i].W) > 1e-9 || math.Abs(rects[i].H-want[i].H) > 1e-9 {
			t.Errorf("rect %d=%+v want=%+v", i, rects[i], want[i])
		}
	}
}

func TestGM2RespawnUsesGM2SourceHeight(t *testing.T) {
	p := &Player{DefaultWeapon: 2, GameMode: SourceGameModeNormal}
	p.Respawn(Map{Number: gm2MapID(1), SpawnMinX: 100, SpawnMaxX: 101})
	if p.Y != -500 {
		t.Fatalf("GM2 respawn Y=%v want source -500", p.Y)
	}
	p.Respawn(Map{Number: 1, SpawnMinX: 100, SpawnMaxX: 101})
	if p.Y != -1000 {
		t.Fatalf("GM1 respawn Y=%v want source -1000", p.Y)
	}
}

func TestGM2SpawnFallCannotTunnelThroughPlatform(t *testing.T) {
	if _, err := findOriginalPathIn("gm2", "fla", "LIBRARY", "Symbol 1940.xml"); err != nil {
		t.Skip("local unpacked GM2 source not available")
	}
	for sourceNumber := 1; sourceNumber <= 21; sourceNumber++ {
		m, err := LoadGM2Map(sourceNumber)
		if err != nil {
			t.Fatal(err)
		}
		for x := int(m.SpawnMinX); x < int(m.SpawnMaxX); x++ {
			y, vy := -500.0, 0.0
			landed := false
			for tick := 0; tick < 200 && y <= 1000; tick++ {
				oldY := y
				y += vy
				if pointHitsPlatforms(m.Platforms, float64(x), y) {
					landed = true
					break
				}
				if _, ok := platformTopCrossed(m.Platforms, float64(x), oldY, y); ok {
					landed = true
					break
				}
				vy += m.Gravity
			}
			if !landed {
				t.Fatalf("GM2 map %d spawn x=%d fell through every source platform", sourceNumber, x)
			}
		}
	}
}

func TestPlayerLandingSweepsThinPlatform(t *testing.T) {
	p := &Player{Active: true, X: 100, Y: 320, VY: 60, JumpNum: 1}
	p.resolvePlatforms([]Rect{{X: 0, Y: 280, W: 200, H: 24}})
	if p.Y != 280 || p.VY != 0 || !p.Grounded {
		t.Fatalf("swept landing failed: y=%v vy=%v grounded=%v", p.Y, p.VY, p.Grounded)
	}
}

func TestGM2JungleUsesStableFlashTimelineRegistration(t *testing.T) {
	if _, err := findOriginalPathIn("gm2", "sprites", "DefineSprite_1835", "9.png"); err != nil {
		t.Skip("local unpacked GM2 Jungle raster not available")
	}
	a := LoadAssets()
	id := gm2MapID(9)
	a.EnsureScene(id)
	got := []*SourceRaster{a.SceneBack[id], a.SceneMid[id], a.SceneFront[id]}
	for i, raster := range got {
		if raster == nil {
			t.Fatalf("Jungle scene layer %d is missing", i)
		}
		want := gm2SceneTimelineOrigins[i]
		if math.Abs(raster.Bounds.X-want[0]) > 1e-6 || math.Abs(raster.Bounds.Y-want[1]) > 1e-6 {
			t.Errorf("Jungle layer %d origin=(%.6f,%.6f) want stable timeline (%.6f,%.6f)",
				i, raster.Bounds.X, raster.Bounds.Y, want[0], want[1])
		}
	}
	// The broken per-frame foreground origin was about -540.92, placing the
	// walkable Jungle artwork roughly 91px away from its source collision.
	if a.SceneFront[id].Bounds.Y < -500 {
		t.Fatalf("Jungle foreground retained broken per-frame Y=%v", a.SceneFront[id].Bounds.Y)
	}
}
