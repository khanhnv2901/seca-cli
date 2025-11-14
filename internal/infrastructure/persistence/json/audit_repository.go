package json

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/csv"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/domain/audit"
	sharedErrors "github.com/khanhnv2901/seca-cli/internal/shared/errors"
	"github.com/khanhnv2901/seca-cli/internal/shared/security"
)

// AuditRepository implements the audit.Repository interface using CSV file storage
type AuditRepository struct {
	resultsDir string
	mu         sync.RWMutex
}

// NewAuditRepository creates a new CSV-based audit repository
func NewAuditRepository(resultsDir string) (*AuditRepository, error) {
	if resultsDir == "" {
		return nil, fmt.Errorf("results directory cannot be empty")
	}

	// Ensure the results directory exists
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create results directory: %w", err)
	}

	return &AuditRepository{
		resultsDir: resultsDir,
	}, nil
}

// Save persists an audit trail
func (r *AuditRepository) Save(ctx context.Context, auditTrail *audit.AuditTrail) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	engagementDir := filepath.Join(r.resultsDir, auditTrail.EngagementID())
	if err := os.MkdirAll(engagementDir, 0755); err != nil {
		return fmt.Errorf("failed to create engagement directory: %w", err)
	}

	filePath := filepath.Join(engagementDir, "audit.csv")
	if !security.IsValidPath(filePath) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create audit file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"timestamp",
		"engagement_id",
		"operator",
		"command",
		"target",
		"status",
		"http_status",
		"tls_expiry",
		"notes",
		"error",
		"duration_seconds",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write entries
	for _, entry := range auditTrail.Entries() {
		record := []string{
			entry.Timestamp.Format(time.RFC3339),
			entry.EngagementID,
			entry.Operator,
			entry.Command,
			entry.Target,
			entry.Status,
			strconv.Itoa(entry.HTTPStatus),
			"",
			entry.Notes,
			entry.Error,
			fmt.Sprintf("%.3f", entry.DurationSeconds),
		}

		if !entry.TLSExpiry.IsZero() {
			record[7] = entry.TLSExpiry.Format(time.RFC3339)
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write entry: %w", err)
		}
	}

	// If sealed, write hash file
	if auditTrail.IsSealed() {
		hashFilePath := filePath + "." + auditTrail.HashAlgorithm()
		hashContent := fmt.Sprintf("%s  %s\n", auditTrail.Hash(), filepath.Base(filePath))
		if err := os.WriteFile(hashFilePath, []byte(hashContent), 0644); err != nil {
			return fmt.Errorf("failed to write hash file: %w", err)
		}
	}

	// If signed, write signature file
	if auditTrail.IsSigned() {
		sigFilePath := filePath + ".asc"
		if err := os.WriteFile(sigFilePath, []byte(auditTrail.Signature()), 0644); err != nil {
			return fmt.Errorf("failed to write signature file: %w", err)
		}
	}

	return nil
}

// FindByEngagementID retrieves the audit trail for an engagement
func (r *AuditRepository) FindByEngagementID(ctx context.Context, engagementID string) (*audit.AuditTrail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filePath := filepath.Join(r.resultsDir, engagementID, "audit.csv")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, sharedErrors.ErrAuditTrailNotFound
	}

	return r.loadFromFile(filePath, engagementID)
}

// AppendEntry appends a single entry to an existing audit trail
func (r *AuditRepository) AppendEntry(ctx context.Context, engagementID string, entry *audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	engagementDir := filepath.Join(r.resultsDir, engagementID)
	if err := os.MkdirAll(engagementDir, 0755); err != nil {
		return fmt.Errorf("failed to create engagement directory: %w", err)
	}

	filePath := filepath.Join(engagementDir, "audit.csv")
	if !security.IsValidPath(filePath) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	// Check if file exists, if not create with header
	fileExists := true
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fileExists = false
	}

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if new file
	if !fileExists {
		header := []string{
			"timestamp",
			"engagement_id",
			"operator",
			"command",
			"target",
			"status",
			"http_status",
			"tls_expiry",
			"notes",
			"error",
			"duration_seconds",
		}
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}
	}

	// Write entry
	record := []string{
		entry.Timestamp.Format(time.RFC3339),
		entry.EngagementID,
		entry.Operator,
		entry.Command,
		entry.Target,
		entry.Status,
		strconv.Itoa(entry.HTTPStatus),
		"",
		entry.Notes,
		entry.Error,
		fmt.Sprintf("%.3f", entry.DurationSeconds),
	}

	if !entry.TLSExpiry.IsZero() {
		record[7] = entry.TLSExpiry.Format(time.RFC3339)
	}

	if err := writer.Write(record); err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}

	return nil
}

// ComputeHash calculates the hash of the audit trail file
func (r *AuditRepository) ComputeHash(ctx context.Context, engagementID, algorithm string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filePath := filepath.Join(r.resultsDir, engagementID, "audit.csv")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", sharedErrors.ErrAuditTrailNotFound
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open audit file: %w", err)
	}
	defer file.Close()

	var h hash.Hash
	switch algorithm {
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		return "", sharedErrors.ErrInvalidHashAlgorithm
	}

	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// VerifyIntegrity verifies the integrity of an audit trail
func (r *AuditRepository) VerifyIntegrity(ctx context.Context, engagementID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	auditFilePath := filepath.Join(r.resultsDir, engagementID, "audit.csv")
	if _, err := os.Stat(auditFilePath); os.IsNotExist(err) {
		return false, sharedErrors.ErrAuditTrailNotFound
	}

	// Try both sha256 and sha512
	algorithms := []string{"sha256", "sha512"}
	for _, algorithm := range algorithms {
		hashFilePath := auditFilePath + "." + algorithm
		if _, err := os.Stat(hashFilePath); os.IsNotExist(err) {
			continue
		}

		// Read expected hash
		hashContent, err := os.ReadFile(hashFilePath)
		if err != nil {
			continue
		}

		var expectedHash string
		fmt.Sscanf(string(hashContent), "%s", &expectedHash)

		// Compute actual hash
		actualHash, err := r.ComputeHash(ctx, engagementID, algorithm)
		if err != nil {
			return false, err
		}

		return expectedHash == actualHash, nil
	}

	return false, fmt.Errorf("no hash file found")
}

// Helper methods

func (r *AuditRepository) loadFromFile(filePath, engagementID string) (*audit.AuditTrail, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	entries := make([]*audit.Entry, 0)

	// Read entries
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read record: %w", err)
		}

		timestamp, err := time.Parse(time.RFC3339, record[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
		}

		httpStatus, _ := strconv.Atoi(record[6])
		durationSeconds, _ := strconv.ParseFloat(record[10], 64)

		var tlsExpiry time.Time
		if record[7] != "" {
			tlsExpiry, _ = time.Parse(time.RFC3339, record[7])
		}

		entry := &audit.Entry{
			Timestamp:       timestamp,
			EngagementID:    record[1],
			Operator:        record[2],
			Command:         record[3],
			Target:          record[4],
			Status:          record[5],
			HTTPStatus:      httpStatus,
			TLSExpiry:       tlsExpiry,
			Notes:           record[8],
			Error:           record[9],
			DurationSeconds: durationSeconds,
		}

		entries = append(entries, entry)
	}

	// Check for hash file
	var hash, hashAlgorithm string
	algorithms := []string{"sha256", "sha512"}
	for _, alg := range algorithms {
		hashFilePath := filePath + "." + alg
		if _, err := os.Stat(hashFilePath); err == nil {
			hashContent, err := os.ReadFile(hashFilePath)
			if err == nil {
				fmt.Sscanf(string(hashContent), "%s", &hash)
				hashAlgorithm = alg
				break
			}
		}
	}

	// Check for signature file
	var signature string
	sigFilePath := filePath + ".asc"
	if _, err := os.Stat(sigFilePath); err == nil {
		sigContent, err := os.ReadFile(sigFilePath)
		if err == nil {
			signature = string(sigContent)
		}
	}

	sealed := hash != ""
	createdAt := time.Now()
	if len(entries) > 0 {
		createdAt = entries[0].Timestamp
	}

	return audit.Reconstruct(
		engagementID,
		entries,
		hash,
		hashAlgorithm,
		signature,
		createdAt,
		sealed,
	), nil
}
