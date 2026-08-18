package game

// PlayerTrailEffect ports fx_playertrail (Symbol 283). The source creates one
// every second speedtime tick with CP2, keeps it at 20% alpha for 10 frames,
// and uses 80% scale (50% while the player is mini).
type PlayerTrailEffect struct {
	X, Y        float64
	Facing      int
	Scale       float64
	PlayerColor int
	ShirtNumber int
	Life        int
	Fresh       bool
	LegFrame1   int
	LegFrame2   int
	Dead        bool
}

func newPlayerTrailEffect(p *Player, mini bool) PlayerTrailEffect {
	scale := 0.8
	if mini {
		scale = 0.5
	}
	return PlayerTrailEffect{
		X:           p.X,
		Y:           p.Y,
		Facing:      p.Facing,
		Scale:       scale,
		PlayerColor: p.PlayerColor,
		ShirtNumber: p.ShirtNumber,
		Life:        10,
		Fresh:       true,
	}
}

func updatePlayerTrailEffects(effects []PlayerTrailEffect, leg1Frames, leg2Frames int) {
	for i := range effects {
		e := &effects[i]
		if e.Dead {
			continue
		}
		// attachMovie() creates the clip after the current player onEnterFrame;
		// its own onEnterFrame starts on the following game frame.
		if e.Fresh {
			e.Fresh = false
			continue
		}
		e.Life--
		if e.Life <= 0 {
			e.Dead = true
			continue
		}
		if leg1Frames > 0 {
			e.LegFrame1 = (e.LegFrame1 + 1) % leg1Frames
		}
		if leg2Frames > 0 {
			e.LegFrame2 = (e.LegFrame2 + 1) % leg2Frames
		}
	}
}

func compactPlayerTrailEffects(in []PlayerTrailEffect) []PlayerTrailEffect {
	out := in[:0]
	for _, e := range in {
		if !e.Dead {
			out = append(out, e)
		}
	}
	return out
}
