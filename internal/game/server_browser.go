package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gunmayhem-go/internal/masterapi"
)

const (
	masterServerConfigFile      = "master_server.txt"
	masterHeartbeatInterval     = 10 * time.Second
	masterRequestTimeout        = 4 * time.Second
	masterServerListTimeout     = 5 * time.Second
	masterHeartbeatResponseMax  = 16 << 10
	masterServerListResponseMax = 2 << 20
	localServerStaleAfter       = 7 * time.Second
)

type multiplayerServerInfo struct {
	ID                   string
	RelayID              string
	Code                 string
	Name                 string
	Address              string
	RelayAddress         string
	Transport            string
	Sources              string
	Players              int
	MaxPlayers           int
	Protocol             int
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

func (s multiplayerServerInfo) Full() bool {
	return s.MaxPlayers > 0 && s.Players >= s.MaxPlayers
}

func (s multiplayerServerInfo) HasRulePreview() bool {
	return s.GameMode >= SourceGameModeNormal && s.GameMode <= SourceGameModeSurvival && s.Rounds >= customSeriesMinRounds
}

func copyServerPreview(dst *multiplayerServerInfo, src multiplayerServerInfo) {
	if dst == nil || !src.HasRulePreview() {
		return
	}
	dst.GameMode = src.GameMode
	dst.Lives = src.Lives
	dst.Rounds = src.Rounds
	dst.Maps = src.Maps
	dst.PlaylistConfigured = src.PlaylistConfigured
	dst.Crates = src.Crates
	dst.Pickups = src.Pickups
	dst.RuleSummary = src.RuleSummary
	dst.ChaosEnabled = src.ChaosEnabled
	dst.ChaosEvents = src.ChaosEvents
	dst.ChaosIntervalSeconds = src.ChaosIntervalSeconds
}

func applyProbeInfoToServer(server multiplayerServerInfo, info netProbeInfo) multiplayerServerInfo {
	server.ID = strings.TrimSpace(info.ID)
	if name := sanitizeRemotePlayerName(info.Name); name != "" {
		server.Name = name
	}
	server.Players = info.Players
	server.MaxPlayers = info.MaxPlayers
	server.GameMode = info.GameMode
	server.Lives = info.Lives
	server.Rounds = info.Rounds
	server.Maps = info.Maps
	server.PlaylistConfigured = info.PlaylistConfigured
	server.Crates = info.Crates
	server.Pickups = info.Pickups
	server.RuleSummary = strings.TrimSpace(info.RuleSummary)
	server.ChaosEnabled = info.ChaosEnabled
	server.ChaosEvents = info.ChaosEvents
	server.ChaosIntervalSeconds = info.ChaosIntervalSeconds
	server.DirectAddress = strings.TrimSpace(info.DirectAddress)
	server.TunnelAddress = strings.TrimSpace(info.TunnelAddress)
	// A discovered LAN/local TCP route stays local. In particular a public room
	// must not make LAN guests opt into Internet or strip its ID-bearing WSS path.
	if server.Transport == "lan" || server.Transport == "local" {
		server.TunnelAddress = ""
	}
	if server.TunnelAddress != "" {
		server.Transport = "cloudflare"
		server.Address = server.TunnelAddress
	}
	return server
}

type masterServerDiscoveryResult struct {
	Servers   []multiplayerServerInfo
	Err       error
	MasterErr error
	DHTErr    error
	Done      bool
}

type localHostRecord struct {
	PID        int    `json:"pid"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	TCPPort    int    `json:"tcp_port"`
	Players    int    `json:"players"`
	MaxPlayers int    `json:"max_players"`
	Protocol   int    `json:"protocol"`
	UpdatedMS  int64  `json:"updated_ms"`
}

type localServerAdvertiser struct {
	closed  atomic.Bool
	name    string
	tcpPort int
	session *netplaySession
	path    string
	stop    chan struct{}
	done    chan struct{}
}

type masterServerAdvertiser struct {
	closed  atomic.Bool
	client  *http.Client
	baseURL string
	name    string
	tcpPort int
	session *netplaySession
	mapper  *natPortMapper
	done    chan struct{}
	stop    chan struct{}

	mu           sync.RWMutex
	id           string
	secret       string
	publicAddr   string
	relayAddress string
	relayTunnel  *relayHostTunnel
	lastErr      error
	registered   bool
}

func decodeLimitedJSON(r io.Reader, maxBytes int64, dst any) error {
	if r == nil || dst == nil || maxBytes <= 0 {
		return errors.New("invalid limited JSON decode")
	}
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > maxBytes {
		return errors.New("master response too large")
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return err
	}
	return nil
}

func defaultPublicServerName() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		return "Gun Mayhem Host"
	}
	return strings.TrimSpace(host)
}

func localServerRegistryDir() string {
	if dir, err := persistentDataDir(); err == nil && strings.TrimSpace(dir) != "" {
		return filepath.Join(dir, "runtime", "hosts")
	}
	// Last-resort fallback stays inside the game/project tree. Never use the
	// operating-system TEMP directory for Gun Mayhem runtime state.
	return filepath.Join("data", "runtime", "hosts")
}

func startLocalServerAdvertiser(tcpPort int, session *netplaySession) (*localServerAdvertiser, error) {
	dir := localServerRegistryDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, strconv.Itoa(os.Getpid())+"-"+strconv.Itoa(tcpPort)+".json")
	name := "Gun Mayhem Game"
	if session != nil && strings.TrimSpace(session.publicSessionName) != "" {
		name = session.publicSessionName
	}
	a := &localServerAdvertiser{
		name:    name,
		tcpPort: tcpPort,
		session: session,
		path:    path,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	if err := a.writeRecord(); err != nil {
		return nil, err
	}
	go a.loop()
	return a, nil
}

func (a *localServerAdvertiser) writeRecord() error {
	players := 1
	if a.session != nil {
		players += a.session.connectedCount()
	}
	maxPlayers := maxOnlinePlayers
	if a.session != nil {
		maxPlayers = a.session.humanCapacity()
	}
	record := localHostRecord{
		PID:        os.Getpid(),
		Name:       a.name,
		TCPPort:    a.tcpPort,
		Players:    players,
		MaxPlayers: maxPlayers,
		Protocol:   netplayProtocolVersion,
		UpdatedMS:  time.Now().UnixMilli(),
	}
	if a.session != nil {
		record.ID = a.session.publicSessionID
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return os.WriteFile(a.path, data, 0o644)
}

func (a *localServerAdvertiser) loop() {
	defer close(a.done)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = a.writeRecord()
		case <-a.stop:
			return
		}
	}
}

func (a *localServerAdvertiser) Close() error {
	if a == nil || a.closed.Swap(true) {
		return nil
	}
	close(a.stop)
	_ = os.Remove(a.path)
	return nil
}

func discoverLocalServers() ([]multiplayerServerInfo, error) {
	dir := localServerRegistryDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	now := time.Now()
	out := make([]multiplayerServerInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var record localHostRecord
		if json.Unmarshal(data, &record) != nil || record.Protocol != netplayProtocolVersion || record.TCPPort <= 0 || record.TCPPort > 65535 {
			continue
		}
		updated := time.UnixMilli(record.UpdatedMS)
		if record.UpdatedMS <= 0 || now.Sub(updated) > localServerStaleAfter {
			_ = os.Remove(path)
			continue
		}
		name := strings.TrimSpace(record.Name)
		if name == "" {
			name = "Gun Mayhem Host"
		}
		out = append(out, multiplayerServerInfo{
			ID:         strings.TrimSpace(record.ID),
			Name:       name,
			Transport:  "local",
			Sources:    "LAN",
			Address:    net.JoinHostPort("127.0.0.1", strconv.Itoa(record.TCPPort)),
			Players:    record.Players,
			MaxPlayers: record.MaxPlayers,
			Protocol:   record.Protocol,
		})
	}
	return out, nil
}

func normalizeMasterServerURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	if u.Scheme == "http" && os.Getenv("GUNMAYHEM_ALLOW_INSECURE_MASTER") != "1" {
		host := strings.TrimSpace(u.Hostname())
		ip := net.ParseIP(host)
		localHTTP := strings.EqualFold(host, "localhost") || (ip != nil && (ip.IsLoopback() || ip.IsPrivate()))
		if !localHTTP {
			return ""
		}
	}
	return strings.TrimRight(u.String(), "/")
}

func configuredMasterServerURL() string {
	if v := normalizeMasterServerURL(os.Getenv("GUNMAYHEM_MASTER_URL")); v != "" {
		return v
	}
	if exe, err := os.Executable(); err == nil {
		if data, readErr := os.ReadFile(filepath.Join(filepath.Dir(exe), masterServerConfigFile)); readErr == nil {
			if v := normalizeMasterServerURL(string(data)); v != "" {
				return v
			}
		}
	}
	if data, err := os.ReadFile(masterServerConfigFile); err == nil {
		if v := normalizeMasterServerURL(string(data)); v != "" {
			return v
		}
	}
	return ""
}

func startMasterServerAdvertiser(tcpPort int, session *netplaySession) *masterServerAdvertiser {
	return startMasterServerAdvertiserWithMapper(tcpPort, session, nil)
}

func startMasterServerAdvertiserWithMapper(tcpPort int, session *netplaySession, mapper *natPortMapper) *masterServerAdvertiser {
	name := "Gun Mayhem Game"
	if session != nil && strings.TrimSpace(session.publicSessionName) != "" {
		name = session.publicSessionName
	}
	a := &masterServerAdvertiser{
		client:  &http.Client{Timeout: masterRequestTimeout},
		baseURL: configuredMasterServerURL(),
		name:    name,
		tcpPort: tcpPort,
		session: session,
		mapper:  mapper,
		done:    make(chan struct{}),
		stop:    make(chan struct{}),
	}
	if a.baseURL == "" {
		a.setStatus("", errors.New("public master server is not configured"), false)
		return a
	}
	go a.loop()
	return a
}

func (a *masterServerAdvertiser) loop() {
	defer close(a.done)
	// Register immediately so the relay can come online without waiting for
	// router discovery. If automatic port mapping later selects another public
	// port, refresh the directory entry once with the final mapping.
	a.sendHeartbeat()
	if a.mapper != nil && !a.closed.Load() {
		a.mapper.Wait(natDiscoveryTimeout + time.Second)
		if !a.closed.Load() {
			a.sendHeartbeat()
		}
	}
	ticker := time.NewTicker(masterHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			a.sendHeartbeat()
		case <-a.stop:
			return
		}
	}
}

func (a *masterServerAdvertiser) sendHeartbeat() {
	if a == nil || a.closed.Load() {
		return
	}
	players := 1
	if a.session != nil {
		players += a.session.connectedCount()
	}
	a.mu.RLock()
	id := a.id
	secret := a.secret
	a.mu.RUnlock()
	maxPlayers := maxOnlinePlayers
	if a.session != nil {
		maxPlayers = a.session.humanCapacity()
	}
	advertisedPort := a.tcpPort
	if a.mapper != nil {
		advertisedPort = a.mapper.AdvertisedPort(a.tcpPort)
	}
	hb := masterapi.HeartbeatRequest{
		ID:         id,
		Secret:     secret,
		Name:       a.name,
		Port:       advertisedPort,
		Players:    players,
		MaxPlayers: maxPlayers,
		Protocol:   netplayProtocolVersion,
	}
	if a.session != nil {
		hb.SessionID = strings.TrimSpace(a.session.publicSessionID)
		hb.Private = a.session.inviteOnly
		if hb.Private {
			hb.JoinCode = strings.TrimSpace(a.session.shortJoinCode)
		}
		info := a.session.publicProbeInfo()
		hb.GameMode = info.GameMode
		hb.Lives = info.Lives
		hb.Rounds = info.Rounds
		hb.Maps = info.Maps
		hb.PlaylistConfigured = info.PlaylistConfigured
		hb.Crates = info.Crates
		hb.Pickups = info.Pickups
		hb.RuleSummary = info.RuleSummary
		hb.ChaosEnabled = info.ChaosEnabled
		hb.ChaosEvents = info.ChaosEvents
		hb.ChaosIntervalSeconds = info.ChaosIntervalSeconds
	}
	data, err := json.Marshal(hb)
	if err != nil {
		a.setStatus("", err, false)
		return
	}
	req, err := http.NewRequest(http.MethodPost, a.baseURL+masterapi.HeartbeatPath, bytes.NewReader(data))
	if err != nil {
		a.setStatus("", err, false)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		a.setStatus("", err, false)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		a.setStatus("", errors.New("master heartbeat HTTP "+resp.Status), false)
		return
	}
	var answer masterapi.HeartbeatResponse
	if err := decodeLimitedJSON(resp.Body, masterHeartbeatResponseMax, &answer); err != nil {
		a.setStatus("", err, false)
		return
	}
	if strings.TrimSpace(answer.ID) == "" || strings.TrimSpace(answer.Secret) == "" || strings.TrimSpace(answer.Address) == "" {
		a.setStatus("", errors.New("master returned incomplete registration"), false)
		return
	}
	a.mu.Lock()
	a.id = strings.TrimSpace(answer.ID)
	a.secret = strings.TrimSpace(answer.Secret)
	a.publicAddr = strings.TrimSpace(answer.Address)
	a.relayAddress = strings.TrimSpace(answer.RelayAddress)
	relayAddress := a.relayAddress
	relayID := a.id
	relaySecret := a.secret
	existingRelay := a.relayTunnel
	a.lastErr = nil
	a.registered = true
	a.mu.Unlock()
	if relayAddress != "" && existingRelay == nil && a.session != nil {
		tunnel := startRelayHostTunnel(relayAddress, relayID, relaySecret, a.session)
		a.mu.Lock()
		if a.relayTunnel == nil && !a.closed.Load() {
			a.relayTunnel = tunnel
			tunnel = nil
		}
		a.mu.Unlock()
		if tunnel != nil {
			_ = tunnel.Close()
		}
	}
}

func (a *masterServerAdvertiser) setStatus(publicAddr string, err error, registered bool) {
	a.mu.Lock()
	if publicAddr != "" {
		a.publicAddr = publicAddr
	}
	a.lastErr = err
	a.registered = registered
	a.mu.Unlock()
}

func (a *masterServerAdvertiser) Status() (registered bool, publicAddr string, err error) {
	if a == nil {
		return false, "", nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.registered, a.publicAddr, a.lastErr
}

func (a *masterServerAdvertiser) RelayStatus() (configured, ready bool, err error) {
	if a == nil {
		return false, false, nil
	}
	a.mu.RLock()
	address := a.relayAddress
	tunnel := a.relayTunnel
	a.mu.RUnlock()
	if address == "" {
		return false, false, nil
	}
	if tunnel == nil {
		return true, false, nil
	}
	ready, err = tunnel.Status()
	return true, ready, err
}

func (a *masterServerAdvertiser) Close() error {
	if a == nil || a.closed.Swap(true) {
		return nil
	}
	close(a.stop)
	a.mu.RLock()
	id := a.id
	secret := a.secret
	relay := a.relayTunnel
	a.mu.RUnlock()
	if relay != nil {
		_ = relay.Close()
	}
	if id == "" || secret == "" || a.baseURL == "" {
		return nil
	}
	req, err := http.NewRequest(http.MethodDelete, a.baseURL+masterapi.ServersPath+"?id="+url.QueryEscape(id), nil)
	if err != nil {
		return err
	}
	req.Header.Set(masterapi.ServerSecretHeader, secret)
	client := &http.Client{Timeout: 1200 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

func fetchMasterServers(timeout time.Duration) ([]multiplayerServerInfo, error) {
	return fetchMasterServersByCode("", timeout)
}

func fetchMasterServersByCode(code string, timeout time.Duration) ([]multiplayerServerInfo, error) {
	if timeout <= 0 {
		timeout = masterServerListTimeout
	}
	baseURL := configuredMasterServerURL()
	if baseURL == "" {
		return nil, errors.New("public master server is not configured")
	}
	client := &http.Client{Timeout: timeout}
	endpoint := baseURL + masterapi.ServersPath + "?protocol=" + strconv.Itoa(netplayProtocolVersion)
	if code = strings.ToUpper(strings.TrimSpace(code)); code != "" {
		endpoint += "&code=" + url.QueryEscape(code)
	}
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("master list HTTP " + resp.Status)
	}
	var result masterapi.ServerListResponse
	if err := decodeLimitedJSON(resp.Body, masterServerListResponseMax, &result); err != nil {
		return nil, err
	}
	servers := make([]multiplayerServerInfo, 0, len(result.Servers))
	seen := make(map[string]bool)
	for _, server := range result.Servers {
		if server.Protocol != netplayProtocolVersion || server.MaxPlayers <= 0 || server.Players < 0 || server.Players > server.MaxPlayers {
			continue
		}
		address := strings.TrimSpace(server.Address)
		if _, _, err := net.SplitHostPort(address); err != nil || seen[address] {
			continue
		}
		seen[address] = true
		name := strings.TrimSpace(server.Name)
		if name == "" {
			name = "Gun Mayhem Host"
		}
		relayAddress := normalizeRelayAddress(server.RelayAddress)
		sessionID := strings.TrimSpace(server.SessionID)
		if sessionID == "" {
			sessionID = strings.TrimSpace(server.ID)
		}
		servers = append(servers, multiplayerServerInfo{
			ID:                   sessionID,
			RelayID:              strings.TrimSpace(server.ID),
			Name:                 name,
			Sources:              "MASTER",
			Address:              address,
			RelayAddress:         relayAddress,
			Players:              server.Players,
			MaxPlayers:           server.MaxPlayers,
			Protocol:             server.Protocol,
			GameMode:             server.GameMode,
			Lives:                server.Lives,
			Rounds:               server.Rounds,
			Maps:                 server.Maps,
			PlaylistConfigured:   server.PlaylistConfigured,
			Crates:               server.Crates,
			Pickups:              server.Pickups,
			RuleSummary:          strings.TrimSpace(server.RuleSummary),
			ChaosEnabled:         server.ChaosEnabled,
			ChaosEvents:          server.ChaosEvents,
			ChaosIntervalSeconds: server.ChaosIntervalSeconds,
		})
	}
	sort.Slice(servers, func(i, j int) bool {
		if servers[i].Full() != servers[j].Full() {
			return !servers[i].Full()
		}
		if servers[i].Name != servers[j].Name {
			return servers[i].Name < servers[j].Name
		}
		return servers[i].Address < servers[j].Address
	})
	return servers, nil
}

func mergeServerLists(lists ...[]multiplayerServerInfo) []multiplayerServerInfo {
	byKey := make(map[string]multiplayerServerInfo)
	byID := make(map[string]string)
	for _, list := range lists {
		for _, server := range list {
			if strings.TrimSpace(server.Address) == "" {
				continue
			}
			key := server.Address
			id := strings.TrimSpace(server.ID)
			if id != "" {
				if existingKey, ok := byID[id]; ok {
					key = existingKey
				} else {
					byID[id] = key
				}
			}
			if old, ok := byKey[key]; ok {
				if server.Players > old.Players {
					old.Players = server.Players
				}
				if server.MaxPlayers > old.MaxPlayers {
					old.MaxPlayers = server.MaxPlayers
				}
				if old.Name == "" || old.Name == "GLOBAL OPEN" {
					if server.Name != "" {
						old.Name = server.Name
					}
				}
				if old.ID == "" {
					old.ID = id
				}
				if old.RelayID == "" && server.RelayID != "" {
					old.RelayID = server.RelayID
				}
				if old.RelayAddress == "" && server.RelayAddress != "" {
					old.RelayAddress = server.RelayAddress
				}
				if !old.HasRulePreview() && server.HasRulePreview() {
					copyServerPreview(&old, server)
				}
				for _, source := range strings.Split(server.Sources, " + ") {
					source = strings.TrimSpace(source)
					if source != "" && !strings.Contains(" + "+old.Sources+" + ", " + "+source+" + ") {
						if old.Sources == "" {
							old.Sources = source
						} else {
							old.Sources += " + " + source
						}
					}
				}
				// Prefer a local/LAN TCP route over the global QUIC route when both
				// identify the same session. It is faster and avoids hairpin NAT.
				if (old.Transport == "" || old.Transport == "quic") && server.Transport != "" && server.Transport != "quic" {
					old.Transport = server.Transport
					old.Address = server.Address
				}
				byKey[key] = old
				continue
			}
			byKey[key] = server
			if id != "" {
				byID[id] = key
			}
		}
	}
	out := make([]multiplayerServerInfo, 0, len(byKey))
	for _, server := range byKey {
		out = append(out, server)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Full() != out[j].Full() {
			return !out[i].Full()
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Address < out[j].Address
	})
	return out
}

type publicCatalogState uint8

const (
	publicCatalogDisabled publicCatalogState = iota
	publicCatalogUnconfigured
	publicCatalogSearching
	publicCatalogAvailable
	publicCatalogUnavailable
)

// Both browser layouts use the same outcome rather than presenting a failed
// catalog request as a successful search with no rooms.
func (g *Game) multiplayerServerListEmptyLabel() string {
	if g == nil || !g.multiplayerDiscoveryDone || g.multiplayerDiscoveryCh != nil {
		return "SEARCHING..."
	}
	switch g.multiplayerCatalogState {
	case publicCatalogUnconfigured:
		return "CATALOG NOT CONFIGURED"
	case publicCatalogUnavailable:
		return "CATALOG UNAVAILABLE - RETRY"
	case publicCatalogAvailable:
		return "NO OPEN ROOMS FOUND"
	default:
		return "NO OPEN SERVERS - USE JOIN CODE"
	}
}

func (g *Game) beginMultiplayerServerBrowser() {
	if g == nil {
		return
	}
	g.multiplayerServers = nil
	g.multiplayerServerScroll = 0
	g.multiplayerSelectedServer = -1
	g.multiplayerDiscoveryDone = false
	g.refreshMultiplayerServers()
}

func (g *Game) refreshMultiplayerServers() {
	if g == nil || g.multiplayerDiscoveryCh != nil {
		return
	}
	ch := make(chan masterServerDiscoveryResult, 2)
	g.multiplayerDiscoveryCh = ch
	g.multiplayerDiscoveryDone = false
	g.multiplayerMessage = "SEARCHING LAN..."
	publicEnabled := experimentalInternetEnabled()
	directoryURL := ""
	g.multiplayerCatalogState = publicCatalogDisabled
	if publicEnabled {
		directoryURL = configuredPublicDirectoryURL()
		g.multiplayerCatalogState = publicCatalogUnconfigured
		if directoryURL != "" {
			g.multiplayerCatalogState = publicCatalogSearching
		}
	}
	go func() {
		type sourceResult struct {
			servers []multiplayerServerInfo
			err     error
		}
		lanCh := make(chan sourceResult, 1)
		publicCh := make(chan sourceResult, 1)
		if publicEnabled {
			go func() {
				servers, err := fetchPublicRooms(directoryURL, nil)
				publicCh <- sourceResult{servers: servers, err: err}
			}()
		}
		go func() {
			servers, err := discoverLANServers(lanDiscoveryTimeout)
			lanCh <- sourceResult{servers: servers, err: err}
		}()

		localServers, localErr := discoverLocalServers()
		lan := <-lanCh
		networkDiagnostic("BROWSER LAN candidates=%d addresses=%v err=%v", len(lan.servers), serverAddressesForDiagnostic(lan.servers), lan.err)

		servers := mergeServerLists(localServers, lan.servers)
		var err error
		if len(servers) == 0 && localErr != nil && lan.err != nil {
			err = errors.Join(localErr, lan.err)
		}
		// LAN renders without waiting for HTTPS. No public request is made unless
		// explicitly opted in; private invitations are never directory lookups.
		ch <- masterServerDiscoveryResult{Servers: servers, Err: err, Done: false}
		if publicEnabled {
			public := <-publicCh
			combined := mergeServerLists(servers, public.servers)
			if len(combined) > 0 {
				err = nil
			}
			// Keep local successes even if the public directory is unavailable.
			ch <- masterServerDiscoveryResult{Servers: combined, MasterErr: public.err, Done: true}
		} else {
			ch <- masterServerDiscoveryResult{Servers: servers, Err: err, Done: true}
		}
	}()
}

func serverAddressesForDiagnostic(servers []multiplayerServerInfo) []string {
	out := make([]string, 0, len(servers))
	for _, server := range servers {
		out = append(out, server.Address)
	}
	return out
}

func (g *Game) filterOwnHostedServer(servers []multiplayerServerInfo) []multiplayerServerInfo {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost || strings.TrimSpace(g.netplay.publicSessionID) == "" {
		return servers
	}
	ownID := strings.TrimSpace(g.netplay.publicSessionID)
	out := servers[:0]
	for _, server := range servers {
		if strings.TrimSpace(server.ID) == ownID {
			continue
		}
		out = append(out, server)
	}
	return out
}

func (g *Game) pollMultiplayerServerBrowser() {
	if g == nil || g.multiplayerDiscoveryCh == nil {
		return
	}
	select {
	case result := <-g.multiplayerDiscoveryCh:
		if result.Done {
			g.multiplayerDiscoveryCh = nil
			g.multiplayerDiscoveryDone = true
			if g.multiplayerCatalogState == publicCatalogSearching {
				g.multiplayerCatalogState = publicCatalogAvailable
				if result.MasterErr != nil {
					g.multiplayerCatalogState = publicCatalogUnavailable
				}
			}
		}
		if result.Err != nil && result.Done {
			g.multiplayerServers = nil
			g.multiplayerMessage = "SERVER SEARCH ERROR: " + result.Err.Error()
			return
		}
		result.Servers = g.filterOwnHostedServer(result.Servers)
		g.multiplayerServers = result.Servers
		g.multiplayerServerScroll = 0
		g.multiplayerSelectedServer = -1
		if !result.Done {
			if len(result.Servers) > 0 {
				g.multiplayerMessage = "FOUND LAN SESSIONS"
			} else {
				g.multiplayerMessage = "SEARCHING LAN..."
			}
			return
		}
		if len(result.Servers) == 0 {
			g.multiplayerMessage = "NO OPEN LAN SESSIONS FOUND"
		} else {
			g.multiplayerMessage = "FOUND OPEN LAN SESSIONS"
		}
		if experimentalInternetEnabled() {
			if result.MasterErr != nil {
				g.multiplayerMessage = "LAN / PRIVATE INVITES AVAILABLE - " + result.MasterErr.Error()
			} else if len(result.Servers) == 0 {
				g.multiplayerMessage = "NO OPEN LAN OR INTERNET ROOMS FOUND"
			} else {
				g.multiplayerMessage = "FOUND OPEN LAN / INTERNET ROOMS"
			}
		}
	default:
	}
}
