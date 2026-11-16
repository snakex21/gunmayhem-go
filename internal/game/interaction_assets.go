package game

import (
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

// GM2 stores its map-preview media as JPEGs under images/, unlike GM1's PNG
// library bitmaps. Decode the original media directly so the developed map
// picker can use the exact source previews instead of reconstructing whole
// scenes just to make a thumbnail.
func loadOriginalImageSourceIn(namespace string, parts ...string) *SourceRaster {
	path, err := findOriginalPathIn(namespace, parts...)
	if err != nil {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var img image.Image
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(f)
	case ".png":
		img, err = png.Decode(f)
	default:
		return nil
	}
	if err != nil || img == nil {
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
	// Secondary-screen art is not needed to show Main Menu. Load it only once
	// the first source fade is already covering the screen. The menus are
	// Mayhem UI kit screens; the Flash menu rasters they replaced are no longer
	// loaded (the credits raster stays for its two doodles).
	a.CreditsMenu = loadSourceRaster("Symbol 1046", 0, "sprites", "DefineSprite_1046", "1.png")
	if a.CreditsMenu == nil {
		if r, err := renderSolidXFLFrame("Symbol 1046", 0); err == nil {
			a.CreditsMenu = r
		}
	}
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
	if base, err := renderSolidXFLFrame("Symbol 1417", 0); err == nil {
		a.TestPickupBase = base
	}
	for number := 1; number <= 6; number++ {
		if def, ok := WeaponByNumber(number); ok {
			a.StarterGunFrames[number] = loadTightOriginalPNG("sprites", def.SpriteDir, "1", "1.png")
		}
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

	// GM2 Symbol 1236 frames 1..9 use these exact bitmap fills for the nine
	// new arenas. Store them under the runtime GM2 map IDs in the same preview
	// table as GM1, so every menu path can treat both games identically.
	gm2PreviewFiles := []string{
		"1170.jpg", // Castle
		"1174.jpg", // Highway
		"1177.jpg", // Sub Base
		"1180.jpg", // Alien Planet
		"1183.jpg", // Venice
		"1186.jpg", // Avalon
		"1189.jpg", // Space Station
		"1192.jpg", // Ski Lift
		"1195.jpg", // Jungle
	}
	for sourceNumber, filename := range gm2PreviewFiles {
		a.MapPreviewImages[gm2MapID(sourceNumber+1)] = loadOriginalImageSourceIn("gm2", "images", filename)
	}

	campaignBitmaps := []string{
		"Bitmap 51.png", "Bitmap 55.png", "Bitmap 59.png", "Bitmap 63.png", "Bitmap 67.png",
		"Bitmap 71.png", "Bitmap 75.png", "Bitmap 79.png", "Bitmap 83.png", "Bitmap 87.png",
	}
	for i, filename := range campaignBitmaps {
		a.CampaignThumbs[i+1] = loadLibraryBitmapSource(filename)
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
	gunLibraryFrames := append([]int{7}, integerRange(1, 6)...)
	gunLibraryFrames = append(gunLibraryFrames, integerRange(10, 86)...)
	for _, frame := range gunLibraryFrames {
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
