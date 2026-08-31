package game

import (
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// The developed library keeps the original Gun Library presentation: four
// weapon-family panels, handguns at the bottom and the original stat/test area.
// GM2 additions are folded into those same four families instead of living on
// a separate GM1/GM2/ALL page.
var developedGunLibraryCategories = []struct {
	guns      []int
	columns   [3]float64
	firstY    float64
	rowStep   float64
	cardScale float64
}{
	// Three compact columns keep every family entirely inside the original
	// upper panels. With at most 20 weapons per family this needs only seven
	// rows, so nothing can spill into SPECIAL / HANDGUNS / the stat card.
	{[]int{12, 19, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 67, 68, 69, 70, 71}, [3]float64{55, 105, 155}, 214, 31.5, 0.68},
	{[]int{11, 14, 16, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 77, 78, 79, 80, 81}, [3]float64{242, 322, 402}, 214, 31.5, 0.68},
	{[]int{13, 15, 18, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 82, 83, 84, 85, 86}, [3]float64{487, 552, 617}, 214, 31.5, 0.68},
	{[]int{10, 17, 20, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 72, 73, 74, 75, 76}, [3]float64{687, 757, 827}, 214, 31.5, 0.68},
}

const developedCustomGunLibraryPageSize = 18

var (
	gunLibraryPreviousPageRect = Rect{X: 285, Y: 426, W: 112, H: 28}
	gunLibraryNextPageRect     = Rect{X: 503, Y: 426, W: 112, H: 28}
)

var organizedGunLibraryTabs = []struct {
	label      string
	category   DevelopedWeaponCategory
	throwables bool
}{
	{"ALL", 0, false},
	{"PISTOLS", DevelopedCategoryPistol, false},
	{"SMG", DevelopedCategorySMG, false},
	{"RIFLES", DevelopedCategoryRifle, false},
	{"LMG", DevelopedCategoryLMG, false},
	{"SNIPERS", DevelopedCategorySniper, false},
	{"SHOTGUNS", DevelopedCategoryShotgun, false},
	{"LAUNCHERS", DevelopedCategoryLauncher, false},
	{"SPECIAL", DevelopedCategorySpecial, false},
	{"THROWABLES", 0, true},
}

const organizedGunLibraryPageSize = 20

func sourceWeaponCategory(number int) DevelopedWeaponCategory {
	switch {
	case number >= 1 && number <= 6:
		return DevelopedCategoryPistol
	case number == 12 || number == 19 || (number >= 21 && number <= 32) || (number >= 67 && number <= 71):
		return DevelopedCategorySMG
	case number == 55 || number == 57 || (number >= 64 && number <= 66):
		return DevelopedCategoryLMG
	case number == 11 || number == 14 || number == 16 || (number >= 33 && number <= 43) || (number >= 77 && number <= 81):
		return DevelopedCategorySniper
	case number == 13 || number == 15 || number == 18 || (number >= 44 && number <= 54) || (number >= 82 && number <= 86):
		return DevelopedCategoryShotgun
	default:
		return DevelopedCategoryRifle
	}
}

func organizedGunLibraryNumbers(category DevelopedWeaponCategory) []int {
	out := make([]int, 0, 186)
	appendIf := func(number int, weaponCategory DevelopedWeaponCategory) {
		if category == 0 || category == weaponCategory {
			out = append(out, number)
		}
	}
	for number := 1; number <= 6; number++ {
		appendIf(number, DevelopedCategoryPistol)
	}
	for number := 10; number <= 86; number++ {
		appendIf(number, sourceWeaponCategory(number))
	}
	for _, number := range allDevelopedWeaponNumbers() {
		spec := developedWeaponSpecs[number]
		appendIf(number, spec.Category)
	}
	return out
}

func organizedGunLibraryPageCount(category DevelopedWeaponCategory) int {
	count := len(organizedGunLibraryNumbers(category))
	if count == 0 {
		return 1
	}
	return (count + organizedGunLibraryPageSize - 1) / organizedGunLibraryPageSize
}

func organizedGunLibraryButtons(category DevelopedWeaponCategory, page int) []gunLibraryButton {
	weapons := organizedGunLibraryNumbers(category)
	pages := organizedGunLibraryPageCount(category)
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	start := page * organizedGunLibraryPageSize
	end := start + organizedGunLibraryPageSize
	if end > len(weapons) {
		end = len(weapons)
	}
	out := make([]gunLibraryButton, 0, end-start)
	for i, number := range weapons[start:end] {
		out = append(out, gunLibraryButton{
			Gun: number, X: 96 + float64(i%5)*177, Y: 156 + float64(i/5)*66,
			Scale: 1, W: 164, H: 58,
		})
	}
	return out
}

func organizedGunLibraryCategoryAt(x, y float64) (int, bool) {
	for i := range organizedGunLibraryTabs {
		r := organizedGunLibraryTabRect(i)
		if r.Contains(x, y) {
			return i, true
		}
	}
	return 0, false
}

func organizedGunLibraryTabRect(index int) Rect {
	return Rect{X: 18 + float64(index)*86, Y: 75, W: 81, H: 31}
}

func (g *Game) organizedGunLibraryThrowableTab() bool {
	return g.gunLibraryCategory >= 0 && g.gunLibraryCategory < len(organizedGunLibraryTabs) &&
		organizedGunLibraryTabs[g.gunLibraryCategory].throwables
}

func organizedThrowableLibraryCardRect(kind int) Rect {
	kind = normalizedThrowableType(kind)
	return Rect{
		X: 30 + float64(kind%3)*290,
		Y: 128 + float64(kind/3)*132,
		W: 260,
		H: 112,
	}
}

func organizedGunLibraryThrowableAt(x, y float64) (int, bool) {
	for kind := 0; kind < throwableTypeCount; kind++ {
		if organizedThrowableLibraryCardRect(kind).Contains(x, y) {
			return kind, true
		}
	}
	return 0, false
}

func (g *Game) organizedGunLibraryHitAt(x, y float64) int {
	if g.organizedGunLibraryThrowableTab() {
		return 0
	}
	category := DevelopedWeaponCategory(0)
	if g.gunLibraryCategory >= 0 && g.gunLibraryCategory < len(organizedGunLibraryTabs) {
		category = organizedGunLibraryTabs[g.gunLibraryCategory].category
	}
	for _, button := range organizedGunLibraryButtons(category, g.gunLibraryPage) {
		if button.contains(x, y) && g.gunUnlocked(button.Gun) {
			return button.Gun
		}
	}
	return 0
}

func developedGunLibraryButtons() []gunLibraryButton {
	out := make([]gunLibraryButton, 0, 83)
	// Preserve the exact source positions of the six starter handguns.
	for _, b := range sourceGunLibraryButtons() {
		if b.Gun >= 1 && b.Gun <= 6 {
			out = append(out, b)
		}
	}
	for _, category := range developedGunLibraryCategories {
		const rowsPerColumn = 7
		for i, gun := range category.guns {
			col := i / rowsPerColumn
			row := i % rowsPerColumn
			if col >= len(category.columns) {
				continue
			}
			out = append(out, gunLibraryButton{
				Gun:   gun,
				X:     category.columns[col],
				Y:     category.firstY + float64(row)*category.rowStep,
				Scale: category.cardScale,
			})
		}
	}
	return out
}

func developedGunLibraryPageCount() int {
	customCount := len(allDevelopedWeaponNumbers())
	return 1 + (customCount+developedCustomGunLibraryPageSize-1)/developedCustomGunLibraryPageSize
}

func developedGunLibraryButtonsForPage(page int) []gunLibraryButton {
	if page <= 0 {
		return developedGunLibraryButtons()
	}
	weapons := allDevelopedWeaponNumbers()
	start := (page - 1) * developedCustomGunLibraryPageSize
	if start >= len(weapons) {
		return nil
	}
	end := start + developedCustomGunLibraryPageSize
	if end > len(weapons) {
		end = len(weapons)
	}
	out := make([]gunLibraryButton, 0, end-start)
	for i, gun := range weapons[start:end] {
		out = append(out, gunLibraryButton{
			Gun:   gun,
			X:     95 + float64(i%6)*142,
			Y:     235 + float64(i/6)*78,
			Scale: 1,
			W:     128,
			H:     64,
		})
	}
	return out
}

func (g *Game) changeGunLibraryPage(delta int) {
	if g.organizedGunLibraryThrowableTab() {
		g.gunLibraryPage = 0
		g.gunLibraryPressed = 0
		return
	}
	category := DevelopedWeaponCategory(0)
	if g.gunLibraryCategory >= 0 && g.gunLibraryCategory < len(organizedGunLibraryTabs) {
		category = organizedGunLibraryTabs[g.gunLibraryCategory].category
	}
	count := organizedGunLibraryPageCount(category)
	if count <= 1 {
		g.gunLibraryPage = 0
		return
	}
	g.gunLibraryPage = (g.gunLibraryPage + delta) % count
	if g.gunLibraryPage < 0 {
		g.gunLibraryPage += count
	}
	g.gunLibraryPressed = 0
}

func (g *Game) developedGunLibraryHitAt(x, y float64) int {
	for _, b := range developedGunLibraryButtonsForPage(g.gunLibraryPage) {
		if b.contains(x, y) && g.gunUnlocked(b.Gun) {
			return b.Gun
		}
	}
	return 0
}

func (g *Game) drawDevelopedGunLibrary(screen *ebiten.Image) {
	background := color.NRGBA{R: 0x20, G: 0x24, B: 0x2b, A: 0xff}
	panel := color.NRGBA{R: 0x33, G: 0x39, B: 0x43, A: 0xff}
	orange := color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
	ebitenutil.DrawRect(screen, 0, 0, 900, 600, background)
	ebitenutil.DrawRect(screen, 18, 16, 864, 48, color.NRGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff})
	drawSourceMenuText(screen, "GUN LIBRARY", menuFontTwCen, 31, color.Black, 34, 24)
	drawSourceMenuText(screen, "BACK", menuFontCondensedExtraBold, 17, color.Black, 728, 31)

	if g.gunLibraryCategory < 0 || g.gunLibraryCategory >= len(organizedGunLibraryTabs) {
		g.gunLibraryCategory = 0
	}
	for i, tab := range organizedGunLibraryTabs {
		r := organizedGunLibraryTabRect(i)
		fill := panel
		textColor := color.White
		if i == g.gunLibraryCategory {
			fill, textColor = orange, color.Black
		}
		ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, fill)
		fontSize := 11.0
		if tab.throwables {
			fontSize = 9.5
		}
		drawSourceMenuText(screen, tab.label, menuFontCondensedExtraBold, fontSize, textColor, r.X+6, r.Y+9)
	}

	if g.organizedGunLibraryThrowableTab() {
		g.gunLibraryPage = 0
		g.drawOrganizedThrowableLibrary(screen)
		return
	}

	category := organizedGunLibraryTabs[g.gunLibraryCategory].category
	pages := organizedGunLibraryPageCount(category)
	if g.gunLibraryPage >= pages {
		g.gunLibraryPage = pages - 1
	}
	for _, b := range organizedGunLibraryButtons(category, g.gunLibraryPage) {
		g.drawOrganizedGunLibraryCard(screen, b)
	}
	g.drawOrganizedGunLibraryFooter(screen, category, pages)
}

func (g *Game) drawOrganizedThrowableLibrary(screen *ebiten.Image) {
	panel := color.NRGBA{R: 0x33, G: 0x39, B: 0x43, A: 0xff}
	orange := color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
	ensureThrowableVisuals()
	for kind := 0; kind < throwableTypeCount; kind++ {
		r := organizedThrowableLibraryCardRect(kind)
		border := color.NRGBA{R: 0x16, G: 0x18, B: 0x1d, A: 0xff}
		if kind == normalizedThrowableType(g.gunLibraryThrowableSelected) {
			border = orange
		}
		ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, border)
		ebitenutil.DrawRect(screen, r.X+3, r.Y+3, r.W-6, r.H-6, color.NRGBA{R: 0xeb, G: 0xeb, B: 0xeb, A: 0xff})
		if kind == ThrowableDynamite {
			drawGunLibraryCleanThumb(screen, g.assets.Grenade, r.X+52, r.Y+54, 76, 70, 1, 0)
		} else if img := throwableImages[kind]; img != nil {
			w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
			scale := math.Min(72/w, 70/h)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-w/2, -h/2)
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(r.X+52, r.Y+54)
			screen.DrawImage(img, op)
		}
		profile := throwableProfiles[kind]
		drawSourceMenuText(screen, profile.Name, menuFontTwCen, 16, color.Black, r.X+95, r.Y+17)
		drawSourceMenuText(screen, "FUSE  "+strconv.Itoa(profile.Fuse), menuFontCondensedExtraBold, 11, color.Black, r.X+95, r.Y+49)
		drawSourceMenuText(screen, "RADIUS  "+strconv.Itoa(int(profile.Radius)), menuFontCondensedExtraBold, 11, color.Black, r.X+95, r.Y+67)
		push := int(math.Round(profile.HorizontalPush))
		label := "PUSH  " + strconv.Itoa(push)
		if kind == ThrowableMolotov {
			label = "BURNING AREA"
		}
		drawSourceMenuText(screen, label, menuFontCondensedExtraBold, 11, color.Black, r.X+95, r.Y+85)
	}

	ebitenutil.DrawRect(screen, 18, 408, 864, 174, panel)
	kind := normalizedThrowableType(g.gunLibraryThrowableSelected)
	profile := throwableProfiles[kind]
	drawSourceMenuText(screen, "SELECTED THROWABLE", menuFontCondensedExtraBold, 13, color.White, 39, 430)
	drawSourceMenuText(screen, profile.Name, menuFontTwCen, 25, color.White, 39, 454)
	drawSourceMenuText(screen, "Campaign: use the arrows on the player card", menuFontCondensed, 15, color.White, 39, 496)
	drawSourceMenuText(screen, "Test mode starts with 30 selected throwables", menuFontCondensed, 15, color.White, 39, 521)
	test := Rect{X: 756, Y: 514, W: 108, H: 44}
	ebitenutil.DrawRect(screen, test.X, test.Y, test.W, test.H, orange)
	drawSourceMenuText(screen, "TEST", menuFontCondensedExtraBold, 19, color.Black, 787, 525)
}

func (g *Game) drawOrganizedGunLibraryCard(screen *ebiten.Image, b gunLibraryButton) {
	x, y := b.X-b.W/2, b.Y-b.H/2
	border := color.NRGBA{R: 0x16, G: 0x18, B: 0x1d, A: 0xff}
	if g.gunLibrarySelected == b.Gun {
		border = color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
	}
	ebitenutil.DrawRect(screen, x, y, b.W, b.H, border)
	ebitenutil.DrawRect(screen, x+3, y+3, b.W-6, b.H-6, color.NRGBA{R: 0xeb, G: 0xeb, B: 0xeb, A: 0xff})
	drawGunLibraryCleanThumb(screen, g.gunLibraryRaster(b.Gun), b.X, b.Y-8, b.W-18, 31, 1, 0)
	name := weaponDisplayName(b.Gun)
	runes := []rune(name)
	if len(runes) > 24 {
		name = string(runes[:23]) + "…"
	}
	drawSourceMenuText(screen, name, menuFontCondensedExtraBold, 10.5, color.Black, x+7, y+b.H-16)
	if !g.gunUnlocked(b.Gun) {
		ebitenutil.DrawRect(screen, x+3, y+3, b.W-6, b.H-6, color.NRGBA{A: 0x9a})
		drawSourceMenuText(screen, "LOCKED", menuFontCondensedExtraBold, 12, color.White, x+56, y+21)
	}
}

func (g *Game) drawOrganizedGunLibraryFooter(screen *ebiten.Image, category DevelopedWeaponCategory, pages int) {
	panel := color.NRGBA{R: 0x33, G: 0x39, B: 0x43, A: 0xff}
	orange := color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
	for _, r := range []Rect{gunLibraryPreviousPageRect, gunLibraryNextPageRect} {
		ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, orange)
	}
	drawSourceMenuText(screen, "< PREV", menuFontCondensedExtraBold, 14, color.Black, 307, 433)
	drawSourceMenuText(screen, "NEXT >", menuFontCondensedExtraBold, 14, color.Black, 529, 433)
	drawSourceMenuText(screen, "PAGE "+strconv.Itoa(g.gunLibraryPage+1)+" / "+strconv.Itoa(pages), menuFontCondensedExtraBold, 13, color.White, 415, 433)

	ebitenutil.DrawRect(screen, 18, 466, 864, 116, panel)
	if g.gunLibrarySelected == 0 {
		drawSourceMenuText(screen, "SELECT A WEAPON", menuFontTwCen, 23, color.White, 39, 491)
		return
	}
	def, ok := WeaponByNumber(g.gunLibrarySelected)
	if !ok {
		return
	}
	drawGunLibraryCleanThumb(screen, g.gunLibraryRaster(g.gunLibrarySelected), 112, 525, 165, 72, 1, 0)
	drawSourceMenuText(screen, strings.ToUpper(def.DisplayName()), menuFontTwCen, 19, color.White, 210, 480)
	stats := []string{
		"DAMAGE  " + strconv.Itoa(int(math.Round(def.Firepower))),
		"FIRE RATE  " + strconv.Itoa(def.ROF),
		"AMMO  " + strconv.Itoa(def.Bullets),
		"RECOIL  " + strconv.Itoa(int(math.Round(def.Recoil))),
	}
	for i, stat := range stats {
		drawSourceMenuText(screen, stat, menuFontCondensedExtraBold, 13, color.White, 210+float64(i%2)*185, 512+float64(i/2)*27)
	}
	test := Rect{X: 756, Y: 514, W: 108, H: 44}
	ebitenutil.DrawRect(screen, test.X, test.Y, test.W, test.H, orange)
	drawSourceMenuText(screen, "TEST", menuFontCondensedExtraBold, 19, color.Black, 787, 525)
}

func (g *Game) drawCustomGunLibraryCard(screen *ebiten.Image, b gunLibraryButton) {
	x := b.X - b.W/2
	y := b.Y - b.H/2
	border := color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xff}
	if g.gunLibrarySelected == b.Gun {
		border = color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
	}
	ebitenutil.DrawRect(screen, x, y, b.W, b.H, border)
	ebitenutil.DrawRect(screen, x+3, y+3, b.W-6, b.H-6, color.NRGBA{R: 0xe3, G: 0xe3, B: 0xe3, A: 0xff})
	thumbHeight := 34.0
	if b.Gun == 178 {
		thumbHeight = 44 // the compound bow is intentionally tall rather than long
	}
	drawGunLibraryCleanThumb(screen, g.gunLibraryRaster(b.Gun), b.X, b.Y-8, b.W-14, thumbHeight, 1, 0)
	name := weaponDisplayName(b.Gun)
	runes := []rune(name)
	if len(runes) > 20 {
		name = string(runes[:19]) + "…"
	}
	drawSourceMenuText(screen, name, menuFontCondensed, 10.8, color.Black, x+6, y+b.H-18)
	if !g.gunUnlocked(b.Gun) {
		ebitenutil.DrawRect(screen, x+3, y+3, b.W-6, b.H-6, color.NRGBA{A: 0x88})
	}
}

func (g *Game) drawDevelopedGunLibraryPager(screen *ebiten.Image) {
	count := developedGunLibraryPageCount()
	if count <= 1 {
		return
	}
	buttonColor := color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
	borderColor := color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xff}
	for _, r := range []Rect{gunLibraryPreviousPageRect, gunLibraryNextPageRect} {
		ebitenutil.DrawRect(screen, r.X-2, r.Y-2, r.W+4, r.H+4, borderColor)
		ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, buttonColor)
	}
	drawSourceMenuText(screen, "< PREV", menuFontCondensed, 18, color.Black, gunLibraryPreviousPageRect.X+15, gunLibraryPreviousPageRect.Y+5)
	drawSourceMenuText(screen, "NEXT >", menuFontCondensed, 18, color.Black, gunLibraryNextPageRect.X+14, gunLibraryNextPageRect.Y+5)
	label := "ARSENAL " + strconv.Itoa(g.gunLibraryPage+1) + "/" + strconv.Itoa(count)
	drawSourceMenuText(screen, label, menuFontCondensed, 19, color.Black, 395, 168)
}

func drawDevelopedStatFill(screen *ebiten.Image, x, y, fraction float64) {
	fraction = math.Max(0, math.Min(1, fraction))
	ebitenutil.DrawRect(screen, x, y, 219.4*fraction, 5.6, color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff})
}

// GM1 keeps its exact source-authored stat frames. GM2 has no matching GM1
// Symbol1190 frames, so build the same five-bar card from the source weapon
// values while keeping the same geometry and typography.
func (g *Game) drawDevelopedGunLibraryStatCard(screen *ebiten.Image, frame int) {
	if frame <= 66 {
		g.drawGunLibraryStatCard(screen, frame)
		return
	}
	def, ok := WeaponByNumber(frame)
	if !ok {
		g.drawGunLibraryStatCard(screen, 7)
		return
	}

	const x = 458.1
	const y = 457.6
	ebitenutil.DrawRect(screen, x, y, 315, 114, color.NRGBA{R: 0x79, G: 0x79, B: 0x79, A: 0xff})
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	drawSourceMenuText(screen, def.DisplayName(), menuFontTwCen, 14.2, white, x+4.5, y+4.5)
	drawSourceMenuText(screen, "Damage", menuFontCondensed, 14.15, white, x+44, y+22.5)
	drawSourceMenuText(screen, "Rate of Fire", menuFontCondensed, 14.15, white, x+27, y+39)
	drawSourceMenuText(screen, "Ammo Capacity", menuFontCondensed, 14.15, white, x+9, y+55.5)
	drawSourceMenuText(screen, "Weight", menuFontCondensed, 14.15, white, x+50, y+73.5)
	drawSourceMenuText(screen, "Recoil", menuFontCondensed, 14.15, white, x+55, y+90)

	tops := []float64{29.0, 45.35, 61.9, 78.4, 95.0}
	for _, top := range tops {
		drawGunLibraryBarOutline(screen, x+91.2, y+top)
	}
	drawDevelopedStatFill(screen, x+91.2, y+29.0, def.Firepower/70)
	drawDevelopedStatFill(screen, x+91.2, y+45.35, (1/math.Max(1, float64(def.ROF)))/0.5)
	drawDevelopedStatFill(screen, x+91.2, y+61.9, float64(def.Bullets)/40)
	drawDevelopedStatFill(screen, x+91.2, y+78.4, def.Weight)
	drawDevelopedStatFill(screen, x+91.2, y+95.0, def.Recoil/10)
}
