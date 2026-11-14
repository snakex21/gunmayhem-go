package game

import (
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// netsnapshot_wire.go: protocol 19 snapshot transport.
//
// Host, once per tick (game thread):
//   makeNetSnapshot -> prepareNetSnapshotForWire (caps effect lists, strips
//   weapon definitions to their number) -> compact encode (netcodec.go) ONCE.
//   The bytes are shared by every peer; nothing mutable crosses goroutines.
// Host, per peer (writer goroutine):
//   envelope (tick, per-peer SFX, map when it changed) + shared body ->
//   per-peer streaming DEFLATE (sync flush per message). The stream keeps the
//   previous snapshots in its 32 KB window, so unchanged data costs a few
//   bits: it acts as a cheap, always-correct delta against what this peer
//   already received (TCP/WSS are reliable and ordered).
// Client (reader goroutine):
//   inflate -> decode -> sanitize -> playout buffer. The game thread plays the
//   buffer ~1 snapshot behind and interpolates players across gaps.
//   Protocol 21: the envelope carries InputAck (this peer's newest simulated
//   input frame) and the snapshot the arena's runtime gravity/friction, for
//   the guest's own-player prediction (netplay_predict.go).
//
// Registering new state: new fields of Player/effect structs are sent
// automatically. A new effect LIST on netSnapshot is sent automatically too;
// add it to netSnapshotListCaps so a burst of effects is trimmed instead of
// growing without bound.

const (
	maxNetSnapshotRawLen = 4 * 1024 * 1024
	// A peer that has this many unacknowledged ticks in flight is skipped
	// until it catches up; it then receives only the newest snapshot.
	maxNetInflightTicks = 12
	netRecentSFXLimit   = 64
	netSFXPerMessageMax = 32
	netPlayoutMaxFrames = 16
	netPlayoutMaxDelay  = 4
	netInterpTeleportPx = 120.0
)

// netStateEnvelope is the per-peer header that precedes the shared snapshot
// body inside one compressed state message.
type netStateEnvelope struct {
	Tick     uint32
	ArenaSeq uint32 // non-zero: an encoded Map follows the snapshot body
	SFX      []netSFXEvent
	// InputAck (protocol 21) is the newest input frame of THIS peer that the
	// host had simulated when it built the snapshot (client-side prediction).
	InputAck uint32
}

type netListCap struct {
	name string
	max  int
	trim func(s *netSnapshot, max int) int // returns the length before trimming
}

func trimNetList[T any](list *[]T, max int) int {
	n := len(*list)
	if n > max {
		// Effects are appended in creation order: keep the newest ones.
		*list = (*list)[n-max:]
	}
	return n
}

// netSnapshotListCaps bounds every list in a snapshot. The host trims before
// sending (a flamethrower or grenade storm can create hundreds of effects);
// the client trims again instead of rejecting the snapshot, so an oversized
// frame can never freeze a client. Register new snapshot lists here.
var netSnapshotListCaps = []netListCap{
	{"players", 16, func(s *netSnapshot, m int) int { return trimNetList(&s.Players, m) }},
	{"bullets", 512, func(s *netSnapshot, m int) int { return trimNetList(&s.Bullets, m) }},
	{"instagib trails", 128, func(s *netSnapshot, m int) int { return trimNetList(&s.InstagibTrails, m) }},
	{"player trails", 128, func(s *netSnapshot, m int) int { return trimNetList(&s.PlayerTrails, m) }},
	{"shells", 128, func(s *netSnapshot, m int) int { return trimNetList(&s.Shells, m) }},
	{"muzzle flashes", 96, func(s *netSnapshot, m int) int { return trimNetList(&s.Flashes, m) }},
	{"grenades", 128, func(s *netSnapshot, m int) int { return trimNetList(&s.Grenades, m) }},
	{"fire patches", 256, func(s *netSnapshot, m int) int { return trimNetList(&s.FirePatches, m) }},
	{"area effects", 64, func(s *netSnapshot, m int) int { return trimNetList(&s.AreaEffects, m) }},
	{"portals", 64, func(s *netSnapshot, m int) int { return trimNetList(&s.Portals, m) }},
	{"dynamite fx", 256, func(s *netSnapshot, m int) int { return trimNetList(&s.DynamiteFX, m) }},
	{"blast fx", 256, func(s *netSnapshot, m int) int { return trimNetList(&s.BlastFX, m) }},
	{"jet fx", 128, func(s *netSnapshot, m int) int { return trimNetList(&s.JetThrustFX, m) }},
	{"drop fx", 64, func(s *netSnapshot, m int) int { return trimNetList(&s.DropPackFX, m) }},
	{"explosions", 128, func(s *netSnapshot, m int) int { return trimNetList(&s.Explosions, m) }},
	{"killfeeds", 32, func(s *netSnapshot, m int) int { return trimNetList(&s.Killfeeds, m) }},
	{"crates", 64, func(s *netSnapshot, m int) int { return trimNetList(&s.Crates, m) }},
	{"powerups", 64, func(s *netSnapshot, m int) int { return trimNetList(&s.Powerups, m) }},
	{"powerup names", 64, func(s *netSnapshot, m int) int { return trimNetList(&s.PowerupNameFX, m) }},
	{"life fx", 64, func(s *netSnapshot, m int) int { return trimNetList(&s.LifeBlingFX, m) }},
	{"campaign enemies", 32, func(s *netSnapshot, m int) int { return trimNetList(&s.CampaignEnemies, m) }},
	{"hazard markers", 32, func(s *netSnapshot, m int) int { return trimNetList(&s.HazardMarkers, m) }},
	{"zombies", zombieSnapshotCap, func(s *netSnapshot, m int) int { return trimNetList(&s.Zombies, m) }},
	{"zombie pops", zombiePopCap, func(s *netSnapshot, m int) int { return trimNetList(&s.ZombiePops, m) }},
	{"sfx events", netRecentSFXLimit, func(s *netSnapshot, m int) int { return trimNetList(&s.SFXEvents, m) }},
}

// trimNetSnapshotLists caps every list and reports which lists were cut.
func trimNetSnapshotLists(s *netSnapshot) []string {
	var trimmed []string
	for _, c := range netSnapshotListCaps {
		if n := c.trim(s, c.max); n > c.max {
			trimmed = append(trimmed, fmt.Sprintf("%s %d>%d", c.name, n, c.max))
		}
	}
	if len(s.Arena.Platforms) > 128 {
		s.Arena.Platforms = s.Arena.Platforms[:128]
	}
	return trimmed
}

// prepareNetSnapshotForWire works on the snapshot COPY built by
// makeNetSnapshot (its Players slice is already a fresh copy).
func prepareNetSnapshotForWire(s *netSnapshot) {
	trimNetSnapshotLists(s)
	for i := range s.Players {
		def := s.Players[i].Weapon.Def
		if def.Number == 0 {
			continue
		}
		// The client has the same weapon tables (checked by the handshake
		// fingerprint), so an unmodified definition travels as its number.
		// Runtime-modified definitions (e.g. firepower perks) are sent whole.
		if catalog, ok := WeaponByNumber(def.Number); ok && catalog == def {
			s.Players[i].Weapon.Def = WeaponDef{Number: def.Number}
		}
	}
}

// restoreNetSnapshotFromWire undoes prepareNetSnapshotForWire on the client.
func restoreNetSnapshotFromWire(s *netSnapshot) {
	for i := range s.Players {
		def := s.Players[i].Weapon.Def
		if def.Number != 0 && def == (WeaponDef{Number: def.Number}) {
			if catalog, ok := WeaponByNumber(def.Number); ok {
				s.Players[i].Weapon.Def = catalog
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Host side: one shared encoded frame per tick

type netHostFrame struct {
	tick uint32
	// views: one encoded snapshot per view. Ordinary matches have exactly
	// one; a multi-zone zombie match has one per zone that a peer is in
	// (zombie_zones.go), so each guest gets only its own zone's entities.
	views []netHostView
	// peerView[playerID] is the index of that peer's view.
	peerView [maxOnlinePlayers + 1]int
	sfx      []netSFXEvent // recent SFX ring (immutable copy)
	// inputAck[playerID] is that peer's netInputReplay.appliedSeq at the
	// moment the snapshot was taken (filled before the frame is shared).
	inputAck [maxOnlinePlayers + 1]uint32
}

// netHostView is one encoded snapshot body and the map that goes with it.
type netHostView struct {
	zone     int // zombie zone of the view (SFX tagged for another zone are skipped)
	body     []byte
	arenaSeq uint32
	arena    []byte
}

type netHostEncoder struct {
	tick      uint32
	arenaSeq  uint32
	arenas    []netHostArena // recently sent maps (one per zone of a multi-zone match)
	recentSFX []netSFXEvent
}

type netHostArena struct {
	seq uint32
	raw []byte
}

const netHostArenaCache = 8

// arenaSeqFor numbers every distinct map: a peer gets the map again only
// when its view switches to a map with another number.
func (h *netHostEncoder) arenaSeqFor(raw []byte) (uint32, []byte) {
	for _, a := range h.arenas {
		if bytes.Equal(a.raw, raw) {
			return a.seq, a.raw
		}
	}
	h.arenaSeq++
	h.arenas = append(h.arenas, netHostArena{seq: h.arenaSeq, raw: raw})
	if len(h.arenas) > netHostArenaCache {
		h.arenas = append(h.arenas[:0], h.arenas[len(h.arenas)-netHostArenaCache:]...)
	}
	return h.arenaSeq, raw
}

// encode builds the shared frame. state must be a snapshot copy owned by the
// caller; it is modified (trimmed/stripped).
func (h *netHostEncoder) encode(state netSnapshot) *netHostFrame {
	return h.encodeViews([]netSnapshot{state}, []int{0}, nil)
}

// encodeViews builds one frame with several views (zones[i] is the zombie
// zone of states[i]). peerView maps a player ID to its view index; peers
// that are not listed get view 0. The SFX of the first state are used.
func (h *netHostEncoder) encodeViews(states []netSnapshot, zones []int, peerView map[int]int) *netHostFrame {
	h.tick++
	frame := &netHostFrame{tick: h.tick}
	for i := range states {
		state := states[i]
		prepareNetSnapshotForWire(&state)
		if i == 0 && len(state.SFXEvents) > 0 {
			h.recentSFX = append(append([]netSFXEvent(nil), h.recentSFX...), state.SFXEvents...)
			if len(h.recentSFX) > netRecentSFXLimit {
				h.recentSFX = h.recentSFX[len(h.recentSFX)-netRecentSFXLimit:]
			}
		}
		seq, raw := h.arenaSeqFor(netEncodeValue(nil, &state.Arena))
		zone := 0
		if i < len(zones) {
			zone = zones[i]
		}
		frame.views = append(frame.views, netHostView{
			zone:     zone,
			body:     netEncodeValue(make([]byte, 0, 4096), &state),
			arenaSeq: seq,
			arena:    raw,
		})
	}
	for id, v := range peerView {
		if id >= 0 && id < len(frame.peerView) && v >= 0 && v < len(frame.views) {
			frame.peerView[id] = v
		}
	}
	frame.sfx = h.recentSFX
	return frame
}

// netPeerStream is the per-peer compression state. Only the peer's writer
// goroutine touches it.
type netPeerStream struct {
	zbuf      bytes.Buffer
	zw        *flate.Writer
	raw       []byte
	arenaSeq  uint32
	sfxSeq    uint64
	sfxSynced bool
}

func newNetPeerStream() *netPeerStream {
	s := &netPeerStream{}
	s.zw, _ = flate.NewWriter(&s.zbuf, flate.BestSpeed)
	return s
}

// message builds this peer's compressed state message for frame f.
func (p *netPeerStream) message(f *netHostFrame, playerID int) (payload []byte, rawLen int, err error) {
	if len(f.views) == 0 {
		return nil, 0, errors.New("snapshot frame has no view")
	}
	view := &f.views[0]
	if playerID >= 0 && playerID < len(f.peerView) && f.peerView[playerID] < len(f.views) {
		view = &f.views[f.peerView[playerID]]
	}
	env := netStateEnvelope{Tick: f.tick}
	if playerID >= 0 && playerID < len(f.inputAck) {
		env.InputAck = f.inputAck[playerID]
	}
	if !p.sfxSynced {
		// A newly joined peer must not replay sounds from before it joined.
		p.sfxSynced = true
		for _, ev := range f.sfx {
			if ev.Seq > p.sfxSeq {
				p.sfxSeq = ev.Seq
			}
		}
	}
	newest := p.sfxSeq
	for _, ev := range f.sfx {
		if ev.Seq <= p.sfxSeq {
			continue
		}
		newest = max(newest, ev.Seq)
		// Zone 0: everybody; zone z+1: only views of zombie zone z.
		if ev.Zone != 0 && ev.Zone != view.zone+1 {
			continue
		}
		env.SFX = append(env.SFX, ev)
	}
	if len(env.SFX) > netSFXPerMessageMax {
		env.SFX = env.SFX[len(env.SFX)-netSFXPerMessageMax:]
	}
	p.sfxSeq = newest
	sendArena := p.arenaSeq != view.arenaSeq
	if sendArena {
		env.ArenaSeq = view.arenaSeq
	}
	p.raw = netEncodeValue(p.raw[:0], &env)
	p.raw = append(p.raw, view.body...)
	if sendArena {
		p.raw = append(p.raw, view.arena...)
	}
	p.zbuf.Reset()
	if _, err := p.zw.Write(p.raw); err != nil {
		return nil, 0, err
	}
	if err := p.zw.Flush(); err != nil {
		return nil, 0, err
	}
	if sendArena {
		p.arenaSeq = view.arenaSeq
	}
	return append([]byte(nil), p.zbuf.Bytes()...), len(p.raw), nil
}

// ---------------------------------------------------------------------------
// Client side: streaming inflate + decode

// netChunkReader feeds the inflater exactly the bytes received so far. It is
// an io.ByteReader so flate never reads ahead past a message boundary.
type netChunkReader struct {
	data []byte
}

func (r *netChunkReader) push(b []byte) {
	if len(r.data) == 0 {
		r.data = append(r.data[:0], b...)
		return
	}
	r.data = append(r.data, b...)
}

func (r *netChunkReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.ErrUnexpectedEOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func (r *netChunkReader) ReadByte() (byte, error) {
	if len(r.data) == 0 {
		return 0, io.ErrUnexpectedEOF
	}
	b := r.data[0]
	r.data = r.data[1:]
	return b, nil
}

type netClientStream struct {
	in    netChunkReader
	zr    io.ReadCloser
	out   []byte
	arena Map
}

func newNetClientStream() *netClientStream {
	s := &netClientStream{}
	s.zr = flate.NewReader(&s.in)
	return s
}

type netDecodedState struct {
	tick     uint32
	inputAck uint32
	state    netSnapshot
	sfx      []netSFXEvent
}

func (c *netClientStream) decode(payload []byte, rawLen int) (netDecodedState, error) {
	if rawLen <= 0 || rawLen > maxNetSnapshotRawLen {
		return netDecodedState{}, fmt.Errorf("invalid snapshot size %d", rawLen)
	}
	c.in.push(payload)
	if cap(c.out) < rawLen {
		c.out = make([]byte, rawLen)
	}
	out := c.out[:rawLen]
	if _, err := io.ReadFull(c.zr, out); err != nil {
		return netDecodedState{}, fmt.Errorf("inflate snapshot: %w", err)
	}
	var env netStateEnvelope
	pos, err := netDecodeValue(out, &env)
	if err != nil {
		return netDecodedState{}, fmt.Errorf("decode snapshot header: %w", err)
	}
	var state netSnapshot
	n, err := netDecodeValue(out[pos:], &state)
	if err != nil {
		return netDecodedState{}, fmt.Errorf("decode snapshot: %w", err)
	}
	pos += n
	if env.ArenaSeq != 0 {
		var arena Map
		n, err := netDecodeValue(out[pos:], &arena)
		if err != nil {
			return netDecodedState{}, fmt.Errorf("decode map: %w", err)
		}
		pos += n
		c.arena = arena
	}
	if pos != len(out) {
		return netDecodedState{}, errors.New("snapshot has trailing data")
	}
	state.Arena = c.arena
	restoreNetSnapshotFromWire(&state)
	return netDecodedState{tick: env.Tick, inputAck: env.InputAck, state: state, sfx: env.SFX}, nil
}

// ---------------------------------------------------------------------------
// Client playout: jitter buffer + interpolation

type netPlayoutFrame struct {
	tick  uint32
	state netSnapshot
}

// netPlayout renders the host's snapshots one host tick per client tick,
// about `delay` ticks behind the newest one. Snapshots over TCP/WSS arrive in
// bursts; playing them from a small buffer turns that into smooth motion.
// When the host skipped ticks for this client (backpressure), players are
// interpolated between the two surrounding snapshots.
type netPlayout struct {
	mu       sync.Mutex
	frames   []netPlayoutFrame
	started  bool
	playTick int64
	delay    int64
	holds    []int64 // client update counter of recent underruns
	updates  int64
	calm     int64
}

func (p *netPlayout) push(tick uint32, state netSnapshot) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if n := len(p.frames); n > 0 && int64(tick) <= int64(p.frames[n-1].tick) {
		return // duplicate or out of order
	}
	p.frames = append(p.frames, netPlayoutFrame{tick: tick, state: state})
	if len(p.frames) > netPlayoutMaxFrames {
		p.frames = append(p.frames[:0], p.frames[len(p.frames)-netPlayoutMaxFrames:]...)
	}
}

// next advances the playback clock by one client tick and returns the state
// to show.
func (p *netPlayout) next() (netSnapshot, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.frames) == 0 {
		return netSnapshot{}, false
	}
	p.updates++
	if p.delay < 1 {
		p.delay = 1
	}
	newest := int64(p.frames[len(p.frames)-1].tick)
	oldest := int64(p.frames[0].tick)
	// The playback clock advances one host tick per client tick, independent
	// of when snapshots arrive; arrival jitter only changes how far behind
	// the newest snapshot it runs (lag).
	if !p.started {
		p.started = true
		p.playTick = max(newest-p.delay, oldest)
	} else {
		p.playTick++
	}
	lag := newest - p.playTick
	switch {
	case lag < 0:
		// Underrun: the next snapshot is late. Hold the newest frame; the
		// hold itself rebuilds the buffer.
		p.playTick = newest
		p.noteHold()
	case lag > p.delay+2*netPlayoutMaxDelay:
		p.playTick = newest - p.delay // far behind (stall): jump
	case lag > p.delay+2:
		p.playTick++ // running behind: skip one frame to catch up
	}
	p.calm++
	if p.calm > 150 && p.delay > 1 { // ~5 s without underruns
		p.delay--
		p.calm = 0
	}

	// Find a (tick <= playTick) and b (first tick > playTick).
	ai := 0
	for i := range p.frames {
		if int64(p.frames[i].tick) <= p.playTick {
			ai = i
		}
	}
	// Drop frames older than a.
	if ai > 0 {
		p.frames = append(p.frames[:0], p.frames[ai:]...)
		ai = 0
	}
	a := p.frames[ai]
	if int64(a.tick) >= p.playTick || ai+1 >= len(p.frames) {
		return a.state, true
	}
	b := p.frames[ai+1]
	alpha := float64(p.playTick-int64(a.tick)) / float64(int64(b.tick)-int64(a.tick))
	return interpolateNetSnapshot(a.state, b.state, alpha), true
}

func (p *netPlayout) noteHold() {
	p.calm = 0
	p.holds = append(p.holds, p.updates)
	// Two underruns within ~2 s: the arrival jitter is larger than the
	// buffer. Allow running one more tick behind before catching up.
	recent := p.holds[:0]
	for _, h := range p.holds {
		if p.updates-h <= 60 {
			recent = append(recent, h)
		}
	}
	p.holds = recent
	if len(p.holds) >= 2 && p.delay < netPlayoutMaxDelay {
		p.delay++
		p.holds = p.holds[:0]
	}
}

// interpolateNetSnapshot returns a with player positions and the camera
// moved alpha of the way towards b. Players that respawned or teleported are
// not interpolated.
func interpolateNetSnapshot(a, b netSnapshot, alpha float64) netSnapshot {
	if alpha <= 0 {
		return a
	}
	if a.Arena.Number != b.Arena.Number || a.ZombieHUD.View != b.ZombieHUD.View {
		// A zone change (multi-zone zombie map): nothing to blend between.
		return a
	}
	if alpha > 1 {
		alpha = 1
	}
	out := a
	out.Players = append([]Player(nil), a.Players...)
	for i := range out.Players {
		pa := &out.Players[i]
		for j := range b.Players {
			pb := &b.Players[j]
			if pb.ID != pa.ID || pb.IsDouble != pa.IsDouble || pb.DeathSerial != pa.DeathSerial {
				continue
			}
			dx, dy := pb.X-pa.X, pb.Y-pa.Y
			if dx*dx+dy*dy > netInterpTeleportPx*netInterpTeleportPx {
				break
			}
			pa.X += dx * alpha
			pa.Y += dy * alpha
			break
		}
	}
	out.CameraX = a.CameraX + (b.CameraX-a.CameraX)*alpha
	out.CameraY = a.CameraY + (b.CameraY-a.CameraY)*alpha
	return out
}

// ---------------------------------------------------------------------------
// Input: per-frame history

const (
	netInputHistoryFrames = 8
	netInputQueueMax      = 32
	netInputSilentAfter   = 300 * time.Millisecond
	netInputBacklogKeep   = 4
	netInputDebtMax       = 8
)

// netInputButtonMask keeps the seven defined button bits (left, right, up,
// down, shoot, grenade, ability) and drops anything else a peer sends.
const netInputButtonMask = 127

func packNetInput(in PlayerInputState) uint8 {
	var b uint8
	if in.Left {
		b |= 1
	}
	if in.Right {
		b |= 2
	}
	if in.Up {
		b |= 4
	}
	if in.Down {
		b |= 8
	}
	if in.Shoot {
		b |= 16
	}
	if in.Grenade {
		b |= 32
	}
	if in.Ability {
		b |= 64
	}
	return b
}

func unpackNetInput(b uint8) PlayerInputState {
	return PlayerInputState{
		Left: b&1 != 0, Right: b&2 != 0, Up: b&4 != 0, Down: b&8 != 0,
		Shoot: b&16 != 0, Grenade: b&32 != 0, Ability: b&64 != 0,
	}
}

// netInputHistory is the client side: every client frame gets a sequence
// number, and each input message repeats the last few frames. A message lost
// to the latest-value outbox, or several frames merged by TCP, therefore
// never loses a tap, a double tap, or a frame of grenade charge.
type netInputHistory struct {
	mu     sync.Mutex
	seq    uint32
	frames []uint8
}

func (h *netInputHistory) record(in PlayerInputState) (uint32, []uint8) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.seq++
	h.frames = append(h.frames, packNetInput(in))
	if len(h.frames) > netInputHistoryFrames {
		h.frames = h.frames[len(h.frames)-netInputHistoryFrames:]
	}
	return h.seq, append([]uint8(nil), h.frames...)
}

// netInputReplay is the host side for one peer: it queues every client frame
// exactly once, in order, and the host consumes one per simulation tick, so
// press/release timing (taps, semi-auto re-press, grenade charge) matches what
// the client player did.
type netInputReplay struct {
	lastSeq uint32
	// appliedSeq is the sequence number of the newest client frame a host
	// tick has consumed. Snapshots echo it (netStateEnvelope.InputAck) so the
	// client can replay only the frames the host has not simulated yet.
	appliedSeq uint32
	// debt counts host ticks that repeated `applied` while the next frame
	// was late (at most netInputDebtMax).
	debt     uint32
	queue    []uint8
	applied  uint8
	newest   uint8
	lastRecv time.Time
}

func (r *netInputReplay) receive(seq uint32, frames []uint8, now time.Time) {
	if len(frames) == 0 || seq == 0 {
		return
	}
	first := int64(seq) - int64(len(frames)) + 1
	for i, b := range frames {
		s := first + int64(i)
		if s <= int64(r.lastSeq) {
			continue
		}
		r.queue = append(r.queue, b&netInputButtonMask)
		r.lastSeq = uint32(s)
	}
	if len(r.queue) > netInputQueueMax {
		r.queue = append(r.queue[:0], r.queue[len(r.queue)-netInputQueueMax:]...)
	}
	r.newest = frames[len(frames)-1] & netInputButtonMask
	r.lastRecv = now
}

// take returns the input for one host tick.
func (r *netInputReplay) take(now time.Time) uint8 {
	// Ticks that repeated the held input because the next frame was late
	// already simulated the frames that later arrive with that same input:
	// skip them, so a network stall does not leave the player running
	// behind (and the client's prediction stays aligned, see ackSeq).
	for r.debt > 0 && len(r.queue) > 0 && r.queue[0] == r.applied {
		r.queue = r.queue[1:]
		r.debt--
	}
	if len(r.queue) == 0 {
		r.appliedSeq = r.lastSeq
		if r.lastRecv.IsZero() || now.Sub(r.lastRecv) > netInputSilentAfter {
			// The client went silent (frozen, alt-tabbed on an old build,
			// or the link stalled): never keep a key held.
			r.applied = 0
			r.debt = 0
		} else if r.debt < netInputDebtMax {
			r.debt++
		}
		return r.applied
	}
	// Running behind (burst after jitter): drop frames that repeat the
	// current state. They carry no press/release, so no input is lost.
	// A backlog of up to netInputBacklogKeep frames is normal jitter and is
	// played out unchanged, so hold lengths (grenade charge) stay exact.
	for len(r.queue) > netInputBacklogKeep && r.queue[0] == r.applied {
		r.queue = r.queue[1:]
	}
	r.applied = r.queue[0]
	r.queue = r.queue[1:]
	r.appliedSeq = r.lastSeq - uint32(len(r.queue))
	return r.applied
}

// ackSeq is the input acknowledgement sent with snapshots: the newest frame
// the host state accounts for. Repeated ticks (debt) stand for the frames
// that were late, so the client does not replay them on top.
func (r *netInputReplay) ackSeq() uint32 {
	return r.appliedSeq + r.debt
}
