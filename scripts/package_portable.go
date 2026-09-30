package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gunmayhem-go/internal/masterapi"
)

const (
	portableDirectoryConfigFile = "master_server.txt"
	portableInternetLauncher    = "Start-Internet.bat"
	// A single maximum-length URL may have one Windows line ending.
	maxDirectoryConfigBytes = masterapi.MaxPublicDirectoryURLBytes + 2
)

// All paths are resolved by cmd relative to this script, never interpolated
// from configuration. Disabling delayed expansion preserves literal ! in paths;
// quoting and avoiding CALL preserve spaces and cmd metacharacters. START /WAIT
// waits for the GUI process so the exit code can be returned after POPD.
const experimentalInternetLauncher = "@echo off\r\n" +
	"setlocal EnableExtensions DisableDelayedExpansion\r\n" +
	"pushd \"%~dp0\"\r\n" +
	"if errorlevel 1 exit /b 1\r\n" +
	"set \"GUNMAYHEM_EXPERIMENTAL_INTERNET=1\"\r\n" +
	"set \"ERRORLEVEL=\"\r\n" +
	"start \"\" /wait \"%~dp0GunMayhem.exe\"\r\n" +
	"set \"game_exit_code=%errorlevel%\"\r\n" +
	"popd\r\n" +
	"endlocal & exit /b %game_exit_code%\r\n"

func readOptionalDirectoryConfig(root string) (string, error) {
	path := filepath.Join(root, portableDirectoryConfigFile)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", errors.New("cannot inspect optional master_server.txt")
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("master_server.txt must be a regular file")
	}
	if info.Size() > maxDirectoryConfigBytes {
		return "", errors.New("master_server.txt exceeds the 2050-byte limit")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", errors.New("cannot read optional master_server.txt")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxDirectoryConfigBytes+1))
	if err != nil {
		return "", errors.New("cannot read optional master_server.txt")
	}
	if len(data) > maxDirectoryConfigBytes {
		return "", errors.New("master_server.txt exceeds the 2050-byte limit")
	}
	raw := string(data)
	if strings.HasSuffix(raw, "\r\n") {
		raw = strings.TrimSuffix(raw, "\r\n")
	} else {
		raw = strings.TrimSuffix(raw, "\n")
	}
	baseURL := masterapi.NormalizePublicDirectoryURL(raw)
	if baseURL == "" {
		// Do not wrap url.Parse errors or include the file contents: rejected
		// configurations may contain credentials, tokens or private invites.
		return "", errors.New("master_server.txt must contain one HTTPS directory URL (at most 2048 bytes), without credentials, query, fragment, escaped path or extra whitespace")
	}
	return baseURL, nil
}

// Prepare before copying assets. Removing stale outputs first prevents a failed
// validation from leaving an older configured package or ZIP ready to distribute.
func preparePortableOutput(root, outRoot, zipPath string) (configured bool, err error) {
	if err := removePortableOutputs(outRoot, zipPath); err != nil {
		return false, err
	}
	baseURL, err := readOptionalDirectoryConfig(root)
	if err != nil {
		return false, err
	}
	// Avoid a partial launcher/config pair if creating either file fails.
	defer func() {
		if err != nil {
			err = errors.Join(err, removePortableOutputs(outRoot, zipPath))
		}
	}()
	if err := os.MkdirAll(filepath.Join(outRoot, "data"), 0o755); err != nil {
		return false, err
	}
	if baseURL == "" {
		return false, nil
	}
	if err := os.WriteFile(filepath.Join(outRoot, portableDirectoryConfigFile), []byte(baseURL+"\r\n"), 0o644); err != nil {
		return false, err
	}
	if err := os.WriteFile(filepath.Join(outRoot, portableInternetLauncher), []byte(experimentalInternetLauncher), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// Only generated outputs are removed. A path occupying the ZIP location is
// never recursively deleted, and both removals are attempted even if one fails.
func removePortableOutputs(outRoot, zipPath string) error {
	var cleanupErrors []error
	if err := os.RemoveAll(outRoot); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("remove portable folder: %w", err))
	}
	if err := os.Remove(zipPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("remove portable ZIP: %w", err))
	}
	return errors.Join(cleanupErrors...)
}

// Cover the whole pipeline, including asset copying and ZIP creation. Returning
// cleanup failures with the original error avoids claiming incomplete output
// was removed when filesystem permissions or another process prevented it.
func withPortableOutputCleanup(outRoot, zipPath string, build func() error) error {
	if err := build(); err != nil {
		return errors.Join(err, removePortableOutputs(outRoot, zipPath))
	}
	return nil
}

var xflLibraryReferenceRE = regexp.MustCompile(`libraryItemName="([^"]+)"`)

var gm2RuntimeSpriteDirs = []string{
	"DefineSprite_148", "DefineSprite_174", "DefineSprite_201", "DefineSprite_223",
	"DefineSprite_279", "DefineSprite_291", "DefineSprite_368", "DefineSprite_760",
	"DefineSprite_1642", "DefineSprite_1690", "DefineSprite_1835",
	"DefineSprite_573_gun_A1", "DefineSprite_575_gun_A2", "DefineSprite_577_gun_A3",
	"DefineSprite_579_gun_A4", "DefineSprite_581_gun_A5", "DefineSprite_583_gun_B1",
	"DefineSprite_585_gun_B2", "DefineSprite_587_gun_B3", "DefineSprite_589_gun_B4",
	"DefineSprite_591_gun_B5", "DefineSprite_593_gun_C1", "DefineSprite_595_gun_C2",
	"DefineSprite_597_gun_C3", "DefineSprite_599_gun_C4", "DefineSprite_601_gun_C5",
	"DefineSprite_603_gun_D1", "DefineSprite_605_gun_D2", "DefineSprite_609_gun_D3",
	"DefineSprite_611_gun_D4", "DefineSprite_613_gun_D5",
}

var gm2RuntimeLibraryRoots = []string{
	"Symbol 148", "Symbol 174", "Symbol 201", "Symbol 223", "Symbol 279",
	"Symbol 291", "Symbol 368", "Symbol 760", "Symbol 1642", "Symbol 1690",
	"Symbol 1835", "Symbol 1940", "Symbol 573", "Symbol 575", "Symbol 577",
	"Symbol 579", "Symbol 581", "Symbol 583", "Symbol 585", "Symbol 587",
	"Symbol 589", "Symbol 591", "Symbol 593", "Symbol 595", "Symbol 597",
	"Symbol 599", "Symbol 601", "Symbol 603", "Symbol 605", "Symbol 609",
	"Symbol 611", "Symbol 613",
}

var gm2RuntimePreviewFiles = []string{
	"1170.jpg", "1174.jpg", "1177.jpg", "1180.jpg", "1183.jpg",
	"1186.jpg", "1189.jpg", "1192.jpg", "1195.jpg",
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
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func safeLibraryXMLPath(libraryDir, name string) (string, bool) {
	name = filepath.FromSlash(strings.TrimSpace(html.UnescapeString(name)))
	if name == "" || filepath.IsAbs(name) {
		return "", false
	}
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", false
	}
	path := filepath.Join(libraryDir, clean+".xml")
	rel, err := filepath.Rel(libraryDir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return path, true
}

func copyXFLLibraryClosure(srcLibrary, dstLibrary string, roots []string) (int, error) {
	queue := append([]string(nil), roots...)
	seen := make(map[string]bool)
	copied := 0
	for len(queue) > 0 {
		name := strings.TrimSpace(queue[0])
		queue = queue[1:]
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		src, ok := safeLibraryXMLPath(srcLibrary, name)
		if !ok {
			return copied, fmt.Errorf("unsafe GM2 XFL library name %q", name)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return copied, fmt.Errorf("read required GM2 XFL item %q: %w", name, err)
		}
		rel, err := filepath.Rel(srcLibrary, src)
		if err != nil {
			return copied, err
		}
		if err := copyFile(src, filepath.Join(dstLibrary, rel)); err != nil {
			return copied, err
		}
		copied++
		for _, match := range xflLibraryReferenceRE.FindAllSubmatch(data, -1) {
			if len(match) == 2 {
				ref := strings.TrimSpace(html.UnescapeString(string(match[1])))
				if refPath, valid := safeLibraryXMLPath(srcLibrary, ref); valid {
					if _, statErr := os.Stat(refPath); statErr == nil && !seen[ref] {
						queue = append(queue, ref)
					}
				}
			}
		}
	}
	return copied, nil
}

func copyGM2RuntimeAssets(gm2Root, dst string) error {
	for _, name := range gm2RuntimeSpriteDirs {
		src := filepath.Join(gm2Root, "sprites", name)
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("required GM2 sprite directory missing %q: %w", name, err)
		}
		if err := copyDir(src, filepath.Join(dst, "sprites", name)); err != nil {
			return err
		}
	}
	for _, name := range gm2RuntimePreviewFiles {
		if err := copyFile(filepath.Join(gm2Root, "images", name), filepath.Join(dst, "images", name)); err != nil {
			return fmt.Errorf("copy required GM2 preview %q: %w", name, err)
		}
	}
	count, err := copyXFLLibraryClosure(
		filepath.Join(gm2Root, "fla", "LIBRARY"),
		filepath.Join(dst, "fla", "LIBRARY"),
		gm2RuntimeLibraryRoots,
	)
	if err != nil {
		return err
	}
	if count < len(gm2RuntimeLibraryRoots) {
		return fmt.Errorf("GM2 XFL closure copied only %d files for %d roots", count, len(gm2RuntimeLibraryRoots))
	}
	return nil
}

func zipDir(srcDir, zipPath string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	walkErr := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(filepath.Dir(srcDir), path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = rel
		h.Method = zip.Deflate
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		r, err := os.Open(path)
		if err != nil {
			return err
		}
		_, cpErr := io.Copy(w, r)
		clErr := r.Close()
		if cpErr != nil {
			return cpErr
		}
		return clErr
	})
	closeZipErr := zw.Close()
	closeFileErr := f.Close()
	if walkErr != nil {
		return walkErr
	}
	if closeZipErr != nil {
		return closeZipErr
	}
	return closeFileErr
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		panic("run from gunmayhem-go root")
	}
	if err := buildPortablePackage(root); err != nil {
		panic(err)
	}
}

func buildPortablePackage(root string) error {
	outRoot := filepath.Join(root, "build", "GunMayhem-Windows-x64")
	zipPath := filepath.Join(root, "build", "GunMayhem-Windows-x64.zip")
	return withPortableOutputCleanup(outRoot, zipPath, func() error {
		directoryConfigured, err := preparePortableOutput(root, outRoot, zipPath)
		if err != nil {
			return err
		}

		if err := copyFile(filepath.Join(root, "GunMayhem.exe"), filepath.Join(outRoot, "GunMayhem.exe")); err != nil {
			return err
		}
		if err := copyDir(filepath.Join(root, "assets"), filepath.Join(outRoot, "assets")); err != nil {
			return err
		}
		// Keep the unpacked GM2/FFDec source beside the repository for development,
		// but never ship that source tree. Curate only the exact runtime sprites,
		// previews and transitive XFL metadata needed by maps, weapons and cosmetics.
		gm2Root := filepath.Join(filepath.Dir(root), "gun mayhem 2")
		if _, err := os.Stat(gm2Root); err != nil {
			return fmt.Errorf("GM2 source tree not found next to gunmayhem-go: %w", err)
		}
		if err := copyGM2RuntimeAssets(gm2Root, filepath.Join(outRoot, "assets", "gm2")); err != nil {
			return err
		}
		if err := copyFile(filepath.Join(root, "MULTIPLAYER.md"), filepath.Join(outRoot, "MULTIPLAYER.md")); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(outRoot, "MULTIPLAYER_TEST.txt"), []byte(portableInstructions(directoryConfigured)), 0o644); err != nil {
			return err
		}

		if err := zipDir(outRoot, zipPath); err != nil {
			return err
		}

		fmt.Println("Portable folder:", outRoot)
		fmt.Println("ZIP:", zipPath)
		return nil
	})
}

func portableInstructions(directoryConfigured bool) string {
	mode := `MULTIPLAYER W TYM WYDANIU DZIALA TYLKO W LAN.
Internetowy multiplayer jest wylaczony, poniewaz obecna implementacja nie jest wystarczajaco niezawodna do wydania.`
	extraFiles := ""
	version := "v0.16.0 / Windows x64, multiplayer LAN only."
	if directoryConfigured {
		mode = `LAN pozostaje domyslny po uruchomieniu GunMayhem.exe.

EKSPERYMENTALNY INTERNET - OPCJONALNIE:
1. Uruchom Start-Internet.bat, aby wlaczyc Internet tylko dla tego uruchomienia gry.
2. Gosc: CUSTOM GAME -> ONLINE -> REFRESH -> wybierz pokoj -> JOIN SESSION.
3. Host: CUSTOM GAME -> ONLINE -> CREATE SESSION -> INTERNET -> OPEN -> CREATE.
4. Host wymaga oficjalnego cloudflared.exe obok GunMayhem.exe lub w PATH. Paczka nie pobiera ani nie instaluje tego narzedzia. Goscie go nie potrzebuja.
5. CREATE uruchamia publiczny, tymczasowy tunel Cloudflare. Host musi pozostawic gre i cloudflared uruchomione oraz ustawic wolne sloty jako ONLINE PLAYER.
6. Wszyscy gracze uzywaja tej samej paczki. OPEN wymaga tego samego zatwierdzonego katalogu HTTPS na kazdym komputerze; goscie nie wpisuja adresu katalogu.
7. master_server.txt zawiera adres katalogu; paczka nie uruchamia ani nie wdraza serwera katalogu.
8. Dostepnosc katalogu nie jest sprawdzana podczas pakowania. Nie wpisuj tokenow ani prywatnych zaproszen do tego pliku.
9. Host nadal prowadzi symulacje meczu. Sam katalog tylko wyswietla otwarte pokoje.

UWAGA: to eksperyment, nie potwierdzone wydanie Internetowe.
Test uruchomienia Windows oraz rzeczywisty test gry miedzy roznymi sieciami nadal oczekuja na wykonanie.`
		extraFiles = "\n- master_server.txt - zatwierdzony katalog HTTPS dla otwartych pokoi,\n- Start-Internet.bat - jawne wlaczenie eksperymentalnego Internetu dla jednej sesji."
		version = "Windows x64 / eksperymentalny Internet, LAN domyslnie; testy Windows i miedzy sieciami oczekuja."
	}
	instructions := fmt.Sprintf(`GUN MAYHEM - PORTABLE MULTIPLAYER

Ta paczka jest gotowa do skopiowania na drugi komputer. Nie wymaga instalacji Go.

%s

JAK ZAGRAC W LAN:
1. Skopiuj caly folder na drugi komputer.
2. Na komputerze hosta uruchom GunMayhem.exe.
3. Wejdz: CUSTOM GAME -> LAN -> CREATE SESSION.
4. Wybierz ACCESS: OPEN albo INVITE CODE.
5. Przy pierwszym uruchomieniu zezwol GunMayhem.exe w Zaporze Windows dla sieci prywatnej.
6. Ustaw co najmniej jeden slot P2-P4 jako ONLINE PLAYER.
7. Na drugim komputerze wejdz do LAN SERVER BROWSER i nacisnij REFRESH.
8. Dla INVITE CODE wpisz kod GM-XXXXXX w pole JOIN CODE / LAN IP:PORT.

OPEN:
- sesja pojawia sie automatycznie w przegladarce LAN,
- kod nie jest wymagany.

INVITE CODE:
- sesja nie pojawia sie na otwartej liscie,
- dolaczenie wymaga lokalnego kodu GM-XXXXXX,
- host sprawdza kod przed przydzieleniem slotu gracza.

PLIKI:
- GunMayhem.exe - kompletna gra klient/host; hostujacy gracz jest serwerem meczu,
- assets/ - wymagane zasoby gry,
- assets/gm2/ - wybrane zasoby runtime GM2,
- data/ - lokalne ustawienia i zapisy tworzone na danym komputerze,
- data/network.log - log diagnostyczny multiplayera,
- MULTIPLAYER.md - dokladniejszy opis multiplayera LAN.%s

Wersja paczki: %s
`, mode, extraFiles, version)
	return strings.ReplaceAll(strings.TrimSpace(instructions), "\n", "\r\n") + "\r\n"
}
