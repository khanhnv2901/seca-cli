package engagement

import "context"

// Repository defines the interface for engagement persistence
type Repository interface {
	// Save persists an engagement
	Save(ctx context.Context, engagement *Engagement) error

	// FindByID retrieves an engagement by its ID
	FindByID(ctx context.Context, id string) (*Engagement, error)

	// FindAll retrieves all engagements
	FindAll(ctx context.Context) ([]*Engagement, error)

	// Delete removes an engagement by its ID
	Delete(ctx context.Context, id string) error

	// Exists checks if an engagement exists by ID
	Exists(ctx context.Context, id string) (bool, error)
}
