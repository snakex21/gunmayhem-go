package game

import (
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestShortOpenCodeRoundTrip(t *testing.T) {
	code, err := encodeShortOpenCode(netip.MustParseAddr("203.0.113.44"), 45123)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(code, shortOpenCodePrefix) || len(code) != len(shortOpenCodePrefix)+10 {
		t.Fatalf("unexpected short code %q len=%d", code, len(code))
	}
	address, err := decodeShortOpenCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if address != "203.0.113.44:45123" {
		t.Fatalf("decoded address=%q", address)
	}
}

func TestDecentralizedOpenCodeIsShortStableAndDoesNotExposeKey(t *testing.T) {
	key := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	a := decentralizedOpenCode(key)
	b := decentralizedOpenCode(key)
	if len(a) != 6 || a != b || strings.Contains(key, a) || !validDecentralizedOpenCode(a) {
		t.Fatalf("decentralized code=%q repeat=%q", a, b)
	}
	if decentralizedOpenCode("ZYXWVUTSRQPONMLKJIHGFEDCBA") == a {
		t.Fatal("different session keys produced the same test code")
	}
}

func TestJoinOpenHostByShortCode(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	port := host.listener.Addr().(*net.TCPAddr).Port
	code, err := encodeShortOpenCode(netip.MustParseAddr("127.0.0.1"), uint16(port))
	if err != nil {
		t.Fatal(err)
	}
	clientGame := &Game{}
	defer clientGame.CloseNetplay()
	if err := clientGame.StartNetClientShortOpenCode(code); err != nil {
		t.Fatalf("join by short OPEN code failed: %v", err)
	}
	if clientGame.netplay == nil || clientGame.netplay.assignedPlayer() < 2 {
		t.Fatal("short code join did not establish a player session")
	}
}

func TestShortCodeAuthenticatesInviteHost(t *testing.T) {
	hostGame := New()
	if err := hostGame.StartNetHostConfig("127.0.0.1:0", onlineSessionScopeLAN, onlineSessionInviteOnly); err != nil {
		t.Fatal(err)
	}
	defer hostGame.CloseNetplay()
	code, ready := hostGame.currentHostDisplayJoinCode()
	if !ready || len(code) != len(shortOpenCodePrefix)+6 {
		t.Fatalf("invite short code=%q ready=%v", code, ready)
	}
	encoded := strings.TrimPrefix(code, shortOpenCodePrefix)
	clientGame := &Game{}
	defer clientGame.CloseNetplay()
	server := multiplayerServerInfo{Address: hostGame.netplay.listener.Addr().String(), Transport: "lan"}
	if err := clientGame.startNetClientCodeServer(server, decentralizedCodeSessionKey(encoded)); err != nil {
		t.Fatalf("short invite code authentication failed: %v", err)
	}
	if clientGame.netplay == nil || clientGame.netplay.assignedPlayer() < 2 {
		t.Fatal("short invite code did not establish a player session")
	}
}

func TestInviteProofDoesNotExposeSessionKey(t *testing.T) {
	key, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := newOnlineAuthChallenge()
	if err != nil {
		t.Fatal(err)
	}
	proof, err := onlineSessionProof(key, challenge)
	if err != nil {
		t.Fatal(err)
	}
	if proof == key || strings.Contains(proof, key) {
		t.Fatal("authentication proof exposed the session key")
	}
	if !onlineSessionProofEqual(key, challenge, proof) {
		t.Fatal("valid authentication proof was rejected")
	}
	otherKey, _ := newOnlineSessionKey()
	if onlineSessionProofEqual(otherKey, challenge, proof) {
		t.Fatal("proof from another session key was accepted")
	}
}

func TestInviteOnlyHostRequiresSessionKey(t *testing.T) {
	key, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	host.inviteOnly = true
	host.expectedSessionKey = key
	address := host.listener.Addr().String()

	wrong, err := newClientNetplayWithKey(address, "", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer wrong.Close()
	if err := waitForClientHandshake(wrong, 700*time.Millisecond); err == nil {
		t.Fatal("invite-only host accepted a client without the session key")
	}
	if got := host.connectedCount(); got != 0 {
		t.Fatalf("unauthorized client occupied a player slot: %d", got)
	}

	good, err := newClientNetplayWithKey(address, key, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer good.Close()
	if err := waitForClientHandshake(good, 2*time.Second); err != nil {
		t.Fatalf("authorized invite client failed: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for host.connectedCount() != 1 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := host.connectedCount(); got != 1 {
		t.Fatalf("authorized client not connected: %d", got)
	}
}

func TestOpenHostAcceptsClientWithoutSessionKey(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	client, err := newClientNetplayWithKey(host.listener.Addr().String(), "", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := waitForClientHandshake(client, 2*time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestCGNATAddressIsNotInternetReady(t *testing.T) {
	if isInternetRoutableAddr(netip.MustParseAddr("100.64.12.34")) {
		t.Fatal("CGNAT address was treated as publicly routable")
	}
	if !isInternetRoutableAddr(netip.MustParseAddr("8.8.8.8")) {
		t.Fatal("public IPv4 was not treated as Internet-routable")
	}
}

func TestShortOpenCodeRejectsGarbage(t *testing.T) {
	for _, code := range []string{"", "GM-", "GM-AAAA", "GM16-AAAAAAAAAA", "GM-!!!!!!!!!!"} {
		if _, err := decodeShortOpenCode(code); err == nil {
			t.Fatalf("accepted invalid short code %q", code)
		}
	}
}

func TestLocalInviteCandidatesHaveValidPorts(t *testing.T) {
	for _, c := range localInviteCandidates(7777) {
		if c.Port != 7777 || !c.Addr.IsValid() {
			t.Fatalf("candidate=%+v", c)
		}
		if net.ParseIP(c.Addr.String()) == nil {
			t.Fatalf("candidate IP not parseable: %s", c.Addr)
		}
	}
}

func TestOpenInternetHostNeverCreatesJoinCode(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	g := &Game{
		netplay:                 host,
		onlineSessionKey:        "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		onlineSessionScope:      onlineSessionScopeInternet,
		onlineSessionVisibility: onlineSessionOpen,
		p2pHost: &p2pQUICHost{
			publicEndpoint:  "203.0.113.44:61234",
			publicEndpoints: []string{"203.0.113.44:61234"},
			dhtReady:        true,
		},
	}
	if code, ready := g.currentHostDisplayJoinCode(); ready || code != "" {
		t.Fatalf("Internet OPEN exposed invite code=%q ready=%v", code, ready)
	}
	if label := g.currentHostDisplayJoinCodeLabel(); !strings.Contains(label, "VISIBLE IN SERVER BROWSER") {
		t.Fatalf("OPEN host label=%q does not direct players to browser", label)
	}
}

func TestOpenInternetHostReportsP2PStartingInsteadOfPortMapping(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	g := &Game{
		netplay:                 host,
		onlineSessionScope:      onlineSessionScopeInternet,
		onlineSessionVisibility: onlineSessionOpen,
	}
	if label := g.currentHostDisplayJoinCodeLabel(); !strings.Contains(label, "P2P STARTING") {
		t.Fatalf("failed mapping label=%q", label)
	}
}
