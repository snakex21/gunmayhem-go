package game

import (
	"math"
	"testing"
)

// scriptedCampaignAssault plays the human side of a mission headlessly: every
// few frames it lands a real hit on each enemy through the normal combat code
// (a bullet into bots and drones from the arena centre, a grenade next to a
// shield-bearer since bullets bounce off his shield).
func scriptedCampaignAssault(g *Game, frame int) {
	if frame%12 != 0 {
		return
	}
	centerX := (g.arena.CrateMinX + g.arena.CrateMaxX) / 2
	shoot := func(p *Player) {
		dir := 1.0
		if p.X < centerX {
			dir = -1
		}
		y := p.Y - 30*playerRenderScale(p)/0.8
		g.bullets = append(g.bullets, Bullet{Kind: BulletNormal, X: p.X - dir*30, Y: y, VX: 25 * dir,
			Firepower: 45, OwnerID: 1, Team: 1, Facing: int(dir), Time: 1, Alpha: 1, TrailWidth: 20, TrailFrame: 1})
	}
	for _, p := range g.players {
		if p.Active && p.Team != 1 {
			shoot(p)
		}
	}
	for _, e := range g.campaignEnemies {
		if !e.alive() {
			continue
		}
		if campaignEnemyKind(e.Kind) == campaignEnemyShieldBearer {
			dir := 1.0
			if e.Body.X < centerX {
				dir = -1
			}
			profile := throwableProfileFor(ThrowableDynamite)
			g.grenades = append(g.grenades, Grenade{X: e.Body.X - dir*45, Y: e.Body.Y - 25, Fuse: profile.Fuse, OwnerID: 1, Type: ThrowableDynamite})
			continue
		}
		shoot(&e.Body)
	}
}

func runCampaignMissionSim(t *testing.T, level int) {
	t.Helper()
	g := startExtraMissionForTest(t, level)
	g.assets = LoadAssets()
	g.screen = screenGameplay
	g.fadeActive = false
	var human *Player
	for _, p := range g.players {
		if p.ID == 1 {
			human = p
		}
	}
	human.Lives, human.TotalLives = 9999, 9999
	frames := 0
	for frame := 1; frame <= 60*240; frame++ {
		frames = frame
		scriptedCampaignAssault(g, frame)
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		if g.GameWin || g.teamGameWin {
			break
		}
	}
	if !g.GameWin || g.winnerPlayerID != 1 {
		t.Fatalf("mission %d not won in the scripted sim: win=%v teamwin=%v remaining=%d", level, g.GameWin, g.teamGameWin, g.campaignEnemiesRemaining())
	}
	if g.campaignEnemiesRemaining() != 0 {
		t.Fatalf("mission %d won with %d enemies left", level, g.campaignEnemiesRemaining())
	}
	want := extraCampaignUnitCount(extraCampaignMissions[level-1])
	if human.Kills < want || math.IsNaN(human.X) {
		t.Fatalf("mission %d: kills credited=%d want >= %d units", level, human.Kills, want)
	}
	t.Logf("mission %d won after %d frames, kills=%d, human deaths=%d", level, frames, human.Kills, human.Deaths)
}

func TestNewEnemyMissionsAreWinnableHeadless(t *testing.T) {
	if testing.Short() {
		t.Skip("long simulation")
	}
	for level := 21; level <= extraCampaignMissionCount; level++ {
		runCampaignMissionSim(t, level)
	}
}
