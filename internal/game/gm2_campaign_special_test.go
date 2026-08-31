package game

import "testing"

func TestGM2InvisibleMissionAIReentersInvisibility(t *testing.T) {
	m := OriginalMap1()
	p := NewPlayer(3, m)
	p.AI = true
	p.AISpecial = sourceAISpecialGM2Invisible
	p.Alpha = 1
	p.InvisibleTime = 0
	g := &Game{
		campaignMode:   true,
		campaignSetGM2: true,
		campaignLevel:  6,
		arena:          m,
		players:        []*Player{p},
	}

	g.prepareCampaignAISpecial(p)
	if p.InvisibleTime != 1 {
		t.Fatalf("GM2 invisible AI timer=%d want1 before source fade tick", p.InvisibleTime)
	}
	p.updatePowerupTimers()
	if p.InvisibleTime != 0 || p.Alpha >= 1 {
		t.Fatalf("GM2 invisible AI did not start fading out: timer=%d alpha=%v", p.InvisibleTime, p.Alpha)
	}

	for i := 0; i < 12; i++ {
		g.prepareCampaignAISpecial(p)
		p.updatePowerupTimers()
	}
	if p.Alpha > 0.01 {
		t.Fatalf("GM2 invisible AI remained visible: alpha=%v", p.Alpha)
	}
}

func TestGM2InvisibleMissionBulletHitRevealsThenHidesAgain(t *testing.T) {
	m := OriginalMap1()
	p := NewPlayer(3, m)
	p.AI = true
	p.AISpecial = sourceAISpecialGM2Invisible
	p.Team = 2
	p.Alpha = 0
	p.InvisibleTime = 1
	b := &Bullet{
		Kind: BulletNormal, X: p.X, Y: p.Y - 30, VX: 25,
		Firepower: 10, OwnerID: 1, Team: 1, Facing: 1,
	}
	if !hitPlayerWithBullet(m, b, []*Player{p}, false) {
		t.Fatal("source bullet did not hit GM2 invisible AI")
	}
	if p.InvisibleTime != -30 {
		t.Fatalf("GM2 invisible AI hit timer=%d want-30", p.InvisibleTime)
	}

	g := &Game{campaignMode: true, campaignSetGM2: true, campaignLevel: 6, arena: m, players: []*Player{p}}
	for i := 0; i < 12; i++ {
		g.prepareCampaignAISpecial(p)
		p.updatePowerupTimers()
	}
	if p.Alpha < 0.99 || p.InvisibleTime >= 0 {
		t.Fatalf("GM2 invisible AI should be visibly revealed during cooldown: timer=%d alpha=%v", p.InvisibleTime, p.Alpha)
	}
	for i := 0; i < 32; i++ {
		g.prepareCampaignAISpecial(p)
		p.updatePowerupTimers()
	}
	if p.Alpha >= 0.99 {
		t.Fatalf("GM2 invisible AI did not begin hiding again: timer=%d alpha=%v", p.InvisibleTime, p.Alpha)
	}
}

func TestGM2MiniMissionKeepsTinyGuyMini(t *testing.T) {
	m := OriginalMap1()
	p := NewPlayer(3, m)
	p.AI = true
	p.AISpecial = sourceAISpecialGM2Mini
	g := &Game{campaignMode: true, campaignSetGM2: true, campaignLevel: 11, arena: m, players: []*Player{p}}

	for i := 0; i < 40; i++ {
		g.prepareCampaignAISpecial(p)
		p.updatePowerupTimers()
	}
	if p.MiniTime != 499 {
		t.Fatalf("GM2 Tiny Guy minitime=%d want499 after source reset/tick", p.MiniTime)
	}
	if p.PlayerScale > 0.51 || p.MiniMulti > 0.61 {
		t.Fatalf("GM2 Tiny Guy did not stay mini: scale=%v minimulti=%v", p.PlayerScale, p.MiniMulti)
	}
}
