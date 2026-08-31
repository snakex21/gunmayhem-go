package game

import (
	"encoding/binary"
	"math"
)

// developedSoundPCM creates short, portable PCM effects in memory. They do not
// write cache files or depend on the original Flash sound palette, and are
// cached by sourceAudioEngine after their first use.
func developedSoundPCM(name string) []byte {
	type soundSpec struct {
		duration float64
		sample   func(float64, int) float64
	}
	noise := func(i int) float64 {
		x := uint32(i)*1664525 + 1013904223
		return float64(int32(x>>16)-32768) / 32768
	}
	decay := func(t, duration, strength float64) float64 {
		return math.Exp(-strength * t / duration)
	}
	chirp := func(t, startHz, endHz, duration float64) float64 {
		sweep := (endHz - startHz) / duration
		return math.Sin(2 * math.Pi * (startHz*t + .5*sweep*t*t))
	}

	var spec soundSpec
	switch name {
	case "dev_energy.wav":
		spec = soundSpec{.24, func(t float64, i int) float64 {
			env := decay(t, .24, 5.2)
			return env * (.64*chirp(t, 1180, 240, .24) + .23*chirp(t, 2360, 520, .24) + .08*noise(i))
		}}
	case "dev_bow.wav":
		spec = soundSpec{.28, func(t float64, i int) float64 {
			env := decay(t, .28, 7)
			snap := 0.0
			if t < .018 {
				snap = noise(i) * (1 - t/.018)
			}
			return .58*env*math.Sin(2*math.Pi*164*t) + .22*env*math.Sin(2*math.Pi*328*t) + .25*snap
		}}
	case "dev_crossbow.wav":
		spec = soundSpec{.16, func(t float64, i int) float64 {
			env := decay(t, .16, 9)
			return env * (.48*math.Sin(2*math.Pi*265*t) + .38*noise(i))
		}}
	case "dev_launcher.wav":
		spec = soundSpec{.34, func(t float64, i int) float64 {
			env := decay(t, .34, 5)
			return env * (.52*chirp(t, 190, 58, .34) + .34*noise(i) + .12*math.Sin(2*math.Pi*52*t))
		}}
	case "dev_grenade.wav":
		spec = soundSpec{.22, func(t float64, i int) float64 {
			env := decay(t, .22, 7)
			return env * (.58*chirp(t, 310, 92, .22) + .24*noise(i))
		}}
	case "dev_harpoon.wav":
		spec = soundSpec{.20, func(t float64, i int) float64 {
			env := decay(t, .20, 7)
			return env * (.58*math.Sin(2*math.Pi*118*t) + .30*noise(i))
		}}
	case "dev_nail.wav":
		spec = soundSpec{.075, func(t float64, i int) float64 {
			env := decay(t, .075, 12)
			return env * (.50*chirp(t, 1500, 420, .075) + .42*noise(i))
		}}
	case "dev_flare.wav":
		spec = soundSpec{.18, func(t float64, i int) float64 {
			env := decay(t, .18, 8)
			return env * (.46*chirp(t, 480, 130, .18) + .24*noise(i))
		}}
	case "dev_flame.wav":
		spec = soundSpec{.105, func(t float64, i int) float64 {
			env := decay(t, .105, 2.4)
			return env * (.42*noise(i) + .28*math.Sin(2*math.Pi*76*t) + .13*math.Sin(2*math.Pi*151*t))
		}}
	case "dev_portal.wav":
		spec = soundSpec{.31, func(t float64, i int) float64 {
			env := decay(t, .31, 4.2)
			return env * (.61*chirp(t, 310, 1380, .31) + .22*math.Sin(2*math.Pi*92*t) + .05*noise(i))
		}}
	default:
		return nil
	}

	frames := int(spec.duration * sourceAudioSampleRate)
	pcm := make([]byte, frames*4) // stereo PCM16
	for i := 0; i < frames; i++ {
		t := float64(i) / sourceAudioSampleRate
		value := math.Max(-1, math.Min(1, spec.sample(t, i)))
		sample := int16(value * 24500)
		binary.LittleEndian.PutUint16(pcm[i*4:i*4+2], uint16(sample))
		binary.LittleEndian.PutUint16(pcm[i*4+2:i*4+4], uint16(sample))
	}
	return pcm
}
