package game

import (
	"fmt"
	"strings"
	"testing"
)

// TestOnlinePostGameCountdownShowsRealSeconds runs a real online round to
// Post Game, readies everyone and checks every tick of the countdown: the
// shown number must be the real seconds left at 30 TPS (it used to divide
// by 60, so a 6 s countdown read 3..1).
func TestOnlinePostGameCountdownShowsRealSeconds(t *testing.T) {
	g := newSeriesRound(t, 2)
	peer := &netplayPeer{playerID: 2, stateSend: make(chan *netHostFrame, 1)}
	g.netplay = &netplaySession{mode: netplayHost, peers: map[int]*netplayPeer{2: peer}}
	g.onlineMatchSetup = true
	playRoundToPostGame(t, g, 2)

	g.markLocalPostGameReady()
	peer.postGameReady.Store(true)
	_ = g.Update()
	if g.postGameReadyCountdown != postGameReadyCountdownFrames {
		t.Fatalf("countdown did not start: %d", g.postGameReadyCountdown)
	}
	wantFirst := fmt.Sprintf("NEXT ROUND IN %d", postGameReadyCountdownFrames/TicksPerSecond)
	if txt := g.onlinePostGameStatusText(); !strings.Contains(txt, wantFirst) {
		t.Fatalf("status %q, want %q", txt, wantFirst)
	}
	ticks := 0
	for g.postGameReadyCountdown > 0 && ticks < 1000 {
		left := g.postGameReadyCountdown
		want := fmt.Sprintf("NEXT ROUND IN %d", (left+TicksPerSecond-1)/TicksPerSecond)
		if txt := g.onlinePostGameStatusText(); !strings.Contains(txt, want) {
			t.Fatalf("%d ticks left: status %q, want %q", left, txt, want)
		}
		_ = g.Update()
		ticks++
	}
	// The countdown lasts exactly as many real seconds as its first number,
	// then the next round starts.
	if g.customSeriesRound != 2 {
		t.Fatalf("next round not started: round %d", g.customSeriesRound)
	}
	if got := ticks / TicksPerSecond; got != postGameReadyCountdownFrames/TicksPerSecond {
		t.Fatalf("countdown ran %d ticks (%d s)", ticks, got)
	}
}

// TestChaosIntervalIsRealSeconds: "EVERY 10s" must fire the next chaos event
// after 10 real seconds of match ticks, and the HUD counts it down in real
// seconds (frames were seconds*60 at 30 TPS: twice as long).
func TestChaosIntervalIsRealSeconds(t *testing.T) {
	g := New()
	g.customLives = 3
	g.customSeriesRounds = 1
	for i := range g.customPlayers {
		g.customPlayers[i].Type = 0
	}
	g.customPlayers[0].Type = 1
	g.customPlayers[1].Type = 1
	g.customChaosEnabled = true
	g.customChaosInterval = 10
	g.initCustomChaosDefaults()
	g.startCustomGame()
	g.fadeActive = false
	g.screen = screenGameplay
	for _, p := range g.players {
		p.RemoteControlled = true
		p.Lives, p.TotalLives = 999, 999
	}
	activations := 0
	lastFrame := -1
	for tick := 0; tick < 10*TicksPerSecond+5 && g.screen == screenGameplay; tick++ {
		_ = g.Update()
		if g.chaosFrame < lastFrame || (lastFrame == -1 && g.chaosCurrentEffect != chaosEffectNone) {
			activations++
		}
		lastFrame = g.chaosFrame
		if g.chaosCurrentEffect != chaosEffectNone {
			left := 10*TicksPerSecond - g.chaosFrame
			if got, want := g.chaosSecondsLeft(), (left+TicksPerSecond-1)/TicksPerSecond; got != want {
				t.Fatalf("chaos HUD %d s with %d ticks left (want %d)", got, left, want)
			}
		}
	}
	if activations != 2 {
		t.Fatalf("chaos events in 10 s + 5 ticks: %d, want the first and one after 10 s", activations)
	}
}

// TestExtraHazardHUDShowsRealSeconds: the EXTRA hazard countdown shows the
// real seconds to the next strike after running the mission.
func TestExtraHazardHUDShowsRealSeconds(t *testing.T) {
	g := newExtraMissionMidFight(t, 3, 0) // crosswind gusts
	interval := g.extraHazardInterval()
	if interval <= TicksPerSecond {
		t.Fatalf("hazard interval %d", interval)
	}
	if got, want := g.extraHazardSecondsLeft(), float64(interval)/TicksPerSecond; got != want {
		t.Fatalf("start: %.2f s, want %.2f", got, want)
	}
	for i := 0; i < TicksPerSecond; i++ {
		_ = g.Update()
	}
	if got, want := g.extraHazardSecondsLeft(), float64(interval-g.extraHazardFrame)/TicksPerSecond; got != want || g.extraHazardFrame == 0 {
		t.Fatalf("after 1 s: %.2f s (frame %d), want %.2f", got, g.extraHazardFrame, want)
	}
}

// TestAbilityDescriptionsMatchTicks: the loadout text states the real
// cooldown / hold time of the tunables at 30 TPS.
func TestAbilityDescriptionsMatchTicks(t *testing.T) {
	if d := abilityDescription(abilityDash); !strings.Contains(d, framesAsSeconds(dashCooldownFrames)+" s cooldown") || framesAsSeconds(dashCooldownFrames) != "8" {
		t.Fatalf("dash description %q", d)
	}
	if d := abilityDescription(abilityAnchor); !strings.Contains(d, "Max 3 s") {
		t.Fatalf("anchor description %q", d)
	}
}
