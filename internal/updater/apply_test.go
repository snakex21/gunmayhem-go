package updater

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func updaterTestDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "build", "test-data", "updater-"+strconv.Itoa(os.Getpid())+"-"+t.Name())
	_ = os.RemoveAll(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func TestInstallPayloadPreservesPersistentData(t *testing.T) {
	root := updaterTestDir(t)
	payload := filepath.Join(root, "payload")
	target := filepath.Join(root, "target")
	for _, dir := range []string{
		filepath.Join(payload, "assets"), filepath.Join(payload, "data"),
		filepath.Join(target, "assets"), filepath.Join(target, "data"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(payload, "GunMayhem.exe"), []byte("new-exe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(payload, "assets", "asset.txt"), []byte("new-asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(payload, "data", "config.json"), []byte("packaged-config"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "GunMayhem.exe"), []byte("old-exe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "assets", "asset.txt"), []byte("old-asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "data", "config.json"), []byte("my-save-config"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := installPayload(payload, target); err != nil {
		t.Fatal(err)
	}
	gotExe, _ := os.ReadFile(filepath.Join(target, "GunMayhem.exe"))
	gotAsset, _ := os.ReadFile(filepath.Join(target, "assets", "asset.txt"))
	gotConfig, _ := os.ReadFile(filepath.Join(target, "data", "config.json"))
	if string(gotExe) != "new-exe" || string(gotAsset) != "new-asset" {
		t.Fatalf("payload did not update executable/assets: exe=%q asset=%q", gotExe, gotAsset)
	}
	if string(gotConfig) != "my-save-config" {
		t.Fatalf("update overwrote persistent data: %q", gotConfig)
	}
}
