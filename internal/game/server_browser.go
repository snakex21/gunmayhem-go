package game

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	localServerStaleAfter       = 7 * time.Second
	internetServerVerifyTimeout = 12 * time.Second
)

type multiplayerServerInfo struct {
	ID                   string
	Code                 string
	Name                 string
	Address              string
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
	if server.TunnelAddress != "" {
		server.Transport = "cloudflare"
		server.Address = server.TunnelAddress
	}
	return server
}

type serverDiscoveryResult struct {
	Servers []multiplayerServerInfo
	Err     error
	DHTErr  error
	Done    bool
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

func (g *Game) beginMultiplayerServerBrowser() {
	if g == nil {
		return
	}
	// A previous zombie session's flag must not turn the next hosted session
	// into a zombie one (HOST ONLINE sets it again after this call).
	g.onlineZombie = false
	if g.zombie != nil && !g.zombie.host {
		g.zombie = nil
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
	ch := make(chan serverDiscoveryResult, 2)
	g.multiplayerDiscoveryCh = ch
	g.setMultiplayerBrowserStatus("SEARCHING NETWORK...")
	go func() {
		type sourceResult struct {
			servers []multiplayerServerInfo
			err     error
		}
		lanCh := make(chan sourceResult, 1)
		go func() {
			servers, err := discoverLANServers(lanDiscoveryTimeout)
			lanCh <- sourceResult{servers: servers, err: err}
		}()

		localServers, localErr := discoverLocalServers()
		lan := <-lanCh
		networkDiagnostic("BROWSER LAN candidates=%d addresses=%v err=%v", len(lan.servers), serverAddressesForDiagnostic(lan.servers), lan.err)

		servers := mergeServerLists(localServers, lan.servers)
		var lanErr error
		if len(servers) == 0 && localErr != nil && lan.err != nil {
			lanErr = errors.Join(localErr, lan.err)
		}
		// Publish LAN/local results immediately. Internet DHT discovery is slower
		// and must never make a LAN host appear frozen in the browser.
		ch <- serverDiscoveryResult{Servers: servers, Err: lanErr, Done: false}

		internetCandidates, dhtErr := discoverGlobalOpenServers(globalDHTLookupTimeout)
		if len(internetCandidates) > 0 {
			internetCandidates = verifyGlobalOpenServers(internetCandidates, internetServerVerifyTimeout)
		}
		// Log the internet/OPEN half of discovery separately from LAN. Without this
		// line a silent DHT failure is indistinguishable from "nobody is hosting",
		// because the LAN line is published first and always reports 0.
		networkDiagnostic("BROWSER global candidates=%d addresses=%v err=%v", len(internetCandidates), serverAddressesForDiagnostic(internetCandidates), dhtErr)
		if len(internetCandidates) > 0 {
			servers = mergeServerLists(servers, internetCandidates)
		}
		networkDiagnostic("BROWSER merged candidates=%d addresses=%v", len(servers), serverAddressesForDiagnostic(servers))

		finalErr := lanErr
		if len(servers) == 0 && dhtErr != nil {
			finalErr = errors.Join(finalErr, dhtErr)
		}
		ch <- serverDiscoveryResult{Servers: servers, Err: finalErr, DHTErr: dhtErr, Done: true}
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
		}
		if result.Err != nil && result.Done {
			g.multiplayerServers = nil
			g.setMultiplayerBrowserStatus("SERVER SEARCH ERROR: " + result.Err.Error())
			return
		}
		result.Servers = g.filterOwnHostedServer(result.Servers)
		g.multiplayerServers = result.Servers
		g.multiplayerServerScroll = 0
		g.multiplayerSelectedServer = -1
		if !result.Done {
			if len(result.Servers) > 0 {
				g.setMultiplayerBrowserStatus("FOUND LAN SESSIONS - SEARCHING INTERNET...")
			} else {
				g.setMultiplayerBrowserStatus("SEARCHING INTERNET...")
			}
			return
		}
		if len(result.Servers) == 0 {
			if result.DHTErr != nil {
				g.setMultiplayerBrowserStatus("NO OPEN SESSIONS - INTERNET DISCOVERY UNAVAILABLE")
			} else {
				g.setMultiplayerBrowserStatus("NO OPEN SESSIONS FOUND")
			}
		} else {
			g.setMultiplayerBrowserStatus("FOUND OPEN SESSIONS")
		}
	default:
	}
}

func defaultPublicServerName() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		return "Gun Mayhem Host"
	}
	return strings.TrimSpace(host)
}
