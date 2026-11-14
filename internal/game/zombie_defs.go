package game

import "math"

// zombie_defs.go: data tables of the ZOMBIE mode (docs/zombie_mode.md).
//
// Everything a designer tunes lives here: zombie kinds, wave scaling, the
// points table, arenas (zones, spawn points, mystery box spots and, for step
// 2, doors) and the player rules. The runtime is in zombie_mode.go (match
// flow), zombie_ai.go (movement and attacks), zombie_shop.go (shop and box),
// zombie_net.go (online) and zombie_visuals.go / zombie_menu.go (drawing).

type zombieKind int

// Append new kinds before zombieKindCount, never reorder: the number travels
// in snapshots (Zombie.Kind).
const (
	zombieKindNone zombieKind = iota
	zombieWalker
	zombieRunner
	zombieBrute
	zombieExploder
	zombieLeaper
	zombieBoss
	zombieKindCount
)

// zombieKindStats are the wave-1 values of one kind. HP and speed are scaled
// per wave by zombieWavePlanFor.
type zombieKindStats struct {
	Name string
	HP   int
	// Move scales MoveSpeed (Player.CampaignMoveScale), Jump the jump power
	// (Player.RuleJumpMultiplier).
	Move, Jump float64
	// Knockback is the share of every push the zombie takes
	// (Player.KnockbackScale, applied in the ability knockback windows).
	Knockback float64
	// Scale multiplies the 0.8 player scale: hitbox and drawing.
	Scale float64
	// Melee: damage to a player's HP and the push it gives.
	Damage      int
	Push, Lift  float64
	Reach       float64 // extra reach in front of the hitbox
	Windup      int     // frames between starting the swing and the hit
	Cooldown    int     // frames between swings
	KillPoints  int
	MinWave     int // first wave the kind joins the mix (boss: every 5th)
	WeightBase  int // spawn weight at MinWave
	WeightGrow  int // + per wave after MinWave
	WeightLimit int
}

var zombieStats = [zombieKindCount]zombieKindStats{
	zombieWalker: {Name: "WALKER", HP: 40, Move: 0.42, Jump: 0.96, Knockback: 0.16, Scale: 1,
		Damage: 12, Push: 7, Lift: 3, Reach: 10, Windup: 10, Cooldown: 34, KillPoints: 60,
		MinWave: 1, WeightBase: 100, WeightGrow: 0, WeightLimit: 100},
	zombieRunner: {Name: "RUNNER", HP: 22, Move: 0.95, Jump: 1.04, Knockback: 0.26, Scale: 0.9,
		Damage: 7, Push: 5, Lift: 2, Reach: 8, Windup: 6, Cooldown: 22, KillPoints: 70,
		MinWave: 2, WeightBase: 20, WeightGrow: 6, WeightLimit: 70},
	zombieBrute: {Name: "BRUTE", HP: 170, Move: 0.32, Jump: 0.9, Knockback: 0.06, Scale: 1.38,
		Damage: 26, Push: 19, Lift: 8, Reach: 14, Windup: 16, Cooldown: 52, KillPoints: 160,
		MinWave: 4, WeightBase: 8, WeightGrow: 3, WeightLimit: 30},
	zombieExploder: {Name: "EXPLODER", HP: 30, Move: 0.55, Jump: 0.92, Knockback: 0.2, Scale: 1.08,
		Damage: 0, Push: 0, Lift: 0, Reach: 6, Windup: zombieExploderFuse, Cooldown: 0, KillPoints: 80,
		MinWave: 3, WeightBase: 10, WeightGrow: 3, WeightLimit: 35},
	zombieLeaper: {Name: "LEAPER", HP: 34, Move: 0.6, Jump: 1.12, Knockback: 0.2, Scale: 0.92,
		Damage: 12, Push: 9, Lift: 6, Reach: 8, Windup: 8, Cooldown: 30, KillPoints: 90,
		MinWave: 6, WeightBase: 10, WeightGrow: 3, WeightLimit: 35},
	zombieBoss: {Name: "ABOMINATION", HP: 900, Move: 0.36, Jump: 0.92, Knockback: 0.025, Scale: 2.2,
		Damage: 30, Push: 22, Lift: 10, Reach: 24, Windup: 18, Cooldown: 46, KillPoints: 1000},
}

func (k zombieKind) stats() zombieKindStats {
	if k <= zombieKindNone || k >= zombieKindCount {
		return zombieStats[zombieWalker]
	}
	return zombieStats[k]
}

// Kind-specific numbers.
const (
	zombieExploderFuse      = 22    // frames of hissing before the blast
	zombieExploderTrigger   = 46.0  // starts the fuse this close to a player
	zombieExploderRadius    = 105.0 // blast radius
	zombieExploderDamage    = 30    // player HP at the centre (falls off to 1/3)
	zombieExploderPush      = 26.0  // knockback at the centre
	zombieExploderChain     = 6     // frames before a caught exploder goes off
	zombieLeapRange         = 230.0 // pounce when the target is this close horizontally
	zombieLeapCooldown      = 75
	zombieLeapVX            = 9.5
	zombieLeapVY            = 10.5
	zombieBossSlamEvery     = 240 // frames between ground slams (phase 2: x0.6)
	zombieBossSlamRadius    = 230.0
	zombieBossSlamDamage    = 22
	zombieBossSlamPush      = 20.0
	zombieBossSummonEvery   = 360
	zombieBossSummonCount   = 3
	zombieRiseFrames        = 42 // telegraph + climbing out of the ground
	zombieRiseTelegraph     = 24 // of those, frames before the body appears
	zombieDeathFrames       = 22
	zombieCreditFrames      = 150 // a ring-out still credits the last shooter
	zombieHeadshotShare     = 0.5 // extra damage of a headshot (of the bullet's firepower)
	zombieHeadFraction      = 0.4 // top part of the hitbox that counts as the head
	zombieBlastDamageScale  = 1.0 // HP per unit of (unscaled) blast impulse
	zombieStuckFrames       = 60  // inside a small box this long: jump / turn around
	zombieRespawnStuck      = 600 // no path progress this long: re-spawn it
	zombieRetargetFrames    = 30
	zombieRepathFrames      = 20
	zombieIDBase            = 200
	zombieTeam              = 2
	zombieMaxActive         = 48 // hard cap of living zombies (performance)
	zombieMaxPerWave        = 120
	zombieSnapshotCap       = 64 // netSnapshotListCaps entry (alive + dying)
	zombiePopCap            = 32
	zombiePopLife           = 34
	zombieSpawnMinPlayerGap = 200.0 // spawn points this close to a player are skipped
)

// Match modes.
const (
	zombieModeSurvival = iota // PRZETRWANIE: zombieSurvivalWaves waves, then victory
	zombieModeEndless         // until everybody is down; record saved
	zombieModeCount
)

var zombieModeLabels = [zombieModeCount]string{"SURVIVAL", "ENDLESS"}

const zombieSurvivalWaves = 20

// Match phases (ZombieHUD.Phase).
const (
	zombiePhaseIntro = iota // wave banner, a short countdown
	zombiePhaseWave         // spawning and fighting
	zombiePhaseBreak        // shop between waves
	zombiePhaseOver         // everybody down: GAME OVER
	zombiePhaseWon          // survived the last wave
)

const (
	zombieIntroFrames    = 90  // 3 s
	zombieBreakFrames    = 750 // 25 s shop break
	zombieReadyCountdown = 90  // everybody READY: 3 s to the next wave
	zombieEndFrames      = 150 // GAME OVER / SURVIVED banner before the results
)

// Player rules.
const (
	zombiePlayerHP        = 100
	zombieReviveFrames    = 90  // a teammate stands at the downed player 3 s
	zombieBleedFrames     = 900 // 30 s to be revived, then DEAD until the wave ends
	zombieReviveRadiusX   = 58.0
	zombieReviveRadiusY   = 64.0
	zombieReviveHP        = 50
	zombieInvulnFrames    = 60 // after a revive / respawn
	zombieSoloSelfRevives = 1  // solo starts with one self-revive
	zombieMaxSelfRevives  = 3
	zombieLives           = 99 // Player.Lives is kept here; downs replace lives
	zombieStartPoints     = 500
)

// Player states (ZombiePlayerState.State).
const (
	zombiePlayerAlive = iota
	zombiePlayerDown
	zombiePlayerDead // bled out: back when the wave ends
)

// Points.
const (
	zombiePointsHit           = 10
	zombiePointsHeadshotBonus = 50
	zombiePointsRingOutBonus  = 40
	zombiePointsRevive        = 150
	zombiePointsWaveClear     = 100
)

// Points popup kinds (ZombiePop.Kind).
const (
	zombiePopHit = iota
	zombiePopKill
	zombiePopHeadshot
	zombiePopRingOut
	zombiePopRevive
	zombiePopSpend
)

// --- arenas ------------------------------------------------------------------

// zombiePoint is a position on a platform top (Y = the platform's top).
type zombiePoint struct {
	X, Y float64
}

// zombieZoneDef is one location of an arena. Step 1 arenas have a single
// zone; step 2 links several zones (each its own map) with doors. Spawn
// points and box spots belong to a zone, and every runtime entity carries
// the zone it is in (Zombie.Zone, ZombiePlayerState.Zone, ZombieBoxState.Zone).
type zombieZoneDef struct {
	ID        int
	Name      string
	MapNumber int
	Spawns    []zombiePoint
	BoxSpots  []zombiePoint
}

// zombieDoorDef connects two zones (step 2). Players buy it once for Cost;
// zombies path through open doors like through any nav link. On a shared
// local screen every living player must stand at the door to pass; online
// each player has an own camera and passes alone.
type zombieDoorDef struct {
	ID       int
	Zone     int // zone the door is in
	At       zombiePoint
	ToZone   int
	To       zombiePoint
	Cost     int
	Name     string
	OpenFrom int // wave from which it can be bought (0: always)
}

type zombieArenaDef struct {
	ID        string // persistence key part, never change
	Name      string
	Blurb     string
	StartZone int
	Zones     []zombieZoneDef
	Doors     []zombieDoorDef
}

func (a *zombieArenaDef) zone(id int) *zombieZoneDef {
	if a == nil {
		return nil
	}
	for i := range a.Zones {
		if a.Zones[i].ID == id {
			return &a.Zones[i]
		}
	}
	if len(a.Zones) > 0 {
		return &a.Zones[0]
	}
	return nil
}

// zombieArenas: horde arenas built on existing maps with wide floors, many
// tiers and short drops, so there is room to kite and zombies always have a
// way up (checked by TestZombieAIReachesPlayersOnEveryArena).
var zombieArenas = []zombieArenaDef{
	{
		ID: "midnight_wood", Name: "MIDNIGHT WOOD", Blurb: "Nine ledges in a dark forest. Zombies climb out at the far ledges and the treetops.",
		Zones: []zombieZoneDef{{
			ID: 0, Name: "FOREST", MapNumber: 9,
			Spawns: []zombiePoint{
				{40, 420}, {890, 420}, // outer low ledges
				{90, 330}, {840, 330}, // outer middle ledges
				{190, 150}, {740, 150}, // treetop ends
				{140, 240}, {790, 240},
			},
			BoxSpots: []zombiePoint{{466, 510}, {469, 330}, {466, 150}},
		}},
	},
	{
		ID: "desert_destruction", Name: "DESERT DESTRUCTION", Blurb: "Long floors and ledges on both sides. Hold the middle bridge or run the decks.",
		Zones: []zombieZoneDef{{
			ID: 0, Name: "DESERT", MapNumber: 7,
			Spawns: []zombiePoint{
				{80, 421}, {885, 421},
				{80, 240}, {885, 240},
				{130, 151}, {835, 151},
				{130, 511}, {835, 511},
			},
			BoxSpots: []zombiePoint{{483, 511}, {484, 331}, {483, 151}},
		}},
	},
}

func zombieArenaByIndex(i int) *zombieArenaDef {
	if i < 0 || i >= len(zombieArenas) {
		return &zombieArenas[0]
	}
	return &zombieArenas[i]
}

// --- waves -------------------------------------------------------------------

// zombieWavePlan is everything the spawner needs for one wave.
type zombieWavePlan struct {
	Wave      int
	Count     int // regular zombies (the boss is extra)
	MaxActive int
	HPMul     float64
	SpeedMul  float64
	Interval  int // frames between spawns
	Weights   [zombieKindCount]int
	Boss      bool
	BossLevel int
	BossHP    int
}

func zombieCoopFactor(players int) float64 {
	if players < 1 {
		players = 1
	}
	return 1 + 0.45*float64(players-1)
}

// zombieWavePlanFor scales count, HP, speed and the type mix with the wave
// number and the number of players.
func zombieWavePlanFor(wave, players int) zombieWavePlan {
	if wave < 1 {
		wave = 1
	}
	if players < 1 {
		players = 1
	}
	if players > 4 {
		players = 4
	}
	w := float64(wave - 1)
	base := 6 + 3*w
	if wave > 10 {
		base += float64(wave-10) * 1.5
	}
	count := int(math.Round(base * zombieCoopFactor(players)))
	plan := zombieWavePlan{Wave: wave}
	plan.HPMul = (1 + 0.13*w) * (1 + 0.04*math.Max(0, float64(wave-10))) * (1 + 0.15*float64(players-1))
	plan.SpeedMul = math.Min(1+0.02*w, 1.35)
	plan.Interval = max(10, 40-2*wave)
	plan.MaxActive = min(14+4*players+wave, zombieMaxActive)
	for k := zombieWalker; k < zombieBoss; k++ {
		s := zombieStats[k]
		if wave < s.MinWave {
			continue
		}
		plan.Weights[k] = min(s.WeightBase+s.WeightGrow*(wave-s.MinWave), s.WeightLimit)
	}
	if wave%5 == 0 {
		plan.Boss = true
		plan.BossLevel = wave / 5
		count = count / 2
		bossHP := float64(zombieStats[zombieBoss].HP) * (1 + 0.6*float64(plan.BossLevel-1)) * (1 + 0.5*float64(players-1))
		plan.BossHP = int(math.Round(bossHP/10) * 10)
	}
	plan.Count = min(count, zombieMaxPerWave)
	return plan
}

// pick chooses a kind from the plan's weights with r in [0,1).
func (p zombieWavePlan) pick(r float64) zombieKind {
	total := 0
	for _, w := range p.Weights {
		total += w
	}
	if total <= 0 {
		return zombieWalker
	}
	x := int(r * float64(total))
	for k, w := range p.Weights {
		if x < w {
			return zombieKind(k)
		}
		x -= w
	}
	return zombieWalker
}

// zombieScaledHP is a regular zombie's HP for a plan.
func zombieScaledHP(kind zombieKind, plan zombieWavePlan) int {
	if kind == zombieBoss && plan.BossHP > 0 {
		return plan.BossHP
	}
	hp := float64(kind.stats().HP) * plan.HPMul
	return max(1, int(math.Round(hp)))
}
