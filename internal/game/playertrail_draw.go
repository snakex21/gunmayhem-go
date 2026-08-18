package game

import "github.com/hajimehoshi/ebiten/v2"

func (g *Game) drawPlayerTrails(screen *ebiten.Image) {
	for i := range g.playerTrails {
		fx := &g.playerTrails[i]
		if fx.Dead || fx.Life <= 0 {
			continue
		}
		p := Player{
			X:             g.worldX(fx.X),
			Y:             g.worldY(fx.Y),
			Facing:        fx.Facing,
			PlayerScale:   fx.Scale,
			Alpha:         0.20,
			PlayerColor:   fx.PlayerColor,
			ShirtNumber:   fx.ShirtNumber,
			HeadFrame:     0,
			VisualBodyY:   -60,
			VisualEyesY:   -46,
			LegFrame1:     fx.LegFrame1,
			LegFrame2:     fx.LegFrame2,
		}

		// Symbol283 source layer order, back to front: leg2, body/head,
		// player_hat, eyes, leg1. The current player renderer does not yet have
		// the separate Symbol269 hat path, so this ports every already-recovered
		// source component rather than substituting a synthetic silhouette.
		leg := g.assets.LegColors[p.PlayerColor]
		if leg == nil {
			leg = g.assets.LegColors[0]
		}
		if len(g.assets.Leg2Timeline) > 0 {
			frame := p.LegFrame2 % len(g.assets.Leg2Timeline)
			drawPlayerLeg(screen, leg, &p, g.assets.Leg2Timeline[frame].Matrix,
				9*float64(p.Facing), -5.6, 0)
		}
		g.drawPlayerBody(screen, &p)
		drawPlayerPart(screen, g.assets.Eyes, &p, 0, p.VisualEyesY)
		if len(g.assets.Leg1Timeline) > 0 {
			frame := p.LegFrame1 % len(g.assets.Leg1Timeline)
			drawPlayerLeg(screen, leg, &p, g.assets.Leg1Timeline[frame].Matrix,
				-12*float64(p.Facing), -0.2, 0)
		}
	}
}
