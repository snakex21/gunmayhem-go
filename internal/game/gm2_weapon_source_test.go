package game

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestGM2WeaponAssetsAndTimelinesFromLocalSource(t *testing.T) {
	if _, err := findOriginalPathIn("gm2", "scripts", "DefineSprite_573_gun_A1", "frame_1", "DoAction.as"); err != nil {
		t.Skip("local unpacked GM2 source not available")
	}

	for number := 67; number <= 86; number++ {
		def, ok := WeaponByNumber(number)
		if !ok {
			t.Fatalf("GM2 weapon %d missing", number)
		}
		t.Run(fmt.Sprintf("%d_%s", number, def.SpriteDir), func(t *testing.T) {
			if img := decodeOriginalPNGIn("gm2", "sprites", def.SpriteDir, "1.png"); img == nil {
				t.Fatalf("GM2 first-frame PNG missing for %s", def.SpriteDir)
			}
			library := sourceLibraryNameFromSpriteDir(def.SpriteDir)
			libraryDir, err := findOriginalPathIn("gm2", "fla", "LIBRARY")
			if err != nil {
				t.Fatal(err)
			}
			timeline, err := loadWeaponTimelineFromDir(libraryDir, library)
			if err != nil {
				t.Fatal(err)
			}
			if timeline.TotalFrames < 2 {
				t.Fatalf("GM2 weapon %d timeline too short: %d", number, timeline.TotalFrames)
			}
			if bounds, err := sourceFrameVisualBoundsInDir(libraryDir, library, 0); err != nil {
				t.Fatalf("GM2 weapon %d XFL bounds: %v", number, err)
			} else if bounds.W <= 0 || bounds.H <= 0 {
				t.Fatalf("GM2 weapon %d has empty source bounds: %+v", number, bounds)
			}
		})
	}
}

func TestGM2CratePoolMatchesSource(t *testing.T) {
	got := GM2CrateWeapons()
	if len(got) != 77 || got[0] != 10 || got[len(got)-1] != 86 {
		t.Fatalf("GM2 crate pool=%v..%v len=%d, want 10..86 len=77", got[0], got[len(got)-1], len(got))
	}
	path, err := findOriginalPathIn("gm2", "scripts", "DefineSprite_784_crate", "frame_1", "DoAction.as")
	if err != nil {
		t.Skip("local unpacked GM2 source not available")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "randgun = random(77) + 10;") {
		t.Fatal("GM2 crate source no longer matches 10..86 pool")
	}
	if !gm2WeaponAssetsAvailable() {
		t.Fatal("local unpacked GM2 source was found but runtime asset availability check rejected it")
	}
	developed := developedCrateWeapons()
	if len(developed) != 77 || developed[0] != 10 || developed[len(developed)-1] != 86 {
		t.Fatalf("developed crate pool did not enable GM2: first=%d last=%d len=%d", developed[0], developed[len(developed)-1], len(developed))
	}
}

func TestGM2WeaponNumbersAreAppendOnly(t *testing.T) {
	for number := 1; number <= 66; number++ {
		// GM1 intentionally has no normal catalog entry for 7.
		if number == 7 {
			continue
		}
		if _, ok := WeaponByNumber(number); !ok {
			t.Fatalf("GM1 weapon %d disappeared while adding GM2", number)
		}
	}
	for number := 67; number <= 86; number++ {
		if def, ok := WeaponByNumber(number); !ok || def.Number != number {
			t.Fatalf("GM2 weapon numbering broken at %d: %+v ok=%v", number, def, ok)
		}
		if weaponAssetNamespace(number) != "gm2" {
			t.Fatalf("GM2 weapon %d not isolated in gm2 asset namespace", number)
		}
	}
}
