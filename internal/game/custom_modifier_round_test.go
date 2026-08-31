package game

import "testing"

func TestPerRoundModifierProfilesAreIndependent(t *testing.T) {
	g := New()
	g.customSeriesRounds = 3
	g.customRuleEnabled[customModifierShotgunsOnly] = true
	g.customRandomCount = 1
	g.enableCustomPerRoundRules()

	if !g.customPerRoundRules || !g.customModifierEnabledForEdit(customModifierShotgunsOnly) {
		t.Fatal("per-round rules did not inherit the series profile")
	}

	g.customRuleEditRound = 2
	g.setCustomModifierEnabledForEdit(customModifierShotgunsOnly, false)
	g.setCustomModifierEnabledForEdit(customModifierLowGravity, true)

	g.customRuleEditRound = 1
	if !g.customModifierEnabledForEdit(customModifierShotgunsOnly) || g.customModifierEnabledForEdit(customModifierLowGravity) {
		t.Fatal("editing round 2 changed round 1")
	}
	g.customRuleEditRound = 2
	if g.customModifierEnabledForEdit(customModifierShotgunsOnly) || !g.customModifierEnabledForEdit(customModifierLowGravity) {
		t.Fatal("round 2 profile was not preserved")
	}
}

func TestPrepareCustomMatchModifiersUsesCurrentSeriesRound(t *testing.T) {
	g := New()
	g.customSeriesRounds = 2
	g.enableCustomPerRoundRules()
	g.customRoundRuleEnabled[0] = [customModifierJetpacks + 1]bool{}
	g.customRoundRuleEnabled[1] = [customModifierJetpacks + 1]bool{}
	g.customRoundRuleEnabled[0][customModifierShotgunsOnly] = true
	g.customRoundRuleEnabled[1][customModifierLowGravity] = true

	g.customSeriesRound = 1
	g.prepareCustomMatchModifiers()
	if !g.customModifierActive(customModifierShotgunsOnly) || g.customModifierActive(customModifierLowGravity) {
		t.Fatal("round 1 used the wrong modifier profile")
	}

	g.customSeriesRound = 2
	g.prepareCustomMatchModifiers()
	if g.customModifierActive(customModifierShotgunsOnly) || !g.customModifierActive(customModifierLowGravity) {
		t.Fatal("round 2 used the wrong modifier profile")
	}
}

func TestPerRoundProfilesSurviveScopeToggle(t *testing.T) {
	g := New()
	g.customSeriesRounds = 3
	g.enableCustomPerRoundRules()
	g.customRuleEditRound = 3
	g.setCustomModifierEnabledForEdit(customModifierJetpacks, true)

	g.customPerRoundRules = false
	g.enableCustomPerRoundRules()
	g.customRuleEditRound = 3
	if !g.customModifierEnabledForEdit(customModifierJetpacks) {
		t.Fatal("switching rule scope erased the saved per-round profile")
	}
}
