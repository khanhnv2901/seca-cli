# SECA-CLI Test Suite

This directory contains unit tests and integration tests for SECA-CLI.

## Test Structure

```
tests/
├── README.md              # This file
└── integration_test.sh    # Integration test script
```

Unit tests are located in the `cmd/` directory alongside the source files:
- `cmd/engagement_test.go` - Tests for engagement management
- `cmd/audit_test.go` - Tests for audit and hashing functions
- `cmd/check_test.go` - Tests for HTTP check functionality

## Running Tests

### Run All Unit Tests

```bash
# From project root
go test ./cmd/... -v

# Or using make
make test
```

### Run Specific Test File

```bash
go test ./cmd/engagement_test.go ./cmd/engagement.go -v
go test ./cmd/audit_test.go ./cmd/audit.go -v
go test ./cmd/check_test.go ./cmd/check.go -v
```

### Run Tests with Coverage

```bash
go test ./cmd/... -cover

# Generate coverage report
go test ./cmd/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run Integration Tests

```bash
# Build the binary first
make build

# Run integration tests
./tests/integration_test.sh

# Or using make
make integration-test
```

## Test Categories

### Unit Tests

**Engagement Tests** (`cmd/engagement_test.go`)
- Test engagement creation and loading
- Test engagement persistence (save/load)
- Test JSON marshaling/unmarshaling
- Test multiple engagements handling

**Audit Tests** (`cmd/audit_test.go`)
- Test audit row appending
- Test CSV format and headers
- Test multiple audit entries
- Test error recording in audit
- Test raw capture saving
- Test SHA256 hash generation and verification
- Test hash file format

**Check Tests** (`cmd/check_test.go`)
- Test CheckResult structure
- Test HTTP status handling
- Test error conditions
- Test JSON serialization
- Test various HTTP status codes
- Test timeout handling
- Test robots.txt checking
- Test HEAD vs GET requests

### Integration Tests

The integration test script (`integration_test.sh`) performs end-to-end testing:

1. Binary existence check
2. Help command validation
3. Engagement creation
4. Engagement listing
5. Scope addition
6. ROE confirmation enforcement
7. HTTP checks execution
8. Results directory structure validation
9. Hash integrity verification
10. Audit CSV format validation
11. Results JSON format validation
12. Makefile target testing

## Test Requirements

### For Unit Tests
- Go 1.21 or higher
- Dependencies from `go.mod`

### For Integration Tests
- Built SECA-CLI binary (`./seca`)
- `sha256sum` command (for hash verification)
- `jq` command (optional, for JSON validation)
- Internet connectivity (for HTTP checks to example.com)

## Writing New Tests

### Unit Test Example

```go
func TestNewFeature(t *testing.T) {
    // Setup
    input := "test data"

    // Execute
    result := YourFunction(input)

    // Assert
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

### Table-Driven Test Example

```go
func TestMultipleScenarios(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"Case 1", "input1", "output1"},
        {"Case 2", "input2", "output2"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := YourFunction(tt.input)
            if result != tt.expected {
                t.Errorf("Expected %v, got %v", tt.expected, result)
            }
        })
    }
}
```

## Continuous Integration

To run tests in CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go test ./cmd/... -v
      - run: make build
      - run: ./tests/integration_test.sh
```

## Test Coverage Goals

- Unit test coverage: >80%
- Critical paths: 100%
- Edge cases: Well covered
- Error handling: Fully tested

## Troubleshooting

### Tests Fail Due to File Permissions
```bash
chmod +x ./tests/integration_test.sh
```

### Tests Fail Due to Missing Binary
```bash
make build
```

### Tests Fail Due to Port Conflicts
The tests use httptest.NewServer which automatically assigns available ports, so this should not be an issue.

### Clean Test Artifacts
```bash
# Clean unit test artifacts
go clean -testcache

# Clean integration test artifacts
rm -rf ./test_results_integration
rm -f ./test_engagements_integration.json
```

## Contributing Tests

When contributing new features:

1. Write unit tests for new functions
2. Ensure existing tests pass
3. Add integration test scenarios if needed
4. Update test documentation
5. Run all tests before submitting PR

## License

Tests are part of the SECA-CLI project and follow the same license.
