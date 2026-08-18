package game

import (
	"os"
	"strings"
	"testing"
)

func TestSourceAudioFilesAndGunMapping(t *testing.T) {
	a := newSourceAudioEngine()
	a.ensureAssets()
	for _, name := range []string{"menu.wav", "pistol3.wav", "hit1.wav", "die1.wav", "explosion1.wav", "music111", "music333"} {
		if got := a.soundPath(name); got == "" {
			t.Fatalf("source sound %q was not resolved", name)
		}
	}
	if got := a.soundPath("music111"); got == "" || !strings.HasSuffix(strings.ToLower(got), ".mp3") {
		t.Fatalf("music111 compact source path=%q want MP3 export", got)
	}
	if pcm := a.decodedPCM("drop1.wav"); len(pcm) == 0 {
		t.Fatal("drop1 source audio did not decode to PCM")
	}
	asset := a.assets["drop1.wav"]
	raw, err := os.ReadFile(asset.Path)
	if err != nil {
		t.Fatalf("read drop1 raw PCM: %v", err)
	}
	// drop1 is mono 11.025 kHz and the engine runs at 22.05 kHz. After stereo
	// expansion + resampling, a full payload must be at least ~4x the raw byte
	// count. The old sampleCount trimming produced a much shorter buffer.
	if pcm := a.decodedPCM("drop1.wav"); len(pcm) < len(raw)*4 {
		t.Fatalf("drop1 decoded PCM length=%d too short for full raw payload %d", len(pcm), len(raw))
	}
	checks := map[int]string{1: "pistol3.wav", 3: "pistol1.wav", 12: "smg1.wav", 33: "snipe4.wav", 44: "shotgun3.wav", 65: "lmg.wav"}
	for gun, want := range checks {
		if got := sourceGunSound(gun); got != want {
			t.Fatalf("gun %d sound=%q want %q", gun, got, want)
		}
	}
}

func TestM1911ReloadQueuesSourceSounds(t *testing.T) {
	p := &Player{Facing: 1, MiniMulti: 1, PlayerScale: 0.8}
	p.Weapon = NewWeapon(1)
	p.Weapon.Bullets = 0
	var shells []Shell
	p.Weapon.Playing = true
	enterWeaponTimelineFrame(p, &p.Weapon, 9, &shells, 0)
	for i := 0; i < 80 && (p.Weapon.Frame != 1 || p.Weapon.Playing); i++ {
		advanceWeaponTimeline(p, &shells)
	}
	seen := map[string]bool{}
	for _, name := range p.PendingSounds {
		seen[name] = true
	}
	if !seen["pistol_mag.wav"] || !seen["pistol_slide.wav"] {
		t.Fatalf("M1911 reload queued sounds=%v; want magazine and slide", p.PendingSounds)
	}
}
