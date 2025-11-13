# Command Reference

Complete reference for all SECA-CLI commands, subcommands, and flags.

## Table of Contents

- [Global Options](#global-options)
- [Main Commands](#main-commands)
  - [seca engagement](#seca-engagement)
  - [seca check](#seca-check)
  - [seca report](#seca-report)
  - [seca tui](#seca-tui)
  - [seca info](#seca-info)
  - [seca version](#seca-version)
- [Engagement Management](#engagement-management)
- [Check Commands](#check-commands)
- [Report Commands](#report-commands)
- [Configuration](#configuration)
- [Exit Codes](#exit-codes)

---

## Global Options

Available for all commands:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.seca-cli.yaml` | Path to configuration file |
| `--operator` | string | `$USER` | Operator name for audit attribution |
| `-h, --help` | - | - | Show help for any command |

### Examples

```bash
# Use custom config file
seca --config /etc/seca/config.yaml engagement list

# Override operator name
seca --operator alice@security.com check http --id eng123 example.com
```

---

## Main Commands

### seca engagement

Manage security testing engagements.

```bash
seca engagement [subcommand] [flags]
```

**Subcommands:**
- `create` - Create a new engagement
- `list` - List all engagements
- `view` - View engagement details
- `delete` - Delete an engagement
- `add-scope` - Add targets to engagement scope

**See:** [Engagement Management](#engagement-management)

---

### seca check

Run authorized security checks against scoped targets.

```bash
seca check [type] [flags] [targets...]
```

**Check Types:**
- `http` - HTTP/HTTPS and TLS checks
- `dns` - DNS resolution checks
- Custom plugins - See [Plugin Development Guide](../developer-guide/plugin-development.md)

**Common Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--id` | string | **required** | Engagement ID |
| `--roe-confirm` | bool | false | Confirm Rules of Engagement (required) |
| `-c, --concurrency` | int | 1 | Max concurrent requests |
| `-r, --rate` | int | 1 | Requests per second (global rate limit) |
| `-t, --timeout` | int | 10 | Request timeout in seconds |
| `--progress` | bool | false | Display live progress bar |
| `--telemetry` | bool | false | Record telemetry metrics |
| `--secure-results` | bool | false | Encrypt results with GPG |
| `--hash` | string | `sha256` | Hash algorithm (`sha256` or `sha512`) |
| `--compliance-mode` | bool | false | Enable compliance enforcement |
| `--auto-sign` | bool | false | Auto-sign with GPG |
| `--gpg-key` | string | - | GPG key ID for signing |

**See:** [Check Commands](#check-commands)

---

### seca report

Generate reports and analytics from engagement data.

```bash
seca report [subcommand] [flags]
```

**Subcommands:**
- `generate` - Generate engagement report
- `stats` - Show engagement statistics
- `telemetry` - Display telemetry trends

**See:** [Report Commands](#report-commands)

---

### seca tui

Launch interactive Terminal UI for engagement management.

```bash
seca tui
```

**Interactive Features:**
- Browse all engagements
- Create new engagements
- View engagement details
- Delete engagements
- Add scope interactively

**Navigation:**
- `↑/↓` - Navigate list
- `Enter` - View details
- `c` - Create engagement
- `d` - Delete engagement
- `a` - Add scope
- `q` - Quit
- `/` - Search

**See:** [Advanced Features Guide](../user-guide/advanced-features.md#interactive-tui-mode)

---

### seca info

Display system information and data directory paths.

```bash
seca info
```

**Output:**
```
SECA-CLI System Information

Version: v1.5.0
Data Directory: /home/user/.local/share/seca-cli
Config File: /home/user/.seca-cli.yaml
Operator: alice@security.com

Directories:
  Engagements: /home/user/.local/share/seca-cli/engagements
  Results: /home/user/.local/share/seca-cli/results
  Telemetry: /home/user/.local/share/seca-cli/telemetry
  Plugins: /home/user/.local/share/seca-cli/plugins
```

---

### seca version

Print version information.

```bash
seca version
```

**Output:**
```
seca-cli version v1.5.0
Build: 2025-01-15T10:30:00Z
Commit: a3b5c8d9
```

---

## Engagement Management

### seca engagement create

Create a new security testing engagement.

```bash
seca engagement create --id <id> --client <name> --start-date <date> [flags]
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--id` | string | Unique engagement identifier |
| `--client` | string | Client organization name |
| `--start-date` | string | Start date (YYYY-MM-DD) |

**Optional Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--end-date` | string | - | End date (YYYY-MM-DD) |
| `--description` | string | - | Engagement description |
| `--scope` | []string | - | Initial scope (URLs/hosts, comma-separated) |

**Examples:**

```bash
# Basic engagement
seca engagement create \
  --id pentest-2025-q1 \
  --client "ACME Corp" \
  --start-date 2025-01-15

# With end date and scope
seca engagement create \
  --id audit-prod \
  --client "Example Inc" \
  --start-date 2025-01-15 \
  --end-date 2025-01-30 \
  --scope "*.example.com,api.example.com"

# With description
seca engagement create \
  --id compliance-check \
  --client "HealthCare Co" \
  --start-date 2025-01-15 \
  --description "HIPAA compliance validation"
```

**Output:**
```
Engagement 'pentest-2025-q1' created successfully.
Engagement file: /home/user/.local/share/seca-cli/engagements/pentest-2025-q1.json

Next steps:
1. Add scope: seca engagement add-scope --id pentest-2025-q1 <targets...>
2. Run checks: seca check http --id pentest-2025-q1 --roe-confirm <target>
```

---

### seca engagement list

List all engagements.

```bash
seca engagement list
```

**Output:**
```
ID                   Client          Start Date   End Date     Status
─────────────────────────────────────────────────────────────────────────
pentest-2025-q1      ACME Corp       2025-01-15   2025-01-30   Active
audit-prod           Example Inc     2025-01-10   2025-01-25   Active
compliance-check     HealthCare Co   2025-01-05   2025-01-20   Completed
```

---

### seca engagement view

View detailed engagement information as JSON.

```bash
seca engagement view --id <id>
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--id` | string | Engagement ID to view |

**Example:**

```bash
seca engagement view --id pentest-2025-q1
```

**Output:**
```json
{
  "id": "pentest-2025-q1",
  "client": "ACME Corp",
  "start_date": "2025-01-15",
  "end_date": "2025-01-30",
  "description": "",
  "operator": "alice@security.com",
  "scope": [
    "*.acme.com",
    "api.acme.com",
    "192.168.1.0/24"
  ],
  "roe_accepted": true,
  "created_at": "2025-01-15T10:30:00Z"
}
```

---

### seca engagement delete

Delete an engagement and all associated data.

```bash
seca engagement delete --id <id>
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--id` | string | Engagement ID to delete |

**Example:**

```bash
seca engagement delete --id old-engagement
```

**Output:**
```
Are you sure you want to delete engagement 'old-engagement'? (y/N): y

Deleted engagement: old-engagement
Removed: /home/user/.local/share/seca-cli/engagements/old-engagement.json
Removed: /home/user/.local/share/seca-cli/results/old-engagement/
Removed: /home/user/.local/share/seca-cli/telemetry/old-engagement.jsonl
```

**Warning:** This action is **irreversible** and removes:
- Engagement definition
- All results and audit logs
- Telemetry data
- Evidence packages

---

### seca engagement add-scope

Add targets to an engagement's authorized scope.

```bash
seca engagement add-scope --id <id> <target1> <target2> ...
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--id` | string | Engagement ID |

**Arguments:**
- One or more targets (URLs, hostnames, IP addresses, CIDR ranges)

**Examples:**

```bash
# Add single target
seca engagement add-scope --id eng123 example.com

# Add multiple targets
seca engagement add-scope --id eng123 \
  api.example.com \
  app.example.com \
  192.168.1.100

# Add wildcard and CIDR range
seca engagement add-scope --id eng123 \
  "*.example.com" \
  "10.0.0.0/24"
```

**Output:**
```
Added 3 scope entries to engagement 'eng123':
  - api.example.com
  - app.example.com
  - 192.168.1.100

Total scope entries: 6
```

---

## Check Commands

### seca check http

Run HTTP/HTTPS and TLS security checks.

```bash
seca check http --id <id> --roe-confirm [flags] <target1> <target2> ...
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--id` | string | Engagement ID |
| `--roe-confirm` | bool | Confirm Rules of Engagement |

**Check-Specific Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--audit-append-raw` | bool | false | Save raw HTTP headers/body for evidence |
| `--crawl` | bool | false | Discover same-host links before running checks |
| `--crawl-depth` | int | 1 | Maximum link depth to follow when crawling |
| `--crawl-max-pages` | int | 25 | Maximum additional pages per scoped target |
| `--retention-days` | int | - | Retention period for raw captures (required with `--audit-append-raw` in compliance mode) |

**Examples:**

```bash
# Basic HTTP check
seca check http --id eng123 --roe-confirm example.com

# Multiple targets with concurrency
seca check http --id eng123 --roe-confirm \
  --concurrency 10 \
  api.example.com app.example.com web.example.com

# Crawl within each scoped host
seca check http --id hipaa-audit --roe-confirm \
  --crawl \
  --crawl-depth 2 \
  --crawl-max-pages 40 \
  portal.examplehospital.com

# Compliance mode with raw capture
seca check http --id hipaa-audit --roe-confirm \
  --compliance-mode \
  --audit-append-raw \
  --retention-days 2555 \
  --hash sha512 \
  patient-portal.example.com

# High-throughput scan with rate limiting
seca check http --id bulk-scan --roe-confirm \
  --concurrency 50 \
  --rate 100 \
  --timeout 5 \
  --progress \
  --targets-file hosts.txt

# Retry failed targets
seca check http --id reliability-test --roe-confirm \
  --retry 3 \
  --concurrency 10 \
  example.com test.com demo.com
```

**Checks Performed:**
- HTTP/HTTPS connectivity
- TLS certificate validation and expiry
- Security headers analysis (OWASP Secure Headers Project)
- Cookie security (Secure/HttpOnly flags)
- CORS policy inspection
- Third-party script inventory
- Cache policy analysis
- robots.txt and sitemap.xml parsing

**Output:**
```
Running HTTP checks for engagement 'eng123'...

Checked 3 targets in 2.4 seconds
Results: /home/user/.local/share/seca-cli/results/eng123/http_results.json
Audit: /home/user/.local/share/seca-cli/results/eng123/audit.csv

SHA-256 audit: a3b5c8d9e1f2a4b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0
SHA-256 results: 1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef

Verification:
  sha256sum /home/user/.local/share/seca-cli/results/eng123/audit.csv
```

---

### seca check dns

Perform DNS resolution checks on engagement targets.

```bash
seca check dns --id <id> --roe-confirm [flags] <target1> <target2> ...
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--id` | string | Engagement ID |
| `--roe-confirm` | bool | Confirm Rules of Engagement |

**Check-Specific Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dns-timeout` | int | 10 | DNS query timeout in seconds |
| `--nameservers` | []string | system default | Custom DNS nameservers (e.g., `8.8.8.8:53`) |

**Examples:**

```bash
# Basic DNS check
seca check dns --id eng123 --roe-confirm example.com

# Multiple targets
seca check dns --id eng123 --roe-confirm \
  example.com test.com demo.com

# Custom nameserver
seca check dns --id eng123 --roe-confirm \
  --nameservers 8.8.8.8:53 \
  example.com

# Multiple nameservers for DNS propagation testing
seca check dns --id dns-migration --roe-confirm \
  --nameservers 8.8.8.8:53,1.1.1.1:53,208.67.222.222:53 \
  newdomain.com

# Internal DNS server
seca check dns --id internal-audit --roe-confirm \
  --nameservers 192.168.1.1:53 \
  internal.corp.local
```

**Checks Performed:**
- A records (IPv4 addresses)
- AAAA records (IPv6 addresses)
- CNAME records (aliases)
- MX records (mail servers)
- NS records (nameservers)
- TXT records (SPF, DKIM, DMARC, etc.)
- PTR records (reverse DNS)

**Output:**
```
Running DNS checks for engagement 'eng123'...

Checked 3 targets in 1.2 seconds
Results: /home/user/.local/share/seca-cli/results/eng123/dns_results.json
Audit: /home/user/.local/share/seca-cli/results/eng123/audit.csv

SHA-256 audit: b4c6d8e0f2a4b6c8d0e2f4a6b8c0d2e4f6a8b0c2d4e6f8a0b2c4d6e8f0a2b4c6
SHA-256 results: 5678901234abcdef5678901234abcdef5678901234abcdef5678901234abcdef
```

---

### seca check [plugin-name]

Run custom plugin checks (see [Plugin Development Guide](../developer-guide/plugin-development.md)).

```bash
seca check <plugin-name> --id <id> --roe-confirm [flags] <target>
```

**Example:**

```bash
# Custom port scanner plugin
seca check port-scan --id eng123 --roe-confirm \
  --concurrency 5 \
  example.com

# Custom API security plugin
seca check api-security --id eng123 --roe-confirm \
  api.example.com
```

---

## Report Commands

### seca report generate

Generate a summary report for an engagement.

```bash
seca report generate --id <id> [flags]
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--id` | string | Engagement ID |

**Optional Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | `markdown` | Output format (`markdown`, `html`, `json`, `pdf`) |
| `--output` | string | auto | Output file path |

**Examples:**

```bash
# Generate Markdown report
seca report generate --id eng123

# Generate HTML report
seca report generate --id eng123 --format html

# Generate PDF report
seca report generate --id eng123 --format pdf --output report.pdf

# Generate JSON report for automation
seca report generate --id eng123 --format json > report.json
```

**Output:**
```
Generating report for engagement 'eng123'...

Report generated: /home/user/.local/share/seca-cli/results/eng123/report.md

Summary:
  Total Checks: 150
  Success: 142 (94.7%)
  Warnings: 5 (3.3%)
  Failures: 2 (1.3%)
  Errors: 1 (0.7%)
```

---

### seca report stats

Show analytics summary for an engagement.

```bash
seca report stats --id <id> [flags]
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--id` | string | Engagement ID |

**Optional Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | `table` | Output format (`table`, `json`, `csv`, `markdown`) |

**Examples:**

```bash
# Colorized table output
seca report stats --id eng123 --format table

# JSON export for analysis
seca report stats --id eng123 --format json > stats.json

# CSV export for spreadsheet
seca report stats --id eng123 --format csv > stats.csv

# Markdown for documentation
seca report stats --id eng123 --format markdown > statistics.md
```

**Output (table format):**
```
Engagement Statistics: eng123

Check Type │ Total │ Success │ Warning │ Failed │ Error │ Success Rate
───────────┼───────┼─────────┼─────────┼────────┼───────┼─────────────
HTTP       │   150 │     142 │       5 │      2 │     1 │        94.7%
DNS        │   150 │     148 │       0 │      1 │     1 │        98.7%
───────────┼───────┼─────────┼─────────┼────────┼───────┼─────────────
Total      │   300 │     290 │       5 │      3 │     2 │        96.7%

Top Errors:
- Connection timeout: 2 occurrences
- Invalid certificate: 1 occurrence
```

---

### seca report telemetry

Display telemetry success rate trends over time.

```bash
seca report telemetry --id <id> [flags]
```

**Required Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--id` | string | Engagement ID |

**Optional Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | `graph` | Output format (`graph`, `json`) |

**Examples:**

```bash
# ASCII graph visualization
seca report telemetry --id eng123

# JSON export for external graphing
seca report telemetry --id eng123 --format json > telemetry.json
```

**Output (graph format):**
```
Engagement: eng123
Telemetry Success Rate Over Time

2025-01-15 10:00 |████████████████████| 100% (50/50)
2025-01-15 11:00 |██████████████████░░| 95%  (47/50)
2025-01-15 12:00 |████████████████░░░░| 88%  (44/50)
2025-01-15 13:00 |█████████████████░░░| 92%  (46/50)

Average Success Rate: 94%
Total Checks: 200
Failed Checks: 12
```

---

## Configuration

### Configuration File

**Location:** `~/.seca-cli.yaml`

**Example:**

```yaml
# Data storage directory
results_dir: /custom/path/to/results

# Default operator
operator: alice@security.com

# GPG configuration
gpg:
  key_id: alice@security.com
  auto_sign: true

# Default check settings
checks:
  concurrency: 10
  rate_limit: 50
  timeout: 30
  hash_algorithm: sha512

# Compliance settings
compliance:
  mode: true
  retention_days: 2555
  secure_results: true
```

**See:** [Configuration Guide](../user-guide/configuration.md)

---

## Exit Codes

SECA-CLI uses standard exit codes:

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Invalid arguments or flags |
| `130` | Interrupted by user (Ctrl-C) |

**Examples:**

```bash
# Check exit code
seca check http --id eng123 --roe-confirm example.com
echo $?  # 0 = success, non-zero = error

# Use in scripts
if seca check http --id eng123 --roe-confirm example.com; then
  echo "Check succeeded"
else
  echo "Check failed"
  exit 1
fi
```

---

## Flag Precedence

When the same setting is configured in multiple places, precedence is:

1. **Command-line flags** (highest priority)
2. **Environment variables**
3. **Configuration file**
4. **Built-in defaults** (lowest priority)

**Example:**

```yaml
# ~/.seca-cli.yaml
operator: alice@security.com
```

```bash
# Environment variable overrides config file
export SECA_OPERATOR=bob@security.com

# Command-line flag overrides everything
seca --operator charlie@security.com check http --id eng123 --roe-confirm example.com
# Effective operator: charlie@security.com
```

---

## Common Flag Combinations

### Production Compliance Scan

```bash
seca check http --id prod-compliance --roe-confirm \
  --compliance-mode \
  --hash sha512 \
  --secure-results \
  --auto-sign \
  --gpg-key alice@security.com \
  --concurrency 5 \
  --rate 10 \
  --timeout 60 \
  --progress \
  --telemetry \
  --targets-file production-hosts.txt
```

### Fast Bulk Scan

```bash
seca check http --id bulk-scan --roe-confirm \
  --concurrency 100 \
  --rate 200 \
  --timeout 5 \
  --progress \
  --retry 2 \
  --targets-file large-list.txt
```

### Internal Network Audit

```bash
seca check dns --id internal-audit --roe-confirm \
  --nameservers 192.168.1.1:53 \
  --concurrency 20 \
  --dns-timeout 5 \
  --targets-file internal-hosts.txt
```

---

## Quick Reference

### Essential Commands

```bash
# Create engagement
seca engagement create --id <id> --client <name> --start-date <date>

# Add scope
seca engagement add-scope --id <id> <target1> <target2> ...

# Run HTTP check
seca check http --id <id> --roe-confirm <target>

# Run DNS check
seca check dns --id <id> --roe-confirm <target>

# View statistics
seca report stats --id <id>

# Generate report
seca report generate --id <id> --format html

# List engagements
seca engagement list

# Launch TUI
seca tui
```

### Useful Flag Shortcuts

```bash
# Concurrency and rate limiting
-c 10 -r 50

# Progress and telemetry
--progress --telemetry

# Compliance mode
--compliance-mode --hash sha512 --secure-results

# Retry failed targets
--retry 3

# Custom timeout
-t 60
```

---

## Getting Help

```bash
# General help
seca --help

# Command-specific help
seca engagement --help
seca check --help
seca check http --help
seca report --help

# Version information
seca version

# System information
seca info
```

---

## See Also

- [Installation Guide](../user-guide/installation.md)
- [Configuration Guide](../user-guide/configuration.md)
- [Advanced Features Guide](../user-guide/advanced-features.md)
- [Plugin Development Guide](../developer-guide/plugin-development.md)
- [Compliance Guide](../operator-guide/compliance.md)
- [Operator Training](../operator-guide/operator-training.md)
