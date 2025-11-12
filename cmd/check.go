package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/checker"
	"github.com/spf13/cobra"
)

type RunMetadata struct {
	Operator       string    `json:"operator"`
	EngagementID   string    `json:"engagement_id"`
	EngagementName string    `json:"engagement_name"`
	Owner          string    `json:"owner"`
	StartAt        time.Time `json:"started_at"`
	CompleteAt     time.Time `json:"completed_at"`
	AuditHash      string    `json:"audit_sha256"`
	TotalTargets   int       `json:"total_targets"`
	// Note: results.json hash is stored in results.json.sha256 file, not here
}

type RunOutput struct {
	Metadata RunMetadata           `json:"metadata"`
	Results  []checker.CheckResult `json:"results"`
}

var (
	concurrency    int
	rateLimit      int // requests per second
	timeoutSecs    int
	auditAppendRaw bool
	complianceMode bool
	retentionDays  int
	autoSign       bool
	gpgKey         string
)

var (
	dnsNameservers []string
	dnsTimeout     int
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run safe, authorized checks against scoped targets (no scanning/exploitation)",
}

var checkHTTPCmd = &cobra.Command{
	Use:   "http",
	Short: "Run safe HTTP/TLS checks for an engagement's scope",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheckCommand(cmd, checkConfig{
			CreateChecker: func(appCtx *AppContext, params checkParams) checker.Checker {
				return &checker.HTTPChecker{
					Timeout:    time.Duration(timeoutSecs) * time.Second,
					CaptureRaw: auditAppendRaw,
					RawHandler: func(target string, headers http.Header, bodySnippet string) error {
						return SaveRawCapture(appCtx.ResultsDir, params.ID, target, headers, bodySnippet)
					},
				}
			},
			CreateAuditFn: func(appCtx *AppContext, params checkParams, chk checker.Checker) func(string, checker.CheckResult, float64) error {
				return func(target string, result checker.CheckResult, duration float64) error {
					return AppendAuditRow(
						appCtx.ResultsDir,
						params.ID,
						appCtx.Operator,
						chk.Name(),
						target,
						result.Status,
						result.HTTPStatus,
						result.TLSExpiry,
						result.Notes,
						result.Error,
						duration,
					)
				}
			},
			ResultsFilename:    "results.json",
			TimeoutSecs:        timeoutSecs,
			VerificationCmd:    "sha256sum -c audit.csv.sha256 && sha256sum -c results_*.sha256",
			SupportsRawCapture: true,
			PrintSummary: func(results []checker.CheckResult, resultsPath, auditPath, auditHash, resultsHash string) {
				fmt.Printf("Run complete.\n")
				fmt.Printf("Results: %s\nAudit: %s\n", resultsPath, auditPath)
				fmt.Printf("SHA256 audit: %s\nSHA256 results: %s\n", auditHash, resultsHash)
			},
		})
	},
}

var checkDNSCmd = &cobra.Command{
	Use:   "dns",
	Short: "Run DNS checks for an engagement's scope",
	Long: `Perform DNS resolution checks on engagement targets.
	This command will:
- Resolve A records (IPv4 addresses)
- Resolve AAAA records (IPv6 addresses)
- Check CNAME records
- Check MX records (mail servers)
- Check NS records (nameservers)
- Check TXT records (SPF, DKIM, etc.)
- Perform reverse DNS lookups (PTR records)

All checks are safe, non-intrusive DNS queries only.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheckCommand(cmd, checkConfig{
			CreateChecker: func(appCtx *AppContext, params checkParams) checker.Checker {
				return &checker.DNSChecker{
					Timeout:    time.Duration(dnsTimeout) * time.Second,
					NameServer: dnsNameservers,
				}
			},
			CreateAuditFn: func(appCtx *AppContext, params checkParams, chk checker.Checker) func(string, checker.CheckResult, float64) error {
				return func(target string, result checker.CheckResult, duration float64) error {
					return AppendAuditRow(
						appCtx.ResultsDir,
						params.ID,
						appCtx.Operator,
						chk.Name(),
						target,
						result.Status,
						0,  // No HTTP status for DNS
						"", // No TLS expiry for DNS
						result.Notes,
						result.Error,
						duration,
					)
				}
			},
			ResultsFilename:    "dns_results.json",
			TimeoutSecs:        dnsTimeout,
			VerificationCmd:    "sha256sum -c audit.csv.sha256 && sha256sum -c dns_results.json.sha256",
			SupportsRawCapture: false,
			PrintSummary: func(results []checker.CheckResult, resultsPath, auditPath, auditHash, resultsHash string) {
				// Count successes and errors
				okCount := 0
				errorCount := 0
				for _, r := range results {
					if r.Status == "ok" {
						okCount++
					} else {
						errorCount++
					}
				}

				fmt.Printf("DNS Check complete.\n")
				fmt.Printf("Results: %s\nAudit: %s\n", resultsPath, auditPath)
				fmt.Printf("SHA256 audit: %s\nSHA256 results: %s\n", auditHash, resultsHash)
				fmt.Printf("Summary: %d OK, %d Errors (out of %d targets)\n", okCount, errorCount, len(results))
			},
		})
	},
}

// checkParams holds common parameters for check commands
type checkParams struct {
	ID             string
	ROEConfirm     bool
	ComplianceMode bool
	AutoSign       bool
	GPGKey         string
}

// checkConfig holds configuration for running a check command
type checkConfig struct {
	// Checker creation function
	CreateChecker func(appCtx *AppContext, params checkParams) checker.Checker

	// Audit function creation
	CreateAuditFn func(appCtx *AppContext, params checkParams, chk checker.Checker) func(string, checker.CheckResult, float64) error

	// Results filename (e.g., "results.json" or "dns_results.json")
	ResultsFilename string

	// Timeout in seconds for the runner
	TimeoutSecs int

	// Verification command for compliance summary
	VerificationCmd string

	// Whether this check supports raw capture
	SupportsRawCapture bool

	// Custom result summary printer (optional)
	PrintSummary func(results []checker.CheckResult, resultsPath, auditPath, auditHash, resultsHash string)
}

// runCheckCommand executes a check command with the given configuration.
// This is the common execution pattern shared by all check commands (HTTP, DNS, etc.)
func runCheckCommand(cmd *cobra.Command, config checkConfig) error {
	// Get application context
	appCtx := getAppContext(cmd)

	// Parse flags
	params := checkParams{
		ID:             cmd.Flag("id").Value.String(),
		ROEConfirm:     cmd.Flag("roe-confirm").Value.String() == "true",
		ComplianceMode: cmd.Flag("compliance-mode").Value.String() == "true",
		AutoSign:       cmd.Flag("auto-sign").Value.String() == "true",
		GPGKey:         cmd.Flag("gpg-key").Value.String(),
	}

	// Validate parameters
	retentionForValidation := 0
	if config.SupportsRawCapture {
		retentionForValidation = retentionDays
	}
	if err := validateCheckParams(params, appCtx, config.SupportsRawCapture && auditAppendRaw, retentionForValidation); err != nil {
		return err
	}

	// Load engagement
	eng, err := loadEngagementByID(params.ID)
	if err != nil {
		return err
	}

	startAll := time.Now()
	dir := filepath.Join(appCtx.ResultsDir, params.ID)
	_ = os.MkdirAll(dir, 0o755)

	// Create checker using the provided factory function
	chk := config.CreateChecker(appCtx, params)

	// Create audit function using the provided factory
	auditFn := config.CreateAuditFn(appCtx, params, chk)

	// Create runner and execute checks
	runner := &checker.Runner{
		Concurrency: concurrency,
		RateLimit:   rateLimit,
		Timeout:     time.Duration(config.TimeoutSecs) * time.Second,
	}

	ctx := context.Background()
	results := runner.RunChecks(ctx, eng.Scope, chk, auditFn)

	// Write results and compute hashes
	metadata := RunMetadata{
		Operator:       appCtx.Operator,
		EngagementID:   params.ID,
		EngagementName: eng.Name,
		Owner:          eng.Owner,
		StartAt:        startAll,
	}

	resultsPath, auditPath, auditHash, resultsHash, err := writeResultsAndHash(
		appCtx, params.ID, config.ResultsFilename, metadata, results, startAll,
	)
	if err != nil {
		return err
	}

	// GPG signing if requested
	if params.AutoSign {
		if err := signHashFiles(auditPath, resultsPath, params.GPGKey); err != nil {
			return err
		}
	}

	// Print results summary
	if config.PrintSummary != nil {
		config.PrintSummary(results, resultsPath, auditPath, auditHash, resultsHash)
	} else {
		// Default summary
		fmt.Printf("Run complete.\n")
		fmt.Printf("Results: %s\nAudit: %s\n", resultsPath, auditPath)
		fmt.Printf("SHA256 audit: %s\nSHA256 results: %s\n", auditHash, resultsHash)
	}

	// Print compliance summary if in compliance mode
	if params.ComplianceMode {
		rawCaptureEnabled := config.SupportsRawCapture && auditAppendRaw
		retentionDaysForSummary := 0
		if rawCaptureEnabled {
			retentionDaysForSummary = retentionDays
		}
		printComplianceSummary(
			appCtx, eng, auditHash, resultsHash,
			config.VerificationCmd,
			rawCaptureEnabled, retentionDaysForSummary,
		)
	}

	return nil
}

// validateCheckParams validates common check command parameters
func validateCheckParams(params checkParams, appCtx *AppContext, auditAppendRaw bool, retentionDays int) error {
	if params.ID == "" {
		return fmt.Errorf("--id is required")
	}

	if !params.ROEConfirm {
		return fmt.Errorf("this action requires --roe-confirm to proceed (ensures explicit written authorization)")
	}

	if appCtx.Operator == "" {
		return fmt.Errorf("--operator is required")
	}

	if params.ComplianceMode {
		fmt.Println("[Compliance Mode] Enabled")

		if appCtx.Operator == "" {
			return fmt.Errorf("--operator required in compliance mode")
		}

		fmt.Println("-> Hash-signing of audit and result files enforced")

		// HTTP-specific validation
		if auditAppendRaw && retentionDays <= 0 {
			return fmt.Errorf("in compliance mode, --audit-append-raw requires --retention-days=<N>")
		}
	}

	return nil
}

// loadEngagementByID loads and validates an engagement by ID
func loadEngagementByID(id string) (*Engagement, error) {
	engs := loadEngagements()
	for i := range engs {
		if engs[i].ID == id {
			if len(engs[i].Scope) == 0 {
				return nil, fmt.Errorf("no scope found for engagement %s", id)
			}
			return &engs[i], nil
		}
	}
	return nil, fmt.Errorf("no engagement found with id %s", id)
}

// writeResultsAndHash writes results to JSON file, computes hashes, and returns paths and hashes
func writeResultsAndHash(appCtx *AppContext, id string, resultsFilename string, metadata RunMetadata, results []checker.CheckResult, startTime time.Time) (resultsPath, auditPath, auditHash, resultsHash string, err error) {
	dir := filepath.Join(appCtx.ResultsDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", "", "", fmt.Errorf("failed to create results directory: %w", err)
	}

	// Write results JSON (first pass without audit hash)
	resultsPath = filepath.Join(dir, resultsFilename)
	out := RunOutput{
		Metadata: metadata,
		Results:  results,
	}

	// Set completion time
	out.Metadata.CompleteAt = time.Now().UTC()
	out.Metadata.TotalTargets = len(results)

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(resultsPath, b, 0o644); err != nil {
		return "", "", "", "", fmt.Errorf("failed to write results: %w", err)
	}

	// Compute hash for audit.csv
	auditPath = filepath.Join(dir, "audit.csv")
	auditHash, err = HashFileSHA256(auditPath)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to hash audit file: %w", err)
	}

	// Update metadata with audit hash and write final results JSON
	out.Metadata.AuditHash = auditHash
	b, err = json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to marshal final results: %w", err)
	}

	if err := os.WriteFile(resultsPath, b, 0o644); err != nil {
		return "", "", "", "", fmt.Errorf("failed to write final results: %w", err)
	}

	// Hash results.json AFTER final write
	resultsHash, err = HashFileSHA256(resultsPath)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to hash results file: %w", err)
	}

	return resultsPath, auditPath, auditHash, resultsHash, nil
}

// signHashFiles signs the .sha256 files using GPG
func signHashFiles(auditPath, resultsPath, gpgKey string) error {
	if gpgKey == "" {
		return fmt.Errorf("--gpg-key required with --auto-sign")
	}

	signFile := func(path string) error {
		cmd := exec.Command("gpg", "--armor", "--local-user", gpgKey, "--sign", path)
		cmd.Dir = filepath.Dir(path)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		return cmd.Run()
	}

	if err := signFile(auditPath + ".sha256"); err != nil {
		return fmt.Errorf("failed to sign audit hash file: %w", err)
	}

	if err := signFile(resultsPath + ".sha256"); err != nil {
		return fmt.Errorf("failed to sign results hash file: %w", err)
	}

	fmt.Println("GPG signatures created for both .sha256 files.")
	return nil
}

// printComplianceSummary prints the compliance mode summary
func printComplianceSummary(appCtx *AppContext, eng *Engagement, auditHash, resultsHash string, verificationCmd string, auditAppendRaw bool, retentionDays int) {
	fmt.Println("------------------------------------------------------")
	fmt.Println("🔒 Compliance Summary")
	fmt.Printf("Operator: %s\nEngagement: %s (%s)\n", appCtx.Operator, eng.Name, eng.ID)
	fmt.Printf("Audit hash : %s\nResults hash: %s\n", auditHash, resultsHash)
	fmt.Printf("Verification: %s\n", verificationCmd)
	if auditAppendRaw {
		fmt.Printf("Retention: raw captures must be deleted or anonymized after %d day(s).\n", retentionDays)
	}
	fmt.Println("Evidence integrity and retention requirements satisfied.")
	fmt.Println("------------------------------------------------------")
}

func init() {
	checkCmd.PersistentFlags().IntVarP(&concurrency, "concurrency", "c", 1, "max concurrent requests")
	checkCmd.PersistentFlags().IntVarP(&rateLimit, "rate", "r", 1, "requests per second (global)")
	checkCmd.PersistentFlags().IntVarP(&timeoutSecs, "timeout", "t", 10, "request timeout in seconds")
	// HTTP command flags
	checkHTTPCmd.Flags().String("id", "", "Engagement id")
	checkHTTPCmd.Flags().Bool("roe-confirm", false, "Confirm you have explicit written authorization (required)")
	checkHTTPCmd.Flags().BoolVar(&auditAppendRaw, "audit-append-raw", false, "Save limited raw headers/body for auditing (handle carefully)")
	checkHTTPCmd.Flags().BoolVar(&complianceMode, "compliance-mode", false, "Enable compliance enforcement (hashing, retention checks)")
	checkHTTPCmd.Flags().IntVar(&retentionDays, "retention-days", 0, "Retention period (days) for raw captures; required in compliance mode if --audit-append-raw is used")
	checkHTTPCmd.Flags().Bool("auto-sign", false, "Automatically sign .sha256 files using configured GPG key")
	checkHTTPCmd.Flags().String("gpg-key", "", "GPG key ID or email for signing (required if --auto-sign)")

	// DNS command flags
	checkDNSCmd.Flags().String("id", "", "Engagement id")
	checkDNSCmd.Flags().Bool("roe-confirm", false, "Confirm you have explicit written authorization (required)")
	checkDNSCmd.Flags().Bool("compliance-mode", false, "Enable compliance enforcement (hashing, retention checks)")
	checkDNSCmd.Flags().Bool("auto-sign", false, "Automatically sign .sha256 files using configured GPG key")
	checkDNSCmd.Flags().String("gpg-key", "", "GPG key ID or email for signing (required if --auto-sign)")
	checkDNSCmd.Flags().StringSliceVar(&dnsNameservers, "nameservers", []string{}, "Custom DNS nameservers (e.g., 8.8.8.8:53,1.1.1.1:53)")
	checkDNSCmd.Flags().IntVar(&dnsTimeout, "dns-timeout", 10, "DNS query timeout in seconds")

	checkCmd.AddCommand(checkHTTPCmd)
	checkCmd.AddCommand(checkDNSCmd)
}
