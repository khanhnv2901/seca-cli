package checker

import (
	"crypto/tls"
	"strings"
	"testing"
)

func TestCheckMixedContent_NoHTTPS(t *testing.T) {
	htmlContent := `<html><body><img src="http://example.com/image.jpg"></body></html>`
	pageURL := "http://example.com"

	result := CheckMixedContent(htmlContent, pageURL)

	if result != nil {
		t.Error("Expected nil for non-HTTPS page")
	}
}

func TestCheckMixedContent_NoMixedContent(t *testing.T) {
	htmlContent := `
		<html>
		<head>
			<link rel="stylesheet" href="https://example.com/styles.css">
			<script src="https://example.com/script.js"></script>
		</head>
		<body>
			<img src="https://example.com/image.jpg">
			<iframe src="https://example.com/frame"></iframe>
		</body>
		</html>
	`
	pageURL := "https://example.com"

	result := CheckMixedContent(htmlContent, pageURL)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.HasMixedContent {
		t.Error("Expected no mixed content")
	}

	if len(result.MixedContentURLs) > 0 {
		t.Errorf("Expected no mixed content URLs, got %d", len(result.MixedContentURLs))
	}
}

func TestCheckMixedContent_InsecureScripts(t *testing.T) {
	htmlContent := `
		<html>
		<head>
			<script src="http://evil.com/malicious.js"></script>
			<script src="https://safe.com/safe.js"></script>
		</head>
		</html>
	`
	pageURL := "https://example.com"

	result := CheckMixedContent(htmlContent, pageURL)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.HasMixedContent {
		t.Error("Expected mixed content to be detected")
	}

	if result.InsecureScripts != 1 {
		t.Errorf("Expected 1 insecure script, got %d", result.InsecureScripts)
	}

	if result.Severity != "critical" {
		t.Errorf("Expected critical severity, got %s", result.Severity)
	}

	if len(result.MixedContentURLs) != 1 {
		t.Errorf("Expected 1 mixed content URL, got %d", len(result.MixedContentURLs))
	}

	if result.MixedContentURLs[0] != "http://evil.com/malicious.js" {
		t.Errorf("Expected http://evil.com/malicious.js, got %s", result.MixedContentURLs[0])
	}
}

func TestCheckMixedContent_InsecureStylesheets(t *testing.T) {
	htmlContent := `
		<html>
		<head>
			<link rel="stylesheet" href="http://example.com/styles.css">
			<link href="http://example.com/other.css" rel="stylesheet">
		</head>
		</html>
	`
	pageURL := "https://example.com"

	result := CheckMixedContent(htmlContent, pageURL)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.HasMixedContent {
		t.Error("Expected mixed content to be detected")
	}

	if result.InsecureStyles != 2 {
		t.Errorf("Expected 2 insecure stylesheets, got %d", result.InsecureStyles)
	}

	if result.Severity != "high" {
		t.Errorf("Expected high severity, got %s", result.Severity)
	}
}

func TestCheckMixedContent_InsecureImages(t *testing.T) {
	htmlContent := `
		<html>
		<body>
			<img src="http://example.com/image1.jpg">
			<img src="http://example.com/image2.png">
			<img src="https://example.com/image3.jpg">
		</body>
		</html>
	`
	pageURL := "https://example.com"

	result := CheckMixedContent(htmlContent, pageURL)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.HasMixedContent {
		t.Error("Expected mixed content to be detected")
	}

	if result.InsecureImages != 2 {
		t.Errorf("Expected 2 insecure images, got %d", result.InsecureImages)
	}

	if result.Severity != "medium" {
		t.Errorf("Expected medium severity for images, got %s", result.Severity)
	}
}

func TestCheckMixedContent_InsecureMedia(t *testing.T) {
	htmlContent := `
		<html>
		<body>
			<video src="http://example.com/video.mp4"></video>
			<audio src="http://example.com/audio.mp3"></audio>
			<video>
				<source src="http://example.com/video.webm" type="video/webm">
			</video>
		</body>
		</html>
	`
	pageURL := "https://example.com"

	result := CheckMixedContent(htmlContent, pageURL)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.HasMixedContent {
		t.Error("Expected mixed content to be detected")
	}

	if result.InsecureMedia != 3 {
		t.Errorf("Expected 3 insecure media elements, got %d", result.InsecureMedia)
	}

	if result.Severity != "medium" {
		t.Errorf("Expected medium severity for media, got %s", result.Severity)
	}
}

func TestCheckMixedContent_InsecureIframes(t *testing.T) {
	htmlContent := `
		<html>
		<body>
			<iframe src="http://evil.com/frame"></iframe>
			<iframe src="https://safe.com/frame"></iframe>
		</body>
		</html>
	`
	pageURL := "https://example.com"

	result := CheckMixedContent(htmlContent, pageURL)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.HasMixedContent {
		t.Error("Expected mixed content to be detected")
	}

	if result.InsecureIframes != 1 {
		t.Errorf("Expected 1 insecure iframe, got %d", result.InsecureIframes)
	}

	if result.Severity != "critical" {
		t.Errorf("Expected critical severity for iframes, got %s", result.Severity)
	}
}

func TestCheckMixedContent_CSSImport(t *testing.T) {
	htmlContent := `
		<html>
		<head>
			<style>
				@import url("http://example.com/imported.css");
				@import url('http://example.com/imported2.css');
			</style>
		</head>
		</html>
	`
	pageURL := "https://example.com"

	result := CheckMixedContent(htmlContent, pageURL)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.HasMixedContent {
		t.Error("Expected mixed content to be detected")
	}

	if result.InsecureStyles != 2 {
		t.Errorf("Expected 2 insecure CSS imports, got %d", result.InsecureStyles)
	}
}

func TestCheckMixedContent_MixedTypes(t *testing.T) {
	htmlContent := `
		<html>
		<head>
			<script src="http://example.com/script.js"></script>
			<link rel="stylesheet" href="http://example.com/style.css">
		</head>
		<body>
			<img src="http://example.com/image.jpg">
			<iframe src="http://example.com/frame"></iframe>
		</body>
		</html>
	`
	pageURL := "https://example.com"

	result := CheckMixedContent(htmlContent, pageURL)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.HasMixedContent {
		t.Error("Expected mixed content to be detected")
	}

	// Should prioritize scripts/iframes as critical
	if result.Severity != "critical" {
		t.Errorf("Expected critical severity, got %s", result.Severity)
	}

	if result.InsecureScripts != 1 {
		t.Errorf("Expected 1 insecure script, got %d", result.InsecureScripts)
	}

	if result.InsecureStyles != 1 {
		t.Errorf("Expected 1 insecure stylesheet, got %d", result.InsecureStyles)
	}

	if result.InsecureImages != 1 {
		t.Errorf("Expected 1 insecure image, got %d", result.InsecureImages)
	}

	if result.InsecureIframes != 1 {
		t.Errorf("Expected 1 insecure iframe, got %d", result.InsecureIframes)
	}

	totalURLs := result.InsecureScripts + result.InsecureStyles + result.InsecureImages + result.InsecureIframes
	if len(result.MixedContentURLs) != totalURLs {
		t.Errorf("Expected %d mixed content URLs, got %d", totalURLs, len(result.MixedContentURLs))
	}
}

func TestCheckOCSPStapling_Nil(t *testing.T) {
	result := CheckOCSPStapling(nil)

	if result {
		t.Error("Expected false for nil connection state")
	}
}

func TestCheckOCSPStapling_NoOCSP(t *testing.T) {
	connState := &tls.ConnectionState{
		OCSPResponse: nil,
	}

	result := CheckOCSPStapling(connState)

	if result {
		t.Error("Expected false when OCSP response is nil")
	}
}

func TestCheckOCSPStapling_EmptyOCSP(t *testing.T) {
	connState := &tls.ConnectionState{
		OCSPResponse: []byte{},
	}

	result := CheckOCSPStapling(connState)

	if result {
		t.Error("Expected false when OCSP response is empty")
	}
}

func TestCheckOCSPStapling_WithOCSP(t *testing.T) {
	connState := &tls.ConnectionState{
		OCSPResponse: []byte{0x30, 0x03, 0x0a, 0x01, 0x00}, // Mock OCSP response
	}

	result := CheckOCSPStapling(connState)

	if !result {
		t.Error("Expected true when OCSP response is present")
	}
}

func TestAnalyzeMixedContentSummary_Nil(t *testing.T) {
	result := AnalyzeMixedContentSummary(nil)

	if result != "" {
		t.Errorf("Expected empty string for nil check, got %s", result)
	}
}

func TestAnalyzeMixedContentSummary_NoMixedContent(t *testing.T) {
	check := &MixedContentCheck{
		HasMixedContent: false,
	}

	result := AnalyzeMixedContentSummary(check)

	if result != "" {
		t.Errorf("Expected empty string for no mixed content, got %s", result)
	}
}

func TestAnalyzeMixedContentSummary_WithMixedContent(t *testing.T) {
	check := &MixedContentCheck{
		HasMixedContent:  true,
		InsecureScripts:  2,
		InsecureStyles:   1,
		InsecureImages:   3,
		InsecureIframes:  1,
	}

	result := AnalyzeMixedContentSummary(check)

	if result == "" {
		t.Error("Expected non-empty summary")
	}

	// Should mention the types found
	if !strings.Contains(result, "scripts") {
		t.Error("Expected summary to mention scripts")
	}

	if !strings.Contains(result, "stylesheets") {
		t.Error("Expected summary to mention stylesheets")
	}

	if !strings.Contains(result, "images") {
		t.Error("Expected summary to mention images")
	}

	if !strings.Contains(result, "iframes") {
		t.Error("Expected summary to mention iframes")
	}
}

func TestMixedContentCheck_Recommendations(t *testing.T) {
	tests := []struct {
		name           string
		htmlContent    string
		expectedSev    string
		checkRecommend bool
	}{
		{
			name:           "Critical - Scripts",
			htmlContent:    `<script src="http://evil.com/script.js"></script>`,
			expectedSev:    "critical",
			checkRecommend: true,
		},
		{
			name:           "Critical - Iframes",
			htmlContent:    `<iframe src="http://evil.com/frame"></iframe>`,
			expectedSev:    "critical",
			checkRecommend: true,
		},
		{
			name:           "High - Stylesheets",
			htmlContent:    `<link rel="stylesheet" href="http://example.com/style.css">`,
			expectedSev:    "high",
			checkRecommend: true,
		},
		{
			name:           "Medium - Images",
			htmlContent:    `<img src="http://example.com/image.jpg">`,
			expectedSev:    "medium",
			checkRecommend: true,
		},
		{
			name:           "Medium - Media",
			htmlContent:    `<video src="http://example.com/video.mp4"></video>`,
			expectedSev:    "medium",
			checkRecommend: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckMixedContent(tt.htmlContent, "https://example.com")

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			if result.Severity != tt.expectedSev {
				t.Errorf("Expected severity %s, got %s", tt.expectedSev, result.Severity)
			}

			if tt.checkRecommend && result.Recommendation == "" {
				t.Error("Expected recommendation to be set")
			}
		})
	}
}

func TestContainsURL(t *testing.T) {
	tests := []struct {
		slice    []string
		item     string
		expected bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{[]string{"test"}, "test", true},
	}

	for _, tt := range tests {
		result := containsURL(tt.slice, tt.item)
		if result != tt.expected {
			t.Errorf("containsURL(%v, %s) = %v, want %v", tt.slice, tt.item, result, tt.expected)
		}
	}
}
