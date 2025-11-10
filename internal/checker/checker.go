package checker

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// CheckResult represents the result of a single target check
type CheckResult struct {
	Target       string                 `json:"target"`
	CheckedAt    time.Time              `json:"checked_at"`
	Status       string                 `json:"status"`
	HTTPStatus   int                    `json:"http_status,omitempty"`
	ServerHeader string                 `json:"server_header,omitempty"`
	TLSExpiry    string                 `json:"tls_expiry,omitempty"`
	DNSRecords   map[string]interface{} `json:"dns_records,omitempty"`
	ResponseTime float64                `json:"response_time_ms,omitempty"`
	Notes        string                 `json:"notes,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// Checker is the interface that all check implementations must satisfy
type Checker interface {
	// Check performs the actual check logic for a single target
	Check(ctx context.Context, target string) CheckResult

	// Name returns the name of this checker (e.g., "check http", "check dns")
	Name() string
}

// AuditFunc is a callback function to log audit information
type AuditFunc func(target string, result CheckResult, duration float64) error

// Runner orchestrates the execution of checks with concurrency and rate limiting
type Runner struct {
	Concurrency int           // Maximum number of concurrent checks
	RateLimit   int           // Requests per second (global)
	Timeout     time.Duration // Timeout for each check
}

// RunChecks executes checks against multiple targets using a worker pool
func (r *Runner) RunChecks(ctx context.Context, targets []string, checker Checker, auditFn AuditFunc) []CheckResult {
	// Rate limiter
	limiter := rate.NewLimiter(rate.Limit(r.RateLimit), r.RateLimit)

	// Worker pool
	sem := make(chan struct{}, r.Concurrency)
	var wg sync.WaitGroup
	mu := sync.Mutex{}
	results := make([]CheckResult, 0, len(targets))

	for _, target := range targets {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Wait for rate limiter
			_ = limiter.Wait(ctx)

			start := time.Now()

			// Create context with timeout
			checkCtx, cancel := context.WithTimeout(ctx, r.Timeout)
			defer cancel()

			// Perform the check
			result := checker.Check(checkCtx, t)

			duration := time.Since(start).Seconds()

			// Call audit function if provided
			if auditFn != nil {
				_ = auditFn(t, result, duration)
			}

			// Append result
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(target)
	}

	wg.Wait()
	return results
}
