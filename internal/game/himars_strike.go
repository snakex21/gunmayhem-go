package game

import "math"

// HIMARS Strike (weapon 90) targeting.
//
// The strike is called on the nearest enemy ahead of the shooter (60-700 px
// in the facing direction, at most 260 px above or below), at that enemy's
// position when the trigger is pulled; there is no lead, so a moving target
// can walk out of the direct hit. Without a target ahead the old fixed drop
// point (260 px ahead, or the AI's own aim point) is kept. The strike falls
// slowly enough to dodge, and a landing marker (a HazardMarker, synced to
// online clients with the snapshot) shows where it will hit.
const (
	himarsStrikeFallSpeed   = 14.0 // initial fall speed (was 24)
	himarsStrikeGravity     = 0.8  // per tick, see updateDevelopedProjectile
	himarsStrikeMinRange    = 60.0
	himarsStrikeMaxRange    = 700.0
	himarsStrikeMaxDY       = 260.0
	himarsMarkerLingerTicks = 10
	hazardMarkerStrike      = 2
)

// himarsStrikeTargetX returns the X of the nearest enemy ahead of shooter
// within the strike window, or ok=false when nobody qualifies. Invisible
// targets are only found at close range, like the AI search does.
func himarsStrikeTargetX(shooter *Player, candidates []*Player) (x float64, ok bool) {
	if shooter == nil {
		return 0, false
	}
	facing := float64(shooter.Facing)
	if facing == 0 {
		facing = 1
	}
	best := math.MaxFloat64
	for _, other := range candidates {
		if other == nil || other == shooter || !other.Active {
			continue
		}
		if other.scoreOwnerID() == shooter.scoreOwnerID() {
			continue
		}
		if shooter.Team != 0 && other.Team == shooter.Team {
			continue
		}
		if !aiCanDetectPlayer(shooter, other) {
			continue
		}
		ahead := (other.X - shooter.X) * facing
		if ahead < himarsStrikeMinRange || ahead > himarsStrikeMaxRange || math.Abs(other.Y-shooter.Y) > himarsStrikeMaxDY {
			continue
		}
		if ahead < best {
			best, x, ok = ahead, other.X, true
		}
	}
	return x, ok
}

// himarsStrikeFallTicks estimates how many ticks a strike dropped at x needs
// to reach the first platform top (or landingY) below it.
func himarsStrikeFallTicks(y, vy, landingY float64) int {
	for tick := 1; tick <= 120; tick++ {
		vy += himarsStrikeGravity
		y += vy
		if y >= landingY {
			return tick
		}
	}
	return 120
}

// aimHimarsStrikes retargets the strikes p fired this tick and places their
// landing markers.
func (g *Game) aimHimarsStrikes(p *Player, fired []Bullet) {
	if g == nil || p == nil {
		return
	}
	var candidates []*Player
	for i := range fired {
		b := &fired[i]
		if b.DevelopedProjectile != DevelopedProjectileStrike || b.Dead {
			continue
		}
		if candidates == nil {
			candidates = g.hitTargets()
		}
		if x, ok := himarsStrikeTargetX(p, candidates); ok {
			b.X = x
		}
		landingY := hazardLandingY(g.customRuntimeArena(), b.X)
		fall := himarsStrikeFallTicks(b.Y, b.VY, landingY)
		life := fall + himarsMarkerLingerTicks
		g.extraHazardMarkers = append(g.extraHazardMarkers, HazardMarker{
			Kind: hazardMarkerStrike, X: b.X, Y: landingY,
			Life: life, MaxLife: life, Drop: himarsMarkerLingerTicks + minInt(8, fall),
		})
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
