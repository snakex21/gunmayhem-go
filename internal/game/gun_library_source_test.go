package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// organizedLibraryButton finds the card for gun on the ALL tab.
func organizedLibraryButton(t *testing.T, gun int) (gunLibraryButton, int) {
	t.Helper()
	for page := 0; page < organizedGunLibraryPageCount(0); page++ {
		for _, b := range organizedGunLibraryButtons(0, page) {
			if b.Gun == gun {
				return b, page
			}
		}
	}
	t.Fatalf("gun %d has no Gun Library card", gun)
	return gunLibraryButton{}, 0
}

func TestGunLibraryRespectsCampaignUnlockArray(t *testing.T) {
	g := &Game{}
	for i := range g.campaignGuns {
		g.campaignGuns[i] = true
	}
	handgun, handgunPage := organizedLibraryButton(t, 1)
	campaignGun, campaignPage := organizedLibraryButton(t, 10)
	hit := func(b gunLibraryButton, page int) int {
		g.gunLibraryPage = page
		return g.organizedGunLibraryHitAt(b.X, b.Y)
	}
	if got := hit(handgun, handgunPage); got != 1 {
		t.Fatalf("source handgun should always be clickable, hit=%d", got)
	}
	if got := hit(campaignGun, campaignPage); got != 10 {
		t.Fatalf("unlocked campaign gun hit=%d want10", got)
	}
	g.campaignGuns[0] = false
	if got := hit(campaignGun, campaignPage); got != 0 {
		t.Fatalf("locked campaign gun remained clickable: %d", got)
	}
	if got := hit(handgun, handgunPage); got != 1 {
		t.Fatalf("campaign lock array must not lock starter handgun, hit=%d", got)
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
