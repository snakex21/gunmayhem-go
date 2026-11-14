package game

import (
	"context"
	"encoding/binary"
	"errors"
	"net/netip"
)

// Test-only helpers for the network tests. Production code no longer needs
// these accessors and constructors.

func (p *netplayPeer) latestInput() PlayerInputState {
	if p == nil {
		return PlayerInputState{}
	}
	p.inputMu.Lock()
	defer p.inputMu.Unlock()
	return unpackNetInput(p.input.newest)
}

func (s *netplaySession) latestInputFor(playerID int) PlayerInputState {
	if s == nil || s.mode != netplayHost {
		return PlayerInputState{}
	}
	s.peersMu.RLock()
	peer := s.peers[playerID]
	s.peersMu.RUnlock()
	if peer == nil {
		return PlayerInputState{}
	}
	return peer.latestInput()
}

// latestState is the newest snapshot received (tests, diagnostics).
func (s *netplaySession) latestState() (netSnapshot, bool) {
	if s == nil || !s.hasState.Load() {
		return netSnapshot{}, false
	}
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state, true
}

func startP2PQUICHost(session *netplaySession, visibility int, sessionKey string) (*p2pQUICHost, error) {
	return startP2PQUICHostWithTunnel(session, visibility, sessionKey, nil)
}

func announceDHTPeer(ctx context.Context, infoHash [20]byte, port int) error {
	if port <= 0 || port > 65535 {
		return errors.New("invalid global DHT announce port")
	}
	client, err := newDHTLiteClient()
	if err != nil {
		return err
	}
	defer client.Close()
	return announceDHTPeerWithClient(ctx, client, infoHash, port, false)
}

// encodeShortOpenCode packs an OPEN server's IPv4 address and network port into
// ten Base32 characters. Unlike an invite code it deliberately carries no
// session secret, so it must never be used to authorize INVITE ONLY sessions.
func encodeShortOpenCode(addr netip.Addr, port uint16) (string, error) {
	addr = addr.Unmap()
	if !addr.Is4() || port == 0 {
		return "", errors.New("short server code requires IPv4 and a TCP port")
	}
	raw := make([]byte, 6)
	ip := addr.As4()
	copy(raw[:4], ip[:])
	binary.BigEndian.PutUint16(raw[4:], port)
	return shortOpenCodePrefix + inviteCodeEncoding.EncodeToString(raw), nil
}
