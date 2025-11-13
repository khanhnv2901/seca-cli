package checker

import (
	"crypto/tls"
	"regexp"
	"strings"
)

// CheckMixedContent analyzes HTML content for mixed content vulnerabilities
// Mixed content occurs when HTTPS pages load HTTP resources, which can compromise security
func CheckMixedContent(htmlContent, pageURL string) *MixedContentCheck {
	// Only check if the page itself is HTTPS
	if !strings.HasPrefix(strings.ToLower(pageURL), "https://") {
		return nil
	}

	check := &MixedContentCheck{
		HasMixedContent:  false,
		MixedContentURLs: []string{},
		InsecureScripts:  0,
		InsecureStyles:   0,
		InsecureImages:   0,
		InsecureMedia:    0,
		InsecureIframes:  0,
		Severity:         "info",
	}

	// Define patterns for different resource types
	// Scripts: <script src="http://...">
	scriptPattern := regexp.MustCompile(`<script[^>]+src=['"]?(http://[^'"\s>]+)['"]?`)
	scripts := scriptPattern.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range scripts {
		if len(match) > 1 {
			check.MixedContentURLs = append(check.MixedContentURLs, match[1])
			check.InsecureScripts++
		}
	}

	// Stylesheets: <link rel="stylesheet" href="http://...">
	stylePattern := regexp.MustCompile(`<link[^>]+href=['"]?(http://[^'"\s>]+)['"]?[^>]*rel=['"]?stylesheet['"]?`)
	styles := stylePattern.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range styles {
		if len(match) > 1 {
			check.MixedContentURLs = append(check.MixedContentURLs, match[1])
			check.InsecureStyles++
		}
	}

	// Alternative stylesheet pattern
	stylePattern2 := regexp.MustCompile(`<link[^>]+rel=['"]?stylesheet['"]?[^>]*href=['"]?(http://[^'"\s>]+)['"]?`)
	styles2 := stylePattern2.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range styles2 {
		if len(match) > 1 && !containsURL(check.MixedContentURLs, match[1]) {
			check.MixedContentURLs = append(check.MixedContentURLs, match[1])
			check.InsecureStyles++
		}
	}

	// Images: <img src="http://...">
	imagePattern := regexp.MustCompile(`<img[^>]+src=['"]?(http://[^'"\s>]+)['"]?`)
	images := imagePattern.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range images {
		if len(match) > 1 {
			check.MixedContentURLs = append(check.MixedContentURLs, match[1])
			check.InsecureImages++
		}
	}

	// Media: <video>, <audio>, <source>
	mediaPattern := regexp.MustCompile(`<(?:video|audio|source)[^>]+src=['"]?(http://[^'"\s>]+)['"]?`)
	media := mediaPattern.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range media {
		if len(match) > 1 {
			check.MixedContentURLs = append(check.MixedContentURLs, match[1])
			check.InsecureMedia++
		}
	}

	// Iframes: <iframe src="http://...">
	iframePattern := regexp.MustCompile(`<iframe[^>]+src=['"]?(http://[^'"\s>]+)['"]?`)
	iframes := iframePattern.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range iframes {
		if len(match) > 1 {
			check.MixedContentURLs = append(check.MixedContentURLs, match[1])
			check.InsecureIframes++
		}
	}

	// Additional checks for inline styles and CSS imports
	// CSS @import: @import url("http://...")
	cssImportPattern := regexp.MustCompile(`@import\s+url\(['"]?(http://[^'"\s)]+)['"]?\)`)
	cssImports := cssImportPattern.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range cssImports {
		if len(match) > 1 {
			check.MixedContentURLs = append(check.MixedContentURLs, match[1])
			check.InsecureStyles++
		}
	}

	// Check for any mixed content
	if len(check.MixedContentURLs) > 0 {
		check.HasMixedContent = true

		// Determine severity based on resource types
		if check.InsecureScripts > 0 || check.InsecureIframes > 0 {
			check.Severity = "critical"
			check.Recommendation = "CRITICAL: HTTP scripts and iframes on HTTPS pages create severe security vulnerabilities. " +
				"Attackers can inject malicious code or capture sensitive data. Migrate all resources to HTTPS immediately."
		} else if check.InsecureStyles > 0 {
			check.Severity = "high"
			check.Recommendation = "HIGH RISK: HTTP stylesheets on HTTPS pages can be manipulated to inject malicious content " +
				"or phishing overlays. Update stylesheet URLs to HTTPS."
		} else if check.InsecureMedia > 0 {
			check.Severity = "medium"
			check.Recommendation = "MEDIUM RISK: HTTP media resources (audio/video) on HTTPS pages can be intercepted and replaced. " +
				"While less critical than scripts, migrate to HTTPS for complete security."
		} else if check.InsecureImages > 0 {
			check.Severity = "medium"
			check.Recommendation = "MEDIUM RISK: HTTP images on HTTPS pages can be replaced with misleading content. " +
				"Browsers may display warnings. Migrate images to HTTPS or use a CDN with SSL."
		}
	}

	return check
}

// CheckOCSPStapling verifies if the server supports OCSP stapling
// OCSP stapling improves security and performance by having the server
// provide certificate status rather than the client querying the CA
func CheckOCSPStapling(connState *tls.ConnectionState) bool {
	if connState == nil {
		return false
	}

	// Check if OCSP response is present and valid
	if len(connState.OCSPResponse) > 0 {
		return true
	}

	return false
}

// containsURL checks if a string slice contains a specific URL string
func containsURL(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// AnalyzeMixedContentSummary provides a summary of mixed content issues
func AnalyzeMixedContentSummary(check *MixedContentCheck) string {
	if check == nil || !check.HasMixedContent {
		return ""
	}

	summary := "Mixed content detected: "
	issues := []string{}

	if check.InsecureScripts > 0 {
		issues = append(issues, "scripts")
	}
	if check.InsecureStyles > 0 {
		issues = append(issues, "stylesheets")
	}
	if check.InsecureIframes > 0 {
		issues = append(issues, "iframes")
	}
	if check.InsecureImages > 0 {
		issues = append(issues, "images")
	}
	if check.InsecureMedia > 0 {
		issues = append(issues, "media")
	}

	return summary + strings.Join(issues, ", ")
}
