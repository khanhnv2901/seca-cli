# Network Security Checks

SECA-CLI now includes comprehensive network security assessments to detect infrastructure vulnerabilities.

---

## Features

### 1. Open Port Scanning

Scans for open TCP ports and identifies potential security risks based on exposed services.

**Capabilities:**
- Concurrent port scanning with configurable worker pools
- Risk classification (Critical, High, Medium, Low, Info)
- Service identification for common ports
- Banner grabbing for additional service details
- Customizable port lists

**Default Ports Scanned:**
- **21** (FTP) - HIGH RISK
- **22** (SSH) - HIGH RISK
- **23** (Telnet) - CRITICAL RISK
- **25** (SMTP) - MEDIUM RISK
- **53** (DNS) - INFO
- **80** (HTTP) - LOW RISK
- **110** (POP3) - MEDIUM RISK
- **143** (IMAP) - MEDIUM RISK
- **443** (HTTPS) - LOW RISK
- **445** (SMB) - HIGH RISK
- **3306** (MySQL) - HIGH RISK
- **3389** (RDP) - CRITICAL RISK
- **5432** (PostgreSQL) - HIGH RISK
- **5900** (VNC) - CRITICAL RISK
- **6379** (Redis) - HIGH RISK
- **8080** (HTTP Alt) - MEDIUM RISK
- **8443** (HTTPS Alt) - MEDIUM RISK
- **27017** (MongoDB) - HIGH RISK

**Risk Classifications:**

- **CRITICAL**: Ports that should never be exposed to the internet (Telnet, RDP, VNC)
  - Recommendation: Close immediately or use VPN for access

- **HIGH**: Administrative and database ports that need strict access control (SSH, MySQL, PostgreSQL, Redis, MongoDB, SMB)
  - Recommendation: Restrict to trusted IPs, use strong authentication

- **MEDIUM**: Mail and alternative web ports (SMTP, POP3, IMAP, 8080, 8443)
  - Recommendation: Review security configuration

- **LOW**: Standard web ports (HTTP, HTTPS)
  - Recommendation: Ensure proper TLS configuration

- **INFO**: Informational ports (DNS, unknown services)
  - Recommendation: Review necessity

---

### 2. Subdomain Takeover Detection

Identifies potential subdomain takeover vulnerabilities where CNAME records point to unclaimed resources.

**Detection Methods:**

1. **CNAME Analysis**: Checks if subdomain has a CNAME record
2. **DNS Resolution**: Verifies if the CNAME target resolves
3. **HTTP Fingerprinting**: Matches response content against known takeover signatures
4. **Provider Detection**: Identifies the hosting provider from CNAME patterns

**Supported Providers** (20+):
- **GitHub Pages** - Detects "There isn't a GitHub Pages site here"
- **AWS S3** - Detects "NoSuchBucket" errors
- **Heroku** - Detects "No such app" errors
- **Azure** - Detects "404 Web Site not found"
- **Shopify** - Detects "Sorry, this shop is currently unavailable"
- **Netlify** - Auto-detects from CNAME patterns
- **Vercel** - Auto-detects from CNAME patterns
- **And 15+ more providers...**

**Confidence Levels:**
- **High**: CNAME doesn't resolve + known provider + matching fingerprint
- **Medium**: CNAME doesn't resolve + known provider
- **Low**: CNAME exists but no other indicators

**Example Vulnerable Scenario:**
```
subdomain.example.com -> CNAME -> old-project.github.io
                                   (doesn't resolve)

Vulnerability: An attacker could claim "old-project" on GitHub
and take control of subdomain.example.com
```

---

## Usage

### Using NetworkChecker in Code

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/khanhnv2901/seca-cli/internal/checker"
)

func main() {
    // Create network checker
    netChecker := &checker.NetworkChecker{
        Timeout:         10 * time.Second,
        PortScanTimeout: 2 * time.Second,
        EnablePortScan:  true,
        CommonPorts:     []int{80, 443, 22, 3306, 3389},
        MaxPortWorkers:  10,
    }

    // Run check
    result := netChecker.Check(context.Background(), "example.com")

    // Access results
    if result.NetworkSecurity != nil {
        fmt.Printf("Open Ports: %d\n", len(result.NetworkSecurity.OpenPorts))

        for _, port := range result.NetworkSecurity.OpenPorts {
            fmt.Printf("  Port %d (%s): %s - %s\n",
                port.Port, port.Service, port.Risk, port.Description)
        }

        if result.NetworkSecurity.SubdomainTakeover.Vulnerable {
            fmt.Printf("⚠️  SUBDOMAIN TAKEOVER DETECTED!\n")
            fmt.Printf("  Provider: %s\n", result.NetworkSecurity.SubdomainTakeover.Provider)
            fmt.Printf("  Confidence: %s\n", result.NetworkSecurity.SubdomainTakeover.Confidence)
        }
    }
}
```

### Configuration Options

```go
type NetworkChecker struct {
    Timeout         time.Duration // Overall check timeout (default: 10s)
    PortScanTimeout time.Duration // Per-port scan timeout (default: 2s)
    EnablePortScan  bool          // Enable port scanning (default: false)
    CommonPorts     []int         // Ports to scan (default: standard 18 ports)
    MaxPortWorkers  int           // Concurrent port scans (default: 10)
}
```

---

## JSON Output

### CheckResult Structure

```json
{
  "target": "example.com",
  "status": "ok",
  "checked_at": "2025-01-20T10:30:00Z",
  "network_security": {
    "open_ports": [
      {
        "port": 22,
        "protocol": "tcp",
        "state": "open",
        "service": "ssh",
        "banner": "SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.5",
        "risk": "high",
        "description": "HIGH RISK: Port 22 (ssh) exposed - ensure proper authentication and encryption"
      },
      {
        "port": 3389,
        "protocol": "tcp",
        "state": "open",
        "service": "rdp",
        "risk": "critical",
        "description": "CRITICAL: Port 3389 (rdp) should not be exposed to the internet"
      }
    ],
    "subdomain_takeover": {
      "vulnerable": true,
      "cname": "old-app.herokuapp.com",
      "provider": "Heroku",
      "fingerprint": "No such app",
      "confidence": "high",
      "http_status_code": 404,
      "recommendation": "The subdomain shows signs of being claimable on Heroku. Detected fingerprint: 'No such app'. Verify ownership of the Heroku resource or remove the DNS record."
    },
    "port_scan_duration_ms": 1234.56,
    "issues": [
      "1 critical port(s) exposed (Telnet/RDP/VNC)",
      "1 high-risk port(s) exposed (SSH/Database/SMB)",
      "Subdomain takeover vulnerability detected (Provider: Heroku, Confidence: high)"
    ],
    "recommendations": [
      "Close or firewall critical ports. Use VPN for remote access instead of direct exposure.",
      "Restrict database and administrative ports to trusted IPs only. Use strong authentication.",
      "The subdomain shows signs of being claimable on Heroku..."
    ]
  }
}
```

---

## Security Best Practices

### Port Scanning Ethics
- ✅ **DO**: Scan your own infrastructure
- ✅ **DO**: Get written permission before scanning
- ✅ **DO**: Follow your organization's security policies
- ✅ **DO**: Scan during approved maintenance windows
- ❌ **DON'T**: Scan systems you don't own without permission
- ❌ **DON'T**: Use aggressive scan settings that could cause DoS
- ❌ **DON'T**: Bypass firewall rules or IDS/IPS systems

### Subdomain Takeover Prevention
1. **Remove unused DNS records**: Delete CNAME records for decommissioned services
2. **Monitor DNS changes**: Track all DNS record modifications
3. **Verify ownership**: Ensure you control all CNAMEd resources
4. **Use automation**: Implement automated checks in CI/CD pipelines
5. **Regular audits**: Scan all subdomains quarterly

### Remediation Priority
1. **CRITICAL** vulnerabilities first (exposed RDP, Telnet, VNC)
2. **Subdomain takeover** vulnerabilities (can lead to phishing, data theft)
3. **HIGH** risk ports (databases, SSH, SMB)
4. **MEDIUM** risk ports (mail servers, alt web ports)
5. **LOW** risk ports (standard web ports with proper TLS)

---

## Performance Considerations

### Port Scanning
- **Default timeout**: 2 seconds per port
- **Concurrency**: 10 workers by default
- **18 default ports** ≈ 2-4 seconds total scan time
- Customize `MaxPortWorkers` for faster/slower scans

### Subdomain Takeover
- **DNS lookups**: ~100-500ms total
- **HTTP checks**: ~1-3 seconds (tries HTTPS first, then HTTP)
- **Total time**: ~2-5 seconds per target

### Optimization Tips
1. Reduce `CommonPorts` list for faster scans
2. Increase `MaxPortWorkers` on powerful machines
3. Lower `PortScanTimeout` for known responsive targets
4. Disable port scanning if only checking subdomain takeover
5. Use Go's context with timeout to prevent hanging

---

## Integration Examples

### CI/CD Pipeline

```yaml
# GitHub Actions example
name: Security Scan
on:
  schedule:
    - cron: '0 0 * * 0'  # Weekly
  workflow_dispatch:

jobs:
  network-scan:
    runs-on: ubuntu-latest
    steps:
      - name: Run Network Security Check
        run: |
          seca check network --target ${{ secrets.DOMAIN }} \
            --enable-port-scan \
            --ports 22,80,443,3306,3389 \
            --output results.json

      - name: Check for vulnerabilities
        run: |
          if jq -e '.network_security.subdomain_takeover.vulnerable == true' results.json; then
            echo "::error::Subdomain takeover vulnerability detected!"
            exit 1
          fi

          if jq -e '.network_security.issues | length > 0' results.json; then
            echo "::warning::Network security issues found"
            jq '.network_security.issues' results.json
          fi
```

### Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: seca-network-scan
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: seca
            image: seca-cli:latest
            command:
            - /bin/sh
            - -c
            - |
              seca check network \
                --target $DOMAIN \
                --enable-port-scan \
                --output /results/network-$(date +%Y%m%d).json
            env:
            - name: DOMAIN
              valueFrom:
                configMapKeyRef:
                  name: seca-config
                  key: domain
            volumeMounts:
            - name: results
              mountPath: /results
          restartPolicy: OnFailure
          volumes:
          - name: results
            persistentVolumeClaim:
              claimName: seca-results
```

---

## Troubleshooting

### Port Scan Returns No Results
- **Cause**: Firewall blocking outbound connections
- **Solution**: Check firewall rules, run on whitelisted machine

### Subdomain Takeover False Positives
- **Cause**: Temporary DNS propagation issues
- **Solution**: Rerun check after 24 hours, verify CNAME manually

### Timeout Errors
- **Cause**: Network latency or slow DNS servers
- **Solution**: Increase `Timeout` and `PortScanTimeout` values

### Permission Denied Errors
- **Cause**: Port scanning requires elevated privileges on some systems
- **Solution**: Run with appropriate permissions (be cautious)

---

## References

### OWASP Resources
- [OWASP Testing Guide - Port Scanning](https://owasp.org/www-project-web-security-testing-guide/)
- [OWASP - Subdomain Takeover](https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/02-Configuration_and_Deployment_Management_Testing/10-Test_for_Subdomain_Takeover)

### Subdomain Takeover Research
- [Subdomain Takeover: Basics](https://developer.mozilla.org/en-US/docs/Web/Security/Subdomain_takeovers)
- [Can I Take Over XYZ?](https://github.com/EdOverflow/can-i-take-over-xyz) - Provider fingerprints database

### Port Scanning Best Practices
- [NMAP Port Scanning Basics](https://nmap.org/book/man-port-scanning-basics.html)
- [SANS - Authorized Port Scanning](https://www.sans.org/reading-room/whitepapers/testing/authorized-port-scanning-960)

---

## Future Enhancements

Planned features for future releases:

- [ ] UDP port scanning support
- [ ] IPv6 support for port scanning
- [ ] OS fingerprinting via TCP/IP stack analysis
- [ ] SSL/TLS certificate chain validation during port scan
- [ ] Bulk subdomain takeover checking
- [ ] Integration with external DNS zone file parsing
- [ ] Historical tracking of port changes
- [ ] Automated remediation suggestions via API
- [ ] Export to SARIF format for security dashboards
- [ ] Custom provider fingerprint configuration

---

## Contributing

To add support for new subdomain takeover providers:

1. Update `getTakeoverFingerprints()` in [network.go](../../internal/checker/network.go)
2. Add provider patterns to `detectProvider()` function
3. Add test cases in [network_test.go](../../internal/checker/network_test.go)
4. Submit a pull request with evidence (screenshots, examples)

---

**Last Updated**: January 2025
**Version**: 1.3.0+
