package game

import "math"

// Gameplay of the second added arsenal: bat swings, the leaf blower cone,
// the harpoon reel and AKIMBO shot placement. The bat and the blower only
// mark the shot in the player update (fireMeleeSwing / fireLeafBlower); the
// push itself is applied in the hit phase (applyArsenal2Hits), inside the
// targets' knockback window, so HEAVY, LIGHT, ANCHOR, SHIELD and campaign /
// zombie knockback scales apply exactly like they do to bullets.

// weaponReadyToFire is the source "gun stopped on its rest frame" test.
// AKIMBO lets the second handgun fire while the first one's shot timeline
// (frames 2-9) still plays; a reload (frame 10+) still blocks.
func weaponReadyToFire(p *Player) bool {
	w := &p.Weapon
	if w.Frame == 1 {
		return true
	}
	return akimboActive(p) && w.Def.Number <= 8 && w.Frame >= 1 && w.Frame < 9
}

// shiftAkimboShot moves the projectiles and the flash of an off-hand AKIMBO
// shot to the second gun (a little behind and below the main one).
func shiftAkimboShot(p *Player, offHand bool, shots []Bullet, flashes []MuzzleFlash) {
	if !offHand {
		return
	}
	mini := p.MiniMulti
	if mini <= 0 {
		mini = 1
	}
	dx := -akimboOffHandBack * mini * float64(p.Facing)
	dy := akimboOffHandDrop * mini
	for i := range shots {
		shots[i].X += dx
		shots[i].Y += dy
	}
	for i := range flashes {
		flashes[i].X += dx
		flashes[i].Y += dy
	}
}

// Off-hand gun offset (world px at scale 1) for AKIMBO shots and art.
const (
	akimboOffHandBack = 5.0
	akimboOffHandDrop = 8.0
)

func fireMeleeSwing(p *Player) {
	p.MeleeSwing = batSwingFrames
	p.MeleeHitPending = true
}

func fireLeafBlower(p *Player) {
	p.BlowerTime = blowerVisualTicks
	p.BlowerPulse = true
}

// arsenal2CanAffect is bulletCanHitPlayer for the non-projectile weapons.
func arsenal2CanAffect(owner *Player, t *Player) bool {
	if owner == nil || t == nil || t == owner || !t.Active {
		return false
	}
	if t.scoreOwnerID() == owner.scoreOwnerID() {
		return false
	}
	return owner.Team == 0 || t.Team != owner.Team
}

// applyArsenal2Hits runs in the bullet hit phase, before updateBullets.
func (g *Game) applyArsenal2Hits(targets []*Player) {
	double := g.customModifierActive(customModifierDoubleKnockback)
	for _, p := range g.players {
		if p == nil {
			continue
		}
		if p.MeleeHitPending {
			p.MeleeHitPending = false
			if p.Active {
				g.applyMeleeSwing(p, targets, double)
			}
		}
		if p.BlowerPulse {
			p.BlowerPulse = false
			if p.Active {
				g.applyLeafBlower(p, targets, double)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Baseball bat

// meleeHitbox is the bat's reach in world space.
func meleeHitbox(p *Player) Rect {
	s := playerRenderScale(p) / 0.8
	f := float64(p.Facing)
	if f == 0 {
		f = 1
	}
	x0, x1 := p.X-batReachBack*s*f, p.X+batReachFront*s*f
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	return Rect{X: x0, Y: p.Y - batReachTop*s, W: x1 - x0, H: batReachTop*s - 2}
}

func (g *Game) applyMeleeSwing(p *Player, targets []*Player, double bool) {
	box := meleeHitbox(p)
	f := float64(p.Facing)
	if f == 0 {
		f = 1
	}
	owner := p.scoreOwnerID()
	power := p.Weapon.Def.Firepower
	if !isMeleeWeapon(p.Weapon.Def.Number) || power <= 0 {
		power = batPush
	}
	if double {
		power *= 2
	}
	s := playerRenderScale(p) / 0.8
	for _, t := range targets {
		if !arsenal2CanAffect(p, t) || !rectsOverlap(box, t.Hitbox()) {
			continue
		}
		ts := playerRenderScale(t) / 0.8
		if !arsenal2LineClear(g.arena, p.X, p.Y-35*s, t.X, t.Y-35*ts) {
			continue // never through a platform
		}
		push := power
		if t.ShieldTime == 0 && t.DamageMulti > 0 {
			push *= t.DamageMulti
		}
		if t.ShieldTime > 0 {
			push *= 0.33
		}
		if ballisticShieldCovers(t, f, 0) {
			push *= batShieldScale
			t.ShieldGearSpark = ballisticShieldSparkFrames
		}
		if e := g.campaignEnemyForBody(t); e != nil && e.shieldUp() && f*float64(e.Body.Facing) < 0 {
			// The shield-bearer's riot plate takes the swing: half push, no
			// hit (no HP loss).
			t.VX += push * batShieldScale * f
			e.ShieldSpark = 8
			continue
		}
		t.VX += push * f
		t.VY -= batLift
		t.HitNumber++
		t.HitTimer = 0
		t.LastHitBy = owner
		t.HitByGrenade = false
		if t.AISpecial == sourceAISpecialGM2Invisible {
			t.InvisibleTime = -30
		}
		incrementOwnerHit(targets, owner)
		if z := g.zombieForBody(t); z != nil {
			// Bats are the zombie-mode melee: a wave-1 walker drops in one hit.
			z.applyDamage(batZombieBonus, owner)
		}
	}
	// Bat grenades (and launcher grenades) back the way the swing goes.
	reach := box
	reach.X -= 10
	reach.W += 20
	reach.Y -= 10
	reach.H += 20
	for i := range g.grenades {
		gr := &g.grenades[i]
		if gr.Dead || !reach.Contains(gr.X, gr.Y) {
			continue
		}
		gr.VX = f * batGrenadeHitSpeed
		gr.VY = -batGrenadeHitLift
		gr.OwnerID = owner
		gr.FreePass = true
	}
	for i := range g.bullets {
		b := &g.bullets[i]
		if b.Dead || !reach.Contains(b.X, b.Y) || !arsenal2Deflectable(b) || b.VX*f >= 0 {
			continue
		}
		reflectProjectile(b, p)
	}
}

// ---------------------------------------------------------------------------
// Leaf blower

// blowerNozzle is the world position of the nozzle mouth.
func blowerNozzle(p *Player) (x, y, f float64) {
	f = float64(p.Facing)
	if f == 0 {
		f = 1
	}
	mini := p.MiniMulti
	if mini <= 0 {
		mini = 1
	}
	return p.X + 52*mini*f, p.Y - 36*mini, f
}

// blowerConeFactor reports whether (x, y) is inside the cone and how close to
// the nozzle it is (k = 1 at the nozzle, 0 at the end of the cone). slack
// widens the cone (target half-height).
func blowerConeFactor(ox, oy, f, x, y, slack float64) (float64, bool) {
	dx := (x - ox) * f
	if dx < -18 || dx > blowerRange {
		return 0, false
	}
	half := blowerNearHalfWidth + math.Max(dx, 0)*blowerHalfAngleTan + slack
	if math.Abs(y-oy) > half {
		return 0, false
	}
	return 1 - math.Max(dx, 0)/blowerRange, true
}

// blowerPushAt is the per-tick push of the blower at cone factor k.
func blowerPushAt(k float64) float64 {
	return blowerPushFar + (blowerPushNear-blowerPushFar)*k
}

func (g *Game) applyLeafBlower(p *Player, targets []*Player, double bool) {
	ox, oy, f := blowerNozzle(p)
	owner := p.scoreOwnerID()
	for _, t := range targets {
		if !arsenal2CanAffect(p, t) {
			continue
		}
		s := playerRenderScale(t) / 0.8
		k, ok := blowerConeFactor(ox, oy, f, t.X, t.Y-35*s, 26*s)
		if !ok || !arsenal2LineClear(g.arena, ox, oy, t.X, t.Y-35*s) {
			continue
		}
		push := blowerPushAt(k)
		if double {
			push *= 2 // DOUBLE KNOCKBACK modifier
		}
		if e := g.campaignEnemyForBody(t); e != nil && e.shieldUp() && f*float64(e.Body.Facing) < 0 {
			push *= batShieldScale // the riot plate catches the wind
		}
		if t.ShieldTime == 0 && t.DamageMulti > 0 {
			push *= t.DamageMulti
		}
		if t.ShieldTime > 0 {
			push *= 0.33
		}
		dvx, dvy := push*f, -blowerLift*k
		t.VX += dvx
		t.VY += dvy
		// Wind credits a ring-out but is not a "hit" (no hit sound / counter,
		// and the one-hit SHIELD ignores it: below shieldMinImpulse).
		t.LastHitBy = owner
		t.HitByGrenade = false
		if z := g.zombieForBody(t); z != nil {
			z.windVX += dvx
			z.windVY += dvy
		}
		if e := g.campaignEnemyForBody(t); e != nil {
			e.windVX += dvx
			e.windVY += dvy
		}
	}
	for i := range g.grenades {
		gr := &g.grenades[i]
		if gr.Dead {
			continue
		}
		k, ok := blowerConeFactor(ox, oy, f, gr.X, gr.Y, 6)
		if !ok || !arsenal2LineClear(g.arena, ox, oy, gr.X, gr.Y) {
			continue
		}
		w := 0.4 + 0.6*k
		gr.VX += blowerGrenadePush * f * w
		gr.VY -= blowerGrenadeLift * w
		if gr.HitGround && gr.VY < 0 {
			gr.HitGround = false
		}
	}
	for i := range g.bullets {
		b := &g.bullets[i]
		if b.Dead || !arsenal2Deflectable(b) {
			continue
		}
		k, ok := blowerConeFactor(ox, oy, f, b.X, b.Y, 10)
		if !ok || !arsenal2LineClear(g.arena, ox, oy, b.X, b.Y) {
			continue
		}
		if b.DevelopedProjectile == DevelopedProjectileGrenade {
			w := 0.4 + 0.6*k
			b.VX += blowerGrenadePush * f * w
			b.VY -= blowerGrenadeLift * w
			continue
		}
		if b.VX*f < 0 {
			reflectProjectile(b, p)
		}
	}
}

// arsenal2Deflectable: rockets and launcher grenades can be blown or batted
// back; bullets, bolts, flames and HIMARS strikes cannot.
func arsenal2Deflectable(b *Bullet) bool {
	switch b.DevelopedProjectile {
	case DevelopedProjectileRocket, DevelopedProjectileArcRocket, DevelopedProjectileTopAttack, DevelopedProjectileGrenade:
		return true
	}
	return false
}

// reflectProjectile turns an incoming rocket around; the deflector owns it.
func reflectProjectile(b *Bullet, by *Player) {
	f := float64(by.Facing)
	if f == 0 {
		f = 1
	}
	speed := math.Max(math.Abs(b.VX), 8)
	b.VX = speed * f
	if b.VY > 0 {
		b.VY *= 0.5
	}
	b.Facing = by.Facing
	b.OwnerID = by.scoreOwnerID()
	b.Team = by.Team
	b.ShooterID = by.ID
	b.GuidanceLost = true
}

// ---------------------------------------------------------------------------
// Harpoon

// harpoonShooter finds the player at the shooter end of the rope.
func harpoonShooter(b *Bullet, players []*Player) *Player {
	if b.ShooterID < 0 {
		return nil // the shooter died: the rope was cut
	}
	var owner *Player
	for _, p := range players {
		if p == nil || !p.Active {
			continue
		}
		if b.ShooterID != 0 && p.ID == b.ShooterID {
			return p
		}
		if owner == nil && p.scoreOwnerID() == b.OwnerID && !p.IsDouble {
			owner = p
		}
	}
	return owner
}

// harpoonPullImpulse is the reel's horizontal impulse for a victim dist px
// from the shooter: enough to slide it to about harpoonStopPx away.
func harpoonPullImpulse(dist float64) float64 {
	if dist <= harpoonStopPx {
		return 0
	}
	v := (dist - harpoonStopPx) * (1 - Friction)
	return math.Max(harpoonPullMin, math.Min(harpoonPullMax, v))
}

// hitPlayerWithHarpoon hooks t and reels it toward the shooter. A BALLISTIC
// SHIELD does not help: the hook bites into the plate (that is its counter).
func hitPlayerWithHarpoon(b *Bullet, t *Player, players []*Player) {
	shooter := harpoonShooter(b, players)
	mult := 1.0
	if t.ShieldTime == 0 && t.DamageMulti > 0 {
		mult = t.DamageMulti
	}
	if t.ShieldTime > 0 {
		mult = 0.33
	}
	// The hook's own tug along its flight, then the reel.
	t.VX += b.Firepower * float64(b.Facing) * mult
	if shooter != nil {
		dx := shooter.X - t.X
		dir := sign(dx)
		t.VX += dir * harpoonPullImpulse(math.Abs(dx)) * mult * harpoonReelScale(b)
		lift := harpoonLift
		if climb := t.Y - shooter.Y; climb > 40 {
			lift += math.Min(harpoonClimbMax, climb*0.04)
		}
		if t.VY > -lift*mult {
			t.VY = -lift * mult
		}
		t.HarpoonTether = harpoonTetherFrames
		t.HarpoonTetherID = shooter.ID
	}
	t.HitNumber++
	t.HitTimer = 0
	t.LastHitBy = b.OwnerID
	t.HitByGrenade = false
	if t.AISpecial == sourceAISpecialGM2Invisible {
		t.InvisibleTime = -30
	}
	incrementOwnerHit(players, b.OwnerID)
	b.Dead = true
}

// zombieForBody maps a hit target back to its zombie (nil for players).
func (g *Game) zombieForBody(t *Player) *Zombie {
	if g == nil || g.zombie == nil || t == nil {
		return nil
	}
	for _, z := range g.zombie.zombies {
		if z != nil && &z.body == t {
			return z
		}
	}
	return nil
}

// zombieClawPlayer is a zombie claw / leap hit. A BALLISTIC SHIELD on the
// side the claw comes from halves the damage and takes most of the push;
// boss slams and exploder blasts (zombieHurtPlayer directly) are not covered.
func (g *Game) zombieClawPlayer(t *Player, damage int, pushX, lift float64, attackerID int) bool {
	if t != nil && pushX != 0 && ballisticShieldCovers(t, pushX, 0) {
		damage = int(math.Ceil(float64(damage) * ballisticShieldClawDamage))
		pushX *= ballisticShieldClawPush
		lift *= ballisticShieldClawPush
		t.ShieldGearSpark = ballisticShieldSparkFrames
	}
	return g.zombieHurtPlayer(t, damage, pushX, lift, attackerID)
}

// harpoonReelScale: the reel follows the hook's firepower relative to the
// catalog value, so the DOUBLE KNOCKBACK modifier and campaign FirepowerMulti
// scale the pull like they scale every bullet.
func harpoonReelScale(b *Bullet) float64 {
	if b == nil || b.Firepower <= 0 {
		return 1
	}
	return math.Max(0.25, math.Min(4, b.Firepower/harpoonDirectPush))
}

// arsenal2LineClear: no platform between the two points (bat and blower
// never act through level geometry).
func arsenal2LineClear(arena Map, x0, y0, x1, y1 float64) bool {
	for _, r := range arena.Platforms {
		if segmentIntersectsRect(x0, y0, x1, y1, r) {
			return false
		}
	}
	return true
}

// campaignEnemyForBody maps a hit target back to its campaign enemy.
func (g *Game) campaignEnemyForBody(t *Player) *CampaignEnemy {
	if g == nil || t == nil {
		return nil
	}
	for _, e := range g.campaignEnemies {
		if e != nil && &e.Body == t {
			return e
		}
	}
	return nil
}

// dropHarpoonRopesOf cuts every rope of a shooter who just died: hooked
// victims lose their tether and harpoons in flight no longer reel.
func (g *Game) dropHarpoonRopesOf(shooterID int) {
	if g == nil || shooterID == 0 {
		return
	}
	for _, t := range g.hitTargets() {
		if t != nil && t.HarpoonTetherID == shooterID {
			t.HarpoonTether, t.HarpoonTetherID = 0, 0
		}
	}
	for i := range g.bullets {
		if b := &g.bullets[i]; b.DevelopedProjectile == DevelopedProjectileHarpoon && b.ShooterID == shooterID {
			b.ShooterID = -1
		}
	}
}
