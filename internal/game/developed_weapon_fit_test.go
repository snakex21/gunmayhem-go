package game

import (
	"image"
	"math"
	"testing"
)

// opaqueBoundsOf returns the bounding box of the clearly opaque pixels of a
// normalized weapon sprite, in sprite pixels.
func opaqueBoundsOf(img image.Image) (minX, minY, maxX, maxY int, ok bool) {
	b := img.Bounds()
	minX, minY, maxX, maxY = b.Dx(), b.Dy(), -1, -1
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if _, _, _, a := img.At(b.Min.X+x, b.Min.Y+y).RGBA(); a>>8 < 110 {
				continue
			}
			minX, minY = min(minX, x), min(minY, y)
			maxX, maxY = max(maxX, x), max(maxY, y)
		}
	}
	return minX, minY, maxX, maxY, maxX >= 0
}

func nearestOpaqueDistance(img image.Image, px, py float64) float64 {
	b := img.Bounds()
	best := math.Inf(1)
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if _, _, _, a := img.At(b.Min.X+x, b.Min.Y+y).RGBA(); a>>8 < 110 {
				continue
			}
			best = math.Min(best, math.Hypot(float64(x)+.5-px, float64(y)+.5-py))
		}
	}
	return best
}

// Every developed weapon must carry an explicit, measured grip/hand/muzzle
// fit; category fallbacks put the hands in empty space on most sprites.
func TestDevelopedWeaponsHaveExplicitGripAndMuzzleFit(t *testing.T) {
	numbers := allDevelopedWeaponNumbers()
	if len(numbers) != 147 {
		t.Fatalf("developed weapons=%d want147", len(numbers))
	}
	for _, number := range numbers {
		origin, okOrigin := customWeaponGripOrigins[number]
		pose, okPose := developedWeaponPoses[number]
		muzzle, okMuzzle := developedWeaponMuzzles[number]
		if !okOrigin || !okPose || !okMuzzle {
			t.Errorf("weapon %d fit missing: origin=%v pose=%v muzzle=%v", number, okOrigin, okPose, okMuzzle)
			continue
		}
		if origin[0] < .05 || origin[0] > .7 || origin[1] < .3 || origin[1] > .85 {
			t.Errorf("weapon %d grip origin=%v outside the sprite's grip region", number, origin)
		}
		if got := poseForWeapon(number); got != pose {
			t.Errorf("weapon %d pose=%+v want table pose %+v", number, got, pose)
		}
		// Source poses span ShootX -10..30, ShootY -7..3, HandX 28..60 and
		// HandY -22..-4; developed weapons stay in the same reach.
		if pose.ShootX < -10 || pose.ShootX > 30 || pose.ShootY < -7 || pose.ShootY > 8 {
			t.Errorf("weapon %d grip-hand pose=%+v outside source reach", number, pose)
		}
		if pose.HandX < -25 || pose.HandX > 62 || pose.HandY < -26 || pose.HandY > 0 {
			t.Errorf("weapon %d support-hand pose=%+v outside source reach", number, pose)
		}
		def, _ := WeaponByNumber(number)
		if def.FlashX != muzzle[0] {
			t.Errorf("weapon %d Def.FlashX=%.1f want measured %.1f", number, def.FlashX, muzzle[0])
		}
		if muzzle[1] < -41 || muzzle[1] > -35 || developedFlashY(number) != muzzle[1] {
			t.Errorf("weapon %d flash height=%.1f is not on the bullet line (-38 +/- 3)", number, muzzle[1])
		}
	}
	if developedFlashY(1) != -38 {
		t.Fatal("source weapons must keep the FIREBULLET flash height")
	}
}

// The grip pivot must sit on the gun art, the flash on the drawn muzzle.
func TestDevelopedWeaponFitMatchesSpriteAlpha(t *testing.T) {
	for _, number := range allDevelopedWeaponNumbers() {
		spec, _ := developedWeaponSpec(number)
		if proceduralWeapon(number) {
			continue // drawn in code; covered by TestArsenal2ProceduralArtFit
		}
		img := decodeOriginalPNGIn("custom", "weapons", spec.AssetDir, "1.png")
		if img == nil {
			t.Fatalf("weapon %d sprite missing", number)
		}
		img = normalizeCustomWeaponScale(img, number, spec.Category)
		minX, minY, maxX, maxY, ok := opaqueBoundsOf(img)
		if !ok {
			t.Fatalf("weapon %d sprite is empty", number)
		}
		w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
		ox, oy := customWeaponOrigin(number, spec.Category)
		gx, gy := w*ox, h*oy
		if d := nearestOpaqueDistance(img, gx, gy); d > 3 {
			t.Errorf("weapon %d grip (%.0f,%.0f) floats %.1f px off the sprite", number, gx, gy, d)
		}
		pose := poseForWeapon(number)
		muzzle := developedWeaponMuzzles[number]
		// World x of the sprite's opaque extent at player scale 0.8.
		left := .8 * (pose.ShootX + float64(minX) - gx)
		tip := .8 * (pose.ShootX + float64(maxX+1) - gx)
		if muzzle[0] < left-3 || muzzle[0] > tip+3 {
			t.Errorf("weapon %d FlashX=%.1f outside the sprite (%.1f..%.1f)", number, muzzle[0], left, tip)
		}
		if math.Abs(muzzle[0]-tip) > 6 {
			t.Errorf("weapon %d FlashX=%.1f is %.1f px from the drawn muzzle tip %.1f", number, muzzle[0], muzzle[0]-tip, tip)
		}
		top := .8 * (-35 + pose.ShootY + float64(minY) - gy)
		bottom := .8 * (-35 + pose.ShootY + float64(maxY+1) - gy)
		if muzzle[1] < top || muzzle[1] > bottom {
			t.Errorf("weapon %d FlashY=%.1f outside the sprite (%.1f..%.1f)", number, muzzle[1], top, bottom)
		}
	}
}
