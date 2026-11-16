package game

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Local Custom Game ("VERSUS"), rebuilt with the Mayhem UI kit: tabs MATCH /
// MAPS / PLAYERS, the modifiers and chaos setup as modals. Every action goes
// through the existing activation API (activateCustomV2Hit and friends), so
// the match state, CustomPlayerConfig and slot semantics are exactly the ones
// the online host setup shares. The online server browser, host setup and
// lobby (online_browser_menu.go, online_lobby_menu.go) reuse these builders.
//
// EDIT LOADOUT opens the full-screen Loadout screen (loadout_menu.go), which
// writes through the same player-editor activation codes.

var customLocalTabLabels = []string{"MATCH", "MAPS", "PLAYERS"}

var (
	customLocalTabPages = [3]int{customPageGame, customPageMaps, customPagePlayers}
	customLocalTabCodes = [3]int{menuHitCustomV2TabMatch, menuHitCustomV2TabPlaylist, menuHitCustomV2TabPlayers}
)

var (
	customLocalTabBarRect   = Rect{X: 470, Y: 30, W: 410, H: 40}
	customLocalModePanel    = Rect{X: uiMargin, Y: uiContentTop, W: 280, H: uiContentBottom - uiContentTop}
	customLocalSettingPanel = Rect{X: 316, Y: uiContentTop, W: 564, H: uiContentBottom - uiContentTop}
	customLocalRoundPanel   = Rect{X: uiMargin, Y: uiContentTop, W: 300, H: uiContentBottom - uiContentTop}
	customLocalRoundList    = Rect{X: 30, Y: 136, W: 280, H: 9 * 32}
	customLocalMapPanel     = Rect{X: 336, Y: uiContentTop, W: 544, H: uiContentBottom - uiContentTop}
	customLocalMapSetRect   = Rect{X: 350, Y: 136, W: 180, H: 34}
	customLocalApplyAllRect = Rect{X: 666, Y: 136, W: 200, H: 34}
	customLocalModalRect    = Rect{X: 30, Y: 26, W: 840, H: 548}
)

const (
	customLocalMapCols  = 5
	customLocalMapTileH = 92.0
)

func customLocalTabIndex(page int) int {
	for i, p := range customLocalTabPages {
		if p == page {
			return i
		}
	}
	return 0
}

func customLocalModeRect(i int) Rect {
	return Rect{X: customLocalModePanel.X + 14, Y: customLocalModePanel.Y + 48 + float64(i)*56, W: customLocalModePanel.W - 28, H: 48}
}

func customLocalSettingRow(i int) Rect {
	return Rect{X: customLocalSettingPanel.X + 14, Y: 196 + float64(i)*46, W: customLocalSettingPanel.W - 28, H: 40}
}

func customLocalMapTileRect(i int) Rect {
	w := (customLocalMapPanel.W - 28 - float64(customLocalMapCols-1)*8) / customLocalMapCols
	col, row := i%customLocalMapCols, i/customLocalMapCols
	return Rect{X: customLocalMapPanel.X + 14 + float64(col)*(w+8), Y: 182 + float64(row)*(customLocalMapTileH+8), W: w, H: customLocalMapTileH}
}

// customLocalCardRect is a PLAYERS column (the classic card's column,
// cardX = 20 + slot*220, 200 wide).
func customLocalCardRect(slot int) Rect {
	return Rect{X: 20 + float64(slot)*220, Y: uiContentTop, W: 200, H: uiContentBottom - uiContentTop}
}

func customLocalTypeIndex(t int) int {
	switch t {
	case 1:
		return 0
	case 2:
		return 1
	default:
		return 2
	}
}

var customLocalTypeActions = [3]int{playerSlotActionHuman, playerSlotActionAI, playerSlotActionClear}

// ---------------------------------------------------------------------------
// Update / draw entry points

func (g *Game) updateCustomLocalMenu() {
	g.updateCustomWarning()
	// A focused text field owns the keyboard for the whole frame, including
	// the Enter/Esc that ends the edit (typed Q/E must not switch tabs), so
	// the filter is decided before the text input can end the edit.
	textFocus := g.customLocalTextFocus()
	if g.customPage == customPageGame {
		g.updateCustomLivesInput()
	}
	if g.customPage == customPagePlayers {
		g.customMenuX = 0
		g.updateCustomNameInput()
	}
	g.updateCustomV2MapThumbPreload()
	in := g.readUIInput(&g.customUI)
	if textFocus {
		in = in.pointerOnly()
	}
	g.runCustomLocalUpdate(in)
}

func (g *Game) customLocalTextFocus() bool { return g.customNameFocus > 0 || g.customLivesFocus }

// runCustomLocalUpdate is one update pass for a frame of input (tests drive it).
func (g *Game) runCustomLocalUpdate(in uiInput) {
	ctx := g.customUI.begin(uiPassUpdate, nil, in)
	g.buildCustomLocal(ctx)
	ctx.end()
}

func (g *Game) drawCustomLocalMenu(screen *ebiten.Image) {
	ctx := g.customUI.begin(uiPassDraw, screen, uiInput{X: g.uiLastCursorX, Y: g.uiLastCursorY})
	g.buildCustomLocal(ctx)
	ctx.end()
}

// drawCustomGameMenu routes screenCustomGame drawing like updateCustomGameMenu.
func (g *Game) drawCustomGameMenu(screen *ebiten.Image) {
	if g.customOnlineMode || g.customNetworkOpen {
		g.drawOnlineCustomGame(screen)
		return
	}
	g.drawCustomLocalMenu(screen)
}

// customLocalActivate runs an existing Custom Game action and turns the
// source warnings (lives / player count) into toasts.
func (g *Game) customLocalActivate(code int) {
	if g.onlineLobbyIsClient() {
		// A guest only ever edits its own slot (the old client click path).
		g.onlineClientActivate(code)
		return
	}
	before := g.customWarningFrame
	g.activateCustomV2Hit(code)
	if g.customWarningFrame == 2 && before != 2 {
		switch g.customWarning {
		case 1:
			g.customUI.pushToast("LIVES MUST BE AT LEAST 1", uiToastWarning)
		case 2:
			g.customUI.pushToast("ADD AT LEAST 2 PLAYERS (HUMAN OR AI) TO START", uiToastWarning)
		}
	}
}

// ---------------------------------------------------------------------------
// Screen

func (g *Game) customLocalBreadcrumb() string {
	parts := []string{"MAIN MENU", "VERSUS", developedModeTitle(g.customMode)}
	if modeHasLives(g.customMode) {
		parts = append(parts, fmt.Sprintf("%d LIVES", g.customLives))
	}
	rounds := clampCustomSeriesRounds(g.customSeriesRounds)
	if rounds == 1 {
		parts = append(parts, "1 ROUND")
	} else {
		parts = append(parts, fmt.Sprintf("%d ROUNDS", rounds))
	}
	return strings.Join(parts, "  /  ")
}

func (g *Game) buildCustomLocal(ctx *uiCtx) {
	ctx.Page(0.92)
	if g.customLoadoutSlot > 0 {
		if g.loadoutOpen() && !g.customRulesOpen {
			g.buildCustomLoadout(ctx)
			return
		}
		// The slot was emptied or the page changed under the loadout.
		g.customLoadoutSlot = 0
	}
	ctx.Header("CUSTOM GAME", g.customLocalBreadcrumb())
	tab := customLocalTabIndex(g.customPage)
	if t := ctx.TabBar("cust.tabs", customLocalTabBarRect, customLocalTabLabels, tab); t != tab {
		g.closeCustomLoadout()
		g.customLivesFocus = false
		g.customLocalActivate(customLocalTabCodes[t])
	}

	switch g.customPage {
	case customPageMaps:
		g.buildCustomLocalMaps(ctx)
	case customPagePlayers:
		g.buildCustomLocalPlayers(ctx)
	default:
		g.buildCustomLocalMatch(ctx, false)
	}

	backLabel, nextLabel := "BACK", "START MATCH"
	switch g.customPage {
	case customPageGame:
		backLabel, nextLabel = "MAIN MENU", "NEXT: MAPS"
	case customPageMaps:
		nextLabel = "NEXT: PLAYERS"
	}
	if ctx.Button("cust.back", uiActionBackRect(), backLabel, uiButtonOpts{InfoTitle: backLabel}) {
		g.customLocalBack()
		return
	}
	if ctx.Button("cust.next", uiActionPrimaryRect(220), nextLabel, uiButtonOpts{Kind: uiButtonPrimary, InfoTitle: nextLabel}) {
		g.closeCustomLoadout()
		g.customLivesFocus = false
		g.customLocalActivate(menuHitCustomV2Start)
	}

	ctx.KeyHintBar([]uiKeyHint{{"ESC", "Back"}, {"UP/DOWN", "Move"}, {"LEFT/RIGHT", "Change"}, {"ENTER", "Select"}, {"Q", "Prev tab"}, {"E", "Next tab"}})
	ctx.Toasts()

	if g.customRulesOpen {
		if g.customChaosOpen {
			g.buildCustomLocalChaos(ctx)
		} else {
			g.buildCustomLocalModifiers(ctx)
		}
		return
	}
	if ctx.Back() {
		g.customLocalBack()
	}
}

func (g *Game) customLocalBack() {
	g.closeCustomLoadout()
	g.customLivesFocus = false
	if g.customPage == customPageGame {
		g.customUI.reset()
	}
	g.customLocalActivate(menuHitCustomV2Back)
}

// ---------------------------------------------------------------------------
// MATCH

func (g *Game) customRulesSummary() string {
	if g.customPerRoundRules {
		return fmt.Sprintf("PER ROUND / MAP - %d ROUNDS", clampCustomSeriesRounds(g.customSeriesRounds))
	}
	if g.customRandomRules {
		return fmt.Sprintf("RANDOM RULES - %d PER ROUND", g.customRandomCountForEdit())
	}
	count := 0
	for id := 1; id <= customModifierJetpacks; id++ {
		if g.customModifierEnabledForEdit(id) {
			count++
		}
	}
	switch count {
	case 0:
		return "NONE"
	case 1:
		return "1 ACTIVE"
	}
	return fmt.Sprintf("%d ACTIVE", count)
}

func (g *Game) customChaosSummary() string {
	if !g.customChaosEnabled {
		return "OFF"
	}
	return fmt.Sprintf("ON - %d EVENTS - EVERY %ds", g.chaosSelectedCount(), g.customChaosInterval)
}

// buildCustomLocalMatch is the MATCH tab. online adds the host's ONLINE
// RULES row (the online lobby uses ctx.readOnly for guests).
func (g *Game) buildCustomLocalMatch(ctx *uiCtx, online bool) {
	ctx.Panel(customLocalModePanel, "GAME MODE")
	for i, row := range customV2ModeRows {
		desc := strings.Join(developedModeDescription(row.mode), " ")
		if ctx.Button(fmt.Sprintf("cust.mode.%d", row.mode), customLocalModeRect(i), row.label, uiButtonOpts{
			Selected: g.customMode == row.mode, InfoTitle: row.label, InfoBody: desc}) {
			g.customLocalActivate(menuHitModeBase + row.mode)
		}
	}

	panel := customLocalSettingPanel
	ctx.Panel(panel, "")
	if ctx.drawing() {
		uiText(ctx.dst, developedModeTitle(g.customMode), true, uiSizeSection, uiTextLight, panel.X+14, panel.Y+26, uiAlignLeft)
		uiParagraph(ctx.dst, strings.Join(developedModeDescription(g.customMode), " "), false, uiSizeBody, uiTextMuted, panel.X+14, panel.Y+56, panel.W-28, 20, 2)
	}

	// LIVES: stepper, the number itself is a typed field (source inputlives).
	ctrl := ctx.SettingRow(customLocalSettingRow(0), "LIVES", "cust.lives", 190)
	stepR := Rect{X: ctrl.X, Y: ctrl.Y, W: 200, H: ctrl.H}
	if modeHasLives(g.customMode) {
		value := fmt.Sprintf("%d", g.customLives)
		if g.customLivesFocus {
			value += "_"
		}
		if d := ctx.Stepper("cust.lives", stepR, value, g.customLives > 1, g.customLives < 99999, false,
			"LIVES", "Lives per player. Click the number to type a value, or use LEFT / RIGHT."); d != 0 {
			if d < 0 {
				g.customLocalActivate(menuHitModeLivesMinus)
			} else {
				g.customLocalActivate(menuHitModeLivesPlus)
			}
		}
		if x, y, ok := ctx.ClickedAt("cust.lives"); ok && (Rect{X: stepR.X + 40, Y: stepR.Y, W: stepR.W - 80, H: stepR.H}).Contains(x, y) {
			g.customLocalActivate(menuHitModeLives)
		}
		if g.customLivesFocus && ctx.PressedOutside("cust.lives") {
			g.customLivesFocus = false
		}
		if !ctx.readOnly {
			ctx.Label("CLICK THE NUMBER TO TYPE", false, uiSizeHint, uiTextMuted, stepR.X+stepR.W+12, stepR.Y+stepR.H/2, uiAlignLeft)
		}
	} else {
		rule := "SET BY THE MODE"
		if g.customMode == SourceGameModeSurvival {
			rule = "10 - SET BY THE MODE"
		}
		ctx.Label(rule, true, uiSizeBody, uiTextMuted, ctrl.X+4, ctrl.Y+ctrl.H/2, uiAlignLeft)
	}

	ctrl = ctx.SettingRow(customLocalSettingRow(1), "ROUNDS", "cust.rounds", 190)
	stepR = Rect{X: ctrl.X, Y: ctrl.Y, W: 200, H: ctrl.H}
	rounds := clampCustomSeriesRounds(g.customSeriesRounds)
	if d := ctx.Stepper("cust.rounds", stepR, fmt.Sprintf("%d", rounds), rounds > 1, rounds < customSeriesMaxRounds, false,
		"ROUNDS", "Maps in the series. Each round can use its own map on the MAPS tab."); d != 0 {
		if d < 0 {
			g.customLocalActivate(menuHitModeRoundsMinus)
		} else {
			g.customLocalActivate(menuHitModeRoundsPlus)
		}
		g.ensureCustomPlaylistState()
	}
	ctx.Label("EACH ROUND HAS ITS OWN MAP", false, uiSizeHint, uiTextMuted, stepR.X+stepR.W+12, stepR.Y+stepR.H/2, uiAlignLeft)

	g.customLocalToggleRow(ctx, 2, "CRATES", "cust.crates", modeHasCrateToggle(g.customMode), g.CrateON, menuHitModeCrates,
		"Weapon crates drop during the match.", g.customMode == SourceGameModeSurvival)
	g.customLocalToggleRow(ctx, 3, "PICKUPS", "cust.pickups", modeHasPickupToggle(g.customMode), g.PowerON, menuHitModePickups,
		"Power-up pickups appear during the match.", false)

	ctrl = ctx.SettingRow(customLocalSettingRow(4), "MODIFIERS", "cust.modifiers", 190)
	ctx.Label(g.customRulesSummary(), true, uiSizeBody, uiTextLight, ctrl.X+4, ctrl.Y+ctrl.H/2, uiAlignLeft)
	if ctx.Button("cust.modifiers", Rect{X: ctrl.X + ctrl.W - 140, Y: ctrl.Y, W: 140, H: ctrl.H}, "CONFIGURE", uiButtonOpts{Size: uiSizeBody,
		InfoTitle: "MODIFIERS", InfoBody: "Weapon rules, gravity, movement, ammo and crate modifiers, random rules and per-round rules."}) {
		g.customLivesFocus = false
		g.customLocalActivate(menuHitCustomModifiersOpen)
	}

	ctrl = ctx.SettingRow(customLocalSettingRow(5), "CHAOS MODE", "cust.chaos", 190)
	ctx.Label(g.customChaosSummary(), true, uiSizeBody, uiTextLight, ctrl.X+4, ctrl.Y+ctrl.H/2, uiAlignLeft)
	if ctx.Button("cust.chaos", Rect{X: ctrl.X + ctrl.W - 140, Y: ctrl.Y, W: 140, H: ctrl.H}, "SETUP", uiButtonOpts{Size: uiSizeBody,
		InfoTitle: "CHAOS MODE", InfoBody: "Random events fire during the match, drawn from the event pool you choose."}) {
		g.customLivesFocus = false
		g.customLocalActivate(menuHitCustomModifiersOpen)
		g.customLocalActivate(menuHitChaosSetup)
	}

	if online {
		g.buildOnlineRulesRow(ctx, customLocalSettingRow(6))
		return
	}
	if ctx.drawing() {
		_, body := ctx.FocusInfo()
		if body != "" {
			uiParagraph(ctx.dst, body, false, uiSizeHint, uiTextMuted, panel.X+16, 484, panel.W-32, 17, 2)
		}
	}
}

func (g *Game) customLocalToggleRow(ctx *uiCtx, row int, label, id string, available, on bool, code int, info string, forcedOn bool) {
	ctrl := ctx.SettingRow(customLocalSettingRow(row), label, id, 190)
	if !available {
		state := "OFF - SET BY THE MODE"
		if forcedOn {
			state = "ON - SET BY THE MODE"
		}
		ctx.Label(state, true, uiSizeBody, uiTextMuted, ctrl.X+4, ctrl.Y+ctrl.H/2, uiAlignLeft)
		return
	}
	if v := ctx.Toggle(id, Rect{X: ctrl.X, Y: ctrl.Y, W: 200, H: ctrl.H}, on, false, label, info); v != on {
		g.customLocalActivate(code)
	}
}

// ---------------------------------------------------------------------------
// MAPS

func customLocalMapTileLabel(choice customV2MapChoice) string {
	switch {
	case choice.value == 0 || choice.value == customGM2RandomNew || choice.value == customGM2RandomAll:
		return "RANDOM"
	case isGM2MapID(choice.value):
		n := gm2SourceMapNumber(choice.value)
		return fmt.Sprintf("%d %s", n, strings.ToUpper(gm2MapDisplayNames[n]))
	}
	return fmt.Sprintf("%d %s", choice.value, sourceMapDisplayNames[choice.value])
}

func (g *Game) buildCustomLocalMaps(ctx *uiCtx) {
	g.ensureCustomPlaylistState()
	rounds := clampCustomSeriesRounds(g.customSeriesRounds)

	ctx.Panel(customLocalRoundPanel, "ROUNDS")
	rows := make([]string, rounds)
	for i := range rows {
		rows[i] = fmt.Sprintf("ROUND %02d   %s", i+1, customSeriesMapLabel(g.customSeriesMapForRound(i+1)))
	}
	if picked := ctx.List("cust.rounds.list", customLocalRoundList, rows, g.customPlaylistSelectedRound-1, &g.customPlaylistScroll); picked >= 0 {
		g.customLocalActivate(menuHitCustomV2RoundBase + picked + 1)
	}
	if ctx.drawing() {
		uiParagraph(ctx.dst, "Pick a round, then its map. The number of rounds is set on the MATCH tab.", false, uiSizeHint, uiTextMuted,
			customLocalRoundPanel.X+14, customLocalRoundList.Y+customLocalRoundList.H+18, customLocalRoundPanel.W-28, 17, 3)
	}

	panel := customLocalMapPanel
	round := g.customPlaylistSelectedRound
	ctx.Panel(panel, fmt.Sprintf("MAP FOR ROUND %d", round))
	set := 0
	if g.customPlaylistGM2 {
		set = 1
	}
	gm2 := gm2MapAssetsAvailable()
	setInfo := "Gun Mayhem 1 or Gun Mayhem 2 arenas."
	if !gm2 {
		setInfo = "GM2 arenas are not installed in this build."
	}
	if v := ctx.Segmented("cust.mapset", customLocalMapSetRect, []string{"GM1", "GM2"}, set, !gm2, "MAP SET", setInfo); v != set {
		if v == 1 {
			g.customLocalActivate(menuHitCustomV2PlaylistGM2)
		} else {
			g.customLocalActivate(menuHitCustomV2PlaylistGM1)
		}
	}
	if ctx.Button("cust.applyall", customLocalApplyAllRect, "APPLY TO ALL ROUNDS", uiButtonOpts{Size: uiSizeBody,
		InfoTitle: "APPLY TO ALL ROUNDS", InfoBody: "Use round " + fmt.Sprint(round) + "'s map for every round."}) {
		g.customLocalActivate(menuHitCustomV2ApplyAll)
		g.customUI.pushToast(fmt.Sprintf("ALL ROUNDS: %s", customSeriesMapLabel(g.customSeriesMapForRound(round))), uiToastOK)
	}

	selected := g.customSeriesMapForRound(round)
	for i, choice := range customV2MapChoices(g.customPlaylistGM2) {
		value := choice.value
		if ctx.Tile(fmt.Sprintf("cust.map.%d", i), customLocalMapTileRect(i), customLocalMapTileLabel(choice), uiTileOpts{
			Selected:  value == selected,
			InfoTitle: choice.label,
			InfoBody:  fmt.Sprintf("Play %s in round %d.", customSeriesMapLabel(value), round),
			Art: func(dst *ebiten.Image, inner Rect) {
				g.drawOnlineMapThumbIfReady(dst, value, inner)
			},
		}) {
			g.customLocalActivate(menuHitCustomV2MapBase + i)
		}
	}
	if ctx.drawing() {
		title, _ := ctx.FocusInfo()
		if title == "" || !strings.HasPrefix(title, "MAP") && !strings.HasPrefix(title, "RANDOM") {
			title = customSeriesMapLabel(selected)
		}
		strip := Rect{X: panel.X + 14, Y: 474, W: panel.W - 28, H: 34}
		uiFillRound(ctx.dst, strip, 4, uiInk900)
		uiTextIn(ctx.dst, fmt.Sprintf("ROUND %d:  %s", round, strings.ToUpper(title)), true, uiSizeBody, uiTextLight, strip, uiAlignLeft, 12)
	}
}

// ---------------------------------------------------------------------------
// PLAYERS

func customLocalControlsHint(c Controls) string {
	keys := []string{optionsKeyName(c.Up), optionsKeyName(c.Left), optionsKeyName(c.Down), optionsKeyName(c.Right)}
	return "KEYS  " + strings.Join(keys, " ") + "  /  FIRE " + optionsKeyName(c.Shoot)
}

func (g *Game) buildCustomLocalPlayers(ctx *uiCtx) {
	for slot := range g.customPlayers {
		g.buildCustomLocalPlayerCard(ctx, slot)
	}
	// Back from the loadout: the keyboard lands on the EDIT button it left.
	if ret := g.customLoadoutReturn; ret > 0 && ctx.pass == uiPassUpdate {
		g.customLoadoutReturn = 0
		ctx.st.focus = fmt.Sprintf("cust.p%d.edit", ret-1)
	}
}

func (g *Game) buildCustomLocalPlayerCard(ctx *uiCtx, slot int) {
	cfg := g.customPlayers[slot]
	r := customLocalCardRect(slot)
	x := r.X + 12
	w := r.W - 24
	empty := cfg.Type == 0
	kind := "EMPTY"
	switch cfg.Type {
	case 1:
		kind = "HUMAN"
	case 2:
		kind = "AI BOT"
	}
	ctx.PlayerCard(r, slot, cfg.Color, empty, kind)

	nameID := fmt.Sprintf("cust.p%d.name", slot)
	focused := g.customNameFocus == slot+1
	if ctx.TextField(nameID, Rect{X: x, Y: r.Y + 48, W: w, H: 32}, cfg.Name, fmt.Sprintf("Player %d", slot+1), focused, empty,
		"NAME", "Click to type a name. ENTER or ESC finishes.") {
		g.customLocalActivate(menuHitPlayerColorBase + 500 + slot)
	}
	if focused && ctx.PressedOutside(nameID) {
		g.customNameFocus = 0
	}

	typeIdx := customLocalTypeIndex(cfg.Type)
	if v := ctx.Segmented(fmt.Sprintf("cust.p%d.type", slot), Rect{X: x, Y: r.Y + 88, W: w, H: 32}, []string{"HUMAN", "AI", "OFF"}, typeIdx, false,
		"PLAYER TYPE", "HUMAN uses this player's keys, AI is a bot, OFF leaves the slot empty."); v != typeIdx {
		g.customLocalActivate(playerSlotHit(slot, customLocalTypeActions[v]))
	}

	if empty {
		if ctx.drawing() {
			uiText(ctx.dst, "SLOT EMPTY", true, uiSizeSection, uiTextDark, x, r.Y+160, uiAlignLeft)
			uiParagraph(ctx.dst, "Add a human player or an AI bot to this slot.", false, uiSizeBody, uiTextMutedDk, x, r.Y+192, w, 19, 3)
		}
		if ctx.Button(fmt.Sprintf("cust.p%d.addhuman", slot), Rect{X: x, Y: r.Y + 318, W: w, H: 40}, "ADD HUMAN", uiButtonOpts{Kind: uiButtonPrimary, Size: uiSizeBody}) {
			g.customLocalActivate(playerSlotHit(slot, playerSlotActionHuman))
		}
		if ctx.Button(fmt.Sprintf("cust.p%d.addai", slot), Rect{X: x, Y: r.Y + 366, W: w, H: 40}, "ADD AI BOT", uiButtonOpts{Size: uiSizeBody}) {
			g.customLocalActivate(playerSlotHit(slot, playerSlotActionAI))
		}
		return
	}

	if ctx.drawing() {
		preview := Rect{X: x, Y: r.Y + 128, W: w, H: 124}
		uiFillRound(ctx.dst, preview, 6, uiPaperShade)
		g.drawSetupPreviewIn(ctx.dst, cfg, uiInset(preview, 6), 1)
		cfgGear := migrateLegacyGearPerk(cfg)
		rows := [][2]string{
			{"GUN", weaponDisplayName(cfg.Gun)},
			{"PERK", onlinePerkName(cfgGear.Perk)},
			{"ABILITY", abilityName(cfgGear.Ability)},
			{"GRENADE", strings.ToUpper(throwableName(cfg.Throwable))},
		}
		for i, row := range rows {
			mid := r.Y + 268 + float64(i)*20
			uiText(ctx.dst, row[0], true, uiSizeHint, uiTextMutedDk, x, mid, uiAlignLeft)
			uiText(ctx.dst, uiFit(strings.ToUpper(row[1]), true, uiSizeHint, w-62), true, uiSizeHint, uiTextDark, x+62, mid, uiAlignLeft)
		}
	}

	if g.customMode == SourceGameModeTeams {
		team := 0
		if cfg.Team == 2 {
			team = 1
		}
		if v := ctx.Segmented(fmt.Sprintf("cust.p%d.team", slot), Rect{X: x, Y: r.Y + 344, W: w, H: 28}, []string{"TEAM A", "TEAM B"}, team, false,
			"TEAM", "Team Battle: players on the same team fight together."); v != team {
			g.customLocalActivate(customTeamHit(slot, v+1))
		}
	} else if ctx.drawing() {
		hint := "CONTROLLED BY AI"
		if cfg.Type == 1 {
			hint = customLocalControlsHint(g.controlConfigs[slot])
		}
		uiText(ctx.dst, uiFit(hint, false, uiSizeHint, w), false, uiSizeHint, uiTextMutedDk, x, r.Y+358, uiAlignLeft)
	}

	if ctx.Button(fmt.Sprintf("cust.p%d.edit", slot), Rect{X: x, Y: r.Y + 378, W: w, H: 36}, "EDIT LOADOUT", uiButtonOpts{Size: uiSizeBody,
		InfoTitle: "EDIT LOADOUT", InfoBody: "Character, colour, starting weapon, perk, ability / gear and throwable."}) {
		g.openCustomLoadout(slot)
	}
}

// ---------------------------------------------------------------------------
// Modifiers / chaos modals

func (g *Game) buildCustomLocalModifiers(ctx *uiCtx) {
	m := customLocalModalRect
	title := "MODIFIERS"
	if g.customPerRoundRules {
		title = fmt.Sprintf("MODIFIERS - ROUND %d OF %d", g.customModifierEditRound(), clampCustomSeriesRounds(g.customSeriesRounds))
	}
	ctx.BeginModal(m, title)
	if ctx.drawing() {
		uiText(ctx.dst, "Pick any modifiers. Conflicting choices are switched off automatically.", false, uiSizeBody, uiTextMutedDk, m.X+14, m.Y+52, uiAlignLeft)
	}
	colW := (m.W - 40 - 2*12) / 3
	for i, spec := range customModifierSpecs {
		col, row := i%3, i/3
		r := Rect{X: m.X + 20 + float64(col)*(colW+12), Y: m.Y + 72 + float64(row)*44, W: colW, H: 36}
		if ctx.Button(fmt.Sprintf("cust.mod.%d", spec.ID), r, spec.Label, uiButtonOpts{Selected: g.customModifierEnabledForEdit(spec.ID), Size: uiSizeBody,
			InfoTitle: spec.Label, InfoBody: customModifierHint(spec.ID)}) {
			g.customLocalActivate(menuHitCustomModifierBase + spec.ID)
		}
	}

	y := m.Y + 72 + 5*44 + 10
	if ctx.Button("cust.mod.random", Rect{X: m.X + 20, Y: y, W: colW, H: 36}, "RANDOM RULES", uiButtonOpts{Selected: g.customRandomRulesForEdit(), Size: uiSizeBody,
		InfoTitle: "RANDOM RULES", InfoBody: "Ignores the manual picks and draws compatible modifiers when each round starts."}) {
		g.customLocalActivate(menuHitCustomModifiersRandom)
	}
	if ctx.drawing() {
		uiText(ctx.dst, "HOW MANY", true, uiSizeHint, uiTextMutedDk, m.X+32+colW, y+18, uiAlignLeft)
	}
	count := g.customRandomCountForEdit() - 1
	if v := ctx.Segmented("cust.mod.count", Rect{X: m.X + 20 + colW + 90, Y: y, W: colW - 90, H: 36}, []string{"1", "2", "3"}, count, false,
		"NUMBER OF RANDOM MODIFIERS", "How many compatible random modifiers are drawn for each round."); v != count {
		g.customLocalActivate(menuHitCustomModifiersRandom1 + v)
	}

	y += 52
	scope := 0
	if g.customPerRoundRules {
		scope = 1
	}
	if v := ctx.Segmented("cust.mod.scope", Rect{X: m.X + 20, Y: y, W: 2*colW + 12, H: 36}, []string{"SAME FOR WHOLE SERIES", "EACH ROUND / MAP"}, scope, false,
		"RULE SCOPE", "Use one modifier setup for the whole series, or give every round / map its own rules."); v != scope {
		if v == 1 {
			g.customLocalActivate(menuHitCustomModifiersScopePerRound)
		} else {
			g.customLocalActivate(menuHitCustomModifiersScopeSeries)
		}
	}
	if g.customPerRoundRules {
		round := g.customModifierEditRound()
		rounds := clampCustomSeriesRounds(g.customSeriesRounds)
		if d := ctx.Stepper("cust.mod.round", Rect{X: m.X + 20 + 2*(colW+12), Y: y, W: colW, H: 36}, fmt.Sprintf("ROUND %d / %d", round, rounds),
			round > 1, round < rounds, false, "EDITING ROUND", "Each round / map remembers its own rules."); d != 0 {
			if d < 0 {
				g.customLocalActivate(menuHitCustomModifiersRoundPrev)
			} else {
				g.customLocalActivate(menuHitCustomModifiersRoundNext)
			}
		}
	}

	y += 52
	chaosLabel := "CHAOS SETUP"
	if g.customChaosEnabled {
		chaosLabel = fmt.Sprintf("CHAOS ON - %d EVENTS", g.chaosSelectedCount())
	}
	if ctx.Button("cust.mod.chaos", Rect{X: m.X + 20, Y: y, W: colW, H: 36}, chaosLabel, uiButtonOpts{Selected: g.customChaosEnabled, Size: uiSizeBody,
		InfoTitle: "CHAOS MODE", InfoBody: "Random events during the match, drawn from a pool you choose."}) {
		g.customLocalActivate(menuHitChaosSetup)
	}

	if ctx.drawing() {
		title, body := ctx.FocusInfo()
		if body == "" {
			title, body = "MODIFIERS", "Hover or focus a modifier to see what it changes."
		}
		uiText(ctx.dst, title, true, uiSizeBody, uiTextDark, m.X+20, m.Y+m.H-82, uiAlignLeft)
		uiParagraph(ctx.dst, body, false, uiSizeBody, uiTextMutedDk, m.X+20, m.Y+m.H-60, m.W-40-220, 19, 2)
	}
	if ctx.Button("cust.mod.done", Rect{X: m.X + m.W - 220, Y: m.Y + m.H - 64, W: 200, H: 44}, "DONE", uiButtonOpts{Kind: uiButtonPrimary}) {
		g.customLocalActivate(menuHitCustomModifiersClose)
	}
	ctx.EndModal()
	if ctx.Back() {
		g.customLocalActivate(menuHitCustomModifiersClose)
	}
}

func (g *Game) buildCustomLocalChaos(ctx *uiCtx) {
	m := customLocalModalRect
	ctx.BeginModal(m, "CHAOS MODE")
	if ctx.drawing() {
		uiText(ctx.dst, "Chaos only draws events from the pool selected below.", false, uiSizeBody, uiTextMutedDk, m.X+14, m.Y+52, uiAlignLeft)
	}
	y := m.Y + 70
	on := 0
	if g.customChaosEnabled {
		on = 1
	}
	if v := ctx.Segmented("cust.chaos.on", Rect{X: m.X + 20, Y: y, W: 160, H: 36}, []string{"OFF", "ON"}, on, false,
		"CHAOS MODE", "Turn Chaos Mode on or off for this match."); v != on {
		g.customLocalActivate(menuHitChaosToggle)
	}
	if d := ctx.Stepper("cust.chaos.every", Rect{X: m.X + 196, Y: y, W: 220, H: 36}, fmt.Sprintf("EVERY %ds", g.customChaosInterval),
		g.customChaosInterval > 10, g.customChaosInterval < 90, false, "INTERVAL", "Seconds between chaos events (10-90)."); d != 0 {
		if d < 0 {
			g.customLocalActivate(menuHitChaosIntervalMinus)
		} else {
			g.customLocalActivate(menuHitChaosIntervalPlus)
		}
	}
	if ctx.Button("cust.chaos.all", Rect{X: m.X + 432, Y: y, W: 140, H: 36}, "SELECT ALL", uiButtonOpts{Size: uiSizeBody}) {
		g.customLocalActivate(menuHitChaosSelectAll)
	}
	if ctx.Button("cust.chaos.clear", Rect{X: m.X + 582, Y: y, W: 110, H: 36}, "CLEAR", uiButtonOpts{Size: uiSizeBody}) {
		g.customLocalActivate(menuHitChaosClear)
	}
	if ctx.drawing() {
		uiText(ctx.dst, fmt.Sprintf("POOL %d / %d", g.chaosSelectedCount(), len(chaosEffectSpecs)), true, uiSizeBody, uiTextDark, m.X+m.W-20, y+18, uiAlignRight)
	}

	const cols = 5
	colW := (m.W - 40 - (cols-1)*8) / cols
	for i, spec := range chaosEffectSpecs {
		col, row := i%cols, i/cols
		r := Rect{X: m.X + 20 + float64(col)*(colW+8), Y: y + 50 + float64(row)*34, W: colW, H: 30}
		if ctx.Button(fmt.Sprintf("cust.chaos.%d", spec.ID), r, spec.Label, uiButtonOpts{Selected: g.customChaosSelected[spec.ID], Size: uiSizeHint,
			InfoTitle: spec.Label, InfoBody: spec.Hint}) {
			g.customLocalActivate(menuHitChaosEffectBase + spec.ID)
		}
	}

	if ctx.drawing() {
		title, body := ctx.FocusInfo()
		if body == "" {
			title, body = "EVENT POOL", "Hover or focus an event for details. Selected events form the Chaos pool."
		}
		uiText(ctx.dst, title, true, uiSizeBody, uiTextDark, m.X+290, m.Y+m.H-54, uiAlignLeft)
		uiParagraph(ctx.dst, body, false, uiSizeHint, uiTextMutedDk, m.X+290, m.Y+m.H-34, m.W-290-240, 16, 2)
	}
	if ctx.Button("cust.chaos.back", Rect{X: m.X + 20, Y: m.Y + m.H - 64, W: 250, H: 44}, "BACK TO MODIFIERS", uiButtonOpts{}) {
		g.customLocalActivate(menuHitChaosBack)
	}
	if ctx.Button("cust.chaos.done", Rect{X: m.X + m.W - 220, Y: m.Y + m.H - 64, W: 200, H: 44}, "DONE", uiButtonOpts{Kind: uiButtonPrimary}) {
		g.customLocalActivate(menuHitCustomModifiersClose)
	}
	ctx.EndModal()
	if ctx.Back() {
		g.customLocalActivate(menuHitChaosBack)
	}
}
