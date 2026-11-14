package game

import (
	"math"
	"math/rand"
)

// Utility throwables (gas / flashbang / smoke), the blind status and the
// goggles perks. Gameplay here is host-authoritative and identical in every
// mode; only presentation (throwable_effects_visuals.go) depends on who is
// looking at the screen.
//
// Online sync: AreaEffect lives in netSnapshot.AreaEffects and the blind
// fields are ordinary Player fields, which the schema-driven snapshot codec
// sends automatically. AI-only bookkeeping uses the AI* prefix, which the
// codec keeps host-side.

// Goggles perks. 10-13 are reserved for the not yet ported GM2 perks.
const (
	perkFlashGoggles   = 14
	perkThermalGoggles = 15
	perkGasMask        = 16
	// perkKevlar: body armour. No existing perk (1-9) touches bullet
	// knockback: BLAST SHIELD (6) only covers explosions, NO RECOIL (5) only
	// the shooter's own recoil.
	perkKevlar = 17
	// onlinePerkTableSize bounds per-perk tables such as the online perk
	// blocklist (index = perk number).
	onlinePerkTableSize = 18

	kevlarBulletKnockbackScale = 0.75 // -25% knockback from non-explosive hits
	kevlarMoveScale            = 0.90 // -10% run speed
	kevlarJumpScale            = 0.92 // -8% jump
)

// kevlarBulletKnockbackMultiplier applies to bullets, pellets, bolts and
// instagib beams only (never explosions, fire, gas or flash).
func kevlarBulletKnockbackMultiplier(p *Player) float64 {
	if p != nil && p.PerkNumber == perkKevlar {
		return kevlarBulletKnockbackScale
	}
	return 1
}

func kevlarMoveMultiplier(p *Player) float64 {
	if p != nil && p.PerkNumber == perkKevlar {
		return kevlarMoveScale
	}
	return 1
}

func kevlarJumpMultiplier(p *Player) float64 {
	if p != nil && p.PerkNumber == perkKevlar {
		return kevlarJumpScale
	}
	return 1
}

// selectablePerkNumbers lists every perk a player may pick, in menu order.
func selectablePerkNumbers() []int {
	return []int{1, 2, 3, 4, 5, 6, 7, 8, 9, perkFlashGoggles, perkThermalGoggles, perkGasMask, perkKevlar}
}

func isSelectablePerk(n int) bool {
	for _, perk := range selectablePerkNumbers() {
		if perk == n {
			return true
		}
	}
	return false
}

// stepSelectablePerk moves through selectablePerkNumbers with wrap-around.
func stepSelectablePerk(current, delta int) int {
	perks := selectablePerkNumbers()
	index := 0
	for i, perk := range perks {
		if perk == current {
			index = i
			break
		}
	}
	index = ((index+delta)%len(perks) + len(perks)) % len(perks)
	return perks[index]
}

// playerHeadgearPerk returns the head-worn perk (flash goggles, thermal
// goggles or gas mask) the player has, or 0. A player has exactly one perk
// slot, so at most one of them can be active; should that ever change, the
// render priority is gas mask > thermal goggles > flash goggles (the first
// match of headgearPerkPriority).
func playerHeadgearPerk(p *Player) int {
	if p == nil {
		return 0
	}
	for _, perk := range headgearPerkPriority {
		if p.PerkNumber == perk {
			return perk
		}
	}
	return 0
}

var headgearPerkPriority = [...]int{perkGasMask, perkThermalGoggles, perkFlashGoggles}

const (
	areaEffectGas   = 1
	areaEffectSmoke = 2
	areaEffectFlash = 3 // short flashbang burst, visual only
	areaEffectTear  = 4 // tear gas cloud
)

// AreaEffect is a lingering throwable effect: a gas cloud that pushes players,
// a smoke cloud that blocks sight, or the brief flashbang burst.
type AreaEffect struct {
	Kind      int     `cbor:",omitempty"`
	X, Y      float64 `cbor:",omitempty"`
	Radius    float64 `cbor:",omitempty"`
	MaxRadius float64 `cbor:",omitempty"`
	OwnerID   int     `cbor:",omitempty"`
	Life      int     `cbor:",omitempty"`
	MaxLife   int     `cbor:",omitempty"`
	Frame     int     `cbor:",omitempty"`
	Dead      bool    `cbor:",omitempty"`
}

const (
	gasCloudRadius      = 150.0
	gasCloudFrames      = 150 // 2.5 s
	gasGrowFrames       = 10
	gasPushStrength     = 0.85 // per-frame push at the centre, falls to 0 at the edge
	gasVerticalFactor   = 0.5  // keep the push mostly sideways (no launching)
	gasBlastShieldScale = 0.5  // BLAST SHIELD / shield powerup
	gasOwnerScale       = 0.5  // the thrower is pushed less by their own gas
	gasFadeFrames       = 30

	flashbangRadius         = 300.0
	flashFullBlindMinFrames = 120 // 2 s at the edge of the radius
	flashFullBlindMaxFrames = 180 // 3 s point blank
	flashAwayBlindFrames    = 30  // 0.5 s when facing away
	flashAwayStrength       = 0.35
	flashFullStrengthMin    = 0.6
	// flashLowerLevelThreshold: feet more than this far below the flash mean
	// the player stands on a lower level and is not affected.
	flashLowerLevelThreshold = 85.0
	flashPointLift           = 12.0 // the flash pops slightly above the canister
	flashBurstFrames         = 24
	flashAIWarningFrames     = 12
	blindFadeFrames          = 30
	blindSpreadMaxDegrees    = 16.0

	smokeCloudRadius   = 170.0
	smokeCloudFrames   = 450 // 7.5 s
	smokeStartRadius   = 40.0
	smokeGrowFrames    = 45
	smokeFadeFrames    = 70
	smokeCoreFactor    = 0.85 // fraction of the radius that actually hides
	smokeMinDensity    = 0.4
	smokeSpreadDegrees = 6.0
	smokeShotReach     = 450.0
	aiSmokeCloseRange  = 70.0

	tearGasCloudRadius  = 140.0
	tearGasCloudFrames  = 270 // 4.5 s
	tearGasStartRadius  = 35.0
	tearGasGrowFrames   = 40
	tearGasFadeFrames   = 60
	tearGasLingerFrames = 60   // coughing continues ~1 s after leaving
	tearGasMoveScale    = 0.65 // -35% movement
	tearGasJumpScale    = 0.75 // -25% jump
	tearGasSpreadDeg    = 9.0
)

func spawnTearGasCloud(g *Grenade, areas *[]AreaEffect) {
	if g == nil || areas == nil {
		return
	}
	*areas = append(*areas, AreaEffect{
		Kind: areaEffectTear, X: g.X, Y: g.Y - 34,
		Radius: tearGasStartRadius, MaxRadius: tearGasCloudRadius,
		OwnerID: g.OwnerID, Life: tearGasCloudFrames, MaxLife: tearGasCloudFrames,
	})
}

// tearGasDensity is 0..1 like smokeDensity.
func tearGasDensity(a *AreaEffect) float64 {
	if a == nil || a.Kind != areaEffectTear || a.Dead || a.Life <= 0 {
		return 0
	}
	d := math.Min(1, float64(a.Frame)/10)
	if a.Life < tearGasFadeFrames {
		d = math.Min(d, float64(a.Life)/tearGasFadeFrames)
	}
	return d
}

// applyTearGas makes every player inside the cloud cough. GAS MASK is
// immune. Teams and the thrower are affected alike (it is a cloud).
func applyTearGas(a *AreaEffect, players []*Player) {
	if tearGasDensity(a) < 0.25 {
		return
	}
	for _, p := range players {
		if p == nil || !p.Active || p.PerkNumber == perkGasMask {
			continue
		}
		if math.Hypot(p.X-a.X, p.Y-30-a.Y) <= a.Radius*0.9 && p.TearGasTime < tearGasLingerFrames {
			p.TearGasTime = tearGasLingerFrames
		}
	}
}

// tearGasMoveMultiplier / tearGasJumpMultiplier are 1 unless coughing.
func tearGasMoveMultiplier(p *Player) float64 {
	if p != nil && p.TearGasTime > 0 {
		return tearGasMoveScale
	}
	return 1
}

func tearGasJumpMultiplier(p *Player) float64 {
	if p != nil && p.TearGasTime > 0 {
		return tearGasJumpScale
	}
	return 1
}

func spawnGasCloud(g *Grenade, areas *[]AreaEffect) {
	if g == nil || areas == nil {
		return
	}
	*areas = append(*areas, AreaEffect{
		Kind: areaEffectGas, X: g.X, Y: g.Y - 22,
		Radius: gasCloudRadius * 0.3, MaxRadius: gasCloudRadius,
		OwnerID: g.OwnerID, Life: gasCloudFrames, MaxLife: gasCloudFrames,
	})
}

func spawnSmokeCloud(g *Grenade, areas *[]AreaEffect) {
	if g == nil || areas == nil {
		return
	}
	*areas = append(*areas, AreaEffect{
		Kind: areaEffectSmoke, X: g.X, Y: g.Y - 40,
		Radius: smokeStartRadius, MaxRadius: smokeCloudRadius,
		OwnerID: g.OwnerID, Life: smokeCloudFrames, MaxLife: smokeCloudFrames,
	})
}

// detonateFlashbang blinds every eligible player (the thrower included) and
// spawns the short burst visual.
func detonateFlashbang(arena Map, g *Grenade, players []*Player, areas *[]AreaEffect) {
	if g == nil {
		return
	}
	fx, fy := g.X, g.Y-flashPointLift
	if areas != nil {
		*areas = append(*areas, AreaEffect{
			Kind: areaEffectFlash, X: fx, Y: fy, Radius: flashbangRadius, MaxRadius: flashbangRadius,
			OwnerID: g.OwnerID, Life: flashBurstFrames, MaxLife: flashBurstFrames,
		})
	}
	for _, p := range players {
		frames, strength := flashbangBlindFor(arena, fx, fy, p)
		applyBlind(p, frames, strength)
	}
}

func playerHeadY(p *Player) float64 {
	return p.Y - 50*playerRenderScale(p)/0.8
}

// flashFacingToward reports whether the player looks at the flash point.
func flashFacingToward(p *Player, fx float64) bool {
	dx := fx - p.X
	if math.Abs(dx) < 24 {
		return true // practically in their face
	}
	facing := p.Facing
	if facing == 0 {
		facing = 1
	}
	return dx*float64(facing) > 0
}

func flashOccluded(arena Map, fx, fy, x, y float64) bool {
	for _, r := range arena.Platforms {
		if segmentIntersectsRect(fx, fy, x, y, r) {
			return true
		}
	}
	return false
}

// flashbangBlindFor returns the blind duration and strength a flash at
// (fx, fy) inflicts on p. Zero frames means no effect.
func flashbangBlindFor(arena Map, fx, fy float64, p *Player) (int, float64) {
	if p == nil || !p.Active || p.PerkNumber == perkFlashGoggles {
		return 0, 0
	}
	if p.Y > fy+flashPointLift+flashLowerLevelThreshold {
		return 0, 0 // on a lower level than the flash
	}
	headY := playerHeadY(p)
	dist := math.Hypot(p.X-fx, headY-fy)
	if dist > flashbangRadius {
		return 0, 0
	}
	if flashOccluded(arena, fx, fy, p.X, headY) {
		return 0, 0
	}
	if !flashFacingToward(p, fx) {
		return flashAwayBlindFrames, flashAwayStrength
	}
	closeness := 1 - dist/flashbangRadius
	frames := flashFullBlindMinFrames + int(math.Round(float64(flashFullBlindMaxFrames-flashFullBlindMinFrames)*closeness))
	return frames, flashFullStrengthMin + (1-flashFullStrengthMin)*closeness
}

func applyBlind(p *Player, frames int, strength float64) {
	if p == nil || frames <= 0 {
		return
	}
	if frames > p.BlindTime {
		// The longer blind replaces the one that is running out.
		p.BlindTime = frames
		p.BlindMax = frames
		p.BlindStrength = strength
		return
	}
	if strength > p.BlindStrength {
		p.BlindStrength = strength
	}
}

func clearBlind(p *Player) {
	p.BlindTime = 0
	p.BlindMax = 0
	p.BlindStrength = 0
}

// blindLevel is the current blind intensity 0..1 (fades over the last
// blindFadeFrames frames).
func blindLevel(p *Player) float64 {
	if p == nil || p.BlindTime <= 0 {
		return 0
	}
	fade := math.Min(1, float64(p.BlindTime)/blindFadeFrames)
	return math.Max(0, math.Min(1, p.BlindStrength)) * fade
}

func updateAreaEffects(areas []AreaEffect, players []*Player) {
	for i := range areas {
		a := &areas[i]
		if a.Dead {
			continue
		}
		a.Frame++
		a.Life--
		switch a.Kind {
		case areaEffectGas:
			grow := math.Min(1, float64(a.Frame)/gasGrowFrames)
			a.Radius = a.MaxRadius * (0.3 + 0.7*grow)
			applyGasPush(a, players)
		case areaEffectSmoke:
			t := math.Min(1, float64(a.Frame)/smokeGrowFrames)
			t = 1 - (1-t)*(1-t) // ease out
			a.Radius = smokeStartRadius + (a.MaxRadius-smokeStartRadius)*t
		case areaEffectTear:
			t := math.Min(1, float64(a.Frame)/tearGasGrowFrames)
			t = 1 - (1-t)*(1-t)
			a.Radius = tearGasStartRadius + (a.MaxRadius-tearGasStartRadius)*t
			applyTearGas(a, players)
		}
		if a.Life <= 0 {
			a.Dead = true
		}
	}
}

// applyGasPush shoves every player inside the cloud outward from its centre.
// It never damages; strength falls off linearly towards the edge.
func applyGasPush(a *AreaEffect, players []*Player) {
	if a == nil || a.Radius <= 0 {
		return
	}
	fade := 1.0
	if a.Life < gasFadeFrames {
		fade = math.Max(0, float64(a.Life)/gasFadeFrames)
	}
	for _, p := range players {
		if p == nil || !p.Active {
			continue
		}
		dx := p.X - a.X
		dy := p.Y - 30 - a.Y
		d := math.Hypot(dx, dy)
		if d >= a.Radius {
			continue
		}
		ux, uy := 0.0, 0.0
		if d < 1 {
			ux = float64(p.Facing)
			if ux == 0 {
				ux = 1
			}
		} else {
			ux, uy = dx/d, dy/d
		}
		force := gasPushStrength * (1 - d/a.Radius) * fade
		if p.PerkNumber == 6 || p.ShieldTime > 0 {
			force *= gasBlastShieldScale
		}
		owner := a.OwnerID != 0 && p.scoreOwnerID() == a.OwnerID
		if owner {
			force *= gasOwnerScale
		}
		p.VX += ux * force
		p.VY += uy * force * gasVerticalFactor
		if !owner && a.OwnerID != 0 {
			p.LastHitBy = a.OwnerID
			p.HitByGrenade = true
		}
	}
}

func compactAreaEffects(in []AreaEffect) []AreaEffect {
	out := in[:0]
	for _, a := range in {
		if !a.Dead {
			out = append(out, a)
		}
	}
	return out
}

// smokeDensity is 0..1: thin while the canister starts and while the cloud
// dissipates.
func smokeDensity(a *AreaEffect) float64 {
	if a == nil || a.Kind != areaEffectSmoke || a.Dead || a.Life <= 0 {
		return 0
	}
	d := math.Min(1, float64(a.Frame)/12)
	if a.Life < smokeFadeFrames {
		d = math.Min(d, float64(a.Life)/smokeFadeFrames)
	}
	return d
}

func smokeBlocksPoint(areas []AreaEffect, x, y float64) bool {
	for i := range areas {
		a := &areas[i]
		if smokeDensity(a) < smokeMinDensity {
			continue
		}
		if math.Hypot(x-a.X, y-a.Y) <= a.Radius*smokeCoreFactor {
			return true
		}
	}
	return false
}

func pointSegmentDistance(px, py, x0, y0, x1, y1 float64) float64 {
	dx, dy := x1-x0, y1-y0
	lenSq := dx*dx + dy*dy
	t := 0.0
	if lenSq > 0 {
		t = math.Max(0, math.Min(1, ((px-x0)*dx+(py-y0)*dy)/lenSq))
	}
	return math.Hypot(px-(x0+dx*t), py-(y0+dy*t))
}

func smokeBlocksSegment(areas []AreaEffect, x0, y0, x1, y1 float64) bool {
	for i := range areas {
		a := &areas[i]
		if smokeDensity(a) < smokeMinDensity {
			continue
		}
		if pointSegmentDistance(a.X, a.Y, x0, y0, x1, y1) <= a.Radius*smokeCoreFactor {
			return true
		}
	}
	return false
}

func playerInSmoke(areas []AreaEffect, p *Player) bool {
	if p == nil || !p.Active {
		return false
	}
	return smokeBlocksPoint(areas, p.X, p.Y-35*playerRenderScale(p)/0.8)
}

// aiSmokeHides reports whether smoke breaks observer's sight of target.
// THERMAL GOGGLES see through smoke; very close targets are still noticed.
func aiSmokeHides(areas []AreaEffect, observer, target *Player) bool {
	if observer == nil || target == nil || observer.PerkNumber == perkThermalGoggles || len(areas) == 0 {
		return false
	}
	if math.Hypot(target.X-observer.X, target.Y-observer.Y) <= aiSmokeCloseRange {
		return false
	}
	ox, oy := observer.X, observer.Y-45
	tx, ty := target.X, target.Y-35
	return smokeBlocksPoint(areas, tx, ty) || smokeBlocksSegment(areas, ox, oy, tx, ty)
}

// aiCanDetect extends aiCanDetectPlayer (invisibility) with blindness and
// smoke.
func (g *Game) aiCanDetect(observer, target *Player) bool {
	if observer != nil && observer.BlindTime > 0 {
		return false
	}
	if !aiCanDetectPlayer(observer, target) {
		return false
	}
	if g == nil {
		return true
	}
	return !aiSmokeHides(g.areaEffects, observer, target)
}

// aiTargetConcealed reports whether the target vanished from observer's view
// because of invisibility or smoke, i.e. the bot should search the last seen
// position instead of simply forgetting the opponent.
func (g *Game) aiTargetConcealed(observer, target *Player) bool {
	if target == nil {
		return false
	}
	if target.InvisibleTime > 0 {
		return true
	}
	return g != nil && aiSmokeHides(g.areaEffects, observer, target)
}

// shooterSmokePenalty: shots fired from inside smoke or through it get a
// small extra spread, unless the shooter wears thermal goggles.
func (g *Game) shooterSmokePenalty(p *Player) bool {
	if g == nil || p == nil || p.PerkNumber == perkThermalGoggles || len(g.areaEffects) == 0 {
		return false
	}
	facing := float64(p.Facing)
	if facing == 0 {
		facing = 1
	}
	mx, my := p.X+23*facing, p.Y-38
	return smokeBlocksPoint(g.areaEffects, mx, my) ||
		smokeBlocksSegment(g.areaEffects, mx, my, mx+smokeShotReach*facing, my)
}

func aimDisturbanceDegrees(p *Player, smokePenalty bool) float64 {
	deg := blindSpreadMaxDegrees * blindLevel(p)
	if p != nil && p.TearGasTime > 0 {
		deg += tearGasSpreadDeg
	}
	if smokePenalty {
		deg += smokeSpreadDegrees
	}
	return deg
}

// disturbBulletAim rotates a freshly fired projectile by deg degrees.
func disturbBulletAim(b *Bullet, deg float64) {
	if b == nil || deg == 0 || b.DevelopedProjectile == DevelopedProjectileStrike {
		return
	}
	rad := deg * math.Pi / 180
	c, s := math.Cos(rad), math.Sin(rad)
	b.VX, b.VY = b.VX*c-b.VY*s, b.VX*s+b.VY*c
	switch {
	case b.Kind == BulletShotgun:
		b.Rotation += deg
	case b.DevelopedProjectile == 0:
		// Straight tracers are drawn along Rotation once it is non-zero.
		b.Rotation = math.Atan2(b.VY, b.VX) * 180 / math.Pi
		if b.Rotation == 0 {
			b.Rotation = 1e-6
		}
	}
}

// applyAimDisturbance spreads the shots p fired this frame (blind and smoke).
func (g *Game) applyAimDisturbance(p *Player, fired []Bullet) {
	if p == nil || len(fired) == 0 {
		return
	}
	maxDeg := aimDisturbanceDegrees(p, g.shooterSmokePenalty(p))
	if maxDeg <= 0 {
		return
	}
	for i := range fired {
		disturbBulletAim(&fired[i], (rand.Float64()*2-1)*maxDeg)
	}
}

// viewerPlayerID returns the one human whose screen this is, or 0 for a
// shared local screen (two or more local humans, or none).
func (g *Game) viewerPlayerID() int {
	if g == nil {
		return 0
	}
	if g.netplay != nil {
		return g.onlineLocalPlayerID()
	}
	id, humans := 0, 0
	for _, p := range g.players {
		if p == nil || p.AI || p.RemoteControlled || p.IsDouble {
			continue
		}
		humans++
		id = p.ID
	}
	if humans == 1 {
		return id
	}
	return 0
}

func (g *Game) playerByID(id int) *Player {
	if g == nil || id == 0 {
		return nil
	}
	for _, p := range g.players {
		if p != nil && p.ID == id {
			return p
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// AI reactions

// aiFlashAwarenessChance is how likely a bot notices a live flashbang and
// turns away. Later campaign levels play smarter.
func (g *Game) aiFlashAwarenessChance() float64 {
	if g != nil && g.campaignMode {
		return math.Max(0.3, math.Min(0.9, 0.3+0.06*float64(g.campaignLevel)))
	}
	return 0.55
}

// updateAIFlashbangAwareness lets bots react to flashbangs that are about to
// pop: with aiFlashAwarenessChance they turn their back to it.
func (g *Game) updateAIFlashbangAwareness() {
	if g == nil {
		return
	}
	chance := g.aiFlashAwarenessChance()
	arena := g.customRuntimeArena()
	for i := range g.grenades {
		gr := &g.grenades[i]
		profile := throwableProfileFor(gr.Type)
		if gr.Dead || profile.Behavior != throwableBehaviorFlash || gr.Fuse != profile.Fuse-flashAIWarningFrames {
			continue
		}
		for _, p := range g.players {
			if p == nil || !p.AI || rand.Float64() >= chance {
				continue
			}
			aiTurnAwayFromFlash(arena, p, gr.X, gr.Y-flashPointLift)
		}
	}
}

// aiTurnAwayFromFlash makes a bot face away from a flash that would fully
// blind it. It returns true when the bot reacted.
func aiTurnAwayFromFlash(arena Map, p *Player, fx, fy float64) bool {
	if p == nil || !p.AI || !p.Active || p.PerkNumber == perkFlashGoggles {
		return false
	}
	probe := *p
	probe.Facing = 1
	if fx < p.X {
		probe.Facing = -1
	}
	if frames, _ := flashbangBlindFor(arena, fx, fy, &probe); frames <= flashAwayBlindFrames {
		return false // out of reach / occluded / lower level anyway
	}
	p.AIFlashAvoidDir = -probe.Facing
	p.AIFlashAvoid = flashAIWarningFrames + 4
	return true
}

// aiApplyFlashAvoid keeps the bot's back turned to a live flashbang: one
// press in the away direction turns it, then it stands still (facing
// persists without input) and holds fire.
func aiApplyFlashAvoid(p *Player) {
	if p == nil || p.AIFlashAvoid <= 0 {
		return
	}
	p.AIFlashAvoid--
	p.AIShoot = false
	p.AILockLeft, p.AILockRight = 0, 0
	if p.JumpNum != 2 {
		return // airborne: leave recovery steering alone
	}
	p.AILeft, p.AIRight = false, false
	if p.Facing != p.AIFlashAvoidDir {
		if p.AIFlashAvoidDir < 0 {
			p.AILeft = true
		} else {
			p.AIRight = true
		}
	}
}

// aiBlindedBehaviour: a blinded bot forgets its target, stumbles around and
// fires wildly (after the worst of the flash).
func aiBlindedBehaviour(p *Player, platforms []Rect) {
	if p == nil || !p.AI || p.BlindTime <= 0 {
		return
	}
	p.AITargetValid = false
	p.AITargetKind = aiTargetNone
	p.AITargetPlayer = false
	p.AITargetPlayerID = 0
	p.AISearchTime = 0
	p.AISearchPlayerID = 0
	// Retarget as soon as sight returns instead of waiting for the next
	// 40-frame scan.
	p.AITargetTimer = 39
	p.AIFlashAvoid = 0

	if p.JumpNum == 2 {
		p.AILockLeft, p.AILockRight = 0, 0
		p.AILeft, p.AIRight = false, false
		switch (p.BlindTime/18 + p.ID) % 4 {
		case 0:
			p.AILeft = true
		case 2:
			p.AIRight = true
		}
		if p.BlindTime%15 == 0 && rand.Intn(2) == 0 {
			// Spin around blindly.
			p.AILeft, p.AIRight = p.Facing > 0, p.Facing < 0
		}
		// Stumbling is fine, walking straight off a ledge is not.
		if p.AIRight && !pointHitsPlatforms(platforms, p.X+64, p.Y+16) {
			p.AIRight = false
		}
		if p.AILeft && !pointHitsPlatforms(platforms, p.X-64, p.Y+16) {
			p.AILeft = false
		}
	}

	p.AIShoot = false
	elapsed := p.BlindMax - p.BlindTime
	if blindLevel(p) < 0.85 || elapsed*100 > p.BlindMax*35 {
		p.AIShoot = (p.BlindTime/5+p.ID)%3 == 0
	}
}

// aiAvoidGasClouds: grounded bots walk out of shock-gas and tear-gas clouds
// and do not walk into one. Gas-mask bots ignore tear gas.
func aiAvoidGasClouds(p *Player, platforms []Rect, areas []AreaEffect) {
	if p == nil || !p.Active || p.JumpNum != 2 || len(areas) == 0 {
		return
	}
	for i := range areas {
		a := &areas[i]
		if a.Dead || a.Life <= 0 {
			continue
		}
		if a.Kind != areaEffectGas && (a.Kind != areaEffectTear || p.PerkNumber == perkGasMask) {
			continue
		}
		vertical := p.Y - 30 - a.Y
		if math.Abs(vertical) > a.Radius {
			continue
		}
		d := math.Hypot(p.X-a.X, vertical)
		if d < a.Radius+10 {
			away := 1.0
			if p.X < a.X || (p.X == a.X && p.Facing < 0) {
				away = -1
			}
			goAway := pointHitsPlatforms(platforms, p.X+64*away, p.Y+16)
			dir := away
			if !goAway {
				// Ledge behind: lean against the shock push, or walk out of
				// tear gas through the other side.
				dir = -away
			}
			p.AILeft, p.AIRight = dir < 0, dir > 0
			if dir < 0 {
				p.AILockLeft, p.AILockRight = 4, 0
			} else {
				p.AILockRight, p.AILockLeft = 4, 0
			}
			return
		}
		if p.AIRight && a.X > p.X && a.X-p.X < a.Radius+40 {
			p.AIRight = false
		}
		if p.AILeft && a.X < p.X && p.X-a.X < a.Radius+40 {
			p.AILeft = false
		}
	}
}

// developedPerkSelectionItems is the source 3x4 perk grid (perks 1..9 in the
// original cells) extended with the developed perks 14+. They take the cells
// of the unused source frames 10..12 and continue on a fifth (compact) row; the GM2
// perk ids 10..13 stay reserved.
func developedPerkSelectionItems() []customSelectionItem {
	source := sourceCustomSelectionItems()[customEditPerk]
	byNumber := map[int]customSelectionItem{}
	for _, item := range source {
		byNumber[item.Number] = item
	}
	out := make([]customSelectionItem, 0, len(selectablePerkNumbers()))
	for _, item := range source {
		if item.Number >= 1 && item.Number <= 9 {
			out = append(out, item)
		}
	}
	first, okFirst := byNumber[1]
	below, okBelow := byNumber[4]
	if !okFirst || !okBelow {
		return out
	}
	rowStep := below.Rect.Y - first.Rect.Y
	for i, perk := range selectablePerkNumbers() {
		if perk <= 9 {
			continue
		}
		cell := 10 + (i - 9) // 10, 11, 12, 13, ...
		col := (cell - 1) % 3
		row := (cell - 1) / 3
		base, ok := byNumber[1+col]
		if !ok {
			continue
		}
		r := base.Rect
		// Developed rows are a bit shorter so a fifth row still ends above
		// the BACK button (Symbol835 y=759.45).
		r.H = math.Min(r.H, 45)
		r.Y = first.Rect.Y + 3*rowStep + float64(row-3)*50
		out = append(out, customSelectionItem{Editor: customEditPerk, Number: perk, Rect: r})
	}
	return out
}
