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
