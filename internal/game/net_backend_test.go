package game

import (
	"errors"
	"net"
	"sync"
	"testing"
)

type markerNetplayCodec struct {
	base netplayCodec
}

func (c markerNetplayCodec) Encode(msg netWireMessage) ([]byte, error) {
	payload, err := c.base.Encode(msg)
	if err != nil {
		return nil, err
	}
	return append([]byte{0xa7}, payload...), nil
}

func (c markerNetplayCodec) Decode(payload []byte, msg *netWireMessage) error {
	if len(payload) == 0 || payload[0] != 0xa7 {
		return errors.New("missing test codec marker")
	}
	return c.base.Decode(payload[1:], msg)
}

func TestNetplayLinkUsesReplaceableCodec(t *testing.T) {
	leftConn, rightConn := net.Pipe()
	defer leftConn.Close()
	defer rightConn.Close()

	codec := markerNetplayCodec{base: cborNetplayCodec{}}
	left := (&netplayStack{codec: codec}).adopt(leftConn)
	right := (&netplayStack{codec: codec}).adopt(rightConn)
	want := netWireMessage{Type: netMessageInput, Protocol: netplayProtocolVersion, PlayerID: 2, Input: PlayerInputState{Left: true, Shoot: true}}

	errCh := make(chan error, 1)
	go func() { errCh <- left.Write(want, maxClientWireFrame) }()
	var got netWireMessage
	if err := right.Read(maxClientWireFrame, &got); err != nil {
		t.Fatal(err)
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	if got.Type != want.Type || got.PlayerID != want.PlayerID || got.Input != want.Input {
		t.Fatalf("decoded message=%+v want=%+v", got, want)
	}
}

func TestNetplayAddressNormalizationSupportsIPv4IPv6AndNames(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"join default", normalizeJoinAddress(""), "127.0.0.1:7777"},
		{"join IPv4", normalizeJoinAddress("192.168.1.25"), "192.168.1.25:7777"},
		{"join IPv6", normalizeJoinAddress("::1"), "[::1]:7777"},
		{"join name", normalizeJoinAddress("gaming-pc"), "gaming-pc:7777"},
		{"listen port", normalizeListenAddress("7788"), ":7788"},
		{"listen IPv6", normalizeListenAddress("::1"), "[::1]:7777"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("address=%q want %q", tt.got, tt.want)
			}
		})
	}
}

func TestNetplayOutboxDoesNotLetInputEvictControlState(t *testing.T) {
	outbox := newNetplayOutbox()
	wantConfig := CustomPlayerConfig{Name: "LAN Player", Type: 1}
	outbox.Push(netWireMessage{Type: netMessagePlayerConfig, Config: wantConfig})
	for i := 0; i < 100; i++ {
		outbox.Push(netWireMessage{Type: netMessageInput, Input: PlayerInputState{Left: i == 99}})
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	gotConfig := false
	gotLatestInput := false
	for i := 0; i < 2; i++ {
		msg, ok := outbox.Next(stop, done)
		if !ok {
			t.Fatal("outbox stopped before returning queued messages")
		}
		switch msg.Type {
		case netMessagePlayerConfig:
			gotConfig = msg.Config.Name == wantConfig.Name
		case netMessageInput:
			gotLatestInput = msg.Input.Left
		}
	}
	if !gotConfig || !gotLatestInput {
		t.Fatalf("outbox config=%v latestInput=%v", gotConfig, gotLatestInput)
	}
}

func TestQueueAndCloseClientSessionCanRunConcurrently(t *testing.T) {
	s := &netplaySession{
		mode:   netplayClient,
		done:   make(chan struct{}),
		outbox: newNetplayOutbox(),
	}
	s.ready.Store(true)
	s.assignedPlayerID.Store(2)

	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				s.queueInput(PlayerInputState{Left: i%2 == 0})
			}
		}()
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}
