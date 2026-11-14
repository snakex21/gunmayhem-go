package game

import (
	"fmt"
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
