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
//
// SLOW-MO (protocol 26). Assumption: the host consumes one of this guest's
// input frames per real tick (netInputReplay), so frame ack+k lands on the
// k-th host tick after the last sim step. Jitter (repeated or dropped
// frames, netInputReplay debt) can shift that grouping by a frame; the
// difference is an ordinary small correction, blended like any other.
// The host consumes one input frame per real tick but
// simulates only every other tick, OR-ing the frames in between (slowmo.go),
// and its InputAck stays at the last sim step. The snapshot carries the
// host's clock, so the replay starts from the clock of that last sim step
// (simClock.lastStepClock) and groups the frames exactly like the host. The
// frames of an incomplete group are shown as a partial step towards where
// they lead, so the own character still answers every real tick.

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
	// noDrop: areas where DOWN uses something (zombie mystery box, doors)
	// instead of dropping through the platform, like on the host.
	noDrop []Rect
	// clock: the host's SLOW-MO clock after the base snapshot's tick.
	clock simClock
}

// netPredictPhase is where a replay stands between host sim steps: the host
// clock after the frame, and the frames consumed since the last sim step.
type netPredictPhase struct {
	clock   simClock
	pending uint8 // OR of the frames of the incomplete group
	held    bool  // pending holds at least one frame
}

// netPredictor runs on the client's game thread only.
type netPredictor struct {
	log []netPredictInput // own input frames, ascending seq

	have          bool
	prev          Player // last prediction (without display offset)
	prevPhase     netPredictPhase
	prevSeq       uint32
	baseTick      uint32
	baseSerial    int
	baseHitTimer  int
	offX, offY    float64
	stats         netPredictStats
	disabled      bool // Ctrl+F6 A/B switch
	lastAppliedOK bool
	// holdForward (SLOW-MO corrections): while the offset is blended out,
	// the drawn character may stand still but never moves against the
	// predicted motion. lastShown/lastDrawn are the previous output
	// without and with the offset.
	holdForward          bool
	lastShown, lastDrawn [2]float64
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
	p.holdForward = false
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
	if in.Down {
		for _, r := range env.noDrop {
			if r.Contains(p.X, p.Y) {
				p.DownHeld = true
				break
			}
		}
	}
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

// replay runs the logged inputs in (from, to] on p, one host sim step per
// complete group of frames (one frame per group at normal speed). It returns
// false when an input frame is missing from the log.
func (p *netPredictor) replay(pl *Player, ph *netPredictPhase, from, to uint32, env *netPredictEnv) bool {
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
		steps, clock := ph.clock.advance()
		ph.clock = clock
		bits := ph.pending | in.bits
		next++
		if steps == 0 {
			ph.pending, ph.held = bits, true
			continue
		}
		ph.pending, ph.held = 0, false
		if !predictStep(pl, unpackNetInput(bits), env) {
			return true // the host decides the death; keep the last alive state
		}
	}
	return next == to+1
}

// predictDisplay is the drawn copy of a replayed player: with frames of an
// incomplete SLOW-MO group pending, the position is moved the matching part
// of the way towards the step those frames lead to.
func predictDisplay(pl Player, ph netPredictPhase, env *netPredictEnv) Player {
	if !ph.held || ph.clock.acc <= 0 {
		return pl
	}
	next := pl
	next.PendingSounds = nil
	if !predictStep(&next, unpackNetInput(ph.pending), env) {
		return pl
	}
	t := ph.clock.acc
	pl.X += (next.X - pl.X) * t
	pl.Y += (next.Y - pl.Y) * t
	return pl
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
	// Inputs the host has simulated are no longer needed, except those the
	// previous prediction has not been continued through yet: when the host
	// acknowledges everything (ack == latest, common on LAN) the continuation
	// below still replays (prevSeq, latest] to measure the correction, and
	// without those frames a small correction would show unsmoothed.
	keepAfter := ack
	if p.have && p.prevSeq < keepAfter {
		keepAfter = p.prevSeq
	}
	drop := 0
	for drop < len(p.log) && p.log[drop].seq <= keepAfter {
		drop++
	}
	if drop > 0 {
		p.log = append(p.log[:0], p.log[drop:]...)
	}

	pred := base
	pred.PendingSounds = nil // never share the snapshot's backing array
	phase := netPredictPhase{clock: env.clock.lastStepClock()}
	if !p.replay(&pred, &phase, ack, latest, &env) {
		p.reset()
		return base, false
	}
	shown := predictDisplay(pred, phase, &env)

	if p.have {
		cont := p.prev
		cont.PendingSounds = nil
		contPhase := p.prevPhase
		contShown := shown
		if p.replay(&cont, &contPhase, p.prevSeq, latest, &env) {
			contShown = predictDisplay(cont, contPhase, &env)
		}
		ex, ey := contShown.X-shown.X, contShown.Y-shown.Y
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
			p.holdForward = false
		} else {
			p.offX += ex
			p.offY += ey
			p.stats.maxError = math.Max(p.stats.maxError, err)
			// The start of a SLOW-MO cannot be predicted: the frames sent
			// before the guest heard of it were predicted at full speed, so
			// the character is ahead. Let the slowed prediction catch up
			// (hold) instead of pulling the character backwards.
			if err > 0.01 && (env.clock.active() || env.clock.acc > 0 || p.prevPhase.clock.active() || p.prevPhase.clock.acc > 0) {
				p.holdForward = true
			}
		}
	}
	lastShown, lastDrawn := p.lastShown, p.lastDrawn
	hadDrawn := p.have
	p.have = true
	p.prev = pred
	p.prev.PendingSounds = nil
	p.prevPhase = phase
	p.prevSeq = latest
	p.baseTick = baseTick
	p.baseSerial = base.DeathSerial
	p.baseHitTimer = base.HitTimer
	p.stats.unacked = int(latest - ack)

	out := shown
	if p.holdForward && hadDrawn {
		p.offX = holdNoReverse(p.offX, shown.X, lastShown[0], lastDrawn[0])
		p.offY = holdNoReverse(p.offY, shown.Y, lastShown[1], lastDrawn[1])
	}
	out.X += p.offX
	out.Y += p.offY
	p.lastShown, p.lastDrawn = [2]float64{shown.X, shown.Y}, [2]float64{out.X, out.Y}
	if p.offX == 0 && p.offY == 0 {
		p.holdForward = false
	}
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
func (s *netplaySession) predictionBase(playerID int) (base Player, tick, ack uint32, arena Map, clock simClock, ok bool) {
	if s == nil || s.mode != netplayClient || !s.hasState.Load() {
		return Player{}, 0, 0, Map{}, simClock{}, false
	}
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	st := &s.state
	if gameScreen(st.Screen) != screenGameplay || st.GameWin || st.Paused || st.FadeActive {
		return Player{}, 0, 0, Map{}, simClock{}, false
	}
	clock = simClock{left: st.SlowMoTicks, acc: st.SlowMoAcc}
	for i := range st.Players {
		if st.Players[i].ID == playerID && !st.Players[i].IsDouble {
			arena = st.Arena
			if st.RuntimeGravity != 0 {
				arena.Gravity = st.RuntimeGravity
			}
			arena.LowFriction = st.RuntimeLowFriction
			return st.Players[i], s.stateTick, s.stateInputAck, arena, clock, true
		}
	}
	return Player{}, 0, 0, Map{}, simClock{}, false
}

// applyNetPrediction replaces the guest's own player (taken from the delayed
// playout state) with the predicted one. latest is the input frame queued
// this tick.
func (g *Game) applyNetPrediction(latest uint32) {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayClient {
		return
	}
	id := g.netplay.assignedPlayer()
	base, tick, ack, arena, clock, ok := g.netplay.predictionBase(id)
	if !ok || g.GameWin || g.paused {
		g.netPredict.reset()
		return
	}
	if g.zombieActive() && arena.Number != g.arena.Number {
		// A zone change (multi-zone zombie map): the newest snapshot is in
		// another zone than the playout state on screen. Wait until the
		// playout catches up instead of mixing the two maps.
		g.netPredict.reset()
		return
	}
	env := netPredictEnv{arena: arena, noDrop: g.zombieNoDropAreas(), clock: clock}
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
			// Drawn with the plain tick alpha: it moves every real tick,
			// also in SLOW-MO (render_interp.go).
			g.renderInterp.freeID = id
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

// holdNoReverse returns the display offset for one axis so that the drawn
// position does not move against the predicted motion (shown - lastShown):
// it stays where it was drawn last instead.
func holdNoReverse(off, shown, lastShown, lastDrawn float64) float64 {
	moved := shown - lastShown
	if moved == 0 {
		return off
	}
	if drawnMove := shown + off - lastDrawn; drawnMove*moved < 0 {
		return lastDrawn - shown
	}
	return off
}
