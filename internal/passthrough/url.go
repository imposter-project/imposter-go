package passthrough

import (
	"fmt"
	"net/url"
	"strings"
)

// JoinURL combines an upstream base URL with the incoming request path and
// query string. The upstream's base path (if any) is prefixed to the request
// path, mirroring the JVM engine's path-joining behaviour. The raw query
// string is forwarded verbatim without re-encoding.
//
// Examples:
//
//	base http://api/v1, path /users           -> http://api/v1/users
//	base http://api/v1/, path /users          -> http://api/v1/users
//	base http://api, path /v1/users           -> http://api/v1/users
//	base http://api/v1, path /, query a=1     -> http://api/v1/?a=1
func JoinURL(base, requestPath, rawQuery string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid upstream URL %q: %w", base, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("upstream URL %q must include a scheme and host", base)
	}

	u.Path = joinPaths(u.Path, requestPath)
	u.RawQuery = rawQuery
	// Avoid url.URL re-encoding the forwarded path/query.
	u.RawPath = ""
	return u.String(), nil
}

// joinPaths concatenates a base path and an appended path, normalising the
// slash at the boundary while preserving a trailing slash on the result.
func joinPaths(base, appendPath string) string {
	if appendPath == "" {
		return base
	}
	if base == "" || base == "/" {
		return appendPath
	}
	base = strings.TrimSuffix(base, "/")
	if !strings.HasPrefix(appendPath, "/") {
		appendPath = "/" + appendPath
	}
	return base + appendPath
}
