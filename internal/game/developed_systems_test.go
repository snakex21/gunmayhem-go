package game

import "testing"

func TestPortalGunAlternatesPortalColours(t *testing.T) {
	p := &Player{ID: 1, Active: true, AI: true, AIShoot: true, Facing: 1, Team: 1, MiniMulti: 1, PlayerScale: .8, PortalNextColor: PortalBlue}
	p.Weapon = NewWeapon(189)
	p.Weapon.Frame = 1
	p.Weapon.WaitTime = p.Weapon.Def.ROF
	var bullets []Bullet
	var shells []Shell
	var flashes []MuzzleFlash
	updateWeapon(p, &bullets, &shells, &flashes)
	p.Weapon.WaitTime = p.Weapon.Def.ROF
	updateWeapon(p, &bullets, &shells, &flashes)
	if len(bullets) != 2 || bullets[0].PortalColor != PortalBlue || bullets[1].PortalColor != PortalOrange {
		t.Fatalf("portal shots=%+v", bullets)
	}
}

func TestGeneratedSpecialSpritesNormalizeToPlayerScale(t *testing.T) {
	for _, number := range []int{187, 188, 189} {
		raster := loadCustomWeaponRaster(number, 0)
		if raster == nil {
			t.Fatalf("weapon %d sprite missing", number)
		}
		bounds := raster.Image.Bounds()
		if bounds.Dx() > 148 || bounds.Dy() > 78 {
			t.Fatalf("weapon %d normalized to %dx%d", number, bounds.Dx(), bounds.Dy())
		}
		if raster.Bounds.X >= 0 || raster.Bounds.Y >= 0 || -raster.Bounds.X >= raster.Bounds.W || -raster.Bounds.Y >= raster.Bounds.H {
			t.Fatalf("weapon %d grip falls outside sprite: %+v", number, raster.Bounds)
		}
	}
}

func TestPortalPairCatchesPlayerAfterPlatformStopsVelocity(t *testing.T) {
	portals := []Portal{
		{X: 100, Y: 72, NormalX: 1, OwnerID: 1, Color: PortalBlue, Active: true},
		{X: 300, Y: 72, NormalX: -1, OwnerID: 1, Color: PortalOrange, Active: true},
	}
	p := &Player{ID: 2, Active: true, X: 100, Y: 100, PlayerScale: .8}
	updatePortalTravel(portals, []*Player{p}, nil)
	if p.X >= 300 || p.VX >= 0 || p.PortalCooldown == 0 {
		t.Fatalf("player did not leave orange portal outward: %+v", p)
	}
}

func TestPortalProjectileFallbackAlwaysPlacesOnMapSurface(t *testing.T) {
	arena := Map{Platforms: []Rect{{X: 100, Y: 100, W: 400, H: 40}}}
	b := &Bullet{X: 1500, Y: 0, OwnerID: 1, PortalColor: PortalBlue}
	var portals []Portal
	if !placePortalOnNearestSurface(&portals, b, arena) || len(portals) != 1 || !portals[0].Active {
		t.Fatalf("fallback portals=%+v", portals)
	}
}

func TestPortalGunCombatHitKnocksEnemyWithoutPair(t *testing.T) {
	target := &Player{ID: 2, Active: true, Team: 2, X: 20, Y: 0, PlayerScale: .8, DamageMulti: 1}
	b := &Bullet{DevelopedProjectile: DevelopedProjectilePortal, X: 0, Y: -20, VX: 28, OwnerID: 1, Team: 1, Facing: 1, PortalColor: PortalBlue}
	if !hitPortalPlayerAlongSegment(b, 0, -20, 28, -20, []*Player{target}, nil) || !b.Dead || target.VX < 24 || target.VY >= 0 {
		t.Fatalf("unpaired portal combat hit failed: bullet=%+v target=%+v", b, target)
	}
}

func TestPortalGunCombatHitSendsEnemyOutOppositePortal(t *testing.T) {
	target := &Player{ID: 2, Active: true, Team: 2, X: 20, Y: 0, PlayerScale: .8, DamageMulti: 1}
	b := &Bullet{DevelopedProjectile: DevelopedProjectilePortal, X: 0, Y: -20, VX: 28, OwnerID: 1, Team: 1, Facing: 1, PortalColor: PortalBlue}
	portals := []Portal{{X: 400, Y: 200, NormalX: -1, OwnerID: 1, Color: PortalOrange, Active: true}}
	if !hitPortalPlayerAlongSegment(b, 0, -20, 28, -20, []*Player{target}, &portals) {
		t.Fatal("paired portal shot missed target")
	}
	if target.X >= 400 || target.PortalCooldown == 0 || target.VX >= 0 {
		t.Fatalf("target did not emerge from orange portal: %+v", target)
	}
}

func TestOldFlameCanBurnItsOwnerForTimedSelfDamage(t *testing.T) {
	p := &Player{ID: 1, Active: true, Team: 1, X: 0, Y: 0, PlayerScale: .8, DamageMulti: 1}
	b := &Bullet{
		DevelopedProjectile: DevelopedProjectileFlame,
		X:                   0, Y: -20, Firepower: 1.5, OwnerID: 1, Team: 1, Facing: 1,
		Time: 5, IgniteFrames: 150, VisualScale: 1,
	}
	if !hitPlayerWithFlame(b, []*Player{p}) || p.BurnTime != 150 {
		t.Fatalf("self flame did not start timed burn: bullet=%+v player=%+v", b, p)
	}
}

func TestMolotovCreatesPersistentFireInsteadOfGrenadeBlast(t *testing.T) {
	arena := Map{Platforms: []Rect{{X: 20, Y: 100, W: 180, H: 20}}}
	grenades := []Grenade{{X: 100, Y: 100, Type: ThrowableMolotov, OwnerID: 1, HitGround: true}}
	var explosions []Explosion
	var blast []BlastEffect
	var fire []FirePatch
	updateGrenades(arena, grenades, nil, &explosions, nil, &blast, &fire)
	if !grenades[0].Dead || len(fire) == 0 || len(explosions) != 0 || len(blast) != 0 {
		t.Fatalf("molotov result: dead=%v fire=%d explosions=%d blast=%d", grenades[0].Dead, len(fire), len(explosions), len(blast))
	}
	for _, patch := range fire {
		if patch.Y != 100 || patch.X < 20 || patch.X > 200 {
			t.Fatalf("molotov fire escaped platform surface: %+v", patch)
		}
	}
}

func TestMolotovImpactIgnitesWithoutBehavingLikeAHighExplosive(t *testing.T) {
	p := &Player{ID: 2, Active: true, X: 90, Y: 30, DamageMulti: 1}
	g := Grenade{X: 0, Y: 0, Type: ThrowableMolotov, OwnerID: 1}
	spawnMolotovFire(Map{}, &g, []*Player{p}, nil)
	if p.BurnTime != 165 || p.BurnPower != 20 || p.VX < 5.5 || p.VX > 9 || p.VY > -2.8 {
		t.Fatalf("molotov impact balance wrong: %+v", p)
	}
}

func TestThrowableProfilesHaveDifferentBlastBehaviour(t *testing.T) {
	mk2 := throwableProfiles[ThrowableMk2]
	f1 := throwableProfiles[ThrowableF1]
	m24 := throwableProfiles[ThrowableStielhand]
	if mk2.Radius == f1.Radius || f1.Fuse == m24.Fuse || m24.HorizontalPush <= mk2.HorizontalPush {
		t.Fatalf("throwable profiles collapsed: mk2=%+v f1=%+v m24=%+v", mk2, f1, m24)
	}
}

func TestDynamiteKeepsSourceStrengthAtBlastEdge(t *testing.T) {
	p := &Player{ID: 2, Active: true, X: 200, Y: 30, DamageMulti: 1}
	g := Grenade{X: 0, Y: 0, Type: ThrowableDynamite, OwnerID: 1}
	explodeGrenade(&g, []*Player{p})
	if p.VX < 29.9 || p.VX > 30.1 {
		t.Fatalf("dynamite edge impulse=%v want source ~30", p.VX)
	}
}

func TestFlamethrowerContactIsContinuousHeatNotBulletHose(t *testing.T) {
	p := &Player{ID: 2, Active: true, Team: 2, X: 0, Y: 0, PlayerScale: .8, DamageMulti: 1}
	for i := 0; i < 10; i++ {
		b := &Bullet{
			DevelopedProjectile: DevelopedProjectileFlame,
			X:                   0, Y: -20, Firepower: 1.45, OwnerID: 1, Team: 1, Facing: 1,
			Time: 6, IgniteFrames: 100, VisualScale: 1,
		}
		if !hitPlayerWithFlame(b, []*Player{p}) {
			t.Fatalf("flame contact %d missed target", i)
		}
	}
	if p.BurnTime != 100 || p.VX < 8 || p.VX > 10 || p.FireContactCooldown == 0 {
		t.Fatalf("continuous flame contact balance wrong: burn=%d vx=%.2f cooldown=%d", p.BurnTime, p.VX, p.FireContactCooldown)
	}
}

func TestMolotovGroundFireGivesReadableContactKnockback(t *testing.T) {
	p := &Player{ID: 2, Active: true, X: 12, Y: 100, PlayerScale: .8, DamageMulti: 1}
	patches := []FirePatch{{X: 0, Y: 100, OwnerID: 1, Life: 200, Frame: 5}}
	updateFirePatches(patches, []*Player{p})
	if p.BurnTime == 0 || p.VX < 7 || p.VY > -4 || p.FireContactCooldown == 0 {
		t.Fatalf("ground fire contact too weak: burn=%d vx=%.2f vy=%.2f cooldown=%d", p.BurnTime, p.VX, p.VY, p.FireContactCooldown)
	}
	vx := p.VX
	patches[0].Frame = 5
	updateFirePatches(patches, []*Player{p})
	if p.VX != vx {
		t.Fatalf("overlapping fire stacked knockback inside cooldown: %.2f -> %.2f", vx, p.VX)
	}
}

func TestAIOnlyDetectsInvisiblePlayerAtCloseRange(t *testing.T) {
	ai := &Player{ID: 1, Active: true, AI: true, Team: 1, AITargetTimer: 39}
	target := &Player{ID: 2, Active: true, Team: 2, X: 300, InvisibleTime: 100}
	g := &Game{players: []*Player{ai, target}}
	prepareAITarget(ai, g)
	if ai.AITargetValid {
		t.Fatal("AI acquired distant invisible target")
	}
	target.X = 100
	ai.AITargetTimer = 39
	prepareAITarget(ai, g)
	if !ai.AITargetValid || ai.AITargetPlayerID != target.ID {
		t.Fatal("AI failed to detect close invisible target")
	}
}

func TestAISearchesLastSeenPositionAfterTargetTurnsInvisible(t *testing.T) {
	ai := &Player{
		ID: 1, Active: true, AI: true, Team: 1,
		AITargetValid: true, AITargetKind: aiTargetPlayer, AITargetPlayerID: 2,
		AITargetX: 220, AITargetY: 300,
	}
	target := &Player{ID: 2, Active: true, Team: 2, X: 700, Y: 120, InvisibleTime: 100}
	g := &Game{players: []*Player{ai, target}}
	prepareAITarget(ai, g)
	if !ai.AITargetValid || ai.AITargetKind != aiTargetSearch || ai.AISearchTime == 0 {
		t.Fatalf("AI did not enter invisible search: %+v", ai)
	}
	if ai.AISearchX != 220 || ai.AISearchY != 300 || ai.AISearchX == target.X || ai.AISearchY == target.Y {
		t.Fatalf("AI used current invisible coordinates instead of last seen point: search=(%v,%v) target=(%v,%v)", ai.AISearchX, ai.AISearchY, target.X, target.Y)
	}
}

func TestAISearchFiresIrregularBursts(t *testing.T) {
	ai := &Player{
		ID: 1, Active: true, AI: true, Team: 1, X: 100, Y: 300, Facing: 1, JumpNum: 2,
		AITargetValid: true, AITargetKind: aiTargetSearch,
		AISearchTime: aiInvisibleSearchDuration, AISearchX: 260, AISearchY: 300, AISearchPlayerID: 2,
		Weapon: NewWeapon(1),
	}
	g := &Game{arena: OriginalMap1(), players: []*Player{ai}}
	shots := 0
	for i := 0; i < 120; i++ {
		decideAIController(ai, g)
		if ai.AIShoot {
			shots++
		}
	}
	if shots == 0 || shots >= 60 {
		t.Fatalf("AI search shooting is absent or not burst-like: shots=%d", shots)
	}
}

func TestAIReacquiresVisiblePlayerDuringSearch(t *testing.T) {
	ai := &Player{
		ID: 1, Active: true, AI: true, Team: 1, AITargetTimer: 39,
		AITargetValid: true, AITargetKind: aiTargetSearch,
		AISearchTime: 100, AISearchX: 200, AISearchY: 300, AISearchPlayerID: 2,
	}
	target := &Player{ID: 2, Active: true, Team: 2, X: 350, Y: 300}
	g := &Game{players: []*Player{ai, target}}
	prepareAITarget(ai, g)
	if !ai.AITargetValid || ai.AITargetKind != aiTargetPlayer || ai.AITargetPlayerID != target.ID || ai.AISearchTime != 0 {
		t.Fatalf("AI did not reacquire visible target: %+v", ai)
	}
}

func TestOrganizedGunLibraryIncludesWholeCatalogueAndNewSpecials(t *testing.T) {
	all := organizedGunLibraryNumbers(0)
	if len(all) != 228 {
		t.Fatalf("organized catalogue=%d want228", len(all))
	}
	special := organizedGunLibraryNumbers(DevelopedCategorySpecial)
	seen := map[int]bool{}
	for _, number := range special {
		seen[number] = true
	}
	for _, number := range []int{187, 188, 189} {
		if !seen[number] {
			t.Fatalf("new special weapon %d missing from category", number)
		}
	}
}

func TestGunLibraryThrowableTabHasClickableCards(t *testing.T) {
	throwableTab := len(organizedGunLibraryTabs) - 1
	if !organizedGunLibraryTabs[throwableTab].throwables {
		t.Fatal("last gun library tab is not THROWABLES")
	}
	r := organizedThrowableLibraryCardRect(ThrowableStielhand)
	kind, ok := organizedGunLibraryThrowableAt(r.X+r.W/2, r.Y+r.H/2)
	if !ok || kind != ThrowableStielhand {
		t.Fatalf("throwable card hit kind=%d ok=%v", kind, ok)
	}
	g := &Game{gunLibraryCategory: throwableTab}
	if !g.organizedGunLibraryThrowableTab() {
		t.Fatal("throwable tab not recognized as throwable library")
	}
}

func TestGeneratedThrowableSpritesHaveTransparentVisibleArtwork(t *testing.T) {
	files := []string{"mk2.png", "f1.png", "m24.png", "rgd5.png", "molotov.png"}
	for _, filename := range files {
		img := decodeOriginalPNGIn("custom", "throwables", filename)
		if img == nil {
			t.Fatalf("missing generated throwable sprite %s", filename)
		}
		visible, ok := alphaBounds(img)
		if !ok || visible.Dx() < 100 || visible.Dy() < 100 {
			t.Fatalf("generated throwable %s has invalid visible bounds %v", filename, visible)
		}
		_, _, _, cornerAlpha := img.At(img.Bounds().Min.X, img.Bounds().Min.Y).RGBA()
		if cornerAlpha != 0 {
			t.Fatalf("generated throwable %s does not have a transparent corner", filename)
		}
	}
}
