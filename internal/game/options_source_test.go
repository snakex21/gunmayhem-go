package game

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// optionsFrame runs one update pass of the kit Options screen.
func optionsFrame(g *Game, in uiInput) { g.runOptionsUpdate(in) }

func optionsItem(t *testing.T, g *Game, id string) uiItem {
	t.Helper()
	it, ok := g.optionsUI.item(id)
	if !ok {
		t.Fatalf("options item %q not registered", id)
	}
	return it
}

// optionsClickAt performs move, press and release at (x,y).
func optionsClickAt(g *Game, x, y float64) {
	optionsFrame(g, uiInput{X: x, Y: y, Moved: true})
	optionsFrame(g, uiInput{X: x, Y: y, Pressed: true, Down: true})
	optionsFrame(g, uiInput{X: x, Y: y, Released: true})
}

func optionsClick(t *testing.T, g *Game, id string) {
	t.Helper()
	it := optionsItem(t, g, id)
	optionsClickAt(g, it.rect.X+it.rect.W/2, it.rect.Y+it.rect.H/2)
}

func newOptionsTestGame() *Game {
	g := New()
	g.screen = screenOptions
	optionsFrame(g, uiInput{}) // populate the registry
	return g
}

func TestOptionsTabsSwitchByClickAndKeys(t *testing.T) {
	g := newOptionsTestGame()
	for i, label := range optionsTabLabels {
		optionsClick(t, g, "opt.tabs/"+label)
		if g.optionsTab != i {
			t.Fatalf("click %s: tab=%d want %d", label, g.optionsTab, i)
		}
	}
	optionsFrame(g, uiInput{NextTab: true})
	if g.optionsTab != optionsTabGame {
		t.Fatalf("E from CONTROLS should wrap to GAME, got %d", g.optionsTab)
	}
	optionsFrame(g, uiInput{PrevTab: true})
	if g.optionsTab != optionsTabControls {
		t.Fatalf("Q from GAME should wrap to CONTROLS, got %d", g.optionsTab)
	}
}

// Every interactive Options control is drawn and hit-tested from the same
// registered rect: hovering the centre of each item makes exactly that item hot.
func TestOptionsEveryControlIsHotAtItsDrawnRect(t *testing.T) {
	g := newOptionsTestGame()
	for tab := range optionsTabLabels {
		g.optionsTab = tab
		optionsFrame(g, uiInput{})
		for _, it := range append([]uiItem(nil), g.optionsUI.items...) {
			x, y := it.rect.X+it.rect.W/2, it.rect.Y+it.rect.H/2
			optionsFrame(g, uiInput{X: x, Y: y, Moved: true})
			if g.optionsUI.hot != it.id {
				t.Fatalf("tab %d: centre of %q is hot=%q", tab, it.id, g.optionsUI.hot)
			}
		}
	}
}

func TestOptionsDisplayControlsApply(t *testing.T) {
	g := newOptionsTestGame()
	optionsClick(t, g, "opt.tabs/DISPLAY")

	q := optionsItem(t, g, "opt.quality")
	optionsClickAt(g, q.rect.X+q.rect.W/6, q.rect.Y+10) // LOW segment
	if g.quality != 1 || sourceRenderQuality != 1 {
		t.Fatalf("LOW click: quality=%d source=%d", g.quality, sourceRenderQuality)
	}
	optionsClickAt(g, q.rect.X+q.rect.W*5/6, q.rect.Y+10) // HIGH segment
	if g.quality != 3 {
		t.Fatalf("HIGH click: quality=%d", g.quality)
	}

	s := optionsItem(t, g, "opt.scaling")
	optionsClickAt(g, s.rect.X+s.rect.W*5/6, s.rect.Y+10)
	if g.displayScaling != displayScalingStretch {
		t.Fatalf("scaling=%d want STRETCH", g.displayScaling)
	}
	f := optionsItem(t, g, "opt.filter")
	optionsClickAt(g, f.rect.X+f.rect.W*7/8, f.rect.Y+10)
	if g.displayFilter != displayFilterSmooth {
		t.Fatalf("filter=%d want SMOOTH", g.displayFilter)
	}
	optionsClick(t, g, "opt.shake")
	if g.displayScreenShake {
		t.Fatal("screen shake toggle did not switch off")
	}
	fps := optionsItem(t, g, "opt.fps")
	optionsClickAt(g, fps.rect.X+fps.rect.W*7/10, fps.rect.Y+10)
	if g.displayFrameRate != displayFrameRate144 {
		t.Fatalf("frame rate=%d want 144", g.displayFrameRate)
	}
	optionsClick(t, g, "opt.smooth")
	if g.displaySmoothMotion {
		t.Fatal("smooth motion toggle did not switch off")
	}
	optionsClick(t, g, "opt.power")
	if !g.displayPowerSave {
		t.Fatal("power save toggle did not switch on")
	}

	g.resetDisplaySettings()
	if g.quality != 2 || g.displayScaling != displayScalingFit || g.displayFilter != displayFilterAuto ||
		!g.displayScreenShake || g.displayFrameRate != displayFrameRateVSync || !g.displaySmoothMotion || g.displayPowerSave {
		t.Fatalf("display reset incomplete: q=%d scale=%d filter=%d shake=%v fps=%v smooth=%v power=%v",
			g.quality, g.displayScaling, g.displayFilter, g.displayScreenShake, g.displayFrameRate, g.displaySmoothMotion, g.displayPowerSave)
	}
	setSourceRenderQuality(2)
}

func TestOptionsKeyboardNavigationChangesFocusedSetting(t *testing.T) {
	g := newOptionsTestGame()
	g.optionsTab = optionsTabDisplay
	optionsFrame(g, uiInput{})
	g.optionsUI.focus = "opt.scaling"
	optionsFrame(g, uiInput{Right: true})
	if g.displayScaling != displayScalingInteger {
		t.Fatalf("Right on SCALING: %d", g.displayScaling)
	}
	optionsFrame(g, uiInput{DownKey: true})
	if g.optionsUI.focus != "opt.filter" {
		t.Fatalf("Down from SCALING focused %q", g.optionsUI.focus)
	}
	optionsFrame(g, uiInput{DownKey: true})
	optionsFrame(g, uiInput{Left: true})
	if g.quality != 1 {
		t.Fatalf("Left on QUALITY: %d", g.quality)
	}
	setSourceRenderQuality(2)
}

func TestOptionsAudioSliderDragAndKeys(t *testing.T) {
	g := newOptionsTestGame()
	g.optionsTab = optionsTabAudio
	optionsFrame(g, uiInput{})
	it := optionsItem(t, g, "opt.music")
	track := Rect{X: it.rect.X + 10, W: it.rect.W - 20}
	x := track.X + track.W*0.25
	y := it.rect.Y + it.rect.H/2
	optionsFrame(g, uiInput{X: x, Y: y, Moved: true})
	optionsFrame(g, uiInput{X: x, Y: y, Pressed: true, Down: true})
	optionsFrame(g, uiInput{X: x + track.W*0.25, Y: y, Moved: true, Down: true}) // drag to 50%
	optionsFrame(g, uiInput{X: x + track.W*0.25, Y: y, Released: true})
	if math.Abs(g.musicVolume-0.5) > 0.01 || !g.musicOn {
		t.Fatalf("music after drag=%.3f on=%v", g.musicVolume, g.musicOn)
	}
	g.optionsUI.focus = "opt.sfx"
	g.soundVolume = 0.05
	optionsFrame(g, uiInput{Left: true})
	if g.soundVolume != 0 || g.soundOn {
		t.Fatalf("Left on SFX at 5%%: %.2f on=%v", g.soundVolume, g.soundOn)
	}
	optionsClick(t, g, "opt.eq")
	if !g.loudnessEQ {
		t.Fatal("loudness EQ toggle")
	}
}

func TestOptionsGameTabCheatsAndBack(t *testing.T) {
	g := newOptionsTestGame()
	g.showCollisions = true
	optionsClick(t, g, "opt.cheats")
	if !g.allowCheats {
		t.Fatal("cheats toggle did not switch on")
	}
	optionsClick(t, g, "opt.cheats")
	if g.allowCheats || g.showCollisions {
		t.Fatal("cheats off must also clear debug collisions")
	}
	if it := optionsItem(t, g, "opt.update"); !it.disabled {
		t.Fatal("UPDATE NOW must be disabled without an available update")
	}
	optionsFrame(g, uiInput{Esc: true})
	if !g.fadeActive {
		t.Fatal("ESC did not leave Options")
	}
}

func TestOptionsKeyCaptureModalBlocksPage(t *testing.T) {
	g := newOptionsTestGame()
	g.optionsTab = optionsTabControls
	optionsFrame(g, uiInput{})
	optionsClick(t, g, "opt.key.1.6")
	if g.optionsKeyCapturePlayer != 2 || g.optionsKeyCaptureIndex != 6 {
		t.Fatalf("capture p=%d i=%d", g.optionsKeyCapturePlayer, g.optionsKeyCaptureIndex)
	}
	optionsFrame(g, uiInput{})
	// The page below the modal is inert: clicking a tab does nothing.
	tab := optionsItem(t, g, "opt.tabs/GAME")
	optionsClickAt(g, tab.rect.X+5, tab.rect.Y+5)
	if g.optionsTab != optionsTabControls || g.optionsKeyCapturePlayer != 2 {
		t.Fatalf("modal leaked a click: tab=%d capture=%d", g.optionsTab, g.optionsKeyCapturePlayer)
	}
	g.assignOptionsKey(90)
	if g.controlConfigs[1].Ability != 90 || g.optionsKeyCapturePlayer != 0 {
		t.Fatalf("assign: ability=%d capture=%d", g.controlConfigs[1].Ability, g.optionsKeyCapturePlayer)
	}
	optionsClick(t, g, "opt.key.0.0")
	optionsFrame(g, uiInput{})
	optionsClick(t, g, "opt.capture.cancel")
	if g.optionsKeyCapturePlayer != 0 {
		t.Fatal("CANCEL did not close key capture")
	}
}

func TestManualUpdateCheckExplainsDevBuild(t *testing.T) {
	g := New()
	g.beginManualUpdateCheck()
	if got := g.settingsUpdateStatus(); got != "DEV BUILD - UPDATE CHECKS ARE ENABLED IN TAGGED RELEASES" {
		t.Fatalf("dev update status=%q", got)
	}
}

func TestOptionSliderValueClamps(t *testing.T) {
	r := Rect{X: 100, Y: 0, W: 200, H: 20}
	if got := optionSliderValueFromX(r, 50); got != 0 {
		t.Fatalf("left clamp=%v", got)
	}
	if got := optionSliderValueFromX(r, 200); got != 0.5 {
		t.Fatalf("middle=%v", got)
	}
	if got := optionSliderValueFromX(r, 400); got != 1 {
		t.Fatalf("right clamp=%v", got)
	}
}

func TestResetOptionControlsRestoresOriginalBindings(t *testing.T) {
	g := New()
	g.controlConfigs[0].Shoot = 90
	g.controlConfigs[3].Grenade = 65
	g.resetOptionControls()
	for i := range g.controlConfigs {
		want := OriginalControls(i + 1)
		if g.controlConfigs[i] != want {
			t.Fatalf("player %d controls=%+v want %+v", i+1, g.controlConfigs[i], want)
		}
	}
}

func TestResetOptionSettingsDisablesCheats(t *testing.T) {
	g := New()
	g.allowCheats = true
	g.displayScaling = displayScalingInteger
	g.musicVolume = 0.2
	g.resetOptionSettings()
	if g.allowCheats {
		t.Fatal("RESET SETTINGS did not restore ALLOW CHEATS to OFF")
	}
	if g.displayScaling != displayScalingFit || g.musicVolume != 1 {
		t.Fatalf("RESET SETTINGS left scaling=%d music=%.2f", g.displayScaling, g.musicVolume)
	}
}

func TestLoudnessEQNarrowsSFXDynamicRange(t *testing.T) {
	g := New()
	g.soundVolume = 1
	g.soundOn = true
	plainQuiet := g.sourceSFXVolume(false)
	plainLoud := g.sourceSFXVolume(true)
	g.loudnessEQ = true
	eqQuiet := g.sourceSFXVolume(false)
	eqLoud := g.sourceSFXVolume(true)
	if plainLoud-plainQuiet <= eqLoud-eqQuiet {
		t.Fatalf("EQ did not narrow range: plain %.2f..%.2f eq %.2f..%.2f", plainQuiet, plainLoud, eqQuiet, eqLoud)
	}
}

func TestOptionsAllTabsRender(t *testing.T) {
	g := New()
	g.screen = screenOptions
	screen := ebiten.NewImage(900, 600)
	for tab := range optionsTabLabels {
		g.optionsTab = tab
		optionsFrame(g, uiInput{})
		g.drawOptionsMenu(screen)
	}
	g.optionsKeyCapturePlayer, g.optionsKeyCaptureIndex = 1, 0
	g.drawOptionsMenu(screen)
}
