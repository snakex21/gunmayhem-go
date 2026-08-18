package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	ebitentext "github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
)

const (
	playerArrowCenterX = 450.0
	playerArrowCenterY = 300.0
	playerArrowRadius  = 262.0
	playerDistRadius   = 214.9
)

func (g *Game) sourcePlayerArrowState(p *Player) (visible bool, angle float64, distance int) {
	if p == nil || !p.Active {
		return false, 0, 0
	}
	sx := g.worldX(p.X)
	sy := g.worldY(p.Y)
	// Symbol649 source bounds. Keep the slightly asymmetric right edge (910)
	// exactly as the ActionScript rather than normalizing it to ScreenWidth.
	visible = sy <= 50 || sy >= 550 || sx <= 50 || sx >= 910
	if !visible {
		return false, 0, 0
	}
	dx := sx - playerArrowCenterX
	dy := sy - playerArrowCenterY
	angle = math.Atan2(dy, dx)
	distance = int(math.Floor(math.Hypot(dx, dy) + 0.5))
	return visible, angle, distance
}

func drawPlayerArrowRaster(screen *ebiten.Image, r *SourceRaster, angle float64) {
	if r == nil || r.Image == nil {
		return
	}
	// Symbol649.arrow is Symbol645 at local (0,-262). The parent rotation is
	// degrees+90, where degrees=atan2(target-center).
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(r.Bounds.X, r.Bounds.Y)
	op.GeoM.Translate(0, -playerArrowRadius)
	op.GeoM.Rotate(angle + math.Pi/2)
	op.GeoM.Translate(playerArrowCenterX, playerArrowCenterY)
	screen.DrawImage(r.Image, op)
}

func drawPlayerArrowDistance(screen *ebiten.Image, value int, angle float64) {
	face, err := sourceMenuFace(menuFontArial, 24)
	if err != nil || face == nil {
		return
	}
	text := fmt.Sprintf("%d", value)
	w := font.MeasureString(face, text).Ceil()
	// Symbol649.dist sits at local y=-214.9 and cancels the parent rotation,
	// so the number stays upright while orbiting around the center.
	x := playerArrowCenterX + math.Cos(angle)*playerDistRadius
	y := playerArrowCenterY + math.Sin(angle)*playerDistRadius
	left := int(math.Round(x)) - w/2
	baseline := int(math.Round(y-10.95)) + face.Metrics().Ascent.Ceil()
	// Source uses a black GlowFilter (blur 4, strength 3). A compact 8-neighbor
	// shadow reproduces the readable black halo without inventing a new asset.
	black := color.NRGBA{A: 0xff}
	for oy := -2; oy <= 2; oy++ {
		for ox := -2; ox <= 2; ox++ {
			if ox == 0 && oy == 0 {
				continue
			}
			ebitentext.Draw(screen, text, face, left+ox, baseline+oy, black)
		}
	}
	ebitentext.Draw(screen, text, face, left, baseline, color.White)
}

func (g *Game) drawPlayerArrows(screen *ebiten.Image) {
	for _, p := range g.players {
		visible, angle, distance := g.sourcePlayerArrowState(p)
		if !visible {
			continue
		}
		arrow := g.assets.PlayerArrowFrames[p.PlayerColor]
		if arrow == nil {
			// Source colours are 0..10. Keep a deterministic fallback for malformed
			// custom data rather than dropping the indicator entirely.
			arrow = g.assets.PlayerArrowFrames[0]
		}
		drawPlayerArrowRaster(screen, arrow, angle)
		drawPlayerArrowDistance(screen, distance, angle)
	}
}
