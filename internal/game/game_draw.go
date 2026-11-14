package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func (g *Game) Draw(screen *ebiten.Image) {
	if g.powerSaveSkipDraw() {
		return
	}
	setSourceRenderQuality(g.quality)
	// Start desktop audio only after Ebitengine has entered its real draw loop;
	// unit tests construct Game values without opening an audio device.
	if !g.audioStarted {
		g.audioStarted = true
		g.syncSourceMusic()
	}
	g.flushNetClientSFX()
	screen.Fill(color.RGBA{R: 30, G: 35, B: 41, A: 255})
	if g.netClientNeedsWaitingScreen() {
		g.drawNetClientWaitingScreen(screen)
		return
	}
	if g.screen != screenGameplay {
		g.drawMenu(screen)
		g.drawNetPingOverlay(screen)
		g.drawGlobalFade(screen)
		return
	}

	g.assets.EnsureGameplay()
	g.assets.EnsureScene(g.arena.Number)
	g.updateScreenShake()

	// Symbol 1391 source order/matrices: scene1 (0,0), scene2 (0,0),
	// gameplay, foreground (51.1,192.6). SourceRaster preserves each child
	// symbol's XFL registration point, so no recovered/manual crop offsets.
	// scene1.update(): (-root.x,-root.y)*0.9, then root transform itself.
	// Net screen motion is 10% of root camera. scene2 is 50%.
	drawSourceRaster(screen, g.assets.SceneBack[g.arena.Number], g.cameraX*0.1, g.cameraY*0.1, 1, 1, 1)
	drawSourceRaster(screen, g.assets.SceneMid[g.arena.Number], g.cameraX*0.5, g.cameraY*0.5, 1, 1, 1)

	if g.showCollisions {
		for _, r := range g.arena.Platforms {
			ebitenutil.DrawRect(screen, g.worldX(r.X), g.worldY(r.Y), r.W, r.H, color.RGBA{R: 255, G: 40, B: 40, A: 105})
		}
		ebitenutil.DrawLine(screen, 0, g.worldY(g.arena.LowestY), ScreenWidth, g.worldY(g.arena.LowestY), color.RGBA{R: 255, G: 210, B: 0, A: 220})
		ebitenutil.DrawLine(screen, g.worldX(g.arena.SpawnMinX), 0, g.worldX(g.arena.SpawnMinX), ScreenHeight, color.RGBA{R: 50, G: 220, B: 255, A: 180})
		ebitenutil.DrawLine(screen, g.worldX(g.arena.SpawnMaxX), 0, g.worldX(g.arena.SpawnMaxX), ScreenHeight, color.RGBA{R: 50, G: 220, B: 255, A: 180})
	}

	// Keep the arena artwork below dynamic gameplay objects. In Flash the
	// players are attached at runtime and must remain visually in front of the
	// map scene; drawing foreground after them made the character look like it
	// was buried one layer inside the arena.
	drawSourceRaster(screen, g.assets.SceneFront[g.arena.Number], g.worldX(51.1), g.worldY(192.6), 1, 1, 1)

	g.drawPlayerTrails(screen)
	g.drawCombat(screen)
	g.drawPortals(screen)
	g.drawFirePatches(screen)
	g.drawGroundAreaEffects(screen)
	g.drawCrates(screen)
	g.drawPowerups(screen)
	g.drawTestGunPickup(screen)
	for _, p := range g.players {
		if p.Active {
			drawP := *p
			drawP.X = g.worldX(p.X)
			drawP.Y = g.worldY(p.Y)
			if g.shouldDrawInvisibleGhost(p) && drawP.Alpha < .24 {
				drawP.Alpha = .24
			}
			g.drawPlayer(screen, &drawP)
		}
	}
	g.drawCampaignEnemies(screen)
	g.drawHazardMarkers(screen)
	g.drawPlayerStatusEffects(screen)
	g.drawCrateScavengerLabels(screen)
	g.drawSmokeClouds(screen)
	g.drawFlashBursts(screen)
	g.drawGrenadeChargeIndicators(screen)
	// fx_powerupname and fx_bling are root-attached runtime clips and sit over
	// the player/world art, before the high-depth HUD/player arrows.
	g.drawPowerupPickupEffects(screen)
	g.drawSourceMapFX(screen)
	// Source player_arrow clips use depth 10005..10008; HUD is depth 10010,
	// so off-screen indicators sit above gameplay/map FX but below the HUD.
	g.drawPlayerArrows(screen)
	g.drawCampaignEnemyArrows(screen)
	if !g.GameWin {
		g.drawHUD(screen)
		g.drawExtraCampaignHazardHUD(screen)
		g.drawExtraCampaignBossHUD(screen)
		g.drawChaosHUD(screen)
	}
	g.drawViewerBlindWhiteout(screen)
	g.drawGameplayInteractions(screen)
	g.drawNetPingOverlay(screen)
	g.drawGlobalFade(screen)
}

// Lobby and online setup are real interactive client screens. The old generic
// net-client guard treated every non-gameplay screen as a passive wait state,
// hiding MY CHARACTER and the client's own slot editor behind three debug
// lines. Only unknown screens, or a match screen before its first snapshot,
// should use the fallback waiting view.
func (g *Game) netClientNeedsWaitingScreen() bool {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayClient {
		return false
	}
	switch g.screen {
	case screenMultiplayerLobby, screenCustomGame:
		return false
	case screenGameplay, screenPostGame, screenSeriesSummary:
		return !g.netplay.hasState.Load()
	default:
		return true
	}
}

func (g *Game) shouldDrawInvisibleGhost(p *Player) bool {
	if g == nil || p == nil || !p.Active || p.InvisibleTime <= 0 {
		return false
	}
	if g.netplay != nil {
		return p.ID == g.onlineLocalPlayerID()
	}
	// A shared local screen cannot present a different image to each keyboard
	// player. Show the guide only for locally controlled humans, never AI.
	return !p.AI && !p.RemoteControlled
}

func (g *Game) drawSourceMapFX(screen *ebiten.Image) {
	switch g.arena.Number {
	case 5:
		// Symbol 1444 frame 5: root-attached mapfx at (0,0), therefore it
		// receives the full _root camera translation.
		drawSourceRaster(screen, g.assets.MapWall, g.cameraX, g.cameraY, 1, 1, 1)
	case 10:
		if g.assets.MapSnow == nil || g.mapFXFrame < 0 {
			return
		}
		// XFL lists top layers first. Ebiten draws back-to-front, so render the
		// two independent Symbol45 snow layers in reverse source layer order.
		for layer := len(g.assets.MapSnowTimelines) - 1; layer >= 0; layer-- {
			timeline := g.assets.MapSnowTimelines[layer]
			if g.mapFXFrame >= len(timeline) {
				continue
			}
			frame := timeline[g.mapFXFrame]
			if !frame.Valid {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(g.assets.MapSnow.Bounds.X, g.assets.MapSnow.Bounds.Y)
			child := geoMFromXFL(frame.Matrix)
			op.GeoM.Concat(child)
			// Symbol 46 frame-1 source: rotation=-20, X=420, Y=350.
			op.GeoM.Rotate(-20 * math.Pi / 180)
			op.GeoM.Translate(g.worldX(420), g.worldY(350))
			screen.DrawImage(g.assets.MapSnow.Image, op)
		}
	}
}

func (g *Game) drawCombat(screen *ebiten.Image) {
	for i := range g.bullets {
		g.drawBullet(screen, &g.bullets[i])
	}
	for _, t := range g.instagibTrails {
		if !t.Dead && t.Alpha > 0.01 && g.assets.InstagibTrail != nil {
			drawSourceRasterRot(screen, g.assets.InstagibTrail, g.worldX(t.X), g.worldY(t.Y), 1, 1, t.Rotation, t.Alpha)
		}
	}

	for _, s := range g.shells {
		if raster := g.assets.Shells[s.Kind]; raster != nil {
			sx := s.ScaleX
			if sx == 0 {
				sx = 1
			}
			drawSourceRasterRot(screen, raster, g.worldX(s.X), g.worldY(s.Y), sx, 1, s.Rotation, 1)
		}
	}

	for _, f := range g.flashes {
		if f.Smoke {
			x, y := float32(g.worldX(f.X)), float32(g.worldY(f.Y))
			s := float32(f.Scale)
			alpha := uint8(math.Max(0, math.Min(180, f.Alpha*180)))
			filledCircle(screen, x, y, 5*s, color.NRGBA{R: 0xc7, G: 0xc7, B: 0xc2, A: alpha})
			filledCircle(screen, x-5*s*float32(f.Facing), y-3*s, 3.8*s, color.NRGBA{R: 0x96, G: 0x96, B: 0x92, A: alpha / 2})
			continue
		}
		if f.Alpha <= 0.01 || g.assets.MuzzleFlash == nil {
			continue
		}
		sy := 1.0
		if f.FlipY {
			sy = -1
		}
		rotation := 0.0
		if f.Facing < 0 {
			rotation = -180
		}
		drawSourceRasterRot(screen, g.assets.MuzzleFlash, g.worldX(f.X), g.worldY(f.Y), 1, sy, rotation, f.Alpha)
	}

	for _, gr := range g.grenades {
		if !gr.Dead {
			g.drawThrowable(screen, gr)
		}
	}
	for _, fx := range g.dynamiteFX {
		if !fx.Dead && fx.Alpha > 0.01 && g.assets.DynamiteFX != nil {
			drawSourceRasterRot(screen, g.assets.DynamiteFX, g.worldX(fx.X), g.worldY(fx.Y), fx.Scale, fx.Scale, fx.Rotation, fx.Alpha)
		}
	}
	particles := g.renderProfile().blastParticles
	for _, fx := range g.blastFX {
		if fx.Dead || fx.Alpha <= 0.01 {
			continue
		}
		if !particles && blastKindIsQualityParticle(fx.Kind) {
			// QUALITY LOW: the original skipped these optional particles.
			continue
		}
		if raster := g.assets.ExplosionFX[fx.Kind]; raster != nil {
			drawSourceRasterRot(screen, raster, g.worldX(fx.X), g.worldY(fx.Y), fx.ScaleX, fx.ScaleY, fx.Rotation, fx.Alpha)
		}
	}
	for _, fx := range g.jetThrustFX {
		if fx.Dead || fx.Scale <= 0 {
			continue
		}
		if g.assets.JetThrustBase != nil {
			drawSourceRasterRot(screen, g.assets.JetThrustBase, g.worldX(fx.X), g.worldY(fx.Y), fx.Scale, fx.Scale, fx.Rotation, 1)
		}
		if g.assets.JetThrustFX != nil && fx.FXAlpha > 0 {
			drawSourceRasterRot(screen, g.assets.JetThrustFX, g.worldX(fx.X), g.worldY(fx.Y), fx.Scale, fx.Scale, fx.Rotation, fx.FXAlpha)
		}
	}
	for _, fx := range g.dropPackFX {
		if !fx.Dead && g.assets.DropPack != nil {
			drawSourceRasterRot(screen, g.assets.DropPack, g.worldX(fx.X), g.worldY(fx.Y), fx.ScaleX, 1, fx.Rotation, 1)
		}
	}

	// Do not draw the old synthetic orange explosion circle. Grenade death FX
	// are ported from the original fx_ex* clips separately; until every member
	// of that source effect stack is present, no invented replacement is drawn.
}

func (g *Game) drawBullet(screen *ebiten.Image, b *Bullet) {
	if b == nil || b.Dead {
		return
	}
	if g.drawDevelopedProjectile(screen, b) {
		return
	}
	if b.Kind == BulletInstagib {
		frame := 0
		if len(g.assets.InstagibFX) > 0 {
			frame = b.Time % len(g.assets.InstagibFX)
		}
		if raster := g.assets.InstagibFX[frame]; raster != nil {
			rotation := 0.0
			if b.Facing < 0 {
				rotation = 180
			}
			drawSourceRasterRot(screen, raster, g.worldX(b.X), g.worldY(b.Y), b.VisualScale, b.VisualScale, rotation, b.Alpha)
		}
		return
	}

	if b.TrailAlpha <= 0 || b.TrailWidth <= 0 {
		return
	}
	raster := g.assets.BulletTrail[b.TrailFrame]
	if raster == nil {
		return
	}
	natural := g.assets.BulletTrailNaturalWidth
	if natural <= 0 {
		natural = b.TrailWidth
	}
	sx := b.TrailWidth / natural
	rotation := b.Rotation
	// Straight tracers keep Rotation 0 unless aim disturbance (blind/smoke)
	// tilted them; then Rotation is their true flight angle.
	if b.Kind != BulletShotgun && b.Facing < 0 && b.Rotation == 0 {
		rotation = 180
	}
	drawSourceRasterRot(screen, raster, g.worldX(b.X), g.worldY(b.Y), sx, 1, rotation, b.TrailAlpha*b.Alpha)
}

func (g *Game) drawCrates(screen *ebiten.Image) {
	for _, c := range g.crates {
		if g.assets.Crate != nil {
			drawSourceRaster(screen, g.assets.Crate, g.worldX(c.X), g.worldY(c.Y), float64(c.Facing), 1, 1)
		} else {
			ebitenutil.DrawRect(screen, g.worldX(c.X-20), g.worldY(c.Y-40), 40, 40, color.RGBA{R: 120, G: 80, B: 35, A: 255})
		}
	}
}

func (g *Game) drawPowerups(screen *ebiten.Image) {
	for i := range g.powerups {
		pu := &g.powerups[i]
		if pu.Dead {
			continue
		}

		// Symbol 734 layer order: stuff -> aura -> flashystuff. Once picked up,
		// ActionScript sets stuff._alpha and aura._alpha to zero and leaves only
		// flashystuff until its 72nd frame removes the clip.
		if !pu.PickedUp {
			if pu.Frame >= 0 && pu.Frame < len(g.assets.PowerupStuffTimeline) {
				f := g.assets.PowerupStuffTimeline[pu.Frame]
				if f.Valid {
					drawPowerupChild(screen, g.assets.PowerupIcons[pu.Type], f.Matrix, g.worldX(pu.X), g.worldY(pu.Y), 1)
				}
			}
			if pu.Frame >= 0 && pu.Frame < len(g.assets.PowerupAuraTimeline) {
				f := g.assets.PowerupAuraTimeline[pu.Frame]
				if f.Valid {
					drawPowerupChild(screen, g.assets.PowerupAura, f.Matrix, g.worldX(pu.X), g.worldY(pu.Y), 1)
				}
			}
		}

		if pu.Frame >= 0 && pu.Frame < len(g.assets.PowerupFlashTimeline) {
			f := g.assets.PowerupFlashTimeline[pu.Frame]
			if f.Valid {
				drawPowerupChild(screen, g.assets.PowerupFlashFrame(pu.FlashFrame), f.Matrix, g.worldX(pu.X), g.worldY(pu.Y), 1)
			}
		}
		if g.showCollisions {
			ebitenutil.DebugPrintAt(screen, powerupName(pu.Type), int(g.worldX(pu.X))-25, int(g.worldY(pu.Y))-28)
		}
	}
}

func drawPowerupChild(screen *ebiten.Image, raster *SourceRaster, child xflMatrix, x, y, alpha float64) {
	if raster == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(raster.Bounds.X, raster.Bounds.Y)
	g := geoMFromXFL(child)
	op.GeoM.Concat(g)
	op.GeoM.Translate(x, y)
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	screen.DrawImage(raster.Image, op)
}

func (g *Game) drawHUD(screen *ebiten.Image) {
	g.drawSourceHUDCards(screen)
	g.drawKillFeeds(screen)
	if g.showCollisions {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("DEV | mapa %d/13 | gamemode %d | bron: %s | F2/F3 mapa | R reset", g.arena.Number, g.GameMode, g.players[0].Weapon.Def.DisplayName()), 10, 30)
		ebitenutil.DebugPrintAt(screen, "P1 strzalki + [ ] | P2 WASD + T Y | F4 AI P2 | F1 wylacz debug", 10, 46)
	}
}
