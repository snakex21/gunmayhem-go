package game

import (
	"encoding/xml"
	"image/color"
	"io"
	"math"
	"os"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type gunLibraryButton struct {
	Gun   int
	X, Y  float64
	Scale float64
}

var (
	gunLibraryButtonsOnce sync.Once
	gunLibraryButtonsData []gunLibraryButton
	gunLibraryNamesOnce   sync.Once
	gunLibraryNamesData   map[int]string
	gunLibraryCardOnce    sync.Once
	gunLibraryCardImage   *ebiten.Image
	gunLibraryEraseOnce   sync.Once
	gunLibraryEraseImage  *ebiten.Image
)

// sourceGunLibraryButtons recovers all 57 Symbol913 buttons directly from
// Symbol1198. The source stores the gun number in each instance's initial
// _rotation, reads Math.round(_rotation), then changes the visible rotation to
// -20 degrees. This avoids a hand-authored 57-entry coordinate table.
func sourceGunLibraryButtons() []gunLibraryButton {
	gunLibraryButtonsOnce.Do(func() {
		doc, err := loadXFLVectorDoc("Symbol 1198")
		if err != nil {
			return
		}
		for _, layer := range doc.Timeline.Layers {
			frame, ok := activeVectorFrame(layer.Frames, 0)
			if !ok {
				continue
			}
			for _, inst := range frame.Elements.Instances {
				if inst.Library != "Symbol 913" {
					continue
				}
				m := matrixFromVector(inst.Matrix.Value)
				gun := int(math.Round(math.Atan2(m.B, m.A) * 180 / math.Pi))
				if !isGunLibraryGunNumber(gun) {
					continue
				}
				s := math.Hypot(m.A, m.B)
				if s <= 0 {
					s = 1
				}
				gunLibraryButtonsData = append(gunLibraryButtonsData, gunLibraryButton{Gun: gun, X: m.TX, Y: m.TY, Scale: s})
			}
		}
	})
	return gunLibraryButtonsData
}

func sourceGunLibraryNames() map[int]string {
	gunLibraryNamesOnce.Do(func() {
		gunLibraryNamesData = map[int]string{}
		path, err := findOriginalPath("fla", "LIBRARY", "Symbol 1190.xml")
		if err != nil {
			return
		}
		f, err := os.Open(path)
		if err != nil {
			return
		}
		defer f.Close()

		dec := xml.NewDecoder(f)
		currentFrame := -1
		for {
			tok, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				return
			}
			switch t := tok.(type) {
			case xml.StartElement:
				switch t.Name.Local {
				case "DOMFrame":
					currentFrame = intAttr(t.Attr, "index", -1)
				case "characters":
					if currentFrame < 0 {
						continue
					}
					var value string
					if err := dec.DecodeElement(&value, &t); err != nil {
						return
					}
					value = strings.TrimSpace(value)
					if value != "" {
						gunLibraryNamesData[currentFrame+1] = value
					}
				}
			case xml.EndElement:
				if t.Name.Local == "DOMFrame" {
					currentFrame = -1
				}
			}
		}
	})
	return gunLibraryNamesData
}

func sourceGunLibraryName(frame int) string {
	if name := sourceGunLibraryNames()[frame]; name != "" {
		return name
	}
	if def, ok := WeaponByNumber(frame); ok {
		return def.Name
	}
	return "Nothing Selected"
}

func (b gunLibraryButton) contains(x, y float64) bool {
	// Constructor forces the clip to _rotation=-20 after extracting gunnumber.
	// Invert that runtime transform and test the exact Symbol913 frame square.
	dx := x - b.X
	dy := y - b.Y
	rad := 20 * math.Pi / 180 // inverse of -20 degrees
	c, s := math.Cos(rad), math.Sin(rad)
	lx := (dx*c - dy*s) / b.Scale
	ly := (dx*s + dy*c) / b.Scale
	return lx >= -20 && lx <= 20 && ly >= -20 && ly <= 20
}

func isGunLibraryGunNumber(gun int) bool {
	return (gun >= 1 && gun <= 6) || (gun >= 10 && gun <= 66)
}

func (g *Game) gunUnlocked(gun int) bool {
	// Source handgun cards 1..6 index gunarray[-9..-4]. That value is
	// undefined, so the `== false` lock test never fires: starter handguns are
	// always selectable. Campaign unlocks only gate guns 10..66.
	if gun >= 1 && gun <= 6 {
		return true
	}
	idx := gun - 10
	return idx >= 0 && idx < len(g.campaignGuns) && g.campaignGuns[idx]
}

func (g *Game) gunLibraryHitAt(x, y float64) int {
	for _, b := range sourceGunLibraryButtons() {
		if b.contains(x, y) && g.gunUnlocked(b.Gun) {
			return b.Gun
		}
	}
	return 0
}

func (g *Game) updateGunLibraryMenu() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.gunLibraryPressed = 0
		g.beginFade(screenMainMenu, fadePurposeScreen)
		return
	}
	mx, my := ebiten.CursorPosition()
	x, y := float64(mx), float64(my)
	backRect := sourceMenuHitRect(wideButtonLocal, 679.75, 20.05, 0.3125, 0.9991607666015625)
	// Symbol1197 is at (458.1,457.6), testbtn local (320.05,65.55).
	testRect := sourceMenuHitRect(Rect{X: 0, Y: 0, W: 76.5, H: 82.4}, 778.15, 523.15, 1, 0.484222412109375)

	hit := g.gunLibraryHitAt(x, y)
	const hitBack = -1
	const hitTest = -2
	if backRect.Contains(x, y) {
		hit = hitBack
	} else if g.gunLibrarySelected != 0 && testRect.Contains(x, y) {
		hit = hitTest
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.gunLibraryPressed = hit
	}
	if !inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		return
	}
	pressed := g.gunLibraryPressed
	g.gunLibraryPressed = 0
	if pressed == 0 || pressed != hit {
		return
	}
	switch pressed {
	case hitBack:
		g.beginFade(screenMainMenu, fadePurposeScreen)
	case hitTest:
		g.startGunLibraryTest()
	default:
		if isGunLibraryGunNumber(pressed) {
			g.gunLibrarySelected = pressed
		}
	}
}

func drawGunLibraryBarOutline(screen *ebiten.Image, x, y float64) {
	const w = 219.4
	const h = 5.6
	c := color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xff}
	ebitenutil.DrawLine(screen, x, y, x, y+h, c)
	ebitenutil.DrawLine(screen, x, y+h, x+w, y+h, c)
	ebitenutil.DrawLine(screen, x+w, y+h, x+w, y, c)
}

func (g *Game) drawGunLibraryStatCard(screen *ebiten.Image, frame int) {
	const x = 458.1
	const y = 457.6

	// Symbol1197 background is #797979. Cover only the dynamic/stat half of
	// the flattened guncard; the TEST button and dropgun live to the right.
	ebitenutil.DrawRect(screen, x, y, 315, 114, color.NRGBA{R: 0x79, G: 0x79, B: 0x79, A: 0xff})

	// Symbol1190's exported PNGs are all the same, but its XFL vector bars are
	// genuinely frame-specific. These source-raster frames contain those bars.
	drawSourceRaster(screen, g.assets.GunLibraryStats[frame], x, y, 1, 1, 1)

	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	name := sourceGunLibraryName(frame)
	if _, ok := WeaponByNumber(frame); ok {
		name = weaponDisplayName(frame)
	}
	drawSourceMenuText(screen, name, menuFontTwCen, 14.2, white, x+4.5, y+4.5)
	drawSourceMenuText(screen, "Damage", menuFontCondensed, 14.15, white, x+44, y+22.5)
	drawSourceMenuText(screen, "Rate of Fire", menuFontCondensed, 14.15, white, x+27, y+39)
	drawSourceMenuText(screen, "Ammo Capacity", menuFontCondensed, 14.15, white, x+9, y+55.5)
	drawSourceMenuText(screen, "Weight", menuFontCondensed, 14.15, white, x+50, y+73.5)
	drawSourceMenuText(screen, "Recoil", menuFontCondensed, 14.15, white, x+55, y+90)

	for _, top := range []float64{29.0, 45.35, 61.9, 78.4, 95.0} {
		drawGunLibraryBarOutline(screen, x+91.2, y+top)
	}
}

func drawGunLibraryCardBackground(screen *ebiten.Image, b gunLibraryButton) {
	// Symbol913 contains Symbol911 at (-20,-20). Symbol911's artwork is
	// Symbol336: an exact 800x800 #ff9900 square scaled by .05, i.e. 40x40.
	// The constructor then forces the whole Symbol913 clip to -20 degrees.
	gunLibraryCardOnce.Do(func() {
		gunLibraryCardImage = ebiten.NewImage(40, 40)
		gunLibraryCardImage.Fill(color.NRGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff})
	})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-20, -20)
	op.GeoM.Scale(b.Scale, b.Scale)
	op.GeoM.Rotate(-20 * math.Pi / 180)
	op.GeoM.Translate(b.X, b.Y)
	screen.DrawImage(gunLibraryCardImage, op)
}

func eraseFrozenGunLibraryButton(screen *ebiten.Image, b gunLibraryButton) {
	// FFDec flattens Symbol1198 before Symbol913's constructor runs. The source
	// matrix rotation (10..66 degrees) is only a hidden gun-number carrier; at
	// runtime AS2 immediately changes it to -20. Erase that frozen instance in
	// its original encoded rotation before drawing the real runtime card.
	gunLibraryEraseOnce.Do(func() {
		gunLibraryEraseImage = ebiten.NewImage(48, 48)
		gunLibraryEraseImage.Fill(color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-24, -24)
	op.GeoM.Scale(b.Scale, b.Scale)
	op.GeoM.Rotate(float64(b.Gun) * math.Pi / 180)
	op.GeoM.Translate(b.X, b.Y)
	screen.DrawImage(gunLibraryEraseImage, op)
}

func drawGunLibraryCleanThumb(screen *ebiten.Image, raster *SourceRaster, x, y, maxW, maxH, scale, rotation float64) {
	if raster == nil || raster.Image == nil {
		return
	}
	bounds := raster.Image.Bounds()
	w, h := float64(bounds.Dx()), float64(bounds.Dy())
	if w <= 0 || h <= 0 {
		return
	}
	s := math.Min(maxW/w, maxH/h) * scale
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Scale(s, s)
	op.GeoM.Rotate(rotation * math.Pi / 180)
	op.GeoM.Translate(x, y)
	screen.DrawImage(raster.Image, op)
}

func (g *Game) drawGunLibraryButtons(screen *ebiten.Image) {
	buttons := sourceGunLibraryButtons()
	// First remove every FFDec-frozen Symbol913. Doing this in a separate pass
	// prevents one erase rectangle from covering a neighboring rebuilt card.
	for _, b := range buttons {
		eraseFrozenGunLibraryButton(screen, b)
	}
	for _, b := range buttons {
		drawGunLibraryCardBackground(screen, b)
		// Use the same clean gun sprite as gameplay, not Symbol595's flattened
		// wrapper (which exposes Flash helper/control-point artwork).
		drawGunLibraryCleanThumb(screen, g.assets.GunLibraryThumbs[b.Gun], b.X, b.Y, 34, 25, b.Scale, -20)
		if !g.gunUnlocked(b.Gun) {
			// Symbol913 frame2 adds this exact white direct-shape overlay when the
			// campaign gunarray entry is false. Nested dropgun artwork is skipped.
			drawSourceRasterRot(screen, g.assets.GunLibraryLockedOverlay, b.X, b.Y, b.Scale, b.Scale, -20, 1)
		}
	}
}

func (g *Game) drawGunLibraryInteractions(screen *ebiten.Image) {
	g.assets.EnsureGunLibrary()
	g.drawGunLibraryButtons(screen)

	frame := g.gunLibrarySelected
	if frame == 0 {
		frame = 7 // Symbol1197 constructor: Nothing Selected
	}
	g.drawGunLibraryStatCard(screen, frame)
	// The nested Symbol595 export exposes helper markers; use the clean gameplay
	// sprite for the selected-weapon preview as well.
	// The flattened Symbol1198/1197 image still contains its constructor-time
	// default pistol here. Clear that exact dropgun area before drawing the
	// runtime-selected weapon, otherwise two guns are visible at once.
	ebitenutil.DrawRect(screen, 780, 463, 67, 58, color.NRGBA{R: 0x79, G: 0x79, B: 0x79, A: 0xff})
	if isGunLibraryGunNumber(frame) {
		drawGunLibraryCleanThumb(screen, g.assets.GunLibraryThumbs[frame], 813.3, 489.55, 58, 42, 1, -20)
	}
}

func (g *Game) startGunLibraryTest() {
	if !isGunLibraryGunNumber(g.gunLibrarySelected) || g.fadeActive {
		return
	}
	arena, ok := g.ensureMap(13)
	if !ok {
		return
	}
	p1 := NewPlayer(1, arena)
	p1.Controls = g.controlConfigs[0]
	p1.Name = "Player"
	p1.PlayerColor = 2
	p1.ShirtNumber = 1
	p1.DefaultWeapon = 2
	if g.gunLibrarySelected >= 1 && g.gunLibrarySelected <= 6 {
		// frame_10: starter handguns are equipped immediately in test mode;
		// Symbol1443 removes the pickup entirely for gototestnumber <= 6.
		p1.DefaultWeapon = g.gunLibrarySelected
	}
	p1.PerkNumber = 0
	p1.configureSourceGameMode(SourceGameModeNormal, 1000, arena)
	p2 := NewPlayer(2, arena)
	p2.Controls = g.controlConfigs[1]
	p2.Name = "Dummy"
	p2.PlayerColor = 8
	p2.ShirtNumber = 24
	p2.DefaultWeapon = 2
	p2.PerkNumber = 0
	p2.AI = true
	p2.configureSourceGameMode(SourceGameModeNormal, 1000, arena)

	g.arena = arena
	g.players = []*Player{p1, p2}
	g.GameMode = SourceGameModeNormal
	g.TotalLives = 1000
	g.TeamGame = false
	g.CrateON = false
	g.PowerON = false
	g.GameWin = false
	g.teamGameWin = false
	g.campaignMode = false
	g.gototest = true
	g.testGunNumber = g.gunLibrarySelected
	g.testGunDisabled = false
	g.testGunRespawn = 0
	g.testGunFrame = 2 // Symbol1443 frame1 immediately proceeds into visual frame2
	g.resetArenaState()
	g.beginFade(screenGameplay, fadePurposeScreen)
}

const (
	testGunPickupX = 468.2 // ground(66,210.2) + Symbol1443 local (402.2,84.15)
	testGunPickupY = 294.35
)

func (g *Game) updateTestGunPickup() {
	if !g.gototest || g.GameMode == SourceGameModeInstagib || g.GameMode == SourceGameModeGunGame || g.testGunNumber <= 6 {
		return
	}
	if g.testGunDisabled {
		g.testGunRespawn++
		if g.testGunRespawn >= 100 {
			g.testGunRespawn = 0
			g.testGunDisabled = false
			g.testGunFrame = 10 // source gotoAndPlay(random(40)+10); first possible frame10
		}
		return
	}
	if g.testGunFrame < 61 {
		g.testGunFrame++
		if g.testGunFrame >= 61 {
			g.testGunFrame = 2
		}
	}
	if len(g.players) == 0 || !g.players[0].Active {
		return
	}
	// Source uses this.hitTest(activeplayers[0].frame). The visible pickup is
	// centered at its registration point; use the source visual bounds for the
	// same movie clip against the reconstructed player frame hitbox.
	pickup := Rect{X: testGunPickupX - 45, Y: testGunPickupY - 45, W: 90, H: 90}
	if rectsOverlap(pickup, g.players[0].Hitbox()) {
		g.players[0].EquipWeapon(g.testGunNumber)
		g.testGunDisabled = true
		g.testGunRespawn = 0
	}
}

func (g *Game) drawTestGunPickup(screen *ebiten.Image) {
	if !g.gototest || g.testGunDisabled || g.testGunNumber < 10 || g.testGunNumber > 66 {
		return
	}
	// Source Symbol1443 frame2: persistent pedestal/base plus dropgun at
	// ~80% scale and -10 degrees. The selected weapon frame is runtime-driven.
	drawSourceRaster(screen, g.assets.TestPickupBase, g.worldX(testGunPickupX), g.worldY(testGunPickupY), 1, 1, 1)
	drawGunLibraryCleanThumb(screen, g.assets.GunLibraryThumbs[g.testGunNumber], g.worldX(testGunPickupX-1.75), g.worldY(testGunPickupY-1.8), 58, 42, 1, -10)
}
