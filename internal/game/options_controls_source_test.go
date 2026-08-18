package game

import (
	"testing"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestOptionsControlHitboxesMatchSymbol972(t *testing.T) {
	// P1 panel is Symbol972 at (50,83.85), key1 at local (55.15,101.7).
	p, i, ok := optionsControlHitAt(50+55.15+10, 83.85+101.7+10)
	if !ok || p != 0 || i != 0 { t.Fatalf("P1 up hit p=%d i=%d ok=%v", p, i, ok) }
	// P4 bomb: panel x700, key6 local (78.85,247.2).
	p, i, ok = optionsControlHitAt(700+78.85+10, 83.85+247.2+10)
	if !ok || p != 3 || i != 5 { t.Fatalf("P4 bomb hit p=%d i=%d ok=%v", p, i, ok) }
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
