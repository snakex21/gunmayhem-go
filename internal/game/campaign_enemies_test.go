package game

import (
	"math"
	"testing"
)

// enemyTestGame is a minimal EXTRA-like game on map 1 with one human.
func enemyTestGame() (*Game, *Player) {
	arena := OriginalMap1()
	human := NewPlayer(1, arena)
	human.Team = 1
	human.X, human.Y = 450, hazardLandingY(arena, 450)
	human.VX, human.VY = 0, 0
	g := &Game{arena: arena, players: []*Player{human}, seenDeaths: map[int]int{}, campaignMode: true, campaignSetExtra: true, campaignLevel: 21}
	g.resetCampaignEnemies()
	return g, human
}

func spawnTestEnemy(g *Game, kind campaignEnemyKind, hp int) *CampaignEnemy {
	return g.spawnCampaignEnemy(extraCampaignUnit{Kind: kind, HP: hp}, -1, g.arena)
}

// runHitPhase pushes the bodies through the same calls Game.Update uses.
func runBulletHitPhase(g *Game) {
	targets := g.hitTargets()
	g.beginCampaignEnemyHitPhase()
	w := beginAbilityKnockbackWindow(targets)
	g.deflectShieldBullets(targets)
	updateBullets(g.arena, g.bullets, targets, &g.instagibTrails, &g.explosions, &g.blastFX, &g.portals)
	endAbilityKnockbackWindow(targets, w, true)
	g.endCampaignEnemyHitPhase(g.arena)
}

func TestDroneHoversWithoutFallingAndStaysInArena(t *testing.T) {
	g, human := enemyTestGame()
	human.Active = false // nobody to chase: pure hover / patrol
	d := spawnTestEnemy(g, campaignEnemyDrone, 0)
	minX, maxX, top, _ := campaignArenaBounds(g.arena)
	for i := 0; i < 900; i++ {
		g.updateCampaignEnemies(g.arena)
	}
	if !d.alive() {
		t.Fatal("an untouched drone died")
	}
	b := &d.Body
	if b.Y > top+10 || b.Y < -200 || b.X < minX-40 || b.X > maxX+40 {
		t.Fatalf("drone drifted away: (%.1f, %.1f) arena x %.0f..%.0f top %.0f", b.X, b.Y, minX, maxX, top)
	}
	if math.Abs(b.VY) > droneMaxSpeed {
		t.Fatalf("drone is falling: vy=%.2f", b.VY)
	}
}

func TestDroneKeepsDistanceAndShootsAtPlayer(t *testing.T) {
	g, human := enemyTestGame()
	human.Lives, human.TotalLives = 99, 99
	d := spawnTestEnemy(g, campaignEnemyDrone, 0)
	shots := 0
	for i := 0; i < 600; i++ {
		before := len(g.bullets)
		g.updateCampaignEnemies(g.arena)
		for _, b := range g.bullets[before:] {
			if b.OwnerID != d.Body.ID || b.Team != campaignEnemyTeam {
				t.Fatalf("drone bullet credit owner=%d team=%d", b.OwnerID, b.Team)
			}
		}
		shots += len(g.bullets) - before
		g.bullets = g.bullets[:0]
		human.VX, human.VY = 0, 0
	}
	if shots == 0 {
		t.Fatal("drone never fired at the player")
	}
	dist := math.Abs(d.Body.X - human.X)
	if dist < droneKeepDistance*0.5 || dist > droneKeepDistance*1.6 {
		t.Fatalf("drone stand-off distance=%.1f want about %.0f", dist, droneKeepDistance)
	}
}

func TestDroneDiesOutOfBoundsWithKillCredit(t *testing.T) {
	g, human := enemyTestGame()
	d := spawnTestEnemy(g, campaignEnemyDrone, 0)
	d.Body.X, d.Body.Y = 700, 200
	d.Body.VX = 80
	d.Body.LastHitBy = human.ID
	for i := 0; i < 120 && d.alive(); i++ {
		g.updateCampaignEnemies(g.arena)
	}
	if d.alive() {
		t.Fatalf("drone survived a ring-out at x=%.0f", d.Body.X)
	}
	if human.Kills != 1 || human.Score != 100 {
		t.Fatalf("ring-out credit kills=%d score=%d", human.Kills, human.Score)
	}
	if len(g.killfeeds) != 1 || g.killfeeds[0].Killer != human.Name || g.killfeeds[0].Victim != "Drone" {
		t.Fatalf("killfeed=%+v", g.killfeeds)
	}
}

func TestDroneHPDestroysAfterHits(t *testing.T) {
	g, human := enemyTestGame()
	d := spawnTestEnemy(g, campaignEnemyDrone, 2)
	d.Body.X, d.Body.Y, d.Body.VX, d.Body.VY = 450, 200, 0, 0
	for i := 0; i < 2; i++ {
		cx, cy := droneCenter(&d.Body)
		g.bullets = append(g.bullets, Bullet{Kind: BulletNormal, X: cx - 30, Y: cy, VX: 25, Firepower: 4, OwnerID: human.ID, Team: 1, Facing: 1, Alpha: 1})
		runBulletHitPhase(g)
		g.bullets = g.bullets[:0]
		d.Body.HitTimer = 0
		d.Body.X, d.Body.VX = 450, 0
	}
	if d.alive() || d.HP != 0 {
		t.Fatalf("drone with HP2 survived two hits: hp=%d", d.HP)
	}
	if human.Kills != 1 {
		t.Fatalf("HP kill not credited: %d", human.Kills)
	}
}

// Shock clouds are the drone's weakness: the gas pushes it harder than a
// player (light frame) and stalls the rotors so it sinks.
func TestDroneIsPushedAndStalledByShockGas(t *testing.T) {
	g, human := enemyTestGame()
	human.Active = false
	d := spawnTestEnemy(g, campaignEnemyDrone, 0)
	d.Body.X, d.Body.Y, d.Body.VX, d.Body.VY = 450, 150, 0, 0
	ref := &Player{ID: 9, Active: true, X: 450, Y: 150, Team: 1}
	g.players = append(g.players, ref)
	cx, cy := droneCenter(&d.Body)
	g.areaEffects = append(g.areaEffects, AreaEffect{Kind: areaEffectGas, X: cx - 60, Y: cy, Radius: gasCloudRadius, MaxRadius: gasCloudRadius, OwnerID: human.ID, Life: gasCloudFrames, MaxLife: gasCloudFrames, Frame: gasGrowFrames})
	targets := g.hitTargets()
	w := beginAbilityKnockbackWindow(targets)
	updateAreaEffects(g.areaEffects, targets)
	endAbilityKnockbackWindow(targets, w, true)
	if d.Body.VX <= ref.VX || ref.VX <= 0 {
		t.Fatalf("shock push drone=%.3f player=%.3f", d.Body.VX, ref.VX)
	}
	if d.Body.LastHitBy != human.ID {
		t.Fatalf("shock gas credit=%d", d.Body.LastHitBy)
	}
	y0 := d.Body.Y
	g.players = g.players[:1]
	g.updateCampaignEnemies(g.arena)
	if d.Stall == 0 {
		t.Fatal("rotors not stalled inside the shock cloud")
	}
	for i := 0; i < 20; i++ {
		g.updateCampaignEnemies(g.arena)
	}
	if d.Body.Y <= y0+5 {
		t.Fatalf("stalled drone did not sink: y0=%.1f y=%.1f", y0, d.Body.Y)
	}
}

func shieldTestBearer(g *Game) *CampaignEnemy {
	e := spawnTestEnemy(g, campaignEnemyShieldBearer, 0)
	e.Body.X, e.Body.Y = 300, hazardLandingY(g.arena, 300)
	e.Body.VX, e.Body.VY = 0, 0
	e.Body.Facing = 1
	e.Body.JumpNum = 2
	return e
}

func TestShieldBearerDeflectsFrontBulletsOnly(t *testing.T) {
	g, human := enemyTestGame()
	e := shieldTestBearer(g)
	b := &e.Body
	y := b.Y - 35

	// Front: travelling left into the raised shield.
	g.bullets = append(g.bullets, Bullet{Kind: BulletNormal, X: b.X + 45, Y: y, VX: -25, Firepower: 20, OwnerID: human.ID, Team: 1, Facing: -1, Alpha: 1})
	runBulletHitPhase(g)
	if !g.bullets[0].Dead || b.HitNumber != 0 || math.Abs(b.VX) > 2 || e.ShieldSpark == 0 {
		t.Fatalf("front shot not deflected: dead=%v hits=%d vx=%.2f spark=%d", g.bullets[0].Dead, b.HitNumber, b.VX, e.ShieldSpark)
	}
	if b.LastHitBy != 0 {
		t.Fatalf("deflected shot gave credit: %d", b.LastHitBy)
	}

	// Back: travelling right into his back hits normally.
	g.bullets = g.bullets[:0]
	b.VX = 0
	g.bullets = append(g.bullets, Bullet{Kind: BulletNormal, X: b.X - 40, Y: y, VX: 25, Firepower: 20, OwnerID: human.ID, Team: 1, Facing: 1, Alpha: 1})
	runBulletHitPhase(g)
	if b.HitNumber != 1 || b.VX < 20*shieldBearerKnockbackScale*0.99 || b.LastHitBy != human.ID {
		t.Fatalf("back shot: hits=%d vx=%.2f credit=%d", b.HitNumber, b.VX, b.LastHitBy)
	}

	// Explosion right in front of the shield pushes him like anybody else.
	b.VX, b.VY = 0, 0
	gr := Grenade{X: b.X + 40, Y: b.Y - 20, OwnerID: human.ID, Type: ThrowableDynamite}
	targets := g.hitTargets()
	w := beginAbilityKnockbackWindow(targets)
	explodeGrenade(&gr, targets)
	endAbilityKnockbackWindow(targets, w, true)
	if b.VX > -10 {
		t.Fatalf("explosion barely moved the shield-bearer: vx=%.2f", b.VX)
	}

	// An explosive round striking the shield detonates on it.
	b.VX, b.VY = 0, 0
	g.bullets = g.bullets[:0]
	g.bullets = append(g.bullets, Bullet{Kind: BulletNormal, X: b.X + 45, Y: y, VX: -25, Firepower: 40, OwnerID: human.ID, Team: 1, Facing: -1,
		DevelopedProjectile: DevelopedProjectileRocket, ExplosionRadius: 150, ExplosionPower: 50, Lifetime: 60, Alpha: 1})
	runBulletHitPhase(g)
	if !g.bullets[0].Dead || b.VX > -5 {
		t.Fatalf("rocket on the shield: dead=%v vx=%.2f", g.bullets[0].Dead, b.VX)
	}
}

func TestShieldBearerLowersShieldWhenFlashedOrGassed(t *testing.T) {
	for _, tc := range []struct {
		name  string
		apply func(*Player)
	}{
		{"flashbang", func(p *Player) { applyBlind(p, 90, 1) }},
		{"tear gas", func(p *Player) { p.TearGasTime = tearGasLingerFrames }},
	} {
		g, human := enemyTestGame()
		e := shieldTestBearer(g)
		tc.apply(&e.Body)
		g.updateCampaignEnemies(g.arena)
		if e.shieldUp() {
			t.Fatalf("%s: shield still up", tc.name)
		}
		b := &e.Body
		b.VX = 0
		g.bullets = append(g.bullets[:0], Bullet{Kind: BulletNormal, X: b.X + 45, Y: b.Y - 35, VX: -25, Firepower: 20, OwnerID: human.ID, Team: 1, Facing: -1, Alpha: 1})
		runBulletHitPhase(g)
		if b.HitNumber == 0 || b.VX >= 0 {
			t.Fatalf("%s: front shot did not land on a lowered shield (hits=%d vx=%.2f)", tc.name, b.HitNumber, b.VX)
		}
	}
}

func TestShieldBearerWalksFallsAndDiesOffMap(t *testing.T) {
	g, human := enemyTestGame()
	human.X = 800
	e := shieldTestBearer(g)
	x0 := e.Body.X
	for i := 0; i < 120; i++ {
		g.updateCampaignEnemies(g.arena)
		human.VX, human.VY = 0, 0
	}
	if !e.alive() || e.Body.X <= x0+10 {
		t.Fatalf("bearer did not advance: x0=%.0f x=%.0f", x0, e.Body.X)
	}
	if !e.Body.Grounded && e.Body.JumpNum != 2 {
		t.Fatalf("bearer is not on the ground: y=%.1f", e.Body.Y)
	}
	// A huge shove off the side: falls and is destroyed with credit.
	e.Body.VX = -90
	e.Body.LastHitBy = human.ID
	for i := 0; i < 400 && e.alive(); i++ {
		g.updateCampaignEnemies(g.arena)
	}
	if e.alive() || human.Kills != 1 {
		t.Fatalf("ring-out alive=%v kills=%d", e.alive(), human.Kills)
	}
}

func TestShieldBearerShovesThroughKnockbackWindow(t *testing.T) {
	g, human := enemyTestGame()
	e := shieldTestBearer(g)
	human.X, human.Y = e.Body.X+40, e.Body.Y
	human.AbilityNumber = abilityAnchor
	human.AnchorTime = 30
	w := beginAbilityKnockbackWindow(g.players)
	g.shieldBearerShove(e, human)
	endAbilityKnockbackWindow(g.players, w, true)
	if human.VX != 0 || human.LastHitBy != e.Body.ID {
		t.Fatalf("anchored player moved by shove: vx=%.2f credit=%d", human.VX, human.LastHitBy)
	}
	human.AbilityNumber, human.AnchorTime = abilityNone, 0
	g.shieldBearerShove(e, human)
	if human.VX <= 0 {
		t.Fatalf("shove did not push: %.2f", human.VX)
	}
}

func TestCampaignEnemiesCountTowardWinCondition(t *testing.T) {
	g, human := enemyTestGame()
	g.setupCampaignEnemyWaves([]extraCampaignUnit{{Kind: campaignEnemyDrone, Count: 2, MaxActive: 1}})
	g.updateMatchInteractions()
	if g.GameWin || g.teamGameWin {
		t.Fatal("won while enemies were still queued")
	}
	if got := g.campaignEnemiesRemaining(); got != 2 {
		t.Fatalf("remaining=%d want2", got)
	}
	g.updateCampaignEnemies(g.arena)
	if len(g.campaignEnemies) != 1 || g.campaignEnemiesRemaining() != 2 {
		t.Fatalf("wave spawn enemies=%d remaining=%d", len(g.campaignEnemies), g.campaignEnemiesRemaining())
	}
	g.destroyCampaignEnemy(g.campaignEnemies[0])
	g.updateMatchInteractions()
	if g.GameWin || g.campaignEnemiesRemaining() != 1 {
		t.Fatalf("won with a reinforcement left: remaining=%d", g.campaignEnemiesRemaining())
	}
	for i := 0; i < 400 && g.campaignEnemiesAlive() == 0; i++ {
		g.updateCampaignEnemies(g.arena) // the reinforcement arrives
	}
	if g.campaignEnemiesAlive() != 1 {
		t.Fatalf("reinforcement did not arrive: alive=%d", g.campaignEnemiesAlive())
	}
	for _, e := range g.campaignEnemies {
		if e.alive() {
			g.destroyCampaignEnemy(e)
		}
	}
	g.updateMatchInteractions()
	if !g.GameWin || g.winnerPlayerID != human.ID {
		t.Fatalf("all enemies down but no win: win=%v winner=%d remaining=%d", g.GameWin, g.winnerPlayerID, g.campaignEnemiesRemaining())
	}

	// The human losing every life with enemies left is a defeat.
	g2, human2 := enemyTestGame()
	spawnTestEnemy(g2, campaignEnemyDrone, 0)
	human2.Active = false
	g2.updateMatchInteractions()
	if !g2.teamGameWin || g2.campaignLoseFrame != 1 {
		t.Fatalf("defeat not detected: teamwin=%v lose=%d", g2.teamGameWin, g2.campaignLoseFrame)
	}
}

func TestEnemyBodiesAreNotPlayers(t *testing.T) {
	g, _ := enemyTestGame()
	spawnTestEnemy(g, campaignEnemyDrone, 0)
	spawnTestEnemy(g, campaignEnemyShieldBearer, 0)
	if len(g.players) != 1 {
		t.Fatalf("enemies leaked into g.players: %d", len(g.players))
	}
	if n := len(g.hitTargets()); n != 3 {
		t.Fatalf("hit targets=%d want3", n)
	}
	for _, e := range g.campaignEnemies {
		if e.Body.ID < campaignEnemyIDBase || e.Body.Team != campaignEnemyTeam {
			t.Fatalf("enemy body id=%d team=%d", e.Body.ID, e.Body.Team)
		}
	}
}

func TestCampaignEnemiesSurviveNetSnapshot(t *testing.T) {
	g, _ := enemyTestGame()
	d := spawnTestEnemy(g, campaignEnemyDrone, 4)
	d.Tilt = 0.25
	g.extraHazardMarkers = append(g.extraHazardMarkers, HazardMarker{Kind: hazardMarkerDynamite, X: 300, Y: 250, Life: 40, MaxLife: 100})
	state := netSnapshot{CampaignEnemies: g.campaignEnemySnapshot(), HazardMarkers: append([]HazardMarker(nil), g.extraHazardMarkers...)}
	data := netEncodeValue(nil, &state)
	var got netSnapshot
	if _, err := netDecodeValue(data, &got); err != nil {
		t.Fatal(err)
	}
	client := &Game{}
	client.applyCampaignEnemySnapshot(got.CampaignEnemies)
	if len(client.campaignEnemies) != 1 || client.campaignEnemies[0].Body.ID != d.Body.ID || client.campaignEnemies[0].HP != 4 ||
		math.Abs(client.campaignEnemies[0].Tilt-0.25) > 1e-6 || len(got.HazardMarkers) != 1 {
		t.Fatalf("snapshot round trip lost enemy state: %+v markers=%d", client.campaignEnemies, len(got.HazardMarkers))
	}
}

// --- bosses ---------------------------------------------------------------

func TestExtraBossPhasesAndKnockbackResistance(t *testing.T) {
	g := startExtraMissionForTest(t, 7)
	var boss *Player
	for _, p := range g.players {
		if p.ID == 3 {
			boss = p
		}
	}
	spec := g.extraBossSpec(boss)
	if spec == nil || boss.KnockbackScale != spec.KnockbackScale || boss.BossPhase != 0 {
		t.Fatalf("boss base not applied: spec=%v kb=%.2f phase=%d", spec, boss.KnockbackScale, boss.BossPhase)
	}

	// Resistance covers bullets through the shared knockback window.
	normal := &Player{ID: 4, Active: true, X: 600, Y: 300, Team: 2, DamageMulti: 1}
	boss.X, boss.Y, boss.VX, boss.VY = 300, 300, 0, 0
	targets := []*Player{boss, normal}
	bullets := []Bullet{
		{Kind: BulletNormal, X: 290, Y: 270, VX: 25, Firepower: 20, OwnerID: 1, Team: 1, Facing: 1},
		{Kind: BulletNormal, X: 590, Y: 270, VX: 25, Firepower: 20, OwnerID: 1, Team: 1, Facing: 1},
	}
	w := beginAbilityKnockbackWindow(targets)
	updateBullets(g.arena, bullets, targets, &g.instagibTrails, &g.explosions, &g.blastFX, &g.portals)
	endAbilityKnockbackWindow(targets, w, true)
	scale := abilityKnockbackScale(boss) / abilityKnockbackScale(normal)
	if normal.VX <= 0 || math.Abs(boss.VX/normal.VX-scale) > 0.01 || scale >= 1 {
		t.Fatalf("boss push=%.2f normal=%.2f want ratio %.2f", boss.VX, normal.VX, scale)
	}
	// ... and blasts too.
	boss.VX, normal.VX = 0, 0
	w = beginAbilityKnockbackWindow(targets)
	explodeGrenade(&Grenade{X: 260, Y: 270, Type: ThrowableDynamite, OwnerID: 1}, []*Player{boss})
	explodeGrenade(&Grenade{X: 560, Y: 270, Type: ThrowableDynamite, OwnerID: 1}, []*Player{normal})
	endAbilityKnockbackWindow(targets, w, true)
	if boss.VX >= normal.VX {
		t.Fatalf("blast resistance missing: boss=%.2f normal=%.2f", boss.VX, normal.VX)
	}

	// Phases by remaining lives; skipped phases still run in order.
	boss.Lives = 8
	g.updateExtraCampaignBosses()
	if boss.BossPhase != 1 || boss.CampaignMoveScale != spec.Phases[0].MoveScale || g.bossBannerFrame == 0 {
		t.Fatalf("phase1 phase=%d move=%.2f banner=%d", boss.BossPhase, boss.CampaignMoveScale, g.bossBannerFrame)
	}
	boss.Lives = 2
	g.updateExtraCampaignBosses()
	if boss.BossPhase != 2 || boss.KnockbackScale != spec.Phases[1].KnockbackScale {
		t.Fatalf("phase2 phase=%d kb=%.2f", boss.BossPhase, boss.KnockbackScale)
	}
	if n := len(g.campaignEnemies); n != spec.Phases[1].SummonDrones {
		t.Fatalf("phase2 summoned %d drones", n)
	}
	if g.campaignEnemiesRemaining() == 0 {
		t.Fatal("summoned drones do not count toward the win condition")
	}
}

func TestExtraBossesHaveRealBossData(t *testing.T) {
	bosses := 0
	for i, m := range extraCampaignMissions {
		for _, e := range m.Enemies {
			if e.Boss == nil {
				continue
			}
			bosses++
			if e.Boss.Title == "" || e.Boss.KnockbackScale <= 0 || e.Boss.KnockbackScale >= 1 || len(e.Boss.Phases) == 0 {
				t.Fatalf("mission %d boss %s has no real boss data: %+v", i+1, e.Name, e.Boss)
			}
			last := e.Lives
			for _, ph := range e.Boss.Phases {
				if ph.Lives >= last || ph.Lives <= 0 {
					t.Fatalf("mission %d boss %s phases must count down: %+v", i+1, e.Name, e.Boss.Phases)
				}
				last = ph.Lives
			}
		}
	}
	for _, level := range []int{7, 9, 11, 16, 20, 25} {
		if extraCampaignMissions[level-1].Enemies[0].Boss == nil {
			t.Fatalf("mission %d boss lost its boss data", level)
		}
	}
	if bosses < 6 {
		t.Fatalf("bosses=%d", bosses)
	}
}

func TestNewEnemyMissionsLoad(t *testing.T) {
	drones, shields, mixed := 0, 0, 0
	for level := 21; level <= extraCampaignMissionCount; level++ {
		g := startExtraMissionForTest(t, level)
		spec := extraCampaignMissions[level-1]
		if len(spec.Units) == 0 || g.campaignEnemiesRemaining() != extraCampaignUnitCount(spec) {
			t.Fatalf("mission %d waves=%d remaining=%d", level, len(spec.Units), g.campaignEnemiesRemaining())
		}
		kinds := map[campaignEnemyKind]bool{}
		for _, u := range spec.Units {
			kinds[u.Kind] = true
		}
		switch {
		case kinds[campaignEnemyDrone] && kinds[campaignEnemyShieldBearer]:
			mixed++
		case kinds[campaignEnemyDrone]:
			drones++
		case kinds[campaignEnemyShieldBearer]:
			shields++
		}
		if len(extraCampaignOpponentLines(spec)) > 3 {
			t.Fatalf("mission %d opponent list overflows the panel", level)
		}
	}
	if drones < 2 || shields < 2 || mixed < 1 {
		t.Fatalf("new missions drones=%d shields=%d mixed=%d", drones, shields, mixed)
	}
}

func TestExtraSaveFromTwentyMissionsUnlocksNewOnes(t *testing.T) {
	var levels [extraCampaignMissionCount]int
	for i := 0; i < 20; i++ {
		levels[i] = 2
	}
	levels = normalizeExtraCampaignLevels(levels)
	if levels[20] != 1 || levels[21] != 0 {
		t.Fatalf("old save upgrade: %v", levels)
	}
}

func (g *Game) campaignEnemiesAlive() int {
	n := 0
	for _, e := range g.campaignEnemies {
		if e.alive() {
			n++
		}
	}
	return n
}
