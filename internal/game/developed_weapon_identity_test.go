package game

import (
	"math"
	"testing"
)

type weaponFeel struct {
	n                          int
	cat                        DevelopedWeaponCategory
	proj                       DevelopedProjectileKind
	rof, mag, pellets, burst   int
	fp, recoil, weight, spread float64
	speed, fan                 float64
}

func weaponFeelOf(n int) weaponFeel {
	d, _ := WeaponByNumber(n)
	f := weaponFeel{n: n, cat: sourceWeaponCategory(n), rof: d.ROF, mag: d.Bullets, pellets: d.Shotgun, burst: 1,
		fp: d.Firepower, recoil: d.Recoil, weight: d.Weight, spread: 3, speed: 25}
	if spec, ok := developedWeaponSpec(n); ok {
		f.cat, f.proj, f.spread, f.speed, f.fan = spec.Category, spec.Projectile, spec.Spread, spec.ProjectileSpeed, spec.PelletFanStep
		f.burst = developedBurstRounds(spec)
	}
	return f
}

// No added weapon may play exactly like another weapon: same class and fire
// delay with push within 1, magazine within 2, recoil within 0.1 and the same
// weight, accuracy, velocity and pellet pattern.
func TestAddedWeaponsHaveNoNearIdenticalTwin(t *testing.T) {
	all := organizedGunLibraryNumbers(0)
	feels := make([]weaponFeel, 0, len(all))
	for _, n := range all {
		feels = append(feels, weaponFeelOf(n))
	}
	for i := range feels {
		for j := i + 1; j < len(feels); j++ {
			a, b := feels[i], feels[j]
			if a.n <= 86 && b.n <= 86 {
				continue // originals are never changed (reported only)
			}
			if a.cat != b.cat || a.proj != b.proj || a.pellets != b.pellets || a.burst != b.burst || a.rof != b.rof {
				continue
			}
			if math.Abs(a.fp-b.fp) > 1 || absInt(a.mag-b.mag) > 2 || math.Abs(a.recoil-b.recoil) > 0.1+1e-9 {
				continue
			}
			if a.weight != b.weight || math.Abs(a.spread-b.spread) >= 0.3 || math.Abs(a.speed-b.speed) >= 1 || a.fan != b.fan {
				continue
			}
			t.Errorf("near-identical: %d %s and %d %s", a.n, weaponDisplayName(a.n), b.n, weaponDisplayName(b.n))
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
