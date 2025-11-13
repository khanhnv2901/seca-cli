# SECA-CLI

**Secure Engagement & Compliance Auditing CLI**

A professional command-line tool for managing authorized security testing engagements with built-in compliance, evidence integrity, and audit trail capabilities.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/dl/)

## Features

### Core Capabilities

- **Engagement Management** - Create, view, delete, and track security testing engagements with ROE (Rules of Engagement) acknowledgment
- **Interactive TUI** - Terminal UI for visual engagement browsing and management
- **Scope Control** - Define and manage authorized targets for testing
- **Safe HTTP Checks** - Perform non-invasive HTTP/HTTPS checks with rate limiting and concurrency control
- **DNS Resolution Checks** - Comprehensive DNS record analysis (A, AAAA, CNAME, MX, NS, TXT, PTR)
- **TLS/Crypto Compliance** - OWASP ASVS §9 and PCI DSS 4.1 validation for TLS versions, cipher suites, and certificates
- **Security Headers Analysis** - OWASP Secure Headers Project compliance checking with scoring and recommendations

### Advanced Security Analysis

- **Cookie Security** - Secure/HttpOnly flag detection for OWASP A1:2021 compliance
- **CORS Policy Inspection** - OWASP A5:2021 cross-origin policy validation
- **Third-Party Scripts** - Supply-chain risk inventory and detection
- **Cache Policy Analysis** - Performance and security cache header evaluation
- **robots.txt & sitemap.xml** - Web crawler policy and site structure parsing
- **In-Scope Link Discovery** - Optional crawler explores same-host links before running checks
  - Static HTML crawling for traditional websites
  - JavaScript-enabled crawling for SPAs (React, Vue, Angular) using headless Chrome
  - Auto-detection of JavaScript requirements

### Compliance & Evidence

- **Compliance Mode** - Built-in compliance enforcement with automatic hash signing and retention policies
- **Audit Trail** - Immutable CSV audit logs with timestamps and operator attribution
- **Evidence Integrity** - SHA-256 and SHA-512 hash generation for cryptographic verification
- **Secure Results** - GPG encryption for audit logs and results
- **Raw Capture** - Optional HTTP response capture with PII safeguards and retention controls
- **TLS Monitoring** - Automatic TLS certificate expiry detection and warnings

### Advanced Features

- **Retry Mechanism** - Automatic retry of failed targets (--retry N flag)
- **Progress Display** - Live progress bars for long-running scans
- **Telemetry & Metrics** - Success rate tracking with trend analysis and ASCII graphs
- **Plugin Architecture** - Extensible checker system with external plugin support
- **Graceful Cancellation** - Ctrl-C saves partial results with integrity verification
- **Custom DNS Servers** - Specify nameservers for internal or specialized DNS queries
- **Report Generation** - Markdown, HTML, PDF, and JSON reports with statistics

## Data Storage

SECA-CLI stores user data in OS-appropriate data directories following platform standards, ensuring proper permissions, multi-user support, and clean separation from the codebase.

### Data Locations

**Linux/Unix:**
```
~/.local/share/seca-cli/
├── engagements.json
└── results/
    └── <engagement-id>/
        ├── audit.csv
        ├── results.json
        └── raw_*.txt
```

**macOS:**
```
~/Library/Application Support/seca-cli/
├── engagements.json
└── results/
    └── <engagement-id>/
        ├── audit.csv
        ├── results.json
        └── raw_*.txt
```

**Windows:**
```
%LOCALAPPDATA%\seca-cli\
├── engagements.json
└── results\
    └── <engagement-id>\
        ├── audit.csv
        ├── results.json
        └── raw_*.txt
```

### Automatic Migration

When upgrading from versions prior to 0.2.0, SECA-CLI automatically migrates `engagements.json` from the project directory to the new location on first run. The old file is backed up as `engagements.json.backup`.

### Custom Data Directory

You can override the default data directory in `~/.seca-cli.yaml`:

```yaml
results_dir: /custom/path/to/results
```

This is useful for:
- Shared team directories
- Network storage
- Custom backup solutions

See [Data Migration Guide](docs/reference/data-migration.md) for detailed migration instructions.

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
seca engagement create --id <id> --client <name> --start-date <YYYY-MM-DD>

# List all engagements
seca engagement list

# View engagement details as JSON
seca engagement view --id <id>

# Delete engagement and all data
seca engagement delete --id <id>

# Add scope to engagement
seca engagement add-scope --id <id> <target1> <target2> ...

# Interactive TUI for engagement management
seca tui
```

### Check Commands

```bash
# Run HTTP/HTTPS checks
seca check http --id <engagement-id> --roe-confirm [options] <targets...>

# Run DNS checks
seca check dns --id <engagement-id> --roe-confirm [options] <targets...>

# Run custom plugin checks
seca check <plugin-name> --id <engagement-id> --roe-confirm <targets...>

Common Options:
  --concurrency, -c      Max concurrent requests (default: 1)
  --rate, -r             Requests per second (default: 1)
  --timeout, -t          Request timeout in seconds (default: 10)
  --retry N              Retry failed targets N times
  --progress             Display live progress bar
  --telemetry            Record telemetry metrics
  --hash sha512          Use SHA-512 instead of SHA-256
  --secure-results       Encrypt results with GPG
  --compliance-mode      Enable compliance enforcement
  --audit-append-raw     Save raw HTTP responses (use with caution)
  --retention-days N     Retention period for raw captures

Crawling Options:
  --crawl                Discover same-host links (auto-detects JavaScript/SPA sites)
  --crawl-depth N        Maximum link depth to follow (default: 2)
  --crawl-max-pages N    Maximum pages to discover per target (default: 50)
  --crawl-force-js       Force JavaScript crawler for all targets
  --crawl-js-wait N      Seconds to wait for JavaScript to render (default: 2)
```

### Report Commands

```bash
# Generate engagement report
seca report generate --id <id> [--format markdown|html|pdf|json]

# View engagement statistics
seca report stats --id <id> [--format table|json|csv|markdown]

# Display telemetry trends
seca report telemetry --id <id> [--format graph|json]
```

## Evidence & Results

All evidence is stored in the OS-specific data directory under `results/<engagement-id>/`:

**Linux/Unix:** `~/.local/share/seca-cli/results/<engagement-id>/`
**macOS:** `~/Library/Application Support/seca-cli/results/<engagement-id>/`
**Windows:** `%LOCALAPPDATA%\seca-cli\results\<engagement-id>\`

```
results/1762627948156627663/
├── audit.csv              # CSV audit log
├── audit.csv.sha256       # SHA256 hash
├── results.json           # JSON results with metadata
├── results.json.sha256    # SHA256 hash
└── raw_*.txt              # Raw captures (if --audit-append-raw used)
```

See the [Data Storage](#data-storage) section for custom directory configuration.

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

SECA-CLI uses OS-appropriate data directories by default. You can customize settings by creating `~/.seca-cli.yaml`:

```yaml
# Optional: Override default data directory
results_dir: /custom/path/to/results

# Example: Use shared team directory
# results_dir: /mnt/shared/seca-data

# Example: Use network storage
# results_dir: /mnt/nfs/security-team/seca
```

Or use the `--config` flag to specify a custom config file:

```bash
seca --config /path/to/config.yaml engagement list
```

**Default Data Directories** (when `results_dir` is not set):
- **Linux/Unix:** `~/.local/share/seca-cli/`
- **macOS:** `~/Library/Application Support/seca-cli/`
- **Windows:** `%LOCALAPPDATA%\seca-cli\`

For complete configuration options including command-line flags, environment variables, and examples, see [Configuration Guide](docs/user-guide/configuration.md).

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

See [Testing Guide](docs/technical/testing.md) for detailed testing documentation.

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
✅ TLS/crypto compliance validation (OWASP ASVS §9, PCI DSS 4.1)
✅ TLS certificate expiry and validation checks
✅ Security headers analysis (OWASP Secure Headers)
✅ Cipher suite strength analysis
✅ robots.txt retrieval
✅ Server header inspection
✅ Rate-limited, controlled testing

### What SECA-CLI Does NOT Do

❌ Vulnerability scanning
❌ Exploitation attempts
❌ Brute force attacks
❌ Service disruption
❌ Unauthorized access

## TLS/Crypto Compliance (OWASP ASVS §9, PCI DSS 4.1)

SECA-CLI automatically validates TLS/cryptography configuration against **OWASP ASVS Section 9 (Communications)** and **PCI DSS 4.1 (Strong Cryptography)** requirements for all HTTPS targets.

### Compliance Checks

**TLS Protocol Version (ASVS 9.1.3, PCI DSS 4.1)**
- ✅ TLS 1.3 (Recommended)
- ✅ TLS 1.2 with strong cipher suites
- ❌ TLS 1.1, TLS 1.0, SSL 3.0 (Non-compliant)

**Cipher Suite Strength (ASVS 9.1.2, PCI DSS 4.1)**
- ✅ AEAD cipher suites (AES-GCM, ChaCha20-Poly1305)
- ✅ Perfect Forward Secrecy (ECDHE/DHE key exchange)
- ❌ RC4, 3DES, CBC-mode ciphers (Weak)
- Minimum 112-bit effective key strength (PCI DSS)

**Certificate Validation (ASVS 9.2.1, PCI DSS 4.1)**
- Certificate expiry and validity period
- RSA keys ≥ 2048 bits, ECC keys ≥ 224 bits
- SHA-256 or stronger signature algorithms
- Self-signed certificate detection
- Certificate chain validation

### Compliance Result Format

```json
{
  "tls_compliance": {
    "compliant": true,
    "tls_version": "TLS 1.3",
    "cipher_suite": "TLS_AES_256_GCM_SHA384",
    "standards": {
      "owasp_asvs_v9": {
        "compliant": true,
        "level": "L1",
        "passed": ["9.1.2", "9.1.3", "9.2.1"],
        "failed": []
      },
      "pci_dss_4_1": {
        "compliant": true,
        "passed": ["4.1-TLS-Version", "4.1-Cipher", "4.1-Certificate-Valid"],
        "failed": []
      }
    },
    "certificate_info": {
      "subject": "CN=example.com",
      "issuer": "CN=Let's Encrypt",
      "days_until_expiry": 75,
      "signature_algorithm": "SHA256-RSA",
      "key_size": 2048
    },
    "issues": [],
    "recommendations": [
      "Consider upgrading to TLS 1.3 for improved security"
    ]
  }
}
```

### Non-Compliance Examples

**Critical Issues:**
- TLS 1.0/1.1 usage → Upgrade to TLS 1.2/1.3 immediately
- Weak cipher suites (RC4, 3DES) → Use AES-GCM or ChaCha20-Poly1305
- Expired certificates → Renew certificate immediately
- Weak RSA keys (<2048 bits) → Generate new certificate with 2048+ bit keys

**High Severity:**
- SHA-1 certificate signatures → Re-issue with SHA-256+
- Missing Perfect Forward Secrecy → Enable ECDHE/DHE key exchange

See [OWASP ASVS v5](https://owasp.org/www-project-application-security-verification-standard/) and [PCI DSS v4.0](https://www.pcisecuritystandards.org/) for complete requirements.

## Security Headers Analysis

SECA-CLI automatically analyzes HTTP security headers based on the **OWASP Secure Headers Project** best practices. Each target receives a security score (0-100) and letter grade (A-F).

### Analyzed Headers

**High Severity Headers:**
- **Strict-Transport-Security (HSTS)** - Forces HTTPS connections (20 points)
- **Content-Security-Policy (CSP)** - Mitigates XSS and injection attacks (20 points)
- **X-Frame-Options** - Prevents clickjacking attacks (15 points)
- **X-Content-Type-Options** - Blocks MIME type sniffing (15 points)

**Medium Severity Headers:**
- **Referrer-Policy** - Controls referrer information leakage (10 points)
- **Permissions-Policy** - Restricts browser features (10 points)
- **Cross-Origin-Opener-Policy (COOP)** - Isolates browsing contexts (5 points)
- **Cross-Origin-Embedder-Policy (COEP)** - Requires explicit resource permissions (5 points)

### Security Headers Output

Results include:
- **Score & Grade**: Overall security posture (e.g., 85/100, Grade B)
- **Present Headers**: Validation and scoring of each header
- **Missing Headers**: Critical security headers not implemented
- **Warnings**: Deprecated headers or information disclosure issues
- **Recommendations**: Specific remediation guidance

### Example Security Headers Result

```json
{
  "security_headers": {
    "score": 85,
    "grade": "B",
    "max_score": 100,
    "headers": {
      "Strict-Transport-Security": {
        "present": true,
        "value": "max-age=31536000; includeSubDomains",
        "severity": "high",
        "score": 18,
        "max_score": 20,
        "issues": ["Missing 'preload' directive"],
        "recommendation": "Add 'preload' for HSTS preload list"
      }
    },
    "missing": ["Content-Security-Policy", "Permissions-Policy"],
    "warnings": ["Server header exposes Apache/2.4.41"]
  }
}
```

### Cookie & Session Flag Analysis

SECA-CLI inspects `Set-Cookie` headers for missing `Secure` or `HttpOnly` attributes in accordance with **OWASP ASVS §3.4**. HTTP reports highlight every cookie that lacks these protections so teams can quickly remediate insecure session handling. CORS headers are validated so overly permissive origins (`*`) or missing `Access-Control-Allow-Origin` headers are flagged (OWASP Top 10 A5:2021), third-party script references (CDNs, analytics, etc.) are inventoried to improve supply-chain insight, and `robots.txt`/`sitemap.xml` files are parsed to spotlight exposed administrative paths. Response-time measurements and cache directives (Cache-Control/Expires) provide additional web performance/compliance evidence.

### Best Practices

1. **Aim for Grade A** (90+ points) - Implement all critical headers
2. **Remove information disclosure** - Obfuscate/remove Server and X-Powered-By headers
3. **Avoid deprecated headers** - Remove X-XSS-Protection, Expect-CT, Public-Key-Pins
4. **Use strict policies** - Avoid 'unsafe-inline', 'unsafe-eval' in CSP
5. **Enable HSTS preloading** - Use `includeSubDomains` and `preload` directives

See [OWASP Secure Headers Project](https://owasp.org/www-project-secure-headers/) for detailed guidance.

## Compliance Standards

SECA-CLI is designed to support compliance with:

- **GDPR** - Data minimization, retention controls, PII safeguards
- **PCI-DSS** - Audit logging, access controls, evidence integrity
- **SOC 2** - Chain of custody, cryptographic hashing, operator attribution
- **ISO 27001** - Documentation, access controls, change tracking

See [Compliance Guide](docs/operator-guide/compliance.md) for detailed compliance guidance.

## Vulnerability Reporting

`seca report generate --format html` produces a comprehensive vulnerability dashboard inside the primary engagement report. It includes:

- **Security Findings by Severity** - Critical, High, Medium, Low categorization
- **Detailed Descriptions** - Comprehensive explanations of each vulnerability
- **CVSS Scores** - Industry-standard vulnerability scoring (CVSS 3.1)
- **Remediation Steps** - Step-by-step instructions to fix issues
- **Code Examples** - Implementation examples for security headers and configurations
- **Testing Strategies** - Guidelines for testing security implementations
- **Affected URLs** - Complete list of pages where issues were detected

### Generating Vulnerability Reports

```bash
# Run security scan with crawling
./seca-cli check http --id <engagement-id> --roe-confirm --crawl

# Generate HTML report (includes vulnerability findings)
./seca-cli report generate --id <engagement-id> --format html

# Report is saved as:
# ~/.local/share/seca-cli/results/<engagement-id>/report.html
```

### Report Features

The vulnerability report includes:

1. **Scan Summary** - Date, duration, URLs scanned, vulnerability counts
2. **Security Findings Table** - Interactive table with all findings
3. **Expandable Details** - Click any finding to see full details
4. **CVSS Scoring** - Base score, severity, and attack vector
5. **Platform-Specific Fixes** - Implementation examples for Apache, Nginx, Express.js, etc.
6. **Compliance References** - OWASP, PCI DSS, and NIST guidelines

### Example Findings

Common vulnerabilities detected:
- Missing Content Security Policy (CSP)
- Missing HTTP Strict Transport Security (HSTS)
- Missing X-Frame-Options header
- Insecure cookie configurations
- Overly permissive CORS policies
- Weak TLS/SSL configurations

Each finding includes detailed remediation steps with code examples.

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

### Example 3: Crawling with Auto-Detection

The `--crawl` flag automatically detects JavaScript-based SPAs (React, Vue, Angular) and uses the appropriate crawler:

```bash
# Simple - Just add --crawl flag (auto-detects JS/SPA sites)
./seca check http \
  --id 1762627948156627663 \
  --roe-confirm \
  --crawl \
  --progress

# Customize depth and pages
./seca check http \
  --id 1762627948156627663 \
  --roe-confirm \
  --crawl \
  --crawl-depth 3 \
  --crawl-max-pages 100

# Force JavaScript crawler for known SPAs (skips detection)
./seca check http \
  --id 1762627948156627663 \
  --roe-confirm \
  --crawl \
  --crawl-force-js \
  --crawl-js-wait 3
```

**How it works:**
- Traditional websites → Uses fast static HTML crawler
- JavaScript SPAs (React/Vue/Angular) → Automatically uses headless Chrome
- No need to specify which type - it's auto-detected!

**Note:** JavaScript crawling requires Google Chrome or Chromium installed on the system.

### Example 4: Large-Scale Testing

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

## Plugin System

SECA-CLI supports an extensible plugin architecture for custom security checkers. Plugins are external programs that integrate seamlessly with SECA-CLI's engagement management and audit features.

### Plugin Location

Place plugin definition files (`.json`) in:
- **Linux/Unix:** `~/.local/share/seca-cli/plugins/`
- **macOS:** `~/Library/Application Support/seca-cli/plugins/`
- **Windows:** `%LOCALAPPDATA%\seca-cli\plugins\`

### Plugin Definition Format

Create a JSON file defining your custom checker:

```json
{
  "name": "my-checker",
  "description": "Custom security checker",
  "command": "/path/to/checker-script",
  "args": ["--verbose"],
  "env": {
    "CUSTOM_VAR": "value"
  },
  "timeout": 30,
  "api_version": 1
}
```

### Plugin Output Format

Plugins must output JSON to stdout:

```json
{
  "status": "success",
  "http_status": 200,
  "notes": "Check passed - all security headers present",
  "error": ""
}
```

### Using Plugins

```bash
# Plugin automatically creates new check command
seca check my-checker --id eng123 --roe-confirm example.com

# Works with all standard flags
seca check my-checker --id eng123 --roe-confirm \
  --concurrency 10 \
  --progress \
  --telemetry \
  example.com
```

For detailed plugin development instructions, see [Plugin Development Guide](docs/developer-guide/plugin-development.md).

## Project Structure

```
seca-cli/
├── cmd/                          # Command implementations
│   ├── audit.go                 # Audit trail functions
│   ├── check.go                 # HTTP check commands
│   ├── engagement.go            # Engagement management
│   ├── paths.go                 # Data directory management
│   ├── report.go                # Report generation (JSON/MD/HTML)
│   ├── root.go                  # Root command & config
│   ├── templates/
│   │   └── report.html          # HTML report template
│   └── *_test.go                # Comprehensive test suite
├── main.go                      # Application entry point
├── Makefile                     # Automation tasks
├── README.md                    # This file
├── COMPLIANCE.md                # Compliance documentation
├── DATA_DIRECTORY_MIGRATION.md  # Migration guide
├── TESTING.md                   # Testing documentation
├── .gitignore
├── go.mod
└── go.sum
```

**User Data** (OS-specific locations):
```
~/.local/share/seca-cli/    (Linux/Unix)
~/Library/Application Support/seca-cli/    (macOS)
%LOCALAPPDATA%\seca-cli\    (Windows)
├── engagements.json        # Engagement database
└── results/                # Evidence storage
    └── <engagement-id>/
        ├── audit.csv
        ├── audit.csv.sha256
        ├── results.json
        ├── results.json.sha256
        └── raw_*.txt
```

> **Note:** User data is now stored in OS-appropriate data directories instead of the project directory. See [Data Storage](#data-storage) for details.

## Dependencies

### Go Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [viper](https://github.com/spf13/viper) - Configuration management
- [zap](https://github.com/uber-go/zap) - Structured logging
- [rate](https://golang.org/x/time/rate) - Rate limiting
- [chromedp](https://github.com/chromedp/chromedp) - Headless Chrome automation for JavaScript crawling

### System Dependencies

- **Google Chrome or Chromium** (optional) - Required for JavaScript-enabled crawling of SPAs
  - Install on Debian/Ubuntu: `sudo apt install chromium-browser`
  - Install on Fedora: `sudo dnf install chromium`
  - Install on macOS: `brew install chromium`
  - Install on Arch: `sudo pacman -S chromium`

## Development

```bash
# Install dependencies
go mod download

# Run tests
make test

# Build (development version)
make build

# Build with specific version
VERSION=1.2.0 make build

# Run
./seca --help
```

For detailed information about version management and build options, see [Version Management Guide](docs/technical/version-management.md).

## Documentation

Comprehensive documentation is available in the `docs/` directory:

### User Guides
- [Installation Guide](docs/user-guide/installation.md) - Complete installation instructions for all platforms
- [Configuration Guide](docs/user-guide/configuration.md) - Configuration reference and examples
- [Advanced Features Guide](docs/user-guide/advanced-features.md) - Retry mechanism, telemetry, GPG encryption, and more

### Operator Guides
- [Operator Training](docs/operator-guide/operator-training.md) - Training materials and certification
- [Compliance Guide](docs/operator-guide/compliance.md) - Compliance requirements and best practices

### Technical Documentation
- [Deployment Guide](docs/technical/deployment.md) - Production deployment
- [Testing Guide](docs/technical/testing.md) - Testing and QA procedures
- [Version Management](docs/technical/version-management.md) - Build versioning

### Developer Guides
- [Plugin Development Guide](docs/developer-guide/plugin-development.md) - Create custom security checkers

### Reference Documentation
- [Command Reference](docs/reference/command-reference.md) - Complete command and flag reference
- [Troubleshooting Guide](docs/reference/troubleshooting.md) - Common issues and solutions
- [Data Migration Guide](docs/reference/data-migration.md) - Data directory migration
- [Template Approaches](docs/reference/template-approaches.md) - Report template implementation

View the [Documentation Index](docs/README.md) for organized access to all guides.

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
