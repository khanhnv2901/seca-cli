# SECA-CLI Architecture (Domain-Driven Design)

This document describes the Domain-Driven Design (DDD) architecture of SECA-CLI, a security assessment command-line tool.

## Table of Contents

- [Overview](#overview)
- [Architecture Layers](#architecture-layers)
- [Domain Layer](#domain-layer)
- [Application Layer](#application-layer)
- [Infrastructure Layer](#infrastructure-layer)
- [Shared Kernel](#shared-kernel)
- [Migration Guide](#migration-guide)

## Overview

SECA-CLI follows a clean, layered Domain-Driven Design architecture that separates business logic from infrastructure concerns. This separation enables:

- **Testability**: Business logic can be tested independently from infrastructure
- **Flexibility**: Easy to swap storage backends (JSON → PostgreSQL/MongoDB)
- **Maintainability**: Clear boundaries between layers reduce coupling
- **Scalability**: Support for future multi-tenancy and distributed operations

## Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                       cmd/ (CLI Layer)                       │
│           Cobra commands, flags, user interaction            │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│            internal/application/ (Use Cases)                 │
│        EngagementService, CheckOrchestrator, etc.            │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│              internal/domain/ (Business Logic)               │
│    Engagement, CheckRun, AuditTrail entities + rules         │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│           internal/infrastructure/ (External)                │
│   Persistence, Checkers, Compliance, API, Cache, Queue       │
└─────────────────────────────────────────────────────────────┘
```

### Directory Structure

```
cmd/                                    → CLI/API presentation layer
internal/
  ├── domain/                           → Core business entities
  │   ├── engagement/                   → Engagement aggregate root
  │   │   ├── engagement.go             → Engagement entity with business rules
  │   │   └── repository.go             → Repository interface
  │   ├── check/                        → Check domain models
  │   │   ├── check_run.go              → CheckRun aggregate root
  │   │   ├── result.go                 → CheckResult value object
  │   │   └── repository.go             → Repository interface
  │   ├── audit/                        → Audit domain models
  │   │   ├── audit_trail.go            → AuditTrail entity
  │   │   └── repository.go             → Repository interface
  │   └── compliance/                   → Compliance framework models
  ├── application/                      → Use cases / application services
  │   ├── engagement/                   → EngagementService
  │   │   └── service.go                → Create, validate, manage engagements
  │   ├── check/                        → CheckOrchestrator
  │   │   └── orchestrator.go           → Coordinate check execution
  │   ├── report/                       → ReportGenerator (planned)
  │   └── audit/                        → AuditService
  │       └── service.go                → Audit trail management
  ├── infrastructure/                   → External concerns
  │   ├── persistence/                  → Data storage
  │   │   ├── json/                     → JSON file-based implementation
  │   │   │   ├── engagement_repository.go
  │   │   │   ├── check_run_repository.go
  │   │   │   └── audit_repository.go
  │   │   └── repository/               → Future: SQL implementations
  │   ├── checker/                      → Check implementations
  │   │   ├── http.go                   → HTTP/HTTPS checker
  │   │   ├── dns.go                    → DNS checker
  │   │   ├── network.go                → Network security checker
  │   │   ├── tls_compliance.go         → TLS compliance checker
  │   │   └── ...                       → Other checkers
  │   ├── compliance/                   → Compliance framework definitions
  │   │   ├── framework.go              → Framework models
  │   │   └── mappings.go               → Check-to-requirement mappings
  │   ├── api/                          → REST/GraphQL API servers
  │   │   ├── server.go                 → HTTP server
  │   │   └── middleware/               → Request middleware
  │   ├── queue/                        → Background job queues (planned)
  │   └── cache/                        → Caching layer (planned)
  └── shared/                           → Shared kernel
      ├── security/                     → Path traversal prevention
      ├── constants/                    → Application constants
      └── errors/                       → Domain errors
```

## Domain Layer

The domain layer contains the core business entities and their business rules. It has **no dependencies** on external frameworks or infrastructure.

### Aggregates

#### 1. Engagement Aggregate

**Root Entity**: `Engagement`

Represents an authorized security testing engagement. Enforces:
- ROE (Rules of Engagement) must be acknowledged
- Scope defines authorized targets
- Time-bound testing windows

**Key Methods**:
```go
func NewEngagement(name, owner, roe string, scope []string) (*Engagement, error)
func (e *Engagement) AcknowledgeROE() error
func (e *Engagement) AddToScope(target string) error
func (e *Engagement) IsAuthorized() bool
func (e *Engagement) IsActive() bool
```

**Repository Interface**:
```go
type Repository interface {
    Save(ctx context.Context, engagement *Engagement) error
    FindByID(ctx context.Context, id string) (*Engagement, error)
    FindAll(ctx context.Context) ([]*Engagement, error)
    Delete(ctx context.Context, id string) error
}
```

#### 2. CheckRun Aggregate

**Root Entity**: `CheckRun`

Represents an execution of security checks against an engagement's scope.

**Members**:
- `Result` entities - Individual check results per target
- `Metadata` - Hash, signature, statistics

**Key Methods**:
```go
func NewCheckRun(engagementID, engagementName, operator string) (*CheckRun, error)
func (cr *CheckRun) Start() error
func (cr *CheckRun) Complete() error
func (cr *CheckRun) AddResult(result *Result) error
func (cr *CheckRun) SetAuditHash(hash, algorithm string) error
```

#### 3. AuditTrail Aggregate

**Root Entity**: `AuditTrail`

Immutable audit trail for evidence integrity. Once sealed, no entries can be added.

**Key Methods**:
```go
func NewAuditTrail(engagementID string) (*AuditTrail, error)
func (at *AuditTrail) AppendEntry(entry *Entry) error
func (at *AuditTrail) Seal(hash, algorithm string) error
func (at *AuditTrail) Sign(signature string) error
func (at *AuditTrail) VerifyIntegrity(computedHash string) bool
```

### Value Objects

- `CheckStatus` - "ok" | "error"
- `RunStatus` - "pending" | "running" | "completed" | "failed"
- `SecurityHeadersResult` - Security header analysis
- `TLSComplianceResult` - TLS compliance findings
- `NetworkSecurityResult` - Network security findings

### Business Rules

All business rules are enforced in the domain layer:

1. **Engagement Authorization**: ROE must be acknowledged before running checks
2. **Scope Validation**: Targets must be within engagement scope
3. **Time Windows**: Engagements can have active/inactive periods
4. **Audit Immutability**: Once sealed, audit trails cannot be modified
5. **Hash Integrity**: Only SHA256 and SHA512 are supported

## Application Layer

Application services coordinate domain entities and infrastructure to implement use cases.

### EngagementService

Manages the lifecycle of engagements.

**Use Cases**:
- Create engagement
- Acknowledge ROE
- Add/remove targets from scope
- Validate engagement for checks
- Delete engagement

**Example**:
```go
service := engagement.NewService(engagementRepo)
eng, err := service.CreateEngagement(ctx, "Acme Corp Assessment", "alice", roe, scope)
err = service.AcknowledgeROE(ctx, eng.ID())
err = service.ValidateEngagementForChecks(ctx, eng.ID(), target)
```

### CheckOrchestrator

Coordinates check execution across multiple components.

**Use Cases**:
- Create and start check runs
- Add results to check runs
- Finalize check runs with audit hashing
- Retrieve check run history

**Example**:
```go
orchestrator := check.NewOrchestrator(engagementRepo, checkRunRepo, auditRepo)
checkRun, err := orchestrator.CreateCheckRun(ctx, engagementID, operator)
err = orchestrator.AddCheckResult(ctx, checkRun, result)
err = orchestrator.FinalizeCheckRun(ctx, checkRun, auditHash, "sha256")
```

### AuditService

Manages audit trails and evidence integrity.

**Use Cases**:
- Record check executions
- Seal audit trails with cryptographic hash
- Verify audit trail integrity
- Sign audit trails with GPG

**Example**:
```go
auditService := audit.NewService(auditRepo)
err := auditService.RecordCheckExecution(ctx, engagementID, operator, "check http", target, "ok", 200, tlsExpiry, notes, "", duration)
hash, err := auditService.SealAuditTrail(ctx, engagementID, "sha256")
valid, err := auditService.VerifyIntegrity(ctx, engagementID)
```

## Infrastructure Layer

The infrastructure layer provides concrete implementations of interfaces defined in the domain layer.

### Persistence

#### JSON File Storage

Current implementation using JSON files and CSV for audit trails.

**Location**: `internal/infrastructure/persistence/json/`

**Implementations**:
- `EngagementRepository` - Stores engagements in `engagements.json`
- `CheckRunRepository` - Stores check runs in `results/<engagement-id>/http_results.json`
- `AuditRepository` - Stores audit trails in `results/<engagement-id>/audit.csv`

**Features**:
- Thread-safe with mutex locks
- Path traversal protection
- Automatic hash file generation
- GPG signature support

**Future Implementations** (Planned):
- `PostgresEngagementRepository`
- `MongoCheckRunRepository`
- `RedisAuditCache`

### Checkers

Security check implementations following the `Checker` interface.

**Location**: `internal/infrastructure/checker/`

**Interface**:
```go
type Checker interface {
    Check(ctx context.Context, target string) CheckResult
    Name() string
}
```

**Implementations**:
- `HTTPChecker` - HTTP/HTTPS, TLS, security headers
- `DNSChecker` - DNS records (A, AAAA, MX, NS, TXT)
- `NetworkChecker` - Port scanning, subdomain takeover
- `TLSAdvancedChecker` - Advanced TLS analysis
- `ClientSecurityChecker` - Vulnerable libraries, CSRF, XSS
- `CORSChecker` - CORS policy validation

### Compliance Frameworks

Framework definitions and check-to-requirement mappings.

**Location**: `internal/infrastructure/compliance/`

**Supported Frameworks**:
- ISO 27001:2022 (Global)
- ISO 27701:2019 (Privacy)
- JIS Q 27001/27002 (Japan)
- PrivacyMark/Pマーク (Japan)
- FISC Security Guidelines (Japan Financial)
- PDPA (Singapore)
- MTCS SS 584 (Singapore Cloud)
- K-ISMS (South Korea)

### API Server

RESTful API for remote operations.

**Location**: `internal/infrastructure/api/`

**Endpoints**:
- `GET /health` - Health check
- `GET /ready` - Readiness probe
- `GET /engagements` - List engagements
- `POST /engagements` - Create engagement
- `GET /results/{id}` - Get check results
- `POST /jobs` - Start background check job
- `GET /jobs/{id}/stream` - Stream job output (SSE)

**Features**:
- Request ID middleware
- Structured logging with Zap
- Rate limiting per IP
- CORS support
- Bearer token authentication

## Shared Kernel

Common utilities shared across all layers.

### Security

**Location**: `internal/shared/security/`

**Features**:
- `ResolveWithin()` - Prevent path traversal attacks
- `IsValidPath()` - Validate file paths

### Constants

**Location**: `internal/shared/constants/`

**Contents**:
- Default file permissions
- Timeout values
- Standard headers

### Errors

**Location**: `internal/shared/errors/`

**Domain Errors**:
- `ErrEngagementNotFound`
- `ErrEngagementUnauthorized`
- `ErrTargetNotInScope`
- `ErrAuditTrailSealed`
- `ErrCheckRunNotStarted`

## Migration Guide

### How to Use the New Architecture

#### 1. Creating an Engagement

**Old Way (cmd/ layer directly)**:
```go
engagements := loadEngagements()
eng := Engagement{
    ID:    time.Now().UnixNano(),
    Name:  name,
    Owner: owner,
}
engagements = append(engagements, eng)
saveEngagements(engagements)
```

**New Way (via Application Service)**:
```go
// Initialize repository
repo, _ := json.NewEngagementRepository(dataDir)

// Use service
service := engagement.NewService(repo)
eng, err := service.CreateEngagement(ctx, name, owner, roe, scope)
```

#### 2. Running Checks

**Old Way**:
```go
results := []CheckResult{}
for _, target := range targets {
    result := checker.Check(ctx, target)
    results = append(results, result)
}
saveToFile(results)
```

**New Way (via Orchestrator)**:
```go
// Initialize dependencies
engagementRepo, _ := json.NewEngagementRepository(dataDir)
checkRunRepo, _ := json.NewCheckRunRepository(resultsDir)
auditRepo, _ := json.NewAuditRepository(resultsDir)

// Create orchestrator
orchestrator := check.NewOrchestrator(engagementRepo, checkRunRepo, auditRepo)

// Create check run
checkRun, _ := orchestrator.CreateCheckRun(ctx, engagementID, operator)

// Add results
for _, target := range targets {
    result := checker.Check(ctx, target)
    domainResult, _ := check.NewResult(result.Target, check.CheckStatus(result.Status))
    orchestrator.AddCheckResult(ctx, checkRun, domainResult)
}

// Finalize
hash, _ := orchestrator.SealAuditTrail(ctx, engagementID, "sha256")
orchestrator.FinalizeCheckRun(ctx, checkRun, hash, "sha256")
```

### Benefits of the New Architecture

1. **Testability**
   - Domain entities can be tested without file I/O
   - Repository interfaces can be mocked easily
   - Application services are pure business logic

2. **Flexibility**
   - Swap JSON storage for PostgreSQL without changing domain/application layers
   - Add caching layer transparently
   - Support multiple storage backends simultaneously

3. **Maintainability**
   - Business rules in one place (domain entities)
   - Clear separation of concerns
   - Easier to onboard new developers

4. **Scalability**
   - Multi-tenancy support via repository filtering
   - Distributed checks via message queues
   - Horizontal scaling of API servers

### Future Enhancements

The DDD architecture enables:

1. **Database Migration**
   ```go
   // Drop-in replacement
   repo := postgres.NewEngagementRepository(db)
   service := engagement.NewService(repo)
   ```

2. **Caching Layer**
   ```go
   baseRepo := postgres.NewEngagementRepository(db)
   cachedRepo := cache.NewCachedRepository(baseRepo, redisClient)
   ```

3. **Event Sourcing**
   ```go
   eventStore := eventsourcing.NewEventStore(db)
   engagement := engagement.Replay(eventStore.GetEvents(engagementID))
   ```

4. **Multi-Tenancy**
   ```go
   repo := postgres.NewEngagementRepository(db)
   tenantRepo := multitenant.WrapRepository(repo, tenantID)
   ```

## Conclusion

The DDD architecture provides a solid foundation for SECA-CLI's evolution. By separating concerns into distinct layers, we achieve:

- Clean, testable code
- Flexibility to adapt to changing requirements
- Scalability for enterprise deployments
- Maintainability for long-term success

For questions or contributions, please refer to the [CONTRIBUTING.md](../CONTRIBUTING.md) guide.
