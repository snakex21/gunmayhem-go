package game

import "testing"

func TestCustomPlayerSourceCardBounds(t *testing.T) {
	r, err := sourceFrameVisualBounds("Symbol 1302", 2)
	if err != nil { t.Fatal(err) }
	t.Logf("Symbol1302 frame3 bounds=%+v raster=%+v", r, gTestRasterBoundsForCustomPlayer())
	for _, name := range []string{"Symbol 1305", "Symbol 1308"} {
		for _, frame := range []int{0,1} {
			if b, e := sourceFrameVisualBounds(name, frame); e == nil { t.Logf("%s frame%d=%+v", name, frame+1, b) }
		}
	}
}

func gTestRasterBoundsForCustomPlayer() Rect {
	r := loadSourceRaster("Symbol 1302", 2, "sprites", "DefineSprite_1302", "1", "3.png")
	if r == nil { return Rect{} }
	return r.Bounds
}

func TestCustomPlayerDefaultsAndSlotHits(t *testing.T) {
	g := &Game{}
	g.initCustomPlayerSetup()
	if g.activeCustomPlayerCount() != 2 { t.Fatalf("default active=%d want2", g.activeCustomPlayerCount()) }
	// Empty source slot3 at X=460,Y=500 exposes HUMAN/AI buttons.
	if got := g.customPlayerSetupHitAt(460+20+50, 500-254.95+10); got != playerSlotHit(2, playerSlotActionHuman) {
		t.Fatalf("empty human hit=%d", got)
	}
}
