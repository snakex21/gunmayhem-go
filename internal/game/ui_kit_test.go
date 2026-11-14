package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// kitDemo is a small screen exercising every kit component.
type kitDemo struct {
	clicks   map[string]int
	tab      int
	seg      int
	step     int
	toggle   bool
	slider   float64
	listSel  int
	scroll   int
	modal    bool
	disabled bool
}

func (d *kitDemo) build(ctx *uiCtx) {
	ctx.Page(1)
	ctx.Header("KIT", "TEST / KIT")
	d.tab = ctx.TabBar("tabs", Rect{X: 440, Y: 30, W: 440, H: 40}, []string{"A", "B", "C"}, d.tab)
	if ctx.Button("b1", Rect{X: 20, Y: 100, W: 150, H: 44}, "ONE", uiButtonOpts{}) {
		d.clicks["b1"]++
	}
	if ctx.Button("b2", Rect{X: 200, Y: 100, W: 150, H: 44}, "TWO", uiButtonOpts{Kind: uiButtonPrimary}) {
		d.clicks["b2"]++
	}
	if ctx.Button("b3", Rect{X: 20, Y: 160, W: 150, H: 44}, "OFF", uiButtonOpts{Kind: uiButtonDanger, Disabled: d.disabled}) {
		d.clicks["b3"]++
	}
	if ctx.Tile("tile", Rect{X: 200, Y: 160, W: 90, H: 90}, "TILE", uiTileOpts{Selected: true, Locked: true}) {
		d.clicks["tile"]++
	}
	d.seg = ctx.Segmented("seg", Rect{X: 20, Y: 270, W: 300, H: 36}, []string{"X", "Y", "Z"}, d.seg, false, "SEG", "info")
	d.step += ctx.Stepper("step", Rect{X: 20, Y: 320, W: 200, H: 36}, "VALUE", d.step > 0, d.step < 3, false, "", "")
	d.toggle = ctx.Toggle("toggle", Rect{X: 20, Y: 370, W: 200, H: 36}, d.toggle, false, "", "")
	d.slider, _, _ = ctx.Slider("slider", Rect{X: 20, Y: 420, W: 300, H: 36}, d.slider, false, "", "")
	if picked := ctx.List("list", Rect{X: 400, Y: 270, W: 200, H: 96}, []string{"a", "b", "c", "d", "e"}, d.listSel, &d.scroll); picked >= 0 {
		d.listSel = picked
	}
	ctx.Panel(Rect{X: 640, Y: 100, W: 240, H: 100}, "PANEL")
	ctx.Card(Rect{X: 640, Y: 220, W: 240, H: 100}, "CARD")
	title, body := ctx.FocusInfo()
	ctx.InfoPanel(Rect{X: 640, Y: 340, W: 240, H: 160}, title, body)
	ctx.Badge(Rect{X: 820, Y: 80, W: 50, H: 18}, "NET", uiNetBlue)
	ctx.KeyHintBar([]uiKeyHint{{"ESC", "Back"}, {"UP/DOWN", "Move"}, {"LEFT/RIGHT", "Change"}})
	ctx.Toasts()
	if d.modal {
		ctx.BeginModal(Rect{X: 250, Y: 200, W: 400, H: 200}, "MODAL")
		if ctx.Button("m.ok", Rect{X: 480, Y: 340, W: 150, H: 44}, "OK", uiButtonOpts{Kind: uiButtonPrimary}) {
			d.modal = false
		}
		ctx.EndModal()
	}
}

func kitFrame(st *uiState, d *kitDemo, in uiInput) {
	ctx := st.begin(uiPassUpdate, nil, in)
	d.build(ctx)
	ctx.end()
}

func kitClick(st *uiState, d *kitDemo, x, y float64) {
	kitFrame(st, d, uiInput{X: x, Y: y, Moved: true})
	kitFrame(st, d, uiInput{X: x, Y: y, Pressed: true, Down: true})
	kitFrame(st, d, uiInput{X: x, Y: y, Released: true})
}

func TestUIKitMouseInteraction(t *testing.T) {
	st := &uiState{}
	d := &kitDemo{clicks: map[string]int{}}
	kitFrame(st, d, uiInput{})
	kitClick(st, d, 90, 120)
	if d.clicks["b1"] != 1 {
		t.Fatalf("b1 clicks=%d", d.clicks["b1"])
	}
	// Press on b1, release on b2: neither fires.
	kitFrame(st, d, uiInput{X: 90, Y: 120, Moved: true})
	kitFrame(st, d, uiInput{X: 90, Y: 120, Pressed: true, Down: true})
	kitFrame(st, d, uiInput{X: 260, Y: 120, Moved: true, Down: true})
	kitFrame(st, d, uiInput{X: 260, Y: 120, Released: true})
	if d.clicks["b1"] != 1 || d.clicks["b2"] != 0 {
		t.Fatalf("drag-off fired: %v", d.clicks)
	}
	d.disabled = true
	kitClick(st, d, 90, 180)
	if d.clicks["b3"] != 0 {
		t.Fatal("disabled button fired")
	}
	kitClick(st, d, 290, 288) // segment Z
	if d.seg != 2 {
		t.Fatalf("segmented=%d", d.seg)
	}
	kitClick(st, d, 205, 338) // stepper +
	kitClick(st, d, 205, 338)
	kitClick(st, d, 30, 338) // stepper -
	if d.step != 1 {
		t.Fatalf("stepper=%d", d.step)
	}
	kitClick(st, d, 40, 388)
	if !d.toggle {
		t.Fatal("toggle")
	}
	kitClick(st, d, 20+10+280*0.75, 438)
	if d.slider < 0.74 || d.slider > 0.76 {
		t.Fatalf("slider=%.3f", d.slider)
	}
	kitClick(st, d, 450, 270+4+32*2+10) // third visible row
	if d.listSel != 2 {
		t.Fatalf("list=%d", d.listSel)
	}
	kitFrame(st, d, uiInput{X: 450, Y: 300, Moved: true, Wheel: -1})
	if d.scroll != 1 {
		t.Fatalf("list scroll=%d", d.scroll)
	}
	kitClick(st, d, 440+440/3*2+20, 50) // tab C
	if d.tab != 2 {
		t.Fatalf("tab=%d", d.tab)
	}
}

func TestUIKitKeyboardFocusAndModal(t *testing.T) {
	st := &uiState{}
	d := &kitDemo{clicks: map[string]int{}}
	kitFrame(st, d, uiInput{})
	kitFrame(st, d, uiInput{DownKey: true}) // first focusable
	if st.focus == "" {
		t.Fatal("arrow did not focus anything")
	}
	st.focus = "b1"
	kitFrame(st, d, uiInput{Right: true})
	if st.focus != "b2" {
		t.Fatalf("right from b1 focused %q", st.focus)
	}
	kitFrame(st, d, uiInput{Enter: true})
	if d.clicks["b2"] != 1 {
		t.Fatal("enter did not activate the focused button")
	}
	st.focus = "seg"
	kitFrame(st, d, uiInput{Right: true})
	if d.seg != 1 || st.focus != "seg" {
		t.Fatalf("right on segmented: seg=%d focus=%q", d.seg, st.focus)
	}
	st.focus = "list"
	kitFrame(st, d, uiInput{DownKey: true})
	if d.listSel != 1 || st.focus != "list" {
		t.Fatalf("down on list: sel=%d focus=%q", d.listSel, st.focus)
	}
	kitFrame(st, d, uiInput{Tab: true})
	if st.focus == "list" {
		t.Fatal("tab did not leave the list")
	}
	kitFrame(st, d, uiInput{NextTab: true})
	if d.tab != 1 {
		t.Fatalf("E: tab=%d", d.tab)
	}

	d.modal = true
	kitFrame(st, d, uiInput{})
	kitClick(st, d, 90, 120) // b1 is below the modal scrim
	if d.clicks["b1"] != 0 {
		t.Fatal("modal let a click through")
	}
	kitFrame(st, d, uiInput{DownKey: true})
	if st.focus != "m.ok" {
		t.Fatalf("focus escaped the modal: %q", st.focus)
	}
	kitFrame(st, d, uiInput{Enter: true})
	if d.modal {
		t.Fatal("modal OK did not close")
	}
	esc := st.begin(uiPassUpdate, nil, uiInput{Esc: true})
	if !esc.Back() {
		t.Fatal("Esc not reported as Back")
	}
	esc.end()
}

func TestUIKitRendersAllComponents(t *testing.T) {
	st := &uiState{}
	d := &kitDemo{clicks: map[string]int{}, modal: true}
	st.pushToast("SAVED", uiToastOK)
	st.pushToast("CONNECTION LOST", uiToastError)
	kitFrame(st, d, uiInput{X: 90, Y: 120, Moved: true})
	st.keyboard = true
	screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
	ctx := st.begin(uiPassDraw, screen, uiInput{X: 90, Y: 120})
	d.build(ctx)
	ctx.end()
	for i := 0; i < 100; i++ {
		kitFrame(st, d, uiInput{})
	}
	if len(st.toasts) != 0 {
		t.Fatalf("toasts did not expire: %d", len(st.toasts))
	}
}

func TestUIKitTypeScaleMinimum(t *testing.T) {
	for _, s := range []float64{uiSizeTitle, uiSizeSection, uiSizeButton, uiSizeBody, uiSizeHint} {
		if s < 13 {
			t.Fatalf("type size %.0f below 13 px", s)
		}
	}
	if uiMeasure(true, uiSizeButton, "OPTIONS") <= 0 || uiMeasure(false, uiSizeHint, "hint") <= 0 {
		t.Fatal("embedded UI font did not load")
	}
}

// B4: the Tw Cen MT Condensed Extra Bold face is the embedded Barlow
// Condensed now, so it can never silently disappear without Office fonts.
func TestCondensedExtraBoldUsesEmbeddedFont(t *testing.T) {
	face, err := sourceMenuFace(menuFontCondensedExtraBold, 20)
	if err != nil || face == nil {
		t.Fatalf("extra bold face: %v", err)
	}
	if !menuFontCaches[menuFontCondensedExtraBold].substitute {
		t.Fatal("extra bold is not the embedded substitute")
	}
	if measureMenuText(menuFontCondensedExtraBold, 20, "START LOCAL GAME") <= 0 {
		t.Fatal("extra bold measures nothing")
	}
}
