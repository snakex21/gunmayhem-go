package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Loadout screen (phase 4): driven through runCustomLocalUpdate like the rest
// of the local Custom Game, so a control is only reachable where it is drawn.

func newLoadoutGame(t *testing.T, slot int) *Game {
	t.Helper()
	g := newCustomLocalGame(t)
	g.setCustomV2Page(customPagePlayers)
	if g.customPlayers[slot].Type == 0 {
		g.customPlayers[slot].Type = 2
	}
	g.runCustomLocalUpdate(uiInput{})
	customLocalClickItem(t, g, fmt.Sprintf("cust.p%d.edit", slot))
	if g.customLoadoutSlot != slot+1 {
		t.Fatalf("EDIT LOADOUT P%d opened slot %d", slot+1, g.customLoadoutSlot)
	}
	return g
}

func loadoutSelectTab(t *testing.T, g *Game, tab int) {
	t.Helper()
	customLocalClickItem(t, g, "lo.tabs/"+loadoutTabLabels[tab])
	if g.customLoadoutTab != tab {
		t.Fatalf("tab %s not selected: %d", loadoutTabLabels[tab], g.customLoadoutTab)
	}
}

func TestLoadoutOpensFullScreenAndEscReturnsToPlayers(t *testing.T) {
	g := newLoadoutGame(t, 1)
	for _, id := range []string{"cust.tabs/PLAYERS", "cust.p0.edit", "cust.next", "cust.back"} {
		if _, ok := g.customUI.item(id); ok {
			t.Fatalf("%s is still interactive under the loadout screen", id)
		}
	}
	for _, id := range []string{"lo.done", "lo.player", "lo.tabs/CHARACTER", "lo.name", "lo.part"} {
		customLocalItem(t, g, id)
	}
	customLocalKey(g, uiInput{Esc: true})
	if g.customLoadoutSlot != 0 || g.customPage != customPagePlayers || g.fadeActive {
		t.Fatalf("ESC: slot=%d page=%d fade=%v", g.customLoadoutSlot, g.customPage, g.fadeActive)
	}
	if g.customUI.focus != "cust.p1.edit" {
		t.Fatalf("focus after ESC = %q, want back on the EDIT LOADOUT button", g.customUI.focus)
	}
	customLocalClickItem(t, g, "cust.p1.edit")
	customLocalClickItem(t, g, "lo.done")
	if g.customLoadoutSlot != 0 || g.customPage != customPagePlayers {
		t.Fatal("DONE did not return to PLAYERS")
	}
	// Leaving PLAYERS (or emptying the slot) also drops the loadout.
	g.openCustomLoadout(1)
	g.customPlayers[1].Type = 0
	g.runCustomLocalUpdate(uiInput{})
	if _, ok := g.customUI.item("lo.done"); ok {
		t.Fatal("loadout of an empty slot is still shown")
	}
}

func TestLoadoutTabsSwitchWithQE(t *testing.T) {
	g := newLoadoutGame(t, 0)
	for want := 1; want <= loadoutTabCount; want++ {
		customLocalKey(g, uiInput{NextTab: true})
		if g.customLoadoutTab != want%loadoutTabCount {
			t.Fatalf("E: tab=%d want %d", g.customLoadoutTab, want%loadoutTabCount)
		}
	}
	customLocalKey(g, uiInput{PrevTab: true})
	if g.customLoadoutTab != loadoutTabThrowable {
		t.Fatalf("Q wrap: tab=%d", g.customLoadoutTab)
	}
	if g.customPage != customPagePlayers || g.customLoadoutSlot != 1 {
		t.Fatal("Q/E in the loadout must not switch the Custom Game tabs")
	}
}

func TestLoadoutCharacterWritesConfig(t *testing.T) {
	g := newLoadoutGame(t, 1)
	p1 := g.customPlayers[0]
	loadoutSelectTab(t, g, loadoutTabCharacter)
	colours := loadoutNumbers(customEditColor)
	last := colours[len(colours)-1]
	customLocalClickItem(t, g, fmt.Sprintf("lo.colour.%d", last))
	if g.customPlayers[1].Color != last {
		t.Fatalf("colour=%d want %d", g.customPlayers[1].Color, last)
	}
	parts := loadoutPartLabels()
	for part, field := range []func() int{
		func() int { return g.customPlayers[1].Color },
		func() int { return g.customPlayers[1].Hat },
		func() int { return g.customPlayers[1].Shirt },
		func() int { return g.customPlayers[1].Eyes },
	} {
		if part >= len(parts) {
			break
		}
		seg := customLocalItem(t, g, "lo.part")
		w := seg.W / float64(len(parts))
		customLocalClick(g, seg.X+w*float64(part)+w/2, seg.Y+seg.H/2)
		if g.customLoadoutPart != part {
			t.Fatalf("part %s not selected", parts[part])
		}
		choices, _ := g.loadoutCharacterChoices(1, part)
		pick := choices[len(choices)-1]
		customLocalClickItem(t, g, pick.id)
		if got := fmt.Sprintf("%d", field()); !strings.HasSuffix(pick.id, "."+got) {
			t.Fatalf("part %s: clicked %s, config value %s", parts[part], pick.id, got)
		}
	}
	if g.customPlayers[0] != p1 {
		t.Fatal("editing P2 changed P1")
	}

	// NAME: click to type, typed Q/E (text focus) must not switch tabs.
	customLocalClickItem(t, g, "lo.name")
	if g.customNameFocus != 2 {
		t.Fatalf("name focus=%d want 2", g.customNameFocus)
	}
	// updateCustomLocalMenu hands a text-focused frame only pointer input.
	typed := uiInput{NextTab: true, Esc: true}
	if g.customLocalTextFocus() {
		typed = typed.pointerOnly()
	}
	g.runCustomLocalUpdate(typed)
	if g.customLoadoutTab != loadoutTabCharacter || g.customLoadoutSlot != 2 {
		t.Fatal("keys typed into the name reached the loadout screen")
	}
	customLocalClick(g, 450, 560)
	if g.customNameFocus != 0 {
		t.Fatal("clicking elsewhere did not end name typing")
	}
}

func TestLoadoutWeaponPerkGearThrowableWriteConfig(t *testing.T) {
	g := newLoadoutGame(t, 2)
	loadoutSelectTab(t, g, loadoutTabWeapon)
	guns := loadoutNumbers(customEditGun)
	if len(guns) < 7 {
		t.Fatalf("handgun roster too small: %v", guns)
	}
	for _, n := range []int{2, guns[len(guns)-1]} {
		customLocalClickItem(t, g, fmt.Sprintf("lo.gun.%d", n))
		if g.customPlayers[2].Gun != n {
			t.Fatalf("gun=%d want %d", g.customPlayers[2].Gun, n)
		}
	}

	loadoutSelectTab(t, g, loadoutTabPerk)
	for _, n := range selectablePerkNumbers() {
		customLocalClickItem(t, g, fmt.Sprintf("lo.perk.%d", n))
		if g.customPlayers[2].Perk != n {
			t.Fatalf("perk=%d want %d", g.customPlayers[2].Perk, n)
		}
	}

	loadoutSelectTab(t, g, loadoutTabGear)
	for _, a := range selectableAbilityNumbers() {
		customLocalClickItem(t, g, fmt.Sprintf("lo.ability.%d", a))
		if g.customPlayers[2].Ability != a {
			t.Fatalf("ability=%d want %d", g.customPlayers[2].Ability, a)
		}
	}

	loadoutSelectTab(t, g, loadoutTabThrowable)
	for kind := throwableTypeCount - 1; kind >= 0; kind-- {
		customLocalClickItem(t, g, fmt.Sprintf("lo.throw.%d", kind))
		if g.customPlayers[2].Throwable != kind {
			t.Fatalf("throwable=%d want %d", g.customPlayers[2].Throwable, kind)
		}
	}
	if g.customLoadoutSlot != 3 || g.customEditor != [4]int{} {
		t.Fatalf("picks must keep the loadout open and leave the classic editor closed: slot=%d editor=%v", g.customLoadoutSlot, g.customEditor)
	}
}

func TestLoadoutKeyboardPicksItems(t *testing.T) {
	g := newCustomLocalGame(t)
	g.setCustomV2Page(customPagePlayers)
	g.runCustomLocalUpdate(uiInput{})
	g.openCustomLoadout(0)
	g.customLoadoutTab = loadoutTabWeapon
	g.customLoadoutFocusPending = true
	g.runCustomLocalUpdate(uiInput{X: -10, Y: -10})
	g.runCustomLocalUpdate(uiInput{X: -10, Y: -10})
	if g.customUI.focus != "lo.gun.1" {
		t.Fatalf("opening focuses %q, want the selected gun", g.customUI.focus)
	}
	customLocalKey(g, uiInput{X: -10, Y: -10, Right: true})
	customLocalKey(g, uiInput{X: -10, Y: -10, Enter: true})
	if g.customPlayers[0].Gun != 2 {
		t.Fatalf("RIGHT + ENTER picked gun %d, want 2", g.customPlayers[0].Gun)
	}
	customLocalKey(g, uiInput{X: -10, Y: -10, DownKey: true})
	customLocalKey(g, uiInput{X: -10, Y: -10, Enter: true})
	guns := loadoutNumbers(customEditGun)
	if g.customPlayers[0].Gun != guns[7] {
		t.Fatalf("DOWN + ENTER picked gun %d, want %d", g.customPlayers[0].Gun, guns[7])
	}
	// E switches tab and moves the focus onto that tab's selected item.
	customLocalKey(g, uiInput{X: -10, Y: -10, NextTab: true})
	if g.customLoadoutTab != loadoutTabPerk || g.customUI.focus != fmt.Sprintf("lo.perk.%d", g.customPlayers[0].Perk) {
		t.Fatalf("E -> tab %d focus %q", g.customLoadoutTab, g.customUI.focus)
	}
	customLocalKey(g, uiInput{X: -10, Y: -10, Right: true})
	customLocalKey(g, uiInput{X: -10, Y: -10, Enter: true})
	if g.customPlayers[0].Perk != 8 {
		t.Fatalf("RIGHT + ENTER from EXTRA AMMO picked perk %d, want 8", g.customPlayers[0].Perk)
	}
	customLocalKey(g, uiInput{X: -10, Y: -10, Esc: true})
	if g.customLoadoutSlot != 0 {
		t.Fatal("ESC did not close the loadout")
	}
}

func TestLoadoutPlayerSwitcherSkipsEmptySlots(t *testing.T) {
	g := newLoadoutGame(t, 0)
	g.customPlayers[3].Type = 2
	g.runCustomLocalUpdate(uiInput{})
	r := customLocalItem(t, g, "lo.player")
	next := func() { customLocalClick(g, r.X+r.W-10, r.Y+r.H/2) }
	want := []int{2, 4, 1}
	for _, w := range want {
		next()
		if g.customLoadoutSlot != w {
			t.Fatalf("next player: slot %d want %d", g.customLoadoutSlot, w)
		}
	}
	customLocalClick(g, r.X+10, r.Y+r.H/2)
	if g.customLoadoutSlot != 4 {
		t.Fatalf("previous player: slot %d want 4", g.customLoadoutSlot)
	}
}

func TestLoadoutRespectsEditGuard(t *testing.T) {
	g := newLoadoutGame(t, 1)
	// An online client may only edit its own character; the guard must still
	// disable every pick (and the setters ignore forced codes).
	g.customOnlineMode = true
	before := g.customPlayers[1]
	for tab := 0; tab < loadoutTabCount; tab++ {
		g.customLoadoutTab = tab
		g.runCustomLocalUpdate(uiInput{})
		for _, it := range g.customUI.items {
			if strings.HasPrefix(it.id, "lo.") && !strings.HasPrefix(it.id, "lo.tabs") && it.id != "lo.done" && it.id != "lo.player" && it.id != "lo.part" && !it.disabled {
				t.Fatalf("tab %d: %s is enabled for a slot the player may not edit", tab, it.id)
			}
		}
	}
	g.activateCustomPlayerEditorHit(customSelectHit(1, customEditGun, 3))
	if g.customPlayers[1] != before {
		t.Fatal("guarded slot changed")
	}
}

func TestLoadoutKitIDsAreUniqueOnEveryTab(t *testing.T) {
	g := newLoadoutGame(t, 0)
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
		if len(seen) < 10 {
			t.Fatalf("%s: only %d controls", name, len(seen))
		}
	}
	for tab := 0; tab < loadoutTabCount; tab++ {
		g.customLoadoutTab = tab
		if tab == loadoutTabCharacter {
			for part := range loadoutPartLabels() {
				g.customLoadoutPart = part
				check(fmt.Sprintf("character part %d", part))
			}
			continue
		}
		check(loadoutTabLabels[tab])
	}
}

// Every tile code must decode to the edited slot and never be consumed by the
// match settings (same contract the classic card had).
func TestLoadoutCodesStayInTheirSlot(t *testing.T) {
	g := New()
	for slot := 0; slot < 4; slot++ {
		var all []loadoutChoice
		for part := range loadoutPartLabels() {
			c, _ := g.loadoutCharacterChoices(slot, part)
			all = append(all, c...)
		}
		all = append(all, g.loadoutWeaponChoices(slot)...)
		all = append(all, g.loadoutPerkChoices(slot)...)
		all = append(all, g.loadoutGearChoices(slot)...)
		all = append(all, g.loadoutThrowableChoices(slot)...)
		for _, c := range all {
			s, ok := customPlayerEditorHitSlot(c.code)
			if !ok || s != slot {
				t.Fatalf("slot %d: %s code %d decodes to slot %d (ok=%v)", slot, c.id, c.code, s, ok)
			}
			if g.activateCustomModeOption(c.code) {
				t.Fatalf("slot %d: %s code %d is consumed by the match settings", slot, c.id, c.code)
			}
		}
	}
}

func TestLoadoutPreviewAndInfoDraw(t *testing.T) {
	g := newLoadoutGame(t, 0)
	g.customPlayers[0].Ability = abilityGasMask
	screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
	for tab := 0; tab < loadoutTabCount; tab++ {
		g.customLoadoutTab = tab
		g.runCustomLocalUpdate(uiInput{})
		g.drawCustomLocalMenu(screen)
	}
	if line := weaponStatsLine(1); !strings.Contains(line, "rounds") {
		t.Fatalf("weapon stats line %q", line)
	}
	for _, n := range selectablePerkNumbers() {
		if loadoutPerkInfo[n] == "" {
			t.Fatalf("perk %d has no description", n)
		}
	}
	for kind := 0; kind < throwableTypeCount; kind++ {
		if throwableLoadoutDescription(kind) == "" {
			t.Fatalf("throwable %d has no description", kind)
		}
	}
}
