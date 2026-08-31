package game

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type extraCampaignHazard int

const (
	extraCampaignHazardNone extraCampaignHazard = iota
	extraCampaignHazardDynamiteRain
	extraCampaignHazardShockPulse
)

func (h extraCampaignHazard) interval() int {
	switch h {
	case extraCampaignHazardDynamiteRain:
		return 100
	case extraCampaignHazardShockPulse:
		return 300
	default:
		return 0
	}
}

func (h extraCampaignHazard) label() string {
	switch h {
	case extraCampaignHazardDynamiteRain:
		return "DYNAMITE RAIN"
	case extraCampaignHazardShockPulse:
		return "REACTOR PULSE"
	default:
		return ""
	}
}

func (g *Game) activeExtraCampaignHazard() extraCampaignHazard {
	if g == nil || !g.campaignMode || !g.campaignSetExtra || g.campaignLevel < 1 || g.campaignLevel > len(extraCampaignMissions) {
		return extraCampaignHazardNone
	}
	return extraCampaignMissions[g.campaignLevel-1].Hazard
}

func (g *Game) updateExtraCampaignHazards() {
	hazard := g.activeExtraCampaignHazard()
	interval := hazard.interval()
	if interval == 0 || g.paused || g.GameWin || g.teamGameWin {
		return
	}
	g.extraHazardFrame++
	if g.extraHazardFrame < interval {
		return
	}
	g.extraHazardFrame = 0

	switch hazard {
	case extraCampaignHazardDynamiteRain:
		width := int(g.arena.CrateMaxX - g.arena.CrateMinX)
		x := g.arena.CrateMinX
		if width > 0 {
			x += float64(rand.Intn(width))
		}
		rot := 7 + rand.Float64()*7
		if rand.Intn(2) == 0 {
			rot = -rot
		}
		g.grenades = append(g.grenades, Grenade{
			X: x, Y: -120, VY: 1.5, Rotation: 0, RotSpeed: rot,
			OwnerID: 0, Type: ThrowableDynamite,
		})
	case extraCampaignHazardShockPulse:
		centerX := (g.arena.CrateMinX + g.arena.CrateMaxX) / 2
		centerY := 285.0
		for _, p := range g.players {
			if p == nil || !p.Active {
				continue
			}
			direction := 1.0
			if p.X < centerX {
				direction = -1
			}
			strength := 9 + math.Min(4, math.Abs(p.X-centerX)/90)
			p.VX += direction * strength
			p.VY -= 8
			p.HitNumber++
			p.HitTimer = 0
			p.LastHitBy = 0
		}
		g.explosions = append(g.explosions, Explosion{X: centerX, Y: centerY, Life: 18})
		g.blastFX = append(g.blastFX, spawnGrenadeBlastEffects(centerX, centerY)...)
		g.playRandomExplosionSound()
	}
}

func (g *Game) drawExtraCampaignHazardHUD(screen *ebiten.Image) {
	hazard := g.activeExtraCampaignHazard()
	interval := hazard.interval()
	if interval == 0 || g.GameWin || g.teamGameWin {
		return
	}
	remaining := interval - g.extraHazardFrame
	if remaining < 0 {
		remaining = 0
	}
	seconds := float64(remaining) / 60
	fill := color.NRGBA{R: 0x22, G: 0x25, B: 0x2a, A: 0xe8}
	accent := color.NRGBA{R: 0xff, G: 0xd0, B: 0x35, A: 0xff}
	if remaining <= 45 {
		fill = color.NRGBA{R: 0x72, G: 0x18, B: 0x10, A: 0xee}
		accent = color.NRGBA{R: 0xff, G: 0x78, B: 0x32, A: 0xff}
	}
	ebitenutil.DrawRect(screen, 330, 72, 240, 35, fill)
	ebitenutil.DrawRect(screen, 330, 72, 240, 3, accent)
	drawSourceMenuText(screen, hazard.label()+"  "+formatOneDecimal(seconds)+"s", menuFontCondensedExtraBold, 15, color.White, 350, 80)
}

func formatOneDecimal(value float64) string {
	whole := int(value)
	tenths := int(math.Round((value - float64(whole)) * 10))
	if tenths >= 10 {
		whole++
		tenths = 0
	}
	return itoaFast(whole) + "." + itoaFast(tenths)
}
