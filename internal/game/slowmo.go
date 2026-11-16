package game

import "math"

// slowmo.go: SLOW-MO for everyone — the crate powerup (powerupSlowMo) and the
// Chaos event SLOW-MO EVERYONE.
//
// The whole match runs at half speed for slowMoRealTicks REAL ticks (5 s):
// every player including the picker, bots, bullets, grenades, knockback,
// zombies, campaign enemies and effects.
//
// How: Update keeps its fixed 30 TPS, but the gameplay simulation (the arena
// step and the per-tick match rules around it) advances through a step
// accumulator. Every real tick adds slowMoScale; a sim step runs when the
// accumulator reaches 1. Nothing scales velocities or timers by hand, so the
// physics, AI and balance are bit-identical per sim step. Menus, the pause
// screen, the zombie shop UI and the network keep the real tick rate.
//
// Input: on a skipped tick each human's keys (or the remote input frame the
// host consumed) are OR-ed into Player.slowMoLatch; the next sim step reads
// the latch together with the live keys, so a tap that starts and ends
// between two sim steps still registers. (A release AND re-press inside one
// sim step collapse into one press: the step cannot see both.)
//
// Rendering: render_interp.go draws between the last two SIM steps; the
// capture of a skipped tick is undone (keepPrevPair) and alpha is
// phase + f*increment (renderPhase), so slow motion is smooth at any display
// rate. Online the snapshot carries the clock (SlowMoTicks, SlowMoAcc), the
// host holds the input acknowledgement on skipped ticks, and the guest groups
// its own unacknowledged frames with the same clock (netplay_predict.go).

const (
	slowMoRealTicks = 150 // 5 s of real time (75 sim steps)
	slowMoScale     = 0.5 // sim steps per real tick while slowed
	// Gameplay sounds play slower and lower while slowed; the pickup cue is
	// a deep, half-speed EMP.
	slowMoSFXRate = 0.72
	slowMoCueRate = 0.5
	// Feedback ramps (real ticks).
	slowMoFadeIn  = 8
	slowMoFadeOut = 18
)

// Input bits of packNetInput used by the latch checks.
const (
	netInputBitShoot   = 16
	netInputBitGrenade = 32
	netInputBitAbility = 64
)

// simClock is the slow-motion state of a match.
type simClock struct {
	left int     // real ticks of slow motion left
	acc  float64 // sim-step accumulator (0 or 0.5 while slowed)
}

func (c simClock) active() bool { return c.left > 0 }

// advance runs the clock for one real gameplay tick and returns the number
// of sim steps (0 or 1) to run. A half step pending when the slow motion
// ends is completed on the next tick.
func (c simClock) advance() (int, simClock) {
	if c.left <= 0 {
		return 1, simClock{}
	}
	c.left--
	c.acc += slowMoScale
	if c.acc >= 1 {
		c.acc--
		return 1, c
	}
	return 0, c
}

// renderPhase: Draw shows alpha = phase + f*inc of the way from the previous
// sim step to the current one, f being the fraction of the real tick elapsed.
// inc is what the next real tick adds, so alpha reaches 1 exactly when the
// next sim step lands.
func (c simClock) renderPhase() (phase, inc float64) {
	switch {
	case c.left > 0:
		return c.acc, slowMoScale
	case c.acc > 0:
		return c.acc, 1 - c.acc
	}
	return 0, 1
}

// lastStepClock is the clock right after the most recent sim step, given the
// clock after a real tick (a snapshot). Every tick since that step was a
// skipped slowed tick, each adding slowMoScale and using one tick of left.
func (c simClock) lastStepClock() simClock {
	if c.acc <= 0 {
		return simClock{left: c.left}
	}
	n := int(math.Round(c.acc / slowMoScale))
	return simClock{left: c.left + n}
}

// validSimClock bounds a clock received from the network.
func validSimClock(left int, acc float64) bool {
	return left >= 0 && left <= slowMoRealTicks && !math.IsNaN(acc) && acc >= 0 && acc < 1
}

// startSlowMo starts (or refreshes) the slow motion. A second pickup while
// active restarts the 5 s; the speed never drops below half.
func (g *Game) startSlowMo() {
	g.slowMo.left = slowMoRealTicks
}

// slowMoSounding: gameplay sounds play slowed.
func (g *Game) slowMoSounding() bool {
	return g != nil && g.screen == screenGameplay && (g.slowMo.left > 0 || g.slowMo.acc > 0)
}

// beginSimTick advances the slow-motion clock for one real gameplay tick and
// reports whether the simulation steps. On a skipped tick it latches the
// human input and keeps the render interpolation on the last two sim steps.
func (g *Game) beginSimTick() bool {
	steps, next := g.slowMo.advance()
	g.slowMo = next
	g.renderInterp.simPhase, g.renderInterp.simInc = next.renderPhase()
	if steps > 0 {
		return true
	}
	g.slowMoSkipped = true
	g.latchSlowMoInput()
	g.renderInterp.keepPrevPair(0)
	return false
}

// latchSlowMoInput records the human input of a skipped tick.
func (g *Game) latchSlowMoInput() {
	for _, p := range g.players {
		if p == nil || (p.AI && !p.RemoteControlled) {
			continue
		}
		var in PlayerInputState
		if p.RemoteControlled {
			in = p.RemoteInput
		} else {
			in = capturePlayerInput(p.Controls)
		}
		p.slowMoLatch |= packNetInput(in)
	}
}

// clearSlowMoLatches runs after a sim step consumed the latched input (and
// when the game pauses, so a stale tap does not fire after the pause).
func (g *Game) clearSlowMoLatches() {
	for _, p := range g.players {
		if p != nil {
			p.slowMoLatch = 0
		}
	}
}

// onSlowMoPickup: the crate powerup and the Chaos event start the effect with
// a deep pickup cue on top of the usual powerup sound.
func (g *Game) onSlowMoPickup() {
	g.startSlowMo()
	g.playSourceSFXRate("emp.wav", true, 1, slowMoCueRate)
}

// slowMoFeedbackAlpha is the strength (0..1) of the screen tint and the timer
// bar: a short fade in after the pickup and a fade out at the end.
func (g *Game) slowMoFeedbackAlpha() float64 {
	if g == nil || g.slowMo.left <= 0 {
		return 0
	}
	a := 1.0
	if in := float64(slowMoRealTicks-g.slowMo.left) / slowMoFadeIn; in < a {
		a = in
	}
	if out := float64(g.slowMo.left) / slowMoFadeOut; out < a {
		a = out
	}
	return math.Max(0.15, math.Min(1, a))
}
