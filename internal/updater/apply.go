package updater

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	internalApplyFlag = "--internal-apply-update"
)

// LaunchApply copies the currently running GUI executable to a project-local
// helper, starts that helper, and returns immediately. The caller should then
// terminate the game normally so config/save writes finish before replacement.
func LaunchApply(packagePath string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	root := filepath.Dir(exe)
	updateDir := filepath.Join(root, "data", "update")
	if err := os.MkdirAll(updateDir, 0o755); err != nil {
		return err
	}
	helper := filepath.Join(updateDir, "GunMayhemUpdater.exe")
	if err := copyFile(exe, helper); err != nil {
		return fmt.Errorf("prepare updater helper: %w", err)
	}
	cmd := exec.Command(helper,
		internalApplyFlag,
		"--update-package", packagePath,
		"--update-target", root,
		"--update-pid", strconv.Itoa(os.Getpid()),
	)
	cmd.Dir = root
	if err := startHidden(cmd); err != nil {
		return fmt.Errorf("launch updater helper: %w", err)
	}
	return nil
}

// MaybeRunInternalApply returns true when args describe the private updater
// helper mode. It is intentionally handled before Ebitengine/window startup.
func MaybeRunInternalApply(args []string) (bool, error) {
	if len(args) == 0 || args[0] != internalApplyFlag {
		return false, nil
	}
	var packagePath, target string
	pid := 0
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--update-package":
			if i+1 < len(args) {
				i++
				packagePath = args[i]
			}
		case "--update-target":
			if i+1 < len(args) {
				i++
				target = args[i]
			}
		case "--update-pid":
			if i+1 < len(args) {
				i++
				pid, _ = strconv.Atoi(args[i])
			}
		}
	}
	if packagePath == "" || target == "" || pid <= 0 {
		return true, errors.New("invalid internal updater arguments")
	}
	return true, ApplyPackage(packagePath, target, pid)
}

func ApplyPackage(packagePath, target string, pid int) error {
	packagePath, err := filepath.Abs(packagePath)
	if err != nil {
		return err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return err
	}
	if filepath.Ext(strings.ToLower(packagePath)) != ".zip" {
		return errors.New("update package must be a ZIP archive")
	}
	if err := waitForProcessExit(pid, 90*time.Second); err != nil {
		return fmt.Errorf("wait for game to exit: %w", err)
	}

	updateDir := filepath.Join(target, "data", "update")
	staging := filepath.Join(updateDir, "staging")
	_ = os.RemoveAll(staging)
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	if err := extractZipSafe(packagePath, staging); err != nil {
		return fmt.Errorf("extract update: %w", err)
	}
	payload, err := findPayloadRoot(staging)
	if err != nil {
		return err
	}

	if err := installPayload(payload, target); err != nil {
		return err
	}

	_ = os.Remove(packagePath)
	_ = removePartFiles(updateDir)

	cmd := exec.Command(filepath.Join(target, "GunMayhem.exe"))
	cmd.Dir = target
	if err := startHidden(cmd); err != nil {
		return fmt.Errorf("restart updated game: %w", err)
	}

	if self, err := os.Executable(); err == nil {
		_ = scheduleSelfDelete(self)
	}
	return nil
}

func installPayload(payload, target string) error {
	// Update assets/docs first and the executable last. The persistent data/
	// directory is never copied from the release package, so saves and config
	// survive every update.
	var executableSource string
	err := filepath.WalkDir(payload, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(payload, path)
		if err != nil || rel == "." {
			return err
		}
		first := strings.Split(filepath.ToSlash(rel), "/")[0]
		if strings.EqualFold(first, "data") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(rel, "GunMayhem.exe") {
			executableSource = path
			return nil
		}
		dst := filepath.Join(target, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		return copyFile(path, dst)
	})
	if err != nil {
		return fmt.Errorf("install update files: %w", err)
	}
	if executableSource == "" {
		return errors.New("update package does not contain GunMayhem.exe")
	}
	if err := copyFile(executableSource, filepath.Join(target, "GunMayhem.exe")); err != nil {
		return fmt.Errorf("replace GunMayhem.exe: %w", err)
	}
	return nil
}

func findPayloadRoot(staging string) (string, error) {
	if _, err := os.Stat(filepath.Join(staging, "GunMayhem.exe")); err == nil {
		return staging, nil
	}
	entries, err := os.ReadDir(staging)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(staging, entry.Name())
		if _, err := os.Stat(filepath.Join(candidate, "GunMayhem.exe")); err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("cannot find GunMayhem.exe in update package")
}

func extractZipSafe(zipPath, dst string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	cleanDst := filepath.Clean(dst) + string(os.PathSeparator)
	for _, zf := range r.File {
		name := filepath.Clean(filepath.FromSlash(zf.Name))
		if name == "." || filepath.IsAbs(name) || strings.HasPrefix(name, ".."+string(os.PathSeparator)) || name == ".." {
			return fmt.Errorf("unsafe ZIP path %q", zf.Name)
		}
		outPath := filepath.Join(dst, name)
		if !strings.HasPrefix(filepath.Clean(outPath)+string(os.PathSeparator), cleanDst) {
			return fmt.Errorf("unsafe ZIP path %q", zf.Name)
		}
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(outPath, 0o755); err != nil {
				return err
			}
			continue
		}
		if zf.UncompressedSize64 > uint64(maxUpdatePackageBytes) {
			return errors.New("ZIP entry is too large")
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(outPath)
		if err != nil {
			rc.Close()
			return err
		}
		_, cpErr := io.Copy(out, io.LimitReader(rc, maxUpdatePackageBytes+1))
		closeOutErr := out.Close()
		closeInErr := rc.Close()
		if cpErr != nil {
			return cpErr
		}
		if closeOutErr != nil {
			return closeOutErr
		}
		if closeInErr != nil {
			return closeInErr
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".update-new"
	_ = os.Remove(tmp)
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, cpErr := io.Copy(out, in)
	closeErr := out.Close()
	if cpErr != nil {
		_ = os.Remove(tmp)
		return cpErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if err := os.Rename(tmp, dst); err != nil {
		// Windows Rename cannot replace an existing destination. Remove only
		// after the new file has been fully written to the sibling temporary file.
		if remErr := os.Remove(dst); remErr != nil && !os.IsNotExist(remErr) {
			_ = os.Remove(tmp)
			return remErr
		}
		if err := os.Rename(tmp, dst); err != nil {
			_ = os.Remove(tmp)
			return err
		}
	}
	return nil
}

func removePartFiles(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".part") {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
	return nil
}
