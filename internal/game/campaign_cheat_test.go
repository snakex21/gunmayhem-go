package game

import "testing"

func TestUnlockAllCampaignLevelsCheatOnlyUnlocks(t *testing.T) {
	g := &Game{}
	g.campaignLevels = [10]int{1, 2, 0, 0, 2, 0, 0, 0, 0, 0}

	g.unlockAllCampaignLevelsCheat()

	for i, state := range g.campaignLevels {
		if i == 1 || i == 4 {
			if state != 2 {
				t.Fatalf("completed level %d changed to state %d", i+1, state)
			}
			continue
		}
		if state != 1 {
			t.Fatalf("level %d state=%d want available(1)", i+1, state)
		}
	}
}
