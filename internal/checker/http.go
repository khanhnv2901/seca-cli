package checker

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
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

	// Normalize URL
	u := target
	parsed, err := url.Parse(target)
	if err != nil || parsed.Scheme == "" {
		u = "http://" + target
		parsed, _ = url.Parse(u)
	}

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

	// Check TLS certificate expiry
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		result.TLSExpiry = cert.NotAfter.Format(time.RFC3339)

		// Warn if expiring within 14 days
		if time.Until(cert.NotAfter) < (14 * 24 * time.Hour) {
			result.Notes = "TLS certificate expires soon"
		}
	}

	// Optional raw response capture
	if h.CaptureRaw && h.RawHandler != nil {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		_ = h.RawHandler(target, resp.Header, string(bodyBytes))
	} else {
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
