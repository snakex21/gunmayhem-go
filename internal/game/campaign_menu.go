package game

const (
	campaignHitNone = iota
	campaignHitBack
	campaignHitStart
	campaignHitUnlockContinue
	campaignHitPanelBase = 100
)

var (
	campaignPanelLocal    = Rect{X: 0, Y: -0.013818359375, W: 310.0006103515625, H: 184.991455078125} // Symbol95
	campaignBackLocal     = Rect{X: 0, Y: 0, W: 135.05, H: 53}                                        // Symbol792
	campaignStartLocal    = Rect{X: 0, Y: 0, W: 398.1, H: 53}                                         // Symbol902
	campaignContinueLocal = Rect{X: 0, Y: 0, W: 148.5, H: 24}                                         // Symbol806
)

func (g *Game) campaignUsesOriginalGM1Rules() bool {
	return !g.campaignSetGM2 && !g.campaignSetExtra
}

func (g *Game) enterCampaignMenu() {
	g.campaignMode = false
	g.campaignPhase = 1
	g.campaignDetailAlpha = 0
	g.campaignPressed = campaignHitNone
	// Both Flash campaigns expose missions 1 and 2 from a fresh save and then
	// propagate completion forward. Keep the two progress arrays independent.
	if g.campaignSetExtra {
		for i := 0; i < len(g.extraCampaignLevels)-1; i++ {
			if g.extraCampaignLevels[i] == 2 && g.extraCampaignLevels[i+1] == 0 {
				g.extraCampaignLevels[i+1] = 1
			}
		}
	} else if g.campaignSetGM2 {
		for i := 1; i < 15; i++ {
			if g.gm2CampaignLevels[i] == 2 && g.gm2CampaignLevels[i+1] == 0 {
				g.gm2CampaignLevels[i+1] = 1
			}
		}
	} else {
		// Symbol948 startup unlock propagation. Level2 is available from a fresh
		// SharedObject, so propagation starts at completed level2 -> level3.
		for i := 1; i < 9; i++ {
			if g.campaignLevels[i] == 2 && g.campaignLevels[i+1] == 0 {
				g.campaignLevels[i+1] = 1
			}
		}
	}
	if g.campaignSliderX > 0 {
		g.campaignSliderX = 0
	}
	if g.campaignSliderX < -857 {
		g.campaignSliderX = -857
	}
}

func campaignPanelPlacements() [10]Rect {
	return [10]Rect{
		{X: 28, Y: 100, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 375, Y: 100, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 722, Y: 100, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 1069, Y: 100, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 1416, Y: 100, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 28, Y: 316.05, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 375, Y: 316.05, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 722, Y: 316.05, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 1069, Y: 316.05, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 1416, Y: 316.05, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
	}
}

func (g *Game) unlockAllCampaignLevelsCheat() {
	for i := range g.campaignLevels {
		// 0 = locked, 1 = available, 2 = completed. The test cheat only makes
		// locked levels selectable; it must not fake campaign completion/rewards.
		if g.campaignLevels[i] == 0 {
			g.campaignLevels[i] = 1
		}
	}
}

// startCampaignMission mirrors the root frame10 campaign switch. Specialized
// AI classes (AI3/AI4/boss) still share the common playerAI movement in the Go
// reconstruction, but their source names/loadout/perks/team/map/mode are set
// here so the campaign flow and interactions are no longer placeholders.
func (g *Game) startCampaignMission() {
	g.campaignSetGM2 = false
	g.campaignSetExtra = false
	level := g.campaignLevel
	if level < 1 || level > 10 || g.fadeActive {
		return
	}
	spec := gm1CampaignSpecs[level-1]
	arena, ok := g.ensureMap(spec.mapNumber)
	if !ok {
		arena, _ = g.ensureMap(1)
	}
	g.arena = arena

	// Campaign menu saves P1/P2 as the two human-side PLAYERNUMBER slots.
	// P1 is mandatory; P2 is optional. Enemy campaign scripts later occupy
	// P4 (all levels except 6) or P3+P4 (level6), exactly as frame10 does.
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
		// frame10 campaign block always assigns p1team=p2team=1. When P2 is
		// enabled, normal/instagib campaign matches become a team game.
		p.Team = 1
		p.configureSourceGameMode(spec.mode, spec.lives, arena)
		p.ThrowableType = normalizedThrowableType(cfg.Throwable)
		players = append(players, p)
	}

	enemySlots := []int{4}
	if level == 6 {
		enemySlots = []int{3, 4}
	}
	for i, id := range enemySlots {
		e := NewPlayer(id, arena)
		e.Controls = g.controlConfigs[id-1]
		e.AI = true
		e.Name = spec.name
		if level == 6 && i == 1 {
			e.Name = "Mafia Dude"
			e.ShirtNumber = 4
			e.HatNumber = 5
			e.DefaultWeapon = 3
		} else {
			e.ShirtNumber = spec.shirt
			e.HatNumber = spec.hat
			e.DefaultWeapon = spec.gun
		}
		e.PlayerColor = spec.color
		e.PerkNumber = spec.perk
		e.AbilityNumber = campaignEnemyAbility(1, level, i)
		// Campaign enemies are always on the opposing side. Source p3/p4 use
		// team2 in level6; in solo levels their reset team0 serves the same
		// purpose, but an explicit opposing team keeps projectile filtering sane.
		e.Team = 2

		// frame10 selects specialized AI movie clips through p4ptype. These are
		// real class variants, not ordinary AI with cosmetic perk values.
		e.AISpecial = campaignAISpecialForLevel(level)
		e.DamageMulti = 1
		e.FirepowerMulti = 1
		switch level {
		case 2:
			// playerAI.adjustrof() weakens Caveman's shots and makes him easier
			// to knock around.
			e.FirepowerMulti = 0.6
			e.DamageMulti = 1.3
		case 3:
			e.FirepowerMulti = 0.8
			e.DamageMulti = 1.2
		case 4:
			e.AIFakeDoubleTime = -80
		case 10:
			// playerAIboss overrides p4gun with MINIGUN before respawn().
			e.DefaultWeapon = 55
			e.DamageMulti = 0.8
		}
		e.configureSourceGameMode(spec.mode, spec.lives, arena)
		players = append(players, e)
	}

	g.players = players
	// Legacy GM1 campaign keeps the original unlock-driven crate pool even
	// though developed Custom Game can use the appended GM2 arsenal.
	g.crateWeapons = DefaultCrateWeapons()
	g.GameMode = spec.mode
	g.TotalLives = spec.lives
	p2Human := g.campaignPlayers[1].Type == 1
	g.TeamGame = spec.mode == SourceGameModeTeams ||
		(p2Human && (spec.mode == SourceGameModeNormal || spec.mode == SourceGameModeInstagib))
	g.CrateON = spec.crate
	g.PowerON = spec.power
	g.customMatchActive = false
	g.matchModifiers = make(map[int]bool)
	g.campaignMode = true
	g.campaignLevel = level
	g.campaignDynamiteTime = 0
	g.GameWin = false
	g.teamGameWin = false
	g.matchWinCountdown = 0
	g.soloWinFrame = 0
	g.winnerPlayerID = 0
	g.nextEntityID = len(players) + 1
	g.seenDeaths = make(map[int]int, len(players))
	for _, p := range players {
		g.seenDeaths[p.ID] = p.DeathSerial
	}
	g.resetArenaState()
	g.beginFade(screenGameplay, fadePurposeScreen)
}

// gm1CampaignSpec is one root frame10 campaign level: arena, lives, mode,
// pickups and the (first) enemy's loadout. The campaign UI reads it for the
// briefing; startCampaignMission uses it to build the match.
type gm1CampaignSpec struct {
	mapNumber int
	lives     int
	mode      int
	crate     bool
	power     bool
	enemies   int
	name      string
	color     int
	shirt     int
	hat       int
	gun       int
	perk      int
}

var gm1CampaignSpecs = [10]gm1CampaignSpec{
	{13, 3, SourceGameModeNormal, false, false, 1, "Dummy", 8, 24, 1, 0, 0},
	{12, 7, SourceGameModeNormal, true, true, 1, "Caveman Johnson", 8, 10, 7, 1, 0},
	{10, 7, SourceGameModeGunGame, true, true, 1, "Santa", 1, 11, 3, 2, 7},
	{8, 7, SourceGameModeNormal, true, false, 1, "Pylon Man", 9, 1, 9, 1, 0},
	{5, 7, SourceGameModeNormal, true, true, 1, "Gun Fu Master", 7, 9, 10, 3, 6},
	{11, 5, SourceGameModeTeams, true, true, 2, "Mafia Guy", 2, 3, 5, 2, 0},
	{2, 7, SourceGameModeNormal, true, true, 1, "Jet Pack Bunny", 4, 30, 17, 4, 5},
	{3, 7, SourceGameModeInstagib, true, true, 1, "Pirate", 10, 8, 15, 1, 5},
	{1, 7, SourceGameModeNormal, true, true, 1, "Sinusoidal Sam", 5, 5, 1, 6, 2},
	{7, 5, SourceGameModeNormal, true, false, 1, "The Boss", 6, 14, 13, 6, 6},
}
