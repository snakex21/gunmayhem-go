package game

import (
	"fmt"
	"math"
)

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
