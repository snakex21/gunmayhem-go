package game

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// The published Gun Mayhem music and the majority of weapon SFX are 22.05 kHz.
// Keeping the engine at the source rate avoids needlessly upsampling everything
// to 44.1 kHz; the few 11.025 kHz effects are resampled only once to 22.05 kHz.
const sourceAudioSampleRate = 22050

type sourceSoundAsset struct {
	Path        string
	RawPCM      bool
	SampleRate  int
	Channels    int
	SampleCount int64
}

type sourceAudioEngine struct {
	mu        sync.Mutex
	ctx       *audio.Context
	assets    map[string]sourceSoundAsset
	pcm       map[string][]byte
	amplified map[string][]byte
	music     *audio.Player
	musicName string
}

func newSourceAudioEngine() *sourceAudioEngine {
	return &sourceAudioEngine{pcm: make(map[string][]byte), amplified: make(map[string][]byte)}
}

func (a *sourceAudioEngine) context() *audio.Context {
	if a == nil {
		return nil
	}
	if a.ctx != nil {
		return a.ctx
	}
	if current := audio.CurrentContext(); current != nil {
		a.ctx = current
	} else {
		a.ctx = audio.NewContext(sourceAudioSampleRate)
	}
	return a.ctx
}

func (a *sourceAudioEngine) ensureAssets() {
	if a == nil || a.assets != nil {
		return
	}
	a.assets = map[string]sourceSoundAsset{}

	// Prefer FFDec's compact MP3 exports. They are equivalent to the published
	// Flash audio but are dramatically smaller than the XFL raw PCM payloads.
	soundsDir, err := findOriginalPath("sounds")
	if err == nil {
		if entries, readErr := os.ReadDir(soundsDir); readErr == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				if !strings.HasSuffix(strings.ToLower(name), ".mp3") {
					continue
				}
				base := name[:len(name)-len(".mp3")]
				underscore := strings.IndexByte(base, '_')
				if underscore < 0 || underscore+1 >= len(base) {
					continue
				}
				linkage := strings.ToLower(base[underscore+1:])
				a.assets[linkage] = sourceSoundAsset{Path: filepath.Join(soundsDir, name)}
			}
		}
	}

	// Developed CC0 sounds live with the portable custom assets. Scan both WAV
	// and MP3 files by filename so moving the whole game folder keeps them.
	if customSoundsDir, customErr := findOriginalPathIn("custom", "sounds"); customErr == nil {
		if entries, readErr := os.ReadDir(customSoundsDir); readErr == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				ext := strings.ToLower(filepath.Ext(name))
				if ext != ".wav" && ext != ".mp3" {
					continue
				}
				key := strings.ToLower(name)
				a.assets[key] = sourceSoundAsset{Path: filepath.Join(customSoundsDir, name)}
			}
		}
	}

	// Keep the original uncompressed 22.05 kHz stereo music in runtime assets.
	// The FFDec MP3 exports introduce audible metallic artifacts in this game,
	// so music111/333/444/555 deliberately override their compact MP3 entries.
	for _, music := range []string{"music111", "music333", "music444", "music555"} {
		wavPath := filepath.Join(soundsDir, music+".wav")
		if _, statErr := os.Stat(wavPath); statErr == nil {
			a.assets[music] = sourceSoundAsset{Path: wavPath}
		}
	}

	// Ebitengine/go-mp3 does not decode the five source 11.025 kHz MPEG-2.5
	// exports reliably. Keep only those tiny sounds as original XFL raw PCM.
	docPath, err := findOriginalPath("fla", "DOMDocument.xml")
	if err != nil {
		return
	}
	data, err := os.ReadFile(docPath)
	if err != nil {
		return
	}
	var doc struct {
		Media struct {
			Sounds []struct {
				LinkageIdentifier string `xml:"linkageIdentifier,attr"`
				SoundDataHRef     string `xml:"soundDataHRef,attr"`
				Format            string `xml:"format,attr"`
				SampleCount       int64  `xml:"sampleCount,attr"`
			} `xml:"DOMSoundItem"`
		} `xml:"media"`
	}
	if err := xml.Unmarshal(data, &doc); err != nil {
		return
	}
	binDir := filepath.Join(filepath.Dir(docPath), "bin")
	for _, item := range doc.Media.Sounds {
		if item.LinkageIdentifier == "" || item.SoundDataHRef == "" || !strings.HasPrefix(item.Format, "11kHz") {
			continue
		}
		channels := 1
		if strings.Contains(item.Format, "Stereo") {
			channels = 2
		}
		a.assets[strings.ToLower(item.LinkageIdentifier)] = sourceSoundAsset{
			Path:        filepath.Join(binDir, item.SoundDataHRef),
			RawPCM:      true,
			SampleRate:  11025,
			Channels:    channels,
			SampleCount: item.SampleCount,
		}
	}
}

func (a *sourceAudioEngine) soundPath(name string) string {
	if a == nil {
		return ""
	}
	a.ensureAssets()
	return a.assets[strings.ToLower(name)].Path
}

func stereoPCM16(raw []byte, channels int) []byte {
	if channels == 2 {
		return raw
	}
	if channels != 1 || len(raw) < 2 {
		return nil
	}
	out := make([]byte, (len(raw)/2)*4)
	j := 0
	for i := 0; i+1 < len(raw); i += 2 {
		out[j], out[j+1] = raw[i], raw[i+1]
		out[j+2], out[j+3] = raw[i], raw[i+1]
		j += 4
	}
	return out
}

func (a *sourceAudioEngine) decodedPCM(name string) []byte {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	key := strings.ToLower(name)
	if pcm, ok := a.pcm[key]; ok {
		return pcm
	}
	if pcm := developedSoundPCM(key); len(pcm) > 0 {
		a.pcm[key] = pcm
		return pcm
	}
	a.ensureAssets()
	asset, ok := a.assets[key]
	if !ok || asset.Path == "" {
		a.pcm[key] = nil
		return nil
	}
	raw, err := os.ReadFile(asset.Path)
	if err != nil {
		a.pcm[key] = nil
		return nil
	}
	ctx := a.context()
	if ctx == nil {
		return nil
	}
	var pcm []byte
	if asset.RawPCM {
		// The XFL .dat files used by these five 11.025 kHz sounds are already
		// raw PCM16 payloads. DOMDocument.xml's sampleCount is not the byte
		// length of those payloads (for example drop1.wav contains 5250 PCM
		// samples while sampleCount reports 3767), so trimming to sampleCount
		// corrupts the beginning of the sound. Use the complete .dat payload.
		pcm = stereoPCM16(raw, asset.Channels)
		if len(pcm) == 0 {
			a.pcm[key] = nil
			return nil
		}
		if asset.SampleRate != ctx.SampleRate() {
			resampled := audio.Resample(bytes.NewReader(pcm), int64(len(pcm)), asset.SampleRate, ctx.SampleRate())
			pcm, err = io.ReadAll(resampled)
			if err != nil {
				a.pcm[key] = nil
				return nil
			}
		}
	} else if strings.HasSuffix(strings.ToLower(asset.Path), ".wav") {
		stream, decodeErr := wav.DecodeWithoutResampling(bytes.NewReader(raw))
		if decodeErr != nil {
			a.pcm[key] = nil
			return nil
		}
		var decoded io.Reader = stream
		if stream.SampleRate() != ctx.SampleRate() {
			decoded = audio.Resample(stream, stream.Length(), stream.SampleRate(), ctx.SampleRate())
		}
		pcm, err = io.ReadAll(decoded)
		if err != nil || len(pcm) == 0 {
			a.pcm[key] = nil
			return nil
		}
	} else {
		stream, decodeErr := mp3.DecodeWithoutResampling(bytes.NewReader(raw))
		if decodeErr != nil {
			a.pcm[key] = nil
			return nil
		}
		var decoded io.Reader = stream
		if stream.SampleRate() != ctx.SampleRate() {
			decoded = audio.Resample(stream, stream.Length(), stream.SampleRate(), ctx.SampleRate())
		}
		pcm, err = io.ReadAll(decoded)
		if err != nil || len(pcm) == 0 {
			a.pcm[key] = nil
			return nil
		}
	}
	a.pcm[key] = pcm
	return pcm
}

func (a *sourceAudioEngine) playSFX(name string, volume float64) {
	a.playSFXGain(name, volume, 1)
}

func (a *sourceAudioEngine) playSFXGain(name string, volume, gain float64) {
	if a == nil || name == "" {
		return
	}
	pcm := a.amplifiedSoundPCM(name, gain)
	if len(pcm) == 0 {
		return
	}
	ctx := a.context()
	if ctx == nil {
		return
	}
	player := audio.NewPlayerFromBytes(ctx, pcm)
	player.SetVolume(volume)
	player.Play()
}

func (a *sourceAudioEngine) amplifiedSoundPCM(name string, gain float64) []byte {
	pcm := a.decodedPCM(name)
	if len(pcm) == 0 || gain <= 1 {
		return pcm
	}
	key := strings.ToLower(name) + "@" + strconv.FormatFloat(gain, 'f', 2, 64)
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.amplified == nil {
		a.amplified = make(map[string][]byte)
	}
	if cached, ok := a.amplified[key]; ok {
		return cached
	}
	boosted := amplifyPCM16(pcm, gain)
	a.amplified[key] = boosted
	return boosted
}

func amplifyPCM16(pcm []byte, gain float64) []byte {
	if len(pcm) == 0 || gain <= 1 {
		return pcm
	}
	out := append([]byte(nil), pcm...)
	for offset := 0; offset+1 < len(out); offset += 2 {
		value := float64(int16(binary.LittleEndian.Uint16(out[offset:]))) * gain
		if value > 32767 {
			value = 32767
		} else if value < -32768 {
			value = -32768
		}
		binary.LittleEndian.PutUint16(out[offset:], uint16(int16(value)))
	}
	return out
}

func (a *sourceAudioEngine) setMusicVolume(volume float64) {
	if a == nil || a.music == nil {
		return
	}
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}
	a.music.SetVolume(volume)
}

func (a *sourceAudioEngine) stopMusic() {
	if a == nil || a.music == nil {
		return
	}
	_ = a.music.Close()
	a.music = nil
	a.musicName = ""
}

func (a *sourceAudioEngine) playMusic(name string) {
	if a == nil || name == "" || a.musicName == name && a.music != nil && a.music.IsPlaying() {
		return
	}
	a.stopMusic()
	pcm := a.decodedPCM(name)
	if len(pcm) == 0 {
		return
	}
	ctx := a.context()
	if ctx == nil {
		return
	}
	loop := audio.NewInfiniteLoop(bytes.NewReader(pcm), int64(len(pcm)))
	player, err := audio.NewPlayer(ctx, loop)
	if err != nil {
		return
	}
	player.SetVolume(1)
	player.Play()
	a.music = player
	a.musicName = name
}

func sourceGunSound(number int) string {
	if number == 139 {
		// AS Val uses the source VSS Vintorez suppressed shot.
		return "silenced1.wav"
	}
	switch number {
	case 190, 206, 207, 216, 219, 231:
		return "dev_historical_pistol.wav"
	case 191, 205, 215, 227, 228:
		return "dev_historical_rifle.wav"
	case 192: // the 13.2 mm Tankgewehr needs the heaviest source rifle report
		return "snipe6.wav"
	case 193: // Martini-Henry black-powder crack and boom
		return "dev_musket.wav"
	case 194, 195, 196, 197, 202, 211, 212, 213, 214, 217, 222, 223, 230:
		return "dev_auto_rifle.wav"
	case 198, 199, 200, 201, 203, 220, 221:
		return "dev_historical_lmg.wav"
	case 204, 229:
		return "dev_historical_shotgun.wav"
	case 208, 209, 210, 218, 224, 225, 226:
		return "dev_sks.wav"
	}
	sounds := map[int]string{
		1: "pistol3.wav", 2: "pistol3.wav", 3: "pistol1.wav", 4: "pistol3.wav", 5: "pistol2.wav", 6: "pistol0.wav",
		8: "pistol3.wav", 9: "pistol0.wav", 10: "rifle6.wav", 11: "snipe1.wav", 12: "smg1.wav", 13: "shotgun1.wav",
		14: "snipe2.wav", 15: "shotgun3.wav", 16: "snipe3.wav", 17: "rifle2.wav", 18: "shotgun3.wav", 19: "silenced2.wav",
		20: "rifle6.wav", 21: "smg3.wav", 22: "smg4.wav", 23: "smg1.wav", 24: "smg2.wav", 25: "smg3.wav", 26: "smg4.wav",
		27: "smg1.wav", 28: "smg2.wav", 29: "smg3.wav", 30: "smg4.wav", 31: "smg1.wav", 32: "smg2.wav",
		33: "snipe4.wav", 34: "snipe5.wav", 35: "snipe6.wav", 36: "snipe1.wav", 37: "snipe2.wav", 38: "snipe3.wav",
		39: "snipe4.wav", 40: "silenced1.wav", 41: "snipe6.wav", 42: "snipe1.wav", 43: "snipe2.wav",
		44: "shotgun3.wav", 45: "shotgun2.wav", 46: "shotgun3.wav", 47: "shotgun3.wav", 48: "shotgun1.wav", 49: "shotgun2.wav",
		50: "shotgun1.wav", 51: "shotgun3.wav", 52: "shotgun3.wav", 53: "shotgun2.wav", 54: "shotgun3.wav",
		55: "rifle3.wav", 56: "rifle6.wav", 57: "rifle1.wav", 58: "silenced2.wav", 59: "rifle3.wav", 60: "rifle4.wav",
		61: "rifle5.wav", 62: "rifle1.wav", 63: "rifle2.wav", 64: "rifle3.wav", 65: "lmg.wav", 66: "rifle5.wav",
		67: "smg3.wav", 68: "smg4.wav", 69: "smg1.wav", 70: "silenced2.wav", 71: "smg3.wav",
		72: "rifle4.wav", 73: "rifle5.wav", 74: "silenced2.wav", 75: "rifle2.wav", 76: "rifle3.wav",
		77: "snipe6.wav", 78: "snipe1.wav", 79: "snipe2.wav", 80: "silenced1.wav", 81: "snipe4.wav",
		82: "shotgun3.wav", 83: "shotgun3.wav", 84: "shotgun1.wav", 85: "shotgun3.wav", 86: "shotgun3.wav",
	}
	if sound := sounds[number]; sound != "" {
		return sound
	}
	// Custom weapons reuse the original game's sound palette so every packaged
	// 87+ weapon has immediate audio without introducing mismatched samples.
	spec, ok := developedWeaponSpec(number)
	if !ok {
		return ""
	}
	switch spec.Category {
	case DevelopedCategoryPistol:
		return "dev_firearm.wav"
	case DevelopedCategorySMG:
		return "smg3.wav"
	case DevelopedCategoryRifle:
		return "dev_firearm.wav"
	case DevelopedCategoryLMG:
		return "lmg.wav"
	case DevelopedCategorySniper:
		return "snipe4.wav"
	case DevelopedCategoryShotgun:
		return "shotgun1.wav"
	case DevelopedCategoryLauncher:
		if spec.Projectile == DevelopedProjectileGrenade {
			return "dev_grenade.wav"
		}
		if spec.Projectile == DevelopedProjectileTopAttack {
			return "dev_javelin.wav"
		}
		return "dev_launcher_rpg.wav"
	case DevelopedCategorySpecial:
		if spec.Projectile == DevelopedProjectileFlame {
			return "dev_flame.wav"
		}
		if spec.Projectile == DevelopedProjectilePortal {
			return "dev_portal.wav"
		}
		switch number {
		case 177:
			return "dev_crossbow.wav"
		case 178:
			return "dev_bow.wav"
		case 179:
			return "dev_harpoon.wav"
		case 180:
			return "dev_nail.wav"
		case 182:
			return "dev_musket.wav"
		}
		switch spec.Projectile {
		case DevelopedProjectileFlare:
			return "dev_flare.wav"
		case DevelopedProjectileEnergy:
			return "dev_energy.wav"
		default:
			return "snipe1.wav"
		}
	}
	return "rifle1.wav"
}

func (g *Game) sourceSFXVolume(loud bool) float64 {
	if g == nil || !g.soundOn || g.soundVolume <= 0 {
		return 0
	}
	base := 0.5 // frame2.playsound()
	if loud {
		base = 1 // frame2.playsound2()
	}
	if g.loudnessEQ {
		// Developed loudness equalizer narrows the original 0.5/1.0 dynamic
		// range so explosions/deaths do not jump to twice the perceived level.
		if loud {
			base = 0.78
		} else {
			base = 0.62
		}
	}
	return base * g.soundVolume
}

func (g *Game) playSourceSFX(name string, loud bool) {
	g.playSourceSFXGain(name, loud, 1)
}

func (g *Game) playSourceSFXGain(name string, loud bool, gain float64) {
	if g == nil || g.audio == nil {
		return
	}
	volume := g.sourceSFXVolume(loud)
	if volume <= 0 {
		return
	}
	g.audio.playSFXGain(name, volume, gain)
	if g.netplay != nil && g.netplay.mode == netplayHost {
		g.netSFXSeq++
		g.netSFXPending = append(g.netSFXPending, netSFXEvent{Seq: g.netSFXSeq, Name: name, Loud: loud, Gain: gain})
	}
}

func (g *Game) playSourceGunSound(number int) {
	if name := sourceGunSound(number); name != "" {
		g.playSourceSFXGain(name, sourceGunSoundLoud(number), developedGunSoundGain(number))
	}
}

func developedGunSoundGain(number int) float64 {
	if number < 87 {
		return 1
	}
	if developedGunReusesOriginalSound(number) {
		// Exact original-bank mix: same decoded PCM and no extra amplification.
		return 1
	}
	if number == 182 {
		// The real black-powder recording is already normalized during packaging.
		return 1.28
	}
	// Small final lift after matching the developed bank against the original
	// rifles/shotguns in an actual match rather than against raw PCM peaks.
	return 1.85
}

func developedGunReusesOriginalSound(number int) bool {
	if number < 87 {
		return false
	}
	name := strings.ToLower(sourceGunSound(number))
	return name != "" && !strings.HasPrefix(name, "dev_")
}

func sourceGunSoundLoud(number int) bool {
	// Keep the original game's 1..86 mix untouched. The developed sounds were
	// previously sent through the half-volume playsound() path and were hard to
	// hear beside impacts and explosions.
	return number >= 87 && !developedGunReusesOriginalSound(number)
}

func (g *Game) playRandomDeathSound() {
	g.playSourceSFX([]string{"die1.wav", "die2.wav", "die3.wav", "die4.wav"}[rand.Intn(4)], true)
}

func (g *Game) playRandomHitSound() {
	g.playSourceSFX([]string{"hit1.wav", "hit2.wav"}[rand.Intn(2)], true)
}

func (g *Game) playRandomExplosionSound() {
	g.playSourceSFX([]string{"explosion1.wav", "explosion2.wav", "explosion3.wav", "explosion4.wav"}[rand.Intn(4)], true)
}

func (g *Game) playRandomDropSound() {
	// frame2.dropsound(): random drop1/drop2/drop3, used by player/AI only on
	// hard platform landing (abs(vy) > 3), not as a walking footstep loop.
	g.playSourceSFX([]string{"drop1.wav", "drop2.wav", "drop3.wav"}[rand.Intn(3)], false)
}

func (g *Game) syncSourceMusic() {
	if g == nil || g.audio == nil {
		return
	}
	if !g.musicOn || g.musicVolume <= 0 {
		g.audio.stopMusic()
		return
	}
	if g.screen == screenGameplay {
		if g.audio.musicName != "music333" && g.audio.musicName != "music444" && g.audio.musicName != "music555" {
			tracks := []string{"music333", "music444", "music555"}
			g.audio.playMusic(tracks[rand.Intn(len(tracks))])
		}
		g.audio.setMusicVolume(g.musicVolume)
		return
	}
	// Main/menu screens use the source menu loop. End-game transitions use
	// music222 in Flash, but the ordinary menu/demo loop is music111.
	g.audio.playMusic("music111")
	g.audio.setMusicVolume(g.musicVolume)
}
