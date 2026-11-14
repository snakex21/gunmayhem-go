package game

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

// --- helpers -----------------------------------------------------------------------

// newZombieTestGame starts a zombie match headlessly. Humans are driven as
// "remote" players with empty input so the developer's keyboard never leaks
// into a test.
func newZombieTestGame(t *testing.T, arena int, slots [4]int) *Game {
	t.Helper()
	g := New()
	g.zombieSetup = zombieSetupConfig{Mode: zombieModeSurvival, Arena: arena, Slots: slots}
	if !g.startZombieMatch() || g.zombie == nil {
		t.Fatal("zombie match did not start")
	}
	g.screen = screenGameplay
	g.fadeActive = false
	g.fadeFrame = 0
	for _, p := range g.players {
		if !p.AI {
			p.RemoteControlled = true
			g.zombie.hostPlayer(p.ID).remote = true
		}
	}
	return g
}

// holdZombieWave keeps the match in a running wave that never ends by itself.
func holdZombieWave(g *Game) {
	m := g.zombie
	m.hud.Phase = zombiePhaseWave
	m.plan = zombieWavePlanFor(3, g.zombieParticipants())
	m.toSpawn = 1
	m.spawnTimer = 1 << 30
}

func spawnTestZombie(g *Game, kind zombieKind, x, y float64, hp int) *Zombie {
	z := g.newZombie(kind, hp, zombiePoint{x, y}, 0)
	z.State = zombieStateWalk
	return z
}

// placePlayer puts p standing still on a platform top.
func placePlayer(p *Player, x, y float64) {
	p.X, p.Y, p.VX, p.VY = x, y, 0, 0
	p.JumpNum, p.Grounded = 2, true
}

func testBullet(x, y float64, dir int, firepower float64) Bullet {
	return Bullet{Kind: BulletNormal, X: x, Y: y, VX: 25 * float64(dir), Firepower: firepower, OwnerID: 1, Team: 1,
		Facing: dir, Time: 1, Alpha: 1, TrailWidth: 20, TrailFrame: 1}
}

func updateN(t *testing.T, g *Game, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
}

func shopItemFor(t *testing.T, kind, value int) zombieShopItem {
	t.Helper()
	for _, it := range zombieShopItems() {
		if it.Kind == kind && (kind != zombieItemWeapon && kind != zombieItemThrowable && kind != zombieItemGear || it.Value == value) {
			return it
		}
	}
	t.Fatalf("no shop item kind=%d value=%d", kind, value)
	return zombieShopItem{}
}

func openZombieBreak(g *Game) {
	m := g.zombie
	m.hud.Phase = zombiePhaseBreak
	m.hud.PhaseTimer = zombieBreakFrames
	for i := range m.hud.Players {
		if m.hud.Players[i].ID != 0 {
			m.hud.Players[i].ShopOpen = true
			m.hud.Players[i].Ready = false
		}
	}
}

// --- waves -----------------------------------------------------------------------------

func TestZombieWavePlanScaling(t *testing.T) {
	prev := zombieWavePlanFor(1, 1)
	if prev.Count < 4 || prev.Boss || prev.Weights[zombieWalker] == 0 || prev.Weights[zombieRunner] != 0 {
		t.Fatalf("wave 1 plan: %+v", prev)
	}
	for wave := 2; wave <= 30; wave++ {
		p := zombieWavePlanFor(wave, 1)
		if p.HPMul <= prev.HPMul || p.SpeedMul < prev.SpeedMul || p.SpeedMul > 1.35 {
			t.Fatalf("wave %d: HP/speed do not scale: %+v after %+v", wave, p, prev)
		}
		if p.MaxActive > zombieMaxActive || p.Count > zombieMaxPerWave || p.Interval < 10 {
			t.Fatalf("wave %d: caps broken: %+v", wave, p)
		}
		if p.Boss != (wave%5 == 0) || (p.Boss && (p.BossLevel != wave/5 || p.BossHP <= 0)) {
			t.Fatalf("wave %d: boss rule broken: %+v", wave, p)
		}
		if !p.Boss && !prev.Boss && p.Count < prev.Count {
			t.Fatalf("wave %d: fewer zombies than before (%d < %d)", wave, p.Count, prev.Count)
		}
		for k := zombieWalker; k < zombieBoss; k++ {
			if (wave >= zombieStats[k].MinWave) != (p.Weights[k] > 0) {
				t.Fatalf("wave %d: %s weight %d (min wave %d)", wave, zombieStats[k].Name, p.Weights[k], zombieStats[k].MinWave)
			}
		}
		prev = p
	}
	if zombieWavePlanFor(10, 1).BossHP >= zombieWavePlanFor(15, 1).BossHP {
		t.Fatal("bosses do not escalate")
	}
	solo, four := zombieWavePlanFor(8, 1), zombieWavePlanFor(8, 4)
	if four.Count <= solo.Count || four.HPMul <= solo.HPMul || four.MaxActive <= solo.MaxActive {
		t.Fatalf("co-op does not scale: solo %+v four %+v", solo, four)
	}
	// The weighted pick covers every kind of the mix.
	seen := map[zombieKind]bool{}
	p := zombieWavePlanFor(12, 2)
	for i := 0; i < 1000; i++ {
		seen[p.pick(float64(i)/1000)] = true
	}
	for k := zombieWalker; k < zombieBoss; k++ {
		if !seen[k] {
			t.Fatalf("wave 12 never picks %s", zombieStats[k].Name)
		}
	}
}

// --- AI / pathing ----------------------------------------------------------------------------

func TestZombieNavReachesEveryPlatform(t *testing.T) {
	for ai := range zombieArenas {
		a := &zombieArenas[ai]
		arena, err := LoadOriginalMap(a.Zones[0].MapNumber)
		if err != nil {
			t.Fatal(err)
		}
		for class := zombieNavNormal; class <= zombieNavHeavy; class++ {
			nav := buildZombieNav(arena, class)
			for _, sp := range a.Zones[0].Spawns {
				from := nav.platformUnder(sp.X, sp.Y)
				if from < 0 {
					t.Fatalf("%s: spawn %v is not on a platform", a.Name, sp)
				}
				for to := range nav.plats {
					if !nav.reachable(from, to) {
						t.Fatalf("%s class %d: platform %d unreachable from spawn %v", a.Name, class, to, sp)
					}
				}
			}
			for _, b := range a.Zones[0].BoxSpots {
				if nav.platformUnder(b.X, b.Y) < 0 {
					t.Fatalf("%s: box spot %v is not on a platform", a.Name, b)
				}
			}
		}
	}
}

// TestZombieAIReachesPlayersOnEveryArena simulates every zombie kind from
// every spawn point toward a player standing on each platform of each arena.
// Every zombie must reach the player within 30 s without being re-spawned by
// the anti-stuck rule.
func TestZombieAIReachesPlayersOnEveryArena(t *testing.T) {
	kinds := []zombieKind{zombieWalker, zombieRunner, zombieBrute, zombieExploder, zombieLeaper}
	for ai := range zombieArenas {
		g := newZombieTestGame(t, ai, [4]int{1, 0, 0, 0})
		m := g.zombie
		p := g.players[0]
		nav := m.navFor(0, g.arena, zombieNavNormal)
		for pi, r := range nav.plats {
			m.zombies = nil
			m.stuckRespawns = 0
			holdZombieWave(g)
			px := r.X + r.W/2
			for _, sp := range m.zone().Spawns {
				for _, k := range kinds {
					spawnTestZombie(g, k, sp.X, sp.Y, 1<<20)
				}
			}
			reached := map[int]bool{}
			total := len(m.zombies)
			for f := 0; f < 900 && len(reached) < total; f++ {
				zp := m.playerState(1)
				zp.Invuln, zp.HP = 1000, zombiePlayerHP
				placePlayer(p, px, r.Y)
				g.updateZombies(g.arena)
				for _, z := range m.zombies {
					if z.Dead || z.kind() == zombieExploder && z.State == zombieStateFuse || zombieInReach(z, p, 20) {
						reached[z.ID] = true
					}
				}
			}
			for _, z := range m.zombies {
				if !reached[z.ID] {
					t.Errorf("%s: player on platform %d (%.0f,%.0f): %s stuck at %.0f,%.0f", zombieArenas[ai].Name, pi, px, r.Y, zombieStats[z.kind()].Name, z.X, z.Y)
				}
			}
			if m.stuckRespawns > 0 {
				t.Errorf("%s platform %d: %d zombies needed the anti-stuck re-spawn", zombieArenas[ai].Name, pi, m.stuckRespawns)
			}
		}
	}
}

func TestZombieBossReachesAndSlams(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	m := g.zombie
	holdZombieWave(g)
	p := g.players[0]
	boss := spawnTestZombie(g, zombieBoss, 835, 151, 5000)
	boss.Level = 1
	zp := m.playerState(1)
	hit := false
	for f := 0; f < 1200 && !hit; f++ {
		zp.Invuln = 0
		zp.HP = zombiePlayerHP
		placePlayer(p, 300, 511)
		g.updateZombies(g.arena)
		hit = zp.HP < zombiePlayerHP
	}
	if !hit {
		t.Fatalf("the boss never hurt the player (boss at %.0f,%.0f)", boss.X, boss.Y)
	}
	if m.livingZombies() <= 1 {
		// The boss summons adds within the first summon period.
		for f := 0; f < zombieBossSummonEvery+10; f++ {
			placePlayer(p, 300, 511)
			zp.Invuln = 1000
			g.updateZombies(g.arena)
		}
		if m.livingZombies() <= 1 {
			t.Fatal("the boss never summoned")
		}
	}
}

// --- damage, points ----------------------------------------------------------------------------

func TestZombieBulletDamageAndPoints(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	holdZombieWave(g)
	m := g.zombie
	p := g.players[0]
	placePlayer(p, 200, 511)
	z := spawnTestZombie(g, zombieWalker, 500, 511, 100)
	zp := m.playerState(1)
	before := zp.Points
	g.bullets = append(g.bullets, testBullet(470, 511-20, 1, 20))
	updateN(t, g, 1)
	if z.HP != 80 {
		t.Fatalf("body hit: HP=%d want 80", z.HP)
	}
	if zp.Points != before+zombiePointsHit {
		t.Fatalf("hit points: %d want %d", zp.Points, before+zombiePointsHit)
	}
	// Headshot: +50% damage.
	g.bullets = append(g.bullets, testBullet(z.X-30, z.Y-50, 1, 20))
	updateN(t, g, 1)
	if z.HP != 50 {
		t.Fatalf("headshot: HP=%d want 50", z.HP)
	}
	// Kill with a headshot: kill points + headshot bonus.
	pts := zp.Points
	g.bullets = append(g.bullets, testBullet(z.X-30, z.Y-50, 1, 60))
	updateN(t, g, 1)
	if !z.Dead || zp.Kills != 1 || zp.Headshots != 1 {
		t.Fatalf("kill: dead=%v kills=%d headshots=%d hp=%d", z.Dead, zp.Kills, zp.Headshots, z.HP)
	}
	want := pts + zombiePointsHit + zombieStats[zombieWalker].KillPoints + zombiePointsHeadshotBonus
	if zp.Points != want {
		t.Fatalf("kill points: %d want %d", zp.Points, want)
	}
}

func TestZombieBruteResistsKnockbackButTakesFullDamage(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	holdZombieWave(g)
	placePlayer(g.players[0], 150, 511)
	walker := spawnTestZombie(g, zombieWalker, 400, 511, 500)
	brute := spawnTestZombie(g, zombieBrute, 600, 511, 500)
	g.bullets = append(g.bullets, testBullet(370, 511-20, 1, 40), testBullet(560, 511-25, 1, 40))
	updateN(t, g, 1)
	if walker.HP != 460 || brute.HP != 460 {
		t.Fatalf("damage walker=%d brute=%d, both want 460", walker.HP, brute.HP)
	}
	if brute.body.VX >= walker.body.VX {
		t.Fatalf("brute pushed as far as a walker: %.2f vs %.2f", brute.body.VX, walker.body.VX)
	}
}

func TestZombieRingOutKillBonus(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	holdZombieWave(g)
	m := g.zombie
	placePlayer(g.players[0], 600, 511)
	z := spawnTestZombie(g, zombieWalker, 140, 511, 10000)
	zp := m.playerState(1)
	pts := zp.Points
	g.bullets = append(g.bullets, testBullet(170, 511-20, -1, 400))
	for i := 0; i < 120 && !z.Dead; i++ {
		updateN(t, g, 1)
	}
	if !z.Dead || !z.RingOut {
		t.Fatalf("zombie not rung out: dead=%v ringout=%v at %.0f,%.0f", z.Dead, z.RingOut, z.X, z.Y)
	}
	want := pts + zombiePointsHit + zombieStats[zombieWalker].KillPoints + zombiePointsRingOutBonus
	if zp.Points != want || zp.Kills != 1 {
		t.Fatalf("ring-out points %d want %d (kills %d)", zp.Points, want, zp.Kills)
	}
}

func TestZombieExploderChainAndPlayerDamage(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	holdZombieWave(g)
	m := g.zombie
	p := g.players[0]
	placePlayer(p, 200, 511)
	a := spawnTestZombie(g, zombieExploder, 400, 511, 5)
	b := spawnTestZombie(g, zombieExploder, 470, 511, 40)
	g.bullets = append(g.bullets, testBullet(370, 511-20, 1, 30))
	for i := 0; i < 15; i++ {
		placePlayer(p, 200, 511)
		updateN(t, g, 1)
	}
	if !a.Dead || !b.Dead {
		t.Fatalf("chain reaction: first dead=%v second dead=%v (hp %d)", a.Dead, b.Dead, b.HP)
	}
	if zp := m.playerState(1); zp.Kills != 2 {
		t.Fatalf("chain kill credit: kills=%d want 2", zp.Kills)
	}
	// An exploder that reaches a player blows up on its own and hurts.
	c := spawnTestZombie(g, zombieExploder, 230, 511, 50)
	zp := m.playerState(1)
	zp.HP, zp.Invuln = zombiePlayerHP, 0
	for i := 0; i < 90 && !c.Dead; i++ {
		placePlayer(p, 200, 511)
		updateN(t, g, 1)
	}
	updateN(t, g, 2)
	if !c.Dead || zp.HP >= zombiePlayerHP {
		t.Fatalf("exploder fuse: dead=%v player HP=%d", c.Dead, zp.HP)
	}
}

func TestZombieMeleeHurtsAndKnocksBack(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	holdZombieWave(g)
	m := g.zombie
	p := g.players[0]
	spawnTestZombie(g, zombieWalker, 330, 511, 1000)
	zp := m.playerState(1)
	for i := 0; i < 60 && zp.HP == zombiePlayerHP; i++ {
		placePlayer(p, 300, 511)
		updateN(t, g, 1)
	}
	if zp.HP != zombiePlayerHP-zombieStats[zombieWalker].Damage {
		t.Fatalf("walker hit: HP=%d", zp.HP)
	}
	if p.VX >= 0 {
		t.Fatalf("no knockback away from the zombie: vx=%.2f", p.VX)
	}
}

// --- down / revive ----------------------------------------------------------------------------

func TestZombieDownReviveBleedOutAndGameOver(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 1, 0, 0})
	holdZombieWave(g)
	m := g.zombie
	p1, p2 := g.players[0], g.players[1]
	placePlayer(p1, 300, 511)
	placePlayer(p2, 600, 511)
	zp1, zp2 := m.playerState(1), m.playerState(2)
	if zp1.SelfRevives != 0 {
		t.Fatal("co-op players must not start with a self-revive")
	}
	zp1.HP = 0
	updateN(t, g, 1)
	if zp1.State != zombiePlayerDown || p1.Active {
		t.Fatalf("not down: state=%d active=%v", zp1.State, p1.Active)
	}
	pts := zp2.Points
	for i := 0; i < zombieReviveFrames+2 && zp1.State == zombiePlayerDown; i++ {
		placePlayer(p2, zp1.DownX+20, zp1.DownY)
		updateN(t, g, 1)
	}
	if zp1.State != zombiePlayerAlive || !p1.Active || zp1.HP != zombieReviveHP {
		t.Fatalf("revive failed: state=%d active=%v hp=%d", zp1.State, p1.Active, zp1.HP)
	}
	if zp2.Revives != 1 || zp2.Points != pts+zombiePointsRevive {
		t.Fatalf("reviver credit: revives=%d points=%d", zp2.Revives, zp2.Points)
	}
	// Moving away interrupts a revive.
	zp1.HP = 0
	updateN(t, g, 1)
	placePlayer(p2, zp1.DownX+20, zp1.DownY)
	updateN(t, g, 30)
	placePlayer(p2, 800, 511)
	updateN(t, g, 2)
	if zp1.ReviveProgress != 0 {
		t.Fatalf("revive progress kept after the reviver left: %d", zp1.ReviveProgress)
	}
	// Nobody comes: bleed out.
	for i := 0; i < zombieBleedFrames+5 && zp1.State == zombiePlayerDown; i++ {
		placePlayer(p2, 800, 511)
		updateN(t, g, 1)
	}
	if zp1.State != zombiePlayerDead {
		t.Fatalf("bleed-out: state=%d timer=%d", zp1.State, zp1.DownTimer)
	}
	if m.hud.Phase != zombiePhaseWave {
		t.Fatalf("game over while a player still stands: phase=%d", m.hud.Phase)
	}
	// Last player down: game over.
	zp2.HP = 0
	updateN(t, g, 1)
	if m.hud.Phase != zombiePhaseOver {
		t.Fatalf("everybody down but phase=%d", m.hud.Phase)
	}
}

func TestZombieRingOutDownsAtTheEdge(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 1, 0, 0})
	holdZombieWave(g)
	m := g.zombie
	p := g.players[0]
	placePlayer(p, 130, 511)
	updateN(t, g, 2)
	p.VX = -40
	for i := 0; i < 90 && m.playerState(1).State == zombiePlayerAlive; i++ {
		updateN(t, g, 1)
	}
	zp := m.playerState(1)
	if zp.State != zombiePlayerDown {
		t.Fatalf("ring-out did not down the player: state=%d", zp.State)
	}
	if zp.DownX < 108 || zp.DownX > 200 || math.Abs(zp.DownY-511) > 2 {
		t.Fatalf("down marker not at the last ground position: %.0f,%.0f", zp.DownX, zp.DownY)
	}
}

func TestZombieSoloSelfRevive(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	holdZombieWave(g)
	m := g.zombie
	placePlayer(g.players[0], 400, 511)
	zp := m.playerState(1)
	if zp.SelfRevives != zombieSoloSelfRevives {
		t.Fatalf("solo self-revives=%d", zp.SelfRevives)
	}
	zp.HP = 0
	updateN(t, g, zombieReviveFrames+3)
	if zp.State != zombiePlayerAlive || zp.SelfRevives != 0 {
		t.Fatalf("self-revive: state=%d tokens=%d", zp.State, zp.SelfRevives)
	}
	zp.HP = 0
	updateN(t, g, 2)
	if m.hud.Phase != zombiePhaseOver {
		t.Fatalf("solo without tokens: phase=%d", m.hud.Phase)
	}
}

func TestZombieWaveEndRevivesAndRespawns(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 1, 1, 0})
	holdZombieWave(g)
	m := g.zombie
	for _, p := range g.players {
		placePlayer(p, 300+float64(p.ID)*100, 511)
	}
	m.playerState(2).HP = 0
	updateN(t, g, 1)
	m.playerState(3).HP = 0
	updateN(t, g, 1)
	m.playerState(3).DownTimer = 1
	updateN(t, g, 2)
	if m.playerState(2).State != zombiePlayerDown || m.playerState(3).State != zombiePlayerDead {
		t.Fatalf("setup: states %d %d", m.playerState(2).State, m.playerState(3).State)
	}
	m.toSpawn = 0
	updateN(t, g, 2)
	if m.hud.Phase != zombiePhaseBreak {
		t.Fatalf("wave did not end: phase=%d", m.hud.Phase)
	}
	for id := 1; id <= 3; id++ {
		zp := m.playerState(id)
		if zp.State != zombiePlayerAlive || !g.playerByID(id).Active || !zp.ShopOpen {
			t.Fatalf("player %d after the wave: state=%d shop=%v", id, zp.State, zp.ShopOpen)
		}
	}
}

// --- shop ---------------------------------------------------------------------------------------

func TestZombieShopRules(t *testing.T) {
	g := newZombieTestGame(t, 0, [4]int{1, 0, 0, 0})
	m := g.zombie
	p := g.players[0]
	zp := m.playerState(1)
	pistol := shopItemFor(t, zombieItemWeapon, 5)
	if pistol.Price != 350 {
		t.Fatalf("FIFTY EIGHT price=%d", pistol.Price)
	}
	if code := g.zombieBuy(p, pistol.ID); code != zombieMsgClosed {
		t.Fatalf("bought outside the break: %d", code)
	}
	openZombieBreak(g)
	zp.Points = 500
	if code := g.zombieBuy(p, pistol.ID); code != zombieMsgBought {
		t.Fatalf("buy: %d", code)
	}
	if zp.Points != 150 || zp.Gun != 5 || p.Weapon.Def.Number != 5 || zp.Reserve != zombieReserveMags(5) {
		t.Fatalf("after buy: points=%d gun=%d weapon=%d reserve=%d", zp.Points, zp.Gun, p.Weapon.Def.Number, zp.Reserve)
	}
	if code := g.zombieBuy(p, pistol.ID); code != zombieMsgOwned {
		t.Fatalf("second buy: %d", code)
	}
	mini := shopItemFor(t, zombieItemWeapon, 55)
	if code := g.zombieBuy(p, mini.ID); code != zombieMsgNoPoints || zp.Points != 150 {
		t.Fatalf("minigun without points: code=%d points=%d", code, zp.Points)
	}
	// Throwables stack to the cap; another type replaces.
	zp.Points = 10000
	dyn := shopItemFor(t, zombieItemThrowable, ThrowableDynamite)
	p.ThrowableType, p.Grenades = ThrowableDynamite, 3
	g.zombieBuy(p, dyn.ID)
	g.zombieBuy(p, dyn.ID)
	if p.Grenades != zombieMaxGrenades {
		t.Fatalf("grenades=%d", p.Grenades)
	}
	if code := g.zombieBuy(p, dyn.ID); code != zombieMsgFull {
		t.Fatalf("grenade cap: %d", code)
	}
	molo := shopItemFor(t, zombieItemThrowable, ThrowableMolotov)
	g.zombieBuy(p, molo.ID)
	if p.ThrowableType != ThrowableMolotov || p.Grenades != zombieThrowableStep {
		t.Fatalf("type switch: type=%d n=%d", p.ThrowableType, p.Grenades)
	}
	// Gear uses the existing ability system.
	shield := shopItemFor(t, zombieItemGear, abilityShield)
	g.zombieBuy(p, shield.ID)
	if p.AbilityNumber != abilityShield || g.zombieBuy(p, shield.ID) != zombieMsgOwned {
		t.Fatalf("gear: ability=%d", p.AbilityNumber)
	}
	// Supplies.
	heal := shopItemFor(t, zombieItemHeal, 0)
	if g.zombieBuy(p, heal.ID) != zombieMsgFull {
		t.Fatal("medkit at full HP")
	}
	zp.HP = 20
	g.zombieBuy(p, heal.ID)
	if zp.HP != zp.MaxHP {
		t.Fatalf("medkit: HP=%d", zp.HP)
	}
	ammo := shopItemFor(t, zombieItemAmmo, 0)
	zp.Reserve = 0
	p.Weapon.Bullets = 2
	pts := zp.Points
	g.zombieBuy(p, ammo.ID)
	if zp.Reserve != zombieReserveMags(5) || p.Weapon.Bullets < p.Weapon.Def.Bullets || zp.Points != pts-zombieAmmoPrice(5) {
		t.Fatalf("ammo: reserve=%d bullets=%d points=%d", zp.Reserve, p.Weapon.Bullets, zp.Points)
	}
	token := shopItemFor(t, zombieItemToken, 0)
	for i := 0; i < 5; i++ {
		g.zombieBuy(p, token.ID)
	}
	if zp.SelfRevives != zombieMaxSelfRevives {
		t.Fatalf("tokens=%d", zp.SelfRevives)
	}
	// A downed player cannot shop.
	zp.State = zombiePlayerDown
	if g.zombieBuy(p, heal.ID) != zombieMsgDown {
		t.Fatal("downed player bought")
	}
}

func TestZombieShopCatalogIsValid(t *testing.T) {
	ids := map[int]bool{}
	tabs := [zombieTabCount]int{}
	for i, it := range zombieShopItems() {
		if it.ID != i || ids[it.ID] {
			t.Fatalf("item id %d at %d", it.ID, i)
		}
		ids[it.ID] = true
		tabs[it.Tab]++
		if it.Name == "" || (it.Kind != zombieItemAmmo && it.Price <= 0) {
			t.Fatalf("bad item %+v", it)
		}
		if it.Kind == zombieItemWeapon {
			if _, ok := WeaponByNumber(it.Value); !ok {
				t.Fatalf("shop weapon %d does not exist", it.Value)
			}
		}
		if it.Kind == zombieItemGear && (it.Value <= abilityNone || it.Value >= abilityCount) {
			t.Fatalf("shop gear %d", it.Value)
		}
	}
	for tab, n := range tabs {
		if n == 0 {
			t.Fatalf("empty shop tab %s", zombieTabLabels[tab])
		}
	}
	if len(zombieShopItems()) >= netZombieItemLimit {
		t.Fatal("shop item IDs exceed the wire limit")
	}
	if len(zombieMysteryPool()) < 40 {
		t.Fatalf("mystery box pool only %d weapons", len(zombieMysteryPool()))
	}
	for _, n := range zombieMysteryPool() {
		if isStarterHandgunNumber(n) {
			t.Fatalf("box gives a starter pistol %d", n)
		}
	}
}

func TestZombieBoughtGunUsesSpareMagazines(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	holdZombieWave(g)
	m := g.zombie
	p := g.players[0]
	placePlayer(p, 400, 511)
	zp := m.playerState(1)
	zp.Gun, zp.Reserve = 10, 2
	// The magazine ran dry: the source rule falls back to the starter pistol.
	p.EquipWeapon(p.DefaultWeapon)
	updateN(t, g, 1)
	if p.Weapon.Def.Number != 10 || zp.Reserve != 1 {
		t.Fatalf("spare magazine: weapon=%d reserve=%d", p.Weapon.Def.Number, zp.Reserve)
	}
	zp.Reserve = 0
	p.EquipWeapon(p.DefaultWeapon)
	updateN(t, g, 1)
	if p.Weapon.Def.Number != p.DefaultWeapon || zp.Gun != 10 {
		t.Fatalf("out of magazines: weapon=%d gun=%d", p.Weapon.Def.Number, zp.Gun)
	}
}

func TestZombieMysteryBox(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 1, 0, 0})
	holdZombieWave(g)
	m := g.zombie
	p, other := g.players[0], g.players[1]
	box := &m.hud.Box
	m.boxMoveAt = 99
	zp := m.playerState(1)
	zp.Points = 100
	placePlayer(p, box.X, box.Y)
	if !m.boxInReach(p) {
		t.Fatal("player at the box is not in reach")
	}
	g.zombieUseBox(p)
	if box.State != zombieBoxIdle || zp.Msg != zombieMsgNoPoints {
		t.Fatalf("box without points: state=%d msg=%d", box.State, zp.Msg)
	}
	zp.Points = 2000
	g.zombieUseBox(p)
	if box.State != zombieBoxRolling || zp.Points != 2000-zombieBoxPrice || box.OwnerID != 1 {
		t.Fatalf("roll: state=%d points=%d owner=%d", box.State, zp.Points, box.OwnerID)
	}
	for i := 0; i < zombieBoxRollFrames; i++ {
		g.updateZombieBox()
	}
	if box.State != zombieBoxOffer || box.Weapon == 0 {
		t.Fatalf("offer: state=%d weapon=%d", box.State, box.Weapon)
	}
	offered := box.Weapon
	ozp := m.playerState(2)
	placePlayer(other, box.X, box.Y)
	g.zombieUseBox(other)
	if box.State != zombieBoxOffer || ozp.Gun == offered {
		t.Fatal("another player took the roller's weapon")
	}
	g.zombieUseBox(p)
	if zp.Gun != offered || p.Weapon.Def.Number != offered || box.State != zombieBoxIdle {
		t.Fatalf("take: gun=%d weapon=%d want %d", zp.Gun, p.Weapon.Def.Number, offered)
	}
	// Relocation: the roll comes up "bye bye", refunds and moves the box.
	m.boxMoveAt = box.Uses + 1
	spot := box.Spot
	g.zombieUseBox(p)
	pts := zp.Points
	for i := 0; i < zombieBoxRollFrames; i++ {
		g.updateZombieBox()
	}
	if box.State != zombieBoxLeaving || zp.Points != pts+zombieBoxPrice {
		t.Fatalf("leave: state=%d points=%d", box.State, zp.Points)
	}
	for i := 0; i < zombieBoxLeaveFrames; i++ {
		g.updateZombieBox()
	}
	if box.State != zombieBoxIdle || box.Spot == spot || box.Uses != 0 {
		t.Fatalf("moved box: state=%d spot=%d (was %d) uses=%d", box.State, box.Spot, spot, box.Uses)
	}
}

func TestZombieBoxDownKeyDoesNotDropThrough(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	holdZombieWave(g)
	m := g.zombie
	m.hud.Box = ZombieBoxState{X: 484, Y: 331}
	p := g.players[0]
	placePlayer(p, 484, 331)
	m.playerState(1).Points = 5000
	updateN(t, g, 2)
	p.RemoteInput = PlayerInputState{Down: true}
	updateN(t, g, 3)
	if m.hud.Box.State != zombieBoxRolling {
		t.Fatalf("DOWN at the box did not roll it: state=%d", m.hud.Box.State)
	}
	if math.Abs(p.Y-331) > 1 {
		t.Fatalf("player dropped through the box platform: y=%.1f", p.Y)
	}
}

func TestZombieBotsShopAndFight(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 2, 0, 0})
	m := g.zombie
	bot := g.players[1]
	if !bot.AI {
		t.Fatal("slot 2 should be a bot")
	}
	zp := m.playerState(2)
	zp.Points = 1300
	m.toSpawn, m.hud.Phase = 0, zombiePhaseWave
	updateN(t, g, 2)
	if m.hud.Phase != zombiePhaseBreak || !zp.Ready || zp.ShopOpen {
		t.Fatalf("bot at break: phase=%d ready=%v shop=%v", m.hud.Phase, zp.Ready, zp.ShopOpen)
	}
	if zp.Gun == 0 || zp.Points >= 1300 {
		t.Fatalf("bot did not buy a gun: gun=%d points=%d", zp.Gun, zp.Points)
	}
	// Bots target zombie bodies.
	holdZombieWave(g)
	z := spawnTestZombie(g, zombieWalker, bot.X+120, bot.Y, 1000)
	pool := g.aiTargetPool()
	found := false
	for _, p := range pool {
		if p == &z.body {
			found = true
		}
	}
	if !found {
		t.Fatal("zombie bodies are not AI targets")
	}
}

// --- online --------------------------------------------------------------------------------------

func TestZombieOnlineRequestValidation(t *testing.T) {
	bad := [][]netZombieReq{
		nil,
		make([]netZombieReq, netZombieReqMax+1),
		{{Seq: 0, Action: netZombieReqBuy}},
		{{Seq: 1, Action: 9}},
		{{Seq: 1, Action: netZombieReqBuy, Item: -1}},
		{{Seq: 1, Action: netZombieReqBuy, Item: netZombieItemLimit}},
		{{Seq: 1, Action: netZombieReqClose, Item: 3}},
	}
	for i, reqs := range bad {
		if validateNetZombieRequests(reqs) == nil {
			t.Fatalf("case %d accepted: %+v", i, reqs)
		}
		msg := netWireMessage{Type: netMessageZombie, Protocol: netplayProtocolVersion, ZombieReqs: reqs}
		if validateNetMessageForHost(msg) == nil {
			t.Fatalf("host accepted message case %d", i)
		}
	}
	good := netWireMessage{Type: netMessageZombie, Protocol: netplayProtocolVersion, ZombieReqs: []netZombieReq{{Seq: 4, Action: netZombieReqBuy, Item: 2}}}
	if err := validateNetMessageForHost(good); err != nil {
		t.Fatal(err)
	}
	if validateNetMessageForHost(netWireMessage{Type: netMessageZombie, Protocol: netplayProtocolVersion, AuthProof: "x", ZombieReqs: good.ZombieReqs}) == nil {
		t.Fatal("auth fields allowed in a zombie request")
	}
	// The wire view round-trips.
	codec := cborNetplayCodec{}
	payload, err := codec.Encode(good)
	if err != nil {
		t.Fatal(err)
	}
	var got netWireMessage
	if err := codec.Decode(payload, &got); err != nil || len(got.ZombieReqs) != 1 || got.ZombieReqs[0] != good.ZombieReqs[0] {
		t.Fatalf("round trip: %+v err=%v", got, err)
	}
}

func TestZombieOnlinePurchasesAreValidatedByTheHost(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 1, 0, 0})
	m := g.zombie
	guest := g.players[1]
	zp := m.playerState(2)
	pistol := shopItemFor(t, zombieItemWeapon, 5)
	mini := shopItemFor(t, zombieItemWeapon, 55)
	// Outside the break nothing is bought, but the request is acknowledged.
	g.applyZombieRequests(2, []netZombieReq{{Seq: 1, Action: netZombieReqBuy, Item: pistol.ID}})
	if zp.Gun != 0 || zp.ReqAck != 1 || zp.Points != zombieStartPoints {
		t.Fatalf("bought in a wave: gun=%d ack=%d", zp.Gun, zp.ReqAck)
	}
	openZombieBreak(g)
	g.applyZombieRequests(2, []netZombieReq{{Seq: 2, Action: netZombieReqBuy, Item: mini.ID}})
	if zp.Gun != 0 || zp.Points != zombieStartPoints || zp.Msg != zombieMsgNoPoints {
		t.Fatalf("bought without points: gun=%d points=%d", zp.Gun, zp.Points)
	}
	// A repeated (already acknowledged) request is never applied twice.
	reqs := []netZombieReq{{Seq: 3, Action: netZombieReqBuy, Item: pistol.ID}}
	g.applyZombieRequests(2, reqs)
	g.applyZombieRequests(2, reqs)
	if zp.Gun != 5 || zp.Points != zombieStartPoints-pistol.Price || zp.ReqAck != 3 {
		t.Fatalf("buy: gun=%d points=%d ack=%d", zp.Gun, zp.Points, zp.ReqAck)
	}
	if guest.Weapon.Def.Number != 5 {
		t.Fatal("the guest's player did not get the weapon")
	}
	// Unknown item and AI slots are ignored.
	g.applyZombieRequests(2, []netZombieReq{{Seq: 4, Action: netZombieReqBuy, Item: netZombieItemLimit - 1}})
	if zp.Points != zombieStartPoints-pistol.Price {
		t.Fatal("unknown item changed points")
	}
	g.applyZombieRequests(2, []netZombieReq{{Seq: 5, Action: netZombieReqClose}})
	if zp.ShopOpen || !zp.Ready {
		t.Fatalf("close: open=%v ready=%v", zp.ShopOpen, zp.Ready)
	}
	g.applyZombieRequests(2, []netZombieReq{{Seq: 6, Action: netZombieReqOpen}})
	if !zp.ShopOpen || zp.Ready {
		t.Fatalf("reopen: open=%v ready=%v", zp.ShopOpen, zp.Ready)
	}
	// Requests for another player's slot never touch that slot.
	before := *m.playerState(1)
	g.applyZombieRequests(3, []netZombieReq{{Seq: 1, Action: netZombieReqBuy, Item: pistol.ID}})
	if *m.playerState(1) != before {
		t.Fatal("a request for an empty slot changed player 1")
	}
}

func TestZombieGuestRequestQueueResendsUntilAcked(t *testing.T) {
	m := &zombieMatch{}
	g := &Game{zombie: m}
	g.queueZombieRequest(netZombieReqBuy, 3)
	g.queueZombieRequest(netZombieReqClose, 0)
	if len(m.clientReqs) != 2 || m.clientReqs[1].Seq != 2 {
		t.Fatalf("queue: %+v", m.clientReqs)
	}
	for i := 0; i < 20; i++ {
		g.queueZombieRequest(netZombieReqBuy, 1)
	}
	if len(m.clientReqs) != netZombieReqMax {
		t.Fatalf("queue not bounded: %d", len(m.clientReqs))
	}
}

func TestZombieSnapshotCapsWireAndClientMirror(t *testing.T) {
	found := map[string]bool{}
	for _, c := range netSnapshotListCaps {
		found[c.name] = true
	}
	if !found["zombies"] || !found["zombie pops"] {
		t.Fatal("zombie lists missing from netSnapshotListCaps")
	}
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	holdZombieWave(g)
	m := g.zombie
	for i := 0; i < zombieMaxActive; i++ {
		spawnTestZombie(g, zombieKind(1+i%5), 150+float64(i)*14, 511, 100)
	}
	for i := 0; i < 30; i++ {
		z := spawnTestZombie(g, zombieWalker, 300, 511, 1)
		z.Dead = true
	}
	state := g.makeNetSnapshot()
	if len(state.Zombies) != zombieMaxActive+30 {
		t.Fatalf("snapshot zombies=%d", len(state.Zombies))
	}
	trimNetSnapshotLists(&state)
	if len(state.Zombies) != zombieSnapshotCap {
		t.Fatalf("trimmed zombies=%d", len(state.Zombies))
	}
	alive := 0
	for _, z := range state.Zombies {
		if !z.Dead {
			alive++
		}
	}
	if alive != zombieMaxActive {
		t.Fatalf("trimming dropped living zombies: %d of %d kept", alive, zombieMaxActive)
	}
	// Wire: encode -> stream -> decode -> validate -> apply on a guest.
	var enc netHostEncoder
	frame := enc.encode(g.makeNetSnapshot())
	payload, rawLen, err := newNetPeerStream().message(frame, 2)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := newNetClientStream().decode(payload, rawLen)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateNetSnapshot(decoded.state); err != nil {
		t.Fatalf("zombie snapshot rejected: %v", err)
	}
	if !decoded.state.ZombieHUD.Active || len(decoded.state.Zombies) != zombieSnapshotCap {
		t.Fatalf("decoded hud=%v zombies=%d", decoded.state.ZombieHUD.Active, len(decoded.state.Zombies))
	}
	guest := New()
	guest.applyZombieSnapshot(&decoded.state)
	if guest.zombie == nil || guest.zombie.host || len(guest.zombie.zombies) != zombieSnapshotCap {
		t.Fatal("guest mirror not built")
	}
	last := m.zombies[zombieMaxActive-1]
	got := guest.zombie.zombies[len(guest.zombie.zombies)-1]
	if got.ID != last.ID || math.Abs(got.X-last.X) > 0.01 || got.HP != last.HP || got.Kind != last.Kind {
		t.Fatalf("mirror %+v want id %d", got.wire(), last.ID)
	}
	if got.body.Active || got.body.ID != 0 {
		t.Fatal("a physics body crossed the wire")
	}
	// Size bound: a zombie is a few dozen bytes raw, not a Player.
	base := netEncodeValue(nil, &netSnapshot{ZombieHUD: m.hud})
	full := netEncodeValue(nil, &netSnapshot{ZombieHUD: m.hud, Zombies: g.zombieSnapshot()[30:]})
	if per := float64(len(full)-len(base)) / zombieMaxActive; per > 64 {
		t.Fatalf("%.1f raw bytes per zombie on the wire", per)
	}
	// A hostile oversized list is rejected / trimmed.
	big := netSnapshot{Zombies: make([]Zombie, 500)}
	if validateNetSnapshot(big) == nil {
		t.Fatal("500 zombies accepted")
	}
	trimNetSnapshotLists(&big)
	if len(big.Zombies) != zombieSnapshotCap {
		t.Fatal("client trim of zombies")
	}
	// Inactive HUD clears a guest's mirror.
	guest.applyZombieSnapshot(&netSnapshot{})
	if guest.zombie != nil {
		t.Fatal("guest kept a zombie match after it ended")
	}
}

func TestZombieRenderInterpolationCoversZombies(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	holdZombieWave(g)
	z := spawnTestZombie(g, zombieWalker, 400, 511, 100)
	rising := spawnTestZombie(g, zombieWalker, 600, 511, 100)
	rising.State = zombieStateRising
	g.displaySmoothMotion = true
	now := time.Now()
	g.captureRenderPrev(now)
	z.X, z.VX = 410, 10
	rising.X = 700 // rising zombies snap
	g.renderInterp.lastUpdate = now
	restore := g.beginRenderInterpolation(now.Add(time.Second / TicksPerSecond / 2))
	var drawn, drawnRising *Zombie
	for _, d := range g.zombie.zombies {
		switch d.ID {
		case z.ID:
			drawn = d
		case rising.ID:
			drawnRising = d
		}
	}
	if drawn == nil || drawn == z || math.Abs(drawn.X-405) > 0.6 {
		t.Fatalf("zombie not interpolated: %+v", drawn)
	}
	if drawnRising.X != 700 {
		t.Fatalf("rising zombie interpolated: %.1f", drawnRising.X)
	}
	restore()
	if g.zombie.zombies[0] != z || z.X != 410 {
		t.Fatal("tick state not restored after Draw")
	}
}

// --- full flow --------------------------------------------------------------------------------

// TestZombieWaveFlowSim plays the first waves through Game.Update with a
// scripted shooter, then the break, the shop and READY.
func TestZombieWaveFlowSim(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	m := g.zombie
	p := g.players[0]
	zp := m.playerState(1)
	if m.hud.Phase != zombiePhaseIntro || m.hud.Wave != 1 {
		t.Fatalf("start: phase=%d wave=%d", m.hud.Phase, m.hud.Wave)
	}
	spawned := 0
	for frame := 0; frame < 30*240 && m.hud.Wave < 3; frame++ {
		zp.Invuln = 10
		zp.HP = zombiePlayerHP
		placePlayer(p, 483, 331)
		if frame%10 == 0 {
			for _, z := range m.zombies {
				if z.alive() {
					dir := 1
					if z.X < 483 {
						dir = -1
					}
					g.bullets = append(g.bullets, testBullet(z.X-float64(dir)*30, z.Y-20, dir, 60))
				}
			}
		}
		n := m.livingZombies()
		phase := m.hud.Phase
		updateN(t, g, 1)
		if m.livingZombies() > n {
			spawned += m.livingZombies() - n
		}
		if m.hud.Phase == zombiePhaseBreak && phase != zombiePhaseBreak {
			if !zp.ShopOpen {
				t.Fatal("shop not open in the break")
			}
			g.zombieSetShopOpen(p, false)
		}
	}
	if m.hud.Wave < 3 {
		t.Fatalf("stuck at wave %d phase %d remaining %d", m.hud.Wave, m.hud.Phase, m.hud.Remaining)
	}
	if zp.Kills < zombieWavePlanFor(1, 1).Count+zombieWavePlanFor(2, 1).Count {
		t.Fatalf("kills=%d after two waves", zp.Kills)
	}
	if zp.Earned < zp.Kills*60 {
		t.Fatalf("points earned %d for %d kills", zp.Earned, zp.Kills)
	}
}

func TestZombieSurvivalVictoryAndRecord(t *testing.T) {
	g := newZombieTestGame(t, 0, [4]int{1, 0, 0, 0})
	m := g.zombie
	m.hud.Wave = zombieSurvivalWaves
	m.hud.Phase = zombiePhaseWave
	m.toSpawn = 0
	updateN(t, g, 1)
	if m.hud.Phase != zombiePhaseWon {
		t.Fatalf("last wave cleared: phase=%d", m.hud.Phase)
	}
	updateN(t, g, zombieEndFrames+2)
	if !g.fadeActive || g.fadeTarget != screenZombieResults {
		t.Fatalf("results: fade=%v target=%d", g.fadeActive, g.fadeTarget)
	}
	rec := g.zombieRecordFor(0, zombieModeSurvival)
	if rec.Games != 1 || rec.Wins != 1 || rec.BestWave != zombieSurvivalWaves {
		t.Fatalf("record %+v", rec)
	}
	// The record survives a save/load round trip.
	data, err := json.Marshal(g.currentSaveData())
	if err != nil {
		t.Fatal(err)
	}
	var save SaveData
	if err := json.Unmarshal(data, &save); err != nil {
		t.Fatal(err)
	}
	save = normalizeSave(save)
	if save.ZombieRecords[zombieRecordKey(0, zombieModeSurvival)] != rec {
		t.Fatalf("record lost: %+v", save.ZombieRecords)
	}
}

func TestZombieMatchIgnoresSoloWinRule(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 0, 0, 0})
	updateN(t, g, 120)
	if g.GameWin || g.teamGameWin || g.soloWinFrame != 0 {
		t.Fatal("a solo zombie match triggered the last-survivor win")
	}
	if len(g.crates) != 0 || len(g.powerups) != 0 {
		t.Fatal("crates / powerups spawn in zombie mode")
	}
}

// --- menus ---------------------------------------------------------------------------------------

func zombieKitIDsUnique(t *testing.T, st *uiState, name string) {
	t.Helper()
	seen := map[string]bool{}
	for _, it := range st.items {
		if seen[it.id] {
			t.Fatalf("%s: duplicate kit id %q", name, it.id)
		}
		seen[it.id] = true
	}
}

func TestZombieSetupMenu(t *testing.T) {
	g := New()
	g.screen = screenZombieSetup
	g.runZombieSetupUpdate(uiInput{})
	zombieKitIDsUnique(t, &g.zombieUI, "setup")
	for _, id := range []string{"zs.mode", "zs.arena.0", "zs.arena.1", "zs.slot.1", "zs.slot.3", "zs.start", "zs.back", "zs.online"} {
		if _, ok := g.zombieUI.item(id); !ok {
			t.Fatalf("setup item %q missing", id)
		}
	}
	click := func(id string, fx float64) {
		it, _ := g.zombieUI.item(id)
		x, y := it.rect.X+it.rect.W*fx, it.rect.Y+it.rect.H/2
		g.runZombieSetupUpdate(uiInput{X: x, Y: y, Moved: true})
		g.runZombieSetupUpdate(uiInput{X: x, Y: y, Pressed: true, Down: true})
		g.runZombieSetupUpdate(uiInput{X: x, Y: y, Released: true})
		g.runZombieSetupUpdate(uiInput{X: x, Y: y})
	}
	click("zs.mode", 0.8)
	if g.zombieSetup.Mode != zombieModeEndless {
		t.Fatalf("mode=%d", g.zombieSetup.Mode)
	}
	click("zs.arena.1", 0.5)
	if g.zombieSetup.Arena != 1 {
		t.Fatalf("arena=%d", g.zombieSetup.Arena)
	}
	click("zs.slot.2", 0.9)
	if g.zombieSetup.Slots[2] != 2 {
		t.Fatalf("slot 3=%d want BOT", g.zombieSetup.Slots[2])
	}
	// Keyboard: arrows move, LEFT/RIGHT change a segmented control.
	g.runZombieSetupUpdate(uiInput{Tab: true})
	g.runZombieSetupUpdate(uiInput{})
	if g.zombieUI.focus == "" {
		t.Fatal("keyboard focus missing")
	}
	click("zs.start", 0.5)
	if g.zombie == nil || len(g.players) != 2 || !g.players[1].AI || g.zombie.hud.Mode != zombieModeEndless || g.arena.Number != zombieArenas[1].Zones[0].MapNumber {
		t.Fatalf("start: players=%d mode=%d map=%d", len(g.players), g.zombie.hud.Mode, g.arena.Number)
	}
	if !g.fadeActive || g.fadeTarget != screenGameplay {
		t.Fatal("start did not fade into gameplay")
	}
	for _, p := range g.players {
		if p.Team != 1 || p.Lives != zombieLives {
			t.Fatalf("player %d team=%d lives=%d", p.ID, p.Team, p.Lives)
		}
	}
	// ESC goes back.
	g2 := New()
	g2.screen = screenZombieSetup
	g2.runZombieSetupUpdate(uiInput{})
	g2.runZombieSetupUpdate(uiInput{Esc: true})
	if !g2.fadeActive || g2.fadeTarget != screenMainMenu {
		t.Fatal("ESC did not leave the setup")
	}
}

func TestZombieResultsMenu(t *testing.T) {
	g := newZombieTestGame(t, 0, [4]int{1, 2, 0, 0})
	g.zombie.hud.Phase = zombiePhaseOver
	g.zombie.hud.Wave = 7
	g.screen = screenZombieResults
	g.runZombieResultsUpdate(uiInput{})
	zombieKitIDsUnique(t, &g.zombieUI, "results")
	if _, ok := g.zombieUI.item("zr.again"); !ok {
		t.Fatal("PLAY AGAIN missing")
	}
	if g.zombie.wavesSurvived() != 6 {
		t.Fatalf("waves survived=%d", g.zombie.wavesSurvived())
	}
	g.runZombieResultsUpdate(uiInput{Esc: true})
	if g.zombie != nil || !g.fadeActive || g.fadeTarget != screenZombieSetup {
		t.Fatal("ESC on results did not return to the setup")
	}
}

func TestZombieShopPanelKitAndKeys(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 1, 0, 0})
	m := g.zombie
	for _, p := range g.players {
		m.hostPlayer(p.ID).remote = false // local panels for this test
	}
	openZombieBreak(g)
	p := g.players[0]
	zp := m.playerState(1)
	zp.Points = 5000
	r := zombieShopRects(2)[0]
	g.runZombieShopPanel(p, PlayerInputState{}, false, uiInput{}, r)
	zombieKitIDsUnique(t, &m.shop[0].st, "shop")
	if _, ok := m.shop[0].st.item("zs.p1.ready"); !ok {
		t.Fatal("READY missing")
	}
	// RIGHT switches the tab, DOWN picks, SHOOT buys, GRENADE is READY.
	press := func(in PlayerInputState) {
		g.runZombieShopPanel(p, in, false, uiInput{}, r)
		g.runZombieShopPanel(p, PlayerInputState{}, false, uiInput{}, r)
	}
	press(PlayerInputState{Right: true})
	if m.shop[0].tab != zombieTabThrowables {
		t.Fatalf("tab=%d", m.shop[0].tab)
	}
	press(PlayerInputState{Left: true})
	press(PlayerInputState{Down: true})
	press(PlayerInputState{Shoot: true})
	want := zombieShopTab(zombieTabGuns)[1]
	if zp.Gun != want.Value || zp.Points != 5000-want.Price {
		t.Fatalf("keyboard buy: gun=%d points=%d want %s", zp.Gun, zp.Points, want.Name)
	}
	press(PlayerInputState{Grenade: true})
	if zp.ShopOpen || !zp.Ready {
		t.Fatalf("GRENADE did not ready up: open=%v ready=%v", zp.ShopOpen, zp.Ready)
	}
	press(PlayerInputState{Ability: true})
	if !zp.ShopOpen || zp.Ready {
		t.Fatal("ABILITY did not reopen the shop")
	}
	// Keys held when the shop opens do not buy.
	zp.ShopOpen = false
	g.runZombieShopPanel(p, PlayerInputState{Shoot: true}, false, uiInput{}, r)
	zp.ShopOpen = true
	pts := zp.Points
	g.runZombieShopPanel(p, PlayerInputState{Shoot: true}, false, uiInput{}, r)
	if zp.Points != pts {
		t.Fatal("a held SHOOT bought on open")
	}
	// Rects: 1, 2 and 4 panels fit the screen and do not overlap.
	for _, n := range []int{1, 2, 3, 4} {
		rs := zombieShopRects(n)
		for i, a := range rs {
			if a.X < 0 || a.Y < 0 || a.X+a.W > ScreenWidth || a.Y+a.H > ScreenHeight {
				t.Fatalf("%d panels: rect %d off screen", n, i)
			}
			for j := i + 1; j < len(rs); j++ {
				if rectsOverlap(a, rs[j]) {
					t.Fatalf("%d panels overlap", n)
				}
			}
		}
	}
}

func TestZombieOnlineLobbyShowsZombieSettings(t *testing.T) {
	g := New()
	g.onlineZombie = true
	g.customOnlineMode = true
	ctx := g.customUI.begin(uiPassUpdate, nil, uiInput{})
	g.buildZombieOnlineSettings(ctx)
	ctx.end()
	zombieKitIDsUnique(t, &g.customUI, "online settings")
	if _, ok := g.customUI.item("zo.mode"); !ok {
		t.Fatal("online zombie mode control missing")
	}
	if s := g.onlineRuleSummary(); !strings.HasPrefix(s, "ZOMBIE CO-OP") {
		t.Fatalf("browser summary %q", s)
	}
}

// --- performance --------------------------------------------------------------------------

func TestZombieHordePerformance(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 1, 0, 0})
	holdZombieWave(g)
	m := g.zombie
	for i := 0; i < zombieMaxActive; i++ {
		sp := m.zone().Spawns[i%len(m.zone().Spawns)]
		spawnTestZombie(g, zombieKind(1+i%5), sp.X, sp.Y, 1<<20)
	}
	for _, p := range g.players {
		placePlayer(p, 300+float64(p.ID)*200, 511)
	}
	start := time.Now()
	const ticks = 300
	for i := 0; i < ticks; i++ {
		for _, zp := range m.hud.Players {
			if zp.ID != 0 {
				m.playerState(zp.ID).Invuln = 10
			}
		}
		if i%3 == 0 {
			g.bullets = append(g.bullets, testBullet(100, 491, 1, 10))
		}
		updateN(t, g, 1)
	}
	per := time.Since(start) / ticks
	if per > 8*time.Millisecond {
		t.Fatalf("%v per tick with %d zombies (budget 33 ms at 30 TPS, test limit 8 ms)", per, zombieMaxActive)
	}
	t.Logf("%v per tick with %d zombies", per, m.livingZombies())
}
