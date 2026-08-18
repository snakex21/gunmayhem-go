package game

import "testing"

func TestSnowMapSourceTimeline(t *testing.T) {
	layers, err := loadChildTransformTimelines("Symbol 46", "Symbol 45")
	if err != nil {
		t.Fatal(err)
	}
	if len(layers) != 2 {
		t.Fatalf("mapfx_snow source layers=%d want2", len(layers))
	}
	for i, frames := range layers {
		if len(frames) != 127 {
			t.Fatalf("mapfx_snow layer%d timeline=%d want127", i, len(frames))
		}
	}
	// Layer 2 starts on Flash frame2; Layer 3 already exists on frame1.
	if layers[0][0].Valid || !layers[0][1].Valid {
		t.Fatal("mapfx_snow Layer2 source visibility mismatch at frames1/2")
	}
	if !layers[1][0].Valid {
		t.Fatal("mapfx_snow Layer3 must be visible on Flash frame1")
	}
}

func TestSnowMapSourceLoop(t *testing.T) {
	g := &Game{arena: Map{Number: 10}, assets: &Assets{MapSnowTimelines: [][]SourceTransformFrame{make([]SourceTransformFrame, 127), make([]SourceTransformFrame, 127)}}, mapFXFrame: 125}
	g.advanceSourceMapFX()
	if g.mapFXFrame != 1 {
		t.Fatalf("Flash frame127 gotoAndPlay(2): got zero-based frame %d want1", g.mapFXFrame)
	}
}
