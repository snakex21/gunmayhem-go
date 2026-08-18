//go:build !windows

package game

import "github.com/hajimehoshi/ebiten/v2"

func sourceKeyDown(code int) bool {
	key, ok := sourceKeyToEbiten(code)
	return ok && ebiten.IsKeyPressed(key)
}

func sourceKeyToEbiten(code int) (ebiten.Key, bool) {
	switch code {
	case 38:
		return ebiten.KeyArrowUp, true
	case 37:
		return ebiten.KeyArrowLeft, true
	case 40:
		return ebiten.KeyArrowDown, true
	case 39:
		return ebiten.KeyArrowRight, true
	case 219:
		return ebiten.KeyBracketLeft, true
	case 221:
		return ebiten.KeyBracketRight, true
	case 87:
		return ebiten.KeyW, true
	case 65:
		return ebiten.KeyA, true
	case 83:
		return ebiten.KeyS, true
	case 68:
		return ebiten.KeyD, true
	case 84:
		return ebiten.KeyT, true
	case 89:
		return ebiten.KeyY, true
	case 111:
		return ebiten.KeyNumpadDivide, true
	case 103:
		return ebiten.KeyNumpad7, true
	case 104:
		return ebiten.KeyNumpad8, true
	case 105:
		return ebiten.KeyNumpad9, true
	case 106:
		return ebiten.KeyNumpadMultiply, true
	case 109:
		return ebiten.KeyNumpadSubtract, true
	case 101:
		return ebiten.KeyNumpad5, true
	case 97:
		return ebiten.KeyNumpad1, true
	case 98:
		return ebiten.KeyNumpad2, true
	case 99:
		return ebiten.KeyNumpad3, true
	case 96:
		return ebiten.KeyNumpad0, true
	case 110:
		return ebiten.KeyNumpadDecimal, true
	default:
		return 0, false
	}
}
