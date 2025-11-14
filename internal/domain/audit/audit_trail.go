package audit

import (
	"errors"
	"time"
)

// AuditTrail represents an immutable audit trail for a check run
// It ensures evidence integrity through cryptographic hashing
type AuditTrail struct {
	engagementID string
	entries      []*Entry
	hash         string
	hashAlgorithm string
	signature    string
	createdAt    time.Time
	sealed       bool // Once sealed, no more entries can be added
}

// Entry represents a single audit trail entry
type Entry struct {
	Timestamp        time.Time
	EngagementID     string
	Operator         string
	Command          string
	Target           string
	Status           string
	HTTPStatus       int
	TLSExpiry        time.Time
	Notes            string
	Error            string
	DurationSeconds  float64
}

// NewAuditTrail creates a new audit trail
func NewAuditTrail(engagementID string) (*AuditTrail, error) {
	if engagementID == "" {
		return nil, errors.New("engagement ID cannot be empty")
	}

	return &AuditTrail{
		engagementID: engagementID,
		entries:      make([]*Entry, 0),
		createdAt:    time.Now(),
		sealed:       false,
	}, nil
}

// Reconstruct creates an audit trail from persisted data
func Reconstruct(engagementID string, entries []*Entry, hash, hashAlgorithm, signature string, createdAt time.Time, sealed bool) *AuditTrail {
	return &AuditTrail{
		engagementID:  engagementID,
		entries:       entries,
		hash:          hash,
		hashAlgorithm: hashAlgorithm,
		signature:     signature,
		createdAt:     createdAt,
		sealed:        sealed,
	}
}

// Business methods

// AppendEntry adds a new entry to the audit trail
func (at *AuditTrail) AppendEntry(entry *Entry) error {
	if at.sealed {
		return errors.New("cannot append to a sealed audit trail")
	}

	if entry == nil {
		return errors.New("entry cannot be nil")
	}

	if entry.EngagementID != at.engagementID {
		return errors.New("entry engagement ID does not match audit trail")
	}

	at.entries = append(at.entries, entry)
	return nil
}

// Seal finalizes the audit trail and computes its hash
func (at *AuditTrail) Seal(hash, algorithm string) error {
	if at.sealed {
		return errors.New("audit trail is already sealed")
	}

	if hash == "" {
		return errors.New("hash cannot be empty")
	}

	if algorithm != "sha256" && algorithm != "sha512" {
		return errors.New("unsupported hash algorithm")
	}

	at.hash = hash
	at.hashAlgorithm = algorithm
	at.sealed = true
	return nil
}

// Sign adds a GPG signature to the audit trail
func (at *AuditTrail) Sign(signature string) error {
	if !at.sealed {
		return errors.New("audit trail must be sealed before signing")
	}

	if signature == "" {
		return errors.New("signature cannot be empty")
	}

	at.signature = signature
	return nil
}

// VerifyIntegrity checks if the computed hash matches the expected hash
func (at *AuditTrail) VerifyIntegrity(computedHash string) bool {
	return at.hash == computedHash
}

// IsSealed checks if the audit trail is sealed
func (at *AuditTrail) IsSealed() bool {
	return at.sealed
}

// IsSigned checks if the audit trail is signed
func (at *AuditTrail) IsSigned() bool {
	return at.signature != ""
}

// Getters

func (at *AuditTrail) EngagementID() string {
	return at.engagementID
}

func (at *AuditTrail) Entries() []*Entry {
	// Return a copy to prevent external modification
	entriesCopy := make([]*Entry, len(at.entries))
	copy(entriesCopy, at.entries)
	return entriesCopy
}

func (at *AuditTrail) Hash() string {
	return at.hash
}

func (at *AuditTrail) HashAlgorithm() string {
	return at.hashAlgorithm
}

func (at *AuditTrail) Signature() string {
	return at.signature
}

func (at *AuditTrail) CreatedAt() time.Time {
	return at.createdAt
}

// NewEntry creates a new audit entry
func NewEntry(timestamp time.Time, engagementID, operator, command, target, status string) *Entry {
	return &Entry{
		Timestamp:    timestamp,
		EngagementID: engagementID,
		Operator:     operator,
		Command:      command,
		Target:       target,
		Status:       status,
	}
}
