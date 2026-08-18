package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPersistenceDefaultsWithoutFiles(t *testing.T) {
	t.Setenv("GUNMAYHEM_DATA_DIR", t.TempDir())

	cfg, err := LoadAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	wantCfg := DefaultAppConfig()
	if !reflect.DeepEqual(cfg, wantCfg) {
		t.Fatalf("default config mismatch\ngot  %#v\nwant %#v", cfg, wantCfg)
	}

	save, err := LoadSaveData()
	if err != nil {
		t.Fatal(err)
	}
	wantSave := DefaultSaveData()
	if !reflect.DeepEqual(save, wantSave) {
		t.Fatalf("default save mismatch\ngot  %#v\nwant %#v", save, wantSave)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GUNMAYHEM_DATA_DIR", dir)

	cfg := DefaultAppConfig()
	cfg.WindowWidth = 1440
	cfg.WindowHeight = 900
	cfg.WindowX = 123
	cfg.WindowY = 234
	cfg.WindowPositionSaved = true
	cfg.Fullscreen = true
	cfg.MusicOn = false
	cfg.SoundOn = false
	cfg.Quality = 3
	cfg.Controls[0].Shoot = 90
	if err := SaveAppConfig(cfg); err != nil {
		t.Fatal(err)
	}

	got, err := LoadAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, cfg) {
		t.Fatalf("config round trip mismatch\ngot  %#v\nwant %#v", got, cfg)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json")); err != nil {
		t.Fatalf("config.json missing: %v", err)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GUNMAYHEM_DATA_DIR", dir)

	save := DefaultSaveData()
	save.CampaignLevels[1] = 2
	save.CampaignLevels[2] = 1
	save.CampaignGuns[40] = true
	save.CampaignGuns[52] = true
	if err := SaveGameData(save); err != nil {
		t.Fatal(err)
	}

	got, err := LoadSaveData()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, save) {
		t.Fatalf("save round trip mismatch\ngot  %#v\nwant %#v", got, save)
	}
	if _, err := os.Stat(filepath.Join(dir, "save.json")); err != nil {
		t.Fatalf("save.json missing: %v", err)
	}
}

func TestNewPersistentAppliesSettingsAndProgress(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.MusicOn = false
	cfg.SoundOn = false
	cfg.Quality = 3
	cfg.Controls[1].Shoot = 88

	save := DefaultSaveData()
	save.CampaignLevels[3] = 2
	save.CampaignGuns[41] = true

	g := NewPersistent(cfg, save)
	if !g.persistent || g.musicOn || g.soundOn || g.quality != 3 {
		t.Fatalf("persistent settings not applied: persistent=%v music=%v sound=%v quality=%d", g.persistent, g.musicOn, g.soundOn, g.quality)
	}
	if g.controlConfigs[1].Shoot != 88 || g.players[1].Controls.Shoot != 88 {
		t.Fatalf("controls not applied: config=%d player=%d", g.controlConfigs[1].Shoot, g.players[1].Controls.Shoot)
	}
	if g.campaignLevels[3] != 2 || !g.campaignGuns[41] {
		t.Fatalf("progress not applied: level4=%d gun41=%v", g.campaignLevels[3], g.campaignGuns[41])
	}
}
