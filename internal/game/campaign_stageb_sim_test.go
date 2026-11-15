package game

import "testing"

// TestStageBMissionsPlayThrough drives missions 26-28 through Game.Update
// with two scripted humans that run, jump and shoot. The big enemies are
// worn down from outside so every phase and death sequence runs; the
// mission must end in a win without panics and with the snapshot valid.
func TestStageBMissionsPlayThrough(t *testing.T) {
	for _, level := range []int{26, 27, 28} {
		g := New()
		g.campaignPlayers[1].Type = 1
		g.extraCampaignLevels[level-1] = 1
		g.campaignLevel = level
		g.startExtraCampaignMission()
		g.screen = screenGameplay
		g.fadeActive = false
		var humans []*Player
		for _, p := range g.players {
			if !p.AI {
				p.RemoteControlled = true
				p.Lives, p.TotalLives = 999, 999
				humans = append(humans, p)
			}
		}
		won := false
		for tick := 0; tick < 9000 && !won; tick++ {
			for i, p := range humans {
				in := &p.RemoteInput
				phase := (tick/40 + i*3) % 6
				in.Left, in.Right = phase == 0 || phase == 1, phase == 3 || phase == 4
				in.Up = tick%55 == i*20
				in.Shoot = tick%3 != 0
			}
			if tick%60 == 30 {
				for _, e := range g.campaignEnemies {
					if e.hittable() && campaignEnemyKind(e.Kind).bigEnemy() {
						g.damageBigEnemy(e, 4, humans[0].ID)
					} else if e.hittable() {
						g.destroyCampaignEnemy(e)
					}
				}
				for _, p := range g.players {
					if p.AI && p.Active && tick > 1200 {
						p.Lives = 1
						p.Kill(g.arena)
					}
				}
			}
			_ = g.Update()
			if tick%300 == 0 {
				s := g.makeNetSnapshot()
				trimNetSnapshotLists(&s)
				if err := validateNetSnapshot(s); err != nil {
					t.Fatalf("mission %d tick %d: own snapshot rejected: %v", level, tick, err)
				}
			}
			if g.teamGameWin && g.teamWinner == 1 || g.screen != screenGameplay {
				won = g.campaignLoseFrame == 0
				if !won {
					t.Fatalf("mission %d lost at tick %d", level, tick)
				}
			}
		}
		if !won {
			t.Fatalf("mission %d not won: %d enemies remaining", level, g.campaignEnemiesRemaining())
		}
	}
}
