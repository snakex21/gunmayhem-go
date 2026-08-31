package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	chaosEffectNone = iota
	chaosEffectOneShotgunAll
	chaosEffectRandomShotguns
	chaosEffectOneRandomGunAll
	chaosEffectRandomGuns
	chaosEffectWeaponShuffle
	chaosEffectDynamiteRain
	chaosEffectGrenadeRain
	chaosEffectFireRain
	chaosEffectMixedExplosiveRain
	chaosEffectShockPulse
	chaosEffectLowGravity
	chaosEffectHighGravity
	chaosEffectDoubleKnockback
	chaosEffectSuperJump
	chaosEffectFastMovement
	chaosEffectJetpacks
	chaosEffectMiniEveryone
	chaosEffectInvisibleEveryone
	chaosEffectSpeedBoost
	chaosEffectCrateStorm
	chaosEffectPowerupStorm
	chaosEffectTeleportShuffle
	chaosEffectNoCrates
	chaosEffectInfiniteAmmo
	chaosEffectIceFloor
	chaosEffectWindLeft
	chaosEffectWindRight
	chaosEffectUpdraft
	chaosEffectEarthquake
	chaosEffectSkyLaunch
	chaosEffectPistolsOnly
	chaosEffectSMGsOnly
	chaosEffectRiflesOnly
	chaosEffectLMGsOnly
	chaosEffectSnipersOnly
	chaosEffectLaunchersOnly
	chaosEffectRandomThrowables
	chaosEffectGrenadeRefill
	chaosEffectExtraLife
	chaosEffectShieldEveryone
	chaosEffectFireFloor
	chaosEffectWeaponRoulette
	chaosEffectTeleportStorm
	chaosEffectZeroGravity
	chaosEffectCount
)

const (
	menuHitChaosSetup = 8700 + iota
	menuHitChaosBack
	menuHitChaosToggle
	menuHitChaosIntervalMinus
	menuHitChaosIntervalPlus
	menuHitChaosSelectAll
	menuHitChaosClear
	menuHitChaosEffectBase = 8800
)

type chaosEffectSpec struct {
	ID    int
	Label string
	Hint  string
}

var chaosEffectSpecs = []chaosEffectSpec{
	{chaosEffectOneShotgunAll, "ONE SHOTGUN FOR ALL", "Everyone instantly receives the same randomly selected shotgun."},
	{chaosEffectRandomShotguns, "RANDOM SHOTGUNS", "Every player instantly receives a different random shotgun."},
	{chaosEffectOneRandomGunAll, "ONE RANDOM GUN FOR ALL", "Everyone instantly receives the same random weapon."},
	{chaosEffectRandomGuns, "RANDOM GUNS FOR ALL", "Every player instantly receives a different random weapon."},
	{chaosEffectWeaponShuffle, "WEAPON SHUFFLE", "Current weapons are rotated between the active players."},
	{chaosEffectDynamiteRain, "DYNAMITE RAIN", "Neutral sticks of dynamite keep falling from the sky."},
	{chaosEffectGrenadeRain, "GRENADE RAIN", "Random military grenades keep falling across the arena."},
	{chaosEffectFireRain, "FIRE RAIN", "Molotovs rain from above and leave burning patches on platforms."},
	{chaosEffectMixedExplosiveRain, "EXPLOSIVE STORM", "Dynamite, grenades and Molotovs rain together."},
	{chaosEffectShockPulse, "SHOCK PULSES", "Periodic arena-wide pulses throw everybody away from the centre."},
	{chaosEffectLowGravity, "LOW GRAVITY", "Gravity is reduced until the next chaos event."},
	{chaosEffectHighGravity, "HIGH GRAVITY", "Gravity is increased until the next chaos event."},
	{chaosEffectDoubleKnockback, "X2 KNOCKBACK", "Weapon and grenade impacts launch players roughly twice as hard."},
	{chaosEffectSuperJump, "SUPER JUMP", "All players jump much higher until the next chaos event."},
	{chaosEffectFastMovement, "FAST MOVEMENT", "All players move faster until the next chaos event."},
	{chaosEffectJetpacks, "JETPACKS", "Everybody receives unlimited jetpack fuel until the next event."},
	{chaosEffectMiniEveryone, "MINI EVERYONE", "All active players shrink for the duration of the chaos event."},
	{chaosEffectInvisibleEveryone, "INVISIBLE EVERYONE", "All active players become invisible for the event duration."},
	{chaosEffectSpeedBoost, "SPEED BOOST", "All active players receive the source speed powerup effect."},
	{chaosEffectCrateStorm, "CRATE STORM", "Several weapon crates fall immediately and continue spawning rapidly."},
	{chaosEffectPowerupStorm, "POWERUP STORM", "Powerups spawn repeatedly around the arena."},
	{chaosEffectTeleportShuffle, "TELEPORT SHUFFLE", "Every active player is suddenly moved to a new random position."},
	{chaosEffectNoCrates, "NO CRATES", "Normal crate spawning is disabled until the next chaos event."},
	{chaosEffectInfiniteAmmo, "INFINITE AMMO", "Weapons cannot run out of ammunition until the next chaos event."},
	{chaosEffectIceFloor, "ICE FLOOR", "Every platform becomes slippery until the next chaos event."},
	{chaosEffectWindLeft, "WIND LEFT", "A strong arena-wide wind continuously pushes everybody to the left."},
	{chaosEffectWindRight, "WIND RIGHT", "A strong arena-wide wind continuously pushes everybody to the right."},
	{chaosEffectUpdraft, "UPDRAFT", "A constant upward current keeps lifting active players."},
	{chaosEffectEarthquake, "EARTHQUAKE", "Frequent random jolts shake and throw players in different directions."},
	{chaosEffectSkyLaunch, "SKY LAUNCH", "Periodic blasts launch every active player high into the air."},
	{chaosEffectPistolsOnly, "PISTOLS FOR ALL", "Everybody instantly receives a random pistol."},
	{chaosEffectSMGsOnly, "SMGS FOR ALL", "Everybody instantly receives a random SMG."},
	{chaosEffectRiflesOnly, "RIFLES FOR ALL", "Everybody instantly receives a random rifle."},
	{chaosEffectLMGsOnly, "LMGS FOR ALL", "Everybody instantly receives a random light machine gun."},
	{chaosEffectSnipersOnly, "SNIPERS FOR ALL", "Everybody instantly receives a random sniper rifle."},
	{chaosEffectLaunchersOnly, "LAUNCHERS FOR ALL", "Everybody instantly receives a random launcher."},
	{chaosEffectRandomThrowables, "RANDOM THROWABLES", "Every player gets a random grenade type and a fresh supply."},
	{chaosEffectGrenadeRefill, "GRENADE REFILL", "Everybody's grenade stock is refilled immediately."},
	{chaosEffectExtraLife, "EXTRA LIFE FOR ALL", "Every active player gains one additional life where the mode allows it."},
	{chaosEffectShieldEveryone, "SHIELDS FOR ALL", "Every active player receives a temporary shield."},
	{chaosEffectFireFloor, "FIRE FLOOR", "Burning patches appear across the arena platforms and keep returning."},
	{chaosEffectWeaponRoulette, "WEAPON ROULETTE", "Everyone repeatedly receives new random weapons during the event."},
	{chaosEffectTeleportStorm, "TELEPORT STORM", "Players are repeatedly shuffled to new positions during the event."},
	{chaosEffectZeroGravity, "ZERO GRAVITY", "Gravity becomes extremely weak until the next chaos event."},
}

var (
	customChaosSetupRect         = Rect{X: -1240, Y: 458, W: 290, H: 40}
	customChaosToggleRect        = Rect{X: -1760, Y: 138, W: 205, H: 42}
	customChaosIntervalMinusRect = Rect{X: -1535, Y: 138, W: 45, H: 42}
	customChaosIntervalPlusRect  = Rect{X: -1335, Y: 138, W: 45, H: 42}
	customChaosSelectAllRect     = Rect{X: -1280, Y: 138, W: 150, H: 42}
	customChaosClearRect         = Rect{X: -1115, Y: 138, W: 150, H: 42}
	customChaosBackRect          = Rect{X: -1195, Y: 548, W: 245, H: 36}
)

func customChaosEffectRect(index int) Rect {
	// Five compact columns keep the enlarged event pool readable without
	// requiring paging or pushing the bottom controls off-screen.
	col := index % 5
	row := index / 5
	return Rect{X: -1760 + float64(col)*164, Y: 190 + float64(row)*31, W: 154, H: 28}
}

func (g *Game) initCustomChaosDefaults() {
	if g == nil {
		return
	}
	if g.customChaosInterval <= 0 {
		g.customChaosInterval = 30
	}
	for _, spec := range chaosEffectSpecs {
		g.customChaosSelected[spec.ID] = true
	}
}

func (g *Game) customChaosHitAt(x, y float64) int {
	if g == nil || !g.customChaosOpen {
		return menuHitNone
	}
	if customChaosToggleRect.Contains(x, y) {
		return menuHitChaosToggle
	}
	if customChaosIntervalMinusRect.Contains(x, y) {
		return menuHitChaosIntervalMinus
	}
	if customChaosIntervalPlusRect.Contains(x, y) {
		return menuHitChaosIntervalPlus
	}
	if customChaosSelectAllRect.Contains(x, y) {
		return menuHitChaosSelectAll
	}
	if customChaosClearRect.Contains(x, y) {
		return menuHitChaosClear
	}
	for i, spec := range chaosEffectSpecs {
		if customChaosEffectRect(i).Contains(x, y) {
			return menuHitChaosEffectBase + spec.ID
		}
	}
	if customChaosBackRect.Contains(x, y) {
		return menuHitChaosBack
	}
	return menuHitCustomModifiersBlock
}

func (g *Game) activateCustomChaosHit(hit int) bool {
	if g == nil {
		return false
	}
	switch hit {
	case menuHitChaosSetup:
		g.customChaosOpen = true
		return true
	case menuHitChaosBack:
		g.customChaosOpen = false
		return true
	case menuHitChaosToggle:
		g.customChaosEnabled = !g.customChaosEnabled
		return true
	case menuHitChaosIntervalMinus:
		g.customChaosInterval -= 5
		if g.customChaosInterval < 10 {
			g.customChaosInterval = 10
		}
		return true
	case menuHitChaosIntervalPlus:
		g.customChaosInterval += 5
		if g.customChaosInterval > 90 {
			g.customChaosInterval = 90
		}
		return true
	case menuHitChaosSelectAll:
		for _, spec := range chaosEffectSpecs {
			g.customChaosSelected[spec.ID] = true
		}
		return true
	case menuHitChaosClear:
		for _, spec := range chaosEffectSpecs {
			g.customChaosSelected[spec.ID] = false
		}
		return true
	}
	if hit > menuHitChaosEffectBase && hit < menuHitChaosEffectBase+chaosEffectCount {
		id := hit - menuHitChaosEffectBase
		g.customChaosSelected[id] = !g.customChaosSelected[id]
		return true
	}
	return false
}

func (g *Game) chaosSelectedCount() int {
	if g == nil {
		return 0
	}
	n := 0
	for _, spec := range chaosEffectSpecs {
		if g.customChaosSelected[spec.ID] {
			n++
		}
	}
	return n
}

func chaosEffectLabel(id int) string {
	for _, spec := range chaosEffectSpecs {
		if spec.ID == id {
			return spec.Label
		}
	}
	return "CHAOS"
}

func chaosEffectHint(id int) string {
	for _, spec := range chaosEffectSpecs {
		if spec.ID == id {
			return spec.Hint
		}
	}
	return "Choose which events may be drawn by Chaos Mode."
}

func drawChaosEffectButton(screen *ebiten.Image, r Rect, label string, selected bool) {
	fill := color.NRGBA{R: 0x2e, G: 0x31, B: 0x36, A: 0xff}
	if selected {
		fill = color.NRGBA{R: 0xff, G: 0x66, B: 0x00, A: 0xff}
	}
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, color.NRGBA{R: 0x55, G: 0x58, B: 0x60, A: 0xff})
	ebitenutil.DrawRect(screen, r.X+2, r.Y+2, r.W-4, r.H-4, fill)
	drawSourceMenuText(screen, label, menuFontCondensedExtraBold, 9.8, color.White, r.X+7, r.Y+7)
}

func (g *Game) drawCustomChaosOverlay(screen *ebiten.Image) {
	if g == nil || !g.customChaosOpen {
		return
	}
	x := g.customMenuX - 1790
	// Chaos is a real full-screen setup page, not a translucent/popup panel.
	// Fill the whole stage first so no LOCAL/ONLINE header or controls can show
	// through around the old 850x516 panel bounds.
	screen.Fill(color.NRGBA{R: 0x11, G: 0x14, B: 0x19, A: 0xff})
	drawSourceMenuText(screen, "CHAOS MODE SETUP", menuFontCondensedExtraBold, 29, color.White, x+20, 22)
	drawSourceMenuText(screen, "Choose the event pool manually. Chaos will only draw from the effects selected below.", menuFontCondensed, 13.5, color.NRGBA{R: 0xc5, G: 0xca, B: 0xd1, A: 0xff}, x+20, 58)

	toggle := customChaosToggleRect
	toggle.X += g.customMenuX
	drawModifierButton(screen, toggle, map[bool]string{false: "CHAOS MODE: OFF", true: "CHAOS MODE: ON"}[g.customChaosEnabled], g.customChaosEnabled)

	minus := customChaosIntervalMinusRect
	minus.X += g.customMenuX
	plus := customChaosIntervalPlusRect
	plus.X += g.customMenuX
	drawModifierButton(screen, minus, "-", false)
	drawModifierButton(screen, plus, "+", false)
	drawSourceMenuText(screen, fmt.Sprintf("CHANGE EVERY %ds", g.customChaosInterval), menuFontCondensedExtraBold, 14, color.White, g.customMenuX-1477, 150)

	all := customChaosSelectAllRect
	all.X += g.customMenuX
	clear := customChaosClearRect
	clear.X += g.customMenuX
	drawModifierButton(screen, all, "SELECT ALL", false)
	drawModifierButton(screen, clear, "CLEAR", false)

	mx, my := ebiten.CursorPosition()
	hover := 0
	for i, spec := range chaosEffectSpecs {
		r := customChaosEffectRect(i)
		r.X += g.customMenuX
		drawChaosEffectButton(screen, r, spec.Label, g.customChaosSelected[spec.ID])
		if r.Contains(float64(mx), float64(my)) {
			hover = spec.ID
		}
	}

	hint := "Hover an event for details. Selected events form the Chaos pool."
	if hover != 0 {
		hint = chaosEffectHint(hover)
	}
	drawSourceMenuText(screen, fmt.Sprintf("POOL: %d/%d", g.chaosSelectedCount(), len(chaosEffectSpecs)), menuFontCondensedExtraBold, 13.5, color.NRGBA{R: 0xff, G: 0xa0, B: 0x24, A: 0xff}, x+20, 482)
	drawSourceMenuText(screen, hint, menuFontCondensed, 13, color.NRGBA{R: 0xc5, G: 0xca, B: 0xd1, A: 0xff}, x+125, 482)

	back := customChaosBackRect
	back.X += g.customMenuX
	drawModifierButton(screen, back, "BACK TO MODIFIERS", false)
}

func (g *Game) chaosIntervalFrames() int {
	seconds := g.customChaosInterval
	if seconds < 10 {
		seconds = 10
	}
	if seconds > 90 {
		seconds = 90
	}
	return seconds * 60
}

func (g *Game) resetChaosRuntime() {
	if g == nil {
		return
	}
	g.chaosCurrentEffect = chaosEffectNone
	g.chaosLastEffect = chaosEffectNone
	g.chaosFrame = 0
	g.chaosHazardFrame = 0
	g.chaosBannerFrame = 0
}

func (g *Game) pickNextChaosEffect() int {
	if g == nil {
		return chaosEffectNone
	}
	pool := make([]int, 0, len(chaosEffectSpecs))
	for _, spec := range chaosEffectSpecs {
		if g.customChaosSelected[spec.ID] {
			pool = append(pool, spec.ID)
		}
	}
	if len(pool) == 0 {
		return chaosEffectNone
	}
	if len(pool) > 1 && g.chaosLastEffect != chaosEffectNone {
		filtered := pool[:0]
		for _, id := range pool {
			if id != g.chaosLastEffect {
				filtered = append(filtered, id)
			}
		}
		pool = filtered
	}
	return pool[rand.Intn(len(pool))]
}

func (g *Game) updateChaosMode() {
	if g == nil || g.campaignMode || !g.customMatchActive || !g.customChaosEnabled || g.GameWin || g.teamGameWin || g.paused {
		return
	}
	if g.chaosCurrentEffect == chaosEffectNone {
		g.activateChaosEffect(g.pickNextChaosEffect())
		return
	}
	g.chaosFrame++
	g.chaosHazardFrame++
	if g.chaosBannerFrame > 0 {
		g.chaosBannerFrame--
	}
	if g.chaosFrame >= g.chaosIntervalFrames() {
		g.activateChaosEffect(g.pickNextChaosEffect())
		return
	}
	g.updateActiveChaosHazard()
}

func (g *Game) activateChaosEffect(id int) {
	if g == nil {
		return
	}
	g.chaosLastEffect = g.chaosCurrentEffect
	g.chaosCurrentEffect = id
	g.chaosFrame = 0
	g.chaosHazardFrame = 0
	g.chaosBannerFrame = 150
	if id == chaosEffectNone {
		return
	}

	duration := g.chaosIntervalFrames() + 90
	switch id {
	case chaosEffectOneShotgunAll:
		g.chaosEquipAllSame(g.chaosShotgunPool())
	case chaosEffectRandomShotguns:
		g.chaosEquipEach(g.chaosShotgunPool())
	case chaosEffectOneRandomGunAll:
		g.chaosEquipAllSame(developedCrateWeapons())
	case chaosEffectRandomGuns:
		g.chaosEquipEach(developedCrateWeapons())
	case chaosEffectWeaponShuffle:
		g.chaosShuffleWeapons()
	case chaosEffectDynamiteRain:
		g.spawnChaosFallingThrowable(ThrowableDynamite)
	case chaosEffectGrenadeRain:
		g.spawnChaosFallingThrowable(ThrowableMk2 + rand.Intn(4))
	case chaosEffectFireRain:
		g.spawnChaosFallingThrowable(ThrowableMolotov)
	case chaosEffectMixedExplosiveRain:
		g.spawnChaosFallingThrowable(rand.Intn(throwableTypeCount))
	case chaosEffectShockPulse:
		g.fireChaosShockPulse()
	case chaosEffectMiniEveryone:
		for _, p := range g.players {
			if p != nil && p.Active && !p.IsDouble {
				p.MiniTime = duration
			}
		}
	case chaosEffectInvisibleEveryone:
		for _, p := range g.players {
			if p != nil && p.Active && !p.IsDouble {
				p.InvisibleTime = duration
			}
		}
	case chaosEffectSpeedBoost:
		for _, p := range g.players {
			if p != nil && p.Active && !p.IsDouble {
				p.SpeedTime = duration
			}
		}
	case chaosEffectCrateStorm:
		g.spawnChaosCrates(4)
	case chaosEffectPowerupStorm:
		g.spawnChaosPowerups(3)
	case chaosEffectTeleportShuffle:
		g.chaosTeleportPlayers()
	case chaosEffectPistolsOnly:
		g.chaosEquipEach(g.chaosWeaponCategoryPool(DevelopedCategoryPistol))
	case chaosEffectSMGsOnly:
		g.chaosEquipEach(g.chaosWeaponCategoryPool(DevelopedCategorySMG))
	case chaosEffectRiflesOnly:
		g.chaosEquipEach(g.chaosWeaponCategoryPool(DevelopedCategoryRifle))
	case chaosEffectLMGsOnly:
		g.chaosEquipEach(g.chaosWeaponCategoryPool(DevelopedCategoryLMG))
	case chaosEffectSnipersOnly:
		g.chaosEquipEach(g.chaosWeaponCategoryPool(DevelopedCategorySniper))
	case chaosEffectLaunchersOnly:
		g.chaosEquipEach(g.chaosWeaponCategoryPool(DevelopedCategoryLauncher))
	case chaosEffectRandomThrowables:
		for _, p := range g.players {
			if p != nil && p.Active && !p.IsDouble {
				p.ThrowableType = rand.Intn(throwableTypeCount)
				if p.Grenades < 12 {
					p.Grenades = 12
				}
			}
		}
	case chaosEffectGrenadeRefill:
		for _, p := range g.players {
			if p != nil && p.Active && !p.IsDouble {
				p.Grenades = 30
			}
		}
	case chaosEffectExtraLife:
		for _, p := range g.players {
			if p != nil && p.Active && !p.IsDouble && p.GameMode != SourceGameModeGunGame && p.Lives < 99999 {
				p.Lives++
			}
		}
	case chaosEffectShieldEveryone:
		for _, p := range g.players {
			if p != nil && p.Active && !p.IsDouble {
				p.ShieldTime = duration
			}
		}
	case chaosEffectFireFloor:
		g.spawnChaosFireFloor(16)
	case chaosEffectWeaponRoulette:
		g.chaosEquipEach(developedCrateWeapons())
	case chaosEffectTeleportStorm:
		g.chaosTeleportPlayers()
	case chaosEffectEarthquake:
		g.fireChaosEarthquake()
	case chaosEffectSkyLaunch:
		g.fireChaosSkyLaunch()
	}
	g.syncRuntimePlayerRuleModifiers()
}

func (g *Game) updateActiveChaosHazard() {
	if g == nil {
		return
	}
	switch g.chaosCurrentEffect {
	case chaosEffectDynamiteRain:
		if g.chaosHazardFrame >= 45 {
			g.chaosHazardFrame = 0
			g.spawnChaosFallingThrowable(ThrowableDynamite)
		}
	case chaosEffectGrenadeRain:
		if g.chaosHazardFrame >= 50 {
			g.chaosHazardFrame = 0
			g.spawnChaosFallingThrowable(ThrowableMk2 + rand.Intn(4))
		}
	case chaosEffectFireRain:
		if g.chaosHazardFrame >= 42 {
			g.chaosHazardFrame = 0
			g.spawnChaosFallingThrowable(ThrowableMolotov)
		}
	case chaosEffectMixedExplosiveRain:
		if g.chaosHazardFrame >= 34 {
			g.chaosHazardFrame = 0
			g.spawnChaosFallingThrowable(rand.Intn(throwableTypeCount))
		}
	case chaosEffectShockPulse:
		if g.chaosHazardFrame >= 240 {
			g.chaosHazardFrame = 0
			g.fireChaosShockPulse()
		}
	case chaosEffectCrateStorm:
		if g.chaosHazardFrame >= 100 {
			g.chaosHazardFrame = 0
			g.spawnChaosCrates(2)
		}
	case chaosEffectPowerupStorm:
		if g.chaosHazardFrame >= 150 {
			g.chaosHazardFrame = 0
			g.spawnChaosPowerups(2)
		}
	case chaosEffectWindLeft:
		g.applyChaosWind(-0.11)
	case chaosEffectWindRight:
		g.applyChaosWind(0.11)
	case chaosEffectUpdraft:
		for _, p := range g.players {
			if p != nil && p.Active && !p.IsDouble {
				p.VY -= 0.16
				if p.VY < -9 {
					p.VY = -9
				}
			}
		}
	case chaosEffectEarthquake:
		if g.chaosHazardFrame >= 36 {
			g.chaosHazardFrame = 0
			g.fireChaosEarthquake()
		}
	case chaosEffectSkyLaunch:
		if g.chaosHazardFrame >= 150 {
			g.chaosHazardFrame = 0
			g.fireChaosSkyLaunch()
		}
	case chaosEffectFireFloor:
		if g.chaosHazardFrame >= 240 {
			g.chaosHazardFrame = 0
			g.spawnChaosFireFloor(10)
		}
	case chaosEffectWeaponRoulette:
		if g.chaosHazardFrame >= 180 {
			g.chaosHazardFrame = 0
			g.chaosEquipEach(developedCrateWeapons())
		}
	case chaosEffectTeleportStorm:
		if g.chaosHazardFrame >= 180 {
			g.chaosHazardFrame = 0
			g.chaosTeleportPlayers()
		}
	}
}

func chaosEffectModifier(id int) int {
	switch id {
	case chaosEffectLowGravity:
		return customModifierLowGravity
	case chaosEffectHighGravity:
		return customModifierHighGravity
	case chaosEffectDoubleKnockback:
		return customModifierDoubleKnockback
	case chaosEffectSuperJump:
		return customModifierSuperJump
	case chaosEffectFastMovement:
		return customModifierFastMovement
	case chaosEffectJetpacks:
		return customModifierJetpacks
	case chaosEffectCrateStorm:
		return customModifierCratesEverywhere
	case chaosEffectNoCrates:
		return customModifierNoCrates
	case chaosEffectInfiniteAmmo:
		return customModifierInfiniteAmmo
	default:
		return 0
	}
}

func (g *Game) chaosModifierActive(id int) bool {
	return g != nil && g.customChaosEnabled && g.chaosCurrentEffect != chaosEffectNone && chaosEffectModifier(g.chaosCurrentEffect) == id
}

func (g *Game) syncRuntimePlayerRuleModifiers() {
	if g == nil {
		return
	}
	for _, p := range g.players {
		if p == nil || p.IsDouble {
			continue
		}
		p.RuleMoveMultiplier = 1
		p.RuleJumpMultiplier = 1
		p.RuleInfiniteJetpack = false
		if g.customModifierActive(customModifierFastMovement) {
			p.RuleMoveMultiplier = 1.6
		}
		if g.customModifierActive(customModifierSuperJump) {
			p.RuleJumpMultiplier = 1.45
		}
		if g.customModifierActive(customModifierJetpacks) {
			p.RuleInfiniteJetpack = true
		}
	}
}

func (g *Game) chaosWeaponCategoryPool(category DevelopedWeaponCategory) []int {
	pool := organizedGunLibraryNumbers(category)
	if len(pool) == 0 {
		return developedCrateWeapons()
	}
	return pool
}

func (g *Game) chaosShotgunPool() []int {
	pool := make([]int, 0, 32)
	for _, n := range developedCrateWeapons() {
		if def, ok := WeaponByNumber(n); ok && def.Shotgun > 0 {
			pool = append(pool, n)
		}
	}
	if len(pool) == 0 {
		pool = []int{13, 15, 18}
	}
	return pool
}

func (g *Game) chaosEquipAllSame(pool []int) {
	if len(pool) == 0 {
		return
	}
	weapon := pool[rand.Intn(len(pool))]
	for _, p := range g.players {
		if p != nil && p.Active && !p.IsDouble {
			p.DefaultWeapon = weapon
			p.EquipWeapon(weapon)
		}
	}
}

func (g *Game) chaosEquipEach(pool []int) {
	if len(pool) == 0 {
		return
	}
	for _, p := range g.players {
		if p != nil && p.Active && !p.IsDouble {
			weapon := pool[rand.Intn(len(pool))]
			p.DefaultWeapon = weapon
			p.EquipWeapon(weapon)
		}
	}
}

func (g *Game) chaosShuffleWeapons() {
	players := make([]*Player, 0, 4)
	weapons := make([]int, 0, 4)
	for _, p := range g.players {
		if p != nil && p.Active && !p.IsDouble {
			players = append(players, p)
			weapons = append(weapons, p.Weapon.Def.Number)
		}
	}
	if len(players) < 2 {
		g.chaosEquipEach(developedCrateWeapons())
		return
	}
	for i, p := range players {
		weapon := weapons[(i+len(weapons)-1)%len(weapons)]
		if weapon <= 0 {
			weapon = developedCrateWeapons()[rand.Intn(len(developedCrateWeapons()))]
		}
		p.DefaultWeapon = weapon
		p.EquipWeapon(weapon)
	}
}

func (g *Game) spawnChaosFallingThrowable(kind int) {
	if g == nil {
		return
	}
	width := int(math.Max(1, g.arena.CrateMaxX-g.arena.CrateMinX))
	x := g.arena.CrateMinX + float64(rand.Intn(width))
	rot := 6 + rand.Float64()*10
	if rand.Intn(2) == 0 {
		rot = -rot
	}
	g.grenades = append(g.grenades, Grenade{
		X: x, Y: -120, VX: rand.Float64()*2 - 1, VY: 1.5 + rand.Float64()*2,
		Rotation: 0, RotSpeed: rot, OwnerID: 0, Type: normalizedThrowableType(kind),
	})
}

func (g *Game) applyChaosWind(force float64) {
	if g == nil {
		return
	}
	for _, p := range g.players {
		if p == nil || !p.Active || p.IsDouble {
			continue
		}
		p.VX += force
		if p.VX > 12 {
			p.VX = 12
		} else if p.VX < -12 {
			p.VX = -12
		}
	}
}

func (g *Game) fireChaosEarthquake() {
	if g == nil {
		return
	}
	for _, p := range g.players {
		if p == nil || !p.Active || p.IsDouble {
			continue
		}
		p.VX += rand.Float64()*9 - 4.5
		p.VY -= 2 + rand.Float64()*5
		p.HitNumber++
		p.HitTimer = 0
		p.LastHitBy = 0
	}
}

func (g *Game) fireChaosSkyLaunch() {
	if g == nil {
		return
	}
	for _, p := range g.players {
		if p == nil || !p.Active || p.IsDouble {
			continue
		}
		p.VY = -(13 + rand.Float64()*6)
		p.VX += rand.Float64()*6 - 3
		p.HitNumber++
		p.HitTimer = 0
		p.LastHitBy = 0
	}
}

func (g *Game) spawnChaosFireFloor(count int) {
	if g == nil || count <= 0 || len(g.arena.Platforms) == 0 {
		return
	}
	for i := 0; i < count; i++ {
		r := g.arena.Platforms[rand.Intn(len(g.arena.Platforms))]
		if r.W <= 8 {
			continue
		}
		x := r.X + 4 + rand.Float64()*math.Max(1, r.W-8)
		g.firePatches = append(g.firePatches, FirePatch{X: x, Y: r.Y, OwnerID: 0, Life: 320})
	}
}

func (g *Game) fireChaosShockPulse() {
	if g == nil {
		return
	}
	centerX := (g.arena.CrateMinX + g.arena.CrateMaxX) / 2
	centerY := 285.0
	for _, p := range g.players {
		if p == nil || !p.Active || p.IsDouble {
			continue
		}
		direction := 1.0
		if p.X < centerX {
			direction = -1
		}
		strength := 10 + math.Min(5, math.Abs(p.X-centerX)/80)
		p.VX += direction * strength
		p.VY -= 9
		p.HitNumber++
		p.HitTimer = 0
		p.LastHitBy = 0
	}
	g.explosions = append(g.explosions, Explosion{X: centerX, Y: centerY, Life: 18})
	g.blastFX = append(g.blastFX, spawnGrenadeBlastEffects(centerX, centerY)...)
	g.playRandomExplosionSound()
}

func (g *Game) spawnChaosCrates(count int) {
	for i := 0; i < count; i++ {
		c := spawnCrate(g.arena)
		c.Serial = g.nextPickupSerial
		g.nextPickupSerial++
		c.Y -= float64(i * 55)
		g.crates = append(g.crates, c)
	}
}

func (g *Game) spawnChaosPowerups(count int) {
	for i := 0; i < count; i++ {
		pu := spawnPowerup(g.arena, g.hasActiveDouble() || g.campaignMode)
		if pu.Dead {
			continue
		}
		pu.Serial = g.nextPickupSerial
		g.nextPickupSerial++
		g.powerups = append(g.powerups, pu)
	}
}

func (g *Game) chaosTeleportPlayers() {
	if g == nil {
		return
	}
	width := int(math.Max(1, g.arena.SpawnMaxX-g.arena.SpawnMinX))
	for _, p := range g.players {
		if p == nil || !p.Active || p.IsDouble {
			continue
		}
		p.X = g.arena.SpawnMinX + float64(rand.Intn(width))
		p.Y = 55 + float64(rand.Intn(120))
		p.VX, p.VY = 0, 0
		p.FreePass = true
	}
}

func (g *Game) drawChaosHUD(screen *ebiten.Image) {
	if g == nil || !g.customChaosEnabled || g.chaosCurrentEffect == chaosEffectNone || g.GameWin || g.teamGameWin {
		return
	}
	remaining := g.chaosIntervalFrames() - g.chaosFrame
	if remaining < 0 {
		remaining = 0
	}
	seconds := (remaining + 59) / 60
	w := 390.0
	x := (900 - w) / 2
	fill := color.NRGBA{R: 0x18, G: 0x1b, B: 0x21, A: 0xe8}
	accent := color.NRGBA{R: 0xff, G: 0x66, B: 0x00, A: 0xff}
	if g.chaosBannerFrame > 0 {
		w = 560
		x = (900 - w) / 2
		fill = color.NRGBA{R: 0x35, G: 0x12, B: 0x08, A: 0xf0}
	}
	ebitenutil.DrawRect(screen, x, 112, w, 38, fill)
	ebitenutil.DrawRect(screen, x, 112, w, 3, accent)
	text := fmt.Sprintf("CHAOS: %s   %ds", chaosEffectLabel(g.chaosCurrentEffect), seconds)
	drawSourceMenuText(screen, text, menuFontCondensedExtraBold, 15, color.White, x+14, 121)
}
