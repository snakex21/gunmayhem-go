package game

import "testing"

func TestTextRepeatDurationTriggers(t *testing.T) {
	tests := []struct {
		duration int
		want     bool
	}{
		{0, false},
		{1, true},
		{2, false},
		{13, false},
		{14, true},
		{15, false},
		{16, true},
		{17, false},
		{18, true},
		{23, false},
		{24, true},
		{25, false},
		{26, true},
		{27, false},
		{30, true},
		{31, false},
	}
	for _, tc := range tests {
		if got := textRepeatDurationTriggers(tc.duration); got != tc.want {
			t.Fatalf("duration %d: got %v want %v", tc.duration, got, tc.want)
		}
	}
}
