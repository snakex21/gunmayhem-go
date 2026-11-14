package game

import (
	"math"
	"math/rand"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Codec

// TestNetCodecSupportsSnapshotSchema fails with the field path when somebody
// adds a snapshot field the compact codec cannot carry (map, pointer, ...).
// Fix: use a plain value/slice/array/struct, or skip it in netWireSkipField.
func TestNetCodecSupportsSnapshotSchema(t *testing.T) {
	if fp := netSchemaFingerprint(); len(fp) != 8 {
		t.Fatalf("fingerprint=%q", fp)
	}
	if v := netplayGameVersion(); !strings.Contains(v, netSchemaFingerprint()) {
		t.Fatalf("game version %q lacks fingerprint", v)
	}
}

// fillNetTestValue sets every field reachable from v to a non-zero value.
func fillNetTestValue(v reflect.Value, seed *int) {
	*seed++
	n := *seed
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(int64(n%100 + 1))
		if n%3 == 0 {
			v.SetInt(-int64(n%50 + 1))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(uint64(n%200 + 1))
	case reflect.Float32, reflect.Float64:
		v.SetFloat(float64(n)*1.37 + 0.123456789)
	case reflect.String:
		v.SetString("s" + strings.Repeat("x", n%5))
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 2, 2)
		for i := 0; i < 2; i++ {
			fillNetTestValue(s.Index(i), seed)
		}
		v.Set(s)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			fillNetTestValue(v.Index(i), seed)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).PkgPath == "" {
				fillNetTestValue(v.Field(i), seed)
			}
		}
	}
}

// expectNetWire applies what the wire does: skipped fields become zero and
// floats are carried as float32.
func expectNetWire(v reflect.Value) {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		v.SetFloat(float64(float32(v.Float())))
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			expectNetWire(v.Index(i))
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if netWireSkipField(v.Type(), f) {
				if f.PkgPath == "" {
					v.Field(i).Set(reflect.Zero(f.Type))
				}
				continue
			}
			expectNetWire(v.Field(i))
		}
	}
}

// Every snapshot field, including fields other agents add to Player or effect
// structs later, must survive the compact codec.
func TestNetCodecRoundTripEveryField(t *testing.T) {
	var want netSnapshot
	seed := 0
	fillNetTestValue(reflect.ValueOf(&want).Elem(), &seed)
	data := netEncodeValue(nil, &want)
	var got netSnapshot
	n, err := netDecodeValue(data, &got)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(data) {
		t.Fatalf("decoded %d of %d bytes", n, len(data))
	}
	expectNetWire(reflect.ValueOf(&want).Elem())
	if !reflect.DeepEqual(got, want) {
		gv, wv := reflect.ValueOf(got), reflect.ValueOf(want)
		for i := 0; i < gv.NumField(); i++ {
			if !reflect.DeepEqual(gv.Field(i).Interface(), wv.Field(i).Interface()) {
				t.Errorf("field %s differs after round trip", gv.Type().Field(i).Name)
			}
		}
		t.FailNow()
	}
	if len(got.Players) == 0 || got.Players[0].PendingSounds != nil || got.Players[0].AITargetX != 0 || got.Players[0].Controls != (Controls{}) {
		t.Fatal("host-only player fields reached the wire")
	}
	if !got.Players[0].AI || got.Players[0].AISpecial == 0 || !got.Players[0].RemoteControlled {
		t.Fatal("render-relevant player fields were dropped")
	}
}

func TestNetCodecRejectsHostileLengths(t *testing.T) {
	var s netSnapshot
	// Struct bitmap with "Players" present, then a huge slice length.
	data := netEncodeValue(nil, &netSnapshot{Players: []Player{{ID: 1}}})
	for cut := 0; cut < len(data); cut++ {
		if _, err := netDecodeValue(data[:cut], &s); err == nil {
			t.Fatalf("truncated input of %d/%d bytes decoded without error", cut, len(data))
		}
	}
}

// ---------------------------------------------------------------------------
// Stream: size budget and correctness

func netStepTestScene(s *netSnapshot, tick int) {
	for i := range s.Players {
		p := &s.Players[i]
		p.X += p.VX
		p.Y += p.VY
		p.HeadFrame = (p.HeadFrame + 1) % 50
		p.VisualBodyY = -60 + float64(tick%7)*0.37
		p.VisualGunRotation = float64(tick) * 1.7
		p.LegFrame1 = tick % 20
		p.Weapon.Frame = tick % 9
		p.AITargetX = rand.Float64() * 900 // host-only noise
	}
	for i := range s.Bullets {
		s.Bullets[i].X += s.Bullets[i].VX
		s.Bullets[i].Y += s.Bullets[i].VY
		s.Bullets[i].Time++
	}
	for i := range s.Shells {
		s.Shells[i].X += s.Shells[i].VX
		s.Shells[i].Y += s.Shells[i].VY
		s.Shells[i].Rotation += s.Shells[i].RotSpeed
	}
	for i := range s.BlastFX {
		s.BlastFX[i].X += s.BlastFX[i].VX
		s.BlastFX[i].Y += s.BlastFX[i].VY
		s.BlastFX[i].Alpha *= 0.97
	}
	for i := range s.JetThrustFX {
		s.JetThrustFX[i].X += s.JetThrustFX[i].VX
		s.JetThrustFX[i].Y += s.JetThrustFX[i].VY
	}
	s.CameraX = float64(tick) * 0.31
	s.MapFXFrame = tick
}

func quietNetTestScene() netSnapshot {
	s := realisticNetSnapshot()
	s.Bullets, s.Shells, s.BlastFX, s.FirePatches, s.JetThrustFX = nil, nil, nil, nil, nil
	s.Flashes, s.Grenades, s.Explosions, s.PlayerTrails, s.InstagibTrails, s.DropPackFX = nil, nil, nil, nil, nil, nil
	return s
}

type netStreamStats struct {
	first, avg, max int
}

func runNetStream(t *testing.T, scene netSnapshot, ticks int) netStreamStats {
	t.Helper()
	var enc netHostEncoder
	peer := newNetPeerStream()
	client := newNetClientStream()
	var st netStreamStats
	total := 0
	for tick := 0; tick < ticks; tick++ {
		netStepTestScene(&scene, tick)
		copyState := scene
		copyState.Players = append([]Player(nil), scene.Players...)
		frame := enc.encode(copyState)
		payload, rawLen, err := peer.message(frame)
		if err != nil {
			t.Fatal(err)
		}
		msg := netWireMessage{Type: netMessageState, Protocol: netplayProtocolVersion, PlayerID: 2, Tick: frame.tick, Payload: payload, PayloadLen: rawLen}
		wire, err := encodeNetMessage(msg)
		if err != nil {
			t.Fatal(err)
		}
		size := len(wire) + 4
		if tick == 0 {
			st.first = size
		} else {
			total += size
			st.max = max(st.max, size)
		}
		decoded, err := client.decode(payload, rawLen)
		if err != nil {
			t.Fatalf("tick %d: %v", tick, err)
		}
		if decoded.tick != frame.tick || len(decoded.state.Players) != len(scene.Players) {
			t.Fatalf("tick %d: decoded tick=%d players=%d", tick, decoded.tick, len(decoded.state.Players))
		}
		for i, p := range decoded.state.Players {
			if math.Abs(p.X-scene.Players[i].X) > 0.01 || p.Weapon.Def != scene.Players[i].Weapon.Def || p.Name != scene.Players[i].Name {
				t.Fatalf("tick %d player %d mismatch: got X=%v def=%+v want X=%v def=%+v", tick, i, p.X, p.Weapon.Def, scene.Players[i].X, scene.Players[i].Weapon.Def)
			}
		}
		if decoded.state.Arena.Number != scene.Arena.Number || len(decoded.state.Arena.Platforms) != len(scene.Arena.Platforms) {
			t.Fatalf("tick %d: arena not restored", tick)
		}
		if len(decoded.state.Bullets) != len(scene.Bullets) {
			t.Fatalf("tick %d: bullets=%d want %d", tick, len(decoded.state.Bullets), len(scene.Bullets))
		}
	}
	st.avg = total / (ticks - 1)
	return st
}

// Size budget for one snapshot on the wire (CBOR envelope + 4-byte frame).
// Protocol 18 measured 11.6 KB quiet / 42.5 KB busy per snapshot.
func TestNetSnapshotStreamSizeBudget(t *testing.T) {
	rand.Seed(7)
	oldQuiet := measureLegacyCBORSnapshot(t, quietNetTestScene())
	oldBusy := measureLegacyCBORSnapshot(t, realisticNetSnapshot())
	quiet := runNetStream(t, quietNetTestScene(), 90)
	busy := runNetStream(t, realisticNetSnapshot(), 90)
	t.Logf("protocol 18 CBOR: quiet=%dB busy=%dB per snapshot", oldQuiet, oldBusy)
	t.Logf("protocol 19 stream: quiet first=%dB avg=%dB max=%dB | busy first=%dB avg=%dB max=%dB", quiet.first, quiet.avg, quiet.max, busy.first, busy.avg, busy.max)
	t.Logf("at 30 Hz: quiet %.1f kbit/s (was %.1f), busy %.1f kbit/s (was %.1f)",
		float64(quiet.avg*30*8)/1000, float64(oldQuiet*30*8)/1000, float64(busy.avg*30*8)/1000, float64(oldBusy*30*8)/1000)
	if quiet.avg > 1024 {
		t.Fatalf("quiet snapshot avg=%dB, budget 1 KiB", quiet.avg)
	}
	if busy.avg > 4*1024 {
		t.Fatalf("busy snapshot avg=%dB, budget 4 KiB", busy.avg)
	}
	if busy.avg*5 > oldBusy || quiet.avg*5 > oldQuiet {
		t.Fatalf("less than 5x reduction: quiet %d->%d busy %d->%d", oldQuiet, quiet.avg, oldBusy, busy.avg)
	}
}

// measureLegacyCBORSnapshot is the protocol 18 cost: the whole netSnapshot as
// CBOR with string keys (including the arena and full weapon definitions).
func measureLegacyCBORSnapshot(t *testing.T, s netSnapshot) int {
	t.Helper()
	type legacy struct {
		Type, Protocol, PlayerID int
		State                    netSnapshot
	}
	b, err := netCBOREnc.Marshal(legacy{Type: netMessageState, Protocol: 18, PlayerID: 2, State: s})
	if err != nil {
		t.Fatal(err)
	}
	return len(b) + 4
}

func TestNetSnapshotEffectStormIsTrimmedNotRejected(t *testing.T) {
	var enc netHostEncoder
	state := netSnapshot{Screen: int(screenGameplay)}
	state.BlastFX = make([]BlastEffect, 900)
	state.DynamiteFX = make([]DynamiteEffect, 700)
	for i := range state.BlastFX {
		state.BlastFX[i].X = float64(i)
	}
	frame := enc.encode(state)
	payload, rawLen, err := newNetPeerStream().message(frame)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := newNetClientStream().decode(payload, rawLen)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateNetSnapshot(decoded.state); err != nil {
		t.Fatalf("trimmed snapshot rejected: %v", err)
	}
	if n := len(decoded.state.BlastFX); n == 0 || n > 512 {
		t.Fatalf("blast fx=%d", n)
	}
	if last := decoded.state.BlastFX[len(decoded.state.BlastFX)-1].X; last != 899 {
		t.Fatalf("trim kept old effects, newest X=%v", last)
	}
	// A client-side oversized list (e.g. hostile or future host) is trimmed too.
	big := netSnapshot{BlastFX: make([]BlastEffect, 2000)}
	if cut := trimNetSnapshotLists(&big); len(cut) == 0 || validateNetSnapshot(big) != nil {
		t.Fatalf("client trim failed: %v", cut)
	}
}

func TestNetSFXSurvivesDroppedSnapshotsAndIsNotReplayed(t *testing.T) {
	var enc netHostEncoder
	peer := newNetPeerStream()
	client := newNetClientStream()
	// Sound before the peer's first frame: must not be replayed on join.
	enc.encode(netSnapshot{SFXEvents: []netSFXEvent{{Seq: 1, Name: "old.wav"}}})
	first := enc.encode(netSnapshot{})
	payload, n, _ := peer.message(first)
	d, err := client.decode(payload, n)
	if err != nil || len(d.sfx) != 0 {
		t.Fatalf("join replayed old sounds: %+v err=%v", d.sfx, err)
	}
	// Two frames with sounds are dropped for this peer (backpressure)...
	enc.encode(netSnapshot{SFXEvents: []netSFXEvent{{Seq: 2, Name: "a.wav"}}})
	enc.encode(netSnapshot{SFXEvents: []netSFXEvent{{Seq: 3, Name: "b.wav"}}})
	latest := enc.encode(netSnapshot{})
	payload, n, _ = peer.message(latest)
	d, err = client.decode(payload, n)
	if err != nil || len(d.sfx) != 2 || d.sfx[0].Seq != 2 || d.sfx[1].Seq != 3 {
		t.Fatalf("sounds lost with dropped snapshots: %+v err=%v", d.sfx, err)
	}
}

// ---------------------------------------------------------------------------
// Interpolation and playout

func TestInterpolateNetSnapshotPlayers(t *testing.T) {
	a := netSnapshot{CameraX: 0, Players: []Player{{ID: 1, X: 100, Y: 200}, {ID: 2, X: 0, Y: 0}}}
	b := netSnapshot{CameraX: 10, Players: []Player{{ID: 1, X: 120, Y: 180}, {ID: 2, X: 500, Y: 0}}}
	got := interpolateNetSnapshot(a, b, 0.25)
	if got.Players[0].X != 105 || got.Players[0].Y != 195 || got.CameraX != 2.5 {
		t.Fatalf("lerp wrong: %+v camera=%v", got.Players[0], got.CameraX)
	}
	if got.Players[1].X != 0 {
		t.Fatalf("teleport was interpolated: X=%v", got.Players[1].X)
	}
	if a.Players[0].X != 100 {
		t.Fatal("interpolation mutated the source snapshot")
	}
}

func TestNetPlayoutSteadyAndGaps(t *testing.T) {
	var p netPlayout
	var shown []int
	pos := func(s netSnapshot) int { return int(s.Players[0].X) }
	frame := func(tick int) netSnapshot { return netSnapshot{Players: []Player{{ID: 1, X: float64(tick * 10)}}} }
	// Steady 1:1 arrival.
	for tick := 1; tick <= 20; tick++ {
		p.push(uint32(tick), frame(tick))
		s, ok := p.next()
		if !ok {
			t.Fatal("no state")
		}
		shown = append(shown, pos(s))
	}
	for i := 2; i < len(shown); i++ {
		if shown[i]-shown[i-1] != 10 {
			t.Fatalf("steady playout stuttered: %v", shown)
		}
	}
	// The host skipped ticks for this client (every second tick arrives).
	shown = shown[:0]
	for tick := 22; tick <= 60; tick += 2 {
		p.push(uint32(tick), frame(tick))
		s, _ := p.next()
		shown = append(shown, pos(s))
		s, _ = p.next()
		shown = append(shown, pos(s))
	}
	smooth := 0
	for i := 4; i < len(shown); i++ {
		if shown[i]-shown[i-1] == 10 {
			smooth++
		}
	}
	if smooth < len(shown)-8 {
		t.Fatalf("half-rate playout not interpolated: %v", shown)
	}
}

func TestNetPlayoutAdaptsToBurstyArrival(t *testing.T) {
	var p netPlayout
	tick := 0
	holds := 0
	last := -1
	for step := 0; step < 300; step++ {
		// Snapshots arrive in pairs every second client tick.
		if step%2 == 0 {
			tick++
			p.push(uint32(tick), netSnapshot{MapFXFrame: tick})
			tick++
			p.push(uint32(tick), netSnapshot{MapFXFrame: tick})
		}
		s, _ := p.next()
		if step > 120 && s.MapFXFrame == last {
			holds++
		}
		last = s.MapFXFrame
	}
	if holds > 0 {
		t.Fatalf("playout still repeats frames after adapting: holds=%d delay=%d", holds, p.delay)
	}
}

// ---------------------------------------------------------------------------
// Input replay

func TestNetInputReplayAppliesEachFrameOnce(t *testing.T) {
	var h netInputHistory
	var r netInputReplay
	now := time.Now()
	shoot := PlayerInputState{Shoot: true}
	// Tap, release, tap, release inside one host tick; only the last message
	// arrives (the earlier ones were replaced in the latest-value outbox).
	h.record(shoot)
	h.record(PlayerInputState{})
	h.record(shoot)
	seq, frames := h.record(PlayerInputState{})
	r.receive(seq, frames, now)
	r.receive(seq, frames, now) // duplicate delivery must not re-apply
	var got []bool
	for i := 0; i < 6; i++ {
		got = append(got, unpackNetInput(r.take(now)).Shoot)
	}
	want := []bool{true, false, true, false, false, false}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("replayed shoot=%v want %v", got, want)
	}
}

func TestNetInputReplayKeepsGrenadeChargeLength(t *testing.T) {
	var h netInputHistory
	var r netInputReplay
	now := time.Now()
	held := 0
	// The client holds the throw key for seven frames. Its messages reach the
	// host in jittery bursts (three frames at once, then nothing), while the
	// host simulates one tick per client frame.
	inputs := []PlayerInputState{{}, {Grenade: true}, {Grenade: true}, {Grenade: true}, {Grenade: true}, {Grenade: true}, {Grenade: true}, {Grenade: true}, {}, {}, {}, {}}
	for i, in := range inputs {
		seq, frames := h.record(in)
		if i%3 == 2 {
			r.receive(seq, frames, now)
		}
		if unpackNetInput(r.take(now)).Grenade {
			held++
		}
	}
	for i := 0; i < 6; i++ {
		if unpackNetInput(r.take(now)).Grenade {
			held++
		}
	}
	if held != 7 {
		t.Fatalf("grenade held for %d host ticks, client held 7", held)
	}
}

func TestNetInputReplayReleasesSilentPeer(t *testing.T) {
	var r netInputReplay
	now := time.Now()
	r.receive(1, []uint8{packNetInput(PlayerInputState{Right: true})}, now)
	if !unpackNetInput(r.take(now)).Right {
		t.Fatal("first frame not applied")
	}
	if !unpackNetInput(r.take(now.Add(100 * time.Millisecond))).Right {
		t.Fatal("short gap should hold the last input")
	}
	if unpackNetInput(r.take(now.Add(400 * time.Millisecond))).Right {
		t.Fatal("silent peer kept a key held")
	}
}

// A semi-auto weapon must fire twice when the client taps twice within one
// snapshot interval (both taps reach the host in one message).
func TestOnlineSemiAutoDoubleTapFiresTwice(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	client, err := newClientNetplay(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	waitFor(t, "client player assignment", func() bool { return client.assignedPlayer() == 2 })
	waitFor(t, "host peer", host.connected)

	for _, in := range []PlayerInputState{{Shoot: true}, {}, {Shoot: true}, {}} {
		client.queueInput(in)
	}
	waitFor(t, "input frames", func() bool {
		host.peersMu.RLock()
		peer := host.peers[2]
		host.peersMu.RUnlock()
		if peer == nil {
			return false
		}
		peer.inputMu.Lock()
		defer peer.inputMu.Unlock()
		return peer.input.lastSeq == 4
	})

	p := &Player{ID: 2, Active: true, RemoteControlled: true, Facing: 1, MiniMulti: 1, PlayerScale: .8}
	p.Weapon = NewWeapon(98) // developed semi-auto pistol
	if spec, ok := developedWeaponSpec(98); !ok || !developedTriggerResetRequired(spec) {
		t.Skip("weapon 98 is no longer semi-auto")
	}
	p.Weapon.Def.ROF = 1 // isolate the trigger latch from the fire-rate cooldown
	p.Weapon.Frame = 1
	p.Weapon.WaitTime = p.Weapon.Def.ROF
	var bullets []Bullet
	var shells []Shell
	var flashes []MuzzleFlash
	shots := 0
	for tick := 0; tick < 6; tick++ {
		p.RemoteInput = host.takeInputFor(2)
		before := p.ShotsFired
		updateWeapon(p, &bullets, &shells, &flashes)
		if p.ShotsFired > before {
			shots++
		}
	}
	if shots != 2 {
		t.Fatalf("double tap fired %d shots online, want 2", shots)
	}
}

// ---------------------------------------------------------------------------
// Handshake, heartbeat, backpressure

func fakeNetHost(t *testing.T, serve func(link *netplayLink)) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		link := defaultNetplayStack().adopt(conn)
		defer link.Close()
		serve(link)
	}()
	return ln.Addr().String()
}

func TestNetClientReportsHostVersionMismatch(t *testing.T) {
	addr := fakeNetHost(t, func(link *netplayLink) {
		_ = link.Write(netWireMessage{Type: netMessageChallenge, Protocol: netplayProtocolVersion, GameVersion: "v9.9.9 (net 00000000)", Challenge: "0123456789abcdef0123"}, maxServerWireFrame)
		time.Sleep(500 * time.Millisecond)
	})
	client, err := newClientNetplay(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	err = waitForClientHandshake(client, 3*time.Second)
	if err == nil || !strings.Contains(err.Error(), "Host has version v9.9.9 (net 00000000), you have "+netplayGameVersion()) {
		t.Fatalf("err=%v", err)
	}
}

func TestNetClientReportsOldProtocolHost(t *testing.T) {
	addr := fakeNetHost(t, func(link *netplayLink) {
		_ = link.Write(netWireMessage{Type: netMessageChallenge, Protocol: 18, Challenge: "0123456789abcdef0123"}, maxServerWireFrame)
		time.Sleep(500 * time.Millisecond)
	})
	client, err := newClientNetplay(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	err = waitForClientHandshake(client, 3*time.Second)
	if err == nil || !strings.Contains(err.Error(), "Host has version an older version (network protocol 18)") {
		t.Fatalf("err=%v", err)
	}
}

func TestNetHostRejectsMismatchedClientVersionWithReason(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	conn, err := net.Dial("tcp", host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	link := defaultNetplayStack().adopt(conn)
	defer link.Close()
	var challenge netWireMessage
	if err := link.Read(maxServerWireFrame, &challenge); err != nil {
		t.Fatal(err)
	}
	if challenge.GameVersion != netplayGameVersion() {
		t.Fatalf("challenge version=%q", challenge.GameVersion)
	}
	if err := link.Write(netWireMessage{Type: netMessageHello, Protocol: netplayProtocolVersion, GameVersion: "v0.0.1 (net deadbeef)"}, maxClientWireFrame); err != nil {
		t.Fatal(err)
	}
	var reply netWireMessage
	if err := link.ReadLenient(maxServerWireFrame, &reply); err != nil {
		t.Fatal(err)
	}
	if reply.Type != netMessageReject || !strings.Contains(reply.Reason, "Host has version "+netplayGameVersion()+", you have v0.0.1 (net deadbeef)") {
		t.Fatalf("reply=%+v", reply)
	}
	if host.connectedCount() != 0 {
		t.Fatal("mismatched client took a slot")
	}
}

func TestNetClientTimesOutSilentHost(t *testing.T) {
	old := netIdleTimeoutNS.Swap(int64(400 * time.Millisecond))
	defer netIdleTimeoutNS.Store(old)
	addr := fakeNetHost(t, func(link *netplayLink) {
		_ = link.Write(netWireMessage{Type: netMessageChallenge, Protocol: netplayProtocolVersion, GameVersion: netplayGameVersion(), Challenge: "0123456789abcdef0123"}, maxServerWireFrame)
		var hello netWireMessage
		_ = link.Read(maxClientWireFrame, &hello)
		_ = link.Write(netWireMessage{Type: netMessageHello, Protocol: netplayProtocolVersion, GameVersion: netplayGameVersion(), PlayerID: 2}, maxServerWireFrame)
		time.Sleep(3 * time.Second) // then say nothing at all
	})
	client, err := newClientNetplay(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := waitForClientHandshake(client, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for client.connected() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if client.connected() {
		t.Fatal("client stayed connected to a silent host")
	}
	if got := friendlyNetError(client.LastError()); !strings.Contains(got, "no data from the host") {
		t.Fatalf("disconnect reason=%q (err=%v)", got, client.LastError())
	}
}

func TestNetHeartbeatMeasuresPingBothWays(t *testing.T) {
	old := netPingIntervalNS.Swap(int64(50 * time.Millisecond))
	defer netPingIntervalNS.Store(old)
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	client, err := newClientNetplay(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	waitFor(t, "client ping", func() bool { _, ok := client.hostRTTMillis(); return ok })
	waitFor(t, "host ping", func() bool { _, ok := host.peerRTTMillis(2); return ok })
}

// A client that stops acknowledging snapshots must not get an unbounded
// backlog: the host stops after maxNetInflightTicks and then sends only the
// newest snapshot once the client catches up.
func TestNetHostBackpressureSendsOnlyNewest(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	conn, err := net.Dial("tcp", host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	link := defaultNetplayStack().adopt(conn)
	defer link.Close()
	var msg netWireMessage
	if err := link.Read(maxServerWireFrame, &msg); err != nil {
		t.Fatal(err)
	}
	_ = link.Write(netWireMessage{Type: netMessageHello, Protocol: netplayProtocolVersion, GameVersion: netplayGameVersion()}, maxClientWireFrame)
	if err := link.Read(maxServerWireFrame, &msg); err != nil || msg.Type != netMessageHello {
		t.Fatalf("hello: %+v %v", msg, err)
	}
	waitFor(t, "peer", host.connected)

	states := make(chan uint32, 256)
	go func() {
		for {
			var m netWireMessage
			if link.Read(maxServerWireFrame, &m) != nil {
				close(states)
				return
			}
			if m.Type == netMessageState {
				states <- m.Tick
			}
		}
	}()
	for i := 0; i < 60; i++ {
		host.queueState(netSnapshot{MapFXFrame: i})
		time.Sleep(2 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)
	var got []uint32
	for len(states) > 0 {
		got = append(got, <-states)
	}
	if len(got) == 0 || len(got) > maxNetInflightTicks {
		t.Fatalf("sent %d unacknowledged snapshots, want 1..%d", len(got), maxNetInflightTicks)
	}
	// Ack everything: the next snapshot sent must be the newest (tick 60).
	if err := link.Write(netWireMessage{Type: netMessageAck, Protocol: netplayProtocolVersion, Tick: got[len(got)-1]}, maxClientWireFrame); err != nil {
		t.Fatal(err)
	}
	select {
	case tick := <-states:
		if tick != 60 {
			t.Fatalf("after catching up the host sent stale tick %d, want newest 60", tick)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("host did not resume after ack")
	}
}

// End to end over loopback with the real snapshot builder. Run with -race:
// the host keeps reusing Player.PendingSounds' backing array (as gameplay
// does) while snapshots are being sent, which used to be shared with the
// writer goroutine.
func TestNetHostGameSnapshotsReachClientWithoutSharingState(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	client, err := newClientNetplay(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	waitFor(t, "client player assignment", func() bool { return client.assignedPlayer() == 2 })
	waitFor(t, "host peer", host.connected)

	hostGame := New()
	hostGame.netplay = host
	hostGame.screen = screenGameplay
	hostGame.arena = OriginalMap1()
	p1 := NewPlayer(1, hostGame.arena)
	p2 := NewPlayer(2, hostGame.arena)
	p2.Weapon.Def.Firepower *= 2 // runtime-modified definition is sent whole
	hostGame.players = []*Player{p1, p2}
	for i := 0; i < 40; i++ {
		p1.X += 1.5
		p1.PendingSounds = append(p1.PendingSounds[:0], "hit1.wav", "drop1.wav")
		hostGame.netSFXPending = append(hostGame.netSFXPending, netSFXEvent{Seq: uint64(i + 1), Name: "hit1.wav"})
		hostGame.queueNetHostSnapshot()
		p1.PendingSounds = p1.PendingSounds[:0]
		time.Sleep(3 * time.Millisecond)
	}
	clientGame := New()
	clientGame.netplay = client
	waitFor(t, "final snapshot", func() bool {
		s, ok := client.latestState()
		return ok && len(s.Players) == 2 && math.Abs(s.Players[0].X-p1.X) < 0.01
	})
	s, _ := client.nextPlayoutState()
	clientGame.applyNetSnapshot(s)
	if len(clientGame.players) != 2 || clientGame.arena.Number != 1 || len(clientGame.arena.Platforms) == 0 {
		t.Fatalf("client state incomplete: players=%d arena=%d", len(clientGame.players), clientGame.arena.Number)
	}
	if got := clientGame.players[1].Weapon.Def; got != p2.Weapon.Def {
		t.Fatalf("modified weapon def=%+v want %+v", got, p2.Weapon.Def)
	}
	if got := clientGame.players[0].Weapon.Def; got != p1.Weapon.Def {
		t.Fatalf("catalog weapon def=%+v want %+v", got, p1.Weapon.Def)
	}
	waitFor(t, "sounds", func() bool { return len(client.drainClientSFX()) > 0 })
}
