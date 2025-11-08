#!/bin/bash
# SECA-CLI Build Script
# Builds binaries for multiple platforms with version information

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Version information
VERSION="${VERSION:-1.0.0}"
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build directory
BUILD_DIR="${BUILD_DIR:-./bin}"
DIST_DIR="${DIST_DIR:-./dist}"

# Platforms to build for
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

echo -e "${YELLOW}Building SECA-CLI v${VERSION}${NC}"
echo "Git Commit: ${GIT_COMMIT}"
echo "Build Date: ${BUILD_DATE}"
echo ""

# Create build directories
mkdir -p "${BUILD_DIR}"
mkdir -p "${DIST_DIR}"

# Ldflags for version injection
LDFLAGS="-X github.com/khanhnv2901/seca-cli/cmd.Version=${VERSION}"
LDFLAGS="${LDFLAGS} -X github.com/khanhnv2901/seca-cli/cmd.GitCommit=${GIT_COMMIT}"
LDFLAGS="${LDFLAGS} -X github.com/khanhnv2901/seca-cli/cmd.BuildDate=${BUILD_DATE}"
LDFLAGS="${LDFLAGS} -s -w" # Strip debug info

echo -e "${YELLOW}Building for multiple platforms...${NC}"
echo ""

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r -a platform_split <<< "$platform"
    GOOS="${platform_split[0]}"
    GOARCH="${platform_split[1]}"

    output_name="seca-${VERSION}-${GOOS}-${GOARCH}"

    if [ "$GOOS" = "windows" ]; then
        output_name+=".exe"
    fi

    echo -e "${GREEN}→${NC} Building ${GOOS}/${GOARCH}..."

    env GOOS="$GOOS" GOARCH="$GOARCH" go build \
        -ldflags="${LDFLAGS}" \
        -o "${DIST_DIR}/${output_name}" \
        main.go

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}  ✓${NC} ${output_name}"
    else
        echo -e "${RED}  ✗${NC} Failed to build ${GOOS}/${GOARCH}"
        exit 1
    fi
done

echo ""
echo -e "${GREEN}✓ Build complete!${NC}"
echo ""
echo "Binaries location: ${DIST_DIR}/"
ls -lh "${DIST_DIR}/"

# Also build local binary
echo ""
echo -e "${YELLOW}Building local binary...${NC}"
go build -ldflags="${LDFLAGS}" -o "${BUILD_DIR}/seca" main.go
echo -e "${GREEN}✓ Local binary: ${BUILD_DIR}/seca${NC}"

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Build Summary:${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo "Version: ${VERSION}"
echo "Commit:  ${GIT_COMMIT}"
echo "Date:    ${BUILD_DATE}"
echo "Output:  ${DIST_DIR}/"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
