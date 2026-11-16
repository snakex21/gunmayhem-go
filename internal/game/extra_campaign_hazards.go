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
	extraCampaignHazardWind
)

const (
	// extraHazardOwnerID marks neutral hazard grenades: they push but keep
	// the kill credit of the last real attacker (see explodeGrenade).
	extraHazardOwnerID = -1
	// dynamiteRainWarningFrames: the landing marker appears this long before
	// the dynamite is dropped, and stays until it has exploded.
	dynamiteRainWarningFrames = 60
	dynamiteRainLingerFrames  = 52
	windWarningFrames         = 90
	windGustFrames            = 150
	windForce                 = 0.32
	windMaxSpeed              = 12.0
	hazardMarkerDynamite      = 1
)

// HazardMarker is a world-space landing warning (synced for rendering).
type HazardMarker struct {
	Kind    int     `cbor:",omitempty"`
	X, Y    float64 `cbor:",omitempty"`
	Life    int     `cbor:",omitempty"`
	MaxLife int     `cbor:",omitempty"`
	// Drop is the frame (counting down Life) at which the payload falls.
	Drop int  `cbor:",omitempty"`
	Dead bool `cbor:",omitempty"`
}

func (h extraCampaignHazard) interval() int {
	switch h {
	case extraCampaignHazardDynamiteRain:
		return 100
	case extraCampaignHazardShockPulse:
		return 300
	case extraCampaignHazardWind:
		return 420
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
	case extraCampaignHazardWind:
		return "CROSSWIND"
	default:
		return ""
	}
}

func (g *Game) activeExtraCampaignMission() *extraCampaignMission {
	if g == nil || !g.campaignMode || !g.campaignSetExtra || g.campaignLevel < 1 || g.campaignLevel > len(extraCampaignMissions) {
		return nil
	}
	return &extraCampaignMissions[g.campaignLevel-1]
}

func (g *Game) activeExtraCampaignHazard() extraCampaignHazard {
	if m := g.activeExtraCampaignMission(); m != nil {
		return m.Hazard
	}
	return extraCampaignHazardNone
}

// extraHazardInterval is the mission's period override or the default.
func (g *Game) extraHazardInterval() int {
	m := g.activeExtraCampaignMission()
	if m == nil {
		return 0
	}
	if m.Hazard != extraCampaignHazardNone && m.HazardInterval > 0 {
		return m.HazardInterval
	}
	return m.Hazard.interval()
}

// extraHazardRoll is a deterministic per-cycle pseudo random value 0..n-1,
// so the landing marker and the dynamite agree and tests are reproducible.
func (g *Game) extraHazardRoll(n int) int {
	if n <= 0 {
		return 0
	}
	h := uint32(g.extraHazardCycle+1)*2654435761 ^ uint32(g.campaignLevel)*40503
	h ^= h >> 15
	h *= 2246822519
	h ^= h >> 13
	return int(h % uint32(n))
}

// hazardLandingY is the top of the first platform below the sky at x, or
// the arena bottom when the shell would fall straight through.
func hazardLandingY(arena Map, x float64) float64 {
	best := math.Inf(1)
	for _, r := range arena.Platforms {
		if r.ContainsX(x) && r.Y < best {
			best = r.Y
		}
	}
	if math.IsInf(best, 1) {
		_, _, _, bottom := campaignArenaBounds(arena)
		return bottom
	}
	return best
}

func (g *Game) updateExtraCampaignHazards() {
	hazard := g.activeExtraCampaignHazard()
	interval := g.extraHazardInterval()
	g.updateHazardMarkers()
	if interval == 0 || g.paused || g.GameWin || g.teamGameWin {
		return
	}
	if hazard == extraCampaignHazardWind && g.extraWindTime > 0 {
		g.extraWindTime--
		g.applyExtraWind(float64(g.extraWindDir) * windForce)
	}
	g.extraHazardFrame++
	if hazard == extraCampaignHazardDynamiteRain && g.extraHazardFrame == interval-dynamiteRainWarningFrames {
		g.placeDynamiteMarker()
	}
	if g.extraHazardFrame < interval {
		return
	}
	g.extraHazardFrame = 0

	switch hazard {
	case extraCampaignHazardDynamiteRain:
		x := g.extraHazardPendingX
		if !g.extraHazardPending {
			// Interval shorter than the warning: mark and drop at once.
			x = g.placeDynamiteMarker()
		}
		g.extraHazardPending = false
		rot := 7 + rand.Float64()*7
		if rand.Intn(2) == 0 {
			rot = -rot
		}
		g.grenades = append(g.grenades, Grenade{
			X: x, Y: -120, VY: 1.5, Rotation: 0, RotSpeed: rot,
			OwnerID: extraHazardOwnerID, Type: ThrowableDynamite,
		})
		g.extraHazardCycle++
	case extraCampaignHazardShockPulse:
		g.fireReactorPulse()
		g.extraHazardCycle++
	case extraCampaignHazardWind:
		if g.extraWindDir == 0 {
			g.extraWindDir = 1
		} else {
			g.extraWindDir = -g.extraWindDir
		}
		g.extraWindTime = windGustFrames
		g.extraHazardCycle++
	}
}

// placeDynamiteMarker picks the next landing point and shows its marker.
func (g *Game) placeDynamiteMarker() float64 {
	width := int(g.arena.CrateMaxX - g.arena.CrateMinX)
	x := g.arena.CrateMinX + float64(g.extraHazardRoll(width))
	life := dynamiteRainWarningFrames + dynamiteRainLingerFrames
	g.extraHazardMarkers = append(g.extraHazardMarkers, HazardMarker{
		Kind: hazardMarkerDynamite, X: x, Y: hazardLandingY(g.arena, x),
		Life: life, MaxLife: life, Drop: dynamiteRainLingerFrames,
	})
	g.extraHazardPendingX = x
	g.extraHazardPending = true
	return x
}

func (g *Game) updateHazardMarkers() {
	out := g.extraHazardMarkers[:0]
	for _, m := range g.extraHazardMarkers {
		m.Life--
		if m.Life <= 0 {
			continue
		}
		out = append(out, m)
	}
	g.extraHazardMarkers = out
}

// fireReactorPulse throws everything away from the reactor core. BLAST
// SHIELD and the shield powerup take one fifth (like a grenade); the ability
// knockback window around the hazard handles HEAVY / ANCHOR / SHIELD. The
// pulse is neutral: whoever last hit a victim keeps the kill credit.
func (g *Game) fireReactorPulse() {
	centerX := (g.arena.CrateMinX + g.arena.CrateMaxX) / 2
	centerY := hazardLandingY(g.arena, centerX) - 40
	for _, p := range g.hitTargets() {
		if p == nil || !p.Active {
			continue
		}
		direction := 1.0
		if p.X < centerX {
			direction = -1
		}
		strength := 9 + math.Min(4, math.Abs(p.X-centerX)/90)
		pushX, pushY := direction*strength, -8.0
		if p.PerkNumber == 6 || p.ShieldTime > 0 {
			pushX /= 5
			pushY /= 5
		}
		p.VX += pushX
		p.VY += pushY
		p.HitNumber++
		p.HitTimer = 0
	}
	g.explosions = append(g.explosions, Explosion{X: centerX, Y: centerY, Life: 18})
	g.blastFX = append(g.blastFX, spawnGrenadeBlastEffects(centerX, centerY)...)
	g.playRandomExplosionSound()
}

func (g *Game) applyExtraWind(force float64) {
	for _, p := range g.hitTargets() {
		if p == nil || !p.Active || p.IsDouble {
			continue
		}
		if p.VX*force < 0 || math.Abs(p.VX) < windMaxSpeed {
			p.VX += force
		}
	}
}

// extraHazardSecondsLeft is the HUD countdown to the next hazard in real
// seconds.
func (g *Game) extraHazardSecondsLeft() float64 {
	remaining := g.extraHazardInterval() - g.extraHazardFrame
	if remaining < 0 {
		remaining = 0
	}
	return float64(remaining) / TicksPerSecond
}

func (g *Game) drawExtraCampaignHazardHUD(screen *ebiten.Image) {
	hazard := g.activeExtraCampaignHazard()
	interval := g.extraHazardInterval()
	if interval == 0 || g.GameWin || g.teamGameWin {
		return
	}
	remaining := interval - g.extraHazardFrame
	if remaining < 0 {
		remaining = 0
	}
	seconds := g.extraHazardSecondsLeft()
	fill := color.NRGBA{R: 0x22, G: 0x25, B: 0x2a, A: 0xe8}
	accent := color.NRGBA{R: 0xff, G: 0xd0, B: 0x35, A: 0xff}
	warn := remaining <= 45
	if hazard == extraCampaignHazardWind {
		warn = remaining <= windWarningFrames || g.extraWindTime > 0
	}
	if warn {
		fill = color.NRGBA{R: 0x72, G: 0x18, B: 0x10, A: 0xee}
		accent = color.NRGBA{R: 0xff, G: 0x78, B: 0x32, A: 0xff}
	}
	text := hazard.label() + "  " + formatOneDecimal(seconds) + "s"
	if hazard == extraCampaignHazardWind {
		next := 1
		if g.extraWindDir > 0 {
			next = -1
		}
		switch {
		case g.extraWindTime > 0:
			text = "GUST " + windArrows(g.extraWindDir)
		case remaining <= windWarningFrames:
			text = "GUST INCOMING " + windArrows(next) + "  " + formatOneDecimal(seconds) + "s"
		}
	}
	ebitenutil.DrawRect(screen, 330, 72, 240, 35, fill)
	ebitenutil.DrawRect(screen, 330, 72, 240, 3, accent)
	drawSourceMenuText(screen, text, menuFontCondensedExtraBold, 15, color.White, 350, 80)
}

func windArrows(dir int) string {
	if dir < 0 {
		return "<<<"
	}
	return ">>>"
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
