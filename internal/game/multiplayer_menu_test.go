package game

import "testing"

func TestDevelopedMainMenuMultiplayerHit(t *testing.T) {
	if got := mainMenuHitAt(mainMultiplayerRect.X+20, mainMultiplayerRect.Y+20); got != menuHitMainMultiplayer {
		t.Fatalf("main multiplayer hit=%d want %d", got, menuHitMainMultiplayer)
	}
}

func TestMultiplayerMenuHitboxes(t *testing.T) {
	checks := []struct {
		r    Rect
		want int
	}{
		{multiplayerHostRect, menuHitMultiplayerHost},
		{multiplayerJoinRect, menuHitMultiplayerJoin},
		{multiplayerBackRect, menuHitMultiplayerBack},
	}
	for _, tc := range checks {
		if got := multiplayerMenuHitAt(tc.r.X+tc.r.W/2, tc.r.Y+tc.r.H/2); got != tc.want {
			t.Errorf("hit=%d want %d for rect %+v", got, tc.want, tc.r)
		}
	}
}
