package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gunmayhem-go/internal/masterapi"
)

func portableTestPaths(t *testing.T) (root, outRoot, zipPath string) {
	t.Helper()
	root = filepath.Join(t.TempDir(), "Game & Friends (100%)! ^")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	outRoot = filepath.Join(root, "build", "GunMayhem-Windows-x64")
	zipPath = filepath.Join(root, "build", "GunMayhem-Windows-x64.zip")
	return
}

func writePortableTestFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func requirePortableFileMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("unexpected package output at %s (stat error: %v)", filepath.Base(path), err)
	}
}

func TestPortableMissingDirectoryConfigCreatesNoDefaultsOrLauncher(t *testing.T) {
	root, outRoot, zipPath := portableTestPaths(t)
	// Packaging takes only explicit file input, never a developer's environment.
	t.Setenv("GUNMAYHEM_MASTER_URL", "https://must-not-be-packaged.example")
	writePortableTestFile(t, filepath.Join(outRoot, portableDirectoryConfigFile), "https://old.example")
	writePortableTestFile(t, filepath.Join(outRoot, portableInternetLauncher), "old launcher")
	writePortableTestFile(t, zipPath, "old archive")
	configured, err := preparePortableOutput(root, outRoot, zipPath)
	if err != nil || configured {
		t.Fatalf("missing config: configured=%v err=%v", configured, err)
	}
	requirePortableFileMissing(t, filepath.Join(root, portableDirectoryConfigFile))
	requirePortableFileMissing(t, filepath.Join(outRoot, portableDirectoryConfigFile))
	requirePortableFileMissing(t, filepath.Join(outRoot, portableInternetLauncher))
	requirePortableFileMissing(t, zipPath)
	entries, err := os.ReadDir(outRoot)
	if err != nil || len(entries) != 1 || entries[0].Name() != "data" || !entries[0].IsDir() {
		t.Fatalf("unconfigured output should contain only the data directory: %v, %v", entries, err)
	}
}

func TestPortableValidDirectoryConfig(t *testing.T) {
	prefix := "https://catalog.example/"
	maxURL := prefix + strings.Repeat("a", masterapi.MaxPublicDirectoryURLBytes-len(prefix))
	for _, tc := range []struct{ name, input, want string }{
		{"no newline", prefix, "https://catalog.example\r\n"},
		{"LF", prefix + "base/\n", prefix + "base\r\n"},
		{"CRLF", prefix + "base/\r\n", prefix + "base\r\n"},
		{"maximum URL and CRLF", maxURL + "\r\n", maxURL + "\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, outRoot, zipPath := portableTestPaths(t)
			source := filepath.Join(root, portableDirectoryConfigFile)
			writePortableTestFile(t, source, tc.input)
			writePortableTestFile(t, filepath.Join(root, "keep.txt"), "keep")
			configured, err := preparePortableOutput(root, outRoot, zipPath)
			if err != nil || !configured {
				t.Fatalf("valid config: configured=%v err=%v", configured, err)
			}
			data, err := os.ReadFile(filepath.Join(outRoot, portableDirectoryConfigFile))
			if err != nil || string(data) != tc.want {
				t.Fatal("validated directory config was not written canonically")
			}
			data, err = os.ReadFile(source)
			if err != nil || string(data) != tc.input {
				t.Fatal("source config was changed")
			}
			data, err = os.ReadFile(filepath.Join(root, "keep.txt"))
			if err != nil || string(data) != "keep" {
				t.Fatal("unrelated root file was changed")
			}
			launcher, err := os.ReadFile(filepath.Join(outRoot, portableInternetLauncher))
			if err != nil || string(launcher) != experimentalInternetLauncher {
				t.Fatal("explicit experimental launcher missing or changed")
			}
			requirePortableFileMissing(t, zipPath)
		})
	}
}

func TestPortableInvalidDirectoryConfigFailsClosedWithoutSecrets(t *testing.T) {
	const secret = "do-not-log-this-secret"
	for _, tc := range []struct{ name, input string }{
		{"empty", ""},
		{"HTTP", "http://catalog.example/" + secret},
		{"missing hostname", "https://:443/" + secret},
		{"credentials", "https://user:" + secret + "@catalog.example"},
		{"query", "https://catalog.example?token=" + secret},
		{"fragment", "https://catalog.example#" + secret},
		{"escaped path", "https://catalog.example/%2f" + secret},
		{"malformed URL", "https://catalog.example:%" + secret},
		{"multiple LF lines", "https://catalog.example\nhttps://" + secret + ".example\n"},
		{"multiple CRLF lines", "https://catalog.example\r\nhttps://" + secret + ".example\r\n"},
		{"extra blank line", "https://catalog.example\n\n"},
		{"leading space", " https://catalog.example/" + secret},
		{"trailing space", "https://catalog.example/" + secret + " "},
		{"lone CR", "https://catalog.example/" + secret + "\r"},
		{"oversized file", "https://catalog.example/" + secret + strings.Repeat("x", maxDirectoryConfigBytes)},
		{"oversized URL", "https://catalog.example/" + strings.Repeat("x", masterapi.MaxPublicDirectoryURLBytes-len("https://catalog.example/")+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, outRoot, zipPath := portableTestPaths(t)
			writePortableTestFile(t, filepath.Join(root, portableDirectoryConfigFile), tc.input)
			writePortableTestFile(t, filepath.Join(outRoot, portableDirectoryConfigFile), "https://old.example")
			writePortableTestFile(t, filepath.Join(outRoot, portableInternetLauncher), "old launcher")
			writePortableTestFile(t, zipPath, "old archive")
			configured, err := preparePortableOutput(root, outRoot, zipPath)
			if err == nil || configured {
				t.Fatal("invalid configuration did not fail closed")
			}
			if !strings.Contains(err.Error(), portableDirectoryConfigFile) || strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "https://") || strings.Contains(err.Error(), "http://") {
				t.Fatal("configuration error is unclear or contains input data")
			}
			requirePortableFileMissing(t, outRoot)
			requirePortableFileMissing(t, zipPath)
		})
	}
}

func TestPortableDirectoryConfigRejectsNonregularFiles(t *testing.T) {
	for _, kind := range []string{"directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			root, outRoot, zipPath := portableTestPaths(t)
			path := filepath.Join(root, portableDirectoryConfigFile)
			if kind == "directory" {
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
			} else {
				target := filepath.Join(root, "private.txt")
				writePortableTestFile(t, target, "https://user:secret@catalog.example")
				if err := os.Symlink(target, path); err != nil {
					t.Skipf("symlink creation unavailable: %v", err)
				}
			}
			if configured, err := preparePortableOutput(root, outRoot, zipPath); err == nil || configured {
				t.Fatal("nonregular configuration accepted")
			}
			requirePortableFileMissing(t, outRoot)
			requirePortableFileMissing(t, zipPath)
		})
	}
}

// Structural assertions only: these tests do not run cmd.exe or certify Windows
// behavior. The path fixture above intentionally includes cmd metacharacters.
func TestPortableInternetLauncherIsQuotedScopedAndReturnsExitCode(t *testing.T) {
	lines := strings.Split(strings.TrimSuffix(experimentalInternetLauncher, "\r\n"), "\r\n")
	want := []string{
		"@echo off",
		"setlocal EnableExtensions DisableDelayedExpansion",
		`pushd "%~dp0"`,
		"if errorlevel 1 exit /b 1",
		`set "GUNMAYHEM_EXPERIMENTAL_INTERNET=1"`,
		`set "ERRORLEVEL="`,
		`start "" /wait "%~dp0GunMayhem.exe"`,
		`set "game_exit_code=%errorlevel%"`,
		"popd",
		"endlocal & exit /b %game_exit_code%",
	}
	if fmt.Sprint(lines) != fmt.Sprint(want) {
		t.Fatal("launcher no longer follows the reviewed, quoted, launch-scoped command structure")
	}
	for _, unsafe := range []string{"https://", "http://", "GUNMAYHEM_MASTER_URL", "%*", "call ", "powershell", "runas", "netsh", "setx "} {
		if strings.Contains(strings.ToLower(experimentalInternetLauncher), strings.ToLower(unsafe)) {
			t.Fatal("launcher includes URL interpolation, argument forwarding, or out-of-scope commands")
		}
	}
}

func TestPortableInstructionsMatchConfiguration(t *testing.T) {
	lan := portableInstructions(false)
	if !strings.Contains(lan, "DZIALA TYLKO W LAN") || !strings.Contains(lan, "multiplayer LAN only") {
		t.Fatal("missing configuration lost LAN-only guidance")
	}
	if strings.Contains(lan, portableDirectoryConfigFile) || strings.Contains(lan, portableInternetLauncher) {
		t.Fatal("unconfigured instructions advertise nonexistent files")
	}
	experimental := portableInstructions(true)
	for _, required := range []string{portableDirectoryConfigFile, portableInternetLauncher, "EKSPERYMENTALNY INTERNET", "LAN pozostaje domyslny", "Test uruchomienia Windows", "miedzy roznymi sieciami", "nadal oczekuja", "kazdym komputerze", "CUSTOM GAME -> ONLINE -> REFRESH -> wybierz pokoj -> JOIN SESSION", "CREATE SESSION -> INTERNET -> OPEN", "cloudflared.exe obok GunMayhem.exe lub w PATH"} {
		if !strings.Contains(experimental, required) {
			t.Errorf("configured package instructions missing %q", required)
		}
	}
	if strings.Contains(experimental, "DZIALA TYLKO W LAN") || strings.Contains(experimental, "multiplayer LAN only") {
		t.Fatal("configured package retains contradictory LAN-only description")
	}
	for _, text := range []string{lan, experimental} {
		if strings.Contains(text, "https://") || strings.Contains(strings.ReplaceAll(text, "\r\n", ""), "\n") {
			t.Fatal("instructions include a default endpoint or non-Windows line endings")
		}
	}
}

func TestPortableTinyArchiveIncludesOnlyConfiguredInternetFiles(t *testing.T) {
	for _, configured := range []bool{false, true} {
		t.Run(fmt.Sprint(configured), func(t *testing.T) {
			root, outRoot, zipPath := portableTestPaths(t)
			if configured {
				writePortableTestFile(t, filepath.Join(root, portableDirectoryConfigFile), "https://catalog.example/base/\n")
			}
			got, err := preparePortableOutput(root, outRoot, zipPath)
			if err != nil || got != configured {
				t.Fatalf("prepare: configured=%v err=%v", got, err)
			}
			writePortableTestFile(t, filepath.Join(outRoot, "MULTIPLAYER_TEST.txt"), portableInstructions(got))
			// Only tiny synthetic config/instruction files; no executable or assets.
			if err := zipDir(outRoot, zipPath); err != nil {
				t.Fatal(err)
			}
			archive, err := zip.OpenReader(zipPath)
			if err != nil {
				t.Fatal(err)
			}
			defer archive.Close()
			files := make(map[string]string)
			for _, entry := range archive.File {
				r, err := entry.Open()
				if err != nil {
					t.Fatal(err)
				}
				data, readErr := io.ReadAll(r)
				closeErr := r.Close()
				if readErr != nil || closeErr != nil {
					t.Fatalf("read archive fixture: %v, %v", readErr, closeErr)
				}
				files[strings.TrimPrefix(entry.Name, filepath.Base(outRoot)+"/")] = string(data)
			}
			if _, exists := files[portableDirectoryConfigFile]; exists != configured {
				t.Fatal("archive configuration presence differs from input")
			}
			if _, exists := files[portableInternetLauncher]; exists != configured {
				t.Fatal("archive launcher presence differs from input")
			}
			if configured && (files[portableDirectoryConfigFile] != "https://catalog.example/base\r\n" || files[portableInternetLauncher] != experimentalInternetLauncher) {
				t.Fatal("archive changed the validated configuration or fixed launcher")
			}
			if files["MULTIPLAYER_TEST.txt"] != portableInstructions(configured) || len(files) != 1+2*boolToInt(configured) {
				t.Fatal("archive contains missing instructions or unexpected files")
			}
		})
	}
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestPortableMainRejectsInvalidConfigBeforeAssetCopy(t *testing.T) {
	root, outRoot, zipPath := portableTestPaths(t)
	writePortableTestFile(t, filepath.Join(root, "go.mod"), "module fixture\n")
	writePortableTestFile(t, filepath.Join(root, portableDirectoryConfigFile), "https://user:do-not-log-this-secret@catalog.example")
	writePortableTestFile(t, filepath.Join(outRoot, portableInternetLauncher), "old launcher")
	writePortableTestFile(t, zipPath, "old archive")
	t.Chdir(root)
	var failure any
	func() {
		defer func() { failure = recover() }()
		main()
	}()
	if failure == nil || !strings.Contains(fmt.Sprint(failure), portableDirectoryConfigFile) || strings.Contains(fmt.Sprint(failure), "do-not-log-this-secret") {
		t.Fatal("main did not fail clearly and safely before touching assets")
	}
	requirePortableFileMissing(t, outRoot)
	requirePortableFileMissing(t, zipPath)
}

func TestPortableMainCleansUpAfterRuntimeCopyFailure(t *testing.T) {
	for _, missing := range []string{"GunMayhem.exe", "assets"} {
		t.Run(missing, func(t *testing.T) {
			root, outRoot, zipPath := portableTestPaths(t)
			const config = "https://catalog.example/base\n"
			writePortableTestFile(t, filepath.Join(root, "go.mod"), "module fixture\n")
			writePortableTestFile(t, filepath.Join(root, portableDirectoryConfigFile), config)
			writePortableTestFile(t, filepath.Join(root, "keep.txt"), "unrelated source")
			if missing == "assets" {
				// Tiny inert fixture, not an executable build or a real asset copy.
				writePortableTestFile(t, filepath.Join(root, "GunMayhem.exe"), "fixture")
			}
			t.Chdir(root)
			var failure any
			func() {
				defer func() { failure = recover() }()
				main()
			}()
			if failure == nil || !strings.Contains(fmt.Sprint(failure), missing) {
				t.Fatal("main did not report the expected late copy failure")
			}
			requirePortableFileMissing(t, outRoot)
			requirePortableFileMissing(t, zipPath)
			for name, want := range map[string]string{portableDirectoryConfigFile: config, "keep.txt": "unrelated source"} {
				data, err := os.ReadFile(filepath.Join(root, name))
				if err != nil || string(data) != want {
					t.Fatal("late failure cleanup changed source files")
				}
			}
		})
	}
}

func TestPortablePipelineCleansPartialZIP(t *testing.T) {
	root, outRoot, zipPath := portableTestPaths(t)
	writePortableTestFile(t, filepath.Join(root, portableDirectoryConfigFile), "https://catalog.example")
	err := withPortableOutputCleanup(outRoot, zipPath, func() error {
		if _, err := preparePortableOutput(root, outRoot, zipPath); err != nil {
			return err
		}
		// zipDir creates its destination before walking the source. A missing
		// synthetic source forces a late failure without real assets or a build.
		err := zipDir(filepath.Join(root, "missing-assets"), zipPath)
		if err == nil {
			t.Fatal("ZIP fixture did not fail")
		}
		if info, statErr := os.Stat(zipPath); statErr != nil || info.Size() == 0 {
			t.Fatal("ZIP fixture did not leave a partial archive before cleanup")
		}
		return err
	})
	if err == nil {
		t.Fatal("pipeline swallowed ZIP failure")
	}
	requirePortableFileMissing(t, outRoot)
	requirePortableFileMissing(t, zipPath)
	if _, err := os.Stat(filepath.Join(root, portableDirectoryConfigFile)); err != nil {
		t.Fatal("ZIP cleanup removed source configuration")
	}
}

func TestPortablePipelineRetainsSuccessfulTinyOutput(t *testing.T) {
	root, outRoot, zipPath := portableTestPaths(t)
	writePortableTestFile(t, filepath.Join(root, portableDirectoryConfigFile), "https://catalog.example")
	err := withPortableOutputCleanup(outRoot, zipPath, func() error {
		if _, err := preparePortableOutput(root, outRoot, zipPath); err != nil {
			return err
		}
		return zipDir(outRoot, zipPath)
	})
	if err != nil {
		t.Fatalf("tiny successful pipeline: %v", err)
	}
	for _, path := range []string{filepath.Join(outRoot, portableDirectoryConfigFile), filepath.Join(outRoot, portableInternetLauncher), zipPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal("successful pipeline removed a completed output")
		}
	}
}

func TestPortablePipelineReportsCleanupFailure(t *testing.T) {
	root, outRoot, zipPath := portableTestPaths(t)
	writePortableTestFile(t, filepath.Join(root, portableDirectoryConfigFile), "https://catalog.example")
	failure := errors.New("synthetic packaging failure")
	err := withPortableOutputCleanup(outRoot, zipPath, func() error {
		if _, err := preparePortableOutput(root, outRoot, zipPath); err != nil {
			return err
		}
		// Do not recursively delete unexpected directory contents at a ZIP path.
		writePortableTestFile(t, filepath.Join(zipPath, "keep.txt"), "unexpected directory")
		return failure
	})
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), "remove portable ZIP") {
		t.Fatal("original failure or cleanup failure was not reported")
	}
	requirePortableFileMissing(t, outRoot)
	if _, err := os.Stat(filepath.Join(zipPath, "keep.txt")); err != nil {
		t.Fatal("cleanup recursively deleted unexpected ZIP-path directory contents")
	}
}
