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

func TestCustomEditorPerkCampaignLocksMatchSymbol831(t *testing.T) {
	g := &Game{}
	if g.customSelectionUnlocked(customEditPerk, 3) || g.customSelectionUnlocked(customEditPerk, 6) || g.customSelectionUnlocked(customEditPerk, 9) {
		t.Fatal("campaign-locked source perks started unlocked")
	}
	g.campaignLevels[1] = 2
	g.campaignLevels[4] = 2
	g.campaignLevels[5] = 2
	for _, perk := range []int{3, 6, 9} {
		if !g.customSelectionUnlocked(customEditPerk, perk) {
			t.Fatalf("perk%d did not unlock from source campaign level", perk)
		}
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

func TestCustomEditorHitCoordinatesPreserveEditorKind(t *testing.T) {
	for _, editor := range []int{customEditShirt, customEditHat1, customEditGun, customEditPerk, customEditHat2} {
		items := sourceCustomSelectionItems()[editor]
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

func TestCustomColorSelectorHitAndActivationAllTen(t *testing.T) {
	g := &Game{}
	g.customPlayers = sourceDefaultCustomPlayers()
	g.customPlayers[0].Type = 1
	g.customCardY[0] = 100
	cardX, cardY := 20.0, 100.0
	for n := 1; n <= 10; n++ {
		col := (n - 1) / 5
		row := (n - 1) % 5
		localX := float64(col)*25 + 12.5
		localY := float64(row)*25 + 12.5
		x := cardX + 22.3 + 1.227752685546875*localY
		y := cardY + 349.55 - 0.4799957275390625*localX
		hit := g.customPlayerEditorHitAt(x, y)
		wantHit := customColorHit(0, n)
		if hit != wantHit {
			t.Fatalf("color%d hit=%d want=%d", n, hit, wantHit)
		}
		if !g.activateCustomPlayerEditorHit(hit) || g.customPlayers[0].Color != n {
			t.Fatalf("color%d activation failed, stored=%d", n, g.customPlayers[0].Color)
		}
	}
}
