package game

import (
	"math"
	"testing"
)

func burnTestPlayer(id int) *Player {
	return &Player{ID: id, Active: true, X: 300, Y: 100, Facing: 1, PlayerScale: .8, DamageMulti: 1, Weight: 1, JumpNum: 2}
}

type burnTrace struct {
	joltFrames []int
	joltDirs   []int
	windows    []int // panic run lengths
	gaps       []int // controllable frames between runs
	vx         []float64
}

// runBurn ticks only the burn status (no physics) and records what it did.
func runBurn(p *Player, frames int) burnTrace {
	var tr burnTrace
	inWindow, runLen, gapLen, sawWindow := false, 0, 0, false
	for f := 0; f < frames && p.BurnTime > 0; f++ {
		p.VX, p.VY = 0, 0
		hits := p.HitNumber
		p.updateBurning()
		tr.vx = append(tr.vx, p.VX)
		if p.HitNumber != hits {
			tr.joltFrames = append(tr.joltFrames, f)
			if p.VX > 0 {
				tr.joltDirs = append(tr.joltDirs, 1)
			} else {
				tr.joltDirs = append(tr.joltDirs, -1)
			}
		}
		if p.BurnPanicTime > 0 {
			if !inWindow {
				if sawWindow {
					tr.gaps = append(tr.gaps, gapLen)
				}
				inWindow, runLen, sawWindow = true, 0, true
			}
			runLen++
		} else {
			if inWindow {
				tr.windows = append(tr.windows, runLen)
				inWindow, gapLen = false, 0
			}
			gapLen++
		}
	}
	return tr
}

func TestBurnJoltsFollowCadenceAndBiasAwayFromFire(t *testing.T) {
	away, total := 0, 0
	for id := 1; id <= 8; id++ {
		p := burnTestPlayer(id)
		p.igniteWithPower(1, 1, 20000, 20) // fire on the left: away = +1
		tr := runBurn(p, 6000)
		if len(tr.joltFrames) < 300 {
			t.Fatalf("id %d: only %d jolts in 6000 frames", id, len(tr.joltFrames))
		}
		if tr.joltFrames[0]+1 < burnJoltIntervalMin || tr.joltFrames[0]+1 > burnJoltIntervalMax {
			t.Fatalf("id %d: first jolt on frame %d", id, tr.joltFrames[0]+1)
		}
		for i := 1; i < len(tr.joltFrames); i++ {
			gap := tr.joltFrames[i] - tr.joltFrames[i-1]
			if gap < burnJoltIntervalMin || gap > burnJoltIntervalMax {
				t.Fatalf("id %d: jolt gap %d outside %d..%d", id, gap, burnJoltIntervalMin, burnJoltIntervalMax)
			}
		}
		for _, d := range tr.joltDirs {
			if d > 0 {
				away++
			}
			total++
		}
		// Moderate strength: one jolt is a nudge, not a knockback hit.
		for _, vx := range tr.vx {
			if math.Abs(vx) > burnJoltMax+1e-9 {
				t.Fatalf("id %d: jolt %.2f exceeds max %.2f", id, vx, burnJoltMax)
			}
		}
	}
	ratio := float64(away) / float64(total)
	if ratio < burnAwayBias-0.05 || ratio > burnAwayBias+0.05 {
		t.Fatalf("away-from-fire ratio %.3f want ~%.2f", ratio, burnAwayBias)
	}
}

func TestBurnPanicRunsOverrideInputThenEnd(t *testing.T) {
	p := burnTestPlayer(3)
	p.igniteWithPower(1, -1, 20000, 20)
	tr := runBurn(p, 3000)
	if len(tr.windows) < 40 {
		t.Fatalf("only %d panic runs in 3000 frames", len(tr.windows))
	}
	for _, w := range tr.windows {
		if w < burnPanicWindowMin || w > burnPanicWindowMax {
			t.Fatalf("panic run length %d outside %d..%d", w, burnPanicWindowMin, burnPanicWindowMax)
		}
	}
	for _, gap := range tr.gaps {
		if gap < burnPanicGapMin || gap > burnPanicGapMax {
			t.Fatalf("control gap %d outside %d..%d", gap, burnPanicGapMin, burnPanicGapMax)
		}
	}

	// Input override only inside a window.
	p.BurnPanicTime, p.BurnPanicDir = 5, -1
	if l, r := p.burnPanicInput(false, true); !l || r {
		t.Fatalf("panic run did not take over input: left=%v right=%v", l, r)
	}
	p.BurnPanicTime = 0
	if l, r := p.burnPanicInput(false, true); l || !r {
		t.Fatalf("control not returned between panic runs: left=%v right=%v", l, r)
	}
}

func TestBurnPanicRunMovesPlayerAgainstHeldInput(t *testing.T) {
	arena := Map{Platforms: []Rect{{X: -5000, Y: 100, W: 10000, H: 30}}, Gravity: 0.6, LowestY: 400}
	p := burnTestPlayer(2)
	p.AI = true
	p.igniteWithPower(1, -1, 20000, 20)
	overridden := 0
	for f := 0; f < 400; f++ {
		p.clearAIInput()
		p.AIRight = true // player holds right the whole time
		panicking := p.BurnPanicTime > 0 && p.BurnPanicDir < 0
		p.Update(arena, nil, nil)
		if panicking {
			overridden++
			if p.Facing != -1 {
				t.Fatalf("frame %d: panic run left but facing %d", f, p.Facing)
			}
		}
	}
	if overridden == 0 {
		t.Fatal("no leftward panic run in 400 frames despite left bias")
	}
	if overridden > 400/2 {
		t.Fatalf("panic overrode %d/400 frames: control lost too often", overridden)
	}
}

func TestJumpPatsDownBurnWithLimits(t *testing.T) {
	p := burnTestPlayer(2)
	p.igniteWithPower(1, 1, 200, 20)
	if !p.burnPatDown() || p.BurnTime != 160 {
		t.Fatalf("first pat-down: burn=%d want160", p.BurnTime)
	}
	if p.burnPatDown() {
		t.Fatal("pat-down ignored its cooldown (double jump counted twice)")
	}
	p.BurnPatCooldown = 0
	p.burnPatDown() // 128
	p.BurnPatCooldown = 0
	p.burnPatDown() // 102
	p.BurnPatCooldown = 0
	if p.BurnTime != 102 || p.burnPatDown() {
		t.Fatalf("pat-down limit: burn=%d count=%d", p.BurnTime, p.BurnPatDowns)
	}
	// Floor: never below burnPatDownFloor.
	q := burnTestPlayer(3)
	q.igniteWithPower(1, 1, 14, 20)
	q.burnPatDown()
	if q.BurnTime != burnPatDownFloor {
		t.Fatalf("pat-down floor: burn=%d want%d", q.BurnTime, burnPatDownFloor)
	}
	if q.burnPatDown() {
		t.Fatal("pat-down below floor")
	}

	// Integration: a real jump in the player update pats the fire down.
	arena := Map{Platforms: []Rect{{X: -5000, Y: 100, W: 10000, H: 30}}, Gravity: 0.6, LowestY: 400}
	r := burnTestPlayer(4)
	r.AI = true
	r.igniteWithPower(1, 1, 200, 20)
	r.AIUp = true
	r.Update(arena, nil, nil)
	if r.BurnTime > 200-40 || r.BurnPatDowns != 1 {
		t.Fatalf("jump did not pat down: burn=%d pats=%d", r.BurnTime, r.BurnPatDowns)
	}
}

func TestBlastShieldHalvesBurnAndJolts(t *testing.T) {
	plain := burnTestPlayer(5)
	shield := burnTestPlayer(5)
	shield.PerkNumber = 6
	plain.igniteWithPower(1, 1, 120, 20)
	shield.igniteWithPower(1, 1, 120, 20)
	if plain.BurnTime != 120 || shield.BurnTime != 60 || shield.BurnMax != 60 {
		t.Fatalf("blast shield burn: plain=%d shield=%d max=%d", plain.BurnTime, shield.BurnTime, shield.BurnMax)
	}
	if got, want := burnJoltStrength(shield), burnJoltStrength(plain)*burnBlastShieldScale; math.Abs(got-want) > 1e-9 {
		t.Fatalf("blast shield jolt %.3f want %.3f", got, want)
	}
	// Same seed => same jolt frames, half the impulse.
	pt, st := runBurn(plain, 60), runBurn(shield, 60)
	if len(pt.joltFrames) == 0 || len(st.joltFrames) == 0 {
		t.Fatal("no jolts")
	}
	if math.Abs(math.Abs(st.vx[st.joltFrames[0]])-math.Abs(pt.vx[pt.joltFrames[0]])*burnBlastShieldScale) > 1e-9 {
		t.Fatalf("blast shield jolt impulse %.3f vs plain %.3f", st.vx[st.joltFrames[0]], pt.vx[pt.joltFrames[0]])
	}
}

func TestFlamethrowerIgnitionIsShorterThanOtherFire(t *testing.T) {
	p := burnTestPlayer(2)
	p.Team = 2
	b := &Bullet{DevelopedProjectile: DevelopedProjectileFlame, X: 300, Y: 80, Firepower: 1.45, OwnerID: 1, Team: 1, Facing: 1, Time: 6, IgniteFrames: 100, VisualScale: 1}
	if !hitPlayerWithFlame(b, []*Player{p}) {
		t.Fatal("flame missed")
	}
	if p.BurnTime != 50 {
		t.Fatalf("flamethrower burn=%d want50", p.BurnTime)
	}
	// Flare keeps its full (Flare Gun spec) burn: 135 ticks = 4.5 s.
	flareSpec, _ := developedWeaponSpec(181)
	if flareSpec.IgniteFrames != 135 {
		t.Fatalf("flare gun ignite=%d want135", flareSpec.IgniteFrames)
	}
	q := burnTestPlayer(3)
	fl := Bullet{DevelopedProjectile: DevelopedProjectileFlare, X: 300, Y: 80, VX: 1, Firepower: 10, Facing: 1, OwnerID: 1, Lifetime: 20, IgniteFrames: flareSpec.IgniteFrames}
	updateDevelopedProjectile(Map{}, &fl, []*Player{q}, nil, nil)
	if q.BurnTime != 135 {
		t.Fatalf("flare burn=%d want135", q.BurnTime)
	}
}

func TestBurnPanicIsDeterministic(t *testing.T) {
	a, b := burnTestPlayer(7), burnTestPlayer(7)
	a.igniteWithPower(1, 1, 400, 20)
	b.igniteWithPower(1, 1, 400, 20)
	ta, tb := runBurn(a, 400), runBurn(b, 400)
	if len(ta.vx) != len(tb.vx) {
		t.Fatal("different burn lengths")
	}
	for i := range ta.vx {
		if ta.vx[i] != tb.vx[i] {
			t.Fatalf("frame %d: vx %.3f vs %.3f", i, ta.vx[i], tb.vx[i])
		}
	}
	// A fixed seed reproduces the exact sequence too.
	c, d := burnTestPlayer(1), burnTestPlayer(9)
	c.igniteWithPower(1, 1, 400, 20)
	d.igniteWithPower(1, 1, 400, 20)
	c.BurnRNG, d.BurnRNG = 12345, 12345
	c.BurnJoltTimer, d.BurnJoltTimer = 10, 10
	c.BurnPanicTimer, d.BurnPanicTimer = 8, 8
	tc, td := runBurn(c, 400), runBurn(d, 400)
	for i := range tc.vx {
		if tc.vx[i] != td.vx[i] {
			t.Fatalf("seeded frame %d differs", i)
		}
	}
}

func TestBotPatsDownFireAndAvoidsEdgesWhenSkilled(t *testing.T) {
	arena := Map{Platforms: []Rect{{X: 0, Y: 100, W: 400, H: 30}}, Gravity: 0.6, LowestY: 400}
	g := &Game{arena: arena}

	skilled := burnTestPlayer(2)
	skilled.AI = true
	skilled.AISpecial = sourceAISpecialBoss // skill 1
	skilled.X = 30                          // left edge within aiBurnEdgeLook
	skilled.igniteWithPower(1, 1, 200, 20)
	skilled.BurnTick = aiBurnPatDelayQuick
	skilled.AILeft = true
	aiHandleBurning(skilled, g)
	if !skilled.AIUp {
		t.Fatal("skilled bot did not jump to pat down the fire")
	}
	if skilled.AILeft || !skilled.AIRight {
		t.Fatal("skilled burning bot kept walking toward the edge")
	}

	g.campaignMode, g.campaignLevel = true, 1 // skill 0
	novice := burnTestPlayer(3)
	novice.AI = true
	novice.X = 30
	novice.igniteWithPower(1, 1, 200, 20)
	novice.BurnTick = 60
	novice.AILeft = true
	aiHandleBurning(novice, g)
	if novice.AIUp || !novice.AILeft {
		t.Fatal("novice bot reacted to burning; it should just panic")
	}
}

func TestShieldBearerDropsShieldWhileBurning(t *testing.T) {
	g, _ := enemyTestGame()
	e := shieldTestBearer(g)
	e.Body.igniteWithPower(1, -1, 120, 20)
	g.updateCampaignEnemies(g.arena)
	if e.shieldUp() {
		t.Fatal("burning shield-bearer kept the shield up")
	}
}

func TestDroneBurnIsShort(t *testing.T) {
	g, _ := enemyTestGame()
	d := spawnTestEnemy(g, campaignEnemyDrone, 4)
	d.Body.igniteWithPower(1, 1, 165, 20)
	g.updateCampaignEnemies(g.arena)
	if d.Body.BurnTime > droneBurnMaxFrames || d.Body.BurnPanicTime != 0 {
		t.Fatalf("drone burn=%d panic=%d", d.Body.BurnTime, d.Body.BurnPanicTime)
	}
}

func TestBurnFieldsSurviveSnapshot(t *testing.T) {
	p := burnTestPlayer(2)
	p.igniteWithPower(1, -1, 120, 20)
	p.BurnPanicTime, p.BurnPanicDir, p.BurnTick = 7, -1, 33
	state := netSnapshot{Players: []Player{*p}}
	data := netEncodeValue(nil, &state)
	var got netSnapshot
	if _, err := netDecodeValue(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Players) != 1 {
		t.Fatal("player lost")
	}
	q := got.Players[0]
	if q.BurnTime != 120 || q.BurnMax != 120 || q.BurnPanicTime != 7 || q.BurnPanicDir != -1 || q.BurnTick != 33 || q.BurnDirection != -1 {
		t.Fatalf("synced burn fields lost: %+v", q)
	}
	if q.BurnRNG != 0 || q.BurnJoltTimer != 0 || q.BurnPanicTimer != 0 || q.BurnIgnitions != 0 {
		t.Fatal("host-only burn state leaked onto the wire")
	}
}

func TestFlamethrowerStreamPushesEveryThreeTicks(t *testing.T) {
	p := burnTestPlayer(2)
	p.Team = 2
	pulses := 0
	for tick := 0; tick < 30; tick++ {
		if p.FireContactCooldown > 0 {
			p.FireContactCooldown--
		}
		before := p.FireContactCooldown
		b := &Bullet{DevelopedProjectile: DevelopedProjectileFlame, X: 300, Y: 80, Firepower: 1.45, OwnerID: 1, Team: 1, Facing: 1, Time: 6, IgniteFrames: 100, VisualScale: 1}
		hitPlayerWithFlame(b, []*Player{p})
		if before == 0 && p.FireContactCooldown == flamethrowerContactCooldown {
			pulses++
		}
	}
	if pulses != 30/flamethrowerContactCooldown {
		t.Fatalf("flame contact pulses=%d in 30 ticks want %d", pulses, 30/flamethrowerContactCooldown)
	}
}
