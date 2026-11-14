package game

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	ebitentext "github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
)

// Mayhem UI kit: a small immediate-mode widget layer.
//
// A screen describes itself in one build function that is run twice per
// frame: once in the update pass (input, returns actions) and once in the
// draw pass (rendering). Every widget call takes its Rect and registers it in
// the frame registry, so drawing and hit-testing always share one geometry;
// there are no separate hit codes. Hover/press/focus are resolved against the
// registry of the previous update pass (topmost item wins, modal layers block
// everything below them). Keyboard: arrows move focus spatially (or adjust a
// focused horizontal control), Enter/Space activates, Tab cycles, Q/E switch
// tab bars and Esc is reported as Back.

type uiPass int

const (
	uiPassUpdate uiPass = iota
	uiPassDraw
)

type uiButtonKind int

const (
	uiButtonSecondary uiButtonKind = iota
	uiButtonPrimary
	uiButtonDanger
	uiButtonGhost
	uiButtonPaper
)

type uiToastKind int

const (
	uiToastInfo uiToastKind = iota
	uiToastOK
	uiToastWarning
	uiToastError
)

type uiItem struct {
	id         string
	rect       Rect
	layer      int
	focusable  bool
	horizontal bool // a focused item consumes Left/Right (sliders, steppers...)
	vertical   bool // a focused item consumes Up/Down (lists)
	disabled   bool
	infoTitle  string
	infoBody   string
}

// uiInput is one frame of input in virtual 900x600 coordinates.
type uiInput struct {
	X, Y                     float64
	Moved                    bool
	Down, Pressed, Released  bool
	Wheel                    float64
	Up, DownKey, Left, Right bool
	Enter, Esc, Tab, Shift   bool
	PrevTab, NextTab         bool
}

func (in uiInput) navigation() bool {
	return in.Up || in.DownKey || in.Left || in.Right || in.Tab || in.Enter
}

// pointerOnly drops the keyboard part of a frame, for screens where a text
// field currently owns the keyboard (typed letters must not switch tabs).
func (in uiInput) pointerOnly() uiInput {
	return uiInput{X: in.X, Y: in.Y, Moved: in.Moved, Down: in.Down, Pressed: in.Pressed, Released: in.Released, Wheel: in.Wheel}
}

type uiToast struct {
	text  string
	kind  uiToastKind
	ticks int
}

// uiState is the retained part of a kit screen.
type uiState struct {
	items    []uiItem
	hot      string
	active   string
	focus    string
	drag     string
	keyboard bool // last interaction came from the keyboard: show focus ring
	toasts   []uiToast
	frame    int // update passes run (caret blink)
}

func (st *uiState) reset() {
	*st = uiState{}
}

func (st *uiState) item(id string) (uiItem, bool) {
	if id == "" {
		return uiItem{}, false
	}
	for _, it := range st.items {
		if it.id == id {
			return it, true
		}
	}
	return uiItem{}, false
}

func (st *uiState) topLayer() int {
	top := 0
	for _, it := range st.items {
		if it.layer > top {
			top = it.layer
		}
	}
	return top
}

func (st *uiState) pushToast(text string, kind uiToastKind) {
	st.toasts = append(st.toasts, uiToast{text: text, kind: kind, ticks: 90})
	if len(st.toasts) > 3 {
		st.toasts = st.toasts[len(st.toasts)-3:]
	}
}

// uiCtx is one pass over a screen's build function.
type uiCtx struct {
	st       *uiState
	pass     uiPass
	dst      *ebiten.Image
	in       uiInput
	items    []uiItem
	layer    int
	topLayer int
	clicked  string // mouse press+release over the same item
	activate string // keyboard Enter/Space on the focused item
	adjustID string
	adjust   int // -1/+1 Left/Right for a focused horizontal item
	vadjust  int // -1/+1 Up/Down for a focused vertical item
	back     bool
	// readOnly registers every widget as inert (no hover, focus or action)
	// while still drawing its current value: a guest viewing the host's
	// online match settings.
	readOnly bool
}

func (st *uiState) begin(pass uiPass, dst *ebiten.Image, in uiInput) *uiCtx {
	ctx := &uiCtx{st: st, pass: pass, dst: dst, in: in, topLayer: st.topLayer()}
	if pass != uiPassUpdate {
		return ctx
	}
	if in.Moved || in.Pressed {
		st.keyboard = false
	}
	st.hot = ""
	for i := len(st.items) - 1; i >= 0; i-- {
		it := st.items[i]
		if it.layer == ctx.topLayer && it.rect.Contains(in.X, in.Y) {
			st.hot = it.id
			break
		}
	}
	hot, hotOK := st.item(st.hot)
	if in.Moved && hotOK && hot.focusable && !hot.disabled {
		st.focus = hot.id
	}
	if in.Pressed {
		st.active = ""
		if hotOK && !hot.disabled {
			st.active = hot.id
			if hot.focusable {
				st.focus = hot.id
			}
		}
	}
	if in.Released {
		if st.active != "" && st.active == st.hot {
			ctx.clicked = st.active
		}
		st.active = ""
	}

	// Keyboard focus must live on the top layer.
	if f, ok := st.item(st.focus); !ok || f.layer != ctx.topLayer {
		st.focus = ""
	}
	if in.navigation() {
		st.keyboard = true
	}
	focused, focusOK := st.item(st.focus)
	switch {
	case !focusOK && (in.Up || in.DownKey || in.Left || in.Right || in.Tab):
		st.focus = st.firstFocusable(ctx.topLayer)
	case focusOK && in.Enter && !focused.disabled:
		ctx.activate = focused.id
	case focusOK && in.Tab:
		st.focus = st.cycleFocus(ctx.topLayer, in.Shift)
	case focusOK && (in.Left || in.Right):
		if focused.horizontal && !focused.disabled {
			ctx.adjustID = focused.id
			if in.Left {
				ctx.adjust = -1
			} else {
				ctx.adjust = 1
			}
		} else {
			dx := 1.0
			if in.Left {
				dx = -1
			}
			st.moveFocus(ctx.topLayer, dx, 0)
		}
	case focusOK && (in.Up || in.DownKey):
		if focused.vertical && !focused.disabled {
			ctx.adjustID = focused.id
			if in.Up {
				ctx.vadjust = -1
			} else {
				ctx.vadjust = 1
			}
		} else {
			dy := 1.0
			if in.Up {
				dy = -1
			}
			st.moveFocus(ctx.topLayer, 0, dy)
		}
	}
	ctx.back = in.Esc
	return ctx
}

func (st *uiState) firstFocusable(layer int) string {
	for _, it := range st.items {
		if it.layer == layer && it.focusable && !it.disabled {
			return it.id
		}
	}
	return ""
}

func (st *uiState) cycleFocus(layer int, backwards bool) string {
	var list []string
	current := -1
	for _, it := range st.items {
		if it.layer == layer && it.focusable && !it.disabled {
			if it.id == st.focus {
				current = len(list)
			}
			list = append(list, it.id)
		}
	}
	if len(list) == 0 {
		return ""
	}
	step := 1
	if backwards {
		step = -1
	}
	return list[((current+step)%len(list)+len(list))%len(list)]
}

// moveFocus picks the nearest focusable item in direction (dx,dy) from the
// focused item's centre. Distance along the axis counts once and the
// perpendicular offset twice, so straight neighbours win over diagonal ones.
func (st *uiState) moveFocus(layer int, dx, dy float64) {
	from, ok := st.item(st.focus)
	if !ok {
		return
	}
	fx, fy := from.rect.X+from.rect.W/2, from.rect.Y+from.rect.H/2
	best, bestScore := "", math.Inf(1)
	for _, it := range st.items {
		if it.id == from.id || it.layer != layer || !it.focusable || it.disabled {
			continue
		}
		cx, cy := it.rect.X+it.rect.W/2, it.rect.Y+it.rect.H/2
		along := (cx-fx)*dx + (cy-fy)*dy
		if along <= 1 {
			continue
		}
		across := math.Abs((cx-fx)*dy) + math.Abs((cy-fy)*dx)
		if score := along + 2*across; score < bestScore {
			best, bestScore = it.id, score
		}
	}
	if best != "" {
		st.focus = best
	}
}

func (ctx *uiCtx) end() {
	if ctx.pass != uiPassUpdate {
		return
	}
	ctx.st.items = ctx.items
	ctx.st.frame++
	if _, ok := ctx.st.item(ctx.st.focus); !ok {
		ctx.st.focus = ""
	}
	if _, ok := ctx.st.item(ctx.st.drag); !ok {
		ctx.st.drag = ""
	}
	alive := ctx.st.toasts[:0]
	for _, t := range ctx.st.toasts {
		t.ticks--
		if t.ticks > 0 {
			alive = append(alive, t)
		}
	}
	ctx.st.toasts = alive
}

func (ctx *uiCtx) drawing() bool { return ctx.pass == uiPassDraw && ctx.dst != nil }

// Back reports Esc in the update pass.
func (ctx *uiCtx) Back() bool { return ctx.pass == uiPassUpdate && ctx.back }

func (ctx *uiCtx) register(it uiItem) {
	it.layer = ctx.layer
	if ctx.readOnly {
		it.disabled = true
		it.focusable = false
	}
	ctx.items = append(ctx.items, it)
}

func (ctx *uiCtx) live(id string) bool { return ctx.layer == ctx.topLayer }

func (ctx *uiCtx) hot(id string) bool { return ctx.live(id) && !ctx.readOnly && ctx.st.hot == id }
func (ctx *uiCtx) pressed(id string) bool {
	return ctx.live(id) && !ctx.readOnly && ctx.st.active == id
}
func (ctx *uiCtx) focused(id string) bool {
	return ctx.live(id) && ctx.st.focus == id
}
func (ctx *uiCtx) showFocus(id string) bool { return ctx.focused(id) && ctx.st.keyboard }

// triggered is true in the update pass when id was clicked or activated.
func (ctx *uiCtx) triggered(id string) bool {
	if ctx.pass != uiPassUpdate || !ctx.live(id) || ctx.readOnly {
		return false
	}
	return ctx.clicked == id || ctx.activate == id
}

// InfoID is the item whose info is shown: the keyboard-focused one, else the
// hovered one, else the focused one.
func (ctx *uiCtx) InfoID() string {
	id := ctx.st.hot
	if ctx.st.keyboard || id == "" {
		id = ctx.st.focus
	}
	return id
}

// FocusInfo returns the info text of the focused (keyboard) or hovered item.
func (ctx *uiCtx) FocusInfo() (title, body string) {
	if it, ok := ctx.st.item(ctx.InfoID()); ok {
		return it.infoTitle, it.infoBody
	}
	return "", ""
}

// FocusedID exposes the keyboard focus (tests, rows that highlight).
func (ctx *uiCtx) FocusedID() string { return ctx.st.focus }

// ClickedAt reports where id was clicked with the mouse this update pass
// (controls with several parts, e.g. the typed value of a stepper).
func (ctx *uiCtx) ClickedAt(id string) (x, y float64, ok bool) {
	if ctx.pass != uiPassUpdate || !ctx.live(id) || ctx.clicked != id {
		return 0, 0, false
	}
	return ctx.in.X, ctx.in.Y, true
}

// PressedOutside is true in the update pass when the mouse went down on
// anything but id (used to drop text-field focus).
func (ctx *uiCtx) PressedOutside(id string) bool {
	return ctx.pass == uiPassUpdate && ctx.in.Pressed && ctx.st.hot != id
}

// Block registers a non-interactive area that owns the pointer, so widgets
// below it never become hot (a region handled by legacy hit-testing).
func (ctx *uiCtx) Block(id string, r Rect) {
	ctx.register(uiItem{id: id, rect: r})
}

// ---------------------------------------------------------------------------
// Input

func (g *Game) readUIInput(st *uiState) uiInput {
	mx, my := g.cursorPos()
	x, y := float64(mx), float64(my)
	in := uiInput{X: x, Y: y}
	in.Moved = x != g.uiLastCursorX || y != g.uiLastCursorY
	g.uiLastCursorX, g.uiLastCursorY = x, y
	in.Down = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	in.Pressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	in.Released = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	_, in.Wheel = ebiten.Wheel()
	in.Up = inpututil.IsKeyJustPressed(ebiten.KeyArrowUp)
	in.DownKey = inpututil.IsKeyJustPressed(ebiten.KeyArrowDown)
	in.Left = inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft)
	in.Right = inpututil.IsKeyJustPressed(ebiten.KeyArrowRight)
	in.Enter = inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace)
	in.Esc = inpututil.IsKeyJustPressed(ebiten.KeyEscape)
	in.Tab = inpututil.IsKeyJustPressed(ebiten.KeyTab)
	in.Shift = ebiten.IsKeyPressed(ebiten.KeyShift)
	in.PrevTab = inpututil.IsKeyJustPressed(ebiten.KeyQ)
	in.NextTab = inpututil.IsKeyJustPressed(ebiten.KeyE)
	return in
}

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

// ---------------------------------------------------------------------------
// Interactive components

type uiButtonOpts struct {
	Kind      uiButtonKind
	Selected  bool
	Disabled  bool
	InfoTitle string
	InfoBody  string
	Size      float64
}

// Button returns true when clicked or activated with Enter/Space.
func (ctx *uiCtx) Button(id string, r Rect, label string, o uiButtonOpts) bool {
	ctx.register(uiItem{id: id, rect: r, focusable: true, disabled: o.Disabled, infoTitle: o.InfoTitle, infoBody: o.InfoBody})
	if ctx.drawing() {
		ctx.drawButton(id, r, label, o)
	}
	return !o.Disabled && ctx.triggered(id)
}

func uiButtonColors(kind uiButtonKind, selected, hover, pressed bool) (fill, border, text color.NRGBA) {
	switch {
	case selected:
		fill, border, text = uiOrange, uiOrangePress, uiTextDark
		if hover {
			fill = uiOrangeHover
		}
	case kind == uiButtonPrimary:
		fill, border, text = uiOrange, uiOrangePress, uiTextDark
		if hover {
			fill = uiOrangeHover
		}
		if pressed {
			fill = uiOrangePress
		}
	case kind == uiButtonDanger:
		fill, border, text = uiDanger, color.NRGBA{R: 0x9e, G: 0x25, B: 0x1c, A: 0xff}, uiTextLight
		if hover {
			fill = uiDangerHover
		}
	case kind == uiButtonGhost:
		fill, border, text = color.NRGBA{}, color.NRGBA{}, uiTextMuted
		if hover {
			text = uiTextLight
		}
	case kind == uiButtonPaper:
		fill, border, text = uiPaper, uiPaperShade, uiTextDark
		if hover {
			fill = uiPaperHover
		}
		if pressed {
			fill = uiPaperShade
		}
	default:
		fill, border, text = uiInk700, uiInk500, uiTextLight
		if hover {
			fill = uiInk600
		}
		if pressed {
			fill = uiInk900
		}
	}
	return fill, border, text
}

func (ctx *uiCtx) drawButton(id string, r Rect, label string, o uiButtonOpts) {
	hover := !o.Disabled && ctx.hot(id)
	pressed := hover && ctx.pressed(id)
	fill, border, text := uiButtonColors(o.Kind, o.Selected, hover, pressed)
	alpha := 1.0
	if o.Disabled {
		alpha = 0.4
	}
	draw := r
	if pressed {
		draw.Y++
	}
	if border.A > 0 {
		uiFillRound(ctx.dst, draw, 4, uiWithAlpha(border, alpha))
		uiFillRound(ctx.dst, uiInset(draw, 2), 3, uiWithAlpha(fill, alpha))
	} else if fill.A > 0 {
		uiFillRound(ctx.dst, draw, 4, uiWithAlpha(fill, alpha))
	}
	if hover && !o.Selected && o.Kind != uiButtonPrimary {
		uiFillRect(ctx.dst, Rect{X: draw.X + 6, Y: draw.Y + draw.H - 5, W: draw.W - 12, H: 3}, uiOrange)
	}
	size := o.Size
	if size == 0 {
		size = uiSizeButton
	}
	uiTextIn(ctx.dst, label, true, size, uiWithAlpha(text, alpha), draw, uiAlignCenter, 10)
	if ctx.showFocus(id) {
		uiFocusRing(ctx.dst, r)
	}
}

// TabBar draws paper tabs on ink; it returns the (possibly changed) selected
// index. Q/E switch tabs from anywhere on the screen.
func (ctx *uiCtx) TabBar(id string, r Rect, labels []string, selected int) int {
	n := len(labels)
	if n == 0 {
		return selected
	}
	w := r.W / float64(n)
	out := selected
	for i, label := range labels {
		tid := id + "/" + label
		tr := Rect{X: r.X + float64(i)*w, Y: r.Y, W: w - 4, H: r.H}
		ctx.register(uiItem{id: tid, rect: tr, focusable: true, infoTitle: label + " TAB", infoBody: "Q / E switch tabs."})
		if ctx.triggered(tid) {
			out = i
		}
		if ctx.drawing() {
			sel := i == selected
			hover := ctx.hot(tid)
			fill, text := uiInk700, uiTextMuted
			if hover {
				fill, text = uiInk600, uiTextLight
			}
			if sel {
				fill, text = uiPaper, uiTextDark
			}
			uiFillRound(ctx.dst, tr, 4, fill)
			if sel {
				uiFillRect(ctx.dst, Rect{X: tr.X, Y: tr.Y + tr.H - 4, W: tr.W, H: 4}, uiOrange)
			}
			uiTextIn(ctx.dst, label, true, uiSizeButton, text, tr, uiAlignCenter, 6)
			if ctx.showFocus(tid) {
				uiFocusRing(ctx.dst, tr)
			}
		}
	}
	if ctx.pass == uiPassUpdate && ctx.layer == ctx.topLayer {
		if ctx.in.PrevTab {
			out = (selected - 1 + n) % n
		}
		if ctx.in.NextTab {
			out = (selected + 1) % n
		}
	}
	return out
}

type uiTileOpts struct {
	Selected  bool
	Disabled  bool
	Locked    bool
	InfoTitle string
	InfoBody  string
	Art       func(dst *ebiten.Image, inner Rect)
}

// Tile is an image + label card (guns, perks, maps, missions...). An empty
// label gives the whole tile to the art (colour swatches, hats...).
func (ctx *uiCtx) Tile(id string, r Rect, label string, o uiTileOpts) bool {
	ctx.register(uiItem{id: id, rect: r, focusable: true, disabled: o.Disabled, infoTitle: o.InfoTitle, infoBody: o.InfoBody})
	if ctx.drawing() {
		hover := !o.Disabled && ctx.hot(id)
		fill := uiPaper
		if hover {
			fill = uiPaperHover
		}
		alpha := 1.0
		if o.Disabled {
			alpha = 0.45
		}
		uiFillRound(ctx.dst, r, 6, uiWithAlpha(fill, alpha))
		labelH := 22.0
		inner := Rect{X: r.X + 6, Y: r.Y + 6, W: r.W - 12, H: r.H - labelH - 10}
		if label == "" {
			inner = uiInset(r, 4)
		}
		if o.Art != nil {
			o.Art(ctx.dst, inner)
		}
		if label != "" {
			uiTextIn(ctx.dst, label, true, uiSizeHint, uiWithAlpha(uiTextDark, alpha), Rect{X: r.X, Y: r.Y + r.H - labelH - 2, W: r.W, H: labelH}, uiAlignCenter, 4)
		}
		if o.Locked {
			badge := Rect{X: r.X + r.W - 26, Y: r.Y + 4, W: 22, H: 18}
			uiFillRound(ctx.dst, badge, 3, uiInk900)
			uiTextIn(ctx.dst, "X", true, uiSizeHint, uiDanger, badge, uiAlignCenter, 2)
		}
		if o.Selected {
			uiStrokeRound(ctx.dst, uiInset(r, 1.5), 6, 3, uiOrange)
		}
		if ctx.showFocus(id) {
			uiFocusRing(ctx.dst, r)
		}
	}
	return !o.Disabled && ctx.triggered(id)
}

// Segmented is one focusable control made of N options; Left/Right step.
func (ctx *uiCtx) Segmented(id string, r Rect, labels []string, selected int, disabled bool, infoTitle, infoBody string) int {
	ctx.register(uiItem{id: id, rect: r, focusable: true, horizontal: true, disabled: disabled, infoTitle: infoTitle, infoBody: infoBody})
	n := len(labels)
	if n == 0 {
		return selected
	}
	w := r.W / float64(n)
	segAt := func(x float64) int {
		i := int((x - r.X) / w)
		if i < 0 {
			i = 0
		}
		if i >= n {
			i = n - 1
		}
		return i
	}
	out := selected
	if !disabled && ctx.pass == uiPassUpdate {
		if ctx.clicked == id && ctx.live(id) {
			out = segAt(ctx.in.X)
		}
		if ctx.adjustID == id {
			out = selected + ctx.adjust
			if out < 0 {
				out = 0
			}
			if out >= n {
				out = n - 1
			}
		}
		if ctx.activate == id {
			out = (selected + 1) % n
		}
	}
	if ctx.drawing() {
		alpha := 1.0
		if disabled {
			alpha = 0.4
		}
		uiFillRound(ctx.dst, r, 5, uiWithAlpha(uiInk500, alpha))
		hoverSeg := -1
		if !disabled && ctx.hot(id) {
			hoverSeg = segAt(ctx.in.X)
		}
		for i, label := range labels {
			sr := Rect{X: r.X + float64(i)*w + 2, Y: r.Y + 2, W: w - 4, H: r.H - 4}
			fill, text := uiInk700, uiTextLight
			if i == hoverSeg {
				fill = uiInk600
			}
			if i == selected {
				fill, text = uiOrange, uiTextDark
			}
			uiFillRound(ctx.dst, sr, 3, uiWithAlpha(fill, alpha))
			uiTextIn(ctx.dst, label, true, uiSizeBody, uiWithAlpha(text, alpha), sr, uiAlignCenter, 4)
		}
		if ctx.showFocus(id) {
			uiFocusRing(ctx.dst, r)
		}
	}
	return out
}

// Stepper is "- value +"; it returns -1, 0 or +1.
func (ctx *uiCtx) Stepper(id string, r Rect, value string, canDec, canInc, disabled bool, infoTitle, infoBody string) int {
	ctx.register(uiItem{id: id, rect: r, focusable: true, horizontal: true, disabled: disabled, infoTitle: infoTitle, infoBody: infoBody})
	bw := math.Min(40, r.W/4)
	dec := Rect{X: r.X, Y: r.Y, W: bw, H: r.H}
	inc := Rect{X: r.X + r.W - bw, Y: r.Y, W: bw, H: r.H}
	delta := 0
	if !disabled && ctx.pass == uiPassUpdate {
		if ctx.clicked == id && ctx.live(id) {
			if dec.Contains(ctx.in.X, ctx.in.Y) {
				delta = -1
			} else if inc.Contains(ctx.in.X, ctx.in.Y) {
				delta = 1
			}
		}
		if ctx.adjustID == id {
			delta = ctx.adjust
		}
		if ctx.activate == id {
			delta = 1
		}
		if (delta < 0 && !canDec) || (delta > 0 && !canInc) {
			delta = 0
		}
	}
	if ctx.drawing() {
		alpha := 1.0
		if disabled {
			alpha = 0.4
		}
		uiFillRound(ctx.dst, r, 5, uiWithAlpha(uiInk700, alpha))
		uiStrokeRound(ctx.dst, r, 5, 2, uiWithAlpha(uiInk500, alpha))
		for i, br := range []Rect{dec, inc} {
			enabled := (i == 0 && canDec) || (i == 1 && canInc)
			fill := uiInk500
			if enabled && !disabled && ctx.hot(id) && br.Contains(ctx.in.X, ctx.in.Y) {
				fill = uiOrange
			}
			a := alpha
			if !enabled {
				a *= 0.4
			}
			uiFillRound(ctx.dst, uiInset(br, 3), 3, uiWithAlpha(fill, a))
			dir := uiArrowLeft
			if i == 1 {
				dir = uiArrowRight
			}
			uiDrawArrow(ctx.dst, br.X+br.W/2, br.Y+br.H/2, 12, dir, uiWithAlpha(uiTextLight, a))
		}
		uiTextIn(ctx.dst, value, true, uiSizeBody, uiWithAlpha(uiTextLight, alpha), Rect{X: dec.X + bw, Y: r.Y, W: r.W - 2*bw, H: r.H}, uiAlignCenter, 4)
		if ctx.showFocus(id) {
			uiFocusRing(ctx.dst, r)
		}
	}
	return delta
}

// Toggle is an ON/OFF pill; it returns the new value.
func (ctx *uiCtx) Toggle(id string, r Rect, on, disabled bool, infoTitle, infoBody string) bool {
	ctx.register(uiItem{id: id, rect: r, focusable: true, horizontal: true, disabled: disabled, infoTitle: infoTitle, infoBody: infoBody})
	out := on
	if !disabled && ctx.pass == uiPassUpdate {
		if ctx.triggered(id) {
			out = !on
		}
		if ctx.adjustID == id {
			out = ctx.adjust > 0
		}
	}
	if ctx.drawing() {
		alpha := 1.0
		if disabled {
			alpha = 0.4
		}
		pill := Rect{X: r.X, Y: r.Y + r.H/2 - 13, W: 54, H: 26}
		track := uiInk500
		if on {
			track = uiOrange
		}
		if !disabled && ctx.hot(id) && !on {
			track = color.NRGBA{R: 0x4a, G: 0x52, B: 0x5e, A: 0xff}
		}
		uiFillRound(ctx.dst, pill, 13, uiWithAlpha(track, alpha))
		knobX := pill.X + 3
		if on {
			knobX = pill.X + pill.W - 23
		}
		uiFillRound(ctx.dst, Rect{X: knobX, Y: pill.Y + 3, W: 20, H: 20}, 10, uiWithAlpha(uiPaper, alpha))
		state := "OFF"
		clr := uiTextMuted
		if on {
			state, clr = "ON", uiTextLight
		}
		uiText(ctx.dst, state, true, uiSizeBody, uiWithAlpha(clr, alpha), pill.X+pill.W+12, r.Y+r.H/2, uiAlignLeft)
		if ctx.showFocus(id) {
			uiFocusRing(ctx.dst, Rect{X: r.X, Y: r.Y, W: math.Min(r.W, 110), H: r.H})
		}
	}
	return out
}

// Slider edits a 0..1 value. changed reports any movement this frame; done is
// true when an edit is finished (mouse released or a key step), so callers can
// persist without writing on every drag frame.
func (ctx *uiCtx) Slider(id string, r Rect, value float64, disabled bool, infoTitle, infoBody string) (out float64, changed, done bool) {
	ctx.register(uiItem{id: id, rect: r, focusable: true, horizontal: true, disabled: disabled, infoTitle: infoTitle, infoBody: infoBody})
	out = clampOptionVolume(value)
	track := Rect{X: r.X + 10, Y: r.Y + r.H/2 - 4, W: r.W - 20, H: 8}
	fromX := func(x float64) float64 { return optionSliderValueFromX(track, x) }
	if !disabled && ctx.pass == uiPassUpdate {
		st := ctx.st
		if ctx.in.Pressed && st.active == id && ctx.live(id) {
			st.drag = id
		}
		if st.drag == id {
			out = fromX(ctx.in.X)
			if !ctx.in.Down || ctx.in.Released {
				st.drag = ""
				done = true
			}
		}
		if ctx.adjustID == id {
			out = clampOptionVolume(math.Round((value+0.05*float64(ctx.adjust))*20) / 20)
			done = true
		}
		changed = out != value
	}
	if ctx.drawing() {
		alpha := 1.0
		if disabled {
			alpha = 0.4
		}
		uiFillRound(ctx.dst, track, 4, uiWithAlpha(uiInk500, alpha))
		if out > 0 {
			uiFillRound(ctx.dst, Rect{X: track.X, Y: track.Y, W: track.W * out, H: track.H}, 4, uiWithAlpha(uiOrange, alpha))
		}
		knob := uiPaper
		if ctx.hot(id) || ctx.st.drag == id {
			knob = uiPaperHover
		}
		kx := track.X + track.W*out
		uiFillRound(ctx.dst, Rect{X: kx - 9, Y: r.Y + r.H/2 - 9, W: 18, H: 18}, 9, uiWithAlpha(knob, alpha))
		uiStrokeRound(ctx.dst, Rect{X: kx - 9, Y: r.Y + r.H/2 - 9, W: 18, H: 18}, 9, 2, uiWithAlpha(uiOrangePress, alpha))
		if ctx.showFocus(id) {
			uiFocusRing(ctx.dst, r)
		}
	}
	return out, changed, done
}

// List draws selectable rows with wheel scrolling; it returns the clicked or
// keyboard-picked row index, or -1. *scroll is the first visible row.
func (ctx *uiCtx) List(id string, r Rect, rows []string, selected int, scroll *int) int {
	const rowH = 32.0
	visible := int(r.H / rowH)
	if visible < 1 {
		visible = 1
	}
	maxScroll := len(rows) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	clampScroll := func() {
		if *scroll > maxScroll {
			*scroll = maxScroll
		}
		if *scroll < 0 {
			*scroll = 0
		}
	}
	clampScroll()
	ctx.register(uiItem{id: id, rect: r, focusable: true, vertical: true})
	picked := -1
	if ctx.pass == uiPassUpdate && ctx.live(id) {
		if ctx.st.hot == id && ctx.in.Wheel != 0 {
			if ctx.in.Wheel > 0 {
				*scroll--
			} else {
				*scroll++
			}
			clampScroll()
		}
		if ctx.clicked == id {
			row := *scroll + int((ctx.in.Y-r.Y)/rowH)
			if row >= 0 && row < len(rows) {
				picked = row
			}
		}
		if ctx.adjustID == id && ctx.vadjust != 0 && len(rows) > 0 {
			picked = selected + ctx.vadjust
			if picked < 0 {
				picked = 0
			}
			if picked >= len(rows) {
				picked = len(rows) - 1
			}
			if picked < *scroll {
				*scroll = picked
			}
			if picked >= *scroll+visible {
				*scroll = picked - visible + 1
			}
		}
	}
	if ctx.drawing() {
		uiFillRound(ctx.dst, r, 5, uiInk700)
		hoverRow := -1
		if ctx.hot(id) {
			hoverRow = *scroll + int((ctx.in.Y-r.Y)/rowH)
		}
		for v := 0; v < visible && *scroll+v < len(rows); v++ {
			i := *scroll + v
			rr := Rect{X: r.X + 4, Y: r.Y + 4 + float64(v)*rowH, W: r.W - 8, H: rowH - 4}
			fill, text := uiInk700, uiTextLight
			if i == hoverRow {
				fill = uiInk600
			}
			if i == selected {
				fill, text = uiOrange, uiTextDark
			}
			uiFillRound(ctx.dst, rr, 3, fill)
			uiTextIn(ctx.dst, rows[i], true, uiSizeBody, text, rr, uiAlignLeft, 10)
		}
		if maxScroll > 0 {
			barH := r.H * float64(visible) / float64(len(rows))
			barY := r.Y + (r.H-barH)*float64(*scroll)/float64(maxScroll)
			uiFillRound(ctx.dst, Rect{X: r.X + r.W - 5, Y: barY, W: 3, H: barH}, 1.5, uiInk500)
		}
		if ctx.showFocus(id) {
			uiFocusRing(ctx.dst, r)
		}
	}
	return picked
}

type uiModeTileOpts struct {
	Primary   bool // orange tile (the main mode)
	Disabled  bool
	Subtitle  string
	Tag       string
	TagFill   color.NRGBA
	TitleSize float64
	InfoTitle string
	InfoBody  string
	// Art draws an illustration into the tile (alpha is 1 or the disabled fade).
	Art func(dst *ebiten.Image, r Rect, alpha float64)
}

// ModeTile is a large paper (or orange) tile with a heavy title, used for the
// main-menu modes. A disabled tile stays visible and keeps its info text.
func (ctx *uiCtx) ModeTile(id string, r Rect, title string, o uiModeTileOpts) bool {
	ctx.register(uiItem{id: id, rect: r, focusable: true, disabled: o.Disabled, infoTitle: o.InfoTitle, infoBody: o.InfoBody})
	if ctx.drawing() {
		hover := !o.Disabled && ctx.hot(id)
		pressed := hover && ctx.pressed(id)
		alpha := 1.0
		if o.Disabled {
			alpha = 0.55
		}
		fill := uiPaper
		if hover {
			fill = uiPaperHover
		}
		if o.Primary {
			fill = uiOrange
			if hover {
				fill = uiOrangeHover
			}
		}
		draw := r
		if pressed {
			draw.Y++
		}
		uiFillRound(ctx.dst, Rect{X: draw.X + 3, Y: draw.Y + 4, W: draw.W, H: draw.H}, 8, color.NRGBA{A: uint8(0x70 * alpha)})
		uiFillRound(ctx.dst, draw, 8, uiWithAlpha(uiTextDark, alpha))
		uiFillRound(ctx.dst, uiInset(draw, 3), 6, uiWithAlpha(fill, alpha))
		if o.Art != nil {
			o.Art(ctx.dst, uiInset(draw, 3), alpha)
		}
		size := o.TitleSize
		if size == 0 {
			size = 30
		}
		pad := 18.0
		subMid := draw.Y + draw.H - pad - 8
		titleMid := subMid - 12 - size*0.42
		if o.Subtitle == "" {
			titleMid = draw.Y + draw.H - pad - size*0.36
		}
		sub := uiTextMutedDk
		if o.Primary {
			sub = color.NRGBA{R: 0x2a, G: 0x1a, B: 0x00, A: 0xff}
		}
		uiText(ctx.dst, uiFit(title, true, size, draw.W-2*pad), true, size, uiWithAlpha(uiTextDark, alpha), draw.X+pad, titleMid, uiAlignLeft)
		if o.Subtitle != "" {
			uiText(ctx.dst, uiFit(o.Subtitle, false, uiSizeBody, draw.W-2*pad), false, uiSizeBody, uiWithAlpha(sub, alpha), draw.X+pad, subMid, uiAlignLeft)
		}
		if o.Tag != "" {
			tagFill := o.TagFill
			if tagFill.A == 0 {
				tagFill = uiInk900
			}
			w := uiMeasure(true, uiSizeHint, o.Tag) + 16
			ctx.Badge(Rect{X: draw.X + draw.W - w - 12, Y: draw.Y + 12, W: w, H: 22}, o.Tag, uiWithAlpha(tagFill, math.Max(alpha, 0.8)))
		}
		if hover && !o.Primary {
			uiFillRect(ctx.dst, Rect{X: draw.X + 10, Y: draw.Y + draw.H - 8, W: draw.W - 20, H: 3}, uiOrange)
		}
		if ctx.showFocus(id) {
			uiFocusRing(ctx.dst, r)
		}
	}
	return !o.Disabled && ctx.triggered(id)
}

// TextField draws an editable one-line field. The caller owns the text and
// the typing (focused says whether this field currently receives characters);
// it returns true when the field is clicked or activated to start editing.
func (ctx *uiCtx) TextField(id string, r Rect, text, placeholder string, focused, disabled bool, infoTitle, infoBody string) bool {
	ctx.register(uiItem{id: id, rect: r, focusable: true, disabled: disabled, infoTitle: infoTitle, infoBody: infoBody})
	if ctx.drawing() {
		alpha := 1.0
		if disabled {
			alpha = 0.45
		}
		border := uiInk500
		if focused {
			border = uiOrange
		} else if !disabled && ctx.hot(id) {
			border = uiOrangePress
		}
		uiFillRound(ctx.dst, r, 4, uiWithAlpha(border, alpha))
		uiFillRound(ctx.dst, uiInset(r, 2), 3, uiWithAlpha(uiPaperHover, alpha))
		shown, clr := text, uiTextDark
		if shown == "" && !focused {
			shown, clr = placeholder, uiTextMutedDk
		}
		inner := Rect{X: r.X + 8, Y: r.Y, W: r.W - 16, H: r.H}
		shown = uiFit(shown, true, uiSizeBody, inner.W-8)
		uiTextIn(ctx.dst, shown, true, uiSizeBody, uiWithAlpha(clr, alpha), inner, uiAlignLeft, 0)
		if focused && (ctx.st.frame/15)%2 == 0 {
			cx := inner.X + uiMeasure(true, uiSizeBody, shown) + 2
			uiFillRect(ctx.dst, Rect{X: cx, Y: r.Y + 7, W: 2, H: r.H - 14}, uiTextDark)
		}
		if ctx.showFocus(id) {
			uiFocusRing(ctx.dst, r)
		}
	}
	return !disabled && ctx.triggered(id)
}

// SettingRow draws a label row on ink (orange marker while its control is
// hovered or keyboard-focused) and returns the rect for the control, which
// starts at labelW from the row's left edge.
func (ctx *uiCtx) SettingRow(row Rect, label, controlID string, labelW float64) Rect {
	if ctx.drawing() {
		fill := uiWithAlpha(uiInk600, 0.55)
		active := controlID != "" && (ctx.st.hot == controlID || (ctx.st.focus == controlID && ctx.st.keyboard))
		if active {
			fill = uiInk600
		}
		uiFillRound(ctx.dst, row, 5, fill)
		if active {
			uiFillRect(ctx.dst, Rect{X: row.X, Y: row.Y + 6, W: 4, H: row.H - 12}, uiOrange)
		}
		uiText(ctx.dst, label, true, uiSizeButton, uiTextLight, row.X+14, row.Y+row.H/2, uiAlignLeft)
	}
	return Rect{X: row.X + labelW, Y: row.Y + 3, W: row.W - labelW - 6, H: row.H - 6}
}

type uiRowOpts struct {
	Selected  bool
	Disabled  bool
	InfoTitle string
	InfoBody  string
	// Draw renders the row's content on top of its background; fg is the
	// text colour that contrasts with that background.
	Draw func(dst *ebiten.Image, r Rect, fg color.NRGBA, alpha float64)
}

// Row is one focusable list row whose content the caller draws (server
// browser, player lists). It returns true when clicked or activated.
func (ctx *uiCtx) Row(id string, r Rect, o uiRowOpts) bool {
	ctx.register(uiItem{id: id, rect: r, focusable: true, disabled: o.Disabled, infoTitle: o.InfoTitle, infoBody: o.InfoBody})
	if ctx.drawing() {
		hover := !o.Disabled && ctx.hot(id)
		alpha := 1.0
		if o.Disabled {
			alpha = 0.5
		}
		fill, fg := uiInk600, uiTextLight
		if hover {
			fill = uiInk500
		}
		if o.Selected {
			fill, fg = uiOrange, uiTextDark
		}
		uiFillRound(ctx.dst, r, 4, uiWithAlpha(fill, alpha))
		if o.Draw != nil {
			o.Draw(ctx.dst, r, fg, alpha)
		}
		if ctx.showFocus(id) {
			uiFocusRing(ctx.dst, r)
		}
	}
	return !o.Disabled && ctx.triggered(id)
}
