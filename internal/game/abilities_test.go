package game

import (
	"math"
	"testing"
	"time"
)

func abilityPlayer(id int, x float64, facing, ability int) *Player {
	p := utilityPlayer(id, x, facing)
	p.AI = true
	p.AbilityNumber = ability
	p.resetAbilityForLife()
	// Settle onto the platform: a player standing exactly on the top with
	// vy=0 is not "landed" until gravity has pulled it into the surface.
	p.Update(utilityArena(), nil, nil)
	p.Update(utilityArena(), nil, nil)
	return p
}

// abilityFrame runs one player update with the given AI buttons held.
func abilityFrame(p *Player, ability, left, right, up bool) {
	p.AIAbility, p.AILeft, p.AIRight, p.AIUp = ability, left, right, up
	p.Update(utilityArena(), nil, nil)
}

// bulletPush hits p with one ordinary bullet from the left inside a
// knockback window and returns the velocity change.
func bulletPushInWindow(p *Player) float64 {
	before := p.VX
	h := p.Hitbox()
	b := Bullet{Kind: BulletNormal, X: h.X + h.W/2, Y: h.Y + h.H/2, Firepower: 10, OwnerID: 9, Team: 9, Facing: 1, Alpha: 1}
	players := []*Player{p}
	window := beginAbilityKnockbackWindow(players)
	hitPlayerWithBullet(utilityArena(), &b, players, false)
	endAbilityKnockbackWindow(players, window, true)
	return p.VX - before
}

func blastPushInWindow(p *Player, ownerID int) (float64, float64) {
	bx, by := p.VX, p.VY
	gr := Grenade{X: p.X - 80, Y: p.Y - 20, Type: ThrowableDynamite, OwnerID: ownerID}
	players := []*Player{p}
	window := beginAbilityKnockbackWindow(players)
	explodeGrenade(&gr, players)
	endAbilityKnockbackWindow(players, window, true)
	return p.VX - bx, p.VY - by
}

func TestAbilityIDsAreAppendOnly(t *testing.T) {
	want := map[int]int{
		abilityNone: 0, abilityDash: 1, abilityAnchor: 2, abilityTripleJump: 3, abilityHeavy: 4,
		abilityLight: 5, abilityShield: 6, abilityGrenadier: 7, abilityScavenger: 8,
		abilityFlashGoggles: 9, abilityThermalGoggles: 10, abilityGasMask: 11, abilityKevlar: 12,
	}
	for got, n := range want {
		if got != n {
			t.Fatalf("ability id moved: got %d want %d", got, n)
		}
	}
	if abilityCount != 13 || abilityRandom < abilityCount {
		t.Fatalf("abilityCount %d random %d", abilityCount, abilityRandom)
	}
	seen := map[int]bool{}
	for i, n := range selectableAbilityNumbers() {
		if seen[n] || !isSelectableAbility(n) || abilityMenuIndex(n) != i {
			t.Fatalf("bad menu entry %d", n)
		}
		if got, ok := abilityFromMenuNumber(i + 1); !ok || got != n {
			t.Fatalf("menu number %d -> %d", i+1, got)
		}
		seen[n] = true
		if abilityName(n) == "ABILITY" || abilityDescription(n) == "" {
			t.Fatalf("ability %d has no name/description", n)
		}
	}
	if len(developedAbilitySelectionItems()) != len(selectableAbilityNumbers()) {
		t.Fatal("ability editor grid does not list every choice")
	}
	for _, item := range developedAbilitySelectionItems() {
		if item.Rect.Y+item.Rect.H > 715 {
			t.Fatalf("ability cell %d overlaps the description / BACK area: %+v", item.Number, item.Rect)
		}
	}
}

func TestDashBurstCooldownAndOncePerAirtime(t *testing.T) {
	p := abilityPlayer(1, 450, 1, abilityDash)
	abilityFrame(p, true, false, false, false)
	if math.Abs(p.VX-dashSpeed*Friction) > 1e-9 || p.AbilityCooldown != dashCooldownFrames || p.DashTime != dashTrailFrames {
		t.Fatalf("ground dash vx=%.3f cd=%d trail=%d", p.VX, p.AbilityCooldown, p.DashTime)
	}
	// Holding the key does nothing more, and a re-press during the cooldown
	// is ignored.
	abilityFrame(p, true, false, false, false)
	abilityFrame(p, false, false, false, false)
	vx := p.VX
	abilityFrame(p, true, false, false, false)
	if p.VX > vx {
		t.Fatalf("dash fired during cooldown: %.3f -> %.3f", vx, p.VX)
	}

	// Facing left dashes left.
	left := abilityPlayer(2, 450, 1, abilityDash)
	abilityFrame(left, true, true, false, false)
	if left.VX > -dashSpeed*0.9 {
		t.Fatalf("left dash vx=%.3f", left.VX)
	}

	// Air dash: once per airtime, even when the cooldown is over.
	air := abilityPlayer(3, 450, 1, abilityDash)
	air.Y, air.JumpNum, air.VY = 100, 1, 6
	abilityFrame(air, true, false, false, false)
	if !air.DashAirUsed || air.VY > 0 || air.VX < dashSpeed*0.9 {
		t.Fatalf("air dash vx=%.2f vy=%.2f used=%v", air.VX, air.VY, air.DashAirUsed)
	}
	air.AbilityCooldown = 0
	air.Y, air.VX = 100, 0
	abilityFrame(air, false, false, false, false)
	abilityFrame(air, true, false, false, false)
	if air.VX > 1 {
		t.Fatalf("second air dash in one airtime: vx=%.2f", air.VX)
	}
	// Landing refills the air dash.
	air.refillAbilityOnLanding()
	if air.DashAirUsed {
		t.Fatal("landing did not refill the air dash")
	}
	// The cooldown runs out after dashCooldownFrames.
	c := abilityPlayer(4, 450, 1, abilityDash)
	c.AbilityCooldown, c.AbilityCooldownMax = 2, dashCooldownFrames
	abilityFrame(c, false, false, false, false)
	abilityFrame(c, false, false, false, false)
	if c.AbilityCooldown != 0 {
		t.Fatalf("cooldown did not expire: %d", c.AbilityCooldown)
	}
}

func TestHeavyAndLightScaleKnockbackMovementAndJump(t *testing.T) {
	plain := bulletPushInWindow(abilityPlayer(1, 450, -1, abilityNone))
	heavy := bulletPushInWindow(abilityPlayer(2, 450, -1, abilityHeavy))
	light := bulletPushInWindow(abilityPlayer(3, 450, -1, abilityLight))
	if math.Abs(heavy-plain*heavyKnockbackScale) > 1e-9 || math.Abs(light-plain*lightKnockbackScale) > 1e-9 {
		t.Fatalf("bullet push plain %.3f heavy %.3f light %.3f", plain, heavy, light)
	}
	px, py := blastPushInWindow(abilityPlayer(4, 450, 1, abilityNone), 9)
	hx, hy := blastPushInWindow(abilityPlayer(5, 450, 1, abilityHeavy), 9)
	lx, ly := blastPushInWindow(abilityPlayer(6, 450, 1, abilityLight), 9)
	if px == 0 || math.Abs(hx-px*heavyKnockbackScale) > 1e-9 || math.Abs(hy-py*heavyKnockbackScale) > 1e-9 ||
		math.Abs(lx-px*lightKnockbackScale) > 1e-9 || math.Abs(ly-py*lightKnockbackScale) > 1e-9 {
		t.Fatalf("blast plain (%.2f,%.2f) heavy (%.2f,%.2f) light (%.2f,%.2f)", px, py, hx, hy, lx, ly)
	}

	run := func(ability int) float64 {
		p := abilityPlayer(7, 450, 1, ability)
		abilityFrame(p, false, false, true, false)
		return p.VX
	}
	jump := func(ability int) float64 {
		p := abilityPlayer(8, 450, 1, ability)
		abilityFrame(p, false, false, false, true)
		return p.VY - DefaultGravity
	}
	if math.Abs(run(abilityHeavy)-run(abilityNone)*heavyMoveScale) > 1e-9 || math.Abs(run(abilityLight)-run(abilityNone)*lightMoveScale) > 1e-9 {
		t.Fatal("HEAVY/LIGHT run speed")
	}
	if math.Abs(jump(abilityHeavy)-jump(abilityNone)*heavyJumpScale) > 1e-9 || math.Abs(jump(abilityLight)-jump(abilityNone)*lightJumpScale) > 1e-9 {
		t.Fatal("HEAVY/LIGHT jump")
	}
	// Burn ticks (applied in the player update) follow the scale too.
	if abilityBurnTickScale(abilityPlayer(9, 0, 1, abilityHeavy)) != heavyKnockbackScale {
		t.Fatal("burn ticks ignore HEAVY")
	}
}

// countAirJumps taps jump every other frame from the ground and counts how
// many jumps fired and the impulse of each.
func countAirJumps(ability int) []float64 {
	p := abilityPlayer(1, 450, 1, ability)
	var jumps []float64
	arena := Map{Gravity: DefaultGravity, Platforms: []Rect{{X: 0, Y: 300, W: 900, H: 24}}, LowestY: 600}
	// One-frame taps every 7 frames: gravity has slowed the previous jump
	// enough that each new impulse is a clear drop in vy.
	for frame := 0; frame < 30; frame++ {
		before := p.VY
		p.AIUp = frame%7 == 0
		p.Update(arena, nil, nil)
		if p.VY < before-1.5 {
			jumps = append(jumps, -(p.VY - DefaultGravity))
		}
	}
	return jumps
}

func TestTripleJumpGivesOneWeakerExtraJump(t *testing.T) {
	normal := countAirJumps(abilityNone)
	triple := countAirJumps(abilityTripleJump)
	if len(normal) != 2 || len(triple) != 3 {
		t.Fatalf("jumps normal=%v triple=%v", normal, triple)
	}
	if math.Abs(triple[2]-JumpPower*tripleJumpScale) > 1e-6 || triple[2] >= triple[1] || triple[2] >= triple[0] {
		t.Fatalf("third jump %.3f must be %.3f and weaker than %v", triple[2], JumpPower*tripleJumpScale, triple[:2])
	}
}

func TestShieldBlocksOneHitPerLife(t *testing.T) {
	p := abilityPlayer(1, 450, -1, abilityShield)
	if !abilityShieldReady(p) {
		t.Fatal("shield not ready at spawn")
	}
	if push := bulletPushInWindow(p); push != 0 || !p.AbilityShieldUsed || p.AbilityShieldBreak != shieldBreakFrames || p.HitNumber != 0 {
		t.Fatalf("first hit not blocked: push=%.2f used=%v hits=%d", push, p.AbilityShieldUsed, p.HitNumber)
	}
	// The rest of that same frame is still covered (e.g. a blast after the bullet).
	if dx, _ := blastPushInWindow(p, 9); dx != 0 {
		t.Fatalf("same-frame follow-up hit got through: %.2f", dx)
	}
	p.updateAbilityTimers()
	if push := bulletPushInWindow(p); push == 0 {
		t.Fatal("second hit was blocked too")
	}
	// Respawn restores it.
	p.Respawn(utilityArena())
	if !abilityShieldReady(p) {
		t.Fatal("shield not restored on respawn")
	}
	// An explosion is a hit as well.
	q := abilityPlayer(2, 450, 1, abilityShield)
	if dx, dy := blastPushInWindow(q, 9); dx != 0 || dy != 0 || !q.AbilityShieldUsed {
		t.Fatalf("blast not blocked: (%.2f,%.2f)", dx, dy)
	}
	// Your own rocket / grenade never spends it.
	self := abilityPlayer(3, 450, 1, abilityShield)
	if dx, _ := blastPushInWindow(self, 3); dx == 0 || self.AbilityShieldUsed {
		t.Fatal("own grenade spent the shield")
	}
	// Environmental windows never spend it.
	env := abilityPlayer(4, 450, 1, abilityShield)
	w := beginAbilityKnockbackWindow([]*Player{env})
	env.VX += 8
	endAbilityKnockbackWindow([]*Player{env}, w, false)
	if env.AbilityShieldUsed || env.VX != 8 {
		t.Fatal("chaos push spent the shield")
	}
}

func TestAnchorIgnoresKnockbackBlocksMoveShootAndTimesOut(t *testing.T) {
	p := abilityPlayer(1, 450, 1, abilityAnchor)
	abilityFrame(p, true, false, true, false)
	if !p.anchored() || p.VX != 0 {
		t.Fatalf("anchor did not plant: time=%d vx=%.2f", p.AnchorTime, p.VX)
	}
	if push := bulletPushInWindow(p); push != 0 {
		t.Fatalf("anchored player pushed by a bullet: %.2f", push)
	}
	if dx, dy := blastPushInWindow(p, 9); dx != 0 || dy != 0 {
		t.Fatalf("anchored player pushed by a blast: (%.2f,%.2f)", dx, dy)
	}
	if p.shootPressed() {
		t.Fatal("anchored player can shoot")
	}
	p.AIShoot = true
	if p.shootPressed() {
		t.Fatal("anchored player can shoot with the key held")
	}
	// Grenade input is frozen, not released into a throw.
	p.Grenades, p.GrenadeHeld = 3, true
	var grenades []Grenade
	updateGrenadeInput(p, &grenades)
	if len(grenades) != 0 || !p.GrenadeHeld {
		t.Fatal("anchored player threw a grenade")
	}
	// Walking / jumping is ignored while held.
	y := p.Y
	abilityFrame(p, true, false, true, true)
	if p.VX != 0 || p.Y < y-0.5 {
		t.Fatalf("anchored player moved: vx=%.2f y=%.2f->%.2f", p.VX, y, p.Y)
	}
	// Max duration, then cooldown, and holding does not re-anchor.
	for i := 0; i < anchorMaxFrames+5; i++ {
		abilityFrame(p, true, false, false, false)
	}
	if p.anchored() || p.AbilityCooldown == 0 {
		t.Fatalf("anchor exceeded its max duration: time=%d cd=%d", p.AnchorTime, p.AbilityCooldown)
	}
	abilityFrame(p, false, false, false, false)
	abilityFrame(p, true, false, false, false)
	if p.anchored() {
		t.Fatal("re-anchored during the cooldown")
	}
	// Releasing early also starts the cooldown; walking works again.
	q := abilityPlayer(2, 450, 1, abilityAnchor)
	abilityFrame(q, true, false, false, false)
	abilityFrame(q, false, false, true, false)
	if q.anchored() || q.AbilityCooldown != anchorCooldownFrames-0 || q.VX <= 0 {
		t.Fatalf("release: anchored=%v cd=%d vx=%.2f", q.anchored(), q.AbilityCooldown, q.VX)
	}
	// Cannot anchor in the air.
	air := abilityPlayer(3, 450, 1, abilityAnchor)
	air.Y, air.JumpNum = 100, 1
	abilityFrame(air, true, false, false, false)
	if air.anchored() {
		t.Fatal("anchored in mid-air")
	}
}

func TestGrenadierAddsTwoThrowablesAtSpawn(t *testing.T) {
	arena := utilityArena()
	p := abilityPlayer(1, 450, 1, abilityGrenadier)
	p.Respawn(arena)
	if p.Grenades != 3+grenadierBonusThrowables {
		t.Fatalf("grenadier spawned with %d throwables", p.Grenades)
	}
	p.PerkNumber = 8
	p.Respawn(arena)
	if p.Grenades != 3+2+grenadierBonusThrowables {
		t.Fatalf("grenadier + EXTRA NADES spawned with %d", p.Grenades)
	}
}

func TestScavengerCrateAmmoStacksWithExtraAmmo(t *testing.T) {
	const gun = 10
	base := NewWeapon(gun).Bullets
	bonus := int(math.Ceil(float64(base) * 0.333))
	pickup := func(perk, ability int) int {
		p := abilityPlayer(1, 450, 1, ability)
		p.AI = false
		p.PerkNumber = perk
		crates := []Crate{{X: p.X, Y: p.Y, Facing: 1, Weapon: gun}}
		updateCrates(utilityArena(), crates, []*Player{p}, []int{gun}, Rect{X: -20, Y: -40, W: 40, H: 40})
		if !crates[0].Dead || p.Weapon.Def.Number != gun {
			t.Fatalf("crate not picked up")
		}
		return p.Weapon.Bullets
	}
	if got := pickup(1, abilityNone); got != base {
		t.Fatalf("plain pickup %d want %d", got, base)
	}
	if got := pickup(1, abilityScavenger); got != base+bonus {
		t.Fatalf("scavenger pickup %d want %d", got, base+bonus)
	}
	if got := pickup(7, abilityScavenger); got != base+2*bonus {
		t.Fatalf("scavenger + EXTRA AMMO pickup %d want %d", got, base+2*bonus)
	}
}

func TestScavengerSeesCratesOnlyOnItsOwnScreen(t *testing.T) {
	scav := abilityPlayer(1, 100, 1, abilityScavenger)
	scav.AI = false
	other := abilityPlayer(2, 300, 1, abilityNone)
	other.AI = false
	bot := abilityPlayer(3, 500, 1, abilityScavenger)
	g := &Game{players: []*Player{scav, other, bot}}
	// Shared local screen: both humans watch; the scavenger's marker shows.
	if v := g.crateRevealViewers(); len(v) != 1 || v[0] != scav {
		t.Fatalf("shared screen viewers %v", v)
	}
	// A single human without the ability never sees a bot scavenger's info.
	solo := &Game{players: []*Player{other, bot}}
	if v := solo.crateRevealViewers(); len(v) != 0 {
		t.Fatal("bot scavenger revealed crates to the human")
	}
	solo2 := &Game{players: []*Player{scav, bot}}
	if v := solo2.crateRevealViewers(); len(v) != 1 || v[0] != scav {
		t.Fatal("solo scavenger does not see crates")
	}
}

func TestLegacyGearPerkConfigsMigrate(t *testing.T) {
	cases := map[int]int{
		legacyPerkFlashGoggles: abilityFlashGoggles, legacyPerkThermalGoggles: abilityThermalGoggles,
		legacyPerkGasMask: abilityGasMask, legacyPerkKevlar: abilityKevlar,
	}
	for perk, gear := range cases {
		got := migrateLegacyGearPerk(CustomPlayerConfig{Perk: perk})
		if got.Perk != 1 || got.Ability != gear {
			t.Fatalf("perk %d migrated to perk %d ability %d", perk, got.Perk, got.Ability)
		}
		// A chosen ability is kept; the stale gear perk is just dropped.
		kept := migrateLegacyGearPerk(CustomPlayerConfig{Perk: perk, Ability: abilityDash})
		if kept.Perk != 1 || kept.Ability != abilityDash {
			t.Fatalf("perk %d with ability: %+v", perk, kept)
		}
	}
	if got := migrateLegacyGearPerk(CustomPlayerConfig{Perk: 6, Ability: 77}); got.Perk != 6 || got.Ability != abilityNone {
		t.Fatalf("normal perk / bad ability migrated wrongly: %+v", got)
	}
	// A custom match started from an old in-memory config uses the gear.
	arena := OriginalMap1()
	g := &Game{maps: map[int]Map{1: arena}, arena: arena, customMap: 1, customMode: SourceGameModeNormal, customLives: 10, seenDeaths: map[int]int{}}
	g.initCustomPlayerSetup()
	for i := 0; i < 4; i++ {
		g.controlConfigs[i] = OriginalControls(i + 1)
	}
	g.customPlayers[0].Perk = legacyPerkKevlar
	g.startCustomGame()
	if g.players[0].PerkNumber != 1 || g.players[0].AbilityNumber != abilityKevlar {
		t.Fatalf("old kevlar perk config started as perk %d ability %d", g.players[0].PerkNumber, g.players[0].AbilityNumber)
	}
}

func TestAbilityKeyMigrationAvoidsConflicts(t *testing.T) {
	var all [4]Controls
	for i := range all {
		all[i] = OriginalControls(i + 1)
		all[i].Ability = 0 // saved before the ability key existed
	}
	all[1].Shoot = 220 // P2 rebound shoot onto P1's default ability key
	migrateAbilityKeys(&all)
	seen := map[int]bool{}
	for i, c := range all {
		if c.Ability <= 0 || !sourceOptionValidVK(c.Ability) {
			t.Fatalf("P%d got no valid ability key: %d", i+1, c.Ability)
		}
		for _, k := range c.keys() {
			if seen[k] {
				t.Fatalf("key %d bound twice after migration: %+v", k, all)
			}
			seen[k] = true
		}
	}
	if all[1].Ability != 85 || all[2].Ability != 107 || all[3].Ability != 102 {
		t.Fatalf("unconflicted defaults not kept: %+v", all)
	}
	// Defaults never collide with any other default key.
	seen = map[int]bool{}
	for id := 1; id <= 4; id++ {
		for _, k := range OriginalControls(id).keys() {
			if seen[k] {
				t.Fatalf("default key %d used twice", k)
			}
			seen[k] = true
		}
	}
	cfg := normalizeConfig(AppConfig{Controls: all})
	for i := range cfg.Controls {
		if cfg.Controls[i].Ability <= 0 {
			t.Fatal("normalizeConfig dropped the ability key")
		}
	}
	// The options panel exposes the key for rebinding.
	c := OriginalControls(1)
	setControlByIndex(&c, 6, 90)
	if controlByIndex(c, 6) != 90 || c.Ability != 90 {
		t.Fatal("ability key not rebindable")
	}
	if p, i, ok := optionsControlHitAt(50+108+10, 83.85+302.5+10); !ok || p != 0 || i != 6 {
		t.Fatalf("ability key hitbox p=%d i=%d ok=%v", p, i, ok)
	}
}

func TestOnlineValidatesAbilitySlot(t *testing.T) {
	base := CustomPlayerConfig{Name: "P", Color: 1, Shirt: 1, Hat: 1, Eyes: 1, Gun: 1, Perk: 7, Type: 1, Team: 1}
	for _, n := range selectableAbilityNumbers() {
		cfg := base
		cfg.Ability = n
		if _, err := sanitizeRemotePlayerConfig(cfg); err != nil {
			t.Fatalf("ability %d rejected: %v", n, err)
		}
	}
	for _, n := range []int{-1, abilityCount, abilityRandom - 1, abilityRandom + 1, 999} {
		cfg := base
		cfg.Ability = n
		if _, err := sanitizeRemotePlayerConfig(cfg); err == nil {
			t.Fatalf("ability %d accepted", n)
		}
	}
	for _, perk := range []int{legacyPerkFlashGoggles, legacyPerkThermalGoggles, legacyPerkGasMask, legacyPerkKevlar} {
		cfg := base
		cfg.Perk = perk
		if _, err := sanitizeRemotePlayerConfig(cfg); err == nil {
			t.Fatalf("legacy gear perk %d accepted online", perk)
		}
	}
	snap := netSnapshot{}
	snap.OnlineCustomPlayers[1].Ability = 77
	if err := validateNetSnapshot(snap); err == nil {
		t.Fatal("snapshot with an invalid ability config accepted")
	}

	g := &Game{onlinePerksEnabled: true}
	if g.onlineAllowedAbility(abilityDash, abilityNone) != abilityDash {
		t.Fatal("allowed ability stripped")
	}
	g.onlineBlockedAbilities[abilityDash] = true
	if g.onlineAllowedAbility(abilityDash, abilityNone) != abilityNone {
		t.Fatal("blocked ability kept")
	}
	for i := 0; i < 50; i++ {
		if got := g.onlineAllowedAbility(abilityRandom, abilityNone); got == abilityDash || got <= abilityNone || got >= abilityCount {
			t.Fatalf("RANDOM rolled %d", got)
		}
	}
	// RANDOM keeps an already rolled ability across lobby syncs.
	if g.onlineAllowedAbility(abilityRandom, abilityShield) != abilityShield {
		t.Fatal("RANDOM re-rolled on sync")
	}
	g.onlinePerksEnabled = false
	if g.onlineAllowedAbility(abilityShield, abilityNone) != abilityNone || g.onlineAllowedAbility(abilityRandom, abilityShield) != abilityNone {
		t.Fatal("PERKS OFF must also disable abilities")
	}
}

func TestAbilityKeyGoesThroughOnlineInputReplay(t *testing.T) {
	in := PlayerInputState{Left: true, Ability: true}
	if got := unpackNetInput(packNetInput(in)); got != in {
		t.Fatalf("ability bit lost: %+v", got)
	}
	var h netInputHistory
	var r netInputReplay
	now := time.Now()
	// A one-frame ability tap in the middle of a burst of frames.
	h.record(PlayerInputState{})
	h.record(PlayerInputState{Ability: true})
	h.record(PlayerInputState{})
	seq, frames := h.record(PlayerInputState{})
	r.receive(seq, frames, now)
	taps := 0
	for i := 0; i < 6; i++ {
		if unpackNetInput(r.take(now)).Ability {
			taps++
		}
	}
	if taps != 1 {
		t.Fatalf("ability tap replayed %d times", taps)
	}
	// Garbage high bits are dropped.
	r2 := netInputReplay{}
	r2.receive(1, []uint8{0xff}, now)
	if b := r2.take(now); b != netInputButtonMask {
		t.Fatalf("mask kept %08b", b)
	}
	// A remote player dashes from the replayed input.
	p := abilityPlayer(2, 450, 1, abilityDash)
	p.AI = false
	p.RemoteControlled = true
	p.RemoteInput = PlayerInputState{Ability: true}
	p.Update(utilityArena(), nil, nil)
	if p.AbilityCooldown != dashCooldownFrames {
		t.Fatal("remote ability input did not dash")
	}
}

func TestBotsUseDashAndAnchor(t *testing.T) {
	arena := utilityArena()
	// Knocked off the right side, falling: dash back toward the middle.
	bot := abilityPlayer(1, 950, 1, abilityDash)
	bot.Y, bot.JumpNum, bot.VY, bot.VX = 250, 0, 4, 6
	g := &Game{arena: arena, players: []*Player{bot}}
	aiUseAbility(bot, g)
	if !bot.AIAbility || !bot.AILeft || bot.AIRight {
		t.Fatalf("bot did not dash home: ability=%v left=%v right=%v", bot.AIAbility, bot.AILeft, bot.AIRight)
	}
	bot.Update(arena, nil, nil)
	if bot.VX > -dashSpeed*0.5 || !bot.DashAirUsed {
		t.Fatalf("recovery dash vx=%.2f", bot.VX)
	}
	// Safe in the middle with nothing incoming: no dash.
	calm := abilityPlayer(2, 450, 1, abilityDash)
	g2 := &Game{arena: arena, players: []*Player{calm}}
	aiUseAbility(calm, g2)
	if calm.AIAbility {
		t.Fatal("bot dashed for no reason")
	}

	// Anchor near the edge with a bullet incoming.
	anchor := abilityPlayer(3, 820, 1, abilityAnchor)
	shooter := abilityPlayer(4, 600, 1, abilityNone)
	g3 := &Game{arena: arena, players: []*Player{anchor, shooter}}
	g3.bullets = []Bullet{{Kind: BulletNormal, X: 700, Y: anchor.Y - 30, VX: 20, Facing: 1, OwnerID: 4, Firepower: 10}}
	aiUseAbility(anchor, g3)
	if !anchor.AIAbility || anchor.AIShoot || anchor.AILeft || anchor.AIRight {
		t.Fatal("bot did not anchor against the incoming shot")
	}
	// And against a grenade about to go off.
	anchor2 := abilityPlayer(5, 450, 1, abilityAnchor)
	g4 := &Game{arena: arena, players: []*Player{anchor2}}
	g4.grenades = []Grenade{{X: 470, Y: anchor2.Y - 30, Type: ThrowableDynamite, Fuse: throwableProfiles[ThrowableDynamite].Fuse - 5, OwnerID: 4}}
	aiUseAbility(anchor2, g4)
	if !anchor2.AIAbility {
		t.Fatal("bot did not anchor against a grenade")
	}
}

func TestCampaignEnemiesGetFixedValidAbilities(t *testing.T) {
	for set := 1; set <= 3; set++ {
		for level := 1; level <= 20; level++ {
			for index := 0; index < 2; index++ {
				a := campaignEnemyAbility(set, level, index)
				if a < abilityNone || a >= abilityCount || a != campaignEnemyAbility(set, level, index) {
					t.Fatalf("set %d level %d enemy %d ability %d", set, level, index, a)
				}
			}
		}
	}
	if campaignEnemyAbility(3, 1, 0) == abilityNone {
		t.Fatal("EXTRA mission 1 enemy has no ability")
	}
}
