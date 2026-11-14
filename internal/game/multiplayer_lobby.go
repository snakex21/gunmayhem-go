package game

import (
	"fmt"
)

const (
	menuHitMultiplayerLobbySetup = 6200 + iota
	menuHitMultiplayerLobbyLeave
	menuHitMultiplayerLobbyCustomize
	menuHitMultiplayerLobbyRules
	menuHitOnlineOverlayBack
	menuHitOnlineRuleMapRotation
	menuHitOnlineRuleWeaponMode
	menuHitOnlineRuleWeaponPrev
	menuHitOnlineRuleWeaponNext
	menuHitOnlineRulePerks
	menuHitOnlineRulePowerups
	menuHitOnlineRuleBlocklist
	menuHitOnlineRulePerkBlocklist
	menuHitOnlineBlockPrevPage
	menuHitOnlineBlockNextPage
)

const (
	menuHitOnlineCharacterBase = 6400
	// The weapon blocklist indexes every weapon (200+), so it needs its own
	// range: at 6600 index 201+ collided with the perk codes at 6800.
	menuHitOnlineBlockBase   = 17000
	menuHitOnlinePerkBase    = 6800
	menuHitOnlineAbilityBase = 6850
)

const (
	onlineCharColor = iota
	onlineCharShirt
	onlineCharHat
	onlineCharFace
	onlineCharGun
	onlineCharPerk
	onlineCharAbility
	onlineCharTeam
	onlineCharFieldCount
)

var (
	multiplayerLobbyLeaveRect     = Rect{X: 45, Y: 526, W: 180, H: 48}
	multiplayerLobbyCustomizeRect = Rect{X: 235, Y: 526, W: 190, H: 48}
	multiplayerLobbyRulesRect     = Rect{X: 435, Y: 526, W: 180, H: 48}
	multiplayerLobbySetupRect     = Rect{X: 625, Y: 526, W: 230, H: 48}
	onlineOverlayBackRect         = Rect{X: 665, Y: 520, W: 190, H: 48}
)

func onlineCharacterHit(field, delta int) int {
	bit := 0
	if delta > 0 {
		bit = 1
	}
	return menuHitOnlineCharacterBase + field*2 + bit
}

func decodeOnlineCharacterHit(hit int) (field, delta int, ok bool) {
	v := hit - menuHitOnlineCharacterBase
	if v < 0 || v >= onlineCharFieldCount*2 {
		return 0, 0, false
	}
	field = v / 2
	delta = -1
	if v%2 == 1 {
		delta = 1
	}
	return field, delta, true
}

func decodeOnlineBlockWeaponHit(hit int) (index int, ok bool) {
	index = hit - menuHitOnlineBlockBase
	weapons := onlineAllWeaponNumbers()
	return index, index >= 0 && index < len(weapons)
}

func (g *Game) multiplayerLobbyIsHost() bool {
	return g != nil && g.netplay != nil && g.netplay.mode == netplayHost
}

func (g *Game) multiplayerLobbyConnected() bool {
	return g != nil && g.onlineConnectedPlayerCount() >= 2
}

func (g *Game) multiplayerLobbyHitAt(x, y float64) int {
	if multiplayerLobbyLeaveRect.Contains(x, y) {
		return menuHitMultiplayerLobbyLeave
	}
	if multiplayerLobbyCustomizeRect.Contains(x, y) && g.onlineLocalPlayerID() > 0 {
		return menuHitMultiplayerLobbyCustomize
	}
	if g.multiplayerLobbyIsHost() && multiplayerLobbyRulesRect.Contains(x, y) {
		return menuHitMultiplayerLobbyRules
	}
	if g.multiplayerLobbyIsHost() && multiplayerLobbySetupRect.Contains(x, y) {
		return menuHitMultiplayerLobbySetup
	}
	return menuHitNone
}

func (g *Game) leaveMultiplayerLobby() {
	_ = g.CloseNetplay()
	g.onlineMatchSetup = false
	g.onlineLobbyCustomize = false
	g.onlineRulesOpen = false
	g.onlineBlocklistOpen = false
	g.onlinePerkBlocklistOpen = false
	g.onlineVisualPicker = onlineVisualNone
	g.onlineVisualPage = 0
	g.multiplayerFocus = false
	g.multiplayerNameFocus = false
	g.multiplayerServers = nil
	g.multiplayerServerScroll = 0
	g.multiplayerDiscoveryCh = nil
	g.multiplayerDiscoveryDone = false
	g.customOnlineMode = true
	g.customNetworkOpen = true
	g.customHostSetupOpen = false
	g.customPage = customPageGame
	g.customMenuX = customPageTargetX(customPageGame)
	g.multiplayerMessage = "SESSION CLOSED - SETTINGS KEPT"
	g.beginMultiplayerServerBrowser()
	g.beginFade(screenCustomGame, fadePurposeScreen)
}

// activateOnlineLobbyHit runs one online lobby / rules overlay action. Each
// code only acts while its overlay is open, exactly as the click handler did;
// the Mayhem UI rules modals (online_lobby_menu.go) call it with the same codes.
func (g *Game) activateOnlineLobbyHit(activated int) {
	if field, ok := decodeOnlineCharacterVisualFieldHit(activated); ok && g.onlineLobbyCustomize && g.onlineVisualPicker == onlineVisualNone {
		g.openOnlineCharacterVisualPicker(field)
		return
	}
	if index, ok := decodeOnlineVisualChoiceHit(activated); ok && g.onlineVisualPicker != onlineVisualNone {
		g.applyOnlineVisualChoice(index)
		return
	}
	if field, delta, decoded := decodeOnlineCharacterHit(activated); decoded && g.onlineLobbyCustomize {
		g.adjustOnlineCharacter(field, delta)
		g.sendOnlineLocalConfigIfChanged()
		return
	}
	if activated >= menuHitOnlinePerkBase && activated < menuHitOnlinePerkBase+onlinePerkTableSize && g.onlinePerkBlocklistOpen {
		perk := activated - menuHitOnlinePerkBase
		if perk >= 1 && perk < len(g.onlineBlockedPerks) {
			g.onlineBlockedPerks[perk] = !g.onlineBlockedPerks[perk]
		}
		return
	}
	if activated >= menuHitOnlineAbilityBase && activated < menuHitOnlineAbilityBase+abilityCount && g.onlinePerkBlocklistOpen {
		ability := activated - menuHitOnlineAbilityBase
		if ability >= 1 && ability < len(g.onlineBlockedAbilities) {
			g.onlineBlockedAbilities[ability] = !g.onlineBlockedAbilities[ability]
		}
		return
	}
	if index, decoded := decodeOnlineBlockWeaponHit(activated); decoded && g.onlineBlocklistOpen {
		weapons := onlineAllWeaponNumbers()
		n := weapons[index]
		if n > 0 && n < len(g.onlineBlockedWeapons) {
			g.onlineBlockedWeapons[n] = !g.onlineBlockedWeapons[n]
		}
		return
	}
	if activated >= menuHitOnlineRuleMapChoiceBase && activated < menuHitOnlineRuleMapChoiceBase+4 {
		g.onlineMapRotation = activated - menuHitOnlineRuleMapChoiceBase
		return
	}
	if activated >= menuHitOnlineRuleWeaponChoiceBase && activated < menuHitOnlineRuleWeaponChoiceBase+3 {
		g.onlineWeaponMode = activated - menuHitOnlineRuleWeaponChoiceBase
		return
	}

	switch activated {
	case menuHitOnlineVisualPrevPage:
		if g.onlineVisualPage > 0 {
			g.onlineVisualPage--
		}
	case menuHitOnlineVisualNextPage:
		if g.onlineVisualPage < g.onlineVisualMaxPage() {
			g.onlineVisualPage++
		}
	case menuHitOnlineRuleOnlyWeaponPicker:
		g.onlineVisualPicker = onlineVisualOnlyWeapon
		g.onlineVisualPage = 0
	case menuHitOnlineRuleFixedMapPicker:
		g.onlineVisualPicker = onlineVisualFixedMap
		g.onlineVisualPage = 0
	case menuHitMultiplayerLobbyLeave:
		g.leaveMultiplayerLobby()
	case menuHitMultiplayerLobbyCustomize:
		g.onlineLobbyCustomize = true
	case menuHitMultiplayerLobbyRules:
		if g.multiplayerLobbyIsHost() {
			g.onlineRulesOpen = true
		}
	case menuHitOnlineOverlayBack:
		if g.onlineVisualPicker != onlineVisualNone {
			g.closeOnlineVisualPicker()
		} else if g.onlinePerkBlocklistOpen {
			g.onlinePerkBlocklistOpen = false
			g.onlineRulesOpen = true
		} else if g.onlineBlocklistOpen {
			g.onlineBlocklistOpen = false
			g.onlineRulesOpen = true
		} else {
			g.onlineLobbyCustomize = false
			g.onlineRulesOpen = false
		}
	case menuHitOnlineRuleMapRotation:
		g.onlineMapRotation = (g.onlineMapRotation + 1) % 4
	case menuHitOnlineRuleWeaponMode:
		g.onlineWeaponMode = (g.onlineWeaponMode + 1) % 3
	case menuHitOnlineRuleWeaponPrev:
		g.stepOnlineOnlyWeapon(-1)
	case menuHitOnlineRuleWeaponNext:
		g.stepOnlineOnlyWeapon(1)
	case menuHitOnlineRulePerks:
		g.onlinePerksEnabled = !g.onlinePerksEnabled
	case menuHitOnlineRulePowerups:
		g.onlinePowerupsEnabled = !g.onlinePowerupsEnabled
	case menuHitOnlineRuleBlocklist:
		g.onlineBlocklistOpen = true
		g.onlineBlocklistPage = 0
	case menuHitOnlineRulePerkBlocklist:
		g.onlinePerkBlocklistOpen = true
	case menuHitOnlineBlockPrevPage:
		if g.onlineBlocklistPage > 0 {
			g.onlineBlocklistPage--
		}
	case menuHitOnlineBlockNextPage:
		if max := g.onlineBlocklistMaxPage(); g.onlineBlocklistPage < max {
			g.onlineBlocklistPage++
		}
	case menuHitMultiplayerLobbySetup:
		if !g.multiplayerLobbyIsHost() {
			return
		}
		// The host may edit the full Custom Game setup even while alone in the
		// lobby. Starting remains blocked until there are at least two active
		// participants, but configuration must never depend on that.
		g.onlineMatchSetup = true
		g.setCustomV2Page(customPageGame)
		g.beginFade(screenCustomGame, fadePurposeScreen)
	}
}

func wrapOnlineValue(v, minV, maxV, delta int) int {
	if maxV < minV {
		return minV
	}
	v += delta
	if v < minV {
		return maxV
	}
	if v > maxV {
		return minV
	}
	return v
}

func (g *Game) adjustOnlineCharacter(field, delta int) {
	id := g.onlineLocalPlayerID()
	if id < 1 || id > 4 {
		return
	}
	cfg := &g.customPlayers[id-1]
	cfg.Type = 1
	switch field {
	case onlineCharColor:
		cfg.Color = wrapOnlineValue(cfg.Color, 1, maxDevelopedColorValue(), delta)
	case onlineCharShirt:
		cfg.Shirt = wrapOnlineValue(cfg.Shirt, 1, maxDevelopedShirtValue(), delta)
	case onlineCharHat:
		cfg.Hat = wrapOnlineValue(cfg.Hat, 1, maxDevelopedHatValue(), delta)
	case onlineCharFace:
		cfg.Eyes = wrapOnlineValue(cfg.Eyes, 1, maxDevelopedEyeValue(), delta)
	case onlineCharGun:
		cfg.Gun = stepStarterHandgun(cfg.Gun, delta)
	case onlineCharPerk:
		cfg.Perk = stepSelectablePerk(cfg.Perk, delta)
	case onlineCharAbility:
		cfg.Ability = stepSelectableAbility(cfg.Ability, delta)
	case onlineCharTeam:
		cfg.Team = wrapOnlineValue(cfg.Team, 1, 2, delta)
	}
}

func (g *Game) onlineCharacterHitAt(x, y float64) int {
	return g.onlineCharacterVisualHitAt(x, y)
}

func (g *Game) stepOnlineOnlyWeapon(delta int) {
	weapons := onlineAllWeaponNumbers()
	if len(weapons) == 0 {
		return
	}
	idx := 0
	for i, n := range weapons {
		if n == g.onlineOnlyWeapon {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(weapons)) % len(weapons)
	g.onlineOnlyWeapon = weapons[idx]
}

func (g *Game) onlineRulesHitAt(x, y float64) int {
	if onlineOverlayBackRect.Contains(x, y) {
		return menuHitOnlineOverlayBack
	}
	for mode := 0; mode < 4; mode++ {
		r := Rect{X: 60 + float64(mode)*195, Y: 125, W: 180, H: 72}
		if r.Contains(x, y) {
			return menuHitOnlineRuleMapChoiceBase + mode
		}
	}
	for mode := 0; mode < 3; mode++ {
		r := Rect{X: 80 + float64(mode)*250, Y: 220, W: 230, H: 70}
		if r.Contains(x, y) {
			return menuHitOnlineRuleWeaponChoiceBase + mode
		}
	}
	if g.onlineMapRotation == onlineMapFixed && (Rect{X: 80, Y: 305, W: 360, H: 65}).Contains(x, y) {
		return menuHitOnlineRuleFixedMapPicker
	}
	if g.onlineWeaponMode == onlineWeaponsOnlyOne && (Rect{X: 460, Y: 305, W: 360, H: 65}).Contains(x, y) {
		return menuHitOnlineRuleOnlyWeaponPicker
	}
	if g.onlineWeaponMode == onlineWeaponsBlocklist && (Rect{X: 460, Y: 305, W: 360, H: 65}).Contains(x, y) {
		return menuHitOnlineRuleBlocklist
	}
	if (Rect{X: 80, Y: 390, W: 220, H: 52}).Contains(x, y) {
		return menuHitOnlineRulePerks
	}
	if (Rect{X: 315, Y: 390, W: 220, H: 52}).Contains(x, y) {
		return menuHitOnlineRulePowerups
	}
	if (Rect{X: 550, Y: 390, W: 270, H: 52}).Contains(x, y) {
		return menuHitOnlineRulePerkBlocklist
	}
	return menuHitNone
}

func (g *Game) onlineBlocklistMaxPage() int {
	weapons := onlineAllWeaponNumbers()
	if len(weapons) == 0 {
		return 0
	}
	return (len(weapons) - 1) / 24
}

func (g *Game) onlineBlocklistHitAt(x, y float64) int {
	if onlineOverlayBackRect.Contains(x, y) {
		return menuHitOnlineOverlayBack
	}
	if (Rect{X: 230, Y: 470, W: 150, H: 42}).Contains(x, y) {
		return menuHitOnlineBlockPrevPage
	}
	if (Rect{X: 520, Y: 470, W: 150, H: 42}).Contains(x, y) {
		return menuHitOnlineBlockNextPage
	}
	start := g.onlineBlocklistPage * 24
	weapons := onlineAllWeaponNumbers()
	for i := 0; i < 24 && start+i < len(weapons); i++ {
		col := i % 4
		row := i / 4
		r := Rect{X: 80 + float64(col)*190, Y: 125 + float64(row)*52, W: 175, H: 42}
		if r.Contains(x, y) {
			return menuHitOnlineBlockBase + start + i
		}
	}
	return menuHitNone
}

func (g *Game) onlinePerkBlocklistHitAt(x, y float64) int {
	if onlineOverlayBackRect.Contains(x, y) {
		return menuHitOnlineOverlayBack
	}
	for i, perk := range selectablePerkNumbers() {
		r := onlinePerkBlocklistRect(i)
		if r.Contains(x, y) {
			return menuHitOnlinePerkBase + perk
		}
	}
	for i, ability := range realAbilityNumbers() {
		if onlineAbilityBlocklistRect(i).Contains(x, y) {
			return menuHitOnlineAbilityBase + ability
		}
	}
	return menuHitNone
}

func (g *Game) onlineMapSummary() string {
	if g.customSeriesPlaylistConfigured {
		return fmt.Sprintf("PLAYLIST - %d ROUNDS", clampCustomSeriesRounds(g.customSeriesRounds))
	}
	if g.onlineMapRotation != onlineMapFixed {
		return onlineMapRotationLabel(g.onlineMapRotation)
	}
	if g.customMap == 0 {
		return "RANDOM"
	}
	if isGM2MapID(g.customMap) {
		return gm2MapDisplayNames[gm2SourceMapNumber(g.customMap)]
	}
	if name := sourceMapDisplayNames[g.customMap]; name != "" {
		return name
	}
	return fmt.Sprintf("MAP %d", g.customMap)
}

func (g *Game) onlineRulesSummary() string {
	weapon := onlineWeaponModeLabel(g.onlineWeaponMode)
	if g.onlineWeaponMode == onlineWeaponsOnlyOne {
		weapon = "ONLY " + weaponDisplayName(g.onlineOnlyWeapon)
	} else if g.onlineWeaponMode == onlineWeaponsBlocklist {
		weapon = fmt.Sprintf("BLOCKLIST %d", g.blockedOnlineWeaponCount())
	}
	perks := fmt.Sprintf("PERKS+ABILITIES ON (%d BLOCKED)", g.blockedOnlinePerkCount()+g.blockedOnlineAbilityCount())
	if !g.onlinePerksEnabled {
		perks = "PERKS+ABILITIES OFF"
	}
	power := "POWERUPS ON"
	if !g.onlinePowerupsEnabled {
		power = "POWERUPS OFF"
	}
	return weapon + " | " + perks + " | " + power
}

// onlinePerkBlocklistRect lays the selectable perks out in five columns.
func onlinePerkBlocklistRect(i int) Rect {
	col := i % 5
	row := i / 5
	return Rect{X: 40 + float64(col)*165, Y: 142 + float64(row)*64, W: 155, H: 56}
}

// onlineAbilityBlocklistRect lays the real abilities out in four columns
// below the perks.
func onlineAbilityBlocklistRect(i int) Rect {
	col := i % 4
	row := i / 4
	return Rect{X: 40 + float64(col)*205, Y: 300 + float64(row)*64, W: 195, H: 56}
}
