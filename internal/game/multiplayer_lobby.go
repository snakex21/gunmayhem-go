package game

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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

func (g *Game) updateMultiplayerLobby() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.onlineVisualPicker != onlineVisualNone {
			g.closeOnlineVisualPicker()
			return
		}
		if g.onlinePerkBlocklistOpen {
			g.onlinePerkBlocklistOpen = false
			return
		}
		if g.onlineBlocklistOpen {
			g.onlineBlocklistOpen = false
			return
		}
		if g.onlineRulesOpen {
			g.onlineRulesOpen = false
			return
		}
		if g.onlineLobbyCustomize {
			g.onlineLobbyCustomize = false
			return
		}
		g.leaveMultiplayerLobby()
		return
	}

	mx, my := g.cursorPos()
	x, y := float64(mx), float64(my)
	var hit int
	switch {
	case g.onlineVisualPicker != onlineVisualNone:
		hit = g.onlineVisualPickerHitAt(x, y)
	case g.onlinePerkBlocklistOpen:
		hit = g.onlinePerkBlocklistHitAt(x, y)
	case g.onlineBlocklistOpen:
		hit = g.onlineBlocklistHitAt(x, y)
	case g.onlineRulesOpen:
		hit = g.onlineRulesHitAt(x, y)
	case g.onlineLobbyCustomize:
		hit = g.onlineCharacterHitAt(x, y)
	default:
		hit = g.multiplayerLobbyHitAt(x, y)
	}
	activated, ok := g.resolveMenuRelease(hit)
	if !ok {
		return
	}

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

func drawLobbySlot(screen *ebiten.Image, r Rect, title, detail, config string, occupied bool, local bool) {
	bg := color.NRGBA{R: 0x2a, G: 0x2f, B: 0x37, A: 0xff}
	if occupied {
		bg = color.NRGBA{R: 0x3c, G: 0x43, B: 0x4e, A: 0xff}
	}
	if local {
		bg = color.NRGBA{R: 0x55, G: 0x45, B: 0x2d, A: 0xff}
	}
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, bg)
	drawSourceMenuText(screen, title, menuFontTwCen, 20, uiTextLight, r.X+12, r.Y+10)
	detailColor := color.NRGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff}
	if occupied {
		detailColor = color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}
	}
	drawSourceMenuText(screen, detail, menuFontCondensed, 13.5, detailColor, r.X+12, r.Y+40)
	if occupied {
		drawSourceMenuText(screen, config, menuFontCondensed, 11.5, color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}, r.X+12, r.Y+61)
	}
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

func (g *Game) drawMultiplayerLobby(screen *ebiten.Image) {
	if g.onlineVisualPicker != onlineVisualNone {
		g.drawOnlineVisualPicker(screen)
		return
	}
	if g.onlinePerkBlocklistOpen {
		g.drawOnlinePerkBlocklist(screen)
		return
	}
	if g.onlineBlocklistOpen {
		g.drawOnlineBlocklist(screen)
		return
	}
	if g.onlineRulesOpen {
		g.drawOnlineRules(screen)
		return
	}
	if g.onlineLobbyCustomize {
		g.drawOnlineCharacterEditor(screen)
		return
	}

	ebitenutil.DrawRect(screen, 25, 45, 850, 540, color.NRGBA{R: 0x18, G: 0x1c, B: 0x22, A: 0xf2})
	drawSourceMenuText(screen, "ONLINE LOBBY", menuFontUIBold, 36, uiTextLight, 55, 62)

	count := g.onlineConnectedPlayerCount()
	role := "CLIENT"
	if g.multiplayerLobbyIsHost() {
		role = "HOST"
	}
	drawSourceMenuText(screen, fmt.Sprintf("%s   PLAYERS %d/%d   JOIN-IN-PROGRESS ON", role, count, maxOnlinePlayers), menuFontCondensedExtraBold, 16, uiTextLight, 55, 102)

	localID := g.onlineLocalPlayerID()
	positions := [4]Rect{
		{X: 55, Y: 135, W: 380, H: 92},
		{X: 465, Y: 135, W: 380, H: 92},
		{X: 55, Y: 242, W: 380, H: 92},
		{X: 465, Y: 242, W: 380, H: 92},
	}
	for id := 1; id <= 4; id++ {
		occupied := g.onlinePlayerSlots[id-1]
		if g.multiplayerLobbyIsHost() {
			occupied = id == 1
			if id > 1 {
				for _, connectedID := range g.netplay.connectedPlayerIDs() {
					if connectedID == id {
						occupied = true
						break
					}
				}
			}
		}
		title := fmt.Sprintf("PLAYER %d", id)
		if id == 1 {
			title += " - HOST"
		} else {
			title += " - ONLINE"
		}
		detail := "WAITING FOR PLAYER..."
		cfgSummary := ""
		if occupied {
			cfg := g.customPlayers[id-1]
			detail = cfg.Name
			if id > 1 && g.multiplayerLobbyIsHost() {
				if addr := g.netplay.peerAddress(id); addr != "" {
					detail += "  " + addr
				}
			}
			team := "A"
			if cfg.Team == 2 {
				team = "B"
			}
			cfgSummary = fmt.Sprintf("%s   %s + %s   TEAM %s", weaponDisplayName(cfg.Gun), onlinePerkName(cfg.Perk), abilityName(cfg.Ability), team)
		}
		drawLobbySlot(screen, positions[id-1], title, detail, cfgSummary, occupied, localID == id)
		if occupied {
			g.drawOnlineCharacterMini(screen, positions[id-1], g.customPlayers[id-1])
		}
	}

	drawSourceMenuText(screen, "LIVE MATCH SETTINGS", menuFontCondensedExtraBold, 15, uiTextLight, 55, 360)
	drawSourceMenuText(screen, fmt.Sprintf("MODE %d   LIVES %d   MAP %s", g.customMode, g.customLives, g.onlineMapSummary()), menuFontCondensed, 14, uiTextLight, 55, 386)
	drawSourceMenuText(screen, g.onlineRulesSummary(), menuFontCondensed, 13.5, uiTextLight, 55, 410)
	drawSourceMenuText(screen, "Host settings are mirrored live to every connected player.", menuFontCondensed, 12.5, color.NRGBA{R: 0xbb, G: 0xbb, B: 0xbb, A: 0xff}, 55, 438)
	if g.multiplayerLobbyIsHost() {
		sessionName := g.multiplayerSessionName
		if sessionName == "" {
			sessionName = "Gun Mayhem Game"
		}
		scopeText := "LAN ONLY"
		if g.onlineSessionScope == onlineSessionScopeInternet {
			scopeText = "INTERNET + LAN"
		}
		accessText := "OPEN"
		if g.onlineSessionVisibility == onlineSessionInviteOnly {
			accessText = "INVITE CODE"
		}
		modeText := scopeText + " / " + accessText
		drawSourceMenuText(screen, "SESSION: "+sessionName+"   "+modeText, menuFontCondensedExtraBold, 12.5, color.NRGBA{R: 0xff, G: 0xa0, B: 0x24, A: 0xff}, 55, 466)
		// OPEN hosts advertise browser status; only INVITE ONLY hosts expose a code.
		drawWrappedMenuText(screen, g.hostJoinCodeLabelWithCopyHint(), menuFontCondensed, 9.5, color.White, 55, 490, 540, 11, 2, nil)
	}

	drawDevelopedMenuButton(screen, multiplayerLobbyLeaveRect, "LEAVE")
	drawDevelopedMenuButton(screen, multiplayerLobbyCustomizeRect, "MY CHARACTER")
	if g.multiplayerLobbyIsHost() {
		drawDevelopedMenuButton(screen, multiplayerLobbyRulesRect, "ADVANCED RULES")
		drawDevelopedMenuButton(screen, multiplayerLobbySetupRect, "MATCH SETUP")
		if count < 2 {
			drawSourceMenuText(screen, "SETUP IS AVAILABLE NOW - START WAITS FOR A 2ND PARTICIPANT", menuFontCondensed, 9.5,
				color.NRGBA{R: 0xb7, G: 0xbc, B: 0xc4, A: 0xff}, 612, 511)
		}
	} else {
		drawSourceMenuText(screen, "WAITING FOR HOST", menuFontCondensedExtraBold, 14, uiTextLight, 650, 542)
	}
}

func (g *Game) drawOnlineCharacterEditor(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, 25, 45, 850, 540, color.NRGBA{R: 0x18, G: 0x1c, B: 0x22, A: 0xf8})
	id := g.onlineLocalPlayerID()
	if id < 1 || id > 4 {
		drawSourceMenuText(screen, "WAITING FOR PLAYER SLOT...", menuFontTwCen, 28, uiTextLight, 170, 250)
		drawDevelopedMenuButton(screen, onlineOverlayBackRect, "BACK")
		return
	}
	cfg := g.customPlayers[id-1]
	drawSourceMenuText(screen, fmt.Sprintf("PLAYER %d - CHARACTER", id), menuFontUIBold, 32, uiTextLight, 55, 65)
	drawSourceMenuText(screen, "Click a picture to open the visual selector.", menuFontCondensed, 13.5, color.NRGBA{R: 0xbb, G: 0xbb, B: 0xbb, A: 0xff}, 405, 100)

	preview := ebiten.NewImage(300, 360)
	preview.Fill(color.NRGBA{R: 0x2d, G: 0x32, B: 0x3a, A: 0xff})
	g.drawSetupPlayerPreview(preview, cfg, 1)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(65, 125)
	screen.DrawImage(preview, op)

	for field := 0; field < onlineCharFieldCount; field++ {
		g.drawOnlineCharacterFieldTile(screen, field, cfg)
	}
	drawDevelopedMenuButton(screen, onlineOverlayBackRect, "BACK TO SETUP")
}

func (g *Game) drawOnlineRules(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, 25, 45, 850, 540, color.NRGBA{R: 0x18, G: 0x1c, B: 0x22, A: 0xf8})
	drawSourceMenuText(screen, "ONLINE MATCH RULES", menuFontUIBold, 31, uiTextLight, 60, 62)
	drawSourceMenuText(screen, "Choose by clicking the pictures. Changes are mirrored live.", menuFontCondensed, 13, color.NRGBA{R: 0xbb, G: 0xbb, B: 0xbb, A: 0xff}, 60, 101)

	drawSourceMenuText(screen, "MAP ROTATION", menuFontCondensedExtraBold, 13, uiTextLight, 60, 112)
	for mode := 0; mode < 4; mode++ {
		g.drawOnlineMapModeTile(screen, mode, Rect{X: 60 + float64(mode)*195, Y: 125, W: 180, H: 72})
	}

	drawSourceMenuText(screen, "WEAPON RULE", menuFontCondensedExtraBold, 13, uiTextLight, 80, 207)
	for mode := 0; mode < 3; mode++ {
		g.drawOnlineWeaponModeTile(screen, mode, Rect{X: 80 + float64(mode)*250, Y: 220, W: 230, H: 70})
	}

	if g.onlineMapRotation == onlineMapFixed {
		r := Rect{X: 80, Y: 305, W: 360, H: 65}
		drawOnlineSelectedFrame(screen, r, false)
		g.drawOnlineMapThumb(screen, g.customMap, Rect{X: r.X + 5, Y: r.Y + 5, W: 105, H: r.H - 10})
		drawSourceMenuText(screen, "FIXED MAP", menuFontCondensedExtraBold, 11.5, color.NRGBA{R: 0xff, G: 0x99, A: 0xff}, r.X+122, r.Y+10)
		drawSourceMenuText(screen, onlineMapNumberLabel(g.customMap), menuFontTwCen, 13.5, uiTextLight, r.X+122, r.Y+31)
	}

	if g.onlineWeaponMode == onlineWeaponsOnlyOne {
		r := Rect{X: 460, Y: 305, W: 360, H: 65}
		drawOnlineSelectedFrame(screen, r, false)
		g.drawOnlineWeaponThumb(screen, g.onlineOnlyWeapon, Rect{X: r.X + 5, Y: r.Y + 5, W: 105, H: r.H - 10})
		drawSourceMenuText(screen, "ONLY WEAPON", menuFontCondensedExtraBold, 11.5, color.NRGBA{R: 0xff, G: 0x99, A: 0xff}, r.X+122, r.Y+10)
		drawSourceMenuText(screen, weaponDisplayName(g.onlineOnlyWeapon), menuFontTwCen, 13, uiTextLight, r.X+122, r.Y+31)
	} else if g.onlineWeaponMode == onlineWeaponsBlocklist {
		drawDevelopedMenuButton(screen, Rect{X: 460, Y: 305, W: 360, H: 65}, fmt.Sprintf("WEAPON BLOCKLIST  %d BLOCKED", g.blockedOnlineWeaponCount()))
	}

	// One toggle covers both loadout slots (perk and ability / gear).
	perks := fmt.Sprintf("PERKS/ABIL. ON (%d)", g.blockedOnlinePerkCount()+g.blockedOnlineAbilityCount())
	if !g.onlinePerksEnabled {
		perks = "PERKS/ABIL. OFF"
	}
	power := "POWERUPS ON"
	if !g.onlinePowerupsEnabled {
		power = "POWERUPS OFF"
	}
	drawDevelopedMenuButton(screen, Rect{X: 80, Y: 390, W: 220, H: 52}, perks)
	drawDevelopedMenuButton(screen, Rect{X: 315, Y: 390, W: 220, H: 52}, power)
	drawDevelopedMenuButton(screen, Rect{X: 550, Y: 390, W: 270, H: 52}, fmt.Sprintf("PERK/ABILITY BLOCKLIST %d", g.blockedOnlinePerkCount()+g.blockedOnlineAbilityCount()))
	drawSourceMenuText(screen, "Selected tiles are outlined in orange.", menuFontCondensed, 12, color.NRGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff}, 80, 461)
	drawDevelopedMenuButton(screen, onlineOverlayBackRect, "BACK TO SETUP")
}

func (g *Game) drawOnlineBlocklist(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, 25, 45, 850, 540, color.NRGBA{R: 0x18, G: 0x1c, B: 0x22, A: 0xf8})
	drawSourceMenuText(screen, "WEAPON BLOCKLIST", menuFontUIBold, 30, uiTextLight, 80, 65)
	drawSourceMenuText(screen, "Click a weapon to allow/block it. Dark red = blocked.", menuFontCondensed, 13, color.NRGBA{R: 0xbb, G: 0xbb, B: 0xbb, A: 0xff}, 80, 102)

	weapons := onlineAllWeaponNumbers()
	start := g.onlineBlocklistPage * 24
	for i := 0; i < 24 && start+i < len(weapons); i++ {
		index := start + i
		n := weapons[index]
		col := i % 4
		row := i / 4
		r := Rect{X: 80 + float64(col)*190, Y: 125 + float64(row)*52, W: 175, H: 42}
		clr := color.NRGBA{R: 0x46, G: 0x4d, B: 0x57, A: 0xff}
		if n > 0 && n < len(g.onlineBlockedWeapons) && g.onlineBlockedWeapons[n] {
			clr = color.NRGBA{R: 0x65, G: 0x2e, B: 0x2e, A: 0xff}
		}
		ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, clr)
		g.drawOnlineWeaponThumb(screen, n, Rect{X: r.X + 4, Y: r.Y + 4, W: 58, H: r.H - 8})
		name := weaponDisplayName(n)
		if len(name) > 16 {
			name = strings.TrimSpace(name[:16])
		}
		drawSourceMenuText(screen, name, menuFontCondensed, 10.5, uiTextLight, r.X+66, r.Y+7)
		drawSourceMenuText(screen, fmt.Sprintf("#%d", n), menuFontCondensedExtraBold, 9.5, color.NRGBA{R: 0xbb, G: 0xbb, B: 0xbb, A: 0xff}, r.X+66, r.Y+23)
	}

	drawDevelopedMenuButton(screen, Rect{X: 230, Y: 470, W: 150, H: 42}, "PREV PAGE")
	drawSourceMenuText(screen, fmt.Sprintf("PAGE %d/%d", g.onlineBlocklistPage+1, g.onlineBlocklistMaxPage()+1), menuFontTwCen, 15, uiTextLight, 410, 481)
	drawDevelopedMenuButton(screen, Rect{X: 520, Y: 470, W: 150, H: 42}, "NEXT PAGE")
	drawDevelopedMenuButton(screen, onlineOverlayBackRect, "BACK TO RULES")
}

func (g *Game) drawOnlinePerkBlocklist(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, 25, 45, 850, 540, color.NRGBA{R: 0x18, G: 0x1c, B: 0x22, A: 0xf8})
	drawSourceMenuText(screen, "PERK / ABILITY BLOCKLIST", menuFontUIBold, 30, uiTextLight, 170, 70)
	drawSourceMenuText(screen, "Click a perk or ability to allow/block it. Blocked ones cannot be used by any player.", menuFontCondensed, 13, color.NRGBA{R: 0xbb, G: 0xbb, B: 0xbb, A: 0xff}, 170, 104)

	blocked := color.NRGBA{R: 0x65, G: 0x2e, B: 0x2e, A: 0xff}
	allowed := color.NRGBA{R: 0x46, G: 0x4d, B: 0x57, A: 0xff}
	drawSourceMenuText(screen, "PERKS", menuFontCondensedExtraBold, 13, uiTextLight, 40, 124)
	for i, perk := range selectablePerkNumbers() {
		r := onlinePerkBlocklistRect(i)
		clr := allowed
		if perk < len(g.onlineBlockedPerks) && g.onlineBlockedPerks[perk] {
			clr = blocked
		}
		ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, clr)
		drawSetupPerkThumb(screen, perk, Rect{X: r.X + 3, Y: r.Y + 3, W: 62, H: r.H - 6}, 1)
		drawSourceMenuText(screen, onlinePerkName(perk), menuFontTwCen, 11.5, uiTextLight, r.X+66, r.Y+12)
		drawSourceMenuText(screen, fmt.Sprintf("#%d", perk), menuFontCondensedExtraBold, 9.5, color.NRGBA{R: 0xbb, G: 0xbb, B: 0xbb, A: 0xff}, r.X+66, r.Y+30)
	}

	drawSourceMenuText(screen, "ABILITIES / GEAR", menuFontCondensedExtraBold, 13, uiTextLight, 40, 282)
	for i, ability := range realAbilityNumbers() {
		r := onlineAbilityBlocklistRect(i)
		clr := allowed
		if g.onlineBlockedAbilities[ability] {
			clr = blocked
		}
		ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, clr)
		drawAbilityIcon(screen, ability, r.X+20, r.Y+r.H/2, 11, 1)
		drawSourceMenuText(screen, abilityName(ability), menuFontTwCen, 12, uiTextLight, r.X+38, r.Y+r.H/2-8)
	}

	drawSourceMenuText(screen, fmt.Sprintf("BLOCKED: %d PERKS, %d ABILITIES", g.blockedOnlinePerkCount(), g.blockedOnlineAbilityCount()), menuFontCondensedExtraBold, 15, uiTextLight, 40, 535)
	drawDevelopedMenuButton(screen, onlineOverlayBackRect, "BACK TO RULES")
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
