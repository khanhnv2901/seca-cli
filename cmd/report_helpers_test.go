package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/khanhnv2901/seca-cli/internal/infrastructure/checker"
)

func TestHeadersPresentHelpers(t *testing.T) {
	sh := &checker.SecurityHeadersResult{
		Headers: map[string]checker.HeaderStatus{
			"Strict-Transport-Security": {Present: true},
			"Content-Security-Policy":   {Present: false, Severity: "high"},
			"X-Frame-Options":           {Present: false, Severity: "medium"},
		},
	}

	if got := headersPresentCount(sh); got != 1 {
		t.Fatalf("expected 1 present header, got %d", got)
	}

	high := missingHighSeverityHeaders(sh)
	if len(high) != 1 || high[0] != "Content-Security-Policy" {
		t.Fatalf("unexpected high severity missing headers: %v", high)
	}

	medium := missingMediumSeverityHeaders(sh)
	if len(medium) != 1 || medium[0] != "X-Frame-Options" {
		t.Fatalf("unexpected medium severity missing headers: %v", medium)
	}

	if !hasCriticalMissingHeaders(sh) {
		t.Fatal("expected critical missing headers to be detected")
	}

	if missing := missingHeadersBySeverity(nil, "high"); missing != nil {
		t.Fatalf("expected nil when result is nil, got %v", missing)
	}
}

func TestFormatHelpers(t *testing.T) {
	ts := time.Date(2024, 2, 3, 15, 30, 0, 0, time.UTC)
	if got := formatShortTimestamp(time.Time{}); got != "" {
		t.Fatalf("expected empty string for zero timestamp, got %q", got)
	}
	if got := formatShortTimestamp(ts); got != "Feb 03 15:30" {
		t.Fatalf("unexpected formatted timestamp: %s", got)
	}

	if got := formatDurationLabel(-1); got != "0s" {
		t.Fatalf("negative durations should clamp to 0s, got %s", got)
	}
	if got := formatDurationLabel(45); got != "45.0s" {
		t.Fatalf("unexpected short duration formatting: %s", got)
	}
	if got := formatDurationLabel(125); got != "2.1 min" {
		t.Fatalf("unexpected minute formatting: %s", got)
	}

	if got := formatSuccessRate(87.654); got != "87.7%" {
		t.Fatalf("unexpected success rate format: %s", got)
	}

	tests := map[string]string{
		"critical": "badge-critical",
		"High":     "badge-high",
		"medium":   "badge-medium",
		"LOW":      "badge-low",
		"unknown":  "badge-info",
	}
	for input, want := range tests {
		if got := riskBadgeClass(input); got != want {
			t.Fatalf("riskBadgeClass(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSummarizeTrendHistory(t *testing.T) {
	if summary := summarizeTrendHistory(nil); summary.AverageSuccess != 0 || summary.AverageDuration != 0 {
		t.Fatalf("expected empty summary for no records, got %+v", summary)
	}

	records := []TelemetryRecord{
		{SuccessRate: 50, DurationSeconds: 10},
		{SuccessRate: 100, DurationSeconds: 20},
	}
	summary := summarizeTrendHistory(records)
	if summary.AverageSuccess != 75 {
		t.Fatalf("expected average success 75, got %.2f", summary.AverageSuccess)
	}
	if summary.AverageDuration != 15 {
		t.Fatalf("expected average duration 15, got %.2f", summary.AverageDuration)
	}
}

func TestSummarizeReportStats(t *testing.T) {
	soon := time.Now().Add(12 * time.Hour).Format(time.RFC3339)
	later := time.Now().Add(45 * 24 * time.Hour).Format(time.RFC3339)

	output := &RunOutput{
		Metadata: RunMetadata{EngagementID: "eng-1"},
		Results: []checker.CheckResult{
			{Target: "https://ok.example.com", Status: "ok", HTTPStatus: 200, TLSExpiry: soon, Notes: "all good"},
			{Target: "https://bad.example.com", Status: "error", HTTPStatus: 500, TLSExpiry: later},
		},
	}

	summary := summarizeReportStats(output)
	if summary.EngagementID != "eng-1" || summary.Total != 2 || summary.Success != 1 || summary.Fail != 1 {
		t.Fatalf("unexpected summary counts: %+v", summary)
	}
	if summary.TLSSoon != 1 {
		t.Fatalf("expected one TLS soon warning, got %d", summary.TLSSoon)
	}
	if len(summary.Results) != 2 || !summary.Results[0].TLSSoon {
		t.Fatalf("expected first result to have TLS warning, got %+v", summary.Results)
	}
}

func TestPrintStatsText(t *testing.T) {
	original := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = original })

	summary := reportStatsSummary{Total: 3, Success: 2, Fail: 1, TLSSoon: 1}
	output := captureStdout(t, func() {
		printStatsText(summary)
	})

	if !strings.Contains(output, "Summary") || !strings.Contains(output, "Targets: 3") {
		t.Fatalf("expected summary output, got %q", output)
	}
	if !strings.Contains(output, "OK: 2") || !strings.Contains(output, "Fail: 1") {
		t.Fatalf("expected counts in summary output, got %q", output)
	}
}

func TestPrintStatsTable(t *testing.T) {
	original := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = original })

	summary := reportStatsSummary{
		Results: []reportStatsEntry{
			{Target: "https://ok.example.com", Status: "ok", HTTPStatus: 200, Notes: "", TLSSoon: false},
			{Target: "https://bad.example.com", Status: "fail", HTTPStatus: 500, Notes: "needs work", TLSSoon: true},
		},
	}

	output := captureStdout(t, func() {
		printStatsTable(summary)
	})

	if !strings.Contains(output, "TARGET") || !strings.Contains(output, "STATUS") {
		t.Fatalf("expected table header, got %q", output)
	}
	if !strings.Contains(output, "https://ok.example.com") || !strings.Contains(output, "https://bad.example.com") {
		t.Fatalf("expected target rows, got %q", output)
	}
	if !strings.Contains(output, "needs work") {
		t.Fatalf("expected notes column, got %q", output)
	}
	if !strings.Contains(output, "yes") {
		t.Fatalf("expected TLS soon indicator, got %q", output)
	}
}

func TestPrintStatsTableEmpty(t *testing.T) {
	original := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = original })

	output := captureStdout(t, func() {
		printStatsTable(reportStatsSummary{})
	})

	if !strings.Contains(output, "No targets found") {
		t.Fatalf("expected empty summary message, got %q", output)
	}
}

func TestPrintTelemetryASCII(t *testing.T) {
	original := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = original })

	records := []TelemetryRecord{
		{Timestamp: time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC), Command: "check http", SuccessRate: 55.5, TargetCount: 10},
		{Timestamp: time.Date(2024, 5, 2, 12, 0, 0, 0, time.UTC), Command: "check dns", SuccessRate: 5, TargetCount: 2},
	}

	output := captureStdout(t, func() {
		printTelemetryASCII(records)
	})

	if !strings.Contains(output, "Telemetry Success Rate Trend") {
		t.Fatalf("expected telemetry header, got %q", output)
	}
	if !strings.Contains(output, "2024-05-01 12:00") || !strings.Contains(output, "check http") {
		t.Fatalf("expected first record details, got %q", output)
	}
	if !strings.Contains(output, "55.50%") || !strings.Contains(output, "5.00%") {
		t.Fatalf("expected success rates, got %q", output)
	}
}

func TestDeriveRunStatus(t *testing.T) {
	tests := []struct {
		name              string
		okCount, errCount int
		total             int
		want              string
	}{
		{"no targets", 0, 0, 0, "No targets"},
		{"success", 3, 0, 3, "Completed"},
		{"failure", 0, 2, 2, "Failed"},
		{"mixed", 2, 1, 3, "Completed with issues"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveRunStatus(tt.okCount, tt.errCount, tt.total); got != tt.want {
				t.Fatalf("deriveRunStatus(%d,%d,%d) = %q, want %q", tt.okCount, tt.errCount, tt.total, got, tt.want)
			}
		})
	}
}

func TestDeriveScanURL(t *testing.T) {
	output := &RunOutput{
		Metadata: RunMetadata{
			EngagementID:   "eng-1",
			EngagementName: "Example Engagement",
		},
		Results: []checker.CheckResult{
			{Target: "https://target.example.com"},
		},
	}

	if got := deriveScanURL(output); got != "https://target.example.com" {
		t.Fatalf("expected first target to be chosen, got %s", got)
	}

	output.Results = nil
	if got := deriveScanURL(output); got != "Example Engagement" {
		t.Fatalf("expected engagement name fallback, got %s", got)
	}

	output.Metadata.EngagementName = ""
	if got := deriveScanURL(output); got != "eng-1" {
		t.Fatalf("expected engagement ID fallback, got %s", got)
	}

	output.Metadata.EngagementID = ""
	if got := deriveScanURL(output); got != "N/A" {
		t.Fatalf("expected default fallback, got %s", got)
	}
}
