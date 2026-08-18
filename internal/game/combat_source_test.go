package game

import "testing"

func TestSourceBulletKinds(t *testing.T) {
	if BulletNormal != 1 || BulletShotgun != 2 || BulletInstagib != 3 || BulletMinigun != 6 {
		t.Fatalf("projectile linkage mapping changed: bullet=%d bullet2=%d bullet3=%d bullet6=%d",
			BulletNormal, BulletShotgun, BulletInstagib, BulletMinigun)
	}
}

func TestSourceBulletTrailWidth(t *testing.T) {
	w := sourceBulletTrailNaturalWidth()
	if w <= 20 {
		t.Fatalf("Symbol 368 source width=%v; expected natural trail wider than 20 px", w)
	}
}

func TestInstagibDelaysSelfDestructBySourceTimeline(t *testing.T) {
	arena := OriginalMap1()
	p := NewPlayer(2, arena)
	p.X, p.Y = 400, 300
	p.Lives = 5
	p.Active = true
	b := Bullet{
		Kind: BulletInstagib, X: p.X - 25, Y: p.Y - 30,
		VX: 25, Firepower: 25, OwnerID: 1, Team: 1, Facing: 1, Alpha: 1,
	}
	var trails []InstagibTrailEffect
	updateInstagibBullet(arena, &b, []*Player{p}, &trails)
	if p.Lives != 5 || !p.InstagibPlaying || p.InstagibFrame != 0 {
		t.Fatalf("bullet3 must start Symbol688 without instant death: lives=%d playing=%v frame=%d",
			p.Lives, p.InstagibPlaying, p.InstagibFrame)
	}
	if !b.Dead {
		t.Fatal("bullet3 should delete itself after player hit")
	}
	if len(trails) != 1 {
		t.Fatalf("bullet3 should emit one fx_instatrail per update; got %d", len(trails))
	}

	for i := 0; i < 60; i++ {
		p.Update(arena, nil, nil)
	}
	if p.Lives != 5 || !p.Active || !p.InstagibPlaying || p.InstagibFrame != 60 {
		t.Fatalf("victim died before Symbol688 frame62: lives=%d active=%v playing=%v frame=%d",
			p.Lives, p.Active, p.InstagibPlaying, p.InstagibFrame)
	}
	p.Update(arena, nil, nil)
	if p.Lives != 4 || !p.Active || p.InstagibPlaying || p.InstagibFrame != 0 {
		t.Fatalf("Symbol688 frame62 must SELFDESTRUCT then respawn: lives=%d active=%v playing=%v frame=%d",
			p.Lives, p.Active, p.InstagibPlaying, p.InstagibFrame)
	}
}

func TestInstagibVictimSourceTimeline(t *testing.T) {
	a := LoadAssets()
	a.EnsureGameplay()
	if a.InstagibVictim == nil || len(a.InstagibVictimTimeline) < 62 {
		t.Fatalf("missing Symbol687/688 victim effect: raster=%v frames=%d", a.InstagibVictim != nil, len(a.InstagibVictimTimeline))
	}
	if !a.InstagibVictimTimeline[1].Valid || !a.InstagibVictimTimeline[60].Valid {
		t.Fatal("Symbol688 visible countdown frames are missing")
	}
}

func TestShieldFactorsFromProjectileSources(t *testing.T) {
	arena := OriginalMap1()
	mkPlayer := func() *Player {
		p := NewPlayer(2, arena)
		p.X, p.Y = 400, 300
		p.VX = 0
		p.ShieldTime = 100
		p.Team = 2
		return p
	}

	normalTarget := mkPlayer()
	normal := Bullet{Kind: BulletNormal, X: normalTarget.X, Y: normalTarget.Y - 30, VX: 25, Firepower: 30, OwnerID: 1, Team: 1, Facing: 1}
	hitPlayerWithBullet(arena, &normal, []*Player{normalTarget}, false)
	if normalTarget.VX < 9.89 || normalTarget.VX > 9.91 { // 30 * .33
		t.Fatalf("normal bullet shield push=%v want 9.9", normalTarget.VX)
	}

	miniTarget := mkPlayer()
	mini := Bullet{Kind: BulletMinigun, X: miniTarget.X, Y: miniTarget.Y - 30, VX: 25, Firepower: 30, OwnerID: 1, Team: 1, Facing: 1}
	hitPlayerWithBullet(arena, &mini, []*Player{miniTarget}, true)
	if miniTarget.VX < 8.99 || miniTarget.VX > 9.01 { // 30 * .3
		t.Fatalf("bullet6 shield push=%v want 9", miniTarget.VX)
	}
}
