package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show system information and data directory paths",
	Long: `Display SECA-CLI configuration information including:
  - Data directory locations
  - Configuration file paths
  - Current operator
  - Platform information`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get application context
		appCtx := getAppContext(cmd)

		// Get data directory
		dataDir, err := getDataDir()
		if err != nil {
			return fmt.Errorf("failed to get data directory: %w", err)
		}

		// Get engagements file path
		engagementsPath, err := getEngagementsFilePath()
		if err != nil {
			return fmt.Errorf("failed to get engagements file path: %w", err)
		}

		// Check if files exist
		engagementsExists := "✗ (not created yet)"
		if _, err := os.Stat(engagementsPath); err == nil {
			engagementsExists = "✓ (exists)"
		}

		resultsExists := "✗ (not created yet)"
		if _, err := os.Stat(appCtx.ResultsDir); err == nil {
			resultsExists = "✓ (exists)"
		}

		configFile := "~/.seca-cli.yaml"
		configExists := "✗ (using defaults)"
		homeDir, _ := os.UserHomeDir()
		configPath := homeDir + "/.seca-cli.yaml"
		if _, err := os.Stat(configPath); err == nil {
			configExists = "✓ (exists)"
		}

		// Get output writer (for testing support)
		out := cmd.OutOrStdout()

		// Print information
		fmt.Fprintln(out, "SECA-CLI System Information")
		fmt.Fprintln(out, "============================")
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Platform:          %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Fprintf(out, "Operator:          %s\n", appCtx.Operator)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Data Locations:")
		fmt.Fprintf(out, "  Data Directory:     %s\n", dataDir)
		fmt.Fprintf(out, "  Engagements File:   %s %s\n", engagementsPath, engagementsExists)
		fmt.Fprintf(out, "  Results Directory:  %s %s\n", appCtx.ResultsDir, resultsExists)
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Configuration File:   %s %s\n", configFile, configExists)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "To override data directory, create ~/.seca-cli.yaml with:")
		fmt.Fprintln(out, "  results_dir: /custom/path/to/results")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Documentation:")
		fmt.Fprintln(out, "  README.md                    - Full documentation")
		fmt.Fprintln(out, "  DATA_DIRECTORY_MIGRATION.md  - Migration guide")
		fmt.Fprintln(out, "  COMPLIANCE.md                - Compliance guidelines")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
