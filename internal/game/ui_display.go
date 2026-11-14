package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// Display and render-quality settings. Everything here is purely local
// presentation: none of it is part of the simulation, the netplay snapshot or
// the host's authority. BlastFX spawning is untouched; LOW quality only skips
// drawing the original's optional explosion particles.

const (
	displayScalingFit = iota
	displayScalingInteger
	displayScalingStretch
	displayScalingCount
)

const (
	displayFilterAuto = iota
	displayFilterSharp
	displayFilterPixelated
	displayFilterSmooth
	displayFilterCount
)

var displayScalingLabels = []string{"FIT", "INTEGER", "STRETCH"}
var displayFilterLabels = []string{"AUTO", "SHARP", "PIXEL", "SMOOTH"}
var displayQualityLabels = []string{"LOW", "MEDIUM", "HIGH"}

// displayWindowPresets are window scales of the 900x600 logical screen in
// device-independent pixels.
var displayWindowPresets = []struct {
	label string
	scale float64
}{
	{"1X", 1}, {"1.5X", 1.5}, {"2X", 2}, {"3X", 3},
}

const displayMinWindowW, displayMinWindowH = 480, 320

// renderQualityProfile is what QUALITY means in this port (all render-only).
type renderQualityProfile struct {
	// blastParticles mirrors the original `_quality != "LOW"` gate around the
	// five fx_ex5 + five fx_ex3 grenade/ducky explosion particles. They are
	// still spawned (BlastFX is in the online snapshot); LOW skips drawing.
	blastParticles bool
	// finalFilter is the window upscale filter used by the AUTO filter:
	// LOW sharp, MEDIUM pixelated, HIGH smooth (Flash-like anti-aliasing).
	finalFilter ebiten.Filter
}

func renderProfileForQuality(quality int) renderQualityProfile {
	switch quality {
	case 1:
		return renderQualityProfile{blastParticles: false, finalFilter: ebiten.FilterNearest}
	case 3:
		return renderQualityProfile{blastParticles: true, finalFilter: ebiten.FilterLinear}
	default:
		return renderQualityProfile{blastParticles: true, finalFilter: ebiten.FilterPixelated}
	}
}

func (g *Game) renderProfile() renderQualityProfile { return renderProfileForQuality(g.quality) }

func blastKindIsQualityParticle(kind BlastEffectKind) bool {
	return kind == BlastEX5 || kind == BlastEX3
}

// setQuality applies a QUALITY level (1..3) to every render consumer.
func (g *Game) setQuality(q int) {
	if q < 1 || q > 3 {
		q = 2
	}
	g.quality = q
	setSourceRenderQuality(q)
	if g.assets != nil {
		g.assets.invalidateQualityDependentPlayerRasters()
	}
}

// displayFinalFilter picks the window upscale filter.
func displayFinalFilter(filterSetting, quality int, scale float64) ebiten.Filter {
	if scale < 1 {
		return ebiten.FilterLinear // never point-sample a downscale
	}
	switch filterSetting {
	case displayFilterSharp:
		return ebiten.FilterNearest
	case displayFilterPixelated:
		return ebiten.FilterPixelated
	case displayFilterSmooth:
		return ebiten.FilterLinear
	}
	return renderProfileForQuality(quality).finalFilter
}

// displayFinalGeoM maps the 900x600 offscreen onto a sw x sh final screen.
// fit is Ebitengine's own aspect-fit (letterbox) transform.
func displayFinalGeoM(mode int, sw, sh float64, fit ebiten.GeoM) ebiten.GeoM {
	switch mode {
	case displayScalingInteger:
		k := math.Floor(math.Min(sw/ScreenWidth, sh/ScreenHeight))
		if k < 1 {
			return fit // window smaller than 1x: integer scaling is impossible
		}
		var geo ebiten.GeoM
		geo.Scale(k, k)
		geo.Translate(math.Floor((sw-ScreenWidth*k)/2), math.Floor((sh-ScreenHeight*k)/2))
		return geo
	case displayScalingStretch:
		var geo ebiten.GeoM
		geo.Scale(sw/ScreenWidth, sh/ScreenHeight)
		return geo
	}
	return fit
}

// DrawFinalScreen implements ebiten.FinalScreenDrawer: scaling mode, filter
// and the render-only screen shake.
func (g *Game) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	b := screen.Bounds()
	sw, sh := float64(b.Dx()), float64(b.Dy())
	geo := displayFinalGeoM(g.displayScaling, sw, sh, geoM)
	g.finalEbitenGeo, g.finalGeo, g.finalGeoValid = geoM, geo, true

	op := &ebiten.DrawImageOptions{}
	op.GeoM = geo
	if shake := g.screenShakeOffset(); shake != 0 {
		op.GeoM.Translate(shake*geo.Element(0, 0), 0)
	}
	op.Filter = displayFinalFilter(g.displayFilter, g.quality, geo.Element(0, 0))
	screen.Fill(color.Black)
	screen.DrawImage(offscreen, op)
}

// cursorPos returns the cursor in virtual 900x600 coordinates. Every menu uses
// it instead of ebiten.CursorPosition(): Ebitengine maps the cursor through
// its own aspect-fit transform, which is wrong for Integer and Stretch
// scaling. In Fit mode the result is exactly ebiten.CursorPosition().
func (g *Game) cursorPos() (int, int) {
	x, y := ebiten.CursorPosition()
	if g == nil || !g.finalGeoValid || g.displayScaling == displayScalingFit {
		return x, y
	}
	return remapCursor(x, y, g.finalEbitenGeo, g.finalGeo)
}

// remapCursor converts a position Ebitengine reported through its fit
// transform into the logical position under our own final transform.
func remapCursor(x, y int, ebitenGeo, ourGeo ebiten.GeoM) (int, int) {
	if !ourGeo.IsInvertible() {
		return x, y
	}
	// Ebitengine floors the logical position; use the pixel centre.
	dx, dy := ebitenGeo.Apply(float64(x)+0.5, float64(y)+0.5)
	inv := ourGeo
	inv.Invert()
	lx, ly := inv.Apply(dx, dy)
	return int(math.Floor(lx)), int(math.Floor(ly))
}

// --- Screen shake ---------------------------------------------------------

// The original grenade/ducky/launcher explosions do `_root._x += shake`
// (random 20..49 px, random sign) and the camera then eases the root back by
// 1/8 per frame. Here the offset is applied only to the final image, detected
// from newly appeared fx_ex6 blasts, and uses its own random source so the
// simulation's random sequence is never consumed.
var screenShakeRand = rand.New(rand.NewSource(0x5eed))

type shakeBlastKey struct{ x, y int }

func (g *Game) updateScreenShake() {
	tick := ebiten.Tick()
	for g.shakeTick < tick {
		g.shakeX -= g.shakeX / 8
		if math.Abs(g.shakeX) < 0.5 {
			g.shakeX = 0
		}
		g.shakeTick++
	}
	if g.shakeTick > tick {
		g.shakeTick = tick
	}
	var current []shakeBlastKey
	for _, fx := range g.blastFX {
		// fx_ex6 starts at scale 1 and grows towards 2: a small one is new.
		if fx.Dead || fx.Kind != BlastEX6 || fx.ScaleX >= 1.5 {
			continue
		}
		key := shakeBlastKey{int(math.Round(fx.X)), int(math.Round(fx.Y))}
		current = append(current, key)
		seen := false
		for _, k := range g.shakeSeen {
			if k == key {
				seen = true
				break
			}
		}
		if !seen && g.screen == screenGameplay && g.displayScreenShake {
			shake := float64(screenShakeRand.Intn(30) + 20)
			if screenShakeRand.Intn(2) == 0 {
				shake = -shake
			}
			g.shakeX += shake
		}
	}
	g.shakeSeen = current
}

func (g *Game) screenShakeOffset() float64 {
	if !g.displayScreenShake || g.screen != screenGameplay {
		return 0
	}
	return math.Round(g.shakeX)
}

// --- Window, monitor, vsync ----------------------------------------------

// displayMonitors lists monitors only in the real game (unit tests have no
// window system).
func (g *Game) displayMonitors() (list []*ebiten.MonitorType) {
	if g == nil || !g.persistent {
		return nil
	}
	defer func() {
		if recover() != nil {
			list = nil
		}
	}()
	return ebiten.AppendMonitors(nil)
}

func (g *Game) currentMonitorInfo() (w, h int, dsf float64, ok bool) {
	if g == nil || !g.persistent {
		return 0, 0, 1, false
	}
	defer func() {
		if recover() != nil {
			w, h, dsf, ok = 0, 0, 1, false
		}
	}()
	m := ebiten.Monitor()
	if m == nil {
		return 0, 0, 1, false
	}
	w, h = m.Size()
	dsf = m.DeviceScaleFactor()
	if dsf <= 0 {
		dsf = 1
	}
	return w, h, dsf, w > 0 && h > 0
}

// displayPresetAvailable hides presets larger than the current monitor.
func (g *Game) displayPresetAvailable(i int) bool {
	mw, mh, _, ok := g.currentMonitorInfo()
	if !ok {
		return true
	}
	s := displayWindowPresets[i].scale
	return int(ScreenWidth*s) <= mw && int(ScreenHeight*s) <= mh
}

func displayPresetSize(i int) (int, int) {
	s := displayWindowPresets[i].scale
	return int(math.Round(ScreenWidth * s)), int(math.Round(ScreenHeight * s))
}

// displayPresetIndex returns the preset matching the windowed size, or -1.
func (g *Game) displayPresetIndex() int {
	for i := range displayWindowPresets {
		w, h := displayPresetSize(i)
		if w == g.windowedWidth && h == g.windowedHeight {
			return i
		}
	}
	return -1
}

// displaySizeLabel describes a window size, adding real device pixels on
// HiDPI monitors (Ebitengine window sizes are device-independent).
func (g *Game) displaySizeLabel(w, h int) string {
	_, _, dsf, _ := g.currentMonitorInfo()
	label := fmt.Sprintf("%d x %d", w, h)
	if dsf != 1 {
		label += fmt.Sprintf(" (%d x %d px at %d%% scaling)", int(math.Round(float64(w)*dsf)), int(math.Round(float64(h)*dsf)), int(math.Round(dsf*100)))
	}
	return label
}

func (g *Game) applyWindowSize(w, h int) {
	if w < displayMinWindowW || h < displayMinWindowH {
		return
	}
	if ebiten.IsFullscreen() {
		return
	}
	if g.displayPresetIndex() < 0 && g.windowedWidth >= displayMinWindowW {
		g.displayCustomW, g.displayCustomH = g.windowedWidth, g.windowedHeight
	}
	g.windowedWidth, g.windowedHeight = w, h
	if g.persistent {
		ebiten.SetWindowSize(w, h)
	}
}

func (g *Game) setFullscreen(on bool) {
	if ebiten.IsFullscreen() == on {
		return
	}
	g.toggleOptionsFullscreen()
}

func (g *Game) setVSync(on bool) {
	g.displayVSync = on
	if g.persistent {
		ebiten.SetVsyncEnabled(on)
	}
}

func (g *Game) setPowerSave(on bool) {
	g.displayPowerSave = on
	if g.persistent {
		ebiten.SetScreenClearedEveryFrame(!on)
	}
}

func (g *Game) setDisplayMonitor(i int) {
	monitors := g.displayMonitors()
	if i < 0 || i >= len(monitors) {
		return
	}
	g.displayMonitor = i
	ebiten.SetMonitor(monitors[i])
}

// powerSaveSkipDraw is true when nothing was updated since the last drawn
// frame; the previous frame stays on screen (screen clearing is off).
func (g *Game) powerSaveSkipDraw() bool {
	if !g.displayPowerSave || !g.persistent {
		return false
	}
	tick := ebiten.Tick()
	if g.powerSaveDrawn && g.powerSaveTick == tick {
		return true
	}
	g.powerSaveDrawn = true
	g.powerSaveTick = tick
	return false
}
