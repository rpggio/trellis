#!/bin/bash
set -e

# Deploy a standalone trellis server in a given directory

usage() {
    echo "Usage: $0 <deployment_directory>"
    echo ""
    echo "Deploys trellis server to the specified directory with:"
    echo "  - Compiled server binary"
    echo "  - Configuration file"
    echo "  - Database directory"
    echo "  - API key for authentication"
    echo ""
    echo "Example:"
    echo "  $0 ~/my-trellis-deployment"
    exit 1
}

if [ $# -ne 1 ]; then
    usage
fi

DEPLOY_DIR="$1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🚀 Deploying trellis to: $DEPLOY_DIR"
echo ""

# Create deployment directory structure
mkdir -p "$DEPLOY_DIR"
mkdir -p "$DEPLOY_DIR/data"
mkdir -p "$DEPLOY_DIR/bin"

# Build the server
echo "📦 Building server binary..."
cd "$PROJECT_ROOT"
go build -o "$DEPLOY_DIR/bin/trellis-server" ./cmd/server

# Generate API key
echo "🔑 Generating API key..."
API_KEY=$(openssl rand -hex 32)
API_KEY_HASH=$(echo -n "$API_KEY" | openssl dgst -sha256 -binary | xxd -p -c 256)

# Create config file
echo "📝 Creating configuration..."
cat > "$DEPLOY_DIR/config.yaml" <<EOF
server:
  host: "127.0.0.1"
  port: 8080

db:
  path: "data/trellis.db"

log:
  level: "info"

auth:
  enabled: false
EOF

# Create environment file
cat > "$DEPLOY_DIR/.env" <<EOF
# Trellis Memory Server Configuration
TRELLIS_CONFIG_PATH=$DEPLOY_DIR/config.yaml
TRELLIS_SERVER_HOST=127.0.0.1
TRELLIS_SERVER_PORT=8080
TRELLIS_DB_PATH=$DEPLOY_DIR/data/trellis.db
TRELLIS_LOG_LEVEL=info
TRELLIS_AUTH_ENABLED=false

# API Key (keep secret!)
TRELLIS_API_KEY=$API_KEY
EOF

# Create a simple SQLite database with the API key
echo "💾 Initializing database..."
sqlite3 "$DEPLOY_DIR/data/trellis.db" <<SQL
CREATE TABLE IF NOT EXISTS api_keys (
    key_hash TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used TIMESTAMP,
    description TEXT
);

INSERT OR IGNORE INTO api_keys (tenant_id, key_hash) VALUES ('default', '$API_KEY_HASH');
SQL

# Create start script
cat > "$DEPLOY_DIR/start.sh" <<'STARTSCRIPT'
#!/bin/bash
cd "$(dirname "$0")"
set -a
source .env
set +a
exec ./bin/trellis-server
STARTSCRIPT
chmod +x "$DEPLOY_DIR/start.sh"

# Create stop script
cat > "$DEPLOY_DIR/stop.sh" <<'STOPSCRIPT'
#!/bin/bash
pkill -f "trellis-server" || true
echo "Server stopped"
STOPSCRIPT
chmod +x "$DEPLOY_DIR/stop.sh"

# Create README
cat > "$DEPLOY_DIR/README.md" <<'README'
# Trellis Memory Server Deployment

This directory contains a standalone deployment of the trellis server.

## Quick Start

Start the server:
```bash
./start.sh
```

Stop the server:
```bash
./stop.sh
```

## Configuration

Configuration is stored in:
- `config.yaml` - Main configuration
- `.env` - Environment variables and API key

The API key is stored in `.env` as `TRELLIS_API_KEY`.
Auth is disabled by default (`TRELLIS_AUTH_ENABLED=false`).

## Database

The SQLite database is located at `data/trellis.db`.

## Testing the Server

```bash
# Check if server is running
curl -X POST http://127.0.0.1:8080/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"list_projects","params":{}}'
```

## MCP Registration

See `REGISTER.md` for instructions on registering this server with various MCP clients.
README

# Create registration instructions
cat > "$DEPLOY_DIR/REGISTER.md" <<'REGISTER'
# MCP Client Registration

## Claude Desktop (Mac)

Remote HTTP servers must be added through the UI:

1. Open Claude Desktop
2. Settings → Extensions (or Connectors)
3. Add Custom Connector
4. Fill in:
   - Name: \`trellis\`
   - URL: \`https://YOUR_HTTPS_HOST/mcp\` (Claude Desktop requires https)
   - OAuth Client ID / Secret: leave blank (auth disabled for local)

For local development, use an HTTPS tunnel or reverse proxy that points to:
`http://127.0.0.1:8080/mcp`.

Note: \`claude_desktop_config.json\` is for local stdio MCP servers only.

## Claude Code CLI

Add to your Claude Code configuration:
\`\`\`bash
claude-code config set mcp.servers.trellis.url "http://127.0.0.1:8080/mcp"
\`\`\`

## Cline (VS Code Extension)

Edit Cline settings and add to MCP servers:
\`\`\`json
{
  "trellis": {
    "url": "http://127.0.0.1:8080/mcp"
  }
}
\`\`\`

## Generic MCP Client

Use these connection details:
- URL: \`http://127.0.0.1:8080/mcp\`
- Authentication: Disabled for local deployments by default
REGISTER

echo ""
echo "✅ Deployment complete!"
echo ""
echo "📂 Deployment location: $DEPLOY_DIR"
echo "🔑 API Key: $API_KEY"
echo ""
echo "ℹ️  Auth is disabled by default for local deployments."
echo "   The API key is stored in: $DEPLOY_DIR/.env (useful if you enable auth)"
echo ""
echo "🚀 To start the server:"
echo "   cd $DEPLOY_DIR"
echo "   ./start.sh"
echo ""
echo "📖 For MCP client registration instructions:"
echo "   cat $DEPLOY_DIR/REGISTER.md"
echo ""
