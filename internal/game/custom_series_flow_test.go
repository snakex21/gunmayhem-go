package game

import "testing"

// newSeriesRound starts a real two-human Custom Game series round.
func newSeriesRound(t *testing.T, rounds int) *Game {
	t.Helper()
	g := New()
	g.customLives = 1
	g.customSeriesRounds = rounds
	for i := range g.customPlayers {
		g.customPlayers[i].Type = 0
	}
	g.customPlayers[0].Type = 1
	g.customPlayers[1].Type = 1
	g.beginCustomSeries()
	g.startCustomGame()
	g.fadeActive = false
	g.screen = screenGameplay
	return g
}

// playRoundToPostGame eliminates loserID and runs Game.Update through the
// winner countdown and the end-game fade until Post Game is on screen.
func playRoundToPostGame(t *testing.T, g *Game, loserID int) {
	t.Helper()
	for _, p := range g.players {
		p.RemoteControlled = true // no live keyboard input in tests
		if p.ID == loserID {
			p.Lives = 1
			p.Kill(g.arena)
		}
	}
	for tick := 0; tick < 600 && (g.screen != screenPostGame || g.fadeActive); tick++ {
		_ = g.Update()
	}
	if g.screen != screenPostGame {
		t.Fatalf("round never reached Post Game (screen=%d gameWin=%v)", g.screen, g.GameWin)
	}
}

// TestCustomSeriesMapWinCountedThroughEndOfRound drives the real end of a
// round (last survivor -> countdown -> fade -> Post Game). The winner used to
// be cleared by finishSourceGameBeforeScreenChange before the round was
// recorded, so map wins always stayed 0.
func TestCustomSeriesMapWinCountedThroughEndOfRound(t *testing.T) {
	g := newSeriesRound(t, 3)
	playRoundToPostGame(t, g, 2)
	if got := g.customSeriesStats[0]; got.MapWins != 1 || got.RoundsPlayed != 1 {
		t.Fatalf("round 1 winner stats %+v", got)
	}
	if g.customSeriesStats[1].MapWins != 0 {
		t.Fatalf("round 1 loser got a map win: %+v", g.customSeriesStats[1])
	}
	// Staying on Post Game must not count the round again.
	for i := 0; i < 60; i++ {
		_ = g.Update()
	}
	if g.customSeriesStats[0].MapWins != 1 || g.customSeriesStats[0].RoundsPlayed != 1 {
		t.Fatalf("round counted twice: %+v", g.customSeriesStats[0])
	}

	// Round 2: player 2 wins.
	g.startNextCustomSeriesRound()
	g.fadeActive = false
	g.screen = screenGameplay
	playRoundToPostGame(t, g, 1)
	if g.customSeriesStats[0].MapWins != 1 || g.customSeriesStats[1].MapWins != 1 {
		t.Fatalf("after round 2: %+v / %+v", g.customSeriesStats[0], g.customSeriesStats[1])
	}
	if g.customSeriesStats[0].RoundsPlayed != 2 || g.customSeriesStats[1].RoundsPlayed != 2 {
		t.Fatalf("rounds played wrong: %+v", g.customSeriesStats)
	}
}

// TestCustomSeriesMapWinReachesOnlineGuest: the online host records the same
// way, and the guest shows the host's totals from the snapshot.
func TestCustomSeriesMapWinReachesOnlineGuest(t *testing.T) {
	g := newSeriesRound(t, 2)
	// Player 2 is a connected guest.
	peer := &netplayPeer{playerID: 2, stateSend: make(chan *netHostFrame, 1)}
	g.netplay = &netplaySession{mode: netplayHost, peers: map[int]*netplayPeer{2: peer}}
	g.onlineMatchSetup = true
	playRoundToPostGame(t, g, 2)
	if g.customSeriesStats[0].MapWins != 1 {
		t.Fatalf("online host map wins %+v", g.customSeriesStats[0])
	}
	snap := g.makeNetSnapshot()
	if snap.CustomSeriesStats[0].MapWins != 1 {
		t.Fatalf("snapshot map wins %+v", snap.CustomSeriesStats[0])
	}
	guest := New()
	guest.netplay = &netplaySession{mode: netplayClient}
	guest.applyNetSnapshot(snap)
	if guest.customSeriesStats[0].MapWins != 1 || guest.customSeriesStats[1].MapWins != 0 {
		t.Fatalf("guest series stats %+v", guest.customSeriesStats)
	}
}
