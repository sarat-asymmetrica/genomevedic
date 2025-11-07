#!/bin/bash

# GenomeVedic Collaboration Server Load Testing Script
# Runs comprehensive load tests and generates reports

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
WS_URL="${WS_URL:-ws://localhost:8080}"
HTTP_URL="${HTTP_URL:-http://localhost:8080}"
RESULTS_DIR="./results"

echo -e "${BLUE}=============================================${NC}"
echo -e "${BLUE}GenomeVedic Load Testing Suite${NC}"
echo -e "${BLUE}=============================================${NC}"
echo ""

# Check if k6 is installed
if ! command -v k6 &> /dev/null; then
    echo -e "${RED}Error: k6 is not installed${NC}"
    echo "Install k6: https://k6.io/docs/getting-started/installation/"
    exit 1
fi

# Check if server is running
echo -e "${YELLOW}[1/5] Checking server health...${NC}"
if ! curl -s "${HTTP_URL}/health" > /dev/null; then
    echo -e "${RED}Error: Server not responding at ${HTTP_URL}${NC}"
    echo "Start the server first: go run backend/cmd/collab_server/main.go"
    exit 1
fi
echo -e "${GREEN}✓ Server is healthy${NC}"
echo ""

# Create results directory
mkdir -p "${RESULTS_DIR}"

# Create session for testing
echo -e "${YELLOW}[2/5] Creating test session...${NC}"
SESSION_RESPONSE=$(curl -s -X POST "${HTTP_URL}/api/v1/collab/sessions" \
  -H "Content-Type: application/json" \
  -d '{"name":"Load Test Session","user_name":"LoadTester","max_users":200}')

SESSION_ID=$(echo "$SESSION_RESPONSE" | grep -o '"session_id":"[^"]*' | cut -d'"' -f4)

if [ -z "$SESSION_ID" ]; then
    echo -e "${RED}Error: Failed to create session${NC}"
    echo "Response: $SESSION_RESPONSE"
    exit 1
fi

echo -e "${GREEN}✓ Session created: ${SESSION_ID}${NC}"
echo ""

# Run load tests
echo -e "${YELLOW}[3/5] Running WebSocket load test...${NC}"
echo "Target: 100+ concurrent users, <100ms p95 latency"
echo ""

k6 run \
  --out json="${RESULTS_DIR}/results.json" \
  -e WS_URL="${WS_URL}" \
  -e SESSION_ID="${SESSION_ID}" \
  websocket_load_test.js

echo ""
echo -e "${GREEN}✓ Load test complete${NC}"
echo ""

# Parse results
echo -e "${YELLOW}[4/5] Analyzing results...${NC}"

# Extract key metrics from JSON results
RESULTS_FILE="${RESULTS_DIR}/results.json"

if [ -f "$RESULTS_FILE" ]; then
    # Calculate metrics using jq if available
    if command -v jq &> /dev/null; then
        echo -e "${BLUE}Key Metrics:${NC}"

        # Connection metrics
        CONNECTIONS=$(jq -r 'select(.type=="Point" and .metric=="ws_connecting") | .data.value' "$RESULTS_FILE" | wc -l)
        echo "  Total connections: $CONNECTIONS"

        # Latency metrics (from summary if available)
        echo ""
        echo -e "${BLUE}Latency Statistics:${NC}"
        jq -r 'select(.type=="Point" and .metric=="cursor_update_latency") |
               "  Latency: \(.data.value)ms (timestamp: \(.data.time))"' "$RESULTS_FILE" | tail -5
    else
        echo "Install jq for detailed metrics analysis: sudo apt-get install jq"
    fi
fi

echo ""

# Get server statistics
echo -e "${YELLOW}[5/5] Getting server statistics...${NC}"
STATS=$(curl -s "${HTTP_URL}/api/v1/collab/stats")
echo "$STATS" | jq '.' 2>/dev/null || echo "$STATS"
echo ""

# Generate summary report
REPORT_FILE="${RESULTS_DIR}/load_test_report.txt"
cat > "$REPORT_FILE" << EOF
GenomeVedic Collaboration Server - Load Test Report
====================================================
Date: $(date)
Server: ${HTTP_URL}
Session: ${SESSION_ID}

Test Configuration:
- Target: 100+ concurrent users
- Duration: 5 minutes
- Latency target: <100ms p95

Results Summary:
$(k6 inspect "${RESULTS_DIR}/results.json" 2>/dev/null || echo "Run with k6 v0.43+ for detailed summary")

Server Statistics:
${STATS}

Test Status: COMPLETED
====================================================
EOF

echo -e "${GREEN}✓ Report generated: ${REPORT_FILE}${NC}"
echo ""

echo -e "${BLUE}=============================================${NC}"
echo -e "${GREEN}Load Testing Complete!${NC}"
echo -e "${BLUE}=============================================${NC}"
echo ""
echo "Results saved to: ${RESULTS_DIR}/"
echo "Report: ${REPORT_FILE}"
echo ""
echo "Next steps:"
echo "  1. Review the report above"
echo "  2. Check if p95 latency < 100ms"
echo "  3. Verify success rate > 99%"
echo "  4. Adjust server configuration if needed"
echo ""
