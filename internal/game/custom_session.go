package game

import (
	"fmt"
	"strings"
)

const (
	menuHitCustomSessionLocal = 9700 + iota
	menuHitCustomSessionOnline
	menuHitCustomSessionNetwork
	menuHitCustomSessionRules
	menuHitCustomNetworkHostOpen
	menuHitCustomNetworkHostInvite
	menuHitCustomNetworkRefresh
	menuHitCustomNetworkJoin
	menuHitCustomNetworkDisconnect
	menuHitCustomNetworkClose
	menuHitCustomNetworkMainMenu
	menuHitCustomNetworkCreateSetup
	menuHitCustomNetworkJoinSelected
)

const menuHitCustomNetworkServerBase = 9800

func (g *Game) returnToOnlineServerBrowser() {
	if g == nil {
		return
	}
	_ = g.CloseNetplay()
	g.customOnlineMode = true
	g.onlineMatchSetup = false
	g.onlineZombie = false
	g.customNetworkOpen = true
	g.customHostSetupOpen = false
	g.multiplayerFocus = false
	g.multiplayerNameFocus = false
	g.multiplayerServers = nil
	g.multiplayerServerScroll = 0
	g.multiplayerSelectedServer = -1
	// Invalidate discovery that may still contain the host being closed.
	g.multiplayerDiscoveryCh = nil
	g.multiplayerDiscoveryDone = false
	g.multiplayerMessage = "CHOOSE A SESSION OR CREATE ONE"
	g.beginMultiplayerServerBrowser()
}

func (g *Game) leaveCustomNetworkPanel() {
	if g == nil {
		return
	}
	_ = g.CloseNetplay()
	g.onlineMatchSetup = false
	g.customNetworkOpen = false
	g.customHostSetupOpen = false
	g.multiplayerFocus = false
	g.multiplayerNameFocus = false
	g.multiplayerServers = nil
	g.multiplayerServerScroll = 0
	g.multiplayerSelectedServer = -1
	g.multiplayerDiscoveryCh = nil
	g.multiplayerDiscoveryDone = false
	g.multiplayerMessage = "ONLINE SETTINGS KEPT"
}

func (g *Game) activateCustomNetworkHit(hit int) {
	switch {
	case hit == menuHitCustomNetworkCreateSetup:
		g.customHostSetupOpen = true
		g.multiplayerFocus = false
		g.multiplayerNameFocus = false
		g.multiplayerMessage = "CONFIGURE THE NEW SESSION"
	case hit == menuHitCustomNetworkRefresh:
		g.multiplayerSelectedServer = -1
		g.refreshMultiplayerServers()
	case hit == menuHitCustomNetworkJoin:
		g.joinFromCustomSession()
	case hit == menuHitCustomNetworkJoinSelected:
		g.joinSelectedCustomServer()
	case hit == menuHitCustomNetworkClose:
		g.leaveCustomNetworkPanel()
	case hit == menuHitCustomNetworkMainMenu:
		_ = g.CloseNetplay()
		g.customOnlineMode = false
		g.customNetworkOpen = false
		g.customHostSetupOpen = false
		g.onlineMatchSetup = false
		g.multiplayerFocus = false
		g.multiplayerNameFocus = false
		g.menuPressed = menuHitNone
		g.beginFade(screenMainMenu, fadePurposeScreen)
	case hit >= menuHitCustomNetworkServerBase:
		index := hit - menuHitCustomNetworkServerBase
		if index < 0 || index >= len(g.multiplayerServers) {
			return
		}
		g.multiplayerSelectedServer = index
		g.multiplayerFocus = false
		server := g.multiplayerServers[index]
		g.multiplayerMessage = "SELECTED: " + server.Name
	}
}

func (g *Game) joinSelectedCustomServer() {
	if g == nil || g.multiplayerSelectedServer < 0 || g.multiplayerSelectedServer >= len(g.multiplayerServers) {
		return
	}
	server := g.multiplayerServers[g.multiplayerSelectedServer]
	if server.Full() {
		g.multiplayerMessage = "SERVER FULL: " + server.Name
		return
	}
	g.multiplayerAddress = server.Address
	g.beginAsyncNetJoin(server.Name, func(scratch *Game) error {
		return scratch.StartNetClientServer(server)
	}, g.enterOnlineCustomGameAsClient)
}

// enterOnlineCustomGameAsClient switches the UI to the SLOTS page after a
// successful join from the Custom Game server browser.
func (g *Game) enterOnlineCustomGameAsClient() {
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.customNetworkOpen = false
	g.customHostSetupOpen = false
	g.multiplayerFocus = false
	// A client joins the final wizard step. Host-owned MATCH/MAPS are visible
	// through the browser preview and synchronized snapshots; the client mainly
	// chooses their own character in the SLOTS step.
	g.setCustomV2Page(customPagePlayers)
	g.onlineClientTab = onlineLobbyTabPlayers
	g.onlineClientTabSet = true
	g.multiplayerMessage = "CONNECTED - CHOOSE YOUR CHARACTER"
	g.screen = screenCustomGame
}

func (g *Game) joinFromCustomSession() {
	address := strings.TrimSpace(g.multiplayerAddress)
	if address == "" {
		g.multiplayerMessage = "ENTER A JOIN CODE OR IP:PORT"
		return
	}
	g.beginAsyncNetJoin(address, func(scratch *Game) error {
		return scratch.startNetClientFromInput(address)
	}, g.enterOnlineCustomGameAsClient)
}

func (g *Game) configuredOnlineBotCount() int {
	count := 0
	for slot := 1; slot < len(g.customPlayers); slot++ {
		if g.customPlayers[slot].Type == 2 {
			count++
		}
	}
	return count
}

func (g *Game) configuredOnlineHumanCapacity() int {
	capacity := 1
	for slot := 1; slot < len(g.customPlayers); slot++ {
		if g.customPlayers[slot].Type == 1 {
			capacity++
		}
	}
	return capacity
}

func (g *Game) customSessionStatus() string {
	if !g.customOnlineMode {
		return "LOCAL"
	}
	if g.netplay == nil {
		return "SERVER BROWSER - OFFLINE"
	}
	bots := g.configuredOnlineBotCount()
	capacity := g.configuredOnlineHumanCapacity()
	if g.netplay.mode == netplayHost {
		scope := "LAN"
		if g.onlineSessionScope == onlineSessionScopeInternet {
			scope = "WORLD"
		}
		access := "OPEN"
		if g.onlineSessionVisibility == onlineSessionInviteOnly {
			access = "INVITE"
		}
		return fmt.Sprintf("%s %s  H%d/%d  AI %d", scope, access, 1+g.netplay.connectedCount(), capacity, bots)
	}
	id := g.netplay.assignedPlayer()
	if id > 0 {
		connected := 0
		for _, present := range g.onlinePlayerSlots {
			if present {
				connected++
			}
		}
		if connected == 0 {
			connected = 1
		}
		return fmt.Sprintf("P%d  H%d/%d  AI %d", id, connected, capacity, bots)
	}
	return "ONLINE - CONNECTING"
}

func onOff(v bool) string {
	if v {
		return "ON"
	}
	return "OFF"
}

func customPlayerEditorHitSlot(hit int) (int, bool) {
	if slot, _, ok := decodeCustomDevelopedGunHit(hit); ok {
		return slot, true
	}
	if hit >= menuHitPlayerThrowableBase && hit < menuHitPlayerThrowableBase+4*throwableHitStride {
		return (hit - menuHitPlayerThrowableBase) / throwableHitStride, true
	}
	if slot, _, _, ok := decodeCustomSelectHit(hit); ok {
		return slot, true
	}
	if hit >= menuHitPlayerTeamBase && hit < menuHitPlayerTeamBase+40 {
		return (hit - menuHitPlayerTeamBase) / 10, true
	}
	if hit >= menuHitPlayerColorBase && hit < menuHitPlayerColorBase+400 {
		return (hit - menuHitPlayerColorBase) / 100, true
	}
	if hit >= menuHitPlayerColorBase+500 && hit < menuHitPlayerColorBase+504 {
		return hit - (menuHitPlayerColorBase + 500), true
	}
	return 0, false
}

func (g *Game) onlineMayEditCustomCharacterSlot(slot int) bool {
	if g == nil || slot < 0 || slot >= len(g.customPlayers) {
		return false
	}
	if !g.customOnlineMode {
		return true
	}
	if g.netplay == nil {
		// Before a network session exists there is no remote character to edit.
		// Keep only the local player's character and AI bot loadouts editable.
		return slot == 0 || g.customPlayers[slot].Type == 2
	}
	switch g.netplay.mode {
	case netplayHost:
		// The host owns Player 1 and may configure AI bots. Human network slots
		// are owned by the corresponding clients, whether they are already
		// connected or merely waiting for a player.
		return slot == 0 || g.customPlayers[slot].Type == 2
	case netplayClient:
		return g.onlineLocalPlayerID() == slot+1
	default:
		return false
	}
}

func (g *Game) removeOnlineCustomSlot(slot int) {
	if g == nil || slot <= 0 || slot >= len(g.customPlayers) || !g.customOnlineMode || g.netplay == nil || g.netplay.mode != netplayHost {
		return
	}
	kind := g.customPlayers[slot].Type
	if kind == 0 {
		return
	}
	wasConnected := g.onlinePlayerSlots[slot]
	g.customPlayers[slot].Type = 0
	g.customEditor[slot] = 0
	if g.customNameFocus == slot+1 {
		g.customNameFocus = 0
	}
	g.syncOnlineHostState()
	g.onlinePlayerSlots[slot] = false
	switch {
	case kind == 2:
		g.multiplayerMessage = fmt.Sprintf("BOT REMOVED FROM SLOT %d", slot+1)
	case wasConnected:
		g.multiplayerMessage = fmt.Sprintf("PLAYER %d KICKED", slot+1)
	default:
		g.multiplayerMessage = fmt.Sprintf("ONLINE SLOT %d CLOSED", slot+1)
	}
}
