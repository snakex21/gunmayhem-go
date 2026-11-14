package game

import (
	"fmt"
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
)

// zombie_hud.go: the in-game zombie HUD (Mayhem UI colours and font): the
// wave panel, banners, the boss bar and one card per player with HP, points,
// weapon and status. It replaces the source lives cards in a zombie match.

var zombieHUDCardX = [4]float64{120, 340, 560, 780}

func (g *Game) drawZombieHUD(screen *ebiten.Image) {
	m := g.zombie
	if m == nil {
		return
	}
	g.drawZombieWavePanel(screen)
	if m.hud.BossID != 0 && m.hud.BossMaxHP > 0 {
		frac := float64(m.hud.BossHP) / float64(m.hud.BossMaxHP)
		phase := 0
		if m.hud.BossHP*2 < m.hud.BossMaxHP {
			phase = 1
		}
		drawBossBar(screen, zombieStats[zombieBoss].Name, fmt.Sprintf("BOSS LV %d", max(1, m.hud.BossLevel)), frac, []float64{0.5}, phase, true, 54)
	}
	for i := range m.hud.Players {
		zp := &m.hud.Players[i]
		if zp.ID == 0 {
			continue
		}
		g.drawZombiePlayerCard(screen, zp, zombieHUDCardX[i])
	}
	g.drawZombieBanner(screen)
}

func (g *Game) drawZombieWavePanel(screen *ebiten.Image) {
	m := g.zombie
	h := m.hud
	r := Rect{X: 350, Y: 6, W: 200, H: 42}
	uiFillRound(screen, r, 6, color.NRGBA{R: 0x12, G: 0x14, B: 0x18, A: 0xd8})
	uiFillRect(screen, Rect{X: r.X, Y: r.Y + r.H - 3, W: r.W, H: 3}, zombieTagColor)
	wave := "WAVE " + strconv.Itoa(h.Wave)
	if h.WaveTotal > 0 {
		wave += " / " + strconv.Itoa(h.WaveTotal)
	}
	uiText(screen, wave, true, 20, uiTextLight, r.X+r.W/2, r.Y+14, uiAlignCenter)
	sub := ""
	switch h.Phase {
	case zombiePhaseIntro:
		sub = "GET READY"
	case zombiePhaseWave:
		sub = strconv.Itoa(h.Remaining) + " ZOMBIES LEFT"
		if h.BossID != 0 {
			sub = "BOSS + " + strconv.Itoa(max(0, h.Remaining-1))
		}
	case zombiePhaseBreak:
		sub = "SHOP  -  NEXT WAVE IN " + strconv.Itoa((h.PhaseTimer+TicksPerSecond-1)/TicksPerSecond)
	case zombiePhaseOver:
		sub = "GAME OVER"
	case zombiePhaseWon:
		sub = "SURVIVED"
	}
	uiText(screen, sub, false, uiSizeHint, uiTextMuted, r.X+r.W/2, r.Y+31, uiAlignCenter)
	if h.Mode == zombieModeEndless && h.RecordWave > 0 {
		uiText(screen, "RECORD: WAVE "+strconv.Itoa(h.RecordWave), false, uiSizeHint, uiTextMuted, r.X+r.W+10, r.Y+14, uiAlignLeft)
	}
}

func (g *Game) drawZombieBanner(screen *ebiten.Image) {
	m := g.zombie
	h := m.hud
	text, sub := "", ""
	clr := uiTextLight
	alpha := 1.0
	switch {
	case h.Phase == zombiePhaseOver:
		text, clr = "GAME OVER", color.NRGBA{R: 0xff, G: 0x50, B: 0x40, A: 0xff}
		sub = fmt.Sprintf("YOU SURVIVED %d WAVES", m.wavesSurvived())
	case h.Phase == zombiePhaseWon:
		text, clr = "YOU SURVIVED!", color.NRGBA{R: 0x80, G: 0xff, B: 0x70, A: 0xff}
		sub = fmt.Sprintf("ALL %d WAVES CLEARED", h.Wave)
	case h.BannerFrame > 0:
		a := float64(h.BannerFrame)
		alpha = math.Min(1, a/10) * math.Min(1, (120-a)/20)
		text = "WAVE " + strconv.Itoa(h.Wave)
		if h.Wave%5 == 0 {
			sub = "BOSS WAVE"
			clr = color.NRGBA{R: 0xff, G: 0x70, B: 0x50, A: 0xff}
		} else {
			sub = "THEY ARE COMING"
		}
	default:
		return
	}
	if alpha <= 0 {
		return
	}
	cy := 200.0
	w := uiMeasure(true, 56, text) + 80
	uiFillRound(screen, Rect{X: ScreenWidth/2 - w/2, Y: cy - 42, W: w, H: 100}, 10, uiWithAlpha(color.NRGBA{R: 0x0c, G: 0x0e, B: 0x10, A: 0xc0}, alpha))
	uiText(screen, text, true, 56, uiWithAlpha(color.NRGBA{A: 0xff}, alpha*0.6), ScreenWidth/2+3, cy+3, uiAlignCenter)
	uiText(screen, text, true, 56, uiWithAlpha(clr, alpha), ScreenWidth/2, cy, uiAlignCenter)
	if sub != "" {
		uiText(screen, sub, true, uiSizeButton, uiWithAlpha(uiTextMuted, alpha), ScreenWidth/2, cy+40, uiAlignCenter)
	}
}

func (g *Game) drawZombiePlayerCard(screen *ebiten.Image, zp *ZombiePlayerState, cx float64) {
	p := g.playerByID(zp.ID)
	r := Rect{X: cx - 104, Y: 532, W: 208, H: 60}
	col := uiOrange
	name := fmt.Sprintf("P%d", zp.ID)
	if p != nil {
		col = developedPlayerColorSwatch(p.PlayerColor)
		name = p.Name
		if p.AI {
			name += " (BOT)"
		}
	}
	uiFillRound(screen, r, 6, color.NRGBA{R: 0x14, G: 0x17, B: 0x1c, A: 0xe0})
	uiFillRect(screen, Rect{X: r.X, Y: r.Y + 6, W: 5, H: r.H - 12}, col)
	uiText(screen, uiFit(name, true, uiSizeBody, 112), true, uiSizeBody, uiTextLight, r.X+12, r.Y+12, uiAlignLeft)
	pts := strconv.Itoa(zp.Points)
	uiText(screen, pts, true, 18, color.NRGBA{R: 0xff, G: 0xd8, B: 0x40, A: 0xff}, r.X+r.W-30, r.Y+12, uiAlignRight)
	uiText(screen, "PTS", false, uiSizeHint, uiTextMuted, r.X+r.W-8, r.Y+13, uiAlignRight)

	// HP bar.
	bar := Rect{X: r.X + 12, Y: r.Y + 24, W: 100, H: 9}
	uiFillRound(screen, bar, 3, color.NRGBA{R: 0x3a, G: 0x10, B: 0x10, A: 0xff})
	frac := 0.0
	if zp.MaxHP > 0 {
		frac = math.Max(0, math.Min(1, float64(zp.HP)/float64(zp.MaxHP)))
	}
	hpCol := uiOK
	switch {
	case frac < 0.3:
		hpCol = uiDanger
	case frac < 0.6:
		hpCol = uiOrange
	}
	if frac > 0 {
		uiFillRound(screen, Rect{X: bar.X, Y: bar.Y, W: math.Max(4, bar.W*frac), H: bar.H}, 3, hpCol)
	}
	uiText(screen, strconv.Itoa(max(zp.HP, 0)), true, uiSizeHint, uiTextLight, bar.X+bar.W+6, bar.Y+bar.H/2, uiAlignLeft)
	// Throwables and self-revives.
	if p != nil {
		uiText(screen, fmt.Sprintf("G x%d", p.Grenades), true, uiSizeHint, uiTextMuted, r.X+r.W-8, bar.Y+bar.H/2, uiAlignRight)
	}
	if zp.SelfRevives > 0 {
		uiText(screen, "REV "+strconv.Itoa(zp.SelfRevives), true, uiSizeHint, uiOK, r.X+146, bar.Y+bar.H/2, uiAlignLeft)
	}
	// Weapon and ammo.
	if p != nil {
		w := p.Weapon.Def
		gun := w.DisplayName()
		ammo := ""
		if w.Number >= 9 {
			ammo = fmt.Sprintf("%d/%d", max(p.Weapon.Bullets, 0), w.Bullets)
		} else {
			ammo = fmt.Sprintf("%d", max(p.Weapon.Bullets, 0))
		}
		if zp.Gun != 0 {
			if w.Number != zp.Gun && zp.Reserve == 0 {
				gun = weaponDisplayName(zp.Gun)
				ammo = "EMPTY"
			} else {
				ammo += " +" + strconv.Itoa(zp.Reserve)
			}
		}
		uiText(screen, uiFit(gun, false, uiSizeHint, 130), false, uiSizeHint, uiTextMuted, r.X+12, r.Y+46, uiAlignLeft)
		uiText(screen, ammo, true, uiSizeHint, uiTextLight, r.X+r.W-8, r.Y+46, uiAlignRight)
	}

	// Status overlays.
	switch zp.State {
	case zombiePlayerDown:
		uiFillRound(screen, r, 6, color.NRGBA{R: 0x50, G: 0x08, B: 0x08, A: 0x90})
		uiText(screen, fmt.Sprintf("DOWN  %ds", (zp.DownTimer+TicksPerSecond-1)/TicksPerSecond), true, 20, uiTextLight, cx, r.Y+r.H/2, uiAlignCenter)
	case zombiePlayerDead:
		uiFillRound(screen, r, 6, color.NRGBA{R: 0x10, G: 0x10, B: 0x10, A: 0xb0})
		uiText(screen, "DEAD - BACK NEXT WAVE", true, uiSizeBody, uiTextMuted, cx, r.Y+r.H/2, uiAlignCenter)
	}
	if g.zombie.hud.Phase == zombiePhaseBreak && zp.State == zombiePlayerAlive {
		tag, fill := "SHOPPING", uiOrange
		if zp.Ready {
			tag, fill = "READY", uiOK
		}
		w := uiMeasure(true, uiSizeHint, tag) + 14
		b := Rect{X: cx - w/2, Y: r.Y - 22, W: w, H: 18}
		uiFillRound(screen, b, 4, fill)
		uiTextIn(screen, tag, true, uiSizeHint, uiTextDark, b, uiAlignCenter, 2)
	}
	if zp.MsgTimer > 0 && zp.Msg != zombieMsgNone && !zp.ShopOpen {
		drawZombieTag(screen, zombieMsgText[zp.Msg], cx, r.Y-34)
	}
}
