package game

import (
	"math"
	"testing"
)

func TestSourceRasterRegistrationMatchesFrameSource(t *testing.T) {
	cases := []struct {
		name    string
		library string
		frame   int
		parts   []string
	}{
		{"body", "Symbol 240", 0, []string{"sprites", "DefineSprite_240", "1.png"}},
		{"hand", "Symbol 111", 0, []string{"sprites", "DefineSprite_111", "1", "1.png"}},
		{"leg1", "Symbol 282", 0, []string{"sprites", "DefineSprite_282", "1", "1.png"}},
		{"leg2", "Symbol 189", 0, []string{"sprites", "DefineSprite_189", "1", "1.png"}},
		{"scene1", "Symbol 1352", 0, []string{"sprites", "DefineSprite_1352", "1", "1.png"}},
		{"m1911", "Symbol 416", 1, []string{"sprites", "DefineSprite_416_gun_m1911", "1", "2.png"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := loadSourceRaster(tc.library, tc.frame, tc.parts...)
			if r == nil || r.Image == nil {
				t.Fatalf("missing source raster for %s", tc.library)
			}
			visible, err := sourceFrameVisualBounds(tc.library, tc.frame)
			if err != nil {
				t.Fatalf("source bounds: %v", err)
			}
			img := decodeOriginalPNG(tc.parts...)
			alpha, ok := alphaBounds(img)
			if !ok {
				t.Fatalf("empty raster")
			}

			gotMinX := r.Bounds.X + float64(alpha.Min.X)
			gotMinY := r.Bounds.Y + float64(alpha.Min.Y)
			gotMaxX := r.Bounds.X + float64(alpha.Max.X)
			gotMaxY := r.Bounds.Y + float64(alpha.Max.Y)
			wantMaxX := visible.X + visible.W
			wantMaxY := visible.Y + visible.H

			if math.Abs(gotMinX-visible.X) > 2 || math.Abs(gotMinY-visible.Y) > 2 ||
				math.Abs(gotMaxX-wantMaxX) > 2 || math.Abs(gotMaxY-wantMaxY) > 2 {
				t.Fatalf("registration mismatch: source=(%.1f,%.1f)-(%.1f,%.1f) raster=(%.1f,%.1f)-(%.1f,%.1f)",
					visible.X, visible.Y, wantMaxX, wantMaxY,
					gotMinX, gotMinY, gotMaxX, gotMaxY)
			}
		})
	}
}
