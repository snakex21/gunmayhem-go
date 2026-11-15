package game

import "math"

// Non-player campaign enemies (drones, shield-bearers, later bosses).
//
// They are deliberately NOT entries of Game.players: the shared camera, the
// win checks, HUD cards, scoreboards and bot targeting all walk that slice and
// must not see them. Each enemy owns a private Player value (Body) that is
// used purely as a physics / hit receiver. During the hit phase of a frame
// the bodies are appended to the player list handed to the existing combat
// code (updateBullets, grenades, gas, flashbang, fire, hazards), so every
// knockback source, the ability knockback windows (KnockbackScale), kill
// credit (LastHitBy) and the hit counters work without a parallel code path.
// See docs/campaign_enemies.md for how to add a new kind or a large boss.

type campaignEnemyKind int

const (
	campaignEnemyNone campaignEnemyKind = iota
	campaignEnemyDrone
	campaignEnemyShieldBearer
	campaignEnemyMech       // campaign_enemy_mech.go (Stage B)
	campaignEnemyHelicopter // campaign_boss_heli.go (Stage B)
	campaignEnemyKindCount
)

const (
	// campaignEnemyIDBase keeps enemy body IDs far away from player slots
	// (1-4) and helper doubles (5+), so OwnerID / LastHitBy stay unique.
	campaignEnemyIDBase = 100
	campaignEnemyTeam   = 2
	// campaignEnemyCreditFrames: a ring-out still credits the last player who
	// hit the enemy within this many frames.
	campaignEnemyCreditFrames = 150
	// campaignEnemyBlastDamageImpulse converts a non-bullet impulse (blast,
	// gas, shove) into HP damage: one point per this much velocity change.
	campaignEnemyBlastDamageImpulse = 14.0
	campaignEnemyDeathFXFrames      = 24
	// Big enemies (mech, helicopter) leave their wreck on screen longer.
	campaignBigWreckFrames = 60
)

// extraCampaignUnit is the mission-data description of one enemy wave.
type extraCampaignUnit struct {
	Kind      campaignEnemyKind
	Name      string
	Count     int // total enemies of this wave that must be defeated
	MaxActive int // simultaneously on the arena (0 means 1)
	Gun       int // 0 = the kind's default weapon
	HP        int // 0 = only a ring-out destroys it; >0 = also HP hits
	Delay     int // frames before the first one arrives
	Interval  int // frames between reinforcements (0 = 150)
	ShotDelay int // frames between shots (0 = the kind's default)
	// Title shows a named boss bar for an HP enemy (mech elite, gunship).
	Title string
	// Rockets arms a mech with its shoulder mortar (marked strikes).
	Rockets bool
}

// CampaignEnemy is one live non-player enemy. Exported fields are carried by
// the netcodec snapshot (CampaignEnemies) so a client can render them.
type CampaignEnemy struct {
	Kind        int
	Body        Player
	HP          int
	MaxHP       int
	Frame       int
	State       int
	StateTimer  int
	ShotTimer   int
	ShotDelay   int
	ShoveTimer  int
	ShieldDown  int // > 0 while the riot shield is lowered
	ShieldSpark int // deflection spark frames left
	Stall       int // drone rotors stalled by a shock cloud
	HitFlash    int
	Tilt        float64
	HomeX       float64
	HomeY       float64
	Dead        bool
	DeadFrames  int
	// Stage B (protocol 25): big enemies. Phase is the gunship phase (0-2);
	// Elite draws the named boss bar (Title); AimX0/AimX1/AimY are the
	// telegraph of the current attack (laser, stomp ring, strafing line);
	// Hooked counts down while a harpoon drags the gunship down.
	Phase  int     `cbor:",omitempty"`
	Elite  bool    `cbor:",omitempty"`
	Title  string  `cbor:",omitempty"`
	AimX0  float64 `cbor:",omitempty"`
	AimX1  float64 `cbor:",omitempty"`
	AimY   float64 `cbor:",omitempty"`
	Hooked int     `cbor:",omitempty"`
	// Mortar: this mech carries the shoulder mortar (drawn as a pod).
	Mortar bool `cbor:",omitempty"`

	// Host-only bookkeeping (unexported: never on the wire).
	preVX, preVY float64
	// windVX/VY: leaf blower push this tick (arsenal2): it moves the enemy
	// but is not a hit (no flash, no credit, no HP).
	windVX, windVY float64
	preHits        int
	creditTimer    int
	finished       bool
	// Big-enemy bookkeeping (host only).
	dmgAcc      float64 // fractional push damage carried to the next tick
	attackCD    int
	burst       int
	salvo       int
	crashCredit int
	hookCD      int
	hookPrev    int
	drops       int
	summoner    int // body ID of the gunship that dropped this add
	nav         zombieNavState
	stuckFrames int
	stuckX      float64
	cycle       int
	hookBy      int
	preTether   int
	preLastHit  int
	approach    int // gunship ticks spent flying to a drop point
	rocketPts   [heliMaxSalvo][2]float64
}

type campaignEnemyWave struct {
	Unit      extraCampaignUnit
	Remaining int
	Timer     int
	Side      int
	spawned   int
}

func (k campaignEnemyKind) defaultName() string {
	switch k {
	case campaignEnemyDrone:
		return "Drone"
	case campaignEnemyShieldBearer:
		return "Shield-Bearer"
	case campaignEnemyMech:
		return "Mech"
	case campaignEnemyHelicopter:
		return "Gunship"
	}
	return "Enemy"
}

func (k campaignEnemyKind) defaultGun() int {
	switch k {
	case campaignEnemyDrone:
		return 99 // Glock 18C: light, quick taps
	case campaignEnemyShieldBearer:
		return 108 // Makarov PM held past the shield edge
	case campaignEnemyMech:
		return mechGun
	}
	return 1
}

func (k campaignEnemyKind) defaultShotDelay() int {
	switch k {
	case campaignEnemyDrone:
		return 48
	case campaignEnemyShieldBearer:
		return 58
	}
	return 60
}

func (e *CampaignEnemy) alive() bool {
	return e != nil && !e.Dead && e.Body.Active
}

// hittable: alive and not in a death sequence (a crashing gunship or a
// melting-down mech still counts as standing, but takes no more hits).
func (e *CampaignEnemy) hittable() bool {
	if !e.alive() {
		return false
	}
	switch campaignEnemyKind(e.Kind) {
	case campaignEnemyMech:
		return e.State != mechStateMeltdown
	case campaignEnemyHelicopter:
		return e.State != heliStateCrash && e.State != heliStateEnter
	}
	return true
}

// bigEnemy: HP comes from the raw push (damage = knockback) instead of hit
// counts, and the de-materializer cannot erase it (campaign_enemies_big.go).
func (k campaignEnemyKind) bigEnemy() bool {
	return k == campaignEnemyMech || k == campaignEnemyHelicopter
}

// flyer: never lands, so kill credit simply expires.
func (k campaignEnemyKind) flyer() bool {
	return k == campaignEnemyDrone || k == campaignEnemyHelicopter
}

// --- setup -----------------------------------------------------------------

func (g *Game) resetCampaignEnemies() {
	g.campaignEnemies = g.campaignEnemies[:0]
	g.campaignEnemyWaves = g.campaignEnemyWaves[:0]
	g.nextCampaignEnemyID = campaignEnemyIDBase + 1
	g.campaignEnemyFrame = 0
	g.campaignStrikes = g.campaignStrikes[:0]
}

func (g *Game) setupCampaignEnemyWaves(units []extraCampaignUnit) {
	g.resetCampaignEnemies()
	for i, u := range units {
		if u.Kind <= campaignEnemyNone || u.Kind >= campaignEnemyKindCount || u.Count <= 0 {
			continue
		}
		side := -1
		if i%2 == 1 {
			side = 1
		}
		g.campaignEnemyWaves = append(g.campaignEnemyWaves, campaignEnemyWave{Unit: u, Remaining: u.Count, Timer: u.Delay, Side: side})
	}
}

// campaignEnemiesRemaining counts living enemies plus queued reinforcements.
// The EXTRA win check treats them exactly like enemy bots with lives left.
func (g *Game) campaignEnemiesRemaining() int {
	if g == nil {
		return 0
	}
	n := 0
	for _, e := range g.campaignEnemies {
		if e.alive() {
			n++
		}
	}
	for _, w := range g.campaignEnemyWaves {
		n += w.Remaining
	}
	return n
}

// hitTargets is g.players plus every living enemy body: the list handed to
// the shared hit code (bullets, explosions, gas, flash, fire, hazards).
func (g *Game) hitTargets() []*Player {
	if len(g.campaignEnemies) == 0 && !g.zombieHasBodies() {
		return g.players
	}
	out := make([]*Player, 0, len(g.players)+len(g.campaignEnemies)+zombieMaxActive)
	out = append(out, g.players...)
	for _, e := range g.campaignEnemies {
		if e.hittable() {
			out = append(out, &e.Body)
		}
	}
	// Zombie bodies (zombie_mode.go) take every hit the same way.
	return g.appendZombieBodies(out)
}

func (g *Game) campaignEnemyByID(id int) *CampaignEnemy {
	if id < campaignEnemyIDBase {
		return nil
	}
	for _, e := range g.campaignEnemies {
		if e.Body.ID == id {
			return e
		}
	}
	return nil
}

// campaignArenaBounds is the platform bounding box: x range and top/bottom.
func campaignArenaBounds(arena Map) (minX, maxX, top, bottom float64) {
	if len(arena.Platforms) == 0 {
		return 0, 900, 100, 500
	}
	minX, maxX = math.Inf(1), math.Inf(-1)
	top, bottom = math.Inf(1), math.Inf(-1)
	for _, r := range arena.Platforms {
		minX = math.Min(minX, r.X)
		maxX = math.Max(maxX, r.X+r.W)
		top = math.Min(top, r.Y)
		bottom = math.Max(bottom, r.Y+r.H)
	}
	return
}

// campaignEnemyOutOfBounds is the enemy kill zone: well past the arena edges
// (off-screen) or below the lowest platform.
func campaignEnemyOutOfBounds(arena Map, x, y float64) bool {
	minX, maxX, _, bottom := campaignArenaBounds(arena)
	return x < minX-260 || x > maxX+260 || y > bottom+230 || y > 1000 || y < -500
}

func (g *Game) spawnCampaignEnemy(unit extraCampaignUnit, side int, arena Map) *CampaignEnemy {
	if g.nextCampaignEnemyID <= campaignEnemyIDBase {
		g.nextCampaignEnemyID = campaignEnemyIDBase + 1
	}
	id := g.nextCampaignEnemyID
	g.nextCampaignEnemyID++
	kind := unit.Kind
	name := unit.Name
	if name == "" {
		name = kind.defaultName()
	}
	gun := unit.Gun
	if gun == 0 {
		gun = kind.defaultGun()
	}
	shotDelay := unit.ShotDelay
	if shotDelay <= 0 {
		shotDelay = kind.defaultShotDelay()
	}
	body := Player{
		ID: id, Name: name, Team: campaignEnemyTeam, AI: true, Active: true,
		Facing: 1, Lives: 1, TotalLives: 1, GameMode: SourceGameModeNormal,
		DefaultWeapon: gun, PerkNumber: 7, Weight: 1, DamageMulti: 1, FirepowerMulti: 1,
		ThrowableType: ThrowableDynamite, PortalNextColor: PortalBlue,
		SpeedTime: -1, MiniTime: -1, MiniMulti: 1, PlayerScale: 0.8, Alpha: 1, ShieldAlpha: 1,
		PlayerColor: 1, ShirtNumber: 1, HatNumber: 1, EyeNumber: 1, HeadFrame: 1 + id%50,
		VisualBodyY: -60, VisualEyesY: -46, VisualHand1Y: -35, VisualHand2Y: -25,
		VisualHand1ChildX: -5, VisualHand1ChildY: 10.15, VisualHand2ChildX: 20, VisualHand2ChildY: -3.75,
		VisualHand1Alpha: 1, VisualHand2Alpha: 1, IdleTime: 40, IdleMax: 40,
		CheapTimer: 120,
	}
	body.EquipWeapon(gun)
	minX, maxX, top, _ := campaignArenaBounds(arena)
	if side == 0 {
		side = 1
	}
	hp := unit.HP
	if kind.bigEnemy() && hp > 0 {
		hp = g.campaignCoopHP(hp)
	}
	e := &CampaignEnemy{Kind: int(kind), HP: hp, MaxHP: hp, ShotDelay: shotDelay, ShotTimer: shotDelay / 2,
		Elite: unit.Title != "", Title: unit.Title, Mortar: unit.Rockets}
	switch kind {
	case campaignEnemyDrone:
		body.PlayerScale = droneHitboxScale
		body.MiniMulti = droneMuzzleScale
		body.KnockbackScale = droneKnockbackScale
		body.Grounded = false
		// Fly in from just beyond the arena edge, high up.
		if side < 0 {
			body.X = minX - 90
		} else {
			body.X = maxX + 90
		}
		body.Y = top - 20
		body.Facing = -side
		e.HomeX = (minX+maxX)/2 + float64(side)*(maxX-minX)*0.22
		e.HomeY = top - 30
	case campaignEnemyShieldBearer:
		body.KnockbackScale = shieldBearerKnockbackScale
		body.CampaignMoveScale = shieldBearerMoveScale
		body.HatNumber = 1 // bare head: a procedural riot helmet is drawn on top
		body.PlayerColor = 9
		body.ShirtNumber = 4
		span := arena.SpawnMaxX - arena.SpawnMinX
		if span <= 0 {
			span = maxX - minX
		}
		center := (arena.SpawnMinX + arena.SpawnMaxX) / 2
		body.X = center + float64(side)*span*0.35
		body.Y = -180
		body.JumpNum = 1
		body.Facing = -side
		e.HomeX = body.X
	case campaignEnemyMech:
		setupMechBody(&body, e, arena, side)
	case campaignEnemyHelicopter:
		setupHeliBody(&body, e, arena, side)
	}
	e.Body = body
	g.campaignEnemies = append(g.campaignEnemies, e)
	return e
}

// --- per-frame update ------------------------------------------------------

func (g *Game) spawnCampaignEnemyWaves(arena Map) {
	for i := range g.campaignEnemyWaves {
		w := &g.campaignEnemyWaves[i]
		if w.Remaining <= 0 {
			continue
		}
		if w.Timer > 0 {
			w.Timer--
			continue
		}
		maxActive := w.Unit.MaxActive
		if maxActive <= 0 {
			maxActive = 1
		}
		active := 0
		for _, e := range g.campaignEnemies {
			if e.alive() && campaignEnemyKind(e.Kind) == w.Unit.Kind && e.Body.Name == unitName(w.Unit) {
				active++
			}
		}
		if active >= maxActive {
			continue
		}
		g.spawnCampaignEnemy(w.Unit, w.Side, arena)
		w.Side = -w.Side
		w.Remaining--
		w.spawned++
		interval := w.Unit.Interval
		if interval <= 0 {
			interval = 150
		}
		w.Timer = interval
	}
}

func unitName(u extraCampaignUnit) string {
	if u.Name != "" {
		return u.Name
	}
	return u.Kind.defaultName()
}

// updateCampaignEnemies runs every enemy's movement and AI. It is called
// after the player loop and before the hit phase, inside a knockback window
// for the players so shoves respect HEAVY / ANCHOR / SHIELD.
func (g *Game) updateCampaignEnemies(arena Map) {
	if len(g.campaignEnemies) == 0 && len(g.campaignEnemyWaves) == 0 {
		return
	}
	if g.GameWin || g.teamGameWin {
		for _, e := range g.campaignEnemies {
			e.Body.VX, e.Body.VY = 0, 0
			e.Body.AIShoot = false
		}
		g.advanceCampaignEnemyDeathFX()
		g.campaignStrikes = g.campaignStrikes[:0]
		return
	}
	g.campaignEnemyFrame++
	g.spawnCampaignEnemyWaves(arena)
	g.updateCampaignStrikes(arena)
	for _, e := range g.campaignEnemies {
		if e.Dead {
			continue
		}
		e.Frame++
		if e.ShieldSpark > 0 {
			e.ShieldSpark--
		}
		if e.HitFlash > 0 {
			e.HitFlash--
		}
		if e.ShoveTimer > 0 {
			e.ShoveTimer--
		}
		if e.ShotTimer > 0 {
			e.ShotTimer--
		}
		if e.creditTimer > 0 {
			e.creditTimer--
			if e.creditTimer == 0 && campaignEnemyKind(e.Kind).flyer() {
				// Walkers lose credit on landing like players (resolvePlatforms);
				// a flyer never lands, so its credit simply expires.
				e.Body.LastHitBy = 0
				e.Body.HitByGrenade = false
			}
		}
		switch campaignEnemyKind(e.Kind) {
		case campaignEnemyDrone:
			g.updateDrone(e, arena)
		case campaignEnemyShieldBearer:
			g.updateShieldBearer(e, arena)
		case campaignEnemyMech:
			g.updateMech(e, arena)
		case campaignEnemyHelicopter:
			g.updateHeli(e, arena)
		}
		if !e.Body.Active && !e.finished {
			g.finishCampaignEnemy(e)
		}
	}
	g.advanceCampaignEnemyDeathFX()
}

func (g *Game) advanceCampaignEnemyDeathFX() {
	out := g.campaignEnemies[:0]
	for _, e := range g.campaignEnemies {
		if e.Dead {
			e.DeadFrames++
			if e.DeadFrames > campaignEnemyKind(e.Kind).wreckFrames() {
				continue
			}
		}
		out = append(out, e)
	}
	for i := len(out); i < len(g.campaignEnemies); i++ {
		g.campaignEnemies[i] = nil
	}
	g.campaignEnemies = out
}

// campaignEnemyTarget picks the nearest opponent the enemy can actually
// see: blindness and smoke use the same rule as bots (aiCanDetect).
func (g *Game) campaignEnemyTarget(e *CampaignEnemy) *Player {
	var best *Player
	bestD := math.MaxFloat64
	for _, p := range g.players {
		if p == nil || !p.Active || p.Team == e.Body.Team {
			continue
		}
		if !g.aiCanDetect(&e.Body, p) {
			continue
		}
		d := math.Hypot(p.X-e.Body.X, p.Y-e.Body.Y)
		if d < bestD {
			bestD = d
			best = p
		}
	}
	return best
}

// fireCampaignEnemyWeapon runs the body's real weapon (ammo, reload, sound,
// blind / smoke aim disturbance, credit through OwnerID) for one frame.
func (g *Game) fireCampaignEnemyWeapon(e *CampaignEnemy, pull bool) {
	b := &e.Body
	b.AIShoot = pull
	g.updatePlayerWeapon(b)
	for _, sound := range b.PendingSounds {
		g.playSourceSFX(sound, false)
	}
	b.PendingSounds = b.PendingSounds[:0]
	b.AIShoot = false
}

// --- hit phase -------------------------------------------------------------

// beginCampaignEnemyHitPhase remembers each body's state before bullets,
// blasts, gas and hazards run, so endCampaignEnemyHitPhase can turn the
// difference into HP damage and refresh the kill-credit timer.
func (g *Game) beginCampaignEnemyHitPhase() {
	for _, e := range g.campaignEnemies {
		if !e.hittable() {
			continue
		}
		e.preVX, e.preVY = e.Body.VX, e.Body.VY
		e.windVX, e.windVY = 0, 0
		e.preHits = e.Body.HitNumber
		e.preTether = e.Body.HarpoonTether
		e.preLastHit = e.Body.LastHitBy
	}
}

func (g *Game) endCampaignEnemyHitPhase(arena Map) {
	for _, e := range g.campaignEnemies {
		if !e.hittable() {
			continue
		}
		if campaignEnemyKind(e.Kind).bigEnemy() {
			g.endBigEnemyHitPhase(e)
			continue
		}
		b := &e.Body
		hits := b.HitNumber - e.preHits
		if hits < 0 {
			hits = 0
		}
		scale := abilityKnockbackScale(b)
		impulse := math.Hypot(b.VX-e.preVX-e.windVX*scale, b.VY-e.preVY-e.windVY*scale)
		e.windVX, e.windVY = 0, 0
		if hits > 0 || impulse > 0.5 {
			e.HitFlash = 6
			e.creditTimer = campaignEnemyCreditFrames
		}
		if b.InstagibPlaying && campaignEnemyKind(e.Kind) == campaignEnemyDrone {
			// The de-materializer destroys a drone outright.
			g.destroyCampaignEnemy(e)
			continue
		}
		if e.MaxHP <= 0 {
			continue
		}
		damage := hits
		if hits == 0 && impulse >= campaignEnemyBlastDamageImpulse {
			damage = int(impulse / campaignEnemyBlastDamageImpulse)
		}
		if damage <= 0 {
			continue
		}
		e.HP -= damage
		if e.HP <= 0 {
			e.HP = 0
			g.destroyCampaignEnemy(e)
		}
	}
}

// destroyCampaignEnemy kills an enemy in place (HP, ring-out of a flyer).
func (g *Game) destroyCampaignEnemy(e *CampaignEnemy) {
	b := &e.Body
	if !b.Active {
		return
	}
	b.LastDeathBy = b.LastHitBy
	b.LastDeathX, b.LastDeathY = b.X, b.Y
	b.LastDeathMod = 1
	if b.HitByGrenade {
		b.LastDeathMod = 3
	}
	b.DeathSerial++
	b.Lives = 0
	b.Active = false
	g.finishCampaignEnemy(e)
}

// finishCampaignEnemy awards the kill to the player who last hit the enemy
// (same rule as a player ring-out), adds the kill-feed line and the FX.
func (g *Game) finishCampaignEnemy(e *CampaignEnemy) {
	if e.finished {
		return
	}
	e.finished = true
	e.Dead = true
	e.DeadFrames = 0
	b := &e.Body
	if b.LastDeathBy == 0 && e.crashCredit > 0 {
		// A melting mech that falls off still belongs to its destroyer.
		b.LastDeathBy = e.crashCredit
	}
	if killer := g.playerByID(b.LastDeathBy); killer != nil && killer.Team != b.Team && !killer.IsDouble {
		killer.Kills++
		score := campaignEnemyKind(e.Kind).killScore()
		if b.LastDeathMod == 3 && score == 100 {
			score = 50
		}
		killer.Score += score
	} else if killer != nil && killer.IsDouble && killer.OwnerPlayerID > 0 {
		if owner := g.playerByID(killer.OwnerPlayerID); owner != nil {
			owner.Kills++
			owner.Score += campaignEnemyKind(e.Kind).killScore()
		}
	}
	g.appendSourceKillFeed(b)
	x, y := b.LastDeathX, b.LastDeathY
	if campaignEnemyKind(e.Kind).bigEnemy() {
		g.bigEnemyDeathFX(e)
		return
	}
	if campaignEnemyKind(e.Kind) == campaignEnemyDrone {
		y -= 16
		g.blastFX = append(g.blastFX, spawnGrenadeBlastEffects(x, y)...)
		g.playRandomExplosionSound()
	} else {
		y -= 20
		g.playRandomDeathSound()
	}
	g.explosions = append(g.explosions, Explosion{X: x, Y: y, Life: 18})
}

// --- netcode ---------------------------------------------------------------

func (g *Game) campaignEnemySnapshot() []CampaignEnemy {
	if len(g.campaignEnemies) == 0 {
		return nil
	}
	out := make([]CampaignEnemy, 0, len(g.campaignEnemies))
	for _, e := range g.campaignEnemies {
		if e != nil {
			out = append(out, *e)
		}
	}
	return out
}

// applyCampaignEnemySnapshot replaces the client's render copies. Clients
// never simulate enemies, so host-only bookkeeping stays zero.
func (g *Game) applyCampaignEnemySnapshot(list []CampaignEnemy) {
	g.campaignEnemies = g.campaignEnemies[:0]
	for i := range list {
		e := list[i]
		g.campaignEnemies = append(g.campaignEnemies, &e)
	}
}
