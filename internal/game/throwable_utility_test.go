package game

import (
	"math"
	"testing"
)

func utilityArena() Map {
	return Map{Gravity: DefaultGravity, Platforms: []Rect{{X: 0, Y: 300, W: 900, H: 24}}, LowestY: 600}
}

func utilityPlayer(id int, x float64, facing int) *Player {
	return &Player{ID: id, Team: id, Active: true, X: x, Y: 300, Facing: facing, JumpNum: 2, PlayerScale: 0.8, DamageMulti: 1, PerkNumber: 7, Weight: 1}
}

func TestThrowableTypeNumbersAreAppendOnly(t *testing.T) {
	want := map[int]int{
		ThrowableDynamite: 0, ThrowableMk2: 1, ThrowableF1: 2, ThrowableStielhand: 3, ThrowableRGD5: 4,
		ThrowableMolotov: 5, ThrowableShock: 6, ThrowableFlashbang: 7, ThrowableSmoke: 8, ThrowableTearGas: 9,
	}
	for got, n := range want {
		if got != n {
			t.Fatalf("throwable number moved: got %d want %d", got, n)
		}
	}
	if throwableTypeCount != 10 || len(throwableProfiles) != throwableTypeCount {
		t.Fatalf("type count %d profiles %d", throwableTypeCount, len(throwableProfiles))
	}
}

func TestThrowableProfileBehaviours(t *testing.T) {
	want := map[int]throwableBehavior{
		ThrowableDynamite: throwableBehaviorExplode, ThrowableMk2: throwableBehaviorExplode,
		ThrowableF1: throwableBehaviorExplode, ThrowableStielhand: throwableBehaviorExplode,
		ThrowableRGD5: throwableBehaviorExplode, ThrowableMolotov: throwableBehaviorIgnite,
		ThrowableShock: throwableBehaviorShock, ThrowableFlashbang: throwableBehaviorFlash,
		ThrowableSmoke: throwableBehaviorSmoke, ThrowableTearGas: throwableBehaviorTearGas,
	}
	for kind, behavior := range want {
		if got := throwableProfiles[kind].Behavior; got != behavior {
			t.Fatalf("%s behaviour %d want %d", throwableName(kind), got, behavior)
		}
		if throwableProfiles[kind].ImpactDetonate != (kind == ThrowableMolotov) {
			t.Fatalf("%s impact detonation flag wrong", throwableName(kind))
		}
	}
	if f := throwableProfiles[ThrowableFlashbang].Fuse; f > 35 {
		t.Fatalf("flashbang fuse %d should be short (~30)", f)
	}
}

// detonate runs updateGrenadesWithAreas until the grenade is gone.
func detonate(t *testing.T, arena Map, gr Grenade, players []*Player) []AreaEffect {
	t.Helper()
	gr.Fuse = throwableProfileFor(gr.Type).Fuse
	grenades := []Grenade{gr}
	var areas []AreaEffect
	var explosions []Explosion
	var dyn []DynamiteEffect
	var blast []BlastEffect
	var fire []FirePatch
	updateGrenadesWithAreas(arena, grenades, players, &explosions, &dyn, &blast, &fire, &areas)
	if !grenades[0].Dead {
		t.Fatal("grenade did not go off")
	}
	if len(explosions) != 0 || len(fire) != 0 {
		t.Fatalf("utility throwable exploded/ignited: %d explosions %d fire", len(explosions), len(fire))
	}
	return areas
}

func TestShockGrenadePushesOutwardWithoutDamage(t *testing.T) {
	arena := utilityArena()
	left := utilityPlayer(2, 360, 1)
	right := utilityPlayer(3, 440, -1)
	far := utilityPlayer(4, 800, -1)
	areas := detonate(t, arena, Grenade{X: 400, Y: 292, Type: ThrowableShock, OwnerID: 1}, []*Player{left, right, far})
	if len(areas) != 1 || areas[0].Kind != areaEffectGas {
		t.Fatalf("no gas cloud: %+v", areas)
	}
	for i := 0; i < 20; i++ {
		updateAreaEffects(areas, []*Player{left, right, far})
	}
	if left.VX >= 0 || right.VX <= 0 {
		t.Fatalf("gas did not push outward: left vx=%.2f right vx=%.2f", left.VX, right.VX)
	}
	if far.VX != 0 || far.VY != 0 {
		t.Fatal("gas reached a player outside its radius")
	}
	if left.HitNumber != 0 || left.BurnTime != 0 || left.LastHitBy != 1 {
		t.Fatalf("gas should only push and credit the thrower: %+v", left)
	}
	// Edge push is weaker than centre push.
	near := utilityPlayer(5, 420, 1)
	edge := utilityPlayer(6, 530, 1)
	a := []AreaEffect{{Kind: areaEffectGas, X: 400, Y: 270, MaxRadius: gasCloudRadius, Life: 100, MaxLife: 150, Frame: 20}}
	updateAreaEffects(a, []*Player{near, edge})
	if near.VX <= edge.VX || edge.VX <= 0 {
		t.Fatalf("falloff wrong: near %.3f edge %.3f", near.VX, edge.VX)
	}
	// BLAST SHIELD halves it; GAS MASK does not protect from the shock push.
	plain, shield, mask := utilityPlayer(7, 450, 1), utilityPlayer(8, 450, 1), utilityPlayer(9, 450, 1)
	shield.PerkNumber = 6
	mask.AbilityNumber = abilityGasMask
	a = []AreaEffect{{Kind: areaEffectGas, X: 400, Y: 270, MaxRadius: gasCloudRadius, Life: 100, MaxLife: 150, Frame: 20}}
	updateAreaEffects(a, []*Player{plain, shield, mask})
	if math.Abs(shield.VX-plain.VX*gasBlastShieldScale) > 1e-9 {
		t.Fatalf("blast shield push %.3f want half of %.3f", shield.VX, plain.VX)
	}
	if mask.VX != plain.VX {
		t.Fatal("gas mask must not reduce the shock push")
	}
}

func TestFlashbangBlindRules(t *testing.T) {
	arena := utilityArena()
	arena.Platforms = append(arena.Platforms, Rect{X: 600, Y: 150, W: 60, H: 140}) // wall
	fx, fy := 300.0, 288.0-flashPointLift

	toward := utilityPlayer(1, 200, 1)
	away := utilityPlayer(2, 200, -1)
	ft, st := flashbangBlindFor(arena, fx, fy, toward)
	fa, sa := flashbangBlindFor(arena, fx, fy, away)
	if ft < 120 || ft > 180 || st < flashFullStrengthMin {
		t.Fatalf("facing toward: %d frames %.2f", ft, st)
	}
	if fa != flashAwayBlindFrames || sa >= st {
		t.Fatalf("facing away: %d frames %.2f", fa, sa)
	}
	// Closer = longer.
	close := utilityPlayer(3, 280, 1)
	if fc, _ := flashbangBlindFor(arena, fx, fy, close); fc <= ft {
		t.Fatalf("closer player blinded %d <= %d", fc, ft)
	}
	lower := utilityPlayer(4, 300, 1)
	lower.Y = 420
	if f, _ := flashbangBlindFor(arena, fx, fy, lower); f != 0 {
		t.Fatal("player on a lower level was blinded")
	}
	occluded := utilityPlayer(5, 700, -1)
	occluded.Y = 288
	if f, _ := flashbangBlindFor(arena, 500, fy, occluded); f != 0 {
		t.Fatal("platform did not block the flash")
	}
	goggles := utilityPlayer(6, 250, 1)
	goggles.AbilityNumber = abilityFlashGoggles
	if f, _ := flashbangBlindFor(arena, fx, fy, goggles); f != 0 {
		t.Fatal("flash goggles did not protect")
	}
	mask := utilityPlayer(7, 250, 1)
	mask.AbilityNumber = abilityGasMask
	if f, _ := flashbangBlindFor(arena, fx, fy, mask); f == 0 {
		t.Fatal("gas mask must not protect from flash")
	}

	// The thrower blinds themselves too.
	thrower := utilityPlayer(1, 260, 1)
	areas := detonate(t, utilityArena(), Grenade{X: 300, Y: 288, Type: ThrowableFlashbang, OwnerID: 1}, []*Player{thrower})
	if thrower.BlindTime == 0 {
		t.Fatal("thrower facing their flash was not blinded")
	}
	if len(areas) != 1 || areas[0].Kind != areaEffectFlash {
		t.Fatalf("missing flash burst: %+v", areas)
	}
}

func TestBlindedPlayerHasSpreadAndCannotThrow(t *testing.T) {
	p := utilityPlayer(1, 300, 1)
	p.Grenades = 3
	p.AI = true
	if aimDisturbanceDegrees(p, false) != 0 {
		t.Fatal("unblinded player has aim spread")
	}
	applyBlind(p, 150, 1)
	if d := aimDisturbanceDegrees(p, false); d < 10 {
		t.Fatalf("blind spread %.1f too small", d)
	}
	b := Bullet{Kind: BulletNormal, VX: 25, Facing: 1}
	disturbBulletAim(&b, 12)
	if math.Abs(math.Atan2(b.VY, b.VX)*180/math.Pi-12) > 1e-6 || math.Abs(math.Hypot(b.VX, b.VY)-25) > 1e-9 {
		t.Fatalf("bullet not rotated: %+v", b)
	}

	var grenades []Grenade
	p.AIGrenade = true
	for i := 0; i < 5; i++ {
		updateGrenadeInput(p, &grenades)
	}
	p.AIGrenade = false
	updateGrenadeInput(p, &grenades)
	if len(grenades) != 0 || p.Grenades != 3 {
		t.Fatal("blinded player threw a grenade")
	}
	for p.BlindTime > 0 {
		p.updatePowerupTimers()
	}
	if p.BlindStrength != 0 || p.BlindMax != 0 {
		t.Fatal("blind state not cleared")
	}
}

func TestSmokeBreaksAIDetectionUnlessThermal(t *testing.T) {
	g := &Game{}
	observer := utilityPlayer(1, 100, 1)
	observer.AI = true
	target := utilityPlayer(2, 500, -1)
	g.areaEffects = []AreaEffect{{Kind: areaEffectSmoke, X: 300, Y: 260, Radius: 150, MaxRadius: 170, Life: 300, MaxLife: 450, Frame: 60}}
	if g.aiCanDetect(observer, target) {
		t.Fatal("bot sees through smoke")
	}
	if !g.aiTargetConcealed(observer, target) {
		t.Fatal("smoke-hidden target should trigger the search behaviour")
	}
	observer.AbilityNumber = abilityThermalGoggles
	if !g.aiCanDetect(observer, target) {
		t.Fatal("thermal goggles bot should see through smoke")
	}
	observer.PerkNumber = 7
	g.areaEffects = nil
	if !g.aiCanDetect(observer, target) {
		t.Fatal("clear line of sight not detected")
	}
	applyBlind(observer, 60, 1)
	if g.aiCanDetect(observer, target) {
		t.Fatal("blinded bot still detects")
	}
}

func TestSmokeAddsShotSpreadExceptThermal(t *testing.T) {
	g := &Game{areaEffects: []AreaEffect{{Kind: areaEffectSmoke, X: 400, Y: 260, Radius: 150, MaxRadius: 170, Life: 300, MaxLife: 450, Frame: 60}}}
	shooter := utilityPlayer(1, 100, 1)
	if !g.shooterSmokePenalty(shooter) {
		t.Fatal("shooting through smoke has no penalty")
	}
	shooter.AbilityNumber = abilityThermalGoggles
	if g.shooterSmokePenalty(shooter) {
		t.Fatal("thermal shooter penalised")
	}
}

func runMoveFrame(p *Player) {
	arena := utilityArena()
	p.AI = true
	p.Update(arena, nil, nil)
}

func TestTearGasSlowsLowersJumpSpreadsAndWearsOff(t *testing.T) {
	base := utilityPlayer(1, 300, 1)
	gassed := utilityPlayer(2, 300, 1)
	mask := utilityPlayer(3, 300, 1)
	mask.AbilityNumber = abilityGasMask
	cloud := []AreaEffect{{Kind: areaEffectTear, X: 300, Y: 266, MaxRadius: tearGasCloudRadius, Life: 200, MaxLife: 270, Frame: 50}}
	updateAreaEffects(cloud, []*Player{gassed, mask})
	if gassed.TearGasTime == 0 {
		t.Fatal("player inside tear gas is not coughing")
	}
	if mask.TearGasTime != 0 {
		t.Fatal("gas mask did not protect from tear gas")
	}
	if gassed.VX != 0 || gassed.VY != 0 {
		t.Fatal("tear gas must not push")
	}
	if aimDisturbanceDegrees(gassed, false) <= 0 {
		t.Fatal("tear gas adds no aim spread")
	}

	base.AIRight, gassed.AIRight = true, true
	runMoveFrame(base)
	runMoveFrame(gassed)
	if math.Abs(gassed.VX-base.VX*tearGasMoveScale) > 1e-6 {
		t.Fatalf("tear gas speed %.3f want %.3f", gassed.VX, base.VX*tearGasMoveScale)
	}

	jb, jg := utilityPlayer(4, 300, 1), utilityPlayer(5, 300, 1)
	jg.TearGasTime = tearGasLingerFrames
	jb.AIUp, jg.AIUp = true, true
	runMoveFrame(jb)
	runMoveFrame(jg)
	if !(jg.VY > jb.VY) || math.Abs((jg.VY-DefaultGravity)-(jb.VY-DefaultGravity)*tearGasJumpScale) > 1e-6 {
		t.Fatalf("tear gas jump vy %.3f vs %.3f", jg.VY, jb.VY)
	}

	// Leaving the cloud: coughing lingers about a second, then wears off.
	left := utilityPlayer(6, 300, 1)
	updateAreaEffects(cloud, []*Player{left})
	left.X = 800
	for i := 0; i < tearGasLingerFrames-1; i++ {
		updateAreaEffects(cloud, []*Player{left})
		left.updatePowerupTimers()
	}
	if left.TearGasTime == 0 {
		t.Fatal("coughing ended immediately after leaving")
	}
	for i := 0; i < 5; i++ {
		left.updatePowerupTimers()
	}
	if left.TearGasTime != 0 {
		t.Fatal("coughing did not wear off")
	}
}

func TestKevlarReducesBulletKnockbackOnly(t *testing.T) {
	arena := utilityArena()
	shoot := func(ability int) float64 {
		p := utilityPlayer(2, 300, -1)
		p.AbilityNumber = ability
		h := p.Hitbox()
		b := Bullet{Kind: BulletNormal, X: h.X + h.W/2, Y: h.Y + h.H/2, Firepower: 10, OwnerID: 1, Team: 1, Facing: 1, Alpha: 1}
		if !hitPlayerWithBullet(arena, &b, []*Player{p}, false) {
			t.Fatal("bullet missed")
		}
		return p.VX
	}
	plain, kevlar := shoot(abilityNone), shoot(abilityKevlar)
	if math.Abs(kevlar-plain*kevlarBulletKnockbackScale) > 1e-9 {
		t.Fatalf("kevlar bullet push %.3f want %.3f", kevlar, plain*kevlarBulletKnockbackScale)
	}

	blastPush := func(ability int) float64 {
		p := utilityPlayer(2, 380, 1)
		p.AbilityNumber = ability
		gr := Grenade{X: 300, Y: 280, Type: ThrowableDynamite, OwnerID: 1}
		explodeGrenade(&gr, []*Player{p})
		return p.VX
	}
	if blastPush(abilityNone) != blastPush(abilityKevlar) {
		t.Fatal("kevlar must not reduce explosion knockback")
	}

	base, vest := utilityPlayer(3, 300, 1), utilityPlayer(4, 300, 1)
	vest.AbilityNumber = abilityKevlar
	base.AIRight, vest.AIRight = true, true
	runMoveFrame(base)
	runMoveFrame(vest)
	if math.Abs(vest.VX-base.VX*kevlarMoveScale) > 1e-6 {
		t.Fatalf("kevlar speed %.3f want %.3f", vest.VX, base.VX*kevlarMoveScale)
	}
	jb, jv := utilityPlayer(5, 300, 1), utilityPlayer(6, 300, 1)
	jv.AbilityNumber = abilityKevlar
	jb.AIUp, jv.AIUp = true, true
	runMoveFrame(jb)
	runMoveFrame(jv)
	if math.Abs((jv.VY-DefaultGravity)-(jb.VY-DefaultGravity)*kevlarJumpScale) > 1e-6 {
		t.Fatalf("kevlar jump vy %.3f vs %.3f", jv.VY, jb.VY)
	}
}

func TestGearMovedToAbilitySlotOnlineAndInMenus(t *testing.T) {
	g := &Game{}
	for _, gear := range []int{abilityFlashGoggles, abilityThermalGoggles, abilityGasMask, abilityKevlar} {
		cfg := CustomPlayerConfig{Name: "P", Color: 1, Shirt: 1, Hat: 1, Eyes: 1, Gun: 1, Perk: 7, Ability: gear, Throwable: ThrowableTearGas, Type: 1, Team: 1}
		if _, err := sanitizeRemotePlayerConfig(cfg); err != nil {
			t.Fatalf("gear ability %d rejected online: %v", gear, err)
		}
		g.onlinePerksEnabled = true
		if g.onlineAllowedAbility(gear, abilityNone) != gear {
			t.Fatalf("gear ability %d stripped online", gear)
		}
		if abilityName(gear) == "ABILITY" || abilityMenuIndex(gear) < 0 {
			t.Fatalf("gear ability %d has no name / menu cell", gear)
		}
	}
	// The old gear perk numbers and the reserved GM2 perks are no perks.
	for _, perk := range []int{10, 11, 12, 13, 14, 15, 16, 17, 18} {
		cfg := CustomPlayerConfig{Name: "P", Color: 1, Shirt: 1, Hat: 1, Eyes: 1, Gun: 1, Perk: perk, Type: 1, Team: 1}
		if _, err := sanitizeRemotePlayerConfig(cfg); err == nil {
			t.Fatalf("perk %d accepted online", perk)
		}
		if g.customSelectionUnlocked(customEditPerk, perk) {
			t.Fatalf("perk %d still selectable in the perk grid", perk)
		}
	}
	// Every selectable perk has a menu cell, all distinct.
	seen := map[int]bool{}
	for _, item := range developedPerkSelectionItems() {
		if item.Number > 9 {
			t.Fatalf("perk grid still offers %d", item.Number)
		}
		seen[item.Number] = true
	}
	if len(sourceCustomSelectionItems()[customEditPerk]) > 0 {
		for _, perk := range selectablePerkNumbers() {
			if !seen[perk] {
				t.Fatalf("perk %d missing from the perk grid", perk)
			}
		}
	}
}

func TestHeadgearReplacesHatOnlyAtRenderTime(t *testing.T) {
	cfg := CustomPlayerConfig{Hat: 12, Ability: abilityThermalGoggles}
	p := utilityPlayer(1, 100, 1)
	p.HatNumber = cfg.Hat
	p.AbilityNumber = cfg.Ability
	if playerHeadgear(p) != abilityThermalGoggles {
		t.Fatal("goggles not active")
	}
	if p.HatNumber != 12 || cfg.Hat != 12 {
		t.Fatal("headgear mutated the chosen hat")
	}
	p.AbilityNumber = abilityNone
	if playerHeadgear(p) != 0 {
		t.Fatal("hat should show again without a head perk")
	}
	p.AbilityNumber = abilityKevlar
	if playerHeadgear(p) != 0 {
		t.Fatal("kevlar is body gear and must not hide the hat")
	}
	for _, perk := range []int{abilityFlashGoggles, abilityGasMask} {
		p.AbilityNumber = perk
		if playerHeadgear(p) != perk {
			t.Fatalf("head perk %d not detected", perk)
		}
	}
}

func TestBotsLeaveGasClouds(t *testing.T) {
	arena := utilityArena()
	p := utilityPlayer(1, 330, 1)
	p.AI = true
	p.AIRight = false
	aiAvoidGasClouds(p, arena.Platforms, []AreaEffect{{Kind: areaEffectTear, X: 300, Y: 270, Radius: 120, Life: 100}})
	if !p.AIRight || p.AILeft {
		t.Fatal("bot did not walk out of tear gas")
	}
	mask := utilityPlayer(2, 330, -1)
	mask.AI = true
	mask.AbilityNumber = abilityGasMask
	aiAvoidGasClouds(mask, arena.Platforms, []AreaEffect{{Kind: areaEffectTear, X: 300, Y: 270, Radius: 120, Life: 100}})
	if mask.AIRight || mask.AILeft {
		t.Fatal("gas-mask bot should ignore tear gas")
	}
}

func TestBotTurnsAwayFromLiveFlashbang(t *testing.T) {
	arena := utilityArena()
	p := utilityPlayer(1, 200, 1)
	p.AI = true
	if !aiTurnAwayFromFlash(arena, p, 300, 276) {
		t.Fatal("bot did not react to a flashbang in view")
	}
	aiApplyFlashAvoid(p)
	if !p.AILeft || p.AIShoot {
		t.Fatal("bot did not turn its back to the flash")
	}
	p.Facing = -1
	aiApplyFlashAvoid(p)
	if p.AILeft || p.AIRight {
		t.Fatal("bot keeps walking instead of holding still facing away")
	}
}

func TestNewPlayerAndAreaFieldsSurviveSnapshotCodec(t *testing.T) {
	in := netSnapshot{
		Players:     []Player{{ID: 1, Active: true, BlindTime: 90, BlindMax: 120, BlindStrength: 0.8, TearGasTime: 40, AbilityNumber: abilityGasMask}},
		AreaEffects: []AreaEffect{{Kind: areaEffectSmoke, X: 10, Y: 20, Radius: 100, MaxRadius: 170, OwnerID: 2, Life: 300, MaxLife: 450, Frame: 5}},
	}
	data := netEncodeValue(nil, &in)
	var out netSnapshot
	if _, err := netDecodeValue(data, &out); err != nil {
		t.Fatal(err)
	}
	p := out.Players[0]
	if p.BlindTime != 90 || p.BlindMax != 120 || math.Abs(p.BlindStrength-0.8) > 1e-6 || p.TearGasTime != 40 || p.AbilityNumber != abilityGasMask {
		t.Fatalf("player status lost: %+v", p)
	}
	if len(out.AreaEffects) != 1 || out.AreaEffects[0].Kind != areaEffectSmoke || out.AreaEffects[0].Life != 300 {
		t.Fatalf("area effect lost: %+v", out.AreaEffects)
	}
	if err := validateNetSnapshot(out); err != nil {
		t.Fatal(err)
	}
}
