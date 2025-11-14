# Phase 2: CMD Layer Migration to DDD Services

This document tracks the migration of the `cmd/` layer to use the new DDD application services.

## 🎯 Goals

- Migrate CLI commands to use application services instead of direct file I/O
- Maintain backward compatibility with existing data formats
- Keep all existing functionality working
- Improve code maintainability and testability

## ✅ Completed

### 1. Service Container (NEW)

**File**: `internal/application/container.go`

Created a dependency injection container that initializes all services:

```go
type Container struct {
    // Repositories
    EngagementRepo engagement.Repository
    CheckRunRepo   check.Repository
    AuditRepo      audit.Repository

    // Services
    EngagementService *engagementapp.Service
    CheckOrchestrator *checkapp.Orchestrator
    AuditService      *auditapp.Service
}

func NewContainer(dataDir, resultsDir string) (*Container, error)
```

**Benefits**:
- Single initialization point for all services
- Clear dependency graph
- Easy to test with mock implementations
- Supports future dependency injection frameworks

### 2. AppContext Enhancement

**File**: `cmd/root.go`

Updated `AppContext` to include the service container:

```go
type AppContext struct {
    Logger     *zap.SugaredLogger
    Operator   string
    ResultsDir string
    Config     *CLIConfig
    Services   *application.Container // NEW
}
```

Services are initialized once during `PersistentPreRunE`:

```go
services, err := application.NewContainer(dataDir, appCtx.ResultsDir)
if err != nil {
    return fmt.Errorf("failed to initialize services: %w", err)
}
appCtx.Services = services
```

### 3. Engagement Commands Migration ✅

**File**: `cmd/engagement_ddd.go` (NEW)

Migrated all engagement commands to use `EngagementService`:

#### Before (Direct File I/O)
```go
func loadEngagements() []Engagement {
    data, _ := os.ReadFile(engagementsFilePath)
    var engagements []Engagement
    json.Unmarshal(data, &engagements)
    return engagements
}

engagements := loadEngagements()
engagements = append(engagements, newEngagement)
saveEngagements(engagements)
```

#### After (Application Service)
```go
ctx := context.Background()
appCtx := getAppContext(cmd)

eng, err := appCtx.Services.EngagementService.CreateEngagement(
    ctx, name, owner, roe, scope
)
if err != nil {
    return fmt.Errorf("failed to create engagement: %w", err)
}
```

**Commands Migrated**:
- ✅ `engagement create` - Uses `EngagementService.CreateEngagement()`
- ✅ `engagement list` - Uses `EngagementService.ListEngagements()`
- ✅ `engagement view` - Uses `EngagementService.GetEngagement()`
- ✅ `engagement add-scope` - Uses `EngagementService.AddToScope()`
- ✅ `engagement remove-scope` - Uses `EngagementService.RemoveFromScope()`
- ✅ `engagement delete` - Uses `EngagementService.DeleteEngagement()`

**Backward Compatibility**:
- ✅ Reads existing `engagements.json` files
- ✅ Writes in same JSON format
- ✅ All existing data preserved
- ✅ Output format unchanged (uses DTO conversion)

## 📊 Testing Results

### Unit Tests
```bash
$ go test ./cmd/... -v
✅ All tests passing
```

### Integration Tests
```bash
# Create engagement
$ ./seca engagement create --name "Test DDD" --owner "tester" \
    --roe "Test ROE" --roe-agree --scope "example.com,test.com"
✅ Created engagement Test DDD (id=20251114143102-000000)

# List engagements
$ ./seca engagement list
✅ Shows both old and new engagements correctly

# View engagement
$ ./seca engagement view --id 20251114143102-000000
✅ Displays engagement details in JSON format

# Add scope
$ ./seca engagement add-scope --id 20251114143102-000000 --scope "new.com"
✅ Success: added 1 scope entries to engagement

# Remove scope
$ ./seca engagement remove-scope --id 20251114143102-000000 --domain "new.com"
✅ Success: removed 1 scope entries from engagement
```

### Full Test Suite
```bash
$ go test ./...
✅ All packages passing
```

## 🏗️ Architecture Benefits

### 1. Separation of Concerns

**Before**:
```go
// Business logic mixed with CLI
func createEngagement(cmd *cobra.Command, args []string) error {
    // Parse flags
    name, _ := cmd.Flags().GetString("name")

    // Validate
    if name == "" { return errors.New("name required") }

    // Business logic
    eng := Engagement{ID: generateID(), Name: name}

    // File I/O
    data, _ := os.ReadFile(file)
    engagements = append(engagements, eng)
    os.WriteFile(file, data, 0644)

    // Output
    fmt.Println("Created")
}
```

**After**:
```go
// CLI handles presentation only
func createEngagement(cmd *cobra.Command, args []string) error {
    // Parse flags
    name, _ := cmd.Flags().GetString("name")

    // Delegate to service
    eng, err := appCtx.Services.EngagementService.CreateEngagement(ctx, name, owner, roe, scope)
    if err != nil {
        return err
    }

    // Output
    fmt.Println("Created")
}

// Service handles business logic
func (s *Service) CreateEngagement(ctx, name, owner, roe, scope) {
    // Validation
    // Business rules
    // Persistence via repository
}
```

### 2. Testability

**Before**: Must mock file system
```go
func TestCreateEngagement(t *testing.T) {
    tmpDir := t.TempDir()
    os.Setenv("DATA_DIR", tmpDir)
    // Complex setup...
}
```

**After**: Mock service interface
```go
func TestCreateEngagement(t *testing.T) {
    mockService := &MockEngagementService{}
    appCtx.Services.EngagementService = mockService
    // Simple, fast test
}
```

### 3. Flexibility

**Example**: Swap storage backend without touching commands

```go
// Development: JSON files
services, _ := application.NewContainer(dataDir, resultsDir)

// Production: PostgreSQL (future)
services, _ := application.NewContainerWithDB(db)

// Same commands, different storage!
```

## 📝 Code Patterns

### Pattern 1: Service Access

All commands follow this pattern:

```go
func commandHandler(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    appCtx := getAppContext(cmd)

    // Use service
    result, err := appCtx.Services.SomeService.SomeMethod(ctx, ...)
    if err != nil {
        return fmt.Errorf("operation failed: %w", err)
    }

    // Display result
    fmt.Println(result)
    return nil
}
```

### Pattern 2: Domain Error Handling

```go
import sharedErrors "github.com/khanhnv2901/seca-cli/internal/shared/errors"

err := appCtx.Services.EngagementService.GetEngagement(ctx, id)
if errors.Is(err, sharedErrors.ErrEngagementNotFound) {
    return fmt.Errorf("engagement %s not found", id)
}
```

### Pattern 3: DTO Conversion

Domain entities → DTOs for JSON output:

```go
func engagementToDTO(eng *engagement.Engagement) engagementDTO {
    return engagementDTO{
        ID:        eng.ID(),
        Name:      eng.Name(),
        Owner:     eng.Owner(),
        // ... all fields
    }
}

// In command
eng, _ := service.GetEngagement(ctx, id)
dto := engagementToDTO(eng)
json.Marshal(dto)  // Output to user
```

## 🔄 Pending Migrations

### Check Commands (Next)
- [ ] `check http` - Use `CheckOrchestrator`
- [ ] `check dns` - Use `CheckOrchestrator`
- [ ] `check network` - Use `CheckOrchestrator`

### Audit Commands
- [ ] `audit verify` - Use `AuditService`
- [ ] `audit list` - Use `AuditService`

### API Server
- [ ] Update `cmd/serve.go` to use application services
- [ ] Replace direct repository access with services

## 📈 Metrics

### Lines of Code

**DDD Infrastructure**:
- `container.go`: 59 lines
- `engagement_ddd.go`: 251 lines
- `root.go` changes: +10 lines

**Total new code**: ~320 lines

**Code eliminated**: 0 lines (kept for backward compatibility)

### Complexity Reduction

**Before**: Each command = 3-4 responsibilities
- Parse flags
- Validate input
- Business logic
- File I/O
- Error handling
- Output formatting

**After**: Each command = 2 responsibilities
- Parse flags & validate input
- Delegate to service & format output

**Result**: ~50% complexity reduction per command

## 🎓 Lessons Learned

### 1. Gradual Migration Works

We can migrate commands one at a time:
- Old commands still work
- New commands use DDD
- No breaking changes
- Users don't notice

### 2. Container Pattern Simplifies DI

Single `Container` initialization:
- All services configured once
- Easy to replace for testing
- Clear dependency graph

### 3. DTOs Maintain Compatibility

Domain entities can change internally:
- DTOs preserve JSON format
- Backward compatible output
- Migration invisible to users

## 🚀 Next Steps

1. **Migrate Check Commands**
   - Create `check_ddd.go`
   - Use `CheckOrchestrator`
   - Integrate with `AuditService`

2. **Migrate Audit Commands**
   - Create `audit_ddd.go`
   - Use `AuditService`
   - Maintain CSV format

3. **Update API Server**
   - Use services instead of repositories
   - Share business logic with CLI

4. **Remove Old Code** (after full migration)
   - Delete old command implementations
   - Remove direct file I/O functions
   - Keep only helper utilities

## ✅ Success Criteria

- [x] Service container created
- [x] AppContext includes services
- [x] Engagement commands migrated
- [x] All tests passing
- [x] Backward compatibility maintained
- [x] Documentation updated

## 📚 Documentation

- **Architecture**: See `docs/ARCHITECTURE.md`
- **Migration Guide**: See `docs/MIGRATION_GUIDE.md`
- **Phase 1**: See `DDD_RESTRUCTURING.md`
- **This Document**: Phase 2 status

---

**Status**: ✅ Engagement commands migration complete
**Next**: Check commands migration
**Timeline**: Gradual rollout, no deadline pressure
