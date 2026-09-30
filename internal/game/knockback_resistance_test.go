package game

import (
	"math"
	"testing"
)

func TestKnockbackResistanceScalesOnlyNewImpulse(t *testing.T) {
	p := &Player{PerkNumber: perkKnockbackResistance, VX: 5, VY: -2}
	p.applyCombatImpulse(20, -10)
	if p.VX != 19 || p.VY != -9 {
		t.Fatalf("velocity=%v,%v; existing movement was scaled", p.VX, p.VY)
	}
	for perk := 1; perk <= 9; perk++ {
		p := &Player{PerkNumber: perk, VX: 5, VY: -2}
		p.applyCombatImpulse(20, -10)
		if p.VX != 25 || p.VY != -12 {
			t.Fatalf("original perk%d behavior changed", perk)
		}
	}
}

func TestKnockbackResistanceAcrossCombatSources(t *testing.T) {
	cases := []struct {
		name string
		hit  func(*Player)
	}{
		{"bullet", func(p *Player) {
			b := Bullet{X: 140, Y: 76, OwnerID: 1, Facing: 1, Firepower: 20}
			if !hitPlayerWithBullet(Map{}, &b, []*Player{p}, false) {
				t.Fatal("bullet missed")
			}
		}},
		{"shielded minigun", func(p *Player) {
			p.ShieldTime = 20
			b := Bullet{X: 140, Y: 76, OwnerID: 1, Facing: -1, Firepower: 20}
			if !hitPlayerWithBullet(Map{}, &b, []*Player{p}, true) {
				t.Fatal("minigun missed")
			}
		}},
		{"grenade", func(p *Player) { explodeGrenade(&Grenade{X: 100, Y: 90, OwnerID: 1, Type: ThrowableMk2}, []*Player{p}) }},
		{"rocket", func(p *Player) {
			explodeDevelopedProjectile(&Bullet{X: 100, Y: 90, OwnerID: 1, ExplosionPower: 60, ExplosionRadius: 180}, []*Player{p}, nil, nil)
		}},
		{"ignition", func(p *Player) { p.igniteWithPower(1, 1, 60, 20) }},
		{"fire contact", func(p *Player) { p.applyFireContactKnockback(1, -1, 5, 2, 8) }},
		{"burn tick", func(p *Player) {
			p.BurnTime = 60
			p.BurnPower = 20
			p.BurnTick = 9
			p.BurnOwnerID = 1
			p.BurnDirection = 1
			p.updateBurning()
		}},
		{"molotov", func(p *Player) { spawnMolotovFire(Map{}, &Grenade{X: 100, Y: 90, OwnerID: 1}, []*Player{p}, nil) }},
		{"reload shell", func(p *Player) {
			updateShells(Map{}, []Shell{{X: 140, Y: 76, OwnerID: 1, Facing: 1, Kind: 12}}, []*Player{p})
		}},
		{"instagib impulse", func(p *Player) {
			b := Bullet{X: 140, Y: 76, OwnerID: 1, Facing: 1, Firepower: 20}
			var trails []InstagibTrailEffect
			updateInstagibBullet(Map{}, &b, []*Player{p}, &trails)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base := &Player{ID: 2, Active: true, X: 140, Y: 100, PerkNumber: 1, DamageMulti: 1}
			resist := *base
			resist.PerkNumber = perkKnockbackResistance
			tc.hit(base)
			tc.hit(&resist)
			if base.VX == 0 && base.VY == 0 {
				t.Fatal("fixture did not apply an impulse")
			}
			if math.Abs(resist.VX-base.VX*knockbackResistanceMultiplier) > 1e-9 || math.Abs(resist.VY-base.VY*knockbackResistanceMultiplier) > 1e-9 {
				t.Fatalf("impulse=%v,%v base=%v,%v", resist.VX, resist.VY, base.VX, base.VY)
			}
			if resist.LastHitBy != base.LastHitBy || resist.HitByGrenade != base.HitByGrenade || resist.HitNumber != base.HitNumber {
				t.Fatal("perk changed attribution/hit events")
			}
			if resist.BurnTime != base.BurnTime {
				t.Fatal("perk shortened burn duration")
			}
			if resist.InstagibPlaying != base.InstagibPlaying {
				t.Fatal("perk changed Instagib lethality")
			}
		})
	}
}

func TestKnockbackResistanceInstagibStillKills(t *testing.T) {
	p := &Player{ID: 2, Active: true, X: 140, Y: 100, PerkNumber: perkKnockbackResistance}
	b := Bullet{X: 140, Y: 76, OwnerID: 1, Facing: 1, Firepower: 20}
	var trails []InstagibTrailEffect
	updateInstagibBullet(Map{}, &b, []*Player{p}, &trails)
	for i := 0; i < 62; i++ {
		p.advanceInstagibTimeline()
	}
	if !p.KillSelf {
		t.Fatal("resistance canceled delayed Instagib death")
	}
}

func TestKnockbackResistanceCustomPickerAndCampaignIsolation(t *testing.T) {
	g := &Game{}
	g.customPlayers = sourceDefaultCustomPlayers()
	g.customPlayers[0].Type = 1
	g.customEditor[0] = customEditPerk
	g.customCardY[0] = -300
	originalCount := len(sourceCustomSelectionItems()[customEditPerk])
	found := false
	for _, item := range developedCustomSelectionItems(customEditPerk) {
		if item.Number != perkKnockbackResistance {
			continue
		}
		found = true
		x := 20 + item.Rect.X + item.Rect.W/2
		y := -300 + item.Rect.Y + item.Rect.H/2
		if y < 100 || y >= 500 {
			t.Fatalf("new perk is clipped outside the player card: y=%v", y)
		}
		hit := g.customPlayerEditorHitAt(x, y)
		if hit == menuHitNone || !g.activateCustomPlayerEditorHit(hit) || g.customPlayers[0].Perk != perkKnockbackResistance {
			t.Fatal("new perk cannot be selected through its actual tile")
		}
	}
	if !found {
		t.Fatal("custom picker omitted new perk")
	}
	if len(sourceCustomSelectionItems()[customEditPerk]) != originalCount {
		t.Fatal("source cache was mutated")
	}
	for _, extra := range []bool{false, true} {
		g.campaignSetExtra = extra
		if g.campaignSelectionUnlocked(customEditPerk, perkKnockbackResistance) {
			t.Fatal("new perk leaked into campaign balance")
		}
	}
	if onlineCharacterVisualChoiceCount(onlineVisualCharacterPerk) != maxCustomPerk {
		t.Fatal("online picker omitted new perk")
	}
}

func TestKnockbackResistanceBlocklistAndConfigValidation(t *testing.T) {
	g := New()
	g.onlinePerksEnabled = true
	if g.onlineAllowedPerk(perkKnockbackResistance) != perkKnockbackResistance {
		t.Fatal("enabled perk rejected")
	}
	r := onlinePerkBlocklistRect(perkKnockbackResistance)
	if r.Y+r.H >= 445 {
		t.Fatal("new blocklist tile overlaps count/footer")
	}
	if g.onlinePerkBlocklistHitAt(r.X+r.W/2, r.Y+r.H/2) != menuHitOnlinePerkBase+perkKnockbackResistance {
		t.Fatal("new perk missing from blocklist hit testing")
	}
	g.onlinePerkBlocklistOpen = true
	if !g.activateOnlinePerkBlocklistHit(g.onlinePerkBlocklistHitAt(r.X+r.W/2, r.Y+r.H/2)) {
		t.Fatal("new blocklist tile cannot activate")
	}
	if g.onlineAllowedPerk(perkKnockbackResistance) != 0 {
		t.Fatal("host blocklist ignored")
	}
	if !g.activateOnlinePerkBlocklistHit(menuHitOnlinePerkBase+perkKnockbackResistance) || g.onlineBlockedPerks[perkKnockbackResistance] {
		t.Fatal("repeated click did not unblock perk")
	}
	if g.activateOnlinePerkBlocklistHit(menuHitOnlinePerkBase + maxCustomPerk + 1) {
		t.Fatal("unknown blocklist ID accepted")
	}
	g.onlinePerksEnabled = false
	if g.onlineAllowedPerk(perkKnockbackResistance) != 0 {
		t.Fatal("host perks-off ignored")
	}
	cfg := sourceDefaultCustomPlayers()[0]
	cfg.Perk = perkKnockbackResistance
	if _, err := sanitizeRemotePlayerConfig(cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Perk = maxCustomPerk + 1
	if _, err := sanitizeRemotePlayerConfig(cfg); err == nil {
		t.Fatal("unknown remote perk accepted")
	}
}

func TestKnockbackResistanceAuthoritativeSnapshotRoundTrip(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	client, err := newClientNetplay(host.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	waitFor(t, "perk client", client.connected)
	cfg := sourceDefaultCustomPlayers()[1]
	cfg.Perk = perkKnockbackResistance
	client.queuePlayerConfig(cfg)
	waitFor(t, "new perk host validation", func() bool { c, ok := host.latestConfigFor(2); return ok && c.Perk == perkKnockbackResistance })
	p := Player{ID: 2, Name: "Resistant", PerkNumber: perkKnockbackResistance}
	p.applyCombatImpulse(20, -10)
	state := netSnapshot{Players: []Player{p}}
	state.OnlineCustomPlayers[1] = cfg
	state.OnlineBlockedPerks[perkKnockbackResistance] = true
	host.queueState(state)
	waitFor(t, "new perk authoritative wire state", func() bool {
		s, ok := client.latestState()
		return ok && len(s.Players) == 1 && s.Players[0].VX == 14 && s.Players[0].VY == -7 && s.Players[0].PerkNumber == perkKnockbackResistance && s.OnlineCustomPlayers[1].Perk == perkKnockbackResistance && s.OnlineBlockedPerks[perkKnockbackResistance]
	})
	if err := validateNetMessageForHost(netWireMessage{Protocol: 19, Type: netMessageHello}); err == nil {
		t.Fatal("network-only old protocol accepted expanded perk schema")
	}
}

func TestKnockbackResistancePortalPulseAndTeleport(t *testing.T) {
	p := &Player{ID: 2, Active: true, PerkNumber: perkKnockbackResistance}
	b := &Bullet{OwnerID: 1, Facing: -1, PortalColor: PortalBlue}
	portalCombatHit(b, p, nil)
	if math.Abs(p.VX+24*.7) > 1e-9 || math.Abs(p.VY+8*.7) > 1e-9 || p.LastHitBy != 1 {
		t.Fatal("unpaired portal pulse did not apply resistance")
	}
	portals := []Portal{{OwnerID: 1, Color: PortalOrange, Active: true, X: 400, Y: 200, NormalX: 1}}
	for _, perk := range []int{1, perkKnockbackResistance} {
		p := &Player{ID: 2, Active: true, PerkNumber: perk}
		portalCombatHit(&Bullet{OwnerID: 1, PortalColor: PortalBlue}, p, &portals)
		if p.X != 458 || p.Y != 228 || p.VX != 19 || p.VY != 0 || !p.FreePass || p.PortalCooldown != 26 {
			t.Fatalf("perk%d altered paired portal movement: %+v", perk, p)
		}
	}
}

func TestKnockbackResistanceDoesNotChangeMovementRecoilOrWind(t *testing.T) {
	arena := OriginalMap1()
	base := NewPlayer(2, arena)
	base.PerkNumber = 1
	base.AI = false
	base.RemoteControlled = true
	base.RemoteInput = PlayerInputState{Right: true, Up: true}
	resist := *base
	resist.PerkNumber = perkKnockbackResistance
	for i := 0; i < 15; i++ {
		base.Update(arena, nil, nil)
		resist.Update(arena, nil, nil)
	}
	if base.X != resist.X || base.Y != resist.Y || base.VX != resist.VX || base.VY != resist.VY {
		t.Fatal("resistance changed movement or jump")
	}
	for _, p := range []*Player{base, &resist} {
		p.Active = true
		p.VX = 0
		p.Facing = 1
		p.RemoteInput = PlayerInputState{Shoot: true}
		p.Weapon = NewWeapon(1)
		p.Weapon.WaitTime = p.Weapon.Def.ROF
		p.Weapon.Frame = 1
		var bullets []Bullet
		var shells []Shell
		var flashes []MuzzleFlash
		updateWeapon(p, &bullets, &shells, &flashes)
		if len(bullets) == 0 {
			t.Fatal("recoil fixture did not fire")
		}
	}
	if base.VX != resist.VX {
		t.Fatal("resistance changed self recoil")
	}
	g := &Game{players: []*Player{base, &resist}}
	g.applyChaosWind(2)
	if base.VX != resist.VX {
		t.Fatal("resistance changed environmental wind")
	}
}
