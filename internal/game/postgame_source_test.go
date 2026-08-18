package game

import "testing"

func TestPostGameSourceValues(t *testing.T) {
	p := &Player{
		Name:              "Player 1",
		Kills:             3,
		Deaths:            2,
		ShotsFired:        10,
		HitsLanded:        5,
		CratesCollected:   2,
		PowerupsCollected: 1,
		Score:             200,
	}
	got := postGameSourceValues(p)
	want := [10]string{"Player 1", "3", "2", "1.5", "10", "5", "50%", "2", "1", "100"}
	if got != want {
		t.Fatalf("post-game source values=%v want=%v", got, want)
	}
}

func TestPostGameSourceWinnerOutlinesAndTies(t *testing.T) {
	p1 := &Player{ID: 1, Kills: 3, Deaths: 2, ShotsFired: 10, HitsLanded: 5, Score: 200}
	p2 := &Player{ID: 2, Kills: 4, Deaths: 4, ShotsFired: 10, HitsLanded: 5, Score: 100}
	g := &Game{players: []*Player{p1, p2}}
	got := g.postGameWinnerSlots()
	// Kills: p2. K/D parseInt: 1 vs 1 -> source hides on tie.
	// Accuracy: 50 vs 50 -> hidden. Total points: 100 vs 50 -> p1.
	want := [4]int{1, -1, -1, 0}
	if got != want {
		t.Fatalf("post-game winners=%v want=%v", got, want)
	}
}
