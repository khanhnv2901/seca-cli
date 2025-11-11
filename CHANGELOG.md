# Changelog

All notable changes to SECA-CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

## [Unreleased]

### Planned for v1.2.0

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
