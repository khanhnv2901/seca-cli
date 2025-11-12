package checker

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	consts "github.com/khanhnv2901/seca-cli/internal/constants"
)

// HTTPChecker performs HTTP/HTTPS checks with TLS monitoring
type HTTPChecker struct {
	Timeout    time.Duration
	CaptureRaw bool
	RawHandler func(target string, headers http.Header, bodySnippet string) error
}

const bodySnippetLimit = 32768

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
	usedGET := false
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
		usedGET = true
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

	readLimit := int64(bodySnippetLimit)
	if rawLimit := int64(consts.RawCaptureLimitBytes); rawLimit > readLimit {
		readLimit = rawLimit
	}
	var bodySnippet []byte
	var bodyErr error
	if usedGET || (resp.Request != nil && resp.Request.Method == http.MethodGet) {
		bodySnippet, bodyErr = readBodySnippet(resp.Body, readLimit)
	} else {
		bodySnippet, bodyErr = fetchBodySnippet(ctx, client, u, readLimit)
		_, _ = io.Copy(io.Discard, resp.Body)
	}
	if bodyErr != nil {
		appendNote(&result, fmt.Sprintf("warning: failed to read response body: %v", bodyErr))
	}
	if h.CaptureRaw && h.RawHandler != nil && len(bodySnippet) > 0 {
		rawBytes := bodySnippet
		if len(rawBytes) > consts.RawCaptureLimitBytes {
			rawBytes = rawBytes[:consts.RawCaptureLimitBytes]
		}
		if err := h.RawHandler(target, resp.Header, string(rawBytes)); err != nil {
			appendNote(&result, fmt.Sprintf("warning: failed to save raw capture: %v", err))
		}
	}

	// Check for robots.txt (safe, small GET)
	if parsed != nil {
		checkRobotsAndSitemap(ctx, client, parsed, &result)
		if len(bodySnippet) > 0 {
			if scripts := AnalyzeThirdPartyScripts(string(bodySnippet), parsed); len(scripts) > 0 {
				result.ThirdPartyScripts = scripts
				appendNote(&result, fmt.Sprintf("%d third-party script(s) detected", len(scripts)))
			}
		}
	}

	return result
}

// Name returns the name of this checker
func (h *HTTPChecker) Name() string {
	return "check http"
}

func checkRobotsAndSitemap(ctx context.Context, client *http.Client, parsed *url.URL, result *CheckResult) {
	base := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	checkRel := func(path string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", base+path, nil)
		if err != nil {
			return nil, err
		}
		return client.Do(req)
	}

	robotsResp, err := checkRel("/robots.txt")
	if err == nil {
		defer robotsResp.Body.Close()
		if robotsResp.StatusCode == http.StatusOK {
			data, _ := io.ReadAll(io.LimitReader(robotsResp.Body, 8192))
			summarizeRobots(string(data), result)
		}
		_, _ = io.Copy(io.Discard, robotsResp.Body)
	}

	sitemapResp, err := checkRel("/sitemap.xml")
	if err == nil {
		defer sitemapResp.Body.Close()
		if sitemapResp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(sitemapResp.Body, 20480))
			discovered := analyzeSitemapURLs(string(body))
			addSitemapNote(result, discovered)
		}
		_, _ = io.Copy(io.Discard, sitemapResp.Body)
	}
}

func summarizeRobots(content string, result *CheckResult) {
	if content == "" {
		appendNote(result, "robots.txt found")
		return
	}
	scanner := bufio.NewScanner(strings.NewReader(content))
	disallow := make([]string, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "disallow:") {
			path := strings.TrimSpace(line[len("Disallow:"):])
			if path != "" {
				disallow = append(disallow, path)
			}
		}
	}
	if len(disallow) == 0 {
		appendNote(result, "robots.txt found")
		return
	}
	preview := disallow
	if len(preview) > 5 {
		preview = preview[:5]
	}
	note := fmt.Sprintf("robots.txt discloses %d path(s): %s", len(disallow), strings.Join(preview, ", "))
	appendNote(result, note)
}

func analyzeSitemapURLs(data string) []string {
	lines := strings.Split(data, "\n")
	urls := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "<loc>") && strings.HasSuffix(line, "</loc>") {
			urls = append(urls, strings.TrimSuffix(strings.TrimPrefix(line, "<loc>"), "</loc>"))
		}
	}
	return urls
}

func addSitemapNote(result *CheckResult, urls []string) {
	if len(urls) == 0 {
		appendNote(result, "sitemap discovered")
		return
	}
	preview := urls
	if len(preview) > 5 {
		preview = preview[:5]
	}
	note := fmt.Sprintf("sitemap exposes %d URL(s), sample: %s", len(urls), strings.Join(preview, ", "))
	appendNote(result, note)
}

func appendNote(result *CheckResult, msg string) {
	if result.Notes != "" {
		result.Notes += "; " + msg
	} else {
		result.Notes = msg
	}
}

func readBodySnippet(body io.ReadCloser, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, limit))
	if _, drainErr := io.Copy(io.Discard, body); err == nil && drainErr != nil {
		err = drainErr
	}
	return data, err
}

func fetchBodySnippet(ctx context.Context, client *http.Client, target string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(io.LimitReader(resp.Body, limit))
	if _, drainErr := io.Copy(io.Discard, resp.Body); readErr == nil && drainErr != nil {
		readErr = drainErr
	}
	return data, readErr
}
