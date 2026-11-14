package game

import "image/color"

// Mayhem UI theme: the single source of colours, type sizes and layout grid
// for screens built with the UI kit (ui_kit_*.go). "Ink + paper": dark page,
// light cards, one orange for action/selection/focus. Blue is reserved for the
// small online/network badge and is never a whole-screen theme.
var (
	uiInk900      = color.NRGBA{R: 0x15, G: 0x18, B: 0x1d, A: 0xff} // page background
	uiInk700      = color.NRGBA{R: 0x26, G: 0x2b, B: 0x33, A: 0xff} // panels, inactive tabs
	uiInk600      = color.NRGBA{R: 0x30, G: 0x36, B: 0x40, A: 0xff} // hovered panel controls
	uiInk500      = color.NRGBA{R: 0x3a, G: 0x41, B: 0x4c, A: 0xff} // borders, secondary buttons
	uiPaper       = color.NRGBA{R: 0xf2, G: 0xf2, B: 0xef, A: 0xff} // cards, header plate, rows
	uiPaperHover  = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	uiPaperShade  = color.NRGBA{R: 0xd9, G: 0xd9, B: 0xd4, A: 0xff} // pressed paper / chips
	uiTextDark    = color.NRGBA{R: 0x11, G: 0x11, B: 0x11, A: 0xff} // text on paper
	uiTextLight   = color.NRGBA{R: 0xf5, G: 0xf5, B: 0xf5, A: 0xff} // text on ink
	uiTextMuted   = color.NRGBA{R: 0xa9, G: 0xb0, B: 0xba, A: 0xff} // hints on ink
	uiTextMutedDk = color.NRGBA{R: 0x5a, G: 0x5f, B: 0x66, A: 0xff} // hints on paper
	uiOrange      = color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff} // primary, selected, focus
	uiOrangeHover = color.NRGBA{R: 0xff, G: 0xad, B: 0x33, A: 0xff}
	uiOrangePress = color.NRGBA{R: 0xd9, G: 0x7f, B: 0x00, A: 0xff}
	uiNetBlue     = color.NRGBA{R: 0x1f, G: 0x7a, B: 0xe0, A: 0xff} // online badge only
	uiDanger      = color.NRGBA{R: 0xd8, G: 0x3a, B: 0x2e, A: 0xff} // kick, leave, delete
	uiDangerHover = color.NRGBA{R: 0xe8, G: 0x55, B: 0x49, A: 0xff}
	uiOK          = color.NRGBA{R: 0x3b, G: 0xb2, B: 0x73, A: 0xff} // ready, connected
	uiScrim       = color.NRGBA{R: 0x08, G: 0x0a, B: 0x0d, A: 0xc8} // behind modals
)

// Type scale. One family (Barlow Condensed, embedded); nothing below 13 px.
const (
	uiSizeTitle   = 36.0 // screen title
	uiSizeSection = 24.0 // section / card title
	uiSizeButton  = 18.0 // buttons, tabs, list rows
	uiSizeBody    = 15.0 // body text, values
	uiSizeHint    = 13.0 // hints and key bar (minimum)
)

// Layout grid for the 900x600 logical screen.
const (
	uiMargin        = 20.0
	uiGap           = 8.0
	uiHeaderTop     = 20.0
	uiHeaderBottom  = 76.0
	uiContentTop    = 92.0
	uiContentBottom = 516.0
	uiActionTop     = 528.0
	uiActionBottom  = 572.0
	uiHintTop       = 578.0
	uiHintBottom    = 596.0
	uiButtonH       = uiActionBottom - uiActionTop // 44
	uiCompactH      = 36.0
)

// uiActionBackRect is the action-bar slot for Back (always bottom-left).
func uiActionBackRect() Rect {
	return Rect{X: uiMargin, Y: uiActionTop, W: 200, H: uiButtonH}
}

// uiActionPrimaryRect is the action-bar slot for the primary action (always
// bottom-right).
func uiActionPrimaryRect(width float64) Rect {
	return Rect{X: ScreenWidth - uiMargin - width, Y: uiActionTop, W: width, H: uiButtonH}
}

func uiWithAlpha(c color.NRGBA, a float64) color.NRGBA {
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	c.A = uint8(float64(c.A)*a + 0.5)
	return c
}
