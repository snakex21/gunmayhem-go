package game

import (
	"testing"
	"time"
)

func waitFor(t *testing.T, what string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestNetplayLoopbackInputAndSnapshot(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	client, err := newClientNetplay(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	waitFor(t, "host connection", host.connected)
	waitFor(t, "client player assignment", func() bool { return client.assignedPlayer() == 2 })

	wantInput := PlayerInputState{Left: true, Up: true, Shoot: true}
	client.queueInput(wantInput)
	waitFor(t, "remote input", func() bool { return host.latestInputFor(2) == wantInput })

	wantState := netSnapshot{
		Screen:     int(screenGameplay),
		Arena:      OriginalMap1(),
		GameMode:   SourceGameModeNormal,
		TotalLives: 7,
		Players: []Player{
			{ID: 1, Name: "Host", X: 123, Y: 234, Active: true, Weapon: NewWeapon(10)},
			{ID: 2, Name: "Client", X: 456, Y: 345, Active: true, Weapon: NewWeapon(19)},
		},
		Bullets: []Bullet{{Kind: BulletNormal, X: 10, Y: 20, VX: 5, OwnerID: 1}},
	}
	host.queueState(wantState)
	waitFor(t, "authoritative snapshot", func() bool {
		got, ok := client.latestState()
		return ok && got.TotalLives == 7 && len(got.Players) == 2 && got.Players[1].Name == "Client" && len(got.Bullets) == 1
	})
}

func TestNetplayAssignsPlayersTwoThroughFour(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	clients := make([]*netplaySession, 0, 4)
	for i := 0; i < 4; i++ {
		client, err := newClientNetplay(host.listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		clients = append(clients, client)
		defer client.Close()
		if i < 3 {
			wantID := i + 2
			waitFor(t, "client player assignment", func() bool { return client.assignedPlayer() == wantID })
		}
	}
	waitFor(t, "three remote host peers", func() bool { return host.connectedCount() == 3 })
	if got := host.connectedPlayerIDs(); len(got) != 3 || got[0] != 2 || got[1] != 3 || got[2] != 4 {
		t.Fatalf("connected player IDs=%v want [2 3 4]", got)
	}

	inputs := []PlayerInputState{
		{Left: true},
		{Right: true, Shoot: true},
		{Up: true, Grenade: true},
	}
	for i := 0; i < 3; i++ {
		clients[i].queueInput(inputs[i])
		id := i + 2
		want := inputs[i]
		waitFor(t, "separate remote input", func() bool { return host.latestInputFor(id) == want })
	}

	// The fourth remote connection is over capacity. It must not displace P2-P4.
	deadline := time.Now().Add(700 * time.Millisecond)
	for time.Now().Before(deadline) && clients[3].connected() {
		time.Sleep(5 * time.Millisecond)
	}
	if host.connectedCount() != 3 {
		t.Fatalf("host peer count=%d want3 after over-capacity join", host.connectedCount())
	}
	if clients[3].assignedPlayer() != 0 {
		t.Fatalf("over-capacity client got player id %d", clients[3].assignedPlayer())
	}
}

func TestNetSnapshotQueuesEachSFXEventOnlyOnce(t *testing.T) {
	g := New()
	state := netSnapshot{SFXEvents: []netSFXEvent{{Seq: 1, Name: "hit1.wav"}, {Seq: 2, Name: "explosion1.wav", Loud: true}}}
	g.applyNetSnapshot(state)
	g.applyNetSnapshot(state)
	if got := len(g.netClientSFXPending); got != 2 {
		t.Fatalf("queued network SFX=%d want 2 unique events", got)
	}
	if g.netLastSFXSeq != 2 {
		t.Fatalf("last network SFX sequence=%d want 2", g.netLastSFXSeq)
	}
}

func TestRemoteControlledPlayerUsesNetworkInput(t *testing.T) {
	p := NewPlayer(2, OriginalMap1())
	p.AI = true
	p.RemoteControlled = true
	p.RemoteInput = PlayerInputState{Right: true, Down: true, Shoot: true, Grenade: true}

	left, right, up, down := p.movementInput()
	if left || !right || up || !down || !p.shootPressed() || !p.grenadePressed() {
		t.Fatalf("remote input not authoritative: left=%v right=%v up=%v down=%v shoot=%v grenade=%v", left, right, up, down, p.shootPressed(), p.grenadePressed())
	}
}

func TestHostRemoteInputDoesNotDisableOnlineBot(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	g := New()
	g.netplay = host
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.customPlayers[1].Type = 2
	bot := NewPlayer(2, OriginalMap1())
	bot.AI = true
	g.players = []*Player{bot}

	g.applyHostRemoteInput()
	if !bot.AI || bot.RemoteControlled {
		t.Fatalf("online bot was converted to remote player: ai=%v remote=%v", bot.AI, bot.RemoteControlled)
	}
}
