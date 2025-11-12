# Plugin Development Guide

## Overview

SECA-CLI supports a powerful plugin architecture that allows you to extend the tool with custom security checkers without modifying the core codebase. Plugins are external programs that integrate seamlessly with SECA-CLI's engagement management, audit logging, and compliance features.

## Table of Contents

- [Plugin Architecture](#plugin-architecture)
- [Plugin Definition Format](#plugin-definition-format)
- [Plugin API Specification](#plugin-api-specification)
- [Creating Your First Plugin](#creating-your-first-plugin)
- [Plugin Output Format](#plugin-output-format)
- [Testing and Debugging](#testing-and-debugging)
- [Best Practices](#best-practices)
- [Example Plugins](#example-plugins)
- [Troubleshooting](#troubleshooting)

---

## Plugin Architecture

### How Plugins Work

1. **Discovery**: SECA-CLI loads plugin definitions from `~/.local/share/seca-cli/plugins/` (or `$XDG_DATA_HOME/seca-cli/plugins/`)
2. **Registration**: Each valid plugin JSON file creates a new `seca check <plugin-name>` command
3. **Execution**: When invoked, SECA-CLI executes the plugin's command with the target as an argument
4. **Integration**: Plugin results are captured, validated, and logged to the engagement's audit trail

### Plugin Lifecycle

```
[Plugin JSON Definition]
         ↓
[SECA-CLI Loads Plugin]
         ↓
[User Runs: seca check <plugin-name> --id <engagement> <target>]
         ↓
[SECA-CLI Executes: <command> <args...> <target>]
         ↓
[Plugin Returns JSON to stdout]
         ↓
[SECA-CLI Validates & Logs Results]
         ↓
[Results Added to Audit Trail]
```

### Plugin Location

**Linux/macOS:**
```
~/.local/share/seca-cli/plugins/
```

**With XDG_DATA_HOME set:**
```
$XDG_DATA_HOME/seca-cli/plugins/
```

Place your plugin definition files (`.json`) in this directory.

---

## Plugin Definition Format

### Basic Structure

Create a JSON file in the plugins directory with the following structure:

```json
{
  "name": "my-checker",
  "description": "Custom security checker for X",
  "command": "/path/to/checker-script",
  "args": ["--verbose", "--format", "json"],
  "env": {
    "CUSTOM_VAR": "value"
  },
  "timeout": 30,
  "results_filename": "my_checker_results.json",
  "api_version": 1
}
```

### Field Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | **Yes** | - | Unique plugin identifier (becomes command name) |
| `description` | string | No | "" | Brief description shown in help text |
| `command` | string | **Yes** | - | Absolute path to executable or command in PATH |
| `args` | []string | No | [] | Arguments passed before the target |
| `env` | map | No | {} | Additional environment variables |
| `timeout` | int | No | 10 | Timeout in seconds (0 = 10 seconds default) |
| `results_filename` | string | No | `<name>_results.json` | Filename for results output |
| `api_version` | int | No | 1 | Plugin API version (current: 1) |

### Validation Rules

- **Name**: Must be non-empty, alphanumeric with hyphens/underscores
- **Command**: Must be non-empty and executable
- **Timeout**: Must be positive integer (0 defaults to 10 seconds)
- **API Version**: Must match current version (1)

---

## Plugin API Specification

### API Version 1

**Current Version:** `1`

Plugins must declare `"api_version": 1` to ensure compatibility.

### Execution Contract

#### Input

Your plugin receives:
1. **Arguments**: Any args from `args` array, followed by the **target**
2. **Environment**: Inherited environment + custom `env` variables
3. **stdin**: Empty (not used)
4. **Context**: Runs with timeout specified in plugin definition

Example invocation:
```bash
# Plugin definition: {"command": "/usr/local/bin/my-checker", "args": ["--verbose"]}
# User command: seca check my-checker --id eng123 example.com

# SECA-CLI executes:
/usr/local/bin/my-checker --verbose example.com
```

#### Output

Your plugin **must**:
1. Write valid JSON to **stdout**
2. Exit with status code **0** for success
3. Return errors via **stderr** or JSON error field

#### Timeout Behavior

- Plugin killed after `timeout` seconds
- Partial output is captured
- Logged as error in audit trail

---

## Plugin Output Format

### Required JSON Structure

Your plugin must output a JSON object matching the `CheckResult` structure:

```json
{
  "target": "example.com",
  "checked_at": "2025-01-15T10:30:00Z",
  "status": "success",
  "http_status": 200,
  "tls_expiry": "2026-01-15T00:00:00Z",
  "notes": "All checks passed",
  "error": ""
}
```

### Field Specifications

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `target` | string | Auto-filled | Target URL/hostname (auto-populated if missing) |
| `checked_at` | string (RFC3339) | Auto-filled | Timestamp (auto-populated if missing) |
| `status` | string | **Yes** | One of: `success`, `warning`, `error`, `fail` |
| `http_status` | int | No | HTTP status code (if applicable) |
| `tls_expiry` | string (RFC3339) | No | TLS certificate expiry (if applicable) |
| `notes` | string | No | Human-readable findings |
| `error` | string | No | Error message (if status = error/fail) |

### Status Values

| Status | Meaning | Use Case |
|--------|---------|----------|
| `success` | Check passed | Target is secure/compliant |
| `warning` | Minor issue | Non-critical finding |
| `error` | Check failed to run | Network error, timeout, invalid target |
| `fail` | Security failure | Vulnerability found, non-compliant |

### Example Outputs

#### Success Case
```json
{
  "status": "success",
  "http_status": 200,
  "notes": "Port 443 open, valid TLS certificate, secure headers present",
  "tls_expiry": "2026-06-01T00:00:00Z"
}
```

#### Failure Case
```json
{
  "status": "fail",
  "http_status": 0,
  "notes": "Port 8080 exposed without authentication",
  "error": ""
}
```

#### Error Case
```json
{
  "status": "error",
  "http_status": 0,
  "notes": "",
  "error": "connection timeout after 30s"
}
```

---

## Creating Your First Plugin

### Example 1: Simple Port Scanner Plugin

**File:** `~/.local/share/seca-cli/plugins/port-scan.json`

```json
{
  "name": "port-scan",
  "description": "Scan common ports on target",
  "command": "/usr/local/bin/port-scanner.sh",
  "args": [],
  "timeout": 60,
  "api_version": 1
}
```

**Script:** `/usr/local/bin/port-scanner.sh`

```bash
#!/bin/bash
set -euo pipefail

TARGET="$1"
PORTS="22,80,443,8080,8443"

# Run scan
RESULTS=$(nmap -p "$PORTS" --open -oG - "$TARGET" 2>&1)

# Parse results
if echo "$RESULTS" | grep -q "open"; then
    STATUS="warning"
    NOTES="Open ports detected: $(echo "$RESULTS" | grep -oP '\d+/open' | tr '\n' ' ')"
else
    STATUS="success"
    NOTES="No unexpected open ports found"
fi

# Output JSON
cat <<EOF
{
  "status": "$STATUS",
  "notes": "$NOTES"
}
EOF
```

**Make executable:**
```bash
chmod +x /usr/local/bin/port-scanner.sh
```

**Usage:**
```bash
seca check port-scan --id eng123 example.com
```

---

### Example 2: Python API Security Checker

**File:** `~/.local/share/seca-cli/plugins/api-security.json`

```json
{
  "name": "api-security",
  "description": "Check API security best practices",
  "command": "python3",
  "args": ["/usr/local/bin/api_checker.py"],
  "env": {
    "API_KEY": "your-api-key-here"
  },
  "timeout": 30,
  "results_filename": "api_security_results.json",
  "api_version": 1
}
```

**Script:** `/usr/local/bin/api_checker.py`

```python
#!/usr/bin/env python3
import sys
import json
import requests
from datetime import datetime

def check_api_security(target):
    try:
        # Perform checks
        response = requests.get(f"https://{target}/api/health", timeout=10)

        issues = []
        if 'X-Frame-Options' not in response.headers:
            issues.append("Missing X-Frame-Options header")
        if 'X-Content-Type-Options' not in response.headers:
            issues.append("Missing X-Content-Type-Options header")

        if issues:
            status = "warning"
            notes = "Security issues: " + "; ".join(issues)
        else:
            status = "success"
            notes = "All API security headers present"

        return {
            "status": status,
            "http_status": response.status_code,
            "notes": notes,
            "error": ""
        }

    except requests.Timeout:
        return {
            "status": "error",
            "notes": "",
            "error": "Request timeout"
        }
    except Exception as e:
        return {
            "status": "error",
            "notes": "",
            "error": str(e)
        }

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(json.dumps({"status": "error", "error": "No target specified"}))
        sys.exit(1)

    target = sys.argv[1]
    result = check_api_security(target)
    print(json.dumps(result, indent=2))
```

**Usage:**
```bash
seca check api-security --id eng123 api.example.com
```

---

### Example 3: Go-based TLS Version Checker

**File:** `~/.local/share/seca-cli/plugins/tls-version.json`

```json
{
  "name": "tls-version",
  "description": "Verify minimum TLS version (TLS 1.2+)",
  "command": "/usr/local/bin/tls-checker",
  "args": ["--min-version", "1.2"],
  "timeout": 20,
  "api_version": 1
}
```

**Go code:** `tls-checker.go`

```go
package main

import (
    "crypto/tls"
    "encoding/json"
    "fmt"
    "net"
    "os"
    "time"
)

type CheckResult struct {
    Status    string    `json:"status"`
    HTTPStatus int      `json:"http_status,omitempty"`
    TLSExpiry string    `json:"tls_expiry,omitempty"`
    Notes     string    `json:"notes"`
    Error     string    `json:"error,omitempty"`
}

func main() {
    if len(os.Args) < 2 {
        outputError("No target specified")
        return
    }

    target := os.Args[1]
    result := checkTLS(target)

    output, _ := json.MarshalIndent(result, "", "  ")
    fmt.Println(string(output))
}

func checkTLS(host string) CheckResult {
    conn, err := tls.Dial("tcp", net.JoinHostPort(host, "443"), &tls.Config{
        MinVersion: tls.VersionTLS12,
    })
    if err != nil {
        return CheckResult{
            Status: "error",
            Error:  err.Error(),
        }
    }
    defer conn.Close()

    state := conn.ConnectionState()
    cert := state.PeerCertificates[0]

    tlsVersion := "Unknown"
    switch state.Version {
    case tls.VersionTLS10:
        tlsVersion = "TLS 1.0 (INSECURE)"
    case tls.VersionTLS11:
        tlsVersion = "TLS 1.1 (INSECURE)"
    case tls.VersionTLS12:
        tlsVersion = "TLS 1.2"
    case tls.VersionTLS13:
        tlsVersion = "TLS 1.3"
    }

    status := "success"
    if state.Version < tls.VersionTLS12 {
        status = "fail"
    }

    return CheckResult{
        Status:    status,
        TLSExpiry: cert.NotAfter.Format(time.RFC3339),
        Notes:     fmt.Sprintf("TLS Version: %s, Cipher: %s", tlsVersion, tls.CipherSuiteName(state.CipherSuite)),
    }
}

func outputError(msg string) {
    result := CheckResult{Status: "error", Error: msg}
    output, _ := json.Marshal(result)
    fmt.Println(string(output))
}
```

**Build and install:**
```bash
go build -o /usr/local/bin/tls-checker tls-checker.go
```

---

## Testing and Debugging

### Manual Testing

Test your plugin independently before registering with SECA-CLI:

```bash
# Test with direct invocation
/path/to/your/plugin example.com

# Expected: Valid JSON output
# {"status": "success", "notes": "..."}
```

### Validation Checklist

- [ ] Plugin outputs valid JSON to stdout
- [ ] JSON matches CheckResult schema
- [ ] Exit code is 0 on success
- [ ] Errors go to stderr or JSON `error` field
- [ ] Execution completes within timeout
- [ ] Works with various target formats (IP, hostname, URL)

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Plugin not found | JSON file not in plugins dir | Check `~/.local/share/seca-cli/plugins/` |
| Command not executed | Missing executable permissions | `chmod +x /path/to/script` |
| Invalid output error | Non-JSON stdout | Ensure only JSON goes to stdout |
| Timeout errors | Plugin takes too long | Increase `timeout` value |
| API version mismatch | Wrong API version | Set `"api_version": 1` |

### Debug Mode

Run SECA-CLI with verbose output:

```bash
# See plugin loading warnings
seca check port-scan --id eng123 example.com 2>&1 | grep -i plugin

# Test plugin execution manually
TARGET="example.com"
/path/to/plugin "$TARGET"
```

---

## Best Practices

### Security

1. **Validate Inputs**: Always sanitize the target parameter
   ```bash
   # Bad: Command injection vulnerability
   curl "http://$TARGET"

   # Good: Validate target format
   if [[ ! "$TARGET" =~ ^[a-zA-Z0-9.-]+$ ]]; then
       echo '{"status":"error","error":"Invalid target format"}'
       exit 1
   fi
   ```

2. **Use Absolute Paths**: Avoid relying on PATH for security-sensitive operations
3. **Limit Permissions**: Run plugins with minimal privileges
4. **Handle Secrets**: Use environment variables for API keys, never hardcode

### Performance

1. **Set Reasonable Timeouts**: Balance thoroughness with speed
2. **Fail Fast**: Return errors quickly rather than hanging
3. **Batch Operations**: Group multiple checks when possible
4. **Cache Results**: Avoid redundant checks (use engagement data)

### Reliability

1. **Error Handling**: Always catch and return errors gracefully
2. **Idempotency**: Plugin should produce same result for same input
3. **Logging**: Use stderr for debug logs (not stdout)
4. **Graceful Degradation**: Return partial results if some checks fail

### Compliance

1. **Audit Trail**: Include detailed notes for compliance reporting
2. **Attribution**: Plugin actions are logged to operator in audit CSV
3. **Evidence**: Use `notes` field for detailed findings
4. **Timestamps**: Let SECA-CLI auto-populate timestamps for accuracy

---

## Advanced Topics

### Multi-Target Plugins

SECA-CLI invokes plugins once per target. For efficiency, consider:

```bash
# Plugin receives ONE target at a time
# SECA-CLI handles parallelization with --concurrency flag

# User runs:
seca check my-plugin --id eng123 host1.com host2.com host3.com --concurrency 3

# SECA-CLI executes 3 parallel instances:
# my-plugin host1.com
# my-plugin host2.com
# my-plugin host3.com
```

### Using External Tools

Plugins can wrap existing security tools:

```bash
#!/bin/bash
# Wrapper for nikto vulnerability scanner

TARGET="$1"
OUTPUT=$(nikto -h "$TARGET" -Format json 2>&1)

# Parse nikto JSON and convert to SECA format
STATUS="success"
NOTES=$(echo "$OUTPUT" | jq -r '.vulnerabilities | length')

cat <<EOF
{
  "status": "$STATUS",
  "notes": "Nikto found $NOTES potential issues"
}
EOF
```

### Environment Variables

Available to plugins:

- Standard OS environment
- Custom `env` from plugin definition
- Target passed as command argument

```python
import os
api_key = os.environ.get('API_KEY', '')
custom_var = os.environ.get('CUSTOM_VAR', 'default')
```

---

## Troubleshooting

### Plugin Not Loading

**Check plugin directory:**
```bash
ls -la ~/.local/share/seca-cli/plugins/
```

**Validate JSON syntax:**
```bash
jq . ~/.local/share/seca-cli/plugins/my-plugin.json
```

**Check for warnings:**
```bash
seca check --help 2>&1 | grep -i warning
```

### Plugin Execution Fails

**Test manually:**
```bash
/path/to/plugin example.com
```

**Check permissions:**
```bash
ls -l /path/to/plugin
# Should show: -rwxr-xr-x
```

**Verify output format:**
```bash
/path/to/plugin example.com | jq .
# Should parse without errors
```

### Timeout Issues

**Increase timeout in plugin definition:**
```json
{
  "timeout": 120
}
```

**Optimize plugin performance:**
- Reduce network retries
- Remove unnecessary sleeps
- Parallelize internal operations

---

## Plugin Distribution

### Sharing Plugins

1. Package plugin definition + script together
2. Document dependencies (Python packages, Go version, etc.)
3. Include installation instructions
4. Provide test cases

### Example Plugin Package

```
my-security-plugin/
├── README.md
├── plugin.json
├── checker.py
├── requirements.txt
└── tests/
    └── test_checker.py
```

**Installation:**
```bash
# Install dependencies
pip install -r requirements.txt

# Copy files
cp plugin.json ~/.local/share/seca-cli/plugins/my-security.json
cp checker.py /usr/local/bin/my-security-checker
chmod +x /usr/local/bin/my-security-checker

# Test
seca check my-security --id test-eng example.com
```

---

## Plugin API Compatibility

### Version History

| API Version | SECA-CLI Version | Changes |
|-------------|------------------|---------|
| 1 | v1.0.0+ | Initial plugin API |

### Future Compatibility

- SECA-CLI will maintain backward compatibility for API version 1
- New API versions will be introduced for breaking changes
- Deprecated features will have 6-month sunset period

---

## Getting Help

- **Issues**: Report plugin issues at https://github.com/khanhnv2901/seca-cli/issues
- **Examples**: See `examples/plugins/` directory in repository
- **Community**: Share plugins in Discussions tab

---

## Summary

SECA-CLI plugins extend the tool's capabilities while leveraging its core compliance, audit, and engagement management features. Follow this guide to create robust, secure, and compliant custom checkers.

**Quick Start Checklist:**
- [ ] Create plugin JSON in `~/.local/share/seca-cli/plugins/`
- [ ] Implement checker script that outputs JSON to stdout
- [ ] Set correct permissions (`chmod +x`)
- [ ] Test manually before using with engagements
- [ ] Verify integration with `seca check <plugin-name> --help`

Happy plugin development!
