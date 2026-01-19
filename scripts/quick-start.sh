#!/bin/bash
set -e

# Quick start script - deploys, starts, populates, and prints registration details

cat << "BANNER"
╔════════════════════════════════════════╗
║   Threds MCP Quick Start for Mac      ║
╚════════════════════════════════════════╝
BANNER

SERVER_PID=""
cleanup() {
    if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" > /dev/null 2>&1; then
        echo ""
        echo "Stopping server..."
        kill "$SERVER_PID" > /dev/null 2>&1 || true
        wait "$SERVER_PID" > /dev/null 2>&1 || true
    fi
}
trap cleanup EXIT INT TERM

echo ""
echo "This script will:"
echo "  1. Deploy a standalone threds-mcp server"
echo "  2. Start the server"
echo "  3. Generate sample data"
echo "  4. Print Claude Desktop registration details"
echo ""
echo "The server stays attached to this script. Press Ctrl-C to stop it."
echo ""

# Get deployment directory (default random /tmp path)
if command -v openssl &> /dev/null; then
    RAND_SUFFIX="$(openssl rand -hex 3)"
else
    RAND_SUFFIX="${RANDOM}${RANDOM}"
fi
DEFAULT_DEPLOY_DIR="/tmp/threds-mcp-$RAND_SUFFIX"
while [ -e "$DEFAULT_DEPLOY_DIR" ]; do
    DEFAULT_DEPLOY_DIR="/tmp/threds-mcp-${RANDOM}${RANDOM}"
done

read -p "Enter deployment directory (default $DEFAULT_DEPLOY_DIR): " DEPLOY_DIR
DEPLOY_DIR="${DEPLOY_DIR:-$DEFAULT_DEPLOY_DIR}"
DEPLOY_DIR="${DEPLOY_DIR/#\~/$HOME}"  # Expand tilde

# Pick a random default port (first free in the ephemeral range)
pick_random_port() {
    local port
    local i
    for i in $(seq 1 50); do
        port=$((49152 + RANDOM % 16384))
        if ! lsof -i :"$port" > /dev/null 2>&1; then
            echo "$port"
            return 0
        fi
    done
    port=49152
    while lsof -i :"$port" > /dev/null 2>&1; do
        port=$((port + 1))
    done
    echo "$port"
}

DEFAULT_PORT="$(pick_random_port)"

read -p "Enter server port (default $DEFAULT_PORT): " SERVER_PORT
SERVER_PORT="${SERVER_PORT:-$DEFAULT_PORT}"

# Auto-generate connector name suffix
if command -v openssl &> /dev/null; then
    NAME_SUFFIX="$(openssl rand -hex 4)"
else
    NAME_SUFFIX="$(date +%s)"
fi

# Ask about sample data
read -p "Generate sample data? (Y/n): " GEN_SAMPLE
GEN_SAMPLE="${GEN_SAMPLE:-Y}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo ""
echo "════════════════════════════════════════"
echo " Step 1: Deploying server"
echo "════════════════════════════════════════"
"$SCRIPT_DIR/deploy-standalone.sh" "$DEPLOY_DIR"

if [ "$SERVER_PORT" != "8080" ]; then
    sed -i '' "s/^  port: .*/  port: $SERVER_PORT/" "$DEPLOY_DIR/config.yaml"
    sed -i '' "s/^THREDS_SERVER_PORT=.*/THREDS_SERVER_PORT=$SERVER_PORT/" "$DEPLOY_DIR/.env"
fi

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
    if curl -s "http://127.0.0.1:$SERVER_PORT/rpc" > /dev/null 2>&1; then
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
echo " Step 4: Preparing Claude Desktop registration"
echo "════════════════════════════════════════"
"$SCRIPT_DIR/register-claude-desktop.sh" "$DEPLOY_DIR" "$NAME_SUFFIX"
SERVER_NAME="threds-$NAME_SUFFIX"

echo ""
echo "════════════════════════════════════════"
echo " ✅ Quick Start Complete!"
echo "════════════════════════════════════════"
echo ""
echo "📍 Deployment: $DEPLOY_DIR"
echo "🖥️  Server:     Running (PID: $SERVER_PID)"
echo "📡 MCP Name:   $SERVER_NAME"
echo "🔌 Port:       $SERVER_PORT"
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
echo "   API key:       cat $DEPLOY_DIR/.env | grep THREDS_API_KEY (if auth enabled)"
echo ""
echo "📚 Documentation:"
echo "   • Server docs:  cat $DEPLOY_DIR/README.md"
echo "   • MCP clients:  cat $DEPLOY_DIR/REGISTER.md"
echo "   • Scripts:      cat $SCRIPT_DIR/README.md"
echo ""
echo "Happy reasoning! 🧠"
echo ""
echo "Press Ctrl-C to stop the server and exit."
wait "$SERVER_PID"
