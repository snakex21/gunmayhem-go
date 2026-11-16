package game

import "testing"

// newExtraMissionMidFight starts EXTRA mission `level` from a clean progress
// state and runs it for a while through Game.Update with invulnerable,
// idle humans.
func newExtraMissionMidFight(t *testing.T, level, ticks int) *Game {
	t.Helper()
	g := New()
	g.extraCampaignLevels = [len(g.extraCampaignLevels)]int{}
	g.extraCampaignLevels[level-1] = 1
	g.campaignLevel = level
	g.startExtraCampaignMission()
	g.screen = screenGameplay
	g.fadeActive = false
	for _, p := range g.players {
		p.RemoteControlled = true
		if !p.AI {
			p.Lives, p.TotalLives = 999, 999
		}
	}
	for i := 0; i < ticks; i++ {
		_ = g.Update()
	}
	if g.screen != screenGameplay || g.GameWin || g.teamGameWin {
		t.Fatalf("mission %d ended on its own (screen=%d)", level, g.screen)
	}
	return g
}

// quitMissionFromPause leaves through the pause menu's quit button and runs
// the end-game fade until the mission list is back.
func quitMissionFromPause(t *testing.T, g *Game, wantLabel string) {
	t.Helper()
	k := pauseIn(t, g)
	if label, _ := g.pauseQuitLabel(); label != wantLabel {
		t.Fatalf("mission %d pause quit label %q want %q", g.campaignLevel, label, wantLabel)
	}
	k.click("pause.quit")
	for i := 0; i < 30 && g.screen == screenGameplay; i++ {
		_ = g.Update()
	}
	if g.screen != screenCampaign {
		t.Fatalf("quit did not return to the mission list (screen=%d)", g.screen)
	}
}

// TestQuittingExtraStageBMidFightDoesNotComplete: missions 27 and 28 have no
// enemy bot at all, and 26's only bot can be down while the mech still
// fights. "No bot left" used to count as a win, so quitting unlocked the
// next mission.
func TestQuittingExtraStageBMidFightDoesNotComplete(t *testing.T) {
	for _, level := range []int{26, 27, 28} {
		g := newExtraMissionMidFight(t, level, 150)
		// Take every enemy bot out; only the campaign enemies remain.
		for _, p := range g.players {
			if p.AI {
				p.Lives = 1
				p.Kill(g.arena)
			}
		}
		for i := 0; i < 5; i++ {
			_ = g.Update()
		}
		if g.campaignEnemiesRemaining() == 0 {
			t.Fatalf("mission %d: no campaign enemies left mid-fight", level)
		}
		quitMissionFromPause(t, g, "QUIT MISSION")
		if g.extraCampaignLevels[level-1] == 2 {
			t.Fatalf("mission %d marked completed by quitting", level)
		}
		if level < len(g.extraCampaignLevels) && g.extraCampaignLevels[level] != 0 {
			t.Fatalf("mission %d: quitting unlocked mission %d", level, level+1)
		}
		if g.campaignResult.open && g.campaignResult.won {
			t.Fatalf("mission %d: quit shown as a win", level)
		}
	}
}

// TestWinningExtraStageBCompletes: beating every bot and campaign enemy ends
// the mission by itself and completes it (and unlocks the next one).
func TestWinningExtraStageBCompletes(t *testing.T) {
	for _, level := range []int{26, 27, 28} {
		g := newExtraMissionMidFight(t, level, 1)
		humanID := 0
		for _, p := range g.players {
			if !p.AI {
				humanID = p.ID
				break
			}
		}
		for tick := 0; tick < 12000 && g.screen == screenGameplay; tick++ {
			if tick%30 == 0 && !g.GameWin && !g.teamGameWin {
				for _, e := range g.campaignEnemies {
					if e.hittable() && campaignEnemyKind(e.Kind).bigEnemy() {
						g.damageBigEnemy(e, 6, humanID)
					} else if e.hittable() {
						g.destroyCampaignEnemy(e)
					}
				}
				for _, p := range g.players {
					if p.AI && p.Active {
						p.Lives = 1
						p.Kill(g.arena)
					}
				}
			}
			_ = g.Update()
		}
		if g.screen != screenCampaign {
			t.Fatalf("mission %d never finished (%d enemies left)", level, g.campaignEnemiesRemaining())
		}
		if g.extraCampaignLevels[level-1] != 2 {
			t.Fatalf("won mission %d not completed", level)
		}
		if level < len(g.extraCampaignLevels) && g.extraCampaignLevels[level] == 0 {
			t.Fatalf("won mission %d did not unlock the next", level)
		}
	}
}

// TestPauseFinishMissionOnlyWhenWon: once every enemy is down the pause
// button reads FINISH MISSION and completes; the GM2 tutorial (no opponents)
// keeps finishing through it.
func TestPauseFinishMissionOnlyWhenWon(t *testing.T) {
	g := newExtraMissionMidFight(t, 1, 30)
	if label, _ := g.pauseQuitLabel(); label != "QUIT MISSION" {
		t.Fatalf("bots alive: label %q", label)
	}
	for _, p := range g.players {
		if p.AI {
			p.Active = false // eliminated, before the win countdown ends it
		}
	}
	quitMissionFromPause(t, g, "FINISH MISSION")
	if g.extraCampaignLevels[0] != 2 || g.extraCampaignLevels[1] == 0 {
		t.Fatalf("FINISH MISSION did not complete: %v", g.extraCampaignLevels[:2])
	}

	gm2 := New()
	gm2.gm2CampaignLevels[0] = 1
	gm2.campaignSetGM2 = true
	gm2.campaignLevel = 1
	gm2.startGM2CampaignMission()
	gm2.fadeActive = false
	gm2.screen = screenGameplay
	for i := 0; i < 60; i++ {
		_ = gm2.Update()
	}
	quitMissionFromPause(t, gm2, "FINISH MISSION")
	if gm2.gm2CampaignLevels[0] != 2 || !gm2.campaignResult.won {
		t.Fatalf("GM2 tutorial not completed: level=%d result=%+v", gm2.gm2CampaignLevels[0], gm2.campaignResult)
	}
}

// TestLosingBotlessExtraMissionDoesNotComplete: mission 28 has no enemy bot,
// so a real loss (every human out) used to pass the old "no bot active"
// completion check as well.
func TestLosingBotlessExtraMissionDoesNotComplete(t *testing.T) {
	g := newExtraMissionMidFight(t, 28, 60)
	for _, p := range g.players {
		if !p.AI {
			p.Lives = 1
			p.Kill(g.arena)
		}
	}
	for i := 0; i < 400 && g.screen == screenGameplay; i++ {
		_ = g.Update()
	}
	if g.screen != screenCampaign {
		t.Fatalf("lost mission did not end (screen=%d)", g.screen)
	}
	if g.extraCampaignLevels[27] == 2 || g.campaignResult.won {
		t.Fatalf("lost mission 28 counted as completed: level=%d result=%+v", g.extraCampaignLevels[27], g.campaignResult)
	}
}
