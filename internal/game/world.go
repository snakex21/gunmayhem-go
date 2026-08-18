package game

const (
	ScreenWidth  = 900
	ScreenHeight = 600

	// Global values from Gunmayhem/scripts/frame_10/DoAction.as.
	DefaultGravity = 0.88
	Friction       = 0.93
	AirFriction    = 0.88
	MoveSpeed      = 0.7
	JumpPower      = 13.5
)

type Rect struct {
	X, Y, W, H float64
}

func (r Rect) ContainsX(x float64) bool {
	return x >= r.X && x <= r.X+r.W
}

func (r Rect) Contains(x, y float64) bool {
	return x >= r.X && x <= r.X+r.W && y >= r.Y && y <= r.Y+r.H
}

type Map struct {
	Number      int
	Gravity     float64
	LowFriction bool
	Platforms   []Rect
	SpawnMinX   float64
	SpawnMaxX   float64
	CrateMinX   float64
	CrateMaxX   float64
	LowestY     float64
}

// Map 1 reconstructed directly from the XFL:
// - ground instance: DOMDocument.xml -> Symbol 1444 at (66, 210.2)
// - map 1 platform symbol: Symbol 1393 at (-60.05, 45.95)
// - Symbol 1393 rectangles are stored in twips (20 twips = 1 px).
func OriginalMap1() Map {
	const ox = 66.0 - 60.05
	const oy = 210.2 + 45.95

	return Map{
		Number:      1,
		Gravity:     DefaultGravity,
		LowFriction: false,
		Platforms: []Rect{
			{X: ox + 0.00, Y: oy + 0.00, W: 909.85, H: 24.00},
			{X: ox + 533.45, Y: oy + 90.00, W: 250.05, H: 24.00},
			{X: ox + 75.95, Y: oy + 270.00, W: 785.95, H: 40.00},
			{X: ox + 559.70, Y: oy + 180.00, W: 172.40, H: 24.00},
			{X: ox + 119.45, Y: oy + 90.00, W: 251.80, H: 24.00},
			{X: ox + 174.45, Y: oy + 180.00, W: 167.80, H: 24.00},
		},
		// Symbol 1397 is ~499.9 px wide; map 1 places it at local x=143.
		SpawnMinX: 66.0 + 143.0,
		SpawnMaxX: 66.0 + 143.0 + 499.9,
		// Symbol 1399 cratearea is ~499.9 px wide and map 1 scales X by 1.362045.
		CrateMinX: 66.0 + 61.1,
		CrateMaxX: 66.0 + 61.1 + 499.9*1.3620452880859375,
		// map1 lowest: Symbol 1395 local y=294.55.
		LowestY: 210.2 + 294.55,
	}
}
