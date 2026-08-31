package game

import (
	"errors"
	"path/filepath"
	"strconv"
)

// Runtime map IDs must be globally unique because Map.Number is also used by
// asset caches and multiplayer snapshots. GM1 keeps its historical 1..13 IDs;
// GM2 source map N is represented internally as 100+N.
const gm2MapIDBase = 100

func gm2MapID(sourceNumber int) int { return gm2MapIDBase + sourceNumber }

func isGM2MapID(runtimeNumber int) bool {
	return runtimeNumber > gm2MapIDBase && runtimeNumber <= gm2MapIDBase+21
}

func gm2SourceMapNumber(runtimeNumber int) int {
	if !isGM2MapID(runtimeNumber) {
		return 0
	}
	return runtimeNumber - gm2MapIDBase
}

// Exact GM2 Symbol 1236 frame order. Frames/source map numbers 1..9 are
// the nine maps added by GM2; 10..21 are the returning GM1 arenas in GM2's
// own numbering. Keeping this order exact is important because source random
// selector 22 uses random(9)+1.
var gm2MapDisplayNames = map[int]string{
	1:  "Castle",
	2:  "Highway",
	3:  "Sub Base",
	4:  "Alien Planet",
	5:  "Venice",
	6:  "Avalon",
	7:  "Space Station",
	8:  "Ski Lift",
	9:  "Jungle",
	10: "No Name",
	11: "Dessert Duel",
	12: "Underwater Slaughter",
	13: "Solar Shootout",
	14: "Great Wall Brawl",
	15: "Mushroom Mountain",
	16: "Desert Destruction",
	17: "Hovering Houses",
	18: "Midnight Wood",
	19: "Polar Pwn4ge",
	20: "Grim City",
	21: "Safari Showdown",
}

// FFDec exports each GM2 scene timeline on one shared canvas, so its symbol
// registration point is constant across all 21 frames. Per-frame alpha bounds
// are not reliable for this: Jungle contains off-stage/dynamic artwork that is
// present in XFL bounds but absent from the raster, which made the old average
// shift its foreground by about 91 pixels. These origins are the medians of
// both XFL/alpha edge candidates across the complete source timelines.
var gm2SceneTimelineOrigins = [3][2]float64{
	{-457.525000, -292.739911}, // Symbol 1642 / scene1
	{-596.229869, -280.077259}, // Symbol 1690 / scene2
	{-792.775000, -449.559501}, // Symbol 1835 / foreground
}

func applyGM2SceneTimelineOrigins(back, mid, front *SourceRaster) {
	for i, raster := range []*SourceRaster{back, mid, front} {
		if raster == nil {
			continue
		}
		raster.Bounds.X = gm2SceneTimelineOrigins[i][0]
		raster.Bounds.Y = gm2SceneTimelineOrigins[i][1]
	}
}

// LoadGM2Map reconstructs one source arena from Symbol 1940. The root places
// `ground` at exactly (66, 210.2), matching GM1, but the frame numbering and
// platform symbols belong to a completely separate XFL namespace.
func LoadGM2Map(sourceNumber int) (Map, error) {
	if sourceNumber < 1 || sourceNumber > 21 {
		return Map{}, errors.New("GM2 XFL: invalid map number " + strconv.Itoa(sourceNumber))
	}
	library, err := findOriginalPathIn("gm2", "fla", "LIBRARY")
	if err != nil {
		return Map{}, err
	}
	instances, err := parseGroundInstances(filepath.Join(library, "Symbol 1940.xml"))
	if err != nil {
		return Map{}, err
	}
	frame := sourceNumber - 1
	items := make(map[string]xflInstance)
	for _, inst := range instances {
		if inst.Frame == frame {
			items[inst.Name] = inst
		}
	}
	platformInst, ok := items["platform"]
	if !ok {
		return Map{}, errors.New("GM2 XFL: missing platform instance for map frame " + strconv.Itoa(frame))
	}
	// Flash checks the actual filled vector shape. Several later GM2 maps use
	// stepped platform paths, so their axis-aligned bounds are not sufficient.
	platformRects, err := symbolCollisionRects(library, platformInst.Library)
	if err != nil {
		return Map{}, err
	}
	const groundX = 66.0
	const groundY = 210.2
	for i := range platformRects {
		platformRects[i] = transformRect(platformRects[i], platformInst.Matrix)
		platformRects[i].X += groundX
		platformRects[i].Y += groundY
	}
	spawnMin, spawnMax, err := instanceXBounds(library, items["spawnarea"], groundX)
	if err != nil {
		return Map{}, err
	}
	crateMin, crateMax, err := instanceXBounds(library, items["cratearea"], groundX)
	if err != nil {
		return Map{}, err
	}
	lowest, ok := items["lowest"]
	if !ok {
		return Map{}, errors.New("GM2 XFL: missing lowest instance for map frame " + strconv.Itoa(frame))
	}

	gravity := DefaultGravity
	// DefineSprite_1940/frame_13/DoAction.as sets _root.gravity = 0.6.
	if sourceNumber == 13 {
		gravity = 0.6
	}

	return Map{
		Number:      gm2MapID(sourceNumber),
		Gravity:     gravity,
		LowFriction: false,
		Platforms:   platformRects,
		SpawnMinX:   spawnMin,
		SpawnMaxX:   spawnMax,
		CrateMinX:   crateMin,
		CrateMaxX:   crateMax,
		LowestY:     groundY + lowest.Matrix.TY,
	}, nil
}
