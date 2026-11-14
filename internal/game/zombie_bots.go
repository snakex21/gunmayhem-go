package game

import "math"

// zombie_bots.go: AI teammates walk to a downed teammate and revive them
// when it is safe enough. The normal bot AI keeps aiming and shooting; only
// the movement keys are taken over, along the zombie nav graph (navSteer).

const (
	zombieBotReviveMinHP   = 30    // a bot below this HP does not go
	zombieBotDangerRadius  = 150.0 // zombies this close to the bot or the marker count as danger
	zombieBotDangerMax     = 3     // this many: too dangerous to start
	zombieBotDangerAbort   = 5     // this many: give up an ongoing revive
	zombieBotReviveSlackX  = 16.0  // stand this far inside the revive box
	zombieBotReviveMaxDist = 1400.0
	zombieBotStallFrames   = 240 // no progress this long: give up on that marker ...
	zombieBotGiveUpFrames  = 450 // ... for this long
)

// zombieBotAssist runs after a bot's AI decided its keys this frame.
func (g *Game) zombieBotAssist(p *Player) {
	if !g.zombieSimulating() || p == nil || !p.AI || p.IsDouble || !p.Active {
		return
	}
	m := g.zombie
	zp, hp := m.playerState(p.ID), m.hostPlayer(p.ID)
	if zp == nil || hp == nil {
		return
	}
	if hp.botGiveUp > 0 {
		hp.botGiveUp--
		if hp.botGiveUp == 0 {
			hp.botGiveUpID = 0
		}
	}
	if zp.State != zombiePlayerAlive || zp.HP < zombieBotReviveMinHP {
		hp.botReviveID = 0
		return
	}
	target := g.zombieBotReviveTarget(p, hp)
	if target == nil {
		hp.botReviveID = 0
		return
	}
	if hp.botReviveID != target.ID {
		hp.botReviveID = target.ID
		hp.botNav = zombieNavState{cur: -1, lastPlat: -1}
		hp.botBestDist = math.Inf(1)
		hp.botStall = 0
	}
	p.AILeft, p.AIRight, p.AIUp, p.AIDown = false, false, false, false
	dx := target.DownX - p.X
	onMarker := p.JumpNum == 2 && math.Abs(dx) <= zombieReviveRadiusX-zombieBotReviveSlackX && math.Abs(target.DownY-p.Y) <= zombieReviveRadiusY-10
	// The nav graph is built for zombie physics: a bot that stops getting
	// closer gives up on that marker for a while instead of looping.
	if d := math.Abs(dx) + math.Abs(target.DownY-p.Y); onMarker || d < hp.botBestDist-4 {
		hp.botBestDist = math.Min(hp.botBestDist, d)
		hp.botStall = 0
	} else if hp.botStall++; hp.botStall >= zombieBotStallFrames {
		hp.botGiveUpID, hp.botGiveUp = target.ID, zombieBotGiveUpFrames
		hp.botReviveID = 0
		return
	}
	if onMarker {
		// On the marker: stay (keep shooting whatever the AI aims at).
		hp.botNav.upPrev = false
		return
	}
	nav := m.navFor(zp.Zone, g.arena, zombieNavNormal)
	navSteer(p, &hp.botNav, int(g.mapFXFrame)+p.ID*7, target.DownX, target.DownY, nav)
}

// zombieBotReviveTarget: the nearest downed (not dead, no self-revive)
// teammate in the bot's zone, if the area is safe enough.
func (g *Game) zombieBotReviveTarget(p *Player, hp *zombieHostPlayer) *ZombiePlayerState {
	m := g.zombie
	zone := m.playerZone(p)
	var best *ZombiePlayerState
	bestD := math.Inf(1)
	for i := range m.hud.Players {
		o := &m.hud.Players[i]
		if o.ID == 0 || o.ID == p.ID || o.State != zombiePlayerDown || o.SelfRevives > 0 || o.Zone != zone || o.ID == hp.botGiveUpID {
			continue
		}
		d := math.Abs(o.DownX-p.X) + math.Abs(o.DownY-p.Y)
		if d < bestD && d < zombieBotReviveMaxDist {
			best, bestD = o, d
		}
	}
	if best == nil {
		return nil
	}
	limit := zombieBotDangerMax
	if hp.botReviveID == best.ID {
		limit = zombieBotDangerAbort // already going: only a crowd stops it
	}
	if g.zombieDangerNear(zone, p.X, p.Y) >= limit || g.zombieDangerNear(zone, best.DownX, best.DownY) >= limit {
		return nil
	}
	return best
}

// zombieDangerNear counts living zombies of a zone near a point.
func (g *Game) zombieDangerNear(zone int, x, y float64) int {
	n := 0
	for _, z := range g.zombie.zombies {
		if z.Dead || z.Zone != zone {
			continue
		}
		if math.Hypot(z.X-x, z.Y-y) <= zombieBotDangerRadius {
			n++
		}
	}
	return n
}
