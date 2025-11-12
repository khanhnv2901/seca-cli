package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
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
	}

	markdownTemplateFuncs = template.FuncMap{
		"add":                    addInts,
		"join":                   strings.Join,
		"headersPresentCount":    headersPresentCount,
		"highSeverityMissing":    missingHighSeverityHeaders,
		"mediumSeverityMissing":  missingMediumSeverityHeaders,
		"hasHighSeverityMissing": hasCriticalMissingHeaders,
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
		if format != "json" && format != "md" && format != "html" {
			return fmt.Errorf("invalid format: %s (must be json, md, or html)", format)
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

		// Generate report based on format
		var reportContent string
		var filename string

		switch format {
		case "json":
			reportContent, err = generateJSONReport(&output)
			filename = "report.json"
		case "md":
			reportContent, err = generateMarkdownReport(&output)
			filename = "report.md"
		case "html":
			reportContent, err = generateHTMLReport(&output)
			filename = "report.html"
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

func generateMarkdownReport(output *RunOutput) (string, error) {
	data := buildTemplateData(output, "%.2f")
	return executeTemplate(markdownReportTemplate, data)
}

// TemplateData holds the data for HTML template rendering
type TemplateData struct {
	Metadata     RunMetadata
	Results      []checker.CheckResult
	GeneratedAt  string
	StartedAt    string
	CompletedAt  string
	Duration     string
	SuccessCount int
	ErrorCount   int
	SuccessRate  string
	FooterDate   string
}

func generateHTMLReport(output *RunOutput) (string, error) {
	data := buildTemplateData(output, "%.1f")
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

func buildTemplateData(output *RunOutput, successRateFmt string) TemplateData {
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
		Metadata:     output.Metadata,
		Results:      output.Results,
		GeneratedAt:  now.Format(time.RFC3339),
		StartedAt:    output.Metadata.StartAt.Format(time.RFC3339),
		CompletedAt:  output.Metadata.CompleteAt.Format(time.RFC3339),
		Duration:     duration.Round(time.Second).String(),
		SuccessCount: okCount,
		ErrorCount:   errorCount,
		SuccessRate:  fmt.Sprintf(successRateFmt, successRate),
		FooterDate:   now.Format("2006-01-02 15:04:05"),
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

func executeTemplate(tmpl *template.Template, data TemplateData) (string, error) {
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute %s template: %w", tmpl.Name(), err)
	}
	return buf.String(), nil
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

func init() {
	reportGenerateCmd.Flags().String("id", "", "Engagement ID")
	reportGenerateCmd.Flags().String("format", "md", "Output format: json|md|html")
	reportStatsCmd.Flags().String("id", "", "Engagement ID")
	reportCmd.AddCommand(reportGenerateCmd)
	reportCmd.AddCommand(reportStatsCmd)
}
