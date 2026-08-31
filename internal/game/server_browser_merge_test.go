package game

import "testing"

func TestMergeServerListsDeduplicatesSameSessionAcrossLANAndGlobal(t *testing.T) {
	lan := multiplayerServerInfo{
		ID: "SESSION-1", Name: "Test Game", Address: "192.168.1.10:7777",
		Transport: "lan", Sources: "LAN", Players: 2, MaxPlayers: 4, Protocol: netplayProtocolVersion,
	}
	global := multiplayerServerInfo{
		ID: "SESSION-1", Name: "Test Game", Address: "203.0.113.20:44000",
		Transport: "quic", Sources: "GLOBAL", Players: 2, MaxPlayers: 4, Protocol: netplayProtocolVersion,
	}

	got := mergeServerLists([]multiplayerServerInfo{lan}, []multiplayerServerInfo{global})
	if len(got) != 1 {
		t.Fatalf("merged servers=%d want 1: %+v", len(got), got)
	}
	if got[0].Address != lan.Address || got[0].Transport != "lan" {
		t.Fatalf("preferred route=%s/%s want LAN %s", got[0].Transport, got[0].Address, lan.Address)
	}
	if got[0].Sources != "LAN + GLOBAL" {
		t.Fatalf("sources=%q want %q", got[0].Sources, "LAN + GLOBAL")
	}
}

func TestPublicSessionIDIsStableAndDoesNotExposeKey(t *testing.T) {
	key := "ABCDEFGHIJKLMNOPQRSTUVWX12"
	a := publicOnlineSessionID(key)
	b := publicOnlineSessionID(key)
	if a == "" || a != b {
		t.Fatalf("session id not stable: %q %q", a, b)
	}
	if a == key {
		t.Fatal("public session id exposes session key")
	}
}
