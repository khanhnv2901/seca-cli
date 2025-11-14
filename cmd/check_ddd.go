package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/domain/audit"
	"github.com/khanhnv2901/seca-cli/internal/domain/check"
	"github.com/khanhnv2901/seca-cli/internal/infrastructure/checker"
	sharedErrors "github.com/khanhnv2901/seca-cli/internal/shared/errors"
	"github.com/spf13/cobra"
)

// resultAdapter converts checker.CheckResult to domain check.Result
// For now, we create a minimal adapter since the infrastructure layer
// is still using the detailed checker.CheckResult structure
type resultAdapter struct{}

func (a *resultAdapter) toDomain(target string, checkerResult checker.CheckResult) (*check.Result, error) {
	// Determine status
	var status check.CheckStatus
	if checkerResult.Status == "ok" {
		status = check.CheckStatusOK
	} else {
		status = check.CheckStatusError
	}

	// Create domain result
	result, err := check.NewResult(target, status)
	if err != nil {
		return nil, err
	}

	// Set basic fields
	result.SetHTTPStatus(checkerResult.HTTPStatus)
	result.SetResponseTime(checkerResult.ResponseTime)

	if checkerResult.TLSExpiry != "" {
		if expiry, err := time.Parse(time.RFC3339, checkerResult.TLSExpiry); err == nil {
			result.SetTLSExpiry(expiry)
		}
	}

	if checkerResult.Error != "" {
		result.SetError(checkerResult.Error)
	}

	// For Phase 2B, we keep the basic conversion
	// Full field mapping can be added incrementally as needed
	// The important part is that audit logging works and check orchestration works

	return result, nil
}

// DDD-based check HTTP command
var checkHTTPCmdDDD = &cobra.Command{
	Use:   "http",
	Short: "Run safe HTTP/TLS checks for an engagement's scope",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

	appCtx := getAppContext(cmd)
	runtimeCfg := appCtx.Config.Check
	startTime := time.Now()

		// Setup signal handling
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

		// Parse flags
		engagementID := cmd.Flag("id").Value.String()
		roeConfirm := cmd.Flag("roe-confirm").Value.String() == "true"

		if engagementID == "" {
			return errors.New("--id is required")
		}

		if !roeConfirm {
			return errors.New("must pass --roe-confirm to run checks")
		}

		// Validate engagement using service
		eng, err := appCtx.Services.EngagementService.GetEngagement(ctx, engagementID)
		if err != nil {
			if errors.Is(err, sharedErrors.ErrEngagementNotFound) {
				return fmt.Errorf("engagement %s not found", engagementID)
			}
			return fmt.Errorf("failed to get engagement: %w", err)
		}

		// Validate authorization
		if err := appCtx.Services.EngagementService.ValidateEngagementForChecks(ctx, engagementID, ""); err != nil {
			return fmt.Errorf("engagement validation failed: %w", err)
		}

		// Create check run
		checkRun, err := appCtx.Services.CheckOrchestrator.CreateCheckRun(ctx, engagementID, appCtx.Operator)
		if err != nil {
			return fmt.Errorf("failed to create check run: %w", err)
		}

		fmt.Printf("%s Starting HTTP checks for engagement: %s\n", colorInfo("→"), eng.Name())
		fmt.Printf("%s Targets: %d\n", colorInfo("→"), len(eng.Scope()))
		fmt.Println()

		// Create HTTP checker
		httpChecker := &checker.HTTPChecker{
			Timeout:    time.Duration(runtimeCfg.TimeoutSecs) * time.Second,
			CaptureRaw: runtimeCfg.AuditAppendRaw,
			RawHandler: func(target string, headers http.Header, bodySnippet string) error {
				return SaveRawCapture(appCtx.ResultsDir, engagementID, target, headers, bodySnippet)
			},
		}

		// Create runner
		runner := &checker.Runner{
			Concurrency: runtimeCfg.Concurrency,
			RateLimit:   runtimeCfg.RateLimit,
			Timeout:     time.Duration(runtimeCfg.TimeoutSecs) * time.Second,
		}

		// Progress tracking
		var progress *progressPrinter
		if runtimeCfg.ProgressEnabled {
			progress = newProgressPrinter(len(eng.Scope()), httpChecker.Name())
			progress.Start()
		}

		// Result adapter
		adapter := &resultAdapter{}

		// Audit function
		auditFn := func(target string, checkerResult checker.CheckResult, duration float64) error {
			// Create audit entry
			entry := &audit.Entry{
				Timestamp:       time.Now(),
				EngagementID:    engagementID,
				Operator:        appCtx.Operator,
				Command:         "check http",
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

			// Record audit entry
			if err := appCtx.Services.CheckOrchestrator.RecordAuditEntry(ctx, entry); err != nil {
				return fmt.Errorf("failed to record audit: %w", err)
			}

			// Convert to domain result and add to check run
			domainResult, err := adapter.toDomain(target, checkerResult)
			if err != nil {
				return fmt.Errorf("failed to convert result: %w", err)
			}

			if err := appCtx.Services.CheckOrchestrator.AddCheckResult(ctx, checkRun, domainResult); err != nil {
				return fmt.Errorf("failed to add result: %w", err)
			}

			// Update progress
			if progress != nil {
				progress.Increment(checkerResult.Status == "ok", duration)
			}

			return nil
		}

		// Run checks
		results := runner.RunChecks(ctx, eng.Scope(), httpChecker, auditFn)

		// Stop progress
		if progress != nil {
			progress.Stop()
		}

		runDuration := time.Since(startTime)
		if runtimeCfg.TelemetryEnabled {
			if err := recordTelemetry(appCtx, engagementID, httpChecker.Name(), results, runDuration); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to record telemetry: %v\n", err)
			}
		}

		fmt.Printf("\n%s Check run complete\n", colorSuccess("✓"))
		fmt.Printf("%s Checked: %d targets\n", colorInfo("→"), len(results))

		// Seal audit trail
		hashAlgo := runtimeCfg.HashAlgorithm
		if hashAlgo == "" {
			hashAlgo = "sha256"
		}

		auditHash, err := appCtx.Services.CheckOrchestrator.SealAuditTrail(ctx, engagementID, hashAlgo)
		if err != nil {
			return fmt.Errorf("failed to seal audit trail: %w", err)
		}

		// Finalize check run
		if err := appCtx.Services.CheckOrchestrator.FinalizeCheckRun(ctx, checkRun, auditHash, hashAlgo); err != nil {
			return fmt.Errorf("failed to finalize check run: %w", err)
		}

		// Print summary
		resultsPath := filepath.Join(appCtx.ResultsDir, engagementID, "http_results.json")
		auditPath := filepath.Join(appCtx.ResultsDir, engagementID, "audit.csv")

		fmt.Println()
		fmt.Printf("%s Results: %s\n", colorSuccess("→"), resultsPath)
		fmt.Printf("%s Audit: %s\n", colorSuccess("→"), auditPath)
		fmt.Printf("%s Audit hash (%s): %s\n", colorSuccess("→"), hashAlgo, auditHash)

		return nil
	},
}

// DDD-based check DNS command
var checkDNSCmdDDD = &cobra.Command{
	Use:   "dns",
	Short: "Run DNS checks for an engagement's scope",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		appCtx := getAppContext(cmd)
		runtimeCfg := appCtx.Config.Check
		startTime := time.Now()

		// Setup signal handling
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

		// Parse flags
		engagementID := cmd.Flag("id").Value.String()
		roeConfirm := cmd.Flag("roe-confirm").Value.String() == "true"

		if engagementID == "" {
			return errors.New("--id is required")
		}

		if !roeConfirm {
			return errors.New("must pass --roe-confirm to run checks")
		}

		// Validate engagement
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

		// Create check run
		checkRun, err := appCtx.Services.CheckOrchestrator.CreateCheckRun(ctx, engagementID, appCtx.Operator)
		if err != nil {
			return fmt.Errorf("failed to create check run: %w", err)
		}

		fmt.Printf("%s Starting DNS checks for engagement: %s\n", colorInfo("→"), eng.Name())
		fmt.Printf("%s Targets: %d\n", colorInfo("→"), len(eng.Scope()))
		fmt.Println()

		// Create DNS checker
		dnsChecker := &checker.DNSChecker{
			Timeout:    time.Duration(runtimeCfg.DNS.Timeout) * time.Second,
			NameServer: runtimeCfg.DNS.Nameservers,
		}

		// Create runner
		runner := &checker.Runner{
			Concurrency: runtimeCfg.Concurrency,
			RateLimit:   runtimeCfg.RateLimit,
			Timeout:     time.Duration(runtimeCfg.DNS.Timeout) * time.Second,
		}

		// Progress tracking
		var progress *progressPrinter
		if runtimeCfg.ProgressEnabled {
			progress = newProgressPrinter(len(eng.Scope()), dnsChecker.Name())
			progress.Start()
		}

		adapter := &resultAdapter{}

		// Audit function
		auditFn := func(target string, checkerResult checker.CheckResult, duration float64) error {
			entry := &audit.Entry{
				Timestamp:       time.Now(),
				EngagementID:    engagementID,
				Operator:        appCtx.Operator,
				Command:         "check dns",
				Target:          target,
				Status:          checkerResult.Status,
				Notes:           checkerResult.Notes,
				Error:           checkerResult.Error,
				DurationSeconds: duration,
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

		// Run checks
		results := runner.RunChecks(ctx, eng.Scope(), dnsChecker, auditFn)

		if progress != nil {
			progress.Stop()
		}

		runDuration := time.Since(startTime)
		if runtimeCfg.TelemetryEnabled {
			if err := recordTelemetry(appCtx, engagementID, dnsChecker.Name(), results, runDuration); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to record telemetry: %v\n", err)
			}
		}

		// Count results
		okCount := 0
		errorCount := 0
		for _, r := range results {
			if r.Status == "ok" {
				okCount++
			} else {
				errorCount++
			}
		}

		fmt.Printf("\n%s DNS checks complete\n", colorSuccess("✓"))
		fmt.Printf("%s Success: %d | Errors: %d\n", colorInfo("→"), okCount, errorCount)

		// Seal audit trail
		hashAlgo := runtimeCfg.HashAlgorithm
		if hashAlgo == "" {
			hashAlgo = "sha256"
		}

		auditHash, err := appCtx.Services.CheckOrchestrator.SealAuditTrail(ctx, engagementID, hashAlgo)
		if err != nil {
			return fmt.Errorf("failed to seal audit trail: %w", err)
		}

		// Finalize check run
		if err := appCtx.Services.CheckOrchestrator.FinalizeCheckRun(ctx, checkRun, auditHash, hashAlgo); err != nil {
			return fmt.Errorf("failed to finalize check run: %w", err)
		}

		// Print summary
		resultsPath := filepath.Join(appCtx.ResultsDir, engagementID, "http_results.json")
		auditPath := filepath.Join(appCtx.ResultsDir, engagementID, "audit.csv")

		fmt.Println()
		fmt.Printf("%s Results: %s\n", colorSuccess("→"), resultsPath)
		fmt.Printf("%s Audit: %s\n", colorSuccess("→"), auditPath)
		fmt.Printf("%s Audit hash (%s): %s\n", colorSuccess("→"), hashAlgo, auditHash)

		return nil
	},
}

var checkNetworkCmdDDD = &cobra.Command{
	Use:   "network",
	Short: "Run network exposure and takeover checks for an engagement's scope",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		appCtx := getAppContext(cmd)
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

		engagementID := cmd.Flag("id").Value.String()
		roeConfirm := cmd.Flag("roe-confirm").Value.String() == "true"

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

		fmt.Printf("%s Starting network checks for engagement: %s\n", colorInfo("→"), eng.Name())
		fmt.Printf("%s Initial targets: %d\n", colorInfo("→"), len(eng.Scope()))
		fmt.Println()

		netCfg := runtimeCfg.Network
		var ports []int
		if len(netCfg.Ports) > 0 {
			ports = append([]int(nil), netCfg.Ports...)
		}

		networkChecker := &checker.NetworkChecker{
			Timeout:         time.Duration(runtimeCfg.TimeoutSecs) * time.Second,
			PortScanTimeout: time.Duration(netCfg.PortScanTimeout) * time.Second,
			EnablePortScan:  netCfg.EnablePortScan,
			CommonPorts:     ports,
			MaxPortWorkers:  netCfg.MaxPortWorkers,
		}

		runner := &checker.Runner{
			Concurrency: runtimeCfg.Concurrency,
			RateLimit:   runtimeCfg.RateLimit,
			Timeout:     time.Duration(runtimeCfg.TimeoutSecs) * time.Second,
		}

		baseTargets := append([]string(nil), eng.Scope()...)
		targets := expandTargetsWithCrawl(ctx, baseTargets, runtimeCfg)

		var progress *progressPrinter
		if runtimeCfg.ProgressEnabled {
			progress = newProgressPrinter(len(targets), networkChecker.Name())
			progress.Start()
		}

		adapter := &resultAdapter{}

		auditFn := func(target string, checkerResult checker.CheckResult, duration float64) error {
			entry := &audit.Entry{
				Timestamp:       time.Now(),
				EngagementID:    engagementID,
				Operator:        appCtx.Operator,
				Command:         "check network",
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

		results := runner.RunChecks(ctx, targets, networkChecker, auditFn)

		if progress != nil {
			progress.Stop()
		}

		runDuration := time.Since(startTime)
		if runtimeCfg.TelemetryEnabled {
			if err := recordTelemetry(appCtx, engagementID, networkChecker.Name(), results, runDuration); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to record telemetry: %v\n", err)
			}
		}

		issues := 0
		takeovers := 0
		totalPorts := 0
		for _, r := range results {
			if r.NetworkSecurity == nil {
				continue
			}
			totalPorts += len(r.NetworkSecurity.OpenPorts)
			if len(r.NetworkSecurity.Issues) > 0 {
				issues++
			}
			if r.NetworkSecurity.SubdomainTakeover != nil && r.NetworkSecurity.SubdomainTakeover.Vulnerable {
				takeovers++
			}
		}

		fmt.Printf("\n%s Network checks complete\n", colorSuccess("✓"))
		fmt.Printf("%s Processed: %d target(s)\n", colorInfo("→"), len(results))
		fmt.Printf("%s Issues: %d | Takeover indicators: %d | Open ports: %d\n", colorInfo("→"), issues, takeovers, totalPorts)

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

func init() {
	// HTTP check flags
	checkHTTPCmdDDD.Flags().String("id", "", "Engagement ID")
	checkHTTPCmdDDD.Flags().Bool("roe-confirm", false, "Confirm ROE and authorization")

	// DNS check flags
	checkDNSCmdDDD.Flags().String("id", "", "Engagement ID")
	checkDNSCmdDDD.Flags().Bool("roe-confirm", false, "Confirm ROE and authorization")

	// Network check flags
	checkNetworkCmdDDD.Flags().String("id", "", "Engagement ID")
	checkNetworkCmdDDD.Flags().Bool("roe-confirm", false, "Confirm ROE and authorization")
	checkNetworkCmdDDD.Flags().BoolVar(&cliConfig.Check.Network.EnablePortScan, "enable-port-scan", cliConfig.Check.Network.EnablePortScan, "Scan TCP ports for exposure and banner details")
	checkNetworkCmdDDD.Flags().IntSliceVar(&cliConfig.Check.Network.Ports, "ports", cliConfig.Check.Network.Ports, "Comma-separated list of TCP ports to scan (defaults to built-in set)")
	checkNetworkCmdDDD.Flags().IntVar(&cliConfig.Check.Network.PortScanTimeout, "port-scan-timeout", cliConfig.Check.Network.PortScanTimeout, "Per-port scan timeout in seconds")
	checkNetworkCmdDDD.Flags().IntVar(&cliConfig.Check.Network.MaxPortWorkers, "port-workers", cliConfig.Check.Network.MaxPortWorkers, "Concurrent port scan workers")
	checkNetworkCmdDDD.Flags().BoolVar(&cliConfig.Check.Crawl.Enabled, "crawl", cliConfig.Check.Crawl.Enabled, "Discover same-host links (auto-detects JavaScript/SPA sites)")
	checkNetworkCmdDDD.Flags().IntVar(&cliConfig.Check.Crawl.MaxDepth, "crawl-depth", cliConfig.Check.Crawl.MaxDepth, "Maximum link depth to follow per target")
	checkNetworkCmdDDD.Flags().IntVar(&cliConfig.Check.Crawl.MaxPages, "crawl-max-pages", cliConfig.Check.Crawl.MaxPages, "Maximum additional pages to discover per target")
	checkNetworkCmdDDD.Flags().BoolVar(&cliConfig.Check.Crawl.EnableJS, "crawl-force-js", cliConfig.Check.Crawl.EnableJS, "Force JavaScript crawler for all targets (overrides auto-detection)")
	checkNetworkCmdDDD.Flags().IntVar(&cliConfig.Check.Crawl.JSWaitTime, "crawl-js-wait", cliConfig.Check.Crawl.JSWaitTime, "Seconds to wait for JavaScript to render (when JS is used)")
}
