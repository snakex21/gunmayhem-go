package game

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Presentation of the utility throwables, blindness, tear gas and the
// head/body perks. Everything is procedural and derived from synced state
// (AreaEffect list, Player.BlindTime/TearGasTime/AbilityNumber), so online
// clients render the same world; only the viewer-specific parts (whiteout,
// thermal view) depend on viewerPlayerID().

var (
	effectPuffOnce   sync.Once
	effectPuffImage  *ebiten.Image
	silhouetteBuffer *ebiten.Image
)

// softPuff is a white radial-gradient disc used for gas, smoke and glows.
func softPuff() *ebiten.Image {
	effectPuffOnce.Do(func() {
		const size = 64
		img := image.NewNRGBA(image.Rect(0, 0, size, size))
		c := float64(size-1) / 2
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				d := math.Hypot(float64(x)-c, float64(y)-c) / c
				if d >= 1 {
					continue
				}
				a := (1 - d) * (1 - d) * (3 - 2*(1-d)) // smooth falloff
				img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(255 * math.Min(1, a*1.25))})
			}
		}
		effectPuffImage = ebiten.NewImageFromImage(img)
	})
	return effectPuffImage
}

func drawPuff(dst *ebiten.Image, x, y, radius float64, r, g, b, a float32) {
	if radius <= 0 || a <= 0 {
		return
	}
	img := softPuff()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-32, -32)
	op.GeoM.Scale(radius/32, radius/32)
	op.GeoM.Translate(x, y)
	op.ColorScale.Scale(r*a, g*a, b*a, a)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(img, op)
}

// effectHash gives deterministic per-cloud puff layouts.
func effectHash(a, b int) int {
	h := (a+1)*73856093 ^ (b+1)*19349663
	if h < 0 {
		h = -h
	}
	return h
}

func (g *Game) playAreaEffectSpawnSounds(before int) {
	for i := before; i < len(g.areaEffects); i++ {
		switch g.areaEffects[i].Kind {
		case areaEffectFlash:
			g.playRandomExplosionSound()
		case areaEffectGas, areaEffectSmoke, areaEffectTear:
			g.playSourceSFX("emp.wav", false)
		}
	}
}

// drawGroundAreaEffects draws the shock gas and tear gas clouds below the
// players (they are thin enough to see people in them).
func (g *Game) drawGroundAreaEffects(screen *ebiten.Image) {
	for i := range g.areaEffects {
		a := &g.areaEffects[i]
		if a.Dead || a.Life <= 0 {
			continue
		}
		switch a.Kind {
		case areaEffectGas:
			g.drawShockCloud(screen, a)
		case areaEffectTear:
			g.drawTearGasCloud(screen, a)
		}
	}
}

func (g *Game) drawShockCloud(screen *ebiten.Image, a *AreaEffect) {
	x, y := g.worldX(a.X), g.worldY(a.Y)
	fade := 1.0
	if a.Life < gasFadeFrames {
		fade = float64(a.Life) / gasFadeFrames
	}
	// Soft pale-cyan body.
	drawPuff(screen, x, y, a.Radius*1.05, 0.75, 0.95, 1, float32(0.22*fade))
	// Outward-travelling pressure rings.
	for k := 0; k < 3; k++ {
		t := math.Mod(float64(a.Frame)/26+float64(k)/3, 1)
		rr := float32(a.Radius * (0.25 + 0.75*t))
		alpha := uint8(150 * (1 - t) * fade)
		vector.StrokeCircle(screen, float32(x), float32(y), rr, 2.5, color.NRGBA{R: 0xc8, G: 0xf4, B: 0xff, A: alpha}, true)
	}
	// Swirl particles drifting outwards.
	for k := 0; k < 10; k++ {
		seed := effectHash(int(a.X), k)
		ang := float64(seed%360)*math.Pi/180 + float64(a.Frame)*0.03
		t := math.Mod(float64(a.Frame+seed%40)/40, 1)
		d := a.Radius * (0.2 + 0.75*t)
		px, py := x+math.Cos(ang)*d, y+math.Sin(ang)*d*0.7
		vector.DrawFilledCircle(screen, float32(px), float32(py), 2.2, color.NRGBA{R: 0xe8, G: 0xff, B: 0xff, A: uint8(170 * (1 - t) * fade)}, true)
	}
}

func cloudGrowFade(a *AreaEffect, fadeFrames int) float64 {
	fade := math.Min(1, float64(a.Frame)/10)
	if a.Life < fadeFrames {
		fade = math.Min(fade, float64(a.Life)/float64(fadeFrames))
	}
	return fade
}

func (g *Game) drawTearGasCloud(screen *ebiten.Image, a *AreaEffect) {
	x, y := g.worldX(a.X), g.worldY(a.Y)
	fade := cloudGrowFade(a, tearGasFadeFrames)
	const puffs = 11
	for k := 0; k < puffs; k++ {
		seed := effectHash(int(a.X)+int(a.Y), k)
		ang := float64(k)/puffs*2*math.Pi + float64(seed%100)/100 + float64(a.Frame)*0.006*float64(1-2*(k%2))
		d := a.Radius * (0.25 + 0.5*float64(seed%100)/100)
		px := x + math.Cos(ang)*d
		py := y + math.Sin(ang)*d*0.55 - math.Sin(float64(a.Frame+seed)*0.05)*4
		r := a.Radius * (0.45 + 0.2*float64(seed%50)/50)
		drawPuff(screen, px, py, r, 0.78, 0.86, 0.32, float32(0.30*fade))
	}
	drawPuff(screen, x, y, a.Radius*0.8, 0.85, 0.92, 0.4, float32(0.28*fade))
}

// drawSmokeClouds is drawn ABOVE the players: on a shared screen it hides
// whoever stands inside. A thermal-goggles viewer (online / single human)
// sees it thin instead.
func (g *Game) drawSmokeClouds(screen *ebiten.Image) {
	if len(g.areaEffects) == 0 {
		return
	}
	viewerThermal := false
	if viewer := g.playerByID(g.viewerPlayerID()); viewer != nil && viewer.Active && viewer.AbilityNumber == abilityThermalGoggles {
		viewerThermal = true
	}
	for i := range g.areaEffects {
		a := &g.areaEffects[i]
		if a.Dead || a.Kind != areaEffectSmoke || a.Life <= 0 {
			continue
		}
		x, y := g.worldX(a.X), g.worldY(a.Y)
		density := float32(smokeDensity(a))
		if a.Frame < 12 {
			density = float32(math.Max(0.35, float64(a.Frame)/12))
		}
		alpha := 0.9 * density
		if viewerThermal {
			alpha *= 0.3
		}
		const puffs = 14
		for k := 0; k < puffs; k++ {
			seed := effectHash(int(a.X)*3+int(a.Y), k)
			ang := float64(k)/puffs*2*math.Pi + float64(a.Frame)*0.004*float64(1-2*(k%2))
			d := a.Radius * (0.15 + 0.45*float64(seed%100)/100)
			px := x + math.Cos(ang)*d
			py := y + math.Sin(ang)*d*0.6 - float64(a.Frame%300)*0.02
			r := a.Radius * (0.55 + 0.25*float64(seed%60)/60)
			shade := float32(0.62 + 0.18*float64(seed%40)/40)
			drawPuff(screen, px, py, r, shade, shade, shade*1.02, alpha*0.55)
		}
		drawPuff(screen, x, y, a.Radius*0.95, 0.7, 0.7, 0.72, alpha*0.85)
	}
	if viewerThermal {
		// Heat silhouettes of everyone inside the smoke, for this viewer only.
		for _, p := range g.players {
			if p == nil || !p.Active || !playerInSmoke(g.areaEffects, p) {
				continue
			}
			drawP := g.worldDrawPlayer(p)
			g.drawPlayerSilhouette(screen, &drawP, 1, 0.45, 0.1, 0.8)
		}
		return
	}
	// Shared screen: thermal wearers mark people hidden in smoke with a small
	// chevron in the wearer's colour (visible to everybody).
	for _, wearer := range g.players {
		if wearer == nil || !wearer.Active || wearer.AbilityNumber != abilityThermalGoggles {
			continue
		}
		clr := developedPlayerColorSwatch(wearer.PlayerColor)
		clr.A = 0xee
		for _, p := range g.players {
			if p == nil || !p.Active || p.ID == wearer.ID || p.scoreOwnerID() == wearer.scoreOwnerID() || !playerInSmoke(g.areaEffects, p) {
				continue
			}
			mx := float32(g.worldX(p.X))
			my := float32(g.worldY(p.Y - 85*playerRenderScale(p)/0.8))
			bob := float32(math.Sin(float64(g.mapFXFrame+p.ID*13)*0.2)) * 2
			vector.StrokeLine(screen, mx-7, my-7+bob, mx, my+bob, 3, clr, true)
			vector.StrokeLine(screen, mx+7, my-7+bob, mx, my+bob, 3, clr, true)
			vector.DrawFilledCircle(screen, mx, my+10+bob, 2.5, color.NRGBA{R: 0xff, G: 0x70, B: 0x20, A: 0xd0}, true)
		}
	}
}

// worldDrawPlayer is the same screen-space copy the main player loop draws.
func (g *Game) worldDrawPlayer(p *Player) Player {
	drawP := *p
	drawP.X = g.worldX(p.X)
	drawP.Y = g.worldY(p.Y)
	if g.shouldDrawInvisibleGhost(p) && drawP.Alpha < .24 {
		drawP.Alpha = .24
	}
	return drawP
}

// drawPlayerSilhouette draws drawP as a flat colour silhouette (white-hot
// flash victims, thermal heat view).
func (g *Game) drawPlayerSilhouette(screen *ebiten.Image, drawP *Player, r, gr, b, alpha float64) {
	if screen == nil || drawP == nil || alpha <= 0 {
		return
	}
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	if silhouetteBuffer == nil || silhouetteBuffer.Bounds().Dx() != w || silhouetteBuffer.Bounds().Dy() != h {
		silhouetteBuffer = ebiten.NewImage(w, h)
	}
	silhouetteBuffer.Clear()
	g.drawPlayer(silhouetteBuffer, drawP)
	var cm colorm.ColorM
	cm.Scale(0, 0, 0, alpha)
	cm.Translate(r, gr, b, 0)
	colorm.DrawImage(screen, silhouetteBuffer, cm, &colorm.DrawImageOptions{})
}

// drawPlayerStatusEffects: blinded players other than this screen's viewer
// get a white bubble and a white-hot sprite; coughing players emit puffs.
func (g *Game) drawPlayerStatusEffects(screen *ebiten.Image) {
	g.drawBurningPanicAll(screen)
	viewer := g.viewerPlayerID()
	for _, p := range g.players {
		if p == nil || !p.Active {
			continue
		}
		if p.TearGasTime > 0 {
			g.drawCoughPuffs(screen, p)
		}
		if p.BlindTime <= 0 || p.ID == viewer {
			continue
		}
		level := blindLevel(p)
		drawP := g.worldDrawPlayer(p)
		cx, cy := drawP.X, drawP.Y-35*playerRenderScale(p)/0.8
		pulse := 1 + 0.06*math.Sin(float64(p.BlindTime)*0.5)
		drawPuff(screen, cx, cy, 95*pulse*(0.5+0.5*level), 1, 1, 1, float32(0.75*level))
		g.drawPlayerSilhouette(screen, &drawP, 1, 1, 0.92, 0.7*level)
		// Little dizzy stars over the head.
		for k := 0; k < 3; k++ {
			ang := float64(p.BlindTime)*0.15 + float64(k)*2.094
			sx := drawP.X + math.Cos(ang)*14
			sy := drawP.Y - 92*playerRenderScale(p)/0.8 + math.Sin(ang)*4
			vector.DrawFilledCircle(screen, float32(sx), float32(sy), 2.2, color.NRGBA{R: 0xff, G: 0xf6, B: 0x9a, A: uint8(230 * level)}, true)
		}
	}
}

func (g *Game) drawCoughPuffs(screen *ebiten.Image, p *Player) {
	scale := playerRenderScale(p)
	hx := g.worldX(p.X) + 14*scale*float64(p.Facing)
	hy := g.worldY(p.Y - 60*scale/0.8)
	for k := 0; k < 3; k++ {
		t := math.Mod(float64(p.TearGasTime+k*9)/27, 1)
		px := hx + float64(p.Facing)*t*16
		py := hy - t*12
		drawPuff(screen, px, py, 4+8*t, 0.85, 0.9, 0.55, float32(0.7*(1-t)))
	}
}

func (g *Game) drawFlashBursts(screen *ebiten.Image) {
	for i := range g.areaEffects {
		a := &g.areaEffects[i]
		if a.Dead || a.Kind != areaEffectFlash || a.MaxLife <= 0 {
			continue
		}
		t := 1 - float64(a.Life)/float64(a.MaxLife) // 0 -> 1
		x, y := g.worldX(a.X), g.worldY(a.Y)
		drawPuff(screen, x, y, 40+a.MaxRadius*0.9*t, 1, 1, 0.95, float32(0.9*(1-t)))
		drawPuff(screen, x, y, 18+30*(1-t), 1, 1, 1, float32(1-t))
		vector.StrokeCircle(screen, float32(x), float32(y), float32(20+a.MaxRadius*t), 3, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: uint8(200 * (1 - t))}, true)
	}
}

// drawViewerBlindWhiteout whites out the whole screen for the one human this
// screen belongs to (single human vs bots, or each online client for its own
// player). Shared local screens never get it.
func (g *Game) drawViewerBlindWhiteout(screen *ebiten.Image) {
	viewer := g.playerByID(g.viewerPlayerID())
	if viewer == nil || !viewer.Active || viewer.BlindTime <= 0 || viewer.BlindMax <= 0 {
		return
	}
	hold := math.Min(1, float64(viewer.BlindTime)/(0.6*float64(viewer.BlindMax)))
	alpha := math.Max(0, math.Min(1, viewer.BlindStrength)) * hold
	if alpha <= 0 {
		return
	}
	w, h := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	ebitenutil.DrawRect(screen, 0, 0, w, h, color.NRGBA{R: 0xff, G: 0xff, B: 0xfa, A: uint8(255 * alpha)})
}

// drawHUDStatusEffects flashes the HUD card of a blinded player and shows a
// remaining-time bar; coughing players get a smaller green bar.
func (g *Game) drawHUDStatusEffects(screen *ebiten.Image, p *Player, x, y float64) {
	if p == nil || !p.Active {
		return
	}
	barX, barW := x-36.0, 132.0
	if p.BlindTime > 0 && p.BlindMax > 0 {
		if (p.BlindTime/6)%2 == 0 {
			ebitenutil.DrawRect(screen, x-40, y-34, 140, 58, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: uint8(150 * blindLevel(p))})
		}
		frac := float64(p.BlindTime) / float64(p.BlindMax)
		ebitenutil.DrawRect(screen, barX, y+26, barW, 9, color.NRGBA{A: 0xb0})
		ebitenutil.DrawRect(screen, barX+1, y+27, (barW-2)*frac, 7, color.NRGBA{R: 0xff, G: 0xff, B: 0xe0, A: 0xff})
		drawSourceHUDText(screen, "BLINDED", 9, color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xff}, barX+3, y+25)
	} else if p.BurnTime > 0 {
		drawHUDBurning(screen, p, x, y)
	} else if p.TearGasTime > 0 {
		frac := float64(p.TearGasTime) / tearGasLingerFrames
		ebitenutil.DrawRect(screen, barX, y+26, barW, 7, color.NRGBA{A: 0xa0})
		ebitenutil.DrawRect(screen, barX+1, y+27, (barW-2)*frac, 5, color.NRGBA{R: 0xc8, G: 0xd8, B: 0x48, A: 0xff})
	}
}

// ---------------------------------------------------------------------------
// Head and body perk gear

// playerPartWorld maps a player-local point to screen space the same way
// drawPlayerPart places parts.
func playerPartWorld(p *Player, lx, ly float64) (float32, float32) {
	scale := playerRenderScale(p)
	facing := float64(p.Facing)
	if facing == 0 {
		facing = 1
	}
	return float32(p.X + lx*scale*facing), float32(p.Y + ly*scale)
}

// eyeAnchor is the local centre and size of the eye layer.
func eyeAnchor(p *Player, eyes *SourceRaster) (cx, cy, w, h float64) {
	if eyes != nil && eyes.Bounds.W > 0 && eyes.Bounds.H > 0 {
		return eyes.Bounds.X + eyes.Bounds.W/2, p.VisualEyesY + eyes.Bounds.Y + eyes.Bounds.H/2, eyes.Bounds.W, eyes.Bounds.H
	}
	return 10, p.VisualEyesY - 8, 16, 8
}

// drawPlayerHeadgear draws goggles / gas mask over the eyes. It is only
// called instead of the chosen hat; the saved hat (Player.HatNumber and the
// player's config) is untouched and shows again without the perk.
func (g *Game) drawPlayerHeadgear(screen *ebiten.Image, p *Player, perk int, eyes *SourceRaster) {
	cx, cy, w, h := eyeAnchor(p, eyes)
	scale := float32(playerRenderScale(p))
	alpha := p.Alpha
	if alpha <= 0 {
		return
	}
	a := func(v uint8) uint8 { return uint8(float64(v) * alpha) }
	lensR := float32(math.Max(4.5, math.Min(9, h*0.65))) * scale
	spread := math.Max(5, w*0.27)
	backX, backY := playerPartWorld(p, cx-26, cy-2)
	leftX, leftY := playerPartWorld(p, cx-spread, cy)
	rightX, rightY := playerPartWorld(p, cx+spread, cy)
	strap := color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: a(0xff)}

	switch perk {
	case abilityGasMask:
		// Strap, rubber face plate over mouth/nose, round filter canister.
		vector.StrokeLine(screen, backX, backY, leftX, leftY, 3*scale, strap, true)
		mx, my := playerPartWorld(p, cx+2, cy+h*0.5+7)
		vector.DrawFilledCircle(screen, mx, my, 10*scale, color.NRGBA{R: 0x3a, G: 0x3f, B: 0x38, A: a(0xff)}, true)
		fx, fy := playerPartWorld(p, cx+11, cy+h*0.5+11)
		vector.DrawFilledCircle(screen, fx, fy, 6.5*scale, color.NRGBA{R: 0x58, G: 0x5e, B: 0x52, A: a(0xff)}, true)
		vector.StrokeCircle(screen, fx, fy, 6.5*scale, 1.4*scale, color.NRGBA{R: 0x1a, G: 0x1a, B: 0x1a, A: a(0xff)}, true)
		vector.StrokeCircle(screen, fx, fy, 3*scale, 1*scale, color.NRGBA{R: 0x8a, G: 0x90, B: 0x80, A: a(0xff)}, true)
		for i, lx := range [2]float32{leftX, rightX} {
			ly := leftY
			if i == 1 {
				ly = rightY
			}
			vector.DrawFilledCircle(screen, lx, ly, lensR*0.85, color.NRGBA{R: 0xb8, G: 0xd8, B: 0xe0, A: a(0xa0)}, true)
			vector.StrokeCircle(screen, lx, ly, lensR*0.85, 1.8*scale, color.NRGBA{R: 0x2c, G: 0x2f, B: 0x2a, A: a(0xff)}, true)
		}
	case abilityFlashGoggles, abilityThermalGoggles:
		vector.StrokeLine(screen, backX, backY, leftX, leftY, 3*scale, strap, true)
		vector.StrokeLine(screen, leftX, leftY, rightX, rightY, 3*scale, strap, true)
		lens := color.NRGBA{R: 0x10, G: 0x10, B: 0x14, A: a(0xf0)}
		glow := color.NRGBA{R: 0x88, G: 0x88, B: 0xa0, A: a(0x90)}
		frame := color.NRGBA{R: 0x55, G: 0x55, B: 0x5a, A: a(0xff)}
		if perk == abilityThermalGoggles {
			lens = color.NRGBA{R: 0x1c, G: 0xc8, B: 0x3a, A: a(0xf0)}
			glow = color.NRGBA{R: 0x9a, G: 0xff, B: 0x9a, A: a(0xd0)}
			frame = color.NRGBA{R: 0x2a, G: 0x34, B: 0x2a, A: a(0xff)}
		}
		for i, lx := range [2]float32{leftX, rightX} {
			ly := leftY
			if i == 1 {
				ly = rightY
			}
			if perk == abilityThermalGoggles {
				drawPuff(screen, float64(lx), float64(ly), float64(lensR)*2.2, 0.3, 1, 0.4, float32(0.35*alpha))
			}
			vector.DrawFilledCircle(screen, lx, ly, lensR, lens, true)
			vector.StrokeCircle(screen, lx, ly, lensR, 1.8*scale, frame, true)
			vector.DrawFilledCircle(screen, lx-lensR*0.3, ly-lensR*0.35, lensR*0.28, glow, true)
		}
	}
}

// drawPlayerKevlar draws a vest over the torso (body slot, so it stacks with
// any head item). Pure render-time decoration.
func (g *Game) drawPlayerKevlar(screen *ebiten.Image, p *Player, torso *SourceRaster) {
	if p == nil || p.AbilityNumber != abilityKevlar || p.Alpha <= 0 {
		return
	}
	cx, cy, w, h := 0.0, p.VisualBodyY+25.3+16, 30.0, 28.0
	if torso != nil && torso.Bounds.W > 0 && torso.Bounds.H > 0 {
		cx = torso.Bounds.X + torso.Bounds.W/2
		cy = p.VisualBodyY + 25.3 + torso.Bounds.Y + torso.Bounds.H*0.55
		w = torso.Bounds.W * 0.82
		h = torso.Bounds.H * 0.6
	}
	scale := float32(playerRenderScale(p))
	a := uint8(255 * math.Min(1, p.Alpha))
	x0, y0 := playerPartWorld(p, cx-w/2, cy-h/2)
	x1, y1 := playerPartWorld(p, cx+w/2, cy+h/2)
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	vw, vh := x1-x0, y1-y0
	vector.DrawFilledRect(screen, x0, y0, vw, vh, color.NRGBA{R: 0x3b, G: 0x45, B: 0x30, A: a}, true)
	vector.StrokeRect(screen, x0, y0, vw, vh, 1.5*scale, color.NRGBA{R: 0x1d, G: 0x22, B: 0x18, A: a}, true)
	// Plate seams and a pouch row.
	vector.StrokeLine(screen, x0+vw/2, y0+2, x0+vw/2, y1-2, 1*scale, color.NRGBA{R: 0x26, G: 0x2d, B: 0x1f, A: a}, true)
	for i := 0; i < 3; i++ {
		px := x0 + vw*(0.12+0.3*float32(i))
		vector.DrawFilledRect(screen, px, y1-vh*0.38, vw*0.2, vh*0.28, color.NRGBA{R: 0x4c, G: 0x58, B: 0x3d, A: a}, true)
	}
}

// ---------------------------------------------------------------------------
// Menu art

func drawGearIcon(dst *ebiten.Image, perk int, cx, cy float64, alpha float64) {
	a := uint8(255 * alpha)
	x, y := float32(cx), float32(cy)
	switch perk {
	case abilityFlashGoggles, abilityThermalGoggles:
		lens := color.NRGBA{R: 0x10, G: 0x10, B: 0x14, A: a}
		if perk == abilityThermalGoggles {
			lens = color.NRGBA{R: 0x1c, G: 0xc8, B: 0x3a, A: a}
		}
		vector.StrokeLine(dst, x-12, y, x+12, y, 2, color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: a}, true)
		vector.DrawFilledCircle(dst, x-5, y, 4.5, lens, true)
		vector.DrawFilledCircle(dst, x+5, y, 4.5, lens, true)
	case abilityGasMask:
		vector.DrawFilledCircle(dst, x, y, 7, color.NRGBA{R: 0x3a, G: 0x3f, B: 0x38, A: a}, true)
		vector.DrawFilledCircle(dst, x+6, y+4, 4, color.NRGBA{R: 0x70, G: 0x76, B: 0x68, A: a}, true)
		vector.DrawFilledCircle(dst, x-3, y-2, 2, color.NRGBA{R: 0xb8, G: 0xd8, B: 0xe0, A: a}, true)
	case abilityKevlar:
		vector.DrawFilledRect(dst, x-7, y-7, 14, 14, color.NRGBA{R: 0x3b, G: 0x45, B: 0x30, A: a}, true)
		vector.StrokeLine(dst, x, y-6, x, y+6, 1, color.NRGBA{R: 0x1d, G: 0x22, B: 0x18, A: a}, true)
	}
}

// utilityThrowableImage draws the procedural sprite of a developed utility
// throwable (shock / flash / smoke / tear gas).
func utilityThrowableImage(kind int) *ebiten.Image {
	img := ebiten.NewImage(20, 40)
	body := color.NRGBA{R: 0x6a, G: 0x6e, B: 0x70, A: 0xff}
	band := color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}
	switch kind {
	case ThrowableShock:
		body = color.NRGBA{R: 0x2a, G: 0x6c, B: 0x9a, A: 0xff}
		band = color.NRGBA{R: 0xb8, G: 0xf0, B: 0xff, A: 0xff}
	case ThrowableFlashbang:
		body = color.NRGBA{R: 0x50, G: 0x56, B: 0x4a, A: 0xff}
		band = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	case ThrowableSmoke:
		body = color.NRGBA{R: 0x8a, G: 0x8e, B: 0x90, A: 0xff}
		band = color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xff}
	case ThrowableTearGas:
		body = color.NRGBA{R: 0x8a, G: 0x8c, B: 0x3a, A: 0xff}
		band = color.NRGBA{R: 0xd8, G: 0xe0, B: 0x50, A: 0xff}
	}
	// Spoon + pin, cylinder body, coloured band.
	vector.DrawFilledRect(img, 6, 1, 8, 6, color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}, true)
	vector.StrokeCircle(img, 15, 4, 2.5, 1.2, color.NRGBA{R: 0xc0, G: 0xa0, B: 0x30, A: 0xff}, true)
	vector.DrawFilledRect(img, 3, 7, 14, 31, body, true)
	vector.DrawFilledRect(img, 3, 17, 14, 6, band, true)
	vector.StrokeRect(img, 3, 7, 14, 31, 1.2, color.NRGBA{R: 0x18, G: 0x18, B: 0x18, A: 0xff}, true)
	if kind == ThrowableFlashbang {
		for yy := float32(27); yy < 36; yy += 4 {
			vector.DrawFilledCircle(img, 7, yy, 1.2, color.NRGBA{A: 0xff}, true)
			vector.DrawFilledCircle(img, 13, yy, 1.2, color.NRGBA{A: 0xff}, true)
		}
	}
	return img
}

// throwableBehaviorLabel is the short stat line the gun library shows.
func throwableBehaviorLabel(kind int) string {
	switch throwableProfileFor(kind).Behavior {
	case throwableBehaviorIgnite:
		return "BURNING AREA"
	case throwableBehaviorShock:
		return "PUSHING GAS, NO DAMAGE"
	case throwableBehaviorFlash:
		return "BLINDS (FACE AWAY!)"
	case throwableBehaviorSmoke:
		return "SMOKE SCREEN"
	case throwableBehaviorTearGas:
		return "SLOW, LOW JUMP, BAD AIM"
	}
	return ""
}

// playerTorsoRaster resolves the torso raster drawPlayerBody uses (for the
// kevlar vest placement).
func (g *Game) playerTorsoRaster(p *Player) *SourceRaster {
	if g == nil || g.assets == nil || p == nil || p.AbilityNumber != abilityKevlar {
		return nil
	}
	colorIndex, useGM2Color := resolvedPlayerColor(p.PlayerColor, p.UseGM2Cosmetics)
	if useGM2Color {
		return g.assets.GM2TorsoColors[colorIndex]
	}
	if t := g.assets.TorsoColors[colorIndex]; t != nil {
		return t
	}
	return g.assets.TorsoColors[0]
}
