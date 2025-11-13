package checker

import (
	"net/http"
	"strings"
	"testing"
)

// ===== Tests for DetectVulnerableLibraries =====

func TestDetectVulnerableLibraries_jQuery(t *testing.T) {
	htmlContent := `
		<script src="https://code.jquery.com/jquery-3.4.1.min.js"></script>
	`

	vulns := DetectVulnerableLibraries(htmlContent)

	if len(vulns) != 1 {
		t.Fatalf("Expected 1 vulnerability, got %d", len(vulns))
	}

	if vulns[0].Name != "jQuery" {
		t.Errorf("Expected jQuery, got %s", vulns[0].Name)
	}

	if vulns[0].DetectedVersion != "3.4.1" {
		t.Errorf("Expected version 3.4.1, got %s", vulns[0].DetectedVersion)
	}

	if vulns[0].Severity != "high" {
		t.Errorf("Expected high severity, got %s", vulns[0].Severity)
	}
}

func TestDetectVulnerableLibraries_jQuery_Safe(t *testing.T) {
	htmlContent := `
		<script src="https://code.jquery.com/jquery-3.5.0.min.js"></script>
	`

	vulns := DetectVulnerableLibraries(htmlContent)

	if len(vulns) != 0 {
		t.Errorf("Expected no vulnerabilities for safe jQuery version, got %d", len(vulns))
	}
}

func TestDetectVulnerableLibraries_AngularJS(t *testing.T) {
	htmlContent := `
		<script src="https://ajax.googleapis.com/ajax/libs/angularjs/1.7.8/angular.min.js"></script>
	`

	vulns := DetectVulnerableLibraries(htmlContent)

	if len(vulns) != 1 {
		t.Fatalf("Expected 1 vulnerability, got %d", len(vulns))
	}

	if vulns[0].Name != "AngularJS" {
		t.Errorf("Expected AngularJS, got %s", vulns[0].Name)
	}

	if vulns[0].Severity != "critical" {
		t.Errorf("Expected critical severity, got %s", vulns[0].Severity)
	}
}

func TestDetectVulnerableLibraries_Lodash(t *testing.T) {
	htmlContent := `
		<script src="https://cdn.jsdelivr.net/npm/lodash@4.17.11/lodash.min.js"></script>
	`

	vulns := DetectVulnerableLibraries(htmlContent)

	if len(vulns) != 1 {
		t.Fatalf("Expected 1 vulnerability, got %d", len(vulns))
	}

	if vulns[0].Name != "Lodash" {
		t.Errorf("Expected Lodash, got %s", vulns[0].Name)
	}

	if vulns[0].CVSS != 9.1 {
		t.Errorf("Expected CVSS 9.1, got %f", vulns[0].CVSS)
	}
}

func TestDetectVulnerableLibraries_MomentJS(t *testing.T) {
	htmlContent := `
		<script src="https://cdnjs.cloudflare.com/ajax/libs/moment.js/2.29.1/moment.min.js"></script>
	`

	vulns := DetectVulnerableLibraries(htmlContent)

	if len(vulns) != 1 {
		t.Fatalf("Expected 1 vulnerability, got %d", len(vulns))
	}

	if vulns[0].Name != "Moment.js" {
		t.Errorf("Expected Moment.js, got %s", vulns[0].Name)
	}
}

func TestDetectVulnerableLibraries_Bootstrap(t *testing.T) {
	htmlContent := `
		<script src="https://stackpath.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js"></script>
	`

	vulns := DetectVulnerableLibraries(htmlContent)

	if len(vulns) != 1 {
		t.Fatalf("Expected 1 vulnerability, got %d", len(vulns))
	}

	if vulns[0].Name != "Bootstrap" {
		t.Errorf("Expected Bootstrap, got %s", vulns[0].Name)
	}

	if vulns[0].Severity != "medium" {
		t.Errorf("Expected medium severity, got %s", vulns[0].Severity)
	}
}

func TestDetectVulnerableLibraries_MultipleLibraries(t *testing.T) {
	htmlContent := `
		<script src="https://code.jquery.com/jquery-3.4.1.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/lodash.js/4.17.10/lodash.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/moment.js/2.29.1/moment.min.js"></script>
	`

	vulns := DetectVulnerableLibraries(htmlContent)

	if len(vulns) != 3 {
		t.Errorf("Expected 3 vulnerabilities, got %d", len(vulns))
	}
}

func TestDetectVulnerableLibraries_NoLibraries(t *testing.T) {
	htmlContent := `
		<html><body>No JavaScript libraries here</body></html>
	`

	vulns := DetectVulnerableLibraries(htmlContent)

	if len(vulns) != 0 {
		t.Errorf("Expected no vulnerabilities, got %d", len(vulns))
	}
}

// ===== Tests for compareVersion =====

func TestCompareVersion_Equal(t *testing.T) {
	result := compareVersion("1.2.3", "1.2.3")
	if result != 0 {
		t.Errorf("Expected 0 for equal versions, got %d", result)
	}
}

func TestCompareVersion_Less(t *testing.T) {
	result := compareVersion("1.2.3", "1.2.4")
	if result != -1 {
		t.Errorf("Expected -1, got %d", result)
	}

	result = compareVersion("1.2.3", "1.3.0")
	if result != -1 {
		t.Errorf("Expected -1, got %d", result)
	}

	result = compareVersion("1.2.3", "2.0.0")
	if result != -1 {
		t.Errorf("Expected -1, got %d", result)
	}
}

func TestCompareVersion_Greater(t *testing.T) {
	result := compareVersion("1.2.4", "1.2.3")
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}

	result = compareVersion("2.0.0", "1.9.9")
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
}

func TestCompareVersion_DifferentLengths(t *testing.T) {
	result := compareVersion("1.2", "1.2.0")
	if result != 0 {
		t.Errorf("Expected 0 for equal versions with different lengths, got %d", result)
	}

	result = compareVersion("1.2", "1.2.1")
	if result != -1 {
		t.Errorf("Expected -1, got %d", result)
	}
}

// ===== Tests for CheckCSRFProtection =====

func TestCheckCSRFProtection_MetaTag(t *testing.T) {
	htmlContent := `
		<html>
		<head>
			<meta name="csrf-token" content="abc123xyz">
		</head>
		</html>
	`

	check := CheckCSRFProtection(htmlContent, http.Header{}, []*http.Cookie{})

	if !check.HasCSRFToken {
		t.Error("Expected CSRF token to be detected")
	}

	if len(check.TokenLocations) == 0 || check.TokenLocations[0] != "meta" {
		t.Error("Expected token location to be 'meta'")
	}

	if check.Protection != "moderate" {
		t.Errorf("Expected moderate protection, got %s", check.Protection)
	}
}

func TestCheckCSRFProtection_FormInput(t *testing.T) {
	htmlContent := `
		<form method="post">
			<input type="hidden" name="csrf" value="token123">
		</form>
	`

	check := CheckCSRFProtection(htmlContent, http.Header{}, []*http.Cookie{})

	if !check.HasCSRFToken {
		t.Error("Expected CSRF token to be detected")
	}

	if len(check.TokenLocations) == 0 || check.TokenLocations[0] != "form" {
		t.Error("Expected token location to be 'form'")
	}
}

func TestCheckCSRFProtection_Header(t *testing.T) {
	htmlContent := `<html></html>`
	headers := http.Header{}
	headers.Set("X-CSRF-Token", "token123")

	check := CheckCSRFProtection(htmlContent, headers, []*http.Cookie{})

	if !check.HasCSRFToken {
		t.Error("Expected CSRF token to be detected in headers")
	}

	if len(check.TokenLocations) == 0 || check.TokenLocations[0] != "header" {
		t.Error("Expected token location to be 'header'")
	}
}

func TestCheckCSRFProtection_DoubleCookie(t *testing.T) {
	htmlContent := `<html></html>`
	cookies := []*http.Cookie{
		{Name: "XSRF-TOKEN", Value: "token123"},
	}

	check := CheckCSRFProtection(htmlContent, http.Header{}, cookies)

	if !check.HasCSRFToken {
		t.Error("Expected CSRF token to be detected in cookies")
	}

	if !check.DoubleCookieUsed {
		t.Error("Expected double cookie pattern to be detected")
	}

	if len(check.TokenLocations) == 0 || check.TokenLocations[0] != "cookie" {
		t.Error("Expected token location to be 'cookie'")
	}
}

func TestCheckCSRFProtection_SameSiteCookies(t *testing.T) {
	htmlContent := `<html></html>`
	cookies := []*http.Cookie{
		{Name: "session", Value: "abc123", SameSite: http.SameSiteStrictMode},
	}

	check := CheckCSRFProtection(htmlContent, http.Header{}, cookies)

	if !check.SameSiteCookies {
		t.Error("Expected SameSite cookies to be detected")
	}
}

func TestCheckCSRFProtection_StrongProtection(t *testing.T) {
	htmlContent := `
		<html>
		<head>
			<meta name="csrf-token" content="abc123">
		</head>
		</html>
	`
	cookies := []*http.Cookie{
		{Name: "session", Value: "abc123", SameSite: http.SameSiteStrictMode},
	}

	check := CheckCSRFProtection(htmlContent, http.Header{}, cookies)

	if check.Protection != "strong" {
		t.Errorf("Expected strong protection, got %s", check.Protection)
	}
}

func TestCheckCSRFProtection_NoProtection(t *testing.T) {
	htmlContent := `<html></html>`

	check := CheckCSRFProtection(htmlContent, http.Header{}, []*http.Cookie{})

	if check.HasCSRFToken {
		t.Error("Expected no CSRF token")
	}

	if check.Protection != "none" {
		t.Errorf("Expected no protection, got %s", check.Protection)
	}

	if len(check.Issues) == 0 {
		t.Error("Expected issues to be reported")
	}
}

// ===== Tests for CheckTrustedTypes =====

func TestCheckTrustedTypes_Present(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Security-Policy", "require-trusted-types-for 'script'; trusted-types default")

	result := CheckTrustedTypes(headers)

	if !result {
		t.Error("Expected Trusted Types to be detected")
	}
}

func TestCheckTrustedTypes_ReportOnly(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Security-Policy-Report-Only", "require-trusted-types-for 'script'")

	result := CheckTrustedTypes(headers)

	if !result {
		t.Error("Expected Trusted Types to be detected in report-only mode")
	}
}

func TestCheckTrustedTypes_NotPresent(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Security-Policy", "default-src 'self'")

	result := CheckTrustedTypes(headers)

	if result {
		t.Error("Expected Trusted Types to not be detected")
	}
}

func TestCheckTrustedTypes_NoCSP(t *testing.T) {
	headers := http.Header{}

	result := CheckTrustedTypes(headers)

	if result {
		t.Error("Expected Trusted Types to not be detected without CSP")
	}
}

// ===== Tests for AnalyzeClientSecurity =====

func TestAnalyzeClientSecurity_VulnerableLibrary(t *testing.T) {
	htmlContent := `
		<script src="https://code.jquery.com/jquery-3.4.1.min.js"></script>
	`

	result := AnalyzeClientSecurity(htmlContent, http.Header{}, []*http.Cookie{})

	if len(result.VulnerableLibraries) != 1 {
		t.Errorf("Expected 1 vulnerable library, got %d", len(result.VulnerableLibraries))
	}

	if len(result.Issues) == 0 {
		t.Error("Expected issues to be reported")
	}

	if !strings.Contains(result.Issues[0], "jQuery") {
		t.Error("Expected issue to mention jQuery")
	}
}

func TestAnalyzeClientSecurity_NoCSRF(t *testing.T) {
	htmlContent := `<html><body>Test</body></html>`

	result := AnalyzeClientSecurity(htmlContent, http.Header{}, []*http.Cookie{})

	if result.CSRFProtection == nil {
		t.Fatal("Expected CSRF check to be performed")
	}

	if result.CSRFProtection.HasCSRFToken {
		t.Error("Expected no CSRF token")
	}

	hasCSRFIssue := false
	for _, issue := range result.Issues {
		if strings.Contains(issue, "CSRF") {
			hasCSRFIssue = true
			break
		}
	}

	if !hasCSRFIssue {
		t.Error("Expected CSRF-related issue")
	}
}

func TestAnalyzeClientSecurity_WithCSRF(t *testing.T) {
	htmlContent := `
		<html>
		<head>
			<meta name="csrf-token" content="abc123">
		</head>
		</html>
	`
	cookies := []*http.Cookie{
		{Name: "session", Value: "abc", SameSite: http.SameSiteStrictMode},
	}

	result := AnalyzeClientSecurity(htmlContent, http.Header{}, cookies)

	if result.CSRFProtection == nil {
		t.Fatal("Expected CSRF check to be performed")
	}

	if !result.CSRFProtection.HasCSRFToken {
		t.Error("Expected CSRF token to be detected")
	}

	if result.CSRFProtection.Protection != "strong" {
		t.Errorf("Expected strong protection, got %s", result.CSRFProtection.Protection)
	}
}

func TestAnalyzeClientSecurity_NoTrustedTypes(t *testing.T) {
	htmlContent := `<html></html>`

	result := AnalyzeClientSecurity(htmlContent, http.Header{}, []*http.Cookie{})

	if result.TrustedTypes {
		t.Error("Expected Trusted Types to not be detected")
	}

	hasTrustedTypesRec := false
	for _, rec := range result.Recommendations {
		if strings.Contains(rec, "Trusted Types") {
			hasTrustedTypesRec = true
			break
		}
	}

	if !hasTrustedTypesRec {
		t.Error("Expected recommendation for Trusted Types")
	}
}

func TestAnalyzeClientSecurity_WithTrustedTypes(t *testing.T) {
	htmlContent := `<html></html>`
	headers := http.Header{}
	headers.Set("Content-Security-Policy", "require-trusted-types-for 'script'")

	result := AnalyzeClientSecurity(htmlContent, headers, []*http.Cookie{})

	if !result.TrustedTypes {
		t.Error("Expected Trusted Types to be detected")
	}
}

func TestAnalyzeClientSecurity_MultipleIssues(t *testing.T) {
	htmlContent := `
		<script src="https://code.jquery.com/jquery-3.4.1.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/lodash.js/4.17.10/lodash.min.js"></script>
	`

	result := AnalyzeClientSecurity(htmlContent, http.Header{}, []*http.Cookie{})

	if len(result.VulnerableLibraries) != 2 {
		t.Errorf("Expected 2 vulnerable libraries, got %d", len(result.VulnerableLibraries))
	}

	if len(result.Issues) < 2 {
		t.Errorf("Expected at least 2 issues, got %d", len(result.Issues))
	}

	if len(result.Recommendations) < 2 {
		t.Errorf("Expected at least 2 recommendations, got %d", len(result.Recommendations))
	}
}

func TestAnalyzeClientSecurity_WeakCSRF(t *testing.T) {
	htmlContent := `
		<html>
		<head>
			<meta name="csrf-token" content="abc123">
		</head>
		</html>
	`

	result := AnalyzeClientSecurity(htmlContent, http.Header{}, []*http.Cookie{})

	if result.CSRFProtection.Protection != "moderate" {
		t.Errorf("Expected moderate protection, got %s", result.CSRFProtection.Protection)
	}

	hasImprovement := false
	for _, issue := range result.Issues {
		if strings.Contains(issue, "could be improved") {
			hasImprovement = true
			break
		}
	}

	if !hasImprovement {
		t.Error("Expected suggestion to improve CSRF protection")
	}
}
