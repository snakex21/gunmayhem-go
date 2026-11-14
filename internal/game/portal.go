package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	PortalBlue = iota + 1
	PortalOrange
)

type Portal struct {
	X, Y             float64
	NormalX, NormalY float64
	OwnerID          int
	Color            int
	Active           bool
}

func placePortalFromSegment(portals *[]Portal, b *Bullet, x0, y0, x1, y1 float64, surface Rect) {
	if portals == nil || b == nil {
		return
	}
	distance := math.Max(math.Abs(x1-x0), math.Abs(y1-y0))
	steps := int(math.Ceil(distance / 2))
	if steps < 1 {
		steps = 1
	}
	hitX, hitY := x1, y1
	for step := 0; step <= steps; step++ {
		t := float64(step) / float64(steps)
		x, y := x0+(x1-x0)*t, y0+(y1-y0)*t
		if surface.Contains(x, y) {
			hitX, hitY = x, y
			break
		}
	}
	// Portals only ever open on platform tops. A side or underside portal at
	// the arena edge turned every Portal Gun hit into a one-hit ring-out (the
	// victim was thrown out sideways past the edge), so a shot that strikes a
	// side or the underside snaps to the top of that platform instead.
	nx, ny := 0.0, -1.0
	hitX, hitY = portalTopX(surface, hitX), surface.Y
	// Keep the portal just outside the solid rectangle, so crossing is detected
	// before ordinary platform collision resolves the player.
	hitX += nx * 3
	hitY += ny * 3
	colorID := b.PortalColor
	if colorID != PortalOrange {
		colorID = PortalBlue
	}
	portal := Portal{X: hitX, Y: hitY, NormalX: nx, NormalY: ny, OwnerID: b.OwnerID, Color: colorID, Active: true}
	storePortal(portals, portal)
}

func storePortal(portals *[]Portal, portal Portal) {
	if portals == nil {
		return
	}
	for i := range *portals {
		if (*portals)[i].OwnerID == portal.OwnerID && (*portals)[i].Color == portal.Color {
			(*portals)[i] = portal
			return
		}
	}
	*portals = append(*portals, portal)
}

func placePortalOnNearestSurface(portals *[]Portal, b *Bullet, arena Map) bool {
	if portals == nil || b == nil || len(arena.Platforms) == 0 {
		return false
	}
	type candidate struct {
		x, y, nx, ny, distance float64
	}
	best := candidate{distance: math.MaxFloat64}
	consider := func(x, y, nx, ny float64) {
		distance := math.Hypot(b.X-x, b.Y-y)
		if distance < best.distance {
			best = candidate{x: x, y: y, nx: nx, ny: ny, distance: distance}
		}
	}
	for _, surface := range arena.Platforms {
		// Tops only (see placePortalFromSegment).
		consider(portalTopX(surface, b.X), surface.Y, 0, -1)
	}
	if best.distance == math.MaxFloat64 {
		return false
	}
	colorID := b.PortalColor
	if colorID != PortalOrange {
		colorID = PortalBlue
	}
	storePortal(portals, Portal{
		X: best.x + best.nx*3, Y: best.y + best.ny*3,
		NormalX: best.nx, NormalY: best.ny,
		OwnerID: b.OwnerID, Color: colorID, Active: true,
	})
	return true
}

// portalTopInset keeps a portal this far inside the ends of its platform.
const portalTopInset = 20.0

// portalTopX clamps x onto the top of surface, portalTopInset inside its ends
// (the middle of a platform narrower than two insets).
func portalTopX(surface Rect, x float64) float64 {
	if surface.W <= 2*portalTopInset {
		return surface.X + surface.W/2
	}
	return math.Max(surface.X+portalTopInset, math.Min(surface.X+surface.W-portalTopInset, x))
}

// portalUnpairedPush is the Portal Gun's pulse when its owner has no
// complete portal pair (or the victim is shielded).
const portalUnpairedPush = 20.0

// portalShielded: the shield powerup or a SHIELD ability that still has its
// block (or spent it this very tick) stops the teleport-hit.
func portalShielded(p *Player) bool {
	if p == nil {
		return false
	}
	if p.ShieldTime > 0 || abilityShieldReady(p) {
		return true
	}
	return p.AbilityNumber == abilityShield && p.AbilityShieldUsed && p.AbilityShieldBreak == shieldBreakFrames
}

// portalOwnerHoldsGun reports whether some living player (or body) scoring
// as ownerID still holds the Portal Gun.
func portalOwnerHoldsGun(ownerID int, players []*Player) bool {
	for _, p := range players {
		if p == nil || !p.Active || p.scoreOwnerID() != ownerID {
			continue
		}
		if spec, ok := developedWeaponSpec(p.Weapon.Def.Number); ok && spec.Projectile == DevelopedProjectilePortal {
			return true
		}
	}
	return false
}

// prunePortals removes the portals of every owner who no longer holds the
// Portal Gun (it ran empty, was swapped for another weapon, or the owner
// died), so abandoned portals cannot linger as death traps. Round reset
// clears all portals separately.
func prunePortals(portals []Portal, players []*Player) []Portal {
	out := portals[:0]
	for _, portal := range portals {
		if portalOwnerHoldsGun(portal.OwnerID, players) {
			out = append(out, portal)
		}
	}
	return out
}

func portalPair(portals []Portal, entry Portal) (Portal, bool) {
	want := PortalBlue
	if entry.Color == PortalBlue {
		want = PortalOrange
	}
	for _, portal := range portals {
		if portal.Active && portal.OwnerID == entry.OwnerID && portal.Color == want {
			return portal, true
		}
	}
	return Portal{}, false
}

func hitPortalPlayerAlongSegment(b *Bullet, x0, y0, x1, y1 float64, players []*Player, portals *[]Portal) bool {
	if b == nil {
		return false
	}
	distance := math.Max(math.Abs(x1-x0), math.Abs(y1-y0))
	steps := int(math.Ceil(distance / 4))
	if steps < 1 {
		steps = 1
	}
	for step := 1; step <= steps; step++ {
		t := float64(step) / float64(steps)
		b.X, b.Y = x0+(x1-x0)*t, y0+(y1-y0)*t
		for _, target := range players {
			if target == nil || !bulletCanHitPlayer(b, target) || !target.Hitbox().Contains(b.X, b.Y) {
				continue
			}
			portalCombatHit(b, target, portals)
			incrementOwnerHit(players, b.OwnerID)
			return true
		}
	}
	return false
}

func portalCombatHit(b *Bullet, target *Player, portals *[]Portal) {
	if b == nil || target == nil {
		return
	}
	var available []Portal
	if portals != nil {
		available = *portals
	}
	entry := Portal{OwnerID: b.OwnerID, Color: b.PortalColor, Active: true}
	exit, paired := portalPair(available, entry)
	if paired && !portalShielded(target) {
		target.X = exit.X + exit.NormalX*58
		target.Y = exit.Y + 28 + exit.NormalY*58
		target.VX = exit.NormalX * 19
		target.VY = exit.NormalY * 19
		target.FreePass = true
		target.PortalCooldown = 26
	} else {
		// Without a completed pair the gun is still useful in combat: the unstable
		// portal pulse throws the victim in the firing direction and slightly up.
		direction := b.Facing
		if direction == 0 {
			direction = 1
		}
		// A shielded victim is never teleported: the shield powerup takes a
		// fifth of the pulse, and the SHIELD ability's knockback window
		// blocks it outright.
		push := portalUnpairedPush
		lift := 8.0
		if target.ShieldTime > 0 {
			push *= 0.2
			lift *= 0.2
		}
		target.VX += push * float64(direction)
		target.VY -= lift
	}
	target.HitNumber++
	target.HitTimer = 0
	target.LastHitBy = b.OwnerID
	target.HitByGrenade = false
	b.Dead = true
}

func portalContact(x, y, vx, vy float64, portal Portal) bool {
	if !portal.Active {
		return false
	}
	dx, dy := x-portal.X, y-portal.Y
	normalDistance := math.Abs(dx*portal.NormalX + dy*portal.NormalY)
	tangentDistance := math.Abs(dx*(-portal.NormalY) + dy*portal.NormalX)
	return normalDistance <= 46 && tangentDistance <= 41
}

func updatePortalTravel(portals []Portal, players []*Player, grenades []Grenade) {
	for _, p := range players {
		if p == nil || !p.Active || p.PortalCooldown > 0 {
			continue
		}
		for _, entry := range portals {
			exit, ok := portalPair(portals, entry)
			if !ok || !portalContact(p.X, p.Y-28, p.VX, p.VY, entry) {
				continue
			}
			speed := math.Max(10, math.Hypot(p.VX, p.VY))
			tangentSpeed := p.VX*(-entry.NormalY) + p.VY*entry.NormalX
			p.X = exit.X + exit.NormalX*54
			p.Y = exit.Y + 28 + exit.NormalY*54
			p.VX = exit.NormalX*speed + (-exit.NormalY)*tangentSpeed*.35
			p.VY = exit.NormalY*speed + exit.NormalX*tangentSpeed*.35
			p.FreePass = true
			p.PortalCooldown = 24
			break
		}
	}
	for i := range grenades {
		g := &grenades[i]
		if g.Dead || g.PortalCooldown > 0 {
			if g.PortalCooldown > 0 {
				g.PortalCooldown--
			}
			continue
		}
		for _, entry := range portals {
			exit, ok := portalPair(portals, entry)
			// Moving objects still have to enter the face of the portal. Players do
			// not use this check because platform collision may already have reset
			// their velocity to zero before the portal system runs.
			movingIntoPortal := g.VX*entry.NormalX+g.VY*entry.NormalY < -.25
			if !ok || !movingIntoPortal || !portalContact(g.X, g.Y, g.VX, g.VY, entry) {
				continue
			}
			speed := math.Max(8, math.Hypot(g.VX, g.VY))
			g.X, g.Y = exit.X+exit.NormalX*18, exit.Y+exit.NormalY*18
			g.VX, g.VY = exit.NormalX*speed, exit.NormalY*speed
			g.FreePass = true
			g.PortalCooldown = 18
			break
		}
	}
}

func (g *Game) drawPortals(screen *ebiten.Image) {
	for _, portal := range g.portals {
		if !portal.Active {
			continue
		}
		outer := color.NRGBA{R: 0x18, G: 0x9f, B: 0xff, A: 0xd8}
		inner := color.NRGBA{R: 0x8b, G: 0xe8, B: 0xff, A: 0xf0}
		if portal.Color == PortalOrange {
			outer = color.NRGBA{R: 0xff, G: 0x72, B: 0x10, A: 0xd8}
			inner = color.NRGBA{R: 0xff, G: 0xd0, B: 0x4c, A: 0xf0}
		}
		img := ebiten.NewImage(26, 78)
		vector.DrawFilledCircle(img, 13, 13, 13, outer, true)
		vector.DrawFilledCircle(img, 13, 65, 13, outer, true)
		vector.DrawFilledRect(img, 0, 13, 26, 52, outer, true)
		vector.DrawFilledCircle(img, 13, 13, 7, color.NRGBA{R: 0x08, G: 0x0a, B: 0x11, A: 0xf0}, true)
		vector.DrawFilledCircle(img, 13, 65, 7, color.NRGBA{R: 0x08, G: 0x0a, B: 0x11, A: 0xf0}, true)
		vector.DrawFilledRect(img, 6, 13, 14, 52, color.NRGBA{R: 0x08, G: 0x0a, B: 0x11, A: 0xf0}, true)
		vector.StrokeLine(img, 4, 13, 4, 65, 2, inner, true)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-13, -39)
		if math.Abs(portal.NormalY) > .5 {
			op.GeoM.Rotate(math.Pi / 2)
		}
		op.GeoM.Translate(g.worldX(portal.X), g.worldY(portal.Y))
		screen.DrawImage(img, op)
	}
}
