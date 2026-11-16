package game

import (
	"fmt"
	"testing"
)

// Gun Library, pause, results and credits (phase 7), driven through their
// run*Update entry points.

func newLibraryGame(t *testing.T) (*Game, kitScreen) {
	t.Helper()
	g := New()
	g.screen = screenGunLibrary
	k := kitScreen{t: t, st: &g.libraryUI, run: g.runGunLibraryUpdate}
	k.idle()
	return g, k
}

func TestGunLibraryTabsAndUniqueIDs(t *testing.T) {
	g, k := newLibraryGame(t)
	for view, label := range gunLibraryViewLabels {
		k.click("lib.tabs/" + label)
		if g.gunLibraryCurrentView() != view {
			t.Fatalf("%s tab: view %d", label, g.gunLibraryCurrentView())
		}
		k.uniqueIDs(label)
		k.item("lib.back")
		k.item("lib.test")
	}
	if g.organizedGunLibraryThrowableTab() {
		t.Fatal("POWERUPS view still has the THROWABLES category")
	}
	k.key(uiInput{PrevTab: true})
	if g.gunLibraryCurrentView() != gunLibraryViewGear {
		t.Fatalf("Q: view %d", g.gunLibraryCurrentView())
	}
	k.key(uiInput{Esc: true})
	if !g.fadeActive || g.fadeTarget != screenMainMenu {
		t.Fatalf("ESC: fade=%v target=%d", g.fadeActive, g.fadeTarget)
	}
}

// Every library weapon is reachable through the category list and pages.
func TestGunLibraryPagesShowEveryWeapon(t *testing.T) {
	g, k := newLibraryGame(t)
	for i := range g.campaignGuns {
		g.campaignGuns[i] = true
	}
	seen := map[int]bool{}
	for cat := 0; cat < gunLibraryThrowableCategory; cat++ {
		g.gunLibraryCategory = cat
		g.gunLibraryPage = 0
		pages := organizedGunLibraryPageCount(organizedGunLibraryTabs[cat].category)
		for page := 0; page < pages; page++ {
			g.gunLibraryPage = page
			k.idle()
			for _, gun := range g.gunLibraryPageWeapons() {
				it := k.item(fmt.Sprintf("lib.gun.%d", gun))
				if it.disabled {
					t.Fatalf("unlocked gun %d is disabled", gun)
				}
				seen[gun] = true
			}
		}
	}
	for _, gun := range organizedGunLibraryNumbers(0) {
		if !seen[gun] {
			t.Fatalf("weapon %d has no library tile", gun)
		}
	}
	for _, gun := range []int{weaponHarpoonGun, weaponLeafBlower, weaponBaseballBat, 67, 86} {
		if !seen[gun] {
			t.Fatalf("new weapon %d missing from the library", gun)
		}
	}
}

func TestGunLibraryLockedGunIsInertAndTestStarts(t *testing.T) {
	g, k := newLibraryGame(t)
	g.campaignGuns = [57]bool{}
	k.idle()
	if !k.item("lib.test").disabled {
		t.Fatal("TEST enabled without a selection")
	}
	k.click("lib.test")
	if g.fadeActive {
		t.Fatal("disabled TEST started a test")
	}
	// Gun 10 is a GM1 campaign unlock.
	g.gunLibraryCategory, g.gunLibraryPage = 0, 0
	for p := 0; p < organizedGunLibraryPageCount(0); p++ {
		g.gunLibraryPage = p
		found := false
		for _, gun := range g.gunLibraryPageWeapons() {
			found = found || gun == 10
		}
		if found {
			break
		}
	}
	k.idle()
	if !k.item("lib.gun.10").disabled {
		t.Fatal("locked gun 10 is selectable")
	}
	k.click("lib.gun.10")
	if g.gunLibrarySelected != 0 {
		t.Fatalf("locked gun selected: %d", g.gunLibrarySelected)
	}
	g.gunLibraryPage = 0
	k.idle()
	k.click("lib.gun.1")
	if g.gunLibrarySelected != 1 {
		t.Fatalf("starter handgun not selected: %d", g.gunLibrarySelected)
	}
	k.click("lib.test")
	if !g.fadeActive || g.fadeTarget != screenGameplay || !g.gototest || g.testGunNumber != 1 {
		t.Fatalf("TEST: fade=%v target=%d test=%v gun=%d", g.fadeActive, g.fadeTarget, g.gototest, g.testGunNumber)
	}
}

func TestGunLibraryThrowableTestAndInfoTabs(t *testing.T) {
	g, k := newLibraryGame(t)
	k.click("lib.tabs/THROWABLES")
	k.click("lib.throw.2")
	if g.gunLibraryThrowableSelected != 2 || !g.organizedGunLibraryThrowableTab() {
		t.Fatalf("throwable %d tab=%v", g.gunLibraryThrowableSelected, g.organizedGunLibraryThrowableTab())
	}
	k.click("lib.tabs/GEAR")
	shield := -1
	for i, a := range selectableAbilityNumbers() {
		if a == abilityBallisticShield {
			shield = i
		}
	}
	if shield < 0 {
		t.Fatal("ballistic shield is not a gear entry")
	}
	k.click(fmt.Sprintf("lib.gear.%d", abilityBallisticShield))
	if g.gunLibraryGear != shield || !k.item("lib.test").disabled {
		t.Fatalf("gear pick %d test=%v", g.gunLibraryGear, k.item("lib.test").disabled)
	}
	k.click("lib.tabs/POWERUPS")
	for kind := 0; kind < powerupKindCount; kind++ {
		k.item(fmt.Sprintf("lib.power.%d", kind))
		if gunLibraryPowerupInfo[kind] == "" {
			t.Fatalf("powerup %d has no description", kind)
		}
	}
	k.click(fmt.Sprintf("lib.power.%d", powerupSlowMo))
	if g.gunLibraryGear != powerupSlowMo {
		t.Fatal("SLOW-MO tile not selected")
	}
	k.click("lib.tabs/THROWABLES")
	k.click("lib.test")
	if !g.fadeActive || !g.testThrowableMode {
		t.Fatalf("throwable TEST: fade=%v mode=%v", g.fadeActive, g.testThrowableMode)
	}
	img := newTestImage()
	g.fadeActive = false
	for view := range gunLibraryViewLabels {
		g.setGunLibraryView(view)
		g.drawGunLibraryMenu(img)
	}
}

// ---------------------------------------------------------------------------
// Pause

func newPausedMatch(t *testing.T) (*Game, kitScreen) {
	t.Helper()
	g := New()
	g.customLives = 3
	g.customSeriesRounds = 2
	for i := range g.customPlayers {
		g.customPlayers[i].Type = 0
	}
	g.customPlayers[0].Type = 1
	g.customPlayers[1].Type = 2
	g.startCustomGame()
	g.fadeActive = false
	g.screen = screenGameplay
	g.openPause()
	k := kitScreen{t: t, st: &g.pauseUI, run: g.runPauseUpdate}
	k.idle()
	return g, k
}

func TestPauseMenuResumeOptionsAndEsc(t *testing.T) {
	g, k := newPausedMatch(t)
	k.uniqueIDs("pause")
	k.keyboardReaches("pause.resume", "pause.restart", "pause.options", "pause.quit")
	k.click("pause.options")
	if !g.pauseOptions || !g.paused {
		t.Fatal("OPTIONS did not open the options view")
	}
	k.uniqueIDs("pause options")
	before := g.soundVolume
	g.pauseUI.focus = "pause.sfx"
	k.key(uiInput{Left: true})
	if g.soundVolume >= before && before > 0 {
		t.Fatalf("LEFT on the SFX slider: %v -> %v", before, g.soundVolume)
	}
	k.key(uiInput{Esc: true})
	if g.pauseOptions || !g.paused {
		t.Fatalf("ESC in options: options=%v paused=%v", g.pauseOptions, g.paused)
	}
	k.key(uiInput{Esc: true})
	if g.paused {
		t.Fatal("ESC did not resume")
	}
	g.openPause()
	k.idle()
	k.click("pause.resume")
	if g.paused {
		t.Fatal("RESUME did not resume")
	}
}

func TestPauseRestartStartsTheSameRound(t *testing.T) {
	g, k := newPausedMatch(t)
	round := g.customSeriesRound
	oldPlayers := g.players
	g.players[0].Kills = 5
	k.click("pause.restart")
	if g.paused || !g.fadeActive || g.fadeTarget != screenGameplay {
		t.Fatalf("RESTART: paused=%v fade=%v target=%d", g.paused, g.fadeActive, g.fadeTarget)
	}
	if g.customSeriesRound != round || len(g.players) != len(oldPlayers) || g.players[0] == oldPlayers[0] || g.players[0].Kills != 0 {
		t.Fatalf("RESTART did not rebuild round %d (now %d)", round, g.customSeriesRound)
	}
}

func TestPauseQuitUsesTheModeExit(t *testing.T) {
	g, k := newPausedMatch(t)
	k.click("pause.quit")
	if g.paused || g.customSeriesRound != 0 || !g.fadeActive || g.fadeTarget != screenCustomGame {
		t.Fatalf("QUIT TO SETUP: paused=%v round=%d fade=%v target=%d", g.paused, g.customSeriesRound, g.fadeActive, g.fadeTarget)
	}

	// Campaign: back to the mission list through the end-game fade.
	g = New()
	g.campaignLevel = 2
	g.startCampaignMission()
	g.fadeActive = false
	g.screen = screenGameplay
	g.openPause()
	k = kitScreen{t: t, st: &g.pauseUI, run: g.runPauseUpdate}
	k.idle()
	if label, _ := g.pauseQuitLabel(); label != "QUIT MISSION" {
		t.Fatalf("campaign quit label %q", label)
	}
	k.click("pause.quit")
	if !g.fadeActive || g.fadeTarget != screenCampaign || g.fadePurpose != fadePurposeEndGame {
		t.Fatalf("QUIT MISSION: fade=%v target=%d purpose=%d", g.fadeActive, g.fadeTarget, g.fadePurpose)
	}
}

func TestPauseRestartDisabledForZombieAndGuests(t *testing.T) {
	g, k := newPausedMatch(t)
	g.zombie = &zombieMatch{}
	k.idle()
	if !k.item("pause.restart").disabled {
		t.Fatal("RESTART enabled in a zombie run")
	}
	players := g.players
	k.click("pause.restart")
	if !g.paused || g.fadeActive || g.players[0] != players[0] {
		t.Fatal("disabled RESTART changed the match")
	}
	g.zombie = nil
	g.netplay = &netplaySession{mode: netplayHost}
	g.onlineMatchSetup = true
	if ok, _ := g.pauseRestart(); !ok {
		t.Fatal("the online host cannot restart")
	}
	g.netplay = &netplaySession{mode: netplayClient}
	if ok, _ := g.pauseRestart(); ok {
		t.Fatal("a guest can restart")
	}
	g.netplay = nil
}

func TestGuestMenuKeepsMatchRunningAndLeaves(t *testing.T) {
	g, _ := newPausedMatch(t)
	g.paused = false
	g.netplay = &netplaySession{mode: netplayClient}
	g.onlineMatchSetup = true
	g.openGuestMenu()
	k := kitScreen{t: t, st: &g.pauseUI, run: g.runGuestMenuUpdate}
	k.idle()
	k.uniqueIDs("guest menu")
	if !k.item("guest.restart").disabled {
		t.Fatal("guest RESTART is enabled")
	}
	k.click("guest.restart")
	if !g.guestMenuOpen || g.paused {
		t.Fatal("disabled guest RESTART changed state")
	}
	k.key(uiInput{Esc: true})
	if g.guestMenuOpen || g.paused {
		t.Fatalf("ESC: open=%v paused=%v", g.guestMenuOpen, g.paused)
	}
	g.openGuestMenu()
	k.idle()
	k.click("guest.leave")
	if g.netplay != nil || g.screen != screenCustomGame || !g.customNetworkOpen || g.guestMenuOpen {
		t.Fatalf("LEAVE: netplay=%v screen=%d network=%v", g.netplay != nil, g.screen, g.customNetworkOpen)
	}
	g.drawPauseOverlay(newTestImage())
}

// ---------------------------------------------------------------------------
// Results

func newPostGame(t *testing.T, rounds int) (*Game, kitScreen) {
	t.Helper()
	g, _ := newPausedMatch(t)
	g.paused = false
	g.customSeriesRounds = rounds
	g.customSeriesRound = 1
	g.screen = screenPostGame
	k := kitScreen{t: t, st: &g.resultsUI, run: g.runResultsUpdate}
	k.idle()
	return g, k
}

func TestPostGameContinueAndNextRound(t *testing.T) {
	g, k := newPostGame(t, 1)
	k.uniqueIDs("post game")
	if k.has("res.leave") {
		t.Fatal("local results show LEAVE SESSION")
	}
	k.click("res.continue")
	if !g.fadeActive || g.fadeTarget != screenSeriesSummary { // a 1-round series still ends on its summary
		t.Fatalf("CONTINUE: fade=%v target=%d", g.fadeActive, g.fadeTarget)
	}

	g, k = newPostGame(t, 3)
	k.key(uiInput{Enter: true}) // inside the grace period: nothing has focus
	if g.customSeriesRound != 1 || g.fadeActive {
		t.Fatalf("ENTER right after the match skipped the results: round %d", g.customSeriesRound)
	}
	for i := 0; i < resultsFocusGrace; i++ {
		k.idle()
	}
	if g.resultsUI.focus != "res.continue" {
		t.Fatalf("focus after the grace period = %q", g.resultsUI.focus)
	}
	k.key(uiInput{Enter: true}) // CONTINUE has the focus now
	if g.customSeriesRound != 2 {
		t.Fatalf("NEXT ROUND (ENTER): round %d", g.customSeriesRound)
	}
	g, k = newPostGame(t, 3)
	k.key(uiInput{Esc: true})
	if g.customSeriesRound != 2 {
		t.Fatalf("ESC: round %d", g.customSeriesRound)
	}
	g.drawResultsMenu(newTestImage())
}

func TestPostGameOnlineReady(t *testing.T) {
	g, k := newPostGame(t, 1)
	g.netplay = &netplaySession{mode: netplayClient}
	g.netplay.assignedPlayerID.Store(2)
	g.onlineMatchSetup = true
	g.onlinePlayerSlots = [4]bool{true, true}
	k.idle()
	k.uniqueIDs("guest post game")
	k.click("res.continue")
	if !g.postGameReady[1] || g.fadeActive {
		t.Fatalf("READY: ready=%v fade=%v", g.postGameReady, g.fadeActive)
	}
	if !k.item("res.continue").disabled {
		t.Fatal("READY! is still clickable")
	}
	k.key(uiInput{Esc: true}) // the guest's ESC is handled by updateOnlineClientPostGame
	if g.netplay == nil {
		t.Fatal("results ESC disconnected inside the kit pass")
	}
	k.click("res.leave")
	if g.netplay != nil || g.screen != screenCustomGame {
		t.Fatal("LEAVE SESSION did not disconnect")
	}
}

func TestSeriesSummaryBackAndGuestWait(t *testing.T) {
	g, k := newPostGame(t, 3)
	g.customSeriesStats[0] = customSeriesPlayerStats{Name: "A", MapWins: 2, RoundsPlayed: 3}
	g.customSeriesStats[1] = customSeriesPlayerStats{Name: "B", MapWins: 1, RoundsPlayed: 3}
	g.screen = screenSeriesSummary
	k.idle()
	k.uniqueIDs("series")
	if rows := g.seriesRanking(); len(rows) != 2 || rows[0].stat.Name != "A" {
		t.Fatalf("ranking %+v", rows)
	}
	g.drawResultsMenu(newTestImage())
	k.click("res.done")
	if !g.fadeActive || g.fadeTarget != screenCustomGame || g.customSeriesRound != 0 {
		t.Fatalf("BACK: fade=%v target=%d round=%d", g.fadeActive, g.fadeTarget, g.customSeriesRound)
	}
	g.fadeActive = false
	g.netplay = &netplaySession{mode: netplayClient}
	k.idle()
	if !k.item("res.wait").disabled || k.has("res.done") {
		t.Fatal("guest can leave the series summary for everyone")
	}
	g.netplay = nil
}

// ---------------------------------------------------------------------------
// Credits

func TestCreditsLinksAndBack(t *testing.T) {
	var opened []string
	old := openExternalURL
	openExternalURL = func(url string) { opened = append(opened, url) }
	defer func() { openExternalURL = old }()
	g := New()
	g.screen = screenCredits
	k := kitScreen{t: t, st: &g.creditsUI, run: g.runCreditsUpdate}
	k.idle()
	k.uniqueIDs("credits")
	k.click("cred.kevin")
	k.click("cred.music")
	if len(opened) != 2 || opened[0] != "http://www.thekevingu.com" || opened[1] != "http://www.incompetech.com" {
		t.Fatalf("links %v", opened)
	}
	g.drawCreditsMenu(newTestImage())
	k.key(uiInput{Esc: true})
	if !g.fadeActive || g.fadeTarget != screenMainMenu {
		t.Fatal("ESC did not return to the main menu")
	}
}

// Review fixes: guest menu lifetime, RESTART per mode and its failure path,
// FINISH MISSION, the campaign result modal's first frame.

func TestGuestMenuClosesWhenTheScreenChanges(t *testing.T) {
	g, _ := newPausedMatch(t)
	g.paused = false
	g.netplay = &netplaySession{mode: netplayClient}
	g.onlineMatchSetup = true
	g.openGuestMenu()
	g.screen = screenPostGame // the host ended the match
	_ = g.updateNetClient()
	if g.guestMenuOpen {
		t.Fatal("guest menu survived the switch to the results screen")
	}
	g.screen = screenGameplay
	if !g.guestMenuMayOpen(false) {
		t.Fatal("guest menu cannot open in a running match")
	}
	g.GameWin = true
	if g.guestMenuMayOpen(false) {
		t.Fatal("guest menu opens over the winner sequence")
	}
	g.GameWin = false
	g.fadeActive = true
	if g.guestMenuMayOpen(false) {
		t.Fatal("guest menu opens during a fade")
	}
	g.fadeActive = false
	if g.guestMenuMayOpen(true) {
		t.Fatal("the ESC that closed the zombie shop opens the guest menu")
	}
	g.netplay = nil
}

func pauseIn(t *testing.T, g *Game) kitScreen {
	t.Helper()
	g.fadeActive = false
	g.fadeFrame = 0
	g.screen = screenGameplay
	g.openPause()
	k := kitScreen{t: t, st: &g.pauseUI, run: g.runPauseUpdate}
	k.idle()
	return k
}

func TestPauseRestartCampaignAndGunTest(t *testing.T) {
	g := New()
	g.campaignSetExtra = true
	g.campaignLevel = 1
	g.startExtraCampaignMission()
	old := g.players[0]
	k := pauseIn(t, g)
	k.click("pause.restart")
	if !g.fadeActive || g.fadeTarget != screenGameplay || !g.campaignMode || !g.campaignSetExtra || g.campaignLevel != 1 || g.players[0] == old || g.paused {
		t.Fatalf("campaign RESTART: fade=%v campaign=%v extra=%v level=%d", g.fadeActive, g.campaignMode, g.campaignSetExtra, g.campaignLevel)
	}
	// The restarted round stays frozen under the fade until frame 10.
	x := g.players[0].X
	g.players[0].VX = 5
	for g.fadeFrame < 9 {
		if !g.updateGameplayInteractionInput() {
			t.Fatalf("round simulated at fade frame %d", g.fadeFrame)
		}
	}
	if g.players[0].X != x {
		t.Fatal("player moved under the restart fade")
	}

	for _, throwables := range []bool{false, true} {
		g = New()
		g.gunLibrarySelected = 10
		g.campaignGuns[0] = true
		if throwables {
			g.setGunLibraryView(gunLibraryViewThrowables)
			g.gunLibraryThrowableSelected = 3
		}
		g.startGunLibraryTest()
		k = pauseIn(t, g)
		if label, _ := g.pauseQuitLabel(); label != "BACK TO LIBRARY" {
			t.Fatalf("gun test quit label %q", label)
		}
		k.click("pause.restart")
		if !g.fadeActive || !g.gototest || g.testThrowableMode != throwables || (!throwables && g.testGunNumber != 10) {
			t.Fatalf("gun test RESTART (throwables=%v): fade=%v test=%v mode=%v gun=%d", throwables, g.fadeActive, g.gototest, g.testThrowableMode, g.testGunNumber)
		}
	}
}

func TestPauseRestartFailureKeepsThePausedMatch(t *testing.T) {
	g := New()
	g.gunLibrarySelected = 1
	g.startGunLibraryTest()
	k := pauseIn(t, g)
	g.gunLibrarySelected = 0 // startGunLibraryTest now refuses
	players := g.players
	k.click("pause.restart")
	if g.fadeActive || !g.paused || !g.gototest || g.players[0] != players[0] {
		t.Fatalf("failed RESTART: fade=%v paused=%v test=%v", g.fadeActive, g.paused, g.gototest)
	}
}

func TestPauseRestartOnlineHost(t *testing.T) {
	g := newOnlineHostLobby(t, onlineSessionOpen)
	g.customPlayers[1].Type = 2 // an AI bot makes two participants
	g.customSeriesRounds = 2
	g.startCustomGame()
	if !g.fadeActive {
		t.Fatalf("online match did not start: %s", g.multiplayerMessage)
	}
	round := g.customSeriesRound
	old := g.players[0]
	k := pauseIn(t, g)
	g.postGameReady[0] = true
	k.click("pause.restart")
	if !g.fadeActive || g.customSeriesRound != round || g.players[0] == old || g.postGameReady[0] {
		t.Fatalf("host RESTART: fade=%v round=%d ready=%v", g.fadeActive, g.customSeriesRound, g.postGameReady)
	}
}

func TestPauseFinishMissionRecordsTheTutorial(t *testing.T) {
	g := New()
	g.campaignSetGM2 = true
	g.campaignLevel = 1
	g.startGM2CampaignMission()
	k := pauseIn(t, g)
	if label, _ := g.pauseQuitLabel(); label != "FINISH MISSION" {
		t.Fatalf("GM2 tutorial quit label %q", label)
	}
	k.click("pause.quit")
	for i := 0; i < 12; i++ {
		g.updateFade()
	}
	if g.gm2CampaignLevels[0] != 2 || !g.campaignResult.open || !g.campaignResult.won {
		t.Fatalf("tutorial finish: level=%d result=%+v", g.gm2CampaignLevels[0], g.campaignResult)
	}
}

func TestCampaignResultModalBlocksTheListOnItsFirstFrame(t *testing.T) {
	g, k := newCampaignUIGame(t, campaignSetGM1)
	k.idle() // list registry without the modal
	g.recordCampaignResult(true)
	r := k.item(campaignMissionID(2)).rect
	x, y := r.X+r.W/2, r.Y+r.H/2
	k.run(uiInput{X: x, Y: y, Moved: true})
	k.run(uiInput{X: x, Y: y, Pressed: true, Down: true})
	k.run(uiInput{X: x, Y: y, Released: true})
	if g.campaignBriefingOpen() {
		t.Fatal("a mission opened under the result modal")
	}
}
