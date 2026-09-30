package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	players          []*Player
	maps             map[int]Map
	arena            Map
	assets           *Assets
	bullets          []Bullet
	instagibTrails   []InstagibTrailEffect
	playerTrails     []PlayerTrailEffect
	shells           []Shell
	flashes          []MuzzleFlash
	grenades         []Grenade
	firePatches      []FirePatch
	portals          []Portal
	dynamiteFX       []DynamiteEffect
	blastFX          []BlastEffect
	jetThrustFX      []JetThrustEffect
	dropPackFX       []DropPackEffect
	explosions       []Explosion
	killfeeds        []KillFeedEntry
	crates           []Crate
	crateTimer       int
	crateWeapons     []int
	powerups         []Powerup
	powerupNameFX    []PowerupNameEffect
	lifeBlingFX      []LifeBlingEffect
	powerupTimer     int
	seenDeaths       map[int]int
	nextEntityID     int
	nextPickupSerial int
	mapFXFrame       int
	cameraX          float64
	cameraY          float64

	// Root variables from the source frame/menu flow.
	GameMode   int
	TotalLives int
	TeamGame   bool
	CrateON    bool
	PowerON    bool
	GameWin    bool

	showCollisions bool

	// Source menu flow: main menu -> custom GAME SETUP -> MAP SELECTION ->
	// PLAYER SETUP -> gameplay.
	screen                         gameScreen
	customPage                     int
	customMenuX                    float64
	customMode                     int
	customMap                      int
	customMapSetGM2                bool
	customLives                    int
	customSeriesRounds             int
	customSeriesRound              int
	customSeriesRecordedRound      int
	customSeriesStartingNext       bool
	customSeriesStats              [4]customSeriesPlayerStats
	customSeriesMaps               [customSeriesMaxRounds]int
	customSeriesPlaylistConfigured bool
	customPlaylistSelectedRound    int
	customPlaylistScroll           int
	customPlaylistGM2              bool
	customMapThumbCache            map[int]*ebiten.Image
	customMapThumbPreloadCursor    int
	customPlayers                  [4]CustomPlayerConfig
	customCardY                    [4]float64
	customEditor                   [4]int
	customColorMask                [4]float64
	customNameFocus                int
	customLivesFocus               bool
	customWarning                  int
	customWarningFrame             int
	customRulesOpen                bool
	customRuleEnabled              map[int]bool
	customRandomRules              bool
	customRandomCount              int
	customPerRoundRules            bool
	customRuleEditRound            int
	customRoundRuleEnabled         [customSeriesMaxRounds][customModifierJetpacks + 1]bool
	customRoundRandomRules         [customSeriesMaxRounds]bool
	customRoundRandomCount         [customSeriesMaxRounds]int
	customChaosOpen                bool
	customChaosEnabled             bool
	customChaosInterval            int
	customChaosSelected            [chaosEffectCount]bool
	chaosCurrentEffect             int
	chaosLastEffect                int
	chaosFrame                     int
	chaosHazardFrame               int
	chaosBannerFrame               int
	matchModifiers                 map[int]bool
	customMatchActive              bool
	menuPressed                    int
	menuDebug                      bool

	// Root interaction state reconstructed from frame10/menu ActionScript.
	paused            bool
	pausePressed      int
	fadeActive        bool
	fadeFrame         int
	fadeTarget        gameScreen
	fadePurpose       fadePurpose
	matchWinCountdown int
	soloWinFrame      int
	teamGameWin       bool
	teamWinner        int
	winnerPlayerID    int
	winnerAnimFrame   int
	teamWinAnimFrame  int
	campaignLoseFrame int
	zombieWaveFrame   int

	// Symbol1480.lastlife keeps an independent 270-frame playhead per HUD card.
	// Frame indexes here are zero-based XFL frames; stop points are 0, 89, 179, 269.
	hudLastLifeFrame   [4]int
	hudLastLifePlaying [4]bool
	hudLastLevel       [4]int

	// Campaign SharedObject/menu state. Values mirror arenagamedata3:
	// 0=locked, 1=available, 2=completed.
	campaignMode         bool
	campaignLevel        int
	campaignShowUnlock   int
	campaignSliderX      float64
	campaignSliderVX     float64
	campaignPhase        int
	campaignDetailAlpha  float64
	campaignLevels       [10]int
	campaignGuns         [57]bool
	campaignPressed      int
	campaignPlayers      [2]CustomPlayerConfig
	campaignCardY        [2]float64
	campaignEditor       [2]int
	campaignColorMask    [2]float64
	campaignNameFocus    int
	campaignDynamiteTime int

	// GM2 arena2gamedata progress is independent from the GM1 campaign above.
	// GM2 has 16 missions and 7 timed challenges.
	gm2CampaignLevels   [16]int
	gm2Challenges       [7]GM2ChallengeProgress
	campaignSetGM2      bool
	extraCampaignLevels [extraCampaignMissionCount]int
	campaignSetExtra    bool
	extraCampaignPage   int
	extraHazardFrame    int

	// arenagamedata2 defaults used by the source Options screen.
	musicOn                 bool
	soundOn                 bool
	musicVolume             float64
	soundVolume             float64
	loudnessEQ              bool
	allowCheats             bool
	quality                 int
	optionsPressed          int
	optionsDraggingSlider   int
	audio                   *sourceAudioEngine
	audioStarted            bool
	controlConfigs          [4]Controls
	optionsKeyCapturePlayer int
	optionsKeyCaptureIndex  int

	gunLibrarySelected          int
	gunLibraryPressed           int
	gunLibraryGM2               bool
	gunLibraryPage              int
	gunLibraryCategory          int
	gunLibraryThrowableSelected int

	gototest          bool
	testGunNumber     int
	testGunDisabled   bool
	testGunRespawn    int
	testGunFrame      int
	testThrowableMode bool

	persistent             bool
	windowedWidth          int
	windowedHeight         int
	windowedX              int
	windowedY              int
	windowPositionSaved    bool
	windowPersistCountdown int

	multiplayerAddress        string
	multiplayerFocus          bool
	multiplayerSessionName    string
	multiplayerNameFocus      bool
	multiplayerMessage        string
	customOnlineMode          bool
	customNetworkOpen         bool
	customHostSetupOpen       bool
	customHostScope           int
	customHostAccess          int
	multiplayerServers        []multiplayerServerInfo
	multiplayerServerScroll   int
	multiplayerSelectedServer int
	multiplayerDiscoveryCh    chan masterServerDiscoveryResult
	multiplayerDiscoveryDone  bool
	multiplayerCatalogState   publicCatalogState
	serverAdvertiser          *masterServerAdvertiser
	publicDirectory           *publicRoomAdvertiser
	localServerAdvertiser     *localServerAdvertiser
	lanDiscoveryResponder     *lanDiscoveryResponder
	globalDHTAdvertiser       *globalDHTAdvertiser
	p2pHost                   *p2pQUICHost
	cloudflareTunnel          *cloudflareQuickTunnel
	natPortMapper             *natPortMapper
	onlineSessionVisibility   int // OPEN vs INVITE CODE
	onlineSessionScope        int // LAN vs INTERNET+LAN
	onlineSessionKey          string
	onlineMatchSetup          bool
	onlinePlayerCount         int
	onlinePlayerSlots         [4]bool
	onlineLobbyCustomize      bool
	onlineRulesOpen           bool
	onlineBlocklistOpen       bool
	onlinePerkBlocklistOpen   bool
	onlineBlocklistPage       int
	onlineVisualPicker        int
	onlineVisualPage          int
	onlineLastSentConfig      CustomPlayerConfig
	onlineHasSentConfig       bool
	onlineMapRotation         int
	onlineWeaponMode          int
	onlineOnlyWeapon          int
	onlinePerksEnabled        bool
	onlinePowerupsEnabled     bool
	onlineBlockedWeapons      [developedWeaponSlotCount]bool
	onlineBlockedPerks        [maxCustomPerk + 1]bool
	postGameReady             [4]bool
	postGameReadyCountdown    int

	netplay             *netplaySession
	netSFXSeq           uint64
	netSFXPending       []netSFXEvent
	netClientSFXPending []netSFXEvent
	netLastSFXSeq       uint64

	updateRuntime updateRuntimeState
}

func New() *Game {
	arena, err := LoadOriginalMap(1)
	if err != nil {
		arena = OriginalMap1()
	}
	maps := map[int]Map{1: arena}
	p1 := NewPlayer(1, arena)
	p2 := NewPlayer(2, arena)
	p2.AI = true

	// Source setup defaults: frame9 enables crates/powerups, the lives input
	// text defaults to 10, and normal mode is mode frame1.
	const mode = SourceGameModeNormal
	const totalLives = 10
	p1.configureSourceGameMode(mode, totalLives, arena)
	p2.configureSourceGameMode(mode, totalLives, arena)

	g := &Game{
		players:                   []*Player{p1, p2},
		maps:                      maps,
		arena:                     arena,
		assets:                    LoadAssets(),
		crateTimer:                0,
		crateWeapons:              DefaultCrateWeapons(),
		powerupTimer:              0,
		seenDeaths:                map[int]int{1: 0, 2: 0},
		nextEntityID:              3,
		nextPickupSerial:          1,
		GameMode:                  mode,
		TotalLives:                totalLives,
		TeamGame:                  false,
		CrateON:                   true,
		PowerON:                   true,
		screen:                    screenMainMenu,
		customPage:                customPageGame,
		customMenuX:               1800,
		customMode:                SourceGameModeNormal,
		customMap:                 0,
		customLives:               10,
		customSeriesRounds:        3,
		customMapThumbCache:       make(map[int]*ebiten.Image),
		customRuleEnabled:         make(map[int]bool),
		customRandomCount:         1,
		customChaosInterval:       30,
		matchModifiers:            make(map[int]bool),
		campaignPhase:             1,
		musicOn:                   true,
		soundOn:                   true,
		musicVolume:               1,
		soundVolume:               1,
		quality:                   2,
		audio:                     newSourceAudioEngine(),
		multiplayerAddress:        "127.0.0.1:7777",
		multiplayerSessionName:    "My Gun Mayhem Game",
		multiplayerSelectedServer: -1,
		customHostScope:           onlineSessionScopeLAN,
		customHostAccess:          onlineSessionOpen,
		onlineOnlyWeapon:          10,
		onlinePerksEnabled:        true,
		onlinePowerupsEnabled:     true,
	}
	// arenagamedata3 defaults from root frame2: levels 1 and 2 available,
	// levels 3..10 locked. All 57 guns start unlocked except the twenty
	// campaign rewards, which are enabled by returntomenu() after a win.
	g.campaignLevels, g.campaignGuns = defaultCampaignState()
	g.gm2CampaignLevels = defaultGM2CampaignState()
	g.extraCampaignLevels = defaultExtraCampaignState()
	g.initCustomChaosDefaults()
	g.initCustomPlayerSetup()
	g.initCampaignPlayerSetup()
	for i := 0; i < 4; i++ {
		g.controlConfigs[i] = OriginalControls(i + 1)
	}
	p1.Controls = g.controlConfigs[0]
	p2.Controls = g.controlConfigs[1]
	return g
}

func (g *Game) Update() error {
	// Network games must keep simulation/snapshots alive when the host alt-tabs.
	// Local games retain pause-on-focus-loss behavior.
	ebiten.SetRunnableOnUnfocused(g.netplay != nil)
	if g.updateUpdaterRuntime() {
		return ebiten.Termination
	}
	g.trackWindowPersistence()
	if g.netplay != nil && g.netplay.mode == netplayClient {
		if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
			if !ebiten.IsFullscreen() {
				g.captureWindowedState()
			}
			ebiten.SetFullscreen(!ebiten.IsFullscreen())
			_ = g.saveConfig()
		}
		return g.updateNetClient()
	}
	if g.netplay != nil && g.netplay.mode == netplayHost {
		g.syncOnlineHostState()
		defer g.queueNetHostSnapshot()
		if g.returnOnlineHostToLobbyIfAlone() {
			return nil
		}
		g.applyHostRemoteInput()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		if !ebiten.IsFullscreen() {
			g.captureWindowedState()
		}
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
		_ = g.saveConfig()
	}
	if g.screen != screenGameplay {
		return g.updateMenu()
	}
	g.assets.EnsureGameplay()
	if g.updateGameplayInteractionInput() {
		return nil
	}
	if g.allowCheats {
		if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
			g.showCollisions = !g.showCollisions
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
			g.switchMap(1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
			g.switchMap(-1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			g.resetArenaState()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyF4) && len(g.players) > 1 {
			g.players[1].AI = !g.players[1].AI
			g.players[1].clearAIInput()
			g.players[1].AITargetTimer = 40
		}
	}

	g.updateChaosMode()
	g.syncRuntimePlayerRuleModifiers()
	runtimeArena := g.customRuntimeArena()
	for _, p := range g.players {
		speedBefore := p.SpeedTime
		miniForTrail := p.MiniTime > 0
		if g.GameWin {
			// Source player/AI onEnterFrame starts with `if(_root.gamewin){vx=0;vy=0}`
			// and skips the entire input block guarded by `!_root.gamewin`.
			p.VX, p.VY = 0, 0
			p.clearAIInput()
		} else {
			var beforeMove, afterPhysics func()
			if p.AI {
				beforeMove = func() {
					g.prepareCampaignAISpecial(p)
					prepareAITarget(p, g)
				}
				afterPhysics = func() { decideAIController(p, g) }
			}
			p.Update(runtimeArena, beforeMove, afterPhysics)
		}
		if p.HardLanded {
			g.playRandomDropSound()
		}
		// DefineSprite_697 speed block decrements speedtime first and then calls
		// CP2(fx_playertrail, ...) on every even remaining tick. speedBefore > 0
		// avoids the respawn/default speedtime=0 frame, which source decrements
		// to -1 before testing the modulo.
		if speedBefore > 0 && p.SpeedTime >= 0 && p.SpeedTime%2 == 0 && p.Active {
			g.playerTrails = append(g.playerTrails, newPlayerTrailEffect(p, miniForTrail))
		}
		if p.JetThrusted {
			g.jetThrustFX = append(g.jetThrustFX, newJetThrustEffect(
				p.X-22*float64(p.Facing), p.Y-18,
			))
			// The original finite jetpack can retrigger its short SFX on every
			// thrust particle. With the developed infinite-jetpack modifier that
			// would stack a new copy every frame, so play it only when a continuous
			// thrust starts and allow another copy after the player releases jump.
			if p.RuleInfiniteJetpack {
				if !p.RuleJetpackSoundLive {
					g.playSourceSFX("jetpack.wav", false)
				}
				p.RuleJetpackSoundLive = true
			} else {
				g.playSourceSFX("jetpack.wav", false)
			}
		} else if p.RuleInfiniteJetpack {
			p.RuleJetpackSoundLive = false
		}
		if p.JetDropped {
			g.dropPackFX = append(g.dropPackFX, newDropPackEffect(
				p.X-20*float64(p.Facing), p.Y-40, p.Facing,
			))
		}
		if p.DeathSerial != g.seenDeaths[p.ID] {
			g.seenDeaths[p.ID] = p.DeathSerial
			g.playRandomDeathSound()
			if g.campaignMode && g.campaignUsesOriginalGM1Rules() && g.campaignLevel == 4 && !p.AI && !p.IsDouble {
				// DefineSprite_697.gotkilled(): a human death in level 4 destroys
				// Pylon Man's currently spawned helper named `double`.
				g.killCampaignDouble()
			}
			g.appendSourceKillFeed(p)
			g.applyCustomRespawnWeapon(p)
			if p.LastDeathBy > 0 && p.LastDeathBy != p.ID {
				for _, killer := range g.players {
					if killer.ID == p.LastDeathBy {
						killer.Kills++
						// fx_bling score: normal kill=100, cheapshot=200,
						// grenade=50 (greedykill=300 once that source flag is ported).
						score := 100
						if p.LastDeathMod == 2 {
							score = 200
						}
						if p.LastDeathMod == 3 {
							score = 50
						}
						killer.Score += score
						if g.GameMode == SourceGameModeGunGame {
							killer.sourceUpgrade(g.players)
						}
						break
					}
				}
			}
			g.explosions = append(g.explosions, Explosion{X: p.LastDeathX, Y: p.LastDeathY - 20, Life: 18})
		}
	}
	for _, p := range g.players {
		if !g.GameWin {
			ammoBefore := p.Weapon.Bullets
			weaponBefore := p.Weapon.Def.Number
			shotsBefore := p.ShotsFired
			bulletsBefore := len(g.bullets)
			if g.customModifierActive(customModifierExplosivesOnly) {
				p.Grenades = 9999
			} else {
				if g.customModifierActive(customModifierInfiniteAmmo) {
					p.Weapon.Bullets = 9999
					ammoBefore = p.Weapon.Bullets
				}
				updateWeapon(p, &g.bullets, &g.shells, &g.flashes)
				if g.customModifierActive(customModifierNoReload) && p.ShotsFired > shotsBefore && ammoBefore <= 1 {
					p.EquipWeapon(weaponBefore)
				}
				if g.customModifierActive(customModifierDoubleKnockback) {
					for i := bulletsBefore; i < len(g.bullets); i++ {
						g.bullets[i].Firepower *= 2
					}
				}
			}
			if p.Weapon.Bullets < ammoBefore || p.ShotsFired > shotsBefore {
				playSound := true
				if spec, ok := developedWeaponSpec(weaponBefore); ok && spec.Projectile == DevelopedProjectileFlame {
					// The flame loop is built from overlapping short bursts. Starting a
					// new PCM clip every 60 Hz tick would stack six copies and clip.
					playSound = p.ShotsFired%6 == 1
				}
				if playSound {
					g.playSourceGunSound(weaponBefore)
				}
			}
			for _, sound := range p.PendingSounds {
				g.playSourceSFX(sound, false)
			}
			p.PendingSounds = p.PendingSounds[:0]
			// Source creates wep_grenade on key release but does not play an
			// SFX at the throw itself. (drop*.wav is the hard-landing sound.)
			updateGrenadeInput(p, &g.grenades)
		}
		p.advanceLegTimelines(len(g.assets.Leg1Timeline), len(g.assets.Leg2Timeline))
		p.advanceHeadTimeline(g.assets.HeadTimeline)
		p.updateSourceVisualState()
	}
	g.updateHUDLastLife()
	g.updateSourceCamera()
	g.advanceSourceMapFX()

	hitsBefore := 0
	for _, p := range g.players {
		hitsBefore += p.HitNumber
	}
	projectileBlastBefore := len(g.blastFX)
	updateBullets(runtimeArena, g.bullets, g.players, &g.instagibTrails, &g.explosions, &g.blastFX, &g.portals)
	if len(g.blastFX) > projectileBlastBefore {
		g.playRandomExplosionSound()
	}
	hitsAfter := 0
	for _, p := range g.players {
		hitsAfter += p.HitNumber
	}
	for i := hitsBefore; i < hitsAfter; i++ {
		g.playRandomHitSound()
	}
	updateInstagibTrails(g.instagibTrails)
	updatePlayerTrailEffects(g.playerTrails, len(g.assets.Leg1Timeline), len(g.assets.Leg2Timeline))
	updateShells(runtimeArena, g.shells, g.players)
	updateFlashes(g.flashes)
	blastBefore := len(g.blastFX)
	var grenadeVX, grenadeVY []float64
	if g.customModifierActive(customModifierDoubleKnockback) {
		grenadeVX = make([]float64, len(g.players))
		grenadeVY = make([]float64, len(g.players))
		for i, p := range g.players {
			grenadeVX[i] = p.VX
			grenadeVY[i] = p.VY
		}
	}
	updateGrenades(runtimeArena, g.grenades, g.players, &g.explosions, &g.dynamiteFX, &g.blastFX, &g.firePatches)
	updateFirePatches(g.firePatches, g.players)
	updatePortalTravel(g.portals, g.players, g.grenades)
	if grenadeVX != nil {
		for i, p := range g.players {
			p.VX += p.VX - grenadeVX[i]
			p.VY += p.VY - grenadeVY[i]
		}
	}
	if len(g.blastFX) > blastBefore {
		g.playRandomExplosionSound()
	}
	// Root frame10 attaches the campaign-level-5 falling grenade after the
	// ordinary player/grenade updates; it begins moving on the following tick.
	g.updateCampaignDynamiteRain()
	g.updateExtraCampaignHazards()
	updateDynamiteEffects(g.dynamiteFX)
	updateBlastEffects(runtimeArena, g.blastFX)
	updateJetThrustEffects(g.jetThrustFX)
	updateDropPackEffects(runtimeArena, g.dropPackFX)
	g.crateTimer++
	if g.crateTimer >= g.customCrateInterval() && g.GameMode != SourceGameModeGunGame && g.customCratesEnabled() {
		c := spawnCrate(runtimeArena)
		c.Serial = g.nextPickupSerial
		g.nextPickupSerial++
		g.crates = append(g.crates, c)
		g.crateTimer = 0
	}
	cratePickupsBefore := 0
	for _, p := range g.players {
		cratePickupsBefore += p.CratesCollected
	}
	updateCrates(runtimeArena, g.crates, g.players, g.crateWeapons, g.assets.CrateHit)
	cratePickupsAfter := 0
	for _, p := range g.players {
		cratePickupsAfter += p.CratesCollected
	}
	for i := cratePickupsBefore; i < cratePickupsAfter; i++ {
		// DefineSprite_668_crate: getgun(randgun); playsound2("bolt2.wav").
		g.playSourceSFX("bolt2.wav", true)
	}
	g.powerupTimer++
	if g.powerupTimer >= 600 && g.GameMode != SourceGameModeGunGame && g.GameMode != SourceGameModeSurvival && g.PowerON {
		// DefineSprite_734 replaces the Double powerup with shield whenever a
		// helper already exists OR campaignmode is active.
		pu := spawnPowerup(runtimeArena, g.hasActiveDouble() || g.campaignMode)
		pu.Serial = g.nextPickupSerial
		g.nextPickupSerial++
		g.powerups = append(g.powerups, pu)
		g.powerupTimer = 0
	}
	wasPowerupPicked := make([]bool, len(g.powerups))
	for i := range g.powerups {
		wasPowerupPicked[i] = g.powerups[i].PickedUp
	}
	updatePowerups(g.powerups, g.players)
	for i := range g.powerups {
		pu := g.powerups[i]
		if wasPowerupPicked[i] || !pu.PickedUp {
			continue
		}
		// DefineSprite_734_powerup: every pickup, including EXTRA LIFE,
		// uses playsound("emp.wav") and creates fx_powerupname.
		g.playSourceSFX("emp.wav", false)
		g.powerupNameFX = append(g.powerupNameFX, newPowerupNameEffect(pu))
		if pu.Type == 0 && pu.PickedByID > 0 {
			// lifebling() -> fx_bling mod=6 -> source green "1-UP".
			g.lifeBlingFX = append(g.lifeBlingFX, newLifeBlingEffect(pu.PickedByID))
		}
	}
	g.updatePowerupPickupEffects()
	g.updateTestGunPickup()
	g.spawnRequestedDoubles()
	updateExplosions(g.explosions)
	updateKillFeeds(g.killfeeds)
	g.bullets = compactBullets(g.bullets)
	g.instagibTrails = compactInstagibTrails(g.instagibTrails)
	g.playerTrails = compactPlayerTrailEffects(g.playerTrails)
	g.shells = compactShells(g.shells)
	g.flashes = compactFlashes(g.flashes)
	g.grenades = compactGrenades(g.grenades)
	g.firePatches = compactFirePatches(g.firePatches)
	g.dynamiteFX = compactDynamiteEffects(g.dynamiteFX)
	g.blastFX = compactBlastEffects(g.blastFX)
	g.jetThrustFX = compactJetThrustEffects(g.jetThrustFX)
	g.dropPackFX = compactDropPackEffects(g.dropPackFX)
	g.explosions = compactExplosions(g.explosions)
	g.killfeeds = compactKillFeeds(g.killfeeds)
	g.crates = compactCrates(g.crates)
	g.powerups = compactPowerups(g.powerups)
	g.updateMatchInteractions()
	return nil
}

func (g *Game) resetArenaState() {
	regular := g.players[:0]
	for _, p := range g.players {
		if !p.IsDouble {
			regular = append(regular, p)
		}
	}
	g.players = regular
	g.seenDeaths = make(map[int]int, len(g.players))
	maxID := 0
	for _, p := range g.players {
		if p.ID > maxID {
			maxID = p.ID
		}
	}
	g.nextEntityID = maxID + 1
	g.nextPickupSerial = 1
	g.mapFXFrame = 0
	g.cameraX = 0
	g.cameraY = 0
	g.bullets = g.bullets[:0]
	g.instagibTrails = g.instagibTrails[:0]
	g.playerTrails = g.playerTrails[:0]
	g.shells = g.shells[:0]
	g.flashes = g.flashes[:0]
	g.grenades = g.grenades[:0]
	g.firePatches = g.firePatches[:0]
	g.portals = g.portals[:0]
	g.dynamiteFX = g.dynamiteFX[:0]
	g.blastFX = g.blastFX[:0]
	g.jetThrustFX = g.jetThrustFX[:0]
	g.dropPackFX = g.dropPackFX[:0]
	g.explosions = g.explosions[:0]
	g.killfeeds = g.killfeeds[:0]
	g.crates = g.crates[:0]
	g.powerups = g.powerups[:0]
	g.powerupNameFX = g.powerupNameFX[:0]
	g.lifeBlingFX = g.lifeBlingFX[:0]
	// Source frame10 initializes cratetime=200 and poweruptime=0 before
	// onEnterFrame starts. This makes the first crate arrive after ~200 ticks,
	// not ~400, while powerups still use their full 600-tick delay.
	g.crateTimer = 200
	g.powerupTimer = 0
	for _, p := range g.players {
		p.ResetRound(g.arena)
		g.seenDeaths[p.ID] = 0
	}
	g.resetHUDLastLife()
}

func (g *Game) hasActiveDouble() bool {
	for _, p := range g.players {
		if p.IsDouble && p.Active {
			return true
		}
	}
	return false
}

func (g *Game) spawnRequestedDoubles() {
	owners := make([]*Player, 0, 2)
	for _, p := range g.players {
		if !p.IsDouble && p.WantsDouble {
			p.WantsDouble = false
			owners = append(owners, p)
		}
	}
	for _, owner := range owners {
		alreadyActive := false
		for _, p := range g.players {
			if p.IsDouble && p.Active && p.OwnerPlayerID == owner.ID {
				alreadyActive = true
				if !p.PersistentDouble {
					p.DoubleTime = 600
				}
				break
			}
		}
		if alreadyActive {
			continue
		}

		d := NewPlayer(g.nextEntityID, g.arena)
		g.nextEntityID++
		campaignDouble := g.campaignMode && g.campaignUsesOriginalGM1Rules() && g.campaignLevel == 4 && owner.AISpecial == sourceAISpecialDoubleSpawner
		spawnX, spawnY := owner.X, owner.Y
		if campaignDouble && owner.DoubleSpawnPositionSet {
			spawnX, spawnY = owner.DoubleSpawnX, owner.DoubleSpawnY
		}
		owner.DoubleSpawnPositionSet = false
		d.Name = owner.Name + " DOUBLE"
		if campaignDouble {
			// playerAI_double.spawnfriend() creates a normal playerAI named
			// `double`; its display name is the owner's name and its timer is not
			// decremented during campaign level 4.
			d.Name = owner.Name
		}
		d.AI = true
		d.IsDouble = true
		d.PersistentDouble = campaignDouble
		d.DoubleTime = 600
		d.Lives = 1
		if campaignDouble {
			// Regular playerAI constructor, campaign level 4 branch.
			d.CheapTimer = 120
			// spawnfriend() creates a fresh AI instance. Do not inherit any of
			// Pylon Man's target/lock/input state or the pair will move in sync.
			d.AITargetTimer = 0
			d.AITargetValid = false
			d.AITargetKind = aiTargetNone
			d.AITargetSerial = 0
			d.AITargetPlayerID = 0
			d.AILockLeft = 0
			d.AILockRight = 0
			d.AILockUp = 0
			d.AIIdleTime2 = 0
			d.AIPrevX = 0
			d.clearAIInput()
		}
		d.Team = owner.Team
		d.OwnerPlayerID = owner.ID
		d.PlayerColor = owner.PlayerColor
		d.ShirtNumber = owner.ShirtNumber
		d.HatNumber = owner.HatNumber
		d.PerkNumber = 0
		d.DefaultWeapon = owner.DefaultWeapon
		d.FirepowerMulti = owner.FirepowerMulti
		d.DamageMulti = owner.DamageMulti
		d.EquipWeapon(owner.DefaultWeapon)
		// spawnfriend() puts the helper exactly on Pylon Man. The regular
		// playerAI respawn() explicitly skips the random Y=-1000/X spawn branch
		// when _name == "double", so this position must be preserved. Its motion
		// state is still fresh (vx=vy=0), independent from the owner's velocity.
		d.X = spawnX
		d.Y = spawnY
		d.VX = 0
		d.VY = 0
		if campaignDouble {
			// Ordinary playerAI campaign-level-4 initialization sets
			// cheapshottimer=120 after respawn().
			d.CheapTimer = 120
		}
		g.players = append(g.players, d)
		g.seenDeaths[d.ID] = 0
	}
}

func (g *Game) ensureMap(number int) (Map, bool) {
	if m, ok := g.maps[number]; ok {
		return m, true
	}
	var (
		m   Map
		err error
	)
	if isGM2MapID(number) {
		m, err = LoadGM2Map(gm2SourceMapNumber(number))
	} else {
		m, err = LoadOriginalMap(number)
	}
	if err != nil {
		return Map{}, false
	}
	g.maps[number] = m
	return m, true
}

func (g *Game) switchMap(delta int) {
	if delta == 0 {
		return
	}
	n := g.arena.Number
	if isGM2MapID(n) {
		source := gm2SourceMapNumber(n) + delta
		for source < 1 {
			source += 21
		}
		for source > 21 {
			source -= 21
		}
		n = gm2MapID(source)
	} else {
		n += delta
		for n < 1 {
			n += 13
		}
		for n > 13 {
			n -= 13
		}
	}
	if arena, ok := g.ensureMap(n); ok {
		g.arena = arena
		g.resetArenaState()
	}
}

func (g *Game) updateSourceCamera() {
	active := make([]*Player, 0, 4)
	for _, p := range g.players {
		if !p.Active {
			continue
		}
		active = append(active, p)
		if len(active) == 4 {
			break
		}
	}
	if len(active) == 0 {
		return
	}

	maxLeft, maxRight := active[0].X, active[0].X
	maxHigh, maxLow := active[0].Y, active[0].Y
	for i := 1; i < len(active); i++ {
		p := active[i]
		if p.X < maxLeft {
			maxLeft = p.X
		}
		if p.X > maxRight {
			maxRight = p.X
		}
		if p.Y < maxHigh {
			maxHigh = p.Y
		}
		if p.Y > maxLow {
			maxLow = p.Y
		}
	}
	if maxLeft < -100 {
		maxLeft = -100
	}
	if maxRight > 1000 {
		maxRight = 1000
	}
	g.cameraX += ((maxLeft+maxRight)/-2 + 450 - g.cameraX) / 8
	g.cameraX = math.Round(g.cameraX)

	if maxHigh < 50 {
		maxHigh = 50
	}
	if maxLow < 50 {
		maxLow = 50
	}
	if maxLow > 500 {
		maxLow = 500
	}
	if maxHigh > 500 {
		maxHigh = 500
	}
	g.cameraY += ((maxHigh+maxLow)/-2 + 280 - g.cameraY) / 8
	g.cameraY = math.Round(g.cameraY)
}

func (g *Game) worldX(x float64) float64 { return x + g.cameraX }
func (g *Game) worldY(y float64) float64 { return y + g.cameraY }

func (g *Game) Draw(screen *ebiten.Image) {
	setSourceRenderQuality(g.quality)
	// Start desktop audio only after Ebitengine has entered its real draw loop;
	// unit tests construct Game values without opening an audio device.
	if !g.audioStarted {
		g.audioStarted = true
		g.syncSourceMusic()
	}
	g.flushNetClientSFX()
	screen.Fill(color.RGBA{R: 30, G: 35, B: 41, A: 255})
	if g.netClientNeedsWaitingScreen() {
		ebitenutil.DebugPrintAt(screen, "MULTIPLAYER", 20, 20)
		ebitenutil.DebugPrintAt(screen, g.NetplayStatus(), 20, 42)
		ebitenutil.DebugPrintAt(screen, "Waiting for the host to start a game...", 20, 64)
		return
	}
	if g.screen != screenGameplay {
		g.drawMenu(screen)
		g.drawGlobalFade(screen)
		return
	}

	g.assets.EnsureGameplay()
	g.assets.EnsureScene(g.arena.Number)

	// Symbol 1391 source order/matrices: scene1 (0,0), scene2 (0,0),
	// gameplay, foreground (51.1,192.6). SourceRaster preserves each child
	// symbol's XFL registration point, so no recovered/manual crop offsets.
	// scene1.update(): (-root.x,-root.y)*0.9, then root transform itself.
	// Net screen motion is 10% of root camera. scene2 is 50%.
	drawSourceRaster(screen, g.assets.SceneBack[g.arena.Number], g.cameraX*0.1, g.cameraY*0.1, 1, 1, 1)
	drawSourceRaster(screen, g.assets.SceneMid[g.arena.Number], g.cameraX*0.5, g.cameraY*0.5, 1, 1, 1)

	if g.showCollisions {
		for _, r := range g.arena.Platforms {
			ebitenutil.DrawRect(screen, g.worldX(r.X), g.worldY(r.Y), r.W, r.H, color.RGBA{R: 255, G: 40, B: 40, A: 105})
		}
		ebitenutil.DrawLine(screen, 0, g.worldY(g.arena.LowestY), ScreenWidth, g.worldY(g.arena.LowestY), color.RGBA{R: 255, G: 210, B: 0, A: 220})
		ebitenutil.DrawLine(screen, g.worldX(g.arena.SpawnMinX), 0, g.worldX(g.arena.SpawnMinX), ScreenHeight, color.RGBA{R: 50, G: 220, B: 255, A: 180})
		ebitenutil.DrawLine(screen, g.worldX(g.arena.SpawnMaxX), 0, g.worldX(g.arena.SpawnMaxX), ScreenHeight, color.RGBA{R: 50, G: 220, B: 255, A: 180})
	}

	// Keep the arena artwork below dynamic gameplay objects. In Flash the
	// players are attached at runtime and must remain visually in front of the
	// map scene; drawing foreground after them made the character look like it
	// was buried one layer inside the arena.
	drawSourceRaster(screen, g.assets.SceneFront[g.arena.Number], g.worldX(51.1), g.worldY(192.6), 1, 1, 1)

	g.drawPlayerTrails(screen)
	g.drawCombat(screen)
	g.drawPortals(screen)
	g.drawFirePatches(screen)
	g.drawCrates(screen)
	g.drawPowerups(screen)
	g.drawTestGunPickup(screen)
	for _, p := range g.players {
		if p.Active {
			drawP := *p
			drawP.X = g.worldX(p.X)
			drawP.Y = g.worldY(p.Y)
			if g.shouldDrawInvisibleGhost(p) && drawP.Alpha < .24 {
				drawP.Alpha = .24
			}
			g.drawPlayer(screen, &drawP)
		}
	}
	// fx_powerupname and fx_bling are root-attached runtime clips and sit over
	// the player/world art, before the high-depth HUD/player arrows.
	g.drawPowerupPickupEffects(screen)
	g.drawSourceMapFX(screen)
	// Source player_arrow clips use depth 10005..10008; HUD is depth 10010,
	// so off-screen indicators sit above gameplay/map FX but below the HUD.
	g.drawPlayerArrows(screen)
	if !g.GameWin {
		g.drawHUD(screen)
		g.drawExtraCampaignHazardHUD(screen)
		g.drawChaosHUD(screen)
	}
	g.drawGameplayInteractions(screen)
	g.drawGlobalFade(screen)
}

// Lobby and online setup are real interactive client screens. The old generic
// net-client guard treated every non-gameplay screen as a passive wait state,
// hiding MY CHARACTER and the client's own slot editor behind three debug
// lines. Only unknown screens, or a match screen before its first snapshot,
// should use the fallback waiting view.
func (g *Game) netClientNeedsWaitingScreen() bool {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayClient {
		return false
	}
	switch g.screen {
	case screenMultiplayerLobby, screenCustomGame:
		return false
	case screenGameplay, screenPostGame, screenSeriesSummary:
		return !g.netplay.hasState.Load()
	default:
		return true
	}
}

func (g *Game) shouldDrawInvisibleGhost(p *Player) bool {
	if g == nil || p == nil || !p.Active || p.InvisibleTime <= 0 {
		return false
	}
	if g.netplay != nil {
		return p.ID == g.onlineLocalPlayerID()
	}
	// A shared local screen cannot present a different image to each keyboard
	// player. Show the guide only for locally controlled humans, never AI.
	return !p.AI && !p.RemoteControlled
}

func (g *Game) advanceSourceMapFX() {
	if g.arena.Number != 10 || len(g.assets.MapSnowTimelines) == 0 {
		return
	}
	g.mapFXFrame++
	// Symbol 46 Flash frame 127 executes gotoAndPlay(2): zero-based 126 -> 1.
	if g.mapFXFrame >= 126 {
		g.mapFXFrame = 1
	}
}

func (g *Game) drawSourceMapFX(screen *ebiten.Image) {
	switch g.arena.Number {
	case 5:
		// Symbol 1444 frame 5: root-attached mapfx at (0,0), therefore it
		// receives the full _root camera translation.
		drawSourceRaster(screen, g.assets.MapWall, g.cameraX, g.cameraY, 1, 1, 1)
	case 10:
		if g.assets.MapSnow == nil || g.mapFXFrame < 0 {
			return
		}
		// XFL lists top layers first. Ebiten draws back-to-front, so render the
		// two independent Symbol45 snow layers in reverse source layer order.
		for layer := len(g.assets.MapSnowTimelines) - 1; layer >= 0; layer-- {
			timeline := g.assets.MapSnowTimelines[layer]
			if g.mapFXFrame >= len(timeline) {
				continue
			}
			frame := timeline[g.mapFXFrame]
			if !frame.Valid {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(g.assets.MapSnow.Bounds.X, g.assets.MapSnow.Bounds.Y)
			child := geoMFromXFL(frame.Matrix)
			op.GeoM.Concat(child)
			// Symbol 46 frame-1 source: rotation=-20, X=420, Y=350.
			op.GeoM.Rotate(-20 * math.Pi / 180)
			op.GeoM.Translate(g.worldX(420), g.worldY(350))
			screen.DrawImage(g.assets.MapSnow.Image, op)
		}
	}
}

func (g *Game) drawCombat(screen *ebiten.Image) {
	for i := range g.bullets {
		g.drawBullet(screen, &g.bullets[i])
	}
	for _, t := range g.instagibTrails {
		if !t.Dead && t.Alpha > 0.01 && g.assets.InstagibTrail != nil {
			drawSourceRasterRot(screen, g.assets.InstagibTrail, g.worldX(t.X), g.worldY(t.Y), 1, 1, t.Rotation, t.Alpha)
		}
	}

	for _, s := range g.shells {
		if raster := g.assets.Shells[s.Kind]; raster != nil {
			sx := s.ScaleX
			if sx == 0 {
				sx = 1
			}
			drawSourceRasterRot(screen, raster, g.worldX(s.X), g.worldY(s.Y), sx, 1, s.Rotation, 1)
		}
	}

	for _, f := range g.flashes {
		if f.Smoke {
			x, y := float32(g.worldX(f.X)), float32(g.worldY(f.Y))
			s := float32(f.Scale)
			alpha := uint8(math.Max(0, math.Min(180, f.Alpha*180)))
			filledCircle(screen, x, y, 5*s, color.NRGBA{R: 0xc7, G: 0xc7, B: 0xc2, A: alpha})
			filledCircle(screen, x-5*s*float32(f.Facing), y-3*s, 3.8*s, color.NRGBA{R: 0x96, G: 0x96, B: 0x92, A: alpha / 2})
			continue
		}
		if f.Alpha <= 0.01 || g.assets.MuzzleFlash == nil {
			continue
		}
		sy := 1.0
		if f.FlipY {
			sy = -1
		}
		rotation := 0.0
		if f.Facing < 0 {
			rotation = -180
		}
		drawSourceRasterRot(screen, g.assets.MuzzleFlash, g.worldX(f.X), g.worldY(f.Y), 1, sy, rotation, f.Alpha)
	}

	for _, gr := range g.grenades {
		if !gr.Dead {
			g.drawThrowable(screen, gr)
		}
	}
	for _, fx := range g.dynamiteFX {
		if !fx.Dead && fx.Alpha > 0.01 && g.assets.DynamiteFX != nil {
			drawSourceRasterRot(screen, g.assets.DynamiteFX, g.worldX(fx.X), g.worldY(fx.Y), fx.Scale, fx.Scale, fx.Rotation, fx.Alpha)
		}
	}
	for _, fx := range g.blastFX {
		if fx.Dead || fx.Alpha <= 0.01 {
			continue
		}
		if raster := g.assets.ExplosionFX[fx.Kind]; raster != nil {
			drawSourceRasterRot(screen, raster, g.worldX(fx.X), g.worldY(fx.Y), fx.ScaleX, fx.ScaleY, fx.Rotation, fx.Alpha)
		}
	}
	for _, fx := range g.jetThrustFX {
		if fx.Dead || fx.Scale <= 0 {
			continue
		}
		if g.assets.JetThrustBase != nil {
			drawSourceRasterRot(screen, g.assets.JetThrustBase, g.worldX(fx.X), g.worldY(fx.Y), fx.Scale, fx.Scale, fx.Rotation, 1)
		}
		if g.assets.JetThrustFX != nil && fx.FXAlpha > 0 {
			drawSourceRasterRot(screen, g.assets.JetThrustFX, g.worldX(fx.X), g.worldY(fx.Y), fx.Scale, fx.Scale, fx.Rotation, fx.FXAlpha)
		}
	}
	for _, fx := range g.dropPackFX {
		if !fx.Dead && g.assets.DropPack != nil {
			drawSourceRasterRot(screen, g.assets.DropPack, g.worldX(fx.X), g.worldY(fx.Y), fx.ScaleX, 1, fx.Rotation, 1)
		}
	}

	// Do not draw the old synthetic orange explosion circle. Grenade death FX
	// are ported from the original fx_ex* clips separately; until every member
	// of that source effect stack is present, no invented replacement is drawn.
}

func (g *Game) drawBullet(screen *ebiten.Image, b *Bullet) {
	if b == nil || b.Dead {
		return
	}
	if g.drawDevelopedProjectile(screen, b) {
		return
	}
	if b.Kind == BulletInstagib {
		frame := 0
		if len(g.assets.InstagibFX) > 0 {
			frame = b.Time % len(g.assets.InstagibFX)
		}
		if raster := g.assets.InstagibFX[frame]; raster != nil {
			rotation := 0.0
			if b.Facing < 0 {
				rotation = 180
			}
			drawSourceRasterRot(screen, raster, g.worldX(b.X), g.worldY(b.Y), b.VisualScale, b.VisualScale, rotation, b.Alpha)
		}
		return
	}

	if b.TrailAlpha <= 0 || b.TrailWidth <= 0 {
		return
	}
	raster := g.assets.BulletTrail[b.TrailFrame]
	if raster == nil {
		return
	}
	natural := g.assets.BulletTrailNaturalWidth
	if natural <= 0 {
		natural = b.TrailWidth
	}
	sx := b.TrailWidth / natural
	rotation := b.Rotation
	if b.Kind != BulletShotgun && b.Facing < 0 {
		rotation = 180
	}
	drawSourceRasterRot(screen, raster, g.worldX(b.X), g.worldY(b.Y), sx, 1, rotation, b.TrailAlpha*b.Alpha)
}

func (g *Game) drawCrates(screen *ebiten.Image) {
	for _, c := range g.crates {
		if g.assets.Crate != nil {
			drawSourceRaster(screen, g.assets.Crate, g.worldX(c.X), g.worldY(c.Y), float64(c.Facing), 1, 1)
		} else {
			ebitenutil.DrawRect(screen, g.worldX(c.X-20), g.worldY(c.Y-40), 40, 40, color.RGBA{R: 120, G: 80, B: 35, A: 255})
		}
	}
}

func (g *Game) drawPowerups(screen *ebiten.Image) {
	for i := range g.powerups {
		pu := &g.powerups[i]
		if pu.Dead {
			continue
		}

		// Symbol 734 layer order: stuff -> aura -> flashystuff. Once picked up,
		// ActionScript sets stuff._alpha and aura._alpha to zero and leaves only
		// flashystuff until its 72nd frame removes the clip.
		if !pu.PickedUp {
			if pu.Frame >= 0 && pu.Frame < len(g.assets.PowerupStuffTimeline) {
				f := g.assets.PowerupStuffTimeline[pu.Frame]
				if f.Valid {
					drawPowerupChild(screen, g.assets.PowerupIcons[pu.Type], f.Matrix, g.worldX(pu.X), g.worldY(pu.Y), 1)
				}
			}
			if pu.Frame >= 0 && pu.Frame < len(g.assets.PowerupAuraTimeline) {
				f := g.assets.PowerupAuraTimeline[pu.Frame]
				if f.Valid {
					drawPowerupChild(screen, g.assets.PowerupAura, f.Matrix, g.worldX(pu.X), g.worldY(pu.Y), 1)
				}
			}
		}

		if pu.Frame >= 0 && pu.Frame < len(g.assets.PowerupFlashTimeline) {
			f := g.assets.PowerupFlashTimeline[pu.Frame]
			if f.Valid {
				drawPowerupChild(screen, g.assets.PowerupFlashFrame(pu.FlashFrame), f.Matrix, g.worldX(pu.X), g.worldY(pu.Y), 1)
			}
		}
		if g.showCollisions {
			ebitenutil.DebugPrintAt(screen, powerupName(pu.Type), int(g.worldX(pu.X))-25, int(g.worldY(pu.Y))-28)
		}
	}
}

func drawPowerupChild(screen *ebiten.Image, raster *SourceRaster, child xflMatrix, x, y, alpha float64) {
	if raster == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(raster.Bounds.X, raster.Bounds.Y)
	g := geoMFromXFL(child)
	op.GeoM.Concat(g)
	op.GeoM.Translate(x, y)
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	screen.DrawImage(raster.Image, op)
}

func (g *Game) drawPlayer(screen *ebiten.Image, p *Player) {
	colorIndex, useGM2Color := resolvedPlayerColor(p.PlayerColor, p.UseGM2Cosmetics)
	if useGM2Color {
		g.assets.EnsureGM2PlayerColor(colorIndex)
	} else {
		g.assets.EnsurePlayerColor(colorIndex)
	}
	// Flash layer order, from back to front: off-hand, back leg, body,
	// eyes/head details, front leg, gun, front hand. The old flattened player
	// bitmap is no longer used, so changing weapons no longer leaves an M1911
	// baked into the character.
	var hand *SourceRaster
	if useGM2Color {
		hand = g.assets.GM2HandColors[colorIndex]
		if hand == nil {
			g.assets.EnsureGM2PlayerColor(1)
			hand = g.assets.GM2HandColors[1]
		}
	} else {
		hand = g.assets.HandColors[colorIndex]
		if hand == nil {
			hand = g.assets.HandColors[0]
		}
	}
	drawPlayerPartAlpha(screen, hand, p,
		p.VisualHand2X+p.VisualHand2ChildX,
		p.VisualHand2Y+p.VisualHand2ChildY,
		p.VisualHand2Alpha,
	)

	var leg *SourceRaster
	if useGM2Color {
		leg = g.assets.GM2LegColors[colorIndex]
		if leg == nil {
			g.assets.EnsureGM2PlayerColor(1)
			leg = g.assets.GM2LegColors[1]
		}
	} else {
		leg = g.assets.LegColors[colorIndex]
		if leg == nil {
			leg = g.assets.LegColors[0]
		}
	}
	if len(g.assets.Leg2Timeline) > 0 {
		frame := p.LegFrame2 % len(g.assets.Leg2Timeline)
		drawPlayerLeg(screen, leg, p, g.assets.Leg2Timeline[frame].Matrix,
			9*float64(p.Facing), -5.6, p.VisualLeg2Rotation)
	}

	if p.JetpackAlpha > 0 {
		g.drawPlayerJetpack(screen, p)
	}
	g.drawPlayerBody(screen, p)
	// Source hats 8 and 22 swapDepths with eyes and therefore sit behind the
	// eye layer; all other player_hat frames stay in front of it.
	if p.HatNumber == 8 || p.HatNumber == 22 {
		g.drawPlayerHat(screen, p)
	}
	eyes := g.assets.Eyes
	eyeIndex, useGM2Eyes := resolvedPlayerEyes(p.EyeNumber, p.UseGM2Cosmetics)
	if gm2CosmeticsAvailable() && useGM2Eyes {
		frame := eyeIndex - 1
		if frame < 0 {
			frame = 0
		}
		if r := g.assets.EyeFrame(frame); r != nil {
			eyes = r
		}
	}
	drawPlayerPart(screen, eyes, p, 0, p.VisualEyesY)
	if p.HatNumber != 8 && p.HatNumber != 22 {
		g.drawPlayerHat(screen, p)
	}

	if len(g.assets.Leg1Timeline) > 0 {
		frame := p.LegFrame1 % len(g.assets.Leg1Timeline)
		drawPlayerLeg(screen, leg, p, g.assets.Leg1Timeline[frame].Matrix,
			-12*float64(p.Facing), -0.2, p.VisualLeg1Rotation)
	}

	// Symbol688 (`instagib`) is above the front-leg layer and below the gun /
	// front-hand layers in Symbol697.
	g.drawPlayerInstagib(screen, p)
	g.drawGun(screen, p)
	drawPlayerPartAlpha(screen, hand, p,
		p.VisualHand1X+p.VisualHand1ChildX,
		p.VisualHand1Y+p.VisualHand1ChildY,
		p.VisualHand1Alpha,
	)

	g.drawPlayerBurn(screen, p)
	g.drawPlayerShield(screen, p)
	if g.showCollisions {
		h := p.Hitbox()
		ebitenutil.DrawRect(screen, h.X, h.Y, h.W, h.H, color.RGBA{R: 30, G: 255, B: 120, A: 80})
	}

	// Source nametag (Symbol685 + embedded font + GlowFilter) is ported as its
	// own renderer. Do not substitute ebitenutil.DebugPrint here.
}

func (g *Game) drawPlayerBurn(screen *ebiten.Image, p *Player) {
	if p == nil || p.BurnTime <= 0 || !p.Active {
		return
	}
	scale := playerRenderScale(p)
	pulse := float32((p.BurnTick/3)%3) * .65
	for i := 0; i < 4; i++ {
		x := float32(p.X + float64(i)*9*scale - 13.5*scale)
		y := float32(p.Y - (20+float64((i*7+p.BurnTick)%15))*scale)
		radius := float32(6.0*scale) + pulse
		filledCircle(screen, x, y, radius+2, color.NRGBA{R: 0xff, G: 0x3d, B: 0x08, A: 0x70})
		filledCircle(screen, x, y+1, radius*.62, color.NRGBA{R: 0xff, G: 0xd6, B: 0x34, A: 0xd8})
	}
}

func (g *Game) drawPlayerInstagib(screen *ebiten.Image, p *Player) {
	if p == nil || !p.InstagibPlaying || p.InstagibFrame <= 0 ||
		g.assets.InstagibVictim == nil || p.InstagibFrame >= len(g.assets.InstagibVictimTimeline) {
		return
	}
	visual := g.assets.InstagibVictimTimeline[p.InstagibFrame]
	if !visual.Valid || visual.Alpha <= 0 {
		return
	}

	// Symbol697 keeps `instagib` at body._y + 60 - 2.65 and flips its xscale
	// with facing. Symbol688 then animates the red Symbol687 child for 60 frames.
	raster := g.assets.InstagibVictim
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(raster.Bounds.X, raster.Bounds.Y)
	op.GeoM.Concat(geoMFromXFL(visual.Matrix))
	op.GeoM.Scale(float64(p.Facing), 1)
	op.GeoM.Translate(0, p.VisualBodyY+60-2.65)
	scale := playerRenderScale(p)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(p.X, p.Y)
	alpha := p.Alpha * visual.Alpha
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	screen.DrawImage(raster.Image, op)
}

func (g *Game) drawPlayerShield(screen *ebiten.Image, p *Player) {
	if p == nil || p.ShieldAlpha <= 0 || p.ShieldFrame < 0 || p.ShieldFrame >= len(g.assets.ShieldTimeline) {
		return
	}
	visual := g.assets.ShieldTimeline[p.ShieldFrame]
	if !visual.Valid {
		return
	}
	raster := g.assets.ShieldFrames[p.ShieldChildFrame]
	if raster == nil {
		return
	}

	// Symbol 697 places Symbol 696 at (0,-35). Symbol 696 then applies the
	// current frame matrix/alpha to its nested Symbol 695 pulse.
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(raster.Bounds.X, raster.Bounds.Y)
	child := geoMFromXFL(visual.Matrix)
	op.GeoM.Concat(child)
	op.GeoM.Translate(0, -35)
	scale := playerRenderScale(p)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(p.X, p.Y)
	alpha := p.Alpha * p.ShieldAlpha * visual.Alpha
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	screen.DrawImage(raster.Image, op)
}

func (g *Game) drawPlayerBody(screen *ebiten.Image, p *Player) {
	// Body colour and clothing are resolved independently in the developed
	// editor. This lets a player combine a GM1 hat with a GM2 shirt/colour while
	// GM2 campaign enemies can still force their raw source IDs as one set.
	var torso, head, shirt *SourceRaster
	colorIndex, useGM2Color := resolvedPlayerColor(p.PlayerColor, p.UseGM2Cosmetics)
	shirtIndex, useGM2Shirt := resolvedPlayerShirt(p.ShirtNumber, p.UseGM2Cosmetics)
	if useGM2Color {
		g.assets.EnsureGM2PlayerColor(colorIndex)
		torso = g.assets.GM2TorsoColors[colorIndex]
		head = g.assets.GM2HeadColors[colorIndex]
	} else {
		g.assets.EnsurePlayerColor(colorIndex)
		torso = g.assets.TorsoColors[colorIndex]
		if torso == nil {
			torso = g.assets.TorsoColors[0]
		}
		head = g.assets.HeadColors[colorIndex]
		if head == nil {
			head = g.assets.HeadColors[0]
		}
	}
	shirtFrame := shirtIndex - 1
	if shirtFrame < 0 {
		shirtFrame = 0
	}
	if useGM2Shirt {
		shirt = g.assets.GM2ShirtFrame(shirtFrame)
	} else {
		shirt = g.assets.ShirtFrame(shirtFrame)
	}

	drawPlayerPart(screen, torso, p, 0, p.VisualBodyY+25.3)
	drawPlayerPart(screen, shirt, p, 0, p.VisualBodyY+26.1)

	if head == nil {
		return
	}
	headFrame := SourceTransformFrame{Matrix: xflMatrix{A: 1, D: 1}, Valid: true}
	if p.HeadFrame >= 0 && p.HeadFrame < len(g.assets.HeadTimeline) && g.assets.HeadTimeline[p.HeadFrame].Valid {
		headFrame = g.assets.HeadTimeline[p.HeadFrame]
	}
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(head.Bounds.X, head.Bounds.Y)
	inner := geoMFromXFL(g.assets.HeadInnerMatrix)
	op.GeoM.Concat(inner)
	headMove := geoMFromXFL(headFrame.Matrix)
	op.GeoM.Concat(headMove)
	op.GeoM.Translate(9.4, p.VisualBodyY+4)
	op.GeoM.Scale(float64(p.Facing), 1)
	scale := playerRenderScale(p)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(p.X, p.Y)
	if p.Alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(p.Alpha))
	}
	screen.DrawImage(head.Image, op)
}

func (g *Game) drawPlayerHat(screen *ebiten.Image, p *Player) {
	hatIndex, useGM2Hat := resolvedPlayerHat(p.HatNumber, p.UseGM2Cosmetics)
	frame := hatIndex - 1
	if frame < 0 {
		frame = 0
	}
	var hat *SourceRaster
	if useGM2Hat {
		hat = g.assets.GM2HatFrame(frame)
	} else {
		hat = g.assets.HatFrame(frame)
	}
	if hat == nil {
		return
	}
	headY := 0.0
	if p.HeadFrame >= 0 && p.HeadFrame < len(g.assets.HeadTimeline) && g.assets.HeadTimeline[p.HeadFrame].Valid {
		headY = g.assets.HeadTimeline[p.HeadFrame].Matrix.TY
	}
	// DefineSprite_697 onEnterFrame:
	// player_hat._x = 9.55*facing;
	// player_hat._y = -75 + body.head.head._y + (body._y + 60).
	scale := playerRenderScale(p)
	drawSourceRaster(screen, hat,
		p.X+9.55*scale*float64(p.Facing),
		p.Y+(-75+headY+(p.VisualBodyY+60))*scale,
		scale*float64(p.Facing), scale, p.Alpha,
	)
}

func (g *Game) drawPlayerJetpack(screen *ebiten.Image, p *Player) {
	if g.assets.JetpackBase == nil {
		return
	}
	alpha := p.Alpha * p.JetpackAlpha
	drawSourceRaster(screen, g.assets.JetpackBase,
		p.X,
		p.Y+(p.VisualBodyY+14.6)*playerRenderScale(p),
		playerRenderScale(p)*float64(p.Facing), playerRenderScale(p), alpha,
	)

	if g.assets.JetpackFuel == nil || p.JetFuel <= 0 {
		return
	}
	fuelScale := p.JetFuel / 100
	if fuelScale < 0 {
		fuelScale = 0
	}
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(g.assets.JetpackFuel.Bounds.X, g.assets.JetpackFuel.Bounds.Y)
	op.GeoM.Scale(1, fuelScale)
	child := geoMFromXFL(g.assets.JetpackFuelMatrix)
	op.GeoM.Concat(child)
	op.GeoM.Scale(float64(p.Facing), 1)
	op.GeoM.Translate(0, p.VisualBodyY+14.6)
	scale := playerRenderScale(p)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(p.X, p.Y)
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	screen.DrawImage(g.assets.JetpackFuel.Image, op)
}

func (g *Game) drawGun(screen *ebiten.Image, p *Player) {
	weaponColor := p.PlayerColor
	if source, useGM2 := resolvedPlayerColor(p.PlayerColor, p.UseGM2Cosmetics); useGM2 {
		weaponColor = -source
	}
	gun := g.assets.WeaponFrameForPlayer(p.Weapon.Def.Number, p.Weapon.Frame, weaponColor)
	if gun == nil {
		return
	}
	// Exact hierarchy: gun local transform -> hand1 facing/position -> player
	// 80% scale/position. The gun's current frame comes from its own timeline.
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(gun.Bounds.X, gun.Bounds.Y)
	op.GeoM.Rotate(p.VisualGunRotation * math.Pi / 180)
	op.GeoM.Translate(p.VisualGunX, p.VisualGunY)
	op.GeoM.Scale(float64(p.Facing), 1)
	op.GeoM.Translate(p.VisualHand1X, p.VisualHand1Y)
	scale := playerRenderScale(p)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(p.X, p.Y)
	gunAlpha := p.Alpha * p.Weapon.Alpha
	if gunAlpha < 1 {
		op.ColorScale.ScaleAlpha(float32(gunAlpha))
	}
	screen.DrawImage(gun.Image, op)
}

func playerRenderScale(p *Player) float64 {
	if p.PlayerScale <= 0 {
		return 0.8
	}
	return p.PlayerScale
}

func drawPlayerPart(screen *ebiten.Image, raster *SourceRaster, p *Player, x, y float64) {
	drawPlayerPartAlpha(screen, raster, p, x, y, 1)
}

func drawPlayerPartAlpha(screen *ebiten.Image, raster *SourceRaster, p *Player, x, y, localAlpha float64) {
	if raster == nil || localAlpha <= 0 {
		return
	}
	scale := playerRenderScale(p)
	facing := float64(p.Facing)
	drawSourceRaster(screen, raster,
		p.X+x*scale*facing,
		p.Y+y*scale,
		scale*facing, scale, p.Alpha*localAlpha,
	)
}

func drawPlayerLeg(screen *ebiten.Image, raster *SourceRaster, p *Player, child xflMatrix, x, y, degrees float64) {
	if raster == nil {
		return
	}

	// Exact hierarchy from XFL:
	// Symbol188 raster -> Symbol282/189 frame matrix -> runtime leg xscale/
	// rotation/x/y from player ActionScript -> player _xscale/_yscale 80%.
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(raster.Bounds.X, raster.Bounds.Y)
	childGeo := geoMFromXFL(child)
	op.GeoM.Concat(childGeo)
	op.GeoM.Scale(float64(p.Facing), 1)
	op.GeoM.Rotate(degrees * math.Pi / 180)
	op.GeoM.Translate(x, y)
	scale := playerRenderScale(p)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(p.X, p.Y)
	if p.Alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(p.Alpha))
	}
	screen.DrawImage(raster.Image, op)
}

func geoMFromXFL(m xflMatrix) ebiten.GeoM {
	var g ebiten.GeoM
	g.SetElement(0, 0, m.A)
	g.SetElement(0, 1, m.C)
	g.SetElement(0, 2, m.TX)
	g.SetElement(1, 0, m.B)
	g.SetElement(1, 1, m.D)
	g.SetElement(1, 2, m.TY)
	return g
}

func (g *Game) drawHUD(screen *ebiten.Image) {
	g.drawSourceHUDCards(screen)
	g.drawKillFeeds(screen)
	if g.showCollisions {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("DEV | mapa %d/13 | gamemode %d | bron: %s | F2/F3 mapa | R reset", g.arena.Number, g.GameMode, g.players[0].Weapon.Def.DisplayName()), 10, 30)
		ebitenutil.DebugPrintAt(screen, "P1 strzalki + [ ] | P2 WASD + T Y | F4 AI P2 | F1 wylacz debug", 10, 46)
	}
}

func drawImageAt(dst, src *ebiten.Image, x, y float64) {
	if src == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	dst.DrawImage(src, op)
}

func (g *Game) Layout(_, _ int) (int, int) {
	return ScreenWidth, ScreenHeight
}
