package game

import (
	"fmt"
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
