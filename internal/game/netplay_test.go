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

	wantInput := PlayerInputState{Left: true, Up: true, Shoot: true}
	client.queueInput(wantInput)
	waitFor(t, "remote input", func() bool { return host.latestInput() == wantInput })

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
