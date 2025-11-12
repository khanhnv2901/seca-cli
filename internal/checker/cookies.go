package checker

import (
	"net/http"
)

// AnalyzeCookies inspects Set-Cookie headers for missing Secure/HttpOnly flags.
func AnalyzeCookies(resp *http.Response) []CookieFinding {
	if resp == nil {
		return nil
	}

	raw := resp.Header["Set-Cookie"]
	if len(raw) == 0 {
		return nil
	}

	findings := make([]CookieFinding, 0)
	for i, cookie := range resp.Cookies() {
		finding := CookieFinding{
			Name:            cookie.Name,
			MissingSecure:   !cookie.Secure,
			MissingHTTPOnly: !cookie.HttpOnly,
		}
		if i < len(raw) {
			finding.OriginalSetCookie = raw[i]
		}
		if finding.MissingSecure || finding.MissingHTTPOnly {
			findings = append(findings, finding)
		}
	}
	return findings
}
