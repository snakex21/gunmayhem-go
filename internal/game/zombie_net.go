package game

import (
	"errors"
	"sort"
)

// zombie_net.go: online co-op for the ZOMBIE mode (protocol 22).
//
// The host simulates everything: zombies, points, the shop and the box. A
// snapshot carries the compact Zombie list (alive + dying, capped in
// netSnapshotListCaps), the points popups and one ZombieHUD value; the
// zombies' physics bodies and AI state are unexported and never sent.
//
// A guest opens its own shop on its own screen. Its purchases travel as
// netZombieReq (message netMessageZombie): each message repeats every request
// the host has not acknowledged yet (ZombiePlayerState.ReqAck in the
// snapshot), so a lost or merged message never loses a purchase and a
// repeated one is never applied twice. The host checks every request with
// the same zombieCheckBuy as local purchases (points, break phase, caps).

const (
	netMessageZombie = 13 // client -> host, ZombieReqs

	netZombieReqBuy   = 1 // Item = shop item ID
	netZombieReqClose = 2 // close the shop: READY
	netZombieReqOpen  = 3 // reopen the shop during the break

	netZombieReqMax    = 8  // requests per message
	netZombieReqQueue  = 16 // host-side pending requests per peer
	netZombieItemLimit = 512
)

type netZombieReq struct {
	Seq    uint32 `cbor:",omitempty"`
	Action int    `cbor:",omitempty"`
	Item   int    `cbor:",omitempty"`
}

type netWireZombieMsg struct {
	Type       int            `cbor:",omitempty"`
	Protocol   int            `cbor:",omitempty"`
	PlayerID   int            `cbor:",omitempty"`
	ZombieReqs []netZombieReq `cbor:",omitempty"`
}

// validateNetZombieRequests is the structural check of a guest message
// (netwire_secure.go style); the game rules are checked on the game thread.
func validateNetZombieRequests(reqs []netZombieReq) error {
	if len(reqs) == 0 || len(reqs) > netZombieReqMax {
		return errors.New("invalid zombie request count")
	}
	for _, r := range reqs {
		if r.Seq == 0 {
			return errors.New("invalid zombie request sequence")
		}
		if r.Action < netZombieReqBuy || r.Action > netZombieReqOpen {
			return errors.New("invalid zombie request action")
		}
		if r.Item < 0 || r.Item >= netZombieItemLimit || (r.Action != netZombieReqBuy && r.Item != 0) {
			return errors.New("invalid zombie request item")
		}
	}
	return nil
}

// receiveZombieRequests stores a guest's requests (reader goroutine).
func (peer *netplayPeer) receiveZombieRequests(reqs []netZombieReq) {
	peer.zombieMu.Lock()
	defer peer.zombieMu.Unlock()
	// No per-connection "newest seen" filter: a new match restarts the
	// numbering from that match's ReqAck, and the game thread drops every
	// request at or below the player's ReqAck anyway. Only exact repeats
	// already waiting in the queue are skipped.
next:
	for _, r := range reqs {
		for _, q := range peer.zombieReqs {
			if q == r {
				continue next
			}
		}
		peer.zombieReqs = append(peer.zombieReqs, r)
	}
	if len(peer.zombieReqs) > netZombieReqQueue {
		peer.zombieReqs = append(peer.zombieReqs[:0], peer.zombieReqs[len(peer.zombieReqs)-netZombieReqQueue:]...)
	}
}

// takeZombieRequests drains every peer's requests (game thread).
func (s *netplaySession) takeZombieRequests() map[int][]netZombieReq {
	if s == nil || s.mode != netplayHost {
		return nil
	}
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()
	var out map[int][]netZombieReq
	for id, peer := range s.peers {
		peer.zombieMu.Lock()
		if len(peer.zombieReqs) > 0 {
			if out == nil {
				out = make(map[int][]netZombieReq)
			}
			out[id] = peer.zombieReqs
			peer.zombieReqs = nil
		}
		peer.zombieMu.Unlock()
	}
	return out
}

// processZombieRequests applies the guests' shop requests on the host.
func (g *Game) processZombieRequests() {
	if !g.zombieSimulating() || g.netplay == nil || g.netplay.mode != netplayHost {
		return
	}
	all := g.netplay.takeZombieRequests()
	ids := make([]int, 0, len(all))
	for id := range all {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		g.applyZombieRequests(id, all[id])
	}
}

// applyZombieRequests runs one guest's requests through the local rules.
func (g *Game) applyZombieRequests(playerID int, reqs []netZombieReq) {
	m := g.zombie
	zp := m.playerState(playerID)
	p := g.playerByID(playerID)
	if zp == nil || p == nil || p.IsDouble || p.AI {
		return
	}
	for _, r := range reqs {
		if r.Seq <= zp.ReqAck {
			continue
		}
		zp.ReqAck = r.Seq
		switch r.Action {
		case netZombieReqBuy:
			g.zombieBuy(p, r.Item)
		case netZombieReqClose:
			g.zombieSetShopOpen(p, false)
		case netZombieReqOpen:
			g.zombieSetShopOpen(p, true)
		}
	}
}

// --- guest side ----------------------------------------------------------------------

// queueZombieRequest records a shop action of this guest.
func (g *Game) queueZombieRequest(action, item int) {
	m := g.zombie
	if m == nil || m.host {
		return
	}
	m.clientReqSeq++
	m.clientReqs = append(m.clientReqs, netZombieReq{Seq: m.clientReqSeq, Action: action, Item: item})
	if len(m.clientReqs) > netZombieReqMax {
		m.clientReqs = append(m.clientReqs[:0], m.clientReqs[len(m.clientReqs)-netZombieReqMax:]...)
	}
	m.clientResend = 0
}

// sendZombieRequests (guest, every tick): drop acknowledged requests and
// (re)send the rest.
func (g *Game) sendZombieRequests() {
	m := g.zombie
	if m == nil || m.host || g.netplay == nil || g.netplay.mode != netplayClient {
		return
	}
	ack := uint32(0)
	if zp := m.playerState(g.onlineLocalPlayerID()); zp != nil {
		ack = zp.ReqAck
	}
	keep := m.clientReqs[:0]
	for _, r := range m.clientReqs {
		if r.Seq > ack {
			keep = append(keep, r)
		}
	}
	m.clientReqs = keep
	if len(m.clientReqs) == 0 {
		return
	}
	if m.clientResend > 0 {
		m.clientResend--
		return
	}
	m.clientResend = 6
	reqs := append([]netZombieReq(nil), m.clientReqs...)
	g.netplay.queueClientMessage(netWireMessage{Type: netMessageZombie, ZombieReqs: reqs})
}

// --- snapshot ------------------------------------------------------------------------

// zombieSnapshot lists dying zombies first and living ones last, so the cap
// (which keeps the end of a list) drops death animations before zombies.
func (g *Game) zombieSnapshot() []Zombie {
	if g.zombie == nil || len(g.zombie.zombies) == 0 {
		return nil
	}
	out := make([]Zombie, 0, len(g.zombie.zombies))
	for _, z := range g.zombie.zombies {
		if z.Dead {
			out = append(out, z.wire())
		}
	}
	for _, z := range g.zombie.zombies {
		if !z.Dead {
			out = append(out, z.wire())
		}
	}
	return out
}

// wire is the snapshot copy (host-only fields cleared).
func (z *Zombie) wire() Zombie {
	return Zombie{ID: z.ID, Kind: z.Kind, X: z.X, Y: z.Y, VX: z.VX, VY: z.VY, Facing: z.Facing, HP: z.HP, MaxHP: z.MaxHP,
		State: z.State, StateTimer: z.StateTimer, Anim: z.Anim, HitFlash: z.HitFlash, Airborne: z.Airborne,
		Dead: z.Dead, DeadFrames: z.DeadFrames, RingOut: z.RingOut, Zone: z.Zone, Level: z.Level}
}

func (g *Game) zombieHUDSnapshot() ZombieHUD {
	if g.zombie == nil {
		return ZombieHUD{}
	}
	return g.zombie.hud
}

func (g *Game) zombiePopSnapshot() []ZombiePop {
	if g.zombie == nil || len(g.zombie.pops) == 0 {
		return nil
	}
	return append([]ZombiePop(nil), g.zombie.pops...)
}

// applyZombieSnapshot mirrors the host's zombie state on a guest.
func (g *Game) applyZombieSnapshot(state *netSnapshot) {
	if !state.ZombieHUD.Active {
		if g.zombie != nil && !g.zombie.host {
			g.zombie = nil
		}
		return
	}
	m := g.zombie
	if m == nil || m.host {
		m = &zombieMatch{}
		g.zombie = m
		if zp := state.ZombieHUD.Players; g.netplay != nil {
			if id := g.onlineLocalPlayerID(); id >= 1 && id <= 4 {
				m.clientReqSeq = zp[id-1].ReqAck
			}
		}
	}
	m.hud = state.ZombieHUD
	m.arena = zombieArenaByIndex(m.hud.Arena)
	m.store = append(m.store[:0], state.Zombies...)
	m.zombies = m.zombies[:0]
	for i := range m.store {
		m.zombies = append(m.zombies, &m.store[i])
	}
	m.pops = append(m.pops[:0], state.ZombiePops...)
}

// validateZombieSnapshot bounds the zombie part of a host snapshot.
func validateZombieSnapshot(s netSnapshot) error {
	if len(s.Zombies) > 2*zombieSnapshotCap || len(s.ZombiePops) > 2*zombiePopCap {
		return errors.New("host snapshot zombie lists exceed safe limit")
	}
	h := s.ZombieHUD
	if !h.Active {
		return nil
	}
	if h.Mode < 0 || h.Mode >= zombieModeCount || h.Arena < 0 || h.Arena >= len(zombieArenas) {
		return errors.New("invalid zombie mode in host snapshot")
	}
	if h.Phase < zombiePhaseIntro || h.Phase > zombiePhaseWon || h.Wave < 0 || h.Wave > 100000 {
		return errors.New("invalid zombie wave in host snapshot")
	}
	for i, zp := range h.Players {
		if zp.ID != 0 && zp.ID != i+1 {
			return errors.New("invalid zombie player in host snapshot")
		}
		if zp.Gun != 0 {
			if _, ok := WeaponByNumber(zp.Gun); !ok {
				return errors.New("invalid zombie player weapon in host snapshot")
			}
		}
	}
	if h.Box.Weapon != 0 {
		if _, ok := WeaponByNumber(h.Box.Weapon); !ok {
			return errors.New("invalid mystery box weapon in host snapshot")
		}
	}
	for _, z := range s.Zombies {
		if z.Kind <= int(zombieKindNone) || z.Kind >= int(zombieKindCount) {
			return errors.New("invalid zombie kind in host snapshot")
		}
	}
	if s.ZombieSetupArena < 0 || s.ZombieSetupArena >= len(zombieArenas) || s.ZombieSetupMode < 0 || s.ZombieSetupMode >= zombieModeCount {
		return errors.New("invalid zombie setup in host snapshot")
	}
	return nil
}
