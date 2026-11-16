package game

type CustomPlayerConfig struct {
	Name      string `cbor:",omitempty"`
	Color     int    `cbor:",omitempty"`
	Shirt     int    `cbor:",omitempty"`
	Hat       int    `cbor:",omitempty"`
	Eyes      int    `cbor:",omitempty"`
	Gun       int    `cbor:",omitempty"`
	Perk      int    `cbor:",omitempty"`
	Throwable int    `cbor:",omitempty"`
	// Ability is the ABILITY / GEAR slot (abilities.go): 0 none, 1..12, or
	// abilityRandom.
	Ability int `cbor:",omitempty"`
	Type    int `cbor:",omitempty"` // source playertype: 0 empty, 1 human, 2 AI
	Team    int `cbor:",omitempty"`
}

func sourceDefaultCustomPlayers() [4]CustomPlayerConfig {
	return [4]CustomPlayerConfig{
		{Name: "Player 1", Color: 2, Shirt: 1, Hat: 1, Eyes: 1, Gun: 1, Perk: 7, Throwable: ThrowableMk2, Type: 1, Team: 1},
		{Name: "Player 2", Color: 5, Shirt: 1, Hat: 1, Eyes: 1, Gun: 1, Perk: 7, Throwable: ThrowableMk2, Type: 1, Team: 2},
		{Name: "Player 3", Color: 8, Shirt: 1, Hat: 1, Eyes: 1, Gun: 1, Perk: 7, Throwable: ThrowableMk2, Type: 0, Team: 1},
		{Name: "Player 4", Color: 10, Shirt: 1, Hat: 1, Eyes: 1, Gun: 1, Perk: 7, Throwable: ThrowableMk2, Type: 0, Team: 2},
	}
}

const (
	menuHitPlayerSlotBase = 300
	playerSlotActionHuman = 1
	playerSlotActionAI    = 2
	playerSlotActionClear = 3
)

func playerSlotHit(slot, action int) int {
	return menuHitPlayerSlotBase + slot*10 + action
}

func decodePlayerSlotHit(hit int) (slot, action int, ok bool) {
	if hit < menuHitPlayerSlotBase || hit >= menuHitPlayerSlotBase+40 {
		return 0, 0, false
	}
	v := hit - menuHitPlayerSlotBase
	slot, action = v/10, v%10
	if slot < 0 || slot >= 4 || action < playerSlotActionHuman || action > playerSlotActionClear {
		return 0, 0, false
	}
	return slot, action, true
}

func (g *Game) initCustomPlayerSetup() {
	g.customPlayers = sourceDefaultCustomPlayers()
}

func (g *Game) updateCustomWarning() {
	if g.customWarningFrame <= 0 {
		return
	}
	g.customWarningFrame++
	limit := 25
	if g.customWarning == 1 {
		limit = 15
	}
	if g.customWarningFrame > limit {
		g.customWarning = 0
		g.customWarningFrame = 0
	}
}

func (g *Game) playCustomWarning(kind int) {
	g.customWarning = kind
	// warning.play() starts a clip stopped on frame1; its first visible source
	// frame on the following tick is frame2.
	g.customWarningFrame = 2
}

func (g *Game) activateCustomPlayerSlot(hit int) bool {
	slot, action, ok := decodePlayerSlotHit(hit)
	if !ok {
		return false
	}
	switch action {
	case playerSlotActionHuman:
		g.customPlayers[slot].Type = 1
	case playerSlotActionAI:
		g.customPlayers[slot].Type = 2
		if g.customPlayers[slot].Ability == abilityNone {
			// Bots get an ability unless the player picks one (or NONE)
			// for them afterwards.
			g.customPlayers[slot].Ability = abilityRandom
		}
	case playerSlotActionClear:
		g.customPlayers[slot].Type = 0
		g.customEditor[slot] = 0
		if g.customNameFocus == slot+1 {
			g.customNameFocus = 0
		}
	}
	return true
}

func (g *Game) activeCustomPlayerCount() int {
	n := 0
	for _, cfg := range g.customPlayers {
		if cfg.Type > 0 {
			n++
		}
	}
	return n
}
