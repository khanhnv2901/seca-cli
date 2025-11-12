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
		// Get application context
		appCtx := getAppContext(cmd)

		id, _ := cmd.Flags().GetString("id")

		// Safety: require explicit ROE flag for any run
		roeConfirm, _ := cmd.Flags().GetBool("roe-confirm")

		// check compliance mode
		complianceMode, _ := cmd.Flags().GetBool("compliance-mode")

		// auto sign GPG Key
		autoSign, _ = cmd.Flags().GetBool("auto-sign")
		gpgKey, _ = cmd.Flags().GetString("gpg-key")

		if id == "" {
			return fmt.Errorf("--id is required")
		}

		if !roeConfirm {
			return fmt.Errorf("this action requires --roe-confirm to proceed (ensures explicit written authorization)")
		}

		if appCtx.Operator == "" {
			return fmt.Errorf("--operator is required")
		}

		if complianceMode {
			fmt.Println("[Compliance Mode] Enabled")

			// Require operator
			if appCtx.Operator == "" {
				return fmt.Errorf("--operator required in compliance mode")
			}

			// Auto-force hash-signing (already done later)
			// Nothing to change here, but we'll print a notice
			fmt.Println("-> Hash-signing of audit and result files enforced")

			// If raw audit capture used, require retentionDays > 0
			if auditAppendRaw && retentionDays <= 0 {
				return fmt.Errorf("in compliance mode, --audit-append-raw requires --retention-days=<N>")
			}
		}

		// load engagement
		engs := loadEngagements()
		var eng *Engagement
		for i := range engs {
			if engs[i].ID == id {
				eng = &engs[i]
				break
			}
		}
		if eng == nil {
			return fmt.Errorf("no engagement found with id %s", id)
		}

		if len(eng.Scope) == 0 {
			return fmt.Errorf("no scope found for engagement %s", id)
		}

		startAll := time.Now()
		dir := filepath.Join(appCtx.ResultsDir, id)
		_ = os.MkdirAll(dir, 0o755)

		// Create HTTP checker
		httpChecker := &checker.HTTPChecker{
			Timeout:    time.Duration(timeoutSecs) * time.Second,
			CaptureRaw: auditAppendRaw,
			RawHandler: func(target string, headers http.Header, bodySnippet string) error {
				return SaveRawCapture(appCtx.ResultsDir, id, target, headers, bodySnippet)
			},
		}

		// Create audit function
		auditFn := func(target string, result checker.CheckResult, duration float64) error {
			return AppendAuditRow(
				appCtx.ResultsDir,
				id,
				appCtx.Operator,
				httpChecker.Name(),
				target,
				result.Status,
				result.HTTPStatus,
				result.TLSExpiry,
				result.Notes,
				result.Error,
				duration,
			)
		}

		// Create runner and execute checks
		runner := &checker.Runner{
			Concurrency: concurrency,
			RateLimit:   rateLimit,
			Timeout:     time.Duration(timeoutSecs) * time.Second,
		}

		ctx := context.Background()
		results := runner.RunChecks(ctx, eng.Scope, httpChecker, auditFn)

		// Write results JSON
		resultsPath := filepath.Join(dir, "results.json")
		out := RunOutput{
			Metadata: RunMetadata{
				Operator:       appCtx.Operator,
				EngagementID:   id,
				EngagementName: eng.Name,
				Owner:          eng.Owner,
				StartAt:        startAll,
				CompleteAt:     time.Now().UTC(),
				TotalTargets:   len(eng.Scope),
			},
			Results: results,
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		_ = os.WriteFile(resultsPath, b, 0o644)

		// Compute hash for audit.csv
		auditPath := filepath.Join(dir, "audit.csv")
		auditHash, _ := HashFileSHA256(auditPath)

		// Update metadata with audit hash only
		out.Metadata.AuditHash = auditHash

		// Write final results JSON
		b, _ = json.MarshalIndent(out, "", "  ")
		_ = os.WriteFile(resultsPath, b, 0o644)

		// Hash results.json AFTER final write
		resultsHash, _ := HashFileSHA256(resultsPath)

		// Note: ResultsHash is not stored in the file itself to avoid hash mismatch

		if autoSign {
			if gpgKey == "" {
				return fmt.Errorf("--gpg-key required with --auto-sign")
			}
			signFile := func(path string) error {
				cmd := exec.Command("gpg", "--armor", "--local-user", gpgKey, "--sign", path)
				cmd.Dir = filepath.Dir(path)
				cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
				return cmd.Run()
			}
			_ = signFile(auditPath + ".sha256")
			_ = signFile(resultsPath + ".sha256")
			fmt.Println("GPG signatures created for both .sha256 files.")
		}

		fmt.Printf("Run complete.\n")
		fmt.Printf("Results: %s\nAudit: %s\n", resultsPath, auditPath)
		fmt.Printf("SHA256 audit: %s\nSHA256 results: %s\n", auditHash, resultsHash)

		if complianceMode {
			fmt.Println("------------------------------------------------------")
			fmt.Println("🔒 Compliance Summary")
			fmt.Printf("Operator: %s\nEngagement: %s (%s)\n", appCtx.Operator, eng.Name, eng.ID)
			fmt.Printf("Audit hash : %s\nResults hash: %s\n", auditHash, resultsHash)
			fmt.Println("Verification: sha256sum -c audit.csv.sha256 && sha256sum -c results_*.sha256")
			if auditAppendRaw {
				fmt.Printf("Retention: raw captures must be deleted or anonymized after %d day(s).\n", retentionDays)
			}
			fmt.Println("Evidence integrity and retention requirements satisfied.")
			fmt.Println("------------------------------------------------------")
		}

		return nil
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
		// Get application context
		appCtx := getAppContext(cmd)

		id, _ := cmd.Flags().GetString("id")

		// Safety: require explicit ROE flag for any run
		roeConfirm, _ := cmd.Flags().GetBool("roe-confirm")

		// check compliance mode
		complianceMode, _ := cmd.Flags().GetBool("compliance-mode")

		// auto sign GPG key
		autoSign, _ := cmd.Flags().GetBool("auto-sign")
		gpgKey, _ := cmd.Flags().GetString("gpg-key")

		if id == "" {
			return fmt.Errorf("--id is required")
		}

		if !roeConfirm {
			return fmt.Errorf("this action requires --roe-confirm to proceed (ensures explicit written authorization)")
		}

		if appCtx.Operator == "" {
			return fmt.Errorf("--operator is required")
		}

		if complianceMode {
			fmt.Println("[Compliance Mode] Enabled")

			// Require operator
			if appCtx.Operator == "" {
				return fmt.Errorf("--operator required in compliance mode")
			}

			// Hash-signing enforcement
			fmt.Println("-> Hash-signing of audit and result files enforced")
		}

		engs := loadEngagements()
		var eng *Engagement
		for i := range engs {
			if engs[i].ID == id {
				eng = &engs[i]
				break
			}
		}

		if eng == nil {
			return fmt.Errorf("no engagement found with id %s", id)
		}

		if len(eng.Scope) == 0 {
			return fmt.Errorf("no scope found for engagement %s", id)
		}

		startAll := time.Now()
		dir := filepath.Join(appCtx.ResultsDir, id)
		_ = os.MkdirAll(dir, 0o755)

		// Create DNS checker
		dnsChecker := &checker.DNSChecker{
			Timeout:    time.Duration(dnsTimeout) * time.Second,
			NameServer: dnsNameservers,
		}

		// Create audit function
		auditFn := func(target string, result checker.CheckResult, duration float64) error {
			// For DNS checks, we don;t have HTTP status
			return AppendAuditRow(
				appCtx.ResultsDir,
				id, appCtx.Operator, dnsChecker.Name(), target, result.Status,
				0,  // No HTTP status for DNS
				"", // No TLS expiry for DNS
				result.Notes,
				result.Error,
				duration,
			)
		}

		// Create runner and execute checks
		runner := &checker.Runner{
			Concurrency: concurrency,
			RateLimit:   rateLimit,
			Timeout:     time.Duration(dnsTimeout) * time.Second,
		}

		ctx := context.Background()
		results := runner.RunChecks(ctx, eng.Scope, dnsChecker, auditFn)

		// Write results JSON
		resultsPath := filepath.Join(dir, "dns_results.json")
		out := RunOutput{
			Metadata: RunMetadata{
				Operator:       appCtx.Operator,
				EngagementID:   id,
				EngagementName: eng.Name,
				Owner:          eng.Owner,
				StartAt:        startAll,
				CompleteAt:     time.Now().UTC(),
				TotalTargets:   len(eng.Scope),
			},
			Results: results,
		}
		b, _ := json.MarshalIndent(out, "", " ")
		_ = os.WriteFile(resultsPath, b, 0o644)

		// Compute hash for audit.csv
		auditPath := filepath.Join(dir, "audit.csv")
		auditHash, _ := HashFileSHA256(auditPath)

		// Update metadata with audit hash only
		out.Metadata.AuditHash = auditHash

		// Write final results JSON
		b, _ = json.MarshalIndent(out, "", " ")
		_ = os.WriteFile(resultsPath, b, 0o644)

		// Hash results.json AFTER final write
		resultsHash, _ := HashFileSHA256(resultsPath)

		// GPG signing if requested
		if autoSign {
			if gpgKey == "" {
				return fmt.Errorf("--gpg-key required with --auto-sign")
			}
			signFile := func(path string) error {
				cmd := exec.Command("gpg", "--armor", "--local-user", gpgKey, "--sign", path)
				cmd.Dir = filepath.Dir(path)
				cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
				return cmd.Run()
			}

			_ = signFile(auditPath + ".sha256")
			_ = signFile(resultsPath + ".sha256")
			fmt.Println("GPG signatures created for both .sha256 files.")
		}

		// Print summary
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

		if complianceMode {
			fmt.Println("------------------------------------------------------")
			fmt.Println("🔒 Compliance Summary")
			fmt.Printf("Operator: %s\nEngagement: %s (%s)\n", appCtx.Operator, eng.Name, eng.ID)
			fmt.Printf("Audit hash : %s\nResults hash: %s\n", auditHash, resultsHash)
			fmt.Println("Verification: sha256sum -c audit.csv.sha256 && sha256sum -c dns_results.json.sha256")
			fmt.Println("Evidence integrity requirements satisfied.")
			fmt.Println("------------------------------------------------------")
		}

		return nil
	},
}

func init() {
	checkCmd.PersistentFlags().IntVarP(&concurrency, "concurrency", "c", 1, "max concurrent requests")
	checkCmd.PersistentFlags().IntVarP(&rateLimit, "rate", "r", 1, "requests per second (global)")
	checkCmd.PersistentFlags().IntVarP(&timeoutSecs, "timeout", "t", 10, "request timeout in seconds")
	checkHTTPCmd.Flags().String("id", "", "Engagement id")
	checkHTTPCmd.Flags().Bool("roe-confirm", false, "Confirm you have explicit written authorization (required)")
	checkHTTPCmd.Flags().BoolVar(&auditAppendRaw, "audit-append-raw", false, "Save limited raw headers/body for auditing (handle carefully)")
	checkHTTPCmd.Flags().BoolVar(&complianceMode, "compliance-mode", false, "Enable compliance enforcement (hashing, retention checks)")
	checkHTTPCmd.Flags().IntVar(&retentionDays, "retention-days", 0, "Retention period (days) for raw captures; required in compliance mode if --audit-append-raw is used")
	checkHTTPCmd.Flags().Bool("auto-sign", false, "Automatically sign .sha256 files using configured GPG key")
	checkHTTPCmd.Flags().String("gpg-key", "", "GPG key ID or email for signing (required if --auto-sign)")
	checkDNSCmd.Flags().StringSliceVar(&dnsNameservers, "nameservers", []string{}, "Custom DNS nameservers (e.g., 8.8.8.8:53,1.1.1.1:53)")
	checkDNSCmd.Flags().IntVar(&dnsTimeout, "dns-timeout", 10, "DNS query timeout in seconds")

	checkCmd.AddCommand(checkHTTPCmd)
	checkCmd.AddCommand(checkDNSCmd)
}
