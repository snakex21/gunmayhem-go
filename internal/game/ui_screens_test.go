package game

import (
	"strings"
	"testing"
)

// Phase 8: every Mayhem UI screen state registers unique control ids, each id
// carries its screen's prefix (so an id names one control across all screens),
// each screen answers ESC and has keyboard focus, and F1 outlines the
// registry of whatever screen is shown.

type kitScreenCase struct {
	name     string
	prefixes []string
	setup    func(t *testing.T) (*Game, kitScreen)
}

func allKitScreenCases() []kitScreenCase {
	idle := func(g *Game, st *uiState, run func(uiInput)) (*Game, kitScreen) {
		return g, kitScreen{st: st, run: run}
	}
	custom := func(page int) func(t *testing.T) (*Game, kitScreen) {
		return func(t *testing.T) (*Game, kitScreen) {
			g := newCustomLocalGame(t)
			g.setCustomV2Page(page)
			return idle(g, &g.customUI, g.runCustomLocalUpdate)
		}
	}
	campaign := func(set int, briefing bool, loadout bool) func(t *testing.T) (*Game, kitScreen) {
		return func(t *testing.T) (*Game, kitScreen) {
			g, k := newCampaignUIGame(t, set)
			if briefing {
				g.openCampaignMission(g.campaignNextMission(set))
			}
			if loadout {
				g.openCampaignLoadout(0)
			}
			return g, k
		}
	}
	library := func(view int) func(t *testing.T) (*Game, kitScreen) {
		return func(t *testing.T) (*Game, kitScreen) {
			g, k := newLibraryGame(t)
			g.setGunLibraryView(view)
			return g, k
		}
	}
	return []kitScreenCase{
		{"main menu", []string{"main."}, func(t *testing.T) (*Game, kitScreen) {
			g := newMainMenuGame()
			return idle(g, &g.mainMenuUI, g.runMainMenuUpdate)
		}},
		{"options", []string{"opt."}, func(t *testing.T) (*Game, kitScreen) {
			g := New()
			g.screen = screenOptions
			return idle(g, &g.optionsUI, g.runOptionsUpdate)
		}},
		{"custom match", []string{"cust."}, custom(customPageGame)},
		{"custom maps", []string{"cust."}, custom(customPageMaps)},
		{"custom players", []string{"cust."}, custom(customPagePlayers)},
		{"custom loadout", []string{"lo."}, func(t *testing.T) (*Game, kitScreen) {
			g := newLoadoutGame(t, 0)
			return idle(g, &g.customUI, g.runCustomLocalUpdate)
		}},
		{"online browser", []string{"ob."}, func(t *testing.T) (*Game, kitScreen) {
			g := newOnlineBrowserGame(t, 3)
			return idle(g, &g.onlineUI, g.runOnlineBrowserUpdate)
		}},
		{"online guest lobby", []string{"on.", "cust."}, func(t *testing.T) (*Game, kitScreen) {
			g := newOnlineGuestLobby(t)
			return idle(g, &g.customUI, g.runOnlineLobbyUpdate)
		}},
		{"zombie setup", []string{"zs."}, func(t *testing.T) (*Game, kitScreen) {
			g := New()
			g.screen = screenZombieSetup
			return idle(g, &g.zombieUI, g.runZombieSetupUpdate)
		}},
		{"campaign GM1 list", []string{"camp."}, campaign(campaignSetGM1, false, false)},
		{"campaign GM2 list", []string{"camp."}, campaign(campaignSetGM2, false, false)},
		{"campaign EXTRA list", []string{"camp."}, campaign(campaignSetExtra, false, false)},
		{"campaign briefing", []string{"camp."}, campaign(campaignSetExtra, true, false)},
		{"campaign loadout", []string{"lo."}, campaign(campaignSetGM2, true, true)},
		{"library weapons", []string{"lib."}, library(gunLibraryViewWeapons)},
		{"library throwables", []string{"lib."}, library(gunLibraryViewThrowables)},
		{"library gear", []string{"lib."}, library(gunLibraryViewGear)},
		{"library powerups", []string{"lib."}, library(gunLibraryViewPowerups)},
		{"pause", []string{"pause."}, func(t *testing.T) (*Game, kitScreen) {
			g, k := newPausedMatch(t)
			return g, k
		}},
		{"pause options", []string{"pause."}, func(t *testing.T) (*Game, kitScreen) {
			g, k := newPausedMatch(t)
			g.pauseOptions = true
			return g, k
		}},
		{"guest menu", []string{"guest."}, func(t *testing.T) (*Game, kitScreen) {
			g, _ := newPausedMatch(t)
			g.paused = false
			g.netplay = &netplaySession{mode: netplayClient}
			g.openGuestMenu()
			return idle(g, &g.pauseUI, g.runGuestMenuUpdate)
		}},
		{"post game", []string{"res."}, func(t *testing.T) (*Game, kitScreen) {
			g, k := newPostGame(t, 3)
			return g, k
		}},
		{"series summary", []string{"res."}, func(t *testing.T) (*Game, kitScreen) {
			g, k := newPostGame(t, 3)
			g.screen = screenSeriesSummary
			return g, k
		}},
		{"credits", []string{"cred."}, func(t *testing.T) (*Game, kitScreen) {
			g := New()
			g.screen = screenCredits
			return idle(g, &g.creditsUI, g.runCreditsUpdate)
		}},
	}
}

// Ids shared by kit widgets themselves (modal blocker).
func kitInternalID(id string) bool { return strings.HasPrefix(id, "__") }

func TestKitScreensHaveUniquePrefixedIDsAndKeyboard(t *testing.T) {
	for _, c := range allKitScreenCases() {
		g, k := c.setup(t)
		k.t = t
		k.idle()
		k.uniqueIDs(c.name)
		focusable := 0
		for _, it := range k.st.items {
			if kitInternalID(it.id) {
				continue
			}
			ok := false
			for _, p := range c.prefixes {
				ok = ok || strings.HasPrefix(it.id, p)
			}
			if !ok {
				t.Fatalf("%s: id %q lacks the screen prefix %v", c.name, it.id, c.prefixes)
			}
			if it.focusable && !it.disabled {
				focusable++
			}
		}
		if focusable == 0 {
			t.Fatalf("%s: no keyboard-focusable control", c.name)
		}
		k.run(uiInput{X: -10, Y: -10, Tab: true})
		if k.st.focus == "" {
			t.Fatalf("%s: TAB does not focus a control", c.name)
		}
		if st := g.activeKitState(); st != k.st && c.name != "guest menu" {
			t.Fatalf("%s: F1 overlay would outline another screen's registry", c.name)
		}
		g.menuDebug = true
		img := newTestImage()
		if g.screen == screenGameplay {
			g.drawGameplayInteractions(img)
		} else {
			g.drawMenu(img)
		}
		_ = g.CloseNetplay()
	}
}

// ESC always does something on these screens: it leaves the screen, closes
// the view or resumes (never a dead key).
func TestKitScreensAnswerEsc(t *testing.T) {
	for _, c := range allKitScreenCases() {
		g, k := c.setup(t)
		k.t = t
		k.idle()
		type snap struct {
			screen                      gameScreen
			fade, paused, opts, guest   bool
			loadout, campLoadout, phase int
			view                        int
			quit, leave                 bool
			page                        int
		}
		take := func() snap {
			return snap{g.screen, g.fadeActive, g.paused, g.pauseOptions, g.guestMenuOpen, g.customLoadoutSlot, g.campaignLoadoutSlot, g.campaignPhase, g.gunLibraryView, g.mainMenuQuitOpen, g.onlineLeaveConfirm, g.customPage}
		}
		before := take()
		roundBefore := g.customSeriesRound
		k.key(uiInput{Esc: true})
		if take() == before && g.customSeriesRound == roundBefore && g.netplay == nil {
			t.Fatalf("%s: ESC did nothing", c.name)
		}
		_ = g.CloseNetplay()
	}
}
