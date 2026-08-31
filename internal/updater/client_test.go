package updater

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersionComparison(t *testing.T) {
	tests := []struct {
		candidate string
		current   string
		want      bool
	}{
		{"v0.2.0", "v0.1.9", true},
		{"v0.2.0", "v0.2.0", false},
		{"v1.0.0", "v0.99.99", true},
		{"v2026.8.27", "v2026.8.26", true},
		{"v1.0.0", "v1.0.0-beta", true},
		{"v1.0.0-beta", "v1.0.0", false},
	}
	for _, tc := range tests {
		got, err := versionNewer(tc.candidate, tc.current)
		if err != nil {
			t.Fatalf("versionNewer(%q,%q): %v", tc.candidate, tc.current, err)
		}
		if got != tc.want {
			t.Fatalf("versionNewer(%q,%q)=%v want%v", tc.candidate, tc.current, got, tc.want)
		}
	}
}

func TestLatestReleaseSelectsPortableZip(t *testing.T) {
	oldVersion := CurrentVersion
	oldBase := githubAPIBase
	defer func() {
		CurrentVersion = oldVersion
		githubAPIBase = oldBase
	}()
	CurrentVersion = "v1.2.0"

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/snakex21/gunmayhem-go/releases/latest" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"tag_name":"v1.3.0","name":"1.3","html_url":"%s/release","body":"notes","draft":false,"prerelease":false,"assets":[{"name":"GunMayhem-Windows-x64.zip","browser_download_url":"%s/pkg.zip","size":12345}]}`, server.URL, server.URL)
	}))
	defer server.Close()
	githubAPIBase = server.URL

	release, available, err := CheckLatest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !available || release.Tag != "v1.3.0" || release.Asset.Name != "GunMayhem-Windows-x64.zip" {
		t.Fatalf("release=%+v available=%v", release, available)
	}
}

func TestDevelopmentBuildDoesNotAutoCheck(t *testing.T) {
	old := CurrentVersion
	defer func() { CurrentVersion = old }()
	CurrentVersion = "dev"
	if IsReleaseBuild() {
		t.Fatal("dev build was treated as a release")
	}
	if _, available, err := CheckLatest(context.Background()); err != nil || available {
		t.Fatalf("dev check available=%v err=%v", available, err)
	}
}
