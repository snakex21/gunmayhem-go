package game

import "testing"

func TestHUDPowerupBadgesAndStarterBatAmmo(t *testing.T) {
	p := arsenal2Player(1, 300, 1)
	if len(hudPowerupBadges(p)) != 0 {
		t.Fatal("badge without a powerup")
	}
	applyPowerup(p, powerupAkimbo)
	applyPowerup(p, powerupBallisticShield)
	got := hudPowerupBadges(p)
	if len(got) != 2 || got[0] != powerupAkimbo || got[1] != powerupBallisticShield {
		t.Fatalf("badges=%v", got)
	}
	// A starter bat (the loadout weapon) has no ammo count; a crate bat does.
	bat := arsenal2Player(2, 300, weaponBaseballBat)
	bat.DefaultWeapon = weaponBaseballBat
	if hudShowsAmmo(bat) {
		t.Fatal("starter bat shows an ammo number")
	}
	bat.DefaultWeapon = 1
	if !hudShowsAmmo(bat) {
		t.Fatal("crate bat hides its swings left")
	}
	if !hudShowsAmmo(p) {
		t.Fatal("pistol hides its ammo")
	}
}

func TestZombieCarriesHarpoonRopeForClients(t *testing.T) {
	g := newZombieTestGame(t, 0, [4]int{1, 0, 0, 0})
	holdZombieWave(g)
	z := spawnTestZombie(g, zombieWalker, 500, hazardLandingY(g.arena, 500), 40)
	z.body.HarpoonTether, z.body.HarpoonTetherID = harpoonTetherFrames, 1
	z.syncFromBody()
	if z.HarpoonTether != harpoonTetherFrames || z.HarpoonTetherID != 1 {
		t.Fatalf("rope not mirrored: %d/%d", z.HarpoonTether, z.HarpoonTetherID)
	}
	s := netSnapshot{Zombies: []Zombie{*z}, ZombieHUD: g.zombie.hud}
	s.ZombieHUD.Active = true
	if err := validateZombieSnapshot(s); err != nil {
		t.Fatalf("valid rope rejected: %v", err)
	}
	s.Zombies[0].HarpoonTether = 999
	if validateZombieSnapshot(s) == nil {
		t.Fatal("bad zombie rope accepted")
	}
}
