package json

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/domain/check"
	sharedErrors "github.com/khanhnv2901/seca-cli/internal/shared/errors"
	"github.com/khanhnv2901/seca-cli/internal/shared/security"
)

// checkRunDTO is the data transfer object for JSON serialization
type checkRunDTO struct {
	ID             string           `json:"id"`
	EngagementID   string           `json:"engagement_id"`
	EngagementName string           `json:"engagement_name"`
	Operator       string           `json:"operator"`
	StartedAt      string           `json:"started_at"`
	CompletedAt    string           `json:"completed_at,omitempty"`
	Status         string           `json:"status"`
	Results        []resultDTO      `json:"results"`
	Metadata       metadataDTO      `json:"metadata"`
}

type metadataDTO struct {
	AuditHash            string `json:"audit_hash,omitempty"`
	HashAlgorithm        string `json:"hash_algorithm,omitempty"`
	SignatureFingerprint string `json:"signature_fingerprint,omitempty"`
	TotalTargets         int    `json:"total_targets"`
}

type resultDTO struct {
	Target       string       `json:"target"`
	Status       string       `json:"status"`
	HTTPStatus   int          `json:"http_status,omitempty"`
	TLSExpiry    string       `json:"tls_expiry,omitempty"`
	CheckedAt    string       `json:"checked_at"`
	ResponseTime float64      `json:"response_time_ms,omitempty"`
	Error        string       `json:"error,omitempty"`
	Findings     findingsDTO  `json:"findings,omitempty"`
}

type findingsDTO struct {
	SecurityHeaders  *securityHeadersDTO  `json:"security_headers,omitempty"`
	TLSCompliance    *tlsComplianceDTO    `json:"tls_compliance,omitempty"`
	NetworkSecurity  *networkSecurityDTO  `json:"network_security,omitempty"`
	ClientSecurity   *clientSecurityDTO   `json:"client_security,omitempty"`
	CORS             *corsReportDTO       `json:"cors,omitempty"`
	Cookies          []cookieFindingDTO   `json:"cookies,omitempty"`
	CachePolicy      *cachePolicyDTO      `json:"cache_policy,omitempty"`
	Vulnerabilities  []vulnerabilityDTO   `json:"vulnerabilities,omitempty"`
}

type securityHeadersDTO struct {
	Score           int               `json:"score"`
	Grade           string            `json:"grade"`
	HeadersPresent  map[string]string `json:"headers_present"`
	HeadersMissing  []string          `json:"headers_missing"`
	Recommendations []string          `json:"recommendations"`
}

type tlsComplianceDTO struct {
	Compliant         bool     `json:"compliant"`
	Version           string   `json:"version"`
	CipherSuite       string   `json:"cipher_suite"`
	Protocol          string   `json:"protocol"`
	CertificateValid  bool     `json:"certificate_valid"`
	CertificateExpiry string   `json:"certificate_expiry"`
	CertificateChain  []string `json:"certificate_chain"`
	Issues            []string `json:"issues"`
}

type networkSecurityDTO struct {
	OpenPorts           []int             `json:"open_ports"`
	SubdomainTakeover   bool              `json:"subdomain_takeover"`
	SubdomainProvider   string            `json:"subdomain_provider,omitempty"`
	ServiceFingerprints map[int]string    `json:"service_fingerprints"`
	RiskLevel           string            `json:"risk_level"`
}

type clientSecurityDTO struct {
	VulnerableLibraries []vulnerableLibraryDTO `json:"vulnerable_libraries"`
	CSRFProtection      bool                   `json:"csrf_protection"`
	TrustedTypes        bool                   `json:"trusted_types"`
}

type vulnerableLibraryDTO struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	CVEs     []string `json:"cves"`
	CVSS     float64  `json:"cvss"`
	Severity string   `json:"severity"`
}

type corsReportDTO struct {
	AllowsAnyOrigin     bool     `json:"allows_any_origin"`
	AllowedOrigins      []string `json:"allowed_origins"`
	AllowedMethods      []string `json:"allowed_methods"`
	AllowCredentials    bool     `json:"allow_credentials"`
	MaxAge              int      `json:"max_age"`
	MissingOriginPolicy bool     `json:"missing_origin_policy"`
}

type cookieFindingDTO struct {
	Name            string `json:"name"`
	MissingSecure   bool   `json:"missing_secure"`
	MissingHTTPOnly bool   `json:"missing_httponly"`
	SameSite        string `json:"same_site"`
}

type cachePolicyDTO struct {
	CacheControl string `json:"cache_control"`
	Expires      string `json:"expires"`
	Pragma       string `json:"pragma"`
	IsPrivate    bool   `json:"is_private"`
}

type vulnerabilityDTO struct {
	CVE         string  `json:"cve"`
	CVSS        float64 `json:"cvss"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Remediation string  `json:"remediation"`
}

// CheckRunRepository implements the check.Repository interface using JSON file storage
type CheckRunRepository struct {
	resultsDir string
	mu         sync.RWMutex
}

// NewCheckRunRepository creates a new JSON-based check run repository
func NewCheckRunRepository(resultsDir string) (*CheckRunRepository, error) {
	if resultsDir == "" {
		return nil, fmt.Errorf("results directory cannot be empty")
	}

	// Ensure the results directory exists
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create results directory: %w", err)
	}

	return &CheckRunRepository{
		resultsDir: resultsDir,
	}, nil
}

// Save persists a check run with all its results
func (r *CheckRunRepository) Save(ctx context.Context, checkRun *check.CheckRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	engagementDir := filepath.Join(r.resultsDir, checkRun.EngagementID())
	if err := os.MkdirAll(engagementDir, 0755); err != nil {
		return fmt.Errorf("failed to create engagement directory: %w", err)
	}

	filePath := filepath.Join(engagementDir, "http_results.json")
	if !security.IsValidPath(filePath) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	dto := r.toDTO(checkRun)

	data, err := json.MarshalIndent(dto, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal check run: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save check run: %w", err)
	}

	return nil
}

// FindByID retrieves a check run by its ID
func (r *CheckRunRepository) FindByID(ctx context.Context, id string) (*check.CheckRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// For now, we'll need to search through all engagement directories
	// In a real database, this would be a simple index lookup
	entries, err := os.ReadDir(r.resultsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read results directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		filePath := filepath.Join(r.resultsDir, entry.Name(), "http_results.json")
		checkRun, err := r.loadFromFile(filePath)
		if err != nil {
			continue
		}

		if checkRun.ID() == id {
			return checkRun, nil
		}
	}

	return nil, sharedErrors.ErrCheckRunNotFound
}

// FindByEngagementID retrieves all check runs for an engagement
func (r *CheckRunRepository) FindByEngagementID(ctx context.Context, engagementID string) ([]*check.CheckRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	engagementDir := filepath.Join(r.resultsDir, engagementID)
	filePath := filepath.Join(engagementDir, "http_results.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []*check.CheckRun{}, nil
	}

	checkRun, err := r.loadFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load check run: %w", err)
	}

	return []*check.CheckRun{checkRun}, nil
}

// FindAll retrieves all check runs
func (r *CheckRunRepository) FindAll(ctx context.Context) ([]*check.CheckRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, err := os.ReadDir(r.resultsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read results directory: %w", err)
	}

	var checkRuns []*check.CheckRun
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		filePath := filepath.Join(r.resultsDir, entry.Name(), "http_results.json")
		checkRun, err := r.loadFromFile(filePath)
		if err != nil {
			continue
		}

		checkRuns = append(checkRuns, checkRun)
	}

	return checkRuns, nil
}

// Delete removes a check run by its ID
func (r *CheckRunRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find the engagement directory containing this check run
	entries, err := os.ReadDir(r.resultsDir)
	if err != nil {
		return fmt.Errorf("failed to read results directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		filePath := filepath.Join(r.resultsDir, entry.Name(), "http_results.json")
		checkRun, err := r.loadFromFile(filePath)
		if err != nil {
			continue
		}

		if checkRun.ID() == id {
			if err := os.Remove(filePath); err != nil {
				return fmt.Errorf("failed to delete check run: %w", err)
			}
			return nil
		}
	}

	return sharedErrors.ErrCheckRunNotFound
}

// Helper methods

func (r *CheckRunRepository) loadFromFile(filePath string) (*check.CheckRun, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var dto checkRunDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}

	return r.fromDTO(dto)
}

func (r *CheckRunRepository) toDTO(checkRun *check.CheckRun) checkRunDTO {
	dto := checkRunDTO{
		ID:             checkRun.ID(),
		EngagementID:   checkRun.EngagementID(),
		EngagementName: checkRun.EngagementName(),
		Operator:       checkRun.Operator(),
		StartedAt:      checkRun.StartedAt().Format(time.RFC3339),
		Status:         string(checkRun.Status()),
		Results:        make([]resultDTO, 0),
		Metadata: metadataDTO{
			AuditHash:            checkRun.Metadata().AuditHash,
			HashAlgorithm:        checkRun.Metadata().HashAlgorithm,
			SignatureFingerprint: checkRun.Metadata().SignatureFingerprint,
			TotalTargets:         checkRun.Metadata().TotalTargets,
		},
	}

	if !checkRun.CompletedAt().IsZero() {
		dto.CompletedAt = checkRun.CompletedAt().Format(time.RFC3339)
	}

	for _, result := range checkRun.Results() {
		dto.Results = append(dto.Results, r.resultToDTO(result))
	}

	return dto
}

func (r *CheckRunRepository) resultToDTO(result *check.Result) resultDTO {
	dto := resultDTO{
		Target:       result.Target(),
		Status:       string(result.Status()),
		HTTPStatus:   result.HTTPStatus(),
		CheckedAt:    result.CheckedAt().Format(time.RFC3339),
		ResponseTime: result.ResponseTime(),
		Error:        result.Error(),
		Findings:     findingsDTO{},
	}

	if !result.TLSExpiry().IsZero() {
		dto.TLSExpiry = result.TLSExpiry().Format(time.RFC3339)
	}

	// Convert findings (simplified for now)
	findings := result.Findings()
	if findings.SecurityHeaders != nil {
		dto.Findings.SecurityHeaders = &securityHeadersDTO{
			Score:           findings.SecurityHeaders.Score,
			Grade:           findings.SecurityHeaders.Grade,
			HeadersPresent:  findings.SecurityHeaders.HeadersPresent,
			HeadersMissing:  findings.SecurityHeaders.HeadersMissing,
			Recommendations: findings.SecurityHeaders.Recommendations,
		}
	}

	return dto
}

func (r *CheckRunRepository) fromDTO(dto checkRunDTO) (*check.CheckRun, error) {
	startedAt, err := time.Parse(time.RFC3339, dto.StartedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse started at time: %w", err)
	}

	var completedAt time.Time
	if dto.CompletedAt != "" {
		completedAt, err = time.Parse(time.RFC3339, dto.CompletedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse completed at time: %w", err)
		}
	}

	results := make([]*check.Result, 0, len(dto.Results))
	for _, resultDTO := range dto.Results {
		result, err := r.resultFromDTO(resultDTO)
		if err != nil {
			return nil, fmt.Errorf("failed to convert result: %w", err)
		}
		results = append(results, result)
	}

	metadata := check.Metadata{
		AuditHash:            dto.Metadata.AuditHash,
		HashAlgorithm:        dto.Metadata.HashAlgorithm,
		SignatureFingerprint: dto.Metadata.SignatureFingerprint,
		TotalTargets:         dto.Metadata.TotalTargets,
	}

	return check.Reconstruct(
		dto.ID,
		dto.EngagementID,
		dto.EngagementName,
		dto.Operator,
		startedAt,
		completedAt,
		check.RunStatus(dto.Status),
		results,
		metadata,
	), nil
}

func (r *CheckRunRepository) resultFromDTO(dto resultDTO) (*check.Result, error) {
	result, err := check.NewResult(dto.Target, check.CheckStatus(dto.Status))
	if err != nil {
		return nil, err
	}

	result.SetHTTPStatus(dto.HTTPStatus)
	result.SetResponseTime(dto.ResponseTime)

	if dto.TLSExpiry != "" {
		tlsExpiry, err := time.Parse(time.RFC3339, dto.TLSExpiry)
		if err != nil {
			return nil, fmt.Errorf("failed to parse TLS expiry: %w", err)
		}
		result.SetTLSExpiry(tlsExpiry)
	}

	if dto.Error != "" {
		result.SetError(dto.Error)
	}

	// Convert findings back (simplified for now)
	if dto.Findings.SecurityHeaders != nil {
		result.AddSecurityHeadersFindings(&check.SecurityHeadersResult{
			Score:           dto.Findings.SecurityHeaders.Score,
			Grade:           dto.Findings.SecurityHeaders.Grade,
			HeadersPresent:  dto.Findings.SecurityHeaders.HeadersPresent,
			HeadersMissing:  dto.Findings.SecurityHeaders.HeadersMissing,
			Recommendations: dto.Findings.SecurityHeaders.Recommendations,
		})
	}

	return result, nil
}
