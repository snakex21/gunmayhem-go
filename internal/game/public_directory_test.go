package game

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/websocket"
	"gunmayhem-go/internal/masterapi"
	"gunmayhem-go/internal/publicrooms"
)

func testPublicRoom(t *testing.T) (*netplaySession, *cloudflareQuickTunnel, string) {
	t.Helper()
	id, owner, err := publicRoomIdentity()
	if err != nil {
		t.Fatal(err)
	}
	s, err := newConfiguredHostNetplay("127.0.0.1:0", nil, func(s *netplaySession) {
		s.publicRoomID = id
		s.publicSessionID = id
		s.publicSessionName = "Local QA Room"
		s.expectedSessionKey = "PRIVATE-KEY-MUST-NOT-LEAK"
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	gateway, err := startCloudflareGateway(s)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { gateway.Close() })
	gateway.running.Store(true)
	gateway.setPublicURL("https://synthetic-room.trycloudflare.com")
	return s, gateway, owner
}

func testPublicRoomDial(t *testing.T, gateway *cloudflareQuickTunnel) (func(string) (net.Conn, error), *httptest.Server) {
	t.Helper()
	server := httptest.NewTLSServer(gateway.server.Handler)
	t.Cleanup(server.Close)
	return func(endpoint string) (net.Conn, error) {
		if _, err := masterapi.NormalizePublicRoomEndpoint(endpoint, gateway.session.publicRoomID); err != nil {
			return nil, err
		}
		config, err := websocket.NewConfig("wss"+strings.TrimPrefix(server.URL, "https")+masterapi.PublicRoomWebSocketPrefix+gateway.session.publicRoomID, "https://gunmayhem.invalid/")
		if err != nil {
			return nil, err
		}
		config.Protocol = []string{cloudflareNetplayProtocol}
		config.TlsConfig = server.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		conn, err := config.DialContext(ctx)
		if err != nil {
			return nil, err
		}
		conn.PayloadType = websocket.BinaryFrame
		return newTrackedWebSocketConn(conn), nil
	}, server
}

func TestPublicRoomDirectoryWSSFourPlayersInputSnapshotsKickRejoinClose(t *testing.T) {
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "")
	s, gateway, owner := testPublicRoom(t)
	directory := httptest.NewTLSServer(publicrooms.New(0, false).Handler())
	defer directory.Close()
	a := startPublicRoomAdvertiser(directory.URL, owner, s, gateway, directory.Client())
	if a == nil {
		t.Fatal("advertiser rejected fixture")
	}
	defer a.Close()
	waitFor(t, "public registration", func() bool { return a.Status() == "PUBLIC ROOM: LISTED" })
	rooms, err := fetchPublicRooms(directory.URL, directory.Client())
	if err != nil || len(rooms) != 1 {
		t.Fatalf("discovery: rooms=%d err=%v", len(rooms), err)
	}
	if rooms[0].Transport != "public-wss" || rooms[0].Sources != "INTERNET" || rooms[0].ID != s.publicRoomID {
		t.Fatal("public list route wrong")
	}
	encoded, _ := json.Marshal(rooms)
	for _, secret := range []string{owner, s.expectedSessionKey, "GMT1-", "127.0.0.1"} {
		if bytes.Contains(encoded, []byte(secret)) {
			t.Fatal("private data leaked in discovery")
		}
	}
	dial, tlsGateway := testPublicRoomDial(t, gateway)
	clients := make([]*netplaySession, 0, 3)
	for i := 0; i < 3; i++ {
		c, err := newPublicRoomClient(rooms[0], dial)
		if err != nil {
			t.Fatal(err)
		}
		clients = append(clients, c)
		defer c.Close()
	}
	if c, err := newPublicRoomClient(rooms[0], dial); err == nil {
		c.Close()
		t.Fatal("fourth guest admitted")
	}
	if s.connectedCount() != 3 {
		t.Fatal("full room corrupted slots")
	}
	clients[0].queueInput(PlayerInputState{Right: true, Grenade: true})
	waitFor(t, "public guest input", func() bool { return s.latestInputFor(2).Right && s.latestInputFor(2).Grenade })
	s.queueState(netSnapshot{TotalLives: 13, Players: []Player{{ID: 1, X: 10}, {ID: 2, X: 70}}})
	waitFor(t, "public authoritative snapshot", func() bool {
		state, ok := clients[0].latestState()
		return ok && state.TotalLives == 13 && len(state.Players) == 2 && state.Players[1].X == 70
	})
	a.heartbeat()
	rooms, err = fetchPublicRooms(directory.URL, directory.Client())
	if err != nil || !rooms[0].Full() {
		t.Fatal("full occupancy not refreshed")
	}
	allowed := [5]bool{false, false, false, true, true}
	s.setAllowedRemotePlayers(allowed)
	waitFor(t, "kick releases peer", func() bool { return s.connectedCount() == 2 && !clients[0].connected() })
	if s.latestInputFor(2).Right {
		t.Fatal("kicked peer input stuck")
	}
	allowed[2] = true
	s.setAllowedRemotePlayers(allowed)
	a.heartbeat()
	rooms, err = fetchPublicRooms(directory.URL, directory.Client())
	if err != nil {
		t.Fatal(err)
	}
	rejoin, err := newPublicRoomClient(rooms[0], dial)
	if err != nil {
		t.Fatal(err)
	}
	defer rejoin.Close()
	if rejoin.assignedPlayer() != 2 {
		t.Fatal("freed slot not reusable")
	}
	// A wrong public routing ID, private /ws path and query-bearing route never
	// reach netplay admission even though no secret is required for OPEN rooms.
	for _, path := range []string{cloudflareNetplayPath, masterapi.PublicRoomWebSocketPrefix + strings.Repeat("0", 32), masterapi.PublicRoomWebSocketPrefix + s.publicRoomID + "?private=1"} {
		cfg, _ := websocket.NewConfig("wss"+strings.TrimPrefix(tlsGateway.URL, "https")+path, "https://gunmayhem.invalid/")
		cfg.Protocol = []string{cloudflareNetplayProtocol}
		cfg.TlsConfig = tlsGateway.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		conn, err := cfg.DialContext(ctx)
		cancel()
		if err == nil {
			conn.Close()
			t.Fatal("wrong room route accepted")
		}
	}
	g := &Game{netplay: s, cloudflareTunnel: gateway, publicDirectory: a, customOnlineMode: true, customPage: customPagePlayers}
	if hit := g.customV2HitAt(customV2CopyInviteRect.X+2, customV2CopyInviteRect.Y+2); hit != menuHitCustomV2ClosePublicRoom {
		t.Fatal("CLOSE ROOM control missing")
	}
	g.activateCustomV2Hit(menuHitCustomV2ClosePublicRoom)
	if g.netplay != nil || g.publicDirectory != nil || !g.customNetworkOpen {
		t.Fatal("CLOSE ROOM action did not return to browser")
	}
	waitFor(t, "close disconnects guests", func() bool { return !rejoin.connected() && !clients[1].connected() })
	rooms, err = fetchPublicRooms(directory.URL, directory.Client())
	if err != nil || len(rooms) != 0 {
		t.Fatal("close left public listing")
	}
	if !s.publicAdmissionClosed.Load() || !s.closed.Load() {
		t.Fatal("close left admission open")
	}
}

func TestPublicRoomPrivateAndPolicyIsolation(t *testing.T) {
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "")
	t.Setenv("GUNMAYHEM_MASTER_URL", "https://directory.example.invalid")
	g := New()
	if err := g.StartNetHostConfig("127.0.0.1:0", onlineSessionScopeInternet, onlineSessionOpen); err == nil {
		t.Fatal("public host without opt-in")
	}
	if g.netplay != nil || g.cloudflareTunnel != nil || g.publicDirectory != nil {
		t.Fatal("rejected host started services")
	}
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "1")
	if err := g.StartNetHost("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer g.CloseNetplay()
	old := g.netplay
	t.Setenv("GUNMAYHEM_MASTER_URL", "https://user:secret@directory.example.invalid")
	if err := g.StartNetHostConfig("127.0.0.1:0", onlineSessionScopeInternet, onlineSessionOpen); err == nil || g.netplay != old {
		t.Fatal("invalid directory destroyed active host")
	}
	t.Setenv("GUNMAYHEM_MASTER_URL", "https://directory.example.invalid")
	g.activateCustomHostSetupHit(menuHitCustomHostScopeInternet)
	if g.customHostAccess != onlineSessionInviteOnly {
		t.Fatal("Internet did not default private")
	}
	g.activateCustomHostSetupHit(menuHitCustomHostAccessOpen)
	if g.customHostAccess != onlineSessionOpen {
		t.Fatal("explicit configured public choice rejected")
	}
	if err := g.StartNetHostConfig("127.0.0.1:0", onlineSessionScopeInternet, onlineSessionInviteOnly); err != nil {
		t.Fatal(err)
	}
	if g.publicDirectory != nil || g.netplay.publicRoomID != "" || !g.netplay.inviteOnly || len(g.onlineSessionKey) != 26 {
		t.Fatal("private invitation changed")
	}
	if a := startPublicRoomAdvertiser("https://directory.example.invalid", strings.Repeat("a", 64), g.netplay, &cloudflareQuickTunnel{}, nil); a != nil {
		a.Close()
		t.Fatal("private session accepted by advertiser")
	}
	if g.serverAdvertiser != nil || g.p2pHost != nil || g.globalDHTAdvertiser != nil || g.natPortMapper != nil {
		t.Fatal("legacy Internet service enabled")
	}
}

func TestPublicRoomIdentityFailsClosed(t *testing.T) {
	if id, owner, err := newPublicRoomIdentity(nil); err == nil || id != "" || owner != "" {
		t.Fatal("nil entropy reader accepted")
	}
	id, owner, err := newPublicRoomIdentity(bytes.NewReader(make([]byte, 47)))
	if err == nil || id != "" || owner != "" {
		t.Fatal("entropy failure did not fail closed")
	}
	id, owner, err = newPublicRoomIdentity(bytes.NewReader(bytes.Repeat([]byte{1}, 48)))
	if err != nil || !masterapi.ValidHexIdentity(id, 16) || !masterapi.ValidHexIdentity(owner, 32) || len(owner) != 64 {
		t.Fatal("identity sizes wrong")
	}
}

func TestPublicRoomDirectoryConfigurationAndUntrustedRoutes(t *testing.T) {
	for _, raw := range []string{"", "https://:443", "http://localhost:7780", "https://u:p@example.com", "https://example.com?token=x", "https://example.com#x", "https://example.com?", "https://example.com/%2f"} {
		if normalizePublicDirectoryURL(raw) != "" {
			t.Fatal("unsafe directory config accepted")
		}
	}
	if normalizePublicDirectoryURL("https://example.com/base/") != "https://example.com/base" {
		t.Fatal("HTTPS directory rejected")
	}
	id := strings.Repeat("a", 32)
	endpoint := "wss://example.trycloudflare.com" + masterapi.PublicRoomWebSocketPrefix + id
	valid := multiplayerServerInfo{ID: id, Address: endpoint, TunnelAddress: endpoint, Transport: "public-wss", Protocol: netplayProtocolVersion, Players: 1, MaxPlayers: 4}
	for _, change := range []func(*multiplayerServerInfo){func(s *multiplayerServerInfo) { s.Address = "wss://127.0.0.1/ws" }, func(s *multiplayerServerInfo) { s.ID = strings.Repeat("b", 32) }, func(s *multiplayerServerInfo) { s.RelayAddress = "127.0.0.1:1" }, func(s *multiplayerServerInfo) { s.Code = "private" }, func(s *multiplayerServerInfo) { s.Protocol-- }, func(s *multiplayerServerInfo) { s.Players = 4 }} {
		s := valid
		change(&s)
		if _, err := newPublicRoomClient(s, func(string) (net.Conn, error) { t.Fatal("untrusted route was dialed"); return nil, nil }); err == nil {
			t.Fatal("invalid public route accepted")
		}
	}
	lan := applyProbeInfoToServer(multiplayerServerInfo{Address: "192.168.1.4:7777", Transport: "lan"}, netProbeInfo{ID: id, TunnelAddress: endpoint})
	if lan.Transport != "lan" || lan.Address != "192.168.1.4:7777" || lan.TunnelAddress != "" {
		t.Fatal("LAN route was replaced by Internet")
	}
}

func TestPublicRoomDirectoryRedirectCannotLeakOwnership(t *testing.T) {
	s, gateway, owner := testPublicRoom(t)
	var targetCalls atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetCalls.Add(1); w.WriteHeader(http.StatusNoContent) }))
	defer target.Close()
	redirect := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	a := startPublicRoomAdvertiser(redirect.URL, owner, s, gateway, redirect.Client())
	defer a.Close()
	waitFor(t, "redirect rejected", func() bool { return a.Status() == "PUBLIC ROOM: DIRECTORY REJECTED UPDATE" })
	if _, err := fetchPublicRooms(redirect.URL, redirect.Client()); err == nil {
		t.Fatal("list redirect accepted")
	}
	if err := a.Close(); err == nil {
		t.Fatal("delete redirect accepted")
	}
	if targetCalls.Load() != 0 {
		t.Fatal("directory redirect forwarded request")
	}
}

func TestPublicRoomCloseBeforeDelayedRegistrationCannotResurrect(t *testing.T) {
	s, gateway, owner := testPublicRoom(t)
	inner := publicrooms.New(0, false).Handler()
	entered := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	directory := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))
			close(entered)
			<-release
			inner.ServeHTTP(w, r)
			close(finished)
			return
		}
		inner.ServeHTTP(w, r)
	}))
	defer directory.Close()
	a := startPublicRoomAdvertiser(directory.URL, owner, s, gateway, directory.Client())
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("registration not received")
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	close(release)
	<-finished
	rooms, err := fetchPublicRooms(directory.URL, directory.Client())
	if err != nil || len(rooms) != 0 {
		t.Fatal("late registration resurrected closed room")
	}
}

func TestPublicRoomTunnelFailureDelistsAndClosesAdmission(t *testing.T) {
	s, gateway, owner := testPublicRoom(t)
	directory := httptest.NewTLSServer(publicrooms.New(0, false).Handler())
	defer directory.Close()
	a := startPublicRoomAdvertiser(directory.URL, owner, s, gateway, directory.Client())
	defer a.Close()
	waitFor(t, "registered", func() bool { return a.Status() == "PUBLIC ROOM: LISTED" })
	gateway.running.Store(false)
	waitFor(t, "dead tunnel closes admission", func() bool { return s.publicAdmissionClosed.Load() })
	waitFor(t, "dead tunnel delists", func() bool {
		rooms, err := fetchPublicRooms(directory.URL, directory.Client())
		return err == nil && len(rooms) == 0
	})
	if !strings.Contains(a.Status(), "TUNNEL STOPPED") {
		t.Fatal("tunnel failure not visible")
	}
}

func TestPublicRoomGatewayListenerFailureDelistsWhileTunnelProcessRuns(t *testing.T) {
	s, gateway, owner := testPublicRoom(t)
	directory := httptest.NewTLSServer(publicrooms.New(0, false).Handler())
	defer directory.Close()
	a := startPublicRoomAdvertiser(directory.URL, owner, s, gateway, directory.Client())
	defer a.Close()
	waitFor(t, "registered", func() bool { return a.Status() == "PUBLIC ROOM: LISTED" })
	if err := gateway.listener.Close(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "local gateway error recorded", func() bool { _, _, _, err := gateway.Status(); return err != nil })
	if !gateway.running.Load() {
		t.Fatal("fixture should retain running process state")
	}
	a.heartbeat()
	waitFor(t, "failed gateway closes admission", func() bool { return s.publicAdmissionClosed.Load() })
	waitFor(t, "failed gateway delists", func() bool {
		rooms, err := fetchPublicRooms(directory.URL, directory.Client())
		return err == nil && len(rooms) == 0
	})
	if !a.terminal.Load() {
		t.Fatal("failed gateway remained renewable")
	}
}

func TestPublicRoomAccessWarningTracksActualChoice(t *testing.T) {
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "1")
	t.Setenv("GUNMAYHEM_MASTER_URL", "https://directory.example.invalid")
	g := &Game{}
	g.activateCustomHostSetupHit(menuHitCustomHostScopeInternet)
	g.activateCustomHostSetupHit(menuHitCustomHostAccessOpen)
	if !strings.HasPrefix(g.multiplayerMessage, "OPEN INTERNET ROOM:") {
		t.Fatal("public warning missing")
	}
	g.activateCustomHostSetupHit(menuHitCustomHostAccessInvite)
	if strings.HasPrefix(g.multiplayerMessage, "OPEN INTERNET ROOM:") || g.customHostAccess != onlineSessionInviteOnly {
		t.Fatal("public warning survived private selection")
	}
	g.activateCustomHostSetupHit(menuHitCustomHostAccessOpen)
	g.activateCustomHostSetupHit(menuHitCustomHostScopeLAN)
	if strings.HasPrefix(g.multiplayerMessage, "OPEN INTERNET ROOM:") || g.customHostScope != onlineSessionScopeLAN {
		t.Fatal("public warning survived LAN selection")
	}
}

func TestPublicRoomDirectoryExpiryIsTerminalAndDeleteFailureUsesTTL(t *testing.T) {
	s, gateway, owner := testPublicRoom(t)
	inner := publicrooms.New(30*time.Millisecond, false).Handler()
	directory := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		inner.ServeHTTP(w, r)
	}))
	defer directory.Close()
	a := startPublicRoomAdvertiser(directory.URL, owner, s, gateway, directory.Client())
	defer a.Close()
	waitFor(t, "registered", func() bool { return a.Status() == "PUBLIC ROOM: LISTED" })
	time.Sleep(45 * time.Millisecond)
	rooms, err := fetchPublicRooms(directory.URL, directory.Client())
	if err != nil || len(rooms) != 0 {
		t.Fatal("TTL failed to remove room")
	}
	a.heartbeat()
	if !a.terminal.Load() || !s.publicAdmissionClosed.Load() {
		t.Fatal("expired room resumed silently")
	}
	if err := a.Close(); err == nil {
		t.Fatal("failed DELETE not reported")
	}
}

func TestPublicRoomListFailurePreservesLocalResults(t *testing.T) {
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "1")
	g := &Game{multiplayerDiscoveryCh: make(chan masterServerDiscoveryResult, 1)}
	g.multiplayerDiscoveryCh <- masterServerDiscoveryResult{Servers: []multiplayerServerInfo{{Address: "192.168.1.2:7777", Transport: "lan"}}, MasterErr: errors.New("PUBLIC DIRECTORY UNAVAILABLE"), Done: true}
	g.pollMultiplayerServerBrowser()
	if len(g.multiplayerServers) != 1 || !strings.Contains(g.multiplayerMessage, "UNAVAILABLE") || g.multiplayerDiscoveryCh != nil {
		t.Fatal("directory outage discarded LAN results")
	}
}
