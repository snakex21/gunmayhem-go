package game

import "testing"

func TestCustomSeriesAggregatesRoundOnce(t *testing.T) {
	g := &Game{
		customSeriesRounds: 3,
		customSeriesRound:  1,
		players: []*Player{
			{ID: 1, Name: "Maks", Kills: 4, Deaths: 2, ShotsFired: 20, HitsLanded: 10, Score: 120},
			{ID: 2, Name: "Player 2", Kills: 2, Deaths: 4, ShotsFired: 12, HitsLanded: 3, Score: 70},
		},
		winnerPlayerID: 1,
	}

	g.recordCustomSeriesRound()
	g.recordCustomSeriesRound()

	got := g.customSeriesStats[0]
	if got.Name != "Maks" || got.MapWins != 1 || got.Kills != 4 || got.Deaths != 2 || got.ShotsFired != 20 || got.HitsLanded != 10 || got.Score != 120 || got.RoundsPlayed != 1 {
		t.Fatalf("unexpected aggregate: %+v", got)
	}
	if g.customSeriesStats[1].MapWins != 0 {
		t.Fatalf("loser received map win: %+v", g.customSeriesStats[1])
	}
}

func TestCustomSeriesTeamWinCreditsWinningTeam(t *testing.T) {
	g := &Game{
		customSeriesRounds: 5,
		customSeriesRound:  2,
		TeamGame:           true,
		teamGameWin:        true,
		teamWinner:         1,
		players: []*Player{
			{ID: 1, Name: "A", Team: 1},
			{ID: 2, Name: "B", Team: 2},
			{ID: 3, Name: "C", Team: 1},
		},
	}
	g.recordCustomSeriesRound()
	if g.customSeriesStats[0].MapWins != 1 || g.customSeriesStats[2].MapWins != 1 || g.customSeriesStats[1].MapWins != 0 {
		t.Fatalf("team map wins wrong: %+v", g.customSeriesStats)
	}
}

func TestCustomSeriesRoundClamp(t *testing.T) {
	if got := clampCustomSeriesRounds(-5); got != 1 {
		t.Fatalf("low clamp=%d", got)
	}
	if got := clampCustomSeriesRounds(99); got != 20 {
		t.Fatalf("high clamp=%d", got)
	}
	if got := clampCustomSeriesRounds(7); got != 7 {
		t.Fatalf("normal clamp=%d", got)
	}
}

func TestAbortCustomSeriesReturnsToSetupWithoutAdvancing(t *testing.T) {
	g := New()
	g.customSeriesRounds = 3
	g.customSeriesRound = 2
	g.customSeriesRecordedRound = 1
	g.customSeriesStats[0].MapWins = 1
	g.screen = screenGameplay
	g.paused = true

	g.abortCustomSeriesToSetup()
	if g.customSeriesRound != 0 || g.customSeriesRecordedRound != 0 {
		t.Fatalf("series was not aborted: round=%d recorded=%d", g.customSeriesRound, g.customSeriesRecordedRound)
	}
	if g.customSeriesStats[0].MapWins != 0 {
		t.Fatalf("aborted series stats survived: %+v", g.customSeriesStats[0])
	}
	if g.fadeTarget != screenCustomGame || g.fadePurpose != fadePurposeScreen {
		t.Fatalf("abort targets screen=%v purpose=%v", g.fadeTarget, g.fadePurpose)
	}
	if g.paused {
		t.Fatal("abort left the game paused")
	}
}
