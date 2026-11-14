package game

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// zombie_menu.go: ZOMBIE setup and results screens (Mayhem UI kit), the
// zombie settings of an online lobby, and the per-arena records.

var (
	zombieSetupModeRect   = Rect{X: 34, Y: 134, W: 392, H: 40}
	zombieSetupArenaY     = 228.0
	zombieSetupPlayersPan = Rect{X: 456, Y: uiContentTop, W: 424, H: 300}
	zombieSetupRulesPan   = Rect{X: 456, Y: 402, W: 424, H: 114}
	zombieSetupLeftPan    = Rect{X: uiMargin, Y: uiContentTop, W: 424, H: uiContentBottom - uiContentTop}
	zombieOnlineHostRect  = Rect{X: 232, Y: uiActionTop, W: 200, H: uiButtonH}
)

var zombieSlotLabels = []string{"OFF", "HUMAN", "BOT"}

func (g *Game) openZombieSetup() {
	g.zombieUI.reset()
	g.onlineZombie = false
	if g.zombieSetup.Slots == [4]int{} {
		g.zombieSetup = defaultZombieSetup()
	}
	g.zombieSetup.Slots[0] = 1
	g.beginFade(screenZombieSetup, fadePurposeScreen)
}

func (g *Game) updateZombieSetupMenu() {
	g.runZombieSetupUpdate(g.readUIInput(&g.zombieUI))
}

// runZombieSetupUpdate is one update pass (tests drive it).
func (g *Game) runZombieSetupUpdate(in uiInput) {
	ctx := g.zombieUI.begin(uiPassUpdate, nil, in)
	g.buildZombieSetup(ctx)
	ctx.end()
}

func (g *Game) drawZombieSetupMenu(screen *ebiten.Image) {
	ctx := g.zombieUI.begin(uiPassDraw, screen, uiInput{X: g.uiLastCursorX, Y: g.uiLastCursorY})
	g.buildZombieSetup(ctx)
	ctx.end()
}

func zombieModeInfo(mode int) string {
	if mode == zombieModeEndless {
		return "Waves never end. Your best wave and score on this arena are saved."
	}
	return fmt.Sprintf("Survive %d waves. A boss comes every 5th wave.", zombieSurvivalWaves)
}

func (g *Game) buildZombieSetup(ctx *uiCtx) {
	ctx.Page(0.9)
	ctx.Header("ZOMBIES", "MAIN MENU  /  ZOMBIES  /  CO-OP SURVIVAL")
	ctx.Badge(Rect{X: uiMargin + uiMeasure(true, uiSizeTitle, "ZOMBIES") + 14, Y: 31, W: 64, H: 22}, "CO-OP", zombieTagColor)
	cfg := &g.zombieSetup

	ctx.Panel(zombieSetupLeftPan, "MODE AND ARENA")
	if m := ctx.Segmented("zs.mode", zombieSetupModeRect, []string{"SURVIVAL (20 WAVES)", "ENDLESS"}, cfg.Mode, false,
		"MODE", "SURVIVAL: 20 waves and a victory. ENDLESS: as long as you last, with a saved record."); m != cfg.Mode {
		cfg.Mode = m
	}
	if ctx.drawing() {
		uiText(ctx.dst, zombieModeInfo(cfg.Mode), false, uiSizeHint, uiTextMuted, zombieSetupModeRect.X, zombieSetupModeRect.Y+56, uiAlignLeft)
		uiText(ctx.dst, "ARENA", true, uiSizeButton, uiTextLight, zombieSetupModeRect.X, zombieSetupArenaY-14, uiAlignLeft)
	}
	g.buildZombieArenaTiles(ctx, "zs.arena", zombieSetupModeRect.X, zombieSetupArenaY)
	if ctx.drawing() {
		a := zombieArenaByIndex(cfg.Arena)
		uiParagraph(ctx.dst, a.Blurb, false, uiSizeHint, uiTextMuted, zombieSetupModeRect.X, zombieSetupArenaY+170, 390, 18, 2)
		rec := g.zombieRecordFor(cfg.Arena, cfg.Mode)
		text := "NO RECORD YET"
		if rec.Games > 0 {
			text = fmt.Sprintf("RECORD: WAVE %d  -  %d PTS  -  %d GAMES", rec.BestWave, rec.BestScore, rec.Games)
			if cfg.Mode == zombieModeSurvival {
				text += fmt.Sprintf("  -  %d WINS", rec.Wins)
			}
		}
		uiText(ctx.dst, text, true, uiSizeBody, color.NRGBA{R: 0xff, G: 0xd8, B: 0x40, A: 0xff}, zombieSetupModeRect.X, zombieSetupArenaY+226, uiAlignLeft)
	}

	ctx.Panel(zombieSetupPlayersPan, "PLAYERS")
	for slot := 0; slot < 4; slot++ {
		row := Rect{X: zombieSetupPlayersPan.X + 12, Y: zombieSetupPlayersPan.Y + 48 + float64(slot)*60, W: zombieSetupPlayersPan.W - 24, H: 52}
		pc := g.customPlayers[slot]
		if ctx.drawing() {
			uiFillRound(ctx.dst, row, 5, uiWithAlpha(uiInk600, 0.55))
			uiFillRect(ctx.dst, Rect{X: row.X, Y: row.Y + 8, W: 5, H: row.H - 16}, developedPlayerColorSwatch(pc.Color))
			name := pc.Name
			if name == "" {
				name = "Player " + strconv.Itoa(slot+1)
			}
			uiText(ctx.dst, uiFit(name, true, uiSizeButton, 160), true, uiSizeButton, uiTextLight, row.X+14, row.Y+18, uiAlignLeft)
			uiText(ctx.dst, "P"+strconv.Itoa(slot+1)+" KEYS  -  "+uiFit(weaponDisplayName(pc.Gun), false, uiSizeHint, 120), false, uiSizeHint, uiTextMuted, row.X+14, row.Y+38, uiAlignLeft)
		}
		seg := Rect{X: row.X + row.W - 214, Y: row.Y + 9, W: 206, H: 34}
		if slot == 0 {
			ctx.Badge(Rect{X: seg.X + 70, Y: seg.Y + 6, W: 66, H: 22}, "HUMAN", uiInk500)
			continue
		}
		if v := ctx.Segmented("zs.slot."+strconv.Itoa(slot), seg, zombieSlotLabels, cfg.Slots[slot], false,
			"PLAYER "+strconv.Itoa(slot+1), "OFF, a human on this keyboard (own keys from Options) or an AI teammate."); v != cfg.Slots[slot] {
			cfg.Slots[slot] = v
		}
	}
	ctx.Panel(zombieSetupRulesPan, "")
	if ctx.drawing() {
		r := zombieSetupRulesPan
		uiParagraph(ctx.dst, "Points: 10 per hit, 60-160 per kill, bonus for headshots and ring-outs. Spend them in the SHOP between waves or press DOWN at the MYSTERY BOX (950). Downed? A teammate stands next to you for 3 s.",
			false, uiSizeHint, uiTextMuted, r.X+14, r.Y+18, r.W-28, 18, 5)
	}

	if ctx.Button("zs.back", uiActionBackRect(), "BACK", uiButtonOpts{InfoTitle: "BACK", InfoBody: "Return to the main menu."}) || ctx.Back() {
		g.mainMenuSelectSound()
		g.beginFade(screenMainMenu, fadePurposeScreen)
	}
	if ctx.Button("zs.online", zombieOnlineHostRect, "HOST ONLINE", uiButtonOpts{InfoTitle: "HOST ONLINE",
		InfoBody: "Open an online co-op session (LAN or Internet). Friends join from ONLINE; every player sees its own shop."}) {
		g.mainMenuSelectSound()
		g.openZombieOnlineHost()
	}
	if ctx.Button("zs.start", uiActionPrimaryRect(220), "START", uiButtonOpts{Kind: uiButtonPrimary, InfoTitle: "START",
		InfoBody: fmt.Sprintf("Start with %d player(s).", cfg.participants())}) {
		g.mainMenuSelectSound()
		g.startZombieMatch()
	}
	ctx.KeyHintBar([]uiKeyHint{{"UP/DOWN", "Move"}, {"LEFT/RIGHT", "Change"}, {"ENTER", "Select"}, {"ESC", "Back"}})
}

// buildZombieArenaTiles draws one tile per arena: a single-arena map
// thumbnail, or a mosaic of the zones of a multi-zone map with a ZONES badge.
func (g *Game) buildZombieArenaTiles(ctx *uiCtx, prefix string, x, y float64) {
	cfg := &g.zombieSetup
	const gap, total = 8.0, 392.0
	n := float64(len(zombieArenas))
	w := math.Floor((total - gap*(n-1)) / n)
	for i := range zombieArenas {
		a := &zombieArenas[i]
		r := Rect{X: x + float64(i)*(w+gap), Y: y, W: w, H: 150}
		info := a.Blurb
		if len(a.Zones) > 1 {
			info = fmt.Sprintf("%d ZONES. %s", len(a.Zones), a.Blurb)
		} else {
			info = "SINGLE ARENA. " + a.Blurb
		}
		if ctx.Tile(prefix+"."+strconv.Itoa(i), r, a.Name, uiTileOpts{Selected: cfg.Arena == i, InfoTitle: a.Name, InfoBody: info,
			Art: func(dst *ebiten.Image, inner Rect) {
				if g.assets == nil {
					return
				}
				if len(a.Zones) <= 1 {
					drawSetupThumbFit(dst, g.assets.MapPreviewImages[a.zone(a.StartZone).MapNumber], inner, 1)
					return
				}
				// 2x2 mosaic of the first zones.
				cw, ch := inner.W/2, inner.H/2
				for zi, z := range a.Zones {
					if zi >= 4 {
						break
					}
					cell := Rect{X: inner.X + float64(zi%2)*cw, Y: inner.Y + float64(zi/2)*ch, W: cw, H: ch}
					drawSetupThumbFit(dst, g.assets.MapPreviewImages[z.MapNumber], uiInset(cell, 1), 1)
				}
				badge := Rect{X: inner.X + 4, Y: inner.Y + 4, W: 64, H: 18}
				uiFillRound(dst, badge, 3, zombieTagColor)
				uiTextIn(dst, fmt.Sprintf("%d ZONES", len(a.Zones)), true, uiSizeHint, uiTextLight, badge, uiAlignCenter, 2)
			}}) {
			cfg.Arena = i
		}
	}
}

// openZombieOnlineHost opens the online host setup for a zombie session.
func (g *Game) openZombieOnlineHost() {
	g.customPage = customPageGame
	g.customMenuX = customPageTargetX(customPageGame)
	g.customLoadoutSlot = 0
	g.returnToOnlineServerBrowser()
	g.onlineZombie = true
	g.customHostSetupOpen = true
	g.multiplayerMessage = "CONFIGURE THE ZOMBIE CO-OP SESSION"
	g.beginFade(screenCustomGame, fadePurposeScreen)
}

// buildZombieOnlineSettings is the MATCH / MAPS tab of a zombie lobby
// (read-only for guests through ctx.readOnly).
func (g *Game) buildZombieOnlineSettings(ctx *uiCtx) {
	pan := Rect{X: uiMargin, Y: uiContentTop, W: ScreenWidth - 2*uiMargin, H: uiContentBottom - uiContentTop}
	ctx.Panel(pan, "ZOMBIE CO-OP")
	cfg := &g.zombieSetup
	if m := ctx.Segmented("zo.mode", Rect{X: pan.X + 14, Y: pan.Y + 52, W: 392, H: 40}, []string{"SURVIVAL (20 WAVES)", "ENDLESS"}, cfg.Mode, false,
		"MODE", "SURVIVAL: 20 waves. ENDLESS: as long as you last."); m != cfg.Mode {
		cfg.Mode = m
	}
	if ctx.drawing() {
		uiText(ctx.dst, zombieModeInfo(cfg.Mode), false, uiSizeHint, uiTextMuted, pan.X+14, pan.Y+108, uiAlignLeft)
		uiText(ctx.dst, "ARENA", true, uiSizeButton, uiTextLight, pan.X+14, pan.Y+140, uiAlignLeft)
		uiParagraph(ctx.dst, "Every player has an own HP bar, points and shop. Guests buy on their own screen; the host checks every purchase. Players can join while the match runs.",
			false, uiSizeHint, uiTextMuted, pan.X+440, pan.Y+60, 400, 18, 6)
	}
	g.buildZombieArenaTiles(ctx, "zo.arena", pan.X+14, pan.Y+156)
}

// --- results -------------------------------------------------------------------------

func (g *Game) updateZombieResultsMenu() {
	g.runZombieResultsUpdate(g.readUIInput(&g.zombieUI))
}

func (g *Game) runZombieResultsUpdate(in uiInput) {
	ctx := g.zombieUI.begin(uiPassUpdate, nil, in)
	g.buildZombieResults(ctx)
	ctx.end()
}

func (g *Game) drawZombieResultsMenu(screen *ebiten.Image) {
	ctx := g.zombieUI.begin(uiPassDraw, screen, uiInput{X: g.uiLastCursorX, Y: g.uiLastCursorY})
	g.buildZombieResults(ctx)
	ctx.end()
}

func (g *Game) buildZombieResults(ctx *uiCtx) {
	ctx.Page(0.9)
	m := g.zombie
	ctx.Header("ZOMBIES  -  RESULTS", "ZOMBIES  /  RESULTS")
	if m == nil {
		if ctx.Button("zr.menu", uiActionBackRect(), "MENU", uiButtonOpts{}) || ctx.Back() {
			g.beginFade(screenZombieSetup, fadePurposeScreen)
		}
		return
	}
	h := m.hud
	a := zombieArenaByIndex(h.Arena)
	title := fmt.Sprintf("GAME OVER  -  %d WAVES SURVIVED", m.wavesSurvived())
	titleCol := color.NRGBA{R: 0xff, G: 0x60, B: 0x50, A: 0xff}
	if h.Phase == zombiePhaseWon {
		title = fmt.Sprintf("YOU SURVIVED ALL %d WAVES!", h.Wave)
		titleCol = color.NRGBA{R: 0x80, G: 0xff, B: 0x70, A: 0xff}
	}
	card := Rect{X: uiMargin, Y: uiContentTop, W: ScreenWidth - 2*uiMargin, H: 96}
	ctx.Panel(card, "")
	if ctx.drawing() {
		uiText(ctx.dst, title, true, 32, titleCol, card.X+20, card.Y+32, uiAlignLeft)
		sub := fmt.Sprintf("%s  -  %s  -  %d ZOMBIES KILLED  -  TEAM SCORE %d", a.Name, zombieModeLabels[min(max(h.Mode, 0), zombieModeCount-1)], h.Kills, m.teamScore())
		uiText(ctx.dst, sub, false, uiSizeBody, uiTextMuted, card.X+20, card.Y+66, uiAlignLeft)
		if h.NewRecord {
			ctx.Badge(Rect{X: card.X + card.W - 140, Y: card.Y + 20, W: 120, H: 26}, "NEW RECORD!", uiOrange)
		} else if h.RecordWave > 0 {
			uiText(ctx.dst, fmt.Sprintf("RECORD: WAVE %d / %d PTS", h.RecordWave, h.RecordScore), true, uiSizeHint, uiTextMuted, card.X+card.W-20, card.Y+32, uiAlignRight)
		}
	}
	table := Rect{X: uiMargin, Y: card.Y + card.H + 12, W: card.W, H: uiContentBottom - card.Y - card.H - 12}
	ctx.Panel(table, "PLAYERS")
	cols := []struct {
		label string
		x     float64
	}{{"POINTS", 330}, {"KILLS", 440}, {"HEADSHOTS", 540}, {"DOWNS", 650}, {"REVIVES", 750}}
	if ctx.drawing() {
		for _, c := range cols {
			uiText(ctx.dst, c.label, true, uiSizeHint, uiTextMuted, table.X+c.x, table.Y+56, uiAlignCenter)
		}
		row := 0
		for i := range h.Players {
			zp := h.Players[i]
			if zp.ID == 0 {
				continue
			}
			y := table.Y + 84 + float64(row)*44
			row++
			name := "P" + strconv.Itoa(zp.ID)
			col := uiOrange
			if p := g.playerByID(zp.ID); p != nil {
				name = p.Name
				if p.AI {
					name += " (BOT)"
				}
				col = developedPlayerColorSwatch(p.PlayerColor)
			}
			uiFillRound(ctx.dst, Rect{X: table.X + 12, Y: y - 18, W: table.W - 24, H: 36}, 5, uiWithAlpha(uiInk600, 0.6))
			uiFillRect(ctx.dst, Rect{X: table.X + 12, Y: y - 12, W: 5, H: 24}, col)
			uiText(ctx.dst, uiFit(name, true, uiSizeButton, 240), true, uiSizeButton, uiTextLight, table.X+28, y, uiAlignLeft)
			vals := []int{zp.Earned, zp.Kills, zp.Headshots, zp.Downs, zp.Revives}
			for k, c := range cols {
				uiText(ctx.dst, strconv.Itoa(vals[k]), true, uiSizeButton, uiTextLight, table.X+c.x, y, uiAlignCenter)
			}
		}
	}

	online := g.netplay != nil
	host := online && g.netplay.mode == netplayHost
	switch {
	case online && !host:
		ctx.Button("zr.wait", uiActionPrimaryRect(240), "WAITING FOR HOST", uiButtonOpts{Kind: uiButtonPrimary, Disabled: true})
		if ctx.Button("zr.leave", uiActionBackRect(), "LEAVE", uiButtonOpts{Kind: uiButtonDanger,
			InfoTitle: "LEAVE", InfoBody: "Disconnect from the host and return to the server browser."}) || ctx.Back() {
			g.zombie = nil
			g.zombieUI.reset()
			_ = g.CloseNetplay()
			g.returnToOnlineServerBrowser()
			g.screen = screenCustomGame
			g.multiplayerMessage = "DISCONNECTED"
		}
	case host:
		if ctx.Button("zr.lobby", uiActionPrimaryRect(240), "BACK TO LOBBY", uiButtonOpts{Kind: uiButtonPrimary,
			InfoTitle: "BACK TO LOBBY", InfoBody: "Return everybody to the session lobby."}) {
			g.zombie = nil
			g.clearDisconnectedOnlineHumanSlots()
			g.openOnlineCustomSlots("ZOMBIE MATCH OVER - START AGAIN WHEN READY")
		}
	default:
		if ctx.Button("zr.again", uiActionPrimaryRect(220), "PLAY AGAIN", uiButtonOpts{Kind: uiButtonPrimary,
			InfoTitle: "PLAY AGAIN", InfoBody: "Same arena, mode and players."}) {
			g.zombie = nil
			g.startZombieMatch()
			return
		}
		if ctx.Button("zr.menu", uiActionBackRect(), "MENU", uiButtonOpts{InfoTitle: "MENU", InfoBody: "Back to the zombie setup."}) || ctx.Back() {
			g.zombie = nil
			g.zombieUI.reset()
			g.beginFade(screenZombieSetup, fadePurposeScreen)
		}
	}
	ctx.KeyHintBar([]uiKeyHint{{"LEFT/RIGHT", "Move"}, {"ENTER", "Select"}, {"ESC", "Menu"}})
}

// --- records -------------------------------------------------------------------------

func zombieRecordKey(arena, mode int) string {
	m := "survival"
	if mode == zombieModeEndless {
		m = "endless"
	}
	return zombieArenaByIndex(arena).ID + "/" + m
}

func (g *Game) zombieRecordFor(arena, mode int) ZombieRecord {
	if g.zombieRecords == nil {
		return ZombieRecord{}
	}
	return g.zombieRecords[zombieRecordKey(arena, mode)]
}

// saveZombieRecord stores the match result (best wave, best team score).
func (g *Game) saveZombieRecord() {
	m := g.zombie
	if m == nil || !m.host {
		return
	}
	if g.zombieRecords == nil {
		g.zombieRecords = make(map[string]ZombieRecord)
	}
	key := zombieRecordKey(m.hud.Arena, m.hud.Mode)
	rec := g.zombieRecords[key]
	rec.Games++
	if m.hud.Phase == zombiePhaseWon {
		rec.Wins++
	}
	wave, score := m.wavesSurvived(), m.teamScore()
	if wave > rec.BestWave || (wave == rec.BestWave && score > rec.BestScore) {
		m.hud.NewRecord = rec.Games > 1 || wave > 0
	}
	if wave > rec.BestWave {
		rec.BestWave = wave
	}
	if score > rec.BestScore {
		rec.BestScore = score
	}
	g.zombieRecords[key] = rec
	_ = g.saveProgress()
}

func (c zombieSetupConfig) String() string {
	parts := make([]string, 0, 4)
	for i, s := range c.Slots {
		parts = append(parts, fmt.Sprintf("P%d=%s", i+1, zombieSlotLabels[min(max(s, 0), 2)]))
	}
	return fmt.Sprintf("%s %s %s", zombieModeLabels[min(max(c.Mode, 0), zombieModeCount-1)], zombieArenaByIndex(c.Arena).Name, strings.Join(parts, " "))
}
