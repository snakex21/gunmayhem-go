package game

import (
	"image/color"
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
)

type sourceMenuFontCache struct {
	once  sync.Once
	font  *opentype.Font
	err   error
	mu    sync.Mutex
	faces map[int]font.Face
}

var menuFontCaches = map[sourceMenuFontKind]*sourceMenuFontCache{
	menuFontCondensed:          {faces: map[int]font.Face{}},
	menuFontCondensedExtraBold: {faces: map[int]font.Face{}},
	menuFontTwCen:              {faces: map[int]font.Face{}},
	menuFontArial:              {faces: map[int]font.Face{}},
	menuFontArialBold:          {faces: map[int]font.Face{}},
	menuFontCenturyGothicBold:  {faces: map[int]font.Face{}},
	menuFontShowcardGothic:     {faces: map[int]font.Face{}},
}

func sourceMenuFace(kind sourceMenuFontKind, size float64) (font.Face, error) {
	cache := menuFontCaches[kind]
	cache.once.Do(func() {
		var candidates []string
		switch kind {
		case menuFontCondensedExtraBold:
			if p, err := findOriginalPath("fonts", "20_Tw Cen MT Condensed Extra Bold.ttf"); err == nil {
				candidates = append(candidates, p)
			}
			candidates = append(candidates, `C:\Windows\Fonts\TCCEB.TTF`)
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
	})
	if cache.font == nil {
		return nil, cache.err
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

func drawSourceMenuText(dst *ebiten.Image, value string, kind sourceMenuFontKind, size float64, clr color.Color, x, topY float64) {
	if value == "" {
		return
	}
	face, err := sourceMenuFace(kind, size)
	if err != nil || face == nil {
		return
	}
	baseline := int(topY) + face.Metrics().Ascent.Ceil()
	ebitentext.Draw(dst, value, face, int(x), baseline, clr)
}
