package publicrooms

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"gunmayhem-go/internal/masterapi"
)

const testOwner = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
const otherOwner = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"

func testRoom(n int) masterapi.PublicRoom {
	id := fmt.Sprintf("%032x", n)
	return masterapi.PublicRoom{ID: id, Endpoint: "wss://public-test.trycloudflare.com" + masterapi.PublicRoomWebSocketPrefix + id, Name: fmt.Sprintf("Room %03d", n), Protocol: 20, Players: 1, MaxPlayers: 4, GameMode: 1, Lives: 10, Rounds: 1}
}
func roomJSON(t *testing.T, room masterapi.PublicRoom) []byte {
	t.Helper()
	data, err := json.Marshal(room)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func request(d *Directory, method, path, ip, owner string, body []byte) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	r.RemoteAddr = ip + ":12345"
	if owner != "" {
		r.Header.Set(masterapi.PublicRoomOwnerHeader, owner)
	}
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	d.Handler().ServeHTTP(w, r)
	return w
}
func post(t *testing.T, d *Directory, room masterapi.PublicRoom, ip, owner string) *httptest.ResponseRecorder {
	t.Helper()
	return request(d, http.MethodPost, masterapi.PublicRoomsPath, ip, owner, roomJSON(t, room))
}
func listing(t *testing.T, d *Directory) masterapi.PublicRoomList {
	t.Helper()
	w := request(d, http.MethodGet, masterapi.PublicRoomsPath+"?protocol=20", "203.0.113.250", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", w.Code, w.Body)
	}
	var got masterapi.PublicRoomList
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	return got
}

func TestDirectoryTLSLifecycleAndSecretSeparation(t *testing.T) {
	d := New(0, false)
	server := httptest.NewTLSServer(d.Handler())
	defer server.Close()
	client := server.Client()
	room := testRoom(1)
	r, err := http.NewRequest(http.MethodPost, server.URL+masterapi.PublicRoomsPath, bytes.NewReader(roomJSON(t, room)))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set(masterapi.PublicRoomOwnerHeader, testOwner)
	resp, err := client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || strings.Contains(string(body), testOwner) {
		t.Fatalf("register status=%d body=%s", resp.StatusCode, body)
	}
	resp, err = client.Get(server.URL + masterapi.PublicRoomsPath + "?protocol=20")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	for _, private := range []string{testOwner, otherOwner, "owner", "secret", "join_code", "GMT1-", "127.0.0.1"} {
		if strings.Contains(string(body), private) {
			t.Fatalf("private marker %q leaked: %s", private, body)
		}
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("cacheable directory")
	}
	var got masterapi.PublicRoomList
	if err := json.Unmarshal(body, &got); err != nil || len(got.Rooms) != 1 || got.Rooms[0] != room {
		t.Fatalf("listed room=%+v err=%v", got, err)
	}
	if d.entries[room.ID].ownerHash != sha256.Sum256([]byte(testOwner)) {
		t.Fatal("ownership was not hashed")
	}
	r, _ = http.NewRequest(http.MethodDelete, server.URL+masterapi.PublicRoomsPath+"?id="+room.ID, nil)
	r.Header.Set(masterapi.PublicRoomOwnerHeader, testOwner)
	resp, err = client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent || len(listing(t, d).Rooms) != 0 {
		t.Fatal("delete did not delist")
	}
}

func TestDirectoryOwnershipAndTombstones(t *testing.T) {
	d := New(0, false)
	now := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	d.now = func() time.Time { return now }
	room := testRoom(1)
	if w := post(t, d, room, "203.0.113.1", testOwner); w.Code != http.StatusOK {
		t.Fatal(w.Code)
	}
	updated := room
	updated.Name = "Attacker"
	if w := post(t, d, updated, "203.0.113.2", otherOwner); w.Code != http.StatusUnauthorized {
		t.Fatal(w.Code)
	}
	path := masterapi.PublicRoomsPath + "?id=" + room.ID
	if w := request(d, http.MethodDelete, path, "203.0.113.2", otherOwner, nil); w.Code != http.StatusUnauthorized {
		t.Fatal(w.Code)
	}
	if listing(t, d).Rooms[0].Name != room.Name {
		t.Fatal("ownership changed")
	}
	if w := request(d, http.MethodDelete, path, "203.0.113.1", testOwner, nil); w.Code != http.StatusNoContent {
		t.Fatal(w.Code)
	}
	deletedAt := d.entries[room.ID].seen
	now = now.Add(time.Second)
	if w := request(d, http.MethodDelete, path, "203.0.113.1", testOwner, nil); w.Code != http.StatusNoContent {
		t.Fatal(w.Code)
	}
	if d.entries[room.ID].seen != deletedAt {
		t.Fatal("idempotent delete refreshed tombstone")
	}
	if w := post(t, d, room, "203.0.113.1", testOwner); w.Code != http.StatusConflict {
		t.Fatalf("late heartbeat status=%d", w.Code)
	}
	if w := post(t, d, room, "203.0.113.1", otherOwner); w.Code != http.StatusUnauthorized {
		t.Fatal(w.Code)
	}
	if len(listing(t, d).Rooms) != 0 {
		t.Fatal("deleted room visible")
	}
	now = deletedAt.Add(TombstoneRetention)
	if w := post(t, d, room, "203.0.113.1", testOwner); w.Code != http.StatusOK {
		t.Fatalf("bounded tombstone did not expire: %d", w.Code)
	}
}

func TestDeleteBeforeRegistrationAndExpiryFailClosed(t *testing.T) {
	d := New(time.Second, false)
	now := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	d.now = func() time.Time { return now }
	room := testRoom(9)
	w := request(d, http.MethodDelete, masterapi.PublicRoomsPath+"?id="+room.ID, "203.0.113.1", testOwner, nil)
	if w.Code != http.StatusNoContent {
		t.Fatal(w.Code)
	}
	if w := post(t, d, room, "203.0.113.1", testOwner); w.Code != http.StatusConflict {
		t.Fatalf("canceled initial registration recreated room: %d", w.Code)
	}
	live := testRoom(10)
	if w := post(t, d, live, "203.0.113.1", testOwner); w.Code != http.StatusOK {
		t.Fatal(w.Code)
	}
	now = now.Add(time.Second)
	if got := listing(t, d); len(got.Rooms) != 0 {
		t.Fatal("TTL boundary did not expire")
	}
	if w := post(t, d, live, "203.0.113.1", testOwner); w.Code != http.StatusConflict {
		t.Fatal("expired ID reopened")
	}
	now = now.Add(TombstoneRetention)
	_ = listing(t, d)
	if len(d.entries) != 0 {
		t.Fatalf("stale tombstones retained=%d", len(d.entries))
	}
}

func TestDirectoryQuotasIncludeIPMovesAndTombstones(t *testing.T) {
	d := New(0, false)
	for i := 1; i <= maxEntriesPerIP; i++ {
		if w := post(t, d, testRoom(i), "203.0.113.1", testOwner); w.Code != http.StatusOK {
			t.Fatalf("%d: %d", i, w.Code)
		}
	}
	if w := post(t, d, testRoom(100), "203.0.113.1", testOwner); w.Code != http.StatusTooManyRequests {
		t.Fatal("per-IP quota absent")
	}
	room := testRoom(101)
	if w := post(t, d, room, "203.0.113.2", testOwner); w.Code != http.StatusOK {
		t.Fatal(w.Code)
	}
	if w := post(t, d, room, "203.0.113.1", testOwner); w.Code != http.StatusTooManyRequests {
		t.Fatal("move bypassed per-IP quota")
	}
	if d.entries[room.ID].ip != "203.0.113.2" {
		t.Fatal("rejected move changed entry")
	}
	id := testRoom(1).ID
	request(d, http.MethodDelete, masterapi.PublicRoomsPath+"?id="+id, "203.0.113.1", testOwner, nil)
	if w := post(t, d, testRoom(100), "203.0.113.1", testOwner); w.Code != http.StatusTooManyRequests {
		t.Fatal("tombstone bypassed IP quota")
	}
	// Existing owner can still refresh its own entry at the capacity limit.
	if w := post(t, d, testRoom(2), "203.0.113.1", testOwner); w.Code != http.StatusOK {
		t.Fatal("existing refresh blocked")
	}
}

func TestDirectoryGlobalAndResponseCaps(t *testing.T) {
	d := New(0, false)
	for i := 0; i < masterapi.MaxPublicRooms; i++ {
		room := testRoom(i + 1)
		room.Name = strings.Repeat("<", 48)
		room.RuleSummary = strings.Repeat("<", 96)
		for j := range room.Maps {
			room.Maps[j] = -100000
		}
		if w := post(t, d, room, fmt.Sprintf("203.0.113.%d", i/maxEntriesPerIP+1), testOwner); w.Code != http.StatusOK {
			t.Fatalf("entry %d status=%d", i, w.Code)
		}
	}
	if w := post(t, d, testRoom(999), "198.51.100.1", testOwner); w.Code != http.StatusTooManyRequests {
		t.Fatal("active cap absent")
	}
	w := request(d, http.MethodGet, masterapi.PublicRoomsPath+"?protocol=20", "198.51.100.1", "", nil)
	if w.Body.Len() > masterapi.MaxPublicRoomListBody {
		t.Fatalf("list bytes=%d limit=%d", w.Body.Len(), masterapi.MaxPublicRoomListBody)
	}
	if len(listing(t, d).Rooms) != masterapi.MaxPublicRooms {
		t.Fatal("incorrect capped list")
	}
	// Tombstones count toward a separate hard memory bound.
	d = New(0, false)
	for i := 0; i < maxEntries; i++ {
		d.entries[fmt.Sprintf("%032x", i+1)] = entry{ip: "203.0.113.1", removed: true, seen: time.Now(), ownerHash: sha256.Sum256([]byte(testOwner))}
	}
	if w := post(t, d, testRoom(5000), "198.51.100.1", testOwner); w.Code != http.StatusTooManyRequests {
		t.Fatal("entry map cap absent")
	}
	if w := request(d, http.MethodDelete, masterapi.PublicRoomsPath+"?id="+testRoom(5000).ID, "198.51.100.1", testOwner, nil); w.Code != http.StatusTooManyRequests {
		t.Fatal("unknown DELETE bypassed entry cap")
	}
	if len(d.entries) != maxEntries {
		t.Fatal("entry map exceeded cap")
	}
}

func TestDirectoryRateLimitsAndHardMapCap(t *testing.T) {
	d := New(0, false)
	now := time.Now()
	for i := 0; i < 120; i++ {
		if !d.allow(http.MethodPost, "203.0.113.1", now) {
			t.Fatal("early limit")
		}
	}
	if d.allow(http.MethodPost, "203.0.113.1", now) {
		t.Fatal("POST rate limit absent")
	}
	if !d.allow(http.MethodGet, "203.0.113.1", now) {
		t.Fatal("endpoint classes share quota")
	}
	d = New(0, false)
	for i := 0; i < maxRateKeys; i++ {
		if !d.allow(http.MethodGet, fmt.Sprint(i), now) {
			t.Fatal("early map cap")
		}
	}
	if d.allow(http.MethodGet, "extra", now) || len(d.rates) != maxRateKeys {
		t.Fatal("rate map exceeded cap")
	}
	if !d.allow(http.MethodGet, "0", now) {
		t.Fatal("known key cannot proceed at cap")
	}
	if !d.allow(http.MethodGet, "extra", now.Add(rateWindow)) || len(d.rates) != 1 {
		t.Fatal("expired rate keys not reclaimed")
	}
}

func TestDirectoryRejectsMalformedJSONAndOwnership(t *testing.T) {
	base := string(roomJSON(t, testRoom(1)))
	bad := map[string]string{
		"unknown":      strings.TrimSuffix(base, "}") + `,"private":true}`,
		"duplicate":    strings.TrimSuffix(base, "}") + `,"id":"` + testRoom(2).ID + `"}`,
		"case alias":   strings.Replace(base, `"id":`, `"ID":`, 1),
		"trailing":     base + ` {}`,
		"null":         strings.Replace(base, `"rule_summary":""`, `"rule_summary":null`, 1),
		"missing":      strings.Replace(base, `"rule_summary":"",`, ``, 1),
		"short maps":   strings.Replace(base, `[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]`, `[0]`, 1),
		"null map":     strings.Replace(base, `[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]`, `[null,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]`, 1),
		"oversized":    base + strings.Repeat(" ", masterapi.MaxPublicRoomBody),
		"invalid UTF8": strings.Replace(base, "Room 001", "bad\xff", 1),
		"array":        "[]",
	}
	for name, body := range bad {
		t.Run(name, func(t *testing.T) {
			d := New(0, false)
			w := request(d, http.MethodPost, masterapi.PublicRoomsPath, "203.0.113.1", testOwner, []byte(body))
			if w.Code != http.StatusBadRequest || len(d.entries) != 0 {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}
		})
	}
	for _, owner := range []string{"", strings.ToUpper(testOwner), testOwner + ", " + otherOwner, "short"} {
		d := New(0, false)
		if w := post(t, d, testRoom(1), "203.0.113.1", owner); w.Code != http.StatusBadRequest {
			t.Fatalf("bad owner accepted: %d", w.Code)
		}
	}
	d := New(0, false)
	r := httptest.NewRequest(http.MethodPost, masterapi.PublicRoomsPath, bytes.NewReader([]byte(base)))
	r.RemoteAddr = "203.0.113.1:1"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Add(masterapi.PublicRoomOwnerHeader, testOwner)
	r.Header.Add(masterapi.PublicRoomOwnerHeader, otherOwner)
	w := httptest.NewRecorder()
	d.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatal("duplicate ownership headers accepted")
	}
}

func TestPublicRoomRejectsRoutesAndControlText(t *testing.T) {
	room := testRoom(1)
	if err := masterapi.ValidatePublicRoom(room); err != nil {
		t.Fatal(err)
	}
	badRoutes := []string{
		"ws://public-test.trycloudflare.com" + masterapi.PublicRoomWebSocketPrefix + room.ID,
		"wss://127.0.0.1" + masterapi.PublicRoomWebSocketPrefix + room.ID,
		"wss://x.trycloudflare.com.evil.example" + masterapi.PublicRoomWebSocketPrefix + room.ID,
		"wss://nested.x.trycloudflare.com" + masterapi.PublicRoomWebSocketPrefix + room.ID,
		"wss://user@public-test.trycloudflare.com" + masterapi.PublicRoomWebSocketPrefix + room.ID,
		strings.Replace(room.Endpoint, ".com/", ".com:443/", 1), room.Endpoint + "?owner=x", room.Endpoint + "?", room.Endpoint + "#", room.Endpoint + "#x", room.Endpoint + "/",
		strings.Replace(room.Endpoint, "/ws/", "/%77s/", 1), strings.Replace(room.Endpoint, "public-test", "PUBLIC-TEST", 1),
		strings.Replace(room.Endpoint, room.ID, testRoom(2).ID, 1),
	}
	for _, endpoint := range badRoutes {
		r := room
		r.Endpoint = endpoint
		if masterapi.ValidatePublicRoom(r) == nil {
			t.Fatalf("route accepted: %q", endpoint)
		}
	}
	for _, name := range []string{"", " leading", "trailing ", "new\nline", "nul\x00", "bidi\u202e", "line\u2028", "paragraph\u2029", strings.Repeat("a", 49)} {
		r := room
		r.Name = name
		if masterapi.ValidatePublicRoom(r) == nil {
			t.Fatalf("text accepted: %q", name)
		}
	}
	room.MaxPlayers = 5
	if masterapi.ValidatePublicRoom(room) == nil {
		t.Fatal("more than four players accepted")
	}
}

func TestDirectoryStrictQueriesRoutingAndProtocolFilter(t *testing.T) {
	d := New(0, false)
	for _, path := range []string{masterapi.PublicRoomsPath, masterapi.PublicRoomsPath + "?protocol=20&code=ABC234", masterapi.PublicRoomsPath + "?protocol=20&protocol=20", masterapi.PublicRoomsPath + "?protocol=020", masterapi.PublicRoomsPath + "?protocol=%GG", masterapi.PublicRoomsPath + "?protocol=20;code=x"} {
		if w := request(d, http.MethodGet, path, "203.0.113.1", "", nil); w.Code != http.StatusBadRequest {
			t.Fatalf("query accepted %s status=%d", path, w.Code)
		}
	}
	for _, path := range []string{masterapi.ServersPath, masterapi.HeartbeatPath, masterapi.RelayWebSocketPath, "/v2/rooms/", "/v2/%72ooms"} {
		if w := request(d, http.MethodGet, path, "203.0.113.1", "", nil); w.Code != http.StatusNotFound {
			t.Fatalf("legacy/alternate route exposed: %s status=%d", path, w.Code)
		}
	}
	room := testRoom(1)
	room.Protocol = 19
	post(t, d, room, "203.0.113.1", testOwner)
	if len(listing(t, d).Rooms) != 0 {
		t.Fatal("incompatible protocol listed")
	}
	if w := request(d, http.MethodPut, masterapi.PublicRoomsPath, "203.0.113.1", "", nil); w.Code != http.StatusMethodNotAllowed {
		t.Fatal(w.Code)
	}
}

func TestDirectoryIgnoresUntrustedForwardedIP(t *testing.T) {
	d := New(0, false)
	room := testRoom(1)
	r := httptest.NewRequest(http.MethodPost, masterapi.PublicRoomsPath, bytes.NewReader(roomJSON(t, room)))
	r.RemoteAddr = "203.0.113.1:321"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set(masterapi.PublicRoomOwnerHeader, testOwner)
	r.Header.Set("X-Forwarded-For", "198.51.100.1")
	w := httptest.NewRecorder()
	d.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK || d.entries[room.ID].ip != "203.0.113.1" {
		t.Fatal("spoofed IP trusted")
	}
	if requestIP(r, true) != "198.51.100.1" {
		t.Fatal("explicit proxy mode ignored")
	}
}

func TestDirectoryConcurrentCloseCannotResurrectRoom(t *testing.T) {
	d := New(0, false)
	room := testRoom(1)
	body := roomJSON(t, room)
	if w := post(t, d, room, "203.0.113.1", testOwner); w.Code != http.StatusOK {
		t.Fatal(w.Code)
	}
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			request(d, http.MethodPost, masterapi.PublicRoomsPath, "203.0.113.1", testOwner, body)
		}()
	}
	request(d, http.MethodDelete, masterapi.PublicRoomsPath+"?id="+room.ID, "203.0.113.1", testOwner, nil)
	wg.Wait()
	if len(listing(t, d).Rooms) != 0 {
		t.Fatal("in-flight heartbeat recreated deleted room")
	}
}
