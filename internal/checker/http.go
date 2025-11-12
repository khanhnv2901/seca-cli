package checker

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	consts "github.com/khanhnv2901/seca-cli/internal/constants"
)

// HTTPChecker performs HTTP/HTTPS checks with TLS monitoring
type HTTPChecker struct {
	Timeout    time.Duration
	CaptureRaw bool
	RawHandler func(target string, headers http.Header, bodySnippet string) error
}

// Check performs an HTTP/HTTPS check on the target
func (h *HTTPChecker) Check(ctx context.Context, target string) CheckResult {
	result := CheckResult{
		Target:    target,
		CheckedAt: time.Now().UTC(),
	}

	// Normalize URL using shared utility
	targetInfo := ParseTarget(target)
	u := targetInfo.FullURL
	parsed, _ := url.Parse(u)

	// Create HTTP client
	client := &http.Client{
		Timeout: h.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	// Try HEAD request first (safe, minimal side effects)
	req, err := http.NewRequestWithContext(ctx, "HEAD", u, nil)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("create request: %v", err)
		return result
	}

	resp, err := client.Do(req)
	if err != nil {
		// Fallback to GET (some servers disallow HEAD)
		req2, err2 := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err2 != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("create GET request: %v", err2)
			return result
		}

		resp2, err2 := client.Do(req2)
		if err2 != nil {
			result.Status = "error"
			result.Error = err2.Error()
			return result
		}
		resp = resp2
	}
	defer resp.Body.Close()

	// Extract HTTP information
	result.HTTPStatus = resp.StatusCode
	result.ServerHeader = resp.Header.Get("Server")
	result.Status = "ok"

	// Analyze security headers
	result.SecurityHeaders = AnalyzeSecurityHeaders(resp.Header)

	// Analyze cookies for Secure/HttpOnly flags (OWASP ASVS §3.4)
	if cookieFindings := AnalyzeCookies(resp); len(cookieFindings) > 0 {
		result.CookieFindings = cookieFindings
		if result.Notes != "" {
			result.Notes += "; "
		}
		result.Notes += fmt.Sprintf("%d cookie(s) missing Secure or HttpOnly flag", len(cookieFindings))
	}

	// Inspect CORS headers for risky configurations (OWASP Top 10 A5:2021)
	if corsReport := AnalyzeCORS(resp); corsReport != nil {
		result.CORSInsights = corsReport
		if result.Notes != "" {
			result.Notes += "; "
		}
		result.Notes += "CORS policy needs review"
	}

	// Analyze TLS/crypto compliance (OWASP ASVS §9, PCI DSS 4.1)
	if resp.TLS != nil {
		result.TLSCompliance = AnalyzeTLSCompliance(resp.TLS)

		// Legacy TLS expiry field for backward compatibility
		if len(resp.TLS.PeerCertificates) > 0 {
			cert := resp.TLS.PeerCertificates[0]
			result.TLSExpiry = cert.NotAfter.Format(time.RFC3339)

			// Warn if expiring within 14 days
			if time.Until(cert.NotAfter) < consts.TLSSoonExpiryWindow {
				if result.Notes != "" {
					result.Notes += "; TLS certificate expires soon"
				} else {
					result.Notes = "TLS certificate expires soon"
				}
			}
		}

		// Add compliance warnings to notes
		if result.TLSCompliance != nil && !result.TLSCompliance.Compliant {
			criticalIssues := 0
			for _, issue := range result.TLSCompliance.Issues {
				if issue.Severity == "critical" {
					criticalIssues++
				}
			}
			if criticalIssues > 0 {
				if result.Notes != "" {
					result.Notes += fmt.Sprintf("; %d critical TLS compliance issue(s)", criticalIssues)
				} else {
					result.Notes = fmt.Sprintf("%d critical TLS compliance issue(s)", criticalIssues)
				}
			}
		}
	}

	// Optional raw response capture
	if h.CaptureRaw && h.RawHandler != nil {
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, int64(consts.RawCaptureLimitBytes)))
		if err != nil {
			// Log but don't fail - partial body is acceptable
			if result.Notes != "" {
				result.Notes += fmt.Sprintf("; warning: failed to read response body: %v", err)
			} else {
				result.Notes = fmt.Sprintf("warning: failed to read response body: %v", err)
			}
		}
		if err := h.RawHandler(target, resp.Header, string(bodyBytes)); err != nil {
			// Log but don't fail - raw capture is optional
			if result.Notes != "" {
				result.Notes += fmt.Sprintf("; warning: failed to save raw capture: %v", err)
			} else {
				result.Notes = fmt.Sprintf("warning: failed to save raw capture: %v", err)
			}
		}
	} else {
		// Discard response body - ignore errors as this is just cleanup
		_, _ = io.Copy(io.Discard, resp.Body)
	}

	// Check for robots.txt (safe, small GET)
	if parsed != nil {
		robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsed.Scheme, parsed.Host)
		robotsReq, err := http.NewRequestWithContext(ctx, "GET", robotsURL, nil)
		if err == nil {
			robotsResp, err := client.Do(robotsReq)
			if err == nil {
				defer robotsResp.Body.Close()
				if robotsResp.StatusCode == 200 {
					if result.Notes != "" {
						result.Notes += "; robots.txt found"
					} else {
						result.Notes = "robots.txt found"
					}
				}
				_, _ = io.Copy(io.Discard, robotsResp.Body)
			}
		}
	}

	return result
}

// Name returns the name of this checker
func (h *HTTPChecker) Name() string {
	return "check http"
}
