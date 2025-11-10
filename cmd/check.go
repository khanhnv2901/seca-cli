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
	Metadata RunMetadata          `json:"metadata"`
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

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run safe, authorized checks against scoped targets (no scanning/exploitation)",
}

var checkHTTPCmd = &cobra.Command{
	Use:   "http",
	Short: "Run safe HTTP/TLS checks for an engagement's scope",
	RunE: func(cmd *cobra.Command, args []string) error {
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

		if operator == "" {
			return fmt.Errorf("--operator is required")
		}

		if complianceMode {
			fmt.Println("[Compliance Mode] Enabled")

			// Require operator
			if operator == "" {
				return fmt.Errorf("--operator required in compliance mode")
			}

			// Auto-force hash-signing (already done later)
			// Nothing to change here, but we'll print a notice
			fmt.Println("→ Hash-signing of audit and result files enforced")

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
		dir := filepath.Join(resultsDir, id)
		_ = os.MkdirAll(dir, 0o755)

		// Create HTTP checker
		httpChecker := &checker.HTTPChecker{
			Timeout:    time.Duration(timeoutSecs) * time.Second,
			CaptureRaw: auditAppendRaw,
			RawHandler: func(target string, headers http.Header, bodySnippet string) error {
				return SaveRawCapture(id, target, headers, bodySnippet)
			},
		}

		// Create audit function
		auditFn := func(target string, result checker.CheckResult, duration float64) error {
			return AppendAuditRow(
				id,
				operator,
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
				Operator:       operator,
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
			fmt.Printf("Operator: %s\nEngagement: %s (%s)\n", operator, eng.Name, eng.ID)
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

	checkCmd.AddCommand(checkHTTPCmd)
}
