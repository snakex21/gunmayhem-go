package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestOptionsDevelopedHitboxes(t *testing.T) {
	tests := []struct {
		x, y float64
		want int
	}{
		{optionQualityButtons[0].rect.X + 10, optionQualityButtons[0].rect.Y + 10, optionHitQualityLow},
		{optionQualityButtons[1].rect.X + 10, optionQualityButtons[1].rect.Y + 10, optionHitQualityMedium},
		{optionQualityButtons[2].rect.X + 10, optionQualityButtons[2].rect.Y + 10, optionHitQualityHigh},
		{optionLoudnessRect.X + 10, optionLoudnessRect.Y + 10, optionHitLoudnessEQ},
		{optionFullscreenRect.X + 10, optionFullscreenRect.Y + 10, optionHitFullscreen},
		{optionResetSettingsRect.X + 10, optionResetSettingsRect.Y + 10, optionHitResetSettings},
		{optionResetControlsRect.X + 10, optionResetControlsRect.Y + 10, optionHitResetControls},
		{optionAllowCheatsRect.X + 10, optionAllowCheatsRect.Y + 10, optionHitAllowCheats},
		{optionBackRect.X + 10, optionBackRect.Y + 10, optionHitBack},
		{optionCheckUpdatesRect.X + 10, optionCheckUpdatesRect.Y + 10, optionHitCheckUpdates},
		{optionUpdateNowRect.X + 10, optionUpdateNowRect.Y + 10, optionHitUpdateNow},
	}
	for _, tc := range tests {
		if got := optionsHitAt(tc.x, tc.y); got != tc.want {
			t.Fatalf("options hit %.0f,%.0f=%d want%d", tc.x, tc.y, got, tc.want)
		}
	}
	if got := optionSliderHitAt(optionMusicSliderRect.X+20, optionMusicSliderRect.Y+5); got != optionSliderMusic {
		t.Fatalf("music slider hit=%d", got)
	}
	if got := optionSliderHitAt(optionSFXSliderRect.X+20, optionSFXSliderRect.Y+5); got != optionSliderSFX {
		t.Fatalf("sfx slider hit=%d", got)
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
	g.resetOptionSettings()
	if g.allowCheats {
		t.Fatal("RESET SETTINGS did not restore ALLOW CHEATS to OFF")
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

func TestDevelopedOptionsPanelRenders(t *testing.T) {
	g := New()
	screen := ebiten.NewImage(900, 600)
	g.drawDevelopedOptionSettings(screen)
}
