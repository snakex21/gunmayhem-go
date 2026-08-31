package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"

	"gunmayhem-go/internal/masterapi"
)

const (
	maxActiveMasterServers       = 4096
	maxMasterServersPerIP        = 16
	masterRateWindow             = time.Minute
	maxHeartbeatRequestsPerIP    = 240
	maxServerListRequestsPerIP   = 120
	maxServerDeleteRequestsPerIP = 60
	maxTrackedMasterRateKeys     = 16384
)

type registryEntry struct {
	server   masterapi.Server
	seen     time.Time
	ip       string
	secret   string
	private  bool
	joinCode string
}

type masterRateEntry struct {
	windowStart time.Time
	count       int
}

type serverRegistry struct {
	mu                 sync.Mutex
	entries            map[string]registryEntry
	ttl                time.Duration
	trustProxy         bool
	publicRelayAddress string
	relayPort          int
	relay              *relayHub
	rate               map[string]masterRateEntry
}

func newServerRegistry(ttl time.Duration) *serverRegistry {
	if ttl <= 0 {
		ttl = 35 * time.Second
	}
	return &serverRegistry{
		entries: make(map[string]registryEntry),
		rate:    make(map[string]masterRateEntry),
		ttl:     ttl,
	}
}

func randomServerID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func randomServerSecret() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("fallback-secret-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func sameServerSecret(a, b string) bool {
	if a == "" || b == "" || len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func sanitizeServerName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Gun Mayhem Host"
	}
	runes := []rune(name)
	if len(runes) > 48 {
		runes = runes[:48]
	}
	return string(runes)
}

func requestIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		for _, candidate := range []string{
			strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]),
			strings.TrimSpace(r.Header.Get("X-Real-IP")),
		} {
			if ip := net.ParseIP(candidate); ip != nil {
				return ip.String()
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func (r *serverRegistry) allowRequest(kind, ip string, limit int) bool {
	if r == nil || kind == "" || ip == "" || limit <= 0 {
		return false
	}
	now := time.Now()
	key := kind + "|" + ip
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.rate) > maxTrackedMasterRateKeys {
		for k, entry := range r.rate {
			if now.Sub(entry.windowStart) >= masterRateWindow {
				delete(r.rate, k)
			}
		}
	}
	entry := r.rate[key]
	if entry.windowStart.IsZero() || now.Sub(entry.windowStart) >= masterRateWindow {
		entry = masterRateEntry{windowStart: now}
	}
	if entry.count >= limit {
		r.rate[key] = entry
		return false
	}
	entry.count++
	r.rate[key] = entry
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (r *serverRegistry) relayAddressForRequest(_ *http.Request) string {
	if r == nil || r.relay == nil {
		return ""
	}
	// Never derive this from the request Host header: a public registrant can
	// control that header and must not be able to make other players connect to
	// an arbitrary relay endpoint. Public relay advertisement is explicit.
	return strings.TrimSpace(r.publicRelayAddress)
}

func (r *serverRegistry) heartbeat(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ip := requestIP(req, r.trustProxy)
	if net.ParseIP(ip) == nil {
		http.Error(w, "cannot determine public address", http.StatusBadRequest)
		return
	}
	if !r.allowRequest("heartbeat", ip, maxHeartbeatRequestsPerIP) {
		http.Error(w, "heartbeat rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	defer req.Body.Close()
	dec := json.NewDecoder(http.MaxBytesReader(w, req.Body, 8<<10))
	dec.DisallowUnknownFields()
	var hb masterapi.HeartbeatRequest
	if err := dec.Decode(&hb); err != nil {
		http.Error(w, "invalid heartbeat", http.StatusBadRequest)
		return
	}
	if hb.Port <= 0 || hb.Port > 65535 || hb.Protocol <= 0 || hb.MaxPlayers <= 0 || hb.MaxPlayers > 16 || hb.Players < 0 || hb.Players > hb.MaxPlayers ||
		hb.GameMode < 0 || hb.GameMode > 16 || hb.Lives < 0 || hb.Lives > 99999 || hb.Rounds < 0 || hb.Rounds > len(hb.Maps) ||
		hb.ChaosEvents < 0 || hb.ChaosEvents > 256 || hb.ChaosIntervalSeconds < 0 || hb.ChaosIntervalSeconds > 3600 {
		http.Error(w, "invalid server data", http.StatusBadRequest)
		return
	}
	for _, mapValue := range hb.Maps {
		if mapValue < -100000 || mapValue > 100000 {
			http.Error(w, "invalid map preview", http.StatusBadRequest)
			return
		}
	}
	hb.RuleSummary = strings.TrimSpace(hb.RuleSummary)
	if len(hb.RuleSummary) > 96 {
		http.Error(w, "rule summary too long", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(hb.ID)
	secret := strings.TrimSpace(hb.Secret)
	sessionID := strings.TrimSpace(hb.SessionID)
	joinCode := strings.ToUpper(strings.TrimSpace(hb.JoinCode))
	if len(id) > 96 || len(secret) > 128 || len(sessionID) > 96 || len(joinCode) > 16 {
		http.Error(w, "invalid server identity", http.StatusBadRequest)
		return
	}
	if hb.Private {
		if len(joinCode) != 6 {
			http.Error(w, "private server requires six-character join code", http.StatusBadRequest)
			return
		}
		for _, r := range joinCode {
			if !(r >= 'A' && r <= 'Z') && !(r >= '2' && r <= '7') {
				http.Error(w, "invalid private join code", http.StatusBadRequest)
				return
			}
		}
	} else {
		joinCode = ""
	}

	now := time.Now()
	r.mu.Lock()
	if id != "" {
		if old, ok := r.entries[id]; ok && now.Sub(old.seen) <= r.ttl {
			if !sameServerSecret(old.secret, secret) {
				r.mu.Unlock()
				http.Error(w, "invalid server secret", http.StatusUnauthorized)
				return
			}
			secret = old.secret
		} else {
			// A stale ID after a master restart gets a fresh cryptographic identity.
			id = ""
			secret = ""
		}
	}
	if id == "" {
		activeForIP := 0
		for existingID, entry := range r.entries {
			if now.Sub(entry.seen) > r.ttl {
				delete(r.entries, existingID)
				if r.relay != nil {
					r.relay.dropHost(existingID)
				}
				continue
			}
			if entry.ip == ip {
				activeForIP++
			}
		}
		if activeForIP >= maxMasterServersPerIP || len(r.entries) >= maxActiveMasterServers {
			r.mu.Unlock()
			http.Error(w, "server registration limit reached", http.StatusTooManyRequests)
			return
		}
		id = randomServerID()
		secret = randomServerSecret()
	}
	server := masterapi.Server{
		ID:                   id,
		SessionID:            sessionID,
		Name:                 sanitizeServerName(hb.Name),
		Address:              net.JoinHostPort(ip, strconv.Itoa(hb.Port)),
		RelayAddress:         r.relayAddressForRequest(req),
		Players:              hb.Players,
		MaxPlayers:           hb.MaxPlayers,
		Protocol:             hb.Protocol,
		GameMode:             hb.GameMode,
		Lives:                hb.Lives,
		Rounds:               hb.Rounds,
		Maps:                 hb.Maps,
		PlaylistConfigured:   hb.PlaylistConfigured,
		Crates:               hb.Crates,
		Pickups:              hb.Pickups,
		RuleSummary:          hb.RuleSummary,
		ChaosEnabled:         hb.ChaosEnabled,
		ChaosEvents:          hb.ChaosEvents,
		ChaosIntervalSeconds: hb.ChaosIntervalSeconds,
	}
	r.entries[id] = registryEntry{server: server, seen: now, ip: ip, secret: secret, private: hb.Private, joinCode: joinCode}
	r.mu.Unlock()

	writeJSON(w, http.StatusOK, masterapi.HeartbeatResponse{ID: id, Secret: secret, Address: server.Address, RelayAddress: server.RelayAddress})
}

func (r *serverRegistry) relayHostSecret(id string, protocol int) (string, bool) {
	if r == nil || id == "" || protocol <= 0 {
		return "", false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.entries[id]
	if !ok || time.Since(entry.seen) > r.ttl || entry.server.Protocol != protocol || entry.secret == "" {
		return "", false
	}
	return entry.secret, true
}

func (r *serverRegistry) relayClientAllowed(id string, protocol int) bool {
	if r == nil || id == "" || protocol <= 0 {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.entries[id]
	return ok && time.Since(entry.seen) <= r.ttl && entry.server.Protocol == protocol
}

func (r *serverRegistry) list(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ip := requestIP(req, r.trustProxy)
	if net.ParseIP(ip) == nil || !r.allowRequest("list", ip, maxServerListRequestsPerIP) {
		http.Error(w, "server list rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	protocol := 0
	if raw := strings.TrimSpace(req.URL.Query().Get("protocol")); raw != "" {
		protocol, _ = strconv.Atoi(raw)
	}
	joinCode := strings.ToUpper(strings.TrimSpace(req.URL.Query().Get("code")))
	if joinCode != "" {
		if len(joinCode) != 6 {
			http.Error(w, "invalid invite code", http.StatusBadRequest)
			return
		}
		for _, r := range joinCode {
			if !(r >= 'A' && r <= 'Z') && !(r >= '2' && r <= '7') {
				http.Error(w, "invalid invite code", http.StatusBadRequest)
				return
			}
		}
	}
	now := time.Now()
	servers := make([]masterapi.Server, 0)
	r.mu.Lock()
	for id, entry := range r.entries {
		if now.Sub(entry.seen) > r.ttl {
			delete(r.entries, id)
			if r.relay != nil {
				r.relay.dropHost(id)
			}
			continue
		}
		if protocol > 0 && entry.server.Protocol != protocol {
			continue
		}
		if joinCode != "" {
			if !entry.private || entry.joinCode != joinCode {
				continue
			}
		} else if entry.private {
			continue
		}
		servers = append(servers, entry.server)
	}
	r.mu.Unlock()
	sort.Slice(servers, func(i, j int) bool {
		fullI := servers[i].MaxPlayers > 0 && servers[i].Players >= servers[i].MaxPlayers
		fullJ := servers[j].MaxPlayers > 0 && servers[j].Players >= servers[j].MaxPlayers
		if fullI != fullJ {
			return !fullI
		}
		if servers[i].Name != servers[j].Name {
			return servers[i].Name < servers[j].Name
		}
		return servers[i].Address < servers[j].Address
	})
	writeJSON(w, http.StatusOK, masterapi.ServerListResponse{Servers: servers})
}

func (r *serverRegistry) remove(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ip := requestIP(req, r.trustProxy)
	if net.ParseIP(ip) == nil || !r.allowRequest("delete", ip, maxServerDeleteRequestsPerIP) {
		http.Error(w, "server delete rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	id := strings.TrimSpace(req.URL.Query().Get("id"))
	secret := strings.TrimSpace(req.Header.Get(masterapi.ServerSecretHeader))
	if id == "" || secret == "" {
		http.Error(w, "missing server identity", http.StatusBadRequest)
		return
	}
	removed := false
	r.mu.Lock()
	if entry, ok := r.entries[id]; ok && sameServerSecret(entry.secret, secret) {
		delete(r.entries, id)
		removed = true
	}
	r.mu.Unlock()
	if !removed {
		http.Error(w, "invalid server secret", http.StatusUnauthorized)
		return
	}
	if r.relay != nil {
		r.relay.dropHost(id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func newHandler(reg *serverRegistry) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(masterapi.HeartbeatPath, reg.heartbeat)
	mux.HandleFunc(masterapi.ServersPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			reg.remove(w, r)
			return
		}
		reg.list(w, r)
	})
	mux.HandleFunc(masterapi.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if reg != nil && reg.relay != nil {
		mux.Handle(masterapi.RelayWebSocketPath, websocket.Server{
			Handshake: func(*websocket.Config, *http.Request) error { return nil },
			Handler: websocket.Handler(func(conn *websocket.Conn) {
				conn.PayloadType = websocket.BinaryFrame
				reg.relay.handleWebSocket(conn)
			}),
		})
	}
	return mux
}

func main() {
	listen := flag.String("addr", ":7780", "HTTP listen address")
	relayListen := flag.String("relay-addr", ":7781", "TCP relay listen address; empty disables relay")
	publicRelayAddress := flag.String("public-relay-address", "", "public relay host:port advertised to games; defaults to HTTP Host with relay port")
	ttl := flag.Duration("ttl", 35*time.Second, "server expiry without heartbeat")
	trustProxy := flag.Bool("trust-proxy", false, "trust X-Forwarded-For/X-Real-IP from a reverse proxy")
	tlsCert := flag.String("tls-cert", "", "TLS certificate for serving HTTPS/WSS directly")
	tlsKey := flag.String("tls-key", "", "TLS private key for serving HTTPS/WSS directly")
	flag.Parse()

	reg := newServerRegistry(*ttl)
	reg.trustProxy = *trustProxy
	reg.publicRelayAddress = strings.TrimSpace(*publicRelayAddress)
	if strings.TrimSpace(*relayListen) != "" || reg.publicRelayAddress != "" {
		hub := newRelayHub(reg)
		reg.relay = hub
	}
	if strings.TrimSpace(*relayListen) != "" {
		_, portText, err := net.SplitHostPort(*relayListen)
		if err != nil {
			log.Fatalf("invalid relay listen address %q: %v", *relayListen, err)
		}
		port, err := strconv.Atoi(portText)
		if err != nil || port <= 0 || port > 65535 {
			log.Fatalf("invalid relay port %q", portText)
		}
		reg.relayPort = port
		hub := reg.relay
		go func() {
			log.Printf("Gun Mayhem relay listening on %s", *relayListen)
			if err := hub.Serve(*relayListen); err != nil {
				log.Fatalf("relay server failed: %v", err)
			}
		}()
	}
	server := &http.Server{
		Addr:              *listen,
		Handler:           newHandler(reg),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       8 * time.Second,
		WriteTimeout:      8 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	log.Printf("Gun Mayhem master server listening on %s (ttl=%s)", *listen, ttl.String())
	if (*tlsCert == "") != (*tlsKey == "") {
		log.Fatal("both -tls-cert and -tls-key are required")
	}
	if *tlsCert != "" {
		log.Fatal(server.ListenAndServeTLS(*tlsCert, *tlsKey))
	}
	log.Fatal(server.ListenAndServe())
}
