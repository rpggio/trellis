# Threds MCP Deployment Scripts

This directory contains scripts for deploying and configuring standalone threds-mcp servers on Mac.

## Scripts Overview

### 1. `deploy-standalone.sh`

Deploys a complete standalone threds-mcp server to a specified directory.

**Usage:**
```bash
./deploy-standalone.sh <deployment_directory>
```

**Example:**
```bash
./deploy-standalone.sh ~/threds-deployments/my-project
```

**What it does:**
- Compiles the Go server binary
- Creates directory structure (bin/, data/)
- Generates a unique API key (auth disabled by default for local)
- Creates configuration files (config.yaml, .env)
- Initializes SQLite database with API key
- Creates start/stop scripts
- Generates documentation (README.md, REGISTER.md)

**Output:**
- Deployment directory with everything needed to run the server
- API key printed to console and saved in `.env`
- Instructions for starting and registering the server

### 2. `generate-sample-data.sh`

Populates a running threds server with realistic sample data.

**Usage:**
```bash
./generate-sample-data.sh <deployment_directory>
```

**Example:**
```bash
# First, start the server
cd ~/threds-deployments/my-project
./start.sh

# Then generate sample data
cd /path/to/threds-mcp
./scripts/generate-sample-data.sh ~/threds-deployments/my-project
```

**What it does:**
- Verifies server is running
- Creates 2 sample projects:
  - "Mobile App Redesign" - UI/UX design exploration
  - "API Architecture Evolution" - Technical architecture planning
- Creates 6 sample records with realistic content:
  - Questions (design problems to solve)
  - Conclusions (resolved decisions)
  - Notes (supporting information)
- Records contain 1-3 paragraph bodies (realistic length)
- Mix of OPEN and LATER states
- Demonstrates parent-child hierarchies

**Sample data includes:**
- Navigation redesign questions and conclusions
- Color system design exploration
- GraphQL vs REST API decision
- Notification badge strategy
- Deferred research tasks

### 3. `register-claude-desktop.sh`

Prints the UI steps to register the threds server with Claude Desktop on Mac.

**Usage:**
```bash
./register-claude-desktop.sh <deployment_directory> [server_name_suffix]
```

**Examples:**
```bash
# Auto-generate random suffix
./register-claude-desktop.sh ~/threds-deployments/my-project

# Use custom suffix
./register-claude-desktop.sh ~/threds-deployments/my-project my-project
```

**What it does:**
- Validates deployment directory
- Reads configuration from `.env`
- Checks Claude Desktop version
- Prints the exact values to enter in the Claude Desktop UI (https required)

**Server naming:**
- Default: `threds-<random_hex>` (e.g., `threds-a3f2c1d9`)
- Custom: `threds-<your_suffix>` (e.g., `threds-my-project`)

**Requirements:**
- macOS only
- Claude Desktop installed (v1.1.0+)
- Claude Desktop registers remote HTTP MCP servers via the UI, not the config file
- Claude Desktop requires https URLs for connectors (use a tunnel or reverse proxy)
- The UI only supports OAuth fields; local deployments disable auth by default

## Complete Workflow

Here's a typical workflow to set up a new threds server:

```bash
# 1. Deploy a standalone server
./scripts/deploy-standalone.sh ~/threds-servers/design-project

# 2. Start the server
cd ~/threds-servers/design-project
./start.sh

# Wait a moment for server to start...

# 3. Generate sample data (optional but recommended)
cd /path/to/threds-mcp
./scripts/generate-sample-data.sh ~/threds-servers/design-project

# 4. Print Claude Desktop UI steps
./scripts/register-claude-desktop.sh ~/threds-servers/design-project design-project

# 5. Add the connector in Claude Desktop UI

# 6. Restart Claude Desktop

# 7. Test in Claude
# Open Claude Desktop and ask:
# "Can you list my threds projects?"
```

## Multiple Deployments

You can run multiple threds servers simultaneously for different projects:

```bash
# Deploy multiple servers on different ports
./deploy-standalone.sh ~/threds-servers/project-a
# Edit ~/threds-servers/project-a/config.yaml to use port 8080

./deploy-standalone.sh ~/threds-servers/project-b
# Edit ~/threds-servers/project-b/config.yaml to use port 8081

./deploy-standalone.sh ~/threds-servers/project-c
# Edit ~/threds-servers/project-c/config.yaml to use port 8082

# Print UI steps for each unique name
./register-claude-desktop.sh ~/threds-servers/project-a project-a
./register-claude-desktop.sh ~/threds-servers/project-b project-b
./register-claude-desktop.sh ~/threds-servers/project-c project-c

# Add each connector in Claude Desktop UI
```

Claude will see all registered servers as separate MCP servers.

## MCP Registration for Other Clients

The scripts focus on Claude Desktop, but you can manually register with other clients:
Local deployments default to auth disabled; omit Authorization headers unless you
enable auth in `config.yaml` or `THREDS_AUTH_ENABLED`.

### Cline (VS Code)

Edit Cline settings and add:
```json
{
  "threds": {
    "url": "http://127.0.0.1:8080/mcp"
  }
}
```

### Claude Code CLI

```bash
claude-code config set mcp.servers.threds.url "http://127.0.0.1:8080/mcp"
```

### Generic MCP Client

Connection details:
- **URL:** `http://127.0.0.1:8080/mcp`
- **Authentication:** Disabled for local deployments by default

## Troubleshooting

### Server won't start

```bash
# Check if port is in use
lsof -i :8080

# Check server logs
cd ~/threds-servers/my-project
./start.sh
# Logs will appear in terminal
```

### Claude Desktop doesn't see the server

1. Verify the connector details in Claude Desktop:
   - Name
   - URL (https://YOUR_HTTPS_HOST/mcp)
   - OAuth fields left blank (auth disabled for local)

2. Restart Claude Desktop completely (quit and reopen)

3. Check server is running:
   ```bash
   curl http://127.0.0.1:8080/health
   ```

### Sample data generation fails

```bash
# Verify server is running
curl -X POST http://127.0.0.1:8080/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"list_projects","params":{}}'
```

### API key issues (if auth is enabled)

```bash
# Regenerate API key
cd ~/threds-servers/my-project
source .env

# Generate new key
NEW_KEY=$(openssl rand -hex 32)
KEY_HASH=$(echo -n "$NEW_KEY" | openssl dgst -sha256 -binary | xxd -p -c 256)

# Update database
sqlite3 data/threds.db "UPDATE api_keys SET key_hash = '$KEY_HASH';"

# Update .env file
sed -i '' "s/THREDS_API_KEY=.*/THREDS_API_KEY=$NEW_KEY/" .env

# Re-register with Claude
cd /path/to/threds-mcp
./scripts/register-claude-desktop.sh ~/threds-servers/my-project
```

## Security Notes

- API keys are stored in plaintext in `.env` files
- Keys are hashed (SHA256) in the database
- Local deployments disable auth by default (set `THREDS_AUTH_ENABLED=true` to enable)
- Default deployment binds to `127.0.0.1` (localhost only)
- For remote access, update `config.yaml` and add proper authentication/TLS
- Keep `.env` files secure and don't commit them to version control

## Customization

### Change server port

Edit `config.yaml` in the deployment directory:
```yaml
server:
  host: "127.0.0.1"
  port: 8081  # Change this
```

Then restart the server and update the connector settings in Claude Desktop.

### Change log level

Edit `.env` or `config.yaml`:
```yaml
log:
  level: "debug"  # Options: debug, info, warn, error
```

### Use different database location

Edit `config.yaml`:
```yaml
db:
  path: "/path/to/custom/location/threds.db"
```

## Uninstalling

To remove a deployment:

```bash
# 1. Stop the server
cd ~/threds-servers/my-project
./stop.sh

# 2. Remove registration from Claude Desktop
# Settings → Extensions/Connectors → Remove the threds connector

# 3. Delete deployment directory
rm -rf ~/threds-servers/my-project
```

## Development vs. Production

These scripts create **development/local** deployments:
- No TLS/HTTPS
- No rate limiting
- No request logging
- Binds to localhost only

For production deployment, consider:
- Reverse proxy (nginx, Caddy) with TLS
- API rate limiting
- Structured logging and monitoring
- Database backups
- Secrets management (not .env files)
- Network isolation
