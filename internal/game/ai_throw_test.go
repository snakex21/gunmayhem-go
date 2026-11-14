package game

import (
	"math"
	"testing"
)

// throwTestArena: a wide floor, a raised ledge and a small high platform, so
// throws bounce, land on different levels and can be lost off the sides.
func throwTestArena() Map {
	return Map{
		Gravity: DefaultGravity,
		Platforms: []Rect{
			{X: 60, Y: 400, W: 780, H: 40},
			{X: 500, Y: 290, W: 180, H: 14},
			{X: 220, Y: 200, W: 120, H: 14},
		},
		LowestY: 600,
	}
}

// realThrowOutcome runs the real updateGrenadesWithAreas on a released
// throwable until it is gone and reports where it went off.
func realThrowOutcome(arena Map, g Grenade) (x, y float64, frames int, detonated bool) {
	grenades := []Grenade{g}
	var explosions []Explosion
	var dyn []DynamiteEffect
	var blast []BlastEffect
	var fire []FirePatch
	var areas []AreaEffect
	for frame := 1; frame <= throwSimMaxFrames; frame++ {
		before := len(explosions) + len(fire) + len(areas) + len(blast)
		updateGrenadesWithAreas(arena, grenades, nil, &explosions, &dyn, &blast, &fire, &areas)
		if grenades[0].Dead {
			after := len(explosions) + len(fire) + len(areas) + len(blast)
			return grenades[0].X, grenades[0].Y, frame, after > before
		}
	}
	return grenades[0].X, grenades[0].Y, throwSimMaxFrames, false
}

func TestThrowSimulatorMatchesRealGrenadeUpdate(t *testing.T) {
	arena := throwTestArena()
	checked := 0
	for kind := 0; kind < throwableTypeCount; kind++ {
		for _, startX := range []float64{120, 420, 760} {
			for _, facing := range []int{1, -1} {
				for power := aiThrowMinPower; power <= grenadeMaxPower; power += 1.5 {
					sim := simulateThrow(arena, startX, 400, facing, power, kind)
					x, y, frames, detonated := realThrowOutcome(arena, throwSpawnGrenade(startX, 400, facing, power, kind))
					if frames != sim.Frames || math.Abs(x-sim.X) > 1e-9 {
						t.Fatalf("kind %d x=%.0f f=%d p=%.1f: sim (%.2f,%.2f)@%d, real (%.2f,%.2f)@%d",
							kind, startX, facing, power, sim.X, sim.Y, sim.Frames, x, y, frames)
					}
					if detonated && math.Abs(y-sim.Y) > 1e-9 {
						t.Fatalf("kind %d p=%.1f: sim y=%.2f real y=%.2f", kind, power, sim.Y, y)
					}
					if kind != ThrowableMolotov && detonated != sim.OK {
						t.Fatalf("kind %d p=%.1f: sim ok=%v, real detonated=%v", kind, power, sim.OK, detonated)
					}
					checked++
				}
			}
		}
	}
	if checked < 100 {
		t.Fatalf("only %d throws compared", checked)
	}
}

func TestGrenadeThrowPreviewIsTheThrowSimulator(t *testing.T) {
	arena := throwTestArena()
	for kind := 0; kind < throwableTypeCount; kind++ {
		p := &Player{ID: 1, Active: true, X: 300, Y: 400, Facing: 1, GrenadePower: 7, ThrowableType: kind}
		points, landed := grenadeThrowPreview(arena, p, 200)
		res := simulateGrenade(arena, throwSpawnGrenade(p.X, p.Y, p.Facing, p.GrenadePower, kind), true)
		if len(points) == 0 || len(points) > len(res.Path) {
			t.Fatalf("kind %d: preview has %d points, sim %d", kind, len(points), len(res.Path))
		}
		for i := range points {
			if points[i] != res.Path[i] {
				t.Fatalf("kind %d: preview point %d %+v != sim %+v", kind, i, points[i], res.Path[i])
			}
		}
		if landed != (res.LandIndex >= 0) || (landed && len(points) != res.LandIndex+1) {
			t.Fatalf("kind %d: landed=%v points=%d, sim land index %d", kind, landed, len(points), res.LandIndex)
		}
	}
}

func aiThrowTestGame(kind int, bot *Player, others ...*Player) *Game {
	arena := Map{Gravity: DefaultGravity, Platforms: []Rect{{X: 0, Y: 400, W: 900, H: 60}}, LowestY: 600}
	players := append([]*Player{bot}, others...)
	bot.ThrowableType = kind
	return &Game{arena: arena, players: players}
}

func aiThrowTestBot(x float64) *Player {
	return &Player{ID: 1, Team: 1, AI: true, Active: true, X: x, Y: 400, Facing: 1, JumpNum: 2,
		Grenades: 3, PlayerScale: .8, Weapon: NewWeapon(1)}
}

func aiThrowTestEnemy(id int, x float64) *Player {
	return &Player{ID: id, Team: id, Active: true, X: x, Y: 400, Facing: -1, JumpNum: 2, PlayerScale: .8}
}

func TestAIThrowChosenPowerLandsOnTarget(t *testing.T) {
	for _, kind := range []int{ThrowableDynamite, ThrowableMk2, ThrowableF1, ThrowableMolotov, ThrowableShock, ThrowableTearGas} {
		bot := aiThrowTestBot(120)
		// A cluster of two enemies far enough away for a safe throw.
		g := aiThrowTestGame(kind, bot, aiThrowTestEnemy(2, 520), aiThrowTestEnemy(3, 560))
		plan, ok := aiBestThrow(bot, g, g.customRuntimeArena(), 1)
		if !ok {
			t.Fatalf("kind %d: no throw at a two-enemy cluster", kind)
		}
		res := simulateThrow(g.arena, bot.X, bot.Y, bot.Facing, plan.Power, kind)
		if d := math.Abs(res.X - 540); d > 90 {
			t.Fatalf("kind %d: power %.1f lands at x=%.1f, %.0f px from the cluster", kind, plan.Power, res.X, d)
		}
	}
}

func TestAIThrowNeverInOwnBlast(t *testing.T) {
	for _, kind := range []int{ThrowableDynamite, ThrowableF1, ThrowableShock, ThrowableTearGas} {
		// Enemies hugging the bot: every blast that reaches them reaches it.
		bot := aiThrowTestBot(400)
		g := aiThrowTestGame(kind, bot, aiThrowTestEnemy(2, 420), aiThrowTestEnemy(3, 440))
		if plan, ok := aiBestThrow(bot, g, g.arena, 1); ok {
			t.Fatalf("kind %d: bot throws into its own blast (power %.1f at x=%.1f)", kind, plan.Power, plan.Result.X)
		}
		// Further away: whatever is chosen stays outside radius+margin.
		bot = aiThrowTestBot(120)
		g = aiThrowTestGame(kind, bot, aiThrowTestEnemy(2, 330), aiThrowTestEnemy(3, 360))
		radius, lift, _ := aiThrowHarmRadius(kind)
		if plan, ok := aiBestThrow(bot, g, g.arena, 1); ok {
			if d := math.Hypot(bot.X-plan.Result.X, bot.Y-30-(plan.Result.Y-lift)); d <= radius+aiThrowSelfMargin {
				t.Fatalf("kind %d: chosen throw goes off %.0f px from the bot (radius %.0f)", kind, d, radius)
			}
		}
	}
}

func TestAIThrowSparesTeammates(t *testing.T) {
	bot := aiThrowTestBot(120)
	mate := aiThrowTestEnemy(4, 540)
	mate.Team = bot.Team
	g := aiThrowTestGame(ThrowableDynamite, bot, aiThrowTestEnemy(2, 520), aiThrowTestEnemy(3, 560), mate)
	if plan, ok := aiBestThrow(bot, g, g.arena, 1); ok {
		if math.Hypot(mate.X-plan.Result.X, mate.Y-30-plan.Result.Y) <= throwableProfileFor(ThrowableDynamite).Radius+aiThrowSelfMargin {
			t.Fatalf("throw at x=%.0f catches the teammate at %.0f", plan.Result.X, mate.X)
		}
	}
}

func TestAIFlashbangNeverBlindsThrower(t *testing.T) {
	// Airborne: the bot cannot be sure to turn away, so no flash within sight.
	bot := aiThrowTestBot(120)
	bot.JumpNum = 1
	bot.Y = 380
	g := aiThrowTestGame(ThrowableFlashbang, bot, aiThrowTestEnemy(2, 300), aiThrowTestEnemy(3, 330))
	if plan, ok := aiBestThrow(bot, g, g.arena, 1); ok {
		fx, fy := plan.Result.X, plan.Result.Y-flashPointLift
		probe := *bot
		probe.Facing = 1
		if frames, _ := flashbangBlindFor(g.arena, fx, fy, &probe); frames > 0 {
			t.Fatalf("airborne bot throws a flashbang that blinds itself for %d frames", frames)
		}
	}

	// Grounded: it may throw, then turns its back. It must end up blinded no
	// longer than a turned-away glance.
	for _, enemyX := range []float64{260, 360, 480, 640} {
		bot = aiThrowTestBot(120)
		g = aiThrowTestGame(ThrowableFlashbang, bot, aiThrowTestEnemy(2, enemyX), aiThrowTestEnemy(3, enemyX+30))
		plan, ok := aiBestThrow(bot, g, g.arena, 1)
		if !ok {
			continue
		}
		fx, fy := plan.Result.X, plan.Result.Y-flashPointLift
		aiTurnAwayFromFlash(g.arena, bot, fx, fy)
		probe := *bot
		if bot.AIFlashAvoid > 0 {
			probe.Facing = bot.AIFlashAvoidDir
		}
		if frames, _ := flashbangBlindFor(g.arena, fx, fy, &probe); frames > flashAwayBlindFrames {
			t.Fatalf("enemy at %.0f: bot blinds itself for %d frames", enemyX, frames)
		}
	}
}

// runBotThrowFrames drives only the throw decision and the real throw input.
func runBotThrowFrames(g *Game, bot *Player, frames int) {
	for i := 0; i < frames; i++ {
		bot.clearAIInput()
		aiDecideThrow(bot, g)
		updateGrenadeInput(bot, &g.grenades)
		updateGrenadesWithAreas(g.arena, g.grenades, nil, &g.explosions, &g.dynamiteFX, &g.blastFX, &g.firePatches, &g.areaEffects)
		g.grenades = compactGrenades(g.grenades)
	}
}

func TestAIThrowSpendsAThrowable(t *testing.T) {
	bot := aiThrowTestBot(120)
	bot.AISpecial = sourceAISpecialBoss // skill 1
	g := aiThrowTestGame(ThrowableDynamite, bot, aiThrowTestEnemy(2, 520), aiThrowTestEnemy(3, 560))
	thrown := 0
	for i := 0; i < 600 && thrown == 0; i++ {
		before := bot.Grenades
		runBotThrowFrames(g, bot, 1)
		if bot.Grenades < before {
			thrown++
			if bot.Grenades != before-1 {
				t.Fatalf("one throw spent %d throwables", before-bot.Grenades)
			}
		}
	}
	if thrown == 0 {
		t.Fatal("bot never threw at a two-enemy cluster")
	}
	if bot.AIThrowCooldown < 90 {
		t.Fatalf("no cooldown after the throw: %d", bot.AIThrowCooldown)
	}
}

func TestAIThrowBlockedInInstagibAndGunGame(t *testing.T) {
	for _, mode := range []int{SourceGameModeInstagib, SourceGameModeGunGame} {
		bot := aiThrowTestBot(120)
		bot.AISpecial = sourceAISpecialBoss
		bot.GameMode = mode
		g := aiThrowTestGame(ThrowableDynamite, bot, aiThrowTestEnemy(2, 520), aiThrowTestEnemy(3, 560))
		runBotThrowFrames(g, bot, 600)
		if bot.Grenades != 3 || bot.GrenadeHeld {
			t.Fatalf("mode %d: bot threw (left %d, held %v)", mode, bot.Grenades, bot.GrenadeHeld)
		}
	}
}

func TestAIThrowLimits(t *testing.T) {
	// Low skill keeps its last throwable in reserve.
	if aiThrowAllowed(&Player{AI: true, Active: true, JumpNum: 2, Grenades: 1}, 0) {
		t.Fatal("low-skill bot throws its last throwable")
	}
	if !aiThrowAllowed(&Player{AI: true, Active: true, JumpNum: 2, Grenades: 1}, 1) {
		t.Fatal("skilled bot may use its last throwable")
	}
	if aiThrowAllowed(&Player{AI: true, Active: true, JumpNum: 2, Grenades: 3, BlindTime: 5}, 1) {
		t.Fatal("blind bot throws")
	}
	if aiThrowAllowed(&Player{AI: true, Active: true, JumpNum: 2, Grenades: 3, AnchorTime: 5}, 1) {
		t.Fatal("anchored bot throws")
	}
	if aiThrowAllowed(&Player{AI: true, Active: true, JumpNum: 1, Grenades: 3}, 0.5) {
		t.Fatal("mid-skill bot throws from the air")
	}
	if !aiThrowAllowed(&Player{AI: true, Active: true, JumpNum: 1, Grenades: 3}, 0.9) {
		t.Fatal("high-skill bot may throw from the air")
	}
	if aiThrowCooldownFrames(0) != 240 || aiThrowCooldownFrames(1) != 90 {
		t.Fatal("cooldown range changed")
	}
}

// TestBotVersusBotThrowsAtASaneRate runs the full game loop headlessly with
// two bots and counts real throws.
func TestBotVersusBotThrowsAtASaneRate(t *testing.T) {
	if testing.Short() {
		t.Skip("long headless sim")
	}
	g := New()
	g.screen = screenGameplay
	g.fadeActive = false
	for _, p := range g.players {
		p.AI = true
		p.Lives, p.TotalLives = 9999, 9999
	}
	g.players[0].ThrowableType = ThrowableMk2
	g.players[1].ThrowableType = ThrowableMolotov
	const frames = 60 * 120
	throws := 0
	for frame := 0; frame < frames; frame++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		for i := range g.grenades {
			if g.grenades[i].Fuse == 1 && g.grenades[i].OwnerID > 0 {
				throws++
			}
		}
		if g.GameWin {
			break
		}
	}
	// Two bots, 3 throwables per life, 90+ frame cooldown: at most one throw
	// per 90 frames per bot, and they should throw at all.
	if throws == 0 {
		t.Fatal("bots never threw in two minutes of bot-vs-bot play")
	}
	if maxThrows := 2 * frames / 90; throws > maxThrows {
		t.Fatalf("bots threw %d times in %d frames (max %d)", throws, frames, maxThrows)
	}
	t.Logf("bot-vs-bot: %d throws in %d frames", throws, frames)
}
