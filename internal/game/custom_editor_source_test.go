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
	g.customEditor[0] = customEditPerk
	g.customCardY[0] = -300
	var perk9 customSelectionItem
	found := false
	for _, item := range developedCustomSelectionItems(customEditPerk) {
		if item.Number == 9 {
			perk9 = item
			found = true
			break
		}
	}
	if !found {
		t.Fatal("GUN SPAWN perk tile not found")
	}
	x := 20 + perk9.Rect.X + perk9.Rect.W/2
	y := -300 + perk9.Rect.Y + perk9.Rect.H/2
	hit := g.customPlayerEditorHitAt(x, y)
	if hit == menuHitNone {
		t.Fatal("GUN SPAWN tile remained locked without campaign progress")
	}
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

func TestCustomThrowablePickerOpensAndSelectsEveryType(t *testing.T) {
	g := &Game{}
	g.customPlayers = sourceDefaultCustomPlayers()
	g.customPlayers[0].Type = 1
	for kind := 0; kind < throwableTypeCount; kind++ {
		g.customCardY[0] = 100
		open := sourceMenuHitRect(customPlayerTypeButtonLocal, 20+17.5, 100+248.0, 0.53875732421875, 1)
		hit := g.customPlayerEditorHitAt(open.X+open.W/2, open.Y+open.H/2)
		if !g.activateCustomPlayerEditorHit(hit) || g.customEditor[0] != customEditThrowable {
			t.Fatalf("throwable picker did not open: hit=%d editor=%d", hit, g.customEditor[0])
		}
		g.customCardY[0] = -300
		item := developedThrowableSelectionItems()[kind]
		hit = g.customPlayerEditorHitAt(20+item.Rect.X+item.Rect.W/2, -300+item.Rect.Y+item.Rect.H/2)
		if !g.activateCustomPlayerEditorHit(hit) || g.customPlayers[0].Throwable != kind || g.customEditor[0] != 0 {
			t.Fatalf("throwable picker choice %d failed: hit=%d selected=%d editor=%d", kind, hit, g.customPlayers[0].Throwable, g.customEditor[0])
		}
	}
}

func TestCustomEditorSelectsDevelopedStarterPistol(t *testing.T) {
	g := &Game{}
	g.customPlayers = sourceDefaultCustomPlayers()
	g.customPlayers[0].Type = 1
	g.customEditor[0] = customEditGun
	g.customCardY[0] = -300
	items := developedCustomSelectionItems(customEditGun)
	item := items[len(items)-1]
	x := 20 + item.Rect.X + item.Rect.W/2
	y := -300 + item.Rect.Y + item.Rect.H/2
	hit := g.customPlayerEditorHitAt(x, y)
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

func TestCustomEditorHitCoordinatesPreserveEditorKind(t *testing.T) {
	for _, editor := range []int{customEditShirt, customEditHat1, customEditGun, customEditPerk, customEditHat2} {
		items := developedCustomSelectionItems(editor)
		if len(items) == 0 {
			t.Fatalf("editor%d has no items", editor)
		}
		g := &Game{}
		g.customPlayers = sourceDefaultCustomPlayers()
		g.campaignLevels[1] = 2
		g.campaignLevels[4] = 2
		g.campaignLevels[5] = 2
		g.customPlayers[0].Type = 1
		g.customEditor[0] = editor
		g.customCardY[0] = -300
		item := items[0]
		x := 20 + item.Rect.X + item.Rect.W/2
		y := -300 + item.Rect.Y + item.Rect.H/2
		hit := g.customPlayerEditorHitAt(x, y)
		_, gotEditor, gotNumber, ok := decodeCustomSelectHit(hit)
		if !ok || gotEditor != editor || gotNumber != item.Number {
			t.Fatalf("editor%d source cell -> hit=%d decoded editor=%d number=%d ok=%v; want editor=%d number=%d", editor, hit, gotEditor, gotNumber, ok, editor, item.Number)
		}
	}
}

func TestSourceColorSelectorMapping(t *testing.T) {
	cardX, cardY := 20.0, 100.0
	for n := 1; n <= 10; n++ {
		// Symbol856 computes number=floor(localX/25)*5+floor(localY/25)+1.
		col := (n - 1) / 5
		row := (n - 1) % 5
		localX := float64(col)*25 + 12.5
		localY := float64(row)*25 + 12.5
		// Exact Symbol856 matrix from Symbol1302.
		x := cardX + 22.3 + 1.227752685546875*localY
		y := cardY + 349.55 - 0.4799957275390625*localX
		got, ok := sourceColorNumberAt(cardX, cardY, x, y)
		if !ok || got != n {
			t.Fatalf("color%d center -> got=%d ok=%v at %.2f,%.2f", n, got, ok, x, y)
		}
	}
}

func TestCustomColorSelectorOpensCombinedPaletteAndSelectsEveryColor(t *testing.T) {
	g := &Game{}
	g.customPlayers = sourceDefaultCustomPlayers()
	g.customPlayers[0].Type = 1
	g.customCardY[0] = 100
	cardX, cardY := 20.0, 100.0
	r := sourceColorSelectorRect(cardX, cardY)
	hit := g.customPlayerEditorHitAt(r.X+r.W/2, r.Y+r.H/2)
	if hit != customEditHit(0, customEditColor) {
		t.Fatalf("COLOR field hit=%d want editor hit=%d", hit, customEditHit(0, customEditColor))
	}
	if !g.activateCustomPlayerEditorHit(hit) || g.customEditor[0] != customEditColor {
		t.Fatalf("COLOR field did not open combined palette: editor=%d", g.customEditor[0])
	}
	g.customCardY[0] = -300
	items := developedCustomSelectionItems(customEditColor)
	want := gm1SelectableColorCount
	if gm2CosmeticsAvailable() {
		want = maxDevelopedColorValue()
	}
	if len(items) != want {
		t.Fatalf("combined color count=%d want%d", len(items), want)
	}
	for _, item := range items {
		x := 20 + item.Rect.X + item.Rect.W/2
		y := -300 + item.Rect.Y + item.Rect.H/2
		hit = g.customPlayerEditorHitAt(x, y)
		_, editor, number, ok := decodeCustomSelectHit(hit)
		if !ok || editor != customEditColor || number != item.Number {
			t.Fatalf("color%d cell hit=%d decoded editor=%d number=%d ok=%v", item.Number, hit, editor, number, ok)
		}
		if !g.activateCustomPlayerEditorHit(hit) || g.customPlayers[0].Color != item.Number {
			t.Fatalf("color%d activation failed, stored=%d", item.Number, g.customPlayers[0].Color)
		}
		g.customEditor[0] = customEditColor
	}
}
