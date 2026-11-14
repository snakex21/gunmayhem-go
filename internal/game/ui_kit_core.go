package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
