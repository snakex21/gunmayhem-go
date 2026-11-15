package game

import (
	"math"
	"testing"
	"time"
)

// --- helpers -------------------------------------------------------------------------

func stageBGame(t *testing.T, mapNumber int) (*Game, *Player) {
	t.Helper()
	arena, err := LoadOriginalMap(mapNumber)
	if err != nil {
		t.Fatal(err)
	}
	human := NewPlayer(1, arena)
	human.Team = 1
	human.Lives, human.TotalLives = 99, 99
	g := &Game{arena: arena, players: []*Player{human}, seenDeaths: map[int]int{}, campaignMode: true, campaignSetExtra: true, campaignLevel: 26}
	g.resetCampaignEnemies()
	return g, human
}

func standOn(g *Game, p *Player, x, y float64) {
	p.X, p.Y, p.VX, p.VY = x, y, 0, 0
	p.JumpNum, p.Grounded, p.Active = 2, true, true
}

func spawnMechAt(g *Game, x, y float64, hp int, facing int) *CampaignEnemy {
	e := g.spawnCampaignEnemy(extraCampaignUnit{Kind: campaignEnemyMech, HP: hp}, -1, g.arena)
	standOn(g, &e.Body, x, y)
	e.Body.Facing = facing
	return e
}

func spawnHeliAt(g *Game, x, y float64, hp int) *CampaignEnemy {
	e := g.spawnCampaignEnemy(extraCampaignUnit{Kind: campaignEnemyHelicopter, HP: hp, Title: "GUNSHIP"}, -1, g.arena)
	e.Body.X, e.Body.Y = x, y
	e.HomeX = x
	e.State, e.StateTimer = heliStateHover, 1000
	return e
}

// --- mech ---------------------------------------------------------------------------------

// TestMechPathfindsAcrossPlatformsWithoutGettingStuck walks the heavy mech
// from the top of a map to a target several platforms below.
func TestMechPathfindsAcrossPlatformsWithoutGettingStuck(t *testing.T) {
	cases := []struct {
		mapN           int
		mx, tx         float64
		mechY, targetY float64
	}{
		{1, 200, 700, 256, 526},
		{1, 820, 250, 256, 436},
		{7, 250, 700, 151, 511},
		{11, 300, 450, 198, 454},
	}
	for _, c := range cases {
		g, human := stageBGame(t, c.mapN)
		m := spawnMechAt(g, c.mx, c.mechY, 400, 1)
		m.attackCD = 1 << 20 // walk only
		nav := g.mechNavFor(g.arena)
		goal := nav.platformNear(c.tx, c.targetY)
		reached := -1
		for i := 0; i < 1500 && reached < 0; i++ {
			standOn(g, human, c.tx, c.targetY)
			g.updateCampaignEnemies(g.arena)
			if !m.alive() {
				t.Fatalf("map %d: the mech fell off at tick %d (%.0f, %.0f)", c.mapN, i, m.Body.X, m.Body.Y)
			}
			if m.Body.JumpNum == 2 && nav.platformUnder(m.Body.X, m.Body.Y) == goal {
				reached = i
			}
		}
		if reached < 0 {
			t.Fatalf("map %d: mech stuck at (%.0f, %.0f), target platform %d", c.mapN, m.Body.X, m.Body.Y, goal)
		}
		t.Logf("map %d: target platform reached in %d ticks", c.mapN, reached)
	}
}

func TestMechWeakSpotTakesMorePushAndDamage(t *testing.T) {
	g, human := stageBGame(t, 1)
	standOn(g, human, 100, 256)
	shoot := func(fromLeft bool) (dvx float64, hp int) {
		m := spawnMechAt(g, 450, 256, 100, 1) // facing right: its back is on the left
		dir := -1.0
		x := m.Body.X + 50
		if fromLeft {
			dir, x = 1, m.Body.X-50
		}
		g.bullets = append(g.bullets[:0], Bullet{Kind: BulletNormal, X: x, Y: m.Body.Y - 40, VX: 25 * dir, Firepower: 20, OwnerID: human.ID, Team: 1, Facing: int(dir), Alpha: 1})
		runBulletHitPhase(g)
		dvx = m.Body.VX
		hp = m.HP
		m.Dead, m.Body.Active = true, false
		return
	}
	frontVX, frontHP := shoot(false)
	backVX, backHP := shoot(true)
	if want := -20 * mechKnockbackScale; math.Abs(frontVX-want) > 1e-6 {
		t.Fatalf("front push %.3f want %.3f", frontVX, want)
	}
	if want := 20 * mechKnockbackScale * mechBackMult; math.Abs(backVX-want) > 1e-6 {
		t.Fatalf("back push %.3f want %.3f", backVX, want)
	}
	if frontHP != 99 || backHP != 98 { // 1 HP vs 2.5 HP (the half carries over)
		t.Fatalf("hp front=%d back=%d", frontHP, backHP)
	}
}

func TestMechDeMaterializerBurnsInsteadOfErasing(t *testing.T) {
	g, _ := stageBGame(t, 1)
	m := spawnMechAt(g, 450, 256, 100, 1)
	g.beginCampaignEnemyHitPhase()
	m.Body.InstagibPlaying = true
	g.endCampaignEnemyHitPhase(g.arena)
	if m.Body.InstagibPlaying || m.HP != 100-bigEnemyInstagibHP || !m.alive() {
		t.Fatalf("instagib on a mech: playing=%v hp=%d", m.Body.InstagibPlaying, m.HP)
	}
}

func TestMechStompTelegraphsThenThrowsOnlyGroundedPlayers(t *testing.T) {
	g, human := stageBGame(t, 1)
	jumper := NewPlayer(2, g.arena)
	jumper.Team = 1
	g.players = append(g.players, jumper)
	m := spawnMechAt(g, 450, 256, 100, 1)
	m.attackCD = 0
	sawWind := false
	for i := 0; i < mechStompWind+10; i++ {
		if m.State != mechStateStompRecover {
			standOn(g, human, 540, 256)
			standOn(g, jumper, 380, 256)
		}
		if m.State == mechStateStompWind {
			sawWind = true
			if m.StateTimer == mechStompWind-1 {
				jumper.JumpNum, jumper.Y = 1, 200 // in the air at the impact
			}
		}
		g.updateCampaignEnemies(g.arena)
		if m.State == mechStateStompRecover {
			break
		}
		if human.VX != 0 {
			t.Fatalf("pushed during the wind-up (tick %d)", i)
		}
	}
	if !sawWind || m.State != mechStateStompRecover {
		t.Fatalf("no telegraphed stomp: state=%d", m.State)
	}
	if human.VX <= 3 || human.LastHitBy != m.Body.ID {
		t.Fatalf("grounded player not thrown: vx=%.2f by=%d", human.VX, human.LastHitBy)
	}
	if jumper.VX != 0 {
		t.Fatalf("airborne player hit by the stomp: vx=%.2f", jumper.VX)
	}
}

func TestMechMachineGunAimsBeforeFiring(t *testing.T) {
	g, human := stageBGame(t, 1)
	m := spawnMechAt(g, 300, 256, 100, 1)
	m.attackCD = 0
	firstShot, aimStart := -1, -1
	for i := 0; i < 120 && firstShot < 0; i++ {
		standOn(g, human, 600, 256)
		before := len(g.bullets)
		g.updateCampaignEnemies(g.arena)
		if m.State == mechStateAim && aimStart < 0 {
			aimStart = i
		}
		if len(g.bullets) > before {
			firstShot = i
		}
	}
	if aimStart < 0 || firstShot < 0 || firstShot-aimStart < mechAimFrames {
		t.Fatalf("aim at %d, first shot at %d (want >= %d ticks of laser)", aimStart, firstShot, mechAimFrames)
	}
}

func TestMechHarpoonHookCancelsWindup(t *testing.T) {
	g, human := stageBGame(t, 1)
	m := spawnMechAt(g, 450, 256, 100, 1)
	m.State, m.StateTimer = mechStateStompWind, 5
	m.Body.HarpoonTether, m.Body.HarpoonTetherID = harpoonTetherFrames, human.ID
	standOn(g, human, 100, 256)
	g.updateCampaignEnemies(g.arena)
	if m.State == mechStateStompWind || m.attackCD == 0 {
		t.Fatalf("hook did not cancel the stomp: state=%d cd=%d", m.State, m.attackCD)
	}
}

func TestMechCanStillBeRungOut(t *testing.T) {
	g, human := stageBGame(t, 1)
	standOn(g, human, 300, 256)
	m := spawnMechAt(g, 860, 256, 100, -1)
	m.attackCD = 1 << 20
	m.Body.VX = 30
	m.Body.LastHitBy = human.ID
	for i := 0; i < 200 && m.alive(); i++ {
		standOn(g, human, 300, 256)
		g.updateCampaignEnemies(g.arena)
	}
	if m.alive() || human.Kills != 1 || human.Score != campaignEnemyMech.killScore() {
		t.Fatalf("ring-out: alive=%v kills=%d score=%d", m.alive(), human.Kills, human.Score)
	}
}

func TestMechMeltdownCountsUntilItExplodes(t *testing.T) {
	g, human := stageBGame(t, 1)
	standOn(g, human, 100, 256)
	m := spawnMechAt(g, 450, 256, 3, 1)
	g.damageBigEnemy(m, 5, human.ID)
	if m.State != mechStateMeltdown || m.hittable() || g.campaignEnemiesRemaining() != 1 {
		t.Fatalf("meltdown: state=%d hittable=%v remaining=%d", m.State, m.hittable(), g.campaignEnemiesRemaining())
	}
	for i := 0; i < mechMeltdownFrames+2; i++ {
		standOn(g, human, 100, 256)
		g.updateCampaignEnemies(g.arena)
	}
	if m.alive() || g.campaignEnemiesRemaining() != 0 || human.Kills != 1 {
		t.Fatalf("after meltdown: alive=%v remaining=%d kills=%d", m.alive(), g.campaignEnemiesRemaining(), human.Kills)
	}
}

// --- gunship -------------------------------------------------------------------------------

func TestHeliPhasesFollowHPWithBanners(t *testing.T) {
	g, human := stageBGame(t, 5)
	standOn(g, human, 450, 486)
	h := spawnHeliAt(g, 450, 250, 90)
	for _, c := range []struct{ hp, phase int }{{61, 0}, {60, 1}, {31, 1}, {30, 2}} {
		h.HP = c.hp
		g.updateCampaignEnemies(g.arena)
		if h.Phase != c.phase {
			t.Fatalf("hp %d: phase %d want %d", c.hp, h.Phase, c.phase)
		}
	}
	if g.bossBannerFrame == 0 || g.bossBanner == "" {
		t.Fatal("no phase banner")
	}
	// Skipping straight to phase 3 still runs phase 2 first.
	h2 := spawnHeliAt(g, 450, 250, 90)
	h2.HP = 5
	g.updateCampaignEnemies(g.arena)
	if h2.Phase != 2 {
		t.Fatalf("phase after a big hit: %d", h2.Phase)
	}
}

// TestHeliSalvoMarkersPrecedeEveryImpact runs a salvo with full bullet
// physics and checks every rocket lands on a marker that was shown at
// least the wind-up earlier.
func TestHeliSalvoMarkersPrecedeEveryImpact(t *testing.T) {
	g, human := stageBGame(t, 5)
	h := spawnHeliAt(g, 300, 260, 90)
	h.Phase = 2
	standOn(g, human, 520, 486)
	g.heliPlanSalvo(h, g.arena)
	n := heliRocketsPerPhase[2]
	if len(g.extraHazardMarkers) != n || h.salvo != n {
		t.Fatalf("markers=%d salvo=%d", len(g.extraHazardMarkers), h.salvo)
	}
	type mark struct {
		x, y float64
		born int
	}
	var marks []mark
	for _, m := range g.extraHazardMarkers {
		marks = append(marks, mark{m.X, m.Y, 0})
	}
	impacts := 0
	for tick := 1; tick < 400 && impacts < n; tick++ {
		standOn(g, human, 900, 486) // out of the way: rockets reach the ground
		g.updateCampaignEnemies(g.arena)
		alive := map[int]bool{}
		for i := range g.bullets {
			if !g.bullets[i].Dead && g.bullets[i].bossShot == bossShotRocket {
				alive[i] = true
			}
		}
		updateBullets(g.arena, g.bullets, g.hitTargets(), &g.instagibTrails, &g.explosions, &g.blastFX, &g.portals)
		g.updateHazardMarkers()
		for i := range alive {
			b := &g.bullets[i]
			if !b.Dead {
				continue
			}
			impacts++
			if tick < heliSalvoWind {
				t.Fatalf("rocket landed at tick %d, before the wind-up ended", tick)
			}
			ok := false
			for _, m := range marks {
				if math.Hypot(b.X-m.x, b.Y-m.y) < 30 {
					ok = true
				}
			}
			if !ok {
				t.Fatalf("rocket landed at (%.0f, %.0f) with no marker", b.X, b.Y)
			}
			// Its marker is still up when it lands.
			still := false
			for _, m := range g.extraHazardMarkers {
				if math.Hypot(b.X-m.X, b.Y-m.Y) < 30 && m.Life > 0 {
					still = true
				}
			}
			if !still {
				t.Fatalf("marker gone before the impact at tick %d", tick)
			}
		}
		g.bullets = compactDeadBullets(g.bullets)
	}
	if impacts != n {
		t.Fatalf("%d of %d rockets landed", impacts, n)
	}
}

func compactDeadBullets(list []Bullet) []Bullet {
	out := list[:0]
	for _, b := range list {
		if !b.Dead {
			out = append(out, b)
		}
	}
	return out
}

func TestHeliStrafeLineShowsBeforeAnyRound(t *testing.T) {
	g, human := stageBGame(t, 5)
	standOn(g, human, 450, 486)
	h := spawnHeliAt(g, 200, 280, 90)
	g.heliNextAttack(h, human, g.arena)
	if h.State != heliStateStrafeLine || math.Abs(h.AimY-486) > 1 || h.AimX0 == h.AimX1 {
		t.Fatalf("strafe line: state=%d aim=%.0f..%.0f y=%.0f", h.State, h.AimX0, h.AimX1, h.AimY)
	}
	firstRound := -1
	for i := 0; i < 400 && firstRound < 0; i++ {
		standOn(g, human, 450, 486)
		before := len(g.bullets)
		g.updateCampaignEnemies(g.arena)
		if len(g.bullets) > before {
			firstRound = i
			if h.State != heliStateStrafe {
				t.Fatalf("round fired in state %d", h.State)
			}
		}
	}
	if firstRound < heliStrafeWarn {
		t.Fatalf("first round at tick %d, warning is %d", firstRound, heliStrafeWarn)
	}
	// Strafing rounds stop on platforms instead of raining through them.
	g.bullets = append(g.bullets[:0], Bullet{Kind: BulletNormal, X: 100, Y: 290, VX: 6, VY: 15, Firepower: 7, OwnerID: h.Body.ID, Team: campaignEnemyTeam, Facing: 1, Alpha: 1, bossShot: bossShotRound})
	for i := 0; i < 5; i++ {
		g.updateCampaignBossBullets(g.hitTargets())
		updateBullets(g.arena, g.bullets, g.hitTargets(), &g.instagibTrails, &g.explosions, &g.blastFX, &g.portals)
	}
	if !g.bullets[0].Dead || g.bullets[0].Y > 330 {
		t.Fatalf("round passed the ledge: y=%.0f dead=%v", g.bullets[0].Y, g.bullets[0].Dead)
	}
}

func TestHeliDropsRespectAliveAndTotalCaps(t *testing.T) {
	g, human := stageBGame(t, 5)
	h := spawnHeliAt(g, 450, 300, 90)
	h.Phase = 2
	for i := 0; i < 20; i++ {
		standOn(g, human, 300, 486)
		h.Body.X = 450
		g.heliDropAdd(h, g.arena)
		if n := g.heliAddsAlive(h); n > heliMaxAdds {
			t.Fatalf("%d adds alive", n)
		}
		if i%3 == 2 {
			// Kill one add so the total cap is reached too.
			for _, e := range g.campaignEnemies {
				if e.summoner == h.Body.ID && e.alive() {
					g.destroyCampaignEnemy(e)
					break
				}
			}
		}
	}
	if h.drops != heliMaxDrops {
		t.Fatalf("drops=%d want %d", h.drops, heliMaxDrops)
	}
	if g.heliCanDrop(h) {
		t.Fatal("can still drop past the total cap")
	}
	if len(g.campaignEnemies) > 32 {
		t.Fatalf("%d campaign enemies exceed the snapshot cap", len(g.campaignEnemies))
	}
}

func TestHeliRocketBattedBackDamagesIt(t *testing.T) {
	g, human := stageBGame(t, 5)
	standOn(g, human, 450, 486)
	human.Facing = -1
	h := spawnHeliAt(g, 250, 300, 90)
	g.bullets = append(g.bullets, Bullet{Kind: BulletNormal, X: 480, Y: 440, VX: -5, VY: 5, Firepower: heliRocketPush,
		OwnerID: h.Body.ID, Team: campaignEnemyTeam, Facing: -1, Alpha: 1, Time: 1, DevelopedProjectile: DevelopedProjectileRocket,
		ExplosionRadius: heliRocketRadius, ExplosionPower: heliRocketPower, Lifetime: 240, bossShot: bossShotRocket, bossFrom: h.Body.ID})
	reflectProjectile(&g.bullets[0], human)
	for i := 0; i < 80 && h.HP == 90; i++ {
		g.updateCampaignBossBullets(g.hitTargets())
		updateBullets(g.arena, g.bullets, g.hitTargets(), &g.instagibTrails, &g.explosions, &g.blastFX, &g.portals)
	}
	if want := 90 - int(heliRocketDamage); h.HP != want || h.Body.LastHitBy != human.ID {
		t.Fatalf("returned rocket: hp=%d want %d credit=%d", h.HP, want, h.Body.LastHitBy)
	}
}

func TestHeliHarpoonDragsItDownThenCoolsDown(t *testing.T) {
	g, human := stageBGame(t, 5)
	standOn(g, human, 300, 486)
	human.EquipWeapon(weaponHarpoonGun)
	h := spawnHeliAt(g, 520, 260, 90)
	h.Body.HarpoonTether, h.Body.HarpoonTetherID = harpoonTetherFrames, human.ID
	for i := 0; i < 60; i++ {
		standOn(g, human, 300, 486)
		g.updateCampaignEnemies(g.arena)
	}
	if h.Hooked <= 0 || h.Body.HarpoonTether <= 0 {
		t.Fatalf("not hooked: %d", h.Hooked)
	}
	if want := 486 - heliHookedAbove; math.Abs(h.Body.Y-want) > 40 {
		t.Fatalf("dragged to y=%.0f want about %.0f", h.Body.Y, want)
	}
	for i := 0; i < heliHookFrames; i++ {
		standOn(g, human, 300, 486)
		g.updateCampaignEnemies(g.arena)
	}
	if h.Hooked != 0 || h.hookCD == 0 {
		t.Fatalf("hook never ended: hooked=%d cd=%d", h.Hooked, h.hookCD)
	}
}

func TestHeliCrashKeepsCountingThenCredits(t *testing.T) {
	g, human := stageBGame(t, 5)
	standOn(g, human, 450, 486)
	h := spawnHeliAt(g, 450, 250, 10)
	g.damageBigEnemy(h, 12, human.ID)
	if h.State != heliStateCrash || h.hittable() || g.campaignEnemiesRemaining() != 1 {
		t.Fatalf("crash start: state=%d hittable=%v remaining=%d", h.State, h.hittable(), g.campaignEnemiesRemaining())
	}
	for i := 0; i < heliCrashFrames+5 && h.alive(); i++ {
		standOn(g, human, 450, 486)
		g.updateCampaignEnemies(g.arena)
	}
	if h.alive() || g.campaignEnemiesRemaining() != 0 {
		t.Fatal("gunship never finished crashing")
	}
	if human.Kills != 1 || human.Score != campaignEnemyHelicopter.killScore() {
		t.Fatalf("crash credit kills=%d score=%d", human.Kills, human.Score)
	}
}

// TestHeliNeverLeavesTheArenaOrFallsWhileFighting plays a long fight with
// random harassment and checks its flight envelope.
func TestHeliNeverLeavesTheArenaOrFallsWhileFighting(t *testing.T) {
	g, human := stageBGame(t, 5)
	h := g.spawnCampaignEnemy(extraCampaignUnit{Kind: campaignEnemyHelicopter, HP: 900, Title: "GUNSHIP"}, 1, g.arena)
	minX, maxX, top, bottom := campaignArenaBounds(g.arena)
	for i := 0; i < 3000; i++ {
		x := 200 + float64((i/200)%3)*250
		standOn(g, human, x, 486)
		if i == 1200 {
			h.HP = 500
		}
		if i == 2000 {
			h.HP = 250
		}
		g.updateCampaignEnemies(g.arena)
		g.updateCampaignBossBullets(g.hitTargets())
		updateBullets(g.arena, g.bullets, g.hitTargets(), &g.instagibTrails, &g.explosions, &g.blastFX, &g.portals)
		g.bullets = compactDeadBullets(g.bullets)
		g.updateHazardMarkers()
		if !h.alive() {
			t.Fatalf("gunship died untouched at tick %d", i)
		}
		if h.State != heliStateEnter && (h.Body.X < minX-80 || h.Body.X > maxX+80 || h.Body.Y > bottom || h.Body.Y < top-500) {
			t.Fatalf("tick %d: gunship out of its envelope (%.0f, %.0f)", i, h.Body.X, h.Body.Y)
		}
		if len(g.campaignEnemies) > 32 {
			t.Fatalf("tick %d: %d enemies", i, len(g.campaignEnemies))
		}
	}
	if h.Phase != 2 || h.drops == 0 {
		t.Fatalf("phase=%d drops=%d after a long fight", h.Phase, h.drops)
	}
}

// --- net, interpolation, wiring, performance -------------------------------------------------

func TestStageBSnapshotCarriesBigEnemiesWithoutHostState(t *testing.T) {
	g, human := stageBGame(t, 5)
	standOn(g, human, 450, 486)
	m := spawnMechAt(g, 300, 486, 70, 1)
	m.State, m.AimX0, m.AimX1, m.AimY = mechStateAim, 340, 820, 448
	m.Mortar, m.dmgAcc = true, 0.7
	h := spawnHeliAt(g, 450, 250, 90)
	h.Phase, h.Hooked, h.hookBy = 1, 40, 1
	h.rocketPts[0] = [2]float64{1, 2}
	g.bossBanner, g.bossBannerFrame = "Black Kite: ROCKET PODS ARMED", 90
	s := netSnapshot{CampaignEnemies: g.campaignEnemySnapshot(), BossBanner: g.bossBanner, BossBannerFrame: g.bossBannerFrame}
	if err := validateNetSnapshot(s); err != nil {
		t.Fatalf("valid big enemies rejected: %v", err)
	}
	data := netEncodeValue(nil, &s)
	var got netSnapshot
	if _, err := netDecodeValue(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.CampaignEnemies) != 2 || got.BossBanner != g.bossBanner || got.BossBannerFrame != 90 {
		t.Fatalf("decoded %d enemies, banner %q", len(got.CampaignEnemies), got.BossBanner)
	}
	gm, gh := got.CampaignEnemies[0], got.CampaignEnemies[1]
	if gm.State != mechStateAim || gm.AimX1 != 820 || !gm.Mortar || gm.dmgAcc != 0 || gm.Body.hitW != 0 {
		t.Fatalf("mech wire: %+v", gm)
	}
	if gh.Phase != 1 || gh.Hooked != 40 || !gh.Elite || gh.Title != "GUNSHIP" || gh.hookBy != 0 || gh.rocketPts[0][0] != 0 {
		t.Fatalf("gunship wire: phase=%d hooked=%d elite=%v title=%q", gh.Phase, gh.Hooked, gh.Elite, gh.Title)
	}
	for _, bad := range []func(*CampaignEnemy){
		func(e *CampaignEnemy) { e.Kind = int(campaignEnemyKindCount) },
		func(e *CampaignEnemy) { e.HP = e.MaxHP + 1 },
		func(e *CampaignEnemy) { e.Phase = 9 },
		func(e *CampaignEnemy) { e.Hooked = 999 },
		func(e *CampaignEnemy) { e.AimX0 = math.NaN() },
	} {
		c := netSnapshot{CampaignEnemies: append([]CampaignEnemy(nil), got.CampaignEnemies...)}
		bad(&c.CampaignEnemies[1])
		if validateNetSnapshot(c) == nil {
			t.Fatal("bad campaign enemy accepted")
		}
	}
	if validateNetSnapshot(netSnapshot{BossBannerFrame: -1}) == nil {
		t.Fatal("bad banner timer accepted")
	}
	// A client mirror draws from the wire copy.
	guest := &Game{}
	guest.applyCampaignEnemySnapshot(got.CampaignEnemies)
	if len(guest.campaignEnemies) != 2 || guest.campaignEnemies[1].Hooked != 40 {
		t.Fatal("client mirror")
	}
}

func TestStageBRenderInterpolationSlidesAndSnaps(t *testing.T) {
	g := newRenderInterpTestGame()
	g.campaignMode, g.campaignSetExtra, g.campaignLevel = true, true, 28
	g.resetCampaignEnemies()
	h := spawnHeliAt(g, 400, 200, 90)
	m := spawnMechAt(g, 300, 256, 70, 1)
	now := time.Unix(3000, 0)
	g.captureRenderPrev(now)
	h.Body.X += 6
	m.Body.X += 2
	fresh := spawnMechAt(g, 700, 256, 70, -1) // new this tick: snaps
	g.renderInterp.lastUpdate = now
	restore := g.beginRenderInterpolation(now.Add(time.Second / TicksPerSecond / 2))
	pos := map[int]float64{}
	for _, e := range g.campaignEnemies {
		pos[e.Body.ID] = e.Body.X
	}
	restore()
	if math.Abs(pos[h.Body.ID]-403) > 0.6 || math.Abs(pos[m.Body.ID]-301) > 0.6 {
		t.Fatalf("not interpolated: heli %.1f mech %.1f", pos[h.Body.ID], pos[m.Body.ID])
	}
	if pos[fresh.Body.ID] != 700 {
		t.Fatalf("a new mech slid in: %.1f", pos[fresh.Body.ID])
	}
	if h.Body.X != 406 {
		t.Fatal("tick state not restored")
	}
}

func TestStageBMissionsWiredIntoExtraCampaign(t *testing.T) {
	want := map[int]campaignEnemyKind{26: campaignEnemyMech, 27: campaignEnemyMech, 28: campaignEnemyHelicopter}
	for level, kind := range want {
		spec := extraCampaignMissions[level-1]
		found := false
		for _, u := range spec.Units {
			if u.Kind == kind {
				found = true
			}
		}
		if !found {
			t.Fatalf("mission %d lacks %v", level, kind)
		}
		if lines := extraCampaignOpponentLines(spec); len(lines) == 0 || len(lines) > 3 {
			t.Fatalf("mission %d opponent lines: %v", level, lines)
		}
	}
	g := New()
	g.extraCampaignLevels[27] = 1
	g.campaignLevel = 28
	g.startExtraCampaignMission()
	if len(g.crateWeapons) != len(extraAntiAirCrates) {
		t.Fatalf("finale crate pool %v", g.crateWeapons)
	}
	if g.campaignEnemiesRemaining() != 1 {
		t.Fatalf("finale enemies remaining %d", g.campaignEnemiesRemaining())
	}
	// Completing mission 25 unlocks 26 (saves made before Stage B too).
	var levels [extraCampaignMissionCount]int
	for i := 0; i < 25; i++ {
		levels[i] = 2
	}
	if normalizeExtraCampaignLevels(levels)[25] != 1 {
		t.Fatal("mission 26 not unlocked by an old save")
	}
}

func TestStageBCoopScalesBigEnemyHP(t *testing.T) {
	g, _ := stageBGame(t, 5)
	p2 := NewPlayer(2, g.arena)
	p2.Team = 1
	g.players = append(g.players, p2)
	h := g.spawnCampaignEnemy(extraCampaignUnit{Kind: campaignEnemyHelicopter, HP: 90}, 1, g.arena)
	if h.MaxHP != 144 {
		t.Fatalf("2-player gunship HP %d want 144", h.MaxHP)
	}
}

// TestStageBFinalePerformance plays the gunship finale through Game.Update.
func TestStageBFinalePerformance(t *testing.T) {
	g := New()
	g.extraCampaignLevels[27] = 1
	g.campaignLevel = 28
	g.startExtraCampaignMission()
	g.screen = screenGameplay
	g.fadeActive = false
	for _, p := range g.players {
		if !p.AI {
			p.RemoteControlled = true
			p.Lives = 99
		}
	}
	start := time.Now()
	const ticks = 1500
	for i := 0; i < ticks; i++ {
		for _, e := range g.campaignEnemies {
			if campaignEnemyKind(e.Kind) == campaignEnemyHelicopter && i == 700 {
				e.HP = e.MaxHP / 4 // phase 3: drops and salvos
			}
		}
		_ = g.Update()
	}
	per := time.Since(start) / ticks
	t.Logf("finale tick: %v average over %d ticks, %d enemies, %d bullets", per, ticks, len(g.campaignEnemies), len(g.bullets))
	if per > 8*time.Millisecond {
		t.Fatalf("finale tick too slow: %v", per)
	}
}
