package game

import "testing"

func TestKillFeedSourceFadeTiming(t *testing.T) {
	entries := []KillFeedEntry{{Victim: "P2", Killer: "P1", Mod: 1, Slot: 1}}
	for i := 0; i < 10; i++ {
		updateKillFeeds(entries)
	}
	if entries[0].Alpha != 1 {
		t.Fatalf("source killfeed alpha after10=%v want1", entries[0].Alpha)
	}
	for i := 10; i < 110; i++ {
		updateKillFeeds(entries)
	}
	if entries[0].Dead || entries[0].Alpha != 1 {
		t.Fatalf("killfeed changed before time>110: time=%d alpha=%v dead=%v", entries[0].Time, entries[0].Alpha, entries[0].Dead)
	}
	for i := 0; i < 10; i++ {
		updateKillFeeds(entries)
	}
	if !entries[0].Dead {
		t.Fatalf("killfeed did not disappear after source fade: time=%d alpha=%v", entries[0].Time, entries[0].Alpha)
	}
}

func TestKillFeedHiddenInSurvivalAndCampaignTutorial(t *testing.T) {
	p := &Player{Name: "Victim", LastDeathMod: 1}
	g := &Game{GameMode: SourceGameModeSurvival}
	g.appendSourceKillFeed(p)
	if len(g.killfeeds) != 0 {
		t.Fatal("source survival must not create killfeed")
	}
	g.GameMode = SourceGameModeNormal
	g.campaignMode = true
	g.campaignLevel = 1
	g.appendSourceKillFeed(p)
	if len(g.killfeeds) != 0 {
		t.Fatal("source campaign tutorial must not create killfeed")
	}
}
