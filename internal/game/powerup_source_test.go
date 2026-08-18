package game

import (
	"hash/crc32"
	"image"
	"testing"
)

func imageCRC(img image.Image) uint32 {
	b := img.Bounds()
	data := make([]byte, 0, b.Dx()*b.Dy()*4)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			data = append(data, byte(r>>8), byte(g>>8), byte(bl>>8), byte(a>>8))
		}
	}
	return crc32.ChecksumIEEE(data)
}

func TestPowerupSourceTimelines(t *testing.T) {
	stuff, err := loadChildTransformTimeline("Symbol 734", "Symbol 730")
	if err != nil {
		t.Fatal(err)
	}
	aura, err := loadChildTransformTimeline("Symbol 734", "Symbol 733")
	if err != nil {
		t.Fatal(err)
	}
	flash, err := loadChildTransformTimeline("Symbol 734", "Symbol 710")
	if err != nil {
		t.Fatal(err)
	}
	if len(stuff) != 98 || len(aura) != 98 || len(flash) != 98 {
		t.Fatalf("Symbol 734 child timelines: stuff=%d aura=%d flash=%d, want 98 each", len(stuff), len(aura), len(flash))
	}
	if aura[18].Valid {
		t.Fatal("aura must still be absent on zero-based frame 18 (Flash frame 19)")
	}
	if !aura[19].Valid || !stuff[19].Valid || !flash[19].Valid {
		t.Fatal("powerup source children must be present from Flash frame 20")
	}
}

func TestPowerupIconSourceFramesAreDistinct(t *testing.T) {
	seen := map[uint32]int{}
	for frame := 0; frame < 7; frame++ {
		img := decodeOriginalPNG("sprites", "DefineSprite_730", "1", itoaFast(frame+1)+".png")
		if img == nil {
			t.Fatalf("missing Symbol730 frame %d", frame+1)
		}
		crc := imageCRC(img)
		t.Logf("powerup frame %d crc=%08x bounds=%v", frame+1, crc, img.Bounds())
		if prev, ok := seen[crc]; ok {
			t.Logf("exported PNG duplicate: frames %d and %d", prev+1, frame+1)
		}
		seen[crc] = frame
		if frame != 4 {
			r, err := renderSolidXFLFrame("Symbol 730", frame)
			if err != nil || r == nil || r.Image == nil {
				t.Fatalf("XFL powerup frame %d: %v", frame+1, err)
			}
		}
	}
}

func TestPowerupPlayheadsFollowSourceStopsAndLoop(t *testing.T) {
	pu := Powerup{Frame: 96, FlashFrame: 70, FlashPlaying: true}
	advancePowerupPlayheads(&pu)
	if pu.Frame != 19 {
		t.Fatalf("Symbol 734 frame 98 gotoAndPlay(20) produced zero-based %d, want 19", pu.Frame)
	}
	if pu.FlashFrame != 71 || pu.FlashPlaying {
		t.Fatalf("Symbol 710 did not stop at source frame 72: frame=%d playing=%v", pu.FlashFrame, pu.FlashPlaying)
	}
}
