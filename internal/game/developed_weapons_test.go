package game

import (
	"image"
	"testing"
)

func TestAllDevelopedWeaponsHavePackagedSprites(t *testing.T) {
	numbers := allDevelopedWeaponNumbers()
	if len(numbers) != 145 || numbers[0] != 87 || numbers[len(numbers)-1] != 231 {
		t.Fatalf("developed catalogue=%d weapons, range %d..%d; want 145 weapons, 87..231", len(numbers), numbers[0], numbers[len(numbers)-1])
	}
	for _, number := range numbers {
		if !customWeaponAssetAvailable(number) {
			t.Errorf("developed weapon %d (%s) has no packaged 1.png", number, weaponDisplayName(number))
		}
	}
	available := availableCustomWeaponNumbers()
	if len(available) != len(numbers) {
		t.Fatalf("available custom weapons=%d want%d", len(available), len(numbers))
	}
}

func TestCustomWeaponOriginsAnchorAtGrip(t *testing.T) {
	for _, number := range []int{87, 98, 124, 146, 160, 177} {
		raster := loadCustomWeaponRaster(number, 0)
		if raster == nil {
			t.Fatalf("weapon %d raster missing", number)
		}
		if raster.Bounds.X >= 0 || raster.Bounds.Y >= 0 {
			t.Errorf("weapon %d origin is not inside sprite: %+v", number, raster.Bounds)
		}
		if -raster.Bounds.X >= raster.Bounds.W || -raster.Bounds.Y >= raster.Bounds.H {
			t.Errorf("weapon %d origin falls outside sprite: %+v", number, raster.Bounds)
		}
	}
}

func TestDevelopedGrenadeProjectileUsesArcAndAnimatedTime(t *testing.T) {
	b := Bullet{
		DevelopedProjectile: DevelopedProjectileGrenade,
		VX:                  10, VY: -2, Gravity: .58, Lifetime: 20,
		Facing: 1,
	}
	updateDevelopedProjectile(Map{}, &b, nil, nil, nil)
	if b.Time != 1 || b.X != 10 || b.VY <= -2 || b.Y != b.VY {
		t.Fatalf("grenade projectile did not advance along arc: %+v", b)
	}
}

func TestDevelopedFastProjectileCannotSkipPlayer(t *testing.T) {
	target := &Player{ID: 2, Active: true, X: 20, Y: 0, PlayerScale: .8, DamageMulti: 1}
	b := Bullet{
		DevelopedProjectile: DevelopedProjectileEnergy,
		X:                   -30, Y: -20, VX: 100, Firepower: 12, Facing: 1,
		OwnerID: 1, Lifetime: 20,
	}
	updateDevelopedProjectile(Map{}, &b, []*Player{target}, nil, nil)
	if !b.Dead || target.VX <= 0 || target.LastHitBy != 1 {
		t.Fatalf("fast energy projectile skipped target: bullet=%+v targetVX=%.1f lastHit=%d", b, target.VX, target.LastHitBy)
	}
}

func TestBowsHaveUsableAmmoAndLongWeaponPose(t *testing.T) {
	for _, number := range []int{177, 178} {
		def, ok := WeaponByNumber(number)
		if !ok || def.Bullets < 6 {
			t.Fatalf("bow weapon %d ammo=%d want at least 6", number, def.Bullets)
		}
		pose := poseForWeapon(number)
		if number == 177 && pose.HandX < 45 {
			t.Fatalf("crossbow support-hand pose=%+v", pose)
		}
		if number == 178 && pose.HandX >= 0 {
			t.Fatalf("compound-bow draw-hand pose=%+v want hand behind riser", pose)
		}
	}
}

func TestTubeLauncherRasterDoesNotAddMenuRocketOverlay(t *testing.T) {
	base := decodeOriginalPNGIn("custom", "weapons", "javelin", "1.png")
	loaded := loadCustomWeaponRaster(89, 0)
	if base == nil || loaded == nil {
		t.Fatal("Javelin weapon assets missing")
	}
	if loaded.Image.Bounds().Dx() != base.Bounds().Dx() {
		t.Fatalf("Javelin width=%d base=%d; raster must not append a fake rocket", loaded.Image.Bounds().Dx(), base.Bounds().Dx())
	}
}

func TestDevelopedPistolsAreScaledForPlayerHands(t *testing.T) {
	base := decodeOriginalPNGIn("custom", "weapons", "glock18c", "1.png")
	raster := loadCustomWeaponRaster(99, 0)
	if base == nil || raster == nil {
		t.Fatal("Glock 18C assets missing")
	}
	if raster.Image.Bounds().Dx() >= base.Bounds().Dx() || raster.Image.Bounds().Dx() > 40 {
		t.Fatalf("developed pistol width=%d original=%d; pistol was not normalized", raster.Image.Bounds().Dx(), base.Bounds().Dx())
	}
}

func TestLateDevelopedSidearmsUsePlayerScale(t *testing.T) {
	limits := map[int]image.Point{
		190: {64, 34}, // Lancaster Howdah
		206: {55, 36}, // Colt Model 1903 Hammerless
		207: {60, 32}, // Frommer Stop
		216: {60, 34}, // Nambu Type 14
		219: {56, 36}, // AF2011-A1
		230: {80, 40}, // Annihilator is a compact SMG, not a pistol
		231: {66, 28}, // Peacekeeper has a long but thin barrel
	}
	for number, limit := range limits {
		raster := loadCustomWeaponRaster(number, 0)
		if raster == nil {
			t.Fatalf("weapon %d raster missing", number)
		}
		got := raster.Image.Bounds().Size()
		if got.X > limit.X || got.Y > limit.Y {
			t.Errorf("weapon %d remains oversized: got %dx%d, limit %dx%d", number, got.X, got.Y, limit.X, limit.Y)
		}
		if got.X < 35 || got.Y < 15 {
			t.Errorf("weapon %d was reduced too far: %dx%d", number, got.X, got.Y)
		}
	}
}

func TestLateDevelopedShoulderWeaponsUseSourceGameScale(t *testing.T) {
	for number := 191; number <= 230; number++ {
		spec, ok := developedWeaponSpec(number)
		if !ok || spec.Category == DevelopedCategoryPistol {
			continue
		}
		raster := loadCustomWeaponRaster(number, 0)
		if raster == nil {
			t.Fatalf("weapon %d raster missing", number)
		}
		got := raster.Image.Bounds().Size()
		maxW, maxH := 120, 50
		switch {
		case number == 192: // deliberately oversized anti-tank rifle
			maxW, maxH = 125, 35
		case spec.Category == DevelopedCategoryLMG:
			maxW, maxH = 115, 60
		}
		if got.X > maxW || got.Y > maxH {
			t.Errorf("late weapon %d remains oversized: got %dx%d, limit %dx%d", number, got.X, got.Y, maxW, maxH)
		}
	}
}

func TestLateDevelopedWeaponMechanicsMatchTheirModels(t *testing.T) {
	wantSpecial := map[int]struct {
		category DevelopedWeaponCategory
		mode     DevelopedFireMode
		ammo     int
		pellets  int
	}{
		192: {DevelopedCategorySniper, DevelopedFireSemi, 1, 0},
		193: {DevelopedCategorySniper, DevelopedFireSemi, 1, 0},
		198: {DevelopedCategoryLMG, DevelopedFireBurst2, 50, 0},
		204: {DevelopedCategoryShotgun, DevelopedFireSemi, 5, 7},
		213: {DevelopedCategoryRifle, DevelopedFireBurst3, 45, 0},
		219: {DevelopedCategoryPistol, DevelopedFireBurst2, 16, 0},
		229: {DevelopedCategoryShotgun, DevelopedFireSemi, 2, 8},
	}
	for number, want := range wantSpecial {
		def, ok := WeaponByNumber(number)
		if !ok {
			t.Fatalf("late weapon %d definition missing", number)
		}
		spec, ok := developedWeaponSpec(number)
		if !ok {
			t.Fatalf("late weapon %d runtime spec missing", number)
		}
		if spec.Category != want.category || spec.FireMode != want.mode || def.Bullets != want.ammo || def.Shotgun != want.pellets {
			t.Errorf("late weapon %d mismatch: category=%d mode=%d ammo=%d pellets=%d", number, spec.Category, spec.FireMode, def.Bullets, def.Shotgun)
		}
	}

	for number := 190; number <= 231; number++ {
		def, ok := WeaponByNumber(number)
		if !ok {
			t.Fatalf("late weapon %d definition missing", number)
		}
		if number != 204 && number != 229 && def.Shotgun != 0 {
			t.Errorf("non-shotgun weapon %d incorrectly emits %d pellets", number, def.Shotgun)
		}
		if sourceGunSound(number) == "" {
			t.Errorf("late weapon %d has no firing sound", number)
		}
	}
}

func TestLancasterHowdahFiresOneHeavyRoundPerChamber(t *testing.T) {
	def, ok := WeaponByNumber(190)
	if !ok {
		t.Fatal("Lancaster Howdah definition missing")
	}
	spec, ok := developedWeaponSpec(190)
	if !ok {
		t.Fatal("Lancaster Howdah runtime spec missing")
	}
	if def.Shotgun != 0 || def.Bullets != 4 || def.Firepower < 25 {
		t.Fatalf("Howdah definition=%+v; want four separate heavy rounds", def)
	}
	if spec.FireMode != DevelopedFireSemi || spec.Spread > 3 || spec.Lifetime < 70 {
		t.Fatalf("Howdah projectile spec=%+v; want controlled single-projectile fire", spec)
	}

	p := &Player{ID: 1, Active: true, AI: true, AIShoot: true, Facing: 1, MiniMulti: 1, PlayerScale: .8}
	p.Weapon = NewWeapon(190)
	p.Weapon.Frame = 1
	p.Weapon.WaitTime = p.Weapon.Def.ROF
	var bullets []Bullet
	var shells []Shell
	var flashes []MuzzleFlash
	updateWeapon(p, &bullets, &shells, &flashes)
	if len(bullets) != 1 || bullets[0].Kind == BulletShotgun {
		t.Fatalf("Howdah fired %+v; want one normal projectile", bullets)
	}
	if p.Weapon.Bullets != def.Bullets-1 {
		t.Fatalf("Howdah ammo=%d want%d after one chamber", p.Weapon.Bullets, def.Bullets-1)
	}
}

func TestDevelopedPistolsAnchorOnTheirIndividualGrips(t *testing.T) {
	pistols := []int{98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 190, 206, 207, 216, 219, 231}
	for _, number := range pistols {
		origin, ok := customWeaponGripOrigins[number]
		if !ok {
			t.Fatalf("developed pistol %d has no individual grip origin", number)
		}
		if origin[0] < .15 || origin[0] > .31 || origin[1] < .45 || origin[1] > .68 {
			t.Errorf("developed pistol %d grip origin=%v outside grip region", number, origin)
		}
	}
	if _, ok := customWeaponGripOrigins[1]; ok {
		t.Fatal("base pistol received a custom generated-sprite origin")
	}
}

func TestRequestedHistoricalRiflesAnchorAtTheirTriggers(t *testing.T) {
	for _, number := range []int{199, 200, 204, 208, 209, 210, 211, 218, 224, 225, 226} {
		origin, ok := customWeaponGripOrigins[number]
		if !ok {
			t.Fatalf("weapon %d has no corrected grip origin", number)
		}
		if origin[0] < .28 || origin[0] > .36 || origin[1] < .54 || origin[1] > .70 {
			t.Errorf("weapon %d trigger origin=%v outside rear trigger region", number, origin)
		}
	}
	if g11Origin := customWeaponGripOrigins[213]; g11Origin[0] < .45 || g11Origin[0] > .53 || g11Origin[1] < .62 || g11Origin[1] > .72 {
		t.Fatalf("G11 trigger origin=%v does not land on its bullpup pistol grip", g11Origin)
	}

	g11Base := decodeOriginalPNGIn("custom", "weapons", "hk_g11", "1.png")
	g11 := loadCustomWeaponRaster(213, 0)
	if g11Base == nil || g11 == nil || g11.Image.Bounds().Dx() >= g11Base.Bounds().Dx() {
		t.Fatalf("G11 was not reduced to player scale: base=%v raster=%v", g11Base, g11)
	}
	if pose := poseForWeapon(213); pose.HandX-pose.ShootX < 35 {
		t.Fatalf("G11 support hand is not placed on forebody: %+v", pose)
	}

	himarsOrigin := customWeaponGripOrigins[90]
	himarsPose := poseForWeapon(90)
	if himarsOrigin[0] > .45 || himarsPose.HandX-himarsPose.ShootX > 35 {
		t.Fatalf("HIMARS grips still land on the muzzle: origin=%v pose=%+v", himarsOrigin, himarsPose)
	}
}

func TestJavelinAcquiresTargetAndExplodes(t *testing.T) {
	target := &Player{ID: 2, Active: true, X: 270, Y: 250, PlayerScale: .8, DamageMulti: 1}
	b := Bullet{DevelopedProjectile: DevelopedProjectileTopAttack, X: 100, Y: 210, VX: 9.8, VY: -9.1, Facing: 1, OwnerID: 1, Lifetime: 180, ExplosionRadius: 225, ExplosionPower: 98}
	var explosions []Explosion
	for i := 0; i < 80 && !b.Dead; i++ {
		updateDevelopedProjectile(Map{}, &b, []*Player{target}, &explosions, nil)
	}
	if !b.Dead || len(explosions) == 0 || target.LastHitBy != 1 {
		t.Fatalf("guided Javelin failed: bullet=%+v explosions=%d targetLastHit=%d", b, len(explosions), target.LastHitBy)
	}
}

func TestDevelopedShotgunActuallyFiresAllPellets(t *testing.T) {
	p := &Player{ID: 1, Active: true, AI: true, AIShoot: true, Facing: 1, MiniMulti: 1, PlayerScale: .8}
	p.Weapon = NewWeapon(162) // Benelli M4: seven pellets
	p.Weapon.Frame = 1
	p.Weapon.WaitTime = p.Weapon.Def.ROF
	var bullets []Bullet
	var shells []Shell
	var flashes []MuzzleFlash
	updateWeapon(p, &bullets, &shells, &flashes)
	if len(bullets) != p.Weapon.Def.Shotgun {
		t.Fatalf("custom shotgun fired %d projectiles want%d", len(bullets), p.Weapon.Def.Shotgun)
	}
	for _, bullet := range bullets {
		if bullet.Kind != BulletShotgun {
			t.Fatalf("custom shotgun emitted kind=%d want%d", bullet.Kind, BulletShotgun)
		}
	}
}

func TestRapidSpecialToolsUseAutomaticFire(t *testing.T) {
	for _, number := range []int{180, 185} {
		spec, ok := developedWeaponSpec(number)
		if !ok || spec.FireMode != DevelopedFireAuto {
			t.Fatalf("rapid special weapon %d fire mode=%d want auto", number, spec.FireMode)
		}
	}
}

func TestDevelopedSemiAutoRepeatsWhileFireIsHeld(t *testing.T) {
	p := &Player{ID: 1, Active: true, AI: true, AIShoot: true, Facing: 1, MiniMulti: 1, PlayerScale: .8}
	p.Weapon = NewWeapon(98) // FN Five-seveN, developed semi-auto pistol
	p.Weapon.Frame = 1
	p.Weapon.WaitTime = p.Weapon.Def.ROF
	var bullets []Bullet
	var shells []Shell
	var flashes []MuzzleFlash

	updateWeapon(p, &bullets, &shells, &flashes)
	for i := 0; i < p.Weapon.Def.ROF; i++ {
		updateWeapon(p, &bullets, &shells, &flashes)
	}
	if len(bullets) != 2 {
		t.Fatalf("held developed semi-auto fired %d bullets want2", len(bullets))
	}
}

func TestDevelopedIdleAnglesStaySubtle(t *testing.T) {
	for _, number := range allDevelopedWeaponNumbers() {
		angle := sourceWeaponAnim(number).IdleRotate
		if angle < -12 || angle > 12 {
			t.Errorf("developed weapon %d idle angle=%.1f is too steep", number, angle)
		}
	}
	if got := sourceWeaponAnim(1); got != weaponAnimCatalog[1] {
		t.Fatalf("base weapon animation changed: got %+v want %+v", got, weaponAnimCatalog[1])
	}
}

func TestDevelopedPistolSpritesFaceRight(t *testing.T) {
	for number, spec := range developedWeaponSpecs {
		if spec.Category != DevelopedCategoryPistol {
			continue
		}
		img := decodeOriginalPNGIn("custom", "weapons", spec.AssetDir, "1.png")
		if img == nil {
			t.Fatalf("pistol %d sprite missing", number)
		}
		bounds := img.Bounds()
		minY, maxY := bounds.Max.Y, bounds.Min.Y
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, alpha := img.At(x, y).RGBA()
				if alpha > 0x5000 {
					if y < minY {
						minY = y
					}
					if y > maxY {
						maxY = y
					}
				}
			}
		}
		splitY := minY + int(float64(maxY-minY)*.48)
		var upperX, lowerX, upperCount, lowerCount int
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, alpha := img.At(x, y).RGBA()
				if alpha <= 0x5000 {
					continue
				}
				if y <= splitY {
					upperX += x
					upperCount++
				} else {
					lowerX += x
					lowerCount++
				}
			}
		}
		if upperCount == 0 || lowerCount == 0 || float64(lowerX)/float64(lowerCount) >= float64(upperX)/float64(upperCount) {
			t.Errorf("pistol %d (%s) does not point right: upper center=%.1f lower center=%.1f", number, spec.AssetDir, float64(upperX)/float64(upperCount), float64(lowerX)/float64(lowerCount))
		}
	}
}

func TestStarterHandgunRosterIncludesAllDevelopedPistols(t *testing.T) {
	got := starterHandgunNumbers()
	if len(got) != 27 {
		t.Fatalf("starter handgun count=%d want27", len(got))
	}
	for i, number := range []int{1, 2, 3, 4, 5, 6} {
		if got[i] != number {
			t.Fatalf("base starter index%d=%d want%d", i, got[i], number)
		}
	}
	developed := []int{98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 190, 206, 207, 216, 219, 231}
	for i, number := range got[6:] {
		want := developed[i]
		if number != want {
			t.Fatalf("developed starter index%d=%d want%d", i, number, want)
		}
		spec, ok := developedWeaponSpec(number)
		if !ok || spec.Category != DevelopedCategoryPistol {
			t.Fatalf("starter %d is not a developed pistol", number)
		}
	}
}

func TestRequestedWWIExpansionHasDistinctCompleteDefinitions(t *testing.T) {
	for number := 190; number <= 231; number++ {
		def, ok := WeaponByNumber(number)
		if !ok || def.Name == "" || def.Bullets <= 0 {
			t.Fatalf("requested weapon %d has incomplete definition: %+v ok=%v", number, def, ok)
		}
		if !customWeaponAssetAvailable(number) {
			t.Fatalf("requested weapon %d (%s) has no sprite", number, def.Name)
		}
	}
	if developedWeaponCatalog[220].Name != "Huot Automatic Rifle" {
		t.Fatalf("BF1 omission slot 220=%q want Huot Automatic Rifle", developedWeaponCatalog[220].Name)
	}
	_, m249OK := WeaponByNumber(65)
	_, auto5OK := WeaponByNumber(86)
	if !m249OK || !auto5OK || weaponDisplayName(65) != "FN M249 SAW" || weaponDisplayName(86) != "Browning Auto-5" {
		t.Fatal("existing source weapons were duplicated or replaced")
	}
}

func TestGlock18CReloadsAfterEmptyMagazine(t *testing.T) {
	p := &Player{ID: 1, Active: true, AI: true, AIShoot: true, Facing: 1, MiniMulti: 1, PlayerScale: .8, DefaultWeapon: 99}
	p.Weapon = NewWeapon(99)
	p.Weapon.Bullets = 1
	p.Weapon.Frame = 1
	p.Weapon.WaitTime = p.Weapon.Def.ROF
	var bullets []Bullet
	var shells []Shell
	var flashes []MuzzleFlash
	updateWeapon(p, &bullets, &shells, &flashes)
	if p.Weapon.Def.Number != 99 || p.Weapon.Bullets != 0 || !p.Weapon.Playing {
		t.Fatalf("empty starter pistol did not begin reload: weapon=%d bullets=%d playing=%v", p.Weapon.Def.Number, p.Weapon.Bullets, p.Weapon.Playing)
	}
	for i := 0; i < 40 && p.Weapon.Playing; i++ {
		updateWeapon(p, &bullets, &shells, &flashes)
	}
	if p.Weapon.Def.Number != 99 || p.Weapon.Bullets != p.Weapon.Def.Bullets || p.Weapon.Playing {
		t.Fatalf("starter pistol did not finish reload: weapon=%d bullets=%d/%d playing=%v", p.Weapon.Def.Number, p.Weapon.Bullets, p.Weapon.Def.Bullets, p.Weapon.Playing)
	}
	seen := map[string]bool{}
	for _, name := range p.PendingSounds {
		seen[name] = true
	}
	if !seen["pistol_mag.wav"] || !seen["pistol_slide.wav"] {
		t.Fatalf("developed reload sounds=%v", p.PendingSounds)
	}
	if len(bullets) != 1 {
		t.Fatalf("Glock fired during reload: bullets=%d want1", len(bullets))
	}
}

func TestHarpoonSpriteUsesDedicatedScale(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 120, 54))
	got := normalizeCustomWeaponScale(img, 179, DevelopedCategorySpecial)
	if got.Bounds().Dx() != 79 || got.Bounds().Dy() != 36 {
		t.Fatalf("harpoon scaled to %dx%d want79x36", got.Bounds().Dx(), got.Bounds().Dy())
	}
}

func TestFlareIgnitesAndBurnKnockbackVaries(t *testing.T) {
	target := &Player{ID: 2, Active: true, X: 20, Y: 0, PlayerScale: .8, DamageMulti: 1}
	b := Bullet{
		DevelopedProjectile: DevelopedProjectileFlare,
		X:                   20, Y: -20, VX: 1, Firepower: 10, Facing: 1,
		OwnerID: 1, Lifetime: 20, IgniteFrames: 105,
	}
	updateDevelopedProjectile(Map{}, &b, []*Player{target}, nil, nil)
	if !b.Dead || target.BurnTime != 105 || target.BurnOwnerID != 1 || target.BurnPower != 10 {
		t.Fatalf("flare did not ignite target: bulletDead=%v burn=%d power=%.1f owner=%d", b.Dead, target.BurnTime, target.BurnPower, target.BurnOwnerID)
	}
	startVX, startVY := target.VX, target.VY
	for i := 0; i < 15; i++ {
		target.updateBurning()
	}
	firstPulseVX := target.VX
	for i := 0; i < 15; i++ {
		target.updateBurning()
	}
	if firstPulseVX == startVX || target.VX == firstPulseVX || target.VY >= startVY || target.LastHitBy != 1 {
		t.Fatalf("burn did not apply varied knockback: vx %.2f->%.2f->%.2f vy %.2f->%.2f lastHit=%d", startVX, firstPulseVX, target.VX, startVY, target.VY, target.LastHitBy)
	}
}

func TestFlamethrowerBurnKeepsPushingAfterStreamStops(t *testing.T) {
	target := &Player{ID: 2, Active: true, DamageMulti: 1}
	target.igniteWithPower(1, 1, 60, 4.2)
	// Remove the direct-hit impulse. Everything observed below must come from
	// the lingering fire status, with no additional projectile or ignition.
	target.VX, target.VY = 0, 0
	for i := 0; i < 20; i++ {
		target.updateBurning()
	}
	if target.BurnTime != 40 || target.VX < 4 || target.VY > -2 || target.LastHitBy != 1 || target.HitNumber < 2 {
		t.Fatalf("lingering flamethrower burn did not push: burn=%d vx=%.2f vy=%.2f hits=%d owner=%d", target.BurnTime, target.VX, target.VY, target.HitNumber, target.LastHitBy)
	}
}

func TestFlamethrowersUseSustainedBalancedStreams(t *testing.T) {
	m2, ok := developedWeaponSpec(187)
	m2Def := developedWeaponCatalog[187]
	if !ok || m2.IgniteFrames != 100 || m2Def.Firepower != 1.45 || m2Def.Bullets != 220 || m2.Lifetime != 20 {
		t.Fatalf("M2 balance=%+v def=%+v ok=%v", m2, m2Def, ok)
	}
	fw35, ok := developedWeaponSpec(188)
	fw35Def := developedWeaponCatalog[188]
	if !ok || fw35.IgniteFrames != 115 || fw35Def.Firepower != 1.8 || fw35Def.Bullets != 190 || fw35.Lifetime != 20 {
		t.Fatalf("Flammenwerfer balance=%+v def=%+v ok=%v", fw35, fw35Def, ok)
	}
}

func TestShieldPreventsFlareIgnition(t *testing.T) {
	target := &Player{ID: 2, Active: true, X: 20, Y: 0, PlayerScale: .8, DamageMulti: 1, ShieldTime: 30}
	b := Bullet{DevelopedProjectile: DevelopedProjectileFlare, X: 20, Y: -20, VX: 1, Firepower: 10, Facing: 1, OwnerID: 1, Lifetime: 20, IgniteFrames: 105}
	updateDevelopedProjectile(Map{}, &b, []*Player{target}, nil, nil)
	if target.BurnTime != 0 {
		t.Fatalf("shielded target burn=%d want0", target.BurnTime)
	}
}

func TestDevelopedOneShotReloadWeaponsDoNotAutoReequip(t *testing.T) {
	oneShotCount := 0
	for number, def := range developedWeaponCatalog {
		if def.Bullets != 1 || number == 192 {
			continue
		}
		oneShotCount++
		reloadFrames := developedManualReloadFrames(number)
		if reloadFrames < 75 || reloadFrames > 210 {
			t.Fatalf("one-shot reload weapon %d (%s) reload=%d outside supported range", number, def.Name, reloadFrames)
		}

		p := &Player{ID: 3, Active: true, AI: true, AIShoot: true, Facing: 1, MiniMulti: 1, PlayerScale: .8, DefaultWeapon: number}
		p.Weapon = NewWeapon(number)
		p.Weapon.Frame = 1
		p.Weapon.WaitTime = p.Weapon.Def.ROF
		var bullets []Bullet
		var shells []Shell
		var flashes []MuzzleFlash

		updateWeapon(p, &bullets, &shells, &flashes)
		if len(bullets) == 0 || p.Weapon.Def.Number != number || p.Weapon.Bullets != 0 || p.Weapon.WaitTime != -reloadFrames {
			t.Fatalf("one-shot weapon %d (%s) first shot/reload state wrong: projectiles=%d weapon=%d ammo=%d wait=%d want wait=%d", number, def.Name, len(bullets), p.Weapon.Def.Number, p.Weapon.Bullets, p.Weapon.WaitTime, -reloadFrames)
		}
		projectilesAfterShot := len(bullets)
		for i := 0; i < reloadFrames; i++ {
			updateWeapon(p, &bullets, &shells, &flashes)
			if len(bullets) != projectilesAfterShot {
				t.Fatalf("one-shot weapon %d (%s) fired during reload at frame %d", number, def.Name, i+1)
			}
		}
		if p.Weapon.Bullets != 1 || p.Weapon.WaitTime != p.Weapon.Def.ROF {
			t.Fatalf("one-shot weapon %d (%s) did not finish reload: ammo=%d wait=%d", number, def.Name, p.Weapon.Bullets, p.Weapon.WaitTime)
		}
		updateWeapon(p, &bullets, &shells, &flashes)
		if len(bullets) <= projectilesAfterShot {
			t.Fatalf("one-shot weapon %d (%s) did not fire after reload", number, def.Name)
		}
	}
	if oneShotCount < 9 {
		t.Fatalf("developed one-shot reload audit found only %d weapons", oneShotCount)
	}
}

func TestTankgewehrDropsAfterItsSingleShot(t *testing.T) {
	const tankgewehr = 192
	if reload := developedManualReloadFrames(tankgewehr); reload != 0 {
		t.Fatalf("Tankgewehr reload=%d want0 so the empty pickup is discarded", reload)
	}
	p := &Player{ID: 1, Active: true, AI: true, AIShoot: true, Facing: 1, MiniMulti: 1, PlayerScale: .8, DefaultWeapon: 1}
	p.Weapon = NewWeapon(tankgewehr)
	p.Weapon.Frame = 1
	p.Weapon.WaitTime = p.Weapon.Def.ROF
	var bullets []Bullet
	var shells []Shell
	var flashes []MuzzleFlash
	updateWeapon(p, &bullets, &shells, &flashes)
	if len(bullets) == 0 {
		t.Fatal("Tankgewehr did not fire its single round")
	}
	if p.Weapon.Def.Number != p.DefaultWeapon {
		t.Fatalf("Tankgewehr stayed/reloaded after empty shot: weapon=%d want default=%d", p.Weapon.Def.Number, p.DefaultWeapon)
	}
}

func TestMusketShotSpawnsLingeringSmoke(t *testing.T) {
	for _, number := range []int{182, 193} {
		p := &Player{ID: 1, Active: true, AI: true, AIShoot: true, Facing: 1, MiniMulti: 1, PlayerScale: .8}
		p.Weapon = NewWeapon(number)
		p.Weapon.Frame = 1
		p.Weapon.WaitTime = p.Weapon.Def.ROF
		var bullets []Bullet
		var shells []Shell
		var flashes []MuzzleFlash
		updateWeapon(p, &bullets, &shells, &flashes)
		if len(flashes) != 2 || !flashes[1].Smoke {
			t.Fatalf("black-powder weapon %d flashes=%+v want muzzle flash and smoke", number, flashes)
		}
		before := flashes[1].Alpha
		updateFlashes(flashes)
		updateFlashes(flashes)
		if flashes[1].Life == 0 || flashes[1].Alpha >= before || flashes[1].Scale <= 1 {
			t.Fatalf("black-powder smoke %d did not expand/fade: %+v", number, flashes[1])
		}
	}
}

func TestHeavyWeaponBalanceCorrections(t *testing.T) {
	mg42, _ := WeaponByNumber(141)
	ks23, _ := WeaponByNumber(165)
	harpoon, _ := WeaponByNumber(179)
	if mg42.Recoil > .4 {
		t.Fatalf("MG42 recoil=%.2f remains excessive", mg42.Recoil)
	}
	if ks23.Firepower < 16 || ks23.Shotgun < 7 {
		t.Fatalf("KS-23 remains underpowered: %+v", ks23)
	}
	if harpoon.Bullets != 1 || harpoon.Firepower < 75 {
		t.Fatalf("harpoon is not a powerful single-shot: %+v", harpoon)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 110, 50))
	got := normalizeCustomWeaponScale(img, 183, DevelopedCategoryShotgun)
	if got.Bounds().Dx() >= img.Bounds().Dx() || got.Bounds().Dy() >= img.Bounds().Dy() {
		t.Fatalf("single-shot sawed-off was not reduced: got=%v base=%v", got.Bounds(), img.Bounds())
	}
}
