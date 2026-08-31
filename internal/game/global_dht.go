package game

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	globalDHTLookupTimeout = 9 * time.Second
	globalDHTAnnounceEvery = 45 * time.Second
	globalDHTSessionWindow = 2 * time.Minute
	globalDHTMaxPeers      = 64
	globalDHTMaxQueries    = 56
	globalDHTMaxQueue      = 256
	globalDHTQueryTimeout  = 850 * time.Millisecond
	globalDHTPacketMax     = 4096
)

// Mainline BitTorrent DHT (BEP-5) is used only as a decentralized address
// book. The swarm key is protocol-version-specific so incompatible builds do
// not discover each other. No torrent metadata or files are transferred.
func globalOpenDHTInfoHashAt(now time.Time) [20]byte {
	window := now.UTC().Unix() / int64(globalDHTSessionWindow/time.Second)
	return sha1.Sum([]byte(fmt.Sprintf("GunMayhem-RE-open-session-protocol-16-quic|%d", window)))
}

func globalOpenDHTLookupHashes(now time.Time) [][20]byte {
	return [][20]byte{
		globalOpenDHTInfoHashAt(now),
		globalOpenDHTInfoHashAt(now.Add(-globalDHTSessionWindow)),
	}
}

var globalDHTBootstrap = []string{
	"router.bittorrent.com:6881",
	"router.utorrent.com:6881",
	"dht.transmissionbt.com:6881",
	"router.bitcomet.com:6881",
	"dht.aelitis.com:6881",
	"dht.libtorrent.org:25401",
	"router.silotis.us:6881",
	"dht.anacrolix.link:42069",
}

type globalDHTAdvertiser struct {
	closed atomic.Bool
	stop   chan struct{}
	done   chan struct{}

	internalPort int
	mapper       *natPortMapper

	mu      sync.RWMutex
	lastErr error
	ready   bool
	port    int
}

type dhtLiteClient struct {
	conn   *net.UDPConn
	mux    *p2pPacketMux
	nodeID [20]byte
	closed atomic.Bool

	mu      sync.Mutex
	pending map[string]dhtLitePending
	done    chan struct{}
	stop    chan struct{}
}

type dhtLitePending struct {
	addr *net.UDPAddr
	ch   chan map[string]any
}

type dhtLiteNode struct {
	ID   [20]byte
	Addr *net.UDPAddr
}

type dhtLiteToken struct {
	Node     dhtLiteNode
	Token    []byte
	Observed string
}

type dhtLiteGetPeers struct {
	NodeID   [20]byte
	Nodes    []dhtLiteNode
	Peers    []string
	Token    []byte
	Observed string
}

func startGlobalDHTAdvertiser(internalPort int, mapper *natPortMapper) *globalDHTAdvertiser {
	a := &globalDHTAdvertiser{
		internalPort: internalPort,
		mapper:       mapper,
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
	}
	go a.loop()
	return a
}

func (a *globalDHTAdvertiser) loop() {
	defer close(a.done)
	if a == nil || a.internalPort <= 0 || a.internalPort > 65535 {
		if a != nil {
			a.setStatus(false, 0, errors.New("invalid DHT game port"))
		}
		return
	}

	if a.mapper != nil {
		a.mapper.Wait(natDiscoveryTimeout + time.Second)
	}
	port, ok := dhtReachableGamePort(a.internalPort, a.mapper)
	if !ok {
		a.setStatus(false, 0, errors.New("internet-reachable gameplay port unavailable"))
		return
	}

	retryDelay := 10 * time.Second
	for {
		ctx, cancel := context.WithTimeout(context.Background(), globalDHTLookupTimeout)
		err := announceGlobalOpenSession(ctx, port)
		cancel()
		next := globalDHTAnnounceEvery
		if err != nil {
			a.setStatus(false, port, err)
			next = retryDelay
			retryDelay *= 2
			if retryDelay > 2*time.Minute {
				retryDelay = 2 * time.Minute
			}
		} else {
			a.setStatus(true, port, nil)
			retryDelay = 10 * time.Second
		}
		timer := time.NewTimer(next)
		select {
		case <-timer.C:
		case <-a.stop:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		}
	}
}

func dhtReachableGamePort(internalPort int, mapper *natPortMapper) (int, bool) {
	if internalPort <= 0 || internalPort > 65535 {
		return 0, false
	}
	if mapper != nil {
		_, mapped, _, externalIP, externalPort, _ := mapper.Status()
		if mapped && externalPort > 0 && externalPort <= 65535 {
			externalIP = strings.TrimSpace(externalIP)
			if externalIP == "" {
				return externalPort, true
			}
			if parsed, err := netip.ParseAddr(externalIP); err == nil && isInternetRoutableAddr(parsed.Unmap()) {
				return externalPort, true
			}
		}
	}
	for _, candidate := range localInviteCandidates(internalPort) {
		if isInternetRoutableAddr(candidate.Addr) {
			return internalPort, true
		}
	}
	return 0, false
}

func newDHTLiteClient() (*dhtLiteClient, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	c := &dhtLiteClient{
		conn:    conn,
		pending: make(map[string]dhtLitePending),
		done:    make(chan struct{}),
		stop:    make(chan struct{}),
	}
	if _, err := rand.Read(c.nodeID[:]); err != nil {
		_ = conn.Close()
		return nil, err
	}
	go c.readLoop()
	return c, nil
}

func newDHTLiteClientOnMux(mux *p2pPacketMux) (*dhtLiteClient, error) {
	if mux == nil || mux.closed.Load() {
		return nil, errors.New("invalid shared DHT socket")
	}
	c := &dhtLiteClient{
		mux:     mux,
		pending: make(map[string]dhtLitePending),
		done:    make(chan struct{}),
		stop:    make(chan struct{}),
	}
	if _, err := rand.Read(c.nodeID[:]); err != nil {
		return nil, err
	}
	go c.readLoop()
	return c, nil
}

func (c *dhtLiteClient) readLoop() {
	defer close(c.done)
	buf := make([]byte, globalDHTPacketMax)
	for !c.closed.Load() {
		var n int
		var from *net.UDPAddr
		if c.mux != nil {
			select {
			case pkt := <-c.mux.dhtCh:
				udp, ok := pkt.addr.(*net.UDPAddr)
				if !ok || udp == nil || len(pkt.data) == 0 || len(pkt.data) > len(buf) {
					continue
				}
				n = copy(buf, pkt.data)
				from = cloneUDPAddr(udp)
			case <-c.stop:
				return
			case <-c.mux.done:
				return
			}
		} else {
			var err error
			n, from, err = c.conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
		}
		if n <= 0 || n > globalDHTPacketMax || from == nil {
			continue
		}
		msg, err := decodeBencodeMessage(buf[:n])
		if err != nil {
			continue
		}
		tx := bgetBytes(msg, "t")
		if len(tx) == 0 || len(tx) > 8 {
			continue
		}
		key := string(tx)
		c.mu.Lock()
		pending, ok := c.pending[key]
		c.mu.Unlock()
		if !ok || !sameUDPAddr(from, pending.addr) {
			continue
		}
		select {
		case pending.ch <- msg:
		default:
		}
	}
}

func (c *dhtLiteClient) registerPending(addr *net.UDPAddr) ([]byte, <-chan map[string]any, func(), error) {
	if c == nil || (c.conn == nil && c.mux == nil) || addr == nil || c.closed.Load() {
		return nil, nil, nil, errors.New("invalid DHT client")
	}
	for attempt := 0; attempt < 8; attempt++ {
		tx, err := randomDHTTransaction()
		if err != nil {
			return nil, nil, nil, err
		}
		key := string(tx)
		ch := make(chan map[string]any, 1)
		c.mu.Lock()
		if _, exists := c.pending[key]; exists {
			c.mu.Unlock()
			continue
		}
		c.pending[key] = dhtLitePending{addr: addr, ch: ch}
		c.mu.Unlock()
		cleanup := func() {
			c.mu.Lock()
			delete(c.pending, key)
			c.mu.Unlock()
		}
		return tx, ch, cleanup, nil
	}
	return nil, nil, nil, errors.New("could not allocate DHT transaction")
}

func (c *dhtLiteClient) waitQuery(ctx context.Context, addr *net.UDPAddr, payloadBuilder func(tx []byte) []byte) (map[string]any, error) {
	tx, ch, cleanup, err := c.registerPending(addr)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	payload := payloadBuilder(tx)
	if len(payload) == 0 || len(payload) > 1200 {
		return nil, errors.New("invalid DHT query packet")
	}
	if c.mux != nil {
		if _, err := c.mux.WriteTo(payload, addr); err != nil {
			return nil, err
		}
	} else if _, err := c.conn.WriteToUDP(payload, addr); err != nil {
		return nil, err
	}
	timer := time.NewTimer(globalDHTQueryTimeout)
	defer timer.Stop()
	select {
	case msg := <-ch:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, errors.New("DHT query timed out")
	case <-c.done:
		return nil, net.ErrClosed
	}
}

func (c *dhtLiteClient) Close() error {
	if c == nil || c.closed.Swap(true) {
		return nil
	}
	if c.stop != nil {
		close(c.stop)
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func resolveDHTBootstrap(ctx context.Context) []dhtLiteNode {
	seen := make(map[string]bool)
	out := make([]dhtLiteNode, 0, len(globalDHTBootstrap)*2)
	for _, raw := range globalDHTBootstrap {
		host, portText, err := net.SplitHostPort(raw)
		if err != nil {
			continue
		}
		port, err := strconv.Atoi(portText)
		if err != nil || port <= 0 || port > 65535 {
			continue
		}
		ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip4", host)
		if err != nil {
			continue
		}
		for _, ip := range ips {
			ip = ip.Unmap()
			if !ip.Is4() || !isInternetRoutableAddr(ip) {
				continue
			}
			addr := &net.UDPAddr{IP: net.IP(ip.AsSlice()), Port: port}
			key := addr.String()
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, dhtLiteNode{Addr: addr})
		}
	}
	return out
}

func randomDHTTransaction() ([]byte, error) {
	var tx [4]byte
	if _, err := rand.Read(tx[:]); err != nil {
		return nil, err
	}
	return tx[:], nil
}

func (c *dhtLiteClient) getPeers(ctx context.Context, node dhtLiteNode, infoHash [20]byte) (dhtLiteGetPeers, error) {
	if c == nil || (c.conn == nil && c.mux == nil) || node.Addr == nil {
		return dhtLiteGetPeers{}, errors.New("invalid DHT get_peers target")
	}
	msg, err := c.waitQuery(ctx, node.Addr, func(tx []byte) []byte {
		return bencodeMap(map[string]any{
			"a": map[string]any{"id": c.nodeID[:], "info_hash": infoHash[:]},
			"q": []byte("get_peers"),
			"t": tx,
			"y": []byte("q"),
		})
	})
	if err != nil {
		return dhtLiteGetPeers{}, err
	}
	if string(bgetBytes(msg, "y")) != "r" {
		return dhtLiteGetPeers{}, errors.New("DHT get_peers rejected")
	}
	r := bgetMap(msg, "r")
	if r == nil {
		return dhtLiteGetPeers{}, errors.New("DHT response missing result")
	}
	var out dhtLiteGetPeers
	if id := bgetBytes(r, "id"); len(id) == 20 {
		copy(out.NodeID[:], id)
	}
	out.Token = append([]byte(nil), bgetBytes(r, "token")...)
	out.Nodes = parseCompactDHTNodes(bgetBytes(r, "nodes"))
	out.Peers = parseCompactDHTPeers(bgetList(r, "values"))
	out.Observed = parseCompactDHTEndpoint(bgetBytes(msg, "ip"))
	return out, nil
}

func (c *dhtLiteClient) announcePeer(ctx context.Context, target dhtLiteToken, infoHash [20]byte, port int, impliedPort bool) error {
	if c == nil || (c.conn == nil && c.mux == nil) || target.Node.Addr == nil || len(target.Token) == 0 || len(target.Token) > 256 || port < 0 || port > 65535 {
		return errors.New("invalid DHT announce target")
	}
	msg, err := c.waitQuery(ctx, target.Node.Addr, func(tx []byte) []byte {
		args := map[string]any{
			"id":        c.nodeID[:],
			"info_hash": infoHash[:],
			"port":      int64(port),
			"token":     target.Token,
		}
		if impliedPort {
			args["implied_port"] = int64(1)
		}
		return bencodeMap(map[string]any{
			"a": args,
			"q": []byte("announce_peer"),
			"t": tx,
			"y": []byte("q"),
		})
	})
	if err != nil {
		return err
	}
	if string(bgetBytes(msg, "y")) != "r" {
		return errors.New("DHT announce rejected")
	}
	return nil
}

func sameUDPAddr(a, b *net.UDPAddr) bool {
	if a == nil || b == nil || a.Port != b.Port {
		return false
	}
	return a.IP.Equal(b.IP)
}

func dhtTraversal(ctx context.Context, client *dhtLiteClient, infoHash [20]byte) (peers []string, tokens []dhtLiteToken, err error) {
	if client == nil {
		return nil, nil, errors.New("nil DHT client")
	}
	queue := resolveDHTBootstrap(ctx)
	if len(queue) == 0 {
		return nil, nil, errors.New("DHT bootstrap unavailable")
	}
	queued := make(map[string]bool, len(queue))
	for _, n := range queue {
		queued[n.Addr.String()] = true
	}
	queried := make(map[string]bool)
	peerSeen := make(map[string]bool)
	tokenSeen := make(map[string]bool)
	queries := 0
	peerFoundAtQueries := -1

	type queryResult struct {
		node dhtLiteNode
		resp dhtLiteGetPeers
		err  error
	}

Traversal:
	for len(queue) > 0 && queries < globalDHTMaxQueries {
		if ctx.Err() != nil {
			break
		}
		batch := make([]dhtLiteNode, 0, 4)
		for len(batch) < 4 && len(queue) > 0 && queries < globalDHTMaxQueries {
			node := queue[0]
			queue = queue[1:]
			key := node.Addr.String()
			if queried[key] {
				continue
			}
			queried[key] = true
			queries++
			batch = append(batch, node)
		}
		if len(batch) == 0 {
			break
		}

		results := make(chan queryResult, len(batch))
		for _, node := range batch {
			node := node
			go func() {
				resp, queryErr := client.getPeers(ctx, node, infoHash)
				results <- queryResult{node: node, resp: resp, err: queryErr}
			}()
		}
		for range batch {
			var result queryResult
			select {
			case result = <-results:
			case <-ctx.Done():
				break Traversal
			}
			if result.err != nil || result.node.Addr == nil {
				continue
			}
			node := result.node
			resp := result.resp
			key := node.Addr.String()
			if resp.NodeID != ([20]byte{}) {
				node.ID = resp.NodeID
			}
			if len(resp.Token) > 0 && len(resp.Token) <= 256 && !tokenSeen[key] {
				tokenSeen[key] = true
				tokens = append(tokens, dhtLiteToken{Node: node, Token: resp.Token, Observed: resp.Observed})
			}
			for _, peer := range resp.Peers {
				if !peerSeen[peer] {
					peerSeen[peer] = true
					peers = append(peers, peer)
					if peerFoundAtQueries < 0 {
						peerFoundAtQueries = queries
					}
					if len(peers) >= globalDHTMaxPeers {
						return peers, tokens, nil
					}
				}
			}
			for _, next := range resp.Nodes {
				if next.Addr == nil {
					continue
				}
				addrKey := next.Addr.String()
				if queued[addrKey] || queried[addrKey] {
					continue
				}
				queued[addrKey] = true
				queue = append(queue, next)
				if len(queue) >= globalDHTMaxQueue {
					break
				}
			}
		}

		// Kademlia alpha=4 traversal: after each parallel batch, continue with
		// the closest discovered nodes. Unknown bootstrap IDs remain first.
		sort.SliceStable(queue, func(i, j int) bool {
			zi := queue[i].ID == ([20]byte{})
			zj := queue[j].ID == ([20]byte{})
			if zi != zj {
				return zi
			}
			return dhtDistanceLess(queue[i].ID, queue[j].ID, infoHash)
		})
		// Peer values are replicated around the nodes closest to the infohash.
		// Once we have values, two extra alpha=4 rounds are enough to collect
		// neighboring replicas without making the UI wait for the full timeout.
		if peerFoundAtQueries >= 0 && queries-peerFoundAtQueries >= 8 {
			return peers, tokens, nil
		}
	}
	if len(peers) == 0 && len(tokens) == 0 {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		return nil, nil, errors.New("DHT traversal found no responsive nodes")
	}
	return peers, tokens, nil
}

func dhtDistanceLess(a, b, target [20]byte) bool {
	for i := 0; i < len(target); i++ {
		da := a[i] ^ target[i]
		db := b[i] ^ target[i]
		if da != db {
			return da < db
		}
	}
	return false
}

func announceGlobalOpenSession(ctx context.Context, port int) error {
	return announceDHTPeer(ctx, globalOpenDHTInfoHashAt(time.Now()), port)
}

func announceDHTPeer(ctx context.Context, infoHash [20]byte, port int) error {
	if port <= 0 || port > 65535 {
		return errors.New("invalid global DHT announce port")
	}
	client, err := newDHTLiteClient()
	if err != nil {
		return err
	}
	defer client.Close()
	return announceDHTPeerWithClient(ctx, client, infoHash, port, false)
}

func announceDHTPeerWithClient(ctx context.Context, client *dhtLiteClient, infoHash [20]byte, port int, impliedPort bool) error {
	_, err := announceDHTPeerWithClientObserved(ctx, client, infoHash, port, impliedPort)
	return err
}

func announceDHTPeerWithClientObserved(ctx context.Context, client *dhtLiteClient, infoHash [20]byte, port int, impliedPort bool) ([]string, error) {
	if client == nil || port < 0 || port > 65535 || (!impliedPort && port == 0) {
		return nil, errors.New("invalid DHT announce configuration")
	}
	// Reserve part of the overall deadline for the actual announce_peer calls.
	traversalCtx, cancelTraversal := context.WithTimeout(ctx, 6*time.Second)
	_, tokens, traversalErr := dhtTraversal(traversalCtx, client, infoHash)
	cancelTraversal()
	if traversalErr != nil && len(tokens) == 0 {
		return nil, traversalErr
	}
	if len(tokens) == 0 {
		return nil, errors.New("DHT announce found no tokens")
	}
	sort.Slice(tokens, func(i, j int) bool {
		return dhtDistanceLess(tokens[i].Node.ID, tokens[j].Node.ID, infoHash)
	})
	limit := min(8, len(tokens))
	type announceResult struct {
		ok       bool
		observed string
	}
	results := make(chan announceResult, limit)
	for i := 0; i < limit; i++ {
		token := tokens[i]
		go func() {
			announcePort := port
			useImplied := impliedPort
			// DHT security-extension responses commonly include our observed
			// public IP:port. Prefer that explicit port over implied_port. This
			// makes the stored endpoint match the mapping this exact DHT node
			// observed, and gives symmetric NATs several useful port samples.
			if impliedPort && token.Observed != "" {
				if _, portText, err := net.SplitHostPort(token.Observed); err == nil {
					if observedPort, err := strconv.Atoi(portText); err == nil && observedPort > 0 && observedPort <= 65535 {
						announcePort = observedPort
						useImplied = false
					}
				}
			}
			ok := client.announcePeer(ctx, token, infoHash, announcePort, useImplied) == nil
			results <- announceResult{ok: ok, observed: token.Observed}
		}()
	}
	success := 0
	observed := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		select {
		case result := <-results:
			if result.ok {
				success++
				if result.observed != "" {
					observed = append(observed, result.observed)
				}
			}
		case <-ctx.Done():
			i = limit
		}
	}
	if success == 0 {
		return observed, errors.New("DHT announce was not accepted by any node")
	}
	return observed, nil
}

func discoverGlobalOpenServers(timeout time.Duration) ([]multiplayerServerInfo, error) {
	if timeout <= 0 {
		timeout = globalDHTLookupTimeout
	}
	type lookupResult struct {
		peers []string
		err   error
	}
	hashes := globalOpenDHTLookupHashes(time.Now())
	results := make(chan lookupResult, len(hashes))
	for _, hash := range hashes {
		hash := hash
		go func() {
			peers, err := discoverDHTPeerAddresses(hash, timeout)
			results <- lookupResult{peers: peers, err: err}
		}()
	}
	var peers []string
	var traversalErr error
	for range hashes {
		result := <-results
		peers = append(peers, result.peers...)
		if result.err != nil {
			traversalErr = errors.Join(traversalErr, result.err)
		}
	}
	if len(peers) > 0 {
		traversalErr = nil
	}
	out := dhtPeerServerCandidates(peers, "GLOBAL OPEN", "GLOBAL")
	return out, traversalErr
}

func dhtPeerServerCandidates(peers []string, name, source string) []multiplayerServerInfo {
	out := make([]multiplayerServerInfo, 0, len(peers))
	seen := make(map[string]bool)
	byPublicIP := make(map[string]int)
	for _, address := range peers {
		host, portText, err := net.SplitHostPort(address)
		if err != nil || seen[address] {
			continue
		}
		ip, err := netip.ParseAddr(host)
		if err != nil || !isInternetRoutableAddr(ip.Unmap()) {
			continue
		}
		port, err := strconv.Atoi(portText)
		// Gun Mayhem hosts bind 7777 or an ephemeral high port. Refusing
		// privileged ports also removes malformed :1 entries produced by DHT
		// implementations that ignored implied_port in older test builds.
		if err != nil || port < 1024 || port > 65535 {
			continue
		}
		seen[address] = true
		candidate := multiplayerServerInfo{
			Name:      name,
			Address:   address,
			Transport: "quic",
			Sources:   source,
			// A DHT result only proves that an endpoint was announced. Player
			// count and match metadata become trustworthy after the preview
			// handshake succeeds, so keep them explicitly unknown until then.
			Players:    -1,
			MaxPlayers: maxOnlinePlayers,
			Protocol:   netplayProtocolVersion,
		}
		ipKey := ip.Unmap().String()
		if index, exists := byPublicIP[ipKey]; exists {
			// One NAT may be observed with several source ports. Present it as
			// one session instead of a row of duplicates; prefer the game's
			// fixed mapped port when one of the observations contains it.
			_, oldPortText, _ := net.SplitHostPort(out[index].Address)
			oldPort, _ := strconv.Atoi(oldPortText)
			if port == 7777 && oldPort != 7777 {
				out[index] = candidate
			}
			continue
		}
		byPublicIP[ipKey] = len(out)
		out = append(out, candidate)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Address < out[j].Address })
	return out
}

func probeOpenNetplayConn(ctx context.Context, conn net.Conn) (netProbeInfo, error) {
	if conn == nil {
		return netProbeInfo{}, errors.New("nil probe connection")
	}
	defer conn.Close()
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetNoDelay(true)
	}
	deadline := time.Now().Add(1200 * time.Millisecond)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = conn.SetReadDeadline(deadline)
	var challenge netWireMessage
	if err := readNetMessage(conn, maxServerWireFrame, &challenge); err != nil || challenge.Type != netMessageChallenge {
		return netProbeInfo{}, errors.New("not a Gun Mayhem host")
	}
	if err := validateNetMessageForClient(challenge); err != nil {
		return netProbeInfo{}, err
	}
	if err := writeNetMessage(conn, netWireMessage{Type: netMessageProbe, Protocol: netplayProtocolVersion}, maxClientWireFrame); err != nil {
		return netProbeInfo{}, err
	}
	_ = conn.SetReadDeadline(deadline)
	var reply netWireMessage
	if err := readNetMessage(conn, maxServerWireFrame, &reply); err != nil || reply.Type != netMessageProbeReply {
		return netProbeInfo{}, errors.New("host did not answer public probe")
	}
	if err := validateNetMessageForClient(reply); err != nil {
		return netProbeInfo{}, err
	}
	return reply.Probe, nil
}

func probeOpenNetplayServer(ctx context.Context, address string) (netProbeInfo, error) {
	address = strings.TrimSpace(address)
	if _, _, err := net.SplitHostPort(address); err != nil {
		return netProbeInfo{}, err
	}
	dialer := net.Dialer{Timeout: 1200 * time.Millisecond}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return netProbeInfo{}, err
	}
	return probeOpenNetplayConn(ctx, conn)
}

func probeOpenNetplayQUIC(ctx context.Context, address string) (netProbeInfo, error) {
	timeout := 10 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = remaining
		}
	}
	if timeout <= 0 {
		return netProbeInfo{}, context.DeadlineExceeded
	}
	conn, err := dialP2PQUIC(address, "", timeout)
	if err != nil {
		return netProbeInfo{}, err
	}
	return probeOpenNetplayConn(ctx, conn)
}

func verifyGlobalOpenServers(candidates []multiplayerServerInfo, timeout time.Duration) []multiplayerServerInfo {
	if len(candidates) == 0 || timeout <= 0 {
		return nil
	}
	if len(candidates) > 32 {
		candidates = candidates[:32]
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	type result struct {
		key    string
		server multiplayerServerInfo
		ok     bool
	}
	results := make(chan result, len(candidates))
	sem := make(chan struct{}, 8)
	for _, candidate := range candidates {
		candidate := candidate
		originalAddress := candidate.Address
		go func() {
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results <- result{key: originalAddress}
				return
			}
			defer func() { <-sem }()
			var info netProbeInfo
			var err error
			if strings.EqualFold(candidate.Transport, "quic") {
				// Ask the host itself for browser metadata over the same UDP endpoint
				// that DHT discovered. Besides being much cheaper than a QUIC session,
				// the reply source reveals the NAT endpoint that is actually reachable
				// from this client and primes the hole-punch path for the later JOIN.
				metaCtx, metaCancel := context.WithTimeout(ctx, 5*time.Second)
				var reachable string
				info, reachable, err = probeP2PHostMetadata(metaCtx, candidate.Address)
				metaCancel()
				if err == nil && strings.TrimSpace(reachable) != "" {
					candidate.Address = strings.TrimSpace(reachable)
				} else {
					info, err = probeOpenNetplayQUIC(ctx, candidate.Address)
				}
			} else {
				info, err = probeOpenNetplayServer(ctx, candidate.Address)
			}
			if err != nil {
				results <- result{key: originalAddress}
				return
			}
			candidate = applyProbeInfoToServer(candidate, info)
			if strings.TrimSpace(candidate.Name) == "" {
				candidate.Name = "Gun Mayhem Host"
			}
			results <- result{key: originalAddress, server: candidate, ok: true}
		}()
	}
	// DHT is only a rendezvous mechanism. Do not expose raw peer addresses as
	// playable servers until the host itself answers our authenticated protocol
	// probe. Mainline DHT entries can be stale for minutes and a NAT endpoint can
	// be discoverable even though no usable Gun Mayhem path exists. Showing those
	// entries produced the misleading ?/4 + no-map rows that then could not join.
	verified := make(map[string]multiplayerServerInfo, len(candidates))
	for range candidates {
		select {
		case got := <-results:
			if got.ok {
				verified[got.key] = got.server
			}
		case <-ctx.Done():
			out := make([]multiplayerServerInfo, 0, len(verified))
			for _, candidate := range candidates {
				if server, ok := verified[candidate.Address]; ok {
					out = append(out, server)
				}
			}
			return out
		}
	}
	out := make([]multiplayerServerInfo, 0, len(verified))
	for _, candidate := range candidates {
		if server, ok := verified[candidate.Address]; ok {
			out = append(out, server)
		}
	}
	return out
}

func (a *globalDHTAdvertiser) setStatus(ready bool, port int, err error) {
	if a == nil {
		return
	}
	a.mu.Lock()
	a.ready = ready
	a.port = port
	a.lastErr = err
	a.mu.Unlock()
}

func (a *globalDHTAdvertiser) Status() (ready bool, port int, err error) {
	if a == nil {
		return false, 0, nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.ready, a.port, a.lastErr
}

func (a *globalDHTAdvertiser) Close() error {
	if a == nil || a.closed.Swap(true) {
		return nil
	}
	close(a.stop)
	return nil
}

func parseCompactDHTNodes(raw []byte) []dhtLiteNode {
	if len(raw) == 0 || len(raw)%26 != 0 || len(raw) > 26*256 {
		return nil
	}
	out := make([]dhtLiteNode, 0, len(raw)/26)
	for off := 0; off+26 <= len(raw); off += 26 {
		var id [20]byte
		copy(id[:], raw[off:off+20])
		ip := netip.AddrFrom4([4]byte{raw[off+20], raw[off+21], raw[off+22], raw[off+23]})
		port := int(binary.BigEndian.Uint16(raw[off+24 : off+26]))
		if port <= 0 || !isInternetRoutableAddr(ip) {
			continue
		}
		out = append(out, dhtLiteNode{ID: id, Addr: &net.UDPAddr{IP: net.IP(ip.AsSlice()), Port: port}})
	}
	return out
}

func parseCompactDHTEndpoint(raw []byte) string {
	if len(raw) == 6 {
		ip := netip.AddrFrom4([4]byte{raw[0], raw[1], raw[2], raw[3]})
		port := int(binary.BigEndian.Uint16(raw[4:6]))
		if port > 0 && isInternetRoutableAddr(ip) {
			return net.JoinHostPort(ip.String(), strconv.Itoa(port))
		}
	}
	return ""
}

func parseCompactDHTPeers(values []any) []string {
	if len(values) == 0 || len(values) > 256 {
		return nil
	}
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, value := range values {
		raw, ok := value.([]byte)
		if !ok || len(raw) != 6 {
			continue
		}
		ip := netip.AddrFrom4([4]byte{raw[0], raw[1], raw[2], raw[3]})
		port := int(binary.BigEndian.Uint16(raw[4:6]))
		if port <= 0 || !isInternetRoutableAddr(ip) {
			continue
		}
		address := net.JoinHostPort(ip.String(), strconv.Itoa(port))
		if !seen[address] {
			seen[address] = true
			out = append(out, address)
		}
	}
	return out
}

// The bencode decoder below is deliberately tiny and bounded because every DHT
// datagram is untrusted Internet input. It accepts only the BEP-5 shapes we
// need, with strict packet/depth/container/string limits.
type bdecoder struct {
	data []byte
	pos  int
}

func decodeBencodeMessage(data []byte) (map[string]any, error) {
	if len(data) == 0 || len(data) > globalDHTPacketMax {
		return nil, errors.New("invalid DHT packet size")
	}
	d := &bdecoder{data: data}
	v, err := d.value(0)
	if err != nil || d.pos != len(data) {
		return nil, errors.New("invalid DHT bencode")
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, errors.New("DHT message is not a dictionary")
	}
	return m, nil
}

func (d *bdecoder) value(depth int) (any, error) {
	if d == nil || depth > 8 || d.pos >= len(d.data) {
		return nil, errors.New("invalid bencode value")
	}
	switch d.data[d.pos] {
	case 'd':
		return d.dict(depth + 1)
	case 'l':
		return d.list(depth + 1)
	case 'i':
		return d.integer()
	default:
		if d.data[d.pos] >= '0' && d.data[d.pos] <= '9' {
			return d.str()
		}
		return nil, errors.New("unsupported bencode type")
	}
}

func (d *bdecoder) str() ([]byte, error) {
	start := d.pos
	for d.pos < len(d.data) && d.data[d.pos] >= '0' && d.data[d.pos] <= '9' {
		d.pos++
		if d.pos-start > 8 {
			return nil, errors.New("bencode length too long")
		}
	}
	if d.pos == start || d.pos >= len(d.data) || d.data[d.pos] != ':' {
		return nil, errors.New("invalid bencode string")
	}
	n64, err := strconv.ParseInt(string(d.data[start:d.pos]), 10, 32)
	if err != nil || n64 < 0 || n64 > globalDHTPacketMax {
		return nil, errors.New("invalid bencode string length")
	}
	d.pos++
	n := int(n64)
	if d.pos+n > len(d.data) {
		return nil, errors.New("truncated bencode string")
	}
	out := append([]byte(nil), d.data[d.pos:d.pos+n]...)
	d.pos += n
	return out, nil
}

func (d *bdecoder) integer() (int64, error) {
	d.pos++
	start := d.pos
	for d.pos < len(d.data) && d.data[d.pos] != 'e' {
		d.pos++
		if d.pos-start > 24 {
			return 0, errors.New("bencode integer too long")
		}
	}
	if d.pos >= len(d.data) || d.pos == start {
		return 0, errors.New("invalid bencode integer")
	}
	raw := string(d.data[start:d.pos])
	d.pos++
	if (len(raw) > 1 && raw[0] == '0') || strings.HasPrefix(raw, "-0") {
		return 0, errors.New("non-canonical bencode integer")
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (d *bdecoder) list(depth int) ([]any, error) {
	d.pos++
	out := make([]any, 0, 8)
	for {
		if d.pos >= len(d.data) {
			return nil, errors.New("truncated bencode list")
		}
		if d.data[d.pos] == 'e' {
			d.pos++
			return out, nil
		}
		if len(out) >= 256 {
			return nil, errors.New("bencode list too large")
		}
		v, err := d.value(depth)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
}

func (d *bdecoder) dict(depth int) (map[string]any, error) {
	d.pos++
	out := make(map[string]any)
	lastKey := ""
	for {
		if d.pos >= len(d.data) {
			return nil, errors.New("truncated bencode dictionary")
		}
		if d.data[d.pos] == 'e' {
			d.pos++
			return out, nil
		}
		if len(out) >= 64 {
			return nil, errors.New("bencode dictionary too large")
		}
		keyRaw, err := d.str()
		if err != nil || len(keyRaw) == 0 || len(keyRaw) > 64 {
			return nil, errors.New("invalid bencode dictionary key")
		}
		key := string(keyRaw)
		if lastKey != "" && key <= lastKey {
			return nil, errors.New("unsorted or duplicate bencode key")
		}
		lastKey = key
		v, err := d.value(depth)
		if err != nil {
			return nil, err
		}
		out[key] = v
	}
}

func bgetBytes(m map[string]any, key string) []byte {
	if m == nil {
		return nil
	}
	v, _ := m[key].([]byte)
	return v
}

func bgetMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, _ := m[key].(map[string]any)
	return v
}

func bgetList(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	v, _ := m[key].([]any)
	return v
}

func bencodeMap(m map[string]any) []byte {
	var buf bytes.Buffer
	if writeBencodeMap(&buf, m, 0) != nil {
		return nil
	}
	return buf.Bytes()
}

func writeBencodeMap(buf *bytes.Buffer, m map[string]any, depth int) error {
	if buf == nil || depth > 8 || len(m) > 64 {
		return errors.New("invalid bencode map")
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	buf.WriteByte('d')
	for _, key := range keys {
		writeBencodeBytes(buf, []byte(key))
		if err := writeBencodeValue(buf, m[key], depth+1); err != nil {
			return err
		}
	}
	buf.WriteByte('e')
	return nil
}

func writeBencodeValue(buf *bytes.Buffer, value any, depth int) error {
	if depth > 8 {
		return errors.New("bencode nesting too deep")
	}
	switch v := value.(type) {
	case []byte:
		writeBencodeBytes(buf, v)
	case string:
		writeBencodeBytes(buf, []byte(v))
	case int:
		buf.WriteByte('i')
		buf.WriteString(strconv.Itoa(v))
		buf.WriteByte('e')
	case int64:
		buf.WriteByte('i')
		buf.WriteString(strconv.FormatInt(v, 10))
		buf.WriteByte('e')
	case map[string]any:
		return writeBencodeMap(buf, v, depth)
	case []any:
		if len(v) > 256 {
			return errors.New("bencode list too large")
		}
		buf.WriteByte('l')
		for _, item := range v {
			if err := writeBencodeValue(buf, item, depth+1); err != nil {
				return err
			}
		}
		buf.WriteByte('e')
	default:
		return fmt.Errorf("unsupported bencode value %T", value)
	}
	return nil
}

func writeBencodeBytes(buf *bytes.Buffer, data []byte) {
	buf.WriteString(strconv.Itoa(len(data)))
	buf.WriteByte(':')
	buf.Write(data)
}
