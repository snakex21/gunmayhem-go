package game

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"gunmayhem-go/internal/masterapi"
)

func TestFetchMasterServersUsesPublicDirectory(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != masterapi.ServersPath {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("protocol") != strconv.Itoa(netplayProtocolVersion) {
			t.Fatalf("protocol query=%q", r.URL.Query().Get("protocol"))
		}
		_ = json.NewEncoder(w).Encode(masterapi.ServerListResponse{Servers: []masterapi.Server{
			{Name: "Internet A", Address: "203.0.113.10:7777", Players: 1, MaxPlayers: 4, Protocol: netplayProtocolVersion,
				GameMode: SourceGameModeNormal, Lives: 7, Rounds: 3, Maps: [20]int{2, 7, 11}, PlaylistConfigured: true,
				Crates: true, Pickups: true, RuleSummary: "LOW GRAVITY"},
			{Name: "Internet B", Address: "198.51.100.8:7777", Players: 4, MaxPlayers: 4, Protocol: netplayProtocolVersion},
		}})
	}))
	defer ts.Close()
	t.Setenv("GUNMAYHEM_MASTER_URL", ts.URL)

	servers, err := fetchMasterServers(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 2 {
		t.Fatalf("servers=%d want2", len(servers))
	}
	if servers[0].Address != "203.0.113.10:7777" || servers[0].Full() {
		t.Fatalf("first server=%+v", servers[0])
	}
	if !servers[0].HasRulePreview() || servers[0].Lives != 7 || servers[0].Rounds != 3 || servers[0].Maps[0] != 2 || servers[0].Maps[1] != 7 || servers[0].Maps[2] != 11 || servers[0].RuleSummary != "LOW GRAVITY" {
		t.Fatalf("master preview was not preserved: %+v", servers[0])
	}
	if !servers[1].Full() {
		t.Fatalf("second server should be full: %+v", servers[1])
	}
}

func TestFetchMasterServerByInviteCodeKeepsRelayIdentity(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != masterapi.ServersPath {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("protocol") != strconv.Itoa(netplayProtocolVersion) || r.URL.Query().Get("code") != "ABC234" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(masterapi.ServerListResponse{Servers: []masterapi.Server{
			{ID: "relay-registration-id", SessionID: "public-session-id", Name: "Invite Host", Address: "203.0.113.20:7777", RelayAddress: "wss://relay.example/v1/relay", Players: 1, MaxPlayers: 4, Protocol: netplayProtocolVersion},
		}})
	}))
	defer ts.Close()
	t.Setenv("GUNMAYHEM_MASTER_URL", ts.URL)

	servers, err := fetchMasterServersByCode("abc234", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 {
		t.Fatalf("servers=%d want1", len(servers))
	}
	got := servers[0]
	if got.ID != "public-session-id" || got.RelayID != "relay-registration-id" || got.RelayAddress != "wss://relay.example/v1/relay" {
		t.Fatalf("invite master route=%+v", got)
	}
}

func TestNormalizeMasterServerURLRequiresTLSForPublicHost(t *testing.T) {
	if got := normalizeMasterServerURL("http://example.com"); got != "" {
		t.Fatalf("public insecure master accepted: %q", got)
	}
	if got := normalizeMasterServerURL("http://127.0.0.1:7780"); got == "" {
		t.Fatal("loopback HTTP master should be allowed for local testing")
	}
	if got := normalizeMasterServerURL("http://192.168.1.20:7780"); got == "" {
		t.Fatal("private LAN HTTP master should be allowed for local testing")
	}
	t.Setenv("GUNMAYHEM_ALLOW_INSECURE_MASTER", "1")
	if got := normalizeMasterServerURL("http://example.com"); got == "" {
		t.Fatal("explicit insecure-master override was ignored")
	}
}

func TestDecodeLimitedJSONRejectsOversizedMasterResponse(t *testing.T) {
	payload := `{"id":"` + strings.Repeat("x", 128) + `"}`
	var dst map[string]any
	if err := decodeLimitedJSON(strings.NewReader(payload), 32, &dst); err == nil {
		t.Fatal("oversized master response was accepted")
	}
}

func TestMasterAdvertiserMarksInviteSessionPrivate(t *testing.T) {
	heartbeat := make(chan masterapi.HeartbeatRequest, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == masterapi.HeartbeatPath {
			var hb masterapi.HeartbeatRequest
			if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
				t.Errorf("decode heartbeat: %v", err)
				http.Error(w, "bad", http.StatusBadRequest)
				return
			}
			heartbeat <- hb
			_ = json.NewEncoder(w).Encode(masterapi.HeartbeatResponse{ID: "private-reg", Secret: "secret", Address: "203.0.113.88:7777"})
			return
		}
		if r.Method == http.MethodDelete && r.URL.Path == masterapi.ServersPath {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()
	t.Setenv("GUNMAYHEM_MASTER_URL", ts.URL)

	s := &netplaySession{inviteOnly: true, shortJoinCode: "ABC234", publicSessionID: "session-private"}
	a := startMasterServerAdvertiser(7777, s)
	defer a.Close()
	select {
	case hb := <-heartbeat:
		if !hb.Private || hb.JoinCode != "ABC234" || hb.SessionID != "session-private" {
			t.Fatalf("private heartbeat=%+v", hb)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("private master heartbeat was not sent")
	}
}

func TestMasterAdvertiserRegistersHostedGame(t *testing.T) {
	heartbeat := make(chan masterapi.HeartbeatRequest, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == masterapi.HeartbeatPath:
			var hb masterapi.HeartbeatRequest
			if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
				t.Errorf("decode heartbeat: %v", err)
				http.Error(w, "bad", http.StatusBadRequest)
				return
			}
			select {
			case heartbeat <- hb:
			default:
			}
			_ = json.NewEncoder(w).Encode(masterapi.HeartbeatResponse{ID: "abc", Secret: "test-secret", Address: "203.0.113.77:7777"})
		case r.Method == http.MethodDelete && r.URL.Path == masterapi.ServersPath:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()
	t.Setenv("GUNMAYHEM_MASTER_URL", ts.URL)

	a := startMasterServerAdvertiser(7777, nil)
	defer a.Close()
	select {
	case hb := <-heartbeat:
		if hb.Port != 7777 || hb.Players != 1 || hb.MaxPlayers != maxOnlinePlayers || hb.Protocol != netplayProtocolVersion {
			t.Fatalf("heartbeat=%+v", hb)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("master heartbeat was not sent")
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		registered, addr, err := a.Status()
		if registered {
			if err != nil || addr != "203.0.113.77:7777" {
				t.Fatalf("advertiser status registered=%v addr=%q err=%v", registered, addr, err)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("advertiser did not enter registered state")
}
