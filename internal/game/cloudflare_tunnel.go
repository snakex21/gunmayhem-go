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

	"golang.org/x/net/websocket"
)

const (
	cloudflareNetplayPath       = "/gm/netplay"
	cloudflareNetplayProtocol   = "gunmayhem-netplay-v1"
	cloudflareInviteCodePrefix  = "GMT1-"
	cloudflareTunnelHostSuffix  = ".trycloudflare.com"
	cloudflareTunnelURLMaxLen   = 160
	cloudflareTunnelStartWindow = 30 * time.Second

	// cloudflareTunnelMaxAttempts is how many cloudflared processes the host
	// starts before it reports a final tunnel failure.
	cloudflareTunnelMaxAttempts = 3
	// Public /healthz probes back off from min to max while the edge still
	// answers 530/1033. The old loop re-selected a closed channel and fired
	// about 35 requests per second, which risks Quick Tunnel rate limiting.
	cloudflareHealthBackoffMin = 500 * time.Millisecond
	cloudflareHealthBackoffMax = 2 * time.Second
	// While cloudflared has not printed its URL yet we only poll local state.
	cloudflareURLPollInterval = 200 * time.Millisecond
)

var quickTunnelURLRE = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

type cloudflareQuickTunnel struct {
	mu sync.RWMutex

	session    *netplaySession
	server     *http.Server
	listener   net.Listener
	cmd        *exec.Cmd
	executable string
	origin     string
	publicURL  string
	wsURL      string
	// attemptErr is the failure of the current cloudflared attempt. It makes
	// WaitReady give up on this attempt, but it is not exposed through Status
	// while monitorStartup may still retry.
	attemptErr error
	// lastErr is the final, user-visible tunnel error.
	lastErr error
	// lastCloudflaredErr is the most recent ERR line printed by cloudflared.
	lastCloudflaredErr string
	attempt            int
	maxAttempts        int
	generation         int

	// Health probe backoff; zero values use the package defaults. Tests use
	// shorter values.
	healthBackoffMin time.Duration
	healthBackoffMax time.Duration

	publicReady atomic.Bool
	closeOnce   sync.Once
	closed      atomic.Bool
	running     atomic.Bool
}

type trackedWebSocketConn struct {
	net.Conn
	done chan struct{}
	once sync.Once
}

type cloudflareWebSocketAddr struct{}

func (cloudflareWebSocketAddr) Network() string { return "cloudflare-wss" }
func (cloudflareWebSocketAddr) String() string  { return "cloudflare-wss" }

func newTrackedWebSocketConn(conn net.Conn) *trackedWebSocketConn {
	return &trackedWebSocketConn{Conn: conn, done: make(chan struct{})}
}

// RemoteAddr returns the Cloudflare relay instead of nil for x/net/websocket
// connections. The game uses this value as a diagnostic peer label; it is not
// the player's real IP address.
func (c *trackedWebSocketConn) RemoteAddr() net.Addr {
	return cloudflareWebSocketAddr{}
}

func (c *trackedWebSocketConn) Close() error {
	if c == nil {
		return nil
	}
	var err error
	c.once.Do(func() {
		if c.Conn != nil {
			err = c.Conn.Close()
		}
		close(c.done)
	})
	return err
}

func containsWebSocketProtocol(raw, wanted string) bool {
	for _, part := range strings.Split(raw, ",") {
		if strings.EqualFold(strings.TrimSpace(part), wanted) {
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
	if session == nil || session.mode != netplayHost || session.closed.Load() {
		return nil, errors.New("invalid Cloudflare tunnel host session")
	}
	executable, err := findCloudflaredExecutable()
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start local WebSocket gateway: %w", err)
	}

	tunnel := &cloudflareQuickTunnel{
		session:     session,
		listener:    listener,
		executable:  executable,
		origin:      "http://" + listener.Addr().String(),
		maxAttempts: cloudflareTunnelMaxAttempts,
	}
	tunnel.server = &http.Server{
		Handler:           newCloudflareGatewayMux(session),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	go func() {
		if serveErr := tunnel.server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) && !tunnel.closed.Load() {
			tunnel.setError(fmt.Errorf("local WebSocket gateway stopped: %w", serveErr))
		}
	}()

	if err := tunnel.startProcess(1); err != nil {
		_ = tunnel.Close()
		return nil, err
	}
	return tunnel, nil
}

// newCloudflareGatewayMux serves the local origin that cloudflared forwards
// public WSS traffic to: the netplay WebSocket and a /healthz probe target.
func newCloudflareGatewayMux(session *netplaySession) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(cloudflareNetplayPath, websocket.Server{
		Handshake: func(config *websocket.Config, request *http.Request) error {
			if request == nil || !containsWebSocketProtocol(request.Header.Get("Sec-WebSocket-Protocol"), cloudflareNetplayProtocol) {
				return errors.New("Gun Mayhem WebSocket protocol required")
			}
			config.Protocol = []string{cloudflareNetplayProtocol}
			return nil
		},
		Handler: websocket.Handler(func(conn *websocket.Conn) {
			conn.PayloadType = websocket.BinaryFrame
			tracked := newTrackedWebSocketConn(conn)
			if session == nil || !session.attachIncomingConn(tracked) {
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
	return mux
}

// startProcess launches one cloudflared Quick Tunnel attempt against the
// already running local gateway. Each attempt gets a new generation so output
// and exit notifications from a killed earlier process are ignored.
func (t *cloudflareQuickTunnel) startProcess(attempt int) error {
	if t == nil || t.executable == "" || t.origin == "" {
		return errors.New("Cloudflare tunnel is not configured")
	}
	cmd := exec.Command(t.executable, "tunnel", "--url", t.origin, "--no-autoupdate")
	configureCloudflaredProcess(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("capture cloudflared output: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("capture cloudflared diagnostics: %w", err)
	}

	t.mu.Lock()
	if t.closed.Load() {
		t.mu.Unlock()
		return errors.New("Cloudflare tunnel closed")
	}
	t.generation++
	gen := t.generation
	t.attempt = attempt
	t.publicURL = ""
	t.wsURL = ""
	t.attemptErr = nil
	t.lastCloudflaredErr = ""
	t.mu.Unlock()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start cloudflared (%s): %w", t.executable, err)
	}
	t.mu.Lock()
	t.cmd = cmd
	t.mu.Unlock()
	t.running.Store(true)
	if t.closed.Load() {
		// Close raced with Start; do not leak the process.
		_ = cmd.Process.Kill()
	}
	networkDiagnostic("CLOUDFLARE quick tunnel process started attempt=%d/%d origin=%s exe=%s", attempt, t.attemptLimit(), t.origin, t.executable)

	go t.scanOutput(stdout, gen)
	go t.scanOutput(stderr, gen)
	go t.waitProcess(cmd, gen)
	return nil
}

// restartProcess kills the current cloudflared process (if any) and starts a
// fresh Quick Tunnel. The new tunnel receives a new trycloudflare hostname.
func (t *cloudflareQuickTunnel) restartProcess(attempt int) error {
	if t == nil {
		return errors.New("Cloudflare tunnel is not running")
	}
	t.mu.Lock()
	old := t.cmd
	t.generation++ // orphan output/exit goroutines of the old process
	t.mu.Unlock()
	t.running.Store(false)
	t.publicReady.Store(false)
	if old != nil && old.Process != nil {
		_ = old.Process.Kill()
	}
	return t.startProcess(attempt)
}

func (t *cloudflareQuickTunnel) attemptLimit() int {
	if t == nil || t.maxAttempts <= 0 {
		return 1
	}
	return t.maxAttempts
}

func (t *cloudflareQuickTunnel) isCurrentGeneration(gen int) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return gen == t.generation
}

// monitorStartup performs the potentially slow public /healthz wait outside the
// game-thread call that creates a session. Status continues to expose STARTING
// until the Cloudflare tunnel is ready. A failed attempt restarts cloudflared up
// to maxAttempts times before a final error becomes visible.
func (t *cloudflareQuickTunnel) monitorStartup() {
	if t == nil {
		return
	}
	go func() {
		attempt, maxAttempts := t.StartupAttempt()
		if attempt < 1 {
			attempt = 1
		}
		for {
			_, err := t.WaitReady(cloudflareTunnelStartWindow)
			if t.closed.Load() {
				return
			}
			if err == nil {
				if ready, publicURL, _, _ := t.Status(); ready {
					networkDiagnostic("CLOUDFLARE quick tunnel startup ready attempt=%d/%d public=%s", attempt, maxAttempts, publicURL)
				}
				return
			}
			err = t.withCloudflaredReason(err)
			networkDiagnostic("CLOUDFLARE quick tunnel attempt %d/%d failed: %v", attempt, maxAttempts, err)
			if _, _, _, finalErr := t.Status(); finalErr != nil {
				// A final error (for example the local gateway died) cannot be
				// fixed by restarting cloudflared.
				return
			}
			if attempt >= maxAttempts || t.executable == "" {
				if maxAttempts > 1 {
					err = fmt.Errorf("Internet tunnel failed after %d attempts: %w", attempt, err)
				} else {
					err = fmt.Errorf("Internet tunnel startup: %w", err)
				}
				t.setError(err)
				networkDiagnostic("CLOUDFLARE quick tunnel startup failed: %v", err)
				return
			}
			attempt++
			time.Sleep(time.Duration(attempt-1) * time.Second)
			if t.closed.Load() {
				return
			}
			if restartErr := t.restartProcess(attempt); restartErr != nil {
				err = fmt.Errorf("Internet tunnel restart failed: %w (previous: %v)", restartErr, err)
				t.setError(err)
				networkDiagnostic("CLOUDFLARE quick tunnel startup failed: %v", err)
				return
			}
		}
	}()
}

func (t *cloudflareQuickTunnel) scanOutput(reader io.Reader, gen int) {
	if t == nil || reader == nil {
		return
	}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 256*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !t.isCurrentGeneration(gen) {
			continue // keep draining the pipe of an orphaned process
		}
		if match := quickTunnelURLRE.FindString(strings.ToLower(line)); match != "" {
			t.setPublicURL(match, gen)
		}
		if isCloudflaredErrorLine(line) {
			reason := strings.TrimSpace(line)
			if len(reason) > 300 {
				reason = reason[:300]
			}
			t.mu.Lock()
			if gen == t.generation {
				t.lastCloudflaredErr = reason
			}
			t.mu.Unlock()
			networkDiagnostic("CLOUDFLARE cloudflared: %s", reason)
		}
	}
	if err := scanner.Err(); err != nil && !t.closed.Load() {
		networkDiagnostic("CLOUDFLARE output read error=%v", err)
	}
}

// isCloudflaredErrorLine recognises cloudflared's zerolog console levels.
func isCloudflaredErrorLine(line string) bool {
	return strings.Contains(line, " ERR ") || strings.Contains(line, " FTL ") || strings.HasPrefix(line, "ERR ") || strings.HasPrefix(line, "FTL ")
}

func (t *cloudflareQuickTunnel) waitProcess(cmd *exec.Cmd, gen int) {
	if t == nil || cmd == nil {
		return
	}
	err := cmd.Wait()
	if !t.isCurrentGeneration(gen) {
		return
	}
	t.running.Store(false)
	wasReady := t.publicReady.Swap(false)
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
	err = t.withCloudflaredReason(err)
	if wasReady {
		// The tunnel was live; invite codes containing its hostname are now
		// invalid, so this is a final, visible error.
		t.setError(err)
	} else {
		t.setAttemptError(err)
	}
	networkDiagnostic("CLOUDFLARE quick tunnel stopped err=%v", err)
}

func (t *cloudflareQuickTunnel) withCloudflaredReason(err error) error {
	if t == nil || err == nil {
		return err
	}
	t.mu.RLock()
	reason := t.lastCloudflaredErr
	t.mu.RUnlock()
	if reason == "" || strings.Contains(err.Error(), reason) {
		return err
	}
	return fmt.Errorf("%w (cloudflared: %s)", err, reason)
}

func (t *cloudflareQuickTunnel) setPublicURL(raw string, gen int) {
	if t == nil || t.closed.Load() {
		return
	}
	wsURL, err := normalizeCloudflareTunnelURL(raw)
	if err != nil {
		return
	}
	u, _ := url.Parse(wsURL)
	publicURL := "https://" + u.Host

	t.mu.Lock()
	if gen != t.generation {
		t.mu.Unlock()
		return
	}
	changed := false
	if t.wsURL == "" {
		t.publicURL = publicURL
		t.wsURL = wsURL
		changed = true
	}
	t.mu.Unlock()
	// cloudflared prints the Quick Tunnel hostname before its public DNS is
	// necessarily resolvable. WaitReady starts health probes now, but the URL
	// is not exposed to LAN/DHT clients until the public health endpoint works.
	if changed {
		networkDiagnostic("CLOUDFLARE quick tunnel advertised public=%s websocket=%s", publicURL, wsURL)
	}
}

// setError records a final, user-visible tunnel error.
func (t *cloudflareQuickTunnel) setError(err error) {
	if t == nil || err == nil {
		return
	}
	t.mu.Lock()
	t.lastErr = err
	t.mu.Unlock()
}

// setAttemptError fails only the current startup attempt; monitorStartup may
// still restart cloudflared.
func (t *cloudflareQuickTunnel) setAttemptError(err error) {
	if t == nil || err == nil {
		return
	}
	t.mu.Lock()
	t.attemptErr = err
	t.mu.Unlock()
}

func (t *cloudflareQuickTunnel) markPublicReady() (string, error) {
	if t == nil {
		return "", errors.New("Cloudflare tunnel is not running")
	}
	t.mu.RLock()
	wsURL := t.wsURL
	t.mu.RUnlock()
	if wsURL == "" || !t.running.Load() || t.closed.Load() {
		return "", errors.New("Cloudflare tunnel is not ready")
	}
	if t.publicReady.CompareAndSwap(false, true) && t.session != nil {
		t.session.setPublicTunnelURL(wsURL)
	}
	t.mu.RLock()
	publicURL := t.publicURL
	t.mu.RUnlock()
	networkDiagnostic("CLOUDFLARE quick tunnel ready public=%s websocket=%s", publicURL, wsURL)
	return wsURL, nil
}

func probeCloudflareTunnelHealth(ctx context.Context, publicURL string) error {
	publicURL = strings.TrimRight(strings.TrimSpace(publicURL), "/")
	if publicURL == "" {
		return errors.New("Cloudflare public URL is empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, publicURL+"/healthz", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cache-Control", "no-cache")
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Cloudflare health returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// Status reports readiness and the final tunnel error. Errors of a startup
// attempt that will be retried are not reported here.
func (t *cloudflareQuickTunnel) Status() (ready bool, publicURL, wsURL string, err error) {
	if t == nil {
		return false, "", "", errors.New("Cloudflare tunnel is not running")
	}
	t.mu.RLock()
	publicURL = t.publicURL
	wsURL = t.wsURL
	err = t.lastErr
	t.mu.RUnlock()
	ready = t.publicReady.Load() && wsURL != "" && t.running.Load() && !t.closed.Load()
	return ready, publicURL, wsURL, err
}

// StartupAttempt returns the current cloudflared attempt and the attempt limit.
func (t *cloudflareQuickTunnel) StartupAttempt() (attempt, maxAttempts int) {
	if t == nil {
		return 0, 0
	}
	t.mu.RLock()
	attempt = t.attempt
	t.mu.RUnlock()
	return attempt, t.attemptLimit()
}

// StartupStatusText is a short upper-case progress label for the host UI.
func (t *cloudflareQuickTunnel) StartupStatusText() string {
	if t == nil {
		return "STARTING TUNNEL..."
	}
	ready, publicURL, _, err := t.Status()
	switch {
	case ready:
		return "TUNNEL READY"
	case err != nil:
		return "TUNNEL ERROR"
	}
	text := "STARTING CLOUDFLARED"
	if publicURL != "" {
		text = "WAITING FOR CLOUDFLARE EDGE"
	}
	if attempt, maxAttempts := t.StartupAttempt(); attempt > 1 {
		text += fmt.Sprintf(" (RETRY %d/%d)", attempt, maxAttempts)
	}
	return text + "..."
}

func (t *cloudflareQuickTunnel) currentAttemptError() error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.attemptErr
}

func (t *cloudflareQuickTunnel) healthBackoffBounds() (time.Duration, time.Duration) {
	minDelay, maxDelay := t.healthBackoffMin, t.healthBackoffMax
	if minDelay <= 0 {
		minDelay = cloudflareHealthBackoffMin
	}
	if maxDelay < minDelay {
		maxDelay = cloudflareHealthBackoffMax
		if maxDelay < minDelay {
			maxDelay = minDelay
		}
	}
	return minDelay, maxDelay
}

// WaitReady waits until cloudflared has printed its public URL and that URL
// answers /healthz through the Cloudflare edge. Health probes back off
// exponentially between healthBackoffMin and healthBackoffMax.
func (t *cloudflareQuickTunnel) WaitReady(timeout time.Duration) (string, error) {
	if t == nil {
		return "", errors.New("Cloudflare tunnel is not running")
	}
	if timeout <= 0 {
		timeout = cloudflareTunnelStartWindow
	}
	deadline := time.Now().Add(timeout)
	backoff, maxBackoff := t.healthBackoffBounds()
	var lastHealthErr error
	sawURL := false
	timedOut := func() error {
		if lastHealthErr != nil {
			return fmt.Errorf("Cloudflare tunnel did not become reachable within %v (last health check: %w)", timeout, lastHealthErr)
		}
		if !sawURL {
			return fmt.Errorf("cloudflared did not report a public trycloudflare.com URL within %v", timeout)
		}
		return fmt.Errorf("Cloudflare tunnel startup timed out after %v", timeout)
	}
	for {
		if t.closed.Load() {
			return "", errors.New("Cloudflare tunnel closed")
		}
		ready, publicURL, wsURL, err := t.Status()
		if ready {
			return wsURL, nil
		}
		if err != nil {
			return "", err
		}
		if err := t.currentAttemptError(); err != nil {
			return "", err
		}
		wait := cloudflareURLPollInterval
		if publicURL != "" {
			sawURL = true
			probeTimeout := time.Until(deadline)
			if probeTimeout <= 0 {
				return "", timedOut()
			}
			if probeTimeout > 2*time.Second {
				probeTimeout = 2 * time.Second
			}
			probeCtx, cancel := context.WithTimeout(context.Background(), probeTimeout)
			healthErr := probeCloudflareTunnelHealth(probeCtx, publicURL)
			cancel()
			if healthErr == nil {
				return t.markPublicReady()
			}
			lastHealthErr = healthErr
			networkDiagnostic("CLOUDFLARE quick tunnel health pending public=%s err=%v retry_in=%v", publicURL, healthErr, backoff)
			wait = backoff
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return "", timedOut()
		}
		if wait > remaining {
			wait = remaining
		}
		time.Sleep(wait)
	}
}

func (t *cloudflareQuickTunnel) Close() error {
	if t == nil {
		return nil
	}
	var closeErr error
	t.closeOnce.Do(func() {
		t.closed.Store(true)
		t.publicReady.Store(false)
		if t.session != nil {
			t.session.setPublicTunnelURL("")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if t.server != nil {
			if err := t.server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				closeErr = err
			}
		}
		t.mu.RLock()
		cmd := t.cmd
		t.mu.RUnlock()
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
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
	return "", errors.New("cloudflared was not found; rebuild the portable package so cloudflared.exe is included")
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
	s, err := newClientNetplayFromConnWithKey(conn, "Cloudflare "+wsURL, strings.TrimSpace(sessionKey))
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return s, nil
}
