package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

const persistenceVersion = 1

// AppConfig contains machine/user preferences. It is deliberately separate
// from campaign progress so settings can evolve without touching save data.
type AppConfig struct {
	Version             int         `json:"version"`
	WindowWidth         int         `json:"window_width"`
	WindowHeight        int         `json:"window_height"`
	WindowX             int         `json:"window_x"`
	WindowY             int         `json:"window_y"`
	WindowPositionSaved bool        `json:"window_position_saved"`
	Fullscreen          bool        `json:"fullscreen"`
	MusicOn             bool        `json:"music_on"`
	SoundOn             bool        `json:"sound_on"`
	Quality             int         `json:"quality"`
	Controls            [4]Controls `json:"controls"`
}

// SaveData contains game progress only. Future GM2/custom campaign data can be
// added here without mixing it with machine-specific settings.
type SaveData struct {
	Version        int      `json:"version"`
	CampaignLevels [10]int  `json:"campaign_levels"`
	CampaignGuns   [57]bool `json:"campaign_guns"`
}

func DefaultAppConfig() AppConfig {
	cfg := AppConfig{
		Version:      persistenceVersion,
		WindowWidth:  ScreenWidth,
		WindowHeight: ScreenHeight,
		MusicOn:      true,
		SoundOn:      true,
		Quality:      2,
	}
	for i := range cfg.Controls {
		cfg.Controls[i] = OriginalControls(i + 1)
	}
	return cfg
}

func defaultCampaignState() ([10]int, [57]bool) {
	var levels [10]int
	var guns [57]bool
	levels[0] = 1
	levels[1] = 1
	for i := range guns {
		guns[i] = true
	}
	for _, i := range []int{18, 19, 20, 21, 22, 29, 30, 31, 32, 33, 40, 41, 42, 43, 44, 52, 53, 54, 55, 56} {
		guns[i] = false
	}
	return levels, guns
}

func DefaultSaveData() SaveData {
	levels, guns := defaultCampaignState()
	return SaveData{
		Version:        persistenceVersion,
		CampaignLevels: levels,
		CampaignGuns:   guns,
	}
}

func persistentDataDir() (string, error) {
	if dir := os.Getenv("GUNMAYHEM_DATA_DIR"); dir != "" {
		return dir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "GunMayhemRE"), nil
}

func persistentPath(name string) (string, error) {
	dir, err := persistentDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

func loadJSON(path string, dst any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("decode %s: %w", filepath.Base(path), err)
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	// Windows does not consistently allow replacing an existing file via
	// Rename, so remove the old tiny JSON only after the replacement is ready.
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func normalizeConfig(cfg AppConfig) AppConfig {
	defaults := DefaultAppConfig()
	cfg.Version = persistenceVersion
	if cfg.WindowWidth < 480 || cfg.WindowWidth > 7680 {
		cfg.WindowWidth = defaults.WindowWidth
	}
	if cfg.WindowHeight < 270 || cfg.WindowHeight > 4320 {
		cfg.WindowHeight = defaults.WindowHeight
	}
	if cfg.Quality < 1 || cfg.Quality > 3 {
		cfg.Quality = defaults.Quality
	}
	for i := range cfg.Controls {
		c := cfg.Controls[i]
		if c.Up <= 0 || c.Left <= 0 || c.Down <= 0 || c.Right <= 0 || c.Shoot <= 0 || c.Grenade <= 0 {
			cfg.Controls[i] = defaults.Controls[i]
		}
	}
	return cfg
}

func normalizeSave(save SaveData) SaveData {
	defaults := DefaultSaveData()
	save.Version = persistenceVersion
	for i, state := range save.CampaignLevels {
		if state < 0 || state > 2 {
			save.CampaignLevels[i] = defaults.CampaignLevels[i]
		}
	}
	// Level 1 and 2 are available from a fresh game and should never become
	// inaccessible due to a damaged/old save file.
	if save.CampaignLevels[0] == 0 {
		save.CampaignLevels[0] = 1
	}
	if save.CampaignLevels[1] == 0 {
		save.CampaignLevels[1] = 1
	}
	return save
}

func LoadAppConfig() (AppConfig, error) {
	cfg := DefaultAppConfig()
	path, err := persistentPath("config.json")
	if err != nil {
		return cfg, err
	}
	if err := loadJSON(path, &cfg); err != nil {
		return DefaultAppConfig(), err
	}
	return normalizeConfig(cfg), nil
}

func LoadSaveData() (SaveData, error) {
	save := DefaultSaveData()
	path, err := persistentPath("save.json")
	if err != nil {
		return save, err
	}
	if err := loadJSON(path, &save); err != nil {
		return DefaultSaveData(), err
	}
	return normalizeSave(save), nil
}

func SaveAppConfig(cfg AppConfig) error {
	path, err := persistentPath("config.json")
	if err != nil {
		return err
	}
	return writeJSON(path, normalizeConfig(cfg))
}

func SaveGameData(save SaveData) error {
	path, err := persistentPath("save.json")
	if err != nil {
		return err
	}
	return writeJSON(path, normalizeSave(save))
}

// ApplyWindowConfig must be called before RunGame. The logical game surface
// remains 900x600; only the desktop window changes size.
func ApplyWindowConfig(cfg AppConfig) {
	cfg = normalizeConfig(cfg)
	ebiten.SetWindowSize(cfg.WindowWidth, cfg.WindowHeight)
	if cfg.WindowPositionSaved {
		ebiten.SetWindowPosition(cfg.WindowX, cfg.WindowY)
	}
	ebiten.SetFullscreen(cfg.Fullscreen)
}

func NewPersistent(cfg AppConfig, save SaveData) *Game {
	g := New()
	cfg = normalizeConfig(cfg)
	save = normalizeSave(save)
	g.persistent = true
	g.windowedWidth = cfg.WindowWidth
	g.windowedHeight = cfg.WindowHeight
	g.windowedX = cfg.WindowX
	g.windowedY = cfg.WindowY
	g.windowPositionSaved = cfg.WindowPositionSaved
	g.musicOn = cfg.MusicOn
	g.soundOn = cfg.SoundOn
	g.quality = cfg.Quality
	g.controlConfigs = cfg.Controls
	g.players[0].Controls = g.controlConfigs[0]
	g.players[1].Controls = g.controlConfigs[1]
	g.campaignLevels = save.CampaignLevels
	g.campaignGuns = save.CampaignGuns
	setSourceRenderQuality(g.quality)
	ebiten.SetScreenFilterEnabled(g.quality != 1)
	return g
}

func (g *Game) captureWindowedState() {
	if g == nil || !g.persistent || ebiten.IsFullscreen() {
		return
	}
	w, h := ebiten.WindowSize()
	x, y := ebiten.WindowPosition()
	if w >= 480 && h >= 270 {
		g.windowedWidth = w
		g.windowedHeight = h
	}
	g.windowedX = x
	g.windowedY = y
	g.windowPositionSaved = true
}

func (g *Game) currentAppConfig() AppConfig {
	cfg := DefaultAppConfig()
	cfg.MusicOn = g.musicOn
	cfg.SoundOn = g.soundOn
	cfg.Quality = g.quality
	cfg.Controls = g.controlConfigs
	if g.persistent {
		g.captureWindowedState()
		cfg.WindowWidth = g.windowedWidth
		cfg.WindowHeight = g.windowedHeight
		cfg.WindowX = g.windowedX
		cfg.WindowY = g.windowedY
		cfg.WindowPositionSaved = g.windowPositionSaved
		cfg.Fullscreen = ebiten.IsFullscreen()
	}
	return cfg
}

func (g *Game) currentSaveData() SaveData {
	return SaveData{
		Version:        persistenceVersion,
		CampaignLevels: g.campaignLevels,
		CampaignGuns:   g.campaignGuns,
	}
}

func (g *Game) saveConfig() error {
	if g == nil || !g.persistent {
		return nil
	}
	return SaveAppConfig(g.currentAppConfig())
}

func (g *Game) saveProgress() error {
	if g == nil || !g.persistent {
		return nil
	}
	return SaveGameData(g.currentSaveData())
}

// SavePersistentState is called on normal exit and can also be used by future
// pause/menu actions. It captures the latest desktop window state as well as
// settings and campaign progress.
func (g *Game) SavePersistentState() error {
	if g == nil || !g.persistent {
		return nil
	}
	if err := g.saveConfig(); err != nil {
		return err
	}
	return g.saveProgress()
}
