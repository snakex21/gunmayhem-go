package game

import "testing"

func TestDeagleReloadEmitsSourceProjectile(t *testing.T) {
	m := OriginalMap1()
	p := NewPlayer(1, m)
	p.EquipWeapon(2)
	p.VisualHand1ChildX = 25
	p.Facing = 1
	var shells []Shell

	// DefineSprite_422_gun_deagle/frame_28 is zero-based timeline frame 27.
	enterWeaponTimelineFrame(p, &p.Weapon, 27, &shells, 0)
	if len(shells) != 1 {
		t.Fatalf("deagle frame28 emitted %d particles, want1", len(shells))
	}
	s := shells[0]
	if s.Kind != 12 || s.VX != 20 || s.VY != -6 || s.Facing != 1 || s.OwnerID != 1 {
		t.Fatalf("fx_deagle source state=%+v", s)
	}
}

func TestDeagleProjectilePushesPlayerOnce(t *testing.T) {
	m := OriginalMap1()
	victim := NewPlayer(2, m)
	victim.X = 100
	victim.Y = 200
	victim.VX = 0

	// Source checks frame.hitTest(_X,_Y,true) after adding vx/vy. Put the
	// thrown Deagle so its next point lands inside Symbol691's exact frame.
	shells := []Shell{{
		Kind: 12, X: 80, Y: 160, VX: 20, VY: 0,
		Facing: 1, OwnerID: 1, ScaleX: 1,
	}}
	updateShells(m, shells, []*Player{victim})
	if victim.VX != 20 {
		t.Fatalf("fx_deagle knockback vx=%v want20", victim.VX)
	}
	if victim.HitNumber != 1 || victim.HitTimer != 0 || !shells[0].HitsSomething {
		t.Fatalf("fx_deagle hit state victim hit=%d timer=%d shell=%+v", victim.HitNumber, victim.HitTimer, shells[0])
	}
	if shells[0].VX != -10 {
		t.Fatalf("fx_deagle rebound vx=%v want-10", shells[0].VX)
	}

	// hitsomething prevents a second player impact, exactly like Symbol170.
	before := victim.VX
	updateShells(m, shells, []*Player{victim})
	if victim.VX != before {
		t.Fatalf("fx_deagle hit twice: vx %v -> %v", before, victim.VX)
	}
}
