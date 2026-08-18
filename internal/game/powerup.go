package game

import (
	"math"
	"math/rand"
)

type Powerup struct {
	Serial     int
	X, Y       float64
	Type       int
	PickedByID int

	// Source playheads. Symbol 734 is explicitly play()'d by frame_10 after
	// attachment. flashystuff is the nested Symbol 710 and advances on its own
	// until its frame-72 stop().
	Frame        int
	FlashFrame   int
	FlashPlaying bool
	PickedUp     bool
	Dead         bool
}

// spawnPowerup ports frame_10/DoAction.as literally for normal arena mode:
// sample an integer point in ground.platform's bounds, retry until hitTest is
// true, walk upward in 2 px steps until leaving collision, then spawn 40 px up.
func spawnPowerup(arena Map, doubleActive bool) Powerup {
	minX, minY, maxX, maxY, ok := platformBounds(arena.Platforms)
	if !ok {
		return Powerup{Dead: true}
	}

	width := int(math.Floor(maxX - minX - 40))
	height := int(math.Floor(maxY - minY))
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	var x, y float64
	for {
		x = minX + 20 + float64(rand.Intn(width))
		y = minY + float64(rand.Intn(height))
		if pointHitsPlatforms(arena.Platforms, x, y) {
			break
		}
	}
	for i := 1; ; i++ {
		if !pointHitsPlatforms(arena.Platforms, x, y-float64(i*2)) {
			y -= float64(i * 2)
			break
		}
	}

	kind := rand.Intn(7)
	// DefineSprite_734: Double is replaced by shield when _root.double exists.
	// campaignmode is not active in the current normal-arena state.
	if kind == 4 && doubleActive {
		kind = 2
	}
	return Powerup{
		X: x, Y: y - 40, Type: kind,
		Frame: 0, FlashFrame: 0, FlashPlaying: true,
	}
}

func updatePowerups(powerups []Powerup, players []*Player) {
	for i := range powerups {
		pu := &powerups[i]
		if pu.Dead {
			continue
		}

		advancePowerupPlayheads(pu)

		// ActionScript tests `_currentframe >= 20` (Flash is one-based).
		if pu.Frame >= 19 && !pu.PickedUp {
			pickup := powerupPickupRect(pu.X, pu.Y)
			for _, p := range players {
				if !p.Active || !rectsOverlap(pickup, p.Hitbox()) {
					continue
				}

				// Source sets pickedup before checking for the helper named double.
				pu.PickedUp = true
				if p.IsDouble {
					// `continue;` in the original for-loop: another overlapping real
					// player may still receive the effect on this same frame.
					continue
				}
				// Source powerup pickup increments pgsdata[player-1][6].
				p.PowerupsCollected++
				pu.PickedByID = p.ID
				applyPowerup(p, pu.Type)
				break
			}
		}

		// DefineSprite_734 removes itself only when the nested flashystuff has
		// reached its last (72nd) frame. No invented immediate deletion.
		if pu.PickedUp && pu.FlashFrame == 71 {
			pu.Dead = true
		}
	}
}

func advancePowerupPlayheads(pu *Powerup) {
	// Outer Symbol 734: frame 98 executes gotoAndPlay(20), therefore the
	// visible loop is source frames 20..97 after the introduction.
	pu.Frame++
	if pu.Frame >= 97 {
		pu.Frame = 19
	}

	// Nested Symbol 710 stops on source frame 72.
	if pu.FlashPlaying {
		if pu.FlashFrame < 71 {
			pu.FlashFrame++
		}
		if pu.FlashFrame >= 71 {
			pu.FlashFrame = 71
			pu.FlashPlaying = false
		}
	}
}

func powerupPickupRect(x, y float64) Rect {
	// Symbol 699 is the named `frame` instance used by hitTest(frame): its
	// source shape is exactly 10x10 px centered on the parent origin.
	return Rect{X: x - 5, Y: y - 5, W: 10, H: 10}
}

func applyPowerup(p *Player, kind int) {
	switch kind {
	case 0:
		p.Lives++
	case 1:
		p.InvisibleTime = 200
	case 2:
		p.ShieldTime = 300
	case 3:
		p.JetFuel = 100
	case 4:
		p.WantsDouble = true
	case 5:
		p.SpeedTime = 300
	case 6:
		p.MiniTime = 400
	}
}

func powerupName(kind int) string {
	switch kind {
	case 0:
		return "EXTRA LIFE"
	case 1:
		return "INVISIBILITY"
	case 2:
		return "SHIELD"
	case 3:
		return "JETPACK"
	case 4:
		return "DOUBLE"
	case 5:
		return "SPEED"
	case 6:
		return "MINI"
	default:
		return "POWERUP"
	}
}

func compactPowerups(in []Powerup) []Powerup {
	out := in[:0]
	for _, pu := range in {
		if !pu.Dead {
			out = append(out, pu)
		}
	}
	return out
}

func platformBounds(platforms []Rect) (minX, minY, maxX, maxY float64, ok bool) {
	if len(platforms) == 0 {
		return 0, 0, 0, 0, false
	}
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, r := range platforms {
		minX = math.Min(minX, r.X)
		minY = math.Min(minY, r.Y)
		maxX = math.Max(maxX, r.X+r.W)
		maxY = math.Max(maxY, r.Y+r.H)
	}
	return minX, minY, maxX, maxY, true
}

func pointHitsPlatforms(platforms []Rect, x, y float64) bool {
	for _, r := range platforms {
		if r.Contains(x, y) {
			return true
		}
	}
	return false
}
