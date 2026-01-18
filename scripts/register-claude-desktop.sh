#!/bin/bash
set -e

# Automatically register threds-mcp with Claude Desktop on Mac

usage() {
    echo "Usage: $0 <deployment_directory> [server_name_suffix]"
    echo ""
    echo "Registers the threds-mcp server with Claude Desktop for Mac."
    echo ""
    echo "Arguments:"
    echo "  deployment_directory  - Path to the deployed threds server"
    echo "  server_name_suffix    - Optional suffix for the server name (default: random)"
    echo ""
    echo "Example:"
    echo "  $0 ~/my-threds-deployment"
    echo "  $0 ~/my-threds-deployment my-project"
    exit 1
}

if [ $# -lt 1 ]; then
    usage
fi

DEPLOY_DIR="$1"
NAME_SUFFIX="${2:-$(openssl rand -hex 4)}"

# Validate deployment directory
if [ ! -f "$DEPLOY_DIR/.env" ]; then
    echo "❌ Error: Deployment not found at $DEPLOY_DIR"
    echo "   Run deploy-standalone.sh first"
    exit 1
fi

# Check if running on Mac
if [ "$(uname)" != "Darwin" ]; then
    echo "❌ Error: This script only works on macOS"
    echo "   For other platforms, manually edit your MCP client configuration"
    exit 1
fi

# Load configuration
source "$DEPLOY_DIR/.env"
API_KEY="$THREDS_API_KEY"
SERVER_HOST="${THREDS_SERVER_HOST:-127.0.0.1}"
SERVER_PORT="${THREDS_SERVER_PORT:-8080}"
SERVER_NAME="threds-${NAME_SUFFIX}"

CLAUDE_CONFIG_DIR="$HOME/Library/Application Support/Claude"
CLAUDE_CONFIG_FILE="$CLAUDE_CONFIG_DIR/claude_desktop_config.json"

echo "🔧 Registering threds-mcp with Claude Desktop"
echo ""
echo "Server name: $SERVER_NAME"
echo "Server URL:  http://$SERVER_HOST:$SERVER_PORT/mcp"
echo ""

# Create config directory if it doesn't exist
mkdir -p "$CLAUDE_CONFIG_DIR"

# Initialize config file if it doesn't exist
if [ ! -f "$CLAUDE_CONFIG_FILE" ]; then
    echo "📝 Creating new Claude Desktop config file..."
    echo '{"mcpServers":{}}' > "$CLAUDE_CONFIG_FILE"
else
    echo "📝 Updating existing Claude Desktop config file..."
fi

# Create backup
BACKUP_FILE="${CLAUDE_CONFIG_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
cp "$CLAUDE_CONFIG_FILE" "$BACKUP_FILE"
echo "💾 Backup created: $BACKUP_FILE"
echo ""

# Read existing config
EXISTING_CONFIG=$(cat "$CLAUDE_CONFIG_FILE")

# Add or update the threds server configuration using jq
# If jq is not available, fall back to Python
if command -v jq &> /dev/null; then
    # Use jq for JSON manipulation
    echo "$EXISTING_CONFIG" | jq \
        --arg name "$SERVER_NAME" \
        --arg url "http://$SERVER_HOST:$SERVER_PORT/mcp" \
        --arg token "Bearer $API_KEY" \
        '.mcpServers[$name] = {"url": $url, "headers": {"Authorization": $token}}' \
        > "$CLAUDE_CONFIG_FILE"
elif command -v python3 &> /dev/null; then
    # Use Python as fallback
    python3 << PYTHON
import json
import sys

config_file = "$CLAUDE_CONFIG_FILE"

try:
    with open(config_file, 'r') as f:
        config = json.load(f)
except:
    config = {"mcpServers": {}}

if "mcpServers" not in config:
    config["mcpServers"] = {}

config["mcpServers"]["$SERVER_NAME"] = {
    "url": "http://$SERVER_HOST:$SERVER_PORT/mcp",
    "headers": {
        "Authorization": "Bearer $API_KEY"
    }
}

with open(config_file, 'w') as f:
    json.dump(config, f, indent=2)

print("✅ Configuration updated successfully")
PYTHON
else
    echo "❌ Error: Neither jq nor python3 found"
    echo "   Please install jq (brew install jq) or ensure python3 is available"
    echo ""
    echo "Manual registration instructions:"
    echo "Add this to $CLAUDE_CONFIG_FILE:"
    echo ""
    cat <<MANUAL
{
  "mcpServers": {
    "$SERVER_NAME": {
      "url": "http://$SERVER_HOST:$SERVER_PORT/mcp",
      "headers": {
        "Authorization": "Bearer $API_KEY"
      }
    }
  }
}
MANUAL
    exit 1
fi

echo ""
echo "✅ Registration complete!"
echo ""
echo "🎯 Next steps:"
echo ""
echo "1. Make sure the server is running:"
echo "   cd $DEPLOY_DIR && ./start.sh"
echo ""
echo "2. Restart Claude Desktop to load the new configuration"
echo ""
echo "3. In a new Claude conversation, the server will be available as: $SERVER_NAME"
echo ""
echo "4. Test by asking Claude to list your projects:"
echo "   \"Can you list my threds projects?\""
echo ""
echo "📋 Server details:"
echo "   Name:         $SERVER_NAME"
echo "   URL:          http://$SERVER_HOST:$SERVER_PORT/mcp"
echo "   Config file:  $CLAUDE_CONFIG_FILE"
echo "   Backup:       $BACKUP_FILE"
echo ""
echo "⚠️  To unregister, remove the \"$SERVER_NAME\" entry from:"
echo "   $CLAUDE_CONFIG_FILE"
echo ""
