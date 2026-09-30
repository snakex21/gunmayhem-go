// Package publicrooms implements an in-memory, directory-only public room
// registry. It never dials room endpoints, relays gameplay or stores invites.
package publicrooms

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"gunmayhem-go/internal/masterapi"
)

const (
	DefaultTTL = 35 * time.Second
	// Tombstones outlive the client's bounded HTTP requests. They prevent a
	// canceled or reordered heartbeat from recreating a recently closed room.
	TombstoneRetention = 2 * time.Minute
	maxEntries         = 1024 // includes deletion/expiry tombstones
	maxEntriesPerIP    = 16
	maxRateKeys        = 4096
	rateWindow         = time.Minute
)

type entry struct {
	room      masterapi.PublicRoom
	ownerHash [32]byte
	ip        string
	seen      time.Time
	removed   bool
}
type rateEntry struct {
	start time.Time
	count int
}
type Directory struct {
	mu         sync.Mutex
	entries    map[string]entry
	rates      map[string]rateEntry
	ttl        time.Duration
	trustProxy bool
	now        func() time.Time
}

// New returns a bounded, memory-only directory. Trust forwarded IP headers only
// when the listener is reachable exclusively through a trusted reverse proxy
// that overwrites these headers. No network listener is created here.
func New(ttl time.Duration, trustProxy bool) *Directory {
	if ttl <= 0 || ttl > 2*time.Minute {
		ttl = DefaultTTL
	}
	return &Directory{entries: make(map[string]entry), rates: make(map[string]rateEntry), ttl: ttl, trustProxy: trustProxy, now: time.Now}
}

func (d *Directory) Handler() http.Handler {
	// Exact routing avoids ServeMux's implicit escaped-path cleaning/redirects.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.URL.RawPath != "" {
			http.NotFound(w, r)
			return
		}
		switch r.URL.Path {
		case masterapi.PublicRoomsPath:
			d.rooms(w, r)
		case masterapi.HealthPath:
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", "GET")
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if r.URL.RawQuery != "" || r.URL.ForceQuery {
				http.Error(w, "invalid query", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}) // no v1/private lookup, WebSocket or legacy relay routes
}

func requestIP(r *http.Request, trust bool) string {
	if trust {
		for _, raw := range []string{strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0], r.Header.Get("X-Real-IP")} {
			if ip := net.ParseIP(strings.TrimSpace(raw)); ip != nil {
				return ip.String()
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return ""
	}
	return ip.String()
}

func (d *Directory) allow(kind, ip string, now time.Time) bool {
	limit := 120
	if kind == http.MethodDelete {
		limit = 60
	}
	key := kind + "|" + ip
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.rates) >= maxRateKeys {
		for k, v := range d.rates {
			if now.Sub(v.start) >= rateWindow {
				delete(d.rates, k)
			}
		}
	}
	v, known := d.rates[key]
	if !known && len(d.rates) >= maxRateKeys {
		return false
	}
	if v.start.IsZero() || now.Sub(v.start) >= rateWindow {
		v = rateEntry{start: now}
	}
	if v.count >= limit {
		return false
	}
	v.count++
	d.rates[key] = v
	return true
}

// cleanup and quota require d.mu. Expiry preserves the original expiration
// time, so sparse requests cannot extend a tombstone's lifetime indefinitely.
func (d *Directory) cleanup(now time.Time) {
	for id, v := range d.entries {
		if !v.removed && now.Sub(v.seen) >= d.ttl {
			v.removed = true
			v.seen = v.seen.Add(d.ttl)
			v.room = masterapi.PublicRoom{}
			d.entries[id] = v
		}
		if v.removed && now.Sub(v.seen) >= TombstoneRetention {
			delete(d.entries, id)
		}
	}
}
func (d *Directory) quota(ip, except string) (ipCount, active int) {
	for id, v := range d.entries {
		if id == except {
			continue
		}
		if v.ip == ip {
			ipCount++
		}
		if !v.removed {
			active++
		}
	}
	return
}

func reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func (d *Directory) rooms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost && r.Method != http.MethodDelete {
		w.Header().Set("Allow", "GET, POST, DELETE")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	now := d.now()
	ip := requestIP(r, d.trustProxy)
	if ip == "" || !d.allow(r.Method, ip, now) {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "request limit reached", http.StatusTooManyRequests)
		return
	}
	switch r.Method {
	case http.MethodGet:
		d.list(w, r, now)
	case http.MethodPost:
		d.register(w, r, ip, now)
	case http.MethodDelete:
		d.remove(w, r, ip, now)
	}
}

func urlValues(r *http.Request) (url.Values, error) {
	if len(r.URL.RawQuery) > 128 {
		return nil, errors.New("query too long")
	}
	return url.ParseQuery(r.URL.RawQuery)
}
func (d *Directory) list(w http.ResponseWriter, r *http.Request, now time.Time) {
	query, err := urlValues(r)
	if err != nil || len(query) != 1 || len(query["protocol"]) != 1 {
		http.Error(w, "protocol required", http.StatusBadRequest)
		return
	}
	raw := query.Get("protocol")
	protocol, err := strconv.Atoi(raw)
	if err != nil || protocol <= 0 || protocol > 100000 || strconv.Itoa(protocol) != raw {
		http.Error(w, "invalid protocol", http.StatusBadRequest)
		return
	}
	rooms := make([]masterapi.PublicRoom, 0)
	d.mu.Lock()
	d.cleanup(now)
	for _, v := range d.entries {
		if !v.removed && v.room.Protocol == protocol {
			rooms = append(rooms, v.room)
		}
	}
	d.mu.Unlock()
	sort.Slice(rooms, func(i, j int) bool {
		fullI, fullJ := rooms[i].Players == rooms[i].MaxPlayers, rooms[j].Players == rooms[j].MaxPlayers
		if fullI != fullJ {
			return !fullI
		}
		if rooms[i].Name != rooms[j].Name {
			return rooms[i].Name < rooms[j].Name
		}
		return rooms[i].ID < rooms[j].ID
	})
	// Registration enforces this globally; retain a response bound defensively.
	if len(rooms) > masterapi.MaxPublicRooms {
		rooms = rooms[:masterapi.MaxPublicRooms]
	}
	reply(w, http.StatusOK, masterapi.PublicRoomList{Rooms: rooms})
}

func ownerHash(r *http.Request) ([32]byte, bool) {
	values := r.Header.Values(masterapi.PublicRoomOwnerHeader)
	if len(values) != 1 || !masterapi.ValidHexIdentity(values[0], 32) {
		return [32]byte{}, false
	}
	return sha256.Sum256([]byte(values[0])), true
}
func sameOwner(a, b [32]byte) bool { return subtle.ConstantTimeCompare(a[:], b[:]) == 1 }

var roomFields = map[string]bool{
	"id": true, "endpoint": true, "name": true, "protocol": true, "players": true, "max_players": true,
	"game_mode": true, "lives": true, "rounds": true, "maps": true, "playlist_configured": true,
	"crates": true, "pickups": true, "rule_summary": true, "chaos_enabled": true, "chaos_events": true, "chaos_interval_seconds": true,
}

func decodeRoom(body io.Reader) (masterapi.PublicRoom, error) {
	var room masterapi.PublicRoom
	data, err := io.ReadAll(io.LimitReader(body, masterapi.MaxPublicRoomBody+1))
	if err != nil || len(data) > masterapi.MaxPublicRoomBody || !utf8.Valid(data) {
		return room, errors.New("invalid registration")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	token, err := dec.Token()
	if err != nil || token != json.Delim('{') {
		return room, errors.New("room object required")
	}
	seen := make(map[string]bool, len(roomFields))
	for dec.More() {
		key, err := dec.Token()
		name, ok := key.(string)
		if err != nil || !ok || !roomFields[name] || seen[name] {
			return room, errors.New("unknown or duplicate field")
		}
		seen[name] = true
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil || bytes.Equal(raw, []byte("null")) {
			return room, errors.New("invalid field")
		}
		if name == "maps" {
			var maps []json.RawMessage
			if err := json.Unmarshal(raw, &maps); err != nil || len(maps) != len(room.Maps) {
				return room, errors.New("invalid maps")
			}
			for _, value := range maps {
				var n int
				if bytes.Equal(value, []byte("null")) || json.Unmarshal(value, &n) != nil {
					return room, errors.New("invalid map value")
				}
			}
		}
	}
	if _, err := dec.Token(); err != nil {
		return room, err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return room, errors.New("trailing JSON")
	}
	if len(seen) != len(roomFields) {
		return room, errors.New("incomplete room")
	}
	// All names and shapes have been checked; normal decoding enforces primitive
	// types and integer ranges. The endpoint/preview validators run afterward.
	if err := json.Unmarshal(data, &room); err != nil {
		return room, err
	}
	return room, nil
}

func (d *Directory) register(w http.ResponseWriter, r *http.Request, ip string, now time.Time) {
	defer r.Body.Close()
	owner, valid := ownerHash(r)
	media, _, mediaErr := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if !valid || r.URL.RawQuery != "" || r.URL.ForceQuery || mediaErr != nil || media != "application/json" {
		http.Error(w, "invalid registration", http.StatusBadRequest)
		return
	}
	room, err := decodeRoom(r.Body)
	if err != nil {
		http.Error(w, "invalid registration", http.StatusBadRequest)
		return
	}
	if masterapi.ValidatePublicRoom(room) != nil {
		http.Error(w, "invalid room", http.StatusBadRequest)
		return
	}
	d.mu.Lock()
	d.cleanup(now)
	old, exists := d.entries[room.ID]
	if exists && !sameOwner(old.ownerHash, owner) {
		d.mu.Unlock()
		http.Error(w, "room ownership required", http.StatusUnauthorized)
		return
	}
	if exists && old.removed {
		d.mu.Unlock()
		http.Error(w, "room closed", http.StatusConflict)
		return
	}
	ipCount, active := d.quota(ip, room.ID)
	if ipCount >= maxEntriesPerIP || (!exists && (len(d.entries) >= maxEntries || active >= masterapi.MaxPublicRooms)) {
		d.mu.Unlock()
		http.Error(w, "room limit reached", http.StatusTooManyRequests)
		return
	}
	d.entries[room.ID] = entry{room: room, ownerHash: owner, ip: ip, seen: now}
	d.mu.Unlock()
	reply(w, http.StatusOK, struct {
		ID string `json:"id"`
	}{room.ID})
}

func (d *Directory) remove(w http.ResponseWriter, r *http.Request, ip string, now time.Time) {
	query, err := urlValues(r)
	if err != nil || len(query) != 1 || len(query["id"]) != 1 || !masterapi.ValidHexIdentity(query.Get("id"), 16) {
		http.Error(w, "invalid room", http.StatusBadRequest)
		return
	}
	id := query.Get("id")
	owner, valid := ownerHash(r)
	if !valid {
		http.Error(w, "room ownership required", http.StatusUnauthorized)
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cleanup(now)
	old, exists := d.entries[id]
	if exists && !sameOwner(old.ownerHash, owner) {
		http.Error(w, "room ownership required", http.StatusUnauthorized)
		return
	}
	if !exists {
		ipCount, _ := d.quota(ip, "")
		if ipCount >= maxEntriesPerIP || len(d.entries) >= maxEntries {
			http.Error(w, "room limit reached", http.StatusTooManyRequests)
			return
		}
		old = entry{ownerHash: owner, ip: ip}
	}
	// Unknown IDs are tombstoned too: DELETE can arrive before the initial POST.
	// Repeated deletion is idempotent and never refreshes the retention clock.
	if !old.removed {
		old.removed = true
		old.seen = now
		old.room = masterapi.PublicRoom{}
	}
	d.entries[id] = old
	w.WriteHeader(http.StatusNoContent)
}
