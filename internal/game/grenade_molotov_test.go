package game

import (
	"math"
	"testing"
)

func molotovTestArena() Map {
	return Map{Platforms: []Rect{{X: 0, Y: 300, W: 800, H: 24}}, LowestY: 600}
}

func TestMolotovDoesNotIgniteItsOwner(t *testing.T) {
	owner := &Player{ID: 1, Active: true, X: 110, Y: 300, Facing: 1, DamageMulti: 1}
	clone := &Player{ID: 5, OwnerPlayerID: 1, Active: true, X: 95, Y: 300, DamageMulti: 1}
	enemy := &Player{ID: 2, Active: true, X: 140, Y: 300, DamageMulti: 1}
	players := []*Player{owner, clone, enemy}
	var fire []FirePatch
	g := Grenade{X: 120, Y: 300, Type: ThrowableMolotov, OwnerID: 1}
	if !spawnMolotovFire(molotovTestArena(), &g, players, &fire) {
		t.Fatal("molotov on a platform did not spawn fire")
	}
	if owner.BurnTime != 0 || owner.VX != 0 || owner.VY != 0 || clone.BurnTime != 0 {
		t.Fatalf("impact affected its thrower: owner=%+v clone burn=%d", owner, clone.BurnTime)
	}
	if enemy.BurnTime == 0 || enemy.VX <= 0 {
		t.Fatalf("impact no longer ignites enemies: %+v", enemy)
	}

	// Standing in their own fire for its whole lifetime never burns the owner.
	for frame := 0; frame < 320; frame++ {
		updateFirePatches(fire, []*Player{owner, clone})
	}
	if owner.BurnTime != 0 || owner.VX != 0 || owner.FireContactCooldown != 0 || clone.BurnTime != 0 {
		t.Fatalf("owner burned in own fire: burn=%d vx=%.2f", owner.BurnTime, owner.VX)
	}
}

func TestEnvironmentalFireStillBurnsEveryone(t *testing.T) {
	p := &Player{ID: 1, Active: true, X: 12, Y: 100, DamageMulti: 1}
	patches := []FirePatch{{X: 0, Y: 100, OwnerID: 0, Life: 200, Frame: 5}}
	updateFirePatches(patches, []*Player{p})
	if p.BurnTime == 0 {
		t.Fatal("chaos fire (owner 0) must burn every player")
	}
}

func TestMolotovTimingOutMidAirSnapsToPlatformBelow(t *testing.T) {
	arena := molotovTestArena()
	grenades := []Grenade{{X: 200, Y: 200, Type: ThrowableMolotov, OwnerID: 1, Fuse: throwableProfiles[ThrowableMolotov].Fuse}}
	var fire []FirePatch
	var sparks []DynamiteEffect
	updateGrenades(arena, grenades, nil, nil, &sparks, nil, &fire)
	if !grenades[0].Dead || len(fire) == 0 {
		t.Fatalf("timed-out molotov: dead=%v fire=%d", grenades[0].Dead, len(fire))
	}
	for _, patch := range fire {
		if patch.Y != 300 {
			t.Fatalf("fire floating at y=%.1f, want platform top 300", patch.Y)
		}
	}
	if len(sparks) == 0 {
		t.Fatal("molotov burst has no impact effect")
	}
}

func TestMolotovTimingOutHighAboveVoidFizzles(t *testing.T) {
	arena := molotovTestArena()
	for _, start := range []Grenade{
		{X: 900, Y: 200}, // past the platform edge
		{X: 200, Y: -50}, // far above the platform (350 px)
	} {
		start.Type, start.OwnerID, start.Fuse = ThrowableMolotov, 1, throwableProfiles[ThrowableMolotov].Fuse
		grenades := []Grenade{start}
		var fire []FirePatch
		var sparks []DynamiteEffect
		victim := &Player{ID: 2, Active: true, X: start.X, Y: start.Y + 30, DamageMulti: 1}
		updateGrenades(arena, grenades, []*Player{victim}, nil, &sparks, nil, &fire)
		if !grenades[0].Dead || len(fire) != 0 || victim.BurnTime != 0 {
			t.Fatalf("molotov at %+v should fizzle: dead=%v fire=%+v burn=%d", start, grenades[0].Dead, fire, victim.BurnTime)
		}
		if len(sparks) == 0 {
			t.Fatal("fizzle should leave a small spark effect")
		}
	}
}

func TestMolotovFireSpawnsOnStruckPlatformNotTheOneAbove(t *testing.T) {
	arena := Map{Platforms: []Rect{
		{X: 0, Y: 250, W: 400, H: 24}, // upper
		{X: 0, Y: 340, W: 400, H: 24}, // lower, the one actually reached
	}}
	// Old absolute-distance search picked the upper floor (40 px) over the
	// lower one (50 px) for a bottle breaking at y=290.
	if y, ok := fireSurfaceY(arena, 100, 290); !ok || y != 340 {
		t.Fatalf("fire surface=%.1f ok=%v, want lower platform 340", y, ok)
	}

	// Falling bottle crosses the lower platform top between frames.
	grenades := []Grenade{{X: 100, Y: 330, VY: 15, Type: ThrowableMolotov, OwnerID: 1, FreePass: true}}
	var fire []FirePatch
	updateGrenades(arena, grenades, nil, nil, nil, nil, &fire)
	if len(fire) == 0 {
		t.Fatal("molotov impact on lower platform spawned no fire")
	}
	for _, patch := range fire {
		if patch.Y != 340 {
			t.Fatalf("fire patch on y=%.1f, want struck platform 340", patch.Y)
		}
	}
}

func TestMolotovFireDoesNotDripOntoLowerFloorPastEdge(t *testing.T) {
	arena := Map{Platforms: []Rect{
		{X: 0, Y: 300, W: 100, H: 24},
		{X: 0, Y: 340, W: 400, H: 24},
	}}
	g := Grenade{X: 90, Y: 300, Type: ThrowableMolotov, OwnerID: 1}
	var fire []FirePatch
	spawnMolotovFire(arena, &g, nil, &fire)
	if len(fire) == 0 {
		t.Fatal("no fire spawned")
	}
	for _, patch := range fire {
		if patch.Y != 300 || patch.X > 100 {
			t.Fatalf("fire spread off the struck platform: %+v", patch)
		}
	}
}

func TestFirePushesConsistentlyOutOfTheStrip(t *testing.T) {
	arena := molotovTestArena()
	for _, startX := range []float64{400, 395, 430, 340} {
		g := Grenade{X: 400, Y: 300, Type: ThrowableMolotov, OwnerID: 1}
		var fire []FirePatch
		spawnMolotovFire(arena, &g, nil, &fire)
		p := &Player{ID: 2, Active: true, X: startX, Y: 300, Facing: 1, DamageMulti: 1}
		sign := 0.0
		for frame := 0; frame < 120; frame++ {
			before := p.VX
			p.X = startX // keep them standing inside the fire
			updateFirePatches(fire, []*Player{p})
			if p.FireContactCooldown > 0 {
				p.FireContactCooldown--
			}
			delta := p.VX - before
			if delta == 0 {
				continue
			}
			if sign == 0 {
				sign = math.Copysign(1, delta)
			} else if math.Copysign(1, delta) != sign {
				t.Fatalf("x=%.0f: fire push flipped direction on frame %d", startX, frame)
			}
		}
		if sign == 0 {
			t.Fatalf("x=%.0f: standing in fire produced no push", startX)
		}
		if startX > 400 && sign < 0 || startX < 395 && sign > 0 {
			t.Fatalf("x=%.0f: pushed toward the fire centre (sign %.0f)", startX, sign)
		}
	}
}

func TestBlastShieldHalvesMolotovFire(t *testing.T) {
	plain := &Player{ID: 2, Active: true, X: 12, Y: 100, DamageMulti: 1}
	shield := &Player{ID: 3, Active: true, X: 12, Y: 100, DamageMulti: 1, PerkNumber: 6}
	patches := []FirePatch{{X: 0, Y: 100, OwnerID: 1, Life: 200, Frame: 5}}
	updateFirePatches(patches, []*Player{plain, shield})
	if shield.BurnTime == 0 || shield.BurnTime >= plain.BurnTime || shield.VX >= plain.VX {
		t.Fatalf("blast shield fire: plain burn=%d vx=%.2f, shield burn=%d vx=%.2f",
			plain.BurnTime, plain.VX, shield.BurnTime, shield.VX)
	}
}

func TestAIFleesLiveGrenadeBeforeItExplodes(t *testing.T) {
	arena := molotovTestArena()
	right := &Player{ID: 2, Active: true, AI: true, X: 180, Y: 300, JumpNum: 2}
	left := &Player{ID: 3, Active: true, AI: true, X: 70, Y: 300, JumpNum: 2}
	human := &Player{ID: 1, Active: true, X: 120, Y: 300, JumpNum: 2}
	shielded := &Player{ID: 4, Active: true, AI: true, X: 130, Y: 300, JumpNum: 2, PerkNumber: 6}
	far := &Player{ID: 5, Active: true, AI: true, X: 600, Y: 300, JumpNum: 2}
	players := []*Player{right, left, human, shielded, far}
	fuse := throwableProfiles[ThrowableDynamite].Fuse - grenadeAIWarningFrames - 1
	grenades := []Grenade{{X: 100, Y: 292, Type: ThrowableDynamite, OwnerID: 1, HitGround: true, Fuse: fuse}}
	var dyn []DynamiteEffect
	var blast []BlastEffect
	var explosions []Explosion
	updateGrenades(arena, grenades, players, &explosions, &dyn, &blast, nil)
	if grenades[0].Fuse != 40 {
		t.Fatalf("dynamite warning tick=%d, want source fusetime 40", grenades[0].Fuse)
	}
	if right.AILockRight != 10 || right.AILockUp < 1 {
		t.Fatalf("bot right of grenade did not flee right: %+v", right)
	}
	if left.AILockLeft != 10 || left.AILockUp < 1 {
		t.Fatalf("bot left of grenade did not flee left: lockL=%d up=%d", left.AILockLeft, left.AILockUp)
	}
	if human.AILockUp != 0 || shielded.AILockUp != 0 || far.AILockUp != 0 {
		t.Fatal("dodge must only affect nearby AI without BLAST SHIELD")
	}

	// The lock really drives the bot away through the normal controller.
	g := &Game{arena: arena, players: []*Player{right}, grenades: grenades}
	right.Weapon = NewWeapon(1)
	decideAIController(right, g)
	if !right.AIRight || right.AILeft || !right.AIUp {
		t.Fatalf("controller did not run/jump away: L=%v R=%v U=%v", right.AILeft, right.AIRight, right.AIUp)
	}
}

func TestAIWalksOutOfFire(t *testing.T) {
	arena := molotovTestArena()
	ai := &Player{ID: 2, Active: true, AI: true, X: 330, Y: 300, JumpNum: 2, Weapon: NewWeapon(1)}
	g := &Game{arena: arena, players: []*Player{ai}}
	grenade := Grenade{X: 400, Y: 300, Type: ThrowableMolotov, OwnerID: 1}
	spawnMolotovFire(arena, &grenade, nil, &g.firePatches)
	decideAIController(ai, g)
	if !ai.AILeft || ai.AIRight || !ai.AIUp {
		t.Fatalf("bot inside fire left side should escape left: L=%v R=%v U=%v", ai.AILeft, ai.AIRight, ai.AIUp)
	}

	// A bot standing next to the strip does not walk into it.
	ai2 := &Player{ID: 3, Active: true, AI: true, X: 250, Y: 300, JumpNum: 2, AIRight: true}
	aiAvoidFire(ai2, arena.Platforms, g.firePatches)
	if ai2.AIRight {
		t.Fatal("bot walked into fire")
	}

	// The thrower's own fire is harmless to them, so it is not avoided.
	owner := &Player{ID: 1, Active: true, AI: true, X: 400, Y: 300, JumpNum: 2}
	aiAvoidFire(owner, arena.Platforms, g.firePatches)
	if owner.AIUp || owner.AILeft || owner.AIRight {
		t.Fatal("owner avoided their own harmless fire")
	}
}

func TestGrenadeThrowDecrementsCountAndAnimatesHand(t *testing.T) {
	p := &Player{ID: 1, Active: true, AI: true, X: 100, Y: 300, Facing: 1, Grenades: 3}
	var grenades []Grenade
	p.AIGrenade = true
	for i := 0; i < 6; i++ {
		updateGrenadeInput(p, &grenades)
	}
	if !p.GrenadeHeld || len(grenades) != 0 || p.VisualHand2ChildX != grenadeWindupHandX {
		t.Fatalf("charging: held=%v thrown=%d hand=%.1f", p.GrenadeHeld, len(grenades), p.VisualHand2ChildX)
	}
	if p.GrenadePower != 4 {
		t.Fatalf("charge power=%.1f want 4 after 6 frames", p.GrenadePower)
	}
	p.AIGrenade = false
	updateGrenadeInput(p, &grenades)
	if len(grenades) != 1 || p.Grenades != 2 || p.VisualHand2ChildX != grenadeThrowHandX {
		t.Fatalf("throw: grenades=%d left=%d hand=%.1f", len(grenades), p.Grenades, p.VisualHand2ChildX)
	}
	if grenades[0].VX != 4 || grenades[0].VY != -4 {
		t.Fatalf("throw velocity changed from source: %+v", grenades[0])
	}
	p.Grenades = 0
	p.AIGrenade = true
	updateGrenadeInput(p, &grenades)
	p.AIGrenade = false
	updateGrenadeInput(p, &grenades)
	if len(grenades) != 1 || p.Grenades != 0 {
		t.Fatal("threw without any grenades left")
	}
}

func TestGrenadeThrowPreviewLandsWhereGrenadeLands(t *testing.T) {
	arena := molotovTestArena()
	arena.Gravity = 0.6
	p := &Player{ID: 1, Active: true, X: 300, Y: 300, Facing: 1, GrenadePower: 8}
	points, landed := grenadeThrowPreview(arena, p, 200)
	if !landed || len(points) == 0 {
		t.Fatal("preview did not land on the floor")
	}
	gr := []Grenade{{X: p.X + 30, Y: p.Y - 35, VX: 8, VY: -8, Type: ThrowableMk2, Fuse: -1000}}
	for i := 0; i < 200 && !gr[0].HitGround; i++ {
		updateGrenades(arena, gr, nil, nil, nil, nil, nil)
	}
	end := points[len(points)-1]
	if math.Abs(end.X-gr[0].X) > 1 {
		t.Fatalf("preview lands at x=%.1f, grenade landed at x=%.1f", end.X, gr[0].X)
	}
}
