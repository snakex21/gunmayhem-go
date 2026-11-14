package game

import (
	"math"
	"testing"
)

// Balance regression tests for the special added weapons (Energy Rifle,
// Javelin, Portal Gun, Flare Gun, Nail Gun, Compound Bow, Martini-Henry,
// HIMARS). The harness drives the real weapon, bullet, portal and player
// physics paths at the game's 30 TPS.

type sbWorld struct {
	m       Map
	players []*Player
	bullets []Bullet
	shells  []Shell
	flashes []MuzzleFlash
	trails  []InstagibTrailEffect
	expl    []Explosion
	blast   []BlastEffect
	portals []Portal
}

func sbPlayer(id int, x, y float64, facing int) *Player {
	p := NewPlayer(id, Map{})
	p.AI = true
	p.X, p.Y = x, y
	p.Facing = facing
	p.PerkNumber = 1
	p.Grounded = true
	p.JumpNum = 2
	p.CheapTimer = 120
	return p
}

func sbFlat() Map {
	return Map{Gravity: DefaultGravity, Platforms: []Rect{{X: -560, Y: 400, W: 2040, H: 30}}, LowestY: 900}
}

func (w *sbWorld) step(shoot bool) {
	s := w.players[0]
	s.AIShoot = shoot
	s.UpdateWithFire(w.m, nil, nil, func() { updateWeapon(s, &w.bullets, &w.shells, &w.flashes) })
	for _, p := range w.players[1:] {
		p.Update(w.m, nil, nil)
	}
	updateBullets(w.m, w.bullets, w.players, &w.trails, &w.expl, &w.blast, &w.portals)
	w.portals = prunePortals(w.portals, w.players)
	updatePortalTravel(w.portals, w.players, nil)
	w.bullets = compactBullets(w.bullets)
}

// sbRingOut: target idle in the middle of map 1's top platform, shooter held
// d px to its left. Returns the tick the target crosses the right edge, or -1.
func sbRingOut(num int, d float64) int {
	m := OriginalMap1()
	m.Platforms = append(append([]Rect{}, m.Platforms...), Rect{X: -140, Y: m.Platforms[0].Y, W: 130, H: 24})
	top := m.Platforms[0]
	tx := top.X + top.W/2
	sx := tx - d
	s := sbPlayer(1, sx, top.Y, 1)
	s.EquipWeapon(num)
	t := sbPlayer(2, tx, top.Y, -1)
	w := &sbWorld{m: m, players: []*Player{s, t}}
	for f := 0; f < 600; f++ {
		s.X, s.VX, s.Y, s.VY = sx, 0, top.Y, 0
		s.Grounded, s.JumpNum = true, 2
		w.step(s.Weapon.Def.Number == num)
		if t.X > top.X+top.W+2 || t.Y > 700 || !t.Active {
			return f
		}
	}
	return -1
}

// sbAverageRingOut averages a few runs (spread is random).
func sbAverageRingOut(t *testing.T, num int, d float64) float64 {
	t.Helper()
	sum := 0
	const runs = 6
	for i := 0; i < runs; i++ {
		f := sbRingOut(num, d)
		if f < 0 {
			t.Fatalf("weapon %d never rang out an idle target from %.0f px", num, d)
		}
		sum += f
	}
	return float64(sum) / runs
}

// sbPickupPush: shooter and target pinned on a flat floor 300 px apart, the
// target's velocity read and zeroed every tick. Returns the total impulse the
// whole pickup delivers (+ = away from the shooter).
func sbPickupPush(num int, d float64) float64 {
	sx, tx := 100.0, 100.0+d
	s := sbPlayer(1, sx, 400, 1)
	s.EquipWeapon(num)
	t := sbPlayer(2, tx, 400, -1)
	w := &sbWorld{m: sbFlat(), players: []*Player{s, t}}
	total := 0.0
	done := -1
	for f := 0; f < 1500; f++ {
		s.X, s.VX, s.Y, s.VY = sx, 0, 400, 0
		t.X, t.VX, t.Y, t.VY = tx, 0, 400, 0
		t.Grounded, t.JumpNum = true, 2
		w.step(s.Weapon.Def.Number == num)
		total += t.VX
		if s.Weapon.Def.Number != num && done < 0 {
			done = f
		}
		if done >= 0 && f > done+150 && len(w.bullets) == 0 && t.BurnTime == 0 {
			break
		}
	}
	return total
}

func TestSpecialWeaponStats(t *testing.T) {
	cases := []struct {
		num     int
		power   float64
		bullets int
	}{
		{185, 26, 24}, // Energy Rifle
		{180, 12, 40}, // Nail Gun
		{181, 35, 4},  // Flare Gun
		{178, 37, 8},  // Compound Bow
		{193, 68, 3},  // Martini-Henry
	}
	for _, c := range cases {
		def, ok := WeaponByNumber(c.num)
		if !ok {
			t.Fatalf("weapon %d missing", c.num)
		}
		if def.Firepower != c.power || def.Bullets != c.bullets {
			t.Errorf("%s: firepower %.0f bullets %d, want %.0f / %d", def.Name, def.Firepower, def.Bullets, c.power, c.bullets)
		}
	}
	if spec, _ := developedWeaponSpec(89); spec.ExplosionPower != 90 {
		t.Errorf("Javelin blast %.0f, want 90", spec.ExplosionPower)
	}
	if spec, _ := developedWeaponSpec(90); spec.ProjectileSpeed != himarsStrikeFallSpeed || himarsStrikeFallSpeed > 16 {
		t.Errorf("HIMARS fall speed %.0f, want slow fall %.0f", spec.ProjectileSpeed, himarsStrikeFallSpeed)
	}
}

// Energy Rifle ring-out sits with the AKM / BAR automatics instead of
// beating every automatic in the game.
func TestEnergyRifleRingOutNearAutomaticBand(t *testing.T) {
	for _, d := range []float64{150, 300, 500} {
		energy := sbAverageRingOut(t, 185, d)
		akm := sbAverageRingOut(t, 132, d)
		bar := sbAverageRingOut(t, 203, d)
		fastest := math.Min(akm, bar)
		t.Logf("%.0f px: energy %.2fs, AKM %.2fs, BAR %.2fs", d, energy/30, akm/30, bar/30)
		// Within 0.2 s of the band (it keeps its fastest-projectile identity).
		if energy < fastest-6 {
			t.Errorf("%.0f px: Energy Rifle rings out in %.2fs, far faster than AKM/BAR (%.2fs)", d, energy/30, fastest/30)
		}
		if energy > math.Max(akm, bar)+3 {
			t.Errorf("%.0f px: Energy Rifle rings out in %.2fs, slower than the AKM/BAR band", d, energy/30)
		}
	}
}

func TestFlareNailBowPickupPush(t *testing.T) {
	flare := sbPickupPush(181, 300)
	nail := sbPickupPush(180, 300)
	bow := sbPickupPush(178, 300)
	crossbow := sbPickupPush(177, 300)
	t.Logf("per pickup @300 px: flare %.0f, nail %.0f, bow %.0f, crossbow %.0f", flare, nail, bow, crossbow)
	// Burn panics the victim instead of pushing it, so a pinned target only
	// takes the four direct flare hits (4 x 35) plus the ignite nudge.
	if flare < 150 || flare > 185 {
		t.Errorf("Flare Gun pickup push %.0f, want about 167 (4 flares x 35)", flare)
	}
	// Pure push stays below the Crossbow; the flare's threat is the long burn.
	if flare >= crossbow {
		t.Errorf("Flare Gun pickup push %.0f matches the Crossbow (%.0f)", flare, crossbow)
	}
	if spec, _ := developedWeaponSpec(181); spec.IgniteFrames != 135 {
		t.Errorf("Flare Gun burn %d ticks, want 135 (4.5 s)", spec.IgniteFrames)
	}
	for _, d := range []float64{150, 300, 500} {
		t.Logf("ring-out %.0f px: flare %.2fs, crossbow %.2fs", d, sbAverageRingOut(t, 181, d)/30, sbAverageRingOut(t, 177, d)/30)
	}
	if nail < 440 || nail > 520 {
		t.Errorf("Nail Gun pickup push %.0f, want about 480", nail)
	}
	if bow < 270 || bow > 320 {
		t.Errorf("Compound Bow pickup push %.0f, want about 296", bow)
	}
	// The bow still trails the crossbow per hit and per ring-out.
	for _, d := range []float64{150, 300} {
		b, c := sbAverageRingOut(t, 178, d), sbAverageRingOut(t, 177, d)
		t.Logf("ring-out %.0f px: bow %.2fs, crossbow %.2fs", d, b/30, c/30)
		if b < c {
			t.Errorf("%.0f px: Compound Bow (%.2fs) rings out faster than the Crossbow (%.2fs)", d, b/30, c/30)
		}
	}
}

// Javelin: still hits standing and running targets, but a target that
// zigzags or jumps at the last moment can dodge it at medium/long range.
func TestJavelinTerminalPhaseDodgeable(t *testing.T) {
	near := func(tg *Player, w *sbWorld, r float64) bool {
		for _, b := range w.bullets {
			if b.DevelopedProjectile != 0 && math.Hypot(b.X-tg.X, b.Y-(tg.Y-28)) < r {
				return true
			}
		}
		return false
	}
	type dodge func(tg *Player, w *sbWorld, f int)
	hits := func(ctl dodge, d float64) int {
		n := 0
		for trial := 0; trial < 10; trial++ {
			s := sbPlayer(1, 100, 400, 1)
			s.EquipWeapon(89)
			tg := sbPlayer(2, 100+d+float64(trial*7), 400, -1)
			w := &sbWorld{m: sbFlat(), players: []*Player{s, tg}}
			best := 0.0
			for f := 0; f < 120; f++ {
				s.X, s.Y, s.VX = 100, 400, 0
				tg.AILeft, tg.AIRight, tg.AIUp = false, false, false
				ctl(tg, w, f)
				vx0 := tg.VX
				w.step(f < 3 && s.Weapon.Def.Number == 89)
				if dv := math.Abs(tg.VX - vx0); dv > best {
					best = dv
				}
			}
			if best > 40 {
				n++
			}
		}
		return n
	}
	standing := func(tg *Player, w *sbWorld, f int) {}
	running := func(tg *Player, w *sbWorld, f int) { tg.AIRight = true }
	zigzag := func(tg *Player, w *sbWorld, f int) {
		tg.AIRight = (f/8)%2 == 0
		tg.AILeft = !tg.AIRight
		tg.AIUp = f%15 == 0
	}
	jump := func(tg *Player, w *sbWorld, f int) { tg.AIUp = near(tg, w, 110) }
	for _, row := range []struct {
		name string
		ctl  dodge
	}{{"standing", standing}, {"running away", running}, {"zigzag+hop", zigzag}, {"jump at 110 px", jump}} {
		t.Logf("%-15s 150/300/500 px: %d / %d / %d of 10", row.name, hits(row.ctl, 150), hits(row.ctl, 300), hits(row.ctl, 500))
	}
	for _, d := range []float64{150, 300, 500} {
		if h := hits(standing, d); h < 9 {
			t.Errorf("standing target at %.0f px: Javelin hit %d/10, want >= 9", d, h)
		}
		if h := hits(running, d); h < 9 {
			t.Errorf("running target at %.0f px: Javelin hit %d/10, want >= 9", d, h)
		}
	}
	for _, d := range []float64{300, 500} {
		if h := hits(zigzag, d); h > 2 {
			t.Errorf("zigzag target at %.0f px: Javelin hit %d/10, want <= 2", d, h)
		}
	}
	if h := hits(jump, 500); h > 2 {
		t.Errorf("target jumping at 110 px from 500 px: Javelin hit %d/10, want <= 2", h)
	}
}

// Portals open only on platform tops, so a missed shot can no longer leave
// an edge-facing portal that rings out the next victim.
func TestPortalGunEdgeExploitGone(t *testing.T) {
	m := OriginalMap1()
	top := m.Platforms[0]
	for _, d := range []float64{150, 300} {
		s := sbPlayer(1, top.X+top.W/2-d, top.Y, 1)
		s.EquipWeapon(189)
		tg := sbPlayer(2, top.X+top.W/2, top.Y, -1)
		w := &sbWorld{m: m, players: []*Player{s, tg}}
		sx := s.X
		// Prime: fire off the left edge so the shots leave the screen.
		s.Facing = -1
		for f := 0; f < 100; f++ {
			s.X, s.VX, s.Y, s.VY = sx, 0, top.Y, 0
			w.step(f < 3)
			tg.X, tg.VX = top.X+top.W/2, 0
		}
		for _, p := range w.portals {
			if p.NormalX != 0 || p.NormalY != -1 {
				t.Fatalf("portal opened on a side/underside: %+v", p)
			}
		}
		s.Weapon.WaitTime = s.Weapon.Def.ROF
		for f := 0; f < 300; f++ {
			s.X, s.VX, s.Y, s.VY = sx, 0, top.Y, 0
			s.Facing = 1
			w.step(true)
			if tg.X > top.X+top.W+2 || tg.X < top.X-2 || tg.Y > 700 || tg.Deaths > 0 {
				t.Fatalf("%.0f px: Portal Gun rang the target out at tick %d (x=%.0f)", d, f, tg.X)
			}
		}
	}
}

func TestPortalPlacementTopsOnly(t *testing.T) {
	arena := Map{Platforms: []Rect{{X: 100, Y: 300, W: 400, H: 30}}}
	cases := []struct{ x0, y0, x1, y1 float64 }{
		{60, 315, 120, 315},  // into the left side
		{540, 315, 480, 315}, // into the right side
		{300, 360, 300, 320}, // into the underside
		{300, 280, 300, 310}, // onto the top
	}
	for _, c := range cases {
		var portals []Portal
		b := &Bullet{OwnerID: 1, PortalColor: PortalBlue}
		placePortalFromSegment(&portals, b, c.x0, c.y0, c.x1, c.y1, arena.Platforms[0])
		if len(portals) != 1 || portals[0].NormalY != -1 || portals[0].NormalX != 0 || portals[0].Y > 300 {
			t.Fatalf("segment %+v placed %+v, want a top portal", c, portals)
		}
		if portals[0].X < 120 || portals[0].X > 480 {
			t.Fatalf("segment %+v: portal at x=%.0f, want 20 px inside the platform ends", c, portals[0].X)
		}
	}
	for _, pos := range [][2]float64{{-200, 315}, {900, 315}, {300, 700}} {
		var portals []Portal
		b := &Bullet{OwnerID: 1, PortalColor: PortalOrange, X: pos[0], Y: pos[1]}
		if !placePortalOnNearestSurface(&portals, b, arena) || portals[0].NormalY != -1 {
			t.Fatalf("off-screen shot at %v placed %+v, want a top portal", pos, portals)
		}
	}
}

func sbPairedPortals(ownerID int) []Portal {
	return []Portal{
		{X: 200, Y: 397, NormalX: 0, NormalY: -1, OwnerID: ownerID, Color: PortalBlue, Active: true},
		{X: 900, Y: 197, NormalX: 0, NormalY: -1, OwnerID: ownerID, Color: PortalOrange, Active: true},
	}
}

func TestPortalTeleportBlockedByShield(t *testing.T) {
	// Unshielded: teleported to the exit.
	{
		portals := sbPairedPortals(1)
		target := sbPlayer(2, 500, 400, -1)
		b := &Bullet{OwnerID: 1, PortalColor: PortalBlue, Facing: 1}
		portalCombatHit(b, target, &portals)
		if math.Abs(target.X-900) > 1 {
			t.Fatalf("unshielded target not teleported: x=%.0f", target.X)
		}
	}
	// Shield powerup: no teleport, a fifth of the pulse.
	{
		portals := sbPairedPortals(1)
		target := sbPlayer(2, 500, 400, -1)
		target.ShieldTime = 100
		b := &Bullet{OwnerID: 1, PortalColor: PortalBlue, Facing: 1}
		portalCombatHit(b, target, &portals)
		if target.X != 500 || target.Y != 400 {
			t.Fatalf("shielded target teleported to %.0f,%.0f", target.X, target.Y)
		}
		if math.Abs(target.VX-portalUnpairedPush*0.2) > 1e-9 {
			t.Fatalf("shielded target VX %.2f, want %.2f", target.VX, portalUnpairedPush*0.2)
		}
	}
	// SHIELD ability: no teleport, and the knockback window blocks the pulse.
	{
		portals := sbPairedPortals(1)
		target := sbPlayer(2, 500, 400, -1)
		target.AbilityNumber = abilityShield
		players := []*Player{target}
		window := beginAbilityKnockbackWindow(players)
		b := &Bullet{OwnerID: 1, PortalColor: PortalBlue, Facing: 1}
		portalCombatHit(b, target, &portals)
		endAbilityKnockbackWindow(players, window, true)
		if target.X != 500 || target.VX != 0 {
			t.Fatalf("SHIELD ability target moved: x=%.0f vx=%.2f", target.X, target.VX)
		}
		if !target.AbilityShieldUsed {
			t.Fatalf("SHIELD ability should have spent its block")
		}
	}
	// Unpaired pulse is 20.
	{
		target := sbPlayer(2, 500, 400, -1)
		b := &Bullet{OwnerID: 1, PortalColor: PortalBlue, Facing: 1}
		portalCombatHit(b, target, nil)
		if target.VX != 20 {
			t.Fatalf("unpaired pulse %.1f, want 20", target.VX)
		}
	}
}

func TestPortalsExpireWhenGunLeavesHands(t *testing.T) {
	owner := sbPlayer(1, 100, 400, 1)
	owner.EquipWeapon(189)
	other := sbPlayer(2, 300, 400, -1)
	other.EquipWeapon(189)
	players := []*Player{owner, other}
	all := append(sbPairedPortals(1), sbPairedPortals(2)...)

	if got := prunePortals(append([]Portal(nil), all...), players); len(got) != 4 {
		t.Fatalf("portals pruned while both owners hold the gun: %d left", len(got))
	}
	owner.EquipWeapon(1) // swapped / ran empty
	got := prunePortals(append([]Portal(nil), all...), players)
	if len(got) != 2 || got[0].OwnerID != 2 || got[1].OwnerID != 2 {
		t.Fatalf("after the owner dropped the gun: %+v", got)
	}
	owner.EquipWeapon(189)
	owner.Active = false // died
	if got := prunePortals(append([]Portal(nil), all...), players); len(got) != 2 {
		t.Fatalf("after the owner died: %d portals left, want 2", len(got))
	}

	// Emptying the magazine through the real fire path removes the portals.
	s := sbPlayer(1, 100, 400, 1)
	s.EquipWeapon(189)
	w := &sbWorld{m: sbFlat(), players: []*Player{s}}
	sawPortals := false
	for f := 0; f < 600 && s.Weapon.Def.Number == 189; f++ {
		s.X, s.VX, s.Y, s.VY = 100, 0, 400, 0
		s.Facing = 1
		if f%40 == 20 {
			s.Facing = -1
		}
		w.step(true)
		if len(w.portals) > 0 {
			sawPortals = true
		}
	}
	if s.Weapon.Def.Number == 189 {
		t.Fatalf("Portal Gun never ran empty")
	}
	for f := 0; f < 60; f++ {
		w.step(false)
	}
	if !sawPortals || len(w.portals) != 0 {
		t.Fatalf("portals after the gun ran empty: saw=%v left=%d", sawPortals, len(w.portals))
	}
}

// HIMARS lands on the nearest enemy ahead at its fire-time position and
// always pushes forward.
func TestHimarsStrikeTargetsEnemyAheadAndPushesForward(t *testing.T) {
	for _, walk := range []int{0, 1, -1} {
		for _, d := range []float64{60, 120, 200, 300, 450, 600} {
			g := &Game{arena: sbFlat()}
			s := sbPlayer(1, 100, 400, 1)

			s.EquipWeapon(90)
			tg := sbPlayer(2, 100+d, 400, -1)
			g.players = []*Player{s, tg}
			total := 0.0
			fired := false
			for f := 0; f < 60 && tg.Deaths == 0; f++ {
				s.X, s.Y, s.VX = 100, 400, 0
				tg.AIRight, tg.AILeft = walk == 1, walk == -1
				if walk == -1 && tg.X < 170 {
					tg.AILeft = false
				}
				vx0 := tg.VX
				before := len(g.bullets)
				s.AIShoot = s.Weapon.Def.Number == 90
				s.UpdateWithFire(g.arena, nil, nil, func() { g.updatePlayerWeapon(s) })
				if !fired && len(g.bullets) > before {
					fired = true
					b := g.bullets[before]
					if walk == 0 && math.Abs(b.X-tg.X) > 1e-6 {
						t.Fatalf("d=%.0f: strike at x=%.0f, standing target at %.0f", d, b.X, tg.X)
					}
					if len(g.extraHazardMarkers) == 0 || g.extraHazardMarkers[0].Kind != hazardMarkerStrike ||
						g.extraHazardMarkers[0].X != b.X || g.extraHazardMarkers[0].Y != 400 {
						t.Fatalf("d=%.0f: no landing marker under the strike: %+v", d, g.extraHazardMarkers)
					}
				}
				tg.Update(g.arena, nil, nil)
				updateBullets(g.arena, g.bullets, g.players, &g.instagibTrails, &g.explosions, &g.blastFX, &g.portals)
				g.bullets = compactBullets(g.bullets)
				g.updateHazardMarkers()
				// Strongest single impulse (later ticks include the fall/death).
				if dv := tg.VX - vx0*Friction; math.Abs(dv) > math.Abs(total) {
					total = dv
				}
			}
			if !fired {
				t.Fatalf("HIMARS did not fire")
			}
			if walk == 0 && total < 109 {
				t.Errorf("standing target at %.0f px: push %.0f, want a direct hit (~110)", d, total)
			}
			if total < 0 {
				t.Errorf("walk=%d d=%.0f: strike pulled the target toward the shooter (%.0f)", walk, d, total)
			}
			if walk != 0 {
				t.Logf("walk=%d d=%.0f: push %.0f", walk, d, total)
			}
		}
	}
}

func TestHimarsStrikeFallbackWithoutEnemyAhead(t *testing.T) {
	g := &Game{arena: sbFlat()}
	s := sbPlayer(1, 100, 400, 1)

	s.EquipWeapon(90)
	behind := sbPlayer(2, 20, 400, 1) // behind the shooter
	far := sbPlayer(3, 900, 400, -1)  // beyond 700 px
	high := sbPlayer(4, 300, 100, -1) // more than 260 px above
	g.players = []*Player{s, behind, far, high}
	for f := 0; f < 10 && len(g.bullets) == 0; f++ {
		s.AIShoot = true
		s.UpdateWithFire(g.arena, nil, nil, func() { g.updatePlayerWeapon(s) })
	}
	if len(g.bullets) != 1 || g.bullets[0].X != 360 {
		t.Fatalf("strike without a target ahead: %+v, want x=360", g.bullets)
	}
}
