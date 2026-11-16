package game

import "testing"

// The main menu logo is cropped from the Flash main-menu raster (Symbol 28);
// the credits doodles from Symbol 1046. Both must load.
func TestMenuSourceRastersLoad(t *testing.T) {
	img := decodeOriginalPNG("sprites", "DefineSprite_28_mainmenu", "1.png")
	if img == nil {
		t.Fatal("MainMenuPNG missing")
	}
	if _, ok := alphaBounds(img); !ok {
		t.Fatal("MainMenuPNG has no alpha")
	}
	a := LoadAssets()
	if a.CreditsMenu != nil {
		t.Fatal("CreditsMenu should be lazy before EnsureInteractions")
	}
	a.EnsureInteractions()
	for name, r := range map[string]*SourceRaster{"MainMenu": a.MainMenu, "CreditsMenu": a.CreditsMenu} {
		if r == nil || r.Image == nil {
			t.Fatalf("%s raster missing", name)
		}
	}
	for _, name := range []string{"Symbol 28", "Symbol 1046"} {
		if _, err := sourceFrameVisualBounds(name, 0); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}
