package game

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	customModifierShotgunsOnly = 1 + iota
	customModifierPistolsOnly
	customModifierExplosivesOnly
	customModifierRandomWeaponEachLife
	customModifierSameWeapon
	customModifierLowGravity
	customModifierHighGravity
	customModifierDoubleKnockback
	customModifierSuperJump
	customModifierFastMovement
	customModifierNoReload
	customModifierInfiniteAmmo
	customModifierNoCrates
	customModifierCratesEverywhere
	customModifierJetpacks
)

const (
	menuHitCustomModifiersOpen = 8200 + iota
	menuHitCustomModifiersClose
	menuHitCustomModifiersRandom
	menuHitCustomModifiersRandom1
	menuHitCustomModifiersRandom2
	menuHitCustomModifiersRandom3
	menuHitCustomModifiersBlock
	menuHitCustomModifiersScopeSeries
	menuHitCustomModifiersScopePerRound
	menuHitCustomModifiersRoundPrev
	menuHitCustomModifiersRoundNext
	menuHitCustomModifierBase = 8300
)

type customModifierSpec struct {
	ID    int
	Label string
}

var customModifierSpecs = []customModifierSpec{
	{customModifierShotgunsOnly, "SHOTGUNS ONLY"},
	{customModifierPistolsOnly, "PISTOLS ONLY"},
	{customModifierExplosivesOnly, "EXPLOSIVES ONLY"},
	{customModifierRandomWeaponEachLife, "RANDOM WEAPON / LIFE"},
	{customModifierSameWeapon, "SAME WEAPON FOR ALL"},
	{customModifierLowGravity, "LOW GRAVITY"},
	{customModifierHighGravity, "HIGH GRAVITY"},
	{customModifierDoubleKnockback, "X2 KNOCKBACK"},
	{customModifierSuperJump, "SUPER JUMP"},
	{customModifierFastMovement, "FAST MOVEMENT"},
	{customModifierNoReload, "NO RELOAD"},
	{customModifierInfiniteAmmo, "INFINITE AMMO"},
	{customModifierNoCrates, "NO CRATES"},
	{customModifierCratesEverywhere, "CRATES EVERYWHERE"},
	{customModifierJetpacks, "JETPACKS ON"},
}

var (
	customModifiersButtonRect        = Rect{X: -1090, Y: 386, W: 145, H: 42}
	customModifiersCloseRect         = Rect{X: -1190, Y: 542, W: 250, H: 42}
	customModifiersRandomRect        = Rect{X: -1760, Y: 378, W: 240, H: 44}
	customModifiersScopeSeriesRect   = Rect{X: -1760, Y: 458, W: 235, H: 40}
	customModifiersScopePerRoundRect = Rect{X: -1510, Y: 458, W: 235, H: 40}
	customModifiersRoundPrevRect     = Rect{X: -1240, Y: 508, W: 48, H: 30}
	customModifiersRoundNextRect     = Rect{X: -998, Y: 508, W: 48, H: 30}
	customModifiersCountRects        = [3]Rect{
		{X: -1500, Y: 378, W: 58, H: 44},
		{X: -1432, Y: 378, W: 58, H: 44},
		{X: -1364, Y: 378, W: 58, H: 44},
	}
)

func customModifierRect(index int) Rect {
	col := index % 3
	row := index / 3
	return Rect{X: -1760 + float64(col)*290, Y: 98 + float64(row)*55, W: 240, H: 44}
}

func customModifiersConflict(a, b int) bool {
	if a == b {
		return false
	}
	weaponGroup := func(v int) bool {
		return v == customModifierShotgunsOnly || v == customModifierPistolsOnly || v == customModifierExplosivesOnly
	}
	if weaponGroup(a) && weaponGroup(b) {
		return true
	}
	if (a == customModifierLowGravity && b == customModifierHighGravity) || (a == customModifierHighGravity && b == customModifierLowGravity) {
		return true
	}
	if (a == customModifierNoReload && b == customModifierInfiniteAmmo) || (a == customModifierInfiniteAmmo && b == customModifierNoReload) {
		return true
	}
	if (a == customModifierNoCrates && b == customModifierCratesEverywhere) || (a == customModifierCratesEverywhere && b == customModifierNoCrates) {
		return true
	}
	if (a == customModifierRandomWeaponEachLife && b == customModifierSameWeapon) || (a == customModifierSameWeapon && b == customModifierRandomWeaponEachLife) {
		return true
	}
	if a == customModifierExplosivesOnly || b == customModifierExplosivesOnly {
		other := b
		if a != customModifierExplosivesOnly {
			other = a
		}
		if other == customModifierRandomWeaponEachLife || other == customModifierSameWeapon || other == customModifierNoReload || other == customModifierInfiniteAmmo {
			return true
		}
	}
	return false
}

func clampCustomRuleEditRound(round, rounds int) int {
	rounds = clampCustomSeriesRounds(rounds)
	if round < 1 {
		return 1
	}
	if round > rounds {
		return rounds
	}
	return round
}

func (g *Game) customModifierEditRound() int {
	if g == nil {
		return 1
	}
	g.customRuleEditRound = clampCustomRuleEditRound(g.customRuleEditRound, g.customSeriesRounds)
	return g.customRuleEditRound
}

func (g *Game) customModifierEnabledForEdit(id int) bool {
	if g == nil || id < 1 || id > customModifierJetpacks {
		return false
	}
	if !g.customPerRoundRules {
		return g.customRuleEnabled[id]
	}
	return g.customRoundRuleEnabled[g.customModifierEditRound()-1][id]
}

func (g *Game) setCustomModifierEnabledForEdit(id int, enabled bool) {
	if g == nil || id < 1 || id > customModifierJetpacks {
		return
	}
	if !g.customPerRoundRules {
		if g.customRuleEnabled == nil {
			g.customRuleEnabled = make(map[int]bool)
		}
		g.customRuleEnabled[id] = enabled
		return
	}
	g.customRoundRuleEnabled[g.customModifierEditRound()-1][id] = enabled
}

func (g *Game) customRandomRulesForEdit() bool {
	if g == nil {
		return false
	}
	if !g.customPerRoundRules {
		return g.customRandomRules
	}
	return g.customRoundRandomRules[g.customModifierEditRound()-1]
}

func (g *Game) customRandomCountForEdit() int {
	if g == nil {
		return 1
	}
	count := g.customRandomCount
	if g.customPerRoundRules {
		count = g.customRoundRandomCount[g.customModifierEditRound()-1]
	}
	if count < 1 || count > 3 {
		count = 1
	}
	return count
}

func (g *Game) setCustomRandomRulesForEdit(enabled bool) {
	if g == nil {
		return
	}
	if !g.customPerRoundRules {
		g.customRandomRules = enabled
		return
	}
	g.customRoundRandomRules[g.customModifierEditRound()-1] = enabled
}

func (g *Game) setCustomRandomCountForEdit(count int) {
	if g == nil {
		return
	}
	if count < 1 {
		count = 1
	}
	if count > 3 {
		count = 3
	}
	if !g.customPerRoundRules {
		g.customRandomCount = count
		return
	}
	g.customRoundRandomCount[g.customModifierEditRound()-1] = count
}

func (g *Game) enableCustomPerRoundRules() {
	if g == nil || g.customPerRoundRules {
		return
	}
	g.customPerRoundRules = true
	g.customRuleEditRound = clampCustomRuleEditRound(g.customRuleEditRound, g.customSeriesRounds)
	// Initialize the per-round profiles only once. Switching temporarily back
	// to SAME FOR WHOLE SERIES must not erase a carefully prepared round list.
	if g.customRoundRandomCount[0] != 0 {
		return
	}
	count := g.customRandomCount
	if count < 1 || count > 3 {
		count = 1
	}
	for round := 0; round < customSeriesMaxRounds; round++ {
		for id := 1; id <= customModifierJetpacks; id++ {
			g.customRoundRuleEnabled[round][id] = g.customRuleEnabled[id]
		}
		g.customRoundRandomRules[round] = g.customRandomRules
		g.customRoundRandomCount[round] = count
	}
}

func customModifierHint(id int) string {
	switch id {
	case customModifierShotgunsOnly:
		return "Only shotguns appear in the weapon pool."
	case customModifierPistolsOnly:
		return "Only pistols appear in the weapon pool."
	case customModifierExplosivesOnly:
		return "Explosive weapons and throwables dominate the round."
	case customModifierRandomWeaponEachLife:
		return "Every respawn gives the player a new random weapon."
	case customModifierSameWeapon:
		return "All players use the same randomly chosen weapon."
	case customModifierLowGravity:
		return "Lower gravity makes jumps and knockback floatier."
	case customModifierHighGravity:
		return "Higher gravity makes movement heavier and faster to fall."
	case customModifierDoubleKnockback:
		return "Weapon hits push players roughly twice as hard."
	case customModifierSuperJump:
		return "Players jump much higher than normal."
	case customModifierFastMovement:
		return "Player movement speed is increased."
	case customModifierNoReload:
		return "Weapons refill instead of performing a normal reload."
	case customModifierInfiniteAmmo:
		return "Weapons do not run out of ammunition."
	case customModifierNoCrates:
		return "Weapon crates do not spawn."
	case customModifierCratesEverywhere:
		return "Crates spawn much more frequently."
	case customModifierJetpacks:
		return "All players have unlimited jetpack access."
	default:
		return "Choose any compatible modifier for this round."
	}
}

func (g *Game) customModifierEditingAllowed() bool {
	if g == nil {
		return false
	}
	// ONLINE before a session exists is still the host's pre-match setup, so it
	// must remain fully editable. Once connected as a client, only the host may
	// change match rules.
	return !g.customOnlineMode || g.netplay == nil || g.netplay.mode == netplayHost
}

func (g *Game) customModifierHitAt(x, y float64) int {
	if g.customChaosOpen {
		return g.customChaosHitAt(x, y)
	}
	if !g.customRulesOpen {
		if g.customModifierEditingAllowed() && customModifiersButtonRect.Contains(x, y) {
			return menuHitCustomModifiersOpen
		}
		return menuHitNone
	}
	for i, spec := range customModifierSpecs {
		if customModifierRect(i).Contains(x, y) {
			return menuHitCustomModifierBase + spec.ID
		}
	}
	if customModifiersRandomRect.Contains(x, y) {
		return menuHitCustomModifiersRandom
	}
	for i, r := range customModifiersCountRects {
		if r.Contains(x, y) {
			return menuHitCustomModifiersRandom1 + i
		}
	}
	if customModifiersScopeSeriesRect.Contains(x, y) {
		return menuHitCustomModifiersScopeSeries
	}
	if customModifiersScopePerRoundRect.Contains(x, y) {
		return menuHitCustomModifiersScopePerRound
	}
	if customChaosSetupRect.Contains(x, y) {
		return menuHitChaosSetup
	}
	if g.customPerRoundRules {
		if customModifiersRoundPrevRect.Contains(x, y) {
			return menuHitCustomModifiersRoundPrev
		}
		if customModifiersRoundNextRect.Contains(x, y) {
			return menuHitCustomModifiersRoundNext
		}
	}
	if customModifiersCloseRect.Contains(x, y) {
		return menuHitCustomModifiersClose
	}
	return menuHitCustomModifiersBlock
}

func (g *Game) activateCustomModifierHit(hit int) bool {
	if g.activateCustomChaosHit(hit) {
		return true
	}
	switch hit {
	case menuHitCustomModifiersOpen:
		g.customRulesOpen = true
		g.customLivesFocus = false
		g.customRuleEditRound = clampCustomRuleEditRound(g.customRuleEditRound, g.customSeriesRounds)
		return true
	case menuHitCustomModifiersClose:
		g.customChaosOpen = false
		g.customRulesOpen = false
		return true
	case menuHitCustomModifiersRandom:
		g.setCustomRandomRulesForEdit(!g.customRandomRulesForEdit())
		return true
	case menuHitCustomModifiersRandom1, menuHitCustomModifiersRandom2, menuHitCustomModifiersRandom3:
		g.setCustomRandomCountForEdit(hit - menuHitCustomModifiersRandom1 + 1)
		return true
	case menuHitCustomModifiersScopeSeries:
		g.customPerRoundRules = false
		return true
	case menuHitCustomModifiersScopePerRound:
		g.enableCustomPerRoundRules()
		return true
	case menuHitCustomModifiersRoundPrev:
		if g.customPerRoundRules {
			g.customRuleEditRound = clampCustomRuleEditRound(g.customModifierEditRound()-1, g.customSeriesRounds)
		}
		return true
	case menuHitCustomModifiersRoundNext:
		if g.customPerRoundRules {
			g.customRuleEditRound = clampCustomRuleEditRound(g.customModifierEditRound()+1, g.customSeriesRounds)
		}
		return true
	case menuHitCustomModifiersBlock:
		return true
	}
	if hit >= menuHitCustomModifierBase+1 && hit <= menuHitCustomModifierBase+len(customModifierSpecs) {
		id := hit - menuHitCustomModifierBase
		enable := !g.customModifierEnabledForEdit(id)
		if enable {
			for other := 1; other <= customModifierJetpacks; other++ {
				if g.customModifierEnabledForEdit(other) && customModifiersConflict(id, other) {
					g.setCustomModifierEnabledForEdit(other, false)
				}
			}
		}
		g.setCustomModifierEnabledForEdit(id, enable)
		return true
	}
	return false
}

func (g *Game) prepareCustomMatchModifiers() {
	g.matchModifiers = make(map[int]bool)
	g.customMatchActive = true

	randomRules := g.customRandomRules
	count := g.customRandomCount
	profileRound := g.customSeriesRound
	if profileRound < 1 {
		profileRound = 1
	}
	profileRound = clampCustomRuleEditRound(profileRound, g.customSeriesRounds)

	if g.customPerRoundRules {
		randomRules = g.customRoundRandomRules[profileRound-1]
		count = g.customRoundRandomCount[profileRound-1]
	}

	if !randomRules {
		for id := 1; id <= customModifierJetpacks; id++ {
			enabled := g.customRuleEnabled[id]
			if g.customPerRoundRules {
				enabled = g.customRoundRuleEnabled[profileRound-1][id]
			}
			if enabled {
				g.matchModifiers[id] = true
			}
		}
		return
	}
	if count < 1 {
		count = 1
	}
	if count > 3 {
		count = 3
	}
	perm := rand.Perm(len(customModifierSpecs))
	for _, index := range perm {
		id := customModifierSpecs[index].ID
		compatible := true
		for chosen := range g.matchModifiers {
			if customModifiersConflict(id, chosen) {
				compatible = false
				break
			}
		}
		if compatible {
			g.matchModifiers[id] = true
			if len(g.matchModifiers) >= count {
				break
			}
		}
	}
}

func (g *Game) customModifierActive(id int) bool {
	if g == nil || !g.customMatchActive {
		return false
	}
	return (g.matchModifiers != nil && g.matchModifiers[id]) || g.chaosModifierActive(id)
}

func (g *Game) modifierWeaponPool() []int {
	if g.customModifierActive(customModifierShotgunsOnly) {
		out := make([]int, 0, 32)
		for _, n := range developedCrateWeapons() {
			if def, ok := WeaponByNumber(n); ok && def.Shotgun > 0 {
				out = append(out, n)
			}
		}
		return out
	}
	if g.customModifierActive(customModifierPistolsOnly) {
		out := []int{1, 2, 3, 4, 5, 6}
		for _, n := range availableCustomWeaponNumbers() {
			if spec, ok := developedWeaponSpec(n); ok && spec.Category == DevelopedCategoryPistol {
				out = append(out, n)
			}
		}
		return out
	}
	return developedCrateWeapons()
}

func randomWeaponFromPool(pool []int) int {
	if len(pool) == 0 {
		return 1
	}
	return pool[rand.Intn(len(pool))]
}

func (g *Game) applyCustomModifierSetup() {
	if !g.customMatchActive {
		return
	}
	pool := g.modifierWeaponPool()
	if !g.customModifierActive(customModifierExplosivesOnly) && (g.customModifierActive(customModifierShotgunsOnly) || g.customModifierActive(customModifierPistolsOnly)) {
		g.crateWeapons = append([]int(nil), pool...)
	}
	if g.customModifierActive(customModifierSameWeapon) && !g.customModifierActive(customModifierExplosivesOnly) {
		weapon := randomWeaponFromPool(pool)
		g.crateWeapons = []int{weapon}
		for _, p := range g.players {
			p.DefaultWeapon = weapon
		}
	} else if !g.customModifierActive(customModifierExplosivesOnly) && (g.customModifierActive(customModifierShotgunsOnly) || g.customModifierActive(customModifierPistolsOnly) || g.customModifierActive(customModifierRandomWeaponEachLife)) {
		for _, p := range g.players {
			p.DefaultWeapon = randomWeaponFromPool(pool)
		}
	}
	for _, p := range g.players {
		p.RuleMoveMultiplier = 1
		p.RuleJumpMultiplier = 1
		p.RuleInfiniteJetpack = false
		if g.customModifierActive(customModifierFastMovement) {
			p.RuleMoveMultiplier = 1.6
		}
		if g.customModifierActive(customModifierSuperJump) {
			p.RuleJumpMultiplier = 1.45
		}
		if g.customModifierActive(customModifierJetpacks) {
			p.RuleInfiniteJetpack = true
		}
	}
}

func (g *Game) applyCustomRespawnWeapon(p *Player) {
	if p == nil || !p.Active || !g.customModifierActive(customModifierRandomWeaponEachLife) || g.customModifierActive(customModifierExplosivesOnly) {
		return
	}
	weapon := randomWeaponFromPool(g.modifierWeaponPool())
	p.DefaultWeapon = weapon
	p.EquipWeapon(weapon)
}

func (g *Game) customRuntimeArena() Map {
	arena := g.arena
	if g.customModifierActive(customModifierLowGravity) {
		arena.Gravity *= 0.55
	} else if g.customModifierActive(customModifierHighGravity) {
		arena.Gravity *= 1.55
	}
	if g.customChaosEnabled {
		switch g.chaosCurrentEffect {
		case chaosEffectZeroGravity:
			arena.Gravity = 0.08
		case chaosEffectIceFloor:
			arena.LowFriction = true
		}
	}
	return arena
}

func (g *Game) customCratesEnabled() bool {
	if g.customModifierActive(customModifierNoCrates) || g.customModifierActive(customModifierExplosivesOnly) {
		return false
	}
	if g.customModifierActive(customModifierCratesEverywhere) {
		return true
	}
	return g.CrateON
}

func (g *Game) customCrateInterval() int {
	if g.customModifierActive(customModifierCratesEverywhere) {
		return 100
	}
	return 400
}

func drawModifierButton(screen *ebiten.Image, r Rect, label string, selected bool) {
	fill := color.NRGBA{R: 0x2e, G: 0x31, B: 0x36, A: 0xff}
	if selected {
		fill = color.NRGBA{R: 0xff, G: 0x66, B: 0x00, A: 0xff}
	}
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, color.NRGBA{R: 0x55, G: 0x58, B: 0x60, A: 0xff})
	ebitenutil.DrawRect(screen, r.X+2, r.Y+2, r.W-4, r.H-4, fill)
	drawSourceMenuText(screen, label, menuFontCondensedExtraBold, 12.5, color.White, r.X+8, r.Y+10)
}

func (g *Game) drawCustomModifierButton(screen *ebiten.Image) {
	if g.customOnlineMode {
		return
	}
	r := customModifiersButtonRect
	r.X += g.customMenuX
	selected := 0
	for _, enabled := range g.customRuleEnabled {
		if enabled {
			selected++
		}
	}
	label := "MODIFIERS"
	if g.customRandomRules {
		label = "RANDOM RULES"
	} else if selected > 0 {
		label = "MODIFIERS ON"
	}
	drawModifierButton(screen, r, label, selected > 0 || g.customRandomRules)
}

func (g *Game) drawCustomModifiersOverlay(screen *ebiten.Image) {
	if !g.customRulesOpen {
		return
	}
	if g.customChaosOpen {
		g.drawCustomChaosOverlay(screen)
		return
	}

	// Full-screen developed overlay. The old panel started its first row on top
	// of the help text, which made the explanation practically unreadable.
	ebitenutil.DrawRect(screen, 0, 0, 900, 600, color.NRGBA{R: 0x12, G: 0x15, B: 0x19, A: 0xff})
	drawSourceMenuText(screen, "CUSTOM GAME MODIFIERS", menuFontCondensedExtraBold, 30, color.White, 24, 18)
	drawSourceMenuText(screen, "Pick modifiers manually. Conflicting choices are switched off automatically.", menuFontCondensed, 14.5,
		color.NRGBA{R: 0xc9, G: 0xcd, B: 0xd3, A: 0xff}, 24, 58)

	mx, my := ebiten.CursorPosition()
	hoverHint := "Hover a modifier to see what it changes."
	for i, spec := range customModifierSpecs {
		r := customModifierRect(i)
		r.X += g.customMenuX
		drawModifierButton(screen, r, spec.Label, g.customModifierEnabledForEdit(spec.ID))
		if r.Contains(float64(mx), float64(my)) {
			hoverHint = customModifierHint(spec.ID)
		}
	}

	randomRect := customModifiersRandomRect
	randomRect.X += g.customMenuX
	drawModifierButton(screen, randomRect, "RANDOM RULES", g.customRandomRulesForEdit())
	if randomRect.Contains(float64(mx), float64(my)) {
		hoverHint = "Random Rules ignores manual picks and chooses compatible modifiers when the round starts."
	}
	count := g.customRandomCountForEdit()
	for i, rr := range customModifiersCountRects {
		rr.X += g.customMenuX
		drawModifierButton(screen, rr, string(rune('1'+i)), count == i+1)
		if rr.Contains(float64(mx), float64(my)) {
			hoverHint = "Choose how many compatible random modifiers are picked for this round."
		}
	}
	drawSourceMenuText(screen, "1 / 2 / 3 = NUMBER OF RANDOM MODIFIERS", menuFontCondensedExtraBold, 12.5,
		color.NRGBA{R: 0xd7, G: 0xda, B: 0xdf, A: 0xff}, 505, 387)
	drawSourceMenuText(screen, "They are re-rolled when each configured round starts.", menuFontCondensed, 12,
		color.NRGBA{R: 0xae, G: 0xb4, B: 0xbc, A: 0xff}, 505, 405)
	drawSourceMenuText(screen, hoverHint, menuFontCondensed, 12.5,
		color.NRGBA{R: 0xff, G: 0xb2, B: 0x47, A: 0xff}, 40, 432)

	drawSourceMenuText(screen, "RULE SCOPE", menuFontCondensedExtraBold, 12.5, color.White, 40, 444)
	seriesRect := customModifiersScopeSeriesRect
	seriesRect.X += g.customMenuX
	perRoundRect := customModifiersScopePerRoundRect
	perRoundRect.X += g.customMenuX
	drawModifierButton(screen, seriesRect, "SAME FOR WHOLE SERIES", !g.customPerRoundRules)
	drawModifierButton(screen, perRoundRect, "EACH ROUND / MAP", g.customPerRoundRules)
	chaosRect := customChaosSetupRect
	chaosRect.X += g.customMenuX
	chaosLabel := "CHAOS SETUP"
	if g.customChaosEnabled {
		chaosLabel = fmt.Sprintf("CHAOS ON - %d EVENTS", g.chaosSelectedCount())
	}
	drawModifierButton(screen, chaosRect, chaosLabel, g.customChaosEnabled)

	if g.customPerRoundRules {
		prev := customModifiersRoundPrevRect
		prev.X += g.customMenuX
		next := customModifiersRoundNextRect
		next.X += g.customMenuX
		round := g.customModifierEditRound()
		drawModifierButton(screen, prev, "<", false)
		drawModifierButton(screen, next, ">", false)
		drawSourceMenuText(screen, fmt.Sprintf("EDITING ROUND %d / %d", round, clampCustomSeriesRounds(g.customSeriesRounds)),
			menuFontCondensedExtraBold, 13.5, color.White, 620, 514)
		drawSourceMenuText(screen, "Each round/map remembers its own rules.", menuFontCondensed, 11.5,
			color.NRGBA{R: 0xaf, G: 0xb4, B: 0xbd, A: 0xff}, 40, 515)
	} else {
		drawSourceMenuText(screen, "The same modifier setup is used on every map in the series.", menuFontCondensed, 11.5,
			color.NRGBA{R: 0xaf, G: 0xb4, B: 0xbd, A: 0xff}, 40, 515)
	}

	closeRect := customModifiersCloseRect
	closeRect.X += g.customMenuX
	drawModifierButton(screen, closeRect, "BACK TO GAME SETUP", false)
}
