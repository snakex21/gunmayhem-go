package game

import "testing"

// Every message type must survive the slim per-type encoder exactly as the
// full union struct would. This guards the wire format against a regression
// where a per-type view forgets a field the remote validator requires.
func TestEncodeNetMessageRoundTripAllTypes(t *testing.T) {
	cases := []netWireMessage{
		{Type: netMessageHello, Protocol: netplayProtocolVersion, PlayerID: 3, AuthProof: "proof-abcdef"},
		{Type: netMessageInput, Protocol: netplayProtocolVersion, PlayerID: 2, Input: PlayerInputState{Left: true, Shoot: true}},
		{Type: netMessageState, Protocol: netplayProtocolVersion, PlayerID: 2, State: netSnapshot{Screen: 1, TotalLives: 7, Players: []Player{{ID: 1, X: 5.5}}}},
		{Type: netMessagePlayerConfig, Protocol: netplayProtocolVersion, PlayerID: 4, Config: CustomPlayerConfig{Name: "Maks", Color: 2, Shirt: 1, Hat: 1, Eyes: 1, Gun: 1, Perk: 7, Type: 1, Team: 1}},
		{Type: netMessageChallenge, Protocol: netplayProtocolVersion, Challenge: "fedcba9876543210"},
		{Type: netMessageProbe, Protocol: netplayProtocolVersion},
		{Type: netMessageProbeReply, Protocol: netplayProtocolVersion, Probe: netProbeInfo{Name: "OPEN", Players: 2, MaxPlayers: 4, GameMode: SourceGameModeNormal, Lives: 7, Rounds: 7, DirectAddress: "1.2.3.4:7777"}},
		{Type: netMessagePostGameReady, Protocol: netplayProtocolVersion, PlayerID: 2, Ready: true},
	}
	codec := cborNetplayCodec{}
	for _, want := range cases {
		payload, err := codec.Encode(want)
		if err != nil {
			t.Fatalf("type=%d encode: %v", want.Type, err)
		}
		if len(payload) == 0 {
			t.Fatalf("type=%d empty payload", want.Type)
		}
		var got netWireMessage
		if err := codec.Decode(payload, &got); err != nil {
			t.Fatalf("type=%d decode: %v", want.Type, err)
		}
		if got.Type != want.Type || got.Protocol != want.Protocol {
			t.Fatalf("type=%d got Type=%d Protocol=%d", want.Type, got.Type, got.Protocol)
		}
		switch want.Type {
		case netMessageHello, netMessageChallenge:
			if got.Challenge != want.Challenge || got.AuthProof != want.AuthProof {
				t.Fatalf("type=%d auth fields mismatch: %+v", want.Type, got)
			}
		case netMessageInput:
			if got.Input != want.Input {
				t.Fatalf("input mismatch: %+v", got.Input)
			}
		case netMessageState:
			if got.State.TotalLives != 7 || len(got.State.Players) != 1 || got.State.Players[0].X != 5.5 {
				t.Fatalf("state mismatch: %+v", got.State)
			}
		case netMessagePlayerConfig:
			if got.Config.Name != "Maks" || got.Config.Gun != 1 {
				t.Fatalf("config mismatch: %+v", got.Config)
			}
		case netMessageProbeReply:
			if got.Probe.Name != "OPEN" || got.Probe.Players != 2 || got.Probe.DirectAddress != "1.2.3.4:7777" {
				t.Fatalf("probe mismatch: %+v", got.Probe)
			}
		case netMessagePostGameReady:
			if !got.Ready {
				t.Fatal("ready flag lost")
			}
		}
		// The remote validators are what actually reject a message on the wire;
		// run the decoded value through the same checks the other side uses.
		switch want.Type {
		case netMessageChallenge, netMessageState, netMessageProbeReply:
			if err := validateNetMessageForClient(got); err != nil {
				t.Fatalf("type=%d client validation: %v", want.Type, err)
			}
		default:
			// Client→host messages: hello (with auth proof), probe, input,
			// config and post-game ready all pass the host-side validator.
			if err := validateNetMessageForHost(got); err != nil {
				t.Fatalf("type=%d host validation: %v", want.Type, err)
			}
		}
	}
}
