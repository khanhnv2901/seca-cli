package check

import "context"

// Repository defines the interface for check run persistence
type Repository interface {
	// Save persists a check run with all its results
	Save(ctx context.Context, checkRun *CheckRun) error

	// FindByID retrieves a check run by its ID
	FindByID(ctx context.Context, id string) (*CheckRun, error)

	// FindByEngagementID retrieves all check runs for an engagement
	FindByEngagementID(ctx context.Context, engagementID string) ([]*CheckRun, error)

	// FindAll retrieves all check runs
	FindAll(ctx context.Context) ([]*CheckRun, error)

	// Delete removes a check run by its ID
	Delete(ctx context.Context, id string) error
}
