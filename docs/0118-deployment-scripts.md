# Deployment Scripts for Mac

*Created: 2025-01-18*

## Overview

A set of shell scripts for deploying, configuring, and registering standalone trellis servers on macOS. These scripts provide a streamlined experience for getting a trellis server running and connected to Claude Desktop.

## Scripts Created

### 1. `deploy-standalone.sh`

**Purpose:** Deploy a complete standalone server to a specified directory

**Key Features:**
- Compiles Go binary from source
- Generates secure API key (32-byte hex)
- Creates configuration files (YAML and .env)
- Initializes SQLite database with hashed API key
- Creates start/stop convenience scripts
- Generates comprehensive documentation

**Output Structure:**
```
deployment-dir/
├── bin/
│   └── trellis-server          # Compiled binary
├── data/
│   └── trellis.db              # SQLite database
├── config.yaml                # Server configuration
├── .env                       # Environment vars + API key
├── start.sh                   # Start server script
├── stop.sh                    # Stop server script
├── README.md                  # Deployment docs
└── REGISTER.md                # MCP client instructions
```

**Security:**
- API keys generated with OpenSSL (cryptographically random)
- Keys stored as SHA256 hash in database
- Plaintext key in .env (user responsible for securing)
- Default binding to 127.0.0.1 (localhost only)

### 2. `generate-sample-data.sh`

**Purpose:** Populate a running server with realistic sample data

**Sample Data Created:**

**Project 1: Mobile App Redesign**
- Root question: Navigation redesign approach
  - Conclusion: Adaptive bottom navigation
  - Follow-up: Notification badge strategy
- Root question: Color system selection
  - Deferred note: Contrast verification tooling (LATER state)

**Project 2: API Architecture Evolution**
- Root question: GraphQL vs REST API decision

**Characteristics:**
- 2 projects with meaningful descriptions
- 6 records total with parent-child relationships
- Record bodies: 1-3 paragraphs (realistic length per spec)
- Mix of types: question, conclusion, note
- Mix of states: OPEN, LATER
- Demonstrates hierarchical reasoning

**API Usage:**
- Uses JSON-RPC over HTTP
- Auth disabled for local deployments (no bearer header required)
- Validates server connectivity before generating data

### 3. `register-claude-desktop.sh`

**Purpose:** Print UI registration steps for Claude Desktop

**Features:**
- Validates deployment directory
- Reads server URL from `.env`
- Checks Claude Desktop version
- Prints exact fields to enter in the UI (https required)
- Generates unique server names with optional suffix

**Server Naming:**
- Pattern: `trellis-<suffix>`
- Default suffix: random 8-char hex
- Custom suffix: user-provided (e.g., `my-project`)

**Notes:**
- Remote HTTP MCP servers are added through Claude Desktop UI
- `claude_desktop_config.json` is for local stdio MCP servers only
- Claude Desktop UI only exposes OAuth fields (no custom headers)

### 4. `quick-start.sh`

**Purpose:** One-command setup from zero to working MCP server

**Interactive Workflow:**
1. Prompts for deployment directory (defaults to a random `/tmp` subdir)
2. Prompts for server port (defaults to a random free port)
3. Auto-generates a connector name (no prompt)
4. Asks about sample data generation
5. Executes full pipeline:
   - Deploy
   - Start (with health check)
   - Generate sample data (optional)
   - Print UI registration steps for Claude Desktop (https required)
6. Displays success banner with next steps

**Health Check:**
- Waits up to 30 seconds for server readiness
- Polls HTTP endpoint
- Fails gracefully with clear error message

**Process Model:**
- Server remains attached to the quick-start process
- Ctrl-C stops the server and exits the script

## Design Decisions

### Why Shell Scripts?

- **Zero dependencies** for Mac users (bash, curl, openssl pre-installed)
- **Simple deployment** model (copy files, no package managers)
- **Transparency** - users can read and understand every step
- **Easy customization** - straightforward to modify

### Why Standalone Deployments?

- **Isolation** - each project can have its own server
- **Portability** - entire deployment is self-contained
- **Simplicity** - no system-wide installation or services
- **Multiple projects** - run several servers simultaneously on different ports

### Sample Data Design

**Record content is intentionally realistic:**
- Questions reflect real design decision-making scenarios
- Conclusions include justification and trade-offs
- Bodies are 1-3 paragraphs (per API spec: "records are supposed to be 1-several paragraphs")
- Topics span UI/UX and technical architecture
- Demonstrates practical usage patterns

**Why these specific examples:**
1. **Mobile App Redesign** - demonstrates hierarchical reasoning with parent-child records
2. **API Architecture** - shows decision documentation for technical choices
3. **Mix of states** - illustrates workflow (OPEN work vs LATER deferred items)

### Security Considerations

**Current Model (Development/Local):**
- API key in plaintext .env file
- Localhost binding only
- No TLS
- Single tenant

**For Production:**
Would need:
- Secrets management (Vault, AWS Secrets Manager, etc.)
- TLS termination (reverse proxy)
- Rate limiting
- Request logging
- Multi-tenant isolation
- Database backups

### MCP Client Focus

**Primary target:** Claude Desktop on Mac

**Why:**
- Most common use case for individual users
- Can be fully automated on Mac
- Other clients (Cline, Claude Code) have similar patterns

**Extensibility:**
- REGISTER.md includes instructions for other clients
- HTTP MCP endpoint is client-agnostic
- Bearer token auth is supported; local deployments default to auth disabled

## Usage Patterns

### Single Project Setup

```bash
./scripts/quick-start.sh
# Use for a single project or personal experimentation
```

### Multiple Projects

```bash
# Deploy multiple servers
./scripts/deploy-standalone.sh ~/trellis/project-a
./scripts/deploy-standalone.sh ~/trellis/project-b
./scripts/deploy-standalone.sh ~/trellis/project-c

# Edit config.yaml for each to use different ports
# Then register each with unique names
./scripts/register-claude-desktop.sh ~/trellis/project-a project-a
./scripts/register-claude-desktop.sh ~/trellis/project-b project-b
./scripts/register-claude-desktop.sh ~/trellis/project-c project-c
```

### CI/CD Testing

```bash
# Deploy test instance
./scripts/deploy-standalone.sh /tmp/trellis-test
cd /tmp/trellis-test && ./start.sh

# Generate data for testing
./scripts/generate-sample-data.sh /tmp/trellis-test

# Run tests against server
# ...

# Cleanup
cd /tmp/trellis-test && ./stop.sh
rm -rf /tmp/trellis-test
```

## Future Enhancements

### Potential Additions

1. **Migration script** - Upgrade existing deployments to new versions
2. **Backup script** - Create timestamped backups of database
3. **Health check script** - Detailed server diagnostics
4. **Log rotation** - Manage log file sizes
5. **Multi-platform support** - Linux and Windows versions
6. **Docker deployment** - Container-based alternative
7. **systemd/launchd integration** - System service management

### Sample Data Expansion

Could add:
- More diverse project types (technical, creative, business)
- Deeper hierarchies (3-4 levels)
- More record types (analysis, decision, risk, opportunity)
- Branched session examples
- Conflict resolution examples
- More RESOLVED records with resolved_by links

### MCP Client Automation

Could support:
- Automatic Claude Code CLI registration
- Cline (VS Code) settings.json modification
- Detection of other MCP clients

## Technical Notes

### API Key Generation

```bash
# Generation
openssl rand -hex 32  # 256-bit key

# Hashing (for database)
echo -n "$API_KEY" | openssl dgst -sha256 -binary | xxd -p -c 256
```

### Server Health Check

Polls `http://127.0.0.1:8080/rpc` endpoint:
- Method: HEAD or POST
- Success: 2xx response
- Timeout: 30 seconds (configurable)

### JSON Manipulation

Priority order:
1. `jq` - preferred (install via `brew install jq`)
2. `python3` - fallback (pre-installed on Mac)
3. Manual instructions - if neither available

### Claude Desktop Config

Location: `~/Library/Application Support/Claude/claude_desktop_config.json`

This file is for **local stdio MCP servers only**. Remote HTTP MCP servers
must be added via the Claude Desktop UI (Settings → Extensions/Connectors →
Add Custom Connector).

## Testing

### Manual Test Procedure

```bash
# 1. Deploy
./scripts/deploy-standalone.sh /tmp/test-deploy

# 2. Verify structure
ls -la /tmp/test-deploy
# Should see: bin/, data/, config.yaml, .env, start.sh, stop.sh

# 3. Start server
cd /tmp/test-deploy && ./start.sh &

# 4. Wait and test connectivity
sleep 3
source /tmp/test-deploy/.env
curl -X POST http://127.0.0.1:8080/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"list_projects","params":{}}'

# 5. Generate sample data
./scripts/generate-sample-data.sh /tmp/test-deploy

# 6. Query sample data
curl -X POST http://127.0.0.1:8080/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"get_project_overview","params":{}}'

# 7. Register (dry run - backup and inspect)
./scripts/register-claude-desktop.sh /tmp/test-deploy test
# Manually verify backup was created

# 8. Cleanup
cd /tmp/test-deploy && ./stop.sh
rm -rf /tmp/test-deploy
```

## Documentation Structure

```
trellis/
├── README.md                          # Updated with quick start section
├── scripts/
│   ├── README.md                      # Comprehensive scripts documentation
│   ├── deploy-standalone.sh           # Deployment script
│   ├── generate-sample-data.sh        # Sample data generation
│   ├── register-claude-desktop.sh     # Claude Desktop registration
│   └── quick-start.sh                 # All-in-one setup
└── docs/
    └── 0118-deployment-scripts.md     # This document
```

Each deployment also generates:
```
deployment-dir/
├── README.md                          # How to use this deployment
└── REGISTER.md                        # MCP client registration instructions
```

## Conclusion

These scripts provide a complete, user-friendly deployment experience for trellis on macOS. They prioritize:

- **Simplicity** - One command to get started
- **Safety** - Backups, validation, clear error messages
- **Flexibility** - Support for multiple deployments
- **Documentation** - Every script includes usage instructions
- **Realism** - Sample data reflects actual use cases

The result is a smooth path from "I want to try trellis" to "I'm having a design conversation with Claude using my reasoning memory" in under 5 minutes.
