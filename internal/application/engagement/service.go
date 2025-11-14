package engagement

import (
	"context"
	"fmt"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/domain/engagement"
	sharedErrors "github.com/khanhnv2901/seca-cli/internal/shared/errors"
)

// Service provides application-level engagement operations
type Service struct {
	repo engagement.Repository
}

// NewService creates a new engagement service
func NewService(repo engagement.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// CreateEngagement creates a new engagement
func (s *Service) CreateEngagement(ctx context.Context, name, owner, roe string, scope []string) (*engagement.Engagement, error) {
	eng, err := engagement.NewEngagement(name, owner, roe, scope)
	if err != nil {
		return nil, fmt.Errorf("failed to create engagement: %w", err)
	}

	if err := s.repo.Save(ctx, eng); err != nil {
		return nil, fmt.Errorf("failed to save engagement: %w", err)
	}

	return eng, nil
}

// GetEngagement retrieves an engagement by ID
func (s *Service) GetEngagement(ctx context.Context, id string) (*engagement.Engagement, error) {
	eng, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get engagement: %w", err)
	}

	return eng, nil
}

// ListEngagements retrieves all engagements
func (s *Service) ListEngagements(ctx context.Context) ([]*engagement.Engagement, error) {
	engagements, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list engagements: %w", err)
	}

	return engagements, nil
}

// AcknowledgeROE acknowledges the rules of engagement for an engagement
func (s *Service) AcknowledgeROE(ctx context.Context, id string) error {
	eng, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get engagement: %w", err)
	}

	if err := eng.AcknowledgeROE(); err != nil {
		return fmt.Errorf("failed to acknowledge ROE: %w", err)
	}

	if err := s.repo.Save(ctx, eng); err != nil {
		return fmt.Errorf("failed to save engagement: %w", err)
	}

	return nil
}

// AddToScope adds targets to an engagement's scope
func (s *Service) AddToScope(ctx context.Context, id string, targets []string) error {
	eng, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get engagement: %w", err)
	}

	for _, target := range targets {
		if err := eng.AddToScope(target); err != nil {
			return fmt.Errorf("failed to add target %s to scope: %w", target, err)
		}
	}

	if err := s.repo.Save(ctx, eng); err != nil {
		return fmt.Errorf("failed to save engagement: %w", err)
	}

	return nil
}

// RemoveFromScope removes targets from an engagement's scope
func (s *Service) RemoveFromScope(ctx context.Context, id string, targets []string) error {
	eng, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get engagement: %w", err)
	}

	for _, target := range targets {
		if err := eng.RemoveFromScope(target); err != nil {
			return fmt.Errorf("failed to remove target %s from scope: %w", target, err)
		}
	}

	if err := s.repo.Save(ctx, eng); err != nil {
		return fmt.Errorf("failed to save engagement: %w", err)
	}

	return nil
}

// SetTimeRange sets the time range for an engagement
func (s *Service) SetTimeRange(ctx context.Context, id string, start, end time.Time) error {
	eng, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get engagement: %w", err)
	}

	if err := eng.SetTimeRange(start, end); err != nil {
		return fmt.Errorf("failed to set time range: %w", err)
	}

	if err := s.repo.Save(ctx, eng); err != nil {
		return fmt.Errorf("failed to save engagement: %w", err)
	}

	return nil
}

// DeleteEngagement deletes an engagement
func (s *Service) DeleteEngagement(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete engagement: %w", err)
	}

	return nil
}

// ValidateEngagementForChecks validates that an engagement is ready for running checks
func (s *Service) ValidateEngagementForChecks(ctx context.Context, id string, target string) error {
	eng, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get engagement: %w", err)
	}

	// Check if ROE is acknowledged
	if !eng.IsAuthorized() {
		return sharedErrors.ErrEngagementUnauthorized
	}

	// Check if engagement is active
	if !eng.IsActive() {
		return sharedErrors.ErrEngagementInactive
	}

	// Check if target is in scope
	if target != "" && !eng.IsInScope(target) {
		return sharedErrors.ErrTargetNotInScope
	}

	return nil
}
