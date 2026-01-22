#!/bin/bash
# Test stdio protocol compliance

set -e

echo "Building server..."
make build > /dev/null 2>&1

echo "Testing stdio protocol compliance..."

# Helper function to run server with timeout (works on macOS and Linux)
run_with_input() {
  local input="$1"
  (echo "$input"; sleep 1) | TRELLIS_TRANSPORT=stdio TRELLIS_DB_PATH=:memory: ./bin/trellis 2>/dev/null
}

# Test 1: Initialize
echo "Test 1: Initialize request"
RESPONSE=$(run_with_input '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}},"id":1}' | head -1)

if echo "$RESPONSE" | jq -e '.result.protocolVersion' > /dev/null 2>&1; then
  echo "✓ Initialize: OK"
  echo "  Protocol version: $(echo "$RESPONSE" | jq -r '.result.protocolVersion')"
  echo "  Server: $(echo "$RESPONSE" | jq -r '.result.serverInfo.name') v$(echo "$RESPONSE" | jq -r '.result.serverInfo.version')"
else
  echo "✗ Initialize: FAILED"
  echo "Response: $RESPONSE"
  exit 1
fi

# Test 2: Verify stdout is clean JSON (no log contamination)
echo ""
echo "Test 2: Verify stdout contains only JSON"
OUTPUT=$(run_with_input '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}},"id":1}')
LINE_COUNT=$(echo "$OUTPUT" | wc -l | tr -d ' ')

# Each line should be valid JSON
VALID=true
while IFS= read -r line; do
  if [ -n "$line" ] && ! echo "$line" | jq empty 2>/dev/null; then
    echo "✗ Found non-JSON output: $line"
    VALID=false
    break
  fi
done <<< "$OUTPUT"

if [ "$VALID" = true ]; then
  echo "✓ Stdout contains only valid JSON ($LINE_COUNT lines)"
else
  echo "✗ Stdout is contaminated with non-JSON output"
  exit 1
fi

# Test 3: Verify capabilities in initialize response
echo ""
echo "Test 3: Verify server capabilities"
RESPONSE=$(run_with_input '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}},"id":1}' | head -1)

if echo "$RESPONSE" | jq -e '.result.capabilities.tools' > /dev/null 2>&1; then
  echo "✓ Server advertises tools capability"
else
  echo "✗ Server missing tools capability"
  echo "Response: $RESPONSE"
  exit 1
fi

# Test 4: Verify logs go to stderr, not stdout
echo ""
echo "Test 4: Verify logs are on stderr only"
STDOUT_OUTPUT=$(run_with_input '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}},"id":1}' 2>&1 1>/dev/null)

if echo "$STDOUT_OUTPUT" | grep -q "level=INFO"; then
  echo "✓ Logs are written to stderr (not contaminating stdout)"
else
  echo "ℹ No logs detected (log level may be set to quiet)"
fi

echo ""
echo "=================================="
echo "All stdio protocol tests passed!"
echo "=================================="
echo ""
echo "For comprehensive compliance testing, install mcp-validation:"
echo "  npm install -g @modelcontextprotocol/mcp-validation"
echo ""
echo "Then run:"
echo "  mcp-validate --transport stdio --env TRELLIS_DB_PATH=:memory: --json-report report.json -- ./bin/trellis"
