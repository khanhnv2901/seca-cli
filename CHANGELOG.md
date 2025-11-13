# Changelog

All notable changes to SECA-CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0] - 2025-01-15

### Added

#### Major Features
- **Plugin System**: Extensible architecture for custom security checkers
  - External plugin support via JSON definitions
  - Plugin API version 1 with backward compatibility commitment
  - Automatic command registration for plugins
  - Plugin directory: `~/.local/share/seca-cli/plugins/`
  - See [Plugin Development Guide](docs/developer-guide/plugin-development.md)

- **Retry Mechanism**: Automatic retry of failed targets
  - `--retry N` flag to retry failed targets up to N times
  - Respects rate limiting and concurrency settings
  - Saves all successful results from any attempt
  - Improves reliability for flaky networks

- **Telemetry & Metrics System**: Success rate tracking and trend analysis
  - `--telemetry` flag to record metrics
  - JSONL storage format for time-series data
  - `seca report telemetry` command with ASCII graphs
  - Success rate trends over time
  - JSON export for external analysis

- **Progress Display**: Live progress bars for long-running scans
  - `--progress` flag for visual feedback
  - Shows completion percentage and elapsed time
  - Useful for large target sets

- **Secure Results Encryption**: GPG encryption for audit logs and results
  - `--secure-results` flag to encrypt evidence
  - Automatic GPG encryption of audit.csv and http_results.json
  - Supports custom GPG recipients
  - Meets encryption-at-rest compliance requirements

- **Multiple Hash Algorithms**: SHA-256 and SHA-512 support
  - `--hash sha512` flag for stricter integrity verification
  - Default remains SHA-256 for performance
  - SHA-512 recommended for high-security environments
  - NIST FIPS 180-4 compliant

- **Interactive TUI**: Terminal UI for engagement management
  - `seca tui` command launches interactive interface
  - Visual engagement browser with arrow key navigation
  - Create, view, delete engagements interactively
  - Color-coded status indicators

#### New Commands

**Engagement Management:**
- `seca engagement view --id <id>` - Display engagement details as JSON
- `seca engagement delete --id <id>` - Delete engagement and all associated data
- `seca tui` - Launch interactive Terminal UI

**Reporting:**
- `seca report stats --id <id>` - Display engagement statistics with multiple format options
- `seca report telemetry --id <id>` - Show success rate trends over time
- `seca report generate --id <id>` - Generate reports in markdown/HTML/PDF/JSON formats

**Check Enhancements:**
- Custom plugin checks: `seca check <plugin-name>`
- Enhanced DNS checks with `--nameservers` flag for custom DNS servers

#### New Flags

**Check Commands:**
- `--retry N` - Retry failed targets N times
- `--progress` - Display live progress bar
- `--telemetry` - Record telemetry metrics
- `--hash sha512` - Use SHA-512 instead of SHA-256
- `--secure-results` - Encrypt results with GPG
- `--nameservers` - Custom DNS nameservers (DNS checks)
- `--dns-timeout` - DNS query timeout in seconds

**Report Commands:**
- `--format` - Output format (table/json/csv/markdown for stats; graph/json for telemetry)

#### Advanced Security Analysis

- **Cookie Security Analysis**: Secure/HttpOnly flag detection (OWASP A1:2021)
- **CORS Policy Inspection**: Cross-origin policy validation (OWASP A5:2021)
- **Third-Party Script Inventory**: Supply-chain risk detection
- **Cache Policy Analysis**: Performance and security cache header evaluation
- **robots.txt & sitemap.xml Parsing**: Web crawler policy and site structure analysis

#### Graceful Cancellation

- Press `Ctrl-C` to gracefully cancel scans
- Partial results saved with integrity hashes
- Audit trail updated for completed targets
- Press `Ctrl-C` twice to force quit

#### Documentation

**New Comprehensive Guides** (~15,000 lines):
- **[Plugin Development Guide](docs/developer-guide/plugin-development.md)** - Complete plugin API documentation
  - Plugin architecture and lifecycle
  - JSON definition format reference
  - Output format specifications
  - 3 example plugins (Bash, Python, Go)
  - Testing and debugging guide

- **[Advanced Features Guide](docs/user-guide/advanced-features.md)** - In-depth feature documentation
  - Retry mechanism usage and strategies
  - Progress display modes
  - Telemetry system and analysis
  - Secure results encryption with GPG
  - SHA-256 vs SHA-512 comparison
  - Custom DNS nameservers
  - Graceful cancellation workflow
  - TUI usage guide

- **[Command Reference](docs/reference/command-reference.md)** - Complete command documentation
  - All commands and subcommands
  - Comprehensive flag reference
  - Examples for every command
  - Exit codes and error handling
  - Flag precedence rules
  - Common flag combinations

- **[Troubleshooting Guide](docs/reference/troubleshooting.md)** - Problem-solving reference
  - Installation issues
  - Engagement management errors
  - Check command troubleshooting
  - Plugin debugging
  - GPG and signing problems
  - Performance optimization
  - Network connectivity issues
  - Comprehensive FAQ

**Updated Documentation:**
- README.md - Updated with all new features and plugin system
- docs/README.md - Updated documentation index with new guides
- Added Developer Guides category to documentation structure

### Changed

- **Engagement Command Updates**:
  - `seca engagement create` now uses `--id`, `--client`, and `--start-date` flags
  - Simplified engagement creation workflow
  - ROE acceptance integrated into engagement definition

- **Data Directory Structure**:
  - Added `plugins/` directory for plugin definitions
  - Added `telemetry/` directory for metrics storage
  - Telemetry stored as JSONL files per engagement

- **Report Generation**:
  - Enhanced statistics with colorized CLI output
  - Multiple export formats (table, JSON, CSV, markdown)
  - Improved error summaries and trending

### Improved

- **Performance**: Optimized concurrent request handling
- **Error Messages**: More descriptive error messages with troubleshooting hints
- **Compliance**: Enhanced compliance mode with stricter validation
- **Testing**: Expanded test coverage for new features
- **User Experience**: Interactive TUI for easier engagement management

### Fixed

- Improved error handling for network timeouts
- Better validation of engagement IDs
- Enhanced GPG error messages
- Fixed edge cases in retry logic

### Security

- GPG encryption for sensitive results
- SHA-512 support for maximum integrity assurance
- Plugin sandboxing considerations documented
- Secure defaults for all new features

---

## [1.0.0] - 2025-01-15

### Added

#### Core Features
- **Engagement Management**: Create, list, and manage security testing engagements
- **Scope Control**: Add and manage authorized targets for testing
- **HTTP Checks**: Safe, non-invasive HTTP/HTTPS connectivity checks
- **Compliance Mode**: Built-in compliance enforcement with automatic hashing
- **Audit Trail**: Immutable CSV audit logs with timestamps and operator attribution
- **Evidence Integrity**: SHA256 hash generation and verification for all evidence files
- **Raw Capture**: Optional HTTP response capture with PII safeguards and retention controls
- **TLS Monitoring**: Automatic TLS certificate expiry detection and warnings

#### Commands
- `seca engagement create` - Create new engagements
- `seca engagement list` - List all engagements
- `seca engagement add-scope` - Add authorized targets to engagement
- `seca check http` - Run safe HTTP checks on scoped targets
- `seca version` - Display version information

#### Compliance & Security
- ROE (Rules of Engagement) acknowledgment requirement
- Explicit `--roe-confirm` flag for all operations
- Operator attribution for accountability
- Automatic SHA256 hashing of all evidence files
- Retention policy enforcement for raw captures
- GPG signing support for evidence packages

#### Documentation
- Comprehensive README with examples
- COMPLIANCE.md for detailed compliance guidance
- OPERATOR_TRAINING.md for operator certification
- TESTING.md for test suite documentation
- INSTALL.md for installation instructions

#### Testing
- 29 unit tests with 89.3% coverage
- Integration test suite
- Makefile automation for testing
- CI/CD pipeline with GitHub Actions

#### Build & Release
- Multi-platform builds (Linux, macOS, Windows)
- Version injection at build time
- Automated release script with checksums
- GPG signature support

### Technical Details

#### Supported Platforms
- Linux (amd64, arm64)
- macOS (amd64, arm64 / Apple Silicon)
- Windows (amd64)

#### Dependencies
- Go 1.21+
- cobra (CLI framework)
- viper (configuration management)
- zap (structured logging)
- golang.org/x/time/rate (rate limiting)

### Known Limitations

- No active vulnerability scanning (by design - safety first)
- No exploitation capabilities (authorized checks only)
- HTTP/HTTPS only (no other protocols)
- CSV audit format (JSON support planned for v1.1.0)

### Security

- All operations require explicit authorization
- Scope boundaries strictly enforced
- Rate limiting to prevent service disruption
- No credentials or PII stored in code

---

## [1.1.0] - 2025-11-11

### Added

#### New Features
- **DNS Checker**: Comprehensive DNS resolution checks for authorized targets
  - A record lookups (IPv4 addresses)
  - AAAA record lookups (IPv6 addresses)
  - CNAME record resolution
  - MX record checks (mail servers)
  - NS record lookups (nameservers)
  - TXT record queries (SPF, DKIM, etc.)
  - PTR record lookups (reverse DNS)
  - Custom nameserver support via `--nameservers` flag
  - Configurable DNS timeout via `--dns-timeout` flag
- **Report Generation**: Generate comprehensive reports from check results
  - JSON format reports
  - Markdown format reports
  - HTML format reports
  - Statistics and summary reports
- **XDG Base Directory Support**: Platform-specific data directories
  - Linux: `~/.local/share/seca-cli/`
  - macOS: `~/Library/Application Support/seca-cli/`
  - Windows: `%LOCALAPPDATA%\seca-cli\`
  - Automatic migration from old `./` location

#### Commands
- `seca check dns` - Run safe DNS resolution checks on scoped targets
- `seca report generate` - Generate reports in various formats
- `seca report stats` - Show statistics for check results
- `seca info` - Display configuration and data directory paths

#### Improvements
- Comprehensive test suite with 93 tests (19 DNS + 17 HTTP + 57 cmd tests)
- Refactored test structure for better organization
- Integration tests updated for XDG directory support
- Improved error messages and debugging output
- Fixed linting issues

#### Bug Fixes
- Fixed results directory detection in integration tests
- Removed unused deprecated constants
- Fixed indentation in test scripts

### Changed
- Test structure reorganized: HTTP tests moved to `internal/checker/http_test.go`
- Data storage now uses platform-specific directories by default
- Improved integration test reliability with multi-location directory search

### Technical Details

#### New Dependencies
- No new external dependencies added

#### Test Coverage
- Total tests: 93 (up from 29)
- DNS checker: 19 tests with comprehensive coverage
- HTTP checker: 17 tests (refactored from cmd package)
- Command tests: 57 tests

---

## [1.2.0] - 2025-11-12

### Added

#### TLS Configuration Audit
- **NIST SP 800-52r2 Compliance**: TLS analyzer now evaluates negotiated versions, suites, and certificates against NIST controls alongside OWASP ASVS §9 and PCI DSS 4.1.
- **Cipher Suite Intelligence**: Explicit allow/deny lists for AES-GCM suites plus forward-secrecy checks ensure only compliant ECDHE/DHE handshakes pass.
- **Certificate Insights**: Results now capture chain depth, verified intermediates, subject list, signature algorithms, and key size to make investigations actionable.

#### Reporting Enhancements
- Detailed compliance sections per standard with pass/fail IDs for cross-referencing audit evidence.
- New recommendations surface remediation guidance (e.g., rotate expiring certs, remove ChaCha-only stacks, upgrade to TLS 1.3).

### Changed
- Overall compliance status now fails when any NIST SP 800-52r2 requirement is violated (e.g., TLS 1.0, ChaCha20-Poly1305-only stacks on TLS 1.3).
- Certificate validation consumes verified chains from the TLS handshake, ensuring accurate trust-path analysis instead of assuming success.
- RSA/ECC key size validation leverages precise bit lengths extracted from `crypto/ecdsa`, `ed25519`, and RSA keys.

### Fixed
- Hardened `go vet`/`golangci-lint` workflows by vendoring a project-local lint binary and cache path, ensuring linting runs cleanly without host-level permissions.

---

## [1.2.1] - 2025-11-12

### Changed

#### Code Quality Improvements
- **Eliminated Global Variables**: Refactored command-line interface to use dependency injection pattern
  - Introduced `AppContext` struct to manage application-wide dependencies (logger, operator, results directory)
  - Removed global variables from [cmd/root.go](cmd/root.go) improving testability and maintainability
  - Updated all command files ([cmd/check.go](cmd/check.go), [cmd/report.go](cmd/report.go), [cmd/info.go](cmd/info.go)) to use context-based approach

- **Reduced Code Duplication**: Significant refactoring of check command handlers
  - Created `runCheckCommand()` helper function implementing Template Method pattern
  - Introduced `checkConfig` struct using Strategy Pattern for command configuration
  - HTTP command handler reduced from 105 to 43 lines (59% reduction)
  - DNS command handler reduced from 125 to 63 lines (50% reduction)
  - Overall [cmd/check.go](cmd/check.go) reduced from 577 to 459 lines (20.4% reduction)
  - Extracted common helper functions: `validateCheckParams`, `loadEngagementByID`, `writeResultsAndHash`, `signHashFiles`, `printComplianceSummary`

- **Enhanced Command Consistency**: Added missing flags to DNS command for feature parity with HTTP command
  - Added `--id`, `--roe-confirm`, `--compliance-mode`, `--auto-sign`, `--gpg-key` flags
  - Standardized flag handling across all check commands

#### Test Infrastructure
- Updated test files to work with new `AppContext` architecture
  - Created `setupTestAppContext()` helper in [cmd/info_test.go](cmd/info_test.go)
  - Updated 11 test functions in info_test.go
  - Updated all audit trail tests in [cmd/audit_test.go](cmd/audit_test.go)
  - Maintained 100% test pass rate throughout refactoring

### Technical Details

#### Architecture Improvements
- **Pattern Implementation**:
  - Dependency Injection: `AppContext` replaces global state
  - Strategy Pattern: `CreateChecker` and `CreateAuditFn` function pointers
  - Factory Pattern: Dynamic checker and audit function creation
  - Template Method: `runCheckCommand()` defines common execution flow

#### Files Modified
- [cmd/root.go](cmd/root.go): Added `AppContext` struct and helper functions
- [cmd/check.go](cmd/check.go): Major refactoring with `runCheckCommand()` helper
- [cmd/report.go](cmd/report.go): Updated to use `appCtx.ResultsDir`
- [cmd/info.go](cmd/info.go): Updated to use `appCtx.Operator` and `appCtx.ResultsDir`
- [cmd/audit.go](cmd/audit.go): Updated function signatures to accept `resultsDir` parameter
- [cmd/audit_test.go](cmd/audit_test.go): Updated all tests for new signatures
- [cmd/info_test.go](cmd/info_test.go): Added test context setup helper

#### Benefits
- **Improved Testability**: Explicit dependency injection makes unit testing easier
- **Better Maintainability**: Reduced duplication means fewer places to update
- **Enhanced Readability**: Command handlers now focus on configuration, not implementation
- **Consistent Behavior**: Shared helper functions ensure uniform error handling and output

---

## [Unreleased]

### Planned for v1.3.0

- JSON audit log format (in addition to CSV)
- Dashboard for visualizing results
- Enhanced TLS analysis
- Custom check scripts
- Database storage option
- API server mode

### Under Consideration

- SSL/TLS certificate validation
- Port scanning (safe, authorized only)
- Webhook notifications
- Multi-user support
- Cloud storage integration

---

## Version History

### v1.2.0 (2025-11-12)
- TLS configuration audit with NIST SP 800-52r2 controls
- Expanded certificate metadata (chain depth, signatures, key sizes)
- Strong/weak cipher tracking with AES-GCM preference and PFS enforcement
- Integrated lint workflow improvements

### v1.1.0 (2025-11-11)
- DNS checker feature with comprehensive record lookups
- Report generation (JSON, Markdown, HTML)
- XDG Base Directory support
- 93 comprehensive tests
- Refactored test structure

### v1.0.0 (2025-01-15)
- Initial release
- Core engagement management
- HTTP checks with compliance mode
- Complete documentation
- Operator training guide

---

## Upgrade Guide

### From Development to v1.0.0

If you were using a development version:

1. Backup your data:
   ```bash
   cp engagements.json engagements.json.backup
   cp -r results/ results.backup/
   ```

2. Install v1.0.0:
   ```bash
   wget https://github.com/khanhnv2901/seca-cli/releases/download/v1.0.0/seca-1.0.0-linux-amd64
   chmod +x seca-1.0.0-linux-amd64
   sudo mv seca-1.0.0-linux-amd64 /usr/local/bin/seca
   ```

3. Verify:
   ```bash
   seca version
   ```

4. Test with existing data:
   ```bash
   seca engagement list
   ```

---

## Breaking Changes

### v1.0.0
- None (initial release)

---

## Deprecation Notices

### v1.0.0
- None (initial release)

---

## Migration Path

No migrations required for v1.0.0 (initial release).

---

## Support

For questions or issues:
- GitHub Issues: https://github.com/khanhnv2901/seca-cli/issues
- Email: khanhnv2901@gmail.com

---

*This changelog is maintained by the SECA-CLI development team.*
