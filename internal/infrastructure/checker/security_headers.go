package checker

import (
	"net/http"
	"strings"
)

// SecurityHeaderSpec defines the specification for a security header
type SecurityHeaderSpec struct {
	Name           string
	Severity       string // "critical", "high", "medium", "low"
	MaxScore       int
	CheckFunc      func(value string) (int, []string, string)
	Recommendation string
}

// securityHeaderSpecs defines all security headers to check
var securityHeaderSpecs = map[string]SecurityHeaderSpec{
	"Strict-Transport-Security": {
		Name:           "Strict-Transport-Security",
		Severity:       "high",
		MaxScore:       20,
		CheckFunc:      checkHSTS,
		Recommendation: "Add 'Strict-Transport-Security: max-age=31536000; includeSubDomains; preload'",
	},
	"Content-Security-Policy": {
		Name:           "Content-Security-Policy",
		Severity:       "high",
		MaxScore:       20,
		CheckFunc:      checkCSP,
		Recommendation: "Implement a strict Content-Security-Policy appropriate for your application",
	},
	"X-Frame-Options": {
		Name:           "X-Frame-Options",
		Severity:       "high",
		MaxScore:       15,
		CheckFunc:      checkXFrameOptions,
		Recommendation: "Add 'X-Frame-Options: DENY' or 'SAMEORIGIN'",
	},
	"X-Content-Type-Options": {
		Name:           "X-Content-Type-Options",
		Severity:       "high",
		MaxScore:       15,
		CheckFunc:      checkXContentTypeOptions,
		Recommendation: "Add 'X-Content-Type-Options: nosniff'",
	},
	"Referrer-Policy": {
		Name:           "Referrer-Policy",
		Severity:       "medium",
		MaxScore:       10,
		CheckFunc:      checkReferrerPolicy,
		Recommendation: "Add 'Referrer-Policy: strict-origin-when-cross-origin' or 'no-referrer'",
	},
	"Permissions-Policy": {
		Name:           "Permissions-Policy",
		Severity:       "medium",
		MaxScore:       10,
		CheckFunc:      checkPermissionsPolicy,
		Recommendation: "Add 'Permissions-Policy' to control browser features (e.g., 'geolocation=(), microphone=()')",
	},
	"Cross-Origin-Opener-Policy": {
		Name:           "Cross-Origin-Opener-Policy",
		Severity:       "medium",
		MaxScore:       5,
		CheckFunc:      checkCOOP,
		Recommendation: "Add 'Cross-Origin-Opener-Policy: same-origin'",
	},
	"Cross-Origin-Embedder-Policy": {
		Name:           "Cross-Origin-Embedder-Policy",
		Severity:       "medium",
		MaxScore:       5,
		CheckFunc:      checkCOEP,
		Recommendation: "Add 'Cross-Origin-Embedder-Policy: require-corp'",
	},
	"Content-Type": {
		Name:           "Content-Type",
		Severity:       "medium",
		MaxScore:       5,
		CheckFunc:      checkContentType,
		Recommendation: "Add 'Content-Type' header with appropriate charset (e.g., 'text/html; charset=utf-8')",
	},
}

// informationDisclosureHeaders lists headers that should be removed/obfuscated
var informationDisclosureHeaders = []string{
	"Server",
	"X-Powered-By",
	"X-AspNet-Version",
	"X-AspNetMvc-Version",
}

// AnalyzeSecurityHeaders analyzes HTTP response headers for security best practices
func AnalyzeSecurityHeaders(headers http.Header) *SecurityHeadersResult {
	result := &SecurityHeadersResult{
		Headers:         make(map[string]HeaderStatus),
		Missing:         []string{},
		Warnings:        []string{},
		Recommendations: []string{},
		MaxScore:        105, // Updated to include Content-Type (5 points)
	}

	totalScore := 0

	// Check each security header
	for headerName, spec := range securityHeaderSpecs {
		value := headers.Get(headerName)

		if value == "" {
			// Header is missing
			result.Headers[headerName] = HeaderStatus{
				Present:        false,
				Severity:       spec.Severity,
				Score:          0,
				MaxScore:       spec.MaxScore,
				Recommendation: spec.Recommendation,
			}
			result.Missing = append(result.Missing, headerName)
		} else {
			// Header is present, evaluate it
			score, issues, recommendation := spec.CheckFunc(value)

			status := HeaderStatus{
				Present:        true,
				Value:          value,
				Severity:       spec.Severity,
				Score:          score,
				MaxScore:       spec.MaxScore,
				Issues:         issues,
				Recommendation: recommendation,
			}

			result.Headers[headerName] = status
			totalScore += score
		}
	}

	// Check for deprecated headers
	checkDeprecatedHeaders(headers, result)

	// Check for information disclosure
	checkInformationDisclosure(headers, result)

	result.Score = totalScore
	result.Grade = calculateGrade(totalScore, result.MaxScore)

	return result
}

// checkHSTS validates the Strict-Transport-Security header
func checkHSTS(value string) (int, []string, string) {
	issues := []string{}
	score := 20
	recommendation := ""

	value = strings.ToLower(value)

	// Check max-age
	if !strings.Contains(value, "max-age=") {
		issues = append(issues, "Missing 'max-age' directive")
		score -= 10
	} else if strings.Contains(value, "max-age=0") {
		issues = append(issues, "max-age is set to 0 (HSTS disabled)")
		score = 0
	} else {
		// Check if max-age is sufficient (at least 6 months)
		if !strings.Contains(value, "max-age=31536000") && !strings.Contains(value, "max-age=63072000") {
			issues = append(issues, "Consider increasing max-age to at least 31536000 (1 year)")
			score -= 3
		}
	}

	// Check includeSubDomains
	if !strings.Contains(value, "includesubdomains") {
		issues = append(issues, "Missing 'includeSubDomains' directive")
		score -= 5
		recommendation = "Add 'includeSubDomains' to protect all subdomains"
	}

	// Check preload
	if !strings.Contains(value, "preload") {
		issues = append(issues, "Missing 'preload' directive (optional but recommended)")
		score -= 2
	}

	if len(issues) == 0 {
		recommendation = "Excellent HSTS configuration"
	}

	if score < 0 {
		score = 0
	}

	return score, issues, recommendation
}

// checkCSP validates the Content-Security-Policy header
func checkCSP(value string) (int, []string, string) {
	issues := []string{}
	score := 20
	recommendation := ""

	value = strings.ToLower(value)
	directives := parseCSPDirectives(value)

	// Check for unsafe practices
	if strings.Contains(value, "'unsafe-inline'") {
		issues = append(issues, "Contains 'unsafe-inline' which weakens CSP protection")
		score -= 5
	}

	if strings.Contains(value, "'unsafe-eval'") {
		issues = append(issues, "Contains 'unsafe-eval' which allows eval() and similar functions")
		score -= 5
	}

	if strings.Contains(value, "*") {
		issues = append(issues, "Contains wildcard (*) which is too permissive")
		score -= 3
	}

	// Check for essential directives
	if !strings.Contains(value, "default-src") {
		issues = append(issues, "Missing 'default-src' directive (recommended fallback)")
		score -= 3
	}

	if !strings.Contains(value, "script-src") {
		issues = append(issues, "Consider adding 'script-src' directive for script control")
		score -= 2
	}

	// Additional bypass heuristics for script/style directives
	scriptTokens := directives["script-src"]
	for _, token := range scriptTokens {
		switch token {
		case "data:":
			issues = append(issues, "Script sources allow data: URIs which can enable CSP bypasses")
			score -= 2
		case "blob:":
			issues = append(issues, "Script sources allow blob: URLs which may enable CSP bypasses")
			score -= 2
		case "filesystem:":
			issues = append(issues, "Script sources allow filesystem: URLs which may enable CSP bypasses")
			score -= 2
		}
		if strings.HasPrefix(token, "http:") {
			issues = append(issues, "Script sources allow insecure http scheme")
			score -= 2
		}
	}

	styleTokens := directives["style-src"]
	for _, token := range styleTokens {
		if token == "data:" {
			issues = append(issues, "Style sources allow data: URIs which may allow inline style injection")
			score -= 1
		}
		if token == "'unsafe-inline'" {
			issues = append(issues, "Style sources permit 'unsafe-inline' which weakens CSP")
			score -= 2
		}
	}

	if len(issues) == 0 {
		recommendation = "CSP is present with good configuration"
	} else {
		recommendation = "Review and strengthen your Content-Security-Policy"
	}

	return score, issues, recommendation
}

func parseCSPDirectives(value string) map[string][]string {
	result := make(map[string][]string)
	parts := strings.Split(value, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Fields(part)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		if len(fields) > 1 {
			result[name] = fields[1:]
		} else {
			result[name] = []string{}
		}
	}
	return result
}

// checkXFrameOptions validates the X-Frame-Options header
func checkXFrameOptions(value string) (int, []string, string) {
	issues := []string{}
	score := 15
	recommendation := ""

	value = strings.ToUpper(value)

	if value == "DENY" || value == "SAMEORIGIN" {
		recommendation = "X-Frame-Options is properly configured"
	} else if strings.HasPrefix(value, "ALLOW-FROM") {
		issues = append(issues, "ALLOW-FROM is deprecated and not supported by modern browsers")
		score = 5
		recommendation = "Use Content-Security-Policy frame-ancestors instead"
	} else {
		issues = append(issues, "Invalid X-Frame-Options value")
		score = 0
		recommendation = "Set to 'DENY' or 'SAMEORIGIN'"
	}

	return score, issues, recommendation
}

// checkXContentTypeOptions validates the X-Content-Type-Options header
func checkXContentTypeOptions(value string) (int, []string, string) {
	issues := []string{}
	score := 15
	recommendation := ""

	if strings.ToLower(value) == "nosniff" {
		recommendation = "X-Content-Type-Options is properly configured"
	} else {
		issues = append(issues, "Invalid value, should be 'nosniff'")
		score = 0
		recommendation = "Set to 'nosniff'"
	}

	return score, issues, recommendation
}

// checkReferrerPolicy validates the Referrer-Policy header
func checkReferrerPolicy(value string) (int, []string, string) {
	issues := []string{}
	score := 10
	recommendation := ""

	value = strings.ToLower(value)

	// Recommended policies
	goodPolicies := []string{
		"no-referrer",
		"strict-origin",
		"strict-origin-when-cross-origin",
		"same-origin",
	}

	isGood := false
	for _, policy := range goodPolicies {
		if strings.Contains(value, policy) {
			isGood = true
			break
		}
	}

	if !isGood {
		if strings.Contains(value, "unsafe-url") || strings.Contains(value, "origin-when-cross-origin") {
			issues = append(issues, "Policy may leak sensitive information in referrer")
			score = 5
			recommendation = "Use 'strict-origin-when-cross-origin' or 'no-referrer'"
		} else {
			issues = append(issues, "Unusual or weak referrer policy")
			score = 7
		}
	} else {
		recommendation = "Referrer-Policy is properly configured"
	}

	return score, issues, recommendation
}

// checkPermissionsPolicy validates the Permissions-Policy header
func checkPermissionsPolicy(value string) (int, []string, string) {
	issues := []string{}
	score := 10
	recommendation := "Permissions-Policy is present"

	// If header exists, give full score (complex to validate all directives)
	// Could be extended to check specific directives
	if len(value) < 10 {
		issues = append(issues, "Permissions-Policy seems minimal, consider adding more restrictions")
		score = 7
	}

	return score, issues, recommendation
}

// checkCOOP validates the Cross-Origin-Opener-Policy header
func checkCOOP(value string) (int, []string, string) {
	issues := []string{}
	score := 5
	recommendation := ""

	value = strings.ToLower(value)

	if value == "same-origin" || value == "same-origin-allow-popups" {
		recommendation = "Cross-Origin-Opener-Policy is properly configured"
	} else if value == "unsafe-none" {
		issues = append(issues, "COOP is set to 'unsafe-none' which provides no protection")
		score = 1
		recommendation = "Set to 'same-origin' for better isolation"
	} else {
		issues = append(issues, "Invalid COOP value")
		score = 0
		recommendation = "Set to 'same-origin'"
	}

	return score, issues, recommendation
}

// checkCOEP validates the Cross-Origin-Embedder-Policy header
func checkCOEP(value string) (int, []string, string) {
	issues := []string{}
	score := 5
	recommendation := ""

	value = strings.ToLower(value)

	if value == "require-corp" || value == "credentialless" {
		recommendation = "Cross-Origin-Embedder-Policy is properly configured"
	} else if value == "unsafe-none" {
		issues = append(issues, "COEP is set to 'unsafe-none' which provides no protection")
		score = 1
		recommendation = "Set to 'require-corp'"
	} else {
		issues = append(issues, "Invalid COEP value")
		score = 0
		recommendation = "Set to 'require-corp' or 'credentialless'"
	}

	return score, issues, recommendation
}

// checkDeprecatedHeaders checks for deprecated security headers
func checkDeprecatedHeaders(headers http.Header, result *SecurityHeadersResult) {
	// X-XSS-Protection
	if xss := headers.Get("X-XSS-Protection"); xss != "" {
		if xss != "0" {
			result.Warnings = append(result.Warnings,
				"X-XSS-Protection is deprecated and may introduce vulnerabilities. Set to '0' or remove it.")
		}
	}

	// Expect-CT
	if headers.Get("Expect-CT") != "" {
		result.Warnings = append(result.Warnings,
			"Expect-CT is deprecated. Remove this header.")
	}

	// Public-Key-Pins
	if headers.Get("Public-Key-Pins") != "" {
		result.Warnings = append(result.Warnings,
			"Public-Key-Pins (HPKP) is deprecated and dangerous. Remove this header immediately.")
	}
}

// checkInformationDisclosure checks for headers that expose sensitive information
func checkInformationDisclosure(headers http.Header, result *SecurityHeadersResult) {
	for _, headerName := range informationDisclosureHeaders {
		if value := headers.Get(headerName); value != "" {
			result.Warnings = append(result.Warnings,
				headerName+" header exposes server information: '"+value+"'. Consider removing or obfuscating.")
		}
	}
}

// checkContentType validates the Content-Type header
func checkContentType(value string) (int, []string, string) {
	issues := []string{}
	score := 5
	recommendation := ""

	value = strings.ToLower(value)

	// Content-Type should be present and specify charset for text content
	if value == "" {
		issues = append(issues, "Content-Type header is missing")
		score = 0
		recommendation = "Add Content-Type header with appropriate MIME type and charset"
		return score, issues, recommendation
	}

	// Check if it's a text-based content type without charset
	textTypes := []string{"text/html", "text/plain", "text/css", "text/javascript", "application/javascript", "application/json"}
	isTextType := false
	for _, textType := range textTypes {
		if strings.Contains(value, textType) {
			isTextType = true
			break
		}
	}

	if isTextType && !strings.Contains(value, "charset") {
		issues = append(issues, "Text content should specify charset (e.g., charset=utf-8)")
		score = 2
		recommendation = "Add charset parameter to Content-Type (e.g., 'text/html; charset=utf-8')"
	} else if isTextType && strings.Contains(value, "charset=utf-8") {
		recommendation = "Content-Type is properly configured with UTF-8 charset"
	} else {
		recommendation = "Content-Type header is present"
	}

	return score, issues, recommendation
}

// calculateGrade converts a score to a letter grade
func calculateGrade(score, maxScore int) string {
	percentage := float64(score) / float64(maxScore) * 100

	switch {
	case percentage >= 90:
		return "A"
	case percentage >= 80:
		return "B"
	case percentage >= 70:
		return "C"
	case percentage >= 60:
		return "D"
	case percentage >= 50:
		return "E"
	default:
		return "F"
	}
}
