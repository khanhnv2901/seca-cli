package check

import (
	"errors"
	"time"
)

// Result represents the outcome of checking a single target
type Result struct {
	target       string
	status       CheckStatus
	httpStatus   int
	tlsExpiry    time.Time
	checkedAt    time.Time
	responseTime float64
	error        string
	findings     Findings
}

// CheckStatus represents the status of a check
type CheckStatus string

const (
	CheckStatusOK    CheckStatus = "ok"
	CheckStatusError CheckStatus = "error"
)

// Findings contains all security findings for a target
type Findings struct {
	SecurityHeaders  *SecurityHeadersResult
	TLSCompliance    *TLSComplianceResult
	NetworkSecurity  *NetworkSecurityResult
	ClientSecurity   *ClientSecurityResult
	CORS             *CORSReport
	Cookies          []CookieFinding
	CachePolicy      *CachePolicy
	Vulnerabilities  []VulnerabilityFinding
}

// SecurityHeadersResult represents security header analysis
type SecurityHeadersResult struct {
	Score           int
	Grade           string
	HeadersPresent  map[string]string
	HeadersMissing  []string
	Recommendations []string
}

// TLSComplianceResult represents TLS compliance analysis
type TLSComplianceResult struct {
	Compliant         bool
	Version           string
	CipherSuite       string
	Protocol          string
	CertificateValid  bool
	CertificateExpiry time.Time
	CertificateChain  []string
	Issues            []string
}

// NetworkSecurityResult represents network security findings
type NetworkSecurityResult struct {
	OpenPorts           []int
	SubdomainTakeover   bool
	SubdomainProvider   string
	ServiceFingerprints map[int]string
	RiskLevel           string
}

// ClientSecurityResult represents client-side security findings
type ClientSecurityResult struct {
	VulnerableLibraries []VulnerableLibrary
	CSRFProtection      bool
	TrustedTypes        bool
}

// VulnerableLibrary represents a vulnerable JavaScript library
type VulnerableLibrary struct {
	Name     string
	Version  string
	CVEs     []string
	CVSS     float64
	Severity string
}

// CORSReport represents CORS configuration analysis
type CORSReport struct {
	AllowsAnyOrigin     bool
	AllowedOrigins      []string
	AllowedMethods      []string
	AllowCredentials    bool
	MaxAge              int
	MissingOriginPolicy bool
}

// CookieFinding represents a cookie security issue
type CookieFinding struct {
	Name           string
	MissingSecure  bool
	MissingHTTPOnly bool
	SameSite       string
}

// CachePolicy represents cache configuration analysis
type CachePolicy struct {
	CacheControl string
	Expires      string
	Pragma       string
	IsPrivate    bool
}

// VulnerabilityFinding represents a known vulnerability
type VulnerabilityFinding struct {
	CVE         string
	CVSS        float64
	Severity    string
	Description string
	Remediation string
}

// NewResult creates a new check result
func NewResult(target string, status CheckStatus) (*Result, error) {
	if target == "" {
		return nil, errors.New("target cannot be empty")
	}

	return &Result{
		target:    target,
		status:    status,
		checkedAt: time.Now(),
		findings:  Findings{},
	}, nil
}

// Business methods

// SetHTTPStatus sets the HTTP status code
func (r *Result) SetHTTPStatus(status int) {
	r.httpStatus = status
}

// SetTLSExpiry sets the TLS certificate expiry
func (r *Result) SetTLSExpiry(expiry time.Time) {
	r.tlsExpiry = expiry
}

// SetResponseTime sets the response time in milliseconds
func (r *Result) SetResponseTime(ms float64) {
	r.responseTime = ms
}

// SetError sets the error message
func (r *Result) SetError(err string) {
	r.error = err
	r.status = CheckStatusError
}

// AddSecurityHeadersFindings adds security header findings
func (r *Result) AddSecurityHeadersFindings(findings *SecurityHeadersResult) {
	r.findings.SecurityHeaders = findings
}

// AddTLSComplianceFindings adds TLS compliance findings
func (r *Result) AddTLSComplianceFindings(findings *TLSComplianceResult) {
	r.findings.TLSCompliance = findings
}

// AddNetworkSecurityFindings adds network security findings
func (r *Result) AddNetworkSecurityFindings(findings *NetworkSecurityResult) {
	r.findings.NetworkSecurity = findings
}

// AddClientSecurityFindings adds client security findings
func (r *Result) AddClientSecurityFindings(findings *ClientSecurityResult) {
	r.findings.ClientSecurity = findings
}

// AddCORSFindings adds CORS findings
func (r *Result) AddCORSFindings(findings *CORSReport) {
	r.findings.CORS = findings
}

// AddCookieFinding adds a cookie security finding
func (r *Result) AddCookieFinding(finding CookieFinding) {
	r.findings.Cookies = append(r.findings.Cookies, finding)
}

// AddVulnerabilityFinding adds a vulnerability finding
func (r *Result) AddVulnerabilityFinding(finding VulnerabilityFinding) {
	r.findings.Vulnerabilities = append(r.findings.Vulnerabilities, finding)
}

// HasCriticalFindings checks if there are any critical security findings
func (r *Result) HasCriticalFindings() bool {
	// Check for critical TLS issues
	if r.findings.TLSCompliance != nil && !r.findings.TLSCompliance.Compliant {
		return true
	}

	// Check for subdomain takeover
	if r.findings.NetworkSecurity != nil && r.findings.NetworkSecurity.SubdomainTakeover {
		return true
	}

	// Check for high-severity vulnerabilities
	for _, vuln := range r.findings.Vulnerabilities {
		if vuln.Severity == "CRITICAL" || vuln.CVSS >= 9.0 {
			return true
		}
	}

	return false
}

// Getters

func (r *Result) Target() string {
	return r.target
}

func (r *Result) Status() CheckStatus {
	return r.status
}

func (r *Result) HTTPStatus() int {
	return r.httpStatus
}

func (r *Result) TLSExpiry() time.Time {
	return r.tlsExpiry
}

func (r *Result) CheckedAt() time.Time {
	return r.checkedAt
}

func (r *Result) ResponseTime() float64 {
	return r.responseTime
}

func (r *Result) Error() string {
	return r.error
}

func (r *Result) Findings() Findings {
	return r.findings
}
