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

func registerTestServer(t *testing.T, url, forwardedIP string, hb masterapi.HeartbeatRequest) (*http.Response, masterapi.HeartbeatResponse) {
	t.Helper()
	body, err := json.Marshal(hb)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url+masterapi.HeartbeatPath, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-For", forwardedIP)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var answer masterapi.HeartbeatResponse
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
			resp.Body.Close()
			t.Fatal(err)
		}
	}
	resp.Body.Close()
	return resp, answer
}

func TestMasterServerIDCannotBeHijackedWithoutSecret(t *testing.T) {
	reg := newServerRegistry(35 * time.Second)
	reg.trustProxy = true
	ts := httptest.NewServer(newHandler(reg))
	defer ts.Close()

	resp, first := registerTestServer(t, ts.URL, "203.0.113.10", masterapi.HeartbeatRequest{
		Name: "Owner", Port: 7777, Players: 1, MaxPlayers: 4, Protocol: 4,
	})
	if resp.StatusCode != http.StatusOK || first.ID == "" || first.Secret == "" {
		t.Fatalf("initial registration failed: status=%d response=%+v", resp.StatusCode, first)
	}

	attackResp, _ := registerTestServer(t, ts.URL, "198.51.100.99", masterapi.HeartbeatRequest{
		ID: first.ID, Secret: "wrong-secret", Name: "Hijacker", Port: 9999, Players: 1, MaxPlayers: 4, Protocol: 4,
	})
	if attackResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("hijack status=%d want %d", attackResp.StatusCode, http.StatusUnauthorized)
	}

	goodResp, updated := registerTestServer(t, ts.URL, "198.51.100.55", masterapi.HeartbeatRequest{
		ID: first.ID, Secret: first.Secret, Name: "Owner", Port: 8888, Players: 1, MaxPlayers: 4, Protocol: 4,
	})
	if goodResp.StatusCode != http.StatusOK || updated.ID != first.ID || updated.Secret != first.Secret {
		t.Fatalf("authenticated update failed: status=%d response=%+v", goodResp.StatusCode, updated)
	}
}

func TestMasterRequestRateLimit(t *testing.T) {
	reg := newServerRegistry(35 * time.Second)
	for i := 0; i < maxServerDeleteRequestsPerIP; i++ {
		if !reg.allowRequest("delete", "203.0.113.8", maxServerDeleteRequestsPerIP) {
			t.Fatalf("request %d rejected before the configured limit", i+1)
		}
	}
	if reg.allowRequest("delete", "203.0.113.8", maxServerDeleteRequestsPerIP) {
		t.Fatal("request above the configured limit was accepted")
	}
	if !reg.allowRequest("delete", "203.0.113.9", maxServerDeleteRequestsPerIP) {
		t.Fatal("rate limit leaked across source IPs")
	}
	if !reg.allowRequest("list", "203.0.113.8", maxServerListRequestsPerIP) {
		t.Fatal("rate limit leaked across endpoint classes")
	}
}

func TestMasterDeleteRequiresSecret(t *testing.T) {
	reg := newServerRegistry(35 * time.Second)
	reg.entries["server"] = registryEntry{
		server: masterapi.Server{ID: "server", Protocol: 4},
		seen:   time.Now(),
		secret: "correct-secret",
	}
	ts := httptest.NewServer(newHandler(reg))
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+masterapi.ServersPath+"?id=server", nil)
	req.Header.Set(masterapi.ServerSecretHeader, "wrong-secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong-secret delete status=%d", resp.StatusCode)
	}

	req2, _ := http.NewRequest(http.MethodDelete, ts.URL+masterapi.ServersPath+"?id=server", nil)
	req2.Header.Set(masterapi.ServerSecretHeader, "correct-secret")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("authenticated delete status=%d", resp2.StatusCode)
	}
}
