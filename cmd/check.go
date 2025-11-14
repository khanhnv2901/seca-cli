package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/infrastructure/checker"
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

func init() {
	checkCmd.PersistentFlags().IntVarP(&cliConfig.Check.Concurrency, "concurrency", "c", cliConfig.Check.Concurrency, "max concurrent requests")
	checkCmd.PersistentFlags().IntVarP(&cliConfig.Check.RateLimit, "rate", "r", cliConfig.Check.RateLimit, "requests per second (global)")
	checkCmd.PersistentFlags().IntVarP(&cliConfig.Check.TimeoutSecs, "timeout", "t", cliConfig.Check.TimeoutSecs, "request timeout in seconds")
	checkCmd.PersistentFlags().BoolVar(&cliConfig.Check.TelemetryEnabled, "telemetry", cliConfig.Check.TelemetryEnabled, "Record telemetry metrics (durations, success rates)")
	checkCmd.PersistentFlags().BoolVar(&cliConfig.Check.ProgressEnabled, "progress", cliConfig.Check.ProgressEnabled, "Display live progress for checks")
	checkCmd.PersistentFlags().StringVar(&cliConfig.Check.HashAlgorithm, "hash", cliConfig.Check.HashAlgorithm, "Hash algorithm for integrity verification (sha256|sha512)")
	checkCmd.PersistentFlags().BoolVar(&cliConfig.Check.SecureResults, "secure-results", cliConfig.Check.SecureResults, "Encrypt audit logs with operator GPG key after run")
	checkCmd.PersistentFlags().IntVar(&cliConfig.Check.RetryCount, "retry", cliConfig.Check.RetryCount, "Number of times to retry failed targets")

	checkCmd.AddCommand(checkHTTPCmdDDD)
	checkCmd.AddCommand(checkDNSCmdDDD)
	checkCmd.AddCommand(checkNetworkCmdDDD)
	registerPluginCommands()
}
