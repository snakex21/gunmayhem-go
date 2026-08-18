package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Symbol1019 places p1..p4 at these exact local X coordinates.
var postGamePanelX = [...]float64{219, 388, 557, 726}

func (g *Game) playerBySlotID(id int) *Player {
	for _, p := range g.players {
		if p != nil && !p.IsDouble && p.ID == id {
			return p
		}
	}
	return nil
}

// postGameSourceValues mirrors DefineSprite_1016/frame_1/DoAction.as.
func postGameSourceValues(p *Player) [10]string {
	if p == nil {
		return [10]string{}
	}
	kills, deaths := p.Kills, p.Deaths
	kd := 0.0
	if deaths > 0 {
		kd = math.Round((float64(kills)/float64(deaths))*100) / 100
	} else if kills > 0 {
		// Source: Infinity > 100000, so text3 becomes text1 (kills).
		kd = float64(kills)
	}
	accuracy := 0
	if p.ShotsFired > 0 {
		accuracy = int(math.Round(float64(p.HitsLanded) / float64(p.ShotsFired) * 100))
	}
	adjustedScore := float64(p.Score) * float64(accuracy) / 100
	return [10]string{
		p.Name,
		fmt.Sprintf("%d", kills),
		fmt.Sprintf("%d", deaths),
		fmt.Sprintf("%g", kd),
		fmt.Sprintf("%d", p.ShotsFired),
		fmt.Sprintf("%d", p.HitsLanded),
		fmt.Sprintf("%d%%", accuracy),
		fmt.Sprintf("%d", p.CratesCollected),
		fmt.Sprintf("%d", p.PowerupsCollected),
		fmt.Sprintf("%g", adjustedScore),
	}
}

// Symbol1019 has four orange winner outlines. Its source compares integer
// parseInt() values while p1..p4 construct, and hides the outline on a tie.
// Return -1 when no unique winner exists for a criterion.
func (g *Game) postGameWinnerSlots() [4]int {
	winner := [4]int{-1, -1, -1, -1}
	maxv := [4]int{}
	for slot := 0; slot < 4; slot++ {
		p := g.playerBySlotID(slot + 1)
		if p == nil {
			continue
		}
		kd := 0.0
		if p.Deaths > 0 {
			kd = math.Round((float64(p.Kills)/float64(p.Deaths))*100) / 100
		} else if p.Kills > 0 {
			kd = float64(p.Kills)
		}
		accuracy := 0
		if p.ShotsFired > 0 {
			accuracy = int(math.Round(float64(p.HitsLanded) / float64(p.ShotsFired) * 100))
		}
		adjustedScore := float64(p.Score) * float64(accuracy) / 100
		criteria := [4]int{p.Kills, int(kd), accuracy, int(adjustedScore)}
		for c, v := range criteria {
			if v > maxv[c] {
				maxv[c] = v
				winner[c] = slot
			} else if v == maxv[c] {
				// Exact source behavior: equality moves the corresponding winN to
				// x=-200, so no border is shown until a later strictly higher value.
				winner[c] = -1
			}
		}
	}
	return winner
}

// drawPostGameScreen reconstructs Symbol1019 without its flattened FFDec PNG.
// The PNG contains four baked Symbol1016 placeholder columns (Cool Dude / 89),
// which show through the source panel's 20%-alpha lower area.
func (g *Game) drawPostGameScreen(screen *ebiten.Image) bool {
	if g.assets.PostGameBackground == nil || g.assets.PostGamePanelBase == nil {
		return false
	}

	// Symbol992 is the clean 900x600 background/table striping.
	drawSourceRaster(screen, g.assets.PostGameBackground, 0, 0, 1, 1, 1)

	// Static text from Symbol1019. X positions include each DOMStaticText's
	// leftMargin where present, matching Flash's actual character start.
	drawSourceMenuText(screen, "Post Game Summary", menuFontCondensed, 53, color.Black, 18, 14)
	labels := [...]struct {
		text string
		x, y float64
	}{
		{"Kills", 123, 168.0},
		{"Deaths", 100, 204.05},
		{"K/D Ratio", 79, 240.1},
		{"Shots Fired", 64, 276.15},
		{"Shots Hit", 84, 312.2},
		{"Accuracy", 85, 348.25},
		{"Crates Collected", 19, 384.35},
		{"Powerups", 77, 420.4},
		{"TOTAL POINTS", 40, 456.45},
	}
	for _, row := range labels {
		drawSourceMenuText(screen, row.text, menuFontCondensed, 26.5, color.Black, row.x, row.y)
	}

	// btn_back = Symbol792 at the exact Symbol1019 transform.
	if g.assets.PostGameButton != nil {
		drawSourceRaster(screen, g.assets.PostGameButton, 638.15, 532.1,
			1.86492919921875, 1.0372772216796875, 1)
	}
	drawSourceMenuText(screen, "Continue", menuFontCondensed, 26.5, color.Black, 796.25, 545.3)

	g.drawPostGameStats(screen)
	return true
}

func (g *Game) drawPostGameStats(screen *ebiten.Image) {
	if g.assets.PostGamePanelBase == nil {
		return
	}
	valueY := [...]float64{3.7, 59.75, 96.25, 132.25, 167.8, 204.3, 240.3, 275.35, 311.85, 348.85}
	for slot, x := range postGamePanelX {
		p := g.playerBySlotID(slot + 1)
		if p == nil {
			// Symbol1016 source: pgsdata[number][1] == -1 -> gotoAndStop(2),
			// whose frame is completely empty. Do not draw a vacant gray column.
			continue
		}

		// Clean shape-only Symbol1016 frame1; no baked Cool Dude / 89 values.
		drawSourceRaster(screen, g.assets.PostGamePanelBase, x, 106.05, 1, 1, 1)
		values := postGameSourceValues(p)
		for i, value := range values {
			if i == 0 {
				// text0: Arial 18.
				drawSourceMenuText(screen, value, menuFontArial, 18, color.Black,
					x+5.15, 106.05+valueY[i])
				continue
			}
			// text1..text9: Century Gothic Bold 28.
			drawSourceMenuText(screen, value, menuFontCenturyGothicBold, 28, color.Black,
				x+6.65, 106.05+valueY[i])
		}
	}

	// Source win1..win4 orange outlines: Kills, integer-parsed K/D, Accuracy,
	// and Total Points. Ties deliberately show no outline.
	if g.assets.PostGameWinnerBorder != nil {
		winners := g.postGameWinnerSlots()
		y := [...]float64{166.05, 238.1, 346.2, 454.95}
		for criterion, slot := range winners {
			if slot < 0 || slot >= len(postGamePanelX) {
				continue
			}
			drawSourceRaster(screen, g.assets.PostGameWinnerBorder,
				postGamePanelX[slot], y[criterion], 1, 1, 1)
		}
	}
}
