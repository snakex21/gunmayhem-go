package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var (
	developedSetupBackRect     = Rect{X: 20, Y: 522, W: 180, H: 56}
	developedSetupContinueRect = Rect{X: 555, Y: 522, W: 325, H: 56}

	developedSetupLivesMinusRect  = Rect{X: 365, Y: 299, W: 42, H: 36}
	developedSetupLivesValueRect  = Rect{X: 414, Y: 299, W: 96, H: 36}
	developedSetupLivesPlusRect   = Rect{X: 517, Y: 299, W: 42, H: 36}
	developedSetupRoundsMinusRect = Rect{X: 615, Y: 299, W: 42, H: 36}
	developedSetupRoundsPlusRect  = Rect{X: 785, Y: 299, W: 42, H: 36}
	developedSetupCratesRect      = Rect{X: 345, Y: 354, W: 215, H: 72}
	developedSetupPickupsRect     = Rect{X: 595, Y: 354, W: 250, H: 72}
	developedSetupModifiersRect   = Rect{X: 345, Y: 441, W: 500, H: 48}
)

var developedSetupModeRows = [...]struct {
	mode  int
	rect  Rect
	label string
}{
	{SourceGameModeNormal, Rect{X: 35, Y: 154, W: 255, H: 54}, "LAST MAN STANDING"},
	{SourceGameModeTeams, Rect{X: 35, Y: 216, W: 255, H: 54}, "TEAM BATTLE"},
	{SourceGameModeSurvival, Rect{X: 35, Y: 278, W: 255, H: 54}, "DUCK SURVIVAL"},
	{SourceGameModeGunGame, Rect{X: 35, Y: 340, W: 255, H: 54}, "GUN GAME"},
	{SourceGameModeInstagib, Rect{X: 35, Y: 402, W: 255, H: 54}, "1 HIT 1 KILL"},
}

func developedCustomGameHitAt(x, y float64) int {
	for _, row := range developedSetupModeRows {
		if row.rect.Contains(x, y) {
			return menuHitModeBase + row.mode
		}
	}
	if developedSetupBackRect.Contains(x, y) {
		return menuHitGameBack
	}
	if developedSetupContinueRect.Contains(x, y) {
		return menuHitGameContinue
	}
	return menuHitNone
}

func (g *Game) developedCustomGameOptionHitAt(x, y float64) int {
	if g == nil {
		return menuHitNone
	}
	if modifierHit := g.customModifierHitAt(x-1800, y); modifierHit != menuHitNone {
		return modifierHit
	}
	if modeHasLives(g.customMode) {
		switch {
		case developedSetupLivesMinusRect.Contains(x, y):
			return menuHitModeLivesMinus
		case developedSetupLivesValueRect.Contains(x, y):
			return menuHitModeLives
		case developedSetupLivesPlusRect.Contains(x, y):
			return menuHitModeLivesPlus
		}
	}
	if developedSetupRoundsMinusRect.Contains(x, y) {
		return menuHitModeRoundsMinus
	}
	if developedSetupRoundsPlusRect.Contains(x, y) {
		return menuHitModeRoundsPlus
	}
	if modeHasCrateToggle(g.customMode) && developedSetupCratesRect.Contains(x, y) {
		return menuHitModeCrates
	}
	if modeHasPickupToggle(g.customMode) && developedSetupPickupsRect.Contains(x, y) {
		return menuHitModePickups
	}
	if g.customModifierEditingAllowed() && developedSetupModifiersRect.Contains(x, y) {
		return menuHitCustomModifiersOpen
	}
	return menuHitNone
}

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

func (g *Game) drawDevelopedCustomGameSetup(screen *ebiten.Image) {
	// This page deliberately no longer depends on the flattened Flash Game Setup
	// raster. The source behavior remains in the underlying game rules, while the
	// developed branch gets a layout that can grow without overlapping controls.
	screen.Fill(color.NRGBA{R: 0x20, G: 0x24, B: 0x2a, A: 0xff})

	header := Rect{X: 20, Y: 20, W: 860, H: 72}
	ebitenutil.DrawRect(screen, header.X, header.Y, header.W, header.H, color.NRGBA{R: 0xf2, G: 0xf2, B: 0xf2, A: 0xff})
	drawSourceMenuText(screen, "GAME SETUP", menuFontCondensedExtraBold, 42, color.Black, 38, 28)
	drawSourceMenuText(screen, "CUSTOM MATCH  /  STEP 1 OF 3", menuFontCondensed, 12.5, color.NRGBA{R: 0x55, G: 0x58, B: 0x60, A: 0xff}, 380, 58)

	drawSetupPanel(screen, Rect{X: 20, Y: 108, W: 285, H: 397})
	drawSetupPanel(screen, Rect{X: 320, Y: 108, W: 560, H: 397})

	drawSourceMenuText(screen, "GAME MODE", menuFontCondensedExtraBold, 25, color.White, 35, 119)
	for _, row := range developedSetupModeRows {
		selected := row.mode == g.customMode
		fill := color.NRGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff}
		text := color.Black
		if selected {
			fill = color.NRGBA{R: 0xff, G: 0x97, B: 0x00, A: 0xff}
		}
		ebitenutil.DrawRect(screen, row.rect.X, row.rect.Y, row.rect.W, row.rect.H, fill)
		drawSourceMenuText(screen, row.label, menuFontTwCen, 19, text, row.rect.X+13, row.rect.Y+15)
	}

	drawSourceMenuText(screen, developedModeTitle(g.customMode), menuFontCondensedExtraBold, 31, color.White, 345, 120)
	for i, line := range developedModeDescription(g.customMode) {
		drawSourceMenuText(screen, line, menuFontCondensed, 14.5, color.NRGBA{R: 0xc9, G: 0xcd, B: 0xd3, A: 0xff}, 345, 160+float64(i)*23)
	}
	ebitenutil.DrawRect(screen, 345, 211, 500, 2, color.NRGBA{R: 0x49, G: 0x4e, B: 0x57, A: 0xff})
	drawSourceMenuText(screen, "MATCH SETTINGS", menuFontCondensedExtraBold, 20, color.White, 345, 223)

	// Lives card.
	ebitenutil.DrawRect(screen, 345, 255, 215, 88, color.NRGBA{R: 0x25, G: 0x29, B: 0x30, A: 0xff})
	drawSourceMenuText(screen, "LIVES", menuFontCondensedExtraBold, 15.5, color.White, 359, 265)
	if modeHasLives(g.customMode) {
		drawSetupSmallButton(screen, developedSetupLivesMinusRect, "-", true)
		drawSetupSmallButton(screen, developedSetupLivesPlusRect, "+", true)
		value := fmt.Sprintf("%d", g.customLives)
		if g.customLivesFocus {
			value += "_"
		}
		drawSourceMenuText(screen, value, menuFontTwCen, 24, color.White, 429, 304)
	} else if g.customMode == SourceGameModeSurvival {
		drawSourceMenuText(screen, "10  FIXED", menuFontTwCen, 23, color.NRGBA{R: 0xb8, G: 0xbd, B: 0xc4, A: 0xff}, 380, 304)
	} else {
		drawSourceMenuText(screen, "MODE RULE", menuFontTwCen, 21, color.NRGBA{R: 0xb8, G: 0xbd, B: 0xc4, A: 0xff}, 380, 304)
	}

	// Series card.
	ebitenutil.DrawRect(screen, 595, 255, 250, 88, color.NRGBA{R: 0x25, G: 0x29, B: 0x30, A: 0xff})
	drawSourceMenuText(screen, "ROUNDS / MAPS", menuFontCondensedExtraBold, 15.5, color.White, 609, 265)
	drawSetupSmallButton(screen, developedSetupRoundsMinusRect, "-", g.customSeriesRounds > 1)
	drawSetupSmallButton(screen, developedSetupRoundsPlusRect, "+", g.customSeriesRounds < customSeriesMaxRounds)
	drawSourceMenuText(screen, fmt.Sprintf("%d", g.customSeriesRounds), menuFontTwCen, 25, color.White, 709, 304)

	crateState, crateActive, crateEnabled := "MODE RULE", false, false
	if modeHasCrateToggle(g.customMode) {
		crateEnabled = true
		crateActive = g.CrateON
		if g.CrateON {
			crateState = "ON"
		} else {
			crateState = "OFF"
		}
	} else if g.customMode == SourceGameModeSurvival {
		crateState, crateActive = "ON - FIXED", true
	}
	drawSetupToggleCard(screen, developedSetupCratesRect, "CRATES", crateState, crateActive, crateEnabled)

	pickupState, pickupActive, pickupEnabled := "MODE RULE", false, false
	if modeHasPickupToggle(g.customMode) {
		pickupEnabled = true
		pickupActive = g.PowerON
		if g.PowerON {
			pickupState = "ON"
		} else {
			pickupState = "OFF"
		}
	} else if g.customMode == SourceGameModeSurvival {
		pickupState = "OFF - FIXED"
	}
	drawSetupToggleCard(screen, developedSetupPickupsRect, "PICKUPS", pickupState, pickupActive, pickupEnabled)

	modifierFill := color.NRGBA{R: 0x31, G: 0x35, B: 0x3d, A: 0xff}
	if g.customModifierEditingAllowed() {
		modifierFill = color.NRGBA{R: 0x36, G: 0x3a, B: 0x42, A: 0xff}
	}
	ebitenutil.DrawRect(screen, developedSetupModifiersRect.X, developedSetupModifiersRect.Y, developedSetupModifiersRect.W, developedSetupModifiersRect.H, modifierFill)
	drawSourceMenuText(screen, "MODIFIERS", menuFontCondensedExtraBold, 15, color.White, 360, 452)
	drawSourceMenuText(screen, g.developedModifierSummary(), menuFontCondensed, 13.5, color.NRGBA{R: 0xc7, G: 0xcb, B: 0xd1, A: 0xff}, 480, 453)

	// Bottom navigation is intentionally separated from settings; this prevents
	// future options from colliding with BACK/CONTINUE as the old source panel did.
	ebitenutil.DrawRect(screen, developedSetupBackRect.X, developedSetupBackRect.Y, developedSetupBackRect.W, developedSetupBackRect.H, color.NRGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff})
	drawSourceMenuText(screen, "BACK", menuFontTwCen, 31, color.Black, 38, 535)
	drawSourceMenuText(screen, "TO MENU", menuFontCondensedExtraBold, 11.5, color.NRGBA{R: 0x55, G: 0x58, B: 0x60, A: 0xff}, 121, 553)

	ebitenutil.DrawRect(screen, developedSetupContinueRect.X, developedSetupContinueRect.Y, developedSetupContinueRect.W, developedSetupContinueRect.H, color.NRGBA{R: 0xff, G: 0x97, B: 0x00, A: 0xff})
	drawSourceMenuText(screen, "CONTINUE", menuFontTwCen, 31, color.Black, 672, 535)
	drawSourceMenuText(screen, "MAP SELECTION", menuFontCondensedExtraBold, 11.5, color.NRGBA{R: 0x58, G: 0x34, B: 0x00, A: 0xff}, 580, 553)

	if g.customWarningFrame > 0 && g.customWarning == 1 {
		ebitenutil.DrawRect(screen, 345, 492, 500, 24, color.NRGBA{R: 0x8e, G: 0x32, B: 0x21, A: 0xff})
		drawSourceMenuText(screen, "LIVES MUST BE AT LEAST 1", menuFontCondensedExtraBold, 12.5, color.White, 360, 497)
	}

	g.drawCustomSessionControls(screen)
	g.drawCustomModifiersOverlay(screen)
}
