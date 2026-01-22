# Trellis Memory

Trellis Memory is a record-based memory system for complex reasoning across chats. `trellis` is an MCP server that stores projects/records/sessions in SQLite and lets agents sync down the minimum context they need to keep working. Supports stdio (default) and HTTP/SSE transports.

> NOTE: Pre-release software, under active development. Available soon!

## Purpose

- Capture design work as **records** (questions, conclusions, notes, etc.) so it survives across chat threads.
- Make “reasoning context” explicit via **activation** instead of dumping the whole project into a prompt.
- Track “one chat’s work” as a **session** so clients can detect staleness and avoid accidental conflicts.

## Core Model

- **Project**: container for records; has a monotonic logical clock (**tick**) that increments on every write.
- **Record**: `{id, type, title, summary, body, state}` with parent/child hierarchy and `related[]` links.
- **Workflow states**: `OPEN | LATER | RESOLVED | DISCARDED`.
- **Record reference**: lightweight pointer (no body) used for browsing/search results.
- **Session**: a chat’s connection to the project, tracking activated records and the last synced tick.

**Activation** (`activate`) returns a context bundle designed for reasoning:
- Target record (full)
- Parent record (full, if present)
- Open children (full)
- Other children + grandchildren (references; depth-limited)

## Workflow (Typical MCP Client / Agent)

1. Orient: `get_project_overview` (root records + open sessions + tick gap signals).
2. Find: `search_records` / `list_records` to locate the next record to work on.
3. Reason: `activate` to load the record’s context into the current session.
4. Persist (on user request): `create_record`, `update_record`, `transition` (updates require activation; `update_record` may return a conflict unless `force=true`).
5. Checkpoint/finish: `sync_session` as needed; call `save_session` only when the user requests a checkpoint; `close_session` when done.
6. New chat: in response to the user’s first message, orient (`get_project_overview`) and `activate` the record you want to work on.

References: `docs/0116-core-requirements.md`, `docs/0122-scenario-analysis.md`.

## MCP Endpoints

### Stdio Transport (Default)

The stdio transport uses stdin/stdout for JSON-RPC communication:

```bash
make dev              # Start server (stdio is default)
# Connect via MCP client (e.g., Claude Desktop, MCP Inspector)
```

**Stdio mode characteristics:**
- Default mode for simplicity and MCP client compatibility
- Auth disabled (single-tenant with default tenant ID)
- Session IDs passed via `_meta.session_id` in request params
- Suitable for local development and Claude Desktop integration

### HTTP Transport

The HTTP transport uses Server-Sent Events over HTTP:

- **MCP URL:** `http://127.0.0.1:8080/mcp`
- **Health:** `http://127.0.0.1:8080/health`
- **Auth:** enabled by default; send `Authorization: Bearer <api-key>`
- **Session ID:** send via `Mcp-Session-Id` header

```bash
make dev-http         # Start server in HTTP mode (auth disabled)
make run              # Production mode (auth enabled)
```

## Make Targets

```text
make help             - Show this help message
make dev              - Run dev server (stdio, auth disabled)
make dev-http         - Run dev server (HTTP, auth disabled, localhost:8080)
make test             - Run all tests
make test-unit        - Run unit tests
make test-integration - Run integration tests
make test-functional  - Run functional tests
make build            - Build the binary
make run              - Run the server with default config
make clean            - Remove build artifacts
```

## Configuration

Configuration can be provided via environment variables or a YAML file.

Environment variables:

- `TRELLIS_TRANSPORT`: transport mode (`stdio` or `http`, default `stdio`)
- `TRELLIS_CONFIG_PATH`: path to YAML config file (optional)
- `TRELLIS_SERVER_HOST`: server host (default `0.0.0.0`, HTTP mode only)
- `TRELLIS_SERVER_PORT`: server port (default `8080`, HTTP mode only)
- `TRELLIS_DB_PATH`: SQLite database path (default `trellis.db`)
- `TRELLIS_LOG_LEVEL`: `debug`, `info`, `warn`, `error` (default `info`)
- `TRELLIS_AUTH_ENABLED`: `true` or `false` (default `true`, HTTP mode only)

Sample YAML:

```yaml
# Stdio is default, only specify if using HTTP
transport:
  mode: "http"
server:
  host: "0.0.0.0"
  port: 8080
db:
  path: "trellis.db"
log:
  level: "info"
auth:
  enabled: true  # Only applies to HTTP mode
```

## Using with MCP Clients

### Claude Desktop (Stdio)

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "trellis": {
      "command": "/path/to/trellis",
      "env": {
        "TRELLIS_DB_PATH": "/path/to/trellis.db"
      }
    }
  }
}
```

Stdio is the default transport, so no `TRELLIS_TRANSPORT` needed.

### HTTP Clients

Use the HTTP endpoint with bearer token:

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -H "Mcp-Session-Id: session-123" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

### Session ID Handling

- **HTTP transport**: Pass session ID via `Mcp-Session-Id` header
- **Stdio transport**: Pass session ID via `_meta.session_id` in params:

```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "create_record",
    "arguments": { "title": "Example" },
    "_meta": { "session_id": "session-123" }
  },
  "id": 1
}
```

## Run

```bash
make dev       # local dev (stdio, auth disabled)
make dev-http  # local dev HTTP (auth disabled)
make run       # run with default config (stdio, auth disabled)
```

Build a binary:

```bash
make build
./bin/trellis  # Runs with stdio (default)

# Or specify HTTP for production:
TRELLIS_TRANSPORT=http TRELLIS_AUTH_ENABLED=true ./bin/trellis
```

## Tests

```bash
make test
```

## MCP Tools (Current)

- Projects: `create_project`, `list_projects`, `get_project`
- Orientation: `get_project_overview`, `search_records`, `list_records`, `get_record_ref`
- Sessions: `activate`, `sync_session`, `save_session`, `close_session`
- Mutations: `create_record`, `update_record`, `transition`
- History: `get_record_history`, `get_recent_activity`, `get_active_sessions`, `get_record_diff` (placeholder)
- Utility: `ping`
