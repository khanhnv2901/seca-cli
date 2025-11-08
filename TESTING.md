# SECA-CLI Testing Guide

Complete guide for running and writing tests for SECA-CLI.

## Quick Start

```bash
# Run all unit tests
make test

# Run unit tests with coverage
make test-coverage

# Run integration tests (builds binary first)
make test-integration

# Run all tests
make test-all

# Clean test artifacts
make test-clean
```

## Test Organization

### Unit Tests (in `cmd/` directory)

| Test File | Coverage | Description |
|-----------|----------|-------------|
| `engagement_test.go` | Engagement management | Create, load, save, and list engagements |
| `audit_test.go` | Audit & hashing | CSV audit logs, raw captures, SHA256 hashing |
| `check_test.go` | HTTP checks | HTTP requests, status codes, error handling |

### Integration Tests (in `tests/` directory)

| Test Script | Type | Description |
|-------------|------|-------------|
| `integration_test.sh` | End-to-end | Full workflow from engagement creation to verification |

## Running Tests

### Unit Tests

```bash
# All unit tests
go test ./cmd/... -v

# Specific test file
go test ./cmd/engagement_test.go ./cmd/engagement.go -v

# Run specific test function
go test ./cmd/... -run TestLoadEngagements -v

# With race detection
go test ./cmd/... -race

# With short mode (skip long tests)
go test ./cmd/... -short
```

### Coverage Analysis

```bash
# Generate coverage report
go test ./cmd/... -coverprofile=coverage.out

# View coverage percentage
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# Open in browser
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

### Integration Tests

```bash
# Build and run integration tests
make test-integration

# Or manually
make build
./tests/integration_test.sh
```

## Test Coverage Summary

Current test coverage by package:

- **Engagement Management**: ~95%
  - ✅ Create engagement
  - ✅ Load/save engagements
  - ✅ JSON marshaling
  - ✅ Multiple engagements
  - ✅ Empty file handling

- **Audit Functions**: ~90%
  - ✅ Append audit rows
  - ✅ CSV format validation
  - ✅ Multiple entries
  - ✅ Error recording
  - ✅ Raw capture saving
  - ✅ SHA256 hash generation
  - ✅ Hash file format

- **HTTP Checks**: ~85%
  - ✅ CheckResult structure
  - ✅ HTTP status handling
  - ✅ JSON serialization
  - ✅ Mock server tests
  - ✅ Timeout handling
  - ✅ Error conditions
  - ✅ robots.txt checking

- **Integration Tests**: Full workflow
  - ✅ Binary execution
  - ✅ Engagement creation
  - ✅ Scope management
  - ✅ HTTP checks
  - ✅ Hash verification
  - ✅ File structure validation

## Writing Tests

### Basic Test Template

```go
func TestFeatureName(t *testing.T) {
    // Arrange
    input := "test data"
    expected := "expected result"

    // Act
    result := YourFunction(input)

    // Assert
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

### Table-Driven Tests

```go
func TestMultipleCases(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "result", false},
        {"invalid input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := YourFunction(tt.input)

            if (err != nil) != tt.wantErr {
                t.Errorf("Expected error: %v, got: %v", tt.wantErr, err)
            }

            if result != tt.expected {
                t.Errorf("Expected %v, got %v", tt.expected, result)
            }
        })
    }
}
```

### Using Test Fixtures

```go
func TestWithTempFiles(t *testing.T) {
    // Create temp directory
    tmpDir := "test_temp"
    defer os.RemoveAll(tmpDir)

    // Test code using tmpDir
    // ...
}
```

### Mock HTTP Server

```go
func TestHTTPRequest(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("test response"))
    }))
    defer server.Close()

    // Test code using server.URL
    // ...
}
```

## Continuous Integration

Tests run automatically on:
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop`

See `.github/workflows/test.yml` for CI configuration.

### CI Jobs

1. **Unit Tests** - Run all unit tests with coverage
2. **Integration Tests** - Full end-to-end workflow
3. **Lint** - Code quality checks with golangci-lint
4. **Build** - Cross-platform builds (Linux, macOS, Windows)

## Test Best Practices

### Do's ✅

- Write tests for all new features
- Test error conditions and edge cases
- Use descriptive test names
- Clean up test artifacts (temp files, etc.)
- Use table-driven tests for multiple scenarios
- Test both success and failure paths
- Keep tests fast and focused
- Use subtests for better organization

### Don'ts ❌

- Don't commit test artifacts to git
- Don't write flaky tests (non-deterministic)
- Don't test external services without mocking
- Don't skip error checking in tests
- Don't leave commented-out test code
- Don't write overly complex test logic

## Debugging Tests

### Verbose Output

```bash
go test ./cmd/... -v
```

### Run Failed Tests Only

```bash
go test ./cmd/... -v --count=1  # Disable cache
```

### Print Test Output

```go
func TestDebug(t *testing.T) {
    result := YourFunction()
    t.Logf("Debug: result = %+v", result)  // Use t.Logf for debug output
}
```

### Run With Debugger (VSCode)

Add to `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Test",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${workspaceFolder}/cmd",
            "args": ["-test.run", "TestName"]
        }
    ]
}
```

## Performance Testing

### Benchmark Tests

```go
func BenchmarkFunction(b *testing.B) {
    for i := 0; i < b.N; i++ {
        YourFunction()
    }
}
```

Run benchmarks:

```bash
go test ./cmd/... -bench=. -benchmem
```

## Test Maintenance

### Updating Tests

When modifying code:

1. Update affected tests
2. Run tests locally
3. Ensure coverage doesn't decrease
4. Update test documentation

### Removing Tests

When removing features:

1. Remove corresponding tests
2. Update coverage reports
3. Clean up test fixtures

## Troubleshooting

### Tests Fail Locally But Pass in CI

- Check Go version compatibility
- Verify dependencies are up to date
- Check for OS-specific code

### Tests Are Slow

- Use `-short` flag to skip long tests
- Run tests in parallel: `go test ./cmd/... -parallel 4`
- Profile tests: `go test -cpuprofile=cpu.prof`

### Coverage Not Generating

```bash
# Ensure you're using coverage flags
go test ./cmd/... -coverprofile=coverage.out

# Check file was created
ls -lh coverage.out
```

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Test Fixtures](https://github.com/go-testfixtures/testfixtures)
- [httptest Package](https://golang.org/pkg/net/http/httptest/)

## Getting Help

- Check test output for specific error messages
- Review test documentation in `tests/README.md`
- Check CI logs for additional context
- Open an issue on GitHub with test failures

---

**Last Updated**: 2025-11-09
