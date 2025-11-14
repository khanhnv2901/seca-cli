# DDD Migration Guide

This guide helps developers understand the transition from the old structure to the new Domain-Driven Design (DDD) architecture.

## Table of Contents

- [Overview](#overview)
- [What Changed](#what-changed)
- [Import Path Changes](#import-path-changes)
- [How to Adapt Existing Code](#how-to-adapt-existing-code)
- [Common Patterns](#common-patterns)
- [Testing](#testing)

## Overview

SECA-CLI has been restructured to follow Domain-Driven Design principles, separating business logic from infrastructure concerns.

### Before (Old Structure)

```
cmd/          → CLI layer with embedded business logic (7,354 LOC)
internal/     → Mixed responsibilities
  ├── checker/      → Check implementations
  ├── compliance/   → Framework definitions
  ├── api/          → API server
  ├── constants/    → Constants
  └── security/     → Security utils
```

### After (DDD Structure)

```
cmd/                      → CLI layer (presentation only)
internal/
  ├── domain/             → Business entities & rules
  ├── application/        → Use cases & services
  ├── infrastructure/     → External implementations
  └── shared/             → Shared utilities
```

## What Changed

### 1. Domain Layer (NEW)

**Created**: `internal/domain/`

Contains pure business entities with no dependencies on infrastructure:

- `internal/domain/engagement/` - Engagement aggregate
- `internal/domain/check/` - CheckRun aggregate
- `internal/domain/audit/` - AuditTrail aggregate

**Key Files**:
- `engagement.go` - Engagement entity with business methods
- `repository.go` - Repository interface (contract)

### 2. Application Layer (NEW)

**Created**: `internal/application/`

Application services that orchestrate domain entities:

- `internal/application/engagement/service.go` - Engagement operations
- `internal/application/check/orchestrator.go` - Check coordination
- `internal/application/audit/service.go` - Audit management

### 3. Infrastructure Layer (MOVED)

**Old Locations** → **New Locations**:

- `internal/checker/` → `internal/infrastructure/checker/`
- `internal/compliance/` → `internal/infrastructure/compliance/`
- `internal/api/` → `internal/infrastructure/api/`

**New Additions**:
- `internal/infrastructure/persistence/json/` - Repository implementations

### 4. Shared Kernel (MOVED)

**Old Locations** → **New Locations**:

- `internal/constants/` → `internal/shared/constants/`
- `internal/security/` → `internal/shared/security/`

**New Additions**:
- `internal/shared/errors/` - Domain error definitions

## Import Path Changes

### Automatic Migration

A migration script has updated all imports automatically. If you need to run it again:

```bash
./migrate_imports.sh
```

### Manual Import Updates

If adding new code, use these import paths:

#### Old Imports → New Imports

```go
// Checkers
"github.com/khanhnv2901/seca-cli/internal/checker"
// BECOMES ↓
"github.com/khanhnv2901/seca-cli/internal/infrastructure/checker"

// Compliance
"github.com/khanhnv2901/seca-cli/internal/compliance"
// BECOMES ↓
"github.com/khanhnv2901/seca-cli/internal/infrastructure/compliance"

// API
"github.com/khanhnv2901/seca-cli/internal/api"
// BECOMES ↓
"github.com/khanhnv2901/seca-cli/internal/infrastructure/api"

// Constants
"github.com/khanhnv2901/seca-cli/internal/constants"
// BECOMES ↓
"github.com/khanhnv2901/seca-cli/internal/shared/constants"

// Security
"github.com/khanhnv2901/seca-cli/internal/security"
// BECOMES ↓
"github.com/khanhnv2901/seca-cli/internal/shared/security"
```

#### New Imports (Domain & Application)

```go
// Domain entities
import "github.com/khanhnv2901/seca-cli/internal/domain/engagement"
import "github.com/khanhnv2901/seca-cli/internal/domain/check"
import "github.com/khanhnv2901/seca-cli/internal/domain/audit"

// Application services
import engagementapp "github.com/khanhnv2901/seca-cli/internal/application/engagement"
import checkapp "github.com/khanhnv2901/seca-cli/internal/application/check"
import auditapp "github.com/khanhnv2901/seca-cli/internal/application/audit"

// Repository implementations
import "github.com/khanhnv2901/seca-cli/internal/infrastructure/persistence/json"

// Shared utilities
import "github.com/khanhnv2901/seca-cli/internal/shared/errors"
```

## How to Adapt Existing Code

### Pattern 1: Direct File I/O → Repository Pattern

#### Before

```go
// cmd/engagement.go (OLD)
func loadEngagements() []Engagement {
    data, _ := os.ReadFile(engagementsFilePath)
    var engagements []Engagement
    json.Unmarshal(data, &engagements)
    return engagements
}

func saveEngagements(engagements []Engagement) {
    data, _ := json.MarshalIndent(engagements, "", "  ")
    os.WriteFile(engagementsFilePath, data, 0644)
}

// Usage in command
engagements := loadEngagements()
engagements = append(engagements, newEngagement)
saveEngagements(engagements)
```

#### After

```go
// Use repository pattern
import (
    "github.com/khanhnv2901/seca-cli/internal/infrastructure/persistence/json"
    engagementapp "github.com/khanhnv2901/seca-cli/internal/application/engagement"
)

// Initialize repository (once)
repo, err := json.NewEngagementRepository(dataDir)
if err != nil {
    return err
}

// Use application service
service := engagementapp.NewService(repo)

// Create engagement
eng, err := service.CreateEngagement(ctx, name, owner, roe, scope)
if err != nil {
    return err
}
```

### Pattern 2: Business Logic in Commands → Domain Entities

#### Before

```go
// cmd/engagement.go (OLD)
// Business logic mixed with CLI code
if !engagement.ROEAgree {
    return errors.New("ROE not acknowledged")
}

for _, target := range targets {
    if !contains(engagement.Scope, target) {
        return errors.New("target not in scope")
    }
}
```

#### After

```go
// Business logic in domain entity
// internal/domain/engagement/engagement.go
func (e *Engagement) IsAuthorized() bool {
    return e.roeAgree
}

func (e *Engagement) IsInScope(target string) bool {
    for _, s := range e.scope {
        if s == target {
            return true
        }
    }
    return false
}

// CLI just calls domain methods
if !engagement.IsAuthorized() {
    return errors.New("not authorized")
}

if !engagement.IsInScope(target) {
    return errors.New("target not in scope")
}
```

### Pattern 3: Check Execution → Orchestrator

#### Before

```go
// cmd/check.go (OLD)
results := []checker.CheckResult{}

for _, target := range targets {
    result := httpChecker.Check(ctx, target)
    results = append(results, result)

    // Audit
    appendAuditRow(engagementID, operator, target, result.Status)
}

// Save
saveResults(engagementID, results)
hash := computeHash(auditFile)
saveHash(hash)
```

#### After

```go
// Use orchestrator
import (
    checkapp "github.com/khanhnv2901/seca-cli/internal/application/check"
    "github.com/khanhnv2901/seca-cli/internal/domain/check"
)

// Initialize
orchestrator := checkapp.NewOrchestrator(engagementRepo, checkRunRepo, auditRepo)

// Create check run
checkRun, err := orchestrator.CreateCheckRun(ctx, engagementID, operator)

// Execute checks
for _, target := range targets {
    rawResult := httpChecker.Check(ctx, target)

    // Convert to domain model
    domainResult, _ := check.NewResult(rawResult.Target, check.CheckStatus(rawResult.Status))
    domainResult.SetHTTPStatus(rawResult.HTTPStatus)

    // Add to check run
    orchestrator.AddCheckResult(ctx, checkRun, domainResult)

    // Audit is handled automatically
    auditEntry := &audit.Entry{
        Timestamp:    time.Now(),
        EngagementID: engagementID,
        Operator:     operator,
        Command:      "check http",
        Target:       target,
        Status:       rawResult.Status,
    }
    orchestrator.RecordAuditEntry(ctx, auditEntry)
}

// Finalize (saves, hashes, everything)
hash, _ := orchestrator.SealAuditTrail(ctx, engagementID, "sha256")
orchestrator.FinalizeCheckRun(ctx, checkRun, hash, "sha256")
```

## Common Patterns

### Pattern: Dependency Injection

**Setup dependencies at application start**:

```go
// In cmd/root.go or similar
type AppContext struct {
    EngagementService *engagement.Service
    CheckOrchestrator *check.Orchestrator
    AuditService      *audit.Service
}

func initializeAppContext(dataDir, resultsDir string) (*AppContext, error) {
    // Repositories
    engagementRepo, err := json.NewEngagementRepository(dataDir)
    if err != nil {
        return nil, err
    }

    checkRunRepo, err := json.NewCheckRunRepository(resultsDir)
    if err != nil {
        return nil, err
    }

    auditRepo, err := json.NewAuditRepository(resultsDir)
    if err != nil {
        return nil, err
    }

    // Services
    engagementService := engagement.NewService(engagementRepo)
    orchestrator := check.NewOrchestrator(engagementRepo, checkRunRepo, auditRepo)
    auditService := audit.NewService(auditRepo)

    return &AppContext{
        EngagementService: engagementService,
        CheckOrchestrator: orchestrator,
        AuditService:      auditService,
    }, nil
}
```

**Use in commands**:

```go
// In a Cobra command
func runCheckHTTPCommand(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    appCtx := getAppContext() // Get from global or persistent flags

    // Use services
    err := appCtx.EngagementService.ValidateEngagementForChecks(ctx, engagementID, target)
    if err != nil {
        return err
    }

    checkRun, err := appCtx.CheckOrchestrator.CreateCheckRun(ctx, engagementID, operator)
    // ... rest of implementation
}
```

### Pattern: Error Handling

**Use domain errors**:

```go
import "github.com/khanhnv2901/seca-cli/internal/shared/errors"

err := service.GetEngagement(ctx, id)
if errors.Is(err, sharedErrors.ErrEngagementNotFound) {
    return fmt.Errorf("engagement %s not found", id)
}

err = service.ValidateEngagementForChecks(ctx, id, target)
if errors.Is(err, sharedErrors.ErrEngagementUnauthorized) {
    return fmt.Errorf("ROE not acknowledged for engagement %s", id)
}
if errors.Is(err, sharedErrors.ErrTargetNotInScope) {
    return fmt.Errorf("target %s is not in engagement scope", target)
}
```

## Testing

### Unit Testing Domain Entities

```go
// internal/domain/engagement/engagement_test.go
func TestEngagement_AcknowledgeROE(t *testing.T) {
    eng, _ := engagement.NewEngagement("Test", "alice", "ROE text", []string{"example.com"})

    // Initially not authorized
    assert.False(t, eng.IsAuthorized())

    // Acknowledge
    err := eng.AcknowledgeROE()
    assert.NoError(t, err)

    // Now authorized
    assert.True(t, eng.IsAuthorized())
}
```

### Integration Testing with Mock Repositories

```go
// internal/application/engagement/service_test.go
type mockEngagementRepository struct {
    engagements map[string]*engagement.Engagement
}

func (m *mockEngagementRepository) Save(ctx context.Context, eng *engagement.Engagement) error {
    m.engagements[eng.ID()] = eng
    return nil
}

func (m *mockEngagementRepository) FindByID(ctx context.Context, id string) (*engagement.Engagement, error) {
    if eng, ok := m.engagements[id]; ok {
        return eng, nil
    }
    return nil, sharedErrors.ErrEngagementNotFound
}

func TestEngagementService_CreateEngagement(t *testing.T) {
    repo := &mockEngagementRepository{engagements: make(map[string]*engagement.Engagement)}
    service := engagement.NewService(repo)

    eng, err := service.CreateEngagement(context.Background(), "Test", "alice", "ROE", []string{"example.com"})
    assert.NoError(t, err)
    assert.NotNil(t, eng)

    // Verify saved
    found, err := service.GetEngagement(context.Background(), eng.ID())
    assert.NoError(t, err)
    assert.Equal(t, eng.ID(), found.ID())
}
```

## Backward Compatibility

The old `cmd/` layer still works with the existing data files. The new DDD structure:

1. **Reads** existing `engagements.json` files
2. **Writes** in the same format
3. **Maintains** CSV audit trail format
4. **Preserves** all existing features

### Gradual Migration

You can migrate commands one at a time:

1. Start with `engagement` commands (create, list, view)
2. Then migrate `check` commands
3. Finally migrate `report` and `audit` commands

The old and new code can coexist during the transition.

## Questions & Support

- **Architecture docs**: See [ARCHITECTURE.md](./ARCHITECTURE.md)
- **Example usage**: Check `examples/` directory
- **Issues**: Report on GitHub

Happy coding! 🚀
