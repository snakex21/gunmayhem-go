package game

import (
	"fmt"
	"testing"
)

type campaignHitCase struct {
	code  int
	label string
	check func(g *Game) error
}

func freshCampaignSetupGame() *Game {
	g := &Game{campaignSetExtra: true}
	g.initCampaignPlayerSetup()
	g.campaignPlayers[1].Type = 1
	return g
}

// campaignPlayerHitCases enumerates every hit code the campaign player cards
// can produce, with the effect activating it must have.
func campaignPlayerHitCases() []campaignHitCase {
	var cases []campaignHitCase
	add := func(code int, label string, check func(g *Game) error) {
		cases = append(cases, campaignHitCase{code, label, check})
	}
	editors := []int{customEditShirt, customEditHat1, customEditGun, customEditPerk, customEditEyes, customEditThrowable, customEditColor, customEditAbility}
	for slot := 0; slot < 2; slot++ {
		slot := slot
		add(campaignPlayerHitHumanBase+slot, fmt.Sprintf("human slot%d", slot), func(g *Game) error {
			if g.campaignPlayers[slot].Type != 1 {
				return fmt.Errorf("type=%d", g.campaignPlayers[slot].Type)
			}
			return nil
		})
		add(campaignPlayerHitBackBase+slot, fmt.Sprintf("editor back slot%d", slot), nil)
		add(campaignPlayerHitHatPageBase+slot, fmt.Sprintf("hat page slot%d", slot), func(g *Game) error {
			if g.campaignEditor[slot] != customEditHat1 && g.campaignEditor[slot] != customEditHat2 {
				return fmt.Errorf("editor=%d", g.campaignEditor[slot])
			}
			return nil
		})
		add(campaignPlayerHitNameBase+slot, fmt.Sprintf("name slot%d", slot), func(g *Game) error {
			if g.campaignNameFocus != slot+1 {
				return fmt.Errorf("name focus=%d", g.campaignNameFocus)
			}
			return nil
		})
		for _, editor := range editors {
			editor := editor
			add(campaignPlayerHitEditBase+slot*10+editor, fmt.Sprintf("open editor %d slot%d", editor, slot), func(g *Game) error {
				if g.campaignEditor[slot] == 0 {
					return fmt.Errorf("editor not opened")
				}
				return nil
			})
			for _, item := range developedCustomSelectionItems(editor) {
				n := item.Number
				switch {
				case editor == customEditGun && n >= 98:
					add(campaignDevelopedGunHit(slot, n), fmt.Sprintf("developed gun %d slot%d", n, slot), func(g *Game) error {
						if g.campaignPlayers[slot].Gun != n {
							return fmt.Errorf("gun=%d", g.campaignPlayers[slot].Gun)
						}
						return nil
					})
				case editor == customEditThrowable:
					add(campaignThrowableChoiceHit(slot, n-1), fmt.Sprintf("throwable %d slot%d", n-1, slot), func(g *Game) error {
						if g.campaignPlayers[slot].Throwable != n-1 {
							return fmt.Errorf("throwable=%d", g.campaignPlayers[slot].Throwable)
						}
						return nil
					})
				default:
					add(campaignSelectHit(slot, editor, n), fmt.Sprintf("select editor %d item %d slot%d", editor, n, slot), func(g *Game) error {
						cfg := g.campaignPlayers[slot]
						got := -1
						switch editor {
						case customEditShirt:
							got = cfg.Shirt
						case customEditHat1, customEditHat2:
							got = cfg.Hat
						case customEditGun:
							got = cfg.Gun
						case customEditPerk:
							got = cfg.Perk
						case customEditEyes:
							got = cfg.Eyes
						case customEditColor:
							got = cfg.Color
						case customEditAbility:
							if a, ok := abilityFromMenuNumber(n); ok && cfg.Ability == a {
								got = n
							}
						}
						if got != n {
							return fmt.Errorf("value=%d", got)
						}
						if g.campaignNameFocus != 0 {
							return fmt.Errorf("selection also focused a name field")
						}
						other := g.campaignPlayers[1-slot]
						fresh := freshCampaignSetupGame().campaignPlayers[1-slot]
						if other != fresh {
							return fmt.Errorf("selection changed the other slot")
						}
						return nil
					})
				}
			}
		}
	}
	add(campaignPlayerHitClearBase+1, "clear slot1", nil)
	return cases
}

func campaignScreenHitCodes() map[int]string {
	codes := map[int]string{
		campaignHitBack: "back", campaignHitStart: "start", campaignHitUnlockContinue: "unlock continue",
		campaignHitSetGM1: "tab GM1", campaignHitSetGM2: "tab GM2", campaignHitSetExtra: "tab EXTRA",
		gm2CampaignHitStart: "GM2 start", extraCampaignHitStart: "EXTRA start",
		extraCampaignHitPrevPage: "EXTRA prev", extraCampaignHitNextPage: "EXTRA next",
	}
	for i := 0; i < 10; i++ {
		codes[campaignHitPanelBase+i] = fmt.Sprintf("GM1 panel %d", i)
	}
	for i := 1; i <= 16; i++ {
		codes[gm2CampaignHitMissionBase+i] = fmt.Sprintf("GM2 mission %d", i)
	}
	for i := 1; i <= extraCampaignMissionCount; i++ {
		codes[extraCampaignHitMissionBase+i] = fmt.Sprintf("EXTRA mission %d", i)
	}
	return codes
}

// Every hit code on the campaign screens must mean exactly one thing.
// Regression: slot 0 colour choices collided with the name fields and eye
// choice 1 with slot 1's hat-page button.
func TestCampaignPlayerSetupHitCodesAreUnique(t *testing.T) {
	seen := campaignScreenHitCodes()
	screenCodes := len(seen)
	for _, c := range campaignPlayerHitCases() {
		if c.code == campaignHitNone {
			t.Fatalf("%s produced campaignHitNone", c.label)
		}
		if prev, dup := seen[c.code]; dup {
			t.Fatalf("hit code %d is both %q and %q", c.code, prev, c.label)
		}
		seen[c.code] = c.label
	}
	if len(seen) < screenCodes+200 {
		t.Fatalf("suspiciously few player hit codes: %d", len(seen)-screenCodes)
	}
}

func TestCampaignPlayerSetupHitCodesActivateTheirOwnAction(t *testing.T) {
	for _, c := range campaignPlayerHitCases() {
		if c.check == nil {
			continue
		}
		g := freshCampaignSetupGame()
		if c.code >= campaignPlayerHitHumanBase && c.code < campaignPlayerHitHumanBase+2 {
			g.campaignPlayers[c.code-campaignPlayerHitHumanBase].Type = 0
		}
		if !g.activateCampaignPlayerHit(c.code) {
			t.Fatalf("%s (code %d) was not handled", c.label, c.code)
		}
		if err := c.check(g); err != nil {
			t.Fatalf("%s (code %d): %v", c.label, c.code, err)
		}
	}
}

// Clicking the centre of each grid item of the editors that used to collide
// (colour, eyes, hats) returns that item's enumerated selection code. (A full
// pixel scan is too slow: one campaignPlayerHitAt call costs ~10 ms.)
func TestCampaignPlayerHitAtReturnsEnumeratedSelectionCodes(t *testing.T) {
	known := map[int]bool{}
	for _, c := range campaignPlayerHitCases() {
		known[c.code] = true
	}
	for _, editor := range []int{customEditColor, customEditEyes, customEditHat1} {
		g := freshCampaignSetupGame()
		g.campaignEditor[0] = editor
		g.campaignCardY[0] = 100
		checked := 0
		for _, item := range developedCustomSelectionItems(editor) {
			if !g.campaignSelectionUnlocked(editor, item.Number) {
				continue
			}
			x := campaignCardX(0) + item.Rect.X + item.Rect.W/2
			y := g.campaignCardY[0] + item.Rect.Y + item.Rect.H/2
			hit := g.campaignPlayerHitAt(x, y)
			if hit != campaignSelectHit(0, editor, item.Number) || !known[hit] {
				t.Fatalf("editor %d item %d: hit=%d want %d", editor, item.Number, hit, campaignSelectHit(0, editor, item.Number))
			}
			checked++
			if checked >= 12 {
				break
			}
		}
		if checked == 0 {
			t.Fatalf("editor %d: no items checked", editor)
		}
	}
}
