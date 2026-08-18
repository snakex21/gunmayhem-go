package game

import (
	"os/exec"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	creditsHitNone = iota
	creditsHitKevinGu
	creditsHitMusic
	creditsHitBack
)

// Source button text/hit extents from Symbol1040/1045 button states.
var (
	creditsKevinLocal = Rect{X: 2, Y: 2, W: 85.75, H: 40.8}
	creditsMusicLocal = Rect{X: 2, Y: 2, W: 130.3, H: 40.8}
)

func creditsHitAt(x, y float64) int {
	if sourceMenuHitRect(creditsKevinLocal, 298.2, 95.75, 1, 1).Contains(x, y) {
		return creditsHitKevinGu
	}
	if sourceMenuHitRect(creditsMusicLocal, 579.65, 282.3, 1, 1).Contains(x, y) {
		return creditsHitMusic
	}
	if sourceMenuHitRect(campaignBackLocal, 716.15, 520.15, 1, 1).Contains(x, y) {
		return creditsHitBack
	}
	return creditsHitNone
}

func (g *Game) updateCreditsMenu() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.menuPressed = menuHitNone
		g.beginFade(screenMainMenu, fadePurposeScreen)
		return
	}
	mx, my := ebiten.CursorPosition()
	hit := creditsHitAt(float64(mx), float64(my))

	// Source credit links are onPress, while Back is onRelease.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		switch hit {
		case creditsHitKevinGu:
			_ = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", "http://www.thekevingu.com").Start()
			return
		case creditsHitMusic:
			_ = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", "http://www.incompetech.com").Start()
			return
		case creditsHitBack:
			g.menuPressed = menuHitGameBack
		}
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		pressed := g.menuPressed
		g.menuPressed = menuHitNone
		if pressed == menuHitGameBack && hit == creditsHitBack {
			g.beginFade(screenMainMenu, fadePurposeScreen)
		}
	}
}
