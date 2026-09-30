package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gunmayhem-go/internal/masterapi"
)

func TestPublicDirectoryModeExcludesLegacyAndRelay(t *testing.T) {
	handler, err := publicDirectoryHandler("public-rooms", 35*time.Second, false, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{masterapi.ServersPath, masterapi.HeartbeatPath, masterapi.RelayWebSocketPath} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusNotFound {
			t.Fatalf("legacy route %s exposed: %d", path, w.Code)
		}
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, masterapi.PublicRoomsPath+"?protocol=20", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("v2 list unavailable: %d", w.Code)
	}
	for _, args := range [][3]string{{"typo", "", ""}, {"public-rooms", ":7781", ""}, {"public-rooms", "", "wss://example.com/v1/relay"}} {
		if _, err := publicDirectoryHandler(args[0], time.Second, false, args[1], args[2]); err == nil {
			t.Fatalf("unsafe mode/flags accepted: %v", args)
		}
	}
}
