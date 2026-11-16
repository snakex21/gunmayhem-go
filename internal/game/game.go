package game

import (
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
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
	areaEffects      []AreaEffect // gas / smoke clouds, flashbang bursts
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
	// SLOW-MO (slowmo.go): the sim-step clock, and whether this real tick
	// skipped the simulation (the host then holds the input acks).
	slowMo        simClock
	slowMoSkipped bool
	inSimStep     bool // sounds emitted now are gameplay sounds (slowed in SLOW-MO)

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
	// EXTRA hazard state (extra_campaign_hazards.go).
	extraHazardCycle    int
	extraHazardMarkers  []HazardMarker
	extraHazardPendingX float64
	extraHazardPending  bool
	extraWindDir        int
	extraWindTime       int
	// Non-player campaign enemies (campaign_enemies.go) and boss banner.
	campaignEnemies     []*CampaignEnemy
	campaignEnemyWaves  []campaignEnemyWave
	nextCampaignEnemyID int
	campaignEnemyFrame  int
	// Stage B: marked mortar strikes waiting to drop (host only) and the
	// heavy mech's platform graph (built once per arena).
	campaignStrikes []campaignStrike
	mechNav         *zombieNav
	bossBanner      string
	bossBannerFrame int

	// ZOMBIE mode (zombie_*.go). zombie is nil outside a zombie match.
	zombie              *zombieMatch
	zombieSetup         zombieSetupConfig
	zombieUI            uiState
	zombieRecords       map[string]ZombieRecord
	onlineZombie        bool // the online session in the lobby is a zombie co-op session
	zombieBulletScratch []int
	// Multi-zone zombie tick (zombie_zones.go): sounds of a zone that this
	// screen does not show are not played locally; sfxZoneTag (zone+1, 0 =
	// every zone) tags the net SFX events for the guests of that zone.
	sfxLocalMute bool
	sfxZoneTag   int

	// arenagamedata2 defaults used by the source Options screen.
	musicOn                 bool
	soundOn                 bool
	musicVolume             float64
	soundVolume             float64
	loudnessEQ              bool
	allowCheats             bool
	quality                 int
	audio                   *sourceAudioEngine
	audioStarted            bool
	controlConfigs          [4]Controls
	optionsKeyCapturePlayer int
	optionsKeyCaptureIndex  int
	optionsTab              int
	optionsUI               uiState
	mainMenuUI              uiState
	mainMenuQuitOpen        bool
	quitRequested           bool
	customUI                uiState
	// Online Mayhem UI (online_browser_menu.go, online_lobby_menu.go). The
	// lobby itself shares customUI with the local Custom Game builders.
	onlineUI               uiState
	onlineUIScreen         int  // 1 + onlineScreen* shown last (0: none yet)
	onlineSessionTab       bool // host lobby shows the SESSION tab
	onlineClientTab        int  // guest lobby tab (view only, see setOnlineLobbyTab)
	onlineClientTabSet     bool
	onlineLeaveConfirm     bool
	onlineBrowserLastFocus string // server row focused before this frame's key
	onlineLastMessage      string // multiplayerMessage already shown as a toast
	// customLoadoutSlot is 1-based: the player whose Loadout screen
	// (loadout_menu.go) is open over the PLAYERS tab; 0 when closed.
	customLoadoutSlot         int
	customLoadoutTab          int
	customLoadoutPart         int  // CHARACTER tab: colour / hat / shirt / face
	customLoadoutFocusPending bool // focus the selected tile on the next update
	customLoadoutReturn       int  // 1-based slot whose EDIT button gets focus back
	loadoutPreviewImg         *ebiten.Image
	loadoutThumbs             map[[2]int]*SourceRaster // UI-only tight GM2 cosmetic crops
	setupPreviewImg           *ebiten.Image
	uiLastCursorX             float64
	uiLastCursorY             float64

	// Local display settings (ui_display.go). Presentation only: never part of
	// the simulation, the netplay snapshot or the host's authority.
	displayScaling      int
	displayFilter       int
	displayFrameRate    int  // displayFrameRate* (render_interp.go)
	displaySmoothMotion bool // render interpolation between ticks
	displayScreenShake  bool
	displayPowerSave    bool
	displayMonitor      int
	displayCustomW      int
	displayCustomH      int
	finalGeoValid       bool
	finalEbitenGeo      ebiten.GeoM
	finalGeo            ebiten.GeoM
	shakeX              float64
	shakeTick           int64
	shakeSeen           []shakeBlastKey
	renderInterp        renderInterpState
	frameLimiter        frameLimiter
	// Online guest: own-player prediction and the F6 network overlay.
	netPredict      netPredictor
	netDebugOverlay bool
	powerSaveDrawn  bool
	powerSaveTick   int64

	gunLibrarySelected          int
	gunLibraryPressed           int
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
	multiplayerDiscoveryCh    chan serverDiscoveryResult
	multiplayerDiscoveryDone  bool
	localServerAdvertiser     *localServerAdvertiser
	lanDiscoveryResponder     *lanDiscoveryResponder
	p2pHost                   *p2pQUICHost
	cloudflareTunnel          *cloudflareQuickTunnel
	pendingNetJoin            *pendingNetJoin // async join in progress (online_ux.go)
	inviteCopyFeedback        string
	inviteCopyFeedbackUntil   time.Time
	runnableUnfocused         bool   // last value passed to ebiten.SetRunnableOnUnfocused
	netDisconnectNotice       string // why the last online session ended (net_session_ux.go)
	netDisconnectNoticeUntil  time.Time
	onlineSessionVisibility   int // OPEN vs INVITE CODE
	onlineSessionScope        int // LAN vs INTERNET+LAN
	onlineSessionKey          string
	onlineMatchSetup          bool
	onlinePlayerCount         int
	onlinePlayerSlots         [4]bool
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
	onlineBlockedPerks        [onlinePerkTableSize]bool
	onlineBlockedAbilities    [abilityCount]bool
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
		displayFrameRate:          displayFrameRateVSync,
		displaySmoothMotion:       true,
		displayScreenShake:        true,
		windowedWidth:             ScreenWidth,
		windowedHeight:            ScreenHeight,
		audio:                     newSourceAudioEngine(),
		multiplayerAddress:        "127.0.0.1:7777",
		multiplayerSessionName:    "My Gun Mayhem Game",
		multiplayerSelectedServer: -1,
		customHostScope:           onlineSessionScopeLAN,
		customHostAccess:          onlineSessionOpen,
		onlineOnlyWeapon:          10,
		onlinePerksEnabled:        true,
		onlinePowerupsEnabled:     true,
		zombieSetup:               defaultZombieSetup(),
		zombieRecords:             make(map[string]ZombieRecord),
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
	if g.updateUpdaterRuntime() {
		return ebiten.Termination
	}
	if g.quitRequested {
		_ = g.CloseNetplay()
		return ebiten.Termination
	}
	// Render interpolation: remember where everything was before this tick.
	g.captureRenderPrev(time.Now())
	g.slowMoSkipped = false
	g.renderInterp.simPhase, g.renderInterp.simInc = 0, 1
	if g.netplay == nil || g.netplay.mode != netplayClient {
		// Only a guest draws a predicted own player (freeID).
		g.renderInterp.freeID = 0
	}
	g.trackWindowPersistence()
	g.updateUnfocusedRunPolicy()
	g.updateNetDebugKeys()
	if g.updatePendingNetJoin() {
		g.updateUnfocusedRunPolicy()
		return nil
	}
	g.updateInviteCodeClipboard()
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
		g.clearSlowMoLatches()
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

	// ZOMBIE: guests' shop requests and the local shop panels run every
	// real tick; input masks and the mystery box (DOWN) belong to the sim
	// step, before the players move.
	g.processZombieRequests()
	g.updateZombieShopUI()
	// SLOW-MO: everything below is one gameplay sim step; while slowed only
	// every other real tick runs it (slowmo.go).
	if !g.beginSimTick() {
		return nil
	}
	g.inSimStep = true
	g.updateZombiePreInput()

	if !g.runZombieZoneTick() {
		g.simulateArenaTick(true)
	}
	g.updateZombieMatch()
	g.updateMatchInteractions()
	g.inSimStep = false
	g.clearSlowMoLatches()
	// A SLOW-MO started (or reset) inside this step changes how the step
	// just simulated is spread over the real ticks: start it at half rate.
	g.renderInterp.simPhase, g.renderInterp.simInc = g.slowMo.renderPhase()
	return nil
}

// simulateArenaTick is one gameplay step of the world in Game (players,
// enemies, zombies, projectiles, effects). A multi-zone zombie match runs it
// once per zone with that zone's lists swapped in (zombie_zones.go);
// primary is false for the zones this screen does not show, which skips the
// once-per-tick parts (HUD, kill feed, map FX, pickup timers).
func (g *Game) simulateArenaTick(primary bool) {
	chaosKnockback := beginAbilityKnockbackWindow(g.players)
	if primary {
		g.updateChaosMode()
	}
	// Chaos wind / shockwaves follow HEAVY, LIGHT and ANCHOR but never spend
	// the one-hit SHIELD.
	endAbilityKnockbackWindow(g.players, chaosKnockback, false)
	g.syncRuntimePlayerRuleModifiers()
	runtimeArena := g.customRuntimeArena()
	weaponFiredInUpdate := make([]bool, len(g.players))
	for i, p := range g.players {
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
				afterPhysics = func() {
					decideAIController(p, g)
					g.zombieBotAssist(p)
				}
			}
			// The weapon fires inside the player update, before that frame's
			// friction, as in the source onEnterFrame (recoil, then vx *= 0.93).
			fire := func() {
				weaponFiredInUpdate[i] = true
				g.updatePlayerWeapon(p)
			}
			p.UpdateWithFire(runtimeArena, beforeMove, afterPhysics, fire)
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
		} else if p.DashTime > 0 && p.DashTime%2 == 0 && p.Active {
			// DASH after-images reuse the speed-powerup trail.
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
			g.dropHarpoonRopesOf(p.ID)
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
	for i, p := range g.players {
		if !g.GameWin {
			if i >= len(weaponFiredInUpdate) || !weaponFiredInUpdate[i] {
				// The frame ended before the shooting step (death/respawn or an
				// inactive player): keep the previous post-update firing.
				g.updatePlayerWeapon(p)
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
	if primary {
		g.updateHUDLastLife()
	}
	g.updateSourceCamera()
	if primary {
		g.advanceSourceMapFX()
	}

	// Non-player campaign enemies move, shove and shoot before the hit
	// phase; their shoves sit in the players' knockback window.
	enemyKnockback := beginAbilityKnockbackWindow(g.players)
	g.updateCampaignEnemies(runtimeArena)
	g.updateZombies(runtimeArena)
	endAbilityKnockbackWindow(g.players, enemyKnockback, true)
	g.updateZombieBlasts()
	g.updateExtraCampaignBosses()
	// targets = players + living enemy bodies: every hit source below sees
	// both through the same code paths and knockback windows.
	targets := g.hitTargets()
	g.beginCampaignEnemyHitPhase()
	g.beginZombieHitPhase()

	hitsBefore := 0
	for _, p := range targets {
		hitsBefore += p.HitNumber
	}
	projectileBlastBefore := len(g.blastFX)
	bulletKnockback := beginAbilityKnockbackWindow(targets)
	if g.deflectShieldBullets(targets) > 0 {
		g.playSourceSFX("bolt1.wav", false)
	}
	shieldSparksBefore := countBallisticShieldSparks(targets)
	// Bat swings and leaf blower gusts (arsenal2_combat.go).
	g.applyArsenal2Hits(targets)
	// Gunship / mech rounds stop on platforms; bounced gunship rockets fly
	// back at it (campaign_enemies_big.go).
	g.updateCampaignBossBullets(targets)
	g.zombieBulletScratch = g.zombieBulletsAlive(g.zombieBulletScratch)
	updateBullets(runtimeArena, g.bullets, targets, &g.instagibTrails, &g.explosions, &g.blastFX, &g.portals)
	g.markZombieHeadshots(g.zombieBulletScratch)
	shieldBroke := endAbilityKnockbackWindow(targets, bulletKnockback, true)
	if countBallisticShieldSparks(targets) > shieldSparksBefore {
		// A round rang off a BALLISTIC SHIELD.
		g.playSourceSFX("bolt1.wav", false)
	}
	if len(g.blastFX) > projectileBlastBefore {
		g.playRandomExplosionSound()
	}
	hitsAfter := 0
	for _, p := range targets {
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
		grenadeVX = make([]float64, len(targets))
		grenadeVY = make([]float64, len(targets))
		for i, p := range targets {
			grenadeVX[i] = p.VX
			grenadeVY[i] = p.VY
		}
	}
	firePatchesBefore := len(g.firePatches)
	areaEffectsBefore := len(g.areaEffects)
	blastKnockback := beginAbilityKnockbackWindow(targets)
	updateGrenadesWithAreas(runtimeArena, g.grenades, targets, &g.explosions, &g.dynamiteFX, &g.blastFX, &g.firePatches, &g.areaEffects)
	if len(g.firePatches) > firePatchesBefore {
		g.playSourceSFXGain("dev_flame.wav", true, 1.6) // Molotov bottle bursting into flame
	}
	g.playAreaEffectSpawnSounds(areaEffectsBefore)
	g.updateAIFlashbangAwareness()
	updateFirePatches(g.firePatches, targets)
	updateAreaEffects(g.areaEffects, targets)
	// Portal travel redirects velocity; it is not knockback, so the window
	// closes before it.
	if endAbilityKnockbackWindow(targets, blastKnockback, true) {
		shieldBroke = true
	}
	// Portals expire once their owner no longer holds the Portal Gun.
	g.portals = prunePortals(g.portals, targets)
	updatePortalTravel(g.portals, g.players, g.grenades)
	if grenadeVX != nil {
		for i, p := range targets {
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
	hazardKnockback := beginAbilityKnockbackWindow(targets)
	g.updateExtraCampaignHazards()
	if endAbilityKnockbackWindow(targets, hazardKnockback, true) {
		shieldBroke = true
	}
	g.endCampaignEnemyHitPhase(runtimeArena)
	g.endZombieHitPhase()
	if shieldBroke {
		g.playSourceSFX("emp.wav", true)
	}
	updateDynamiteEffects(g.dynamiteFX)
	updateBlastEffects(runtimeArena, g.blastFX)
	updateJetThrustEffects(g.jetThrustFX)
	updateDropPackEffects(runtimeArena, g.dropPackFX)
	if primary {
		g.crateTimer++
	}
	if primary && g.crateTimer >= g.customCrateInterval() && g.GameMode != SourceGameModeGunGame && g.customCratesEnabled() {
		c := spawnCrate(runtimeArena)
		c.Weapon = g.rollCrateWeapon()
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
	if primary {
		g.powerupTimer++
	}
	if primary && g.powerupTimer >= 600 && g.GameMode != SourceGameModeGunGame && g.GameMode != SourceGameModeSurvival && g.PowerON {
		// DefineSprite_734 replaces the Double powerup with shield whenever a
		// helper already exists OR campaignmode is active.
		pu := spawnPowerup(runtimeArena, g.hasActiveDouble() || g.campaignMode, g.extendedPickups())
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
		if pu.Type == powerupSlowMo && pu.PickedByID > 0 {
			g.onSlowMoPickup()
		}
		if pu.Type == 0 && pu.PickedByID > 0 {
			// lifebling() -> fx_bling mod=6 -> source green "1-UP".
			g.lifeBlingFX = append(g.lifeBlingFX, newLifeBlingEffect(pu.PickedByID))
		}
	}
	g.updatePowerupPickupEffects()
	if primary {
		g.updateTestGunPickup()
		g.spawnRequestedDoubles()
	}
	updateExplosions(g.explosions)
	if primary {
		updateKillFeeds(g.killfeeds)
	}
	g.bullets = compactBullets(g.bullets)
	g.instagibTrails = compactInstagibTrails(g.instagibTrails)
	g.playerTrails = compactPlayerTrailEffects(g.playerTrails)
	g.shells = compactShells(g.shells)
	g.flashes = compactFlashes(g.flashes)
	g.grenades = compactGrenades(g.grenades)
	g.firePatches = compactFirePatches(g.firePatches)
	g.areaEffects = compactAreaEffects(g.areaEffects)
	g.dynamiteFX = compactDynamiteEffects(g.dynamiteFX)
	g.blastFX = compactBlastEffects(g.blastFX)
	g.jetThrustFX = compactJetThrustEffects(g.jetThrustFX)
	g.dropPackFX = compactDropPackEffects(g.dropPackFX)
	g.explosions = compactExplosions(g.explosions)
	if primary {
		g.killfeeds = compactKillFeeds(g.killfeeds)
	}
	g.crates = compactCrates(g.crates)
	g.powerups = compactPowerups(g.powerups)
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
	g.areaEffects = g.areaEffects[:0]
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
	g.slowMo = simClock{}
	g.clearSlowMoLatches()
	for _, p := range g.players {
		p.ResetRound(g.arena)
		g.seenDeaths[p.ID] = 0
	}
	g.resetHUDLastLife()
	// Campaign-only arena state: enemy waves, hazard markers, boss banner.
	g.extraHazardFrame = 0
	g.extraHazardCycle = 0
	g.extraHazardMarkers = g.extraHazardMarkers[:0]
	g.extraHazardPending = false
	g.extraWindDir, g.extraWindTime = 0, 0
	g.bossBanner, g.bossBannerFrame = "", 0
	g.resetCampaignEnemies()
	// A new arena state ends any zombie match (startZombieMatch creates its
	// match after this reset).
	g.zombie = nil
	if m := g.activeExtraCampaignMission(); m != nil {
		g.setupCampaignEnemyWaves(m.Units)
	}
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

// updatePlayerWeapon runs one player's shooting step for this frame: the
// trigger/ROF check and FIREBULLET() (with recoil) plus the custom-modifier and
// gun-sound bookkeeping around it.
func (g *Game) updatePlayerWeapon(p *Player) {
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
		// HIMARS: drop on the nearest enemy ahead and mark the landing point.
		g.aimHimarsStrikes(p, g.bullets[bulletsBefore:])
		// Flashbang blindness and smoke throw this frame's shots off.
		g.applyAimDisturbance(p, g.bullets[bulletsBefore:])
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
		if spec, ok := developedWeaponSpec(weaponBefore); ok && (spec.Projectile == DevelopedProjectileFlame || spec.Projectile == DevelopedProjectileGust) {
			// The flame loop is built from overlapping short bursts. Starting a
			// new PCM clip every (30 Hz) tick would stack six copies and clip.
			playSound = p.ShotsFired%6 == 1
		}
		if playSound {
			g.playSourceGunSound(weaponBefore)
		}
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

func (g *Game) Layout(_, _ int) (int, int) {
	return ScreenWidth, ScreenHeight
}
