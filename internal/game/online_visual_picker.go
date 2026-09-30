package game

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	onlineVisualNone = iota
	onlineVisualCharacterColor
	onlineVisualCharacterShirt
	onlineVisualCharacterHat
	onlineVisualCharacterFace
	onlineVisualCharacterGun
	onlineVisualCharacterPerk
	onlineVisualCharacterTeam
)

const (
	onlineVisualOnlyWeapon = 100
	onlineVisualFixedMap   = 101
)

const (
	menuHitOnlineVisualFieldBase      = 9000
	menuHitOnlineVisualChoiceBase     = 9100
	menuHitOnlineVisualPrevPage       = 9250
	menuHitOnlineVisualNextPage       = 9251
	menuHitOnlineRuleMapChoiceBase    = 9300
	menuHitOnlineRuleWeaponChoiceBase = 9310
	menuHitOnlineRuleOnlyWeaponPicker = 9320
	menuHitOnlineRuleFixedMapPicker   = 9321
)

func onlineCharacterVisualFieldHit(field int) int {
	return menuHitOnlineVisualFieldBase + field
}

func decodeOnlineCharacterVisualFieldHit(hit int) (int, bool) {
	field := hit - menuHitOnlineVisualFieldBase
	return field, field >= 0 && field < onlineCharFieldCount
}

func onlineVisualChoiceHit(index int) int {
	return menuHitOnlineVisualChoiceBase + index
}

func decodeOnlineVisualChoiceHit(hit int) (int, bool) {
	index := hit - menuHitOnlineVisualChoiceBase
	return index, index >= 0 && index < 100
}

func onlineCharacterFieldRect(field int) Rect {
	col := field % 2
	row := field / 2
	return Rect{X: 405 + float64(col)*220, Y: 130 + float64(row)*95, W: 205, H: 84}
}

func (g *Game) onlineCharacterVisualHitAt(x, y float64) int {
	if onlineOverlayBackRect.Contains(x, y) {
		return menuHitOnlineOverlayBack
	}
	for field := 0; field < onlineCharFieldCount; field++ {
		if onlineCharacterFieldRect(field).Contains(x, y) {
			return onlineCharacterVisualFieldHit(field)
		}
	}
	return menuHitNone
}

func onlineCharacterPickerForField(field int) int {
	switch field {
	case onlineCharColor:
		return onlineVisualCharacterColor
	case onlineCharShirt:
		return onlineVisualCharacterShirt
	case onlineCharHat:
		return onlineVisualCharacterHat
	case onlineCharFace:
		return onlineVisualCharacterFace
	case onlineCharGun:
		return onlineVisualCharacterGun
	case onlineCharPerk:
		return onlineVisualCharacterPerk
	case onlineCharTeam:
		return onlineVisualCharacterTeam
	}
	return onlineVisualNone
}

func (g *Game) openOnlineCharacterVisualPicker(field int) {
	g.onlineVisualPicker = onlineCharacterPickerForField(field)
	g.onlineVisualPage = 0
}

func (g *Game) closeOnlineVisualPicker() {
	g.onlineVisualPicker = onlineVisualNone
	g.onlineVisualPage = 0
}

func onlineCharacterVisualChoiceCount(picker int) int {
	switch picker {
	case onlineVisualCharacterColor:
		return maxDevelopedColorValue()
	case onlineVisualCharacterShirt:
		return maxDevelopedShirtValue()
	case onlineVisualCharacterHat:
		return maxDevelopedHatValue()
	case onlineVisualCharacterFace:
		return maxDevelopedEyeValue()
	case onlineVisualCharacterGun:
		return len(starterHandgunNumbers())
	case onlineVisualCharacterPerk:
		return maxCustomPerk
	case onlineVisualCharacterTeam:
		return 2
	}
	return 0
}

func onlineCharacterVisualCols(picker int) int {
	switch picker {
	case onlineVisualCharacterColor:
		return 5
	case onlineVisualCharacterShirt, onlineVisualCharacterFace:
		return 4
	case onlineVisualCharacterHat:
		return 6
	case onlineVisualCharacterGun:
		return 7
	case onlineVisualCharacterPerk:
		return 3
	case onlineVisualCharacterTeam:
		return 2
	}
	return 4
}

func onlineCharacterVisualPageSize(picker int) int {
	cols := onlineCharacterVisualCols(picker)
	if cols <= 0 {
		return 1
	}
	// Four visual rows fit above the navigation buttons at y=520.
	return cols * 4
}

func onlineVisualCharacterChoiceRect(picker, visibleIndex int) Rect {
	cols := onlineCharacterVisualCols(picker)
	pageSize := onlineCharacterVisualPageSize(picker)
	if cols <= 0 || visibleIndex < 0 || visibleIndex >= pageSize {
		return Rect{}
	}
	gap := 10.0
	areaX, areaW := 70.0, 760.0
	cellW := (areaW - gap*float64(cols-1)) / float64(cols)
	cellH := 82.0
	return Rect{
		X: areaX + float64(visibleIndex%cols)*(cellW+gap),
		Y: 125 + float64(visibleIndex/cols)*(cellH+gap),
		W: cellW,
		H: cellH,
	}
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
	if g.onlineVisualPicker >= onlineVisualCharacterColor && g.onlineVisualPicker <= onlineVisualCharacterTeam {
		return onlineCharacterVisualPageSize(g.onlineVisualPicker)
	}
	if g.onlineVisualPicker == onlineVisualFixedMap {
		return 12
	}
	return 24
}

func (g *Game) onlineVisualMaxPage() int {
	if g.onlineVisualPicker >= onlineVisualCharacterColor && g.onlineVisualPicker <= onlineVisualCharacterTeam {
		count := onlineCharacterVisualChoiceCount(g.onlineVisualPicker)
		if count <= 0 {
			return 0
		}
		return (count - 1) / g.onlineVisualPageSize()
	}
	items := g.onlineVisualPagedItems()
	if len(items) == 0 {
		return 0
	}
	return (len(items) - 1) / g.onlineVisualPageSize()
}

func onlineVisualPagedRect(picker, visibleIndex int) Rect {
	if picker == onlineVisualFixedMap {
		col := visibleIndex % 3
		row := visibleIndex / 3
		return Rect{X: 65 + float64(col)*260, Y: 125 + float64(row)*82, W: 245, H: 72}
	}
	col := visibleIndex % 4
	row := visibleIndex / 4
	return Rect{X: 65 + float64(col)*195, Y: 125 + float64(row)*54, W: 180, H: 46}
}

func (g *Game) onlineVisualPickerHitAt(x, y float64) int {
	if onlineOverlayBackRect.Contains(x, y) {
		return menuHitOnlineOverlayBack
	}
	if g.onlineVisualPicker >= onlineVisualCharacterColor && g.onlineVisualPicker <= onlineVisualCharacterTeam {
		count := onlineCharacterVisualChoiceCount(g.onlineVisualPicker)
		pageSize := g.onlineVisualPageSize()
		start := g.onlineVisualPage * pageSize
		for i := 0; i < pageSize && start+i < count; i++ {
			if onlineVisualCharacterChoiceRect(g.onlineVisualPicker, i).Contains(x, y) {
				return onlineVisualChoiceHit(i)
			}
		}
		if g.onlineVisualPage > 0 && (Rect{X: 245, Y: 520, W: 150, H: 42}).Contains(x, y) {
			return menuHitOnlineVisualPrevPage
		}
		if g.onlineVisualPage < g.onlineVisualMaxPage() && (Rect{X: 505, Y: 520, W: 150, H: 42}).Contains(x, y) {
			return menuHitOnlineVisualNextPage
		}
		return menuHitNone
	}
	items := g.onlineVisualPagedItems()
	pageSize := g.onlineVisualPageSize()
	start := g.onlineVisualPage * pageSize
	for i := 0; i < pageSize && start+i < len(items); i++ {
		if onlineVisualPagedRect(g.onlineVisualPicker, i).Contains(x, y) {
			return onlineVisualChoiceHit(i)
		}
	}
	if g.onlineVisualPage > 0 && (Rect{X: 245, Y: 470, W: 150, H: 42}).Contains(x, y) {
		return menuHitOnlineVisualPrevPage
	}
	if g.onlineVisualPage < g.onlineVisualMaxPage() && (Rect{X: 505, Y: 470, W: 150, H: 42}).Contains(x, y) {
		return menuHitOnlineVisualNextPage
	}
	return menuHitNone
}

func (g *Game) applyOnlineVisualChoice(visibleIndex int) {
	id := g.onlineLocalPlayerID()
	if g.onlineVisualPicker >= onlineVisualCharacterColor && g.onlineVisualPicker <= onlineVisualCharacterTeam {
		if id < 1 || id > 4 {
			return
		}
		choiceCount := onlineCharacterVisualChoiceCount(g.onlineVisualPicker)
		absolute := g.onlineVisualPage*g.onlineVisualPageSize() + visibleIndex
		if visibleIndex < 0 || absolute < 0 || absolute >= choiceCount {
			return
		}
		choice := absolute + 1
		if g.onlineVisualPicker == onlineVisualCharacterGun {
			handguns := starterHandgunNumbers()
			choice = handguns[absolute]
		}
		cfg := &g.customPlayers[id-1]
		cfg.Type = 1
		switch g.onlineVisualPicker {
		case onlineVisualCharacterColor:
			cfg.Color = choice
		case onlineVisualCharacterShirt:
			cfg.Shirt = choice
		case onlineVisualCharacterHat:
			cfg.Hat = choice
		case onlineVisualCharacterFace:
			cfg.Eyes = choice
		case onlineVisualCharacterGun:
			cfg.Gun = choice
		case onlineVisualCharacterPerk:
			cfg.Perk = choice
		case onlineVisualCharacterTeam:
			cfg.Team = choice
		}
		g.sendOnlineLocalConfigIfChanged()
		g.closeOnlineVisualPicker()
		return
	}

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
	case onlineVisualCharacterColor:
		return "CHOOSE COLOR"
	case onlineVisualCharacterShirt:
		return "CHOOSE SHIRT"
	case onlineVisualCharacterHat:
		return "CHOOSE HAT"
	case onlineVisualCharacterFace:
		return "CHOOSE FACE"
	case onlineVisualCharacterGun:
		return "CHOOSE HANDGUN"
	case onlineVisualCharacterPerk:
		return "CHOOSE PERK"
	case onlineVisualCharacterTeam:
		return "CHOOSE TEAM"
	case onlineVisualOnlyWeapon:
		return "CHOOSE THE ONLY WEAPON"
	case onlineVisualFixedMap:
		return "CHOOSE FIXED MAP"
	}
	return "CHOOSE"
}

func drawOnlineSelectedFrame(dst *ebiten.Image, r Rect, selected bool) {
	border := color.NRGBA{R: 0x52, G: 0x5a, B: 0x66, A: 0xff}
	if selected {
		border = color.NRGBA{R: 0xff, G: 0x94, B: 0x00, A: 0xff}
	}
	ebitenutil.DrawRect(dst, r.X, r.Y, r.W, r.H, border)
	ebitenutil.DrawRect(dst, r.X+3, r.Y+3, r.W-6, r.H-6, color.NRGBA{R: 0x32, G: 0x38, B: 0x41, A: 0xff})
}

func drawOnlineColorSwatch(dst *ebiten.Image, number int, r Rect) {
	if number < 1 || number > maxDevelopedColorValue() || r.W <= 0 || r.H <= 0 {
		return
	}
	ebitenutil.DrawRect(dst, r.X, r.Y, r.W, r.H, developedPlayerColorSwatch(number))
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

func (g *Game) drawOnlineCharacterChoice(dst *ebiten.Image, picker, choice int, r Rect, selected bool) {
	drawOnlineSelectedFrame(dst, r, selected)
	inner := Rect{X: r.X + 8, Y: r.Y + 8, W: r.W - 16, H: r.H - 28}
	switch picker {
	case onlineVisualCharacterColor:
		drawOnlineColorSwatch(dst, choice, inner)
	case onlineVisualCharacterShirt:
		drawSetupThumbFit(dst, g.setupSelectionRaster(customEditShirt, choice), inner, 1)
	case onlineVisualCharacterHat:
		drawSetupThumbFit(dst, g.setupSelectionRaster(customEditHat1, choice), inner, 1)
	case onlineVisualCharacterFace:
		drawSetupThumbFit(dst, g.setupSelectionRaster(customEditEyes, choice), inner, 1)
	case onlineVisualCharacterGun:
		g.drawOnlineWeaponThumb(dst, choice, inner)
	case onlineVisualCharacterPerk:
		drawSetupPerkThumb(dst, choice, inner, 1)
	case onlineVisualCharacterTeam:
		label := "TEAM A"
		if choice == 2 {
			label = "TEAM B"
		}
		drawSourceMenuText(dst, label, menuFontTwCen, 25, color.NRGBA{A: 0xff}, r.X+25, r.Y+r.H/2-12)
	}
	drawSourceMenuText(dst, fmt.Sprintf("%d", choice), menuFontCondensedExtraBold, 11, color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}, r.X+7, r.Y+r.H-18)
}

func (g *Game) drawOnlineVisualPicker(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, 25, 45, 850, 540, color.NRGBA{R: 0x18, G: 0x1c, B: 0x22, A: 0xfb})
	drawSourceMenuText(screen, onlineVisualPickerTitle(g.onlineVisualPicker), menuFontTwCen, 31, color.NRGBA{A: 0xff}, 70, 66)
	drawSourceMenuText(screen, "Click the picture you want.", menuFontCondensed, 13, color.NRGBA{R: 0xbb, G: 0xbb, B: 0xbb, A: 0xff}, 70, 105)

	if g.onlineVisualPicker >= onlineVisualCharacterColor && g.onlineVisualPicker <= onlineVisualCharacterTeam {
		id := g.onlineLocalPlayerID()
		selected := 0
		if id >= 1 && id <= 4 {
			cfg := g.customPlayers[id-1]
			switch g.onlineVisualPicker {
			case onlineVisualCharacterColor:
				selected = cfg.Color
			case onlineVisualCharacterShirt:
				selected = cfg.Shirt
			case onlineVisualCharacterHat:
				selected = cfg.Hat
			case onlineVisualCharacterFace:
				selected = cfg.Eyes
			case onlineVisualCharacterGun:
				selected = cfg.Gun
			case onlineVisualCharacterPerk:
				selected = cfg.Perk
			case onlineVisualCharacterTeam:
				selected = cfg.Team
			}
		}
		count := onlineCharacterVisualChoiceCount(g.onlineVisualPicker)
		pageSize := g.onlineVisualPageSize()
		start := g.onlineVisualPage * pageSize
		for i := 0; i < pageSize && start+i < count; i++ {
			absolute := start + i
			choice := absolute + 1
			if g.onlineVisualPicker == onlineVisualCharacterGun {
				choice = starterHandgunNumbers()[absolute]
			}
			r := onlineVisualCharacterChoiceRect(g.onlineVisualPicker, i)
			g.drawOnlineCharacterChoice(screen, g.onlineVisualPicker, choice, r, choice == selected)
		}
		if g.onlineVisualPage > 0 {
			drawDevelopedMenuButton(screen, Rect{X: 245, Y: 520, W: 150, H: 42}, "PREVIOUS")
		}
		if g.onlineVisualPage < g.onlineVisualMaxPage() {
			drawDevelopedMenuButton(screen, Rect{X: 505, Y: 520, W: 150, H: 42}, "NEXT")
		}
		if g.onlineVisualMaxPage() > 0 {
			drawSourceMenuText(screen, fmt.Sprintf("PAGE %d / %d", g.onlineVisualPage+1, g.onlineVisualMaxPage()+1),
				menuFontCondensedExtraBold, 12, color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}, 70, 535)
		}
		drawDevelopedMenuButton(screen, onlineOverlayBackRect, "BACK")
		return
	}

	items := g.onlineVisualPagedItems()
	pageSize := g.onlineVisualPageSize()
	start := g.onlineVisualPage * pageSize
	for i := 0; i < pageSize && start+i < len(items); i++ {
		value := items[start+i]
		r := onlineVisualPagedRect(g.onlineVisualPicker, i)
		selected := false
		if g.onlineVisualPicker == onlineVisualOnlyWeapon {
			selected = value == g.onlineOnlyWeapon
		} else if g.onlineVisualPicker == onlineVisualFixedMap {
			selected = value == g.customMap && g.onlineMapRotation == onlineMapFixed
		}
		drawOnlineSelectedFrame(screen, r, selected)
		if g.onlineVisualPicker == onlineVisualOnlyWeapon {
			g.drawOnlineWeaponThumb(screen, value, Rect{X: r.X + 5, Y: r.Y + 5, W: 62, H: r.H - 10})
			name := weaponDisplayName(value)
			if len(name) > 18 {
				name = strings.TrimSpace(name[:18])
			}
			drawSourceMenuText(screen, name, menuFontCondensed, 10.5, color.NRGBA{A: 0xff}, r.X+72, r.Y+8)
			drawSourceMenuText(screen, fmt.Sprintf("#%d", value), menuFontCondensedExtraBold, 10, color.NRGBA{R: 0xbb, G: 0xbb, B: 0xbb, A: 0xff}, r.X+72, r.Y+25)
		} else {
			g.drawOnlineMapThumb(screen, value, Rect{X: r.X + 4, Y: r.Y + 4, W: 100, H: r.H - 8})
			name := onlineMapNumberLabel(value)
			if len(name) > 21 {
				name = strings.TrimSpace(name[:21])
			}
			drawSourceMenuText(screen, name, menuFontCondensed, 11.5, color.NRGBA{A: 0xff}, r.X+112, r.Y+17)
		}
	}

	if g.onlineVisualPage > 0 {
		drawDevelopedMenuButton(screen, Rect{X: 245, Y: 470, W: 150, H: 42}, "PREV PAGE")
	}
	if g.onlineVisualPage < g.onlineVisualMaxPage() {
		drawDevelopedMenuButton(screen, Rect{X: 505, Y: 470, W: 150, H: 42}, "NEXT PAGE")
	}
	drawSourceMenuText(screen, fmt.Sprintf("PAGE %d/%d", g.onlineVisualPage+1, g.onlineVisualMaxPage()+1), menuFontTwCen, 14, color.NRGBA{A: 0xff}, 415, 482)
	drawDevelopedMenuButton(screen, onlineOverlayBackRect, "BACK")
}

func (g *Game) drawOnlineCharacterFieldTile(dst *ebiten.Image, field int, cfg CustomPlayerConfig) {
	r := onlineCharacterFieldRect(field)
	drawOnlineSelectedFrame(dst, r, false)
	label := ""
	value := ""
	thumb := Rect{X: r.X + 105, Y: r.Y + 9, W: 88, H: r.H - 18}
	switch field {
	case onlineCharColor:
		label = "COLOR"
		value = fmt.Sprintf("%d", cfg.Color)
		drawOnlineColorSwatch(dst, cfg.Color, thumb)
	case onlineCharShirt:
		label = "SHIRT"
		value = fmt.Sprintf("%d", cfg.Shirt)
		drawSetupThumbFit(dst, g.setupSelectionRaster(customEditShirt, cfg.Shirt), thumb, 1)
	case onlineCharHat:
		label = "HAT"
		value = fmt.Sprintf("%d", cfg.Hat)
		drawSetupThumbFit(dst, g.setupSelectionRaster(customEditHat1, cfg.Hat), thumb, 1)
	case onlineCharFace:
		label = "FACE"
		value = fmt.Sprintf("%d", cfg.Eyes)
		drawSetupThumbFit(dst, g.setupSelectionRaster(customEditEyes, cfg.Eyes), thumb, 1)
	case onlineCharGun:
		label = "HANDGUN"
		value = weaponDisplayName(cfg.Gun)
		g.drawOnlineWeaponThumb(dst, cfg.Gun, thumb)
	case onlineCharPerk:
		label = "PERK"
		value = onlinePerkName(cfg.Perk)
		drawSetupPerkThumb(dst, cfg.Perk, thumb, 1)
	case onlineCharTeam:
		label = "TEAM"
		if cfg.Team == 2 {
			value = "B"
		} else {
			value = "A"
		}
		drawSourceMenuText(dst, "TEAM "+value, menuFontTwCen, 20, color.NRGBA{A: 0xff}, thumb.X+5, thumb.Y+22)
	}
	if len(value) > 14 {
		value = strings.TrimSpace(value[:14])
	}
	drawSourceMenuText(dst, label, menuFontCondensedExtraBold, 12.5, color.NRGBA{R: 0xff, G: 0x99, A: 0xff}, r.X+10, r.Y+12)
	valueFont, valueSize := menuFontTwCen, 14.0
	if field == onlineCharPerk && cfg.Perk == perkKnockbackResistance {
		// The exported regular face lacks digit 3; the existing extra-bold
		// face contains this full label and also fits the narrow text column.
		valueFont, valueSize = menuFontCondensedExtraBold, 11.0
	}
	drawSourceMenuText(dst, value, valueFont, valueSize, color.NRGBA{A: 0xff}, r.X+10, r.Y+40)
}

func (g *Game) drawOnlineCharacterMini(dst *ebiten.Image, r Rect, cfg CustomPlayerConfig) {
	preview := ebiten.NewImage(200, 230)
	g.drawSetupPlayerPreview(preview, cfg, 1)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.34, 0.34)
	op.GeoM.Translate(r.X+r.W-75, r.Y+5)
	dst.DrawImage(preview, op)
}

func (g *Game) drawOnlineMapModeTile(dst *ebiten.Image, mode int, r Rect) {
	selected := g.onlineMapRotation == mode
	drawOnlineSelectedFrame(dst, r, selected)
	thumb := Rect{X: r.X + 5, Y: r.Y + 5, W: r.W - 10, H: 38}
	switch mode {
	case onlineMapFixed:
		g.drawOnlineMapThumb(dst, g.customMap, thumb)
	case onlineMapRandomGM1:
		g.drawOnlineMapThumb(dst, 1, thumb)
	case onlineMapRandomGM2:
		g.drawOnlineMapThumb(dst, gm2MapID(1), thumb)
	case onlineMapRandomAll:
		left := thumb
		left.W /= 2
		right := left
		right.X += left.W
		g.drawOnlineMapThumb(dst, 1, left)
		g.drawOnlineMapThumb(dst, gm2MapID(1), right)
	}
	drawSourceMenuText(dst, onlineMapRotationLabel(mode), menuFontCondensedExtraBold, 11.5, color.NRGBA{A: 0xff}, r.X+7, r.Y+49)
}

func (g *Game) drawOnlineWeaponModeTile(dst *ebiten.Image, mode int, r Rect) {
	selected := g.onlineWeaponMode == mode
	drawOnlineSelectedFrame(dst, r, selected)
	inner := Rect{X: r.X + 8, Y: r.Y + 5, W: 72, H: r.H - 10}
	switch mode {
	case onlineWeaponsOnlyOne:
		g.drawOnlineWeaponThumb(dst, g.onlineOnlyWeapon, inner)
	case onlineWeaponsBlocklist:
		g.drawOnlineWeaponThumb(dst, 10, inner)
		g.drawOnlineWeaponThumb(dst, 67, Rect{X: inner.X + 42, Y: inner.Y, W: inner.W, H: inner.H})
	default:
		g.drawOnlineWeaponThumb(dst, 1, inner)
		g.drawOnlineWeaponThumb(dst, 10, Rect{X: inner.X + 42, Y: inner.Y, W: inner.W, H: inner.H})
	}
	drawSourceMenuText(dst, onlineWeaponModeLabel(mode), menuFontCondensedExtraBold, 11.5, color.NRGBA{A: 0xff}, r.X+90, r.Y+25)
}
