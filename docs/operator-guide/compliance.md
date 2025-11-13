# SECA-CLI Compliance Guide

## Overview

This guide explains how to use SECA-CLI's compliance features to ensure evidence integrity, proper verification, and retention requirements for security testing engagements.

## Compliance Mode

SECA-CLI includes a `--compliance-mode` flag that enforces best practices for evidence handling:

- **Mandatory operator identification** - All actions are attributed to a named operator
- **Automatic hash signing** - SHA256 hashes are generated for audit and results files
- **Retention policy enforcement** - Raw captures require explicit retention periods
- **Audit trail integrity** - Immutable CSV audit logs with timestamps

## Running Checks in Compliance Mode

```bash
./seca --operator "your-name" check http \
  --id <engagement-id> \
  --roe-confirm \
  --compliance-mode \
  --concurrency 4 \
  --rate-limit 3 \
  --timeout 15
```

### With Raw Capture (for detailed auditing)

```bash
./seca --operator "your-name" check http \
  --id <engagement-id> \
  --roe-confirm \
  --compliance-mode \
  --audit-append-raw \
  --retention-days 90 \
  --concurrency 4 \
  --rate-limit 3
```

## Evidence Files

After running a check, SECA-CLI generates the following evidence in `results/<engagement-id>/`:

```
results/<engagement-id>/
├── audit.csv              # CSV audit log with all HTTP checks
├── audit.csv.sha256       # SHA256 hash of audit.csv
├── http_results.json           # JSON results with metadata
├── http_results.json.sha256    # SHA256 hash of http_results.json
└── raw_*.txt              # Optional: raw HTTP captures (if --audit-append-raw used)
```

## Evidence Verification

### 1. Verify Audit File Integrity

```bash
cd results/<engagement-id>/
sha256sum -c audit.csv.sha256
```

Expected output:
```
audit.csv: OK
```

### 2. Verify Results File Integrity

```bash
sha256sum -c http_results.json.sha256
```

Expected output:
```
http_results.json: OK
```

### 3. Verify Both Files at Once

```bash
sha256sum -c *.sha256
```

### 4. Manual Hash Verification

If you need to manually verify:

```bash
# Compute hash
sha256sum audit.csv

# Compare with stored hash
cat audit.csv.sha256
```

## Evidence Signing

For additional non-repudiation, you may want to sign evidence files with GPG.

### Sign Evidence Files

```bash
cd results/<engagement-id>/

# Sign audit file
gpg --detach-sign --armor audit.csv

# Sign results file
gpg --detach-sign --armor http_results.json

# This creates:
# - audit.csv.asc
# - http_results.json.asc
```

### Verify Signatures

```bash
# Verify audit signature
gpg --verify audit.csv.asc audit.csv

# Verify results signature
gpg --verify http_results.json.asc http_results.json
```

## Retention Requirements

### Raw Capture Retention

When using `--audit-append-raw`, raw HTTP response data is captured. This data may contain:
- Response headers
- Response body snippets (limited to 2048 bytes)
- Potentially sensitive or PII data

**Compliance Mode Requirement:** If `--audit-append-raw` is used with `--compliance-mode`, you MUST specify `--retention-days`.

### Retention Best Practices

1. **Standard Audit Files** (audit.csv, http_results.json)
   - Retain for the duration of the engagement + statutory period
   - Typically: 3-7 years depending on jurisdiction and contract
   - Keep SHA256 hashes and signatures with the files

2. **Raw Captures** (raw_*.txt)
   - Minimize collection (only when necessary)
   - Delete or anonymize after the specified retention period
   - Default recommendation: 90 days for active engagements
   - Extend to 1-3 years only if contractually required

3. **Engagement Metadata** (engagements.json)
   - Keep for audit trail purposes
   - Contains ROE agreement timestamps
   - No PII in scope definitions

### Automated Retention Cleanup

Create a cleanup script for raw captures:

```bash
#!/bin/bash
# cleanup-raw-captures.sh
# Run this after retention period expires

ENGAGEMENT_ID="$1"
RETENTION_DAYS="${2:-90}"

find "results/$ENGAGEMENT_ID" -name "raw_*.txt" -mtime +$RETENTION_DAYS -delete

echo "Deleted raw captures older than $RETENTION_DAYS days for engagement $ENGAGEMENT_ID"
```

Usage:
```bash
./cleanup-raw-captures.sh 1762627948156627663 90
```

## Chain of Custody

### 1. Document Operator Identity

Every check run includes operator attribution:

```json
{
  "metadata": {
    "operator": "john.doe",
    "engagement_id": "1762627948156627663",
    "engagement_name": "Client XYZ Pentest",
    "owner": "jane.smith@example.com",
    "started_at": "2025-11-09T10:30:00Z",
    "completed_at": "2025-11-09T10:35:00Z"
  }
}
```

### 2. Maintain Audit Trail

The `audit.csv` contains timestamped records:

```csv
timestamp,engagement_id,operator,command,target,status,http_status,tls_expiry,notes,error,duration_seconds
2025-11-09T10:30:15Z,1762627948156627663,john.doe,check http,https://example.com,ok,200,2026-01-15T00:00:00Z,robots.txt found,,0.234
```

### 3. Evidence Package for Delivery

When delivering evidence to clients:

```bash
# Create evidence package
cd results
tar -czf ../evidence-<engagement-id>.tar.gz <engagement-id>/

# Sign the tarball
gpg --detach-sign --armor ../evidence-<engagement-id>.tar.gz

# Generate final hash
sha256sum ../evidence-<engagement-id>.tar.gz > ../evidence-<engagement-id>.tar.gz.sha256
```

Deliver:
- `evidence-<engagement-id>.tar.gz`
- `evidence-<engagement-id>.tar.gz.asc` (GPG signature)
- `evidence-<engagement-id>.tar.gz.sha256` (SHA256 hash)

## Compliance Checklist

Before starting an engagement:

- [ ] Engagement created with `engagement create`
- [ ] ROE documented and `--roe-agree` flag acknowledged
- [ ] Scope added via `engagement add-scope`
- [ ] Operator identity configured (`--operator` flag or USER env)

During testing:

- [ ] Use `--compliance-mode` for all checks
- [ ] Use `--roe-confirm` to acknowledge authorization
- [ ] Specify `--retention-days` if using `--audit-append-raw`
- [ ] Monitor that hash files are generated

After testing:

- [ ] Verify all `.sha256` files using `sha256sum -c`
- [ ] Sign evidence files with GPG (if required)
- [ ] Package evidence for delivery
- [ ] Document retention period in engagement notes
- [ ] Schedule retention cleanup (for raw captures)

## Regulatory Considerations

### GDPR (General Data Protection Regulation)

- Minimize collection of personal data
- Use `--audit-append-raw` only when necessary
- Set appropriate `--retention-days` (typically 90 days or less)
- Document legal basis for data processing
- Implement deletion after retention period

### PCI-DSS (Payment Card Industry)

- Do not capture payment card data in raw responses
- Maintain audit logs for at least 1 year
- Restrict access to evidence files (use file permissions)
- Encrypt evidence at rest and in transit

### SOC 2 / ISO 27001

- Maintain chain of custody documentation
- Use cryptographic hashing for integrity
- Implement access controls to evidence files
- Regular audit of testing activities
- Document all changes to engagement scope

## File Permissions

Secure your evidence files:

```bash
# Restrict results directory
chmod 750 results/
chmod 640 results/<engagement-id>/*

# Only operator and authorized users can access
chown $USER:security-team results/ -R
```

## Questions & Support

For compliance-related questions or to report issues:

- GitHub Issues: https://github.com/khanhnv2901/seca-cli/issues
- Security Contact: khanhnv2901@gmail.com

## License

This compliance guide is provided for informational purposes. Consult with legal counsel to ensure compliance with applicable laws and regulations in your jurisdiction.
