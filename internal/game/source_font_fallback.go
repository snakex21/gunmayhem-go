package game

import (
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// sourceFallbackFaceCache holds whole-label fallback faces built from the
// embedded Barlow Condensed Bold (ui_font.go), keyed by size in hundredths.
var sourceFallbackFaceCache = sourceMenuFontCache{faces: map[int]font.Face{}}

func sourceHUDFaceForText(size float64, value string) (font.Face, error) {
	face, err := sourceHUDFace(size)
	return sourceFaceForText(face, err, size, value)
}

// sourceFaceForText selects one complete face before either measuring or
// drawing. The source typography is kept for labels it covers. The normalized
// FFDec exports are glyph subsets, so a label with a missing glyph is drawn
// entirely in the fallback face, never as a per-glyph mixture.
func sourceFaceForText(preferred font.Face, preferredErr error, size float64, value string) (font.Face, error) {
	if preferredErr == nil && sourceFaceCoversText(preferred, value) {
		return preferred, nil
	}
	fallback, err := sourceFallbackFace(size)
	if err == nil && sourceFaceCoversText(fallback, value) {
		return fallback, nil
	}
	// The fallback is not universal Unicode coverage. If neither face covers
	// the label, preserve the original result rather than claim to repair it.
	return preferred, preferredErr
}

func sourceFaceCoversText(face font.Face, value string) bool {
	if face == nil {
		return false
	}
	for _, r := range value {
		if r == '\n' { // Ebitengine draws this as a line break, not a glyph.
			continue
		}
		if _, ok := face.GlyphAdvance(r); !ok {
			return false
		}
	}
	return true
}

func sourceFallbackFace(size float64) (font.Face, error) {
	cache := &sourceFallbackFaceCache
	cache.once.Do(func() { cache.font, cache.err = embeddedUIFont(true) })
	if cache.err != nil {
		return nil, cache.err
	}
	key := int(size*100 + 0.5)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if face := cache.faces[key]; face != nil {
		return face, nil
	}
	face, err := opentype.NewFace(cache.font, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, err
	}
	cache.faces[key] = face
	return face, nil
}
