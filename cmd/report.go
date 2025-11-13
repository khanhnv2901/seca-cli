package cmd

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"math"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/khanhnv2901/seca-cli/internal/checker"
	consts "github.com/khanhnv2901/seca-cli/internal/constants"
	"github.com/spf13/cobra"
)

const (
	htmlTemplatePath     = "templates/report.html"
	markdownTemplatePath = "templates/report.md"
	statsTLSSoonWindow   = 30 * 24 * time.Hour
)

//go:embed templates/report.html templates/report.md
var reportTemplateFS embed.FS

var preferredResultFilenames = []string{
	"http_results.json",
	"network_results.json",
	"dns_results.json",
}

var securityHeaderNames = []string{
	"Strict-Transport-Security",
	"Content-Security-Policy",
	"X-Content-Type-Options",
	"X-Frame-Options",
	"Referrer-Policy",
	"Permissions-Policy",
	"Cross-Origin-Opener-Policy",
	"Cross-Origin-Embedder-Policy",
}

var (
	htmlTemplateFuncs = template.FuncMap{
		"add":                 addInts,
		"join":                strings.Join,
		"headersPresentCount": headersPresentCount,
		"formatTime":          formatShortTimestamp,
		"formatDuration":      formatDurationLabel,
		"formatSuccess":       formatSuccessRate,
		"lower":               strings.ToLower,
		"riskBadgeClass":      riskBadgeClass,
	}

	markdownTemplateFuncs = template.FuncMap{
		"add":                    addInts,
		"join":                   strings.Join,
		"headersPresentCount":    headersPresentCount,
		"highSeverityMissing":    missingHighSeverityHeaders,
		"mediumSeverityMissing":  missingMediumSeverityHeaders,
		"hasHighSeverityMissing": hasCriticalMissingHeaders,
		"formatTime":             formatShortTimestamp,
		"formatDuration":         formatDurationLabel,
		"formatSuccess":          formatSuccessRate,
	}

	htmlReportTemplate = template.Must(
		template.New("report.html").Funcs(htmlTemplateFuncs).ParseFS(reportTemplateFS, htmlTemplatePath),
	)
	markdownReportTemplate = template.Must(
		template.New("report.md").Funcs(markdownTemplateFuncs).ParseFS(reportTemplateFS, markdownTemplatePath),
	)
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a summary report (markdown or HTML)",
}

var reportGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate report for an engagement",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get application context
		appCtx := getAppContext(cmd)

		id, _ := cmd.Flags().GetString("id")
		format, _ := cmd.Flags().GetString("format")

		if id == "" {
			return fmt.Errorf("--id is required")
		}

		// Validate format
		format = strings.ToLower(format)
		if format != "json" && format != "md" && format != "html" && format != "pdf" {
			return fmt.Errorf("invalid format: %s (must be json, md, html, or pdf)", format)
		}

		output, sources, err := loadAggregatedRunOutput(appCtx.ResultsDir, id)
		if err != nil {
			return err
		}
		normalizeRunMetadata(&output.Metadata)

		// Generate report based on format
		var reportContent string
		var filename string

		trendHistory, histErr := loadTelemetryHistory(appCtx.ResultsDir, output.Metadata.EngagementID, 8)
		if histErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load telemetry history: %v\n", histErr)
		}

		switch format {
		case "json":
			reportContent, err = generateJSONReport(output)
			filename = "report.json"
		case "md":
			data := buildTemplateData(output, sources, "%.2f", trendHistory)
			reportContent, err = generateMarkdownReport(data)
			filename = "report.md"
		case "html":
			data := buildTemplateData(output, sources, "%.1f", trendHistory)
			reportContent, err = generateHTMLReport(data)
			filename = "report.html"
		case "pdf":
			data := buildTemplateData(output, sources, "%.1f", trendHistory)
			pdfBytes, perr := generatePDFReportBytes(data)
			if perr != nil {
				return fmt.Errorf("failed to generate PDF report: %w", perr)
			}
			filename = "report.pdf"
			reportPath, err := resolveResultsPath(appCtx.ResultsDir, id, filename)
			if err != nil {
				return fmt.Errorf("resolve report path: %w", err)
			}
			if err := os.WriteFile(reportPath, pdfBytes, consts.DefaultFilePerm); err != nil {
				return fmt.Errorf("failed to write report: %w", err)
			}
			fmt.Printf("Report generated: %s\n", reportPath)
			fmt.Printf("Format: %s\n", format)
			fmt.Printf("Total targets: %d\n", output.Metadata.TotalTargets)
			if len(sources) > 0 {
				fmt.Printf("Result files included: %s\n", strings.Join(sources, ", "))
			}
			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to generate report: %w", err)
		}

		// Write report to file
		reportPath, err := resolveResultsPath(appCtx.ResultsDir, id, filename)
		if err != nil {
			return fmt.Errorf("resolve report path: %w", err)
		}
		if err := os.WriteFile(reportPath, []byte(reportContent), consts.DefaultFilePerm); err != nil {
			return fmt.Errorf("failed to write report: %w", err)
		}

		fmt.Printf("Report generated: %s\n", reportPath)
		fmt.Printf("Format: %s\n", format)
		fmt.Printf("Total targets: %d\n", output.Metadata.TotalTargets)
		if len(sources) > 0 {
			fmt.Printf("Result files included: %s\n", strings.Join(sources, ", "))
		}

		return nil
	},
}

func generateJSONReport(output *RunOutput) (string, error) {
	data, err := json.MarshalIndent(output, jsonPrefix, jsonIndent)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func discoverResultFiles(resultsDir, engagementID string) ([]string, error) {
	dirPath, err := resolveResultsPath(resultsDir, engagementID)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	available := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.EqualFold(name, "http_results.json") || strings.HasSuffix(name, "_results.json") {
			available[name] = struct{}{}
		}
	}

	ordered := make([]string, 0, len(available))
	for _, pref := range preferredResultFilenames {
		if _, ok := available[pref]; ok {
			ordered = append(ordered, pref)
			delete(available, pref)
		}
	}
	if len(available) > 0 {
		extra := make([]string, 0, len(available))
		for name := range available {
			extra = append(extra, name)
		}
		sort.Strings(extra)
		ordered = append(ordered, extra...)
	}

	return ordered, nil
}

func loadAggregatedRunOutput(resultsDir, engagementID string) (*RunOutput, []string, error) {
	files, err := discoverResultFiles(resultsDir, engagementID)
	if err != nil {
		return nil, nil, fmt.Errorf("discover result files: %w", err)
	}
	if len(files) == 0 {
		return nil, nil, fmt.Errorf("no results found for engagement %s", engagementID)
	}

	var aggregated *RunOutput
	var earliestStart time.Time
	var latestComplete time.Time
	sourcesUsed := make([]string, 0, len(files))

	for _, name := range files {
		path, err := resolveResultsPath(resultsDir, engagementID, name)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve results path for %s: %w", name, err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, nil, fmt.Errorf("read %s: %w", path, err)
		}

		var current RunOutput
		if err := json.Unmarshal(data, &current); err != nil {
			return nil, nil, fmt.Errorf("parse %s: %w", name, err)
		}
		if len(current.Results) == 0 {
			continue
		}

		sourcesUsed = append(sourcesUsed, name)

		if aggregated == nil {
			aggregated = &RunOutput{
				Metadata: current.Metadata,
				Results:  append([]checker.CheckResult(nil), current.Results...),
			}
			earliestStart = current.Metadata.StartAt
			latestComplete = current.Metadata.CompleteAt
			continue
		}

		aggregated.Results = append(aggregated.Results, current.Results...)
		if isEarlier(current.Metadata.StartAt, earliestStart) {
			earliestStart = current.Metadata.StartAt
		}
		if current.Metadata.CompleteAt.After(latestComplete) {
			latestComplete = current.Metadata.CompleteAt
		}
	}

	if aggregated == nil {
		return nil, nil, fmt.Errorf("no results found for engagement %s", engagementID)
	}

	if !earliestStart.IsZero() && (aggregated.Metadata.StartAt.IsZero() || earliestStart.Before(aggregated.Metadata.StartAt)) {
		aggregated.Metadata.StartAt = earliestStart
	}
	if !latestComplete.IsZero() && latestComplete.After(aggregated.Metadata.CompleteAt) {
		aggregated.Metadata.CompleteAt = latestComplete
	}
	aggregated.Metadata.TotalTargets = len(aggregated.Results)

	return aggregated, sourcesUsed, nil
}

func isEarlier(candidate, reference time.Time) bool {
	if candidate.IsZero() {
		return false
	}
	if reference.IsZero() {
		return true
	}
	return candidate.Before(reference)
}

func normalizeRunMetadata(meta *RunMetadata) {
	if meta == nil {
		return
	}
	if meta.AuditHash == "" && meta.LegacyAuditHash != "" {
		meta.AuditHash = meta.LegacyAuditHash
	}
	if meta.HashAlgorithm == "" {
		meta.HashAlgorithm = HashAlgorithmSHA256.String()
	}
}

func generateMarkdownReport(data TemplateData) (string, error) {
	return executeTemplate(markdownReportTemplate, data)
}

// TemplateData holds the data for HTML/PDF/Markdown template rendering
type TemplateData struct {
	Metadata           RunMetadata
	Results            []checker.CheckResult
	ResultSources      []string
	CheckCatalog       []SecurityCheckSpec
	GeneratedAt        string
	StartedAt          string
	CompletedAt        string
	Duration           string
	SuccessCount       int
	ErrorCount         int
	SuccessRate        string
	FooterDate         string
	TrendHistory       []TelemetryRecord
	TrendSummary       TrendSummary
	HashAlgorithmLabel string

	// Fields used by the revamped HTML template
	ScanDate        string
	ScanURL         string
	Status          string
	Summary         checker.VulnerabilitySummary
	Vulnerabilities []checker.Vulnerability
}

type reportStatsEntry struct {
	Target     string `json:"target"`
	Status     string `json:"status"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Notes      string `json:"notes,omitempty"`
	TLSSoon    bool   `json:"tls_expires_soon"`
}

type reportStatsSummary struct {
	EngagementID string             `json:"engagement_id"`
	Total        int                `json:"total"`
	Success      int                `json:"success"`
	Fail         int                `json:"fail"`
	TLSSoon      int                `json:"tls_expiring"`
	Results      []reportStatsEntry `json:"results"`
}

type TrendSummary struct {
	AverageSuccess  float64
	AverageDuration float64
}

func generateHTMLReport(data TemplateData) (string, error) {
	return executeTemplate(htmlReportTemplate, data)
}

func addInts(a, b int) int {
	return a + b
}

func headersPresentCount(sh *checker.SecurityHeadersResult) int {
	if sh == nil || sh.Headers == nil {
		return 0
	}
	count := 0
	for _, header := range sh.Headers {
		if header.Present {
			count++
		}
	}
	return count
}

func missingHighSeverityHeaders(sh *checker.SecurityHeadersResult) []string {
	return missingHeadersBySeverity(sh, "high")
}

func missingMediumSeverityHeaders(sh *checker.SecurityHeadersResult) []string {
	return missingHeadersBySeverity(sh, "medium")
}

func missingHeadersBySeverity(sh *checker.SecurityHeadersResult, severity string) []string {
	if sh == nil || sh.Headers == nil {
		return nil
	}
	var missing []string
	for _, name := range securityHeaderNames {
		header, ok := sh.Headers[name]
		if !ok || header.Present {
			continue
		}
		if header.Severity == severity {
			missing = append(missing, name)
		}
	}
	return missing
}

func hasCriticalMissingHeaders(sh *checker.SecurityHeadersResult) bool {
	if sh == nil || sh.Headers == nil {
		return false
	}
	for _, name := range securityHeaderNames {
		header, ok := sh.Headers[name]
		if !ok || header.Present {
			continue
		}
		if header.Severity == "high" || header.Severity == "medium" {
			return true
		}
	}
	return false
}

func formatShortTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("Jan 02 15:04")
}

func formatDurationLabel(seconds float64) string {
	if seconds <= 0 {
		return "0s"
	}
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}
	min := seconds / 60
	return fmt.Sprintf("%.1f min", min)
}

func formatSuccessRate(rate float64) string {
	return fmt.Sprintf("%.1f%%", rate)
}

func riskBadgeClass(risk string) string {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "critical":
		return "badge-critical"
	case "high":
		return "badge-high"
	case "medium":
		return "badge-medium"
	case "low":
		return "badge-low"
	default:
		return "badge-info"
	}
}

func generatePDFReportBytes(data TemplateData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Title
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, fmt.Sprintf("Engagement Report: %s", data.Metadata.EngagementName), "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// Metadata section
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Engagement ID: %s", data.Metadata.EngagementID), "", 1, "", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Operator: %s", data.Metadata.Operator), "", 1, "", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Started: %s", data.StartedAt), "", 1, "", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Completed: %s", data.CompletedAt), "", 1, "", false, 0, "")
	if len(data.ResultSources) > 0 {
		pdf.CellFormat(0, 6, fmt.Sprintf("Result files: %s", strings.Join(data.ResultSources, ", ")), "", 1, "", false, 0, "")
	}
	pdf.Ln(5)

	// Summary section
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "Summary", "", 1, "", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Success: %d | Errors: %d | Success Rate: %s",
		data.SuccessCount, data.ErrorCount, data.SuccessRate), "", 1, "", false, 0, "")
	pdf.Ln(5)

	// Security check catalog
	if len(data.CheckCatalog) > 0 {
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(0, 8, "Security Check Catalog", "", 1, "", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		for _, check := range data.CheckCatalog {
			pdf.MultiCell(0, 5, fmt.Sprintf("• %s — %s", check.Name, check.Category), "", "", false)
		}
		pdf.Ln(3)
	}

	// Trend Analysis section (if available)
	if len(data.TrendHistory) > 0 {
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(0, 8, "Trend Analysis", "", 1, "", false, 0, "")
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(0, 6, fmt.Sprintf("Average Success: %.1f%%", data.TrendSummary.AverageSuccess), "", 1, "", false, 0, "")
		pdf.CellFormat(0, 6, fmt.Sprintf("Average Duration: %s", formatDurationLabel(data.TrendSummary.AverageDuration)), "", 1, "", false, 0, "")
		pdf.Ln(3)

		for _, rec := range data.TrendHistory {
			pdf.CellFormat(0, 6, fmt.Sprintf("  %s -> %s success, %s",
				formatShortTimestamp(rec.Timestamp),
				formatSuccessRate(rec.SuccessRate),
				formatDurationLabel(rec.DurationSeconds)), "", 1, "", false, 0, "")
		}
		pdf.Ln(5)
	}

	// Results section - Detailed Security Analysis
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "Detailed Security Analysis", "", 1, "", false, 0, "")
	pdf.Ln(2)

	maxResults := 50
	for i, r := range data.Results {
		if i == maxResults {
			pdf.SetFont("Arial", "I", 9)
			pdf.CellFormat(0, 6, fmt.Sprintf("... %d additional targets omitted ...", len(data.Results)-maxResults), "", 1, "", false, 0, "")
			break
		}

		// Check if we need a new page before adding content
		if pdf.GetY() > 250 {
			pdf.AddPage()
		}

		status := strings.ToUpper(r.Status)

		// Target header with status
		pdf.SetFont("Arial", "B", 11)
		pdf.SetFillColor(240, 240, 240)
		pdf.CellFormat(0, 7, fmt.Sprintf("%s - %s", r.Target, status), "", 1, "", true, 0, "")
		pdf.Ln(1)

		// Basic information
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(0, 5, fmt.Sprintf("Response Time: %.2f ms | Server: %s", r.ResponseTime, r.ServerHeader), "", 1, "", false, 0, "")

		// Security Headers Score
		if r.SecurityHeaders.MaxScore > 0 {
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(0, 5, fmt.Sprintf("Security Headers: %d/%d (Grade: %s)",
				r.SecurityHeaders.Score, r.SecurityHeaders.MaxScore, r.SecurityHeaders.Grade), "", 1, "", false, 0, "")

			// Missing headers
			if len(r.SecurityHeaders.Missing) > 0 {
				pdf.SetFont("Arial", "", 8)
				pdf.CellFormat(0, 4, fmt.Sprintf("  Missing: %s", strings.Join(r.SecurityHeaders.Missing, ", ")), "", 1, "", false, 0, "")
			}

			// Warnings
			if len(r.SecurityHeaders.Warnings) > 0 {
				for _, warning := range r.SecurityHeaders.Warnings {
					pdf.SetFont("Arial", "I", 8)
					pdf.MultiCell(0, 4, fmt.Sprintf("  Warning: %s", warning), "", "", false)
				}
			}
		}

		// TLS/SSL Information
		if r.TLSCompliance.TLSVersion != "" {
			pdf.SetFont("Arial", "B", 9)
			compliance := "Non-Compliant"
			if r.TLSCompliance.Compliant {
				compliance = "Compliant"
			}
			pdf.CellFormat(0, 5, fmt.Sprintf("TLS: %s | Cipher: %s | %s",
				r.TLSCompliance.TLSVersion, r.TLSCompliance.CipherSuite, compliance), "", 1, "", false, 0, "")

			// Certificate info
			if r.TLSCompliance.CertificateInfo.Subject != "" {
				pdf.SetFont("Arial", "", 8)
				pdf.CellFormat(0, 4, fmt.Sprintf("  Certificate: %s (Expires: %d days)",
					r.TLSCompliance.CertificateInfo.Subject,
					r.TLSCompliance.CertificateInfo.DaysUntilExpiry), "", 1, "", false, 0, "")
			}

			// TLS Recommendations
			if len(r.TLSCompliance.Recommendations) > 0 {
				pdf.SetFont("Arial", "I", 8)
				for _, rec := range r.TLSCompliance.Recommendations {
					if pdf.GetY() > 270 {
						pdf.AddPage()
					}
					pdf.MultiCell(0, 4, fmt.Sprintf("  - %s", rec), "", "", false)
				}
			}
		}

		// CORS Issues
		if r.CORSInsights != nil && len(r.CORSInsights.Issues) > 0 {
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(0, 5, "CORS Issues:", "", 1, "", false, 0, "")
			pdf.SetFont("Arial", "", 8)
			for _, issue := range r.CORSInsights.Issues {
				pdf.MultiCell(0, 4, fmt.Sprintf("  - %s", issue), "", "", false)
			}
		}

		// Cache Policy Issues
		if r.CachePolicy != nil && len(r.CachePolicy.Issues) > 0 {
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(0, 5, "Cache Policy Issues:", "", 1, "", false, 0, "")
			pdf.SetFont("Arial", "", 8)
			for _, issue := range r.CachePolicy.Issues {
				pdf.MultiCell(0, 4, fmt.Sprintf("  - %s", issue), "", "", false)
			}
		}

		// Notes
		notes := strings.TrimSpace(r.Notes)
		if notes != "" {
			pdf.SetFont("Arial", "I", 8)
			pdf.MultiCell(0, 4, fmt.Sprintf("Notes: %s", notes), "", "", false)
		}

		pdf.Ln(3) // Gap between targets
	}

	// Generate PDF bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

func buildTemplateData(output *RunOutput, sources []string, successRateFmt string, trends []TelemetryRecord) TemplateData {
	normalizeRunMetadata(&output.Metadata)
	okCount, errorCount := summarizeResults(output.Results)
	total := len(output.Results)
	successRate := 0.0
	if total > 0 {
		successRate = float64(okCount) / float64(total) * 100
	}

	now := time.Now()
	duration := output.Metadata.CompleteAt.Sub(output.Metadata.StartAt)
	if duration < 0 {
		duration = 0
	}

	scanURL := deriveScanURL(output)
	scanDate := ""
	if !output.Metadata.StartAt.IsZero() {
		scanDate = output.Metadata.StartAt.Format(time.RFC1123)
	}

	durationLabel := duration.Round(time.Second).String()
	vulnReport := checker.BuildVulnerabilityReport(
		output.Results,
		scanURL,
		scanDate,
		durationLabel,
	)

	return TemplateData{
		Metadata:           output.Metadata,
		Results:            output.Results,
		ResultSources:      append([]string(nil), sources...),
		CheckCatalog:       getSecurityCheckCatalog(),
		GeneratedAt:        now.Format(time.RFC3339),
		StartedAt:          output.Metadata.StartAt.Format(time.RFC3339),
		CompletedAt:        output.Metadata.CompleteAt.Format(time.RFC3339),
		Duration:           durationLabel,
		SuccessCount:       okCount,
		ErrorCount:         errorCount,
		SuccessRate:        fmt.Sprintf(successRateFmt, successRate),
		FooterDate:         now.Format("2006-01-02 15:04:05"),
		TrendHistory:       trends,
		TrendSummary:       summarizeTrendHistory(trends),
		HashAlgorithmLabel: strings.ToUpper(output.Metadata.HashAlgorithm),
		ScanDate:           scanDate,
		ScanURL:            scanURL,
		Status:             deriveRunStatus(okCount, errorCount, total),
		Summary:            vulnReport.Summary,
		Vulnerabilities:    vulnReport.Vulnerabilities,
	}
}

func summarizeResults(results []checker.CheckResult) (okCount, errorCount int) {
	for _, r := range results {
		if r.Status == "ok" {
			okCount++
		} else {
			errorCount++
		}
	}
	return okCount, errorCount
}

func summarizeTrendHistory(trends []TelemetryRecord) TrendSummary {
	if len(trends) == 0 {
		return TrendSummary{}
	}
	sumSuccess := 0.0
	sumDuration := 0.0
	for _, rec := range trends {
		sumSuccess += rec.SuccessRate
		sumDuration += rec.DurationSeconds
	}
	count := float64(len(trends))
	return TrendSummary{
		AverageSuccess:  sumSuccess / count,
		AverageDuration: sumDuration / count,
	}
}

func executeTemplate(tmpl *template.Template, data TemplateData) (string, error) {
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute %s template: %w", tmpl.Name(), err)
	}
	return buf.String(), nil
}

func deriveScanURL(output *RunOutput) string {
	for _, result := range output.Results {
		if strings.TrimSpace(result.Target) != "" {
			return result.Target
		}
	}
	if output.Metadata.EngagementName != "" {
		return output.Metadata.EngagementName
	}
	if output.Metadata.EngagementID != "" {
		return output.Metadata.EngagementID
	}
	return "N/A"
}

func deriveRunStatus(okCount, errorCount, total int) string {
	switch {
	case total == 0:
		return "No targets"
	case errorCount == 0:
		return "Completed"
	case okCount == 0:
		return "Failed"
	default:
		return "Completed with issues"
	}
}

func summarizeReportStats(output *RunOutput) reportStatsSummary {
	summary := reportStatsSummary{
		EngagementID: output.Metadata.EngagementID,
		Results:      make([]reportStatsEntry, 0, len(output.Results)),
	}

	for _, r := range output.Results {
		entry := reportStatsEntry{
			Target:     r.Target,
			Status:     r.Status,
			HTTPStatus: r.HTTPStatus,
			Notes:      r.Notes,
		}
		summary.Total++
		if strings.EqualFold(r.Status, "ok") {
			summary.Success++
		} else {
			summary.Fail++
		}
		if r.TLSExpiry != "" {
			if t, err := time.Parse(time.RFC3339, r.TLSExpiry); err == nil && time.Until(t) < statsTLSSoonWindow {
				entry.TLSSoon = true
				summary.TLSSoon++
			}
		}
		summary.Results = append(summary.Results, entry)
	}

	return summary
}

func printStatsText(summary reportStatsSummary) {
	fmt.Println(colorInfo("Summary"))
	fmt.Printf("Targets: %d | OK: %s | Fail: %s | TLS <30d: %s\n",
		summary.Total,
		colorSuccess(fmt.Sprintf("%d", summary.Success)),
		colorError(fmt.Sprintf("%d", summary.Fail)),
		colorWarn(fmt.Sprintf("%d", summary.TLSSoon)),
	)
}

func printStatsTable(summary reportStatsSummary) {
	if len(summary.Results) == 0 {
		fmt.Println(colorWarn("No targets found in results."))
		return
	}

	tw := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "TARGET\tSTATUS\tHTTP\tTLS<30d?\tNOTES")
	for _, entry := range summary.Results {
		status := formatStatusWithColor(entry.Status)
		tlsCol := "no"
		if entry.TLSSoon {
			tlsCol = colorWarn("yes")
		}
		notes := entry.Notes
		if notes == "" {
			notes = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\n", entry.Target, status, entry.HTTPStatus, tlsCol, notes)
	}
	if err := tw.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to flush stats table: %v\n", err)
	}
}

func printTelemetryASCII(records []TelemetryRecord) {
	const barWidth = 40
	fmt.Println(colorInfo("Telemetry Success Rate Trend"))
	for _, rec := range records {
		barLen := int(math.Round((rec.SuccessRate / 100.0) * barWidth))
		if barLen < 0 {
			barLen = 0
		}
		if barLen > barWidth {
			barLen = barWidth
		}
		if barLen == 0 && rec.SuccessRate > 0 {
			barLen = 1
		}
		bar := strings.Repeat("#", barLen)
		fmt.Printf("%s | %6.2f%% | %-*s | %s (%d targets)\n",
			rec.Timestamp.Format("2006-01-02 15:04"),
			rec.SuccessRate,
			barWidth,
			bar,
			rec.Command,
			rec.TargetCount,
		)
	}
}

var reportStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show analytics summary for engagement",
	RunE: func(cmd *cobra.Command, args []string) error {
		appCtx := getAppContext(cmd)

		id, _ := cmd.Flags().GetString("id")
		format, _ := cmd.Flags().GetString("format")
		format = strings.ToLower(strings.TrimSpace(format))
		if format == "" {
			format = "text"
		}

		path, err := resolveResultsPath(appCtx.ResultsDir, id, "http_results.json")
		if err != nil {
			return fmt.Errorf("resolve results path: %w", err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var output RunOutput
		if err := json.Unmarshal(data, &output); err != nil {
			return err
		}
		normalizeRunMetadata(&output.Metadata)

		summary := summarizeReportStats(&output)

		switch format {
		case "json":
			payload, err := json.MarshalIndent(summary, jsonPrefix, jsonIndent)
			if err != nil {
				return err
			}
			fmt.Println(string(payload))
		case "table":
			printStatsTable(summary)
		case "text":
			printStatsText(summary)
		default:
			return fmt.Errorf("unsupported format %q (use text|table|json)", format)
		}
		return nil
	},
}

var reportTelemetryCmd = &cobra.Command{
	Use:   "telemetry",
	Short: "Graph telemetry success rate trend for an engagement",
	RunE: func(cmd *cobra.Command, args []string) error {
		appCtx := getAppContext(cmd)

		id, _ := cmd.Flags().GetString("id")
		format, _ := cmd.Flags().GetString("format")
		limit, _ := cmd.Flags().GetInt("limit")

		if id == "" {
			return fmt.Errorf("--id is required")
		}

		history, err := loadTelemetryHistory(appCtx.ResultsDir, id, limit)
		if err != nil {
			return err
		}
		if len(history) == 0 {
			fmt.Printf("%s telemetry records found for engagement %s\n", colorWarn("No"), id)
			return nil
		}

		switch strings.ToLower(format) {
		case "json":
			out, err := json.MarshalIndent(history, jsonPrefix, jsonIndent)
			if err != nil {
				return fmt.Errorf("marshal telemetry: %w", err)
			}
			fmt.Println(string(out))
		case "ascii":
			printTelemetryASCII(history)
		default:
			return fmt.Errorf("unsupported format %s (use ascii or json)", format)
		}

		return nil
	},
}

func init() {
	reportGenerateCmd.Flags().String("id", "", "Engagement ID")
	reportGenerateCmd.Flags().String("format", "md", "Output format: json|md|html|pdf")
	reportStatsCmd.Flags().String("id", "", "Engagement ID")
	reportStatsCmd.Flags().String("format", "text", "Output format: text|table|json")
	reportTelemetryCmd.Flags().String("id", "", "Engagement ID")
	reportTelemetryCmd.Flags().String("format", "ascii", "Output format: ascii|json")
	reportTelemetryCmd.Flags().Int("limit", 10, "Number of recent runs to display")
	reportCmd.AddCommand(reportGenerateCmd)
	reportCmd.AddCommand(reportStatsCmd)
	reportCmd.AddCommand(reportTelemetryCmd)
}
