package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type fadePurpose int

const (
	fadePurposeScreen fadePurpose = iota
	fadePurposeEndGame
)

const (
	pauseHitNone = iota
	pauseHitResume
	pauseHitExit
)

// Symbol630 button hit shape, decoded from the XFL hexadecimal edge values:
// #BAE.FC / 20 = 149.5498px, #29D.F9 / 20 = 33.4980px.
var pauseButtonLocal = Rect{X: 0, Y: 0, W: 149.5498, H: 33.4980}

func (g *Game) beginFade(target gameScreen, purpose fadePurpose) {
	if g.fadeActive {
		return
	}
	g.fadeActive = true
	g.fadeFrame = 1
	g.fadeTarget = target
	g.fadePurpose = purpose
	g.menuPressed = menuHitNone
	g.pausePressed = pauseHitNone
}

// updateFade reproduces Symbol625_fadeaway: frame10 changes the root frame and
// frame21 removes the clip. Screen changes therefore happen on source frame10,
// not at the beginning/end of the fade.
func (g *Game) updateFade() {
	if !g.fadeActive {
		return
	}
	if g.fadeFrame < 21 {
		g.fadeFrame++
	}
	if g.fadeFrame == 10 {
		// Load secondary menu/campaign/postgame assets while the source fade is
		// fully covering the screen, not before the Main Menu first appears.
		if g.assets != nil {
			g.assets.EnsureInteractions()
			if g.fadeTarget == screenGameplay {
				g.assets.EnsureGameplay()
			}
		}
		if g.fadePurpose == fadePurposeEndGame {
			g.finishSourceGameBeforeScreenChange()
		}
		g.screen = g.fadeTarget
		if g.screen == screenCampaign {
			g.enterCampaignMenu()
		}
		if g.audioStarted {
			g.syncSourceMusic()
		}
	}
	if g.fadeFrame >= 21 {
		g.fadeActive = false
		g.fadeFrame = 0
	}
}

func (g *Game) finishSourceGameBeforeScreenChange() {
	// frame10.returntomenu(): a campaign win is determined by whether any AI
	// player survived. Only a human victory completes the level and unlocks its
	// two source gunarray entries.
	if g.campaignMode {
		playerWin := true
		for _, p := range g.players {
			if p.Active && p.AI {
				playerWin = false
				break
			}
		}
		if playerWin && g.campaignLevel >= 1 && g.campaignLevel <= 10 {
			g.completeCampaignLevel(g.campaignLevel)
			g.campaignShowUnlock = g.campaignLevel
		}
	}
	g.paused = false
	g.GameWin = false
	g.teamGameWin = false
	g.matchWinCountdown = 0
	g.soloWinFrame = 0
	g.winnerPlayerID = 0
	g.winnerAnimFrame = 0
	g.teamWinAnimFrame = 0
	g.campaignLoseFrame = 0
	g.zombieWaveFrame = 0
	g.gototest = false
	g.testGunDisabled = false
	g.testGunRespawn = 0
	g.testGunFrame = 0
}

func (g *Game) completeCampaignLevel(level int) {
	if level < 1 || level > 10 {
		return
	}
	g.campaignLevels[level-1] = 2
	// Source campaign menu unlock propagation happens when frame4 is entered.
	// Level2 starts available; completing level2 unlocks level3, and so on.
	if level >= 2 && level < 10 && g.campaignLevels[level] == 0 {
		g.campaignLevels[level] = 1
	}
	rewards := [10][2]int{
		{18, 29}, {40, 52}, {19, 30}, {41, 53}, {20, 31},
		{42, 54}, {21, 32}, {43, 55}, {22, 33}, {44, 56},
	}
	for _, idx := range rewards[level-1] {
		if idx >= 0 && idx < len(g.campaignGuns) {
			g.campaignGuns[idx] = true
		}
	}
	_ = g.saveProgress()
}

func (g *Game) openPause() {
	if g.paused || g.fadeActive || g.GameWin || g.teamGameWin {
		return
	}
	g.paused = true
	g.pausePressed = pauseHitNone
}

func pauseHitAt(x, y float64) int {
	// Symbol633: btn1 Resume Game, btn2 exit game.
	if sourceMenuHitRect(pauseButtonLocal, 373.1, 253.1, 1, 1.35821533203125).Contains(x, y) {
		return pauseHitResume
	}
	if sourceMenuHitRect(pauseButtonLocal, 373.1, 313.6, 1, 0.6119384765625).Contains(x, y) {
		return pauseHitExit
	}
	return pauseHitNone
}

func (g *Game) updatePause() {
	mx, my := ebiten.CursorPosition()
	hit := pauseHitAt(float64(mx), float64(my))
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.pausePressed = hit
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		pressed := g.pausePressed
		g.pausePressed = pauseHitNone
		if pressed != hit || pressed == pauseHitNone {
			return
		}
		switch pressed {
		case pauseHitResume:
			g.paused = false
		case pauseHitExit:
			g.paused = false
			target := screenPostGame
			if g.gototest {
				target = screenGunLibrary
			} else if g.campaignMode {
				target = screenCampaign
			}
			g.beginFade(target, fadePurposeEndGame)
		}
	}
}

// updateGameplayInteractionInput must run before simulation. GAMEPAUSED halts
// root/player/effect onEnterFrame handlers in the source, so a pause frame must
// return without advancing gameplay.
func (g *Game) updateGameplayInteractionInput() bool {
	if g.fadeActive {
		g.updateFade()
		if g.screen != screenGameplay {
			return true
		}
	}
	if g.paused {
		g.updatePause()
		return true
	}
	if !g.fadeActive && !g.GameWin && !g.teamGameWin &&
		(inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeySpace)) {
		g.openPause()
		return true
	}
	return false
}

func activeSourcePlayers(players []*Player) []*Player {
	out := make([]*Player, 0, len(players))
	for _, p := range players {
		if p != nil && p.Active && !p.IsDouble {
			out = append(out, p)
		}
	}
	return out
}

func (g *Game) updateMatchInteractions() {
	// Existing attached movie clips advance before this tick can attach a new
	// one. A freshly attached source effect therefore remains on frame1 for its
	// first rendered frame, just like attachMovie in the Flash source.
	if g.GameWin && g.winnerAnimFrame > 0 && g.winnerAnimFrame < 41 {
		g.winnerAnimFrame++
	}
	if g.teamWinAnimFrame > 0 && g.teamWinAnimFrame < 20 {
		g.teamWinAnimFrame++
	}
	if g.campaignLoseFrame > 0 && g.campaignLoseFrame < 20 {
		g.campaignLoseFrame++
	}
	if g.zombieWaveFrame > 0 && g.zombieWaveFrame < 20 {
		g.zombieWaveFrame++
	}

	active := activeSourcePlayers(g.players)

	// Root frame10 teamgame test: if all active players have the same team,
	// teamgamewin is set immediately and fx_teamwin is attached.
	if g.TeamGame && !g.teamGameWin && len(active) > 0 {
		team := active[0].Team
		same := true
		for _, p := range active[1:] {
			if p.Team != team {
				same = false
				break
			}
		}
		if same {
			g.teamGameWin = true
			g.teamWinner = team
			g.teamWinAnimFrame = 1
			g.matchWinCountdown = 0
		}
	}

	// Player/AI source clips handle the last survivor differently in campaign.
	if !g.TeamGame && !g.teamGameWin && !g.GameWin && len(active) == 1 && g.GameMode != SourceGameModeSurvival {
		winner := active[0]
		if g.campaignMode && winner.AI {
			// playerAI: campaign loss sets teamgamewin and attaches fx_campaignlose.
			g.teamGameWin = true
			g.campaignLoseFrame = 1
			g.matchWinCountdown = 0
		} else if g.campaignMode {
			// Countdown Symbol1487 frame2 jumps directly to frame51 in campaign.
			g.setSourceGameWinner(winner)
		} else {
			// Non-campaign countdown plays frames2..51 before gamewin=true.
			if g.soloWinFrame == 0 {
				g.soloWinFrame = 2
			} else if g.soloWinFrame < 51 {
				g.soloWinFrame++
			}
			if g.soloWinFrame >= 51 {
				g.setSourceGameWinner(winner)
			}
		}
	}

	// root frame10: after either gamewin or teamgamewin, wait exactly 100
	// source ticks then call endgame(), which creates fadeaway.
	if g.GameWin || g.teamGameWin {
		g.matchWinCountdown++
		if g.matchWinCountdown >= 100 && !g.fadeActive {
			target := screenPostGame
			if g.gototest {
				target = screenGunLibrary
			} else if g.campaignMode {
				target = screenCampaign
			}
			g.beginFade(target, fadePurposeEndGame)
		}
	}
}

func (g *Game) setSourceGameWinner(p *Player) {
	if p == nil || g.GameWin {
		return
	}
	g.GameWin = true
	g.winnerPlayerID = p.ID
	g.winnerAnimFrame = 1
	g.matchWinCountdown = 0
	g.soloWinFrame = 51
	// Source immediately zeroes winner velocity while gamewin is true.
	p.VX, p.VY = 0, 0
}

func (g *Game) sourceWinnerPlayer() *Player {
	for _, p := range g.players {
		if p.ID == g.winnerPlayerID {
			return p
		}
	}
	return nil
}

func (g *Game) drawGameplayInteractions(screen *ebiten.Image) {
	if g.soloWinFrame >= 2 && g.soloWinFrame <= 51 && !g.campaignMode && !g.GameWin {
		g.drawSourceWinCountdown(screen, g.soloWinFrame)
	}
	if g.GameWin && g.winnerAnimFrame > 0 {
		if p := g.sourceWinnerPlayer(); p != nil {
			scale := playerRenderScale(p)
			drawSourceRaster(screen, g.assets.WinnerFrames[g.winnerAnimFrame],
				g.worldX(p.X), g.worldY(p.Y-150*scale), scale*float64(p.Facing), scale, 1)
		}
	}
	if g.teamWinAnimFrame > 0 {
		// Source sets clip._x=-root._x/_y=-root._y; the root transform then
		// cancels that translation, so the effect is screen-fixed at (0,0).
		drawSourceRaster(screen, g.assets.TeamWinFrames[g.teamWinAnimFrame], 0, 0, 1, 1, 1)
	}
	if g.campaignLoseFrame > 0 {
		drawSourceRaster(screen, g.assets.CampaignLoseFrames[g.campaignLoseFrame], 0, 0, 1, 1, 1)
	}
	if g.zombieWaveFrame > 0 {
		drawSourceRaster(screen, g.assets.ZombieWaveFrames[g.zombieWaveFrame], 0, 0, 1, 1, 1)
	}
	if g.paused {
		// game_pause copies hud._x/_y, cancelling the root camera and staying
		// screen-fixed. Our screen is already in final coordinates.
		drawSourceRaster(screen, g.assets.PauseMenu, 0, 0, 1, 1, 1)
	}
}

var sourceFadeX = [...]float64{
	-1050, -840.1, -654.95, -494.45, -358.65, -247.5, -161.1,
	-99.4, -62.35, -50, -50, -50, -50, -33.6, 15.65, 97.65,
	212.5, 360.15, 540.65, 753.9, 1000,
}

func (g *Game) drawGlobalFade(screen *ebiten.Image) {
	if !g.fadeActive || g.fadeFrame < 1 || g.fadeFrame > len(sourceFadeX) {
		return
	}
	// Symbol623 is [-79.9..968.85] x [-33..653.05]. Symbol624 scales/translates
	// that to exactly about [-100..1100] x [-100..800], then Symbol625 places
	// it at (sourceFadeX,-50). Net source rectangle: x=tx-100,y=-150,w=1200,h=900.
	x := sourceFadeX[g.fadeFrame-1] - 100
	ebitenutil.DrawRect(screen, x, -150, 1200, 900, color.Black)
}
