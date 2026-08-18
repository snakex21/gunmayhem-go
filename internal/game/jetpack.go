package game

import "math/rand"

type JetThrustEffect struct {
	X, Y     float64
	VX, VY   float64
	Rotation float64
	Scale    float64
	FXAlpha  float64
	Fresh    bool
	Dead     bool
}

type DropPackEffect struct {
	X, Y     float64
	VX, VY   float64
	Rotation float64
	RotSpeed float64
	ScaleX   float64
	Fresh    bool
	Dead     bool
}

// DefineSprite_176_fx_jetpack/frame_1/DoAction.as.
func newJetThrustEffect(x, y float64) JetThrustEffect {
	vx := rand.Float64()*10 - 5
	vy := rand.Float64()*5 + 1
	return JetThrustEffect{
		X:        x + (rand.Float64()*2 - 1),
		Y:        y + rand.Float64()*2,
		VX:       vx,
		VY:       vy,
		Rotation: float64(rand.Intn(360)),
		Scale:    1,
		FXAlpha:  0,
		Fresh:    true,
	}
}

func updateJetThrustEffects(effects []JetThrustEffect) {
	for i := range effects {
		e := &effects[i]
		if e.Dead {
			continue
		}
		if e.Fresh {
			e.Fresh = false
			continue
		}
		if e.FXAlpha < 1 {
			e.FXAlpha += 0.15
			if e.FXAlpha > 1 {
				e.FXAlpha = 1
			}
		}
		e.X += e.VX
		e.Y += e.VY
		e.VX *= 0.9
		e.Scale -= 0.07
		if e.Y >= 900 || e.Scale <= 0.10 {
			e.Dead = true
		}
	}
}

func compactJetThrustEffects(in []JetThrustEffect) []JetThrustEffect {
	out := in[:0]
	for _, e := range in {
		if !e.Dead {
			out = append(out, e)
		}
	}
	return out
}

// DefineSprite_172_fx_droppack/frame_1/DoAction.as. The parent passes
// rotation=facing*-1, so the source script derives vx directly from that value.
func newDropPackEffect(x, y float64, facing int) DropPackEffect {
	rotation := float64(facing * -1)
	sx := 1.0
	if rotation < 0 {
		sx = -1
	}
	rotSpeed := rand.Float64()*2 + 5
	if rand.Intn(2) == 0 {
		rotSpeed *= -1
	}
	return DropPackEffect{
		X: x, Y: y,
		VX:       5 * rotation,
		VY:       rand.Float64()*5 - 10,
		Rotation: rotation,
		RotSpeed: rotSpeed,
		ScaleX:   sx,
		Fresh:    true,
	}
}

func updateDropPackEffects(arena Map, effects []DropPackEffect) {
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
		e.Y += e.VY
		e.Rotation += e.RotSpeed
		e.VY += arena.Gravity * 1.3
		if e.Y >= 900 {
			e.Dead = true
		}
	}
}

func compactDropPackEffects(in []DropPackEffect) []DropPackEffect {
	out := in[:0]
	for _, e := range in {
		if !e.Dead {
			out = append(out, e)
		}
	}
	return out
}
