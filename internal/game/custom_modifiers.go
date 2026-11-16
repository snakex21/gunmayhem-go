package game

import (
	"math/rand"
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
