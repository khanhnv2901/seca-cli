#!/bin/bash
# SECA-CLI Release Script
# Creates a release with checksums and signatures

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Version (required)
VERSION="${1:-}"

if [ -z "$VERSION" ]; then
    echo -e "${RED}Error: Version is required${NC}"
    echo "Usage: $0 <version>"
    echo "Example: $0 1.0.0"
    exit 1
fi

# Directories
DIST_DIR="./dist"
RELEASE_DIR="./release/v${VERSION}"

echo -e "${YELLOW}═══════════════════════════════════════════${NC}"
echo -e "${YELLOW}  SECA-CLI Release v${VERSION}${NC}"
echo -e "${YELLOW}═══════════════════════════════════════════${NC}"
echo ""

# Check if dist directory exists
if [ ! -d "${DIST_DIR}" ]; then
    echo -e "${YELLOW}→ Building binaries first...${NC}"
    VERSION="${VERSION}" ./scripts/build.sh
fi

# Create release directory
echo -e "${YELLOW}→ Creating release directory...${NC}"
mkdir -p "${RELEASE_DIR}"

# Copy binaries
echo -e "${YELLOW}→ Copying binaries...${NC}"
cp -v "${DIST_DIR}"/* "${RELEASE_DIR}/"

# Generate checksums
echo ""
echo -e "${YELLOW}→ Generating SHA256 checksums...${NC}"
cd "${RELEASE_DIR}"

for file in seca-*; do
    if [ -f "$file" ]; then
        sha256sum "$file" >> checksums.txt
        echo -e "${GREEN}  ✓${NC} $(sha256sum $file | cut -d' ' -f1) ${file}"
    fi
done

# Generate overall checksum file
sha256sum -c checksums.txt > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}  ✓${NC} All checksums verified"
fi

cd - > /dev/null

# Sign release (if GPG is available)
if command -v gpg &> /dev/null; then
    echo ""
    echo -e "${YELLOW}→ Signing release with GPG...${NC}"
    cd "${RELEASE_DIR}"

    if gpg --detach-sign --armor checksums.txt 2>/dev/null; then
        echo -e "${GREEN}  ✓${NC} checksums.txt.asc created"
    else
        echo -e "${YELLOW}  ⚠${NC} GPG signing skipped (no key configured)"
    fi

    cd - > /dev/null
else
    echo -e "${YELLOW}  ⚠${NC} GPG not available, skipping signing"
fi

# Create release notes template
echo ""
echo -e "${YELLOW}→ Creating release notes template...${NC}"
cat > "${RELEASE_DIR}/RELEASE_NOTES.md" << EOF
# SECA-CLI v${VERSION}

Release Date: $(date +"%Y-%m-%d")

## What's New

<!-- Add release highlights here -->

- Feature: ...
- Improvement: ...
- Bug fix: ...

## Installation

### Linux (amd64)
\`\`\`bash
wget https://github.com/khanhnv2901/seca-cli/releases/download/v${VERSION}/seca-${VERSION}-linux-amd64
chmod +x seca-${VERSION}-linux-amd64
sudo mv seca-${VERSION}-linux-amd64 /usr/local/bin/seca
\`\`\`

### macOS (arm64 - Apple Silicon)
\`\`\`bash
wget https://github.com/khanhnv2901/seca-cli/releases/download/v${VERSION}/seca-${VERSION}-darwin-arm64
chmod +x seca-${VERSION}-darwin-arm64
sudo mv seca-${VERSION}-darwin-arm64 /usr/local/bin/seca
\`\`\`

### Windows (amd64)
Download \`seca-${VERSION}-windows-amd64.exe\` from the releases page.

## Verification

Verify the download:
\`\`\`bash
sha256sum -c checksums.txt
gpg --verify checksums.txt.asc checksums.txt
\`\`\`

## Checksums

See \`checksums.txt\` for SHA256 hashes of all binaries.

## Documentation

- [README.md](../../README.md)
- [COMPLIANCE.md](../../COMPLIANCE.md)
- [TESTING.md](../../TESTING.md)

## Support

- Issues: https://github.com/khanhnv2901/seca-cli/issues
- Email: khanhnv2901@gmail.com
EOF

echo -e "${GREEN}  ✓${NC} RELEASE_NOTES.md created"

# Create archive
echo ""
echo -e "${YELLOW}→ Creating release archive...${NC}"
cd release
tar -czf "seca-v${VERSION}.tar.gz" "v${VERSION}/"
echo -e "${GREEN}  ✓${NC} seca-v${VERSION}.tar.gz created"
cd - > /dev/null

# Summary
echo ""
echo -e "${GREEN}═══════════════════════════════════════════${NC}"
echo -e "${GREEN}  Release v${VERSION} Complete!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════${NC}"
echo ""
echo "Release location: ${RELEASE_DIR}/"
echo ""
echo "Files created:"
ls -lh "${RELEASE_DIR}/"
echo ""
echo "Archive: release/seca-v${VERSION}.tar.gz"
ls -lh "release/seca-v${VERSION}.tar.gz"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Review and edit ${RELEASE_DIR}/RELEASE_NOTES.md"
echo "2. Test binaries on target platforms"
echo "3. Create git tag: git tag -a v${VERSION} -m 'Release v${VERSION}'"
echo "4. Push tag: git push origin v${VERSION}"
echo "5. Upload release artifacts to GitHub"
echo ""
