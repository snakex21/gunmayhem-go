package game

import (
	"fmt"
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	customSeriesMinRounds = 1
	customSeriesMaxRounds = 20
)

type customSeriesPlayerStats struct {
	Name         string
	MapWins      int
	Kills        int
	Deaths       int
	ShotsFired   int
	HitsLanded   int
	Score        int
	RoundsPlayed int
}

func clampCustomSeriesRounds(v int) int {
	if v < customSeriesMinRounds {
		return customSeriesMinRounds
	}
	if v > customSeriesMaxRounds {
		return customSeriesMaxRounds
	}
	return v
}

func (g *Game) beginCustomSeries() {
	if g == nil {
		return
	}
	g.customSeriesRounds = clampCustomSeriesRounds(g.customSeriesRounds)
	g.ensureCustomPlaylistState()
	g.customSeriesRound = 1
	g.customSeriesRecordedRound = 0
	g.customSeriesStats = [4]customSeriesPlayerStats{}
	g.customSeriesStartingNext = false
}

func (g *Game) customSeriesIsFinalRound() bool {
	if g == nil {
		return true
	}
	return g.customSeriesRound >= clampCustomSeriesRounds(g.customSeriesRounds)
}

func (g *Game) recordCustomSeriesRound() {
	if g == nil || g.campaignMode || g.customSeriesRound <= 0 || g.customSeriesRecordedRound == g.customSeriesRound {
		return
	}
	for _, p := range g.players {
		if p == nil || p.ID < 1 || p.ID > len(g.customSeriesStats) || p.IsDouble {
			continue
		}
		s := &g.customSeriesStats[p.ID-1]
		name := sanitizeRemotePlayerName(p.Name)
		if name == "" {
			name = fmt.Sprintf("Player %d", p.ID)
		}
		s.Name = name
		s.Kills += p.Kills
		s.Deaths += p.Deaths
		s.ShotsFired += p.ShotsFired
		s.HitsLanded += p.HitsLanded
		s.Score += p.Score
		s.RoundsPlayed++
	}

	if g.TeamGame && g.teamGameWin && g.teamWinner != 0 {
		for _, p := range g.players {
			if p == nil || p.ID < 1 || p.ID > len(g.customSeriesStats) || p.IsDouble || p.Team != g.teamWinner {
				continue
			}
			g.customSeriesStats[p.ID-1].MapWins++
		}
	} else if g.winnerPlayerID >= 1 && g.winnerPlayerID <= len(g.customSeriesStats) {
		g.customSeriesStats[g.winnerPlayerID-1].MapWins++
	}
	g.customSeriesRecordedRound = g.customSeriesRound
}

func (g *Game) startNextCustomSeriesRound() {
	if g == nil || g.customSeriesIsFinalRound() {
		return
	}
	g.customSeriesRound++
	g.customSeriesStartingNext = true
	g.resetOnlinePostGameReady()
	g.startCustomGame()
	g.customSeriesStartingNext = false
}

// abortCustomSeriesToSetup is used by the pause-menu EXIT action. This is not
// a completed round: do not enter Post Game, do not record stats and do not
// advance the playlist. Online hosts keep the session alive and return everyone
// to the SLOTS/character step; local Custom Game returns to MATCH setup.
func (g *Game) abortCustomSeriesToSetup() {
	if g == nil {
		return
	}
	g.paused = false
	g.fadeActive = false
	g.fadeFrame = 0
	g.GameWin = false
	g.teamGameWin = false
	g.matchWinCountdown = 0
	g.soloWinFrame = 0
	g.winnerPlayerID = 0
	g.winnerAnimFrame = 0
	g.teamWinAnimFrame = 0
	g.customSeriesRound = 0
	g.customSeriesRecordedRound = 0
	g.customSeriesStartingNext = false
	g.customSeriesStats = [4]customSeriesPlayerStats{}
	g.resetOnlinePostGameReady()
	g.resetChaosRuntime()
	g.customRulesOpen = false
	g.customChaosOpen = false
	g.menuPressed = menuHitNone
	g.pausePressed = pauseHitNone
	if g.onlineMatchSetup && g.netplay != nil && g.netplay.mode == netplayHost {
		g.customOnlineMode = true
		g.customNetworkOpen = false
		g.setCustomV2Page(customPagePlayers)
		g.multiplayerMessage = "MATCH ABORTED - SESSION STILL OPEN"
	} else {
		g.onlineMatchSetup = false
		g.setCustomV2Page(customPageGame)
	}
	g.beginFade(screenCustomGame, fadePurposeScreen)
}

func (g *Game) finishCustomSeriesToSummary() {
	if g == nil {
		return
	}
	g.beginFade(screenSeriesSummary, fadePurposeScreen)
}

func (g *Game) updateSeriesSummaryMenu() {
	if g == nil {
		return
	}
	if g.netplay != nil && g.netplay.mode == netplayClient {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			_ = g.CloseNetplay()
			g.screen = screenCustomGame
			g.customOnlineMode = true
			g.onlineMatchSetup = false
			g.customNetworkOpen = true
			g.multiplayerMessage = "DISCONNECTED"
			g.beginMultiplayerServerBrowser()
		}
		return
	}
	mx, my := ebiten.CursorPosition()
	button := Rect{X: 615, Y: 525, W: 240, H: 48}
	hit := menuHitNone
	if button.Contains(float64(mx), float64(my)) {
		hit = menuHitGameContinue
	}
	activated, ok := g.resolveMenuRelease(hit)
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || (ok && activated == menuHitGameContinue) {
		g.playSourceSFX("menu.wav", false)
		g.customPage = customPageGame
		g.customMenuX = 1800
		g.customSeriesRound = 0
		g.customSeriesRecordedRound = 0
		g.beginFade(screenCustomGame, fadePurposeScreen)
	}
}

func (g *Game) customSeriesLeaderText() string {
	if g == nil || g.customSeriesRound <= 0 {
		return ""
	}
	bestWins := -1
	leaders := make([]string, 0, 4)
	for i, s := range g.customSeriesStats {
		if s.RoundsPlayed <= 0 {
			continue
		}
		name := s.Name
		if name == "" {
			name = fmt.Sprintf("Player %d", i+1)
		}
		switch {
		case s.MapWins > bestWins:
			bestWins = s.MapWins
			leaders = []string{name}
		case s.MapWins == bestWins:
			leaders = append(leaders, name)
		}
	}
	if len(leaders) == 0 {
		return ""
	}
	if len(leaders) > 1 {
		return fmt.Sprintf("SERIES LEAD: TIE AT %d MAP WINS", bestWins)
	}
	return fmt.Sprintf("SERIES LEADER: %s - %d MAP WINS", leaders[0], bestWins)
}

func seriesAccuracy(s customSeriesPlayerStats) int {
	if s.ShotsFired <= 0 {
		return 0
	}
	return int(float64(s.HitsLanded)/float64(s.ShotsFired)*100 + 0.5)
}

func seriesKD(s customSeriesPlayerStats) string {
	if s.Deaths <= 0 {
		if s.Kills <= 0 {
			return "0.00"
		}
		return fmt.Sprintf("%.2f", float64(s.Kills))
	}
	return fmt.Sprintf("%.2f", float64(s.Kills)/float64(s.Deaths))
}

func (g *Game) drawSeriesSummary(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, 0, 0, 900, 600, color.NRGBA{R: 0x18, G: 0x1c, B: 0x22, A: 0xff})
	ebitenutil.DrawRect(screen, 28, 22, 844, 556, color.NRGBA{R: 0xf3, G: 0xf3, B: 0xf3, A: 0xff})
	drawSourceMenuText(screen, "SERIES SUMMARY", menuFontTwCen, 42, color.Black, 52, 42)
	drawSourceMenuText(screen, fmt.Sprintf("%d MAPS PLAYED", clampCustomSeriesRounds(g.customSeriesRounds)), menuFontCondensedExtraBold, 16, color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff}, 55, 91)
	drawSourceMenuText(screen, "RANKING: MAP WINS > KILLS > POINTS", menuFontCondensed, 13.5, color.NRGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xff}, 610, 94)

	headY := 134.0
	ebitenutil.DrawRect(screen, 52, headY, 796, 34, color.NRGBA{R: 0x28, G: 0x32, B: 0x42, A: 0xff})
	cols := []struct {
		text string
		x    float64
	}{{"PLAYER", 68}, {"MAP WINS", 330}, {"KILLS", 455}, {"DEATHS", 535}, {"K/D", 625}, {"ACC", 700}, {"POINTS", 770}}
	for _, c := range cols {
		drawSourceMenuText(screen, c.text, menuFontCondensedExtraBold, 14, color.White, c.x, headY+8)
	}

	type row struct {
		slot int
		stat customSeriesPlayerStats
	}
	rows := make([]row, 0, 4)
	for i, s := range g.customSeriesStats {
		if s.RoundsPlayed <= 0 {
			continue
		}
		rows = append(rows, row{slot: i, stat: s})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].stat.MapWins != rows[j].stat.MapWins {
			return rows[i].stat.MapWins > rows[j].stat.MapWins
		}
		if rows[i].stat.Kills != rows[j].stat.Kills {
			return rows[i].stat.Kills > rows[j].stat.Kills
		}
		return rows[i].stat.Score > rows[j].stat.Score
	})

	for i, r := range rows {
		y := 176.0 + float64(i)*72
		fill := color.NRGBA{R: 0xe3, G: 0xe3, B: 0xe3, A: 0xff}
		if i == 0 {
			fill = color.NRGBA{R: 0xff, G: 0xcc, B: 0x80, A: 0xff}
		}
		ebitenutil.DrawRect(screen, 52, y, 796, 60, fill)
		name := r.stat.Name
		if name == "" {
			name = fmt.Sprintf("Player %d", r.slot+1)
		}
		drawSourceMenuText(screen, name, menuFontTwCen, 20, color.Black, 68, y+16)
		drawSourceMenuText(screen, fmt.Sprintf("%d", r.stat.MapWins), menuFontCenturyGothicBold, 22, color.Black, 360, y+14)
		drawSourceMenuText(screen, fmt.Sprintf("%d", r.stat.Kills), menuFontCenturyGothicBold, 20, color.Black, 470, y+15)
		drawSourceMenuText(screen, fmt.Sprintf("%d", r.stat.Deaths), menuFontCenturyGothicBold, 20, color.Black, 552, y+15)
		drawSourceMenuText(screen, seriesKD(r.stat), menuFontCenturyGothicBold, 18, color.Black, 620, y+17)
		drawSourceMenuText(screen, fmt.Sprintf("%d%%", seriesAccuracy(r.stat)), menuFontCenturyGothicBold, 18, color.Black, 700, y+17)
		drawSourceMenuText(screen, fmt.Sprintf("%d", r.stat.Score), menuFontCenturyGothicBold, 18, color.Black, 777, y+17)
	}

	if len(rows) > 0 {
		winner := rows[0].stat.Name
		if winner == "" {
			winner = fmt.Sprintf("Player %d", rows[0].slot+1)
		}
		drawSourceMenuText(screen, "SERIES WINNER: "+winner, menuFontCondensedExtraBold, 24, color.Black, 55, 492)
	}

	button := Rect{X: 615, Y: 525, W: 240, H: 48}
	ebitenutil.DrawRect(screen, button.X, button.Y, button.W, button.H, color.NRGBA{R: 0x24, G: 0x27, B: 0x2d, A: 0xff})
	ebitenutil.DrawRect(screen, button.X+2, button.Y+2, button.W-4, button.H-4, color.NRGBA{R: 0xf2, G: 0x8c, B: 0x00, A: 0xff})
	label := "BACK TO CUSTOM GAME"
	if g.netplay != nil && g.netplay.mode == netplayClient {
		label = "WAITING FOR HOST"
	}
	drawSourceMenuText(screen, label, menuFontTwCen, 17, color.Black, button.X+12, button.Y+13)
}
