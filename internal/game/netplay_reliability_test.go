package game

import (
	"encoding/binary"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNetplayStaleAndClosedInputIsNeutral(t *testing.T) {
	p := &netplayPeer{input: PlayerInputState{Right: true, Shoot: true}, inputReceived: time.Now()}
	if got := p.latestInput(); !got.Right || !got.Shoot {
		t.Fatal("fresh input was lost")
	}
	p.inputReceived = time.Now().Add(-netInputMaxAge - time.Millisecond)
	if got := p.latestInput(); got != (PlayerInputState{}) {
		t.Fatalf("stalled client still controls player: %+v", got)
	}
	p.inputReceived = time.Now()
	p.closed.Store(true)
	if got := p.latestInput(); got != (PlayerInputState{}) {
		t.Fatalf("closed client still controls player: %+v", got)
	}
}

func TestNetplayCloseCancelsPendingHandshake(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	conn, err := net.Dial("tcp", host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var challenge netWireMessage
	if err := readNetMessage(conn, maxServerWireFrame, &challenge); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "pending handshake", func() bool { return host.pendingHandshakes.Load() == 1 })
	if err := host.Close(); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	var one [1]byte
	if _, err := conn.Read(one[:]); err == nil {
		t.Fatal("pending connection survived host close")
	}
	waitFor(t, "pending handshake cleanup", func() bool { return host.pendingHandshakes.Load() == 0 })
	if host.connectedCount() != 0 {
		t.Fatal("closed host assigned a player slot")
	}
}

func TestNetplayConfiguredHostAuthenticatesFirstConnection(t *testing.T) {
	key, err := newOnlineSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	host, err := newConfiguredHostNetplay("127.0.0.1:0", nil, func(s *netplaySession) {
		s.inviteOnly = true
		s.expectedSessionKey = key
	})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	intruder, err := newClientNetplay(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer intruder.Close()
	waitFor(t, "unauthenticated rejection", func() bool { return intruder.LastError() != nil })
	if host.connectedCount() != 0 {
		t.Fatal("unauthenticated first connection got a slot")
	}
	client, err := newClientNetplayWithKey(host.listener.Addr().String(), key, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	waitFor(t, "authenticated connection", client.connected)
}

func TestNetplayRejectsPlayerIdentitySpoof(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	client, err := newClientNetplay(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	waitFor(t, "client connection", client.connected)
	if err := client.link.Write(netWireMessage{Type: netMessageInput, Protocol: netplayProtocolVersion,
		PlayerID: 4, Input: PlayerInputState{Shoot: true}}, maxClientWireFrame); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "spoof disconnection", func() bool { return !client.connected() && host.connectedCount() == 0 })
	if got := host.latestInputFor(4); got != (PlayerInputState{}) {
		t.Fatal("spoofed input reached another slot")
	}
}

func TestNetplayRejoinGetsCleanVacatedSlot(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	client, err := newClientNetplay(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "first connection", client.connected)
	client.queueInput(PlayerInputState{Shoot: true})
	waitFor(t, "first input", func() bool { return host.latestInputFor(2).Shoot })
	client.Close()
	waitFor(t, "slot removal", func() bool { return host.connectedCount() == 0 })
	rejoined, err := newClientNetplay(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer rejoined.Close()
	waitFor(t, "rejoined connection", rejoined.connected)
	if rejoined.assignedPlayer() != 2 {
		t.Fatalf("rejoin got slot %d", rejoined.assignedPlayer())
	}
	if got := host.latestInputFor(2); got != (PlayerInputState{}) {
		t.Fatal("old input leaked into rejoined slot")
	}
	if host.postGameReadyFor(2) {
		t.Fatal("old ready status leaked into rejoined slot")
	}
}

func TestNetplayHeartbeatKeepsIdleLobbyConnected(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	client, err := newClientNetplay(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	waitFor(t, "idle lobby connection", client.connected)
	// There are deliberately no snapshots or input updates during this window.
	time.Sleep(netPeerIdleTimeout + 2*netHeartbeatInterval)
	if !client.connected() || host.connectedCount() != 1 {
		t.Fatalf("heartbeat failed: client=%v host=%v", client.LastError(), host.LastError())
	}
}

func TestNetplaySilentPeerExpires(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	conn, err := net.Dial("tcp", host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var msg netWireMessage
	if err := readNetMessage(conn, maxServerWireFrame, &msg); err != nil {
		t.Fatal(err)
	}
	if err := writeNetMessage(conn, netWireMessage{Type: netMessageHello, Protocol: netplayProtocolVersion}, maxClientWireFrame); err != nil {
		t.Fatal(err)
	}
	if err := readNetMessage(conn, maxServerWireFrame, &msg); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "silent peer accepted", host.connected)
	time.Sleep(netPeerIdleTimeout + 100*time.Millisecond)
	waitFor(t, "silent peer removal", func() bool { return host.connectedCount() == 0 })
}

// jitterWriteConn adds synthetic per-write delay, not a real Internet RTT model.
// TCP stays ordered, while coalescing must prevent old queued input accumulating.
type jitterWriteConn struct {
	net.Conn
	delay  time.Duration
	writes int
}

func (c *jitterWriteConn) Write(b []byte) (int, error) {
	c.writes++
	time.Sleep(c.delay + time.Duration(c.writes%3)*time.Millisecond)
	return c.Conn.Write(b)
}

func TestNetplaySyntheticLatencyKeepsLatestInputAndState(t *testing.T) {
	for _, delay := range []time.Duration{0, 40 * time.Millisecond, 100 * time.Millisecond} {
		t.Run(delay.String(), func(t *testing.T) {
			host, err := newHostNetplay("127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer host.Close()
			conn, err := net.Dial("tcp", host.listener.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			client, err := newClientNetplayFromConn(&jitterWriteConn{Conn: conn, delay: delay}, "synthetic delayed TCP")
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			waitFor(t, "delayed client", client.connected)
			for i := 0; i < 300; i++ {
				client.queueInput(PlayerInputState{Left: true})
			}
			want := PlayerInputState{Right: true, Shoot: true}
			client.queueInput(want)
			waitFor(t, "latest delayed input", func() bool { return host.latestInputFor(2) == want })
			for i := 1; i <= 100; i++ {
				host.queueState(netSnapshot{TotalLives: i})
			}
			waitFor(t, "latest snapshot", func() bool { s, ok := client.latestState(); return ok && s.TotalLives == 100 })
		})
	}
}

func TestNetplayOversizedFrameDisconnectsOnlySender(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	good, err := newClientNetplay(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer good.Close()
	waitFor(t, "good client", good.connected)
	bad, err := net.Dial("tcp", host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer bad.Close()
	var challenge netWireMessage
	if err := readNetMessage(bad, maxServerWireFrame, &challenge); err != nil {
		t.Fatal(err)
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], maxClientWireFrame+1)
	if _, err := bad.Write(header[:]); err != nil {
		t.Fatal(err)
	}
	_ = bad.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := bad.Read(header[:]); err == nil {
		t.Fatal("oversize frame not rejected")
	}
	if !good.connected() || host.connectedCount() != 1 {
		t.Fatal("malformed client disrupted healthy peer")
	}
	good.queueInput(PlayerInputState{Up: true})
	waitFor(t, "healthy input after attack", func() bool { return host.latestInputFor(2).Up })
}

func TestExperimentalInternetDefaultsToInviteWithoutDirectory(t *testing.T) {
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "1")
	t.Setenv("GUNMAYHEM_MASTER_URL", "")
	g := New()
	g.activateCustomHostSetupHit(menuHitCustomHostScopeInternet)
	if g.customHostScope != onlineSessionScopeInternet || g.customHostAccess != onlineSessionInviteOnly {
		t.Fatal("Internet did not select private invite flow")
	}
	g.activateCustomHostSetupHit(menuHitCustomHostAccessOpen)
	if g.customHostAccess != onlineSessionInviteOnly {
		t.Fatal("experimental Internet exposed OPEN discovery")
	}
	if !strings.Contains(g.multiplayerMessage, "PUBLIC DIRECTORY NOT CONFIGURED") {
		t.Fatal("restriction not explained")
	}
}

func TestInternetInviteKeepsFullSessionSecret(t *testing.T) {
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "1")
	g := New()
	if err := g.StartNetHostConfig("127.0.0.1:0", onlineSessionScopeInternet, onlineSessionInviteOnly); err != nil {
		t.Fatal(err)
	}
	defer g.CloseNetplay()
	if g.netplay.shortJoinCode != "" {
		t.Fatal("Internet secret was reduced to short LAN code")
	}
	if len(g.onlineSessionKey) != 26 {
		t.Fatalf("unexpected full secret length %d", len(g.onlineSessionKey))
	}
	if g.p2pHost != nil || g.cloudflareTunnel != nil {
		t.Fatal("loopback fixture started Internet services")
	}
}

func TestInternetHostPolicyCannotBeBypassedByLegacyEntryPoints(t *testing.T) {
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "")
	g := New()
	if err := g.StartNetHostConfig("127.0.0.1:0", onlineSessionScopeInternet, onlineSessionInviteOnly); err == nil {
		t.Fatal("Internet hosting did not require opt-in")
	}
	if g.netplay != nil {
		t.Fatal("rejected host created a listener")
	}
	if err := g.StartNetHost("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer g.CloseNetplay()
	if g.onlineSessionScope != onlineSessionScopeLAN || g.cloudflareTunnel != nil || g.p2pHost != nil {
		t.Fatal("legacy --host escaped LAN scope")
	}
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "1")
	if err := g.StartNetHostConfig("127.0.0.1:0", onlineSessionScopeInternet, onlineSessionOpen); err == nil {
		t.Fatal("Internet OPEN host was accepted")
	}
	if g.netplay == nil || g.onlineSessionScope != onlineSessionScopeLAN {
		t.Fatal("rejected policy destroyed existing LAN session")
	}
}

func TestNetplaySlowReceiverCannotBlockHealthyPeer(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	server, slow := net.Pipe()
	defer slow.Close()
	if !host.attachIncomingConn(server) {
		t.Fatal("could not attach slow fixture")
	}
	var msg netWireMessage
	if err := readNetMessage(slow, maxServerWireFrame, &msg); err != nil {
		t.Fatal(err)
	}
	if err := writeNetMessage(slow, netWireMessage{Type: netMessageHello, Protocol: netplayProtocolVersion}, maxClientWireFrame); err != nil {
		t.Fatal(err)
	}
	if err := readNetMessage(slow, maxServerWireFrame, &msg); err != nil {
		t.Fatal(err)
	}
	good, err := newClientNetplay(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer good.Close()
	waitFor(t, "healthy peer assignment", good.connected)
	if good.assignedPlayer() != 3 {
		t.Fatal("slow fixture did not reserve slot 2")
	}
	for i := 1; i <= 500; i++ {
		host.queueState(netSnapshot{TotalLives: i})
	}
	waitFor(t, "healthy snapshot beside blocked receiver", func() bool { s, ok := good.latestState(); return ok && s.TotalLives == 500 })
	// The unresponsive net.Pipe cannot consume even the first queued snapshot.
	// Its own writer times out independently of the healthy peer.
	time.Sleep(netWriteTimeout + 100*time.Millisecond)
	waitFor(t, "slow receiver eviction", func() bool { return host.connectedCount() == 1 })
	if !good.connected() {
		t.Fatal("healthy peer was evicted with slow receiver")
	}
}
