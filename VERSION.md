# Version Management

SECA-CLI uses compile-time version injection via Go's `-ldflags` to embed version information into the binary.

## Version Variables

The following variables are defined in [cmd/version.go](cmd/version.go):

```go
var (
    Version   = "dev"      // Semantic version (e.g., "1.2.0")
    GitCommit = "unknown"  // Short git commit hash
    BuildDate = "unknown"  // ISO 8601 timestamp
)
```

These default values indicate a development build. Production builds should override them using `-ldflags`.

## Building with Version Information

### Local Development Build (default)

```bash
make build
# or
go build -o seca main.go
```

This creates a development build with:
- Version: `dev`
- Git Commit: Current git commit hash (or "unknown" if not in git repo)
- Build Date: Current UTC timestamp

### Production Build with Specific Version

```bash
VERSION=1.2.0 make build
```

This creates a versioned build with:
- Version: `1.2.0`
- Git Commit: Current git commit hash
- Build Date: Current UTC timestamp

### Multi-Platform Release Build

```bash
VERSION=1.2.0 ./scripts/build.sh
```

Builds for all supported platforms:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

Outputs binaries to `./dist/` directory.

### Complete Release Process

```bash
make release VERSION=1.2.0
```

This runs [scripts/release.sh](scripts/release.sh) which:
1. Builds binaries for all platforms
2. Generates SHA256 checksums
3. Signs with GPG (if available)
4. Creates release archive
5. Generates release notes template

## Checking Version Information

### Simple Version

```bash
seca version
# Output: SECA-CLI version 1.2.0
```

### Detailed Version

```bash
seca version --verbose
# Output:
# SECA-CLI Version Information:
#   Version:    1.2.0
#   Git Commit: 942cf33
#   Build Date: 2025-11-12T18:47:04Z
#   Go Version: go1.25.1
#   OS/Arch:    linux/amd64
#   Compiler:   gc
```

## How It Works

### Compile-Time Injection

Go's linker (`-ldflags`) allows overriding variables at build time:

```bash
go build -ldflags="-X package.Variable=value" -o binary main.go
```

### SECA-CLI Implementation

In [cmd/version.go](cmd/version.go), we define package-level variables:

```go
package cmd

var Version = "dev"
var GitCommit = "unknown"
var BuildDate = "unknown"
```

During build, the Makefile and build scripts inject values:

```bash
LDFLAGS="-X github.com/khanhnv2901/seca-cli/cmd.Version=${VERSION}"
LDFLAGS="${LDFLAGS} -X github.com/khanhnv2901/seca-cli/cmd.GitCommit=${GIT_COMMIT}"
LDFLAGS="${LDFLAGS} -X github.com/khanhnv2901/seca-cli/cmd.BuildDate=${BUILD_DATE}"

go build -ldflags="${LDFLAGS}" -o seca main.go
```

### Version Extraction

```bash
# Get git commit hash
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Get build timestamp (ISO 8601 format)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

## Best Practices

### Development Builds

- Use `make build` without VERSION environment variable
- This creates `dev` builds that are easily identifiable
- Git commit is still tracked for debugging

### Pre-Release Builds

```bash
VERSION=1.2.0-rc.1 make build
VERSION=1.2.0-beta.2 make build
```

### Production Releases

```bash
# Always use semantic versioning
VERSION=1.2.0 make release

# Tag the release in git
git tag -a v1.2.0 -m "Release v1.2.0"
git push origin v1.2.0
```

### CI/CD Integration

For automated builds, the version can be injected from:
- Git tags: `git describe --tags --always`
- Environment variables: `$VERSION` or `$CI_COMMIT_TAG`
- Build metadata: `$BUILD_NUMBER` or `$GITHUB_RUN_NUMBER`

Example CI build:

```bash
VERSION=$(git describe --tags --always)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$GITHUB_SHA

go build \
  -ldflags="-X github.com/khanhnv2901/seca-cli/cmd.Version=$VERSION \
            -X github.com/khanhnv2901/seca-cli/cmd.GitCommit=$GIT_COMMIT \
            -X github.com/khanhnv2901/seca-cli/cmd.BuildDate=$BUILD_DATE" \
  -o seca main.go
```

## Troubleshooting

### Version shows "dev" in production

**Problem**: Production binary shows `Version: dev`

**Solution**: Ensure you're building with the VERSION environment variable:
```bash
VERSION=1.2.0 make build
```

### Git commit shows "unknown"

**Problem**: Git commit shows "unknown" instead of hash

**Solution**: Ensure you're building from within a git repository, or git is installed

### Build date shows "unknown"

**Problem**: Build date is not set

**Solution**: Ensure `date` command is available on your system

## Version History

See [CHANGELOG.md](CHANGELOG.md) for complete version history and release notes.

## Related Files

- [cmd/version.go](cmd/version.go) - Version command implementation
- [Makefile](Makefile) - Build automation with version injection
- [scripts/build.sh](scripts/build.sh) - Multi-platform build script
- [scripts/release.sh](scripts/release.sh) - Release automation
- [CHANGELOG.md](CHANGELOG.md) - Version history
