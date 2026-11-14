package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// B2: the online rules overlay was live for input but never drawn in the v2
// online wizard, and its RULES button was unreachable there.
func TestOnlineWizardRulesOverlayIsReachableAndDrawn(t *testing.T) {
	g := New()
	g.screen = screenCustomGame
	g.customOnlineMode = true
	g.onlineMatchSetup = true
	g.netplay = &netplaySession{mode: netplayHost}
	for _, page := range []int{customPageGame, customPageMaps, customPagePlayers} {
		g.setCustomV2Page(page)
		r := customSessionRulesRect
		if hit := g.customV2HitAt(r.X+r.W/2, r.Y+r.H/2); hit != menuHitCustomSessionRules {
			t.Fatalf("page %d: RULES hit=%d", page, hit)
		}
	}
	g.setCustomV2Page(customPageGame)

	g.activateCustomV2Hit(menuHitCustomSessionRules)
	if !g.onlineRulesOpen || !g.customOnlineRulesOverlayActive() {
		t.Fatal("RULES did not open the rules overlay")
	}
	// Drawn on top of every wizard page (visual check: menucheck/phase01).
	screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
	for _, page := range []int{customPageGame, customPageMaps, customPagePlayers} {
		g.setCustomV2Page(page)
		g.drawCustomGameV2(screen)
	}

	// Clients have no RULES button.
	c := New()
	c.customOnlineMode = true
	c.netplay = &netplaySession{mode: netplayClient}
	r := customSessionRulesRect
	if hit := c.customV2HitAt(r.X+r.W/2, r.Y+r.H/2); hit == menuHitCustomSessionRules {
		t.Fatal("client got the host RULES button")
	}
}
