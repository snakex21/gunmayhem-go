package game

import (
	"errors"
	"math"
	"unicode/utf8"
)

// Shared rules of the big Stage B campaign enemies (the MECH walker and the
// GUNSHIP helicopter boss): push-based HP, co-op HP scaling, marked mortar
// strikes, machine-gun rounds that stop on platforms, gunship rockets that
// fly back at the gunship once batted or blown, and the death sequences.
// See docs/campaign_enemies.md ("Stage B").

const (
	// bigEnemyPushPerHP: damage = knockback. Every bigEnemyPushPerHP of raw
	// push (before the enemy's own resistance) costs one HP, so a starter
	// pistol round (push 20) is 1 HP, a sniper round (60) 3 HP, a bat swing
	// (26) 1.3 HP and a point-blank rocket about 6 HP.
	bigEnemyPushPerHP = 20.0
	// bigEnemyInstagibHP: the de-materializer cannot erase a big enemy; its
	// beam burns this much HP instead.
	bigEnemyInstagibHP = 10
	// bigEnemyCoopHPStep: each extra human on the player side adds this
	// fraction of the base HP.
	bigEnemyCoopHPStep = 0.6

	bossShotRound  int8 = 1 // machine-gun round: stops on platforms
	bossShotRocket int8 = 2 // gunship rocket: can be sent back

	campaignStrikeLinger = 10
)

// campaignCoopHP scales a big enemy's HP with the number of humans.
func (g *Game) campaignCoopHP(hp int) int {
	humans := 0
	for _, p := range g.players {
		if p != nil && !p.AI && !p.IsDouble && p.Team != campaignEnemyTeam {
			humans++
		}
	}
	if humans <= 1 {
		return hp
	}
	return int(math.Round(float64(hp) * (1 + bigEnemyCoopHPStep*float64(humans-1))))
}

func (k campaignEnemyKind) wreckFrames() int {
	if k.bigEnemy() {
		return campaignBigWreckFrames
	}
	return campaignEnemyDeathFXFrames
}

func (k campaignEnemyKind) killScore() int {
	switch k {
	case campaignEnemyMech:
		return 400
	case campaignEnemyHelicopter:
		return 1000
	}
	return 100
}

// endBigEnemyHitPhase turns this tick's push into HP. The impulse seen here
// already went through the enemy's KnockbackScale (the ability windows), so
// dividing by that scale recovers the raw push of the attack. A mech hit in
// the back (its exposed reactor) is pushed and damaged mechBackMult times
// harder.
func (g *Game) endBigEnemyHitPhase(e *CampaignEnemy) {
	b := &e.Body
	scale := abilityKnockbackScale(b)
	if scale <= 0 {
		scale = 1
	}
	dvx := b.VX - e.preVX - e.windVX*scale
	dvy := b.VY - e.preVY - e.windVY*scale
	e.windVX, e.windVY = 0, 0
	raw := math.Hypot(dvx, dvy) / scale
	hits := b.HitNumber - e.preHits
	if campaignEnemyKind(e.Kind) == campaignEnemyMech && raw > 0.5 {
		if back := g.mechBackHit(e, dvx); back {
			b.VX += dvx * (mechBackMult - 1)
			b.VY += dvy * (mechBackMult - 1)
			raw *= mechBackMult
			e.ShieldSpark = 8 // weak-spot sparks on the reactor
		}
	}
	if b.InstagibPlaying {
		// The beam cannot erase it: a heavy burn instead.
		b.InstagibPlaying = false
		b.InstagibFrame = 0
		raw += bigEnemyInstagibHP * bigEnemyPushPerHP
	}
	if hits > 0 || raw > 0.5 {
		e.HitFlash = 6
		e.creditTimer = campaignEnemyCreditFrames
	}
	if e.MaxHP <= 0 || raw <= 0 {
		return
	}
	g.damageBigEnemy(e, raw/bigEnemyPushPerHP, b.LastHitBy)
}

// damageBigEnemy removes hp (fractions carry over) and starts the death
// sequence at zero. creditID is the attacker kept for the kill.
func (g *Game) damageBigEnemy(e *CampaignEnemy, hp float64, creditID int) {
	if e == nil || !e.hittable() || e.MaxHP <= 0 || hp <= 0 {
		return
	}
	if creditID > 0 {
		e.Body.LastHitBy = creditID
		e.creditTimer = campaignEnemyCreditFrames
	}
	e.dmgAcc += hp
	whole := int(e.dmgAcc)
	if whole <= 0 {
		return
	}
	e.dmgAcc -= float64(whole)
	e.HP -= whole
	if e.HP > 0 {
		return
	}
	e.HP = 0
	e.crashCredit = e.Body.LastHitBy
	switch campaignEnemyKind(e.Kind) {
	case campaignEnemyMech:
		e.State, e.StateTimer = mechStateMeltdown, 0
	case campaignEnemyHelicopter:
		g.startHeliCrash(e)
	}
}

// finishBigEnemy ends a death sequence: the kill goes to whoever brought
// the HP to zero.
func (g *Game) finishBigEnemy(e *CampaignEnemy) {
	if e.crashCredit > 0 {
		e.Body.LastHitBy = e.crashCredit
	}
	g.destroyCampaignEnemy(e)
}

// bigEnemyDeathFX: a big blast; the wreck is drawn from DeadFrames.
func (g *Game) bigEnemyDeathFX(e *CampaignEnemy) {
	b := &e.Body
	x, y := b.LastDeathX, b.LastDeathY-40
	for i := 0; i < 3; i++ {
		g.blastFX = append(g.blastFX, spawnGrenadeBlastEffects(x+float64(i-1)*26, y-float64(i%2)*18)...)
	}
	g.explosions = append(g.explosions, Explosion{X: x, Y: y, Life: 18})
	spawnMolotovSparks(&g.dynamiteFX, x, y, 10)
	g.playRandomExplosionSound()
	g.playSourceSFX("emp.wav", true)
}

// --- marked strikes (mech mortar) -------------------------------------------

// campaignStrike is a strike waiting for its warning to run out.
type campaignStrike struct {
	x, landY, startY float64
	delay            int
	ownerID          int
	facing           int
}

const (
	campaignStrikeStartAbove = 300.0
	campaignStrikeFallSpeed  = 6.0
	campaignStrikeRadius     = 110.0
	campaignStrikePower      = 22.0
	campaignStrikePush       = 18.0
)

// queueCampaignStrike marks the surface under (x, refY) now and drops a
// strike there after warn ticks. The shell appears just under the next
// platform above (a mortar round comes down onto the target's floor, not on
// the roof of a tiered map). The marker stays until the shell has landed.
func (g *Game) queueCampaignStrike(arena Map, x, refY float64, warn, ownerID, facing int) {
	landY := surfaceUnder(arena, x, refY)
	if !platformAtX(arena, x) {
		landY = hazardLandingY(arena, x)
	}
	startY := campaignStrikeStart(arena, x, landY)
	fall := himarsStrikeFallTicks(startY, campaignStrikeFallSpeed, landY)
	life := warn + fall + campaignStrikeLinger
	g.extraHazardMarkers = append(g.extraHazardMarkers, HazardMarker{
		Kind: hazardMarkerStrike, X: x, Y: landY, Life: life, MaxLife: life, Drop: fall + campaignStrikeLinger,
	})
	g.campaignStrikes = append(g.campaignStrikes, campaignStrike{x: x, landY: landY, startY: startY, delay: warn, ownerID: ownerID, facing: facing})
}

// campaignStrikeStart: campaignStrikeStartAbove over the landing surface, or
// just under the closest platform above it.
func campaignStrikeStart(arena Map, x, landY float64) float64 {
	start := landY - campaignStrikeStartAbove
	for _, r := range arena.Platforms {
		if r.ContainsX(x) && r.Y+r.H <= landY-1 && r.Y+r.H+8 > start {
			start = r.Y + r.H + 8
		}
	}
	return math.Min(start, landY-40)
}

func (g *Game) updateCampaignStrikes(arena Map) {
	out := g.campaignStrikes[:0]
	for _, s := range g.campaignStrikes {
		s.delay--
		if s.delay > 0 {
			out = append(out, s)
			continue
		}
		f := s.facing
		if f == 0 {
			f = 1
		}
		g.bullets = append(g.bullets, Bullet{
			Kind: BulletNormal, X: s.x, Y: s.startY, VY: campaignStrikeFallSpeed,
			Firepower: campaignStrikePush, OwnerID: s.ownerID, Team: campaignEnemyTeam, Facing: f,
			Time: 1, Alpha: 1, TrailWidth: 20, TrailFrame: 1, ShooterID: s.ownerID,
			DevelopedProjectile: DevelopedProjectileStrike, ExplosionRadius: campaignStrikeRadius,
			ExplosionPower: campaignStrikePower, Lifetime: 120,
		})
	}
	g.campaignStrikes = out
}

// --- machine-gun rounds and returned rockets --------------------------------

// updateCampaignBossBullets runs in the hit phase right before
// updateBullets: big-enemy machine-gun rounds stop on platforms (cover
// works), and a gunship rocket that a player batted or blew back homes on
// the gunship and hits it for heliRocketDamage.
func (g *Game) updateCampaignBossBullets(targets []*Player) {
	if len(g.campaignEnemies) == 0 {
		return
	}
	for i := range g.bullets {
		bl := &g.bullets[i]
		if bl.Dead || bl.bossShot == 0 {
			continue
		}
		switch bl.bossShot {
		case bossShotRound:
			nx, ny := bl.X+bl.VX, bl.Y+bl.VY
			if ny > 1000 {
				bl.Dead = true
				continue
			}
			if bossRoundHitsTarget(bl, nx, ny, targets) {
				continue // the hit comes first (updateBullets)
			}
			for _, r := range g.arena.Platforms {
				if segmentIntersectsRect(bl.X, bl.Y, nx, ny, r) {
					bl.Dead = true
					spawnMolotovSparks(&g.dynamiteFX, nx, math.Min(ny, r.Y), 1)
					break
				}
			}
		case bossShotRocket:
			if bl.Team == campaignEnemyTeam {
				continue // still the gunship's own rocket
			}
			g.steerReturnedRocket(bl)
		}
	}
}

func bossRoundHitsTarget(bl *Bullet, nx, ny float64, targets []*Player) bool {
	for _, p := range targets {
		if bulletCanHitPlayer(bl, p) && p.Hitbox().Contains(nx, ny) {
			return true
		}
	}
	return false
}

// steerReturnedRocket turns a sent-back rocket toward the gunship that fired
// it and detonates it on the hull.
func (g *Game) steerReturnedRocket(bl *Bullet) {
	e := g.campaignEnemyByID(bl.bossFrom)
	if e == nil || !e.hittable() {
		return
	}
	cx, cy := heliCenter(&e.Body)
	dx, dy := cx-bl.X, cy-bl.Y
	d := math.Hypot(dx, dy)
	speed := math.Max(heliReturnSpeed, math.Hypot(bl.VX, bl.VY))
	if d > 0.001 {
		bl.VX, bl.VY = dx/d*speed, dy/d*speed
	}
	bl.GuidanceLost = true
	if bl.VX != 0 {
		bl.Facing = int(sign(bl.VX))
	}
	hb := e.Body.Hitbox()
	hb.X -= 10
	hb.Y -= 10
	hb.W += 20
	hb.H += 20
	if d > speed+40 && !segmentIntersectsRect(bl.X, bl.Y, bl.X+bl.VX, bl.Y+bl.VY, hb) {
		return
	}
	bl.Dead = true
	g.explosions = append(g.explosions, Explosion{X: cx, Y: cy, Life: 18})
	g.blastFX = append(g.blastFX, spawnGrenadeBlastEffects(cx, cy)...)
	g.damageBigEnemy(e, heliRocketDamage, bl.OwnerID)
	e.HitFlash = 10
}

// campaignEnemySnapshotMax: twice the trim cap (netSnapshotListCaps).
const campaignEnemySnapshotMax = 64

// maxState is the highest State a kind uses (the shield-bearer counts its
// shield turn in State).
func (k campaignEnemyKind) maxState() int {
	switch k {
	case campaignEnemyShieldBearer:
		return shieldBearerTurnFrames
	case campaignEnemyMech:
		return mechStateMeltdown
	case campaignEnemyHelicopter:
		return heliStateCrash
	}
	return 0
}

// validateCampaignSnapshot bounds the campaign enemies and the boss banner
// a host sends (netwire_secure.go).
func validateCampaignSnapshot(s netSnapshot) error {
	if len(s.BossBanner) > 128 || !utf8.ValidString(s.BossBanner) || s.BossBannerFrame < 0 || s.BossBannerFrame > bossBannerFrames {
		return errors.New("invalid boss banner in host snapshot")
	}
	finite := func(v ...float64) bool {
		for _, x := range v {
			if math.IsNaN(x) || math.IsInf(x, 0) || math.Abs(x) > 1e6 {
				return false
			}
		}
		return true
	}
	if len(s.CampaignEnemies) > campaignEnemySnapshotMax {
		return errors.New("host snapshot campaign enemies exceed safe limit")
	}
	for i := range s.CampaignEnemies {
		e := &s.CampaignEnemies[i]
		switch {
		case e.Kind <= int(campaignEnemyNone) || e.Kind >= int(campaignEnemyKindCount):
			return errors.New("invalid campaign enemy kind in host snapshot")
		case e.MaxHP < 0 || e.MaxHP > 100000 || e.HP < 0 || e.HP > e.MaxHP:
			return errors.New("invalid campaign enemy HP in host snapshot")
		case e.Phase < 0 || e.Phase > 2 || e.State < 0 || e.State > campaignEnemyKind(e.Kind).maxState():
			return errors.New("invalid campaign enemy state in host snapshot")
		case e.Hooked < 0 || e.Hooked > heliHookFrames:
			return errors.New("invalid campaign enemy hook in host snapshot")
		case len(e.Title) > 64 || !utf8.ValidString(e.Title) || len(e.Body.Name) > 96 || !utf8.ValidString(e.Body.Name):
			return errors.New("invalid campaign enemy name in host snapshot")
		case !finite(e.AimX0, e.AimX1, e.AimY, e.Tilt, e.Body.X, e.Body.Y):
			return errors.New("invalid campaign enemy position in host snapshot")
		}
		if err := validateArsenal2Player(&e.Body); err != nil {
			return err
		}
	}
	return nil
}
