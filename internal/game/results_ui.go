package game

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Match results (Post Game Summary) and the series summary, rebuilt with the
// Mayhem UI kit (phase 7). The values and "best" highlights are the source
// ones (postGameSourceValues, postGameWinnerSlots); CONTINUE / READY run the
// existing flow (continueFromPostGame, markLocalPostGameReady) and the online
// ready countdown stays in updateOnlineHostPostGame.

var postGameRowLabels = [...]string{"KILLS", "DEATHS", "K/D RATIO", "SHOTS FIRED", "SHOTS HIT", "ACCURACY", "CRATES", "POWERUPS", "TOTAL POINTS"}

// postGameBestRow maps postGameWinnerSlots criteria to rows (values 1..9).
var postGameBestRow = [4]int{1, 3, 6, 9}

const (
	resultsLabelW   = 150.0
	resultsCardTop  = uiContentTop
	resultsRowTop   = 196.0
	resultsRowH     = 34.0
	resultsCardStep = 178.0
)

var resultsStatusRect = Rect{X: 240, Y: uiActionTop, W: 400, H: uiButtonH}

func resultsCardRect(slot int) Rect {
	return Rect{X: uiMargin + resultsLabelW + 10 + float64(slot)*resultsCardStep, Y: resultsCardTop, W: resultsCardStep - 8, H: uiContentBottom - resultsCardTop}
}

func (g *Game) resultsIsGuest() bool { return g.netplay != nil && g.netplay.mode == netplayClient }

func (g *Game) resultsOnline() bool { return g.onlineMatchSetup && g.netplay != nil }

// continueFromPostGame is the source Continue button (and ESC) of a local
// match: next series round, the series summary or back to the setup.
func (g *Game) continueFromPostGame() {
	// DefineSprite_1019.btn_back.
	g.playSourceSFX("menu.wav", false)
	if g.customSeriesRound > 0 {
		if !g.customSeriesIsFinalRound() {
			g.startNextCustomSeriesRound()
			return
		}
		g.finishCustomSeriesToSummary()
		return
	}
	g.customPage = customPageGame
	g.customMenuX = 1800
	g.beginFade(screenCustomGame, fadePurposeScreen)
}

func (g *Game) updatePostGameMenu() {
	if g.onlineMatchSetup && g.netplay != nil && g.netplay.mode == netplayHost {
		g.updateOnlineHostPostGame()
		return
	}
	g.runResultsUpdate(g.readUIInput(&g.resultsUI))
}

// runResultsUpdate is one update pass of the post-game or series screen for
// a frame of input (tests drive it).
func (g *Game) runResultsUpdate(in uiInput) {
	// A fresh results screen starts with a fresh kit state (focus, frame
	// count for the default-focus grace period).
	if g.resultsUIScreen != g.screen {
		g.resultsUI.reset()
		g.resultsUIScreen = g.screen
	}
	ctx := g.resultsUI.begin(uiPassUpdate, nil, in)
	g.buildResults(ctx)
	ctx.end()
}

func (g *Game) drawResultsMenu(screen *ebiten.Image) {
	ctx := g.resultsUI.begin(uiPassDraw, screen, uiInput{X: g.uiLastCursorX, Y: g.uiLastCursorY})
	g.buildResults(ctx)
	ctx.end()
}

func (g *Game) buildResults(ctx *uiCtx) {
	ctx.Page(0.9)
	if g.screen == screenSeriesSummary {
		g.buildSeriesSummary(ctx)
		return
	}
	g.buildPostGame(ctx)
}

// ---------------------------------------------------------------------------
// Post Game Summary

func (g *Game) postGameTitle() (string, string) {
	mode := developedModeTitle(g.GameMode)
	trail := []string{"VERSUS"}
	if g.onlineMatchSetup {
		trail[0] = "ONLINE"
	}
	trail = append(trail, mode, strings.ToUpper(onlineMapNumberLabel(g.arena.Number)))
	title := "MATCH RESULTS"
	if g.customSeriesRound > 0 && clampCustomSeriesRounds(g.customSeriesRounds) > 1 {
		title = fmt.Sprintf("ROUND %d / %d RESULTS", g.customSeriesRound, clampCustomSeriesRounds(g.customSeriesRounds))
		if leader := g.customSeriesLeaderText(); leader != "" {
			trail = append(trail, leader)
		}
	}
	return title, strings.Join(trail, "  /  ")
}

func (g *Game) buildPostGame(ctx *uiCtx) {
	title, trail := g.postGameTitle()
	ctx.Header(title, trail)
	g.drawPostGameTable(ctx)

	online := g.resultsOnline()
	guest := g.resultsIsGuest()
	label, info := "CONTINUE", "Back to the Custom Game setup."
	if g.customSeriesRound > 0 {
		info = "See the match summary."
		if g.customSeriesIsFinalRound() && clampCustomSeriesRounds(g.customSeriesRounds) > 1 {
			label, info = "SERIES SUMMARY", "See the series ranking."
		} else if !g.customSeriesIsFinalRound() {
			label, info = "NEXT ROUND", fmt.Sprintf("Start round %d.", g.customSeriesRound+1)
		}
	}
	ready := false
	if online {
		ready = g.localPostGameReady()
		label, info = "READY", "The next step starts when every player is ready."
		if ready {
			label = "READY!"
		}
	}
	if ctx.Button("res.continue", uiActionPrimaryRect(240), label, uiButtonOpts{Kind: uiButtonPrimary, Disabled: ready, InfoTitle: label, InfoBody: info}) {
		if online {
			g.playSourceSFX("menu.wav", false)
			g.markLocalPostGameReady()
		} else {
			g.continueFromPostGame()
		}
		return
	}
	if guest {
		if ctx.Button("res.leave", uiActionBackRect(), "LEAVE SESSION", uiButtonOpts{Kind: uiButtonDanger,
			InfoTitle: "LEAVE SESSION", InfoBody: "Disconnect from the host (ESC)."}) {
			g.guestLeaveSession()
			return
		}
	}
	if ctx.drawing() && online {
		if status := g.onlinePostGameStatusText(); status != "" {
			uiFillRound(ctx.dst, resultsStatusRect, 5, uiInk700)
			uiTextIn(ctx.dst, status, true, uiSizeBody, uiTextLight, resultsStatusRect, uiAlignCenter, 8)
		}
	}
	hints := []uiKeyHint{{"ENTER", label}}
	switch {
	case guest:
		hints = append(hints, uiKeyHint{"ESC", "Leave session"})
	case !online:
		hints = append(hints, uiKeyHint{"ESC", "Continue"})
	}
	ctx.KeyHintBar(hints)
	g.resultsDefaultFocus(ctx, "res.continue")
	// ESC: local = Continue (source); a guest's ESC leaves in
	// updateOnlineClientPostGame; the host has no ESC action.
	if ctx.Back() && !online {
		g.continueFromPostGame()
	}
}

func (g *Game) drawPostGameTable(ctx *uiCtx) {
	if !ctx.drawing() {
		return
	}
	labels := Rect{X: uiMargin, Y: resultsRowTop - 6, W: resultsLabelW, H: float64(len(postGameRowLabels))*resultsRowH + 8}
	uiFillRound(ctx.dst, labels, 6, uiInk700)
	for i, label := range postGameRowLabels {
		mid := resultsRowTop + float64(i)*resultsRowH + resultsRowH/2 - 2
		clr := uiTextMuted
		if i == len(postGameRowLabels)-1 {
			clr = uiOrange
		}
		uiText(ctx.dst, label, true, uiSizeBody, clr, uiMargin+14, mid, uiAlignLeft)
	}
	best := g.postGameWinnerSlots()
	for slot := 0; slot < 4; slot++ {
		p := g.playerBySlotID(slot + 1)
		r := resultsCardRect(slot)
		if p == nil {
			// Symbol1016 frame 2 is empty for a missing player.
			uiStrokeRound(ctx.dst, r, 8, 2, uiWithAlpha(uiInk500, 0.6))
			uiText(ctx.dst, fmt.Sprintf("P%d", slot+1), true, uiSizeSection, uiWithAlpha(uiTextMuted, 0.5), r.X+r.W/2, r.Y+40, uiAlignCenter)
			continue
		}
		values := postGameSourceValues(p)
		ctx.PlayerCard(r, slot, p.PlayerColor, false, "")
		name := strings.TrimSpace(values[0])
		if name == "" {
			name = fmt.Sprintf("Player %d", slot+1)
		}
		uiText(ctx.dst, uiFit(strings.ToUpper(name), true, uiSizeButton, r.W-60), true, uiSizeButton, uiTextDark, r.X+50, r.Y+27, uiAlignLeft)
		stage := Rect{X: r.X + 10, Y: r.Y + 44, W: r.W - 20, H: 52}
		uiFillRound(ctx.dst, stage, 5, uiPaperShade)
		cfg := CustomPlayerConfig{Color: p.PlayerColor, Shirt: p.ShirtNumber, Hat: p.HatNumber, Eyes: p.EyeNumber, Gun: p.DefaultWeapon, Ability: p.AbilityNumber}
		g.drawLoadoutCharacterIn(ctx.dst, cfg, uiInset(stage, 2))
		for i := 1; i < len(values); i++ {
			cell := Rect{X: r.X + 8, Y: resultsRowTop + float64(i-1)*resultsRowH, W: r.W - 16, H: resultsRowH - 4}
			isBest := false
			for c, row := range postGameBestRow {
				if row == i && best[c] == slot {
					isBest = true
				}
			}
			clr := uiTextDark
			if isBest {
				uiFillRound(ctx.dst, cell, 4, uiOrange)
				uiText(ctx.dst, "BEST", true, uiSizeHint, uiTextDark, cell.X+8, cell.Y+cell.H/2, uiAlignLeft)
			} else if i%2 == 1 {
				uiFillRound(ctx.dst, cell, 4, uiWithAlpha(uiPaperShade, 0.6))
			}
			size := uiSizeButton
			if i == len(values)-1 {
				size = uiSizeSection
			}
			uiText(ctx.dst, values[i], true, size, clr, cell.X+cell.W-10, cell.Y+cell.H/2, uiAlignRight)
		}
	}
}

// ---------------------------------------------------------------------------
// Series summary

type seriesRow struct {
	slot int
	stat customSeriesPlayerStats
}

// seriesRanking sorts by map wins, then kills, then points.
func (g *Game) seriesRanking() []seriesRow {
	rows := make([]seriesRow, 0, 4)
	for i, s := range g.customSeriesStats {
		if s.RoundsPlayed <= 0 {
			continue
		}
		rows = append(rows, seriesRow{slot: i, stat: s})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].stat.MapWins != rows[j].stat.MapWins {
			return rows[i].stat.MapWins > rows[j].stat.MapWins
		}
		if rows[i].stat.Kills != rows[j].stat.Kills {
			return rows[i].stat.Kills > rows[j].stat.Kills
		}
		return rows[i].stat.Score > rows[j].stat.Score
	})
	return rows
}

func seriesRowName(r seriesRow) string {
	if r.stat.Name == "" {
		return fmt.Sprintf("Player %d", r.slot+1)
	}
	return r.stat.Name
}

// backFromSeriesSummary is the old series BACK TO CUSTOM GAME (and ESC).
func (g *Game) backFromSeriesSummary() {
	g.playSourceSFX("menu.wav", false)
	g.customPage = customPageGame
	g.customMenuX = 1800
	g.customSeriesRound = 0
	g.customSeriesRecordedRound = 0
	g.beginFade(screenCustomGame, fadePurposeScreen)
}

func (g *Game) updateSeriesSummaryMenu() {
	if g == nil {
		return
	}
	if g.resultsIsGuest() && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.guestLeaveSession()
		return
	}
	g.runResultsUpdate(g.readUIInput(&g.resultsUI))
}

var seriesColumns = []struct {
	label string
	x     float64
}{{"MAP WINS", 420}, {"KILLS", 500}, {"DEATHS", 570}, {"K/D", 640}, {"ACC", 710}, {"POINTS", 860}}

func (g *Game) buildSeriesSummary(ctx *uiCtx) {
	rounds := clampCustomSeriesRounds(g.customSeriesRounds)
	ctx.Header("SERIES SUMMARY", fmt.Sprintf("%d MAPS PLAYED  /  RANKING: MAP WINS > KILLS > POINTS", rounds))
	rows := g.seriesRanking()
	if ctx.drawing() {
		win := Rect{X: uiMargin, Y: uiContentTop, W: ScreenWidth - 2*uiMargin, H: 92}
		ctx.Card(win, "")
		uiFillRound(ctx.dst, Rect{X: win.X, Y: win.Y, W: 10, H: win.H}, 4, uiOrange)
		uiText(ctx.dst, "SERIES WINNER", true, uiSizeHint, uiTextMutedDk, win.X+28, win.Y+26, uiAlignLeft)
		winner := "NO ROUNDS PLAYED"
		if len(rows) > 0 {
			winner = strings.ToUpper(seriesRowName(rows[0]))
			s := rows[0].stat
			uiText(ctx.dst, fmt.Sprintf("%d MAP WINS  /  %d KILLS  /  %d POINTS", s.MapWins, s.Kills, s.Score), true, uiSizeBody, uiTextMutedDk, win.X+win.W-20, win.Y+win.H/2, uiAlignRight)
		}
		uiText(ctx.dst, uiFit(winner, true, uiSizeTitle, 480), true, uiSizeTitle, uiTextDark, win.X+28, win.Y+58, uiAlignLeft)

		head := Rect{X: uiMargin, Y: 200, W: ScreenWidth - 2*uiMargin, H: 30}
		uiFillRound(ctx.dst, head, 4, uiInk700)
		uiText(ctx.dst, "PLAYER", true, uiSizeHint, uiTextMuted, head.X+60, head.Y+15, uiAlignLeft)
		for _, c := range seriesColumns {
			uiText(ctx.dst, c.label, true, uiSizeHint, uiTextMuted, c.x, head.Y+15, uiAlignRight)
		}
		for i, row := range rows {
			r := Rect{X: uiMargin, Y: 238 + float64(i)*62, W: ScreenWidth - 2*uiMargin, H: 54}
			fill, text := uiPaper, uiTextDark
			if i == 0 {
				fill = uiOrange
			}
			uiFillRound(ctx.dst, r, 6, fill)
			badge := Rect{X: r.X + 12, Y: r.Y + 13, W: 32, H: 28}
			uiFillRound(ctx.dst, badge, 4, uiInk900)
			uiTextIn(ctx.dst, fmt.Sprint(i+1), true, uiSizeButton, uiTextLight, badge, uiAlignCenter, 2)
			uiText(ctx.dst, uiFit(strings.ToUpper(seriesRowName(row)), true, uiSizeSection, 280), true, uiSizeSection, text, r.X+60, r.Y+r.H/2, uiAlignLeft)
			s := row.stat
			values := []string{fmt.Sprint(s.MapWins), fmt.Sprint(s.Kills), fmt.Sprint(s.Deaths), seriesKD(s), fmt.Sprintf("%d%%", seriesAccuracy(s)), fmt.Sprint(s.Score)}
			for j, c := range seriesColumns {
				size := uiSizeButton
				if j == 0 {
					size = uiSizeSection
				}
				uiText(ctx.dst, values[j], true, size, text, c.x, r.Y+r.H/2, uiAlignRight)
			}
		}
	}

	if g.resultsIsGuest() {
		ctx.Button("res.wait", uiActionPrimaryRect(240), "WAITING FOR HOST", uiButtonOpts{Kind: uiButtonPrimary, Disabled: true})
		if ctx.Button("res.leave", uiActionBackRect(), "LEAVE SESSION", uiButtonOpts{Kind: uiButtonDanger,
			InfoTitle: "LEAVE SESSION", InfoBody: "Disconnect from the host (ESC)."}) {
			g.guestLeaveSession()
		}
		ctx.KeyHintBar([]uiKeyHint{{"ESC", "Leave session"}})
		return
	}
	label := "BACK TO VERSUS"
	if g.onlineMatchSetup {
		label = "BACK TO MATCH SETUP"
	}
	if ctx.Button("res.done", uiActionPrimaryRect(240), label, uiButtonOpts{Kind: uiButtonPrimary, InfoTitle: label}) {
		g.backFromSeriesSummary()
		return
	}
	ctx.KeyHintBar([]uiKeyHint{{"ENTER", label}, {"ESC", "Back"}})
	g.resultsDefaultFocus(ctx, "res.done")
	if ctx.Back() {
		g.backFromSeriesSummary()
	}
}

// resultsFocusGrace is how long (update frames) a results screen ignores
// ENTER / SPACE before its main button takes the keyboard focus, so a key
// still pressed from the match cannot skip the results.
const resultsFocusGrace = 20

func (g *Game) resultsDefaultFocus(ctx *uiCtx, id string) {
	if ctx.pass == uiPassUpdate && ctx.st.focus == "" && ctx.st.frame >= resultsFocusGrace {
		ctx.st.focus = id
	}
}
