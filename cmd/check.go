package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/checker"
	consts "github.com/khanhnv2901/seca-cli/internal/constants"
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
	// Note: results.json hash is stored in results.json.<hash> file, not here
}

type RunOutput struct {
	Metadata RunMetadata           `json:"metadata"`
	Results  []checker.CheckResult `json:"results"`
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run safe, authorized checks against scoped targets (no scanning/exploitation)",
}

var checkHTTPCmd = &cobra.Command{
	Use:   "http",
	Short: "Run safe HTTP/TLS checks for an engagement's scope",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheckCommand(cmd, checkConfig{
			CreateChecker: func(appCtx *AppContext, params checkParams) checker.Checker {
				runtimeCfg := appCtx.Config.Check
				return &checker.HTTPChecker{
					Timeout:    time.Duration(runtimeCfg.TimeoutSecs) * time.Second,
					CaptureRaw: runtimeCfg.AuditAppendRaw,
					RawHandler: func(target string, headers http.Header, bodySnippet string) error {
						return SaveRawCapture(appCtx.ResultsDir, params.ID, target, headers, bodySnippet)
					},
				}
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
			ResultsFilename:        "results.json",
			TimeoutSecs:            cliConfig.Check.TimeoutSecs,
			VerificationCmdBuilder: makeVerificationCommand("results.json"),
			SupportsRawCapture:     true,
			PrintSummary: func(results []checker.CheckResult, resultsPath, auditPath, auditHash, resultsHash string, hashAlgo HashAlgorithm) {
				fmt.Println(colorSuccess("Run complete."))
				fmt.Printf("%s %s\n", colorInfo("Results:"), resultsPath)
				fmt.Printf("%s %s\n", colorInfo("Audit:"), auditPath)
				fmt.Printf("%s audit: %s\n%s results: %s\n", hashAlgo.DisplayName(), auditHash, hashAlgo.DisplayName(), resultsHash)
			},
		})
	},
}

var checkDNSCmd = &cobra.Command{
	Use:   "dns",
	Short: "Run DNS checks for an engagement's scope",
	Long: `Perform DNS resolution checks on engagement targets.
	This command will:
- Resolve A records (IPv4 addresses)
- Resolve AAAA records (IPv6 addresses)
- Check CNAME records
- Check MX records (mail servers)
- Check NS records (nameservers)
- Check TXT records (SPF, DKIM, etc.)
- Perform reverse DNS lookups (PTR records)

All checks are safe, non-intrusive DNS queries only.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheckCommand(cmd, checkConfig{
			CreateChecker: func(appCtx *AppContext, params checkParams) checker.Checker {
				runtimeCfg := appCtx.Config.Check
				return &checker.DNSChecker{
					Timeout:    time.Duration(runtimeCfg.DNS.Timeout) * time.Second,
					NameServer: runtimeCfg.DNS.Nameservers,
				}
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
						0,  // No HTTP status for DNS
						"", // No TLS expiry for DNS
						result.Notes,
						result.Error,
						duration,
					)
				}
			},
			ResultsFilename:        "dns_results.json",
			TimeoutSecs:            cliConfig.Check.DNS.Timeout,
			VerificationCmdBuilder: makeVerificationCommand("dns_results.json"),
			SupportsRawCapture:     false,
			PrintSummary: func(results []checker.CheckResult, resultsPath, auditPath, auditHash, resultsHash string, hashAlgo HashAlgorithm) {
				// Count successes and errors
				okCount := 0
				errorCount := 0
				for _, r := range results {
					if r.Status == "ok" {
						okCount++
					} else {
						errorCount++
					}
				}

				fmt.Println(colorSuccess("DNS Check complete."))
				fmt.Printf("%s %s\n", colorInfo("Results:"), resultsPath)
				fmt.Printf("%s %s\n", colorInfo("Audit:"), auditPath)
				fmt.Printf("%s audit: %s\n%s results: %s\n", hashAlgo.DisplayName(), auditHash, hashAlgo.DisplayName(), resultsHash)
				fmt.Printf("Summary: %d OK, %d Errors (out of %d targets)\n", okCount, errorCount, len(results))
			},
		})
	},
}

var checkNetworkCmd = &cobra.Command{
	Use:   "network",
	Short: "Run network exposure and takeover checks for an engagement's scope",
	Long: `Perform network-layer safety checks for each scoped target.

This command performs:
- Subdomain takeover detection via DNS + HTTP fingerprints
- Optional TCP port scanning with banner grabbing and risk insights

All checks are passive and respect the engagement scope.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheckCommand(cmd, checkConfig{
			CreateChecker: func(appCtx *AppContext, params checkParams) checker.Checker {
				runtimeCfg := appCtx.Config.Check
				netCfg := runtimeCfg.Network

				var ports []int
				if len(netCfg.Ports) > 0 {
					ports = append([]int(nil), netCfg.Ports...)
				}

				return &checker.NetworkChecker{
					Timeout:         time.Duration(runtimeCfg.TimeoutSecs) * time.Second,
					PortScanTimeout: time.Duration(netCfg.PortScanTimeout) * time.Second,
					EnablePortScan:  netCfg.EnablePortScan,
					CommonPorts:     ports,
					MaxPortWorkers:  netCfg.MaxPortWorkers,
				}
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
			ResultsFilename:        "network_results.json",
			TimeoutSecs:            cliConfig.Check.TimeoutSecs,
			VerificationCmdBuilder: makeVerificationCommand("network_results.json"),
			SupportsRawCapture:     false,
			PrintSummary: func(results []checker.CheckResult, resultsPath, auditPath, auditHash, resultsHash string, hashAlgo HashAlgorithm) {
				issues := 0
				takeovers := 0
				totalPorts := 0
				for _, r := range results {
					if r.NetworkSecurity == nil {
						continue
					}
					netSec := r.NetworkSecurity
					totalPorts += len(netSec.OpenPorts)
					if len(netSec.Issues) > 0 {
						issues++
					}
					if netSec.SubdomainTakeover != nil && netSec.SubdomainTakeover.Vulnerable {
						takeovers++
					}
				}

				fmt.Println(colorSuccess("Network check complete."))
				fmt.Printf("%s %s\n", colorInfo("Results:"), resultsPath)
				fmt.Printf("%s %s\n", colorInfo("Audit:"), auditPath)
				fmt.Printf("%s audit: %s\n%s results: %s\n", hashAlgo.DisplayName(), auditHash, hashAlgo.DisplayName(), resultsHash)
				fmt.Printf("Summary: %d target(s), %d with issues, %d takeover indicators, %d open port(s)\n", len(results), issues, takeovers, totalPorts)
			},
		})
	},
}

// checkParams holds common parameters for check commands
type checkParams struct {
	ID             string
	ROEConfirm     bool
	ComplianceMode bool
	AutoSign       bool
	GPGKey         string
}

// checkConfig holds configuration for running a check command
type checkConfig struct {
	// Checker creation function
	CreateChecker func(appCtx *AppContext, params checkParams) checker.Checker

	// Audit function creation
	CreateAuditFn func(appCtx *AppContext, params checkParams, chk checker.Checker) func(string, checker.CheckResult, float64) error

	// Results filename (e.g., "results.json" or "dns_results.json")
	ResultsFilename string

	// Timeout in seconds for the runner
	TimeoutSecs int

	// Verification command for compliance summary
	VerificationCmdBuilder func(HashAlgorithm) string

	// Whether this check supports raw capture
	SupportsRawCapture bool

	// Custom result summary printer (optional)
	PrintSummary func(results []checker.CheckResult, resultsPath, auditPath, auditHash, resultsHash string, algo HashAlgorithm)
}

func makeVerificationCommand(resultsFilename string) func(HashAlgorithm) string {
	return func(algo HashAlgorithm) string {
		sumCmd := algo.SumCommand()
		ext := algo.FileExtension()
		return fmt.Sprintf("%s -c audit.csv%s && %s -c %s%s", sumCmd, ext, sumCmd, resultsFilename, ext)
	}
}

// runCheckCommand executes a check command with the given configuration.
// This is the common execution pattern shared by all check commands (HTTP, DNS, etc.)
func runCheckCommand(cmd *cobra.Command, config checkConfig) error {
	// Get application context
	appCtx := getAppContext(cmd)
	runtimeCfg := appCtx.Config.Check

	hashAlgo, err := ParseHashAlgorithm(runtimeCfg.HashAlgorithm)
	if err != nil {
		return err
	}
	retryCount := runtimeCfg.RetryCount
	if retryCount < 0 {
		retryCount = 0
	}

	// Parse flags
	params := checkParams{
		ID:             cmd.Flag("id").Value.String(),
		ROEConfirm:     cmd.Flag("roe-confirm").Value.String() == "true",
		ComplianceMode: cmd.Flag("compliance-mode").Value.String() == "true",
		AutoSign:       runtimeCfg.AutoSign,
		GPGKey:         runtimeCfg.GPGKey,
	}

	// Validate parameters
	retentionForValidation := 0
	if config.SupportsRawCapture {
		retentionForValidation = runtimeCfg.RetentionDays
	}
	if err := validateCheckParams(params, appCtx, config.SupportsRawCapture && runtimeCfg.AuditAppendRaw, retentionForValidation); err != nil {
		return err
	}

	if runtimeCfg.SecureResults && runtimeCfg.GPGKey == "" {
		return fmt.Errorf("--secure-results requires --gpg-key")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// Load engagement
	eng, err := loadEngagementByID(params.ID)
	if err != nil {
		return err
	}

	startAll := time.Now()
	if _, err := ensureResultsDir(appCtx.ResultsDir, params.ID); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	// Create checker using the provided factory function
	chk := config.CreateChecker(appCtx, params)

	// Create audit function using the provided factory
	auditFn := config.CreateAuditFn(appCtx, params, chk)

	var progress *progressPrinter
	if runtimeCfg.ProgressEnabled {
		progress = newProgressPrinter(len(eng.Scope), chk.Name())
		progress.Start()
		if auditFn != nil {
			orig := auditFn
			auditFn = func(target string, result checker.CheckResult, duration float64) error {
				if err := orig(target, result, duration); err != nil {
					return err
				}
				progress.Increment(result.Status == "ok", duration)
				return nil
			}
		} else {
			auditFn = func(target string, result checker.CheckResult, duration float64) error {
				progress.Increment(result.Status == "ok", duration)
				return nil
			}
		}
	}

	// Create runner
	runner := &checker.Runner{
		Concurrency: runtimeCfg.Concurrency,
		RateLimit:   runtimeCfg.RateLimit,
		Timeout:     time.Duration(config.TimeoutSecs) * time.Second,
	}

	targetOrder := append([]string(nil), eng.Scope...)
	targetOrder = expandTargetsWithCrawl(ctx, targetOrder, runtimeCfg)
	pending := append([]string(nil), targetOrder...)
	finalResults := make(map[string]checker.CheckResult, len(targetOrder))
	maxAttempts := retryCount + 1
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts && len(pending) > 0; attempt++ {
		attemptTargets := append([]string(nil), pending...)
		attemptResults := runner.RunChecks(ctx, attemptTargets, chk, auditFn)

		resultMap := make(map[string]checker.CheckResult, len(attemptResults))
		for _, res := range attemptResults {
			resultMap[res.Target] = res
		}

		nextPending := make([]string, 0)
		for _, target := range pending {
			if res, ok := resultMap[target]; ok {
				finalResults[target] = res
				if ctx.Err() == nil && !strings.EqualFold(res.Status, "ok") && attempt < maxAttempts {
					nextPending = append(nextPending, target)
				}
			} else if ctx.Err() == nil && attempt < maxAttempts {
				nextPending = append(nextPending, target)
			}
		}

		if ctx.Err() != nil {
			break
		}

		if len(nextPending) > 0 && attempt < maxAttempts {
			fmt.Printf("%s retrying %d target(s) (attempt %d/%d)\n", colorWarn("Retrying"), len(nextPending), attempt+1, maxAttempts)
		}
		pending = nextPending
	}

	if progress != nil {
		progress.Stop()
	}

	if ctx.Err() != nil {
		fmt.Printf("\n%s Run cancelled. Writing partial results...\n", colorWarn("!"))
	} else if len(pending) > 0 {
		fmt.Printf("%s %d target(s) still failing after %d attempt(s).\n", colorWarn("Retries exhausted."), len(pending), maxAttempts)
	}

	results := make([]checker.CheckResult, 0, len(finalResults))
	for _, target := range targetOrder {
		if res, ok := finalResults[target]; ok {
			results = append(results, res)
		}
	}

	// Write results and compute hashes
	metadata := RunMetadata{
		Operator:             appCtx.Operator,
		EngagementID:         params.ID,
		EngagementName:       eng.Name,
		Owner:                eng.Owner,
		StartAt:              startAll,
		HashAlgorithm:        hashAlgo.String(),
		SignatureFingerprint: "",
	}

	if params.AutoSign {
		fingerprint, err := getGPGFingerprint(params.GPGKey)
		if err != nil {
			return fmt.Errorf("resolve GPG fingerprint: %w", err)
		}
		metadata.SignatureFingerprint = fingerprint
	}

	resultsPath, auditPath, auditHash, resultsHash, err := writeResultsAndHash(
		appCtx, params.ID, config.ResultsFilename, metadata, results, startAll, hashAlgo,
	)
	if err != nil {
		return err
	}

	// GPG signing if requested
	if params.AutoSign {
		if err := signHashFiles(auditPath, resultsPath, hashAlgo, params.GPGKey); err != nil {
			return err
		}
	}

	// Optional audit encryption
	if runtimeCfg.SecureResults {
		if _, err := encryptAuditLog(auditPath, runtimeCfg.GPGKey); err != nil {
			return err
		}
	}

	// Print results summary
	if config.PrintSummary != nil {
		config.PrintSummary(results, resultsPath, auditPath, auditHash, resultsHash, hashAlgo)
	} else {
		// Default summary
		fmt.Println(colorSuccess("Run complete."))
		fmt.Printf("%s %s\n", colorInfo("Results:"), resultsPath)
		fmt.Printf("%s %s\n", colorInfo("Audit:"), auditPath)
		fmt.Printf("%s audit: %s\n%s results: %s\n", hashAlgo.DisplayName(), auditHash, hashAlgo.DisplayName(), resultsHash)
	}

	verificationCmd := ""
	if config.VerificationCmdBuilder != nil {
		verificationCmd = config.VerificationCmdBuilder(hashAlgo)
	}

	// Print compliance summary if in compliance mode
	if params.ComplianceMode {
		rawCaptureEnabled := config.SupportsRawCapture && runtimeCfg.AuditAppendRaw
		retentionDaysForSummary := 0
		if rawCaptureEnabled {
			retentionDaysForSummary = runtimeCfg.RetentionDays
		}
		printComplianceSummary(
			appCtx, eng, auditHash, resultsHash,
			verificationCmd,
			rawCaptureEnabled, retentionDaysForSummary,
			hashAlgo,
		)
	}

	if runtimeCfg.TelemetryEnabled {
		runDuration := time.Since(startAll)
		if err := recordTelemetry(appCtx, params.ID, chk.Name(), results, runDuration); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record telemetry: %v\n", err)
		}
	}

	return nil
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
			// Auto-detect if JavaScript is needed
			discovered, err = checker.DiscoverInScopeLinksAuto(ctx, target, jsCrawlOpts)
		} else if crawl.EnableJS {
			// Force JavaScript crawler
			discovered, err = checker.DiscoverInScopeLinksJS(ctx, target, jsCrawlOpts)
		} else {
			// Use static crawler only
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

// validateCheckParams validates common check command parameters
func validateCheckParams(params checkParams, appCtx *AppContext, auditAppendRaw bool, retentionDays int) error {
	if params.ID == "" {
		return fmt.Errorf("--id is required")
	}
	if err := validateEngagementID(params.ID); err != nil {
		return err
	}

	if !params.ROEConfirm {
		return fmt.Errorf("this action requires --roe-confirm to proceed (ensures explicit written authorization)")
	}

	if appCtx.Operator == "" {
		return fmt.Errorf("--operator is required")
	}

	if params.ComplianceMode {
		fmt.Println("[Compliance Mode] Enabled")

		if appCtx.Operator == "" {
			return fmt.Errorf("--operator required in compliance mode")
		}

		fmt.Println("-> Hash-signing of audit and result files enforced")

		// HTTP-specific validation
		if auditAppendRaw && retentionDays <= 0 {
			return fmt.Errorf("in compliance mode, --audit-append-raw requires --retention-days=<N>")
		}
	}

	return nil
}

// loadEngagementByID loads and validates an engagement by ID
func loadEngagementByID(id string) (*Engagement, error) {
	engs := loadEngagements()
	for i := range engs {
		if engs[i].ID == id {
			if len(engs[i].Scope) == 0 {
				return nil, &ScopeViolationError{Scope: id}
			}
			return &engs[i], nil
		}
	}
	return nil, &EngagementNotFoundError{ID: id}
}

// writeResultsAndHash writes results to JSON file, computes hashes, and returns paths and hashes
func writeResultsAndHash(appCtx *AppContext, id string, resultsFilename string, metadata RunMetadata, results []checker.CheckResult, startTime time.Time, hashAlgo HashAlgorithm) (resultsPath, auditPath, auditHash, resultsHash string, err error) {
	if _, err := ensureResultsDir(appCtx.ResultsDir, id); err != nil {
		return "", "", "", "", fmt.Errorf("failed to create results directory: %w", err)
	}

	// Write results JSON (first pass without audit hash)
	resultsPath, err = resolveResultsPath(appCtx.ResultsDir, id, resultsFilename)
	if err != nil {
		return "", "", "", "", fmt.Errorf("resolve results path: %w", err)
	}
	out := RunOutput{
		Metadata: metadata,
		Results:  results,
	}

	// Set completion time
	out.Metadata.CompleteAt = time.Now().UTC()
	out.Metadata.TotalTargets = len(results)

	b, err := json.MarshalIndent(out, jsonPrefix, jsonIndent)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(resultsPath, b, consts.DefaultFilePerm); err != nil {
		return "", "", "", "", fmt.Errorf("failed to write results: %w", err)
	}

	// Compute hash for audit.csv
	auditPath, err = resolveResultsPath(appCtx.ResultsDir, id, "audit.csv")
	if err != nil {
		return "", "", "", "", fmt.Errorf("resolve audit path: %w", err)
	}
	if err := ensureAuditFile(auditPath); err != nil {
		return "", "", "", "", fmt.Errorf("failed to initialize audit file: %w", err)
	}
	auditHash, err = HashFile(auditPath, hashAlgo)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to hash audit file: %w", err)
	}

	// Update metadata with audit hash and write final results JSON
	out.Metadata.AuditHash = auditHash
	if hashAlgo == HashAlgorithmSHA256 {
		out.Metadata.LegacyAuditHash = auditHash
	} else {
		out.Metadata.LegacyAuditHash = ""
	}
	if out.Metadata.HashAlgorithm == "" {
		out.Metadata.HashAlgorithm = hashAlgo.String()
	}
	b, err = json.MarshalIndent(out, jsonPrefix, jsonIndent)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to marshal final results: %w", err)
	}

	if err := os.WriteFile(resultsPath, b, consts.DefaultFilePerm); err != nil {
		return "", "", "", "", fmt.Errorf("failed to write final results: %w", err)
	}

	// Hash results.json AFTER final write
	resultsHash, err = HashFile(resultsPath, hashAlgo)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to hash results file: %w", err)
	}

	return resultsPath, auditPath, auditHash, resultsHash, nil
}

// signHashFiles signs the hash files using GPG
func signHashFiles(auditPath, resultsPath string, hashAlgo HashAlgorithm, gpgKey string) error {
	if gpgKey == "" {
		return fmt.Errorf("--gpg-key required with --auto-sign")
	}

	if err := validateGPGKey(gpgKey); err != nil {
		return fmt.Errorf("invalid gpg key: %w", err)
	}

	extension := hashAlgo.FileExtension()
	signFile := func(path string) error {
		cmd := exec.Command("gpg", "--armor", "--local-user", gpgKey, "--sign", path) // #nosec G204 -- arguments are validated and passed directly without shell expansion.
		cmd.Dir = filepath.Dir(path)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		return cmd.Run()
	}

	if err := signFile(auditPath + extension); err != nil {
		return fmt.Errorf("failed to sign audit hash file: %w", err)
	}

	if err := signFile(resultsPath + extension); err != nil {
		return fmt.Errorf("failed to sign results hash file: %w", err)
	}

	fmt.Printf("GPG signatures created for %s hash files.\n", hashAlgo.DisplayName())
	return nil
}

func getGPGFingerprint(gpgKey string) (string, error) {
	if gpgKey == "" {
		return "", errors.New("--gpg-key required to determine fingerprint")
	}
	if err := validateGPGKey(gpgKey); err != nil {
		return "", fmt.Errorf("invalid gpg key: %w", err)
	}
	cmd := exec.Command("gpg", "--with-colons", "--list-keys", gpgKey) // #nosec G204 -- controlled arguments, no shell expansion.
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to inspect GPG key %s: %w", gpgKey, err)
	}

	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.HasPrefix(line, "fpr:") {
			parts := strings.Split(line, ":")
			if len(parts) > 9 {
				return parts[9], nil
			}
		}
	}
	return "", fmt.Errorf("could not find fingerprint for %s", gpgKey)
}

func encryptAuditLog(auditPath, gpgKey string) (string, error) {
	if gpgKey == "" {
		return "", errors.New("--gpg-key required for secure results")
	}
	if err := validateGPGKey(gpgKey); err != nil {
		return "", fmt.Errorf("invalid gpg key: %w", err)
	}
	encryptedPath := auditPath + ".gpg"
	cmd := exec.Command("gpg", "--yes", "--recipient", gpgKey, "--output", encryptedPath, "--encrypt", auditPath) // #nosec G204 -- key validated and passed as argv, not via shell.
	cmd.Dir = filepath.Dir(auditPath)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to encrypt audit log: %w", err)
	}
	fmt.Printf("Encrypted audit log created at %s\n", encryptedPath)
	return encryptedPath, nil
}

// printComplianceSummary prints the compliance mode summary
func printComplianceSummary(appCtx *AppContext, eng *Engagement, auditHash, resultsHash string, verificationCmd string, auditAppendRaw bool, retentionDays int, hashAlgo HashAlgorithm) {
	border := strings.Repeat("-", 54)
	fmt.Println(colorInfo(border))
	fmt.Println(colorInfo("🔒 Compliance Summary"))
	fmt.Printf("%s %s\n", colorInfo("Operator:"), appCtx.Operator)
	fmt.Printf("%s %s (%s)\n", colorInfo("Engagement:"), eng.Name, eng.ID)
	fmt.Printf("%s audit hash : %s\n", hashAlgo.DisplayName(), auditHash)
	fmt.Printf("%s results hash: %s\n", hashAlgo.DisplayName(), resultsHash)
	fmt.Printf("%s %s\n", colorInfo("Verification:"), verificationCmd)
	if auditAppendRaw {
		fmt.Printf("%s raw captures must be deleted or anonymized after %d day(s).\n", colorWarn("Retention:"), retentionDays)
	}
	fmt.Println(colorSuccess("Evidence integrity and retention requirements satisfied."))
	fmt.Println(colorInfo(border))
}

// addCommonCheckFlags adds flags that are common to all check commands.
// This reduces duplication and ensures consistent flag definitions across commands.
func addCommonCheckFlags(cmd *cobra.Command) {
	cmd.Flags().String("id", "", "Engagement id")
	cmd.Flags().Bool("roe-confirm", false, "Confirm you have explicit written authorization (required)")
	cmd.Flags().Bool("compliance-mode", false, "Enable compliance enforcement (hashing, retention checks)")
	cmd.Flags().BoolVar(&cliConfig.Check.AutoSign, "auto-sign", cliConfig.Check.AutoSign, "Automatically sign .sha256 files using configured GPG key")
	cmd.Flags().StringVar(&cliConfig.Check.GPGKey, "gpg-key", cliConfig.Check.GPGKey, "GPG key ID or email for signing (required if --auto-sign)")
}

func init() {
	// Global check flags (apply to all subcommands)
	checkCmd.PersistentFlags().IntVarP(&cliConfig.Check.Concurrency, "concurrency", "c", cliConfig.Check.Concurrency, "max concurrent requests")
	checkCmd.PersistentFlags().IntVarP(&cliConfig.Check.RateLimit, "rate", "r", cliConfig.Check.RateLimit, "requests per second (global)")
	checkCmd.PersistentFlags().IntVarP(&cliConfig.Check.TimeoutSecs, "timeout", "t", cliConfig.Check.TimeoutSecs, "request timeout in seconds")
	checkCmd.PersistentFlags().BoolVar(&cliConfig.Check.TelemetryEnabled, "telemetry", cliConfig.Check.TelemetryEnabled, "Record telemetry metrics (durations, success rates)")
	checkCmd.PersistentFlags().BoolVar(&cliConfig.Check.ProgressEnabled, "progress", cliConfig.Check.ProgressEnabled, "Display live progress for checks")
	checkCmd.PersistentFlags().StringVar(&cliConfig.Check.HashAlgorithm, "hash", cliConfig.Check.HashAlgorithm, "Hash algorithm for integrity verification (sha256|sha512)")
	checkCmd.PersistentFlags().BoolVar(&cliConfig.Check.SecureResults, "secure-results", cliConfig.Check.SecureResults, "Encrypt audit logs with operator GPG key after run")
	checkCmd.PersistentFlags().IntVar(&cliConfig.Check.RetryCount, "retry", cliConfig.Check.RetryCount, "Number of times to retry failed targets")

	// HTTP-specific flags
	addCommonCheckFlags(checkHTTPCmd)
	checkHTTPCmd.Flags().BoolVar(&cliConfig.Check.AuditAppendRaw, "audit-append-raw", cliConfig.Check.AuditAppendRaw, "Save limited raw headers/body for auditing (handle carefully)")
	checkHTTPCmd.Flags().IntVar(&cliConfig.Check.RetentionDays, "retention-days", cliConfig.Check.RetentionDays, "Retention period (days) for raw captures; required in compliance mode if --audit-append-raw is used")
	checkHTTPCmd.Flags().BoolVar(&cliConfig.Check.Crawl.Enabled, "crawl", cliConfig.Check.Crawl.Enabled, "Discover same-host links (auto-detects JavaScript/SPA sites)")
	checkHTTPCmd.Flags().IntVar(&cliConfig.Check.Crawl.MaxDepth, "crawl-depth", cliConfig.Check.Crawl.MaxDepth, "Maximum link depth to follow per target")
	checkHTTPCmd.Flags().IntVar(&cliConfig.Check.Crawl.MaxPages, "crawl-max-pages", cliConfig.Check.Crawl.MaxPages, "Maximum additional pages to discover per target")
	checkHTTPCmd.Flags().BoolVar(&cliConfig.Check.Crawl.EnableJS, "crawl-force-js", cliConfig.Check.Crawl.EnableJS, "Force JavaScript crawler for all targets (overrides auto-detection)")
	checkHTTPCmd.Flags().IntVar(&cliConfig.Check.Crawl.JSWaitTime, "crawl-js-wait", cliConfig.Check.Crawl.JSWaitTime, "Seconds to wait for JavaScript to render (when JS is used)")

	// DNS-specific flags
	addCommonCheckFlags(checkDNSCmd)
	checkDNSCmd.Flags().StringSliceVar(&cliConfig.Check.DNS.Nameservers, "nameservers", cliConfig.Check.DNS.Nameservers, "Custom DNS nameservers (e.g., 8.8.8.8:53,1.1.1.1:53)")
	checkDNSCmd.Flags().IntVar(&cliConfig.Check.DNS.Timeout, "dns-timeout", cliConfig.Check.DNS.Timeout, "DNS query timeout in seconds")

	// Network-specific flags
	addCommonCheckFlags(checkNetworkCmd)
	checkNetworkCmd.Flags().BoolVar(&cliConfig.Check.Network.EnablePortScan, "enable-port-scan", cliConfig.Check.Network.EnablePortScan, "Scan TCP ports for exposure and banner details")
	checkNetworkCmd.Flags().IntSliceVar(&cliConfig.Check.Network.Ports, "ports", cliConfig.Check.Network.Ports, "Comma-separated list of TCP ports to scan (defaults to built-in set)")
	checkNetworkCmd.Flags().IntVar(&cliConfig.Check.Network.PortScanTimeout, "port-scan-timeout", cliConfig.Check.Network.PortScanTimeout, "Per-port scan timeout in seconds")
	checkNetworkCmd.Flags().IntVar(&cliConfig.Check.Network.MaxPortWorkers, "port-workers", cliConfig.Check.Network.MaxPortWorkers, "Concurrent port scan workers")
	checkNetworkCmd.Flags().BoolVar(&cliConfig.Check.Crawl.Enabled, "crawl", cliConfig.Check.Crawl.Enabled, "Discover same-host links (auto-detects JavaScript/SPA sites)")
	checkNetworkCmd.Flags().IntVar(&cliConfig.Check.Crawl.MaxDepth, "crawl-depth", cliConfig.Check.Crawl.MaxDepth, "Maximum link depth to follow per target")
	checkNetworkCmd.Flags().IntVar(&cliConfig.Check.Crawl.MaxPages, "crawl-max-pages", cliConfig.Check.Crawl.MaxPages, "Maximum additional pages to discover per target")
	checkNetworkCmd.Flags().BoolVar(&cliConfig.Check.Crawl.EnableJS, "crawl-force-js", cliConfig.Check.Crawl.EnableJS, "Force JavaScript crawler for all targets (overrides auto-detection)")
	checkNetworkCmd.Flags().IntVar(&cliConfig.Check.Crawl.JSWaitTime, "crawl-js-wait", cliConfig.Check.Crawl.JSWaitTime, "Seconds to wait for JavaScript to render (when JS is used)")

	checkCmd.AddCommand(checkHTTPCmd)
	checkCmd.AddCommand(checkDNSCmd)
	checkCmd.AddCommand(checkNetworkCmd)
	registerPluginCommands()
}
