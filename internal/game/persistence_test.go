package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPersistenceDefaultsWithoutFiles(t *testing.T) {
	t.Setenv("GUNMAYHEM_DATA_DIR", projectTestDir(t))

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
	dir := projectTestDir(t)
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
	cfg.MusicVolume = 0
	cfg.SoundVolume = 0
	cfg.LoudnessEQ = true
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
	dir := projectTestDir(t)
	t.Setenv("GUNMAYHEM_DATA_DIR", dir)

	save := DefaultSaveData()
	save.CampaignLevels[1] = 2
	save.CampaignLevels[2] = 1
	save.CampaignGuns[40] = true
	save.CampaignGuns[52] = true
	save.GM2CampaignLevels[1] = 2
	save.GM2CampaignLevels[2] = 1
	save.GM2Challenges[3] = GM2ChallengeProgress{Medal: 2, BestTimeMS: 27890}
	save.ExtraCampaignLevels[0] = 2
	save.ExtraCampaignLevels[1] = 1
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
	cfg.MusicVolume = 0
	cfg.SoundVolume = 0
	cfg.LoudnessEQ = true
	cfg.Quality = 3
	cfg.Controls[1].Shoot = 88

	save := DefaultSaveData()
	save.CampaignLevels[3] = 2
	save.CampaignGuns[41] = true
	save.GM2CampaignLevels[4] = 2
	save.GM2Challenges[1] = GM2ChallengeProgress{Medal: 3, BestTimeMS: 19876}
	save.ExtraCampaignLevels[2] = 2

	g := NewPersistent(cfg, save)
	if !g.persistent || g.musicOn || g.soundOn || g.musicVolume != 0 || g.soundVolume != 0 || !g.loudnessEQ || g.quality != 3 {
		t.Fatalf("persistent settings not applied: persistent=%v music=%v/%v sound=%v/%v eq=%v quality=%d", g.persistent, g.musicOn, g.musicVolume, g.soundOn, g.soundVolume, g.loudnessEQ, g.quality)
	}
	if g.controlConfigs[1].Shoot != 88 || g.players[1].Controls.Shoot != 88 {
		t.Fatalf("controls not applied: config=%d player=%d", g.controlConfigs[1].Shoot, g.players[1].Controls.Shoot)
	}
	if g.campaignLevels[3] != 2 || !g.campaignGuns[41] {
		t.Fatalf("GM1 progress not applied: level4=%d gun41=%v", g.campaignLevels[3], g.campaignGuns[41])
	}
	if g.gm2CampaignLevels[4] != 2 || g.gm2Challenges[1].Medal != 3 || g.gm2Challenges[1].BestTimeMS != 19876 {
		t.Fatalf("GM2 progress not applied: levels=%v challenge=%+v", g.gm2CampaignLevels, g.gm2Challenges[1])
	}
	if g.extraCampaignLevels[2] != 2 {
		t.Fatalf("EXTRA progress not applied: %v", g.extraCampaignLevels)
	}
}

func TestLoadV2ConfigMigratesAudioTogglesToSliders(t *testing.T) {
	dir := projectTestDir(t)
	t.Setenv("GUNMAYHEM_DATA_DIR", dir)
	legacy := `{
  "version": 2,
  "window_width": 900,
  "window_height": 600,
  "music_on": false,
  "sound_on": true,
  "quality": 2
}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != persistenceVersion || got.MusicVolume != 0 || got.SoundVolume != 1 || got.MusicOn || !got.SoundOn {
		t.Fatalf("legacy audio migration failed: %+v", got)
	}
}

func TestLoadV1SaveMigratesGM2Defaults(t *testing.T) {
	dir := projectTestDir(t)
	t.Setenv("GUNMAYHEM_DATA_DIR", dir)
	legacy := `{
  "version": 1,
  "campaign_levels": [1,2,1,0,0,0,0,0,0,0],
  "campaign_guns": [true]
}`
	if err := os.WriteFile(filepath.Join(dir, "save.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSaveData()
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != persistenceVersion {
		t.Fatalf("version=%d want %d", got.Version, persistenceVersion)
	}
	if got.CampaignLevels[1] != 2 || got.CampaignLevels[2] != 1 {
		t.Fatalf("GM1 progress was lost during migration: %v", got.CampaignLevels)
	}
	if got.GM2CampaignLevels[0] != 1 || got.GM2CampaignLevels[1] != 1 {
		t.Fatalf("GM2 fresh defaults not initialized: %v", got.GM2CampaignLevels)
	}
	if got.ExtraCampaignLevels[0] != 1 {
		t.Fatalf("EXTRA fresh defaults not initialized: %v", got.ExtraCampaignLevels)
	}
}

func TestLoadV4SaveUnlocksExpandedExtraMissionNine(t *testing.T) {
	dir := projectTestDir(t)
	t.Setenv("GUNMAYHEM_DATA_DIR", dir)
	legacy := `{
  "version": 4,
  "extra_campaign_levels": [2,2,2,2,2,2,2,2]
}`
	if err := os.WriteFile(filepath.Join(dir, "save.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSaveData()
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != persistenceVersion || got.ExtraCampaignLevels[7] != 2 || got.ExtraCampaignLevels[8] != 1 {
		t.Fatalf("expanded EXTRA migration failed: version=%d levels=%v", got.Version, got.ExtraCampaignLevels)
	}
}

func TestLoadV5SaveUnlocksExpandedExtraMissionThirteen(t *testing.T) {
	dir := projectTestDir(t)
	t.Setenv("GUNMAYHEM_DATA_DIR", dir)
	legacy := `{
  "version": 5,
  "extra_campaign_levels": [2,2,2,2,2,2,2,2,2,2,2,2]
}`
	if err := os.WriteFile(filepath.Join(dir, "save.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSaveData()
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != persistenceVersion || got.ExtraCampaignLevels[11] != 2 || got.ExtraCampaignLevels[12] != 1 {
		t.Fatalf("20-mission EXTRA migration failed: version=%d levels=%v", got.Version, got.ExtraCampaignLevels)
	}
}
