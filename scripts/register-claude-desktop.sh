#!/bin/bash
set -e

# Print Claude Desktop registration steps for threds-mcp on Mac

usage() {
    echo "Usage: $0 <deployment_directory> [server_name_suffix]"
    echo ""
    echo "Prints UI steps to register threds-mcp with Claude Desktop for Mac."
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
SERVER_HOST="${THREDS_SERVER_HOST:-127.0.0.1}"
SERVER_PORT="${THREDS_SERVER_PORT:-8080}"
SERVER_NAME="threds-${NAME_SUFFIX}"
AUTH_ENABLED="${THREDS_AUTH_ENABLED:-true}"

CLAUDE_APP_PLIST="/Applications/Claude.app/Contents/Info.plist"
MIN_CLAUDE_VERSION="1.1.0"

version_ge() {
    local IFS=.
    local -a current=($1)
    local -a required=($2)
    local i
    for ((i=0; i<${#current[@]} || i<${#required[@]}; i++)); do
        local cur="${current[i]:-0}"
        local req="${required[i]:-0}"
        if ((10#$cur > 10#$req)); then
            return 0
        fi
        if ((10#$cur < 10#$req)); then
            return 1
        fi
    done
    return 0
}

echo "🔧 Preparing Claude Desktop registration details"
echo ""
echo "Server name:    $SERVER_NAME"
echo "Local MCP URL:  http://$SERVER_HOST:$SERVER_PORT/mcp"
echo ""

APP_VERSION="unknown"
if [ -f "$CLAUDE_APP_PLIST" ] && command -v /usr/libexec/PlistBuddy &> /dev/null; then
    APP_VERSION=$(/usr/libexec/PlistBuddy -c "Print :CFBundleShortVersionString" "$CLAUDE_APP_PLIST" 2>/dev/null || echo "unknown")
fi

if [ "$APP_VERSION" = "unknown" ]; then
    echo "❌ Unable to detect Claude Desktop version."
    echo "   Expected: $CLAUDE_APP_PLIST"
    echo "   Please install Claude Desktop and re-run this script."
    exit 1
fi

if ! version_ge "$APP_VERSION" "$MIN_CLAUDE_VERSION"; then
    echo "❌ Claude Desktop $APP_VERSION is older than required $MIN_CLAUDE_VERSION."
    echo "   Please upgrade Claude Desktop and re-run this script."
    exit 1
fi

echo "ℹ️  Claude Desktop registers remote HTTP MCP servers through the UI."
echo "   The claude_desktop_config.json file is for local stdio servers."
echo "   Claude Desktop requires https URLs for connectors."
echo ""
echo "   Add this server in Claude Desktop:"
echo "   Settings → Extensions (or Connectors) → Add Custom Connector"
echo ""
echo "   Name: $SERVER_NAME"
echo "   URL:  https://YOUR_HTTPS_HOST/mcp"
if [ "$AUTH_ENABLED" = "false" ]; then
    echo "   OAuth Client ID / Secret: leave blank (auth disabled for local)"
else
    echo "   OAuth Client ID / Secret: required (bearer auth is not supported in the UI)"
fi
echo ""
echo "   For local development, create an HTTPS tunnel or reverse proxy that"
echo "   forwards https://YOUR_HTTPS_HOST/mcp → http://$SERVER_HOST:$SERVER_PORT/mcp"
echo ""
echo "✅ Config details provided. No file changes were made."
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
echo "   Name:   $SERVER_NAME"
echo "   Local:  http://$SERVER_HOST:$SERVER_PORT/mcp"
echo "   HTTPS:  https://YOUR_HTTPS_HOST/mcp"
echo ""
