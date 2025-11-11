package checker

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// CheckResult represents the result of a single target check
type CheckResult struct {
	Target          string                 `json:"target"`
	CheckedAt       time.Time              `json:"checked_at"`
	Status          string                 `json:"status"`
	HTTPStatus      int                    `json:"http_status,omitempty"`
	ServerHeader    string                 `json:"server_header,omitempty"`
	TLSExpiry       string                 `json:"tls_expiry,omitempty"`
	DNSRecords      map[string]interface{} `json:"dns_records,omitempty"`
	ResponseTime    float64                `json:"response_time_ms,omitempty"`
	SecurityHeaders *SecurityHeadersResult `json:"security_headers,omitempty"`
	TLSCompliance   *TLSComplianceResult   `json:"tls_compliance,omitempty"`
	Notes           string                 `json:"notes,omitempty"`
	Error           string                 `json:"error,omitempty"`
}

// SecurityHeadersResult contains security headers analysis
type SecurityHeadersResult struct {
	Score            int                       `json:"score"`
	Grade            string                    `json:"grade"`
	MaxScore         int                       `json:"max_score"`
	Headers          map[string]HeaderStatus   `json:"headers"`
	Missing          []string                  `json:"missing"`
	Warnings         []string                  `json:"warnings,omitempty"`
	Recommendations  []string                  `json:"recommendations,omitempty"`
}

// HeaderStatus represents the status of a single security header
type HeaderStatus struct {
	Present       bool     `json:"present"`
	Value         string   `json:"value,omitempty"`
	Severity      string   `json:"severity"`
	Score         int      `json:"score"`
	MaxScore      int      `json:"max_score"`
	Issues        []string `json:"issues,omitempty"`
	Recommendation string   `json:"recommendation,omitempty"`
}

// TLSComplianceResult contains TLS/crypto compliance analysis (OWASP ASVS §9, PCI DSS 4.1)
type TLSComplianceResult struct {
	Compliant        bool                  `json:"compliant"`
	TLSVersion       string                `json:"tls_version"`
	CipherSuite      string                `json:"cipher_suite"`
	Protocol         string                `json:"protocol"`
	Issues           []ComplianceIssue     `json:"issues,omitempty"`
	Recommendations  []string              `json:"recommendations,omitempty"`
	Standards        ComplianceStandards   `json:"standards"`
	CertificateInfo  *CertificateInfo      `json:"certificate_info,omitempty"`
}

// ComplianceIssue represents a compliance violation
type ComplianceIssue struct {
	Standard    string `json:"standard"`      // "OWASP ASVS 9.1.3", "PCI DSS 4.1"
	Requirement string `json:"requirement"`   // Specific requirement number
	Severity    string `json:"severity"`      // "critical", "high", "medium", "low"
	Description string `json:"description"`   // Issue description
	Remediation string `json:"remediation"`   // How to fix
}

// ComplianceStandards tracks compliance with specific standards
type ComplianceStandards struct {
	OWASPASVS9 ComplianceStatus `json:"owasp_asvs_v9"`
	PCIDSS41   ComplianceStatus `json:"pci_dss_4_1"`
}

// ComplianceStatus represents compliance status for a standard
type ComplianceStatus struct {
	Compliant   bool     `json:"compliant"`
	Level       string   `json:"level,omitempty"`        // For ASVS: "L1", "L2", "L3"
	Passed      []string `json:"passed,omitempty"`       // Passed requirements
	Failed      []string `json:"failed,omitempty"`       // Failed requirements
	Score       int      `json:"score,omitempty"`        // Optional scoring
}

// CertificateInfo contains certificate validation details
type CertificateInfo struct {
	Subject           string   `json:"subject"`
	Issuer            string   `json:"issuer"`
	NotBefore         string   `json:"not_before"`
	NotAfter          string   `json:"not_after"`
	DNSNames          []string `json:"dns_names,omitempty"`
	SelfSigned        bool     `json:"self_signed"`
	ValidChain        bool     `json:"valid_chain"`
	DaysUntilExpiry   int      `json:"days_until_expiry"`
	SignatureAlg      string   `json:"signature_algorithm"`
	PublicKeyAlg      string   `json:"public_key_algorithm"`
	KeySize           int      `json:"key_size,omitempty"`
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
