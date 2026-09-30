package masterapi

import (
	"strings"
	"testing"
)

func TestNormalizePublicDirectoryURL(t *testing.T) {
	for _, raw := range []string{
		"", "https://:443", "http://localhost:7780", "https://user:secret@example.com",
		"https://example.com?token=secret", "https://example.com#secret", "https://example.com?",
		"https://example.com/%2f", "https:example.com", "https://example.com:invalid",
		" https://example.com", "https://example.com ", "https://example.com\n",
		"https://example.com\nhttps://other.example", "https://example.com/\x00",
		"https://example.com/" + strings.Repeat("a", MaxPublicDirectoryURLBytes),
	} {
		if got := NormalizePublicDirectoryURL(raw); got != "" {
			t.Error("unsafe public directory URL accepted")
		}
	}
	for _, tc := range []struct{ raw, want string }{
		{"https://example.com", "https://example.com"},
		{"https://example.com/base///", "https://example.com/base"},
		{"https://example.com:8443/base/", "https://example.com:8443/base"},
		{"https://[::1]:8443/", "https://[::1]:8443"},
	} {
		if got := NormalizePublicDirectoryURL(tc.raw); got != tc.want {
			t.Errorf("NormalizePublicDirectoryURL(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestNormalizePublicDirectoryURLSizeBoundary(t *testing.T) {
	prefix := "https://example.com/"
	raw := prefix + strings.Repeat("a", MaxPublicDirectoryURLBytes-len(prefix))
	if NormalizePublicDirectoryURL(raw) != raw {
		t.Fatal("maximum-length public directory URL rejected")
	}
	if NormalizePublicDirectoryURL(raw+"a") != "" {
		t.Fatal("oversized public directory URL accepted")
	}
}
