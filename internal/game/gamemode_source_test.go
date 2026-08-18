package game

import "testing"

func TestSourceModeDefaults(t *testing.T) {
	arena := OriginalMap1()
	p := NewPlayer(1, arena)
	p.configureSourceGameMode(SourceGameModeNormal, 10, arena)
	if p.Lives != 10 || p.TotalLives != 10 || p.GameMode != 1 {
		t.Fatalf("normal source mode state: mode=%d lives=%d total=%d", p.GameMode, p.Lives, p.TotalLives)
	}
	if p.Weapon.Def.Number != p.DefaultWeapon {
		t.Fatalf("normal respawn gun=%d want default %d", p.Weapon.Def.Number, p.DefaultWeapon)
	}
}

func TestSourceInstagibModeLoadout(t *testing.T) {
	arena := OriginalMap1()
	p := NewPlayer(1, arena)
	p.configureSourceGameMode(SourceGameModeInstagib, 10, arena)
	if p.Weapon.Def.Number != 9 {
		t.Fatalf("Instagib respawn gun=%d want9", p.Weapon.Def.Number)
	}
	if p.GameMode != 2 {
		t.Fatalf("mode=%d want2", p.GameMode)
	}
}

func TestSourceGunGameProgression(t *testing.T) {
	arena := OriginalMap1()
	p := NewPlayer(1, arena)
	p.configureSourceGameMode(SourceGameModeGunGame, 10, arena)
	if p.Lives != 99999 || p.CurrentLevel != 1 || p.CurrentGun != 2 || p.Weapon.Def.Number != 2 {
		t.Fatalf("Gun Game init: lives=%d level=%d currentgun=%d weapon=%d", p.Lives, p.CurrentLevel, p.CurrentGun, p.Weapon.Def.Number)
	}
	p.sourceUpgrade(nil)
	if p.CurrentLevel != 2 || p.CurrentGun != 29 || p.Weapon.Def.Number != 29 {
		t.Fatalf("Gun Game level2: level=%d currentgun=%d weapon=%d", p.CurrentLevel, p.CurrentGun, p.Weapon.Def.Number)
	}
	p.CurrentLevel -= 2
	p.sourceUpgrade(nil)
	p.EquipWeapon(p.CurrentGun)
	if p.CurrentLevel != 1 || p.CurrentGun != 2 || p.Weapon.Def.Number != 2 {
		t.Fatalf("Gun Game empty-mag downgrade: level=%d currentgun=%d weapon=%d", p.CurrentLevel, p.CurrentGun, p.Weapon.Def.Number)
	}
}

func TestSourceGunGameLevel16KillsOthers(t *testing.T) {
	arena := OriginalMap1()
	winner := NewPlayer(1, arena)
	other := NewPlayer(2, arena)
	winner.GameMode = SourceGameModeGunGame
	winner.CurrentLevel = 15
	other.Lives = 99999
	winner.sourceUpgrade([]*Player{winner, other})
	if winner.CurrentLevel != 16 {
		t.Fatalf("winner level=%d want16", winner.CurrentLevel)
	}
	if other.Lives != 0 || !other.KillSelf {
		t.Fatalf("source level16 other state: lives=%d killself=%v", other.Lives, other.KillSelf)
	}
}

func TestSourceSurvivalRespawnShield(t *testing.T) {
	arena := OriginalMap1()
	p := NewPlayer(1, arena)
	p.configureSourceGameMode(SourceGameModeSurvival, 10, arena)
	if p.ShieldTime != 140 {
		t.Fatalf("survival respawn shieldtime=%d want140", p.ShieldTime)
	}
}
