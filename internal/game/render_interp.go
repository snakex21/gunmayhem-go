package game

import (
	"math"
	"time"
)

// render_interp.go: render interpolation between simulation ticks.
//
// The game logic runs at a fixed TicksPerSecond (30). Ebitengine draws at the
// display refresh (60/120/144 Hz), so without interpolation every tick is
// shown 2-5 times and motion looks choppy. Here Draw shows moving objects
// between the previous and the current tick, alpha (0..1) of the way, where
// alpha is the fraction of a tick elapsed since the last Update.
//
// Rules:
//   - Logic, hit detection, AI and the netplay snapshot never see these
//     positions. Draw works on interpolated COPIES of the entity lists that are
//     swapped in for the duration of Draw and swapped back afterwards.
//   - Only positions (and rotations of spinning objects) are interpolated.
//     Sprite animation frames, alphas, scales and timers stay tick-based:
//     blending discrete animation frames would mix two poses.
//   - Objects that did not exist on the previous tick, teleported (portal,
//     respawn, wrap), or changed map are drawn at their current position
//     (snap) instead of being slid across the screen.
//   - The same code runs for local matches and for online guests (there the
//     "ticks" are the playout states applied by the client each update).

// TicksPerSecond is the fixed simulation rate (ebiten.SetTPS in main.go).
const TicksPerSecond = 30

const (
	// A player whose new position is further than this from where its
	// previous velocity would carry it teleported (portal, push-out, wrap).
	renderSnapPlayerPx = 40.0
	// Hard limit for any single-tick move that is still interpolated.
	renderMaxPlayerStepPx = 300.0
	// List objects (bullets, shells, grenades, debris): tolerance on top of
	// their own speed for bounces and drag.
	renderSnapObjectPx = 24.0
	// Camera jumps larger than this (map change, round reset) snap.
	renderSnapCameraPx = 150.0
	// Rotation changes larger than this (degrees per tick) snap.
	renderSnapRotationDeg = 120.0
	// How many previous objects may be skipped (died this tick) when
	// matching an object to its previous position.
	renderMatchLookahead = 8
)

// renderSample is the interpolation-relevant state of one object at a tick.
type renderSample struct {
	x, y, vx, vy, rot float64
	key               int  // player ID / enemy ID, or the object kind
	serial            int  // death serial (players)
	double            bool // players: helper double
	active            bool
}

// renderListScratch is reused by every interpolated list in a frame.
type renderListScratch struct {
	cur   []renderSample
	match []int
}

// renderInterpState is the previous-tick capture plus Draw scratch buffers.
type renderInterpState struct {
	valid      bool
	arena      int
	screen     gameScreen
	camX, camY float64
	lastUpdate time.Time

	players, enemies                  []renderSample
	bullets, shells, grenades, blasts []renderSample
	jets, drops, dynamite, crates     []renderSample
	scratch                           renderListScratch
	drawPlayers                       []Player
	drawPlayerPtrs                    []*Player
	drawEnemies                       []CampaignEnemy
	drawEnemyPtrs                     []*CampaignEnemy
	drawBullets                       []Bullet
	drawShells                        []Shell
	drawGrenades                      []Grenade
	drawBlasts                        []BlastEffect
	drawJets                          []JetThrustEffect
	drawDrops                         []DropPackEffect
	drawDynamite                      []DynamiteEffect
	drawCrates                        []Crate
	lastAlpha                         float64
}

func samplePlayer(p *Player) renderSample {
	return renderSample{x: p.X, y: p.Y, vx: p.VX, vy: p.VY, key: p.ID, serial: p.DeathSerial, double: p.IsDouble, active: p.Active}
}

func sampleBullet(b *Bullet) renderSample {
	return renderSample{x: b.X, y: b.Y, vx: b.VX, vy: b.VY, rot: b.Rotation, key: int(b.Kind), active: !b.Dead}
}

func sampleShell(s *Shell) renderSample {
	return renderSample{x: s.X, y: s.Y, vx: s.VX, vy: s.VY, rot: s.Rotation, key: s.Kind, active: !s.Dead}
}

func sampleGrenade(g *Grenade) renderSample {
	return renderSample{x: g.X, y: g.Y, vx: g.VX, vy: g.VY, rot: g.Rotation, active: !g.Dead}
}

func sampleBlast(b *BlastEffect) renderSample {
	return renderSample{x: b.X, y: b.Y, vx: b.VX, vy: b.VY, rot: b.Rotation, key: int(b.Kind), active: !b.Dead}
}

func sampleJet(j *JetThrustEffect) renderSample {
	return renderSample{x: j.X, y: j.Y, vx: j.VX, vy: j.VY, rot: j.Rotation, active: !j.Dead}
}

func sampleDrop(d *DropPackEffect) renderSample {
	return renderSample{x: d.X, y: d.Y, vx: d.VX, vy: d.VY, rot: d.Rotation, active: !d.Dead}
}

func sampleDynamite(d *DynamiteEffect) renderSample {
	return renderSample{x: d.X, y: d.Y, vx: d.VX, vy: d.VY, rot: d.Rotation, active: !d.Dead}
}

func sampleCrate(c *Crate) renderSample {
	return renderSample{x: c.X, y: c.Y, vy: c.VY, key: c.Serial, active: !c.Dead}
}

func captureRenderSamples[T any](dst []renderSample, list []T, sample func(*T) renderSample) []renderSample {
	dst = dst[:0]
	for i := range list {
		dst = append(dst, sample(&list[i]))
	}
	return dst
}

// captureRenderPrev records the state at the start of a tick (top of
// Update). Draw interpolates from it to the state the tick produces.
func (g *Game) captureRenderPrev(now time.Time) {
	r := &g.renderInterp
	r.lastUpdate = now
	r.screen = g.screen
	r.valid = g.screen == screenGameplay
	if !r.valid {
		return
	}
	r.arena = g.arena.Number
	r.camX, r.camY = g.cameraX, g.cameraY
	r.players = r.players[:0]
	for _, p := range g.players {
		if p != nil {
			r.players = append(r.players, samplePlayer(p))
		}
	}
	r.enemies = r.enemies[:0]
	for _, e := range g.campaignEnemies {
		if e != nil {
			s := samplePlayer(&e.Body)
			s.active = !e.Dead
			r.enemies = append(r.enemies, s)
		}
	}
	r.bullets = captureRenderSamples(r.bullets, g.bullets, sampleBullet)
	r.shells = captureRenderSamples(r.shells, g.shells, sampleShell)
	r.grenades = captureRenderSamples(r.grenades, g.grenades, sampleGrenade)
	r.blasts = captureRenderSamples(r.blasts, g.blastFX, sampleBlast)
	r.jets = captureRenderSamples(r.jets, g.jetThrustFX, sampleJet)
	r.drops = captureRenderSamples(r.drops, g.dropPackFX, sampleDrop)
	r.dynamite = captureRenderSamples(r.dynamite, g.dynamiteFX, sampleDynamite)
	r.crates = captureRenderSamples(r.crates, g.crates, sampleCrate)
}

// renderAlphaAt is the fraction of a tick elapsed since the last update,
// clamped to [0,1]. 1 means "draw the current tick" (no interpolation).
func renderAlphaAt(lastUpdate, now time.Time) float64 {
	if lastUpdate.IsZero() {
		return 1
	}
	tick := time.Second / TicksPerSecond
	a := float64(now.Sub(lastUpdate)) / float64(tick)
	if a < 0 {
		return 0
	}
	if a > 1 {
		return 1
	}
	return a
}

// renderInterpolationEnabled: SMOOTH MOTION on, and POWER SAVE off (power
// save draws once per tick, where interpolation would only add latency).
func (g *Game) renderInterpolationEnabled() bool {
	return g != nil && g.displaySmoothMotion && !g.displayPowerSave
}

func lerpRender(a, b, t float64) float64 { return a + (b-a)*t }

// lerpAngleDeg interpolates rotations in degrees along the shorter arc and
// snaps (returns b) on large changes.
func lerpAngleDeg(a, b, t float64) float64 {
	d := math.Mod(b-a, 360)
	if d > 180 {
		d -= 360
	} else if d < -180 {
		d += 360
	}
	if math.Abs(d) > renderSnapRotationDeg {
		return b
	}
	return b - d*(1-t)
}

// renderObjectError is how far cur is from where prev's velocity would put
// it after one tick, or -1 when cur cannot be prev advanced by one tick.
func renderObjectError(prev, cur renderSample) float64 {
	if prev.key != cur.key || !prev.active {
		return -1
	}
	speed := math.Hypot(prev.vx, prev.vy)
	err := math.Hypot(cur.x-(prev.x+prev.vx), cur.y-(prev.y+prev.vy))
	if err > renderSnapObjectPx+speed {
		return -1
	}
	return err
}

// renderPlayerContinues reports whether a player can be interpolated from
// prev to cur (same life, no teleport).
func renderPlayerContinues(prev, cur renderSample) bool {
	if !prev.active || !cur.active || prev.key != cur.key || prev.double != cur.double || prev.serial != cur.serial {
		return false
	}
	jx, jy := cur.x-prev.x, cur.y-prev.y
	if jx*jx+jy*jy > renderMaxPlayerStepPx*renderMaxPlayerStepPx {
		return false
	}
	ex, ey := cur.x-(prev.x+prev.vx), cur.y-(prev.y+prev.vy)
	tol := renderSnapPlayerPx + 0.5*math.Hypot(prev.vx, prev.vy)
	return ex*ex+ey*ey <= tol*tol
}

// matchRenderSamples maps every current list object to the previous-tick
// object it continues (-1: new or teleported). Lists only lose objects
// (compaction keeps order) and gain new ones at the end, so a forward scan
// with a short lookahead for objects that died this tick is enough.
func matchRenderSamples(prev, cur []renderSample, out []int) []int {
	out = out[:0]
	i := 0
	for _, c := range cur {
		// Closest plausible candidate: shotgun pellets and debris travel in
		// tight groups, so the first plausible one is not always the right one.
		m, best := -1, math.MaxFloat64
		for k := i; k < len(prev) && k < i+renderMatchLookahead; k++ {
			if e := renderObjectError(prev[k], c); e >= 0 && e < best {
				m, best = k, e
				if e < 0.5 {
					break
				}
			}
		}
		if m >= 0 {
			i = m + 1
		}
		out = append(out, m)
	}
	return out
}

// interpolateRenderList returns copies of list with positions (and, when
// rotates, rotations) interpolated from the matched previous samples.
func interpolateRenderList[T any](dst []T, list []T, prev []renderSample, sample func(*T) renderSample, set func(*T, float64, float64, float64), alpha float64, rotates bool, scratch *renderListScratch) []T {
	dst = append(dst[:0], list...)
	if len(list) == 0 || len(prev) == 0 || alpha >= 1 {
		return dst
	}
	scratch.cur = captureRenderSamples(scratch.cur, list, sample)
	cur := scratch.cur
	scratch.match = matchRenderSamples(prev, cur, scratch.match)
	match := &scratch.match
	for i, m := range *match {
		if m < 0 {
			continue
		}
		p, c := prev[m], cur[i]
		rot := c.rot
		if rotates {
			rot = lerpAngleDeg(p.rot, c.rot, alpha)
		}
		set(&dst[i], lerpRender(p.x, c.x, alpha), lerpRender(p.y, c.y, alpha), rot)
	}
	return dst
}

// beginRenderInterpolation swaps interpolated copies of the moving entity
// lists into the Game for Draw. The returned function restores the tick
// state and must be called before Draw returns.
func (g *Game) beginRenderInterpolation(now time.Time) func() {
	r := &g.renderInterp
	alpha := 1.0
	if g.renderInterpolationEnabled() && r.valid && r.screen == screenGameplay && g.screen == screenGameplay && r.arena == g.arena.Number {
		alpha = renderAlphaAt(r.lastUpdate, now)
	}
	r.lastAlpha = alpha
	if alpha >= 1 {
		return func() {}
	}

	players, enemies := g.players, g.campaignEnemies
	bullets, shells, grenades := g.bullets, g.shells, g.grenades
	blasts, jets, drops, dynamite, crates := g.blastFX, g.jetThrustFX, g.dropPackFX, g.dynamiteFX, g.crates
	camX, camY := g.cameraX, g.cameraY

	// Players: matched by slot (ID + double), never across a death.
	r.drawPlayers = r.drawPlayers[:0]
	for _, p := range players {
		if p != nil {
			r.drawPlayers = append(r.drawPlayers, *p)
		}
	}
	r.drawPlayerPtrs = r.drawPlayerPtrs[:0]
	for i := range r.drawPlayers {
		p := &r.drawPlayers[i]
		cur := samplePlayer(p)
		for _, prev := range r.players {
			if prev.key == cur.key && prev.double == cur.double {
				if renderPlayerContinues(prev, cur) {
					p.X = lerpRender(prev.x, cur.x, alpha)
					p.Y = lerpRender(prev.y, cur.y, alpha)
				}
				break
			}
		}
		r.drawPlayerPtrs = append(r.drawPlayerPtrs, p)
	}
	g.players = r.drawPlayerPtrs

	// Campaign enemies (drones, shield bearers): matched by body ID.
	if len(enemies) > 0 {
		r.drawEnemies = r.drawEnemies[:0]
		for _, e := range enemies {
			if e != nil {
				r.drawEnemies = append(r.drawEnemies, *e)
			}
		}
		r.drawEnemyPtrs = r.drawEnemyPtrs[:0]
		for i := range r.drawEnemies {
			e := &r.drawEnemies[i]
			cur := samplePlayer(&e.Body)
			cur.active = !e.Dead
			for _, prev := range r.enemies {
				if prev.key == cur.key {
					if renderPlayerContinues(prev, cur) {
						e.Body.X = lerpRender(prev.x, cur.x, alpha)
						e.Body.Y = lerpRender(prev.y, cur.y, alpha)
					}
					break
				}
			}
			r.drawEnemyPtrs = append(r.drawEnemyPtrs, e)
		}
		g.campaignEnemies = r.drawEnemyPtrs
	}

	r.drawBullets = interpolateRenderList(r.drawBullets, bullets, r.bullets, sampleBullet,
		func(b *Bullet, x, y, rot float64) { b.X, b.Y, b.Rotation = x, y, rot }, alpha, false, &r.scratch)
	g.bullets = r.drawBullets
	r.drawShells = interpolateRenderList(r.drawShells, shells, r.shells, sampleShell,
		func(s *Shell, x, y, rot float64) { s.X, s.Y, s.Rotation = x, y, rot }, alpha, true, &r.scratch)
	g.shells = r.drawShells
	r.drawGrenades = interpolateRenderList(r.drawGrenades, grenades, r.grenades, sampleGrenade,
		func(gr *Grenade, x, y, rot float64) { gr.X, gr.Y, gr.Rotation = x, y, rot }, alpha, true, &r.scratch)
	g.grenades = r.drawGrenades
	r.drawBlasts = interpolateRenderList(r.drawBlasts, blasts, r.blasts, sampleBlast,
		func(b *BlastEffect, x, y, rot float64) { b.X, b.Y, b.Rotation = x, y, rot }, alpha, true, &r.scratch)
	g.blastFX = r.drawBlasts
	r.drawJets = interpolateRenderList(r.drawJets, jets, r.jets, sampleJet,
		func(j *JetThrustEffect, x, y, rot float64) { j.X, j.Y, j.Rotation = x, y, rot }, alpha, true, &r.scratch)
	g.jetThrustFX = r.drawJets
	r.drawDrops = interpolateRenderList(r.drawDrops, drops, r.drops, sampleDrop,
		func(d *DropPackEffect, x, y, rot float64) { d.X, d.Y, d.Rotation = x, y, rot }, alpha, true, &r.scratch)
	g.dropPackFX = r.drawDrops
	r.drawDynamite = interpolateRenderList(r.drawDynamite, dynamite, r.dynamite, sampleDynamite,
		func(d *DynamiteEffect, x, y, rot float64) { d.X, d.Y, d.Rotation = x, y, rot }, alpha, true, &r.scratch)
	g.dynamiteFX = r.drawDynamite
	r.drawCrates = interpolateRenderList(r.drawCrates, crates, r.crates, sampleCrate,
		func(c *Crate, x, y, _ float64) { c.X, c.Y = x, y }, alpha, false, &r.scratch)
	g.crates = r.drawCrates

	if math.Abs(camX-r.camX) <= renderSnapCameraPx && math.Abs(camY-r.camY) <= renderSnapCameraPx {
		g.cameraX = lerpRender(r.camX, camX, alpha)
		g.cameraY = lerpRender(r.camY, camY, alpha)
	}

	return func() {
		g.players, g.campaignEnemies = players, enemies
		g.bullets, g.shells, g.grenades = bullets, shells, grenades
		g.blastFX, g.jetThrustFX, g.dropPackFX, g.dynamiteFX, g.crates = blasts, jets, drops, dynamite, crates
		g.cameraX, g.cameraY = camX, camY
	}
}

// ---------------------------------------------------------------------------
// Frame rate

const (
	displayFrameRateVSync = iota
	displayFrameRate60
	displayFrameRate120
	displayFrameRate144
	displayFrameRateUnlimited
	displayFrameRateCount
)

var displayFrameRateLabels = []string{"VSYNC", "60", "120", "144", "UNLIMITED"}

// displayFrameRateCap returns the frame cap in frames per second (0: none).
func displayFrameRateCap(mode int) int {
	switch mode {
	case displayFrameRate60:
		return 60
	case displayFrameRate120:
		return 120
	case displayFrameRate144:
		return 144
	}
	return 0
}

// displayFrameRateVSyncOn: only VSYNC syncs to the monitor; the numeric caps
// and UNLIMITED turn vsync off (the caps are enforced by frameLimiter).
func displayFrameRateVSyncOn(mode int) bool { return mode == displayFrameRateVSync }

// frameLimiter paces Draw to a fixed rate when vsync is off.
type frameLimiter struct {
	next time.Time
}

// delay returns how long to wait before drawing a frame at now, and books
// the following frame slot. A limiter that fell more than a frame behind
// (a hitch, a window drag) restarts from now instead of racing to catch up.
func (l *frameLimiter) delay(now time.Time, fps int) time.Duration {
	if fps <= 0 {
		l.next = time.Time{}
		return 0
	}
	interval := time.Second / time.Duration(fps)
	if l.next.IsZero() || now.Sub(l.next) > interval {
		l.next = now.Add(interval)
		return 0
	}
	wait := l.next.Sub(now)
	if wait < 0 {
		wait = 0
	}
	l.next = l.next.Add(interval)
	return wait
}
