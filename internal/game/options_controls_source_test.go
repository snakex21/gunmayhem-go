package game

import (
	"fmt"
	"testing"
	"github.com/hajimehoshi/ebiten/v2"
)

// Every player's seven key buttons (incl. ABILITY) are registered in the kit
// at their drawn rect, and clicking one starts capturing that binding.
func TestOptionsControlKeyButtonsHitTheirBinding(t *testing.T) {
	g := New()
	g.optionsTab = optionsTabControls
	g.runOptionsUpdate(uiInput{})
	for p := 0; p < 4; p++ {
		for i := range optionControlLabels {
			r := optionsControlKeyRect(p, i)
			it, found := g.optionsUI.item(fmt.Sprintf("opt.key.%d.%d", p, i))
			if !found || it.rect != r {
				t.Fatalf("key p%d i%d registered=%v rect=%+v want %+v", p, i, found, it.rect, r)
			}
			g.optionsKeyCapturePlayer, g.optionsKeyCaptureIndex = 0, 0
			x, y := r.X+r.W/2, r.Y+r.H/2
			g.runOptionsUpdate(uiInput{X: x, Y: y, Moved: true})
			g.runOptionsUpdate(uiInput{X: x, Y: y, Pressed: true, Down: true})
			g.runOptionsUpdate(uiInput{X: x, Y: y, Released: true})
			if g.optionsKeyCapturePlayer != p+1 || g.optionsKeyCaptureIndex != i {
				t.Fatalf("click on key p%d i%d captured player=%d index=%d", p, i, g.optionsKeyCapturePlayer, g.optionsKeyCaptureIndex)
			}
			g.optionsKeyCapturePlayer, g.optionsKeyCaptureIndex = 0, 0
			g.runOptionsUpdate(uiInput{X: x, Y: y})
		}
	}
}

func TestOptionsValidVKMatchesSourceSubset(t *testing.T) {
	for _, vk := range []int{38,37,40,39,219,221,87,65,83,68,84,89,96,111,123,186} {
		if !sourceOptionValidVK(vk) { t.Fatalf("source valid VK %d rejected", vk) }
	}
	for _, vk := range []int{27,112,114,116,117,122} {
		if sourceOptionValidVK(vk) { t.Fatalf("source invalid VK %d accepted", vk) }
	}
}

func TestOptionsEbitenToSourceVKDefaults(t *testing.T) {
	tests := map[ebiten.Key]int{
		ebiten.KeyArrowUp:38, ebiten.KeyArrowLeft:37, ebiten.KeyArrowDown:40, ebiten.KeyArrowRight:39,
		ebiten.KeyBracketLeft:219, ebiten.KeyBracketRight:221,
		ebiten.KeyW:87, ebiten.KeyA:65, ebiten.KeyS:83, ebiten.KeyD:68, ebiten.KeyT:84, ebiten.KeyY:89,
	}
	for key, want := range tests {
		got, ok := ebitenKeyToSourceVK(key)
		if !ok || got != want { t.Fatalf("key %v -> %d ok=%v want%d", key, got, ok, want) }
	}
}

func TestRemappedControlsAreUsedByCustomPlayers(t *testing.T) {
	arena := OriginalMap1()
	g := &Game{maps:map[int]Map{1:arena}, arena:arena, customMap:1, customMode:SourceGameModeNormal, customLives:10, seenDeaths:map[int]int{}}
	g.initCustomPlayerSetup()
	for i:=0;i<4;i++ { g.controlConfigs[i]=OriginalControls(i+1) }
	g.controlConfigs[0].Shoot = 90 // Z
	g.startCustomGame()
	if len(g.players)<1 || g.players[0].Controls.Shoot != 90 { t.Fatalf("custom player did not inherit remapped control: %+v", g.players[0].Controls) }
}
