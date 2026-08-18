package game

import "testing"

func TestGM2WeaponLibraryExposesAllTwentyAdditions(t *testing.T) {
	seen := map[int]bool{}
	for n := 67; n <= 86; n++ {
		if !isGunLibraryGunNumber(n) {
			t.Fatalf("GM2 weapon %d is not accepted by Gun Library", n)
		}
		r := gm2GunLibraryRect(n)
		if r.W <= 0 || r.H <= 0 {
			t.Fatalf("GM2 weapon %d has no visible library card", n)
		}
		hit := gm2GunLibraryHitAt(r.X+r.W/2, r.Y+r.H/2)
		if hit != n {
			t.Fatalf("GM2 weapon card %d hit=%d", n, hit)
		}
		seen[hit] = true
	}
	if len(seen) != 20 {
		t.Fatalf("visible GM2 weapon cards=%d want20", len(seen))
	}
}

func TestGM2CampaignExposesSixteenMissions(t *testing.T) {
	for i := 0; i < 16; i++ {
		if gm2CampaignTitles[i] == "" {
			t.Fatalf("GM2 mission %d has no visible title", i+1)
		}
		r := gm2CampaignCardRect(i)
		if r.W <= 0 || r.H <= 0 {
			t.Fatalf("GM2 mission %d has no visible card", i+1)
		}
		if gm2CampaignMissions[i].MapNumber < 1 || gm2CampaignMissions[i].MapNumber > 21 {
			t.Fatalf("GM2 mission %d bad map=%d", i+1, gm2CampaignMissions[i].MapNumber)
		}
	}
}

func TestGM2CampaignProgressStaysSeparateFromGM1(t *testing.T) {
	g := New()
	gm1 := g.campaignLevels
	g.completeGM2CampaignLevel(2)
	if g.gm2CampaignLevels[1] != 2 || g.gm2CampaignLevels[2] != 1 {
		t.Fatalf("GM2 progression=%v want mission2 complete and mission3 unlocked", g.gm2CampaignLevels[:4])
	}
	if g.campaignLevels != gm1 {
		t.Fatalf("GM2 completion changed GM1 progress: got=%v want=%v", g.campaignLevels, gm1)
	}
}
