package game

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

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
			s.stateTick = decoded.tick
			s.stateInputAck = decoded.inputAck
			s.stateMu.Unlock()
			s.snapshotMeter.note(time.Now())
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

func (s *netplaySession) queuePlayerConfig(cfg CustomPlayerConfig) {
	cfg.Type = 1
	s.queueClientMessage(netWireMessage{Type: netMessagePlayerConfig, Config: cfg})
}

func (s *netplaySession) queuePostGameReady(ready bool) {
	s.queueClientMessage(netWireMessage{Type: netMessagePostGameReady, Ready: ready})
}

// nextPlayoutState advances the client's playout buffer by one tick and
// returns the (possibly interpolated) snapshot to render.
func (s *netplaySession) nextPlayoutState() (netSnapshot, bool) {
	if s == nil || !s.hasState.Load() {
		return netSnapshot{}, false
	}
	return s.playout.next()
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
	g.bossBanner, g.bossBannerFrame = state.BossBanner, state.BossBannerFrame
	g.applyZombieSnapshot(&state)
	g.onlineZombie = state.OnlineZombie
	g.zombieSetup.Mode = state.ZombieSetupMode
	g.zombieSetup.Arena = state.ZombieSetupArena
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
	g.slowMo = simClock{left: state.SlowMoTicks, acc: state.SlowMoAcc}
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
				g.audio.playSFX(event.Name, volume, gain, event.Rate)
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
	// SLOW-MO: the predicted own player (if any) keeps its own interpolation.
	keepID := g.renderInterp.freeID
	g.renderInterp.freeID = 0
	if state, ok := g.netplay.nextPlayoutState(); ok {
		g.applyNetSnapshot(state)
		if g.screen == screenGameplay && state.SlowMoAcc > 0 {
			// The host skipped its simulation on this tick: keep drawing
			// between the last two sim steps.
			g.renderInterp.keepPrevPair(keepID)
		}
	}
	if g.screen == screenGameplay {
		g.renderInterp.simPhase, g.renderInterp.simInc = g.slowMo.renderPhase()
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
	if g.screen == screenZombieResults {
		g.updateZombieResultsMenu()
		return nil
	}
	// A guest's own zombie shop owns the keys while it is open (ESC closes it).
	zombieShop := g.screen == screenGameplay && g.updateZombieGuestShop()
	if !zombieShop && inpututil.IsKeyJustPressed(ebiten.KeyEscape) { // Leave the remote session outside Custom Game.
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
		input := netClientInputForFocus(capturePlayerInput(g.controlConfigs[0]), ebiten.IsFocused())
		if zombieShop {
			input = PlayerInputState{} // the host masks it too
		} else {
			input = g.zombieGuestInputMask(input)
		}
		g.sendZombieRequests()
		if seq, ok := g.netplay.queueInput(input); ok {
			g.netPredict.record(seq, input)
			g.applyNetPrediction(seq)
		}
	} else {
		g.netPredict.reset()
	}
	return nil
}
