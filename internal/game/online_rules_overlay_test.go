package game

import "testing"

// B2: the online rules overlay used to be live for input but undrawn, and its
// RULES button unreachable. On the Mayhem UI it is the host's CONFIGURE
// button in the MATCH tab's ONLINE RULES row; guests see the row read-only.
func TestOnlineWizardRulesOverlayIsReachableAndDrawn(t *testing.T) {
	g := New()
	g.screen = screenCustomGame
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.netplay = &netplaySession{mode: netplayHost}
	g.setOnlineLobbyTab(onlineLobbyTabMatch)
	g.runOnlineLobbyUpdate(uiInput{X: -10, Y: -10})
	k := onlineLobbyKit(g)
	k.click(t, "on.rules")
	if !g.onlineRulesOpen || !(g.onlineRulesOpen || g.onlineBlocklistOpen || g.onlinePerkBlocklistOpen || g.onlineVisualPicker != onlineVisualNone) {
		t.Fatal("RULES did not open the rules overlay")
	}
	if !k.has("on.rules.done") {
		t.Fatal("the rules modal is not on screen")
	}

	c := New()
	c.screen = screenCustomGame
	c.customOnlineMode = true
	c.netplay = &netplaySession{mode: netplayClient}
	c.netplay.assignedPlayerID.Store(2)
	c.setOnlineLobbyTab(onlineLobbyTabMatch)
	c.runOnlineLobbyUpdate(uiInput{X: -10, Y: -10})
	onlineLobbyKit(c).click(t, "on.rules")
	if c.onlineRulesOpen {
		t.Fatal("a guest opened the host RULES")
	}
	if it, _ := c.customUI.item("on.rules"); !it.disabled {
		t.Fatal("the guest RULES button is interactive")
	}
}
