package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	winCountdownRootX = 450.0
	winCountdownRootY = 280.0
	winCountdownTextX = -23.75
	winCountdownTextY = -59.4
)

// drawSourceWinCountdown reconstructs Symbol1487 directly from XFL. FFDec
// exported all 51 frames as one frozen raster containing only "3", while the
// Flash source animates three independent static-text children: 3, then 2,
// then 1. Their per-frame matrices and alpha values live in
// Assets.WinCountdownTimelines.
func (g *Game) drawSourceWinCountdown(screen *ebiten.Image, sourceFrame int) {
	if screen == nil || sourceFrame < 1 {
		return
	}
	frame := sourceFrame - 1 // Go stores Flash frame numbers; XFL is zero-based.
	for _, digit := range []int{3, 2, 1} {
		timeline := g.assets.WinCountdownTimelines[digit]
		if frame < 0 || frame >= len(timeline) {
			continue
		}
		state := timeline[frame]
		if !state.Valid || state.Alpha <= 0.001 {
			continue
		}
		alpha := uint8(math.Round(math.Max(0, math.Min(1, state.Alpha)) * 255))
		x := winCountdownRootX + state.Matrix.TX + winCountdownTextX
		y := winCountdownRootY + state.Matrix.TY + winCountdownTextY

		// Symbol1482/1484/1486 use Showcard Gothic 272.05, #ff3300 and a
		// black GlowFilter blur5/strength3. Reproduce the compact halo without
		// relying on the broken flattened countdown frames.
		shadow := color.NRGBA{A: alpha}
		for oy := -3; oy <= 3; oy++ {
			for ox := -3; ox <= 3; ox++ {
				if ox == 0 && oy == 0 {
					continue
				}
				drawSourceMenuText(screen, string(rune('0'+digit)), menuFontShowcardGothic, 272.05,
					shadow, x+float64(ox), y+float64(oy))
			}
		}
		drawSourceMenuText(screen, string(rune('0'+digit)), menuFontShowcardGothic, 272.05,
			color.NRGBA{R: 0xff, G: 0x33, A: alpha}, x, y)
		return // source frames contain at most one countdown digit at a time
	}
}
