package game

import "testing"

func TestExtraCampaignHasTwentyVariedRespawnMissions(t *testing.T) {
	if len(extraCampaignTitles) != extraCampaignMissionCount || len(extraCampaignMissions) != extraCampaignMissionCount {
		t.Fatalf("EXTRA campaign size titles=%d missions=%d", len(extraCampaignTitles), len(extraCampaignMissions))
	}
	singleEnemyMissions := 0
	twoEnemyMissions := 0
	for i, spec := range extraCampaignMissions {
		if extraCampaignTitles[i] == "" || spec.Mode == "" || spec.Briefing == "" || spec.MapNumber < 1 || spec.MapNumber > 12 || len(spec.Enemies) < 1 || len(spec.Enemies) > 2 {
			t.Fatalf("EXTRA mission %d invalid: title=%q spec=%+v", i+1, extraCampaignTitles[i], spec)
		}
		if len(spec.Enemies) == 1 {
			singleEnemyMissions++
		} else {
			twoEnemyMissions++
		}
		if lives := extraCampaignEnemyLives(spec); lives <= spec.PlayerLives {
			t.Fatalf("EXTRA mission %d enemy respawns=%d must exceed player lives=%d", i+1, lives, spec.PlayerLives)
		}
		if extraCampaignEnemyRespawns(spec) < 6 {
			t.Fatalf("EXTRA mission %d does not have enough enemy returns", i+1)
		}
		for _, enemy := range spec.Enemies {
			if enemy.ID < 3 || enemy.ID > 4 || enemy.Lives < 4 || !isGunLibraryGunNumber(enemy.Gun) {
				t.Fatalf("EXTRA mission %d bad enemy=%+v", i+1, enemy)
			}
		}
	}
	if singleEnemyMissions < 4 || twoEnemyMissions < 4 {
		t.Fatalf("EXTRA roster lacks variety: solo=%d squads=%d", singleEnemyMissions, twoEnemyMissions)
	}
	if mission := extraCampaignMissions[3]; len(mission.Enemies) != 2 || mission.Enemies[0].Gun != 151 || mission.Enemies[1].Gun != 158 {
		t.Fatalf("sniper mission 4 was unexpectedly nerfed: %+v", mission)
	}
	if mission := extraCampaignMissions[8]; len(mission.Enemies) != 1 || mission.Enemies[0].Gun != 90 {
		t.Fatalf("mission 9 is not the HIMARS airstrike boss: %+v", mission)
	}
	mechanicMissions := 0
	for _, mission := range extraCampaignMissions[12:] {
		if len(mission.Modifiers) > 0 || mission.Hazard != extraCampaignHazardNone {
			mechanicMissions++
		}
	}
	if mechanicMissions != 8 {
		t.Fatalf("new continuation mechanics=%d want8", mechanicMissions)
	}
}

func TestExtraCampaignStartsWithIndependentEnemyLives(t *testing.T) {
	arena := OriginalMap1()
	arena.Number = 8
	g := &Game{maps: map[int]Map{8: arena}, arena: arena, campaignLevel: 1, screen: screenCampaign, seenDeaths: map[int]int{}}
	for i := range g.controlConfigs {
		g.controlConfigs[i] = OriginalControls(i + 1)
	}
	g.initCampaignPlayerSetup()
	g.campaignPlayers[0].Name = "Custom Hero"
	g.campaignPlayers[0].Color = 7
	g.campaignPlayers[0].Gun = 112
	g.campaignPlayers[0].Perk = 3
	g.campaignPlayers[0].Throwable = ThrowableMolotov
	g.extraCampaignLevels = defaultExtraCampaignState()
	g.campaignSetExtra = true
	g.startExtraCampaignMission()

	players := map[int]*Player{}
	for _, p := range g.players {
		players[p.ID] = p
	}
	if !g.campaignMode || !g.campaignSetExtra || g.campaignSetGM2 || players[1] == nil || players[3] == nil || players[4] == nil {
		t.Fatalf("EXTRA mission did not start correctly: mode=%v extra=%v gm2=%v players=%v", g.campaignMode, g.campaignSetExtra, g.campaignSetGM2, players)
	}
	if players[1].Lives != 6 || players[3].Lives != 4 || players[4].Lives != 4 {
		t.Fatalf("EXTRA independent lives human=%d enemies=%d/%d", players[1].Lives, players[3].Lives, players[4].Lives)
	}
	if players[1].Name != "Custom Hero" || players[1].PlayerColor != 7 || players[1].Weapon.Def.Number != 112 || players[1].PerkNumber != 3 || players[1].ThrowableType != ThrowableMolotov {
		t.Fatalf("EXTRA player customization was not applied: %+v", players[1])
	}
	if players[3].Weapon.Def.Number != 99 || players[4].Weapon.Def.Number != 91 || !players[3].AI || !players[4].AI {
		t.Fatalf("EXTRA starting enemies P3=%+v P4=%+v", players[3], players[4])
	}
}

func TestExtraSingleBossMissionStartsWithoutPhantomSecondEnemy(t *testing.T) {
	arena := OriginalMap1()
	arena.Number = 11
	g := &Game{maps: map[int]Map{11: arena}, arena: arena, campaignLevel: 2, screen: screenCampaign, seenDeaths: map[int]int{}}
	for i := range g.controlConfigs {
		g.controlConfigs[i] = OriginalControls(i + 1)
	}
	g.initCampaignPlayerSetup()
	g.extraCampaignLevels = defaultExtraCampaignState()
	g.extraCampaignLevels[1] = 1
	g.campaignSetExtra = true
	g.startExtraCampaignMission()

	players := map[int]*Player{}
	for _, p := range g.players {
		players[p.ID] = p
	}
	if len(players) != 2 || players[1] == nil || players[3] == nil || players[4] != nil {
		t.Fatalf("single-boss mission spawned wrong roster: %v", players)
	}
	if players[3].Weapon.Def.Number != 165 || players[3].Lives != 9 || !players[3].AI {
		t.Fatalf("single-boss Hunter setup mismatch: %+v", players[3])
	}
}

func TestExtraCampaignProgressIsSeparate(t *testing.T) {
	g := New()
	gm1, gm2 := g.campaignLevels, g.gm2CampaignLevels
	g.completeExtraCampaignLevel(1)
	if g.extraCampaignLevels[0] != 2 || g.extraCampaignLevels[1] != 1 {
		t.Fatalf("EXTRA progression=%v", g.extraCampaignLevels)
	}
	if g.campaignLevels != gm1 || g.gm2CampaignLevels != gm2 {
		t.Fatal("EXTRA completion changed an original campaign")
	}
}

func TestAllCampaignSetupsStartWithEveryPerkAvailable(t *testing.T) {
	for _, setup := range []struct {
		name  string
		extra bool
		gm2   bool
	}{
		{name: "GM1"},
		{name: "GM2", gm2: true},
		{name: "EXTRA", extra: true},
	} {
		g := &Game{campaignSetExtra: setup.extra, campaignSetGM2: setup.gm2}
		for perk := 1; perk <= 9; perk++ {
			if !g.campaignSelectionUnlocked(customEditPerk, perk) {
				t.Fatalf("%s perk %d remained locked", setup.name, perk)
			}
		}
	}
}

func TestExtraCampaignTabIsClickable(t *testing.T) {
	_, _, extra := campaignSetRects()
	if got := campaignSetHitAt(extra.X+extra.W/2, extra.Y+extra.H/2); got != campaignHitSetExtra {
		t.Fatalf("EXTRA campaign tab hit=%d want%d", got, campaignHitSetExtra)
	}
	g := &Game{campaignSetExtra: true}
	if g.campaignUsesOriginalGM1Rules() {
		t.Fatal("EXTRA campaign accidentally enables GM1 scripted rules")
	}
}

func TestExtraCampaignEndsWhenEitherSideIsEliminated(t *testing.T) {
	human := &Player{ID: 1, Active: true, Team: 1}
	enemyA := &Player{ID: 3, Active: false, AI: true, Team: 2}
	enemyB := &Player{ID: 4, Active: false, AI: true, Team: 2}
	g := &Game{campaignMode: true, campaignSetExtra: true, players: []*Player{human, enemyA, enemyB}}
	g.updateMatchInteractions()
	if !g.GameWin || g.winnerPlayerID != human.ID {
		t.Fatalf("EXTRA victory did not end match: win=%v winner=%d", g.GameWin, g.winnerPlayerID)
	}

	human.Active = false
	enemyA.Active = true
	enemyB.Active = true
	g.GameWin = false
	g.teamGameWin = false
	g.winnerPlayerID = 0
	g.updateMatchInteractions()
	if !g.teamGameWin || g.campaignLoseFrame != 1 {
		t.Fatalf("EXTRA defeat did not end match: teamwin=%v loseframe=%d", g.teamGameWin, g.campaignLoseFrame)
	}
}

func TestExtraMissionSixFlareAIWaitsBetweenSingleShots(t *testing.T) {
	g := &Game{campaignMode: true, campaignSetExtra: true, campaignLevel: 6}
	flare := &Player{AI: true, Weapon: NewWeapon(181)}
	triggerFrames := 0
	for i := 0; i < 270; i++ {
		flare.AIShoot = true
		applyExtraCampaignAIFireCadence(flare, g)
		if flare.AIShoot {
			triggerFrames++
		}
	}
	if triggerFrames != 3 {
		t.Fatalf("mission6 Flare Gun trigger frames=%d want3", triggerFrames)
	}

	// The correction is intentionally limited to mission 6. Mission 8 keeps
	// the original finale weapon behavior.
	g.campaignLevel = 8
	energy := &Player{AI: true, Weapon: NewWeapon(185), AIShoot: true}
	applyExtraCampaignAIFireCadence(energy, g)
	if !energy.AIShoot || energy.AIFireCadence != 0 {
		t.Fatalf("mission8 cadence was changed by mission6 fix: %+v", energy)
	}
}

func TestExtraMissionNineAirstrikeBossUsesAimedCadence(t *testing.T) {
	g := &Game{campaignMode: true, campaignSetExtra: true, campaignLevel: 9}
	marshal := &Player{ID: 3, Active: true, AI: true, Facing: 1, Weapon: NewWeapon(90), AITargetValid: true, AITargetX: 500, AITargetY: 300}
	triggerFrames := 0
	for i := 0; i < 405; i++ {
		marshal.AIShoot = true
		applyExtraCampaignAIFireCadence(marshal, g)
		if marshal.AIShoot {
			triggerFrames++
		}
	}
	if triggerFrames != 3 {
		t.Fatalf("mission9 airstrike trigger frames=%d want3", triggerFrames)
	}

	marshal.AIFireCadence = 0
	marshal.AIShoot = true
	marshal.Weapon.WaitTime = marshal.Weapon.Def.ROF
	marshal.Weapon.Frame = 1
	var bullets []Bullet
	var shells []Shell
	var flashes []MuzzleFlash
	updateWeapon(marshal, &bullets, &shells, &flashes)
	if len(bullets) != 1 || bullets[0].DevelopedProjectile != DevelopedProjectileStrike || bullets[0].X != 522 || bullets[0].Y != -80 {
		t.Fatalf("AI airstrike was not aimed near target: bullets=%+v", bullets)
	}
}

func TestExtraJuggernautUsesDeliberateHarpoonShots(t *testing.T) {
	g := &Game{campaignMode: true, campaignSetExtra: true, campaignLevel: 11}
	juggernaut := &Player{AI: true, Weapon: NewWeapon(179)}
	triggerFrames := 0
	for i := 0; i < 225; i++ {
		juggernaut.AIShoot = true
		applyExtraCampaignAIFireCadence(juggernaut, g)
		if juggernaut.AIShoot {
			triggerFrames++
		}
	}
	if triggerFrames != 3 {
		t.Fatalf("mission11 harpoon trigger frames=%d want3", triggerFrames)
	}
}

func TestExtraCampaignCardsFitPagedThreeRowsAboveNavigation(t *testing.T) {
	if extraCampaignPageCount() != 2 {
		t.Fatalf("EXTRA pages=%d want2", extraCampaignPageCount())
	}
	for page := 0; page < extraCampaignPageCount(); page++ {
		start, end := extraCampaignPageRange(page)
		for i := start; i < end; i++ {
			a := extraCampaignCardRect(i - start)
			if a.X < 0 || a.Y < 100 || a.X+a.W > ScreenWidth || a.Y+a.H >= 525 {
				t.Fatalf("mission card %d outside menu area: %+v", i+1, a)
			}
			for j := i + 1; j < end; j++ {
				b := extraCampaignCardRect(j - start)
				overlaps := a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
				if overlaps {
					t.Fatalf("mission cards %d and %d overlap: %+v %+v", i+1, j+1, a, b)
				}
			}
		}
	}
}

func TestExtraMissionThirteenAppliesLowGravityJetpackRules(t *testing.T) {
	arena := OriginalMap1()
	arena.Number = 8
	g := &Game{maps: map[int]Map{8: arena}, arena: arena, campaignLevel: 13, screen: screenCampaign, seenDeaths: map[int]int{}}
	for i := range g.controlConfigs {
		g.controlConfigs[i] = OriginalControls(i + 1)
	}
	g.initCampaignPlayerSetup()
	g.extraCampaignLevels = defaultExtraCampaignState()
	g.extraCampaignLevels[12] = 1
	g.campaignSetExtra = true
	g.startExtraCampaignMission()

	if !g.customModifierActive(customModifierLowGravity) || !g.customModifierActive(customModifierJetpacks) {
		t.Fatalf("mission13 modifiers=%v", g.matchModifiers)
	}
	if got := g.customRuntimeArena().Gravity; got >= arena.Gravity*.7 {
		t.Fatalf("mission13 gravity=%f base=%f", got, arena.Gravity)
	}
	for _, p := range g.players {
		if !p.RuleInfiniteJetpack {
			t.Fatalf("mission13 player %d has no infinite jetpack", p.ID)
		}
	}
}

func TestExtraDynamiteRainWarnsThenSpawnsNeutralGrenade(t *testing.T) {
	g := &Game{campaignMode: true, campaignSetExtra: true, campaignLevel: 18, arena: OriginalMap1()}
	for i := 0; i < 99; i++ {
		g.updateExtraCampaignHazards()
	}
	if len(g.grenades) != 0 {
		t.Fatalf("dynamite rain spawned early: %d", len(g.grenades))
	}
	g.updateExtraCampaignHazards()
	if len(g.grenades) != 1 || g.grenades[0].OwnerID != 0 || g.grenades[0].Type != ThrowableDynamite {
		t.Fatalf("dynamite rain spawn=%+v", g.grenades)
	}
}

func TestExtraShockReactorPushesBothSidesFromCenter(t *testing.T) {
	arena := OriginalMap1()
	center := (arena.CrateMinX + arena.CrateMaxX) / 2
	left := &Player{ID: 1, Active: true, X: center - 80}
	right := &Player{ID: 3, Active: true, AI: true, X: center + 80}
	g := &Game{campaignMode: true, campaignSetExtra: true, campaignLevel: 19, arena: arena, players: []*Player{left, right}, extraHazardFrame: 299}
	g.updateExtraCampaignHazards()
	if left.VX >= 0 || right.VX <= 0 || left.VY >= 0 || right.VY >= 0 {
		t.Fatalf("reactor pulse left=(%.1f,%.1f) right=(%.1f,%.1f)", left.VX, left.VY, right.VX, right.VY)
	}
	if len(g.blastFX) == 0 || len(g.explosions) != 1 {
		t.Fatalf("reactor pulse visuals blast=%d explosions=%d", len(g.blastFX), len(g.explosions))
	}
}
