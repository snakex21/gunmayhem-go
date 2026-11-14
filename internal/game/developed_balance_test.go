package game

import (
	"math"
	"testing"
)

// Source weapons grouped by role, used to derive the push caps the added
// weapons must respect. Push is knockback per frame: Firepower / ROF.
var (
	sourceSMGNumbers   = []int{12, 19, 21, 22, 23, 24, 25, 26, 27, 28, 30, 31, 32, 67, 68, 69, 70, 71}
	sourceRifleNumbers = []int{10, 17, 20, 29, 56, 58, 59, 60, 61, 62, 63, 72, 73, 74, 75, 76}
	sourceLMGNumbers   = []int{57, 64, 65, 66}
)

const sourceMinigunNumber = 55

func sourceMaxPush(t *testing.T, numbers []int) float64 {
	t.Helper()
	best := 0.0
	for _, n := range numbers {
		def, ok := weaponCatalog[n]
		if !ok {
			t.Fatalf("source weapon %d missing", n)
		}
		best = math.Max(best, def.Firepower/float64(def.ROF))
	}
	return best
}

// developedCyclePush is the average knockback per frame of a developed weapon
// held down: every round of a burst, over the burst plus its pause.
func developedCyclePush(def WeaponDef, spec DevelopedWeaponSpec) float64 {
	rounds := developedBurstRounds(spec)
	pellets := 1
	if def.Shotgun > 0 {
		pellets = def.Shotgun
	}
	cycle := def.ROF
	if rounds > 1 {
		cycle += (rounds - 1) * developedBurstGap(spec)
	}
	return float64(rounds*pellets) * def.Firepower / float64(cycle)
}

func newShooter(number int) *Player {
	p := &Player{ID: 1, Active: true, AI: true, AIShoot: true, Facing: 1, MiniMulti: 1, PlayerScale: .8, DefaultWeapon: 1}
	p.Weapon = NewWeapon(number)
	p.Weapon.Frame = 1
	p.Weapon.WaitTime = p.Weapon.Def.ROF
	return p
}

func burstWeaponNumbers() []int {
	var out []int
	for _, n := range allDevelopedWeaponNumbers() {
		if spec, _ := developedWeaponSpec(n); developedBurstRounds(spec) > 1 {
			out = append(out, n)
		}
	}
	return out
}

func TestBurstWeaponsAreAllFound(t *testing.T) {
	got := burstWeaponNumbers()
	want := []int{128, 135, 198, 213, 219} // AN-94, FAMAS, Villar Perosa, G11, AF2011
	if len(got) != len(want) {
		t.Fatalf("burst weapons=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("burst weapons=%v want %v", got, want)
		}
	}
}

func TestBurstRoundsLeaveOnSeparateFrames(t *testing.T) {
	for _, number := range burstWeaponNumbers() {
		spec, _ := developedWeaponSpec(number)
		def := developedWeaponCatalog[number]
		rounds := developedBurstRounds(spec)
		gap := developedBurstGap(spec)
		if gap < 2 || gap > 3 {
			t.Fatalf("%s burst gap=%d want 2-3 frames", def.Name, gap)
		}
		if def.ROF <= gap {
			t.Fatalf("%s pause after burst=%d must be longer than the %d-frame gap", def.Name, def.ROF, gap)
		}

		p := newShooter(number)
		var bullets []Bullet
		var shells []Shell
		var flashes []MuzzleFlash
		var shotFrames []int
		for frame := 0; frame < 200 && p.Weapon.Def.Number == number && p.Weapon.Bullets > 0; frame++ {
			before := len(bullets)
			ammo := p.Weapon.Bullets
			updateWeapon(p, &bullets, &shells, &flashes)
			if fired := len(bullets) - before; fired > 0 {
				if fired != 1 {
					t.Fatalf("%s fired %d rounds on frame %d, want exactly one", def.Name, fired, frame)
				}
				if p.Weapon.Def.Number == number && p.Weapon.Bullets != ammo-1 {
					t.Fatalf("%s round on frame %d spent %d ammo", def.Name, frame, ammo-p.Weapon.Bullets)
				}
				shotFrames = append(shotFrames, frame)
			}
		}
		if len(shotFrames) < 2*rounds {
			t.Fatalf("%s fired only %d rounds", def.Name, len(shotFrames))
		}
		// Expected cadence: rounds-1 gaps of `gap`, then one pause of ROF.
		for i := 1; i < 2*rounds; i++ {
			want := gap
			if i%rounds == 0 {
				want = def.ROF
			}
			if got := shotFrames[i] - shotFrames[i-1]; got != want {
				t.Fatalf("%s interval %d = %d frames want %d (shots at %v)", def.Name, i, got, want, shotFrames[:2*rounds])
			}
		}
	}
}

func TestBurstContinuesAfterTriggerRelease(t *testing.T) {
	p := newShooter(135) // FAMAS
	p.AI = false
	p.RemoteControlled = true
	p.RemoteInput.Shoot = true
	var bullets []Bullet
	var shells []Shell
	var flashes []MuzzleFlash
	updateWeapon(p, &bullets, &shells, &flashes)
	p.RemoteInput.Shoot = false
	for i := 0; i < 20; i++ {
		updateWeapon(p, &bullets, &shells, &flashes)
	}
	if len(bullets) != 3 {
		t.Fatalf("FAMAS tap fired %d rounds want a full 3-round burst", len(bullets))
	}
}

func TestBurstStateRidesInWeaponState(t *testing.T) {
	// Network snapshots copy whole Player values (netplay makeNetSnapshot),
	// so burst progress must live in WeaponState, not in package state.
	p := newShooter(135)
	var bullets []Bullet
	var shells []Shell
	var flashes []MuzzleFlash
	updateWeapon(p, &bullets, &shells, &flashes)
	copyOfPlayer := *p
	if copyOfPlayer.Weapon.BurstLeft != 2 || copyOfPlayer.Weapon.BurstWait != 3 {
		t.Fatalf("burst state not in the copied player: %+v", copyOfPlayer.Weapon)
	}
	updateWeapon(&copyOfPlayer, &bullets, &shells, &flashes)
	updateWeapon(&copyOfPlayer, &bullets, &shells, &flashes)
	updateWeapon(&copyOfPlayer, &bullets, &shells, &flashes)
	if len(bullets) != 2 {
		t.Fatalf("copied player did not continue the burst: bullets=%d", len(bullets))
	}
}

func TestBurstPushNearOriginalAssaultRifle(t *testing.T) {
	rifleCap := sourceMaxPush(t, sourceRifleNumbers)
	for _, number := range []int{128, 135, 213} {
		def := developedWeaponCatalog[number]
		spec, _ := developedWeaponSpec(number)
		push := developedCyclePush(def, spec)
		if push < 4.4 || push > rifleCap {
			t.Fatalf("%s burst-cycle push=%.2f want inside the source rifle band 4.4..%.2f", def.Name, push, rifleCap)
		}
		if spec.Spread >= 2.5 {
			t.Fatalf("%s spread=%.1f, bursts should be tighter than full-auto rifles (2.5)", def.Name, spec.Spread)
		}
		if def.Firepower > 22 {
			t.Fatalf("%s per-round knockback %.0f should be below a full-auto rifle's 22", def.Name, def.Firepower)
		}
	}
}

func TestSemiAutoNeedsNewTriggerPress(t *testing.T) {
	semi := 0
	for _, number := range allDevelopedWeaponNumbers() {
		spec, _ := developedWeaponSpec(number)
		if !developedTriggerResetRequired(spec) {
			continue
		}
		semi++
		def := developedWeaponCatalog[number]
		if def.Bullets < 2 {
			continue // a one-round pickup is discarded after its only shot
		}
		p := newShooter(number)
		p.AI = false
		p.RemoteControlled = true
		p.RemoteInput.Shoot = true
		var bullets []Bullet
		var shells []Shell
		var flashes []MuzzleFlash
		perTrigger := developedBurstRounds(spec)
		if def.Shotgun > 0 {
			perTrigger = def.Shotgun
		}
		for i := 0; i < 3*def.ROF+10; i++ {
			updateWeapon(p, &bullets, &shells, &flashes)
		}
		if len(bullets) != perTrigger {
			t.Fatalf("%s held trigger fired %d projectiles, want one trigger's worth (%d)", def.Name, len(bullets), perTrigger)
		}
		p.RemoteInput.Shoot = false
		updateWeapon(p, &bullets, &shells, &flashes)
		p.RemoteInput.Shoot = true
		for i := 0; i < def.ROF+2 && len(bullets) == perTrigger; i++ {
			updateWeapon(p, &bullets, &shells, &flashes)
		}
		if len(bullets) <= perTrigger {
			t.Fatalf("%s did not fire again after the trigger was pressed again", def.Name)
		}
	}
	if semi < 60 {
		t.Fatalf("only %d semi-auto weapons found", semi)
	}
}

func TestAutomaticAndSourceWeaponsStillFireWhileHeld(t *testing.T) {
	for _, number := range []int{1, 10, 55, 124, 141} {
		p := newShooter(number)
		p.AI = false
		p.RemoteControlled = true
		p.RemoteInput.Shoot = true
		var bullets []Bullet
		var shells []Shell
		var flashes []MuzzleFlash
		for i := 0; i < 3*p.Weapon.Def.ROF+2; i++ {
			updateWeapon(p, &bullets, &shells, &flashes)
		}
		if len(bullets) < 2 {
			t.Fatalf("weapon %d fired %d bullets while held; hold-to-fire must keep working", number, len(bullets))
		}
	}
}

func TestAddedAutomaticsStayWithinClassPushCap(t *testing.T) {
	smgCap := sourceMaxPush(t, sourceSMGNumbers)
	rifleCap := sourceMaxPush(t, sourceRifleNumbers)
	lmgCap := sourceMaxPush(t, sourceLMGNumbers)
	minigun := weaponCatalog[sourceMinigunNumber]
	minigunPush := minigun.Firepower / float64(minigun.ROF)
	if lmgCap > minigunPush {
		t.Fatalf("source LMG cap %.2f above the minigun %.2f", lmgCap, minigunPush)
	}
	checked := 0
	for _, number := range allDevelopedWeaponNumbers() {
		spec, _ := developedWeaponSpec(number)
		if developedTriggerResetRequired(spec) || spec.ExplosionRadius > 0 {
			continue // semi-auto and explosive weapons are not sustained-fire guns
		}
		def := developedWeaponCatalog[number]
		capPush := rifleCap
		switch spec.Category {
		case DevelopedCategorySMG, DevelopedCategoryPistol:
			capPush = smgCap
		case DevelopedCategoryLMG:
			capPush = lmgCap
		}
		push := developedCyclePush(def, spec)
		if push > capPush+1e-9 {
			t.Errorf("%d %s sustained push %.2f px/frame exceeds its class cap %.2f", number, def.Name, push, capPush)
		}
		checked++
	}
	if checked < 60 {
		t.Fatalf("only %d added automatics checked", checked)
	}
}

func TestAddedLMGMagazinesMatchSourceClass(t *testing.T) {
	maxMag := 0
	for _, n := range sourceLMGNumbers {
		maxMag = max(maxMag, weaponCatalog[n].Bullets)
	}
	for _, number := range allDevelopedWeaponNumbers() {
		if spec, _ := developedWeaponSpec(number); spec.Category == DevelopedCategoryLMG {
			if def := developedWeaponCatalog[number]; def.Bullets > maxMag {
				t.Errorf("%s magazine %d above the largest source LMG magazine %d", def.Name, def.Bullets, maxMag)
			}
		}
	}
}

func TestMG42FireDelayAtLeastTwoFrames(t *testing.T) {
	mg42, _ := WeaponByNumber(141)
	if mg42.ROF < 2 {
		t.Fatalf("MG42 ROF=%d fires every frame", mg42.ROF)
	}
	for _, number := range allDevelopedWeaponNumbers() {
		spec, _ := developedWeaponSpec(number)
		if spec.Projectile == DevelopedProjectileFlame {
			continue // flame particles are emitted every tick by design
		}
		if def := developedWeaponCatalog[number]; def.ROF < developedMinFireDelay {
			t.Errorf("%s ROF=%d faster than the source minigun", def.Name, def.ROF)
		}
	}
}

func TestDevelopedFireDelaysFollowRealRateMapping(t *testing.T) {
	notMapped := map[int]bool{
		87: true, 88: true, 89: true, 90: true, 165: true, 168: true, 169: true, 170: true, 171: true,
		172: true, 173: true, 174: true, 175: true, 176: true, 177: true, 178: true, 179: true, 180: true,
		181: true, 182: true, 183: true, 184: true, 185: true, 187: true, 188: true, 189: true, 192: true, 193: true,
	}
	for _, number := range allDevelopedWeaponNumbers() {
		rate, ok := developedFireRates[number]
		if notMapped[number] {
			if ok {
				t.Errorf("weapon %d is listed as unmapped but has a rate", number)
			}
			continue
		}
		if !ok {
			t.Errorf("added firearm %d (%s) has no real-world rate", number, developedWeaponCatalog[number].Name)
			continue
		}
		def := developedWeaponCatalog[number]
		spec, _ := developedWeaponSpec(number)
		if rate.RPM > 0 {
			if want := developedFireDelayForRPM(rate.Basis, rate.RPM); def.ROF != want {
				t.Errorf("%s ROF=%d, %v rpm maps to %d", def.Name, def.ROF, rate.RPM, want)
			}
		}
		if rate.BurstRPM > 0 {
			if want := developedFireDelayForRPM(rate.Basis, rate.BurstRPM); spec.BurstGap != want {
				t.Errorf("%s burst gap=%d, %v rpm maps to %d", def.Name, spec.BurstGap, rate.BurstRPM, want)
			}
		}
	}
	// The calibration must put the source anchors on (or next to) their own
	// delays: AK-47 600 rpm and MAC-10 1090 rpm.
	if d := developedFireDelayForRPM(rateCyclicRifle, 600); d < 4 || d > 5 {
		t.Fatalf("600 rpm rifle maps to %d, source AK-47 uses 4", d)
	}
	if d := developedFireDelayForRPM(rateCyclicSMG, 1090); d < 2 || d > 3 {
		t.Fatalf("1090 rpm SMG maps to %d, source MAC-10 uses 2", d)
	}
}

// sustainedHeldPush simulates holding fire with an AI shooter (no trigger
// latch) and returns average knockback per frame, including reloads.
func sustainedHeldPush(number, frames int) float64 {
	p := newShooter(number)
	p.DefaultWeapon = number
	var bullets []Bullet
	var shells []Shell
	var flashes []MuzzleFlash
	for i := 0; i < frames; i++ {
		updateWeapon(p, &bullets, &shells, &flashes)
	}
	def, _ := WeaponByNumber(number)
	avg := def.Firepower
	if number < 87 {
		avg -= 0.5 // source FIREBULLET adds random(8)-4
	}
	return float64(len(bullets)) * avg / float64(frames)
}

func TestAddedStarterPistolsStayInSourceSustainedBand(t *testing.T) {
	sourceBest := 0.0
	for number := 1; number <= 6; number++ {
		sourceBest = math.Max(sourceBest, sustainedHeldPush(number, 3000))
	}
	if sourceBest < 1 || sourceBest > 2 {
		t.Fatalf("source starter sustained push %.2f outside the expected 1-2 band", sourceBest)
	}
	for _, number := range starterHandgunNumbers() {
		if number <= 6 {
			continue
		}
		if got := sustainedHeldPush(number, 3000); got > sourceBest*1.03 {
			t.Errorf("%s sustained push %.2f above the best source starter %.2f", developedWeaponCatalog[number].Name, got, sourceBest)
		}
	}
}

func TestCratePoolExcludesAddedPistolsAndPortalGun(t *testing.T) {
	pool := developedCrateWeapons()
	seen := map[int]bool{}
	for _, n := range pool {
		seen[n] = true
		if spec, ok := developedWeaponSpec(n); ok && (spec.Category == DevelopedCategoryPistol || spec.Projectile == DevelopedProjectilePortal) {
			t.Fatalf("crate pool still contains %d (%s)", n, developedWeaponCatalog[n].Name)
		}
	}
	if seen[189] {
		t.Fatal("Portal Gun still drops from crates")
	}
	// Still reachable elsewhere.
	if !isStarterHandgunNumber(99) || !isStarterHandgunNumber(231) {
		t.Fatal("added pistols are no longer starter handguns")
	}
	if !customWeaponAssetAvailable(189) {
		t.Fatal("Portal Gun disappeared from the defined/library weapons")
	}
}

func TestCrateWeightingOriginalsTwiceAdded(t *testing.T) {
	pool := developedCrateWeapons()
	originals, added := 0, 0
	for _, n := range pool {
		if n <= 86 {
			originals++
		} else {
			added++
		}
	}
	if originals == 0 || added == 0 {
		t.Fatalf("crate pool has %d originals and %d added weapons", originals, added)
	}
	counts := map[int]int{}
	const draws = 400000
	for i := 0; i < draws; i++ {
		counts[pickCrateWeapon(pool)]++
	}
	origDraws, addedDraws := 0, 0
	for n, c := range counts {
		if n <= 86 {
			origDraws += c
		} else {
			addedDraws += c
		}
	}
	perOriginal := float64(origDraws) / float64(originals)
	perAdded := float64(addedDraws) / float64(added)
	ratio := perOriginal / perAdded
	if ratio < 1.9 || ratio > 2.1 {
		t.Fatalf("per-weapon crate ratio original:added = %.3f want 2", ratio)
	}
	wantShare := float64(2*originals) / float64(2*originals+added)
	if share := float64(origDraws) / draws; math.Abs(share-wantShare) > 0.01 {
		t.Fatalf("original crate share %.3f want %.3f", share, wantShare)
	}
	// A source-only pool keeps the plain uniform source draw.
	for i := 0; i < 1000; i++ {
		if n := pickCrateWeapon(DefaultCrateWeapons()); n < 10 || n > 66 {
			t.Fatalf("source pool drew %d", n)
		}
	}
}

func TestBARVariantsAreDistinct(t *testing.T) {
	a2, _ := WeaponByNumber(145)
	wwi, _ := WeaponByNumber(203)
	if a2.Name == wwi.Name || a2.ROF == wwi.ROF || a2.Firepower == wwi.Firepower {
		t.Fatalf("BAR 145 %+v and 203 %+v are still duplicates", a2, wwi)
	}
}

func TestBoltAndFlareWeaponsHitStandingTargets(t *testing.T) {
	arena := Map{Gravity: DefaultGravity, Platforms: []Rect{{X: -2000, Y: 400, W: 4000, H: 40}}}
	for _, number := range []int{177, 178, 179, 180, 181} {
		for _, distance := range []float64{150, 300, 500} {
			for _, facing := range []int{1, -1} {
				for trial := 0; trial < 12; trial++ {
					shooter := newShooter(number)
					shooter.X, shooter.Y, shooter.Facing = 0, 400, facing
					target := &Player{ID: 2, Team: 2, Active: true, X: distance * float64(facing), Y: 400, PlayerScale: .8, DamageMulti: 1}
					var bullets []Bullet
					var shells []Shell
					var flashes []MuzzleFlash
					updateWeapon(shooter, &bullets, &shells, &flashes)
					if len(bullets) != 1 {
						t.Fatalf("weapon %d fired %d projectiles", number, len(bullets))
					}
					players := []*Player{shooter, target}
					for f := 0; f < 120 && !bullets[0].Dead; f++ {
						updateBullets(arena, bullets, players, nil, nil, nil, nil)
					}
					if target.HitNumber == 0 && target.BurnTime == 0 {
						t.Fatalf("%s missed a standing target %v px away (facing %d, trial %d)", developedWeaponCatalog[number].Name, distance, facing, trial)
					}
				}
			}
		}
	}
}

func TestBowsAndHarpoonHaveNoMuzzleFlash(t *testing.T) {
	for _, number := range []int{177, 178, 179} {
		p := newShooter(number)
		var bullets []Bullet
		var shells []Shell
		var flashes []MuzzleFlash
		updateWeapon(p, &bullets, &shells, &flashes)
		if len(bullets) != 1 || len(flashes) != 0 {
			t.Fatalf("%s bullets=%d flashes=%d want 1 bolt and no gunpowder flash", developedWeaponCatalog[number].Name, len(bullets), len(flashes))
		}
	}
}
