package game

import "testing"

func TestOriginalControlArrayVirtualKeys(t *testing.T) {
	want := []Controls{
		{Up: 38, Left: 37, Down: 40, Right: 39, Shoot: 219, Grenade: 221},
		{Up: 87, Left: 65, Down: 83, Right: 68, Shoot: 84, Grenade: 89},
		{Up: 111, Left: 103, Down: 104, Right: 105, Shoot: 106, Grenade: 109},
		{Up: 101, Left: 97, Down: 98, Right: 99, Shoot: 96, Grenade: 110},
	}
	for i := range want {
		if got := OriginalControls(i + 1); got != want[i] {
			t.Fatalf("P%d controls = %+v, want %+v", i+1, got, want[i])
		}
	}
}
