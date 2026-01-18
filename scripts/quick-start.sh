#!/bin/bash
set -e

# Quick start script - deploys, starts, populates, and registers threds in one go

cat << "BANNER"
╔════════════════════════════════════════╗
║   Threds MCP Quick Start for Mac      ║
╚════════════════════════════════════════╝
BANNER

echo ""
echo "This script will:"
echo "  1. Deploy a standalone threds-mcp server"
echo "  2. Start the server"
echo "  3. Generate sample data"
echo "  4. Register with Claude Desktop"
echo ""

# Get deployment directory
read -p "Enter deployment directory (e.g., ~/threds-server): " DEPLOY_DIR
DEPLOY_DIR="${DEPLOY_DIR/#\~/$HOME}"  # Expand tilde

if [ -z "$DEPLOY_DIR" ]; then
    echo "❌ Deployment directory required"
    exit 1
fi

# Get optional server name
read -p "Enter server name suffix (optional, press Enter for random): " NAME_SUFFIX

# Ask about sample data
read -p "Generate sample data? (Y/n): " GEN_SAMPLE
GEN_SAMPLE="${GEN_SAMPLE:-Y}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo ""
echo "════════════════════════════════════════"
echo " Step 1: Deploying server"
echo "════════════════════════════════════════"
"$SCRIPT_DIR/deploy-standalone.sh" "$DEPLOY_DIR"

echo ""
echo "════════════════════════════════════════"
echo " Step 2: Starting server"
echo "════════════════════════════════════════"
cd "$DEPLOY_DIR"
./start.sh &
SERVER_PID=$!
echo "Server started (PID: $SERVER_PID)"

# Wait for server to be ready
echo "Waiting for server to be ready..."
for i in {1..30}; do
    if curl -s http://127.0.0.1:8080/rpc > /dev/null 2>&1; then
        echo "✅ Server is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ Server failed to start within 30 seconds"
        exit 1
    fi
    sleep 1
    echo -n "."
done
echo ""

if [[ "$GEN_SAMPLE" =~ ^[Yy] ]]; then
    echo ""
    echo "════════════════════════════════════════"
    echo " Step 3: Generating sample data"
    echo "════════════════════════════════════════"
    "$SCRIPT_DIR/generate-sample-data.sh" "$DEPLOY_DIR"
else
    echo ""
    echo "Skipping sample data generation"
fi

echo ""
echo "════════════════════════════════════════"
echo " Step 4: Registering with Claude Desktop"
echo "════════════════════════════════════════"
if [ -n "$NAME_SUFFIX" ]; then
    "$SCRIPT_DIR/register-claude-desktop.sh" "$DEPLOY_DIR" "$NAME_SUFFIX"
    SERVER_NAME="threds-$NAME_SUFFIX"
else
    "$SCRIPT_DIR/register-claude-desktop.sh" "$DEPLOY_DIR"
    # Extract the server name from the config
    SERVER_NAME=$(jq -r '.mcpServers | keys[] | select(startswith("threds-"))' "$HOME/Library/Application Support/Claude/claude_desktop_config.json" | tail -1)
fi

echo ""
echo "════════════════════════════════════════"
echo " ✅ Quick Start Complete!"
echo "════════════════════════════════════════"
echo ""
echo "📍 Deployment: $DEPLOY_DIR"
echo "🖥️  Server:     Running (PID: $SERVER_PID)"
echo "📡 MCP Name:   $SERVER_NAME"
echo ""
echo "🎯 Next Steps:"
echo ""
echo "1. Restart Claude Desktop"
echo ""
echo "2. In Claude, try asking:"
echo "   • \"Can you list my threds projects?\""
echo "   • \"Show me the records in the Mobile App Redesign project\""
echo "   • \"What questions are open in my design projects?\""
echo ""
echo "📖 Useful commands:"
echo ""
echo "   Stop server:   cd $DEPLOY_DIR && ./stop.sh"
echo "   Start server:  cd $DEPLOY_DIR && ./start.sh"
echo "   View logs:     cd $DEPLOY_DIR && tail -f nohup.out"
echo "   API key:       cat $DEPLOY_DIR/.env | grep THREDS_API_KEY"
echo ""
echo "📚 Documentation:"
echo "   • Server docs:  cat $DEPLOY_DIR/README.md"
echo "   • MCP clients:  cat $DEPLOY_DIR/REGISTER.md"
echo "   • Scripts:      cat $SCRIPT_DIR/README.md"
echo ""
echo "Happy reasoning! 🧠"
echo ""
