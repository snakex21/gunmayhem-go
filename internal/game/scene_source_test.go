package game

import "testing"

func TestSourceSceneFramesRenderFromXFL(t *testing.T) {
	for _, symbol := range []string{"Symbol 1352", "Symbol 1376", "Symbol 1390"} {
		for frame := 0; frame < 13; frame++ {
			if _, err := renderSolidXFLFrame(symbol, frame); err != nil {
				t.Fatalf("%s frame %d: %v", symbol, frame+1, err)
			}
		}
	}
}
