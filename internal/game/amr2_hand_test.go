package game

import (
	"image/color"
	"strconv"
	"testing"
)

func TestAMR2BoltHandUsesOneRequestedPlayerColor(t *testing.T) {
	tint := sourcePlayerPalette[2]
	for frame := 2; frame < 30; frame++ {
		src := decodeOriginalPNGIn("gm2", "sprites", "DefineSprite_601_gun_C5", strconv.Itoa(frame+1)+".png")
		if src == nil {
			t.Fatalf("AMR-2 frame %d missing", frame+1)
		}
		sourceFill, ok := amr2ExportedHandColor(frame)
		if !ok {
			t.Fatalf("AMR-2 frame %d has no expected exported hand color", frame+1)
		}
		fixed, changed := recolorAMR2EmbeddedHand(src, frame, tint)
		if !changed {
			t.Fatalf("AMR-2 frame %d embedded hand was not detected", frame+1)
		}
		seenTint := 0
		oldFill := 0
		b := fixed.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				c := color.NRGBAModel.Convert(fixed.At(x, y)).(color.NRGBA)
				if c.R == tint.R && c.G == tint.G && c.B == tint.B {
					seenTint++
				}
				if sourceFill != tint && c.R == sourceFill.R && c.G == sourceFill.G && c.B == sourceFill.B {
					oldFill++
				}
			}
		}
		if seenTint < 20 {
			t.Errorf("AMR-2 frame %d has only %d pixels of requested hand color", frame+1, seenTint)
		}
		if oldFill != 0 {
			t.Errorf("AMR-2 frame %d retains %d pixels of its exported rainbow hand", frame+1, oldFill)
		}
	}
}

func TestAMR2BoltHandDetectionUsesPixelsInsteadOfTimelineGuess(t *testing.T) {
	// File 24 is the first frame after the nested 21-tick color loop. Passing a
	// deliberately unrelated outer frame proves recoloring follows the pixels,
	// not a fragile parallel timeline calculation.
	src := decodeOriginalPNGIn("gm2", "sprites", "DefineSprite_601_gun_C5", "24.png")
	if src == nil {
		t.Fatal("AMR-2 looped hand frame missing")
	}
	actual, _, _, _, _, ok := findAMR2ExportedHand(src)
	if !ok || actual != amr2ExportedHandPalette[0] {
		t.Fatalf("detected AMR-2 hand=%v ok=%v want first pink palette entry", actual, ok)
	}
	tint := sourcePlayerPalette[0]
	fixed, changed := recolorAMR2EmbeddedHand(src, 22, tint) // intentionally wrong prediction
	if !changed {
		t.Fatal("pixel-detected AMR-2 hand was not recolored")
	}
	seenTint := 0
	b := fixed.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(fixed.At(x, y)).(color.NRGBA)
			if c.R == tint.R && c.G == tint.G && c.B == tint.B {
				seenTint++
			}
			if c.R == actual.R && c.G == actual.G && c.B == actual.B {
				t.Fatalf("looped AMR-2 frame retained rainbow hand color at %d,%d", x, y)
			}
		}
	}
	if seenTint < 20 {
		t.Fatalf("looped AMR-2 frame has only %d requested-color pixels", seenTint)
	}
}
