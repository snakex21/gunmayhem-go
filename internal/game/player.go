package game

import (
	"math"
	"math/rand"
)

type Player struct {
	ID       int
	Name     string
	Controls Controls

	X, Y       float64
	VX, VY     float64
	Facing     int
	JumpNum    int
	JumpHeld   bool
	DownHeld   bool
	FreePass   bool
	Grounded   bool
	HardLanded bool

	Lives             int
	TotalLives        int
	GameMode          int
	CurrentLevel      int
	CurrentGun        int
	KillSelf          bool
	Active            bool
	LastHitBy         int
	LastDeathBy       int
	LastDeathX        float64
	LastDeathY        float64
	LastDeathMod      int
	HitByGrenade      bool
	HitNumber         int
	HitTimer          int
	InstagibFrame     int
	InstagibPlaying   bool
	CheapTimer        int
	Kills             int
	Deaths            int
	ShotsFired        int
	HitsLanded        int
	CratesCollected   int
	PowerupsCollected int
	Score             int
	DeathSerial       int
	DefaultWeapon     int
	PerkNumber        int
	Weight            float64
	Weapon            WeaponState
	Grenades          int
	GrenadeHeld       bool
	GrenadePower      float64
	InvisibleTime     int
	ShieldTime        int
	ShieldFrame       int
	ShieldChildFrame  int
	ShieldPlaying     bool
	ShieldChildActive bool
	ShieldAlpha       float64
	JetFuel           float64
	JetpackAlpha      float64
	JetThrusted       bool
	JetDropped        bool
	SpeedTime         int
	MiniTime          int
	MiniMulti         float64
	PlayerScale       float64
	Alpha             float64
	PlayerColor       int
	ShirtNumber       int
	HatNumber         int
	HeadFrame         int
	VisualHeadY       float64

	// Source visual state from DefineSprite_697_player. These are MovieClip
	// properties in Flash and are eased every onEnterFrame; keeping them as
	// state avoids inventing positions in Draw().
	VisualBodyY        float64
	VisualEyesY        float64
	VisualHand1X       float64
	VisualHand1Y       float64
	VisualHand2X       float64
	VisualHand2Y       float64
	VisualHand1ChildX  float64
	VisualHand1ChildY  float64
	VisualHand2ChildX  float64
	VisualHand2ChildY  float64
	VisualHand1Alpha   float64
	VisualHand2Alpha   float64
	VisualGunX         float64
	VisualGunY         float64
	VisualGunRotation  float64
	VisualLeg1Rotation float64
	VisualLeg2Rotation float64
	IdleTime           int
	IdleMax            int
	WalkAnim           int
	WalkRequested      bool

	// Flash leg timelines are independent movie clips (Symbol 282 and 189).
	// They play while |vx| > 3 and stop on frame 1 after movement ends.
	LegFrame1   int
	LegFrame2   int
	LegPlaying1 bool
	LegPlaying2 bool
	TripleJump  bool

	AI               bool
	AILeft           bool
	AIRight          bool
	AIUp             bool
	AIDown           bool
	AIShoot          bool
	AIGrenade        bool
	AITargetPlayerID int
	AITargetPlayer   bool
	AITargetX        float64
	AITargetY        float64
	AITargetTimer    int
	AITargetValid    bool
	AITargetKind     int
	AITargetSerial   int
	AILockLeft       int
	AILockRight      int
	AILockUp         int
	AIIdleTime2      int
	AIPrevX          float64
	AISpecial        int
	AIFakeDoubleTime int
	DamageMulti      float64
	FirepowerMulti   float64

	Team             int
	OwnerPlayerID    int
	IsDouble         bool
	DoubleTime       int
	PersistentDouble       bool
	WantsDouble            bool
	DoubleSpawnPositionSet bool
	DoubleSpawnX           float64
	DoubleSpawnY           float64
	PendingSounds          []string
}

func NewPlayer(id int, m Map) *Player {
	p := &Player{
		ID:                id,
		Name:              "Player " + string(rune('0'+id)),
		Team:              id,
		Controls:          OriginalControls(id),
		Facing:            1,
		Lives:             10,
		TotalLives:        10,
		GameMode:          SourceGameModeNormal,
		CurrentLevel:      1,
		CurrentGun:        2,
		Active:            true,
		DefaultWeapon:     1,
		PerkNumber:        7,
		Weight:            1,
		DamageMulti:       1,
		FirepowerMulti:    1,
		Weapon:            NewWeapon(1),
		Grenades:          3,
		SpeedTime:         -1,
		MiniTime:          -1,
		MiniMulti:         1,
		PlayerScale:       0.8,
		Alpha:             1,
		ShieldAlpha:       1,
		PlayerColor:       sourceDefaultPlayerColor(id),
		ShirtNumber:       1,
		HatNumber:         1,
		HeadFrame:         rand.Intn(50) + 1,
		VisualBodyY:       -60,
		VisualEyesY:       -46,
		VisualHand1Y:      -35,
		VisualHand2Y:      -25,
		VisualHand1ChildX: -5,
		VisualHand1ChildY: 10.15,
		VisualHand2ChildX: 20,
		VisualHand2ChildY: -3.75,
		VisualHand1Alpha:  1,
		VisualHand2Alpha:  1,
		VisualGunX:        0,
		VisualGunY:        0,
		IdleTime:          40,
		IdleMax:           40,
	}
	p.Respawn(m)
	return p
}

func (p *Player) movementInput() (left, right, up, down bool) {
	if p.AI {
		return p.AILeft, p.AIRight, p.AIUp, p.AIDown
	}
	left = sourceKeyDown(p.Controls.Left)
	right = sourceKeyDown(p.Controls.Right)
	up = sourceKeyDown(p.Controls.Up)
	down = sourceKeyDown(p.Controls.Down)
	return
}

func (p *Player) shootPressed() bool {
	if p.AI {
		return p.AIShoot
	}
	return sourceKeyDown(p.Controls.Shoot)
}

func (p *Player) grenadePressed() bool {
	if p.AI {
		return p.AIGrenade
	}
	return sourceKeyDown(p.Controls.Grenade)
}

func (p *Player) clearAIInput() {
	p.AILeft = false
	p.AIRight = false
	p.AIUp = false
	p.AIDown = false
	p.AIShoot = false
	p.AIGrenade = false
}

func (p *Player) scoreOwnerID() int {
	if p.OwnerPlayerID > 0 {
		return p.OwnerPlayerID
	}
	return p.ID
}

func (p *Player) Update(m Map, beforeMove, afterPhysics func()) {
	if !p.Active {
		return
	}
	p.advanceShieldTimeline()
	p.HardLanded = false
	p.JetThrusted = false
	p.JetDropped = false

	if p.IsDouble && !p.PersistentDouble && p.DoubleTime >= 0 {
		p.DoubleTime--
		if p.DoubleTime <= 30 {
			p.Kill(m)
			return
		}
	}

	if beforeMove != nil {
		beforeMove()
	}

	p.X += p.VX
	p.Y += p.VY
	p.resolvePlatforms(m.Platforms)

	// Original AI checks death bounds before its movement decisions.
	if p.Y > 1000 || p.X < -600 || p.X > 1500 {
		p.Kill(m)
		return
	}
	if afterPhysics != nil {
		afterPhysics()
	}

	left, right, up, down := p.movementInput()
	p.WalkRequested = left || right
	moveSpeed := MoveSpeed
	if p.SpeedTime >= 0 {
		moveSpeed *= 2
	}

	// Original: player may drop through a platform only above ground.lowest.
	if down && p.JumpNum == 2 && !p.FreePass && !p.DownHeld && p.Y < m.LowestY {
		p.FreePass = true
		p.VY += 1
		p.Y += 5
		p.JumpNum = 1
		p.DownHeld = true
	} else if !down {
		p.DownHeld = false
	}

	if up && !p.JumpHeld && p.JumpNum > 0 {
		p.JumpHeld = true
		p.JumpNum--
		if p.JumpNum == 1 {
			p.VY = -JumpPower
		} else if p.JumpNum == 0 {
			p.VY = -JumpPower * 0.83
			if p.TripleJump {
				p.JumpNum = 11
				p.TripleJump = false
			}
		}
		if p.JumpNum == 10 {
			p.JumpNum = 0
			if p.VY < -4 {
				p.VY -= JumpPower * 0.2
			} else {
				p.VY = -JumpPower * 0.55
			}
		}
		p.Y--
	}
	if !up {
		p.JumpHeld = false
	}
	if up && p.JumpNum <= 0 && p.JetFuel > 0 && !p.JumpHeld {
		if p.JetpackAlpha != 1 {
			p.JetpackAlpha = 1
		}
		p.VY += (-15 - p.VY) / 4
		p.JetThrusted = true
		p.JetFuel -= 1.5
	}
	// Source updates the visible fuel bar first, then drops the pack if fuel is
	// exhausted, and otherwise forces the pack alpha to 100.
	if p.JetpackAlpha != 0 && p.JetFuel <= 0 {
		p.JetDropped = true
		p.JetpackAlpha = 0
	}
	if p.JetFuel > 0 && p.JetpackAlpha != 1 {
		p.JetpackAlpha = 1
	}

	if p.PerkNumber == 2 && p.Weight != 1 {
		// Source perk 2 forces weight back to 1 every frame.
		p.Weight = 1
	}
	weight := p.Weight
	if weight <= 0 {
		weight = 1
	}
	if left {
		if p.Facing == 1 {
			p.VisualLeg1Rotation *= -1
			p.VisualLeg2Rotation *= -1
		}
		p.Facing = -1
		if p.JumpNum == 2 {
			p.VX -= moveSpeed * weight
		} else if m.LowFriction {
			p.VX -= moveSpeed / 1.4
		} else {
			p.VX -= moveSpeed / 1.1 * weight
		}
	}
	if right {
		if p.Facing == -1 {
			p.VisualLeg1Rotation *= -1
			p.VisualLeg2Rotation *= -1
		}
		p.Facing = 1
		if p.JumpNum == 2 {
			p.VX += moveSpeed * weight
		} else if m.LowFriction {
			p.VX += moveSpeed / 1.4
		} else {
			p.VX += moveSpeed / 1.1 * weight
		}
	}

	p.VX *= Friction
	if p.VY > 30 {
		p.VY = 30
	}
	if p.VY < -30 {
		p.VY = -30
	}
	if p.VY < 24 {
		p.VY += m.Gravity
	}
	if p.VX > -0.1 && p.VX < 0.1 {
		p.VX = 0
	}

	if p.HitTimer <= 40 {
		p.HitTimer++
	} else {
		p.HitNumber = 0
	}
	if p.CheapTimer < 120 {
		p.CheapTimer++
	}
	p.updatePowerupTimers()
	p.advanceInstagibTimeline()
	if p.KillSelf {
		// Late source branch: SELFDESTRUCT() clears killself before gotkilled().
		p.KillSelf = false
		p.Kill(m)
	}
}

func (p *Player) resolvePlatforms(platforms []Rect) {
	landed := false
	if !p.FreePass && p.VY > 0 && pointHitsPlatforms(platforms, p.X, p.Y) {
		if math.Abs(p.VX) < 3 {
			p.LastHitBy = 0
			p.HitByGrenade = false
		}
		p.JumpNum = 2
		if p.PerkNumber == 3 {
			p.TripleJump = true
		}

		fallSpeed := p.VY
		contactY := p.Y
		p.Y -= fallSpeed * 1.01
		for i := 1; i <= 5; i++ {
			if pointHitsPlatforms(platforms, p.X, p.Y+float64(i)*(fallSpeed/5)) {
				p.Y += fallSpeed / 5 * (float64(i) - 0.5)
				break
			}
		}
		// Flash resolves the contact against the vector platform shape. With our
		// exact rectangular reconstruction, keeping the five-step approximation
		// above leaves the registration point a fraction of a pixel inside the
		// platform. Gravity then repeats that error every frame and the player
		// slowly sinks through the floor. Snap to the top edge of the exact
		// platform that produced this hit so the source registration point stays
		// on the surface instead of accumulating numerical drift.
		if top, ok := platformTopAtPoint(platforms, p.X, contactY); ok {
			p.Y = top
		}
		if math.Abs(fallSpeed) > 3 {
			p.HardLanded = true
			// Exact visual squash kick from the landing branch. The subsequent
			// player animation eases these values back to their source targets.
			p.VisualEyesY += 10
			p.VisualHand1Y += 10
			p.VisualHand2Y += 8
		}
		p.VY = 0
		p.Grounded = true
		landed = true
	}

	if !landed && p.JumpNum == 2 {
		p.JumpNum = 1
		p.Grounded = false
	}

	// DefineSprite_697_player uses these two hitTests separately.
	if !pointHitsPlatforms(platforms, p.X, p.Y-8) && !pointHitsPlatforms(platforms, p.X, p.Y) {
		p.FreePass = false
	}
	if pointHitsPlatforms(platforms, p.X, p.Y-8) && !p.FreePass {
		p.FreePass = true
	}
}

func platformTopAtPoint(platforms []Rect, x, y float64) (float64, bool) {
	var top float64
	found := false
	for _, r := range platforms {
		if !r.Contains(x, y) {
			continue
		}
		if !found || r.Y < top {
			top = r.Y
			found = true
		}
	}
	return top, found
}

func (p *Player) Hitbox() Rect {
	// Exact XFL hierarchy:
	// Symbol 691 shape = [-25,+25] x [-35,+35]
	// Symbol 697 instance `frame` = (0,-35), then player _xscale/_yscale.
	s := p.PlayerScale
	if s <= 0 {
		s = 0.8
	}
	return Rect{X: p.X - 25*s, Y: p.Y - 70*s, W: 50 * s, H: 70 * s}
}

func (p *Player) advanceShieldTimeline() {
	// Child Symbol695 exists only once Symbol696 reaches Flash frame6
	// (zero-based frame5). It is a normal nested movie clip and keeps looping
	// even while the parent shield is stopped on frame22.
	wasChildVisible := p.ShieldFrame >= 5
	if p.ShieldPlaying {
		p.ShieldFrame++
		if p.ShieldFrame >= 30 {
			p.ShieldFrame = 21 // frame30 gotoAndPlay(22), then frame22 stop()
			p.ShieldPlaying = false
		}
		if p.ShieldFrame == 21 {
			p.ShieldPlaying = false // source frame22 stop()
		}
	}
	childVisible := p.ShieldFrame >= 5
	if childVisible {
		if !wasChildVisible || !p.ShieldChildActive {
			p.ShieldChildFrame = 0
			p.ShieldChildActive = true
		} else {
			p.ShieldChildFrame++
			if p.ShieldChildFrame >= 5 {
				p.ShieldChildFrame = 0
			}
		}
	} else {
		p.ShieldChildFrame = 0
		p.ShieldChildActive = false
	}
}

func (p *Player) updateShieldSource() {
	// DefineSprite_697_player late shield block.
	if p.ShieldTime > 0 {
		p.ShieldTime--
		if p.ShieldAlpha < 1 {
			p.ShieldAlpha += 1
			if p.ShieldAlpha > 1 {
				p.ShieldAlpha = 1
			}
		}
		if p.ShieldFrame == 0 {
			p.ShieldPlaying = true // shield.play(); frame advances next tick
		}
		return
	}

	if p.ShieldAlpha > 0 {
		p.ShieldAlpha -= 0.10
		if p.ShieldAlpha < 0 {
			p.ShieldAlpha = 0
		}
	}
	if p.ShieldAlpha <= 0.01 && p.ShieldFrame != 0 {
		// gotoAndPlay(1) immediately enters source frame1 whose script stop()s.
		p.ShieldFrame = 0
		p.ShieldPlaying = false
		p.ShieldChildFrame = 0
		p.ShieldChildActive = false
	}
}

func (p *Player) startInstagibTimeline() {
	// Symbol688 starts stopped on Flash frame1. bullet3 calls instagib.play(),
	// which resumes the existing playhead rather than restarting the effect.
	if p.InstagibFrame == 0 && !p.InstagibPlaying {
		p.InstagibPlaying = true
	}
}

func (p *Player) advanceInstagibTimeline() {
	if !p.InstagibPlaying {
		return
	}
	// Symbol688: zero-based frame1..60 is the visible/delayed-death section;
	// zero-based frame61 executes _parent.SELFDESTRUCT().
	p.InstagibFrame++
	if p.InstagibFrame >= 61 {
		p.InstagibPlaying = false
		p.KillSelf = true
	}
}

func (p *Player) updatePowerupTimers() {
	if p.InvisibleTime > 0 {
		p.InvisibleTime--
		p.Alpha -= 0.1
		if p.Alpha < 0 {
			p.Alpha = 0
		}
	} else if p.Alpha < 1 {
		p.Alpha += 0.1
		if p.Alpha > 1 {
			p.Alpha = 1
		}
	}
	p.updateShieldSource()
	if p.JetFuel > 0 {
		// Late source tick: this happens after the jetpack drop/visibility checks.
		// It can make fuel slightly negative; the pack is dropped next frame.
		p.JetFuel -= 0.12
	}
	if p.SpeedTime >= 0 {
		p.SpeedTime--
	}
	if p.MiniTime >= 0 {
		p.MiniTime--
		if p.MiniTime > 260 {
			p.MiniMulti += (0.6 - p.MiniMulti) / 6
			p.PlayerScale += (0.5 - p.PlayerScale) / 3
		}
		if p.MiniTime == 260 {
			p.MiniMulti = 0.6
			p.PlayerScale = 0.5
		}
		if p.MiniTime <= 40 {
			p.MiniMulti += (1 - p.MiniMulti) / 6
			p.PlayerScale += (0.8 - p.PlayerScale) / 3
		}
	} else {
		if p.MiniMulti != 1 {
			p.MiniMulti = 1
		}
		if p.PlayerScale != 0.8 {
			p.PlayerScale = 0.8
		}
	}
}

func (p *Player) Kill(m Map) {
	if !p.Active {
		return
	}
	p.LastDeathBy = p.LastHitBy
	p.LastDeathX = p.X
	p.LastDeathY = p.Y
	p.LastDeathMod = 1
	if p.CheapTimer < 120 {
		p.LastDeathMod = 2
	} else if p.HitByGrenade {
		p.LastDeathMod = 3
	}
	if !p.IsDouble {
		p.Deaths++
	}
	p.DeathSerial++
	p.Lives--
	if p.Lives < 0 {
		p.Lives = 0
	}
	if p.Lives <= 0 {
		p.Active = false
		p.X = 0
		p.Y = 1100
		p.VX = 0
		p.VY = 0
		p.Alpha = 0
		p.ShieldTime = 0
		p.InvisibleTime = 0
		p.JetFuel = 0
		p.SpeedTime = 0
		p.KillSelf = false
		p.InstagibFrame = 0
		p.InstagibPlaying = false
		return
	}
	p.Respawn(m)
}

func (p *Player) Respawn(m Map) {
	// DefineSprite_697_player.respawn(): weapon branch is chosen before the
	// spawn position/state is reset.
	switch p.GameMode {
	case SourceGameModeInstagib:
		p.EquipWeapon(9)
	case SourceGameModeGunGame:
		p.EquipWeapon(p.CurrentGun)
	default: // source modes 1,3,5
		p.EquipWeapon(p.DefaultWeapon)
	}

	p.Y = -1000
	p.X = m.SpawnMinX
	width := int(m.SpawnMaxX - m.SpawnMinX)
	if width > 0 {
		// AS2 random(width) is an integer 0..width-1.
		p.X += float64(rand.Intn(width))
	}
	p.VY = 0
	p.VX = 0
	p.JumpNum = 1
	p.FreePass = false
	p.Grounded = false
	p.JumpHeld = false
	p.DownHeld = false
	p.LastHitBy = 0
	p.HitByGrenade = false
	p.HitNumber = 0
	p.HitTimer = 0
	p.InstagibFrame = 0
	p.InstagibPlaying = false
	p.CheapTimer = 0

	p.Grenades = 3
	if p.PerkNumber == 8 {
		p.Grenades += 2
	}
	p.GrenadeHeld = false
	p.GrenadePower = 0
	p.InvisibleTime = 0
	p.ShieldTime = 0
	p.JetFuel = 0
	if p.AISpecial == sourceAISpecialJetpack {
		// playerAI3.respawn() always restores a full jet pack.
		p.JetFuel = 100
	}
	p.SpeedTime = 0
	p.MiniTime = 0
	p.MiniMulti = 1
	p.PlayerScale = 0.8
	p.Alpha = 1
	p.JetpackAlpha = 0
	p.JetThrusted = false
	p.JetDropped = false
	p.LegFrame1 = 0
	p.LegFrame2 = 0
	p.LegPlaying1 = false
	p.LegPlaying2 = false

	if p.PerkNumber == 9 && p.GameMode != SourceGameModeGunGame && p.GameMode != SourceGameModeInstagib {
		p.EquipWeapon(rand.Intn(57) + 10)
	}
	if p.GameMode == SourceGameModeSurvival {
		p.ShieldTime = 140
	}
}

func sourceDefaultPlayerColor(id int) int {
	// Defaults from DefineSprite_1309/frame_1/DoAction.as.
	switch id {
	case 1:
		return 2
	case 2:
		return 5
	case 3:
		return 8
	case 4:
		return 10
	default:
		return 2
	}
}

func (p *Player) advanceLegTimelines(len1, len2 int) {
	if !p.Active {
		return
	}
	if p.JumpNum != 2 {
		p.LegFrame1, p.LegFrame2 = 0, 0
		p.LegPlaying1, p.LegPlaying2 = false, false
		p.WalkAnim = 0
		return
	}

	movingFast := math.Abs(p.VX) > 3
	// startwalk(): if the leg clip is stopped on Flash frame 1, pressing a
	// movement key executes gotoAndPlay(2) immediately.
	if p.WalkRequested && p.LegFrame1 == 0 && !p.LegPlaying1 {
		if len1 > 1 {
			p.LegFrame1 = 1
			p.LegPlaying1 = true
		}
		if len2 > 1 {
			p.LegFrame2 = 1
			p.LegPlaying2 = true
		}
	} else {
		if movingFast {
			p.LegPlaying1 = true
			p.LegPlaying2 = true
		}
		p.LegFrame1, p.LegPlaying1 = advanceFlashStoppedLoop(p.LegFrame1, p.LegPlaying1, movingFast, len1)
		p.LegFrame2, p.LegPlaying2 = advanceFlashStoppedLoop(p.LegFrame2, p.LegPlaying2, movingFast, len2)
	}

	// Symbol 189 frame scripts update parent.walkanim.
	if p.LegFrame2 == 5 {
		p.WalkAnim = 2
	}
	if p.LegFrame2 == 14 {
		p.WalkAnim = 1
	}
	if !movingFast && p.LegFrame1 == 0 {
		p.WalkAnim = 0
	}
}

func (p *Player) advanceHeadTimeline(timeline []SourceTransformFrame) {
	if !p.Active || len(timeline) == 0 {
		return
	}
	// Symbol 239 frame 1 executes gotoAndPlay(random(50)+2) on creation.
	// Frame 60 executes gotoAndPlay(2), so the steady loop is zero-based 1..58.
	p.HeadFrame++
	if p.HeadFrame >= 59 || p.HeadFrame >= len(timeline) {
		p.HeadFrame = 1
	}
	if p.HeadFrame >= 0 && p.HeadFrame < len(timeline) && timeline[p.HeadFrame].Valid {
		p.VisualHeadY = timeline[p.HeadFrame].Matrix.TY
	}
}

func (p *Player) updateSourceVisualState() {
	if !p.Active {
		return
	}
	pose := poseForWeapon(p.Weapon.Def.Number)
	anim := sourceWeaponAnim(p.Weapon.Def.Number)

	// idletime is advanced after shooting in the source player.onEnterFrame.
	if p.IdleTime <= p.IdleMax {
		p.IdleTime++
	}
	weaponBeforeFrame40 := p.Weapon.Frame < 39 // Flash _currentframe < 40
	idle := p.IdleTime >= p.IdleMax && p.Weapon.Def.Number != 55 && weaponBeforeFrame40
	if idle {
		p.VisualGunX += (-5.5 - p.VisualGunX) / 3
		p.VisualGunY += (10.85 - p.VisualGunY) / 3
		p.VisualGunRotation += (anim.IdleRotate - p.VisualGunRotation) / 3
	} else {
		p.VisualHand1ChildX += (pose.ShootX - p.VisualHand1ChildX) / 3
		p.VisualHand1ChildY += (pose.ShootY - p.VisualHand1ChildY) / 3
		p.VisualHand2ChildX += (pose.HandX - p.VisualHand2ChildX) / 3
		p.VisualHand2ChildY += (pose.HandY - p.VisualHand2ChildY) / 3
		p.VisualGunX += (pose.ShootX - p.VisualGunX) / 3
		p.VisualGunY += (pose.ShootY - p.VisualGunY) / 3
		p.VisualGunRotation += -p.VisualGunRotation / 3
	}

	// Late idle-hand block from the same source frame.
	if idle {
		switch p.WalkAnim {
		case 1:
			p.VisualHand1ChildX += -p.VisualHand1ChildX / 3
			p.VisualHand2ChildX += (15 - p.VisualHand2ChildX) / 3
		case 2:
			p.VisualHand1ChildX += (-10 - p.VisualHand1ChildX) / 3
			p.VisualHand2ChildX += (25 - p.VisualHand2ChildX) / 3
		default:
			p.VisualHand1ChildX += (-5 - p.VisualHand1ChildX) / 3
			p.VisualHand2ChildX += (20 - p.VisualHand2ChildX) / 3
		}
		p.VisualHand1ChildY += (10.5 - p.VisualHand1ChildY) / 3
		p.VisualHand2ChildY += (-3.75 - p.VisualHand2ChildY) / 3
	}
	// Source unconditionally assigns hand1.gun._x = hand1.hand._x here.
	p.VisualGunX = p.VisualHand1ChildX

	if p.VY <= -2 {
		p.VisualLeg1Rotation += (80*float64(p.Facing) - p.VisualLeg1Rotation) / 3
		p.VisualLeg2Rotation += (80*float64(p.Facing) - p.VisualLeg2Rotation) / 3
		p.VisualBodyY += (-70 - p.VisualBodyY) / 5
	} else if p.VY >= 2 {
		p.VisualLeg1Rotation += (-10*float64(p.Facing) - p.VisualLeg1Rotation) / 3
		p.VisualLeg2Rotation += (-10*float64(p.Facing) - p.VisualLeg2Rotation) / 3
		p.VisualBodyY += (-55 - p.VisualBodyY) / 5
	} else {
		p.VisualLeg1Rotation += -p.VisualLeg1Rotation / 1.5
		p.VisualLeg2Rotation += -p.VisualLeg2Rotation / 1.5
		p.VisualBodyY += (-60 - p.VisualBodyY) / 2
	}

	// Exact source target: -46 + body.head.head._y.
	eyesTarget := -46 + p.VisualHeadY
	if math.Abs(eyesTarget-p.VisualEyesY) >= 1 {
		p.VisualEyesY += (eyesTarget - p.VisualEyesY) / 5
		p.VisualHand1Y += (-35 - p.VisualHand1Y) / 5
		p.VisualHand2Y += (-25 - p.VisualHand2Y) / 5
	} else {
		p.VisualEyesY = math.Round(eyesTarget)
	}
}

func advanceFlashStoppedLoop(frame int, playing, keepPlaying bool, length int) (int, bool) {
	if length <= 0 {
		return 0, false
	}
	if frame < 0 || frame >= length {
		frame = 0
	}
	if !playing {
		return frame, false
	}
	frame++
	if frame >= length {
		frame = 0
		// Frame 1 of both leg movie clips executes stop(). While the player is
		// still moving, player.onEnterFrame calls play() again on the next tick.
		if !keepPlaying {
			playing = false
		}
	}
	return frame, playing
}

func (p *Player) EquipWeapon(number int) {
	def, ok := WeaponByNumber(number)
	if !ok {
		return
	}
	p.Weapon = NewWeapon(number)
	// The attached gun's first frame calls adjustrof() after getgun() returns.
	// That leaves waittime at ROF-2 for ordinary guns (or -10 for pistols),
	// so a freshly picked-up weapon needs only the source's short startup delay
	// instead of a full extra ROF cycle.
	if p.FirepowerMulti > 0 && p.FirepowerMulti != 1 {
		p.Weapon.Def.Firepower *= p.FirepowerMulti
	}
	// attachMovie() creates the replacement gun at the hand1 origin.
	p.VisualGunX = 0
	p.VisualGunY = 0
	p.VisualGunRotation = 0
	p.VisualHand1Alpha = 1
	p.VisualHand2Alpha = 1
	if p.PerkNumber == 7 {
		base := p.Weapon.Bullets
		p.Weapon.Bullets += int(math.Ceil(float64(base) * 0.333))
	}
	// A few original gun timelines do not write weight at all; in Flash that
	// leaves the previous player.weight untouched.
	if def.Weight > 0 {
		p.Weight = def.Weight
	}
	if p.PerkNumber == 2 {
		p.Weight = 1
	}
	if p.AISpecial == sourceAISpecialBoss {
		// playerAIboss.adjustrof(): boss weapons always have 4x ammo and weight 1.
		p.Weapon.Bullets *= 4
		p.Weight = 1
	}
}

func (p *Player) ResetRound(m Map) {
	p.configureSourceGameMode(p.GameMode, p.TotalLives, m)
}
