package shared

import (
	"net/url"
	"strings"
)

// ServerBasePath returns the path component of the given server URL, with any
// trailing slash removed. Plugins prepend it to browser-facing, root-relative
// URLs so they resolve correctly when Imposter is hosted behind a reverse proxy
// under a base path (e.g. /myapp). Internal route matching is unaffected, as the
// proxy strips the base path before requests reach Imposter.
//
// It returns an empty string when the server URL has no path component or cannot
// be parsed.
func ServerBasePath(serverURL string) string {
	if parsed, err := url.Parse(serverURL); err == nil {
		return strings.TrimRight(parsed.Path, "/")
	}
	return ""
}
