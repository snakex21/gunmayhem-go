package game

import "github.com/hajimehoshi/ebiten/v2"

// PowerupNameEffect ports fx_powerupname (Symbol292). It is attached at the
// pickup's world position, rises 2 px per source tick, fades in to 80%, waits
// through tick 40, then fades out by 10% per tick.
type PowerupNameEffect struct {
	X, Y  float64
	Kind  int
	Time  int
	Level int
	Alpha float64
	Fresh bool
	Dead  bool
}

// LifeBlingEffect ports lifebling() -> fx_bling (Symbol159) with mod=6. The
// clip follows the player and displays the source frame-7 green "1-UP".
type LifeBlingEffect struct {
	PlayerID int
	X, Y     float64
	Time     int
	Alpha    float64
	Scale    float64
	YOffset  float64
	Fresh    bool
	Dead     bool
}

func newPowerupNameEffect(pu Powerup) PowerupNameEffect {
	return PowerupNameEffect{X: pu.X, Y: pu.Y, Kind: pu.Type, Level: 1, Fresh: true}
}

func newLifeBlingEffect(playerID int) LifeBlingEffect {
	return LifeBlingEffect{PlayerID: playerID, Scale: 7, YOffset: 30, Fresh: true}
}

func (g *Game) updatePowerupPickupEffects() {
	if g.GameWin {
		g.powerupNameFX = g.powerupNameFX[:0]
		g.lifeBlingFX = g.lifeBlingFX[:0]
		return
	}
	for i := range g.powerupNameFX {
		e := &g.powerupNameFX[i]
		if e.Dead {
			continue
		}
		if e.Fresh {
			e.Fresh = false
			continue
		}
		e.Y -= 2
		if e.Level == 1 {
			if e.Alpha < 0.8 {
				e.Alpha += 0.1
				if e.Alpha > 0.8 {
					e.Alpha = 0.8
				}
			}
			e.Time++
			if e.Time >= 40 {
				e.Level = 2
			}
		}
		// Source has a second independent `if(level == 2)`, so tick 40 starts
		// fading immediately after switching levels.
		if e.Level == 2 {
			e.Alpha -= 0.1
		}
		if e.Alpha <= 0.01 {
			e.Dead = true
		}
	}

	for i := range g.lifeBlingFX {
		e := &g.lifeBlingFX[i]
		if e.Dead {
			continue
		}
		if e.Fresh {
			e.Fresh = false
			continue
		}
		var p *Player
		for _, candidate := range g.players {
			if candidate.ID == e.PlayerID {
				p = candidate
				break
			}
		}
		if p == nil {
			e.Dead = true
			continue
		}
		e.X = p.X
		e.Y = p.Y - e.YOffset
		e.Time++
		if e.Time <= 20 {
			if e.Alpha < 1 {
				e.Alpha += 0.5
				if e.Alpha > 1 {
					e.Alpha = 1
				}
			}
			if e.Scale > 1 {
				e.Scale += (0.9 - e.Scale) / 5
			}
		}
		if e.Time >= 40 {
			e.Alpha -= 0.1
		}
		e.YOffset += (110 - e.YOffset) / 5
		if e.Alpha <= 0.01 && e.Time >= 40 {
			e.Dead = true
		}
	}
	g.powerupNameFX = compactPowerupNameEffects(g.powerupNameFX)
	g.lifeBlingFX = compactLifeBlingEffects(g.lifeBlingFX)
}

func compactPowerupNameEffects(in []PowerupNameEffect) []PowerupNameEffect {
	out := in[:0]
	for _, e := range in {
		if !e.Dead {
			out = append(out, e)
		}
	}
	return out
}

func compactLifeBlingEffects(in []LifeBlingEffect) []LifeBlingEffect {
	out := in[:0]
	for _, e := range in {
		if !e.Dead {
			out = append(out, e)
		}
	}
	return out
}

func (g *Game) drawPowerupPickupEffects(screen *ebiten.Image) {
	for i := range g.powerupNameFX {
		e := &g.powerupNameFX[i]
		if e.Dead || e.Alpha <= 0 {
			continue
		}
		drawSourceRaster(screen, g.assets.PowerupNameFrame(e.Kind), g.worldX(e.X), g.worldY(e.Y), 1, 1, e.Alpha)
	}
	for i := range g.lifeBlingFX {
		e := &g.lifeBlingFX[i]
		if e.Dead || e.Alpha <= 0 {
			continue
		}
		drawSourceRaster(screen, g.assets.LifeBlingFrame(), g.worldX(e.X), g.worldY(e.Y), e.Scale, e.Scale, e.Alpha)
	}
}
