package game

import (
	"image/color"
	"math"
	"os"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	ebitentext "github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

type sourceMenuFontKind int

const (
	menuFontCondensed sourceMenuFontKind = iota
	menuFontCondensedExtraBold
	menuFontTwCen
	menuFontArial
	menuFontArialBold
	menuFontCenturyGothicBold
	menuFontShowcardGothic
	// Mayhem UI faces: the embedded OFL Barlow Condensed family (ui_font.go).
	menuFontUIBold
	menuFontUI
)

type sourceMenuFontCache struct {
	once  sync.Once
	font  *opentype.Font
	err   error
	mu    sync.Mutex
	faces map[int]font.Face
	// substitute is set when the face is the embedded Barlow Condensed
	// standing in for a Flash-era Tw Cen/Showcard/Century face. The legacy
	// layouts were positioned with Tw Cen metrics, so a substitute is drawn
	// slightly smaller and with a Tw Cen-like baseline (see sourceMenuFace).
	substitute bool
}

var menuFontCaches = map[sourceMenuFontKind]*sourceMenuFontCache{
	menuFontCondensed:          {faces: map[int]font.Face{}},
	menuFontCondensedExtraBold: {faces: map[int]font.Face{}},
	menuFontTwCen:              {faces: map[int]font.Face{}},
	menuFontArial:              {faces: map[int]font.Face{}},
	menuFontArialBold:          {faces: map[int]font.Face{}},
	menuFontCenturyGothicBold:  {faces: map[int]font.Face{}},
	menuFontShowcardGothic:     {faces: map[int]font.Face{}},
	menuFontUIBold:             {faces: map[int]font.Face{}},
	menuFontUI:                 {faces: map[int]font.Face{}},
}

const (
	// Barlow Condensed caps are ~6% taller than Tw Cen MT Condensed caps at
	// the same point size; scaling keeps legacy labels inside their boxes.
	substituteFontScale = 0.94
	// Tw Cen MT ascent is 0.85 em; Barlow's is 1.0 em. Legacy code passes the
	// top of the Tw Cen line box, so substitutes use this baseline instead.
	substituteBaselineFactor = 0.86
)

func loadSourceMenuFont(kind sourceMenuFontKind, cache *sourceMenuFontCache) {
	var candidates []string
	switch kind {
	case menuFontUIBold:
		cache.font, cache.err = embeddedUIFont(true)
		return
	case menuFontUI:
		cache.font, cache.err = embeddedUIFont(false)
		return
	case menuFontCondensedExtraBold:
		// B4: the FFDec export of Tw Cen MT Condensed Extra Bold is a malformed
		// subset and the system copy (TCCEB.TTF) only ships with Office, so this
		// face silently vanished on clean Windows installs. It is always the
		// embedded OFL Barlow Condensed Bold now.
		cache.font, cache.err = embeddedUIFont(true)
		cache.substitute = cache.err == nil
		return
	case menuFontCondensed:
		if p, err := findOriginalPath("fonts", "49_Tw Cen MT Condensed.ttf"); err == nil {
			candidates = append(candidates, p)
		}
		candidates = append(candidates, `C:\Windows\Fonts\TCCM____.TTF`, `C:\Windows\Fonts\TCCB____.TTF`)
	case menuFontTwCen:
		if p, err := findOriginalPath("fonts", "10_Tw Cen MT.ttf"); err == nil {
			candidates = append(candidates, p)
		}
		candidates = append(candidates, `C:\Windows\Fonts\TCM_____.TTF`)
	case menuFontArial:
		candidates = append(candidates, `C:\Windows\Fonts\arial.ttf`)
	case menuFontArialBold:
		candidates = append(candidates, `C:\Windows\Fonts\arialbd.ttf`)
	case menuFontCenturyGothicBold:
		// Symbol1016 post-game values use Century Gothic Bold. Prefer the
		// native Windows face. FFDec only exported the regular embedded face,
		// so keep that as a portable fallback rather than dropping the text.
		candidates = append(candidates, `C:\Windows\Fonts\GOTHICB.TTF`, `C:\Windows\Fonts\gothicb.ttf`)
		if p, err := findOriginalPath("fonts", "1006_Century Gothic.ttf"); err == nil {
			candidates = append(candidates, p)
		}
		candidates = append(candidates, `C:\Windows\Fonts\arialbd.ttf`)
	case menuFontShowcardGothic:
		if p, err := findOriginalPath("fonts", "18_Showcard Gothic.ttf"); err == nil {
			candidates = append(candidates, p)
		}
		candidates = append(candidates, `C:\Windows\Fonts\SHOWG.TTF`, `C:\Windows\Fonts\showg.ttf`)
	}

	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			cache.err = err
			continue
		}
		parsed, err := opentype.Parse(data)
		if err != nil {
			cache.err = err
			continue
		}
		cache.font = parsed
		cache.err = nil
		return
	}
	// Never draw nothing: the FFDec exports are all malformed subsets and the
	// Tw Cen/Showcard system files come with Office, not Windows.
	bold := kind != menuFontCondensed && kind != menuFontArial
	if parsed, err := embeddedUIFont(bold); err == nil {
		cache.font = parsed
		cache.err = nil
		cache.substitute = kind != menuFontArial && kind != menuFontArialBold
	}
}

func sourceMenuFace(kind sourceMenuFontKind, size float64) (font.Face, error) {
	cache := menuFontCaches[kind]
	if cache == nil {
		cache = menuFontCaches[menuFontUI]
	}
	cache.once.Do(func() { loadSourceMenuFont(kind, cache) })
	if cache.font == nil {
		return nil, cache.err
	}
	if cache.substitute {
		size *= substituteFontScale
	}
	key := int(size*100 + 0.5)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if f := cache.faces[key]; f != nil {
		return f, nil
	}
	f, err := opentype.NewFace(cache.font, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, err
	}
	cache.faces[key] = f
	return f, nil
}

// sourceMenuBaseline returns the pixel distance from the requested top of a
// text line to its baseline.
func sourceMenuBaseline(kind sourceMenuFontKind, size float64, face font.Face) int {
	if cache := menuFontCaches[kind]; cache != nil && cache.substitute {
		return int(math.Round(size * substituteBaselineFactor))
	}
	return face.Metrics().Ascent.Ceil()
}

func drawSourceMenuText(dst *ebiten.Image, value string, kind sourceMenuFontKind, size float64, clr color.Color, x, topY float64) {
	if value == "" {
		return
	}
	face, err := sourceMenuFace(kind, size)
	if err != nil || face == nil {
		return
	}
	baseline := int(topY) + sourceMenuBaseline(kind, size, face)
	ebitentext.Draw(dst, value, face, int(x), baseline, clr)
}
