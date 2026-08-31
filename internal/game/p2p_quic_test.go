package game

import (
	"context"
	"net"
	"os"
	"strconv"
	"testing"
	"time"
)

func localP2PHostAddress(t *testing.T, h *p2pQUICHost) string {
	t.Helper()
	udp, ok := h.mux.LocalAddr().(*net.UDPAddr)
	if !ok || udp.Port <= 0 {
		t.Fatalf("unexpected P2P local address: %v", h.mux.LocalAddr())
	}
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(udp.Port))
}

func TestP2PQUICDirectOpenSession(t *testing.T) {
	hostSession, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer hostSession.Close()
	key, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	host, err := startP2PQUICHost(hostSession, onlineSessionOpen, key)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	conn, err := dialP2PQUIC(localP2PHostAddress(t, host), "", 4*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client, err := newClientNetplayFromConn(conn, "local P2P")
	if err != nil {
		conn.Close()
		t.Fatal(err)
	}
	defer client.Close()
	if err := waitForClientHandshake(client, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if got := client.assignedPlayer(); got != 2 {
		t.Fatalf("assigned player=%d want 2", got)
	}
}

func TestP2POpenHostServesBrowserMetadataFromHostSocket(t *testing.T) {
	hostSession, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer hostSession.Close()
	key, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	hostSession.publicSessionID = publicOnlineSessionID(key)
	hostSession.publicSessionName = "Host Master Test"
	var allowed [5]bool
	allowed[2] = true
	hostSession.setAllowedRemotePlayers(allowed)
	hostSession.setPublicProbeDetails(netProbeInfo{
		GameMode: SourceGameModeNormal, Lives: 7, Rounds: 3,
		Maps: [customSeriesMaxRounds]int{2, 7, 11}, PlaylistConfigured: true,
		Crates: true, Pickups: true, RuleSummary: "LOW GRAVITY",
	})
	host, err := startP2PQUICHost(hostSession, onlineSessionOpen, key)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	info, reachable, err := probeP2PHostMetadata(ctx, localP2PHostAddress(t, host))
	if err != nil {
		t.Fatal(err)
	}
	if reachable == "" || info.ID != hostSession.publicSessionID || info.Name != "Host Master Test" || info.Players != 1 || info.MaxPlayers != 2 {
		t.Fatalf("host metadata identity=%+v reachable=%q", info, reachable)
	}
	if info.GameMode != SourceGameModeNormal || info.Lives != 7 || info.Rounds != 3 || info.Maps[0] != 2 || info.Maps[1] != 7 || info.Maps[2] != 11 || info.RuleSummary != "LOW GRAVITY" {
		t.Fatalf("host metadata preview=%+v", info)
	}
}

func TestGlobalBrowserVerificationUsesHostMetadata(t *testing.T) {
	hostSession, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer hostSession.Close()
	key, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	hostSession.publicSessionID = publicOnlineSessionID(key)
	hostSession.publicSessionName = "Verified Internet Host"
	var allowed [5]bool
	allowed[2] = true
	hostSession.setAllowedRemotePlayers(allowed)
	hostSession.setPublicProbeDetails(netProbeInfo{
		GameMode: SourceGameModeTeams, Lives: 9, Rounds: 3,
		Maps: [customSeriesMaxRounds]int{4, 8, 12}, PlaylistConfigured: true,
		Crates: true, Pickups: true, RuleSummary: "FAST MOVEMENT",
	})
	host, err := startP2PQUICHost(hostSession, onlineSessionOpen, key)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	_, port, err := net.SplitHostPort(localP2PHostAddress(t, host))
	if err != nil {
		t.Fatal(err)
	}
	candidate := multiplayerServerInfo{
		Name: "GLOBAL OPEN", Address: net.JoinHostPort("localhost", port), Transport: "quic", Sources: "GLOBAL",
		Players: -1, MaxPlayers: maxOnlinePlayers, Protocol: netplayProtocolVersion,
	}
	verified := verifyGlobalOpenServers([]multiplayerServerInfo{candidate}, 3*time.Second)
	if len(verified) != 1 {
		t.Fatalf("verified servers=%+v", verified)
	}
	got := verified[0]
	if got.ID != hostSession.publicSessionID || got.Name != "Verified Internet Host" || got.Players != 1 || got.MaxPlayers != 2 || !got.HasRulePreview() {
		t.Fatalf("verified metadata=%+v", got)
	}
	if got.GameMode != SourceGameModeTeams || got.Lives != 9 || got.Maps[0] != 4 || got.Maps[1] != 8 || got.Maps[2] != 12 || got.RuleSummary != "FAST MOVEMENT" {
		t.Fatalf("verified preview=%+v", got)
	}
}

func TestP2PQUICInviteSessionUsesHMAC(t *testing.T) {
	hostSession, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer hostSession.Close()
	key, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	hostSession.inviteOnly = true
	hostSession.expectedSessionKey = key
	host, err := startP2PQUICHost(hostSession, onlineSessionInviteOnly, key)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	conn, err := dialP2PQUIC(localP2PHostAddress(t, host), key, 4*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client, err := newClientNetplayFromConnWithKey(conn, "local invite P2P", key)
	if err != nil {
		conn.Close()
		t.Fatal(err)
	}
	defer client.Close()
	if err := waitForClientHandshake(client, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if got := client.assignedPlayer(); got != 2 {
		t.Fatalf("assigned player=%d want 2", got)
	}
}

func TestP2PLiveDHTPublishesSharedQUICSocket(t *testing.T) {
	if os.Getenv("GUNMAYHEM_DHT_LIVE_TEST") != "1" {
		t.Skip("set GUNMAYHEM_DHT_LIVE_TEST=1 for public DHT P2P integration test")
	}
	hostSession, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer hostSession.Close()
	key, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	host, err := startP2PQUICHost(hostSession, onlineSessionOpen, key)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	deadline := time.Now().Add(35 * time.Second)
	var endpoint string
	for time.Now().Before(deadline) {
		var ready bool
		endpoint, ready, err = host.Status()
		if ready && endpoint != "" {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if endpoint == "" {
		t.Fatalf("P2P host never learned its public DHT endpoint: %v", err)
	}
	peers, lookupErr := discoverDHTPeerAddresses(globalOpenDHTInfoHashAt(time.Now()), 12*time.Second)
	if lookupErr != nil && len(peers) == 0 {
		t.Fatalf("global OPEN DHT lookup failed: %v", lookupErr)
	}
	observed := host.ObservedEndpoints()
	found := false
	for _, peer := range peers {
		for _, candidate := range observed {
			if peer == candidate {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatalf("global OPEN DHT did not return any observed host endpoint; primary=%s observed=%v peers=%v", endpoint, observed, peers)
	}
	code := decentralizedOpenCode(key)
	codePeers, codeLookupErr := discoverDHTPeerAddresses(decentralizedOpenCodeDHTHash(code), 12*time.Second)
	if codeLookupErr != nil && len(codePeers) == 0 {
		t.Fatalf("short code %s DHT lookup failed: %v", code, codeLookupErr)
	}
	found = false
	for _, peer := range codePeers {
		for _, candidate := range observed {
			if peer == candidate {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatalf("short code %s did not resolve the host; observed=%v peers=%v", code, observed, codePeers)
	}
}

func TestP2PLiveShortInviteCodePublishes(t *testing.T) {
	if os.Getenv("GUNMAYHEM_DHT_LIVE_TEST") != "1" {
		t.Skip("set GUNMAYHEM_DHT_LIVE_TEST=1 for public DHT P2P integration test")
	}
	hostSession, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer hostSession.Close()
	rawKey, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	code := decentralizedOpenCode(rawKey)
	authKey := decentralizedCodeSessionKey(code)
	hostSession.inviteOnly = true
	hostSession.expectedSessionKey = authKey
	hostSession.shortJoinCode = code
	host, err := startP2PQUICHost(hostSession, onlineSessionInviteOnly, authKey)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	deadline := time.Now().Add(35 * time.Second)
	for time.Now().Before(deadline) {
		if _, ready, _ := host.Status(); ready {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if _, ready, statusErr := host.Status(); !ready {
		t.Fatalf("short invite host did not publish: %v", statusErr)
	}
	peers, lookupErr := discoverDHTPeerAddresses(decentralizedOpenCodeDHTHash(code), 12*time.Second)
	if lookupErr != nil && len(peers) == 0 {
		t.Fatalf("short invite code %s lookup failed: %v", code, lookupErr)
	}
	servers := dhtPeerServerCandidates(peers, "TEST INVITE", "JOIN CODE")
	if len(servers) == 0 {
		t.Fatalf("short invite code %s returned no host endpoints", code)
	}
	observed := host.ObservedEndpoints()
	for _, server := range servers {
		for _, endpoint := range observed {
			if server.Address == endpoint {
				return
			}
		}
	}
	t.Fatalf("short invite code %s did not resolve an observed host endpoint: servers=%v observed=%v", code, serverAddressesForDiagnostic(servers), observed)
}
