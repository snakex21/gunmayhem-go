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
	aiTargetSearch
)

const aiInvisibleSearchDuration = 210

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

	searching := p.AISearchTime > 0 && p.AITargetKind == aiTargetSearch
	// Preserve the last genuinely visible player position before a full
	// retarget. If that player turns invisible on exactly this 40-frame
	// boundary, clearing the target first used to lose the search origin and
	// the bot would never enter its blind-fire/search behaviour.
	previousTargetID := 0
	previousTargetX, previousTargetY := p.AITargetX, p.AITargetY
	if p.AITargetValid && p.AITargetKind == aiTargetPlayer {
		previousTargetID = p.AITargetPlayerID
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
		if !aiCanDetectPlayer(p, other) {
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
	if p.AITargetValid && p.AITargetKind == aiTargetPlayer {
		p.AISearchTime = 0
		p.AISearchPlayerID = 0
	} else if searching {
		setAISearchTarget(p)
		p.AITargetTimer = 0
		return
	} else if previousTargetID != 0 {
		// Full retarget happened on the same frame the current enemy became
		// invisible. Keep searching from the last visible coordinates instead of
		// silently forgetting the opponent (and therefore never blind-firing).
		for _, other := range g.players {
			if other.ID != previousTargetID || !other.Active || other.InvisibleTime <= 0 || aiCanDetectPlayer(p, other) {
				continue
			}
			p.AITargetX, p.AITargetY = previousTargetX, previousTargetY
			beginAISearch(p, previousTargetID)
			p.AITargetTimer = 0
			return
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
			if other.ID != p.AITargetPlayerID || !other.Active {
				continue
			}
			if aiCanDetectPlayer(p, other) {
				p.AITargetX, p.AITargetY = other.X, other.Y
				return true
			}
			if other.InvisibleTime > 0 {
				beginAISearch(p, other.ID)
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
	case aiTargetSearch:
		if p.AISearchTime > 0 {
			setAISearchTarget(p)
			return true
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
	updateAISearchTarget(p)

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
			approachMargin := 40.0
			leftLimit := groundLeft + 200
			rightLimit := groundRight - 200
			if targetPlayer != nil {
				dir := -1.0
				if tx >= groundMiddle {
					dir = 1
				}
				optimalX = tx - 150*dir
			} else if p.AITargetKind == aiTargetPowerup {
				// Powerup pickup frame is only 10x10. The old generic ±40 stop
				// distance let AI stand visibly beside MINI/INVISIBILITY forever
				// without ever overlapping it. Walk almost to the centre instead.
				approachMargin = 10
				leftLimit = groundLeft + 20
				rightLimit = groundRight - 20
			}
			if p.X > leftLimit && optimalX < p.X-approachMargin {
				p.AILeft = true
			}
			if p.X < rightLimit && optimalX > p.X+approachMargin {
				p.AIRight = true
			}
			if p.X > optimalX-approachMargin && p.X < optimalX+approachMargin {
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
			withinMargin := 30.0
			if p.AITargetKind == aiTargetPowerup {
				withinMargin = 10
			}
			within := p.X > tx-withinMargin && p.X < tx+withinMargin
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
	if p.AITargetValid && p.AITargetKind == aiTargetSearch && p.AITargetY < p.Y+35 && p.AITargetY > p.Y-105 {
		facingSearch := (p.AITargetX > p.X && p.Facing == 1) || (p.AITargetX < p.X && p.Facing == -1)
		if facingSearch {
			elapsed := aiInvisibleSearchDuration - p.AISearchTime
			// Deterministic but irregular bursts: this stays network-safe while
			// looking much less robotic than firing at a fixed interval.
			cycle := 23 + (p.ID*7+(elapsed/31)*5)%17
			if elapsed%cycle < 3 {
				p.AIShoot = true
			}
		}
	}
	applyExtraCampaignAIFireCadence(p, g)
	// Original playerAI never sets KEYNADE=true.
}

func applyExtraCampaignAIFireCadence(p *Player, g *Game) {
	if p == nil || g == nil || !p.AIShoot {
		if p != nil {
			p.AIFireCadence = 0
		}
		return
	}
	if !g.campaignMode || !g.campaignSetExtra || !p.AI {
		p.AIFireCadence = 0
		return
	}

	delay := 0
	switch {
	case g.campaignLevel == 6 && p.Weapon.Def.Number == 181:
		delay = 90
	case g.campaignLevel == 9 && p.Weapon.Def.Number == 90:
		// The airstrike boss announces roughly one strike every 2.25 seconds.
		delay = 135
	case g.campaignLevel == 11 && p.Weapon.Def.Number == 179:
		// A harpoon hit is extremely strong, so the duel is built around dodging
		// deliberate shots instead of enduring automatic one-shot reloads.
		delay = 75
	}
	if delay == 0 {
		p.AIFireCadence = 0
		return
	}
	p.AIFireCadence++
	// These single-shot campaign weapons get a visible manual-reload-like pause
	// instead of instantly re-equipping their default weapon every source ROF.
	p.AIShoot = (p.AIFireCadence-1)%delay == 0
}

func beginAISearch(p *Player, lostPlayerID int) {
	if p == nil {
		return
	}
	// AITargetX/Y still contain the last genuinely visible coordinates. Do not
	// read the invisible player's new coordinates here.
	p.AISearchX = p.AITargetX
	p.AISearchY = p.AITargetY
	p.AISearchTime = aiInvisibleSearchDuration
	p.AISearchPlayerID = lostPlayerID
	setAISearchTarget(p)
}

func setAISearchTarget(p *Player) {
	if p == nil {
		return
	}
	p.AITargetValid = p.AISearchTime > 0
	if !p.AITargetValid {
		return
	}
	p.AITargetKind = aiTargetSearch
	p.AITargetPlayer = false
	p.AITargetPlayerID = 0
	p.AITargetSerial = 0
	p.AITargetX = p.AISearchX
	p.AITargetY = p.AISearchY
}

func updateAISearchTarget(p *Player) {
	if p == nil || p.AITargetKind != aiTargetSearch || p.AISearchTime <= 0 {
		return
	}
	p.AISearchTime--
	if p.AISearchTime <= 0 {
		p.AITargetValid = false
		p.AITargetKind = aiTargetNone
		p.AISearchPlayerID = 0
		return
	}

	elapsed := aiInvisibleSearchDuration - p.AISearchTime
	step := elapsed / 28
	seed := uint32(step*1103515245 + p.ID*12345 + p.AISearchPlayerID*265443576)
	spread := 55.0 + float64(step)*9
	if spread > 145 {
		spread = 145
	}
	xUnit := float64(int(seed%2001)-1000) / 1000
	yUnit := float64(int((seed/2001)%2001)-1000) / 1000
	p.AITargetX = p.AISearchX + xUnit*spread
	p.AITargetY = p.AISearchY + yUnit*42
}

func sourceAITargetPlayer(p *Player, g *Game) *Player {
	if !p.AITargetValid || p.AITargetKind != aiTargetPlayer {
		return nil
	}
	for _, other := range g.players {
		if other.ID == p.AITargetPlayerID && other.Active && aiCanDetectPlayer(p, other) {
			return other
		}
	}
	return nil
}

// Invisibility is camouflage, not invulnerability: AI loses distant targets
// and only reacquires an invisible player at close range. Human opponents still
// use their own eyes, while the network renderer handles the local-only ghost.
func aiCanDetectPlayer(observer, target *Player) bool {
	if observer == nil || target == nil || target.InvisibleTime <= 0 {
		return true
	}
	return math.Hypot(target.X-observer.X, target.Y-observer.Y) <= 135
}
