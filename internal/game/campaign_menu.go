package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	campaignHitNone = iota
	campaignHitBack
	campaignHitStart
	campaignHitUnlockContinue
	campaignHitPanelBase = 100
)

var (
	campaignPanelLocal    = Rect{X: 0, Y: -0.013818359375, W: 310.0006103515625, H: 184.991455078125} // Symbol95
	campaignBackLocal     = Rect{X: 0, Y: 0, W: 135.05, H: 53}                                        // Symbol792
	campaignStartLocal    = Rect{X: 0, Y: 0, W: 398.1, H: 53}                                         // Symbol902
	campaignContinueLocal = Rect{X: 0, Y: 0, W: 148.5, H: 24}                                         // Symbol806
)

func (g *Game) enterCampaignMenu() {
	g.campaignMode = false
	g.campaignPhase = 1
	g.campaignDetailAlpha = 0
	g.campaignPressed = campaignHitNone
	// Symbol948 startup unlock propagation. Level2 is available from a fresh
	// SharedObject, so propagation starts at completed level2 -> level3.
	for i := 1; i < 9; i++ {
		if g.campaignLevels[i] == 2 && g.campaignLevels[i+1] == 0 {
			g.campaignLevels[i+1] = 1
		}
	}
	if g.campaignSliderX > 0 {
		g.campaignSliderX = 0
	}
	if g.campaignSliderX < -857 {
		g.campaignSliderX = -857
	}
}

func campaignPanelPlacements() [10]Rect {
	return [10]Rect{
		{X: 28, Y: 100, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 375, Y: 100, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 722, Y: 100, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 1069, Y: 100, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 1416, Y: 100, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 28, Y: 316.05, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 375, Y: 316.05, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 722, Y: 316.05, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 1069, Y: 316.05, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
		{X: 1416, Y: 316.05, W: campaignPanelLocal.W, H: campaignPanelLocal.H},
	}
}

func (g *Game) unlockAllCampaignLevelsCheat() {
	for i := range g.campaignLevels {
		// 0 = locked, 1 = available, 2 = completed. The test cheat only makes
		// locked levels selectable; it must not fake campaign completion/rewards.
		if g.campaignLevels[i] == 0 {
			g.campaignLevels[i] = 1
		}
	}
}

func (g *Game) updateCampaignMenu() {
	if inpututil.IsKeyJustPressed(ebiten.KeyF10) {
		g.unlockAllCampaignLevelsCheat()
		g.playSourceSFX("menu.wav", false)
	}

	if g.campaignShowUnlock > 0 {
		g.updateCampaignUnlockPopup()
		return
	}

	if g.campaignPhase != 1 {
		g.updateCampaignPlayerCards()
		g.updateCampaignNameInput()
	}
	mx, my := ebiten.CursorPosition()
	if g.campaignPhase == 1 {
		// Symbol948.onEnterFrame phase1 edge-scroll physics.
		if my > 100 && my < 510 {
			if mx < 150 && g.campaignSliderX < 0 {
				g.campaignSliderVX += (150 - float64(mx)) / 8
			}
			if mx > 750 && g.campaignSliderX > -857 {
				g.campaignSliderVX -= (float64(mx) - 750) / 8
			}
		}
		g.campaignSliderVX *= 0.8
		g.campaignSliderX += g.campaignSliderVX
		if g.campaignSliderX > 0 {
			g.campaignSliderX = 0
			g.campaignSliderVX = 0
		}
		if g.campaignSliderX < -857 {
			g.campaignSliderX = -857
			g.campaignSliderVX = 0
		}
		g.campaignSliderX = math.Round(g.campaignSliderX)

		// Symbol95 uses onPress, unlike most menu buttons.
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			localX := float64(mx) - g.campaignSliderX
			for i, r := range campaignPanelPlacements() {
				if r.Contains(localX, float64(my)) && g.campaignLevels[i] != 0 {
					// Symbol95.onPress plays menu.wav before opening the level detail.
					g.playSourceSFX("menu.wav", false)
					g.campaignLevel = i + 1
					g.campaignPhase = 2
					g.campaignDetailAlpha = 0
					g.campaignSliderVX = 0
					return
				}
			}
		}
	} else if g.campaignPhase == 2 {
		// Source alpha +=20 until the detail/player panels are fully visible.
		g.campaignDetailAlpha += 0.2
		if g.campaignDetailAlpha >= 1 {
			g.campaignDetailAlpha = 1
			g.campaignPhase = 3
		}
	} else if g.campaignPhase == 4 {
		g.campaignDetailAlpha -= 0.2
		if g.campaignDetailAlpha <= 0 {
			g.campaignDetailAlpha = 0
			g.campaignPhase = 1
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.campaignPhase == 1 {
			g.beginFade(screenMainMenu, fadePurposeScreen)
		} else if g.campaignPhase == 3 {
			g.campaignPhase = 4
		}
		return
	}

	// Symbol948 btn_back is onRelease. In phase1 it returns to root frame10;
	// in phase3 it transitions back to the slider.
	back := sourceMenuHitRect(campaignBackLocal, 27, 525.15, 1, 1)
	start := sourceMenuHitRect(campaignStartLocal, 469.1, 522.15, 1, 1)
	hit := campaignHitNone
	if g.campaignPhase == 3 {
		if playerHit := g.campaignPlayerHitAt(float64(mx), float64(my)); playerHit != campaignHitNone {
			hit = playerHit
		}
	}
	if hit == campaignHitNone && back.Contains(float64(mx), float64(my)) {
		hit = campaignHitBack
	} else if hit == campaignHitNone && g.campaignPhase == 3 && start.Contains(float64(mx), float64(my)) {
		hit = campaignHitStart
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.campaignPressed = hit
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		pressed := g.campaignPressed
		g.campaignPressed = campaignHitNone
		if pressed == campaignHitNone || pressed != hit {
			return
		}
		if g.activateCampaignPlayerHit(pressed) {
			return
		}
		switch pressed {
		case campaignHitBack:
			// Symbol948.btn_back plays menu.wav in both slider and detail phases.
			g.playSourceSFX("menu.wav", false)
			if g.campaignPhase == 1 {
				g.beginFade(screenMainMenu, fadePurposeScreen)
			} else if g.campaignPhase == 3 {
				g.campaignPhase = 4
			}
		case campaignHitStart:
			// Symbol904 frame2 btn_start plays menu.wav on release.
			g.playSourceSFX("menu.wav", false)
			g.startCampaignMission()
		}
	}
}

func (g *Game) updateCampaignUnlockPopup() {
	// Symbol947 frame-specific popup consumes interaction and disables all
	// slider panel hand cursors until Continue is released.
	mx, my := ebiten.CursorPosition()
	continueRect := sourceMenuHitRect(campaignContinueLocal, 344.85, 356.1, 1.3468017578125, 1.25)
	hit := campaignHitNone
	if continueRect.Contains(float64(mx), float64(my)) {
		hit = campaignHitUnlockContinue
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.campaignPressed = hit
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		pressed := g.campaignPressed
		g.campaignPressed = campaignHitNone
		if pressed == campaignHitUnlockContinue && hit == campaignHitUnlockContinue {
			// Symbol947 Continue plays menu.wav after clearing the unlock popup.
			g.playSourceSFX("menu.wav", false)
			g.campaignShowUnlock = 0
		}
	}
}

func (g *Game) drawCampaignMenu(screen *ebiten.Image) {
	// Recompose Symbol948 instead of using its flattened FFDec PNG. The flat
	// export bakes show_unlock into the image even though ActionScript removes
	// that movie clip whenever root.showunlocks==0.
	drawSourceRaster(screen, g.assets.CampaignBase, 0, 0, 1, 1, 1)
	drawSourceMenuText(screen, "CAMPAIGN", menuFontCondensed, 42.4, color.Black, 34, 27.75)
	drawSourceRaster(screen, g.assets.CampaignBack, 27, 525.15, 1, 1, 1)
	drawSourceMenuText(screen, "Back", menuFontCondensed, 26.5, color.Black, 74.05, 536.35)

	sliderAlpha := 1.0
	if g.campaignPhase != 1 {
		sliderAlpha = 1 - g.campaignDetailAlpha
	}
	if sliderAlpha > 0 {
		g.drawCampaignSlider(screen, sliderAlpha)
	}

	if g.campaignPhase != 1 && g.campaignDetailAlpha > 0 {
		a := g.campaignDetailAlpha
		g.drawCampaignPlayerCards(screen, a)
		drawSourceRaster(screen, g.assets.CampaignInfoPanel, 467.1, 100, 1, 1, a)
		if g.campaignLevel >= 1 && g.campaignLevel <= 10 {
			g.drawCampaignInfoRuntime(screen, g.campaignLevel, a)
		}
		// Symbol904 frame2: Symbol902 button at local (2,422.15) and the
		// "Start Level" text at (126.05,429). This runtime child disappeared
		// when the broken flattened Symbol904 frame was replaced.
		if g.campaignPhase == 3 {
			drawSourceRaster(screen, g.assets.CampaignStartButton, 469.1, 522.15, 1, 1, a)
			drawSourceMenuText(screen, "Start Level", menuFontCondensedExtraBold, 31.95,
				color.Black, 593.15, 529.0)
		}
	}

	if g.campaignShowUnlock >= 1 && g.campaignShowUnlock <= 10 {
		drawSourceRaster(screen, g.assets.CampaignUnlock[g.campaignShowUnlock], 0, 0, 1, 1, 1)
	}
}

// startCampaignMission mirrors the root frame10 campaign switch. Specialized
// AI classes (AI3/AI4/boss) still share the common playerAI movement in the Go
// reconstruction, but their source names/loadout/perks/team/map/mode are set
// here so the campaign flow and interactions are no longer placeholders.
func (g *Game) drawCampaignSlider(screen *ebiten.Image, alpha float64) {
	for i, p := range campaignPanelPlacements() {
		x := g.campaignSliderX + p.X
		y := p.Y
		// Symbol95 source hierarchy. FFDec flattened Symbol90/Symbol94 to
		// their first frame, so compose the actual level/card state directly.
		drawSourceRaster(screen, g.assets.CampaignLevelBase, x, y+61.65, 1.5500030517578125, 0.1541595458984375, alpha)
		g.drawCampaignLevelRuntime(screen, i+1, x, y, alpha)
	}
}

func (g *Game) startCampaignMission() {
	level := g.campaignLevel
	if level < 1 || level > 10 || g.fadeActive {
		return
	}
	type campaignSpec struct {
		mapNumber int
		lives     int
		mode      int
		crate     bool
		power     bool
		enemies   int
		name      string
		color     int
		shirt     int
		hat       int
		gun       int
		perk      int
	}
	specs := [10]campaignSpec{
		{13, 3, SourceGameModeNormal, false, false, 1, "Dummy", 8, 24, 1, 0, 0},
		{12, 7, SourceGameModeNormal, true, true, 1, "Caveman Johnson", 8, 10, 7, 1, 0},
		{10, 7, SourceGameModeGunGame, true, true, 1, "Santa", 1, 11, 3, 2, 7},
		{8, 7, SourceGameModeNormal, true, false, 1, "Pylon Man", 9, 1, 9, 1, 0},
		{5, 7, SourceGameModeNormal, true, true, 1, "Gun Fu Master", 7, 9, 10, 3, 6},
		{11, 5, SourceGameModeTeams, true, true, 2, "Mafia Guy", 2, 3, 5, 2, 0},
		{2, 7, SourceGameModeNormal, true, true, 1, "Jet Pack Bunny", 4, 30, 17, 4, 5},
		{3, 7, SourceGameModeInstagib, true, true, 1, "Pirate", 10, 8, 15, 1, 5},
		{1, 7, SourceGameModeNormal, true, true, 1, "Sinusoidal Sam", 5, 5, 1, 6, 2},
		{7, 5, SourceGameModeNormal, true, false, 1, "The Boss", 6, 14, 13, 6, 6},
	}
	spec := specs[level-1]
	arena, ok := g.ensureMap(spec.mapNumber)
	if !ok {
		arena, _ = g.ensureMap(1)
	}
	g.arena = arena

	// Campaign menu saves P1/P2 as the two human-side PLAYERNUMBER slots.
	// P1 is mandatory; P2 is optional. Enemy campaign scripts later occupy
	// P4 (all levels except 6) or P3+P4 (level6), exactly as frame10 does.
	players := make([]*Player, 0, 4)
	for slot, cfg := range g.campaignPlayers {
		if slot == 0 && cfg.Type == 0 {
			cfg.Type = 1
		}
		if cfg.Type == 0 {
			continue
		}
		id := slot + 1
		p := NewPlayer(id, arena)
		p.Controls = g.controlConfigs[id-1]
		p.Name = cfg.Name
		p.PlayerColor = cfg.Color
		p.ShirtNumber = cfg.Shirt
		p.HatNumber = cfg.Hat
		p.DefaultWeapon = cfg.Gun
		p.PerkNumber = cfg.Perk
		// frame10 campaign block always assigns p1team=p2team=1. When P2 is
		// enabled, normal/instagib campaign matches become a team game.
		p.Team = 1
		p.configureSourceGameMode(spec.mode, spec.lives, arena)
		players = append(players, p)
	}

	enemySlots := []int{4}
	if level == 6 {
		enemySlots = []int{3, 4}
	}
	for i, id := range enemySlots {
		e := NewPlayer(id, arena)
		e.Controls = g.controlConfigs[id-1]
		e.AI = true
		e.Name = spec.name
		if level == 6 && i == 1 {
			e.Name = "Mafia Dude"
			e.ShirtNumber = 4
			e.HatNumber = 5
			e.DefaultWeapon = 3
		} else {
			e.ShirtNumber = spec.shirt
			e.HatNumber = spec.hat
			e.DefaultWeapon = spec.gun
		}
		e.PlayerColor = spec.color
		e.PerkNumber = spec.perk
		// Campaign enemies are always on the opposing side. Source p3/p4 use
		// team2 in level6; in solo levels their reset team0 serves the same
		// purpose, but an explicit opposing team keeps projectile filtering sane.
		e.Team = 2

		// frame10 selects specialized AI movie clips through p4ptype. These are
		// real class variants, not ordinary AI with cosmetic perk values.
		e.AISpecial = campaignAISpecialForLevel(level)
		e.DamageMulti = 1
		e.FirepowerMulti = 1
		switch level {
		case 2:
			// playerAI.adjustrof() weakens Caveman's shots and makes him easier
			// to knock around.
			e.FirepowerMulti = 0.6
			e.DamageMulti = 1.3
		case 3:
			e.FirepowerMulti = 0.8
			e.DamageMulti = 1.2
		case 4:
			e.AIFakeDoubleTime = -80
		case 10:
			// playerAIboss overrides p4gun with MINIGUN before respawn().
			e.DefaultWeapon = 55
			e.DamageMulti = 0.8
		}
		e.configureSourceGameMode(spec.mode, spec.lives, arena)
		players = append(players, e)
	}

	g.players = players
	g.GameMode = spec.mode
	g.TotalLives = spec.lives
	p2Human := g.campaignPlayers[1].Type == 1
	g.TeamGame = spec.mode == SourceGameModeTeams ||
		(p2Human && (spec.mode == SourceGameModeNormal || spec.mode == SourceGameModeInstagib))
	g.CrateON = spec.crate
	g.PowerON = spec.power
	g.campaignMode = true
	g.campaignLevel = level
	g.campaignDynamiteTime = 0
	g.GameWin = false
	g.teamGameWin = false
	g.matchWinCountdown = 0
	g.soloWinFrame = 0
	g.winnerPlayerID = 0
	g.nextEntityID = len(players) + 1
	g.seenDeaths = make(map[int]int, len(players))
	for _, p := range players {
		g.seenDeaths[p.ID] = p.DeathSerial
	}
	g.resetArenaState()
	g.beginFade(screenGameplay, fadePurposeScreen)
}
