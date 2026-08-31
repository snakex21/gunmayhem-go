package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestOnlineClientCanSeeInteractivePregameScreens(t *testing.T) {
	g := New()
	g.netplay = &netplaySession{mode: netplayClient}
	g.screen = screenMultiplayerLobby
	if g.netClientNeedsWaitingScreen() {
		t.Fatal("client lobby was hidden behind the passive waiting screen")
	}
	g.screen = screenCustomGame
	if g.netClientNeedsWaitingScreen() {
		t.Fatal("client slot/character setup was hidden behind the passive waiting screen")
	}
	g.screen = screenGameplay
	if !g.netClientNeedsWaitingScreen() {
		t.Fatal("client gameplay without a host snapshot must still show a safe waiting screen")
	}
	g.netplay.hasState.Store(true)
	if g.netClientNeedsWaitingScreen() {
		t.Fatal("client gameplay stayed hidden after receiving a host snapshot")
	}
}

func TestOnlineClientSnapshotDoesNotUndoLocalCharacterChoice(t *testing.T) {
	g := New()
	g.netplay = &netplaySession{mode: netplayClient}
	g.netplay.assignedPlayerID.Store(2)
	g.screen = screenMultiplayerLobby
	g.customPlayers[1] = CustomPlayerConfig{
		Name: "Client Choice", Color: 9, Shirt: 7, Hat: 18, Eyes: 11,
		Gun: 6, Perk: 8, Throwable: ThrowableMolotov, Type: 1, Team: 2,
	}

	state := netSnapshot{Screen: int(screenMultiplayerLobby)}
	state.OnlineCustomPlayers = sourceDefaultCustomPlayers()
	state.OnlineCustomPlayers[1] = CustomPlayerConfig{Name: "Stale Host Copy", Color: 5, Gun: 1, Type: 1}
	g.applyNetSnapshot(state)

	got := g.customPlayers[1]
	if got.Name != "Client Choice" || got.Color != 9 || got.Shirt != 7 || got.Hat != 18 || got.Eyes != 11 || got.Gun != 6 || got.Perk != 8 || got.Throwable != ThrowableMolotov {
		t.Fatalf("host snapshot undid the client's local character choice: %+v", got)
	}
}

func TestOnlineLobbyHostCanOpenSetupWhileAlone(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	g := New()
	g.netplay = host
	g.screen = screenMultiplayerLobby
	if got := g.multiplayerLobbyHitAt(multiplayerLobbySetupRect.X+10, multiplayerLobbySetupRect.Y+10); got != menuHitMultiplayerLobbySetup {
		t.Fatalf("host setup must be available while alone, hit=%d", got)
	}

	client, err := newClientNetplay(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	waitFor(t, "online lobby 2/2", host.connected)
	if got := g.multiplayerLobbyHitAt(multiplayerLobbySetupRect.X+10, multiplayerLobbySetupRect.Y+10); got != menuHitMultiplayerLobbySetup {
		t.Fatalf("setup hit=%d want=%d after real connection", got, menuHitMultiplayerLobbySetup)
	}
}

func TestOnlineMatchSkipsLocalPlayerSetupAndCreatesTwoNetworkSlots(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	client, err := newClientNetplay(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	waitFor(t, "online connection", host.connected)

	g := New()
	g.netplay = host
	g.onlineMatchSetup = true
	g.customMap = 1
	g.customMode = SourceGameModeNormal
	g.customLives = 7
	for i := range g.customPlayers {
		g.customPlayers[i].Type = 1 // P3/P4 remain open ONLINE slots for join-in-progress.
	}
	g.startCustomGame()

	if len(g.players) != 2 {
		t.Fatalf("online players=%d want2", len(g.players))
	}
	if g.players[0].ID != 1 || g.players[1].ID != 2 {
		t.Fatalf("online player IDs=%d,%d want1,2", g.players[0].ID, g.players[1].ID)
	}
	if g.players[0].AI || g.players[1].AI {
		t.Fatalf("online player unexpectedly AI: p1=%v p2=%v", g.players[0].AI, g.players[1].AI)
	}
	if g.customPlayers[2].Type != 1 || g.customPlayers[3].Type != 1 {
		t.Fatalf("open online slots were destroyed by match start: %+v", g.customPlayers)
	}
}

func TestOnlineCharacterConfigSynchronizesToHost(t *testing.T) {
	hostGame := New()
	if err := hostGame.StartNetHost("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer hostGame.CloseNetplay()
	clientGame := New()
	if err := clientGame.StartNetClient(hostGame.netplay.listener.Addr().String()); err != nil {
		t.Fatal(err)
	}
	defer clientGame.CloseNetplay()
	waitFor(t, "client player id", func() bool { return clientGame.netplay.assignedPlayer() == 2 })

	clientGame.customPlayers[1] = CustomPlayerConfig{Name: "Remote", Color: 9, Shirt: 7, Hat: 18, Eyes: 11, Gun: 231, Perk: 8, Type: 1, Team: 2}
	clientGame.sendOnlineLocalConfigIfChanged()
	waitFor(t, "host remote config", func() bool {
		cfg, ok := hostGame.netplay.latestConfigFor(2)
		return ok && cfg.Name == "Remote" && cfg.Color == 9 && cfg.Hat == 18 && cfg.Eyes == 11 && cfg.Gun == 231
	})
	hostGame.syncOnlineHostState()
	if got := hostGame.customPlayers[1]; got.Name != "Remote" || got.Color != 9 || got.Shirt != 7 || got.Hat != 18 || got.Eyes != 11 || got.Gun != 231 || got.Perk != 8 {
		t.Fatalf("host lobby config not synchronized: %+v", got)
	}
}

func TestOnlineHandgunArrowsCycleThroughFullStarterRoster(t *testing.T) {
	g := New()
	g.netplay = &netplaySession{mode: netplayHost}
	handguns := starterHandgunNumbers()
	if len(handguns) <= 6 {
		t.Fatalf("starter handgun roster has only %d entries", len(handguns))
	}
	g.customPlayers[0].Gun = 6
	g.adjustOnlineCharacter(onlineCharGun, 1)
	if got := g.customPlayers[0].Gun; got != handguns[6] {
		t.Fatalf("next handgun after base roster=%d want%d", got, handguns[6])
	}
	g.adjustOnlineCharacter(onlineCharGun, -1)
	if got := g.customPlayers[0].Gun; got != 6 {
		t.Fatalf("previous handgun did not return to base roster: %d", got)
	}
	g.customPlayers[0].Gun = handguns[len(handguns)-1]
	g.adjustOnlineCharacter(onlineCharGun, 1)
	if got := g.customPlayers[0].Gun; got != handguns[0] {
		t.Fatalf("handgun roster did not wrap: %d want%d", got, handguns[0])
	}
}

func TestEveryOnlineStarterHandgunTileCanBeSelected(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	g := New()
	g.netplay = host
	handguns := starterHandgunNumbers()
	pageSize := onlineCharacterVisualPageSize(onlineVisualCharacterGun)
	for absolute, want := range handguns {
		g.onlineVisualPicker = onlineVisualCharacterGun
		g.onlineVisualPage = absolute / pageSize
		visible := absolute % pageSize
		r := onlineVisualCharacterChoiceRect(onlineVisualCharacterGun, visible)
		hit := g.onlineVisualPickerHitAt(r.X+r.W/2, r.Y+r.H/2)
		index, ok := decodeOnlineVisualChoiceHit(hit)
		if !ok || index != visible {
			t.Fatalf("handgun %d tile hit=%d index=%d ok=%v", want, hit, index, ok)
		}
		g.applyOnlineVisualChoice(index)
		if got := g.customPlayers[0].Gun; got != want {
			t.Fatalf("handgun tile selected %d want%d", got, want)
		}
	}
}

func TestEveryOnlineWeaponBlocklistTileDecodesAcrossAllPages(t *testing.T) {
	g := New()
	weapons := onlineAllWeaponNumbers()
	if len(weapons) <= 100 {
		t.Fatalf("weapon list has only %d entries; test needs the formerly broken later pages", len(weapons))
	}
	for absolute, weapon := range weapons {
		g.onlineBlocklistPage = absolute / 24
		visible := absolute % 24
		col := visible % 4
		row := visible / 4
		r := Rect{X: 80 + float64(col)*190, Y: 125 + float64(row)*52, W: 175, H: 42}
		hit := g.onlineBlocklistHitAt(r.X+r.W/2, r.Y+r.H/2)
		index, ok := decodeOnlineBlockWeaponHit(hit)
		if !ok || index != absolute {
			t.Fatalf("weapon #%d blocklist tile hit=%d decoded index=%d ok=%v want index=%d", weapon, hit, index, ok, absolute)
		}
	}
}

func TestJoinInProgressAddsThirdNetworkPlayer(t *testing.T) {
	hostGame := New()
	if err := hostGame.StartNetHost("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer hostGame.CloseNetplay()
	client2, err := newClientNetplay(hostGame.netplay.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client2.Close()
	waitFor(t, "P2 assignment", func() bool { return client2.assignedPlayer() == 2 })

	hostGame.onlineMatchSetup = true
	hostGame.customPlayers[2].Type = 1 // P3 must explicitly be an ONLINE slot.
	hostGame.syncOnlineHostState()
	hostGame.customMap = 1
	hostGame.customMode = SourceGameModeNormal
	hostGame.customLives = 7
	hostGame.startCustomGame()
	hostGame.screen = screenGameplay
	hostGame.syncOnlineHostState()
	if len(hostGame.players) != 2 {
		t.Fatalf("initial online players=%d want2", len(hostGame.players))
	}

	client3, err := newClientNetplay(hostGame.netplay.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client3.Close()
	waitFor(t, "P3 assignment", func() bool { return client3.assignedPlayer() == 3 })
	hostGame.syncOnlineHostState()
	found3 := false
	for _, p := range hostGame.players {
		if p != nil && p.ID == 3 && p.RemoteControlled && !p.AI {
			found3 = true
		}
	}
	if !found3 {
		t.Fatalf("join-in-progress did not add P3: %+v", hostGame.players)
	}
}

func TestOnlineCustomGameSupportsTwoHumansVsTwoBots(t *testing.T) {
	hostGame := New()
	if err := hostGame.StartNetHost("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer hostGame.CloseNetplay()

	client, err := newClientNetplay(hostGame.netplay.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	waitFor(t, "P2 assignment", func() bool { return client.assignedPlayer() == 2 })

	hostGame.customOnlineMode = true
	hostGame.onlineMatchSetup = true
	hostGame.customPlayers[0].Type = 1
	hostGame.customPlayers[0].Team = 1
	hostGame.customPlayers[1].Type = 1
	hostGame.customPlayers[1].Team = 1
	hostGame.customPlayers[2].Type = 2
	hostGame.customPlayers[2].Team = 2
	hostGame.customPlayers[3].Type = 2
	hostGame.customPlayers[3].Team = 2
	hostGame.customMode = SourceGameModeTeams
	hostGame.customMap = 1
	hostGame.customLives = 5
	hostGame.syncOnlineHostState()
	hostGame.startCustomGame()

	if len(hostGame.players) != 4 {
		t.Fatalf("players=%d want4", len(hostGame.players))
	}
	byID := map[int]*Player{}
	for _, p := range hostGame.players {
		byID[p.ID] = p
	}
	if byID[1] == nil || byID[1].AI || byID[1].Team != 1 {
		t.Fatalf("P1=%+v", byID[1])
	}
	if byID[2] == nil || byID[2].AI || !byID[2].RemoteControlled || byID[2].Team != 1 {
		t.Fatalf("P2=%+v", byID[2])
	}
	if byID[3] == nil || !byID[3].AI || byID[3].RemoteControlled || byID[3].Team != 2 {
		t.Fatalf("P3 bot=%+v", byID[3])
	}
	if byID[4] == nil || !byID[4].AI || byID[4].RemoteControlled || byID[4].Team != 2 {
		t.Fatalf("P4 bot=%+v", byID[4])
	}
}

func TestOnlineRulesFilterWeaponsAndPerks(t *testing.T) {
	g := New()
	g.onlineWeaponMode = onlineWeaponsOnlyOne
	g.onlineOnlyWeapon = 67
	if pool := g.onlineCrateWeaponPool(); len(pool) != 1 || pool[0] != 67 {
		t.Fatalf("only-one weapon pool=%v", pool)
	}
	g.onlineWeaponMode = onlineWeaponsBlocklist
	g.onlineBlockedWeapons[10] = true
	for _, n := range g.onlineCrateWeaponPool() {
		if n == 10 {
			t.Fatal("blocked weapon 10 survived online crate pool")
		}
	}
	g.onlineWeaponMode = onlineWeaponsAll
	found10 := false
	for _, n := range g.onlineCrateWeaponPool() {
		if n == 10 {
			found10 = true
			break
		}
	}
	if !found10 {
		t.Fatal("ALL WEAPONS incorrectly kept stale blocklist")
	}
	g.customPlayers[1].Perk = 9
	g.onlinePerksEnabled = false
	p := g.newOnlinePlayer(2)
	if p.PerkNumber != 0 {
		t.Fatalf("perks disabled but player perk=%d", p.PerkNumber)
	}
	g.onlinePerksEnabled = true
	g.onlineBlockedPerks[9] = true
	p = g.newOnlinePlayer(2)
	if p.PerkNumber != 0 {
		t.Fatalf("blocked perk survived: %d", p.PerkNumber)
	}
}

func TestOnlineVisualCharacterPickerAppliesChoice(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	g := New()
	g.netplay = host
	g.onlineLobbyCustomize = true
	g.onlineVisualPicker = onlineVisualCharacterFace
	g.applyOnlineVisualChoice(10) // visible index 10 -> face 11
	if g.customPlayers[0].Eyes != 11 {
		t.Fatalf("visual face choice=%d want11", g.customPlayers[0].Eyes)
	}
	if g.onlineVisualPicker != onlineVisualNone {
		t.Fatalf("visual picker stayed open: %d", g.onlineVisualPicker)
	}

	g.onlineVisualPicker = onlineVisualCharacterGun
	g.applyOnlineVisualChoice(len(starterHandgunNumbers()) - 1)
	if g.customPlayers[0].Gun != 231 {
		t.Fatalf("visual developed handgun choice=%d want231", g.customPlayers[0].Gun)
	}
}

func TestOnlineVisualWeaponAndMapPickerUsePicturesByIndex(t *testing.T) {
	g := New()
	weapons := onlineAllWeaponNumbers()
	if len(weapons) < 30 {
		t.Fatalf("weapon catalog too small: %d", len(weapons))
	}
	g.onlineVisualPicker = onlineVisualOnlyWeapon
	g.onlineVisualPage = 1
	wantWeapon := weapons[24+3]
	g.applyOnlineVisualChoice(3)
	if g.onlineOnlyWeapon != wantWeapon {
		t.Fatalf("visual only-weapon=%d want%d", g.onlineOnlyWeapon, wantWeapon)
	}

	maps := onlineFixedMapNumbers()
	g.onlineVisualPicker = onlineVisualFixedMap
	g.onlineVisualPage = 1
	wantMap := maps[12+2]
	g.applyOnlineVisualChoice(2)
	if g.customMap != wantMap || g.onlineMapRotation != onlineMapFixed || g.customMapSetGM2 != isGM2MapID(wantMap) {
		t.Fatalf("visual fixed map=%d rotation=%d gm2=%v want map=%d", g.customMap, g.onlineMapRotation, g.customMapSetGM2, wantMap)
	}
}

func TestOnlineVisualHitAreasOpenPictureSelectors(t *testing.T) {
	g := New()
	for field := 0; field < onlineCharFieldCount; field++ {
		r := onlineCharacterFieldRect(field)
		if got := g.onlineCharacterHitAt(r.X+r.W/2, r.Y+r.H/2); got != onlineCharacterVisualFieldHit(field) {
			t.Fatalf("character field %d hit=%d", field, got)
		}
	}
	for mode := 0; mode < 4; mode++ {
		r := Rect{X: 60 + float64(mode)*195, Y: 125, W: 180, H: 72}
		if got := g.onlineRulesHitAt(r.X+r.W/2, r.Y+r.H/2); got != menuHitOnlineRuleMapChoiceBase+mode {
			t.Fatalf("map visual mode %d hit=%d", mode, got)
		}
	}
	for mode := 0; mode < 3; mode++ {
		r := Rect{X: 80 + float64(mode)*250, Y: 220, W: 230, H: 70}
		if got := g.onlineRulesHitAt(r.X+r.W/2, r.Y+r.H/2); got != menuHitOnlineRuleWeaponChoiceBase+mode {
			t.Fatalf("weapon visual mode %d hit=%d", mode, got)
		}
	}
}

func TestOnlineVisualPickerPagesCoverWholeCatalog(t *testing.T) {
	g := New()
	g.onlineVisualPicker = onlineVisualOnlyWeapon
	weapons := onlineAllWeaponNumbers()
	covered := 0
	for page := 0; page <= g.onlineVisualMaxPage(); page++ {
		g.onlineVisualPage = page
		start := page * g.onlineVisualPageSize()
		end := start + g.onlineVisualPageSize()
		if end > len(weapons) {
			end = len(weapons)
		}
		covered += end - start
	}
	if covered != len(weapons) {
		t.Fatalf("visual weapon pages cover %d want%d", covered, len(weapons))
	}
}

func TestOnlineVisualScreensRenderWithoutPanic(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	g := New()
	g.netplay = host
	g.onlinePlayerSlots[0] = true
	g.customPlayers[0].Type = 1
	screen := ebiten.NewImage(900, 600)

	g.onlineLobbyCustomize = true
	g.drawOnlineCharacterEditor(screen)
	for _, picker := range []int{
		onlineVisualCharacterColor,
		onlineVisualCharacterShirt,
		onlineVisualCharacterHat,
		onlineVisualCharacterFace,
		onlineVisualCharacterGun,
		onlineVisualCharacterPerk,
		onlineVisualCharacterTeam,
		onlineVisualOnlyWeapon,
		onlineVisualFixedMap,
	} {
		g.onlineVisualPicker = picker
		g.onlineVisualPage = 0
		g.drawOnlineVisualPicker(screen)
		if max := g.onlineVisualMaxPage(); max > 0 {
			g.onlineVisualPage = max
			g.drawOnlineVisualPicker(screen)
		}
	}
	g.onlineVisualPicker = onlineVisualNone
	g.onlineRulesOpen = true
	g.drawOnlineRules(screen)
	g.onlineBlocklistPage = 0
	g.drawOnlineBlocklist(screen)
	g.onlineBlocklistPage = g.onlineBlocklistMaxPage()
	g.drawOnlineBlocklist(screen)
	g.drawOnlinePerkBlocklist(screen)
}

func TestNetSnapshotMirrorsOnlineHostSetupToClient(t *testing.T) {
	hostGame := New()
	hostGame.screen = screenCustomGame
	hostGame.onlineMatchSetup = true
	hostGame.customPage = customPageMaps
	hostGame.customMenuX = 900
	hostGame.customMode = SourceGameModeGunGame
	hostGame.customMap = gm2MapID(3)
	hostGame.customMapSetGM2 = true
	hostGame.customLives = 5
	hostGame.onlineMapRotation = onlineMapRandomAll
	hostGame.onlineWeaponMode = onlineWeaponsOnlyOne
	hostGame.onlineOnlyWeapon = 67
	hostGame.onlinePowerupsEnabled = false
	hostGame.onlineBlockedWeapons[10] = true
	hostGame.onlineBlockedPerks[6] = true
	hostGame.customPlayers[2] = CustomPlayerConfig{Name: "P3 Config", Color: 8, Shirt: 4, Hat: 12, Eyes: 10, Gun: 5, Perk: 6, Type: 1, Team: 1}

	clientGame := New()
	clientGame.applyNetSnapshot(hostGame.makeNetSnapshot())
	if clientGame.screen != screenCustomGame || !clientGame.onlineMatchSetup || clientGame.customPage != customPageMaps {
		t.Fatalf("client did not mirror online setup screen: screen=%d online=%v page=%d", clientGame.screen, clientGame.onlineMatchSetup, clientGame.customPage)
	}
	if clientGame.customMap != gm2MapID(3) || !clientGame.customMapSetGM2 || clientGame.customMode != SourceGameModeGunGame || clientGame.customLives != 5 {
		t.Fatalf("client setup mismatch: map=%d gm2=%v mode=%d lives=%d", clientGame.customMap, clientGame.customMapSetGM2, clientGame.customMode, clientGame.customLives)
	}
	if clientGame.onlineMapRotation != onlineMapRandomAll || clientGame.onlineWeaponMode != onlineWeaponsOnlyOne || clientGame.onlineOnlyWeapon != 67 || clientGame.onlinePowerupsEnabled {
		t.Fatalf("client rules mismatch: rotation=%d weaponMode=%d only=%d power=%v", clientGame.onlineMapRotation, clientGame.onlineWeaponMode, clientGame.onlineOnlyWeapon, clientGame.onlinePowerupsEnabled)
	}
	if !clientGame.onlineBlockedWeapons[10] || !clientGame.onlineBlockedPerks[6] || clientGame.customPlayers[2].Name != "P3 Config" {
		t.Fatalf("client live restrictions/config not mirrored")
	}
}
