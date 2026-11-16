package game

import (
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	ebitentext "github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
)

// Mayhem UI kit: drawing primitives, text and static layout pieces
// (see ui_kit_core.go).

// ---------------------------------------------------------------------------
// Drawing primitives

func uiRoundRectPath(r Rect, radius float64) *vector.Path {
	var p vector.Path
	rad := float32(math.Min(radius, math.Min(r.W, r.H)/2))
	x0, y0 := float32(r.X), float32(r.Y)
	x1, y1 := float32(r.X+r.W), float32(r.Y+r.H)
	if rad <= 0 {
		p.MoveTo(x0, y0)
		p.LineTo(x1, y0)
		p.LineTo(x1, y1)
		p.LineTo(x0, y1)
		p.Close()
		return &p
	}
	p.MoveTo(x0+rad, y0)
	p.LineTo(x1-rad, y0)
	p.ArcTo(x1, y0, x1, y0+rad, rad)
	p.LineTo(x1, y1-rad)
	p.ArcTo(x1, y1, x1-rad, y1, rad)
	p.LineTo(x0+rad, y1)
	p.ArcTo(x0, y1, x0, y1-rad, rad)
	p.LineTo(x0, y0+rad)
	p.ArcTo(x0, y0, x0+rad, y0, rad)
	p.Close()
	return &p
}

func uiFillRound(dst *ebiten.Image, r Rect, radius float64, clr color.NRGBA) {
	if dst == nil || r.W <= 0 || r.H <= 0 || clr.A == 0 {
		return
	}
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, uiRoundRectPath(r, radius), nil, op)
}

func uiStrokeRound(dst *ebiten.Image, r Rect, radius, width float64, clr color.NRGBA) {
	if dst == nil || r.W <= 0 || r.H <= 0 || clr.A == 0 {
		return
	}
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.StrokePath(dst, uiRoundRectPath(r, radius), &vector.StrokeOptions{Width: float32(width)}, op)
}

func uiFillRect(dst *ebiten.Image, r Rect, clr color.NRGBA) {
	if dst == nil || clr.A == 0 {
		return
	}
	ebitenutil.DrawRect(dst, r.X, r.Y, r.W, r.H, clr)
}

func uiInset(r Rect, d float64) Rect {
	return Rect{X: r.X + d, Y: r.Y + d, W: r.W - 2*d, H: r.H - 2*d}
}

func uiFocusRing(dst *ebiten.Image, r Rect) {
	uiStrokeRound(dst, uiInset(r, -3), 6, 2, uiOrange)
}

type uiArrowDir int

const (
	uiArrowUp uiArrowDir = iota
	uiArrowDown
	uiArrowLeft
	uiArrowRight
)

// uiDrawArrow draws a filled arrow glyph (Barlow has no arrow characters).
func uiDrawArrow(dst *ebiten.Image, cx, cy, size float64, dir uiArrowDir, clr color.NRGBA) {
	if dst == nil {
		return
	}
	s := float32(size / 2)
	x, y := float32(cx), float32(cy)
	var p vector.Path
	switch dir {
	case uiArrowUp:
		p.MoveTo(x, y-s)
		p.LineTo(x+s, y+s*0.2)
		p.LineTo(x+s*0.35, y+s*0.2)
		p.LineTo(x+s*0.35, y+s)
		p.LineTo(x-s*0.35, y+s)
		p.LineTo(x-s*0.35, y+s*0.2)
		p.LineTo(x-s, y+s*0.2)
	case uiArrowDown:
		p.MoveTo(x, y+s)
		p.LineTo(x+s, y-s*0.2)
		p.LineTo(x+s*0.35, y-s*0.2)
		p.LineTo(x+s*0.35, y-s)
		p.LineTo(x-s*0.35, y-s)
		p.LineTo(x-s*0.35, y-s*0.2)
		p.LineTo(x-s, y-s*0.2)
	case uiArrowLeft:
		p.MoveTo(x-s, y)
		p.LineTo(x+s*0.2, y-s)
		p.LineTo(x+s*0.2, y-s*0.35)
		p.LineTo(x+s, y-s*0.35)
		p.LineTo(x+s, y+s*0.35)
		p.LineTo(x+s*0.2, y+s*0.35)
		p.LineTo(x+s*0.2, y+s)
	default:
		p.MoveTo(x+s, y)
		p.LineTo(x-s*0.2, y-s)
		p.LineTo(x-s*0.2, y-s*0.35)
		p.LineTo(x-s, y-s*0.35)
		p.LineTo(x-s, y+s*0.35)
		p.LineTo(x-s*0.2, y+s*0.35)
		p.LineTo(x-s*0.2, y+s)
	}
	p.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, &p, nil, op)
}

// ---------------------------------------------------------------------------
// Text

type uiTextAlign int

const (
	uiAlignLeft uiTextAlign = iota
	uiAlignCenter
	uiAlignRight
)

func uiFace(bold bool, size float64) font.Face {
	kind := menuFontUI
	if bold {
		kind = menuFontUIBold
	}
	face, err := sourceMenuFace(kind, size)
	if err != nil {
		return nil
	}
	return face
}

func uiMeasure(bold bool, size float64, s string) float64 {
	face := uiFace(bold, size)
	if face == nil || s == "" {
		return 0
	}
	return float64(font.MeasureString(face, s).Ceil())
}

func uiCapHeight(size float64) float64 { return size * 0.7 }

// uiText draws s with its capital letters vertically centred on midY.
func uiText(dst *ebiten.Image, s string, bold bool, size float64, clr color.NRGBA, x, midY float64, align uiTextAlign) {
	if dst == nil || s == "" || clr.A == 0 {
		return
	}
	face := uiFace(bold, size)
	if face == nil {
		return
	}
	switch align {
	case uiAlignCenter:
		x -= uiMeasure(bold, size, s) / 2
	case uiAlignRight:
		x -= uiMeasure(bold, size, s)
	}
	baseline := midY + uiCapHeight(size)/2
	ebitentext.Draw(dst, s, face, int(math.Round(x)), int(math.Round(baseline)), clr)
}

// uiTextIn draws s centred vertically in r with the given horizontal alignment
// and padding, shortening it with "..." when it does not fit.
func uiTextIn(dst *ebiten.Image, s string, bold bool, size float64, clr color.NRGBA, r Rect, align uiTextAlign, pad float64) {
	s = uiFit(s, bold, size, r.W-2*pad)
	x := r.X + pad
	switch align {
	case uiAlignCenter:
		x = r.X + r.W/2
	case uiAlignRight:
		x = r.X + r.W - pad
	}
	uiText(dst, s, bold, size, clr, x, r.Y+r.H/2, align)
}

func uiFit(s string, bold bool, size, width float64) string {
	if width <= 0 || uiMeasure(bold, size, s) <= width {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && uiMeasure(bold, size, string(runes)+"...") > width {
		runes = runes[:len(runes)-1]
	}
	return strings.TrimSpace(string(runes)) + "..."
}

// uiParagraph draws wrapped text starting with the first line's caps centred
// on firstMidY. It returns the number of lines drawn.
func uiParagraph(dst *ebiten.Image, s string, bold bool, size float64, clr color.NRGBA, x, firstMidY, width, lineH float64, maxLines int) int {
	lines := wrapTextToWidth(s, width, func(v string) float64 { return uiMeasure(bold, size, v) })
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
		lines[maxLines-1] = uiFit(lines[maxLines-1]+" ...", bold, size, width)
	}
	for i, line := range lines {
		uiText(dst, line, bold, size, clr, x, firstMidY+float64(i)*lineH, uiAlignLeft)
	}
	return len(lines)
}

// ---------------------------------------------------------------------------
// Static components

// Header draws the screen title plate (20-76) with an optional breadcrumb.
func (ctx *uiCtx) Header(title, breadcrumb string) {
	if !ctx.drawing() {
		return
	}
	uiText(ctx.dst, title, true, uiSizeTitle, uiTextLight, uiMargin, 42, uiAlignLeft)
	if breadcrumb != "" {
		uiText(ctx.dst, breadcrumb, false, uiSizeHint, uiTextMuted, uiMargin+1, 68, uiAlignLeft)
	}
	uiFillRect(ctx.dst, Rect{X: uiMargin, Y: uiHeaderBottom + 4, W: ScreenWidth - 2*uiMargin, H: 1}, uiInk500)
}

// Page fills the whole screen with the ink page colour (alpha lets dimmed
// arena art show through).
func (ctx *uiCtx) Page(alpha float64) {
	if ctx.drawing() {
		uiFillRect(ctx.dst, Rect{W: ScreenWidth, H: ScreenHeight}, uiWithAlpha(uiInk900, alpha))
	}
}

// Panel is an ink-700 area with an optional title strip.
func (ctx *uiCtx) Panel(r Rect, title string) {
	if !ctx.drawing() {
		return
	}
	uiFillRound(ctx.dst, r, 6, uiInk700)
	if title != "" {
		uiText(ctx.dst, title, true, uiSizeButton, uiTextLight, r.X+14, r.Y+20, uiAlignLeft)
		uiFillRect(ctx.dst, Rect{X: r.X + 14, Y: r.Y + 36, W: r.W - 28, H: 1}, uiInk500)
	}
}

// Card is a paper card.
func (ctx *uiCtx) Card(r Rect, title string) {
	if !ctx.drawing() {
		return
	}
	uiFillRound(ctx.dst, Rect{X: r.X + 2, Y: r.Y + 3, W: r.W, H: r.H}, 8, color.NRGBA{A: 0x60})
	uiFillRound(ctx.dst, r, 8, uiPaper)
	if title != "" {
		uiText(ctx.dst, title, true, uiSizeSection, uiTextDark, r.X+14, r.Y+22, uiAlignLeft)
	}
}

// Label draws plain text on the page.
func (ctx *uiCtx) Label(s string, bold bool, size float64, clr color.NRGBA, x, midY float64, align uiTextAlign) {
	if ctx.drawing() {
		uiText(ctx.dst, s, bold, size, clr, x, midY, align)
	}
}

// InfoPanel shows a title and wrapped body (usually ctx.FocusInfo()).
func (ctx *uiCtx) InfoPanel(r Rect, title, body string) {
	if !ctx.drawing() {
		return
	}
	uiFillRound(ctx.dst, r, 6, uiInk700)
	uiFillRect(ctx.dst, Rect{X: r.X, Y: r.Y + 10, W: 4, H: r.H - 20}, uiOrange)
	if title == "" {
		title = "INFO"
	}
	uiText(ctx.dst, uiFit(title, true, uiSizeButton, r.W-32), true, uiSizeButton, uiTextLight, r.X+18, r.Y+24, uiAlignLeft)
	if body != "" {
		maxLines := int((r.H - 56) / 20)
		uiParagraph(ctx.dst, body, false, uiSizeBody, uiTextMuted, r.X+18, r.Y+54, r.W-34, 20, maxLines)
	}
}

// uiKeyHint is one entry of the key-hint bar.
type uiKeyHint struct {
	Key   string
	Label string
}

// KeyHintBar draws the key-hint strip (578-596). Keys "UP/DOWN" and
// "LEFT/RIGHT" are drawn as arrow glyphs.
func (ctx *uiCtx) KeyHintBar(hints []uiKeyHint) {
	if !ctx.drawing() {
		return
	}
	x := uiMargin
	mid := (uiHintTop + uiHintBottom) / 2
	for _, h := range hints {
		var chipW float64
		switch h.Key {
		case "UP/DOWN", "LEFT/RIGHT":
			chipW = 34
		default:
			chipW = uiMeasure(true, uiSizeHint, h.Key) + 12
		}
		chip := Rect{X: x, Y: uiHintTop, W: chipW, H: uiHintBottom - uiHintTop}
		uiFillRound(ctx.dst, chip, 3, uiInk500)
		switch h.Key {
		case "UP/DOWN":
			uiDrawArrow(ctx.dst, chip.X+11, mid, 11, uiArrowUp, uiTextLight)
			uiDrawArrow(ctx.dst, chip.X+23, mid, 11, uiArrowDown, uiTextLight)
		case "LEFT/RIGHT":
			uiDrawArrow(ctx.dst, chip.X+11, mid, 11, uiArrowLeft, uiTextLight)
			uiDrawArrow(ctx.dst, chip.X+23, mid, 11, uiArrowRight, uiTextLight)
		default:
			uiText(ctx.dst, h.Key, true, uiSizeHint, uiTextLight, chip.X+chipW/2, mid, uiAlignCenter)
		}
		x += chipW + 6
		uiText(ctx.dst, h.Label, false, uiSizeHint, uiTextMuted, x, mid, uiAlignLeft)
		x += uiMeasure(false, uiSizeHint, h.Label) + 18
	}
}

// Toasts draws the queued status messages above the action bar.
func (ctx *uiCtx) Toasts() {
	if !ctx.drawing() {
		return
	}
	y := uiActionTop - 8.0
	for i := len(ctx.st.toasts) - 1; i >= 0; i-- {
		t := ctx.st.toasts[i]
		w := math.Min(uiMeasure(false, uiSizeBody, t.text)+40, 640)
		r := Rect{X: ScreenWidth/2 - w/2, Y: y - 32, W: w, H: 32}
		alpha := math.Min(1, float64(t.ticks)/12)
		uiFillRound(ctx.dst, r, 5, uiWithAlpha(uiInk700, alpha))
		accent := uiNetBlue
		switch t.kind {
		case uiToastOK:
			accent = uiOK
		case uiToastWarning:
			accent = uiOrange
		case uiToastError:
			accent = uiDanger
		}
		uiFillRect(ctx.dst, Rect{X: r.X, Y: r.Y + 6, W: 4, H: r.H - 12}, uiWithAlpha(accent, alpha))
		uiTextIn(ctx.dst, t.text, false, uiSizeBody, uiWithAlpha(uiTextLight, alpha), Rect{X: r.X + 10, Y: r.Y, W: r.W - 14, H: r.H}, uiAlignLeft, 6)
		y -= 38
	}
}

// Badge is a small coloured tag (the NET badge uses uiNetBlue).
func (ctx *uiCtx) Badge(r Rect, text string, fill color.NRGBA) {
	if !ctx.drawing() {
		return
	}
	uiFillRound(ctx.dst, r, 4, fill)
	uiTextIn(ctx.dst, text, true, uiSizeHint, uiTextLight, r, uiAlignCenter, 4)
}

// ---------------------------------------------------------------------------
// Modal

// BeginModal dims everything below and opens a paper dialog; widgets until
// EndModal live on the modal layer and are the only interactive ones.
func (ctx *uiCtx) BeginModal(r Rect, title string) {
	if ctx.drawing() {
		uiFillRect(ctx.dst, Rect{W: ScreenWidth, H: ScreenHeight}, uiScrim)
	}
	ctx.layer++
	// A modal blocks the page while it exists, including in the frame it opens.
	if ctx.layer > ctx.topLayer {
		ctx.topLayer = ctx.layer
	}
	ctx.register(uiItem{id: "__modal_block", rect: Rect{W: ScreenWidth, H: ScreenHeight}})
	ctx.Card(r, title)
}

func (ctx *uiCtx) EndModal() {
	if ctx.layer > 0 {
		ctx.layer--
	}
}

// wrapTextToWidth splits text into lines no wider than maxWidth according to
// measure. Words longer than a line (URLs, codes) are broken by character.
func wrapTextToWidth(text string, maxWidth float64, measure func(string) float64) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if maxWidth <= 0 || measure == nil {
		return []string{text}
	}
	var lines []string
	current := ""
	flush := func() {
		if current != "" {
			lines = append(lines, current)
			current = ""
		}
	}
	for _, word := range strings.Fields(text) {
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if measure(candidate) <= maxWidth {
			current = candidate
			continue
		}
		flush()
		for measure(word) > maxWidth {
			runes := []rune(word)
			cut := 1
			for cut < len(runes) && measure(string(runes[:cut+1])) <= maxWidth {
				cut++
			}
			lines = append(lines, string(runes[:cut]))
			word = string(runes[cut:])
		}
		current = word
	}
	flush()
	return lines
}

// PlayerCard is the paper card of one player slot (Custom Game, online lobby,
// campaign briefing, Loadout, results): a colour strip on top (ink when the
// slot shows no character), the P# badge and an optional right-aligned kind.
func (ctx *uiCtx) PlayerCard(r Rect, slot, colour int, noColour bool, kind string) {
	if !ctx.drawing() {
		return
	}
	ctx.Card(r, "")
	strip := uiInk500
	if !noColour {
		strip = developedPlayerColorSwatch(colour)
		strip.A = 0xff
	}
	uiFillRound(ctx.dst, Rect{X: r.X, Y: r.Y, W: r.W, H: 8}, 4, strip)
	ctx.Badge(Rect{X: r.X + 12, Y: r.Y + 16, W: 34, H: 24}, "P"+strconv.Itoa(slot+1), uiInk900)
	if kind != "" {
		uiText(ctx.dst, kind, true, uiSizeHint, uiTextMutedDk, r.X+r.W-12, r.Y+28, uiAlignRight)
	}
}
