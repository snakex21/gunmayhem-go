package game

import (
	"math"
	"testing"
	"time"
)

func TestSimClockRunsHalfRateForFiveSeconds(t *testing.T) {
	c := simClock{left: slowMoRealTicks}
	steps, ticks := 0, 0
	for c.active() {
		n, next := c.advance()
		c = next
		steps += n
		ticks++
	}
	if ticks != slowMoRealTicks || steps != slowMoRealTicks/2 {
		t.Fatalf("slow motion: %d real ticks, %d sim steps", ticks, steps)
	}
	if n, next := c.advance(); n != 1 || next != (simClock{}) {
		t.Fatalf("after slow motion: steps=%d clock=%+v", n, next)
	}
	// A half step pending at the end is completed on the next tick.
	c = simClock{left: 1}
	n, c := c.advance()
	if n != 0 || c.left != 0 || c.acc != 0.5 {
		t.Fatalf("last slowed tick: steps=%d %+v", n, c)
	}
	if n, c = c.advance(); n != 1 || c != (simClock{}) {
		t.Fatalf("pending half step: steps=%d %+v", n, c)
	}
}

func TestSimClockRenderPhaseIsContinuous(t *testing.T) {
	// alpha at the end of one real tick (f = 1 before clamping) must equal
	// alpha at the start of the next one, or 1 when the next tick steps.
	c := simClock{left: 7}
	for i := 0; i < 12; i++ {
		phase, inc := c.renderPhase()
		end := phase + inc
		steps, next := c.advance()
		nphase, _ := next.renderPhase()
		if steps == 1 {
			if math.Abs(end-1) > 1e-12 || nphase != 0 {
				t.Fatalf("tick %d: alpha ends at %.2f, next phase %.2f (clock %+v)", i, end, nphase, c)
			}
		} else if math.Abs(end-nphase) > 1e-12 {
			t.Fatalf("tick %d: alpha ends at %.2f, next starts at %.2f", i, end, nphase)
		}
		c = next
	}
	var r renderInterpState
	r.simPhase, r.simInc = 0.5, 0.5
	if a := r.renderSimAlpha(0.5); a != 0.75 {
		t.Fatalf("slowed alpha %.3f, want 0.75", a)
	}
	r = renderInterpState{}
	if a := r.renderSimAlpha(0.3); a != 0.3 {
		t.Fatalf("normal alpha %.3f", a)
	}
}

func TestSimClockLastStepClock(t *testing.T) {
	// The host steps, skips, steps... lastStepClock from the clock after any
	// real tick is the clock after the most recent step.
	c := simClock{left: 9}
	last := c
	for i := 0; i < 14; i++ {
		steps, next := c.advance()
		c = next
		if steps > 0 {
			last = c
		}
		if got := c.lastStepClock(); got != last {
			t.Fatalf("tick %d: lastStepClock %+v want %+v (clock %+v)", i, got, last, c)
		}
	}
}

// slowMoTwinHosts returns two identical deterministic matches.
func slowMoTwinHosts() (*Game, *Game) {
	a, b := newPredSimHost(), newPredSimHost()
	for i := range a.players {
		cp := *a.players[i]
		b.players[i] = &cp
	}
	return a, b
}

func TestSlowMoRunsTheSimulationAtHalfRate(t *testing.T) {
	slow, normal := slowMoTwinHosts()
	// Knockback in flight and a bullet that will hit player 2.
	slow.players[0].VX, slow.players[0].VY = 13, -9
	normal.players[0].VX, normal.players[0].VY = 13, -9
	b := testBullet(slow.players[1].X-120, slow.players[1].Y-35, 1, 3)
	slow.bullets = []Bullet{b}
	normal.bullets = []Bullet{b}

	script := func(step int) PlayerInputState {
		return PlayerInputState{Right: step < 25, Left: step >= 25, Up: step == 6 || step == 9}
	}
	const steps = 40
	slow.startSlowMo()
	for k := 0; k < 2*steps; k++ {
		slow.players[1].RemoteInput = script(k / 2)
		slow.players[0].RemoteInput = PlayerInputState{}
		if err := slow.Update(); err != nil {
			t.Fatal(err)
		}
	}
	for k := 0; k < steps; k++ {
		normal.players[1].RemoteInput = script(k)
		normal.players[0].RemoteInput = PlayerInputState{}
		if err := normal.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if slow.slowMo.left != slowMoRealTicks-2*steps || slow.slowMo.acc != 0 {
		t.Fatalf("clock %+v", slow.slowMo)
	}
	for i := range slow.players {
		s, n := slow.players[i], normal.players[i]
		if s.X != n.X || s.Y != n.Y || s.VX != n.VX || s.VY != n.VY || s.JumpNum != n.JumpNum || s.HitNumber != n.HitNumber {
			t.Fatalf("player %d after %d real ticks: (%.4f,%.4f v %.4f,%.4f jump %d hits %d), after %d steps: (%.4f,%.4f v %.4f,%.4f jump %d hits %d)",
				s.ID, 2*steps, s.X, s.Y, s.VX, s.VY, s.JumpNum, s.HitNumber, steps, n.X, n.Y, n.VX, n.VY, n.JumpNum, n.HitNumber)
		}
	}
	if normal.players[1].HitNumber == 0 {
		t.Fatal("the bullet did not hit: knockback not covered")
	}
	if len(slow.bullets) != len(normal.bullets) {
		t.Fatalf("bullets %d vs %d", len(slow.bullets), len(normal.bullets))
	}
}

func TestSlowMoKeepsTapsOfSkippedTicks(t *testing.T) {
	g := newPredSimHost()
	p := g.players[1]
	updateN(t, g, 3) // settle on the floor
	jumps := p.JumpNum
	g.startSlowMo()
	p.RemoteInput = PlayerInputState{Up: true}
	if err := g.Update(); err != nil { // skipped tick: the tap starts and ends here
		t.Fatal(err)
	}
	if !g.slowMoSkipped || p.VY < 0 {
		t.Fatalf("first slowed tick must skip (skipped=%v vy=%.2f)", g.slowMoSkipped, p.VY)
	}
	p.RemoteInput = PlayerInputState{}
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	if g.slowMoSkipped || p.VY >= 0 || p.JumpNum != jumps-1 {
		t.Fatalf("jump tapped on a skipped tick was lost: vy=%.2f jumps=%d", p.VY, p.JumpNum)
	}
	if p.slowMoLatch != 0 {
		t.Fatal("latch not cleared after the sim step")
	}

	// A shot tapped on a skipped tick fires on the next step, like a tap at
	// normal speed.
	shots := func(slow bool) int {
		g2 := newPredSimHost()
		q := g2.players[1]
		updateN(t, g2, 20)
		before := q.ShotsFired
		if slow {
			g2.startSlowMo()
		}
		q.RemoteInput = PlayerInputState{Shoot: true}
		_ = g2.Update()
		q.RemoteInput = PlayerInputState{}
		_ = g2.Update()
		return q.ShotsFired - before
	}
	if normal, slow := shots(false), shots(true); normal != 1 || slow != 1 {
		t.Fatalf("shot tap: %d at normal speed, %d tapped on a skipped tick", normal, slow)
	}
}

func slowMoPowerupAt(g *Game, p *Player) {
	g.powerups = append(g.powerups, Powerup{X: p.X, Y: p.Y - 30, Type: powerupSlowMo, Frame: 30, FlashFrame: 0, FlashPlaying: true, Serial: 99})
}

func TestSlowMoPowerupDurationAndRefresh(t *testing.T) {
	g := newPredSimHost()
	p := g.players[0]
	slowMoPowerupAt(g, p)
	updateN(t, g, 1)
	if g.slowMo.left != slowMoRealTicks || p.PowerupsCollected != 1 {
		t.Fatalf("pickup: clock %+v collected %d", g.slowMo, p.PowerupsCollected)
	}
	if len(g.powerupNameFX) == 0 || g.powerupNameFX[len(g.powerupNameFX)-1].Kind != powerupSlowMo {
		t.Fatal("no SLOW-MO name effect")
	}
	updateN(t, g, 75)
	if g.slowMo.left != slowMoRealTicks-75 {
		t.Fatalf("after 75 ticks: %+v", g.slowMo)
	}
	// A second pickup while active refreshes the 5 s, never stacks below half speed.
	for g.slowMo.acc != 0 {
		updateN(t, g, 1)
	}
	slowMoPowerupAt(g, p)
	updateN(t, g, 2)
	if g.slowMo.left < slowMoRealTicks-2 || g.slowMo.left > slowMoRealTicks {
		t.Fatalf("refresh: %+v", g.slowMo)
	}
	steps := 0
	for i := 0; i < 2*slowMoRealTicks && g.slowMo.active(); i++ {
		_ = g.Update()
		if !g.slowMoSkipped {
			steps++
		}
	}
	if g.slowMo.active() || steps < slowMoRealTicks/2-2 || steps > slowMoRealTicks/2+1 {
		t.Fatalf("refreshed run: clock %+v, %d steps", g.slowMo, steps)
	}
	if powerupName(powerupSlowMo) != "SLOW-MO" || arsenal2PowerupIcon(powerupSlowMo) == nil {
		t.Fatal("SLOW-MO name/icon")
	}
	// Leaving the round resets it.
	g.startSlowMo()
	g.resetArenaState()
	if g.slowMo != (simClock{}) {
		t.Fatal("round reset kept slow motion")
	}
}

func TestSlowMoInExtendedPowerupRollOnly(t *testing.T) {
	arena := OriginalMap1()
	seen := false
	for i := 0; i < 3000; i++ {
		if pu := spawnPowerup(arena, false); pu.Type >= powerupAkimbo {
			t.Fatalf("source roll produced kind %d", pu.Type)
		}
		if pu := spawnPowerup(arena, false, true); pu.Type == powerupSlowMo {
			seen = true
		}
	}
	if !seen {
		t.Fatal("extended roll never produced SLOW-MO")
	}
}

func TestChaosSlowMoEveryone(t *testing.T) {
	g := newPredSimHost()
	label := ""
	for _, spec := range chaosEffectSpecs {
		if spec.ID == chaosEffectSlowMo {
			label = spec.Label
		}
	}
	if label != "SLOW-MO EVERYONE" || chaosEffectLabel(chaosEffectSlowMo) != label {
		t.Fatalf("chaos spec %q", label)
	}
	g.initCustomChaosDefaults()
	if !g.customChaosSelected[chaosEffectSlowMo] {
		t.Fatal("not in the default chaos pool")
	}
	g.activateCustomChaosHit(menuHitChaosEffectBase + chaosEffectSlowMo)
	if g.customChaosSelected[chaosEffectSlowMo] {
		t.Fatal("chaos toggle did not deselect SLOW-MO EVERYONE")
	}
	g.activateCustomChaosHit(menuHitChaosEffectBase + chaosEffectSlowMo)
	g.activateChaosEffect(chaosEffectSlowMo)
	if g.slowMo.left != slowMoRealTicks {
		t.Fatalf("chaos event clock %+v", g.slowMo)
	}
	if err := validateNetSnapshot(netSnapshot{ChaosCurrentEffect: chaosEffectSlowMo, SlowMoTicks: g.slowMo.left}); err != nil {
		t.Fatal(err)
	}
}

func TestSlowMoRenderKeepsTheSimStepPair(t *testing.T) {
	g := newRenderInterpTestGame()
	for _, p := range g.players {
		p.AI, p.RemoteControlled = false, true
		p.RemoteInput = PlayerInputState{}
	}
	g.fadeActive = false
	p := g.players[0]
	p.X, p.Y, p.VX, p.VY = 300, 100, 6, 0 // in the air: moves every step
	g.startSlowMo()
	t0 := time.Unix(2000, 0)
	tick := time.Second / TicksPerSecond
	// Tick 1 (skipped), tick 2 (step), tick 3 (skipped).
	var xs []float64
	for i := 0; i < 3; i++ {
		xs = append(xs, p.X)
		g.captureRenderPrev(t0.Add(time.Duration(i) * tick))
		g.slowMoSkipped = false
		g.renderInterp.simPhase, g.renderInterp.simInc = 0, 1
		g.processZombieRequests()
		if g.beginSimTick() {
			g.simulateArenaTick(true)
			g.clearSlowMoLatches()
		}
	}
	x1, x2 := xs[1], xs[2] // the step of tick 2 went from x1 to x2
	if x1 == x2 || xs[0] != xs[1] || p.X != x2 {
		t.Fatalf("expected a step between ticks 2 and 3: %v -> %.2f", xs, p.X)
	}
	// After the skipped tick 3: draw from x1 to x2 with alpha 0.5 + f/2.
	restore := g.beginRenderInterpolation(t0.Add(2*tick + tick/2))
	got := g.players[0].X
	restore()
	want := x1 + (x2-x1)*0.75
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("drawn at %.3f, want %.3f (pair %.3f -> %.3f)", got, want, x1, x2)
	}
	// The predicted own player of a guest keeps the plain tick alpha.
	g.renderInterp.freeID = p.ID
	restore = g.beginRenderInterpolation(t0.Add(2*tick + tick/2))
	got = g.players[0].X
	restore()
	if math.Abs(got-(x1+(x2-x1)*0.5)) > 1e-6 {
		t.Fatalf("free player drawn at %.3f", got)
	}
}

func TestSlowMoSnapshotRoundTripAndValidation(t *testing.T) {
	g := newPredSimHost()
	g.startSlowMo()
	_ = g.Update()
	snap := g.makeNetSnapshot()
	if snap.SlowMoTicks != slowMoRealTicks-1 || snap.SlowMoAcc != 0.5 {
		t.Fatalf("snapshot clock %d %.2f", snap.SlowMoTicks, snap.SlowMoAcc)
	}
	prepareNetSnapshotForWire(&snap)
	var enc netHostEncoder
	frame := enc.encode(snap)
	payload, rawLen, err := newNetPeerStream().message(frame, 2)
	if err != nil {
		t.Fatal(err)
	}
	got, err := newNetClientStream().decode(payload, rawLen)
	if err != nil {
		t.Fatal(err)
	}
	if got.state.SlowMoTicks != snap.SlowMoTicks || got.state.SlowMoAcc != snap.SlowMoAcc {
		t.Fatalf("decoded clock %d %.2f", got.state.SlowMoTicks, got.state.SlowMoAcc)
	}
	if err := validateNetSnapshot(got.state); err != nil {
		t.Fatal(err)
	}
	bad := []netSnapshot{
		{SlowMoTicks: -1},
		{SlowMoTicks: slowMoRealTicks + 1},
		{SlowMoAcc: 1},
		{SlowMoAcc: -0.5},
		{SlowMoAcc: math.NaN()},
		{SlowMoAcc: math.Inf(1)},
		{Powerups: []Powerup{{Type: powerupKindCount}}},
	}
	for i, s := range bad {
		if validateNetSnapshot(s) == nil {
			t.Fatalf("bad snapshot %d accepted", i)
		}
	}
	if validateNetSnapshot(netSnapshot{Powerups: []Powerup{{Type: powerupSlowMo}}, PowerupNameFX: []PowerupNameEffect{{Kind: powerupSlowMo}}}) != nil {
		t.Fatal("SLOW-MO powerup rejected")
	}
	// The guest adopts the clock.
	c := New()
	c.applyNetSnapshot(got.state)
	if c.slowMo.left != snap.SlowMoTicks || c.slowMo.acc != 0.5 {
		t.Fatalf("guest clock %+v", c.slowMo)
	}
}

func TestSlowMoSFXRateIsSanitized(t *testing.T) {
	for _, c := range []struct{ in, want float64 }{
		{0, 1}, {1, 1}, {2, 1}, {-1, 1}, {math.NaN(), 1}, {0.1, 1}, {slowMoSFXRate, slowMoSFXRate}, {slowMoCueRate, slowMoCueRate},
	} {
		if got := sanitizeSFXRate(c.in); got != c.want {
			t.Fatalf("rate %v -> %v want %v", c.in, got, c.want)
		}
	}
	pcm := make([]byte, 400) // 100 stereo frames
	for i := 0; i < 100; i++ {
		v := uint16(int16(i * 100))
		pcm[i*4], pcm[i*4+1] = byte(v), byte(v>>8)
		pcm[i*4+2], pcm[i*4+3] = byte(v), byte(v>>8)
	}
	out := resamplePCM16Stereo(pcm, 0.5)
	if len(out) != 800 {
		t.Fatalf("half speed length %d", len(out))
	}
	mid := int16(uint16(out[4*3]) | uint16(out[4*3+1])<<8) // frame 3 = source 1.5
	if mid != 150 {
		t.Fatalf("interpolated sample %d", mid)
	}
}

func TestSlowMoHostHoldsInputAckOnSkippedTicks(t *testing.T) {
	peer := &netplayPeer{playerID: 2, stateSend: make(chan *netHostFrame, 1)}
	s := &netplaySession{mode: netplayHost, peers: map[int]*netplayPeer{2: peer}}
	peer.input.receive(3, []uint8{1, 1, 1}, time.Now())
	ack := func() uint32 {
		s.queueState(netSnapshot{})
		f := <-peer.stateSend
		return f.inputAck[2]
	}
	peer.input.take(time.Now())
	if a := ack(); a != 1 {
		t.Fatalf("ack %d", a)
	}
	peer.input.take(time.Now()) // consumed on a skipped tick
	s.inputAckHold = true
	if a := ack(); a != 1 {
		t.Fatalf("held ack %d", a)
	}
	peer.input.take(time.Now())
	s.inputAckHold = false
	if a := ack(); a != 3 {
		t.Fatalf("ack after the step %d", a)
	}
}

func TestSlowMoZombieAndCampaignTickAtHalfRate(t *testing.T) {
	// Zombie intro countdown: one per sim step.
	z := newZombieTestGame(t, 1, [4]int{1, 2, 0, 0})
	z.zombie.hud.Phase = zombiePhaseIntro
	z.zombie.hud.PhaseTimer = 1000
	z.startSlowMo()
	updateN(t, z, 40)
	if got := 1000 - z.zombie.hud.PhaseTimer; got != 20 {
		t.Fatalf("zombie intro advanced %d sim steps in 40 slowed ticks", got)
	}
	holdZombieWave(z)
	zb := spawnTestZombie(z, zombieWalker, 300, 511, 5000)
	startX := zb.X
	for i := 0; i < 120; i++ {
		placePlayer(z.players[0], 700, 511)
		z.startSlowMo()
		updateN(t, z, 1)
	}
	if zb.X <= startX+5 {
		t.Fatalf("zombie did not walk under slow motion: %.1f -> %.1f", startX, zb.X)
	}

	// Campaign enemies (mech mission, bots): one enemy frame per sim step.
	c := startExtraMissionForTest(t, 26)
	c.assets = LoadAssets()
	c.screen = screenGameplay
	c.fadeActive = false
	updateN(t, c, 30)
	before := c.campaignEnemyFrame
	c.startSlowMo()
	updateN(t, c, 60)
	if got := c.campaignEnemyFrame - before; got != 30 {
		t.Fatalf("campaign enemies advanced %d frames in 60 slowed ticks", got)
	}
	for i := 0; i < 600; i++ {
		c.startSlowMo()
		if err := c.Update(); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range c.players {
		if math.IsNaN(p.X) || math.IsNaN(p.Y) {
			t.Fatal("NaN position under slow motion")
		}
	}
}
