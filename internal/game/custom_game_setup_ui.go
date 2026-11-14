package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func developedModeTitle(mode int) string {
	switch mode {
	case SourceGameModeTeams:
		return "TEAM BATTLE"
	case SourceGameModeSurvival:
		return "DUCK SURVIVAL"
	case SourceGameModeGunGame:
		return "GUN GAME"
	case SourceGameModeInstagib:
		return "1 HIT 1 KILL"
	default:
		return "LAST MAN STANDING"
	}
}

func developedModeDescription(mode int) []string {
	switch mode {
	case SourceGameModeTeams:
		return []string{"Fight in teams. The last surviving team wins the round.", "Team assignments are chosen on the Player Setup screen."}
	case SourceGameModeSurvival:
		return []string{"Survive as long as possible against the ducks.", "Lives and pickups follow the mode rules automatically."}
	case SourceGameModeGunGame:
		return []string{"Every kill advances your weapon level.", "Win by completing the full weapon progression first."}
	case SourceGameModeInstagib:
		return []string{"High-powered weapons and lethal hits.", "Ammo matters: every missed shot can cost the round."}
	default:
		return []string{"Free for all. The last player alive wins the round.", "Crates and pickups can be enabled independently."}
	}
}

func drawSetupPanel(dst *ebiten.Image, r Rect) {
	ebitenutil.DrawRect(dst, r.X, r.Y, r.W, r.H, color.NRGBA{R: 0x14, G: 0x17, B: 0x1c, A: 0xff})
	ebitenutil.DrawRect(dst, r.X, r.Y, r.W, 3, color.NRGBA{R: 0xff, G: 0x97, B: 0x00, A: 0xff})
}

func drawSetupSmallButton(dst *ebiten.Image, r Rect, label string, enabled bool) {
	fill := color.NRGBA{R: 0x3d, G: 0x42, B: 0x4b, A: 0xff}
	text := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	if !enabled {
		fill = color.NRGBA{R: 0x2a, G: 0x2d, B: 0x33, A: 0xff}
		text = color.NRGBA{R: 0x78, G: 0x7d, B: 0x86, A: 0xff}
	}
	ebitenutil.DrawRect(dst, r.X, r.Y, r.W, r.H, fill)
	drawSourceMenuText(dst, label, menuFontTwCen, 23, text, r.X+r.W/2-7, r.Y+6)
}

func drawSetupToggleCard(dst *ebiten.Image, r Rect, title, state string, active, enabled bool) {
	fill := color.NRGBA{R: 0x25, G: 0x29, B: 0x30, A: 0xff}
	if enabled && active {
		fill = color.NRGBA{R: 0x2d, G: 0x31, B: 0x38, A: 0xff}
	}
	ebitenutil.DrawRect(dst, r.X, r.Y, r.W, r.H, fill)
	drawSourceMenuText(dst, title, menuFontCondensedExtraBold, 16, color.White, r.X+14, r.Y+11)
	stateColor := color.NRGBA{R: 0xa4, G: 0xaa, B: 0xb3, A: 0xff}
	if active && enabled {
		stateColor = color.NRGBA{R: 0xff, G: 0x97, B: 0x00, A: 0xff}
	}
	drawSourceMenuText(dst, state, menuFontTwCen, 22, stateColor, r.X+14, r.Y+36)
}

func (g *Game) developedModifierSummary() string {
	if g.customOnlineMode && !g.customModifierEditingAllowed() {
		return "ONLINE RULES ARE CONTROLLED BY THE HOST"
	}
	if g.customChaosEnabled {
		return fmt.Sprintf("CHAOS MODE - %d EVENTS - EVERY %ds", g.chaosSelectedCount(), g.customChaosInterval)
	}
	if g.customPerRoundRules {
		return fmt.Sprintf("PER ROUND / MAP RULES - %d ROUNDS", clampCustomSeriesRounds(g.customSeriesRounds))
	}
	if g.customRandomRules {
		return fmt.Sprintf("RANDOM RULES - %d PER ROUND", g.customRandomCountForEdit())
	}
	count := 0
	for id := 1; id <= customModifierJetpacks; id++ {
		if g.customModifierEnabledForEdit(id) {
			count++
		}
	}
	if count == 0 {
		return "NO MODIFIERS"
	}
	if count == 1 {
		return "1 MODIFIER ACTIVE"
	}
	return fmt.Sprintf("%d MODIFIERS ACTIVE", count)
}
