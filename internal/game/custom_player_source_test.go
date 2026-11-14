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
	// An empty slot accepts the HUMAN action code.
	if !g.activateCustomPlayerSlot(playerSlotHit(2, playerSlotActionHuman)) || g.customPlayers[2].Type != 1 {
		t.Fatalf("empty slot HUMAN action: type=%d", g.customPlayers[2].Type)
	}
}
