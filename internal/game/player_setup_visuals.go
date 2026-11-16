package game

import (
	"github.com/hajimehoshi/ebiten/v2"
)

func drawSetupThumbFit(dst *ebiten.Image, raster *SourceRaster, r Rect, alpha float64) {
	if raster == nil || raster.Image == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	b := raster.Image.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return
	}
	pad := 4.0
	sx := (r.W - 2*pad) / float64(b.Dx())
	sy := (r.H - 2*pad) / float64(b.Dy())
	s := sx
	if sy < s {
		s = sy
	}
	if s <= 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(r.X+r.W/2, r.Y+r.H/2)
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	dst.DrawImage(raster.Image, op)
}

func (g *Game) setupSelectionRaster(editor, number int) *SourceRaster {
	switch editor {
	case customEditShirt:
		if source, ok := gm2ShirtSourceNumber(number); ok {
			return g.assets.GM2ShirtFrame(source - 1)
		}
		return g.assets.ShirtFrame(number - 1)
	case customEditHat1, customEditHat2:
		if source, ok := gm2HatSourceNumber(number); ok {
			return g.assets.GM2HatFrame(source - 1)
		}
		return g.assets.HatFrame(number - 1)
	case customEditGun:
		// UI-only tight crops of the exact gameplay gun frame. This avoids the
		// large Flash registration canvas that made the six guns look microscopic.
		if raster := g.assets.StarterGunFrames[number]; raster != nil {
			return raster
		}
		if proceduralWeapon(number) {
			return proceduralWeaponThumb(number)
		}
		return g.assets.WeaponFrame(number, 0)
	case customEditEyes:
		if source, ok := gm2EyeSourceNumber(number); ok {
			return g.assets.EyeFrame(source - 1)
		}
		return g.assets.Eyes
	}
	return nil
}

func (g *Game) drawSetupPlayerPreview(dst *ebiten.Image, cfg CustomPlayerConfig, alpha float64) {
	// Symbol1302.player is Symbol851 at (144.9,191.45), scale 1.34509277.
	// The gameplay renderer uses the same source body/head/shirt/hat timelines,
	// so reuse it here with the exact static setup pose instead of the broken
	// FFDec "LOADING" placeholder baked into the flattened card export.
	p := NewPlayer(1, g.arena)
	p.X = 144.9
	p.Y = 191.45
	p.Facing = 1
	p.PlayerScale = 1.3450927734375
	p.PlayerColor = cfg.Color
	p.ShirtNumber = cfg.Shirt
	p.HatNumber = cfg.Hat
	p.EyeNumber = cfg.Eyes
	// Preview the chosen gear (goggles, mask, vest) on the character.
	p.AbilityNumber = migrateLegacyGearPerk(cfg).Ability
	p.HeadFrame = 0
	p.LegFrame1 = 0
	p.LegFrame2 = 0
	p.VisualBodyY = -60
	p.VisualEyesY = -46
	p.VisualHand1X = -7.05
	p.VisualHand1Y = -24.6
	p.VisualHand1ChildX = 0
	p.VisualHand1ChildY = 0
	p.VisualHand2X = 0
	p.VisualHand2Y = -25
	p.VisualHand2ChildX = 0
	p.VisualHand2ChildY = 0
	p.Alpha = alpha
	p.ShieldAlpha = 0
	p.JetpackAlpha = 0
	p.Weapon = NewWeapon(cfg.Gun)
	// Symbol851 uses its own compact handgun display transform. Until that
	// nested Symbol816 transform is rendered separately, hide the gameplay-size
	// weapon rather than drawing a visibly oversized gun across the card.
	p.Weapon.Alpha = 0
	g.drawPlayer(dst, p)
}
