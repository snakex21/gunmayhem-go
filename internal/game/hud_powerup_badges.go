package game

import (
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// HUD badges for the timed / per-life crate powerups of the second arsenal:
// AKIMBO (a 12 s timer, dimmed while a non-pistol suspends it) and the crate
// BALLISTIC SHIELD (lasts until death, so no timer). They sit above the left
// half of the HUD card; the ability badge owns the right half of that row.

const (
	hudBadgeW = 62.0
	hudBadgeH = 15.0
)

// hudPowerupBadges lists the badges player p shows, top row first.
func hudPowerupBadges(p *Player) []int {
	if p == nil || !p.Active {
		return nil
	}
	var out []int
	if p.AkimboTime > 0 {
		out = append(out, powerupAkimbo)
	}
	if p.ShieldGearPickup {
		out = append(out, powerupBallisticShield)
	}
	return out
}

// drawHUDPowerupBadges draws them; each further badge stacks one row higher.
func (g *Game) drawHUDPowerupBadges(screen *ebiten.Image, p *Player, x, y float64) {
	for i, kind := range hudPowerupBadges(p) {
		r := Rect{X: x - 38, Y: y - 51 - float64(i)*(hudBadgeH+2), W: hudBadgeW, H: hudBadgeH}
		drawHUDPowerupBadge(screen, p, kind, r)
	}
}

func drawHUDPowerupBadge(screen *ebiten.Image, p *Player, kind int, r Rect) {
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, color.NRGBA{R: 0x14, G: 0x16, B: 0x1a, A: 0xd0})
	textClr := color.NRGBA{R: 0x10, G: 0x10, B: 0x10, A: 0xff}
	switch kind {
	case powerupAkimbo:
		frac := math.Max(0, math.Min(1, float64(p.AkimboTime)/akimboFrames))
		fill := color.NRGBA{R: 0xff, G: 0xc0, B: 0x30, A: 0xe8}
		label := "AKIMBO"
		if !akimboActive(p) {
			// Suspended by a two-handed gun or the bat: the timer still runs.
			fill = color.NRGBA{R: 0x8a, G: 0x7a, B: 0x50, A: 0xc0}
			textClr = color.NRGBA{R: 0xe8, G: 0xe8, B: 0xe8, A: 0xff}
		}
		ebitenutil.DrawRect(screen, r.X+1, r.Y+1, (r.W-2)*frac, r.H-2, fill)
		if p.AkimboTime < 90 && (p.AkimboTime/5)%2 == 0 {
			vector.StrokeRect(screen, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), 1.5, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, false)
		}
		drawAkimboBadgeIcon(screen, r.X+8, r.Y+r.H/2)
		drawSourceHUDText(screen, label, 9, textClr, r.X+16, r.Y+2)
		secs := strconv.Itoa((p.AkimboTime + TicksPerSecond - 1) / TicksPerSecond)
		drawSourceHUDText(screen, secs, 9, textClr, r.X+r.W-11, r.Y+2)
	case powerupBallisticShield:
		ebitenutil.DrawRect(screen, r.X+1, r.Y+1, r.W-2, r.H-2, color.NRGBA{R: 0x9c, G: 0xb4, B: 0xcc, A: 0xe0})
		drawBallisticShieldIcon(screen, r.X+8, r.Y+r.H/2, 6, 1)
		drawSourceHUDText(screen, "SHIELD", 9, textClr, r.X+16, r.Y+2)
	}
}

// drawAkimboBadgeIcon: two tiny crossed pistols.
func drawAkimboBadgeIcon(dst *ebiten.Image, cx, cy float64) {
	dark := color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xff}
	for _, s := range []float64{-1, 1} {
		x := cx + s*2
		vector.StrokeLine(dst, float32(x-s*4), float32(cy-3), float32(x+s*3), float32(cy-3), 2.4, dark, true)
		vector.StrokeLine(dst, float32(x-s*3), float32(cy-3), float32(x-s*4), float32(cy+3), 2, dark, true)
	}
}

// hudShowsAmmo: a starter bat never breaks, so it has no ammo count.
func hudShowsAmmo(p *Player) bool {
	if p == nil {
		return false
	}
	n := p.Weapon.Def.Number
	return !(isMeleeWeapon(n) && p.DefaultWeapon == n)
}
