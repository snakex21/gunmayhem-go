package game

const (
	SourceGameModeNormal   = 1
	SourceGameModeInstagib = 2
	SourceGameModeTeams    = 3
	SourceGameModeGunGame  = 4
	SourceGameModeSurvival = 5
)

var sourceGunGameGuns = [...]int{
	0,
	2,  // level 1
	29, // 2
	19, // 3
	46, // 4
	13, // 5
	51, // 6
	50, // 7
	11, // 8
	38, // 9
	33, // 10
	58, // 11
	62, // 12
	66, // 13
	65, // 14
	44, // 15
}

// configureSourceGameMode reproduces the player-constructor state that depends
// on root.gamemode/root.totallives. It is used when frame10 creates a fresh
// player movie clip (and therefore also when our R reset recreates the round).
func (p *Player) configureSourceGameMode(mode, totalLives int, arena Map) {
	p.GameMode = mode
	p.TotalLives = totalLives
	p.CurrentLevel = 0
	p.CurrentGun = 2
	p.KillSelf = false

	p.Lives = totalLives
	if mode == SourceGameModeGunGame {
		p.Lives = 99999
	}

	// Source constructor executes currentlevel=0; UPGRADE(); currentgun=2;
	// respawn(). UPGRADE level1 sets/equips gun2.
	p.sourceUpgrade(nil)
	p.CurrentGun = 2
	p.Active = true
	p.Kills = 0
	p.Deaths = 0
	p.ShotsFired = 0
	p.HitsLanded = 0
	p.CratesCollected = 0
	p.PowerupsCollected = 0
	p.Score = 0
	p.DeathSerial = 0
	p.LastDeathBy = 0
	p.Respawn(arena)
}

// sourceUpgrade is DefineSprite_697_player.UPGRADE(). It intentionally calls
// getgun(currentgun) immediately for levels1..15. At level16 it does not equip
// another gun: all other active players receive lives=0 and killself=true.
func (p *Player) sourceUpgrade(activePlayers []*Player) {
	p.CurrentLevel++
	if p.CurrentLevel == 16 {
		for _, other := range activePlayers {
			if other == nil || other == p || !other.Active {
				continue
			}
			other.Lives = 0
			other.KillSelf = true
		}
		return
	}
	if p.CurrentLevel > 0 && p.CurrentLevel < len(sourceGunGameGuns) {
		p.CurrentGun = sourceGunGameGuns[p.CurrentLevel]
		p.EquipWeapon(p.CurrentGun)
	}
}
