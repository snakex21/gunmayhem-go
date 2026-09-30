package game

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicCatalogEmptyStateOutcomes(t *testing.T) {
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "1")
	for _, tc := range []struct {
		name   string
		status int
		body   string
		label  string
		state  publicCatalogState
	}{
		{"empty", http.StatusOK, `{"rooms":[]}`, "NO OPEN ROOMS FOUND", publicCatalogAvailable},
		{"unavailable", http.StatusServiceUnavailable, `unavailable`, "CATALOG UNAVAILABLE - RETRY", publicCatalogUnavailable},
		{"invalid reply", http.StatusOK, `not JSON`, "CATALOG UNAVAILABLE - RETRY", publicCatalogUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer directory.Close()
			servers, err := fetchPublicRooms(directory.URL, directory.Client())
			g := &Game{multiplayerCatalogState: publicCatalogSearching, multiplayerDiscoveryCh: make(chan masterServerDiscoveryResult, 1)}
			g.multiplayerDiscoveryCh <- masterServerDiscoveryResult{Servers: servers, MasterErr: err, Done: true}
			g.pollMultiplayerServerBrowser()
			if got := g.multiplayerServerListEmptyLabel(); got != tc.label || g.multiplayerCatalogState != tc.state {
				t.Fatalf("label=%q state=%d", got, g.multiplayerCatalogState)
			}
			if len(g.multiplayerServers) != 0 || g.multiplayerDiscoveryCh != nil {
				t.Fatal("empty catalog outcome not completed")
			}
		})
	}
}

func TestPublicCatalogMissingIsNotEmpty(t *testing.T) {
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "1")
	servers, err := fetchPublicRooms("", nil)
	if err == nil || err.Error() != "PUBLIC DIRECTORY NOT CONFIGURED" {
		t.Fatal("missing catalog not explained")
	}
	g := &Game{multiplayerCatalogState: publicCatalogUnconfigured, multiplayerDiscoveryCh: make(chan masterServerDiscoveryResult, 1)}
	g.multiplayerDiscoveryCh <- masterServerDiscoveryResult{Servers: servers, MasterErr: err, Done: true}
	g.pollMultiplayerServerBrowser()
	if got := g.multiplayerServerListEmptyLabel(); got != "CATALOG NOT CONFIGURED" {
		t.Fatalf("label=%q", got)
	}
	if !strings.Contains(g.multiplayerMessage, "NOT CONFIGURED") {
		t.Fatal("missing catalog detail was hidden")
	}
}

func TestPublicCatalogEmptyLabelSearchingAndLAN(t *testing.T) {
	g := &Game{multiplayerCatalogState: publicCatalogUnavailable, multiplayerDiscoveryDone: true}
	// An in-flight retry must not continue displaying the previous failure.
	g.multiplayerDiscoveryCh = make(chan masterServerDiscoveryResult, 1)
	if got := g.multiplayerServerListEmptyLabel(); got != "SEARCHING..." {
		t.Fatalf("retry label=%q", got)
	}
	g.multiplayerCatalogState = publicCatalogSearching
	g.multiplayerDiscoveryCh <- masterServerDiscoveryResult{Done: false}
	g.pollMultiplayerServerBrowser()
	if got := g.multiplayerServerListEmptyLabel(); got != "SEARCHING..." {
		t.Fatalf("partial LAN result label=%q", got)
	}
	g.multiplayerDiscoveryCh <- masterServerDiscoveryResult{Done: true}
	g.pollMultiplayerServerBrowser()
	if got := g.multiplayerServerListEmptyLabel(); got != "NO OPEN ROOMS FOUND" {
		t.Fatalf("successful retry label=%q", got)
	}
	g.multiplayerCatalogState = publicCatalogDisabled
	if got := g.multiplayerServerListEmptyLabel(); got != "NO OPEN SERVERS - USE JOIN CODE" {
		t.Fatalf("LAN-only label=%q", got)
	}
}

func TestPublicCatalogRefreshResetsPreviousOutcome(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "1")
	t.Setenv("GUNMAYHEM_MASTER_URL", "")
	g := &Game{multiplayerCatalogState: publicCatalogUnavailable, multiplayerDiscoveryDone: true}
	g.refreshMultiplayerServers()
	if g.multiplayerCatalogState != publicCatalogUnconfigured || g.multiplayerDiscoveryDone {
		t.Fatal("refresh retained previous catalog outcome")
	}
	if got := g.multiplayerServerListEmptyLabel(); got != "SEARCHING..." {
		t.Fatalf("refresh label=%q", got)
	}
	waitFor(t, "unconfigured browser search", func() bool {
		g.pollMultiplayerServerBrowser()
		return g.multiplayerDiscoveryCh == nil
	})
	if got := g.multiplayerServerListEmptyLabel(); got != "CATALOG NOT CONFIGURED" {
		t.Fatalf("unconfigured label=%q", got)
	}
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "")
	g.refreshMultiplayerServers()
	if g.multiplayerCatalogState != publicCatalogDisabled {
		t.Fatal("plain LAN refresh retained Internet status")
	}
	waitFor(t, "LAN browser search", func() bool {
		g.pollMultiplayerServerBrowser()
		return g.multiplayerDiscoveryCh == nil
	})
	if got := g.multiplayerServerListEmptyLabel(); got != "NO OPEN SERVERS - USE JOIN CODE" {
		t.Fatalf("LAN label=%q", got)
	}
}
