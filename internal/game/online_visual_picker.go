package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	onlineVisualNone       = 0
	onlineVisualOnlyWeapon = 100
	onlineVisualFixedMap   = 101
)

const (
	menuHitOnlineVisualChoiceBase     = 9100
	menuHitOnlineVisualPrevPage       = 9250
	menuHitOnlineVisualNextPage       = 9251
	menuHitOnlineRuleMapChoiceBase    = 9300
	menuHitOnlineRuleWeaponChoiceBase = 9310
	menuHitOnlineRuleOnlyWeaponPicker = 9320
	menuHitOnlineRuleFixedMapPicker   = 9321
)

func onlineVisualChoiceHit(index int) int {
	return menuHitOnlineVisualChoiceBase + index
}

func decodeOnlineVisualChoiceHit(hit int) (int, bool) {
	index := hit - menuHitOnlineVisualChoiceBase
	return index, index >= 0 && index < 100
}

func (g *Game) closeOnlineVisualPicker() {
	g.onlineVisualPicker = onlineVisualNone
	g.onlineVisualPage = 0
}

func onlineFixedMapNumbers() []int {
	out := make([]int, 0, 21)
	for n := 1; n <= 12; n++ {
		out = append(out, n)
	}
	for n := 1; n <= 9; n++ {
		out = append(out, gm2MapID(n))
	}
	return out
}

func (g *Game) onlineVisualPagedItems() []int {
	switch g.onlineVisualPicker {
	case onlineVisualOnlyWeapon:
		return onlineAllWeaponNumbers()
	case onlineVisualFixedMap:
		return onlineFixedMapNumbers()
	}
	return nil
}

func (g *Game) onlineVisualPageSize() int {
	if g.onlineVisualPicker == onlineVisualFixedMap {
		return 12
	}
	return 24
}

func (g *Game) onlineVisualMaxPage() int {
	items := g.onlineVisualPagedItems()
	if len(items) == 0 {
		return 0
	}
	return (len(items) - 1) / g.onlineVisualPageSize()
}

func (g *Game) applyOnlineVisualChoice(visibleIndex int) {
	items := g.onlineVisualPagedItems()
	absolute := g.onlineVisualPage*g.onlineVisualPageSize() + visibleIndex
	if absolute < 0 || absolute >= len(items) {
		return
	}
	choice := items[absolute]
	switch g.onlineVisualPicker {
	case onlineVisualOnlyWeapon:
		g.onlineOnlyWeapon = choice
		g.closeOnlineVisualPicker()
	case onlineVisualFixedMap:
		g.customMap = choice
		g.customMapSetGM2 = isGM2MapID(choice)
		g.onlineMapRotation = onlineMapFixed
		g.closeOnlineVisualPicker()
	}
}

func onlineVisualPickerTitle(picker int) string {
	switch picker {
	case onlineVisualOnlyWeapon:
		return "CHOOSE THE ONLY WEAPON"
	case onlineVisualFixedMap:
		return "CHOOSE FIXED MAP"
	}
	return "CHOOSE"
}

func (g *Game) drawOnlineWeaponThumb(dst *ebiten.Image, number int, r Rect) {
	var raster *SourceRaster
	if number >= 1 && number <= 6 {
		raster = g.assets.StarterGunFrames[number]
	}
	if raster == nil {
		raster = g.assets.WeaponFrame(number, 0)
	}
	drawSetupThumbFit(dst, raster, r, 1)
}

func drawMapThumbPlaceholder(dst *ebiten.Image, r Rect) {
	ebitenutil.DrawRect(dst, r.X, r.Y, r.W, r.H, color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xff})
	drawSourceMenuText(dst, "?", menuFontCondensedExtraBold, r.H*0.65,
		color.NRGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff}, r.X+r.W*0.42, r.Y+r.H*0.05)
}

func drawCachedMapThumb(dst, thumb *ebiten.Image, r Rect) {
	if dst == nil || thumb == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	w, h := float64(thumb.Bounds().Dx()), float64(thumb.Bounds().Dy())
	if w <= 0 || h <= 0 {
		return
	}
	// Draw only the already-clipped cache. The small playlist row is deliberately
	// filled edge-to-edge so the preview stays readable; most importantly, no
	// GM2 SourceRaster can escape the tile anymore.
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Scale(r.W/w, r.H/h)
	op.GeoM.Translate(r.X, r.Y)
	dst.DrawImage(thumb, op)
}

func (g *Game) ensureOnlineMapThumb(mapNumber int) *ebiten.Image {
	if g == nil || g.assets == nil || mapNumber <= 0 {
		return nil
	}
	if g.customMapThumbCache == nil {
		g.customMapThumbCache = make(map[int]*ebiten.Image)
	}
	if thumb := g.customMapThumbCache[mapNumber]; thumb != nil {
		return thumb
	}

	thumb := ebiten.NewImage(300, 180)
	thumb.Fill(color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xff})
	if preview := g.assets.MapPreviewImages[mapNumber]; preview != nil {
		drawSetupThumbFit(thumb, preview, Rect{X: 0, Y: 0, W: 300, H: 180}, 1)
		g.customMapThumbCache[mapNumber] = thumb
		return thumb
	}

	// Fallback only for maps whose source preview media is unavailable.
	g.assets.EnsureScene(mapNumber)
	const s = 180.0 / 600.0
	ox := (300.0 - 900.0*s) / 2
	drawSourceRaster(thumb, g.assets.SceneBack[mapNumber], ox, 0, s, s, 1)
	drawSourceRaster(thumb, g.assets.SceneMid[mapNumber], ox, 0, s, s, 1)
	drawSourceRaster(thumb, g.assets.SceneFront[mapNumber], ox+51.1*s, 192.6*s, s, s, 1)
	g.customMapThumbCache[mapNumber] = thumb
	return thumb
}

func (g *Game) drawOnlineMapThumb(dst *ebiten.Image, mapNumber int, r Rect) {
	if mapNumber <= 0 {
		drawMapThumbPlaceholder(dst, r)
		return
	}
	drawCachedMapThumb(dst, g.ensureOnlineMapThumb(mapNumber), r)
}

// The Custom Game playlist displays up to ten map tiles simultaneously. It
// must never synchronously decode all nine GM2 scenes in one frame. Missing
// thumbnails therefore draw as placeholders and are filled progressively by
// updateCustomV2MapThumbPreload().
func (g *Game) drawOnlineMapThumbIfReady(dst *ebiten.Image, mapNumber int, r Rect) {
	if mapNumber <= 0 {
		drawMapThumbPlaceholder(dst, r)
		return
	}
	// GM1 and GM2 both have real source preview bitmaps. Draw those directly in
	// the playlist so switching map sets is immediate and never waits for scene
	// reconstruction or the progressive thumbnail cache.
	if g.assets != nil {
		if preview := g.assets.MapPreviewImages[mapNumber]; preview != nil {
			drawSetupThumbFit(dst, preview, r, 1)
			return
		}
	}
	if g.customMapThumbCache != nil {
		if thumb := g.customMapThumbCache[mapNumber]; thumb != nil {
			drawCachedMapThumb(dst, thumb, r)
			return
		}
	}
	drawMapThumbPlaceholder(dst, r)
}

func onlineMapNumberLabel(mapNumber int) string {
	if isGM2MapID(mapNumber) {
		if name := gm2MapDisplayNames[gm2SourceMapNumber(mapNumber)]; name != "" {
			return name
		}
	}
	if name := sourceMapDisplayNames[mapNumber]; name != "" {
		return name
	}
	return fmt.Sprintf("MAP %d", mapNumber)
}
