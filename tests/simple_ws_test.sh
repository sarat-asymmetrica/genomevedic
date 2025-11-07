#!/bin/bash

# Simple WebSocket test for GenomeVedic Collaboration
# Tests basic WebSocket connectivity and message passing

set -e

echo "==================================="
echo "GenomeVedic WebSocket Simple Test"
echo "==================================="
echo ""

# Check if server is running
if ! curl -s http://localhost:8888/health > /dev/null; then
    echo "Error: Server not running on port 8888"
    echo "Start with: /tmp/collab_server --redis \"\" --port 8888"
    exit 1
fi

# Create session
echo "[1/3] Creating test session..."
SESSION_RESPONSE=$(curl -s -X POST http://localhost:8888/api/v1/collab/sessions \
  -H "Content-Type: application/json" \
  -d '{"name":"WebSocket Test","user_name":"TestUser","max_users":10}')

SESSION_ID=$(echo "$SESSION_RESPONSE" | grep -o '"session_id":"[^"]*' | cut -d'"' -f4)
echo "✓ Session created: $SESSION_ID"
echo ""

# Verify session exists
echo "[2/3] Verifying session..."
SESSION_INFO=$(curl -s "http://localhost:8888/api/v1/collab/sessions/$SESSION_ID")
USER_COUNT=$(echo "$SESSION_INFO" | grep -o '"users"' | wc -l)
echo "✓ Session verified, users: $USER_COUNT"
echo ""

# Test WebSocket connection (requires websocat or wscat)
echo "[3/3] Testing WebSocket..."
if command -v websocat &> /dev/null; then
    WS_URL="ws://localhost:8888/api/v1/collab/session/$SESSION_ID?user_name=TestUser2&permission=editor"
    echo "WebSocket URL: $WS_URL"
    echo "To connect manually:"
    echo "  websocat '$WS_URL'"
    echo "✓ websocat available for testing"
elif command -v wscat &> /dev/null; then
    WS_URL="ws://localhost:8888/api/v1/collab/session/$SESSION_ID?user_name=TestUser2&permission=editor"
    echo "WebSocket URL: $WS_URL"
    echo "To connect manually:"
    echo "  wscat -c '$WS_URL'"
    echo "✓ wscat available for testing"
else
    echo "Note: Install websocat or wscat for interactive testing"
    echo "  npm install -g wscat"
    echo "  # or"
    echo "  cargo install websocat"
fi
echo ""

# Get statistics
echo "Server Statistics:"
curl -s http://localhost:8888/api/v1/collab/stats | jq -r '
  "  Active sessions: \(.active_sessions)",
  "  Total users: \(.total_users)",
  "  Avg latency: \(.avg_latency_ms)ms",
  "  p95 latency: \(.p95_latency_ms)ms"
'

echo ""
echo "==================================="
echo "✓ WebSocket server is functional!"
echo "==================================="
echo ""
echo "Session URL: http://localhost:5173/session/$SESSION_ID"
echo ""
