package game

import "math/rand"

const (
	sourceAISpecialNone          = iota
	sourceAISpecialDoubleSpawner // playerAI_double, campaign level 4
	sourceAISpecialJetpack       // playerAI3, campaign level 7
	sourceAISpecialInvisible     // playerAI4, campaign level 9
	sourceAISpecialBoss          // playerAIboss, campaign level 10
)

func campaignAISpecialForLevel(level int) int {
	switch level {
	case 4:
		return sourceAISpecialDoubleSpawner
	case 7:
		return sourceAISpecialJetpack
	case 9:
		return sourceAISpecialInvisible
	case 10:
		return sourceAISpecialBoss
	default:
		return sourceAISpecialNone
	}
}

// prepareCampaignAISpecial mirrors the class-specific blocks that run at the
// start of playerAI_double/playerAI3/playerAI4/playerAIboss.onEnterFrame.
func (g *Game) prepareCampaignAISpecial(p *Player) {
	if !g.campaignMode || !p.AI || !p.Active {
		return
	}

	// The helper spawned by playerAI_double is a regular playerAI movie clip.
	// In campaign level 4 that class has its own campaign block: it flickers at
	// 20..49% alpha and SELFDESTRUCTs as soon as gothitby is no longer "none".
	// Keeping this separate from Pylon Man's playerAI_double class is important:
	// the two actors are independent AI instances, not a mirrored pair.
	if g.campaignLevel == 4 && p.IsDouble && p.PersistentDouble {
		if p.LastHitBy != 0 {
			p.Kill(g.arena)
			return
		}
		p.Alpha = float64(20+rand.Intn(30)) / 100
	}

	switch p.AISpecial {
	case sourceAISpecialDoubleSpawner:
		// playerAI_double starts at -80. While there is no global `double`, it
		// increments every tick; at 35 it spawns a helper and resets to -120.
		if g.campaignLevel == 4 && !g.hasActiveDouble() {
			p.AIFakeDoubleTime++
			if p.AIFakeDoubleTime >= 35 {
				p.AIFakeDoubleTime = -120
				p.WantsDouble = true
				// spawnfriend() runs before playerAI_double applies vx/vy this frame.
				// Preserve that exact pre-move position for the helper created later
				// in the Go update, otherwise both actors start at the owner's
				// post-move position and are much more likely to remain in lockstep.
				p.DoubleSpawnPositionSet = true
				p.DoubleSpawnX = p.X
				p.DoubleSpawnY = p.Y
			}
		}
	case sourceAISpecialInvisible:
		// playerAI4: invisible while gothitby == "none"; being hit reveals it
		// until the normal landing code clears LastHitBy again.
		if p.LastHitBy == 0 {
			p.InvisibleTime = 10
		} else {
			p.InvisibleTime = 0
		}
	}
}

// updateCampaignDynamiteRain is the root frame10 campaignlevel==5 block:
// every 80 unpaused ticks a neutral grenade is attached at y=-200 over the
// current map's crate area with asdf=-1000 (firepower/owner both resolve to 0).
func (g *Game) updateCampaignDynamiteRain() {
	if !g.campaignMode || g.campaignLevel != 5 || g.paused {
		return
	}
	g.campaignDynamiteTime++
	if g.campaignDynamiteTime < 80 {
		return
	}
	g.campaignDynamiteTime = 0
	width := int(g.arena.CrateMaxX - g.arena.CrateMinX)
	x := g.arena.CrateMinX
	if width > 0 {
		x += float64(rand.Intn(width))
	}
	rot := rand.Float64()*10 + 5
	if rand.Intn(2) == 0 {
		rot = -rot
	}
	g.grenades = append(g.grenades, Grenade{X: x, Y: -200, Rotation: 0, RotSpeed: rot, OwnerID: 0})
}

func (g *Game) killCampaignDouble() {
	if !g.campaignMode || g.campaignLevel != 4 {
		return
	}
	for _, p := range g.players {
		if !p.IsDouble || !p.PersistentDouble || !p.Active {
			continue
		}
		p.Lives = 1
		p.Kill(g.arena)
	}
}
