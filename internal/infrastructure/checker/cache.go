package checker

import (
	"net/http"
	"strings"
)

// AnalyzeCachePolicy extracts cache headers for visibility/compliance.
func AnalyzeCachePolicy(h http.Header) *CachePolicy {
	if h == nil {
		return nil
	}

	policy := &CachePolicy{
		CacheControl: h.Get("Cache-Control"),
		Expires:      h.Get("Expires"),
		Pragma:       h.Get("Pragma"),
	}

	cc := strings.ToLower(policy.CacheControl)
	if policy.CacheControl == "" && policy.Expires == "" {
		policy.Issues = append(policy.Issues, "No caching headers (Cache-Control/Expires) present")
	} else {
		if policy.CacheControl == "" {
			policy.Issues = append(policy.Issues, "Cache-Control header missing")
		}
		if policy.Expires == "" {
			policy.Issues = append(policy.Issues, "Expires header missing")
		}
	}

	if policy.CacheControl != "" && !strings.Contains(cc, "max-age") && !strings.Contains(cc, "no-cache") && !strings.Contains(cc, "no-store") {
		policy.Issues = append(policy.Issues, "Cache-Control lacks explicit max-age/no-cache directives")
	}

	if policy.Pragma == "no-cache" {
		policy.Issues = append(policy.Issues, "Pragma: no-cache detected (legacy caching directive)")
	}

	if policy.CacheControl == "" && policy.Expires == "" && policy.Pragma == "" {
		// Nothing to report beyond the default issue
		return policy
	}

	return policy
}
