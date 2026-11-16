package game

import (
	"math"
	"math/rand"
)

// zombie_mode.go: the ZOMBIE match (host / local authority).
//
// Zombies are NOT entries of Game.players. Like the campaign enemies
// (campaign_enemies.go) each one owns a private Player body that is used only
// for platform physics and as a hit receiver: hitTargets() hands the living
// bodies to the shared bullet / blast / gas / fire code, so every weapon,
// throwable and knockback window works on zombies without a parallel path.
// Bullets do knockback AND damage (endZombieHitPhase turns the unscaled
// impulse of each frame into HP); falling off the map kills a zombie.
//
// Players get an HP bar (ZombiePlayerState). Zombie attacks deal damage and
// knockback; 0 HP or a ring-out puts the player DOWN at the last place they
// stood. A teammate standing there for 3 s revives; after 30 s the player is
// DEAD until the wave ends. Everybody down = GAME OVER.

// Zombie is one zombie. Exported fields are the snapshot (wire) state; the
// body and the AI bookkeeping are host-only.
type Zombie struct {
	ID         int
	Kind       int
	X, Y       float64
	VX, VY     float64
	Facing     int
	HP, MaxHP  int
	State      int
	StateTimer int
	Anim       int
	HitFlash   int
	Airborne   bool
	Dead       bool
	DeadFrames int
	RingOut    bool
	Zone       int
	Level      int // boss level (1, 2, ...)
	// DoorPasses counts the doors this zombie came through (render
	// interpolation snaps on a change).
	DoorPasses int
	// HarpoonTether / HarpoonTetherID mirror the body's harpoon rope so a
	// client can draw it (protocol 25).
	HarpoonTether   int `cbor:",omitempty"`
	HarpoonTetherID int `cbor:",omitempty"`

	body         Player
	preVX, preVY float64
	preHits      int
	headshot     bool
	creditID     int
	creditTimer  int
	cooldown     int
	targetID     int
	targetTimer  int
	nav          zombieNavState
	slamTimer    int
	summonTimer  int
	leapHit      bool
	burnTick     int
	bonusDamage  float64
	// windVX/VY: leaf blower push this tick (arsenal2). Wind moves zombies
	// but deals no damage, so it is taken out of the impulse.
	windVX, windVY float64
	chained        bool // killed by another exploder's blast
	finished       bool
	stuckFrames    int
	stuckBestDist  float64
	boxFrames      int
	boxMinX        float64
	boxMaxX        float64
	boxMinY        float64
	boxMaxY        float64
	unstickDir     int
	unstickTimer   int
	lastX          float64
}

// Zombie states.
const (
	zombieStateRising = iota
	zombieStateWalk
	zombieStateAttack // wind-up of a melee swing
	zombieStateFuse   // exploder hissing
	zombieStateLeap   // leaper pounce in the air
	zombieStateSlam   // boss in the air before a ground slam
)

func (z *Zombie) kind() zombieKind { return zombieKind(z.Kind) }

// alive: physically present and hittable.
func (z *Zombie) alive() bool {
	return z != nil && !z.Dead && z.State != zombieStateRising && z.body.Active
}

// ZombiePlayerState is the synced per-player part of the zombie HUD.
type ZombiePlayerState struct {
	ID             int
	HP, MaxHP      int
	Points         int
	Earned         int
	Kills          int
	Headshots      int
	Downs          int
	Revives        int
	State          int
	DownX, DownY   float64
	DownTimer      int
	ReviveProgress int
	SelfRevives    int
	Gun            int // bought / box weapon (0: starter pistol only)
	Reserve        int // spare magazines of Gun
	ShopOpen       bool
	Ready          bool
	ReqAck         uint32 // newest online shop request the host handled
	Invuln         int
	Zone           int
	Msg            int // last shop / box feedback (zombieMsg*)
	MsgTimer       int
}

// ZombieBoxState is the mystery box.
type ZombieBoxState struct {
	Spot    int
	X, Y    float64
	Zone    int
	State   int
	Timer   int
	Weapon  int // weapon shown while rolling / offered
	OwnerID int
	Uses    int
}

// ZombiePop is a floating "+60" points text.
type ZombiePop struct {
	X, Y   float64
	Amount int
	Kind   int
	Life   int
	Zone   int
}

// ZombieHUD is the synced match state (one value per snapshot).
type ZombieHUD struct {
	Active      bool
	Mode        int
	Arena       int
	Wave        int
	WaveTotal   int
	Phase       int
	PhaseTimer  int
	Remaining   int
	BossID      int
	BossHP      int
	BossMaxHP   int
	BossLevel   int
	BannerFrame int
	Players     [4]ZombiePlayerState
	Box         ZombieBoxState
	Kills       int
	RecordWave  int // best wave before this match (results screen)
	RecordScore int
	NewRecord   bool
	// Step 2 (zombie_zones.go): doors, the shared-screen gather and the
	// zone this snapshot shows (online: per guest).
	Doors  [zombieMaxDoors]ZombieDoorState
	Gather ZombieGatherState
	View   int
}

// zombieSetupConfig is what the setup screen chooses.
type zombieSetupConfig struct {
	Mode  int
	Arena int
	Slots [4]int // 0 off, 1 human, 2 bot
}

func defaultZombieSetup() zombieSetupConfig {
	return zombieSetupConfig{Mode: zombieModeSurvival, Arena: 0, Slots: [4]int{1, 0, 0, 0}}
}

// zombieLoadout is what a downed player gets back when revived.
type zombieLoadout struct {
	valid     bool
	weapon    WeaponState
	grenades  int
	throwable int
}

type zombieHostPlayer struct {
	seenSerial       int
	groundX, groundY float64
	loadout          zombieLoadout
	downHeld         bool
	remote           bool // online guest (RemoteInput comes from the network)
	masked           bool // local player whose game input is masked by the shop
	reviverID        int
	// Online door passing: doorCooldown blocks another pass, doorInvulnWait
	// withholds the arrival invulnerability from a door hopper.
	doorCooldown   int
	doorInvulnWait int
	// AI teammates: path to a downed teammate (zombie_bots.go).
	botNav      zombieNavState
	botReviveID int
	botBestDist float64 // closest the bot got to the marker
	botStall    int     // frames without getting closer
	botGiveUpID int     // marker the bot gave up on ...
	botGiveUp   int     // ... for this many frames
}

type zombiePendingBlast struct {
	x, y    float64
	timer   int
	ownerID int
	zone    int
}

// zombieMatch is the whole zombie state. On an online guest only hud,
// zombies and pops are filled (from snapshots) and host is false.
type zombieMatch struct {
	hud     ZombieHUD
	zombies []*Zombie
	pops    []ZombiePop
	host    bool

	arena      *zombieArenaDef
	plan       zombieWavePlan
	toSpawn    int
	spawnTimer int
	nextID     int
	rng        *rand.Rand
	navs       map[int]*zombieNav
	players    [4]zombieHostPlayer
	blasts     []zombiePendingBlast
	boxMoveAt  int
	ended      bool
	// stuckRespawns counts zombies re-spawned by the anti-stuck rule (tests).
	stuckRespawns int

	// Zones (zombie_zones.go, host / local only).
	zoneMaps  map[int]Map
	worlds    map[int]*zombieZoneWorld
	resident  int                 // zone whose lists are in the Game fields
	viewZone  int                 // zone this screen shows
	passZone  int                 // zone being simulated (-1 outside a zone pass)
	zoneCount [zombieMaxZones]int // living, standing players per zone
	// zonePresent: standing or downed players per zone (spawn weights).
	zonePresent [zombieMaxZones]int
	otherLiving int                 // during a zone pass: living zombies of the other zones
	zoneRoute   [zombieMaxZones]int // door toward the nearest player (-1 none)
	allPlayers  []*Player           // every player while a zone pass runs
	doorPasses  int                 // zombies that came through a door (tests)

	// Shop UI (local and guest): one panel per local player.
	shop [4]zombieShopUI
	// Online guest: queued shop requests until the host acknowledges them.
	clientReqs   []netZombieReq
	clientReqSeq uint32
	clientResend int
	store        []Zombie // guest: backing array of zombies
}

func (g *Game) zombieActive() bool {
	return g != nil && g.zombie != nil && g.zombie.hud.Active
}

// zombieSimulating: this process runs the zombie simulation (local or host).
func (g *Game) zombieSimulating() bool {
	return g.zombieActive() && g.zombie.host
}

func (m *zombieMatch) playerState(id int) *ZombiePlayerState {
	if m == nil || id < 1 || id > 4 {
		return nil
	}
	zp := &m.hud.Players[id-1]
	if zp.ID != id {
		return nil
	}
	return zp
}

func (m *zombieMatch) hostPlayer(id int) *zombieHostPlayer {
	if m == nil || id < 1 || id > 4 {
		return nil
	}
	return &m.players[id-1]
}

// --- match start ---------------------------------------------------------------

// zombieSlotPlayers counts the participants of a setup.
func (c zombieSetupConfig) participants() int {
	n := 0
	for _, s := range c.Slots {
		if s != 0 {
			n++
		}
	}
	return n
}

// startZombieMatch starts a local match (from the setup screen) or, when this
// game hosts an online zombie session, the online match.
func (g *Game) startZombieMatch() bool {
	online := g.onlineZombie && g.netplay != nil && g.netplay.mode == netplayHost
	cfg := g.zombieSetup
	if cfg.Mode < 0 || cfg.Mode >= zombieModeCount {
		cfg.Mode = zombieModeSurvival
	}
	if cfg.Arena < 0 || cfg.Arena >= len(zombieArenas) {
		cfg.Arena = 0
	}
	if online {
		g.syncOnlineHostState()
		cfg.Slots[0] = 1
		for id := 2; id <= maxOnlinePlayers; id++ {
			switch c := g.customPlayers[id-1]; {
			case c.Type == 2:
				cfg.Slots[id-1] = 2
			case c.Type == 1 && g.onlinePlayerSlots[id-1]:
				cfg.Slots[id-1] = 1
			default:
				cfg.Slots[id-1] = 0
			}
		}
	}
	cfg.Slots[0] = max(cfg.Slots[0], 1)
	arenaDef := zombieArenaByIndex(cfg.Arena)
	zone := arenaDef.zone(arenaDef.StartZone)
	arena, ok := g.ensureMap(zone.MapNumber)
	if !ok {
		arena, _ = g.ensureMap(1)
	}
	g.arena = arena

	players := make([]*Player, 0, 4)
	for slot, kind := range cfg.Slots {
		if kind == 0 {
			continue
		}
		pc := g.customPlayers[slot]
		id := slot + 1
		p := NewPlayer(id, arena)
		p.Controls = g.controlConfigs[slot]
		p.Name = pc.Name
		if p.Name == "" {
			p.Name = "Player " + string(rune('0'+id))
		}
		p.PlayerColor = pc.Color
		p.ShirtNumber = pc.Shirt
		p.HatNumber = pc.Hat
		p.EyeNumber = pc.Eyes
		p.UseGM2Cosmetics = false
		pc = migrateLegacyGearPerk(pc)
		if online {
			p.DefaultWeapon = g.onlineStartingWeapon(pc.Gun)
			p.PerkNumber = g.onlineAllowedPerk(pc.Perk)
			p.AbilityNumber = g.onlineAllowedAbility(pc.Ability, abilityNone)
			p.RemoteControlled = kind == 1 && id != 1
		} else {
			p.DefaultWeapon = pc.Gun
			p.PerkNumber = pc.Perk
			p.AbilityNumber = resolveConfigAbility(pc.Ability, nil)
		}
		if !isStarterHandgunNumber(p.DefaultWeapon) {
			p.DefaultWeapon = 1
		}
		p.ThrowableType = normalizedThrowableType(pc.Throwable)
		p.AI = kind == 2
		p.configureSourceGameMode(SourceGameModeNormal, zombieLives, arena)
		p.Team = 1
		players = append(players, p)
	}
	g.players = players
	g.customMatchActive = false
	g.matchModifiers = make(map[int]bool)
	g.resetChaosRuntime()
	g.customChaosEnabled = false
	g.GameMode = SourceGameModeNormal
	g.TotalLives = zombieLives
	g.TeamGame = false
	g.CrateON = false
	g.PowerON = false
	g.GameWin = false
	g.teamGameWin = false
	g.matchWinCountdown = 0
	g.soloWinFrame = 0
	g.winnerPlayerID = 0
	g.campaignMode = false
	g.gototest = false
	g.onlineMatchSetup = online
	g.resetArenaState()
	g.zombie = newZombieMatch(cfg, arenaDef, g.players, g.zombieRecordFor(cfg.Arena, cfg.Mode))
	g.initZombieZones()
	for _, p := range g.players {
		hp := g.zombie.hostPlayer(p.ID)
		if hp != nil {
			hp.remote = p.RemoteControlled
			hp.groundX, hp.groundY = g.zombieSafeGround(p.X)
		}
	}
	g.beginFade(screenGameplay, fadePurposeScreen)
	return true
}

func newZombieMatch(cfg zombieSetupConfig, arena *zombieArenaDef, players []*Player, record ZombieRecord) *zombieMatch {
	m := &zombieMatch{
		host:     true,
		arena:    arena,
		nextID:   zombieIDBase,
		rng:      rand.New(rand.NewSource(rand.Int63())),
		navs:     make(map[int]*zombieNav),
		passZone: -1,
	}
	m.hud = ZombieHUD{Active: true, Mode: cfg.Mode, Arena: cfg.Arena, Wave: 1, Phase: zombiePhaseIntro, PhaseTimer: zombieIntroFrames,
		BannerFrame: 1, RecordWave: record.BestWave, RecordScore: record.BestScore}
	if cfg.Mode == zombieModeSurvival {
		m.hud.WaveTotal = zombieSurvivalWaves
	}
	humans := 0
	for _, p := range players {
		if p != nil && !p.IsDouble {
			humans++
		}
	}
	for _, p := range players {
		if p == nil || p.ID < 1 || p.ID > 4 {
			continue
		}
		zp := &m.hud.Players[p.ID-1]
		*zp = ZombiePlayerState{ID: p.ID, HP: zombiePlayerHP, MaxHP: zombiePlayerHP, Points: zombieStartPoints, Zone: arena.StartZone}
		if humans == 1 {
			zp.SelfRevives = zombieSoloSelfRevives
		}
		m.players[p.ID-1] = zombieHostPlayer{seenSerial: p.DeathSerial, groundX: p.X, groundY: p.Y}
	}
	m.placeBox(-1)
	return m
}

// zombieJoinPlayer sets up a player that joined an online match in progress.
func (g *Game) zombieJoinPlayer(p *Player) {
	m := g.zombie
	if m == nil || p == nil || p.ID < 1 || p.ID > 4 {
		return
	}
	p.Team = 1
	p.Lives = zombieLives
	p.TotalLives = zombieLives
	zp := &m.hud.Players[p.ID-1]
	if zp.ID != p.ID {
		*zp = ZombiePlayerState{ID: p.ID, HP: zombiePlayerHP, MaxHP: zombiePlayerHP, Points: zombieStartPoints, Zone: m.arena.StartZone}
	}
	// A player who left while DOWN / DEAD comes back standing (the new body
	// is active); points and stats are kept.
	if zp.State != zombiePlayerAlive || zp.HP <= 0 {
		zp.State = zombiePlayerAlive
		zp.HP = zp.MaxHP
		zp.ReviveProgress = 0
		zp.DownTimer = 0
	}
	zp.ShopOpen, zp.Ready = false, false
	zp.Gun, zp.Reserve = 0, 0 // the new body has only its starter pistol
	if !m.zoneUnlocked(zp.Zone) {
		zp.Zone = m.arena.StartZone
	}
	gx, gy := g.zombieSafeGroundIn(zp.Zone, p.X)
	if zp.Zone != m.resident {
		// The new body dropped in on the resident map: put it on its zone.
		placeZombieArrival(p, gx, gy)
	}
	m.players[p.ID-1] = zombieHostPlayer{seenSerial: p.DeathSerial, groundX: gx, groundY: gy, remote: p.RemoteControlled}
	if m.hud.Phase == zombiePhaseBreak && !p.AI {
		zp.ShopOpen = true
	}
}

func (m *zombieMatch) zone() *zombieZoneDef {
	return m.arena.zone(m.arena.StartZone)
}

func (m *zombieMatch) navFor(zone int, arena Map, class int) *zombieNav {
	key := zone*4 + class
	if nav, ok := m.navs[key]; ok && nav.mapNumber == arena.Number {
		return nav
	}
	nav := buildZombieNav(arena, class)
	m.navs[key] = nav
	return nav
}

// --- per-tick: before the player loop --------------------------------------------

// updateZombiePreInput runs before the players move: it masks the game input
// of players whose shop is open and turns DOWN next to the mystery box into
// "use" (without dropping through the platform).
func (g *Game) updateZombiePreInput() {
	if !g.zombieSimulating() {
		return
	}
	m := g.zombie
	for _, p := range g.players {
		if p == nil || p.IsDouble || p.ID < 1 || p.ID > 4 {
			continue
		}
		zp := m.playerState(p.ID)
		hp := m.hostPlayer(p.ID)
		if zp == nil {
			continue
		}
		p.Lives, p.TotalLives = zombieLives, zombieLives
		if hp.doorCooldown > 0 {
			hp.doorCooldown--
		}
		if hp.doorInvulnWait > 0 {
			hp.doorInvulnWait--
		}
		masked := zp.ShopOpen && !p.AI
		hold := false
		var held PlayerInputState
		if !masked && !hp.remote && !p.AI {
			held = capturePlayerInput(p.Controls)
			// The key that closed the shop (GRENADE / ABILITY / SHOOT) is
			// still held: it must not throw, dash or shoot on release.
			hold = m.shop[p.ID-1].holdingKeys(held)
		}
		switch {
		case hold:
			held.Shoot, held.Grenade, held.Ability = false, false, false
			p.RemoteControlled = true
			p.RemoteInput = held
			hp.masked = true
			// SLOW-MO latch: the held shop key must not fire either.
			p.slowMoLatch &^= netInputBitShoot | netInputBitGrenade | netInputBitAbility
		case hp.remote:
			p.RemoteControlled = true
			if masked {
				p.RemoteInput = PlayerInputState{}
				p.slowMoLatch = 0
			}
		case masked:
			p.RemoteControlled = true
			p.RemoteInput = PlayerInputState{}
			p.slowMoLatch = 0
			hp.masked = true
		case hp.masked:
			p.RemoteControlled = false
			hp.masked = false
		}
		if !p.Active || masked {
			hp.downHeld = false
			continue
		}
		_, _, _, down := p.movementInput()
		if down && !p.AI && m.boxInReach(p) {
			p.DownHeld = true // no drop-through while using the box
			if !hp.downHeld {
				g.zombieUseBox(p)
			}
		} else if di := m.doorNear(p); down && !p.AI && di >= 0 {
			p.DownHeld = true // no drop-through at a door either
			if !hp.downHeld {
				g.zombieUseDoor(p, di)
			}
		}
		hp.downHeld = down
	}
}

// --- hit targets and the hit phase ---------------------------------------------

// aiTargetPool is what AI teammates may target: in a zombie match the
// zombies' bodies as well as the players (bots ignore their own team).
func (g *Game) aiTargetPool() []*Player {
	if !g.zombieSimulating() {
		return g.players
	}
	return g.hitTargets()
}

func (g *Game) appendZombieBodies(out []*Player) []*Player {
	if g.zombie == nil || !g.zombie.host {
		return out
	}
	for _, z := range g.zombie.zombies {
		if z.alive() && (g.zombie.passZone < 0 || z.Zone == g.zombie.passZone) {
			out = append(out, &z.body)
		}
	}
	return out
}

func (g *Game) zombieHasBodies() bool {
	if g.zombie == nil || !g.zombie.host {
		return false
	}
	for _, z := range g.zombie.zombies {
		if z.alive() {
			return true
		}
	}
	return false
}

func (g *Game) beginZombieHitPhase() {
	if !g.zombieSimulating() {
		return
	}
	for _, z := range g.zombie.zombies {
		if !z.alive() {
			continue
		}
		z.preVX, z.preVY = z.body.VX, z.body.VY
		z.windVX, z.windVY = 0, 0
		z.preHits = z.body.HitNumber
		z.headshot = false
	}
}

// zombieBulletsAlive lists the bullets that can still hit (headshot check).
func (g *Game) zombieBulletsAlive(dst []int) []int {
	dst = dst[:0]
	if !g.zombieHasBodies() {
		return dst
	}
	for i := range g.bullets {
		if !g.bullets[i].Dead && g.bullets[i].DevelopedProjectile == 0 {
			dst = append(dst, i)
		}
	}
	return dst
}

// markZombieHeadshots: a bullet that died this frame inside the top part of
// a zombie's hitbox hit the head (bullets stop where they hit).
func (g *Game) markZombieHeadshots(alive []int) {
	if len(alive) == 0 || g.zombie == nil {
		return
	}
	for _, i := range alive {
		if i >= len(g.bullets) || !g.bullets[i].Dead {
			continue
		}
		b := &g.bullets[i]
		if b.OwnerID < 1 {
			continue
		}
		for _, z := range g.zombie.zombies {
			if !z.alive() {
				continue
			}
			h := z.body.Hitbox()
			head := Rect{X: h.X - 3, Y: h.Y - 4, W: h.W + 6, H: h.H*zombieHeadFraction + 4}
			if head.Contains(b.X, b.Y) {
				if !z.headshot {
					z.headshot = true
					z.applyDamage(b.Firepower*zombieHeadshotShare*zombieBlastDamageScale, b.OwnerID)
				}
				break
			}
		}
	}
}

// pendingDamage accumulates headshot bonus damage; applied in the end phase.
func (z *Zombie) applyDamage(amount float64, ownerID int) {
	z.bonusDamage += amount
	if ownerID > 0 {
		z.creditID = ownerID
		z.creditTimer = zombieCreditFrames
	}
}

// endZombieHitPhase turns this frame's hits into HP damage and points. The
// impulse is divided by the zombie's knockback share, so a knockback-resistant
// brute still takes full damage.
func (g *Game) endZombieHitPhase() {
	if !g.zombieSimulating() {
		return
	}
	m := g.zombie
	for _, z := range m.zombies {
		if !z.alive() {
			continue
		}
		b := &z.body
		hits := max(0, b.HitNumber-z.preHits)
		scale := abilityKnockbackScale(b)
		if scale <= 0.001 {
			scale = 0.001
		}
		impulse := math.Hypot(b.VX-z.preVX-z.windVX*scale, b.VY-z.preVY-z.windVY*scale)
		z.windVX, z.windVY = 0, 0
		if hits > 0 || impulse > 0.5 {
			z.HitFlash = 5
			if b.LastHitBy > 0 && b.LastHitBy < zombieIDBase {
				z.creditID = b.LastHitBy
				z.creditTimer = zombieCreditFrames
			}
		}
		damage := impulse / scale * zombieBlastDamageScale
		if hits > 0 && damage < float64(hits)*4 {
			damage = float64(hits) * 4
		}
		damage += z.bonusDamage
		z.bonusDamage = 0
		if b.BurnTime > 0 {
			z.burnTick++
			if z.burnTick%6 == 0 {
				damage += 1
				if b.BurnOwnerID > 0 && b.BurnOwnerID < zombieIDBase {
					z.creditID = b.BurnOwnerID
					z.creditTimer = zombieCreditFrames
				}
			}
		}
		if hits > 0 && z.creditID > 0 {
			g.zombieAward(z.creditID, zombiePointsHit*hits, zombiePopHit, b.X, b.Y-60)
		}
		if damage < 0.5 {
			continue
		}
		g.zombieDamage(z, int(math.Round(damage)), z.creditID, z.headshot)
	}
}

// zombieDamage removes HP and kills the zombie at 0.
func (g *Game) zombieDamage(z *Zombie, damage, creditID int, headshot bool) {
	if z == nil || z.Dead || damage <= 0 {
		return
	}
	z.HP -= damage
	z.HitFlash = 5
	if z.HP > 0 {
		return
	}
	z.HP = 0
	g.zombieKill(z, creditID, false, headshot)
}

// zombieKill finishes a zombie: points, effects, exploder blast.
func (g *Game) zombieKill(z *Zombie, creditID int, ringOut, headshot bool) {
	if z == nil || z.finished {
		return
	}
	m := g.zombie
	z.finished = true
	z.Dead = true
	z.RingOut = ringOut
	z.DeadFrames = 0
	z.X, z.Y = z.body.X, z.body.Y
	b := &z.body
	if b.Active {
		b.DeathSerial++
		b.Active = false
	}
	m.hud.Kills++
	k := z.kind()
	if owner := g.zombieCreditOwner(creditID); owner != nil {
		pts := k.stats().KillPoints
		kind := zombiePopKill
		if ringOut {
			pts += zombiePointsRingOutBonus
			kind = zombiePopRingOut
		} else if headshot {
			pts += zombiePointsHeadshotBonus
			kind = zombiePopHeadshot
		}
		g.zombieAward(owner.ID, pts, kind, z.X, z.Y-80*z.body.PlayerScale)
		if zp := m.playerState(owner.ID); zp != nil {
			zp.Kills++
			if headshot && !ringOut {
				zp.Headshots++
			}
		}
		owner.Kills++
	}
	if k == zombieExploder && !ringOut {
		delay := 1
		if z.chained {
			delay = zombieExploderChain
		}
		m.blasts = append(m.blasts, zombiePendingBlast{x: z.X, y: z.Y, timer: delay, ownerID: creditID, zone: z.Zone})
	} else if !ringOut {
		g.playRandomDeathSound()
	}
	if k == zombieBoss {
		m.hud.BossHP = 0
		g.explosions = append(g.explosions, Explosion{X: z.X, Y: z.Y - 60, Life: 18})
		g.playRandomExplosionSound()
	}
}

func (g *Game) zombieCreditOwner(id int) *Player {
	if id <= 0 || id >= zombieIDBase {
		return nil
	}
	// Every player of the match: during a zone pass g.players holds only
	// that zone, but a grenade, a burn or a chain can credit a shooter who
	// has gone through a door since.
	byID := func(id int) *Player {
		for _, p := range g.zombiePlayersAll() {
			if p != nil && p.ID == id {
				return p
			}
		}
		return nil
	}
	p := byID(id)
	if p == nil {
		return nil
	}
	if p.IsDouble && p.OwnerPlayerID > 0 {
		p = byID(p.OwnerPlayerID)
	}
	return p
}

// zombieAward gives points to a player (by any ID it owns) with a popup.
func (g *Game) zombieAward(id, amount, kind int, x, y float64) {
	owner := g.zombieCreditOwner(id)
	if owner == nil || amount == 0 {
		return
	}
	zp := g.zombie.playerState(owner.ID)
	if zp == nil {
		return
	}
	zp.Points += amount
	if amount > 0 {
		zp.Earned += amount
		owner.Score += amount
	}
	// The popup shows where the points were earned: the zone being
	// simulated (the zombie's), else the owner's zone.
	zone := zp.Zone
	if g.zombie.passZone >= 0 {
		zone = g.zombie.passZone
	}
	g.zombie.addPopIn(zone, x, y, amount, kind)
}

func (m *zombieMatch) addPop(x, y float64, amount, kind int) {
	zone := m.viewZone
	if m.passZone >= 0 {
		zone = m.passZone
	}
	m.addPopIn(zone, x, y, amount, kind)
}

// addPopIn adds a popup in a zone (multi-zone matches show it only there).
func (m *zombieMatch) addPopIn(zone int, x, y float64, amount, kind int) {
	if kind == zombiePopHit {
		// Merge hit popups at the same spot into one growing number.
		for i := len(m.pops) - 1; i >= 0 && i >= len(m.pops)-6; i-- {
			p := &m.pops[i]
			if p.Kind == zombiePopHit && p.Zone == zone && p.Life > zombiePopLife-10 && math.Abs(p.X-x) < 40 && math.Abs(p.Y-y) < 40 {
				p.Amount += amount
				return
			}
		}
	}
	m.pops = append(m.pops, ZombiePop{X: x, Y: y, Amount: amount, Kind: kind, Life: zombiePopLife, Zone: zone})
	if len(m.pops) > zombiePopCap {
		m.pops = append(m.pops[:0], m.pops[len(m.pops)-zombiePopCap:]...)
	}
}

// --- per-tick: after the hit phase -----------------------------------------------

// updateZombieMatch runs the player rules (HP, down, revive), the wave phases
// and the popups. It is called once per gameplay tick after the hit phase.
func (g *Game) updateZombieMatch() {
	if !g.zombieSimulating() {
		return
	}
	m := g.zombie
	m.updatePops()
	if m.hud.BannerFrame > 0 {
		m.hud.BannerFrame++
		if m.hud.BannerFrame > 120 {
			m.hud.BannerFrame = 0
		}
	}
	if !m.multiZone() {
		// Multi-zone: runZombieZoneTick already ran it per zone.
		g.updateZombiePlayers()
	}
	g.updateZombieBox()
	g.updateZombieDoors()
	switch m.hud.Phase {
	case zombiePhaseIntro:
		m.hud.PhaseTimer--
		if m.hud.PhaseTimer <= 0 {
			g.beginZombieWave()
		}
	case zombiePhaseWave:
		if m.toSpawn <= 0 && m.livingZombies() == 0 && len(m.blasts) == 0 {
			g.endZombieWave()
		}
	case zombiePhaseBreak:
		g.updateZombieBreak()
	case zombiePhaseOver, zombiePhaseWon:
		m.hud.PhaseTimer--
		if m.hud.PhaseTimer <= 0 && !m.ended {
			g.finishZombieMatch()
		}
	}
	if m.hud.Phase == zombiePhaseWave || m.hud.Phase == zombiePhaseIntro || m.hud.Phase == zombiePhaseBreak {
		if g.zombieEveryoneDown() {
			m.hud.Phase = zombiePhaseOver
			m.hud.PhaseTimer = zombieEndFrames
			m.closeShops()
		}
	}
	m.hud.Remaining = m.toSpawn + m.livingZombies()
	m.hud.BossID, m.hud.BossHP, m.hud.BossMaxHP = 0, 0, 0
	for _, z := range m.zombies {
		if z.kind() == zombieBoss && !z.Dead {
			m.hud.BossID, m.hud.BossHP, m.hud.BossMaxHP, m.hud.BossLevel = z.ID, z.HP, z.MaxHP, z.Level
		}
	}
}

func (m *zombieMatch) updatePops() {
	out := m.pops[:0]
	for _, p := range m.pops {
		p.Life--
		p.Y -= 0.9
		if p.Life > 0 {
			out = append(out, p)
		}
	}
	m.pops = out
}

func (m *zombieMatch) livingZombies() int {
	n := 0
	for _, z := range m.zombies {
		if !z.Dead {
			n++
		}
	}
	return n
}

func (g *Game) zombieEveryoneDown() bool {
	m := g.zombie
	anyone := false
	for _, p := range g.players {
		if p == nil || p.IsDouble {
			continue
		}
		zp := m.playerState(p.ID)
		if zp == nil {
			continue
		}
		anyone = true
		if zp.State == zombiePlayerAlive || (zp.State == zombiePlayerDown && zp.SelfRevives > 0) {
			return false
		}
	}
	return anyone
}

func (m *zombieMatch) closeShops() {
	for i := range m.hud.Players {
		m.hud.Players[i].ShopOpen = false
		m.hud.Players[i].Ready = false
	}
}

func (g *Game) beginZombieWave() {
	m := g.zombie
	m.closeShops()
	m.plan = zombieWavePlanFor(m.hud.Wave, g.zombieParticipants())
	m.toSpawn = m.plan.Count
	m.spawnTimer = 20
	m.hud.Phase = zombiePhaseWave
	m.hud.PhaseTimer = 0
	if m.plan.Boss {
		g.spawnZombieBoss()
	}
}

func (g *Game) zombieParticipants() int {
	n := 0
	for _, p := range g.players {
		if p != nil && !p.IsDouble && p.ID >= 1 && p.ID <= 4 {
			n++
		}
	}
	return max(n, 1)
}

func (g *Game) endZombieWave() {
	m := g.zombie
	for _, p := range g.players {
		if p == nil || p.IsDouble {
			continue
		}
		if zp := m.playerState(p.ID); zp != nil && zp.State == zombiePlayerAlive {
			g.zombieAward(p.ID, zombiePointsWaveClear, zombiePopKill, p.X, p.Y-90)
		}
	}
	if m.hud.Mode == zombieModeSurvival && m.hud.Wave >= zombieSurvivalWaves {
		m.hud.Phase = zombiePhaseWon
		m.hud.PhaseTimer = zombieEndFrames
		return
	}
	// The wave is over: downed players get up, the dead come back.
	for _, p := range g.players {
		if p == nil || p.IsDouble {
			continue
		}
		zp := m.playerState(p.ID)
		if zp == nil {
			continue
		}
		switch zp.State {
		case zombiePlayerDown:
			g.zombieRevive(p, zp, zombieReviveHP)
		case zombiePlayerDead:
			g.zombieRespawnDead(p, zp, zp.MaxHP)
		}
	}
	m.hud.Phase = zombiePhaseBreak
	m.hud.PhaseTimer = zombieBreakFrames
	for _, p := range g.players {
		if p == nil || p.IsDouble {
			continue
		}
		zp := m.playerState(p.ID)
		if zp == nil {
			continue
		}
		zp.Ready = p.AI
		zp.ShopOpen = !p.AI
		if p.AI {
			g.zombieBotShop(p, zp)
		}
	}
	m.blasts = m.blasts[:0]
}

func (g *Game) updateZombieBreak() {
	m := g.zombie
	m.hud.PhaseTimer--
	allReady := true
	for _, p := range g.players {
		if p == nil || p.IsDouble {
			continue
		}
		if zp := m.playerState(p.ID); zp != nil && !zp.Ready {
			allReady = false
		}
	}
	if allReady && m.hud.PhaseTimer > zombieReadyCountdown {
		m.hud.PhaseTimer = zombieReadyCountdown
	}
	if m.hud.PhaseTimer <= 0 {
		m.hud.Wave++
		m.hud.Phase = zombiePhaseIntro
		m.hud.PhaseTimer = zombieIntroFrames
		m.hud.BannerFrame = 1
		m.closeShops()
	}
}

// --- players: HP, down, revive -----------------------------------------------------

func (g *Game) updateZombiePlayers() {
	m := g.zombie
	for _, p := range g.players {
		if p == nil || p.IsDouble || p.ID < 1 || p.ID > 4 {
			continue
		}
		zp := m.playerState(p.ID)
		hp := m.hostPlayer(p.ID)
		if zp == nil {
			continue
		}
		if zp.Invuln > 0 {
			zp.Invuln--
		}
		if zp.MsgTimer > 0 {
			zp.MsgTimer--
		}
		switch zp.State {
		case zombiePlayerAlive:
			if m.hud.Phase != zombiePhaseWave && (p.DeathSerial != hp.seenSerial || zp.HP <= 0) {
				// No zombies around (intro, break, end): a fall is not a down,
				// the player just drops back in.
				zp.HP = max(zp.HP, zombieReviveHP)
				g.zombieRespawnDead(p, zp, zp.HP)
				continue
			}
			if p.DeathSerial != hp.seenSerial {
				// Ring-out (Player.Kill respawned the body): down at the edge
				// the player last stood on.
				g.zombieDownPlayer(p, zp, hp, hp.groundX, hp.groundY, false)
				continue
			}
			if zp.HP <= 0 {
				// Down on the platform under the player (in the air: the one
				// below; above the void: where they last stood).
				x, y := hp.groundX, hp.groundY
				if top, ok := zombieFloorBelow(g.arena, p.X, p.Y-2); ok {
					x, y = p.X, top
				}
				g.zombieDownPlayer(p, zp, hp, x, y, true)
				continue
			}
			if p.Active && p.Grounded && p.JumpNum == 2 {
				hp.groundX, hp.groundY = p.X, p.Y
			}
			hp.loadout = zombieLoadout{valid: true, weapon: p.Weapon, grenades: p.Grenades, throwable: p.ThrowableType}
			g.zombieKeepBoughtGun(p, zp)
		case zombiePlayerDown:
			g.updateZombieDowned(p, zp, hp)
		}
	}
}

// zombieKeepBoughtGun: an emptied bought weapon falls back to the starter
// pistol (source rule); a spare magazine re-equips it.
func (g *Game) zombieKeepBoughtGun(p *Player, zp *ZombiePlayerState) {
	if zp.Gun == 0 || !p.Active {
		return
	}
	if p.Weapon.Def.Number == zp.Gun {
		return
	}
	if p.Weapon.Def.Number != p.DefaultWeapon {
		return
	}
	if zp.Reserve > 0 {
		zp.Reserve--
		p.EquipWeapon(zp.Gun)
		p.PendingSounds = append(p.PendingSounds, "pistol_mag.wav")
	}
	// Out of magazines: the gun stays "owned" (AMMO REFILL re-equips it).
}

// zombieDownPlayer puts a player DOWN at (x, y). effects: play the death
// sound and burst (a ring-out already got them from the player loop).
func (g *Game) zombieDownPlayer(p *Player, zp *ZombiePlayerState, hp *zombieHostPlayer, x, y float64, effects bool) {
	fellX, fellY := p.X, p.Y
	zp.State = zombiePlayerDown
	zp.HP = 0
	zp.DownX, zp.DownY = x, y
	zp.DownTimer = zombieBleedFrames
	zp.ReviveProgress = 0
	zp.Downs++
	zp.ShopOpen = false
	p.Active = false
	p.VX, p.VY = 0, 0
	p.X, p.Y = x, y
	p.clearBurn()
	clearBlind(p)
	p.TearGasTime = 0
	p.GrenadeHeld = false
	p.GrenadePower = 0
	hp.seenSerial = p.DeathSerial
	hp.reviverID = 0
	if effects {
		g.playRandomDeathSound()
		g.explosions = append(g.explosions, Explosion{X: fellX, Y: fellY - 20, Life: 18})
	}
}

func (g *Game) updateZombieDowned(p *Player, zp *ZombiePlayerState, hp *zombieHostPlayer) {
	reviver := 0
	for _, o := range g.players {
		if o == nil || o == p || !o.Active || o.ID < 1 {
			continue
		}
		ozp := g.zombie.playerState(o.ID)
		if o.IsDouble || ozp == nil || ozp.State != zombiePlayerAlive {
			continue
		}
		if math.Abs(o.X-zp.DownX) <= zombieReviveRadiusX && math.Abs(o.Y-zp.DownY) <= zombieReviveRadiusY {
			reviver = o.ID
			break
		}
	}
	switch {
	case reviver != 0:
		if hp.reviverID != reviver {
			zp.ReviveProgress = 0
			hp.reviverID = reviver
		}
		zp.ReviveProgress++
	case zp.SelfRevives > 0:
		hp.reviverID = p.ID
		zp.ReviveProgress++
	default:
		hp.reviverID = 0
		zp.ReviveProgress = 0
		zp.DownTimer--
	}
	if zp.ReviveProgress >= zombieReviveFrames {
		if hp.reviverID == p.ID {
			zp.SelfRevives--
		} else if r := g.zombie.playerState(hp.reviverID); r != nil {
			r.Revives++
			g.zombieAward(hp.reviverID, zombiePointsRevive, zombiePopRevive, zp.DownX, zp.DownY-80)
		}
		g.zombieRevive(p, zp, zombieReviveHP)
		return
	}
	if zp.DownTimer <= 0 {
		zp.State = zombiePlayerDead
		zp.ReviveProgress = 0
	}
}

// zombieRevive brings a downed player back where they fell, with the
// weapons they had.
func (g *Game) zombieRevive(p *Player, zp *ZombiePlayerState, hpValue int) {
	hp := g.zombie.hostPlayer(p.ID)
	p.Active = true
	p.Alpha = 1
	p.X, p.Y = zp.DownX, zp.DownY-2
	p.VX, p.VY = 0, 0
	p.JumpNum = 1
	p.Grounded = false
	p.FreePass = false
	p.LastHitBy = 0
	p.HitNumber = 0
	p.InvisibleTime = 0
	p.resetAbilityForLife()
	if hp != nil && hp.loadout.valid {
		p.Weapon = hp.loadout.weapon
		p.Grenades = hp.loadout.grenades
		p.ThrowableType = hp.loadout.throwable
	}
	zp.State = zombiePlayerAlive
	zp.HP = min(max(hpValue, 1), zp.MaxHP)
	zp.Invuln = zombieInvulnFrames
	zp.ReviveProgress = 0
	if hp != nil {
		hp.seenSerial = p.DeathSerial
		hp.reviverID = 0
		hp.groundX, hp.groundY = zp.DownX, zp.DownY
	}
	g.playSourceSFX("emp.wav", false)
}

// zombieRespawnDead brings a bled-out player back at the start of a break,
// dropping in from the top like a normal respawn.
func (g *Game) zombieRespawnDead(p *Player, zp *ZombiePlayerState, hpValue int) {
	mp := g.zombieZoneMap(zp.Zone)
	x, y := g.zombieSafeGroundIn(zp.Zone, (mp.SpawnMinX+mp.SpawnMaxX)/2)
	zp.DownX, zp.DownY = x, y
	g.zombieRevive(p, zp, hpValue)
}

// zombieSafeGround is a down position on the highest platform under x (a
// player who falls before ever landing is downed there, not in the sky).
func (g *Game) zombieSafeGround(x float64) (float64, float64) {
	return safeGroundOn(g.arena, x)
}

// zombieSafeGroundIn is zombieSafeGround on the map of a zone.
func (g *Game) zombieSafeGroundIn(zone int, x float64) (float64, float64) {
	return safeGroundOn(g.zombieZoneMap(zone), x)
}

func safeGroundOn(arena Map, x float64) (float64, float64) {
	if y, ok := zombieFloorBelow(arena, x, -2000); ok {
		return x, y
	}
	cx := (arena.SpawnMinX + arena.SpawnMaxX) / 2
	y, _ := zombieFloorBelow(arena, cx, -2000)
	return cx, y
}

// placeZombieArrival stands a player still on a platform top.
func placeZombieArrival(p *Player, x, y float64) {
	p.X, p.Y, p.VX, p.VY = x, y, 0, 0
	p.JumpNum, p.Grounded, p.FreePass = 2, true, false
}

// zombieFloorBelow is the top of the first platform at or below y at x.
// When no platform is there it returns the arena's lowest platform line (a
// finite fallback) and false.
func zombieFloorBelow(arena Map, x, y float64) (float64, bool) {
	best := math.Inf(1)
	for _, r := range arena.Platforms {
		if x >= r.X && x <= r.X+r.W && r.Y >= y && r.Y < best {
			best = r.Y
		}
	}
	if math.IsInf(best, 1) {
		return arena.LowestY, false
	}
	return best, true
}

// zombieHurtPlayer applies zombie damage to a player's HP and the knockback
// (the caller runs it inside the players' knockback window, so HEAVY, ANCHOR
// and SHIELD apply to the push). SHIELD also blocks the damage of that hit.
func (g *Game) zombieHurtPlayer(t *Player, damage int, pushX, lift float64, attackerID int) bool {
	zp := g.zombie.playerState(t.ID)
	if t.IsDouble && t.OwnerPlayerID > 0 {
		// Helper doubles take the push but have no HP bar.
		zp = nil
	}
	if zp != nil && (zp.State != zombiePlayerAlive || zp.Invuln > 0) {
		return false
	}
	t.VX += pushX
	t.VY -= lift
	t.HitNumber++
	t.HitTimer = 0
	t.LastHitBy = attackerID
	t.HitByGrenade = false
	if zp != nil && damage > 0 && !abilityShieldReady(t) {
		d := float64(damage)
		if t.ShieldTime > 0 {
			d *= 0.2
		}
		zp.HP -= int(math.Ceil(d))
	}
	g.playRandomHitSound()
	return true
}

// --- end of the match ---------------------------------------------------------------

func (g *Game) finishZombieMatch() {
	m := g.zombie
	if m == nil || m.ended {
		return
	}
	m.ended = true
	m.closeShops()
	g.saveZombieRecord()
	g.zombieUI.reset()
	g.beginFade(screenZombieResults, fadePurposeScreen)
}

// zombieBestScore is the best single-player score of the match.
func (m *zombieMatch) teamScore() int {
	total := 0
	for _, zp := range m.hud.Players {
		if zp.ID != 0 {
			total += zp.Earned
		}
	}
	return total
}

// zombieWavesSurvived: waves fully cleared.
func (m *zombieMatch) wavesSurvived() int {
	switch m.hud.Phase {
	case zombiePhaseWon:
		return m.hud.Wave
	case zombiePhaseBreak:
		return m.hud.Wave
	}
	return max(0, m.hud.Wave-1)
}
