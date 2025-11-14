package json

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/domain/engagement"
	sharedErrors "github.com/khanhnv2901/seca-cli/internal/shared/errors"
	"github.com/khanhnv2901/seca-cli/internal/shared/security"
)

// engagementDTO is the data transfer object for JSON serialization
type engagementDTO struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Owner     string   `json:"owner"`
	Start     string   `json:"start,omitempty"`
	End       string   `json:"end,omitempty"`
	Scope     []string `json:"scope,omitempty"`
	ROE       string   `json:"roe,omitempty"`
	ROEAgree  bool     `json:"roe_agree"`
	CreatedAt string   `json:"created_at"`
}

// EngagementRepository implements the engagement.Repository interface using JSON file storage
type EngagementRepository struct {
	filePath string
	mu       sync.RWMutex
}

// NewEngagementRepository creates a new JSON-based engagement repository
func NewEngagementRepository(dataDir string) (*EngagementRepository, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("data directory cannot be empty")
	}

	// Ensure the data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	filePath := filepath.Join(dataDir, "engagements.json")

	// Validate the file path for security
	if !security.IsValidPath(filePath) {
		return nil, fmt.Errorf("invalid file path: %s", filePath)
	}

	repo := &EngagementRepository{
		filePath: filePath,
	}

	// Initialize the file if it doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := repo.saveToFile([]engagementDTO{}); err != nil {
			return nil, fmt.Errorf("failed to initialize engagements file: %w", err)
		}
	}

	return repo, nil
}

// Save persists an engagement
func (r *EngagementRepository) Save(ctx context.Context, eng *engagement.Engagement) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	engagements, err := r.loadFromFile()
	if err != nil {
		return fmt.Errorf("failed to load engagements: %w", err)
	}

	dto := r.toDTO(eng)

	// Check if engagement already exists
	found := false
	for i, e := range engagements {
		if e.ID == dto.ID {
			engagements[i] = dto
			found = true
			break
		}
	}

	if !found {
		engagements = append(engagements, dto)
	}

	if err := r.saveToFile(engagements); err != nil {
		return fmt.Errorf("failed to save engagements: %w", err)
	}

	return nil
}

// FindByID retrieves an engagement by its ID
func (r *EngagementRepository) FindByID(ctx context.Context, id string) (*engagement.Engagement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	engagements, err := r.loadFromFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load engagements: %w", err)
	}

	for _, dto := range engagements {
		if dto.ID == id {
			return r.fromDTO(dto)
		}
	}

	return nil, sharedErrors.ErrEngagementNotFound
}

// FindAll retrieves all engagements
func (r *EngagementRepository) FindAll(ctx context.Context) ([]*engagement.Engagement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	engagements, err := r.loadFromFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load engagements: %w", err)
	}

	result := make([]*engagement.Engagement, 0, len(engagements))
	for _, dto := range engagements {
		eng, err := r.fromDTO(dto)
		if err != nil {
			return nil, fmt.Errorf("failed to convert engagement %s: %w", dto.ID, err)
		}
		result = append(result, eng)
	}

	return result, nil
}

// Delete removes an engagement by its ID
func (r *EngagementRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	engagements, err := r.loadFromFile()
	if err != nil {
		return fmt.Errorf("failed to load engagements: %w", err)
	}

	found := false
	for i, e := range engagements {
		if e.ID == id {
			engagements = append(engagements[:i], engagements[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return sharedErrors.ErrEngagementNotFound
	}

	if err := r.saveToFile(engagements); err != nil {
		return fmt.Errorf("failed to save engagements: %w", err)
	}

	return nil
}

// Exists checks if an engagement exists by ID
func (r *EngagementRepository) Exists(ctx context.Context, id string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	engagements, err := r.loadFromFile()
	if err != nil {
		return false, fmt.Errorf("failed to load engagements: %w", err)
	}

	for _, e := range engagements {
		if e.ID == id {
			return true, nil
		}
	}

	return false, nil
}

// Helper methods

func (r *EngagementRepository) loadFromFile() ([]engagementDTO, error) {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []engagementDTO{}, nil
		}
		return nil, err
	}

	var engagements []engagementDTO
	if err := json.Unmarshal(data, &engagements); err != nil {
		return nil, err
	}

	return engagements, nil
}

func (r *EngagementRepository) saveToFile(engagements []engagementDTO) error {
	data, err := json.MarshalIndent(engagements, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, data, 0644)
}

func (r *EngagementRepository) toDTO(eng *engagement.Engagement) engagementDTO {
	dto := engagementDTO{
		ID:       eng.ID(),
		Name:     eng.Name(),
		Owner:    eng.Owner(),
		Scope:    eng.Scope(),
		ROE:      eng.ROE(),
		ROEAgree: eng.ROEAgreed(),
	}

	if !eng.Start().IsZero() {
		dto.Start = eng.Start().Format("2006-01-02T15:04:05Z07:00")
	}
	if !eng.End().IsZero() {
		dto.End = eng.End().Format("2006-01-02T15:04:05Z07:00")
	}
	if !eng.CreatedAt().IsZero() {
		dto.CreatedAt = eng.CreatedAt().Format("2006-01-02T15:04:05Z07:00")
	}

	return dto
}

func (r *EngagementRepository) fromDTO(dto engagementDTO) (*engagement.Engagement, error) {
	var start, end, createdAt time.Time
	var err error

	if dto.Start != "" {
		start, err = time.Parse("2006-01-02T15:04:05Z07:00", dto.Start)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start time: %w", err)
		}
	}

	if dto.End != "" {
		end, err = time.Parse("2006-01-02T15:04:05Z07:00", dto.End)
		if err != nil {
			return nil, fmt.Errorf("failed to parse end time: %w", err)
		}
	}

	if dto.CreatedAt != "" {
		createdAt, err = time.Parse("2006-01-02T15:04:05Z07:00", dto.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created at time: %w", err)
		}
	}

	return engagement.Reconstruct(
		dto.ID,
		dto.Name,
		dto.Owner,
		dto.ROE,
		dto.Scope,
		dto.ROEAgree,
		start,
		end,
		createdAt,
	), nil
}
