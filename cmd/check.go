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
	consts "github.com/khanhnv2901/seca-cli/internal/constants"
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
				runtimeCfg := appCtx.Config.Check
				return &checker.HTTPChecker{
					Timeout:    time.Duration(runtimeCfg.TimeoutSecs) * time.Second,
					CaptureRaw: runtimeCfg.AuditAppendRaw,
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
			TimeoutSecs:        cliConfig.Check.TimeoutSecs,
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
				runtimeCfg := appCtx.Config.Check
				return &checker.DNSChecker{
					Timeout:    time.Duration(runtimeCfg.DNS.Timeout) * time.Second,
					NameServer: runtimeCfg.DNS.Nameservers,
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
			TimeoutSecs:        cliConfig.Check.DNS.Timeout,
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
	runtimeCfg := appCtx.Config.Check

	// Parse flags
	params := checkParams{
		ID:             cmd.Flag("id").Value.String(),
		ROEConfirm:     cmd.Flag("roe-confirm").Value.String() == "true",
		ComplianceMode: cmd.Flag("compliance-mode").Value.String() == "true",
		AutoSign:       runtimeCfg.AutoSign,
		GPGKey:         runtimeCfg.GPGKey,
	}

	// Validate parameters
	retentionForValidation := 0
	if config.SupportsRawCapture {
		retentionForValidation = runtimeCfg.RetentionDays
	}
	if err := validateCheckParams(params, appCtx, config.SupportsRawCapture && runtimeCfg.AuditAppendRaw, retentionForValidation); err != nil {
		return err
	}

	// Load engagement
	eng, err := loadEngagementByID(params.ID)
	if err != nil {
		return err
	}

	startAll := time.Now()
	dir := filepath.Join(appCtx.ResultsDir, params.ID)
	if err := os.MkdirAll(dir, consts.DefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	// Create checker using the provided factory function
	chk := config.CreateChecker(appCtx, params)

	// Create audit function using the provided factory
	auditFn := config.CreateAuditFn(appCtx, params, chk)

	var progress *progressPrinter
	if runtimeCfg.ProgressEnabled {
		progress = newProgressPrinter(len(eng.Scope), chk.Name())
		progress.Start()
		if auditFn != nil {
			orig := auditFn
			auditFn = func(target string, result checker.CheckResult, duration float64) error {
				if err := orig(target, result, duration); err != nil {
					return err
				}
				progress.Increment(result.Status == "ok", duration)
				return nil
			}
		} else {
			auditFn = func(target string, result checker.CheckResult, duration float64) error {
				progress.Increment(result.Status == "ok", duration)
				return nil
			}
		}
	}

	// Create runner and execute checks
	runner := &checker.Runner{
		Concurrency: runtimeCfg.Concurrency,
		RateLimit:   runtimeCfg.RateLimit,
		Timeout:     time.Duration(config.TimeoutSecs) * time.Second,
	}

	ctx := context.Background()
	results := runner.RunChecks(ctx, eng.Scope, chk, auditFn)

	if progress != nil {
		progress.Stop()
	}

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
		rawCaptureEnabled := config.SupportsRawCapture && runtimeCfg.AuditAppendRaw
		retentionDaysForSummary := 0
		if rawCaptureEnabled {
			retentionDaysForSummary = runtimeCfg.RetentionDays
		}
		printComplianceSummary(
			appCtx, eng, auditHash, resultsHash,
			config.VerificationCmd,
			rawCaptureEnabled, retentionDaysForSummary,
		)
	}

	if runtimeCfg.TelemetryEnabled {
		runDuration := time.Since(startAll)
		if err := recordTelemetry(appCtx, params.ID, chk.Name(), results, runDuration); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record telemetry: %v\n", err)
		}
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
				return nil, &ScopeViolationError{Scope: id}
			}
			return &engs[i], nil
		}
	}
	return nil, &EngagementNotFoundError{ID: id}
}

// writeResultsAndHash writes results to JSON file, computes hashes, and returns paths and hashes
func writeResultsAndHash(appCtx *AppContext, id string, resultsFilename string, metadata RunMetadata, results []checker.CheckResult, startTime time.Time) (resultsPath, auditPath, auditHash, resultsHash string, err error) {
	dir := filepath.Join(appCtx.ResultsDir, id)
	if err := os.MkdirAll(dir, consts.DefaultDirPerm); err != nil {
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

	b, err := json.MarshalIndent(out, jsonPrefix, jsonIndent)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(resultsPath, b, consts.DefaultFilePerm); err != nil {
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
	b, err = json.MarshalIndent(out, jsonPrefix, jsonIndent)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to marshal final results: %w", err)
	}

	if err := os.WriteFile(resultsPath, b, consts.DefaultFilePerm); err != nil {
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

// addCommonCheckFlags adds flags that are common to all check commands.
// This reduces duplication and ensures consistent flag definitions across commands.
func addCommonCheckFlags(cmd *cobra.Command) {
	cmd.Flags().String("id", "", "Engagement id")
	cmd.Flags().Bool("roe-confirm", false, "Confirm you have explicit written authorization (required)")
	cmd.Flags().Bool("compliance-mode", false, "Enable compliance enforcement (hashing, retention checks)")
	cmd.Flags().BoolVar(&cliConfig.Check.AutoSign, "auto-sign", cliConfig.Check.AutoSign, "Automatically sign .sha256 files using configured GPG key")
	cmd.Flags().StringVar(&cliConfig.Check.GPGKey, "gpg-key", cliConfig.Check.GPGKey, "GPG key ID or email for signing (required if --auto-sign)")
}

func init() {
	// Global check flags (apply to all subcommands)
	checkCmd.PersistentFlags().IntVarP(&cliConfig.Check.Concurrency, "concurrency", "c", cliConfig.Check.Concurrency, "max concurrent requests")
	checkCmd.PersistentFlags().IntVarP(&cliConfig.Check.RateLimit, "rate", "r", cliConfig.Check.RateLimit, "requests per second (global)")
	checkCmd.PersistentFlags().IntVarP(&cliConfig.Check.TimeoutSecs, "timeout", "t", cliConfig.Check.TimeoutSecs, "request timeout in seconds")
	checkCmd.PersistentFlags().BoolVar(&cliConfig.Check.TelemetryEnabled, "telemetry", cliConfig.Check.TelemetryEnabled, "Record telemetry metrics (durations, success rates)")
	checkCmd.PersistentFlags().BoolVar(&cliConfig.Check.ProgressEnabled, "progress", cliConfig.Check.ProgressEnabled, "Display live progress for checks")

	// HTTP-specific flags
	addCommonCheckFlags(checkHTTPCmd)
	checkHTTPCmd.Flags().BoolVar(&cliConfig.Check.AuditAppendRaw, "audit-append-raw", cliConfig.Check.AuditAppendRaw, "Save limited raw headers/body for auditing (handle carefully)")
	checkHTTPCmd.Flags().IntVar(&cliConfig.Check.RetentionDays, "retention-days", cliConfig.Check.RetentionDays, "Retention period (days) for raw captures; required in compliance mode if --audit-append-raw is used")

	// DNS-specific flags
	addCommonCheckFlags(checkDNSCmd)
	checkDNSCmd.Flags().StringSliceVar(&cliConfig.Check.DNS.Nameservers, "nameservers", cliConfig.Check.DNS.Nameservers, "Custom DNS nameservers (e.g., 8.8.8.8:53,1.1.1.1:53)")
	checkDNSCmd.Flags().IntVar(&cliConfig.Check.DNS.Timeout, "dns-timeout", cliConfig.Check.DNS.Timeout, "DNS query timeout in seconds")

	checkCmd.AddCommand(checkHTTPCmd)
	checkCmd.AddCommand(checkDNSCmd)
	registerPluginCommands()
}
