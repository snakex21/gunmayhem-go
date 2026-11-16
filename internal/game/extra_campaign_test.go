package game

import "testing"

func TestExtraCampaignHasVariedRespawnMissions(t *testing.T) {
	if len(extraCampaignTitles) != extraCampaignMissionCount || len(extraCampaignMissions) != extraCampaignMissionCount {
		t.Fatalf("EXTRA campaign size titles=%d missions=%d", len(extraCampaignTitles), len(extraCampaignMissions))
	}
	singleEnemyMissions := 0
	twoEnemyMissions := 0
	for i, spec := range extraCampaignMissions {
		bigHP := 0
		for _, u := range spec.Units {
			if u.Kind.bigEnemy() {
				bigHP += u.HP * u.Count
			}
		}
		if bigHP > 0 {
			// Stage B boss missions (mech, gunship) are measured by the HP
			// pool of their big enemies, not by bot lives.
			if extraCampaignTitles[i] == "" || spec.Mode == "" || spec.Briefing == "" || spec.MapNumber < 1 || spec.MapNumber > 12 || len(spec.Enemies) > 2 || bigHP < 50 {
				t.Fatalf("EXTRA boss mission %d invalid: title=%q spec=%+v", i+1, extraCampaignTitles[i], spec)
			}
			for _, enemy := range spec.Enemies {
				if enemy.ID < 3 || enemy.ID > 4 || enemy.Lives < 3 || !isGunLibraryGunNumber(enemy.Gun) {
					t.Fatalf("EXTRA mission %d bad enemy=%+v", i+1, enemy)
				}
			}
			continue
		}
		if extraCampaignTitles[i] == "" || spec.Mode == "" || spec.Briefing == "" || spec.MapNumber < 1 || spec.MapNumber > 12 || len(spec.Enemies) < 1 || len(spec.Enemies) > 2 {
			t.Fatalf("EXTRA mission %d invalid: title=%q spec=%+v", i+1, extraCampaignTitles[i], spec)
		}
		if len(spec.Enemies) == 1 {
			singleEnemyMissions++
		} else {
			twoEnemyMissions++
		}
		units := extraCampaignUnitCount(spec)
		if lives := extraCampaignEnemyLives(spec) + units; lives <= spec.PlayerLives {
			t.Fatalf("EXTRA mission %d enemy respawns=%d must exceed player lives=%d", i+1, lives, spec.PlayerLives)
		}
		if extraCampaignEnemyRespawns(spec)+units < 6 {
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
	for _, mission := range extraCampaignMissions[12:20] {
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

// extraMissionBot builds the bot of mission/enemy index exactly as the
// mission start does for the fire-pace fields (gun + data ShotDelay).
func extraMissionBot(mission, index int) *Player {
	src := extraCampaignMissions[mission-1].Enemies[index]
	return &Player{ID: src.ID, Active: true, AI: true, Facing: 1, DefaultWeapon: src.Gun, Weapon: NewWeapon(src.Gun), AIShotDelay: src.ShotDelay}
}

func countCadenceTriggers(p *Player, g *Game, frames int) int {
	n := 0
	for i := 0; i < frames; i++ {
		p.AIShoot = true
		applyExtraCampaignAIFireCadence(p, g)
		if p.AIShoot {
			n++
		}
	}
	return n
}

func TestExtraMissionSixFlareAIWaitsBetweenSingleShots(t *testing.T) {
	if got := extraCampaignMissions[5].Enemies[0].ShotDelay; got != 90 {
		t.Fatalf("mission6 Firestarter ShotDelay=%d want90", got)
	}
	g := &Game{campaignMode: true, campaignSetExtra: true, campaignLevel: 6}
	if got := countCadenceTriggers(extraMissionBot(6, 0), g, 270); got != 3 {
		t.Fatalf("mission6 Flare Gun trigger frames=%d want3", got)
	}
	// Mission 8 keeps the original finale weapon behavior (no data delay).
	g.campaignLevel = 8
	energy := extraMissionBot(8, 1)
	energy.AIShoot = true
	applyExtraCampaignAIFireCadence(energy, g)
	if !energy.AIShoot || energy.AIFireCadence != 0 {
		t.Fatalf("mission8 cadence was changed by mission6 data: %+v", energy)
	}
}

// The pace comes from mission data, not from the mission number: the same
// bot keeps its pause whatever campaignLevel says, and a crate weapon fires
// at its own rate.
func TestExtraFirePaceComesFromMissionData(t *testing.T) {
	for _, level := range []int{1, 6, 11, 25} {
		g := &Game{campaignMode: true, campaignSetExtra: true, campaignLevel: level}
		if got := countCadenceTriggers(extraMissionBot(6, 0), g, 270); got != 3 {
			t.Fatalf("level %d changed the data-driven pace: %d", level, got)
		}
	}
	bot := extraMissionBot(6, 0)
	bot.Weapon = NewWeapon(91) // picked up a crate rifle
	if got := countCadenceTriggers(bot, &Game{}, 30); got != 30 {
		t.Fatalf("crate weapon inherited the mission pause: %d", got)
	}
	// Mission start copies ShotDelay onto the bot.
	g := startExtraMissionForTest(t, 9)
	found := false
	for _, p := range g.players {
		if p.ID == 3 {
			found = true
			if p.AIShotDelay != 135 {
				t.Fatalf("mission9 bot AIShotDelay=%d want135", p.AIShotDelay)
			}
		}
	}
	if !found {
		t.Fatal("mission9 boss missing")
	}
}

func TestExtraMissionNineAirstrikeBossUsesAimedCadence(t *testing.T) {
	g := &Game{campaignMode: true, campaignSetExtra: true, campaignLevel: 9}
	marshal := extraMissionBot(9, 0)
	marshal.AITargetValid, marshal.AITargetX, marshal.AITargetY = true, 500, 300
	if got := countCadenceTriggers(marshal, g, 405); got != 3 {
		t.Fatalf("mission9 airstrike trigger frames=%d want3", got)
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
	if got := countCadenceTriggers(extraMissionBot(11, 0), g, 225); got != 3 {
		t.Fatalf("mission11 harpoon trigger frames=%d want3", got)
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
	if len(g.grenades) != 1 || g.grenades[0].OwnerID != extraHazardOwnerID || g.grenades[0].Type != ThrowableDynamite {
		t.Fatalf("dynamite rain spawn=%+v", g.grenades)
	}
}

// The briefing promises a warning: a landing marker appears a full second
// before the dynamite drops, at the exact landing X, and stays until the
// blast.
func TestExtraDynamiteRainMarkerAppearsBeforeImpact(t *testing.T) {
	g := &Game{campaignMode: true, campaignSetExtra: true, campaignLevel: 18, arena: OriginalMap1()}
	firstMarker := -1
	for frame := 1; frame <= 100; frame++ {
		g.updateExtraCampaignHazards()
		if firstMarker < 0 && len(g.extraHazardMarkers) > 0 {
			firstMarker = frame
		}
	}
	if firstMarker < 0 || 100-firstMarker < dynamiteRainWarningFrames-1 {
		t.Fatalf("marker first frame=%d, want >= %d frames of warning", firstMarker, dynamiteRainWarningFrames)
	}
	if len(g.grenades) != 1 || len(g.extraHazardMarkers) != 1 {
		t.Fatalf("grenades=%d markers=%d", len(g.grenades), len(g.extraHazardMarkers))
	}
	m := g.extraHazardMarkers[0]
	if m.X != g.grenades[0].X {
		t.Fatalf("marker x=%.1f dynamite x=%.1f", m.X, g.grenades[0].X)
	}
	if want := hazardLandingY(g.arena, m.X); m.Y != want {
		t.Fatalf("marker y=%.1f want landing %.1f", m.Y, want)
	}
	if m.Life < throwableProfileFor(ThrowableDynamite).Fuse {
		t.Fatalf("marker disappears before the fuse ends: life=%d", m.Life)
	}
	// Deterministic: a second run picks the same spot.
	g2 := &Game{campaignMode: true, campaignSetExtra: true, campaignLevel: 18, arena: OriginalMap1()}
	for frame := 1; frame <= 100; frame++ {
		g2.updateExtraCampaignHazards()
	}
	if g2.grenades[0].X != g.grenades[0].X {
		t.Fatalf("landing point not deterministic: %.1f vs %.1f", g2.grenades[0].X, g.grenades[0].X)
	}
}

func TestExtraNeutralHazardsKeepKillCreditAndRespectBlastShield(t *testing.T) {
	arena := OriginalMap1()
	center := (arena.CrateMinX + arena.CrateMaxX) / 2
	victim := &Player{ID: 3, Active: true, AI: true, X: center + 60, Y: 300, LastHitBy: 1}
	shielded := &Player{ID: 4, Active: true, AI: true, X: center + 60, Y: 300, PerkNumber: 6}
	g := &Game{campaignMode: true, campaignSetExtra: true, campaignLevel: 19, arena: arena, players: []*Player{victim, shielded}, extraHazardFrame: 299}
	g.updateExtraCampaignHazards()
	if victim.LastHitBy != 1 {
		t.Fatalf("reactor pulse erased kill credit: LastHitBy=%d", victim.LastHitBy)
	}
	if shielded.VX <= 0 || shielded.VX*4.9 > victim.VX || shielded.VX*5.1 < victim.VX {
		t.Fatalf("blast shield should take one fifth: shielded=%.2f normal=%.2f", shielded.VX, victim.VX)
	}
	// Neutral dynamite pushes but keeps the credit too.
	gr := Grenade{X: victim.X - 40, Y: victim.Y - 30, OwnerID: extraHazardOwnerID, Type: ThrowableDynamite}
	before := victim.VX
	explodeGrenade(&gr, []*Player{victim})
	if victim.VX <= before || victim.LastHitBy != 1 {
		t.Fatalf("neutral dynamite push=%.1f credit=%d", victim.VX-before, victim.LastHitBy)
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

func startExtraMissionForTest(t *testing.T, level int) *Game {
	t.Helper()
	spec := extraCampaignMissions[level-1]
	arena := OriginalMap1()
	arena.Number = spec.MapNumber
	g := &Game{maps: map[int]Map{spec.MapNumber: arena}, arena: arena, campaignLevel: level, screen: screenCampaign, seenDeaths: map[int]int{}}
	for i := range g.controlConfigs {
		g.controlConfigs[i] = OriginalControls(i + 1)
	}
	g.initCampaignPlayerSetup()
	g.extraCampaignLevels = defaultExtraCampaignState()
	g.extraCampaignLevels[level-1] = 1
	g.campaignSetExtra = true
	g.startExtraCampaignMission()
	if !g.campaignMode || g.campaignLevel != level {
		t.Fatalf("mission %d did not start", level)
	}
	return g
}

func extraCampaignEnemyLives(spec extraCampaignMission) int {
	total := 0
	for _, enemy := range spec.Enemies {
		total += enemy.Lives
	}
	return total
}

func TestExtraCampaignTabIsClickable(t *testing.T) {
	g := New()
	g.screen = screenCampaign
	g.enterCampaignMenu()
	g.runCampaignUpdate(uiInput{X: -10, Y: -10})
	it, ok := g.campaignUI.item("camp.set/EXTRA")
	if !ok || it.disabled {
		t.Fatal("EXTRA campaign tab missing")
	}
	x, y := it.rect.X+it.rect.W/2, it.rect.Y+it.rect.H/2
	g.runCampaignUpdate(uiInput{X: x, Y: y, Moved: true})
	g.runCampaignUpdate(uiInput{X: x, Y: y, Pressed: true, Down: true})
	g.runCampaignUpdate(uiInput{X: x, Y: y, Released: true})
	if !g.campaignSetExtra || g.campaignUsesOriginalGM1Rules() {
		t.Fatal("EXTRA tab did not select the EXTRA campaign")
	}
}

func TestExtraCampaignCardsFitPagedThreeRowsAboveNavigation(t *testing.T) {
	if want := (extraCampaignMissionCount + extraCampaignPageSize - 1) / extraCampaignPageSize; extraCampaignPageCount() != want || want < 3 {
		t.Fatalf("EXTRA pages=%d want%d", extraCampaignPageCount(), want)
	}
	for page := 0; page < extraCampaignPageCount(); page++ {
		start, end := extraCampaignPageRange(page)
		for i := start; i < end; i++ {
			a := campaignTileRect(campaignSetExtra, i-start)
			if a.X < uiMargin || a.Y < uiContentTop || a.X+a.W > campaignInfoRect.X || a.Y+a.H > uiContentBottom {
				t.Fatalf("mission tile %d outside the grid: %+v", i+1, a)
			}
			for j := i + 1; j < end; j++ {
				b := campaignTileRect(campaignSetExtra, j-start)
				overlaps := a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
				if overlaps {
					t.Fatalf("mission tiles %d and %d overlap: %+v %+v", i+1, j+1, a, b)
				}
			}
		}
	}
}
