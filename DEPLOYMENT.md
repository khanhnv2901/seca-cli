# SECA-CLI Deployment Guide

**Phase 2: Internal Distribution**

This guide walks you through deploying SECA-CLI v1.0.0 to your organization.

---

## Pre-Deployment Checklist

### Code Quality
- [x] All tests passing (29/29 unit tests, integration tests)
- [x] Code coverage >85% for tested functions
- [x] No security vulnerabilities
- [x] Documentation complete

### Legal & Compliance
- [ ] Legal team review of tool capabilities
- [ ] Compliance team approval for usage
- [ ] Terms of use documented
- [ ] Operator training materials prepared

### Infrastructure
- [ ] Internal artifact repository configured (optional)
- [ ] GPG keys for signing (recommended)
- [ ] Network access for target systems
- [ ] Storage for results/evidence

---

## Deployment Steps

### Step 1: Build Binaries

```bash
# Navigate to project directory
cd seca-cli

# Run tests first
make test-all

# Build for all platforms
VERSION=1.0.0 ./scripts/build.sh

# Verify builds
ls -lh dist/
```

**Expected Output:**
```
seca-1.0.0-linux-amd64
seca-1.0.0-linux-arm64
seca-1.0.0-darwin-amd64
seca-1.0.0-darwin-arm64
seca-1.0.0-windows-amd64.exe
```

### Step 2: Create Release

```bash
# Create release package
./scripts/release.sh 1.0.0

# Verify release
ls -lh release/v1.0.0/
```

**Release Contents:**
- Binaries for all platforms
- `checksums.txt` with SHA256 hashes
- `checksums.txt.asc` (GPG signature)
- `RELEASE_NOTES.md`

### Step 3: Sign Release (Recommended)

If your organization uses GPG signing:

```bash
cd release/v1.0.0/

# Sign checksums
gpg --detach-sign --armor checksums.txt

# Verify signature
gpg --verify checksums.txt.asc checksums.txt
```

### Step 4: Internal Distribution

#### Option A: Internal Artifact Repository

```bash
# Upload to internal repository (Artifactory, Nexus, etc.)
# Example with curl to Artifactory:
for file in release/v1.0.0/seca-*; do
    curl -u user:password \
         -T "$file" \
         "https://artifactory.internal/seca-cli/v1.0.0/$(basename $file)"
done
```

#### Option B: Shared Network Drive

```bash
# Copy to shared drive
cp -r release/v1.0.0/ /mnt/shared/tools/seca-cli/

# Set permissions
chmod -R 755 /mnt/shared/tools/seca-cli/v1.0.0/
```

#### Option C: Internal GitHub Release

```bash
# Tag the release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Create GitHub release using gh CLI
gh release create v1.0.0 \
    --title "SECA-CLI v1.0.0" \
    --notes-file release/v1.0.0/RELEASE_NOTES.md \
    release/v1.0.0/seca-* \
    release/v1.0.0/checksums.txt \
    release/v1.0.0/checksums.txt.asc
```

---

## Operator Onboarding

### Step 1: Provide Documentation

Send operators:
1. [INSTALL.md](INSTALL.md) - Installation instructions
2. [OPERATOR_TRAINING.md](OPERATOR_TRAINING.md) - Training guide
3. [COMPLIANCE.md](COMPLIANCE.md) - Compliance requirements
4. [README.md](README.md) - General documentation

### Step 2: Training Session

Conduct a training session covering:

**Hour 1: Introduction & Setup**
- Tool overview and capabilities
- Legal and ethical requirements
- Installation walkthrough
- Configuration setup

**Hour 2: Hands-On Practice**
- Creating first engagement
- Adding scope
- Running HTTP checks
- Verifying evidence

**Hour 3: Compliance & Best Practices**
- Compliance mode usage
- Evidence packaging
- Audit trail review
- Common workflows

### Step 3: Certification

Have operators complete:

1. **Knowledge Assessment**
   - Understanding of legal requirements
   - Command syntax
   - Compliance procedures

2. **Practical Exercise**
   - Create test engagement
   - Execute checks
   - Package evidence
   - Verify integrity

3. **Sign-off**
   - Operator signs OPERATOR_TRAINING.md certification
   - Manager approval
   - Record in training log

---

## Installation Instructions for Operators

### Linux

```bash
# Download from internal repository
wget https://internal-repo/seca-cli/v1.0.0/seca-1.0.0-linux-amd64

# Verify checksum
sha256sum seca-1.0.0-linux-amd64
# Compare with checksums.txt

# Install
chmod +x seca-1.0.0-linux-amd64
sudo mv seca-1.0.0-linux-amd64 /usr/local/bin/seca

# Verify
seca version
```

### macOS

```bash
# Download
curl -L -o seca https://internal-repo/seca-cli/v1.0.0/seca-1.0.0-darwin-arm64

# Verify checksum
shasum -a 256 seca
# Compare with checksums.txt

# Install
chmod +x seca
sudo mv seca /usr/local/bin/

# Verify
seca version
```

### Windows

```powershell
# Download
Invoke-WebRequest -Uri "https://internal-repo/seca-cli/v1.0.0/seca-1.0.0-windows-amd64.exe" -OutFile "seca.exe"

# Verify checksum
Get-FileHash -Algorithm SHA256 seca.exe
# Compare with checksums.txt

# Install to Program Files
mkdir "C:\Program Files\seca-cli"
Move-Item seca.exe "C:\Program Files\seca-cli\"

# Add to PATH
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Program Files\seca-cli\", "Machine")

# Verify (new PowerShell window)
seca version
```

---

## Configuration Management

### Centralized Configuration

Create a centralized config template:

```yaml
# /mnt/shared/seca-cli/default-config.yaml
results_dir: /mnt/shared/seca-cli/results
operator: ${OPERATOR_NAME}
```

Operators copy and customize:

```bash
mkdir -p ~/.config/seca-cli
cp /mnt/shared/seca-cli/default-config.yaml ~/.seca-cli.yaml
# Edit to set operator name
```

### Results Storage

**Option 1: Local Storage**
```yaml
results_dir: ~/seca-results
```

**Option 2: Shared Network Storage**
```yaml
results_dir: /mnt/shared/seca-results
```

**Option 3: Cloud Storage (mounted)**
```yaml
results_dir: /mnt/s3-bucket/seca-results
```

---

## Support Infrastructure

### Internal Support Channels

1. **Slack Channel**: #seca-cli-support
2. **Email**: security-tools@company.com
3. **Wiki**: https://wiki.internal/seca-cli
4. **Issue Tracker**: https://jira.internal/projects/SECA

### Support Tiers

**Tier 1: Self-Service**
- Documentation review
- FAQ check
- Training materials

**Tier 2: Team Support**
- Slack channel
- Peer assistance
- Training sessions

**Tier 3: Tool Maintainer**
- Complex issues
- Bug reports
- Feature requests

---

## Monitoring & Metrics

### Usage Tracking

Monitor:
- Number of engagements created
- Number of checks executed
- Compliance mode adoption rate
- Common error patterns

### Audit Log Collection (Optional)

Centralize audit logs:

```bash
# Cron job to collect audit logs
0 * * * * rsync -avz ~/seca-results/*/audit.csv /mnt/shared/seca-audit-logs/
```

### Compliance Reporting

Generate monthly reports:
- Total engagements
- Operators trained
- Evidence packages created
- Hash verification statistics

---

## Maintenance

### Update Procedure

1. **Announce Update**
   - Email all operators
   - Slack announcement
   - Post on wiki

2. **Test New Version**
   - Run full test suite
   - Validate on each platform
   - Test with real engagements

3. **Deploy**
   - Upload to repository
   - Update documentation
   - Provide migration guide (if needed)

4. **Verify**
   - Check operator feedback
   - Monitor for issues
   - Address problems quickly

### Backup Strategy

**Configuration Backups**
```bash
# Weekly backup of configs
0 0 * * 0 tar -czf ~/backups/seca-config-$(date +\%Y\%m\%d).tar.gz ~/.seca-cli.yaml
```

**Evidence Backups**
```bash
# Daily backup of results
0 2 * * * rsync -avz ~/seca-results/ /backup/seca-results/
```

---

## Security Considerations

### Access Control

- Limit tool distribution to authorized personnel only
- Maintain operator registry
- Revoke access for departed employees

### Audit & Compliance

- Regular review of usage patterns
- Compliance audits of generated evidence
- Verification of ROE documentation

### Incident Response

If unauthorized use is detected:

1. Disable operator's access
2. Investigate scope of usage
3. Review all engagements
4. Report to security team
5. Document lessons learned

---

## Success Metrics

### Phase 2 Goals

- [ ] 100% of security team trained
- [ ] All compliance requirements met
- [ ] Evidence integrity at 100%
- [ ] Zero unauthorized usage incidents
- [ ] Positive operator feedback

### KPIs

- **Training Completion Rate**: >95%
- **Tool Adoption Rate**: >80% of eligible staff
- **Compliance Mode Usage**: >90% of checks
- **Hash Verification Success**: 100%
- **Average Time to Evidence Package**: <5 minutes

---

## Troubleshooting

### Common Issues

**Issue**: Operators can't download binaries
**Solution**: Check network access to repository

**Issue**: Hash verification fails
**Solution**: Re-download binary, check network integrity

**Issue**: GPG signature verification fails
**Solution**: Import public key, verify key trust chain

**Issue**: Results not being saved
**Solution**: Check `results_dir` permissions and disk space

---

## Rollback Plan

If critical issues are discovered:

```bash
# 1. Announce rollback
# 2. Revert to previous version
sudo mv /usr/local/bin/seca /usr/local/bin/seca.new
sudo cp /backup/seca.old /usr/local/bin/seca

# 3. Verify
seca version

# 4. Notify operators
# 5. Investigate issue
# 6. Plan fix and re-deployment
```

---

## Next Steps (Phase 3)

After successful internal distribution:

1. **Gather Feedback** - Collect operator experiences
2. **Iterate** - Implement improvements
3. **Expand** - Roll out to additional teams
4. **External Distribution** - Consider public release (if applicable)

---

## Questions & Support

For deployment assistance:
- Email: khanhnv2901@gmail.com
- GitHub: https://github.com/khanhnv2901/seca-cli

---

*Last Updated: 2025-01-15*
