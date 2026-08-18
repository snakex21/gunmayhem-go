package game

import (
	"os"
	"regexp"
	"strconv"
	"sync"
)

type WeaponAnimSource struct {
	Blowback   float64
	Pushback   float64
	IdleRotate float64
}

var (
	weaponAnimMu    sync.Mutex
	weaponAnimCache = map[int]WeaponAnimSource{}
)

func sourceWeaponAnim(number int) WeaponAnimSource {
	weaponAnimMu.Lock()
	defer weaponAnimMu.Unlock()
	if v, ok := weaponAnimCache[number]; ok {
		return v
	}
	def, ok := WeaponByNumber(number)
	if !ok {
		return WeaponAnimSource{}
	}
	path, err := findOriginalPath("scripts", def.SpriteDir, "frame_1", "DoAction.as")
	if err != nil {
		return WeaponAnimSource{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return WeaponAnimSource{}
	}
	s := string(data)
	v := WeaponAnimSource{
		Blowback:   sourceNumericAssignment(s, `_parent\._parent\.blowback\s*=\s*(-?[0-9.]+);`),
		Pushback:   sourceNumericAssignment(s, `_parent\._parent\.pushback\s*=\s*(-?[0-9.]+);`),
		IdleRotate: sourceNumericAssignment(s, `_parent\._parent\.idlerotate\s*=\s*(-?[0-9.]+);`),
	}
	weaponAnimCache[number] = v
	return v
}

func sourceNumericAssignment(source, pattern string) float64 {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(source)
	if len(m) != 2 {
		return 0
	}
	v, _ := strconv.ParseFloat(m[1], 64)
	return v
}
