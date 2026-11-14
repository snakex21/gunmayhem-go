package game

import "testing"

func TestCustomEditorRecoversSourceSelectionItems(t *testing.T) {
	items := sourceCustomSelectionItems()
	for editor := 1; editor <= 5; editor++ {
		if len(items[editor]) == 0 {
			t.Fatalf("Symbol835 editor frame%d has no recovered clickable items", editor)
		}
		t.Logf("editor%d items=%d", editor, len(items[editor]))
		seen := map[int]bool{}
		for _, item := range items[editor] {
			if seen[item.Number] {
				t.Fatalf("editor%d duplicate source number%d", editor, item.Number)
			}
			seen[item.Number] = true
		}
	}
}

func TestDevelopedCustomEditorStartsWithAllPerksUnlocked(t *testing.T) {
	g := &Game{}
	for perk := 1; perk <= 9; perk++ {
		if !g.customSelectionUnlocked(customEditPerk, perk) {
			t.Fatalf("perk%d should be available from the start", perk)
		}
	}
	if g.customSelectionUnlocked(customEditPerk, 10) {
		t.Fatal("non-perk frame10 must stay unavailable")
	}
}

func TestGunSpawnPerkCanBePickedWithoutCampaignProgress(t *testing.T) {
	g := &Game{}
	g.customPlayers = sourceDefaultCustomPlayers()
	g.customPlayers[0].Type = 1
	found := false
	for _, item := range developedCustomSelectionItems(customEditPerk) {
		if item.Number == 9 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("GUN SPAWN perk tile not found")
	}
	if !g.customSelectionUnlocked(customEditPerk, 9) {
		t.Fatal("GUN SPAWN tile remained locked without campaign progress")
	}
	hit := customSelectHit(0, customEditPerk, 9)
	if !g.activateCustomPlayerEditorHit(hit) || g.customPlayers[0].Perk != 9 {
		t.Fatalf("GUN SPAWN selection failed: hit=%d perk=%d", hit, g.customPlayers[0].Perk)
	}
}

func TestCustomEditorSelectionWritesCorrectField(t *testing.T) {
	g := &Game{}
	g.customPlayers = sourceDefaultCustomPlayers()

	if !g.activateCustomPlayerEditorHit(customSelectHit(0, customEditShirt, 7)) || g.customPlayers[0].Shirt != 7 {
		t.Fatalf("shirt selection wrote %d want7", g.customPlayers[0].Shirt)
	}
	if !g.activateCustomPlayerEditorHit(customSelectHit(0, customEditHat1, 5)) || g.customPlayers[0].Hat != 5 {
		t.Fatalf("hat selection wrote %d want5", g.customPlayers[0].Hat)
	}
	if !g.activateCustomPlayerEditorHit(customSelectHit(0, customEditGun, 4)) || g.customPlayers[0].Gun != 4 {
		t.Fatalf("gun selection wrote %d want4", g.customPlayers[0].Gun)
	}
	if !g.activateCustomPlayerEditorHit(customSelectHit(0, customEditPerk, 8)) || g.customPlayers[0].Perk != 8 {
		t.Fatalf("perk selection wrote %d want8", g.customPlayers[0].Perk)
	}
}

func TestCustomThrowableChoiceSelectsEveryType(t *testing.T) {
	g := &Game{}
	g.customPlayers = sourceDefaultCustomPlayers()
	g.customPlayers[0].Type = 1
	if n := len(developedThrowableSelectionItems()); n != throwableTypeCount {
		t.Fatalf("throwable picker has %d items want %d", n, throwableTypeCount)
	}
	for kind := 0; kind < throwableTypeCount; kind++ {
		hit := customThrowableChoiceHit(0, kind)
		if !g.activateCustomPlayerEditorHit(hit) || g.customPlayers[0].Throwable != kind {
			t.Fatalf("throwable choice %d failed: hit=%d selected=%d", kind, hit, g.customPlayers[0].Throwable)
		}
	}
}

func TestCustomEditorSelectsDevelopedStarterPistol(t *testing.T) {
	g := &Game{}
	g.customPlayers = sourceDefaultCustomPlayers()
	g.customPlayers[0].Type = 1
	items := developedCustomSelectionItems(customEditGun)
	// The last developed pistol (Peacekeeper); the bat follows it.
	item := items[len(items)-2]
	hit := customDevelopedGunHit(0, item.Number)
	slot, number, ok := decodeCustomDevelopedGunHit(hit)
	if !ok || slot != 0 || number != 231 {
		t.Fatalf("developed pistol cell hit=%d decoded slot=%d gun=%d ok=%v", hit, slot, number, ok)
	}
	if !g.activateCustomPlayerEditorHit(hit) || g.customPlayers[0].Gun != 231 || g.customEditor[0] != 0 {
		t.Fatalf("developed pistol selection failed: cfg=%+v editor=%d", g.customPlayers[0], g.customEditor[0])
	}
}

func TestOnlinePeacekeeperSelectionBecomesStartingWeapon(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.customPlayers[0].Type = 1
	g.customPlayers[0].Gun = 231
	p := g.newOnlinePlayer(1)
	if p.DefaultWeapon != 231 {
		t.Fatalf("online Peacekeeper selection became gun %d want231", p.DefaultWeapon)
	}
	if p.Weapon.Def.Number != 231 {
		t.Fatalf("online Peacekeeper did not equip: weapon=%d want231", p.Weapon.Def.Number)
	}
}

func TestCustomEditorSelectCodesPreserveEditorKind(t *testing.T) {
	for _, editor := range []int{customEditShirt, customEditHat1, customEditGun, customEditPerk, customEditHat2} {
		items := developedCustomSelectionItems(editor)
		if len(items) == 0 {
			t.Fatalf("editor%d has no items", editor)
		}
		for slot := 0; slot < 4; slot++ {
			item := items[0]
			hit := customSelectHit(slot, editor, item.Number)
			gotSlot, gotEditor, gotNumber, ok := decodeCustomSelectHit(hit)
			if !ok || gotSlot != slot || gotEditor != editor || gotNumber != item.Number {
				t.Fatalf("editor%d slot%d -> hit=%d decoded slot=%d editor=%d number=%d ok=%v", editor, slot, hit, gotSlot, gotEditor, gotNumber, ok)
			}
		}
	}
}

func TestCustomColorSelectorSelectsEveryColor(t *testing.T) {
	g := &Game{}
	g.customPlayers = sourceDefaultCustomPlayers()
	g.customPlayers[0].Type = 1
	items := developedCustomSelectionItems(customEditColor)
	want := gm1SelectableColorCount
	if gm2CosmeticsAvailable() {
		want = maxDevelopedColorValue()
	}
	if len(items) != want {
		t.Fatalf("combined color count=%d want%d", len(items), want)
	}
	for _, item := range items {
		for _, hit := range []int{customSelectHit(0, customEditColor, item.Number), customColorHit(0, item.Number)} {
			g.customPlayers[0].Color = 0
			if !g.activateCustomPlayerEditorHit(hit) || g.customPlayers[0].Color != item.Number {
				t.Fatalf("color%d activation via %d failed, stored=%d", item.Number, hit, g.customPlayers[0].Color)
			}
		}
	}
}
