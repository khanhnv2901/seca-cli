package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/domain/audit"
)

// Service provides application-level audit operations
type Service struct {
	repo audit.Repository
}

// NewService creates a new audit service
func NewService(repo audit.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// RecordCheckExecution records an audit entry for a check execution
func (s *Service) RecordCheckExecution(
	ctx context.Context,
	engagementID, operator, command, target, status string,
	httpStatus int,
	tlsExpiry time.Time,
	notes, errorMsg string,
	duration float64,
) error {
	entry := &audit.Entry{
		Timestamp:       time.Now(),
		EngagementID:    engagementID,
		Operator:        operator,
		Command:         command,
		Target:          target,
		Status:          status,
		HTTPStatus:      httpStatus,
		TLSExpiry:       tlsExpiry,
		Notes:           notes,
		Error:           errorMsg,
		DurationSeconds: duration,
	}

	if err := s.repo.AppendEntry(ctx, engagementID, entry); err != nil {
		return fmt.Errorf("failed to record audit entry: %w", err)
	}

	return nil
}

// GetAuditTrail retrieves the audit trail for an engagement
func (s *Service) GetAuditTrail(ctx context.Context, engagementID string) (*audit.AuditTrail, error) {
	auditTrail, err := s.repo.FindByEngagementID(ctx, engagementID)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit trail: %w", err)
	}

	return auditTrail, nil
}

// SealAuditTrail seals an audit trail with a cryptographic hash
func (s *Service) SealAuditTrail(ctx context.Context, engagementID, hashAlgorithm string) (string, error) {
	// Compute hash
	hash, err := s.repo.ComputeHash(ctx, engagementID, hashAlgorithm)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	// Get audit trail
	auditTrail, err := s.repo.FindByEngagementID(ctx, engagementID)
	if err != nil {
		return "", fmt.Errorf("failed to get audit trail: %w", err)
	}

	// Seal it
	if err := auditTrail.Seal(hash, hashAlgorithm); err != nil {
		return "", fmt.Errorf("failed to seal audit trail: %w", err)
	}

	// Save sealed audit trail
	if err := s.repo.Save(ctx, auditTrail); err != nil {
		return "", fmt.Errorf("failed to save audit trail: %w", err)
	}

	return hash, nil
}

// VerifyIntegrity verifies the integrity of an audit trail
func (s *Service) VerifyIntegrity(ctx context.Context, engagementID string) (bool, error) {
	valid, err := s.repo.VerifyIntegrity(ctx, engagementID)
	if err != nil {
		return false, fmt.Errorf("failed to verify integrity: %w", err)
	}

	return valid, nil
}

// SignAuditTrail adds a GPG signature to an audit trail
func (s *Service) SignAuditTrail(ctx context.Context, engagementID, signature string) error {
	auditTrail, err := s.repo.FindByEngagementID(ctx, engagementID)
	if err != nil {
		return fmt.Errorf("failed to get audit trail: %w", err)
	}

	if err := auditTrail.Sign(signature); err != nil {
		return fmt.Errorf("failed to sign audit trail: %w", err)
	}

	if err := s.repo.Save(ctx, auditTrail); err != nil {
		return fmt.Errorf("failed to save audit trail: %w", err)
	}

	return nil
}
