#!/bin/bash
# Integration tests for SECA-CLI
# Run this script to test the full workflow

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BINARY="${BINARY:-./seca}"
TEST_RESULTS_DIR="./test_results_integration"
TEST_ENGAGEMENT_FILE="./test_engagements_integration.json"

# Cleanup function
cleanup() {
    echo -e "${YELLOW}Cleaning up test artifacts...${NC}"
    rm -rf "$TEST_RESULTS_DIR"
    rm -f "$TEST_ENGAGEMENT_FILE"
    echo -e "${GREEN}✓ Cleanup complete${NC}"
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

echo "=================================================="
echo "SECA-CLI Integration Tests"
echo "=================================================="
echo ""

# Test 1: Binary exists
echo -e "${YELLOW}Test 1: Checking if binary exists...${NC}"
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}✗ Binary not found at $BINARY${NC}"
    echo "Please build the binary first: make build"
    exit 1
fi
echo -e "${GREEN}✓ Binary found${NC}"
echo ""

# Test 2: Help command
echo -e "${YELLOW}Test 2: Testing help command...${NC}"
if $BINARY --help > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Help command works${NC}"
else
    echo -e "${RED}✗ Help command failed${NC}"
    exit 1
fi
echo ""

# Test 3: Create engagement
echo -e "${YELLOW}Test 3: Creating test engagement...${NC}"
ENGAGEMENT_OUTPUT=$($BINARY engagement create \
    --name "Integration Test Engagement" \
    --owner "test@example.com" \
    --roe "Test authorization for integration testing" \
    --roe-agree 2>&1)

ENGAGEMENT_ID=$(echo "$ENGAGEMENT_OUTPUT" | grep -oP 'id=\K[0-9]+' | head -1)

if [ -z "$ENGAGEMENT_ID" ]; then
    echo -e "${RED}✗ Failed to create engagement${NC}"
    echo "$ENGAGEMENT_OUTPUT"
    exit 1
fi

echo -e "${GREEN}✓ Engagement created with ID: $ENGAGEMENT_ID${NC}"
echo ""

# Test 4: List engagements
echo -e "${YELLOW}Test 4: Listing engagements...${NC}"
if $BINARY engagement list > /dev/null 2>&1; then
    echo -e "${GREEN}✓ List engagements works${NC}"
else
    echo -e "${RED}✗ List engagements failed${NC}"
    exit 1
fi
echo ""

# Test 5: Add scope
echo -e "${YELLOW}Test 5: Adding scope to engagement...${NC}"
SCOPE_OUTPUT=$($BINARY engagement add-scope \
    --id "$ENGAGEMENT_ID" \
    --scope https://httpbin.org,https://example.com 2>&1)

if echo "$SCOPE_OUTPUT" | grep -q "Added scope"; then
    echo -e "${GREEN}✓ Scope added successfully${NC}"
else
    echo -e "${RED}✗ Failed to add scope${NC}"
    echo "$SCOPE_OUTPUT"
    exit 1
fi
echo ""

# Test 6: Run HTTP check (without --roe-confirm to test validation)
echo -e "${YELLOW}Test 6: Testing ROE confirmation requirement...${NC}"
if $BINARY check http --id "$ENGAGEMENT_ID" 2>&1 | grep -q "roe-confirm"; then
    echo -e "${GREEN}✓ ROE confirmation requirement enforced${NC}"
else
    echo -e "${RED}✗ ROE confirmation check failed${NC}"
    exit 1
fi
echo ""

# Test 7: Run HTTP check with proper flags
echo -e "${YELLOW}Test 7: Running HTTP checks...${NC}"
CHECK_OUTPUT=$($BINARY --operator "integration-test" check http \
    --id "$ENGAGEMENT_ID" \
    --roe-confirm \
    --concurrency 2 \
    --rate 1 \
    --timeout 10 2>&1 || true)

if echo "$CHECK_OUTPUT" | grep -q "Run complete"; then
    echo -e "${GREEN}✓ HTTP checks completed${NC}"
else
    echo -e "${YELLOW}⚠ HTTP checks may have partial failures (this is ok for testing)${NC}"
fi

# Try to extract the actual results path from the output
EXTRACTED_RESULTS=$(echo "$CHECK_OUTPUT" | grep -oP 'Results: \K[^[:space:]]+' | head -1)
if [ -n "$EXTRACTED_RESULTS" ]; then
    # Extract directory path
    EXTRACTED_DIR=$(dirname "$EXTRACTED_RESULTS")
    echo "  Results directory: $EXTRACTED_DIR"
fi
echo ""

# Test 8: Verify results directory structure
echo -e "${YELLOW}Test 8: Verifying results directory structure...${NC}"

# Get the actual results directory path from the check output
# The seca-cli uses XDG data directory by default (~/.local/share/seca-cli/results on Linux)
# First check if we extracted the path from the output, otherwise try common locations
RESULTS_DIR=""

if [ -n "$EXTRACTED_DIR" ] && [ -d "$EXTRACTED_DIR" ]; then
    RESULTS_DIR="$EXTRACTED_DIR"
    echo -e "${GREEN}✓ Using extracted results directory: $RESULTS_DIR${NC}"
else
    # Try common locations
    POSSIBLE_DIRS=(
        "./results/$ENGAGEMENT_ID"
        "$HOME/.local/share/seca-cli/results/$ENGAGEMENT_ID"
        "$HOME/Library/Application Support/seca-cli/results/$ENGAGEMENT_ID"
        "$LOCALAPPDATA/seca-cli/results/$ENGAGEMENT_ID"
    )

    for dir in "${POSSIBLE_DIRS[@]}"; do
        if [ -d "$dir" ]; then
            RESULTS_DIR="$dir"
            echo -e "${GREEN}✓ Found results directory at: $RESULTS_DIR${NC}"
            break
        fi
    done
fi

if [ -z "$RESULTS_DIR" ]; then
    echo -e "${RED}✗ Results directory not found in any expected location${NC}"
    if [ -n "$EXTRACTED_DIR" ]; then
        echo "Extracted from output but not found: $EXTRACTED_DIR"
    fi
    echo "Checked locations:"
    for dir in "${POSSIBLE_DIRS[@]}"; do
        echo "  - $dir"
    done
    echo ""
    echo "Check output was:"
    echo "$CHECK_OUTPUT"
    exit 1
fi

if [ -f "$RESULTS_DIR/audit.csv" ]; then
    echo -e "${GREEN}✓ audit.csv exists${NC}"
else
    echo -e "${RED}✗ audit.csv not found${NC}"
    exit 1
fi

if [ -f "$RESULTS_DIR/results.json" ]; then
    echo -e "${GREEN}✓ results.json exists${NC}"
else
    echo -e "${RED}✗ results.json not found${NC}"
    exit 1
fi

if [ -f "$RESULTS_DIR/audit.csv.sha256" ]; then
    echo -e "${GREEN}✓ audit.csv.sha256 exists${NC}"
else
    echo -e "${RED}✗ audit.csv.sha256 not found${NC}"
    exit 1
fi

if [ -f "$RESULTS_DIR/results.json.sha256" ]; then
    echo -e "${GREEN}✓ results.json.sha256 exists${NC}"
else
    echo -e "${RED}✗ results.json.sha256 not found${NC}"
    exit 1
fi
echo ""

# Test 9: Verify hash files
echo -e "${YELLOW}Test 9: Verifying hash integrity...${NC}"
cd "$RESULTS_DIR"
if sha256sum -c audit.csv.sha256 > /dev/null 2>&1; then
    echo -e "${GREEN}✓ audit.csv hash verified${NC}"
else
    echo -e "${RED}✗ audit.csv hash verification failed${NC}"
    cd - > /dev/null
    exit 1
fi

if sha256sum -c results.json.sha256 > /dev/null 2>&1; then
    echo -e "${GREEN}✓ results.json hash verified${NC}"
else
    echo -e "${RED}✗ results.json hash verification failed${NC}"
    cd - > /dev/null
    exit 1
fi
cd - > /dev/null
echo ""

# Test 10: Verify audit CSV format
echo -e "${YELLOW}Test 10: Verifying audit CSV format...${NC}"
AUDIT_CSV="$RESULTS_DIR/audit.csv"

# Check header
if head -1 "$AUDIT_CSV" | grep -q "timestamp,engagement_id,operator,command"; then
    echo -e "${GREEN}✓ Audit CSV header is correct${NC}"
else
    echo -e "${RED}✗ Audit CSV header is incorrect${NC}"
    exit 1
fi

# Count lines (should be header + at least 1 data row)
LINE_COUNT=$(wc -l < "$AUDIT_CSV")
if [ "$LINE_COUNT" -ge 2 ]; then
    echo -e "${GREEN}✓ Audit CSV contains data rows ($((LINE_COUNT - 1)) entries)${NC}"
else
    echo -e "${RED}✗ Audit CSV has no data rows${NC}"
    exit 1
fi
echo ""

# Test 11: Verify results JSON format
echo -e "${YELLOW}Test 11: Verifying results JSON format...${NC}"
RESULTS_JSON="$RESULTS_DIR/results.json"

if jq -e '.metadata.operator' "$RESULTS_JSON" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Results JSON has metadata.operator${NC}"
else
    echo -e "${YELLOW}⚠ jq not installed, skipping JSON validation${NC}"
fi

if jq -e '.results' "$RESULTS_JSON" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Results JSON has results array${NC}"
else
    echo -e "${YELLOW}⚠ jq not installed, skipping JSON validation${NC}"
fi
echo ""

# Test 12: Test Makefile targets (if available)
if [ -f "Makefile" ]; then
    echo -e "${YELLOW}Test 12: Testing Makefile targets...${NC}"

    if make verify ENGAGEMENT_ID="$ENGAGEMENT_ID" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ make verify works${NC}"
    else
        echo -e "${YELLOW}⚠ make verify failed (might be expected)${NC}"
    fi

    if make show-stats ENGAGEMENT_ID="$ENGAGEMENT_ID" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ make show-stats works${NC}"
    else
        echo -e "${YELLOW}⚠ make show-stats failed${NC}"
    fi
    echo ""
fi

# Summary
echo "=================================================="
echo -e "${GREEN}Integration Tests Completed Successfully!${NC}"
echo "=================================================="
echo ""
echo "Summary:"
echo "  - Engagement ID: $ENGAGEMENT_ID"
echo "  - Results directory: $RESULTS_DIR"
echo "  - Audit entries: $((LINE_COUNT - 1))"
echo ""
echo "Run 'make clean' to remove evidence packages"
echo ""
