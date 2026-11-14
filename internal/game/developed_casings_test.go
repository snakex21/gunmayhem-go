package game

import (
	"math"
	"testing"
)

func TestOriginalWeaponAnimationsUnchanged(t *testing.T) {
	for number := 1; number <= 86; number++ {
		want, ok := weaponAnimCatalog[number]
		if !ok {
			continue
		}
		if got := sourceWeaponAnim(number); got != want {
			t.Fatalf("source weapon %d animation changed: %+v want %+v", number, got, want)
		}
	}
}

func TestDevelopedWeaponsHaveShotKick(t *testing.T) {
	for number := 87; number <= 231; number++ {
		spec, ok := developedWeaponSpec(number)
		if !ok {
			continue
		}
		anim := sourceWeaponAnim(number)
		if anim.Blowback == 0 && anim.Pushback == 0 {
			t.Fatalf("developed weapon %d has no shot kick", number)
		}
		if anim.Blowback > 40 || anim.Pushback > 40 {
			t.Fatalf("developed weapon %d kick %+v exceeds every source gun", number, anim)
		}
		if spec.Projectile == DevelopedProjectileFlame && anim.Blowback != 0 {
			t.Fatalf("flamethrower %d would sit permanently tilted: %+v", number, anim)
		}
	}
	// Same class, same kick as the source medians.
	if got := sourceWeaponAnim(131); got.Blowback != 7 || got.Pushback != 10 { // AK-74 vs AK-47 (10)
		t.Fatalf("AK-74 kick %+v, source AK-47 is 7/10", got)
	}
	if got := sourceWeaponAnim(155); got.Blowback != 10 || got.Pushback != 30 { // M24 vs AWM (33)
		t.Fatalf("M24 kick %+v, source bolt snipers are 10/30", got)
	}
}

func fireDevelopedOnce(t *testing.T, number, facing int) (*Player, []Bullet, []Shell) {
	t.Helper()
	p := &Player{ID: 1, Team: 1, Active: true, X: 400, Y: 300, Facing: facing, MiniMulti: 1, PlayerScale: .8, Weapon: NewWeapon(number)}
	p.Weapon.WaitTime = p.Weapon.Def.ROF
	p.AI, p.AIShoot = true, true
	var bullets []Bullet
	var shells []Shell
	var flashes []MuzzleFlash
	updateWeapon(p, &bullets, &shells, &flashes)
	if len(bullets) == 0 {
		t.Fatalf("weapon %d did not fire", number)
	}
	return p, bullets, shells
}

func TestDevelopedWeaponsEjectCasingsPerClass(t *testing.T) {
	cases := map[int]int{
		100: 1, // Beretta 92FS
		114: 1, // Uzi
		131: 1, // AK-74
		134: 1, // Tavor (bullpup)
		143: 4, // M60 belt
		145: 1, // BAR (magazine)
		155: 2, // M24 SWS
		162: 5, // Benelli M4
		166: 6, // DP-12 bottom eject
		87:  0, // RPG-7
		173: 0, // M79
		177: 0, // crossbow
		178: 0, // bow
		179: 0, // harpoon
		187: 0, // flamethrower
		185: 0, // energy rifle
		189: 0, // portal gun
		111: 0, // revolver
		213: 0, // G11 caseless
	}
	for number, wantKind := range cases {
		for _, facing := range []int{1, -1} {
			p, _, shells := fireDevelopedOnce(t, number, facing)
			if wantKind == 0 {
				if len(shells) != 0 {
					t.Fatalf("weapon %d ejected %d casings, want none", number, len(shells))
				}
				continue
			}
			if len(shells) != 1 || shells[0].Kind != wantKind {
				t.Fatalf("weapon %d facing %d: casings %+v, want one kind %d", number, facing, shells, wantKind)
			}
			s := shells[0]
			portX, _ := developedCasingPort(number, developedWeaponSpecs[number], wantKind)
			if math.Abs(s.X-(p.X+portX*float64(facing))) > 1e-9 {
				t.Fatalf("weapon %d facing %d: casing at x=%.1f, port %.1f", number, facing, s.X, portX)
			}
			if wantKind != 6 && s.VX*float64(facing) >= 0 {
				t.Fatalf("weapon %d facing %d: casing flies forward (vx=%.2f)", number, facing, s.VX)
			}
			if s.ScaleX != float64(facing) {
				t.Fatalf("weapon %d: casing not mirrored with facing", number)
			}
		}
	}
	// The port sits between grip and muzzle (behind the grip on bullpups).
	pose := poseForWeapon(131)
	if x, _ := developedCasingPort(131, developedWeaponSpecs[131], 1); x <= pose.ShootX || x >= developedWeaponCatalog[131].FlashX {
		t.Fatalf("AK-74 ejection port %.1f outside grip %.1f..muzzle", x, pose.ShootX)
	}
	if x, _ := developedCasingPort(134, developedWeaponSpecs[134], 1); x >= poseForWeapon(134).ShootX {
		t.Fatalf("Tavor ejection port %.1f not behind the grip", x)
	}
}

func TestSourceWeaponFireDoesNotEjectExtraCasings(t *testing.T) {
	// Source guns eject from their timelines only; the shot itself adds none.
	_, _, shells := fireDevelopedOnce(t, 10, 1)
	if len(shells) != 0 {
		t.Fatalf("source AK fire spawned %d casings at the shot", len(shells))
	}
}

func TestFlamethrowerSpawnsAtNozzleWithSameReach(t *testing.T) {
	for _, number := range []int{187, 188} {
		spec := developedWeaponSpecs[number]
		def := developedWeaponCatalog[number]
		_, bullets, _ := fireDevelopedOnce(t, number, 1)
		b := bullets[0]
		if math.Abs(b.X-(400+def.FlashX)) > 1e-9 {
			t.Fatalf("flame %d spawns at x=%.1f, nozzle at %.1f", number, b.X, 400+def.FlashX)
		}
		oldReach := 23 + developedFlameTravel(spec.ProjectileSpeed, spec.Lifetime)
		newReach := def.FlashX + developedFlameTravel(spec.ProjectileSpeed, b.Lifetime)
		step := spec.ProjectileSpeed * math.Pow(0.988, float64(spec.Lifetime))
		if math.Abs(newReach-oldReach) > step/2+0.5 {
			t.Fatalf("flame %d reach %.1f px, was %.1f (lifetime %d -> %d)", number, newReach, oldReach, spec.Lifetime, b.Lifetime)
		}
		if b.Lifetime >= spec.Lifetime {
			t.Fatalf("flame %d lifetime not shortened: %d", number, b.Lifetime)
		}

		// Point-blank: an enemy between the old +23 origin and the nozzle is
		// still burned on the first step.
		enemy := &Player{ID: 2, Team: 2, Active: true, X: 400 + 45, Y: 300, PlayerScale: .8, DamageMulti: 1}
		arena := Map{Platforms: []Rect{{X: 0, Y: 300, W: 900, H: 40}}, LowestY: 600}
		shots := []Bullet{b}
		updateBullets(arena, shots, []*Player{enemy}, nil, nil, nil, nil)
		if enemy.BurnTime == 0 {
			t.Fatalf("flame %d misses a point-blank enemy behind the nozzle", number)
		}
	}
}
