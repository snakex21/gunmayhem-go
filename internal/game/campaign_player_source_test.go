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

func TestCampaignSetupSelectsDevelopedStarterPistol(t *testing.T) {
	g := &Game{}
	g.initCampaignPlayerSetup()
	g.campaignEditor[0] = customEditGun
	g.campaignCardY[0] = -300
	items := developedCustomSelectionItems(customEditGun)
	item := items[len(items)-1]
	x := campaignCardX(0) + item.Rect.X + item.Rect.W/2
	y := -300 + item.Rect.Y + item.Rect.H/2
	hit := g.campaignPlayerHitAt(x, y)
	slot, number, ok := decodeCampaignDevelopedGunHit(hit)
	if !ok || slot != 0 || number != 231 {
		t.Fatalf("campaign developed pistol hit=%d decoded slot=%d gun=%d ok=%v", hit, slot, number, ok)
	}
	if !g.activateCampaignPlayerHit(hit) || g.campaignPlayers[0].Gun != 231 || g.campaignEditor[0] != 0 {
		t.Fatalf("campaign developed pistol selection failed: cfg=%+v editor=%d", g.campaignPlayers[0], g.campaignEditor[0])
	}
}

func TestCampaignThrowablePickerOpensAndSelectsEveryType(t *testing.T) {
	g := &Game{}
	g.initCampaignPlayerSetup()
	cx := campaignCardX(0)
	for kind := 0; kind < throwableTypeCount; kind++ {
		g.campaignCardY[0] = 100
		open := sourceMenuHitRect(customPlayerTypeButtonLocal, cx+17.5, 100+248.0, 0.53875732421875, 1)
		hit := g.campaignPlayerHitAt(open.X+open.W/2, open.Y+open.H/2)
		if !g.activateCampaignPlayerHit(hit) || g.campaignEditor[0] != customEditThrowable {
			t.Fatalf("campaign throwable picker did not open: hit=%d editor=%d", hit, g.campaignEditor[0])
		}
		g.campaignCardY[0] = -300
		item := developedThrowableSelectionItems()[kind]
		hit = g.campaignPlayerHitAt(cx+item.Rect.X+item.Rect.W/2, -300+item.Rect.Y+item.Rect.H/2)
		if !g.activateCampaignPlayerHit(hit) || g.campaignPlayers[0].Throwable != kind || g.campaignEditor[0] != 0 {
			t.Fatalf("campaign throwable picker choice %d failed: hit=%d selected=%d editor=%d", kind, hit, g.campaignPlayers[0].Throwable, g.campaignEditor[0])
		}
	}
}

func TestCampaignFacePickerCanOpenAndSelectFace(t *testing.T) {
	if !gm2CosmeticsAvailable() {
		t.Skip("GM2 face assets unavailable")
	}
	g := &Game{}
	g.initCampaignPlayerSetup()
	g.campaignCardY[0] = 100
	cx, cy := campaignCardX(0), g.campaignCardY[0]
	hit := g.campaignPlayerHitAt(cx+65, cy+208)
	if !g.activateCampaignPlayerHit(hit) || g.campaignEditor[0] != customEditEyes {
		t.Fatalf("campaign face editor did not open: hit=%d editor=%d", hit, g.campaignEditor[0])
	}
	g.campaignCardY[0] = -300
	item := gm2FaceSelectionItems()[7]
	hit = g.campaignPlayerHitAt(cx+item.Rect.X+item.Rect.W/2, -300+item.Rect.Y+item.Rect.H/2)
	if !g.activateCampaignPlayerHit(hit) || g.campaignPlayers[0].Eyes != item.Number || g.campaignEditor[0] != 0 {
		t.Fatalf("campaign face selection failed: hit=%d cfg=%+v editor=%d", hit, g.campaignPlayers[0], g.campaignEditor[0])
	}
}

func TestCampaignLevelUsesP4EnemyAndOptionalP2(t *testing.T) {
	arena1 := OriginalMap1()
	maps := map[int]Map{1: arena1, 12: arena1}
	maps[12] = Map{Number: 12, Platforms: arena1.Platforms, SpawnMinX: arena1.SpawnMinX, SpawnMaxX: arena1.SpawnMaxX, CrateMinX: arena1.CrateMinX, CrateMaxX: arena1.CrateMaxX, LowestY: arena1.LowestY}
	g := &Game{maps: maps, arena: maps[12], campaignLevel: 2, screen: screenCampaign, seenDeaths: map[int]int{}}
	for i := 0; i < 4; i++ {
		g.controlConfigs[i] = OriginalControls(i + 1)
	}
	g.initCampaignPlayerSetup()
	g.campaignPlayers[0].Throwable = ThrowableMolotov
	g.campaignPlayers[1].Type = 1
	g.startCampaignMission()
	ids := map[int]*Player{}
	for _, p := range g.players {
		ids[p.ID] = p
	}
	if ids[1] == nil || ids[2] == nil || ids[4] == nil || ids[3] != nil {
		t.Fatalf("campaign level2 source slots present: ids=%v", ids)
	}
	if !ids[4].AI || ids[4].Name != "Caveman Johnson" {
		t.Fatalf("campaign P4 enemy=%+v", ids[4])
	}
	if ids[1].ThrowableType != ThrowableMolotov {
		t.Fatalf("campaign P1 throwable=%d want molotov", ids[1].ThrowableType)
	}
}

func TestCampaignLevel6UsesP3P4EnemyTeam(t *testing.T) {
	arena := OriginalMap1()
	arena.Number = 11
	g := &Game{maps: map[int]Map{11: arena, 1: OriginalMap1()}, arena: arena, campaignLevel: 6, screen: screenCampaign, seenDeaths: map[int]int{}}
	for i := 0; i < 4; i++ {
		g.controlConfigs[i] = OriginalControls(i + 1)
	}
	g.initCampaignPlayerSetup()
	g.campaignPlayers[1].Type = 1
	g.startCampaignMission()
	ids := map[int]*Player{}
	for _, p := range g.players {
		ids[p.ID] = p
	}
	for _, id := range []int{1, 2, 3, 4} {
		if ids[id] == nil {
			t.Fatalf("level6 missing P%d", id)
		}
	}
	if ids[1].Team != 1 || ids[2].Team != 1 || ids[3].Team != 2 || ids[4].Team != 2 || !ids[3].AI || !ids[4].AI {
		t.Fatalf("level6 team/source slots mismatch: %d/%d/%d/%d", ids[1].Team, ids[2].Team, ids[3].Team, ids[4].Team)
	}
}
