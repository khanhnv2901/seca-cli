# SECA-CLI Installation Guide

Complete installation instructions for all supported platforms.

## Table of Contents

- [Quick Install](#quick-install)
- [Platform-Specific Instructions](#platform-specific-instructions)
- [Building from Source](#building-from-source)
- [Configuration](#configuration)
- [Verification](#verification)
- [Upgrading](#upgrading)
- [Uninstallation](#uninstallation)

---

## Quick Install

### Linux (amd64)

```bash
# Download
wget https://github.com/khanhnv2901/seca-cli/releases/download/v1.0.0/seca-1.0.0-linux-amd64

# Make executable
chmod +x seca-1.0.0-linux-amd64

# Move to system path
sudo mv seca-1.0.0-linux-amd64 /usr/local/bin/seca

# Verify
seca version
```

### macOS (Apple Silicon)

```bash
# Download
curl -L -o seca https://github.com/khanhnv2901/seca-cli/releases/download/v1.0.0/seca-1.0.0-darwin-arm64

# Make executable
chmod +x seca

# Move to system path
sudo mv seca /usr/local/bin/

# Verify
seca version
```

### Windows (PowerShell)

```powershell
# Download
Invoke-WebRequest -Uri "https://github.com/khanhnv2901/seca-cli/releases/download/v1.0.0/seca-1.0.0-windows-amd64.exe" -OutFile "seca.exe"

# Move to a directory in PATH (example: C:\Program Files\seca-cli\)
mkdir "C:\Program Files\seca-cli"
Move-Item seca.exe "C:\Program Files\seca-cli\"

# Add to PATH (run as Administrator)
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Program Files\seca-cli\", "Machine")

# Verify (open new PowerShell window)
seca version
```

---

## Platform-Specific Instructions

### Ubuntu / Debian

```bash
# Install dependencies (if building from source)
sudo apt-get update
sudo apt-get install -y git make golang-go

# Install pre-built binary
wget https://github.com/khanhnv2901/seca-cli/releases/download/v1.0.0/seca-1.0.0-linux-amd64
chmod +x seca-1.0.0-linux-amd64
sudo mv seca-1.0.0-linux-amd64 /usr/local/bin/seca

# Verify
seca version
```

### CentOS / RHEL / Fedora

```bash
# Install dependencies (if building from source)
sudo dnf install -y git make golang

# Install pre-built binary
wget https://github.com/khanhnv2901/seca-cli/releases/download/v1.0.0/seca-1.0.0-linux-amd64
chmod +x seca-1.0.0-linux-amd64
sudo mv seca-1.0.0-linux-amd64 /usr/local/bin/seca

# Verify
seca version
```

### macOS (Intel)

```bash
# Download
curl -L -o seca https://github.com/khanhnv2901/seca-cli/releases/download/v1.0.0/seca-1.0.0-darwin-amd64

# Make executable
chmod +x seca

# Move to system path
sudo mv seca /usr/local/bin/

# If you get a security warning, allow it in System Preferences > Security & Privacy

# Verify
seca version
```

### macOS (Homebrew - Future)

```bash
# Coming soon
# brew tap khanhnv2901/seca-cli
# brew install seca-cli
```

### Windows (WSL)

If using Windows Subsystem for Linux, follow the Linux instructions for your WSL distribution.

---

## Building from Source

### Prerequisites

- Go 1.21 or higher
- Git
- Make (optional but recommended)

### Clone and Build

```bash
# Clone repository
git clone https://github.com/khanhnv2901/seca-cli.git
cd seca-cli

# Build
make build

# Install (optional)
sudo make install

# Verify
seca version
```

### Build for Specific Platform

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o seca main.go

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o seca main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o seca.exe main.go
```

### Build with Version Information

```bash
VERSION=1.0.0 GIT_COMMIT=$(git rev-parse --short HEAD) ./scripts/build.sh
```

---

## Configuration

### Data Storage Locations

**SECA-CLI v0.2.0+ stores data in OS-appropriate directories:**

**Linux/Unix:**
```
~/.local/share/seca-cli/
├── engagements.json
└── results/
    └── <engagement-id>/
```

**macOS:**
```
~/Library/Application Support/seca-cli/
├── engagements.json
└── results/
    └── <engagement-id>/
```

**Windows:**
```
%LOCALAPPDATA%\seca-cli\
├── engagements.json
└── results\
    └── <engagement-id>\
```

### Automatic Migration

When upgrading from versions prior to 0.2.0:
- SECA-CLI automatically migrates `engagements.json` from the project directory on first run
- The old file is backed up as `engagements.json.backup`
- See [Data Migration Guide](../reference/data-migration.md) for details

### Global Configuration File

Create `~/.seca-cli.yaml`:

```yaml
# Optional: Override default data directory
results_dir: /custom/path/to/results

# Example: Use shared team directory
# results_dir: /mnt/shared/seca-data

# Example: Use network storage
# results_dir: /mnt/nfs/security-team/seca

# Default operator name (optional)
operator: your-name
```

**Default Behavior (when `results_dir` is not set):**
- Linux/Unix: `~/.local/share/seca-cli/`
- macOS: `~/Library/Application Support/seca-cli/`
- Windows: `%LOCALAPPDATA%\seca-cli\`

### Environment Variables

```bash
# Set operator name
export SECA_OPERATOR="your-name"

# Set config file location
export SECA_CONFIG="/path/to/config.yaml"
```

### Project-Specific Configuration

Create `.seca-cli.yaml` in your project directory:

```yaml
results_dir: ./project-results
operator: project-team
```

> **Note:** Project-specific configurations override user-level data directories.

---

## Verification

### Verify Installation

```bash
# Check version
seca version

# Should output something like:
# SECA-CLI version 1.0.0
```

### Verify with Verbose Output

```bash
seca version --verbose

# Should show:
# SECA-CLI Version Information:
#   Version:    1.0.0
#   Git Commit: abc1234
#   Build Date: 2025-01-15T10:30:00Z
#   Go Version: go1.21.5
#   OS/Arch:    linux/amd64
```

### Verify Commands

```bash
# Show help
seca --help

# List available commands
seca engagement --help
seca check --help
```

### Run Tests (from source)

```bash
cd seca-cli
make test
```

---

## Upgrading

### Upgrade to Latest Version

```bash
# Download new version
wget https://github.com/khanhnv2901/seca-cli/releases/download/v1.1.0/seca-1.1.0-linux-amd64

# Replace existing binary
sudo mv seca-1.1.0-linux-amd64 /usr/local/bin/seca
chmod +x /usr/local/bin/seca

# Verify new version
seca version
```

### Upgrade from Source

```bash
cd seca-cli

# Pull latest changes
git pull origin main

# Rebuild and install
make build
sudo make install

# Verify
seca version
```

### Migration Notes

When upgrading, check `CHANGELOG.md` for:
- Breaking changes
- New features
- Configuration changes
- Data migration requirements

---

## Uninstallation

### Remove Binary

```bash
# Linux / macOS
sudo rm /usr/local/bin/seca

# Windows (PowerShell, as Administrator)
Remove-Item "C:\Program Files\seca-cli\seca.exe"
```

### Remove Configuration

```bash
# Remove user configuration
rm ~/.seca-cli.yaml

# Remove project configuration
rm .seca-cli.yaml
```

### Remove Data

**CAUTION: This deletes all engagements and evidence!**

```bash
# Linux/Unix
rm -rf ~/.local/share/seca-cli/

# macOS
rm -rf ~/Library/Application\ Support/seca-cli/

# Windows (PowerShell)
Remove-Item -Recurse -Force "$env:LOCALAPPDATA\seca-cli"
```

**If using custom data directory**, check your config:
```bash
# Check config for custom results_dir
cat ~/.seca-cli.yaml | grep results_dir
```

### Complete Uninstall

**Linux/Unix:**
```bash
# Remove binary
sudo rm /usr/local/bin/seca

# Remove configuration
rm ~/.seca-cli.yaml

# Remove data (CAUTION!)
rm -rf ~/.local/share/seca-cli/

# Remove old project-directory data if exists
rm -rf results/
rm -f engagements.json engagements.json.backup

echo "SECA-CLI uninstalled"
```

**macOS:**
```bash
# Remove binary
sudo rm /usr/local/bin/seca

# Remove configuration
rm ~/.seca-cli.yaml

# Remove data (CAUTION!)
rm -rf ~/Library/Application\ Support/seca-cli/

# Remove old project-directory data if exists
rm -rf results/
rm -f engagements.json engagements.json.backup

echo "SECA-CLI uninstalled"
```

**Windows (PowerShell):**
```powershell
# Remove binary
Remove-Item "C:\Program Files\seca-cli\seca.exe"

# Remove configuration
Remove-Item "$env:USERPROFILE\.seca-cli.yaml"

# Remove data (CAUTION!)
Remove-Item -Recurse -Force "$env:LOCALAPPDATA\seca-cli"

# Remove old project-directory data if exists
Remove-Item -Recurse -Force "results\"
Remove-Item "engagements.json", "engagements.json.backup" -ErrorAction SilentlyContinue

Write-Host "SECA-CLI uninstalled"
```

---

## Troubleshooting

### Binary Not Found

```bash
# Check if binary is in PATH
which seca

# If not found, ensure /usr/local/bin is in PATH
echo $PATH | grep /usr/local/bin

# Add to PATH if needed (add to ~/.bashrc or ~/.zshrc)
export PATH=$PATH:/usr/local/bin
```

### Permission Denied

```bash
# Make binary executable
chmod +x /usr/local/bin/seca

# Or if installing to user directory
chmod +x ~/bin/seca
```

### macOS Security Warning

If you see "cannot be opened because the developer cannot be verified":

1. Open System Preferences > Security & Privacy
2. Click "Open Anyway" for seca
3. Or remove quarantine attribute:
   ```bash
   xattr -d com.apple.quarantine /usr/local/bin/seca
   ```

### Windows SmartScreen Warning

If Windows Defender SmartScreen blocks execution:

1. Click "More info"
2. Click "Run anyway"
3. Or add seca.exe to Windows Defender exclusions

### Build Errors

```bash
# Ensure Go is installed and up to date
go version  # Should be 1.21 or higher

# Clean and rebuild
go clean -cache
go mod tidy
go build -v main.go
```

---

## Post-Installation

### Recommended Next Steps

1. **Read Documentation**
   ```bash
   cat README.md
   cat docs/operator-guide/compliance.md
   ```

2. **Complete Operator Training**
   ```bash
   cat docs/operator-guide/operator-training.md
   ```

3. **Run Integration Tests**
   ```bash
   make test-integration
   ```

4. **Create First Engagement**
   ```bash
   seca engagement create \
     --name "Test Engagement" \
     --owner "your-email@example.com" \
     --roe "Training exercise" \
     --roe-agree
   ```

---

## System Requirements

### Minimum

- CPU: 1 core
- RAM: 256 MB
- Disk: 50 MB
- Network: Internet connectivity (for HTTP checks)

### Recommended

- CPU: 2+ cores
- RAM: 512 MB+
- Disk: 1 GB+ (for storing results)
- Network: Stable internet connection

---

## Support

### Getting Help

- **Documentation**: Check [README.md](../../README.md), [Compliance Guide](../operator-guide/compliance.md), [Testing Guide](../technical/testing.md)
- **Issues**: https://github.com/khanhnv2901/seca-cli/issues
- **Email**: khanhnv2901@gmail.com

### Reporting Bugs

When reporting issues, include:

```bash
seca version --verbose
# Output of the command
# Error messages
# Steps to reproduce
```

---

## License

This software is provided under the MIT License. See LICENSE file for details.

## Security

For security issues, please email khanhnv2901@gmail.com with details.

---

*Last updated: 2025-11-09*
