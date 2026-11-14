package game

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Online screens on the Mayhem UI kit (phase 5): server browser, host setup
// and the lobby for host and guests. Every test drives the real build
// function through its run*Update pass, so a control is only reachable where
// it is drawn.

type onlineKit struct {
	st  *uiState
	run func(uiInput)
}

func onlineBrowserKit(g *Game) onlineKit {
	return onlineKit{st: &g.onlineUI, run: g.runOnlineBrowserUpdate}
}

func onlineHostSetupKit(g *Game) onlineKit {
	return onlineKit{st: &g.onlineUI, run: g.runOnlineHostSetupUpdate}
}

func onlineLobbyKit(g *Game) onlineKit {
	return onlineKit{st: &g.customUI, run: g.runOnlineLobbyUpdate}
}

func (k onlineKit) item(t *testing.T, id string) Rect {
	t.Helper()
	it, ok := k.st.item(id)
	if !ok {
		var ids []string
		for _, it := range k.st.items {
			ids = append(ids, it.id)
		}
		t.Fatalf("kit item %q not on screen; have %v", id, ids)
	}
	return it.rect
}

func (k onlineKit) has(id string) bool {
	_, ok := k.st.item(id)
	return ok
}

func (k onlineKit) clickAt(x, y float64) {
	k.run(uiInput{X: x, Y: y, Moved: true})
	k.run(uiInput{X: x, Y: y, Pressed: true, Down: true})
	k.run(uiInput{X: x, Y: y, Released: true})
	k.run(uiInput{X: x, Y: y})
}

func (k onlineKit) click(t *testing.T, id string) {
	t.Helper()
	r := k.item(t, id)
	k.clickAt(r.X+r.W/2, r.Y+r.H/2)
}

func (k onlineKit) key(in uiInput) {
	in.X, in.Y = -10, -10
	k.run(in)
	k.run(uiInput{X: -10, Y: -10})
}

func (k onlineKit) uniqueIDs(t *testing.T, name string) {
	t.Helper()
	k.run(uiInput{X: -10, Y: -10})
	seen := map[string]bool{}
	for _, it := range k.st.items {
		if seen[it.id] {
			t.Fatalf("%s: duplicate kit id %q", name, it.id)
		}
		seen[it.id] = true
	}
	if len(seen) == 0 {
		t.Fatalf("%s: no kit items", name)
	}
}

func testOnlineServers(n int) []multiplayerServerInfo {
	out := []multiplayerServerInfo{
		{ID: "a", Name: "Open LAN", Address: "127.0.0.1:1", Sources: "LAN", Players: 1, MaxPlayers: 4,
			GameMode: SourceGameModeNormal, Lives: 10, Rounds: 2, Maps: [customSeriesMaxRounds]int{2, 7}},
		{ID: "b", Name: "Full one", Address: "127.0.0.1:2", Sources: "MASTER", Players: 4, MaxPlayers: 4},
	}
	for i := len(out); i < n; i++ {
		out = append(out, multiplayerServerInfo{ID: fmt.Sprint(i), Name: fmt.Sprintf("Server %d", i), Address: "127.0.0.1:1", Sources: "LAN", Players: 1, MaxPlayers: 4})
	}
	return out
}

func newOnlineBrowserGame(t *testing.T, servers int) *Game {
	t.Helper()
	g := New()
	g.screen = screenCustomGame
	g.customOnlineMode = true
	g.customNetworkOpen = true
	g.multiplayerSelectedServer = -1
	g.multiplayerDiscoveryDone = true
	g.multiplayerServers = testOnlineServers(servers)
	g.onlineEnterScreen(onlineScreenBrowser)
	g.runOnlineBrowserUpdate(uiInput{X: -10, Y: -10})
	return g
}

func newOnlineHostLobby(t *testing.T, access int) *Game {
	t.Helper()
	g := New()
	if err := g.StartNetHostConfig("127.0.0.1:0", onlineSessionScopeLAN, access); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = g.CloseNetplay() })
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.customNetworkOpen = false
	g.screen = screenCustomGame
	g.customPlayers[1].Type = 0
	g.customPlayers[2].Type = 0
	g.customPlayers[3].Type = 0
	g.setCustomV2Page(customPagePlayers)
	g.syncOnlineHostState()
	g.onlineEnterScreen(onlineScreenLobby)
	g.runOnlineLobbyUpdate(uiInput{X: -10, Y: -10})
	return g
}

func newOnlineGuestLobby(t *testing.T) *Game {
	t.Helper()
	g := New()
	g.netplay = &netplaySession{mode: netplayClient}
	g.netplay.assignedPlayerID.Store(2)
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.customNetworkOpen = false
	g.screen = screenCustomGame
	g.customPlayers[0].Type = 1
	g.customPlayers[1].Type = 1
	g.customPlayers[2].Type = 2
	g.customPlayers[3].Type = 0
	g.onlinePlayerSlots = [4]bool{true, true, false, false}
	g.setCustomV2Page(customPagePlayers)
	g.onlineEnterScreen(onlineScreenLobby)
	g.runOnlineLobbyUpdate(uiInput{X: -10, Y: -10})
	return g
}

// ---------------------------------------------------------------------------
// Server browser

func TestOnlineBrowserSelectAndJoinCallExistingActions(t *testing.T) {
	g := newOnlineBrowserGame(t, 2)
	k := onlineBrowserKit(g)

	// JOIN SESSION is disabled until a session is selected.
	k.click(t, "ob.joinsel")
	if g.pendingNetJoin != nil || g.netplay != nil {
		t.Fatal("disabled JOIN SESSION started a join")
	}

	k.click(t, onlineBrowserRowID(1))
	if g.multiplayerSelectedServer != 1 || g.netplay != nil || g.pendingNetJoin != nil {
		t.Fatalf("clicking a server must only preview it: selected=%d", g.multiplayerSelectedServer)
	}
	// The full session's JOIN is disabled: nothing changes.
	msg := g.multiplayerMessage
	k.click(t, "ob.joinsel")
	if g.pendingNetJoin != nil || g.multiplayerMessage != msg {
		t.Fatalf("full session JOIN acted: pending=%v msg=%q", g.pendingNetJoin != nil, g.multiplayerMessage)
	}

	k.click(t, onlineBrowserRowID(0))
	k.click(t, "ob.joinsel")
	if g.pendingNetJoin == nil || g.pendingNetJoin.target != "Open LAN" {
		t.Fatal("JOIN SESSION did not start the async join of the selected server")
	}
	if g.multiplayerAddress != "127.0.0.1:1" {
		t.Fatalf("join address=%q", g.multiplayerAddress)
	}
	g.cancelPendingNetJoin()
}

func TestOnlineBrowserJoinFieldAndButtons(t *testing.T) {
	g := newOnlineBrowserGame(t, 0)
	k := onlineBrowserKit(g)
	g.multiplayerAddress = ""

	k.click(t, "ob.join") // disabled while the field is empty
	if g.pendingNetJoin != nil || strings.Contains(g.multiplayerMessage, "ENTER A JOIN CODE") {
		t.Fatal("JOIN with an empty field acted")
	}
	k.click(t, "ob.addr")
	if !g.multiplayerFocus {
		t.Fatal("clicking the code field did not focus it")
	}
	k.clickAt(450, 120) // the list panel
	if g.multiplayerFocus {
		t.Fatal("clicking elsewhere kept the field focused")
	}
	g.multiplayerAddress = "127.0.0.1:1"
	k.run(uiInput{X: -10, Y: -10})
	k.click(t, "ob.join")
	if g.pendingNetJoin == nil {
		t.Fatal("JOIN did not start a join from the field")
	}
	g.cancelPendingNetJoin()
	if !k.has("ob.paste") || !k.has("ob.refresh") {
		t.Fatal("PASTE / REFRESH missing")
	}

	k.click(t, "ob.host")
	if !g.customHostSetupOpen || !g.customNetworkOpen {
		t.Fatal("HOST SESSION did not open the host setup")
	}

	g = newOnlineBrowserGame(t, 0)
	onlineBrowserKit(g).click(t, "ob.local")
	if g.customOnlineMode || g.customNetworkOpen || g.onlineMatchSetup {
		t.Fatal("LOCAL VERSUS did not switch to the local Custom Game")
	}

	g = newOnlineBrowserGame(t, 0)
	onlineBrowserKit(g).click(t, "ob.back")
	if g.customOnlineMode || !g.fadeActive || g.fadeTarget != screenMainMenu {
		t.Fatal("MAIN MENU did not leave online")
	}
}

func TestOnlineBrowserKeyboard(t *testing.T) {
	g := newOnlineBrowserGame(t, 12)
	k := onlineBrowserKit(g)
	g.onlineUI.focus = onlineBrowserRowID(0)
	k.key(uiInput{Enter: true})
	if g.multiplayerSelectedServer != 0 {
		t.Fatalf("ENTER on a row selected %d", g.multiplayerSelectedServer)
	}
	k.key(uiInput{DownKey: true})
	if g.onlineUI.focus != onlineBrowserRowID(1) {
		t.Fatalf("DOWN focus=%q", g.onlineUI.focus)
	}
	// Walking past the last visible row scrolls the list.
	visible := onlineBrowserVisibleRows()
	for i := 0; i < visible+2; i++ {
		k.key(uiInput{DownKey: true})
	}
	if g.multiplayerServerScroll == 0 {
		t.Fatal("keyboard did not scroll the server list")
	}
	k.key(uiInput{Up: true})
	if !strings.HasPrefix(g.onlineUI.focus, "ob.srv.") {
		t.Fatalf("UP left the list: %q", g.onlineUI.focus)
	}
	// Mouse wheel scrolls too, clamped.
	for i := 0; i < 50; i++ {
		k.run(uiInput{X: 100, Y: 200, Wheel: -1})
	}
	if max := len(g.multiplayerServers) - visible; g.multiplayerServerScroll != max {
		t.Fatalf("scroll=%d want max %d", g.multiplayerServerScroll, max)
	}

	k.key(uiInput{Esc: true})
	if g.customOnlineMode || g.fadeTarget != screenMainMenu {
		t.Fatal("ESC did not return to the main menu")
	}
}

// ---------------------------------------------------------------------------
// Host setup

func TestOnlineHostSetupCreateControls(t *testing.T) {
	g := newOnlineBrowserGame(t, 0)
	g.customHostSetupOpen = true
	g.onlineEnterScreen(onlineScreenHostSetup)
	k := onlineHostSetupKit(g)
	k.run(uiInput{X: -10, Y: -10})

	k.click(t, "oh.reach.1")
	if g.customHostScope != onlineSessionScopeInternet {
		t.Fatalf("INTERNET tile scope=%d", g.customHostScope)
	}
	k.click(t, "oh.reach.0")
	if g.customHostScope != onlineSessionScopeLAN {
		t.Fatalf("LAN tile scope=%d", g.customHostScope)
	}
	k.click(t, "oh.access.1")
	if g.customHostAccess != onlineSessionInviteOnly {
		t.Fatal("INVITE CODE tile did not select invite access")
	}
	k.click(t, "oh.access.0")
	if g.customHostAccess != onlineSessionOpen {
		t.Fatal("OPEN tile did not select open access")
	}
	k.click(t, "oh.name")
	if !g.multiplayerNameFocus {
		t.Fatal("name field did not take focus")
	}
	k.clickAt(800, 300)
	if g.multiplayerNameFocus {
		t.Fatal("name field kept focus")
	}
	if !k.has("oh.create") {
		t.Fatal("CREATE SESSION missing")
	}
	k.key(uiInput{Esc: true})
	if g.customHostSetupOpen || !g.customNetworkOpen {
		t.Fatal("ESC did not return to the server browser")
	}
}

func TestOnlineHostSetupEditKeepsHostAndLocksSettings(t *testing.T) {
	g := newOnlineHostLobby(t, onlineSessionOpen)
	host := g.netplay
	g.customLocalActivate(menuHitCustomV2TabSession) // SESSION SETTINGS
	if !g.customNetworkOpen || !g.customHostSetupOpen {
		t.Fatal("SESSION SETTINGS did not open the editor")
	}
	g.onlineEnterScreen(onlineScreenHostSetup)
	k := onlineHostSetupKit(g)
	k.run(uiInput{X: -10, Y: -10})
	scope, access := g.customHostScope, g.customHostAccess
	k.click(t, "oh.reach.1")
	k.click(t, "oh.access.1")
	k.click(t, "oh.name")
	if g.customHostScope != scope || g.customHostAccess != access || g.multiplayerNameFocus {
		t.Fatal("running-session settings changed through disabled controls")
	}
	if !k.has("oh.leave") || !k.has("oh.cancel") {
		t.Fatal("LEAVE / CANCEL missing while editing")
	}
	k.key(uiInput{Esc: true}) // ESC is CANCEL: the host keeps running
	if g.netplay != host || g.customNetworkOpen || !g.onlineSessionTab {
		t.Fatalf("ESC while editing: hostSame=%v network=%v sessionTab=%v", g.netplay == host, g.customNetworkOpen, g.onlineSessionTab)
	}
}

// ---------------------------------------------------------------------------
// Lobby: host

func TestOnlineLobbyHostTabsAndKeyboard(t *testing.T) {
	g := newOnlineHostLobby(t, onlineSessionInviteOnly)
	k := onlineLobbyKit(g)
	if g.onlineLobbyTab() != onlineLobbyTabPlayers {
		t.Fatalf("tab=%d want PLAYERS", g.onlineLobbyTab())
	}
	k.click(t, "on.tabs/SESSION")
	if !g.onlineSessionTab || g.onlineLobbyTab() != onlineLobbyTabSession {
		t.Fatal("SESSION tab not shown")
	}
	k.click(t, "on.tabs/MATCH")
	if g.onlineSessionTab || g.customPage != customPageGame {
		t.Fatalf("MATCH tab page=%d", g.customPage)
	}
	k.key(uiInput{NextTab: true})
	if g.customPage != customPageMaps {
		t.Fatalf("E: page=%d want MAPS", g.customPage)
	}
	k.key(uiInput{PrevTab: true})
	k.key(uiInput{PrevTab: true})
	if g.onlineLobbyTab() != onlineLobbyTabSession {
		t.Fatalf("Q Q: tab=%d want SESSION", g.onlineLobbyTab())
	}

	// MATCH is editable for the host and writes through the wizard codes.
	g.setOnlineLobbyTab(onlineLobbyTabMatch)
	k.run(uiInput{X: -10, Y: -10})
	k.click(t, fmt.Sprintf("cust.mode.%d", SourceGameModeTeams))
	if g.customMode != SourceGameModeTeams {
		t.Fatal("host could not change the mode")
	}
	k.click(t, "on.rules")
	if !g.onlineRulesOpen {
		t.Fatal("CONFIGURE did not open the online rules")
	}
}

func TestOnlineLobbyHostSlotsAndStart(t *testing.T) {
	g := newOnlineHostLobby(t, onlineSessionInviteOnly)
	k := onlineLobbyKit(g)

	// Alone: START MATCH is disabled and does nothing.
	k.click(t, "on.start")
	if g.fadeActive || g.screen != screenCustomGame {
		t.Fatal("disabled START MATCH started")
	}

	k.click(t, "on.p1.open")
	if g.customPlayers[1].Type != 1 {
		t.Fatalf("OPEN ONLINE SLOT type=%d", g.customPlayers[1].Type)
	}
	k.run(uiInput{X: -10, Y: -10})
	// A waiting remote slot belongs to its future player: no host loadout.
	k.click(t, "on.p1.edit")
	if g.customLoadoutSlot != 0 {
		t.Fatal("host opened a remote player's loadout")
	}
	k.click(t, "on.p1.remove") // CLOSE SLOT
	if g.customPlayers[1].Type != 0 {
		t.Fatal("CLOSE SLOT did not close the slot")
	}

	k.click(t, "on.p2.addai")
	if g.customPlayers[2].Type != 2 {
		t.Fatal("ADD AI BOT failed")
	}
	k.click(t, "on.p2.edit")
	if g.customLoadoutSlot != 3 {
		t.Fatalf("bot loadout not opened: slot=%d", g.customLoadoutSlot)
	}
	k.key(uiInput{Esc: true}) // DONE
	if g.customLoadoutSlot != 0 || g.onlineLeaveConfirm {
		t.Fatal("ESC in the loadout should only close it")
	}

	k.click(t, "on.start")
	if !g.fadeActive || g.fadeTarget != screenGameplay {
		t.Fatal("START MATCH with a bot did not start")
	}
	if len(g.players) != 2 || !g.players[1].AI {
		t.Fatalf("players=%d", len(g.players))
	}
}

func TestOnlineLobbyHostRemovesBot(t *testing.T) {
	g := newOnlineHostLobby(t, onlineSessionOpen)
	k := onlineLobbyKit(g)
	k.click(t, "on.p3.addai")
	k.click(t, "on.p3.remove")
	if g.customPlayers[3].Type != 0 {
		t.Fatal("REMOVE BOT failed")
	}
}

func TestOnlineLobbyLeaveConfirm(t *testing.T) {
	g := newOnlineHostLobby(t, onlineSessionOpen)
	k := onlineLobbyKit(g)
	host := g.netplay
	k.key(uiInput{Esc: true})
	if !g.onlineLeaveConfirm || g.netplay != host {
		t.Fatal("ESC must ask before closing the session")
	}
	k.click(t, "on.confirm.stay")
	if g.onlineLeaveConfirm || g.netplay != host {
		t.Fatal("STAY closed the session")
	}
	k.click(t, "on.leave")
	if !g.onlineLeaveConfirm {
		t.Fatal("LEAVE SESSION did not ask")
	}
	k.key(uiInput{Esc: true})
	if g.onlineLeaveConfirm {
		t.Fatal("ESC did not dismiss the confirmation")
	}
	k.click(t, "on.leave")
	k.click(t, "on.confirm.leave")
	if g.netplay != nil || !g.customNetworkOpen {
		t.Fatal("CLOSE SESSION did not close the host and open the browser")
	}
}

func TestOnlineLobbySessionStepsAndCopyLink(t *testing.T) {
	g := newOnlineHostLobby(t, onlineSessionInviteOnly)
	g.onlineSessionTab = true
	k := onlineLobbyKit(g)
	k.run(uiInput{X: -10, Y: -10})

	steps := g.onlineHostSteps()
	if steps[0].state != onlineStepDone || steps[2].state == onlineStepDone {
		t.Fatalf("steps=%+v", steps)
	}
	code, ready := g.currentHostDisplayJoinCode()
	it, _ := g.customUI.item("on.copybig")
	if it.disabled != (!ready || code == "") {
		t.Fatalf("COPY LINK disabled=%v ready=%v", it.disabled, ready)
	}
	if ready && steps[1].state != onlineStepDone {
		t.Fatalf("code ready but step 2=%+v", steps[1])
	}

	// Internet invite: tunnel starting -> error -> ready.
	key := g.onlineSessionKey
	g.onlineSessionScope = onlineSessionScopeInternet
	g.cloudflareTunnel = &cloudflareQuickTunnel{maxAttempts: 3, attempt: 1}
	if s := g.onlineHostSteps()[1]; s.state != onlineStepActive {
		t.Fatalf("starting tunnel step=%+v", s)
	}
	k.run(uiInput{X: -10, Y: -10})
	if it, _ := g.customUI.item("on.copybig"); !it.disabled {
		t.Fatal("COPY LINK enabled before the tunnel is ready")
	}

	g.cloudflareTunnel = &cloudflareQuickTunnel{lastErr: errors.New("cloudflared exited")}
	if s := g.onlineHostSteps()[1]; s.state != onlineStepError {
		t.Fatalf("tunnel error step=%+v", s)
	}
	k.run(uiInput{X: -10, Y: -10})
	if !k.has("on.restart") {
		t.Fatal("tunnel error offers no RESTART SESSION")
	}

	tun := &cloudflareQuickTunnel{wsURL: "wss://brave-otter.trycloudflare.com/netplay"}
	tun.publicReady.Store(true)
	tun.running.Store(true)
	g.cloudflareTunnel = tun
	g.onlineSessionKey = key
	if s := g.onlineHostSteps()[1]; s.state != onlineStepDone || s.title != "INVITE LINK READY" {
		t.Fatalf("ready step=%+v", s)
	}
	k.run(uiInput{X: -10, Y: -10})
	if it, _ := g.customUI.item("on.copybig"); it.disabled {
		t.Fatal("COPY LINK disabled with the link ready")
	}
	g.cloudflareTunnel = nil
	g.onlineSessionScope = onlineSessionScopeLAN
}

func TestOnlineLobbyOpenSessionHasNoLink(t *testing.T) {
	g := newOnlineHostLobby(t, onlineSessionOpen)
	g.onlineSessionTab = true
	k := onlineLobbyKit(g)
	k.run(uiInput{X: -10, Y: -10})
	if s := g.onlineHostSteps()[1]; s.state != onlineStepDone {
		t.Fatalf("open session step 2=%+v", s)
	}
	it, _ := g.customUI.item("on.copybig")
	if !it.disabled {
		t.Fatal("open session offers a link to copy")
	}
	k.click(t, "on.settings")
	if !g.customHostSetupOpen {
		t.Fatal("SESSION SETTINGS did not open the editor")
	}
}

func TestOnlineLobbyShowsConnectedPlayerAndPing(t *testing.T) {
	g := newOnlineHostLobby(t, onlineSessionOpen)
	g.customPlayers[1].Type = 1
	g.syncOnlineHostState()
	client, err := newClientNetplay(g.netplay.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	waitFor(t, "connected player 2", func() bool { return g.netplay.connectedCount() == 1 })
	g.syncOnlineHostState()
	status, _, ping := g.onlineSlotStatus(1)
	if status != "CONNECTED" || ping == "" {
		t.Fatalf("slot 2 status=%q ping=%q", status, ping)
	}
	if s := g.onlineHostSteps()[2]; s.state != onlineStepDone {
		t.Fatalf("players step=%+v", s)
	}
	k := onlineLobbyKit(g)
	k.run(uiInput{X: -10, Y: -10})
	if it, _ := g.customUI.item("on.p1.edit"); !it.disabled {
		t.Fatal("host may edit a connected player's loadout")
	}
	if it, _ := g.customUI.item("on.p1.name"); !it.disabled {
		t.Fatal("host may edit a connected player's name")
	}
	if !k.has("on.p1.remove") {
		t.Fatal("no KICK PLAYER button")
	}
}

func TestOnlineRulesModals(t *testing.T) {
	g := newOnlineHostLobby(t, onlineSessionOpen)
	k := onlineLobbyKit(g)
	g.setOnlineLobbyTab(onlineLobbyTabMatch)
	k.run(uiInput{X: -10, Y: -10})
	k.click(t, "on.rules")

	seg := k.item(t, "on.rules.maprot")
	k.clickAt(seg.X+seg.W*3/8, seg.Y+seg.H/2) // RANDOM GM1
	if g.onlineMapRotation != onlineMapRandomGM1 {
		t.Fatalf("map rotation=%d", g.onlineMapRotation)
	}
	k.clickAt(seg.X+10, seg.Y+seg.H/2) // FIXED MAP
	k.click(t, "on.rules.fixedmap")
	if g.onlineVisualPicker != onlineVisualFixedMap {
		t.Fatal("fixed map picker not open")
	}
	pick := onlineFixedMapNumbers()[2]
	k.click(t, fmt.Sprintf("on.pick.%d", pick))
	if g.customMap != pick || g.onlineVisualPicker != onlineVisualNone || !g.onlineRulesOpen {
		t.Fatalf("map pick: map=%d picker=%d", g.customMap, g.onlineVisualPicker)
	}

	seg = k.item(t, "on.rules.weapon")
	k.clickAt(seg.X+seg.W*5/6, seg.Y+seg.H/2) // BLOCKLIST
	if g.onlineWeaponMode != onlineWeaponsBlocklist {
		t.Fatalf("weapon mode=%d", g.onlineWeaponMode)
	}
	k.click(t, "on.rules.blocklist")
	if !g.onlineBlocklistOpen {
		t.Fatal("blocklist not open")
	}
	n := onlineAllWeaponNumbers()[1]
	k.click(t, fmt.Sprintf("on.block.%d", n))
	if !g.onlineBlockedWeapons[n] {
		t.Fatal("weapon tile did not block")
	}
	k.click(t, "on.page.next")
	if g.onlineBlocklistPage != 1 {
		t.Fatalf("blocklist page=%d", g.onlineBlocklistPage)
	}
	k.key(uiInput{Esc: true})
	if g.onlineBlocklistOpen || !g.onlineRulesOpen {
		t.Fatal("ESC did not return from the blocklist to the rules")
	}

	seg = k.item(t, "on.rules.weapon")
	k.clickAt(seg.X+seg.W/2, seg.Y+seg.H/2) // ONLY ONE
	k.click(t, "on.rules.onlyweapon")
	if g.onlineVisualPicker != onlineVisualOnlyWeapon {
		t.Fatal("only-weapon picker not open")
	}
	w := onlineAllWeaponNumbers()[3]
	k.click(t, fmt.Sprintf("on.pick.%d", w))
	if g.onlineOnlyWeapon != w {
		t.Fatalf("only weapon=%d want %d", g.onlineOnlyWeapon, w)
	}

	perks := g.onlinePerksEnabled
	seg = k.item(t, "on.rules.perks")
	k.clickAt(seg.X+10, seg.Y+seg.H/2) // OFF
	if g.onlinePerksEnabled == perks && perks {
		t.Fatal("perks toggle did nothing")
	}
	k.click(t, "on.rules.perkblock")
	if !g.onlinePerkBlocklistOpen {
		t.Fatal("perk blocklist not open")
	}
	k.click(t, "on.perkblock.3")
	k.click(t, "on.abilityblock.2")
	if !g.onlineBlockedPerks[3] || !g.onlineBlockedAbilities[2] {
		t.Fatal("perk / ability tiles did not block")
	}
	k.click(t, "on.perkblock.back")
	k.click(t, "on.rules.done")
	if g.onlineRulesOpen || g.customOnlineRulesOverlayActive() {
		t.Fatal("DONE did not close the rules")
	}
}

// ---------------------------------------------------------------------------
// Lobby: guest

func onlineGuestMatchState(g *Game) []any {
	return []any{g.customMode, g.customLives, g.customSeriesRounds, g.CrateON, g.PowerON, g.customRulesOpen, g.customChaosOpen,
		g.onlineRulesOpen, g.customSeriesMaps, g.customSeriesPlaylistConfigured, g.customPlaylistGM2, g.customLivesFocus,
		g.customRuleEnabled, g.customChaosEnabled, g.customPlayers[0], g.customPlayers[2], g.customPlayers[3], g.customPage}
}

func TestOnlineGuestMatchAndMapsAreReadOnly(t *testing.T) {
	g := newOnlineGuestLobby(t)
	k := onlineLobbyKit(g)
	for _, tab := range []string{"MATCH", "MAPS"} {
		k.click(t, "on.tabs/"+tab)
		if g.customPage != customPagePlayers {
			t.Fatal("a guest's tab changed its (host-owned) page")
		}
		before := onlineGuestMatchState(g)
		var targets []uiItem
		for _, it := range g.customUI.items {
			if strings.HasPrefix(it.id, "cust.") {
				targets = append(targets, it)
				if !it.disabled || it.focusable {
					t.Fatalf("%s: guest control %q is interactive", tab, it.id)
				}
			}
		}
		if len(targets) < 5 {
			t.Fatalf("%s: only %d match controls drawn", tab, len(targets))
		}
		for _, it := range targets {
			k.clickAt(it.rect.X+it.rect.W/2, it.rect.Y+it.rect.H/2)
			k.clickAt(it.rect.X+4, it.rect.Y+it.rect.H/2)
			k.clickAt(it.rect.X+it.rect.W-4, it.rect.Y+it.rect.H/2)
		}
		// Keyboard: the read-only controls are not focusable, so Tab and the
		// arrows only ever reach the tabs and the action bar.
		for _, in := range []uiInput{{Tab: true}, {Left: true}, {Right: true}, {DownKey: true}, {Tab: true}} {
			k.key(in)
			if f := g.customUI.focus; strings.HasPrefix(f, "cust.") {
				t.Fatalf("%s: keyboard focused read-only %q", tab, f)
			}
		}
		if after := onlineGuestMatchState(g); !reflect.DeepEqual(before, after) {
			t.Fatalf("%s: guest changed match state\nbefore %v\nafter  %v", tab, before, after)
		}
	}
	// Even a forged match code is ignored for a guest.
	before := onlineGuestMatchState(g)
	g.customLocalActivate(menuHitModeBase + SourceGameModeTeams)
	g.customLocalActivate(menuHitModeCrates)
	g.customLocalActivate(menuHitCustomSessionRules)
	g.customLocalActivate(playerSlotHit(3, playerSlotActionAI))
	if after := onlineGuestMatchState(g); !reflect.DeepEqual(before, after) {
		t.Fatal("guest activation changed host-owned state")
	}
}

func TestOnlineGuestEditsOnlyOwnSlot(t *testing.T) {
	g := newOnlineGuestLobby(t)
	k := onlineLobbyKit(g)
	for _, id := range []string{"on.p0.edit", "on.p2.edit", "on.p0.name", "on.p2.name"} {
		if it, _ := g.customUI.item(id); !it.disabled {
			t.Fatalf("guest control %q is enabled", id)
		}
	}
	for _, id := range []string{"on.p1.remove", "on.p3.addai", "on.p3.open", "on.copy", "on.settings"} {
		if k.has(id) {
			t.Fatalf("guest sees host control %q", id)
		}
	}
	if it, _ := g.customUI.item("on.start"); !it.disabled {
		t.Fatal("guest START is enabled")
	}
	k.click(t, "on.p0.edit")
	if g.customLoadoutSlot != 0 {
		t.Fatal("guest opened the host's loadout")
	}
	host := g.customPlayers[0]
	g.customLocalActivate(customSelectHit(0, customEditGun, 4))
	if g.customPlayers[0] != host {
		t.Fatal("guest changed the host's loadout through a forged code")
	}

	k.click(t, "on.p1.name")
	if g.customNameFocus != 2 {
		t.Fatalf("guest name focus=%d", g.customNameFocus)
	}
	k.clickAt(5, 300)
	k.click(t, "on.p1.edit")
	if g.customLoadoutSlot != 2 {
		t.Fatalf("guest loadout slot=%d", g.customLoadoutSlot)
	}
	g.customLoadoutTab = loadoutTabWeapon
	k.run(uiInput{X: -10, Y: -10})
	gun := starterHandgunNumbers()[3]
	k.click(t, fmt.Sprintf("lo.gun.%d", gun))
	if g.customPlayers[1].Gun != gun || g.customPlayers[1].Type != 1 || !g.onlineHasSentConfig {
		t.Fatalf("guest loadout pick: gun=%d sent=%v", g.customPlayers[1].Gun, g.onlineHasSentConfig)
	}
	if it, _ := g.customUI.item("lo.player"); !it.disabled {
		t.Fatal("a guest can switch to other players' loadouts")
	}
	k.key(uiInput{Esc: true})
	if g.customLoadoutSlot != 0 || g.netplay == nil {
		t.Fatal("ESC in the guest loadout should only close it")
	}
}

func TestOnlineGuestSessionTabAndLeave(t *testing.T) {
	g := newOnlineGuestLobby(t)
	k := onlineLobbyKit(g)
	k.click(t, "on.tabs/SESSION")
	if g.onlineLobbyTab() != onlineLobbyTabSession {
		t.Fatal("guest SESSION tab not shown")
	}
	k.click(t, "on.myloadout")
	if g.customLoadoutSlot != 2 || g.onlineLobbyTab() != onlineLobbyTabPlayers {
		t.Fatalf("EDIT MY LOADOUT slot=%d tab=%d", g.customLoadoutSlot, g.onlineLobbyTab())
	}
	k.key(uiInput{Esc: true})
	k.key(uiInput{Esc: true})
	if !g.onlineLeaveConfirm || g.netplay == nil {
		t.Fatal("guest ESC must ask before leaving")
	}
	k.click(t, "on.confirm.leave")
	if g.netplay != nil || !g.customNetworkOpen || g.multiplayerMessage != "DISCONNECTED" {
		t.Fatalf("guest LEAVE: netplay=%v network=%v msg=%q", g.netplay != nil, g.customNetworkOpen, g.multiplayerMessage)
	}
}

func TestOnlineClientUpdateRunsTheLobby(t *testing.T) {
	g := newOnlineGuestLobby(t)
	g.customUI.reset()
	g.updateOnlineClientCustomGame()
	if _, ok := g.customUI.item("on.p1.edit"); !ok {
		t.Fatal("the guest frame did not build the lobby")
	}
}

// ---------------------------------------------------------------------------
// Every screen

func TestOnlineKitIDsAreUnique(t *testing.T) {
	g := newOnlineBrowserGame(t, 9)
	g.multiplayerSelectedServer = 0
	onlineBrowserKit(g).uniqueIDs(t, "browser")
	g.customHostSetupOpen = true
	onlineHostSetupKit(g).uniqueIDs(t, "host setup")

	h := newOnlineHostLobby(t, onlineSessionInviteOnly)
	h.customPlayers[1].Type = 1
	h.customPlayers[2].Type = 2
	k := onlineLobbyKit(h)
	for tab := onlineLobbyTabSession; tab <= onlineLobbyTabPlayers; tab++ {
		h.setOnlineLobbyTab(tab)
		k.uniqueIDs(t, fmt.Sprintf("host tab %d", tab))
	}
	h.customMode = SourceGameModeTeams
	k.uniqueIDs(t, "host players teams")
	h.openCustomLoadout(0)
	k.uniqueIDs(t, "host loadout")
	h.closeCustomLoadout()
	h.setOnlineLobbyTab(onlineLobbyTabMatch)
	h.onlineRulesOpen = true
	k.uniqueIDs(t, "rules")
	h.onlineBlocklistOpen = true
	k.uniqueIDs(t, "weapon blocklist")
	h.onlineBlocklistOpen = false
	h.onlinePerkBlocklistOpen = true
	k.uniqueIDs(t, "perk blocklist")
	h.onlinePerkBlocklistOpen = false
	h.onlineVisualPicker = onlineVisualFixedMap
	k.uniqueIDs(t, "map picker")
	h.onlineVisualPicker = onlineVisualOnlyWeapon
	k.uniqueIDs(t, "weapon picker")
	h.onlineVisualPicker = onlineVisualNone
	h.onlineRulesOpen = false
	h.customRulesOpen = true
	k.uniqueIDs(t, "modifiers")
	h.customRulesOpen = false
	h.onlineLeaveConfirm = true
	k.uniqueIDs(t, "leave confirm")

	c := newOnlineGuestLobby(t)
	kc := onlineLobbyKit(c)
	for tab := onlineLobbyTabSession; tab <= onlineLobbyTabPlayers; tab++ {
		c.setOnlineLobbyTab(tab)
		kc.uniqueIDs(t, fmt.Sprintf("guest tab %d", tab))
	}
}

func TestOnlineScreensDraw(t *testing.T) {
	screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
	g := newOnlineBrowserGame(t, 3)
	g.multiplayerSelectedServer = 0
	g.multiplayerMessage = "JOIN ERROR: something long went wrong while connecting to the host, please try again later"
	g.drawCustomGameMenu(screen)
	g.pendingNetJoin = &pendingNetJoin{target: "x", done: make(chan netJoinResult)}
	g.drawCustomGameMenu(screen)
	g.pendingNetJoin = nil
	g.customHostSetupOpen = true
	g.multiplayerMessage = "HOST ERROR: listen tcp :7777: bind: address already in use"
	g.drawCustomGameMenu(screen)

	h := newOnlineHostLobby(t, onlineSessionInviteOnly)
	for tab := onlineLobbyTabSession; tab <= onlineLobbyTabPlayers; tab++ {
		h.setOnlineLobbyTab(tab)
		h.drawCustomGameMenu(screen)
	}
	h.onlineRulesOpen = true
	h.drawCustomGameMenu(screen)
	h.onlinePerkBlocklistOpen = true
	h.drawCustomGameMenu(screen)

	c := newOnlineGuestLobby(t)
	for tab := onlineLobbyTabSession; tab <= onlineLobbyTabPlayers; tab++ {
		c.setOnlineLobbyTab(tab)
		c.drawCustomGameMenu(screen)
	}
	c.drawNetClientWaitingScreen(screen)
	c.drawNetPingOverlay(screen)
}

func TestOnlineMessageKinds(t *testing.T) {
	cases := map[string]uiToastKind{
		"JOIN ERROR: timeout":            uiToastError,
		"HOST DISCONNECTED: closed":      uiToastError,
		"COPIED TO CLIPBOARD!":           uiToastOK,
		"CHOOSE A SESSION OR CREATE ONE": uiToastInfo,
	}
	for msg, want := range cases {
		if got := onlineMessageKind(msg); got != want {
			t.Fatalf("%q kind=%d want %d", msg, got, want)
		}
	}
}
