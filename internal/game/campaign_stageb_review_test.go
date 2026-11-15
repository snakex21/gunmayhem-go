package game

import (
	"math"
	"testing"
)

// Regression tests for the stage B review findings.

// TestMechClimbsToEveryUpperFloor: from every platform the mech must reach
// every reachable platform at least 80 px higher. Routes up often start away
// from the target; the mech used to flip-flop turning there.
func TestMechClimbsToEveryUpperFloor(t *testing.T) {
	for _, mapN := range []int{1, 7} {
		g0, _ := stageBGame(t, mapN)
		nav := buildZombieNav(g0.arena, zombieNavHeavy)
		pairs := 0
		for a, pa := range nav.plats {
			for bi, pb := range nav.plats {
				if pb.Y > pa.Y-80 || !nav.reachable(a, bi) {
					continue
				}
				pairs++
				g, human := stageBGame(t, mapN)
				m := spawnMechAt(g, pa.X+pa.W/2, pa.Y, 400, 1)
				m.attackCD = 1 << 20
				tx := pb.X + pb.W/2
				reached, turns, prev := false, 0, m.State
				for i := 0; i < 1800 && !reached; i++ {
					standOn(g, human, tx, pb.Y)
					g.updateCampaignEnemies(g.arena)
					if !m.alive() {
						t.Fatalf("map %d %d->%d: fell off", mapN, a, bi)
					}
					if m.State == mechStateTurn && prev != mechStateTurn {
						turns++
					}
					prev = m.State
					if m.Body.JumpNum == 2 && nav.platformUnder(m.Body.X, m.Body.Y) == bi {
						reached = true
					}
				}
				if !reached {
					t.Fatalf("map %d: mech never climbed %d (y %.0f) -> %d (y %.0f); stuck at (%.0f, %.0f) after %d turns",
						mapN, a, pa.Y, bi, pb.Y, m.Body.X, m.Body.Y, turns)
				}
				if turns > 8 {
					t.Fatalf("map %d %d->%d: %d turns (flip-flop)", mapN, a, bi, turns)
				}
			}
		}
		if pairs == 0 {
			t.Fatalf("map %d: no upper floors tested", mapN)
		}
		t.Logf("map %d: %d upward routes", mapN, pairs)
	}
}

func TestCampaignSnapshotStatePerKindAndListCap(t *testing.T) {
	g, _ := stageBGame(t, 1)
	sb := g.spawnCampaignEnemy(extraCampaignUnit{Kind: campaignEnemyShieldBearer}, 1, g.arena)
	sb.State = shieldBearerTurnFrames - 1 // mid-turn: a legitimate host state
	if err := validateNetSnapshot(netSnapshot{CampaignEnemies: g.campaignEnemySnapshot()}); err != nil {
		t.Fatalf("turning shield-bearer rejected: %v", err)
	}
	sb.State = shieldBearerTurnFrames + 1
	if validateNetSnapshot(netSnapshot{CampaignEnemies: g.campaignEnemySnapshot()}) == nil {
		t.Fatal("impossible shield-bearer state accepted")
	}
	sb.State = 0
	big := make([]CampaignEnemy, campaignEnemySnapshotMax+1)
	for i := range big {
		big[i] = *sb
	}
	if validateNetSnapshot(netSnapshot{CampaignEnemies: big}) == nil {
		t.Fatal("oversized campaign enemy list accepted")
	}
}

func TestMechJammedGunLeavesFireState(t *testing.T) {
	g, human := stageBGame(t, 1)
	m := spawnMechAt(g, 300, 256, 100, 1)
	m.State, m.StateTimer, m.burst = mechStateFire, 0, mechBurstRounds
	for i := 0; i < 200 && m.State == mechStateFire; i++ {
		standOn(g, human, 700, 256)
		m.Body.Weapon.WaitTime = -1000 // the gun never comes back
		g.updateCampaignEnemies(g.arena)
	}
	if m.State == mechStateFire {
		t.Fatal("mech stuck in its fire state with a jammed gun")
	}
}

func TestHeliHookDuringSalvoCallsOffUnfiredRocketsAndTheirMarkers(t *testing.T) {
	g, human := stageBGame(t, 5)
	h := spawnHeliAt(g, 300, 270, 90)
	h.Phase = 1
	standOn(g, human, 520, 486)
	g.heliPlanSalvo(h, g.arena)
	n := len(g.extraHazardMarkers)
	// Let the first rocket go, then hook it.
	for i := 0; i < heliSalvoWind+2; i++ {
		standOn(g, human, 520, 486)
		g.updateCampaignEnemies(g.arena)
	}
	if h.burst != 1 {
		t.Fatalf("rockets fired before the hook: %d", h.burst)
	}
	h.Body.HarpoonTether, h.Body.HarpoonTetherID = harpoonTetherFrames, human.ID
	before := len(g.bullets)
	for i := 0; i < 60; i++ {
		standOn(g, human, 520, 486)
		g.updateCampaignEnemies(g.arena)
	}
	if len(g.bullets) != before {
		t.Fatal("a called-off rocket still launched")
	}
	if len(g.extraHazardMarkers) != 1 || n < 2 {
		t.Fatalf("markers left %d of %d: only the fired rocket's should stay", len(g.extraHazardMarkers), n)
	}
}

func TestMechBackHitUsesTheAttackersSideAndIgnoresTheReel(t *testing.T) {
	g, human := stageBGame(t, 1)
	m := spawnMechAt(g, 450, 256, 100, 1) // facing right
	// A blast behind the mech thrown by a player standing in front of it.
	standOn(g, human, 700, 256)
	g.beginCampaignEnemyHitPhase()
	m.Body.VX += 20 * mechKnockbackScale
	m.Body.LastHitBy = human.ID
	g.endCampaignEnemyHitPhase(g.arena)
	if math.Abs(m.Body.VX-20*mechKnockbackScale) > 1e-6 {
		t.Fatalf("front attacker got the weak-spot bonus: vx=%.3f", m.Body.VX)
	}
	// A harpoon reel from behind is not a strike on the reactor.
	m.Body.VX = 0
	m.Body.LastHitBy = 0
	standOn(g, human, 200, 256)
	g.beginCampaignEnemyHitPhase()
	m.Body.VX += 15 * mechKnockbackScale
	m.Body.LastHitBy = human.ID
	m.Body.HarpoonTether, m.Body.HarpoonTetherID = harpoonTetherFrames, human.ID
	g.endCampaignEnemyHitPhase(g.arena)
	if math.Abs(m.Body.VX-15*mechKnockbackScale) > 1e-6 {
		t.Fatalf("harpoon reel got the weak-spot bonus: vx=%.3f", m.Body.VX)
	}
	// A shot from a player standing behind still is.
	m.Body.VX = 0
	m.Body.LastHitBy = 0
	g.beginCampaignEnemyHitPhase()
	m.Body.VX += 20 * mechKnockbackScale
	m.Body.LastHitBy = human.ID
	g.endCampaignEnemyHitPhase(g.arena)
	if math.Abs(m.Body.VX-20*mechKnockbackScale*mechBackMult) > 1e-6 {
		t.Fatalf("shot from behind missed the weak spot: vx=%.3f", m.Body.VX)
	}
}

func TestMeltingMechFallingOffKeepsItsKiller(t *testing.T) {
	g, human := stageBGame(t, 1)
	standOn(g, human, 100, 256)
	m := spawnMechAt(g, 880, 256, 3, 1)
	g.damageBigEnemy(m, 5, human.ID)
	m.Body.LastHitBy = 0 // credit expired (landed)
	m.Body.VX = 30
	for i := 0; i < mechMeltdownFrames && m.alive(); i++ {
		standOn(g, human, 100, 256)
		g.updateCampaignEnemies(g.arena)
	}
	if m.alive() || human.Kills != 1 {
		t.Fatalf("ring-out during meltdown: alive=%v kills=%d", m.alive(), human.Kills)
	}
}

func TestHeliGivesUpAnUnreachableDropPoint(t *testing.T) {
	g, human := stageBGame(t, 5)
	h := spawnHeliAt(g, 450, 270, 90)
	h.Phase = 2
	h.State, h.AimX0, h.StateTimer, h.approach = heliStateDrop, 99999, 0, 0
	for i := 0; i < heliDropMaxApproach+5 && h.State == heliStateDrop; i++ {
		standOn(g, human, 300, 486)
		g.updateCampaignEnemies(g.arena)
	}
	if h.State == heliStateDrop {
		t.Fatal("gunship hangs in the drop state forever")
	}
}

func TestMechRoundsStopOnPlatforms(t *testing.T) {
	g, human := stageBGame(t, 1)
	standOn(g, human, 700, 256)
	m := spawnMechAt(g, 300, 256, 100, 1)
	m.State, m.StateTimer, m.burst = mechStateFire, 0, mechBurstRounds
	m.Body.Weapon.WaitTime = m.Body.Weapon.Def.ROF
	tagged := 0
	for i := 0; i < 8; i++ {
		before := len(g.bullets)
		g.updateCampaignEnemies(g.arena)
		for _, bl := range g.bullets[before:] {
			if bl.OwnerID == m.Body.ID && bl.bossShot == bossShotRound {
				tagged++
			}
		}
	}
	if tagged == 0 {
		t.Fatal("mech rounds are not tagged to stop on platforms")
	}
	// A tagged round flying into a platform's side stops there.
	g.bullets = []Bullet{{Kind: BulletNormal, X: 520, Y: 360, VX: 25, Firepower: 12, OwnerID: m.Body.ID, Team: campaignEnemyTeam, Facing: 1, Alpha: 1, bossShot: bossShotRound}}
	for i := 0; i < 4; i++ {
		g.updateCampaignBossBullets(g.hitTargets())
		updateBullets(g.arena, g.bullets, g.hitTargets(), &g.instagibTrails, &g.explosions, &g.blastFX, &g.portals)
	}
	if !g.bullets[0].Dead || g.bullets[0].X > 600 {
		t.Fatalf("round went through the platform: x=%.0f", g.bullets[0].X)
	}
}

func TestZombieRopeAcceptsAnyPlayerID(t *testing.T) {
	g := newZombieTestGame(t, 0, [4]int{1, 0, 0, 0})
	z := spawnTestZombie(g, zombieWalker, 500, hazardLandingY(g.arena, 500), 40)
	z.HarpoonTether, z.HarpoonTetherID = harpoonTetherFrames, 5000
	s := netSnapshot{Zombies: []Zombie{*z}, ZombieHUD: g.zombie.hud}
	s.ZombieHUD.Active = true
	if err := validateZombieSnapshot(s); err != nil {
		t.Fatalf("rope from a high player ID rejected: %v", err)
	}
}
