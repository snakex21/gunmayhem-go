package game

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Campaign screens rebuilt with the Mayhem UI kit (phase 6): one mission list
// per set (GM1 / GM2 / EXTRA tabs), the mission briefing with the P1 / P2
// cards and local co-op choice, the campaign Loadout (the Custom Game loadout
// screen bound to campaignPlayers) and a results modal after a mission.
//
// Every action runs the existing campaign code: the set switch and mission
// open replicate the old click handlers, the cards write through
// activateCampaignPlayerHit codes and START calls startCampaignMission /
// startGM2CampaignMission / startExtraCampaignMission, so progress, unlocks
// and the mission rules are untouched.

const (
	campaignSetGM1 = iota
	campaignSetGM2
	campaignSetExtra
)

var campaignSetLabels = []string{"GM1", "GM2", "EXTRA"}

var campaignSetNames = [3]string{"GUN MAYHEM", "GUN MAYHEM 2", "EXTRA"}

var (
	campaignTabBarRect   = Rect{X: 470, Y: 30, W: 410, H: 40}
	campaignGridRect     = Rect{X: uiMargin, Y: uiContentTop, W: 590, H: uiContentBottom - uiContentTop}
	campaignInfoRect     = Rect{X: 622, Y: uiContentTop, W: 258, H: uiContentBottom - uiContentTop}
	campaignPageRect     = Rect{X: 300, Y: uiActionTop, W: 260, H: uiButtonH}
	campaignCoopRect     = Rect{X: 620, Y: 30, W: 260, H: 40}
	campaignBriefCard    = Rect{X: 444, Y: uiContentTop, W: 436, H: uiContentBottom - uiContentTop}
	campaignResultRect   = Rect{X: 190, Y: 110, W: 520, H: 380}
	campaignStartBtnRect = uiActionPrimaryRect(240)
)

func campaignPlayerCardRect(slot int) Rect {
	return Rect{X: uiMargin + float64(slot)*210, Y: uiContentTop, W: 200, H: uiContentBottom - uiContentTop}
}

// campaignResultInfo is the outcome of the last campaign match, shown as a
// modal over the mission list (UI only; progress is saved by the existing
// complete*Level functions).
type campaignResultInfo struct {
	open   bool
	won    bool
	set    int
	level  int
	unlock [2]int // GM1 reward guns (0: none)
	lines  []string
}

// ---------------------------------------------------------------------------
// Set / mission state (existing rules)

func (g *Game) campaignSet() int {
	switch {
	case g.campaignSetExtra:
		return campaignSetExtra
	case g.campaignSetGM2:
		return campaignSetGM2
	}
	return campaignSetGM1
}

func campaignMissionCount(set int) int {
	switch set {
	case campaignSetGM2:
		return len(gm2CampaignMissions)
	case campaignSetExtra:
		return extraCampaignMissionCount
	}
	return len(gm1CampaignSpecs)
}

// campaignMissionState is 0 locked, 1 available, 2 completed.
func (g *Game) campaignMissionState(set, level int) int {
	if level < 1 || level > campaignMissionCount(set) {
		return 0
	}
	switch set {
	case campaignSetGM2:
		return g.gm2CampaignLevels[level-1]
	case campaignSetExtra:
		return g.extraCampaignLevels[level-1]
	}
	return g.campaignLevels[level-1]
}

func (g *Game) campaignCompletedCount(set int) int {
	n := 0
	for level := 1; level <= campaignMissionCount(set); level++ {
		if g.campaignMissionState(set, level) == 2 {
			n++
		}
	}
	return n
}

// selectCampaignSet is the old GM1 / GM2 / EXTRA tab click.
func (g *Game) selectCampaignSet(set int) {
	if set == g.campaignSet() {
		return
	}
	g.campaignSetGM2 = set == campaignSetGM2
	g.campaignSetExtra = set == campaignSetExtra
	g.campaignLevel = 0
	g.campaignPhase = 1
	if set == campaignSetExtra {
		g.extraCampaignPage = 0
	}
	g.playSourceSFX("menu.wav", false)
}

// openCampaignMission is the old mission-card click of the current set.
func (g *Game) openCampaignMission(level int) {
	set := g.campaignSet()
	if g.campaignMissionState(set, level) == 0 {
		return
	}
	switch set {
	case campaignSetGM2:
		g.openGM2CampaignMissionSetup(level)
	case campaignSetExtra:
		g.campaignLevel = level
		g.campaignPhase = 2
		g.campaignNameFocus = 0
	default:
		g.campaignLevel = level
		g.campaignPhase = 2
	}
	g.playSourceSFX("menu.wav", false)
	if set == campaignSetExtra {
		g.extraCampaignPage = (level - 1) / extraCampaignPageSize
	}
}

// startSelectedCampaignMission is the old START button of the current set.
func (g *Game) startSelectedCampaignMission() {
	g.playSourceSFX("menu.wav", false)
	switch g.campaignSet() {
	case campaignSetGM2:
		g.startGM2CampaignMission()
	case campaignSetExtra:
		g.startExtraCampaignMission()
	default:
		g.startCampaignMission()
	}
}

func (g *Game) campaignBriefingOpen() bool {
	return g.campaignPhase == 2 || g.campaignPhase == 3
}

// closeCampaignBriefing is the old briefing BACK (phase 3 -> 4 -> 1).
func (g *Game) closeCampaignBriefing() {
	g.closeCampaignLoadout()
	g.campaignLoadoutReturn = 0
	g.campaignPhase = 1
	g.campaignNameFocus = 0
}

// campaignCoopAllowed: GM2 missions 1 and 16 are solo
// (openGM2CampaignMissionSetup / startGM2CampaignMission).
func (g *Game) campaignCoopAllowed() bool {
	return !(g.campaignSet() == campaignSetGM2 && (g.campaignLevel == 1 || g.campaignLevel == 16))
}

func (g *Game) campaignNextMission(set int) int {
	for level := 1; level <= campaignMissionCount(set); level++ {
		if g.campaignMissionState(set, level) == 1 {
			return level
		}
	}
	return 0
}

// ---------------------------------------------------------------------------
// Mission descriptions

type campaignMissionInfo struct {
	title     string
	mapName   string
	mapID     int
	mode      string
	lives     string
	pickups   string
	rules     string
	opponents []string
	briefing  string
	returns   string // EXTRA: enemy respawns and non-player units
	boss      bool
	solo      bool
	rewards   []int
}

func (g *Game) campaignMission(set, level int) campaignMissionInfo {
	if level < 1 || level > campaignMissionCount(set) {
		return campaignMissionInfo{}
	}
	switch set {
	case campaignSetGM2:
		spec := gm2CampaignMissions[level-1]
		info := campaignMissionInfo{
			title: strings.ToUpper(gm2CampaignTitles[level-1]), mapName: gm2MapDisplayNames[spec.MapNumber], mapID: gm2MapID(spec.MapNumber),
			mode: "LAST MAN STANDING", lives: fmt.Sprint(spec.Lives),
			pickups: "CRATES " + onOff(spec.Crates) + "  /  POWERUPS " + onOff(spec.Powerups),
			solo:    level == 1 || level == 16, boss: level == 16,
		}
		if spec.Lives > 9999 {
			info.lives = "UNLIMITED"
		}
		for _, e := range spec.Enemies {
			info.opponents = append(info.opponents, e.Name+" - "+weaponDisplayName(e.Gun))
		}
		if len(info.opponents) == 0 {
			info.opponents = []string{"None: training only"}
		}
		switch {
		case len(spec.Enemies) == 0:
			info.briefing = "Tutorial / training mission: learn to move, jump and shoot."
		case len(spec.Enemies) == 1:
			info.briefing = "Defeat your opponent."
		default:
			info.briefing = fmt.Sprintf("Defeat all %d opponents.", len(spec.Enemies))
		}
		return info
	case campaignSetExtra:
		spec := extraCampaignMissions[level-1]
		info := campaignMissionInfo{
			title: strings.ToUpper(extraCampaignTitles[level-1]), mapName: sourceMapDisplayNames[spec.MapNumber], mapID: spec.MapNumber,
			mode: spec.Mode, lives: fmt.Sprint(spec.PlayerLives),
			pickups:   "CRATES " + onOff(spec.Crates) + "  /  POWERUPS " + onOff(spec.Powerups),
			rules:     spec.RuleText,
			opponents: extraCampaignOpponentLines(spec),
			briefing:  spec.Briefing,
			returns:   fmt.Sprint(extraCampaignEnemyRespawns(spec)),
		}
		if units := extraCampaignUnitCount(spec); units > 0 {
			info.returns += fmt.Sprintf("  /  %d UNITS", units)
		}
		for _, e := range spec.Enemies {
			info.boss = info.boss || e.Boss != nil
		}
		for _, u := range spec.Units {
			info.boss = info.boss || u.Title != ""
		}
		return info
	}
	spec := gm1CampaignSpecs[level-1]
	title := campaignLevelTitles[level]
	if i := strings.Index(title, ": "); i >= 0 {
		title = title[i+2:]
	}
	info := campaignMissionInfo{
		title: strings.ToUpper(title), mapName: strings.TrimPrefix(campaignMapTitles[level], "Map: "), mapID: spec.mapNumber,
		mode: developedModeTitle(spec.mode), lives: fmt.Sprint(spec.lives),
		pickups:  "CRATES " + onOff(spec.crate) + "  /  POWERUPS " + onOff(spec.power),
		briefing: strings.Join(campaignDescriptions[level], " "),
		boss:     strings.HasPrefix(campaignLevelTitles[level], "BOSS"),
	}
	gun := spec.gun
	if level == 10 {
		gun = 55 // playerAIboss overrides p4gun with the minigun
	}
	opponent := spec.name
	if gun > 0 {
		opponent += " - " + weaponDisplayName(gun)
	}
	info.opponents = []string{opponent}
	if level == 6 {
		info.opponents = append(info.opponents, "Mafia Dude - "+weaponDisplayName(3))
	}
	for _, idx := range gm1CampaignRewards[level-1] {
		info.rewards = append(info.rewards, idx+10)
	}
	return info
}

func campaignStateLabel(state int) string {
	switch state {
	case 2:
		return "COMPLETED"
	case 1:
		return "AVAILABLE"
	}
	return "LOCKED"
}

func campaignStateColor(state int) (fill, text color.NRGBA) {
	switch state {
	case 2:
		return uiOK, uiTextLight
	case 1:
		return uiOrange, uiTextDark
	}
	return uiInk500, uiTextLight
}

// ---------------------------------------------------------------------------
// Update / draw entry points

func (g *Game) updateCampaignMenu() {
	if g.allowCheats && inpututil.IsKeyJustPressed(ebiten.KeyF10) {
		switch g.campaignSet() {
		case campaignSetGM2:
			if g.campaignPhase == 1 {
				g.unlockAllGM2CampaignLevelsCheat()
				g.playSourceSFX("menu.wav", false)
			}
		case campaignSetExtra:
			if g.campaignPhase == 1 {
				g.unlockAllExtraCampaignLevelsCheat()
				g.playSourceSFX("menu.wav", false)
			}
		default:
			g.unlockAllCampaignLevelsCheat()
			g.playSourceSFX("menu.wav", false)
		}
	}
	// A focused name field owns the keyboard for the whole frame, including
	// the Enter/Esc that ends the edit.
	textFocus := g.campaignNameFocus > 0
	if g.campaignPhase != 1 {
		g.updateCampaignNameInput()
	}
	g.preloadCampaignMapThumbs()
	in := g.readUIInput(&g.campaignUI)
	if textFocus {
		in = in.pointerOnly()
	}
	g.runCampaignUpdate(in)
}

// runCampaignUpdate is one update pass for a frame of input (tests drive it).
func (g *Game) runCampaignUpdate(in uiInput) {
	ctx := g.campaignUI.begin(uiPassUpdate, nil, in)
	g.buildCampaign(ctx)
	ctx.end()
}

func (g *Game) drawCampaignMenu(screen *ebiten.Image) {
	ctx := g.campaignUI.begin(uiPassDraw, screen, uiInput{X: g.uiLastCursorX, Y: g.uiLastCursorY})
	g.buildCampaign(ctx)
	ctx.end()
}

func (g *Game) buildCampaign(ctx *uiCtx) {
	// The kit screen has no Flash cross-fade: the briefing opens and closes
	// at once (old phases 2 -> 3 and 4 -> 1).
	if ctx.pass == uiPassUpdate {
		switch g.campaignPhase {
		case 2:
			g.campaignPhase = 3
		case 4:
			g.campaignPhase = 1
		}
	}
	ctx.Page(0.9)
	if g.campaignLoadoutSlot > 0 {
		if g.campaignBriefingOpen() && g.campaignPlayers[g.campaignLoadoutSlot-1].Type != 0 {
			g.buildCampaignLoadout(ctx)
			return
		}
		g.campaignLoadoutSlot = 0
	}
	if g.campaignBriefingOpen() {
		g.buildCampaignBriefing(ctx)
		return
	}
	g.buildCampaignList(ctx)
}

// ---------------------------------------------------------------------------
// Mission list

func campaignGridLayout(set int) (cols, rows int) {
	switch set {
	case campaignSetGM2:
		return 4, 4
	case campaignSetExtra:
		return 4, 3
	}
	return 5, 2
}

func campaignTileRect(set, i int) Rect {
	cols, rows := campaignGridLayout(set)
	gap := 8.0
	w := (campaignGridRect.W - float64(cols-1)*gap) / float64(cols)
	h := (campaignGridRect.H - float64(rows-1)*gap) / float64(rows)
	return Rect{X: campaignGridRect.X + float64(i%cols)*(w+gap), Y: campaignGridRect.Y + float64(i/cols)*(h+gap), W: w, H: h}
}

// campaignPageRange is the mission range (1-based levels) shown on the list.
func (g *Game) campaignPageRange(set int) (first, last int) {
	if set != campaignSetExtra {
		return 1, campaignMissionCount(set)
	}
	if g.extraCampaignPage < 0 || g.extraCampaignPage >= extraCampaignPageCount() {
		g.extraCampaignPage = 0
	}
	start, end := extraCampaignPageRange(g.extraCampaignPage)
	return start + 1, end
}

func campaignMissionID(level int) string { return fmt.Sprintf("camp.m.%d", level) }

func (g *Game) buildCampaignList(ctx *uiCtx) {
	set := g.campaignSet()
	done := g.campaignCompletedCount(set)
	ctx.Header("CAMPAIGN", fmt.Sprintf("MAIN MENU  /  CAMPAIGN  /  %s  -  %d / %d COMPLETED", campaignSetNames[set], done, campaignMissionCount(set)))
	if t := ctx.TabBar("camp.set", campaignTabBarRect, campaignSetLabels, set); t != set {
		g.selectCampaignSet(t)
		return
	}

	first, last := g.campaignPageRange(set)
	infoLevel := 0
	infoID := ctx.InfoID()
	for level := first; level <= last; level++ {
		level := level
		info := g.campaignMission(set, level)
		state := g.campaignMissionState(set, level)
		id := campaignMissionID(level)
		body := info.mapName + ". " + info.briefing
		if state == 0 {
			body = "Locked: complete the previous mission first."
		}
		if ctx.Tile(id, campaignTileRect(set, level-first), "", uiTileOpts{
			Selected: g.campaignLevel == level, Disabled: state == 0, Locked: state == 0,
			InfoTitle: fmt.Sprintf("MISSION %d: %s", level, info.title), InfoBody: body,
			Art: func(dst *ebiten.Image, inner Rect) { g.drawCampaignMissionTile(dst, inner, set, level, info, state) },
		}) {
			g.openCampaignMission(level)
			g.campaignUI.focus = ""
			return
		}
		if id == infoID {
			infoLevel = level
		}
	}
	if infoLevel == 0 {
		infoLevel = g.campaignLevel
		if infoLevel < first || infoLevel > last {
			infoLevel = g.campaignNextMission(set)
		}
		if infoLevel == 0 {
			infoLevel = first
		}
	}
	g.drawCampaignListInfo(ctx, set, infoLevel)

	if ctx.Button("camp.back", uiActionBackRect(), "MAIN MENU", uiButtonOpts{InfoTitle: "MAIN MENU"}) {
		g.campaignBackToMainMenu()
		return
	}
	if set == campaignSetExtra {
		pages := extraCampaignPageCount()
		page := g.extraCampaignPage
		if d := ctx.Stepper("camp.page", campaignPageRect, fmt.Sprintf("PAGE %d / %d", page+1, pages), page > 0, page+1 < pages, false,
			"PAGE", "LEFT / RIGHT or the arrows: more EXTRA missions."); d != 0 {
			g.extraCampaignPage = page + d
			g.playSourceSFX("menu.wav", false)
		}
	}
	next := g.campaignNextMission(set)
	nextLabel := "ALL COMPLETED"
	if next > 0 {
		nextLabel = fmt.Sprintf("NEXT: MISSION %d", next)
	}
	if ctx.Button("camp.next", uiActionPrimaryRect(240), nextLabel, uiButtonOpts{Kind: uiButtonPrimary, Disabled: next == 0,
		InfoTitle: "NEXT MISSION", InfoBody: "Open the first mission you have not completed yet."}) {
		g.openCampaignMission(next)
		g.campaignUI.focus = ""
		return
	}
	hints := []uiKeyHint{{"ESC", "Main menu"}, {"UP/DOWN", "Move"}, {"LEFT/RIGHT", "Move"}, {"ENTER", "Briefing"}, {"Q", "Prev set"}, {"E", "Next set"}}
	if g.allowCheats {
		hints = append(hints, uiKeyHint{"F10", "Unlock all"})
	}
	ctx.KeyHintBar(hints)

	if g.campaignResultVisible() {
		g.buildCampaignResult(ctx)
		return
	}
	if ctx.Back() {
		g.campaignBackToMainMenu()
	}
}

func (g *Game) campaignBackToMainMenu() {
	// Symbol948.btn_back plays menu.wav.
	g.playSourceSFX("menu.wav", false)
	g.campaignUI.reset()
	g.beginFade(screenMainMenu, fadePurposeScreen)
}

// drawCampaignMissionTile draws a mission tile's content: arena / level art,
// number, title and state.
func (g *Game) drawCampaignMissionTile(dst *ebiten.Image, r Rect, set, level int, info campaignMissionInfo, state int) {
	bandH := math.Min(r.H*0.5, 96)
	band := Rect{X: r.X, Y: r.Y, W: r.W, H: bandH}
	uiFillRound(dst, band, 4, uiInk700)
	g.drawCampaignMissionArt(dst, band, set, level, info.mapID)
	badge := Rect{X: r.X + 4, Y: r.Y + 4, W: uiMeasure(true, uiSizeHint, fmt.Sprint(level)) + 14, H: 20}
	uiFillRound(dst, badge, 3, uiInk900)
	uiTextIn(dst, fmt.Sprint(level), true, uiSizeHint, uiTextLight, badge, uiAlignCenter, 2)
	if info.boss {
		w := uiMeasure(true, uiSizeHint, "BOSS") + 12
		b := Rect{X: r.X + r.W - w - 4, Y: r.Y + bandH - 24, W: w, H: 20}
		uiFillRound(dst, b, 3, uiDanger)
		uiTextIn(dst, "BOSS", true, uiSizeHint, uiTextLight, b, uiAlignCenter, 2)
	}
	y := r.Y + bandH + 14
	lines := wrapTextToWidth(info.title, r.W-8, func(s string) float64 { return uiMeasure(true, uiSizeBody, s) })
	maxLines := 1
	if r.H-bandH > 90 {
		maxLines = 2
	}
	for i, line := range lines {
		if i >= maxLines {
			break
		}
		if i == maxLines-1 && len(lines) > maxLines {
			line = uiFit(line+" "+strings.Join(lines[i+1:], " "), true, uiSizeBody, r.W-8)
		}
		uiText(dst, line, true, uiSizeBody, uiTextDark, r.X+4, y, uiAlignLeft)
		y += 17
	}
	sub := info.mapName
	if set == campaignSetExtra {
		sub = info.mode
	}
	if r.Y+r.H-y > 30 {
		uiText(dst, uiFit(strings.ToUpper(sub), false, uiSizeHint, r.W-8), false, uiSizeHint, uiTextMutedDk, r.X+4, y, uiAlignLeft)
	}
	drawCampaignStateChip(dst, r.X+4, r.Y+r.H-20, 18, state, false)
}

// drawCampaignMissionArt fills r with the GM1 level thumbnail (Symbol 52...)
// or the mission's arena preview.
func (g *Game) drawCampaignMissionArt(dst *ebiten.Image, r Rect, set, level, mapID int) {
	if g.assets == nil {
		return
	}
	var art *SourceRaster
	if set == campaignSetGM1 {
		art = g.assets.CampaignThumbs[level]
	}
	if art == nil {
		art = g.assets.MapPreviewImages[mapID]
	}
	if art == nil && g.customMapThumbCache != nil {
		if thumb := g.customMapThumbCache[mapID]; thumb != nil {
			art = &SourceRaster{Image: thumb}
		}
	}
	if art == nil {
		// GM2 arenas 10-21 have no preview bitmap: a rendered thumbnail
		// arrives over the next frames (preloadCampaignMapThumbs).
		uiTextIn(dst, strings.ToUpper(onlineMapNumberLabel(mapID)), true, uiSizeHint, uiTextMuted, r, uiAlignCenter, 4)
		return
	}
	drawRasterCover(dst, art, r)
}

// drawRasterCover scales raster to cover r (cropping the overflow).
func drawRasterCover(dst *ebiten.Image, raster *SourceRaster, r Rect) {
	if dst == nil || raster == nil || raster.Image == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	b := raster.Image.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return
	}
	s := math.Max(r.W/float64(b.Dx()), r.H/float64(b.Dy()))
	clip := image.Rect(int(math.Ceil(r.X)), int(math.Ceil(r.Y)), int(math.Floor(r.X+r.W)), int(math.Floor(r.Y+r.H)))
	sub, ok := dst.SubImage(clip.Intersect(dst.Bounds())).(*ebiten.Image)
	if !ok {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(-float64(b.Min.X), -float64(b.Min.Y))
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(r.X+(r.W-float64(b.Dx())*s)/2, r.Y+(r.H-float64(b.Dy())*s)/2)
	sub.DrawImage(raster.Image, op)
}

func (g *Game) drawCampaignListInfo(ctx *uiCtx, set, level int) {
	if !ctx.drawing() {
		return
	}
	r := campaignInfoRect
	uiFillRound(ctx.dst, r, 6, uiInk700)
	info := g.campaignMission(set, level)
	state := g.campaignMissionState(set, level)
	art := Rect{X: r.X + 10, Y: r.Y + 10, W: r.W - 20, H: 120}
	uiFillRound(ctx.dst, art, 4, uiInk900)
	g.drawCampaignMissionArt(ctx.dst, art, set, level, info.mapID)
	y := art.Y + art.H + 22
	uiText(ctx.dst, fmt.Sprintf("MISSION %d", level), true, uiSizeHint, uiOrange, r.X+14, y, uiAlignLeft)
	drawCampaignStateChip(ctx.dst, r.X+r.W-14, y-10, 20, state, true)
	y += 24
	uiText(ctx.dst, uiFit(info.title, true, uiSizeSection, r.W-28), true, uiSizeSection, uiTextLight, r.X+14, y, uiAlignLeft)
	y += 24
	uiText(ctx.dst, uiFit(strings.ToUpper(info.mapName), true, uiSizeHint, r.W-28), true, uiSizeHint, uiTextMuted, r.X+14, y, uiAlignLeft)
	y += 24
	rows := [][2]string{{"MODE", info.mode}, {"LIVES", info.lives}}
	for _, row := range rows {
		uiText(ctx.dst, row[0], true, uiSizeHint, uiTextMuted, r.X+14, y, uiAlignLeft)
		uiText(ctx.dst, uiFit(strings.ToUpper(row[1]), true, uiSizeHint, r.W-90), true, uiSizeHint, uiTextLight, r.X+76, y, uiAlignLeft)
		y += 19
	}
	y += 6
	uiText(ctx.dst, "OPPONENTS", true, uiSizeHint, uiOrange, r.X+14, y, uiAlignLeft)
	y += 19
	for i, line := range info.opponents {
		if i >= 4 {
			break
		}
		uiText(ctx.dst, uiFit(line, false, uiSizeHint, r.W-28), false, uiSizeHint, uiTextLight, r.X+14, y, uiAlignLeft)
		y += 18
	}
	if state == 0 {
		uiParagraph(ctx.dst, "Complete the previous mission to unlock it.", false, uiSizeHint, uiTextMuted, r.X+14, y+8, r.W-28, 17, 2)
		return
	}
	left := int((r.Y + r.H - 10 - y) / 17)
	uiParagraph(ctx.dst, info.briefing, false, uiSizeHint, uiTextMuted, r.X+14, y+8, r.W-28, 17, left)
}

// ---------------------------------------------------------------------------
// Briefing

func (g *Game) buildCampaignBriefing(ctx *uiCtx) {
	set := g.campaignSet()
	level := g.campaignLevel
	info := g.campaignMission(set, level)
	ctx.Header(info.title, fmt.Sprintf("CAMPAIGN  /  %s  /  MISSION %d", campaignSetNames[set], level))

	coop := 0
	if g.campaignPlayers[1].Type != 0 {
		coop = 1
	}
	coopBody := "SOLO: player 1 only. CO-OP: player 2 joins on the same keyboard."
	if !g.campaignCoopAllowed() {
		coopBody = "This mission is played solo."
	}
	if v := ctx.Segmented("camp.coop", campaignCoopRect, []string{"SOLO", "CO-OP"}, coop, !g.campaignCoopAllowed(), "PLAYERS", coopBody); v != coop {
		g.setCampaignCoop(v == 1)
	}

	for slot := range g.campaignPlayers {
		g.buildCampaignPlayerCard(ctx, slot)
	}
	g.drawCampaignBriefingCard(ctx, set, level, info)

	if ctx.Button("camp.brief.back", uiActionBackRect(), "MISSIONS", uiButtonOpts{InfoTitle: "BACK", InfoBody: "Back to the mission list."}) {
		g.playSourceSFX("menu.wav", false)
		g.closeCampaignBriefing()
		return
	}
	if ctx.Button("camp.start", campaignStartBtnRect, "START MISSION", uiButtonOpts{Kind: uiButtonPrimary,
		Disabled: g.campaignMissionState(set, level) == 0 || g.fadeActive, InfoTitle: "START MISSION"}) {
		g.startSelectedCampaignMission()
		return
	}
	ctx.KeyHintBar([]uiKeyHint{{"ESC", "Missions"}, {"UP/DOWN", "Move"}, {"LEFT/RIGHT", "Move"}, {"ENTER", "Select"}, {"TAB", "Next control"}})

	if ret := g.campaignLoadoutReturn; ret > 0 && ctx.pass == uiPassUpdate {
		g.campaignLoadoutReturn = 0
		ctx.st.focus = fmt.Sprintf("camp.p%d.edit", ret-1)
	}
	if ctx.Back() {
		g.playSourceSFX("menu.wav", false)
		g.closeCampaignBriefing()
	}
}

// setCampaignCoop is the old P2 HUMAN PLAYER / CLEAR SLOT card buttons.
func (g *Game) setCampaignCoop(on bool) {
	if on {
		if g.campaignCoopAllowed() {
			g.activateCampaignPlayerHit(campaignPlayerHitHumanBase + 1)
		}
		return
	}
	g.activateCampaignPlayerHit(campaignPlayerHitClearBase + 1)
}

func (g *Game) buildCampaignPlayerCard(ctx *uiCtx, slot int) {
	cfg := g.campaignPlayers[slot]
	if slot == 0 && cfg.Type == 0 {
		cfg.Type = 1 // P1 always plays (start*Mission forces it)
	}
	r := campaignPlayerCardRect(slot)
	x, w := r.X+12, r.W-24
	empty := cfg.Type == 0
	kind := "HUMAN"
	if empty {
		kind = "EMPTY"
	}
	ctx.PlayerCard(r, slot, cfg.Color, empty, kind)
	if empty {
		allowed := g.campaignCoopAllowed()
		if ctx.drawing() {
			title, body := "PLAYER 2", "Add a second player on this keyboard: co-op against the mission's enemies."
			if !allowed {
				title, body = "SOLO MISSION", "This mission is played by player 1 alone."
			}
			uiText(ctx.dst, title, true, uiSizeSection, uiTextDark, x, r.Y+70, uiAlignLeft)
			uiParagraph(ctx.dst, body, false, uiSizeBody, uiTextMutedDk, x, r.Y+102, w, 19, 5)
		}
		if ctx.Button("camp.p2.add", Rect{X: x, Y: r.Y + r.H - 56, W: w, H: 40}, "ADD PLAYER 2", uiButtonOpts{Kind: uiButtonPrimary, Size: uiSizeBody,
			Disabled: !allowed, InfoTitle: "CO-OP", InfoBody: "Player 2 uses the second key set from Options."}) {
			g.setCampaignCoop(true)
		}
		return
	}
	if ctx.drawing() {
		name := strings.TrimSpace(cfg.Name)
		if name == "" {
			name = fmt.Sprintf("Player %d", slot+1)
		}
		uiText(ctx.dst, uiFit(strings.ToUpper(name), true, uiSizeButton, w), true, uiSizeButton, uiTextDark, x, r.Y+62, uiAlignLeft)
		preview := Rect{X: x, Y: r.Y + 80, W: w, H: 120}
		uiFillRound(ctx.dst, preview, 6, uiPaperShade)
		g.drawLoadoutCharacterIn(ctx.dst, cfg, uiInset(preview, 4))
		drawLoadoutGearRows(ctx.dst, cfg, x, r.Y+220, w, 20, uiTextMutedDk, uiTextDark)
		uiText(ctx.dst, uiFit(customLocalControlsHint(g.controlConfigs[slot]), false, uiSizeHint, w), false, uiSizeHint, uiTextMutedDk, x, r.Y+306, uiAlignLeft)
	}
	editY := r.Y + r.H - 56
	if slot == 1 {
		editY = r.Y + r.H - 100
		if ctx.Button("camp.p2.remove", Rect{X: x, Y: r.Y + r.H - 52, W: w, H: 36}, "REMOVE PLAYER 2", uiButtonOpts{Kind: uiButtonDanger, Size: uiSizeBody,
			InfoTitle: "SOLO", InfoBody: "Play this mission alone."}) {
			g.setCampaignCoop(false)
			return
		}
	}
	if ctx.Button(fmt.Sprintf("camp.p%d.edit", slot), Rect{X: x, Y: editY, W: w, H: 40}, "EDIT LOADOUT", uiButtonOpts{Size: uiSizeBody,
		InfoTitle: "EDIT LOADOUT", InfoBody: "Character, colour, starting handgun, perk, ability / gear and throwable."}) {
		g.openCampaignLoadout(slot)
	}
}

func (g *Game) drawCampaignBriefingCard(ctx *uiCtx, set, level int, info campaignMissionInfo) {
	if !ctx.drawing() {
		return
	}
	r := campaignBriefCard
	ctx.Card(r, "")
	art := Rect{X: r.X + 10, Y: r.Y + 10, W: r.W - 20, H: 132}
	uiFillRound(ctx.dst, art, 4, uiInk900)
	g.drawCampaignMissionArt(ctx.dst, art, set, level, info.mapID)
	state := g.campaignMissionState(set, level)
	drawCampaignStateChip(ctx.dst, art.X+art.W-8, art.Y+8, 20, state, true)

	x, w := r.X+16, r.W-32
	y := art.Y + art.H + 22
	uiText(ctx.dst, fmt.Sprintf("MISSION %d  /  %s", level, strings.ToUpper(info.mapName)), true, uiSizeHint, uiTextMutedDk, x, y, uiAlignLeft)
	y += 24
	rows := [][2]string{{"MODE", info.mode}, {"YOUR LIVES", info.lives}, {"PICKUPS", info.pickups}}
	if info.rules != "" {
		rows = append(rows, [2]string{"RULES", info.rules})
	}
	if info.returns != "" {
		rows = append(rows, [2]string{"ENEMY RETURNS", info.returns})
	}
	for _, row := range rows {
		uiText(ctx.dst, row[0], true, uiSizeHint, uiTextMutedDk, x, y, uiAlignLeft)
		uiText(ctx.dst, uiFit(strings.ToUpper(row[1]), true, uiSizeHint, w-96), true, uiSizeHint, uiTextDark, x+96, y, uiAlignLeft)
		y += 19
	}
	y += 6
	uiText(ctx.dst, "OPPONENTS", true, uiSizeHint, uiOrangePress, x, y, uiAlignLeft)
	y += 19
	for _, line := range info.opponents {
		uiText(ctx.dst, uiFit(line, true, uiSizeHint, w), true, uiSizeHint, uiTextDark, x, y, uiAlignLeft)
		y += 18
	}
	if len(info.rewards) > 0 {
		names := make([]string, 0, len(info.rewards))
		for _, gun := range info.rewards {
			names = append(names, weaponDisplayName(gun))
		}
		y += 4
		uiText(ctx.dst, "REWARD", true, uiSizeHint, uiOrangePress, x, y, uiAlignLeft)
		uiText(ctx.dst, uiFit(strings.ToUpper(strings.Join(names, " + ")), true, uiSizeHint, w-96), true, uiSizeHint, uiTextDark, x+96, y, uiAlignLeft)
		y += 18
	}
	left := int((r.Y + r.H - 10 - y) / 18)
	uiParagraph(ctx.dst, info.briefing, false, uiSizeBody, uiTextMutedDk, x, y+10, w, 18, left)
}

// ---------------------------------------------------------------------------
// Campaign loadout

func (g *Game) campaignLoadoutBinding(slot int) loadoutBinding {
	return loadoutBinding{
		slot: slot, cfg: g.campaignPlayers[slot],
		unlocked:  g.campaignSelectionUnlocked,
		activate:  func(code int) { g.activateCampaignPlayerHit(code) },
		selectHit: func(editor, n int) int { return campaignSelectHit(slot, editor, n) },
		colorHit:  func(n int) int { return campaignSelectHit(slot, customEditColor, n) },
		gunHit: func(n int) int {
			if n >= 98 {
				return campaignDevelopedGunHit(slot, n)
			}
			return campaignSelectHit(slot, customEditGun, n)
		},
		throwHit:  func(kind int) int { return campaignThrowableChoiceHit(slot, kind) },
		nameHit:   campaignPlayerHitNameBase + slot,
		nameFocus: &g.campaignNameFocus,
	}
}

func (g *Game) openCampaignLoadout(slot int) {
	if slot < 0 || slot >= len(g.campaignPlayers) {
		return
	}
	if slot == 0 && g.campaignPlayers[0].Type == 0 {
		g.campaignPlayers[0].Type = 1
	}
	if g.campaignPlayers[slot].Type == 0 {
		return
	}
	g.campaignLoadoutSlot = slot + 1
	g.campaignNameFocus = 0
	if g.customLoadoutTab < 0 || g.customLoadoutTab >= loadoutTabCount {
		g.customLoadoutTab = loadoutTabCharacter
	}
	g.customLoadoutFocusPending = true
}

func (g *Game) closeCampaignLoadout() {
	if g.campaignLoadoutSlot > 0 {
		g.campaignLoadoutReturn = g.campaignLoadoutSlot
	}
	g.campaignLoadoutSlot = 0
	g.campaignNameFocus = 0
	g.customLoadoutFocusPending = false
}

func (g *Game) buildCampaignLoadout(ctx *uiCtx) {
	slot := g.campaignLoadoutSlot - 1
	other := 1 - slot
	switchable := g.campaignPlayers[other].Type != 0 || other == 0
	g.buildLoadoutScreen(ctx, g.campaignLoadoutBinding(slot), loadoutScreenOpts{
		trail:      fmt.Sprintf("CAMPAIGN  /  %s  /  MISSION %d", campaignSetNames[g.campaignSet()], g.campaignLevel),
		switchable: switchable,
		onSwitch: func(int) {
			g.campaignNameFocus = 0
			g.campaignLoadoutSlot = other + 1
		},
		onClose:  g.closeCampaignLoadout,
		doneInfo: "Back to the mission briefing.",
	})
}

// ---------------------------------------------------------------------------
// Results

// recordCampaignResult remembers the outcome of a campaign match that ended
// by itself (not left through the pause menu) for the results modal.
func (g *Game) recordCampaignResult(won bool) {
	res := campaignResultInfo{open: true, won: won, set: g.campaignSet(), level: g.campaignLevel}
	if won && res.set == campaignSetGM1 && res.level >= 1 && res.level <= len(gm1CampaignRewards) {
		for i, idx := range gm1CampaignRewards[res.level-1] {
			res.unlock[i] = idx + 10
		}
	}
	for _, p := range g.players {
		if p == nil || p.AI || p.IsDouble {
			continue
		}
		acc := 0
		if p.ShotsFired > 0 {
			acc = int(math.Round(float64(p.HitsLanded) / float64(p.ShotsFired) * 100))
		}
		name := strings.TrimSpace(p.Name)
		if name == "" {
			name = fmt.Sprintf("Player %d", p.ID)
		}
		res.lines = append(res.lines, fmt.Sprintf("%s:  %d KILLS  /  %d DEATHS  /  %d%% ACCURACY", strings.ToUpper(name), p.Kills, p.Deaths, acc))
	}
	g.campaignResult = res
}

func (g *Game) campaignResultVisible() bool {
	return g.campaignResult.open || (g.campaignShowUnlock >= 1 && g.campaignShowUnlock <= 10)
}

func (g *Game) closeCampaignResult() {
	// Symbol947 Continue plays menu.wav after clearing the unlock popup.
	g.playSourceSFX("menu.wav", false)
	g.campaignResult.open = false
	g.campaignShowUnlock = 0
}

func (g *Game) buildCampaignResult(ctx *uiCtx) {
	res := g.campaignResult
	if !res.open {
		// GM1 unlock popup without a recorded match (old state only).
		res = campaignResultInfo{won: true, set: campaignSetGM1, level: g.campaignShowUnlock}
		for i, idx := range gm1CampaignRewards[res.level-1] {
			res.unlock[i] = idx + 10
		}
	}
	r := campaignResultRect
	title := "MISSION COMPLETE"
	if !res.won {
		title = "MISSION FAILED"
	}
	ctx.BeginModal(r, title)
	info := g.campaignMission(res.set, res.level)
	if ctx.drawing() {
		accent := uiOK
		if !res.won {
			accent = uiDanger
		}
		uiFillRect(ctx.dst, Rect{X: r.X, Y: r.Y + 44, W: r.W, H: 4}, accent)
		uiText(ctx.dst, fmt.Sprintf("%s  /  MISSION %d: %s", campaignSetNames[res.set], res.level, info.title), true, uiSizeBody, uiTextMutedDk, r.X+16, r.Y+68, uiAlignLeft)
		y := r.Y + 100
		for _, line := range res.lines {
			uiText(ctx.dst, uiFit(line, true, uiSizeBody, r.W-32), true, uiSizeBody, uiTextDark, r.X+16, y, uiAlignLeft)
			y += 22
		}
		if res.unlock[0] != 0 {
			y += 8
			uiText(ctx.dst, "NEW WEAPONS UNLOCKED", true, uiSizeButton, uiOrangePress, r.X+16, y, uiAlignLeft)
			y += 18
			for i, gun := range res.unlock {
				g.assets.EnsureGunLibrary()
				cell := Rect{X: r.X + 16 + float64(i)*246, Y: y, W: 238, H: 92}
				uiFillRound(ctx.dst, cell, 5, uiPaperShade)
				drawSetupThumbFit(ctx.dst, g.gunLibraryRaster(gun), Rect{X: cell.X + 6, Y: cell.Y + 4, W: cell.W - 12, H: 60}, 1)
				uiTextIn(ctx.dst, strings.ToUpper(weaponDisplayName(gun)), true, uiSizeHint, uiTextDark, Rect{X: cell.X, Y: cell.Y + 66, W: cell.W, H: 22}, uiAlignCenter, 4)
			}
		} else if !res.won {
			uiParagraph(ctx.dst, "The enemy won this time. Retry with another loadout or a co-op partner.", false, uiSizeBody, uiTextMutedDk, r.X+16, y+10, r.W-32, 19, 3)
		} else if next := g.campaignNextMission(res.set); next > 0 {
			uiText(ctx.dst, fmt.Sprintf("Mission %d is available.", next), false, uiSizeBody, uiTextMutedDk, r.X+16, y+10, uiAlignLeft)
		}
	}
	bw := (r.W - 16*3) / 2
	if ctx.Button("camp.res.retry", Rect{X: r.X + 16, Y: r.Y + r.H - 60, W: bw, H: 44}, "RETRY", uiButtonOpts{
		InfoTitle: "RETRY", InfoBody: "Play this mission again with the same players."}) {
		g.closeCampaignResult()
		g.campaignSetGM2 = res.set == campaignSetGM2
		g.campaignSetExtra = res.set == campaignSetExtra
		g.campaignLevel = res.level
		g.startSelectedCampaignMission()
		ctx.EndModal()
		return
	}
	if ctx.Button("camp.res.ok", Rect{X: r.X + 32 + bw, Y: r.Y + r.H - 60, W: bw, H: 44}, "CONTINUE", uiButtonOpts{Kind: uiButtonPrimary}) {
		g.closeCampaignResult()
	}
	ctx.EndModal()
	if ctx.Back() {
		g.closeCampaignResult()
	}
}

// preloadCampaignMapThumbs renders at most one missing arena thumbnail per
// frame (GM2 arenas without a preview bitmap), like the Custom Game MAPS tab.
func (g *Game) preloadCampaignMapThumbs() {
	if g.assets == nil || g.campaignSet() == campaignSetGM1 {
		return
	}
	set := g.campaignSet()
	for level := 1; level <= campaignMissionCount(set); level++ {
		id := g.campaignMission(set, level).mapID
		if id <= 0 || g.assets.MapPreviewImages[id] != nil {
			continue
		}
		if g.customMapThumbCache != nil && g.customMapThumbCache[id] != nil {
			continue
		}
		g.ensureOnlineMapThumb(id)
		return
	}
}

// drawCampaignStateChip draws the LOCKED / AVAILABLE / COMPLETED chip with its
// left edge (or right edge when alignRight) at x.
func drawCampaignStateChip(dst *ebiten.Image, x, y, h float64, state int, alignRight bool) {
	fill, text := campaignStateColor(state)
	label := campaignStateLabel(state)
	w := uiMeasure(true, uiSizeHint, label) + 14
	if alignRight {
		x -= w
	}
	chip := Rect{X: x, Y: y, W: w, H: h}
	uiFillRound(dst, chip, 3, fill)
	uiTextIn(dst, label, true, uiSizeHint, text, chip, uiAlignCenter, 2)
}
