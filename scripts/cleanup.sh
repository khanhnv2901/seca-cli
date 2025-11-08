#!/bin/bash
# SECA-CLI Cleanup Script
# Removes test data, build artifacts, and temporary files

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}SECA-CLI Cleanup${NC}"
echo ""

# Confirm before cleaning
if [ "$1" != "--force" ]; then
    echo "This will remove:"
    echo "  - engagements.json (test data)"
    echo "  - results/ (test results)"
    echo "  - bin/ (build artifacts)"
    echo "  - dist/ (distribution files)"
    echo "  - release/ (release packages)"
    echo "  - coverage files"
    echo "  - Go cache"
    echo ""
    read -p "Continue? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Cleanup cancelled"
        exit 0
    fi
fi

echo -e "${YELLOW}→ Cleaning test data...${NC}"
if [ -f "engagements.json" ]; then
    rm -f engagements.json
    echo -e "${GREEN}  ✓${NC} Removed engagements.json"
else
    echo "  ⊘ No engagements.json found"
fi

if [ -f "engagements.json.backup" ]; then
    rm -f engagements.json.backup
    echo -e "${GREEN}  ✓${NC} Removed engagements.json.backup"
fi

echo ""
echo -e "${YELLOW}→ Cleaning test results...${NC}"
if [ -d "results" ]; then
    rm -rf results/
    echo -e "${GREEN}  ✓${NC} Removed results/"
else
    echo "  ⊘ No results/ directory found"
fi

if ls test_results* 1> /dev/null 2>&1; then
    rm -rf test_results*
    echo -e "${GREEN}  ✓${NC} Removed test_results*"
fi

if ls test_engagements*.json 1> /dev/null 2>&1; then
    rm -f test_engagements*.json
    echo -e "${GREEN}  ✓${NC} Removed test_engagements*.json"
fi

echo ""
echo -e "${YELLOW}→ Cleaning build artifacts...${NC}"
if [ -d "bin" ]; then
    rm -rf bin/
    echo -e "${GREEN}  ✓${NC} Removed bin/"
else
    echo "  ⊘ No bin/ directory found"
fi

if [ -d "dist" ]; then
    rm -rf dist/
    echo -e "${GREEN}  ✓${NC} Removed dist/"
else
    echo "  ⊘ No dist/ directory found"
fi

if [ -d "release" ]; then
    rm -rf release/
    echo -e "${GREEN}  ✓${NC} Removed release/"
else
    echo "  ⊘ No release/ directory found"
fi

if [ -f "seca" ]; then
    rm -f seca
    echo -e "${GREEN}  ✓${NC} Removed seca binary"
fi

echo ""
echo -e "${YELLOW}→ Cleaning coverage files...${NC}"
if [ -f "coverage.out" ]; then
    rm -f coverage.out
    echo -e "${GREEN}  ✓${NC} Removed coverage.out"
fi

if [ -f "coverage.html" ]; then
    rm -f coverage.html
    echo -e "${GREEN}  ✓${NC} Removed coverage.html"
fi

echo ""
echo -e "${YELLOW}→ Cleaning evidence packages...${NC}"
if ls evidence-*.tar.gz 1> /dev/null 2>&1; then
    rm -f evidence-*.tar.gz evidence-*.tar.gz.asc evidence-*.tar.gz.sha256
    echo -e "${GREEN}  ✓${NC} Removed evidence packages"
else
    echo "  ⊘ No evidence packages found"
fi

echo ""
echo -e "${YELLOW}→ Cleaning backup files...${NC}"
if ls *.backup 1> /dev/null 2>&1; then
    rm -f *.backup
    echo -e "${GREEN}  ✓${NC} Removed backup files"
else
    echo "  ⊘ No backup files found"
fi

echo ""
echo -e "${YELLOW}→ Cleaning Go caches...${NC}"
go clean -testcache 2>/dev/null && echo -e "${GREEN}  ✓${NC} Cleaned test cache" || echo "  ⊘ Test cache already clean"
go clean -cache 2>/dev/null && echo -e "${GREEN}  ✓${NC} Cleaned build cache" || echo "  ⊘ Build cache already clean"

echo ""
echo -e "${GREEN}✓ Cleanup complete!${NC}"
echo ""
echo "Repository is clean and ready for commit."
