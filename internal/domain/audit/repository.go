package audit

import "context"

// Repository defines the interface for audit trail persistence
type Repository interface {
	// Save persists an audit trail
	Save(ctx context.Context, auditTrail *AuditTrail) error

	// FindByEngagementID retrieves the audit trail for an engagement
	FindByEngagementID(ctx context.Context, engagementID string) (*AuditTrail, error)

	// AppendEntry appends a single entry to an existing audit trail
	AppendEntry(ctx context.Context, engagementID string, entry *Entry) error

	// ComputeHash calculates the hash of the audit trail file
	ComputeHash(ctx context.Context, engagementID, algorithm string) (string, error)

	// VerifyIntegrity verifies the integrity of an audit trail
	VerifyIntegrity(ctx context.Context, engagementID string) (bool, error)
}
