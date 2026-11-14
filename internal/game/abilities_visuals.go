package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
)

// Presentation of the ABILITY / GEAR slot. Everything is procedural vector
// art drawn from synced Player / Crate state, so online clients render it
// without new sprite files or extra protocol.

func abilityColor(n int) color.NRGBA {
	switch n {
	case abilityDash:
		return color.NRGBA{R: 0x33, G: 0xcc, B: 0xff, A: 0xff}
	case abilityAnchor:
		return color.NRGBA{R: 0x8a, G: 0x9b, B: 0xb0, A: 0xff}
	case abilityTripleJump:
		return color.NRGBA{R: 0xff, G: 0x55, B: 0xaa, A: 0xff}
	case abilityHeavy:
		return color.NRGBA{R: 0x9a, G: 0x6a, B: 0x3a, A: 0xff}
	case abilityLight:
		return color.NRGBA{R: 0xc8, G: 0xff, B: 0x7a, A: 0xff}
	case abilityShield:
		return color.NRGBA{R: 0x55, G: 0xc8, B: 0xff, A: 0xff}
	case abilityGrenadier:
		return color.NRGBA{R: 0xff, G: 0x66, B: 0x22, A: 0xff}
	case abilityScavenger:
		return color.NRGBA{R: 0xff, G: 0xcc, B: 0x33, A: 0xff}
	case abilityFlashGoggles:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xcc, A: 0xff}
	case abilityThermalGoggles:
		return color.NRGBA{R: 0x33, G: 0xff, B: 0x66, A: 0xff}
	case abilityGasMask:
		return color.NRGBA{R: 0xcc, G: 0xdd, B: 0x44, A: 0xff}
	case abilityKevlar:
		return color.NRGBA{R: 0x88, G: 0x99, B: 0x66, A: 0xff}
	case abilityBallisticShield:
		return color.NRGBA{R: 0x9c, G: 0xb4, B: 0xcc, A: 0xff}
	case abilityRandom:
		return color.NRGBA{R: 0xdd, G: 0xaa, B: 0xff, A: 0xff}
	}
	return color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}
}

func strokePathColor(dst *ebiten.Image, path *vector.Path, clr color.NRGBA, width float32) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.StrokePath(dst, path, &vector.StrokeOptions{Width: width}, op)
}

func withAlpha(c color.NRGBA, alpha float64) color.NRGBA {
	c.A = uint8(float64(c.A) * math.Max(0, math.Min(1, alpha)))
	return c
}

func measureMenuText(kind sourceMenuFontKind, size float64, value string) float64 {
	face, err := sourceMenuFace(kind, size)
	if err != nil || face == nil || value == "" {
		return 0
	}
	return float64(font.MeasureString(face, value).Ceil())
}

// drawAbilityIcon draws the ability symbol centred on (cx, cy); s is the
// icon half-size in pixels (about 10 for menu tiles).
func drawAbilityIcon(dst *ebiten.Image, n int, cx, cy, s, alpha float64) {
	x, y := float32(cx), float32(cy)
	k := float32(s / 10)
	clr := withAlpha(abilityColor(n), alpha)
	dark := withAlpha(color.NRGBA{R: 0x22, G: 0x22, B: 0x26, A: 0xff}, alpha)
	switch n {
	case abilityNone:
		vector.StrokeCircle(dst, x, y, 8*k, 2*k, withAlpha(color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff}, alpha), true)
		vector.StrokeLine(dst, x-5.5*k, y+5.5*k, x+5.5*k, y-5.5*k, 2*k, withAlpha(color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff}, alpha), true)
	case abilityDash:
		for i := 0; i < 3; i++ {
			yy := y + float32(i-1)*5*k
			vector.StrokeLine(dst, x-11*k+float32(i%2)*3*k, yy, x+1*k, yy, 2*k, clr, true)
		}
		vector.StrokeLine(dst, x+2*k, y-7*k, x+10*k, y, 2.5*k, clr, true)
		vector.StrokeLine(dst, x+10*k, y, x+2*k, y+7*k, 2.5*k, clr, true)
	case abilityAnchor:
		vector.StrokeCircle(dst, x, y-8*k, 2.6*k, 1.8*k, clr, true)
		vector.StrokeLine(dst, x, y-5.5*k, x, y+8*k, 2.2*k, clr, true)
		vector.StrokeLine(dst, x-5*k, y-3*k, x+5*k, y-3*k, 2*k, clr, true)
		var path vector.Path
		path.Arc(x, y+1*k, 8*k, 0.15*math.Pi, 0.85*math.Pi, vector.Clockwise)
		strokePathColor(dst, &path, clr, 2.2*k)
		vector.StrokeLine(dst, x-8*k, y+3*k, x-9.5*k, y-0.5*k, 2*k, clr, true)
		vector.StrokeLine(dst, x+8*k, y+3*k, x+9.5*k, y-0.5*k, 2*k, clr, true)
	case abilityTripleJump:
		for i := 0; i < 3; i++ {
			yy := y + 6*k - float32(i)*5.5*k
			w := 7*k - float32(i)*1.2*k
			vector.StrokeLine(dst, x-w, yy+3*k, x, yy-2*k, 2.2*k, clr, true)
			vector.StrokeLine(dst, x, yy-2*k, x+w, yy+3*k, 2.2*k, clr, true)
		}
	case abilityHeavy:
		var path vector.Path
		path.MoveTo(x-6*k, y-4*k)
		path.LineTo(x+6*k, y-4*k)
		path.LineTo(x+9*k, y+8*k)
		path.LineTo(x-9*k, y+8*k)
		path.Close()
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(clr)
		vector.FillPath(dst, &path, &vector.FillOptions{}, op)
		vector.StrokeCircle(dst, x, y-6.5*k, 3.2*k, 1.8*k, clr, true)
		vector.StrokeLine(dst, x-3*k, y+2*k, x+3*k, y+2*k, 1.5*k, dark, true)
	case abilityLight:
		// Feather: a quill with barbs.
		vector.StrokeLine(dst, x-7*k, y+8*k, x+7*k, y-8*k, 1.8*k, clr, true)
		for i := 0; i < 4; i++ {
			t := float32(i) / 4
			bx, by := x-4*k+t*10*k, y+4*k-t*11*k
			vector.StrokeLine(dst, bx, by, bx-5*k, by-2*k, 1.4*k, clr, true)
			vector.StrokeLine(dst, bx, by, bx+2*k, by+5*k, 1.4*k, clr, true)
		}
	case abilityShield:
		vector.DrawFilledCircle(dst, x, y, 9*k, withAlpha(color.NRGBA{R: 0x55, G: 0xc8, B: 0xff, A: 0x55}, alpha), true)
		vector.StrokeCircle(dst, x, y, 9*k, 2*k, clr, true)
		vector.DrawFilledCircle(dst, x-3.5*k, y-3.5*k, 2*k, withAlpha(color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xdd}, alpha), true)
	case abilityGrenadier:
		for i := 0; i < 2; i++ {
			gx := x + float32(i*2-1)*4.5*k
			vector.DrawFilledCircle(dst, gx, y+2*k, 4.5*k, withAlpha(color.NRGBA{R: 0x4a, G: 0x5a, B: 0x30, A: 0xff}, alpha), true)
			vector.DrawFilledRect(dst, gx-1.5*k, y-5*k, 3*k, 3*k, dark, true)
		}
		vector.StrokeLine(dst, x+5*k, y-8*k, x+11*k, y-8*k, 2*k, clr, true)
		vector.StrokeLine(dst, x+8*k, y-11*k, x+8*k, y-5*k, 2*k, clr, true)
	case abilityScavenger:
		vector.DrawFilledRect(dst, x-9*k, y-5*k, 12*k, 12*k, withAlpha(color.NRGBA{R: 0x9a, G: 0x6a, B: 0x30, A: 0xff}, alpha), true)
		vector.StrokeRect(dst, x-9*k, y-5*k, 12*k, 12*k, 1.2*k, dark, true)
		vector.StrokeCircle(dst, x+4*k, y-2*k, 4.5*k, 2*k, clr, true)
		vector.StrokeLine(dst, x+7*k, y+1.5*k, x+10.5*k, y+6*k, 2.4*k, clr, true)
	case abilityFlashGoggles, abilityThermalGoggles, abilityGasMask, abilityKevlar:
		drawGearIcon(dst, n, cx, cy, alpha)
	case abilityBallisticShield:
		drawBallisticShieldIcon(dst, cx, cy, s, alpha)
	case abilityRandom:
		vector.StrokeCircle(dst, x, y, 9*k, 2*k, clr, true)
		size := 15 * s / 10
		drawSourceMenuText(dst, "?", menuFontCondensedExtraBold, size, clr, cx-size*0.22, cy-size*0.62)
	}
}

// drawSetupAbilityThumb is one tile of the ABILITY / GEAR editor and picker.
func drawSetupAbilityThumb(dst *ebiten.Image, n int, r Rect, alpha float64) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	iconS := math.Min(10, math.Min(r.W, r.H-14)/2.6)
	drawAbilityIcon(dst, n, r.X+r.W/2, r.Y+4+iconS+2, iconS, alpha)
	name := abilityName(n)
	size := 9.5
	for size > 6.5 && measureMenuText(menuFontCondensedExtraBold, size, name) > r.W-4 {
		size -= 0.5
	}
	w := measureMenuText(menuFontCondensedExtraBold, size, name)
	drawSourceMenuText(dst, name, menuFontCondensedExtraBold, size,
		color.NRGBA{A: uint8(255 * alpha)}, r.X+(r.W-w)/2, r.Y+r.H-size-5)
}

// developedAbilitySelectionItems lays out the ABILITY / GEAR editor page
// (Symbol835 local coordinates) like the throwable page: three columns that
// end above BACK (y=759.45). Number is the menu index + 1.
func developedAbilitySelectionItems() []customSelectionItem {
	const (
		cols   = 3
		cellW  = 59.0
		cellH  = 54.0
		stepX  = 62.0
		stepY  = 58.0
		startX = 5.0
		startY = 428.0
	)
	list := selectableAbilityNumbers()
	out := make([]customSelectionItem, 0, len(list))
	for i := range list {
		out = append(out, customSelectionItem{Editor: customEditAbility, Number: i + 1, Rect: Rect{
			X: startX + float64(i%cols)*stepX, Y: startY + float64(i/cols)*stepY, W: cellW, H: cellH,
		}})
	}
	return out
}

// setupAbilityButtonX is the ABILITY edit button on the collapsed player card,
// right of the PERK button (Symbol806 at the PERK row's scale, 80 px wide).
const setupAbilityButtonX = 103.0

// drawSetupAbilityDescription explains the selected ability under the grid.
func drawSetupAbilityDescription(dst *ebiten.Image, ability int, localY, alpha float64) {
	text := abilityName(ability) + ": " + abilityDescription(ability)
	lines := wrapTextToWidth(text, 186, func(s string) float64 { return measureMenuText(menuFontCondensed, 9.5, s) })
	for i, line := range lines {
		if i >= 3 {
			break
		}
		drawSourceMenuText(dst, line, menuFontCondensed, 9.5,
			color.NRGBA{R: 0xe0, G: 0xe0, B: 0xe0, A: uint8(255 * alpha)}, 7, localY+718+float64(i)*12)
	}
}

// drawSetupCardAbilitySummary shows the chosen ability next to the GRENADE
// button on the collapsed player card.
func drawSetupCardAbilitySummary(dst *ebiten.Image, ability int, localY, alpha float64) {
	drawAbilityIcon(dst, ability, 112, localY+260, 8, alpha)
	name := abilityShortName(ability)
	drawSourceMenuText(dst, name, menuFontCondensedExtraBold, 9,
		color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: uint8(255 * alpha)}, 124, localY+254)
}

// ---------------------------------------------------------------------------
// In-game

// drawPlayerAbilityFX draws the shield bubble / break, the anchor clamps and
// the dash streaks. p is the camera-translated draw copy.
func (g *Game) drawPlayerAbilityFX(screen *ebiten.Image, p *Player) {
	if p == nil || !p.Active || p.Alpha <= 0 {
		return
	}
	scale := playerRenderScale(p)
	alpha := math.Min(1, p.Alpha)
	cx, cy := p.X, p.Y-35*scale
	if abilityShieldReady(p) {
		pulse := 0.75 + 0.25*math.Sin(float64(p.HitTimer+p.ID*13+p.IdleTime)*0.12)
		radius := 46 * scale
		drawPuff(screen, cx, cy, radius*1.25, 0.35, 0.8, 1, float32(0.16*alpha*pulse))
		vector.StrokeCircle(screen, float32(cx), float32(cy), float32(radius), float32(1.6*scale+0.6),
			color.NRGBA{R: 0x8c, G: 0xe0, B: 0xff, A: uint8(150 * alpha * pulse)}, true)
		vector.DrawFilledCircle(screen, float32(cx-radius*0.45), float32(cy-radius*0.5), float32(3.5*scale),
			color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: uint8(150 * alpha)}, true)
	}
	if p.AbilityShieldBreak > 0 {
		t := 1 - float64(p.AbilityShieldBreak)/shieldBreakFrames
		radius := (46 + 40*t) * scale
		a := uint8(230 * (1 - t) * alpha)
		vector.StrokeCircle(screen, float32(cx), float32(cy), float32(radius), float32(3*scale), color.NRGBA{R: 0xc8, G: 0xf0, B: 0xff, A: a}, true)
		for i := 0; i < 10; i++ {
			ang := float64(i)*math.Pi/5 + 0.3
			r0 := radius * 0.8
			r1 := radius*0.8 + 12*scale
			vector.StrokeLine(screen,
				float32(cx+math.Cos(ang)*r0), float32(cy+math.Sin(ang)*r0),
				float32(cx+math.Cos(ang)*r1), float32(cy+math.Sin(ang)*r1),
				float32(2.2*scale), color.NRGBA{R: 0xe8, G: 0xfa, B: 0xff, A: a}, true)
		}
	}
	if p.AnchorTime > 0 {
		g.drawAnchoredFX(screen, p, scale, alpha)
	}
	if p.DashTime > 0 {
		t := float64(p.DashTime) / dashTrailFrames
		dir := -float64(p.Facing)
		if dir == 0 {
			dir = -1
		}
		for i := 0; i < 4; i++ {
			yy := p.Y - (14+float64(i)*14)*scale
			x0 := p.X + dir*(18+float64(i%2)*8)*scale
			x1 := x0 + dir*(30+40*t)*scale
			vector.StrokeLine(screen, float32(x0), float32(yy), float32(x1), float32(yy), float32(2*scale),
				color.NRGBA{R: 0xbb, G: 0xee, B: 0xff, A: uint8(190 * t * alpha)}, true)
		}
	}
}

func (g *Game) drawAnchoredFX(screen *ebiten.Image, p *Player, scale, alpha float64) {
	left := float64(anchorMaxFrames-p.AnchorTime) / anchorMaxFrames
	steel := color.NRGBA{R: 0x5a, G: 0x66, B: 0x75, A: uint8(255 * alpha)}
	edge := color.NRGBA{R: 0x2a, G: 0x30, B: 0x38, A: uint8(255 * alpha)}
	// A steel base plate with two clamps bolted onto the platform.
	vector.DrawFilledRect(screen, float32(p.X-26*scale), float32(p.Y-1*scale), float32(52*scale), float32(5*scale), edge, true)
	for _, side := range [2]float64{-1, 1} {
		x := p.X + side*14*scale
		vector.DrawFilledRect(screen, float32(x-7*scale), float32(p.Y-5*scale), float32(14*scale), float32(7*scale), steel, true)
		vector.StrokeRect(screen, float32(x-7*scale), float32(p.Y-5*scale), float32(14*scale), float32(7*scale), float32(1.2*scale), edge, true)
		vector.StrokeLine(screen, float32(x), float32(p.Y+2*scale), float32(x), float32(p.Y+9*scale), float32(2.5*scale), edge, true)
	}
	// Heavy grey outline glow while planted.
	drawPuff(screen, p.X, p.Y-35*scale, 44*scale, 0.55, 0.62, 0.72, float32(0.22*alpha))
	// Anchor badge above the head with the remaining hold time as a ring.
	bx, by := p.X, p.Y-98*scale
	vector.DrawFilledCircle(screen, float32(bx), float32(by), float32(11*scale), color.NRGBA{R: 0x1c, G: 0x22, B: 0x2a, A: uint8(200 * alpha)}, true)
	if left > 0 {
		var path vector.Path
		path.Arc(float32(bx), float32(by), float32(11*scale), -math.Pi/2, float32(-math.Pi/2+2*math.Pi*left), vector.Clockwise)
		strokePathColor(screen, &path, color.NRGBA{R: 0xd0, G: 0xe4, B: 0xff, A: uint8(230 * alpha)}, float32(2.2*scale))
	}
	drawAbilityIcon(screen, abilityAnchor, bx, by+0.5*scale, 7*scale, alpha)
}

// drawHUDAbility is the small ability badge above the HUD card with the
// DASH / ANCHOR cooldown and the SHIELD state.
func (g *Game) drawHUDAbility(screen *ebiten.Image, p *Player, x, y float64) {
	if p == nil || p.AbilityNumber <= abilityNone || p.AbilityNumber >= abilityCount {
		return
	}
	r := Rect{X: x + 28, Y: y - 51, W: 72, H: 15}
	n := p.AbilityNumber
	base := abilityColor(n)
	ready := 1.0
	switch {
	case p.AnchorTime > 0:
		ready = float64(anchorMaxFrames-p.AnchorTime) / anchorMaxFrames
	case abilityUsesKey(n) && p.AbilityCooldown > 0 && p.AbilityCooldownMax > 0:
		ready = 1 - float64(p.AbilityCooldown)/float64(p.AbilityCooldownMax)
	case n == abilityShield && p.AbilityShieldUsed:
		ready = 0
	}
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, color.NRGBA{R: 0x14, G: 0x16, B: 0x1a, A: 0xd0})
	fill := withAlpha(base, 0.55)
	if ready >= 1 {
		fill = withAlpha(base, 0.9)
	}
	if ready > 0 {
		ebitenutil.DrawRect(screen, r.X+1, r.Y+1, (r.W-2)*math.Min(1, ready), r.H-2, fill)
	}
	if p.AnchorTime > 0 && (p.AnchorTime/6)%2 == 0 {
		vector.StrokeRect(screen, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), 1.5, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, false)
	}
	drawAbilityIcon(screen, n, r.X+8, r.Y+r.H/2, 5.5, 1)
	textClr := color.NRGBA{R: 0x10, G: 0x10, B: 0x10, A: 0xff}
	if ready < 1 {
		textClr = color.NRGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff}
	}
	drawSourceHUDText(screen, abilityShortName(n), 9, textClr, r.X+16, r.Y+2)
	if n == abilityShield && p.AbilityShieldUsed {
		vector.StrokeLine(screen, float32(r.X+2), float32(r.Y+r.H-2), float32(r.X+r.W-2), float32(r.Y+2), 1.2,
			color.NRGBA{R: 0xff, G: 0x55, B: 0x55, A: 0xff}, false)
	}
}

// ---------------------------------------------------------------------------
// SCAVENGER crate reveal

// crateRevealViewers returns the scavengers whose crate knowledge this screen
// may show: online, only the local player; with one local human, only that
// human; on a shared local screen, every local human scavenger (marked with
// their colour so the others know whose knowledge it is).
func (g *Game) crateRevealViewers() []*Player {
	if g == nil {
		return nil
	}
	if id := g.viewerPlayerID(); id != 0 {
		if p := g.playerByID(id); p != nil && p.Active && p.AbilityNumber == abilityScavenger {
			return []*Player{p}
		}
		return nil
	}
	if g.netplay != nil {
		return nil
	}
	var out []*Player
	for _, p := range g.players {
		if p == nil || !p.Active || p.AI || p.RemoteControlled || p.IsDouble || p.AbilityNumber != abilityScavenger {
			continue
		}
		out = append(out, p)
	}
	return out
}

// drawCrateScavengerLabels draws the crate contents above the players (so a
// scavenger standing next to a crate does not hide its label).
func (g *Game) drawCrateScavengerLabels(screen *ebiten.Image) {
	viewers := g.crateRevealViewers()
	if len(viewers) == 0 {
		return
	}
	for _, c := range g.crates {
		g.drawCrateScavengerInfo(screen, c, viewers)
	}
}

func (g *Game) drawCrateScavengerInfo(screen *ebiten.Image, c Crate, viewers []*Player) {
	if len(viewers) == 0 || c.Dead || c.Weapon <= 0 {
		return
	}
	name := weaponDisplayName(c.Weapon)
	if len([]rune(name)) > 16 {
		name = string([]rune(name)[:16])
	}
	x, y := g.worldX(c.X), g.worldY(c.Y)-62
	w := measureMenuText(menuFontCondensedExtraBold, 10, name) + 10
	shared := g.viewerPlayerID() == 0
	if shared {
		w += float64(len(viewers)) * 9
	}
	left := x - w/2
	ebitenutil.DrawRect(screen, left, y, w, 15, color.NRGBA{R: 0x10, G: 0x12, B: 0x16, A: 0xc8})
	ebitenutil.DrawRect(screen, left, y+15, w, 2, abilityColor(abilityScavenger))
	textX := left + 5
	if shared {
		for i, v := range viewers {
			sw := developedPlayerColorSwatch(v.PlayerColor)
			sw.A = 0xff
			vector.DrawFilledCircle(screen, float32(left+8+float64(i)*9), float32(y+7.5), 3.6, sw, true)
		}
		textX += float64(len(viewers)) * 9
	}
	drawSourceMenuText(screen, name, menuFontCondensedExtraBold, 10, color.NRGBA{R: 0xff, G: 0xe0, B: 0x80, A: 0xff}, textX, y+1)
}
