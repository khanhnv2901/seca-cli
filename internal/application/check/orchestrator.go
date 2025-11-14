package check

import (
	"context"
	"fmt"

	"github.com/khanhnv2901/seca-cli/internal/domain/audit"
	"github.com/khanhnv2901/seca-cli/internal/domain/check"
	"github.com/khanhnv2901/seca-cli/internal/domain/engagement"
)

// Orchestrator coordinates check execution across multiple components
type Orchestrator struct {
	engagementRepo engagement.Repository
	checkRunRepo   check.Repository
	auditRepo      audit.Repository
}

// NewOrchestrator creates a new check orchestrator
func NewOrchestrator(
	engagementRepo engagement.Repository,
	checkRunRepo check.Repository,
	auditRepo audit.Repository,
) *Orchestrator {
	return &Orchestrator{
		engagementRepo: engagementRepo,
		checkRunRepo:   checkRunRepo,
		auditRepo:      auditRepo,
	}
}

// CreateCheckRun creates a new check run for an engagement
func (o *Orchestrator) CreateCheckRun(ctx context.Context, engagementID, operator string) (*check.CheckRun, error) {
	// Validate engagement exists and is authorized
	eng, err := o.engagementRepo.FindByID(ctx, engagementID)
	if err != nil {
		return nil, fmt.Errorf("failed to get engagement: %w", err)
	}

	if !eng.IsAuthorized() {
		return nil, fmt.Errorf("engagement not authorized: ROE not acknowledged")
	}

	if !eng.IsActive() {
		return nil, fmt.Errorf("engagement is not active")
	}

	// Create check run
	checkRun, err := check.NewCheckRun(engagementID, eng.Name(), operator)
	if err != nil {
		return nil, fmt.Errorf("failed to create check run: %w", err)
	}

	// Start the check run
	if err := checkRun.Start(); err != nil {
		return nil, fmt.Errorf("failed to start check run: %w", err)
	}

	return checkRun, nil
}

// AddCheckResult adds a result to a check run
func (o *Orchestrator) AddCheckResult(ctx context.Context, checkRun *check.CheckRun, result *check.Result) error {
	if err := checkRun.AddResult(result); err != nil {
		return fmt.Errorf("failed to add result: %w", err)
	}

	return nil
}

// FinalizeCheckRun completes a check run and persists it
func (o *Orchestrator) FinalizeCheckRun(ctx context.Context, checkRun *check.CheckRun, auditHash, hashAlgorithm string) error {
	// Complete the check run
	if err := checkRun.Complete(); err != nil {
		return fmt.Errorf("failed to complete check run: %w", err)
	}

	// Set audit hash
	if auditHash != "" {
		if err := checkRun.SetAuditHash(auditHash, hashAlgorithm); err != nil {
			return fmt.Errorf("failed to set audit hash: %w", err)
		}
	}

	// Save the check run
	if err := o.checkRunRepo.Save(ctx, checkRun); err != nil {
		return fmt.Errorf("failed to save check run: %w", err)
	}

	return nil
}

// GetCheckRun retrieves a check run by ID
func (o *Orchestrator) GetCheckRun(ctx context.Context, id string) (*check.CheckRun, error) {
	checkRun, err := o.checkRunRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get check run: %w", err)
	}

	return checkRun, nil
}

// GetCheckRunsByEngagement retrieves all check runs for an engagement
func (o *Orchestrator) GetCheckRunsByEngagement(ctx context.Context, engagementID string) ([]*check.CheckRun, error) {
	checkRuns, err := o.checkRunRepo.FindByEngagementID(ctx, engagementID)
	if err != nil {
		return nil, fmt.Errorf("failed to get check runs: %w", err)
	}

	return checkRuns, nil
}

// RecordAuditEntry records an audit entry for a check
func (o *Orchestrator) RecordAuditEntry(ctx context.Context, entry *audit.Entry) error {
	if err := o.auditRepo.AppendEntry(ctx, entry.EngagementID, entry); err != nil {
		return fmt.Errorf("failed to record audit entry: %w", err)
	}

	return nil
}

// SealAuditTrail seals an audit trail with a hash
func (o *Orchestrator) SealAuditTrail(ctx context.Context, engagementID, hashAlgorithm string) (string, error) {
	// Compute hash
	hash, err := o.auditRepo.ComputeHash(ctx, engagementID, hashAlgorithm)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	// Get audit trail
	auditTrail, err := o.auditRepo.FindByEngagementID(ctx, engagementID)
	if err != nil {
		return "", fmt.Errorf("failed to get audit trail: %w", err)
	}

	// Seal it
	if err := auditTrail.Seal(hash, hashAlgorithm); err != nil {
		return "", fmt.Errorf("failed to seal audit trail: %w", err)
	}

	// Save sealed audit trail
	if err := o.auditRepo.Save(ctx, auditTrail); err != nil {
		return "", fmt.Errorf("failed to save audit trail: %w", err)
	}

	return hash, nil
}

// VerifyAuditTrail verifies the integrity of an audit trail
func (o *Orchestrator) VerifyAuditTrail(ctx context.Context, engagementID string) (bool, error) {
	valid, err := o.auditRepo.VerifyIntegrity(ctx, engagementID)
	if err != nil {
		return false, fmt.Errorf("failed to verify audit trail: %w", err)
	}

	return valid, nil
}
