package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var campaignLevelTitles = [...]string{
	"",
	"Level 1: Tutorial",
	"Level 2: Easy Win",
	"Level 3: Gun Game",
	"Level 4: Seeing Double",
	"BOSS 1: Dynamite Dodge",
	"Level 6: Double Team",
	"Level 7: Unfair Advantage",
	"Level 8: One Hit One Kill",
	"Level 9: The Ghost",
	"BOSS 2: Minigun Maniac",
}

var campaignMapTitles = [...]string{
	"",
	"Map: Testing Room",
	"Map: Safari Showdown",
	"Map: Polar Pwnage",
	"Map: Hovering Houses",
	"Map: Great Wall Brawl",
	"Map: Grim City",
	"Map: Dessert Duel",
	"Map: Underwater Slaughter",
	"Map: No Name",
	"Map: Desert Destruction",
}

var campaignDescriptions = [...][]string{
	nil,
	{"Learn the basics of the game."},
	{"Face off against a single AI opponent."},
	{"Level up your gun by killing your opponent. First", "kill on level 15 wins."},
	{"Just a simple one on one. Or is it?"},
	{"Defeat your opponent, but mind the falling", "dynamite! P.S. Your opponent is resistant to", "dynamites :)"},
	{"Face off against 2 AI opponents."},
	{"Your opponent has a recharging jetpack."},
	{"Defeat your opponent using the most powerful", "gun in the game."},
	{"Put your perception skills to the test."},
	{"The final test. Only the best will win."},
}

func (g *Game) drawCampaignLevelRuntime(screen *ebiten.Image, level int, x, y, alpha float64) {
	if level < 1 || level > 10 {
		return
	}
	if thumb := g.assets.CampaignThumbs[level]; thumb != nil {
		drawSourceRaster(screen, thumb, x+5, y+5, 0.5, 0.5, alpha)
	}
	// Symbols52/56/... put a 50% black strip over the lower 35px of the
	// thumbnail, with Symbol90's level title on top.
	strip := color.NRGBA{A: uint8(128 * alpha)}
	ebitenutil.DrawRect(screen, x+5, y+144.8, 300, 35.2, strip)
	drawSourceMenuText(screen, campaignLevelTitles[level], menuFontCondensed, 24.75,
		color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: uint8(255 * alpha)}, x+11, y+146.8)

	state := g.campaignLevels[level-1]
	switch state {
	case 0:
		if g.assets.CampaignLockOverlay != nil {
			drawSourceRaster(screen, g.assets.CampaignLockOverlay, x+5, y+5, 1, 1, alpha)
		}
	case 2:
		if g.assets.CampaignDoneOverlay != nil {
			drawSourceRaster(screen, g.assets.CampaignDoneOverlay, x+5, y+5, 1, 1, alpha)
		}
		drawSourceMenuText(screen, "Completed", menuFontCondensed, 24.75,
			color.NRGBA{R: 0xff, G: 0xcc, A: uint8(255 * alpha)}, x+154.55, y+20)
	}
}

func (g *Game) drawCampaignInfoRuntime(screen *ebiten.Image, level int, alpha float64) {
	if level < 1 || level > 10 {
		return
	}
	// Symbol904.info is Symbol900 at (486.1,120). Cover the stale frame-1
	// FFDec raster and reconstruct the selected frame from the source bitmap and
	// static text layers.
	x, y := 486.1, 120.0
	ebitenutil.DrawRect(screen, x, y, 361, 300, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: uint8(255 * alpha)})
	if thumb := g.assets.CampaignThumbs[level]; thumb != nil {
		drawSourceRaster(screen, thumb, x+1, y, 0.6, 0.6, alpha)
	}
	drawSourceMenuText(screen, campaignLevelTitles[level], menuFontCondensed, 30.05,
		color.NRGBA{A: uint8(255 * alpha)}, x+2, y+219)
	drawSourceMenuText(screen, campaignMapTitles[level], menuFontCondensed, 30.05,
		color.NRGBA{A: uint8(255 * alpha)}, x+2, y+248.5)
	ebitenutil.DrawRect(screen, x, y+286.8, 359.6, 1.5,
		color.NRGBA{A: uint8(255 * alpha)})
	for i, line := range campaignDescriptions[level] {
		drawSourceMenuText(screen, line, menuFontCondensed, 21.2,
			color.NRGBA{A: uint8(255 * alpha)}, x+2, y+293+float64(i)*25)
	}
}
