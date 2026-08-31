package game

import (
	"encoding/binary"
	"net"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestSecureWireRoundTripClientInput(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	want := netWireMessage{
		Type:     netMessageInput,
		Protocol: netplayProtocolVersion,
		Input: PlayerInputState{
			Left:  true,
			Shoot: true,
		},
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- writeNetMessage(left, want, maxClientWireFrame)
	}()
	var got netWireMessage
	if err := readNetMessage(right, maxClientWireFrame, &got); err != nil {
		t.Fatal(err)
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	if got.Type != want.Type || got.Protocol != want.Protocol || !got.Input.Left || !got.Input.Shoot {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestSecureWireRejectsOversizedClientFrameBeforePayload(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	go func() {
		var header [4]byte
		binary.BigEndian.PutUint32(header[:], uint32(maxClientWireFrame+1))
		_, _ = left.Write(header[:])
	}()
	var msg netWireMessage
	err := readNetMessage(right, maxClientWireFrame, &msg)
	if err == nil || !strings.Contains(err.Error(), "frame size") {
		t.Fatalf("expected frame-size rejection, got %v", err)
	}
}

func TestSanitizeRemotePlayerConfig(t *testing.T) {
	cfg := CustomPlayerConfig{
		Name:      "  Evil\x00Name-123456789012345678901234567890  ",
		Color:     4,
		Shirt:     3,
		Hat:       7,
		Eyes:      2,
		Gun:       1,
		Perk:      7,
		Throwable: ThrowableMk2,
		Type:      2,
		Team:      1,
	}
	got, err := sanitizeRemotePlayerConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != 1 {
		t.Fatalf("remote type=%d want human", got.Type)
	}
	if utf8.RuneCountInString(got.Name) > 24 {
		t.Fatalf("name was not truncated: %q", got.Name)
	}
	for _, r := range got.Name {
		if unicode.IsControl(r) {
			t.Fatalf("control character survived sanitization: %q", got.Name)
		}
	}

	cfg.Color = 999
	if _, err := sanitizeRemotePlayerConfig(cfg); err == nil {
		t.Fatal("invalid config should be rejected")
	}
}

func TestValidateHostRejectsForbiddenMessageType(t *testing.T) {
	err := validateNetMessageForHost(netWireMessage{Type: netMessageState, Protocol: netplayProtocolVersion})
	if err == nil {
		t.Fatal("client must not be allowed to submit authoritative state")
	}
}
