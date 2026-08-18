package game

import (
	"math"
	"math/rand"
)

type Grenade struct {
	X, Y      float64
	VX, VY    float64
	Rotation  float64
	RotSpeed  float64
	Fuse      int
	OwnerID   int
	FreePass  bool
	HitGround bool
	Dead      bool
}

type Explosion struct {
	X, Y float64
	Life int
}

type DynamiteEffect struct {
	X, Y     float64
	VX, VY   float64
	Rotation float64
	RotSpeed float64
	Scale    float64
	Alpha    float64
	Fresh    bool
	Dead     bool
}

func updateGrenadeInput(p *Player, grenades *[]Grenade) {
	if !p.Active || p.Grenades <= 0 {
		return
	}
	// Source wraps offhand input in gamemode != 2, while Gun Game constructor
	// sets offhandnumber=-1. Both modes therefore have no grenade input.
	if p.GameMode == SourceGameModeInstagib || p.GameMode == SourceGameModeGunGame {
		return
	}

	pressed := p.grenadePressed()
	if pressed && !p.GrenadeHeld {
		p.GrenadeHeld = true
		p.GrenadePower = 1
	} else if !pressed && p.GrenadeHeld {
		p.GrenadeHeld = false
		rot := rand.Float64()*10 + 5
		if rand.Intn(2) == 0 {
			rot = -rot
		}
		*grenades = append(*grenades, Grenade{
			X:        p.X + 30*float64(p.Facing),
			Y:        p.Y - 35,
			VX:       float64(p.Facing) * p.GrenadePower,
			VY:       -p.GrenadePower,
			Rotation: float64(p.Facing) * 50,
			RotSpeed: rot,
			OwnerID:  p.scoreOwnerID(),
		})
		p.Grenades--
	}

	if pressed && p.GrenadePower < 14 {
		p.GrenadePower += 0.5
		if p.GrenadePower > 14 {
			p.GrenadePower = 14
		}
	}
}

func updateGrenades(arena Map, grenades []Grenade, players []*Player, explosions *[]Explosion, dynamite *[]DynamiteEffect, blast *[]BlastEffect) {
	for i := range grenades {
		g := &grenades[i]
		if g.Dead {
			continue
		}

		if !g.HitGround {
			g.Rotation += g.RotSpeed
		} else if math.Abs(90-g.Rotation) < math.Abs(-90-g.Rotation) {
			g.Rotation += (90 - g.Rotation) / 5
		} else {
			g.Rotation += (-90 - g.Rotation) / 5
		}

		if g.VY < 24 {
			g.VY += arena.Gravity + 0.12
		}
		// Source spawns fx_dynamite at the grenade's tip before moving the
		// grenade itself this frame.
		dirX := math.Cos((g.Rotation-90)*math.Pi/180) * 25
		dirY := math.Sin((g.Rotation-90)*math.Pi/180) * 25
		*dynamite = append(*dynamite, newDynamiteEffect(g.X+dirX, g.Y+dirY))
		g.X += g.VX
		g.Y += g.VY

		if !g.FreePass && g.VY > 0 {
			for _, r := range arena.Platforms {
				if !r.ContainsX(g.X) || g.Y+8 < r.Y || g.Y+8-g.VY > r.Y {
					continue
				}
				g.Y -= g.VY
				for step := 1; step <= 5; step++ {
					y := g.Y + 8 + float64(step)*(g.VY/5)
					if r.Contains(g.X, y) {
						g.Y += g.VY / 5 * float64(step-1)
						g.RotSpeed *= 0.5
						g.VX *= 0.6
						g.VY *= -0.4
						g.HitGround = true
						break
					}
				}
				break
			}
		}

		insideY := false
		insideY10 := false
		for _, r := range arena.Platforms {
			if r.Contains(g.X, g.Y) {
				insideY = true
			}
			if r.Contains(g.X, g.Y+10) {
				insideY10 = true
			}
			if insideY && insideY10 {
				break
			}
		}
		// Exact source order:
		// 1) clear freepass only when neither _Y nor _Y+10 hits ground,
		// 2) set freepass only when _Y itself hits ground.
		// Using (_Y || _Y+10) for step 2 lets the grenade enter a platform
		// several pixels too far before collisions resume.
		if !insideY && !insideY10 {
			g.FreePass = false
		}
		if insideY && !g.FreePass {
			g.FreePass = true
		}

		g.Fuse++
		if g.Fuse > 50 {
			explodeGrenade(g, players)
			*blast = append(*blast, spawnGrenadeBlastEffects(g.X, g.Y-20)...)
			*explosions = append(*explosions, Explosion{X: g.X, Y: g.Y - 20, Life: 18})
			g.Dead = true
			continue
		}

		if g.Y >= 900 || g.X < -500 || g.X > 1400 {
			g.Dead = true
		}
	}
}

func explodeGrenade(g *Grenade, players []*Player) {
	for _, p := range players {
		if !p.Active {
			continue
		}
		dx := p.X - g.X
		dy := p.Y - 30 - g.Y
		distance := math.Round(math.Sqrt(dx*dx + dy*dy))
		if distance > 200 {
			continue
		}
		radians := math.Atan2(dy, dx)
		pushX := math.Cos(radians) * (350 - distance) / 5
		pushY := math.Sin(radians) * (350 - distance) / 10
		if p.PerkNumber == 6 || p.ShieldTime > 0 {
			// Source switches from /5,/10 to /25,/50: exactly one fifth
			// of the normal grenade impulse on each axis.
			pushX /= 5
			pushY /= 5
		}
		p.VX += pushX
		p.VY += pushY
		p.LastHitBy = g.OwnerID
		p.HitByGrenade = true
	}
}

func newDynamiteEffect(x, y float64) DynamiteEffect {
	vx := rand.Float64()*10 - 5
	vy := rand.Float64()*4 - 2
	rot := rand.Float64()*10 + 5
	if rand.Intn(2) == 0 {
		rot *= -1
	}
	// DefineSprite_343 frame script moves the new clip once immediately.
	return DynamiteEffect{
		X: x + vx, Y: y + vy,
		VX: vx, VY: vy, RotSpeed: rot,
		Scale: 1, Alpha: 1, Fresh: true,
	}
}

func updateDynamiteEffects(effects []DynamiteEffect) {
	for i := range effects {
		e := &effects[i]
		if e.Dead {
			continue
		}
		if e.Fresh {
			e.Fresh = false
			continue
		}
		e.X += e.VX
		e.VX += e.VX * -0.8
		e.Y += e.VY
		e.Scale -= 0.05
		e.Rotation += e.RotSpeed
		e.Alpha -= 0.20
		if e.Alpha <= 0.01 {
			e.Dead = true
		}
	}
}

func compactDynamiteEffects(in []DynamiteEffect) []DynamiteEffect {
	out := in[:0]
	for _, e := range in {
		if !e.Dead {
			out = append(out, e)
		}
	}
	return out
}

func updateExplosions(explosions []Explosion) {
	for i := range explosions {
		if explosions[i].Life > 0 {
			explosions[i].Life--
		}
	}
}

func compactGrenades(in []Grenade) []Grenade {
	out := in[:0]
	for _, v := range in {
		if !v.Dead {
			out = append(out, v)
		}
	}
	return out
}

func compactExplosions(in []Explosion) []Explosion {
	out := in[:0]
	for _, v := range in {
		if v.Life > 0 {
			out = append(out, v)
		}
	}
	return out
}
