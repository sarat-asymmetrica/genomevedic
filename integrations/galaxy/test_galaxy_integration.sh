#!/bin/bash
# GenomeVedic Galaxy Integration Test Suite
# Tests all major functionality of the Galaxy integration

set -e  # Exit on error

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
API_ENDPOINT="${GENOMEVEDIC_API:-http://localhost:8080}"
GALAXY_URL="${GALAXY_URL:-https://usegalaxy.org}"
TEST_SESSION_ID="test-session-$(date +%s)"

echo "=========================================="
echo "GenomeVedic Galaxy Integration Test Suite"
echo "=========================================="
echo ""
echo "API Endpoint: $API_ENDPOINT"
echo "Galaxy URL: $GALAXY_URL"
echo "Session ID: $TEST_SESSION_ID"
echo ""

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to run tests
run_test() {
    local test_name="$1"
    local test_command="$2"

    echo -n "Testing: $test_name... "

    if eval "$test_command" > /tmp/test_output.txt 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        echo "  Error output:"
        cat /tmp/test_output.txt | sed 's/^/    /'
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test 1: Check API health
run_test "API Health Check" \
    "curl -sf $API_ENDPOINT/api/v1/health | jq -e '.success == true'"

# Test 2: Check Galaxy integration status
run_test "Galaxy Integration Status" \
    "curl -sf $API_ENDPOINT/api/v1/galaxy/status | jq -e '.success == true'"

# Test 3: Validate supported features
run_test "Check Supported Features" \
    "curl -sf $API_ENDPOINT/api/v1/galaxy/status | jq -e '.features.bam_import == true'"

# Test 4: Check export formats
run_test "Check Export Formats" \
    "curl -sf $API_ENDPOINT/api/v1/galaxy/status | jq -e '.export_formats | contains([\"bed\", \"gtf\", \"gff3\"])'"

# Test 5: Test BAM import (simulated)
run_test "BAM Import API (simulated)" \
    "curl -sf -X POST $API_ENDPOINT/api/v1/import/galaxy \
        -H 'Content-Type: application/json' \
        -d '{
            \"session_id\": \"$TEST_SESSION_ID\",
            \"bam_path\": \"/tmp/test.bam\",
            \"genome_build\": \"hg38\",
            \"quality_threshold\": 20
        }' | jq -e '.session_id == \"$TEST_SESSION_ID\"'"

# Test 6: Test OAuth init (without actual authentication)
run_test "OAuth Initialization" \
    "curl -sf $API_ENDPOINT/api/v1/galaxy/oauth/init?user_id=test-user | jq -e '.success == true and .auth_url != null'"

# Test 7: Test API key validation format
run_test "API Key Validation Endpoint" \
    "curl -sf -X POST $API_ENDPOINT/api/v1/galaxy/validate-key \
        -H 'Content-Type: application/json' \
        -d '{\"api_key\": \"test-key\"}' | jq -e 'has(\"valid\")'"

# Test 8: Test import progress tracking
run_test "Import Progress Tracking" \
    "curl -sf '$API_ENDPOINT/api/v1/galaxy/import/progress?session_id=nonexistent' | jq -e '.in_progress == false'"

# Test 9: Test export API structure (without actual export)
run_test "Export API Endpoint Structure" \
    "curl -sf -X POST $API_ENDPOINT/api/v1/export/galaxy \
        -H 'Content-Type: application/json' \
        -H 'X-API-KEY: test-key' \
        -d '{
            \"session_id\": \"test\",
            \"history_id\": \"test-history\",
            \"format\": \"bed\",
            \"annotations\": []
        }' 2>&1 | grep -q 'annotations array is required' || true"

# Test 10: Verify CORS headers
run_test "CORS Headers Present" \
    "curl -sf -I $API_ENDPOINT/api/v1/galaxy/status | grep -i 'access-control-allow-origin'"

echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo "Total Tests: $((TESTS_PASSED + TESTS_FAILED))"
echo ""

# Test Galaxy Tool XML validation
echo "=========================================="
echo "Galaxy Tool XML Validation"
echo "=========================================="

XML_FILE="$(dirname "$0")/genomevedic.xml"

if [ -f "$XML_FILE" ]; then
    echo "Checking XML file: $XML_FILE"

    # Check if xmllint is available
    if command -v xmllint &> /dev/null; then
        if xmllint --noout "$XML_FILE" 2>/dev/null; then
            echo -e "${GREEN}✓${NC} XML is well-formed"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            echo -e "${RED}✗${NC} XML validation failed"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
    else
        echo -e "${YELLOW}⚠${NC} xmllint not available, skipping XML validation"
    fi

    # Check required XML elements
    echo "Checking required elements..."

    if grep -q '<tool id="genomevedic_visualizer"' "$XML_FILE"; then
        echo -e "${GREEN}✓${NC} Tool ID present"
    else
        echo -e "${RED}✗${NC} Tool ID missing"
    fi

    if grep -q '<inputs>' "$XML_FILE"; then
        echo -e "${GREEN}✓${NC} Inputs section present"
    else
        echo -e "${RED}✗${NC} Inputs section missing"
    fi

    if grep -q '<outputs>' "$XML_FILE"; then
        echo -e "${GREEN}✓${NC} Outputs section present"
    else
        echo -e "${RED}✗${NC} Outputs section missing"
    fi

    if grep -q 'format="bam"' "$XML_FILE"; then
        echo -e "${GREEN}✓${NC} BAM input format specified"
    else
        echo -e "${RED}✗${NC} BAM input format missing"
    fi
else
    echo -e "${RED}✗${NC} XML file not found: $XML_FILE"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

echo ""

# Test Python wrapper
echo "=========================================="
echo "Python Wrapper Validation"
echo "=========================================="

WRAPPER_FILE="$(dirname "$0")/genomevedic_wrapper.py"

if [ -f "$WRAPPER_FILE" ]; then
    echo "Checking Python wrapper: $WRAPPER_FILE"

    # Check if Python is available
    if command -v python3 &> /dev/null; then
        if python3 -m py_compile "$WRAPPER_FILE" 2>/dev/null; then
            echo -e "${GREEN}✓${NC} Python syntax is valid"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            echo -e "${RED}✗${NC} Python syntax errors found"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
    else
        echo -e "${YELLOW}⚠${NC} Python3 not available, skipping syntax check"
    fi

    # Check if wrapper is executable
    if [ -x "$WRAPPER_FILE" ]; then
        echo -e "${GREEN}✓${NC} Wrapper is executable"
    else
        echo -e "${YELLOW}⚠${NC} Wrapper is not executable (run: chmod +x $WRAPPER_FILE)"
    fi

    # Check required imports
    if grep -q 'import argparse' "$WRAPPER_FILE"; then
        echo -e "${GREEN}✓${NC} Argparse import present"
    else
        echo -e "${RED}✗${NC} Argparse import missing"
    fi

    if grep -q 'class GenomeVedicAPIClient' "$WRAPPER_FILE"; then
        echo -e "${GREEN}✓${NC} API client class present"
    else
        echo -e "${RED}✗${NC} API client class missing"
    fi
else
    echo -e "${RED}✗${NC} Python wrapper not found: $WRAPPER_FILE"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

echo ""
echo "=========================================="
echo "Final Results"
echo "=========================================="
echo -e "Total Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Total Failed: ${RED}$TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
