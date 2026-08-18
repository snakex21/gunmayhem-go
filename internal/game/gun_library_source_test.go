package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
)

func TestGunLibraryRecoversAllSourceGunButtons(t *testing.T) {
	buttons := sourceGunLibraryButtons()
	if len(buttons) != 63 {
		t.Fatalf("Symbol1198 gun buttons=%d want63 (handguns 1..6 + guns 10..66)", len(buttons))
	}
	seen := map[int]bool{}
	for _, b := range buttons {
		if !isGunLibraryGunNumber(b.Gun) {
			t.Fatalf("invalid source gun number %d", b.Gun)
		}
		seen[b.Gun] = true
	}
	for gun := 1; gun <= 6; gun++ {
		if !seen[gun] {
			t.Fatalf("source handgun button %d missing", gun)
		}
	}
	for gun := 10; gun <= 66; gun++ {
		if !seen[gun] {
			t.Fatalf("source gun button %d missing", gun)
		}
	}
}

func TestGunLibraryRespectsCampaignUnlockArray(t *testing.T) {
	g := &Game{}
	for i := range g.campaignGuns {
		g.campaignGuns[i] = true
	}
	var handgun, campaignGun gunLibraryButton
	for _, b := range sourceGunLibraryButtons() {
		if b.Gun == 1 {
			handgun = b
		}
		if b.Gun == 10 {
			campaignGun = b
		}
	}
	if got := g.gunLibraryHitAt(handgun.X, handgun.Y); got != 1 {
		t.Fatalf("source handgun should always be clickable, hit=%d", got)
	}
	if got := g.gunLibraryHitAt(campaignGun.X, campaignGun.Y); got != 10 {
		t.Fatalf("unlocked campaign gun hit=%d want10", got)
	}
	g.campaignGuns[0] = false
	if got := g.gunLibraryHitAt(campaignGun.X, campaignGun.Y); got != 0 {
		t.Fatalf("locked campaign gun remained clickable: %d", got)
	}
	if got := g.gunLibraryHitAt(handgun.X, handgun.Y); got != 1 {
		t.Fatalf("campaign lock array must not lock starter handgun, hit=%d", got)
	}
}

func TestGunLibraryDetectsBrokenStatExportAndDistinctDropExport(t *testing.T) {
	hashes := func(dir string, frames []int) map[[32]byte]bool {
		seen := map[[32]byte]bool{}
		for _, frame := range frames {
			path, err := findOriginalPath("sprites", dir, "1", fmt.Sprintf("%d.png", frame))
			if err != nil {
				t.Fatalf("%s frame%d: %v", dir, frame, err)
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
	if got := sourceGunLibraryName(7); got != "Nothing Selected" {
		t.Fatalf("frame7 name=%q want Nothing Selected", got)
	}
	if sourceGunLibraryName(10) == sourceGunLibraryName(20) {
		t.Fatalf("source stat names for guns 10 and 20 collapsed to %q", sourceGunLibraryName(10))
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
