package game

const (
	perkKnockbackResistance       = 10
	maxCustomPerk                 = perkKnockbackResistance
	knockbackResistanceMultiplier = 0.70
)

// applyCombatImpulse is called only for received combat knockback. It does not
// change movement, jumping, weapon recoil, hit attribution or Instagib death.
// Only the authoritative host/local game simulates combat; clients render its
// resulting velocity and perk from the existing snapshot fields.
func (p *Player) applyCombatImpulse(x, y float64) {
	if p == nil {
		return
	}
	multiplier := 1.0
	if p.PerkNumber == perkKnockbackResistance {
		multiplier = knockbackResistanceMultiplier
	}
	p.VX += x * multiplier
	p.VY += y * multiplier
}

// The original source has nine perk cells. Add a code-native tenth cell below
// the existing first column without mutating the parsed source cache.
func developedPerkSelectionItems() []customSelectionItem {
	source := sourceCustomSelectionItems()[customEditPerk]
	items := append([]customSelectionItem(nil), source...)
	var middle, bottom Rect
	for _, item := range source {
		if item.Number == 4 {
			middle = item.Rect
		}
		if item.Number == 7 {
			bottom = item.Rect
		}
	}
	if bottom.W > 0 && bottom.H > 0 && bottom.Y > middle.Y {
		cell := bottom
		cell.Y += bottom.Y - middle.Y
		items = append(items, customSelectionItem{Editor: customEditPerk, Number: perkKnockbackResistance, Rect: cell})
	}
	return items
}

func onlinePerkBlocklistRect(perk int) Rect {
	i := perk - 1
	return Rect{X: 170 + float64(i%3)*190, Y: 155 + float64(i/3)*70, W: 175, H: 58}
}
