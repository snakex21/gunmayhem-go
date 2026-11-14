package game

import (
	"math/rand"
	"testing"
)

// measureNetSnapshot is the wire size of a snapshot sent as the FIRST message
// of a fresh stream (worst case: no history to compress against, map included).
func measureNetSnapshot(t testing.TB, state netSnapshot) int {
	t.Helper()
	var enc netHostEncoder
	frame := enc.encode(state)
	body, rawLen, err := newNetPeerStream().message(frame)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := (cborNetplayCodec{}).Encode(netWireMessage{
		Type:       netMessageState,
		Protocol:   netplayProtocolVersion,
		Tick:       frame.tick,
		Payload:    body,
		PayloadLen: rawLen,
	})
	if err != nil {
		t.Fatal(err)
	}
	return len(payload)
}

func measureNetMessage(t testing.TB, msg netWireMessage) int {
	t.Helper()
	payload, err := (cborNetplayCodec{}).Encode(msg)
	if err != nil {
		t.Fatal(err)
	}
	return len(payload)
}

// An input message is sent every rendered frame per client. It must stay tiny:
// it used to carry an empty netSnapshot plus a full netProbeInfo.
func TestNetInputMessageWireSize(t *testing.T) {
	idle := measureNetMessage(t, netWireMessage{
		Type: netMessageInput, Protocol: netplayProtocolVersion, PlayerID: 2,
	})
	held := measureNetMessage(t, netWireMessage{
		Type: netMessageInput, Protocol: netplayProtocolVersion, PlayerID: 2,
		InputSeq: 1000, InputFrames: []byte{1, 3, 19, 19, 17, 16, 0, 4},
	})
	config := measureNetMessage(t, netWireMessage{
		Type: netMessagePlayerConfig, Protocol: netplayProtocolVersion, PlayerID: 2,
		Config: CustomPlayerConfig{Name: "Guest", Type: 1, Gun: 10, Hat: 3},
	})
	t.Logf("netplay input CBOR: idle=%dB held=%dB config=%dB", idle, held, config)
	if held > 128 {
		t.Fatalf("input message=%dB, want <=128B: input is sent every frame and must stay tiny", held)
	}
	if config > 512 {
		t.Fatalf("player config message=%dB, want <=512B", config)
	}
}

// realisticNetSnapshot mirrors a busy four-player online game frame so payload
// size regressions are visible in tests instead of only on Cloudflare WSS.
func realisticNetSnapshot() netSnapshot {
	arena := OriginalMap1()
	arena.Platforms = append(arena.Platforms, make([]Rect, 24)...)

	state := netSnapshot{
		Screen:                int(screenGameplay),
		Arena:                 arena,
		GameMode:              SourceGameModeNormal,
		TotalLives:            7,
		TeamGame:              true,
		CrateON:               true,
		PowerON:               true,
		OnlineMatchSetup:      true,
		CustomLives:           7,
		CustomSeriesRounds:    7,
		ChaosEnabled:          true,
		ChaosCurrentEffect:    12,
		ChaosIntervalSeconds:  45,
		OnlinePlayerCount:     4,
		OnlinePlayerSlots:     [4]bool{true, true, true, true},
		OnlineMapRotation:     3,
		OnlineWeaponMode:      2,
		OnlineOnlyWeapon:      192,
		OnlinePerksEnabled:    true,
		OnlinePowerupsEnabled: true,
		PostGameReady:         [4]bool{true, false, false, false},
		OnlineCustomPlayers: [4]CustomPlayerConfig{
			{Name: "ASRock", Type: 1, Gun: 10, Hat: 3, Shirt: 7, Color: 2, Perk: 7, Throwable: ThrowableMk2, Team: 1},
			{Name: "Guest2", Type: 1, Gun: 19, Hat: 4, Shirt: 8, Color: 5, Perk: 7, Throwable: ThrowableMk2, Team: 2},
			{Name: "Guest3", Type: 1, Gun: 25, Hat: 5, Shirt: 9, Color: 8, Perk: 8, Throwable: ThrowableMk2, Team: 1},
			{Name: "Guest4", Type: 1, Gun: 30, Hat: 6, Shirt: 10, Color: 10, Perk: 9, Throwable: ThrowableMk2, Team: 2},
		},
		Players:        make([]Player, 0, 4),
		Bullets:        make([]Bullet, 0, 64),
		Shells:         make([]Shell, 0, 24),
		Flashes:        make([]MuzzleFlash, 0, 8),
		Grenades:       make([]Grenade, 0, 8),
		FirePatches:    make([]FirePatch, 0, 12),
		Portals:        make([]Portal, 0, 4),
		BlastFX:        make([]BlastEffect, 0, 24),
		JetThrustFX:    make([]JetThrustEffect, 0, 12),
		DropPackFX:     make([]DropPackEffect, 0, 8),
		Explosions:     make([]Explosion, 0, 6),
		Crates:         make([]Crate, 0, 8),
		Powerups:       make([]Powerup, 0, 6),
		PlayerTrails:   make([]PlayerTrailEffect, 0, 8),
		InstagibTrails: make([]InstagibTrailEffect, 0, 8),
		Killfeeds:      make([]KillFeedEntry, 0, 4),
		SFXEvents:      []netSFXEvent{{Seq: 11, Name: "hit1.wav"}, {Seq: 12, Name: "explosion1.wav", Loud: true}},
	}

	for i := 0; i < 4; i++ {
		p := Player{
			ID:           i + 1,
			Name:         state.OnlineCustomPlayers[i].Name,
			X:            320 + float64(i)*97.25,
			Y:            240.5 + float64(i)*13.75,
			VX:           2.5,
			VY:           -1.25,
			Facing:       1,
			Active:       true,
			Grounded:     true,
			Lives:        4,
			TotalLives:   7,
			CurrentGun:   10 + i,
			CurrentLevel: 3 + i,
			Score:        120 + i,
			ShotsFired:   40,
			HitsLanded:   21,
			Deaths:       2,
			Kills:        3,
			PlayerColor:  i + 1,
			ShirtNumber:  i + 7,
			HatNumber:    i + 3,
			HeadFrame:    4,
			VisualBodyY:  1.5,
			VisualEyesY:  2.5,
			Alpha:        1,
			Weapon: WeaponState{
				Def:      NewWeapon(10 + i).Def,
				Bullets:  7,
				Frame:    3,
				Playing:  true,
				Alpha:    1,
				WaitTime: 4,
			},
		}
		state.Players = append(state.Players, p)
	}

	for i := 0; i < 64; i++ {
		state.Bullets = append(state.Bullets, Bullet{
			Kind:            BulletNormal,
			X:               rand.Float64() * 800,
			Y:               rand.Float64() * 600,
			VX:              rand.Float64() * 8,
			VY:              rand.Float64() * 8,
			Rotation:        rand.Float64() * 360,
			Firepower:       12.5,
			OwnerID:         1 + i%4,
			Facing:          1,
			Gravity:         0.25,
			ExplosionRadius: 48,
			Lifetime:        30,
			Time:            i % 20,
			Alpha:           1,
			TrailAlpha:      0.5,
			TrailWidth:      16,
			TrailFrame:      3,
			VisualScale:     1,
		})
	}
	for i := 0; i < 24; i++ {
		state.Shells = append(state.Shells, Shell{
			Kind: i % 3, OwnerID: 1 + i%4, X: rand.Float64() * 800, Y: rand.Float64() * 600,
			VX: rand.Float64() * 6, VY: rand.Float64() * 6, Rotation: rand.Float64() * 360,
			RotSpeed: 4, ScaleX: 1,
		})
	}
	for i := 0; i < 8; i++ {
		state.Flashes = append(state.Flashes, MuzzleFlash{X: rand.Float64() * 800, Y: rand.Float64() * 600, Alpha: 1, Life: 3, Scale: 1.5})
		state.Grenades = append(state.Grenades, Grenade{X: rand.Float64() * 800, Y: rand.Float64() * 600, VX: rand.Float64() * 4, VY: rand.Float64() * 4, Rotation: rand.Float64() * 360, Fuse: 90, OwnerID: 1 + i%4, Type: 1})
		state.Powerups = append(state.Powerups, Powerup{Serial: i + 1, X: rand.Float64() * 800, Y: rand.Float64() * 600, Type: i%7 + 1})
		state.Explosions = append(state.Explosions, Explosion{X: rand.Float64() * 800, Y: rand.Float64() * 600, Life: 12})
		state.PlayerTrails = append(state.PlayerTrails, PlayerTrailEffect{X: rand.Float64() * 800, Y: rand.Float64() * 600, PlayerColor: i%4 + 1, Life: 8, Scale: 0.8})
		state.InstagibTrails = append(state.InstagibTrails, InstagibTrailEffect{X: rand.Float64() * 800, Y: rand.Float64() * 600, Rotation: rand.Float64() * 360, Alpha: 0.6})
	}
	for i := 0; i < 12; i++ {
		state.FirePatches = append(state.FirePatches, FirePatch{X: rand.Float64() * 800, Y: rand.Float64() * 600, OwnerID: 1 + i%4, Life: 20, Frame: 2})
		state.JetThrustFX = append(state.JetThrustFX, JetThrustEffect{X: rand.Float64() * 800, Y: rand.Float64() * 600, VX: rand.Float64() * 5, VY: rand.Float64() * 5, Rotation: rand.Float64() * 360, Scale: 1, FXAlpha: 0.5})
	}
	for i := 0; i < 8; i++ {
		state.Crates = append(state.Crates, Crate{Serial: i + 1, X: rand.Float64() * 800, Y: rand.Float64() * 600, VY: 1.5, Facing: 1})
		state.DropPackFX = append(state.DropPackFX, DropPackEffect{X: rand.Float64() * 800, Y: rand.Float64() * 600, VX: rand.Float64() * 3, VY: rand.Float64() * 3, Rotation: rand.Float64() * 360, ScaleX: 1})
	}
	for i := 0; i < 24; i++ {
		state.BlastFX = append(state.BlastFX, BlastEffect{Kind: BlastEffectKind(1 + i%6), X: rand.Float64() * 800, Y: rand.Float64() * 600, VX: rand.Float64() * 5, VY: rand.Float64() * 5, Rotation: rand.Float64() * 360, RotSpeed: 3, ScaleX: 1, ScaleY: 1, Alpha: 0.8, Level: 1})
	}
	for i := 0; i < 4; i++ {
		state.Portals = append(state.Portals, Portal{X: rand.Float64() * 800, Y: rand.Float64() * 600, NormalX: 1, NormalY: 0, OwnerID: 1 + i%4, Color: 1 + i%2, Active: true})
		state.Killfeeds = append(state.Killfeeds, KillFeedEntry{Victim: state.OnlineCustomPlayers[i].Name, Killer: state.OnlineCustomPlayers[(i+1)%4].Name, Mod: 1 + i%4, Time: 30 + i, Alpha: 1, Y: 40 + float64(i)*18, Slot: i})
	}
	return state
}

func TestNetSnapshotWireSizeReport(t *testing.T) {
	rand.Seed(7)
	empty := measureNetSnapshot(t, netSnapshot{})
	full := measureNetSnapshot(t, realisticNetSnapshot())
	playersOnly := measureNetSnapshot(t, func() netSnapshot {
		s := realisticNetSnapshot()
		s.Bullets, s.Shells, s.BlastFX, s.FirePatches, s.JetThrustFX = nil, nil, nil, nil, nil
		return s
	}())
	t.Logf("netplay snapshot CBOR: empty=%dB players=%dB busy=%dB", empty, playersOnly, full)
	// A busy four-player frame is the worst case that must still fit a
	// comfortable Cloudflare WSS budget at 60 Hz. The measured payload fell
	// from ~51.7 kB to this bound; raise it deliberately, never silently.
	if full > 16*1024 {
		t.Fatalf("busy first snapshot=%dB, want <=16KiB", full)
	}
}
