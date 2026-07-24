package nginxproxy

import "strings"

// JoinPath joins two URL path segments with exactly one slash between them.
func JoinPath(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	aSlash := strings.HasSuffix(a, "/")
	bSlash := strings.HasPrefix(b, "/")
	switch {
	case aSlash && bSlash:
		return a + b[1:]
	case !aSlash && !bSlash:
		return a + "/" + b
	default:
		return a + b
	}
}

// MatchLocation reports whether requestPath matches an nginx prefix location.
//
//	location /api/  matches /api and /api/ and /api/x
//	location /api   matches /api and /api/ and /api/x
func MatchLocation(location, requestPath string) bool {
	if location == "" || requestPath == "" {
		return false
	}
	if !strings.HasPrefix(location, "/") {
		location = "/" + location
	}
	if strings.HasSuffix(location, "/") {
		trimmed := strings.TrimSuffix(location, "/")
		return requestPath == trimmed || strings.HasPrefix(requestPath, location) || requestPath == location
	}
	return requestPath == location || strings.HasPrefix(requestPath, location+"/")
}

// LocationRemainder returns the request path suffix after the matched location prefix.
// For location "/api/" and request "/api/users", remainder is "users".
// For location "/api" and request "/api/users", remainder is "/users".
func LocationRemainder(location, requestPath string) string {
	if !MatchLocation(location, requestPath) {
		return ""
	}
	if !strings.HasPrefix(location, "/") {
		location = "/" + location
	}
	if strings.HasSuffix(location, "/") {
		trimmed := strings.TrimSuffix(location, "/")
		if requestPath == trimmed || requestPath == location {
			return ""
		}
		if strings.HasPrefix(requestPath, location) {
			return requestPath[len(location):]
		}
		return ""
	}
	if requestPath == location {
		return ""
	}
	if strings.HasPrefix(requestPath, location) {
		return requestPath[len(location):]
	}
	return ""
}

// ResolveUpstreamPath implements nginx proxy_pass path resolution.
//
// nginx rules:
//   - proxy_pass http://host:port;        (no URI)  -> pass full original request path
//   - proxy_pass http://host:port/;       (has URI) -> replace location prefix with URI
//   - proxy_pass http://host:port/v2/;    (has URI) -> replace location prefix with /v2/
//
// rewrite, when non-empty, acts as the proxy_pass URI and overrides targetPath.
func ResolveUpstreamPath(location, requestPath, targetPath, rewrite string) string {
	if location == "" {
		return requestPath
	}
	if !strings.HasPrefix(location, "/") {
		location = "/" + location
	}

	replacement := ""
	useReplace := false
	if strings.TrimSpace(rewrite) != "" {
		useReplace = true
		replacement = rewrite
		if !strings.HasPrefix(replacement, "/") {
			replacement = "/" + replacement
		}
	} else if targetPath != "" {
		// Non-empty path (including "/") means proxy_pass was given a URI.
		useReplace = true
		replacement = targetPath
	}

	if !useReplace {
		return requestPath
	}

	remainder := LocationRemainder(location, requestPath)
	if remainder == "" {
		if replacement == "" {
			return "/"
		}
		return replacement
	}
	return JoinPath(replacement, remainder)
}

// NormalizeLocation ensures a location path starts with "/".
func NormalizeLocation(location string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return ""
	}
	if !strings.HasPrefix(location, "/") {
		location = "/" + location
	}
	return location
}
