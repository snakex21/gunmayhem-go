package game

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestSourceRenderQualityMapsFlashLevels(t *testing.T) {
	for _, tc := range []struct {
		quality        int
		filter         ebiten.Filter
		disableMipmaps bool
	}{
		{1, ebiten.FilterNearest, false},
		{2, ebiten.FilterLinear, true},
		{3, ebiten.FilterLinear, false},
	} {
		setSourceRenderQuality(tc.quality)
		op := &ebiten.DrawImageOptions{}
		applySourceRenderQuality(op)
		if op.Filter != tc.filter || op.DisableMipmaps != tc.disableMipmaps {
			t.Fatalf("quality %d => filter=%v disableMipmaps=%v; want %v/%v", tc.quality, op.Filter, op.DisableMipmaps, tc.filter, tc.disableMipmaps)
		}
	}
	setSourceRenderQuality(2)
}

// QUALITY must visibly do something: LOW drops the original's optional
// explosion particles (draw-time) and keeps a sharp upscale; HIGH upscales
// smoothly like Flash.
func TestQualityProfileLevels(t *testing.T) {
	low, med, high := renderProfileForQuality(1), renderProfileForQuality(2), renderProfileForQuality(3)
	if low.blastParticles || !med.blastParticles || !high.blastParticles {
		t.Fatalf("blast particles low=%v med=%v high=%v", low.blastParticles, med.blastParticles, high.blastParticles)
	}
	if low.finalFilter != ebiten.FilterNearest || med.finalFilter != ebiten.FilterPixelated || high.finalFilter != ebiten.FilterLinear {
		t.Fatalf("final filters low=%v med=%v high=%v", low.finalFilter, med.finalFilter, high.finalFilter)
	}
	// Exactly the ten `_quality != "LOW"` particles of the grenade stack are
	// optional; the four core blast clips always draw.
	optional := 0
	for _, fx := range spawnGrenadeBlastEffects(100, 100) {
		if blastKindIsQualityParticle(fx.Kind) {
			optional++
		}
	}
	if optional != 10 {
		t.Fatalf("optional particles=%d want 10", optional)
	}
}

func TestDisplayFinalFilterSetting(t *testing.T) {
	cases := []struct {
		setting, quality int
		scale            float64
		want             ebiten.Filter
	}{
		{displayFilterAuto, 1, 1.5, ebiten.FilterNearest},
		{displayFilterAuto, 2, 1.5, ebiten.FilterPixelated},
		{displayFilterAuto, 3, 1.5, ebiten.FilterLinear},
		{displayFilterSharp, 3, 2, ebiten.FilterNearest},
		{displayFilterPixelated, 1, 2, ebiten.FilterPixelated},
		{displayFilterSmooth, 1, 2, ebiten.FilterLinear},
		{displayFilterSharp, 1, 0.5, ebiten.FilterLinear}, // never point-sample a downscale
	}
	for _, c := range cases {
		if got := displayFinalFilter(c.setting, c.quality, c.scale); got != c.want {
			t.Fatalf("filter(%d,q%d,%.1f)=%v want %v", c.setting, c.quality, c.scale, got, c.want)
		}
	}
}

func TestDisplayFinalGeoMModes(t *testing.T) {
	// 1600x1000 final screen: Ebitengine's fit is scale 1.6667, x offset 50.
	var fit ebiten.GeoM
	fit.Scale(1000.0/600, 1000.0/600)
	fit.Translate(50, 0)
	if got := displayFinalGeoM(displayScalingFit, 1600, 1000, fit); got != fit {
		t.Fatal("FIT must use Ebitengine's transform unchanged")
	}
	in := displayFinalGeoM(displayScalingInteger, 1600, 1000, fit)
	if in.Element(0, 0) != 1 || in.Element(1, 1) != 1 || in.Element(0, 2) != 350 || in.Element(1, 2) != 200 {
		t.Fatalf("INTEGER geo=%v", in.String())
	}
	in2 := displayFinalGeoM(displayScalingInteger, 1920, 1200, fit)
	if in2.Element(0, 0) != 2 || in2.Element(0, 2) != 60 || in2.Element(1, 2) != 0 {
		t.Fatalf("INTEGER 1920x1200 geo=%v", in2.String())
	}
	if small := displayFinalGeoM(displayScalingInteger, 800, 500, fit); small != fit {
		t.Fatal("INTEGER below 1x must fall back to FIT")
	}
	st := displayFinalGeoM(displayScalingStretch, 1600, 1000, fit)
	if x, y := st.Apply(900, 600); x != 1600 || y != 1000 {
		t.Fatalf("STRETCH corner=(%v,%v)", x, y)
	}
}

// The single cursor helper maps Ebitengine's fit-based cursor into our own
// final transform; in FIT the two are identical.
func TestRemapCursorThroughScalingModes(t *testing.T) {
	var fit ebiten.GeoM
	fit.Scale(1000.0/600, 1000.0/600)
	fit.Translate(50, 0)
	// A device pixel at (750,500) is logical (420,300) under FIT.
	ex, ey := 420, 300
	if x, y := remapCursor(ex, ey, fit, fit); x != ex || y != ey {
		t.Fatalf("FIT remap=(%d,%d)", x, y)
	}
	integer := displayFinalGeoM(displayScalingInteger, 1600, 1000, fit) // 1x at (350,200)
	if x, y := remapCursor(ex, ey, fit, integer); x != 400 || y != 300 {
		t.Fatalf("INTEGER remap=(%d,%d) want (400,300)", x, y)
	}
	stretch := displayFinalGeoM(displayScalingStretch, 1600, 1000, fit)
	if x, y := remapCursor(ex, ey, fit, stretch); x != 422 || y != 300 {
		t.Fatalf("STRETCH remap=(%d,%d) want (422,300)", x, y)
	}
	g := New()
	g.displayScaling = displayScalingFit
	g.finalGeoValid = true
	g.finalEbitenGeo, g.finalGeo = fit, integer
	if x, y := g.cursorPos(); x != 0 || y != 0 {
		t.Fatalf("FIT cursorPos must equal ebiten.CursorPosition, got (%d,%d)", x, y)
	}
}

// Display and quality settings are local presentation: they must never change
// the simulation or what the host sends to clients.
func TestDisplayAndQualitySettingsAreLocalOnly(t *testing.T) {
	g := New()
	g.screen = screenGameplay
	g.blastFX = spawnGrenadeBlastEffects(300, 200)
	screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
	ref := g.makeNetSnapshot()
	for _, s := range []struct{ quality, scaling, filter int }{
		{1, displayScalingInteger, displayFilterSharp},
		{3, displayScalingStretch, displayFilterSmooth},
		{2, displayScalingFit, displayFilterAuto},
	} {
		g.setQuality(s.quality)
		g.displayScaling = s.scaling
		g.displayFilter = s.filter
		g.displayScreenShake = !g.displayScreenShake
		g.displayFrameRate = (g.displayFrameRate + 1) % displayFrameRateCount
		g.displaySmoothMotion = !g.displaySmoothMotion
		// Rendering (incl. the LOW particle cull and shake detection) must not
		// mutate any simulated or replicated state.
		g.Draw(screen)
		if len(g.blastFX) != 14 {
			t.Fatalf("quality %d changed BlastFX count to %d", s.quality, len(g.blastFX))
		}
		if got := g.makeNetSnapshot(); !reflect.DeepEqual(ref, got) {
			t.Fatalf("quality %d / scaling %d / filter %d changed the netplay snapshot", s.quality, s.scaling, s.filter)
		}
	}
	setSourceRenderQuality(2)

	// No snapshot field may carry a display/quality setting.
	var walk func(reflect.Type, string, map[reflect.Type]bool)
	walk = func(tp reflect.Type, path string, seen map[reflect.Type]bool) {
		for tp.Kind() == reflect.Slice || tp.Kind() == reflect.Array || tp.Kind() == reflect.Pointer || tp.Kind() == reflect.Map {
			tp = tp.Elem()
		}
		if tp.Kind() != reflect.Struct || seen[tp] {
			return
		}
		seen[tp] = true
		for i := 0; i < tp.NumField(); i++ {
			f := tp.Field(i)
			name := strings.ToLower(f.Name)
			for _, banned := range []string{"quality", "scaling", "shake", "vsync", "powersave", "filter"} {
				if strings.Contains(name, banned) {
					t.Fatalf("netSnapshot field %s.%s looks like a local display setting", path, f.Name)
				}
			}
			walk(f.Type, path+"."+f.Name, seen)
		}
	}
	walk(reflect.TypeOf(netSnapshot{}), "netSnapshot", map[reflect.Type]bool{})
}

func TestScreenShakeUsesItsOwnRandomSource(t *testing.T) {
	g := New()
	g.screen = screenGameplay
	g.blastFX = []BlastEffect{newBlastEffect(BlastEX6, 100, 100)}
	g.updateScreenShake()
	if g.shakeX == 0 || math.Abs(g.shakeX) < 20 || math.Abs(g.shakeX) > 49 {
		t.Fatalf("new explosion shake=%v want 20..49 px", g.shakeX)
	}
	first := g.shakeX
	g.updateScreenShake() // the same blast must not shake twice
	if g.shakeX != first {
		t.Fatalf("same explosion shook again: %v -> %v", first, g.shakeX)
	}
	g.displayScreenShake = false
	if g.screenShakeOffset() != 0 {
		t.Fatal("SCREEN SHAKE off must not offset the frame")
	}
}
