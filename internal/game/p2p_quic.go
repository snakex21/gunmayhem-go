package game

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	quic "github.com/quic-go/quic-go"
)

const (
	p2pPunchPrefix       = "\x1fGMP8"
	p2pMetadataRequest   = "\x1fGMM8Q"
	p2pMetadataResponse  = "\x1fGMM8R"
	p2pMetadataMaxPacket = 4096
	p2pStreamPreface     = "GMP8S"
	p2pPunchPoll         = 800 * time.Millisecond
	p2pPunchBurstEvery   = 400 * time.Millisecond
	p2pPunchLookup       = 2200 * time.Millisecond
	p2pDialTimeout       = 20 * time.Second
	p2pAnnounceTimeout   = 8 * time.Second
	p2pReannounceEvery   = 10 * time.Minute
	p2pMaxPunchPeers     = 32
	p2pPortPredictRadius = 2
)

type p2pPacket struct {
	data []byte
	addr net.Addr
}

// p2pPacketMux lets QUIC and our tiny BEP-5 client safely share one UDP socket.
// This is important for NAT traversal: the endpoint DHT observes is the exact
// endpoint later used by QUIC, rather than a different ephemeral UDP mapping.
type p2pPacketMux struct {
	conn *net.UDPConn

	quicCh  chan p2pPacket
	dhtCh   chan p2pPacket
	metaCh  chan p2pPacket
	punchCh chan *net.UDPAddr
	done    chan struct{}
	closed  atomic.Bool
}

func newP2PPacketMux() (*p2pPacketMux, error) {
	return newP2PPacketMuxAtPort(0)
}

func newP2PPacketMuxAtPort(preferredPort int) (*p2pPacketMux, error) {
	if preferredPort < 0 || preferredPort > 65535 {
		preferredPort = 0
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: preferredPort})
	if err != nil && preferredPort != 0 {
		// Development can run more than one host on a machine. Production gets
		// the stable gameplay port, while a second local instance safely falls
		// back to an ephemeral socket instead of failing to start.
		conn, err = net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	}
	if err != nil {
		return nil, err
	}
	m := &p2pPacketMux{
		conn:    conn,
		quicCh:  make(chan p2pPacket, 128),
		dhtCh:   make(chan p2pPacket, 128),
		metaCh:  make(chan p2pPacket, 64),
		punchCh: make(chan *net.UDPAddr, 64),
		done:    make(chan struct{}),
	}
	go m.readLoop()
	return m, nil
}

func (m *p2pPacketMux) readLoop() {
	defer close(m.done)
	buf := make([]byte, 64*1024)
	for !m.closed.Load() {
		n, addr, err := m.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		if n <= 0 || addr == nil {
			continue
		}
		data := append([]byte(nil), buf[:n]...)
		if strings.HasPrefix(string(data), p2pMetadataRequest) || strings.HasPrefix(string(data), p2pMetadataResponse) {
			if len(data) <= p2pMetadataMaxPacket {
				select {
				case m.metaCh <- p2pPacket{data: data, addr: cloneUDPAddr(addr)}:
				default:
				}
			}
			continue
		}
		if strings.HasPrefix(string(data), p2pPunchPrefix) {
			select {
			case m.punchCh <- cloneUDPAddr(addr):
			default:
			}
			continue
		}
		// DHT responses are bencoded dictionaries. QUIC short headers can share
		// the same high-bit pattern as ASCII 'd', so validate the BEP-5 shape
		// before routing away from QUIC.
		if len(data) > 0 && data[0] == 'd' {
			if msg, decErr := decodeBencodeMessage(data); decErr == nil && len(bgetBytes(msg, "t")) > 0 && len(bgetBytes(msg, "y")) > 0 {
				select {
				case m.dhtCh <- p2pPacket{data: data, addr: cloneUDPAddr(addr)}:
				default:
				}
				continue
			}
		}
		select {
		case m.quicCh <- p2pPacket{data: data, addr: cloneUDPAddr(addr)}:
		case <-m.done:
			return
		}
	}
}

func cloneUDPAddr(addr *net.UDPAddr) *net.UDPAddr {
	if addr == nil {
		return nil
	}
	return &net.UDPAddr{IP: append(net.IP(nil), addr.IP...), Port: addr.Port, Zone: addr.Zone}
}

func (m *p2pPacketMux) ReadFrom(p []byte) (int, net.Addr, error) {
	if m == nil {
		return 0, nil, net.ErrClosed
	}
	select {
	case pkt := <-m.quicCh:
		n := copy(p, pkt.data)
		if n < len(pkt.data) {
			return n, pkt.addr, io.ErrShortBuffer
		}
		return n, pkt.addr, nil
	case <-m.done:
		return 0, nil, net.ErrClosed
	}
}

func (m *p2pPacketMux) WriteTo(p []byte, addr net.Addr) (int, error) {
	if m == nil || m.conn == nil || m.closed.Load() {
		return 0, net.ErrClosed
	}
	return m.conn.WriteTo(p, addr)
}

func (m *p2pPacketMux) Close() error {
	if m == nil || m.closed.Swap(true) {
		return nil
	}
	return m.conn.Close()
}

func (m *p2pPacketMux) LocalAddr() net.Addr {
	if m == nil || m.conn == nil {
		return &net.UDPAddr{}
	}
	return m.conn.LocalAddr()
}

func (m *p2pPacketMux) SetDeadline(time.Time) error      { return nil }
func (m *p2pPacketMux) SetReadDeadline(time.Time) error  { return nil }
func (m *p2pPacketMux) SetWriteDeadline(time.Time) error { return nil }

type quicStreamNetConn struct {
	stream    *quic.Stream
	conn      *quic.Conn
	local     net.Addr
	remote    net.Addr
	closeOnce sync.Once
	closeFn   func()
}

func (c *quicStreamNetConn) Read(p []byte) (int, error)         { return c.stream.Read(p) }
func (c *quicStreamNetConn) Write(p []byte) (int, error)        { return c.stream.Write(p) }
func (c *quicStreamNetConn) LocalAddr() net.Addr                { return c.local }
func (c *quicStreamNetConn) RemoteAddr() net.Addr               { return c.remote }
func (c *quicStreamNetConn) SetDeadline(t time.Time) error      { return c.stream.SetDeadline(t) }
func (c *quicStreamNetConn) SetReadDeadline(t time.Time) error  { return c.stream.SetReadDeadline(t) }
func (c *quicStreamNetConn) SetWriteDeadline(t time.Time) error { return c.stream.SetWriteDeadline(t) }
func (c *quicStreamNetConn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		err = c.stream.Close()
		if c.conn != nil {
			_ = c.conn.CloseWithError(0, "netplay closed")
		}
		if c.closeFn != nil {
			c.closeFn()
		}
	})
	return err
}

var (
	p2pServerTLSOnce sync.Once
	p2pServerTLS     *tls.Config
	p2pServerTLSErr  error
)

func p2pServerTLSConfig() (*tls.Config, error) {
	p2pServerTLSOnce.Do(func() {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			p2pServerTLSErr = err
			return
		}
		serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
		if err != nil {
			p2pServerTLSErr = err
			return
		}
		tmpl := x509.Certificate{
			SerialNumber: serial,
			Subject:      pkix.Name{CommonName: "Gun Mayhem P2P"},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(365 * 24 * time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		}
		der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, pub, priv)
		if err != nil {
			p2pServerTLSErr = err
			return
		}
		keyDER, err := x509.MarshalPKCS8PrivateKey(priv)
		if err != nil {
			p2pServerTLSErr = err
			return
		}
		cert, err := tls.X509KeyPair(
			pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
			pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}),
		)
		if err != nil {
			p2pServerTLSErr = err
			return
		}
		p2pServerTLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS13, NextProtos: []string{p2pALPN()}}
	})
	if p2pServerTLSErr != nil {
		return nil, p2pServerTLSErr
	}
	return p2pServerTLS.Clone(), nil
}

func p2pClientTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS13,
		NextProtos:         []string{p2pALPN()},
		InsecureSkipVerify: true, // app-level protocol/HMAC authenticates invite-only sessions
	}
}

func p2pALPN() string { return fmt.Sprintf("gunmayhem-p2p-%d", netplayProtocolVersion) }

func p2pQUICConfig() *quic.Config {
	return &quic.Config{
		HandshakeIdleTimeout:  4 * time.Second,
		MaxIdleTimeout:        20 * time.Second,
		KeepAlivePeriod:       5 * time.Second,
		MaxIncomingStreams:    4,
		MaxIncomingUniStreams: 0,
	}
}

type p2pQUICHost struct {
	closed atomic.Bool
	stop   chan struct{}

	session    *netplaySession
	visibility int
	sessionKey string
	mux        *p2pPacketMux
	transport  *quic.Transport
	listener   *quic.Listener
	dht        *dhtLiteClient

	mu              sync.RWMutex
	publicEndpoint  string
	publicEndpoints []string
	dhtReady        bool
	lastErr         error
}

func startP2PQUICHost(session *netplaySession, visibility int, sessionKey string) (*p2pQUICHost, error) {
	if session == nil || session.mode != netplayHost || strings.TrimSpace(sessionKey) == "" {
		return nil, errors.New("invalid P2P host configuration")
	}
	preferredUDPPort := 0
	if tcpAddr, ok := session.listener.Addr().(*net.TCPAddr); ok {
		preferredUDPPort = tcpAddr.Port
	}
	mux, err := newP2PPacketMuxAtPort(preferredUDPPort)
	if err != nil {
		return nil, err
	}
	transport := &quic.Transport{Conn: mux, ConnectionIDLength: 8}
	tlsConf, err := p2pServerTLSConfig()
	if err != nil {
		_ = transport.Close()
		_ = mux.Close()
		return nil, err
	}
	listener, err := transport.Listen(tlsConf, p2pQUICConfig())
	if err != nil {
		_ = transport.Close()
		_ = mux.Close()
		return nil, err
	}
	dht, err := newDHTLiteClientOnMux(mux)
	if err != nil {
		_ = listener.Close()
		_ = transport.Close()
		_ = mux.Close()
		return nil, err
	}
	h := &p2pQUICHost{
		stop: make(chan struct{}), session: session, visibility: visibility,
		sessionKey: strings.TrimSpace(sessionKey), mux: mux, transport: transport,
		listener: listener, dht: dht,
	}
	networkDiagnostic("P2P host socket local=%s preferredUDP=%d", mux.LocalAddr(), preferredUDPPort)
	go h.acceptLoop()
	go h.metadataLoop()
	go h.discoveryLoop()
	return h, nil
}

func privateHostDHTHash(sessionKey string) [20]byte {
	return sha1.Sum([]byte("GunMayhem-RE-private-host-16|" + strings.TrimSpace(sessionKey)))
}

func punchDHTHash(hostEndpoint string) [20]byte {
	return sha1.Sum([]byte("GunMayhem-RE-punch-16|" + strings.TrimSpace(hostEndpoint)))
}

func p2pPortCandidates(addr *net.UDPAddr) []*net.UDPAddr {
	if addr == nil || addr.Port <= 0 || addr.Port > 65535 {
		return nil
	}
	out := []*net.UDPAddr{cloneUDPAddr(addr)}
	for delta := 1; delta <= p2pPortPredictRadius; delta++ {
		for _, port := range []int{addr.Port - delta, addr.Port + delta} {
			if port <= 0 || port > 65535 {
				continue
			}
			candidate := cloneUDPAddr(addr)
			candidate.Port = port
			out = append(out, candidate)
		}
	}
	return out
}

func sendP2PPunchBurst(transport *quic.Transport, addr *net.UDPAddr) {
	if transport == nil || addr == nil {
		return
	}
	for n := 0; n < 3; n++ {
		_, _ = transport.WriteTo([]byte(p2pPunchPrefix), addr)
	}
}

func sendP2PMetadataRequest(mux *p2pPacketMux, addr *net.UDPAddr) {
	if mux == nil || addr == nil {
		return
	}
	for n := 0; n < 2; n++ {
		_, _ = mux.WriteTo([]byte(p2pMetadataRequest), addr)
	}
}

func probeP2PHostMetadata(ctx context.Context, hostAddress string) (netProbeInfo, string, error) {
	hostAddress = strings.TrimSpace(hostAddress)
	hostUDP, err := net.ResolveUDPAddr("udp4", hostAddress)
	if err != nil {
		return netProbeInfo{}, "", err
	}
	mux, err := newP2PPacketMux()
	if err != nil {
		return netProbeInfo{}, "", err
	}
	defer mux.Close()
	dht, err := newDHTLiteClientOnMux(mux)
	if err != nil {
		return netProbeInfo{}, "", err
	}
	defer dht.Close()

	announceCtx, announceCancel := context.WithCancel(ctx)
	defer announceCancel()
	go func() {
		announcePort := 0
		if local, ok := mux.LocalAddr().(*net.UDPAddr); ok && local != nil {
			announcePort = local.Port
		}
		if announcePort > 0 {
			_ = announceDHTPeerWithClient(announceCtx, dht, punchDHTHash(hostAddress), announcePort, true)
		}
	}()

	targets := p2pPortCandidates(hostUDP)
	for _, target := range targets {
		sendP2PMetadataRequest(mux, target)
	}
	ticker := time.NewTicker(350 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case pkt := <-mux.metaCh:
			if !strings.HasPrefix(string(pkt.data), p2pMetadataResponse) || pkt.addr == nil {
				continue
			}
			payload := pkt.data[len(p2pMetadataResponse):]
			if len(payload) == 0 || len(payload) > p2pMetadataMaxPacket-len(p2pMetadataResponse) {
				continue
			}
			var info netProbeInfo
			if json.Unmarshal(payload, &info) != nil || validateNetProbeInfo(info) != nil {
				continue
			}
			return info, pkt.addr.String(), nil
		case punched := <-mux.punchCh:
			if punched == nil {
				continue
			}
			sendP2PMetadataRequest(mux, punched)
			for _, candidate := range p2pPortCandidates(punched) {
				sendP2PMetadataRequest(mux, candidate)
			}
		case <-ticker.C:
			for _, target := range targets {
				sendP2PMetadataRequest(mux, target)
			}
		case <-ctx.Done():
			return netProbeInfo{}, "", ctx.Err()
		case <-mux.done:
			return netProbeInfo{}, "", net.ErrClosed
		}
	}
}

func discoverPunchPeers(endpoints []string, timeout time.Duration) []string {
	if len(endpoints) == 0 || timeout <= 0 {
		return nil
	}
	if len(endpoints) > 16 {
		endpoints = endpoints[:16]
	}
	type lookupResult struct{ peers []string }
	results := make(chan lookupResult, len(endpoints))
	for _, endpoint := range endpoints {
		endpoint := endpoint
		go func() {
			peers, _ := discoverDHTPeerAddresses(punchDHTHash(endpoint), timeout)
			results <- lookupResult{peers: peers}
		}()
	}
	seen := make(map[string]bool)
	out := make([]string, 0, p2pMaxPunchPeers)
	deadline := time.NewTimer(timeout + 250*time.Millisecond)
	defer deadline.Stop()
	for range endpoints {
		select {
		case result := <-results:
			for _, peer := range result.peers {
				if seen[peer] {
					continue
				}
				seen[peer] = true
				out = append(out, peer)
				if len(out) >= p2pMaxPunchPeers {
					return out
				}
			}
		case <-deadline.C:
			return out
		}
	}
	return out
}

func chooseObservedEndpoint(observed []string) string {
	counts := make(map[string]int)
	best := ""
	bestCount := 0
	for _, endpoint := range observed {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" {
			continue
		}
		counts[endpoint]++
		if counts[endpoint] > bestCount {
			best = endpoint
			bestCount = counts[endpoint]
		}
	}
	return best
}

func (h *p2pQUICHost) metadataLoop() {
	if h == nil || h.mux == nil || h.session == nil {
		return
	}
	for !h.closed.Load() {
		select {
		case pkt := <-h.mux.metaCh:
			if h.visibility != onlineSessionOpen || string(pkt.data) != p2pMetadataRequest || pkt.addr == nil {
				continue
			}
			info := h.session.publicProbeInfo()
			payload, err := json.Marshal(info)
			if err != nil || len(payload)+len(p2pMetadataResponse) > p2pMetadataMaxPacket {
				continue
			}
			message := append([]byte(p2pMetadataResponse), payload...)
			_, _ = h.mux.WriteTo(message, pkt.addr)
		case <-h.stop:
			return
		case <-h.mux.done:
			return
		}
	}
}

func (h *p2pQUICHost) acceptLoop() {
	for !h.closed.Load() {
		qc, err := h.listener.Accept(context.Background())
		if err != nil {
			if !h.closed.Load() {
				h.setStatus("", false, err)
			}
			return
		}
		networkDiagnostic("P2P host accepted QUIC remote=%s", qc.RemoteAddr())
		go func(conn *quic.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			stream, err := conn.AcceptStream(ctx)
			if err != nil {
				_ = conn.CloseWithError(1, "stream required")
				return
			}
			_ = stream.SetReadDeadline(time.Now().Add(2 * time.Second))
			preface := make([]byte, len(p2pStreamPreface))
			if _, err := io.ReadFull(stream, preface); err != nil || string(preface) != p2pStreamPreface {
				_ = conn.CloseWithError(2, "invalid Gun Mayhem stream")
				return
			}
			_ = stream.SetReadDeadline(time.Time{})
			nc := &quicStreamNetConn{stream: stream, conn: conn, local: h.mux.LocalAddr(), remote: conn.RemoteAddr()}
			if !h.session.attachIncomingConn(nc) {
				_ = nc.Close()
			}
		}(qc)
	}
}

func (h *p2pQUICHost) discoveryLoop() {
	privateHash := privateHostDHTHash(h.sessionKey)
	announce := func() (string, error) {
		// OPEN publishes only to the public browser rendezvous. INVITE ONLY stays
		// off that swarm and publishes only private/code-derived rendezvous data.
		code := strings.TrimSpace(h.session.shortJoinCode)
		hashes := [][20]byte{globalOpenDHTInfoHashAt(time.Now())}
		if h.visibility == onlineSessionInviteOnly {
			if code == "" {
				code = decentralizedOpenCode(h.sessionKey)
			}
			hashes = [][20]byte{privateHash, decentralizedOpenCodeDHTHash(code)}
		}
		announcePort := 0
		if local, ok := h.mux.LocalAddr().(*net.UDPAddr); ok && local != nil {
			announcePort = local.Port
		}
		if announcePort <= 0 {
			return "", errors.New("P2P host has no usable UDP port")
		}
		announceCtx, announceCancel := context.WithTimeout(context.Background(), p2pAnnounceTimeout)
		defer announceCancel()
		// Some public DHT nodes ignore implied_port. Supplying the real local
		// socket port as a fallback prevents them from publishing the bogus
		// endpoint :1. Nodes that support BEP 5 still store the observed NAT
		// source port, while extension-aware responses override it below.
		type result struct {
			observed []string
			err      error
		}
		results := make(chan result, len(hashes))
		for _, hash := range hashes {
			hash := hash
			go func() {
				observed, err := announceDHTPeerWithClientObserved(announceCtx, h.dht, hash, announcePort, true)
				results <- result{observed: observed, err: err}
			}()
		}
		var observed []string
		var announceErr error
		successes := 0
		for range hashes {
			result := <-results
			observed = append(observed, result.observed...)
			if result.err != nil {
				announceErr = errors.Join(announceErr, result.err)
			} else {
				successes++
			}
		}
		// Read the short-code record back through an independent DHT traversal.
		// Multi-homed/VPN hosts can be stored under a public route that differs
		// from the address extension returned by the announce node. Keeping both
		// makes the host poll the same punch rendezvous the remote client uses.
		if h.visibility == onlineSessionInviteOnly && validDecentralizedOpenCode(code) {
			published, _ := discoverDHTPeerAddresses(decentralizedOpenCodeDHTHash(code), 3*time.Second)
			observed = append(observed, published...)
		}
		h.setObservedEndpoints(observed)
		endpoint := chooseObservedEndpoint(observed)
		if successes != len(hashes) && announceErr == nil {
			announceErr = errors.New("not all P2P rendezvous hashes were published")
		}
		return endpoint, announceErr
	}

	for attempt := 0; attempt < 4 && !h.closed.Load(); attempt++ {
		endpoint, err := announce()
		networkDiagnostic("P2P DHT announce attempt=%d endpoint=%s observed=%v err=%v", attempt+1, endpoint, h.ObservedEndpoints(), err)
		if endpoint != "" {
			h.setStatus(endpoint, err == nil, err)
			if err == nil {
				break
			}
		} else {
			h.setStatus("", false, err)
		}
		time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
	}

	reannounce := time.NewTicker(p2pReannounceEvery)
	defer reannounce.Stop()
	punchTick := time.NewTicker(p2pPunchPoll)
	defer punchTick.Stop()
	for !h.closed.Load() {
		select {
		case <-reannounce.C:
			endpoint, err := announce()
			h.setStatus(endpoint, err == nil, err)
		case punched := <-h.mux.punchCh:
			// Reply from the same UDP socket so the client learns the mapping the
			// host's NAT created specifically toward that client.
			sendP2PPunchBurst(h.transport, punched)
		case <-punchTick.C:
			endpoints := h.ObservedEndpoints()
			if len(endpoints) == 0 {
				continue
			}
			// Every DHT node may observe a different port through symmetric NAT.
			// Query all rendezvous hashes concurrently; the old sequential loop
			// could exceed the client's entire JOIN timeout before reaching the
			// endpoint that the client actually discovered.
			peers := discoverPunchPeers(endpoints, p2pPunchLookup)
			if len(peers) > 0 {
				networkDiagnostic("P2P host punch rendezvous endpoints=%v clientPeers=%v", endpoints, peers)
			}
			for _, peer := range peers {
				udp, err := net.ResolveUDPAddr("udp4", peer)
				if err != nil {
					continue
				}
				// Symmetric NATs may allocate nearby-but-different ports for each
				// destination. Try a tiny bounded neighborhood around DHT-observed
				// ports; this is prediction, not an unbounded port scan.
				for _, candidate := range p2pPortCandidates(udp) {
					sendP2PPunchBurst(h.transport, candidate)
				}
			}
		case <-h.stop:
			return
		}
	}
}

func (h *p2pQUICHost) setObservedEndpoints(observed []string) {
	if h == nil {
		return
	}
	seen := make(map[string]bool)
	clean := make([]string, 0, len(observed))
	for _, endpoint := range observed {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" || seen[endpoint] {
			continue
		}
		if _, _, err := net.SplitHostPort(endpoint); err != nil {
			continue
		}
		seen[endpoint] = true
		clean = append(clean, endpoint)
	}
	h.mu.Lock()
	h.publicEndpoints = clean
	if h.publicEndpoint == "" && len(clean) > 0 {
		h.publicEndpoint = clean[0]
	}
	h.mu.Unlock()
}

func (h *p2pQUICHost) ObservedEndpoints() []string {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := append([]string(nil), h.publicEndpoints...)
	if len(out) == 0 && h.publicEndpoint != "" {
		out = append(out, h.publicEndpoint)
	}
	return out
}

func (h *p2pQUICHost) setStatus(endpoint string, ready bool, err error) {
	h.mu.Lock()
	if endpoint != "" {
		h.publicEndpoint = endpoint
	}
	h.dhtReady = ready
	h.lastErr = err
	h.mu.Unlock()
}

func (h *p2pQUICHost) Status() (endpoint string, ready bool, err error) {
	if h == nil {
		return "", false, nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.publicEndpoint, h.dhtReady, h.lastErr
}

func (h *p2pQUICHost) Close() error {
	if h == nil || h.closed.Swap(true) {
		return nil
	}
	close(h.stop)
	if h.dht != nil {
		_ = h.dht.Close()
	}
	if h.listener != nil {
		_ = h.listener.Close()
	}
	// Close the underlying socket before Transport.Close so the transport's
	// blocking ReadFrom wakes up instead of deadlocking shutdown.
	if h.mux != nil {
		_ = h.mux.Close()
	}
	if h.transport != nil {
		_ = h.transport.Close()
	}
	return nil
}

func dialP2PQUIC(hostAddress, sessionKey string, timeout time.Duration) (net.Conn, error) {
	hostAddress = strings.TrimSpace(hostAddress)
	hostUDP, err := net.ResolveUDPAddr("udp4", hostAddress)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = p2pDialTimeout
	}
	mux, err := newP2PPacketMux()
	if err != nil {
		return nil, err
	}
	networkDiagnostic("P2P dial start host=%s local=%s timeout=%s", hostAddress, mux.LocalAddr(), timeout)
	transport := &quic.Transport{Conn: mux, ConnectionIDLength: 8}
	dht, err := newDHTLiteClientOnMux(mux)
	if err != nil {
		_ = transport.Close()
		_ = mux.Close()
		return nil, err
	}
	cleanup := func() {
		_ = dht.Close()
		_ = mux.Close()
		_ = transport.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	punchHash := punchDHTHash(hostAddress)
	announceDone := make(chan struct{})
	go func() {
		defer close(announceDone)
		announceCtx, announceCancel := context.WithTimeout(ctx, 6*time.Second)
		defer announceCancel()
		announcePort := 0
		if local, ok := mux.LocalAddr().(*net.UDPAddr); ok && local != nil {
			announcePort = local.Port
		}
		if announcePort > 0 {
			_ = announceDHTPeerWithClient(announceCtx, dht, punchHash, announcePort, true)
		}
	}()

	type dialResult struct {
		conn *quic.Conn
		err  error
	}
	results := make(chan dialResult, 32)
	seen := map[string]bool{}
	var seenMu sync.Mutex
	tryDial := func(addr *net.UDPAddr) {
		if addr == nil {
			return
		}
		key := addr.String()
		seenMu.Lock()
		if seen[key] {
			seenMu.Unlock()
			return
		}
		seen[key] = true
		seenMu.Unlock()
		go func() {
			qc, err := transport.Dial(ctx, addr, p2pClientTLSConfig(), p2pQUICConfig())
			select {
			case results <- dialResult{conn: qc, err: err}:
			case <-ctx.Done():
				if qc != nil {
					_ = qc.CloseWithError(0, "dial cancelled")
				}
			}
		}()
	}
	// Start punching and QUIC handshakes against the DHT-observed host port
	// plus a tiny bounded neighborhood for port-predictable symmetric NATs.
	for _, candidate := range p2pPortCandidates(hostUDP) {
		sendP2PPunchBurst(transport, candidate)
		sendP2PMetadataRequest(mux, candidate)
		tryDial(candidate)
	}
	// NAT bindings can expire or be destination-dependent while DHT rendezvous
	// is still completing. Keep sending tiny bounded bursts for the duration of
	// the JOIN instead of relying on one packet at time zero.
	stopPunchBursts := make(chan struct{})
	defer close(stopPunchBursts)
	go func() {
		ticker := time.NewTicker(p2pPunchBurstEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				for _, candidate := range p2pPortCandidates(hostUDP) {
					sendP2PPunchBurst(transport, candidate)
					sendP2PMetadataRequest(mux, candidate)
				}
			case <-stopPunchBursts:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case res := <-results:
			if res.err != nil || res.conn == nil {
				continue
			}
			stream, err := res.conn.OpenStreamSync(ctx)
			if err != nil {
				_ = res.conn.CloseWithError(1, "stream open failed")
				continue
			}
			_ = stream.SetWriteDeadline(time.Now().Add(2 * time.Second))
			if _, err := stream.Write([]byte(p2pStreamPreface)); err != nil {
				_ = res.conn.CloseWithError(2, "stream preface failed")
				continue
			}
			_ = stream.SetWriteDeadline(time.Time{})
			_ = dht.Close()
			networkDiagnostic("P2P dial connected host=%s remote=%s", hostAddress, res.conn.RemoteAddr())
			nc := &quicStreamNetConn{
				stream: stream, conn: res.conn, local: mux.LocalAddr(), remote: res.conn.RemoteAddr(),
				closeFn: func() { _ = mux.Close(); _ = transport.Close() },
			}
			return nc, nil
		case punched := <-mux.punchCh:
			networkDiagnostic("P2P dial received punch from=%s host=%s", punched, hostAddress)
			sendP2PPunchBurst(transport, punched)
			sendP2PMetadataRequest(mux, punched)
			tryDial(punched)
		case pkt := <-mux.metaCh:
			if !strings.HasPrefix(string(pkt.data), p2pMetadataResponse) {
				continue
			}
			payload := pkt.data[len(p2pMetadataResponse):]
			var info netProbeInfo
			if json.Unmarshal(payload, &info) != nil || validateNetProbeInfo(info) != nil {
				continue
			}
			if reachable, ok := pkt.addr.(*net.UDPAddr); ok && reachable != nil {
				networkDiagnostic("P2P dial metadata path host=%s reachable=%s session=%s", hostAddress, reachable, info.ID)
				sendP2PPunchBurst(transport, reachable)
				tryDial(reachable)
			}
		case <-ctx.Done():
			cleanup()
			networkDiagnostic("P2P dial timeout host=%s err=%v", hostAddress, ctx.Err())
			return nil, fmt.Errorf("P2P hole punch failed: %w", ctx.Err())
		}
	}
}

func discoverSingleDHTPeer(hash [20]byte, timeout time.Duration) (string, error) {
	peers, err := discoverDHTPeerAddresses(hash, timeout)
	if len(peers) > 0 {
		return peers[0], nil
	}
	if err != nil {
		return "", err
	}
	return "", errors.New("DHT peer not found")
}

func discoverDHTPeerAddresses(hash [20]byte, timeout time.Duration) ([]string, error) {
	if timeout <= 0 {
		timeout = globalDHTLookupTimeout
	}
	deadline := time.Now().Add(timeout)
	attempts := 1
	if timeout >= 4*time.Second {
		attempts = 2
	}
	seen := make(map[string]bool)
	out := make([]string, 0)
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		attemptBudget := remaining / time.Duration(attempts-attempt)
		ctx, cancel := context.WithTimeout(context.Background(), attemptBudget)
		client, err := newDHTLiteClient()
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		peers, _, traversalErr := dhtTraversal(ctx, client, hash)
		_ = client.Close()
		cancel()
		if traversalErr != nil {
			lastErr = traversalErr
		}
		for _, peer := range peers {
			if !seen[peer] {
				seen[peer] = true
				out = append(out, peer)
			}
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	if len(out) > 0 {
		return out, nil
	}
	if lastErr == nil {
		lastErr = errors.New("DHT peer not found")
	}
	return nil, lastErr
}
