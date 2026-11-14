package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Mayhem UI kit: interactive widgets (see ui_kit_core.go).

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
