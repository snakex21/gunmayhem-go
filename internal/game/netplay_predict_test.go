package game

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

var timeZero = time.Unix(0, 0)

func simTime(ms float64) time.Time {
	return timeZero.Add(time.Duration(ms * float64(time.Millisecond)))
}

// ---------------------------------------------------------------------------
// Unit tests: replay and reconciliation

func predictTestPlayer() Player {
	m := OriginalMap1()
	p := NewPlayer(2, m)
	p.X, p.Y, p.VX, p.VY = 450, m.Platforms[2].Y, 0, 0
	p.JumpNum, p.Grounded = 2, true
	return *p
}

func predictTestEnv() netPredictEnv { return netPredictEnv{arena: OriginalMap1()} }

func TestPredictReplaysUnackedInputsLikeTheHost(t *testing.T) {
	env := predictTestEnv()
	host := predictTestPlayer()
	right := PlayerInputState{Right: true}
	var pr netPredictor
	// Host simulated frames 1..3; the client has queued 1..8.
	for seq := uint32(1); seq <= 8; seq++ {
		pr.record(seq, right)
	}
	base := host
	for i := 0; i < 3; i++ {
		predictStep(&base, right, &env)
	}
	got, ok := pr.predict(base, 10, 3, 8, env)
	if !ok {
		t.Fatal("no prediction")
	}
	want := host
	for i := 0; i < 8; i++ {
		predictStep(&want, right, &env)
	}
	if math.Abs(got.X-want.X) > 1e-9 || math.Abs(got.Y-want.Y) > 1e-9 || got.Facing != 1 {
		t.Fatalf("predicted %.3f,%.3f want %.3f,%.3f", got.X, got.Y, want.X, want.Y)
	}
	if got.X <= base.X {
		t.Fatal("prediction did not move ahead of the host state")
	}
	if len(pr.log) != 5 {
		t.Fatalf("acknowledged inputs kept: %d", len(pr.log))
	}
}

func TestPredictSmoothsSmallCorrectionsAndSnapsLargeOnes(t *testing.T) {
	env := predictTestEnv()
	right := PlayerInputState{Right: true}
	var pr netPredictor
	base := predictTestPlayer()
	pr.record(1, right)
	first, _ := pr.predict(base, 1, 0, 1, env)

	// The host state comes back 6 px to the left of what was predicted
	// (for example the host ran one stale input): the drawn position must not
	// jump, and the offset must fade out over a few ticks.
	pr.record(2, right)
	base2 := base
	predictStep(&base2, right, &env)
	base2.X -= 6
	got, _ := pr.predict(base2, 2, 1, 2, env)
	cont := first
	predictStep(&cont, right, &env)
	if math.Abs(got.X-cont.X) > 1e-6 {
		t.Fatalf("small correction jumped: drawn %.3f, continuation %.3f", got.X, cont.X)
	}
	if pr.stats.lastError < 5.9 || pr.stats.lastError > 6.1 || pr.stats.snaps != 0 {
		t.Fatalf("error stats %+v", pr.stats)
	}
	for seq := uint32(3); seq < 9; seq++ {
		pr.record(seq, PlayerInputState{})
		pr.predict(base2, 2, 1, seq, env)
	}
	if pr.offX != 0 || pr.offY != 0 {
		t.Fatalf("correction offset not blended out: %v,%v", pr.offX, pr.offY)
	}

	// A large correction (knockback that the guest did not know about) is
	// adopted immediately: the prediction follows the host, it never resists.
	pr.record(9, PlayerInputState{})
	hit := base2
	hit.X += 200
	hit.VX = 30
	hit.HitTimer = 0
	got, _ = pr.predict(hit, 3, 8, 9, env)
	want := hit
	predictStep(&want, PlayerInputState{}, &env)
	if math.Abs(got.X-want.X) > 1e-6 || pr.stats.snaps != 1 {
		t.Fatalf("knockback not adopted: drawn %.2f want %.2f snaps=%d", got.X, want.X, pr.stats.snaps)
	}
}

func TestPredictSnapsOnRespawnAndStopsOnDeathOrStall(t *testing.T) {
	env := predictTestEnv()
	var pr netPredictor
	base := predictTestPlayer()
	pr.record(1, PlayerInputState{Left: true})
	pr.predict(base, 1, 0, 1, env)
	respawned := base
	respawned.DeathSerial++
	respawned.X += 20 // within the smoothing range, but a new life
	pr.record(2, PlayerInputState{})
	got, _ := pr.predict(respawned, 2, 1, 2, env)
	if pr.offX != 0 || math.Abs(got.X-respawned.X) > 10 {
		t.Fatalf("respawn was blended: offset %v drawn %v", pr.offX, got.X)
	}

	// The prediction never kills the player: walking off the map keeps the
	// last alive state; the host decides the death.
	edge := predictTestPlayer()
	edge.X, edge.Y, edge.VY = 300, 990, 25
	edge.JumpNum, edge.Grounded = 0, false
	pr = netPredictor{}
	pr.record(1, PlayerInputState{})
	pr.record(2, PlayerInputState{})
	got, ok := pr.predict(edge, 1, 0, 2, env)
	if !ok || !got.Active || got.DeathSerial != edge.DeathSerial {
		t.Fatalf("prediction killed the player: ok=%v active=%v", ok, got.Active)
	}

	// Too many unacknowledged frames: show the host state.
	pr = netPredictor{}
	for seq := uint32(1); seq <= netPredictMaxFrames+5; seq++ {
		pr.record(seq, PlayerInputState{Right: true})
	}
	if _, ok := pr.predict(base, 1, 0, netPredictMaxFrames+5, env); ok {
		t.Fatal("stalled link still predicted")
	}
	// Dead (inactive) host state: nothing to predict.
	dead := base
	dead.Active = false
	if _, ok := pr.predict(dead, 1, 0, 1, env); ok {
		t.Fatal("predicted an inactive player")
	}
}

func TestNetInputReplayReportsAppliedSequence(t *testing.T) {
	var h netInputHistory
	var r netInputReplay
	var seq uint32
	var frames []uint8
	for i := 0; i < 5; i++ {
		seq, frames = h.record(PlayerInputState{Right: i%2 == 0})
	}
	r.receive(seq, frames, timeZero)
	r.take(timeZero)
	r.take(timeZero)
	if r.appliedSeq != 2 {
		t.Fatalf("appliedSeq=%d want 2", r.appliedSeq)
	}
	for i := 0; i < 5; i++ {
		r.take(timeZero) // 3 frames, then 2 ticks repeat the last input
	}
	if r.appliedSeq != 5 || r.ackSeq() != 7 {
		t.Fatalf("appliedSeq=%d ack=%d want 5/7", r.appliedSeq, r.ackSeq())
	}
	// The late frames arrive with the input that was repeated: they were in
	// effect simulated already and are skipped, the ack does not move back.
	for i := 0; i < 3; i++ {
		seq, frames = h.record(PlayerInputState{Right: true})
	}
	r.receive(seq, frames, timeZero)
	if got := unpackNetInput(r.take(timeZero)); !got.Right {
		t.Fatalf("frame after the stall = %+v", got)
	}
	if r.appliedSeq != 8 || r.ackSeq() != 8 || r.debt != 0 {
		t.Fatalf("after stall appliedSeq=%d ack=%d debt=%d want 8/8/0", r.appliedSeq, r.ackSeq(), r.debt)
	}
}

func TestNetStateEnvelopeCarriesPerPeerInputAck(t *testing.T) {
	var enc netHostEncoder
	frame := enc.encode(netSnapshot{Players: []Player{{ID: 1}, {ID: 2}}})
	frame.inputAck[2] = 77
	frame.inputAck[3] = 5
	payload, rawLen, err := newNetPeerStream().message(frame, 2)
	if err != nil {
		t.Fatal(err)
	}
	got, err := newNetClientStream().decode(payload, rawLen)
	if err != nil {
		t.Fatal(err)
	}
	if got.inputAck != 77 || got.tick != frame.tick {
		t.Fatalf("decoded ack=%d tick=%d", got.inputAck, got.tick)
	}
}

// ---------------------------------------------------------------------------
// Latency / jitter / loss simulation
//
// A real host Game (Game.Update, real bullets and knockback) and a guest
// predictor exchange inputs and snapshots through simulated TCP pipes: fixed
// one-way latency, uniform jitter, and loss modelled the way TCP shows it to
// the application (the segment is retransmitted after an RTO and everything
// behind it waits: head-of-line blocking).

type simPipeEvent[T any] struct {
	at float64
	v  T
}

type simPipe[T any] struct {
	rng                       *rand.Rand
	oneWay, jitter, loss, rto float64
	last                      float64
	q                         []simPipeEvent[T]
	retransmits               int
}

func (p *simPipe[T]) send(now float64, v T) {
	at := now + p.oneWay + p.rng.Float64()*p.jitter
	if p.rng.Float64() < p.loss {
		at += p.rto
		p.retransmits++
	}
	if at < p.last {
		at = p.last // TCP delivers in order
	}
	p.last = at
	p.q = append(p.q, simPipeEvent[T]{at, v})
}

func (p *simPipe[T]) recv(now float64) []T {
	var out []T
	for len(p.q) > 0 && p.q[0].at <= now {
		out = append(out, p.q[0].v)
		p.q = p.q[1:]
	}
	return out
}

type simInputMsg struct {
	seq    uint32
	frames []uint8
}

type simSnapshot struct {
	tick, ack uint32
	self      Player
	remote    Player
	arena     Map
	clock     simClock
}

type predSimConfig struct {
	rttMS, jitterMS, loss float64
	seed                  int64
	script                func(k int) PlayerInputState // guest input per client tick
	hostScript            func(k int) PlayerInputState // host player input per host tick
	ticks                 int
	shootAt               int   // host tick at which the host player fires (0: never)
	slowMoAt              []int // host ticks at which SLOW-MO starts
	statsFrom             int   // client tick at which the correction stats restart (0: start)
}

type predSimResult struct {
	smoothMaxErr   float64 // largest correction that was blended
	snaps          int     // corrections that snapped
	backwardSteady int     // drawn steps against the held direction mid-segment
	maxUnacked     int
	finalDiff      float64 // |drawn - host| after the guest stops
	remoteHolds    int     // client ticks where the remote player froze while moving
	remoteTicks    int
	hostKnockVX    float64
	drawnAfterHit  []float64
	hostAfterHit   []float64
	retransmits    int
}

const simTickMS = 1000.0 / TicksPerSecond

func newPredSimHost() *Game {
	g := New()
	g.screen = screenGameplay
	g.arena = OriginalMap1()
	g.fadeActive = false
	g.CrateON, g.PowerON = false, false
	floor := g.arena.Platforms[2].Y
	for i, p := range g.players {
		p.AI = false
		p.RemoteControlled = true
		p.Lives, p.TotalLives = 99, 99
		p.X, p.Y, p.VX, p.VY = 250+float64(i)*200, floor-1, 0, 0
		p.ShieldTime = 0
	}
	return g
}

func runPredictionSim(t *testing.T, cfg predSimConfig) predSimResult {
	t.Helper()
	rng := rand.New(rand.NewSource(cfg.seed))
	up := &simPipe[simInputMsg]{rng: rng, oneWay: cfg.rttMS / 2, jitter: cfg.jitterMS, loss: cfg.loss, rto: math.Max(200, cfg.rttMS*1.5)}
	down := &simPipe[simSnapshot]{rng: rng, oneWay: cfg.rttMS / 2, jitter: cfg.jitterMS, loss: cfg.loss, rto: math.Max(200, cfg.rttMS*1.5)}

	host := newPredSimHost()
	var replay netInputReplay
	var history netInputHistory
	var pred netPredictor
	var playout netPlayout
	env := netPredictEnv{}

	var res predSimResult
	var newest simSnapshot
	have := false
	hostTick, clientTick := 0, 0
	hostNext, clientNext := 0.0, 7.0
	clientStep := simTickMS * 0.997 // the guest's clock runs slightly fast
	var lastDrawn, lastRemote float64
	var hostAck uint32
	var drawn []float64
	var held []int
	hitTick := -1
	end := float64(cfg.ticks) * simTickMS
	for now := 0.0; now < end; now += 1 {
		if now >= hostNext {
			hostNext += simTickMS
			hostTick++
			for _, m := range up.recv(now) {
				replay.receive(m.seq, m.frames, simTime(now))
			}
			host.players[1].RemoteInput = unpackNetInput(replay.take(simTime(now)))
			in := PlayerInputState{}
			if cfg.hostScript != nil {
				in = cfg.hostScript(hostTick)
			}
			if cfg.shootAt > 0 && hostTick == cfg.shootAt {
				in.Shoot = true
			}
			host.players[0].RemoteInput = in
			for _, at := range cfg.slowMoAt {
				if hostTick == at {
					host.startSlowMo()
				}
			}
			hitTimerBefore := host.players[1].HitTimer
			if err := host.Update(); err != nil {
				t.Fatal(err)
			}
			if cfg.shootAt > 0 && hitTick < 0 && host.players[1].HitTimer < hitTimerBefore {
				hitTick = hostTick
				res.hostKnockVX = host.players[1].VX
			}
			arena := host.customRuntimeArena()
			if !host.slowMoSkipped {
				hostAck = replay.ackSeq() // held on SLOW-MO skipped ticks, like queueStateViews
			}
			snap := simSnapshot{tick: uint32(hostTick), ack: hostAck, self: *host.players[1], remote: *host.players[0], arena: arena, clock: host.slowMo}
			snap.self.PendingSounds, snap.remote.PendingSounds = nil, nil
			down.send(now, snap)
			if hitTick > 0 && hostTick > hitTick && len(res.hostAfterHit) < 40 {
				res.hostAfterHit = append(res.hostAfterHit, host.players[1].X)
			}
		}
		if now >= clientNext {
			clientNext += clientStep
			clientTick++
			for _, s := range down.recv(now) {
				newest, have = s, true
				playout.push(s.tick, netSnapshot{Players: []Player{s.remote}})
			}
			if cfg.statsFrom > 0 && clientTick == cfg.statsFrom {
				pred.stats.maxError, pred.stats.snaps = 0, 0
			}
			in := cfg.script(clientTick)
			seq, frames := history.record(in)
			up.send(now, simInputMsg{seq: seq, frames: frames})
			pred.record(seq, in)
			if !have {
				continue
			}
			env.arena = newest.arena
			env.clock = newest.clock
			p, ok := pred.predict(newest.self, newest.tick, newest.ack, seq, env)
			if !ok {
				p = newest.self
			}
			res.maxUnacked = max(res.maxUnacked, pred.stats.unacked)
			drawn = append(drawn, p.X)
			dir := 0
			if in.Right {
				dir = 1
			} else if in.Left {
				dir = -1
			}
			held = append(held, dir)
			if hitTick > 0 && len(res.drawnAfterHit) < 40 {
				res.drawnAfterHit = append(res.drawnAfterHit, p.X)
			}
			lastDrawn = p.X

			if s, ok := playout.next(); ok && len(s.Players) == 1 {
				x := s.Players[0].X
				if clientTick > 60 && math.Abs(s.Players[0].VX) > 2 {
					res.remoteTicks++
					if x == lastRemote {
						res.remoteHolds++
					}
				}
				lastRemote = x
			}
		}
	}
	// Steady movement: inside a held direction, after the first 25 ticks of
	// the segment (long after any stale-input stall at the switch), the drawn
	// player must never step backwards.
	run := 0
	for i := 1; i < len(drawn); i++ {
		if held[i] != 0 && held[i] == held[i-1] {
			run++
		} else {
			run = 0
		}
		if run >= 25 && (drawn[i]-drawn[i-1])*float64(held[i]) < -0.5 {
			res.backwardSteady++
		}
	}
	res.smoothMaxErr = pred.stats.maxError
	res.snaps = pred.stats.snaps
	res.finalDiff = math.Abs(lastDrawn - host.players[1].X)
	res.retransmits = up.retransmits + down.retransmits
	return res
}

// walkScript: settle, then walk right and left in 40-tick (1.3 s) stretches
// across the bottom floor (optionally jumping once per stretch), then stand
// still so the prediction must converge.
func walkScript(total int, jump bool) func(int) PlayerInputState {
	return func(k int) PlayerInputState {
		if k < 20 || k > total-90 {
			return PlayerInputState{}
		}
		seg := (k - 20) / 40
		in := PlayerInputState{Right: seg%2 == 0, Left: seg%2 == 1}
		if jump && ((k-20)%40 == 12 || (k-20)%40 == 13) {
			in.Up = true
		}
		return in
	}
}

func TestPredictionSimulatedLatencyJitterLoss(t *testing.T) {
	const ticks = 900 // 30 s
	cases := []struct {
		name                  string
		jump                  bool
		rtt, jitter, loss     float64
		maxSmooth             float64
		maxSnaps, maxBackward int
		maxHoldRate           float64
	}{
		// Steady walking: corrections only come from timing (jitter, TCP
		// retransmits); they must stay small and blended, never snap.
		{"lan", false, 2, 1, 0, 0.5, 0, 0, 0.02},
		{"rtt80 jitter30", false, 80, 30, 0, 4, 0, 0, 0.03},
		{"rtt150 jitter30", false, 150, 30, 0, 4, 0, 0, 0.03},
		{"rtt80 jitter30 loss2%", false, 80, 30, 0.02, 16, 0, 0, 0.08},
		{"rtt150 jitter30 loss2%", false, 150, 30, 0.02, 16, 0, 0, 0.08},
		// Jumps that hit a retransmit stall start late on the host (it cannot
		// apply input it does not have yet); those corrections may snap.
		{"jumps rtt150 jitter30", true, 150, 30, 0, 16, 2, 0, 0.03},
		{"jumps rtt150 jitter30 loss2%", true, 150, 30, 0.02, netPredictSnapPx, 12, 0, 0.08},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := runPredictionSim(t, predSimConfig{rttMS: c.rtt, jitterMS: c.jitter, loss: c.loss, seed: 7, ticks: ticks,
				script: walkScript(ticks, c.jump),
				hostScript: func(k int) PlayerInputState {
					return PlayerInputState{Right: (k/50)%2 == 0, Left: (k/50)%2 == 1}
				}})
			holdRate := float64(r.remoteHolds) / math.Max(1, float64(r.remoteTicks))
			t.Logf("%s: blended max %.2f px, snaps %d, backward steady steps %d, max unacked %d, final diff %.3f px, remote holds %d/%d (%.1f%%), retransmits %d",
				c.name, r.smoothMaxErr, r.snaps, r.backwardSteady, r.maxUnacked, r.finalDiff, r.remoteHolds, r.remoteTicks, holdRate*100, r.retransmits)
			if r.smoothMaxErr > c.maxSmooth {
				t.Errorf("blended correction %.2f px > %.2f", r.smoothMaxErr, c.maxSmooth)
			}
			if r.snaps > c.maxSnaps {
				t.Errorf("%d snapped corrections > %d", r.snaps, c.maxSnaps)
			}
			if r.backwardSteady > c.maxBackward {
				t.Errorf("rubber-banding: %d backward steps during steady movement", r.backwardSteady)
			}
			if r.finalDiff > 0.5 {
				t.Errorf("prediction did not converge to the host: %.3f px", r.finalDiff)
			}
			if holdRate > c.maxHoldRate {
				t.Errorf("remote player froze on %.1f%% of moving ticks", holdRate*100)
			}
		})
	}
}

// Being hit by a host-side bullet: the guest's prediction must adopt the
// knockback (move with it), never pull the player back against it.
func TestPredictionAdoptsHostKnockback(t *testing.T) {
	for _, rtt := range []float64{80, 150} {
		r := runPredictionSim(t, predSimConfig{rttMS: rtt, jitterMS: 30, seed: 3, ticks: 240, shootAt: 60,
			script: func(int) PlayerInputState { return PlayerInputState{} },
		})
		if r.hostKnockVX <= 3 {
			t.Fatalf("rtt %.0f: host bullet did not knock the guest back (vx=%.2f)", rtt, r.hostKnockVX)
		}
		// Once the hit is visible to the guest, the drawn player keeps moving
		// in the knockback direction (no backward step against it).
		backward := 0
		for i := 1; i < len(r.drawnAfterHit); i++ {
			if r.drawnAfterHit[i] < r.drawnAfterHit[i-1]-0.5 {
				backward++
			}
		}
		moved := r.drawnAfterHit[len(r.drawnAfterHit)-1] - r.drawnAfterHit[0]
		t.Logf("rtt=%.0fms: knockback vx %.1f, drawn moved %.1f px in 40 ticks, backward steps %d, final diff %.3f px",
			rtt, r.hostKnockVX, moved, backward, r.finalDiff)
		if backward > 0 || moved <= 0 {
			t.Fatalf("rtt %.0f: prediction fought the knockback (backward=%d moved=%.1f)", rtt, backward, moved)
		}
		if r.finalDiff > 0.5 {
			t.Fatalf("rtt %.0f: guest did not end where the host put it (%.3f px)", rtt, r.finalDiff)
		}
	}
}

// On LAN the host often acknowledges the newest input already (ack ==
// latest). The continuation of the previous prediction still needs the frames
// after prevSeq, so a small correction must be smoothed, not shown at once.
func TestPredictSmoothsWhenHostAcksNewestInput(t *testing.T) {
	env := predictTestEnv()
	right := PlayerInputState{Right: true}
	var pr netPredictor
	base := predictTestPlayer()
	pr.record(1, right)
	first, _ := pr.predict(base, 1, 0, 1, env)

	pr.record(2, right)
	base2 := base
	predictStep(&base2, right, &env)
	predictStep(&base2, right, &env)
	base2.X -= 6
	got, ok := pr.predict(base2, 2, 2, 2, env) // ack == latest
	if !ok {
		t.Fatal("no prediction")
	}
	cont := first
	predictStep(&cont, right, &env)
	if math.Abs(got.X-cont.X) > 1e-6 {
		t.Fatalf("correction with ack==latest jumped: drawn %.3f, continuation %.3f", got.X, cont.X)
	}
	if pr.stats.lastError < 5.9 || pr.stats.lastError > 6.1 {
		t.Fatalf("error stats %+v", pr.stats)
	}
	for _, in := range pr.log {
		if in.seq <= 1 {
			t.Fatalf("log kept frames up to min(ack, prevSeq): %+v", pr.log)
		}
	}
}

// ---------------------------------------------------------------------------
// SLOW-MO (slowmo.go)

// The host simulates every other tick and ORs the frames between; the guest
// groups its unacknowledged frames with the host clock, so after a whole
// group the prediction equals the host exactly.
func TestPredictGroupsFramesLikeTheSlowedHost(t *testing.T) {
	env := predictTestEnv()
	host := predictTestPlayer()
	frames := []PlayerInputState{{Right: true}, {Up: true}, {}, {Right: true}, {Right: true}, {Left: true}, {}}
	var pr netPredictor
	for i, in := range frames {
		pr.record(uint32(i+1), in)
	}
	// Host: slow motion started; tick 1 skips (frame 1 latched), tick 2
	// steps with frames 1|2, tick 3 skips (frame 3 latched). Snapshot after
	// tick 3: ack held at frame 2.
	clock := simClock{left: 20}
	base := host
	var latch uint8
	for i := 0; i < 3; i++ {
		steps, next := clock.advance()
		clock = next
		bits := latch | packNetInput(frames[i])
		if steps == 0 {
			latch = bits
			continue
		}
		latch = 0
		predictStep(&base, unpackNetInput(bits), &env)
	}
	env.clock = clock
	got, ok := pr.predict(base, 3, 2, 7, env)
	if !ok {
		t.Fatal("no prediction")
	}
	// The host continues: frames 3|4, 5|6 step; frame 7 is pending.
	want := host
	predictStep(&want, unpackNetInput(packNetInput(frames[0])|packNetInput(frames[1])), &env)
	predictStep(&want, unpackNetInput(packNetInput(frames[2])|packNetInput(frames[3])), &env)
	predictStep(&want, unpackNetInput(packNetInput(frames[4])|packNetInput(frames[5])), &env)
	if pr.prev.X != want.X || pr.prev.Y != want.Y || pr.prev.VY != want.VY {
		t.Fatalf("predicted %.3f,%.3f want %.3f,%.3f", pr.prev.X, pr.prev.Y, want.X, want.Y)
	}
	if !pr.prevPhase.held || pr.prevPhase.pending != packNetInput(frames[6]) {
		t.Fatalf("pending group %+v", pr.prevPhase)
	}
	// Drawn half way towards the step the pending frame leads to.
	next := want
	predictStep(&next, frames[6], &env)
	if math.Abs(got.X-(want.X+(next.X-want.X)*0.5)) > 1e-9 {
		t.Fatalf("drawn %.3f, step %.3f -> %.3f", got.X, want.X, next.X)
	}
}

// Slow motion starting and ending under latency must not rubber-band the
// guest's own character beyond the normal bounds.
func TestPredictionSlowMoUnderLatency(t *testing.T) {
	const ticks = 900
	cases := []struct {
		name              string
		jump              bool
		rtt, jitter, loss float64
		maxSmooth         float64
		maxSnaps          int
	}{
		// The start of a slow motion is a correction the guest cannot
		// avoid (frames already predicted at full speed): it must be
		// blended (no snap) and must never pull the character backwards.
		{"lan", false, 2, 1, 0, 16, 0},
		{"rtt80 jitter30", false, 80, 30, 0, 36, 0},
		{"rtt150 jitter30", false, 150, 30, 0, netPredictSnapPx, 0},
		{"jumps rtt150 jitter30", true, 150, 30, 0, netPredictSnapPx, 2},
		{"rtt150 jitter30 loss2%", false, 150, 30, 0.02, netPredictSnapPx, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Three slow-motion windows, one refreshed while active.
			r := runPredictionSim(t, predSimConfig{rttMS: c.rtt, jitterMS: c.jitter, loss: c.loss, seed: 11, ticks: ticks,
				script:   walkScript(ticks, c.jump),
				slowMoAt: []int{100, 330, 400, 640},
				hostScript: func(k int) PlayerInputState {
					return PlayerInputState{Right: (k/50)%2 == 0, Left: (k/50)%2 == 1}
				}})
			t.Logf("%s: blended max %.2f px, snaps %d, backward steady steps %d, max unacked %d, final diff %.3f px, remote holds %d/%d",
				c.name, r.smoothMaxErr, r.snaps, r.backwardSteady, r.maxUnacked, r.finalDiff, r.remoteHolds, r.remoteTicks)
			if r.smoothMaxErr > c.maxSmooth {
				t.Errorf("blended correction %.2f px > %.2f", r.smoothMaxErr, c.maxSmooth)
			}
			if r.snaps > c.maxSnaps {
				t.Errorf("%d snapped corrections > %d", r.snaps, c.maxSnaps)
			}
			if r.backwardSteady > 0 {
				t.Errorf("rubber-banding: %d backward steps during steady movement", r.backwardSteady)
			}
			if r.finalDiff > 0.5 {
				t.Errorf("prediction did not converge to the host: %.3f px", r.finalDiff)
			}
		})
	}
}

// Inside a long slow motion (after the unavoidable correction at its start)
// the grouped prediction stays within the normal-speed bounds for walking
// and jumping (jitter can shift which frames the host groups together).
// The guest ends still slowed: its drawn character is half a pending step
// ahead of the host's last sim step, hence the 1 px final tolerance.
func TestPredictionSteadySlowMoMatchesTheHost(t *testing.T) {
	const ticks = 900
	var refresh []int
	for k := 60; k < ticks-60; k += 20 {
		refresh = append(refresh, k)
	}
	for _, c := range []struct {
		name        string
		rtt, jitter float64
		maxSmooth   float64
	}{{"lan", 2, 1, 1}, {"rtt80 jitter30", 80, 30, 16}, {"rtt150 jitter30", 150, 30, 16}} {
		r := runPredictionSim(t, predSimConfig{rttMS: c.rtt, jitterMS: c.jitter, seed: 5, ticks: ticks,
			script: walkScript(ticks, true), slowMoAt: refresh, statsFrom: 120})
		t.Logf("steady slow-mo %s: blended max %.2f px, snaps %d, backward %d, final diff %.3f", c.name, r.smoothMaxErr, r.snaps, r.backwardSteady, r.finalDiff)
		if r.smoothMaxErr > c.maxSmooth || r.snaps > 0 || r.backwardSteady > 0 || r.finalDiff > 1 {
			t.Errorf("%s: steady slow-mo prediction drifts: %.2f px, %d snaps, %d backward", c.name, r.smoothMaxErr, r.snaps, r.backwardSteady)
		}
	}
}
