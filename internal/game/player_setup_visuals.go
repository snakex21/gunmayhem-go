package game

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func drawSetupEditButton(dst *ebiten.Image, raster *SourceRaster, x, y, sx, alpha float64) {
	if raster != nil {
		drawSourceRaster(dst, raster, x, y, sx, 1, alpha)
	}
}

var (
	setupColorPaletteOnce  sync.Once
	setupColorPaletteImage *ebiten.Image
)

// sourceSetupColorPalette renders only Symbol856's root vector shape. The
// nested Symbol855 instance is the gray COLOR mask, which is animated
// separately by ActionScript and must not be baked into the palette itself.
func sourceSetupColorPalette() *ebiten.Image {
	setupColorPaletteOnce.Do(func() {
		doc, err := loadXFLVectorDoc("Symbol 856")
		if err != nil {
			return
		}
		img := ebiten.NewImage(50, 125)
		identity := xflMatrix{A: 1, D: 1}
		for li := len(doc.Timeline.Layers) - 1; li >= 0; li-- {
			frame, ok := activeVectorFrame(doc.Timeline.Layers[li].Frames, 0)
			if !ok {
				continue
			}
			for _, shape := range frame.Elements.Shapes {
				if err := drawSolidXFLShape(img, shape, identity, 1); err != nil {
					return
				}
			}
		}
		setupColorPaletteImage = img
	})
	return setupColorPaletteImage
}

func sourceColorTransform(originX, originY float64) ebiten.GeoM {
	// Exact Symbol856 matrix from Symbol1302:
	// [a=0 c=1.2277526855 tx=22.3; b=-0.4799957275 d=0 ty=349.55].
	var m ebiten.GeoM
	m.SetElement(0, 0, 0)
	m.SetElement(0, 1, 1.227752685546875)
	m.SetElement(0, 2, originX+22.3)
	m.SetElement(1, 0, -0.4799957275390625)
	m.SetElement(1, 1, 0)
	m.SetElement(1, 2, originY+349.55)
	return m
}

func sourceColorSelectorRect(cardX, cardY float64) Rect {
	m := sourceColorTransform(cardX, cardY)
	points := [4][2]float64{{0, 0}, {50, 0}, {0, 125}, {50, 125}}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, p := range points {
		x, y := m.Apply(p[0], p[1])
		minX = math.Min(minX, x)
		minY = math.Min(minY, y)
		maxX = math.Max(maxX, x)
		maxY = math.Max(maxY, y)
	}
	return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
}

func advanceSourceColorMask(alpha float64, hovered bool) float64 {
	if hovered {
		alpha -= 20 // Symbol856: masking._alpha -= 20
	} else {
		alpha += 10 // Symbol856: masking._alpha += 10
	}
	if alpha < 0 {
		return 0
	}
	if alpha > 100 {
		return 100
	}
	return alpha
}

func drawSetupColorSelector(dst *ebiten.Image, mask *SourceRaster, localY, cardAlpha, maskAlpha float64) {
	palette := sourceSetupColorPalette()
	if palette == nil {
		return
	}
	drawTransformed := func(img *ebiten.Image, bounds Rect, alpha float64) {
		if img == nil || alpha <= 0 {
			return
		}
		op := &ebiten.DrawImageOptions{}
		applySourceRenderQuality(op)
		op.GeoM.Translate(bounds.X, bounds.Y)
		op.GeoM.Concat(sourceColorTransform(0, localY))
		if alpha < 1 {
			op.ColorScale.ScaleAlpha(float32(alpha))
		}
		dst.DrawImage(img, op)
	}
	drawTransformed(palette, Rect{}, cardAlpha)
	if mask != nil && mask.Image != nil {
		drawTransformed(mask.Image, mask.Bounds, cardAlpha*maskAlpha/100)
	}
}

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
		return g.assets.ShirtFrame(number - 1)
	case customEditHat1, customEditHat2:
		return g.assets.HatFrame(number - 1)
	case customEditGun:
		// UI-only tight crops of the exact gameplay gun frame. This avoids the
		// large Flash registration canvas that made the six guns look microscopic.
		return g.assets.StarterGunFrames[number]
	}
	return nil
}

func drawSetupPerkThumb(dst *ebiten.Image, number int, r Rect, alpha float64) {
	labels := map[int][]string{
		1: {"NO", "PERKS"}, 2: {"SPEED+"}, 3: {"JUMP+"},
		4: {"MINI", "GUN"}, 5: {"NO", "RECOIL"}, 6: {"BLAST", "SHIELD"},
		7: {"EXTRA", "AMMO"}, 8: {"EXTRA", "NADES"}, 9: {"GUN", "SPAWN"},
	}
	colors := map[int]color.NRGBA{
		1: {R: 0xff, G: 0xff, B: 0xff, A: uint8(255 * alpha)},
		2: {R: 0x33, G: 0xcc, B: 0x00, A: uint8(255 * alpha)},
		3: {R: 0xff, G: 0x33, B: 0x99, A: uint8(255 * alpha)},
		4: {R: 0x66, G: 0x66, B: 0x66, A: uint8(255 * alpha)},
		5: {R: 0xff, G: 0xcc, B: 0x66, A: uint8(255 * alpha)},
		6: {R: 0x00, G: 0x66, B: 0xff, A: uint8(255 * alpha)},
		7: {R: 0xff, G: 0x99, B: 0x00, A: uint8(255 * alpha)},
		8: {R: 0xff, G: 0x33, B: 0x00, A: uint8(255 * alpha)},
		9: {R: 0x99, G: 0xff, B: 0x00, A: uint8(255 * alpha)},
	}
	lines := labels[number]
	if len(lines) == 0 {
		return
	}
	clr := colors[number]
	fontSize := 11.5
	for i, line := range lines {
		x := r.X + 5
		y := r.Y + r.H/2 - float64(len(lines))*7 + float64(i)*14
		drawSourceMenuText(dst, line, menuFontCondensedExtraBold, fontSize, clr, x, y)
	}
}

func setupSelectionVisualRect(editor int, source Rect, localY float64) Rect {
	r := source
	r.Y += localY
	// Symbol817's visual bounds include the rotated handgun display and are
	// wider than the actual grid cell. The source grid spacing is 61 px, so a
	// 55x55 tile is the correct visual cell while the parsed source bounds stay
	// in use for hit-testing.
	if editor == customEditGun {
		cx := r.X + r.W/2
		cy := r.Y + r.H/2
		r = Rect{X: cx - 27.5, Y: cy - 27.5, W: 55, H: 55}
	}
	return r
}

func (g *Game) drawSetupSelectionContents(dst *ebiten.Image, editor int, localY, alpha float64) {
	for _, item := range sourceCustomSelectionItems()[editor] {
		r := setupSelectionVisualRect(editor, item.Rect, localY)
		ebitenutil.DrawRect(dst, r.X, r.Y, r.W, r.H,
			color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: uint8(255 * alpha)})
		if editor == customEditPerk {
			drawSetupPerkThumb(dst, item.Number, r, alpha)
		} else {
			thumb := g.setupSelectionRaster(editor, item.Number)
			drawSetupThumbFit(dst, thumb, r, alpha)
		}
		if !g.customSelectionUnlocked(editor, item.Number) {
			ebitenutil.DrawRect(dst, r.X, r.Y, r.W, r.H,
				color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: uint8(150 * alpha)})
		}
	}

	// Symbol835 page switch for hats and Symbol1302.btn_back. These controls
	// used to be supplied only by the broken flattened frame, so draw them from
	// the same source button clip plus the static XFL labels.
	if editor == customEditHat1 || editor == customEditHat2 {
		y := localY + 671.7
		drawSetupEditButton(dst, g.assets.CustomEditButton, 27.25, y, 1, alpha)
		label := "NEXT PAGE"
		if editor == customEditHat2 {
			label = "PREVIOUS PAGE"
		}
		drawSourceMenuText(dst, label, menuFontCondensed, 15.9,
			color.NRGBA{A: uint8(255 * alpha)}, 50, y+3.5)
	}
	backY := localY + 759.45
	drawSetupEditButton(dst, g.assets.CustomEditButton, 27.5, backY, 1, alpha)
	drawSourceMenuText(dst, "BACK", menuFontCondensed, 15.9,
		color.NRGBA{A: uint8(255 * alpha)}, 83, backY+3.5)
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

// drawPlayerSetupCardMasked reproduces the 200x400 source mask used around
// Symbol1302/Symbol857. This matters when slide=2 moves the card upward to
// reveal Symbol835: without the mask, stale FFDec content leaks over neighbors.
func (g *Game) drawPlayerSetupCardMasked(screen *ebiten.Image, cfg CustomPlayerConfig, editor int, slot int, cardX, cardY, alpha, colorMaskAlpha float64, campaign bool) {
	clip := ebiten.NewImage(200, 400)
	localY := cardY - 100
	base := g.assets.CustomPlayerCard
	if campaign && g.assets.CampaignPlayerCard != nil {
		base = g.assets.CampaignPlayerCard
	}
	// The flattened card embeds Symbol835 on its startup (shirt) frame. Never
	// draw that snapshot while an editor is open, otherwise every other editor
	// inherits shirt thumbnails underneath its real contents.
	if editor == 0 && base != nil {
		drawSourceRaster(clip, base, 0, localY, 1, 1, alpha)
	}
	if editor >= 1 && editor <= 5 {
		g.drawSetupSelectionContents(clip, editor, localY, alpha)
	}

	if cfg.Type == 0 {
		drawSourceMenuText(clip, "SLOT EMPTY", menuFontCondensed, 15.95,
			color.NRGBA{R: 0xff, A: uint8(255 * alpha)}, 25.15, localY-386)
		drawSourceMenuText(clip, "HUMAN PLAYER", menuFontCondensed, 15.9,
			color.NRGBA{A: uint8(255 * alpha)}, 42, localY-252.55)
		if !campaign {
			drawSourceMenuText(clip, "AI PLAYER", menuFontCondensed, 15.9,
				color.NRGBA{A: uint8(255 * alpha)}, 42, localY-218.55)
		}
	} else if editor == 0 {
		// The FFDec card has the source preloader text baked into this area.
		// Clear the runtime preview zone first, then rebuild its live controls.
		ebitenutil.DrawRect(clip, 0, localY+70, 200, 190,
			color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: uint8(255 * alpha)})

		// Symbol806 is only the gray button background; the labels live on
		// separate Symbol1302 layers and therefore must be drawn AFTER it.
		drawSetupEditButton(clip, g.assets.CustomEditButton, 17.5, localY+93.0, 0.538726806640625, alpha)
		drawSetupEditButton(clip, g.assets.CustomEditButton, 17.5, localY+127.5, 0.538726806640625, alpha)
		drawSetupEditButton(clip, g.assets.CustomEditButton, 17.5, localY+161.7, 0.5387115478515625, alpha)
		drawSetupEditButton(clip, g.assets.CustomEditButton, 17.5, localY+220.75, 0.53875732421875, alpha)

		drawSourceMenuText(clip, "HAT", menuFontTwCen, 15.9,
			color.NRGBA{A: uint8(255 * alpha)}, 35.75, localY+96.25)
		drawSourceMenuText(clip, "SHIRT", menuFontTwCen, 15.9,
			color.NRGBA{A: uint8(255 * alpha)}, 34.25, localY+130.75)
		drawSourceMenuText(clip, "HANDGUN", menuFontTwCen, 15.9,
			color.NRGBA{A: uint8(255 * alpha)}, 21.05, localY+164.95)
		drawSourceMenuText(clip, "PERK", menuFontTwCen, 15.9,
			color.NRGBA{A: uint8(255 * alpha)}, 29.65, localY+224.5)
		drawSourceMenuText(clip, "NAME:", menuFontTwCen, 15.9,
			color.NRGBA{A: uint8(255 * alpha)}, 26.65, localY+271.2)

		g.drawSetupPlayerPreview(clip, cfg, alpha)
		drawSetupColorSelector(clip, g.assets.PlayerColorMask, localY, alpha, colorMaskAlpha)

		// DOMInputText in the source is Arial 16.
		drawSourceMenuText(clip, cfg.Name, menuFontArial, 16,
			color.NRGBA{A: uint8(255 * alpha)}, 32.5, localY+294.35)
		if !campaign || slot == 1 {
			drawSourceMenuText(clip, "CLEAR SLOT", menuFontCondensed, 15.9,
				color.NRGBA{A: uint8(255 * alpha)}, 42, localY+11.55)
		}
	}

	// Wipe the flattened Symbol1309/Symbol948 card underneath. Those exports
	// contain their startup ptype frame and would otherwise show through the
	// transparent parts of the real sliding card.
	ebitenutil.DrawRect(screen, cardX, 100, 200, 400,
		color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: uint8(255 * alpha)})
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(cardX, 100)
	screen.DrawImage(clip, op)
}
