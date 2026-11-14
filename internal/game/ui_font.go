package game

import (
	_ "embed"
	"sync"

	"golang.org/x/image/font/opentype"
)

// Barlow Condensed (c) 2017 The Barlow Project Authors, SIL Open Font
// License 1.1. The licence ships beside the binary as
// assets/fonts/BarlowCondensed-OFL.txt and is kept here as uifonts/OFL.txt.
var (
	//go:embed uifonts/BarlowCondensed-Bold.ttf
	uiFontBoldTTF []byte
	//go:embed uifonts/BarlowCondensed-SemiBold.ttf
	uiFontSemiBoldTTF []byte
)

var (
	uiFontParseOnce   sync.Once
	uiFontBoldParsed  *opentype.Font
	uiFontBodyParsed  *opentype.Font
	uiFontParseErrors [2]error
)

func embeddedUIFont(bold bool) (*opentype.Font, error) {
	uiFontParseOnce.Do(func() {
		uiFontBoldParsed, uiFontParseErrors[0] = opentype.Parse(uiFontBoldTTF)
		uiFontBodyParsed, uiFontParseErrors[1] = opentype.Parse(uiFontSemiBoldTTF)
	})
	if bold {
		return uiFontBoldParsed, uiFontParseErrors[0]
	}
	return uiFontBodyParsed, uiFontParseErrors[1]
}
