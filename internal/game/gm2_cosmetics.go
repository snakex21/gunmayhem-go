package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	gm1SelectableColorCount = 10
	gm1SelectableShirtCount = 12
	gm1SelectableHatCount   = 24
	gm2ColorCount           = 20
	gm2ShirtCount           = 27
	gm2HatCount             = 48
	gm2EyesCount            = 12
	gm1SelectableEyesCount  = 1
)

// The combined editor keeps every GM1 item addressable and appends the full
// GM2 source set behind it. This avoids ID collisions: GM2 color 1 is virtual
// color 11, GM2 shirt 1 is virtual shirt 13, and GM2 hat 1 is virtual hat 25.
// GM2 campaign enemies still carry raw source IDs and force UseGM2Cosmetics.
func gm2ColorSourceNumber(value int) (int, bool) {
	if value <= gm1SelectableColorCount || value > gm1SelectableColorCount+gm2ColorCount {
		return 0, false
	}
	return value - gm1SelectableColorCount, true
}

func gm2ShirtSourceNumber(value int) (int, bool) {
	if value <= gm1SelectableShirtCount || value > gm1SelectableShirtCount+gm2ShirtCount {
		return 0, false
	}
	return value - gm1SelectableShirtCount, true
}

func gm2HatSourceNumber(value int) (int, bool) {
	if value <= gm1SelectableHatCount || value > gm1SelectableHatCount+gm2HatCount {
		return 0, false
	}
	return value - gm1SelectableHatCount, true
}

func gm2EyeSourceNumber(value int) (int, bool) {
	if value <= gm1SelectableEyesCount || value > gm1SelectableEyesCount+gm2EyesCount {
		return 0, false
	}
	return value - gm1SelectableEyesCount, true
}

func maxDevelopedColorValue() int { return gm1SelectableColorCount + gm2ColorCount }
func maxDevelopedShirtValue() int { return gm1SelectableShirtCount + gm2ShirtCount }
func maxDevelopedHatValue() int   { return gm1SelectableHatCount + gm2HatCount }
func maxDevelopedEyeValue() int   { return gm1SelectableEyesCount + gm2EyesCount }

func developedPlayerColorSwatch(value int) color.NRGBA {
	if value >= 1 && value <= gm1SelectableColorCount {
		return sourcePlayerPalette[value]
	}
	if n, ok := gm2ColorSourceNumber(value); ok && n > 0 && n < len(gm2PlayerPalette) {
		return gm2PlayerPalette[n]
	}
	return color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}
}

func resolvedPlayerColor(value int, forceGM2 bool) (source int, gm2 bool) {
	if forceGM2 {
		return value, true
	}
	if n, ok := gm2ColorSourceNumber(value); ok {
		return n, true
	}
	return value, false
}

func resolvedPlayerShirt(value int, forceGM2 bool) (source int, gm2 bool) {
	if forceGM2 {
		return value, true
	}
	if n, ok := gm2ShirtSourceNumber(value); ok {
		return n, true
	}
	return value, false
}

func resolvedPlayerHat(value int, forceGM2 bool) (source int, gm2 bool) {
	if forceGM2 {
		return value, true
	}
	if n, ok := gm2HatSourceNumber(value); ok {
		return n, true
	}
	return value, false
}

func resolvedPlayerEyes(value int, forceGM2 bool) (source int, gm2 bool) {
	if forceGM2 {
		return value, true
	}
	if n, ok := gm2EyeSourceNumber(value); ok {
		return n, true
	}
	return value, false
}

// Exact GM2 body palette from Symbol 148 frames 1..20.
var gm2PlayerPalette = [...]color.NRGBA{
	{},
	{R: 0xff, G: 0x99, B: 0xcc, A: 0xff},
	{R: 0xff, G: 0x00, B: 0xff, A: 0xff},
	{R: 0x99, G: 0x00, B: 0xcc, A: 0xff},
	{R: 0xff, G: 0x5e, B: 0x5e, A: 0xff},
	{R: 0x99, G: 0x00, B: 0x00, A: 0xff},
	{R: 0xff, G: 0x66, B: 0x00, A: 0xff},
	{R: 0x66, G: 0x33, B: 0x00, A: 0xff},
	{R: 0x99, G: 0x66, B: 0x00, A: 0xff},
	{R: 0xff, G: 0xcc, B: 0x00, A: 0xff},
	{R: 0xff, G: 0x99, B: 0x00, A: 0xff},
	{R: 0x66, G: 0x66, B: 0x66, A: 0xff},
	{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff},
	{R: 0x66, G: 0xff, B: 0xff, A: 0xff},
	{R: 0x00, G: 0xcc, B: 0xcc, A: 0xff},
	{R: 0x00, G: 0x66, B: 0xff, A: 0xff},
	{R: 0x00, G: 0x66, B: 0x66, A: 0xff},
	{R: 0x00, G: 0x99, B: 0x00, A: 0xff},
	{R: 0x33, G: 0xcc, B: 0x66, A: 0xff},
	{R: 0x00, G: 0xff, B: 0x00, A: 0xff},
	{R: 0xcc, G: 0xff, B: 0x00, A: 0xff},
}

// The developed cosmetic editor is only exposed when every source family it
// needs is present. This prevents offering an item that can be selected but not
// rendered in a packaged build.
func gm2CosmeticsAvailable() bool {
	checks := [][]string{
		{"fla", "LIBRARY", "Symbol 148.xml"},
		{"fla", "LIBRARY", "Symbol 201.xml"},
		{"fla", "LIBRARY", "Symbol 279.xml"},
		{"fla", "LIBRARY", "Symbol 291.xml"},
		{"sprites", "DefineSprite_148", "20.png"},
		{"sprites", "DefineSprite_201", "27.png"},
		{"sprites", "DefineSprite_279", "48.png"},
		{"sprites", "DefineSprite_291", "12.png"},
	}
	for _, parts := range checks {
		if _, err := findOriginalPathIn("gm2", parts...); err != nil {
			return false
		}
	}
	return true
}

func gm2FaceSelectionItems() []customSelectionItem {
	const (
		cols   = 3
		cellW  = 54.0
		cellH  = 50.0
		stepX  = 61.0
		stepY  = 55.0
		startX = 12.0
		startY = 445.0
	)
	count := maxDevelopedEyeValue()
	out := make([]customSelectionItem, 0, count)
	for value := 1; value <= count; value++ {
		i := value - 1
		// Value 1 remains the original GM1 face; GM2 source faces 1..12 are
		// exposed as virtual values 2..13 so none of the source art collides.
		out = append(out, customSelectionItem{Editor: customEditEyes, Number: value, Rect: Rect{
			X: startX + float64(i%cols)*stepX, Y: startY + float64(i/cols)*stepY, W: cellW, H: cellH,
		}})
	}
	return out
}

func developedColorSelectionItems() []customSelectionItem {
	const (
		cols   = 5
		cellW  = 32.0
		cellH  = 32.0
		stepX  = 37.0
		stepY  = 43.0
		startX = 6.0
		startY = 430.0
	)
	count := gm1SelectableColorCount
	if gm2CosmeticsAvailable() {
		count = maxDevelopedColorValue()
	}
	out := make([]customSelectionItem, 0, count)
	for n := 1; n <= count; n++ {
		i := n - 1
		out = append(out, customSelectionItem{Editor: customEditColor, Number: n, Rect: Rect{
			X: startX + float64(i%cols)*stepX, Y: startY + float64(i/cols)*stepY, W: cellW, H: cellH,
		}})
	}
	return out
}

func developedShirtSelectionItems() []customSelectionItem {
	if !gm2CosmeticsAvailable() {
		return sourceCustomSelectionItems()[customEditShirt]
	}
	const (
		cols   = 5
		cellW  = 32.0
		cellH  = 30.0
		stepX  = 37.0
		stepY  = 36.0
		startX = 6.0
		startY = 425.0
	)
	count := maxDevelopedShirtValue()
	out := make([]customSelectionItem, 0, count)
	for n := 1; n <= count; n++ {
		i := n - 1
		out = append(out, customSelectionItem{Editor: customEditShirt, Number: n, Rect: Rect{
			X: startX + float64(i%cols)*stepX, Y: startY + float64(i/cols)*stepY, W: cellW, H: cellH,
		}})
	}
	return out
}

func developedHatSelectionItems(editor int) []customSelectionItem {
	if !gm2CosmeticsAvailable() {
		return sourceCustomSelectionItems()[editor]
	}
	const (
		cols   = 6
		cellW  = 27.0
		cellH  = 34.0
		stepX  = 31.5
		stepY  = 40.0
		startX = 6.0
		startY = 425.0
		page   = 36
	)
	start := 1
	if editor == customEditHat2 {
		start = page + 1
	}
	end := start + page - 1
	if max := maxDevelopedHatValue(); end > max {
		end = max
	}
	out := make([]customSelectionItem, 0, end-start+1)
	for n := start; n <= end; n++ {
		i := n - start
		out = append(out, customSelectionItem{Editor: editor, Number: n, Rect: Rect{
			X: startX + float64(i%cols)*stepX, Y: startY + float64(i/cols)*stepY, W: cellW, H: cellH,
		}})
	}
	return out
}

func developedHandgunSelectionItems() []customSelectionItem {
	const (
		cols   = 4
		cellW  = 41.0
		cellH  = 34.0
		stepX  = 47.0
		stepY  = 38.0
		startX = 6.0
		startY = 425.0
	)
	handguns := starterHandgunNumbers()
	out := make([]customSelectionItem, 0, len(handguns))
	for i, number := range handguns {
		out = append(out, customSelectionItem{Editor: customEditGun, Number: number, Rect: Rect{
			X: startX + float64(i%cols)*stepX, Y: startY + float64(i/cols)*stepY, W: cellW, H: cellH,
		}})
	}
	return out
}

func developedThrowableSelectionItems() []customSelectionItem {
	const (
		cols   = 2
		cellW  = 84.0
		cellH  = 72.0
		stepX  = 92.0
		stepY  = 86.0
		startX = 10.0
		startY = 430.0
	)
	out := make([]customSelectionItem, 0, throwableTypeCount)
	for kind := 0; kind < throwableTypeCount; kind++ {
		out = append(out, customSelectionItem{Editor: customEditThrowable, Number: kind + 1, Rect: Rect{
			X: startX + float64(kind%cols)*stepX, Y: startY + float64(kind/cols)*stepY, W: cellW, H: cellH,
		}})
	}
	return out
}

func developedCustomSelectionItems(editor int) []customSelectionItem {
	switch editor {
	case customEditPerk:
		return developedPerkSelectionItems()
	case customEditColor:
		return developedColorSelectionItems()
	case customEditShirt:
		return developedShirtSelectionItems()
	case customEditHat1, customEditHat2:
		return developedHatSelectionItems(editor)
	case customEditGun:
		return developedHandgunSelectionItems()
	case customEditThrowable:
		return developedThrowableSelectionItems()
	case customEditEyes:
		if gm2CosmeticsAvailable() {
			return gm2FaceSelectionItems()
		}
	}
	return sourceCustomSelectionItems()[editor]
}

func drawGM2FaceGridLabel(dst *ebiten.Image, localY, alpha float64) {
	drawSourceMenuText(dst, "GM1 + GM2 FACES", menuFontCondensedExtraBold, 15,
		color.NRGBA{A: uint8(255 * alpha)}, 36, localY+420)
}
