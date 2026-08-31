package game

import "testing"

func TestCustomV2ModeButtonsSelectEveryMode(t *testing.T) {
	g := New()
	for _, row := range customV2ModeRows {
		hit := g.customV2HitAt(row.rect.X+10, row.rect.Y+10)
		wantHit := menuHitModeBase + row.mode
		if hit != wantHit {
			t.Fatalf("mode %d hit=%d want=%d", row.mode, hit, wantHit)
		}
		g.activateCustomV2Hit(hit)
		if g.customMode != row.mode {
			t.Fatalf("mode button selected %d want %d", g.customMode, row.mode)
		}
	}
}

func TestLocalWizardAdvancesMatchMapsPlayers(t *testing.T) {
	g := New()
	g.customOnlineMode = false
	g.customPage = customPageGame
	g.activateCustomV2Hit(menuHitCustomV2Start)
	if g.customPage != customPageMaps {
		t.Fatalf("local after match page=%d want maps", g.customPage)
	}
	g.activateCustomV2Hit(menuHitCustomV2Start)
	if g.customPage != customPagePlayers {
		t.Fatalf("local after maps page=%d want players", g.customPage)
	}
}

func TestOnlineHostStepHeadersAreClickable(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.customPage = customPagePlayers
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	g.netplay = host

	wantHits := []int{menuHitCustomV2TabSession, menuHitCustomV2TabMatch, menuHitCustomV2TabPlaylist, menuHitCustomV2TabPlayers}
	for i, r := range customV2OnlineStepRects {
		if got := g.customV2HitAt(r.X+r.W/2, r.Y+r.H/2); got != wantHits[i] {
			t.Fatalf("online step %d hit=%d want=%d", i+1, got, wantHits[i])
		}
	}

	originalHost := g.netplay
	g.activateCustomV2Hit(menuHitCustomV2TabSession)
	if g.netplay != originalHost || !g.customNetworkOpen || !g.customHostSetupOpen {
		t.Fatalf("opening SESSION destroyed host or failed to open settings: hostSame=%v network=%v setup=%v", g.netplay == originalHost, g.customNetworkOpen, g.customHostSetupOpen)
	}
	g.customNetworkOpen = false
	g.customHostSetupOpen = false
	g.activateCustomV2Hit(menuHitCustomV2TabPlaylist)
	if g.customPage != customPageMaps {
		t.Fatalf("MAPS header did not navigate back while hosting: page=%d", g.customPage)
	}
}

func TestOnlineRemoteCardCannotKickThroughOldClearHitbox(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.customPage = customPagePlayers
	g.customPlayers[1].Type = 1
	g.customCardY[1] = 100
	g.netplay = &netplaySession{mode: netplayHost}
	g.onlinePlayerSlots[1] = true

	cardX := 20.0 + 220
	oldClear := sourceMenuHitRect(customPlayerTypeButtonLocal, cardX+20, g.customCardY[1]+9.15, 1.0775146484375, 1)
	if hit := g.customV2HitAt(oldClear.X+oldClear.W/2, oldClear.Y+oldClear.H/2); hit != menuHitNone {
		t.Fatalf("remote card old CLEAR SLOT area is still active: hit=%d", hit)
	}
	kick := customV2OnlineSlotRemoveRect(1)
	if hit := g.customV2HitAt(kick.X+kick.W/2, kick.Y+kick.H/2); hit != menuHitCustomV2SlotRemoveBase+1 {
		t.Fatalf("dedicated KICK PLAYER hit=%d", hit)
	}
}

func TestOnlineHostWizardAdvancesModeMapsSlots(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.customPage = customPageGame
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	g.netplay = host
	g.activateCustomV2Hit(menuHitCustomV2Start)
	if g.customPage != customPageMaps {
		t.Fatalf("after mode page=%d want maps", g.customPage)
	}
	g.activateCustomV2Hit(menuHitCustomV2Start)
	if g.customPage != customPagePlayers {
		t.Fatalf("after maps page=%d want players", g.customPage)
	}
}

func TestCustomPlaylistStoresDifferentMapForEveryRound(t *testing.T) {
	g := New()
	g.customSeriesRounds = 3
	g.customMap = 1
	g.setCustomSeriesMap(1, 2)
	g.setCustomSeriesMap(2, 7)
	g.setCustomSeriesMap(3, 11)

	want := []int{2, 7, 11}
	for round, mapValue := range want {
		if got := g.customSeriesMapForRound(round + 1); got != mapValue {
			t.Fatalf("round %d map=%d want=%d", round+1, got, mapValue)
		}
	}
	if !g.customSeriesPlaylistConfigured {
		t.Fatal("playlist was not marked configured")
	}
}

func TestCustomPlaylistInitializesAllRoundsFromOldSingleMap(t *testing.T) {
	g := New()
	g.customSeriesRounds = 4
	g.customMap = 5
	g.setCustomSeriesMap(2, 8)
	if got := g.customSeriesMapForRound(1); got != 5 {
		t.Fatalf("round1=%d want5", got)
	}
	if got := g.customSeriesMapForRound(2); got != 8 {
		t.Fatalf("round2=%d want8", got)
	}
	if got := g.customSeriesMapForRound(3); got != 5 {
		t.Fatalf("round3=%d want5", got)
	}
	if got := g.customSeriesMapForRound(4); got != 5 {
		t.Fatalf("round4=%d want5", got)
	}
}

func TestCustomPlaylistApplyToAll(t *testing.T) {
	g := New()
	g.customSeriesRounds = 3
	g.setCustomSeriesMap(1, 2)
	g.setCustomSeriesMap(2, 7)
	g.setCustomSeriesMap(3, 11)
	g.customPlaylistSelectedRound = 2
	g.activateCustomV2Hit(menuHitCustomV2ApplyAll)
	for round := 1; round <= 3; round++ {
		if got := g.customSeriesMapForRound(round); got != 7 {
			t.Fatalf("round %d map=%d want7", round, got)
		}
	}
}

func TestOnlinePreHostCustomGameRemainsEditable(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	if g.netplay != nil {
		t.Fatal("new game unexpectedly has netplay")
	}
	if !g.customModifierEditingAllowed() {
		t.Fatal("online pre-host setup must allow modifier editing")
	}
	row := customV2ModeRows[3] // Gun Game
	hit := g.customV2HitAt(row.rect.X+10, row.rect.Y+10)
	g.activateCustomV2Hit(hit)
	if g.customMode != SourceGameModeGunGame {
		t.Fatalf("online pre-host mode=%d want gun game", g.customMode)
	}
}

func TestReturningFromHostSetupToLobbyKeepsHostAlive(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	if err := g.StartNetHost("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer g.CloseNetplay()
	host := g.netplay
	g.returnToOnlineLobbyFromCustomSetup()
	if g.netplay != host || g.netplay == nil {
		t.Fatal("back to lobby destroyed the active host")
	}
	if g.customNetworkOpen {
		t.Fatal("back to lobby unexpectedly opened the server browser")
	}
}

func TestCustomPlaylistRoundSelectionIsIndependent(t *testing.T) {
	g := New()
	g.customSeriesRounds = 3
	g.customPage = customPageMaps
	g.ensureCustomPlaylistState()
	for round := 1; round <= 3; round++ {
		row := round - 1
		r := customV2RoundRect(row)
		hit := g.customV2HitAt(r.X+5, r.Y+5)
		if hit != menuHitCustomV2RoundBase+round {
			t.Fatalf("round %d hit=%d", round, hit)
		}
		g.activateCustomV2Hit(hit)
		if g.customPlaylistSelectedRound != round {
			t.Fatalf("selected round=%d want=%d", g.customPlaylistSelectedRound, round)
		}
	}
}
