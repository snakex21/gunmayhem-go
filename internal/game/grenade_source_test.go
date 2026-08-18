package game

import "testing"

func TestGrenadeSourceRegistration(t *testing.T) {
	visible, err := sourceFrameVisualBounds("Symbol 665", 0)
	if err != nil {
		t.Fatal(err)
	}
	r := loadSourceRaster("Symbol 665", 0, "sprites", "DefineSprite_665_wep_grenade", "1.png")
	if r == nil {
		t.Fatal("missing grenade raster")
	}
	t.Logf("grenade visible=%+v rasterBounds=%+v image=%v", visible, r.Bounds, sourceRasterImageSize(r))
}
