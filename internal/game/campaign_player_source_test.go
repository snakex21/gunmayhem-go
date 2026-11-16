package game

import (
	"fmt"
	"testing"
)

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

// campaignLoadoutPick activates the campaign Loadout tile with id.
func campaignLoadoutPick(t *testing.T, g *Game, choices []loadoutChoice, id string) {
	t.Helper()
	for _, c := range choices {
		if c.id == id {
			if c.locked {
				t.Fatalf("%s is locked", id)
			}
			if !g.activateCampaignPlayerHit(c.code) {
				t.Fatalf("%s code %d not handled", id, c.code)
			}
			return
		}
	}
	t.Fatalf("no campaign loadout tile %s", id)
}

func TestCampaignSetupSelectsDevelopedStarterPistol(t *testing.T) {
	g := &Game{}
	g.initCampaignPlayerSetup()
	// The last developed pistol (Peacekeeper); the bat follows it.
	choices := g.loadoutWeaponChoicesFor(g.campaignLoadoutBinding(0))
	var code int
	for _, c := range choices {
		if c.gun == 231 {
			code = c.code
		}
	}
	slot, number, ok := decodeCampaignDevelopedGunHit(code)
	if !ok || slot != 0 || number != 231 {
		t.Fatalf("campaign developed pistol code=%d decoded slot=%d gun=%d ok=%v", code, slot, number, ok)
	}
	campaignLoadoutPick(t, g, choices, "lo.gun.231")
	if g.campaignPlayers[0].Gun != 231 {
		t.Fatalf("campaign developed pistol selection failed: cfg=%+v", g.campaignPlayers[0])
	}
}

func TestCampaignThrowablePickerSelectsEveryType(t *testing.T) {
	g := &Game{}
	g.initCampaignPlayerSetup()
	for kind := 0; kind < throwableTypeCount; kind++ {
		campaignLoadoutPick(t, g, g.loadoutThrowableChoicesFor(g.campaignLoadoutBinding(0)), fmt.Sprintf("lo.throw.%d", kind))
		if g.campaignPlayers[0].Throwable != kind {
			t.Fatalf("campaign throwable choice %d selected %d", kind, g.campaignPlayers[0].Throwable)
		}
	}
}

func TestCampaignFacePickerSelectsFace(t *testing.T) {
	if !gm2CosmeticsAvailable() {
		t.Skip("GM2 face assets unavailable")
	}
	g := &Game{}
	g.initCampaignPlayerSetup()
	item := gm2FaceSelectionItems()[7]
	choices, _ := g.loadoutCharacterChoicesFor(g.campaignLoadoutBinding(1), loadoutPartFace)
	campaignLoadoutPick(t, g, choices, fmt.Sprintf("lo.face.%d", item.Number))
	if g.campaignPlayers[1].Eyes != item.Number || g.campaignPlayers[0].Eyes == item.Number && item.Number != 1 {
		t.Fatalf("campaign face selection failed: cfg=%+v", g.campaignPlayers[1])
	}
}
