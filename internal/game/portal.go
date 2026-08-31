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
	dLeft := math.Abs(hitX - surface.X)
	dRight := math.Abs(surface.X + surface.W - hitX)
	dTop := math.Abs(hitY - surface.Y)
	dBottom := math.Abs(surface.Y + surface.H - hitY)
	nx, ny := -1.0, 0.0
	best := dLeft
	if dRight < best {
		best, nx, ny = dRight, 1, 0
	}
	if dTop < best {
		best, nx, ny = dTop, 0, -1
	}
	if dBottom < best {
		nx, ny = 0, 1
	}
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
	clamp := func(value, low, high float64) float64 {
		return math.Max(low, math.Min(high, value))
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
		consider(surface.X, clamp(b.Y, surface.Y, surface.Y+surface.H), -1, 0)
		consider(surface.X+surface.W, clamp(b.Y, surface.Y, surface.Y+surface.H), 1, 0)
		consider(clamp(b.X, surface.X, surface.X+surface.W), surface.Y, 0, -1)
		consider(clamp(b.X, surface.X, surface.X+surface.W), surface.Y+surface.H, 0, 1)
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
	if exit, ok := portalPair(available, entry); ok {
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
		target.VX += 24 * float64(direction)
		target.VY -= 8
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
