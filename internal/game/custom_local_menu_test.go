package game

import (
	"fmt"
	"testing"
)

// Local Custom Game (VERSUS) on the Mayhem UI kit: every test drives the real
// build function through runCustomLocalUpdate, so a control is only reachable
// where it is drawn.

func newCustomLocalGame(t *testing.T) *Game {
	t.Helper()
	g := New()
	g.screen = screenCustomGame
	g.customOnlineMode = false
	g.customNetworkOpen = false
	g.setCustomV2Page(customPageGame)
	g.runCustomLocalUpdate(uiInput{})
	return g
}

func customLocalClick(g *Game, x, y float64) {
	g.runCustomLocalUpdate(uiInput{X: x, Y: y, Moved: true})
	g.runCustomLocalUpdate(uiInput{X: x, Y: y, Pressed: true, Down: true})
	g.runCustomLocalUpdate(uiInput{X: x, Y: y, Released: true})
	// One idle frame so the registry reflects what the click opened.
	g.runCustomLocalUpdate(uiInput{X: x, Y: y})
}

func customLocalItem(t *testing.T, g *Game, id string) Rect {
	t.Helper()
	it, ok := g.customUI.item(id)
	if !ok {
		t.Fatalf("kit item %q not on screen (page %d)", id, g.customPage)
	}
	return it.rect
}

func customLocalClickItem(t *testing.T, g *Game, id string) {
	t.Helper()
	r := customLocalItem(t, g, id)
	customLocalClick(g, r.X+r.W/2, r.Y+r.H/2)
}

func customLocalKey(g *Game, in uiInput) {
	g.runCustomLocalUpdate(in)
	g.runCustomLocalUpdate(uiInput{})
}

func TestCustomLocalModeButtonsSelectEveryMode(t *testing.T) {
	g := newCustomLocalGame(t)
	for i, row := range customV2ModeRows {
		r := customLocalModeRect(i)
		if got := customLocalItem(t, g, fmt.Sprintf("cust.mode.%d", row.mode)); got != r {
			t.Fatalf("mode %d drawn at %+v, layout %+v", row.mode, got, r)
		}
		customLocalClick(g, r.X+10, r.Y+r.H-4)
		if g.customMode != row.mode {
			t.Fatalf("mode button %q selected %d want %d", row.label, g.customMode, row.mode)
		}
	}
}

func TestCustomLocalMatchSettingsControls(t *testing.T) {
	g := newCustomLocalGame(t)
	g.customMode = SourceGameModeNormal
	g.customLives = 10
	g.customSeriesRounds = 3
	g.runCustomLocalUpdate(uiInput{})

	lives := customLocalItem(t, g, "cust.lives")
	customLocalClick(g, lives.X+lives.W-10, lives.Y+lives.H/2)
	customLocalClick(g, lives.X+lives.W-10, lives.Y+lives.H/2)
	customLocalClick(g, lives.X+10, lives.Y+lives.H/2)
	if g.customLives != 11 {
		t.Fatalf("lives stepper: lives=%d want 11", g.customLives)
	}
	customLocalClick(g, lives.X+lives.W/2, lives.Y+lives.H/2)
	if !g.customLivesFocus {
		t.Fatal("clicking the lives number did not start typing")
	}
	customLocalClick(g, 600, 120) // empty panel area
	if g.customLivesFocus {
		t.Fatal("clicking elsewhere did not end lives typing")
	}

	rounds := customLocalItem(t, g, "cust.rounds")
	customLocalClick(g, rounds.X+rounds.W-10, rounds.Y+rounds.H/2)
	if g.customSeriesRounds != 4 {
		t.Fatalf("rounds=%d want 4", g.customSeriesRounds)
	}
	customLocalClick(g, rounds.X+10, rounds.Y+rounds.H/2)
	customLocalClick(g, rounds.X+10, rounds.Y+rounds.H/2)
	if g.customSeriesRounds != 2 {
		t.Fatalf("rounds=%d want 2", g.customSeriesRounds)
	}

	customLocalClickItem(t, g, "cust.crates")
	customLocalClickItem(t, g, "cust.pickups")
	if g.CrateON || g.PowerON {
		t.Fatalf("toggles did not switch off: crates=%v pickups=%v", g.CrateON, g.PowerON)
	}
	customLocalClickItem(t, g, "cust.crates")
	if !g.CrateON {
		t.Fatal("crates toggle did not switch back on")
	}

	// Gun Game has no lives / crate / pickup options: those controls vanish.
	g.customMode = SourceGameModeGunGame
	g.runCustomLocalUpdate(uiInput{})
	for _, id := range []string{"cust.lives", "cust.crates", "cust.pickups"} {
		if _, ok := g.customUI.item(id); ok {
			t.Fatalf("%s is still interactive in Gun Game", id)
		}
	}
}

func TestCustomLocalWizardButtonsAndTabs(t *testing.T) {
	g := newCustomLocalGame(t)
	customLocalClickItem(t, g, "cust.next")
	if g.customPage != customPageMaps {
		t.Fatalf("NEXT on MATCH -> page %d want maps", g.customPage)
	}
	customLocalClickItem(t, g, "cust.next")
	if g.customPage != customPagePlayers {
		t.Fatalf("NEXT on MAPS -> page %d want players", g.customPage)
	}
	customLocalClickItem(t, g, "cust.back")
	if g.customPage != customPageMaps {
		t.Fatalf("BACK on PLAYERS -> page %d want maps", g.customPage)
	}
	customLocalKey(g, uiInput{NextTab: true})
	if g.customPage != customPagePlayers {
		t.Fatalf("E -> page %d want players", g.customPage)
	}
	customLocalClickItem(t, g, "cust.tabs/MATCH")
	if g.customPage != customPageGame {
		t.Fatalf("MATCH tab -> page %d", g.customPage)
	}
	customLocalKey(g, uiInput{Esc: true})
	if !g.fadeActive || g.fadeTarget != screenMainMenu {
		t.Fatal("ESC on MATCH did not return to the main menu")
	}

	// The lives warning still blocks NEXT exactly like the source warning1.
	g = newCustomLocalGame(t)
	g.customLives = 0
	customLocalClickItem(t, g, "cust.next")
	if g.customPage != customPageGame || g.customWarning != 1 {
		t.Fatalf("NEXT with 0 lives: page=%d warning=%d", g.customPage, g.customWarning)
	}
	if len(g.customUI.toasts) == 0 {
		t.Fatal("lives warning shows no toast")
	}
}

func TestCustomLocalKeyboardNavigation(t *testing.T) {
	g := newCustomLocalGame(t)
	customLocalKey(g, uiInput{Tab: true})
	if g.customUI.focus != "cust.tabs/MATCH" {
		t.Fatalf("first focus=%q want the MATCH tab", g.customUI.focus)
	}
	first := fmt.Sprintf("cust.mode.%d", customV2ModeRows[0].mode)
	for i := 0; i < 10 && g.customUI.focus != first; i++ {
		customLocalKey(g, uiInput{Tab: true})
	}
	if g.customUI.focus != first {
		t.Fatalf("TAB never reached the mode list: %q", g.customUI.focus)
	}
	customLocalKey(g, uiInput{DownKey: true})
	customLocalKey(g, uiInput{DownKey: true})
	customLocalKey(g, uiInput{Enter: true})
	if g.customMode != customV2ModeRows[2].mode {
		t.Fatalf("keyboard selected mode %d want %d", g.customMode, customV2ModeRows[2].mode)
	}
	// RIGHT from the mode list reaches the settings column.
	customLocalKey(g, uiInput{Right: true})
	if f := g.customUI.focus; f == "" || f[:9] == "cust.mode" {
		t.Fatalf("RIGHT from the mode list stayed at %q", f)
	}
}

func TestCustomLocalMapTilesSelectEveryMap(t *testing.T) {
	g := newCustomLocalGame(t)
	g.customSeriesRounds = 2
	g.setCustomV2Page(customPageMaps)
	g.runCustomLocalUpdate(uiInput{})
	choices := customV2MapChoices(false)
	for i, choice := range choices {
		r := customLocalItem(t, g, fmt.Sprintf("cust.map.%d", i))
		if r != customLocalMapTileRect(i) {
			t.Fatalf("map tile %d drawn at %+v", i, r)
		}
		// The whole tile is the hit area, including its label strip.
		customLocalClick(g, r.X+4, r.Y+r.H-3)
		if got := g.customSeriesMapForRound(1); got != choice.value {
			t.Fatalf("tile %d (%s) selected %d want %d", i, choice.label, got, choice.value)
		}
	}
	if r := customLocalMapTileRect(len(choices) - 1); r.Y+r.H > uiContentBottom {
		t.Fatalf("last map tile leaves the content area: %+v", r)
	}

	// Round 2 keeps its own map.
	list := customLocalItem(t, g, "cust.rounds.list")
	customLocalClick(g, list.X+20, list.Y+32+10)
	if g.customPlaylistSelectedRound != 2 {
		t.Fatalf("round list picked %d want 2", g.customPlaylistSelectedRound)
	}
	customLocalClickItem(t, g, "cust.map.12")
	if g.customSeriesMapForRound(2) != 12 || g.customSeriesMapForRound(1) != choices[len(choices)-1].value {
		t.Fatalf("rounds share a map: r1=%d r2=%d", g.customSeriesMapForRound(1), g.customSeriesMapForRound(2))
	}
	customLocalClickItem(t, g, "cust.applyall")
	if g.customSeriesMapForRound(1) != 12 {
		t.Fatalf("APPLY TO ALL: r1=%d", g.customSeriesMapForRound(1))
	}
}

func TestCustomLocalGM2MapSet(t *testing.T) {
	if !gm2MapAssetsAvailable() {
		t.Skip("local unpacked GM2 map assets not available")
	}
	g := newCustomLocalGame(t)
	g.setCustomV2Page(customPageMaps)
	g.runCustomLocalUpdate(uiInput{})
	set := customLocalItem(t, g, "cust.mapset")
	customLocalClick(g, set.X+set.W*0.75, set.Y+set.H/2)
	if !g.customPlaylistGM2 {
		t.Fatal("GM2 segment did not switch the map set")
	}
	customLocalClickItem(t, g, "cust.map.0")
	if got := g.customSeriesMapForRound(1); got != customGM2RandomNew {
		t.Fatalf("GM2 random tile -> %d", got)
	}
	customLocalClickItem(t, g, "cust.map.1")
	if got := g.customSeriesMapForRound(1); got != gm2MapID(1) {
		t.Fatalf("first GM2 map tile -> %d want %d", got, gm2MapID(1))
	}
	customLocalClick(g, set.X+set.W*0.25, set.Y+set.H/2)
	if g.customPlaylistGM2 {
		t.Fatal("GM1 segment did not switch back")
	}
}

func TestCustomLocalModifiersAndChaosModals(t *testing.T) {
	g := newCustomLocalGame(t)
	customLocalClickItem(t, g, "cust.modifiers")
	if !g.customRulesOpen {
		t.Fatal("CONFIGURE did not open the modifiers")
	}
	// The modal blocks the page under it.
	mode := customLocalModeRect(3)
	customLocalClick(g, mode.X+10, mode.Y+10)
	if g.customMode != SourceGameModeNormal {
		t.Fatalf("click went through the modal: mode=%d", g.customMode)
	}

	customLocalClickItem(t, g, fmt.Sprintf("cust.mod.%d", customModifierShotgunsOnly))
	customLocalClickItem(t, g, fmt.Sprintf("cust.mod.%d", customModifierPistolsOnly))
	if g.customModifierEnabledForEdit(customModifierShotgunsOnly) || !g.customModifierEnabledForEdit(customModifierPistolsOnly) {
		t.Fatal("conflicting weapon modifiers were not resolved")
	}
	customLocalClickItem(t, g, "cust.mod.random")
	if !g.customRandomRulesForEdit() {
		t.Fatal("RANDOM RULES did not switch on")
	}
	count := customLocalItem(t, g, "cust.mod.count")
	customLocalClick(g, count.X+count.W*5/6, count.Y+count.H/2)
	if g.customRandomCountForEdit() != 3 {
		t.Fatalf("random count=%d want 3", g.customRandomCountForEdit())
	}
	scope := customLocalItem(t, g, "cust.mod.scope")
	customLocalClick(g, scope.X+scope.W*3/4, scope.Y+scope.H/2)
	if !g.customPerRoundRules {
		t.Fatal("EACH ROUND / MAP did not enable per-round rules")
	}
	round := customLocalItem(t, g, "cust.mod.round")
	customLocalClick(g, round.X+round.W-8, round.Y+round.H/2)
	if g.customModifierEditRound() != 2 {
		t.Fatalf("editing round=%d want 2", g.customModifierEditRound())
	}

	customLocalClickItem(t, g, "cust.mod.chaos")
	if !g.customChaosOpen {
		t.Fatal("CHAOS SETUP did not open")
	}
	on := customLocalItem(t, g, "cust.chaos.on")
	customLocalClick(g, on.X+on.W*3/4, on.Y+on.H/2)
	if !g.customChaosEnabled {
		t.Fatal("chaos ON segment")
	}
	customLocalClickItem(t, g, "cust.chaos.clear")
	spec := chaosEffectSpecs[0]
	customLocalClickItem(t, g, fmt.Sprintf("cust.chaos.%d", spec.ID))
	if g.chaosSelectedCount() != 1 || !g.customChaosSelected[spec.ID] {
		t.Fatalf("chaos pool=%d", g.chaosSelectedCount())
	}
	every := customLocalItem(t, g, "cust.chaos.every")
	before := g.customChaosInterval
	customLocalClick(g, every.X+every.W-8, every.Y+every.H/2)
	if g.customChaosInterval != before+5 {
		t.Fatalf("interval %d -> %d", before, g.customChaosInterval)
	}
	customLocalKey(g, uiInput{Esc: true})
	if g.customChaosOpen || !g.customRulesOpen {
		t.Fatal("ESC in chaos must return to the modifiers")
	}
	customLocalKey(g, uiInput{Esc: true})
	if g.customRulesOpen {
		t.Fatal("ESC did not close the modifiers")
	}
	if g.fadeActive {
		t.Fatal("closing the modal also left the screen")
	}

	// CHAOS row on MATCH opens chaos directly.
	customLocalClickItem(t, g, "cust.chaos")
	if !g.customRulesOpen || !g.customChaosOpen {
		t.Fatal("chaos SETUP row did not open the chaos modal")
	}
	customLocalClickItem(t, g, "cust.chaos.done")
	if g.customRulesOpen || g.customChaosOpen {
		t.Fatal("DONE did not close everything")
	}
}

func TestCustomLocalPlayerSlotTypes(t *testing.T) {
	g := newCustomLocalGame(t)
	g.setCustomV2Page(customPagePlayers)
	g.runCustomLocalUpdate(uiInput{})

	seg := customLocalItem(t, g, "cust.p2.type")
	customLocalClick(g, seg.X+seg.W/2, seg.Y+seg.H/2) // AI
	if g.customPlayers[2].Type != 2 || g.customPlayers[2].Ability != abilityRandom {
		t.Fatalf("P3 AI: type=%d ability=%d", g.customPlayers[2].Type, g.customPlayers[2].Ability)
	}
	customLocalClick(g, seg.X+seg.W-5, seg.Y+seg.H/2) // OFF
	if g.customPlayers[2].Type != 0 {
		t.Fatalf("P3 OFF: type=%d", g.customPlayers[2].Type)
	}
	customLocalClickItem(t, g, "cust.p3.addhuman")
	if g.customPlayers[3].Type != 1 {
		t.Fatalf("P4 ADD HUMAN: type=%d", g.customPlayers[3].Type)
	}

	name := customLocalItem(t, g, "cust.p0.name")
	customLocalClick(g, name.X+10, name.Y+10)
	if g.customNameFocus != 1 {
		t.Fatalf("name field focus=%d want 1", g.customNameFocus)
	}
	customLocalClick(g, 450, 60)
	if g.customNameFocus != 0 {
		t.Fatal("clicking elsewhere did not end name typing")
	}

	g.customMode = SourceGameModeTeams
	g.runCustomLocalUpdate(uiInput{})
	team := customLocalItem(t, g, "cust.p0.team")
	customLocalClick(g, team.X+team.W*3/4, team.Y+team.H/2)
	if g.customPlayers[0].Team != 2 {
		t.Fatalf("P1 team=%d want 2", g.customPlayers[0].Team)
	}
}

// The PLAYERS card's EDIT LOADOUT still opens the classic card editor (phase 4
// replaces it). It must sit in the same column, take the mouse only there, and
// only ever produce codes for its own slot.
func TestCustomLocalCardEditTransition(t *testing.T) {
	g := newCustomLocalGame(t)
	g.setCustomV2Page(customPagePlayers)
	g.runCustomLocalUpdate(uiInput{})
	customLocalClickItem(t, g, "cust.p1.edit")
	if g.customCardEditSlot != 2 {
		t.Fatalf("EDIT LOADOUT P2 -> edit slot %d", g.customCardEditSlot)
	}
	if _, ok := g.customUI.item("cust.p1.edit"); ok {
		t.Fatal("kit card is still interactive under the classic card")
	}
	if _, ok := g.customUI.item("cust.p0.edit"); !ok {
		t.Fatal("other players' cards must stay usable")
	}

	cardX := 20.0 + 220
	gunButton := sourceMenuHitRect(customPlayerTypeButtonLocal, cardX+17.5, g.customCardY[1]+161.7, 0.5387115478515625, 1)
	x, y := gunButton.X+gunButton.W/2, gunButton.Y+gunButton.H/2
	if hit := g.customCardEditHitAt(x, y); hit != customEditHit(1, customEditGun) {
		t.Fatalf("classic HANDGUN button hit=%d", hit)
	}
	// The same button of P1's card (not being edited) is not live.
	if hit := g.customCardEditHitAt(x-220, y); hit != menuHitNone {
		t.Fatalf("P1's classic card answered while P2 is edited: %d", hit)
	}
	customLocalClick(g, x, y)
	if g.customEditor[1] != customEditGun {
		t.Fatalf("classic HANDGUN did not open: editor=%d", g.customEditor[1])
	}
	if g.customPlayers[0].Gun != 1 || g.customMode != SourceGameModeNormal {
		t.Fatal("editor click changed something else")
	}

	// Pick a developed pistol in the slid-up selection grid.
	g.customCardY[1] = -300
	picked := 0
	for _, item := range developedCustomSelectionItems(customEditGun) {
		if item.Number < 98 {
			continue
		}
		r := item.Rect
		r.X += cardX
		r.Y += g.customCardY[1]
		if r.Y < uiContentTop || r.Y+r.H > uiContentBottom {
			continue
		}
		customLocalClick(g, r.X+r.W/2, r.Y+r.H/2)
		picked = item.Number
		break
	}
	if picked == 0 {
		t.Fatal("no developed pistol visible in the classic grid")
	}
	if g.customPlayers[1].Gun != picked || g.customEditor[1] != 0 {
		t.Fatalf("pistol pick: gun=%d want %d editor=%d", g.customPlayers[1].Gun, picked, g.customEditor[1])
	}

	g.customEditor[1] = customEditPerk
	customLocalKey(g, uiInput{Esc: true})
	if g.customEditor[1] != 0 || g.customCardEditSlot != 2 {
		t.Fatalf("ESC must first close the sub-editor: editor=%d slot=%d", g.customEditor[1], g.customCardEditSlot)
	}
	customLocalKey(g, uiInput{Esc: true})
	if g.customCardEditSlot != 0 || g.fadeActive || g.customPage != customPagePlayers {
		t.Fatalf("second ESC must only close the editor: slot=%d page=%d", g.customCardEditSlot, g.customPage)
	}

	customLocalClickItem(t, g, "cust.p1.edit")
	customLocalClickItem(t, g, "cust.editdone")
	if g.customCardEditSlot != 0 {
		t.Fatal("DONE did not close the classic editor")
	}
}

func TestCustomLocalCardEditCodesStayInTheirSlot(t *testing.T) {
	for slot := 0; slot < 4; slot++ {
		g := newCustomLocalGame(t)
		g.setCustomV2Page(customPagePlayers)
		for i := range g.customPlayers {
			g.customPlayers[i].Type = 1
			g.customCardY[i] = 100
		}
		g.openCustomCardEdit(slot)
		seen := 0
		// Sample the edited column only, on an 8 px grid.
		col := customLocalCardRect(slot)
		for _, editor := range []int{0, customEditGun, customEditPerk, customEditAbility, customEditThrowable} {
			g.customEditor[slot] = editor
			g.customCardY[slot] = 100
			if editor != 0 {
				g.customCardY[slot] = -300
			}
			for y := col.Y; y < col.Y+col.H; y += 8 {
				for x := col.X - 16; x < col.X+col.W+16; x += 8 {
					hit := g.customCardEditHitAt(x, y)
					if hit == menuHitNone {
						continue
					}
					seen++
					s, ok := customPlayerEditorHitSlot(hit)
					if !ok {
						s, _, ok = decodePlayerSlotHit(hit)
					}
					if !ok || s != slot {
						t.Fatalf("slot %d editor %d: code %d at (%.0f,%.0f) belongs to slot %d (ok=%v)", slot, editor, hit, x, y, s, ok)
					}
					if g.activateCustomModeOption(hit) {
						t.Fatalf("slot %d: code %d is consumed by the match settings", slot, hit)
					}
				}
			}
		}
		if seen == 0 {
			t.Fatalf("slot %d: classic card produced no hits", slot)
		}
	}
}

func TestCustomLocalKitIDsAreUnique(t *testing.T) {
	g := newCustomLocalGame(t)
	g.customPlayers[2].Type = 2
	check := func(name string) {
		t.Helper()
		g.runCustomLocalUpdate(uiInput{})
		seen := map[string]bool{}
		for _, it := range g.customUI.items {
			if seen[it.id] {
				t.Fatalf("%s: duplicate kit id %q", name, it.id)
			}
			seen[it.id] = true
		}
	}
	for _, page := range customLocalTabPages {
		g.setCustomV2Page(page)
		check(fmt.Sprintf("page %d", page))
	}
	g.customPlaylistGM2 = true
	g.setCustomV2Page(customPageMaps)
	check("maps gm2")
	g.setCustomV2Page(customPagePlayers)
	g.openCustomCardEdit(0)
	check("players edit")
	g.closeCustomCardEdit()
	g.setCustomV2Page(customPageGame)
	g.customRulesOpen = true
	check("modifiers")
	g.customChaosOpen = true
	check("chaos")
}

func TestCustomLocalStartDefaultMatch(t *testing.T) {
	g := newCustomLocalGame(t)
	customLocalClickItem(t, g, "cust.next")
	customLocalClickItem(t, g, "cust.next")
	customLocalClickItem(t, g, "cust.next") // START MATCH
	if !g.fadeActive || g.fadeTarget != screenGameplay {
		t.Fatal("START MATCH with defaults did not start the match")
	}
	if len(g.players) != 2 || g.players[0].ID != 1 || g.players[1].ID != 2 || g.players[0].AI || g.players[1].AI {
		t.Fatalf("default match players: %d", len(g.players))
	}
	if g.GameMode != SourceGameModeNormal || g.TotalLives != 10 {
		t.Fatalf("default rules: mode=%d lives=%d", g.GameMode, g.TotalLives)
	}
}

func TestCustomLocalStartFourPlayersWithAI(t *testing.T) {
	g := newCustomLocalGame(t)
	g.setCustomV2Page(customPagePlayers)
	g.runCustomLocalUpdate(uiInput{})
	customLocalClickItem(t, g, "cust.p2.addai")
	customLocalClickItem(t, g, "cust.p3.addai")
	seg := customLocalItem(t, g, "cust.p1.type")
	customLocalClick(g, seg.X+seg.W/2, seg.Y+seg.H/2) // P2 -> AI
	customLocalClickItem(t, g, "cust.next")
	if !g.fadeActive || g.fadeTarget != screenGameplay {
		t.Fatal("START MATCH with 4 players did not start")
	}
	if len(g.players) != 4 {
		t.Fatalf("players=%d want 4", len(g.players))
	}
	wantAI := []bool{false, true, true, true}
	for i, p := range g.players {
		if p.ID != i+1 || p.AI != wantAI[i] {
			t.Fatalf("player %d: id=%d ai=%v want ai=%v", i, p.ID, p.AI, wantAI[i])
		}
	}
}

func TestCustomLocalStartNeedsTwoPlayers(t *testing.T) {
	g := newCustomLocalGame(t)
	g.setCustomV2Page(customPagePlayers)
	g.runCustomLocalUpdate(uiInput{})
	seg := customLocalItem(t, g, "cust.p1.type")
	customLocalClick(g, seg.X+seg.W-5, seg.Y+seg.H/2) // P2 OFF
	customLocalClickItem(t, g, "cust.next")
	if g.fadeActive {
		t.Fatal("started with one player")
	}
	if g.customWarning != 2 || len(g.customUI.toasts) == 0 {
		t.Fatalf("missing player-count warning: kind=%d toasts=%d", g.customWarning, len(g.customUI.toasts))
	}
}

func TestCustomGameRoutesLocalAndOnline(t *testing.T) {
	g := newCustomLocalGame(t)
	g.customOnlineMode = true
	g.customNetworkOpen = true
	before := len(g.customUI.items)
	g.updateCustomGameMenu() // online browser: the kit screen must not run
	if len(g.customUI.items) != before {
		t.Fatal("online browser ran the local kit screen")
	}
}
