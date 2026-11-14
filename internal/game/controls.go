package game

// Controls stores the exact Flash/Windows virtual-key values from
// savedata2.data.controlarray. The original game calls Key.isDown(number), so
// keeping these numeric codes avoids keyboard-layout reinterpretation.
type Controls struct {
	Up, Left, Down, Right int `cbor:",omitempty"`
	Shoot, Grenade        int `cbor:",omitempty"`
	// Ability is the developed ABILITY key (DASH / ANCHOR). It is not part of
	// the source controlarray; configs saved before it existed get a free
	// default through migrateAbilityKeys.
	Ability int `cbor:",omitempty"`
}

// Original defaults from frame_2/DoAction.as controlarray.
func OriginalControls(playerID int) Controls {
	switch playerID {
	case 1:
		return Controls{Up: 38, Left: 37, Down: 40, Right: 39, Shoot: 219, Grenade: 221, Ability: 220}
	case 2:
		return Controls{Up: 87, Left: 65, Down: 83, Right: 68, Shoot: 84, Grenade: 89, Ability: 85}
	case 3:
		return Controls{Up: 111, Left: 103, Down: 104, Right: 105, Shoot: 106, Grenade: 109, Ability: 107}
	case 4:
		return Controls{Up: 101, Left: 97, Down: 98, Right: 99, Shoot: 96, Grenade: 110, Ability: 102}
	default:
		return Controls{}
	}
}

// Default ABILITY keys, next to each scheme's grenade key:
// P1 \ (VK 220, beside ]), P2 U (85, beside T Y), P3 numpad + (107),
// P4 numpad 6 (102, beside numpad 5 / 3).
// abilityKeyFallbacks are tried, in order, when a default is already bound
// to something else by a customised old config.
var abilityKeyFallbacks = [...]int{222, 186, 191, 71, 72, 74, 100, 34, 35, 46}

func (c Controls) keys() [7]int {
	return [7]int{c.Up, c.Left, c.Down, c.Right, c.Shoot, c.Grenade, c.Ability}
}

// migrateAbilityKeys gives every control scheme without an ability key a
// default that no player is already using. Defaults are handed out first, so
// one scheme's fallback never takes another scheme's default.
func migrateAbilityKeys(all *[4]Controls) {
	used := func(vk int) bool {
		for _, c := range all {
			for _, k := range c.keys() {
				if k == vk {
					return true
				}
			}
		}
		return false
	}
	isDefault := func(vk int) bool {
		for id := 1; id <= len(all); id++ {
			if OriginalControls(id).Ability == vk {
				return true
			}
		}
		return false
	}
	for i := range all {
		if def := OriginalControls(i + 1).Ability; all[i].Ability <= 0 && def > 0 && !used(def) {
			all[i].Ability = def
		}
	}
	for i := range all {
		if all[i].Ability > 0 {
			continue
		}
		for _, vk := range abilityKeyFallbacks {
			if !isDefault(vk) && !used(vk) {
				all[i].Ability = vk
				break
			}
		}
	}
}
