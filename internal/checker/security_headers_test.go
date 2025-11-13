package checker

import (
	"net/http"
	"testing"
)

func TestAnalyzeSecurityHeaders_AllPresent(t *testing.T) {
	headers := http.Header{}
	headers.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
	headers.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'")
	headers.Set("X-Frame-Options", "DENY")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	headers.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Cross-Origin-Embedder-Policy", "require-corp")
	headers.Set("Content-Type", "text/html; charset=utf-8")

	result := AnalyzeSecurityHeaders(headers)

	if result.Score < 90 {
		t.Errorf("Expected high score with all headers present, got %d", result.Score)
	}

	if result.Grade != "A" {
		t.Errorf("Expected grade A, got %s", result.Grade)
	}

	if len(result.Missing) != 0 {
		t.Errorf("Expected no missing headers, got %d: %v", len(result.Missing), result.Missing)
	}
}

func TestAnalyzeSecurityHeaders_AllMissing(t *testing.T) {
	headers := http.Header{}

	result := AnalyzeSecurityHeaders(headers)

	if result.Score != 0 {
		t.Errorf("Expected score 0 with no headers, got %d", result.Score)
	}

	if result.Grade != "F" {
		t.Errorf("Expected grade F, got %s", result.Grade)
	}

	if len(result.Missing) != 9 {
		t.Errorf("Expected 9 missing headers, got %d: %v", len(result.Missing), result.Missing)
	}
}

func TestCheckHSTS_Perfect(t *testing.T) {
	score, issues, _ := checkHSTS("max-age=31536000; includeSubDomains; preload")

	if score != 20 {
		t.Errorf("Expected score 20 for perfect HSTS, got %d", score)
	}

	if len(issues) != 0 {
		t.Errorf("Expected no issues for perfect HSTS, got %d: %v", len(issues), issues)
	}
}

func TestCheckHSTS_MissingIncludeSubDomains(t *testing.T) {
	score, issues, _ := checkHSTS("max-age=31536000; preload")

	if score >= 20 {
		t.Errorf("Expected reduced score without includeSubDomains, got %d", score)
	}

	if len(issues) == 0 {
		t.Error("Expected issues without includeSubDomains")
	}
}

func TestCheckHSTS_Disabled(t *testing.T) {
	score, issues, _ := checkHSTS("max-age=0")

	if score != 0 {
		t.Errorf("Expected score 0 for disabled HSTS, got %d", score)
	}

	if len(issues) == 0 {
		t.Error("Expected issues for disabled HSTS")
	}
}

func TestCheckCSP_WithUnsafeInline(t *testing.T) {
	score, issues, _ := checkCSP("default-src 'self'; script-src 'self' 'unsafe-inline'")

	if score >= 20 {
		t.Errorf("Expected reduced score with unsafe-inline, got %d", score)
	}

	hasUnsafeIssue := false
	for _, issue := range issues {
		if containsIgnoreCase(issue, "unsafe-inline") {
			hasUnsafeIssue = true
			break
		}
	}

	if !hasUnsafeIssue {
		t.Error("Expected issue about unsafe-inline")
	}
}

func TestCheckCSP_WithUnsafeEval(t *testing.T) {
	score, issues, _ := checkCSP("default-src 'self'; script-src 'self' 'unsafe-eval'")

	if score >= 20 {
		t.Errorf("Expected reduced score with unsafe-eval, got %d", score)
	}

	hasUnsafeEvalIssue := false
	for _, issue := range issues {
		if containsIgnoreCase(issue, "unsafe-eval") {
			hasUnsafeEvalIssue = true
			break
		}
	}

	if !hasUnsafeEvalIssue {
		t.Error("Expected issue about unsafe-eval")
	}
}

func TestCheckCSP_WithWildcard(t *testing.T) {
	score, issues, _ := checkCSP("default-src *")

	if score >= 20 {
		t.Errorf("Expected reduced score with wildcard, got %d", score)
	}

	if len(issues) == 0 {
		t.Error("Expected issues with wildcard in CSP")
	}
}

func TestCheckXFrameOptions_Deny(t *testing.T) {
	score, issues, _ := checkXFrameOptions("DENY")

	if score != 15 {
		t.Errorf("Expected score 15 for DENY, got %d", score)
	}

	if len(issues) != 0 {
		t.Errorf("Expected no issues for DENY, got %v", issues)
	}
}

func TestCheckXFrameOptions_SameOrigin(t *testing.T) {
	score, issues, _ := checkXFrameOptions("SAMEORIGIN")

	if score != 15 {
		t.Errorf("Expected score 15 for SAMEORIGIN, got %d", score)
	}

	if len(issues) != 0 {
		t.Errorf("Expected no issues for SAMEORIGIN, got %v", issues)
	}
}

func TestCheckXFrameOptions_AllowFrom_Deprecated(t *testing.T) {
	score, issues, _ := checkXFrameOptions("ALLOW-FROM https://example.com")

	if score >= 15 {
		t.Errorf("Expected reduced score for deprecated ALLOW-FROM, got %d", score)
	}

	if len(issues) == 0 {
		t.Error("Expected issues for deprecated ALLOW-FROM")
	}
}

func TestCheckXContentTypeOptions_NoSniff(t *testing.T) {
	score, issues, _ := checkXContentTypeOptions("nosniff")

	if score != 15 {
		t.Errorf("Expected score 15 for nosniff, got %d", score)
	}

	if len(issues) != 0 {
		t.Errorf("Expected no issues for nosniff, got %v", issues)
	}
}

func TestCheckXContentTypeOptions_Invalid(t *testing.T) {
	score, issues, _ := checkXContentTypeOptions("invalid-value")

	if score != 0 {
		t.Errorf("Expected score 0 for invalid value, got %d", score)
	}

	if len(issues) == 0 {
		t.Error("Expected issues for invalid value")
	}
}

func TestCheckReferrerPolicy_StrictOrigin(t *testing.T) {
	score, issues, _ := checkReferrerPolicy("strict-origin-when-cross-origin")

	if score != 10 {
		t.Errorf("Expected score 10 for good policy, got %d", score)
	}

	if len(issues) != 0 {
		t.Errorf("Expected no issues for good policy, got %v", issues)
	}
}

func TestCheckReferrerPolicy_UnsafeURL(t *testing.T) {
	score, issues, _ := checkReferrerPolicy("unsafe-url")

	if score >= 10 {
		t.Errorf("Expected reduced score for unsafe policy, got %d", score)
	}

	if len(issues) == 0 {
		t.Error("Expected issues for unsafe policy")
	}
}

func TestCheckCOOP_SameOrigin(t *testing.T) {
	score, issues, _ := checkCOOP("same-origin")

	if score != 5 {
		t.Errorf("Expected score 5 for same-origin, got %d", score)
	}

	if len(issues) != 0 {
		t.Errorf("Expected no issues for same-origin, got %v", issues)
	}
}

func TestCheckCOOP_UnsafeNone(t *testing.T) {
	score, issues, _ := checkCOOP("unsafe-none")

	if score >= 5 {
		t.Errorf("Expected reduced score for unsafe-none, got %d", score)
	}

	if len(issues) == 0 {
		t.Error("Expected issues for unsafe-none")
	}
}

func TestCheckCOEP_RequireCorp(t *testing.T) {
	score, issues, _ := checkCOEP("require-corp")

	if score != 5 {
		t.Errorf("Expected score 5 for require-corp, got %d", score)
	}

	if len(issues) != 0 {
		t.Errorf("Expected no issues for require-corp, got %v", issues)
	}
}

func TestCheckDeprecatedHeaders_XXSS(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-XSS-Protection", "1; mode=block")

	result := &SecurityHeadersResult{
		Headers:         make(map[string]HeaderStatus),
		Missing:         []string{},
		Warnings:        []string{},
		Recommendations: []string{},
	}

	checkDeprecatedHeaders(headers, result)

	if len(result.Warnings) == 0 {
		t.Error("Expected warning for deprecated X-XSS-Protection")
	}
}

func TestCheckDeprecatedHeaders_ExpectCT(t *testing.T) {
	headers := http.Header{}
	headers.Set("Expect-CT", "max-age=86400")

	result := &SecurityHeadersResult{
		Headers:         make(map[string]HeaderStatus),
		Missing:         []string{},
		Warnings:        []string{},
		Recommendations: []string{},
	}

	checkDeprecatedHeaders(headers, result)

	if len(result.Warnings) == 0 {
		t.Error("Expected warning for deprecated Expect-CT")
	}
}

func TestCheckInformationDisclosure_Server(t *testing.T) {
	headers := http.Header{}
	headers.Set("Server", "Apache/2.4.41 (Ubuntu)")

	result := &SecurityHeadersResult{
		Headers:         make(map[string]HeaderStatus),
		Missing:         []string{},
		Warnings:        []string{},
		Recommendations: []string{},
	}

	checkInformationDisclosure(headers, result)

	if len(result.Warnings) == 0 {
		t.Error("Expected warning for Server header information disclosure")
	}
}

func TestCheckInformationDisclosure_XPoweredBy(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Powered-By", "PHP/7.4.3")

	result := &SecurityHeadersResult{
		Headers:         make(map[string]HeaderStatus),
		Missing:         []string{},
		Warnings:        []string{},
		Recommendations: []string{},
	}

	checkInformationDisclosure(headers, result)

	if len(result.Warnings) == 0 {
		t.Error("Expected warning for X-Powered-By information disclosure")
	}
}

func TestCalculateGrade_A(t *testing.T) {
	grade := calculateGrade(95, 100)
	if grade != "A" {
		t.Errorf("Expected grade A for 95/100, got %s", grade)
	}
}

func TestCalculateGrade_B(t *testing.T) {
	grade := calculateGrade(85, 100)
	if grade != "B" {
		t.Errorf("Expected grade B for 85/100, got %s", grade)
	}
}

func TestCalculateGrade_C(t *testing.T) {
	grade := calculateGrade(75, 100)
	if grade != "C" {
		t.Errorf("Expected grade C for 75/100, got %s", grade)
	}
}

func TestCalculateGrade_D(t *testing.T) {
	grade := calculateGrade(65, 100)
	if grade != "D" {
		t.Errorf("Expected grade D for 65/100, got %s", grade)
	}
}

func TestCalculateGrade_E(t *testing.T) {
	grade := calculateGrade(55, 100)
	if grade != "E" {
		t.Errorf("Expected grade E for 55/100, got %s", grade)
	}
}

func TestCalculateGrade_F(t *testing.T) {
	grade := calculateGrade(45, 100)
	if grade != "F" {
		t.Errorf("Expected grade F for 45/100, got %s", grade)
	}
}

func TestAnalyzeSecurityHeaders_PartialImplementation(t *testing.T) {
	headers := http.Header{}
	headers.Set("Strict-Transport-Security", "max-age=31536000")
	headers.Set("X-Content-Type-Options", "nosniff")

	result := AnalyzeSecurityHeaders(headers)

	if result.Score == 0 {
		t.Error("Expected some score for partial implementation")
	}

	if result.Score == result.MaxScore {
		t.Error("Expected reduced score for partial implementation")
	}

	if len(result.Missing) == 0 {
		t.Error("Expected some missing headers")
	}

	if len(result.Headers) == 0 {
		t.Error("Expected some headers to be analyzed")
	}
}

func TestSecurityHeadersResult_JSONSerialization(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Frame-Options", "DENY")

	result := AnalyzeSecurityHeaders(headers)

	if result.Headers == nil {
		t.Error("Expected Headers map to be initialized")
	}

	if result.Missing == nil {
		t.Error("Expected Missing slice to be initialized")
	}

	if result.Warnings == nil {
		t.Error("Expected Warnings slice to be initialized")
	}

	if result.Grade == "" {
		t.Error("Expected Grade to be set")
	}
}

func TestAnalyzeSecurityHeaders_CaseInsensitive(t *testing.T) {
	headers := http.Header{}
	headers.Set("x-frame-options", "deny")
	headers.Set("X-CONTENT-TYPE-OPTIONS", "NOSNIFF")

	result := AnalyzeSecurityHeaders(headers)

	// Check if headers were detected despite case variations
	if status, ok := result.Headers["X-Frame-Options"]; !ok || !status.Present {
		t.Error("Expected X-Frame-Options to be detected (case-insensitive)")
	}

	if status, ok := result.Headers["X-Content-Type-Options"]; !ok || !status.Present {
		t.Error("Expected X-Content-Type-Options to be detected (case-insensitive)")
	}
}

func TestHeaderStatus_ScoreRange(t *testing.T) {
	headers := http.Header{}
	headers.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
	headers.Set("Content-Security-Policy", "default-src 'self'")
	headers.Set("X-Frame-Options", "DENY")

	result := AnalyzeSecurityHeaders(headers)

	for headerName, status := range result.Headers {
		if status.Present && status.Score > status.MaxScore {
			t.Errorf("Header %s: Score (%d) exceeds MaxScore (%d)", headerName, status.Score, status.MaxScore)
		}

		if status.Present && status.Score < 0 {
			t.Errorf("Header %s: Score is negative (%d)", headerName, status.Score)
		}
	}
}

// Helper function for case-insensitive string contains check
func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return contains(s, substr)
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexStr(s, substr) >= 0)
}

func indexStr(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	for i := 0; i <= len(s)-n; i++ {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}

func TestCheckContentType_Perfect(t *testing.T) {
	score, issues, rec := checkContentType("text/html; charset=utf-8")
	if score != 5 {
		t.Errorf("Expected score 5, got %d", score)
	}
	if len(issues) != 0 {
		t.Errorf("Expected no issues, got %v", issues)
	}
	if rec != "Content-Type is properly configured with UTF-8 charset" {
		t.Errorf("Unexpected recommendation: %s", rec)
	}
}

func TestCheckContentType_MissingCharset(t *testing.T) {
	score, issues, rec := checkContentType("text/html")
	if score != 2 {
		t.Errorf("Expected score 2, got %d", score)
	}
	if len(issues) == 0 {
		t.Error("Expected issues about missing charset")
	}
	if rec != "Add charset parameter to Content-Type (e.g., 'text/html; charset=utf-8')" {
		t.Errorf("Unexpected recommendation: %s", rec)
	}
}

func TestCheckContentType_Missing(t *testing.T) {
	score, issues, rec := checkContentType("")
	if score != 0 {
		t.Errorf("Expected score 0, got %d", score)
	}
	if len(issues) == 0 {
		t.Error("Expected issues about missing header")
	}
	if rec != "Add Content-Type header with appropriate MIME type and charset" {
		t.Errorf("Unexpected recommendation: %s", rec)
	}
}

func TestCheckContentType_NonTextType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{"image", "image/png"},
		{"video", "video/mp4"},
		{"binary", "application/octet-stream"},
		{"pdf", "application/pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, issues, rec := checkContentType(tt.contentType)
			if score != 5 {
				t.Errorf("Expected score 5 for non-text type, got %d", score)
			}
			if len(issues) != 0 {
				t.Errorf("Expected no issues for non-text type, got %v", issues)
			}
			if rec != "Content-Type header is present" {
				t.Errorf("Unexpected recommendation: %s", rec)
			}
		})
	}
}

func TestCheckContentType_TextTypes(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		wantScore   int
		wantIssues  bool
	}{
		{"html with charset", "text/html; charset=utf-8", 5, false},
		{"html no charset", "text/html", 2, true},
		{"json with charset", "application/json; charset=utf-8", 5, false},
		{"json no charset", "application/json", 2, true},
		{"javascript with charset", "text/javascript; charset=utf-8", 5, false},
		{"javascript no charset", "application/javascript", 2, true},
		{"css with charset", "text/css; charset=utf-8", 5, false},
		{"css no charset", "text/css", 2, true},
		{"plain with charset", "text/plain; charset=utf-8", 5, false},
		{"plain no charset", "text/plain", 2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, issues, _ := checkContentType(tt.contentType)
			if score != tt.wantScore {
				t.Errorf("Expected score %d, got %d", tt.wantScore, score)
			}
			if tt.wantIssues && len(issues) == 0 {
				t.Error("Expected issues but got none")
			}
			if !tt.wantIssues && len(issues) != 0 {
				t.Errorf("Expected no issues but got: %v", issues)
			}
		})
	}
}
