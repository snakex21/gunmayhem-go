package game

import (
	"fmt"
	"testing"
	"github.com/hajimehoshi/ebiten/v2"
)

// Every player's seven key buttons (incl. ABILITY) are hit at their drawn rect
// and registered in the kit with the same geometry.
func TestOptionsControlKeyButtonsHitTheirBinding(t *testing.T) {
	g := New()
	g.optionsTab = optionsTabControls
	g.runOptionsUpdate(uiInput{})
	for p := 0; p < 4; p++ {
		for i := range optionControlLabels {
			r := optionsControlKeyRect(p, i)
			gp, gi, ok := optionsControlHitAt(r.X+r.W/2, r.Y+r.H/2)
			if !ok || gp != p || gi != i {
				t.Fatalf("key p%d i%d hit p=%d i=%d ok=%v", p, i, gp, gi, ok)
			}
			it, found := g.optionsUI.item(fmt.Sprintf("opt.key.%d.%d", p, i))
			if !found || it.rect != r {
				t.Fatalf("key p%d i%d registered=%v rect=%+v want %+v", p, i, found, it.rect, r)
			}
		}
	}
	if _, _, ok := optionsControlHitAt(5, 5); ok {
		t.Fatal("hit outside the key buttons")
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
