package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	repoOwner = "snakex21"
	repoName  = "gunmayhem-go"
)

// CurrentVersion is injected by build.bat for tagged/release builds.
// Local development builds intentionally remain "dev" so they never try to
// replace a newer checkout with an older GitHub release.
var CurrentVersion = "dev"

var githubAPIBase = "https://api.github.com"

const maxUpdatePackageBytes int64 = 2 << 30 // 2 GiB safety cap

type Asset struct {
	Name   string
	URL    string
	Size   int64
	Digest string
}

type Release struct {
	Tag     string
	Name    string
	HTMLURL string
	Body    string
	Asset   Asset
}

type githubRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	HTMLURL    string `json:"html_url"`
	Body       string `json:"body"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
		Digest             string `json:"digest"`
	} `json:"assets"`
}

func IsReleaseBuild() bool {
	_, ok := parseVersion(CurrentVersion)
	return ok && !strings.Contains(strings.ToLower(CurrentVersion), "dev")
}

func CheckLatest(ctx context.Context) (Release, bool, error) {
	if !IsReleaseBuild() {
		return Release{}, false, nil
	}
	url := strings.TrimRight(githubAPIBase, "/") + "/repos/" + repoOwner + "/" + repoName + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Release{}, false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "GunMayhem-Updater/"+CurrentVersion)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Release{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Release{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return Release{}, false, fmt.Errorf("GitHub release check: HTTP %d", resp.StatusCode)
	}
	var gh githubRelease
	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := dec.Decode(&gh); err != nil {
		return Release{}, false, err
	}
	if gh.Draft || gh.Prerelease {
		return Release{}, false, nil
	}
	newer, err := versionNewer(gh.TagName, CurrentVersion)
	if err != nil {
		return Release{}, false, err
	}
	if !newer {
		return Release{}, false, nil
	}
	asset, ok := pickPortableAsset(gh)
	if !ok {
		return Release{}, false, errors.New("new release has no supported Windows ZIP asset")
	}
	return Release{Tag: gh.TagName, Name: gh.Name, HTMLURL: gh.HTMLURL, Body: gh.Body, Asset: asset}, true, nil
}

func pickPortableAsset(gh githubRelease) (Asset, bool) {
	preferred := []string{"gunmayhem-windows-x64.zip", "gunmayhem-multiplayer-portable.zip"}
	for _, want := range preferred {
		for _, a := range gh.Assets {
			if strings.EqualFold(a.Name, want) && a.BrowserDownloadURL != "" && a.Size > 0 && a.Size <= maxUpdatePackageBytes {
				return Asset{Name: a.Name, URL: a.BrowserDownloadURL, Size: a.Size, Digest: a.Digest}, true
			}
		}
	}
	for _, a := range gh.Assets {
		name := strings.ToLower(a.Name)
		if strings.HasSuffix(name, ".zip") && strings.Contains(name, "gunmayhem") && a.BrowserDownloadURL != "" && a.Size > 0 && a.Size <= maxUpdatePackageBytes {
			return Asset{Name: a.Name, URL: a.BrowserDownloadURL, Size: a.Size, Digest: a.Digest}, true
		}
	}
	return Asset{}, false
}

func Download(ctx context.Context, release Release, dir string, progress func(done, total int64)) (string, error) {
	if release.Asset.URL == "" || release.Asset.Name == "" {
		return "", errors.New("release has no downloadable asset")
	}
	if release.Asset.Size <= 0 || release.Asset.Size > maxUpdatePackageBytes {
		return "", errors.New("update package size is invalid")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	finalPath := filepath.Join(dir, filepath.Base(release.Asset.Name))
	partPath := finalPath + ".part"
	_ = os.Remove(partPath)
	_ = os.Remove(finalPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, release.Asset.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GunMayhem-Updater/"+CurrentVersion)
	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download update: HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxUpdatePackageBytes {
		return "", errors.New("update package is too large")
	}

	f, err := os.Create(partPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()

	total := release.Asset.Size
	if resp.ContentLength > 0 {
		total = resp.ContentLength
	}
	buf := make([]byte, 128*1024)
	hash := sha256.New()
	var done int64
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			done += int64(n)
			if done > maxUpdatePackageBytes {
				_ = f.Close()
				_ = os.Remove(partPath)
				return "", errors.New("update package exceeded safety limit")
			}
			if _, err := f.Write(buf[:n]); err != nil {
				_ = os.Remove(partPath)
				return "", err
			}
			_, _ = hash.Write(buf[:n])
			if progress != nil {
				progress(done, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = os.Remove(partPath)
			return "", readErr
		}
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(partPath)
		return "", err
	}
	if release.Asset.Size > 0 && done != release.Asset.Size {
		_ = os.Remove(partPath)
		return "", fmt.Errorf("incomplete update download: got %d want %d", done, release.Asset.Size)
	}
	if digest := strings.TrimSpace(release.Asset.Digest); strings.HasPrefix(strings.ToLower(digest), "sha256:") {
		want := strings.TrimSpace(digest[len("sha256:"):])
		got := hex.EncodeToString(hash.Sum(nil))
		if !strings.EqualFold(want, got) {
			_ = os.Remove(partPath)
			return "", errors.New("update package SHA-256 verification failed")
		}
	}
	if err := os.Rename(partPath, finalPath); err != nil {
		_ = os.Remove(partPath)
		return "", err
	}
	return finalPath, nil
}

type versionParts struct {
	nums       []int
	prerelease string
}

func parseVersion(v string) (versionParts, bool) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
	if v == "" || strings.Contains(strings.ToLower(v), "dev") {
		return versionParts{}, false
	}
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	pre := ""
	if i := strings.IndexByte(v, '-'); i >= 0 {
		pre = v[i+1:]
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) < 2 || len(parts) > 4 {
		return versionParts{}, false
	}
	nums := make([]int, len(parts))
	for i, p := range parts {
		if p == "" {
			return versionParts{}, false
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return versionParts{}, false
		}
		nums[i] = n
	}
	return versionParts{nums: nums, prerelease: pre}, true
}

func versionNewer(candidate, current string) (bool, error) {
	a, ok := parseVersion(candidate)
	if !ok {
		return false, fmt.Errorf("unsupported release version %q", candidate)
	}
	b, ok := parseVersion(current)
	if !ok {
		return false, fmt.Errorf("unsupported current version %q", current)
	}
	max := len(a.nums)
	if len(b.nums) > max {
		max = len(b.nums)
	}
	for i := 0; i < max; i++ {
		av, bv := 0, 0
		if i < len(a.nums) {
			av = a.nums[i]
		}
		if i < len(b.nums) {
			bv = b.nums[i]
		}
		if av != bv {
			return av > bv, nil
		}
	}
	if a.prerelease == b.prerelease {
		return false, nil
	}
	if a.prerelease == "" && b.prerelease != "" {
		return true, nil
	}
	return false, nil
}
