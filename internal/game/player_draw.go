package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func (g *Game) drawPlayer(screen *ebiten.Image, p *Player) {
	colorIndex, useGM2Color := resolvedPlayerColor(p.PlayerColor, p.UseGM2Cosmetics)
	if useGM2Color {
		g.assets.EnsureGM2PlayerColor(colorIndex)
	} else {
		g.assets.EnsurePlayerColor(colorIndex)
	}
	// Flash layer order, from back to front: off-hand, back leg, body,
	// eyes/head details, front leg, gun, front hand. The old flattened player
	// bitmap is no longer used, so changing weapons no longer leaves an M1911
	// baked into the character.
	var hand *SourceRaster
	if useGM2Color {
		hand = g.assets.GM2HandColors[colorIndex]
		if hand == nil {
			g.assets.EnsureGM2PlayerColor(1)
			hand = g.assets.GM2HandColors[1]
		}
	} else {
		hand = g.assets.HandColors[colorIndex]
		if hand == nil {
			hand = g.assets.HandColors[0]
		}
	}
	if akimboActive(p) {
		// AKIMBO: the off hand holds the second handgun.
		g.drawAkimboOffHandGun(screen, p)
		_, _, hx, hy := akimboOffHandLocal(p)
		drawPlayerPartAlpha(screen, hand, p, hx, hy, p.VisualHand2Alpha)
	} else {
		drawPlayerPartAlpha(screen, hand, p,
			p.VisualHand2X+p.VisualHand2ChildX,
			p.VisualHand2Y+p.VisualHand2ChildY,
			p.VisualHand2Alpha,
		)
	}

	var leg *SourceRaster
	if useGM2Color {
		leg = g.assets.GM2LegColors[colorIndex]
		if leg == nil {
			g.assets.EnsureGM2PlayerColor(1)
			leg = g.assets.GM2LegColors[1]
		}
	} else {
		leg = g.assets.LegColors[colorIndex]
		if leg == nil {
			leg = g.assets.LegColors[0]
		}
	}
	if len(g.assets.Leg2Timeline) > 0 {
		frame := p.LegFrame2 % len(g.assets.Leg2Timeline)
		drawPlayerLeg(screen, leg, p, g.assets.Leg2Timeline[frame].Matrix,
			9*float64(p.Facing), -5.6, p.VisualLeg2Rotation)
	}

	if p.JetpackAlpha > 0 {
		g.drawPlayerJetpack(screen, p)
	}
	shieldFront := ballisticShieldInFront(p)
	if hasBallisticShield(p) && !shieldFront {
		g.drawBallisticShieldBack(screen, p)
	}
	g.drawPlayerBody(screen, p)
	g.drawPlayerKevlar(screen, p, g.playerTorsoRaster(p))
	// Head gear (goggles / gas mask) replaces the chosen hat at render time
	// only; HatNumber and the saved config stay untouched.
	headgear := playerHeadgear(p)
	// Source hats 8 and 22 swapDepths with eyes and therefore sit behind the
	// eye layer; all other player_hat frames stay in front of it.
	if headgear == 0 && (p.HatNumber == 8 || p.HatNumber == 22) {
		g.drawPlayerHat(screen, p)
	}
	eyes := g.assets.Eyes
	eyeIndex, useGM2Eyes := resolvedPlayerEyes(p.EyeNumber, p.UseGM2Cosmetics)
	if gm2CosmeticsAvailable() && useGM2Eyes {
		frame := eyeIndex - 1
		if frame < 0 {
			frame = 0
		}
		if r := g.assets.EyeFrame(frame); r != nil {
			eyes = r
		}
	}
	drawPlayerPart(screen, eyes, p, 0, p.VisualEyesY)
	if headgear != 0 {
		g.drawPlayerHeadgear(screen, p, headgear, eyes)
	} else if p.HatNumber != 8 && p.HatNumber != 22 {
		g.drawPlayerHat(screen, p)
	}

	if len(g.assets.Leg1Timeline) > 0 {
		frame := p.LegFrame1 % len(g.assets.Leg1Timeline)
		drawPlayerLeg(screen, leg, p, g.assets.Leg1Timeline[frame].Matrix,
			-12*float64(p.Facing), -0.2, p.VisualLeg1Rotation)
	}

	// Symbol688 (`instagib`) is above the front-leg layer and below the gun /
	// front-hand layers in Symbol697.
	g.drawPlayerInstagib(screen, p)
	if shieldFront {
		g.drawBallisticShieldFront(screen, p)
	}
	g.drawGun(screen, p)
	drawPlayerPartAlpha(screen, hand, p,
		p.VisualHand1X+p.VisualHand1ChildX,
		p.VisualHand1Y+p.VisualHand1ChildY,
		p.VisualHand1Alpha,
	)

	g.drawLeafBlowerGust(screen, p)
	g.drawPlayerBurn(screen, p)
	g.drawPlayerShield(screen, p)
	g.drawPlayerAbilityFX(screen, p)
	if g.showCollisions {
		h := p.Hitbox()
		ebitenutil.DrawRect(screen, h.X, h.Y, h.W, h.H, color.RGBA{R: 30, G: 255, B: 120, A: 80})
	}

	// Source nametag (Symbol685 + embedded font + GlowFilter) is ported as its
	// own renderer. Do not substitute ebitenutil.DebugPrint here.
}

func (g *Game) drawPlayerBurn(screen *ebiten.Image, p *Player) {
	if p == nil || p.BurnTime <= 0 || !p.Active {
		return
	}
	scale := playerRenderScale(p)
	pulse := float32((p.BurnTick/3)%3) * .65
	for i := 0; i < 4; i++ {
		x := float32(p.X + float64(i)*9*scale - 13.5*scale)
		y := float32(p.Y - (20+float64((i*7+p.BurnTick)%15))*scale)
		radius := float32(6.0*scale) + pulse
		filledCircle(screen, x, y, radius+2, color.NRGBA{R: 0xff, G: 0x3d, B: 0x08, A: 0x70})
		filledCircle(screen, x, y+1, radius*.62, color.NRGBA{R: 0xff, G: 0xd6, B: 0x34, A: 0xd8})
	}
}

func (g *Game) drawPlayerInstagib(screen *ebiten.Image, p *Player) {
	if p == nil || !p.InstagibPlaying || p.InstagibFrame <= 0 ||
		g.assets.InstagibVictim == nil || p.InstagibFrame >= len(g.assets.InstagibVictimTimeline) {
		return
	}
	visual := g.assets.InstagibVictimTimeline[p.InstagibFrame]
	if !visual.Valid || visual.Alpha <= 0 {
		return
	}

	// Symbol697 keeps `instagib` at body._y + 60 - 2.65 and flips its xscale
	// with facing. Symbol688 then animates the red Symbol687 child for 60 frames.
	raster := g.assets.InstagibVictim
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(raster.Bounds.X, raster.Bounds.Y)
	op.GeoM.Concat(geoMFromXFL(visual.Matrix))
	op.GeoM.Scale(float64(p.Facing), 1)
	op.GeoM.Translate(0, p.VisualBodyY+60-2.65)
	scale := playerRenderScale(p)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(p.X, p.Y)
	alpha := p.Alpha * visual.Alpha
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	screen.DrawImage(raster.Image, op)
}

func (g *Game) drawPlayerShield(screen *ebiten.Image, p *Player) {
	if p == nil || p.ShieldAlpha <= 0 || p.ShieldFrame < 0 || p.ShieldFrame >= len(g.assets.ShieldTimeline) {
		return
	}
	visual := g.assets.ShieldTimeline[p.ShieldFrame]
	if !visual.Valid {
		return
	}
	raster := g.assets.ShieldFrames[p.ShieldChildFrame]
	if raster == nil {
		return
	}

	// Symbol 697 places Symbol 696 at (0,-35). Symbol 696 then applies the
	// current frame matrix/alpha to its nested Symbol 695 pulse.
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(raster.Bounds.X, raster.Bounds.Y)
	child := geoMFromXFL(visual.Matrix)
	op.GeoM.Concat(child)
	op.GeoM.Translate(0, -35)
	scale := playerRenderScale(p)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(p.X, p.Y)
	alpha := p.Alpha * p.ShieldAlpha * visual.Alpha
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	screen.DrawImage(raster.Image, op)
}

func (g *Game) drawPlayerBody(screen *ebiten.Image, p *Player) {
	// Body colour and clothing are resolved independently in the developed
	// editor. This lets a player combine a GM1 hat with a GM2 shirt/colour while
	// GM2 campaign enemies can still force their raw source IDs as one set.
	var torso, head, shirt *SourceRaster
	colorIndex, useGM2Color := resolvedPlayerColor(p.PlayerColor, p.UseGM2Cosmetics)
	shirtIndex, useGM2Shirt := resolvedPlayerShirt(p.ShirtNumber, p.UseGM2Cosmetics)
	if useGM2Color {
		g.assets.EnsureGM2PlayerColor(colorIndex)
		torso = g.assets.GM2TorsoColors[colorIndex]
		head = g.assets.GM2HeadColors[colorIndex]
	} else {
		g.assets.EnsurePlayerColor(colorIndex)
		torso = g.assets.TorsoColors[colorIndex]
		if torso == nil {
			torso = g.assets.TorsoColors[0]
		}
		head = g.assets.HeadColors[colorIndex]
		if head == nil {
			head = g.assets.HeadColors[0]
		}
	}
	shirtFrame := shirtIndex - 1
	if shirtFrame < 0 {
		shirtFrame = 0
	}
	if useGM2Shirt {
		shirt = g.assets.GM2ShirtFrame(shirtFrame)
	} else {
		shirt = g.assets.ShirtFrame(shirtFrame)
	}

	drawPlayerPart(screen, torso, p, 0, p.VisualBodyY+25.3)
	drawPlayerPart(screen, shirt, p, 0, p.VisualBodyY+26.1)

	if head == nil {
		return
	}
	headFrame := SourceTransformFrame{Matrix: xflMatrix{A: 1, D: 1}, Valid: true}
	if p.HeadFrame >= 0 && p.HeadFrame < len(g.assets.HeadTimeline) && g.assets.HeadTimeline[p.HeadFrame].Valid {
		headFrame = g.assets.HeadTimeline[p.HeadFrame]
	}
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(head.Bounds.X, head.Bounds.Y)
	inner := geoMFromXFL(g.assets.HeadInnerMatrix)
	op.GeoM.Concat(inner)
	headMove := geoMFromXFL(headFrame.Matrix)
	op.GeoM.Concat(headMove)
	op.GeoM.Translate(9.4, p.VisualBodyY+4)
	op.GeoM.Scale(float64(p.Facing), 1)
	scale := playerRenderScale(p)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(p.X, p.Y)
	if p.Alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(p.Alpha))
	}
	screen.DrawImage(head.Image, op)
}

func (g *Game) drawPlayerHat(screen *ebiten.Image, p *Player) {
	hatIndex, useGM2Hat := resolvedPlayerHat(p.HatNumber, p.UseGM2Cosmetics)
	frame := hatIndex - 1
	if frame < 0 {
		frame = 0
	}
	var hat *SourceRaster
	if useGM2Hat {
		hat = g.assets.GM2HatFrame(frame)
	} else {
		hat = g.assets.HatFrame(frame)
	}
	if hat == nil {
		return
	}
	headY := 0.0
	if p.HeadFrame >= 0 && p.HeadFrame < len(g.assets.HeadTimeline) && g.assets.HeadTimeline[p.HeadFrame].Valid {
		headY = g.assets.HeadTimeline[p.HeadFrame].Matrix.TY
	}
	// DefineSprite_697 onEnterFrame:
	// player_hat._x = 9.55*facing;
	// player_hat._y = -75 + body.head.head._y + (body._y + 60).
	scale := playerRenderScale(p)
	drawSourceRaster(screen, hat,
		p.X+9.55*scale*float64(p.Facing),
		p.Y+(-75+headY+(p.VisualBodyY+60))*scale,
		scale*float64(p.Facing), scale, p.Alpha,
	)
}

func (g *Game) drawPlayerJetpack(screen *ebiten.Image, p *Player) {
	if g.assets.JetpackBase == nil {
		return
	}
	alpha := p.Alpha * p.JetpackAlpha
	drawSourceRaster(screen, g.assets.JetpackBase,
		p.X,
		p.Y+(p.VisualBodyY+14.6)*playerRenderScale(p),
		playerRenderScale(p)*float64(p.Facing), playerRenderScale(p), alpha,
	)

	if g.assets.JetpackFuel == nil || p.JetFuel <= 0 {
		return
	}
	fuelScale := p.JetFuel / 100
	if fuelScale < 0 {
		fuelScale = 0
	}
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(g.assets.JetpackFuel.Bounds.X, g.assets.JetpackFuel.Bounds.Y)
	op.GeoM.Scale(1, fuelScale)
	child := geoMFromXFL(g.assets.JetpackFuelMatrix)
	op.GeoM.Concat(child)
	op.GeoM.Scale(float64(p.Facing), 1)
	op.GeoM.Translate(0, p.VisualBodyY+14.6)
	scale := playerRenderScale(p)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(p.X, p.Y)
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	screen.DrawImage(g.assets.JetpackFuel.Image, op)
}

func (g *Game) drawGun(screen *ebiten.Image, p *Player) {
	weaponColor := p.PlayerColor
	if source, useGM2 := resolvedPlayerColor(p.PlayerColor, p.UseGM2Cosmetics); useGM2 {
		weaponColor = -source
	}
	gun := g.assets.WeaponFrameForPlayer(p.Weapon.Def.Number, p.Weapon.Frame, weaponColor)
	if gun == nil {
		return
	}
	// Exact hierarchy: gun local transform -> hand1 facing/position -> player
	// 80% scale/position. The gun's current frame comes from its own timeline.
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(gun.Bounds.X, gun.Bounds.Y)
	op.GeoM.Rotate(gunDrawRotation(p) * math.Pi / 180)
	op.GeoM.Translate(p.VisualGunX, p.VisualGunY)
	op.GeoM.Scale(float64(p.Facing), 1)
	op.GeoM.Translate(p.VisualHand1X, p.VisualHand1Y)
	scale := playerRenderScale(p)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(p.X, p.Y)
	gunAlpha := p.Alpha * p.Weapon.Alpha
	if gunAlpha < 1 {
		op.ColorScale.ScaleAlpha(float32(gunAlpha))
	}
	screen.DrawImage(gun.Image, op)
}

func playerRenderScale(p *Player) float64 {
	if p.PlayerScale <= 0 {
		return 0.8
	}
	return p.PlayerScale
}

func drawPlayerPart(screen *ebiten.Image, raster *SourceRaster, p *Player, x, y float64) {
	drawPlayerPartAlpha(screen, raster, p, x, y, 1)
}

func drawPlayerPartAlpha(screen *ebiten.Image, raster *SourceRaster, p *Player, x, y, localAlpha float64) {
	if raster == nil || localAlpha <= 0 {
		return
	}
	scale := playerRenderScale(p)
	facing := float64(p.Facing)
	drawSourceRaster(screen, raster,
		p.X+x*scale*facing,
		p.Y+y*scale,
		scale*facing, scale, p.Alpha*localAlpha,
	)
}

func drawPlayerLeg(screen *ebiten.Image, raster *SourceRaster, p *Player, child xflMatrix, x, y, degrees float64) {
	if raster == nil {
		return
	}

	// Exact hierarchy from XFL:
	// Symbol188 raster -> Symbol282/189 frame matrix -> runtime leg xscale/
	// rotation/x/y from player ActionScript -> player _xscale/_yscale 80%.
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(raster.Bounds.X, raster.Bounds.Y)
	childGeo := geoMFromXFL(child)
	op.GeoM.Concat(childGeo)
	op.GeoM.Scale(float64(p.Facing), 1)
	op.GeoM.Rotate(degrees * math.Pi / 180)
	op.GeoM.Translate(x, y)
	scale := playerRenderScale(p)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(p.X, p.Y)
	if p.Alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(p.Alpha))
	}
	screen.DrawImage(raster.Image, op)
}

func geoMFromXFL(m xflMatrix) ebiten.GeoM {
	var g ebiten.GeoM
	g.SetElement(0, 0, m.A)
	g.SetElement(0, 1, m.C)
	g.SetElement(0, 2, m.TX)
	g.SetElement(1, 0, m.B)
	g.SetElement(1, 1, m.D)
	g.SetElement(1, 2, m.TY)
	return g
}
