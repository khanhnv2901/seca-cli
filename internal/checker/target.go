package checker

import (
	"net/url"
	"strings"
)

// TargetInfo contains parsed target information
type TargetInfo struct {
	Original string // Original target string
	Scheme   string // http, https, or empty
	Host     string // Hostname (without protocol, path, port)
	Port     string // Port if specified
	Path     string // Path if specified
	FullURL  string // Full normalized URL (for HTTP requests)
}

// ParseTarget parses a target string into structured components.
// This handles various input formats:
//   - example.com
//   - http://example.com
//   - https://example.com:443/path
//   - example.com:8080
func ParseTarget(target string) *TargetInfo {
	info := &TargetInfo{
		Original: target,
	}

	// Try to parse as URL first
	parsed, err := url.Parse(target)

	// If parsing fails OR scheme is empty OR scheme doesn't look like a real scheme (contains dots)
	// then prepend http:// and parse again
	if err != nil || parsed.Scheme == "" || strings.Contains(parsed.Scheme, ".") {
		// Not a valid URL or missing scheme, try to parse manually
		parsed, _ = url.Parse("http://" + target)
	}

	// Extract components
	if parsed != nil {
		info.Scheme = parsed.Scheme
		info.Host = parsed.Hostname()
		info.Port = parsed.Port()
		info.Path = parsed.Path
		info.FullURL = parsed.String()
	}

	// Fallback: if URL parsing completely failed, extract host manually
	if info.Host == "" {
		host := target
		// Remove common protocols
		host = strings.TrimPrefix(host, "http://")
		host = strings.TrimPrefix(host, "https://")
		// Remove path
		host = strings.Split(host, "/")[0]
		// Split host and port
		parts := strings.Split(host, ":")
		info.Host = parts[0]
		if len(parts) > 1 {
			info.Port = parts[1]
		}
		// Build full URL for HTTP requests
		if info.Scheme == "" {
			info.Scheme = "http"
		}
		info.FullURL = info.Scheme + "://" + host
	}

	return info
}

// NormalizeHTTPTarget normalizes a target for HTTP/HTTPS requests.
// Returns a full URL with scheme.
func NormalizeHTTPTarget(target string) string {
	info := ParseTarget(target)
	return info.FullURL
}

// ExtractHost extracts just the hostname from a target.
// This is useful for DNS lookups where we need the bare hostname.
func ExtractHost(target string) string {
	info := ParseTarget(target)
	return info.Host
}
