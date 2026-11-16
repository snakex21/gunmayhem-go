package game

import (
	"fmt"
	"testing"
)

// Campaign screens (phase 6) are driven through runCampaignUpdate, so a
// control is only reachable where it is drawn.

func newCampaignUIGame(t *testing.T, set int) (*Game, kitScreen) {
	t.Helper()
	g := New()
	g.screen = screenCampaign
	g.campaignSetGM2 = set == campaignSetGM2
	g.campaignSetExtra = set == campaignSetExtra
	g.enterCampaignMenu()
	k := kitScreen{t: t, st: &g.campaignUI, run: g.runCampaignUpdate}
	k.idle()
	return g, k
}

func TestCampaignListTabsSwitchSets(t *testing.T) {
	g, k := newCampaignUIGame(t, campaignSetGM1)
	k.uniqueIDs("gm1 list")
	for level := 1; level <= 10; level++ {
		k.item(campaignMissionID(level))
	}
	k.click("camp.set/GM2")
	if !g.campaignSetGM2 || g.campaignSetExtra || g.campaignPhase != 1 || g.campaignLevel != 0 {
		t.Fatalf("GM2 tab: gm2=%v extra=%v phase=%d level=%d", g.campaignSetGM2, g.campaignSetExtra, g.campaignPhase, g.campaignLevel)
	}
	for level := 1; level <= 16; level++ {
		k.item(campaignMissionID(level))
	}
	k.uniqueIDs("gm2 list")
	k.key(uiInput{NextTab: true})
	if !g.campaignSetExtra || g.campaignSetGM2 || g.extraCampaignPage != 0 {
		t.Fatalf("E: extra=%v gm2=%v page=%d", g.campaignSetExtra, g.campaignSetGM2, g.extraCampaignPage)
	}
	k.uniqueIDs("extra list")
	k.key(uiInput{PrevTab: true})
	k.key(uiInput{PrevTab: true})
	if g.campaignSetExtra || g.campaignSetGM2 {
		t.Fatal("Q twice did not return to GM1")
	}
}

func TestCampaignExtraPagesShowEveryMission(t *testing.T) {
	g, k := newCampaignUIGame(t, campaignSetExtra)
	seen := map[int]bool{}
	for page := 0; page < extraCampaignPageCount(); page++ {
		first, last := g.campaignPageRange(campaignSetExtra)
		for level := first; level <= last; level++ {
			k.item(campaignMissionID(level))
			seen[level] = true
		}
		if page+1 < extraCampaignPageCount() {
			r := k.item("camp.page").rect
			k.run(uiInput{X: r.X + r.W - 5, Y: r.Y + r.H/2, Moved: true})
			k.run(uiInput{X: r.X + r.W - 5, Y: r.Y + r.H/2, Pressed: true, Down: true})
			k.run(uiInput{X: r.X + r.W - 5, Y: r.Y + r.H/2, Released: true})
			k.idle()
			if g.extraCampaignPage != page+1 {
				t.Fatalf("pager: page %d want %d", g.extraCampaignPage, page+1)
			}
		}
	}
	if len(seen) != extraCampaignMissionCount {
		t.Fatalf("EXTRA list shows %d of %d missions", len(seen), extraCampaignMissionCount)
	}
	// The stage B missions (mech, gunship) describe their bosses.
	for _, level := range []int{26, 27, 28} {
		info := g.campaignMission(campaignSetExtra, level)
		if len(info.opponents) == 0 || info.title == "" {
			t.Fatalf("mission %d has no briefing: %+v", level, info)
		}
	}
	if !g.campaignMission(campaignSetExtra, 28).boss || !g.campaignMission(campaignSetExtra, 26).boss {
		t.Fatal("mech / gunship missions are not marked BOSS")
	}
}

func TestCampaignLockedMissionIsInert(t *testing.T) {
	g, k := newCampaignUIGame(t, campaignSetGM2)
	if g.gm2CampaignLevels[9] != 0 {
		t.Fatalf("fresh save: GM2 mission 10 state %d", g.gm2CampaignLevels[9])
	}
	it := k.item(campaignMissionID(10))
	if !it.disabled || it.focusable && !it.disabled {
		t.Fatalf("locked mission is interactive: %+v", it)
	}
	before := *g
	k.click(campaignMissionID(10))
	if g.campaignPhase != 1 || g.campaignLevel != before.campaignLevel || g.gm2CampaignLevels != before.gm2CampaignLevels {
		t.Fatalf("locked mission changed state: phase=%d level=%d", g.campaignPhase, g.campaignLevel)
	}
}

func TestCampaignOpenBriefingStartAndBack(t *testing.T) {
	for _, set := range []int{campaignSetGM1, campaignSetGM2, campaignSetExtra} {
		g, k := newCampaignUIGame(t, set)
		level := g.campaignNextMission(set)
		if set != campaignSetExtra {
			level = 2
		}
		k.click(campaignMissionID(level))
		if !g.campaignBriefingOpen() || g.campaignLevel != level {
			t.Fatalf("set %d: mission %d not opened: phase=%d level=%d", set, level, g.campaignPhase, g.campaignLevel)
		}
		k.uniqueIDs(fmt.Sprintf("set %d briefing", set))
		for _, id := range []string{"camp.coop", "camp.p0.edit", "camp.p2.add", "camp.brief.back", "camp.start"} {
			k.item(id)
		}
		if k.has(campaignMissionID(1)) || k.has("camp.set/GM1") {
			t.Fatal("list controls still live under the briefing")
		}
		k.key(uiInput{Esc: true})
		if g.campaignPhase != 1 || g.fadeActive {
			t.Fatalf("set %d: ESC on briefing: phase=%d fade=%v", set, g.campaignPhase, g.fadeActive)
		}
		k.click(campaignMissionID(level))
		k.click("camp.start")
		if !g.fadeActive || g.fadeTarget != screenGameplay || !g.campaignMode || g.campaignLevel != level || g.campaignSet() != set {
			t.Fatalf("set %d: START: fade=%v target=%d campaign=%v level=%d set=%d", set, g.fadeActive, g.fadeTarget, g.campaignMode, g.campaignLevel, g.campaignSet())
		}
	}
}

func TestCampaignListEscAndKeyboardOpen(t *testing.T) {
	g, k := newCampaignUIGame(t, campaignSetGM1)
	k.key(uiInput{DownKey: true})
	if g.campaignUI.focus == "" {
		t.Fatal("arrow did not focus a control")
	}
	g.campaignUI.focus = campaignMissionID(2)
	k.key(uiInput{Enter: true})
	if !g.campaignBriefingOpen() || g.campaignLevel != 2 {
		t.Fatalf("ENTER on mission 2: phase=%d level=%d", g.campaignPhase, g.campaignLevel)
	}
	k.keyboardReaches("camp.coop", "camp.p0.edit", "camp.p2.add", "camp.start", "camp.brief.back")
	k.key(uiInput{Esc: true})
	k.key(uiInput{Esc: true})
	if !g.fadeActive || g.fadeTarget != screenMainMenu {
		t.Fatalf("ESC on the list: fade=%v target=%d", g.fadeActive, g.fadeTarget)
	}
}

func TestCampaignCoopUsesTheCardCodes(t *testing.T) {
	g, k := newCampaignUIGame(t, campaignSetGM1)
	k.click(campaignMissionID(1))
	if g.campaignPlayers[1].Type != 0 {
		t.Fatal("fresh campaign P2 is not empty")
	}
	k.click("camp.p2.add")
	if g.campaignPlayers[1].Type != 1 {
		t.Fatal("ADD PLAYER 2 did not enable P2")
	}
	k.item("camp.p1.edit")
	k.click("camp.p2.remove")
	if g.campaignPlayers[1].Type != 0 {
		t.Fatal("REMOVE PLAYER 2 kept P2")
	}
	// The segmented SOLO / CO-OP control does the same.
	r := k.item("camp.coop").rect
	k.run(uiInput{X: r.X + r.W*0.75, Y: r.Y + r.H/2, Moved: true})
	k.run(uiInput{X: r.X + r.W*0.75, Y: r.Y + r.H/2, Pressed: true, Down: true})
	k.run(uiInput{X: r.X + r.W*0.75, Y: r.Y + r.H/2, Released: true})
	k.idle()
	if g.campaignPlayers[1].Type != 1 {
		t.Fatal("CO-OP segment did not enable P2")
	}
}

func TestCampaignSoloMissionKeepsP2Out(t *testing.T) {
	g, k := newCampaignUIGame(t, campaignSetGM2)
	g.campaignPlayers[1].Type = 1
	k.click(campaignMissionID(1)) // GM2 tutorial is solo
	if g.campaignPlayers[1].Type != 0 {
		t.Fatal("solo mission kept P2")
	}
	if !k.item("camp.coop").disabled || !k.item("camp.p2.add").disabled {
		t.Fatal("co-op controls are live on a solo mission")
	}
	k.click("camp.p2.add")
	k.key(uiInput{Right: true})
	if g.campaignPlayers[1].Type != 0 {
		t.Fatal("disabled co-op control enabled P2")
	}
}

func TestCampaignLoadoutWritesCampaignPlayers(t *testing.T) {
	g, k := newCampaignUIGame(t, campaignSetExtra)
	k.click(campaignMissionID(1))
	k.click("camp.p2.add")
	customBefore := g.customPlayers
	k.click("camp.p1.edit")
	if g.campaignLoadoutSlot != 2 {
		t.Fatalf("EDIT LOADOUT P2 opened slot %d", g.campaignLoadoutSlot)
	}
	k.uniqueIDs("campaign loadout")
	k.click("lo.tabs/WEAPON")
	guns := loadoutNumbers(customEditGun)
	want := guns[len(guns)-1]
	k.click(fmt.Sprintf("lo.gun.%d", want))
	if g.campaignPlayers[1].Gun != want {
		t.Fatalf("gun tile wrote %d want %d", g.campaignPlayers[1].Gun, want)
	}
	k.click("lo.tabs/PERK")
	k.click("lo.perk.3")
	if migrateLegacyGearPerk(g.campaignPlayers[1]).Perk != 3 {
		t.Fatalf("perk %d", g.campaignPlayers[1].Perk)
	}
	k.click("lo.tabs/THROWABLE")
	k.click("lo.throw.2")
	if g.campaignPlayers[1].Throwable != 2 {
		t.Fatalf("throwable %d", g.campaignPlayers[1].Throwable)
	}
	k.click("lo.tabs/CHARACTER")
	k.click("lo.name")
	if g.campaignNameFocus != 2 {
		t.Fatalf("name focus %d", g.campaignNameFocus)
	}
	if g.customPlayers != customBefore {
		t.Fatal("campaign loadout changed the Custom Game players")
	}
	// The player stepper moves to P1, DONE / ESC returns to the briefing.
	r := k.item("lo.player").rect
	k.run(uiInput{X: r.X + 5, Y: r.Y + r.H/2, Moved: true})
	k.run(uiInput{X: r.X + 5, Y: r.Y + r.H/2, Pressed: true, Down: true})
	k.run(uiInput{X: r.X + 5, Y: r.Y + r.H/2, Released: true})
	k.idle()
	if g.campaignLoadoutSlot != 1 || g.campaignNameFocus != 0 {
		t.Fatalf("stepper: slot %d focus %d", g.campaignLoadoutSlot, g.campaignNameFocus)
	}
	k.key(uiInput{Esc: true})
	if g.campaignLoadoutSlot != 0 || !g.campaignBriefingOpen() {
		t.Fatalf("ESC: slot=%d phase=%d", g.campaignLoadoutSlot, g.campaignPhase)
	}
	if g.campaignUI.focus != "camp.p0.edit" {
		t.Fatalf("focus after the loadout = %q", g.campaignUI.focus)
	}
}

// EXTRA offers every real perk; the GM1 / GM2 cards use the shared rule.
func TestCampaignLoadoutUsesCampaignUnlockRule(t *testing.T) {
	g := New()
	g.campaignSetExtra = true
	for _, c := range g.loadoutPerkChoicesFor(g.campaignLoadoutBinding(0)) {
		var n int
		fmt.Sscanf(c.id, "lo.perk.%d", &n)
		if c.locked != !g.campaignSelectionUnlocked(customEditPerk, n) {
			t.Fatalf("perk %d locked=%v", n, c.locked)
		}
		slot, _ := campaignSelectHitSlot(c.code)
		if slot != 0 {
			t.Fatalf("perk code %d decodes to slot %d", c.code, slot)
		}
	}
}

func campaignSelectHitSlot(code int) (int, bool) {
	v := code - campaignPlayerHitSelectBase
	if v < 0 || v >= 2*campaignSelectSlotStride {
		return -1, false
	}
	return v / campaignSelectSlotStride, true
}

func TestCampaignResultModal(t *testing.T) {
	g, k := newCampaignUIGame(t, campaignSetGM1)
	g.campaignLevel = 3
	p := NewPlayer(1, g.arena)
	p.Kills, p.Deaths = 4, 1
	g.players = []*Player{p}
	g.campaignMode = true
	g.GameWin = true
	g.finishSourceGameBeforeScreenChange()
	if !g.campaignResult.open || !g.campaignResult.won || g.campaignResult.unlock[0] == 0 {
		t.Fatalf("win not recorded: %+v", g.campaignResult)
	}
	if g.campaignLevels[2] != 2 {
		t.Fatal("level 3 not completed by the existing rule")
	}
	k.idle()
	k.uniqueIDs("result")
	if k.has(campaignMissionID(1)) {
		mission := k.item(campaignMissionID(1))
		if mission.layer == k.item("camp.res.ok").layer {
			t.Fatal("mission tiles are live under the result modal")
		}
	}
	k.key(uiInput{Esc: true})
	if g.campaignResultVisible() || g.campaignShowUnlock != 0 {
		t.Fatal("ESC did not close the result")
	}

	// Leaving through the pause menu (no winner) shows no result.
	g.campaignMode = true
	g.GameWin, g.teamGameWin = false, false
	g.players = []*Player{p, func() *Player { e := NewPlayer(4, g.arena); e.AI = true; return e }()}
	g.finishSourceGameBeforeScreenChange()
	if g.campaignResult.open {
		t.Fatal("quitting a mission opened the result modal")
	}

	// A loss offers RETRY, which starts the same mission.
	g.campaignMode = true
	g.campaignSetGM2 = true
	g.campaignLevel = 2
	g.teamGameWin = true
	g.finishSourceGameBeforeScreenChange()
	if !g.campaignResult.open || g.campaignResult.won {
		t.Fatalf("loss not recorded: %+v", g.campaignResult)
	}
	g.campaignSetGM2 = false // RETRY restores the result's set
	k.idle()
	k.click("camp.res.retry")
	if !g.fadeActive || g.fadeTarget != screenGameplay || !g.campaignSetGM2 || g.campaignLevel != 2 {
		t.Fatalf("RETRY: fade=%v target=%d gm2=%v level=%d", g.fadeActive, g.fadeTarget, g.campaignSetGM2, g.campaignLevel)
	}
}

func TestCampaignScreensDraw(t *testing.T) {
	g, k := newCampaignUIGame(t, campaignSetExtra)
	img := newTestImage()
	for _, set := range []int{campaignSetGM1, campaignSetGM2, campaignSetExtra} {
		g.selectCampaignSet(set)
		k.idle()
		g.drawCampaignMenu(img)
		k.click(campaignMissionID(1))
		g.drawCampaignMenu(img)
		k.click("camp.p0.edit")
		g.drawCampaignMenu(img)
		k.key(uiInput{Esc: true})
		k.key(uiInput{Esc: true})
	}
	g.recordCampaignResult(true)
	g.drawCampaignMenu(img)
}
