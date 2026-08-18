package game

import (
	"crypto/sha256"
	"encoding/xml"
	"image"
	"testing"
)

func cpuImageHash(img image.Image, crop image.Rectangle) [32]byte {
	h := sha256.New()
	for y := crop.Min.Y; y < crop.Max.Y; y++ {
		for x := crop.Min.X; x < crop.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			h.Write([]byte{byte(r >> 8), byte(g >> 8), byte(b >> 8), byte(a >> 8)})
		}
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func TestSelectableShirtsHaveDistinctSourceXFLFrames(t *testing.T) {
	doc, err := loadXFLVectorDoc("Symbol 224")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[[32]byte]int{}
	for frameIndex := 0; frameIndex < 15; frameIndex++ {
		h := sha256.New()
		for _, layer := range doc.Timeline.Layers {
			frame, ok := activeVectorFrame(layer.Frames, frameIndex)
			if !ok {
				continue
			}
			data, err := xml.Marshal(frame.Elements)
			if err != nil {
				t.Fatal(err)
			}
			h.Write(data)
		}
		var sum [32]byte
		copy(sum[:], h.Sum(nil))
		if prev, ok := seen[sum]; ok {
			t.Fatalf("shirt frame %d has identical XFL geometry to frame %d", frameIndex+1, prev)
		}
		seen[sum] = frameIndex + 1
	}
}

func TestStarterGunMenuThumbsUseDistinctTightGameplayArt(t *testing.T) {
	seen := map[[32]byte]int{}
	for number := 1; number <= 6; number++ {
		def, ok := WeaponByNumber(number)
		if !ok {
			t.Fatalf("starter gun %d missing definition", number)
		}
		img := decodeOriginalPNG("sprites", def.SpriteDir, "1", "1.png")
		if img == nil {
			t.Fatalf("starter gun %d source image failed to load", number)
		}
		crop, ok := alphaBounds(img)
		if !ok || crop.Empty() {
			t.Fatalf("starter gun %d has no visible pixels", number)
		}
		if crop.Dx() < 15 || crop.Dy() < 8 {
			t.Fatalf("starter gun %d visible crop unexpectedly tiny: %v", number, crop)
		}
		h := cpuImageHash(img, crop)
		if prev, ok := seen[h]; ok {
			t.Fatalf("starter gun %d has identical visible pixels to gun %d", number, prev)
		}
		seen[h] = number
	}
}
