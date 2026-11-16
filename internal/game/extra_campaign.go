package game

import (
	"strconv"
)

const (
	extraCampaignMissionCount = 28
	extraCampaignPageSize     = 12
)

var extraCampaignTitles = [extraCampaignMissionCount]string{
	"Reinforcement Run",
	"Close Quarters",
	"Eastern Front",
	"Sniper Nest",
	"Rocket Rain",
	"Burning Point",
	"Heavy Metal",
	"Last Stand",
	"Red Sky",
	"Old Guard",
	"Juggernaut",
	"Portal Storm",
	"Gravity Break",
	"Knockback Lab",
	"Arms Roulette",
	"Bullet Furnace",
	"Gravity Well",
	"Bombardment",
	"Shock Reactor",
	"Mayhem Crown",
	"Rotor Wash",
	"Swarm Front",
	"Riot Line",
	"Breach Point",
	"Combined Arms",
	"Iron Stride",
	"Steel Rain",
	"Gunship",
}

type extraCampaignEnemy struct {
	ID    int
	Name  string
	Color int
	Shirt int
	Hat   int
	Gun   int
	Perk  int
	Lives int
	// ShotDelay is a forced pause (frames) between trigger pulls while the
	// bot holds its mission gun. Zero keeps the weapon's own rate of fire.
	ShotDelay int
	// Boss turns this bot into a real boss: knockback resistance, phases
	// and the boss HUD bar (extra_campaign_boss.go).
	Boss *extraCampaignBoss
}

// xe builds an EXTRA bot in the compact order the mission table uses.
func xe(id int, name string, colour, shirt, hat, gun, perk, lives int) extraCampaignEnemy {
	return extraCampaignEnemy{ID: id, Name: name, Color: colour, Shirt: shirt, Hat: hat, Gun: gun, Perk: perk, Lives: lives}
}

func (e extraCampaignEnemy) withShotDelay(frames int) extraCampaignEnemy {
	e.ShotDelay = frames
	return e
}

func (e extraCampaignEnemy) asBoss(b *extraCampaignBoss) extraCampaignEnemy {
	e.Boss = b
	return e
}

type extraCampaignMission struct {
	MapNumber   int
	PlayerLives int
	Crates      bool
	Powerups    bool
	Mode        string
	Briefing    string
	RuleText    string
	Modifiers   []int
	Hazard      extraCampaignHazard
	// HazardInterval overrides the hazard's default period (frames).
	HazardInterval int
	Enemies        []extraCampaignEnemy
	// Units are non-player campaign enemies (drones, shield-bearers) spawned
	// in waves; every one of them must be defeated to win.
	Units []extraCampaignUnit
	// CratePool replaces the developed crate pool (the gunship finale drops
	// an anti-air kit).
	CratePool []int
}

// extraAntiAirCrates: bats and blowers send gunship rockets back, the
// harpoon drags it down, launchers and rifles hit it when it is low.
var extraAntiAirCrates = []int{weaponBaseballBat, weaponBaseballBat, weaponLeafBlower, weaponHarpoonGun, weaponHarpoonGun, 87, 168, 11, 10, 19}

// EXTRA is an original continuation built for this port. Enemy life pools are
// deliberately independent from the human life count: later missions contain
// many reinforcements without putting more than two readable opponents on the
// arena at once.
var extraCampaignMissions = [extraCampaignMissionCount]extraCampaignMission{
	{MapNumber: 8, PlayerLives: 6, Crates: true, Powerups: true, Mode: "REINFORCEMENT SQUAD", Briefing: "Two fast enemies control both sides.", Enemies: []extraCampaignEnemy{
		xe(3, "Rapid", 3, 4, 9, 99, 0, 4), xe(4, "Scout", 8, 2, 3, 91, 0, 4),
	}},
	{MapNumber: 11, PlayerLives: 6, Crates: true, Powerups: true, Mode: "SHOTGUN DUEL", Briefing: "One KS-23 hunter with a large life pool.", Enemies: []extraCampaignEnemy{
		xe(3, "Hunter", 9, 8, 15, 165, 0, 9),
	}},
	{MapNumber: 9, PlayerLives: 6, Crates: true, Powerups: false, Mode: "EASTERN SQUAD", Briefing: "Automatic rifles attack from two angles in the gusts.", RuleText: "CROSSWIND GUSTS", Hazard: extraCampaignHazardWind, HazardInterval: 540, Enemies: []extraCampaignEnemy{
		xe(3, "Red Star", 2, 14, 4, 96, 0, 5), xe(4, "Comrade", 10, 6, 31, 131, 7, 5),
	}},
	{MapNumber: 12, PlayerLives: 5, Crates: false, Powerups: true, Mode: "SNIPER CROSS-FIRE", Briefing: "Hard but fair: break their firing lanes.", Enemies: []extraCampaignEnemy{
		xe(3, "Marksman", 4, 15, 41, 151, 0, 4), xe(4, "Ghost", 7, 5, 8, 158, 4, 4),
	}},
	{MapNumber: 5, PlayerLives: 7, Crates: false, Powerups: true, Mode: "EXPLOSIVE SQUAD", Briefing: "Keep moving between rocket impacts and shell markers.", RuleText: "DYNAMITE RAIN (SLOW)", Hazard: extraCampaignHazardDynamiteRain, HazardInterval: 170, Enemies: []extraCampaignEnemy{
		xe(3, "Rocketeer", 5, 9, 10, 168, 2, 4), xe(4, "Grenadier", 1, 11, 17, 173, 8, 4),
	}},
	{MapNumber: 6, PlayerLives: 6, Crates: true, Powerups: true, Mode: "FIRE TEAM", Briefing: "Flares ignite; the flamethrower closes in.", Enemies: []extraCampaignEnemy{
		xe(3, "Firestarter", 8, 13, 42, 181, 5, 5).withShotDelay(90), xe(4, "Torch", 3, 12, 1, 187, 0, 5),
	}},
	{MapNumber: 3, PlayerLives: 7, Crates: true, Powerups: false, Mode: "HEAVY BOSS", Briefing: "One MG42 gunner with thirteen lives.", Enemies: []extraCampaignEnemy{
		xe(3, "Gunner", 9, 4, 9, 141, 2, 13).asBoss(&extraCampaignBoss{Title: "HEAVY GUNNER", KnockbackScale: 0.8, Phases: []extraCampaignBossPhase{
			{Lives: 8, Banner: "BELT FED - MOVING UP", MoveScale: 1.15},
			{Lives: 4, Banner: "LAST BELT - DRONE SUPPORT", SummonDrones: 1, KnockbackScale: 0.7},
		}}),
	}},
	{MapNumber: 1, PlayerLives: 8, Crates: false, Powerups: true, Mode: "FINAL SQUAD", Briefing: "Grenades and energy fire demand movement.", Enemies: []extraCampaignEnemy{
		xe(3, "Warden", 10, 8, 15, 175, 2, 8), xe(4, "Overseer", 1, 14, 41, 185, 5, 8),
	}},
	{MapNumber: 4, PlayerLives: 7, Crates: false, Powerups: true, Mode: "AIRSTRIKE BOSS", Briefing: "Watch the warning marker before each strike.", Enemies: []extraCampaignEnemy{
		// The airstrike boss announces roughly one strike every 2.25 seconds.
		xe(3, "Sky Marshal", 4, 9, 10, 90, 6, 10).withShotDelay(135).asBoss(&extraCampaignBoss{Title: "AIRSTRIKE COMMANDER", KnockbackScale: 0.85, Phases: []extraCampaignBossPhase{
			{Lives: 6, Banner: "STRIKES QUICKEN", ShotDelay: 105},
			{Lives: 3, Banner: "SPOTTER DRONES", ShotDelay: 85, SummonDrones: 2},
		}}),
	}},
	{MapNumber: 10, PlayerLives: 6, Crates: true, Powerups: false, Mode: "OLD-SCHOOL SQUAD", Briefing: "Slow weapons, powerful single hits.", Enemies: []extraCampaignEnemy{
		xe(3, "Musketeer", 6, 7, 31, 182, 0, 5), xe(4, "Ranger", 2, 3, 15, 177, 3, 5),
	}},
	{MapNumber: 7, PlayerLives: 7, Crates: true, Powerups: true, Mode: "JUGGERNAUT DUEL", Briefing: "A blast-shielded harpoon boss reels you in over the gaps.", Enemies: []extraCampaignEnemy{
		// The harpoon reels its victim in (arsenal2_combat.go), so the duel is
		// built around dodging deliberate shots near the gaps.
		xe(3, "Juggernaut", 10, 14, 42, 179, 6, 13).withShotDelay(75).asBoss(&extraCampaignBoss{Title: "JUGGERNAUT", KnockbackScale: 0.65, Phases: []extraCampaignBossPhase{
			{Lives: 8, Banner: "ARMOR CRACKED", KnockbackScale: 0.8, MoveScale: 1.1},
			{Lives: 4, Banner: "ENRAGED", KnockbackScale: 0.95, MoveScale: 1.25, ShotDelay: 55},
		}}),
	}},
	{MapNumber: 2, PlayerLives: 8, Crates: false, Powerups: true, Mode: "PORTAL SUPPORT", Briefing: "Portals reposition you into energy fire.", Enemies: []extraCampaignEnemy{
		xe(3, "Portal Tech", 5, 5, 8, 189, 3, 7), xe(4, "Storm", 1, 12, 41, 185, 5, 7),
	}},
	{MapNumber: 8, PlayerLives: 6, Crates: true, Powerups: false, Mode: "ZERO-G PURSUIT", Briefing: "Fight through the air; fuel never runs out.", RuleText: "LOW GRAVITY + JETPACKS", Modifiers: []int{customModifierLowGravity, customModifierJetpacks}, Enemies: []extraCampaignEnemy{
		xe(3, "Sky Raider", 4, 11, 10, 213, 7, 5), xe(4, "Drifter", 8, 5, 17, 212, 0, 5),
	}},
	{MapNumber: 11, PlayerLives: 6, Crates: false, Powerups: true, Mode: "IMPACT TEST", Briefing: "Every hit launches twice as hard.", RuleText: "DOUBLE KNOCKBACK", Modifiers: []int{customModifierDoubleKnockback}, Enemies: []extraCampaignEnemy{
		xe(3, "Ram", 9, 8, 15, 167, 6, 11),
	}},
	{MapNumber: 9, PlayerLives: 7, Crates: true, Powerups: false, Mode: "WEAPON ROULETTE", Briefing: "Everyone receives a new weapon after each return.", RuleText: "RANDOM WEAPON / LIFE + FAST CRATES", Modifiers: []int{customModifierRandomWeaponEachLife, customModifierCratesEverywhere}, Enemies: []extraCampaignEnemy{
		xe(3, "Wildcard", 2, 13, 31, 208, 0, 6), xe(4, "Dealer", 10, 4, 42, 204, 3, 6),
	}},
	{MapNumber: 3, PlayerLives: 7, Crates: false, Powerups: true, Mode: "ENDLESS BELTS", Briefing: "The Maxim never reloads and never slows down.", RuleText: "INFINITE AMMO + FAST MOVEMENT", Modifiers: []int{customModifierInfiniteAmmo, customModifierFastMovement}, Enemies: []extraCampaignEnemy{
		xe(3, "Furnace", 5, 14, 9, 199, 2, 14).asBoss(&extraCampaignBoss{Title: "THE FURNACE", KnockbackScale: 0.8, Phases: []extraCampaignBossPhase{
			{Lives: 9, Banner: "OVERHEATING", MoveScale: 1.1},
			{Lives: 4, Banner: "MELTDOWN", KnockbackScale: 0.95, SummonDrones: 1},
		}}),
	}},
	{MapNumber: 12, PlayerLives: 6, Crates: true, Powerups: false, Mode: "HEAVY GRAVITY", Briefing: "Short arcs and boosted jumps change every route.", RuleText: "HIGH GRAVITY + SUPER JUMP", Modifiers: []int{customModifierHighGravity, customModifierSuperJump}, Enemies: []extraCampaignEnemy{
		xe(3, "Anchor", 6, 7, 4, 208, 6, 5), xe(4, "Weight", 3, 3, 15, 210, 0, 5),
	}},
	{MapNumber: 4, PlayerLives: 7, Crates: false, Powerups: true, Mode: "DYNAMITE WEATHER", Briefing: "Watch the warning: neutral dynamite falls from above.", RuleText: "DYNAMITE RAIN", Hazard: extraCampaignHazardDynamiteRain, Enemies: []extraCampaignEnemy{
		xe(3, "Spotter", 4, 9, 10, 169, 2, 5), xe(4, "Sapper", 1, 11, 17, 173, 8, 5),
	}},
	{MapNumber: 2, PlayerLives: 7, Crates: true, Powerups: true, Mode: "REACTOR PULSE", Briefing: "The reactor periodically throws everyone from its core.", RuleText: "CENTER SHOCK PULSE", Hazard: extraCampaignHazardShockPulse, Enemies: []extraCampaignEnemy{
		xe(3, "Conductor", 5, 5, 8, 189, 3, 6), xe(4, "Voltage", 1, 12, 41, 185, 5, 6),
	}},
	{MapNumber: 7, PlayerLives: 9, Crates: true, Powerups: true, Mode: "FINAL CHAOS", Briefing: "New weapons each life in low gravity with brutal impacts.", RuleText: "LOW GRAVITY + RANDOM GUNS + X2 KNOCKBACK", Modifiers: []int{customModifierLowGravity, customModifierRandomWeaponEachLife, customModifierDoubleKnockback}, Enemies: []extraCampaignEnemy{
		xe(3, "Mayhem Crown", 10, 14, 42, 213, 6, 17).asBoss(&extraCampaignBoss{Title: "THE MAYHEM CROWN", KnockbackScale: 0.75, Phases: []extraCampaignBossPhase{
			{Lives: 11, Banner: "CROWN GUARD", SummonShields: 1},
			{Lives: 5, Banner: "FINAL MAYHEM", SummonDrones: 2, MoveScale: 1.2, KnockbackScale: 0.85},
		}}),
	}},
	// 21-25: non-player enemies (campaign_enemies.go). Appended so existing
	// mission numbers and saved progress stay valid.
	{MapNumber: 1, PlayerLives: 6, Crates: true, Powerups: true, Mode: "DRONE PATROL", Briefing: "Drones hover out of reach: knock them past the edge.", RuleText: "HOSTILE DRONES", Enemies: []extraCampaignEnemy{
		xe(3, "Operator", 7, 4, 9, 98, 0, 4),
	}, Units: []extraCampaignUnit{
		{Kind: campaignEnemyDrone, Name: "Drone", Count: 4, MaxActive: 2, HP: 6, Delay: 90, Interval: 240},
	}},
	{MapNumber: 8, PlayerLives: 6, Crates: true, Powerups: false, Mode: "DRONE SWARM", Briefing: "Shock grenades stall rotors; gusts finish the job.", RuleText: "DRONE SWARM + CROSSWIND", Hazard: extraCampaignHazardWind, Enemies: []extraCampaignEnemy{
		xe(3, "Swarm Lead", 4, 11, 17, 113, 0, 5),
	}, Units: []extraCampaignUnit{
		{Kind: campaignEnemyDrone, Name: "Drone", Count: 7, MaxActive: 3, HP: 6, Delay: 60, Interval: 200},
	}},
	{MapNumber: 10, PlayerLives: 6, Crates: true, Powerups: true, Mode: "SHIELD WALL", Briefing: "Shields stop bullets from the front: flank, jump or blast.", RuleText: "RIOT SHIELDS", Enemies: []extraCampaignEnemy{
		xe(3, "Sergeant", 2, 14, 31, 131, 0, 4),
	}, Units: []extraCampaignUnit{
		{Kind: campaignEnemyShieldBearer, Name: "Shield-Bearer", Count: 4, MaxActive: 2, Delay: 60, Interval: 260},
	}},
	{MapNumber: 4, PlayerLives: 7, Crates: false, Powerups: true, Mode: "BREACH TEAM", Briefing: "Blasts, flashbangs and tear gas crack the shield line.", RuleText: "RIOT SHIELDS + DYNAMITE RAIN", Hazard: extraCampaignHazardDynamiteRain, HazardInterval: 160, Enemies: []extraCampaignEnemy{
		xe(3, "Breacher", 9, 9, 15, 162, 2, 5),
	}, Units: []extraCampaignUnit{
		{Kind: campaignEnemyShieldBearer, Name: "Shield-Bearer", Count: 5, MaxActive: 2, Delay: 60, Interval: 220},
	}},
	{MapNumber: 7, PlayerLives: 8, Crates: true, Powerups: true, Mode: "COMBINED ARMS BOSS", Briefing: "The Commander calls in drones and shields as he falls.", RuleText: "DRONES + SHIELDS + BOSS", Enemies: []extraCampaignEnemy{
		xe(3, "Commander", 10, 14, 42, 143, 6, 9).asBoss(&extraCampaignBoss{Title: "FIELD COMMANDER", KnockbackScale: 0.8, Phases: []extraCampaignBossPhase{
			{Lives: 6, Banner: "DRONE SUPPORT", SummonDrones: 2},
			{Lives: 3, Banner: "SHIELD WALL", SummonShields: 2, MoveScale: 1.15},
		}}),
	}, Units: []extraCampaignUnit{
		{Kind: campaignEnemyDrone, Name: "Drone", Count: 3, MaxActive: 1, HP: 6, Delay: 300, Interval: 420},
		{Kind: campaignEnemyShieldBearer, Name: "Shield-Bearer", Count: 2, MaxActive: 1, Delay: 600, Interval: 600},
	}},
	// 26-28 (Stage B): the mech walker and the gunship finale
	// (campaign_enemy_mech.go, campaign_boss_heli.go).
	{MapNumber: 1, PlayerLives: 7, Crates: true, Powerups: true, Mode: "WALKER ASSAULT", Briefing: "A walking tank: flank it, its reactor is on its back.", RuleText: "ELITE MECH + DRONE COVER", Enemies: []extraCampaignEnemy{
		xe(3, "Spotter", 7, 4, 9, 113, 0, 3),
	}, Units: []extraCampaignUnit{
		{Kind: campaignEnemyMech, Name: "Strider", Count: 1, HP: 70, Title: "ELITE MECH", Delay: 60},
		{Kind: campaignEnemyDrone, Name: "Drone", Count: 2, MaxActive: 1, HP: 6, Delay: 420, Interval: 450},
	}},
	{MapNumber: 7, PlayerLives: 7, Crates: true, Powerups: true, Mode: "ARTILLERY WALKERS", Briefing: "Mortar mechs mark their strikes: keep moving, hit their backs.", RuleText: "MORTAR MECHS + RIOT SHIELDS", Units: []extraCampaignUnit{
		{Kind: campaignEnemyMech, Name: "Howitzer", Count: 2, MaxActive: 1, HP: 55, Rockets: true, Delay: 60, Interval: 240},
		{Kind: campaignEnemyShieldBearer, Name: "Shield-Bearer", Count: 3, MaxActive: 1, Delay: 360, Interval: 420},
	}},
	{MapNumber: 5, PlayerLives: 8, Crates: true, Powerups: true, Mode: "GUNSHIP FINALE", Briefing: "Bat or blow its rockets back, harpoon it down, hit it low.", RuleText: "HELICOPTER BOSS + ANTI-AIR CRATES", CratePool: extraAntiAirCrates, Units: []extraCampaignUnit{
		{Kind: campaignEnemyHelicopter, Name: "Black Kite", Count: 1, HP: 90, Title: "GUNSHIP", Delay: 90},
	}},
}

func extraCampaignPageCount() int {
	return (extraCampaignMissionCount + extraCampaignPageSize - 1) / extraCampaignPageSize
}

func extraCampaignPageRange(page int) (start, end int) {
	if page < 0 {
		page = 0
	}
	if page >= extraCampaignPageCount() {
		page = extraCampaignPageCount() - 1
	}
	start = page * extraCampaignPageSize
	end = start + extraCampaignPageSize
	if end > extraCampaignMissionCount {
		end = extraCampaignMissionCount
	}
	return start, end
}

func (g *Game) unlockAllExtraCampaignLevelsCheat() {
	for i := range g.extraCampaignLevels {
		if g.extraCampaignLevels[i] == 0 {
			g.extraCampaignLevels[i] = 1
		}
	}
	_ = g.saveProgress()
}

// extraCampaignUnitCount is the number of non-player enemies to defeat.
func extraCampaignUnitCount(spec extraCampaignMission) int {
	total := 0
	for _, u := range spec.Units {
		total += u.Count
	}
	return total
}

// extraCampaignOpponentLines lists bots (with boss titles) and enemy waves.
func extraCampaignOpponentLines(spec extraCampaignMission) []string {
	lines := make([]string, 0, len(spec.Enemies)+len(spec.Units))
	for _, enemy := range spec.Enemies {
		line := enemy.Name + " - " + weaponDisplayName(enemy.Gun) + " x" + strconv.Itoa(enemy.Lives)
		if enemy.Boss != nil {
			line = "BOSS " + line
		}
		lines = append(lines, line)
	}
	for _, u := range spec.Units {
		line := unitName(u) + " x" + strconv.Itoa(u.Count)
		switch u.Kind {
		case campaignEnemyDrone:
			line += " - flying, light"
		case campaignEnemyShieldBearer:
			line += " - front riot shield"
		case campaignEnemyMech:
			line += " - walker, weak back"
			if u.Rockets {
				line += ", mortar"
			}
		case campaignEnemyHelicopter:
			line += " - rockets, strafing, drops troops"
		}
		if u.Title != "" {
			line = "BOSS " + line
		}
		lines = append(lines, line)
	}
	return lines
}

func extraCampaignEnemyRespawns(spec extraCampaignMission) int {
	total := 0
	for _, enemy := range spec.Enemies {
		if enemy.Lives > 1 {
			total += enemy.Lives - 1
		}
	}
	return total
}

func (g *Game) startExtraCampaignMission() {
	level := g.campaignLevel
	if level < 1 || level > len(extraCampaignMissions) || g.fadeActive || g.extraCampaignLevels[level-1] == 0 {
		return
	}
	spec := extraCampaignMissions[level-1]
	arena, ok := g.ensureMap(spec.MapNumber)
	if !ok {
		return
	}

	players := make([]*Player, 0, 4)
	for slot, cfg := range g.campaignPlayers {
		if slot == 0 && cfg.Type == 0 {
			cfg.Type = 1
		}
		if cfg.Type == 0 {
			continue
		}
		id := slot + 1
		p := NewPlayer(id, arena)
		p.Controls = g.controlConfigs[id-1]
		p.Name = cfg.Name
		p.PlayerColor = cfg.Color
		p.ShirtNumber = cfg.Shirt
		p.HatNumber = cfg.Hat
		p.EyeNumber = cfg.Eyes
		p.DefaultWeapon = cfg.Gun
		cfg = migrateLegacyGearPerk(cfg)
		p.PerkNumber = cfg.Perk
		p.AbilityNumber = resolveConfigAbility(cfg.Ability, nil)
		p.Team = 1
		p.configureSourceGameMode(SourceGameModeNormal, spec.PlayerLives, arena)
		p.ThrowableType = normalizedThrowableType(cfg.Throwable)
		players = append(players, p)
	}

	for enemyIndex, src := range spec.Enemies {
		e := NewPlayer(src.ID, arena)
		e.Controls = g.controlConfigs[src.ID-1]
		e.AI = true
		e.Name = src.Name
		e.PlayerColor = src.Color
		e.ShirtNumber = src.Shirt
		e.HatNumber = src.Hat
		e.DefaultWeapon = src.Gun
		e.PerkNumber = src.Perk
		e.AbilityNumber = campaignEnemyAbility(3, level, enemyIndex)
		e.Team = 2
		e.AIShotDelay = src.ShotDelay
		e.configureSourceGameMode(SourceGameModeNormal, src.Lives, arena)
		applyExtraBossBase(e, src.Boss)
		players = append(players, e)
	}

	g.arena = arena
	g.players = players
	g.GameMode = SourceGameModeNormal
	g.TotalLives = spec.PlayerLives
	g.TeamGame = len(g.campaignPlayers) > 1 && g.campaignPlayers[1].Type == 1
	g.CrateON = spec.Crates
	g.PowerON = spec.Powerups
	g.crateWeapons = developedCrateWeapons()
	g.customMatchActive = len(spec.Modifiers) > 0
	g.matchModifiers = make(map[int]bool)
	for _, modifier := range spec.Modifiers {
		g.matchModifiers[modifier] = true
	}
	g.applyCustomModifierSetup()
	if len(spec.CratePool) > 0 {
		g.crateWeapons = append([]int(nil), spec.CratePool...)
	}
	g.campaignMode = true
	g.campaignSetGM2 = false
	g.campaignSetExtra = true
	g.campaignLevel = level
	g.extraHazardFrame = 0
	g.GameWin = false
	g.teamGameWin = false
	g.matchWinCountdown = 0
	g.soloWinFrame = 0
	g.winnerPlayerID = 0
	g.nextEntityID = 5
	g.seenDeaths = make(map[int]int, len(players))
	for _, p := range players {
		g.seenDeaths[p.ID] = p.DeathSerial
	}
	g.resetArenaState()
	g.beginFade(screenGameplay, fadePurposeScreen)
}

func (g *Game) completeExtraCampaignLevel(level int) {
	if level < 1 || level > len(g.extraCampaignLevels) {
		return
	}
	g.extraCampaignLevels[level-1] = 2
	if level < len(g.extraCampaignLevels) && g.extraCampaignLevels[level] == 0 {
		g.extraCampaignLevels[level] = 1
	}
	_ = g.saveProgress()
}
