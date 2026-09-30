package game

import (
	"errors"
	"image"
	"image/color"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

func TestSourceFontWholeLabelFallback(t *testing.T) {
	parsed, err := parseSourceFont(readLegacySourceFontFixture(t, "10_Tw Cen MT.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	primary, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: 20, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		t.Fatal(err)
	}
	defer primary.Close()
	fallback, err := sourceGoRegularFace(20)
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"JOIN SESSION", "PLAYER 3", "PAGE 10/20"} {
		t.Run(label, func(t *testing.T) {
			if sourceFaceCoversText(primary, label) {
				t.Fatal("fixture should demonstrate missing subset glyphs")
			}
			selected, err := sourceFaceForText(primary, nil, 20, label)
			if err != nil || selected != fallback || !sourceFaceCoversText(selected, label) {
				t.Fatalf("complete whole-label fallback not selected: %v", err)
			}
			// Measure and draw the very same selected face, including centering.
			width := font.MeasureString(selected, label).Ceil()
			canvas := image.NewAlpha(image.Rect(0, 0, width+40, 48))
			dot := fixed.P((canvas.Rect.Dx()-width)/2, 30)
			drawer := font.Drawer{Dst: canvas, Src: image.NewUniform(color.White), Face: selected, Dot: dot}
			drawer.DrawString(label)
			painted := 0
			for y := 0; y < canvas.Rect.Dy(); y++ {
				for x := 0; x < canvas.Rect.Dx(); x++ {
					if canvas.AlphaAt(x, y).A != 0 {
						painted++
						if x == 0 || y == 0 || x == canvas.Rect.Max.X-1 || y == canvas.Rect.Max.Y-1 {
							t.Fatal("fallback label clipped at canvas edge")
						}
					}
				}
			}
			if painted == 0 || width != font.MeasureString(fallback, label).Ceil() {
				t.Fatal("fallback measurement and rendering disagreed")
			}
		})
	}
	for _, label := range []string{"GAME OVER", "LEVEL UP", "GAME\nOVER", ""} {
		selected, err := sourceFaceForText(primary, nil, 20, label)
		if err != nil || selected != primary {
			t.Fatalf("source face replaced for covered label %q: %v", label, err)
		}
	}
	// Likewise, an ordinary native/full font must keep its identity if covered.
	selected, err := sourceFaceForText(fallback, nil, 20, "JOIN SESSION")
	if err != nil || selected != fallback {
		t.Fatal("covered ordinary face replaced")
	}
}

func TestSourceFontMissingFaceFallback(t *testing.T) {
	fallback, err := sourceGoRegularFace(20)
	if err != nil {
		t.Fatal(err)
	}
	parseErr := errors.New("invalid font")
	for _, inputErr := range []error{parseErr, nil} {
		selected, err := sourceFaceForText(nil, inputErr, 20, "0123456789")
		if err != nil || selected != fallback {
			t.Fatalf("missing-face fallback: %v", err)
		}
	}
	// Missing glyphs in both fonts are not falsely reported as covered.
	const unsupported = "\U0001f680"
	if sourceFaceCoversText(fallback, unsupported) {
		t.Fatal("test expects Go Regular not to contain this symbol")
	}
	selected, err := sourceFaceForText(nil, parseErr, 20, unsupported)
	if selected != nil || err != parseErr {
		t.Fatal("unsupported fallback should preserve the original failure")
	}
	selected, err = sourceFaceForText(fallback, nil, 20, unsupported)
	if selected != fallback || err != nil {
		t.Fatal("unsupported glyph should preserve the original face")
	}
}

func TestSourceFontTextAwareMenuAndHUD(t *testing.T) {
	for _, tc := range []struct {
		kind  sourceMenuFontKind
		label string
	}{
		{menuFontTwCen, "JOIN SESSION"}, {menuFontTwCen, "PLAYER 3"},
		{menuFontTwCen, "PAGE 10/20"}, {menuFontArial, "0123456789"},
		{menuFontArialBold, "10"},
	} {
		face, err := sourceMenuFaceForText(tc.kind, 20, tc.label)
		if err != nil || !sourceFaceCoversText(face, tc.label) {
			t.Fatalf("menu %d label %q is not covered: %v", tc.kind, tc.label, err)
		}
	}
	for _, label := range []string{"X2", "LEVEL UP", "GAME OVER", "PLAYER 3"} {
		face, err := sourceHUDFaceForText(20, label)
		if err != nil || !sourceFaceCoversText(face, label) {
			t.Fatalf("HUD label %q is not covered: %v", label, err)
		}
	}
	first, err := sourceGoRegularFace(13.15)
	second, err2 := sourceGoRegularFace(13.15)
	if err != nil || err2 != nil || first != second {
		t.Fatal("fallback did not retain fractional-size cache entry")
	}
}
