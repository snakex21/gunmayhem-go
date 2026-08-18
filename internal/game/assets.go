package game

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
)

type Assets struct {
	SceneBack           map[int]*SourceRaster
	SceneMid            map[int]*SourceRaster
	SceneFront          map[int]*SourceRaster
	sceneLoaded         map[int]bool
	interactionLoaded   bool
	gunLibraryLoaded    bool
	playerVisualsLoaded bool
	gameplayLoaded      bool
	MainMenu            *SourceRaster
	CustomMenu          *SourceRaster
	MenuChosen          *SourceRaster
	CampaignMenu        *SourceRaster
	OptionsMenu         *SourceRaster
	CreditsMenu         *SourceRaster
	GunLibraryMenu      *SourceRaster

	// Root/menu interaction clips. These are kept frame-by-frame because the
	// source ActionScript drives their timelines directly (fadeaway, countdown,
	// campaign unlocks, etc.) rather than treating them as static overlays.
	CampaignBase            *SourceRaster
	CampaignSlider          *SourceRaster
	CampaignBack            *SourceRaster
	CampaignUnlock          map[int]*SourceRaster
	CampaignPlayerCard      *SourceRaster
	CampaignInfoPanel       *SourceRaster
	CampaignStartButton     *SourceRaster
	CampaignInfoFrames      map[int]*SourceRaster
	CampaignLevelIcons      map[int]*SourceRaster
	CampaignLevelLocks      map[int]*SourceRaster
	CampaignLevelBase       *SourceRaster
	PauseMenu               *SourceRaster
	FadeFrames              map[int]*SourceRaster
	WinCountdownFrames      map[int]*SourceRaster // FFDec fallback only; exported frames are frozen on "3"
	WinCountdownTimelines   map[int][]SourceTransformFrame
	WinnerFrames            map[int]*SourceRaster
	CampaignLoseFrames      map[int]*SourceRaster
	TeamWinFrames           map[int]*SourceRaster
	ZombieWaveFrames        map[int]*SourceRaster
	PostGameMenu            *SourceRaster // flattened fallback only; contains baked placeholder values
	PostGameBackground      *SourceRaster
	PostGamePanelBase       *SourceRaster
	PostGameButton          *SourceRaster
	PostGameWinnerBorder    *SourceRaster
	OptionToggleFrames      map[int]*SourceRaster
	OptionLockup            *SourceRaster
	OptionPanelBase         *SourceRaster
	OptionKeyFrames         map[int]*SourceRaster
	OptionArrowGlyphs       map[int]*SourceRaster
	OptionLockupBase        *SourceRaster
	GunLibraryStats         map[int]*SourceRaster
	GunLibraryDrops         map[int]*SourceRaster
	GunLibraryThumbs        map[int]*SourceRaster
	GunLibraryLockedOverlay *SourceRaster
	TestPickupBase          *SourceRaster
	CustomWarningLives      map[int]*SourceRaster
	CustomWarningPlayers    map[int]*SourceRaster
	CustomPlayerCard        *SourceRaster
	CustomEditButton        *SourceRaster
	PlayerColorMask         *SourceRaster
	CustomSelectionFrames   map[int]*SourceRaster
	StarterGunFrames        map[int]*SourceRaster
	PerkFrames              map[int]*SourceRaster
	ModeDisplayFrames       map[int]*SourceRaster
	ModeCheckFrames         map[int]*SourceRaster
	MapPreviewImages        map[int]*SourceRaster
	CampaignThumbs          map[int]*SourceRaster
	CampaignLockOverlay     *SourceRaster
	CampaignDoneOverlay     *SourceRaster

	TorsoColors       map[int]*SourceRaster
	HeadColors        map[int]*SourceRaster
	playerColorLoaded map[int]bool
	ShirtFrames       map[int]*SourceRaster
	HatFrames         map[int]*SourceRaster
	HeadTimeline      []SourceTransformFrame
	HeadInnerMatrix   xflMatrix
	Eyes              *SourceRaster
	HandColors        map[int]*SourceRaster
	ShieldFrames      map[int]*SourceRaster
	ShieldTimeline    []SourceVisualFrame
	JetpackBase       *SourceRaster
	JetpackFuel       *SourceRaster
	JetpackFuelMatrix xflMatrix
	JetThrustBase     *SourceRaster
	JetThrustFX       *SourceRaster
	DropPack          *SourceRaster

	// Symbol 188 owns the leg color frames. Symbol 282 and Symbol 189 own
	// the walking transforms. Keeping these separate mirrors the Flash source.
	LegColors    map[int]*SourceRaster
	Leg1Timeline []SourceTransformFrame
	Leg2Timeline []SourceTransformFrame

	Crate                   *SourceRaster
	CrateHit                Rect
	Grenade                 *SourceRaster
	DynamiteFX              *SourceRaster
	ExplosionFX             map[BlastEffectKind]*SourceRaster
	Shells                  map[int]*SourceRaster
	PlayerArrowFrames       map[int]*SourceRaster
	MuzzleFlash             *SourceRaster
	BulletTrail             map[int]*SourceRaster
	BulletTrailNaturalWidth float64
	InstagibFX              map[int]*SourceRaster
	InstagibTrail           *SourceRaster
	InstagibVictim          *SourceRaster
	InstagibVictimTimeline  []SourceVisualFrame
	PowerupIcons            map[int]*SourceRaster
	PowerupAura             *SourceRaster
	PowerupStuffTimeline    []SourceTransformFrame
	PowerupAuraTimeline     []SourceTransformFrame
	PowerupFlashTimeline    []SourceTransformFrame
	PowerupFlashFrames      map[int]*SourceRaster
	PowerupNameFrames       map[int]*SourceRaster
	LifeBling               *SourceRaster
	MapWall                 *SourceRaster
	MapSnow                 *SourceRaster
	MapSnowTimelines        [][]SourceTransformFrame
	HUDCard                 *SourceRaster
	HUDCardBG               *SourceRaster
	HUDCardDivider          *SourceRaster
	HUDGameOver             *SourceRaster
	HUDLevelUp              *SourceRaster
	HUDLastLifeFrames       map[int]*SourceRaster
	HUDLastLifeTimelines    [3][]SourceTransformFrame
	WeaponFrames            map[int]map[int]*SourceRaster
}

func LoadAssets() *Assets {
	a := &Assets{
		SceneBack:   make(map[int]*SourceRaster),
		SceneMid:    make(map[int]*SourceRaster),
		SceneFront:  make(map[int]*SourceRaster),
		sceneLoaded: make(map[int]bool),
		// Only Main Menu is needed before the first frame. Secondary screens are
		// loaded behind the first source fade by EnsureInteractions/EnsureGunLibrary.
		MainMenu:              loadSourceRaster("Symbol 28", 0, "sprites", "DefineSprite_28_mainmenu", "1.png"),
		CampaignUnlock:        make(map[int]*SourceRaster),
		CampaignInfoFrames:    make(map[int]*SourceRaster),
		CampaignLevelIcons:    make(map[int]*SourceRaster),
		CampaignLevelLocks:    make(map[int]*SourceRaster),
		FadeFrames:            make(map[int]*SourceRaster),
		WinCountdownFrames:    make(map[int]*SourceRaster),
		WinCountdownTimelines: make(map[int][]SourceTransformFrame),
		WinnerFrames:          make(map[int]*SourceRaster),
		CampaignLoseFrames:    make(map[int]*SourceRaster),
		TeamWinFrames:         make(map[int]*SourceRaster),
		ZombieWaveFrames:      make(map[int]*SourceRaster),
		OptionToggleFrames:    make(map[int]*SourceRaster),
		OptionKeyFrames:       make(map[int]*SourceRaster),
		OptionArrowGlyphs:     make(map[int]*SourceRaster),
		GunLibraryStats:       make(map[int]*SourceRaster),
		GunLibraryDrops:       make(map[int]*SourceRaster),
		GunLibraryThumbs:      make(map[int]*SourceRaster),
		CustomWarningLives:    make(map[int]*SourceRaster),
		CustomWarningPlayers:  make(map[int]*SourceRaster),
		CustomSelectionFrames: make(map[int]*SourceRaster),
		StarterGunFrames:      make(map[int]*SourceRaster),
		PerkFrames:            make(map[int]*SourceRaster),
		ModeDisplayFrames:     make(map[int]*SourceRaster),
		ModeCheckFrames:       make(map[int]*SourceRaster),
		MapPreviewImages:      make(map[int]*SourceRaster),
		CampaignThumbs:        make(map[int]*SourceRaster),
		TorsoColors:           make(map[int]*SourceRaster),
		HeadColors:            make(map[int]*SourceRaster),
		playerColorLoaded:     make(map[int]bool),
		ShirtFrames:           make(map[int]*SourceRaster),
		HatFrames:             make(map[int]*SourceRaster),
		HandColors:            make(map[int]*SourceRaster),
		ShieldFrames:          make(map[int]*SourceRaster),
		LegColors:             make(map[int]*SourceRaster),
		ExplosionFX:           make(map[BlastEffectKind]*SourceRaster),
		Shells:                make(map[int]*SourceRaster),
		PlayerArrowFrames:     make(map[int]*SourceRaster),
		BulletTrail:           make(map[int]*SourceRaster),
		InstagibFX:            make(map[int]*SourceRaster),
		PowerupIcons:          make(map[int]*SourceRaster),
		PowerupFlashFrames:    make(map[int]*SourceRaster),
		PowerupNameFrames:     make(map[int]*SourceRaster),
		HUDLastLifeFrames:     make(map[int]*SourceRaster),
		WeaponFrames:          make(map[int]map[int]*SourceRaster),
	}
	// Symbol625 fadeaway is one black 1200x900 rectangle translated through
	// 21 source positions. drawGlobalFade reconstructs it directly, so startup
	// does not decode 21 full-screen PNGs.
	return a
}

func (a *Assets) EnsurePlayerVisuals() {
	if a.playerVisualsLoaded {
		return
	}
	a.Eyes = loadSourceRaster("Symbol 281", 0, "sprites", "DefineSprite_281", "1", "1.png")
	if frames, err := loadChildTransformTimeline("Symbol 239", "Symbol 238"); err == nil {
		a.HeadTimeline = frames
	}
	if frames, err := loadChildTransformTimeline("Symbol 238", "Symbol 237"); err == nil && len(frames) > 0 && frames[0].Valid {
		a.HeadInnerMatrix = frames[0].Matrix
	} else {
		a.HeadInnerMatrix = xflMatrix{A: 1, D: 1, TX: 0.05}
	}
	if frames, err := loadChildTransformTimeline("Symbol 282", "Symbol 188"); err == nil {
		a.Leg1Timeline = frames
	}
	if frames, err := loadChildTransformTimeline("Symbol 189", "Symbol 188"); err == nil {
		a.Leg2Timeline = frames
	}
	a.playerVisualsLoaded = true
}

func (a *Assets) EnsureGameplay() {
	if a.gameplayLoaded {
		return
	}
	a.EnsurePlayerVisuals()
	a.HUDCard = loadSourceRaster("Symbol 1480", 0, "sprites", "DefineSprite_1480", "1.png")
	a.HUDGameOver = loadSourceRaster("Symbol 1453", 0, "sprites", "DefineSprite_1453", "1.png")
	a.HUDLevelUp = loadSourceRaster("Symbol 1455", 0, "sprites", "DefineSprite_1455", "1.png")
	if r, err := renderSolidXFLFrame("Symbol 1459", 0); err == nil {
		a.HUDCardBG = r
	}
	if r, err := renderSolidXFLFrame("Symbol 1461", 0); err == nil {
		a.HUDCardDivider = r
	}
	for frame := 0; frame < 5; frame++ {
		a.ShieldFrames[frame] = loadSourceRaster("Symbol 695", frame, "sprites", "DefineSprite_695", "1", strconv.Itoa(frame+1)+".png")
	}
	if frames, err := loadChildVisualTimeline("Symbol 696", "Symbol 695"); err == nil {
		a.ShieldTimeline = frames
	}
	a.JetpackBase, a.JetpackFuel, a.JetpackFuelMatrix = loadJetpackSourceParts()
	a.JetThrustBase, a.JetThrustFX = loadJetThrustSourceParts()
	a.DropPack = loadSourceRaster("Symbol 172", 0, "sprites", "DefineSprite_172_fx_droppack", "1.png")
	a.Crate = loadSourceRaster("Symbol 668", 0, "sprites", "DefineSprite_668_crate", "1", "1.png")
	a.Grenade = loadSourceRaster("Symbol 665", 0, "sprites", "DefineSprite_665_wep_grenade", "1.png")
	a.DynamiteFX = loadSourceRaster("Symbol 343", 0, "sprites", "DefineSprite_343_fx_dynamite", "1.png")
	a.MuzzleFlash = loadSourceRaster("Symbol 348", 0, "sprites", "DefineSprite_348_fx_flash", "1.png")
	a.InstagibTrail = loadSourceRaster("Symbol 351", 0, "sprites", "DefineSprite_351_fx_instatrail", "1.png")
	if r, err := renderSolidXFLFrame("Symbol 687", 0); err == nil {
		a.InstagibVictim = r
	} else {
		a.InstagibVictim = loadSourceRaster("Symbol 687", 0, "sprites", "DefineSprite_687", "1.png")
	}
	if frames, err := loadChildVisualTimeline("Symbol 688", "Symbol 687"); err == nil {
		a.InstagibVictimTimeline = frames
	}
	a.PowerupAura = loadSourceRaster("Symbol 733", 0, "sprites", "DefineSprite_733", "1.png")
	a.MapWall = loadSourceRaster("Symbol 43", 0, "sprites", "DefineSprite_43_mapfx_wall", "1.png")
	a.MapSnow = loadSourceRaster("Symbol 45", 0, "sprites", "DefineSprite_45", "1.png")
	if hit, err := sourceFrameVisualBounds("Symbol 668", 0); err == nil {
		a.CrateHit = hit
	}
	a.ExplosionFX[BlastEX6] = loadSourceRaster("Symbol 318", 0, "sprites", "DefineSprite_318_fx_ex6", "1.png")
	a.ExplosionFX[BlastEX2] = loadSourceRaster("Symbol 306", 0, "sprites", "DefineSprite_306_fx_ex2", "1.png")
	a.ExplosionFX[BlastEX] = loadSourceRaster("Symbol 308", 0, "sprites", "DefineSprite_308_fx_ex", "1.png")
	a.ExplosionFX[BlastEX4] = loadSourceRaster("Symbol 312", 0, "sprites", "DefineSprite_312_fx_ex4", "1.png")
	a.ExplosionFX[BlastEX5] = loadSourceRaster("Symbol 315", 0, "sprites", "DefineSprite_315_fx_ex5", "1.png")
	a.ExplosionFX[BlastEX3] = loadSourceRaster("Symbol 310", 0, "sprites", "DefineSprite_310_fx_ex3", "1.png")
	a.Shells[1] = loadSourceRaster("Symbol 294", 0, "sprites", "DefineSprite_294_fx_shell", "1.png")
	a.Shells[2] = loadSourceRaster("Symbol 296", 0, "sprites", "DefineSprite_296_fx_shell2", "1.png")
	a.Shells[3] = loadSourceRaster("Symbol 298", 0, "sprites", "DefineSprite_298_fx_shell3", "1.png")
	a.Shells[4] = a.Shells[3]
	a.Shells[5] = loadSourceRaster("Symbol 302", 0, "sprites", "DefineSprite_302_fx_shot", "1.png")
	a.Shells[6] = loadSourceRaster("Symbol 303", 0, "sprites", "DefineSprite_303_fx_shot3", "1.png")
	a.Shells[7] = loadSourceRaster("Symbol 335", 0, "sprites", "DefineSprite_335_fx_dropmag", "1.png")
	a.Shells[8] = loadSourceRaster("Symbol 339", 0, "sprites", "DefineSprite_339_fx_dropmag2", "1.png")
	a.Shells[9] = loadSourceRaster("Symbol 341", 0, "sprites", "DefineSprite_341_fx_dropmag3", "1.png")
	a.Shells[10] = loadSourceRaster("Symbol 300", 0, "sprites", "DefineSprite_300_fx_shell4", "1.png")
	a.Shells[11] = loadSourceRaster("Symbol 357", 0, "sprites", "DefineSprite_357_fx_speedloader", "1.png")
	a.Shells[12] = loadSourceRaster("Symbol 170", 0, "sprites", "DefineSprite_170_fx_deagle", "1.png")
	a.BulletTrail[1] = loadSourceRaster("Symbol 368", 0, "sprites", "DefineSprite_368", "1", "1.png")
	a.BulletTrail[2] = loadSourceRaster("Symbol 368", 1, "sprites", "DefineSprite_368", "1", "2.png")
	if b, err := sourceFrameVisualBounds("Symbol 368", 0); err == nil {
		a.BulletTrailNaturalWidth = b.W
	}
	for frame := 0; frame < 5; frame++ {
		a.InstagibFX[frame] = loadSourceRaster("Symbol 377", frame, "sprites", "DefineSprite_377", "1", strconv.Itoa(frame+1)+".png")
	}
	for kind := 0; kind < 7; kind++ {
		if kind != 4 {
			if icon, err := renderSolidXFLFrame("Symbol 730", kind); err == nil {
				a.PowerupIcons[kind] = icon
				continue
			}
			a.PowerupIcons[kind] = loadSourceRaster("Symbol 730", kind, "sprites", "DefineSprite_730", "1", strconv.Itoa(kind+1)+".png")
		}
	}
	a.PowerupIcons[4] = makeSourcePowerupDoubleIcon()
	if frames, err := loadChildTransformTimeline("Symbol 734", "Symbol 730"); err == nil {
		a.PowerupStuffTimeline = frames
	}
	if frames, err := loadChildTransformTimeline("Symbol 734", "Symbol 733"); err == nil {
		a.PowerupAuraTimeline = frames
	}
	if frames, err := loadChildTransformTimeline("Symbol 734", "Symbol 710"); err == nil {
		a.PowerupFlashTimeline = frames
	}
	if frames, err := loadChildTransformTimelines("Symbol 46", "Symbol 45"); err == nil {
		a.MapSnowTimelines = frames
	}
	a.gameplayLoaded = true
}

// EnsureScene loads only the map currently entering gameplay. The old startup
// path rendered all 13 maps x three XFL layers before even showing Main Menu.
// EnsurePlayerColor reconstructs the selected body palette directly from XFL.
// FFDec flattened Symbol188/111/205/237 so all exported color PNGs are the same.
// Loading only colors that are actually used also keeps startup light.
func (a *Assets) PowerupFlashFrame(frame int) *SourceRaster {
	if frame < 0 || frame >= 72 {
		return nil
	}
	if r, ok := a.PowerupFlashFrames[frame]; ok {
		return r
	}
	r := loadSourceRaster("Symbol 710", frame, "sprites", "DefineSprite_710", "1", strconv.Itoa(frame+1)+".png")
	a.PowerupFlashFrames[frame] = r
	return r
}

// Symbol292 is fx_powerupname. Source passes powerupnumber+1 via `asdf`, then
// its constructor gotoAndStop(asdf+1), so runtime kinds 0..6 use XFL frames 1..7.
func (a *Assets) PowerupNameFrame(kind int) *SourceRaster {
	if kind < 0 || kind > 6 {
		return nil
	}
	if r, ok := a.PowerupNameFrames[kind]; ok {
		return r
	}
	if r, err := renderSolidXFLFrame("Symbol 292", kind+1); err == nil {
		a.PowerupNameFrames[kind] = r
		return r
	}
	return nil
}

// lifebling() creates fx_bling with mod=6; Symbol159 immediately selects
// source frame7 (zero-based frame6), whose text is the green "1-UP".
func (a *Assets) LifeBlingFrame() *SourceRaster {
	if a.LifeBling != nil {
		return a.LifeBling
	}
	if r, err := renderSolidXFLFrame("Symbol 159", 6); err == nil {
		a.LifeBling = r
	}
	return a.LifeBling
}

// Symbol1456 is the exact HUD warning timeline used by Symbol1480.lastlife:
// frames 1..90 = LAST LIFE, 91..180 = GAME OVER, 181..270 = LEVEL UP.
// These frames are DOMStaticText. The generic XFL bounds reducer deliberately
// ignores text, so using loadSourceRaster here would lose the symbol's negative
// registration point and shift the warning down/right. Recover the registration
// from the exact text bounds + Symbol1456 child transform instead.
func (a *Assets) hudLastLifeTransform(frame int) (SourceTransformFrame, bool) {
	if frame < 0 || frame >= 270 {
		return SourceTransformFrame{}, false
	}
	phase := frame / 90
	child := [...]string{"Symbol 1451", "Symbol 1453", "Symbol 1455"}[phase]
	if a.HUDLastLifeTimelines[phase] == nil {
		if timeline, err := loadChildTransformTimeline("Symbol 1456", child); err == nil {
			a.HUDLastLifeTimelines[phase] = timeline
		}
	}
	if frame >= len(a.HUDLastLifeTimelines[phase]) || !a.HUDLastLifeTimelines[phase][frame].Valid {
		return SourceTransformFrame{}, false
	}
	return a.HUDLastLifeTimelines[phase][frame], true
}

func (a *Assets) HUDLastLifeFrame(frame int) *SourceRaster {
	if frame < 0 || frame >= 270 {
		return nil
	}
	if r, ok := a.HUDLastLifeFrames[frame]; ok {
		return r
	}

	phase := frame / 90
	// XFL DOMStaticText width/height values are stored at 10x stage units.
	// Matrices are direct stage units. These are copied from the three source
	// text symbols, including their exact local registration matrices.
	base := [...]Rect{
		{X: -84.3, Y: -39.2, W: 157.8, H: 57.9}, // LAST LIFE
		{X: -95.3, Y: -39.2, W: 165.7, H: 53.2}, // GAME OVER
		{X: -95.3, Y: -39.2, W: 135.6, H: 51.8}, // LEVEL UP
	}[phase]
	state, ok := a.hudLastLifeTransform(frame)
	if !ok {
		return nil
	}
	visible := transformRect(base, state.Matrix)

	img := decodeOriginalPNG("sprites", "DefineSprite_1456", "1", strconv.Itoa(frame+1)+".png")
	if img == nil {
		return nil
	}
	bounds := visible
	if alpha, ok := alphaBounds(img); ok {
		originX1 := visible.X - float64(alpha.Min.X)
		originY1 := visible.Y - float64(alpha.Min.Y)
		originX2 := visible.X + visible.W - float64(alpha.Max.X)
		originY2 := visible.Y + visible.H - float64(alpha.Max.Y)
		bounds.X = (originX1 + originX2) / 2
		bounds.Y = (originY1 + originY2) / 2
	}
	b := img.Bounds()
	bounds.W = float64(b.Dx())
	bounds.H = float64(b.Dy())
	r := &SourceRaster{Image: ebiten.NewImageFromImage(img), Bounds: bounds}
	a.HUDLastLifeFrames[frame] = r
	return r
}

func (a *Assets) ShirtFrame(frame int) *SourceRaster {
	if frame < 0 || frame >= 32 {
		return nil
	}
	if r, ok := a.ShirtFrames[frame]; ok {
		return r
	}
	var r *SourceRaster
	if frame < 15 {
		if xfl, err := renderSolidXFLFrame("Symbol 224", frame); err == nil {
			r = xfl
		}
	}
	if r == nil {
		r = loadSourceRaster("Symbol 224", frame, "sprites", "DefineSprite_224", "1", strconv.Itoa(frame+1)+".png")
	}
	a.ShirtFrames[frame] = r
	return r
}

func (a *Assets) HatFrame(frame int) *SourceRaster {
	if frame < 0 || frame >= 33 {
		return nil
	}
	if r, ok := a.HatFrames[frame]; ok {
		return r
	}
	var r *SourceRaster
	if xfl, err := renderSolidXFLFrame("Symbol 269", frame); err == nil {
		r = xfl
	} else {
		r = loadSourceRaster("Symbol 269", frame, "sprites", "DefineSprite_269", "1", strconv.Itoa(frame+1)+".png")
	}
	a.HatFrames[frame] = r
	return r
}

var sourcePlayerPalette = [...]color.NRGBA{
	{R: 0x99, G: 0x99, B: 0x99, A: 0xff}, // source frame1 / non-selectable fallback
	{R: 0x00, G: 0xcc, B: 0xff, A: 0xff},
	{R: 0x00, G: 0x66, B: 0xff, A: 0xff},
	{R: 0x99, G: 0x66, B: 0xff, A: 0xff},
	{R: 0xff, G: 0x99, B: 0xff, A: 0xff},
	{R: 0xff, G: 0x5e, B: 0x5e, A: 0xff},
	{R: 0xcc, G: 0x00, B: 0x00, A: 0xff},
	{R: 0xff, G: 0xff, B: 0x99, A: 0xff},
	{R: 0xff, G: 0xcc, B: 0x00, A: 0xff},
	{R: 0xcc, G: 0xff, B: 0x00, A: 0xff},
	{R: 0x33, G: 0xcc, B: 0x66, A: 0xff},
}

func loadTintedPlayerPart(symbol, dir string, tint color.NRGBA) *SourceRaster {
	base := loadSourceRaster(symbol, 0, "sprites", dir, "1", "1.png")
	img := decodeOriginalPNG("sprites", dir, "1", "1.png")
	if base == nil || img == nil {
		return base
	}
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r16, g16, b16, a16 := img.At(x, y).RGBA()
			if a16 == 0 {
				continue
			}
			r8, g8, b8 := uint8(r16>>8), uint8(g16>>8), uint8(b16>>8)
			// Source player-color parts are one flat #999999 fill plus black
			// anti-aliased outline. Preserve that coverage/outline and only replace
			// the fill hue; this avoids the bad opaque rectangle produced when the
			// nested head vector is rasterized recursively.
			v := (float64(r8) + float64(g8) + float64(b8)) / (3 * 153.0)
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			out.SetNRGBA(x-b.Min.X, y-b.Min.Y, color.NRGBA{
				R: uint8(float64(tint.R)*v + 0.5),
				G: uint8(float64(tint.G)*v + 0.5),
				B: uint8(float64(tint.B)*v + 0.5),
				A: uint8(a16 >> 8),
			})
		}
	}
	return &SourceRaster{Image: ebiten.NewImageFromImage(out), Bounds: base.Bounds}
}

func loadQualityPlayerPart(symbol, dir string, colorIndex int, tint color.NRGBA) *SourceRaster {
	// HIGH uses the actual colour frame from the XFL vector timeline. FFDec
	// flattened these palettes into identical PNGs, so tinting that old raster
	// can never recover the original vector edge quality.
	if sourceRenderQuality == 3 {
		if r, err := renderSolidXFLFrame(symbol, colorIndex); err == nil && r != nil {
			return r
		}
	}
	return loadTintedPlayerPart(symbol, dir, tint)
}

func (a *Assets) invalidateQualityDependentPlayerRasters() {
	a.TorsoColors = make(map[int]*SourceRaster)
	a.HandColors = make(map[int]*SourceRaster)
	a.LegColors = make(map[int]*SourceRaster)
	a.HeadColors = make(map[int]*SourceRaster)
	a.PlayerArrowFrames = make(map[int]*SourceRaster)
	a.playerColorLoaded = make(map[int]bool)
}

func (a *Assets) EnsurePlayerColor(colorIndex int) {
	if colorIndex < 0 || colorIndex > 10 || a.playerColorLoaded[colorIndex] {
		return
	}
	tint := sourcePlayerPalette[colorIndex]
	a.LegColors[colorIndex] = loadQualityPlayerPart("Symbol 188", "DefineSprite_188", colorIndex, tint)
	a.HandColors[colorIndex] = loadQualityPlayerPart("Symbol 111", "DefineSprite_111", colorIndex, tint)
	a.TorsoColors[colorIndex] = loadQualityPlayerPart("Symbol 205", "DefineSprite_205", colorIndex, tint)
	// The head container carries a nested 0.05 transform. Keep the proven
	// source-coverage recolour here; hats/shirts/body/limbs still gain the
	// vector HIGH path without reintroducing the old opaque-head regression.
	a.HeadColors[colorIndex] = loadTintedPlayerPart("Symbol 237", "DefineSprite_237", tint)
	// player_arrow uses the same playernumber/color index.
	if arrow, err := renderSolidXFLFrame("Symbol 645", colorIndex); err == nil {
		a.PlayerArrowFrames[colorIndex] = arrow
	}
	a.playerColorLoaded[colorIndex] = true
}

func (a *Assets) EnsureInteractions() {
	if a.interactionLoaded {
		return
	}
	a.EnsurePlayerVisuals()
	loadInteractionAssets(a)
	a.interactionLoaded = true
}

func (a *Assets) EnsureScene(n int) {
	if a.sceneLoaded[n] {
		return
	}
	if isGM2MapID(n) {
		source := gm2SourceMapNumber(n)
		if source < 1 || source > 21 {
			return
		}
		frame := strconv.Itoa(source) + ".png"
		a.SceneBack[n] = loadSourceRasterIn("gm2", "Symbol 1642", source-1, "sprites", "DefineSprite_1642", frame)
		a.SceneMid[n] = loadSourceRasterIn("gm2", "Symbol 1690", source-1, "sprites", "DefineSprite_1690", frame)
		a.SceneFront[n] = loadSourceRasterIn("gm2", "Symbol 1835", source-1, "sprites", "DefineSprite_1835", frame)
		a.sceneLoaded[n] = true
		return
	}
	if n < 1 || n > 13 {
		return
	}
	frame := strconv.Itoa(n) + ".png"
	if back, err := renderSolidXFLFrame("Symbol 1352", n-1); err == nil {
		a.SceneBack[n] = back
	} else {
		a.SceneBack[n] = loadSourceRaster("Symbol 1352", n-1, "sprites", "DefineSprite_1352", "1", frame)
	}
	if mid, err := renderSolidXFLFrame("Symbol 1376", n-1); err == nil {
		a.SceneMid[n] = mid
	} else {
		a.SceneMid[n] = loadSourceRaster("Symbol 1376", n-1, "sprites", "DefineSprite_1376", "1", frame)
	}
	if foreground, err := renderSolidXFLFrame("Symbol 1390", n-1); err == nil {
		a.SceneFront[n] = foreground
	} else {
		a.SceneFront[n] = loadSourceRaster("Symbol 1390", n-1, "sprites", "DefineSprite_1390", "1", frame)
	}
	a.sceneLoaded[n] = true
}

func (a *Assets) WeaponFrame(number, frame int) *SourceRaster {
	frames := a.WeaponFrames[number]
	if frames == nil {
		frames = make(map[int]*SourceRaster)
		a.WeaponFrames[number] = frames
	}
	if r, loaded := frames[frame]; loaded {
		return r
	}
	def, ok := WeaponByNumber(number)
	if !ok {
		frames[frame] = nil
		return nil
	}
	libraryName := sourceLibraryNameFromSpriteDir(def.SpriteDir)
	if libraryName == "" {
		frames[frame] = nil
		return nil
	}
	pngFrame := strconv.Itoa(frame+1) + ".png"
	namespace := weaponAssetNamespace(number)
	var r *SourceRaster
	if namespace == "gm2" {
		// The unpacked GM2 FFDec export stores weapon frames directly in the
		// sprite directory, while the curated GM1 runtime tree uses an extra
		// `1/` level. Keep both layouts source-faithful instead of renaming files.
		r = loadSourceRasterIn(namespace, libraryName, frame, "sprites", def.SpriteDir, pngFrame)
	} else {
		r = loadSourceRasterIn(namespace, libraryName, frame, "sprites", def.SpriteDir, "1", pngFrame)
	}
	frames[frame] = r
	return r
}

func loadJetThrustSourceParts() (base, fx *SourceRaster) {
	fx = loadSourceRaster("Symbol 175", 0, "sprites", "DefineSprite_175", "1.png")
	if fx == nil {
		return nil, nil
	}
	img := decodeOriginalPNG("sprites", "DefineSprite_175", "1.png")
	if img == nil {
		return nil, fx
	}
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			// Symbol 176's base square has the same source geometry/stroke as
			// Symbol 175. Only the fill changes from #ff6600 to #ffff00, so for
			// anti-aliased fill/stroke pixels G becomes R while R/B/A stay intact.
			c.G = c.R
			out.SetNRGBA(x, y, c)
		}
	}
	return &SourceRaster{Image: ebiten.NewImageFromImage(out), Bounds: fx.Bounds}, fx
}

func loadJetpackSourceParts() (base, fuel *SourceRaster, child xflMatrix) {
	full := loadSourceRaster("Symbol 193", 0, "sprites", "DefineSprite_193", "1.png")
	fuel = loadSourceRaster("Symbol 192", 0, "sprites", "DefineSprite_192", "1.png")
	if full == nil || fuel == nil {
		return full, fuel, xflMatrix{A: 1, D: 1}
	}
	frames, err := loadChildTransformTimeline("Symbol 193", "Symbol 192")
	if err != nil || len(frames) == 0 || !frames[0].Valid {
		return full, fuel, xflMatrix{A: 1, D: 1}
	}
	child = frames[0].Matrix

	// Symbol 193 layer 1 is the orange fuel (Symbol 192), drawn above the
	// jetpack body. Recover the body pixels under that layer from the original
	// FFDec composite instead of painting an approximation. Where fuel is fully
	// opaque, the hidden backing fill is #666666 from Symbol 190 layer 1.
	fullCPU := decodeOriginalPNG("sprites", "DefineSprite_193", "1.png")
	fuelCPU := decodeOriginalPNG("sprites", "DefineSprite_192", "1.png")
	if fullCPU == nil || fuelCPU == nil || math.Abs(child.A-1) > 1e-9 || math.Abs(child.D-1) > 1e-9 || math.Abs(child.B) > 1e-9 || math.Abs(child.C) > 1e-9 {
		return full, fuel, child
	}

	fb := fullCPU.Bounds()
	baseCPU := image.NewNRGBA(image.Rect(0, 0, fb.Dx(), fb.Dy()))
	for y := 0; y < fb.Dy(); y++ {
		for x := 0; x < fb.Dx(); x++ {
			baseCPU.SetNRGBA(x, y, color.NRGBAModel.Convert(fullCPU.At(fb.Min.X+x, fb.Min.Y+y)).(color.NRGBA))
		}
	}
	fuelBounds := fuelCPU.Bounds()
	for fy := 0; fy < fuelBounds.Dy(); fy++ {
		for fx := 0; fx < fuelBounds.Dx(); fx++ {
			fc := color.NRGBAModel.Convert(fuelCPU.At(fuelBounds.Min.X+fx, fuelBounds.Min.Y+fy)).(color.NRGBA)
			if fc.A == 0 {
				continue
			}
			lx := fuel.Bounds.X + float64(fx)
			ly := fuel.Bounds.Y + float64(fy)
			pxLocal := child.A*lx + child.C*ly + child.TX
			pyLocal := child.B*lx + child.D*ly + child.TY
			px := int(math.Round(pxLocal - full.Bounds.X))
			py := int(math.Round(pyLocal - full.Bounds.Y))
			if px < 0 || py < 0 || px >= baseCPU.Bounds().Dx() || py >= baseCPU.Bounds().Dy() {
				continue
			}
			cc := baseCPU.NRGBAAt(px, py)
			baseCPU.SetNRGBA(px, py, recoverJetpackBackingPixel(cc, fc))
		}
	}
	base = &SourceRaster{Image: ebiten.NewImageFromImage(baseCPU), Bounds: full.Bounds}
	return base, fuel, child
}

func recoverJetpackBackingPixel(composite, foreground color.NRGBA) color.NRGBA {
	af := float64(foreground.A) / 255
	ac := float64(composite.A) / 255
	if af >= 1-1e-9 {
		// Hidden backing color comes directly from Symbol 190, layer 1.
		return color.NRGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xff}
	}
	ab := (ac - af) / (1 - af)
	if ab <= 1e-9 {
		return color.NRGBA{}
	}
	if ab > 1 {
		ab = 1
	}
	den := ab * (1 - af)
	recover := func(c, f uint8) uint8 {
		v := ((float64(c)/255)*ac - (float64(f)/255)*af) / den
		v = math.Max(0, math.Min(1, v))
		return uint8(math.Round(v * 255))
	}
	return color.NRGBA{
		R: recover(composite.R, foreground.R),
		G: recover(composite.G, foreground.G),
		B: recover(composite.B, foreground.B),
		A: uint8(math.Round(ab * 255)),
	}
}

func decodeOriginalPNG(parts ...string) image.Image {
	return decodeOriginalPNGIn("", parts...)
}

func decodeOriginalPNGIn(namespace string, parts ...string) image.Image {
	path, err := findOriginalPathIn(namespace, parts...)
	if err != nil {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil
	}
	return img
}
