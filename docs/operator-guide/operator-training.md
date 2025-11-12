## SECA-CLI Operator Training Guide

**Security Engagement & Compliance Auditing CLI**

Version: 1.0.0
Last Updated: 2025-11-09

---

## Table of Contents

1. [Introduction](#introduction)
2. [Prerequisites](#prerequisites)
3. [Installation](#installation)
4. [Core Concepts](#core-concepts)
5. [Hands-On Training](#hands-on-training)
6. [Compliance Requirements](#compliance-requirements)
7. [Common Workflows](#common-workflows)
8. [Troubleshooting](#troubleshooting)
9. [Certification Checklist](#certification-checklist)

---

## Introduction

### What is SECA-CLI?

SECA-CLI is a professional command-line tool designed for **authorized** security testing with built-in compliance, evidence integrity, and audit trail capabilities.

### Training Objectives

By the end of this training, operators will be able to:

- ✅ Create and manage security testing engagements
- ✅ Define and maintain authorized scope
- ✅ Execute safe, non-invasive HTTP checks
- ✅ Generate compliant audit trails
- ✅ Verify evidence integrity with cryptographic hashing
- ✅ Package evidence for delivery to clients

### Target Audience

- Security analysts
- Penetration testers
- Compliance auditors
- Security operations teams

---

## Prerequisites

### Required Knowledge

- Basic command-line interface (CLI) usage
- Understanding of HTTP/HTTPS protocols
- Familiarity with security testing concepts
- Basic understanding of legal and ethical hacking

### System Requirements

- Operating System: Linux, macOS, or Windows
- Go 1.21+ (for building from source)
- `sha256sum` command (for hash verification)
- `gpg` (optional, for signing)
- Internet connectivity

### Legal Requirements

⚠️ **CRITICAL**: Operators MUST have:

- Written authorization for all testing activities
- Documented Rules of Engagement (ROE)
- Clear understanding of authorized scope
- Compliance with applicable laws and regulations

---

## Installation

### Method 1: Download Pre-built Binary

```bash
# Linux (amd64)
wget https://github.com/khanhnv2901/seca-cli/releases/download/v1.0.0/seca-1.0.0-linux-amd64
chmod +x seca-1.0.0-linux-amd64
sudo mv seca-1.0.0-linux-amd64 /usr/local/bin/seca

# Verify installation
seca version
```

### Method 2: Build from Source

```bash
git clone https://github.com/khanhnv2901/seca-cli.git
cd seca-cli
make build
sudo make install
```

### Configuration

Create a configuration file (optional):

```bash
mkdir -p ~/.config/seca-cli
cat > ~/.config/seca-cli/config.yaml << EOF
results_dir: /path/to/results
operator: your-name
EOF
```

---

## Core Concepts

### 1. Engagements

An **engagement** represents a security testing project with:
- Unique ID
- Client/project name
- Owner/POC
- Rules of Engagement (ROE)
- Authorized scope
- Start/end dates

### 2. Scope

**Scope** defines the authorized targets:
- URLs (e.g., `https://example.com`)
- Hosts (e.g., `api.example.com`)
- IP ranges (documented in engagement notes)

**⚠️ Testing outside of scope is PROHIBITED**

### 3. Audit Trail

Every action is logged in `audit.csv` with:
- Timestamp (UTC)
- Engagement ID
- Operator name
- Command executed
- Target
- Status
- Duration
- Errors (if any)

### 4. Evidence Integrity

All evidence files include SHA256 hashes:
- `audit.csv.sha256`
- `results.json.sha256`

Verification ensures files haven't been tampered with.

### 5. Compliance Mode

Compliance mode enforces:
- Mandatory operator identification
- Automatic hash generation
- Retention policy validation
- Immutable audit logs

---

## Hands-On Training

### Module 1: Creating Your First Engagement

**Scenario**: You've been tasked to perform a security assessment for Acme Corp.

**Step 1: Review Authorization**

Before anything, verify you have:
- [ ] Written authorization letter
- [ ] Signed ROE document
- [ ] Clear scope definition
- [ ] Contact information for emergencies

**Step 2: Create Engagement**

```bash
seca engagement create \
  --name "Acme Corp Q1 2025 Security Assessment" \
  --owner "security@acmecorp.com" \
  --roe "Authorization letter signed 2025-01-10 by CTO" \
  --roe-agree
```

**Expected Output:**
```
Created engagement Acme Corp Q1 2025 Security Assessment (id=1762627948156627663)
```

**Step 3: Save Engagement ID**

```bash
# Save for future use
ENGAGEMENT_ID=1762627948156627663
```

---

### Module 2: Managing Scope

**Step 1: Add Authorized Targets**

```bash
seca engagement add-scope \
  --id $ENGAGEMENT_ID \
  --scope https://acmecorp.com,https://api.acmecorp.com,https://admin.acmecorp.com
```

**Step 2: Verify Scope**

```bash
seca engagement list | jq ".[] | select(.id==\"$ENGAGEMENT_ID\")"
```

**Step 3: Document Exclusions**

Always note what is OUT of scope:
- Production databases
- Customer data
- Third-party services
- Legacy systems marked off-limits

---

### Module 3: Running HTTP Checks

**Step 1: Basic Check**

```bash
seca --operator "$(whoami)" check http \
  --id $ENGAGEMENT_ID \
  --roe-confirm \
  --concurrency 2 \
  --rate 1 \
  --timeout 10
```

**Parameters Explained:**
- `--operator`: Your name (for accountability)
- `--roe-confirm`: Explicit confirmation of authorization
- `--concurrency 2`: Max 2 parallel requests
- `--rate 1`: 1 request per second (be respectful!)
- `--timeout 10`: 10-second timeout

**Step 2: Compliance Mode Check**

```bash
seca --operator "$(whoami)" check http \
  --id $ENGAGEMENT_ID \
  --roe-confirm \
  --compliance-mode \
  --concurrency 4 \
  --rate 3
```

**Step 3: With Raw Capture (Advanced)**

```bash
seca --operator "$(whoami)" check http \
  --id $ENGAGEMENT_ID \
  --roe-confirm \
  --compliance-mode \
  --audit-append-raw \
  --retention-days 90
```

⚠️ **Raw captures may contain PII - use with caution!**

---

### Module 4: Verifying Results

**Step 1: Check Results Directory**

```bash
ls -la results/$ENGAGEMENT_ID/
```

**Expected files:**
```
audit.csv
audit.csv.sha256
results.json
results.json.sha256
```

**Step 2: Verify Hashes**

```bash
cd results/$ENGAGEMENT_ID/
sha256sum -c audit.csv.sha256
sha256sum -c results.json.sha256
```

**Expected output:**
```
audit.csv: OK
results.json: OK
```

**Step 3: Review Audit Log**

```bash
cat audit.csv | column -t -s ','
```

---

### Module 5: Packaging Evidence

**Step 1: Verify Evidence**

```bash
make verify ENGAGEMENT_ID=$ENGAGEMENT_ID
```

**Step 2: Sign Evidence (Optional)**

```bash
make sign ENGAGEMENT_ID=$ENGAGEMENT_ID
```

**Step 3: Create Delivery Package**

```bash
make package ENGAGEMENT_ID=$ENGAGEMENT_ID
```

**Output:**
```
evidence-1762627948156627663.tar.gz
evidence-1762627948156627663.tar.gz.asc
evidence-1762627948156627663.tar.gz.sha256
```

---

## Compliance Requirements

### Before Testing

- [ ] Obtain written authorization
- [ ] Document ROE and get acknowledgment
- [ ] Define clear scope boundaries
- [ ] Identify emergency contacts
- [ ] Schedule testing window (if applicable)

### During Testing

- [ ] Use `--roe-confirm` for all checks
- [ ] Respect rate limits
- [ ] Stay within authorized scope
- [ ] Monitor for unintended impact
- [ ] Document any issues immediately

### After Testing

- [ ] Verify all hash files
- [ ] Review audit logs for completeness
- [ ] Package evidence properly
- [ ] Securely transmit to client
- [ ] Follow retention policies
- [ ] Delete raw captures after retention period

---

## Common Workflows

### Workflow 1: Quick Health Check

```bash
# 1. Create engagement
ID=$(seca engagement create --name "Health Check" --owner "ops@company.com" --roe "Routine check" --roe-agree | grep -oP 'id=\K[0-9]+')

# 2. Add scope
seca engagement add-scope --id $ID --scope https://example.com

# 3. Run check
seca check http --id $ID --roe-confirm

# 4. Verify
make verify ENGAGEMENT_ID=$ID
```

### Workflow 2: Compliance Audit

```bash
# Full compliance workflow
seca --operator "auditor-name" check http \
  --id $ID \
  --roe-confirm \
  --compliance-mode \
  --audit-append-raw \
  --retention-days 90 \
  --concurrency 4 \
  --rate 3

# Package for delivery
make package ENGAGEMENT_ID=$ID
```

### Workflow 3: Large-Scale Assessment

```bash
# Add multiple targets
seca engagement add-scope --id $ID \
  --scope https://app1.example.com,https://app2.example.com,https://app3.example.com

# Higher concurrency for speed
seca --operator "$(whoami)" check http \
  --id $ID \
  --roe-confirm \
  --compliance-mode \
  --concurrency 10 \
  --rate 5
```

---

## Troubleshooting

### Issue: "ROE must be agreed"

**Solution:**
```bash
# Use --roe-agree flag
seca engagement create ... --roe-agree
```

### Issue: "Unknown flag: --rate"

**Solution:**
```bash
# Use --rate-limit instead
seca check http ... --rate-limit 3
```

### Issue: "No engagement found"

**Solution:**
```bash
# List all engagements
seca engagement list

# Verify ID is correct
```

### Issue: Hash Verification Failed

**Solution:**
```bash
# File may have been modified
# DO NOT proceed - investigate tampering
# Report to security team immediately
```

---

## Certification Checklist

### Knowledge Assessment

- [ ] I understand what SECA-CLI does and doesn't do
- [ ] I can create and manage engagements
- [ ] I know how to define authorized scope
- [ ] I can execute HTTP checks with proper parameters
- [ ] I understand compliance mode requirements
- [ ] I can verify evidence integrity
- [ ] I know how to package evidence for delivery

### Practical Skills

- [ ] Successfully created an engagement
- [ ] Added scope to an engagement
- [ ] Executed HTTP checks in compliance mode
- [ ] Verified hash integrity
- [ ] Packaged evidence for delivery
- [ ] Reviewed audit logs

### Legal & Ethical

- [ ] I will only test systems with written authorization
- [ ] I will respect scope boundaries
- [ ] I will use `--roe-confirm` for all operations
- [ ] I will follow retention and data handling policies
- [ ] I understand the consequences of unauthorized testing

---

## Additional Resources

- [README.md](README.md) - General documentation
- [Compliance Guide](../operator-guide/compliance.md) - Detailed compliance guide
- [Testing Guide](../technical/testing.md) - Testing and validation
- [Makefile](Makefile) - Automation commands

## Support

- GitHub Issues: https://github.com/khanhnv2901/seca-cli/issues
- Email: khanhnv2901@gmail.com

---

**Certification Statement**

I, __________________________, have completed the SECA-CLI Operator Training and understand the legal, ethical, and technical requirements for using this tool.

Date: ______________
Signature: ______________

---

*This training guide is provided for educational purposes. Operators are responsible for compliance with all applicable laws and regulations.*
