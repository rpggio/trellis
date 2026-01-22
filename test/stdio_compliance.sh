#!/bin/bash
# Stdio protocol compliance tests
#
# This script has been replaced with a more comprehensive test suite.
# Use one of the following options:

set -e

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "  MCP Stdio Protocol Compliance Testing"
echo "═══════════════════════════════════════════════════════════"
echo ""

# Build first
echo "Building server..."
make build > /dev/null 2>&1
echo "✓ Build complete"
echo ""

echo "Available Testing Options:"
echo ""
echo "1. QUICK SANITY TEST (Recommended for development)"
echo "   ./test/stdio_test.sh"
echo ""
echo "2. COMPREHENSIVE COMPLIANCE TEST (Recommended for CI/CD)"
echo "   # Install mcp-validation first:"
echo "   npm install -g @modelcontextprotocol/mcp-validation"
echo ""
echo "   # Then run:"
echo "   mcp-validate --transport stdio --env TRELLIS_DB_PATH=:memory: \\"
echo "     --json-report compliance-report.json --debug -- ./bin/trellis"
echo ""
echo "3. INTERACTIVE TESTING (For manual debugging)"
echo "   npx @modelcontextprotocol/inspector ./bin/trellis"
echo "   # Opens web UI at http://localhost:6274"
echo ""
echo "4. SCRIPTED TOOL TESTING"
echo "   # Install mcp-cli first:"
echo "   npm install -g @wong2/mcp-cli"
echo ""
echo "   # Then configure and run:"
echo "   cat > test-config.json <<EOF"
echo "   {"
echo "     \"mcpServers\": {"
echo "       \"trellis\": {"
echo "         \"command\": \"./bin/trellis\","
echo "         \"env\": { \"TRELLIS_DB_PATH\": \":memory:\" }"
echo "       }"
echo "     }"
echo "   }"
echo "   EOF"
echo ""
echo "   npx @wong2/mcp-cli --config test-config.json call-tool trellis:list_records"
echo ""
echo "═══════════════════════════════════════════════════════════"
echo ""

# Prompt user
read -p "Run quick sanity test now? [Y/n] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
  echo ""
  ./test/stdio_test.sh
else
  echo ""
  echo "See docs/0120-stdio-transport.md for detailed testing documentation."
fi
