package game

import "testing"

func TestHostSetupReleaseStaysLANOnly(t *testing.T) {
	g := New()
	g.customHostSetupOpen = true
	g.multiplayerSessionName = "Weekend Chaos"

	g.activateCustomHostSetupHit(menuHitCustomHostAccessInvite)
	if g.customHostScope != onlineSessionScopeLAN || g.customHostAccess != onlineSessionInviteOnly {
		t.Fatalf("LAN+INVITE selection mismatch: scope=%d access=%d", g.customHostScope, g.customHostAccess)
	}

	g.activateCustomHostSetupHit(menuHitCustomHostScopeInternet)
	if g.customHostScope != onlineSessionScopeLAN || g.customHostAccess != onlineSessionInviteOnly {
		t.Fatalf("disabled Internet option changed release scope: scope=%d access=%d", g.customHostScope, g.customHostAccess)
	}
	if g.multiplayerMessage == "" {
		t.Fatal("disabled Internet option did not explain that the release is LAN-only")
	}

	g.activateCustomHostSetupHit(menuHitCustomHostAccessOpen)
	if g.customHostScope != onlineSessionScopeLAN || g.customHostAccess != onlineSessionOpen {
		t.Fatalf("changing access unexpectedly changed LAN scope: scope=%d access=%d", g.customHostScope, g.customHostAccess)
	}

	g.activateCustomHostSetupHit(menuHitCustomHostBack)
	if g.multiplayerSessionName != "Weekend Chaos" || g.customHostScope != onlineSessionScopeLAN || g.customHostAccess != onlineSessionOpen {
		t.Fatal("host setup values were reset when closing create-session screen")
	}
}

func TestLiveHostSessionEditorCanLeaveToServerBrowser(t *testing.T) {
	g := New()
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	g.netplay = host
	g.customOnlineMode = true
	g.customNetworkOpen = true
	g.customHostSetupOpen = true
	x := customHostLeaveRect.X + customHostLeaveRect.W/2
	y := customHostLeaveRect.Y + customHostLeaveRect.H/2
	if hit := g.customHostSetupHitAt(x, y); hit != menuHitCustomHostLeave {
		t.Fatalf("LEAVE SESSION hit=%d want=%d", hit, menuHitCustomHostLeave)
	}
	g.activateCustomHostSetupHit(menuHitCustomHostLeave)
	if g.netplay != nil || !g.customNetworkOpen || g.customHostSetupOpen {
		t.Fatalf("leave session did not return to browser: netplay=%v network=%v setup=%v", g.netplay != nil, g.customNetworkOpen, g.customHostSetupOpen)
	}
}

func TestLANOpenHostAdvertisesOnlyLocally(t *testing.T) {
	g := New()
	g.multiplayerSessionName = "LAN Open"
	if err := g.StartNetHostConfig("127.0.0.1:0", onlineSessionScopeLAN, onlineSessionOpen); err != nil {
		t.Fatal(err)
	}
	defer g.CloseNetplay()
	if g.p2pHost != nil || g.natPortMapper != nil || g.serverAdvertiser != nil {
		t.Fatalf("LAN OPEN unexpectedly started Internet services: p2p=%v nat=%v master=%v", g.p2pHost != nil, g.natPortMapper != nil, g.serverAdvertiser != nil)
	}
	if g.localServerAdvertiser == nil || g.lanDiscoveryResponder == nil {
		t.Fatal("LAN OPEN did not advertise on local browser/discovery")
	}
	if g.netplay.inviteOnly {
		t.Fatal("LAN OPEN incorrectly requires invite authentication")
	}
}

func TestLANInviteHostIsPrivateAndCodeReachable(t *testing.T) {
	g := New()
	g.multiplayerSessionName = "LAN Invite"
	if err := g.StartNetHostConfig("127.0.0.1:0", onlineSessionScopeLAN, onlineSessionInviteOnly); err != nil {
		t.Fatal(err)
	}
	defer g.CloseNetplay()
	if g.p2pHost != nil || g.natPortMapper != nil || g.serverAdvertiser != nil {
		t.Fatal("LAN INVITE unexpectedly started Internet services")
	}
	if g.localServerAdvertiser != nil || g.lanDiscoveryResponder == nil {
		t.Fatal("LAN INVITE leaked onto an open LAN server list")
	}
	if !g.netplay.inviteOnly {
		t.Fatal("LAN INVITE did not require invite authentication")
	}
	code, ready := g.currentHostDisplayJoinCode()
	if !ready || len(code) != len(shortOpenCodePrefix)+6 {
		t.Fatalf("LAN INVITE did not produce a short join code: code=%q ready=%v", code, ready)
	}
}

func TestInternetInviteStaysOffOpenBrowser(t *testing.T) {
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "1")
	g := New()
	if err := g.StartNetHostConfig("127.0.0.1:0", onlineSessionScopeInternet, onlineSessionInviteOnly); err != nil {
		t.Fatal(err)
	}
	defer g.CloseNetplay()
	if g.localServerAdvertiser != nil || g.lanDiscoveryResponder == nil || g.serverAdvertiser != nil {
		t.Fatal("INTERNET INVITE leaked onto an OPEN browser")
	}
	if !g.netplay.inviteOnly {
		t.Fatal("INTERNET INVITE did not enable invite authentication")
	}
	if g.onlineSessionScope != onlineSessionScopeInternet || g.onlineSessionVisibility != onlineSessionInviteOnly {
		t.Fatal("host did not retain Internet+Invite configuration")
	}
}

func TestReturningToServerBrowserClosesHostAndKeepsCustomSettings(t *testing.T) {
	g := New()
	g.customMode = SourceGameModeGunGame
	g.customLives = 37
	g.customSeriesRounds = 7
	g.customMap = 5
	g.multiplayerSessionName = "Keep Me"
	g.customHostScope = onlineSessionScopeLAN
	g.customHostAccess = onlineSessionInviteOnly
	if err := g.StartNetHostConfig("127.0.0.1:0", onlineSessionScopeLAN, onlineSessionInviteOnly); err != nil {
		t.Fatal(err)
	}
	g.returnToOnlineServerBrowser()
	if g.netplay != nil {
		t.Fatal("host remained alive after returning to server browser")
	}
	if !g.customOnlineMode || !g.customNetworkOpen || g.onlineMatchSetup {
		t.Fatalf("browser state mismatch: online=%v browser=%v setup=%v", g.customOnlineMode, g.customNetworkOpen, g.onlineMatchSetup)
	}
	if g.customMode != SourceGameModeGunGame || g.customLives != 37 || g.customSeriesRounds != 7 || g.customMap != 5 {
		t.Fatalf("custom settings were reset: mode=%d lives=%d rounds=%d map=%d", g.customMode, g.customLives, g.customSeriesRounds, g.customMap)
	}
	if g.multiplayerSessionName != "Keep Me" || g.customHostScope != onlineSessionScopeLAN || g.customHostAccess != onlineSessionInviteOnly {
		t.Fatalf("session create settings were reset: name=%q scope=%d access=%d", g.multiplayerSessionName, g.customHostScope, g.customHostAccess)
	}
}
