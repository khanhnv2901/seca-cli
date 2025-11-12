package cmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/checker"
	"github.com/spf13/cobra"
)

//go:embed templates/report.html
var htmlTemplate string

//go:embed templates/report.md
var markdownTemplate string

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
		if err := os.WriteFile(reportPath, []byte(reportContent), 0644); err != nil {
			return fmt.Errorf("failed to write report: %w", err)
		}

		fmt.Printf("Report generated: %s\n", reportPath)
		fmt.Printf("Format: %s\n", format)
		fmt.Printf("Total targets: %d\n", output.Metadata.TotalTargets)

		return nil
	},
}

func generateJSONReport(output *RunOutput) (string, error) {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func generateMarkdownReport(output *RunOutput) (string, error) {
	// Calculate summary statistics
	okCount := 0
	errorCount := 0
	for _, r := range output.Results {
		if r.Status == "ok" {
			okCount++
		} else {
			errorCount++
		}
	}

	// Calculate success rate
	successRate := 0.0
	if len(output.Results) > 0 {
		successRate = float64(okCount) / float64(len(output.Results)) * 100
	}

	// Prepare template data
	data := TemplateData{
		Metadata:     output.Metadata,
		Results:      output.Results,
		GeneratedAt:  time.Now().Format(time.RFC3339),
		StartedAt:    output.Metadata.StartAt.Format(time.RFC3339),
		CompletedAt:  output.Metadata.CompleteAt.Format(time.RFC3339),
		Duration:     output.Metadata.CompleteAt.Sub(output.Metadata.StartAt).Round(time.Second).String(),
		SuccessCount: okCount,
		ErrorCount:   errorCount,
		SuccessRate:  fmt.Sprintf("%.2f", successRate),
		FooterDate:   time.Now().Format("2006-01-02 15:04:05"),
	}

	// Create template with helper functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"join": func(arr []string, sep string) string {
			return strings.Join(arr, sep)
		},
		"headersPresentCount": func(sh *checker.SecurityHeadersResult) int {
			count := 0
			for _, header := range sh.Headers {
				if header.Present {
					count++
				}
			}
			return count
		},
		"highSeverityMissing": func(sh *checker.SecurityHeadersResult) []string {
			var missing []string
			headerNames := []string{
				"Strict-Transport-Security",
				"Content-Security-Policy",
				"X-Content-Type-Options",
				"X-Frame-Options",
				"Referrer-Policy",
				"Permissions-Policy",
				"Cross-Origin-Opener-Policy",
				"Cross-Origin-Embedder-Policy",
			}
			for _, name := range headerNames {
				if header, ok := sh.Headers[name]; ok && !header.Present && header.Severity == "high" {
					missing = append(missing, name)
				}
			}
			return missing
		},
		"mediumSeverityMissing": func(sh *checker.SecurityHeadersResult) []string {
			var missing []string
			headerNames := []string{
				"Strict-Transport-Security",
				"Content-Security-Policy",
				"X-Content-Type-Options",
				"X-Frame-Options",
				"Referrer-Policy",
				"Permissions-Policy",
				"Cross-Origin-Opener-Policy",
				"Cross-Origin-Embedder-Policy",
			}
			for _, name := range headerNames {
				if header, ok := sh.Headers[name]; ok && !header.Present && header.Severity == "medium" {
					missing = append(missing, name)
				}
			}
			return missing
		},
		"hasHighSeverityMissing": func(sh *checker.SecurityHeadersResult) bool {
			headerNames := []string{
				"Strict-Transport-Security",
				"Content-Security-Policy",
				"X-Content-Type-Options",
				"X-Frame-Options",
				"Referrer-Policy",
				"Permissions-Policy",
				"Cross-Origin-Opener-Policy",
				"Cross-Origin-Embedder-Policy",
			}
			for _, name := range headerNames {
				if header, ok := sh.Headers[name]; ok && !header.Present && (header.Severity == "high" || header.Severity == "medium") {
					return true
				}
			}
			return false
		},
	}

	// Parse and execute template
	tmpl, err := template.New("report").Funcs(funcMap).Parse(markdownTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
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
	// Calculate summary statistics
	okCount := 0
	errorCount := 0
	for _, r := range output.Results {
		if r.Status == "ok" {
			okCount++
		} else {
			errorCount++
		}
	}

	// Calculate success rate
	successRate := 0.0
	if len(output.Results) > 0 {
		successRate = float64(okCount) / float64(len(output.Results)) * 100
	}

	// Prepare template data
	data := TemplateData{
		Metadata:     output.Metadata,
		Results:      output.Results,
		GeneratedAt:  time.Now().Format(time.RFC3339),
		StartedAt:    output.Metadata.StartAt.Format(time.RFC3339),
		CompletedAt:  output.Metadata.CompleteAt.Format(time.RFC3339),
		Duration:     output.Metadata.CompleteAt.Sub(output.Metadata.StartAt).Round(time.Second).String(),
		SuccessCount: okCount,
		ErrorCount:   errorCount,
		SuccessRate:  fmt.Sprintf("%.1f", successRate),
		FooterDate:   time.Now().Format("2006-01-02 15:04:05"),
	}

	// Create template with helper functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"join": func(arr []string, sep string) string {
			return strings.Join(arr, sep)
		},
		"headersPresentCount": func(sh *checker.SecurityHeadersResult) int {
			count := 0
			for _, header := range sh.Headers {
				if header.Present {
					count++
				}
			}
			return count
		},
	}

	// Parse and execute template
	tmpl, err := template.New("report").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
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
