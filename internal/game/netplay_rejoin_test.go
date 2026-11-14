package game

import "testing"

func TestNetplaySameGameRejoinResendsUnchangedCharacter(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	clientGame := New()
	defer clientGame.CloseNetplay()
	address := host.listener.Addr().String()
	for attempt := 0; attempt < 2; attempt++ {
		if err := clientGame.StartNetClient(address); err != nil {
			t.Fatal(err)
		}
		id := clientGame.netplay.assignedPlayer()
		cfg := sourceDefaultCustomPlayers()[0]
		cfg.Name = "Same Character"
		cfg.Perk = 7
		clientGame.customPlayers[id-1] = cfg
		clientGame.sendOnlineLocalConfigIfChanged()
		waitFor(t, "same-game character after join", func() bool {
			got, ok := host.latestConfigFor(id)
			return ok && got.Name == cfg.Name && got.Perk == cfg.Perk
		})
		if err := clientGame.CloseNetplay(); err != nil {
			t.Fatal(err)
		}
		waitFor(t, "same-game slot removal", func() bool { return host.connectedCount() == 0 })
	}
}

func TestNetplayCloseClearsOnlySessionCaches(t *testing.T) {
	g := New()
	g.customLives = 17
	g.customPlayers[0].Name = "Keep My Setup"
	g.onlineHasSentConfig = true
	g.onlineLastSentConfig = g.customPlayers[0]
	g.netLastSFXSeq = 900
	g.netSFXSeq = 400
	g.netSFXPending = []netSFXEvent{{Seq: 400, Name: "old-host.wav"}}
	g.netClientSFXPending = []netSFXEvent{{Seq: 900, Name: "old-client.wav"}}
	if err := g.CloseNetplay(); err != nil {
		t.Fatal(err)
	}
	if g.onlineHasSentConfig || g.onlineLastSentConfig != (CustomPlayerConfig{}) || g.netLastSFXSeq != 0 || g.netSFXSeq != 0 || len(g.netSFXPending) != 0 || len(g.netClientSFXPending) != 0 {
		t.Fatal("session-scoped state survived close")
	}
	if g.customLives != 17 || g.customPlayers[0].Name != "Keep My Setup" {
		t.Fatal("session close erased local setup")
	}
	g.applyNetSnapshot(netSnapshot{SFXEvents: []netSFXEvent{{Seq: 1, Name: "new-host.wav"}}})
	if len(g.netClientSFXPending) != 1 || g.netClientSFXPending[0].Name != "new-host.wav" {
		t.Fatal("new host's low sound sequence was suppressed")
	}
}
