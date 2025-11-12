package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/khanhnv2901/seca-cli/internal/checker"
	"github.com/spf13/cobra"
)

type checkerPluginDefinition struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Command         string            `json:"command"`
	Args            []string          `json:"args"`
	Env             map[string]string `json:"env"`
	TimeoutSeconds  int               `json:"timeout"`
	ResultsFilename string            `json:"results_filename"`
	APIVersion      int               `json:"api_version"`
}

const currentPluginAPIVersion = 1

func registerPluginCommands() {
	defs, err := loadCheckerPlugins()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: unable to load plugins: %v\n", err)
		return
	}

	for _, def := range defs {
		if err := addPluginCommand(def); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping plugin %s: %v\n", def.Name, err)
		}
	}
}

func loadCheckerPlugins() ([]checkerPluginDefinition, error) {
	dataDir, err := getDataDir()
	if err != nil {
		return nil, err
	}

	pluginsDir := filepath.Join(dataDir, "plugins")
	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return nil, err
	}

	defs := make([]checkerPluginDefinition, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(pluginsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read plugin %s: %v\n", entry.Name(), err)
			continue
		}

		var def checkerPluginDefinition
		if err := json.Unmarshal(data, &def); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse plugin %s: %v\n", entry.Name(), err)
			continue
		}

		if def.APIVersion == 0 {
			def.APIVersion = currentPluginAPIVersion
		}

		if def.APIVersion != currentPluginAPIVersion {
			fmt.Fprintf(os.Stderr, "Warning: unsupported plugin API version %d in %s (expected %d)\n", def.APIVersion, entry.Name(), currentPluginAPIVersion)
			continue
		}

		if def.Name == "" || def.Command == "" {
			fmt.Fprintf(os.Stderr, "Warning: invalid plugin %s (name and command required)\n", entry.Name())
			continue
		}

		if def.TimeoutSeconds <= 0 {
			def.TimeoutSeconds = 10
		}

		if def.ResultsFilename == "" {
			def.ResultsFilename = fmt.Sprintf("%s_results.json", def.Name)
		}

		defs = append(defs, def)
	}

	return defs, nil
}

func addPluginCommand(def checkerPluginDefinition) error {
	cmd := &cobra.Command{
		Use:   def.Name,
		Short: def.Description,
		RunE: func(c *cobra.Command, args []string) error {
			return runCheckCommand(c, checkConfig{
				CreateChecker: func(appCtx *AppContext, params checkParams) checker.Checker {
					return checker.NewExternalChecker(checker.ExternalCheckerConfig{
						Name:           def.Name,
						Command:        def.Command,
						Args:           def.Args,
						Env:            def.Env,
						TimeoutSeconds: def.TimeoutSeconds,
					})
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
				ResultsFilename:    def.ResultsFilename,
				TimeoutSecs:        def.TimeoutSeconds,
				VerificationCmd:    fmt.Sprintf("sha256sum -c audit.csv.sha256 && sha256sum -c %s.sha256", def.ResultsFilename),
				SupportsRawCapture: false,
				PrintSummary: func(results []checker.CheckResult, resultsPath, auditPath, auditHash, resultsHash string) {
					fmt.Printf("%s plugin run complete.\n", def.Name)
					fmt.Printf("Results: %s\nAudit: %s\n", resultsPath, auditPath)
					fmt.Printf("SHA256 audit: %s\nSHA256 results: %s\n", auditHash, resultsHash)
				},
			})
		},
	}

	addCommonCheckFlags(cmd)
	checkCmd.AddCommand(cmd)
	return nil
}
