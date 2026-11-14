package game

import (
	"math"
	"sort"
)

// zombie_zones.go: step 2 of the ZOMBIE mode - arenas made of several zones
// (each zone is one existing map) linked by doors that players buy with
// points (docs/zombie_mode.md, "Zones and doors").
//
// Design: zone worlds. A zone keeps its own map, camera and entity lists
// (bullets, grenades, effects). The Game fields hold one zone at a time, the
// resident zone. A multi-zone tick (runZombieZoneTick) runs the ordinary
// gameplay step (simulateArenaTick) once per unlocked zone with that zone's
// lists, players and zombies swapped in. Every weapon, AI, physics and hit
// path therefore works unchanged inside a zone, and nothing crosses a zone
// border except through a door. Between ticks the view zone (the zone this
// screen shows) is resident, so Draw, render interpolation and the snapshot
// code always see exactly one zone.
//
// Doors: a door joins a spot of one zone to a spot of another. It is bought
// once (zombieCheckDoor is the one rule for local players and online guests)
// and then works both ways for players and zombies. On a shared local
// screen the whole team moves together: every living human must stand at
// the door (the gather prompt), then everybody - downed, dead and bots too -
// moves to the next zone. Online every player has an own view and passes
// alone. Zombies in a zone without players walk to the door toward the
// nearest player and come through.

const (
	zombieMaxDoors         = 8
	zombieMaxZones         = 8
	zombieDoorReachX       = 46.0
	zombieDoorReachY       = 30.0
	zombieDoorOpenFrames   = 36  // opening animation
	zombieGatherFrames     = 45  // everybody at the door: 1.5 s, then the team moves
	zombieGatherTimeout    = 600 // a gather nobody completes ends after 20 s
	zombieDoorArriveInvuln = 30
	zombieDoorZombieReach  = 18.0 // a zombie this close to its door goes through
	zombieDoorZombieSpread = 26.0 // zombies come out of a door spread by up to this
	zombieSpawnWeightHere  = 6    // spawn weight per living player in a zone
	zombieSpawnWeightNear  = 2    // spawn weight of an open neighbour zone
	zombieDoorPassCooldown = 45   // online: frames before the same player passes again
	zombieDoorInvulnWait   = 300  // online: arrival invulnerability at most this often
)

// ZombieDoorState is the synced state of one door (ZombieHUD.Doors).
type ZombieDoorState struct {
	Open     bool
	Anim     int // frames left of the opening animation
	BoughtBy int
}

// ZombieGatherState is the shared-screen "all players to the door" prompt.
type ZombieGatherState struct {
	Door     int // door index + 1 (0: no gather)
	At, Need int // living humans at the door / in total
	Progress int // frames everybody has been at the door
	Timer    int
}

// zombieZoneWorld holds the entity lists of a zone that is not resident.
// The resident zone's lists are the Game fields; worlds[resident] is then
// an empty shell (see zombieLoadZone).
type zombieZoneWorld struct {
	arena            Map
	cameraX, cameraY float64
	bullets          []Bullet
	instagibTrails   []InstagibTrailEffect
	playerTrails     []PlayerTrailEffect
	shells           []Shell
	flashes          []MuzzleFlash
	grenades         []Grenade
	firePatches      []FirePatch
	areaEffects      []AreaEffect
	portals          []Portal
	dynamiteFX       []DynamiteEffect
	blastFX          []BlastEffect
	jetThrustFX      []JetThrustEffect
	dropPackFX       []DropPackEffect
	explosions       []Explosion
	crates           []Crate
	powerups         []Powerup
	powerupNameFX    []PowerupNameEffect
	lifeBlingFX      []LifeBlingEffect
}

// swapZoneWorld exchanges the Game's world fields with w.
func (g *Game) swapZoneWorld(w *zombieZoneWorld) {
	g.arena, w.arena = w.arena, g.arena
	g.cameraX, w.cameraX = w.cameraX, g.cameraX
	g.cameraY, w.cameraY = w.cameraY, g.cameraY
	g.bullets, w.bullets = w.bullets, g.bullets
	g.instagibTrails, w.instagibTrails = w.instagibTrails, g.instagibTrails
	g.playerTrails, w.playerTrails = w.playerTrails, g.playerTrails
	g.shells, w.shells = w.shells, g.shells
	g.flashes, w.flashes = w.flashes, g.flashes
	g.grenades, w.grenades = w.grenades, g.grenades
	g.firePatches, w.firePatches = w.firePatches, g.firePatches
	g.areaEffects, w.areaEffects = w.areaEffects, g.areaEffects
	g.portals, w.portals = w.portals, g.portals
	g.dynamiteFX, w.dynamiteFX = w.dynamiteFX, g.dynamiteFX
	g.blastFX, w.blastFX = w.blastFX, g.blastFX
	g.jetThrustFX, w.jetThrustFX = w.jetThrustFX, g.jetThrustFX
	g.dropPackFX, w.dropPackFX = w.dropPackFX, g.dropPackFX
	g.explosions, w.explosions = w.explosions, g.explosions
	g.crates, w.crates = w.crates, g.crates
	g.powerups, w.powerups = w.powerups, g.powerups
	g.powerupNameFX, w.powerupNameFX = w.powerupNameFX, g.powerupNameFX
	g.lifeBlingFX, w.lifeBlingFX = w.lifeBlingFX, g.lifeBlingFX
}

// multiZone: the arena has more than one zone (step 2 rules apply).
func (m *zombieMatch) multiZone() bool {
	return m != nil && m.arena != nil && len(m.arena.Zones) > 1
}

// initZombieZones loads every zone's map and the zone worlds (host/local).
func (g *Game) initZombieZones() {
	m := g.zombie
	m.zoneMaps = make(map[int]Map, len(m.arena.Zones))
	m.worlds = make(map[int]*zombieZoneWorld, len(m.arena.Zones))
	for _, z := range m.arena.Zones {
		mp, ok := g.ensureMap(z.MapNumber)
		if !ok {
			mp, _ = g.ensureMap(1)
		}
		m.zoneMaps[z.ID] = mp
		m.worlds[z.ID] = &zombieZoneWorld{arena: mp}
	}
	m.resident = m.arena.StartZone
	m.viewZone = m.arena.StartZone
	m.passZone = -1
	m.refreshZoneInfo(g.players)
}

// zombieZoneMap is the map of a zone (the resident map when unknown).
func (g *Game) zombieZoneMap(zone int) Map {
	if m := g.zombie; m != nil {
		if mp, ok := m.zoneMaps[zone]; ok {
			return mp
		}
	}
	return g.arena
}

// zombieLoadZone makes zone z resident: the Game fields get its lists.
func (g *Game) zombieLoadZone(z int) {
	m := g.zombie
	if !m.multiZone() || z == m.resident {
		return
	}
	w, ok := m.worlds[z]
	shell, okShell := m.worlds[m.resident]
	if !ok || !okShell {
		return
	}
	g.swapZoneWorld(shell) // shell <- resident lists, Game <- empty
	g.swapZoneWorld(w)     // Game <- zone z, w <- empty (the new shell)
	m.resident = z
}

// zombieSyncView: online, the host's own player decides what the host
// screen shows; the view zone is made resident at once.
func (g *Game) zombieSyncView() {
	m := g.zombie
	if g.netplay != nil && g.netplay.mode == netplayHost {
		if zp := m.playerState(1); zp != nil && m.zoneUnlocked(zp.Zone) {
			m.viewZone = zp.Zone
		}
	}
	if m.passZone < 0 {
		g.zombieLoadZone(m.viewZone)
	}
}

// zombieViewZone is the zone this screen shows.
func (g *Game) zombieViewZone() int {
	m := g.zombie
	if m == nil {
		return 0
	}
	if !m.host {
		return m.hud.View
	}
	return m.viewZone
}

// playerZone is the zone a player (or its helper double) is in.
func (m *zombieMatch) playerZone(p *Player) int {
	if m == nil || p == nil {
		return 0
	}
	id := p.ID
	if p.IsDouble && p.OwnerPlayerID > 0 {
		id = p.OwnerPlayerID
	}
	if zp := m.playerState(id); zp != nil {
		return zp.Zone
	}
	if m.arena != nil {
		return m.viewZone
	}
	return 0
}

// doorEnd returns the side of door d that lies in zone, and the other side.
func (d *zombieDoorDef) end(zone int) (here zombiePoint, toZone int, there zombiePoint, ok bool) {
	switch zone {
	case d.Zone:
		return d.At, d.ToZone, d.To, true
	case d.ToZone:
		return d.To, d.Zone, d.At, true
	}
	return zombiePoint{}, 0, zombiePoint{}, false
}

func (m *zombieMatch) doorOpen(di int) bool {
	return di >= 0 && di < zombieMaxDoors && m.hud.Doors[di].Open
}

// zoneUnlocked: the start zone, or a zone behind an open door.
func (m *zombieMatch) zoneUnlocked(zone int) bool {
	if m == nil || m.arena == nil {
		return false
	}
	if zone == m.arena.StartZone || !m.multiZone() {
		return true
	}
	for di := range m.arena.Doors {
		d := &m.arena.Doors[di]
		if m.doorOpen(di) && (d.Zone == zone || d.ToZone == zone) {
			return true
		}
	}
	return false
}

// unlockedZones lists the unlocked zone IDs in ascending order.
func (m *zombieMatch) unlockedZones() []int {
	out := make([]int, 0, len(m.arena.Zones))
	for _, z := range m.arena.Zones {
		if m.zoneUnlocked(z.ID) {
			out = append(out, z.ID)
		}
	}
	sort.Ints(out)
	return out
}

// refreshZoneInfo counts the living players per zone and routes every zone
// without players to the door toward the nearest player (breadth-first over
// the open doors). Called once per tick before the zone passes.
func (m *zombieMatch) refreshZoneInfo(players []*Player) {
	if m == nil || m.arena == nil {
		return
	}
	m.zoneCount = [zombieMaxZones]int{}
	m.zonePresent = [zombieMaxZones]int{}
	for _, p := range players {
		if p == nil || p.IsDouble {
			continue
		}
		zp := m.playerState(p.ID)
		if zp == nil || zp.Zone < 0 || zp.Zone >= zombieMaxZones {
			continue
		}
		if zp.State != zombiePlayerDead {
			m.zonePresent[zp.Zone]++ // standing or downed: spawns come here
		}
		if p.Active && zp.State == zombiePlayerAlive {
			m.zoneCount[zp.Zone]++
		}
	}
	dist := [zombieMaxZones]int{}
	for i := range dist {
		dist[i] = -1
		m.zoneRoute[i] = -1
	}
	queue := make([]int, 0, zombieMaxZones)
	for z := 0; z < zombieMaxZones; z++ {
		if m.zoneCount[z] > 0 {
			dist[z] = 0
			queue = append(queue, z)
		}
	}
	for len(queue) > 0 {
		z := queue[0]
		queue = queue[1:]
		for di := range m.arena.Doors {
			if !m.doorOpen(di) {
				continue
			}
			d := &m.arena.Doors[di]
			_, other, _, ok := d.end(z)
			if !ok || other < 0 || other >= zombieMaxZones || dist[other] >= 0 {
				continue
			}
			dist[other] = dist[z] + 1
			m.zoneRoute[other] = di // from "other", this door leads toward z
			queue = append(queue, other)
		}
	}
}

// --- per tick ---------------------------------------------------------------------

// runZombieZoneTick runs the gameplay step once per unlocked zone of a
// multi-zone zombie match. It returns false for every other match (the
// caller then runs the single ordinary step).
func (g *Game) runZombieZoneTick() bool {
	m := g.zombie
	if !g.zombieSimulating() || !m.multiZone() {
		return false
	}
	all := g.players
	g.zombieSyncView()
	m.refreshZoneInfo(all)
	m.allPlayers = all
	if m.hud.Phase == zombiePhaseWave {
		g.spawnZombieWave()
	}
	zones := m.unlockedZones()
	// The view zone runs last: it is the primary pass (HUD, kill feed).
	sort.SliceStable(zones, func(i, j int) bool { return zones[j] == m.viewZone && zones[i] != m.viewZone })
	allZombies := m.zombies
	next := make([]*Zombie, 0, len(allZombies)+4)
	zonePlayers := make([]*Player, 0, len(all))
	// Bucket zombies and players by their zone BEFORE any pass: a zombie or
	// player that goes through a door during one zone's pass must not run
	// again in the next zone's pass of the same tick.
	zombieZone := make([]int, len(allZombies))
	for i, z := range allZombies {
		zombieZone[i] = z.Zone
	}
	playerZone := make([]int, len(all))
	for i, p := range all {
		playerZone[i] = m.playerZone(p)
	}
	zoneZombies := make([]*Zombie, 0, len(allZombies))
	for _, zone := range zones {
		g.zombieLoadZone(zone)
		zonePlayers = zonePlayers[:0]
		for i, p := range all {
			if p != nil && i < len(playerZone) && playerZone[i] == zone {
				zonePlayers = append(zonePlayers, p)
			}
		}
		zoneZombies = zoneZombies[:0]
		for i, z := range allZombies {
			if zombieZone[i] == zone {
				zoneZombies = append(zoneZombies, z)
			}
		}
		g.players = append([]*Player(nil), zonePlayers...)
		m.zombies = append([]*Zombie(nil), zoneZombies...)
		m.passZone = zone
		m.otherLiving = 0
		for i, z := range allZombies {
			if zombieZone[i] != zone && !z.Dead {
				m.otherLiving++
			}
		}
		primary := zone == m.viewZone
		g.sfxLocalMute = !primary
		g.sfxZoneTag = zone + 1
		g.simulateArenaTick(primary)
		g.updateZombiePlayers()
		next = append(next, m.zombies...)
		for _, p := range g.players {
			// A body added during the pass (none in zombie mode today)
			// must not be lost with the zone list.
			found := false
			for _, q := range all {
				if q == p {
					found = true
					break
				}
			}
			if !found {
				all = append(all, p)
			}
		}
	}
	for i, z := range allZombies {
		if !m.zoneUnlocked(zombieZone[i]) {
			next = append(next, z) // cannot happen; never drop a zombie
		}
	}
	g.sfxLocalMute = false
	g.sfxZoneTag = 0
	m.passZone = -1
	m.otherLiving = 0
	m.allPlayers = nil
	g.players = all
	m.zombies = next
	g.zombieLoadZone(m.viewZone)
	return true
}

// --- doors: players ----------------------------------------------------------------

// zombieSharedScreen: every player of the match is on this one screen
// (local play). Online each player has an own view.
func (g *Game) zombieSharedScreen() bool {
	return g.netplay == nil || g.netplay.mode != netplayHost
}

// doorInReach reports whether p stands at the side of door di in its zone.
func (m *zombieMatch) doorInReach(p *Player, di int) bool {
	if m == nil || p == nil || !p.Active || m.arena == nil || di < 0 || di >= len(m.arena.Doors) {
		return false
	}
	here, _, _, ok := m.arena.Doors[di].end(m.playerZone(p))
	if !ok {
		return false
	}
	return math.Abs(p.X-here.X) <= zombieDoorReachX && math.Abs(p.Y-here.Y) <= zombieDoorReachY
}

// doorNear is the door p stands at (-1 none).
func (m *zombieMatch) doorNear(p *Player) int {
	if !m.multiZone() {
		return -1
	}
	for di := range m.arena.Doors {
		if di < zombieMaxDoors && m.doorInReach(p, di) {
			return di
		}
	}
	return -1
}

// zombieCheckDoor is the one rule for buying a door, for local players and
// for online guests (the host runs it on the guest's player). It returns
// zombieMsgBought when p may buy door di now.
func (g *Game) zombieCheckDoor(p *Player, di int) int {
	m := g.zombie
	if m == nil || p == nil || p.IsDouble || !m.multiZone() || di < 0 || di >= len(m.arena.Doors) || di >= zombieMaxDoors {
		return zombieMsgDoorNoPath
	}
	zp := m.playerState(p.ID)
	if zp == nil {
		return zombieMsgDoorNoPath
	}
	if zp.State != zombiePlayerAlive || !p.Active {
		return zombieMsgDown
	}
	if m.hud.Phase == zombiePhaseOver || m.hud.Phase == zombiePhaseWon {
		return zombieMsgDoorNoPath
	}
	if !m.doorInReach(p, di) {
		return zombieMsgDoorNoPath
	}
	if m.hud.Doors[di].Open {
		return zombieMsgOwned
	}
	d := &m.arena.Doors[di]
	if d.OpenFrom > 0 && m.hud.Wave < d.OpenFrom {
		return zombieMsgDoorLocked
	}
	if zp.Points < d.Cost {
		return zombieMsgNoPoints
	}
	return zombieMsgBought
}

// zombieUseDoor: DOWN at a door. A closed door is bought; an open one is
// passed (alone online, as a team on a shared screen).
func (g *Game) zombieUseDoor(p *Player, di int) {
	m := g.zombie
	zp := m.playerState(p.ID)
	if zp == nil || di < 0 || di >= len(m.arena.Doors) || di >= zombieMaxDoors {
		return
	}
	st := &m.hud.Doors[di]
	if !st.Open {
		code := g.zombieCheckDoor(p, di)
		if code != zombieMsgBought {
			zp.Msg, zp.MsgTimer = code, 60
			return
		}
		d := &m.arena.Doors[di]
		here, _, _, _ := d.end(zp.Zone)
		zp.Points -= d.Cost
		m.addPopIn(zp.Zone, here.X, here.Y-90, -d.Cost, zombiePopSpend)
		st.Open = true
		st.Anim = zombieDoorOpenFrames
		st.BoughtBy = p.ID
		zp.Msg, zp.MsgTimer = zombieMsgDoorOpened, 60
		g.playSourceSFX("bolt2.wav", true)
		g.playRandomExplosionSound()
		return
	}
	if zp.State != zombiePlayerAlive || !p.Active || !m.doorInReach(p, di) {
		return
	}
	if g.zombieSharedScreen() {
		g.zombieStartGather(di)
		return
	}
	hp := m.hostPlayer(p.ID)
	if hp.doorCooldown > 0 {
		return // the arrival spot is at the same door: no door hopping
	}
	invuln := hp.doorInvulnWait == 0
	keep := zp.Invuln
	g.zombiePassPlayer(p, di)
	if !invuln {
		zp.Invuln = keep // a quick pass back does not protect again
	}
	hp.doorCooldown = zombieDoorPassCooldown
	hp.doorInvulnWait = zombieDoorInvulnWait
}

func (g *Game) zombieStartGather(di int) {
	gs := &g.zombie.hud.Gather
	if gs.Door != di+1 {
		*gs = ZombieGatherState{Door: di + 1}
	}
	gs.Timer = zombieGatherTimeout
}

// updateZombieGather runs the shared-screen gather: every living human at
// the door for zombieGatherFrames moves the whole team.
func (g *Game) updateZombieGather() {
	m := g.zombie
	gs := &m.hud.Gather
	if gs.Door == 0 {
		return
	}
	di := gs.Door - 1
	if !m.doorOpen(di) || m.hud.Phase == zombiePhaseOver || m.hud.Phase == zombiePhaseWon {
		*gs = ZombieGatherState{}
		return
	}
	need, at := 0, 0
	for _, p := range g.players {
		if p == nil || p.IsDouble || p.AI {
			continue
		}
		zp := m.playerState(p.ID)
		if zp == nil || zp.State != zombiePlayerAlive || !p.Active {
			continue
		}
		need++
		if m.doorInReach(p, di) {
			at++
		}
	}
	gs.At, gs.Need = at, need
	gs.Timer--
	if need == 0 || at == 0 || gs.Timer <= 0 {
		*gs = ZombieGatherState{}
		return
	}
	if at < need {
		gs.Progress = 0
		return
	}
	gs.Progress++
	if gs.Progress >= zombieGatherFrames {
		g.zombieMoveTeam(di)
	}
}

// zombieMoveTeam moves every player in the view zone (downed, dead and bots
// included) through door di; the shared camera follows.
func (g *Game) zombieMoveTeam(di int) {
	m := g.zombie
	from := m.viewZone
	_, toZone, there, ok := m.arena.Doors[di].end(from)
	m.hud.Gather = ZombieGatherState{}
	if !ok {
		return
	}
	movers := make([]*Player, 0, len(g.players))
	for _, p := range g.players {
		if p != nil && !p.IsDouble && m.playerZone(p) == from {
			movers = append(movers, p)
		}
	}
	for i, p := range movers {
		off := (float64(i) - float64(len(movers)-1)/2) * 22
		g.zombieMovePlayer(p, toZone, there.X+off, there.Y)
	}
	m.viewZone = toZone
	g.snapZoneCamera(toZone)
	g.zombieLoadZone(m.viewZone) // Draw shows the new zone right away
	g.playSourceSFX("emp.wav", false)
}

// snapZoneCamera puts a zone's camera straight onto its players (the source
// camera eases in 1/8 steps, which would pan in from the zone's old view).
func (g *Game) snapZoneCamera(zone int) {
	m := g.zombie
	resident := m.resident
	g.zombieLoadZone(zone)
	all := g.players
	vis := make([]*Player, 0, len(all))
	for _, p := range all {
		if p != nil && m.playerZone(p) == zone {
			vis = append(vis, p)
		}
	}
	g.players = vis
	for i := 0; i < 64; i++ {
		g.updateSourceCamera()
	}
	g.players = all
	g.zombieLoadZone(resident)
}

// zombiePassPlayer moves one online player through door di. Bots left in a
// zone without humans follow.
func (g *Game) zombiePassPlayer(p *Player, di int) {
	m := g.zombie
	from := m.playerZone(p)
	_, toZone, there, ok := m.arena.Doors[di].end(from)
	if !ok {
		return
	}
	alone := m.zoneCount[toZone] == 0
	g.zombieMovePlayer(p, toZone, there.X, there.Y)
	if alone {
		g.snapZoneCamera(toZone) // nobody there: its camera is stale
	}
	g.zombieSyncView()
	humans := false
	for _, o := range g.players {
		if o != nil && !o.IsDouble && !o.AI && m.playerZone(o) == from {
			humans = true
		}
	}
	if !humans {
		n := 0
		for _, o := range g.players {
			if o != nil && !o.IsDouble && o.AI && m.playerZone(o) == from {
				n++
				g.zombieMovePlayer(o, toZone, there.X+float64(n)*20, there.Y)
			}
		}
	}
}

// zombieMovePlayer puts a player (standing, downed or dead) at (x, y) of a
// zone. The x is kept on the platform under the door spot.
func (g *Game) zombieMovePlayer(p *Player, zone int, x, y float64) {
	m := g.zombie
	zp := m.playerState(p.ID)
	hp := m.hostPlayer(p.ID)
	if zp == nil || hp == nil {
		return
	}
	mp := g.zombieZoneMap(zone)
	for _, r := range mp.Platforms {
		if math.Abs(r.Y-y) <= 2 && x >= r.X-30 && x <= r.X+r.W+30 {
			x = math.Max(r.X+12, math.Min(x, r.X+r.W-12))
			break
		}
	}
	zp.Zone = zone
	p.X, p.Y, p.VX, p.VY = x, y, 0, 0
	p.FreePass = false
	hp.groundX, hp.groundY = x, y
	switch zp.State {
	case zombiePlayerAlive:
		p.JumpNum, p.Grounded = 2, true
		zp.Invuln = max(zp.Invuln, zombieDoorArriveInvuln)
	default:
		zp.DownX, zp.DownY = x, y
		zp.ReviveProgress = 0
		hp.reviverID = 0
	}
	for _, d := range g.players {
		if d != nil && d.IsDouble && d.OwnerPlayerID == p.ID {
			d.X, d.Y, d.VX, d.VY = x, y-20, 0, 0
		}
	}
}

// updateZombieDoors advances the opening animations and the gather.
func (g *Game) updateZombieDoors() {
	m := g.zombie
	if !m.multiZone() {
		return
	}
	for i := range m.hud.Doors {
		if m.hud.Doors[i].Anim > 0 {
			m.hud.Doors[i].Anim--
		}
	}
	if g.zombieSharedScreen() {
		g.updateZombieGather()
	} else {
		m.hud.Gather = ZombieGatherState{}
	}
}

// --- doors: zombies -----------------------------------------------------------------

// zombieDoorGoal: a zombie in a zone without living players walks to the
// door toward the nearest player. It returns the door and the spot.
func (g *Game) zombieDoorGoal(z *Zombie) (int, zombiePoint, bool) {
	m := g.zombie
	if !m.multiZone() || z.Zone < 0 || z.Zone >= zombieMaxZones || m.zoneCount[z.Zone] > 0 {
		return -1, zombiePoint{}, false
	}
	di := m.zoneRoute[z.Zone]
	if di < 0 || !m.doorOpen(di) {
		return -1, zombiePoint{}, false
	}
	here, _, _, ok := m.arena.Doors[di].end(z.Zone)
	return di, here, ok
}

// zombieDoorTarget is a stand-in target at a door spot for the steering.
func zombieDoorTarget(pt zombiePoint) Player {
	return Player{X: pt.X, Y: pt.Y, Active: true, PlayerScale: 0.8, JumpNum: 2, Grounded: true}
}

// zombieThroughDoor moves a zombie to the other side of door di.
func (g *Game) zombieThroughDoor(z *Zombie, di int) {
	m := g.zombie
	_, toZone, there, ok := m.arena.Doors[di].end(z.Zone)
	if !ok {
		return
	}
	b := &z.body
	x := there.X + (m.rng.Float64()*2-1)*zombieDoorZombieSpread
	mp := g.zombieZoneMap(toZone)
	for _, r := range mp.Platforms {
		if math.Abs(r.Y-there.Y) <= 2 && there.X >= r.X-2 && there.X <= r.X+r.W+2 {
			x = math.Max(r.X+10, math.Min(x, r.X+r.W-10))
			break
		}
	}
	b.X, b.Y, b.VX, b.VY = x, there.Y, 0, 0
	b.JumpNum, b.Grounded, b.FreePass = 2, true, false
	z.Zone = toZone
	z.nav = zombieNavState{cur: -1, lastPlat: -1}
	z.targetID, z.targetTimer = 0, 0
	z.stuckFrames = 0
	z.stuckBestDist = math.Inf(1)
	z.boxFrames = 0
	z.unstickTimer = 0
	z.DoorPasses++
	z.syncFromBody()
	m.doorPasses++
}

// --- spawning across zones -----------------------------------------------------------

// pickZombieSpawnZone weighs the unlocked zones toward the players: a zone
// with living players by their number, an open neighbour of such a zone a
// little. farthest (the boss) takes the zone with the most players.
func (m *zombieMatch) pickZombieSpawnZone(farthest bool) int {
	if !m.multiZone() {
		return m.arena.StartZone
	}
	zones := m.unlockedZones()
	weights := make([]int, len(zones))
	total := 0
	best, bestN := m.viewZone, 0
	for i, z := range zones {
		if z < 0 || z >= zombieMaxZones {
			continue
		}
		n := m.zonePresent[z]
		w := zombieSpawnWeightHere * n
		if n == 0 {
			for di := range m.arena.Doors {
				if !m.doorOpen(di) {
					continue
				}
				if _, other, _, ok := m.arena.Doors[di].end(z); ok && other >= 0 && other < zombieMaxZones && m.zonePresent[other] > 0 {
					w = zombieSpawnWeightNear
					break
				}
			}
		}
		weights[i] = w
		total += w
		if n > bestN {
			best, bestN = z, n
		}
	}
	if farthest || total == 0 {
		return best
	}
	x := m.rng.Intn(total)
	for i, w := range weights {
		if x < w {
			return zones[i]
		}
		x -= w
	}
	return best
}

// zombiePlayersAll is every player of the match, also inside a zone pass.
func (g *Game) zombiePlayersAll() []*Player {
	if m := g.zombie; m != nil && m.allPlayers != nil {
		return m.allPlayers
	}
	return g.players
}

// --- online views ----------------------------------------------------------------------

// zombieNetViews builds one snapshot per zone that a connected guest is in
// (multi-zone matches only). Every view carries all players (at most four;
// a guest draws only those in its zone) but only its own zone's zombies,
// effects and map, so a snapshot never grows with the number of zones.
func (g *Game) zombieNetViews() ([]netSnapshot, []int, map[int]int, bool) {
	m := g.zombie
	if !g.zombieSimulating() || !m.multiZone() || g.netplay == nil {
		return nil, nil, nil, false
	}
	ids := g.netplay.connectedPlayerIDs()
	zoneOf := func(id int) int {
		if zp := m.playerState(id); zp != nil && m.zoneUnlocked(zp.Zone) {
			return zp.Zone
		}
		return m.viewZone
	}
	zones := []int{}
	seen := map[int]int{}
	peerView := make(map[int]int, len(ids))
	add := func(z int) int {
		if i, ok := seen[z]; ok {
			return i
		}
		seen[z] = len(zones)
		zones = append(zones, z)
		return len(zones) - 1
	}
	add(m.viewZone)
	for _, id := range ids {
		peerView[id] = add(zoneOf(id))
	}
	states := make([]netSnapshot, 0, len(zones))
	for _, z := range zones {
		g.zombieLoadZone(z)
		s := g.makeNetSnapshot()
		g.zombieCullSnapshot(&s, z)
		states = append(states, s)
	}
	g.zombieLoadZone(m.viewZone)
	return states, zones, peerView, true
}

// zombieCullSnapshot keeps the zombies and popups of zone z and marks the
// view.
func (g *Game) zombieCullSnapshot(s *netSnapshot, z int) {
	zs := s.Zombies[:0]
	for _, zb := range s.Zombies {
		if zb.Zone == z {
			zs = append(zs, zb)
		}
	}
	s.Zombies = zs
	pops := s.ZombiePops[:0]
	for _, p := range s.ZombiePops {
		if p.Zone == z {
			pops = append(pops, p)
		}
	}
	s.ZombiePops = pops
	s.ZombieHUD.View = z
}

// zombieInView: an entity of zone z is drawn on this screen.
func (g *Game) zombieInView(zone int) bool {
	m := g.zombie
	if m == nil || !m.multiZone() {
		return true
	}
	return zone == g.zombieViewZone()
}

// beginZombieViewDraw hides the players of other zones from the world
// drawing; the returned function restores the list.
func (g *Game) beginZombieViewDraw() func() {
	m := g.zombie
	if !g.zombieActive() || !m.multiZone() {
		return func() {}
	}
	view := g.zombieViewZone()
	all := g.players
	vis := make([]*Player, 0, len(all))
	for _, p := range all {
		if p != nil && m.playerZone(p) == view {
			vis = append(vis, p)
		}
	}
	g.players = vis
	return func() { g.players = all }
}

// zombieNoDropAreas (online guest prediction): where DOWN uses the box or a
// door on the host instead of dropping through the platform.
func (g *Game) zombieNoDropAreas() []Rect {
	m := g.zombie
	if !g.zombieActive() || m.arena == nil {
		return nil
	}
	var out []Rect
	if b := m.hud.Box; (b.State == zombieBoxIdle || b.State == zombieBoxOffer) && g.zombieInView(b.Zone) && (b.X != 0 || b.Y != 0) {
		out = append(out, Rect{X: b.X - zombieBoxReachX, Y: b.Y - zombieBoxReachY, W: 2 * zombieBoxReachX, H: 2 * zombieBoxReachY})
	}
	if m.multiZone() {
		view := g.zombieViewZone()
		for di := range m.arena.Doors {
			if here, _, _, ok := m.arena.Doors[di].end(view); ok {
				out = append(out, Rect{X: here.X - zombieDoorReachX, Y: here.Y - zombieDoorReachY, W: 2 * zombieDoorReachX, H: 2 * zombieDoorReachY})
			}
		}
	}
	return out
}
