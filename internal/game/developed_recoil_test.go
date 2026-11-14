package game

import (
	"math"
	"testing"
)

// Self-recoil: every accepted round does `vx -= recoil * facing` on the
// shooter (DefineSprite_697_player shooting branch), once per trigger for a
// shotgun load and once per round of a burst. The same 0.93 friction is
// applied on the ground and in the air, so a weapon held down settles at
// about 13x its average recoil per frame. The caps below are the strongest
// source weapon of each class; no added weapon may push its shooter harder.

// sourceRecoilGolden pins the ActionScript fire delay and recoil of every
// source weapon (gun DoAction.as: rof, recoil). The added-weapon caps are
// derived from these, so the originals must never drift.
var sourceRecoilGolden = map[int]struct {
	ROF    int
	Recoil float64
}{
	1: {10, 1}, 2: {13, 2}, 3: {6, .7}, 4: {16, 3}, 5: {8, 1}, 6: {22, 5},
	8: {10, 1}, 9: {12, 7}, 10: {4, .8}, 11: {31, 10}, 12: {4, .8}, 13: {25, 5},
	14: {13, 1.5}, 15: {6, .8}, 16: {25, 8}, 17: {5, .8}, 18: {18, 5}, 19: {3, .3},
	20: {4, .5}, 21: {3, .8}, 22: {4, .8}, 23: {4, .8}, 24: {4, .8}, 25: {4, .8},
	26: {4, .2}, 27: {4, .8}, 28: {3, .8}, 29: {4, 1}, 30: {3, .2}, 31: {3, .5},
	32: {3, .8}, 33: {31, 10}, 34: {31, 10}, 35: {31, 10}, 36: {31, 10}, 37: {25, 10},
	38: {25, 10}, 39: {15, 1.2}, 40: {15, 1.2}, 41: {12, 1.2}, 42: {12, 1.5}, 43: {18, 5},
	44: {40, 8}, 45: {24, 5}, 46: {6, .2}, 47: {6, .8}, 48: {25, 5}, 49: {25, 5},
	50: {25, 10}, 51: {12, 3}, 52: {12, 2}, 53: {25, 5}, 54: {18, 3}, 55: {2, .5},
	56: {4, .8}, 57: {3, .8}, 58: {5, .8}, 59: {5, .8}, 60: {4, .8}, 61: {7, 2},
	62: {4, .2}, 63: {7, 2}, 64: {3, .8}, 65: {3, .8}, 66: {3, .8}, 67: {3, .4},
	68: {2, .3}, 69: {3, .5}, 70: {4, .6}, 71: {4, .6}, 72: {5, 1}, 73: {5, 1},
	74: {4, .4}, 75: {5, .4}, 76: {3, .6}, 77: {18, 1.5}, 78: {12, 1.2}, 79: {31, 10},
	80: {25, 5}, 81: {31, 7}, 82: {6, .8}, 83: {12, .6}, 84: {25, 5}, 85: {10, .5},
	86: {12, 2},
}

var (
	sourcePistolNumbers  = []int{1, 2, 3, 4, 5, 6, 8}
	sourceSniperNumbers  = []int{11, 14, 16, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 77, 78, 79, 80, 81}
	sourceShotgunNumbers = []int{13, 15, 18, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 82, 83, 84, 85, 86}
)

// selfRecoilPerFrame is the average shooter push per frame with the trigger
// held: every round of a burst over the burst plus its pause.
func selfRecoilPerFrame(number int) float64 {
	def, _ := WeaponByNumber(number)
	spec, developed := developedWeaponSpec(number)
	rounds, cycle := 1, def.ROF
	if developed {
		rounds = developedBurstRounds(spec)
		if rounds > 1 {
			cycle += (rounds - 1) * developedBurstGap(spec)
		}
	}
	return float64(rounds) * def.Recoil / float64(max(cycle, 1))
}

// selfRecoilPerTrigger is the push of one trigger pull (a whole burst).
func selfRecoilPerTrigger(number int) float64 {
	def, _ := WeaponByNumber(number)
	if spec, developed := developedWeaponSpec(number); developed {
		return float64(developedBurstRounds(spec)) * def.Recoil
	}
	return def.Recoil
}

type selfRecoilCap struct {
	PerFrame, PerTrigger float64
}

func sourceSelfRecoilCap(t *testing.T, numbers ...[]int) selfRecoilCap {
	t.Helper()
	var c selfRecoilCap
	for _, list := range numbers {
		for _, n := range list {
			if _, ok := weaponCatalog[n]; !ok {
				t.Fatalf("source weapon %d missing", n)
			}
			c.PerFrame = math.Max(c.PerFrame, selfRecoilPerFrame(n))
			c.PerTrigger = math.Max(c.PerTrigger, selfRecoilPerTrigger(n))
		}
	}
	return c
}

func TestSourceWeaponRecoilUnchanged(t *testing.T) {
	if len(weaponCatalog) != len(sourceRecoilGolden) {
		t.Fatalf("source catalog has %d weapons, golden %d", len(weaponCatalog), len(sourceRecoilGolden))
	}
	for n, want := range sourceRecoilGolden {
		def := weaponCatalog[n]
		if def.ROF != want.ROF || def.Recoil != want.Recoil {
			t.Errorf("source weapon %d %s rof/recoil=%d/%.2f, ActionScript %d/%.2f", n, def.Name, def.ROF, def.Recoil, want.ROF, want.Recoil)
		}
	}
}

func TestAddedWeaponSelfRecoilWithinSourceClassCaps(t *testing.T) {
	pistol := sourceSelfRecoilCap(t, sourcePistolNumbers)
	smg := sourceSelfRecoilCap(t, sourceSMGNumbers)
	rifle := sourceSelfRecoilCap(t, sourceRifleNumbers)
	lmg := sourceSelfRecoilCap(t, sourceLMGNumbers, []int{sourceMinigunNumber})
	sniper := sourceSelfRecoilCap(t, sourceSniperNumbers)
	shotgun := sourceSelfRecoilCap(t, sourceShotgunNumbers)
	// Launchers and oddities have no source class: they may kick as hard as
	// the strongest single source shot (snipers / Anti Aircraft Gun, 10) and
	// push no harder over time than a source sniper or shotgun.
	heavy := selfRecoilCap{
		PerFrame:   math.Max(sniper.PerFrame, shotgun.PerFrame),
		PerTrigger: math.Max(sniper.PerTrigger, shotgun.PerTrigger),
	}

	// The caps themselves, from the ActionScript values above.
	for _, c := range []struct {
		name      string
		got, want selfRecoilCap
	}{
		{"pistol", pistol, selfRecoilCap{5.0 / 22, 5}},
		{"smg", smg, selfRecoilCap{.8 / 3, .8}},
		{"rifle", rifle, selfRecoilCap{2.0 / 7, 2}},
		{"lmg+minigun", lmg, selfRecoilCap{.8 / 3, .8}},
		{"sniper", sniper, selfRecoilCap{10.0 / 25, 10}},
		{"shotgun", shotgun, selfRecoilCap{10.0 / 25, 10}},
	} {
		if math.Abs(c.got.PerFrame-c.want.PerFrame) > 1e-9 || math.Abs(c.got.PerTrigger-c.want.PerTrigger) > 1e-9 {
			t.Fatalf("%s source cap %.3f/frame %.2f/trigger, want %.3f/%.2f", c.name, c.got.PerFrame, c.got.PerTrigger, c.want.PerFrame, c.want.PerTrigger)
		}
	}

	checked := 0
	for _, number := range allDevelopedWeaponNumbers() {
		spec, _ := developedWeaponSpec(number)
		def := developedWeaponCatalog[number]
		capRecoil := heavy
		switch spec.Category {
		case DevelopedCategoryPistol:
			capRecoil = pistol
		case DevelopedCategorySMG:
			capRecoil = smg
		case DevelopedCategoryRifle:
			capRecoil = rifle
		case DevelopedCategoryLMG:
			capRecoil = lmg
		case DevelopedCategorySniper:
			capRecoil = sniper
		case DevelopedCategoryShotgun:
			capRecoil = shotgun
		}
		if spec.Projectile == DevelopedProjectileFlame {
			capRecoil = lmg // a held stream, like a machine gun
		}
		// Automatic rifles and LMGs may kick as hard per shot as a source
		// rifle (2) when their cadence is slow; the sustained cap is what keeps
		// them from carrying the shooter away. Semi-auto rifles are marksman
		// guns and share the source sniper per-shot cap.
		perTriggerCap := capRecoil.PerTrigger
		if spec.Category == DevelopedCategoryLMG || spec.Category == DevelopedCategoryRifle {
			perTriggerCap = rifle.PerTrigger
			if spec.FireMode == DevelopedFireSemi {
				perTriggerCap = sniper.PerTrigger
			}
		}
		if got := selfRecoilPerFrame(number); got > capRecoil.PerFrame+1e-9 {
			t.Errorf("%d %s sustained self-recoil %.3f px/frame above its source class cap %.3f", number, def.Name, got, capRecoil.PerFrame)
		}
		if got := selfRecoilPerTrigger(number); got > perTriggerCap+1e-9 {
			t.Errorf("%d %s self-recoil %.2f per trigger above its source class cap %.2f", number, def.Name, got, perTriggerCap)
		}
		checked++
	}
	if checked != len(developedWeaponCatalog) || checked < 140 {
		t.Fatalf("checked %d of %d added weapons", checked, len(developedWeaponCatalog))
	}

	minigun := selfRecoilPerFrame(sourceMinigunNumber)
	for _, number := range []int{141, 142} { // MG42 and its MG3 descendant
		if got := selfRecoilPerFrame(number); got > minigun+1e-9 {
			t.Errorf("%s self-recoil %.3f px/frame above the source minigun %.3f", developedWeaponCatalog[number].Name, got, minigun)
		}
	}
}

// The firing code applies recoil once per accepted round: once for a whole
// shotgun load, once for each round of a burst, never per pellet.
func TestSelfRecoilAppliedOncePerRound(t *testing.T) {
	for _, number := range []int{15, 50, 55, 141, 165, 183, 128, 135, 198, 213, 219} {
		def, _ := WeaponByNumber(number)
		p := newShooter(number)
		var bullets []Bullet
		var shells []Shell
		var flashes []MuzzleFlash
		rounds := 0
		for frame := 0; frame < 120 && p.Weapon.Def.Number == number && p.Weapon.Bullets > 0; frame++ {
			before := p.VX
			ammo := p.Weapon.Bullets
			updateWeapon(p, &bullets, &shells, &flashes)
			spent := ammo - p.Weapon.Bullets
			if p.Weapon.Def.Number != number {
				spent = 1 // an emptied source gun is swapped on its last round
			}
			if got, want := before-p.VX, float64(spent)*def.Recoil; math.Abs(got-want) > 1e-9 {
				t.Fatalf("%s frame %d: self-push %.3f for %d round(s), want %.3f", def.Name, frame, got, spent, want)
			}
			rounds += spent
		}
		if rounds < 1 {
			t.Fatalf("%s never fired", def.Name)
		}
	}
}

// --- Self-recoil through the real frame order -------------------------------
//
// DefineSprite_697_player (and every playerAI variant) runs, in one
// onEnterFrame: _x += vx, platform tests, movement keys, the shooting branch
// (vx -= recoil * facing), then vx *= _root.friction (0.93) and the
// |vx| <= 0.1 snap. A shot is therefore damped by its own frame's friction:
// it leaves the shooter at recoil * 0.93, and from rest a lone shot slides
// about recoil * 0.93 / 0.07 px. The helpers below drive Player.UpdateWithFire
// with updateWeapon as the fire step, exactly as Game.Update does.

// recoilTestGround is a flat floor wide enough for any slide and inside the
// source death bounds (-600 < x < 1500).
func recoilTestGround() Map {
	return Map{Gravity: DefaultGravity, Platforms: []Rect{{X: -590, Y: 400, W: 2080, H: 40}}, LowestY: 2000}
}

// newGroundedShooter returns a trigger-held shooter standing still on the
// test floor, facing right (recoil pushes it left).
func newGroundedShooter(t *testing.T, number int, m Map) *Player {
	t.Helper()
	p := newShooter(number)
	p.X, p.Y = 450, 380
	p.AIShoot = false
	for i := 0; i < 40; i++ {
		p.Update(m, nil, nil)
	}
	if p.JumpNum != 2 || p.VX != 0 {
		t.Fatalf("weapon %d shooter did not settle on the floor: jump=%d vx=%v", number, p.JumpNum, p.VX)
	}
	p.Weapon = NewWeapon(number)
	p.Weapon.Frame = 1
	p.Weapon.WaitTime = p.Weapon.Def.ROF
	p.AIShoot = true
	return p
}

// recoilFrame runs one game frame for p the way Game.Update does: the weapon
// fires inside the player update, before friction. It reports rounds spent.
func recoilFrame(p *Player, m Map, bullets *[]Bullet) int {
	var shells []Shell
	var flashes []MuzzleFlash
	number := p.Weapon.Def.Number
	ammo := p.Weapon.Bullets
	p.UpdateWithFire(m, nil, nil, func() { updateWeapon(p, bullets, &shells, &flashes) })
	if p.Weapon.Def.Number != number {
		return 1 // an emptied source gun is swapped on its last round
	}
	return ammo - p.Weapon.Bullets
}

// selfRecoilSlide is how far the shooter is carried while holding the trigger
// for holdFrames and then until it stops (the |vx| <= 0.1 snap), plus the
// largest speed reached.
func selfRecoilSlide(t *testing.T, number, holdFrames int) (slide, topSpeed float64) {
	t.Helper()
	m := recoilTestGround()
	p := newGroundedShooter(t, number, m)
	startX := p.X
	var bullets []Bullet
	for frame := 0; frame < holdFrames+400; frame++ {
		if frame == holdFrames {
			p.AIShoot = false
		}
		recoilFrame(p, m, &bullets)
		topSpeed = math.Max(topSpeed, math.Abs(p.VX))
		if frame >= holdFrames && p.VX == 0 {
			break
		}
	}
	if !p.Active || p.JumpNum != 2 {
		t.Fatalf("weapon %d shooter left the floor during the slide test", number)
	}
	return startX - p.X, topSpeed
}

// selfRecoilSustained is the shooter's steady drift with the trigger held
// forever (px/frame, averaged over whole fire cycles once the start-up has
// died out); 60x this is the 1 s slide of sustained fire. Ammo is topped up and
// x re-centred each frame so the floor never runs out; neither touches vx.
func selfRecoilSustained(t *testing.T, number int) float64 {
	t.Helper()
	def, _ := WeaponByNumber(number)
	spec, developed := developedWeaponSpec(number)
	cycle := def.ROF
	if developed {
		if rounds := developedBurstRounds(spec); rounds > 1 {
			cycle += (rounds - 1) * developedBurstGap(spec)
		}
	}
	cycle = max(cycle, 1)
	m := recoilTestGround()
	p := newGroundedShooter(t, number, m)
	var bullets []Bullet
	const warmup = 300 // 0.93^300 ~ 3e-10 of the start-up transient is left
	window := cycle * ((600 + cycle - 1) / cycle)
	moved := 0.0
	for frame := 0; frame < warmup+window; frame++ {
		p.Weapon.Bullets = 9999
		bullets = bullets[:0]
		before := p.X
		recoilFrame(p, m, &bullets)
		if frame >= warmup {
			moved += before - p.X
		}
		p.X = 450
		if p.Weapon.Def.Number != number || !p.Active || p.JumpNum != 2 {
			t.Fatalf("weapon %d sustained-fire test lost its gun or the floor", number)
		}
	}
	return moved / float64(window)
}

// The source order, checked frame by frame against the ActionScript formula:
// vx = snap((vx - recoil*facing) * 0.93), with _x += vx at the start of the
// next frame. Covers a fast automatic and two snipers, grounded and airborne.
func TestSelfRecoilFollowsSourceFrameOrder(t *testing.T) {
	cases := []struct {
		number   int
		airborne bool
	}{
		{sourceMinigunNumber, false},
		{sourceMinigunNumber, true},
		{11, false}, // COOL SNIPER, recoil 10
		{11, true},
		{16, false},  // CLASSIC SNIPER, recoil 8
		{141, false}, // MG42
	}
	for _, c := range cases {
		def, _ := WeaponByNumber(c.number)
		m := recoilTestGround()
		var p *Player
		if c.airborne {
			m.Platforms = nil
			p = newShooter(c.number)
			p.X, p.Y = 450, -3000
		} else {
			p = newGroundedShooter(t, c.number, m)
		}
		var bullets []Bullet
		vx, x := p.VX, p.X
		shots := 0
		for frame := 0; frame < 90; frame++ {
			rounds := recoilFrame(p, m, &bullets)
			x += vx
			vx -= float64(rounds) * def.Recoil * float64(p.Facing)
			vx *= Friction
			if math.Abs(vx) <= 0.1 {
				vx = 0
			}
			if p.VX != vx || p.X != x {
				t.Fatalf("%s airborne=%v frame %d: vx=%.12f x=%.6f, ActionScript order gives vx=%.12f x=%.6f",
					def.Name, c.airborne, frame, p.VX, p.X, vx, x)
			}
			if c.airborne && p.JumpNum == 2 {
				t.Fatalf("%s airborne case landed at frame %d", def.Name, frame)
			}
			if rounds > 0 && shots == 0 {
				// First shot from rest: recoil, then this frame's friction.
				if want := -def.Recoil * Friction; math.Abs(p.VX-want) > 1e-12 {
					t.Fatalf("%s airborne=%v first shot vx=%.6f, want -recoil*0.93=%.6f", def.Name, c.airborne, p.VX, want)
				}
			}
			shots += rounds
		}
		if shots < 2 {
			t.Fatalf("%s fired only %d rounds", def.Name, shots)
		}
	}
}

// A single shot from rest slides recoil*0.93/(1-0.93) px, less the tail the
// |vx| <= 0.1 snap cuts off (under 0.1/(1-0.93) px).
func TestSingleShotSlideMatchesSourcePhysics(t *testing.T) {
	for _, number := range []int{1, 6, 11, 16, sourceMinigunNumber} {
		def, _ := WeaponByNumber(number)
		slide, top := selfRecoilSlide(t, number, 1)
		ideal := def.Recoil * Friction / (1 - Friction)
		if math.Abs(top-def.Recoil*Friction) > 1e-12 {
			t.Errorf("%s launch speed %.4f, want recoil*0.93 = %.4f", def.Name, top, def.Recoil*Friction)
		}
		if slide > ideal+1e-9 || slide < ideal-0.1/(1-Friction) {
			t.Errorf("%s single-shot slide %.2f px, want about %.2f px", def.Name, slide, ideal)
		}
	}
}

// The class caps of TestAddedWeaponSelfRecoilWithinSourceClassCaps measured in
// pixels through the corrected frame order: the sustained drift of held fire
// (shown as its 1 s slide) and the slide of one trigger pull. The caps are the
// strongest source weapon of each class, so they move with the physics; no
// added weapon may push its shooter further.
func TestAddedWeaponSelfRecoilSlideWithinSourceClassCaps(t *testing.T) {
	type slideCap struct{ Held, Trigger float64 } // px/frame sustained, px per trigger
	measure := func(number int) slideCap {
		trigger, _ := selfRecoilSlide(t, number, 1)
		return slideCap{selfRecoilSustained(t, number), trigger}
	}
	classCap := func(numbers ...[]int) slideCap {
		var c slideCap
		for _, list := range numbers {
			for _, n := range list {
				s := measure(n)
				c.Held = math.Max(c.Held, s.Held)
				c.Trigger = math.Max(c.Trigger, s.Trigger)
			}
		}
		return c
	}
	pistol := classCap(sourcePistolNumbers)
	smg := classCap(sourceSMGNumbers)
	rifle := classCap(sourceRifleNumbers)
	lmg := classCap(sourceLMGNumbers, []int{sourceMinigunNumber})
	sniper := classCap(sourceSniperNumbers)
	shotgun := classCap(sourceShotgunNumbers)
	heavy := slideCap{math.Max(sniper.Held, shotgun.Held), math.Max(sniper.Trigger, shotgun.Trigger)}

	// Under the source order a held weapon drifts at exactly
	// (recoil per frame) * 0.93/0.07, and one trigger pull of the class's
	// strongest shot slides about recoil * 0.93/0.07 px (the snap trims the
	// last < 0.1/0.07 px). The recoil-unit caps are pinned in
	// TestAddedWeaponSelfRecoilWithinSourceClassCaps.
	for _, c := range []struct {
		name              string
		got               slideCap
		perFrame, perShot float64
	}{
		{"pistol", pistol, 5.0 / 22, 5}, {"smg", smg, .8 / 3, .8}, {"rifle", rifle, 2.0 / 7, 2},
		{"lmg+minigun", lmg, .8 / 3, .8}, {"sniper", sniper, 10.0 / 25, 10}, {"shotgun", shotgun, 10.0 / 25, 10},
	} {
		gain := Friction / (1 - Friction)
		if want := c.perFrame * gain; math.Abs(c.got.Held-want) > 1e-6 {
			t.Fatalf("%s source sustained drift %.4f px/frame, want %.4f", c.name, c.got.Held, want)
		}
		ideal := c.perShot * gain
		if c.got.Trigger > ideal+1e-9 || c.got.Trigger < ideal-0.1/(1-Friction) {
			t.Fatalf("%s source trigger slide %.2f px, want about %.2f px", c.name, c.got.Trigger, ideal)
		}
	}
	t.Logf("source caps (1 s sustained slide px / one-trigger slide px): pistol %.1f/%.1f smg %.1f/%.1f rifle %.1f/%.1f lmg %.1f/%.1f sniper %.1f/%.1f shotgun %.1f/%.1f",
		60*pistol.Held, pistol.Trigger, 60*smg.Held, smg.Trigger, 60*rifle.Held, rifle.Trigger, 60*lmg.Held, lmg.Trigger, 60*sniper.Held, sniper.Trigger, 60*shotgun.Held, shotgun.Trigger)

	checked := 0
	for _, number := range allDevelopedWeaponNumbers() {
		spec, _ := developedWeaponSpec(number)
		def := developedWeaponCatalog[number]
		c := heavy
		switch spec.Category {
		case DevelopedCategoryPistol:
			c = pistol
		case DevelopedCategorySMG:
			c = smg
		case DevelopedCategoryRifle:
			c = rifle
		case DevelopedCategoryLMG:
			c = lmg
		case DevelopedCategorySniper:
			c = sniper
		case DevelopedCategoryShotgun:
			c = shotgun
		}
		if spec.Projectile == DevelopedProjectileFlame {
			c = lmg
		}
		triggerCap := c.Trigger
		if spec.Category == DevelopedCategoryLMG || spec.Category == DevelopedCategoryRifle {
			triggerCap = rifle.Trigger
			if spec.FireMode == DevelopedFireSemi {
				triggerCap = sniper.Trigger
			}
		}
		got := measure(number)
		if got.Held > c.Held+1e-9 {
			t.Errorf("%d %s sustained 1 s slide %.1f px above its source class cap %.1f", number, def.Name, 60*got.Held, 60*c.Held)
		}
		if got.Trigger > triggerCap+1e-9 {
			t.Errorf("%d %s one-trigger slide %.1f px above its source class cap %.1f", number, def.Name, got.Trigger, triggerCap)
		}
		checked++
	}
	if checked != len(developedWeaponCatalog) {
		t.Fatalf("checked %d of %d added weapons", checked, len(developedWeaponCatalog))
	}
}
