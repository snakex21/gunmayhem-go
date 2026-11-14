package game

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Loadout screen (Mayhem UI, phase 4): PLAYERS -> EDIT LOADOUT opens one
// player's loadout as a full screen with five tabs (CHARACTER, WEAPON, PERK,
// ABILITY / GEAR, THROWABLE), a live preview of the character holding the
// chosen gun and a stats / description panel.
//
// Every pick is written through the classic editor's activation codes
// (customColorHit, customSelectHit, customDevelopedGunHit,
// customThrowableChoiceHit, the name focus code) via customLocalActivate, so
// the same setters, onlineMayEditCustomCharacterSlot guard and value checks
// apply as before. The item lists come from the classic pickers
// (developedCustomSelectionItems, selectableAbilityNumbers, ...), so the
// availability rules (GM2 cosmetics installed, selectable perks) are shared.

const (
	loadoutTabCharacter = iota
	loadoutTabWeapon
	loadoutTabPerk
	loadoutTabGear
	loadoutTabThrowable
	loadoutTabCount
)

var loadoutTabLabels = []string{"CHARACTER", "WEAPON", "PERK", "ABILITY/GEAR", "THROWABLE"}

const (
	loadoutPartColour = iota
	loadoutPartHat
	loadoutPartShirt
	loadoutPartFace
)

var (
	loadoutTabBarRect   = Rect{X: 300, Y: 30, W: 580, H: 40}
	loadoutPreviewCard  = Rect{X: uiMargin, Y: uiContentTop, W: 268, H: 268}
	loadoutInfoRect     = Rect{X: uiMargin, Y: uiContentTop + 276, W: 268, H: uiContentBottom - uiContentTop - 276}
	loadoutContentPanel = Rect{X: 300, Y: uiContentTop, W: 580, H: uiContentBottom - uiContentTop}
	loadoutPlayerRect   = Rect{X: uiMargin, Y: uiActionTop, W: 268, H: uiButtonH}
	loadoutDoneRect     = Rect{X: ScreenWidth - uiMargin - 200, Y: uiActionTop, W: 200, H: uiButtonH}
)

// loadoutChoice is one pickable item of a tab grid.
type loadoutChoice struct {
	id        string
	label     string // empty: icon-only tile
	selected  bool
	locked    bool
	infoTitle string
	infoBody  string
	art       func(dst *ebiten.Image, inner Rect)
	code      int // activation code written through customLocalActivate
	gun       int // WEAPON tab: weapon number for the stats panel
}

// ---------------------------------------------------------------------------
// Open / close

func (g *Game) loadoutSlot() int { return g.customLoadoutSlot - 1 }

func (g *Game) loadoutOpen() bool {
	slot := g.loadoutSlot()
	return slot >= 0 && slot < len(g.customPlayers) && g.customPage == customPagePlayers && g.customPlayers[slot].Type != 0
}

func (g *Game) openCustomLoadout(slot int) {
	if slot < 0 || slot >= len(g.customPlayers) || g.customPlayers[slot].Type == 0 {
		return
	}
	g.closeCustomLoadout()
	g.customLoadoutSlot = slot + 1
	if g.customLoadoutTab < 0 || g.customLoadoutTab >= loadoutTabCount {
		g.customLoadoutTab = loadoutTabCharacter
	}
	g.customLoadoutFocusPending = true
}

func (g *Game) closeCustomLoadout() {
	if slot := g.loadoutSlot(); slot >= 0 && slot < len(g.customEditor) {
		g.customEditor[slot] = 0
		g.customLoadoutReturn = slot + 1
	}
	g.customLoadoutSlot = 0
	g.customNameFocus = 0
	g.customLoadoutFocusPending = false
}

// loadoutSwitchPlayer moves the screen to the next / previous non-empty slot.
func (g *Game) loadoutSwitchPlayer(delta int) {
	slot := g.loadoutSlot()
	n := len(g.customPlayers)
	for i := 1; i < n; i++ {
		s := ((slot+delta*i)%n + n) % n
		if g.customPlayers[s].Type != 0 {
			g.customNameFocus = 0
			g.customEditor[slot] = 0
			g.customLoadoutSlot = s + 1
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Item lists (shared with the classic pickers)

func loadoutNumbers(editors ...int) []int {
	var out []int
	seen := map[int]bool{}
	for _, editor := range editors {
		for _, item := range developedCustomSelectionItems(editor) {
			if !seen[item.Number] {
				seen[item.Number] = true
				out = append(out, item.Number)
			}
		}
	}
	return out
}

func loadoutPartLabels() []string {
	if gm2CosmeticsAvailable() {
		return []string{"COLOUR", "HAT", "SHIRT", "FACE"}
	}
	return []string{"COLOUR", "HAT", "SHIRT"}
}

func cosmeticSetLabel(gm2 bool) string {
	if gm2 {
		return "GUN MAYHEM 2"
	}
	return "GUN MAYHEM"
}

func (g *Game) loadoutCharacterChoices(slot, part int) (choices []loadoutChoice, cols int) {
	cfg := g.customPlayers[slot]
	switch part {
	case loadoutPartHat:
		headgear := 0
		if a := migrateLegacyGearPerk(cfg).Ability; a == abilityGasMask || a == abilityThermalGoggles || a == abilityFlashGoggles {
			headgear = a
		}
		for _, n := range loadoutNumbers(customEditHat1, customEditHat2) {
			n := n
			_, gm2 := gm2HatSourceNumber(n)
			body := cosmeticSetLabel(gm2) + " hat."
			if headgear != 0 {
				body += " Your " + strings.ToLower(abilityName(headgear)) + " is drawn instead of the hat while equipped."
			}
			choices = append(choices, loadoutChoice{
				id: fmt.Sprintf("lo.hat.%d", n), selected: cfg.Hat == n,
				infoTitle: fmt.Sprintf("HAT %d", n), infoBody: body,
				art: func(dst *ebiten.Image, inner Rect) {
					drawSetupThumbFit(dst, g.loadoutCosmeticThumb(customEditHat1, n), inner, 1)
				},
				code: customSelectHit(slot, customEditHat1, n),
			})
		}
		return choices, 12
	case loadoutPartShirt:
		for _, n := range loadoutNumbers(customEditShirt) {
			n := n
			_, gm2 := gm2ShirtSourceNumber(n)
			choices = append(choices, loadoutChoice{
				id: fmt.Sprintf("lo.shirt.%d", n), selected: cfg.Shirt == n,
				infoTitle: fmt.Sprintf("SHIRT %d", n), infoBody: cosmeticSetLabel(gm2) + " shirt.",
				art: func(dst *ebiten.Image, inner Rect) {
					drawSetupThumbFit(dst, g.loadoutCosmeticThumb(customEditShirt, n), inner, 1)
				},
				code: customSelectHit(slot, customEditShirt, n),
			})
		}
		return choices, 10
	case loadoutPartFace:
		for _, n := range loadoutNumbers(customEditEyes) {
			n := n
			_, gm2 := gm2EyeSourceNumber(n)
			choices = append(choices, loadoutChoice{
				id: fmt.Sprintf("lo.face.%d", n), selected: cfg.Eyes == n,
				infoTitle: fmt.Sprintf("FACE %d", n), infoBody: cosmeticSetLabel(gm2) + " face.",
				art: func(dst *ebiten.Image, inner Rect) {
					drawSetupThumbFit(dst, g.loadoutCosmeticThumb(customEditEyes, n), inner, 1)
				},
				code: customSelectHit(slot, customEditEyes, n),
			})
		}
		return choices, 8
	}
	for _, n := range loadoutNumbers(customEditColor) {
		n := n
		_, gm2 := gm2ColorSourceNumber(n)
		choices = append(choices, loadoutChoice{
			id: fmt.Sprintf("lo.colour.%d", n), selected: cfg.Color == n,
			infoTitle: fmt.Sprintf("COLOUR %d", n), infoBody: cosmeticSetLabel(gm2) + " body colour. Your gun is tinted to match.",
			art: func(dst *ebiten.Image, inner Rect) {
				sw := developedPlayerColorSwatch(n)
				sw.A = 0xff
				uiFillRound(dst, uiInset(inner, 2), 5, sw)
			},
			code: customColorHit(slot, n),
		})
	}
	return choices, 10
}

func (g *Game) loadoutWeaponChoices(slot int) []loadoutChoice {
	cfg := g.customPlayers[slot]
	var out []loadoutChoice
	for _, n := range loadoutNumbers(customEditGun) {
		n := n
		code := customSelectHit(slot, customEditGun, n)
		if n >= 98 {
			code = customDevelopedGunHit(slot, n)
		}
		out = append(out, loadoutChoice{
			id: fmt.Sprintf("lo.gun.%d", n), label: strings.ToUpper(weaponDisplayName(n)), selected: cfg.Gun == n,
			infoTitle: strings.ToUpper(weaponDisplayName(n)), infoBody: weaponStatsLine(n),
			art: func(dst *ebiten.Image, inner Rect) {
				drawSetupThumbFit(dst, g.setupSelectionRaster(customEditGun, n), inner, 1)
			},
			code: code, gun: n,
		})
	}
	return out
}

// loadoutPerkInfo describes perks 1..9 from what the gameplay code does.
var loadoutPerkInfo = map[int]string{
	1: "No perk.",
	2: "Heavy guns never slow you down: your weight always stays at 1.",
	3: "Triple jump: one extra jump in the air.",
	4: "One in three crate pickups becomes the minigun.",
	5: "Your own shots have no recoil push.",
	6: "Explosions push you far less. Fire and gas burn for half as long.",
	7: "Every gun you pick up has +1/3 more ammo.",
	8: "+2 throwables at every spawn.",
	9: "Respawn with a random gun instead of the handgun (not in Gun Game or Instagib).",
}

func (g *Game) loadoutPerkChoices(slot int) []loadoutChoice {
	cfg := migrateLegacyGearPerk(g.customPlayers[slot])
	var out []loadoutChoice
	offered := map[int]bool{}
	for _, item := range developedPerkSelectionItems() {
		offered[item.Number] = true
	}
	// Menu order 1..9 (the source grid's parse order is not numeric).
	for _, n := range selectablePerkNumbers() {
		n := n
		if !offered[n] {
			continue
		}
		out = append(out, loadoutChoice{
			id: fmt.Sprintf("lo.perk.%d", n), label: onlinePerkName(n), selected: cfg.Perk == n,
			locked:    !g.customSelectionUnlocked(customEditPerk, n),
			infoTitle: onlinePerkName(n), infoBody: loadoutPerkInfo[n],
			art:  func(dst *ebiten.Image, inner Rect) { g.drawLoadoutPerkArt(dst, n, inner) },
			code: customSelectHit(slot, customEditPerk, n),
		})
	}
	return out
}

func (g *Game) loadoutGearChoices(slot int) []loadoutChoice {
	cfg := migrateLegacyGearPerk(g.customPlayers[slot])
	var out []loadoutChoice
	for i, a := range selectableAbilityNumbers() {
		a := a
		body := abilityDescription(a)
		switch {
		case a == abilityFlashGoggles || a == abilityThermalGoggles || a == abilityGasMask:
			body += " Gear: worn instead of your hat."
		case a == abilityKevlar:
			body += " Gear: a vest over your shirt."
		case a == abilityBallisticShield:
			body += " Gear: carried in the off hand (or on the back); your hat stays."
		}
		out = append(out, loadoutChoice{
			id: fmt.Sprintf("lo.ability.%d", a), label: abilityName(a), selected: cfg.Ability == a,
			infoTitle: abilityName(a), infoBody: body,
			art: func(dst *ebiten.Image, inner Rect) {
				s := math.Min(22, math.Min(inner.W, inner.H)/2.4)
				drawAbilityIcon(dst, a, inner.X+inner.W/2, inner.Y+inner.H/2, s, 1)
			},
			code: customSelectHit(slot, customEditAbility, i+1),
		})
	}
	return out
}

func throwableLoadoutDescription(kind int) string {
	p := throwableProfileFor(kind)
	fuse := float64(p.Fuse) / 30
	switch p.Behavior {
	case throwableBehaviorIgnite:
		return "Shatters on impact and leaves a strip of fire that burns and panics anyone inside."
	case throwableBehaviorShock:
		return fmt.Sprintf("Fuse %.1f s. A shock gas cloud that shoves players around. No damage.", fuse)
	case throwableBehaviorFlash:
		return fmt.Sprintf("Fuse %.1f s. A blinding flash: players who see it are blinded for a while.", fuse)
	case throwableBehaviorSmoke:
		return fmt.Sprintf("Fuse %.1f s. A thick smoke cloud that hides everyone inside it.", fuse)
	case throwableBehaviorTearGas:
		return fmt.Sprintf("Fuse %.1f s. Tear gas: players inside cough, move slower, jump lower and aim worse.", fuse)
	}
	return fmt.Sprintf("Explosive. Fuse %.1f s, blast radius %.0f, push %.0f.", fuse, p.Radius, p.HorizontalPush)
}

func (g *Game) loadoutThrowableChoices(slot int) []loadoutChoice {
	cfg := g.customPlayers[slot]
	var out []loadoutChoice
	for kind := 0; kind < throwableTypeCount; kind++ {
		kind := kind
		out = append(out, loadoutChoice{
			id: fmt.Sprintf("lo.throw.%d", kind), label: throwableName(kind), selected: cfg.Throwable == kind,
			infoTitle: throwableName(kind), infoBody: throwableLoadoutDescription(kind),
			art:  func(dst *ebiten.Image, inner Rect) { g.drawThrowableArt(dst, kind, inner) },
			code: customThrowableChoiceHit(slot, kind),
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Weapon stats

func weaponStatsLine(n int) string {
	def, ok := WeaponByNumber(n)
	if !ok {
		return ""
	}
	rounds := fmt.Sprint(def.Bullets)
	if def.Bullets >= 9999 {
		rounds = "INFINITE"
	}
	if isMeleeWeapon(n) {
		return fmt.Sprintf("Melee, %.1f swings/s, push %.0f. %s", weaponShotsPerSecond(def), def.Firepower, weaponArsenal2Description(n))
	}
	line := fmt.Sprintf("%s rounds, %.1f shots/s, push %.0f.", rounds, weaponShotsPerSecond(def), def.Firepower)
	if desc := weaponArsenal2Description(n); desc != "" {
		line += " " + desc
	}
	return line
}

func weaponShotsPerSecond(def WeaponDef) float64 {
	if def.ROF <= 0 {
		return 30
	}
	return 30 / float64(def.ROF)
}

func weaponFireModeLabel(n int) string {
	if spec, ok := developedWeaponSpec(n); ok && spec.FireMode == DevelopedFireAuto {
		return "FULL AUTO"
	}
	return "SEMI AUTO"
}

// loadoutWeaponMaxima are the roster maxima the stat bars are scaled to.
func loadoutWeaponMaxima() (rounds, rate, push float64) {
	for _, n := range starterHandgunNumbers() {
		def, ok := WeaponByNumber(n)
		if !ok {
			continue
		}
		if b := float64(def.Bullets); b < 9999 && b > rounds {
			rounds = b
		}
		rate = math.Max(rate, weaponShotsPerSecond(def))
		push = math.Max(push, def.Firepower)
	}
	return math.Max(rounds, 1), math.Max(rate, 1), math.Max(push, 1)
}

// ---------------------------------------------------------------------------
// Screen

func (g *Game) buildCustomLoadout(ctx *uiCtx) {
	slot := g.loadoutSlot()
	cfg := g.customPlayers[slot]
	locked := !g.onlineMayEditCustomCharacterSlot(slot)
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		name = fmt.Sprintf("PLAYER %d", slot+1)
	}
	trail := "MAIN MENU  /  VERSUS  /  PLAYERS"
	if g.customOnlineMode {
		trail = "ONLINE  /  LOBBY  /  PLAYERS"
	}
	ctx.Header("LOADOUT", fmt.Sprintf("%s  /  P%d %s", trail, slot+1, strings.ToUpper(name)))

	tab := g.customLoadoutTab
	if t := ctx.TabBar("lo.tabs", loadoutTabBarRect, loadoutTabLabels, tab); t != tab {
		g.customLoadoutTab = t
		g.customNameFocus = 0
		g.customLoadoutFocusPending = true
	}

	ctx.Panel(loadoutContentPanel, "")
	var choices []loadoutChoice
	grid := Rect{X: loadoutContentPanel.X + 14, Y: loadoutContentPanel.Y + 14, W: loadoutContentPanel.W - 28, H: loadoutContentPanel.H - 28}
	cols, tileH := 5, 0.0
	switch g.customLoadoutTab {
	case loadoutTabCharacter:
		grid = g.buildLoadoutCharacterHead(ctx, slot, locked)
		choices, cols = g.loadoutCharacterChoices(slot, g.customLoadoutPart)
	case loadoutTabWeapon:
		choices, cols, tileH = g.loadoutWeaponChoices(slot), 6, 70
	case loadoutTabPerk:
		choices, cols, tileH = g.loadoutPerkChoices(slot), 3, 112
	case loadoutTabGear:
		choices, cols, tileH = g.loadoutGearChoices(slot), 5, 88
	case loadoutTabThrowable:
		choices, cols, tileH = g.loadoutThrowableChoices(slot), 5, 150
	}
	info := g.buildLoadoutGrid(ctx, grid, cols, tileH, choices, locked)

	g.drawLoadoutPreview(ctx, slot)
	g.drawLoadoutInfo(ctx, slot, info)

	// A guest only has its own loadout; the host may look at every slot
	// (remote ones read-only, see onlineMayEditCustomCharacterSlot).
	switchable := g.activeCustomPlayerCount() > 1 && !g.onlineLobbyIsClient()
	if d := ctx.Stepper("lo.player", loadoutPlayerRect, fmt.Sprintf("P%d  %s", slot+1, strings.ToUpper(name)), switchable, switchable, g.onlineLobbyIsClient(),
		"PLAYER", "Switch to another player's loadout."); d != 0 {
		g.loadoutSwitchPlayer(d)
		return
	}
	if ctx.Button("lo.done", loadoutDoneRect, "DONE", uiButtonOpts{Kind: uiButtonPrimary, InfoTitle: "DONE", InfoBody: "Back to the PLAYERS tab."}) {
		g.closeCustomLoadout()
		return
	}
	ctx.KeyHintBar([]uiKeyHint{{"ESC", "Done"}, {"UP/DOWN", "Move"}, {"LEFT/RIGHT", "Move"}, {"ENTER", "Pick"}, {"Q", "Prev tab"}, {"E", "Next tab"}, {"TAB", "Next control"}})
	ctx.Toasts()
	if ctx.Back() {
		g.closeCustomLoadout()
	}
}

// buildLoadoutCharacterHead draws the NAME field and the part selector and
// returns the grid area under them.
func (g *Game) buildLoadoutCharacterHead(ctx *uiCtx, slot int, locked bool) Rect {
	p := loadoutContentPanel
	cfg := g.customPlayers[slot]
	ctx.Label("NAME", true, uiSizeButton, uiTextLight, p.X+14, p.Y+32, uiAlignLeft)
	focused := g.customNameFocus == slot+1
	if ctx.TextField("lo.name", Rect{X: p.X + 70, Y: p.Y + 14, W: 200, H: 36}, cfg.Name, fmt.Sprintf("Player %d", slot+1), focused, locked,
		"NAME", "Click or press ENTER to type a name. ENTER or ESC finishes.") {
		g.customLocalActivate(menuHitPlayerColorBase + 500 + slot)
	}
	if focused && ctx.PressedOutside("lo.name") {
		g.customNameFocus = 0
	}
	labels := loadoutPartLabels()
	if g.customLoadoutPart >= len(labels) {
		g.customLoadoutPart = loadoutPartColour
	}
	if v := ctx.Segmented("lo.part", Rect{X: p.X + 286, Y: p.Y + 14, W: p.W - 300, H: 36}, labels, g.customLoadoutPart, false,
		"CHARACTER PART", "Colour, hat, shirt and face. LEFT / RIGHT switch."); v != g.customLoadoutPart {
		g.customLoadoutPart = v
	}
	return Rect{X: p.X + 14, Y: p.Y + 62, W: p.W - 28, H: p.H - 76}
}

// buildLoadoutGrid lays the choices out in cols columns (tileH 0 fits the
// rows into area) and returns the choice whose details the info panel shows:
// the hovered / focused one, else the selected one.
func (g *Game) buildLoadoutGrid(ctx *uiCtx, area Rect, cols int, tileH float64, choices []loadoutChoice, locked bool) *loadoutChoice {
	if len(choices) == 0 || cols <= 0 {
		return nil
	}
	const gap = 6.0
	rows := (len(choices) + cols - 1) / cols
	w := (area.W - float64(cols-1)*gap) / float64(cols)
	if tileH <= 0 || float64(rows)*(tileH+gap)-gap > area.H {
		tileH = math.Min((area.H-float64(rows-1)*gap)/float64(rows), w+8)
	}
	infoID := ctx.InfoID()
	var info, selected *loadoutChoice
	for i := range choices {
		c := &choices[i]
		r := Rect{X: area.X + float64(i%cols)*(w+gap), Y: area.Y + float64(i/cols)*(tileH+gap), W: w, H: tileH}
		if ctx.Tile(c.id, r, c.label, uiTileOpts{Selected: c.selected, Disabled: locked || c.locked, Locked: c.locked,
			InfoTitle: c.infoTitle, InfoBody: c.infoBody, Art: c.art}) {
			g.customLocalActivate(c.code)
		}
		if c.id == infoID {
			info = c
		}
		if c.selected {
			selected = c
		}
	}
	if ctx.pass == uiPassUpdate && g.customLoadoutFocusPending {
		g.customLoadoutFocusPending = false
		if selected != nil {
			ctx.st.focus = selected.id
		} else {
			ctx.st.focus = choices[0].id
		}
	}
	if info == nil {
		info = selected
	}
	return info
}

func (g *Game) drawLoadoutPreview(ctx *uiCtx, slot int) {
	if !ctx.drawing() {
		return
	}
	cfg := g.customPlayers[slot]
	r := loadoutPreviewCard
	ctx.Card(r, "")
	strip := developedPlayerColorSwatch(cfg.Color)
	strip.A = 0xff
	uiFillRound(ctx.dst, Rect{X: r.X, Y: r.Y, W: r.W, H: 8}, 4, strip)
	ctx.Badge(Rect{X: r.X + 12, Y: r.Y + 16, W: 34, H: 24}, fmt.Sprintf("P%d", slot+1), uiInk900)
	kind := "HUMAN"
	if cfg.Type == 2 {
		kind = "AI BOT"
	}
	name := cfg.Name
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("Player %d", slot+1)
	}
	uiText(ctx.dst, uiFit(strings.ToUpper(name), true, uiSizeButton, r.W-120), true, uiSizeButton, uiTextDark, r.X+54, r.Y+28, uiAlignLeft)
	uiText(ctx.dst, kind, true, uiSizeHint, uiTextMutedDk, r.X+r.W-12, r.Y+28, uiAlignRight)

	stage := Rect{X: r.X + 12, Y: r.Y + 48, W: r.W - 24, H: 132}
	uiFillRound(ctx.dst, stage, 6, uiPaperShade)
	g.drawLoadoutCharacterIn(ctx.dst, cfg, uiInset(stage, 4))

	gear := migrateLegacyGearPerk(cfg)
	rows := [][2]string{
		{"GUN", weaponDisplayName(cfg.Gun)},
		{"PERK", onlinePerkName(gear.Perk)},
		{"ABILITY", abilityName(gear.Ability)},
		{"THROWABLE", throwableName(cfg.Throwable)},
	}
	for i, row := range rows {
		mid := r.Y + 196 + float64(i)*18
		uiText(ctx.dst, row[0], true, uiSizeHint, uiTextMutedDk, r.X+12, mid, uiAlignLeft)
		uiText(ctx.dst, uiFit(strings.ToUpper(row[1]), true, uiSizeHint, r.W-110), true, uiSizeHint, uiTextDark, r.X+96, mid, uiAlignLeft)
	}
}

func (g *Game) drawLoadoutInfo(ctx *uiCtx, slot int, info *loadoutChoice) {
	if !ctx.drawing() {
		return
	}
	r := loadoutInfoRect
	if info == nil {
		title, body := ctx.FocusInfo()
		ctx.InfoPanel(r, title, body)
		return
	}
	if info.gun == 0 {
		ctx.InfoPanel(r, info.infoTitle, info.infoBody)
		return
	}
	ctx.InfoPanel(r, info.infoTitle, "")
	def, ok := WeaponByNumber(info.gun)
	if !ok {
		return
	}
	maxRounds, maxRate, maxPush := loadoutWeaponMaxima()
	rounds := fmt.Sprint(def.Bullets)
	roundsV := float64(def.Bullets) / maxRounds
	if def.Bullets >= 9999 {
		rounds, roundsV = "INF", 1
	}
	rate := weaponShotsPerSecond(def)
	stats := []struct {
		label, value string
		v            float64
	}{
		{"ROUNDS", rounds, roundsV},
		{"FIRE RATE", fmt.Sprintf("%.1f/s", rate), rate / maxRate},
		{"PUSH", fmt.Sprintf("%.0f", def.Firepower), def.Firepower / maxPush},
	}
	for i, s := range stats {
		mid := r.Y + 54 + float64(i)*22
		uiText(ctx.dst, s.label, true, uiSizeHint, uiTextMuted, r.X+18, mid, uiAlignLeft)
		bar := Rect{X: r.X + 92, Y: mid - 4, W: r.W - 160, H: 8}
		uiFillRound(ctx.dst, bar, 4, uiInk500)
		v := math.Max(0.04, math.Min(1, s.v))
		uiFillRound(ctx.dst, Rect{X: bar.X, Y: bar.Y, W: bar.W * v, H: bar.H}, 4, uiOrange)
		uiText(ctx.dst, s.value, true, uiSizeHint, uiTextLight, r.X+r.W-14, mid, uiAlignRight)
	}
	extra := weaponFireModeLabel(info.gun)
	if def.Recoil > 0 {
		extra += fmt.Sprintf("  /  RECOIL %.1f", def.Recoil)
	}
	uiText(ctx.dst, extra, true, uiSizeHint, uiTextMuted, r.X+18, r.Y+54+3*22, uiAlignLeft)
}

// ---------------------------------------------------------------------------
// Art

// drawLoadoutCharacterIn renders the character in the gameplay aiming pose
// with the chosen gun in hand (weapon_pose.go / developed_weapon_fit.go grip
// data), worn gear included, fitted into r.
func (g *Game) drawLoadoutCharacterIn(dst *ebiten.Image, cfg CustomPlayerConfig, r Rect) {
	if dst == nil || g.assets == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	if g.loadoutPreviewImg == nil {
		g.loadoutPreviewImg = ebiten.NewImage(240, 170)
	}
	img := g.loadoutPreviewImg
	img.Clear()
	p := NewPlayer(1, g.arena)
	p.X, p.Y = 84, 148
	p.Facing = 1
	p.PlayerScale = 1
	p.PlayerColor = cfg.Color
	p.ShirtNumber = cfg.Shirt
	p.HatNumber = cfg.Hat
	p.EyeNumber = cfg.Eyes
	p.AbilityNumber = migrateLegacyGearPerk(cfg).Ability
	p.HeadFrame = 0
	p.LegFrame1, p.LegFrame2 = 0, 0
	p.Alpha = 1
	p.ShieldAlpha = 0
	p.JetpackAlpha = 0
	p.Weapon = NewWeapon(cfg.Gun)
	p.Weapon.Alpha = 1
	pose := poseForWeapon(p.Weapon.Def.Number)
	p.VisualBodyY, p.VisualEyesY = -60, -46
	p.VisualHand1X, p.VisualHand1Y = 0, -35
	p.VisualHand2X, p.VisualHand2Y = 0, -25
	p.VisualHand1ChildX, p.VisualHand1ChildY = pose.ShootX, pose.ShootY
	p.VisualHand2ChildX, p.VisualHand2ChildY = pose.HandX, pose.HandY
	p.VisualGunX, p.VisualGunY, p.VisualGunRotation = pose.ShootX, pose.ShootY, 0
	g.drawPlayer(img, p)

	src := image.Rect(42, 36, 202, 164)
	scale := math.Min(r.W/float64(src.Dx()), r.H/float64(src.Dy()))
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(r.X+(r.W-float64(src.Dx())*scale)/2, r.Y+(r.H-float64(src.Dy())*scale)/2)
	dst.DrawImage(img.SubImage(src).(*ebiten.Image), op)
}

// loadoutCosmeticThumb is the picker thumbnail of a hat / shirt / face. GM2
// frames keep their large Flash registration canvas, which made the art tiny
// in a tile, so they are reloaded once as UI-only tight crops of the same
// original PNG (loadTightOriginalPNGIn); GM1 frames are already tight.
func (g *Game) loadoutCosmeticThumb(editor, n int) *SourceRaster {
	var dir string
	source, gm2 := 0, false
	switch editor {
	case customEditHat1, customEditHat2:
		source, gm2 = gm2HatSourceNumber(n)
		dir = "DefineSprite_279"
	case customEditShirt:
		source, gm2 = gm2ShirtSourceNumber(n)
		dir = "DefineSprite_201"
	case customEditEyes:
		source, gm2 = gm2EyeSourceNumber(n)
		dir = "DefineSprite_291"
	}
	if gm2 {
		key := [2]int{editor, n}
		if editor == customEditHat2 {
			key[0] = customEditHat1
		}
		if g.loadoutThumbs == nil {
			g.loadoutThumbs = map[[2]int]*SourceRaster{}
		}
		r, ok := g.loadoutThumbs[key]
		if !ok {
			r = loadTightOriginalPNGIn("gm2", "sprites", dir, fmt.Sprintf("%d.png", source))
			g.loadoutThumbs[key] = r
		}
		if r != nil {
			return r
		}
	}
	return g.setupSelectionRaster(editor, n)
}

// drawLoadoutPerkArt uses original sprites where the game has them (minigun,
// grenade, crate) and simple vector glyphs otherwise.
func (g *Game) drawLoadoutPerkArt(dst *ebiten.Image, n int, r Rect) {
	if g.assets != nil {
		switch n {
		case 4:
			if raster := g.assets.WeaponFrame(55, 0); raster != nil {
				drawSetupThumbFit(dst, raster, r, 1)
				return
			}
		case 8:
			g.drawThrowableArt(dst, ThrowableMk2, Rect{X: r.X + r.W/2 - r.W*0.32, Y: r.Y + 4, W: r.W * 0.3, H: r.H - 8})
			g.drawThrowableArt(dst, ThrowableMk2, Rect{X: r.X + r.W/2 + r.W*0.02, Y: r.Y + 4, W: r.W * 0.3, H: r.H - 8})
			return
		case 9:
			if g.assets.Crate != nil {
				drawSetupThumbFit(dst, g.assets.Crate, uiInset(r, 6), 1)
				return
			}
		}
	}
	cx, cy := float32(r.X+r.W/2), float32(r.Y+r.H/2)
	k := float32(math.Min(r.W, r.H) / 60)
	ink := color.NRGBA{R: 0x26, G: 0x2b, B: 0x33, A: 0xff}
	line := func(x0, y0, x1, y1 float32, clr color.NRGBA) {
		vector.StrokeLine(dst, cx+x0*k, cy+y0*k, cx+x1*k, cy+y1*k, 5*k, clr, true)
	}
	switch n {
	case 1:
		grey := color.NRGBA{R: 0x8a, G: 0x8f, B: 0x96, A: 0xff}
		vector.StrokeCircle(dst, cx, cy, 20*k, 5*k, grey, true)
		line(-14, 14, 14, -14, grey)
	case 2:
		green := color.NRGBA{R: 0x33, G: 0xb5, B: 0x22, A: 0xff}
		for _, dx := range []float32{-14, 4} {
			line(dx, -16, dx+14, 0, green)
			line(dx+14, 0, dx, 16, green)
		}
	case 3:
		pink := color.NRGBA{R: 0xe8, G: 0x2e, B: 0x8a, A: 0xff}
		for _, dy := range []float32{-16, -2, 12} {
			line(-14, dy+10, 0, dy, pink)
			line(0, dy, 14, dy+10, pink)
		}
	case 5:
		amber := color.NRGBA{R: 0xe0, G: 0x9a, B: 0x1a, A: 0xff}
		vector.StrokeCircle(dst, cx, cy, 16*k, 4*k, amber, true)
		line(-24, 0, -8, 0, amber)
		line(8, 0, 24, 0, amber)
		line(0, -24, 0, -8, amber)
		line(0, 8, 0, 24, amber)
	case 6:
		blue := color.NRGBA{R: 0x1f, G: 0x6a, B: 0xe0, A: 0xff}
		var path vector.Path
		path.MoveTo(cx, cy-22*k)
		path.LineTo(cx+18*k, cy-14*k)
		path.LineTo(cx+14*k, cy+8*k)
		path.LineTo(cx, cy+22*k)
		path.LineTo(cx-14*k, cy+8*k)
		path.LineTo(cx-18*k, cy-14*k)
		path.Close()
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(blue)
		vector.FillPath(dst, &path, nil, op)
	case 7:
		brass := color.NRGBA{R: 0xd9, G: 0x8e, B: 0x1c, A: 0xff}
		for i := -1; i <= 1; i++ {
			x := float32(i) * 12
			uiFillRound(dst, Rect{X: float64(cx + (x-4)*k), Y: float64(cy - 8*k), W: float64(8 * k), H: float64(24 * k)}, 2, brass)
			uiFillRound(dst, Rect{X: float64(cx + (x-4)*k), Y: float64(cy - 20*k), W: float64(8 * k), H: float64(14 * k)}, float64(4*k), ink)
		}
	case 9:
		// Weapon crate with a question mark.
		wood := color.NRGBA{R: 0xb0, G: 0x74, B: 0x2c, A: 0xff}
		box := Rect{X: float64(cx - 20*k), Y: float64(cy - 18*k), W: float64(40 * k), H: float64(36 * k)}
		uiFillRound(dst, box, 3, wood)
		uiStrokeRound(dst, box, 3, float64(3*k), ink)
		uiText(dst, "?", true, float64(30*k), uiPaper, float64(cx), float64(cy), uiAlignCenter)
	default:
		vector.StrokeCircle(dst, cx, cy, 18*k, 4*k, ink, true)
	}
}
