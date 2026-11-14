package game

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const renderTestTick = time.Second / TicksPerSecond

func TestRenderAlphaIsFractionOfTick(t *testing.T) {
	t0 := time.Unix(100, 0)
	for _, c := range []struct {
		dt   time.Duration
		want float64
	}{
		{0, 0}, {renderTestTick / 4, 0.25}, {renderTestTick / 2, 0.5}, {renderTestTick, 1}, {3 * renderTestTick, 1}, {-time.Millisecond, 0},
	} {
		if got := renderAlphaAt(t0, t0.Add(c.dt)); math.Abs(got-c.want) > 1e-6 {
			t.Fatalf("alpha(%v)=%v want %v", c.dt, got, c.want)
		}
	}
	if renderAlphaAt(time.Time{}, t0) != 1 {
		t.Fatal("no update yet must draw the current state")
	}
}

func TestLerpAngleTakesShortArcAndSnapsSpins(t *testing.T) {
	if got := lerpAngleDeg(350, 10, 0.5); math.Abs(math.Mod(got+360, 360)) > 1e-9 {
		t.Fatalf("350->10 half way = %v, want 0", got)
	}
	if got := lerpAngleDeg(0, 90, 0.5); got != 45 {
		t.Fatalf("0->90 half way = %v", got)
	}
	if got := lerpAngleDeg(0, 170, 0.25); got != 170 {
		t.Fatalf("a 170 degree jump must snap, got %v", got)
	}
}

func TestRenderPlayerSnapRules(t *testing.T) {
	prev := renderSample{x: 100, y: 200, vx: 8, vy: -4, key: 1, serial: 3, active: true}
	moved := renderSample{x: 108, y: 197, vx: 8, vy: -3, key: 1, serial: 3, active: true}
	if !renderPlayerContinues(prev, moved) {
		t.Fatal("ordinary movement must interpolate")
	}
	knocked := prev
	knocked.vx = 45 // knockback velocity from the previous tick
	far := moved
	far.x = prev.x + 45
	if !renderPlayerContinues(knocked, far) {
		t.Fatal("knockback flight must interpolate (it continues the velocity)")
	}
	respawned := moved
	respawned.serial = 4
	if renderPlayerContinues(prev, respawned) {
		t.Fatal("respawn (new death serial) must snap")
	}
	portal := moved
	portal.x = 700
	if renderPlayerContinues(prev, portal) {
		t.Fatal("portal teleport must snap")
	}
	wrapped := moved
	wrapped.x = prev.x + 80 // not explained by vx=8
	if renderPlayerContinues(prev, wrapped) {
		t.Fatal("unexplained jump must snap")
	}
	dead := moved
	dead.active = false
	if renderPlayerContinues(prev, dead) {
		t.Fatal("inactive player must not interpolate")
	}
}

func TestMatchRenderSamplesSkipsDeadAndNewObjects(t *testing.T) {
	prev := []renderSample{
		{x: 0, y: 0, vx: 30, key: 1, active: true},
		{x: 0, y: 50, vx: 30, key: 1, active: true}, // dies this tick
		{x: 0, y: 100, vx: 30, key: 1, active: true},
	}
	cur := []renderSample{
		{x: 30, y: 0, vx: 30, key: 1, active: true},
		{x: 30, y: 100, vx: 30, key: 1, active: true},
		{x: 500, y: 300, vx: -30, key: 1, active: true}, // fired this tick
	}
	got := matchRenderSamples(prev, cur, nil)
	if want := []int{0, 2, -1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("match=%v want %v", got, want)
	}
	// A different kind is never matched.
	cur[0].key = 2
	if got := matchRenderSamples(prev, cur[:1], nil); got[0] != -1 {
		t.Fatalf("kind mismatch matched: %v", got)
	}
}

func newRenderInterpTestGame() *Game {
	g := New()
	g.screen = screenGameplay
	g.arena = OriginalMap1()
	g.displaySmoothMotion = true
	return g
}

func TestRenderInterpolationDrawsBetweenTicksAndRestores(t *testing.T) {
	g := newRenderInterpTestGame()
	p := g.players[0]
	p.X, p.Y, p.VX, p.VY = 100, 200, 10, 0
	g.bullets = []Bullet{{X: 50, Y: 80, VX: 40, Facing: 1}}
	g.grenades = []Grenade{{X: 300, Y: 100, VX: 4, VY: -6, Rotation: 10}}
	g.cameraX = 0
	t0 := time.Unix(1000, 0)
	g.captureRenderPrev(t0)

	// One tick of simulated motion.
	p.X += p.VX
	g.bullets[0].X += 40
	g.bullets = append(g.bullets, Bullet{X: 600, Y: 80, VX: -40, Facing: -1}) // new this tick
	g.grenades[0].X += 4
	g.grenades[0].Y -= 6
	g.grenades[0].Rotation = 30
	g.cameraX = 4
	origPlayers, origBullets := g.players, g.bullets
	before := g.makeNetSnapshot()

	restore := g.beginRenderInterpolation(t0.Add(renderTestTick / 2))
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-4 }
	if got := g.players[0].X; !near(got, 105) {
		t.Fatalf("player drawn at %v, want 105", got)
	}
	if got := g.bullets[0].X; !near(got, 70) {
		t.Fatalf("bullet drawn at %v, want 70", got)
	}
	if got := g.bullets[1].X; got != 600 {
		t.Fatalf("new bullet must be drawn where it spawned, got %v", got)
	}
	if gr := g.grenades[0]; !near(gr.X, 302) || !near(gr.Y, 97) || !near(gr.Rotation, 20) {
		t.Fatalf("grenade drawn at %+v", gr)
	}
	if !near(g.cameraX, 2) {
		t.Fatalf("camera drawn at %v, want 2", g.cameraX)
	}
	if p.X != 110 || origBullets[0].X != 90 {
		t.Fatal("interpolation wrote into the simulation state")
	}
	restore()
	if !reflect.DeepEqual(g.makeNetSnapshot(), before) {
		t.Fatal("tick state differs after Draw interpolation")
	}
	if &g.players[0] != &origPlayers[0] || g.players[0] != p || &g.bullets[0] != &origBullets[0] {
		t.Fatal("entity lists were not restored")
	}
}

func TestRenderInterpolationSnapsAcrossMapChangeAndWhenOff(t *testing.T) {
	g := newRenderInterpTestGame()
	p := g.players[0]
	p.X, p.VX = 100, 10
	t0 := time.Unix(1000, 0)
	g.captureRenderPrev(t0)
	p.X = 110
	half := t0.Add(renderTestTick / 2)

	g.displaySmoothMotion = false
	restore := g.beginRenderInterpolation(half)
	if g.players[0].X != 110 {
		t.Fatalf("SMOOTH MOTION off still interpolated: %v", g.players[0].X)
	}
	restore()

	g.displaySmoothMotion = true
	g.displayPowerSave = true
	restore = g.beginRenderInterpolation(half)
	if g.players[0].X != 110 {
		t.Fatal("POWER SAVE must disable interpolation")
	}
	restore()
	g.displayPowerSave = false

	g.arena.Number = 2 // the map changed during the tick
	restore = g.beginRenderInterpolation(half)
	if g.players[0].X != 110 {
		t.Fatal("interpolated across a map change")
	}
	restore()
}

// Drawing between ticks must never change the simulation: run a short local
// match and draw at several alphas between updates.
func TestRenderInterpolationLeavesSimulationUntouched(t *testing.T) {
	g := newRenderInterpTestGame()
	g.fadeActive = false
	screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
	for _, p := range g.players {
		p.AI = true
	}
	for tick := 0; tick < 45; tick++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		ref := g.makeNetSnapshot()
		for _, frac := range []time.Duration{0, renderTestTick / 3, 2 * renderTestTick / 3} {
			g.renderInterp.lastUpdate = time.Now().Add(-frac)
			g.Draw(screen)
			if got := g.makeNetSnapshot(); !reflect.DeepEqual(ref, got) {
				t.Fatalf("tick %d: Draw at alpha %.2f changed the simulation state", tick, float64(frac)/float64(renderTestTick))
			}
		}
	}
}

func TestFrameLimiterPacesToCap(t *testing.T) {
	var l frameLimiter
	t0 := time.Unix(50, 0)
	interval := time.Second / 60
	if d := l.delay(t0, 60); d != 0 {
		t.Fatalf("first frame waited %v", d)
	}
	// A frame drawn 4 ms later must wait for the rest of the 60 Hz slot.
	if d := l.delay(t0.Add(4*time.Millisecond), 60); d != interval-4*time.Millisecond {
		t.Fatalf("wait=%v want %v", d, interval-4*time.Millisecond)
	}
	// After a long hitch the limiter restarts instead of bursting.
	if d := l.delay(t0.Add(time.Second), 60); d != 0 {
		t.Fatalf("after hitch waited %v", d)
	}
	if d := l.delay(t0, 0); d != 0 || !l.next.IsZero() {
		t.Fatal("uncapped mode must not wait")
	}
	for mode, want := range map[int]int{displayFrameRateVSync: 0, displayFrameRate60: 60, displayFrameRate120: 120, displayFrameRate144: 144, displayFrameRateUnlimited: 0} {
		if got := displayFrameRateCap(mode); got != want {
			t.Fatalf("cap(%d)=%d want %d", mode, got, want)
		}
		if displayFrameRateVSyncOn(mode) != (mode == displayFrameRateVSync) {
			t.Fatalf("vsync(%d) wrong", mode)
		}
	}
}
