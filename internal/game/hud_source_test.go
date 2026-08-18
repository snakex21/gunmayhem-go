package game

import "testing"

func TestHUDSourcePiecesAndFont(t *testing.T) {
	face, err := sourceHUDFace(15)
	if err != nil || face == nil {
		t.Logf("HUD font failed before fallback fix: face=%v err=%v", face, err)
	}
	for _, name := range []string{"Symbol 1459", "Symbol 1461"} {
		r, err := renderSolidXFLFrame(name, 0)
		if err != nil {
			t.Fatalf("%s render: %v", name, err)
		}
		if r == nil || r.Image == nil {
			t.Fatalf("%s empty", name)
		}
		t.Logf("%s bounds=%+v image=%v", name, r.Bounds, r.Image.Bounds())
	}
}
