# Advanced Features Guide

## Overview

SECA-CLI includes several advanced features for power users, automation, and specialized security testing scenarios. This guide covers features beyond basic engagement management and security checks.

## Table of Contents

- [Retry Mechanism](#retry-mechanism)
- [Progress Display](#progress-display)
- [Telemetry and Metrics](#telemetry-and-metrics)
- [Secure Results Encryption](#secure-results-encryption)
- [Hash Algorithm Selection](#hash-algorithm-selection)
- [Custom DNS Nameservers](#custom-dns-nameservers)
- [Graceful Cancellation](#graceful-cancellation)
- [Advanced Engagement Commands](#advanced-engagement-commands)
- [Interactive TUI Mode](#interactive-tui-mode)

---

## Retry Mechanism

### Overview

The `--retry` flag automatically retries failed targets, improving reliability for flaky networks or intermittent failures.

### Usage

```bash
seca check http --id eng123 --retry 3 host1.com host2.com host3.com
```

**Behavior:**
- Initial check attempt for all targets
- Failed targets are retried up to N times
- Successful results are kept, failed targets retry
- Final results include all successful checks

### Example Scenario

```bash
# Check 100 hosts with 2 retries for failures
seca check http --id prod-scan \
  --retry 2 \
  --concurrency 10 \
  --targets-file hosts.txt
```

**Output:**
```
Initial run: 95/100 succeeded, 5 failed
Retry 1: 3/5 succeeded, 2 failed
Retry 2: 1/2 succeeded, 1 failed
Final: 99/100 targets checked
```

### Retry Strategy

- **Exponential Backoff**: Not implemented; retries happen immediately after initial batch
- **Rate Limiting**: Retries respect `--rate-limit` flag
- **Concurrency**: Retries use same `--concurrency` setting
- **Partial Results**: Successful checks from any attempt are saved

### Use Cases

| Scenario | Recommended Retries |
|----------|---------------------|
| Stable internal network | `--retry 0` (default, no retries) |
| Public internet targets | `--retry 2` |
| Flaky network/VPN | `--retry 3` |
| Critical compliance scan | `--retry 5` |

### Limitations

- Does not retry **successful** checks
- Timeout failures are retried (may hit timeout again)
- No exponential backoff (consider adding delay between batches manually)

---

## Progress Display

### Overview

The `--progress` flag shows real-time progress for long-running scans with multiple targets.

### Usage

```bash
seca check http --id eng123 --progress host1.com host2.com host3.com
```

**Output:**
```
Checking targets: 15/100 (15%) [===>                    ] 00:45 elapsed
```

### Features

- **Live Progress Bar**: Visual indicator of completion percentage
- **Elapsed Time**: Total time since scan started
- **Completion Count**: Current/total targets processed
- **ETA**: Not currently displayed (future enhancement)

### Example with Large Target Set

```bash
seca check http --id quarterly-scan \
  --targets-file 500-hosts.txt \
  --concurrency 20 \
  --progress
```

**Output:**
```
Checking targets: 127/500 (25%) [=====>              ] 02:15 elapsed
Checking targets: 256/500 (51%) [==========>         ] 04:32 elapsed
Checking targets: 500/500 (100%) [====================] 08:47 elapsed

Results: /home/user/.local/share/seca-cli/results/quarterly-scan/http_results.json
```

### When to Use Progress Display

- Large target sets (50+ hosts)
- Long-running checks (DNS, TLS with many targets)
- Interactive terminal sessions (not in CI/CD)
- Monitoring scan status

### Disabling Progress

Progress is **off by default**. Omit `--progress` flag for cleaner output in automation:

```bash
# CI/CD: No progress bar
seca check http --id ci-scan --targets-file hosts.txt > scan.log 2>&1
```

---

## Telemetry and Metrics

### Overview

Telemetry tracks success rates, errors, and performance metrics over time for operational monitoring and trend analysis.

### Enabling Telemetry

Add `--telemetry` flag to any check command:

```bash
seca check http --id eng123 --telemetry example.com
```

### Telemetry Data

Stored in: `~/.local/share/seca-cli/telemetry/<engagement-id>.jsonl`

**Format (JSONL - one JSON object per line):**
```json
{"timestamp":"2025-01-15T10:30:00Z","check":"http","target":"example.com","status":"success","duration":0.234}
{"timestamp":"2025-01-15T10:30:01Z","check":"http","target":"fail.com","status":"error","duration":30.001}
```

### Viewing Telemetry

```bash
# Generate telemetry report with ASCII graph
seca report telemetry --id eng123

# Export as JSON for analysis
seca report telemetry --id eng123 --format json > metrics.json
```

**Example Output (ASCII Graph):**
```
Engagement: eng123
Check Type: http
Success Rate Over Time:

2025-01-15 10:00 |████████████████████| 100% (50/50)
2025-01-15 11:00 |██████████████████░░| 95%  (47/50)
2025-01-15 12:00 |████████████████░░░░| 88%  (44/50)
2025-01-15 13:00 |█████████████████░░░| 92%  (46/50)

Average Success Rate: 94%
Total Checks: 200
Failed Checks: 12
```

### Telemetry Use Cases

**1. Trend Analysis**
```bash
# Track daily scans over a month
for day in {01..30}; do
  seca check http --id monthly-scan --telemetry --targets-file hosts.txt
  sleep 86400  # Daily
done

# Analyze trends
seca report telemetry --id monthly-scan
```

**2. Performance Monitoring**
```bash
# Monitor response times
seca report telemetry --id eng123 --format json | \
  jq '.[] | select(.status=="success") | .duration' | \
  awk '{sum+=$1; count+=1} END {print "Avg:", sum/count, "seconds"}'
```

**3. Reliability Tracking**
```bash
# Success rate for compliance reporting
seca report telemetry --id compliance-q1 --format json | \
  jq '[.[] | select(.status=="success")] | length'
```

### Telemetry Data Retention

- Stored indefinitely until engagement is deleted
- Use `seca engagement delete --id <id>` to remove engagement + telemetry
- Manual cleanup: `rm ~/.local/share/seca-cli/telemetry/<engagement-id>.jsonl`

---

## Secure Results Encryption

### Overview

The `--secure-results` flag encrypts audit logs and results using GPG, ensuring evidence integrity and confidentiality.

### Prerequisites

1. **GPG Installed**: `gpg --version`
2. **GPG Key Generated**: See [GPG Setup](#gpg-setup)
3. **Public Key Available**: For encryption

### GPG Setup

```bash
# Generate a new GPG key (if needed)
gpg --full-generate-key

# List available keys
gpg --list-keys

# Export public key for sharing
gpg --export --armor your-email@example.com > pubkey.asc
```

### Usage

```bash
seca check http --id eng123 --secure-results host1.com host2.com

# Results are encrypted with your default GPG key
# Output:
# Results: /home/user/.local/share/seca-cli/results/eng123/http_results.json.gpg
# Audit: /home/user/.local/share/seca-cli/results/eng123/audit.csv.gpg
```

### Decrypting Results

```bash
# Decrypt results file
gpg --decrypt ~/.local/share/seca-cli/results/eng123/http_results.json.gpg > results.json

# Decrypt audit log
gpg --decrypt ~/.local/share/seca-cli/results/eng123/audit.csv.gpg > audit.csv
```

### Specifying GPG Recipient

```bash
# Encrypt for specific recipient
export GPG_RECIPIENT="security-team@company.com"
seca check http --id eng123 --secure-results example.com
```

### Use Cases

- **Sensitive Engagements**: PCI-DSS, PHI, classified systems
- **Evidence Delivery**: Encrypt before transmitting to client
- **Compliance**: Meet encryption-at-rest requirements
- **Multi-Operator**: Share results securely with team members

### Limitations

- Requires GPG key setup
- Results must be decrypted before analysis
- Adds overhead to check execution
- Not compatible with all automated pipelines

---

## Hash Algorithm Selection

### Overview

SECA-CLI supports SHA-256 (default) and SHA-512 for evidence integrity verification.

### Usage

```bash
# Default: SHA-256
seca check http --id eng123 example.com

# Use SHA-512 for stricter integrity
seca check http --id eng123 --hash sha512 example.com
```

### Algorithm Comparison

| Feature | SHA-256 | SHA-512 |
|---------|---------|---------|
| **Hash Length** | 64 characters | 128 characters |
| **Security Level** | 128-bit | 256-bit |
| **Performance** | Faster | Slightly slower |
| **Compliance** | NIST FIPS 180-4 | NIST FIPS 180-4 |
| **Recommended For** | General use | High-security environments |

### When to Use SHA-512

- **Regulated Industries**: Banking, healthcare, government
- **Long-Term Archival**: Evidence stored for 5+ years
- **High-Security Engagements**: Critical infrastructure, classified systems
- **Compliance Requirements**: SOC 2 Type II, ISO 27001, NIST 800-53

### Hash Output Examples

**SHA-256:**
```
SHA-256 audit: a3b5c8d9e1f2a4b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0
SHA-256 results: 1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
```

**SHA-512:**
```
SHA-512 audit: a3b5c8d9e1f2a4b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8
SHA-512 results: 1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
```

### Verification

```bash
# Verify SHA-256 hash
sha256sum ~/.local/share/seca-cli/results/eng123/audit.csv
# Compare with output from seca check

# Verify SHA-512 hash
sha512sum ~/.local/share/seca-cli/results/eng123/audit.csv
# Compare with output from seca check
```

### Performance Impact

- **SHA-256**: Negligible impact on most systems
- **SHA-512**: ~5-10% slower on large result sets
- **Recommendation**: Use SHA-256 unless compliance mandates SHA-512

---

## Custom DNS Nameservers

### Overview

Specify custom DNS nameservers for DNS checks, useful for internal networks or testing DNS propagation.

### Usage

```bash
# Use Google Public DNS
seca check dns --id eng123 --nameserver 8.8.8.8 example.com

# Use multiple nameservers
seca check dns --id eng123 --nameserver 8.8.8.8 --nameserver 1.1.1.1 example.com

# Use internal DNS server
seca check dns --id eng123 --nameserver 192.168.1.1 internal.corp
```

### Use Cases

**1. Internal Networks**
```bash
# Query internal DNS server for private domains
seca check dns --id internal-audit \
  --nameserver 10.0.0.53 \
  --targets-file internal-hosts.txt
```

**2. DNS Propagation Testing**
```bash
# Check if DNS changes propagated to multiple providers
seca check dns --id dns-migration \
  --nameserver 8.8.8.8 \     # Google
  --nameserver 1.1.1.1 \     # Cloudflare
  --nameserver 208.67.222.222 \  # OpenDNS
  newdomain.com
```

**3. DNS Security Validation**
```bash
# Verify DNSSEC with specific resolver
seca check dns --id dnssec-check \
  --nameserver 9.9.9.9 \  # Quad9 (DNSSEC validating)
  secure-domain.com
```

### Default Behavior

Without `--nameserver`, SECA-CLI uses system default resolvers (`/etc/resolv.conf`).

---

## Graceful Cancellation

### Overview

Press `Ctrl-C` during a scan to gracefully cancel, saving partial results and audit entries for completed targets.

### Behavior

**Before Cancellation:**
```bash
seca check http --id eng123 host1.com host2.com host3.com ... host100.com
Checking targets: 42/100...
^C  # User presses Ctrl-C
```

**After Cancellation:**
```
Gracefully cancelling... (press Ctrl-C again to force quit)
Saving partial results for 42 completed targets...

Results: /home/user/.local/share/seca-cli/results/eng123/http_results.json (42 entries)
Audit: /home/user/.local/share/seca-cli/results/eng123/audit.csv (42 entries)
SHA-256 audit: a3b5c8d9...
```

### Features

- **Partial Results Saved**: All completed checks are written to disk
- **Audit Trail Updated**: CSV includes all finished targets
- **Integrity Hashes**: Generated for partial results
- **Resume Later**: Re-run with same engagement ID to check remaining targets

### Force Quit

Press `Ctrl-C` **twice** to force immediate exit (no partial results saved):

```bash
^C  # First press: Graceful cancel
^C  # Second press: Force quit (no results saved)
```

### Use Cases

**1. Long-Running Scans**
```bash
# Start large scan, cancel after sampling
seca check http --id test-sample --targets-file 1000-hosts.txt
# ^C after 100 hosts to review sample results
```

**2. Accidental Scans**
```bash
# Oops, wrong target list
seca check http --id wrong-engagement --targets-file production.txt
# ^C immediately, partial results saved
```

**3. Time-Limited Testing**
```bash
# "Check as many as possible in 30 minutes"
timeout 30m seca check http --id time-limited --targets-file large-list.txt
# Partial results saved on timeout
```

### Resuming After Cancellation

```bash
# Initial scan (cancelled)
seca check http --id eng123 --targets-file all-hosts.txt
# ^C after 50/100 targets

# Resume: Only unchecked targets remain
seca check http --id eng123 --targets-file remaining-hosts.txt
# OR: Re-run with same file (skip duplicates manually)
```

---

## Advanced Engagement Commands

### Quick View Engagement

```bash
# View engagement details as JSON
seca engagement view --id eng123
```

**Output:**
```json
{
  "id": "eng123",
  "client": "ACME Corp",
  "start_date": "2025-01-15",
  "end_date": "2025-01-30",
  "operator": "alice@security.com",
  "scope": [
    "*.acme.com",
    "192.168.1.0/24"
  ],
  "roe_accepted": true
}
```

### Delete Engagement

```bash
# Delete engagement and all associated data
seca engagement delete --id eng123

# Confirmation prompt
Are you sure you want to delete engagement 'eng123'? (y/N): y

# Output
Deleted engagement: eng123
Removed: /home/user/.local/share/seca-cli/engagements/eng123.json
Removed: /home/user/.local/share/seca-cli/results/eng123/
Removed: /home/user/.local/share/seca-cli/telemetry/eng123.jsonl
```

**Warning:** Deletion is **permanent** and removes:
- Engagement definition
- All results (JSON, CSV, audit logs)
- Telemetry data
- Evidence packages

### Interactive Statistics

```bash
# View engagement statistics with colorized output
seca report stats --id eng123 --format table
```

**Output:**
```
Engagement Statistics: eng123

Check Type │ Total │ Success │ Warning │ Failed │ Error │ Success Rate
───────────┼───────┼─────────┼─────────┼────────┼───────┼─────────────
HTTP       │   150 │     142 │       5 │      2 │     1 │        94.7%
DNS        │   150 │     148 │       0 │      1 │     1 │        98.7%
TLS        │   150 │     145 │       3 │      0 │     2 │        96.7%
───────────┼───────┼─────────┼─────────┼────────┼───────┼─────────────
Total      │   450 │     435 │       8 │      3 │     4 │        96.7%

Top Errors:
- Connection timeout: 2 occurrences
- Invalid certificate: 1 occurrence
- DNS resolution failed: 1 occurrence
```

### Export Formats

```bash
# JSON format
seca report stats --id eng123 --format json > stats.json

# Markdown format
seca report stats --id eng123 --format markdown > report.md

# CSV format
seca report stats --id eng123 --format csv > stats.csv
```

---

## Interactive TUI Mode

### Overview

The Terminal UI (TUI) provides an interactive interface for managing engagements without memorizing commands.

### Launching TUI

```bash
seca engagement tui
```

### Features

- **Visual Engagement Browser**: Navigate all engagements with arrow keys
- **Quick Actions**: Create, view, delete engagements interactively
- **Scope Management**: Add targets to engagement scope via interactive prompts
- **Color-Coded Status**: Visual indicators for active/completed engagements

### TUI Navigation

| Key | Action |
|-----|--------|
| `↑/↓` | Navigate engagement list |
| `Enter` | View selected engagement details |
| `c` | Create new engagement |
| `d` | Delete selected engagement |
| `a` | Add scope to selected engagement |
| `q` | Quit TUI |
| `/` | Search engagements |

### TUI vs CLI Workflows

**CLI Workflow (scripting, automation):**
```bash
seca engagement create --id eng123 --client "ACME Corp" --start-date 2025-01-15
seca check http --id eng123 example.com
```

**TUI Workflow (interactive, exploration):**
1. Launch: `seca engagement tui`
2. Press `c` to create engagement
3. Fill in prompts interactively
4. Press `a` to add scope
5. Exit TUI (`q`) and run checks from CLI

### When to Use TUI

- Learning SECA-CLI commands
- Exploring existing engagements
- Quick engagement creation without flags
- Visual overview of all engagements

### When to Use CLI

- Automation and scripting
- CI/CD pipelines
- Batch operations
- Precise control over parameters

---

## Combining Advanced Features

### Example: High-Security Compliance Scan

```bash
seca check http \
  --id hipaa-audit-q1 \
  --targets-file patient-portals.txt \
  --concurrency 5 \
  --rate-limit 10 \
  --retry 3 \
  --progress \
  --telemetry \
  --secure-results \
  --hash sha512 \
  --timeout 60
```

**Explanation:**
- `--retry 3`: Ensure all targets checked despite network issues
- `--progress`: Monitor long-running scan
- `--telemetry`: Track success rates for compliance reporting
- `--secure-results`: Encrypt evidence (HIPAA requirement)
- `--hash sha512`: Maximum integrity assurance
- `--concurrency 5` + `--rate-limit 10`: Minimize impact on production systems

---

## Performance Considerations

### Feature Impact on Scan Speed

| Feature | Performance Impact |
|---------|-------------------|
| `--retry N` | +N × failure_count execution time |
| `--progress` | Negligible (<1%) |
| `--telemetry` | ~2-5% overhead (JSONL writes) |
| `--secure-results` | ~10-20% overhead (GPG encryption) |
| `--hash sha512` | ~5-10% overhead vs SHA-256 |

### Optimization Tips

1. **Use SHA-256** unless compliance requires SHA-512
2. **Skip `--secure-results`** for internal scans (encrypt later if needed)
3. **Limit retries** to 2-3 for most cases
4. **Increase concurrency** for faster scans (balance with rate limits)
5. **Batch operations** instead of many small scans

---

## Summary

SECA-CLI's advanced features provide flexibility for complex security testing scenarios:

- **Retry mechanism** improves reliability
- **Progress display** provides visibility into long scans
- **Telemetry** enables trend analysis and monitoring
- **Secure results** meet encryption requirements
- **SHA-512 hashing** provides maximum integrity assurance
- **Graceful cancellation** preserves partial work
- **TUI mode** simplifies interactive workflows

Combine these features to build robust, compliant, and efficient security testing workflows tailored to your environment.
