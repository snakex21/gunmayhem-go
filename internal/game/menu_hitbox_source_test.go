package game

import "testing"

func TestMenuSourceHitboxBounds(t *testing.T) {
	for name, parts := range map[string][]string{
		"MainMenuPNG":   {"sprites", "DefineSprite_28_mainmenu", "1.png"},
		"CustomMenuPNG": {"sprites", "DefineSprite_1309", "1.png"},
	} {
		img := decodeOriginalPNG(parts...)
		if img == nil {
			t.Fatalf("%s missing", name)
		}
		ab, ok := alphaBounds(img)
		if !ok {
			t.Fatalf("%s has no alpha", name)
		}
		t.Logf("%s image=%v alpha=%v", name, img.Bounds(), ab)
	}

	a := LoadAssets()
	if a.CustomMenu != nil {
		t.Fatal("CustomMenu should be lazy before EnsureInteractions")
	}
	a.EnsureInteractions()
	for name, r := range map[string]*SourceRaster{"MainMenu": a.MainMenu, "CustomMenu": a.CustomMenu} {
		if r == nil || r.Image == nil {
			t.Fatalf("%s raster missing", name)
		}
		t.Logf("%s rasterBounds=%+v image=%v", name, r.Bounds, r.Image.Bounds())
	}
	for _, name := range []string{"Symbol 8", "Symbol 1282", "Symbol 1200", "Symbol 1048"} {
		b, err := sourceFrameVisualBounds(name, 0)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		t.Logf("%s bounds=%+v", name, b)
	}
	for _, name := range []string{"Symbol 28", "Symbol 1309"} {
		b, err := sourceFrameVisualBounds(name, 0)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		t.Logf("%s bounds=%+v", name, b)
	}
}
