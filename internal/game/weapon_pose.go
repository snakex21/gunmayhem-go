package game

type WeaponPose struct {
	ShootX float64
	ShootY float64
	HandX  float64
	HandY  float64
}

var weaponPoseCatalog = map[int]WeaponPose{
	1:   {25, 2, 30, -5},
	2:   {25, 2, 30, -5},
	3:   {25, 2, 30, -5},
	4:   {23, 2, 28, -5},
	5:   {25, 2, 30, -5},
	6:   {23, 2, 28, -5},
	8:   {25, 2, 30, -5},
	9:   {12, 0, 42, -14},
	10:  {12, -1, 42, -14},
	11:  {10, 1, 42, -11},
	12:  {10, 3, 42, -11},
	13:  {2, -3, 48, -12},
	14:  {12, -1, 48, -14},
	15:  {12, 3, 49, -11},
	16:  {15, 1, 47, -11},
	17:  {12, 1, 48, -14},
	18:  {2, 0, 40, -12},
	19:  {30, -5, 45, -11},
	20:  {23, -1, 51, -11},
	21:  {10, 3, 42, -11},
	22:  {10, 1, 42, -11},
	23:  {15, 3, 47, -12},
	24:  {25, 2, 47, -10},
	25:  {10, 3, 42, -13},
	26:  {15, 3, 42, -12},
	27:  {15, 3, 47, -12},
	28:  {18, 2, 40, -12},
	29:  {18, -2, 52, -14},
	30:  {15, -6, 51, -12},
	31:  {20, 3, 43, -10},
	32:  {18, 2, 40, -12},
	33:  {10, 1, 42, -11},
	34:  {10, 3, 47, -11},
	35:  {20, 1, 42, -11},
	36:  {20, 1, 42, -11},
	37:  {10, -3, 42, -11},
	38:  {13, 1, 42, -11},
	39:  {12, -1, 40, -16},
	40:  {15, -3, 45, -16},
	41:  {12, -2, 50, -16},
	42:  {12, 3, 50, -12},
	43:  {7, 0, 45, -12},
	44:  {6, -3, 48, -12},
	45:  {16, 2, 58, -7},
	46:  {12, 3, 49, -15},
	47:  {23, 3, 52, -12},
	48:  {2, -1, 58, -11},
	49:  {11, -7, 53, -13},
	50:  {2, 2, 50, -10},
	51:  {12, -1, 52, -14},
	52:  {12, -4, 52, -14},
	53:  {2, 2, 55, -10},
	54:  {0, 2, 51, -4},
	55:  {-10, -3, 40, -22},
	56:  {16, -1, 60, -12},
	57:  {7, 1, 50, -13},
	58:  {7, 0, 42, -14},
	59:  {12, -1, 48, -14},
	60:  {12, 2, 51, -16},
	61:  {7, 1, 50, -17},
	62:  {23, -1, 47, -9},
	63:  {9, 1, 48, -15},
	64:  {14, 1, 50, -11},
	65:  {14, 1, 50, -8},
	66:  {14, 1, 45, -11},
	67:  {10, 3, 42, -13},
	68:  {10, 3, 42, -13},
	69:  {20, 3, 43, -10},
	70:  {7, 0, 42, -14},
	71:  {15, 3, 47, -12},
	72:  {9, 1, 48, -15},
	73:  {12, -1, 48, -14},
	74:  {23, -1, 51, -15},
	75:  {12, 2, 51, -16},
	76:  {7, 0, 42, -14},
	77:  {12, 3, 50, -16},
	78:  {15, -3, 53, -8},
	79:  {10, 1, 42, -11},
	80:  {20, 1, 42, -11},
	81:  {10, 1, 42, -11},
	82:  {12, 3, 49, -11},
	83:  {12, 3, 49, -8},
	84:  {2, 2, 54, -10},
	85:  {12, 3, 59, -10},
	86:  {12, -4, 52, -14},
	90:  {10, 0, 39, -10}, // HIMARS designator: support hand stays on the body
	198: {10, 0, 24, -8},  // Villar Perosa: both hands stay on the rear spade grips
	201: {4, -1, 38, -13}, // Chauchat: support hand under the barrel shroud
	213: {10, 0, 50, -11}, // G11: visible second hand under the long forebody
	214: {13, 2, 35, -10}, // M3 Grease Gun: support hand at the magazine well
}

func poseForWeapon(number int) WeaponPose {
	if p, ok := weaponPoseCatalog[number]; ok {
		return p
	}
	if spec, ok := developedWeaponSpec(number); ok {
		switch spec.Category {
		case DevelopedCategoryPistol:
			return WeaponPose{ShootX: 24, ShootY: 2, HandX: 31, HandY: -5}
		case DevelopedCategorySMG:
			return WeaponPose{ShootX: 13, ShootY: 2, HandX: 45, HandY: -11}
		case DevelopedCategoryRifle:
			return WeaponPose{ShootX: 10, ShootY: 0, HandX: 51, HandY: -13}
		case DevelopedCategoryLMG:
			return WeaponPose{ShootX: 4, ShootY: -1, HandX: 57, HandY: -14}
		case DevelopedCategorySniper:
			return WeaponPose{ShootX: 8, ShootY: 0, HandX: 54, HandY: -13}
		case DevelopedCategoryShotgun:
			return WeaponPose{ShootX: 7, ShootY: 1, HandX: 53, HandY: -12}
		case DevelopedCategoryLauncher:
			return WeaponPose{ShootX: 2, ShootY: -2, HandX: 58, HandY: -13}
		case DevelopedCategorySpecial:
			switch number {
			case 180, 181: // nail gun and flare gun are one-handed tools
				return WeaponPose{ShootX: 22, ShootY: 2, HandX: 32, HandY: -6}
			case 177: // crossbow
				return WeaponPose{ShootX: 6, ShootY: 0, HandX: 49, HandY: -12}
			case 178: // bow hand grips the riser; the other hand draws leftward
				return WeaponPose{ShootX: 6, ShootY: 0, HandX: -18, HandY: -6}
			default:
				return WeaponPose{ShootX: 8, ShootY: 0, HandX: 52, HandY: -12}
			}
		}
	}
	return WeaponPose{ShootX: 25, ShootY: 2, HandX: 30, HandY: -5}
}
