package game

import "testing"

func TestAllowCheatsDefaultsOffAndPersistsInConfig(t *testing.T) {
	defaults := DefaultAppConfig()
	if defaults.AllowCheats {
		t.Fatal("ALLOW CHEATS must default to OFF")
	}

	cfg := defaults
	cfg.AllowCheats = true
	g := NewPersistent(cfg, DefaultSaveData())
	if !g.allowCheats {
		t.Fatal("NewPersistent did not apply ALLOW CHEATS")
	}
	// currentAppConfig only needs persistence enabled for window geometry.
	// Disable that branch in this headless unit test; the setting fields are
	// still serialized exactly the same way by saveConfig at runtime.
	g.persistent = false
	if got := g.currentAppConfig(); !got.AllowCheats {
		t.Fatal("currentAppConfig did not preserve ALLOW CHEATS")
	}
}

func TestResetOptionSettingsTurnsCheatsOff(t *testing.T) {
	g := New()
	g.allowCheats = true
	g.showCollisions = true
	g.menuDebug = true
	g.resetOptionSettings()
	if g.allowCheats {
		t.Fatal("RESET SETTINGS did not turn ALLOW CHEATS off")
	}
}
