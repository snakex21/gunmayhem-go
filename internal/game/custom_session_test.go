package game

import (
	"testing"
)

func TestServerBrowserClickSelectsPreviewWithoutJoining(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	g.customNetworkOpen = true
	g.multiplayerServers = []multiplayerServerInfo{{
		ID: "preview", Name: "Preview Server", Address: "127.0.0.1:7777", Players: 1, MaxPlayers: 4,
		GameMode: SourceGameModeNormal, Lives: 7, Rounds: 3, Maps: [customSeriesMaxRounds]int{2, 7, 11}, RuleSummary: "LOW GRAVITY",
	}}
	g.activateCustomNetworkHit(menuHitCustomNetworkServerBase)
	if g.multiplayerSelectedServer != 0 {
		t.Fatalf("selected server=%d want0", g.multiplayerSelectedServer)
	}
	if g.netplay != nil {
		t.Fatal("clicking a server must preview it, not join immediately")
	}
}

func TestHostProbePublishesMatchMapsAndRules(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	g.customMode = SourceGameModeTeams
	g.customLives = 7
	g.customSeriesRounds = 3
	g.setCustomSeriesMap(1, 2)
	g.setCustomSeriesMap(2, 7)
	g.setCustomSeriesMap(3, 11)
	g.customRuleEnabled[customModifierLowGravity] = true
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	host.publicSessionID = "probe-preview"
	host.publicSessionName = "Preview Host"
	g.netplay = host
	g.syncOnlineHostState()
	info := host.publicProbeInfo()
	if info.GameMode != SourceGameModeTeams || info.Lives != 7 || info.Rounds != 3 {
		t.Fatalf("probe match=%+v", info)
	}
	if info.Maps[0] != 2 || info.Maps[1] != 7 || info.Maps[2] != 11 {
		t.Fatalf("probe maps=%v", info.Maps[:3])
	}
	if info.RuleSummary == "" || info.RuleSummary == "NO MODIFIERS" {
		t.Fatalf("probe rules=%q", info.RuleSummary)
	}
}

func TestHostHumanCapacityFollowsOnlineSlotsNotBots(t *testing.T) {
	g := New()
	g.customPlayers[0].Type = 1
	g.customPlayers[1].Type = 1 // ONLINE P2
	g.customPlayers[2].Type = 2 // AI bot
	g.customPlayers[3].Type = 0 // empty
	if err := g.StartNetHost("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer g.CloseNetplay()
	if got := g.netplay.humanCapacity(); got != 2 {
		t.Fatalf("human capacity=%d want2", got)
	}

	g.customPlayers[3].Type = 1 // open P4 for another online human
	g.syncOnlineHostState()
	if got := g.netplay.humanCapacity(); got != 3 {
		t.Fatalf("human capacity after opening P4=%d want3", got)
	}
}

func TestOnlineCustomCanStartWithBotWhileOnlineSlotWaits(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.customPlayers[0].Type = 1
	g.customPlayers[1].Type = 1 // waiting ONLINE P2
	g.customPlayers[2].Type = 2 // AI P3
	g.customPlayers[3].Type = 0
	g.customMap = 1
	g.customMode = SourceGameModeNormal
	g.customLives = 5
	if err := g.StartNetHost("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer g.CloseNetplay()
	g.syncOnlineHostState()
	if got := g.onlineActiveParticipantCount(); got != 2 {
		t.Fatalf("active participants=%d want host+bot=2", got)
	}
	g.startCustomGame()
	if len(g.players) != 2 || g.players[0].ID != 1 || g.players[1].ID != 3 || !g.players[1].AI {
		t.Fatalf("players=%+v; waiting ONLINE slot must not create a ghost player", g.players)
	}
	if g.customPlayers[1].Type != 1 {
		t.Fatal("waiting ONLINE slot was destroyed at match start")
	}
}

func TestLeavingCustomNetworkPanelClosesHostedSession(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	g.customNetworkOpen = true
	g.onlineMatchSetup = true
	if err := g.StartNetHost("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	if g.netplay == nil {
		t.Fatal("host did not start")
	}
	g.multiplayerServers = []multiplayerServerInfo{{ID: g.netplay.publicSessionID, Name: "self", Address: "127.0.0.1:7777"}}
	g.leaveCustomNetworkPanel()
	if g.netplay != nil {
		t.Fatal("host session remained alive after leaving network panel")
	}
	if g.customNetworkOpen || g.onlineMatchSetup {
		t.Fatalf("network state remained open: panel=%v setup=%v", g.customNetworkOpen, g.onlineMatchSetup)
	}
	if len(g.multiplayerServers) != 0 {
		t.Fatalf("stale server list remained: %+v", g.multiplayerServers)
	}
}

func TestBrowserFiltersCurrentHostSession(t *testing.T) {
	g := New()
	if err := g.StartNetHost("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer g.CloseNetplay()
	ownID := g.netplay.publicSessionID
	servers := []multiplayerServerInfo{
		{ID: ownID, Name: "self", Address: "127.0.0.1:7777"},
		{ID: "other", Name: "other", Address: "127.0.0.1:8888"},
	}
	filtered := g.filterOwnHostedServer(servers)
	if len(filtered) != 1 || filtered[0].ID != "other" {
		t.Fatalf("filtered servers=%+v", filtered)
	}
}

func TestOnlineHostMayEditHostAndBotsButNotRemoteSlots(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	g.customPlayers[1].Type = 1
	g.customPlayers[2].Type = 2
	if err := g.StartNetHost("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer g.CloseNetplay()
	if !g.onlineMayEditCustomCharacterSlot(0) {
		t.Fatal("host must be able to edit its own slot")
	}
	if g.onlineMayEditCustomCharacterSlot(1) {
		t.Fatal("host must not edit a human ONLINE slot, even while it is waiting")
	}
	g.onlinePlayerSlots[1] = true
	if g.onlineMayEditCustomCharacterSlot(1) {
		t.Fatal("host must not edit a connected ONLINE player's cosmetics")
	}
	if !g.onlineMayEditCustomCharacterSlot(2) {
		t.Fatal("host must be able to edit an AI bot slot")
	}
}

func TestOnlineCharacterOwnershipBlocksHostFromRemoteLoadout(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	g.customPage = customPagePlayers
	g.customPlayers[1].Type = 1
	g.customPlayers[1].Gun = 1
	g.customPlayers[2].Type = 2
	g.customPlayers[2].Gun = 1
	if err := g.StartNetHost("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer g.CloseNetplay()

	if !g.activateCustomPlayerEditorHit(customSelectHit(1, customEditGun, 4)) {
		t.Fatal("remote player edit hit was not consumed")
	}
	if g.customPlayers[1].Gun != 1 {
		t.Fatalf("host changed remote player's gun to %d", g.customPlayers[1].Gun)
	}
	if !g.activateCustomPlayerEditorHit(customSelectHit(2, customEditGun, 4)) || g.customPlayers[2].Gun != 4 {
		t.Fatalf("host could not edit bot loadout: gun=%d", g.customPlayers[2].Gun)
	}

	g.customNameFocus = 0
	if !g.activateCustomPlayerEditorHit(menuHitPlayerColorBase+500) || g.customNameFocus != 1 {
		t.Fatalf("host nickname field did not take focus: focus=%d", g.customNameFocus)
	}
	g.customNameFocus = 0
	g.activateCustomPlayerEditorHit(menuHitPlayerColorBase + 501)
	if g.customNameFocus != 0 {
		t.Fatal("host can focus a remote player's nickname field")
	}
}

func TestOnlineClientOwnsOnlyItsCharacterSlot(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	g.customPlayers[0].Type = 1
	g.customPlayers[1].Type = 1
	g.netplay = &netplaySession{mode: netplayClient}
	g.netplay.assignedPlayerID.Store(2)
	if !g.onlineMayEditCustomCharacterSlot(1) {
		t.Fatal("client cannot edit its own character")
	}
	if g.onlineMayEditCustomCharacterSlot(0) {
		t.Fatal("client can edit host character")
	}
}

func TestOnlineClientCanSelectDevelopedPistolOnOwnSlot(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	g.customPage = customPagePlayers
	g.customPlayers[1].Type = 1
	g.customPlayers[1].Gun = 1
	g.netplay = &netplaySession{mode: netplayClient}
	g.netplay.assignedPlayerID.Store(2)

	hit := customDevelopedGunHit(1, 231)
	slot, ok := customPlayerEditorHitSlot(hit)
	if !ok || slot != 1 {
		t.Fatalf("developed pistol hit was not attributed to client slot: hit=%d slot=%d ok=%v", hit, slot, ok)
	}
	if !g.activateCustomPlayerEditorHit(hit) {
		t.Fatal("developed pistol hit was not consumed")
	}
	if g.customPlayers[1].Gun != 231 {
		t.Fatalf("client selected gun=%d want Peacekeeper 231", g.customPlayers[1].Gun)
	}
}

func TestHostCanKickConnectedPlayerAndRemoveBot(t *testing.T) {
	g := New()
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.customPlayers[1].Type = 1
	g.customPlayers[2].Type = 2
	if err := g.StartNetHost("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer g.CloseNetplay()

	client, err := newClientNetplay(g.netplay.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	waitFor(t, "connected player 2", func() bool { return g.netplay.connectedCount() == 1 })
	g.syncOnlineHostState()
	if !g.onlinePlayerSlots[1] {
		t.Fatal("player 2 was not marked connected")
	}

	g.removeOnlineCustomSlot(1)
	if g.customPlayers[1].Type != 0 || g.onlinePlayerSlots[1] {
		t.Fatalf("kicked player slot survived: type=%d online=%v", g.customPlayers[1].Type, g.onlinePlayerSlots[1])
	}
	waitFor(t, "kicked peer disconnect", func() bool { return g.netplay.connectedCount() == 0 })
	g.syncOnlineHostState()
	if g.customPlayers[1].Type != 0 {
		t.Fatalf("stale peer config resurrected kicked player: type=%d", g.customPlayers[1].Type)
	}

	g.removeOnlineCustomSlot(2)
	if g.customPlayers[2].Type != 0 {
		t.Fatalf("bot was not removed: type=%d", g.customPlayers[2].Type)
	}
}
