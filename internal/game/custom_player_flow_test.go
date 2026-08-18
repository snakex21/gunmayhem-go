package game

import "testing"

func TestCustomStartUsesActiveSourceSlotsAndTypes(t *testing.T) {
	arena := OriginalMap1()
	g := &Game{
		maps: map[int]Map{1: arena}, arena: arena,
		customMap: 1, customMode: SourceGameModeNormal, customLives: 10,
		seenDeaths: map[int]int{},
	}
	g.initCustomPlayerSetup()
	// Source defaults are P1/P2 human. Make P2 AI, clear P1, enable P3 human:
	// active PLAYERNUMBERs must remain 2 and 3, not be renumbered to 1/2.
	g.customPlayers[0].Type = 0
	g.customPlayers[1].Type = 2
	g.customPlayers[2].Type = 1
	g.startCustomGame()
	if len(g.players) != 2 {
		t.Fatalf("custom players=%d want2", len(g.players))
	}
	if g.players[0].ID != 2 || !g.players[0].AI || g.players[1].ID != 3 || g.players[1].AI {
		t.Fatalf("source slot identity/type mismatch: p0 id=%d ai=%v p1 id=%d ai=%v", g.players[0].ID, g.players[0].AI, g.players[1].ID, g.players[1].AI)
	}
	if !g.fadeActive || g.fadeTarget != screenGameplay {
		t.Fatal("custom START did not enter source fade to gameplay")
	}
}

func TestCustomStartWarnsWithFewerThanTwoPlayers(t *testing.T) {
	arena := OriginalMap1()
	g := &Game{maps: map[int]Map{1: arena}, arena: arena, customMap: 1, customMode: SourceGameModeNormal, customLives: 10}
	g.initCustomPlayerSetup()
	g.customPlayers[1].Type = 0
	g.startCustomGame()
	if g.fadeActive {
		t.Fatal("source START must not fade with only one active player")
	}
	if g.customWarning != 2 || g.customWarningFrame != 2 {
		t.Fatalf("source warning2 state kind=%d frame=%d", g.customWarning, g.customWarningFrame)
	}
}

func TestCustomTeamModeUsesSavedTeams(t *testing.T) {
	arena := OriginalMap1()
	g := &Game{maps: map[int]Map{1: arena}, arena: arena, customMap: 1, customMode: SourceGameModeTeams, customLives: 10, seenDeaths: map[int]int{}}
	g.initCustomPlayerSetup()
	g.customPlayers[0].Team = 2
	g.customPlayers[1].Team = 2
	g.startCustomGame()
	if !g.TeamGame || len(g.players) != 2 || g.players[0].Team != 2 || g.players[1].Team != 2 {
		t.Fatalf("team source config mismatch teamgame=%v teams=%v/%v", g.TeamGame, g.players[0].Team, g.players[1].Team)
	}
}
