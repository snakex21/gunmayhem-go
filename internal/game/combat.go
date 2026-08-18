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

	Time        int
	Alpha       float64
	TrailAlpha  float64
	TrailWidth  float64
	TrailFrame  int
	VisualScale float64
	Dead        bool
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
	if w.WaitTime < w.Def.ROF {
		w.WaitTime++
	}

	// DefineSprite_697_player: shots are accepted only while the weapon movie
	// clip is stopped on Flash frame 2.
	if !p.shootPressed() || w.WaitTime < w.Def.ROF || w.Bullets <= 0 || w.Frame != 1 {
		return
	}

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
	recoil := w.Def.Recoil
	if p.PerkNumber == 5 {
		recoil = 0
	}
	p.VX -= recoil * float64(p.Facing)

	mini := p.MiniMulti
	if mini <= 0 {
		mini = 1
	}
	owner := p.scoreOwnerID()
	facing := p.Facing

	// FIREBULLET() from DefineSprite_697_player/frame_1/DoAction.as.
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

	// Source increments pgsdata[player-1][3] once per accepted trigger shot,
	// after FIREBULLET() (shotguns still count as one shot, not one per pellet).
	p.ShotsFired++
	w.Bullets--
	if w.Bullets <= 0 {
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

func sourceBulletTrailFrame(firepower float64) int {
	if firepower > 50 {
		return 2
	}
	return 1
}

// updateBullets is a direct split of the four source projectile movie clips.
// The unused dothehittest() helpers in the original clips are intentionally not
// invoked here: the exported ActionScript defines them but never calls them.
func updateBullets(arena Map, bullets []Bullet, players []*Player, instatrails *[]InstagibTrailEffect) {
	for i := range bullets {
		b := &bullets[i]
		if b.Dead {
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
		p.startInstagibTimeline()
		b.Dead = true
		return
	}
	if b.X < -600 || b.X > 1500 {
		b.Dead = true
	}
	b.Time++
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
		p.VX += push
		p.HitNumber++
		p.HitTimer = 0
		p.LastHitBy = b.OwnerID
		p.HitByGrenade = false
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
	if b.Team != 0 && p.Team == b.Team {
		return false
	}
	return p.scoreOwnerID() != b.OwnerID
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
