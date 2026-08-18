package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	campaignHitSetGM1 = 1801
	campaignHitSetGM2 = 1802
	gm2CampaignHitMissionBase = 2000
	gm2CampaignHitStart = 2100
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
	ID     int
	Name   string
	Color  int
	Shirt  int
	Hat    int
	Gun    int
	Perk   int
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
	{MapNumber: 9, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Easy AI", 4, 1, 33, 1, 0}}},
	{MapNumber: 19, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Easy AI", 4, 1, 43, 1, 0}, {4, "Easy AI 2", 5, 1, 43, 1, 0}}},
	{MapNumber: 2, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Triple Jumper", 20, 17, 37, 2, 3}}},
	{MapNumber: 7, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Moderate AI", 18, 6, 46, 2, 0}}},
	{MapNumber: 4, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "The Ghost", 12, 5, 5, 4, 0}}},
	{MapNumber: 6, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Moderate AI", 18, 6, 46, 2, 0}, {4, "Moderate AI 2", 17, 6, 31, 4, 0}}},
	{MapNumber: 14, Lives: 3, Crates: false, Powerups: false, Enemies: []gm2CampaignEnemy{{3, "Easy AI", 4, 1, 33, 6, 0}}},
	{MapNumber: 1, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Hard AI", 9, 4, 9, 4, 0}}},
	{MapNumber: 20, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Gangster", 11, 2, 3, 44, 0}, {4, "Gangster", 12, 3, 4, 44, 0}}},
	{MapNumber: 12, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Tiny Guy", 1, 7, 11, 5, 0}}},
	{MapNumber: 3, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Hard AI", 9, 4, 9, 4, 0}, {4, "Hard AI 2", 10, 5, 8, 4, 0}}},
	{MapNumber: 21, Lives: 5, Crates: false, Powerups: false, Enemies: []gm2CampaignEnemy{{3, "Pistol Man", 8, 15, 41, 4, 0}}},
	{MapNumber: 18, Lives: 5, Crates: true, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Too Fast", 3, 13, 42, 5, 0}}},
	{MapNumber: 10, Lives: 5, Crates: false, Powerups: true, Enemies: []gm2CampaignEnemy{{3, "Terminator", 11, 12, 1, 9, 0}}},
	{MapNumber: 16, Lives: 5, Crates: false, Powerups: false, Enemies: []gm2CampaignEnemy{{3, "Hard AI", 14, 8, 15, 2, 0}, {4, "Hard AI 2", 14, 14, 15, 2, 0}, {2, "Hard AI 3", 14, 4, 15, 2, 0}}},
}

func campaignSetRects() (Rect, Rect) {
	return Rect{X: 610, Y: 28, W: 110, H: 30}, Rect{X: 730, Y: 28, W: 140, H: 30}
}

func campaignSetHitAt(x, y float64) int {
	gm1, gm2 := campaignSetRects()
	if gm1.Contains(x, y) {
		return campaignHitSetGM1
	}
	if gm2.Contains(x, y) {
		return campaignHitSetGM2
	}
	return campaignHitNone
}

func (g *Game) drawCampaignSetTabs(screen *ebiten.Image) {
	gm1, gm2 := campaignSetRects()
	for _, tab := range []struct {
		r Rect
		label string
		active bool
	}{
		{gm1, "GM1", !g.campaignSetGM2},
		{gm2, "GM2", g.campaignSetGM2},
	} {
		c := color.NRGBA{R: 0xd0, G: 0xd0, B: 0xd0, A: 0xff}
		if tab.active {
			c = color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
		}
		ebitenutil.DrawRect(screen, tab.r.X, tab.r.Y, tab.r.W, tab.r.H, c)
		drawSourceMenuText(screen, tab.label, menuFontCondensedExtraBold, 18, color.Black, tab.r.X+34, tab.r.Y+4)
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
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.beginFade(screenMainMenu, fadePurposeScreen)
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF10) {
		g.unlockAllGM2CampaignLevelsCheat()
		g.playSourceSFX("menu.wav", false)
	}

	mx, my := ebiten.CursorPosition()
	x, y := float64(mx), float64(my)
	hit := campaignSetHitAt(x, y)
	if hit == campaignHitNone {
		back := sourceMenuHitRect(campaignBackLocal, 27, 525.15, 1, 1)
		start := Rect{X: 620, Y: 522, W: 250, H: 54}
		if back.Contains(x, y) {
			hit = campaignHitBack
		} else if start.Contains(x, y) && g.campaignLevel >= 1 && g.campaignLevel <= 16 && g.gm2CampaignLevels[g.campaignLevel-1] != 0 {
			hit = gm2CampaignHitStart
		} else {
			for i := 0; i < 16; i++ {
				if gm2CampaignCardRect(i).Contains(x, y) && g.gm2CampaignLevels[i] != 0 {
					hit = gm2CampaignHitMissionBase + i + 1
					break
				}
			}
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
	switch {
	case pressed == campaignHitSetGM1:
		g.campaignSetGM2 = false
		g.campaignLevel = 0
		g.campaignPhase = 1
		g.playSourceSFX("menu.wav", false)
	case pressed == campaignHitSetGM2:
		// already selected
	case pressed == campaignHitBack:
		g.playSourceSFX("menu.wav", false)
		g.beginFade(screenMainMenu, fadePurposeScreen)
	case pressed == gm2CampaignHitStart:
		g.playSourceSFX("menu.wav", false)
		g.startGM2CampaignMission()
	case pressed > gm2CampaignHitMissionBase && pressed <= gm2CampaignHitMissionBase+16:
		g.campaignLevel = pressed - gm2CampaignHitMissionBase
		g.playSourceSFX("menu.wav", false)
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
	g.drawCampaignSetTabs(screen)
	drawSourceRaster(screen, g.assets.CampaignBack, 27, 525.15, 1, 1, 1)
	drawSourceMenuText(screen, "Back", menuFontCondensed, 26.5, color.Black, 74.05, 536.35)

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

	start := Rect{X: 620, Y: 522, W: 250, H: 54}
	startColor := color.NRGBA{R: 0x75, G: 0x75, B: 0x75, A: 0xff}
	if g.campaignLevel >= 1 && g.campaignLevel <= 16 && g.gm2CampaignLevels[g.campaignLevel-1] != 0 {
		startColor = color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
	}
	ebitenutil.DrawRect(screen, start.X, start.Y, start.W, start.H, startColor)
	label := "SELECT MISSION"
	if g.campaignLevel >= 1 && g.campaignLevel <= 16 {
		label = "START: " + gm2CampaignTitles[g.campaignLevel-1]
	}
	drawSourceMenuText(screen, label, menuFontCondensedExtraBold, 19, color.Black, start.X+12, start.Y+13)
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
		p.DefaultWeapon = cfg.Gun
		p.PerkNumber = cfg.Perk
		p.Team = 1
		p.configureSourceGameMode(SourceGameModeNormal, spec.Lives, arena)
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
		e.DefaultWeapon = src.Gun
		e.PerkNumber = src.Perk
		e.Team = 2
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
	g.campaignMode = true
	g.campaignSetGM2 = true
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
