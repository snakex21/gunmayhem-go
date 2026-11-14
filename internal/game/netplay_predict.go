package game

import (
	"math"
	"sync"
	"time"
)

// netplay_predict.go: client-side prediction of the guest's own player.
//
// The host stays authoritative. Every snapshot carries InputAck, the newest
// input frame of this guest that the host had simulated (protocol 21). The
// guest takes its own player from the NEWEST snapshot and re-runs the same
// per-player movement step the host runs (Player.UpdateWithFire plus the
// weapon step, so recoil is predicted too) for every input frame the host has
// not simulated yet. The result is drawn instead of the delayed playout copy,
// so the guest's own character answers its keys immediately.
//
// Reconciliation: because the prediction always restarts from host state,
// knockback, portals, deaths and every other host-side effect are adopted on
// the next snapshot; the prediction never resists them. A small difference
// between the previous prediction and the reconciled one is hidden by a
// display offset that decays over a few ticks. Large corrections, respawns
// and new hits snap at once.

const (
	// netPredictMaxFrames bounds the replay (3 s of input). With a longer
	// unacknowledged backlog the link is stalled; show the host state.
	netPredictMaxFrames = 90
	netPredictLogMax    = 128
	// netPredictSnapPx: corrections larger than this are not smoothed.
	netPredictSnapPx = 48.0
	// netPredictSmoothKeep is the part of a small correction still shown
	// one tick later (0.5 -> ~90 % hidden after 3 ticks / 100 ms).
	netPredictSmoothKeep = 0.5
)

type netPredictInput struct {
	seq  uint32
	bits uint8
}

// netPredictEnv is the arena state a prediction step needs besides the player.
type netPredictEnv struct {
	arena      Map // runtime gravity/friction already applied
	leg1, leg2 int
	head       []SourceTransformFrame
}

// netPredictor runs on the client's game thread only.
type netPredictor struct {
	log []netPredictInput // own input frames, ascending seq

	have          bool
	prev          Player // last prediction (without display offset)
	prevSeq       uint32
	baseTick      uint32
	baseSerial    int
	baseHitTimer  int
	offX, offY    float64
	stats         netPredictStats
	disabled      bool // Ctrl+F6 A/B switch
	lastAppliedOK bool
}

// netPredictStats feeds the F6 network overlay.
type netPredictStats struct {
	unacked   int
	lastError float64 // size of the newest reconciliation (px)
	maxError  float64 // largest smoothed (non-snapped) correction in the window
	snaps     int
	offset    float64 // display offset still being blended out (px)
	windowAt  time.Time
}

func (p *netPredictor) record(seq uint32, in PlayerInputState) {
	if p == nil || seq == 0 {
		return
	}
	if n := len(p.log); n > 0 && seq <= p.log[n-1].seq {
		p.log = p.log[:0] // a new session restarted the numbering
	}
	p.log = append(p.log, netPredictInput{seq: seq, bits: packNetInput(in)})
	if len(p.log) > netPredictLogMax {
		p.log = append(p.log[:0], p.log[len(p.log)-netPredictLogMax:]...)
	}
}

// reset forgets the prediction (not the input log).
func (p *netPredictor) reset() {
	if p == nil {
		return
	}
	p.have = false
	p.offX, p.offY = 0, 0
	p.stats.unacked = 0
	p.stats.offset = 0
	p.lastAppliedOK = false
}

// predictStep advances a predicted player by one host tick with one input
// frame, exactly like the host's per-player step in Game.Update (movement,
// shooting/recoil inside the update, grenade wind-up, then the visual
// timelines). It reports
// false, leaving p untouched, when the step would kill or respawn the player:
// deaths are decided by the host only.
func predictStep(p *Player, in PlayerInputState, env *netPredictEnv) bool {
	before := *p
	p.RemoteControlled = true
	p.AI = false
	p.RemoteInput = in
	var bullets []Bullet
	var shells []Shell
	var flashes []MuzzleFlash
	fired := false
	p.UpdateWithFire(env.arena, nil, nil, func() {
		fired = true
		updateWeapon(p, &bullets, &shells, &flashes)
	})
	if !p.Active || p.DeathSerial != before.DeathSerial {
		*p = before
		return false
	}
	if !fired {
		updateWeapon(p, &bullets, &shells, &flashes)
	}
	p.PendingSounds = p.PendingSounds[:0]
	// Grenade wind-up (the charge indicator); a thrown grenade itself is
	// the host's, so it goes to a scratch list.
	var grenades []Grenade
	updateGrenadeInput(p, &grenades)
	p.advanceLegTimelines(env.leg1, env.leg2)
	p.advanceHeadTimeline(env.head)
	p.updateSourceVisualState()
	return true
}

// replay runs the logged inputs in (from, to] on p. It returns false when an
// input frame is missing from the log.
func (p *netPredictor) replay(pl *Player, from, to uint32, env *netPredictEnv) bool {
	if to <= from {
		return true
	}
	next := from + 1
	for _, in := range p.log {
		if in.seq < next {
			continue
		}
		if in.seq > to {
			break
		}
		if in.seq != next {
			return false
		}
		if !predictStep(pl, unpackNetInput(in.bits), env) {
			return true // the host decides the death; keep the last alive state
		}
		next++
	}
	return next == to+1
}

// predict returns the own player to draw at input frame latest, given the
// newest host state base (snapshot tick baseTick, which already contains the
// guest's inputs up to ack). ok is false when the host state should be shown
// unchanged.
func (p *netPredictor) predict(base Player, baseTick, ack, latest uint32, env netPredictEnv) (Player, bool) {
	if ack > latest {
		// The host repeated input for frames still in flight; everything
		// this client sent is accounted for.
		ack = latest
	}
	if p == nil || p.disabled || !base.Active || latest-ack > netPredictMaxFrames {
		p.reset()
		return base, false
	}
	// Inputs the host has simulated are no longer needed.
	drop := 0
	for drop < len(p.log) && p.log[drop].seq <= ack {
		drop++
	}
	if drop > 0 {
		p.log = append(p.log[:0], p.log[drop:]...)
	}

	pred := base
	pred.PendingSounds = nil // never share the snapshot's backing array
	if !p.replay(&pred, ack, latest, &env) {
		p.reset()
		return base, false
	}

	if p.have {
		cont := p.prev
		cont.PendingSounds = nil
		if !p.replay(&cont, p.prevSeq, latest, &env) {
			cont = pred
		}
		ex, ey := cont.X-pred.X, cont.Y-pred.Y
		err := math.Hypot(ex, ey)
		respawned := base.DeathSerial != p.baseSerial
		hit := baseTick != p.baseTick && base.HitTimer < p.baseHitTimer
		if baseTick != p.baseTick {
			p.stats.lastError = err
		}
		if err > netPredictSnapPx || respawned || hit {
			if err > 0.01 {
				p.stats.snaps++
			}
			p.offX, p.offY = 0, 0
		} else {
			p.offX += ex
			p.offY += ey
			p.stats.maxError = math.Max(p.stats.maxError, err)
		}
	}
	p.have = true
	p.prev = pred
	p.prev.PendingSounds = nil
	p.prevSeq = latest
	p.baseTick = baseTick
	p.baseSerial = base.DeathSerial
	p.baseHitTimer = base.HitTimer
	p.stats.unacked = int(latest - ack)

	out := pred
	out.X += p.offX
	out.Y += p.offY
	p.stats.offset = math.Hypot(p.offX, p.offY)
	p.offX *= netPredictSmoothKeep
	p.offY *= netPredictSmoothKeep
	if math.Abs(p.offX) < 0.05 {
		p.offX = 0
	}
	if math.Abs(p.offY) < 0.05 {
		p.offY = 0
	}
	p.lastAppliedOK = true
	return out, true
}

// predictionBase returns the guest's own player from the newest snapshot,
// with that snapshot's tick, input acknowledgement and physics.
func (s *netplaySession) predictionBase(playerID int) (base Player, tick, ack uint32, arena Map, ok bool) {
	if s == nil || s.mode != netplayClient || !s.hasState.Load() {
		return Player{}, 0, 0, Map{}, false
	}
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	st := &s.state
	if gameScreen(st.Screen) != screenGameplay || st.GameWin || st.Paused || st.FadeActive {
		return Player{}, 0, 0, Map{}, false
	}
	for i := range st.Players {
		if st.Players[i].ID == playerID && !st.Players[i].IsDouble {
			arena = st.Arena
			if st.RuntimeGravity != 0 {
				arena.Gravity = st.RuntimeGravity
			}
			arena.LowFriction = st.RuntimeLowFriction
			return st.Players[i], s.stateTick, s.stateInputAck, arena, true
		}
	}
	return Player{}, 0, 0, Map{}, false
}

// applyNetPrediction replaces the guest's own player (taken from the delayed
// playout state) with the predicted one. latest is the input frame queued
// this tick.
func (g *Game) applyNetPrediction(latest uint32) {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayClient {
		return
	}
	id := g.netplay.assignedPlayer()
	base, tick, ack, arena, ok := g.netplay.predictionBase(id)
	if !ok || g.GameWin || g.paused {
		g.netPredict.reset()
		return
	}
	env := netPredictEnv{arena: arena}
	if g.assets != nil {
		env.leg1, env.leg2 = len(g.assets.Leg1Timeline), len(g.assets.Leg2Timeline)
		env.head = g.assets.HeadTimeline
	}
	pred, ok := g.netPredict.predict(base, tick, ack, latest, env)
	if !ok {
		return
	}
	for i, p := range g.players {
		if p != nil && p.ID == id && !p.IsDouble {
			cp := pred
			g.players[i] = &cp
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Snapshot arrival statistics (F6 overlay)

// netRateMeter records arrival times (reader goroutine) for the overlay.
type netRateMeter struct {
	mu    sync.Mutex
	times []time.Time
}

const netRateMeterKeep = 96

func (m *netRateMeter) note(t time.Time) {
	m.mu.Lock()
	m.times = append(m.times, t)
	if len(m.times) > netRateMeterKeep {
		m.times = append(m.times[:0], m.times[len(m.times)-netRateMeterKeep:]...)
	}
	m.mu.Unlock()
}

// stats returns arrivals per second over the last second, and the mean
// absolute deviation of the arrival interval from one host tick (jitter).
func (m *netRateMeter) stats(now time.Time) (rate float64, jitterMS float64, maxGapMS float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	from := now.Add(-time.Second)
	var prev time.Time
	n := 0
	var dev float64
	gaps := 0
	for _, t := range m.times {
		if t.Before(from) {
			prev = t
			continue
		}
		n++
		if !prev.IsZero() {
			gap := float64(t.Sub(prev)) / float64(time.Millisecond)
			dev += math.Abs(gap - 1000.0/TicksPerSecond)
			maxGapMS = math.Max(maxGapMS, gap)
			gaps++
		}
		prev = t
	}
	if gaps > 0 {
		jitterMS = dev / float64(gaps)
	}
	return float64(n), jitterMS, maxGapMS
}

// stats reports the playout buffer: snapshots buffered past the frame being
// shown, and the target delay in ticks.
func (p *netPlayout) stats() (buffered int, delay int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, f := range p.frames {
		if int64(f.tick) > p.playTick {
			buffered++
		}
	}
	return buffered, p.delay
}
