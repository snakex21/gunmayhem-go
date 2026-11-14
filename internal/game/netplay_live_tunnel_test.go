package game

import (
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"
)

// TestCloudflareLiveNetplaySession runs a real host+client session through a
// Cloudflare Quick Tunnel and reports bandwidth, ping and playout quality.
//
//	GUNMAYHEM_CLOUDFLARE_LIVE_TEST=1 GUNMAYHEM_CLOUDFLARED=build/tools/cloudflared.exe \
//	go test ./internal/game -run TestCloudflareLiveNetplaySession -v -timeout 5m
//
// GUNMAYHEM_LIVE_SECONDS overrides the 30 s session length.
func TestCloudflareLiveNetplaySession(t *testing.T) {
	if os.Getenv("GUNMAYHEM_CLOUDFLARE_LIVE_TEST") != "1" {
		t.Skip("set GUNMAYHEM_CLOUDFLARE_LIVE_TEST=1 to create a real Quick Tunnel")
	}
	seconds := 30
	if v, err := strconv.Atoi(os.Getenv("GUNMAYHEM_LIVE_SECONDS")); err == nil && v > 0 {
		seconds = v
	}
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	host.setAllowedRemotePlayers([5]bool{false, true, true, true, true})
	tunnel, err := startCloudflareQuickTunnel(host)
	if err != nil {
		t.Fatal(err)
	}
	defer tunnel.Close()
	tunnel.monitorStartup()
	var wsURL string
	deadline := time.Now().Add(time.Duration(cloudflareTunnelMaxAttempts) * (cloudflareTunnelStartWindow + 5*time.Second))
	for {
		ready, _, url, statusErr := tunnel.Status()
		if ready {
			wsURL = url
			break
		}
		if statusErr != nil {
			t.Fatal(statusErr)
		}
		if time.Now().After(deadline) {
			t.Fatal("Quick Tunnel did not become ready")
		}
		time.Sleep(250 * time.Millisecond)
	}
	client, err := newClientNetplayCloudflare(wsURL, "", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := waitForClientHandshake(client, 6*time.Second); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "host peer", host.connected)
	host.peersMu.RLock()
	peer := host.peers[2]
	host.peersMu.RUnlock()

	rand.Seed(11)
	quiet, busy := quietNetTestScene(), realisticNetSnapshot()
	tick := time.NewTicker(time.Second / 30)
	defer tick.Stop()
	start := time.Now()
	startOut := peer.link.bytesOut.Load()
	frames, holds, lastFX := 0, 0, -1
	for i := 0; time.Since(start) < time.Duration(seconds)*time.Second; i++ {
		<-tick.C
		// Alternate 5 s quiet / 5 s busy.
		scene := &quiet
		if (i/150)%2 == 1 {
			scene = &busy
		}
		netStepTestScene(scene, i)
		state := *scene
		state.Players = append([]Player(nil), scene.Players...)
		state.MapFXFrame = i
		host.queueState(state)
		client.queueInput(PlayerInputState{Right: i%20 < 10, Shoot: i%7 == 0})
		if s, ok := client.nextPlayoutState(); ok {
			frames++
			if s.MapFXFrame == lastFX {
				holds++
			}
			lastFX = s.MapFXFrame
		}
		_ = host.takeInputFor(2)
		if !client.connected() {
			t.Fatalf("client disconnected after %v: %v", time.Since(start), client.LastError())
		}
	}
	elapsed := time.Since(start).Seconds()
	sent := peer.link.bytesOut.Load() - startOut
	hostRTT, _ := host.peerRTTMillis(2)
	clientRTT, _ := client.hostRTTMillis()
	t.Logf("live tunnel %ds: host->client %.1f KB/s (%.0f kbit/s), client->host %.1f KB/s total, rtt host=%dms client=%dms, playout frames=%d holds=%d (%.1f%%), ack lag=%d ticks",
		seconds, float64(sent)/elapsed/1024, float64(sent)*8/elapsed/1000,
		float64(peer.link.bytesIn.Load())/elapsed/1024, hostRTT, clientRTT, frames, holds, 100*float64(holds)/float64(max(frames, 1)), peer.inflightTicks())
}
