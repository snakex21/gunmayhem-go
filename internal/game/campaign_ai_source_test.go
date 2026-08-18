package game

import "testing"

func TestCampaignSpecialAIClassMappingMatchesFrame10(t *testing.T) {
	want := map[int]int{
		4:  sourceAISpecialDoubleSpawner,
		7:  sourceAISpecialJetpack,
		9:  sourceAISpecialInvisible,
		10: sourceAISpecialBoss,
	}
	for level := 1; level <= 10; level++ {
		got := campaignAISpecialForLevel(level)
		if w, ok := want[level]; ok {
			if got != w {
				t.Fatalf("level%d special=%d want%d", level, got, w)
			}
		} else if got != sourceAISpecialNone {
			t.Fatalf("level%d unexpectedly has special=%d", level, got)
		}
	}
}

func TestCampaignJetpackAIRespawnRestoresFullFuel(t *testing.T) {
	m := OriginalMap1()
	p := NewPlayer(4, m)
	p.AI = true
	p.AISpecial = sourceAISpecialJetpack
	p.JetFuel = 0
	p.Respawn(m)
	if p.JetFuel != 100 {
		t.Fatalf("playerAI3 jetfuel=%v want100", p.JetFuel)
	}
}

func TestCampaignInvisibleAITracksGothitbyState(t *testing.T) {
	m := OriginalMap1()
	p := NewPlayer(4, m)
	p.AI = true
	p.AISpecial = sourceAISpecialInvisible
	g := &Game{campaignMode: true, campaignLevel: 9, arena: m, players: []*Player{p}}
	p.LastHitBy = 0
	g.prepareCampaignAISpecial(p)
	if p.InvisibleTime != 10 {
		t.Fatalf("playerAI4 invisibletime=%d want10 while gothitby none", p.InvisibleTime)
	}
	p.LastHitBy = 1
	g.prepareCampaignAISpecial(p)
	if p.InvisibleTime != 0 {
		t.Fatalf("playerAI4 invisibletime=%d want0 after hit", p.InvisibleTime)
	}
}

func TestCampaignLevel5SpawnsNeutralDynamiteEvery80Ticks(t *testing.T) {
	m := OriginalMap1()
	g := &Game{campaignMode: true, campaignLevel: 5, arena: m}
	for i := 0; i < 79; i++ {
		g.updateCampaignDynamiteRain()
	}
	if len(g.grenades) != 0 {
		t.Fatalf("dynamite spawned early at tick79: %d", len(g.grenades))
	}
	g.updateCampaignDynamiteRain()
	if len(g.grenades) != 1 {
		t.Fatalf("tick80 grenades=%d want1", len(g.grenades))
	}
	gr := g.grenades[0]
	if gr.Y != -200 || gr.OwnerID != 0 || gr.VX != 0 || gr.VY != 0 {
		t.Fatalf("source falling dynamite mismatch: %+v", gr)
	}
	if gr.X < m.CrateMinX || gr.X >= m.CrateMaxX {
		t.Fatalf("dynamite x=%f outside crate area", gr.X)
	}
}

func TestCampaignPylonSpawnerCreatesPersistentDouble(t *testing.T) {
	m := OriginalMap1()
	p := NewPlayer(4, m)
	p.AI = true
	p.Name = "Pylon Man"
	p.AISpecial = sourceAISpecialDoubleSpawner
	p.AIFakeDoubleTime = 34
	p.DefaultWeapon = 1
	p.VX = 13
	p.VY = -7
	p.AITargetTimer = 35
	p.AITargetValid = true
	p.AITargetKind = aiTargetPlayer
	p.AITargetPlayerID = 1
	p.AILockLeft = 9
	p.AIRight = true
	g := &Game{
		campaignMode: true, campaignLevel: 4, arena: m,
		players: []*Player{p}, seenDeaths: map[int]int{4: 0}, nextEntityID: 5,
	}
	g.prepareCampaignAISpecial(p)
	if !p.WantsDouble || p.AIFakeDoubleTime != -120 {
		t.Fatalf("playerAI_double spawn trigger wants=%v timer=%d", p.WantsDouble, p.AIFakeDoubleTime)
	}
	spawnX, spawnY := p.X, p.Y
	// Source spawnfriend() captures Pylon Man's position before his movement in
	// this frame. Move the owner here to ensure the helper uses the stored point.
	p.X += p.VX
	p.Y += p.VY
	g.spawnRequestedDoubles()
	if len(g.players) != 2 {
		t.Fatalf("campaign double count=%d want2 players", len(g.players))
	}
	d := g.players[1]
	if !d.IsDouble || !d.PersistentDouble || d.OwnerPlayerID != p.ID || d.Name != p.Name || d.Lives != 1 {
		t.Fatalf("campaign double mismatch: %+v", d)
	}
	if d.X != spawnX || d.Y != spawnY || d.VX != 0 || d.VY != 0 {
		t.Fatalf("spawnfriend source position/velocity mismatch: helper=(%v,%v,%v,%v) preMove=(%v,%v) ownerNow=(%v,%v)",
			d.X, d.Y, d.VX, d.VY, spawnX, spawnY, p.X, p.Y)
	}
	if d.AITargetTimer != 0 || d.AITargetValid || d.AILockLeft != 0 || d.AILockRight != 0 || d.AIRight {
		t.Fatalf("campaign helper inherited owner AI state: timer=%d valid=%v locks=%d/%d right=%v",
			d.AITargetTimer, d.AITargetValid, d.AILockLeft, d.AILockRight, d.AIRight)
	}
	if d.CheapTimer != 120 {
		t.Fatalf("campaign helper cheapshottimer=%d want120", d.CheapTimer)
	}
}

func TestCampaignPylonHelperUsesRegularPlayerAICampaignBlock(t *testing.T) {
	m := OriginalMap1()
	d := NewPlayer(5, m)
	d.AI = true
	d.IsDouble = true
	d.PersistentDouble = true
	d.Lives = 1
	g := &Game{campaignMode: true, campaignLevel: 4, arena: m, players: []*Player{d}}

	g.prepareCampaignAISpecial(d)
	if d.Alpha < 0.20 || d.Alpha > 0.49 || !d.Active {
		t.Fatalf("campaign helper alpha/active=%v/%v want 0.20..0.49 active", d.Alpha, d.Active)
	}
	d.LastHitBy = 1
	g.prepareCampaignAISpecial(d)
	if d.Active || d.Lives != 0 {
		t.Fatalf("campaign helper must self-destruct after hit: active=%v lives=%d", d.Active, d.Lives)
	}
}

func TestCampaignBossSourceModifiers(t *testing.T) {
	m := OriginalMap1()
	p := NewPlayer(4, m)
	p.AI = true
	p.AISpecial = sourceAISpecialBoss
	p.DamageMulti = 0.8
	p.FirepowerMulti = 1
	p.PerkNumber = 6
	p.DefaultWeapon = 55
	p.EquipWeapon(55)
	base := NewWeapon(55)
	if p.Weapon.Bullets != base.Bullets*4 {
		t.Fatalf("boss ammo=%d want%d", p.Weapon.Bullets, base.Bullets*4)
	}
	if p.Weight != 1 {
		t.Fatalf("boss weight=%v want1", p.Weight)
	}
	if p.DamageMulti != 0.8 {
		t.Fatalf("boss damage multiplier=%v want0.8", p.DamageMulti)
	}
}
