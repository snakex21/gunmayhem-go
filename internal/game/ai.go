package game

import (
	"math"
	"sort"
)

const (
	aiTargetNone = iota
	aiTargetPlayer
	aiTargetCrate
	aiTargetPowerup
)

type aiPickupRef struct {
	Serial int
	Kind   int
	X, Y   float64
	Type   int // powerupnumber for powerups; undefined/not used for crates
}

// prepareAITarget is the part of DefineSprite_701_playerAI.onEnterFrame that
// runs before `_X = _X + vx; _Y = _Y + vy;`.
func prepareAITarget(p *Player, g *Game) {
	if !p.AI || !p.Active {
		return
	}
	p.AITargetTimer++
	if p.AITargetTimer < 40 {
		refreshSourceAITarget(p, g)
		return
	}

	low := 5000.0
	p.AITargetValid = false
	p.AITargetKind = aiTargetNone
	p.AITargetSerial = 0
	p.AITargetPlayerID = 0
	p.AITargetPlayer = false

	// Original activeplayers order, with PLAYERNUMBER rather than movie-clip
	// identity. A Double has the owner's PLAYERNUMBER and therefore is skipped
	// by its owner/other helper with that same player number.
	for _, other := range g.players {
		if !other.Active || other.Team == p.Team || other.scoreOwnerID() == p.scoreOwnerID() {
			continue
		}
		distance := math.Round(math.Sqrt(
			math.Pow(other.X-p.X, 2) + math.Pow(other.Y-30-p.Y, 2),
		))
		if distance < low {
			low = distance
			p.AITargetValid = true
			p.AITargetKind = aiTargetPlayer
			p.AITargetPlayer = true
			p.AITargetPlayerID = other.ID
			p.AITargetX = other.X
			p.AITargetY = other.Y
		}
	}

	// _root.cratearray contains crates and powerups in their attachment order.
	for _, pickup := range sourceCrateArray(g) {
		distance := math.Round(math.Sqrt(
			math.Pow(pickup.X-p.X, 2) + math.Pow(pickup.Y-30-p.Y, 2),
		))
		if distance < low {
			low = distance
			setAIPickupTarget(p, pickup)
		}

		// Exact AS2 precedence:
		// (this._name != "double" && powerupnumber == 0) || powerupnumber == 3
		if pickup.Kind == aiTargetPowerup && ((!p.IsDouble && pickup.Type == 0) || pickup.Type == 3) {
			setAIPickupTarget(p, pickup)
			break
		}
	}
	p.AITargetTimer = 0
}

func setAIPickupTarget(p *Player, pickup aiPickupRef) {
	p.AITargetValid = true
	p.AITargetKind = pickup.Kind
	p.AITargetSerial = pickup.Serial
	p.AITargetPlayer = false
	p.AITargetPlayerID = 0
	p.AITargetX = pickup.X
	p.AITargetY = pickup.Y
}

func sourceCrateArray(g *Game) []aiPickupRef {
	out := make([]aiPickupRef, 0, len(g.crates)+len(g.powerups))
	for i := range g.crates {
		c := &g.crates[i]
		if c.Dead {
			continue
		}
		out = append(out, aiPickupRef{Serial: c.Serial, Kind: aiTargetCrate, X: c.X, Y: c.Y})
	}
	for i := range g.powerups {
		pu := &g.powerups[i]
		if pu.Dead {
			continue
		}
		out = append(out, aiPickupRef{Serial: pu.Serial, Kind: aiTargetPowerup, X: pu.X, Y: pu.Y, Type: pu.Type})
	}
	sort.SliceStable(out, func(i, j int) bool {
		// Zero is only possible in direct unit-test fixtures. Preserve insertion
		// stability there instead of pretending a source attachment order.
		if out[i].Serial == 0 || out[j].Serial == 0 {
			return false
		}
		return out[i].Serial < out[j].Serial
	})
	return out
}

func refreshSourceAITarget(p *Player, g *Game) bool {
	if !p.AITargetValid {
		return false
	}
	switch p.AITargetKind {
	case aiTargetPlayer:
		for _, other := range g.players {
			if other.ID == p.AITargetPlayerID && other.Active {
				p.AITargetX, p.AITargetY = other.X, other.Y
				return true
			}
		}
	case aiTargetCrate:
		for i := range g.crates {
			c := &g.crates[i]
			if !c.Dead && c.Serial == p.AITargetSerial {
				p.AITargetX, p.AITargetY = c.X, c.Y
				return true
			}
		}
	case aiTargetPowerup:
		for i := range g.powerups {
			pu := &g.powerups[i]
			if !pu.Dead && pu.Serial == p.AITargetSerial {
				p.AITargetX, p.AITargetY = pu.X, pu.Y
				return true
			}
		}
	}
	p.AITargetValid = false
	return false
}

// decideAIController is the source block after player movement/collision and
// before the KEY* input block in DefineSprite_701_playerAI.
func decideAIController(p *Player, g *Game) {
	p.clearAIInput()
	if !p.AI || !p.Active {
		return
	}
	refreshSourceAITarget(p, g)

	groundLeft, _, groundRight, _, ok := platformBounds(g.arena.Platforms)
	if !ok {
		groundLeft, groundRight = 0, ScreenWidth
	}
	groundMiddle := (groundLeft + groundRight) / 2

	// Symbol701 source points: land1=(-80,20), land2=(80,20), noland=(0,30),
	// each sampled through the player's 80% scale.
	land1x := p.X - 64
	land1y := p.Y + 16
	land2x := p.X + 64
	land2y := p.Y + 16
	nolandY := p.Y + 24

	if p.AIPrevX == math.Round(p.X) {
		p.AIIdleTime2++
		if p.AIIdleTime2 >= 4 {
			p.AIIdleTime2 = 0
			if p.JumpNum == 2 {
				p.AIUp = true
			}
			if p.AITargetValid {
				if p.AITargetX < p.X {
					p.AILockLeft = 10
				}
				if p.AITargetX >= p.X {
					p.AILockRight = 10
				}
			}
		}
	} else {
		p.AIIdleTime2 = 0
	}
	p.AIPrevX = math.Round(p.X)

	targetPlayer := sourceAITargetPlayer(p, g)
	if p.AITargetValid {
		tx, ty := p.AITargetX, p.AITargetY
		if ty <= p.Y+10 && ty >= p.Y-80 && p.JumpNum == 2 {
			optimalX := tx
			if targetPlayer != nil {
				dir := -1.0
				if tx >= groundMiddle {
					dir = 1
				}
				optimalX = tx - 150*dir
			}
			if p.X > groundLeft+200 && optimalX < p.X-40 {
				p.AILeft = true
			}
			if p.X < groundRight-200 && optimalX > p.X+40 {
				p.AIRight = true
			}
			if p.X > optimalX-40 && p.X < optimalX+40 {
				if tx > p.X && p.Facing == -1 {
					p.AIRight = true
				}
				if tx <= p.X && p.Facing == 1 {
					p.AILeft = true
				}
			}
		} else if ty > p.Y+10 {
			if ((targetPlayer != nil && p.X > groundLeft+50 && targetPlayer.JumpNum == 2) ||
				(targetPlayer == nil && p.X > groundLeft-10)) && tx < p.X-30 {
				p.AILeft = true
			}
			if ((targetPlayer != nil && p.X < groundRight-50 && targetPlayer.JumpNum == 2) ||
				(targetPlayer == nil && p.X < groundRight+10)) && tx > p.X+30 {
				p.AIRight = true
			}
			within := p.X > tx-30 && p.X < tx+30
			if (targetPlayer != nil && within && math.Round(targetPlayer.VY) == 1 && targetPlayer.JumpNum == 2) ||
				(targetPlayer == nil && within) {
				p.AIDown = true
			}
			if targetPlayer != nil && pointHitsPlatforms(g.arena.Platforms, p.X, p.Y+95) && targetPlayer.JumpNum == 2 {
				p.AIDown = true
			}
		} else if ty < p.Y-80 {
			if p.VY < 0 && p.JumpNum == 1 && pointHitsPlatforms(g.arena.Platforms, p.X, p.Y-30) && math.Abs(p.VX) <= 5 {
				p.AIUp = true
			}
			if p.JumpNum == 2 && ((pointHitsPlatforms(g.arena.Platforms, p.X, p.Y-75) && math.Abs(p.VX) <= 5) ||
				(pointHitsPlatforms(g.arena.Platforms, p.X, p.Y-120) && math.Abs(p.VX) <= 5)) {
				p.AIUp = true
			} else if p.JumpNum == 2 && p.AILockLeft == 0 && p.VX >= -5 &&
				(pointHitsPlatforms(g.arena.Platforms, p.X-100, p.Y-80) || pointHitsPlatforms(g.arena.Platforms, p.X-100, p.Y-120)) {
				p.AIUp = true
				p.AILockLeft = 20
			} else if p.JumpNum == 2 && p.AILockRight == 0 && p.VX <= 5 &&
				(pointHitsPlatforms(g.arena.Platforms, p.X+100, p.Y-80) || pointHitsPlatforms(g.arena.Platforms, p.X+100, p.Y-120)) {
				p.AIUp = true
				p.AILockRight = 20
			} else if p.JumpNum == 2 {
				if tx < p.X && p.AILockRight == 0 {
					p.AILockLeft = 10
				} else if tx > p.X && p.AILockLeft == 0 {
					p.AILockRight = 10
				}
			}
		}
	}

	if p.X < groundLeft+100 || p.X > groundRight-100 {
		if groundMiddle < p.X-40 {
			p.AILeft = true
		}
		if groundMiddle > p.X+40 {
			p.AIRight = true
		}
	}
	if math.Abs(p.VX) > 30 && p.JumpNum == 2 {
		p.AIUp = true
	}
	if (p.VX > 15 || p.VX < -15) && p.JumpNum == 2 {
		p.AIUp = true
	}
	if !pointHitsPlatforms(g.arena.Platforms, land2x, land2y) && p.VX > 10 && p.JumpNum == 2 {
		p.AIUp = true
	}
	if !pointHitsPlatforms(g.arena.Platforms, land1x, land1y) && p.VX < -10 && p.JumpNum == 2 {
		p.AIUp = true
	}

	targetPlayerMode := targetPlayer != nil
	if (p.X > groundRight-100 && targetPlayerMode) || (p.X > groundRight && !targetPlayerMode) {
		if p.JetFuel > 0 && p.X > groundRight && p.VY > -1 {
			p.AILockUp = 10
		}
		if p.JumpNum == 2 {
			p.AIUp = true
		}
		if p.VY > 0 && p.VX < -1 && (p.JumpNum == 1 || p.JumpNum == 11) && p.X < groundRight+100 {
			p.AIUp = true
		}
		if p.X > groundRight && !pointHitsPlatforms(g.arena.Platforms, p.X, nolandY) &&
			(p.JumpNum == 1 || p.JumpNum == 11) && p.Y > g.arena.LowestY-100 {
			p.AIUp = true
		}
	}
	if (p.X < groundLeft+100 && targetPlayerMode) || (p.X < groundLeft && !targetPlayerMode) {
		if p.JetFuel > 0 && p.X < groundLeft && p.VY > -1 {
			p.AILockUp = 10
		}
		if p.JumpNum == 2 {
			p.AIUp = true
		}
		if p.VY > 0 && p.VX > 1 && (p.JumpNum == 1 || p.JumpNum == 11) && p.X > groundLeft-100 {
			p.AIUp = true
		}
		if p.X < groundLeft && !pointHitsPlatforms(g.arena.Platforms, p.X, nolandY) &&
			(p.JumpNum == 1 || p.JumpNum == 11) && p.Y > g.arena.LowestY-100 {
			p.AIUp = true
		}
	}

	if !pointHitsPlatforms(g.arena.Platforms, land2x, land2y) && p.JumpNum == 2 {
		p.AIRight = false
	}
	if !pointHitsPlatforms(g.arena.Platforms, land1x, land1y) && p.JumpNum == 2 {
		p.AILeft = false
	}
	if !pointHitsPlatforms(g.arena.Platforms, land2x-15, land2y) && p.JumpNum == 2 {
		p.AIRight = false
		p.AILeft = true
	}
	if !pointHitsPlatforms(g.arena.Platforms, land1x+15, land1y) && p.JumpNum == 2 {
		p.AILeft = false
		p.AIRight = true
	}

	if p.AILockRight >= p.AILockLeft {
		p.AILockLeft = 0
	}
	if p.AILockLeft > p.AILockRight {
		p.AILockRight = 0
	}
	if p.AILockRight > 0 {
		p.AIRight = true
		p.AILockRight--
	}
	if p.AILockLeft > 0 {
		p.AILeft = true
		p.AILockLeft--
	}
	if p.AILockUp > 0 {
		p.AIUp = true
		p.AILockUp--
	}

	if targetPlayer != nil && targetPlayer.Y < p.Y+20 && targetPlayer.Y > p.Y-80 {
		facingTarget := (targetPlayer.X > p.X && p.Facing == 1) || (targetPlayer.X < p.X && p.Facing == -1)
		if facingTarget {
			if p.Weapon.Def.Shotgun > 0 {
				d := math.Abs(targetPlayer.X - p.X)
				if d > 15 && d < 150 {
					p.AIShoot = true
				}
			} else {
				p.AIShoot = true
			}
		}
	}
	// Original playerAI never sets KEYNADE=true.
}

func sourceAITargetPlayer(p *Player, g *Game) *Player {
	if !p.AITargetValid || p.AITargetKind != aiTargetPlayer {
		return nil
	}
	for _, other := range g.players {
		if other.ID == p.AITargetPlayerID && other.Active {
			return other
		}
	}
	return nil
}
