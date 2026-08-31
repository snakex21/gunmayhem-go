package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gunmayhem-go/internal/masterapi"
)

func TestMasterHeartbeatAndList(t *testing.T) {
	reg := newServerRegistry(35 * time.Second)
	reg.trustProxy = true
	ts := httptest.NewServer(newHandler(reg))
	defer ts.Close()

	hb := masterapi.HeartbeatRequest{Name: "Test Host", Port: 7777, Players: 1, MaxPlayers: 2, Protocol: 1,
		GameMode: 1, Lives: 7, Rounds: 3, Maps: [20]int{2, 7, 11}, PlaylistConfigured: true,
		Crates: true, Pickups: true, RuleSummary: "LOW GRAVITY"}
	body, _ := json.Marshal(hb)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+masterapi.HeartbeatPath, bytes.NewReader(body))
	req.Header.Set("X-Forwarded-For", "203.0.113.44")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("heartbeat status=%d", resp.StatusCode)
	}
	var registered masterapi.HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&registered); err != nil {
		t.Fatal(err)
	}
	if registered.ID == "" || registered.Address != "203.0.113.44:7777" {
		t.Fatalf("heartbeat response=%+v", registered)
	}

	resp2, err := http.Get(ts.URL + masterapi.ServersPath + "?protocol=1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var list masterapi.ServerListResponse
	if err := json.NewDecoder(resp2.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list.Servers) != 1 || list.Servers[0].Address != "203.0.113.44:7777" || list.Servers[0].Name != "Test Host" {
		t.Fatalf("server list=%+v", list.Servers)
	}
	if list.Servers[0].GameMode != 1 || list.Servers[0].Lives != 7 || list.Servers[0].Rounds != 3 || list.Servers[0].Maps[0] != 2 || list.Servers[0].Maps[1] != 7 || list.Servers[0].Maps[2] != 11 || list.Servers[0].RuleSummary != "LOW GRAVITY" {
		t.Fatalf("server preview metadata=%+v", list.Servers[0])
	}
	resp3, err := http.Get(ts.URL + masterapi.ServersPath + "?protocol=1&code=ABC234")
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("invite-code lookup status=%d", resp3.StatusCode)
	}
	var codeList masterapi.ServerListResponse
	if err := json.NewDecoder(resp3.Body).Decode(&codeList); err != nil {
		t.Fatal(err)
	}
	if len(codeList.Servers) != 0 {
		t.Fatalf("unmatched invite code returned public servers: %+v", codeList.Servers)
	}
}

func TestPrivateMasterServerHiddenFromBrowserButResolvableByCode(t *testing.T) {
	reg := newServerRegistry(35 * time.Second)
	reg.trustProxy = true
	ts := httptest.NewServer(newHandler(reg))
	defer ts.Close()

	hb := masterapi.HeartbeatRequest{
		SessionID:  "session-private-1",
		JoinCode:   "ABC234",
		Private:    true,
		Name:       "Private Host",
		Port:       7777,
		Players:    1,
		MaxPlayers: 2,
		Protocol:   17,
	}
	body, _ := json.Marshal(hb)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+masterapi.HeartbeatPath, bytes.NewReader(body))
	req.Header.Set("X-Forwarded-For", "203.0.113.45")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("private heartbeat status=%d", resp.StatusCode)
	}

	publicResp, err := http.Get(ts.URL + masterapi.ServersPath + "?protocol=17")
	if err != nil {
		t.Fatal(err)
	}
	defer publicResp.Body.Close()
	var publicList masterapi.ServerListResponse
	if err := json.NewDecoder(publicResp.Body).Decode(&publicList); err != nil {
		t.Fatal(err)
	}
	if len(publicList.Servers) != 0 {
		t.Fatalf("private server leaked into browser: %+v", publicList.Servers)
	}

	privateResp, err := http.Get(ts.URL + masterapi.ServersPath + "?protocol=17&code=ABC234")
	if err != nil {
		t.Fatal(err)
	}
	defer privateResp.Body.Close()
	var privateList masterapi.ServerListResponse
	if err := json.NewDecoder(privateResp.Body).Decode(&privateList); err != nil {
		t.Fatal(err)
	}
	if len(privateList.Servers) != 1 || privateList.Servers[0].SessionID != "session-private-1" || privateList.Servers[0].Address != "203.0.113.45:7777" {
		t.Fatalf("private code lookup=%+v", privateList.Servers)
	}
}

func TestMasterExpiresStaleServers(t *testing.T) {
	reg := newServerRegistry(10 * time.Millisecond)
	reg.entries["dead"] = registryEntry{
		server: masterapi.Server{ID: "dead", Name: "Dead", Address: "203.0.113.1:7777", Players: 1, MaxPlayers: 2, Protocol: 1},
		seen:   time.Now().Add(-time.Second),
		ip:     "203.0.113.1",
	}

	req := httptest.NewRequest(http.MethodGet, masterapi.ServersPath+"?protocol=1", nil)
	w := httptest.NewRecorder()
	reg.list(w, req)
	var list masterapi.ServerListResponse
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list.Servers) != 0 {
		t.Fatalf("stale server remained: %+v", list.Servers)
	}
}
