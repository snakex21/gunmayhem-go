package game

import (
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

func advanceWeaponTimeline(p *Player, shells *[]Shell) {
	w := &p.Weapon
	if !w.Playing {
		return
	}
	timeline := weaponTimeline(w.Def.Number)
	if timeline.TotalFrames <= 0 {
		return
	}
	next := w.Frame + 1
	if next >= timeline.TotalFrames {
		next = 0
	}
	enterWeaponTimelineFrame(p, w, next, shells, 0)
}

func enterWeaponTimelineFrame(p *Player, w *WeaponState, frame int, shells *[]Shell, depth int) {
	if depth > 8 {
		return
	}
	timeline := weaponTimeline(w.Def.Number)
	if timeline.TotalFrames <= 0 {
		return
	}
	frame %= timeline.TotalFrames
	if frame < 0 {
		frame += timeline.TotalFrames
	}
	w.Frame = frame

	script := timeline.Scripts[frame]
	if script == "" {
		return
	}
	// Weapon movie clips own reload/bolt sounds on individual source frames.
	// Queue exactly those playsound() calls and let Game's audio channel honor
	// SOUND FX ON/OFF without coupling the timeline parser to Ebitengine audio.
	for _, match := range regexp.MustCompile(`_root\.playsound\("([^"]+)"\)`).FindAllStringSubmatch(script, -1) {
		if len(match) == 2 {
			p.PendingSounds = append(p.PendingSounds, match[1])
		}
	}

	// Frame actions below are the commands actually used by the exported gun
	// timelines. Their ordering mirrors ActionScript: assignments/events run on
	// the entered frame, then gotoAndPlay changes the playhead.
	if strings.Contains(script, "_parent._parent.ejectshell4();") {
		spawnWeaponShell(p, w, 4, shells)
	} else if strings.Contains(script, "_parent._parent.ejectshell3();") {
		spawnWeaponShell(p, w, 3, shells)
	} else if strings.Contains(script, "_parent._parent.ejectshell2();") {
		spawnWeaponShell(p, w, 2, shells)
	} else if strings.Contains(script, "_parent._parent.ejectshell();") {
		spawnWeaponShell(p, w, 1, shells)
	}
	if strings.Contains(script, "_parent._parent.ejectshot3();") {
		spawnWeaponShell(p, w, 6, shells)
	} else if strings.Contains(script, "_parent._parent.ejectshot();") {
		spawnWeaponShell(p, w, 5, shells)
	}
	// Reload-only CP() effects used by the six source handguns. These are not
	// weapon drops: the pistol stays in-hand while magazines/cylinder parts fall.
	if strings.Contains(script, "fx_dropmag") && !strings.Contains(script, "fx_dropmag2") && !strings.Contains(script, "fx_dropmag3") {
		spawnReloadParticle(p, 7, shells)
	}
	if strings.Contains(script, "fx_dropmag2") {
		spawnReloadParticle(p, 8, shells)
	}
	if strings.Contains(script, "fx_dropmag3") {
		spawnReloadParticle(p, 9, shells)
	}
	if strings.Contains(script, "fx_shell4") {
		spawnReloadParticle(p, 10, shells)
	}
	if strings.Contains(script, "fx_speedloader") {
		spawnReloadParticle(p, 11, shells)
	}
	if strings.Contains(script, "fx_deagle") {
		spawnReloadParticle(p, 12, shells)
	}

	if magazine, ok := sourceWeaponMagazineFromScript(script); ok {
		w.Bullets = magazine
	}
	if v, ok := sourceFrameNumeric(script, `_parent\._parent\.hand2\.hand\._alpha\s*=\s*([0-9.]+)`); ok {
		p.VisualHand2Alpha = v / 100
	}
	if v, ok := sourceFrameNumeric(script, `_parent\.hand\._alpha\s*=\s*([0-9.]+)`); ok {
		p.VisualHand1Alpha = v / 100
	}
	if v, ok := sourceFrameNumeric(script, `this\._alpha\s*=\s*([0-9.]+)`); ok {
		w.Alpha = v / 100
	}
	if v, ok := sourceFrameNumeric(script, `_parent\._parent\.idletime\s*=\s*([0-9.]+)`); ok {
		p.IdleTime = int(v)
	}
	if strings.Contains(script, "_parent._parent.adjustrof2();") && p.PerkNumber == 7 {
		w.Bullets += int(math.Ceil(float64(w.Bullets) * 0.333))
	}

	if strings.Contains(script, "stop();") {
		w.Playing = false
	}
	if target, ok := sourceWeaponGoto(script); ok {
		w.Playing = true
		enterWeaponTimelineFrame(p, w, target, shells, depth+1)
	}
}

func sourceFrameNumeric(script, pattern string) (float64, bool) {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(script)
	if len(m) != 2 {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func spawnReloadParticle(p *Player, kind int, shells *[]Shell) {
	facing := p.Facing
	if facing == 0 {
		facing = 1
	}
	randRot := func() float64 {
		v := rand.Float64()*10 + 5
		if rand.Intn(2) == 0 {
			v = -v
		}
		return v
	}
	s := Shell{Kind: kind, ScaleX: 1}
	switch kind {
	case 7, 8, 9: // fx_dropmag / fx_dropmag2 / fx_dropmag3
		s.X = p.X + (p.VisualHand1ChildX-5)*float64(facing)
		s.Y = p.Y - 20
		s.VX = 0
		s.VY = 0
		s.RotSpeed = 5
		if rand.Intn(2) == 0 {
			s.RotSpeed = -s.RotSpeed
		}
	case 10: // direct CP("fx_shell4", ...) from revolver reloads
		s.X = p.X + (p.VisualHand1ChildX+4)*float64(facing)
		s.Y = p.Y - 38
		s.Rotation = -60 * float64(facing)
		s.VX = rand.Float64()*2 - 1
		s.VY = rand.Float64() * 5
		s.RotSpeed = randRot()
	case 11: // fx_speedloader
		s.X = p.X + (p.VisualHand1ChildX+4)*float64(facing)
		s.Y = p.Y - 38
		s.Rotation = -60 * float64(facing)
		s.VX = rand.Float64()*2 - 1
		s.VY = 0 // source assigns random*5, then immediately overwrites vy=0
		s.RotSpeed = randRot()
	case 12: // fx_deagle
		s.X = p.X + (p.VisualHand1ChildX+30)*float64(facing)
		s.Y = p.Y - 55
		s.Rotation = -60 * float64(facing)
		s.VX = 20 * float64(facing)
		s.VY = -6
		s.RotSpeed = 15 * float64(facing)
		s.ScaleX = float64(facing)
		s.OwnerID = p.scoreOwnerID()
		s.Facing = facing
	default:
		return
	}
	*shells = append(*shells, s)
}

func spawnWeaponShell(p *Player, w *WeaponState, kind int, shells *[]Shell) {
	mini := p.MiniMulti
	if mini <= 0 {
		mini = 1
	}
	yOffset := 38.0
	if kind == 3 || kind == 6 {
		yOffset = 27
	}
	rot := rand.Float64()*10 + 5
	if rand.Intn(2) == 0 {
		rot = -rot
	}
	*shells = append(*shells, Shell{
		X:        p.X + w.Def.ShellX*mini*float64(p.Facing),
		Y:        p.Y - yOffset*mini,
		VX:       rand.Float64()*10 - 5,
		VY:       rand.Float64()*5 - 10,
		RotSpeed: rot,
		ScaleX:   1,
		Kind:     kind,
	})
}
