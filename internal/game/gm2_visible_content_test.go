package game

import "testing"

func TestGM2WeaponLibraryExposesAllTwentyAdditions(t *testing.T) {
	seen := map[int]gunLibraryButton{}
	for _, b := range developedGunLibraryButtons() {
		seen[b.Gun] = b
	}
	for n := 67; n <= 86; n++ {
		if !isGunLibraryGunNumber(n) {
			t.Fatalf("GM2 weapon %d is not accepted by Gun Library", n)
		}
		b, ok := seen[n]
		if !ok {
			t.Fatalf("GM2 weapon %d has no visible library card", n)
		}
		g := &Game{}
		for i := range g.campaignGuns {
			g.campaignGuns[i] = true
		}
		if hit := g.developedGunLibraryHitAt(b.X, b.Y); hit != n {
			t.Fatalf("GM2 weapon card %d hit=%d", n, hit)
		}
	}
}

func TestCombinedGunLibraryExposesWholePlayableCatalogue(t *testing.T) {
	buttons := developedGunLibraryButtons()
	if len(buttons) != 83 {
		t.Fatalf("combined weapon count=%d want83", len(buttons))
	}
	seen := map[int]bool{}
	for _, b := range buttons {
		if seen[b.Gun] {
			t.Fatalf("duplicate combined weapon card %d", b.Gun)
		}
		seen[b.Gun] = true
		if b.Gun >= 10 && b.Y > 404 {
			t.Fatalf("developed weapon %d spills below upper family panels at y=%.1f", b.Gun, b.Y)
		}
	}
	for _, n := range []int{1, 6, 10, 66, 67, 86} {
		if !seen[n] {
			t.Fatalf("combined weapon %d has no visible card", n)
		}
	}
}

func TestCustomGunLibraryPagesExposeEveryAvailableDefinedWeapon(t *testing.T) {
	want := availableCustomWeaponNumbers()
	if len(want) != 145 {
		t.Fatalf("available custom weapon count=%d want145", len(want))
	}
	if pages := developedGunLibraryPageCount(); pages != 10 {
		t.Fatalf("library page count=%d want10", pages)
	}

	seen := map[int]bool{}
	for page := 1; page < developedGunLibraryPageCount(); page++ {
		buttons := developedGunLibraryButtonsForPage(page)
		expected := developedCustomGunLibraryPageSize
		remaining := len(want) - (page-1)*developedCustomGunLibraryPageSize
		if remaining < expected {
			expected = remaining
		}
		if len(buttons) != expected {
			t.Fatalf("custom page %d card count=%d want%d", page, len(buttons), expected)
		}
		g := &Game{gunLibraryPage: page}
		for _, b := range buttons {
			if seen[b.Gun] {
				t.Fatalf("duplicate custom weapon card %d", b.Gun)
			}
			seen[b.Gun] = true
			if !isGunLibraryGunNumber(b.Gun) {
				t.Fatalf("custom weapon %d is rejected by Gun Library", b.Gun)
			}
			if hit := g.developedGunLibraryHitAt(b.X, b.Y); hit != b.Gun {
				t.Fatalf("custom page %d weapon %d hit=%d", page, b.Gun, hit)
			}
		}
	}
	for _, number := range want {
		if !seen[number] {
			t.Fatalf("available custom weapon %d has no library card", number)
		}
	}
}

func TestGM2FaceGridUsesAvailableEditorSpace(t *testing.T) {
	items := gm2FaceSelectionItems()
	if len(items) != maxDevelopedEyeValue() {
		t.Fatalf("combined face count=%d want%d", len(items), maxDevelopedEyeValue())
	}
	for _, item := range items {
		if item.Rect.W < 50 || item.Rect.H < 50 {
			t.Fatalf("GM2 face %d is still too small: %.1fx%.1f", item.Number, item.Rect.W, item.Rect.H)
		}
		// At the fully-open editor position localY=-400, all face tiles must
		// stay inside the 200x400 masked card and above the BACK control.
		x := item.Rect.X
		y := item.Rect.Y - 400
		if x < 0 || x+item.Rect.W > 200 || y < 45 || y+item.Rect.H > 320 {
			t.Fatalf("GM2 face %d outside editor area: x=%.1f y=%.1f w=%.1f h=%.1f", item.Number, x, y, item.Rect.W, item.Rect.H)
		}
	}
}

func TestGM2CampaignExposesSixteenMissions(t *testing.T) {
	for i := 0; i < 16; i++ {
		if gm2CampaignTitles[i] == "" {
			t.Fatalf("GM2 mission %d has no visible title", i+1)
		}
		r := gm2CampaignCardRect(i)
		if r.W <= 0 || r.H <= 0 {
			t.Fatalf("GM2 mission %d has no visible card", i+1)
		}
		if gm2CampaignMissions[i].MapNumber < 1 || gm2CampaignMissions[i].MapNumber > 21 {
			t.Fatalf("GM2 mission %d bad map=%d", i+1, gm2CampaignMissions[i].MapNumber)
		}
	}
}

func TestGM2MissionSelectionOpensCharacterSetup(t *testing.T) {
	g := New()
	g.screen = screenCampaign
	g.campaignSetGM2 = true
	g.openGM2CampaignMissionSetup(2)
	if g.campaignLevel != 2 || g.campaignPhase != 2 || g.campaignDetailAlpha != 0 {
		t.Fatalf("GM2 mission did not enter character setup: level=%d phase=%d alpha=%v", g.campaignLevel, g.campaignPhase, g.campaignDetailAlpha)
	}
	cx, cy := campaignCardX(0), g.campaignCardY[0]
	hit := g.campaignPlayerHitAt(cx+30, cy+170)
	if hit == campaignHitNone {
		t.Fatal("GM2 character card exposes no editable controls")
	}
}

func TestGM2SoloMissionsCloseSecondHumanSlot(t *testing.T) {
	g := New()
	g.campaignPlayers[1].Type = 1
	g.openGM2CampaignMissionSetup(1)
	if g.campaignPlayers[1].Type != 0 {
		t.Fatal("GM2 tutorial should remain a solo mission")
	}
}

func TestGM2CampaignEnemyCosmeticsResolveAcrossAllMissions(t *testing.T) {
	if !gm2CosmeticsAvailable() {
		t.Skip("GM2 source cosmetics are not available in this checkout")
	}
	a := LoadAssets()
	for missionIndex, mission := range gm2CampaignMissions {
		for enemyIndex, enemy := range mission.Enemies {
			if enemy.Color < 1 || enemy.Color > 20 {
				t.Fatalf("mission %d enemy %d color=%d outside GM2 palette", missionIndex+1, enemyIndex+1, enemy.Color)
			}
			if enemy.Eyes < 1 || enemy.Eyes > gm2EyesCount {
				t.Fatalf("mission %d enemy %d eyes=%d outside GM2 face set", missionIndex+1, enemyIndex+1, enemy.Eyes)
			}
			a.EnsureGM2PlayerColor(enemy.Color)
			if a.GM2LegColors[enemy.Color] == nil || a.GM2HandColors[enemy.Color] == nil || a.GM2TorsoColors[enemy.Color] == nil || a.GM2HeadColors[enemy.Color] == nil {
				t.Fatalf("mission %d enemy %q color %d has incomplete GM2 body rasters", missionIndex+1, enemy.Name, enemy.Color)
			}
			if a.GM2ShirtFrame(enemy.Shirt-1) == nil {
				t.Fatalf("mission %d enemy %q shirt %d missing", missionIndex+1, enemy.Name, enemy.Shirt)
			}
			if a.GM2HatFrame(enemy.Hat-1) == nil {
				t.Fatalf("mission %d enemy %q hat %d missing", missionIndex+1, enemy.Name, enemy.Hat)
			}
			if a.EyeFrame(enemy.Eyes-1) == nil {
				t.Fatalf("mission %d enemy %q eyes %d missing", missionIndex+1, enemy.Name, enemy.Eyes)
			}
		}
	}
}

func TestDevelopedPlayerEditorExposesFullGM2Cosmetics(t *testing.T) {
	if !gm2CosmeticsAvailable() {
		t.Skip("GM2 source cosmetics are not available in this checkout")
	}
	if got := len(developedCustomSelectionItems(customEditColor)); got != 30 {
		t.Fatalf("combined color choices=%d want30", got)
	}
	if got := len(developedCustomSelectionItems(customEditShirt)); got != 39 {
		t.Fatalf("combined shirt choices=%d want39 (12 GM1 + 27 GM2)", got)
	}
	if got := len(developedCustomSelectionItems(customEditHat1)) + len(developedCustomSelectionItems(customEditHat2)); got != 72 {
		t.Fatalf("combined hat choices=%d want72 (24 GM1 + 48 GM2)", got)
	}
	if got := len(developedCustomSelectionItems(customEditEyes)); got != 13 {
		t.Fatalf("combined face choices=%d want13 (1 GM1 + 12 GM2)", got)
	}

	a := LoadAssets()
	for value := gm1SelectableColorCount + 1; value <= maxDevelopedColorValue(); value++ {
		source, ok := gm2ColorSourceNumber(value)
		if !ok {
			t.Fatalf("combined color %d did not resolve to GM2", value)
		}
		a.EnsureGM2PlayerColor(source)
		if a.GM2LegColors[source] == nil || a.GM2HandColors[source] == nil || a.GM2TorsoColors[source] == nil || a.GM2HeadColors[source] == nil {
			t.Fatalf("combined color %d / GM2 %d has incomplete body", value, source)
		}
	}
	for value := gm1SelectableShirtCount + 1; value <= maxDevelopedShirtValue(); value++ {
		source, _ := gm2ShirtSourceNumber(value)
		if a.GM2ShirtFrame(source-1) == nil {
			t.Fatalf("combined shirt %d / GM2 %d missing", value, source)
		}
	}
	for value := gm1SelectableHatCount + 1; value <= maxDevelopedHatValue(); value++ {
		source, _ := gm2HatSourceNumber(value)
		if a.GM2HatFrame(source-1) == nil {
			t.Fatalf("combined hat %d / GM2 %d missing", value, source)
		}
	}
	for value := gm1SelectableEyesCount + 1; value <= maxDevelopedEyeValue(); value++ {
		source, _ := gm2EyeSourceNumber(value)
		if a.EyeFrame(source-1) == nil {
			t.Fatalf("combined face %d / GM2 %d missing", value, source)
		}
	}
}

func TestGM2CampaignProgressStaysSeparateFromGM1(t *testing.T) {
	g := New()
	gm1 := g.campaignLevels
	g.completeGM2CampaignLevel(2)
	if g.gm2CampaignLevels[1] != 2 || g.gm2CampaignLevels[2] != 1 {
		t.Fatalf("GM2 progression=%v want mission2 complete and mission3 unlocked", g.gm2CampaignLevels[:4])
	}
	if g.campaignLevels != gm1 {
		t.Fatalf("GM2 completion changed GM1 progress: got=%v want=%v", g.campaignLevels, gm1)
	}
}
