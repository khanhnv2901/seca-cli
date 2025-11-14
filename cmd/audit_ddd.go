package cmd

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	sharedErrors "github.com/khanhnv2901/seca-cli/internal/shared/errors"
	"github.com/spf13/cobra"
)

// auditCmd is the parent command for audit-related operations
var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit trail management and verification",
	Long: `Manage and verify audit trails for engagements.

Audit trails provide an immutable record of all security checks performed,
including timestamps, operators, targets, and results. Each audit trail is
cryptographically hashed to ensure integrity.`,
}

// auditVerifyCmd verifies the integrity of an audit trail
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

		// Verify the engagement exists
		_, err := appCtx.Services.EngagementService.GetEngagement(ctx, engagementID)
		if err != nil {
			if errors.Is(err, sharedErrors.ErrEngagementNotFound) {
				return fmt.Errorf("engagement %s not found", engagementID)
			}
			return fmt.Errorf("failed to get engagement: %w", err)
		}

		// Verify audit trail integrity using AuditService
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

// auditListCmd lists audit trail entries for an engagement
var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audit trail entries for an engagement",
	Long: `Display all audit trail entries for a given engagement.

Shows a tabular view of all security checks performed, including:
- Timestamp
- Operator
- Command
- Target
- Status
- Duration`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		engagementID, _ := cmd.Flags().GetString("id")
		if engagementID == "" {
			return errors.New("--id is required")
		}

		limit, _ := cmd.Flags().GetInt("limit")
		showAll, _ := cmd.Flags().GetBool("all")

		// Get engagement
		eng, err := appCtx.Services.EngagementService.GetEngagement(ctx, engagementID)
		if err != nil {
			if errors.Is(err, sharedErrors.ErrEngagementNotFound) {
				return fmt.Errorf("engagement %s not found", engagementID)
			}
			return fmt.Errorf("failed to get engagement: %w", err)
		}

		// Get audit trail
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

		// Determine how many entries to show
		entriesToShow := entries
		if !showAll && limit > 0 && len(entries) > limit {
			entriesToShow = entries[len(entries)-limit:]
			fmt.Printf("Showing last %d entries (use --all to show all %d entries)\n\n", limit, len(entries))
		}

		// Create table writer
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Timestamp\tOperator\tCommand\tTarget\tStatus\tDuration")
		fmt.Fprintln(w, "---------\t--------\t-------\t------\t------\t--------")

		for _, entry := range entriesToShow {
			timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
			duration := fmt.Sprintf("%.2fs", entry.DurationSeconds)

			// Color code the status
			statusStr := entry.Status
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

// auditShowCmd shows detailed information about a specific audit entry
var auditShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show detailed audit trail information",
	Long:  `Display detailed information about an engagement's audit trail, including metadata and statistics.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		engagementID, _ := cmd.Flags().GetString("id")
		if engagementID == "" {
			return errors.New("--id is required")
		}

		// Get engagement
		eng, err := appCtx.Services.EngagementService.GetEngagement(ctx, engagementID)
		if err != nil {
			if errors.Is(err, sharedErrors.ErrEngagementNotFound) {
				return fmt.Errorf("engagement %s not found", engagementID)
			}
			return fmt.Errorf("failed to get engagement: %w", err)
		}

		// Get audit trail
		auditTrail, err := appCtx.Services.AuditService.GetAuditTrail(ctx, engagementID)
		if err != nil {
			if errors.Is(err, sharedErrors.ErrAuditTrailNotFound) {
				fmt.Printf("No audit trail found for engagement: %s\n", eng.Name())
				return nil
			}
			return fmt.Errorf("failed to get audit trail: %w", err)
		}

		entries := auditTrail.Entries()

		// Print header
		fmt.Printf("\n%s Audit Trail Details\n", colorInfo("═══"))
		fmt.Println(strings.Repeat("═", 60))
		fmt.Println()

		// Engagement info
		fmt.Printf("%s: %s\n", colorInfo("Engagement"), eng.Name())
		fmt.Printf("%s: %s\n", colorInfo("ID"), eng.ID())
		fmt.Printf("%s: %s\n", colorInfo("Owner"), eng.Owner())
		fmt.Println()

		// Audit trail metadata
		fmt.Printf("%s: %d\n", colorInfo("Total Entries"), len(entries))
		fmt.Printf("%s: %s\n", colorInfo("Created"), auditTrail.CreatedAt().Format(time.RFC3339))

		if auditTrail.IsSealed() {
			fmt.Printf("%s: %s\n", colorSuccess("Status"), "Sealed")
			fmt.Printf("%s: %s\n", colorInfo("Hash Algorithm"), auditTrail.HashAlgorithm())
			fmt.Printf("%s: %s\n", colorInfo("Hash"), auditTrail.Hash())
		} else {
			fmt.Printf("%s: %s\n", colorWarn("Status"), "Unsealed")
		}

		if auditTrail.IsSigned() {
			fmt.Printf("%s: %s\n", colorSuccess("GPG Signed"), "Yes")
		}

		// Statistics
		if len(entries) > 0 {
			fmt.Println()
			fmt.Printf("%s\n", colorInfo("Statistics:"))

			okCount := 0
			errorCount := 0
			var totalDuration float64
			operators := make(map[string]int)
			commands := make(map[string]int)

			for _, entry := range entries {
				if entry.Status == "ok" {
					okCount++
				} else {
					errorCount++
				}
				totalDuration += entry.DurationSeconds
				operators[entry.Operator]++
				commands[entry.Command]++
			}

			fmt.Printf("  Success: %s | Errors: %s\n",
				colorSuccess(fmt.Sprintf("%d", okCount)),
				colorError(fmt.Sprintf("%d", errorCount)))
			fmt.Printf("  Total Duration: %.2fs\n", totalDuration)
			fmt.Printf("  Average Duration: %.2fs\n", totalDuration/float64(len(entries)))

			fmt.Printf("  Operators: %v\n", operators)
			fmt.Printf("  Commands: %v\n", commands)
		}

		// File location
		fmt.Println()
		auditPath := filepath.Join(appCtx.ResultsDir, engagementID, "audit.csv")
		fmt.Printf("%s: %s\n", colorInfo("File"), auditPath)

		fmt.Println()
		fmt.Println(strings.Repeat("═", 60))

		return nil
	},
}

// auditExportCmd exports audit trail to different formats
var auditExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export audit trail to different formats",
	Long:  `Export the audit trail to JSON, CSV, or other formats for external analysis.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		engagementID, _ := cmd.Flags().GetString("id")
		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")

		if engagementID == "" {
			return errors.New("--id is required")
		}

		// Get audit trail
		auditTrail, err := appCtx.Services.AuditService.GetAuditTrail(ctx, engagementID)
		if err != nil {
			if errors.Is(err, sharedErrors.ErrAuditTrailNotFound) {
				return fmt.Errorf("no audit trail found for engagement %s", engagementID)
			}
			return fmt.Errorf("failed to get audit trail: %w", err)
		}

		// Default output to stdout
		var outFile *os.File
		if output == "" || output == "-" {
			outFile = os.Stdout
		} else {
			f, err := os.Create(output)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer f.Close()
			outFile = f
		}

		switch format {
		case "csv":
			writer := csv.NewWriter(outFile)
			defer writer.Flush()

			// Write header
			writer.Write([]string{"timestamp", "engagement_id", "operator", "command", "target", "status", "http_status", "tls_expiry", "notes", "error", "duration_seconds"})

			// Write entries
			for _, entry := range auditTrail.Entries() {
				tlsExpiry := ""
				if !entry.TLSExpiry.IsZero() {
					tlsExpiry = entry.TLSExpiry.Format(time.RFC3339)
				}

				writer.Write([]string{
					entry.Timestamp.Format(time.RFC3339),
					entry.EngagementID,
					entry.Operator,
					entry.Command,
					entry.Target,
					entry.Status,
					fmt.Sprintf("%d", entry.HTTPStatus),
					tlsExpiry,
					entry.Notes,
					entry.Error,
					fmt.Sprintf("%.3f", entry.DurationSeconds),
				})
			}

		case "json":
			// Simple JSON export (could be enhanced with proper encoding/json)
			fmt.Fprintln(outFile, "[")
			entries := auditTrail.Entries()
			for i, entry := range entries {
				fmt.Fprintf(outFile, `  {
    "timestamp": "%s",
    "engagement_id": "%s",
    "operator": "%s",
    "command": "%s",
    "target": "%s",
    "status": "%s",
    "http_status": %d,
    "notes": "%s",
    "error": "%s",
    "duration_seconds": %.3f
  }`,
					entry.Timestamp.Format(time.RFC3339),
					entry.EngagementID,
					entry.Operator,
					entry.Command,
					entry.Target,
					entry.Status,
					entry.HTTPStatus,
					entry.Notes,
					entry.Error,
					entry.DurationSeconds,
				)
				if i < len(entries)-1 {
					fmt.Fprintln(outFile, ",")
				} else {
					fmt.Fprintln(outFile)
				}
			}
			fmt.Fprintln(outFile, "]")

		default:
			return fmt.Errorf("unsupported format: %s (supported: csv, json)", format)
		}

		if output != "" && output != "-" {
			fmt.Printf("%s Audit trail exported to: %s\n", colorSuccess("✓"), output)
		}

		return nil
	},
}

func init() {
	// Verify command flags
	auditVerifyCmd.Flags().String("id", "", "Engagement ID")

	// List command flags
	auditListCmd.Flags().String("id", "", "Engagement ID")
	auditListCmd.Flags().Int("limit", 20, "Number of recent entries to show (0 for all)")
	auditListCmd.Flags().Bool("all", false, "Show all entries")

	// Show command flags
	auditShowCmd.Flags().String("id", "", "Engagement ID")

	// Export command flags
	auditExportCmd.Flags().String("id", "", "Engagement ID")
	auditExportCmd.Flags().String("format", "json", "Export format (json, csv)")
	auditExportCmd.Flags().String("output", "", "Output file (default: stdout)")

	// Add subcommands to audit command
	auditCmd.AddCommand(auditVerifyCmd)
	auditCmd.AddCommand(auditListCmd)
	auditCmd.AddCommand(auditShowCmd)
	auditCmd.AddCommand(auditExportCmd)
}
