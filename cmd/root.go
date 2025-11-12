package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// AppContext holds application-wide dependencies and configuration.
// This struct is passed to command handlers to avoid global state and improve testability.
type AppContext struct {
	Logger     *zap.SugaredLogger
	Operator   string
	ResultsDir string
}

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "seca",
	Short: "Authorized engagement management & safe checks (for lawful testing only)",
	Long: `SECA-CLI - Secure Engagement & Compliance Auditing CLI

A professional command-line tool for managing authorized security testing engagements
with built-in compliance, evidence integrity, and audit trail capabilities.

Data Storage:
  Linux/Unix:  ~/.local/share/seca-cli/
  macOS:       ~/Library/Application Support/seca-cli/
  Windows:     %LOCALAPPDATA%\seca-cli\

You can override the data directory in ~/.seca-cli.yaml with:
  results_dir: /custom/path/to/results

Documentation:
  README.md                    - Full documentation
  DATA_DIRECTORY_MIGRATION.md  - Data migration guide
  COMPLIANCE.md                - Compliance guidelines
`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// init config
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		} else {
			viper.AddConfigPath("$HOME")
			viper.SetConfigName(".seca-cli")
			viper.SetConfigType("yaml")
		}

		_ = viper.ReadInConfig()

		// Initialize AppContext
		appCtx := &AppContext{}

		// Set results directory
		appCtx.ResultsDir = viper.GetString("results_dir")
		if appCtx.ResultsDir == "" {
			// Use proper data directory by default
			dataDir, err := getResultsDir()
			if err != nil {
				// Fallback to old behavior if data directory fails
				fmt.Fprintf(os.Stderr, "Warning: Could not get data directory: %v\n", err)
				fmt.Fprintf(os.Stderr, "Falling back to ./results\n")
				appCtx.ResultsDir = "./results"
			} else {
				appCtx.ResultsDir = dataDir
			}
		}

		// create results dir if not exists
		if err := os.MkdirAll(appCtx.ResultsDir, 0o755); err != nil {
			return fmt.Errorf("failed to create results directory: %s", err.Error())
		}

		// init logger
		l, _ := zap.NewProduction()
		appCtx.Logger = l.Sugar()

		// Get operator from flag
		operatorFlag, _ := cmd.Flags().GetString("operator")
		appCtx.Operator = operatorFlag

		// ensure operator is set (via flag or env default)
		if appCtx.Operator == "" {
			// fallback to environment variable USER / LOGNAME if provided
			if env := os.Getenv("USER"); env != "" {
				appCtx.Operator = env
			} else if env := os.Getenv("LOGNAME"); env != "" {
				appCtx.Operator = env
			}
		}
		if appCtx.Operator == "" {
			return fmt.Errorf("operator identity is required (use --operator or set USER env)")
		}

		// Make final resultsDir absolute (for clarity in logs)
		if abs, err := filepath.Abs(appCtx.ResultsDir); err == nil {
			appCtx.ResultsDir = abs
		}

		appCtx.Logger.Infof("operator=%s results_dir=%s", appCtx.Operator, appCtx.ResultsDir)

		// Store AppContext in command context for access by subcommands
		cmd.SetContext(cmd.Context())
		// Store in command's context using a custom field
		// We'll use a helper function to retrieve it
		storeAppContext(cmd, appCtx)

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// storeAppContext stores the AppContext in the command for access by subcommands.
// This uses a simple approach of storing in the command's annotations.
func storeAppContext(cmd *cobra.Command, appCtx *AppContext) {
	// Store the context in the root command so all subcommands can access it
	root := cmd.Root()
	if root.Annotations == nil {
		root.Annotations = make(map[string]string)
	}
	// We'll use a package-level variable as the simplest approach for cobra
	// since cobra doesn't have built-in context passing between parent and child commands
	globalAppContext = appCtx
}

// getAppContext retrieves the AppContext from the command.
// Returns the AppContext or panics if not initialized (which indicates a bug).
func getAppContext(cmd *cobra.Command) *AppContext {
	if globalAppContext == nil {
		panic("AppContext not initialized - this is a bug")
	}
	return globalAppContext
}

// globalAppContext is a package-level variable to store the app context.
// This is initialized once in PersistentPreRunE and accessed by subcommands.
// While this is still a global, it's much better than having multiple globals
// and makes testing easier since we can set it explicitly in tests.
var globalAppContext *AppContext

func init() {
	// config file flag
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.seca-cli.yaml)")

	// operator persistent flag (default from USER env)
	defaultOperator := os.Getenv("USER")
	rootCmd.PersistentFlags().StringP("operator", "o", defaultOperator, "operator name (or set via USER env)")

	// add subcommands
	rootCmd.AddCommand(engagementCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(versionCmd)
}
