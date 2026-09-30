package game

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/websocket"
	"gunmayhem-go/internal/masterapi"
)

func newPublicRoomIdentity(reader io.Reader) (id, owner string, err error) {
	if reader == nil {
		return "", "", errors.New("public room randomness unavailable")
	}
	var entropy [48]byte
	if _, err = io.ReadFull(reader, entropy[:]); err != nil {
		return "", "", errors.New("public room randomness unavailable")
	}
	return hex.EncodeToString(entropy[:16]), hex.EncodeToString(entropy[16:]), nil
}

func normalizePublicDirectoryURL(raw string) string {
	if len(raw) > 2048 || raw != strings.TrimSpace(raw) {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" {
		return ""
	}
	return strings.TrimRight(u.String(), "/")
}

func configuredPublicDirectoryURL() string {
	return normalizePublicDirectoryURL(configuredMasterServerURL())
}

func (g *Game) isPublicRoomHost() bool {
	return g != nil && g.netplay != nil && g.netplay.mode == netplayHost && !g.netplay.inviteOnly && masterapi.ValidHexIdentity(g.netplay.publicRoomID, 16)
}

func publicDirectoryClient(client *http.Client) *http.Client {
	copy := http.Client{}
	if client != nil {
		copy = *client
	}
	copy.Timeout = masterRequestTimeout
	// Ownership headers must never be forwarded to a redirect target, including
	// a same-origin path, HTTPS downgrade or a 307/308 replay of registration.
	copy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &copy
}

type publicRoomAdvertiser struct {
	baseURL, owner string
	session        *netplaySession
	tunnel         *cloudflareQuickTunnel
	client         *http.Client
	ctx            context.Context
	cancel         context.CancelFunc
	done           chan struct{}
	closed         atomic.Bool
	terminal       atomic.Bool
	mu             sync.RWMutex
	status         string
}

func startPublicRoomAdvertiser(baseURL, owner string, s *netplaySession, tunnel *cloudflareQuickTunnel, client *http.Client) *publicRoomAdvertiser {
	baseURL = normalizePublicDirectoryURL(baseURL)
	if baseURL == "" || !masterapi.ValidHexIdentity(owner, 32) || s == nil || s.mode != netplayHost || s.inviteOnly || s.closed.Load() || !masterapi.ValidHexIdentity(s.publicRoomID, 16) || tunnel == nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	a := &publicRoomAdvertiser{baseURL: baseURL, owner: owner, session: s, tunnel: tunnel, client: publicDirectoryClient(client), ctx: ctx, cancel: cancel, done: make(chan struct{}), status: "PUBLIC ROOM: REGISTERING"}
	go a.loop()
	return a
}

func (a *publicRoomAdvertiser) setStatus(status string) {
	a.mu.Lock()
	a.status = status
	a.mu.Unlock()
}
func (a *publicRoomAdvertiser) Status() string {
	if a == nil {
		return "PUBLIC ROOM: UNLISTED"
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

func (a *publicRoomAdvertiser) loop() {
	defer close(a.done)
	a.heartbeat()
	heartbeat := time.NewTicker(masterHeartbeatInterval)
	defer heartbeat.Stop()
	health := time.NewTicker(time.Second)
	defer health.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-heartbeat.C:
			if !a.terminal.Load() {
				a.heartbeat()
			}
		case <-health.C:
			if a.terminal.Load() {
				continue
			}
			if ready, _, _, err := a.tunnel.Status(); !ready || err != nil {
				a.terminal.Store(true)
				a.session.publicAdmissionClosed.Store(true)
				a.setStatus("PUBLIC ROOM CLOSED: TUNNEL STOPPED")
				_ = a.remove()
			}
		}
	}
}

func publicRoomDescriptor(s *netplaySession, endpoint string) masterapi.PublicRoom {
	info := s.publicProbeInfo()
	return masterapi.PublicRoom{ID: s.publicRoomID, Endpoint: endpoint, Name: info.Name, Protocol: netplayProtocolVersion, Players: 1 + s.connectedCount(), MaxPlayers: s.humanCapacity(), GameMode: info.GameMode, Lives: info.Lives, Rounds: info.Rounds, Maps: info.Maps, PlaylistConfigured: info.PlaylistConfigured, Crates: info.Crates, Pickups: info.Pickups, RuleSummary: info.RuleSummary, ChaosEnabled: info.ChaosEnabled, ChaosEvents: info.ChaosEvents, ChaosIntervalSeconds: info.ChaosIntervalSeconds}
}

func (a *publicRoomAdvertiser) heartbeat() {
	if a.closed.Load() || a.terminal.Load() {
		return
	}
	ready, _, endpoint, tunnelErr := a.tunnel.Status()
	if !ready || tunnelErr != nil || a.session.closed.Load() || a.session.inviteOnly || a.session.publicAdmissionClosed.Load() {
		return
	}
	room := publicRoomDescriptor(a.session, endpoint)
	if masterapi.ValidatePublicRoom(room) != nil {
		a.setStatus("PUBLIC ROOM: INVALID PREVIEW")
		return
	}
	body, err := json.Marshal(room)
	if err != nil {
		a.setStatus("PUBLIC ROOM: INVALID PREVIEW")
		return
	}
	req, err := http.NewRequestWithContext(a.ctx, http.MethodPost, a.baseURL+masterapi.PublicRoomsPath, bytes.NewReader(body))
	if err != nil {
		a.setStatus("PUBLIC ROOM: DIRECTORY ERROR")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(masterapi.PublicRoomOwnerHeader, a.owner)
	resp, err := a.client.Do(req)
	if err != nil {
		if !a.closed.Load() {
			a.setStatus("PUBLIC ROOM: DIRECTORY UNAVAILABLE")
		}
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusUnauthorized {
		a.terminal.Store(true)
		a.session.publicAdmissionClosed.Store(true)
		a.setStatus("PUBLIC ROOM CLOSED: RECREATE SESSION")
		return
	}
	if resp.StatusCode != http.StatusOK {
		a.setStatus("PUBLIC ROOM: DIRECTORY REJECTED UPDATE")
		return
	}
	var result struct {
		ID string `json:"id"`
	}
	if decodeLimitedJSON(resp.Body, 1024, &result) != nil || result.ID != a.session.publicRoomID {
		a.setStatus("PUBLIC ROOM: INVALID DIRECTORY REPLY")
		return
	}
	a.setStatus("PUBLIC ROOM: LISTED")
}

func (a *publicRoomAdvertiser) remove() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, a.baseURL+masterapi.PublicRoomsPath+"?id="+url.QueryEscape(a.session.publicRoomID), nil)
	if err != nil {
		return errors.New("public room deletion failed")
	}
	req.Header.Set(masterapi.PublicRoomOwnerHeader, a.owner)
	resp, err := a.client.Do(req)
	if err != nil {
		return errors.New("public room deletion failed; listing expires by TTL")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return errors.New("public room deletion rejected; listing expires by TTL")
	}
	return nil
}

func (a *publicRoomAdvertiser) Close() error {
	if a == nil || a.closed.Swap(true) {
		return nil
	}
	a.session.publicAdmissionClosed.Store(true)
	a.cancel()
	<-a.done // no heartbeat may start after the final deletion
	a.setStatus("PUBLIC ROOM: CLOSED")
	return a.remove() // server tombstone covers a canceled initial registration
}

func fetchPublicRooms(baseURL string, client *http.Client) ([]multiplayerServerInfo, error) {
	baseURL = normalizePublicDirectoryURL(baseURL)
	if baseURL == "" {
		return nil, errors.New("PUBLIC DIRECTORY NOT CONFIGURED")
	}
	client = publicDirectoryClient(client)
	resp, err := client.Get(baseURL + masterapi.PublicRoomsPath + "?protocol=" + strconv.Itoa(netplayProtocolVersion))
	if err != nil {
		return nil, errors.New("PUBLIC DIRECTORY UNAVAILABLE")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("PUBLIC DIRECTORY REQUEST REJECTED")
	}
	var list masterapi.PublicRoomList
	if decodeLimitedJSON(resp.Body, masterapi.MaxPublicRoomListBody, &list) != nil || len(list.Rooms) > masterapi.MaxPublicRooms {
		return nil, errors.New("INVALID PUBLIC DIRECTORY RESPONSE")
	}
	servers := make([]multiplayerServerInfo, 0, len(list.Rooms))
	seen := make(map[string]bool)
	for _, room := range list.Rooms {
		if masterapi.ValidatePublicRoom(room) != nil || room.Protocol != netplayProtocolVersion || seen[room.ID] {
			return nil, errors.New("INVALID PUBLIC ROOM LIST")
		}
		seen[room.ID] = true
		servers = append(servers, multiplayerServerInfo{ID: room.ID, Name: room.Name, Address: room.Endpoint, TunnelAddress: room.Endpoint, Transport: "public-wss", Sources: "INTERNET", Players: room.Players, MaxPlayers: room.MaxPlayers, Protocol: room.Protocol, GameMode: room.GameMode, Lives: room.Lives, Rounds: room.Rounds, Maps: room.Maps, PlaylistConfigured: room.PlaylistConfigured, Crates: room.Crates, Pickups: room.Pickups, RuleSummary: room.RuleSummary, ChaosEnabled: room.ChaosEnabled, ChaosEvents: room.ChaosEvents, ChaosIntervalSeconds: room.ChaosIntervalSeconds})
	}
	return servers, nil
}

func dialPublicRoomWebSocket(endpoint string) (net.Conn, error) {
	config, err := websocket.NewConfig(endpoint, "https://gunmayhem.invalid/")
	if err != nil {
		return nil, err
	}
	config.Protocol = []string{cloudflareNetplayProtocol}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	conn, err := config.DialContext(ctx)
	if err != nil {
		return nil, errors.New("public room connection failed")
	}
	conn.PayloadType = websocket.BinaryFrame
	return newTrackedWebSocketConn(conn), nil
}

func newPublicRoomClient(server multiplayerServerInfo, dial func(string) (net.Conn, error)) (*netplaySession, error) {
	if server.Transport != "public-wss" || server.Protocol != netplayProtocolVersion || server.Full() {
		return nil, errors.New("public room unavailable or incompatible")
	}
	endpoint, err := masterapi.NormalizePublicRoomEndpoint(server.Address, server.ID)
	if err != nil || endpoint != server.Address || server.TunnelAddress != endpoint || server.RelayAddress != "" || server.DirectAddress != "" || server.Code != "" {
		return nil, errors.New("invalid public room route")
	}
	conn, err := dial(endpoint)
	if err != nil {
		return nil, err
	}
	s, err := newClientNetplayFromConnWithKey(conn, "Public room "+server.ID, "")
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err = waitForClientHandshake(s, netHandshakeTimeout+time.Second); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

// Keep entropy generation separate from the private-invite key and UI state.
func publicRoomIdentity() (string, string, error) { return newPublicRoomIdentity(rand.Reader) }
