# Quick Reference Card

## One-Command Setup

```bash
./scripts/quick-start.sh
```

## Individual Commands

### Deploy Server
```bash
./scripts/deploy-standalone.sh ~/my-threds-server
```

### Start/Stop Server
```bash
cd ~/my-threds-server
./start.sh    # Start
./stop.sh     # Stop
```

### Generate Sample Data
```bash
./scripts/generate-sample-data.sh ~/my-threds-server
```

### Register with Claude Desktop
```bash
# Auto-generated name
./scripts/register-claude-desktop.sh ~/my-threds-server

# Custom name
./scripts/register-claude-desktop.sh ~/my-threds-server my-project
```
The script prints the exact values to enter in Claude Desktop:
Settings → Extensions/Connectors → Add Custom Connector.
Claude Desktop requires https URLs for connectors.
Use an HTTPS tunnel or reverse proxy that forwards to the local http endpoint.

## Testing API

### Get API Key
```bash
cat ~/my-threds-server/.env | grep THREDS_API_KEY
```
Only needed if you enable auth.

### List Projects
```bash
curl -X POST http://127.0.0.1:8080/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"list_projects","params":{}}'
```

### Get Project Overview
```bash
curl -X POST http://127.0.0.1:8080/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"get_project_overview","params":{}}'
```

### Search Records
```bash
curl -X POST http://127.0.0.1:8080/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"search_records","params":{"query":"navigation"}}'
```

## File Locations

### Deployment Files
```
~/my-threds-server/
├── bin/threds-server    # Binary
├── data/threds.db       # Database
├── config.yaml          # Config
├── .env                 # API key (keep secure!)
├── start.sh             # Start script
└── stop.sh              # Stop script
```

### Claude Desktop Config
`~/Library/Application Support/Claude/claude_desktop_config.json` is for
local stdio MCP servers only. Remote HTTP MCP servers are added via the UI.

## Multiple Servers

### Different Ports
1. Deploy: `./scripts/deploy-standalone.sh ~/threds-projectA`
2. Edit: `~/threds-projectA/config.yaml` → change port to 8081
3. Repeat for project B (port 8082), C (port 8083), etc.
4. Register each: `./scripts/register-claude-desktop.sh ~/threds-projectA projectA`

### Result
Claude sees multiple servers:
- `threds-projectA` → port 8081
- `threds-projectB` → port 8082
- `threds-projectC` → port 8083

## Troubleshooting

### Server Won't Start
```bash
# Check if port in use
lsof -i :8080

# Check database
sqlite3 ~/my-threds-server/data/threds.db ".tables"
```

### Claude Doesn't See Server
```bash
# 1. Verify connector details in Claude Desktop:
#    Name, URL (https://YOUR_HTTPS_HOST/mcp), OAuth fields blank

# 2. Restart Claude Desktop completely

# 3. Check server is running
curl http://127.0.0.1:8080/rpc
```

### Sample Data Failed
```bash
# Verify server is up
curl -I http://127.0.0.1:8080/rpc

# Check API key
source ~/my-threds-server/.env
echo $THREDS_API_KEY
```

## Common Tasks

### View Server Logs
```bash
cd ~/my-threds-server
./start.sh
# Logs appear in terminal
```

### Backup Database
```bash
cp ~/my-threds-server/data/threds.db ~/my-threds-server/data/threds.db.backup
```

### Reset Database
```bash
cd ~/my-threds-server
./stop.sh
rm data/threds.db
# Redeploy will create fresh database
```

### Change Port
```bash
# Edit config.yaml
vim ~/my-threds-server/config.yaml
# Change server.port value

# Restart server
cd ~/my-threds-server
./stop.sh && ./start.sh

# Re-register with Claude
./scripts/register-claude-desktop.sh ~/my-threds-server
```

### Uninstall
```bash
# 1. Stop server
cd ~/my-threds-server && ./stop.sh

# 2. Remove from Claude config
# Settings → Extensions/Connectors → Remove the threds connector

# 3. Delete directory
rm -rf ~/my-threds-server
```

## API Methods

Available via JSON-RPC at `http://127.0.0.1:8080/rpc`:

### Projects
- `create_project` - Create new project
- `list_projects` - List all projects
- `get_project` - Get project details
- `get_project_overview` - Full project state

### Records
- `create_record` - Create new record
- `update_record` - Update record content
- `transition` - Change record state
- `list_records` - List records with filters
- `get_record_ref` - Get record reference
- `search_records` - Full-text search

### Sessions
- `activate` - Activate record for reasoning
- `sync_session` - Sync changes
- `save_session` - Save current state
- `close_session` - End session
- `branch_session` - Create branch session

### Activity
- `get_recent_activity` - Recent events
- `get_record_history` - Record change history
- `get_active_sessions` - Active sessions for record

## MCP Client Registration

### Claude Desktop
Add via the UI:
1. Settings → Extensions/Connectors → Add Custom Connector
2. Name: `threds`
3. URL: `https://YOUR_HTTPS_HOST/mcp`
4. OAuth Client ID / Secret: leave blank (auth disabled for local)

### Claude Code CLI
```bash
claude-code config set mcp.servers.threds.url "http://127.0.0.1:8080/mcp"
```

### Cline (VS Code)
Add to Cline settings:
```json
{
  "threds": {
    "url": "http://127.0.0.1:8080/mcp"
  }
}
```

## Quick Tips

- **API Key:** Stored in `.env` if you enable auth
- **Localhost Only:** Default binding is 127.0.0.1 (secure)
- **Sample Data:** Includes 2 projects, 6 records (realistic examples)
- **Unique Names:** Use different connector names in the UI
- **Restart Claude:** Required after config changes
- **Backups:** Created automatically by registration script

## Getting Help

- **Scripts:** `cat scripts/README.md`
- **Deployment:** `cat ~/my-threds-server/README.md`
- **MCP Clients:** `cat ~/my-threds-server/REGISTER.md`
- **Architecture:** `cat docs/0118-deployment-scripts.md`
