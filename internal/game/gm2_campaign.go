package game

var gm2CampaignTitles = [16]string{
	"Tutorial",
	"Easy AI",
	"2x Easy AI",
	"Triple Jump",
	"Moderate AI",
	"Invisible Man",
	"2x Moderate AI",
	"Dynamite Only",
	"Hard AI",
	"Gangster Shotguns",
	"Mini AI",
	"2x Hard AI",
	"Flawless Pistol Duel",
	"Too Fast",
	"De-materializer",
	"Final Test",
}

type gm2CampaignEnemy struct {
	ID    int
	Name  string
	Color int
	Shirt int
	Hat   int
	Eyes  int
	Gun   int
	Perk  int
}

type gm2CampaignMission struct {
	MapNumber int
	Lives     int
	Crates    bool
	Powerups  bool
	Enemies   []gm2CampaignEnemy
}

var gm2CampaignMissions = [16]gm2CampaignMission{
	{MapNumber: 5, Lives: 99999, Crates: false, Powerups: false},
	{MapNumber: 9, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Easy AI", 4, 1, 33, 9, 1, 0}}},
	{MapNumber: 19, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Easy AI", 4, 1, 43, 9, 1, 0}, {4, "Easy AI 2", 5, 1, 43, 9, 1, 0}}},
	{MapNumber: 2, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Triple Jumper", 20, 17, 37, 10, 2, 3}}},
	{MapNumber: 7, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Moderate AI", 18, 6, 46, 1, 2, 0}}},
	{MapNumber: 4, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "The Ghost", 12, 5, 5, 1, 4, 0}}},
	{MapNumber: 6, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Moderate AI", 18, 6, 46, 1, 2, 0}, {4, "Moderate AI 2", 17, 6, 31, 1, 4, 0}}},
	{MapNumber: 14, Lives: 3, Crates: false, Powerups: false, Enemies: []gm2CampaignEnemy{{3, "Easy AI", 4, 1, 33, 9, 6, 0}}},
	{MapNumber: 1, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Hard AI", 9, 4, 9, 11, 4, 0}}},
	{MapNumber: 20, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Gangster", 11, 2, 3, 4, 44, 0}, {4, "Gangster", 12, 3, 4, 4, 44, 0}}},
	{MapNumber: 12, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Tiny Guy", 1, 7, 11, 4, 5, 0}}},
	{MapNumber: 3, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Hard AI", 9, 4, 9, 11, 4, 0}, {4, "Hard AI 2", 10, 5, 8, 11, 4, 0}}},
	{MapNumber: 21, Lives: 5, Crates: false, Powerups: false, Enemies: []gm2CampaignEnemy{{3, "Pistol Man", 8, 15, 41, 11, 4, 0}}},
	{MapNumber: 18, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Too Fast", 3, 13, 42, 9, 5, 0}}},
	{MapNumber: 10, Lives: 5, Crates: false, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Terminator", 11, 12, 1, 4, 9, 0}}},
	{MapNumber: 16, Lives: 5, Crates: false, Powerups: false, Enemies: []gm2CampaignEnemy{{3, "Hard AI", 14, 8, 15, 3, 2, 0}, {4, "Hard AI 2", 14, 14, 15, 4, 2, 0}, {2, "Hard AI 3", 14, 4, 15, 2, 2, 0}}},
}

func (g *Game) unlockAllGM2CampaignLevelsCheat() {
	for i := range g.gm2CampaignLevels {
		if g.gm2CampaignLevels[i] == 0 {
			g.gm2CampaignLevels[i] = 1
		}
	}
	_ = g.saveProgress()
}

func (g *Game) openGM2CampaignMissionSetup(level int) {
	if level < 1 || level > len(gm2CampaignMissions) || g.gm2CampaignLevels[level-1] == 0 {
		return
	}
	g.campaignLevel = level
	g.campaignPhase = 2
	g.campaignNameFocus = 0
	if level == 1 || level == 16 {
		g.campaignPlayers[1].Type = 0
	}
}

func (g *Game) startGM2CampaignMission() {
	level := g.campaignLevel
	if level < 1 || level > 16 || g.fadeActive || g.gm2CampaignLevels[level-1] == 0 {
		return
	}
	spec := gm2CampaignMissions[level-1]
	arena, ok := g.ensureMap(gm2MapID(spec.MapNumber))
	if !ok {
		return
	}

	players := make([]*Player, 0, 4)
	allowCoop := level != 1 && level != 16
	for slot, cfg := range g.campaignPlayers {
		if slot == 0 && cfg.Type == 0 {
			cfg.Type = 1
		}
		if slot == 1 && !allowCoop {
			continue
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
		p.configureSourceGameMode(SourceGameModeNormal, spec.Lives, arena)
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
		e.EyeNumber = src.Eyes
		e.UseGM2Cosmetics = true
		e.DefaultWeapon = src.Gun
		e.PerkNumber = src.Perk
		e.AbilityNumber = campaignEnemyAbility(2, level, enemyIndex)
		e.Team = 2
		switch level {
		case 6:
			e.AISpecial = sourceAISpecialGM2Invisible
		case 11:
			e.AISpecial = sourceAISpecialGM2Mini
		}
		e.configureSourceGameMode(SourceGameModeNormal, spec.Lives, arena)
		players = append(players, e)
	}

	g.arena = arena
	g.players = players
	g.GameMode = SourceGameModeNormal
	g.TotalLives = spec.Lives
	g.TeamGame = allowCoop && len(players) > 0 && len(g.campaignPlayers) > 1 && g.campaignPlayers[1].Type == 1
	g.CrateON = spec.Crates
	g.PowerON = spec.Powerups
	g.customMatchActive = false
	g.matchModifiers = make(map[int]bool)
	g.campaignMode = true
	g.campaignSetGM2 = true
	g.campaignSetExtra = false
	g.campaignLevel = level
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

func (g *Game) completeGM2CampaignLevel(level int) {
	if level < 1 || level > 16 {
		return
	}
	g.gm2CampaignLevels[level-1] = 2
	if level >= 2 && level < 16 && g.gm2CampaignLevels[level] == 0 {
		g.gm2CampaignLevels[level] = 1
	}
	_ = g.saveProgress()
}
