# SECA-CLI

**Secure Engagement & Compliance Auditing CLI**

A professional command-line tool for managing authorized security testing engagements with built-in compliance, evidence integrity, and audit trail capabilities.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/dl/)

## Features

- **Engagement Management** - Create and track security testing engagements with ROE (Rules of Engagement) acknowledgment
- **Scope Control** - Define and manage authorized targets for testing
- **Safe HTTP Checks** - Perform non-invasive HTTP/HTTPS checks with rate limiting and concurrency control
- **Compliance Mode** - Built-in compliance enforcement with automatic hash signing and retention policies
- **Audit Trail** - Immutable CSV audit logs with timestamps and operator attribution
- **Evidence Integrity** - SHA256 hash generation and verification for all evidence files
- **Raw Capture** - Optional HTTP response capture with PII safeguards and retention controls
- **TLS Monitoring** - Automatic TLS certificate expiry detection and warnings

## Quick Start

### Installation

#### Build from Source

```bash
# Clone the repository
git clone https://github.com/khanhnv2901/seca-cli.git
cd seca-cli

# Build
make build

# Install (optional)
sudo make install
```

#### Manual Build

```bash
go build -o seca main.go
```

### Basic Usage

```bash
# Create an engagement
./seca engagement create \
  --name "Client XYZ Pentest" \
  --owner "client@example.com" \
  --roe "Written authorization received on 2025-01-15" \
  --roe-agree

# Add scope (authorized targets)
./seca engagement add-scope \
  --id <engagement-id> \
  --scope https://example.com,https://api.example.com

# List engagements
./seca engagement list

# Run safe HTTP checks
./seca check http \
  --id <engagement-id> \
  --roe-confirm \
  --concurrency 4 \
  --rate 3 \
  --timeout 15
```

## Engagement Workflow

### 1. Create Engagement

Every security testing activity must be associated with an engagement:

```bash
./seca engagement create \
  --name "Q1 2025 Security Assessment" \
  --owner "security-team@company.com" \
  --roe "Authorization letter signed 2025-01-10" \
  --roe-agree
```

This returns an engagement ID (e.g., `1762627948156627663`).

### 2. Define Scope

Add authorized targets to the engagement:

```bash
./seca engagement add-scope \
  --id 1762627948156627663 \
  --scope https://example.com,https://api.example.com,https://staging.example.com
```

### 3. Run Authorized Checks

Execute safe, non-invasive HTTP checks:

```bash
./seca --operator "john.doe" check http \
  --id 1762627948156627663 \
  --roe-confirm \
  --concurrency 4 \
  --rate 3
```

### 4. Compliance Mode (Recommended)

For regulated environments, use compliance mode:

```bash
./seca --operator "john.doe" check http \
  --id 1762627948156627663 \
  --roe-confirm \
  --compliance-mode \
  --audit-append-raw \
  --retention-days 90 \
  --concurrency 4 \
  --rate 3
```

## Command Reference

### Global Flags

```
--operator, -o    Operator name (default: $USER)
--config          Config file (default: $HOME/.seca-cli.yaml)
```

### Engagement Commands

```bash
# Create engagement
seca engagement create --name <name> --owner <email> --roe <text> --roe-agree

# List engagements
seca engagement list

# Add scope to engagement
seca engagement add-scope --id <id> --scope <url1>,<url2>,...
```

### Check Commands

```bash
# Run HTTP checks
seca check http --id <engagement-id> --roe-confirm [options]

Options:
  --concurrency, -c      Max concurrent requests (default: 1)
  --rate, -r             Requests per second (default: 1)
  --timeout, -t          Request timeout in seconds (default: 10)
  --compliance-mode      Enable compliance enforcement
  --audit-append-raw     Save raw HTTP responses (use with caution)
  --retention-days N     Retention period for raw captures
```

## Evidence & Results

All evidence is stored in `results/<engagement-id>/`:

```
results/1762627948156627663/
├── audit.csv              # CSV audit log
├── audit.csv.sha256       # SHA256 hash
├── results.json           # JSON results with metadata
├── results.json.sha256    # SHA256 hash
└── raw_*.txt              # Raw captures (if --audit-append-raw used)
```

### Audit Log Format

The `audit.csv` contains timestamped records:

```csv
timestamp,engagement_id,operator,command,target,status,http_status,tls_expiry,notes,error,duration_seconds
2025-11-09T10:30:15Z,1762627948156627663,john.doe,check http,https://example.com,ok,200,2026-01-15T00:00:00Z,robots.txt found,,0.234
```

### Results JSON Format

```json
{
  "metadata": {
    "operator": "john.doe",
    "engagement_id": "1762627948156627663",
    "engagement_name": "Client XYZ Pentest",
    "owner": "client@example.com",
    "started_at": "2025-11-09T10:30:00Z",
    "completed_at": "2025-11-09T10:35:00Z",
    "audit_sha256": "abc123...",
    "results_sha256": "def456...",
    "total_targets": 3
  },
  "results": [
    {
      "target": "https://example.com",
      "checked_at": "2025-11-09T10:30:15Z",
      "status": "ok",
      "http_status": 200,
      "server_header": "nginx/1.21.0",
      "tls_expiry": "2026-01-15T00:00:00Z",
      "notes": "robots.txt found"
    }
  ]
}
```

## Compliance & Verification

### Verify Evidence Integrity

Using the Makefile:

```bash
# Verify single engagement
make verify ENGAGEMENT_ID=1762627948156627663

# Verify all engagements
make verify-all
```

Manual verification:

```bash
cd results/1762627948156627663/
sha256sum -c audit.csv.sha256
sha256sum -c results.json.sha256
```

### Sign Evidence with GPG

```bash
# Sign single engagement
make sign ENGAGEMENT_ID=1762627948156627663

# Sign all engagements
make sign-all
```

### Retention Management

Delete raw captures after retention period:

```bash
# Purge captures older than 90 days
make purge-raw ENGAGEMENT_ID=1762627948156627663 RETENTION_DAYS=90

# Purge for all engagements
make purge-raw-all RETENTION_DAYS=90
```

### Create Evidence Package

Package evidence for delivery:

```bash
make package ENGAGEMENT_ID=1762627948156627663
```

This creates:
- `evidence-<id>.tar.gz` - Compressed evidence archive
- `evidence-<id>.tar.gz.asc` - GPG signature
- `evidence-<id>.tar.gz.sha256` - SHA256 hash

## Configuration

Create `~/.seca-cli.yaml`:

```yaml
results_dir: /path/to/results
```

Or use the `--config` flag:

```bash
seca --config /path/to/config.yaml engagement list
```

## Testing

SECA-CLI includes comprehensive unit and integration tests.

### Run Tests

```bash
# Run all unit tests
make test

# Run with coverage
make test-coverage

# Run integration tests
make test-integration

# Run all tests
make test-all
```

See [TESTING.md](TESTING.md) for detailed testing documentation.

## Makefile Targets

The included Makefile provides convenient automation:

**Build & Install:**
```bash
make build                   # Build the binary
make install                 # Install to /usr/local/bin
```

**Testing:**
```bash
make test                    # Run unit tests
make test-coverage           # Run tests with coverage
make test-integration        # Run integration tests
make test-all                # Run all tests
make test-clean              # Clean test artifacts
```

**Compliance:**
```bash
make verify ENGAGEMENT_ID=<id>   # Verify hashes
make sign ENGAGEMENT_ID=<id>     # Sign with GPG
make purge-raw ENGAGEMENT_ID=<id> RETENTION_DAYS=90  # Delete old captures
make package ENGAGEMENT_ID=<id>  # Create evidence package
```

**Utilities:**
```bash
make help                    # Show all available targets
make list-engagements        # List all engagements
make show-stats ENGAGEMENT_ID=<id>  # Show statistics
make clean                   # Remove evidence packages
```

## Safety & Authorization

### Important Notes

- **Always obtain written authorization** before testing any systems
- Use `--roe-confirm` flag as explicit acknowledgment of authorization
- Respect rate limits and concurrency settings to avoid service disruption
- Only test systems within the defined engagement scope
- SECA-CLI performs **safe, non-invasive checks only** (no exploitation)

### What SECA-CLI Does

✅ HTTP HEAD/GET requests to check availability
✅ TLS certificate expiry checks
✅ robots.txt retrieval
✅ Server header inspection
✅ Rate-limited, controlled testing

### What SECA-CLI Does NOT Do

❌ Vulnerability scanning
❌ Exploitation attempts
❌ Brute force attacks
❌ Service disruption
❌ Unauthorized access

## Compliance Standards

SECA-CLI is designed to support compliance with:

- **GDPR** - Data minimization, retention controls, PII safeguards
- **PCI-DSS** - Audit logging, access controls, evidence integrity
- **SOC 2** - Chain of custody, cryptographic hashing, operator attribution
- **ISO 27001** - Documentation, access controls, change tracking

See [COMPLIANCE.md](COMPLIANCE.md) for detailed compliance guidance.

## Examples

### Example 1: Basic Engagement

```bash
# Create engagement
./seca engagement create \
  --name "Website Health Check" \
  --owner "ops@company.com" \
  --roe "Approved by CTO on 2025-01-15" \
  --roe-agree

# Add targets
./seca engagement add-scope \
  --id 1762627948156627663 \
  --scope https://example.com

# Run checks
./seca check http \
  --id 1762627948156627663 \
  --roe-confirm \
  --concurrency 1 \
  --rate 1
```

### Example 2: Compliance Mode with Raw Capture

```bash
# Run with full compliance enforcement
./seca --operator "alice.security" check http \
  --id 1762627948156627663 \
  --roe-confirm \
  --compliance-mode \
  --audit-append-raw \
  --retention-days 90 \
  --concurrency 4 \
  --rate 3

# Verify evidence
make verify ENGAGEMENT_ID=1762627948156627663

# Sign for delivery
make sign ENGAGEMENT_ID=1762627948156627663

# Package for client
make package ENGAGEMENT_ID=1762627948156627663
```

### Example 3: Large-Scale Testing

```bash
# Add multiple targets
./seca engagement add-scope \
  --id 1762627948156627663 \
  --scope https://app1.example.com,https://app2.example.com,https://app3.example.com

# Run with higher concurrency
./seca --operator "bob.pentester" check http \
  --id 1762627948156627663 \
  --roe-confirm \
  --compliance-mode \
  --concurrency 10 \
  --rate 5 \
  --timeout 30
```

## Project Structure

```
seca-cli/
├── cmd/                    # Command implementations
│   ├── audit.go           # Audit trail functions
│   ├── check.go           # HTTP check commands
│   ├── engagement.go      # Engagement management
│   ├── report.go          # Reporting (placeholder)
│   └── root.go            # Root command & config
├── main.go                # Application entry point
├── Makefile               # Automation tasks
├── README.md              # This file
├── COMPLIANCE.md          # Compliance documentation
├── .gitignore
├── go.mod
├── go.sum
├── engagements.json       # Engagement database
└── results/               # Evidence storage
    └── <engagement-id>/
        ├── audit.csv
        ├── audit.csv.sha256
        ├── results.json
        ├── results.json.sha256
        └── raw_*.txt
```

## Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [viper](https://github.com/spf13/viper) - Configuration management
- [zap](https://github.com/uber-go/zap) - Structured logging
- [rate](https://golang.org/x/time/rate) - Rate limiting

## Development

```bash
# Install dependencies
go mod download

# Run tests
make test

# Build
make build

# Run
./seca --help
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/khanhnv2901/seca-cli/issues)
- **Email**: khanhnv2901@gmail.com

## Disclaimer

This tool is designed for **authorized security testing only**. Users are responsible for obtaining proper authorization before testing any systems. Unauthorized access to computer systems is illegal. Always comply with applicable laws and regulations.

## Author

**Khanh Nguyen**
GitHub: [@khanhnv2901](https://github.com/khanhnv2901)

---

**Remember**: With great power comes great responsibility. Always test ethically and with proper authorization.
