package game

import (
	"math"
	"testing"
)

// Regression tests for the arsenal2 review findings.

func TestHarpoonedDroneRopeDecays(t *testing.T) {
	g, _ := enemyTestGame()
	d := spawnTestEnemy(g, campaignEnemyDrone, 0)
	d.Body.HarpoonTether, d.Body.HarpoonTetherID = harpoonTetherFrames, 1
	for i := 0; i < harpoonTetherFrames+1; i++ {
		g.updateCampaignEnemies(g.arena)
	}
	if d.Body.HarpoonTether != 0 || d.Body.HarpoonTetherID != 0 {
		t.Fatalf("drone rope never decays: %d / %d", d.Body.HarpoonTether, d.Body.HarpoonTetherID)
	}
}

func TestBatAndBlowerRespectShieldBearerPlate(t *testing.T) {
	g, human := enemyTestGame()
	e := spawnTestEnemy(g, campaignEnemyShieldBearer, 5)
	human.EquipWeapon(weaponBaseballBat)
	human.Facing = 1
	human.X, human.VX, human.VY = 400, 0, 0
	e.Body.X, e.Body.Y, e.Body.VX, e.Body.VY = 440, human.Y, 0, 0
	e.Body.Facing = -1 // plate toward the batter
	e.ShieldDown = 0
	hp, hits := e.HP, e.Body.HitNumber
	human.MeleeHitPending = true
	targets := g.hitTargets()
	g.beginCampaignEnemyHitPhase()
	w := beginAbilityKnockbackWindow(targets)
	g.applyArsenal2Hits(targets)
	endAbilityKnockbackWindow(targets, w, true)
	g.endCampaignEnemyHitPhase(g.arena)
	scale := abilityKnockbackScale(&e.Body)
	if want := batPush * batShieldScale * scale; math.Abs(e.Body.VX-want) > 1e-6 {
		t.Fatalf("bat into the plate vx=%.3f want %.3f", e.Body.VX, want)
	}
	if e.Body.HitNumber != hits || e.HP != hp {
		t.Fatalf("a blocked swing counted as a hit: hits %d->%d hp %d->%d", hits, e.Body.HitNumber, hp, e.HP)
	}
	// Blower into the plate: reduced, and wind is never a hit.
	human.EquipWeapon(weaponLeafBlower)
	e.Body.X, e.Body.VX, e.Body.VY = 480, 0, 0
	e.HitFlash = 0
	human.BlowerPulse = true
	targets = g.hitTargets()
	g.beginCampaignEnemyHitPhase()
	w = beginAbilityKnockbackWindow(targets)
	g.applyArsenal2Hits(targets)
	endAbilityKnockbackWindow(targets, w, true)
	g.endCampaignEnemyHitPhase(g.arena)
	if e.Body.VX <= 0 || e.Body.VX > blowerPushNear*batShieldScale*scale+1e-6 {
		t.Fatalf("blower into the plate vx=%.3f", e.Body.VX)
	}
	if e.HitFlash != 0 || e.HP != hp {
		t.Fatalf("wind flashed / hurt a campaign enemy: flash=%d hp=%d", e.HitFlash, e.HP)
	}
}

func TestBatAndBlowerNeverActThroughPlatforms(t *testing.T) {
	wall := Rect{X: 245, Y: 300, W: 10, H: 120}
	blower := arsenal2Player(1, 200, weaponLeafBlower)
	target := arsenal2Player(2, 300, 1)
	g := arsenal2Game(blower, target)
	g.arena.Platforms = append(g.arena.Platforms, wall)
	g.grenades = []Grenade{{X: 290, Y: 364, VX: -2, OwnerID: 2}}
	fireOnce(g, blower)
	hitPhase(g)
	if target.VX != 0 || g.grenades[0].VX != -2 {
		t.Fatalf("blower through a wall: target %.2f grenade %.2f", target.VX, g.grenades[0].VX)
	}
	blower.AI = true
	aiDecideBlower(blower, g, target)
	if blower.AIShoot {
		t.Fatal("blower bot blows at targets behind a wall")
	}
	bat := arsenal2Player(3, 220, weaponBaseballBat)
	near := arsenal2Player(4, 270, 1)
	g2 := arsenal2Game(bat, near)
	g2.arena.Platforms = append(g2.arena.Platforms, wall)
	fireOnce(g2, bat)
	hitPhase(g2)
	if near.VX != 0 {
		t.Fatalf("bat through a wall: %.2f", near.VX)
	}
}

func TestOrdinaryBotsKeepSourceRangeAndEdgeLimit(t *testing.T) {
	for _, n := range []int{1, 10, 13, 55, 87, 124} {
		p := arsenal2Player(1, 300, n)
		if aiPreferredRange(p) != 150 || aiEdgeLimit(p) != 200 {
			t.Fatalf("weapon %d: range %.0f edge %.0f", n, aiPreferredRange(p), aiEdgeLimit(p))
		}
	}
	shield := arsenal2Player(1, 300, 1)
	shield.AbilityNumber = abilityBallisticShield
	if aiPreferredRange(shield) != 100 || aiEdgeLimit(shield) != 200 {
		t.Fatal("shield bot range")
	}
	shield.EquipWeapon(10) // slung shield: back to the source range
	if aiPreferredRange(shield) != 150 {
		t.Fatal("slung shield changed the range")
	}
}

func TestBallisticShieldDoesNotStopInstagib(t *testing.T) {
	p := arsenal2Player(1, 400, 1)
	p.AbilityNumber = abilityBallisticShield
	b := Bullet{Kind: BulletInstagib, X: p.X, Y: p.Y - 30, VX: -25, Facing: -1, Firepower: 25, OwnerID: 9, Team: 9}
	var trails []InstagibTrailEffect
	b.X += 25 // updateInstagibBullet moves first
	updateInstagibBullet(Map{}, &b, []*Player{p}, &trails)
	if !p.InstagibPlaying {
		t.Fatal("the shield stopped an instagib kill")
	}
}

func TestDoubleKnockbackScalesBatBlowerAndReel(t *testing.T) {
	run := func(double bool, weapon int, victimX float64) *Player {
		p := arsenal2Player(1, 200, weapon)
		v := arsenal2Player(2, victimX, 1)
		g := arsenal2Game(p, v)
		if double {
			g.customMatchActive = true
			g.matchModifiers = map[int]bool{customModifierDoubleKnockback: true}
		}
		fireOnce(g, p)
		if double {
			for i := range g.bullets {
				g.bullets[i].Firepower *= 2 // updatePlayerWeapon's modifier step
			}
		}
		for i := 0; i < 40 && v.VX == 0; i++ {
			hitPhase(g)
		}
		return v
	}
	for _, tc := range []struct {
		weapon int
		x      float64
	}{{weaponBaseballBat, 245}, {weaponLeafBlower, 300}, {weaponHarpoonGun, 450}} {
		one, two := run(false, tc.weapon, tc.x), run(true, tc.weapon, tc.x)
		if one.VX == 0 || math.Abs(two.VX-2*one.VX) > 1e-6 {
			t.Fatalf("weapon %d: x1 %.3f x2 %.3f", tc.weapon, one.VX, two.VX)
		}
	}
}

func TestAkimboHandResetsAndSurvivesRealTimelineThroughReload(t *testing.T) {
	p := arsenal2Player(1, 300, 1)
	p.AkimboTime, p.AkimboHand = akimboFrames, 1
	p.EquipWeapon(2)
	if p.AkimboHand != 0 {
		t.Fatal("equip kept the off-hand turn")
	}
	p.AkimboTime = 0
	p.AkimboHand = 1
	p.tickArsenal2Timers()
	if p.AkimboHand != 0 {
		t.Fatal("hand state kept without akimbo")
	}
	// Hold the trigger through the real source timeline (shots, shells and a
	// full reload) with and without akimbo.
	count := func(akimbo bool) (shots, shells int, reloaded bool) {
		q := arsenal2Player(1, 300, 1)
		q.Weapon = NewWeapon(1)
		if akimbo {
			q.AkimboTime = akimboFrames
		}
		var bullets []Bullet
		var sh []Shell
		var fl []MuzzleFlash
		q.AI, q.AIShoot = true, true
		for i := 0; i < 240; i++ {
			updateWeapon(q, &bullets, &sh, &fl)
			if q.Weapon.Frame >= 9 {
				reloaded = true
			}
		}
		return q.ShotsFired, len(sh), reloaded
	}
	normal, _, normalReload := count(false)
	dual, shells, dualReload := count(true)
	if !normalReload || !dualReload {
		t.Fatalf("no reload seen: normal=%v akimbo=%v", normalReload, dualReload)
	}
	if dual < normal*3/2 {
		t.Fatalf("akimbo fired %d shots vs %d", dual, normal)
	}
	if shells < dual/2 {
		t.Fatalf("akimbo ejected %d shells for %d shots", shells, dual)
	}
}

func TestShooterDeathCutsHarpoonRopes(t *testing.T) {
	shooter := arsenal2Player(1, 200, weaponHarpoonGun)
	victim := arsenal2Player(2, 500, 1)
	victim.HarpoonTether, victim.HarpoonTetherID = 5, 1
	g := arsenal2Game(shooter, victim)
	g.bullets = []Bullet{{DevelopedProjectile: DevelopedProjectileHarpoon, ShooterID: 1, OwnerID: 1, X: 300, Y: 364, VX: 22, Facing: 1, Firepower: harpoonDirectPush, Lifetime: 140}}
	g.dropHarpoonRopesOf(1)
	if victim.HarpoonTether != 0 || victim.HarpoonTetherID != 0 || g.bullets[0].ShooterID != -1 {
		t.Fatalf("rope not cut: %+v / bullet shooter %d", victim, g.bullets[0].ShooterID)
	}
	if harpoonShooter(&g.bullets[0], g.players) != nil {
		t.Fatal("a cut harpoon still reels")
	}
}

func TestZombieBulletAndWindSameTick(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	holdZombieWave(g)
	p := g.players[0]
	placePlayer(p, 200, 511)
	p.EquipWeapon(weaponLeafBlower)
	p.Facing = 1
	p.RemoteInput.Shoot = true
	z := spawnTestZombie(g, zombieWalker, 300, 511, 100)
	g.bullets = append(g.bullets, testBullet(270, 511-20, 1, 20))
	updateN(t, g, 1)
	if z.HP != 80 {
		t.Fatalf("bullet + wind: HP=%d want 80 (wind deals no damage)", z.HP)
	}
}
