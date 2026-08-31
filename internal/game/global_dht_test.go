package game

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"net"
	"os"
	"testing"
	"time"

	quic "github.com/quic-go/quic-go"
)

func TestDHTBencodeRoundTrip(t *testing.T) {
	var id [20]byte
	for i := range id {
		id[i] = byte(i + 1)
	}
	payload := bencodeMap(map[string]any{
		"a": map[string]any{"id": id[:], "port": int64(7777)},
		"q": []byte("announce_peer"),
		"t": []byte{1, 2, 3, 4},
		"y": []byte("q"),
	})
	decoded, err := decodeBencodeMessage(payload)
	if err != nil {
		t.Fatal(err)
	}
	if string(bgetBytes(decoded, "q")) != "announce_peer" || string(bgetBytes(decoded, "y")) != "q" {
		t.Fatalf("unexpected decoded message: %#v", decoded)
	}
	a := bgetMap(decoded, "a")
	if a == nil || len(bgetBytes(a, "id")) != 20 {
		t.Fatalf("missing announce arguments: %#v", a)
	}
}

func TestDHTBencodeRejectsUntrustedShapes(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte("d1:bi1e1:ai2ee"),                       // unsorted dict keys
		[]byte("d1:ai01ee"),                            // non-canonical integer
		append([]byte("4097:"), make([]byte, 4097)...), // oversized packet/string
		[]byte("l1:a1:b"),                              // unterminated list
	} {
		if _, err := decodeBencodeMessage(raw); err == nil {
			t.Fatalf("accepted invalid bencode %q", raw[:min(len(raw), 32)])
		}
	}
}

func TestDHTGetPeersAndAnnounceAgainstMockNode(t *testing.T) {
	server, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	var remoteID [20]byte
	for i := range remoteID {
		remoteID[i] = byte(0xa0 + i)
	}
	peerCompact := make([]byte, 6)
	copy(peerCompact[:4], net.ParseIP("203.0.113.25").To4())
	binary.BigEndian.PutUint16(peerCompact[4:], 7777)

	done := make(chan error, 1)
	go func() {
		buf := make([]byte, globalDHTPacketMax)
		seenGetPeers := false
		seenAnnounce := false
		for !(seenGetPeers && seenAnnounce) {
			n, addr, err := server.ReadFromUDP(buf)
			if err != nil {
				done <- err
				return
			}
			msg, err := decodeBencodeMessage(buf[:n])
			if err != nil {
				done <- err
				return
			}
			tx := append([]byte(nil), bgetBytes(msg, "t")...)
			switch string(bgetBytes(msg, "q")) {
			case "get_peers":
				seenGetPeers = true
				resp := bencodeMap(map[string]any{
					"r": map[string]any{
						"id":     remoteID[:],
						"token":  []byte("mock-token"),
						"values": []any{peerCompact},
					},
					"t": tx,
					"y": []byte("r"),
				})
				_, err = server.WriteToUDP(resp, addr)
			case "announce_peer":
				seenAnnounce = true
				resp := bencodeMap(map[string]any{
					"r": map[string]any{"id": remoteID[:]},
					"t": tx,
					"y": []byte("r"),
				})
				_, err = server.WriteToUDP(resp, addr)
			default:
				done <- nil
				return
			}
			if err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()

	client, err := newDHTLiteClient()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	testHash := globalOpenDHTInfoHashAt(time.Now())
	resp, err := client.getPeers(ctx, dhtLiteNode{Addr: server.LocalAddr().(*net.UDPAddr)}, testHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Token) == 0 || len(resp.Peers) != 1 || resp.Peers[0] != "203.0.113.25:7777" {
		t.Fatalf("unexpected get_peers response: %+v", resp)
	}
	if err := client.announcePeer(ctx, dhtLiteToken{Node: dhtLiteNode{ID: resp.NodeID, Addr: server.LocalAddr().(*net.UDPAddr)}, Token: resp.Token}, testHash, 7777, false); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("mock DHT node did not finish")
	}
}

func TestSharedP2PSocketDHTAgainstMockNode(t *testing.T) {
	server, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	var remoteID [20]byte
	for i := range remoteID {
		remoteID[i] = byte(0x30 + i)
	}
	go func() {
		buf := make([]byte, globalDHTPacketMax)
		n, addr, err := server.ReadFromUDP(buf)
		if err != nil {
			return
		}
		msg, err := decodeBencodeMessage(buf[:n])
		if err != nil {
			return
		}
		tx := append([]byte(nil), bgetBytes(msg, "t")...)
		resp := bencodeMap(map[string]any{
			"r": map[string]any{"id": remoteID[:], "token": []byte("shared-token")},
			"t": tx,
			"y": []byte("r"),
		})
		_, _ = server.WriteToUDP(resp, addr)
	}()

	mux, err := newP2PPacketMux()
	if err != nil {
		t.Fatal(err)
	}
	defer mux.Close()
	client, err := newDHTLiteClientOnMux(mux)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := client.getPeers(ctx, dhtLiteNode{Addr: server.LocalAddr().(*net.UDPAddr)}, globalOpenDHTInfoHashAt(time.Now()))
	if err != nil {
		t.Fatal(err)
	}
	if string(resp.Token) != "shared-token" {
		t.Fatalf("unexpected shared-socket token %q", resp.Token)
	}
}

func TestPublicProbeVerifiesOpenHostWithoutTakingSlot(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	addr := host.listener.Addr().String()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	info, err := probeOpenNetplayServer(ctx, addr)
	if err != nil {
		t.Fatal(err)
	}
	if info.Players != 1 || info.MaxPlayers != maxOnlinePlayers {
		t.Fatalf("probe info=%+v", info)
	}
	if host.connectedCount() != 0 {
		t.Fatalf("probe consumed a player slot: %d", host.connectedCount())
	}

	host.inviteOnly = true
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel2()
	if _, err := probeOpenNetplayServer(ctx2, addr); err == nil {
		t.Fatal("invite-only host answered public probe")
	}
}

func TestSharedSocketLiveBootstrapQueries(t *testing.T) {
	if os.Getenv("GUNMAYHEM_DHT_LIVE_TEST") != "1" {
		t.Skip("set GUNMAYHEM_DHT_LIVE_TEST=1 for public Mainline DHT integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	nodes := resolveDHTBootstrap(ctx)
	if len(nodes) == 0 {
		t.Fatal("no public DHT bootstrap addresses resolved")
	}
	mux, err := newP2PPacketMux()
	if err != nil {
		t.Fatal(err)
	}
	defer mux.Close()
	client, err := newDHTLiteClientOnMux(mux)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	success := 0
	for _, node := range nodes {
		resp, err := client.getPeers(ctx, node, globalOpenDHTInfoHashAt(time.Now()))
		if err != nil {
			t.Logf("shared bootstrap %s: %v", node.Addr, err)
			continue
		}
		success++
		t.Logf("shared bootstrap %s: nodes=%d token=%d peers=%d", node.Addr, len(resp.Nodes), len(resp.Token), len(resp.Peers))
	}
	if success == 0 {
		t.Fatal("no bootstrap node answered shared-socket get_peers")
	}
}

func TestSharedSocketWithQUICLiveBootstrapQueries(t *testing.T) {
	if os.Getenv("GUNMAYHEM_DHT_LIVE_TEST") != "1" {
		t.Skip("set GUNMAYHEM_DHT_LIVE_TEST=1 for public Mainline DHT integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	nodes := resolveDHTBootstrap(ctx)
	if len(nodes) == 0 {
		t.Fatal("no public DHT bootstrap addresses resolved")
	}
	mux, err := newP2PPacketMux()
	if err != nil {
		t.Fatal(err)
	}
	transport := &quic.Transport{Conn: mux, ConnectionIDLength: 8}
	tlsConf, err := p2pServerTLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	listener, err := transport.Listen(tlsConf, p2pQUICConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = listener.Close()
		_ = mux.Close()
		_ = transport.Close()
	}()
	client, err := newDHTLiteClientOnMux(mux)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	success := 0
	for _, node := range nodes {
		resp, err := client.getPeers(ctx, node, globalOpenDHTInfoHashAt(time.Now()))
		if err != nil {
			t.Logf("shared+quic bootstrap %s: %v", node.Addr, err)
			continue
		}
		success++
		t.Logf("shared+quic bootstrap %s: nodes=%d token=%d peers=%d", node.Addr, len(resp.Nodes), len(resp.Token), len(resp.Peers))
	}
	if success == 0 {
		t.Fatal("no bootstrap node answered shared-socket get_peers with QUIC active")
	}
}

func TestGlobalDHTLiveBootstrapQueries(t *testing.T) {
	if os.Getenv("GUNMAYHEM_DHT_LIVE_TEST") != "1" {
		t.Skip("set GUNMAYHEM_DHT_LIVE_TEST=1 for public Mainline DHT integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	nodes := resolveDHTBootstrap(ctx)
	if len(nodes) == 0 {
		t.Fatal("no public DHT bootstrap addresses resolved")
	}
	client, err := newDHTLiteClient()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	success := 0
	for _, node := range nodes {
		resp, err := client.getPeers(ctx, node, globalOpenDHTInfoHashAt(time.Now()))
		if err != nil {
			t.Logf("bootstrap %s: %v", node.Addr, err)
			continue
		}
		success++
		t.Logf("bootstrap %s: nodes=%d token=%d peers=%d", node.Addr, len(resp.Nodes), len(resp.Token), len(resp.Peers))
	}
	if success == 0 {
		t.Fatal("no bootstrap node answered get_peers")
	}
}

func TestGlobalBrowserLiveDiscovery(t *testing.T) {
	if os.Getenv("GUNMAYHEM_DHT_LIVE_TEST") != "1" {
		t.Skip("set GUNMAYHEM_DHT_LIVE_TEST=1 for public DHT browser integration test")
	}
	servers, err := discoverGlobalOpenServers(12 * time.Second)
	t.Logf("DHT candidates=%+v err=%v", servers, err)
	if len(servers) == 0 {
		t.Fatal("global browser found no DHT candidates")
	}
	verified := verifyGlobalOpenServers(servers, 12*time.Second)
	t.Logf("global candidates after preview=%+v", verified)
}

func TestGlobalBrowserHidesUnverifiedDHTCandidate(t *testing.T) {
	candidate := multiplayerServerInfo{
		Name: "GLOBAL OPEN", Address: "203.0.113.77:45678", Transport: "quic",
		Sources: "GLOBAL", Players: -1, MaxPlayers: maxOnlinePlayers, Protocol: netplayProtocolVersion,
	}
	got := verifyGlobalOpenServers([]multiplayerServerInfo{candidate}, 20*time.Millisecond)
	if len(got) != 0 {
		t.Fatalf("unverified DHT peer leaked into playable browser: %+v", got)
	}
}

func TestSharedSocketLiveAnnounceExplicit(t *testing.T) {
	if os.Getenv("GUNMAYHEM_DHT_LIVE_TEST") != "1" {
		t.Skip("set GUNMAYHEM_DHT_LIVE_TEST=1 for public Mainline DHT integration test")
	}
	mux, err := newP2PPacketMux()
	if err != nil {
		t.Fatal(err)
	}
	defer mux.Close()
	client, err := newDHTLiteClientOnMux(mux)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	udp := mux.LocalAddr().(*net.UDPAddr)
	var hash [20]byte
	if _, err := rand.Read(hash[:]); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Second)
	defer cancel()
	if err := announceDHTPeerWithClient(ctx, client, hash, udp.Port, false); err != nil {
		t.Fatalf("shared explicit-port announce failed: %v", err)
	}
}

func TestGlobalDHTLiveAnnounce(t *testing.T) {
	if os.Getenv("GUNMAYHEM_DHT_LIVE_TEST") != "1" {
		t.Skip("set GUNMAYHEM_DHT_LIVE_TEST=1 for public Mainline DHT integration test")
	}
	var hash [20]byte
	if _, err := rand.Read(hash[:]); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Second)
	defer cancel()
	if err := announceDHTPeer(ctx, hash, 54321); err != nil {
		t.Fatalf("public DHT announce failed: %v", err)
	}

	// A fresh DHT client, with no knowledge of the announcer's IP, should be
	// able to recover the peer from the same infohash just like a torrent peer.
	time.Sleep(350 * time.Millisecond)
	lookup, err := newDHTLiteClient()
	if err != nil {
		t.Fatal(err)
	}
	defer lookup.Close()
	lookupCtx, lookupCancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer lookupCancel()
	peers, _, err := dhtTraversal(lookupCtx, lookup, hash)
	if err != nil && len(peers) == 0 {
		t.Fatalf("public DHT lookup failed: %v", err)
	}
	found := false
	for _, peer := range peers {
		_, port, splitErr := net.SplitHostPort(peer)
		if splitErr == nil && port == "54321" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("fresh DHT client did not recover announced peer; peers=%v", peers)
	}
}
