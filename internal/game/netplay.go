package game

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const netplayProtocolVersion = 1

type netplayMode int

const (
	netplayOff netplayMode = iota
	netplayHost
	netplayClient
)

const (
	netMessageHello = 1
	netMessageInput = 2
	netMessageState = 3
)

type netSFXEvent struct {
	Seq  uint64
	Name string
	Loud bool
}

type netWireMessage struct {
	Type     int
	Protocol int
	Input    PlayerInputState
	State    netSnapshot
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

type netplaySession struct {
	mode netplayMode

	listener net.Listener
	connMu   sync.Mutex
	conn     net.Conn
	closed   atomic.Bool
	ready    atomic.Bool

	inputMu sync.RWMutex
	input   PlayerInputState

	stateMu    sync.RWMutex
	state      netSnapshot
	hasState   atomic.Bool
	stateSend  chan netSnapshot
	inputSend  chan PlayerInputState
	lastErrMu  sync.RWMutex
	lastErr    error
	remoteAddr string
}

func normalizeListenAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ":7777"
	}
	if !strings.Contains(addr, ":") {
		return ":" + addr
	}
	return addr
}

func normalizeJoinAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "127.0.0.1:7777"
	}
	if !strings.Contains(addr, ":") {
		return addr + ":7777"
	}
	return addr
}

func newHostNetplay(addr string) (*netplaySession, error) {
	ln, err := net.Listen("tcp", normalizeListenAddress(addr))
	if err != nil {
		return nil, err
	}
	s := &netplaySession{
		mode:      netplayHost,
		listener:  ln,
		stateSend: make(chan netSnapshot, 1),
	}
	go s.acceptLoop()
	return s, nil
}

func newClientNetplay(addr string) (*netplaySession, error) {
	address := normalizeJoinAddress(addr)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return nil, err
	}
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetNoDelay(true)
	}
	s := &netplaySession{
		mode:       netplayClient,
		conn:       conn,
		inputSend:  make(chan PlayerInputState, 1),
		remoteAddr: conn.RemoteAddr().String(),
	}
	s.ready.Store(true)
	go s.clientReadLoop(conn)
	go s.clientWriteLoop(conn)
	return s, nil
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
		if tcp, ok := conn.(*net.TCPConn); ok {
			_ = tcp.SetNoDelay(true)
		}
		// One remote player at a time. Replacing a stale connection lets a
		// client reconnect without restarting the host process.
		s.connMu.Lock()
		if s.conn != nil {
			_ = s.conn.Close()
		}
		s.conn = conn
		s.remoteAddr = conn.RemoteAddr().String()
		s.connMu.Unlock()
		s.ready.Store(true)
		go s.hostServeConn(conn)
	}
}

func (s *netplaySession) hostServeConn(conn net.Conn) {
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		enc := gob.NewEncoder(conn)
		if err := enc.Encode(netWireMessage{Type: netMessageHello, Protocol: netplayProtocolVersion}); err != nil {
			s.setErr(err)
			return
		}
		for state := range s.stateSend {
			if err := enc.Encode(netWireMessage{Type: netMessageState, Protocol: netplayProtocolVersion, State: state}); err != nil {
				s.setErr(err)
				return
			}
		}
	}()

	dec := gob.NewDecoder(conn)
	for !s.closed.Load() {
		var msg netWireMessage
		if err := dec.Decode(&msg); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				s.setErr(err)
			}
			break
		}
		if msg.Protocol != netplayProtocolVersion {
			s.setErr(fmt.Errorf("netplay protocol mismatch: remote=%d local=%d", msg.Protocol, netplayProtocolVersion))
			break
		}
		if msg.Type == netMessageInput {
			s.inputMu.Lock()
			s.input = msg.Input
			s.inputMu.Unlock()
		}
	}
	_ = conn.Close()
	<-writerDone
	s.connMu.Lock()
	if s.conn == conn {
		s.conn = nil
		s.ready.Store(false)
		s.inputMu.Lock()
		s.input = PlayerInputState{}
		s.inputMu.Unlock()
	}
	s.connMu.Unlock()
}

func (s *netplaySession) clientReadLoop(conn net.Conn) {
	dec := gob.NewDecoder(conn)
	for !s.closed.Load() {
		var msg netWireMessage
		if err := dec.Decode(&msg); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				s.setErr(err)
			}
			s.ready.Store(false)
			return
		}
		if msg.Protocol != netplayProtocolVersion {
			s.setErr(fmt.Errorf("netplay protocol mismatch: remote=%d local=%d", msg.Protocol, netplayProtocolVersion))
			s.ready.Store(false)
			return
		}
		if msg.Type == netMessageState {
			s.stateMu.Lock()
			s.state = msg.State
			s.stateMu.Unlock()
			s.hasState.Store(true)
		}
	}
}

func (s *netplaySession) clientWriteLoop(conn net.Conn) {
	enc := gob.NewEncoder(conn)
	if err := enc.Encode(netWireMessage{Type: netMessageHello, Protocol: netplayProtocolVersion}); err != nil {
		s.setErr(err)
		s.ready.Store(false)
		return
	}
	for input := range s.inputSend {
		if err := enc.Encode(netWireMessage{Type: netMessageInput, Protocol: netplayProtocolVersion, Input: input}); err != nil {
			s.setErr(err)
			s.ready.Store(false)
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

func (s *netplaySession) connected() bool {
	return s != nil && s.ready.Load()
}

func (s *netplaySession) latestInput() PlayerInputState {
	if s == nil {
		return PlayerInputState{}
	}
	s.inputMu.RLock()
	defer s.inputMu.RUnlock()
	return s.input
}

func (s *netplaySession) queueInput(input PlayerInputState) {
	if s == nil || s.closed.Load() || s.mode != netplayClient || !s.connected() {
		return
	}
	select {
	case s.inputSend <- input:
	default:
		select {
		case <-s.inputSend:
		default:
		}
		select {
		case s.inputSend <- input:
		default:
		}
	}
}

func (s *netplaySession) queueState(state netSnapshot) {
	if s == nil || s.closed.Load() || s.mode != netplayHost || !s.connected() {
		return
	}
	select {
	case s.stateSend <- state:
	default:
		select {
		case <-s.stateSend:
		default:
		}
		select {
		case s.stateSend <- state:
		default:
		}
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
	s.connMu.Lock()
	if s.conn != nil {
		_ = s.conn.Close()
	}
	s.connMu.Unlock()
	if s.stateSend != nil {
		close(s.stateSend)
	}
	if s.inputSend != nil {
		close(s.inputSend)
	}
	return nil
}

// StartNetHost starts a one-remote-player authoritative host. The host keeps
// normal control of Player 1; the remote client controls Player 2.
func (g *Game) StartNetHost(addr string) error {
	if g == nil {
		return errors.New("nil game")
	}
	s, err := newHostNetplay(addr)
	if err != nil {
		return err
	}
	if g.netplay != nil {
		_ = g.netplay.Close()
	}
	g.netplay = s
	return nil
}

// StartNetClient connects this process as Player 2. The client is a rendering
// mirror of the host and does not run authoritative combat simulation.
func (g *Game) StartNetClient(addr string) error {
	if g == nil {
		return errors.New("nil game")
	}
	s, err := newClientNetplay(addr)
	if err != nil {
		return err
	}
	if g.netplay != nil {
		_ = g.netplay.Close()
	}
	g.netplay = s
	return nil
}

func (g *Game) CloseNetplay() error {
	if g == nil || g.netplay == nil {
		return nil
	}
	err := g.netplay.Close()
	g.netplay = nil
	return err
}

func (g *Game) NetplayStatus() string {
	if g == nil || g.netplay == nil {
		return "offline"
	}
	s := g.netplay
	switch s.mode {
	case netplayHost:
		if s.connected() {
			return "hosting - connected: " + s.remoteAddr
		}
		return "hosting - waiting for player"
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
	input := g.netplay.latestInput()
	for _, p := range g.players {
		if p == nil || p.ID != 2 || p.IsDouble {
			continue
		}
		p.RemoteControlled = true
		p.RemoteInput = input
		p.AI = false
	}
}

func (g *Game) makeNetSnapshot() netSnapshot {
	state := netSnapshot{
		Screen:             int(g.screen),
		Arena:              g.arena,
		Bullets:            append([]Bullet(nil), g.bullets...),
		InstagibTrails:     append([]InstagibTrailEffect(nil), g.instagibTrails...),
		PlayerTrails:       append([]PlayerTrailEffect(nil), g.playerTrails...),
		Shells:             append([]Shell(nil), g.shells...),
		Flashes:            append([]MuzzleFlash(nil), g.flashes...),
		Grenades:           append([]Grenade(nil), g.grenades...),
		DynamiteFX:         append([]DynamiteEffect(nil), g.dynamiteFX...),
		BlastFX:            append([]BlastEffect(nil), g.blastFX...),
		JetThrustFX:        append([]JetThrustEffect(nil), g.jetThrustFX...),
		DropPackFX:         append([]DropPackEffect(nil), g.dropPackFX...),
		Explosions:         append([]Explosion(nil), g.explosions...),
		Killfeeds:          append([]KillFeedEntry(nil), g.killfeeds...),
		Crates:             append([]Crate(nil), g.crates...),
		Powerups:           append([]Powerup(nil), g.powerups...),
		PowerupNameFX:      append([]PowerupNameEffect(nil), g.powerupNameFX...),
		LifeBlingFX:        append([]LifeBlingEffect(nil), g.lifeBlingFX...),
		SFXEvents:          append([]netSFXEvent(nil), g.netSFXPending...),
		MapFXFrame:         g.mapFXFrame,
		CameraX:            g.cameraX,
		CameraY:            g.cameraY,
		GameMode:           g.GameMode,
		TotalLives:         g.TotalLives,
		TeamGame:           g.TeamGame,
		CrateON:            g.CrateON,
		PowerON:            g.PowerON,
		GameWin:            g.GameWin,
		Paused:             g.paused,
		PausePressed:       g.pausePressed,
		FadeActive:         g.fadeActive,
		FadeFrame:          g.fadeFrame,
		FadeTarget:         int(g.fadeTarget),
		FadePurpose:        int(g.fadePurpose),
		MatchWinCountdown:  g.matchWinCountdown,
		SoloWinFrame:       g.soloWinFrame,
		TeamGameWin:        g.teamGameWin,
		TeamWinner:         g.teamWinner,
		WinnerPlayerID:     g.winnerPlayerID,
		WinnerAnimFrame:    g.winnerAnimFrame,
		TeamWinAnimFrame:   g.teamWinAnimFrame,
		CampaignLoseFrame:  g.campaignLoseFrame,
		ZombieWaveFrame:    g.zombieWaveFrame,
		HUDLastLifeFrame:   g.hudLastLifeFrame,
		HUDLastLifePlaying: g.hudLastLifePlaying,
		HUDLastLevel:       g.hudLastLevel,
		CampaignMode:       g.campaignMode,
		CampaignLevel:      g.campaignLevel,
		Gototest:           g.gototest,
		TestGunNumber:      g.testGunNumber,
		TestGunDisabled:    g.testGunDisabled,
		TestGunRespawn:     g.testGunRespawn,
		TestGunFrame:       g.testGunFrame,
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
	if g.soundOn && g.audio != nil {
		for _, event := range g.netClientSFXPending {
			volume := 0.5
			if event.Loud {
				volume = 1
			}
			g.audio.playSFX(event.Name, volume)
		}
	}
	g.netClientSFXPending = g.netClientSFXPending[:0]
}

func (g *Game) updateNetClient() error {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayClient {
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) { // Leave the remote session locally.
		_ = g.CloseNetplay()
		g.screen = screenMultiplayer
		g.multiplayerMessage = "DISCONNECTED"
		return nil
	}
	g.netplay.queueInput(capturePlayerInput(g.controlConfigs[0]))
	if state, ok := g.netplay.latestState(); ok {
		g.applyNetSnapshot(state)
	}
	return nil
}
