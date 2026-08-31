package game

import "testing"

func TestOnlinePostGameReadyCountsConnectedPlayersOnly(t *testing.T) {
	g := New()
	g.onlineMatchSetup = true
	g.netplay = &netplaySession{
		mode: netplayHost,
		peers: map[int]*netplayPeer{
			2: {playerID: 2},
			3: {playerID: 3},
		},
	}
	g.postGameReady[0] = true
	g.netplay.peers[2].postGameReady.Store(true)

	ready, total := g.onlinePostGameReadyCounts()
	if ready != 2 || total != 3 {
		t.Fatalf("ready=%d total=%d want 2/3", ready, total)
	}

	delete(g.netplay.peers, 3)
	ready, total = g.onlinePostGameReadyCounts()
	if ready != 2 || total != 2 {
		t.Fatalf("after disconnect ready=%d total=%d want 2/2", ready, total)
	}
}

func TestResetOnlinePostGameReadyClearsHostAndPeers(t *testing.T) {
	g := New()
	g.onlineMatchSetup = true
	g.netplay = &netplaySession{
		mode: netplayHost,
		peers: map[int]*netplayPeer{
			2: {playerID: 2},
		},
	}
	g.postGameReady[0] = true
	g.postGameReadyCountdown = 90
	g.netplay.peers[2].postGameReady.Store(true)

	g.resetOnlinePostGameReady()
	if g.postGameReady[0] || g.postGameReadyCountdown != 0 || g.netplay.peers[2].postGameReady.Load() {
		t.Fatal("post-game ready state was not fully reset")
	}
}
