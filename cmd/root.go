package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var cfgFile string
var logger *zap.SugaredLogger
var operator string
var resultsDir string

var rootCmd = &cobra.Command{
	Use:   "seca",
	Short: "Authorized engagement management & safe checks (for lawful testing only)",
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
		resultsDir = viper.GetString("results_dir")
		if resultsDir == "" {
			resultsDir = "./results"
		}

		// create results dir if not exists
		if err := os.MkdirAll(resultsDir, 0o755); err != nil {
			return fmt.Errorf("failed to create results directory: %s", err.Error())
		}

		// init logger
		l, _ := zap.NewProduction()
		logger = l.Sugar()

		// ensure operator is set (via flag or env default)
		if operator == "" {
			// fallback to environment variable USER / LOGNAME if provided
			if env := os.Getenv("USER"); env != "" {
				operator = env
			} else if env := os.Getenv("LOGNAME"); env != "" {
				operator = env
			}
		}
		if operator == "" {
			return fmt.Errorf("operator identity is required (use --operator or set USER env)")
		}

		// Make final resultsDir absolute (for clarity in logs)
		if abs, err := filepath.Abs(resultsDir); err == nil {
			resultsDir = abs
		}

		logger.Infof("operator=%s results_dir=%s", operator, resultsDir)

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// config file flag
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.seca-cli.yaml)")

	// operator persistent flag (default from USER env)
	defaultOperator := os.Getenv("USER")

	rootCmd.PersistentFlags().StringVarP(&operator, "operator", "o", defaultOperator, "operator name (or set via USER env)")

	// add subcommands
	rootCmd.AddCommand(engagementCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(versionCmd)
}
