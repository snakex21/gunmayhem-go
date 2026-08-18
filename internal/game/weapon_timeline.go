package game

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type WeaponTimelineDef struct {
	TotalFrames int
	RestFrame   int
	Scripts     map[int]string
}

var (
	weaponTimelineMu    sync.Mutex
	weaponTimelineCache = map[int]WeaponTimelineDef{}
	bulletsAssignRE     = regexp.MustCompile(`_parent\._parent\.bullets\s*=\s*(\d+)\s*;`)
	gotoAndPlayRE       = regexp.MustCompile(`gotoAndPlay\((\d+)\)\s*;`)
)

func weaponTimeline(number int) WeaponTimelineDef {
	weaponTimelineMu.Lock()
	defer weaponTimelineMu.Unlock()
	if def, ok := weaponTimelineCache[number]; ok {
		return def
	}
	w, ok := WeaponByNumber(number)
	if !ok {
		return WeaponTimelineDef{TotalFrames: 2, RestFrame: 1, Scripts: map[int]string{}}
	}
	library := sourceLibraryNameFromSpriteDir(w.SpriteDir)
	def, err := loadWeaponTimeline(library)
	if err != nil {
		def = WeaponTimelineDef{TotalFrames: 2, RestFrame: 1, Scripts: map[int]string{}}
	}
	weaponTimelineCache[number] = def
	return def
}

func loadWeaponTimeline(libraryName string) (WeaponTimelineDef, error) {
	libraryDir, err := findOriginalPath("fla", "LIBRARY")
	if err != nil {
		return WeaponTimelineDef{}, err
	}
	f, err := os.Open(filepath.Join(libraryDir, libraryName+".xml"))
	if err != nil {
		return WeaponTimelineDef{}, err
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	currentFrame := -1
	maxFrame := 0
	scripts := map[int]string{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return WeaponTimelineDef{}, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "DOMFrame":
				currentFrame = intAttr(t.Attr, "index", 0)
				duration := intAttr(t.Attr, "duration", 1)
				if duration < 1 {
					duration = 1
				}
				if currentFrame+duration > maxFrame {
					maxFrame = currentFrame + duration
				}
			case "script":
				if currentFrame >= 0 {
					var script string
					if err := dec.DecodeElement(&script, &t); err != nil {
						return WeaponTimelineDef{}, err
					}
					if strings.TrimSpace(script) != "" {
						scripts[currentFrame] += script + "\n"
					}
				}
			}
		case xml.EndElement:
			if t.Name.Local == "DOMFrame" {
				currentFrame = -1
			}
		}
	}
	if maxFrame < 2 {
		maxFrame = 2
	}

	rest := 1 // all normal guns idle on ActionScript frame 2
	keys := make([]int, 0, len(scripts))
	for frame := range scripts {
		keys = append(keys, frame)
	}
	sort.Ints(keys)
	for _, frame := range keys {
		if frame > 0 && strings.Contains(scripts[frame], "stop();") {
			rest = frame
			break
		}
	}
	return WeaponTimelineDef{TotalFrames: maxFrame, RestFrame: rest, Scripts: scripts}, nil
}

func sourceWeaponMagazineFromScript(script string) (int, bool) {
	matches := bulletsAssignRE.FindAllStringSubmatch(script, -1)
	if len(matches) == 0 {
		return 0, false
	}
	// The final assignment in a frame is the value ActionScript leaves behind.
	v, err := strconv.Atoi(matches[len(matches)-1][1])
	if err != nil {
		return 0, false
	}
	return v, true
}

func sourceWeaponGoto(script string) (int, bool) {
	m := gotoAndPlayRE.FindStringSubmatch(script)
	if len(m) != 2 {
		return 0, false
	}
	v, err := strconv.Atoi(m[1])
	if err != nil || v <= 0 {
		return 0, false
	}
	return v - 1, true
}
