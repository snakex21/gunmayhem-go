package game

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

func TestWeaponCatalogMatchesOriginalActionScript(t *testing.T) {
	for number, def := range weaponCatalog {
		def := def
		t.Run(strconv.Itoa(number)+"_"+def.Name, func(t *testing.T) {
			path := filepath.Join("..", "..", "source", "scripts", def.SpriteDir, "frame_1", "DoAction.as")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			s := string(data)

			checkStringAssign(t, s, `(?m)^Name\s*=\s*"([^"]*)";`, "Name", def.Name)
			checkNumAssign(t, s, `(?m)^_parent\._parent\.rof\s*=\s*(-?[0-9.]+);`, "rof", float64(def.ROF), true)
			checkNumAssign(t, s, `(?m)^_parent\._parent\.firepower\s*=\s*(-?[0-9.]+);`, "firepower", def.Firepower, true)
			checkNumAssign(t, s, `(?m)^_parent\._parent\.recoil\s*=\s*(-?[0-9.]+);`, "recoil", def.Recoil, true)
			checkNumAssign(t, s, `(?m)^_parent\._parent\.bullets\s*=\s*(-?[0-9.]+);`, "bullets", float64(def.Bullets), true)
			checkNumAssign(t, s, `(?m)^shotgun\s*=\s*(-?[0-9.]+);`, "shotgun", float64(def.Shotgun), def.Shotgun != 0)
			checkNumAssign(t, s, `(?m)^_parent\._parent\.weight\s*=\s*(-?[0-9.]+);`, "weight", def.Weight, false)
			checkNumAssign(t, s, `(?m)^shellX\s*=\s*(-?[0-9.]+);`, "shellX", def.ShellX, true)
			checkNumAssign(t, s, `(?m)^flashX\s*=\s*(-?[0-9.]+);`, "flashX", def.FlashX, true)

			pose := poseForWeapon(number)
			checkNumAssign(t, s, `(?m)^shootx\s*=\s*(-?[0-9.]+);`, "shootx", pose.ShootX, true)
			checkNumAssign(t, s, `(?m)^shooty\s*=\s*(-?[0-9.]+);`, "shooty", pose.ShootY, true)
			checkNumAssign(t, s, `(?m)^handx\s*=\s*(-?[0-9.]+);`, "handx", pose.HandX, true)
			checkNumAssign(t, s, `(?m)^handy\s*=\s*(-?[0-9.]+);`, "handy", pose.HandY, true)
		})
	}
}

func checkNumAssign(t *testing.T, source, pattern, name string, want float64, required bool) {
	t.Helper()
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(source)
	if len(m) != 2 {
		if required {
			t.Fatalf("source assignment %s missing", name)
		}
		// If the source does not assign a property, Flash leaves the previous
		// value in place. Catalog zero is our sentinel for that source behavior.
		if want != 0 {
			t.Fatalf("source does not assign %s but catalog has %v", name, want)
		}
		return
	}
	got, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("%s source=%v catalog=%v", name, got, want)
	}
}

func checkStringAssign(t *testing.T, source, pattern, name, want string) {
	t.Helper()
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(source)
	if len(m) != 2 {
		t.Fatalf("source assignment %s missing", name)
	}
	if m[1] != want {
		t.Fatalf("%s source=%q catalog=%q", name, m[1], want)
	}
}
