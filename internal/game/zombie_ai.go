package game

import (
	"math"
)

// zombie_ai.go: zombie spawning, platform pathfinding, movement and attacks.
//
// Navigation: every platform of the arena is a node. Links say how a body
// gets from one platform to another with the normal player physics:
//   - walk:   same height, touching
//   - hop:    same height or a little higher across a gap (jump at the edge)
//   - climb:  higher and overlapping (jump up through the one-way platform,
//             double jump at the top of the arc)
//   - drop:   lower and overlapping (DOWN, drop through)
//   - fall:   lower, past an edge (walk off and steer onto it)
// All-pairs next hops are precomputed once per map (a handful of platforms),
// so a zombie only looks up "which link next" from its platform toward the
// target's platform. The zombie presses the same AI keys a bot does and
// Player.UpdateWithFire moves it, so knockback, burning, tear gas and
// ring-outs work exactly like for players.

const (
	zombieLinkWalk = iota
	zombieLinkHop
	zombieLinkClimb
	zombieLinkDrop
	zombieLinkFall
)

type zombieNavLink struct {
	to   int
	kind int
	x    float64 // where on the from-platform the move starts
	dir  int     // direction to hold for hop / fall
	cost float64
}

type zombieNav struct {
	mapNumber int
	plats     []Rect
	links     [][]zombieNavLink
	next      [][]int // next[i][j]: index into links[i] toward j, -1 none
	dist      [][]float64
	lowestY   float64
}

// zombieNavState is a zombie's progress along its path (host only).
type zombieNavState struct {
	cur       int // platform the zombie stands on (-1 in the air / unknown)
	link      zombieNavLink
	hasLink   bool
	upPrev    bool
	jumped    bool
	repath    int
	lastPlat  int
	airFrames int
}

// zombieMaxRise is how high a zombie can reach with both jumps, with margin.
func zombieMaxRise(gravity float64) float64 {
	if gravity <= 0 {
		gravity = DefaultGravity
	}
	j := JumpPower * 0.9
	h1 := j * j / (2 * gravity)
	j2 := j * 0.83
	h2 := j2 * j2 / (2 * gravity)
	return h1 + h2 - 12
}

// Mobility classes: heavy zombies (brute, boss) are slow, so they cannot
// jump the long gaps the others can and get their own nav graph.
const (
	zombieNavNormal = iota
	zombieNavHeavy
)

func zombieNavClass(k zombieKind) int {
	if k == zombieBrute || k == zombieBoss {
		return zombieNavHeavy
	}
	return zombieNavNormal
}

func zombieMaxHop(class int) float64 {
	if class == zombieNavHeavy {
		return 60
	}
	return 110
}

// zombieWalkSpeed is the slowest top walking speed (px/frame) of a class:
// how far a body drifts while it falls off an edge.
func zombieWalkSpeed(class int) float64 {
	if class == zombieNavHeavy {
		return 2.8
	}
	return 3.6
}

func buildZombieNav(arena Map, class int) *zombieNav {
	n := len(arena.Platforms)
	nav := &zombieNav{mapNumber: arena.Number, plats: append([]Rect(nil), arena.Platforms...), lowestY: arena.LowestY}
	nav.links = make([][]zombieNavLink, n)
	maxRise := zombieMaxRise(arena.Gravity)
	maxHop := zombieMaxHop(class)
	speed := zombieWalkSpeed(class)
	gravity := arena.Gravity
	if gravity <= 0 {
		gravity = DefaultGravity
	}
	for i, a := range nav.plats {
		for j, b := range nav.plats {
			if i == j {
				continue
			}
			// Horizontal drift while falling from a to b (walked off an edge).
			drift := 0.0
			if dy := b.Y - a.Y; dy > 0 {
				drift = speed * math.Sqrt(2*dy/gravity) * 0.85
			}
			if l, ok := zombieLinkBetween(a, b, maxRise, maxHop, drift, arena.LowestY); ok {
				l.to = j
				nav.links[i] = append(nav.links[i], l)
			}
		}
	}
	// Floyd-Warshall over the few platforms of a map.
	nav.dist = make([][]float64, n)
	nav.next = make([][]int, n)
	for i := 0; i < n; i++ {
		nav.dist[i] = make([]float64, n)
		nav.next[i] = make([]int, n)
		for j := 0; j < n; j++ {
			nav.dist[i][j] = math.Inf(1)
			nav.next[i][j] = -1
		}
		nav.dist[i][i] = 0
		for li, l := range nav.links[i] {
			if l.cost < nav.dist[i][l.to] {
				nav.dist[i][l.to] = l.cost
				nav.next[i][l.to] = li
			}
		}
	}
	for k := 0; k < n; k++ {
		for i := 0; i < n; i++ {
			if math.IsInf(nav.dist[i][k], 1) {
				continue
			}
			for j := 0; j < n; j++ {
				if d := nav.dist[i][k] + nav.dist[k][j]; d < nav.dist[i][j] {
					nav.dist[i][j] = d
					nav.next[i][j] = nav.next[i][k]
				}
			}
		}
	}
	return nav
}

// zombieLinkBetween decides whether (and how) a body on a can reach b.
func zombieLinkBetween(a, b Rect, maxRise, maxHop, drift, lowestY float64) (zombieNavLink, bool) {
	ax0, ax1 := a.X, a.X+a.W
	bx0, bx1 := b.X, b.X+b.W
	ov0, ov1 := math.Max(ax0, bx0), math.Min(ax1, bx1)
	overlap := ov1 - ov0
	dy := b.Y - a.Y // > 0: b is lower
	acx := (ax0 + ax1) / 2
	edgeDir := 1
	edgeX := ax1
	gap := bx0 - ax1
	if bx1 <= ax0+1 || (bx0 < ax0 && bx1 < ax1) {
		edgeDir = -1
		edgeX = ax0
		gap = ax0 - bx1
	}
	cost := func(x float64) float64 { return 20 + math.Abs(x-acx) + math.Abs(dy)*1.3 }
	switch {
	case math.Abs(dy) <= 6:
		if gap <= 2 {
			x := (ov0 + ov1) / 2
			return zombieNavLink{kind: zombieLinkWalk, x: x, dir: edgeDir, cost: cost(x)}, true
		}
		if gap <= maxHop {
			x := edgeX - float64(edgeDir)*14
			return zombieNavLink{kind: zombieLinkHop, x: x, dir: edgeDir, cost: cost(x) + gap}, true
		}
	case dy < 0:
		rise := -dy
		if rise > maxRise {
			return zombieNavLink{}, false
		}
		if overlap >= 28 {
			x := (ov0 + ov1) / 2
			// Prefer a launch point close to the near end of the overlap, but
			// well inside both platforms.
			return zombieNavLink{kind: zombieLinkClimb, x: x, cost: cost(x) + rise*0.5}, true
		}
		if gap <= maxHop*0.75 && rise <= maxRise-25 {
			x := edgeX - float64(edgeDir)*12
			return zombieNavLink{kind: zombieLinkHop, x: x, dir: edgeDir, cost: cost(x) + gap + rise}, true
		}
	default:
		best := zombieNavLink{}
		found := false
		if overlap >= 28 && a.Y < lowestY-0.5 {
			x := (ov0 + ov1) / 2
			best = zombieNavLink{kind: zombieLinkDrop, x: x, cost: cost(x)}
			found = true
		}
		// Walk off the right / left edge when b continues past it and the
		// drift of the fall reaches it.
		// Too far for a plain fall: jump off the edge toward it (hop down).
		consider := func(l zombieNavLink) {
			if !found || l.cost < best.cost {
				best, found = l, true
			}
		}
		if bx1 >= ax1+40 {
			switch gapR := bx0 - ax1; {
			case gapR <= drift-8:
				consider(zombieNavLink{kind: zombieLinkFall, x: ax1 - 6, dir: 1, cost: cost(ax1) + 10})
			case gapR <= maxHop:
				consider(zombieNavLink{kind: zombieLinkHop, x: ax1 - 14, dir: 1, cost: cost(ax1) + 30 + gapR})
			}
		}
		if bx0 <= ax0-40 {
			switch gapL := ax0 - bx1; {
			case gapL <= drift-8:
				consider(zombieNavLink{kind: zombieLinkFall, x: ax0 + 6, dir: -1, cost: cost(ax0) + 10})
			case gapL <= maxHop:
				consider(zombieNavLink{kind: zombieLinkHop, x: ax0 + 14, dir: -1, cost: cost(ax0) + 30 + gapL})
			}
		}
		return best, found
	}
	return zombieNavLink{}, false
}

// platformUnder returns the platform a standing body is on (-1 none).
func (nav *zombieNav) platformUnder(x, y float64) int {
	best := -1
	for i, r := range nav.plats {
		if x < r.X-2 || x > r.X+r.W+2 {
			continue
		}
		if math.Abs(r.Y-y) <= 4 {
			return i
		}
	}
	return best
}

// platformBelow is the highest platform at or below y at x (-1 none).
func (nav *zombieNav) platformBelow(x, y float64) int {
	best, bestY := -1, math.Inf(1)
	for i, r := range nav.plats {
		if x < r.X || x > r.X+r.W || r.Y < y-4 {
			continue
		}
		if r.Y < bestY {
			best, bestY = i, r.Y
		}
	}
	return best
}

// platformNear: the platform under a target, or below it, or the nearest
// one horizontally (a player in mid-air beside the map).
func (nav *zombieNav) platformNear(x, y float64) int {
	if i := nav.platformUnder(x, y); i >= 0 {
		return i
	}
	if i := nav.platformBelow(x, y); i >= 0 {
		return i
	}
	best, bestD := -1, math.Inf(1)
	for i, r := range nav.plats {
		cx := math.Max(r.X, math.Min(x, r.X+r.W))
		d := math.Abs(cx-x) + math.Abs(r.Y-y)*0.5
		if d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

// reachable reports whether a path exists from platform a to b.
func (nav *zombieNav) reachable(a, b int) bool {
	if a < 0 || b < 0 || a >= len(nav.plats) || b >= len(nav.plats) {
		return false
	}
	return !math.IsInf(nav.dist[a][b], 1)
}

// --- spawning ----------------------------------------------------------------------

func (g *Game) newZombie(kind zombieKind, hp int, pt zombiePoint, zone int) *Zombie {
	m := g.zombie
	m.nextID++
	st := kind.stats()
	scale := st.Scale
	if scale <= 0 {
		scale = 1
	}
	speed := m.plan.SpeedMul
	if speed <= 0 {
		speed = 1
	}
	facing := 1
	if mp := g.zombieZoneMap(zone); pt.X > (mp.SpawnMinX+mp.SpawnMaxX)/2 {
		facing = -1
	}
	z := &Zombie{ID: m.nextID, Kind: int(kind), X: pt.X, Y: pt.Y, Facing: facing, HP: hp, MaxHP: hp,
		State: zombieStateRising, StateTimer: zombieRiseFrames, Zone: zone, lastX: pt.X}
	z.body = Player{
		ID: z.ID, Name: st.Name, Team: zombieTeam, AI: true, Active: true, Facing: facing,
		Lives: 1, TotalLives: 1, GameMode: SourceGameModeNormal, PerkNumber: 7, Weight: 1,
		DamageMulti: 1, FirepowerMulti: 1, SpeedTime: -1, MiniTime: -1, MiniMulti: 1,
		PlayerScale: 0.8 * scale, Alpha: 1, ShieldAlpha: 1, CheapTimer: 120,
		KnockbackScale: st.Knockback, CampaignMoveScale: st.Move * speed, RuleJumpMultiplier: st.Jump,
		X: pt.X, Y: pt.Y, JumpNum: 2, Grounded: true,
	}
	z.nav.cur = -1
	z.nav.lastPlat = -1
	z.stuckBestDist = math.Inf(1)
	z.targetTimer = int(m.nextID % zombieRetargetFrames)
	z.slamTimer = zombieBossSlamEvery / 2
	z.summonTimer = zombieBossSummonEvery
	m.zombies = append(m.zombies, z)
	return z
}

// pickZombieSpawn chooses a zone (multi-zone: weighted toward the players,
// pickZombieSpawnZone) and a spawn point of it away from the players.
func (g *Game) pickZombieSpawn(farthest bool) (zombiePoint, int) {
	m := g.zombie
	zone := m.arena.zone(m.pickZombieSpawnZone(farthest))
	if zone == nil || len(zone.Spawns) == 0 {
		mp := g.arena
		id := 0
		if zone != nil {
			mp, id = g.zombieZoneMap(zone.ID), zone.ID
		}
		x := (mp.SpawnMinX + mp.SpawnMaxX) / 2
		y, _ := zombieFloorBelow(mp, x, -1000)
		return zombiePoint{x, y}, id
	}
	type cand struct {
		pt   zombiePoint
		dist float64
	}
	cands := make([]cand, 0, len(zone.Spawns))
	for _, sp := range zone.Spawns {
		d := math.Inf(1)
		for _, p := range g.zombiePlayersAll() {
			if p == nil || !p.Active || m.playerZone(p) != zone.ID {
				continue
			}
			d = math.Min(d, math.Hypot(p.X-sp.X, p.Y-sp.Y))
		}
		cands = append(cands, cand{sp, d})
	}
	if farthest {
		best := cands[0]
		for _, c := range cands[1:] {
			if c.dist > best.dist {
				best = c
			}
		}
		return best.pt, zone.ID
	}
	ok := cands[:0:0]
	for _, c := range cands {
		if c.dist >= zombieSpawnMinPlayerGap {
			ok = append(ok, c)
		}
	}
	if len(ok) == 0 {
		ok = cands
	}
	return ok[m.rng.Intn(len(ok))].pt, zone.ID
}

func (g *Game) spawnZombieBoss() {
	m := g.zombie
	pt, zone := g.pickZombieSpawn(true)
	z := g.newZombie(zombieBoss, m.plan.BossHP, pt, zone)
	z.Level = m.plan.BossLevel
	z.StateTimer = zombieRiseFrames * 2
	m.hud.BannerFrame = 1
	g.playSourceSFX("emp.wav", true)
}

func (g *Game) spawnZombieWave() {
	m := g.zombie
	if m.hud.Phase != zombiePhaseWave || m.toSpawn <= 0 {
		return
	}
	if m.spawnTimer > 0 {
		m.spawnTimer--
		return
	}
	if m.livingZombies() >= m.plan.MaxActive {
		return
	}
	kind := m.plan.pick(m.rng.Float64())
	pt, zone := g.pickZombieSpawn(false)
	g.newZombie(kind, zombieScaledHP(kind, m.plan), pt, zone)
	m.toSpawn--
	m.spawnTimer = m.plan.Interval
}

// --- per-tick update ------------------------------------------------------------------

// updateZombies moves every zombie, resolves melee and spawns the wave. It is
// called after the player loop inside the players' knockback window.
func (g *Game) updateZombies(arena Map) {
	if !g.zombieSimulating() {
		return
	}
	m := g.zombie
	if m.hud.Phase == zombiePhaseWave && m.passZone < 0 {
		// Multi-zone matches spawn once per tick before the zone passes.
		g.spawnZombieWave()
	}
	for i := 0; i < len(m.zombies); i++ {
		z := m.zombies[i]
		if z.Dead {
			z.DeadFrames++
			continue
		}
		if m.passZone >= 0 && z.Zone != m.passZone {
			continue // came through a door during this pass
		}
		g.updateZombie(z, arena, m.navFor(z.Zone, arena, zombieNavClass(z.kind())))
	}
	out := m.zombies[:0]
	for _, z := range m.zombies {
		if z.Dead && z.DeadFrames > zombieDeathFrames {
			continue
		}
		out = append(out, z)
	}
	for i := len(out); i < len(m.zombies); i++ {
		m.zombies[i] = nil
	}
	m.zombies = out
}

func (g *Game) updateZombie(z *Zombie, arena Map, nav *zombieNav) {
	b := &z.body
	z.Anim++
	if z.HitFlash > 0 {
		z.HitFlash--
	}
	if z.cooldown > 0 {
		z.cooldown--
	}
	if z.creditTimer > 0 {
		z.creditTimer--
		if z.creditTimer == 0 {
			z.creditID = 0
		}
	}
	if z.State == zombieStateRising {
		z.StateTimer--
		if z.StateTimer <= 0 {
			z.State = zombieStateWalk
			z.StateTimer = 0
		}
		z.syncFromBody()
		return
	}
	b.clearAIInput()
	target := g.zombieTarget(z)
	st := z.kind().stats()
	wasAirborne := b.JumpNum != 2

	var doorTarget Player
	steerTarget := target
	switch z.State {
	case zombieStateWalk:
		if di, goal, ok := g.zombieDoorGoal(z); ok && target == nil && b.BlindTime <= 0 {
			// No player in this zone: walk to the door toward the nearest
			// one and come through.
			if b.JumpNum == 2 && math.Abs(b.X-goal.X) <= zombieDoorZombieReach && math.Abs(b.Y-goal.Y) <= 8 {
				g.zombieThroughDoor(z, di)
				return
			}
			doorTarget = zombieDoorTarget(goal)
			steerTarget = &doorTarget
			g.zombieSteer(z, steerTarget, nav)
			break
		}
		if target != nil && b.BlindTime <= 0 {
			if z.kind() == zombieExploder && zombieDist(b, target) < zombieExploderTrigger*b.PlayerScale/0.8 {
				z.State = zombieStateFuse
				z.StateTimer = zombieExploderFuse
				g.playSourceSFX("bolt1.wav", false)
				break
			}
			if z.kind() == zombieLeaper && z.cooldown == 0 && !wasAirborne && g.zombieTryLeap(z, target, nav) {
				break
			}
			if z.kind() == zombieBoss && g.zombieBossSpecial(z, target, arena, wasAirborne) {
				break
			}
			if st.Damage > 0 && z.cooldown == 0 && zombieInReach(z, target, st.Reach) {
				z.State = zombieStateAttack
				z.StateTimer = st.Windup
				if target.X < b.X {
					b.Facing = -1
				} else {
					b.Facing = 1
				}
				break
			}
			g.zombieSteer(z, target, nav)
		} else {
			g.zombieWander(z, nav)
		}
	case zombieStateAttack:
		z.StateTimer--
		if z.StateTimer <= 0 {
			if target != nil && zombieInReach(z, target, st.Reach+10) {
				dir := float64(b.Facing)
				g.zombieHurtPlayer(target, st.Damage, st.Push*dir, st.Lift, b.ID)
			}
			z.cooldown = st.Cooldown
			z.State = zombieStateWalk
		}
	case zombieStateFuse:
		z.StateTimer--
		if z.StateTimer <= 0 {
			z.chained = false
			g.zombieKill(z, 0, false, false)
			return
		}
	case zombieStateLeap:
		if !z.leapHit && target != nil && rectsOverlap(b.Hitbox(), target.Hitbox()) {
			z.leapHit = true
			g.zombieHurtPlayer(target, st.Damage, float64(b.Facing)*st.Push, st.Lift, b.ID)
		}
	case zombieStateSlam:
	}

	b.UpdateWithFire(arena, nil, nil, nil)
	if !b.Active {
		credit := 0
		if z.creditTimer > 0 {
			credit = z.creditID
		}
		z.X, z.Y = b.LastDeathX, b.LastDeathY
		b.X, b.Y = z.X, z.Y
		g.zombieKill(z, credit, true, false)
		return
	}
	landed := wasAirborne && b.JumpNum == 2
	switch z.State {
	case zombieStateLeap:
		z.StateTimer++
		if landed || z.StateTimer > 60 {
			z.State = zombieStateWalk
			z.cooldown = zombieLeapCooldown
		}
	case zombieStateSlam:
		z.StateTimer++
		if (landed && z.StateTimer > 6) || z.StateTimer > 90 {
			g.zombieBossSlam(z)
		}
	}
	g.zombieTrackStuck(z, steerTarget, nav)
	z.syncFromBody()
}

func (z *Zombie) syncFromBody() {
	b := &z.body
	z.X, z.Y = b.X, b.Y
	z.VX, z.VY = b.VX, b.VY
	z.Facing = b.Facing
	z.Airborne = b.JumpNum != 2
}

func zombieDist(a, b *Player) float64 {
	return math.Hypot(a.X-b.X, (a.Y-30*a.PlayerScale/0.8)-(b.Y-30))
}

// zombieInReach: the target's hitbox touches the zombie's hitbox extended
// by reach toward the target.
func zombieInReach(z *Zombie, t *Player, reach float64) bool {
	h := z.body.Hitbox()
	r := Rect{X: h.X - reach, Y: h.Y + 4, W: h.W + 2*reach, H: h.H - 4}
	return rectsOverlap(r, t.Hitbox())
}

// zombieTarget: nearest living, visible player; re-evaluated every
// zombieRetargetFrames.
func (g *Game) zombieTarget(z *Zombie) *Player {
	b := &z.body
	if z.targetTimer > 0 && z.targetID != 0 {
		z.targetTimer--
		if p := g.playerByID(z.targetID); p != nil && p.Active && g.aiCanDetect(b, p) {
			return p
		}
	}
	z.targetTimer = zombieRetargetFrames
	z.targetID = 0
	var best *Player
	bestD := math.Inf(1)
	for _, p := range g.players {
		if p == nil || !p.Active || p.Team == zombieTeam {
			continue
		}
		if !g.aiCanDetect(b, p) {
			continue
		}
		d := math.Abs(p.X-b.X) + math.Abs(p.Y-b.Y)*1.5
		if d < bestD {
			best, bestD = p, d
		}
	}
	if best != nil {
		z.targetID = best.ID
	}
	return best
}

// zombieSteer presses the AI keys that move the zombie along its path.
func (g *Game) zombieSteer(z *Zombie, target *Player, nav *zombieNav) {
	b := &z.body
	if z.unstickTimer > 0 {
		z.unstickTimer--
		zombieHold(b, z.unstickDir)
		if z.unstickTimer%12 == 11 {
			zombiePressUp(z)
		} else {
			z.nav.upPrev = false
		}
		return
	}
	navSteer(b, &z.nav, z.Anim, target.X, target.Y, nav)
}

// navSteer presses the keys that move body b along the nav graph toward
// (tx, ty). ns is the body's path state. Zombies and AI teammates (walking
// to a downed player, zombie_bots.go) share it.
func navSteer(b *Player, ns *zombieNavState, anim int, tx, ty float64, nav *zombieNav) {
	navSteerKeys(b, ns, anim, tx, ty, nav)
	navEdgeGuard(b, ns, nav)
}

// navEdgeGuard brakes a fast grounded body that would slide off its
// platform where the path does not leave it (a runner at full speed coasts
// some 60 px).
func navEdgeGuard(b *Player, ns *zombieNavState, nav *zombieNav) {
	if b.JumpNum != 2 || ns.cur < 0 || ns.cur >= len(nav.plats) || math.Abs(b.VX) <= 2.5 {
		return
	}
	dir := zsign(b.VX)
	if ns.hasLink {
		switch ns.link.kind {
		case zombieLinkHop, zombieLinkFall, zombieLinkWalk:
			if ns.link.dir == dir || ns.link.kind == zombieLinkWalk {
				return // leaving the platform this way is the plan
			}
		}
	}
	r := nav.plats[ns.cur]
	stopX := b.X + b.VX*math.Abs(b.VX)/1.2
	if stopX >= r.X+6 && stopX <= r.X+r.W-6 {
		return
	}
	b.AILeft, b.AIRight = false, false
	zombieHold(b, -dir)
}

func navSteerKeys(b *Player, ns *zombieNavState, anim int, tx, ty float64, nav *zombieNav) {
	grounded := b.JumpNum == 2
	if grounded {
		ns.cur = nav.platformUnder(b.X, b.Y)
		if ns.cur >= 0 {
			ns.lastPlat = ns.cur
		}
		ns.airFrames = 0
	} else {
		ns.airFrames++
	}
	goal := nav.platformNear(tx, ty)
	if grounded {
		ns.hasLink = false
		ns.jumped = false
		if ns.cur >= 0 && goal >= 0 && ns.cur != goal {
			if li := nav.next[ns.cur][goal]; li >= 0 {
				ns.link = nav.links[ns.cur][li]
				ns.hasLink = true
			}
		}
	}
	if !ns.hasLink {
		// Same platform (or no known way): go straight for the target.
		dx := tx - b.X
		if math.Abs(dx) > 6 {
			zombieHold(b, zsign(dx))
		}
		if grounded && ty < b.Y-60 && math.Abs(dx) < 40 && anim%30 == 0 {
			navPressUp(b, ns)
		} else {
			ns.upPrev = false
		}
		return
	}
	l := ns.link
	dest := nav.plats[l.to]
	margin := math.Min(30, dest.W/3)
	landX := math.Max(dest.X+margin, math.Min(tx, dest.X+dest.W-margin))
	switch l.kind {
	case zombieLinkWalk:
		zombieHold(b, zsign(landX-b.X))
		ns.upPrev = false
	case zombieLinkHop:
		if grounded {
			ahead := (l.x - b.X) * float64(l.dir) // > 0: launch point still ahead
			switch {
			case ahead < -40:
				zombieHold(b, -l.dir) // went past the edge zone: come back
				ns.upPrev = false
			case ahead > 12:
				zombieHold(b, l.dir)
				ns.upPrev = false
			default:
				zombieHold(b, l.dir)
				navPressUp(b, ns)
				ns.jumped = true
			}
		} else {
			far := dest.X + dest.W - margin
			if l.dir < 0 {
				far = dest.X + margin
			}
			switch {
			case (b.X-far)*float64(l.dir) > 0 && b.Y < dest.Y && b.VY > 0 && dest.Y <= nav.plats[nav.clampIndex(ns.lastPlat)].Y+6:
				// A fast body is flying past a platform at its own height
				// (nothing below to land on): come back.
				zombieHold(b, -l.dir)
			case b.X > dest.X+margin && b.X < dest.X+dest.W-margin:
				navAirSteer(b, landX)
			default:
				zombieHold(b, l.dir)
			}
			if ns.jumped && b.VY > -1.5 && b.JumpNum == 1 && b.Y > dest.Y-25 {
				navPressUp(b, ns)
			} else {
				ns.upPrev = false
			}
		}
	case zombieLinkClimb:
		if grounded {
			dx := l.x - b.X
			// Coasting distance of a fast body: a runner at full speed
			// would slide past a launch point near a platform's edge.
			stop := b.VX * b.VX / 1.2
			switch {
			case math.Abs(dx) <= 10 && math.Abs(b.VX) < 3:
				navPressUp(b, ns)
				ns.jumped = true
			case math.Abs(dx) <= 10+math.Max(3*math.Abs(b.VX), stop) && zsign(dx) == zsign(b.VX):
				if math.Abs(b.VX) > 4 {
					zombieHold(b, -zsign(b.VX)) // brake hard
				}
				ns.upPrev = false
			default:
				zombieHold(b, zsign(dx))
				ns.upPrev = false
			}
		} else {
			// Stay over the overlap until landed; the path goes on from there.
			if dx := l.x - b.X; math.Abs(dx) > 8 && !(math.Abs(dx) < 4*math.Abs(b.VX) && zsign(dx) == zsign(b.VX)) {
				zombieHold(b, zsign(dx))
			}
			if ns.jumped && b.VY > -1.5 && b.JumpNum == 1 && b.Y > dest.Y-25 {
				navPressUp(b, ns)
			} else {
				ns.upPrev = false
			}
		}
	case zombieLinkDrop:
		ns.upPrev = false
		if grounded {
			dx := l.x - b.X
			switch {
			case zombieDropLandsOn(b, dest):
				// The fall (with the current speed) ends well inside the
				// platform below: drop now.
				b.AIDown = true
			case math.Abs(dx) <= 14 && math.Abs(b.VX) > 3:
				// Too fast at the drop point: brake against the motion.
				zombieHold(b, -zsign(b.VX))
			case math.Abs(dx) <= 14:
				b.AIDown = true
			default:
				zombieHold(b, zsign(dx))
			}
		} else {
			navAirSteer(b, landX)
		}
	case zombieLinkFall:
		ns.upPrev = false
		if grounded {
			zombieHold(b, l.dir)
		} else if b.Y > nav.plats[nav.clampIndex(ns.lastPlat)].Y+8 {
			navAirSteer(b, landX)
		} else {
			zombieHold(b, l.dir)
		}
	}
}

// zombieDropLandsOn: a body dropping through its platform now would land
// on dest with a margin, given its horizontal speed (a fast runner drifts
// far while it falls).
func zombieDropLandsOn(b *Player, dest Rect) bool {
	dy := dest.Y - b.Y
	if dy <= 0 {
		return false
	}
	t := math.Sqrt(2 * dy / DefaultGravity)
	land := b.X + b.VX*t
	const margin = 24.0
	return b.X >= dest.X+margin && b.X <= dest.X+dest.W-margin && land >= dest.X+margin && land <= dest.X+dest.W-margin
}

// navAirSteer steers a falling body toward landX, braking early enough for
// its speed (air control takes about 1 px/frame off the speed per frame).
func navAirSteer(b *Player, landX float64) {
	predict := b.X + b.VX*math.Abs(b.VX)/2
	if d := landX - predict; math.Abs(d) > 3 {
		zombieHold(b, zsign(d))
	}
}

func (nav *zombieNav) clampIndex(i int) int {
	if i < 0 || i >= len(nav.plats) {
		return 0
	}
	return i
}

// zombieWander: no visible target (smoke, blindness, everybody down) -
// shamble back and forth on the platform.
func (g *Game) zombieWander(z *Zombie, nav *zombieNav) {
	b := &z.body
	z.nav.upPrev = false
	if b.BlindTime > 0 {
		if (z.Anim/20)%2 == 0 {
			zombieHold(b, b.Facing)
		}
		return
	}
	if b.JumpNum != 2 {
		return
	}
	cur := nav.platformUnder(b.X, b.Y)
	if cur < 0 {
		return
	}
	r := nav.plats[cur]
	if b.X < r.X+30 {
		b.Facing = 1
	} else if b.X > r.X+r.W-30 {
		b.Facing = -1
	}
	if (z.Anim/45)%3 != 2 {
		zombieHold(b, b.Facing)
	}
}

func zombieHold(b *Player, dir int) {
	switch {
	case dir < 0:
		b.AILeft = true
	case dir > 0:
		b.AIRight = true
	}
}

// zombiePressUp presses jump on every other frame so the jump key is seen
// as released in between (double jumps need a fresh press).
func zombiePressUp(z *Zombie) {
	navPressUp(&z.body, &z.nav)
}

func navPressUp(b *Player, ns *zombieNavState) {
	if ns.upPrev {
		ns.upPrev = false
		return
	}
	b.AIUp = true
	ns.upPrev = true
}

func zsign(v float64) int {
	switch {
	case v < 0:
		return -1
	case v > 0:
		return 1
	}
	return 0
}

// zombieTrackStuck has two levels. Short: a zombie that wants to move but
// stays inside a small box for zombieStuckFrames (pushing against something,
// bouncing on an edge) jumps and turns around for a moment. Long: when its
// path distance to the target has not improved for zombieRespawnStuck frames
// it climbs out of the ground again at a spawn point.
func (g *Game) zombieTrackStuck(z *Zombie, target *Player, nav *zombieNav) {
	b := &z.body
	if target == nil || z.State != zombieStateWalk || zombieInReach(z, target, 30) {
		z.stuckFrames = 0
		z.stuckBestDist = math.Inf(1)
		z.boxFrames = 0
		return
	}
	// Short level: movement box.
	if z.boxFrames == 0 {
		z.boxMinX, z.boxMaxX, z.boxMinY, z.boxMaxY = b.X, b.X, b.Y, b.Y
	}
	z.boxMinX, z.boxMaxX = math.Min(z.boxMinX, b.X), math.Max(z.boxMaxX, b.X)
	z.boxMinY, z.boxMaxY = math.Min(z.boxMinY, b.Y), math.Max(z.boxMaxY, b.Y)
	z.boxFrames++
	if z.boxFrames >= zombieStuckFrames {
		if z.boxMaxX-z.boxMinX < 22 && z.boxMaxY-z.boxMinY < 22 && math.Abs(target.X-b.X) > 40 {
			z.unstickDir = -b.Facing
			if z.unstickDir == 0 {
				z.unstickDir = 1
			}
			z.unstickTimer = 20
		}
		z.boxFrames = 0
	}
	// Long level: path progress.
	if b.JumpNum != 2 {
		return
	}
	cur := nav.platformUnder(b.X, b.Y)
	goal := nav.platformNear(target.X, target.Y)
	d := math.Abs(target.X-b.X) + math.Abs(target.Y-b.Y)
	if cur >= 0 && goal >= 0 && cur != goal && nav.reachable(cur, goal) {
		d = nav.dist[cur][goal] + 1000
	}
	if d < z.stuckBestDist-5 {
		z.stuckBestDist = d
		z.stuckFrames = 0
		return
	}
	z.stuckFrames++
	if z.stuckFrames >= zombieRespawnStuck {
		pt, zone := g.pickZombieSpawn(false)
		b.X, b.Y, b.VX, b.VY = pt.X, pt.Y, 0, 0
		b.JumpNum, b.Grounded = 2, true
		z.Zone = zone
		z.State = zombieStateRising
		z.StateTimer = zombieRiseFrames
		z.stuckFrames = 0
		z.stuckBestDist = math.Inf(1)
		z.boxFrames = 0
		z.nav = zombieNavState{cur: -1, lastPlat: -1}
		g.zombie.stuckRespawns++
	}
}

// --- kind specials ------------------------------------------------------------------

// zombieTryLeap pounces at a target on the same platform (or jumping above
// it), so a leap never throws the leaper into a gap.
func (g *Game) zombieTryLeap(z *Zombie, t *Player, nav *zombieNav) bool {
	b := &z.body
	dx := t.X - b.X
	dy := t.Y - b.Y
	if math.Abs(dx) > zombieLeapRange || math.Abs(dx) < 30 || dy < -150 || dy > 60 {
		return false
	}
	cur := nav.platformUnder(b.X, b.Y)
	if cur < 0 || cur != nav.platformNear(t.X, t.Y) {
		return false
	}
	dir := float64(zsign(dx))
	speed := math.Max(1, g.zombie.plan.SpeedMul)
	b.VX = dir * zombieLeapVX * math.Min(speed, 1.2)
	b.VY = -(zombieLeapVY + math.Max(0, -dy)*0.035)
	b.Y--
	b.JumpNum = 1
	b.Grounded = false
	b.Facing = int(dir)
	z.State = zombieStateLeap
	z.StateTimer = 0
	z.leapHit = false
	return true
}

// zombieBossSpecial: ground slam and summons. Returns true when it took the
// boss's action for this frame.
func (g *Game) zombieBossSpecial(z *Zombie, t *Player, arena Map, airborne bool) bool {
	b := &z.body
	phase2 := z.HP*2 < z.MaxHP
	if z.summonTimer > 0 {
		z.summonTimer--
	}
	if z.slamTimer > 0 {
		z.slamTimer--
	}
	if z.summonTimer == 0 && !airborne {
		z.summonTimer = zombieBossSummonEvery
		if phase2 {
			z.summonTimer = zombieBossSummonEvery * 3 / 4
		}
		g.zombieBossSummon(z, phase2)
	}
	if z.slamTimer == 0 && !airborne && math.Abs(t.X-b.X) < 420 {
		every := zombieBossSlamEvery
		if phase2 {
			every = every * 3 / 5
		}
		z.slamTimer = every
		b.VY = -13.5
		b.VX = float64(zsign(t.X-b.X)) * 4
		b.Y--
		b.JumpNum = 1
		b.Grounded = false
		z.State = zombieStateSlam
		z.StateTimer = 0
		return true
	}
	return false
}

func (g *Game) zombieBossSlam(z *Zombie) {
	b := &z.body
	z.State = zombieStateWalk
	z.cooldown = 20
	for _, p := range g.players {
		if p == nil || !p.Active {
			continue
		}
		dx := p.X - b.X
		if math.Abs(dx) > zombieBossSlamRadius || math.Abs(p.Y-b.Y) > 80 {
			continue
		}
		k := 1 - math.Abs(dx)/zombieBossSlamRadius*0.6
		dir := float64(zsign(dx))
		if dir == 0 {
			dir = 1
		}
		g.zombieHurtPlayer(p, int(math.Round(zombieBossSlamDamage*k)), dir*zombieBossSlamPush*k, 9*k, b.ID)
	}
	g.blastFX = append(g.blastFX, spawnGrenadeBlastEffects(b.X-40, b.Y-6)...)
	g.blastFX = append(g.blastFX, spawnGrenadeBlastEffects(b.X+40, b.Y-6)...)
	g.playRandomExplosionSound()
}

func (g *Game) zombieBossSummon(z *Zombie, phase2 bool) {
	m := g.zombie
	b := &z.body
	if m.livingZombies() >= zombieMaxActive {
		return
	}
	nav := m.navFor(z.Zone, g.zombieZoneMap(z.Zone), zombieNavNormal)
	cur := nav.platformUnder(b.X, b.Y)
	for i := 0; i < zombieBossSummonCount; i++ {
		kind := zombieWalker
		if i%2 == 1 {
			kind = zombieRunner
		}
		if phase2 && i == 2 {
			kind = zombieExploder
		}
		x := b.X + float64(i-1)*70
		y := b.Y
		if cur >= 0 {
			r := nav.plats[cur]
			x = math.Max(r.X+16, math.Min(x, r.X+r.W-16))
			y = r.Y
		}
		g.newZombie(kind, zombieScaledHP(kind, m.plan), zombiePoint{x, y}, z.Zone)
	}
	g.playSourceSFX("emp.wav", true)
}

// --- exploder blasts --------------------------------------------------------------------

// updateZombieBlasts sets off pending exploder blasts. It runs outside the
// players' knockback window with its own window over every target.
func (g *Game) updateZombieBlasts() {
	if !g.zombieSimulating() || len(g.zombie.blasts) == 0 {
		return
	}
	m := g.zombie
	ready := make([]zombiePendingBlast, 0, 4)
	keep := m.blasts[:0]
	for _, bl := range m.blasts {
		if m.passZone >= 0 && bl.zone != m.passZone {
			keep = append(keep, bl) // another zone's pass sets it off
			continue
		}
		bl.timer--
		if bl.timer <= 0 {
			ready = append(ready, bl)
		} else {
			keep = append(keep, bl)
		}
	}
	m.blasts = keep
	for _, bl := range ready {
		g.zombieExplode(bl)
	}
}

func (g *Game) zombieExplode(bl zombiePendingBlast) {
	m := g.zombie
	targets := g.hitTargets()
	win := beginAbilityKnockbackWindow(targets)
	cx, cy := bl.x, bl.y-25
	for _, t := range targets {
		if t == nil || !t.Active {
			continue
		}
		dx := t.X - cx
		dy := (t.Y - 30*t.PlayerScale/0.8) - cy
		d := math.Hypot(dx, dy)
		if d > zombieExploderRadius {
			continue
		}
		k := 1 - d/zombieExploderRadius*0.66
		dir := float64(zsign(dx))
		if dir == 0 {
			dir = 1
		}
		if t.ID >= zombieIDBase {
			z := m.zombieByBodyID(t.ID)
			if z == nil || !z.alive() {
				continue
			}
			t.VX += dir * zombieExploderPush * k
			t.VY -= zombieExploderPush * 0.45 * k
			if bl.ownerID > 0 {
				t.LastHitBy = bl.ownerID
				z.creditID = bl.ownerID
				z.creditTimer = zombieCreditFrames
			}
			if z.kind() == zombieExploder {
				z.chained = true
			}
			g.zombieDamage(z, int(math.Round(90*k)), bl.ownerID, false)
			if !z.Dead {
				z.chained = false // survived: a later bullet kill blows at once
			}
			continue
		}
		g.zombieHurtPlayer(t, int(math.Round(zombieExploderDamage*k)), dir*zombieExploderPush*k, zombieExploderPush*0.5*k, 0)
	}
	endAbilityKnockbackWindow(targets, win, true)
	g.blastFX = append(g.blastFX, spawnGrenadeBlastEffects(bl.x, bl.y-15)...)
	g.explosions = append(g.explosions, Explosion{X: bl.x, Y: bl.y - 25, Life: 18})
	g.playRandomExplosionSound()
}

func (m *zombieMatch) zombieByBodyID(id int) *Zombie {
	for _, z := range m.zombies {
		if z.ID == id {
			return z
		}
	}
	return nil
}
