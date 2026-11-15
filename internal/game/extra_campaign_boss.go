package game

import (
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Real EXTRA bosses: a boss bot keeps its independent life pool, but gains
// knockback resistance (Player.KnockbackScale, applied in the ability
// knockback windows so it covers bullets, blasts, gas, fire and hazards),
// phases keyed to its remaining lives, and a named HUD bar. The same phase
// table works for an HP-based non-player boss (CampaignEnemy.HP); see
// docs/campaign_enemies.md.

type extraCampaignBossPhase struct {
	// Lives: the phase starts once the boss has this many lives (or HP)
	// left or fewer. Phases are listed from the first to the last.
	Lives  int
	Banner string
	// Gun swaps the mission weapon (0 keeps it).
	Gun int
	// ShotDelay: >0 sets a new trigger pause, <0 removes it, 0 keeps it.
	ShotDelay int
	// MoveScale / KnockbackScale replace the current values when > 0.
	MoveScale      float64
	KnockbackScale float64
	// Reinforcements dropped in when the phase starts.
	SummonDrones  int
	SummonShields int
}

type extraCampaignBoss struct {
	Title          string
	KnockbackScale float64
	MoveScale      float64
	Phases         []extraCampaignBossPhase
}

const bossBannerFrames = 150

// bossPhaseFor returns how many phases are active for the remaining
// lives / HP (0 = the opening phase).
func bossPhaseFor(remaining int, phases []extraCampaignBossPhase) int {
	n := 0
	for _, ph := range phases {
		if remaining <= ph.Lives {
			n++
		} else {
			break
		}
	}
	return n
}

// extraBossSpec returns the boss data of bot p in the running EXTRA mission.
func (g *Game) extraBossSpec(p *Player) *extraCampaignBoss {
	if g == nil || p == nil || !p.AI || p.IsDouble || !g.campaignMode || !g.campaignSetExtra ||
		g.campaignLevel < 1 || g.campaignLevel > len(extraCampaignMissions) {
		return nil
	}
	for _, src := range extraCampaignMissions[g.campaignLevel-1].Enemies {
		if src.ID == p.ID {
			return src.Boss
		}
	}
	return nil
}

func applyExtraBossBase(p *Player, boss *extraCampaignBoss) {
	if p == nil || boss == nil {
		return
	}
	p.KnockbackScale = boss.KnockbackScale
	p.CampaignMoveScale = boss.MoveScale
	p.BossPhase = 0
}

// updateExtraCampaignBosses advances boss phases. A boss that loses several
// lives at once still runs every skipped phase, in order.
func (g *Game) updateExtraCampaignBosses() {
	if g == nil || !g.campaignMode || !g.campaignSetExtra || g.GameWin || g.teamGameWin {
		return
	}
	if g.bossBannerFrame > 0 {
		g.bossBannerFrame--
	}
	for _, p := range g.players {
		boss := g.extraBossSpec(p)
		if boss == nil || !p.Active {
			continue
		}
		want := bossPhaseFor(p.Lives, boss.Phases)
		for p.BossPhase < want {
			g.enterExtraBossPhase(p, boss.Phases[p.BossPhase])
			p.BossPhase++
		}
	}
}

func (g *Game) enterExtraBossPhase(p *Player, ph extraCampaignBossPhase) {
	if ph.Gun != 0 {
		p.DefaultWeapon = ph.Gun
		p.EquipWeapon(ph.Gun)
	}
	switch {
	case ph.ShotDelay > 0:
		p.AIShotDelay = ph.ShotDelay
	case ph.ShotDelay < 0:
		p.AIShotDelay = 0
	}
	p.AIFireCadence = 0
	if ph.MoveScale > 0 {
		p.CampaignMoveScale = ph.MoveScale
	}
	if ph.KnockbackScale > 0 {
		p.KnockbackScale = ph.KnockbackScale
	}
	side := 1
	if p.X > (g.arena.CrateMinX+g.arena.CrateMaxX)/2 {
		side = -1 // reinforcements arrive on the far side of the boss
	}
	for i := 0; i < ph.SummonDrones; i++ {
		g.spawnCampaignEnemy(extraCampaignUnit{Kind: campaignEnemyDrone, HP: 6}, side, g.arena)
		side = -side
	}
	for i := 0; i < ph.SummonShields; i++ {
		g.spawnCampaignEnemy(extraCampaignUnit{Kind: campaignEnemyShieldBearer}, side, g.arena)
		side = -side
	}
	if ph.Banner != "" {
		g.bossBanner = p.Name + ": " + ph.Banner
		g.bossBannerFrame = bossBannerFrames
	}
	g.playSourceSFX("emp.wav", true)
}

// --- HUD -------------------------------------------------------------------

func (g *Game) drawExtraCampaignBossHUD(screen *ebiten.Image) {
	if g == nil || !g.campaignMode || !g.campaignSetExtra || g.GameWin || g.teamGameWin {
		return
	}
	y := 10.0
	for _, p := range g.players {
		boss := g.extraBossSpec(p)
		if boss == nil {
			continue
		}
		maxLives := p.TotalLives
		if maxLives <= 0 {
			maxLives = 1
		}
		ticks := make([]float64, 0, len(boss.Phases))
		for _, ph := range boss.Phases {
			ticks = append(ticks, float64(ph.Lives)/float64(maxLives))
		}
		frac := float64(p.Lives) / float64(maxLives)
		drawBossBar(screen, p.Name, boss.Title, frac, ticks, p.BossPhase, p.Active, y)
		y += 44
	}
	// HP bosses: elite mechs and the gunship (campaign_enemies_visuals_big.go).
	g.drawCampaignBigEnemyBars(screen, y)
	if g.bossBannerFrame > 0 && g.bossBanner != "" {
		a := math.Min(1, float64(g.bossBannerFrame)/25)
		w := measureMenuText(menuFontCondensedExtraBold, 22, g.bossBanner)
		x := (ScreenWidth - w) / 2
		ebitenutil.DrawRect(screen, x-18, 186, w+36, 38, color.NRGBA{R: 0x60, G: 0x08, B: 0x08, A: uint8(220 * a)})
		drawSourceMenuText(screen, g.bossBanner, menuFontCondensedExtraBold, 22, color.NRGBA{R: 0xff, G: 0xe0, B: 0x60, A: uint8(255 * a)}, x, 193)
	}
}

// drawBossBar draws one named boss health bar; frac is 0..1, ticks are the
// phase thresholds (0..1).
func drawBossBar(screen *ebiten.Image, name, title string, frac float64, ticks []float64, phase int, alive bool, y float64) {
	const x, w, h = 270.0, 360.0, 12.0
	frac = math.Max(0, math.Min(1, frac))
	ebitenutil.DrawRect(screen, x-6, y, w+12, 40, color.NRGBA{R: 0x14, G: 0x16, B: 0x1a, A: 0xd8})
	label := name
	if title != "" {
		label = title + "  -  " + name
	}
	drawSourceMenuText(screen, label, menuFontCondensedExtraBold, 13, color.White, x, y+3)
	phaseText := "PHASE " + strconv.Itoa(phase+1)
	if !alive {
		phaseText = "DEFEATED"
	}
	pw := measureMenuText(menuFontCondensedExtraBold, 12, phaseText)
	drawSourceMenuText(screen, phaseText, menuFontCondensedExtraBold, 12, color.NRGBA{R: 0xff, G: 0xb0, B: 0x40, A: 0xff}, x+w-pw, y+4)
	by := y + 22
	ebitenutil.DrawRect(screen, x, by, w, h, color.NRGBA{R: 0x3a, G: 0x0c, B: 0x0c, A: 0xff})
	fill := color.NRGBA{R: 0xe0, G: 0x2a, B: 0x1e, A: 0xff}
	if phase > 0 {
		fill = color.NRGBA{R: 0xff, G: 0x60, B: 0x10, A: 0xff}
	}
	ebitenutil.DrawRect(screen, x, by, w*frac, h, fill)
	ebitenutil.DrawRect(screen, x, by, w*frac, 3, color.NRGBA{R: 0xff, G: 0xc0, B: 0x90, A: 0xb0})
	for _, t := range ticks {
		if t <= 0 || t >= 1 {
			continue
		}
		ebitenutil.DrawRect(screen, x+w*t-1, by-3, 2, h+6, color.NRGBA{R: 0xff, G: 0xe0, B: 0x60, A: 0xff})
	}
}
