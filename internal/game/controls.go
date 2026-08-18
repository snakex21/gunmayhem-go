package game

// Controls stores the exact Flash/Windows virtual-key values from
// savedata2.data.controlarray. The original game calls Key.isDown(number), so
// keeping these numeric codes avoids keyboard-layout reinterpretation.
type Controls struct {
	Up, Left, Down, Right int
	Shoot, Grenade        int
}

// Original defaults from frame_2/DoAction.as controlarray.
func OriginalControls(playerID int) Controls {
	switch playerID {
	case 1:
		return Controls{Up: 38, Left: 37, Down: 40, Right: 39, Shoot: 219, Grenade: 221}
	case 2:
		return Controls{Up: 87, Left: 65, Down: 83, Right: 68, Shoot: 84, Grenade: 89}
	case 3:
		return Controls{Up: 111, Left: 103, Down: 104, Right: 105, Shoot: 106, Grenade: 109}
	case 4:
		return Controls{Up: 101, Left: 97, Down: 98, Right: 99, Shoot: 96, Grenade: 110}
	default:
		return Controls{}
	}
}
