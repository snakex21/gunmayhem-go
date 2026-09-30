package masterapi

import (
	"net/url"
	"strings"
)

const MaxPublicDirectoryURLBytes = 2048

// NormalizePublicDirectoryURL is the shared policy for public-room runtime
// configuration and portable packages. An empty result means invalid input.
// Callers must not log the raw URL or URL parser errors, which may hold secrets.
func NormalizePublicDirectoryURL(raw string) string {
	if len(raw) > MaxPublicDirectoryURLBytes || raw != strings.TrimSpace(raw) {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" {
		return ""
	}
	return strings.TrimRight(u.String(), "/")
}
