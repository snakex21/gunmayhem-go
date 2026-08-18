package game

import "testing"

func TestSourceFadeChangesScreenOnFrame10AndRemovesOn21(t *testing.T) {
	g := &Game{screen: screenMainMenu, campaignPhase: 1}
	g.beginFade(screenCampaign, fadePurposeScreen)
	if g.fadeFrame != 1 || !g.fadeActive {
		t.Fatalf("fade start frame=%d active=%v", g.fadeFrame, g.fadeActive)
	}
	for i := 0; i < 8; i++ {
		g.updateFade()
	}
	if g.screen != screenMainMenu || g.fadeFrame != 9 {
		t.Fatalf("before source frame10 screen=%v frame=%d", g.screen, g.fadeFrame)
	}
	g.updateFade()
	if g.screen != screenCampaign || g.fadeFrame != 10 {
		t.Fatalf("source frame10 did not change screen: screen=%v frame=%d", g.screen, g.fadeFrame)
	}
	for g.fadeActive {
		g.updateFade()
	}
	if g.fadeFrame != 0 {
		t.Fatalf("source frame21 did not remove fade: frame=%d", g.fadeFrame)
	}
}

func TestCampaignLevel4WinUnlocksLevel5(t *testing.T) {
	arena := OriginalMap1()
	human := NewPlayer(1, arena)
	human.AI = false
	human.Active = true
	ai := NewPlayer(4, arena)
	ai.AI = true
	ai.Active = false
	double := NewPlayer(5, arena)
	double.AI = true
	double.IsDouble = true
	double.Active = false
	g := &Game{
		players:        []*Player{human, ai, double},
		campaignMode:   true,
		campaignLevel:  4,
		GameWin:        true,
		winnerPlayerID: human.ID,
	}
	g.campaignLevels[3] = 1
	for i := range g.campaignGuns {
		g.campaignGuns[i] = false
	}
	g.finishSourceGameBeforeScreenChange()
	if g.campaignLevels[3] != 2 || g.campaignLevels[4] != 1 {
		t.Fatalf("level4 progression=%v want level4 complete and level5 available", g.campaignLevels[:6])
	}
	if !g.campaignGuns[41] || !g.campaignGuns[53] {
		t.Fatalf("level4 rewards missing: gun41=%v gun53=%v", g.campaignGuns[41], g.campaignGuns[53])
	}
}

func TestCampaignCompletionUsesSourceRewardPairs(t *testing.T) {
	g := &Game{}
	for i := range g.campaignGuns {
		g.campaignGuns[i] = false
	}
	g.campaignLevels[1] = 1
	g.completeCampaignLevel(2)
	if g.campaignLevels[1] != 2 || g.campaignLevels[2] != 1 {
		t.Fatalf("level progression=%v want level2 complete level3 available", g.campaignLevels[:4])
	}
	if !g.campaignGuns[40] || !g.campaignGuns[52] {
		t.Fatal("level2 source rewards 40/52 were not unlocked")
	}
	if g.campaignGuns[18] || g.campaignGuns[29] {
		t.Fatal("level1 rewards were incorrectly unlocked by level2")
	}
}

func TestNonCampaignLastSurvivorUsesSourceCountdown(t *testing.T) {
	arena := OriginalMap1()
	p := NewPlayer(1, arena)
	p.Active = true
	g := &Game{players: []*Player{p}, GameMode: SourceGameModeNormal, TotalLives: 10}
	g.updateMatchInteractions()
	if g.soloWinFrame != 2 || g.GameWin {
		t.Fatalf("countdown start frame=%d gamewin=%v", g.soloWinFrame, g.GameWin)
	}
	for i := 0; i < 48; i++ {
		g.updateMatchInteractions()
	}
	if g.soloWinFrame != 50 || g.GameWin {
		t.Fatalf("before source countdown frame51 frame=%d win=%v", g.soloWinFrame, g.GameWin)
	}
	g.updateMatchInteractions()
	if !g.GameWin || g.soloWinFrame != 51 || g.winnerPlayerID != p.ID {
		t.Fatalf("source countdown frame51 win mismatch: frame=%d win=%v winner=%d", g.soloWinFrame, g.GameWin, g.winnerPlayerID)
	}
}

func TestCampaignAISoleSurvivorUsesCampaignLoseNotGameWin(t *testing.T) {
	arena := OriginalMap1()
	ai := NewPlayer(2, arena)
	ai.AI = true
	g := &Game{players: []*Player{ai}, GameMode: SourceGameModeNormal, campaignMode: true}
	g.updateMatchInteractions()
	if !g.teamGameWin || g.GameWin || g.campaignLoseFrame != 1 {
		t.Fatalf("campaign AI survivor state: teamWin=%v gameWin=%v loseFrame=%d", g.teamGameWin, g.GameWin, g.campaignLoseFrame)
	}
}

func TestPauseButtonsMatchSourcePositions(t *testing.T) {
	if got := pauseHitAt(448, 275); got != pauseHitResume {
		t.Fatalf("resume hit=%d", got)
	}
	if got := pauseHitAt(448, 324); got != pauseHitExit {
		t.Fatalf("exit hit=%d", got)
	}
}
