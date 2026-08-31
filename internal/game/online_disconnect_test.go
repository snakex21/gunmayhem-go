package game

import "testing"

func TestOnlineHostReturnsToUnifiedSlotsWhenLeftAloneDuringMatch(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	g := New()
	g.netplay = host
	g.screen = screenGameplay
	g.onlineMatchSetup = true
	g.customOnlineMode = true
	g.customPlayers[0].Type = 1
	g.customPlayers[1].Type = 1 // remote human slot that has disconnected
	g.onlinePlayerSlots[1] = true
	for i := 2; i < len(g.customPlayers); i++ {
		g.customPlayers[i].Type = 0
	}
	g.GameWin = true
	g.teamGameWin = true
	g.customSeriesRound = 3
	g.customSeriesRecordedRound = 3

	originalHost := g.netplay
	g.syncOnlineHostState()
	if !g.returnOnlineHostToLobbyIfAlone() {
		t.Fatal("host stayed in gameplay after becoming the only active participant")
	}
	if g.screen != screenCustomGame || g.customPage != customPagePlayers {
		t.Fatalf("screen/page=%v/%d want Custom Game SLOTS", g.screen, g.customPage)
	}
	if g.netplay != originalHost || g.netplay == nil || g.netplay.mode != netplayHost {
		t.Fatal("returning to slots destroyed the hosted session")
	}
	if !g.onlineMatchSetup || !g.customOnlineMode || g.customNetworkOpen {
		t.Fatalf("online slots state mismatch: setup=%v online=%v network=%v", g.onlineMatchSetup, g.customOnlineMode, g.customNetworkOpen)
	}
	if g.customPlayers[1].Type != 0 || g.onlinePlayerSlots[1] {
		t.Fatalf("disconnected P2 did not become an empty slot: type=%d online=%v", g.customPlayers[1].Type, g.onlinePlayerSlots[1])
	}
	if g.GameWin || g.teamGameWin {
		t.Fatal("winner state survived the disconnect abort")
	}
	if g.customSeriesRound != 0 || g.customSeriesRecordedRound != 0 {
		t.Fatalf("series state survived disconnect: round=%d recorded=%d", g.customSeriesRound, g.customSeriesRecordedRound)
	}
	if g.multiplayerMessage != "PLAYER LEFT - WAITING FOR PLAYER / ADD AI" {
		t.Fatalf("message=%q", g.multiplayerMessage)
	}
}

func TestOnlineHostKeepsMatchWhenBotStillMakesTwoParticipants(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	g := New()
	g.netplay = host
	g.screen = screenGameplay
	g.onlineMatchSetup = true
	g.customPlayers[0].Type = 1
	g.customPlayers[1].Type = 2
	for i := 2; i < len(g.customPlayers); i++ {
		g.customPlayers[i].Type = 0
	}
	g.syncOnlineHostState()

	if g.returnOnlineHostToLobbyIfAlone() {
		t.Fatal("host aborted a valid host-versus-bot match")
	}
	if g.screen != screenGameplay {
		t.Fatalf("screen=%v want gameplay", g.screen)
	}
}
