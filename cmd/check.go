package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/domain/audit"
	"github.com/khanhnv2901/seca-cli/internal/domain/check"
	"github.com/khanhnv2901/seca-cli/internal/infrastructure/checker"
	sharedErrors "github.com/khanhnv2901/seca-cli/internal/shared/errors"
	"github.com/spf13/cobra"
)

type RunMetadata struct {
	Operator             string    `json:"operator"`
	EngagementID         string    `json:"engagement_id"`
	EngagementName       string    `json:"engagement_name"`
	Owner                string    `json:"owner"`
	StartAt              time.Time `json:"started_at"`
	CompleteAt           time.Time `json:"completed_at"`
	AuditHash            string    `json:"audit_hash,omitempty"`
	LegacyAuditHash      string    `json:"audit_sha256,omitempty"`
	HashAlgorithm        string    `json:"hash_algorithm,omitempty"`
	SignatureFingerprint string    `json:"signature_fingerprint,omitempty"`
	TotalTargets         int       `json:"total_targets"`
	// Note: http_results.json hash is stored in http_results.json.<hash> file, not here
}

type RunOutput struct {
	Metadata RunMetadata           `json:"metadata"`
	Results  []checker.CheckResult `json:"results"`
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run safe, authorized checks against scoped targets (no scanning/exploitation)",
}

func expandTargetsWithCrawl(ctx context.Context, targets []string, runtimeCfg CheckRuntimeConfig) []string {
	crawl := runtimeCfg.Crawl
	if !crawl.Enabled || crawl.MaxDepth <= 0 || crawl.MaxPages <= 0 {
		return targets
	}

	crawlOpts := checker.CrawlOptions{
		MaxDepth:     crawl.MaxDepth,
		MaxPages:     crawl.MaxPages,
		SameHostOnly: true,
		Timeout:      time.Duration(runtimeCfg.TimeoutSecs) * time.Second,
	}

	jsCrawlOpts := checker.JSCrawlOptions{
		CrawlOptions:     crawlOpts,
		EnableJavaScript: crawl.EnableJS,
		WaitTime:         time.Duration(crawl.JSWaitTime) * time.Second,
	}

	set := newTargetSet()
	expanded := make([]string, 0, len(targets)+crawl.MaxPages*len(targets))

	for _, target := range targets {
		if set.Add(target) {
			expanded = append(expanded, target)
		}

		var discovered []string
		var err error

		if crawl.AutoDetectJS {
			discovered, err = checker.DiscoverInScopeLinksAuto(ctx, target, jsCrawlOpts)
		} else if crawl.EnableJS {
			discovered, err = checker.DiscoverInScopeLinksJS(ctx, target, jsCrawlOpts)
		} else {
			discovered, err = checker.DiscoverInScopeLinks(ctx, target, crawlOpts)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: crawl failed for %s: %v\n", target, err)
			continue
		}

		appended := 0
		for _, url := range discovered {
			if set.Add(url) {
				expanded = append(expanded, url)
				appended++
			}
		}
		if appended > 0 {
			crawlType := "static"
			if crawl.EnableJS {
				crawlType = "JavaScript-enabled"
			} else if crawl.AutoDetectJS {
				crawlType = "auto-detect"
			}
			fmt.Printf("%s discovered %d page(s) under %s [%s]\n", colorInfo("→"), appended, checker.NormalizeHTTPTarget(target), crawlType)
		}
	}

	return expanded
}

type targetSet struct {
	seen map[string]struct{}
}

func newTargetSet() *targetSet {
	return &targetSet{seen: make(map[string]struct{})}
}

func (s *targetSet) Add(target string) bool {
	if s.seen == nil {
		s.seen = make(map[string]struct{})
	}
	key := canonicalTarget(target)
	if key == "" {
		key = target
	}
	if _, exists := s.seen[key]; exists {
		return false
	}
	s.seen[key] = struct{}{}
	return true
}

func canonicalTarget(target string) string {
	normalized := checker.NormalizeHTTPTarget(target)
	parsed, err := url.Parse(normalized)
	if err != nil {
		return strings.TrimRight(normalized, "/")
	}
	parsed.Fragment = ""
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String()
}

// resultAdapter converts infrastructure checker results to domain entities
type resultAdapter struct{}

func (a *resultAdapter) toDomain(target string, checkerResult checker.CheckResult) (*check.Result, error) {
	var status check.CheckStatus
	if checkerResult.Status == "ok" {
		status = check.CheckStatusOK
	} else {
		status = check.CheckStatusError
	}

	result, err := check.NewResult(target, status)
	if err != nil {
		return nil, err
	}

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

	return result, nil
}

var checkHTTPCmd = &cobra.Command{
	Use:   "http",
	Short: "Run safe HTTP/TLS checks for an engagement's scope",
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

		fmt.Printf("%s Starting HTTP checks for engagement: %s\n", colorInfo("→"), eng.Name())
		fmt.Printf("%s Targets: %d\n", colorInfo("→"), len(eng.Scope()))
		fmt.Println()

		httpChecker := &checker.HTTPChecker{
			Timeout:    time.Duration(runtimeCfg.TimeoutSecs) * time.Second,
			CaptureRaw: runtimeCfg.AuditAppendRaw,
			RawHandler: func(target string, headers http.Header, bodySnippet string) error {
				return SaveRawCapture(appCtx.ResultsDir, engagementID, target, headers, bodySnippet)
			},
		}

		runner := &checker.Runner{
			Concurrency: runtimeCfg.Concurrency,
			RateLimit:   runtimeCfg.RateLimit,
			Timeout:     time.Duration(runtimeCfg.TimeoutSecs) * time.Second,
		}

		var progress *progressPrinter
		if runtimeCfg.ProgressEnabled {
			progress = newProgressPrinter(len(eng.Scope()), httpChecker.Name())
			progress.Start()
		}

		adapter := &resultAdapter{}

		auditFn := func(target string, checkerResult checker.CheckResult, duration float64) error {
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

		results := runner.RunChecks(ctx, eng.Scope(), httpChecker, auditFn)

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

var checkDNSCmd = &cobra.Command{
	Use:   "dns",
	Short: "Run DNS checks for an engagement's scope",
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

		fmt.Printf("%s Starting DNS checks for engagement: %s\n", colorInfo("→"), eng.Name())
		fmt.Printf("%s Targets: %d\n", colorInfo("→"), len(eng.Scope()))
		fmt.Println()

		dnsChecker := &checker.DNSChecker{
			Timeout:    time.Duration(runtimeCfg.DNS.Timeout) * time.Second,
			NameServer: runtimeCfg.DNS.Nameservers,
		}

		runner := &checker.Runner{
			Concurrency: runtimeCfg.Concurrency,
			RateLimit:   runtimeCfg.RateLimit,
			Timeout:     time.Duration(runtimeCfg.DNS.Timeout) * time.Second,
		}

		var progress *progressPrinter
		if runtimeCfg.ProgressEnabled {
			progress = newProgressPrinter(len(eng.Scope()), dnsChecker.Name())
			progress.Start()
		}

		adapter := &resultAdapter{}

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

var checkNetworkCmd = &cobra.Command{
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
	checkCmd.PersistentFlags().IntVarP(&cliConfig.Check.Concurrency, "concurrency", "c", cliConfig.Check.Concurrency, "max concurrent requests")
	checkCmd.PersistentFlags().IntVarP(&cliConfig.Check.RateLimit, "rate", "r", cliConfig.Check.RateLimit, "requests per second (global)")
	checkCmd.PersistentFlags().IntVarP(&cliConfig.Check.TimeoutSecs, "timeout", "t", cliConfig.Check.TimeoutSecs, "request timeout in seconds")
	checkCmd.PersistentFlags().BoolVar(&cliConfig.Check.TelemetryEnabled, "telemetry", cliConfig.Check.TelemetryEnabled, "Record telemetry metrics (durations, success rates)")
	checkCmd.PersistentFlags().BoolVar(&cliConfig.Check.ProgressEnabled, "progress", cliConfig.Check.ProgressEnabled, "Display live progress for checks")
	checkCmd.PersistentFlags().StringVar(&cliConfig.Check.HashAlgorithm, "hash", cliConfig.Check.HashAlgorithm, "Hash algorithm for integrity verification (sha256|sha512)")
	checkCmd.PersistentFlags().BoolVar(&cliConfig.Check.SecureResults, "secure-results", cliConfig.Check.SecureResults, "Encrypt audit logs with operator GPG key after run")
	checkCmd.PersistentFlags().IntVar(&cliConfig.Check.RetryCount, "retry", cliConfig.Check.RetryCount, "Number of times to retry failed targets")

	checkCmd.AddCommand(checkHTTPCmd)
	checkCmd.AddCommand(checkDNSCmd)
	checkCmd.AddCommand(checkNetworkCmd)

	checkHTTPCmd.Flags().String("id", "", "Engagement ID")
	checkHTTPCmd.Flags().Bool("roe-confirm", false, "Confirm ROE and authorization")

	checkDNSCmd.Flags().String("id", "", "Engagement ID")
	checkDNSCmd.Flags().Bool("roe-confirm", false, "Confirm ROE and authorization")

	checkNetworkCmd.Flags().String("id", "", "Engagement ID")
	checkNetworkCmd.Flags().Bool("roe-confirm", false, "Confirm ROE and authorization")
	checkNetworkCmd.Flags().BoolVar(&cliConfig.Check.Network.EnablePortScan, "enable-port-scan", cliConfig.Check.Network.EnablePortScan, "Scan TCP ports for exposure and banner details")
	checkNetworkCmd.Flags().IntSliceVar(&cliConfig.Check.Network.Ports, "ports", cliConfig.Check.Network.Ports, "Comma-separated list of TCP ports to scan (defaults to built-in set)")
	checkNetworkCmd.Flags().IntVar(&cliConfig.Check.Network.PortScanTimeout, "port-scan-timeout", cliConfig.Check.Network.PortScanTimeout, "Per-port scan timeout in seconds")
	checkNetworkCmd.Flags().IntVar(&cliConfig.Check.Network.MaxPortWorkers, "port-workers", cliConfig.Check.Network.MaxPortWorkers, "Concurrent port scan workers")
	checkNetworkCmd.Flags().BoolVar(&cliConfig.Check.Crawl.Enabled, "crawl", cliConfig.Check.Crawl.Enabled, "Discover same-host links (auto-detects JavaScript/SPA sites)")
	checkNetworkCmd.Flags().IntVar(&cliConfig.Check.Crawl.MaxDepth, "crawl-depth", cliConfig.Check.Crawl.MaxDepth, "Maximum link depth to follow per target")
	checkNetworkCmd.Flags().IntVar(&cliConfig.Check.Crawl.MaxPages, "crawl-max-pages", cliConfig.Check.Crawl.MaxPages, "Maximum additional pages to discover per target")
	checkNetworkCmd.Flags().BoolVar(&cliConfig.Check.Crawl.EnableJS, "crawl-force-js", cliConfig.Check.Crawl.EnableJS, "Force JavaScript crawler for all targets (overrides auto-detection)")
	checkNetworkCmd.Flags().IntVar(&cliConfig.Check.Crawl.JSWaitTime, "crawl-js-wait", cliConfig.Check.Crawl.JSWaitTime, "Seconds to wait for JavaScript to render (when JS is used)")
	registerPluginCommands()
}
