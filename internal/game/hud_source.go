package game

import (
	"fmt"
	"image/color"
	"os"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	ebitentext "github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

var (
	hudFontOnce sync.Once
	hudTTF      *opentype.Font
	hudFontErr  error
	hudFaceMu   sync.Mutex
	hudFaces    = map[int]font.Face{}
)

func sourceHUDFace(size float64) (font.Face, error) {
	hudFontOnce.Do(func() {
		// FFDec exported the embedded Tw Cen MT Condensed Extra Bold file with
		// a malformed `head` table, so x/image/opentype correctly rejects it.
		// Flash used this exact Windows font family for the HUD. Prefer the
		// extracted asset when it is valid, then the native Windows TCCEB font,
		// and finally the bundled condensed face as a portable fallback.
		var candidates []string
		if path, err := findOriginalPath("fonts", "20_Tw Cen MT Condensed Extra Bold.ttf"); err == nil {
			candidates = append(candidates, path)
		}
		candidates = append(candidates, `C:\Windows\Fonts\TCCEB.TTF`)
		if path, err := findOriginalPath("fonts", "49_Tw Cen MT Condensed.ttf"); err == nil {
			candidates = append(candidates, path)
		}
		for _, path := range candidates {
			data, err := os.ReadFile(path)
			if err != nil {
				hudFontErr = err
				continue
			}
			parsed, err := opentype.Parse(data)
			if err != nil {
				hudFontErr = err
				continue
			}
			hudTTF = parsed
			hudFontErr = nil
			return
		}
	})
	if hudFontErr != nil || hudTTF == nil {
		return nil, hudFontErr
	}

	// XFL text sizes used by this HUD are fixed. Key in hundredths so 13.15
	// remains a distinct source size without floating-point map ambiguity.
	key := int(size*100 + 0.5)
	hudFaceMu.Lock()
	defer hudFaceMu.Unlock()
	if f := hudFaces[key]; f != nil {
		return f, nil
	}
	f, err := opentype.NewFace(hudTTF, &opentype.FaceOptions{
		Size: size,
		DPI:  72,
		// Flash's dynamic text here is anti-aliased vector text; do not force a
		// pixel-grid font substitution.
		Hinting: font.HintingNone,
	})
	if err != nil {
		return nil, err
	}
	hudFaces[key] = f
	return f, nil
}

func (g *Game) drawSourceHUDCards(screen *ebiten.Image) {
	// Symbol1489 frame1 places the four Symbol1480 cards at these exact points.
	cardX := [...]float64{120, 340, 560, 780}
	for i := 0; i < 4; i++ {
		p := g.sourceHUDPlayer(i + 1)
		if p == nil {
			continue // source: cardN._alpha = 0 when playerN does not exist
		}
		g.drawSourceHUDCard(screen, p, cardX[i], 550)
	}
}

func (g *Game) sourceHUDPlayer(id int) *Player {
	for _, p := range g.players {
		if p.IsDouble || p.ID != id {
			continue
		}
		return p
	}
	return nil
}

// updateHUDLastLife mirrors Symbol1480.lastlife. Each card owns an independent
// Symbol1456 playhead: frame1 stops; one remaining life plays through frame90;
// zero lives resumes through frame180; Gun Game/Survival level-ups play 181..270.
func (g *Game) updateHUDLastLife() {
	for slot := 0; slot < 4; slot++ {
		p := g.sourceHUDPlayer(slot + 1)
		if p == nil {
			g.hudLastLifeFrame[slot] = 0
			g.hudLastLifePlaying[slot] = false
			g.hudLastLevel[slot] = 0
			continue
		}

		// Advance a clip that was already playing before this tick. Source stop()
		// points are Flash frames 90, 180 and 270 (zero-based 89/179/269).
		if g.hudLastLifePlaying[slot] {
			frame := g.hudLastLifeFrame[slot] + 1
			switch {
			case frame >= 269:
				frame = 269
				g.hudLastLifePlaying[slot] = false
			case frame == 179:
				g.hudLastLifePlaying[slot] = false
			case frame == 89:
				g.hudLastLifePlaying[slot] = false
			}
			g.hudLastLifeFrame[slot] = frame
		}

		if g.GameMode != SourceGameModeGunGame && g.GameMode != SourceGameModeSurvival {
			// Symbol1480 starts LAST LIFE only from source frame1. After the
			// warning reaches frame90 it must not retrigger on later HUD updates.
			if p.Lives == 1 && g.hudLastLifeFrame[slot] == 0 {
				g.hudLastLifePlaying[slot] = true
			}
			if p.Lives == 0 {
				// If the final death happens before LAST LIFE has finished sliding
				// away, do not leave the obsolete LAST LIFE text exposed while the
				// HUD already says 0. Collapse to the hidden frame90 stop and resume
				// the GAME OVER phase from there.
				if g.hudLastLifeFrame[slot] < 89 {
					g.hudLastLifeFrame[slot] = 89
				}
				if g.hudLastLifeFrame[slot] == 89 {
					g.hudLastLifePlaying[slot] = true
				}
			}
		} else {
			if g.hudLastLevel[slot] < p.CurrentLevel {
				// gotoAndPlay(181) -> zero-based frame180.
				g.hudLastLifeFrame[slot] = 180
				g.hudLastLifePlaying[slot] = true
			}
			g.hudLastLevel[slot] = p.CurrentLevel
		}
	}
}

func (g *Game) resetHUDLastLife() {
	for slot := 0; slot < 4; slot++ {
		g.hudLastLifeFrame[slot] = 0
		g.hudLastLifePlaying[slot] = false
		if p := g.sourceHUDPlayer(slot + 1); p != nil {
			g.hudLastLevel[slot] = p.CurrentLevel
		} else {
			g.hudLastLevel[slot] = 0
		}
	}
}

func (g *Game) hudLastLifeShouldDraw(slot int) bool {
	if slot < 0 || slot >= len(g.hudLastLifeFrame) {
		return false
	}
	frame := g.hudLastLifeFrame[slot]

	// Gun Game/Survival use this timeline only for LEVEL UP (181..270).
	// LAST LIFE and GAME OVER must never appear in those modes.
	if g.GameMode == SourceGameModeGunGame || g.GameMode == SourceGameModeSurvival {
		return frame >= 180
	}

	p := g.sourceHUDPlayer(slot + 1)
	if p == nil {
		return false
	}

	// Hard guard: the LAST LIFE phase is drawable only while the player's
	// actual life counter is exactly 1. Any stale animation state from a kill,
	// respawn or mode transition must not leak the text at 0, 2, 3, etc.
	if frame > 0 && frame < 89 {
		return p.Lives == 1
	}

	// Source frame90 is the hidden stop between LAST LIFE and GAME OVER.
	if frame == 0 || frame == 89 {
		return false
	}

	// GAME OVER belongs only to a player that is actually out of lives.
	if frame >= 90 && frame < 180 {
		return p.Lives == 0
	}
	return false
}

func (g *Game) drawSourceHUDWarningText(screen *ebiten.Image, frame int, value string, size, localX, localY, x, y float64) {
	if screen == nil || value == "" {
		return
	}
	state, ok := g.assets.hudLastLifeTransform(frame)
	if !ok || state.Alpha <= 0 {
		return
	}
	alpha := state.Alpha
	if alpha > 1 {
		alpha = 1
	}
	a := uint8(alpha*255 + 0.5)
	drawX := x + state.Matrix.TX + localX
	drawY := y + state.Matrix.TY + localY
	shadow := color.NRGBA{A: a}
	for oy := -3; oy <= 3; oy++ {
		for ox := -3; ox <= 3; ox++ {
			if ox == 0 && oy == 0 {
				continue
			}
			drawSourceMenuText(screen, value, menuFontShowcardGothic, size, shadow,
				drawX+float64(ox), drawY+float64(oy))
		}
	}
	drawSourceMenuText(screen, value, menuFontShowcardGothic, size,
		color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: a}, drawX, drawY)
}

func (g *Game) drawSourceHUDCard(screen *ebiten.Image, p *Player, x, y float64) {
	// Symbol1480 puts lastlife at local (1.85,3.9) behind the ordinary card.
	// LAST LIFE pops up above the card, then returns behind it and stops on
	// source frame90. The player's final death resumes the same timeline into
	// GAME OVER, exactly as in the Flash source.
	if p.ID >= 1 && p.ID <= 4 {
		slot := p.ID - 1
		frame := g.hudLastLifeFrame[slot]
		if g.hudLastLifeShouldDraw(slot) {
			switch {
			case frame >= 90 && frame < 180:
				// Symbol1453 DOMStaticText: tx=-95.3, ty=-39.2, leftMargin=19.
				g.drawSourceHUDWarningText(screen, frame, "GAME OVER", 79.35,
					-95.3+19, -82.2, x+1.85, y+3.9)
			case frame >= 180:
				// Symbol1455 DOMStaticText: tx=-95.3, ty=-39.2, leftMargin=34.
				g.drawSourceHUDWarningText(screen, frame, "LEVEL UP", 79.35,
					-95.3+34, -39.2, x+1.85, y+3.9)
			default:
				drawSourceRaster(screen, g.assets.HUDLastLifeFrame(frame), x+1.85, y+3.9, 1, 1, 1)
			}
		}
	}

	// Do not use the flattened Symbol1480 PNG when the clean source pieces are
	// available. That export contains the authoring placeholders "PlayerName",
	// "99", "12" and "WEAPON NAME", which Flash replaces through dynamic text
	// fields at runtime. Drawing it underneath the live values makes the HUD
	// look corrupted. Compose the non-text source pieces first, then draw the
	// runtime values exactly once below.
	if g.assets.HUDCardBG != nil || g.assets.HUDCardDivider != nil {
		drawSourceRaster(screen, g.assets.HUDCardBG, x-1.0, y+2.1, 1, 1, 1)
		drawSourceRaster(screen, g.assets.HUDCardDivider, x+24.8, y-11.5, 1, 1, 1)
	} else if g.assets.HUDCard != nil {
		drawSourceRaster(screen, g.assets.HUDCard, x, y, 1, 1, 1)
	}

	// displayheader Symbol1476: frame1=LIVES, frame2=LEVEL, frame3=HEALTH.
	header := "LIVES:"
	if g.GameMode == SourceGameModeGunGame {
		header = "LEVEL"
	} else if g.GameMode == SourceGameModeSurvival {
		header = "HEALTH"
	}
	drawSourceHUDText(screen, header, 13.15, color.NRGBA{A: 255}, x-34.35, y-9.3)

	// Dynamic text matrices and text attributes are copied from Symbol1480.
	drawSourceHUDText(screen, p.Name, 15, color.NRGBA{A: 255}, x-29.95, y-30.9)

	life := p.Lives
	if g.GameMode == SourceGameModeGunGame {
		life = p.CurrentLevel
	}
	drawSourceHUDTextCentered(screen, fmt.Sprintf("%d", life), 28,
		color.NRGBA{R: 0x99, A: 255}, x-36.15, y+4.2, 42)

	if life == 0 {
		return
	}

	ammo := fmt.Sprintf("%d", p.Weapon.Bullets)
	if p.Weapon.Bullets > 1000 {
		ammo = "∞"
	}
	drawSourceHUDText(screen, ammo, 21,
		color.NRGBA{R: 0xff, G: 0x99, A: 255}, x+13.4, y-8.55)
	drawSourceHUDText(screen, p.Weapon.Def.Name, 11,
		color.NRGBA{A: 255}, x+13.5, y+13.6)
}

func makeSourcePowerupDoubleIcon() *SourceRaster {
	face, err := sourceHUDFace(46.05)
	if err != nil || face == nil {
		return nil
	}
	// Symbol730 frame5 DOMStaticText: left=47.35, tx=-71.65, ty=-27.15.
	// The resulting visible text starts at local x=-24.3. Keep a compact local
	// canvas around that registration point so the parent Symbol734 transform
	// can be applied exactly like the other icon frames.
	const minX, minY = -30.0, -30.0
	img := ebiten.NewImage(64, 52)
	baseline := 3 + face.Metrics().Ascent.Ceil()
	ebitentext.Draw(img, "X2", face, 6, baseline,
		color.NRGBA{R: 0xff, G: 0xcc, A: 0xff})
	return &SourceRaster{Image: img, Bounds: Rect{X: minX, Y: minY, W: 64, H: 52}}
}

func drawSourceHUDText(dst *ebiten.Image, value string, size float64, clr color.Color, x, topY float64) {
	if value == "" {
		return
	}
	face, err := sourceHUDFace(size)
	if err != nil || face == nil {
		return
	}
	baseline := int(topY) + face.Metrics().Ascent.Ceil()
	ebitentext.Draw(dst, value, face, int(x), baseline, clr)
}

func drawSourceHUDTextCentered(dst *ebiten.Image, value string, size float64, clr color.Color, x, topY, width float64) {
	face, err := sourceHUDFace(size)
	if err != nil || face == nil || value == "" {
		return
	}
	textWidth := font.MeasureString(face, value).Ceil()
	baseline := int(topY) + face.Metrics().Ascent.Ceil()
	drawX := int(x + (width-float64(textWidth))/2)
	ebitentext.Draw(dst, value, face, drawX, baseline, clr)
}
