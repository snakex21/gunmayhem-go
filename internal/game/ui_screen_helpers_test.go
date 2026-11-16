package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// kitScreen drives one Mayhem UI screen in tests: run is the screen's
// run*Update entry point, st its uiState.
type kitScreen struct {
	t   *testing.T
	st  *uiState
	run func(uiInput)
}

func (k kitScreen) idle() { k.run(uiInput{X: -10, Y: -10}) }

func (k kitScreen) item(id string) uiItem {
	k.t.Helper()
	it, ok := k.st.item(id)
	if !ok {
		var ids []string
		for _, it := range k.st.items {
			ids = append(ids, it.id)
		}
		k.t.Fatalf("kit item %q not on screen; have %v", id, ids)
	}
	return it
}

func (k kitScreen) has(id string) bool {
	_, ok := k.st.item(id)
	return ok
}

// click presses and releases the mouse over id, then runs one idle frame so
// the registry reflects what the click opened.
func (k kitScreen) click(id string) {
	k.t.Helper()
	r := k.item(id).rect
	x, y := r.X+r.W/2, r.Y+r.H/2
	k.run(uiInput{X: x, Y: y, Moved: true})
	k.run(uiInput{X: x, Y: y, Pressed: true, Down: true})
	k.run(uiInput{X: x, Y: y, Released: true})
	k.run(uiInput{X: x, Y: y})
}

func (k kitScreen) key(in uiInput) {
	in.X, in.Y = -10, -10
	k.run(in)
	k.run(uiInput{X: -10, Y: -10})
}

// uniqueIDs fails on a duplicate control id in the current registry.
func (k kitScreen) uniqueIDs(name string) {
	k.t.Helper()
	assertKitIDsUnique(k.t, k.st, name)
}

func assertKitIDsUnique(t *testing.T, st *uiState, name string) {
	t.Helper()
	seen := map[string]bool{}
	for _, it := range st.items {
		if it.id == "" {
			t.Fatalf("%s: control without id at %+v", name, it.rect)
		}
		if seen[it.id] {
			t.Fatalf("%s: duplicate kit id %q", name, it.id)
		}
		seen[it.id] = true
	}
}

// keyboardReaches checks that Tab cycles through every focusable control.
func (k kitScreen) keyboardReaches(ids ...string) {
	k.t.Helper()
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		k.run(uiInput{X: -10, Y: -10, Tab: true})
		seen[k.st.focus] = true
	}
	for _, id := range ids {
		if !seen[id] {
			k.t.Fatalf("Tab never focuses %q (saw %v)", id, seen)
		}
	}
}

func newTestImage() *ebiten.Image { return ebiten.NewImage(ScreenWidth, ScreenHeight) }
