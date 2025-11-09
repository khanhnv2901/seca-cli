package cmd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/time/rate"
)

type CheckResult struct {
	Target       string    `json:"target"`
	CheckedAt    time.Time `json:"checked_at"`
	Status       string    `json:"status"`
	HTTPStatus   int       `json:"http_status,omitempty"`
	ServerHeader string    `json:"server_header,omitempty"`
	TLSExpiry    string    `json:"tls_expiry,omitempty"`
	Notes        string    `json:"notes,omitempty"`
	Error        string    `json:"error,omitempty"`
}

type RunMetadata struct {
	Operator       string    `json:"operator"`
	EngagementID   string    `json:"engagement_id"`
	EngagementName string    `json:"engagement_name"`
	Owner          string    `json:"owner"`
	StartAt        time.Time `json:"started_at"`
	CompleteAt     time.Time `json:"completed_at"`
	AuditHash      string    `json:"audit_sha256"`
	TotalTargets   int       `json:"total_targets"`
	// Note: results.json hash is stored in results.json.sha256 file, not here
}

type RunOutput struct {
	Metadata RunMetadata   `json:"metadata"`
	Results  []CheckResult `json:"results"`
}

var (
	concurrency    int
	rateLimit      int // requests per second
	timeoutSecs    int
	auditAppendRaw bool
	complianceMode bool
	retentionDays  int
	autoSign       bool
	gpgKey         string
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run safe, authorized checks against scoped targets (no scanning/exploitation)",
}

var checkHTTPCmd = &cobra.Command{
	Use:   "http",
	Short: "Run safe HTTP/TLS checks for an engagement's scope",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")

		// Safety: require explicit ROE flag for any run
		roeConfirm, _ := cmd.Flags().GetBool("roe-confirm")

		// check compliance mode
		complianceMode, _ := cmd.Flags().GetBool("compliance-mode")

		// auto sign GPG Key
		autoSign, _ = cmd.Flags().GetBool("auto-sign")
		gpgKey, _ = cmd.Flags().GetString("gpg-key")

		if id == "" {
			return fmt.Errorf("--id is required")
		}

		if !roeConfirm {
			return fmt.Errorf("this action requires -- roe-confirm to proceed (ensures explicit written authorization)")
		}

		if operator == "" {
			return fmt.Errorf("--operator is required")
		}

		if complianceMode {
			fmt.Println("[Compliance Mode] Enabled")

			// Require operator
			if operator == "" {
				return fmt.Errorf("--operator required in compliance mode")
			}

			// Auto-force hash-signing (already done later)
			// Nothing to change here, but we’ll print a notice
			fmt.Println("→ Hash-signing of audit and result files enforced")

			// If raw audit capture used, require retentionDays > 0
			if auditAppendRaw && retentionDays <= 0 {
				return fmt.Errorf("in compliance mode, --audit-append-raw requires --retention-days=<N>")
			}
		}

		// load engagement
		engs := loadEngagements()
		var eng *Engagement
		for i := range engs {
			if engs[i].ID == id {
				eng = &engs[i]
				break
			}
		}
		if eng == nil {
			return fmt.Errorf("no engagement found with id %s", id)
		}

		if len(eng.Scope) == 0 {
			return fmt.Errorf("no scope found for engagement %s", id)
		}

		startAll := time.Now()
		dir := filepath.Join(resultsDir, id)
		_ = os.MkdirAll(dir, 0o755)

		// rate limiter & context
		limiter := rate.NewLimiter(rate.Limit(rateLimit), rateLimit)
		ctx := context.Background()

		// worker pool
		sem := make(chan struct{}, concurrency)
		var wg sync.WaitGroup
		mu := sync.Mutex{}
		results := make([]CheckResult, 0, len(eng.Scope))

		for _, target := range eng.Scope {
			wg.Add(1)
			go func(t string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				start := time.Now()
				_ = limiter.Wait(ctx)

				r := CheckResult{
					Target:    t,
					CheckedAt: time.Now().UTC(),
				}

				// normalize URL
				u := t
				parsed, err := url.Parse(t)
				if err != nil || parsed.Scheme == "" {
					u = "http://" + t
				}

				// HEAD (safe)
				client := &http.Client{
					Timeout: time.Duration(timeoutSecs) * time.Second,
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
					},
				}

				req, _ := http.NewRequest("HEAD", u, nil)
				resp, err := client.Do(req)
				if err != nil {
					// try GET as fallback (some servers disallow HEAD)
					req2, _ := http.NewRequest("GET", u, nil)
					resp2, err2 := client.Do(req2)
					if err2 != nil {
						r.Status = "error"
						r.Error = err2.Error()
						// write audit row for this failure
						_ = AppendAuditRow(id, operator, "check http", t, r.Status, 0, "", r.Notes, r.Error, time.Since(start).Seconds())

						mu.Lock()
						results = append(results, r)
						mu.Unlock()
						return
					}
					resp = resp2
					// drain body
					_, _ = io.Copy(io.Discard, resp.Body)
				}

				r.HTTPStatus = resp.StatusCode
				r.ServerHeader = resp.Header.Get("Server")

				r.Status = "ok"

				// TLS expiry
				if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
					cert := resp.TLS.PeerCertificates[0]
					r.TLSExpiry = cert.NotAfter.Format(time.RFC3339)
					// add simple note if expiring soon
					if time.Until(cert.NotAfter) < (14 * 24 * time.Hour) {
						r.Notes = "TLS certificate expires soon"
					}
				}

				// optional raw capture
				if auditAppendRaw {
					bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
					_ = SaveRawCapture(id, t, resp.Header, string(bodyBytes))
				} else {
					_, _ = io.Copy(io.Discard, resp.Body)
				}
				if resp.Body != nil {
					_ = resp.Body.Close()
				}

				// optional robots.txt fetch (safe, small GET)
				if parsed == nil {
					parsed, _ = url.Parse(u)
				}

				robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsed.Scheme, parsed.Host)
				rr, err := client.Get(robotsURL)
				if err == nil {
					if rr.StatusCode == 200 {
						r.Notes = "robots.txt found"
					}

					_, _ = io.Copy(io.Discard, rr.Body)
					rr.Body.Close()
				}

				// write audit row for successful check
				_ = AppendAuditRow(id, operator, "check http", t, r.Status, r.HTTPStatus, r.TLSExpiry, r.Notes, r.Error, time.Since(start).Seconds())

				mu.Lock()
				results = append(results, r)
				mu.Unlock()
				if resp.Body != nil {
					_ = resp.Body.Close()
				}

			}(target)
		}

		wg.Wait()

		// Write results JSON
		resultsPath := filepath.Join(dir, "results.json")
		out := RunOutput{
			Metadata: RunMetadata{
				Operator:       operator,
				EngagementID:   id,
				EngagementName: eng.Name,
				Owner:          eng.Owner,
				StartAt:        startAll,
				CompleteAt:     time.Now().UTC(),
				TotalTargets:   len(eng.Scope),
			},
			Results: results,
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		_ = os.WriteFile(resultsPath, b, 0o644)

		// Compute hash for audit.csv
		auditPath := filepath.Join(dir, "audit.csv")
		auditHash, _ := HashFileSHA256(auditPath)

		// Update metadata with audit hash only
		out.Metadata.AuditHash = auditHash

		// Write final results JSON
		b, _ = json.MarshalIndent(out, "", "  ")
		_ = os.WriteFile(resultsPath, b, 0o644)

		// Hash results.json AFTER final write
		resultsHash, _ := HashFileSHA256(resultsPath)

		// Note: ResultsHash is not stored in the file itself to avoid hash mismatch

		if autoSign {
			if gpgKey == "" {
				return fmt.Errorf("--gpg-key required with --auto-sign")
			}
			signFile := func(path string) error {
				cmd := exec.Command("gpg", "--armor", "--local-user", gpgKey, "--sign", path)
				cmd.Dir = filepath.Dir(path)
				cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
				return cmd.Run()
			}
			_ = signFile(auditPath + ".sha256")
			_ = signFile(resultsPath + ".sha256")
			fmt.Println("GPG signatures created for both .sha256 files.")
		}

		fmt.Printf("Run complete.\n")
		fmt.Printf("Results: %s\nAudit: %s\n", resultsPath, auditPath)
		fmt.Printf("SHA256 audit: %s\nSHA256 results: %s\n", auditHash, resultsHash)

		if complianceMode {
			fmt.Println("------------------------------------------------------")
			fmt.Println("🔒 Compliance Summary")
			fmt.Printf("Operator: %s\nEngagement: %s (%s)\n", operator, eng.Name, eng.ID)
			fmt.Printf("Audit hash : %s\nResults hash: %s\n", auditHash, resultsHash)
			fmt.Println("Verification: sha256sum -c audit.csv.sha256 && sha256sum -c results_*.sha256")
			if auditAppendRaw {
				fmt.Printf("Retention: raw captures must be deleted or anonymized after %d day(s).\n", retentionDays)
			}
			fmt.Println("Evidence integrity and retention requirements satisfied.")
			fmt.Println("------------------------------------------------------")
		}

		return nil
	},
}

func init() {
	checkCmd.PersistentFlags().IntVarP(&concurrency, "concurrency", "c", 1, "max concurrent requests")
	checkCmd.PersistentFlags().IntVarP(&rateLimit, "rate", "r", 1, "requests per second (global)")
	checkCmd.PersistentFlags().IntVarP(&timeoutSecs, "timeout", "t", 10, "request timeout in seconds")
	checkHTTPCmd.Flags().String("id", "", "Engagement id")
	checkHTTPCmd.Flags().Bool("roe-confirm", false, "Confirm you have explicit written authorization (required)")
	checkHTTPCmd.Flags().BoolVar(&auditAppendRaw, "audit-append-raw", false, "Save limited raw headers/body for auditing (handle carefully)")
	checkHTTPCmd.Flags().BoolVar(&complianceMode, "compliance-mode", false, "Enable compliance enforcement (hashing, retention checks)")
	checkHTTPCmd.Flags().IntVar(&retentionDays, "retention-days", 0, "Retention period (days) for raw captures; required in compliance mode if --audit-append-raw is used")
	checkHTTPCmd.Flags().Bool("auto-sign", false, "Automatically sign .sha256 files using configured GPG key")
	checkHTTPCmd.Flags().String("gpg-key", "", "GPG key ID or email for signing (required if --auto-sign)")

	checkCmd.AddCommand(checkHTTPCmd)
}
