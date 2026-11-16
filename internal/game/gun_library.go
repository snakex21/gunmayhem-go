package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type gunLibraryButton struct {
	Gun   int
	X, Y  float64
	Scale float64
	W, H  float64
}

func (b gunLibraryButton) contains(x, y float64) bool {
	if b.W > 0 && b.H > 0 {
		return x >= b.X-b.W/2 && x <= b.X+b.W/2 && y >= b.Y-b.H/2 && y <= b.Y+b.H/2
	}
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
	if (gun >= 1 && gun <= 6) || (gun >= 10 && gun <= 86) {
		return true
	}
	_, ok := developedWeaponSpec(gun)
	return ok
}

func (g *Game) gunUnlocked(gun int) bool {
	// Source handgun cards 1..6 index gunarray[-9..-4]. That value is
	// undefined, so the `== false` lock test never fires: starter handguns are
	// always selectable. Campaign unlocks only gate guns 10..66.
	if gun >= 1 && gun <= 6 {
		return true
	}
	if gun >= 67 && gun <= 86 {
		return gm2WeaponAssetsAvailable()
	}
	if _, ok := developedWeaponSpec(gun); ok {
		return customWeaponAssetAvailable(gun)
	}
	idx := gun - 10
	return idx >= 0 && idx < len(g.campaignGuns) && g.campaignGuns[idx]
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

func (g *Game) drawGunLibraryInteractions(screen *ebiten.Image) {
	g.assets.EnsureGunLibrary()
	g.drawDevelopedGunLibrary(screen)
}

func (g *Game) gunLibraryRaster(number int) *SourceRaster {
	if proceduralWeapon(number) {
		return proceduralWeaponThumb(number)
	}
	if number >= 87 {
		return g.assets.WeaponFrame(number, 0)
	}
	return g.assets.GunLibraryThumbs[number]
}

func (g *Game) startGunLibraryTest() {
	throwableTest := g.organizedGunLibraryThrowableTab()
	if (!throwableTest && !isGunLibraryGunNumber(g.gunLibrarySelected)) || g.fadeActive {
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
	if !throwableTest && g.gunLibrarySelected >= 1 && g.gunLibrarySelected <= 6 {
		// frame_10: starter handguns are equipped immediately in test mode;
		// Symbol1443 removes the pickup entirely for gototestnumber <= 6.
		p1.DefaultWeapon = g.gunLibrarySelected
	}
	p1.PerkNumber = 0
	p1.configureSourceGameMode(SourceGameModeNormal, 1000, arena)
	if throwableTest {
		p1.ThrowableType = normalizedThrowableType(g.gunLibraryThrowableSelected)
	}
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
	g.testThrowableMode = throwableTest
	g.testGunNumber = g.gunLibrarySelected
	if throwableTest {
		g.testGunNumber = 0
	}
	g.testGunDisabled = false
	g.testGunRespawn = 0
	g.testGunFrame = 2 // Symbol1443 frame1 immediately proceeds into visual frame2
	g.resetArenaState()
	if throwableTest {
		p1.ThrowableType = normalizedThrowableType(g.gunLibraryThrowableSelected)
		p1.Grenades = 30
	}
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
	if !g.gototest || g.testGunDisabled || g.testGunNumber <= 6 || !isGunLibraryGunNumber(g.testGunNumber) {
		return
	}
	// Source Symbol1443 frame2: persistent pedestal/base plus dropgun at
	// ~80% scale and -10 degrees. The selected weapon frame is runtime-driven.
	drawSourceRaster(screen, g.assets.TestPickupBase, g.worldX(testGunPickupX), g.worldY(testGunPickupY), 1, 1, 1)
	rotation := -10.0
	if g.testGunNumber >= 87 {
		rotation = 0
	}
	drawGunLibraryCleanThumb(screen, g.gunLibraryRaster(g.testGunNumber), g.worldX(testGunPickupX-1.75), g.worldY(testGunPickupY-1.8), 62, 44, 1, rotation)
}
