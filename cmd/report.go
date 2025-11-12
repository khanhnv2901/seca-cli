package cmd

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/checker"
	consts "github.com/khanhnv2901/seca-cli/internal/constants"
	"github.com/spf13/cobra"
)

const (
	htmlTemplatePath     = "templates/report.html"
	markdownTemplatePath = "templates/report.md"
)

//go:embed templates/report.html templates/report.md
var reportTemplateFS embed.FS

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

		// Read results from results directory
		path := filepath.Join(appCtx.ResultsDir, id, "results.json")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("no results found at %s", path)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Parse results
		var output RunOutput
		if err := json.Unmarshal(data, &output); err != nil {
			return fmt.Errorf("failed to parse results: %w", err)
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
			reportContent, err = generateJSONReport(&output)
			filename = "report.json"
		case "md":
			data := buildTemplateData(&output, "%.2f", trendHistory)
			reportContent, err = generateMarkdownReport(data)
			filename = "report.md"
		case "html":
			data := buildTemplateData(&output, "%.1f", trendHistory)
			reportContent, err = generateHTMLReport(data)
			filename = "report.html"
		case "pdf":
			data := buildTemplateData(&output, "%.1f", trendHistory)
			pdfBytes, perr := generatePDFReportBytes(data)
			if perr != nil {
				return fmt.Errorf("failed to generate PDF report: %w", perr)
			}
			filename = "report.pdf"
			reportPath := filepath.Join(appCtx.ResultsDir, id, filename)
			if err := os.WriteFile(reportPath, pdfBytes, consts.DefaultFilePerm); err != nil {
				return fmt.Errorf("failed to write report: %w", err)
			}
			fmt.Printf("Report generated: %s\n", reportPath)
			fmt.Printf("Format: %s\n", format)
			fmt.Printf("Total targets: %d\n", output.Metadata.TotalTargets)
			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to generate report: %w", err)
		}

		// Write report to file
		reportPath := filepath.Join(appCtx.ResultsDir, id, filename)
		if err := os.WriteFile(reportPath, []byte(reportContent), consts.DefaultFilePerm); err != nil {
			return fmt.Errorf("failed to write report: %w", err)
		}

		fmt.Printf("Report generated: %s\n", reportPath)
		fmt.Printf("Format: %s\n", format)
		fmt.Printf("Total targets: %d\n", output.Metadata.TotalTargets)

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

func generatePDFReportBytes(data TemplateData) ([]byte, error) {
	lines := buildPDFLines(data)
	var contentBuilder strings.Builder
	contentBuilder.WriteString("BT\n/F1 12 Tf\n72 750 Td\n")
	firstLine := true
	for _, line := range lines {
		escaped := pdfEscape(line)
		if firstLine {
			contentBuilder.WriteString(fmt.Sprintf("(%s) Tj\n", escaped))
			firstLine = false
			continue
		}
		contentBuilder.WriteString("T*\n")
		contentBuilder.WriteString(fmt.Sprintf("(%s) Tj\n", escaped))
	}
	contentBuilder.WriteString("ET\n")
	content := contentBuilder.String()
	contentLen := len(content)

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	offsets := make([]int, 6)
	writeObj := func(idx int, obj string) {
		offsets[idx] = buf.Len()
		buf.WriteString(obj)
		if !strings.HasSuffix(obj, "\n") {
			buf.WriteString("\n")
		}
	}

	writeObj(1, "1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj")
	writeObj(2, "2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj")
	writeObj(3, "3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >> endobj")
	writeObj(4, fmt.Sprintf("4 0 obj << /Length %d >> stream\n%sendstream\nendobj", contentLen, content))
	writeObj(5, "5 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> endobj")

	xrefStart := buf.Len()
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	buf.WriteString("trailer << /Size 6 /Root 1 0 R >>\n")
	buf.WriteString(fmt.Sprintf("startxref\n%d\n%%%%EOF\n", xrefStart))

	return buf.Bytes(), nil
}

func buildPDFLines(data TemplateData) []string {
	lines := []string{
		fmt.Sprintf("Engagement Report: %s", data.Metadata.EngagementName),
		fmt.Sprintf("Engagement ID: %s", data.Metadata.EngagementID),
		fmt.Sprintf("Operator: %s", data.Metadata.Operator),
		fmt.Sprintf("Started: %s", data.StartedAt),
		fmt.Sprintf("Completed: %s", data.CompletedAt),
		"",
		fmt.Sprintf("Summary: %d OK / %d Errors (Success %s)", data.SuccessCount, data.ErrorCount, data.SuccessRate),
	}

	if len(data.TrendHistory) > 0 {
		lines = append(lines, "", "Trend Analysis:")
		lines = append(lines, fmt.Sprintf("Average Success: %.1f%%", data.TrendSummary.AverageSuccess))
		lines = append(lines, fmt.Sprintf("Average Duration: %s", formatDurationLabel(data.TrendSummary.AverageDuration)))
		for _, rec := range data.TrendHistory {
			lines = append(lines, fmt.Sprintf("%s -> %s success, %s", formatShortTimestamp(rec.Timestamp), formatSuccessRate(rec.SuccessRate), formatDurationLabel(rec.DurationSeconds)))
		}
	}

	lines = append(lines, "", "Results:")
	maxResults := 25
	for i, r := range data.Results {
		if i == maxResults {
			lines = append(lines, fmt.Sprintf("... %d additional targets omitted ...", len(data.Results)-maxResults))
			break
		}
		notes := strings.TrimSpace(r.Notes)
		if notes == "" {
			notes = "No notes"
		}
		lines = append(lines, fmt.Sprintf("%s - %s (%s)", r.Target, strings.ToUpper(r.Status), notes))
	}

	return lines
}

func pdfEscape(line string) string {
	line = strings.ReplaceAll(line, "\\", "\\\\")
	line = strings.ReplaceAll(line, "(", "\\(")
	line = strings.ReplaceAll(line, ")", "\\)")
	return line
}

func buildTemplateData(output *RunOutput, successRateFmt string, trends []TelemetryRecord) TemplateData {
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

	return TemplateData{
		Metadata:           output.Metadata,
		Results:            output.Results,
		GeneratedAt:        now.Format(time.RFC3339),
		StartedAt:          output.Metadata.StartAt.Format(time.RFC3339),
		CompletedAt:        output.Metadata.CompleteAt.Format(time.RFC3339),
		Duration:           duration.Round(time.Second).String(),
		SuccessCount:       okCount,
		ErrorCount:         errorCount,
		SuccessRate:        fmt.Sprintf(successRateFmt, successRate),
		FooterDate:         now.Format("2006-01-02 15:04:05"),
		TrendHistory:       trends,
		TrendSummary:       summarizeTrendHistory(trends),
		HashAlgorithmLabel: strings.ToUpper(output.Metadata.HashAlgorithm),
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

func printTelemetryASCII(records []TelemetryRecord) {
	const barWidth = 40
	fmt.Println("Telemetry Success Rate Trend")
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
		// Get application context
		appCtx := getAppContext(cmd)

		id, _ := cmd.Flags().GetString("id")
		path := filepath.Join(appCtx.ResultsDir, id, "results.json")
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var output RunOutput
		if err := json.Unmarshal(data, &output); err != nil {
			return err
		}
		normalizeRunMetadata(&output.Metadata)

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
		fmt.Printf("Targets: %d | OK: %d | Fail: %d | TLS <30d: %d\n", total, ok, fail, soon)
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
			fmt.Printf("No telemetry records found for engagement %s\n", id)
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
	reportTelemetryCmd.Flags().String("id", "", "Engagement ID")
	reportTelemetryCmd.Flags().String("format", "ascii", "Output format: ascii|json")
	reportTelemetryCmd.Flags().Int("limit", 10, "Number of recent runs to display")
	reportCmd.AddCommand(reportGenerateCmd)
	reportCmd.AddCommand(reportStatsCmd)
	reportCmd.AddCommand(reportTelemetryCmd)
}
