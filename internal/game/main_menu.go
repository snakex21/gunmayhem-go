package game

import (
	"image"
	"math"
	"os/exec"

	"github.com/hajimehoshi/ebiten/v2"
)

// Main menu, rebuilt with the Mayhem UI kit (ui_kit.go): mode tiles over the
// dimmed arena backdrop, with the original Gun Mayhem logo cropped from the
// Flash main menu (Symbol 28) as the nod to the source game.

var (
	// mainMenuLogoSource is the logo's area of the Symbol 28 raster in stage
	// coordinates; the rest of that raster (skewed Flash buttons, social
	// icons) is no longer drawn.
	mainMenuLogoSource = Rect{X: 8, Y: 24, W: 640, H: 92}

	mainTileVersusRect   = Rect{X: 20, Y: 150, W: 414, H: 362}
	mainTileCampaignRect = Rect{X: 450, Y: 150, W: 207, H: 173}
	mainTileOnlineRect   = Rect{X: 673, Y: 150, W: 207, H: 173}
	mainTileLibraryRect  = Rect{X: 450, Y: 339, W: 207, H: 173}
	mainTileZombieRect   = Rect{X: 673, Y: 339, W: 207, H: 173}

	mainCreditsRect   = Rect{X: 538, Y: 24, W: 104, H: 36}
	mainMoreGamesRect = Rect{X: 650, Y: 24, W: 120, H: 36}
	mainQuitRect      = Rect{X: 778, Y: 24, W: 102, H: 36}
	mainUpdateRect    = Rect{X: uiMargin, Y: uiActionTop, W: 520, H: uiButtonH}
	mainOptionsRect   = uiActionPrimaryRect(200)
	mainQuitModalRect = Rect{X: 270, Y: 215, W: 360, H: 170}
)

func (g *Game) updateMainMenu() {
	g.runMainMenuUpdate(g.readUIInput(&g.mainMenuUI))
}

// runMainMenuUpdate is one update pass for a frame of input (tests drive it).
func (g *Game) runMainMenuUpdate(in uiInput) {
	ctx := g.mainMenuUI.begin(uiPassUpdate, nil, in)
	g.buildMainMenu(ctx)
	ctx.end()
}

func (g *Game) drawMainMenu(screen *ebiten.Image) {
	ctx := g.mainMenuUI.begin(uiPassDraw, screen, uiInput{X: g.uiLastCursorX, Y: g.uiLastCursorY})
	g.buildMainMenu(ctx)
	ctx.end()
}

func (g *Game) mainMenuSelectSound() { g.playSourceSFX("menu.wav", false) }

// openLocalCustomGame is the VERSUS tile: the local Custom Game setup.
func (g *Game) openLocalCustomGame() {
	g.customOnlineMode = false
	g.customNetworkOpen = false
	g.onlineMatchSetup = false
	g.customPage = customPageGame
	g.customMenuX = customPageTargetX(customPageGame)
	g.customCardEditSlot = 0
	g.customUI.reset()
	g.beginFade(screenCustomGame, fadePurposeScreen)
}

// openOnlineFromMainMenu goes straight to the server browser (previously
// reachable only through Custom Game -> ONLINE).
func (g *Game) openOnlineFromMainMenu() {
	g.customPage = customPageGame
	g.customMenuX = customPageTargetX(customPageGame)
	g.customCardEditSlot = 0
	g.returnToOnlineServerBrowser()
	g.beginFade(screenCustomGame, fadePurposeScreen)
}

func (g *Game) buildMainMenu(ctx *uiCtx) {
	ctx.Page(0.55)
	if ctx.drawing() {
		g.drawMainMenuLogo(ctx.dst)
		uiText(ctx.dst, "GO EDITION  /  "+currentUpdateVersionLabel(), true, uiSizeHint, uiOrange, uiMargin+4, 124, uiAlignLeft)
	}

	if ctx.ModeTile("main.versus", mainTileVersusRect, "VERSUS", uiModeTileOpts{
		Primary: true, TitleSize: 58, Tag: "CUSTOM GAME",
		Subtitle:  "Local match: 2-4 players and AI bots, your rules, map playlist",
		InfoTitle: "VERSUS", InfoBody: "Local Custom Game on this computer.",
		Art: g.drawMainVersusArt,
	}) {
		g.mainMenuSelectSound()
		g.openLocalCustomGame()
	}
	if ctx.ModeTile("main.campaign", mainTileCampaignRect, "CAMPAIGN", uiModeTileOpts{
		Tag: "GM1 / GM2 / EXTRA", Subtitle: "Missions, solo or co-op",
		InfoTitle: "CAMPAIGN", InfoBody: "Gun Mayhem 1 and 2 missions plus the EXTRA set.",
		Art: g.drawMainCampaignArt,
	}) {
		g.mainMenuSelectSound()
		g.beginFade(screenCampaign, fadePurposeScreen)
	}
	if ctx.ModeTile("main.online", mainTileOnlineRect, "ONLINE", uiModeTileOpts{
		Tag: "NET", TagFill: uiNetBlue, Subtitle: "Servers, invite codes, host",
		InfoTitle: "ONLINE", InfoBody: "Browse sessions, join with a code or host your own.",
		Art: drawMainOnlineArt,
	}) {
		g.mainMenuSelectSound()
		g.openOnlineFromMainMenu()
	}
	if ctx.ModeTile("main.library", mainTileLibraryRect, "GUN LIBRARY", uiModeTileOpts{
		Subtitle: "Every weapon, test range", TitleSize: 28,
		InfoTitle: "GUN LIBRARY", InfoBody: "Browse and test every weapon.",
		Art: g.drawMainLibraryArt,
	}) {
		g.mainMenuSelectSound()
		g.beginFade(screenGunLibrary, fadePurposeScreen)
	}
	ctx.ModeTile("main.zombie", mainTileZombieRect, "ZOMBIE", uiModeTileOpts{
		Disabled: true, Tag: "COMING SOON", Subtitle: "Waves of enemies",
		InfoTitle: "ZOMBIE", InfoBody: "Coming soon.",
	})

	if ctx.Button("main.credits", mainCreditsRect, "CREDITS", uiButtonOpts{Size: uiSizeBody}) {
		g.mainMenuSelectSound()
		g.beginFade(screenCredits, fadePurposeScreen)
	}
	if ctx.Button("main.more", mainMoreGamesRect, "MORE GAMES", uiButtonOpts{Size: uiSizeBody,
		InfoTitle: "MORE GAMES", InfoBody: "Opens armorgames.com in the browser, like the original button."}) {
		// Exact source action: getURL("http://armorgames.com", _blank). The
		// original btn6 deliberately plays no menu.wav.
		_ = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", "http://armorgames.com").Start()
	}
	if ctx.Button("main.quit", mainQuitRect, "QUIT", uiButtonOpts{Size: uiSizeBody}) {
		g.mainMenuSelectSound()
		g.mainMenuQuitOpen = true
	}

	if g.updateNoticeVisible() {
		kind := uiButtonPrimary
		if g.updateRuntime.phase == updatePhaseFailed {
			kind = uiButtonDanger
		}
		downloading := g.updateRuntime.phase == updatePhaseDownloading
		if ctx.Button("main.update", mainUpdateRect, g.updateNoticeLabel(), uiButtonOpts{Kind: kind, Disabled: downloading, Size: uiSizeBody,
			InfoTitle: "UPDATE", InfoBody: "Download and install the new version, then restart."}) {
			g.beginUpdateDownload()
		}
		if ctx.drawing() && downloading && g.updateRuntime.total > 0 {
			frac := math.Max(0, math.Min(1, float64(g.updateRuntime.done)/float64(g.updateRuntime.total)))
			r := mainUpdateRect
			uiFillRect(ctx.dst, Rect{X: r.X + 4, Y: r.Y + r.H - 7, W: (r.W - 8) * frac, H: 3}, uiOrange)
		}
	}
	if ctx.Button("main.options", mainOptionsRect, "OPTIONS", uiButtonOpts{}) {
		g.mainMenuSelectSound()
		g.beginFade(screenOptions, fadePurposeScreen)
	}

	ctx.KeyHintBar([]uiKeyHint{{"UP/DOWN", "Move"}, {"LEFT/RIGHT", "Move"}, {"ENTER", "Open"}, {"ESC", "Quit"}, {"F11", "Fullscreen"}})

	if g.mainMenuQuitOpen {
		g.buildMainQuitModal(ctx)
		return
	}
	if ctx.Back() {
		g.mainMenuQuitOpen = true
	}
}

func (g *Game) buildMainQuitModal(ctx *uiCtx) {
	r := mainQuitModalRect
	ctx.BeginModal(r, "QUIT GUN MAYHEM?")
	if ctx.drawing() {
		uiText(ctx.dst, "Settings are already saved.", false, uiSizeBody, uiTextMutedDk, r.X+14, r.Y+60, uiAlignLeft)
	}
	bw := (r.W - 14*3) / 2
	if ctx.Button("main.quit.no", Rect{X: r.X + 14, Y: r.Y + r.H - 58, W: bw, H: 44}, "CANCEL", uiButtonOpts{}) {
		g.mainMenuQuitOpen = false
	}
	if ctx.Button("main.quit.yes", Rect{X: r.X + 28 + bw, Y: r.Y + r.H - 58, W: bw, H: 44}, "QUIT", uiButtonOpts{Kind: uiButtonDanger}) {
		g.quitRequested = true
	}
	ctx.EndModal()
	if ctx.Back() {
		g.mainMenuQuitOpen = false
	}
}

// drawMainMenuLogo draws the logo part of the original Flash main menu.
func (g *Game) drawMainMenuLogo(dst *ebiten.Image) {
	if g.assets == nil {
		return
	}
	src := mainMenuLogoSource
	drawSourceRasterRegion(dst, g.assets.MainMenu, src, uiMargin-4, 18, 0.78)
}

// drawSourceRasterRegion draws the part of r that covers src (stage
// coordinates of the raster) with its top-left at (x, y).
func drawSourceRasterRegion(dst *ebiten.Image, r *SourceRaster, src Rect, x, y, scale float64) {
	if dst == nil || r == nil || r.Image == nil {
		return
	}
	b := r.Image.Bounds()
	px := 1.0
	if r.Bounds.W > 0 {
		px = float64(b.Dx()) / r.Bounds.W
	}
	py := 1.0
	if r.Bounds.H > 0 {
		py = float64(b.Dy()) / r.Bounds.H
	}
	x0 := b.Min.X + int(math.Floor((src.X-r.Bounds.X)*px))
	y0 := b.Min.Y + int(math.Floor((src.Y-r.Bounds.Y)*py))
	x1 := b.Min.X + int(math.Ceil((src.X+src.W-r.Bounds.X)*px))
	y1 := b.Min.Y + int(math.Ceil((src.Y+src.H-r.Bounds.Y)*py))
	crop := image.Rect(x0, y0, x1, y1).Intersect(b)
	if crop.Empty() {
		return
	}
	sub := r.Image.SubImage(crop).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(scale/px, scale/py)
	op.GeoM.Translate(x+float64(crop.Min.X-x0)*scale/px, y+float64(crop.Min.Y-y0)*scale/py)
	dst.DrawImage(sub, op)
}

// drawSetupPreviewIn draws the setup-card character (drawSetupPlayerPreview
// renders at the card's fixed Symbol1302 position) fitted into r.
func (g *Game) drawSetupPreviewIn(dst *ebiten.Image, cfg CustomPlayerConfig, r Rect, alpha float64) {
	if dst == nil || g.assets == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	if g.setupPreviewImg == nil {
		g.setupPreviewImg = ebiten.NewImage(220, 260)
	}
	img := g.setupPreviewImg
	img.Clear()
	g.drawSetupPlayerPreview(img, cfg, 1)
	// The figure (hat to feet) around the card's preview anchor (144.9,191.45).
	src := image.Rect(92, 52, 198, 210)
	scale := math.Min(r.W/float64(src.Dx()), r.H/float64(src.Dy()))
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(r.X+(r.W-float64(src.Dx())*scale)/2, r.Y+(r.H-float64(src.Dy())*scale)/2)
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	dst.DrawImage(img.SubImage(src).(*ebiten.Image), op)
}

func (g *Game) drawMainVersusArt(dst *ebiten.Image, r Rect, alpha float64) {
	players := sourceDefaultCustomPlayers()
	h := r.H * 0.56
	w := h * 0.7
	g.drawSetupPreviewIn(dst, players[1], Rect{X: r.X + r.W - w*1.75 - 14, Y: r.Y + 34, W: w, H: h}, alpha)
	g.drawSetupPreviewIn(dst, players[0], Rect{X: r.X + r.W - w - 10, Y: r.Y + 46, W: w, H: h}, alpha)
}

func (g *Game) drawMainCampaignArt(dst *ebiten.Image, r Rect, alpha float64) {
	if g.assets == nil {
		return
	}
	preview := g.assets.MapPreviewImages[5]
	if preview == nil {
		return
	}
	art := Rect{X: r.X + 10, Y: r.Y + 40, W: r.W - 20, H: r.H - 104}
	drawSetupThumbFit(dst, preview, art, alpha)
}

func drawMainOnlineArt(dst *ebiten.Image, r Rect, alpha float64) {
	// Signal-strength bars as a simple network glyph.
	const bars, barW, gap = 4, 16.0, 8.0
	total := bars*barW + (bars-1)*gap
	x := r.X + (r.W-total)/2
	base := r.Y + r.H*0.62
	for i := 0; i < bars; i++ {
		h := 18 + float64(i)*16
		uiFillRound(dst, Rect{X: x + float64(i)*(barW+gap), Y: base - h, W: barW, H: h}, 3, uiWithAlpha(uiNetBlue, alpha))
	}
}

func (g *Game) drawMainLibraryArt(dst *ebiten.Image, r Rect, alpha float64) {
	if g.assets == nil {
		return
	}
	art := Rect{X: r.X + 14, Y: r.Y + 34, W: r.W - 28, H: r.H - 98}
	drawSetupThumbFit(dst, g.assets.WeaponFrame(10, 0), art, alpha)
}
