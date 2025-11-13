package checker

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// AnalyzeClientSecurity performs comprehensive client-side security analysis
func AnalyzeClientSecurity(htmlContent string, headers http.Header, cookies []*http.Cookie) *ClientSecurityResult {
	result := &ClientSecurityResult{
		VulnerableLibraries: []VulnerableLibrary{},
		Issues:              []string{},
		Recommendations:     []string{},
	}

	// 1. Detect vulnerable JavaScript libraries
	vulnLibs := DetectVulnerableLibraries(htmlContent)
	if len(vulnLibs) > 0 {
		result.VulnerableLibraries = vulnLibs
		for _, lib := range vulnLibs {
			result.Issues = append(result.Issues,
				strings.ToUpper(lib.Severity)+": "+lib.Name+" "+lib.DetectedVersion+" - "+lib.Description)
		}
	}

	// 2. Check CSRF protection
	csrfCheck := CheckCSRFProtection(htmlContent, headers, cookies)
	result.CSRFProtection = csrfCheck
	if csrfCheck != nil {
		if !csrfCheck.HasCSRFToken {
			result.Issues = append(result.Issues, "No CSRF protection detected")
			result.Recommendations = append(result.Recommendations, csrfCheck.Recommendation)
		} else if csrfCheck.Protection == "weak" || csrfCheck.Protection == "moderate" {
			result.Issues = append(result.Issues, "CSRF protection could be improved")
			result.Recommendations = append(result.Recommendations, csrfCheck.Recommendation)
		}
	}

	// 3. Check Trusted Types support
	result.TrustedTypes = CheckTrustedTypes(headers)
	if !result.TrustedTypes {
		result.Recommendations = append(result.Recommendations,
			"Consider implementing Trusted Types to prevent DOM-based XSS attacks")
	}

	return result
}

// DetectVulnerableLibraries scans HTML for known vulnerable JavaScript libraries
func DetectVulnerableLibraries(htmlContent string) []VulnerableLibrary {
	vulnerabilities := []VulnerableLibrary{}

	// Define patterns for popular libraries with known vulnerabilities
	// This is a curated list of common libraries and their vulnerable versions
	libraryPatterns := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		// jQuery: matches jquery-3.4.1.min.js or jquery/3.4.1/jquery.min.js
		{"jQuery", regexp.MustCompile(`jquery[/-](\d+\.\d+\.?\d*)`)},
		// AngularJS: matches angularjs/1.7.8/angular.min.js
		{"AngularJS", regexp.MustCompile(`angularjs?[/@](\d+\.\d+\.?\d*)`)},
		// Lodash: matches lodash@4.17.11 or lodash.js/4.17.10
		{"Lodash", regexp.MustCompile(`lodash(?:\.js)?[@/](\d+\.\d+\.?\d*)`)},
		// Moment.js: matches moment.js/2.29.1
		{"Moment.js", regexp.MustCompile(`moment\.js[/@](\d+\.\d+\.?\d*)`)},
		// Bootstrap: matches bootstrap/3.3.7/js/bootstrap.min.js
		{"Bootstrap", regexp.MustCompile(`bootstrap[/@](\d+\.\d+\.?\d*)`)},
		// React: matches react-16.8.0.min.js or react/16.8.0
		{"React", regexp.MustCompile(`react[/@-](\d+\.\d+\.?\d*)`)},
		// Vue: matches vue-2.6.0.min.js or vue/2.6.0
		{"Vue", regexp.MustCompile(`vue[/@-](\d+\.\d+\.?\d*)`)},
	}

	for _, lib := range libraryPatterns {
		matches := lib.pattern.FindAllStringSubmatch(htmlContent, -1)
		for _, match := range matches {
			if len(match) > 1 && match[1] != "" {
				version := match[1]
				// Check if this version has known vulnerabilities
				if vuln := checkLibraryVulnerability(lib.name, version); vuln != nil {
					vulnerabilities = append(vulnerabilities, *vuln)
				}
			}
		}
	}

	return vulnerabilities
}

// checkLibraryVulnerability checks if a library version has known vulnerabilities
// This uses a curated database of common vulnerable versions
func checkLibraryVulnerability(library, version string) *VulnerableLibrary {
	// Common vulnerable versions database
	// In production, this should be updated regularly or use an external vulnerability DB

	switch library {
	case "jQuery":
		// jQuery < 3.5.0 has XSS vulnerabilities
		if compareVersion(version, "3.5.0") < 0 {
			return &VulnerableLibrary{
				Name:             "jQuery",
				DetectedVersion:  version,
				VulnerabilityIDs: []string{"CVE-2020-11022", "CVE-2020-11023"},
				Severity:         "high",
				Description:      "jQuery versions before 3.5.0 contain XSS vulnerabilities in htmlPrefilter",
				Recommendation:   "Update jQuery to version 3.5.0 or later",
				CVSS:             6.1,
			}
		}

	case "Angular", "AngularJS":
		// AngularJS < 1.7.9 has XSS vulnerabilities
		if compareVersion(version, "1.7.9") < 0 {
			return &VulnerableLibrary{
				Name:             "AngularJS",
				DetectedVersion:  version,
				VulnerabilityIDs: []string{"CVE-2019-10768"},
				Severity:         "critical",
				Description:      "AngularJS versions before 1.7.9 contain prototype pollution vulnerability",
				Recommendation:   "Update AngularJS to 1.7.9+ or migrate to Angular 2+",
				CVSS:             7.5,
			}
		}

	case "Lodash":
		// Lodash < 4.17.12 has prototype pollution
		if compareVersion(version, "4.17.12") < 0 {
			return &VulnerableLibrary{
				Name:             "Lodash",
				DetectedVersion:  version,
				VulnerabilityIDs: []string{"CVE-2019-10744"},
				Severity:         "critical",
				Description:      "Lodash versions before 4.17.12 contain prototype pollution vulnerability",
				Recommendation:   "Update Lodash to version 4.17.12 or later",
				CVSS:             9.1,
			}
		}

	case "Moment.js":
		// Moment.js < 2.29.2 has ReDoS vulnerability
		if compareVersion(version, "2.29.2") < 0 {
			return &VulnerableLibrary{
				Name:             "Moment.js",
				DetectedVersion:  version,
				VulnerabilityIDs: []string{"CVE-2022-24785"},
				Severity:         "high",
				Description:      "Moment.js versions before 2.29.2 contain ReDoS vulnerability",
				Recommendation:   "Update to 2.29.2+ or consider migrating to modern alternatives like date-fns or Luxon",
				CVSS:             7.5,
			}
		}

	case "Bootstrap":
		// Bootstrap < 3.4.0 has XSS vulnerabilities
		if compareVersion(version, "3.4.0") < 0 {
			return &VulnerableLibrary{
				Name:             "Bootstrap",
				DetectedVersion:  version,
				VulnerabilityIDs: []string{"CVE-2019-8331"},
				Severity:         "medium",
				Description:      "Bootstrap versions before 3.4.0 contain XSS vulnerability in tooltip/popover",
				Recommendation:   "Update Bootstrap to version 3.4.0 or later",
				CVSS:             6.1,
			}
		}
	}

	return nil
}

// compareVersion compares two semantic versions
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersion(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Pad with zeros
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for len(parts1) < maxLen {
		parts1 = append(parts1, "0")
	}
	for len(parts2) < maxLen {
		parts2 = append(parts2, "0")
	}

	// Compare each part
	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		// Simple integer parsing (ignoring errors for simplicity)
		if _, err := fmt.Sscanf(parts1[i], "%d", &n1); err != nil {
			n1 = 0
		}
		if _, err := fmt.Sscanf(parts2[i], "%d", &n2); err != nil {
			n2 = 0
		}

		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}

	return 0
}

// CheckCSRFProtection analyzes CSRF protection mechanisms
func CheckCSRFProtection(htmlContent string, headers http.Header, cookies []*http.Cookie) *CSRFCheck {
	check := &CSRFCheck{
		HasCSRFToken:     false,
		TokenLocations:   []string{},
		TokenNames:       []string{},
		DoubleCookieUsed: false,
		SameSiteCookies:  false,
		Protection:       "none",
		Issues:           []string{},
	}

	// 1. Check for CSRF tokens in meta tags
	metaTokenPattern := regexp.MustCompile(`<meta[^>]+name=['"](?:csrf-token|_csrf|xsrf-token)['"][^>]+content=['"]([^'"]+)['"]`)
	if matches := metaTokenPattern.FindAllStringSubmatch(htmlContent, -1); len(matches) > 0 {
		check.HasCSRFToken = true
		check.TokenLocations = append(check.TokenLocations, "meta")
		for range matches {
			check.TokenNames = append(check.TokenNames, "meta-tag")
		}
	}

	// 2. Check for CSRF tokens in forms (hidden inputs)
	formTokenPattern := regexp.MustCompile(`<input[^>]+type=['"]hidden['"][^>]+name=['"](?:csrf|_csrf|csrfmiddlewaretoken|authenticity_token|__RequestVerificationToken)['"]`)
	if matches := formTokenPattern.FindAllStringSubmatch(htmlContent, -1); len(matches) > 0 {
		check.HasCSRFToken = true
		check.TokenLocations = append(check.TokenLocations, "form")
		check.TokenNames = append(check.TokenNames, "form-input")
	}

	// 3. Check for CSRF token in custom headers
	csrfHeaders := []string{"X-CSRF-Token", "X-XSRF-Token", "X-CSRFToken"}
	for _, headerName := range csrfHeaders {
		if headers.Get(headerName) != "" {
			check.HasCSRFToken = true
			check.TokenLocations = append(check.TokenLocations, "header")
			check.TokenNames = append(check.TokenNames, headerName)
		}
	}

	// 4. Check for Double Submit Cookie pattern
	csrfCookieNames := []string{"XSRF-TOKEN", "csrf_token", "csrftoken"}
	for _, cookie := range cookies {
		for _, csrfName := range csrfCookieNames {
			if strings.EqualFold(cookie.Name, csrfName) {
				check.DoubleCookieUsed = true
				check.HasCSRFToken = true
				check.TokenLocations = append(check.TokenLocations, "cookie")
				check.TokenNames = append(check.TokenNames, cookie.Name)
			}
		}
	}

	// 5. Check for SameSite cookie attribute
	sameSiteCount := 0
	for _, cookie := range cookies {
		// Note: http.Cookie doesn't expose SameSite directly in all Go versions
		// We check the raw Set-Cookie header
		if cookie.SameSite != 0 {
			sameSiteCount++
		}
	}
	if sameSiteCount > 0 {
		check.SameSiteCookies = true
	}

	// Assess overall protection level
	if !check.HasCSRFToken && !check.SameSiteCookies {
		check.Protection = "none"
		check.Issues = append(check.Issues, "No CSRF protection mechanisms detected")
		check.Recommendation = "Implement CSRF tokens using synchronizer token pattern or double-submit cookie. " +
			"Add SameSite attribute to cookies. Consider using frameworks with built-in CSRF protection."
	} else if check.HasCSRFToken && check.SameSiteCookies {
		check.Protection = "strong"
		check.Recommendation = "CSRF protection is properly implemented with multiple layers"
	} else if check.HasCSRFToken {
		check.Protection = "moderate"
		check.Issues = append(check.Issues, "CSRF tokens present but SameSite cookies not used")
		check.Recommendation = "Add SameSite=Strict or SameSite=Lax attribute to session cookies for defense-in-depth"
	} else if check.SameSiteCookies {
		check.Protection = "moderate"
		check.Issues = append(check.Issues, "SameSite cookies used but no CSRF tokens")
		check.Recommendation = "Add CSRF tokens to forms for stronger protection, especially for older browsers"
	} else {
		check.Protection = "weak"
		check.Issues = append(check.Issues, "Incomplete CSRF protection")
		check.Recommendation = "Implement comprehensive CSRF protection with both tokens and SameSite cookies"
	}

	return check
}

// CheckTrustedTypes checks if the page implements Trusted Types for DOM XSS prevention
func CheckTrustedTypes(headers http.Header) bool {
	// Check Content-Security-Policy header for require-trusted-types-for directive
	csp := headers.Get("Content-Security-Policy")
	if csp == "" {
		csp = headers.Get("Content-Security-Policy-Report-Only")
	}

	if csp != "" {
		// Check for require-trusted-types-for 'script'
		if strings.Contains(csp, "require-trusted-types-for") &&
		   strings.Contains(csp, "'script'") {
			return true
		}
	}

	return false
}
