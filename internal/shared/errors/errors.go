package errors

import "errors"

// Domain errors
var (
	// Engagement errors
	ErrEngagementNotFound      = errors.New("engagement not found")
	ErrEngagementAlreadyExists = errors.New("engagement already exists")
	ErrEngagementUnauthorized  = errors.New("engagement not authorized - ROE not acknowledged")
	ErrInvalidEngagementID     = errors.New("invalid engagement ID")
	ErrEmptyEngagementName     = errors.New("engagement name cannot be empty")
	ErrEmptyOwner              = errors.New("engagement owner cannot be empty")
	ErrEmptyROE                = errors.New("rules of engagement (ROE) cannot be empty")
	ErrTargetNotInScope        = errors.New("target not in engagement scope")
	ErrDuplicateTarget         = errors.New("target already in scope")
	ErrEngagementInactive      = errors.New("engagement is not active")

	// Check errors
	ErrCheckRunNotFound     = errors.New("check run not found")
	ErrInvalidCheckStatus   = errors.New("invalid check status")
	ErrCheckRunAlreadyStarted = errors.New("check run already started")
	ErrCheckRunNotStarted   = errors.New("check run not started")
	ErrCheckRunAlreadyCompleted = errors.New("check run already completed")
	ErrEmptyTarget          = errors.New("target cannot be empty")
	ErrInvalidHashAlgorithm = errors.New("unsupported hash algorithm")

	// Audit errors
	ErrAuditTrailNotFound    = errors.New("audit trail not found")
	ErrAuditTrailSealed      = errors.New("audit trail is sealed")
	ErrAuditTrailNotSealed   = errors.New("audit trail is not sealed")
	ErrAuditIntegrityFailed  = errors.New("audit integrity verification failed")
	ErrEmptyHash             = errors.New("hash cannot be empty")
	ErrEmptySignature        = errors.New("signature cannot be empty")

	// Repository errors
	ErrRepositoryOperation = errors.New("repository operation failed")
	ErrInvalidData         = errors.New("invalid data")
	ErrSerializationFailed = errors.New("serialization failed")
	ErrDeserializationFailed = errors.New("deserialization failed")

	// Validation errors
	ErrValidation       = errors.New("validation error")
	ErrInvalidInput     = errors.New("invalid input")
	ErrMissingRequired  = errors.New("missing required field")
)
