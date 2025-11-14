package engagement

import (
	"errors"
	"time"
)

// Engagement represents an authorized security testing engagement
// It serves as an aggregate root in the DDD context
type Engagement struct {
	id        string
	name      string
	owner     string
	start     time.Time
	end       time.Time
	scope     []string
	roe       string
	roeAgree  bool
	createdAt time.Time
}

// NewEngagement creates a new engagement with validation
func NewEngagement(name, owner, roe string, scope []string) (*Engagement, error) {
	if name == "" {
		return nil, errors.New("engagement name cannot be empty")
	}
	if owner == "" {
		return nil, errors.New("engagement owner cannot be empty")
	}
	if roe == "" {
		return nil, errors.New("rules of engagement (ROE) cannot be empty")
	}

	now := time.Now()
	return &Engagement{
		id:        generateID(),
		name:      name,
		owner:     owner,
		roe:       roe,
		scope:     scope,
		roeAgree:  false,
		createdAt: now,
	}, nil
}

// Reconstruct creates an engagement from persisted data (for repository use)
func Reconstruct(id, name, owner, roe string, scope []string, roeAgree bool, start, end, createdAt time.Time) *Engagement {
	return &Engagement{
		id:        id,
		name:      name,
		owner:     owner,
		start:     start,
		end:       end,
		scope:     scope,
		roe:       roe,
		roeAgree:  roeAgree,
		createdAt: createdAt,
	}
}

// Business methods

// AcknowledgeROE marks that the rules of engagement have been acknowledged
func (e *Engagement) AcknowledgeROE() error {
	if e.roeAgree {
		return errors.New("ROE already acknowledged")
	}
	e.roeAgree = true
	return nil
}

// IsAuthorized checks if the engagement is authorized to run checks
func (e *Engagement) IsAuthorized() bool {
	return e.roeAgree
}

// AddToScope adds a target to the engagement scope
func (e *Engagement) AddToScope(target string) error {
	if target == "" {
		return errors.New("target cannot be empty")
	}

	// Check for duplicates
	for _, s := range e.scope {
		if s == target {
			return errors.New("target already in scope")
		}
	}

	e.scope = append(e.scope, target)
	return nil
}

// RemoveFromScope removes a target from the engagement scope
func (e *Engagement) RemoveFromScope(target string) error {
	for i, s := range e.scope {
		if s == target {
			e.scope = append(e.scope[:i], e.scope[i+1:]...)
			return nil
		}
	}
	return errors.New("target not found in scope")
}

// IsInScope checks if a target is within the engagement scope
func (e *Engagement) IsInScope(target string) bool {
	for _, s := range e.scope {
		if s == target {
			return true
		}
	}
	return false
}

// SetTimeRange sets the start and end time for the engagement
func (e *Engagement) SetTimeRange(start, end time.Time) error {
	if !end.IsZero() && end.Before(start) {
		return errors.New("end time cannot be before start time")
	}
	e.start = start
	e.end = end
	return nil
}

// IsActive checks if the engagement is currently active based on time range
func (e *Engagement) IsActive() bool {
	now := time.Now()

	// If no start time set, consider it active
	if e.start.IsZero() {
		return true
	}

	// Check if current time is after start
	if now.Before(e.start) {
		return false
	}

	// If no end time set, and we're past start, consider it active
	if e.end.IsZero() {
		return true
	}

	// Check if current time is before end
	return now.Before(e.end)
}

// Getters (exposing internal state)

func (e *Engagement) ID() string {
	return e.id
}

func (e *Engagement) Name() string {
	return e.name
}

func (e *Engagement) Owner() string {
	return e.owner
}

func (e *Engagement) Start() time.Time {
	return e.start
}

func (e *Engagement) End() time.Time {
	return e.end
}

func (e *Engagement) Scope() []string {
	// Return a copy to prevent external modification
	scopeCopy := make([]string, len(e.scope))
	copy(scopeCopy, e.scope)
	return scopeCopy
}

func (e *Engagement) ROE() string {
	return e.roe
}

func (e *Engagement) ROEAgreed() bool {
	return e.roeAgree
}

func (e *Engagement) CreatedAt() time.Time {
	return e.createdAt
}

// Helper function to generate engagement IDs
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + time.Now().Format("000000000")[0:6]
}
