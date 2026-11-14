package game

import (
	"math"
	"math/rand"
)

const (
	ThrowableDynamite = iota
	ThrowableMk2
	ThrowableF1
	ThrowableStielhand
	ThrowableRGD5
	ThrowableMolotov
	// Developed utility throwables. ALWAYS append new kinds here: saves and
	// online player configs carry the number, so existing values never move.
	ThrowableShock
	ThrowableFlashbang
	ThrowableSmoke
	ThrowableTearGas
	throwableTypeCount
)

// throwableHitStride is the per-slot stride of the throwable choice menu hit
// codes; it must exceed throwableTypeCount (code = base + slot*stride + kind+1).
const throwableHitStride = 20

// throwableBehavior is what a throwable does when it goes off. Grenade update
// code dispatches on it instead of testing individual throwable kinds.
type throwableBehavior int

const (
	throwableBehaviorExplode throwableBehavior = iota // knockback blast
	throwableBehaviorIgnite                           // Molotov fire strip
	throwableBehaviorShock                            // pushing gas cloud, no damage
	throwableBehaviorFlash                            // blinding flash
	throwableBehaviorSmoke                            // vision-blocking smoke cloud
	throwableBehaviorTearGas                          // coughing cloud: slow, low jump, bad aim
)

type throwableProfile struct {
	Name     string
	Fuse     int
	Behavior throwableBehavior
	// ImpactDetonate breaks the throwable on its first ground contact
	// (Molotov) instead of waiting for the fuse.
	ImpactDetonate     bool
	Radius             float64
	HorizontalPush     float64
	VerticalPush       float64
	EdgeHorizontalPush float64
	EdgeVerticalPush   float64
	// EffectFrames is the lifetime of the cloud (gas/smoke) or the longest
	// blind (flashbang). Zero for plain explosives.
	EffectFrames int
}

var throwableProfiles = [...]throwableProfile{
	// Dynamite keeps the original GM1 impulse exactly: at distance d<=200 the
	// source uses (350-d)/5 horizontally and (350-d)/10 vertically. That means
	// the edge of the blast is still dangerous instead of fading almost to zero.
	{Name: "DYNAMITE", Fuse: 50, Radius: 200, HorizontalPush: 70, VerticalPush: 35, EdgeHorizontalPush: 30, EdgeVerticalPush: 15},
	// Added grenades use the same readable Gun Mayhem-style blast curve, but each
	// keeps a distinct personality: F-1 reaches furthest, M24 hits hardest, RGD-5
	// is balanced and Mk2 sits between them.
	{Name: "US MK 2", Fuse: 58, Radius: 185, HorizontalPush: 61, VerticalPush: 39, EdgeHorizontalPush: 23, EdgeVerticalPush: 17},
	{Name: "SOVIET F-1", Fuse: 70, Radius: 220, HorizontalPush: 56, VerticalPush: 34, EdgeHorizontalPush: 24, EdgeVerticalPush: 15},
	{Name: "GERMAN M24", Fuse: 44, Radius: 170, HorizontalPush: 82, VerticalPush: 45, EdgeHorizontalPush: 31, EdgeVerticalPush: 19},
	{Name: "SOVIET RGD-5", Fuse: 54, Radius: 180, HorizontalPush: 58, VerticalPush: 38, EdgeHorizontalPush: 22, EdgeVerticalPush: 16},
	{Name: "MOLOTOV", Fuse: 90, Behavior: throwableBehaviorIgnite, ImpactDetonate: true, Radius: 100, HorizontalPush: 7, VerticalPush: 4, EdgeHorizontalPush: 2, EdgeVerticalPush: 1},
	// Utility throwables (no blast). Radius is the cloud / flash reach.
	{Name: "SHOCK GRENADE", Fuse: 46, Behavior: throwableBehaviorShock, Radius: gasCloudRadius, EffectFrames: gasCloudFrames},
	{Name: "FLASHBANG", Fuse: 30, Behavior: throwableBehaviorFlash, Radius: flashbangRadius, EffectFrames: flashFullBlindMaxFrames},
	{Name: "SMOKE GRENADE", Fuse: 50, Behavior: throwableBehaviorSmoke, Radius: smokeCloudRadius, EffectFrames: smokeCloudFrames},
	{Name: "TEAR GAS", Fuse: 50, Behavior: throwableBehaviorTearGas, Radius: tearGasCloudRadius, EffectFrames: tearGasCloudFrames},
}

func throwableProfileFor(kind int) throwableProfile {
	return throwableProfiles[normalizedThrowableType(kind)]
}

func normalizedThrowableType(kind int) int {
	if kind < 0 || kind >= throwableTypeCount {
		return ThrowableDynamite
	}
	return kind
}

func throwableName(kind int) string {
	return throwableProfiles[normalizedThrowableType(kind)].Name
}

type Grenade struct {
	X, Y           float64
	VX, VY         float64
	Rotation       float64
	RotSpeed       float64
	Fuse           int
	OwnerID        int
	Type           int
	PortalCooldown int
	FreePass       bool
	HitGround      bool
	Dead           bool
}

type FirePatch struct {
	X, Y    float64
	OwnerID int
	Life    int
	Frame   int
	Dead    bool
}

type Explosion struct {
	X, Y float64
	Life int
}

type DynamiteEffect struct {
	X, Y     float64
	VX, VY   float64
	Rotation float64
	RotSpeed float64
	Scale    float64
	Alpha    float64
	Fresh    bool
	Dead     bool
}

func updateGrenadeInput(p *Player, grenades *[]Grenade) {
	if !p.Active || p.Grenades <= 0 {
		return
	}
	// Source wraps offhand input in gamemode != 2, while Gun Game constructor
	// sets offhandnumber=-1. Both modes therefore have no grenade input.
	if p.GameMode == SourceGameModeInstagib || p.GameMode == SourceGameModeGunGame {
		return
	}

	if p.anchored() {
		// ANCHOR: no throwing while planted. A wind-up in progress is kept
		// and continues after the anchor is released.
		return
	}
	pressed := p.grenadePressed()
	if p.BlindTime > 0 {
		// A blinded player cannot throw: an active wind-up is dropped without
		// spending the throwable, and no new charge starts until sight returns.
		p.GrenadeHeld = false
		p.GrenadePower = 0
		return
	}
	if pressed && !p.GrenadeHeld {
		p.GrenadeHeld = true
		p.GrenadePower = 1
	} else if !pressed && p.GrenadeHeld {
		p.GrenadeHeld = false
		rot := rand.Float64()*10 + 5
		if rand.Intn(2) == 0 {
			rot = -rot
		}
		thrown := throwSpawnGrenade(p.X, p.Y, p.Facing, p.GrenadePower, p.ThrowableType)
		thrown.RotSpeed = rot
		thrown.OwnerID = p.scoreOwnerID()
		*grenades = append(*grenades, thrown)
		p.Grenades--
		// Throw follow-through: the off hand snaps forward and the regular
		// pose easing in updateSourceVisualState swings it back over the next
		// frames (source hand2.gotoAndPlay(6)).
		p.VisualHand2ChildX = grenadeThrowHandX
		p.VisualHand2ChildY = grenadeThrowHandY
	}

	if pressed && p.GrenadePower < 14 {
		p.GrenadePower += 0.5
		if p.GrenadePower > 14 {
			p.GrenadePower = 14
		}
	}
	if pressed && p.GrenadeHeld {
		// Wind-up (source hand2.gotoAndPlay(2), held on frame 5): the off hand
		// is pulled back and up while the throw is charging.
		p.VisualHand2ChildX = grenadeWindupHandX
		p.VisualHand2ChildY = grenadeWindupHandY
	}
}

// Off-hand poses for the grenade wind-up/throw. The original DefineSprite_681
// tween cannot be recovered from the exported assets (all 41 exported frames
// are pixel-identical and the Symbol 681 XFL is missing), so these are local
// hand offsets approximating the wind-up hold and the forward release.
const (
	grenadeWindupHandX = -16.0
	grenadeWindupHandY = -20.0
	grenadeThrowHandX  = 34.0
	grenadeThrowHandY  = -14.0
)

func updateGrenades(arena Map, grenades []Grenade, players []*Player, explosions *[]Explosion, dynamite *[]DynamiteEffect, blast *[]BlastEffect, fire *[]FirePatch) {
	updateGrenadesWithAreas(arena, grenades, players, explosions, dynamite, blast, fire, nil)
}

// updateGrenadesWithAreas is updateGrenades plus the area-effect list that gas
// and smoke clouds (and the flashbang burst visual) are spawned into. A nil
// areas pointer still applies every direct effect (blast, fire, blind).
func updateGrenadesWithAreas(arena Map, grenades []Grenade, players []*Player, explosions *[]Explosion, dynamite *[]DynamiteEffect, blast *[]BlastEffect, fire *[]FirePatch, areas *[]AreaEffect) {
	for i := range grenades {
		g := &grenades[i]
		if g.Dead {
			continue
		}

		previousX, previousY := stepGrenadeFlight(arena, g, dynamite)

		g.Fuse++
		profile := throwableProfileFor(g.Type)
		if (profile.Behavior == throwableBehaviorExplode || profile.Behavior == throwableBehaviorShock) &&
			!profile.ImpactDetonate && g.Fuse == profile.Fuse-grenadeAIWarningFrames {
			// DefineSprite_665: at fusetime==40 (10 frames before the 50-frame
			// dynamite blast) nearby AI players jump away from the grenade.
			// (Flashbangs use their own turn-away reaction, see
			// Game.updateAIFlashbangAwareness.)
			aiFleeGrenade(arena, g.X, g.Y, players)
		}
		goesOff, impact := grenadeGoesOff(arena, g, profile, previousX, previousY)
		if goesOff {
			switch profile.Behavior {
			case throwableBehaviorShock:
				g.Dead = true
				spawnGasCloud(g, areas)
				spawnMolotovSparks(dynamite, g.X, g.Y-6, 5)
				continue
			case throwableBehaviorFlash:
				g.Dead = true
				detonateFlashbang(arena, g, players, areas)
				continue
			case throwableBehaviorSmoke:
				g.Dead = true
				spawnSmokeCloud(g, areas)
				spawnMolotovSparks(dynamite, g.X, g.Y-6, 4)
				continue
			case throwableBehaviorTearGas:
				g.Dead = true
				spawnTearGasCloud(g, areas)
				spawnMolotovSparks(dynamite, g.X, g.Y-6, 4)
				continue
			}
			if profile.Behavior == throwableBehaviorIgnite {
				molotovImpact := impact
				g.Dead = true
				if !molotovImpact {
					// Fuse ran out in mid-air: the burning bottle drops to the
					// nearest surface below. With no surface in reach it fizzles
					// instead of leaving a fire patch floating in the sky.
					surfaceY, ok := molotovSurfaceBelow(arena, g.X, g.Y, molotovTimeoutDrop)
					if !ok {
						spawnMolotovSparks(dynamite, g.X, g.Y, 4)
						continue
					}
					g.Y = surfaceY
				}
				if spawnMolotovFire(arena, g, players, fire) {
					spawnMolotovSparks(dynamite, g.X, g.Y-6, 9)
				}
				continue
			}
			explodeGrenade(g, players)
			*blast = append(*blast, spawnGrenadeBlastEffects(g.X, g.Y-20)...)
			*explosions = append(*explosions, Explosion{X: g.X, Y: g.Y - 20, Life: 18})
			g.Dead = true
			continue
		}

		if grenadeOffMap(g) {
			g.Dead = true
		}
	}
}

// stepGrenadeFlight advances one live throwable by one frame of flight: spin,
// gravity, movement, the platform bounce and the free-pass flag, in the exact
// source order. It returns the position before this frame's move (needed for
// the Molotov top-crossing test). dynamite may be nil (no spark particles):
// the throw simulator (simulateThrow) runs this same code headlessly.
func stepGrenadeFlight(arena Map, g *Grenade, dynamite *[]DynamiteEffect) (previousX, previousY float64) {
	if !g.HitGround {
		g.Rotation += g.RotSpeed
	} else if math.Abs(90-g.Rotation) < math.Abs(-90-g.Rotation) {
		g.Rotation += (90 - g.Rotation) / 5
	} else {
		g.Rotation += (-90 - g.Rotation) / 5
	}

	if g.VY < 24 {
		g.VY += arena.Gravity + 0.12
	}
	// Source spawns fx_dynamite at the grenade's tip before moving the
	// grenade itself this frame.
	dirX := math.Cos((g.Rotation-90)*math.Pi/180) * 25
	dirY := math.Sin((g.Rotation-90)*math.Pi/180) * 25
	if dynamite != nil && normalizedThrowableType(g.Type) == ThrowableDynamite {
		*dynamite = append(*dynamite, newDynamiteEffect(g.X+dirX, g.Y+dirY))
	}
	previousX, previousY = g.X, g.Y
	g.X += g.VX
	g.Y += g.VY

	if !g.FreePass && g.VY > 0 {
		for _, r := range arena.Platforms {
			if !r.ContainsX(g.X) || g.Y+8 < r.Y || g.Y+8-g.VY > r.Y {
				continue
			}
			g.Y -= g.VY
			for step := 1; step <= 5; step++ {
				y := g.Y + 8 + float64(step)*(g.VY/5)
				if r.Contains(g.X, y) {
					g.Y += g.VY / 5 * float64(step-1)
					g.RotSpeed *= 0.5
					g.VX *= 0.6
					g.VY *= -0.4
					g.HitGround = true
					break
				}
			}
			break
		}
	}

	insideY := false
	insideY10 := false
	for _, r := range arena.Platforms {
		if r.Contains(g.X, g.Y) {
			insideY = true
		}
		if r.Contains(g.X, g.Y+10) {
			insideY10 = true
		}
		if insideY && insideY10 {
			break
		}
	}
	// Exact source order:
	// 1) clear freepass only when neither _Y nor _Y+10 hits ground,
	// 2) set freepass only when _Y itself hits ground.
	// Using (_Y || _Y+10) for step 2 lets the grenade enter a platform
	// several pixels too far before collisions resume.
	if !insideY && !insideY10 {
		g.FreePass = false
	}
	if insideY && !g.FreePass {
		g.FreePass = true
	}
	return previousX, previousY
}

// grenadeGoesOff runs after the fuse tick: it reports whether the throwable
// detonates this frame and whether that is an impact (Molotov) rather than
// the fuse running out. A Molotov crossing a platform top is snapped onto it.
func grenadeGoesOff(arena Map, g *Grenade, profile throwableProfile, previousX, previousY float64) (goesOff, impact bool) {
	if profile.ImpactDetonate {
		impact = g.HitGround
		// A fast Molotov should not tunnel through a thin platform between
		// frames. Break it on the first platform top crossed while descending.
		if !impact {
			if impactX, impactY, ok := molotovTopImpact(arena, previousX, previousY, g.X, g.Y); ok {
				g.X, g.Y = impactX, impactY
				g.HitGround = true
				impact = true
			}
		}
	}
	return g.Fuse > profile.Fuse || impact, impact
}

// grenadeOffMap is the source cleanup bound for a lost throwable.
func grenadeOffMap(g *Grenade) bool {
	return g.Y >= 900 || g.X < -500 || g.X > 1400
}

// throwSpawnGrenade is the throwable updateGrenadeInput releases from a
// player standing at (x, y) facing `facing` with the given charge.
func throwSpawnGrenade(x, y float64, facing int, power float64, kind int) Grenade {
	if facing == 0 {
		facing = 1
	}
	return Grenade{
		X:        x + 30*float64(facing),
		Y:        y - 35,
		VX:       float64(facing) * power,
		VY:       -power,
		Rotation: float64(facing) * 50,
		Type:     normalizedThrowableType(kind),
	}
}

// throwSimMaxFrames bounds a simulated flight; every fuse is far shorter.
const throwSimMaxFrames = 400

// throwSimResult is the predicted outcome of one throw.
type throwSimResult struct {
	// X, Y is where the throwable goes off: the grenade position on its
	// detonation frame (the blast / cloud / flash origin), for a Molotov the
	// break point after the top-crossing snap or the surface it falls to.
	X, Y float64
	// Frames from the release frame (inclusive) until it goes off.
	Frames int
	// Landed: it touched a platform before going off.
	Landed bool
	// OK is false when the throwable is lost off the map or a Molotov
	// fizzles with no surface below; X/Y then hold the last position.
	OK bool
	// Path holds the position after every simulated frame (only filled when
	// requested), and LandIndex the index of the first ground contact (-1).
	Path      []grenadePreviewPoint
	LandIndex int
}

// simulateThrow predicts a throw released by a player at (x, y). It runs the
// exact flight/fuse code of updateGrenadesWithAreas (stepGrenadeFlight and
// grenadeGoesOff) without touching any game state. Portals are ignored.
func simulateThrow(arena Map, x, y float64, facing int, power float64, kind int) throwSimResult {
	return simulateGrenade(arena, throwSpawnGrenade(x, y, facing, power, kind), false)
}

// simulateGrenade advances a copy of g until it goes off or is lost. A thrown
// grenade receives its first update on its release frame, so frame 1 here is
// that frame.
func simulateGrenade(arena Map, g Grenade, withPath bool) throwSimResult {
	res := throwSimResult{LandIndex: -1}
	profile := throwableProfileFor(g.Type)
	for frame := 1; frame <= throwSimMaxFrames; frame++ {
		previousX, previousY := stepGrenadeFlight(arena, &g, nil)
		g.Fuse++
		goesOff, impact := grenadeGoesOff(arena, &g, profile, previousX, previousY)
		if withPath {
			res.Path = append(res.Path, grenadePreviewPoint{g.X, g.Y})
		}
		if g.HitGround && res.LandIndex < 0 {
			res.LandIndex = frame - 1
			res.Landed = true
		}
		res.X, res.Y, res.Frames = g.X, g.Y, frame
		if goesOff {
			res.OK = true
			if profile.Behavior == throwableBehaviorIgnite && !impact {
				surfaceY, ok := molotovSurfaceBelow(arena, g.X, g.Y, molotovTimeoutDrop)
				res.OK = ok
				if ok {
					res.Y = surfaceY
				}
			}
			return res
		}
		if grenadeOffMap(&g) {
			return res
		}
	}
	return res
}

func grenadeBlastAxis(center, edge, distance, radius float64) float64 {
	if radius <= 0 || distance >= radius {
		return edge
	}
	t := distance / radius
	return center + (edge-center)*t
}

func explodeGrenade(g *Grenade, players []*Player) {
	profile := throwableProfiles[normalizedThrowableType(g.Type)]
	for _, p := range players {
		if !p.Active {
			continue
		}
		dx := p.X - g.X
		dy := p.Y - 30 - g.Y
		distance := math.Round(math.Sqrt(dx*dx + dy*dy))
		if distance > profile.Radius {
			continue
		}
		radians := math.Atan2(dy, dx)
		pushXMag := grenadeBlastAxis(profile.HorizontalPush, profile.EdgeHorizontalPush, distance, profile.Radius)
		pushYMag := grenadeBlastAxis(profile.VerticalPush, profile.EdgeVerticalPush, distance, profile.Radius)
		pushX := math.Cos(radians) * pushXMag
		pushY := math.Sin(radians) * pushYMag
		if p.PerkNumber == 6 || p.ShieldTime > 0 {
			// Source switches from /5,/10 to /25,/50: exactly one fifth
			// of the normal grenade impulse on each axis.
			pushX /= 5
			pushY /= 5
		}
		p.VX += pushX
		p.VY += pushY
		if g.OwnerID >= 0 {
			p.LastHitBy = g.OwnerID
			p.HitByGrenade = true
		}
		// A negative owner marks a neutral arena hazard (EXTRA dynamite rain):
		// it pushes, but the last real attacker keeps the kill credit.
	}
}

func molotovTopImpact(arena Map, x0, y0, x1, y1 float64) (float64, float64, bool) {
	if y1 <= y0 {
		return 0, 0, false
	}
	bestT := 2.0
	bestX, bestY := 0.0, 0.0
	for _, r := range arena.Platforms {
		if y0 > r.Y || y1 < r.Y {
			continue
		}
		t := (r.Y - y0) / (y1 - y0)
		if t < 0 || t > 1 || t >= bestT {
			continue
		}
		x := x0 + (x1-x0)*t
		if x < r.X || x > r.X+r.W {
			continue
		}
		bestT, bestX, bestY = t, x, r.Y
	}
	return bestX, bestY, bestT <= 1
}

const (
	// grenadeAIWarningFrames mirrors the source dodge at fusetime==40 for the
	// 50-frame dynamite fuse; typed grenades warn the same 10 frames early.
	grenadeAIWarningFrames = 10
	// molotovImpactDrop is how far below the impact point a fire patch may
	// settle (bounce-landing path).
	molotovImpactDrop = 70.0
	// molotovTimeoutDrop is how far a Molotov that timed out mid-air may fall
	// to find a surface before it fizzles.
	molotovTimeoutDrop = 160.0
	// molotovSurfaceAbove tolerates the bounce code resting the bottle up to
	// 8 px above the surface it actually hit; anything higher is a different
	// (upper) platform and must not catch fire.
	molotovSurfaceAbove  = 12.0
	molotovFireHalfWidth = 34.0
	// molotovBlastShield is the BLAST SHIELD (perk 6) multiplier for Molotov
	// impact/fire: half burn time and half contact push.
	molotovBlastShield = 0.5
)

// molotovSurfaceBelow returns the nearest platform top under x that lies at
// most molotovSurfaceAbove px above referenceY and at most maxDrop px below
// it. Unlike the old absolute-distance search it never snaps fire *up* onto a
// platform above the bottle.
func molotovSurfaceBelow(arena Map, x, referenceY, maxDrop float64) (float64, bool) {
	bestY := 0.0
	found := false
	for _, r := range arena.Platforms {
		if x < r.X || x > r.X+r.W {
			continue
		}
		if r.Y < referenceY-molotovSurfaceAbove || r.Y > referenceY+maxDrop {
			continue
		}
		if !found || r.Y < bestY {
			bestY = r.Y
			found = true
		}
	}
	return bestY, found
}

// fireSurfaceY returns the surface a Molotov breaking at (x, referenceY) burns on.
func fireSurfaceY(arena Map, x, referenceY float64) (float64, bool) {
	return molotovSurfaceBelow(arena, x, referenceY, molotovImpactDrop)
}

// fireOwnerImmune reports whether p started this fire. The thrower (and their
// DOUBLE clones) never burn in their own Molotov fire; environmental fire
// (owner 0, chaos rain) burns everybody. Other players, teammates included,
// are still affected exactly as before (fire has never had team rules).
func fireOwnerImmune(p *Player, ownerID int) bool {
	return p != nil && ownerID != 0 && p.scoreOwnerID() == ownerID
}

// spawnMolotovSparks reuses the fx_dynamite spark particle for the bottle
// breaking (impact) or fizzling (timed out with no surface below).
func spawnMolotovSparks(dynamite *[]DynamiteEffect, x, y float64, count int) {
	if dynamite == nil {
		return
	}
	for i := 0; i < count; i++ {
		e := newDynamiteEffect(x, y)
		e.VY -= rand.Float64() * 3
		*dynamite = append(*dynamite, e)
	}
}

// spawnMolotovFire breaks the bottle at g's position. It returns false when no
// surface was found, in which case nothing ignites (no floating fire).
func spawnMolotovFire(arena Map, g *Grenade, players []*Player, patches *[]FirePatch) bool {
	profile := throwableProfiles[ThrowableMolotov]
	if patches != nil {
		centerY, ok := fireSurfaceY(arena, g.X, g.Y)
		if !ok {
			return false
		}
		// Spread only along the struck surface itself. A neighbouring x that
		// runs past the platform edge does not drip fire onto a lower floor.
		for i := -3; i <= 3; i++ {
			x := g.X + float64(i)*28
			y, ok := molotovSurfaceBelow(arena, x, centerY, 0)
			if !ok || math.Abs(y-centerY) > 0.5 {
				continue
			}
			*patches = append(*patches, FirePatch{
				X: x, Y: y, OwnerID: g.OwnerID,
				Life: 300 - int(math.Abs(float64(i)))*20,
			})
		}
	}
	for _, p := range players {
		if p == nil || !p.Active || fireOwnerImmune(p, g.OwnerID) || math.Hypot(p.X-g.X, p.Y-30-g.Y) > profile.Radius {
			continue
		}
		dir := 1
		if p.X < g.X {
			dir = -1
		}
		// BLAST SHIELD halves the burn inside igniteWithPower and the shove here.
		frames, shove, lift := 165, 5.5, 2.8
		if p.PerkNumber == 6 {
			shove *= molotovBlastShield
			lift *= molotovBlastShield
		}
		p.igniteWithPower(g.OwnerID, dir, frames, 20)
		// Molotov is primarily an ignition weapon. Give the bottle impact a small
		// readable shove without turning it into another high-explosive grenade.
		p.VX += float64(dir) * shove
		p.VY -= lift
		p.LastHitBy = g.OwnerID
		p.HitByGrenade = true
	}
	return true
}

func firePatchTouches(p *Player, patch *FirePatch) bool {
	if math.Abs(p.X-patch.X) > molotovFireHalfWidth {
		return false
	}
	// Player Y is effectively the feet/ground contact. Fire rises from the
	// platform, so it can catch somebody standing in it or jumping just over
	// it, but not players a full platform-height away.
	vertical := p.Y - patch.Y
	return vertical >= -52 && vertical <= 16
}

func updateFirePatches(patches []FirePatch, players []*Player) {
	for i := range patches {
		patch := &patches[i]
		if patch.Dead {
			continue
		}
		patch.Frame++
		patch.Life--
		if patch.Life <= 0 {
			patch.Dead = true
		}
	}
	for _, p := range players {
		if p == nil || !p.Active {
			continue
		}
		// Resolve every patch touching this player once per tick. The old code
		// let each overlapping patch re-aim BurnDirection and the contact push,
		// so a player standing inside a fire line was shoved left/right/left.
		var source *FirePatch
		for i := range patches {
			patch := &patches[i]
			if patch.Dead || patch.Frame%6 != 0 || fireOwnerImmune(p, patch.OwnerID) || !firePatchTouches(p, patch) {
				continue
			}
			if source == nil || math.Abs(p.X-patch.X) < math.Abs(p.X-source.X) {
				source = patch
			}
		}
		if source == nil {
			continue
		}
		dir := fireEjectDirection(p, patches, source)
		frames, power, push, lift := 125, 18.0, 5.5, 4.2
		if p.PerkNumber == 6 {
			// BLAST SHIELD halves Molotov fire: the burn (and its panic jolts)
			// inside igniteWithPower, the contact pop here. Shield powerup
			// immunity is handled by igniteWithPower.
			push *= molotovBlastShield
			lift *= molotovBlastShield
		}
		p.igniteWithPower(source.OwnerID, dir, frames, power)
		// Standing in burning Molotov fuel must physically matter in a
		// knockback game. Contact gives a readable pop away from the fire's
		// centre; the per-player cooldown prevents stacking.
		p.applyFireContactKnockback(source.OwnerID, dir, push, lift, 8)
		p.LastHitBy = source.OwnerID
	}
}

// fireEjectDirection pushes a player out of the nearer side of the burning
// strip they stand in: away from the centre of all live patches on the same
// surface within reach. Exactly at the centre it keeps the current burn
// direction so the push never alternates.
func fireEjectDirection(p *Player, patches []FirePatch, source *FirePatch) int {
	sum, n := 0.0, 0
	for i := range patches {
		patch := &patches[i]
		if patch.Dead || math.Abs(patch.Y-source.Y) > 0.5 || math.Abs(patch.X-p.X) > 120 {
			continue
		}
		sum += patch.X
		n++
	}
	center := source.X
	if n > 0 {
		center = sum / float64(n)
	}
	switch {
	case p.X > center+2:
		return 1
	case p.X < center-2:
		return -1
	case p.BurnTime > 0 && p.BurnDirection != 0:
		return p.BurnDirection
	case p.Facing < 0:
		return 1 // back out the way they walked in
	default:
		return -1
	}
}

func compactFirePatches(in []FirePatch) []FirePatch {
	out := in[:0]
	for _, patch := range in {
		if !patch.Dead {
			out = append(out, patch)
		}
	}
	return out
}

func newDynamiteEffect(x, y float64) DynamiteEffect {
	vx := rand.Float64()*10 - 5
	vy := rand.Float64()*4 - 2
	rot := rand.Float64()*10 + 5
	if rand.Intn(2) == 0 {
		rot *= -1
	}
	// DefineSprite_343 frame script moves the new clip once immediately.
	return DynamiteEffect{
		X: x + vx, Y: y + vy,
		VX: vx, VY: vy, RotSpeed: rot,
		Scale: 1, Alpha: 1, Fresh: true,
	}
}

func updateDynamiteEffects(effects []DynamiteEffect) {
	for i := range effects {
		e := &effects[i]
		if e.Dead {
			continue
		}
		if e.Fresh {
			e.Fresh = false
			continue
		}
		e.X += e.VX
		e.VX += e.VX * -0.8
		e.Y += e.VY
		e.Scale -= 0.05
		e.Rotation += e.RotSpeed
		e.Alpha -= 0.20
		if e.Alpha <= 0.01 {
			e.Dead = true
		}
	}
}

func compactDynamiteEffects(in []DynamiteEffect) []DynamiteEffect {
	out := in[:0]
	for _, e := range in {
		if !e.Dead {
			out = append(out, e)
		}
	}
	return out
}

func updateExplosions(explosions []Explosion) {
	for i := range explosions {
		if explosions[i].Life > 0 {
			explosions[i].Life--
		}
	}
}

func compactGrenades(in []Grenade) []Grenade {
	out := in[:0]
	for _, v := range in {
		if !v.Dead {
			out = append(out, v)
		}
	}
	return out
}

func compactExplosions(in []Explosion) []Explosion {
	out := in[:0]
	for _, v := range in {
		if v.Life > 0 {
			out = append(out, v)
		}
	}
	return out
}
