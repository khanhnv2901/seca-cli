package cmd

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	consts "github.com/khanhnv2901/seca-cli/internal/shared/constants"
	sharedErrors "github.com/khanhnv2901/seca-cli/internal/shared/errors"
	"github.com/spf13/cobra"
)

// audit header fields:
var auditHeader = []string{
	"timestamp",
	"engagement_id",
	"operator",
	"command",
	"target",
	"status",
	"http_status",
	"tls_expiry",
	"notes",
	"error",
	"duration_seconds",
}

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit trail management and verification",
	Long: `Manage and verify audit trails for engagements.

Audit trails provide an immutable record of all security checks performed,
including timestamps, operators, targets, and results. Each audit trail is
cryptographically hashed to ensure integrity.`,
}

var auditVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify audit trail integrity using cryptographic hash",
	Long: `Verify that an audit trail has not been tampered with by checking its cryptographic hash.

The audit trail is hashed using SHA256 or SHA512, and the hash is stored in a companion file.
This command recomputes the hash and compares it with the stored value.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		engagementID, _ := cmd.Flags().GetString("id")
		if engagementID == "" {
			return errors.New("--id is required")
		}

		_, err := appCtx.Services.EngagementService.GetEngagement(ctx, engagementID)
		if err != nil {
			if errors.Is(err, sharedErrors.ErrEngagementNotFound) {
				return fmt.Errorf("engagement %s not found", engagementID)
			}
			return fmt.Errorf("failed to get engagement: %w", err)
		}

		valid, err := appCtx.Services.AuditService.VerifyIntegrity(ctx, engagementID)
		if err != nil {
			if errors.Is(err, sharedErrors.ErrAuditTrailNotFound) {
				return fmt.Errorf("no audit trail found for engagement %s", engagementID)
			}
			return fmt.Errorf("failed to verify audit trail: %w", err)
		}

		auditPath := filepath.Join(appCtx.ResultsDir, engagementID, "audit.csv")

		if valid {
			fmt.Printf("%s Audit trail integrity verified: %s\n", colorSuccess("✓"), auditPath)
			fmt.Printf("%s The audit trail has NOT been tampered with\n", colorSuccess("✓"))
		} else {
			fmt.Printf("%s Audit trail integrity verification FAILED: %s\n", colorError("✗"), auditPath)
			fmt.Printf("%s WARNING: The audit trail may have been tampered with!\n", colorError("✗"))
			return fmt.Errorf("audit trail integrity check failed")
		}

		return nil
	},
}

var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audit trail entries for an engagement",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		engagementID, _ := cmd.Flags().GetString("id")
		if engagementID == "" {
			return errors.New("--id is required")
		}

		limit, _ := cmd.Flags().GetInt("limit")
		showAll, _ := cmd.Flags().GetBool("all")

		eng, err := appCtx.Services.EngagementService.GetEngagement(ctx, engagementID)
		if err != nil {
			if errors.Is(err, sharedErrors.ErrEngagementNotFound) {
				return fmt.Errorf("engagement %s not found", engagementID)
			}
			return fmt.Errorf("failed to get engagement: %w", err)
		}

		auditTrail, err := appCtx.Services.AuditService.GetAuditTrail(ctx, engagementID)
		if err != nil {
			if errors.Is(err, sharedErrors.ErrAuditTrailNotFound) {
				fmt.Printf("No audit trail found for engagement: %s\n", eng.Name())
				return nil
			}
			return fmt.Errorf("failed to get audit trail: %w", err)
		}

		entries := auditTrail.Entries()
		if len(entries) == 0 {
			fmt.Printf("No audit entries found for engagement: %s\n", eng.Name())
			return nil
		}

		fmt.Printf("Audit Trail for Engagement: %s\n", colorInfo(eng.Name()))
		fmt.Printf("Engagement ID: %s\n", eng.ID())
		fmt.Printf("Total Entries: %d\n", len(entries))

		if auditTrail.IsSealed() {
			fmt.Printf("Status: %s (Hash: %s)\n", colorSuccess("Sealed"), auditTrail.HashAlgorithm())
		} else {
			fmt.Printf("Status: %s\n", colorWarn("Unsealed"))
		}

		if auditTrail.IsSigned() {
			fmt.Printf("Signed: %s\n", colorSuccess("Yes"))
		}

		fmt.Println()

		entriesToShow := entries
		if !showAll && limit > 0 && len(entries) > limit {
			entriesToShow = entries[len(entries)-limit:]
			fmt.Printf("Showing last %d entries (use --all to show all %d entries)\n\n", limit, len(entries))
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Timestamp\tOperator\tCommand\tTarget\tStatus\tDuration")
		fmt.Fprintln(w, "---------\t--------\t-------\t------\t------\t--------")

		for _, entry := range entriesToShow {
			timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
			duration := fmt.Sprintf("%.2fs", entry.DurationSeconds)
			var statusStr string
			if entry.Status == "ok" {
				statusStr = colorSuccess(entry.Status)
			} else {
				statusStr = colorError(entry.Status)
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				timestamp,
				entry.Operator,
				entry.Command,
				entry.Target,
				statusStr,
				duration,
			)
		}

		w.Flush()

		return nil
	},
}

var auditShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show detailed audit trail information",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		engagementID, _ := cmd.Flags().GetString("id")
		if engagementID == "" {
			return errors.New("--id is required")
		}

		eng, err := appCtx.Services.EngagementService.GetEngagement(ctx, engagementID)
		if err != nil {
			if errors.Is(err, sharedErrors.ErrEngagementNotFound) {
				return fmt.Errorf("engagement %s not found", engagementID)
			}
			return fmt.Errorf("failed to get engagement: %w", err)
		}

		auditTrail, err := appCtx.Services.AuditService.GetAuditTrail(ctx, engagementID)
		if err != nil {
			if errors.Is(err, sharedErrors.ErrAuditTrailNotFound) {
				fmt.Printf("No audit trail found for engagement: %s\n", eng.Name())
				return nil
			}
			return fmt.Errorf("failed to get audit trail: %w", err)
		}

		entries := auditTrail.Entries()

		fmt.Printf("Audit Trail Summary for %s (%s)\n", eng.Name(), eng.ID())
		fmt.Printf("Total Entries: %d\n", len(entries))
		if auditTrail.IsSealed() {
			fmt.Printf("Status: %s (Hash: %s)\n", colorSuccess("Sealed"), auditTrail.HashAlgorithm())
		} else {
			fmt.Printf("Status: %s\n", colorWarn("Unsealed"))
		}
		fmt.Println()

		latestEntries := 5
		if len(entries) < latestEntries {
			latestEntries = len(entries)
		}

		fmt.Println("Most Recent Entries:")
		for i := len(entries) - latestEntries; i < len(entries); i++ {
			entry := entries[i]
			fmt.Printf("%s | %s | %s | %s | %s\n",
				entry.Timestamp.Format("2006-01-02 15:04:05"),
				entry.Operator,
				entry.Command,
				entry.Target,
				entry.Status,
			)
		}

		return nil
	},
}

var auditExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export audit trail to CSV/JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		engagementID, _ := cmd.Flags().GetString("id")
		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")

		if engagementID == "" {
			return errors.New("--id is required")
		}

		format = strings.ToLower(format)
		if format != "csv" && format != "json" {
			return fmt.Errorf("unsupported format %s (use csv or json)", format)
		}

		auditTrail, err := appCtx.Services.AuditService.GetAuditTrail(ctx, engagementID)
		if err != nil {
			return fmt.Errorf("failed to get audit trail: %w", err)
		}

		entries := auditTrail.Entries()
		if len(entries) == 0 {
			return fmt.Errorf("no audit entries found for engagement %s", engagementID)
		}

		var data []byte
		if format == "json" {
			data, err = json.MarshalIndent(entries, jsonPrefix, jsonIndent)
			if err != nil {
				return fmt.Errorf("failed to marshal json: %w", err)
			}
		} else {
			var b strings.Builder
			w := csv.NewWriter(&b)
			_ = w.Write(auditHeader)
			for _, entry := range entries {
				row := []string{
					entry.Timestamp.Format(time.RFC3339),
					entry.EngagementID,
					entry.Operator,
					entry.Command,
					entry.Target,
					entry.Status,
					fmt.Sprintf("%d", entry.HTTPStatus),
					entry.TLSExpiry.Format(time.RFC3339),
					entry.Notes,
					entry.Error,
					fmt.Sprintf("%.3f", entry.DurationSeconds),
				}
				_ = w.Write(row)
			}
			w.Flush()
			data = []byte(b.String())
		}

		outputPath := output
		if outputPath == "" {
			filename := fmt.Sprintf("audit_%s.%s", engagementID, format)
			resolved, err := resolveResultsPath(appCtx.ResultsDir, engagementID, filename)
			if err != nil {
				return fmt.Errorf("resolve output path: %w", err)
			}
			outputPath = resolved
		}

		if err := os.WriteFile(outputPath, data, consts.DefaultFilePerm); err != nil {
			return fmt.Errorf("failed to write export: %w", err)
		}

		fmt.Printf("Audit trail exported to %s\n", outputPath)
		return nil
	},
}

// AppendAuditRow appends a single audit row to results/<engagementID>/audit.csv
func AppendAuditRow(resultsDir string, engagementID string, operatorName string, commandName string, target string, status string, httpStatus int, tlsExpiry string, notes string, errMsg string, durationSeconds float64) error {
	// ensure engagement-specific directory under resultsDir
	dir, err := ensureResultsDir(resultsDir, engagementID)
	if err != nil {
		return fmt.Errorf("create results subdir failed: %w", err)
	}

	auditPath := filepath.Join(dir, "audit.csv")
	// check if file exists
	exists := true
	if _, err := os.Stat(auditPath); os.IsNotExist(err) {
		exists = false
	}

	f, err := os.OpenFile(auditPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, consts.DefaultFilePerm)
	if err != nil {
		return fmt.Errorf("open audit file failed: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// if new file, write header first
	if !exists {
		if err := writer.Write(auditHeader); err != nil {
			return fmt.Errorf("write audit header failed: %w", err)
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return fmt.Errorf("flush audit header failed: %w", err)
		}
	}

	row := []string{
		time.Now().UTC().Format(time.RFC3339),
		engagementID,
		operatorName,
		commandName,
		target,
		status,
		fmt.Sprintf("%d", httpStatus),
		tlsExpiry,
		notes,
		errMsg,
		fmt.Sprintf("%.3f", durationSeconds),
	}

	if err := writer.Write(row); err != nil {
		return fmt.Errorf("write audit row failed: %w", err)
	}
	writer.Flush()

	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush audit data failed: %w", err)
	}

	return nil
}

// SaveRawCapture writes a limited raw HTTP response for auditing (be careful with PII)
func SaveRawCapture(resultsDir string, engamentID, target string, headers map[string][]string, bodySnippet string) error {
	dir, err := ensureResultsDir(resultsDir, engamentID)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("raw_%d.txt", time.Now().UnixNano())
	path := filepath.Join(dir, filename)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "Target: %s\nCaptureAt: %s\n\nHeaders:\n", target, time.Now().UTC().Format(time.RFC3339))
	for k, v := range headers {
		fmt.Fprintf(f, "%s: %s\n", k, v)
	}
	fmt.Fprintf(f, "\n--- Body Snippet (max %d bytes) ---\n%s\n", consts.RawCaptureLimitBytes, bodySnippet)
	return nil
}

// HashFileSHA256 computes and writes a .sha256 companion file
func HashFileSHA256(path string) (string, error) {
	return HashFile(path, HashAlgorithmSHA256)
}

// HashFile computes and writes a companion file for the given algorithm.
func HashFile(path string, algorithm HashAlgorithm) (string, error) {
	hasher, err := algorithm.newHasher()
	if err != nil {
		return "", err
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}

	sum := hex.EncodeToString(hasher.Sum(nil))
	hashPath := path + algorithm.FileExtension()
	content := fmt.Sprintf("%s  %s\n", sum, filepath.Base(path))
	if err := os.WriteFile(hashPath, []byte(content), consts.DefaultFilePerm); err != nil {
		return "", err
	}
	return sum, nil
}

// HashAlgorithm represents supported hashing algorithms for integrity files.
type HashAlgorithm string

const (
	HashAlgorithmSHA256 HashAlgorithm = "sha256"
	HashAlgorithmSHA512 HashAlgorithm = "sha512"
)

// ParseHashAlgorithm normalizes and validates the requested algorithm.
func ParseHashAlgorithm(raw string) (HashAlgorithm, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "sha256", "":
		return HashAlgorithmSHA256, nil
	case "sha512":
		return HashAlgorithmSHA512, nil
	default:
		return "", fmt.Errorf("unsupported hash algorithm %q (use sha256 or sha512)", raw)
	}
}

func (h HashAlgorithm) String() string {
	if h == "" {
		return string(HashAlgorithmSHA256)
	}
	return string(h)
}

func (h HashAlgorithm) DisplayName() string {
	return strings.ToUpper(h.String())
}

func (h HashAlgorithm) FileExtension() string {
	return "." + h.String()
}

func (h HashAlgorithm) SumCommand() string {
	return fmt.Sprintf("%ssum", h.String())
}

func (h HashAlgorithm) newHasher() (hash.Hash, error) {
	switch h {
	case HashAlgorithmSHA256, "":
		return sha256.New(), nil
	case HashAlgorithmSHA512:
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm %q", h)
	}
}

func init() {
	auditCmd.AddCommand(auditVerifyCmd)
	auditCmd.AddCommand(auditListCmd)
	auditCmd.AddCommand(auditShowCmd)
	auditCmd.AddCommand(auditExportCmd)

	auditVerifyCmd.Flags().String("id", "", "Engagement ID")
	auditListCmd.Flags().String("id", "", "Engagement ID")
	auditListCmd.Flags().Int("limit", 20, "Number of entries to show")
	auditListCmd.Flags().Bool("all", false, "Show all entries")
	auditShowCmd.Flags().String("id", "", "Engagement ID")
	auditExportCmd.Flags().String("id", "", "Engagement ID")
	auditExportCmd.Flags().String("format", "json", "Export format (json|csv)")
	auditExportCmd.Flags().String("output", "", "Output path (defaults to engagement results dir)")
}
