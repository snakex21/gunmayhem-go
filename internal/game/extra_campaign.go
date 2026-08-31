package game

import (
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	extraCampaignHitMissionBase = 3000
	extraCampaignHitStart       = 3100
	extraCampaignHitPrevPage    = 3101
	extraCampaignHitNextPage    = 3102
	extraCampaignMissionCount   = 20
	extraCampaignPageSize       = 12
)

var extraCampaignTitles = [extraCampaignMissionCount]string{
	"Reinforcement Run",
	"Close Quarters",
	"Eastern Front",
	"Sniper Nest",
	"Rocket Rain",
	"Burning Point",
	"Heavy Metal",
	"Last Stand",
	"Red Sky",
	"Old Guard",
	"Juggernaut",
	"Portal Storm",
	"Gravity Break",
	"Knockback Lab",
	"Arms Roulette",
	"Bullet Furnace",
	"Gravity Well",
	"Bombardment",
	"Shock Reactor",
	"Mayhem Crown",
}

type extraCampaignEnemy struct {
	ID    int
	Name  string
	Color int
	Shirt int
	Hat   int
	Gun   int
	Perk  int
	Lives int
}

type extraCampaignMission struct {
	MapNumber   int
	PlayerLives int
	Crates      bool
	Powerups    bool
	Mode        string
	Briefing    string
	RuleText    string
	Modifiers   []int
	Hazard      extraCampaignHazard
	Enemies     []extraCampaignEnemy
}

// EXTRA is an original continuation built for this port. Enemy life pools are
// deliberately independent from the human life count: later missions contain
// many reinforcements without putting more than two readable opponents on the
// arena at once.
var extraCampaignMissions = [extraCampaignMissionCount]extraCampaignMission{
	{MapNumber: 8, PlayerLives: 6, Crates: true, Powerups: true, Mode: "REINFORCEMENT SQUAD", Briefing: "Two fast enemies control both sides.", Enemies: []extraCampaignEnemy{
		{3, "Rapid", 3, 4, 9, 99, 0, 4}, {4, "Scout", 8, 2, 3, 91, 0, 4},
	}},
	{MapNumber: 11, PlayerLives: 6, Crates: true, Powerups: true, Mode: "SHOTGUN DUEL", Briefing: "One KS-23 hunter with a large life pool.", Enemies: []extraCampaignEnemy{
		{3, "Hunter", 9, 8, 15, 165, 0, 9},
	}},
	{MapNumber: 9, PlayerLives: 6, Crates: true, Powerups: false, Mode: "EASTERN SQUAD", Briefing: "Automatic rifles attack from two angles.", Enemies: []extraCampaignEnemy{
		{3, "Red Star", 2, 14, 4, 96, 0, 5}, {4, "Comrade", 10, 6, 31, 131, 7, 5},
	}},
	{MapNumber: 12, PlayerLives: 5, Crates: false, Powerups: true, Mode: "SNIPER CROSS-FIRE", Briefing: "Hard but fair: break their firing lanes.", Enemies: []extraCampaignEnemy{
		{3, "Marksman", 4, 15, 41, 151, 0, 4}, {4, "Ghost", 7, 5, 8, 158, 4, 4},
	}},
	{MapNumber: 5, PlayerLives: 7, Crates: false, Powerups: true, Mode: "EXPLOSIVE SQUAD", Briefing: "Keep moving between rocket impacts.", Enemies: []extraCampaignEnemy{
		{3, "Rocketeer", 5, 9, 10, 168, 2, 4}, {4, "Grenadier", 1, 11, 17, 173, 8, 4},
	}},
	{MapNumber: 6, PlayerLives: 6, Crates: true, Powerups: true, Mode: "FIRE TEAM", Briefing: "Flares ignite; the flamethrower closes in.", Enemies: []extraCampaignEnemy{
		{3, "Firestarter", 8, 13, 42, 181, 5, 5}, {4, "Torch", 3, 12, 1, 187, 0, 5},
	}},
	{MapNumber: 3, PlayerLives: 7, Crates: true, Powerups: false, Mode: "HEAVY BOSS", Briefing: "One MG42 gunner with thirteen lives.", Enemies: []extraCampaignEnemy{
		{3, "Gunner", 9, 4, 9, 141, 2, 13},
	}},
	{MapNumber: 1, PlayerLives: 8, Crates: false, Powerups: true, Mode: "FINAL SQUAD", Briefing: "Grenades and energy fire demand movement.", Enemies: []extraCampaignEnemy{
		{3, "Warden", 10, 8, 15, 175, 2, 8}, {4, "Overseer", 1, 14, 41, 185, 5, 8},
	}},
	{MapNumber: 4, PlayerLives: 7, Crates: false, Powerups: true, Mode: "AIRSTRIKE BOSS", Briefing: "Watch the warning marker before each strike.", Enemies: []extraCampaignEnemy{
		{3, "Sky Marshal", 4, 9, 10, 90, 6, 10},
	}},
	{MapNumber: 10, PlayerLives: 6, Crates: true, Powerups: false, Mode: "OLD-SCHOOL SQUAD", Briefing: "Slow weapons, powerful single hits.", Enemies: []extraCampaignEnemy{
		{3, "Musketeer", 6, 7, 31, 182, 0, 5}, {4, "Ranger", 2, 3, 15, 177, 3, 5},
	}},
	{MapNumber: 7, PlayerLives: 7, Crates: true, Powerups: true, Mode: "JUGGERNAUT DUEL", Briefing: "A blast-shielded harpoon boss hunts you.", Enemies: []extraCampaignEnemy{
		{3, "Juggernaut", 10, 14, 42, 179, 6, 13},
	}},
	{MapNumber: 2, PlayerLives: 8, Crates: false, Powerups: true, Mode: "PORTAL SUPPORT", Briefing: "Portals reposition you into energy fire.", Enemies: []extraCampaignEnemy{
		{3, "Portal Tech", 5, 5, 8, 189, 3, 7}, {4, "Storm", 1, 12, 41, 185, 5, 7},
	}},
	{MapNumber: 8, PlayerLives: 6, Crates: true, Powerups: false, Mode: "ZERO-G PURSUIT", Briefing: "Fight through the air; fuel never runs out.", RuleText: "LOW GRAVITY + JETPACKS", Modifiers: []int{customModifierLowGravity, customModifierJetpacks}, Enemies: []extraCampaignEnemy{
		{3, "Sky Raider", 4, 11, 10, 213, 7, 5}, {4, "Drifter", 8, 5, 17, 212, 0, 5},
	}},
	{MapNumber: 11, PlayerLives: 6, Crates: false, Powerups: true, Mode: "IMPACT TEST", Briefing: "Every hit launches twice as hard.", RuleText: "DOUBLE KNOCKBACK", Modifiers: []int{customModifierDoubleKnockback}, Enemies: []extraCampaignEnemy{
		{3, "Ram", 9, 8, 15, 167, 6, 11},
	}},
	{MapNumber: 9, PlayerLives: 7, Crates: true, Powerups: false, Mode: "WEAPON ROULETTE", Briefing: "Everyone receives a new weapon after each return.", RuleText: "RANDOM WEAPON / LIFE + FAST CRATES", Modifiers: []int{customModifierRandomWeaponEachLife, customModifierCratesEverywhere}, Enemies: []extraCampaignEnemy{
		{3, "Wildcard", 2, 13, 31, 208, 0, 6}, {4, "Dealer", 10, 4, 42, 204, 3, 6},
	}},
	{MapNumber: 3, PlayerLives: 7, Crates: false, Powerups: true, Mode: "ENDLESS BELTS", Briefing: "The Maxim never reloads and never slows down.", RuleText: "INFINITE AMMO + FAST MOVEMENT", Modifiers: []int{customModifierInfiniteAmmo, customModifierFastMovement}, Enemies: []extraCampaignEnemy{
		{3, "Furnace", 5, 14, 9, 199, 2, 14},
	}},
	{MapNumber: 12, PlayerLives: 6, Crates: true, Powerups: false, Mode: "HEAVY GRAVITY", Briefing: "Short arcs and boosted jumps change every route.", RuleText: "HIGH GRAVITY + SUPER JUMP", Modifiers: []int{customModifierHighGravity, customModifierSuperJump}, Enemies: []extraCampaignEnemy{
		{3, "Anchor", 6, 7, 4, 208, 6, 5}, {4, "Weight", 3, 3, 15, 210, 0, 5},
	}},
	{MapNumber: 4, PlayerLives: 7, Crates: false, Powerups: true, Mode: "DYNAMITE WEATHER", Briefing: "Watch the warning: neutral dynamite falls from above.", RuleText: "DYNAMITE RAIN", Hazard: extraCampaignHazardDynamiteRain, Enemies: []extraCampaignEnemy{
		{3, "Spotter", 4, 9, 10, 169, 2, 5}, {4, "Sapper", 1, 11, 17, 173, 8, 5},
	}},
	{MapNumber: 2, PlayerLives: 7, Crates: true, Powerups: true, Mode: "REACTOR PULSE", Briefing: "The reactor periodically throws everyone from its core.", RuleText: "CENTER SHOCK PULSE", Hazard: extraCampaignHazardShockPulse, Enemies: []extraCampaignEnemy{
		{3, "Conductor", 5, 5, 8, 189, 3, 6}, {4, "Voltage", 1, 12, 41, 185, 5, 6},
	}},
	{MapNumber: 7, PlayerLives: 9, Crates: true, Powerups: true, Mode: "FINAL CHAOS", Briefing: "New weapons each life in low gravity with brutal impacts.", RuleText: "LOW GRAVITY + RANDOM GUNS + X2 KNOCKBACK", Modifiers: []int{customModifierLowGravity, customModifierRandomWeaponEachLife, customModifierDoubleKnockback}, Enemies: []extraCampaignEnemy{
		{3, "Mayhem Crown", 10, 14, 42, 213, 6, 17},
	}},
}

func extraCampaignCardRect(index int) Rect {
	col := index % 4
	row := index / 4
	return Rect{X: 28 + float64(col)*216, Y: 116 + float64(row)*132, W: 198, H: 116}
}

func extraCampaignPageCount() int {
	return (extraCampaignMissionCount + extraCampaignPageSize - 1) / extraCampaignPageSize
}

func extraCampaignPageRange(page int) (start, end int) {
	if page < 0 {
		page = 0
	}
	if page >= extraCampaignPageCount() {
		page = extraCampaignPageCount() - 1
	}
	start = page * extraCampaignPageSize
	end = start + extraCampaignPageSize
	if end > extraCampaignMissionCount {
		end = extraCampaignMissionCount
	}
	return start, end
}

var (
	extraCampaignPrevPageRect = Rect{X: 238, Y: 528, W: 122, H: 38}
	extraCampaignNextPageRect = Rect{X: 376, Y: 528, W: 122, H: 38}
)

func (g *Game) unlockAllExtraCampaignLevelsCheat() {
	for i := range g.extraCampaignLevels {
		if g.extraCampaignLevels[i] == 0 {
			g.extraCampaignLevels[i] = 1
		}
	}
	_ = g.saveProgress()
}

func (g *Game) updateExtraCampaignMenu() {
	if g.allowCheats && inpututil.IsKeyJustPressed(ebiten.KeyF10) && g.campaignPhase == 1 {
		g.unlockAllExtraCampaignLevelsCheat()
		g.playSourceSFX("menu.wav", false)
	}
	if g.extraCampaignPage < 0 || g.extraCampaignPage >= extraCampaignPageCount() {
		g.extraCampaignPage = 0
	}
	if g.campaignPhase == 1 {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) && g.extraCampaignPage > 0 {
			g.extraCampaignPage--
			g.playSourceSFX("menu.wav", false)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) && g.extraCampaignPage+1 < extraCampaignPageCount() {
			g.extraCampaignPage++
			g.playSourceSFX("menu.wav", false)
		}
	}
	if g.campaignPhase != 1 {
		g.updateCampaignPlayerCards()
		g.updateCampaignNameInput()
	}
	if g.campaignPhase == 2 {
		g.campaignDetailAlpha += .2
		if g.campaignDetailAlpha >= 1 {
			g.campaignDetailAlpha = 1
			g.campaignPhase = 3
		}
	} else if g.campaignPhase == 4 {
		g.campaignDetailAlpha -= .2
		if g.campaignDetailAlpha <= 0 {
			g.campaignDetailAlpha = 0
			g.campaignPhase = 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.campaignPhase == 1 {
			g.beginFade(screenMainMenu, fadePurposeScreen)
		} else if g.campaignPhase == 3 {
			g.campaignPhase = 4
			g.campaignNameFocus = 0
		}
		return
	}

	mx, my := ebiten.CursorPosition()
	x, y := float64(mx), float64(my)
	hit := campaignHitNone
	if g.campaignPhase == 1 {
		hit = campaignSetHitAt(x, y)
		back := sourceMenuHitRect(campaignBackLocal, 27, 525.15, 1, 1)
		if hit == campaignHitNone && back.Contains(x, y) {
			hit = campaignHitBack
		} else if hit == campaignHitNone && g.extraCampaignPage > 0 && extraCampaignPrevPageRect.Contains(x, y) {
			hit = extraCampaignHitPrevPage
		} else if hit == campaignHitNone && g.extraCampaignPage+1 < extraCampaignPageCount() && extraCampaignNextPageRect.Contains(x, y) {
			hit = extraCampaignHitNextPage
		} else if hit == campaignHitNone {
			start, end := extraCampaignPageRange(g.extraCampaignPage)
			for i := start; i < end; i++ {
				if extraCampaignCardRect(i-start).Contains(x, y) && g.extraCampaignLevels[i] != 0 {
					hit = extraCampaignHitMissionBase + i + 1
					break
				}
			}
		}
	} else if g.campaignPhase == 3 {
		if playerHit := g.campaignPlayerHitAt(x, y); playerHit != campaignHitNone {
			hit = playerHit
		}
		back := sourceMenuHitRect(campaignBackLocal, 27, 525.15, 1, 1)
		start := sourceMenuHitRect(campaignStartLocal, 469.1, 522.15, 1, 1)
		if hit == campaignHitNone && back.Contains(x, y) {
			hit = campaignHitBack
		} else if hit == campaignHitNone && start.Contains(x, y) {
			hit = extraCampaignHitStart
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.campaignPressed = hit
		return
	}
	if !inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		return
	}
	pressed := g.campaignPressed
	g.campaignPressed = campaignHitNone
	if pressed == campaignHitNone || pressed != hit {
		return
	}
	if g.activateCampaignPlayerHit(pressed) {
		return
	}
	switch {
	case pressed == campaignHitSetGM1:
		g.campaignSetExtra = false
		g.campaignSetGM2 = false
		g.campaignLevel = 0
		g.campaignPhase = 1
		g.playSourceSFX("menu.wav", false)
	case pressed == campaignHitSetGM2:
		g.campaignSetExtra = false
		g.campaignSetGM2 = true
		g.campaignLevel = 0
		g.playSourceSFX("menu.wav", false)
	case pressed == campaignHitSetExtra:
		// Already selected.
	case pressed == campaignHitBack:
		g.playSourceSFX("menu.wav", false)
		if g.campaignPhase == 1 {
			g.beginFade(screenMainMenu, fadePurposeScreen)
		} else if g.campaignPhase == 3 {
			g.campaignPhase = 4
			g.campaignNameFocus = 0
		}
	case pressed == extraCampaignHitStart:
		g.playSourceSFX("menu.wav", false)
		g.startExtraCampaignMission()
	case pressed == extraCampaignHitPrevPage && g.extraCampaignPage > 0:
		g.extraCampaignPage--
		g.playSourceSFX("menu.wav", false)
	case pressed == extraCampaignHitNextPage && g.extraCampaignPage+1 < extraCampaignPageCount():
		g.extraCampaignPage++
		g.playSourceSFX("menu.wav", false)
	case pressed > extraCampaignHitMissionBase && pressed <= extraCampaignHitMissionBase+len(extraCampaignMissions):
		g.campaignLevel = pressed - extraCampaignHitMissionBase
		g.campaignPhase = 2
		g.campaignDetailAlpha = 0
		g.campaignNameFocus = 0
		for i := range g.campaignEditor {
			g.campaignEditor[i] = 0
		}
		g.playSourceSFX("menu.wav", false)
	}
}

func extraCampaignEnemyLives(spec extraCampaignMission) int {
	total := 0
	for _, enemy := range spec.Enemies {
		total += enemy.Lives
	}
	return total
}

func extraCampaignEnemyRespawns(spec extraCampaignMission) int {
	total := 0
	for _, enemy := range spec.Enemies {
		if enemy.Lives > 1 {
			total += enemy.Lives - 1
		}
	}
	return total
}

func (g *Game) drawExtraCampaignMenu(screen *ebiten.Image) {
	drawSourceRaster(screen, g.assets.CampaignBase, 0, 0, 1, 1, 1)
	drawSourceMenuText(screen, "CAMPAIGN", menuFontCondensed, 42.4, color.Black, 34, 27.75)
	if g.campaignPhase == 1 {
		g.drawCampaignSetTabs(screen)
	}
	drawSourceMenuText(screen, "EXTRA CAMPAIGN - 20 MISSIONS", menuFontCondensedExtraBold, 23, color.Black, 28, 80)
	drawSourceMenuText(screen, "New enemies, arena hazards and changing match rules.", menuFontTwCen, 14, color.Black, 390, 86)
	drawSourceRaster(screen, g.assets.CampaignBack, 27, 525.15, 1, 1, 1)
	drawSourceMenuText(screen, "Back", menuFontCondensed, 26.5, color.Black, 74.05, 536.35)
	if g.campaignPhase != 1 {
		a := g.campaignDetailAlpha
		if a > 0 {
			g.drawCampaignPlayerCards(screen, a)
			g.drawExtraCampaignMissionInfo(screen, a)
			if g.campaignPhase == 3 {
				drawSourceRaster(screen, g.assets.CampaignStartButton, 469.1, 522.15, 1, 1, a)
				drawSourceMenuText(screen, "Start Mission", menuFontCondensedExtraBold, 30,
					color.NRGBA{A: uint8(255 * a)}, 580, 529)
			}
		}
		return
	}

	start, end := extraCampaignPageRange(g.extraCampaignPage)
	for i := start; i < end; i++ {
		spec := extraCampaignMissions[i]
		r := extraCampaignCardRect(i - start)
		state := g.extraCampaignLevels[i]
		fill := color.NRGBA{R: 0xb8, G: 0xb8, B: 0xb8, A: 0xff}
		if state == 0 {
			fill = color.NRGBA{R: 0x72, G: 0x72, B: 0x72, A: 0xff}
		}
		if g.campaignLevel == i+1 {
			fill = color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
		}
		ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, fill)
		drawSourceMenuText(screen, extraCampaignTitles[i], menuFontCondensedExtraBold, 16.5, color.Black, r.X+7, r.Y+7)
		drawSourceMenuText(screen, sourceMapDisplayNames[spec.MapNumber], menuFontTwCen, 11.5, color.Black, r.X+7, r.Y+31)
		drawSourceMenuText(screen, spec.Mode, menuFontCondensedExtraBold, 10.5, color.Black, r.X+7, r.Y+51)
		drawSourceMenuText(screen, "ENEMY RETURNS  "+strconv.Itoa(extraCampaignEnemyRespawns(spec)), menuFontCondensedExtraBold, 11, color.Black, r.X+7, r.Y+71)
		drawSourceMenuText(screen, gm2CampaignStateLabel(state), menuFontTwCen, 11.5, color.Black, r.X+7, r.Y+94)
		drawSourceMenuText(screen, "#"+itoaFast(i+1), menuFontCondensedExtraBold, 18, color.Black, r.X+r.W-32, r.Y+93)
	}

	if g.extraCampaignPage > 0 {
		drawModifierButton(screen, extraCampaignPrevPageRect, "< PREVIOUS", false)
	}
	if g.extraCampaignPage+1 < extraCampaignPageCount() {
		drawModifierButton(screen, extraCampaignNextPageRect, "NEXT >", false)
	}
	drawSourceMenuText(screen, "PAGE "+strconv.Itoa(g.extraCampaignPage+1)+" / "+strconv.Itoa(extraCampaignPageCount()), menuFontCondensedExtraBold, 13, color.Black, 514, 538)
	drawSourceMenuText(screen, "CLICK A MISSION TO CUSTOMIZE", menuFontCondensedExtraBold, 15, color.Black, 650, 538)
}

func (g *Game) drawExtraCampaignMissionInfo(screen *ebiten.Image, alpha float64) {
	if g.campaignLevel < 1 || g.campaignLevel > len(extraCampaignMissions) || alpha <= 0 {
		return
	}
	spec := extraCampaignMissions[g.campaignLevel-1]
	a := uint8(255 * alpha)
	panel := color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: a}
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: a}
	orange := color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: a}
	ebitenutil.DrawRect(screen, 467.1, 100, 405, 400, panel)
	drawSourceMenuText(screen, "MISSION "+strconv.Itoa(g.campaignLevel), menuFontCondensedExtraBold, 17, orange, 489, 122)
	drawSourceMenuText(screen, extraCampaignTitles[g.campaignLevel-1], menuFontTwCen, 29, white, 489, 151)
	drawSourceMenuText(screen, sourceMapDisplayNames[spec.MapNumber], menuFontCondensedExtraBold, 17, white, 489, 202)
	drawSourceMenuText(screen, "YOUR LIVES: "+strconv.Itoa(spec.PlayerLives), menuFontCondensedExtraBold, 15, white, 489, 232)
	drawSourceMenuText(screen, "ENEMY RETURNS: "+strconv.Itoa(extraCampaignEnemyRespawns(spec)), menuFontCondensedExtraBold, 15, white, 489, 255)
	drawSourceMenuText(screen, "TYPE: "+spec.Mode, menuFontCondensedExtraBold, 14, orange, 489, 279)
	rules := spec.RuleText
	if rules == "" {
		rules = "STANDARD ARENA RULES"
	}
	drawSourceMenuText(screen, "RULES: "+rules, menuFontCondensedExtraBold, 11.5, white, 489, 302)
	drawSourceMenuText(screen, "OPPONENTS", menuFontCondensedExtraBold, 16, orange, 489, 327)
	for i, enemy := range spec.Enemies {
		line := enemy.Name + " - " + weaponDisplayName(enemy.Gun) + " x" + strconv.Itoa(enemy.Lives)
		drawSourceMenuText(screen, line, menuFontCondensed, 14, white, 489, 353+float64(i)*25)
	}
	drawSourceMenuText(screen, spec.Briefing, menuFontTwCen, 13.5, white, 489, 414)
	drawSourceMenuText(screen, "Customize character, handgun, perk and throwable on the left.", menuFontTwCen, 12.5, white, 489, 451)
}

func (g *Game) startExtraCampaignMission() {
	level := g.campaignLevel
	if level < 1 || level > len(extraCampaignMissions) || g.fadeActive || g.extraCampaignLevels[level-1] == 0 {
		return
	}
	spec := extraCampaignMissions[level-1]
	arena, ok := g.ensureMap(spec.MapNumber)
	if !ok {
		return
	}

	players := make([]*Player, 0, 4)
	for slot, cfg := range g.campaignPlayers {
		if slot == 0 && cfg.Type == 0 {
			cfg.Type = 1
		}
		if cfg.Type == 0 {
			continue
		}
		id := slot + 1
		p := NewPlayer(id, arena)
		p.Controls = g.controlConfigs[id-1]
		p.Name = cfg.Name
		p.PlayerColor = cfg.Color
		p.ShirtNumber = cfg.Shirt
		p.HatNumber = cfg.Hat
		p.EyeNumber = cfg.Eyes
		p.DefaultWeapon = cfg.Gun
		p.PerkNumber = cfg.Perk
		p.Team = 1
		p.configureSourceGameMode(SourceGameModeNormal, spec.PlayerLives, arena)
		p.ThrowableType = normalizedThrowableType(cfg.Throwable)
		players = append(players, p)
	}

	for _, src := range spec.Enemies {
		e := NewPlayer(src.ID, arena)
		e.Controls = g.controlConfigs[src.ID-1]
		e.AI = true
		e.Name = src.Name
		e.PlayerColor = src.Color
		e.ShirtNumber = src.Shirt
		e.HatNumber = src.Hat
		e.DefaultWeapon = src.Gun
		e.PerkNumber = src.Perk
		e.Team = 2
		e.configureSourceGameMode(SourceGameModeNormal, src.Lives, arena)
		players = append(players, e)
	}

	g.arena = arena
	g.players = players
	g.GameMode = SourceGameModeNormal
	g.TotalLives = spec.PlayerLives
	g.TeamGame = len(g.campaignPlayers) > 1 && g.campaignPlayers[1].Type == 1
	g.CrateON = spec.Crates
	g.PowerON = spec.Powerups
	g.crateWeapons = developedCrateWeapons()
	g.customMatchActive = len(spec.Modifiers) > 0
	g.matchModifiers = make(map[int]bool)
	for _, modifier := range spec.Modifiers {
		g.matchModifiers[modifier] = true
	}
	g.applyCustomModifierSetup()
	g.campaignMode = true
	g.campaignSetGM2 = false
	g.campaignSetExtra = true
	g.campaignLevel = level
	g.extraHazardFrame = 0
	g.GameWin = false
	g.teamGameWin = false
	g.matchWinCountdown = 0
	g.soloWinFrame = 0
	g.winnerPlayerID = 0
	g.nextEntityID = 5
	g.seenDeaths = make(map[int]int, len(players))
	for _, p := range players {
		g.seenDeaths[p.ID] = p.DeathSerial
	}
	g.resetArenaState()
	g.beginFade(screenGameplay, fadePurposeScreen)
}

func (g *Game) completeExtraCampaignLevel(level int) {
	if level < 1 || level > len(g.extraCampaignLevels) {
		return
	}
	g.extraCampaignLevels[level-1] = 2
	if level < len(g.extraCampaignLevels) && g.extraCampaignLevels[level] == 0 {
		g.extraCampaignLevels[level] = 1
	}
	_ = g.saveProgress()
}
