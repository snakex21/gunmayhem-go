package game

const (
	menuHitOnlineOverlayBack = 6200 + iota
	menuHitOnlineRulePerks
	menuHitOnlineRulePowerups
	menuHitOnlineRuleBlocklist
	menuHitOnlineRulePerkBlocklist
	menuHitOnlineBlockPrevPage
	menuHitOnlineBlockNextPage
)

const (
	// The weapon blocklist indexes every weapon (200+), so it needs its own
	// range: at 6600 index 201+ collided with the perk codes at 6800.
	menuHitOnlineBlockBase   = 17000
	menuHitOnlinePerkBase    = 6800
	menuHitOnlineAbilityBase = 6850
)

func decodeOnlineBlockWeaponHit(hit int) (index int, ok bool) {
	index = hit - menuHitOnlineBlockBase
	weapons := onlineAllWeaponNumbers()
	return index, index >= 0 && index < len(weapons)
}

// activateOnlineLobbyHit runs one online lobby / rules overlay action. Each
// code only acts while its overlay is open, exactly as the click handler did;
// the Mayhem UI rules modals (online_lobby_menu.go) call it with the same codes.
func (g *Game) activateOnlineLobbyHit(activated int) {
	if index, ok := decodeOnlineVisualChoiceHit(activated); ok && g.onlineVisualPicker != onlineVisualNone {
		g.applyOnlineVisualChoice(index)
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
			g.onlineRulesOpen = false
		}
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
	}
}

func (g *Game) onlineBlocklistMaxPage() int {
	weapons := onlineAllWeaponNumbers()
	if len(weapons) == 0 {
		return 0
	}
	return (len(weapons) - 1) / 24
}
