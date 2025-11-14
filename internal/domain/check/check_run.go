package check

import (
	"errors"
	"time"
)

// CheckRun represents an execution of security checks against an engagement's scope
// It serves as an aggregate root that owns CheckResults and AuditTrail
type CheckRun struct {
	id             string
	engagementID   string
	engagementName string
	operator       string
	startedAt      time.Time
	completedAt    time.Time
	status         RunStatus
	results        []*Result
	metadata       Metadata
}

// RunStatus represents the status of a check run
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
)

// Metadata contains additional information about the check run
type Metadata struct {
	AuditHash            string
	HashAlgorithm        string
	SignatureFingerprint string
	TotalTargets         int
}

// NewCheckRun creates a new check run
func NewCheckRun(engagementID, engagementName, operator string) (*CheckRun, error) {
	if engagementID == "" {
		return nil, errors.New("engagement ID cannot be empty")
	}
	if operator == "" {
		return nil, errors.New("operator cannot be empty")
	}

	return &CheckRun{
		id:             generateCheckRunID(),
		engagementID:   engagementID,
		engagementName: engagementName,
		operator:       operator,
		startedAt:      time.Now(),
		status:         RunStatusPending,
		results:        make([]*Result, 0),
		metadata:       Metadata{},
	}, nil
}

// Reconstruct creates a check run from persisted data
func Reconstruct(id, engagementID, engagementName, operator string, startedAt, completedAt time.Time,
	status RunStatus, results []*Result, metadata Metadata) *CheckRun {
	return &CheckRun{
		id:             id,
		engagementID:   engagementID,
		engagementName: engagementName,
		operator:       operator,
		startedAt:      startedAt,
		completedAt:    completedAt,
		status:         status,
		results:        results,
		metadata:       metadata,
	}
}

// Business methods

// Start marks the check run as running
func (cr *CheckRun) Start() error {
	if cr.status != RunStatusPending {
		return errors.New("check run can only be started from pending status")
	}
	cr.status = RunStatusRunning
	cr.startedAt = time.Now()
	return nil
}

// Complete marks the check run as completed
func (cr *CheckRun) Complete() error {
	if cr.status != RunStatusRunning {
		return errors.New("check run can only be completed from running status")
	}
	cr.status = RunStatusCompleted
	cr.completedAt = time.Now()
	return nil
}

// Fail marks the check run as failed
func (cr *CheckRun) Fail() error {
	if cr.status == RunStatusCompleted {
		return errors.New("cannot fail a completed check run")
	}
	cr.status = RunStatusFailed
	cr.completedAt = time.Now()
	return nil
}

// AddResult adds a check result to the run
func (cr *CheckRun) AddResult(result *Result) error {
	if cr.status == RunStatusCompleted || cr.status == RunStatusFailed {
		return errors.New("cannot add results to a finished check run")
	}

	cr.results = append(cr.results, result)
	cr.metadata.TotalTargets = len(cr.results)
	return nil
}

// SetAuditHash sets the audit trail hash for integrity verification
func (cr *CheckRun) SetAuditHash(hash, algorithm string) error {
	if hash == "" {
		return errors.New("hash cannot be empty")
	}
	if algorithm != "sha256" && algorithm != "sha512" {
		return errors.New("unsupported hash algorithm")
	}

	cr.metadata.AuditHash = hash
	cr.metadata.HashAlgorithm = algorithm
	return nil
}

// SetSignature sets the GPG signature fingerprint
func (cr *CheckRun) SetSignature(fingerprint string) {
	cr.metadata.SignatureFingerprint = fingerprint
}

// Getters

func (cr *CheckRun) ID() string {
	return cr.id
}

func (cr *CheckRun) EngagementID() string {
	return cr.engagementID
}

func (cr *CheckRun) EngagementName() string {
	return cr.engagementName
}

func (cr *CheckRun) Operator() string {
	return cr.operator
}

func (cr *CheckRun) StartedAt() time.Time {
	return cr.startedAt
}

func (cr *CheckRun) CompletedAt() time.Time {
	return cr.completedAt
}

func (cr *CheckRun) Status() RunStatus {
	return cr.status
}

func (cr *CheckRun) Results() []*Result {
	// Return a copy to prevent external modification
	resultsCopy := make([]*Result, len(cr.results))
	copy(resultsCopy, cr.results)
	return resultsCopy
}

func (cr *CheckRun) Metadata() Metadata {
	return cr.metadata
}

// Helper function to generate check run IDs
func generateCheckRunID() string {
	return "run-" + time.Now().Format("20060102150405") + "-" + time.Now().Format("000000000")[0:6]
}
