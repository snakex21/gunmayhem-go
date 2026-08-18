package game

import (
	"image"
	"image/draw"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
)

// loadInteractionAssets loads the source UI/movie-clip timelines that are
// controlled directly by root/menu ActionScript. Keeping them as source frames
// avoids replacing Flash interactions with invented Ebiten widgets.
func loadLibraryBitmapSource(filename string) *SourceRaster {
	img := decodeOriginalPNG("fla", "LIBRARY", filename)
	if img == nil {
		return nil
	}
	b := img.Bounds()
	return &SourceRaster{
		Image:  ebiten.NewImageFromImage(img),
		Bounds: Rect{X: 0, Y: 0, W: float64(b.Dx()), H: float64(b.Dy())},
	}
}

// loadTightOriginalPNG is intentionally UI-only. Gameplay needs the original
// Flash registration canvas, while a menu thumbnail should fit the visible
// artwork, not the transparent registration padding around it.
func loadTightOriginalPNG(parts ...string) *SourceRaster {
	return loadTightOriginalPNGIn("", parts...)
}

func loadTightOriginalPNGIn(namespace string, parts ...string) *SourceRaster {
	img := decodeOriginalPNGIn(namespace, parts...)
	if img == nil {
		return nil
	}
	alpha, ok := alphaBounds(img)
	if !ok || alpha.Empty() {
		return nil
	}
	out := image.NewNRGBA(image.Rect(0, 0, alpha.Dx(), alpha.Dy()))
	draw.Draw(out, out.Bounds(), img, alpha.Min, draw.Src)
	return &SourceRaster{
		Image:  ebiten.NewImageFromImage(out),
		Bounds: Rect{X: 0, Y: 0, W: float64(alpha.Dx()), H: float64(alpha.Dy())},
	}
}

func loadInteractionAssets(a *Assets) {
	// Secondary menu backdrops are not needed to show Main Menu. Load them only
	// once the first source fade is already covering the screen.
	a.CustomMenu = loadSourceRaster("Symbol 1309", 0, "sprites", "DefineSprite_1309", "1.png")
	a.MenuChosen = loadSourceRaster("Symbol 1199", 0, "sprites", "DefineSprite_1199", "1.png")
	if a.MenuChosen == nil {
		if r, err := renderSolidXFLFrame("Symbol 1199", 0); err == nil {
			a.MenuChosen = r
		}
	}
	a.OptionsMenu = loadSourceRaster("Symbol 991", 0, "sprites", "DefineSprite_991", "1.png")
	if a.OptionsMenu == nil {
		if r, err := renderSolidXFLFrame("Symbol 991", 0); err == nil {
			a.OptionsMenu = r
		}
	}
	a.CreditsMenu = loadSourceRaster("Symbol 1046", 0, "sprites", "DefineSprite_1046", "1.png")
	if a.CreditsMenu == nil {
		if r, err := renderSolidXFLFrame("Symbol 1046", 0); err == nil {
			a.CreditsMenu = r
		}
	}
	if base, err := renderSolidXFLFrame("Symbol 790", 0); err == nil {
		a.CampaignBase = base
	}
	a.CampaignSlider = loadSourceRaster("Symbol 96", 0, "sprites", "DefineSprite_96_menuc_slider", "1.png")
	a.CampaignBack = loadSourceRaster("Symbol 792", 0, "sprites", "DefineSprite_792", "1.png")
	a.CampaignPlayerCard = loadSourceRaster("Symbol 857", 1, "sprites", "DefineSprite_857", "1", "2.png")
	a.CampaignInfoPanel = loadSourceRaster("Symbol 904", 1, "sprites", "DefineSprite_904", "1", "2.png")
	a.CampaignStartButton = loadSourceRaster("Symbol 902", 0, "sprites", "DefineSprite_902", "1.png")
	if base, err := renderSolidXFLFrame("Symbol 47", 0); err == nil {
		a.CampaignLevelBase = base
	}
	for frame := 1; frame <= 10; frame++ {
		a.CampaignUnlock[frame] = loadSourceRaster("Symbol 947", frame-1, "sprites", "DefineSprite_947", "1", strconv.Itoa(frame)+".png")
		a.CampaignInfoFrames[frame] = loadSourceRaster("Symbol 900", frame-1, "sprites", "DefineSprite_900", "1", strconv.Itoa(frame)+".png")
		a.CampaignLevelIcons[frame] = loadSourceRaster("Symbol 90", frame-1, "sprites", "DefineSprite_90", "1", strconv.Itoa(frame)+".png")
	}
	for frame := 1; frame <= 3; frame++ {
		a.CampaignLevelLocks[frame] = loadSourceRaster("Symbol 94", frame-1, "sprites", "DefineSprite_94", "1", strconv.Itoa(frame)+".png")
	}

	a.PauseMenu = loadSourceRaster("Symbol 633", 0, "sprites", "DefineSprite_633_game_pause", "1.png")
	// FFDec exports all 51 Symbol1487 frames as the same frozen digit "3".
	// Reconstruct the source 3->2->1 animation from the three nested XFL
	// timelines instead; each frame preserves its exact translation and alpha.
	for digit, symbol := range map[int]string{3: "Symbol 1482", 2: "Symbol 1484", 1: "Symbol 1486"} {
		if timeline, err := loadChildTransformTimeline("Symbol 1487", symbol); err == nil {
			a.WinCountdownTimelines[digit] = timeline
		}
	}
	for frame := 1; frame <= 41; frame++ {
		a.WinnerFrames[frame] = loadSourceRaster("Symbol 654", frame-1, "sprites", "DefineSprite_654_player_winner", "1", strconv.Itoa(frame)+".png")
	}
	for frame := 1; frame <= 20; frame++ {
		a.CampaignLoseFrames[frame] = loadSourceRaster("Symbol 331", frame-1, "sprites", "DefineSprite_331_fx_campaignlose", "1", strconv.Itoa(frame)+".png")
		a.TeamWinFrames[frame] = loadSourceRaster("Symbol 361", frame-1, "sprites", "DefineSprite_361_fx_teamwin", "1", strconv.Itoa(frame)+".png")
		a.ZombieWaveFrames[frame] = loadSourceRaster("Symbol 365", frame-1, "sprites", "DefineSprite_365_fx_zombiewaveup", "1", strconv.Itoa(frame)+".png")
	}
	// Keep the flattened Symbol1019 only as a fallback. It contains authoring
	// placeholders from all four Symbol1016 panels (Cool Dude / 89), so the
	// normal renderer reconstructs the post-game screen from clean XFL pieces.
	a.PostGameMenu = loadSourceRaster("Symbol 1019", 0, "sprites", "DefineSprite_1019", "1.png")
	if bg, err := renderSolidXFLFrame("Symbol 992", 0); err == nil {
		a.PostGameBackground = bg
	}
	if base, err := renderSolidXFLFrameShapesOnly("Symbol 1016", 0); err == nil {
		a.PostGamePanelBase = base
	}
	if button, err := renderSolidXFLFrameShapesOnly("Symbol 792", 0); err == nil {
		a.PostGameButton = button
	}
	if winner, err := renderSolidXFLFrameShapesOnly("Symbol 1018", 0); err == nil {
		a.PostGameWinnerBorder = winner
	}
	for frame := 1; frame <= 3; frame++ {
		// Symbol982 is a runtime three-state button (idle/hover/selected).
		// FFDec freezes these simple frames inconsistently, so use the XFL
		// shapes directly: #666666, #797979, #FF8000.
		if toggle, err := renderSolidXFLFrame("Symbol 982", frame-1); err == nil {
			a.OptionToggleFrames[frame] = toggle
		} else {
			a.OptionToggleFrames[frame] = loadSourceRaster("Symbol 982", frame-1, "sprites", "DefineSprite_982", "1", strconv.Itoa(frame)+".png")
		}
		if keyFrame, err := renderSolidXFLFrame("Symbol 959", frame-1); err == nil {
			a.OptionKeyFrames[frame] = keyFrame
		}
	}
	if panel, err := renderSolidXFLFrame("Symbol 950", 0); err == nil {
		a.OptionPanelBase = panel
	}
	for vk, symbol := range map[int]string{38: "Symbol 961", 37: "Symbol 962", 40: "Symbol 963", 39: "Symbol 964"} {
		if glyph, err := renderSolidXFLFrame(symbol, 0); err == nil {
			a.OptionArrowGlyphs[vk] = glyph
		}
	}
	if lockup, err := renderSolidXFLFrame("Symbol 988", 0); err == nil {
		a.OptionLockupBase = lockup
	}
	// Keep the flattened Symbol990 only as a fallback. FFDec bakes unrelated
	// dynamic key text into it, so the runtime renderer prefers Symbol988 + text.
	a.OptionLockup = loadSourceRaster("Symbol 990", 0, "sprites", "DefineSprite_990", "1.png")
	if base, err := renderSolidXFLFrame("Symbol 1417", 0); err == nil {
		a.TestPickupBase = base
	}
	if overlay, err := renderSolidXFLFrameShapesOnly("Symbol 913", 1); err == nil {
		a.GunLibraryLockedOverlay = overlay
	}
	a.CustomPlayerCard = loadSourceRaster("Symbol 1302", 2, "sprites", "DefineSprite_1302", "1", "3.png")
	// Symbol806 constructor immediately gotoAndStop(2).
	a.CustomEditButton = loadSourceRaster("Symbol 806", 1, "sprites", "DefineSprite_806", "1", "2.png")
	// Symbol855 is the actual gray COLOR cover nested inside Symbol856. Runtime
	// fades only this child away; loading the whole Symbol856 here would bake the
	// palette into the mask a second time.
	a.PlayerColorMask = loadSourceRaster("Symbol 855", 0, "sprites", "DefineSprite_855", "1.png")
	for frame := 1; frame <= 5; frame++ {
		a.CustomSelectionFrames[frame] = loadSourceRaster("Symbol 835", frame-1, "sprites", "DefineSprite_835", "1", strconv.Itoa(frame)+".png")
	}
	for number := 1; number <= 6; number++ {
		if def, ok := WeaponByNumber(number); ok {
			a.StarterGunFrames[number] = loadTightOriginalPNG("sprites", def.SpriteDir, "1", "1.png")
		}
	}
	for frame := 1; frame <= 10; frame++ {
		a.PerkFrames[frame] = loadSourceRaster("Symbol 830", frame-1, "sprites", "DefineSprite_830", "1", strconv.Itoa(frame)+".png")
	}
	for mode := SourceGameModeNormal; mode <= SourceGameModeSurvival; mode++ {
		frame := 7 - mode // Flash gotoAndStop(totalframes-mode+1), totalframes=6
		a.ModeDisplayFrames[mode] = loadSourceRaster("Symbol 1273", frame-1, "sprites", "DefineSprite_1273", "1", strconv.Itoa(frame)+".png")
	}

	// FFDec repeats the first raster for every frame of Symbol1230/90/94.
	// Pull the actual source bitmaps from the XFL library instead. Map numbers
	// are the root mapnumber values used by frame10 and Symbol1200.
	mapBitmaps := map[int]string{
		1:  "Bitmap 83.png",   // No Name
		2:  "Bitmap 75.png",   // Dessert Duel
		3:  "Bitmap 79.png",   // Underwater Slaughter
		4:  "Bitmap 1210.png", // Solar Shootout
		5:  "Bitmap 67.png",   // Great Wall Brawl
		6:  "Bitmap 1215.png", // Magic Mushroom Mountain Melee
		7:  "Bitmap 87.png",   // Desert Destruction
		8:  "Bitmap 63.png",   // Hovering Houses
		9:  "Bitmap 1222.png", // Midnight Wood
		10: "Bitmap 59.png",   // Polar Pwnage
		11: "Bitmap 71.png",   // Grim City
		12: "Bitmap 55.png",   // Safari Showdown
	}
	for n, filename := range mapBitmaps {
		a.MapPreviewImages[n] = loadLibraryBitmapSource(filename)
	}
	campaignBitmaps := []string{
		"Bitmap 51.png", "Bitmap 55.png", "Bitmap 59.png", "Bitmap 63.png", "Bitmap 67.png",
		"Bitmap 71.png", "Bitmap 75.png", "Bitmap 79.png", "Bitmap 83.png", "Bitmap 87.png",
	}
	for i, filename := range campaignBitmaps {
		a.CampaignThumbs[i+1] = loadLibraryBitmapSource(filename)
	}
	if r, err := renderSolidXFLFrame("Symbol 91", 0); err == nil {
		a.CampaignLockOverlay = r
	}
	if r, err := renderSolidXFLFrame("Symbol 94", 2); err == nil {
		a.CampaignDoneOverlay = r
	}
	for frame := 1; frame <= 2; frame++ {
		a.ModeCheckFrames[frame] = loadSourceRaster("Symbol 1265", frame-1, "sprites", "DefineSprite_1265", "1", strconv.Itoa(frame)+".png")
	}
	for frame := 1; frame <= 15; frame++ {
		a.CustomWarningLives[frame] = loadSourceRaster("Symbol 1305", frame-1, "sprites", "DefineSprite_1305", "1", strconv.Itoa(frame)+".png")
	}
	for frame := 1; frame <= 25; frame++ {
		a.CustomWarningPlayers[frame] = loadSourceRaster("Symbol 1308", frame-1, "sprites", "DefineSprite_1308", "1", strconv.Itoa(frame)+".png")
	}
}

// EnsureGunLibrary defers the expensive 64 XFL stat renders and weapon-thumb
// decoding until the player actually opens Gun Library. Symbol595 wrappers are
// intentionally not loaded anymore: the current renderer uses clean gameplay
// sprites and the old GunLibraryDrops cache had become dead startup work.
func (a *Assets) EnsureGunLibrary() {
	if a.gunLibraryLoaded {
		return
	}
	a.GunLibraryMenu = loadSourceRaster("Symbol 1198", 0, "sprites", "DefineSprite_1198", "1.png")
	if a.GunLibraryMenu == nil {
		if r, err := renderSolidXFLFrame("Symbol 1198", 0); err == nil {
			a.GunLibraryMenu = r
		}
	}
	gunLibraryFrames := append([]int{7}, integerRange(1, 6)...)
	gunLibraryFrames = append(gunLibraryFrames, integerRange(10, 86)...)
	for _, frame := range gunLibraryFrames {
		if frame <= 66 {
			if stat, err := renderSolidXFLFrame("Symbol 1190", frame-1); err == nil {
				a.GunLibraryStats[frame] = stat
			} else {
				a.GunLibraryStats[frame] = loadSourceRaster("Symbol 1190", frame-1, "sprites", "DefineSprite_1190", "1", strconv.Itoa(frame)+".png")
			}
		}
		if def, ok := WeaponByNumber(frame); ok {
			if frame >= 67 {
				a.GunLibraryThumbs[frame] = loadTightOriginalPNGIn("gm2", "sprites", def.SpriteDir, "1.png")
			} else {
				a.GunLibraryThumbs[frame] = loadTightOriginalPNG("sprites", def.SpriteDir, "1", "1.png")
			}
		}
	}
	a.gunLibraryLoaded = true
}

func integerRange(first, last int) []int {
	out := make([]int, 0, last-first+1)
	for i := first; i <= last; i++ {
		out = append(out, i)
	}
	return out
}
