package game

import (
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

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
		category := spec.Category
		if category == DevelopedCategoryMelee {
			category = DevelopedCategorySpecial // the bat sits with the specials
		}
		appendIf(number, category)
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
	// Four columns x three rows: room for 12 throwables above the
	// SELECTED THROWABLE panel at y=408.
	return Rect{
		X: 24 + float64(kind%4)*214,
		Y: 126 + float64(kind/4)*94,
		W: 204,
		H: 88,
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
			drawGunLibraryCleanThumb(screen, g.assets.Grenade, r.X+36, r.Y+44, 56, 60, 1, 0)
		} else if img := throwableImages[kind]; img != nil {
			w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
			scale := math.Min(56/w, 62/h)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-w/2, -h/2)
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(r.X+36, r.Y+44)
			screen.DrawImage(img, op)
		}
		profile := throwableProfiles[kind]
		drawSourceMenuText(screen, profile.Name, menuFontTwCen, 14, color.Black, r.X+70, r.Y+10)
		drawSourceMenuText(screen, "FUSE  "+strconv.Itoa(profile.Fuse), menuFontCondensedExtraBold, 10, color.Black, r.X+70, r.Y+34)
		drawSourceMenuText(screen, "RADIUS  "+strconv.Itoa(int(profile.Radius)), menuFontCondensedExtraBold, 10, color.Black, r.X+70, r.Y+49)
		label := throwableBehaviorLabel(kind)
		if profile.Behavior == throwableBehaviorExplode {
			label = "PUSH  " + strconv.Itoa(int(math.Round(profile.HorizontalPush)))
		}
		drawSourceMenuText(screen, label, menuFontCondensedExtraBold, 9, color.Black, r.X+70, r.Y+64)
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
	if desc := weaponArsenal2Description(g.gunLibrarySelected); desc != "" {
		lines := wrapTextToWidth(desc, 160, func(s string) float64 { return measureMenuText(menuFontCondensed, 10, s) })
		for i, line := range lines {
			if i >= 6 {
				break
			}
			drawSourceMenuText(screen, line, menuFontCondensed, 10, color.NRGBA{R: 0xff, G: 0xcc, B: 0x66, A: 0xff}, 585, 478+float64(i)*13)
		}
	}
	test := Rect{X: 756, Y: 514, W: 108, H: 44}
	ebitenutil.DrawRect(screen, test.X, test.Y, test.W, test.H, orange)
	drawSourceMenuText(screen, "TEST", menuFontCondensedExtraBold, 19, color.Black, 787, 525)
}
