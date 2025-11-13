# Troubleshooting and FAQ

Common issues, solutions, and frequently asked questions for SECA-CLI.

## Table of Contents

- [Installation Issues](#installation-issues)
- [Engagement Management](#engagement-management)
- [Check Command Errors](#check-command-errors)
- [Plugin Issues](#plugin-issues)
- [GPG and Signing Issues](#gpg-and-signing-issues)
- [Performance Problems](#performance-problems)
- [Data and File System](#data-and-file-system)
- [Network and Connectivity](#network-and-connectivity)
- [Compliance Mode Issues](#compliance-mode-issues)
- [FAQ](#faq)

---

## Installation Issues

### Binary Not Found After Installation

**Problem:** `command not found: seca` after installation

**Solutions:**

1. **Check installation location:**
   ```bash
   which seca
   # If empty, binary not in PATH
   ```

2. **Add to PATH (Linux/macOS):**
   ```bash
   # Add to ~/.bashrc or ~/.zshrc
   export PATH="$PATH:/usr/local/bin"

   # Reload shell
   source ~/.bashrc
   ```

3. **Verify binary exists:**
   ```bash
   ls -l /usr/local/bin/seca
   # Should show executable permissions
   ```

4. **Fix permissions:**
   ```bash
   chmod +x /usr/local/bin/seca
   ```

---

### Permission Denied Errors

**Problem:** `permission denied` when running seca

**Solution:**

```bash
# Check current permissions
ls -l $(which seca)

# Fix permissions
sudo chmod +x /usr/local/bin/seca

# Or reinstall with correct permissions
sudo cp seca /usr/local/bin/
sudo chmod 755 /usr/local/bin/seca
```

---

### Version Mismatch or Build Errors

**Problem:** `seca version` shows unexpected version or build date

**Solution:**

```bash
# Clean old versions
rm -f /usr/local/bin/seca

# Download latest release
wget https://github.com/khanhnv2901/seca-cli/releases/latest/download/seca-linux-amd64

# Install
sudo mv seca-linux-amd64 /usr/local/bin/seca
sudo chmod +x /usr/local/bin/seca

# Verify
seca version
```

---

## Engagement Management

### Engagement Already Exists

**Problem:** `engagement already exists: eng123`

**Solutions:**

1. **Use a different ID:**
   ```bash
   seca engagement create --id eng123-v2 --client "ACME Corp" --start-date 2025-01-15
   ```

2. **Delete existing engagement:**
   ```bash
   seca engagement delete --id eng123
   ```

3. **View existing engagement:**
   ```bash
   seca engagement view --id eng123
   # Verify it's safe to delete before recreating
   ```

---

### Engagement Not Found

**Problem:** `engagement not found: unknown-id`

**Solutions:**

1. **List all engagements:**
   ```bash
   seca engagement list
   ```

2. **Check engagement ID spelling:**
   - IDs are case-sensitive
   - Verify exact ID from list output

3. **Check data directory:**
   ```bash
   ls ~/.local/share/seca-cli/engagements/
   # Verify .json file exists
   ```

4. **Verify custom data directory (if configured):**
   ```bash
   cat ~/.seca-cli.yaml | grep results_dir
   # Check configured path
   ```

---

### Cannot Add Scope to Engagement

**Problem:** `failed to add scope: <error>`

**Solutions:**

1. **Verify engagement exists:**
   ```bash
   seca engagement view --id eng123
   ```

2. **Check target format:**
   ```bash
   # Valid formats
   seca engagement add-scope --id eng123 example.com
   seca engagement add-scope --id eng123 192.168.1.100
   seca engagement add-scope --id eng123 "*.example.com"
   seca engagement add-scope --id eng123 "10.0.0.0/24"
   ```

3. **Quote wildcards and special characters:**
   ```bash
   # Correct
   seca engagement add-scope --id eng123 "*.example.com"

   # Incorrect (shell expansion)
   seca engagement add-scope --id eng123 *.example.com
   ```

---

## Check Command Errors

### ROE Not Confirmed

**Problem:** `Rules of Engagement (ROE) not confirmed`

**Solution:**

Add `--roe-confirm` flag to acknowledge authorization:

```bash
seca check http --id eng123 --roe-confirm example.com
```

**Important:** Only use `--roe-confirm` if you have explicit written authorization to test the target.

---

### Engagement ID Required

**Problem:** `--id flag is required`

**Solution:**

Always specify an engagement ID for check commands:

```bash
seca check http --id eng123 --roe-confirm example.com
```

---

### Connection Timeout

**Problem:** `error: connection timeout after 10s`

**Solutions:**

1. **Increase timeout:**
   ```bash
   seca check http --id eng123 --roe-confirm --timeout 60 slow-server.com
   ```

2. **Check target reachability:**
   ```bash
   ping slow-server.com
   curl -I https://slow-server.com
   ```

3. **Verify firewall/network:**
   - Ensure target is accessible from your network
   - Check for VPN or proxy requirements

4. **Use retry mechanism:**
   ```bash
   seca check http --id eng123 --roe-confirm --retry 3 --timeout 30 flaky-server.com
   ```

---

### TLS/SSL Certificate Errors

**Problem:** `x509: certificate signed by unknown authority`

**Solutions:**

1. **Check certificate validity:**
   ```bash
   openssl s_client -connect example.com:443 -servername example.com
   ```

2. **Update system CA certificates:**
   ```bash
   # Ubuntu/Debian
   sudo update-ca-certificates

   # RHEL/CentOS
   sudo update-ca-trust
   ```

3. **Expected behavior:**
   - SECA-CLI reports invalid certificates as findings (not errors)
   - This is correct behavior for security testing

---

### DNS Resolution Failed

**Problem:** `no such host` or `DNS resolution failed`

**Solutions:**

1. **Verify DNS resolution:**
   ```bash
   nslookup example.com
   dig example.com
   ```

2. **Use custom nameserver:**
   ```bash
   seca check dns --id eng123 --roe-confirm --nameservers 8.8.8.8:53 example.com
   ```

3. **Check /etc/resolv.conf:**
   ```bash
   cat /etc/resolv.conf
   # Verify nameserver configuration
   ```

4. **For internal domains:**
   ```bash
   # Use internal DNS server
   seca check dns --id eng123 --roe-confirm --nameservers 192.168.1.1:53 internal.corp
   ```

---

### Rate Limiting or 429 Errors

**Problem:** `HTTP 429: Too Many Requests`

**Solutions:**

1. **Reduce concurrency:**
   ```bash
   seca check http --id eng123 --roe-confirm --concurrency 1 example.com
   ```

2. **Reduce rate limit:**
   ```bash
   seca check http --id eng123 --roe-confirm --rate 1 --concurrency 5 example.com
   ```

3. **Increase timeout:**
   ```bash
   seca check http --id eng123 --roe-confirm --timeout 30 example.com
   ```

4. **Add delays between batches:**
   ```bash
   # Check 50 hosts, wait 60s, check next 50
   seca check http --id eng123 --roe-confirm hosts1-50.txt
   sleep 60
   seca check http --id eng123 --roe-confirm hosts51-100.txt
   ```

---

## Plugin Issues

### Plugin Not Loading

**Problem:** Plugin command not available after adding JSON definition

**Solutions:**

1. **Check plugin directory:**
   ```bash
   ls -la ~/.local/share/seca-cli/plugins/
   # Verify .json file exists
   ```

2. **Validate JSON syntax:**
   ```bash
   jq . ~/.local/share/seca-cli/plugins/my-plugin.json
   # Should parse without errors
   ```

3. **Check for loading warnings:**
   ```bash
   seca check --help 2>&1 | grep -i warning
   # Look for plugin-related warnings
   ```

4. **Verify required fields:**
   ```json
   {
     "name": "my-checker",
     "command": "/path/to/script",
     "api_version": 1
   }
   ```

5. **Restart shell:**
   ```bash
   # Plugin commands are loaded on startup
   exec $SHELL
   ```

---

### Plugin Execution Failed

**Problem:** `plugin execution failed: <error>`

**Solutions:**

1. **Test plugin manually:**
   ```bash
   /path/to/plugin example.com
   # Should output valid JSON
   ```

2. **Check executable permissions:**
   ```bash
   ls -l /path/to/plugin
   # Should show: -rwxr-xr-x

   chmod +x /path/to/plugin
   ```

3. **Verify JSON output:**
   ```bash
   /path/to/plugin example.com | jq .
   # Should parse without errors
   ```

4. **Check for errors on stderr:**
   ```bash
   /path/to/plugin example.com 2>&1
   # Look for error messages
   ```

5. **Verify command in PATH:**
   ```bash
   which python3  # If plugin uses python3
   which node     # If plugin uses node
   ```

---

### Plugin Timeout

**Problem:** `plugin timeout after 10s`

**Solutions:**

1. **Increase timeout in plugin definition:**
   ```json
   {
     "name": "slow-checker",
     "command": "/path/to/script",
     "timeout": 120,
     "api_version": 1
   }
   ```

2. **Optimize plugin performance:**
   - Remove unnecessary sleeps
   - Reduce network retries
   - Parallelize internal operations

---

### Invalid Plugin Output

**Problem:** `invalid plugin output: unexpected end of JSON input`

**Solutions:**

1. **Ensure only JSON goes to stdout:**
   ```bash
   # Bad: mixes debug output with JSON
   echo "Checking target..."
   echo '{"status":"success"}'

   # Good: debug to stderr, JSON to stdout
   echo "Checking target..." >&2
   echo '{"status":"success"}'
   ```

2. **Validate JSON format:**
   ```bash
   /path/to/plugin example.com | jq .
   # Should show valid JSON
   ```

3. **Check required fields:**
   ```json
   {
     "status": "success",
     "notes": "Check passed"
   }
   ```

---

## GPG and Signing Issues

### GPG Key Not Found

**Problem:** `gpg: no default secret key`

**Solutions:**

1. **List available keys:**
   ```bash
   gpg --list-secret-keys
   ```

2. **Generate new key:**
   ```bash
   gpg --full-generate-key
   # Follow prompts
   ```

3. **Specify key explicitly:**
   ```bash
   seca check http --id eng123 --roe-confirm \
     --auto-sign \
     --gpg-key alice@security.com \
     example.com
   ```

---

### GPG Signing Failed

**Problem:** `gpg: signing failed: <error>`

**Solutions:**

1. **Check key expiry:**
   ```bash
   gpg --list-keys alice@security.com
   # Look for "expired" status
   ```

2. **Verify passphrase:**
   ```bash
   # Test signing manually
   echo "test" | gpg --clear-sign
   ```

3. **Use GPG agent:**
   ```bash
   # Start GPG agent
   eval $(gpg-agent --daemon)

   # Add to ~/.bashrc for persistence
   export GPG_TTY=$(tty)
   ```

4. **Check permissions:**
   ```bash
   # GPG home directory
   chmod 700 ~/.gnupg
   chmod 600 ~/.gnupg/*
   ```

---

### Cannot Decrypt Results

**Problem:** `gpg: decryption failed: No secret key`

**Solutions:**

1. **Verify recipient:**
   ```bash
   gpg --list-packets http_results.json.gpg
   # Check recipient key ID
   ```

2. **Import private key:**
   ```bash
   gpg --import private-key.asc
   ```

3. **Check key availability:**
   ```bash
   gpg --list-secret-keys
   # Verify decryption key is present
   ```

---

## Performance Problems

### Slow Check Execution

**Problem:** Checks taking too long to complete

**Solutions:**

1. **Increase concurrency:**
   ```bash
   seca check http --id eng123 --roe-confirm \
     --concurrency 50 \
     --targets-file large-list.txt
   ```

2. **Reduce timeout:**
   ```bash
   # For fast networks
   seca check http --id eng123 --roe-confirm --timeout 5 example.com
   ```

3. **Use progress display:**
   ```bash
   seca check http --id eng123 --roe-confirm --progress --targets-file hosts.txt
   ```

4. **Profile network:**
   ```bash
   # Check average response time
   time curl -I https://example.com
   ```

5. **Balance rate limiting:**
   ```bash
   # High throughput
   seca check http --id eng123 --roe-confirm \
     --concurrency 100 \
     --rate 200 \
     --targets-file hosts.txt
   ```

---

### High Memory Usage

**Problem:** SECA-CLI consuming excessive memory

**Solutions:**

1. **Reduce concurrency:**
   ```bash
   seca check http --id eng123 --roe-confirm --concurrency 10 example.com
   ```

2. **Process targets in batches:**
   ```bash
   # Split large file
   split -l 100 large-list.txt batch-

   # Process each batch
   for batch in batch-*; do
     seca check http --id eng123 --roe-confirm --targets-file $batch
   done
   ```

3. **Disable raw capture:**
   ```bash
   # Remove --audit-append-raw flag
   seca check http --id eng123 --roe-confirm example.com
   ```

---

### Disk Space Issues

**Problem:** `no space left on device`

**Solutions:**

1. **Check disk usage:**
   ```bash
   df -h ~/.local/share/seca-cli
   ```

2. **Clean old engagements:**
   ```bash
   seca engagement delete --id old-engagement-1
   seca engagement delete --id old-engagement-2
   ```

3. **Archive and remove results:**
   ```bash
   tar czf archive-2024.tar.gz ~/.local/share/seca-cli/results/2024-*
   rm -rf ~/.local/share/seca-cli/results/2024-*
   ```

4. **Configure retention:**
   ```yaml
   # ~/.seca-cli.yaml
   compliance:
     retention_days: 90
   ```

---

## Data and File System

### Results Directory Not Found

**Problem:** `results directory does not exist`

**Solutions:**

1. **Create data directory:**
   ```bash
   mkdir -p ~/.local/share/seca-cli/{engagements,results,telemetry,plugins}
   ```

2. **Check permissions:**
   ```bash
   ls -ld ~/.local/share/seca-cli
   # Should show: drwx------

   chmod 700 ~/.local/share/seca-cli
   ```

3. **Verify custom data directory:**
   ```bash
   cat ~/.seca-cli.yaml | grep results_dir
   # Ensure directory exists
   ```

---

### Corrupted Engagement File

**Problem:** `failed to parse engagement: invalid JSON`

**Solutions:**

1. **Validate JSON:**
   ```bash
   jq . ~/.local/share/seca-cli/engagements/eng123.json
   ```

2. **Restore from backup:**
   ```bash
   cp ~/.local/share/seca-cli/engagements/eng123.json.backup \
      ~/.local/share/seca-cli/engagements/eng123.json
   ```

3. **Manually edit:**
   ```bash
   vim ~/.local/share/seca-cli/engagements/eng123.json
   # Fix JSON syntax errors
   ```

4. **Recreate engagement:**
   ```bash
   mv ~/.local/share/seca-cli/engagements/eng123.json \
      ~/.local/share/seca-cli/engagements/eng123.json.broken

   seca engagement create --id eng123 --client "ACME Corp" --start-date 2025-01-15
   ```

---

### Hash Verification Failed

**Problem:** `hash mismatch: expected X, got Y`

**Solutions:**

1. **File was modified:**
   - This indicates tampering or corruption
   - Investigate changes
   - Regenerate results if needed

2. **Verify hash algorithm:**
   ```bash
   # Use correct algorithm
   sha256sum audit.csv  # If --hash sha256
   sha512sum audit.csv  # If --hash sha512
   ```

3. **Check file integrity:**
   ```bash
   # Compare with backup
   diff audit.csv audit.csv.backup
   ```

---

## Network and Connectivity

### Proxy Issues

**Problem:** Cannot connect through corporate proxy

**Solutions:**

1. **Set HTTP proxy:**
   ```bash
   export HTTP_PROXY=http://proxy.company.com:8080
   export HTTPS_PROXY=http://proxy.company.com:8080
   export NO_PROXY=localhost,127.0.0.1

   seca check http --id eng123 --roe-confirm example.com
   ```

2. **Verify proxy settings:**
   ```bash
   curl -I https://example.com
   # Should work with same proxy config
   ```

---

### VPN Required

**Problem:** Targets only accessible via VPN

**Solutions:**

1. **Connect VPN first:**
   ```bash
   # Connect VPN
   sudo openvpn client.ovpn

   # Then run checks
   seca check http --id eng123 --roe-confirm internal.corp
   ```

2. **Verify VPN connectivity:**
   ```bash
   ping internal.corp
   ```

---

### IPv6 Issues

**Problem:** IPv6 targets not resolving or connecting

**Solutions:**

1. **Check IPv6 support:**
   ```bash
   ping6 ipv6.google.com
   ```

2. **Force IPv4:**
   ```bash
   # Use A records only (no AAAA)
   seca check dns --id eng123 --roe-confirm example.com
   ```

3. **Enable IPv6:**
   ```bash
   # Linux: Check IPv6 enabled
   sysctl net.ipv6.conf.all.disable_ipv6
   # Should be: 0 (enabled)
   ```

---

## Compliance Mode Issues

### Retention Days Required

**Problem:** `--retention-days required in compliance mode`

**Solution:**

Specify retention period when using `--audit-append-raw`:

```bash
seca check http --id eng123 --roe-confirm \
  --compliance-mode \
  --audit-append-raw \
  --retention-days 2555 \
  example.com
```

---

### Hash Algorithm Not Allowed

**Problem:** `SHA-256 not allowed in compliance mode for this engagement`

**Solution:**

Use SHA-512 for high-security engagements:

```bash
seca check http --id eng123 --roe-confirm \
  --compliance-mode \
  --hash sha512 \
  example.com
```

---

### Evidence Encryption Failed

**Problem:** `failed to encrypt results: <error>`

**Solutions:**

1. **Check GPG key:**
   ```bash
   gpg --list-keys
   ```

2. **Specify recipient:**
   ```bash
   export GPG_RECIPIENT=alice@security.com
   seca check http --id eng123 --roe-confirm --secure-results example.com
   ```

3. **Test encryption:**
   ```bash
   echo "test" | gpg --encrypt --recipient alice@security.com
   ```

---

## FAQ

### Q: Can I run multiple checks in parallel?

**A:** Yes, use multiple terminal sessions or background jobs:

```bash
# Terminal 1
seca check http --id eng123 --roe-confirm --targets-file batch1.txt &

# Terminal 2
seca check dns --id eng123 --roe-confirm --targets-file batch2.txt &

# Wait for both
wait
```

---

### Q: How do I skip already-checked targets?

**A:** SECA-CLI appends to audit logs. To avoid duplicates:

1. **Manual filtering:**
   ```bash
   # Extract checked targets from audit CSV
   awk -F',' '{print $5}' audit.csv | sort -u > checked.txt

   # Remove from target list
   comm -23 <(sort all-targets.txt) <(sort checked.txt) > remaining.txt

   # Check remaining
   seca check http --id eng123 --roe-confirm --targets-file remaining.txt
   ```

2. **Use different engagement IDs:**
   ```bash
   seca check http --id eng123-batch1 --roe-confirm --targets-file batch1.txt
   seca check http --id eng123-batch2 --roe-confirm --targets-file batch2.txt
   ```

---

### Q: Can I cancel a running check?

**A:** Yes, press `Ctrl-C` for graceful cancellation:

```bash
seca check http --id eng123 --roe-confirm --targets-file large-list.txt
# Press Ctrl-C
# Partial results saved
```

Press `Ctrl-C` twice to force quit (no results saved).

---

### Q: How do I resume after Ctrl-C?

**A:** Re-run with remaining targets:

```bash
# Initial run (cancelled)
seca check http --id eng123 --roe-confirm --targets-file all.txt
^C

# Extract checked targets
awk -F',' '{print $5}' ~/.local/share/seca-cli/results/eng123/audit.csv > checked.txt

# Create remaining list
comm -23 <(sort all.txt) <(sort checked.txt) > remaining.txt

# Resume
seca check http --id eng123 --roe-confirm --targets-file remaining.txt
```

---

### Q: What's the difference between `--concurrency` and `--rate`?

**A:**

- **`--concurrency`**: Max parallel requests (e.g., 10 simultaneous connections)
- **`--rate`**: Max requests per second (e.g., 50 req/s globally)

Example:
```bash
# 100 parallel connections, limited to 200 req/s total
seca check http --id eng123 --roe-confirm \
  --concurrency 100 \
  --rate 200 \
  --targets-file hosts.txt
```

---

### Q: How do I test SECA-CLI safely?

**A:** Use test domains you own or public test services:

```bash
# Your own domain
seca check http --id test --roe-confirm myowndomain.com

# Public test services
seca check http --id test --roe-confirm example.com
seca check http --id test --roe-confirm httpbin.org
```

**Never** test targets without authorization.

---

### Q: Can I use SECA-CLI in CI/CD?

**A:** Yes, designed for automation:

```bash
# CI/CD script
set -euo pipefail

# Create engagement
seca engagement create --id ci-scan-$(date +%Y%m%d) \
  --client "CI/CD" \
  --start-date $(date +%Y-%m-%d)

# Run checks
seca check http --id ci-scan-$(date +%Y%m%d) --roe-confirm \
  --targets-file targets.txt \
  --concurrency 50 \
  --timeout 30

# Generate report
seca report generate --id ci-scan-$(date +%Y%m%d) --format json > report.json

# Check exit code
if [ $? -eq 0 ]; then
  echo "Scan completed successfully"
else
  echo "Scan failed"
  exit 1
fi
```

---

### Q: How do I integrate with Jira/ticketing?

**A:** Export results as JSON and parse:

```bash
# Generate JSON report
seca report stats --id eng123 --format json > stats.json

# Parse and create tickets
jq -r '.failures[] | "Title: \(.target)\nDescription: \(.notes)"' stats.json | \
while IFS= read -r title && IFS= read -r description; do
  # Create Jira ticket (using Jira CLI or API)
  jira issue create --project SEC --type Bug --summary "$title" --description "$description"
done
```

---

### Q: Where are my results stored?

**A:**

Default locations:
- **Linux/Unix:** `~/.local/share/seca-cli/`
- **macOS:** `~/Library/Application Support/seca-cli/`
- **Windows:** `%LOCALAPPDATA%\seca-cli\`

Check with:
```bash
seca info
```

---

### Q: How do I migrate data to a new directory?

**A:** See [Data Migration Guide](data-migration.md):

```bash
# Copy all data
cp -r ~/.local/share/seca-cli /new/path/seca-cli-data

# Update config
cat > ~/.seca-cli.yaml <<EOF
results_dir: /new/path/seca-cli-data
EOF

# Verify
seca info
```

---

## Getting Additional Help

### Support Channels

- **GitHub Issues:** https://github.com/khanhnv2901/seca-cli/issues
- **Documentation:** `docs/` directory
- **Built-in Help:** `seca --help`, `seca [command] --help`

### Reporting Bugs

Include the following information:

1. **Version:** `seca version`
2. **System:** `uname -a`
3. **Command:** Full command that failed
4. **Error:** Complete error message
5. **Logs:** Relevant output or stack trace

### Feature Requests

Submit feature requests on GitHub with:

- Use case description
- Expected behavior
- Example commands or workflows
- Compliance/security justification (if applicable)

---

## Summary

Most issues can be resolved by:

1. **Verifying permissions** (file system, GPG, executable)
2. **Checking configuration** (engagement ID, data directory, custom settings)
3. **Testing components** (plugins, GPG, network connectivity)
4. **Reviewing documentation** (command reference, guides)
5. **Using built-in help** (`--help` flags)

For persistent issues, consult GitHub Issues or submit a bug report with detailed diagnostics.
