package game

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// netplayTransport is the replaceable connection brick. The game session does
// not know whether a connection came from TCP, a relay, or QUIC; it only adopts
// the resulting net.Conn into a netplayLink.
type netplayTransport interface {
	Listen(context.Context, string) (net.Listener, error)
	Dial(context.Context, string) (net.Conn, error)
}

// netplayCodec is the replaceable wire-format brick. Framing and connection
// lifecycle stay independent from CBOR serialization.
type netplayCodec interface {
	Encode(netWireMessage) ([]byte, error)
	Decode([]byte, *netWireMessage) error
}

type tcpNetplayTransport struct{}

func (tcpNetplayTransport) Listen(ctx context.Context, address string) (net.Listener, error) {
	lc := net.ListenConfig{
		KeepAlive: 15 * time.Second,
		KeepAliveConfig: net.KeepAliveConfig{
			Enable:   true,
			Idle:     15 * time.Second,
			Interval: 5 * time.Second,
			Count:    3,
		},
	}
	return lc.Listen(ctx, netplayListenNetwork(address), address)
}

func netplayListenNetwork(address string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil {
		return "tcp"
	}
	host = strings.Trim(host, "[]")
	if host == "" {
		// Bind dual-stack when the OS allows it. This keeps the existing IPv4/LAN
		// path while also accepting native IPv6 Internet clients on the same port.
		return "tcp"
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "tcp"
	}
	if ip.To4() != nil {
		return "tcp4"
	}
	return "tcp6"
}

func (tcpNetplayTransport) Dial(ctx context.Context, address string) (net.Conn, error) {
	d := net.Dialer{
		KeepAlive: 15 * time.Second,
		KeepAliveConfig: net.KeepAliveConfig{
			Enable:   true,
			Idle:     15 * time.Second,
			Interval: 5 * time.Second,
			Count:    3,
		},
	}
	return d.DialContext(ctx, "tcp", address)
}

type cborNetplayCodec struct{}

func (cborNetplayCodec) Encode(msg netWireMessage) ([]byte, error) {
	payload, err := netCBOREnc.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("encode netplay message: %w", err)
	}
	return payload, nil
}

func (cborNetplayCodec) Decode(payload []byte, msg *netWireMessage) error {
	if msg == nil {
		return errors.New("nil netplay message target")
	}
	*msg = netWireMessage{}
	if err := netCBORDec.Unmarshal(payload, msg); err != nil {
		return fmt.Errorf("decode netplay message: %w", err)
	}
	return nil
}

type netplayStack struct {
	transport netplayTransport
	codec     netplayCodec
}

func defaultNetplayStack() *netplayStack {
	return &netplayStack{transport: tcpNetplayTransport{}, codec: cborNetplayCodec{}}
}

func normalizeNetplayStack(stack *netplayStack) *netplayStack {
	if stack == nil {
		return defaultNetplayStack()
	}
	transport := stack.transport
	if transport == nil {
		transport = tcpNetplayTransport{}
	}
	codec := stack.codec
	if codec == nil {
		codec = cborNetplayCodec{}
	}
	return &netplayStack{transport: transport, codec: codec}
}

func (s *netplayStack) listen(address string) (net.Listener, error) {
	s = normalizeNetplayStack(s)
	ln, err := s.transport.Listen(context.Background(), address)
	if err != nil {
		return nil, fmt.Errorf("listen for netplay on %s: %w", address, err)
	}
	return ln, nil
}

func (s *netplayStack) dial(address string, timeout time.Duration) (net.Conn, error) {
	s = normalizeNetplayStack(s)
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := s.transport.Dial(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", address, err)
	}
	return conn, nil
}

func (s *netplayStack) adopt(conn net.Conn) *netplayLink {
	s = normalizeNetplayStack(s)
	configureNetplayConn(conn)
	return &netplayLink{conn: conn, codec: s.codec}
}

func configureNetplayConn(conn net.Conn) {
	tcp, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}
	_ = tcp.SetNoDelay(true)
	_ = tcp.SetKeepAliveConfig(net.KeepAliveConfig{
		Enable:   true,
		Idle:     15 * time.Second,
		Interval: 5 * time.Second,
		Count:    3,
	})
	_ = tcp.SetReadBuffer(512 * 1024)
	_ = tcp.SetWriteBuffer(512 * 1024)
}

// netplayLink owns one duplex connection. It guarantees that a frame is
// written atomically even if a future feature adds another producer.
type netplayLink struct {
	conn    net.Conn
	codec   netplayCodec
	writeMu sync.Mutex
	closed  atomic.Bool
}

// netplayOutbox keeps realtime input separate from control state. A flood of
// 60 Hz input can therefore never evict a character configuration or a
// post-game ready change. Every lane keeps only its newest authoritative value.
type netplayOutbox struct {
	input   chan netWireMessage
	config  chan netWireMessage
	ready   chan netWireMessage
	control chan netWireMessage
}

func newNetplayOutbox() *netplayOutbox {
	return &netplayOutbox{
		input:   make(chan netWireMessage, 1),
		config:  make(chan netWireMessage, 1),
		ready:   make(chan netWireMessage, 1),
		control: make(chan netWireMessage, 1),
	}
}

func queueLatestNetplayMessage(ch chan netWireMessage, msg netWireMessage) {
	select {
	case ch <- msg:
		return
	default:
	}
	select {
	case <-ch:
	default:
	}
	select {
	case ch <- msg:
	default:
	}
}

func (o *netplayOutbox) Push(msg netWireMessage) {
	if o == nil {
		return
	}
	switch msg.Type {
	case netMessageInput:
		queueLatestNetplayMessage(o.input, msg)
	case netMessagePlayerConfig:
		queueLatestNetplayMessage(o.config, msg)
	case netMessagePostGameReady:
		queueLatestNetplayMessage(o.ready, msg)
	default:
		queueLatestNetplayMessage(o.control, msg)
	}
}

func (o *netplayOutbox) Next(stop, sessionDone <-chan struct{}) (netWireMessage, bool) {
	return o.nextWithHeartbeat(stop, sessionDone, nil)
}

func (o *netplayOutbox) nextWithHeartbeat(stop, sessionDone <-chan struct{}, heartbeat <-chan time.Time) (netWireMessage, bool) {
	if o == nil {
		return netWireMessage{}, false
	}
	select {
	case msg := <-o.control:
		return msg, true
	case msg := <-o.config:
		return msg, true
	case msg := <-o.ready:
		return msg, true
	case msg := <-o.input:
		return msg, true
	case <-heartbeat:
		return netWireMessage{Type: netMessageKeepAlive}, true
	case <-stop:
		return netWireMessage{}, false
	case <-sessionDone:
		return netWireMessage{}, false
	}
}

func (l *netplayLink) Read(maxFrame int, msg *netWireMessage) error {
	if l == nil || l.conn == nil || l.closed.Load() {
		return net.ErrClosed
	}
	return readNetFrame(l.conn, l.codec, maxFrame, msg)
}

func (l *netplayLink) Write(msg netWireMessage, maxFrame int) error {
	if l == nil || l.conn == nil || l.closed.Load() {
		return net.ErrClosed
	}
	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	if l.closed.Load() {
		return net.ErrClosed
	}
	return writeNetFrame(l.conn, l.codec, msg, maxFrame)
}

func (l *netplayLink) SetReadDeadline(deadline time.Time) error {
	if l == nil || l.conn == nil {
		return net.ErrClosed
	}
	return l.conn.SetReadDeadline(deadline)
}

func (l *netplayLink) RemoteAddr() net.Addr {
	if l == nil || l.conn == nil {
		return nil
	}
	return l.conn.RemoteAddr()
}

func (l *netplayLink) Close() error {
	if l == nil || l.conn == nil || l.closed.Swap(true) {
		return nil
	}
	return l.conn.Close()
}
