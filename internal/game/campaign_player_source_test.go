package game

import "testing"

func TestCampaignPlayerDefaultsMatchSavedata3(t *testing.T) {
	g := &Game{}
	g.initCampaignPlayerSetup()
	p1, p2 := g.campaignPlayers[0], g.campaignPlayers[1]
	if p1.Name != "Player 1" || p1.Color != 2 || p1.Shirt != 1 || p1.Hat != 1 || p1.Gun != 1 || p1.Perk != 7 || p1.Type != 1 {
		t.Fatalf("campaign P1 defaults=%+v", p1)
	}
	if p2.Name != "Player 2" || p2.Color != 5 || p2.Type != 0 {
		t.Fatalf("campaign P2 defaults=%+v", p2)
	}
}

func TestCampaignLevelUsesP4EnemyAndOptionalP2(t *testing.T) {
	arena1 := OriginalMap1()
	maps := map[int]Map{1: arena1, 12: arena1}
	maps[12] = Map{Number:12, Platforms:arena1.Platforms, SpawnMinX:arena1.SpawnMinX, SpawnMaxX:arena1.SpawnMaxX, CrateMinX:arena1.CrateMinX, CrateMaxX:arena1.CrateMaxX, LowestY:arena1.LowestY}
	g := &Game{maps:maps, arena:maps[12], campaignLevel:2, screen:screenCampaign, seenDeaths:map[int]int{}}
	for i:=0;i<4;i++ { g.controlConfigs[i]=OriginalControls(i+1) }
	g.initCampaignPlayerSetup()
	g.campaignPlayers[1].Type = 1
	g.startCampaignMission()
	ids := map[int]*Player{}
	for _, p := range g.players { ids[p.ID]=p }
	if ids[1] == nil || ids[2] == nil || ids[4] == nil || ids[3] != nil {
		t.Fatalf("campaign level2 source slots present: ids=%v", ids)
	}
	if !ids[4].AI || ids[4].Name != "Caveman Johnson" {
		t.Fatalf("campaign P4 enemy=%+v", ids[4])
	}
}

func TestCampaignLevel6UsesP3P4EnemyTeam(t *testing.T) {
	arena := OriginalMap1(); arena.Number=11
	g := &Game{maps:map[int]Map{11:arena,1:OriginalMap1()}, arena:arena, campaignLevel:6, screen:screenCampaign, seenDeaths:map[int]int{}}
	for i:=0;i<4;i++ { g.controlConfigs[i]=OriginalControls(i+1) }
	g.initCampaignPlayerSetup()
	g.campaignPlayers[1].Type=1
	g.startCampaignMission()
	ids:=map[int]*Player{}; for _,p:=range g.players { ids[p.ID]=p }
	for _, id := range []int{1,2,3,4} { if ids[id]==nil { t.Fatalf("level6 missing P%d",id) } }
	if ids[1].Team!=1 || ids[2].Team!=1 || ids[3].Team!=2 || ids[4].Team!=2 || !ids[3].AI || !ids[4].AI {
		t.Fatalf("level6 team/source slots mismatch: %d/%d/%d/%d",ids[1].Team,ids[2].Team,ids[3].Team,ids[4].Team)
	}
}
