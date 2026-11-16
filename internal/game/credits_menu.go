package game

import (
	"os/exec"

	"github.com/hajimehoshi/ebiten/v2"
)

// Credits rebuilt with the Mayhem UI kit (phase 7). The original game's
// credits (Symbol 1040-1045) keep their names, quotes, links and the two
// doodles cropped from the Flash credits raster; a second card credits this
// port.

// openExternalURL opens a link in the default browser (tests replace it).
var openExternalURL = func(url string) {
	_ = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url).Start()
}

var (
	creditsOriginalCard = Rect{X: uiMargin, Y: uiContentTop, W: 520, H: uiContentBottom - uiContentTop}
	creditsPortCard     = Rect{X: 552, Y: uiContentTop, W: 328, H: uiContentBottom - uiContentTop}
	creditsKevinRect    = Rect{X: 300, Y: 154, W: 220, H: 34}
	creditsMusicRect    = Rect{X: 300, Y: 266, W: 220, H: 34}

	// Doodles in the Symbol 1046 credits raster (stage coordinates).
	creditsDoodleKevin = Rect{X: 505, Y: 37, W: 359, H: 144}
	creditsDoodleArvin = Rect{X: 72, Y: 279, W: 268, H: 290}
)

type creditsLine struct {
	role, name, quote string
}

var creditsOriginal = []creditsLine{
	{"CODE / DESIGN", "Kevin Gu", "\"Don't touch the screen!\""},
	{"ART GUY", "Arvin Eksiri", "\"I can finish drawing 20 maps in one week\""},
	{"MUSIC", "Kevin MacLeod", "incompetech.com"},
	{"TESTING", "TimmyFool, Lufas, Arfizzy", "\"I've got this in the bag\" / \"Get Owned!\""},
}

var creditsPort = []creditsLine{
	{"GAMES", "Gun Mayhem and Gun Mayhem 2", "Kevin Gu, Armor Games"},
	{"ENGINE", "Go + Ebitengine", "Port of the Flash / ActionScript original"},
	{"UI FONT", "Barlow Condensed", "SIL Open Font License"},
	{"ONLINE", "Cloudflare Quick Tunnel", "cloudflared, bundled"},
}

func (g *Game) updateCreditsMenu() {
	g.runCreditsUpdate(g.readUIInput(&g.creditsUI))
}

// runCreditsUpdate is one update pass for a frame of input (tests drive it).
func (g *Game) runCreditsUpdate(in uiInput) {
	ctx := g.creditsUI.begin(uiPassUpdate, nil, in)
	g.buildCredits(ctx)
	ctx.end()
}

func (g *Game) drawCreditsMenu(screen *ebiten.Image) {
	ctx := g.creditsUI.begin(uiPassDraw, screen, uiInput{X: g.uiLastCursorX, Y: g.uiLastCursorY})
	g.buildCredits(ctx)
	ctx.end()
}

func (g *Game) leaveCredits() {
	g.menuPressed = menuHitNone
	g.playSourceSFX("menu.wav", false)
	g.beginFade(screenMainMenu, fadePurposeScreen)
}

func (g *Game) buildCredits(ctx *uiCtx) {
	ctx.Page(0.9)
	ctx.Header("CREDITS", "MAIN MENU  /  CREDITS  /  "+currentUpdateVersionLabel())
	if ctx.drawing() {
		r := creditsOriginalCard
		ctx.Card(r, "GUN MAYHEM")
		if g.assets != nil && g.assets.CreditsMenu != nil {
			art := Rect{X: r.X + 14, Y: r.Y + 300, W: 150, H: 110}
			uiFillRound(ctx.dst, art, 5, uiPaperHover)
			scale := minFloat(art.W/creditsDoodleArvin.W, art.H/creditsDoodleArvin.H)
			drawSourceRasterRegion(ctx.dst, g.assets.CreditsMenu, creditsDoodleArvin,
				art.X+(art.W-creditsDoodleArvin.W*scale)/2, art.Y+(art.H-creditsDoodleArvin.H*scale)/2, scale)
			code := Rect{X: r.X + 176, Y: r.Y + 300, W: r.W - 190, H: 110}
			uiFillRound(ctx.dst, code, 5, uiPaperHover)
			scale = minFloat(code.W/creditsDoodleKevin.W, code.H/creditsDoodleKevin.H)
			drawSourceRasterRegion(ctx.dst, g.assets.CreditsMenu, creditsDoodleKevin,
				code.X+(code.W-creditsDoodleKevin.W*scale)/2, code.Y+(code.H-creditsDoodleKevin.H*scale)/2, scale)
		}
		drawCreditsLines(ctx.dst, creditsOriginal, r.X+16, r.Y+58, r.W-260, 56)

		p := creditsPortCard
		ctx.Card(p, "THIS PORT")
		drawCreditsLines(ctx.dst, creditsPort, p.X+16, p.Y+58, p.W-32, 56)
		uiParagraph(ctx.dst, "A private, non-commercial fan port. All original art, sounds and music belong to their authors.", false, uiSizeHint, uiTextMutedDk,
			p.X+16, p.Y+p.H-48, p.W-32, 17, 3)
	}
	if ctx.Button("cred.kevin", creditsKevinRect, "THEKEVINGU.COM", uiButtonOpts{Size: uiSizeBody,
		InfoTitle: "KEVIN GU", InfoBody: "Opens thekevingu.com in the browser."}) {
		openExternalURL("http://www.thekevingu.com")
	}
	if ctx.Button("cred.music", creditsMusicRect, "INCOMPETECH.COM", uiButtonOpts{Size: uiSizeBody,
		InfoTitle: "MUSIC", InfoBody: "Opens incompetech.com in the browser."}) {
		openExternalURL("http://www.incompetech.com")
	}
	if ctx.Button("cred.back", uiActionBackRect(), "MAIN MENU", uiButtonOpts{InfoTitle: "MAIN MENU"}) {
		g.leaveCredits()
		return
	}
	ctx.KeyHintBar([]uiKeyHint{{"ESC", "Main menu"}, {"UP/DOWN", "Move"}, {"ENTER", "Open link"}})
	if ctx.Back() {
		g.leaveCredits()
	}
}

func drawCreditsLines(dst *ebiten.Image, lines []creditsLine, x, y, w, step float64) {
	for i, l := range lines {
		top := y + float64(i)*step
		uiText(dst, l.role, true, uiSizeHint, uiOrangePress, x, top, uiAlignLeft)
		uiText(dst, uiFit(l.name, true, uiSizeSection, w), true, uiSizeSection, uiTextDark, x, top+20, uiAlignLeft)
		if l.quote != "" {
			uiText(dst, uiFit(l.quote, false, uiSizeHint, w), false, uiSizeHint, uiTextMutedDk, x, top+40, uiAlignLeft)
		}
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
