package game

import "testing"

func TestM1911SourceTimeline(t *testing.T) {
	def := weaponTimeline(1)
	if def.TotalFrames != 60 {
		t.Fatalf("M1911 timeline frames=%d want 60", def.TotalFrames)
	}
	if def.RestFrame != 1 {
		t.Fatalf("M1911 rest frame=%d want zero-based 1 (Flash frame 2)", def.RestFrame)
	}
	if _, ok := def.Scripts[9]; !ok {
		t.Fatal("M1911 missing source frame 10 reload script")
	}
	if _, ok := def.Scripts[59]; !ok {
		t.Fatal("M1911 missing source frame 60 reload completion script")
	}
}

func TestM1911ReloadRunsSourceFrames(t *testing.T) {
	p := &Player{PerkNumber: 7, Facing: 1, MiniMulti: 1, PlayerScale: 0.8}
	p.Weapon = NewWeapon(1)
	p.Weapon.Bullets = 0
	var shells []Shell

	p.Weapon.Playing = true
	enterWeaponTimelineFrame(p, &p.Weapon, 9, &shells, 0) // gotoAndPlay(10)
	for i := 0; i < 80 && (p.Weapon.Frame != 1 || p.Weapon.Playing); i++ {
		advanceWeaponTimeline(p, &shells)
	}

	if p.Weapon.Frame != 1 || p.Weapon.Playing {
		t.Fatalf("reload did not return to stopped Flash frame 2: frame=%d playing=%v", p.Weapon.Frame, p.Weapon.Playing)
	}
	if p.Weapon.Bullets != 12 {
		t.Fatalf("M1911 reload ammo=%d want 12 (9 + ceil(9*0.333) perk 7)", p.Weapon.Bullets)
	}
	if len(shells) == 0 {
		t.Fatal("M1911 source reload/fire timeline emitted no shell event")
	}
	foundMag := false
	for _, s := range shells {
		if s.Kind == 7 { // Symbol335 fx_dropmag, source frame 19
			foundMag = true
			break
		}
	}
	if !foundMag {
		t.Fatal("M1911 reload missed source fx_dropmag event from Flash frame 19")
	}
}
