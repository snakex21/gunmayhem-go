package game

import "math/rand"

type BlastEffectKind int

const (
	BlastEX6 BlastEffectKind = iota + 1
	BlastEX2
	BlastEX
	BlastEX4
	BlastEX5
	BlastEX3
)

type BlastEffect struct {
	Kind BlastEffectKind
	X, Y float64

	VX, VY   float64
	Rotation float64
	RotSpeed float64
	ScaleX   float64
	ScaleY   float64
	Alpha    float64
	Level    int
	Fresh    bool
	Dead     bool
}

// Grenade source stack from DefineSprite_665_wep_grenade. The original saved
// default quality is MEDIUM, so the `_quality != "LOW"` five-particle loops are
// active unless a later ported settings screen changes that source value.
func spawnGrenadeBlastEffects(x, y float64) []BlastEffect {
	out := make([]BlastEffect, 0, 14)
	out = append(out,
		newBlastEffect(BlastEX6, x, y),
		newBlastEffect(BlastEX2, x, y),
		newBlastEffect(BlastEX, x, y),
		newBlastEffect(BlastEX4, x, y),
	)
	for i := 0; i < 5; i++ {
		out = append(out, newBlastEffect(BlastEX5, x, y))
	}
	for i := 0; i < 5; i++ {
		out = append(out, newBlastEffect(BlastEX3, x, y))
	}
	return out
}

func newBlastEffect(kind BlastEffectKind, x, y float64) BlastEffect {
	e := BlastEffect{
		Kind: kind, X: x, Y: y,
		ScaleX: 1, ScaleY: 1, Alpha: 1,
		Level: 1, Fresh: true,
	}
	switch kind {
	case BlastEX2:
		// DefineSprite_306_fx_ex2.
		e.ScaleX, e.ScaleY = 0.10, 0.10
	case BlastEX:
		// DefineSprite_308_fx_ex. Rotation parameter is zero for the grenade;
		// multiplying it by -1 therefore intentionally leaves zero.
	case BlastEX4:
		// DefineSprite_312_fx_ex4. Source contains `_yscale = xscale` (without
		// underscore), so the first attached frame has an undefined/zero Y scale;
		// the first onEnterFrame immediately sets it from _xscale.
		e.Rotation += float64(rand.Intn(20) - 10)
		e.ScaleX, e.ScaleY = 0.10, 0
	case BlastEX5:
		// DefineSprite_315_fx_ex5 constructor, including its immediate vx/vy move.
		e.X += float64(rand.Intn(80) - 40)
		e.Y += float64(rand.Intn(80) - 20)
		e.VX = rand.Float64()*10 - 5
		e.VY = rand.Float64()*4 - 5
		e.X += e.VX
		e.Y += e.VY
		e.ScaleX = 1 + float64(rand.Intn(100))/100
		e.ScaleY = e.ScaleX
	case BlastEX3:
		// DefineSprite_310_fx_ex3 constructor.
		e.X += float64(rand.Intn(20) - 10)
		e.Y -= float64(rand.Intn(20))
		e.VX = rand.Float64()*20 - 10
		e.VY = rand.Float64()*10 - 20
		e.RotSpeed = rand.Float64()*10 + 5
		if rand.Intn(2) == 0 {
			e.RotSpeed *= -1
		}
	}
	return e
}

func updateBlastEffects(arena Map, effects []BlastEffect) {
	for i := range effects {
		e := &effects[i]
		if e.Dead {
			continue
		}
		if e.Fresh {
			e.Fresh = false
			continue
		}

		switch e.Kind {
		case BlastEX6, BlastEX:
			if e.Level == 1 {
				e.ScaleX += (2 - e.ScaleX) / 3
				e.ScaleY = e.ScaleX
				if e.ScaleX >= 1.90 {
					e.Level = 2
				}
			}
			if e.Level == 2 {
				e.Alpha -= 0.20
			}
		case BlastEX2:
			if e.Level == 1 {
				e.ScaleX += (2 - e.ScaleX) / 3
				e.ScaleY = e.ScaleX
				if e.ScaleX >= 1.95 {
					e.Level = 2
				}
			}
			if e.Level == 2 {
				e.Alpha -= 0.10
			}
		case BlastEX4:
			if e.Level == 1 {
				e.ScaleX += (2.5 - e.ScaleX) / 3
				e.ScaleY = e.ScaleX
				if e.ScaleX >= 2.49 {
					e.Level = 2
				}
			}
			if e.Level == 2 {
				e.Alpha -= 0.10
			}
		case BlastEX5:
			e.X += e.VX
			e.VY -= 0.1
			e.VX *= 0.9
			e.Y += e.VY
			e.ScaleX -= 0.06
			e.ScaleY = e.ScaleX
			if e.ScaleX <= 0.10 {
				e.Dead = true
			}
		case BlastEX3:
			e.X += e.VX
			e.Y += e.VY
			e.Rotation += e.RotSpeed
			e.ScaleX -= 0.02
			e.ScaleY = e.ScaleX
			e.VY += arena.Gravity * 1.3
			if e.Y >= 900 {
				e.Dead = true
			}
		}

		if e.Alpha <= 0.01 {
			e.Dead = true
		}
	}
}

func compactBlastEffects(in []BlastEffect) []BlastEffect {
	out := in[:0]
	for _, e := range in {
		if !e.Dead {
			out = append(out, e)
		}
	}
	return out
}
