package game

import (
	"encoding/binary"
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
	if got := a.soundPath("music111"); got == "" || !strings.HasSuffix(strings.ToLower(got), ".wav") {
		t.Fatalf("music111 source path=%q want original WAV", got)
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
	checks := map[int]string{
		1: "pistol3.wav", 3: "pistol1.wav", 12: "smg1.wav", 33: "snipe4.wav", 44: "shotgun3.wav", 65: "lmg.wav",
		87: "dev_launcher_rpg.wav", 89: "dev_javelin.wav", 98: "dev_firearm.wav", 124: "dev_firearm.wav", 139: "silenced1.wav", 146: "lmg.wav", 162: "shotgun1.wav", 178: "dev_bow.wav", 182: "dev_musket.wav", 185: "dev_energy.wav", 190: "dev_historical_pistol.wav", 192: "snipe6.wav", 193: "dev_musket.wav", 199: "dev_historical_lmg.wav", 204: "dev_historical_shotgun.wav", 209: "dev_sks.wav", 213: "dev_auto_rifle.wav",
	}
	for gun, want := range checks {
		if got := sourceGunSound(gun); got != want {
			t.Fatalf("gun %d sound=%q want %q", gun, got, want)
		}
	}
	for _, name := range []string{"dev_energy.wav", "dev_bow.wav", "dev_crossbow.wav", "dev_launcher.wav", "dev_grenade.wav"} {
		if pcm := a.decodedPCM(name); len(pcm) < 1000 {
			t.Fatalf("procedural sound %q PCM length=%d", name, len(pcm))
		}
	}
	for _, name := range []string{"dev_firearm.wav", "dev_launcher_rpg.wav", "dev_javelin.wav", "dev_musket.wav", "dev_historical_pistol.wav", "dev_historical_rifle.wav", "dev_sks.wav", "dev_historical_shotgun.wav", "dev_auto_rifle.wav", "dev_historical_lmg.wav"} {
		if path := a.soundPath(name); path == "" {
			t.Fatalf("custom CC0 sound %q was not resolved", name)
		}
		if pcm := a.decodedPCM(name); len(pcm) < 1000 {
			t.Fatalf("custom CC0 sound %q PCM length=%d", name, len(pcm))
		}
	}
}

func TestDevelopedGunSoundsUseLoudChannelOnly(t *testing.T) {
	if sourceGunSoundLoud(1) {
		t.Fatal("base weapon sound mix changed")
	}
	if !sourceGunSoundLoud(87) || !sourceGunSoundLoud(182) {
		t.Fatal("developed weapon sounds should use the loud channel")
	}
	if developedGunSoundGain(1) != 1 || developedGunSoundGain(182) < 1.2 {
		t.Fatal("developed weapon PCM gain is not enabled")
	}
	if !developedGunReusesOriginalSound(96) || !developedGunReusesOriginalSound(146) || !developedGunReusesOriginalSound(162) {
		t.Fatal("developed guns using the original SMG/LMG/shotgun bank were not detected")
	}
	if developedGunReusesOriginalSound(98) || developedGunSoundGain(96) != 1 || sourceGunSoundLoud(96) {
		t.Fatal("original-bank developed guns do not use the exact original mix")
	}
}

func TestAmplifyPCM16BoostsAndClipsSafely(t *testing.T) {
	pcm := make([]byte, 4)
	binary.LittleEndian.PutUint16(pcm[0:], uint16(int16(1000)))
	binary.LittleEndian.PutUint16(pcm[2:], uint16(int16(25000)))
	got := amplifyPCM16(pcm, 2)
	if sample := int16(binary.LittleEndian.Uint16(got[0:])); sample != 2000 {
		t.Fatalf("boosted sample=%d want2000", sample)
	}
	if sample := int16(binary.LittleEndian.Uint16(got[2:])); sample != 32767 {
		t.Fatalf("clipped sample=%d want32767", sample)
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
