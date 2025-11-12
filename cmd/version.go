package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version information (injected at build time via -ldflags)
// These default values indicate a development build
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Display detailed version information for SECA-CLI",
	Run: func(cmd *cobra.Command, args []string) {
		verbose, _ := cmd.Flags().GetBool("verbose")

		if verbose {
			fmt.Printf(`SECA-CLI Version Information:
  Version:    %s
  Git Commit: %s
  Build Date: %s
  Go Version: %s
  OS/Arch:    %s/%s
  Compiler:   %s
`, Version, GitCommit, BuildDate, runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.Compiler)
		} else {
			fmt.Printf("SECA-CLI version %s\n", Version)
		}
	},
}

func init() {
	versionCmd.Flags().BoolP("verbose", "v", false, "Show detailed version information")
}
