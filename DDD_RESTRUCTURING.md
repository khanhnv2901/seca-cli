# DDD Restructuring Summary

This document summarizes the Domain-Driven Design (DDD) restructuring of SECA-CLI.

## 🎯 Goals Achieved

✅ **Clear separation** between business logic and infrastructure
✅ **Easier testing** with mockable repository interfaces
✅ **Flexible persistence** - can swap JSON → PostgreSQL/MongoDB
✅ **Better maintainability** with well-defined layer boundaries
✅ **Future-ready** for multi-tenancy and distributed operations

## 📁 New Structure

```
cmd/                                    → CLI layer (presentation)
internal/
  ├── domain/                           → Business entities & rules
  │   ├── engagement/                   → Engagement aggregate
  │   ├── check/                        → CheckRun aggregate
  │   └── audit/                        → AuditTrail aggregate
  ├── application/                      → Use cases & orchestration
  │   ├── engagement/                   → EngagementService
  │   ├── check/                        → CheckOrchestrator
  │   └── audit/                        → AuditService
  ├── infrastructure/                   → External implementations
  │   ├── persistence/json/             → JSON file repositories
  │   ├── checker/                      → Security check implementations
  │   ├── compliance/                   → Framework definitions
  │   └── api/                          → REST API server
  └── shared/                           → Shared utilities
      ├── security/                     → Path security
      ├── constants/                    → Constants
      └── errors/                       → Domain errors
```

## 🔄 What Changed

### Before

```
internal/checker/        → Security checks
internal/compliance/     → Frameworks
internal/api/            → API server
internal/constants/      → Constants
internal/security/       → Security utils
```

### After

```
internal/infrastructure/checker/      → Security checks (MOVED)
internal/infrastructure/compliance/   → Frameworks (MOVED)
internal/infrastructure/api/          → API server (MOVED)
internal/shared/constants/            → Constants (MOVED)
internal/shared/security/             → Security utils (MOVED)

internal/domain/                      → Business entities (NEW)
internal/application/                 → Use cases (NEW)
internal/infrastructure/persistence/  → Repositories (NEW)
```

## 🚀 Key Components

### 1. Domain Entities (NEW)

**Engagement** - Authorized testing engagement
- Enforces ROE acknowledgment
- Validates scope
- Time-bound testing windows

**CheckRun** - Security check execution
- Owns check results
- Tracks execution metadata
- Links to audit trail

**AuditTrail** - Immutable evidence
- Append-only entries
- Cryptographic sealing
- GPG signature support

### 2. Application Services (NEW)

**EngagementService** - Manages engagements
```go
service.CreateEngagement(ctx, name, owner, roe, scope)
service.AcknowledgeROE(ctx, id)
service.ValidateEngagementForChecks(ctx, id, target)
```

**CheckOrchestrator** - Coordinates checks
```go
orchestrator.CreateCheckRun(ctx, engagementID, operator)
orchestrator.AddCheckResult(ctx, checkRun, result)
orchestrator.FinalizeCheckRun(ctx, checkRun, hash, algorithm)
```

**AuditService** - Manages audit trails
```go
auditService.RecordCheckExecution(ctx, engagementID, operator, command, target, status, ...)
auditService.SealAuditTrail(ctx, engagementID, algorithm)
auditService.VerifyIntegrity(ctx, engagementID)
```

### 3. Repositories (NEW)

**Engagement Repository**
- JSON file implementation
- Save/FindByID/FindAll/Delete operations
- Thread-safe with mutex

**CheckRun Repository**
- Stores in `results/<engagement-id>/http_results.json`
- Supports multiple check runs per engagement

**Audit Repository**
- CSV format (backward compatible)
- Hash computation & verification
- Signature file support

## 📊 Benefits

### 1. Testability

**Before**: Testing required file system access
```go
func TestEngagement(t *testing.T) {
    // Had to create temp files
    tmpDir := t.TempDir()
    engagementsFile := filepath.Join(tmpDir, "engagements.json")
    // Complex setup...
}
```

**After**: Mock repositories for isolated tests
```go
func TestEngagement(t *testing.T) {
    mockRepo := &MockRepository{}
    service := engagement.NewService(mockRepo)
    // Pure business logic testing
}
```

### 2. Flexibility

**Swap storage backends without changing business logic**:

```go
// JSON storage
repo := json.NewEngagementRepository(dataDir)

// PostgreSQL storage (future)
repo := postgres.NewEngagementRepository(db)

// Redis cache (future)
repo := cache.NewCachedRepository(baseRepo, redisClient)

// Same service, different storage!
service := engagement.NewService(repo)
```

### 3. Maintainability

**Business rules in one place**:

```go
// internal/domain/engagement/engagement.go
func (e *Engagement) IsAuthorized() bool {
    return e.roeAgree  // Single source of truth
}

func (e *Engagement) IsInScope(target string) bool {
    for _, s := range e.scope {
        if s == target {
            return true
        }
    }
    return false
}
```

### 4. Scalability

**Ready for distributed operations**:

- Multi-tenancy via repository filtering
- Background job queues for async checks
- Horizontal scaling of API servers
- Event sourcing for audit compliance

## 🔧 Migration

### Imports Updated

All imports automatically migrated:

```go
// Old
import "github.com/khanhnv2901/seca-cli/internal/checker"

// New
import "github.com/khanhnv2901/seca-cli/internal/infrastructure/checker"
```

### Backward Compatibility

✅ Reads existing `engagements.json`
✅ Writes in same format
✅ CSV audit trail unchanged
✅ All existing features preserved

### Tests

✅ **All tests passing**

```bash
$ go test ./...
ok      github.com/khanhnv2901/seca-cli                          0.004s
ok      github.com/khanhnv2901/seca-cli/cmd                      0.018s
ok      github.com/khanhnv2901/seca-cli/internal/infrastructure/api       0.082s
ok      github.com/khanhnv2901/seca-cli/internal/infrastructure/checker  10.013s
ok      github.com/khanhnv2901/seca-cli/internal/shared/security  0.002s
```

## 📚 Documentation

- **[ARCHITECTURE.md](docs/ARCHITECTURE.md)** - Detailed architecture guide
- **[MIGRATION_GUIDE.md](docs/MIGRATION_GUIDE.md)** - How to adapt existing code

## 🎓 Quick Examples

### Create Engagement

```go
import (
    "github.com/khanhnv2901/seca-cli/internal/infrastructure/persistence/json"
    engagementapp "github.com/khanhnv2901/seca-cli/internal/application/engagement"
)

// Setup
repo, _ := json.NewEngagementRepository(dataDir)
service := engagementapp.NewService(repo)

// Use
eng, err := service.CreateEngagement(ctx, "Acme Corp", "alice", roe, scope)
err = service.AcknowledgeROE(ctx, eng.ID())
```

### Run Checks

```go
import (
    checkapp "github.com/khanhnv2901/seca-cli/internal/application/check"
)

// Setup
orchestrator := checkapp.NewOrchestrator(engagementRepo, checkRunRepo, auditRepo)

// Use
checkRun, _ := orchestrator.CreateCheckRun(ctx, engagementID, operator)

for _, target := range targets {
    result, _ := check.NewResult(target, check.CheckStatusOK)
    orchestrator.AddCheckResult(ctx, checkRun, result)
}

hash, _ := orchestrator.SealAuditTrail(ctx, engagementID, "sha256")
orchestrator.FinalizeCheckRun(ctx, checkRun, hash, "sha256")
```

### Verify Audit

```go
import (
    auditapp "github.com/khanhnv2901/seca-cli/internal/application/audit"
)

// Setup
auditService := auditapp.NewService(auditRepo)

// Use
valid, err := auditService.VerifyIntegrity(ctx, engagementID)
if !valid {
    fmt.Println("Audit trail compromised!")
}
```

## 🔮 Future Enhancements

With the DDD architecture, these are now possible:

### 1. Database Migration

```go
// Just swap the repository!
repo := postgres.NewEngagementRepository(db)
service := engagement.NewService(repo)
```

### 2. Caching Layer

```go
baseRepo := postgres.NewEngagementRepository(db)
cachedRepo := cache.NewCachedRepository(baseRepo, redisClient)
service := engagement.NewService(cachedRepo)
```

### 3. Event Sourcing

```go
eventStore := eventsourcing.NewEventStore(db)
engagement := engagement.Replay(eventStore.GetEvents(engagementID))
```

### 4. Multi-Tenancy

```go
repo := postgres.NewEngagementRepository(db)
tenantRepo := multitenant.WrapRepository(repo, tenantID)
service := engagement.NewService(tenantRepo)
```

## ✅ Checklist

- [x] Create domain entities (Engagement, CheckRun, AuditTrail)
- [x] Define repository interfaces
- [x] Implement JSON repositories
- [x] Create application services
- [x] Move infrastructure components
- [x] Update all imports
- [x] All tests passing
- [x] Documentation complete

## 🎉 Result

A clean, maintainable, and scalable architecture ready for enterprise deployments!

---

**For detailed information, see:**
- [ARCHITECTURE.md](docs/ARCHITECTURE.md)
- [MIGRATION_GUIDE.md](docs/MIGRATION_GUIDE.md)
