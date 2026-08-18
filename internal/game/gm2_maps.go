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

var gm2MapDisplayNames = map[int]string{
	1:  "Safari Showdown",
	2:  "Polar Pwn4ge",
	3:  "Midnight Wood",
	4:  "Hovering Houses",
	5:  "Desert Destruction",
	6:  "Great Wall Brawl",
	7:  "Solar Shootout",
	8:  "Underwater Slaughter",
	9:  "Dessert Duel",
	10: "No Name",
	11: "Grim City",
	12: "Mushroom Mountain",
	13: "Jungle",
	14: "Ski Lift",
	15: "Space Station",
	16: "Avalon",
	17: "Venice",
	18: "Alien Planet",
	19: "Sub Base",
	20: "Highway",
	21: "Castle",
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
	platformRects, err := symbolRects(library, platformInst.Library)
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
