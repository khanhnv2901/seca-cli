package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/domain/audit"
	"github.com/khanhnv2901/seca-cli/internal/infrastructure/checker"
	sharedErrors "github.com/khanhnv2901/seca-cli/internal/shared/errors"
	"github.com/khanhnv2901/seca-cli/internal/shared/security"
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
		path, err := security.ResolveWithin(pluginsDir, entry.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid plugin path %s: %v\n", entry.Name(), err)
			continue
		}
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			appCtx := getAppContext(c)
			runtimeCfg := appCtx.Config.Check
			startTime := time.Now()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(sigCh)

			go func() {
				select {
				case sig := <-sigCh:
					fmt.Printf("\n%s Received %s, finalizing partial results...\n", colorWarn("!"), sig.String())
					cancel()
				case <-ctx.Done():
				}
			}()

			engagementID := c.Flag("id").Value.String()
			roeConfirm := c.Flag("roe-confirm").Value.String() == "true"

			if engagementID == "" {
				return errors.New("--id is required")
			}

			if !roeConfirm {
				return errors.New("must pass --roe-confirm to run checks")
			}

			eng, err := appCtx.Services.EngagementService.GetEngagement(ctx, engagementID)
			if err != nil {
				if errors.Is(err, sharedErrors.ErrEngagementNotFound) {
					return fmt.Errorf("engagement %s not found", engagementID)
				}
				return fmt.Errorf("failed to get engagement: %w", err)
			}

			if err := appCtx.Services.EngagementService.ValidateEngagementForChecks(ctx, engagementID, ""); err != nil {
				return fmt.Errorf("engagement validation failed: %w", err)
			}

			checkRun, err := appCtx.Services.CheckOrchestrator.CreateCheckRun(ctx, engagementID, appCtx.Operator)
			if err != nil {
				return fmt.Errorf("failed to create check run: %w", err)
			}

			fmt.Printf("%s Starting plugin %s for engagement: %s\n", colorInfo("→"), def.Name, eng.Name())
			fmt.Printf("%s Targets: %d\n", colorInfo("→"), len(eng.Scope()))
			fmt.Println()

			externalChecker := checker.NewExternalChecker(checker.ExternalCheckerConfig{
				Name:           def.Name,
				Command:        def.Command,
				Args:           def.Args,
				Env:            def.Env,
				TimeoutSeconds: def.TimeoutSeconds,
			})

			timeout := time.Duration(def.TimeoutSeconds) * time.Second
			if timeout <= 0 {
				timeout = time.Duration(runtimeCfg.TimeoutSecs) * time.Second
			}

			runner := &checker.Runner{
				Concurrency: runtimeCfg.Concurrency,
				RateLimit:   runtimeCfg.RateLimit,
				Timeout:     timeout,
			}

			baseTargets := append([]string(nil), eng.Scope()...)
			targets := expandTargetsWithCrawl(ctx, baseTargets, runtimeCfg)

			var progress *progressPrinter
			if runtimeCfg.ProgressEnabled {
				progress = newProgressPrinter(len(targets), externalChecker.Name())
				progress.Start()
			}

			adapter := &resultAdapter{}

			auditFn := func(target string, checkerResult checker.CheckResult, duration float64) error {
				entry := &audit.Entry{
					Timestamp:       time.Now(),
					EngagementID:    engagementID,
					Operator:        appCtx.Operator,
					Command:         fmt.Sprintf("plugin %s", def.Name),
					Target:          target,
					Status:          checkerResult.Status,
					HTTPStatus:      checkerResult.HTTPStatus,
					Notes:           checkerResult.Notes,
					Error:           checkerResult.Error,
					DurationSeconds: duration,
				}

				if checkerResult.TLSExpiry != "" {
					if expiry, err := time.Parse(time.RFC3339, checkerResult.TLSExpiry); err == nil {
						entry.TLSExpiry = expiry
					}
				}

				if err := appCtx.Services.CheckOrchestrator.RecordAuditEntry(ctx, entry); err != nil {
					return fmt.Errorf("failed to record audit: %w", err)
				}

				domainResult, err := adapter.toDomain(target, checkerResult)
				if err != nil {
					return fmt.Errorf("failed to convert result: %w", err)
				}

				if err := appCtx.Services.CheckOrchestrator.AddCheckResult(ctx, checkRun, domainResult); err != nil {
					return fmt.Errorf("failed to add result: %w", err)
				}

				if progress != nil {
					progress.Increment(checkerResult.Status == "ok", duration)
				}

				return nil
			}

			results := runner.RunChecks(ctx, targets, externalChecker, auditFn)

			if progress != nil {
				progress.Stop()
			}

			runDuration := time.Since(startTime)
			if runtimeCfg.TelemetryEnabled {
				if err := recordTelemetry(appCtx, engagementID, externalChecker.Name(), results, runDuration); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to record telemetry: %v\n", err)
				}
			}

			fmt.Printf("\n%s Plugin %s run complete (%d target(s))\n", colorSuccess("✓"), def.Name, len(results))

			hashAlgo := runtimeCfg.HashAlgorithm
			if hashAlgo == "" {
				hashAlgo = "sha256"
			}

			auditHash, err := appCtx.Services.CheckOrchestrator.SealAuditTrail(ctx, engagementID, hashAlgo)
			if err != nil {
				return fmt.Errorf("failed to seal audit trail: %w", err)
			}

			if err := appCtx.Services.CheckOrchestrator.FinalizeCheckRun(ctx, checkRun, auditHash, hashAlgo); err != nil {
				return fmt.Errorf("failed to finalize check run: %w", err)
			}

			resultsPath := filepath.Join(appCtx.ResultsDir, engagementID, "http_results.json")
			auditPath := filepath.Join(appCtx.ResultsDir, engagementID, "audit.csv")

			fmt.Println()
			fmt.Printf("%s Results: %s\n", colorSuccess("→"), resultsPath)
			fmt.Printf("%s Audit: %s\n", colorSuccess("→"), auditPath)
			fmt.Printf("%s Audit hash (%s): %s\n", colorSuccess("→"), hashAlgo, auditHash)

			return nil
		},
	}

	cmd.Flags().String("id", "", "Engagement ID")
	cmd.Flags().Bool("roe-confirm", false, "Confirm ROE and authorization")
	checkCmd.AddCommand(cmd)
	return nil
}
