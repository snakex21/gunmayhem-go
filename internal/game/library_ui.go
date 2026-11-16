package game

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Gun Library rebuilt with the Mayhem UI kit (phase 7): WEAPONS (category
// list, 20 per page, locked GM1 campaign guns), THROWABLES, GEAR (abilities)
// and POWERUPS tabs, an info panel with stats and descriptions, and TEST for
// weapons and throwables. Selection rules (gunUnlocked, isGunLibraryGunNumber)
// and the test range (startGunLibraryTest) are the existing ones.

const (
	gunLibraryViewWeapons = iota
	gunLibraryViewThrowables
	gunLibraryViewGear
	gunLibraryViewPowerups
)

// gunLibraryThrowableCategory is organizedGunLibraryTabs' THROWABLES entry,
// which startGunLibraryTest reads through organizedGunLibraryThrowableTab.
const gunLibraryThrowableCategory = 9

var gunLibraryViewLabels = []string{"WEAPONS", "THROWABLES", "GEAR", "POWERUPS"}

var (
	libraryTabBarRect   = Rect{X: 380, Y: 30, W: 500, H: 40}
	libraryCategoryRect = Rect{X: uiMargin, Y: uiContentTop, W: 160, H: uiContentBottom - uiContentTop}
	libraryGridRect     = Rect{X: 192, Y: uiContentTop, W: 418, H: uiContentBottom - uiContentTop}
	libraryWideGridRect = Rect{X: uiMargin, Y: uiContentTop, W: 590, H: uiContentBottom - uiContentTop}
	libraryInfoRect     = Rect{X: 622, Y: uiContentTop, W: 258, H: uiContentBottom - uiContentTop}
	libraryPageRect     = Rect{X: 300, Y: uiActionTop, W: 260, H: uiButtonH}
	libraryTestRect     = uiActionPrimaryRect(200)
)

// gunLibraryPowerupInfo describes the crate powerups (applyPowerup).
var gunLibraryPowerupInfo = [powerupKindCount]string{
	"One more life.",
	"Invisible for about 7 seconds. Thermal goggles still see you.",
	"A bubble shield for 10 seconds: hits push you far less.",
	"A full jetpack tank: hold UP in the air to fly.",
	"A clone of you joins the fight for a while.",
	"Run faster for 10 seconds.",
	"Shrink for about 13 seconds: a much smaller target.",
	"A second copy of your gun for 12 seconds: you fire both hands.",
	"A riot shield for the rest of this life: it stops shots and pushes from the front.",
	"The whole match runs at half speed for 5 seconds, for everyone.",
}

func (g *Game) gunLibraryCurrentView() int {
	if g.gunLibraryCategory == gunLibraryThrowableCategory {
		return gunLibraryViewThrowables
	}
	switch g.gunLibraryView {
	case gunLibraryViewGear, gunLibraryViewPowerups:
		return g.gunLibraryView
	}
	return gunLibraryViewWeapons
}

func (g *Game) setGunLibraryView(view int) {
	switch view {
	case gunLibraryViewThrowables:
		g.gunLibraryCategory = gunLibraryThrowableCategory
		g.gunLibraryPage = 0
	default:
		if g.gunLibraryCategory == gunLibraryThrowableCategory {
			g.gunLibraryCategory = 0
			g.gunLibraryPage = 0
		}
	}
	g.gunLibraryView = view
	g.gunLibraryGear = 0
	if view == gunLibraryViewPowerups && g.assets != nil {
		g.assets.EnsureGameplay() // the source powerup icons (Symbol 730)
	}
}

func (g *Game) gunLibraryWeaponCategory() DevelopedWeaponCategory {
	if g.gunLibraryCategory < 0 || g.gunLibraryCategory >= len(organizedGunLibraryTabs) || g.gunLibraryCategory == gunLibraryThrowableCategory {
		g.gunLibraryCategory = 0
	}
	return organizedGunLibraryTabs[g.gunLibraryCategory].category
}

// gunLibraryPageWeapons is the current page of the current category.
func (g *Game) gunLibraryPageWeapons() []int {
	category := g.gunLibraryWeaponCategory()
	weapons := organizedGunLibraryNumbers(category)
	pages := organizedGunLibraryPageCount(category)
	if g.gunLibraryPage < 0 || g.gunLibraryPage >= pages {
		g.gunLibraryPage = 0
	}
	start := g.gunLibraryPage * organizedGunLibraryPageSize
	end := start + organizedGunLibraryPageSize
	if end > len(weapons) {
		end = len(weapons)
	}
	return weapons[start:end]
}

// selectGunLibraryWeapon is the old card click: only unlocked library guns.
func (g *Game) selectGunLibraryWeapon(gun int) {
	if isGunLibraryGunNumber(gun) && g.gunUnlocked(gun) {
		g.gunLibrarySelected = gun
	}
}

func (g *Game) gunLibraryCanTest() bool {
	switch g.gunLibraryCurrentView() {
	case gunLibraryViewThrowables:
		return true
	case gunLibraryViewWeapons:
		return isGunLibraryGunNumber(g.gunLibrarySelected) && g.gunUnlocked(g.gunLibrarySelected)
	}
	return false
}

var (
	gunLibraryMaximaOnce                                      sync.Once
	gunLibraryMaxRounds, gunLibraryMaxRate, gunLibraryMaxPush float64
)

// gunLibraryWeaponMaxima scales the stat bars to the whole library.
func gunLibraryWeaponMaxima() (rounds, rate, push float64) {
	gunLibraryMaximaOnce.Do(func() {
		for _, n := range organizedGunLibraryNumbers(0) {
			def, ok := WeaponByNumber(n)
			if !ok {
				continue
			}
			if b := float64(def.Bullets); b < 9999 && b > gunLibraryMaxRounds {
				gunLibraryMaxRounds = b
			}
			gunLibraryMaxRate = math.Max(gunLibraryMaxRate, weaponShotsPerSecond(def))
			gunLibraryMaxPush = math.Max(gunLibraryMaxPush, def.Firepower)
		}
		gunLibraryMaxRounds = math.Max(gunLibraryMaxRounds, 1)
		gunLibraryMaxRate = math.Max(gunLibraryMaxRate, 1)
		gunLibraryMaxPush = math.Max(gunLibraryMaxPush, 1)
	})
	return gunLibraryMaxRounds, gunLibraryMaxRate, gunLibraryMaxPush
}

// ---------------------------------------------------------------------------
// Update / draw entry points

func (g *Game) updateGunLibraryMenu() {
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		g.changeGunLibraryPage(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		g.changeGunLibraryPage(1)
	}
	g.runGunLibraryUpdate(g.readUIInput(&g.libraryUI))
}

// runGunLibraryUpdate is one update pass for a frame of input (tests drive it).
func (g *Game) runGunLibraryUpdate(in uiInput) {
	ctx := g.libraryUI.begin(uiPassUpdate, nil, in)
	g.buildGunLibrary(ctx)
	ctx.end()
}

func (g *Game) drawGunLibraryMenu(screen *ebiten.Image) {
	if g.assets != nil {
		g.assets.EnsureGunLibrary()
	}
	ctx := g.libraryUI.begin(uiPassDraw, screen, uiInput{X: g.uiLastCursorX, Y: g.uiLastCursorY})
	g.buildGunLibrary(ctx)
	ctx.end()
}

func (g *Game) buildGunLibrary(ctx *uiCtx) {
	ctx.Page(0.92)
	view := g.gunLibraryCurrentView()
	trail := "MAIN MENU  /  GUN LIBRARY  /  " + gunLibraryViewLabels[view]
	if view == gunLibraryViewWeapons {
		category := g.gunLibraryWeaponCategory()
		trail += fmt.Sprintf("  /  %s  -  %d", organizedGunLibraryTabs[g.gunLibraryCategory].label, len(organizedGunLibraryNumbers(category)))
	}
	ctx.Header("GUN LIBRARY", trail)
	if v := ctx.TabBar("lib.tabs", libraryTabBarRect, gunLibraryViewLabels, view); v != view {
		g.setGunLibraryView(v)
		g.playSourceSFX("menu.wav", false)
		return
	}

	switch view {
	case gunLibraryViewThrowables:
		g.buildGunLibraryThrowables(ctx)
	case gunLibraryViewGear:
		g.buildGunLibraryGear(ctx)
	case gunLibraryViewPowerups:
		g.buildGunLibraryPowerups(ctx)
	default:
		g.buildGunLibraryWeapons(ctx)
	}

	if ctx.Button("lib.back", uiActionBackRect(), "MAIN MENU", uiButtonOpts{InfoTitle: "MAIN MENU"}) {
		g.leaveGunLibrary()
		return
	}
	testInfo := "Try the selected weapon on the test range against a dummy."
	if view == gunLibraryViewThrowables {
		testInfo = "Try the selected throwable on the test range with 30 of them."
	} else if view != gunLibraryViewWeapons {
		testInfo = "Gear and powerups are tried in a match."
	}
	if ctx.Button("lib.test", libraryTestRect, "TEST", uiButtonOpts{Kind: uiButtonPrimary, Disabled: !g.gunLibraryCanTest() || g.fadeActive,
		InfoTitle: "TEST", InfoBody: testInfo}) {
		g.playSourceSFX("menu.wav", false)
		g.startGunLibraryTest()
		return
	}
	hints := []uiKeyHint{{"ESC", "Main menu"}, {"UP/DOWN", "Move"}, {"LEFT/RIGHT", "Move"}, {"ENTER", "Select"}, {"Q", "Prev tab"}, {"E", "Next tab"}}
	if view == gunLibraryViewWeapons {
		hints = append(hints, uiKeyHint{"PGUP/PGDN", "Page"})
	}
	ctx.KeyHintBar(hints)
	if ctx.Back() {
		g.leaveGunLibrary()
	}
}

func (g *Game) leaveGunLibrary() {
	g.gunLibraryPressed = 0
	g.playSourceSFX("menu.wav", false)
	g.beginFade(screenMainMenu, fadePurposeScreen)
}

// libraryGrid lays n tiles out in cols columns inside area.
func libraryGridTile(area Rect, cols, rows, i int) Rect {
	const gap = 8.0
	w := (area.W - float64(cols-1)*gap) / float64(cols)
	h := (area.H - float64(rows-1)*gap) / float64(rows)
	return Rect{X: area.X + float64(i%cols)*(w+gap), Y: area.Y + float64(i/cols)*(h+gap), W: w, H: h}
}

// ---------------------------------------------------------------------------
// WEAPONS

func (g *Game) buildGunLibraryWeapons(ctx *uiCtx) {
	ctx.Panel(libraryCategoryRect, "")
	weaponTabs := len(organizedGunLibraryTabs) - 1 // THROWABLES is its own tab
	rows := make([]string, 0, weaponTabs)
	for _, tab := range organizedGunLibraryTabs[:weaponTabs] {
		rows = append(rows, tab.label)
	}
	scroll := 0
	listRect := Rect{X: libraryCategoryRect.X + 6, Y: libraryCategoryRect.Y + 6, W: libraryCategoryRect.W - 12, H: float64(weaponTabs) * 32}
	if picked := ctx.List("lib.cat", listRect, rows, g.gunLibraryCategory, &scroll); picked >= 0 && picked != g.gunLibraryCategory {
		g.gunLibraryCategory = picked
		g.gunLibraryPage = 0
	}
	category := g.gunLibraryWeaponCategory()
	if ctx.drawing() {
		uiParagraph(ctx.dst, "Locked guns are unlocked by the Gun Mayhem 1 campaign.", false, uiSizeHint, uiTextMuted,
			libraryCategoryRect.X+10, listRect.Y+listRect.H+22, libraryCategoryRect.W-20, 17, 5)
	}

	infoID := ctx.InfoID()
	shown := g.gunLibrarySelected
	for i, gun := range g.gunLibraryPageWeapons() {
		gun := gun
		id := fmt.Sprintf("lib.gun.%d", gun)
		locked := !g.gunUnlocked(gun)
		body := weaponStatsLine(gun)
		if locked {
			body = "Locked: complete Gun Mayhem 1 campaign levels to unlock it."
		}
		if ctx.Tile(id, libraryGridTile(libraryGridRect, 4, 5, i), strings.ToUpper(weaponDisplayName(gun)), uiTileOpts{
			Selected: g.gunLibrarySelected == gun, Disabled: locked, Locked: locked,
			InfoTitle: strings.ToUpper(weaponDisplayName(gun)), InfoBody: body,
			Art: func(dst *ebiten.Image, inner Rect) {
				drawSetupThumbFit(dst, g.gunLibraryRaster(gun), inner, 1)
			},
		}) {
			g.selectGunLibraryWeapon(gun)
			g.playSourceSFX("menu.wav", false)
		}
		if id == infoID {
			shown = gun
		}
	}
	// The mouse wheel over the grid turns pages.
	if ctx.pass == uiPassUpdate && ctx.in.Wheel != 0 && libraryGridRect.Contains(ctx.in.X, ctx.in.Y) && ctx.layer == ctx.topLayer {
		if ctx.in.Wheel > 0 {
			g.changeGunLibraryPage(-1)
		} else {
			g.changeGunLibraryPage(1)
		}
	}
	pages := organizedGunLibraryPageCount(category)
	if d := ctx.Stepper("lib.page", libraryPageRect, fmt.Sprintf("PAGE %d / %d", g.gunLibraryPage+1, pages), pages > 1, pages > 1, false,
		"PAGE", "PAGE UP / PAGE DOWN or the mouse wheel also turn pages."); d != 0 {
		g.changeGunLibraryPage(d)
	}
	g.drawGunLibraryWeaponInfo(ctx, shown)
}

func (g *Game) drawGunLibraryWeaponInfo(ctx *uiCtx, gun int) {
	if !ctx.drawing() {
		return
	}
	r := libraryInfoRect
	uiFillRound(ctx.dst, r, 6, uiInk700)
	if gun == 0 || !isGunLibraryGunNumber(gun) {
		uiText(ctx.dst, "SELECT A WEAPON", true, uiSizeSection, uiTextLight, r.X+14, r.Y+30, uiAlignLeft)
		uiParagraph(ctx.dst, "Pick a weapon to see its stats, then TEST it on the range.", false, uiSizeBody, uiTextMuted, r.X+14, r.Y+62, r.W-28, 19, 4)
		return
	}
	art := Rect{X: r.X + 10, Y: r.Y + 10, W: r.W - 20, H: 96}
	uiFillRound(ctx.dst, art, 4, uiPaper)
	drawSetupThumbFit(ctx.dst, g.gunLibraryRaster(gun), uiInset(art, 6), 1)
	y := art.Y + art.H + 22
	uiText(ctx.dst, uiFit(strings.ToUpper(weaponDisplayName(gun)), true, uiSizeSection, r.W-28), true, uiSizeSection, uiTextLight, r.X+14, y, uiAlignLeft)
	y += 22
	sub := gunLibraryCategoryLabel(gun)
	if !g.gunUnlocked(gun) {
		sub += "  /  LOCKED"
	} else if gun == g.gunLibrarySelected {
		sub += "  /  SELECTED"
	}
	uiText(ctx.dst, sub, true, uiSizeHint, uiOrange, r.X+14, y, uiAlignLeft)
	y += 26
	maxRounds, maxRate, maxPush := gunLibraryWeaponMaxima()
	y = drawWeaponStatBars(ctx.dst, gun, r.X+14, y, r.W-28, maxRounds, maxRate, maxPush)
	if desc := weaponArsenal2Description(gun); desc != "" {
		left := int((r.Y + r.H - 8 - y) / 17)
		uiParagraph(ctx.dst, desc, false, uiSizeHint, uiTextMuted, r.X+14, y+4, r.W-28, 17, left)
	}
}

func gunLibraryCategoryLabel(gun int) string {
	category := sourceWeaponCategory(gun)
	if spec, ok := developedWeaponSpec(gun); ok {
		category = spec.Category
		if category == DevelopedCategoryMelee {
			return "MELEE"
		}
	}
	for _, tab := range organizedGunLibraryTabs {
		if tab.category == category && !tab.throwables && tab.category != 0 {
			return tab.label
		}
	}
	return "WEAPON"
}

// ---------------------------------------------------------------------------
// THROWABLES

func (g *Game) buildGunLibraryThrowables(ctx *uiCtx) {
	ensureThrowableVisuals()
	selected := normalizedThrowableType(g.gunLibraryThrowableSelected)
	shown := selected
	infoID := ctx.InfoID()
	rows := (throwableTypeCount + 3) / 4
	for kind := 0; kind < throwableTypeCount; kind++ {
		kind := kind
		id := fmt.Sprintf("lib.throw.%d", kind)
		if ctx.Tile(id, libraryGridTile(libraryWideGridRect, 4, rows, kind), strings.ToUpper(throwableName(kind)), uiTileOpts{
			Selected: kind == selected, InfoTitle: strings.ToUpper(throwableName(kind)), InfoBody: throwableLoadoutDescription(kind),
			Art: func(dst *ebiten.Image, inner Rect) { g.drawThrowableArt(dst, kind, inner) },
		}) {
			g.gunLibraryThrowableSelected = kind
			g.playSourceSFX("menu.wav", false)
		}
		if id == infoID {
			shown = kind
		}
	}
	if !ctx.drawing() {
		return
	}
	r := libraryInfoRect
	uiFillRound(ctx.dst, r, 6, uiInk700)
	art := Rect{X: r.X + 10, Y: r.Y + 10, W: r.W - 20, H: 96}
	uiFillRound(ctx.dst, art, 4, uiPaper)
	g.drawThrowableArt(ctx.dst, shown, uiInset(art, 8))
	p := throwableProfileFor(shown)
	y := art.Y + art.H + 22
	uiText(ctx.dst, uiFit(strings.ToUpper(p.Name), true, uiSizeSection, r.W-28), true, uiSizeSection, uiTextLight, r.X+14, y, uiAlignLeft)
	y += 22
	uiText(ctx.dst, throwableBehaviorLabel(shown), true, uiSizeHint, uiOrange, r.X+14, y, uiAlignLeft)
	y += 26
	rowsText := [][2]string{{"FUSE", fmt.Sprintf("%.1f S", float64(p.Fuse)/30)}, {"RADIUS", fmt.Sprintf("%.0f", p.Radius)}}
	if p.Behavior == throwableBehaviorExplode {
		rowsText = append(rowsText, [2]string{"PUSH", fmt.Sprintf("%.0f", p.HorizontalPush)})
	}
	for _, row := range rowsText {
		uiText(ctx.dst, row[0], true, uiSizeHint, uiTextMuted, r.X+14, y, uiAlignLeft)
		uiText(ctx.dst, row[1], true, uiSizeHint, uiTextLight, r.X+r.W-14, y, uiAlignRight)
		y += 20
	}
	left := int((r.Y + r.H - 8 - y) / 17)
	uiParagraph(ctx.dst, throwableLoadoutDescription(shown)+" TEST starts the range with 30 of them.", false, uiSizeHint, uiTextMuted, r.X+14, y+6, r.W-28, 17, left)
}

// ---------------------------------------------------------------------------
// GEAR and POWERUPS (information only)

func (g *Game) buildGunLibraryGear(ctx *uiCtx) {
	abilities := selectableAbilityNumbers()
	cols := 5
	rows := (len(abilities) + cols - 1) / cols
	shown := -1
	infoID := ctx.InfoID()
	for i, a := range abilities {
		a := a
		id := fmt.Sprintf("lib.gear.%d", a)
		if ctx.Tile(id, libraryGridTile(libraryWideGridRect, cols, rows, i), abilityName(a), uiTileOpts{
			Selected: g.gunLibraryGear == i, InfoTitle: abilityName(a), InfoBody: abilityDescription(a),
			Art: func(dst *ebiten.Image, inner Rect) {
				s := math.Min(inner.W, inner.H) / 2.6
				drawAbilityIcon(dst, a, inner.X+inner.W/2, inner.Y+inner.H/2, s, 1)
			},
		}) {
			g.gunLibraryGear = i
		}
		if id == infoID || (shown < 0 && g.gunLibraryGear == i) {
			shown = a
		}
	}
	if !ctx.drawing() || shown < 0 {
		return
	}
	body := abilityDescription(shown)
	switch {
	case shown == abilityFlashGoggles || shown == abilityThermalGoggles || shown == abilityGasMask:
		body += " Gear: worn instead of your hat."
	case shown == abilityKevlar:
		body += " Gear: a vest over your shirt."
	}
	g.drawGunLibraryIconInfo(ctx, abilityName(shown), "ABILITY / GEAR", body+" Pick it in a player's LOADOUT.", func(dst *ebiten.Image, r Rect) {
		drawAbilityIcon(dst, shown, r.X+r.W/2, r.Y+r.H/2, math.Min(32, r.H/2.4), 1)
	})
}

func (g *Game) buildGunLibraryPowerups(ctx *uiCtx) {
	shown := -1
	infoID := ctx.InfoID()
	for kind := 0; kind < powerupKindCount; kind++ {
		kind := kind
		id := fmt.Sprintf("lib.power.%d", kind)
		if ctx.Tile(id, libraryGridTile(libraryWideGridRect, 5, 2, kind), powerupName(kind), uiTileOpts{
			Selected: g.gunLibraryGear == kind, InfoTitle: powerupName(kind), InfoBody: gunLibraryPowerupInfo[kind],
			Art: func(dst *ebiten.Image, inner Rect) { g.drawGunLibraryPowerupIcon(dst, kind, inner) },
		}) {
			g.gunLibraryGear = kind
		}
		if id == infoID || (shown < 0 && g.gunLibraryGear == kind) {
			shown = kind
		}
	}
	if !ctx.drawing() || shown < 0 {
		return
	}
	sub := "CRATE POWERUP"
	body := gunLibraryPowerupInfo[shown]
	if shown >= powerupAkimbo {
		sub = "NEW POWERUP"
		body += " Drops when the developed arsenal is in the crate pool (Custom Game, EXTRA campaign)."
	}
	g.drawGunLibraryIconInfo(ctx, powerupName(shown), sub, body, func(dst *ebiten.Image, r Rect) {
		g.drawGunLibraryPowerupIcon(dst, shown, r)
	})
}

func (g *Game) drawGunLibraryPowerupIcon(dst *ebiten.Image, kind int, r Rect) {
	var icon *SourceRaster
	if g.assets != nil {
		icon = g.assets.PowerupIcons[kind]
	}
	if icon == nil {
		icon = arsenal2PowerupIcon(kind)
	}
	if icon == nil {
		uiTextIn(dst, "?", true, uiSizeSection, uiTextMutedDk, r, uiAlignCenter, 0)
		return
	}
	drawSetupThumbFit(dst, icon, r, 1)
}

func (g *Game) drawGunLibraryIconInfo(ctx *uiCtx, title, sub, body string, art func(dst *ebiten.Image, r Rect)) {
	r := libraryInfoRect
	uiFillRound(ctx.dst, r, 6, uiInk700)
	box := Rect{X: r.X + 10, Y: r.Y + 10, W: r.W - 20, H: 96}
	uiFillRound(ctx.dst, box, 4, uiPaper)
	art(ctx.dst, uiInset(box, 8))
	y := box.Y + box.H + 22
	uiText(ctx.dst, uiFit(strings.ToUpper(title), true, uiSizeSection, r.W-28), true, uiSizeSection, uiTextLight, r.X+14, y, uiAlignLeft)
	y += 22
	uiText(ctx.dst, sub, true, uiSizeHint, uiOrange, r.X+14, y, uiAlignLeft)
	left := int((r.Y + r.H - 8 - y - 16) / 19)
	uiParagraph(ctx.dst, body, false, uiSizeBody, uiTextMuted, r.X+14, y+26, r.W-28, 19, left)
}
