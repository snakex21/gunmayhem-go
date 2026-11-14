package game

import (
	"math"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Tests of step 2: zones, doors, the shared-screen gather, independent
// zones online, zombie door pathing, per-zone snapshot views, bots that
// revive, and the server browser badge.

func outbreakArenaIndex(t *testing.T) int {
	t.Helper()
	for i := range zombieArenas {
		if zombieArenas[i].ID == "outbreak" {
			return i
		}
	}
	t.Fatal("no multi-zone arena")
	return 0
}

func newZoneTestGame(t *testing.T, slots [4]int) *Game {
	t.Helper()
	g := newZombieTestGame(t, outbreakArenaIndex(t), slots)
	if !g.zombie.multiZone() {
		t.Fatal("outbreak is not multi-zone")
	}
	return g
}

// openAllDoors opens every door of the arena (no points spent).
func openAllDoors(g *Game) {
	for di := range g.zombie.arena.Doors {
		g.zombie.hud.Doors[di].Open = true
	}
}

// putPlayerInZone moves a player to (x, y) of a zone and refreshes the
// per-zone counts.
func putPlayerInZone(g *Game, p *Player, zone int, x, y float64) {
	g.zombieMovePlayer(p, zone, x, y)
	if g.zombieSharedScreen() {
		g.zombie.viewZone = zone // the shared camera goes where player 1 is
	}
	g.zombieSyncView()
	g.zombie.refreshZoneInfo(g.players)
}

func pressDown(g *Game, p *Player, down bool) {
	p.RemoteControlled = true
	p.RemoteInput = PlayerInputState{Down: down}
}

// --- data and navigation ------------------------------------------------------------

func TestZombieZonesDataAndNavigation(t *testing.T) {
	a := &zombieArenas[outbreakArenaIndex(t)]
	if len(a.Zones) < 3 || len(a.Zones) > zombieMaxZones || len(a.Doors) > zombieMaxDoors {
		t.Fatalf("zones=%d doors=%d", len(a.Zones), len(a.Doors))
	}
	maps := map[int]bool{}
	for zi, z := range a.Zones {
		if z.ID != zi {
			t.Fatalf("zone %d has ID %d", zi, z.ID)
		}
		if maps[z.MapNumber] {
			t.Fatalf("map %d used by two zones (render interpolation keys on it)", z.MapNumber)
		}
		maps[z.MapNumber] = true
	}
	// Prices grow with depth (breadth-first from the start zone).
	depth := map[int]int{a.StartZone: 0}
	for changed := true; changed; {
		changed = false
		for _, d := range a.Doors {
			if dz, ok := depth[d.Zone]; ok {
				if _, seen := depth[d.ToZone]; !seen {
					depth[d.ToZone] = dz + 1
					changed = true
				}
			}
		}
	}
	if len(depth) != len(a.Zones) {
		t.Fatalf("zones not all reachable through doors: %v", depth)
	}
	for _, d := range a.Doors {
		for _, e := range a.Doors {
			if depth[e.Zone] > depth[d.Zone] && e.Cost <= d.Cost {
				t.Fatalf("door %s (%d) deeper than %s (%d) but not dearer", e.Name, e.Cost, d.Name, d.Cost)
			}
		}
	}
	for _, z := range a.Zones {
		mp, err := LoadOriginalMap(z.MapNumber)
		if err != nil {
			t.Fatal(err)
		}
		var spots []zombiePoint
		spots = append(spots, z.Spawns...)
		spots = append(spots, z.BoxSpots...)
		for _, d := range a.Doors {
			if here, _, _, ok := d.end(z.ID); ok {
				spots = append(spots, here)
			}
		}
		for class := zombieNavNormal; class <= zombieNavHeavy; class++ {
			nav := buildZombieNav(mp, class)
			for _, sp := range spots {
				if nav.platformUnder(sp.X, sp.Y) < 0 {
					t.Fatalf("%s: spot %v is not on a platform", z.Name, sp)
				}
			}
			for _, sp := range z.Spawns {
				from := nav.platformUnder(sp.X, sp.Y)
				for to := range nav.plats {
					if !nav.reachable(from, to) {
						t.Fatalf("%s class %d: platform %d unreachable from spawn %v", z.Name, class, to, sp)
					}
				}
			}
		}
	}
}

// --- doors: purchase rules ----------------------------------------------------------------

func TestZombieDoorPurchaseRulesLocal(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 1, 0, 0})
	m := g.zombie
	holdZombieWave(g)
	p := g.players[0]
	zp := m.playerState(1)
	d := m.arena.Doors[0]
	// Away from the door: nothing.
	placePlayer(p, 300, 364)
	if c := g.zombieCheckDoor(p, 0); c != zombieMsgDoorNoPath {
		t.Fatalf("away from the door: %d", c)
	}
	placePlayer(p, d.At.X, d.At.Y)
	zp.Points = d.Cost - 1
	if c := g.zombieCheckDoor(p, 0); c != zombieMsgNoPoints {
		t.Fatalf("without points: %d", c)
	}
	g.zombieUseDoor(p, 0)
	if m.hud.Doors[0].Open || zp.Points != d.Cost-1 || zp.Msg != zombieMsgNoPoints {
		t.Fatalf("bought without points: open=%v points=%d", m.hud.Doors[0].Open, zp.Points)
	}
	// A door far from the player's zone cannot be bought.
	zp.Points = 100000
	if c := g.zombieCheckDoor(p, 1); c != zombieMsgDoorNoPath {
		t.Fatalf("door of another zone: %d", c)
	}
	// A wave lock.
	m.arena.Doors[0].OpenFrom = m.hud.Wave + 1
	if c := g.zombieCheckDoor(p, 0); c != zombieMsgDoorLocked {
		t.Fatalf("wave lock: %d", c)
	}
	m.arena.Doors[0].OpenFrom = 0
	// Downed players cannot buy.
	zp.State = zombiePlayerDown
	if c := g.zombieCheckDoor(p, 0); c != zombieMsgDown {
		t.Fatalf("downed: %d", c)
	}
	zp.State = zombiePlayerAlive
	// Bought through DOWN at the door (updateZombiePreInput).
	zp.Points = d.Cost + 40
	pressDown(g, p, true)
	g.updateZombiePreInput()
	if !m.hud.Doors[0].Open || zp.Points != 40 || m.hud.Doors[0].BoughtBy != 1 || m.hud.Doors[0].Anim != zombieDoorOpenFrames {
		t.Fatalf("buy: open=%v points=%d by=%d", m.hud.Doors[0].Open, zp.Points, m.hud.Doors[0].BoughtBy)
	}
	if !p.DownHeld {
		t.Fatal("DOWN at a door must not drop through the platform")
	}
	// Holding DOWN does not buy or pass again; an open door is not sold twice.
	g.updateZombiePreInput()
	if c := g.zombieCheckDoor(p, 0); c != zombieMsgOwned || zp.Points != 40 {
		t.Fatalf("second buy: %d points=%d", c, zp.Points)
	}
	if !m.zoneUnlocked(1) || m.zoneUnlocked(2) {
		t.Fatal("unlock state wrong")
	}
	m.arena.Doors[0].OpenFrom = 0
}

// TestZombieDoorGuestValidatedByHost: an online guest buys and passes a door
// with DOWN; the host runs the same rule on the guest's player.
func TestZombieDoorGuestValidatedByHost(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 1, 0, 0})
	g.netplay = &netplaySession{mode: netplayHost}
	defer func() { g.netplay = nil }()
	m := g.zombie
	holdZombieWave(g)
	host, guest := g.players[0], g.players[1]
	zp := m.playerState(2)
	d := m.arena.Doors[0]
	placePlayer(host, 300, 364)
	placePlayer(guest, d.At.X, d.At.Y)
	zp.Points = d.Cost - 10
	pressDown(g, guest, true)
	g.updateZombiePreInput()
	if m.hud.Doors[0].Open || zp.Points != d.Cost-10 {
		t.Fatal("the host sold a door to a guest without points")
	}
	pressDown(g, guest, false)
	g.updateZombiePreInput()
	zp.Points = d.Cost
	pressDown(g, guest, true)
	g.updateZombiePreInput()
	if !m.hud.Doors[0].Open || zp.Points != 0 || m.hud.Doors[0].BoughtBy != 2 {
		t.Fatalf("guest purchase: open=%v points=%d", m.hud.Doors[0].Open, zp.Points)
	}
	// A guest far from a door cannot pass it (DOWN elsewhere drops through).
	pressDown(g, host, true)
	g.updateZombiePreInput()
	if m.playerState(1).Zone != 0 {
		t.Fatal("the host passed a door it does not stand at")
	}
	// Online: the guest passes alone, no gather.
	pressDown(g, guest, false)
	g.updateZombiePreInput()
	pressDown(g, guest, true)
	g.updateZombiePreInput()
	if zp.Zone != 1 || m.playerState(1).Zone != 0 || m.hud.Gather.Door != 0 {
		t.Fatalf("online pass: guest zone %d host zone %d gather %+v", zp.Zone, m.playerState(1).Zone, m.hud.Gather)
	}
	if math.Abs(guest.X-d.To.X) > 1 || guest.Y != d.To.Y || zp.Invuln < zombieDoorArriveInvuln-1 {
		t.Fatalf("guest arrived at %.0f,%.0f (door %v)", guest.X, guest.Y, d.To)
	}
	if m.viewZone != 0 || m.resident != 0 {
		t.Fatalf("the host screen followed the guest: view %d resident %d", m.viewZone, m.resident)
	}
}

// --- shared screen: gather -------------------------------------------------------------------

func TestZombieSharedScreenGatherMovesTheTeam(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 1, 1, 2})
	m := g.zombie
	holdZombieWave(g)
	openAllDoors(g)
	d := m.arena.Doors[0]
	p1, p2, p3, bot := g.players[0], g.players[1], g.players[2], g.players[3]
	placePlayer(p1, d.At.X, d.At.Y)
	placePlayer(p2, 300, 364)
	placePlayer(bot, 200, 198)
	// P3 bled out far away (DEAD, so the bot cannot revive it): it comes
	// along anyway.
	g.zombieDownPlayer(p3, m.playerState(3), m.hostPlayer(3), 120, 540, false)
	m.playerState(3).State = zombiePlayerDead
	pressDown(g, p1, true)
	updateN(t, g, 2)
	gs := m.hud.Gather
	if gs.Door != 1 || gs.Need != 2 || gs.At != 1 {
		t.Fatalf("gather after DOWN: %+v", gs)
	}
	for i := 0; i < zombieGatherFrames*3; i++ {
		placePlayer(p1, d.At.X, d.At.Y)
		placePlayer(p2, 300, 364)
		updateN(t, g, 1)
	}
	if m.playerState(1).Zone != 0 || m.hud.Gather.Progress != 0 {
		t.Fatal("the team moved before everybody stood at the door")
	}
	moved := false
	for i := 0; i < zombieGatherFrames+10 && !moved; i++ {
		placePlayer(p1, d.At.X-10, d.At.Y)
		placePlayer(p2, d.At.X+10, d.At.Y)
		updateN(t, g, 1)
		moved = m.playerState(1).Zone == 1
	}
	if !moved {
		t.Fatalf("the team never moved: %+v", m.hud.Gather)
	}
	for id := 1; id <= 4; id++ {
		if z := m.playerState(id).Zone; z != 1 {
			t.Fatalf("player %d stayed in zone %d", id, z)
		}
	}
	if zp3 := m.playerState(3); zp3.State != zombiePlayerDead || math.Abs(zp3.DownX-d.To.X) > 80 {
		t.Fatalf("downed player not carried: %+v", zp3)
	}
	if m.viewZone != 1 || m.resident != 1 || g.arena.Number != m.arena.Zones[1].MapNumber {
		t.Fatalf("camera/scene did not move: view %d resident %d map %d", m.viewZone, m.resident, g.arena.Number)
	}
	if m.hud.Gather.Door != 0 {
		t.Fatal("gather not cleared")
	}
}

func TestZombieGatherCancelsWhenNobodyIsAtTheDoor(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 1, 0, 0})
	m := g.zombie
	holdZombieWave(g)
	openAllDoors(g)
	d := m.arena.Doors[0]
	p1 := g.players[0]
	placePlayer(p1, d.At.X, d.At.Y)
	pressDown(g, p1, true)
	updateN(t, g, 1)
	if m.hud.Gather.Door == 0 {
		t.Fatal("no gather")
	}
	pressDown(g, p1, false)
	placePlayer(p1, 300, 364)
	updateN(t, g, 2)
	if m.hud.Gather.Door != 0 {
		t.Fatal("gather kept with nobody at the door")
	}
}

// --- online: independent zones ------------------------------------------------------------------

func TestZombieOnlineZonesAreIndependent(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 1, 0, 0})
	g.netplay = &netplaySession{mode: netplayHost, peers: map[int]*netplayPeer{2: {playerID: 2}}}
	defer func() { g.netplay = nil }()
	m := g.zombie
	holdZombieWave(g)
	openAllDoors(g)
	host, guest := g.players[0], g.players[1]
	putPlayerInZone(g, guest, 1, 450, 477)
	placePlayer(host, 450, 511)
	// A zombie right next to the guest in zone 1, and one at the same
	// coordinates in zone 0 far from the host.
	zGuest := g.newZombie(zombieWalker, 1000, zombiePoint{430, 477}, 1)
	zGuest.State = zombieStateWalk
	zFar := g.newZombie(zombieWalker, 1000, zombiePoint{430, 477}, 0)
	zFar.State = zombieStateWalk
	hostHP := m.playerState(1).HP
	guestHit := false
	for f := 0; f < 120 && !guestHit; f++ {
		placePlayer(host, 120, 540)
		placePlayer(guest, 450, 477)
		m.playerState(2).Invuln = 0
		updateN(t, g, 1)
		guestHit = m.playerState(2).HP < zombiePlayerHP
	}
	if !guestHit {
		t.Fatal("the zone-1 zombie never hit the guest")
	}
	if m.playerState(1).HP != hostHP {
		t.Fatal("the host was hurt by a zombie of another zone")
	}
	if zGuest.Zone != 1 || g.arena.Number != m.arena.Zones[0].MapNumber || m.resident != 0 {
		t.Fatalf("zone state: zombie %d resident %d map %d", zGuest.Zone, m.resident, g.arena.Number)
	}
	// A bullet of the host in zone 0 never touches the zone-1 zombie.
	hpBefore := zGuest.HP
	g.bullets = append(g.bullets, testBullet(zGuest.X-40, zGuest.Y-30, 1, 10))
	updateN(t, g, 4)
	if zGuest.HP != hpBefore {
		t.Fatal("a zone-0 bullet hit a zone-1 zombie")
	}

	// Snapshot views: the guest gets zone 1 only.
	states, zones, peerView, ok := g.zombieNetViews()
	if !ok || len(states) != 2 || zones[peerView[2]] != 1 {
		t.Fatalf("views: ok=%v n=%d zones=%v peerView=%v", ok, len(states), zones, peerView)
	}
	gv := states[peerView[2]]
	if gv.Arena.Number != m.arena.Zones[1].MapNumber || gv.ZombieHUD.View != 1 {
		t.Fatalf("guest view: map %d view %d", gv.Arena.Number, gv.ZombieHUD.View)
	}
	for _, z := range gv.Zombies {
		if z.Zone != 1 {
			t.Fatalf("guest view carries a zone-%d zombie", z.Zone)
		}
	}
	if len(gv.Players) != 2 {
		t.Fatal("views carry every player")
	}
	if m.resident != m.viewZone {
		t.Fatal("building the views left another zone resident")
	}
	// Through the wire: encode both views, the guest's peer stream picks its own.
	var enc netHostEncoder
	frame := enc.encodeViews(states, zones, peerView)
	payload, rawLen, err := newNetPeerStream().message(frame, 2)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := newNetClientStream().decode(payload, rawLen)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateNetSnapshot(dec.state); err != nil {
		t.Fatal(err)
	}
	if dec.state.Arena.Number != m.arena.Zones[1].MapNumber || dec.state.ZombieHUD.View != 1 {
		t.Fatalf("decoded guest view: map %d view %d", dec.state.Arena.Number, dec.state.ZombieHUD.View)
	}
	c := New()
	c.netplay = &netplaySession{mode: netplayClient}
	defer func() { c.netplay = nil }()
	c.applyNetSnapshot(dec.state)
	if c.zombieViewZone() != 1 || !c.zombieInView(1) || c.zombieInView(0) {
		t.Fatal("guest view zone")
	}
	// The guest draws its zone only (no host-side zone worlds exist there).
	end := c.beginZombieViewDraw()
	if len(c.players) != 1 || c.players[0].ID != 2 {
		t.Fatalf("guest draws %d players", len(c.players))
	}
	end()
	img := ebiten.NewImage(ScreenWidth, ScreenHeight)
	c.drawZombieWorld(img)
	c.drawZombieHUD(img)
	// DOWN at the door the guest stands at must not be predicted as a drop.
	found := false
	for _, r := range c.zombieNoDropAreas() {
		if r.Contains(m.arena.Doors[0].To.X, m.arena.Doors[0].To.Y) {
			found = true
		}
	}
	if !found {
		t.Fatal("no no-drop area at the guest's door")
	}
}

func TestZombieZoneSFXGoOnlyToThatZone(t *testing.T) {
	var enc netHostEncoder
	st := netSnapshot{SFXEvents: []netSFXEvent{{Seq: 1, Name: "all.wav"}, {Seq: 2, Name: "z1.wav", Zone: 2}, {Seq: 3, Name: "z0.wav", Zone: 1}}}
	frame := enc.encodeViews([]netSnapshot{st, {}}, []int{0, 1}, map[int]int{2: 1, 3: 0})
	names := func(id int) []string {
		ps := newNetPeerStream()
		ps.sfxSynced = true
		payload, n, err := ps.message(frame, id)
		if err != nil {
			t.Fatal(err)
		}
		dec, err := newNetClientStream().decode(payload, n)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, ev := range dec.sfx {
			out = append(out, ev.Name)
		}
		return out
	}
	if got := names(2); len(got) != 2 || got[0] != "all.wav" || got[1] != "z1.wav" {
		t.Fatalf("zone-1 peer got %v", got)
	}
	if got := names(3); len(got) != 2 || got[0] != "all.wav" || got[1] != "z0.wav" {
		t.Fatalf("zone-0 peer got %v", got)
	}
}

// --- zombies: door pathing, no stuck zombies -----------------------------------------------------

// TestZombieZonesNoStuckZombiesInsideEveryZone: every kind from every spawn
// point of every zone reaches a player on each platform of that zone, and
// none walks off the map. (Exploders are left out: their blasts knock the
// others off the map on purpose; they share the walkers' nav class.)
func TestZombieZonesNoStuckZombiesInsideEveryZone(t *testing.T) {
	kinds := []zombieKind{zombieWalker, zombieRunner, zombieBrute, zombieLeaper}
	g := newZoneTestGame(t, [4]int{1, 0, 0, 0})
	m := g.zombie
	openAllDoors(g)
	p := g.players[0]
	frames, runs := 0, 0
	for _, zone := range m.arena.Zones {
		putPlayerInZone(g, p, zone.ID, zone.Spawns[0].X, zone.Spawns[0].Y)
		nav := m.navFor(zone.ID, g.arena, zombieNavNormal)
		for pi, r := range nav.plats {
			m.zombies = nil
			m.stuckRespawns = 0
			holdZombieWave(g)
			px := r.X + r.W/2
			for _, sp := range zone.Spawns {
				for _, k := range kinds {
					z := g.newZombie(k, 1<<20, sp, zone.ID)
					z.State = zombieStateWalk
				}
			}
			reached := map[int]bool{}
			total := len(m.zombies)
			f := 0
			for ; f < 900 && len(reached) < total; f++ {
				zp := m.playerState(1)
				zp.Invuln, zp.HP = 1000, zombiePlayerHP
				placePlayer(p, px, r.Y)
				g.updateZombies(g.arena)
				for _, z := range m.zombies {
					if z.Dead && z.RingOut && !reached[z.ID] {
						t.Errorf("%s platform %d: %s fell off the map at %.0f", zone.Name, pi, zombieStats[z.kind()].Name, z.X)
						reached[z.ID] = true
					}
					if z.Dead || z.kind() == zombieExploder && z.State == zombieStateFuse || zombieInReach(z, p, 20) {
						reached[z.ID] = true
					}
				}
			}
			for _, z := range m.zombies {
				if !reached[z.ID] {
					t.Errorf("%s: player on platform %d (%.0f,%.0f): %s stuck at %.0f,%.0f", zone.Name, pi, px, r.Y, zombieStats[z.kind()].Name, z.X, z.Y)
				}
			}
			if m.stuckRespawns > 0 {
				t.Errorf("%s platform %d: %d anti-stuck re-spawns", zone.Name, pi, m.stuckRespawns)
			}
			frames += f
			runs++
		}
	}
	t.Logf("%d runs, %d frames simulated", runs, frames)
}

// TestZombieZonesDoorPathing: zombies of every other zone walk through the
// open doors (several in a row) to a player and reach them, without being
// re-spawned by the anti-stuck rule. Run with the player at both ends.
func TestZombieZonesDoorPathing(t *testing.T) {
	kinds := []zombieKind{zombieWalker, zombieRunner, zombieBrute, zombieLeaper}
	for _, playerZone := range []int{0, 3} {
		g := newZoneTestGame(t, [4]int{1, 0, 0, 0})
		m := g.zombie
		openAllDoors(g)
		holdZombieWave(g)
		p := g.players[0]
		zone := m.arena.zone(playerZone)
		spot := zone.BoxSpots[0]
		putPlayerInZone(g, p, playerZone, spot.X, spot.Y)
		for _, z := range m.arena.Zones {
			if z.ID == playerZone {
				continue
			}
			for si, sp := range z.Spawns {
				zb := g.newZombie(kinds[si%len(kinds)], 1<<20, sp, z.ID)
				zb.State = zombieStateWalk
			}
		}
		total := len(m.zombies)
		reached := map[int]bool{}
		f := 0
		for ; f < 3600 && len(reached) < total; f++ {
			zp := m.playerState(1)
			zp.Invuln, zp.HP = 1000, zombiePlayerHP
			placePlayer(p, spot.X, spot.Y)
			if !g.runZombieZoneTick() {
				t.Fatal("no zone tick")
			}
			ids := map[int]bool{}
			for _, z := range m.zombies {
				if ids[z.ID] || z.Dead {
					t.Fatalf("frame %d: %s %d duplicated or dead (zone %d at %.0f,%.0f ringout=%v)", f, zombieStats[z.kind()].Name, z.ID, z.Zone, z.X, z.Y, z.RingOut)
				}
				ids[z.ID] = true
			}
			if len(m.zombies) != total {
				t.Fatalf("frame %d: %d zombies, want %d", f, len(m.zombies), total)
			}
			for _, z := range m.zombies {
				if z.Zone == playerZone && zombieInReach(z, p, 20) {
					reached[z.ID] = true
				}
			}
		}
		for _, z := range m.zombies {
			if !reached[z.ID] {
				t.Errorf("player in %s: %s stuck in zone %d at %.0f,%.0f (doors passed %d)", zone.Name, zombieStats[z.kind()].Name, z.Zone, z.X, z.Y, z.DoorPasses)
			}
		}
		if m.stuckRespawns > 0 {
			t.Errorf("player in %s: %d anti-stuck re-spawns", zone.Name, m.stuckRespawns)
		}
		if len(reached) != total {
			t.Errorf("player in %s: %d of %d zombies arrived", zone.Name, len(reached), total)
		}
		if m.doorPasses < total {
			t.Errorf("player in %s: only %d door passes for %d zombies", zone.Name, m.doorPasses, total)
		}
		t.Logf("player in %s: %d zombies from the other zones arrived after %d frames, %d door passes", zone.Name, total, f, m.doorPasses)
	}
}

func TestZombieClosedDoorsBlockZombies(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 0, 0, 0})
	m := g.zombie
	holdZombieWave(g)
	m.hud.Doors[0].Open = true // zone 1 unlocked; door 1 (to zone 2) closed
	p := g.players[0]
	putPlayerInZone(g, p, 1, 450, 477)
	// A zombie that somehow is in zone 2 (behind a closed door) never routes.
	m.hud.Doors[1].Open = false
	z := g.newZombie(zombieWalker, 100, zombiePoint{450, 540}, 2)
	z.State = zombieStateWalk
	if _, _, ok := g.zombieDoorGoal(z); ok {
		t.Fatal("a zombie routed through a closed door")
	}
	// Zone 2 is locked: never simulated as a spawn zone.
	for i := 0; i < 200; i++ {
		if zone := m.pickZombieSpawnZone(false); zone != 0 && zone != 1 {
			t.Fatalf("spawned in locked zone %d", zone)
		}
	}
}

func TestZombieSpawnsWeightedTowardPlayers(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 0, 0, 0})
	m := g.zombie
	openAllDoors(g)
	p := g.players[0]
	putPlayerInZone(g, p, 1, 450, 477)
	counts := map[int]int{}
	for i := 0; i < 2000; i++ {
		counts[m.pickZombieSpawnZone(false)]++
	}
	if counts[1] < counts[0] || counts[1] < counts[2] || counts[3] != 0 {
		t.Fatalf("spawn zones %v (player in zone 1)", counts)
	}
	if counts[0] == 0 || counts[2] == 0 {
		t.Fatalf("open neighbour zones never spawn: %v", counts)
	}
	if m.pickZombieSpawnZone(true) != 1 {
		t.Fatal("the boss does not come to the players' zone")
	}
	pt, zone := g.pickZombieSpawn(false)
	if zone < 0 || zone > 2 {
		t.Fatalf("spawn %v in zone %d", pt, zone)
	}
}

func TestZombieBoxMovesAcrossUnlockedZones(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 0, 0, 0})
	m := g.zombie
	seen := map[int]bool{}
	for i := 0; i < 200; i++ {
		m.placeBox(m.hud.Box.Spot)
		seen[m.hud.Box.Zone] = true
	}
	if len(seen) != 1 || !seen[0] {
		t.Fatalf("box left the start zone with every door closed: %v", seen)
	}
	m.hud.Doors[0].Open = true
	for i := 0; i < 400; i++ {
		m.placeBox(m.hud.Box.Spot)
		seen[m.hud.Box.Zone] = true
	}
	if !seen[1] || seen[2] || seen[3] {
		t.Fatalf("box zones %v with door 0 open", seen)
	}
	// Only a player in the box's zone can use it.
	p := g.players[0]
	m.hud.Box = ZombieBoxState{X: 450, Y: 477, Zone: 1}
	placePlayer(p, 450, 477)
	if m.boxInReach(p) {
		t.Fatal("box of zone 1 usable from zone 0")
	}
	putPlayerInZone(g, p, 1, 450, 477)
	if !m.boxInReach(p) {
		t.Fatal("box not usable in its zone")
	}
}

// --- rendering ------------------------------------------------------------------------------------

func TestZombieZoneTransitionSnapsInterpolation(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 0, 0, 0})
	m := g.zombie
	holdZombieWave(g)
	openAllDoors(g)
	g.displaySmoothMotion = true
	p := g.players[0]
	placePlayer(p, 400, 364)
	g.captureRenderPrev(time.Now())
	g.zombieMoveTeam(0)
	if m.viewZone != 1 || m.resident != 1 {
		t.Fatal("no move")
	}
	restore := g.beginRenderInterpolation(time.Now())
	restore()
	if g.renderInterp.lastAlpha < 1 {
		t.Fatal("a zone change was interpolated (must snap)")
	}
	// Online guests: playout interpolation never blends two zones.
	a := netSnapshot{Players: []Player{{ID: 2, X: 100}}, Arena: Map{Number: 11}}
	b := netSnapshot{Players: []Player{{ID: 2, X: 110}}, Arena: Map{Number: 12}}
	b.ZombieHUD.View = 1
	if out := interpolateNetSnapshot(a, b, 0.5); out.Players[0].X != 100 {
		t.Fatal("playout blended across zones")
	}
	// Zombies that came through a door snap too.
	z := g.newZombie(zombieWalker, 100, zombiePoint{300, 477}, 1)
	z.State = zombieStateWalk
	prev := sampleZombie(z)
	z.DoorPasses++
	if renderPlayerContinues(prev, sampleZombie(z)) {
		t.Fatal("a door pass was interpolated")
	}
}

func TestZombieDrawMultiZoneHidesOtherZones(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 1, 0, 0})
	g.netplay = &netplaySession{mode: netplayHost}
	defer func() { g.netplay = nil }()
	m := g.zombie
	openAllDoors(g)
	putPlayerInZone(g, g.players[1], 2, 400, 450)
	end := g.beginZombieViewDraw()
	if len(g.players) != 1 || g.players[0].ID != 1 {
		t.Fatalf("view players %d", len(g.players))
	}
	end()
	if len(g.players) != 2 {
		t.Fatal("players not restored")
	}
	img := ebiten.NewImage(ScreenWidth, ScreenHeight)
	g.drawZombieWorld(img)
	g.drawZombieHUD(img)
	if m.viewZone != 0 {
		t.Fatal("view moved")
	}
}

// --- bots -------------------------------------------------------------------------------------------

func TestZombieBotRevivesDownedTeammate(t *testing.T) {
	for _, arena := range []int{0, 1} {
		g := newZombieTestGame(t, arena, [4]int{1, 2, 0, 0})
		m := g.zombie
		holdZombieWave(g)
		p1, bot := g.players[0], g.players[1]
		zp1 := m.playerState(1)
		zp1.SelfRevives = 0
		nav := m.navFor(0, g.arena, zombieNavNormal)
		// Down the human on the top platform, the bot starts at the bottom.
		top, bottom := nav.plats[0], nav.plats[0]
		for _, r := range nav.plats {
			if r.Y < top.Y {
				top = r
			}
			if r.Y > bottom.Y {
				bottom = r
			}
		}
		placePlayer(bot, bottom.X+bottom.W/2, bottom.Y)
		g.zombieDownPlayer(p1, zp1, m.hostPlayer(1), top.X+top.W/2, top.Y, false)
		revived := false
		for f := 0; f < 900 && !revived; f++ {
			updateN(t, g, 1)
			revived = zp1.State == zombiePlayerAlive
		}
		if !revived {
			t.Fatalf("arena %d: the bot never revived (bot at %.0f,%.0f, marker %.0f,%.0f)", arena, bot.X, bot.Y, zp1.DownX, zp1.DownY)
		}
		if m.playerState(2).Revives != 1 {
			t.Fatalf("arena %d: revive not credited to the bot", arena)
		}
	}
}

func TestZombieBotDoesNotReviveInACrowd(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 2, 0, 0})
	m := g.zombie
	holdZombieWave(g)
	p1, bot := g.players[0], g.players[1]
	zp1 := m.playerState(1)
	zp1.SelfRevives = 0
	g.zombieDownPlayer(p1, zp1, m.hostPlayer(1), 483, 151, false)
	placePlayer(bot, 300, 511)
	for i := 0; i < zombieBotDangerMax; i++ {
		spawnTestZombie(g, zombieWalker, 470+float64(i)*10, 151, 1000)
	}
	g.zombieBotAssist(bot)
	if m.hostPlayer(2).botReviveID != 0 {
		t.Fatal("the bot walked into a crowd")
	}
	m.zombies = nil
	g.zombieBotAssist(bot)
	if m.hostPlayer(2).botReviveID != 1 {
		t.Fatal("the bot does not go when it is safe")
	}
}

// --- browser ---------------------------------------------------------------------------------------

func TestServerBrowserMarksZombieSessions(t *testing.T) {
	g := New()
	g.onlineZombie = true
	g.zombieSetup = zombieSetupConfig{Mode: zombieModeEndless, Arena: outbreakArenaIndex(t)}
	zs := multiplayerServerInfo{Name: "Z", RuleSummary: g.onlineRuleSummary()}
	if !onlineServerIsZombie(zs) || onlineServerIsZombie(multiplayerServerInfo{RuleSummary: "DOUBLE KNOCKBACK"}) {
		t.Fatal("zombie session detection")
	}
	img := ebiten.NewImage(200, 40)
	drawZombieSessionBadge(img, Rect{X: 4, Y: 4, W: 82, H: 24}, 1)
	// The browser list draws the badge row without trouble.
	g.onlineZombie = false
	g.multiplayerServers = []multiplayerServerInfo{zs, {Name: "plain", RuleSummary: "NO MODIFIERS"}}
	g.multiplayerSelectedServer = 0
	screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
	g.drawOnlineCustomGame(screen)
}

// --- performance ---------------------------------------------------------------------------------------

func TestZombieMultiZoneTickPerformance(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 1, 0, 0})
	g.netplay = &netplaySession{mode: netplayHost}
	defer func() { g.netplay = nil }()
	m := g.zombie
	holdZombieWave(g)
	openAllDoors(g)
	putPlayerInZone(g, g.players[1], 2, 400, 450)
	for i := 0; i < zombieMaxActive; i++ {
		zone := i % len(m.arena.Zones)
		sp := m.arena.Zones[zone].Spawns[i%len(m.arena.Zones[zone].Spawns)]
		z := g.newZombie(zombieKind(1+i%5), 1<<20, sp, zone)
		z.State = zombieStateWalk
	}
	start := time.Now()
	const ticks = 90
	for i := 0; i < ticks; i++ {
		m.playerState(1).Invuln, m.playerState(2).Invuln = 1000, 1000
		g.runZombieZoneTick()
	}
	per := time.Since(start) / ticks
	t.Logf("%d zombies over %d zones: %v per tick", zombieMaxActive, len(m.arena.Zones), per)
	if per > 15*time.Millisecond {
		t.Fatalf("multi-zone tick too slow: %v", per)
	}
}

func TestZombieSingleZoneArenasKeepTheStepOneTick(t *testing.T) {
	for _, arena := range []int{0, 1} {
		g := newZombieTestGame(t, arena, [4]int{1, 0, 0, 0})
		if g.zombie.multiZone() || g.runZombieZoneTick() {
			t.Fatalf("arena %d runs zone passes", arena)
		}
		if g.zombie.doorNear(g.players[0]) != -1 {
			t.Fatal("doors on a single-zone arena")
		}
	}
}

func TestZombieSnapshotRejectsBadZones(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 0, 0, 0})
	s := g.makeNetSnapshot()
	if err := validateZombieSnapshot(s); err != nil {
		t.Fatal(err)
	}
	bad := s
	bad.ZombieHUD.View = 9
	if validateZombieSnapshot(bad) == nil {
		t.Fatal("view zone 9 accepted")
	}
	bad = s
	bad.ZombieHUD.Gather.Door = 50
	if validateZombieSnapshot(bad) == nil {
		t.Fatal("gather door 50 accepted")
	}
	bad = s
	bad.Zombies = []Zombie{{Kind: int(zombieWalker), Zone: -1}}
	if validateZombieSnapshot(bad) == nil {
		t.Fatal("zombie zone -1 accepted")
	}
}

// --- review fixes -------------------------------------------------------------------------------

func TestZombieZoneIDsAreIndices(t *testing.T) {
	for _, a := range zombieArenas {
		if len(a.Zones) > zombieMaxZones || len(a.Doors) > zombieMaxDoors {
			t.Fatalf("%s: %d zones, %d doors", a.Name, len(a.Zones), len(a.Doors))
		}
		for i, z := range a.Zones {
			if z.ID != i {
				t.Fatalf("%s: zone %d has ID %d (zone IDs index arrays and the wire)", a.Name, i, z.ID)
			}
		}
		for _, d := range a.Doors {
			if d.Zone < 0 || d.Zone >= len(a.Zones) || d.ToZone < 0 || d.ToZone >= len(a.Zones) || d.Zone == d.ToZone {
				t.Fatalf("%s: door %s joins zones %d and %d", a.Name, d.Name, d.Zone, d.ToZone)
			}
		}
	}
}

// A kill credited to a shooter who is in another zone (a grenade still
// flying after a door pass, a burn) still pays, and the popup shows where the
// zombie died.
func TestZombieCreditAcrossZones(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 1, 0, 0})
	g.netplay = &netplaySession{mode: netplayHost}
	defer func() { g.netplay = nil }()
	m := g.zombie
	holdZombieWave(g)
	openAllDoors(g)
	putPlayerInZone(g, g.players[1], 1, 450, 477)
	placePlayer(g.players[0], 120, 540)
	z := g.newZombie(zombieWalker, 30, zombiePoint{700, 540}, 0)
	z.State = zombieStateWalk
	z.creditID, z.creditTimer = 2, zombieCreditFrames
	z.bonusDamage = 1000
	before := m.playerState(2).Points
	g.runZombieZoneTick()
	if !z.Dead {
		t.Fatal("zombie survived")
	}
	zp := m.playerState(2)
	if zp.Points <= before || zp.Kills != 1 {
		t.Fatalf("cross-zone kill lost: points %d->%d kills %d", before, zp.Points, zp.Kills)
	}
	for _, p := range m.pops {
		if p.Zone != 0 {
			t.Fatalf("kill popup in zone %d, the zombie died in zone 0", p.Zone)
		}
	}
}

// Online, tapping DOWN at a door cannot hop back and forth for invulnerability.
func TestZombieOnlineDoorHoppingIsLimited(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 1, 0, 0})
	g.netplay = &netplaySession{mode: netplayHost}
	defer func() { g.netplay = nil }()
	m := g.zombie
	holdZombieWave(g)
	openAllDoors(g)
	guest := g.players[1]
	zp := m.playerState(2)
	d := m.arena.Doors[0]
	placePlayer(guest, d.At.X, d.At.Y)
	passes, last := 0, zp.Zone
	for f := 0; f < 120; f++ {
		pressDown(g, guest, f%2 == 0)
		g.updateZombiePreInput()
		if zp.Zone != last {
			passes++
			last = zp.Zone
			if passes > 1 && zp.Invuln > 1 {
				t.Fatalf("pass %d granted invulnerability again", passes)
			}
		}
		if zp.Invuln > 0 {
			zp.Invuln--
		}
	}
	if passes < 2 || passes > 3 {
		t.Fatalf("%d passes in 4 s of tapping (cooldown %d frames)", passes, zombieDoorPassCooldown)
	}
}

func TestZombieShopPopupInBuyersZone(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 1, 0, 0})
	g.netplay = &netplaySession{mode: netplayHost}
	defer func() { g.netplay = nil }()
	m := g.zombie
	openAllDoors(g)
	putPlayerInZone(g, g.players[1], 2, 400, 450)
	openZombieBreak(g)
	m.pops = nil
	g.zombieBuy(g.players[1], shopItemFor(t, zombieItemWeapon, 5).ID)
	if len(m.pops) != 1 || m.pops[0].Zone != 2 {
		t.Fatalf("spend popup %+v", m.pops)
	}
}

func TestZombieSpawnZoneWithOnlyDownedPlayers(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 0, 0, 0})
	m := g.zombie
	openAllDoors(g)
	p := g.players[0]
	putPlayerInZone(g, p, 2, 400, 450)
	g.zombieDownPlayer(p, m.playerState(1), m.hostPlayer(1), 400, 450, false)
	m.playerState(1).SelfRevives = 1
	m.refreshZoneInfo(g.players)
	for i := 0; i < 200; i++ {
		if z := m.pickZombieSpawnZone(false); z < 1 || z > 3 {
			t.Fatalf("spawn in zone %d while the player lies in zone 2", z)
		}
	}
	if m.pickZombieSpawnZone(true) != 2 {
		t.Fatal("boss not in the downed player's zone")
	}
}

func TestZombieBossSummonRespectsTheGlobalCap(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 0, 0, 0})
	m := g.zombie
	holdZombieWave(g)
	openAllDoors(g)
	for i := 0; i < zombieMaxActive-1; i++ {
		g.newZombie(zombieWalker, 100, m.arena.Zones[1].Spawns[0], 1).State = zombieStateWalk
	}
	boss := g.newZombie(zombieBoss, 1000, zombiePoint{420, 454}, 0)
	boss.State = zombieStateWalk
	m.passZone, m.otherLiving = 0, zombieMaxActive-1
	n := len(m.zombies)
	g.zombieBossSummon(boss, false)
	m.passZone, m.otherLiving = -1, 0
	if len(m.zombies) != n {
		t.Fatal("the boss summoned past the global cap")
	}
}

func TestZombieBotGivesUpWhenStuck(t *testing.T) {
	g := newZombieTestGame(t, 1, [4]int{1, 2, 0, 0})
	m := g.zombie
	holdZombieWave(g)
	bot := g.players[1]
	zp1 := m.playerState(1)
	zp1.SelfRevives = 0
	g.zombieDownPlayer(g.players[0], zp1, m.hostPlayer(1), 483, 151, false)
	hp := m.hostPlayer(2)
	for f := 0; f < zombieBotStallFrames+5; f++ {
		placePlayer(bot, 200, 511) // held in place: never gets closer
		g.zombieBotAssist(bot)
	}
	if hp.botGiveUpID != 1 || hp.botReviveID != 0 {
		t.Fatalf("bot did not give up: giveUp=%d revive=%d", hp.botGiveUpID, hp.botReviveID)
	}
	g.zombieBotAssist(bot)
	if hp.botReviveID != 0 {
		t.Fatal("bot went back at once")
	}
}

func TestZombieBoxNeverStaysOnItsSpot(t *testing.T) {
	g := newZoneTestGame(t, [4]int{1, 0, 0, 0})
	m := g.zombie
	for i := 0; i < 300; i++ {
		if i == 150 {
			m.hud.Doors[0].Open = true // the spot list grows
		}
		old := m.hud.Box
		m.placeBox(old.Spot)
		if b := m.hud.Box; b.Zone == old.Zone && b.X == old.X && b.Y == old.Y {
			t.Fatalf("box stayed at %v", b)
		}
	}
}
