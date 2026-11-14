package game

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

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

// queueState offers the newest frame to the peer's writer. Only the newest
// frame is kept: a slow peer skips stale snapshots instead of queueing them.
func (p *netplayPeer) queueState(frame *netHostFrame) {
	if p == nil || frame == nil || p.closed.Load() {
		return
	}
	signalLatest(p.stateSend, frame)
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
