package game

import "testing"

func TestOptionsSourceToggleHitboxes(t *testing.T) {
	tests := []struct {
		x, y float64
		want int
	}{
		{102, 527, optionHitMusicOn},
		{165, 527, optionHitMusicOff},
		{281, 527, optionHitSoundOn},
		{344, 527, optionHitSoundOff},
		{459, 527, optionHitQualityLow},
		{522, 527, optionHitQualityMedium},
		{585, 528, optionHitQualityHigh},
		{780, 538, optionHitBack},
	}
	for _, tc := range tests {
		if got := optionsHitAt(tc.x, tc.y); got != tc.want {
			t.Fatalf("options hit %.0f,%.0f=%d want%d", tc.x, tc.y, got, tc.want)
		}
	}
}
