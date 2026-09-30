package game

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
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
	netMessageKeepAlive     = 9
)

type netSFXEvent struct {
	Seq  uint64
	Name string
	Loud bool
	Gain float64
}

type netProbeInfo struct {
	ID                   string
	Name                 string
	Players              int
	MaxPlayers           int
	GameMode             int
	Lives                int
	Rounds               int
	Maps                 [customSeriesMaxRounds]int
	PlaylistConfigured   bool
	Crates               bool
	Pickups              bool
	RuleSummary          string
	ChaosEnabled         bool
	ChaosEvents          int
	ChaosIntervalSeconds int
	DirectAddress        string
	TunnelAddress        string
}

type netWireMessage struct {
	Type      int
	Protocol  int
	PlayerID  int
	Challenge string
	AuthProof string
	Probe     netProbeInfo
	Input     PlayerInputState
	Config    CustomPlayerConfig
	Ready     bool
	State     netSnapshot
}

// netSnapshot is deliberately a render-complete snapshot of the authoritative
// host. The client does not run combat physics; it renders the newest host
// snapshot and only sends its local input back to the host.
type netSnapshot struct {
	Screen int
	Arena  Map

	Players        []Player
	Bullets        []Bullet
	InstagibTrails []InstagibTrailEffect
	PlayerTrails   []PlayerTrailEffect
	Shells         []Shell
	Flashes        []MuzzleFlash
	Grenades       []Grenade
	FirePatches    []FirePatch
	Portals        []Portal
	DynamiteFX     []DynamiteEffect
	BlastFX        []BlastEffect
	JetThrustFX    []JetThrustEffect
	DropPackFX     []DropPackEffect
	Explosions     []Explosion
	Killfeeds      []KillFeedEntry
	Crates         []Crate
	Powerups       []Powerup
	PowerupNameFX  []PowerupNameEffect
	LifeBlingFX    []LifeBlingEffect
	SFXEvents      []netSFXEvent

	MapFXFrame int
	CameraX    float64
	CameraY    float64

	GameMode   int
	TotalLives int
	TeamGame   bool
	CrateON    bool
	PowerON    bool
	GameWin    bool

	OnlineMatchSetup               bool
	CustomPage                     int
	CustomMenuX                    float64
	CustomMode                     int
	CustomMap                      int
	CustomMapSetGM2                bool
	CustomLives                    int
	CustomSeriesRounds             int
	CustomSeriesRound              int
	CustomSeriesStats              [4]customSeriesPlayerStats
	CustomSeriesMaps               [customSeriesMaxRounds]int
	CustomSeriesPlaylistConfigured bool
	ChaosEnabled                   bool
	ChaosCurrentEffect             int
	ChaosFrame                     int
	ChaosBannerFrame               int
	ChaosIntervalSeconds           int
	OnlinePlayerCount              int
	OnlinePlayerSlots              [4]bool
	OnlineCustomPlayers            [4]CustomPlayerConfig
	OnlineMapRotation              int
	OnlineWeaponMode               int
	OnlineOnlyWeapon               int
	OnlinePerksEnabled             bool
	OnlinePowerupsEnabled          bool
	OnlineBlockedWeapons           [developedWeaponSlotCount]bool
	OnlineBlockedPerks             [maxCustomPerk + 1]bool
	PostGameReady                  [4]bool
	PostGameReadyCountdown         int

	Paused            bool
	PausePressed      int
	FadeActive        bool
	FadeFrame         int
	FadeTarget        int
	FadePurpose       int
	MatchWinCountdown int
	SoloWinFrame      int
	TeamGameWin       bool
	TeamWinner        int
	WinnerPlayerID    int
	WinnerAnimFrame   int
	TeamWinAnimFrame  int
	CampaignLoseFrame int
	ZombieWaveFrame   int

	HUDLastLifeFrame   [4]int
	HUDLastLifePlaying [4]bool
	HUDLastLevel       [4]int

	CampaignMode  bool
	CampaignLevel int

	Gototest        bool
	TestGunNumber   int
	TestGunDisabled bool
	TestGunRespawn  int
	TestGunFrame    int
}

type netplayPeer struct {
	playerID int
	conn     net.Conn
	link     *netplayLink
	addr     string

	inputMu       sync.RWMutex
	input         PlayerInputState
	inputReceived time.Time
	configMu      sync.RWMutex
	config        CustomPlayerConfig
	hasConfig     atomic.Bool
	postGameReady atomic.Bool

	stateSend chan netSnapshot
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
	pendingMu         sync.Mutex
	pending           map[net.Conn]struct{}

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

	stateMu    sync.RWMutex
	state      netSnapshot
	hasState   atomic.Bool
	outbox     *netplayOutbox
	lastErrMu  sync.RWMutex
	lastErr    error
	remoteAddr string
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
	return newConfiguredHostNetplay(addr, stack, nil)
}

// Configure immutable authentication and slot policy before the accept loop starts.
func newConfiguredHostNetplay(addr string, stack *netplayStack, configure func(*netplaySession)) (*netplaySession, error) {
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
	if configure != nil {
		configure(s)
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
	s.pendingMu.Lock()
	if s.closed.Load() || len(s.pending) >= maxPendingNetHandshakes {
		s.pendingMu.Unlock()
		_ = conn.Close()
		return false
	}
	if s.pending == nil {
		s.pending = make(map[net.Conn]struct{})
	}
	s.pending[conn] = struct{}{}
	s.pendingHandshakes.Add(1)
	s.pendingMu.Unlock()
	go func() {
		defer func() {
			s.pendingMu.Lock()
			delete(s.pending, conn)
			s.pendingHandshakes.Add(-1)
			s.pendingMu.Unlock()
		}()
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
	_ = link.SetReadDeadline(time.Now().Add(netHandshakeTimeout))
	if err := link.Write(netWireMessage{Type: netMessageChallenge, Protocol: netplayProtocolVersion, Challenge: challenge}, maxServerWireFrame); err != nil {
		networkDiagnostic("HOST handshake reject remote=%s reason=write_challenge err=%v", conn.RemoteAddr(), err)
		_ = link.Close()
		return
	}
	var first netWireMessage
	if err := link.Read(maxClientWireFrame, &first); err != nil {
		networkDiagnostic("HOST handshake reject remote=%s reason=read_hello err=%v", conn.RemoteAddr(), err)
		_ = link.Close()
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
		networkDiagnostic("HOST handshake reject remote=%s reason=invite_auth", conn.RemoteAddr())
		_ = link.Close()
		return
	}
	_ = link.SetReadDeadline(time.Time{})
	if s.closed.Load() {
		_ = link.Close()
		return
	}

	s.peersMu.Lock()
	if s.closed.Load() {
		s.peersMu.Unlock()
		_ = link.Close()
		return
	}
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
		networkDiagnostic("HOST handshake reject remote=%s reason=no_online_slot", conn.RemoteAddr())
		_ = link.Close()
		return
	}
	peer := &netplayPeer{
		playerID:  playerID,
		conn:      conn,
		link:      link,
		addr:      conn.RemoteAddr().String(),
		stateSend: make(chan netSnapshot, 1),
		done:      make(chan struct{}),
	}
	s.peers[playerID] = peer
	s.peersMu.Unlock()
	if err := link.Write(netWireMessage{Type: netMessageHello, Protocol: netplayProtocolVersion, PlayerID: playerID}, maxServerWireFrame); err != nil {
		s.peersMu.Lock()
		if s.peers[playerID] == peer {
			delete(s.peers, playerID)
		}
		s.peersMu.Unlock()
		s.setErr(err)
		_ = link.Close()
		return
	}
	networkDiagnostic("HOST handshake accepted remote=%s player=%d", conn.RemoteAddr(), playerID)
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

func (p *netplayPeer) latestInput() PlayerInputState {
	if p == nil {
		return PlayerInputState{}
	}
	p.inputMu.RLock()
	defer p.inputMu.RUnlock()
	if p.closed.Load() || p.inputReceived.IsZero() || time.Since(p.inputReceived) > netInputMaxAge {
		return PlayerInputState{}
	}
	return p.input
}

func (p *netplayPeer) queueState(state netSnapshot) {
	if p == nil || p.closed.Load() {
		return
	}
	select {
	case p.stateSend <- state:
	default:
		select {
		case <-p.stateSend:
		default:
		}
		select {
		case p.stateSend <- state:
		default:
		}
	}
}

func (s *netplaySession) hostServePeer(peer *netplayPeer) {
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		heartbeat := time.NewTicker(netHeartbeatInterval)
		defer heartbeat.Stop()
		for {
			select {
			case state := <-peer.stateSend:
				if err := peer.link.Write(netWireMessage{Type: netMessageState, Protocol: netplayProtocolVersion, PlayerID: peer.playerID, State: state}, maxServerWireFrame); err != nil {
					s.setErr(err)
					peer.close()
					return
				}
			case <-heartbeat.C:
				if err := peer.link.Write(netWireMessage{Type: netMessageKeepAlive, Protocol: netplayProtocolVersion, PlayerID: peer.playerID}, maxServerWireFrame); err != nil {
					s.setErr(err)
					peer.close()
					return
				}
			case <-peer.done:
				return
			}
		}
	}()

	var rate netRateGuard
	for !s.closed.Load() && !peer.closed.Load() {
		var msg netWireMessage
		_ = peer.link.SetReadDeadline(time.Now().Add(netPeerIdleTimeout))
		if err := peer.link.Read(maxClientWireFrame, &msg); err != nil {
			if !errors.Is(err, net.ErrClosed) && !peer.closed.Load() {
				s.setErr(err)
			}
			break
		}
		if !rate.Allow(time.Now()) {
			s.setErr(errors.New("remote client exceeded netplay message rate limit"))
			break
		}
		if err := validateNetMessageForHost(msg); err != nil || msg.Type == netMessageHello || msg.Type == netMessageProbe || msg.PlayerID != peer.playerID {
			if err == nil {
				err = errors.New("unexpected message or player identity on established netplay connection")
			}
			s.setErr(err)
			break
		}
		switch msg.Type {
		case netMessageInput:
			peer.inputMu.Lock()
			peer.input = msg.Input
			peer.inputReceived = time.Now()
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
		}
	}
	peer.close()
	<-writerDone
	s.peersMu.Lock()
	if current := s.peers[peer.playerID]; current == peer {
		delete(s.peers, peer.playerID)
	}
	s.peersMu.Unlock()
}

func (s *netplaySession) clientRun(link *netplayLink) {
	defer func() {
		s.ready.Store(false)
		_ = link.Close()
	}()

	_ = link.SetReadDeadline(time.Now().Add(netHandshakeTimeout))
	var challenge netWireMessage
	if err := link.Read(maxServerWireFrame, &challenge); err != nil || challenge.Type != netMessageChallenge {
		if err == nil {
			err = errors.New("invalid host authentication challenge")
		}
		s.setErr(fmt.Errorf("netplay handshake challenge: %w", err))
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
	if err := link.Write(netWireMessage{Type: netMessageHello, Protocol: netplayProtocolVersion, AuthProof: proof}, maxClientWireFrame); err != nil {
		s.setErr(fmt.Errorf("netplay handshake response: %w", err))
		return
	}

	var hello netWireMessage
	if err := link.Read(maxServerWireFrame, &hello); err != nil || hello.Type != netMessageHello {
		if err == nil {
			err = errors.New("host rejected the connection")
		}
		s.setErr(fmt.Errorf("netplay handshake acceptance: %w", err))
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
	_ = link.SetReadDeadline(time.Time{})

	stopWriter := make(chan struct{})
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		s.clientWriteLoop(link, stopWriter)
	}()
	s.clientReadLoop(link)
	close(stopWriter)
	_ = link.Close()
	<-writerDone
}

func (s *netplaySession) clientReadLoop(link *netplayLink) {
	for !s.closed.Load() {
		var msg netWireMessage
		_ = link.SetReadDeadline(time.Now().Add(netPeerIdleTimeout))
		if err := link.Read(maxServerWireFrame, &msg); err != nil {
			s.setErr(fmt.Errorf("receive host message: %w", err))
			return
		}
		if err := validateNetMessageForClient(msg); err != nil || (msg.Type != netMessageState && msg.Type != netMessageKeepAlive) || msg.PlayerID != s.assignedPlayer() {
			if err == nil {
				err = errors.New("unexpected host message")
			}
			s.setErr(err)
			s.ready.Store(false)
			return
		}
		if msg.Type == netMessageKeepAlive {
			continue
		}
		if err := validateNetSnapshot(msg.State); err != nil {
			s.setErr(err)
			return
		}
		s.stateMu.Lock()
		s.state = msg.State
		s.stateMu.Unlock()
		s.hasState.Store(true)
	}
}

func (s *netplaySession) clientWriteLoop(link *netplayLink, stop <-chan struct{}) {
	heartbeat := time.NewTicker(netHeartbeatInterval)
	defer heartbeat.Stop()
	for {
		msg, ok := s.outbox.nextWithHeartbeat(stop, s.done, heartbeat.C)
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

func (s *netplaySession) peerAddress(playerID int) string {
	if s == nil || s.mode != netplayHost {
		return ""
	}
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()
	if peer := s.peers[playerID]; peer != nil {
		return peer.addr
	}
	return ""
}

func (s *netplaySession) latestInputFor(playerID int) PlayerInputState {
	if s == nil || s.mode != netplayHost {
		return PlayerInputState{}
	}
	s.peersMu.RLock()
	peer := s.peers[playerID]
	s.peersMu.RUnlock()
	if peer == nil {
		return PlayerInputState{}
	}
	return peer.latestInput()
}

func (s *netplaySession) queueClientMessage(msg netWireMessage) {
	if s == nil || s.closed.Load() || s.mode != netplayClient || !s.connected() || s.outbox == nil {
		return
	}
	s.outbox.Push(msg)
}

func (s *netplaySession) queueInput(input PlayerInputState) {
	s.queueClientMessage(netWireMessage{Type: netMessageInput, Input: input})
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

func (s *netplaySession) queueState(state netSnapshot) {
	if s == nil || s.closed.Load() || s.mode != netplayHost {
		return
	}
	s.peersMu.RLock()
	peers := make([]*netplayPeer, 0, len(s.peers))
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	s.peersMu.RUnlock()
	for _, peer := range peers {
		peer.queueState(state)
	}
}

func (s *netplaySession) latestState() (netSnapshot, bool) {
	if s == nil || !s.hasState.Load() {
		return netSnapshot{}, false
	}
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state, true
}

func (s *netplaySession) Close() error {
	if s == nil || s.closed.Swap(true) {
		return nil
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.ready.Store(false)
	s.pendingMu.Lock()
	pending := make([]net.Conn, 0, len(s.pending))
	for conn := range s.pending {
		pending = append(pending, conn)
	}
	s.pendingMu.Unlock()
	for _, conn := range pending {
		_ = conn.Close()
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

// StartNetHost starts a LAN-only OPEN host. CLI and legacy menus never
// publish a public endpoint implicitly.
func (g *Game) StartNetHost(addr string) error {
	return g.StartNetHostConfig(addr, onlineSessionScopeLAN, onlineSessionOpen)
}

// StartNetHostMode is a LAN compatibility entry point. Internet hosting must
// explicitly select the experimental scope and private invite policy.
func (g *Game) StartNetHostMode(addr string, visibility int) error {
	return g.StartNetHostConfig(addr, onlineSessionScopeLAN, visibility)
}

// StartNetHostConfig supports LAN+OPEN, LAN+INVITE and opt-in INTERNET+INVITE.
// Experimental Internet sessions never publish an OPEN DHT listing.

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
	if scope == onlineSessionScopeInternet {
		if !experimentalInternetEnabled() {
			return errors.New("Internet hosting is experimental; set GUNMAYHEM_EXPERIMENTAL_INTERNET=1 to opt in")
		}
		if visibility != onlineSessionInviteOnly {
			return errors.New("experimental Internet hosting requires a private invite")
		}
	}
	_ = g.CloseNetplay()
	key, err := newOnlineSessionKey()
	if err != nil {
		return fmt.Errorf("create session key: %w", err)
	}
	shortCode := ""
	authKey := key
	if visibility == onlineSessionInviteOnly && scope == onlineSessionScopeLAN {
		shortCode = decentralizedOpenCode(key)
		authKey = decentralizedCodeSessionKey(shortCode)
	}
	s, err := newConfiguredHostNetplay(addr, defaultNetplayStack(), func(s *netplaySession) {
		s.inviteOnly = visibility == onlineSessionInviteOnly
		s.expectedSessionKey = authKey
		s.shortJoinCode = shortCode
		s.publicSessionID = publicOnlineSessionID(authKey)
		s.publicSessionName = sanitizeRemotePlayerName(g.multiplayerSessionName)
		if s.publicSessionName == "" {
			s.publicSessionName = "Gun Mayhem Game"
		}
		for id := 2; id <= maxOnlinePlayers; id++ {
			s.allowedRemote[id].Store(g.customPlayers[id-1].Type == 1)
		}
	})
	if err != nil {
		return err
	}
	g.netplay = s
	g.onlineSessionScope = scope
	g.onlineSessionVisibility = visibility
	g.onlineSessionKey = authKey
	networkDiagnostic("HOST started listener=%s scope=%d visibility=%d", s.listener.Addr(), scope, visibility)
	// Seed browser metadata before LAN responders become visible so even a
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
			// Private invitations need no public directory or router port mapping.
			tunnel, tunnelErr := startCloudflareQuickTunnel(s)
			if tunnelErr != nil {
				_ = g.CloseNetplay()
				return fmt.Errorf("start Internet tunnel: %w", tunnelErr)
			}
			g.cloudflareTunnel = tunnel
			if _, tunnelErr = tunnel.WaitReady(cloudflareTunnelStartWindow); tunnelErr != nil {
				_ = g.CloseNetplay()
				return fmt.Errorf("start Internet tunnel: %w", tunnelErr)
			}
		}
	}

	// Internet invites contain the WSS hostname and full secret. They require
	// no public directory, DHT announcement, QUIC port or router port mapping.

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
	s, err := newClientNetplay(addr)
	if err != nil {
		return err
	}
	if err := waitForClientHandshake(s, netHandshakeTimeout+time.Second); err != nil {
		_ = s.Close()
		return err
	}
	g.netplay = s
	return nil
}

func (g *Game) StartNetClientInvite(code string) error {
	if g == nil {
		return errors.New("nil game")
	}
	invite, err := decodeInviteCode(code)
	if err != nil {
		return err
	}
	_ = g.CloseNetplay()
	var failures []string
	for _, address := range invite.Candidates {
		s, dialErr := newClientNetplayWithKey(address, invite.SessionKey, 1200*time.Millisecond)
		if dialErr != nil {
			failures = append(failures, address+": "+dialErr.Error())
			continue
		}
		if handshakeErr := waitForClientHandshake(s, 2*time.Second); handshakeErr != nil {
			_ = s.Close()
			failures = append(failures, address+": "+handshakeErr.Error())
			continue
		}
		g.netplay = s
		g.onlineSessionVisibility = onlineSessionOpen
		if invite.InviteOnly {
			g.onlineSessionVisibility = onlineSessionInviteOnly
		}
		return nil
	}

	// Direct TCP can fail behind CGNAT. The private DHT key is derived from the
	// invite secret, so only somebody holding the code can discover this host's
	// UDP/QUIC endpoint. QUIC then uses decentralized hole punching.
	if p2pAddresses, p2pErr := discoverDHTPeerAddresses(privateHostDHTHash(invite.SessionKey), 7*time.Second); p2pErr == nil {
		for _, address := range p2pAddresses {
			conn, dialErr := dialP2PQUIC(address, invite.SessionKey, p2pDialTimeout)
			if dialErr != nil {
				failures = append(failures, "P2P "+address+": "+dialErr.Error())
				continue
			}
			s, sessionErr := newClientNetplayFromConnWithKey(conn, "P2P "+address, invite.SessionKey)
			if sessionErr != nil {
				_ = conn.Close()
				failures = append(failures, "P2P "+address+": "+sessionErr.Error())
				continue
			}
			if handshakeErr := waitForClientHandshake(s, 4*time.Second); handshakeErr != nil {
				_ = s.Close()
				failures = append(failures, "P2P "+address+": "+handshakeErr.Error())
				continue
			}
			g.netplay = s
			g.onlineSessionVisibility = onlineSessionOpen
			if invite.InviteOnly {
				g.onlineSessionVisibility = onlineSessionInviteOnly
			}
			return nil
		}
	} else {
		failures = append(failures, "P2P discovery: "+p2pErr.Error())
	}
	if len(failures) == 0 {
		return errors.New("invite code has no usable address")
	}
	return errors.New("could not reach host: " + strings.Join(failures, "; "))
}

func (g *Game) StartNetClientServer(server multiplayerServerInfo) (finalErr error) {
	if g == nil {
		return errors.New("nil game")
	}
	networkDiagnostic("JOIN start name=%q source=%q transport=%q address=%s relay=%s", server.Name, server.Sources, server.Transport, server.Address, server.RelayAddress)
	defer func() {
		networkDiagnostic("JOIN finish address=%s err=%v", server.Address, finalErr)
	}()
	_ = g.CloseNetplay()

	if strings.EqualFold(strings.TrimSpace(server.Transport), "cloudflare") || strings.TrimSpace(server.TunnelAddress) != "" {
		address := strings.TrimSpace(server.TunnelAddress)
		if address == "" {
			address = strings.TrimSpace(server.Address)
		}
		s, err := newClientNetplayCloudflare(address, "", 8*time.Second)
		if err != nil {
			return err
		}
		if err := waitForClientHandshake(s, netHandshakeTimeout+time.Second); err != nil {
			_ = s.Close()
			return err
		}
		g.netplay = s
		return nil
	}

	if strings.EqualFold(strings.TrimSpace(server.Transport), "quic") && strings.TrimSpace(server.Address) != "" {
		// Browser metadata can advertise a native IPv6/public IPv4/UPnP TCP
		// address. Prefer that direct host path before NAT hole punching.
		if direct := strings.TrimSpace(server.DirectAddress); direct != "" {
			if s, err := tryDirectClientSession(direct, "", 1800*time.Millisecond); err == nil {
				g.netplay = s
				return nil
			} else {
				networkDiagnostic("JOIN direct advertised path failed address=%s err=%v", direct, err)
			}
		}
		conn, err := dialP2PQUIC(server.Address, "", p2pDialTimeout)
		if err != nil {
			return err
		}
		s, err := newClientNetplayFromConn(conn, "P2P "+server.Address)
		if err != nil {
			_ = conn.Close()
			return err
		}
		if err := waitForClientHandshake(s, 4*time.Second); err != nil {
			_ = s.Close()
			return err
		}
		g.netplay = s
		return nil
	}

	relayID := strings.TrimSpace(server.RelayID)
	if relayID == "" {
		relayID = strings.TrimSpace(server.ID)
	}
	var directErr error
	if strings.TrimSpace(server.Address) != "" {
		directTimeout := 5 * time.Second
		if strings.TrimSpace(server.RelayAddress) != "" && relayID != "" {
			directTimeout = 750 * time.Millisecond
		}
		if s, err := newClientNetplayWithTimeout(server.Address, directTimeout); err == nil {
			if handshakeErr := waitForClientHandshake(s, netHandshakeTimeout+time.Second); handshakeErr == nil {
				g.netplay = s
				return nil
			} else {
				directErr = handshakeErr
				_ = s.Close()
			}
		} else {
			directErr = err
		}
	}
	if strings.TrimSpace(server.RelayAddress) != "" && relayID != "" {
		conn, err := dialRelayNetplay(server.RelayAddress, relayID)
		if err == nil {
			s, sessionErr := newClientNetplayFromConn(conn, "relay "+server.RelayAddress)
			if sessionErr == nil {
				if handshakeErr := waitForClientHandshake(s, netHandshakeTimeout+time.Second); handshakeErr == nil {
					g.netplay = s
					return nil
				} else {
					sessionErr = handshakeErr
					_ = s.Close()
				}
			}
			_ = conn.Close()
			err = sessionErr
		}
		if directErr != nil {
			return fmt.Errorf("direct connection failed: %v; relay failed: %w", directErr, err)
		}
		return fmt.Errorf("relay failed: %w", err)
	}
	if directErr != nil {
		return directErr
	}
	return errors.New("server has no usable address")
}

func (g *Game) CloseNetplay() error {
	if g == nil {
		return nil
	}
	if g.serverAdvertiser != nil {
		_ = g.serverAdvertiser.Close()
		g.serverAdvertiser = nil
	}
	if g.localServerAdvertiser != nil {
		_ = g.localServerAdvertiser.Close()
		g.localServerAdvertiser = nil
	}
	if g.lanDiscoveryResponder != nil {
		_ = g.lanDiscoveryResponder.Close()
		g.lanDiscoveryResponder = nil
	}
	if g.globalDHTAdvertiser != nil {
		_ = g.globalDHTAdvertiser.Close()
		g.globalDHTAdvertiser = nil
	}
	if g.p2pHost != nil {
		_ = g.p2pHost.Close()
		g.p2pHost = nil
	}
	if g.cloudflareTunnel != nil {
		_ = g.cloudflareTunnel.Close()
		g.cloudflareTunnel = nil
	}
	if g.natPortMapper != nil {
		_ = g.natPortMapper.Close()
		g.natPortMapper = nil
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
		public := ""
		if g.serverAdvertiser != nil {
			registered, addr, regErr := g.serverAdvertiser.Status()
			switch {
			case registered && addr != "":
				public = " | public: " + addr
			case regErr != nil:
				public = " | master offline"
			default:
				public = " | registering..."
			}
		}
		if g.natPortMapper != nil {
			done, mapped, method, _, externalPort, _ := g.natPortMapper.Status()
			switch {
			case mapped:
				public += fmt.Sprintf(" | router: %s/%d", method, externalPort)
			case !done:
				public += " | opening router..."
			default:
				public += " | auto port mapping unavailable"
			}
		}
		if g.globalDHTAdvertiser != nil {
			ready, port, dhtErr := g.globalDHTAdvertiser.Status()
			switch {
			case ready:
				public += fmt.Sprintf(" | global DHT:%d", port)
			case dhtErr != nil:
				public += " | global DHT unavailable"
			default:
				public += " | global DHT publishing..."
			}
		}
		if g.serverAdvertiser != nil {
			configured, ready, _ := g.serverAdvertiser.RelayStatus()
			switch {
			case ready:
				public += " | relay ready"
			case configured:
				public += " | relay connecting..."
			}
		}
		count := s.connectedCount()
		if count > 0 {
			return fmt.Sprintf("hosting - players %d/%d%s", count+1, maxOnlinePlayers, public)
		}
		return "hosting - waiting for players" + public
	case netplayClient:
		if s.connected() {
			if s.hasState.Load() {
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
			p.RemoteInput = g.netplay.latestInputFor(p.ID)
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
		_ = g.CloseNetplay()
		g.screen = screenCustomGame
		g.customOnlineMode = true
		g.onlineMatchSetup = false
		g.customNetworkOpen = true
		g.multiplayerMessage = "HOST DISCONNECTED"
		g.beginMultiplayerServerBrowser()
		return nil
	}
	if state, ok := g.netplay.latestState(); ok {
		g.applyNetSnapshot(state)
	}
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
		g.netplay.queueInput(capturePlayerInput(g.controlConfigs[0]))
	}
	return nil
}
