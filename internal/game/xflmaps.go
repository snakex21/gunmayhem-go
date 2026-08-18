package game

import (
	"encoding/xml"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type xflMatrix struct {
	A, B, C, D float64
	TX, TY     float64
}

type xflInstance struct {
	Frame    int
	Duration int
	Name     string
	Library  string
	Matrix   xflMatrix
}

var numberRE = regexp.MustCompile(`-?\d+`)

func LoadOriginalMaps() (map[int]Map, error) {
	library, err := findOriginalPath("fla", "LIBRARY")
	if err != nil {
		return nil, err
	}

	instances, err := parseGroundInstances(filepath.Join(library, "Symbol 1444.xml"))
	if err != nil {
		return nil, err
	}

	byFrame := make(map[int]map[string]xflInstance)
	for _, inst := range instances {
		if byFrame[inst.Frame] == nil {
			byFrame[inst.Frame] = make(map[string]xflInstance)
		}
		byFrame[inst.Frame][inst.Name] = inst
	}

	maps := make(map[int]Map)
	for frame := 0; frame < 13; frame++ {
		items := byFrame[frame]
		platformInst, ok := items["platform"]
		if !ok {
			return nil, errors.New("XFL: missing platform instance for map frame " + strconv.Itoa(frame))
		}
		platformRects, err := symbolRects(library, platformInst.Library)
		if err != nil {
			return nil, err
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
			return nil, err
		}
		crateMin, crateMax, err := instanceXBounds(library, items["cratearea"], groundX)
		if err != nil {
			return nil, err
		}
		lowest, ok := items["lowest"]
		if !ok {
			return nil, errors.New("XFL: missing lowest instance for map frame " + strconv.Itoa(frame))
		}

		gravity := DefaultGravity
		// Symbol 1444 frame index 3 contains `_root.gravity = 0.6`.
		if frame == 3 {
			gravity = 0.6
		}

		maps[frame+1] = Map{
			Number:      frame + 1,
			Gravity:     gravity,
			LowFriction: false,
			Platforms:   platformRects,
			SpawnMinX:   spawnMin,
			SpawnMaxX:   spawnMax,
			CrateMinX:   crateMin,
			CrateMaxX:   crateMax,
			LowestY:     groundY + lowest.Matrix.TY,
		}
	}
	return maps, nil
}

// LoadOriginalMap parses only the requested source frame. Main Menu needs map1
// for its background/demo state; the other twelve maps are deferred until a
// campaign/custom/test match actually selects them.
func LoadOriginalMap(number int) (Map, error) {
	if number < 1 || number > 13 {
		return Map{}, errors.New("XFL: invalid map number " + strconv.Itoa(number))
	}
	library, err := findOriginalPath("fla", "LIBRARY")
	if err != nil {
		return Map{}, err
	}
	instances, err := parseGroundInstances(filepath.Join(library, "Symbol 1444.xml"))
	if err != nil {
		return Map{}, err
	}
	frame := number - 1
	items := make(map[string]xflInstance)
	for _, inst := range instances {
		if inst.Frame == frame {
			items[inst.Name] = inst
		}
	}
	platformInst, ok := items["platform"]
	if !ok {
		return Map{}, errors.New("XFL: missing platform instance for map frame " + strconv.Itoa(frame))
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
		return Map{}, errors.New("XFL: missing lowest instance for map frame " + strconv.Itoa(frame))
	}
	gravity := DefaultGravity
	if frame == 3 {
		gravity = 0.6
	}
	return Map{
		Number: number, Gravity: gravity, LowFriction: false,
		Platforms: platformRects,
		SpawnMinX: spawnMin, SpawnMaxX: spawnMax,
		CrateMinX: crateMin, CrateMaxX: crateMax,
		LowestY: groundY + lowest.Matrix.TY,
	}, nil
}

func parseGroundInstances(path string) ([]xflInstance, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	currentFrame := -1
	currentDuration := 1
	frameDepth := 0
	var current *xflInstance
	instDepth := 0
	instances := make([]xflInstance, 0, 52)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "DOMFrame":
				if currentFrame < 0 {
					currentFrame = intAttr(t.Attr, "index", -1)
					currentDuration = intAttr(t.Attr, "duration", 1)
					if currentDuration < 1 {
						currentDuration = 1
					}
					frameDepth = 1
				} else {
					frameDepth++
				}
			case "DOMSymbolInstance":
				name := stringAttr(t.Attr, "name", "")
				if currentFrame >= 0 && (name == "platform" || name == "spawnarea" || name == "cratearea" || name == "lowest") {
					current = &xflInstance{
						Frame:    currentFrame,
						Duration: currentDuration,
						Name:     name,
						Library:  stringAttr(t.Attr, "libraryItemName", ""),
						Matrix:   xflMatrix{A: 1, D: 1},
					}
					instDepth = 1
				} else if current != nil {
					instDepth++
				}
			case "Matrix":
				if current != nil {
					current.Matrix = xflMatrix{
						A:  floatAttr(t.Attr, "a", 1),
						B:  floatAttr(t.Attr, "b", 0),
						C:  floatAttr(t.Attr, "c", 0),
						D:  floatAttr(t.Attr, "d", 1),
						TX: floatAttr(t.Attr, "tx", 0),
						TY: floatAttr(t.Attr, "ty", 0),
					}
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "DOMSymbolInstance":
				if current != nil {
					instDepth--
					if instDepth == 0 {
						for f := current.Frame; f < current.Frame+current.Duration; f++ {
							copy := *current
							copy.Frame = f
							copy.Duration = 1
							instances = append(instances, copy)
						}
						current = nil
					}
				}
			case "DOMFrame":
				if frameDepth > 0 {
					frameDepth--
					if frameDepth == 0 {
						currentFrame = -1
						currentDuration = 1
					}
				}
			}
		}
	}
	return instances, nil
}

func symbolRects(libraryDir, libraryName string) ([]Rect, error) {
	path := filepath.Join(libraryDir, libraryName+".xml")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	var rects []Rect
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "Edge" {
			continue
		}
		edges := stringAttr(start.Attr, "edges", "")
		nums := numberRE.FindAllString(edges, -1)
		if len(nums) < 8 || len(nums)%2 != 0 {
			continue
		}
		minX, minY := math.Inf(1), math.Inf(1)
		maxX, maxY := math.Inf(-1), math.Inf(-1)
		for i := 0; i+1 < len(nums); i += 2 {
			xi, _ := strconv.Atoi(nums[i])
			yi, _ := strconv.Atoi(nums[i+1])
			x := float64(xi) / 20
			y := float64(yi) / 20
			minX = math.Min(minX, x)
			maxX = math.Max(maxX, x)
			minY = math.Min(minY, y)
			maxY = math.Max(maxY, y)
		}
		if maxX > minX && maxY > minY {
			rects = append(rects, Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY})
		}
	}
	if len(rects) == 0 {
		return nil, errors.New("XFL: no rectangles in " + libraryName)
	}
	return rects, nil
}

func instanceXBounds(libraryDir string, inst xflInstance, groundX float64) (float64, float64, error) {
	if inst.Library == "" {
		return 0, 0, errors.New("XFL: missing instance used for X bounds")
	}
	rects, err := symbolRects(libraryDir, inst.Library)
	if err != nil {
		return 0, 0, err
	}
	minX := math.Inf(1)
	maxX := math.Inf(-1)
	for _, r := range rects {
		t := transformRect(r, inst.Matrix)
		minX = math.Min(minX, t.X)
		maxX = math.Max(maxX, t.X+t.W)
	}
	return groundX + minX, groundX + maxX, nil
}

func transformRect(r Rect, m xflMatrix) Rect {
	corners := [4][2]float64{
		{r.X, r.Y},
		{r.X + r.W, r.Y},
		{r.X, r.Y + r.H},
		{r.X + r.W, r.Y + r.H},
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, p := range corners {
		x := m.A*p[0] + m.C*p[1] + m.TX
		y := m.B*p[0] + m.D*p[1] + m.TY
		minX, maxX = math.Min(minX, x), math.Max(maxX, x)
		minY, maxY = math.Min(minY, y), math.Max(maxY, y)
	}
	return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
}

func findOriginalPath(parts ...string) (string, error) {
	var starts []string
	if exe, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		starts = append(starts, cwd)
	}

	seen := map[string]bool{}
	for _, start := range starts {
		dir := filepath.Clean(start)
		for depth := 0; depth < 8; depth++ {
			candidates := [][]string{
				append([]string{dir, "assets"}, parts...),
			}
			if strings.EqualFold(filepath.Base(dir), "assets") {
				candidates = append(candidates, append([]string{dir}, parts...))
			}
			for _, c := range candidates {
				p := filepath.Join(c...)
				if seen[p] {
					continue
				}
				seen[p] = true
				if _, err := os.Stat(p); err == nil {
					return p, nil
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return "", errors.New("game asset path not found: " + strings.Join(parts, "/"))
}

func stringAttr(attrs []xml.Attr, name, fallback string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return fallback
}

func intAttr(attrs []xml.Attr, name string, fallback int) int {
	v := stringAttr(attrs, name, "")
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func floatAttr(attrs []xml.Attr, name string, fallback float64) float64 {
	v := stringAttr(attrs, name, "")
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func sortedMapNumbers(m map[int]Map) []int {
	out := make([]int, 0, len(m))
	for n := range m {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}
