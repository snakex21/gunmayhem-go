package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// libraryShowWeapon opens the ALL category page that lists gun and returns
// its kit tile.
func libraryShowWeapon(t *testing.T, g *Game, gun int) uiItem {
	t.Helper()
	g.screen = screenGunLibrary
	g.setGunLibraryView(gunLibraryViewWeapons)
	g.gunLibraryCategory = 0
	for page := 0; page < organizedGunLibraryPageCount(0); page++ {
		g.gunLibraryPage = page
		for _, n := range g.gunLibraryPageWeapons() {
			if n == gun {
				g.runGunLibraryUpdate(uiInput{X: -10, Y: -10})
				it, ok := g.libraryUI.item(fmt.Sprintf("lib.gun.%d", gun))
				if !ok {
					t.Fatalf("gun %d listed on page %d but has no tile", gun, page)
				}
				return it
			}
		}
	}
	t.Fatalf("gun %d has no Gun Library tile", gun)
	return uiItem{}
}

func TestGunLibraryRespectsCampaignUnlockArray(t *testing.T) {
	g := New()
	for i := range g.campaignGuns {
		g.campaignGuns[i] = true
	}
	if it := libraryShowWeapon(t, g, 1); it.disabled {
		t.Fatal("source handgun should always be selectable")
	}
	if it := libraryShowWeapon(t, g, 10); it.disabled {
		t.Fatal("unlocked campaign gun 10 is disabled")
	}
	g.campaignGuns[0] = false
	if it := libraryShowWeapon(t, g, 10); !it.disabled {
		t.Fatal("locked campaign gun remained selectable")
	}
	g.selectGunLibraryWeapon(10)
	if g.gunLibrarySelected == 10 {
		t.Fatal("locked campaign gun was selected")
	}
	if it := libraryShowWeapon(t, g, 1); it.disabled {
		t.Fatal("campaign lock array must not lock the starter handgun")
	}
}

func TestGunLibraryDetectsBrokenStatExportAndDistinctDropExport(t *testing.T) {
	hashes := func(dir string, frames []int) map[[32]byte]bool {
		seen := map[[32]byte]bool{}
		for _, frame := range frames {
			path, err := findOriginalPath("sprites", dir, "1", fmt.Sprintf("%d.png", frame))
			if err != nil {
				path = filepath.Join("..", "..", "source", "sprites", dir, "1", fmt.Sprintf("%d.png", frame))
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			seen[sha256.Sum256(data)] = true
		}
		return seen
	}

	// FFDec flattened every stat frame to the same image. Runtime must therefore
	// use Symbol1190 XFL vectors + source text, never these PNGs as truth.
	if got := len(hashes("DefineSprite_1190", []int{7, 10, 20, 30, 40, 50, 60, 66})); got != 1 {
		t.Fatalf("expected known Symbol1190 FFDec flattening, distinct hashes=%d", got)
	}
	// Symbol595 dropgun frames are healthy and can stay as source rasters.
	if got := len(hashes("DefineSprite_595", []int{10, 20, 30, 40, 50, 60, 66})); got < 3 {
		t.Fatalf("Symbol595 unexpectedly flattened, distinct hashes=%d", got)
	}
}

func TestGunLibraryTestPickupRespawnsAfter100SourceTicks(t *testing.T) {
	g := &Game{gototest: true, testGunNumber: 10, testGunDisabled: true, GameMode: SourceGameModeNormal}
	for i := 0; i < 99; i++ {
		g.updateTestGunPickup()
	}
	if !g.testGunDisabled {
		t.Fatal("test pickup respawned before source timer reached 100")
	}
	g.updateTestGunPickup()
	if g.testGunDisabled || g.testGunRespawn != 0 || g.testGunFrame != 10 {
		t.Fatalf("test pickup source respawn mismatch disabled=%v timer=%d frame=%d", g.testGunDisabled, g.testGunRespawn, g.testGunFrame)
	}
}
