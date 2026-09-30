package game

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gunmayhem-go/internal/masterapi"

	"golang.org/x/net/websocket"
)

const (
	cloudflareNetplayPath       = "/gm/netplay"
	cloudflareNetplayProtocol   = "gunmayhem-netplay-v1"
	cloudflareInviteCodePrefix  = "GMT1-"
	cloudflareTunnelHostSuffix  = ".trycloudflare.com"
	cloudflareTunnelURLMaxLen   = 160
	cloudflareTunnelStartWindow = 20 * time.Second
)

var quickTunnelURLRE = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

type cloudflareQuickTunnel struct {
	mu sync.RWMutex

	session   *netplaySession
	server    *http.Server
	listener  net.Listener
	cmd       *exec.Cmd
	publicURL string
	wsURL     string
	lastErr   error

	connectionsMu sync.Mutex
	connections   map[*trackedWebSocketConn]struct{}

	ready     chan struct{}
	readyOnce sync.Once
	closeOnce sync.Once
	closed    atomic.Bool
	running   atomic.Bool
}

type trackedWebSocketConn struct {
	net.Conn
	done chan struct{}
	once sync.Once
}

func newTrackedWebSocketConn(conn net.Conn) *trackedWebSocketConn {
	return &trackedWebSocketConn{Conn: conn, done: make(chan struct{})}
}

func (c *trackedWebSocketConn) Close() error {
	if c == nil {
		return nil
	}
	var err error
	c.once.Do(func() {
		if c.Conn != nil {
			// Close on x/net/websocket writes a close frame first. An expired
			// deadline also releases a blocked writer before that close handshake.
			_ = c.Conn.SetDeadline(time.Now())
			err = c.Conn.Close()
		}
		close(c.done)
	})
	return err
}

func containsWebSocketProtocol(raw, wanted string) bool {
	for _, part := range strings.Split(raw, ",") {
		if strings.TrimSpace(part) == wanted {
			return true
		}
	}
	return false
}

func shouldUseCloudflareInternetTunnel(listenAddress string) bool {
	listenAddress = normalizeListenAddress(listenAddress)
	host, port, err := net.SplitHostPort(listenAddress)
	if err != nil {
		return true
	}
	// Unit/integration tests intentionally bind a loopback ephemeral port. Keep
	// that deterministic local-only harness independent from cloudflared while
	// production Internet hosting (":7777") always uses the Quick Tunnel path.
	if port == "0" {
		ip := net.ParseIP(strings.Trim(host, "[]"))
		return ip == nil || !ip.IsLoopback()
	}
	return true
}

func startCloudflareQuickTunnel(session *netplaySession) (*cloudflareQuickTunnel, error) {
	if !experimentalInternetEnabled() || session == nil || (!session.inviteOnly && (!masterapi.ValidHexIdentity(session.publicRoomID, 16) || configuredPublicDirectoryURL() == "")) {
		return nil, errors.New("Cloudflare hosting requires experimental opt-in and a private invite or configured public room")
	}
	if session == nil || session.mode != netplayHost || session.closed.Load() {
		return nil, errors.New("invalid Cloudflare tunnel host session")
	}
	executable, err := findCloudflaredExecutable()
	if err != nil {
		return nil, err
	}

	tunnel, err := startCloudflareGateway(session)
	if err != nil {
		return nil, err
	}

	origin := "http://" + tunnel.listener.Addr().String()
	cmd := exec.Command(executable, "tunnel", "--url", origin, "--no-autoupdate")
	configureCloudflaredProcess(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = tunnel.Close()
		return nil, fmt.Errorf("capture cloudflared output: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = tunnel.Close()
		return nil, fmt.Errorf("capture cloudflared diagnostics: %w", err)
	}
	tunnel.cmd = cmd
	if err := cmd.Start(); err != nil {
		_ = tunnel.Close()
		return nil, fmt.Errorf("start cloudflared: %w", err)
	}
	tunnel.running.Store(true)
	networkDiagnostic("CLOUDFLARE quick tunnel process started origin=%s exe=%s", origin, executable)

	go tunnel.scanOutput(stdout)
	go tunnel.scanOutput(stderr)
	go tunnel.waitProcess()
	return tunnel, nil
}

// startCloudflareGateway is also the integration-test seam: it binds only to
// loopback and never launches cloudflared, creates an account or publishes a URL.
func startCloudflareGateway(session *netplaySession) (*cloudflareQuickTunnel, error) {
	if session == nil || session.mode != netplayHost || session.closed.Load() {
		return nil, errors.New("invalid WebSocket gateway host session")
	}
	if session.publicRoomID != "" && !masterapi.ValidHexIdentity(session.publicRoomID, 16) {
		return nil, errors.New("invalid public room identity")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start local WebSocket gateway: %w", err)
	}

	tunnel := &cloudflareQuickTunnel{
		connections: make(map[*trackedWebSocketConn]struct{}),
		session:     session,
		listener:    listener,
		ready:       make(chan struct{}),
	}
	mux := http.NewServeMux()
	path := cloudflareNetplayPath
	if session.publicRoomID != "" {
		path = masterapi.PublicRoomWebSocketPrefix + session.publicRoomID
	}
	mux.Handle(path, websocket.Server{
		Handshake: func(config *websocket.Config, request *http.Request) error {
			if session.publicAdmissionClosed.Load() || (request != nil && (request.URL.Path != path || request.URL.RawPath != "" || request.URL.RawQuery != "" || request.URL.ForceQuery)) {
				return errors.New("room unavailable")
			}
			if request == nil || !containsWebSocketProtocol(request.Header.Get("Sec-WebSocket-Protocol"), cloudflareNetplayProtocol) {
				return errors.New("Gun Mayhem WebSocket protocol required")
			}
			origin, err := websocket.Origin(config, request)
			if err != nil || origin == nil || origin.Scheme != "https" || origin.Host != "gunmayhem.invalid" || origin.User != nil || (origin.Path != "" && origin.Path != "/") || origin.RawQuery != "" || origin.Fragment != "" {
				return errors.New("Gun Mayhem native client origin required")
			}
			// x/net/websocket uses Config.Origin for RemoteAddr on the server.
			// Leaving it nil panics when netplay records the peer address.
			config.Origin = origin
			config.Protocol = []string{cloudflareNetplayProtocol}
			return nil
		},
		Handler: websocket.Handler(func(conn *websocket.Conn) {
			conn.PayloadType = websocket.BinaryFrame
			tracked := newTrackedWebSocketConn(conn)
			tunnel.connectionsMu.Lock()
			if tunnel.closed.Load() {
				tunnel.connectionsMu.Unlock()
				_ = tracked.Close()
				return
			}
			tunnel.connections[tracked] = struct{}{}
			tunnel.connectionsMu.Unlock()
			defer func() {
				_ = tracked.Close()
				tunnel.connectionsMu.Lock()
				delete(tunnel.connections, tracked)
				tunnel.connectionsMu.Unlock()
			}()
			if !session.attachIncomingConn(tracked) {
				_ = tracked.Close()
				return
			}
			// x/net/websocket closes the connection when this handler returns.
			// Netplay owns the connection after attachIncomingConn, so keep the
			// HTTP handler alive until netplay itself closes that connection.
			<-tracked.done
		}),
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	tunnel.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    8 * 1024,
		IdleTimeout:       30 * time.Second,
	}
	go func() {
		if serveErr := tunnel.server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) && !tunnel.closed.Load() {
			tunnel.setError(fmt.Errorf("local WebSocket gateway stopped: %w", serveErr))
		}
	}()

	return tunnel, nil
}

func (t *cloudflareQuickTunnel) scanOutput(reader io.Reader) {
	if t == nil || reader == nil {
		return
	}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 256*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if match := quickTunnelURLRE.FindString(strings.ToLower(line)); match != "" {
			t.setPublicURL(match)
		}
	}
	if err := scanner.Err(); err != nil && !t.closed.Load() {
		networkDiagnostic("CLOUDFLARE output read error=%v", err)
	}
}

func (t *cloudflareQuickTunnel) waitProcess() {
	if t == nil || t.cmd == nil {
		return
	}
	err := t.cmd.Wait()
	t.running.Store(false)
	if t.session != nil {
		t.session.setPublicTunnelURL("")
	}
	if t.closed.Load() {
		return
	}
	if err == nil {
		err = errors.New("cloudflared stopped")
	} else {
		err = fmt.Errorf("cloudflared stopped: %w", err)
	}
	t.setError(err)
	networkDiagnostic("CLOUDFLARE quick tunnel stopped err=%v", err)
}

func (t *cloudflareQuickTunnel) setPublicURL(raw string) {
	if t == nil || t.closed.Load() {
		return
	}
	wsURL, err := normalizeCloudflareTunnelURL(raw)
	if err != nil {
		return
	}
	if t.session != nil && t.session.publicRoomID != "" {
		u, _ := url.Parse(wsURL)
		wsURL = "wss://" + u.Host + masterapi.PublicRoomWebSocketPrefix + t.session.publicRoomID
	}
	u, _ := url.Parse(wsURL)
	publicURL := "https://" + u.Host

	t.mu.Lock()
	if t.wsURL == "" {
		t.publicURL = publicURL
		t.wsURL = wsURL
		t.lastErr = nil
	}
	ready := t.wsURL != ""
	t.mu.Unlock()
	if !ready {
		return
	}
	if t.session != nil {
		t.session.setPublicTunnelURL(wsURL)
	}
	t.readyOnce.Do(func() { close(t.ready) })
	networkDiagnostic("CLOUDFLARE quick tunnel ready public=%s websocket=%s", publicURL, wsURL)
}

func (t *cloudflareQuickTunnel) setError(err error) {
	if t == nil || err == nil {
		return
	}
	t.mu.Lock()
	t.lastErr = err
	wsReady := t.wsURL != ""
	t.mu.Unlock()
	if !wsReady {
		t.readyOnce.Do(func() { close(t.ready) })
	}
}

func (t *cloudflareQuickTunnel) Status() (ready bool, publicURL, wsURL string, err error) {
	if t == nil {
		return false, "", "", errors.New("Cloudflare tunnel is not running")
	}
	t.mu.RLock()
	publicURL = t.publicURL
	wsURL = t.wsURL
	err = t.lastErr
	t.mu.RUnlock()
	ready = wsURL != "" && t.running.Load() && !t.closed.Load()
	return ready, publicURL, wsURL, err
}

func (t *cloudflareQuickTunnel) WaitReady(timeout time.Duration) (string, error) {
	if t == nil {
		return "", errors.New("Cloudflare tunnel is not running")
	}
	if timeout <= 0 {
		timeout = cloudflareTunnelStartWindow
	}
	if ready, _, wsURL, err := t.Status(); ready {
		return wsURL, nil
	} else if err != nil {
		return "", err
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-t.ready:
		ready, _, wsURL, err := t.Status()
		if ready {
			return wsURL, nil
		}
		if err != nil {
			return "", err
		}
		return "", errors.New("Cloudflare tunnel did not become ready")
	case <-timer.C:
		return "", errors.New("Cloudflare tunnel startup timed out")
	}
}

func (t *cloudflareQuickTunnel) Close() error {
	if t == nil {
		return nil
	}
	var closeErr error
	t.closeOnce.Do(func() {
		t.closed.Store(true)
		if t.session != nil {
			t.session.setPublicTunnelURL("")
		}
		// HTTP shutdown does not own hijacked WebSocket connections. Explicitly
		// close both accepted peers and pending netplay handshakes on this gateway.
		t.connectionsMu.Lock()
		connections := make([]*trackedWebSocketConn, 0, len(t.connections))
		for conn := range t.connections {
			connections = append(connections, conn)
		}
		t.connectionsMu.Unlock()
		for _, conn := range connections {
			_ = conn.Close()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if t.server != nil {
			if err := t.server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				closeErr = err
			}
		}
		if t.cmd != nil && t.cmd.Process != nil && t.running.Load() {
			_ = t.cmd.Process.Kill()
		}
		t.readyOnce.Do(func() { close(t.ready) })
	})
	return closeErr
}

func findCloudflaredExecutable() (string, error) {
	var candidates []string
	if configured := strings.TrimSpace(os.Getenv("GUNMAYHEM_CLOUDFLARED")); configured != "" {
		candidates = append(candidates, configured)
	}
	if executable, err := os.Executable(); err == nil {
		dir := filepath.Dir(executable)
		candidates = append(candidates,
			filepath.Join(dir, "cloudflared.exe"),
			filepath.Join(dir, "cloudflared"),
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "cloudflared.exe"),
			filepath.Join(cwd, "build", "tools", "cloudflared.exe"),
			filepath.Join(cwd, "build", "tools", "cloudflared"),
		)
	}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	if executable, err := exec.LookPath("cloudflared"); err == nil {
		return executable, nil
	}
	return "", errors.New("cloudflared was not found; install the official cloudflared binary and set GUNMAYHEM_CLOUDFLARED to its path")
}

func normalizeCloudflareTunnelURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > cloudflareTunnelURLMaxLen {
		return "", errors.New("invalid Cloudflare tunnel URL")
	}
	if !strings.Contains(raw, "://") {
		raw = "wss://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil {
		return "", errors.New("invalid Cloudflare tunnel URL")
	}
	if u.Scheme != "https" && u.Scheme != "wss" {
		return "", errors.New("Cloudflare tunnel must use HTTPS/WSS")
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "" || !strings.HasSuffix(host, cloudflareTunnelHostSuffix) {
		return "", errors.New("not a trycloudflare.com tunnel")
	}
	subdomain := strings.TrimSuffix(host, cloudflareTunnelHostSuffix)
	if !validCloudflareTunnelSubdomain(subdomain) {
		return "", errors.New("invalid trycloudflare.com tunnel hostname")
	}
	if port := u.Port(); port != "" && port != "443" {
		return "", errors.New("unexpected Cloudflare tunnel port")
	}
	return "wss://" + host + cloudflareNetplayPath, nil
}

func validCloudflareTunnelSubdomain(subdomain string) bool {
	if subdomain == "" || len(subdomain) > 63 || strings.HasPrefix(subdomain, "-") || strings.HasSuffix(subdomain, "-") {
		return false
	}
	for _, r := range subdomain {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}

func encodeCloudflareInviteCode(wsURL, sessionKey string) (string, error) {
	wsURL, err := normalizeCloudflareTunnelURL(wsURL)
	if err != nil {
		return "", err
	}
	if _, err := decodeOnlineSessionKey(sessionKey); err != nil {
		return "", err
	}
	u, _ := url.Parse(wsURL)
	subdomain := strings.TrimSuffix(strings.ToLower(u.Hostname()), cloudflareTunnelHostSuffix)
	return cloudflareInviteCodePrefix + strings.ToUpper(subdomain) + "." + strings.ToUpper(strings.TrimSpace(sessionKey)), nil
}

func decodeCloudflareInviteCode(code string) (wsURL, sessionKey string, err error) {
	code = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
	if !strings.HasPrefix(code, cloudflareInviteCodePrefix) {
		return "", "", errors.New("not a Cloudflare Gun Mayhem invite code")
	}
	payload := strings.TrimPrefix(code, cloudflareInviteCodePrefix)
	sep := strings.LastIndexByte(payload, '.')
	if sep <= 0 || sep >= len(payload)-1 {
		return "", "", errors.New("invalid Cloudflare invite code")
	}
	subdomain := strings.ToLower(payload[:sep])
	sessionKey = payload[sep+1:]
	if !validCloudflareTunnelSubdomain(subdomain) {
		return "", "", errors.New("invalid Cloudflare invite hostname")
	}
	if _, err := decodeOnlineSessionKey(sessionKey); err != nil {
		return "", "", errors.New("invalid Cloudflare invite session key")
	}
	wsURL, err = normalizeCloudflareTunnelURL("wss://" + subdomain + cloudflareTunnelHostSuffix)
	if err != nil {
		return "", "", err
	}
	return wsURL, sessionKey, nil
}

func (g *Game) StartNetClientCloudflareInvite(code string) error {
	if g == nil {
		return errors.New("nil game")
	}
	wsURL, sessionKey, err := decodeCloudflareInviteCode(code)
	if err != nil {
		return err
	}
	_ = g.CloseNetplay()
	s, err := newClientNetplayCloudflare(wsURL, sessionKey, 8*time.Second)
	if err != nil {
		return err
	}
	if err := waitForClientHandshake(s, netHandshakeTimeout+time.Second); err != nil {
		_ = s.Close()
		return err
	}
	g.netplay = s
	g.onlineSessionVisibility = onlineSessionInviteOnly
	g.onlineSessionScope = onlineSessionScopeInternet
	return nil
}

func newClientNetplayCloudflare(rawURL, sessionKey string, timeout time.Duration) (*netplaySession, error) {
	wsURL, err := normalizeCloudflareTunnelURL(rawURL)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	config, err := websocket.NewConfig(wsURL, "https://gunmayhem.invalid/")
	if err != nil {
		return nil, err
	}
	config.Protocol = []string{cloudflareNetplayProtocol}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := config.DialContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect through Cloudflare: %w", err)
	}
	conn.PayloadType = websocket.BinaryFrame
	s, err := newClientNetplayFromConnWithKey(newTrackedWebSocketConn(conn), "Cloudflare "+wsURL, strings.TrimSpace(sessionKey))
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return s, nil
}
