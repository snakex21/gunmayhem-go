package game

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	// 19: compact streamed snapshots, per-frame input history, heartbeat,
	// game version in the handshake.
	netplayProtocolVersion = 20
	maxOnlinePlayers       = 4
)

type netplayMode int

const (
	netplayOff netplayMode = iota
	netplayHost
	netplayClient
)

const (
	netMessageHello         = 1
	netMessageInput         = 2
	netMessageState         = 3
	netMessagePlayerConfig  = 4
	netMessageChallenge     = 5
	netMessageProbe         = 6
	netMessageProbeReply    = 7
	netMessagePostGameReady = 8
	netMessageReject        = 9  // host -> client, Reason
	netMessagePing          = 10 // both ways, PingTime
	netMessagePong          = 11 // both ways, echoes PingTime
	netMessageAck           = 12 // client -> host, newest received Tick
)

// netIdleTimeout disconnects a peer that sent nothing (not even a heartbeat)
// for this long. Both sides send a ping every netPingInterval, which also keeps
// Cloudflare's ~100 s idle WebSocket timeout from firing.
var (
	netIdleTimeoutNS  atomic.Int64 // tests shorten these; see setNetTimingForTest
	netPingIntervalNS atomic.Int64
)

func init() {
	netIdleTimeoutNS.Store(int64(10 * time.Second))
	netPingIntervalNS.Store(int64(time.Second))
}

func netIdleTimeout() time.Duration  { return time.Duration(netIdleTimeoutNS.Load()) }
func netPingInterval() time.Duration { return time.Duration(netPingIntervalNS.Load()) }

var errNetPeerTimeout = errors.New("connection timed out: no data for 10 s")

type netSFXEvent struct {
	Seq  uint64
	Name string
	Loud bool
	Gain float64
}

type netProbeInfo struct {
	ID                   string                     `cbor:",omitempty"`
	Name                 string                     `cbor:",omitempty"`
	Players              int                        `cbor:",omitempty"`
	MaxPlayers           int                        `cbor:",omitempty"`
	GameMode             int                        `cbor:",omitempty"`
	Lives                int                        `cbor:",omitempty"`
	Rounds               int                        `cbor:",omitempty"`
	Maps                 [customSeriesMaxRounds]int `cbor:",omitempty"`
	PlaylistConfigured   bool                       `cbor:",omitempty"`
	Crates               bool                       `cbor:",omitempty"`
	Pickups              bool                       `cbor:",omitempty"`
	RuleSummary          string                     `cbor:",omitempty"`
	ChaosEnabled         bool                       `cbor:",omitempty"`
	ChaosEvents          int                        `cbor:",omitempty"`
	ChaosIntervalSeconds int                        `cbor:",omitempty"`
	DirectAddress        string                     `cbor:",omitempty"`
	TunnelAddress        string                     `cbor:",omitempty"`
}

type netWireMessage struct {
	Type      int                `cbor:",omitempty"`
	Protocol  int                `cbor:",omitempty"`
	PlayerID  int                `cbor:",omitempty"`
	Challenge string             `cbor:",omitempty"`
	AuthProof string             `cbor:",omitempty"`
	Probe     netProbeInfo       `cbor:",omitempty"`
	Config    CustomPlayerConfig `cbor:",omitempty"`
	Ready     bool               `cbor:",omitempty"`

	// Handshake identity (protocol 19+).
	GameVersion string `cbor:",omitempty"`
	Reason      string `cbor:",omitempty"`

	// Heartbeat and flow control.
	PingTime int64  `cbor:",omitempty"`
	Tick     uint32 `cbor:",omitempty"`

	// Input: InputFrames are the packed buttons of client frames
	// InputSeq-len+1 .. InputSeq (oldest first).
	InputSeq    uint32 `cbor:",omitempty"`
	InputFrames []byte `cbor:",omitempty"`

	// State: one DEFLATE-stream chunk of a compact snapshot (netsnapshot_wire.go).
	Payload    []byte `cbor:",omitempty"`
	PayloadLen int    `cbor:",omitempty"`
}

// netSnapshot is deliberately a render-complete snapshot of the authoritative
// host. The client does not run combat physics; it renders the newest host
// snapshot and only sends its local input back to the host.
type netSnapshot struct {
	// cbor:",omitempty" on snapshot fields keeps the render-complete state small.
	// The host sends one of these per game frame over a Cloudflare WSS link, so
	// the mostly-zero fields were previously pure overhead.
	Screen int `cbor:",omitempty"`
	// Arena is sent separately, only when it changes (netsnapshot_wire.go).
	Arena Map `cbor:",omitempty" net:"-"`

	Players        []Player              `cbor:",omitempty"`
	Bullets        []Bullet              `cbor:",omitempty"`
	InstagibTrails []InstagibTrailEffect `cbor:",omitempty"`
	PlayerTrails   []PlayerTrailEffect   `cbor:",omitempty"`
	Shells         []Shell               `cbor:",omitempty"`
	Flashes        []MuzzleFlash         `cbor:",omitempty"`
	Grenades       []Grenade             `cbor:",omitempty"`
	FirePatches    []FirePatch           `cbor:",omitempty"`
	AreaEffects    []AreaEffect          `cbor:",omitempty"`
	Portals        []Portal              `cbor:",omitempty"`
	DynamiteFX     []DynamiteEffect      `cbor:",omitempty"`
	BlastFX        []BlastEffect         `cbor:",omitempty"`
	JetThrustFX    []JetThrustEffect     `cbor:",omitempty"`
	DropPackFX     []DropPackEffect      `cbor:",omitempty"`
	Explosions     []Explosion           `cbor:",omitempty"`
	Killfeeds      []KillFeedEntry       `cbor:",omitempty"`
	Crates         []Crate               `cbor:",omitempty"`
	Powerups       []Powerup             `cbor:",omitempty"`
	PowerupNameFX  []PowerupNameEffect   `cbor:",omitempty"`
	LifeBlingFX    []LifeBlingEffect     `cbor:",omitempty"`
	// Campaign-only state. Campaigns are local today; carrying these keeps a
	// future online co-op campaign rendering correctly with no extra work.
	CampaignEnemies []CampaignEnemy `cbor:",omitempty"`
	HazardMarkers   []HazardMarker  `cbor:",omitempty"`
	// SFXEvents travel in the per-peer envelope so a dropped snapshot never
	// drops a sound (netsnapshot_wire.go).
	SFXEvents []netSFXEvent `cbor:",omitempty" net:"-"`

	MapFXFrame int     `cbor:",omitempty"`
	CameraX    float64 `cbor:",omitempty"`
	CameraY    float64 `cbor:",omitempty"`

	GameMode   int  `cbor:",omitempty"`
	TotalLives int  `cbor:",omitempty"`
	TeamGame   bool `cbor:",omitempty"`
	CrateON    bool `cbor:",omitempty"`
	PowerON    bool `cbor:",omitempty"`
	GameWin    bool `cbor:",omitempty"`

	OnlineMatchSetup               bool                           `cbor:",omitempty"`
	CustomPage                     int                            `cbor:",omitempty"`
	CustomMenuX                    float64                        `cbor:",omitempty"`
	CustomMode                     int                            `cbor:",omitempty"`
	CustomMap                      int                            `cbor:",omitempty"`
	CustomMapSetGM2                bool                           `cbor:",omitempty"`
	CustomLives                    int                            `cbor:",omitempty"`
	CustomSeriesRounds             int                            `cbor:",omitempty"`
	CustomSeriesRound              int                            `cbor:",omitempty"`
	CustomSeriesStats              [4]customSeriesPlayerStats     `cbor:",omitempty"`
	CustomSeriesMaps               [customSeriesMaxRounds]int     `cbor:",omitempty"`
	CustomSeriesPlaylistConfigured bool                           `cbor:",omitempty"`
	ChaosEnabled                   bool                           `cbor:",omitempty"`
	ChaosCurrentEffect             int                            `cbor:",omitempty"`
	ChaosFrame                     int                            `cbor:",omitempty"`
	ChaosBannerFrame               int                            `cbor:",omitempty"`
	ChaosIntervalSeconds           int                            `cbor:",omitempty"`
	OnlinePlayerCount              int                            `cbor:",omitempty"`
	OnlinePlayerSlots              [4]bool                        `cbor:",omitempty"`
	OnlineCustomPlayers            [4]CustomPlayerConfig          `cbor:",omitempty"`
	OnlineMapRotation              int                            `cbor:",omitempty"`
	OnlineWeaponMode               int                            `cbor:",omitempty"`
	OnlineOnlyWeapon               int                            `cbor:",omitempty"`
	OnlinePerksEnabled             bool                           `cbor:",omitempty"`
	OnlinePowerupsEnabled          bool                           `cbor:",omitempty"`
	OnlineBlockedWeapons           [developedWeaponSlotCount]bool `cbor:",omitempty"`
	OnlineBlockedPerks             [onlinePerkTableSize]bool      `cbor:",omitempty"`
	OnlineBlockedAbilities         [abilityCount]bool             `cbor:",omitempty"`
	PostGameReady                  [4]bool                        `cbor:",omitempty"`
	PostGameReadyCountdown         int                            `cbor:",omitempty"`

	Paused            bool `cbor:",omitempty"`
	PausePressed      int  `cbor:",omitempty"`
	FadeActive        bool `cbor:",omitempty"`
	FadeFrame         int  `cbor:",omitempty"`
	FadeTarget        int  `cbor:",omitempty"`
	FadePurpose       int  `cbor:",omitempty"`
	MatchWinCountdown int  `cbor:",omitempty"`
	SoloWinFrame      int  `cbor:",omitempty"`
	TeamGameWin       bool `cbor:",omitempty"`
	TeamWinner        int  `cbor:",omitempty"`
	WinnerPlayerID    int  `cbor:",omitempty"`
	WinnerAnimFrame   int  `cbor:",omitempty"`
	TeamWinAnimFrame  int  `cbor:",omitempty"`
	CampaignLoseFrame int  `cbor:",omitempty"`
	ZombieWaveFrame   int  `cbor:",omitempty"`

	HUDLastLifeFrame   [4]int  `cbor:",omitempty"`
	HUDLastLifePlaying [4]bool `cbor:",omitempty"`
	HUDLastLevel       [4]int  `cbor:",omitempty"`

	CampaignMode  bool `cbor:",omitempty"`
	CampaignLevel int  `cbor:",omitempty"`

	Gototest        bool `cbor:",omitempty"`
	TestGunNumber   int  `cbor:",omitempty"`
	TestGunDisabled bool `cbor:",omitempty"`
	TestGunRespawn  int  `cbor:",omitempty"`
	TestGunFrame    int  `cbor:",omitempty"`
}

type netplayPeer struct {
	playerID int
	conn     net.Conn
	link     *netplayLink
	addr     string

	inputMu       sync.Mutex
	input         netInputReplay
	configMu      sync.RWMutex
	config        CustomPlayerConfig
	hasConfig     atomic.Bool
	postGameReady atomic.Bool

	stateSend chan *netHostFrame
	pongSend  chan int64
	ackSignal chan struct{}
	ackTick   atomic.Uint32
	sentTick  atomic.Uint32
	rttMillis atomic.Int64
	done      chan struct{}
	closed    atomic.Bool
}

type netplaySession struct {
	mode  netplayMode
	stack *netplayStack

	listener net.Listener
	closed   atomic.Bool
	done     chan struct{}

	peersMu           sync.RWMutex
	peers             map[int]*netplayPeer
	pendingHandshakes atomic.Int32

	connMu sync.Mutex
	conn   net.Conn
	link   *netplayLink
	ready  atomic.Bool

	assignedPlayerID   atomic.Int32
	allowedRemote      [5]atomic.Bool
	inviteOnly         bool
	expectedSessionKey string
	shortJoinCode      string
	publicSessionID    string
	publicSessionName  string
	clientSessionKey   string
	probeMu            sync.RWMutex
	probeInfo          netProbeInfo
	publicTunnelURL    string

	stateMu  sync.RWMutex
	state    netSnapshot
	hasState atomic.Bool

	// Host: shared per-tick snapshot encoder (game thread).
	hostEncMu sync.Mutex
	hostEnc   netHostEncoder
	// Client: playout buffer, received sounds, input history, ping.
	playout      netPlayout
	sfxMu        sync.Mutex
	sfxPending   []netSFXEvent
	sfxLastSeq   uint64
	inputHistory netInputHistory
	rttMillis    atomic.Int64
	outbox       *netplayOutbox
	lastErrMu    sync.RWMutex
	lastErr      error
	remoteAddr   string
}

func (s *netplaySession) setPublicProbeDetails(info netProbeInfo) {
	if s == nil {
		return
	}
	s.probeMu.Lock()
	info.ID = ""
	info.Name = ""
	info.Players = 0
	info.MaxPlayers = 0
	s.probeInfo = info
	s.probeMu.Unlock()
}

func (s *netplaySession) publicProbeInfo() netProbeInfo {
	if s == nil {
		return netProbeInfo{}
	}
	s.probeMu.RLock()
	info := s.probeInfo
	info.TunnelAddress = s.publicTunnelURL
	s.probeMu.RUnlock()
	info.ID = s.publicSessionID
	info.Name = s.publicSessionName
	info.Players = 1 + s.connectedCount()
	info.MaxPlayers = s.humanCapacity()
	return info
}

func (s *netplaySession) setPublicTunnelURL(raw string) {
	if s == nil {
		return
	}
	s.probeMu.Lock()
	s.publicTunnelURL = strings.TrimSpace(raw)
	s.probeMu.Unlock()
}

func normalizeListenAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ":7777"
	}
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	if _, err := strconv.ParseUint(addr, 10, 16); err == nil {
		return ":" + addr
	}
	if ip := net.ParseIP(strings.Trim(addr, "[]")); ip != nil {
		return net.JoinHostPort(ip.String(), "7777")
	}
	return net.JoinHostPort(strings.Trim(addr, "[]"), "7777")
}

func normalizeJoinAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "127.0.0.1:7777"
	}
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	if ip := net.ParseIP(strings.Trim(addr, "[]")); ip != nil {
		return net.JoinHostPort(ip.String(), "7777")
	}
	return net.JoinHostPort(strings.Trim(addr, "[]"), "7777")
}

func publicOnlineSessionID(sessionKey string) string {
	key := strings.TrimSpace(sessionKey)
	if key == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("GunMayhem-RE-public-session|" + key))
	return strings.ToUpper(hex.EncodeToString(sum[:8]))
}

func newHostNetplay(addr string) (*netplaySession, error) {
	return newHostNetplayWithStack(addr, defaultNetplayStack())
}

func newHostNetplayWithStack(addr string, stack *netplayStack) (*netplaySession, error) {
	stack = normalizeNetplayStack(stack)
	ln, err := stack.listen(normalizeListenAddress(addr))
	if err != nil {
		return nil, err
	}
	s := &netplaySession{
		mode:     netplayHost,
		stack:    stack,
		listener: ln,
		peers:    make(map[int]*netplayPeer),
		done:     make(chan struct{}),
	}
	for id := 2; id <= maxOnlinePlayers; id++ {
		s.allowedRemote[id].Store(true)
	}
	go s.acceptLoop()
	return s, nil
}

func newClientNetplay(addr string) (*netplaySession, error) {
	return newClientNetplayWithKey(addr, "", 5*time.Second)
}

func newClientNetplayWithTimeout(addr string, timeout time.Duration) (*netplaySession, error) {
	return newClientNetplayWithKey(addr, "", timeout)
}

func newClientNetplayWithKey(addr, sessionKey string, timeout time.Duration) (*netplaySession, error) {
	return newClientNetplayWithStack(addr, sessionKey, timeout, defaultNetplayStack())
}

func newClientNetplayWithStack(addr, sessionKey string, timeout time.Duration, stack *netplayStack) (*netplaySession, error) {
	address := normalizeJoinAddress(addr)
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	stack = normalizeNetplayStack(stack)
	conn, err := stack.dial(address, timeout)
	if err != nil {
		return nil, err
	}
	return newClientNetplayFromConnWithStack(conn, conn.RemoteAddr().String(), sessionKey, stack)
}

func newClientNetplayFromConn(conn net.Conn, remoteLabel string) (*netplaySession, error) {
	return newClientNetplayFromConnWithKey(conn, remoteLabel, "")
}

func newClientNetplayFromConnWithKey(conn net.Conn, remoteLabel, sessionKey string) (*netplaySession, error) {
	return newClientNetplayFromConnWithStack(conn, remoteLabel, sessionKey, defaultNetplayStack())
}

func newClientNetplayFromConnWithStack(conn net.Conn, remoteLabel, sessionKey string, stack *netplayStack) (*netplaySession, error) {
	if conn == nil {
		return nil, errors.New("nil netplay connection")
	}
	stack = normalizeNetplayStack(stack)
	link := stack.adopt(conn)
	if strings.TrimSpace(remoteLabel) == "" {
		remoteLabel = conn.RemoteAddr().String()
	}
	s := &netplaySession{
		mode:             netplayClient,
		stack:            stack,
		conn:             conn,
		link:             link,
		outbox:           newNetplayOutbox(),
		done:             make(chan struct{}),
		remoteAddr:       remoteLabel,
		clientSessionKey: strings.TrimSpace(sessionKey),
	}
	go s.clientRun(link)
	return s, nil
}

func (s *netplaySession) attachIncomingConn(conn net.Conn) bool {
	if s == nil || conn == nil || s.closed.Load() || s.mode != netplayHost {
		if conn != nil {
			_ = conn.Close()
		}
		return false
	}
	if s.pendingHandshakes.Add(1) > maxPendingNetHandshakes {
		s.pendingHandshakes.Add(-1)
		_ = conn.Close()
		return false
	}
	go func() {
		defer s.pendingHandshakes.Add(-1)
		s.prepareIncomingConn(conn)
	}()
	return true
}

func (s *netplaySession) prepareIncomingConn(conn net.Conn) {
	link := normalizeNetplayStack(s.stack).adopt(conn)
	challenge, err := newOnlineAuthChallenge()
	if err != nil {
		_ = link.Close()
		return
	}
	// reject tells a protocol-19+ client why it was turned away (shown in its
	// join error) before closing.
	reject := func(reason, logReason string) {
		networkDiagnostic("HOST handshake reject remote=%s reason=%s", conn.RemoteAddr(), logReason)
		if reason != "" {
			_ = link.Write(netWireMessage{Type: netMessageReject, Protocol: netplayProtocolVersion, GameVersion: netplayGameVersion(), Reason: reason}, maxServerWireFrame)
		}
		_ = link.Close()
	}
	_ = link.SetReadDeadline(time.Now().Add(netHandshakeTimeout))
	if err := link.Write(netWireMessage{Type: netMessageChallenge, Protocol: netplayProtocolVersion, GameVersion: netplayGameVersion(), Challenge: challenge}, maxServerWireFrame); err != nil {
		networkDiagnostic("HOST handshake reject remote=%s reason=write_challenge err=%v", conn.RemoteAddr(), err)
		_ = link.Close()
		return
	}
	var first netWireMessage
	// Lenient: a client from another build may carry fields this build does
	// not know; decode what we can so it gets a clear version message.
	if err := link.ReadLenient(maxClientWireFrame, &first); err != nil {
		networkDiagnostic("HOST handshake reject remote=%s reason=read_hello err=%v", conn.RemoteAddr(), err)
		_ = link.Close()
		return
	}
	if first.Type == netMessageHello && (first.Protocol != netplayProtocolVersion || first.GameVersion != netplayGameVersion()) {
		reject(netVersionMismatchMessage(netplayGameVersion(), netRemoteVersionLabel(first.Protocol, first.GameVersion)),
			fmt.Sprintf("version_mismatch remote=%q protocol=%d", first.GameVersion, first.Protocol))
		return
	}
	if err := validateNetMessageForHost(first); err != nil {
		networkDiagnostic("HOST handshake reject remote=%s reason=invalid_message err=%v", conn.RemoteAddr(), err)
		_ = link.Close()
		return
	}
	if first.Type == netMessageProbe {
		// Public OPEN discovery can verify a DHT endpoint without consuming a
		// player slot. INVITE ONLY sessions never answer public probes.
		if s.inviteOnly {
			networkDiagnostic("HOST handshake reject remote=%s reason=invite_probe", conn.RemoteAddr())
			_ = link.Close()
			return
		}
		info := s.publicProbeInfo()
		_ = link.Write(netWireMessage{Type: netMessageProbeReply, Protocol: netplayProtocolVersion, Probe: info}, maxServerWireFrame)
		_ = link.Close()
		return
	}
	if first.Type != netMessageHello {
		networkDiagnostic("HOST handshake reject remote=%s reason=expected_hello type=%d", conn.RemoteAddr(), first.Type)
		_ = link.Close()
		return
	}
	if s.inviteOnly && !onlineSessionProofEqual(s.expectedSessionKey, challenge, strings.TrimSpace(first.AuthProof)) {
		reject("wrong or expired invite code", "invite_auth")
		return
	}
	_ = link.SetReadDeadline(time.Time{})
	if s.closed.Load() {
		_ = link.Close()
		return
	}

	s.peersMu.Lock()
	playerID := 0
	for id := 2; id <= maxOnlinePlayers; id++ {
		if !s.allowedRemote[id].Load() {
			continue
		}
		if _, used := s.peers[id]; !used {
			playerID = id
			break
		}
	}
	if playerID == 0 {
		s.peersMu.Unlock()
		reject("the game is full", "no_online_slot")
		return
	}
	peer := &netplayPeer{
		playerID:  playerID,
		conn:      conn,
		link:      link,
		addr:      conn.RemoteAddr().String(),
		stateSend: make(chan *netHostFrame, 1),
		pongSend:  make(chan int64, 1),
		ackSignal: make(chan struct{}, 1),
		done:      make(chan struct{}),
	}
	s.peers[playerID] = peer
	s.peersMu.Unlock()
	if err := link.Write(netWireMessage{Type: netMessageHello, Protocol: netplayProtocolVersion, GameVersion: netplayGameVersion(), PlayerID: playerID}, maxServerWireFrame); err != nil {
		s.peersMu.Lock()
		if s.peers[playerID] == peer {
			delete(s.peers, playerID)
		}
		s.peersMu.Unlock()
		s.setErr(err)
		_ = link.Close()
		return
	}
	networkDiagnostic("HOST handshake accepted remote=%s player=%d version=%q", conn.RemoteAddr(), playerID, first.GameVersion)
	go s.hostServePeer(peer)
}

func (s *netplaySession) acceptLoop() {
	for !s.closed.Load() {
		conn, err := s.listener.Accept()
		if err != nil {
			if !s.closed.Load() {
				s.setErr(err)
			}
			return
		}
		s.attachIncomingConn(conn)
	}
}

func (p *netplayPeer) close() {
	if p == nil || p.closed.Swap(true) {
		return
	}
	close(p.done)
	if p.link != nil {
		_ = p.link.Close()
	} else {
		_ = p.conn.Close()
	}
}

// takeInput returns the input for one host simulation tick (see
// netInputReplay): every client frame is applied once and in order.
func (p *netplayPeer) takeInput(now time.Time) PlayerInputState {
	if p == nil {
		return PlayerInputState{}
	}
	p.inputMu.Lock()
	defer p.inputMu.Unlock()
	return unpackNetInput(p.input.take(now))
}

// queueState offers the newest frame to the peer's writer. Only the newest
// frame is kept: a slow peer skips stale snapshots instead of queueing them.
func (p *netplayPeer) queueState(frame *netHostFrame) {
	if p == nil || frame == nil || p.closed.Load() {
		return
	}
	signalLatest(p.stateSend, frame)
}

// signalLatest puts v in a capacity-1 channel, replacing an unread value.
func signalLatest[T any](ch chan T, v T) {
	select {
	case ch <- v:
		return
	default:
	}
	select {
	case <-ch:
	default:
	}
	select {
	case ch <- v:
	default:
	}
}

func isNetTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// recordNetRTT folds one ping round trip into dst (stored as ms+1; 0 means
// unknown).
func recordNetRTT(dst *atomic.Int64, sentUnixNano int64) {
	if dst == nil || sentUnixNano <= 0 {
		return
	}
	rtt := time.Since(time.Unix(0, sentUnixNano)).Milliseconds()
	if rtt < 0 || rtt > 60000 {
		return
	}
	if old := dst.Load() - 1; old >= 0 {
		rtt = (old*3 + rtt) / 4
	}
	dst.Store(rtt + 1)
}

func loadNetRTT(src *atomic.Int64) (int64, bool) {
	v := src.Load()
	return v - 1, v > 0
}

func (p *netplayPeer) inflightTicks() int64 {
	return int64(p.sentTick.Load()) - int64(p.ackTick.Load())
}

// hostPeerWriteLoop is the only writer for one peer: state, pings and pongs.
func (s *netplaySession) hostPeerWriteLoop(peer *netplayPeer) {
	stream := newNetPeerStream()
	ping := time.NewTicker(netPingInterval())
	defer ping.Stop()
	write := func(msg netWireMessage) bool {
		msg.Protocol = netplayProtocolVersion
		msg.PlayerID = peer.playerID
		if err := peer.link.Write(msg, maxServerWireFrame); err != nil {
			s.setErr(err)
			peer.close()
			return false
		}
		return true
	}
	var pending *netHostFrame
	for {
		select {
		case frame := <-peer.stateSend:
			pending = frame
		case echo := <-peer.pongSend:
			if !write(netWireMessage{Type: netMessagePong, PingTime: echo}) {
				return
			}
		case <-ping.C:
			if !write(netWireMessage{Type: netMessagePing, PingTime: time.Now().UnixNano()}) {
				return
			}
		case <-peer.ackSignal:
		case <-peer.done:
			return
		}
		if pending == nil {
			continue
		}
		// Backpressure that also works through cloudflared, where writes to
		// the local tunnel process never block: stop sending while too many
		// snapshots are unacknowledged, then send only the newest one.
		if peer.sentTick.Load() != 0 && peer.inflightTicks() >= maxNetInflightTicks {
			continue
		}
		payload, rawLen, err := stream.message(pending)
		if err != nil {
			s.setErr(err)
			peer.close()
			return
		}
		if !write(netWireMessage{Type: netMessageState, Tick: pending.tick, Payload: payload, PayloadLen: rawLen}) {
			return
		}
		peer.sentTick.Store(pending.tick)
		pending = nil
	}
}

func (s *netplaySession) hostServePeer(peer *netplayPeer) {
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		s.hostPeerWriteLoop(peer)
	}()

	var rate netRateGuard
	var peerReadErr error
	for !s.closed.Load() && !peer.closed.Load() {
		var msg netWireMessage
		_ = peer.link.SetReadDeadline(time.Now().Add(netIdleTimeout()))
		if err := peer.link.Read(maxClientWireFrame, &msg); err != nil {
			if isNetTimeout(err) {
				err = errNetPeerTimeout
			}
			peerReadErr = err
			if !errors.Is(err, net.ErrClosed) && !peer.closed.Load() {
				s.setErr(err)
			}
			break
		}
		if !rate.Allow(time.Now()) {
			peerReadErr = errors.New("remote client exceeded netplay message rate limit")
			s.setErr(peerReadErr)
			break
		}
		if err := validateNetMessageForHost(msg); err != nil || msg.Type == netMessageHello {
			if err == nil {
				err = errors.New("unexpected repeated netplay hello")
			}
			peerReadErr = err
			s.setErr(err)
			break
		}
		switch msg.Type {
		case netMessageInput:
			peer.inputMu.Lock()
			peer.input.receive(msg.InputSeq, msg.InputFrames, time.Now())
			peer.inputMu.Unlock()
		case netMessagePlayerConfig:
			cfg, err := sanitizeRemotePlayerConfig(msg.Config)
			if err != nil {
				s.setErr(err)
				peer.close()
				break
			}
			peer.configMu.Lock()
			peer.config = cfg
			peer.configMu.Unlock()
			peer.hasConfig.Store(true)
		case netMessagePostGameReady:
			peer.postGameReady.Store(msg.Ready)
		case netMessageAck:
			if msg.Tick > peer.ackTick.Load() && msg.Tick <= peer.sentTick.Load() {
				peer.ackTick.Store(msg.Tick)
			}
			signalLatest(peer.ackSignal, struct{}{})
		case netMessagePing:
			signalLatest(peer.pongSend, msg.PingTime)
		case netMessagePong:
			recordNetRTT(&peer.rttMillis, msg.PingTime)
		}
	}
	peer.close()
	<-writerDone
	s.peersMu.Lock()
	removed := false
	if current := s.peers[peer.playerID]; current == peer {
		delete(s.peers, peer.playerID)
		removed = true
	}
	remaining := make([]int, 0, len(s.peers))
	for id := 2; id <= maxOnlinePlayers; id++ {
		if _, ok := s.peers[id]; ok {
			remaining = append(remaining, id)
		}
	}
	s.peersMu.Unlock()
	// Without this line a match abort is indistinguishable from a genuine client
	// disconnect: the guard only reports "fewer than two participants".
	if removed {
		rtt, _ := loadNetRTT(&peer.rttMillis)
		networkDiagnostic("HOST peer disconnected player=%d addr=%s err=%v remaining=%v sent=%dB recv=%dB rtt=%dms", peer.playerID, peer.addr, peerReadErr, remaining, peer.link.bytesOut.Load(), peer.link.bytesIn.Load(), rtt)
	}
}

func (s *netplaySession) clientRun(link *netplayLink) {
	defer func() {
		s.ready.Store(false)
		_ = link.Close()
	}()

	_ = link.SetReadDeadline(time.Now().Add(netHandshakeTimeout))
	var challenge netWireMessage
	if err := link.ReadLenient(maxServerWireFrame, &challenge); err != nil {
		s.setErr(fmt.Errorf("netplay handshake challenge: %w", err))
		return
	}
	if challenge.Type == netMessageReject {
		s.setErr(fmt.Errorf("host rejected the connection: %s", sanitizeNetReason(challenge.Reason)))
		return
	}
	if challenge.Type != netMessageChallenge {
		s.setErr(errors.New("netplay handshake challenge: invalid host authentication challenge"))
		return
	}
	if challenge.Protocol != netplayProtocolVersion || challenge.GameVersion != netplayGameVersion() {
		s.setErr(errors.New(netVersionMismatchMessage(netRemoteVersionLabel(challenge.Protocol, challenge.GameVersion), netplayGameVersion())))
		return
	}
	if err := validateNetMessageForClient(challenge); err != nil {
		s.setErr(fmt.Errorf("netplay handshake challenge: %w", err))
		return
	}

	proof := ""
	if s.clientSessionKey != "" {
		var err error
		proof, err = onlineSessionProof(s.clientSessionKey, challenge.Challenge)
		if err != nil {
			s.setErr(fmt.Errorf("netplay handshake proof: %w", err))
			return
		}
	}
	if err := link.Write(netWireMessage{Type: netMessageHello, Protocol: netplayProtocolVersion, GameVersion: netplayGameVersion(), AuthProof: proof}, maxClientWireFrame); err != nil {
		s.setErr(fmt.Errorf("netplay handshake response: %w", err))
		return
	}

	var hello netWireMessage
	if err := link.ReadLenient(maxServerWireFrame, &hello); err != nil {
		s.setErr(fmt.Errorf("netplay handshake acceptance: %w", err))
		return
	}
	if hello.Type == netMessageReject {
		s.setErr(fmt.Errorf("host rejected the connection: %s", sanitizeNetReason(hello.Reason)))
		return
	}
	if hello.Type != netMessageHello {
		s.setErr(errors.New("netplay handshake acceptance: host rejected the connection"))
		return
	}
	if err := validateNetMessageForClient(hello); err != nil || hello.PlayerID < 2 || hello.PlayerID > maxOnlinePlayers {
		if err == nil {
			err = errors.New("invalid player assignment from host")
		}
		s.setErr(fmt.Errorf("netplay handshake assignment: %w", err))
		return
	}
	s.assignedPlayerID.Store(int32(hello.PlayerID))
	s.ready.Store(true)

	stopWriter := make(chan struct{})
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		s.clientWriteLoop(link, stopWriter)
	}()
	go s.clientPingLoop(stopWriter)
	s.clientReadLoop(link)
	close(stopWriter)
	_ = link.Close()
	<-writerDone
}

func (s *netplaySession) clientPingLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(netPingInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.outbox.Push(netWireMessage{Type: netMessagePing, PingTime: time.Now().UnixNano()})
		case <-stop:
			return
		case <-s.done:
			return
		}
	}
}

// maxInvalidNetSnapshots: a client drops an individual invalid snapshot and
// keeps the connection; only a host that keeps sending them is disconnected.
const maxInvalidNetSnapshots = 30

func (s *netplaySession) clientReadLoop(link *netplayLink) {
	stream := newNetClientStream()
	invalid := 0
	for !s.closed.Load() {
		var msg netWireMessage
		_ = link.SetReadDeadline(time.Now().Add(netIdleTimeout()))
		if err := link.Read(maxServerWireFrame, &msg); err != nil {
			if isNetTimeout(err) {
				s.setErr(fmt.Errorf("lost connection to host: %w", errNetPeerTimeout))
			} else {
				s.setErr(fmt.Errorf("receive host message: %w", err))
			}
			return
		}
		if err := validateNetMessageForClient(msg); err != nil {
			s.setErr(err)
			s.ready.Store(false)
			return
		}
		switch msg.Type {
		case netMessageReject:
			s.setErr(fmt.Errorf("host closed the connection: %s", sanitizeNetReason(msg.Reason)))
			return
		case netMessagePing:
			s.outbox.Push(netWireMessage{Type: netMessagePong, PingTime: msg.PingTime})
		case netMessagePong:
			recordNetRTT(&s.rttMillis, msg.PingTime)
		case netMessageState:
			decoded, err := stream.decode(msg.Payload, msg.PayloadLen)
			if err != nil {
				// The compression stream cannot resynchronise after a bad chunk.
				s.setErr(fmt.Errorf("host snapshot: %w", err))
				return
			}
			s.outbox.Push(netWireMessage{Type: netMessageAck, Tick: decoded.tick})
			s.queueClientSFX(decoded.sfx)
			if trimmed := trimNetSnapshotLists(&decoded.state); len(trimmed) > 0 {
				networkDiagnostic("CLIENT trimmed oversized host snapshot tick=%d lists=%v", decoded.tick, trimmed)
			}
			if err := validateNetSnapshot(decoded.state); err != nil {
				invalid++
				networkDiagnostic("CLIENT dropped invalid host snapshot tick=%d count=%d err=%v", decoded.tick, invalid, err)
				if invalid >= maxInvalidNetSnapshots {
					s.setErr(fmt.Errorf("host keeps sending invalid snapshots: %w", err))
					return
				}
				continue
			}
			invalid = 0
			s.playout.push(decoded.tick, decoded.state)
			s.stateMu.Lock()
			s.state = decoded.state
			s.stateMu.Unlock()
			s.hasState.Store(true)
		default:
			s.setErr(errors.New("unexpected host message"))
			s.ready.Store(false)
			return
		}
	}
}

func (s *netplaySession) queueClientSFX(events []netSFXEvent) {
	if len(events) == 0 {
		return
	}
	s.sfxMu.Lock()
	defer s.sfxMu.Unlock()
	for _, ev := range events {
		if ev.Seq <= s.sfxLastSeq {
			continue
		}
		s.sfxLastSeq = ev.Seq
		s.sfxPending = append(s.sfxPending, ev)
	}
	if len(s.sfxPending) > netRecentSFXLimit {
		s.sfxPending = s.sfxPending[len(s.sfxPending)-netRecentSFXLimit:]
	}
}

func (s *netplaySession) drainClientSFX() []netSFXEvent {
	if s == nil {
		return nil
	}
	s.sfxMu.Lock()
	defer s.sfxMu.Unlock()
	out := s.sfxPending
	s.sfxPending = nil
	return out
}

func (s *netplaySession) clientWriteLoop(link *netplayLink, stop <-chan struct{}) {
	for {
		msg, ok := s.outbox.Next(stop, s.done)
		if !ok {
			return
		}
		msg.Protocol = netplayProtocolVersion
		msg.PlayerID = s.assignedPlayer()
		if err := link.Write(msg, maxClientWireFrame); err != nil {
			s.setErr(fmt.Errorf("send client message: %w", err))
			_ = link.Close()
			return
		}
	}
}

func (s *netplaySession) setErr(err error) {
	if err == nil || s.closed.Load() {
		return
	}
	s.lastErrMu.Lock()
	s.lastErr = err
	s.lastErrMu.Unlock()
}

func (s *netplaySession) LastError() error {
	if s == nil {
		return nil
	}
	s.lastErrMu.RLock()
	defer s.lastErrMu.RUnlock()
	return s.lastErr
}

func (s *netplaySession) connectedCount() int {
	if s == nil {
		return 0
	}
	if s.mode == netplayClient {
		if s.ready.Load() {
			return 1
		}
		return 0
	}
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()
	return len(s.peers)
}

func (s *netplaySession) connected() bool {
	return s != nil && s.connectedCount() > 0
}

func (s *netplaySession) humanCapacity() int {
	if s == nil {
		return 1
	}
	if s.mode != netplayHost {
		return maxOnlinePlayers
	}
	capacity := 1
	for id := 2; id <= maxOnlinePlayers; id++ {
		if s.allowedRemote[id].Load() {
			capacity++
		}
	}
	if capacity < 1 {
		capacity = 1
	}
	return capacity
}

func (s *netplaySession) assignedPlayer() int {
	if s == nil || s.mode != netplayClient {
		return 0
	}
	return int(s.assignedPlayerID.Load())
}

func (s *netplaySession) connectedPlayerIDs() []int {
	if s == nil || s.mode != netplayHost {
		return nil
	}
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()
	ids := make([]int, 0, len(s.peers))
	for id := 2; id <= maxOnlinePlayers; id++ {
		if _, ok := s.peers[id]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func (s *netplaySession) setAllowedRemotePlayers(allowed [5]bool) {
	if s == nil || s.mode != netplayHost {
		return
	}
	for id := 2; id <= maxOnlinePlayers; id++ {
		s.allowedRemote[id].Store(allowed[id])
	}
	var closePeers []*netplayPeer
	s.peersMu.RLock()
	for id, peer := range s.peers {
		if id >= 2 && id <= maxOnlinePlayers && !allowed[id] {
			closePeers = append(closePeers, peer)
		}
	}
	s.peersMu.RUnlock()
	for _, peer := range closePeers {
		peer.close()
	}
}

// takeInputFor consumes one host tick of a remote player's input.
func (s *netplaySession) takeInputFor(playerID int) PlayerInputState {
	if s == nil || s.mode != netplayHost {
		return PlayerInputState{}
	}
	s.peersMu.RLock()
	peer := s.peers[playerID]
	s.peersMu.RUnlock()
	return peer.takeInput(time.Now())
}

// peerRTTMillis reports the measured round trip to a remote player.
func (s *netplaySession) peerRTTMillis(playerID int) (int64, bool) {
	if s == nil || s.mode != netplayHost {
		return 0, false
	}
	s.peersMu.RLock()
	peer := s.peers[playerID]
	s.peersMu.RUnlock()
	if peer == nil {
		return 0, false
	}
	return loadNetRTT(&peer.rttMillis)
}

// hostRTTMillis reports the client's measured round trip to the host.
func (s *netplaySession) hostRTTMillis() (int64, bool) {
	if s == nil || s.mode != netplayClient {
		return 0, false
	}
	return loadNetRTT(&s.rttMillis)
}

func (s *netplaySession) queueClientMessage(msg netWireMessage) {
	if s == nil || s.closed.Load() || s.mode != netplayClient || !s.connected() || s.outbox == nil {
		return
	}
	s.outbox.Push(msg)
}

// queueInput records one client frame of input. Call it exactly once per
// client update during gameplay: the host replays frames one per tick.
func (s *netplaySession) queueInput(input PlayerInputState) {
	if s == nil || s.closed.Load() || s.mode != netplayClient || !s.connected() || s.outbox == nil {
		return
	}
	seq, frames := s.inputHistory.record(input)
	s.outbox.Push(netWireMessage{Type: netMessageInput, InputSeq: seq, InputFrames: frames})
}

func (s *netplaySession) queuePlayerConfig(cfg CustomPlayerConfig) {
	cfg.Type = 1
	s.queueClientMessage(netWireMessage{Type: netMessagePlayerConfig, Config: cfg})
}

func (s *netplaySession) queuePostGameReady(ready bool) {
	s.queueClientMessage(netWireMessage{Type: netMessagePostGameReady, Ready: ready})
}

func (s *netplaySession) postGameReadyFor(playerID int) bool {
	if s == nil || s.mode != netplayHost {
		return false
	}
	s.peersMu.RLock()
	peer := s.peers[playerID]
	s.peersMu.RUnlock()
	return peer != nil && peer.postGameReady.Load()
}

func (s *netplaySession) resetPostGameReady() {
	if s == nil || s.mode != netplayHost {
		return
	}
	s.peersMu.RLock()
	peers := make([]*netplayPeer, 0, len(s.peers))
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	s.peersMu.RUnlock()
	for _, peer := range peers {
		peer.postGameReady.Store(false)
	}
}

func (s *netplaySession) latestConfigFor(playerID int) (CustomPlayerConfig, bool) {
	if s == nil || s.mode != netplayHost {
		return CustomPlayerConfig{}, false
	}
	s.peersMu.RLock()
	peer := s.peers[playerID]
	s.peersMu.RUnlock()
	if peer == nil || !peer.hasConfig.Load() {
		return CustomPlayerConfig{}, false
	}
	peer.configMu.RLock()
	defer peer.configMu.RUnlock()
	return peer.config, true
}

// queueState encodes the snapshot once (on the caller's goroutine, so no live
// game state is shared with the writers) and offers the bytes to every peer.
// state must be a copy owned by the caller (makeNetSnapshot builds one).
func (s *netplaySession) queueState(state netSnapshot) {
	if s == nil || s.closed.Load() || s.mode != netplayHost {
		return
	}
	s.hostEncMu.Lock()
	frame := s.hostEnc.encode(state)
	s.hostEncMu.Unlock()
	s.peersMu.RLock()
	peers := make([]*netplayPeer, 0, len(s.peers))
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	s.peersMu.RUnlock()
	for _, peer := range peers {
		peer.queueState(frame)
	}
}

// nextPlayoutState advances the client's playout buffer by one tick and
// returns the (possibly interpolated) snapshot to render.
func (s *netplaySession) nextPlayoutState() (netSnapshot, bool) {
	if s == nil || !s.hasState.Load() {
		return netSnapshot{}, false
	}
	return s.playout.next()
}

func (s *netplaySession) Close() error {
	if s == nil || s.closed.Swap(true) {
		return nil
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	if s.done != nil {
		close(s.done)
	}
	s.peersMu.RLock()
	peers := make([]*netplayPeer, 0, len(s.peers))
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	s.peersMu.RUnlock()
	for _, peer := range peers {
		peer.close()
	}
	s.connMu.Lock()
	if s.link != nil {
		_ = s.link.Close()
	} else if s.conn != nil {
		_ = s.conn.Close()
	}
	s.connMu.Unlock()
	return nil
}

// StartNetHost starts an Internet+LAN OPEN authoritative host for compatibility
// with CLI and older tests.
func (g *Game) StartNetHost(addr string) error {
	return g.StartNetHostConfig(addr, onlineSessionScopeInternet, onlineSessionOpen)
}

// StartNetHostMode is kept for compatibility. OPEN and INVITE both use the
// Internet+LAN scope; new UI code should call StartNetHostConfig so network
// reach and access policy remain independent.
func (g *Game) StartNetHostMode(addr string, visibility int) error {
	return g.StartNetHostConfig(addr, onlineSessionScopeInternet, visibility)
}

// StartNetHostConfig creates one of four combinations:
// LAN+OPEN, LAN+INVITE, INTERNET+OPEN, INTERNET+INVITE.
// OPEN controls browser visibility; scope controls whether Internet/NAT/DHT
// services are started. INVITE sessions stay off browser lists in either scope.
func (g *Game) StartNetHostConfig(addr string, scope, visibility int) error {
	if g == nil {
		return errors.New("nil game")
	}
	if scope != onlineSessionScopeLAN && scope != onlineSessionScopeInternet {
		return errors.New("invalid online session scope")
	}
	if visibility != onlineSessionOpen && visibility != onlineSessionInviteOnly {
		return errors.New("invalid online session visibility")
	}
	_ = g.CloseNetplay()
	key, err := newOnlineSessionKey()
	if err != nil {
		return fmt.Errorf("create session key: %w", err)
	}
	s, err := newHostNetplay(addr)
	if err != nil {
		return err
	}
	g.netplay = s
	shortCode := ""
	authKey := key
	if visibility == onlineSessionInviteOnly {
		shortCode = decentralizedOpenCode(key)
		authKey = decentralizedCodeSessionKey(shortCode)
	}
	networkDiagnostic("HOST started listener=%s scope=%d visibility=%d", s.listener.Addr(), scope, visibility)
	g.onlineSessionScope = scope
	g.onlineSessionVisibility = visibility
	g.onlineSessionKey = authKey
	s.inviteOnly = visibility == onlineSessionInviteOnly
	s.expectedSessionKey = authKey
	s.shortJoinCode = shortCode
	s.publicSessionID = publicOnlineSessionID(authKey)
	s.publicSessionName = sanitizeRemotePlayerName(g.multiplayerSessionName)
	if s.publicSessionName == "" {
		s.publicSessionName = "Gun Mayhem Game"
	}
	var allowed [5]bool
	for id := 2; id <= maxOnlinePlayers; id++ {
		allowed[id] = g.customPlayers[id-1].Type == 1
	}
	s.setAllowedRemotePlayers(allowed)
	// Seed browser metadata before LAN/DHT responders become visible so even a
	// client that discovers the host immediately can preview the configured match.
	g.updateNetplayPublicProbeInfo()

	if tcpAddr, ok := s.listener.Addr().(*net.TCPAddr); ok && tcpAddr.Port > 0 {
		// OPEN sessions are discoverable on LAN regardless of whether Internet
		// reach is enabled. INVITE sessions are deliberately absent from lists.
		if visibility == onlineSessionOpen {
			if localAdvertiser, localErr := startLocalServerAdvertiser(tcpAddr.Port, s); localErr == nil {
				g.localServerAdvertiser = localAdvertiser
			} else {
				networkDiagnostic("HOST local advertiser unavailable err=%v", localErr)
			}
		}
		// INVITE sessions also listen for a LAN discovery packet, but answer
		// only when it carries their exact short code.
		if responder, lanErr := startLANDiscoveryResponder(tcpAddr.Port, s); lanErr == nil {
			g.lanDiscoveryResponder = responder
		} else {
			networkDiagnostic("HOST LAN discovery unavailable udp=%d err=%v", lanDiscoveryPort, lanErr)
		}

		if scope == onlineSessionScopeInternet && shouldUseCloudflareInternetTunnel(addr) {
			// Internet gameplay goes through a temporary Cloudflare Quick Tunnel.
			// LAN remains direct and unchanged. OPEN still uses Mainline DHT only
			// as a decentralized directory that points clients at this tunnel.
			tunnel, tunnelErr := startCloudflareQuickTunnel(s)
			if tunnelErr != nil {
				_ = g.CloseNetplay()
				return fmt.Errorf("start Internet tunnel: %w", tunnelErr)
			}
			g.cloudflareTunnel = tunnel
			// Do not block the game thread while Cloudflare allocates and health-checks
			// a public URL. OPEN discovery waits for this monitor before publishing.
			tunnel.monitorStartup()
		}
	}

	// OPEN Internet sessions publish only discovery metadata to Mainline DHT.
	// The discovered UDP endpoint answers the tiny metadata probe with the
	// temporary WSS tunnel address; gameplay itself never uses this QUIC path.
	// INVITE ONLY needs no public DHT publication because its code contains the
	// temporary tunnel hostname and authentication secret directly.
	if scope == onlineSessionScopeInternet && (visibility == onlineSessionOpen || g.cloudflareTunnel == nil) {
		if p2p, p2pErr := startP2PQUICHostWithTunnel(s, visibility, authKey, g.cloudflareTunnel); p2pErr == nil {
			g.p2pHost = p2p
		} else if g.cloudflareTunnel != nil {
			_ = g.CloseNetplay()
			return fmt.Errorf("start OPEN session discovery: %w", p2pErr)
		}
	}
	return nil
}

// StartNetClient connects this process to the next free Player 2..4 slot. The
// client is a rendering mirror of the host and does not run authoritative combat simulation.
func waitForClientHandshake(s *netplaySession, timeout time.Duration) error {
	if s == nil {
		return errors.New("nil client session")
	}
	if timeout <= 0 {
		timeout = netHandshakeTimeout + time.Second
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if s.ready.Load() && s.assignedPlayer() >= 2 {
			return nil
		}
		if err := s.LastError(); err != nil {
			return err
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			return errors.New("netplay handshake timed out")
		}
	}
}

func (g *Game) StartNetClient(addr string) error {
	if g == nil {
		return errors.New("nil game")
	}
	_ = g.CloseNetplay()
	networkDiagnostic("JOIN path=manual dial start address=%s", addr)
	s, err := newClientNetplay(addr)
	if err != nil {
		networkDiagnostic("JOIN path=manual dial failed address=%s err=%v", addr, err)
		return err
	}
	if err := waitForClientHandshake(s, netHandshakeTimeout+time.Second); err != nil {
		networkDiagnostic("JOIN path=manual handshake failed address=%s err=%v", addr, err)
		_ = s.Close()
		return err
	}
	g.netplay = s
	networkDiagnostic("JOIN path=manual ready address=%s player=%d", addr, s.assignedPlayer())
	return nil
}

func (g *Game) StartNetClientServer(server multiplayerServerInfo) (finalErr error) {
	if g == nil {
		return errors.New("nil game")
	}
	networkDiagnostic("JOIN start name=%q source=%q transport=%q address=%s", server.Name, server.Sources, server.Transport, server.Address)
	defer func() {
		player := 0
		if g.netplay != nil {
			player = g.netplay.assignedPlayer()
		}
		networkDiagnostic("JOIN finish address=%s transport=%s player=%d err=%v", server.Address, server.Transport, player, finalErr)
	}()
	_ = g.CloseNetplay()

	if strings.EqualFold(strings.TrimSpace(server.Transport), "cloudflare") || strings.TrimSpace(server.TunnelAddress) != "" {
		address := strings.TrimSpace(server.TunnelAddress)
		if address == "" {
			address = strings.TrimSpace(server.Address)
		}
		networkDiagnostic("JOIN path=cloudflare selected tunnel=%s advertised=%s", address, server.Address)
		s, err := newClientNetplayCloudflare(address, "", 8*time.Second)
		if err != nil {
			networkDiagnostic("JOIN path=cloudflare dial failed tunnel=%s err=%v", address, err)
			return err
		}
		if err := waitForClientHandshake(s, netHandshakeTimeout+time.Second); err != nil {
			networkDiagnostic("JOIN path=cloudflare handshake failed tunnel=%s err=%v", address, err)
			_ = s.Close()
			return err
		}
		g.netplay = s
		networkDiagnostic("JOIN path=cloudflare ready tunnel=%s player=%d", address, s.assignedPlayer())
		return nil
	}

	if strings.EqualFold(strings.TrimSpace(server.Transport), "quic") && strings.TrimSpace(server.Address) != "" {
		// Browser metadata can advertise a native IPv6/public IPv4/UPnP TCP
		// address. Prefer that direct host path before NAT hole punching.
		if direct := strings.TrimSpace(server.DirectAddress); direct != "" {
			if s, err := tryDirectClientSession(direct, "", 1800*time.Millisecond); err == nil {
				g.netplay = s
				networkDiagnostic("JOIN path=quic-direct ready address=%s player=%d", direct, s.assignedPlayer())
				return nil
			} else {
				networkDiagnostic("JOIN direct advertised path failed address=%s err=%v", direct, err)
			}
		}
		networkDiagnostic("JOIN path=quic dial start address=%s", server.Address)
		conn, err := dialP2PQUIC(server.Address, "", p2pDialTimeout)
		if err != nil {
			networkDiagnostic("JOIN path=quic dial failed address=%s err=%v", server.Address, err)
			return err
		}
		s, err := newClientNetplayFromConn(conn, "P2P "+server.Address)
		if err != nil {
			networkDiagnostic("JOIN path=quic session failed address=%s err=%v", server.Address, err)
			_ = conn.Close()
			return err
		}
		if err := waitForClientHandshake(s, 4*time.Second); err != nil {
			networkDiagnostic("JOIN path=quic handshake failed address=%s err=%v", server.Address, err)
			_ = s.Close()
			return err
		}
		g.netplay = s
		networkDiagnostic("JOIN path=quic ready address=%s player=%d", server.Address, s.assignedPlayer())
		return nil
	}

	var directErr error
	if strings.TrimSpace(server.Address) != "" {
		directTimeout := 5 * time.Second
		networkDiagnostic("JOIN path=direct dial start address=%s timeout=%s", server.Address, directTimeout)
		if s, err := newClientNetplayWithTimeout(server.Address, directTimeout); err == nil {
			if handshakeErr := waitForClientHandshake(s, netHandshakeTimeout+time.Second); handshakeErr == nil {
				g.netplay = s
				networkDiagnostic("JOIN path=direct ready address=%s player=%d", server.Address, s.assignedPlayer())
				return nil
			} else {
				networkDiagnostic("JOIN path=direct handshake failed address=%s err=%v", server.Address, handshakeErr)
				directErr = handshakeErr
				_ = s.Close()
			}
		} else {
			networkDiagnostic("JOIN path=direct dial failed address=%s err=%v", server.Address, err)
			directErr = err
		}
	}
	if directErr != nil {
		networkDiagnostic("JOIN all paths failed address=%s directErr=%v", server.Address, directErr)
		return directErr
	}
	return errors.New("server has no usable address")
}

func (g *Game) CloseNetplay() error {
	if g == nil {
		return nil
	}
	g.cancelPendingNetJoin()
	// These caches belong to one connection/session, not to the user's saved
	// character. Rejoining in the same process must resend unchanged config
	// and accept a new host's sound sequence starting at one.
	g.onlineHasSentConfig = false
	g.onlineLastSentConfig = CustomPlayerConfig{}
	g.netLastSFXSeq = 0
	g.netSFXSeq = 0
	g.netClientSFXPending = nil
	g.netSFXPending = nil
	if g.localServerAdvertiser != nil {
		_ = g.localServerAdvertiser.Close()
		g.localServerAdvertiser = nil
	}
	if g.lanDiscoveryResponder != nil {
		_ = g.lanDiscoveryResponder.Close()
		g.lanDiscoveryResponder = nil
	}
	if g.p2pHost != nil {
		_ = g.p2pHost.Close()
		g.p2pHost = nil
	}
	if g.cloudflareTunnel != nil {
		_ = g.cloudflareTunnel.Close()
		g.cloudflareTunnel = nil
	}
	if g.netplay == nil {
		return nil
	}
	err := g.netplay.Close()
	g.netplay = nil
	g.onlineSessionKey = ""
	return err
}

func (g *Game) NetplayStatus() string {
	if g == nil || g.netplay == nil {
		return "offline"
	}
	s := g.netplay
	switch s.mode {
	case netplayHost:
		count := s.connectedCount()
		if count > 0 {
			return fmt.Sprintf("hosting - players %d/%d", count+1, maxOnlinePlayers)
		}
		return "hosting - waiting for players"
	case netplayClient:
		if s.connected() {
			if s.hasState.Load() {
				if rtt, ok := s.hostRTTMillis(); ok {
					return fmt.Sprintf("connected to %s - ping %d ms", s.remoteAddr, rtt)
				}
				return "connected to " + s.remoteAddr
			}
			return "connected - waiting for host game"
		}
		return "disconnected"
	default:
		return "offline"
	}
}

func capturePlayerInput(c Controls) PlayerInputState {
	return PlayerInputState{
		Left:    sourceKeyDown(c.Left),
		Right:   sourceKeyDown(c.Right),
		Up:      sourceKeyDown(c.Up),
		Down:    sourceKeyDown(c.Down),
		Shoot:   sourceKeyDown(c.Shoot),
		Grenade: sourceKeyDown(c.Grenade),
		Ability: sourceKeyDown(c.Ability),
	}
}

func (g *Game) applyHostRemoteInput() {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost {
		return
	}
	connected := [5]bool{}
	for _, id := range g.netplay.connectedPlayerIDs() {
		if id >= 2 && id <= maxOnlinePlayers {
			connected[id] = true
		}
	}
	for _, p := range g.players {
		if p == nil || p.ID < 2 || p.ID > maxOnlinePlayers || p.IsDouble {
			continue
		}
		cfg := g.customPlayers[p.ID-1]
		if cfg.Type == 2 {
			// AI bots are authoritative on the host. Never turn them into a
			// network-controlled player merely because they use slot IDs 2..4.
			p.RemoteControlled = false
			p.RemoteInput = PlayerInputState{}
			p.AI = true
			continue
		}
		if cfg.Type == 1 && connected[p.ID] {
			p.RemoteControlled = true
			p.RemoteInput = g.netplay.takeInputFor(p.ID)
			p.AI = false
			continue
		}
		// Waiting/empty online slots must not accidentally consume stale input.
		p.RemoteControlled = false
		p.RemoteInput = PlayerInputState{}
		p.AI = false
	}
}

func (g *Game) makeNetSnapshot() netSnapshot {
	onlineCount := g.onlinePlayerCount
	onlineSlots := g.onlinePlayerSlots
	if g.netplay != nil && g.netplay.mode == netplayHost {
		onlineSlots = [4]bool{true, false, false, false}
		for _, id := range g.netplay.connectedPlayerIDs() {
			if id >= 1 && id <= len(onlineSlots) {
				onlineSlots[id-1] = true
			}
		}
		onlineCount = 1 + g.netplay.connectedCount()
	}
	state := netSnapshot{
		Screen:                         int(g.screen),
		Arena:                          g.arena,
		Bullets:                        append([]Bullet(nil), g.bullets...),
		InstagibTrails:                 append([]InstagibTrailEffect(nil), g.instagibTrails...),
		PlayerTrails:                   append([]PlayerTrailEffect(nil), g.playerTrails...),
		Shells:                         append([]Shell(nil), g.shells...),
		Flashes:                        append([]MuzzleFlash(nil), g.flashes...),
		Grenades:                       append([]Grenade(nil), g.grenades...),
		FirePatches:                    append([]FirePatch(nil), g.firePatches...),
		AreaEffects:                    append([]AreaEffect(nil), g.areaEffects...),
		Portals:                        append([]Portal(nil), g.portals...),
		DynamiteFX:                     append([]DynamiteEffect(nil), g.dynamiteFX...),
		BlastFX:                        append([]BlastEffect(nil), g.blastFX...),
		JetThrustFX:                    append([]JetThrustEffect(nil), g.jetThrustFX...),
		DropPackFX:                     append([]DropPackEffect(nil), g.dropPackFX...),
		Explosions:                     append([]Explosion(nil), g.explosions...),
		Killfeeds:                      append([]KillFeedEntry(nil), g.killfeeds...),
		Crates:                         append([]Crate(nil), g.crates...),
		Powerups:                       append([]Powerup(nil), g.powerups...),
		PowerupNameFX:                  append([]PowerupNameEffect(nil), g.powerupNameFX...),
		LifeBlingFX:                    append([]LifeBlingEffect(nil), g.lifeBlingFX...),
		CampaignEnemies:                g.campaignEnemySnapshot(),
		HazardMarkers:                  append([]HazardMarker(nil), g.extraHazardMarkers...),
		SFXEvents:                      append([]netSFXEvent(nil), g.netSFXPending...),
		MapFXFrame:                     g.mapFXFrame,
		CameraX:                        g.cameraX,
		CameraY:                        g.cameraY,
		GameMode:                       g.GameMode,
		TotalLives:                     g.TotalLives,
		TeamGame:                       g.TeamGame,
		CrateON:                        g.CrateON,
		PowerON:                        g.PowerON,
		GameWin:                        g.GameWin,
		OnlineMatchSetup:               g.onlineMatchSetup,
		CustomPage:                     g.customPage,
		CustomMenuX:                    g.customMenuX,
		CustomMode:                     g.customMode,
		CustomMap:                      g.customMap,
		CustomMapSetGM2:                g.customMapSetGM2,
		CustomLives:                    g.customLives,
		CustomSeriesRounds:             g.customSeriesRounds,
		CustomSeriesRound:              g.customSeriesRound,
		CustomSeriesStats:              g.customSeriesStats,
		CustomSeriesMaps:               g.customSeriesMaps,
		CustomSeriesPlaylistConfigured: g.customSeriesPlaylistConfigured,
		ChaosEnabled:                   g.customChaosEnabled,
		ChaosCurrentEffect:             g.chaosCurrentEffect,
		ChaosFrame:                     g.chaosFrame,
		ChaosBannerFrame:               g.chaosBannerFrame,
		ChaosIntervalSeconds:           g.customChaosInterval,
		OnlinePlayerCount:              onlineCount,
		OnlinePlayerSlots:              onlineSlots,
		OnlineCustomPlayers:            g.customPlayers,
		OnlineMapRotation:              g.onlineMapRotation,
		OnlineWeaponMode:               g.onlineWeaponMode,
		OnlineOnlyWeapon:               g.onlineOnlyWeapon,
		OnlinePerksEnabled:             g.onlinePerksEnabled,
		OnlinePowerupsEnabled:          g.onlinePowerupsEnabled,
		OnlineBlockedWeapons:           g.onlineBlockedWeapons,
		OnlineBlockedPerks:             g.onlineBlockedPerks,
		OnlineBlockedAbilities:         g.onlineBlockedAbilities,
		PostGameReady:                  g.onlinePostGameReadySnapshot(),
		PostGameReadyCountdown:         g.postGameReadyCountdown,
		Paused:                         g.paused,
		PausePressed:                   g.pausePressed,
		FadeActive:                     g.fadeActive,
		FadeFrame:                      g.fadeFrame,
		FadeTarget:                     int(g.fadeTarget),
		FadePurpose:                    int(g.fadePurpose),
		MatchWinCountdown:              g.matchWinCountdown,
		SoloWinFrame:                   g.soloWinFrame,
		TeamGameWin:                    g.teamGameWin,
		TeamWinner:                     g.teamWinner,
		WinnerPlayerID:                 g.winnerPlayerID,
		WinnerAnimFrame:                g.winnerAnimFrame,
		TeamWinAnimFrame:               g.teamWinAnimFrame,
		CampaignLoseFrame:              g.campaignLoseFrame,
		ZombieWaveFrame:                g.zombieWaveFrame,
		HUDLastLifeFrame:               g.hudLastLifeFrame,
		HUDLastLifePlaying:             g.hudLastLifePlaying,
		HUDLastLevel:                   g.hudLastLevel,
		CampaignMode:                   g.campaignMode,
		CampaignLevel:                  g.campaignLevel,
		Gototest:                       g.gototest,
		TestGunNumber:                  g.testGunNumber,
		TestGunDisabled:                g.testGunDisabled,
		TestGunRespawn:                 g.testGunRespawn,
		TestGunFrame:                   g.testGunFrame,
	}
	state.Players = make([]Player, 0, len(g.players))
	for _, p := range g.players {
		if p != nil {
			state.Players = append(state.Players, *p)
		}
	}
	return state
}

func (g *Game) queueNetHostSnapshot() {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost || !g.netplay.connected() {
		return
	}
	g.netplay.queueState(g.makeNetSnapshot())
	g.netSFXPending = g.netSFXPending[:0]
}

func (g *Game) applyNetSnapshot(state netSnapshot) {
	oldScreen := g.screen
	// A connected client owns its character slot. Preserve that local edit
	// while applying the host snapshot; otherwise one older network tick can
	// overwrite a click and immediately send the stale value back to the host.
	localConfigID := 0
	localConfig := CustomPlayerConfig{}
	if g.netplay != nil && g.netplay.mode == netplayClient {
		if id := g.netplay.assignedPlayer(); id >= 2 && id <= maxOnlinePlayers {
			localConfigID = id
			localConfig = g.customPlayers[id-1]
			localConfig.Type = 1
		}
	}
	g.screen = gameScreen(state.Screen)
	g.arena = state.Arena
	g.maps[state.Arena.Number] = state.Arena
	g.players = make([]*Player, len(state.Players))
	for i := range state.Players {
		p := state.Players[i]
		g.players[i] = &p
	}
	g.bullets = append(g.bullets[:0], state.Bullets...)
	g.instagibTrails = append(g.instagibTrails[:0], state.InstagibTrails...)
	g.playerTrails = append(g.playerTrails[:0], state.PlayerTrails...)
	g.shells = append(g.shells[:0], state.Shells...)
	g.flashes = append(g.flashes[:0], state.Flashes...)
	g.grenades = append(g.grenades[:0], state.Grenades...)
	g.firePatches = append(g.firePatches[:0], state.FirePatches...)
	g.areaEffects = append(g.areaEffects[:0], state.AreaEffects...)
	g.portals = append(g.portals[:0], state.Portals...)
	g.dynamiteFX = append(g.dynamiteFX[:0], state.DynamiteFX...)
	g.blastFX = append(g.blastFX[:0], state.BlastFX...)
	g.jetThrustFX = append(g.jetThrustFX[:0], state.JetThrustFX...)
	g.dropPackFX = append(g.dropPackFX[:0], state.DropPackFX...)
	g.explosions = append(g.explosions[:0], state.Explosions...)
	g.killfeeds = append(g.killfeeds[:0], state.Killfeeds...)
	g.crates = append(g.crates[:0], state.Crates...)
	g.powerups = append(g.powerups[:0], state.Powerups...)
	g.powerupNameFX = append(g.powerupNameFX[:0], state.PowerupNameFX...)
	g.lifeBlingFX = append(g.lifeBlingFX[:0], state.LifeBlingFX...)
	g.applyCampaignEnemySnapshot(state.CampaignEnemies)
	g.extraHazardMarkers = append(g.extraHazardMarkers[:0], state.HazardMarkers...)
	g.mapFXFrame = state.MapFXFrame
	g.cameraX = state.CameraX
	g.cameraY = state.CameraY
	g.GameMode = state.GameMode
	g.TotalLives = state.TotalLives
	g.TeamGame = state.TeamGame
	g.CrateON = state.CrateON
	g.PowerON = state.PowerON
	g.GameWin = state.GameWin
	g.onlineMatchSetup = state.OnlineMatchSetup
	if state.OnlineMatchSetup {
		g.customOnlineMode = true
	}
	if g.netplay != nil && g.netplay.mode == netplayClient && state.OnlineMatchSetup && state.Screen == int(screenCustomGame) {
		// Clients stay on the final online wizard step (slots/own character).
		// Host-owned mode/maps are previewed in the browser and mirrored in state,
		// but the client should not be bounced through the host's setup pages.
		g.customPage = customPagePlayers
		g.customMenuX = 0
	} else {
		g.customPage = state.CustomPage
		g.customMenuX = state.CustomMenuX
	}
	g.customMode = state.CustomMode
	g.customMap = state.CustomMap
	g.customMapSetGM2 = state.CustomMapSetGM2
	g.customLives = state.CustomLives
	g.customSeriesRounds = clampCustomSeriesRounds(state.CustomSeriesRounds)
	g.customSeriesRound = state.CustomSeriesRound
	g.customSeriesStats = state.CustomSeriesStats
	g.customSeriesMaps = state.CustomSeriesMaps
	g.customSeriesPlaylistConfigured = state.CustomSeriesPlaylistConfigured
	g.ensureCustomPlaylistState()
	g.customChaosEnabled = state.ChaosEnabled
	g.chaosCurrentEffect = state.ChaosCurrentEffect
	g.chaosFrame = state.ChaosFrame
	g.chaosBannerFrame = state.ChaosBannerFrame
	g.customChaosInterval = state.ChaosIntervalSeconds
	g.onlinePlayerCount = state.OnlinePlayerCount
	g.onlinePlayerSlots = state.OnlinePlayerSlots
	g.customPlayers = state.OnlineCustomPlayers
	if localConfigID != 0 {
		g.customPlayers[localConfigID-1] = localConfig
	}
	g.onlineMapRotation = state.OnlineMapRotation
	g.onlineWeaponMode = state.OnlineWeaponMode
	g.onlineOnlyWeapon = state.OnlineOnlyWeapon
	g.onlinePerksEnabled = state.OnlinePerksEnabled
	g.onlinePowerupsEnabled = state.OnlinePowerupsEnabled
	g.onlineBlockedWeapons = state.OnlineBlockedWeapons
	g.onlineBlockedPerks = state.OnlineBlockedPerks
	g.onlineBlockedAbilities = state.OnlineBlockedAbilities
	g.postGameReady = state.PostGameReady
	g.postGameReadyCountdown = state.PostGameReadyCountdown
	g.paused = state.Paused
	g.pausePressed = state.PausePressed
	g.fadeActive = state.FadeActive
	g.fadeFrame = state.FadeFrame
	g.fadeTarget = gameScreen(state.FadeTarget)
	g.fadePurpose = fadePurpose(state.FadePurpose)
	g.matchWinCountdown = state.MatchWinCountdown
	g.soloWinFrame = state.SoloWinFrame
	g.teamGameWin = state.TeamGameWin
	g.teamWinner = state.TeamWinner
	g.winnerPlayerID = state.WinnerPlayerID
	g.winnerAnimFrame = state.WinnerAnimFrame
	g.teamWinAnimFrame = state.TeamWinAnimFrame
	g.campaignLoseFrame = state.CampaignLoseFrame
	g.zombieWaveFrame = state.ZombieWaveFrame
	g.hudLastLifeFrame = state.HUDLastLifeFrame
	g.hudLastLifePlaying = state.HUDLastLifePlaying
	g.hudLastLevel = state.HUDLastLevel
	g.campaignMode = state.CampaignMode
	g.campaignLevel = state.CampaignLevel
	g.gototest = state.Gototest
	g.testGunNumber = state.TestGunNumber
	g.testGunDisabled = state.TestGunDisabled
	g.testGunRespawn = state.TestGunRespawn
	g.testGunFrame = state.TestGunFrame
	for _, event := range state.SFXEvents {
		if event.Seq <= g.netLastSFXSeq {
			continue
		}
		g.netLastSFXSeq = event.Seq
		g.netClientSFXPending = append(g.netClientSFXPending, event)
	}
	if oldScreen != g.screen && g.audioStarted {
		g.syncSourceMusic()
	}
}

func (g *Game) flushNetClientSFX() {
	if g == nil || len(g.netClientSFXPending) == 0 || !g.audioStarted {
		return
	}
	if g.audio != nil {
		for _, event := range g.netClientSFXPending {
			volume := g.sourceSFXVolume(event.Loud)
			if volume > 0 {
				gain := event.Gain
				if gain <= 0 {
					gain = 1
				}
				g.audio.playSFXGain(event.Name, volume, gain)
			}
		}
	}
	g.netClientSFXPending = g.netClientSFXPending[:0]
}

func (g *Game) updateNetClient() error {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayClient {
		return nil
	}
	if !g.netplay.connected() && g.netplay.LastError() != nil {
		reason := g.netplay.LastError()
		_ = g.CloseNetplay()
		g.screen = screenCustomGame
		g.customOnlineMode = true
		g.onlineMatchSetup = false
		g.customNetworkOpen = true
		g.beginMultiplayerServerBrowser()
		g.setNetDisconnectNotice("HOST DISCONNECTED: " + friendlyNetError(reason))
		networkDiagnostic("CLIENT disconnected err=%v", reason)
		return nil
	}
	if state, ok := g.netplay.nextPlayoutState(); ok {
		g.applyNetSnapshot(state)
	}
	g.netClientSFXPending = append(g.netClientSFXPending, g.netplay.drainClientSFX()...)
	if g.screen == screenMultiplayerLobby {
		// Compatibility with snapshots from the retired standalone lobby.
		// Online waiting/setup now always uses the unified Custom Game SLOTS page.
		g.openOnlineCustomSlots("CONNECTED - CHOOSE YOUR CHARACTER")
	}
	if g.screen == screenCustomGame {
		g.customOnlineMode = true
		g.updateOnlineClientCustomGame()
		g.sendOnlineLocalConfigIfChanged()
		return nil
	}
	if g.screen == screenSeriesSummary {
		g.updateSeriesSummaryMenu()
		return nil
	}
	if g.screen == screenPostGame {
		g.updateOnlineClientPostGame()
		g.sendOnlineLocalConfigIfChanged()
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) { // Leave the remote session outside Custom Game.
		_ = g.CloseNetplay()
		g.screen = screenCustomGame
		g.customOnlineMode = true
		g.onlineMatchSetup = false
		g.customNetworkOpen = true
		g.multiplayerMessage = "DISCONNECTED"
		g.beginMultiplayerServerBrowser()
		return nil
	}
	g.sendOnlineLocalConfigIfChanged()
	if g.screen == screenGameplay {
		g.netplay.queueInput(netClientInputForFocus(capturePlayerInput(g.controlConfigs[0]), ebiten.IsFocused()))
	}
	return nil
}
