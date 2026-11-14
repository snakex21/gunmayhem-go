package game

import (
	"math"
	"math/rand"
)

type Crate struct {
	Serial  int
	X, Y    float64
	VY      float64
	Facing  int
	Falling bool
	Dead    bool
}

func spawnCrate(arena Map) Crate {
	width := int(math.Floor(arena.CrateMaxX - arena.CrateMinX))
	if width < 1 {
		width = 1
	}
	x := arena.CrateMinX + float64(rand.Intn(width))

	facing := 1
	if minX, _, maxX, _, ok := platformBounds(arena.Platforms); ok && x > (minX+maxX)/2 {
		// DefineSprite_668_crate flips _xscale when spawned on the right half.
		facing = -1
	}
	return Crate{X: x, Y: -300, VY: 10, Facing: facing, Falling: true}
}

// updateCrates follows DefineSprite_668_crate/frame_1/DoAction.as. The
// platform collision is point-based ground.platform.hitTest(x,y,true), then
// the crate backs out and searches the fall distance in five equal slices.
func updateCrates(arena Map, crates []Crate, players []*Player, unlocked []int, localHit Rect) {
	for i := range crates {
		c := &crates[i]
		if c.Dead {
			continue
		}

		if c.Falling {
			if c.VY < 18 {
				c.VY += 1
			}
			c.Y += c.VY
			if pointHitsPlatforms(arena.Platforms, c.X, c.Y) {
				c.Falling = false
				c.Y -= c.VY
				for step := 1; step <= 5; step++ {
					testY := c.Y + float64(step)*(c.VY/5)
					if pointHitsPlatforms(arena.Platforms, c.X, testY) {
						c.Y += c.VY / 5 * float64(step-1)
						break
					}
				}
			}
		}

		box := transformedLocalRect(localHit, c.X, c.Y, float64(c.Facing), 1)
		for _, p := range players {
			if !p.Active || !rectsOverlap(box, p.Hitbox()) {
				continue
			}

			// Source crate clip increments pgsdata[player-1][5] on pickup.
			p.CratesCollected++
			if len(unlocked) > 0 {
				randGun := pickCrateWeapon(unlocked)
				// Source perk 4: one in three crate pickups becomes the minigun.
				if p.PerkNumber == 4 && rand.Intn(3) == 0 {
					randGun = 55
				}
				// AI refuses gun 26 in the original crate code and substitutes 55.
				if randGun == 26 && p.AI {
					randGun = 55
				}
				p.EquipWeapon(randGun)
			}
			c.Dead = true
			break
		}
	}
}

// Crate draw weights: a source weapon (1-86) is twice as likely as any one
// added weapon (87+), which keeps the original arsenal at roughly half of all
// crate drops even though the added roster is larger.
const (
	crateWeightOriginal = 2
	crateWeightAdded    = 1
)

func crateWeaponWeight(number int) int {
	if number >= 1 && number <= 86 {
		return crateWeightOriginal
	}
	return crateWeightAdded
}

// pickCrateWeapon draws one weapon from pool with crateWeaponWeight. A pool
// of source weapons only keeps the exact source draw (random(n) over the
// pool), so GM1/GM2 crate behaviour is unchanged.
func pickCrateWeapon(pool []int) int {
	total := 0
	mixed := false
	for _, n := range pool {
		w := crateWeaponWeight(n)
		total += w
		if w != crateWeightOriginal {
			mixed = true
		}
	}
	if !mixed || total <= 0 {
		return pool[rand.Intn(len(pool))]
	}
	r := rand.Intn(total)
	for _, n := range pool {
		r -= crateWeaponWeight(n)
		if r < 0 {
			return n
		}
	}
	return pool[len(pool)-1]
}

func transformedLocalRect(r Rect, x, y, sx, sy float64) Rect {
	x1 := r.X * sx
	x2 := (r.X + r.W) * sx
	y1 := r.Y * sy
	y2 := (r.Y + r.H) * sy
	minX, maxX := math.Min(x1, x2), math.Max(x1, x2)
	minY, maxY := math.Min(y1, y2), math.Max(y1, y2)
	return Rect{X: x + minX, Y: y + minY, W: maxX - minX, H: maxY - minY}
}

func compactCrates(in []Crate) []Crate {
	out := in[:0]
	for _, c := range in {
		if !c.Dead {
			out = append(out, c)
		}
	}
	return out
}

func rectsOverlap(a, b Rect) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
}
