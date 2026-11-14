package game

import (
	"math"
	"sort"
	"sync"
)

// zombie_shop.go: the shop between waves and the mystery box. Purchases go
// through the existing systems (EquipWeapon, ThrowableType / Grenades,
// AbilityNumber); nothing here copies a weapon or an ability.
//
// One rule function (zombieCheckBuy) decides whether a purchase is allowed.
// Local players and the host's own player call zombieBuy directly; online
// guests send a netZombieReq and the host runs the same zombieBuy for them
// (zombie_net.go), so a guest can never buy without points, outside the
// break, or more than the caps.

const (
	zombieItemWeapon = iota + 1
	zombieItemThrowable
	zombieItemGear
	zombieItemHeal
	zombieItemAmmo
	zombieItemToken
)

const (
	zombieTabGuns = iota
	zombieTabThrowables
	zombieTabGear
	zombieTabSupplies
	zombieTabCount
)

var zombieTabLabels = []string{"GUNS", "THROWABLES", "GEAR", "SUPPLIES"}

const (
	zombieHealPrice      = 300
	zombieTokenPrice     = 1500
	zombieBoxPrice       = 950
	zombieMaxGrenades    = 9
	zombieThrowableStep  = 3
	zombieBoxRollFrames  = 75
	zombieBoxOfferFrames = 180
	zombieBoxLeaveFrames = 90
	zombieBoxReachX      = 44.0
	zombieBoxReachY      = 26.0
)

// Feedback codes (ZombiePlayerState.Msg).
const (
	zombieMsgNone = iota
	zombieMsgBought
	zombieMsgNoPoints
	zombieMsgOwned
	zombieMsgFull
	zombieMsgNoGun
	zombieMsgClosed
	zombieMsgBoxBusy
	zombieMsgBoxMoved
	zombieMsgTaken
	zombieMsgDown
	zombieMsgDoorOpened // step 2: doors (zombie_zones.go)
	zombieMsgDoorLocked
	zombieMsgDoorNoPath
)

var zombieMsgText = map[int]string{
	zombieMsgBought:     "BOUGHT",
	zombieMsgNoPoints:   "NOT ENOUGH POINTS",
	zombieMsgOwned:      "ALREADY OWNED",
	zombieMsgFull:       "ALREADY FULL",
	zombieMsgNoGun:      "NO BOUGHT GUN",
	zombieMsgClosed:     "SHOP IS CLOSED",
	zombieMsgBoxBusy:    "BOX IS BUSY",
	zombieMsgBoxMoved:   "THE BOX MOVED!",
	zombieMsgTaken:      "WEAPON TAKEN",
	zombieMsgDown:       "YOU ARE DOWN",
	zombieMsgDoorOpened: "DOOR OPENED",
	zombieMsgDoorLocked: "LOCKED UNTIL A LATER WAVE",
	zombieMsgDoorNoPath: "CANNOT USE THE DOOR NOW",
}

type zombieShopItem struct {
	ID    int
	Tab   int
	Kind  int
	Value int // weapon number, throwable kind or ability
	Price int // 0 for ammo: depends on the gun
	Name  string
	Info  string
}

// zombieShopGuns: the wall-buy arsenal, cheapest first. Everything else
// (every pistol to the de-materializer and the developed weapons) comes out
// of the mystery box.
var zombieShopGuns = []struct{ number, price int }{
	{5, 350}, {6, 500}, {12, 800}, {13, 900}, {22, 1000}, {11, 1000},
	{19, 1100}, {17, 1100}, {10, 1200}, {85, 1400}, {59, 1500}, {63, 1600},
	{14, 1800}, {57, 2000}, {65, 2400}, {55, 3000},
	// arsenal2: the bat one-hits wave-1 walkers; the blower is crowd control
	// (it pushes zombies off ledges but deals no damage). The harpoon is
	// box-only: reeling zombies in is rarely what you want.
	{weaponBaseballBat, 450}, {weaponLeafBlower, 700},
}

var zombieThrowablePrices = [throwableTypeCount]int{
	ThrowableDynamite: 300, ThrowableMk2: 300, ThrowableF1: 350, ThrowableStielhand: 400,
	ThrowableRGD5: 300, ThrowableMolotov: 450, ThrowableShock: 350, ThrowableFlashbang: 300,
	ThrowableSmoke: 200, ThrowableTearGas: 350,
}

var zombieGearPrices = map[int]int{
	abilityDash: 750, abilityAnchor: 900, abilityTripleJump: 750, abilityHeavy: 1000,
	abilityLight: 500, abilityShield: 1250, abilityGrenadier: 800, abilityFlashGoggles: 400,
	abilityThermalGoggles: 400, abilityGasMask: 500, abilityKevlar: 1000,
	abilityBallisticShield: 1100,
}

var (
	zombieShopOnce  sync.Once
	zombieShopList  []zombieShopItem
	zombieShopByTab [zombieTabCount][]zombieShopItem
	zombieBoxPool   []int
)

// zombieShopLauncher is the developed grenade launcher sold in the shop (0:
// none in this build).
func zombieShopLauncher() int {
	for _, n := range allDevelopedWeaponNumbers() {
		spec, ok := developedWeaponSpec(n)
		if ok && spec.Category == DevelopedCategoryLauncher && spec.Projectile == DevelopedProjectileGrenade {
			return n
		}
	}
	return 0
}

func buildZombieShop() {
	add := func(it zombieShopItem) {
		it.ID = len(zombieShopList)
		zombieShopList = append(zombieShopList, it)
	}
	guns := append([]struct{ number, price int }(nil), zombieShopGuns...)
	if n := zombieShopLauncher(); n != 0 {
		guns = append(guns, struct{ number, price int }{n, 2500})
	}
	sort.SliceStable(guns, func(i, j int) bool { return guns[i].price < guns[j].price })
	for _, gn := range guns {
		def, ok := WeaponByNumber(gn.number)
		if !ok {
			continue
		}
		info := "Weapon + spare magazines."
		switch gn.number {
		case weaponBaseballBat:
			info = "Melee bat: big knockback, one-hits early walkers."
		case weaponLeafBlower:
			info = "Blows zombies off ledges. No damage. Refuel = ammo."
		}
		add(zombieShopItem{Tab: zombieTabGuns, Kind: zombieItemWeapon, Value: gn.number, Price: gn.price,
			Name: def.DisplayName(), Info: info})
	}
	for kind := 0; kind < throwableTypeCount; kind++ {
		add(zombieShopItem{Tab: zombieTabThrowables, Kind: zombieItemThrowable, Value: kind, Price: zombieThrowablePrices[kind],
			Name: throwableName(kind) + " x3", Info: "+3 throwables (max 9). Another type replaces yours."})
	}
	gear := make([]int, 0, len(zombieGearPrices))
	for a := range zombieGearPrices {
		gear = append(gear, a)
	}
	sort.Ints(gear)
	for _, a := range gear {
		add(zombieShopItem{Tab: zombieTabGear, Kind: zombieItemGear, Value: a, Price: zombieGearPrices[a],
			Name: abilityName(a), Info: abilityDescription(a)})
	}
	add(zombieShopItem{Tab: zombieTabSupplies, Kind: zombieItemHeal, Price: zombieHealPrice, Name: "MEDKIT", Info: "Heal to full HP."})
	add(zombieShopItem{Tab: zombieTabSupplies, Kind: zombieItemAmmo, Name: "AMMO REFILL", Info: "All spare magazines of your bought gun."})
	add(zombieShopItem{Tab: zombieTabSupplies, Kind: zombieItemToken, Price: zombieTokenPrice, Name: "REVIVE TOKEN",
		Info: "When you go down, you get up by yourself after 3 s (max 3)."})
	for _, it := range zombieShopList {
		zombieShopByTab[it.Tab] = append(zombieShopByTab[it.Tab], it)
	}

	// Mystery box: every non-pistol weapon of the arsenal, including the
	// developed ones whose art is packaged.
	for _, n := range organizedGunLibraryNumbers(0) {
		if n <= 9 {
			continue
		}
		if spec, ok := developedWeaponSpec(n); ok {
			if spec.Category == DevelopedCategoryPistol || spec.Category == DevelopedCategoryMelee || spec.Projectile == DevelopedProjectilePortal || !customWeaponAssetAvailable(n) {
				continue
			}
		}
		zombieBoxPool = append(zombieBoxPool, n)
	}
}

func zombieShopItems() []zombieShopItem {
	zombieShopOnce.Do(buildZombieShop)
	return zombieShopList
}

func zombieShopTab(tab int) []zombieShopItem {
	zombieShopOnce.Do(buildZombieShop)
	if tab < 0 || tab >= zombieTabCount {
		return nil
	}
	return zombieShopByTab[tab]
}

func zombieShopItemByID(id int) (zombieShopItem, bool) {
	items := zombieShopItems()
	if id < 0 || id >= len(items) {
		return zombieShopItem{}, false
	}
	return items[id], true
}

func zombieMysteryPool() []int {
	zombieShopOnce.Do(buildZombieShop)
	return zombieBoxPool
}

// zombieWeaponValue is a weapon's shop price, or an estimate by category
// for box-only weapons (ammo refills cost half of it).
func zombieWeaponValue(number int) int {
	for _, gn := range zombieShopGuns {
		if gn.number == number {
			return gn.price
		}
	}
	if number != 0 && number == zombieShopLauncher() {
		return 2500
	}
	cat := sourceWeaponCategory(number)
	if spec, ok := developedWeaponSpec(number); ok {
		cat = spec.Category
	}
	switch cat {
	case DevelopedCategoryPistol:
		return 400
	case DevelopedCategorySMG:
		return 1100
	case DevelopedCategoryShotgun:
		return 1200
	case DevelopedCategorySniper:
		return 1400
	case DevelopedCategoryLMG:
		return 2200
	case DevelopedCategoryLauncher:
		return 2600
	case DevelopedCategorySpecial:
		return 2400
	}
	return 1400
}

func zombieAmmoPrice(number int) int {
	if number == 0 {
		return 0
	}
	p := zombieWeaponValue(number) / 2
	p = int(math.Round(float64(p)/50) * 50)
	return max(p, 150)
}

// zombieReserveMags: spare magazines that come with a weapon (about 90
// rounds, 2..8 magazines).
func zombieReserveMags(number int) int {
	def, ok := WeaponByNumber(number)
	if !ok || def.Bullets <= 0 {
		return 2
	}
	n := int(math.Ceil(90 / float64(def.Bullets)))
	return min(max(n, 2), 8)
}

func (it zombieShopItem) priceFor(zp *ZombiePlayerState) int {
	if it.Kind == zombieItemAmmo {
		if zp == nil {
			return 0
		}
		return zombieAmmoPrice(zp.Gun)
	}
	return it.Price
}

// zombieCheckBuy is THE purchase rule (local, host and validated guest
// requests). It returns zombieMsgBought when the purchase is allowed.
func (g *Game) zombieCheckBuy(p *Player, it zombieShopItem) int {
	m := g.zombie
	if m == nil || p == nil {
		return zombieMsgClosed
	}
	zp := m.playerState(p.ID)
	if zp == nil {
		return zombieMsgClosed
	}
	if m.hud.Phase != zombiePhaseBreak || !zp.ShopOpen {
		return zombieMsgClosed
	}
	if zp.State != zombiePlayerAlive {
		return zombieMsgDown
	}
	switch it.Kind {
	case zombieItemWeapon:
		if zp.Gun == it.Value && zp.Reserve >= zombieReserveMags(it.Value) && p.Weapon.Def.Number == it.Value {
			return zombieMsgOwned
		}
	case zombieItemThrowable:
		if p.ThrowableType == it.Value && p.Grenades >= zombieMaxGrenades {
			return zombieMsgFull
		}
	case zombieItemGear:
		if p.AbilityNumber == it.Value {
			return zombieMsgOwned
		}
	case zombieItemHeal:
		if zp.HP >= zp.MaxHP {
			return zombieMsgFull
		}
	case zombieItemAmmo:
		if zp.Gun == 0 {
			return zombieMsgNoGun
		}
		if zp.Reserve >= zombieReserveMags(zp.Gun) && p.Weapon.Def.Number == zp.Gun && p.Weapon.Bullets >= p.Weapon.Def.Bullets {
			return zombieMsgFull
		}
	case zombieItemToken:
		if zp.SelfRevives >= zombieMaxSelfRevives {
			return zombieMsgFull
		}
	default:
		return zombieMsgClosed
	}
	if zp.Points < it.priceFor(zp) {
		return zombieMsgNoPoints
	}
	return zombieMsgBought
}

// zombieBuy applies a purchase when zombieCheckBuy allows it. Host / local
// only. Returns the feedback code (also stored in the player's Msg).
func (g *Game) zombieBuy(p *Player, itemID int) int {
	it, ok := zombieShopItemByID(itemID)
	if !ok {
		return zombieMsgClosed
	}
	code := g.zombieCheckBuy(p, it)
	zp := g.zombie.playerState(p.ID)
	if zp != nil {
		zp.Msg, zp.MsgTimer = code, 75
	}
	if code != zombieMsgBought {
		return code
	}
	price := it.priceFor(zp)
	switch it.Kind {
	case zombieItemWeapon:
		zp.Gun = it.Value
		zp.Reserve = zombieReserveMags(it.Value)
		p.EquipWeapon(it.Value)
	case zombieItemThrowable:
		if p.ThrowableType != it.Value {
			p.ThrowableType = it.Value
			p.Grenades = zombieThrowableStep
		} else {
			p.Grenades = min(p.Grenades+zombieThrowableStep, zombieMaxGrenades)
		}
	case zombieItemGear:
		p.AbilityNumber = it.Value
		p.resetAbilityForLife()
	case zombieItemHeal:
		zp.HP = zp.MaxHP
	case zombieItemAmmo:
		zp.Reserve = zombieReserveMags(zp.Gun)
		p.EquipWeapon(zp.Gun)
	case zombieItemToken:
		zp.SelfRevives++
	}
	zp.Points -= price
	g.zombie.addPopIn(zp.Zone, p.X, p.Y-90, -price, zombiePopSpend)
	g.playSourceSFX("bolt2.wav", true)
	return code
}

// zombieSetShopOpen opens / closes a player's shop during the break. Closing
// marks the player READY.
func (g *Game) zombieSetShopOpen(p *Player, open bool) {
	m := g.zombie
	if m == nil || p == nil {
		return
	}
	zp := m.playerState(p.ID)
	if zp == nil || m.hud.Phase != zombiePhaseBreak || zp.State != zombiePlayerAlive {
		return
	}
	zp.ShopOpen = open
	zp.Ready = !open
}

// zombieBotShop: an AI teammate spends its points at the start of a break:
// the best gun it can afford that beats the current one, then a medkit and
// ammo.
func (g *Game) zombieBotShop(p *Player, zp *ZombiePlayerState) {
	zp.ShopOpen = true
	defer func() { zp.ShopOpen = false }()
	current := zombieWeaponValue(zp.Gun)
	if zp.Gun == 0 {
		current = 0
	}
	best := -1
	for _, it := range zombieShopTab(zombieTabGuns) {
		if isLeafBlower(it.Value) {
			continue // no damage: bots leave crowd control to the humans
		}
		if it.Price > current && it.Price <= zp.Points && g.zombieCheckBuy(p, it) == zombieMsgBought {
			best = it.ID
		}
	}
	if best >= 0 {
		g.zombieBuy(p, best)
	}
	for _, it := range zombieShopTab(zombieTabSupplies) {
		if it.Kind == zombieItemHeal && zp.HP < zp.MaxHP*7/10 && g.zombieCheckBuy(p, it) == zombieMsgBought {
			g.zombieBuy(p, it.ID)
		}
		if it.Kind == zombieItemAmmo && zp.Gun != 0 && zp.Reserve < 2 && g.zombieCheckBuy(p, it) == zombieMsgBought {
			g.zombieBuy(p, it.ID)
		}
	}
}

// --- mystery box -------------------------------------------------------------------

const (
	zombieBoxIdle = iota
	zombieBoxRolling
	zombieBoxOffer
	zombieBoxLeaving // the roll came up with the "bye bye" teddy: refund, then move
)

// placeBox puts the box on a random spot of an unlocked zone (multi-zone
// arenas: every unlocked zone's spots), never on the spot it left (avoid is
// the index into that list, -1 none).
func (m *zombieMatch) placeBox(avoid int) {
	type spot struct {
		zone int
		pt   zombiePoint
	}
	var spots []spot
	for _, z := range m.arena.Zones {
		if !m.zoneUnlocked(z.ID) {
			continue
		}
		for _, pt := range z.BoxSpots {
			spots = append(spots, spot{z.ID, pt})
		}
	}
	if len(spots) == 0 {
		return
	}
	// avoid >= 0: not the spot the box is leaving (compared by place: the
	// list grows when zones unlock, so indices are not stable).
	old := m.hud.Box
	i := m.rng.Intn(len(spots))
	if avoid >= 0 && len(spots) > 1 && spots[i].zone == old.Zone && spots[i].pt.X == old.X && spots[i].pt.Y == old.Y {
		i = (i + 1 + m.rng.Intn(len(spots)-1)) % len(spots)
	}
	s := spots[i]
	m.hud.Box = ZombieBoxState{Spot: i, X: s.pt.X, Y: s.pt.Y, Zone: s.zone, State: zombieBoxIdle, Uses: 0}
	m.boxMoveAt = 4 + m.rng.Intn(5)
}

func (m *zombieMatch) boxInReach(p *Player) bool {
	if m == nil || p == nil || !p.Active {
		return false
	}
	b := m.hud.Box
	if b.State != zombieBoxIdle && b.State != zombieBoxOffer {
		return false
	}
	if m.multiZone() && m.playerZone(p) != b.Zone {
		return false
	}
	return math.Abs(p.X-b.X) <= zombieBoxReachX && math.Abs(p.Y-b.Y) <= zombieBoxReachY
}

func (m *zombieMatch) rollBoxWeapon(exclude int) int {
	pool := zombieMysteryPool()
	if len(pool) == 0 {
		return 10
	}
	for i := 0; i < 8; i++ {
		w := pool[m.rng.Intn(len(pool))]
		if w != exclude {
			return w
		}
	}
	return pool[0]
}

// zombieUseBox: DOWN at the box. Idle: pay and roll. Offer: the roller
// takes the weapon.
func (g *Game) zombieUseBox(p *Player) {
	m := g.zombie
	zp := m.playerState(p.ID)
	if zp == nil || zp.State != zombiePlayerAlive {
		return
	}
	box := &m.hud.Box
	switch box.State {
	case zombieBoxIdle:
		if zp.Points < zombieBoxPrice {
			zp.Msg, zp.MsgTimer = zombieMsgNoPoints, 75
			return
		}
		zp.Points -= zombieBoxPrice
		m.addPopIn(box.Zone, box.X, box.Y-70, -zombieBoxPrice, zombiePopSpend)
		box.Uses++
		box.State = zombieBoxRolling
		box.Timer = zombieBoxRollFrames
		box.OwnerID = p.ID
		box.Weapon = m.rollBoxWeapon(0)
		g.playSourceSFX("bolt2.wav", true)
	case zombieBoxOffer:
		if box.OwnerID != p.ID {
			zp.Msg, zp.MsgTimer = zombieMsgBoxBusy, 60
			return
		}
		zp.Gun = box.Weapon
		zp.Reserve = zombieReserveMags(box.Weapon)
		p.EquipWeapon(box.Weapon)
		zp.Msg, zp.MsgTimer = zombieMsgTaken, 60
		box.State = zombieBoxIdle
		box.OwnerID = 0
		box.Weapon = 0
		g.playSourceSFX("bolt2.wav", true)
	}
}

func (g *Game) updateZombieBox() {
	m := g.zombie
	box := &m.hud.Box
	switch box.State {
	case zombieBoxRolling:
		box.Timer--
		if box.Timer%4 == 0 {
			box.Weapon = m.rollBoxWeapon(box.Weapon)
		}
		if box.Timer <= 0 {
			if box.Uses >= m.boxMoveAt {
				box.State = zombieBoxLeaving
				box.Timer = zombieBoxLeaveFrames
				box.Weapon = 0
				if zp := m.playerState(box.OwnerID); zp != nil {
					zp.Points += zombieBoxPrice // refunded, like the teddy bear
					zp.Msg, zp.MsgTimer = zombieMsgBoxMoved, 90
				}
				return
			}
			box.State = zombieBoxOffer
			box.Timer = zombieBoxOfferFrames
			exclude := 0
			if zp := m.playerState(box.OwnerID); zp != nil {
				exclude = zp.Gun
			}
			box.Weapon = m.rollBoxWeapon(exclude)
		}
	case zombieBoxOffer:
		box.Timer--
		if box.Timer <= 0 {
			box.State = zombieBoxIdle
			box.OwnerID = 0
			box.Weapon = 0
		}
	case zombieBoxLeaving:
		box.Timer--
		if box.Timer <= 0 {
			m.placeBox(box.Spot)
		}
	}
}
