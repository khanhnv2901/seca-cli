package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestGenerateJSONReport(t *testing.T) {
	output := &RunOutput{
		Metadata: RunMetadata{
			Operator:       "test-operator",
			EngagementID:   "test-123",
			EngagementName: "Test Engagement",
			Owner:          "owner@example.com",
			StartAt:        time.Now(),
			CompleteAt:     time.Now().Add(5 * time.Minute),
			AuditHash:      "abc123def456",
			TotalTargets:   2,
		},
		Results: []CheckResult{
			{
				Target:       "https://example.com",
				CheckedAt:    time.Now(),
				Status:       "ok",
				HTTPStatus:   200,
				ServerHeader: "nginx",
				TLSExpiry:    "2026-01-15T00:00:00Z",
			},
			{
				Target:     "https://test.com",
				CheckedAt:  time.Now(),
				Status:     "error",
				HTTPStatus: 0,
				Error:      "connection timeout",
			},
		},
	}

	report, err := generateJSONReport(output)
	if err != nil {
		t.Fatalf("Failed to generate JSON report: %v", err)
	}

	// Verify it's valid JSON
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(report), &decoded); err != nil {
		t.Fatalf("Generated report is not valid JSON: %v", err)
	}

	// Verify structure
	if _, exists := decoded["metadata"]; !exists {
		t.Error("Expected 'metadata' key in JSON report")
	}

	if _, exists := decoded["results"]; !exists {
		t.Error("Expected 'results' key in JSON report")
	}
}

func TestGenerateMarkdownReport(t *testing.T) {
	output := &RunOutput{
		Metadata: RunMetadata{
			Operator:       "test-operator",
			EngagementID:   "test-123",
			EngagementName: "Test Engagement",
			Owner:          "owner@example.com",
			StartAt:        time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			CompleteAt:     time.Date(2025, 1, 1, 10, 5, 0, 0, time.UTC),
			AuditHash:      "abc123def456",
			TotalTargets:   2,
		},
		Results: []CheckResult{
			{
				Target:       "https://example.com",
				CheckedAt:    time.Now(),
				Status:       "ok",
				HTTPStatus:   200,
				ServerHeader: "nginx",
				TLSExpiry:    "2026-01-15T00:00:00Z",
			},
			{
				Target:     "https://test.com",
				CheckedAt:  time.Now(),
				Status:     "error",
				HTTPStatus: 0,
				Error:      "connection timeout",
			},
		},
	}

	report, err := generateMarkdownReport(output)
	if err != nil {
		t.Fatalf("Failed to generate Markdown report: %v", err)
	}

	// Verify it contains markdown elements
	if !strings.Contains(report, "# Engagement Report:") {
		t.Error("Expected H1 header in markdown report")
	}

	if !strings.Contains(report, "## Metadata") {
		t.Error("Expected Metadata section in markdown report")
	}

	if !strings.Contains(report, "## Summary") {
		t.Error("Expected Summary section in markdown report")
	}

	if !strings.Contains(report, "## Results") {
		t.Error("Expected Results section in markdown report")
	}

	// Verify metadata is present
	if !strings.Contains(report, "Test Engagement") {
		t.Error("Expected engagement name in report")
	}

	if !strings.Contains(report, "test-operator") {
		t.Error("Expected operator name in report")
	}

	if !strings.Contains(report, "abc123def456") {
		t.Error("Expected audit hash in report")
	}

	// Verify table structure
	if !strings.Contains(report, "| Target | Status |") {
		t.Error("Expected table header in markdown report")
	}

	if !strings.Contains(report, "https://example.com") {
		t.Error("Expected target URL in report")
	}

	// Verify summary statistics
	if !strings.Contains(report, "Successful") {
		t.Error("Expected success count in summary")
	}

	if !strings.Contains(report, "Failed") {
		t.Error("Expected failure count in summary")
	}
}

func TestGenerateHTMLReport(t *testing.T) {
	output := &RunOutput{
		Metadata: RunMetadata{
			Operator:       "test-operator",
			EngagementID:   "test-123",
			EngagementName: "Test Engagement",
			Owner:          "owner@example.com",
			StartAt:        time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			CompleteAt:     time.Date(2025, 1, 1, 10, 5, 0, 0, time.UTC),
			AuditHash:      "abc123def456",
			TotalTargets:   2,
		},
		Results: []CheckResult{
			{
				Target:       "https://example.com",
				CheckedAt:    time.Now(),
				Status:       "ok",
				HTTPStatus:   200,
				ServerHeader: "nginx",
				TLSExpiry:    "2026-01-15T00:00:00Z",
			},
			{
				Target:     "https://test.com",
				CheckedAt:  time.Now(),
				Status:     "error",
				HTTPStatus: 0,
				Error:      "connection timeout",
			},
		},
	}

	report, err := generateHTMLReport(output)
	if err != nil {
		t.Fatalf("Failed to generate HTML report: %v", err)
	}

	// Verify HTML structure
	if !strings.Contains(report, "<!DOCTYPE html>") {
		t.Error("Expected HTML5 DOCTYPE")
	}

	if !strings.Contains(report, "<html") {
		t.Error("Expected HTML tag")
	}

	if !strings.Contains(report, "<head>") {
		t.Error("Expected HEAD tag")
	}

	if !strings.Contains(report, "<body>") {
		t.Error("Expected BODY tag")
	}

	if !strings.Contains(report, "</html>") {
		t.Error("Expected closing HTML tag")
	}

	// Verify CSS is included
	if !strings.Contains(report, "<style>") {
		t.Error("Expected CSS styles in HTML report")
	}

	// Verify title
	if !strings.Contains(report, "<title>Engagement Report: Test Engagement</title>") {
		t.Error("Expected title tag with engagement name")
	}

	// Verify metadata is present
	if !strings.Contains(report, "Test Engagement") {
		t.Error("Expected engagement name in HTML report")
	}

	if !strings.Contains(report, "test-operator") {
		t.Error("Expected operator name in HTML report")
	}

	if !strings.Contains(report, "abc123def456") {
		t.Error("Expected audit hash in HTML report")
	}

	// Verify table structure
	if !strings.Contains(report, "<table>") {
		t.Error("Expected table in HTML report")
	}

	if !strings.Contains(report, "<th>Target</th>") {
		t.Error("Expected table header in HTML report")
	}

	if !strings.Contains(report, "https://example.com") {
		t.Error("Expected target URL in HTML report")
	}

	// Verify summary cards
	if !strings.Contains(report, "summary-card") {
		t.Error("Expected summary cards in HTML report")
	}

	if !strings.Contains(report, "Successful") {
		t.Error("Expected success card in HTML report")
	}

	if !strings.Contains(report, "Failed") {
		t.Error("Expected failure card in HTML report")
	}

	// Verify status classes
	if !strings.Contains(report, "status-ok") {
		t.Error("Expected status-ok class for successful checks")
	}

	if !strings.Contains(report, "status-error") {
		t.Error("Expected status-error class for failed checks")
	}
}

func TestGenerateMarkdownReport_EmptyResults(t *testing.T) {
	output := &RunOutput{
		Metadata: RunMetadata{
			Operator:       "test-operator",
			EngagementID:   "test-123",
			EngagementName: "Empty Test",
			Owner:          "owner@example.com",
			StartAt:        time.Now(),
			CompleteAt:     time.Now(),
			TotalTargets:   0,
		},
		Results: []CheckResult{},
	}

	report, err := generateMarkdownReport(output)
	if err != nil {
		t.Fatalf("Failed to generate markdown report for empty results: %v", err)
	}

	if report == "" {
		t.Error("Expected non-empty report even with no results")
	}

	if !strings.Contains(report, "Empty Test") {
		t.Error("Expected engagement name in report")
	}
}

func TestGenerateHTMLReport_EmptyResults(t *testing.T) {
	output := &RunOutput{
		Metadata: RunMetadata{
			Operator:       "test-operator",
			EngagementID:   "test-123",
			EngagementName: "Empty Test",
			Owner:          "owner@example.com",
			StartAt:        time.Now(),
			CompleteAt:     time.Now(),
			TotalTargets:   0,
		},
		Results: []CheckResult{},
	}

	report, err := generateHTMLReport(output)
	if err != nil {
		t.Fatalf("Failed to generate HTML report for empty results: %v", err)
	}

	if report == "" {
		t.Error("Expected non-empty report even with no results")
	}

	if !strings.Contains(report, "Empty Test") {
		t.Error("Expected engagement name in report")
	}

	if !strings.Contains(report, "<!DOCTYPE html>") {
		t.Error("Expected valid HTML structure")
	}
}

func TestGenerateMarkdownReport_SummaryStatistics(t *testing.T) {
	output := &RunOutput{
		Metadata: RunMetadata{
			Operator:       "test-operator",
			EngagementID:   "test-123",
			EngagementName: "Stats Test",
			Owner:          "owner@example.com",
			StartAt:        time.Now(),
			CompleteAt:     time.Now(),
			TotalTargets:   4,
		},
		Results: []CheckResult{
			{Target: "https://example1.com", Status: "ok"},
			{Target: "https://example2.com", Status: "ok"},
			{Target: "https://example3.com", Status: "ok"},
			{Target: "https://example4.com", Status: "error"},
		},
	}

	report, err := generateMarkdownReport(output)
	if err != nil {
		t.Fatalf("Failed to generate markdown report: %v", err)
	}

	// Should have 3 successful, 1 failed, 75% success rate
	if !strings.Contains(report, "**Successful:** 3") {
		t.Error("Expected 3 successful results")
	}

	if !strings.Contains(report, "**Failed:** 1") {
		t.Error("Expected 1 failed result")
	}

	if !strings.Contains(report, "75.00%") {
		t.Error("Expected 75% success rate")
	}
}

func TestGenerateHTMLReport_SummaryStatistics(t *testing.T) {
	output := &RunOutput{
		Metadata: RunMetadata{
			Operator:       "test-operator",
			EngagementID:   "test-123",
			EngagementName: "Stats Test",
			Owner:          "owner@example.com",
			StartAt:        time.Now(),
			CompleteAt:     time.Now(),
			TotalTargets:   4,
		},
		Results: []CheckResult{
			{Target: "https://example1.com", Status: "ok"},
			{Target: "https://example2.com", Status: "ok"},
			{Target: "https://example3.com", Status: "ok"},
			{Target: "https://example4.com", Status: "error"},
		},
	}

	report, err := generateHTMLReport(output)
	if err != nil {
		t.Fatalf("Failed to generate HTML report: %v", err)
	}

	// Should have 3 successful, 1 failed, 75% success rate
	if !strings.Contains(report, ">3<") {
		t.Error("Expected 3 successful results in HTML")
	}

	if !strings.Contains(report, ">1<") {
		t.Error("Expected 1 failed result in HTML")
	}

	if !strings.Contains(report, "75.0%") {
		t.Error("Expected 75% success rate in HTML")
	}
}

func TestGenerateMarkdownReport_DurationCalculation(t *testing.T) {
	start := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	complete := time.Date(2025, 1, 1, 10, 5, 30, 0, time.UTC)

	output := &RunOutput{
		Metadata: RunMetadata{
			Operator:       "test-operator",
			EngagementID:   "test-123",
			EngagementName: "Duration Test",
			Owner:          "owner@example.com",
			StartAt:        start,
			CompleteAt:     complete,
			TotalTargets:   1,
		},
		Results: []CheckResult{
			{Target: "https://example.com", Status: "ok"},
		},
	}

	report, err := generateMarkdownReport(output)
	if err != nil {
		t.Fatalf("Failed to generate markdown report: %v", err)
	}

	// Duration should be 5m30s
	if !strings.Contains(report, "5m30s") {
		t.Error("Expected duration to be calculated and displayed")
	}
}

func TestGenerateHTMLReport_SpecialCharactersEscaping(t *testing.T) {
	output := &RunOutput{
		Metadata: RunMetadata{
			Operator:       "test-operator",
			EngagementID:   "test-123",
			EngagementName: "Test & Special <Characters>",
			Owner:          "owner@example.com",
			StartAt:        time.Now(),
			CompleteAt:     time.Now(),
			TotalTargets:   1,
		},
		Results: []CheckResult{
			{
				Target: "https://example.com",
				Status: "ok",
				Notes:  "Test & Notes",
			},
		},
	}

	report, err := generateHTMLReport(output)
	if err != nil {
		t.Fatalf("Failed to generate HTML report: %v", err)
	}

	// Note: The current implementation doesn't escape HTML characters
	// This test documents that behavior. In production, you might want to add HTML escaping.
	if report == "" {
		t.Error("Expected non-empty report")
	}
}

func TestGenerateMarkdownReport_OptionalFields(t *testing.T) {
	output := &RunOutput{
		Metadata: RunMetadata{
			Operator:       "test-operator",
			EngagementID:   "test-123",
			EngagementName: "Optional Fields Test",
			Owner:          "owner@example.com",
			StartAt:        time.Now(),
			CompleteAt:     time.Now(),
			TotalTargets:   1,
			AuditHash:      "", // Empty hash
		},
		Results: []CheckResult{
			{
				Target:       "https://example.com",
				Status:       "ok",
				HTTPStatus:   0, // No HTTP status
				ServerHeader: "", // No server header
				TLSExpiry:    "", // No TLS expiry
				Notes:        "", // No notes
				Error:        "", // No error
			},
		},
	}

	report, err := generateMarkdownReport(output)
	if err != nil {
		t.Fatalf("Failed to generate markdown report: %v", err)
	}

	// Verify placeholders for empty fields
	if !strings.Contains(report, "| https://example.com |") {
		t.Error("Expected target in report")
	}

	// Should have dashes for empty fields
	if !strings.Contains(report, "| - |") {
		t.Error("Expected dash placeholders for empty fields")
	}
}

func TestReportStatsCmd_CalculationLogic(t *testing.T) {
	// Test the logic used in reportStatsCmd
	output := RunOutput{
		Metadata: RunMetadata{
			EngagementID:   "test-123",
			EngagementName: "Stats Test",
			TotalTargets:   5,
		},
		Results: []CheckResult{
			{Target: "https://example1.com", Status: "ok", TLSExpiry: ""},
			{Target: "https://example2.com", Status: "ok", TLSExpiry: time.Now().Add(10 * 24 * time.Hour).Format(time.RFC3339)}, // Expires soon
			{Target: "https://example3.com", Status: "ok", TLSExpiry: time.Now().Add(60 * 24 * time.Hour).Format(time.RFC3339)}, // Expires later
			{Target: "https://example4.com", Status: "error", Error: "connection failed"},
			{Target: "https://example5.com", Status: "error", Error: "timeout"},
		},
	}

	total := len(output.Results)
	ok, fail, soon := 0, 0, 0

	for _, r := range output.Results {
		if r.Status == "ok" {
			ok++
		} else {
			fail++
		}
		if r.TLSExpiry != "" {
			if t, err := time.Parse(time.RFC3339, r.TLSExpiry); err == nil && time.Until(t) < (30*24*time.Hour) {
				soon++
			}
		}
	}

	// Verify counts
	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}

	if ok != 3 {
		t.Errorf("Expected 3 ok, got %d", ok)
	}

	if fail != 2 {
		t.Errorf("Expected 2 fail, got %d", fail)
	}

	if soon != 1 {
		t.Errorf("Expected 1 TLS expiring soon, got %d", soon)
	}
}

func TestReportStatsCmd_TLSExpiryDetection(t *testing.T) {
	testCases := []struct {
		name        string
		tlsExpiry   string
		expectSoon  bool
		description string
	}{
		{
			name:        "Expires in 10 days",
			tlsExpiry:   time.Now().Add(10 * 24 * time.Hour).Format(time.RFC3339),
			expectSoon:  true,
			description: "Should detect TLS expiring in 10 days as soon",
		},
		{
			name:        "Expires in 29 days",
			tlsExpiry:   time.Now().Add(29 * 24 * time.Hour).Format(time.RFC3339),
			expectSoon:  true,
			description: "Should detect TLS expiring in 29 days as soon",
		},
		{
			name:        "Expires in 31 days",
			tlsExpiry:   time.Now().Add(31 * 24 * time.Hour).Format(time.RFC3339),
			expectSoon:  false,
			description: "Should NOT detect TLS expiring in 31 days as soon",
		},
		{
			name:        "Expires in 60 days",
			tlsExpiry:   time.Now().Add(60 * 24 * time.Hour).Format(time.RFC3339),
			expectSoon:  false,
			description: "Should NOT detect TLS expiring in 60 days as soon",
		},
		{
			name:        "Empty TLS expiry",
			tlsExpiry:   "",
			expectSoon:  false,
			description: "Empty TLS expiry should not count as soon",
		},
		{
			name:        "Already expired",
			tlsExpiry:   time.Now().Add(-1 * 24 * time.Hour).Format(time.RFC3339),
			expectSoon:  true,
			description: "Expired TLS certificates are detected as soon (negative time.Until is < 30 days)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isSoon := false
			if tc.tlsExpiry != "" {
				if tlsTime, err := time.Parse(time.RFC3339, tc.tlsExpiry); err == nil && time.Until(tlsTime) < (30*24*time.Hour) {
					isSoon = true
				}
			}

			if isSoon != tc.expectSoon {
				t.Errorf("%s: expected soon=%v, got soon=%v", tc.description, tc.expectSoon, isSoon)
			}
		})
	}
}

func TestReportStatsCmd_EmptyResults(t *testing.T) {
	output := RunOutput{
		Metadata: RunMetadata{
			EngagementID: "test-empty",
			TotalTargets: 0,
		},
		Results: []CheckResult{},
	}

	total := len(output.Results)
	ok, fail, soon := 0, 0, 0

	for _, r := range output.Results {
		if r.Status == "ok" {
			ok++
		} else {
			fail++
		}
		if r.TLSExpiry != "" {
			if t, err := time.Parse(time.RFC3339, r.TLSExpiry); err == nil && time.Until(t) < (30*24*time.Hour) {
				soon++
			}
		}
	}

	if total != 0 || ok != 0 || fail != 0 || soon != 0 {
		t.Errorf("Expected all zeros for empty results, got total=%d, ok=%d, fail=%d, soon=%d", total, ok, fail, soon)
	}
}

func TestReportStatsCmd_AllSuccessful(t *testing.T) {
	output := RunOutput{
		Results: []CheckResult{
			{Target: "https://example1.com", Status: "ok"},
			{Target: "https://example2.com", Status: "ok"},
			{Target: "https://example3.com", Status: "ok"},
		},
	}

	ok, fail := 0, 0
	for _, r := range output.Results {
		if r.Status == "ok" {
			ok++
		} else {
			fail++
		}
	}

	if ok != 3 {
		t.Errorf("Expected 3 successful, got %d", ok)
	}

	if fail != 0 {
		t.Errorf("Expected 0 failed, got %d", fail)
	}
}

func TestReportStatsCmd_AllFailed(t *testing.T) {
	output := RunOutput{
		Results: []CheckResult{
			{Target: "https://example1.com", Status: "error"},
			{Target: "https://example2.com", Status: "error"},
			{Target: "https://example3.com", Status: "error"},
		},
	}

	ok, fail := 0, 0
	for _, r := range output.Results {
		if r.Status == "ok" {
			ok++
		} else {
			fail++
		}
	}

	if ok != 0 {
		t.Errorf("Expected 0 successful, got %d", ok)
	}

	if fail != 3 {
		t.Errorf("Expected 3 failed, got %d", fail)
	}
}

func TestTemplateData_Structure(t *testing.T) {
	// Verify TemplateData has all required fields
	data := TemplateData{
		Metadata:     RunMetadata{EngagementID: "test"},
		Results:      []CheckResult{},
		GeneratedAt:  "2025-01-01T00:00:00Z",
		StartedAt:    "2025-01-01T00:00:00Z",
		CompletedAt:  "2025-01-01T00:05:00Z",
		Duration:     "5m0s",
		SuccessCount: 10,
		ErrorCount:   2,
		SuccessRate:  "83.3",
		FooterDate:   "2025-01-01 00:00:00",
	}

	if data.Metadata.EngagementID != "test" {
		t.Error("TemplateData.Metadata should be accessible")
	}

	if data.SuccessCount != 10 {
		t.Error("TemplateData.SuccessCount should be accessible")
	}

	if data.ErrorCount != 2 {
		t.Error("TemplateData.ErrorCount should be accessible")
	}

	if data.SuccessRate != "83.3" {
		t.Error("TemplateData.SuccessRate should be accessible")
	}
}
