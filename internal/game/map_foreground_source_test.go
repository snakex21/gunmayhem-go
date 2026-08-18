package game

import "testing"

func TestAllSourceMapForegroundFramesRenderFromXFL(t *testing.T) {
	for frame := 0; frame < 13; frame++ {
		r, err := renderSolidXFLFrame("Symbol 1390", frame)
		if err != nil {
			t.Fatalf("Symbol 1390 frame %d: %v", frame+1, err)
		}
		if r == nil || r.Image == nil || r.Bounds.W <= 0 || r.Bounds.H <= 0 {
			t.Fatalf("Symbol 1390 frame %d rendered empty", frame+1)
		}
	}
}
