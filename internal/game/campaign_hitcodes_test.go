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
		add(campaignPlayerHitNameBase+slot, fmt.Sprintf("name slot%d", slot), func(g *Game) error {
			if g.campaignNameFocus != slot+1 {
				return fmt.Errorf("name focus=%d", g.campaignNameFocus)
			}
			return nil
		})
		for _, editor := range editors {
			editor := editor
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
	// The Loadout lists both hat pages on one grid and writes every hat
	// through the page-1 hat code (activateCampaignPlayerHit takes both).
	for slot := 0; slot < 2; slot++ {
		slot := slot
		for _, item := range developedCustomSelectionItems(customEditHat2) {
			n := item.Number
			add(campaignSelectHit(slot, customEditHat1, n), fmt.Sprintf("hat page 2 item %d slot%d", n, slot), func(g *Game) error {
				if g.campaignPlayers[slot].Hat != n {
					return fmt.Errorf("hat=%d", g.campaignPlayers[slot].Hat)
				}
				return nil
			})
		}
	}
	add(campaignPlayerHitClearBase+1, "clear slot1", nil)
	return cases
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

// campaignScreenHitCodes: the campaign screens are Mayhem UI kit screens
// (campaign_ui.go) with string ids, so only the player codes remain.
func campaignScreenHitCodes() map[int]string { return map[int]string{} }

// Every tile of the campaign Loadout writes an enumerated player code of
// its own slot.
func TestCampaignLoadoutTilesUseEnumeratedCodes(t *testing.T) {
	known := map[int]bool{}
	for _, c := range campaignPlayerHitCases() {
		known[c.code] = true
	}
	g := freshCampaignSetupGame()
	for slot := 0; slot < 2; slot++ {
		b := g.campaignLoadoutBinding(slot)
		var all []loadoutChoice
		for part := range loadoutPartLabels() {
			c, _ := g.loadoutCharacterChoicesFor(b, part)
			all = append(all, c...)
		}
		all = append(all, g.loadoutWeaponChoicesFor(b)...)
		all = append(all, g.loadoutPerkChoicesFor(b)...)
		all = append(all, g.loadoutGearChoicesFor(b)...)
		all = append(all, g.loadoutThrowableChoicesFor(b)...)
		for _, c := range all {
			if c.locked {
				continue
			}
			if !known[c.code] {
				t.Fatalf("slot %d: %s code %d is not an enumerated campaign code", slot, c.id, c.code)
			}
		}
		if b.nameHit != campaignPlayerHitNameBase+slot {
			t.Fatalf("slot %d name code %d", slot, b.nameHit)
		}
	}
}
