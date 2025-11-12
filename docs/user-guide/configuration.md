# SECA-CLI Configuration Guide

This guide explains how to configure SECA-CLI using configuration files, command-line flags, and environment variables.

## Configuration Hierarchy

SECA-CLI uses a three-tier configuration system (highest to lowest priority):

1. **Command-line flags** (highest priority) - Always override everything
2. **Configuration file** - Settings in `~/.seca-cli.yaml`
3. **Environment variables** (lowest priority) - Used as fallback

## Configuration File

### Location

SECA-CLI looks for configuration in the following locations:

**Default location:**
```
~/.seca-cli.yaml
```

**Custom location:**
```bash
seca --config /path/to/custom-config.yaml [command]
```

### Creating Configuration File

Copy the example configuration:

```bash
cp .seca-cli.yaml.example ~/.seca-cli.yaml
```

Or create manually:

```bash
cat > ~/.seca-cli.yaml << 'EOF'
# Custom results directory
results_dir: ~/security-testing
EOF
```

### Supported Settings

Currently, SECA-CLI supports the following configuration file settings:

#### `results_dir` (string)

Custom directory for storing engagement data and results.

**Default:** OS-appropriate data directory
- Linux/Unix: `~/.local/share/seca-cli/results/`
- macOS: `~/Library/Application Support/seca-cli/results/`
- Windows: `%LOCALAPPDATA%\seca-cli\results\`

**Example:**
```yaml
results_dir: /opt/security/seca-results
```

**Use cases:**
- Shared team directories
- Network storage locations
- Custom backup solutions
- Encrypted volumes

## Command-Line Flags

All operational settings are configured via command-line flags. These **always override** configuration file and environment variables.

### Global Flags

Available for all commands:

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | | Custom config file path | `~/.seca-cli.yaml` |
| `--operator` | `-o` | Operator name for audit trail | `$USER` env var |

**Example:**
```bash
seca --operator "John Doe" engagement list
```

### Check Command Flags

Available for `seca check http` and `seca check dns`:

#### Performance Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--concurrency` | `-c` | Max concurrent requests | `1` |
| `--rate` | `-r` | Requests per second (global) | `1` |
| `--timeout` | `-t` | Request timeout (seconds) | `10` |

**Example:**
```bash
seca check http --id 123 --concurrency 10 --rate 5 --timeout 30 \
  --roe-confirm https://example.com
```

#### Compliance Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--compliance-mode` | Enable compliance enforcement | `false` |
| `--retention-days` | Retention period for raw captures | `0` (required if using `--audit-append-raw`) |
| `--auto-sign` | Automatically sign .sha256 files | `false` |
| `--gpg-key` | GPG key ID or email for signing | `""` (required if `--auto-sign`) |
| `--audit-append-raw` | Save raw HTTP responses | `false` |

**Example:**
```bash
seca check http --id 123 --roe-confirm \
  --compliance-mode \
  --retention-days 90 \
  --auto-sign \
  --gpg-key "security@example.com" \
  --audit-append-raw \
  https://example.com
```

#### DNS-Specific Flags

Available only for `seca check dns`:

| Flag | Description | Default |
|------|-------------|---------|
| `--nameservers` | Custom DNS servers (comma-separated) | System default |
| `--dns-timeout` | DNS query timeout (seconds) | `10` |

**Example:**
```bash
seca check dns --id 123 --roe-confirm \
  --nameservers "8.8.8.8:53,1.1.1.1:53" \
  --dns-timeout 5 \
  example.com
```

## Environment Variables

### Operator Identity

SECA-CLI uses environment variables as a fallback for operator identity:

| Variable | Priority | Description |
|----------|----------|-------------|
| `USER` | 1 | Unix/Linux username |
| `LOGNAME` | 2 | Alternative username variable |

**Priority order:**
1. `--operator` flag (highest)
2. `$USER` environment variable
3. `$LOGNAME` environment variable (lowest)

**Setting operator via environment:**

```bash
# Temporary (current session)
export USER="John Doe"
seca engagement list

# Permanent (add to ~/.bashrc or ~/.zshrc)
echo 'export USER="John Doe"' >> ~/.bashrc
```

## Configuration Examples

### Example 1: Single User Development

**File:** `~/.seca-cli.yaml`
```yaml
results_dir: ~/security-testing
```

**Usage:**
```bash
# Uses config file for results_dir, $USER for operator
seca engagement create --name "Dev Test" --owner "Me" \
  --roe "Development testing" --roe-agree
```

### Example 2: Team Shared Storage

**File:** `~/.seca-cli.yaml`
```yaml
results_dir: /mnt/shared/security-engagements
```

**Usage:**
```bash
# Team member explicitly identifies themselves
seca --operator "Alice Smith" engagement list

# High-volume testing with custom settings
seca check http --id 123 --roe-confirm \
  --operator "Alice Smith" \
  --concurrency 10 \
  --rate 20 \
  --timeout 15 \
  -f targets.txt
```

### Example 3: Compliance Audit

**File:** `~/.seca-cli.yaml`
```yaml
results_dir: /secure/compliance-audits
```

**Usage:**
```bash
# Strict compliance mode with signing
seca check http --id 123 --roe-confirm \
  --operator "Compliance Team" \
  --compliance-mode \
  --retention-days 365 \
  --auto-sign \
  --gpg-key "compliance@example.com" \
  --audit-append-raw \
  --concurrency 1 \
  --rate 1 \
  https://example.com
```

### Example 4: Custom Config Location

**File:** `/opt/security/seca-config.yaml`
```yaml
results_dir: /opt/security/results
```

**Usage:**
```bash
# Specify config file explicitly
seca --config /opt/security/seca-config.yaml \
  --operator "Security Bot" \
  engagement list
```

## Verifying Configuration

### Check Current Settings

Use the `info` command to verify configuration:

```bash
seca info
```

**Output:**
```
SECA-CLI Information:
  Version:    1.2.0
  Operator:   khanhnv

Data Storage:
  Data Directory:    /home/khanhnv/.local/share/seca-cli
  Engagements File:  /home/khanhnv/.local/share/seca-cli/engagements.json (✓ exists)
  Results Directory: /home/khanhnv/.local/share/seca-cli/results (✓ exists)

Configuration:
  Config File: /home/khanhnv/.seca-cli.yaml (✓ found)
  Custom Config: No
...
```

### Test Configuration

1. **Create test engagement:**
```bash
seca engagement create \
  --name "Config Test" \
  --owner "Test User" \
  --roe "Configuration testing" \
  --roe-agree
```

2. **Check logs for operator attribution:**
```bash
# Look for: operator=<name> results_dir=<path>
```

3. **Verify results directory:**
```bash
ls -la ~/.local/share/seca-cli/results/
# or your custom results_dir
```

## Troubleshooting

### Issue: Config file not found

**Problem:** `seca info` shows "Config File: ~/.seca-cli.yaml (✗ not found)"

**Solution:**
1. Create config file: `cp .seca-cli.yaml.example ~/.seca-cli.yaml`
2. Or specify custom location: `seca --config /path/to/config.yaml`

### Issue: Operator identity required

**Problem:** `Error: operator identity is required (use --operator or set USER env)`

**Solution:**
```bash
# Option 1: Set via flag
seca --operator "Your Name" engagement list

# Option 2: Set via environment
export USER="Your Name"
seca engagement list

# Option 3: Add to shell profile
echo 'export USER="Your Name"' >> ~/.bashrc
```

### Issue: Results directory permission denied

**Problem:** `Error: failed to create results directory: permission denied`

**Solution:**
```bash
# Option 1: Use directory you have write access to
cat > ~/.seca-cli.yaml << 'EOF'
results_dir: ~/security-testing
EOF

# Option 2: Create directory with correct permissions
mkdir -p /opt/security/seca-results
chmod 755 /opt/security/seca-results

# Option 3: Use default OS data directory (recommended)
# Remove results_dir from config file
```

### Issue: Custom results_dir not working

**Problem:** SECA-CLI still uses default data directory

**Solution:**
1. Check YAML syntax (use spaces, not tabs)
2. Verify config file location: `seca info`
3. Use absolute paths or proper ~ expansion
4. Check file permissions: `ls -la ~/.seca-cli.yaml`

**Correct:**
```yaml
results_dir: /opt/security/results
```

**Incorrect:**
```yaml
results_dir:/opt/security/results  # Missing space after colon
	results_dir: /opt/security/results  # Tab instead of spaces
```

## Best Practices

### 1. Use OS Data Directories (Default)

For single-user scenarios, prefer the default OS data directories:

```yaml
# Leave results_dir empty or omit it
results_dir: ""
```

**Benefits:**
- Automatic backup with user data
- Proper permissions
- OS-appropriate location
- Multi-user support

### 2. Team Configuration

For shared team environments:

```yaml
results_dir: /mnt/shared/security-engagements
```

**Requirements:**
- Shared directory with write access for all team members
- Proper permission management (group write)
- Network storage with adequate performance
- Backup strategy for shared location

### 3. Compliance Configuration

For compliance-heavy environments, use command-line flags:

```bash
# Create alias or script
alias seca-compliance='seca \
  --operator "$(whoami)" \
  check http \
  --compliance-mode \
  --retention-days 365 \
  --auto-sign \
  --gpg-key "compliance@example.com"'

# Usage
seca-compliance --id 123 --roe-confirm https://example.com
```

### 4. Configuration Management

**Development:**
```bash
# .seca-cli.yaml
results_dir: ~/seca-dev
```

**Production:**
```bash
# /opt/security/seca-config.yaml
results_dir: /opt/security/seca-results
```

**Usage:**
```bash
# Development
seca engagement list

# Production
seca --config /opt/security/seca-config.yaml engagement list
```

### 5. Operator Attribution

Always ensure proper operator identification:

```bash
# Good: Explicit identification
seca --operator "John Doe <john@example.com>" engagement create ...

# Good: Environment variable
export USER="John Doe <john@example.com>"
seca engagement create ...

# Bad: Generic or missing identity
# (fails compliance requirements)
```

## Advanced Configuration

### Scripting and Automation

For automated testing, use environment variables and flags:

```bash
#!/bin/bash
# automated-security-check.sh

set -e

OPERATOR="Automated Security Bot"
CONFIG="/opt/security/seca-config.yaml"
ENGAGEMENT_ID="$1"

seca --config "$CONFIG" --operator "$OPERATOR" \
  check http --id "$ENGAGEMENT_ID" \
  --roe-confirm \
  --compliance-mode \
  --retention-days 90 \
  --concurrency 10 \
  --rate 20 \
  --timeout 30 \
  -f targets.txt
```

### CI/CD Integration

For continuous integration:

```yaml
# .github/workflows/security-check.yml
name: Security Check
on: [push]

jobs:
  security-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Configure SECA-CLI
        run: |
          cat > ~/.seca-cli.yaml << 'EOF'
          results_dir: ${{ github.workspace }}/results
          EOF

      - name: Run Security Checks
        run: |
          seca --operator "GitHub Actions" \
            check http --id ${{ secrets.ENGAGEMENT_ID }} \
            --roe-confirm \
            --concurrency 5 \
            -f targets.txt
```

## Related Documentation

- [README.md](../../README.md) - Complete project documentation
- [Compliance Guide](../operator-guide/compliance.md) - Compliance guidelines
- [Data Migration Guide](../reference/data-migration.md) - Data storage details
- [Operator Training Guide](../operator-guide/operator-training.md) - Operator training guide
- [.seca-cli.yaml.example](../../.seca-cli.yaml.example) - Example configuration file

## Future Configuration Options

The following settings may be added to the configuration file in future versions:

- Default concurrency level
- Default rate limit
- Default timeout
- Default DNS nameservers
- Default compliance mode settings
- Logging configuration
- Report format defaults
- GPG signing defaults

To request a new configuration option, please open an issue at:
https://github.com/khanhnv2901/seca-cli/issues
