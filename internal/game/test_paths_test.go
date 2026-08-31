package game

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync/atomic"
	"testing"
)

var projectTestDirCounter atomic.Uint64
var unsafeTestPathChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		os.Exit(2)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	shared := filepath.Join(root, "build", "test-data", "package-shared")
	_ = os.RemoveAll(shared)
	if err := os.MkdirAll(shared, 0o755); err != nil {
		os.Exit(2)
	}
	_ = os.Setenv("GUNMAYHEM_DATA_DIR", shared)
	code := m.Run()
	_ = os.RemoveAll(shared)
	os.Exit(code)
}

// projectTestDir keeps test scratch data inside the repository instead of the
// operating-system TEMP directory. Every directory is removed automatically at
// the end of its test.
func projectTestDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	name := unsafeTestPathChars.ReplaceAllString(t.Name(), "_")
	id := projectTestDirCounter.Add(1)
	dir := filepath.Join(root, "build", "test-data", name+"-"+strconv.FormatUint(id, 10))
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func useProjectTestDataDir(t *testing.T) string {
	t.Helper()
	dir := projectTestDir(t)
	t.Setenv("GUNMAYHEM_DATA_DIR", dir)
	return dir
}
