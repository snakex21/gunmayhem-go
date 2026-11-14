package game

import "time"

// takeInput returns the input for one host simulation tick (see
// netInputReplay): every client frame is applied once and in order.
func (p *netplayPeer) takeInput(now time.Time) PlayerInputState {
	if p == nil {
		return PlayerInputState{}
	}
	p.inputMu.Lock()
	defer p.inputMu.Unlock()
	return unpackNetInput(p.input.take(now))
}

// takeInputFor consumes one host tick of a remote player's input.
func (s *netplaySession) takeInputFor(playerID int) PlayerInputState {
	if s == nil || s.mode != netplayHost {
		return PlayerInputState{}
	}
	s.peersMu.RLock()
	peer := s.peers[playerID]
	s.peersMu.RUnlock()
	return peer.takeInput(time.Now())
}

// queueInput records one client frame of input. Call it exactly once per
// client update during gameplay: the host replays frames one per tick.
// It returns the frame's sequence number (used by client-side prediction).
func (s *netplaySession) queueInput(input PlayerInputState) (uint32, bool) {
	if s == nil || s.closed.Load() || s.mode != netplayClient || !s.connected() || s.outbox == nil {
		return 0, false
	}
	seq, frames := s.inputHistory.record(input)
	s.outbox.Push(netWireMessage{Type: netMessageInput, InputSeq: seq, InputFrames: frames})
	return seq, true
}

func capturePlayerInput(c Controls) PlayerInputState {
	return PlayerInputState{
		Left:    sourceKeyDown(c.Left),
		Right:   sourceKeyDown(c.Right),
		Up:      sourceKeyDown(c.Up),
		Down:    sourceKeyDown(c.Down),
		Shoot:   sourceKeyDown(c.Shoot),
		Grenade: sourceKeyDown(c.Grenade),
		Ability: sourceKeyDown(c.Ability),
	}
}

func (g *Game) applyHostRemoteInput() {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost {
		return
	}
	connected := [5]bool{}
	for _, id := range g.netplay.connectedPlayerIDs() {
		if id >= 2 && id <= maxOnlinePlayers {
			connected[id] = true
		}
	}
	for _, p := range g.players {
		if p == nil || p.ID < 2 || p.ID > maxOnlinePlayers || p.IsDouble {
			continue
		}
		cfg := g.customPlayers[p.ID-1]
		if cfg.Type == 2 {
			// AI bots are authoritative on the host. Never turn them into a
			// network-controlled player merely because they use slot IDs 2..4.
			p.RemoteControlled = false
			p.RemoteInput = PlayerInputState{}
			p.AI = true
			continue
		}
		if cfg.Type == 1 && connected[p.ID] {
			p.RemoteControlled = true
			p.RemoteInput = g.netplay.takeInputFor(p.ID)
			p.AI = false
			continue
		}
		// Waiting/empty online slots must not accidentally consume stale input.
		p.RemoteControlled = false
		p.RemoteInput = PlayerInputState{}
		p.AI = false
	}
}
