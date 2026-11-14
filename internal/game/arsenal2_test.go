package game

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

// arsenal2Arena: one wide floor and a separate right-hand platform with a
// gap between them.
func arsenal2Arena() Map {
	return Map{Gravity: DefaultGravity, Platforms: []Rect{
		{X: 0, Y: 400, W: 500, H: 40},
		{X: 640, Y: 400, W: 300, H: 40},
	}, LowestY: 600}
}

func arsenal2Player(id int, x float64, weapon int) *Player {
	p := &Player{ID: id, Team: id, Active: true, X: x, Y: 400, Facing: 1, JumpNum: 2,
		PlayerScale: .8, MiniMulti: 1, DamageMulti: 1, FirepowerMulti: 1, Weight: 1, Alpha: 1,
		Lives: 5, DefaultWeapon: 1}
	p.EquipWeapon(weapon)
	p.Weapon.Frame = 1
	p.Weapon.WaitTime = p.Weapon.Def.ROF
	return p
}

func arsenal2Game(players ...*Player) *Game {
	return &Game{arena: arsenal2Arena(), players: players, seenDeaths: map[int]int{}}
}

// fireOnce pulls the trigger of p for one weapon step.
func fireOnce(g *Game, p *Player) {
	p.AI, p.AIShoot = true, true
	updateWeapon(p, &g.bullets, &g.shells, &g.flashes)
	p.AIShoot = false
}

// hitPhase runs the arsenal2 + bullet hit phase like simulateArenaTick.
func hitPhase(g *Game) {
	targets := g.hitTargets()
	w := beginAbilityKnockbackWindow(targets)
	g.applyArsenal2Hits(targets)
	updateBullets(g.arena, g.bullets, targets, &g.instagibTrails, &g.explosions, &g.blastFX, &g.portals)
	endAbilityKnockbackWindow(targets, w, true)
	g.bullets = compactBullets(g.bullets)
}

// slide integrates a free horizontal slide (friction only).
func slide(vx float64) float64 {
	total := 0.0
	for math.Abs(vx) > 0.1 {
		total += vx
		vx *= Friction
	}
	return total
}

func TestArsenal2WeaponHandedness(t *testing.T) {
	oneHanded := []int{1, 2, 3, 4, 5, 6, 8, 98, 99, 111, 190, 219, 231, 181, weaponBaseballBat}
	for _, n := range oneHanded {
		if !weaponOneHanded(n) {
			t.Errorf("weapon %d (%s) should be one-handed", n, weaponDisplayName(n))
		}
	}
	twoHanded := []int{9, 10, 11, 13, 19, 55, 68, 86, 87, 96, 116, 141, 165, 180, weaponHarpoonGun, 187, 189, weaponLeafBlower}
	for _, n := range twoHanded {
		if weaponOneHanded(n) {
			t.Errorf("weapon %d (%s) should be two-handed", n, weaponDisplayName(n))
		}
	}
	// Every weapon in the game is classified, and every pistol category
	// weapon is one-handed.
	for _, n := range organizedGunLibraryNumbers(0) {
		spec, dev := developedWeaponSpec(n)
		if dev && spec.Category == DevelopedCategoryPistol && !weaponOneHanded(n) {
			t.Errorf("developed pistol %d is two-handed", n)
		}
		if !dev && n > 8 && weaponOneHanded(n) {
			t.Errorf("source long gun %d is one-handed", n)
		}
	}
}

func TestArsenal2CatalogEntries(t *testing.T) {
	bat, _ := WeaponByNumber(weaponBaseballBat)
	blower, _ := WeaponByNumber(weaponLeafBlower)
	harpoon, _ := WeaponByNumber(weaponHarpoonGun)
	if bat.ROF != batSwingCooldown || bat.Firepower != batPush || bat.Bullets != batPickupSwings {
		t.Fatalf("bat %+v", bat)
	}
	if blower.ROF != 1 || blower.Bullets != blowerFuelTicks {
		t.Fatalf("blower %+v", blower)
	}
	if harpoon.Bullets != 3 || harpoon.ROF != 40 {
		t.Fatalf("harpoon %+v", harpoon)
	}
	if !isStarterHandgunNumber(weaponBaseballBat) || isStarterHandgunNumber(weaponLeafBlower) {
		t.Fatal("the bat is a starter option, the blower is not")
	}
	pool := map[int]bool{}
	for _, n := range developedCrateWeapons() {
		pool[n] = true
	}
	for _, n := range []int{weaponBaseballBat, weaponLeafBlower, weaponHarpoonGun} {
		if !pool[n] {
			t.Errorf("weapon %d missing from the crate pool", n)
		}
		if weaponArsenal2Description(n) == "" || weaponStatsLine(n) == "" {
			t.Errorf("weapon %d has no description", n)
		}
	}
	for _, n := range []int{weaponBaseballBat, weaponLeafBlower} {
		r := proceduralWeaponRaster(n)
		if r == nil || r.Image == nil {
			t.Fatalf("weapon %d has no procedural art", n)
		}
		grip := customWeaponGripOrigins[n]
		if math.Abs(-r.Bounds.X/r.Bounds.W-grip[0]) > 0.01 || math.Abs(-r.Bounds.Y/r.Bounds.H-grip[1]) > 0.01 {
			t.Errorf("weapon %d grip table %v does not match the art origin %v", n, grip, r.Bounds)
		}
	}
	if got := sourceGunSound(weaponBaseballBat); got != "dev_bat.wav" || developedSoundPCM(got) == nil {
		t.Fatalf("bat sound %q", got)
	}
	if got := sourceGunSound(weaponLeafBlower); got != "dev_blower.wav" || developedSoundPCM(got) == nil {
		t.Fatalf("blower sound %q", got)
	}
}

func TestHarpoonPullMath(t *testing.T) {
	if harpoonPullImpulse(harpoonStopPx) != 0 || harpoonPullImpulse(30) != 0 {
		t.Fatal("no reel inside the stop distance")
	}
	if got := harpoonPullImpulse(100); got != harpoonPullMin {
		t.Fatalf("short pull %.2f", got)
	}
	if got := harpoonPullImpulse(2000); got != harpoonPullMax {
		t.Fatalf("long pull %.2f", got)
	}
	// In range the reel slides the victim to about harpoonStopPx from the
	// shooter.
	dist := 300.0
	v := harpoonPullImpulse(dist)
	if left := dist - slide(v); math.Abs(left-harpoonStopPx) > 6 {
		t.Fatalf("pull from %.0f px leaves the victim %.1f px away", dist, left)
	}
}

func TestHarpoonPullsVictimTowardShooterAcrossGap(t *testing.T) {
	shooter := arsenal2Player(1, 380, weaponHarpoonGun)
	victim := arsenal2Player(2, 700, 1)
	victim.Facing = -1
	g := arsenal2Game(shooter, victim)
	fireOnce(g, shooter)
	if len(g.bullets) != 1 || g.bullets[0].DevelopedProjectile != DevelopedProjectileHarpoon || g.bullets[0].ShooterID != 1 {
		t.Fatalf("harpoon shot %+v", g.bullets)
	}
	if shooter.Weapon.Bullets != 2 {
		t.Fatalf("ammo after one hook %d", shooter.Weapon.Bullets)
	}
	for i := 0; i < 40 && victim.HitNumber == 0; i++ {
		hitPhase(g)
	}
	if victim.HitNumber == 0 {
		t.Fatal("harpoon never hooked the victim")
	}
	if victim.VX >= -harpoonPullMin || victim.VY >= 0 {
		t.Fatalf("victim not reeled in: vx=%.2f vy=%.2f", victim.VX, victim.VY)
	}
	if victim.HarpoonTether != harpoonTetherFrames || victim.HarpoonTetherID != shooter.ID || victim.LastHitBy != shooter.ID {
		t.Fatalf("rope state %+v", victim)
	}
	// It slides back over the gap (x 500..640) and never past the shooter.
	end := victim.X + slide(victim.VX)
	if end > 500 || end < shooter.X {
		t.Fatalf("reel ends at x=%.1f (shooter %.1f, gap 500..640)", end, shooter.X)
	}
}

func TestHarpoonRespectsAnchorAndHooksBallisticShield(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prepare func(v *Player)
		pulled  bool
	}{
		{"anchor", func(v *Player) { v.AbilityNumber = abilityAnchor; v.AnchorTime = 5 }, false},
		{"ballistic shield front", func(v *Player) { v.AbilityNumber = abilityBallisticShield }, true},
	} {
		shooter := arsenal2Player(1, 200, weaponHarpoonGun)
		victim := arsenal2Player(2, 450, 1)
		victim.Facing = -1
		tc.prepare(victim)
		g := arsenal2Game(shooter, victim)
		fireOnce(g, shooter)
		for i := 0; i < 30 && victim.HitNumber == 0; i++ {
			hitPhase(g)
		}
		if pulled := victim.VX < -3; pulled != tc.pulled {
			t.Fatalf("%s: pulled=%v vx=%.2f", tc.name, pulled, victim.VX)
		}
	}
}

func TestHarpoonCannotHookThroughPlatform(t *testing.T) {
	shooter := arsenal2Player(1, 100, weaponHarpoonGun)
	victim := arsenal2Player(2, 450, 1)
	g := arsenal2Game(shooter, victim)
	g.arena.Platforms = append(g.arena.Platforms, Rect{X: 260, Y: 300, W: 20, H: 120}) // a wall
	fireOnce(g, shooter)
	for i := 0; i < 40; i++ {
		hitPhase(g)
	}
	if victim.HitNumber != 0 || victim.VX != 0 {
		t.Fatalf("hooked through a wall: %+v", victim)
	}
}

func TestLeafBlowerConePushAndFuel(t *testing.T) {
	blower := arsenal2Player(1, 200, weaponLeafBlower)
	near := arsenal2Player(2, 290, 1)
	far := arsenal2Player(3, 200+52+blowerRange+60, 1)
	behind := arsenal2Player(4, 120, 1)
	shielded := arsenal2Player(5, 330, 1)
	shielded.AbilityNumber = abilityShield // one-hit shield: wind must not spend it
	g := arsenal2Game(blower, near, far, behind, shielded)
	fireOnce(g, blower)
	hitPhase(g)
	if near.VX <= blowerPushFar || near.VX > blowerPushNear+0.01 {
		t.Fatalf("near target push %.3f", near.VX)
	}
	if near.HitNumber != 0 || near.LastHitBy != blower.ID {
		t.Fatalf("wind is not a hit but credits a ring-out: hits=%d last=%d", near.HitNumber, near.LastHitBy)
	}
	if far.VX != 0 || behind.VX != 0 {
		t.Fatalf("outside the cone pushed: far=%.2f behind=%.2f", far.VX, behind.VX)
	}
	if shielded.AbilityShieldUsed || shielded.VX <= 0 {
		t.Fatalf("wind spent the one-hit shield or did not push: used=%v vx=%.2f", shielded.AbilityShieldUsed, shielded.VX)
	}
	// Holding it for a second beats walking (0.64/tick) up close.
	if blowerPushAt(0.8) <= 0.64 {
		t.Fatal("blower weaker than walking at close range")
	}
	// Fuel: one tick per round; empty drops back to the starter.
	if blower.Weapon.Bullets != blowerFuelTicks-1 {
		t.Fatalf("fuel %d", blower.Weapon.Bullets)
	}
	blower.Weapon.Bullets = 1
	fireOnce(g, blower)
	if blower.Weapon.Def.Number != blower.DefaultWeapon {
		t.Fatalf("empty blower kept: %d", blower.Weapon.Def.Number)
	}
}

func TestLeafBlowerDeflectsThrowablesAndRockets(t *testing.T) {
	blower := arsenal2Player(1, 200, weaponLeafBlower)
	enemy := arsenal2Player(2, 800, 1)
	g := arsenal2Game(blower, enemy)
	g.grenades = []Grenade{{X: 300, Y: 364, VX: -3, VY: 0, OwnerID: 2, Fuse: 10}}
	g.bullets = []Bullet{{X: 330, Y: 364, VX: -17, Facing: -1, OwnerID: 2, Team: 2, DevelopedProjectile: DevelopedProjectileRocket,
		ExplosionRadius: 190, ExplosionPower: 120, Lifetime: 180, Alpha: 1}}
	fireOnce(g, blower)
	targets := g.hitTargets()
	g.applyArsenal2Hits(targets)
	if g.grenades[0].VX <= -3+blowerGrenadePush*0.4 || g.grenades[0].VY >= 0 {
		t.Fatalf("grenade not blown away: %+v", g.grenades[0])
	}
	r := g.bullets[0]
	if r.VX <= 0 || r.OwnerID != blower.ID || r.Team != blower.Team || !r.GuidanceLost {
		t.Fatalf("rocket not turned around: %+v", r)
	}
}

func TestBaseballBatSwingHitAndCooldown(t *testing.T) {
	bat := arsenal2Player(1, 200, weaponBaseballBat)
	bat.DefaultWeapon = weaponBaseballBat
	target := arsenal2Player(2, 245, 1)
	out := arsenal2Player(3, 320, 1)
	g := arsenal2Game(bat, target, out)
	fireOnce(g, bat)
	if bat.MeleeSwing != batSwingFrames || !bat.MeleeHitPending || len(g.bullets) != 0 || len(g.flashes) != 0 {
		t.Fatalf("swing state %+v bullets=%d flashes=%d", bat, len(g.bullets), len(g.flashes))
	}
	hitPhase(g)
	if math.Abs(target.VX-batPush) > 1e-6 || target.VY != -batLift || target.HitNumber != 1 || target.LastHitBy != bat.ID {
		t.Fatalf("bat hit vx=%.2f vy=%.2f hits=%d", target.VX, target.VY, target.HitNumber)
	}
	if out.VX != 0 {
		t.Fatal("bat reached 120 px")
	}
	if bat.HitsLanded != 1 {
		t.Fatalf("hits landed %d", bat.HitsLanded)
	}
	// Cooldown: no new swing before batSwingCooldown steps.
	swings := bat.ShotsFired
	for i := 0; i < batSwingCooldown-1; i++ {
		fireOnce(g, bat)
	}
	if bat.ShotsFired != swings {
		t.Fatalf("swung again inside the cooldown")
	}
	fireOnce(g, bat)
	if bat.ShotsFired != swings+1 {
		t.Fatalf("no swing after the cooldown")
	}
	// A starter bat never breaks; a crate bat lasts batPickupSwings swings.
	if bat.Weapon.Bullets != batPickupSwings {
		t.Fatalf("starter bat lost durability: %d", bat.Weapon.Bullets)
	}
	crate := arsenal2Player(4, 600, weaponBaseballBat)
	for i := 0; i < batPickupSwings; i++ {
		crate.Weapon.WaitTime = crate.Weapon.Def.ROF
		fireOnce(g, crate)
	}
	if crate.Weapon.Def.Number != crate.DefaultWeapon {
		t.Fatalf("crate bat did not break after %d swings", batPickupSwings)
	}
}

func TestBaseballBatHitsGrenadesBackAndIsHalvedByShield(t *testing.T) {
	bat := arsenal2Player(1, 200, weaponBaseballBat)
	target := arsenal2Player(2, 245, 1)
	target.Facing = -1
	target.AbilityNumber = abilityBallisticShield
	g := arsenal2Game(bat, target)
	g.grenades = []Grenade{{X: 230, Y: 360, VX: -4, VY: 2, OwnerID: 2, Fuse: 20}}
	fireOnce(g, bat)
	hitPhase(g)
	if want := batPush * batShieldScale; math.Abs(target.VX-want) > 1e-6 {
		t.Fatalf("bat into a raised shield vx=%.2f want %.2f", target.VX, want)
	}
	if gr := g.grenades[0]; gr.VX != batGrenadeHitSpeed || gr.VY != -batGrenadeHitLift || gr.OwnerID != bat.ID {
		t.Fatalf("grenade not batted back: %+v", gr)
	}
}

func TestBallisticShieldFrontBackTopAndExplosions(t *testing.T) {
	bullet := func(vx, vy float64) Bullet {
		f := 1
		if vx < 0 {
			f = -1
		}
		return Bullet{Kind: BulletNormal, VX: vx, VY: vy, Firepower: 40, OwnerID: 9, Team: 9, Facing: f, Alpha: 1}
	}
	hit := func(holder *Player, b Bullet) float64 {
		b.X, b.Y = holder.X, holder.Y-30
		holder.VX = 0
		hitPlayerWithBullet(Map{}, &b, []*Player{holder}, false)
		return math.Abs(holder.VX)
	}
	p := arsenal2Player(1, 400, 1) // pistol: shield in front, facing right
	p.AbilityNumber = abilityBallisticShield
	if !ballisticShieldInFront(p) {
		t.Fatal("pistol + shield must hold it in front")
	}
	if got := hit(p, bullet(-25, 0)); math.Abs(got-40*ballisticShieldBulletScale) > 1e-6 || p.ShieldGearSpark != ballisticShieldSparkFrames {
		t.Fatalf("front hit push %.2f spark %d", got, p.ShieldGearSpark)
	}
	if got := hit(p, bullet(25, 0)); got != 40 {
		t.Fatalf("back hit with the shield in front %.2f", got)
	}
	if got := hit(p, bullet(-4, 20)); got != 40 {
		t.Fatalf("steep hit from above %.2f", got)
	}
	// Two-handed rifle: the shield moves to the back and covers it.
	p.EquipWeapon(10)
	if ballisticShieldInFront(p) {
		t.Fatal("a rifle must sling the shield")
	}
	if got := hit(p, bullet(25, 0)); math.Abs(got-40*ballisticShieldBulletScale) > 1e-6 {
		t.Fatalf("back hit with the shield on the back %.2f", got)
	}
	if got := hit(p, bullet(-25, 0)); got != 40 {
		t.Fatalf("front hit with the shield on the back %.2f", got)
	}
	// Back to a pistol: in front again.
	p.EquipWeapon(1)
	if !ballisticShieldInFront(p) {
		t.Fatal("shield did not come back to the front")
	}
	// Explosions are never covered.
	q := arsenal2Player(2, 400, 1)
	ref := arsenal2Player(3, 400, 1)
	q.AbilityNumber = abilityBallisticShield
	for _, v := range []*Player{q, ref} {
		b := Bullet{X: 460, Y: 370, VX: -17, Facing: -1, OwnerID: 9, ExplosionRadius: 190, ExplosionPower: 100}
		explodeDevelopedProjectile(&b, []*Player{v}, nil, nil)
	}
	if q.VX != ref.VX || q.VX == 0 {
		t.Fatalf("explosion vs shield %.2f, without %.2f", q.VX, ref.VX)
	}
	// Movement cost.
	if ballisticShieldMoveMultiplier(q) >= 1 || ballisticShieldJumpMultiplier(q) >= 1 || ballisticShieldMoveMultiplier(ref) != 1 {
		t.Fatal("shield must slow its bearer")
	}
}

func TestBallisticShieldPowerupLastsOneLife(t *testing.T) {
	p := arsenal2Player(1, 300, 1)
	applyPowerup(p, powerupBallisticShield)
	if !hasBallisticShield(p) || !ballisticShieldInFront(p) {
		t.Fatal("powerup gave no shield")
	}
	p.Kill(arsenal2Arena())
	if hasBallisticShield(p) {
		t.Fatal("pickup shield survived death")
	}
}

func TestAkimboTimingAndShieldInteraction(t *testing.T) {
	p := arsenal2Player(1, 300, 1)
	p.AbilityNumber = abilityBallisticShield
	applyPowerup(p, powerupAkimbo)
	if p.AkimboTime != akimboFrames || !akimboActive(p) {
		t.Fatalf("akimbo not started: %d", p.AkimboTime)
	}
	if ballisticShieldInFront(p) {
		t.Fatal("akimbo must sling the shield")
	}
	if got, want := effectiveWeaponROF(p), (p.Weapon.Def.ROF+1)/2; got != want {
		t.Fatalf("akimbo rof %d want %d", got, want)
	}
	// Shots alternate guns; the off-hand gun does not spend main ammo.
	g := arsenal2Game(p)
	ammo := p.Weapon.Bullets
	p.Weapon.WaitTime = 100
	fireOnce(g, p)
	p.Weapon.WaitTime, p.Weapon.Frame = 100, 1
	fireOnce(g, p)
	if len(g.bullets) != 2 || p.Weapon.Bullets != ammo-1 {
		t.Fatalf("akimbo pair: bullets=%d ammo %d->%d", len(g.bullets), ammo, p.Weapon.Bullets)
	}
	if g.bullets[1].Y <= g.bullets[0].Y-4+akimboOffHandDrop-1 && g.bullets[1].X >= g.bullets[0].X {
		t.Fatalf("off-hand shot not offset: %+v %+v", g.bullets[0], g.bullets[1])
	}
	// A two-handed pickup suspends it (the timer keeps running).
	p.EquipWeapon(10)
	if akimboActive(p) || p.AkimboTime == 0 {
		t.Fatal("akimbo still active with a rifle, or its timer stopped")
	}
	p.AkimboTime = 3
	for i := 0; i < 3; i++ {
		p.tickArsenal2Timers()
	}
	p.EquipWeapon(1)
	if akimboActive(p) || p.AkimboTime != 0 {
		t.Fatalf("akimbo outlived its timer: %d", p.AkimboTime)
	}
	// Respawn clears it.
	applyPowerup(p, powerupAkimbo)
	p.Respawn(arsenal2Arena())
	if p.AkimboTime != 0 {
		t.Fatal("akimbo survived respawn")
	}
}

func TestAkimboSourcePistolFiresDuringShotTimeline(t *testing.T) {
	p := arsenal2Player(1, 300, 3)
	p.Weapon.Frame = 4 // the first gun's shot timeline is still playing
	if weaponReadyToFire(p) {
		t.Fatal("without akimbo the timeline blocks the next shot")
	}
	p.AkimboTime = akimboFrames
	if !weaponReadyToFire(p) {
		t.Fatal("akimbo second gun blocked by the first gun's timeline")
	}
	p.Weapon.Frame = 12 // reloading
	if weaponReadyToFire(p) {
		t.Fatal("akimbo fired during a reload")
	}
}

func TestThermalGogglesSeeInvisiblePerViewer(t *testing.T) {
	viewer := arsenal2Player(1, 200, 1)
	ghost := arsenal2Player(2, 500, 1)
	ghost.InvisibleTime = 100
	ghost.AI = true // one human: this screen has a single viewer
	g := arsenal2Game(viewer, ghost)
	if g.viewerPlayerID() != 1 {
		t.Fatalf("viewer %d", g.viewerPlayerID())
	}
	if g.thermalSeesInvisible(ghost) {
		t.Fatal("no goggles, no shimmer")
	}
	viewer.AbilityNumber = abilityThermalGoggles
	if !g.thermalSeesInvisible(ghost) || g.thermalSeesInvisible(viewer) {
		t.Fatal("thermal viewer must see the invisible enemy (and not itself)")
	}
	// The other player's goggles do not help this viewer.
	viewer.AbilityNumber = abilityNone
	ghost.AbilityNumber = abilityThermalGoggles
	if g.thermalSeesInvisible(ghost) {
		t.Fatal("shimmer leaked to a viewer without goggles")
	}
	// Bots: thermal detects at 420 px instead of 135.
	bot := arsenal2Player(3, 200, 1)
	if aiCanDetectPlayer(bot, ghost) {
		t.Fatal("plain bot saw an invisible player 300 px away")
	}
	bot.AbilityNumber = abilityThermalGoggles
	if !aiCanDetectPlayer(bot, ghost) {
		t.Fatal("thermal bot missed an invisible player 300 px away")
	}
	ghost.X = 200 + aiThermalDetectRange + 20
	if aiCanDetectPlayer(bot, ghost) {
		t.Fatal("thermal range unbounded")
	}
}

func TestArsenal2PowerupSpawnRoll(t *testing.T) {
	rand.Seed(7)
	arena := arsenal2Arena()
	seenNew := map[int]bool{}
	for i := 0; i < 400; i++ {
		if pu := spawnPowerup(arena, false); pu.Type >= 7 {
			t.Fatalf("source roll produced kind %d", pu.Type)
		}
		if pu := spawnPowerup(arena, false, true); pu.Type >= 7 {
			seenNew[pu.Type] = true
		}
	}
	if !seenNew[powerupAkimbo] || !seenNew[powerupBallisticShield] {
		t.Fatalf("extended roll never gave both new kinds: %v", seenNew)
	}
	g := &Game{crateWeapons: DefaultCrateWeapons()}
	if g.extendedPickups() {
		t.Fatal("the GM1 pool must keep the source powerups")
	}
	g.crateWeapons = developedCrateWeapons()
	if !g.extendedPickups() {
		t.Fatal("developed pool must add the new powerups")
	}
	if powerupName(powerupAkimbo) != "AKIMBO" || arsenal2PowerupIcon(powerupAkimbo) == nil || arsenal2PowerupIcon(powerupBallisticShield) == nil {
		t.Fatal("new powerups lack name or icon")
	}
}

func TestArsenal2SnapshotWireAndValidation(t *testing.T) {
	p := Player{ID: 1, Name: "A", Active: true, AkimboTime: 100, AkimboHand: 1, ShieldGearPickup: true, ShieldGearSpark: 3,
		MeleeSwing: 4, BlowerTime: 2, HarpoonTether: 5, HarpoonTetherID: 2, MeleeHitPending: true, BlowerPulse: true}
	data := netEncodeValue(nil, &p)
	var back Player
	if _, err := netDecodeValue(data, &back); err != nil {
		t.Fatal(err)
	}
	if back.AkimboTime != 100 || back.AkimboHand != 1 || !back.ShieldGearPickup || back.ShieldGearSpark != 3 ||
		back.MeleeSwing != 4 || back.BlowerTime != 2 || back.HarpoonTether != 5 || back.HarpoonTetherID != 2 {
		t.Fatalf("synced fields lost: %+v", back)
	}
	if back.MeleeHitPending || back.BlowerPulse {
		t.Fatal("host-only pending flags went on the wire")
	}
	b := Bullet{ShooterID: 3, DevelopedProjectile: DevelopedProjectileHarpoon}
	var bb Bullet
	if _, err := netDecodeValue(netEncodeValue(nil, &b), &bb); err != nil || bb.ShooterID != 3 || bb.DevelopedProjectile != DevelopedProjectileHarpoon {
		t.Fatalf("bullet shooter lost: %+v %v", bb, err)
	}
	ok := netSnapshot{Players: []Player{p}, Powerups: []Powerup{{Type: powerupBallisticShield}}, Bullets: []Bullet{b}}
	if err := validateNetSnapshot(ok); err != nil {
		t.Fatalf("valid snapshot rejected: %v", err)
	}
	bad := []netSnapshot{
		{Players: []Player{{AkimboTime: akimboFrames + 1}}},
		{Players: []Player{{MeleeSwing: -1}}},
		{Players: []Player{{HarpoonTether: 999}}},
		{Powerups: []Powerup{{Type: powerupKindCount}}},
		{PowerupNameFX: []PowerupNameEffect{{Kind: -1}}},
		{Bullets: []Bullet{{DevelopedProjectile: DevelopedProjectileGust + 1}}},
	}
	for i, s := range bad {
		if validateNetSnapshot(s) == nil {
			t.Fatalf("bad snapshot %d accepted", i)
		}
	}
	if netplayProtocolVersion != 24 {
		t.Fatalf("protocol %d", netplayProtocolVersion)
	}
	cfg := CustomPlayerConfig{Name: "G", Color: 1, Shirt: 1, Hat: 1, Eyes: 1, Gun: weaponBaseballBat, Perk: 1, Ability: abilityBallisticShield, Team: 1}
	if _, err := sanitizeRemotePlayerConfig(cfg); err != nil {
		t.Fatalf("bat + ballistic shield guest config rejected: %v", err)
	}
	cfg.Gun = weaponLeafBlower
	if _, err := sanitizeRemotePlayerConfig(cfg); err == nil {
		t.Fatal("blower accepted as a starter weapon")
	}
}

func TestHarpoonReelInterpolatesAndNewHarpoonSnaps(t *testing.T) {
	g := newRenderInterpTestGame()
	p := g.players[0]
	p.Active, p.DeathSerial = true, 0
	p.X, p.Y, p.VX, p.VY = 600, 300, 0, 0
	t0 := time.Unix(2000, 0)
	g.captureRenderPrev(t0)
	// The reel hits this tick: a big velocity change, then one tick of
	// movement (friction applied by the next update).
	p.VX = -harpoonPullMax
	p.X += p.VX
	g.bullets = []Bullet{{X: 300, Y: 260, VX: 22, Facing: 1, DevelopedProjectile: DevelopedProjectileHarpoon, ShooterID: 1}}
	restore := g.beginRenderInterpolation(t0.Add(renderTestTick / 2))
	if got := g.players[0].X; math.Abs(got-(600-harpoonPullMax/2)) > 1e-6 {
		t.Fatalf("reeled player snapped instead of sliding: %.2f", got)
	}
	if got := g.bullets[0].X; got != 300 {
		t.Fatalf("a new harpoon must be drawn where it spawned: %.2f", got)
	}
	restore()
	// Next tick the harpoon is interpolated.
	g.captureRenderPrev(t0.Add(renderTestTick))
	g.bullets[0].X += 22
	restore = g.beginRenderInterpolation(t0.Add(renderTestTick + renderTestTick/2))
	if got := g.bullets[0].X; math.Abs(got-311) > 1e-6 {
		t.Fatalf("harpoon in flight drawn at %.2f want 311", got)
	}
	restore()
}

func TestArsenal2ZombieRules(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	holdZombieWave(g)
	p := g.players[0]
	p.EquipWeapon(weaponBaseballBat)
	placePlayer(p, p.X, p.Y)
	z := spawnTestZombie(g, zombieWalker, p.X+40, p.Y, 40)
	hp := z.HP
	p.Facing = 1
	p.MeleeHitPending = true
	targets := g.hitTargets()
	g.beginZombieHitPhase()
	w := beginAbilityKnockbackWindow(targets)
	g.applyArsenal2Hits(targets)
	endAbilityKnockbackWindow(targets, w, true)
	g.endZombieHitPhase()
	if z.HP > 0 && hp-z.HP < int(batPush) {
		t.Fatalf("bat did %d damage to a walker (%d HP)", hp-z.HP, hp)
	}
	// Wind moves zombies but deals no damage.
	z2 := spawnTestZombie(g, zombieBrute, p.X+90, p.Y, 170)
	hp2 := z2.HP
	p.EquipWeapon(weaponLeafBlower)
	p.BlowerPulse = true
	targets = g.hitTargets()
	g.beginZombieHitPhase()
	w = beginAbilityKnockbackWindow(targets)
	g.applyArsenal2Hits(targets)
	endAbilityKnockbackWindow(targets, w, true)
	moved := z2.body.VX
	g.endZombieHitPhase()
	if moved <= 0 || z2.HP != hp2 {
		t.Fatalf("blower on a zombie: push %.3f, HP %d -> %d", moved, hp2, z2.HP)
	}
	// Claws into a raised ballistic shield: half damage, 40% push.
	zp := g.zombie.playerState(p.ID)
	p.EquipWeapon(1)
	p.AbilityNumber = abilityBallisticShield
	p.Facing = 1
	zp.HP, p.VX = 100, 0
	g.zombieClawPlayer(p, 20, -10, 3, 9000)
	if zp.HP != 90 || math.Abs(p.VX+4) > 1e-6 {
		t.Fatalf("claw into shield: hp=%d vx=%.2f", zp.HP, p.VX)
	}
	zp.HP, p.VX = 100, 0
	g.zombieClawPlayer(p, 20, 10, 3, 9000) // from behind
	if zp.HP != 80 || p.VX != 10 {
		t.Fatalf("claw into the back: hp=%d vx=%.2f", zp.HP, p.VX)
	}
}

func TestArsenal2ZombieShop(t *testing.T) {
	var bat, blower, shield, harpoon bool
	for _, it := range zombieShopItems() {
		switch {
		case it.Kind == zombieItemWeapon && it.Value == weaponBaseballBat:
			bat = true
		case it.Kind == zombieItemWeapon && it.Value == weaponLeafBlower:
			blower = true
		case it.Kind == zombieItemWeapon && it.Value == weaponHarpoonGun:
			harpoon = true
		case it.Kind == zombieItemGear && it.Value == abilityBallisticShield:
			shield = true
		}
	}
	if !bat || !blower || !shield || harpoon {
		t.Fatalf("shop bat=%v blower=%v shield=%v harpoon=%v", bat, blower, shield, harpoon)
	}
	inBox := map[int]bool{}
	for _, n := range zombieMysteryPool() {
		inBox[n] = true
	}
	if !inBox[weaponHarpoonGun] || !inBox[weaponLeafBlower] || inBox[weaponBaseballBat] {
		t.Fatalf("box pool harpoon=%v blower=%v bat=%v", inBox[weaponHarpoonGun], inBox[weaponLeafBlower], inBox[weaponBaseballBat])
	}
}

func TestArsenal2BotsUseTheNewWeapons(t *testing.T) {
	arena := arsenal2Arena()
	// Bat: swings only in reach.
	bot := arsenal2Player(1, 200, weaponBaseballBat)
	bot.AI = true
	enemy := arsenal2Player(2, 330, 1)
	g := &Game{arena: arena, players: []*Player{bot, enemy}}
	bot.AIShoot = true
	aiArsenal2Decide(bot, g, enemy)
	if bot.AIShoot {
		t.Fatal("bat bot swung at 130 px")
	}
	enemy.X = 240
	aiArsenal2Decide(bot, g, enemy)
	if !bot.AIShoot {
		t.Fatal("bat bot did not swing in reach")
	}
	if aiPreferredRange(bot) >= 60 {
		t.Fatal("bat bot keeps its distance")
	}
	// Blower: blows away an incoming grenade without a target.
	bot.EquipWeapon(weaponLeafBlower)
	g.grenades = []Grenade{{X: 300, Y: 364, VX: -3, OwnerID: 2}}
	bot.AIShoot = false
	aiArsenal2Decide(bot, g, nil)
	if !bot.AIShoot {
		t.Fatal("blower bot ignored an incoming grenade")
	}
	g.grenades = nil
	// Harpoon: no hook at point blank.
	bot.EquipWeapon(weaponHarpoonGun)
	enemy.X = 260
	bot.AIShoot = true
	aiArsenal2Decide(bot, g, enemy)
	if bot.AIShoot {
		t.Fatal("harpoon bot hooked at point blank")
	}
	// Shield flank: a rifle bot jumps over a shield bearer facing it.
	bot.EquipWeapon(10)
	enemy.X, enemy.Facing = 320, -1
	enemy.AbilityNumber = abilityBallisticShield
	bot.AIUp = false
	aiArsenal2Decide(bot, g, enemy)
	if !bot.AIUp || !bot.AIRight {
		t.Fatalf("bot did not flank the shield: up=%v right=%v", bot.AIUp, bot.AIRight)
	}
	if !aiShotIntoShield(bot, enemy) {
		t.Fatal("shot into the shield not recognised")
	}
	enemy.Facing = 1
	if aiShotIntoShield(bot, enemy) {
		t.Fatal("bearer's unprotected back counted as shielded")
	}
}

func TestBatDrawRotationSweepsAndRests(t *testing.T) {
	p := &Player{Weapon: NewWeapon(weaponBaseballBat)}
	if r := gunDrawRotation(p); r != -55 {
		t.Fatalf("rest angle %.1f", r)
	}
	prev := -1000.0
	peak := -1000.0
	for s := batSwingFrames; s >= 1; s-- {
		p.MeleeSwing = s
		r := meleeDrawRotation(p)
		peak = math.Max(peak, r)
		if s > 3 && r < prev {
			t.Fatalf("swing goes backward at %d: %.1f < %.1f", s, r, prev)
		}
		prev = r
	}
	if peak < 40 {
		t.Fatalf("swing never reaches the front: %.1f", peak)
	}
}
