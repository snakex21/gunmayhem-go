package game

import (
	"math"
	"math/rand"
	"sync"
)

type WeaponState struct {
	Def      WeaponDef
	Bullets  int
	WaitTime int
	Frame    int
	Playing  bool
	Alpha    float64

	// Developed burst weapons (87+) fire their burst as separate rounds a few
	// frames apart. BurstLeft counts the rounds still to come in the current
	// burst and BurstWait the frames until the next one. TriggerLatched marks
	// a semi-auto trigger that has fired and must be released before the next
	// shot. All three live in the player's WeaponState, so they ride along in
	// the host's whole-Player network snapshot.
	BurstLeft      int  `cbor:",omitempty"`
	BurstWait      int  `cbor:",omitempty"`
	TriggerLatched bool `cbor:",omitempty"`
}

type BulletKind int

const (
	BulletNormal   BulletKind = 1 // Symbol 369 / linkage bullet
	BulletShotgun  BulletKind = 2 // Symbol 370 / linkage bullet2
	BulletInstagib BulletKind = 3 // Symbol 378 / linkage bullet3
	BulletMinigun  BulletKind = 6 // Symbol 383 / linkage bullet6
)

type Bullet struct {
	Kind      BulletKind
	X, Y      float64
	VX, VY    float64
	Rotation  float64
	Firepower float64
	OwnerID   int
	Team      int
	Facing    int

	DevelopedProjectile DevelopedProjectileKind
	Gravity             float64
	ExplosionRadius     float64
	ExplosionPower      float64
	Lifetime            int
	IgniteFrames        int
	PortalColor         int
	// GuidanceLost: a Javelin in its terminal phase (or one that lost its
	// target) no longer steers; it only falls.
	GuidanceLost bool `cbor:",omitempty"`
	// ShooterID is the player (or double) that fired: the harpoon rope is
	// drawn from that player's gun and reels toward it (protocol 24).
	ShooterID int `cbor:",omitempty"`

	Time        int
	Alpha       float64
	TrailAlpha  float64
	TrailWidth  float64
	TrailFrame  int
	VisualScale float64
	Dead        bool

	// FlameSweepBack (host-only): a flame spawned at the flamethrower nozzle
	// also sweeps this many px back toward the shooter on its first step, so
	// point-blank targets between the old +23 origin and the nozzle are still
	// hit (developedFlameStart).
	FlameSweepBack float64 `net:"-"`
}

type InstagibTrailEffect struct {
	X, Y     float64
	Rotation float64
	Alpha    float64
	Dead     bool
}

type Shell struct {
	X, Y          float64
	VX, VY        float64
	Rotation      float64
	RotSpeed      float64
	ScaleX        float64
	Kind          int
	OwnerID       int
	Facing        int
	HitsSomething bool
	Dead          bool
}

type MuzzleFlash struct {
	X, Y   float64
	Facing int
	Alpha  float64
	FlipY  bool
	Fresh  bool
	Smoke  bool
	Life   int
	Scale  float64
}

var (
	bulletTrailWidthOnce sync.Once
	bulletTrailWidth     float64
)

func sourceBulletTrailNaturalWidth() float64 {
	bulletTrailWidthOnce.Do(func() {
		if r, err := sourceFrameVisualBounds("Symbol 368", 0); err == nil {
			bulletTrailWidth = r.W
		}
	})
	return bulletTrailWidth
}

func updateWeapon(p *Player, bullets *[]Bullet, shells *[]Shell, flashes *[]MuzzleFlash) {
	if !p.Active {
		return
	}

	w := &p.Weapon
	advanceWeaponTimeline(p, shells)
	pressed := p.shootPressed()
	spec, developed := developedWeaponSpec(w.Def.Number)
	if !pressed {
		w.TriggerLatched = false
	}

	// A developed burst finishes on its own once started, one round every
	// BurstGap frames, whether or not the trigger is still held. The post-burst
	// pause (Def.ROF) only starts counting after its last round.
	if developed && w.BurstLeft > 0 {
		continueDevelopedBurst(p, w, spec, bullets, shells, flashes)
		return
	}

	if w.WaitTime < w.Def.ROF {
		w.WaitTime++
	}

	// DefineSprite_697_player: source shots are accepted only while the weapon
	// movie clip is stopped on Flash frame 2. Custom weapons use the same rest
	// frame but their animation is optional until assets are supplied.
	// AKIMBO (arsenal2) halves the cooldown and lets the second gun fire while
	// the first one's shot timeline still plays (never during a reload).
	if !pressed || w.WaitTime < effectiveWeaponROF(p) || w.Bullets <= 0 || !weaponReadyToFire(p) {
		return
	}
	// Every source weapon 1-86 is hold-to-fire (the AS only tests
	// Key.isDown(shoot) && waittime >= rof). Developed weapons marked semi-auto
	// need the trigger released between shots, using the same held-latch idea
	// as the source grenade key (GrenadeHeld). AI players have no physical
	// trigger: their cadence is already limited by ROF, so they are exempt.
	if developed && w.TriggerLatched && developedTriggerResetRequired(spec) && !p.AI {
		return
	}
	fireWeaponRound(p, w, spec, developed, true, bullets, shells, flashes)
}

// continueDevelopedBurst counts down to the next round of a burst in
// progress and fires it.
func continueDevelopedBurst(p *Player, w *WeaponState, spec DevelopedWeaponSpec, bullets *[]Bullet, shells *[]Shell, flashes *[]MuzzleFlash) {
	w.BurstWait--
	if w.BurstWait > 0 {
		return
	}
	if w.Bullets <= 0 || w.Frame != 1 {
		w.BurstLeft = 0
		w.BurstWait = 0
		return
	}
	w.BurstLeft--
	w.BurstWait = developedBurstGap(spec)
	fireWeaponRound(p, w, spec, true, false, bullets, shells, flashes)
}

// fireWeaponRound fires one accepted shot: a source FIREBULLET() for weapons
// 1-86, or one developed projectile (or pellet load). triggerShot is true for
// the shot that the trigger itself released, false for the follow-up rounds of
// a developed burst.
func fireWeaponRound(p *Player, w *WeaponState, spec DevelopedWeaponSpec, developed, triggerShot bool, bullets *[]Bullet, shells *[]Shell, flashes *[]MuzzleFlash) {
	w.WaitTime = 0
	p.IdleTime = 0

	// Exact visual assignments from the source shooting branch, before
	// hand1.gun.play() and FIREBULLET().
	pose := poseForWeapon(w.Def.Number)
	anim := sourceWeaponAnim(w.Def.Number)
	p.VisualHand2ChildY = pose.HandY
	p.VisualGunRotation = -anim.Blowback
	p.VisualHand1ChildX = pose.ShootX
	p.VisualHand1ChildY = pose.ShootY
	p.VisualHand2ChildX = pose.HandX
	p.VisualHand2ChildY = pose.HandY
	p.VisualGunX = pose.ShootX
	p.VisualGunY = pose.ShootY
	p.VisualHand1ChildX -= anim.Pushback
	p.VisualHand2ChildX -= anim.Pushback
	if anim.Blowback >= 20 && pose.HandX > 40 {
		p.VisualHand2ChildY = -anim.Blowback
	}

	w.Playing = true // hand1.gun.play();
	if developed {
		w.Playing = false
		w.Frame = 1
	}
	recoil := w.Def.Recoil
	if p.PerkNumber == 5 {
		recoil = 0
	}
	p.VX -= recoil * float64(p.Facing)
	offHand := akimboActive(p) && p.AkimboHand == 1
	bulletsBefore, flashesBefore := len(*bullets), len(*flashes)

	mini := p.MiniMulti
	if mini <= 0 {
		mini = 1
	}
	owner := p.scoreOwnerID()
	facing := p.Facing

	// FIREBULLET() from DefineSprite_697_player/frame_1/DoAction.as, extended
	// for the optional 87+ developed arsenal.
	if developed && spec.Category == DevelopedCategoryMelee {
		fireMeleeSwing(p)
	} else if developed && spec.Projectile == DevelopedProjectileGust {
		fireLeafBlower(p)
	} else if developed {
		if w.Def.Shotgun > 0 {
			naturalWidth := sourceBulletTrailNaturalWidth()
			for i := 0; i < w.Def.Shotgun; i++ {
				rotation := -90 + 90*float64(facing) - float64(w.Def.Shotgun)/1.9 + float64(i)*2
				if spec.PelletFanStep > 0 {
					// Wide-choke heavy shotguns: a symmetric fan around the aim
					// line, so only part of the load connects beyond close range.
					rotation = -90 + 90*float64(facing) + (float64(i)-float64(w.Def.Shotgun-1)/2)*spec.PelletFanStep
				}
				speed := spec.ProjectileSpeed + float64(rand.Intn(7)-3)
				rad := rotation * math.Pi / 180
				*bullets = append(*bullets, Bullet{
					Kind: BulletShotgun,
					X:    p.X + 23*mini*float64(facing), Y: p.Y - 38*mini,
					VX: math.Cos(rad) * speed, VY: math.Sin(rad) * speed,
					Rotation: rotation, Firepower: w.Def.Firepower,
					OwnerID: owner, Team: p.Team, Facing: facing,
					Time: 0, Alpha: 1, TrailWidth: naturalWidth, TrailFrame: 1,
				})
			}
		} else {
			// One projectile per call: burst weapons come back through
			// continueDevelopedBurst for each further round.
			{
				spread := (rand.Float64()*2 - 1) * spec.Spread
				speed := spec.ProjectileSpeed
				if speed <= 0 {
					speed = 25
				}
				firepower := w.Def.Firepower
				kind := BulletNormal
				if spec.Category == DevelopedCategoryLMG && w.Def.ROF <= 2 {
					kind = BulletMinigun
				}
				b := Bullet{
					Kind: kind,
					X:    p.X + 23*mini*float64(facing), Y: p.Y - 38*mini + spread,
					VX: speed * float64(facing), VY: 0,
					Firepower: firepower, OwnerID: owner, Team: p.Team, Facing: facing,
					Time: 1, Alpha: 1, TrailWidth: 20, TrailFrame: sourceBulletTrailFrame(firepower),
					DevelopedProjectile: spec.Projectile, Gravity: spec.ProjectileGravity,
					ExplosionRadius: spec.ExplosionRadius, ExplosionPower: spec.ExplosionPower, Lifetime: spec.Lifetime,
					IgniteFrames: spec.IgniteFrames,
				}
				if spec.Projectile == DevelopedProjectileFlame {
					b.X, b.Y, b.Lifetime, b.FlameSweepBack = developedFlameStart(p, w, spec, mini)
					b.Y += spread
					b.VY = (rand.Float64()*2 - 1) * 2.1
					b.VisualScale = .78 + rand.Float64()*.42
				}
				if spec.Projectile == DevelopedProjectilePortal {
					if p.PortalNextColor != PortalOrange {
						p.PortalNextColor = PortalBlue
					}
					b.PortalColor = p.PortalNextColor
					if p.PortalNextColor == PortalBlue {
						p.PortalNextColor = PortalOrange
					} else {
						p.PortalNextColor = PortalBlue
					}
				}
				if spec.Projectile == DevelopedProjectileArcRocket || spec.Projectile == DevelopedProjectileGrenade {
					b.VY = -speed * 0.22
				}
				if spec.Projectile == DevelopedProjectileBolt || spec.Projectile == DevelopedProjectileFlare || spec.Projectile == DevelopedProjectileHarpoon {
					// Bolts/flares are near-flat shots: zeroed at ~420 px with an
					// apex of at most 14 px, so they never clear the 18 px between
					// the muzzle (p.Y-38) and the hitbox top (p.Y-56). The grenade
					// lob (-0.22*speed) overflew every standing target past ~150 px.
					g := spec.ProjectileGravity
					b.VY = -math.Min(g*(420/speed)/2, math.Sqrt(2*g*14))
				}
				if spec.Projectile == DevelopedProjectileTopAttack {
					// Keep the launch visible, then acquire a target for the dive.
					b.VX = speed * .70 * float64(facing)
					b.VY = -speed * .65
				}
				if spec.Projectile == DevelopedProjectileStrike {
					b.X = p.X + 260*float64(facing)
					if p.AI && p.AITargetValid {
						// AI calls the strike around the last position it can actually
						// target. Search targets contain only the last-seen/jittered point,
						// so invisibility is not bypassed here.
						jitter := float64((p.ShotsFired*37+p.ID*19)%71 - 35)
						b.X = p.AITargetX + jitter
					}
					b.Y = -80
					b.VX = 0
					b.VY = speed
				}
				b.ShooterID = p.ID
				*bullets = append(*bullets, b)
			}
		}
		flashX, flashY := p.X+w.Def.FlashX*mini*float64(facing), p.Y+developedFlashY(w.Def.Number)*mini
		// Bows, crossbow and harpoon have no powder charge: no muzzle flash.
		// The nail gun keeps its small flash.
		silentBolt := (spec.Projectile == DevelopedProjectileBolt && w.Def.Number != 180) || spec.Projectile == DevelopedProjectileHarpoon
		if spec.Projectile != DevelopedProjectileFlame && spec.Projectile != DevelopedProjectilePortal && !silentBolt {
			appendMuzzleFlash(flashes, flashX, flashY, facing)
		}
		if w.Def.Number == 182 || w.Def.Number == 193 {
			appendMusketSmoke(flashes, flashX, flashY, facing)
		}
		spawnDevelopedCasing(p, w, spec, shells)
	} else {
		switch {
		case w.Def.Number == 55:
			spread := rand.Float64()*14 - 7
			firepower := w.Def.Firepower + float64(rand.Intn(8)-4)
			*bullets = append(*bullets, Bullet{
				Kind: BulletMinigun,
				X:    p.X + 23*mini*float64(facing), Y: p.Y - 25*mini + spread,
				VX: 25 * float64(facing), Firepower: firepower,
				OwnerID: owner, Team: p.Team, Facing: facing,
				Time: 1, Alpha: 1, TrailWidth: 20, TrailFrame: sourceBulletTrailFrame(firepower),
			})
			appendMuzzleFlash(flashes, p.X+w.Def.FlashX*mini*float64(facing), p.Y-25*mini, facing)

		case w.Def.Number == 9:
			spread := rand.Float64()*6 - 3
			firepower := w.Def.Firepower + float64(rand.Intn(8)-4)
			*bullets = append(*bullets, Bullet{
				Kind: BulletInstagib,
				X:    p.X + 23*mini*float64(facing), Y: p.Y - 38*mini + spread,
				VX: 25 * float64(facing), Firepower: firepower,
				OwnerID: owner, Team: p.Team, Facing: facing,
				Time: 1, Alpha: 1, VisualScale: 1,
			})
			appendMuzzleFlash(flashes, p.X+w.Def.FlashX*mini*float64(facing), p.Y-38*mini, facing)

		case w.Def.Shotgun > 0:
			naturalWidth := sourceBulletTrailNaturalWidth()
			for i := 0; i < w.Def.Shotgun; i++ {
				rotation := -90 + 90*float64(facing) - float64(w.Def.Shotgun)/1.9 + float64(i)*2
				speed := float64(25 + rand.Intn(6) - 3)
				rad := rotation * math.Pi / 180
				*bullets = append(*bullets, Bullet{
					Kind: BulletShotgun,
					X:    p.X + 23*mini*float64(facing), Y: p.Y - 38*mini,
					VX: math.Cos(rad) * speed, VY: math.Sin(rad) * speed,
					Rotation: rotation, Firepower: w.Def.Firepower,
					OwnerID: owner, Team: p.Team, Facing: facing,
					Time: 0, Alpha: 1, TrailWidth: naturalWidth, TrailFrame: 1,
				})
			}
			appendMuzzleFlash(flashes, p.X+w.Def.FlashX*mini*float64(facing), p.Y-38*mini, facing)

		default:
			spread := rand.Float64()*6 - 3
			firepower := w.Def.Firepower + float64(rand.Intn(8)-4)
			*bullets = append(*bullets, Bullet{
				Kind: BulletNormal,
				X:    p.X + 23*mini*float64(facing), Y: p.Y - 38*mini + spread,
				VX: 25 * float64(facing), Firepower: firepower,
				OwnerID: owner, Team: p.Team, Facing: facing,
				Time: 1, Alpha: 1, TrailWidth: 20, TrailFrame: sourceBulletTrailFrame(firepower),
			})
			appendMuzzleFlash(flashes, p.X+w.Def.FlashX*mini*float64(facing), p.Y-38*mini, facing)
		}
	}

	if akimboActive(p) {
		// The off-hand gun fires from a little behind and below the main one.
		shiftAkimboShot(p, offHand, (*bullets)[bulletsBefore:], (*flashes)[flashesBefore:])
		p.AkimboHand = 1 - p.AkimboHand
	}

	// Source increments pgsdata[player-1][3] once per accepted trigger shot,
	// after FIREBULLET() (shotguns still count as one shot, not one per pellet).
	// Each round of a developed burst is a separate shot with its own sound
	// and ammo, so it counts too.
	p.ShotsFired++
	if triggerShot {
		w.TriggerLatched = true
		if developed {
			if rounds := developedBurstRounds(spec); rounds > 1 {
				w.BurstLeft = rounds - 1
				w.BurstWait = developedBurstGap(spec)
			}
		}
	}
	if offHand {
		// The off-hand gun has its own magazine: AKIMBO doubles the fire
		// rate, not the reload count.
		return
	}
	if developed && spec.Category == DevelopedCategoryMelee && p.DefaultWeapon == w.Def.Number {
		// A starter bat never breaks.
		return
	}
	w.Bullets--
	if w.Bullets <= 0 {
		w.BurstLeft = 0
		w.BurstWait = 0
		// Gun Mayhem pickups carry exactly one magazine. Every developed pickup,
		// including the one-round launchers and single-shot rifles, falls through
		// to the source discard branch below; only developed starter pistols
		// reload in place, like the source handguns 1-6.
		if developed {
			if spec.Category == DevelopedCategoryPistol {
				beginDevelopedPistolReload(p, w)
				return
			}
		}
		if w.Def.Number >= 9 {
			// Exact source branch after fx_dropgun creation.
			switch p.GameMode {
			case SourceGameModeInstagib:
				p.EquipWeapon(8)
			case SourceGameModeGunGame:
				p.CurrentLevel -= 2
				p.sourceUpgrade(nil)
				p.EquipWeapon(p.CurrentGun)
			default: // modes1,3,5
				p.EquipWeapon(p.DefaultWeapon)
			}
		} else {
			// hand1.gun.gotoAndPlay(10); reload is executed by source frame scripts.
			w.Playing = true
			enterWeaponTimelineFrame(p, w, 9, shells, 0)
		}
	}
}

func appendMuzzleFlash(flashes *[]MuzzleFlash, x, y float64, facing int) {
	*flashes = append(*flashes, MuzzleFlash{
		X: x, Y: y, Facing: facing,
		Alpha: 1, FlipY: rand.Intn(2) == 0, Fresh: true,
	})
}

func appendMusketSmoke(flashes *[]MuzzleFlash, x, y float64, facing int) {
	*flashes = append(*flashes, MuzzleFlash{
		X: x, Y: y, Facing: facing, Alpha: .72, Fresh: true, Smoke: true, Scale: 1,
	})
}

func sourceBulletTrailFrame(firepower float64) int {
	if firepower > 50 {
		return 2
	}
	return 1
}

// updateBullets is a direct split of the four source projectile movie clips.
// The unused dothehittest() helpers in the original clips are intentionally not
// invoked here: the exported ActionScript defines them but never calls them.
func updateBullets(arena Map, bullets []Bullet, players []*Player, instatrails *[]InstagibTrailEffect, explosions *[]Explosion, blast *[]BlastEffect, portals *[]Portal) {
	for i := range bullets {
		b := &bullets[i]
		if b.Dead {
			continue
		}
		if b.DevelopedProjectile != 0 {
			updateDevelopedProjectile(arena, b, players, explosions, blast, portals)
			continue
		}

		switch b.Kind {
		case BulletShotgun:
			updateShotgunBullet(arena, b, players)
		case BulletInstagib:
			updateInstagibBullet(arena, b, players, instatrails)
		case BulletMinigun:
			updateStraightBullet(arena, b, players, true)
		default:
			updateStraightBullet(arena, b, players, false)
		}
	}
}

func updateDevelopedProjectile(arena Map, b *Bullet, players []*Player, explosions *[]Explosion, blast *[]BlastEffect, portalTargets ...*[]Portal) {
	var portals *[]Portal
	if len(portalTargets) > 0 {
		portals = portalTargets[0]
	}
	if b.Lifetime <= 0 {
		b.Lifetime = 120
	}
	b.Time++

	switch b.DevelopedProjectile {
	case DevelopedProjectileTopAttack:
		updateJavelinGuidance(b, players)
	case DevelopedProjectileStrike:
		b.VY += 0.8
	case DevelopedProjectileFlame:
		// Keep the stream readable as fire rather than a row of bullets: it loses
		// a little horizontal speed and curls upward as it travels.
		b.VX *= 0.988
		b.VY -= 0.035
	default:
		b.VY += b.Gravity
	}

	previousX, previousY := b.X, b.Y
	nextX, nextY := b.X+b.VX, b.Y+b.VY
	sweepX := previousX
	if b.FlameSweepBack > 0 {
		facing := float64(b.Facing)
		if facing == 0 {
			facing = 1
		}
		sweepX -= b.FlameSweepBack * facing
		b.FlameSweepBack = 0
	}
	if b.DevelopedProjectile == DevelopedProjectilePortal {
		if hitPortalPlayerAlongSegment(b, previousX, previousY, nextX, nextY, players, portals) {
			return
		}
	} else {
		if hitDevelopedPlayerAlongSegment(arena, b, sweepX, previousY, nextX, nextY, players, explosions, blast) {
			return
		}
	}
	b.X, b.Y = nextX, nextY

	for _, r := range arena.Platforms {
		if !segmentIntersectsRect(previousX, previousY, nextX, nextY, r) {
			continue
		}
		if b.DevelopedProjectile == DevelopedProjectilePortal {
			placePortalFromSegment(portals, b, previousX, previousY, nextX, nextY, r)
			b.Dead = true
		} else if b.ExplosionRadius > 0 {
			explodeDevelopedProjectile(b, players, explosions, blast)
		} else {
			// Bolts, nails, flares and energy pulses stop on level geometry instead
			// of silently passing through thin platforms between two frames.
			b.Dead = true
		}
		return
	}
	if b.Time >= b.Lifetime || b.Y > 900 || b.X < -600 || b.X > 1500 {
		if b.DevelopedProjectile == DevelopedProjectilePortal {
			placePortalOnNearestSurface(portals, b, arena)
			b.Dead = true
		} else if b.ExplosionRadius > 0 && b.DevelopedProjectile == DevelopedProjectileStrike {
			explodeDevelopedProjectile(b, players, explosions, blast)
		} else {
			b.Dead = true
		}
	}
}

func updateJavelinGuidance(b *Bullet, players []*Player) {
	if b.Time <= 10 {
		// A short climb remains visible on normal maps.
		b.VY += .42
		return
	}
	if b.GuidanceLost {
		b.VY += .35
		return
	}

	var target *Player
	bestDistance := math.MaxFloat64
	for _, p := range players {
		if !bulletCanHitPlayer(b, p) {
			continue
		}
		dx := p.X - b.X
		if dx*float64(b.Facing) < -45 {
			continue
		}
		dy := p.Y - 28 - b.Y
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			target = p
		}
	}

	if target == nil {
		b.VY += 1.35
		if math.Abs(b.VX) < 18 {
			b.VX += .35 * float64(b.Facing)
		}
		return
	}

	dx := target.X - b.X
	dy := target.Y - 28 - b.Y
	distance := math.Hypot(dx, dy)
	if distance < 1 {
		return
	}
	if distance < javelinTerminalDistance && b.Time > javelinTerminalMinTime {
		// Terminal phase: the seeker commits to its current line, so a target
		// that zigzags or jumps at the last moment can still dodge.
		b.GuidanceLost = true
		b.VY += .35
		return
	}
	desiredVX := dx / distance * javelinGuidedSpeed
	desiredVY := dy / distance * javelinGuidedSpeed
	// Smooth steering preserves the top-attack arc instead of snapping.
	b.VX += (desiredVX - b.VX) * javelinSteerRate
	b.VY += (desiredVY - b.VY) * javelinSteerRate
}

// Javelin guidance tuning (30 TPS ticks). Speed 18 / steer .16 with a
// terminal phase inside 130 px keeps standing and running targets hit, while
// zigzagging or jumping targets can dodge it at medium and long range.
const (
	javelinGuidedSpeed      = 18.0
	javelinSteerRate        = .16
	javelinTerminalDistance = 130.0
	javelinTerminalMinTime  = 16
)

func hitDevelopedPlayerAlongSegment(arena Map, b *Bullet, x0, y0, x1, y1 float64, players []*Player, explosions *[]Explosion, blast *[]BlastEffect) bool {
	distance := math.Max(math.Abs(x1-x0), math.Abs(y1-y0))
	steps := int(math.Ceil(distance / 4))
	if steps < 1 {
		steps = 1
	}
	for step := 1; step <= steps; step++ {
		t := float64(step) / float64(steps)
		b.X = x0 + (x1-x0)*t
		b.Y = y0 + (y1-y0)*t
		if b.DevelopedProjectile == DevelopedProjectileFlame {
			if hitPlayerWithFlame(b, players) {
				return true
			}
			continue
		}
		for _, p := range players {
			if !bulletCanHitPlayer(b, p) || !p.Hitbox().Contains(b.X, b.Y) {
				continue
			}
			if b.DevelopedProjectile == DevelopedProjectileHarpoon {
				hitPlayerWithHarpoon(b, p, players)
			} else if b.ExplosionRadius > 0 {
				explodeDevelopedProjectileWithDirectHit(b, p, players, explosions, blast)
			} else {
				hitPlayerWithBullet(arena, b, players, false)
			}
			return true
		}
	}
	return false
}

func segmentIntersectsRect(x0, y0, x1, y1 float64, r Rect) bool {
	distance := math.Max(math.Abs(x1-x0), math.Abs(y1-y0))
	steps := int(math.Ceil(distance / 4))
	if steps < 1 {
		steps = 1
	}
	for step := 0; step <= steps; step++ {
		t := float64(step) / float64(steps)
		if r.Contains(x0+(x1-x0)*t, y0+(y1-y0)*t) {
			return true
		}
	}
	return false
}

// developedExplosionSelfPushScale damps the blast on the shooter (and the
// shooter's doubles). Point-blank rockets still shove the shooter, but no
// longer as hard as the target.
const developedExplosionSelfPushScale = 0.5

func explodeDevelopedProjectile(b *Bullet, players []*Player, explosions *[]Explosion, blast *[]BlastEffect) {
	explodeDevelopedProjectileWithDirectHit(b, nil, players, explosions, blast)
}

// explodeDevelopedProjectileWithDirectHit detonates b. A player struck
// directly by the projectile (direct != nil) takes the full ExplosionPower
// along the projectile's flight direction instead of the distance falloff,
// so a direct rocket hit is the strongest single impulse of that weapon.
func explodeDevelopedProjectileWithDirectHit(b *Bullet, direct *Player, players []*Player, explosions *[]Explosion, blast *[]BlastEffect) {
	if b == nil || b.Dead {
		return
	}
	radius := b.ExplosionRadius
	if radius <= 0 {
		radius = 180
	}
	power := b.ExplosionPower
	if power <= 0 {
		power = 70
	}
	for _, p := range players {
		if p == nil || !p.Active {
			continue
		}
		dx := p.X - b.X
		dy := p.Y - 30 - b.Y
		distance := math.Sqrt(dx*dx + dy*dy)
		if distance > radius {
			continue
		}
		falloff := 1 - distance/radius
		if falloff < 0 {
			falloff = 0
		}
		angle := math.Atan2(dy, dx)
		push := power * falloff
		if p == direct {
			push = power
		}
		if p.ShieldTime > 0 || p.PerkNumber == 6 {
			push *= 0.2
		}
		if p.scoreOwnerID() == b.OwnerID {
			push *= developedExplosionSelfPushScale
		}
		if p == direct {
			dir := float64(b.Facing)
			if b.VX > 0.5 {
				dir = 1
			} else if b.VX < -0.5 {
				dir = -1
			}
			if dir == 0 {
				dir = sign(dx)
			}
			p.VX += dir * push
			p.VY -= push * 0.15
		} else if b.DevelopedProjectile == DevelopedProjectileStrike && p.scoreOwnerID() != b.OwnerID && b.Facing != 0 {
			// A HIMARS strike always pushes its victims along the shooter's
			// facing, so one landing just in front of a target can never pull
			// it back toward the shooter.
			p.VX += float64(b.Facing) * math.Abs(math.Cos(angle)) * push
			p.VY += math.Sin(angle) * push * 0.7
		} else {
			p.VX += math.Cos(angle) * push
			p.VY += math.Sin(angle) * push * 0.7
		}
		p.LastHitBy = b.OwnerID
		p.HitByGrenade = true
	}
	if explosions != nil {
		*explosions = append(*explosions, Explosion{X: b.X, Y: b.Y - 20, Life: 18})
	}
	if blast != nil {
		*blast = append(*blast, spawnGrenadeBlastEffects(b.X, b.Y-20)...)
	}
	b.Dead = true
}

func updateStraightBullet(arena Map, b *Bullet, players []*Player, minigun bool) {
	b.X += b.VX
	b.Y += b.VY

	if hitPlayerWithBullet(arena, b, players, minigun) {
		return
	}
	if b.X < -600 || b.X > 1500 {
		b.Dead = true
		return
	}

	b.Time++
	if minigun {
		if b.Time >= 5 && b.TrailAlpha < 1 {
			b.TrailAlpha += 0.5
			if b.TrailAlpha > 1 {
				b.TrailAlpha = 1
			}
		}
		if b.TrailWidth < 80 {
			b.TrailWidth += 10
		}
		return
	}
	if b.Time >= 3 && b.TrailAlpha < 1 {
		b.TrailAlpha += 0.5
		if b.TrailAlpha > 1 {
			b.TrailAlpha = 1
		}
	}
	if b.TrailWidth < 100 {
		b.TrailWidth += 10
	}
}

func updateShotgunBullet(arena Map, b *Bullet, players []*Player) {
	b.X += b.VX
	b.Y += b.VY

	if hitPlayerWithBullet(arena, b, players, false) {
		return
	}
	if b.X < -600 || b.X > 1500 {
		b.Dead = true
		return
	}

	b.Time++
	if b.Time == 5 {
		b.TrailAlpha = 1
	}
	if b.Firepower >= 7 {
		if b.Time > 6 {
			b.Alpha -= 0.25
		}
		if b.Time > 9 {
			b.Dead = true
		}
	} else {
		if b.Time > 8 {
			b.Alpha -= 0.25
		}
		if b.Time > 11 {
			b.Dead = true
		}
	}
	if b.TrailWidth > 20 {
		b.TrailWidth -= 20
		if b.TrailWidth < 20 {
			b.TrailWidth = 20
		}
	}
}

func updateInstagibBullet(arena Map, b *Bullet, players []*Player, instatrails *[]InstagibTrailEffect) {
	// DefineSprite_378: fx scale is randomized every onEnterFrame before moving.
	b.VisualScale = float64(rand.Intn(50)+50) / 100
	b.X += b.VX
	b.Y += b.VY
	rotation := -90 + 90*float64(b.Facing)
	*instatrails = append(*instatrails, InstagibTrailEffect{X: b.X, Y: b.Y, Rotation: rotation, Alpha: 1})

	for _, p := range players {
		if !bulletCanHitPlayer(b, p) || !p.Hitbox().Contains(b.X, b.Y) {
			continue
		}
		push := b.Firepower * float64(b.Facing)
		if p.DamageMulti > 0 {
			push *= p.DamageMulti
		}
		push *= kevlarBulletKnockbackMultiplier(p)
		// A BALLISTIC SHIELD (arsenal2) stops the de-materializer bolt too.
		shieldBlocked := ballisticShieldCovers(p, b.VX, b.VY)
		if shieldBlocked {
			push *= ballisticShieldBulletScale
			p.ShieldGearSpark = ballisticShieldSparkFrames
		}
		p.VX += push
		p.HitNumber++
		p.HitTimer = 0
		p.LastHitBy = b.OwnerID
		p.HitByGrenade = false
		incrementOwnerHit(players, b.OwnerID)
		// DefineSprite_378 does not kill the target immediately. It resumes the
		// victim's Symbol688 `instagib` child; that timeline runs for 60 frames
		// and only then executes _parent.SELFDESTRUCT(). The target remains fully
		// active during the countdown.
		if !shieldBlocked {
			p.startInstagibTimeline()
		}
		b.Dead = true
		return
	}
	if b.X < -600 || b.X > 1500 {
		b.Dead = true
	}
	b.Time++
}

func hitPlayerWithFlame(b *Bullet, players []*Player) bool {
	if b == nil || b.DevelopedProjectile != DevelopedProjectileFlame {
		return false
	}
	for _, p := range players {
		if !bulletCanHitPlayer(b, p) {
			continue
		}
		h := p.Hitbox()
		margin := 8.0 + math.Max(0, b.VisualScale-0.8)*7
		if b.X < h.X-margin || b.X > h.X+h.W+margin || b.Y < h.Y-margin || b.Y > h.Y+h.H+margin {
			continue
		}
		wasBurning := p.BurnTime > 0
		burnPower := math.Max(16, b.Firepower*12)
		// Flamethrower burns are shorter than Molotov / flare burns so the
		// panic they cause cannot dominate (burnFlamethrowerScale).
		p.igniteWithPower(b.OwnerID, b.Facing, flamethrowerIgniteFrames(b.IgniteFrames), burnPower)
		// A flamethrower is a continuous pressure stream, not sixty tiny bullets
		// per second. Contact refreshes the burn and periodically gives a real
		// knockback pulse; the per-player fire cooldown prevents per-tick
		// stacking (flamethrowerContact* in burn_panic.go).
		p.applyFireContactKnockback(b.OwnerID, b.Facing, flamethrowerContactPush, flamethrowerContactLift, flamethrowerContactCooldown)
		p.LastHitBy = b.OwnerID
		p.HitByGrenade = false
		if p.AISpecial == sourceAISpecialGM2Invisible {
			p.InvisibleTime = -30
		}
		if !wasBurning {
			incrementOwnerHit(players, b.OwnerID)
		}
		b.Dead = true
		return true
	}
	return false
}

func hitPlayerWithBullet(arena Map, b *Bullet, players []*Player, minigun bool) bool {
	for _, p := range players {
		if !bulletCanHitPlayer(b, p) || !p.Hitbox().Contains(b.X, b.Y) {
			continue
		}
		push := b.Firepower * float64(b.Facing)
		if p.ShieldTime == 0 && p.DamageMulti > 0 {
			// Source bullets multiply ordinary knockback by player.damagemulti.
			// Shielded hits use their fixed 0.33/0.30 multipliers instead.
			push *= p.DamageMulti
		}
		if p.ShieldTime > 0 {
			if minigun {
				push *= 0.3 // DefineSprite_383_bullet6
			} else {
				push *= 0.33 // DefineSprite_369/370
			}
		}
		push *= kevlarBulletKnockbackMultiplier(p)
		// BALLISTIC SHIELD (arsenal2): a round into the raised plate keeps 22%
		// of its push and cannot set the bearer on fire.
		shieldBlocked := ballisticShieldCovers(p, b.VX, b.VY)
		if shieldBlocked {
			push *= ballisticShieldBulletScale
			p.ShieldGearSpark = ballisticShieldSparkFrames
		}
		p.VX += push
		p.HitNumber++
		p.HitTimer = 0
		p.LastHitBy = b.OwnerID
		p.HitByGrenade = false
		if p.AISpecial == sourceAISpecialGM2Invisible {
			// GM2 mission 6 reveals its invisible AI for roughly half a second
			// after ordinary bullet/shotgun/minigun hits. The original scripts set
			// invisibletime=-30 and playerAI counts it back toward invisibility.
			p.InvisibleTime = -30
		}
		if b.IgniteFrames > 0 && p.ShieldTime <= 0 && !shieldBlocked {
			p.igniteWithPower(b.OwnerID, b.Facing, b.IgniteFrames, b.Firepower)
		}
		incrementOwnerHit(players, b.OwnerID)
		b.Dead = true
		return true
	}
	return false
}

func incrementOwnerHit(players []*Player, ownerID int) {
	for _, p := range players {
		if p != nil && p.scoreOwnerID() == ownerID && !p.IsDouble {
			p.HitsLanded++
			return
		}
	}
}

func bulletCanHitPlayer(b *Bullet, p *Player) bool {
	if !p.Active {
		return false
	}
	if p.scoreOwnerID() == b.OwnerID {
		// A fresh flame starts beyond the muzzle, so it cannot instantly burn the
		// shooter. After five ticks it becomes an ordinary hazard: running or
		// being knocked into your own stream causes the same timed burn as enemy
		// fire. Other projectiles retain the original no-self-hit rule.
		return b.DevelopedProjectile == DevelopedProjectileFlame && b.Time >= 5
	}
	if b.Team != 0 && p.Team == b.Team {
		return false
	}
	return true
}

func updateInstagibTrails(trails []InstagibTrailEffect) {
	for i := range trails {
		t := &trails[i]
		t.Alpha -= 0.20 // DefineSprite_351_fx_instatrail
		if t.Alpha <= 0.01 {
			t.Dead = true
		}
	}
}

func updateShells(arena Map, shells []Shell, players []*Player) {
	for i := range shells {
		s := &shells[i]
		if s.Dead {
			continue
		}
		s.X += s.VX
		s.Y += s.VY
		s.Rotation += s.RotSpeed
		s.VY += arena.Gravity * 1.3

		// Symbol170 fx_deagle is a reload particle with gameplay collision: the
		// thrown part pushes the first player it touches, tags the hit, then
		// bounces backward at half horizontal speed and cannot hit again.
		if s.Kind == 12 && !s.HitsSomething {
			for _, p := range players {
				if p == nil || !p.Active || !p.Hitbox().Contains(s.X, s.Y) {
					continue
				}
				p.VX += 20 * float64(s.Facing)
				p.HitNumber++
				p.HitTimer = 0
				p.LastHitBy = s.OwnerID
				p.HitByGrenade = false
				incrementOwnerHit(players, s.OwnerID)
				s.VX *= -0.5
				s.X += s.VX
				s.HitsSomething = true
				break
			}
		}
		if s.Y >= 900 {
			s.Dead = true
		}
	}
}

func updateFlashes(flashes []MuzzleFlash) {
	for i := range flashes {
		f := &flashes[i]
		if f.Fresh {
			f.Fresh = false
			continue
		}
		if f.Smoke {
			f.Life++
			f.Y -= .42
			f.X -= .08 * float64(f.Facing)
			f.Scale += .075
			f.Alpha -= .045
			continue
		}
		// DefineSprite_348_fx_flash: _alpha = _alpha - 51.
		f.Alpha -= 0.51
	}
}

func compactBullets(in []Bullet) []Bullet {
	out := in[:0]
	for _, v := range in {
		if !v.Dead {
			out = append(out, v)
		}
	}
	return out
}

func compactInstagibTrails(in []InstagibTrailEffect) []InstagibTrailEffect {
	out := in[:0]
	for _, v := range in {
		if !v.Dead {
			out = append(out, v)
		}
	}
	return out
}

func compactShells(in []Shell) []Shell {
	out := in[:0]
	for _, v := range in {
		if !v.Dead {
			out = append(out, v)
		}
	}
	return out
}

func compactFlashes(in []MuzzleFlash) []MuzzleFlash {
	out := in[:0]
	for _, v := range in {
		if v.Alpha > 0.01 {
			out = append(out, v)
		}
	}
	return out
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}
