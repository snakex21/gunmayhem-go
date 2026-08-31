package game

import (
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	campaignHitSetGM1         = 1801
	campaignHitSetGM2         = 1802
	campaignHitSetExtra       = 1803
	gm2CampaignHitMissionBase = 2000
	gm2CampaignHitStart       = 2100
)

var gm2CampaignTitles = [16]string{
	"Tutorial",
	"Easy AI",
	"2x Easy AI",
	"Triple Jump",
	"Moderate AI",
	"Invisible Man",
	"2x Moderate AI",
	"Dynamite Only",
	"Hard AI",
	"Gangster Shotguns",
	"Mini AI",
	"2x Hard AI",
	"Flawless Pistol Duel",
	"Too Fast",
	"De-materializer",
	"Final Test",
}

type gm2CampaignEnemy struct {
	ID    int
	Name  string
	Color int
	Shirt int
	Hat   int
	Eyes  int
	Gun   int
	Perk  int
}

type gm2CampaignMission struct {
	MapNumber int
	Lives     int
	Crates    bool
	Powerups  bool
	Enemies   []gm2CampaignEnemy
}

var gm2CampaignMissions = [16]gm2CampaignMission{
	{MapNumber: 5, Lives: 99999, Crates: false, Powerups: false},
	{MapNumber: 9, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Easy AI", 4, 1, 33, 9, 1, 0}}},
	{MapNumber: 19, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Easy AI", 4, 1, 43, 9, 1, 0}, {4, "Easy AI 2", 5, 1, 43, 9, 1, 0}}},
	{MapNumber: 2, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Triple Jumper", 20, 17, 37, 10, 2, 3}}},
	{MapNumber: 7, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Moderate AI", 18, 6, 46, 1, 2, 0}}},
	{MapNumber: 4, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "The Ghost", 12, 5, 5, 1, 4, 0}}},
	{MapNumber: 6, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Moderate AI", 18, 6, 46, 1, 2, 0}, {4, "Moderate AI 2", 17, 6, 31, 1, 4, 0}}},
	{MapNumber: 14, Lives: 3, Crates: false, Powerups: false, Enemies: []gm2CampaignEnemy{{3, "Easy AI", 4, 1, 33, 9, 6, 0}}},
	{MapNumber: 1, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Hard AI", 9, 4, 9, 11, 4, 0}}},
	{MapNumber: 20, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Gangster", 11, 2, 3, 4, 44, 0}, {4, "Gangster", 12, 3, 4, 4, 44, 0}}},
	{MapNumber: 12, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Tiny Guy", 1, 7, 11, 4, 5, 0}}},
	{MapNumber: 3, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Hard AI", 9, 4, 9, 11, 4, 0}, {4, "Hard AI 2", 10, 5, 8, 11, 4, 0}}},
	{MapNumber: 21, Lives: 5, Crates: false, Powerups: false, Enemies: []gm2CampaignEnemy{{3, "Pistol Man", 8, 15, 41, 11, 4, 0}}},
	{MapNumber: 18, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Too Fast", 3, 13, 42, 9, 5, 0}}},
	{MapNumber: 10, Lives: 5, Crates: false, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Terminator", 11, 12, 1, 4, 9, 0}}},
	{MapNumber: 16, Lives: 5, Crates: false, Powerups: false, Enemies: []gm2CampaignEnemy{{3, "Hard AI", 14, 8, 15, 3, 2, 0}, {4, "Hard AI 2", 14, 14, 15, 4, 2, 0}, {2, "Hard AI 3", 14, 4, 15, 2, 2, 0}}},
}

func campaignSetRects() (Rect, Rect, Rect) {
	return Rect{X: 550, Y: 28, W: 90, H: 30},
		Rect{X: 650, Y: 28, W: 90, H: 30},
		Rect{X: 750, Y: 28, W: 120, H: 30}
}

func campaignSetHitAt(x, y float64) int {
	gm1, gm2, extra := campaignSetRects()
	if gm1.Contains(x, y) {
		return campaignHitSetGM1
	}
	if gm2.Contains(x, y) {
		return campaignHitSetGM2
	}
	if extra.Contains(x, y) {
		return campaignHitSetExtra
	}
	return campaignHitNone
}

func (g *Game) drawCampaignSetTabs(screen *ebiten.Image) {
	gm1, gm2, extra := campaignSetRects()
	for _, tab := range []struct {
		r      Rect
		label  string
		active bool
	}{
		{gm1, "GM1", !g.campaignSetGM2 && !g.campaignSetExtra},
		{gm2, "GM2", g.campaignSetGM2 && !g.campaignSetExtra},
		{extra, "EXTRA", g.campaignSetExtra},
	} {
		c := color.NRGBA{R: 0xd0, G: 0xd0, B: 0xd0, A: 0xff}
		if tab.active {
			c = color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
		}
		ebitenutil.DrawRect(screen, tab.r.X, tab.r.Y, tab.r.W, tab.r.H, c)
		offset := 24.0
		if tab.label == "EXTRA" {
			offset = 30
		}
		drawSourceMenuText(screen, tab.label, menuFontCondensedExtraBold, 18, color.Black, tab.r.X+offset, tab.r.Y+4)
	}
}

func gm2CampaignCardRect(index int) Rect {
	col := index % 4
	row := index / 4
	return Rect{X: 28 + float64(col)*216, Y: 95 + float64(row)*98, W: 198, H: 82}
}

func (g *Game) unlockAllGM2CampaignLevelsCheat() {
	for i := range g.gm2CampaignLevels {
		if g.gm2CampaignLevels[i] == 0 {
			g.gm2CampaignLevels[i] = 1
		}
	}
	_ = g.saveProgress()
}

func (g *Game) updateGM2CampaignMenu() {
	if g.allowCheats && inpututil.IsKeyJustPressed(ebiten.KeyF10) && g.campaignPhase == 1 {
		g.unlockAllGM2CampaignLevelsCheat()
		g.playSourceSFX("menu.wav", false)
	}
	if g.campaignPhase != 1 {
		g.updateCampaignPlayerCards()
		g.updateCampaignNameInput()
	}
	if g.campaignPhase == 2 {
		g.campaignDetailAlpha += .2
		if g.campaignDetailAlpha >= 1 {
			g.campaignDetailAlpha = 1
			g.campaignPhase = 3
		}
	} else if g.campaignPhase == 4 {
		g.campaignDetailAlpha -= .2
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
			g.campaignNameFocus = 0
		}
		return
	}

	mx, my := ebiten.CursorPosition()
	x, y := float64(mx), float64(my)
	hit := campaignHitNone
	if g.campaignPhase == 1 {
		hit = campaignSetHitAt(x, y)
		back := sourceMenuHitRect(campaignBackLocal, 27, 525.15, 1, 1)
		if hit == campaignHitNone && back.Contains(x, y) {
			hit = campaignHitBack
		} else if hit == campaignHitNone {
			for i := 0; i < 16; i++ {
				if gm2CampaignCardRect(i).Contains(x, y) && g.gm2CampaignLevels[i] != 0 {
					hit = gm2CampaignHitMissionBase + i + 1
					break
				}
			}
		}
	} else if g.campaignPhase == 3 {
		if playerHit := g.campaignPlayerHitAt(x, y); playerHit != campaignHitNone {
			hit = playerHit
		}
		back := sourceMenuHitRect(campaignBackLocal, 27, 525.15, 1, 1)
		start := sourceMenuHitRect(campaignStartLocal, 469.1, 522.15, 1, 1)
		if hit == campaignHitNone && back.Contains(x, y) {
			hit = campaignHitBack
		} else if hit == campaignHitNone && start.Contains(x, y) {
			hit = gm2CampaignHitStart
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.campaignPressed = hit
		return
	}
	if !inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		return
	}
	pressed := g.campaignPressed
	g.campaignPressed = campaignHitNone
	if pressed == campaignHitNone || pressed != hit {
		return
	}
	if g.activateCampaignPlayerHit(pressed) {
		return
	}
	switch {
	case pressed == campaignHitSetGM1:
		g.campaignSetGM2 = false
		g.campaignSetExtra = false
		g.campaignLevel = 0
		g.campaignPhase = 1
		g.playSourceSFX("menu.wav", false)
	case pressed == campaignHitSetGM2:
		// already selected
	case pressed == campaignHitSetExtra:
		g.campaignSetGM2 = false
		g.campaignSetExtra = true
		g.campaignLevel = 0
		g.extraCampaignPage = 0
		g.campaignPhase = 1
		g.playSourceSFX("menu.wav", false)
	case pressed == campaignHitBack:
		g.playSourceSFX("menu.wav", false)
		if g.campaignPhase == 1 {
			g.beginFade(screenMainMenu, fadePurposeScreen)
		} else if g.campaignPhase == 3 {
			g.campaignPhase = 4
			g.campaignNameFocus = 0
		}
	case pressed == gm2CampaignHitStart:
		g.playSourceSFX("menu.wav", false)
		g.startGM2CampaignMission()
	case pressed > gm2CampaignHitMissionBase && pressed <= gm2CampaignHitMissionBase+16:
		g.openGM2CampaignMissionSetup(pressed - gm2CampaignHitMissionBase)
		g.playSourceSFX("menu.wav", false)
	}
}

func (g *Game) openGM2CampaignMissionSetup(level int) {
	if level < 1 || level > len(gm2CampaignMissions) || g.gm2CampaignLevels[level-1] == 0 {
		return
	}
	g.campaignLevel = level
	g.campaignPhase = 2
	g.campaignDetailAlpha = 0
	g.campaignNameFocus = 0
	for i := range g.campaignEditor {
		g.campaignEditor[i] = 0
	}
	if level == 1 || level == 16 {
		g.campaignPlayers[1].Type = 0
	}
}

func gm2CampaignStateLabel(state int) string {
	switch state {
	case 2:
		return "COMPLETED"
	case 1:
		return "AVAILABLE"
	default:
		return "LOCKED"
	}
}

func (g *Game) drawGM2CampaignMenu(screen *ebiten.Image) {
	drawSourceRaster(screen, g.assets.CampaignBase, 0, 0, 1, 1, 1)
	drawSourceMenuText(screen, "CAMPAIGN", menuFontCondensed, 42.4, color.Black, 34, 27.75)
	drawSourceRaster(screen, g.assets.CampaignBack, 27, 525.15, 1, 1, 1)
	drawSourceMenuText(screen, "Back", menuFontCondensed, 26.5, color.Black, 74.05, 536.35)
	if g.campaignPhase != 1 {
		a := g.campaignDetailAlpha
		if a > 0 {
			g.drawCampaignPlayerCards(screen, a)
			g.drawGM2CampaignMissionInfo(screen, a)
			if g.campaignPhase == 3 {
				drawSourceRaster(screen, g.assets.CampaignStartButton, 469.1, 522.15, 1, 1, a)
				drawSourceMenuText(screen, "Start Mission", menuFontCondensedExtraBold, 30,
					color.NRGBA{A: uint8(255 * a)}, 580, 529)
			}
		}
		return
	}
	g.drawCampaignSetTabs(screen)

	for i := 0; i < 16; i++ {
		r := gm2CampaignCardRect(i)
		state := g.gm2CampaignLevels[i]
		c := color.NRGBA{R: 0xb8, G: 0xb8, B: 0xb8, A: 0xff}
		if state == 0 {
			c = color.NRGBA{R: 0x72, G: 0x72, B: 0x72, A: 0xff}
		}
		if g.campaignLevel == i+1 {
			c = color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
		}
		ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, c)
		drawSourceMenuText(screen, gm2CampaignTitles[i], menuFontCondensedExtraBold, 17.5, color.Black, r.X+7, r.Y+6)
		mapName := gm2MapDisplayNames[gm2CampaignMissions[i].MapNumber]
		drawSourceMenuText(screen, mapName, menuFontTwCen, 12.5, color.Black, r.X+7, r.Y+32)
		drawSourceMenuText(screen, gm2CampaignStateLabel(state), menuFontTwCen, 11.5, color.Black, r.X+7, r.Y+57)
		drawSourceMenuText(screen, "#"+itoaFast(i+1), menuFontCondensedExtraBold, 18, color.Black, r.X+r.W-32, r.Y+56)
	}

	drawSourceMenuText(screen, "CLICK A MISSION TO CUSTOMIZE", menuFontCondensedExtraBold, 15, color.Black, 642, 538)
}

func (g *Game) drawGM2CampaignMissionInfo(screen *ebiten.Image, alpha float64) {
	if g.campaignLevel < 1 || g.campaignLevel > len(gm2CampaignMissions) || alpha <= 0 {
		return
	}
	spec := gm2CampaignMissions[g.campaignLevel-1]
	a := uint8(255 * alpha)
	panel := color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: a}
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: a}
	orange := color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: a}
	ebitenutil.DrawRect(screen, 467.1, 100, 405, 400, panel)
	drawSourceMenuText(screen, "GM2 MISSION "+strconv.Itoa(g.campaignLevel), menuFontCondensedExtraBold, 17, orange, 489, 122)
	drawSourceMenuText(screen, gm2CampaignTitles[g.campaignLevel-1], menuFontTwCen, 29, white, 489, 151)
	drawSourceMenuText(screen, gm2MapDisplayNames[spec.MapNumber], menuFontCondensedExtraBold, 17, white, 489, 202)
	lives := strconv.Itoa(spec.Lives)
	if spec.Lives > 9999 {
		lives = "UNLIMITED"
	}
	drawSourceMenuText(screen, "LIVES: "+lives, menuFontCondensedExtraBold, 15, white, 489, 232)
	crateText := "OFF"
	if spec.Crates {
		crateText = "ON"
	}
	powerText := "OFF"
	if spec.Powerups {
		powerText = "ON"
	}
	drawSourceMenuText(screen, "CRATES: "+crateText+"    POWERUPS: "+powerText, menuFontCondensedExtraBold, 14, white, 489, 257)
	drawSourceMenuText(screen, "OPPONENTS", menuFontCondensedExtraBold, 16, orange, 489, 292)
	if len(spec.Enemies) == 0 {
		drawSourceMenuText(screen, "Tutorial / training mission", menuFontCondensed, 14, white, 489, 319)
	} else {
		for i, enemy := range spec.Enemies {
			line := enemy.Name + " - " + weaponDisplayName(enemy.Gun)
			drawSourceMenuText(screen, line, menuFontCondensed, 14, white, 489, 319+float64(i)*25)
		}
	}
	coop := "P2 CO-OP AVAILABLE"
	if g.campaignLevel == 1 || g.campaignLevel == 16 {
		coop = "SOLO MISSION"
	}
	drawSourceMenuText(screen, coop, menuFontCondensedExtraBold, 14, orange, 489, 408)
	drawSourceMenuText(screen, "Customize your character on the left, then start the mission.", menuFontTwCen, 12.5, white, 489, 451)
}

func (g *Game) startGM2CampaignMission() {
	level := g.campaignLevel
	if level < 1 || level > 16 || g.fadeActive || g.gm2CampaignLevels[level-1] == 0 {
		return
	}
	spec := gm2CampaignMissions[level-1]
	arena, ok := g.ensureMap(gm2MapID(spec.MapNumber))
	if !ok {
		return
	}

	players := make([]*Player, 0, 4)
	allowCoop := level != 1 && level != 16
	for slot, cfg := range g.campaignPlayers {
		if slot == 0 && cfg.Type == 0 {
			cfg.Type = 1
		}
		if slot == 1 && !allowCoop {
			continue
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
		p.EyeNumber = cfg.Eyes
		p.DefaultWeapon = cfg.Gun
		p.PerkNumber = cfg.Perk
		p.Team = 1
		p.configureSourceGameMode(SourceGameModeNormal, spec.Lives, arena)
		p.ThrowableType = normalizedThrowableType(cfg.Throwable)
		players = append(players, p)
	}

	for _, src := range spec.Enemies {
		e := NewPlayer(src.ID, arena)
		e.Controls = g.controlConfigs[src.ID-1]
		e.AI = true
		e.Name = src.Name
		e.PlayerColor = src.Color
		e.ShirtNumber = src.Shirt
		e.HatNumber = src.Hat
		e.EyeNumber = src.Eyes
		e.UseGM2Cosmetics = true
		e.DefaultWeapon = src.Gun
		e.PerkNumber = src.Perk
		e.Team = 2
		switch level {
		case 6:
			e.AISpecial = sourceAISpecialGM2Invisible
		case 11:
			e.AISpecial = sourceAISpecialGM2Mini
		}
		e.configureSourceGameMode(SourceGameModeNormal, spec.Lives, arena)
		players = append(players, e)
	}

	g.arena = arena
	g.players = players
	g.GameMode = SourceGameModeNormal
	g.TotalLives = spec.Lives
	g.TeamGame = allowCoop && len(players) > 0 && len(g.campaignPlayers) > 1 && g.campaignPlayers[1].Type == 1
	g.CrateON = spec.Crates
	g.PowerON = spec.Powerups
	g.customMatchActive = false
	g.matchModifiers = make(map[int]bool)
	g.campaignMode = true
	g.campaignSetGM2 = true
	g.campaignSetExtra = false
	g.campaignLevel = level
	g.GameWin = false
	g.teamGameWin = false
	g.matchWinCountdown = 0
	g.soloWinFrame = 0
	g.winnerPlayerID = 0
	g.nextEntityID = 5
	g.seenDeaths = make(map[int]int, len(players))
	for _, p := range players {
		g.seenDeaths[p.ID] = p.DeathSerial
	}
	g.resetArenaState()
	g.beginFade(screenGameplay, fadePurposeScreen)
}

func (g *Game) completeGM2CampaignLevel(level int) {
	if level < 1 || level > 16 {
		return
	}
	g.gm2CampaignLevels[level-1] = 2
	if level >= 2 && level < 16 && g.gm2CampaignLevels[level] == 0 {
		g.gm2CampaignLevels[level] = 1
	}
	_ = g.saveProgress()
}
