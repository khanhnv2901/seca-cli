package cmd

import "fmt"

// EngagementNotFoundError indicates an engagement lookup failure.
type EngagementNotFoundError struct {
	ID string
}

func (e *EngagementNotFoundError) Error() string {
	return fmt.Sprintf("engagement %s not found", e.ID)
}

// ScopeViolationError signals that a target violates the engagement scope.
type ScopeViolationError struct {
	Target string
	Scope  string
}

func (e *ScopeViolationError) Error() string {
	switch {
	case e.Target != "" && e.Scope != "":
		return fmt.Sprintf("target %s is not permitted for engagement %s", e.Target, e.Scope)
	case e.Scope != "":
		return fmt.Sprintf("scope %s is invalid or empty", e.Scope)
	}
	return fmt.Sprintf("target %s violates scope policy", e.Target)
}
