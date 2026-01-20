# threds-mcp

Threds is a record-based memory system for design reasoning across chats. `threds-mcp` is an HTTP MCP server that stores projects/records/sessions in SQLite and lets agents sync down the minimum context they need to keep working.

> NOTE: Pre-release software, under active development.

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

1. Orient: `get_project_overview` (root records + open sessions + tick gap warnings).
2. Find: `search_records` / `list_records` to locate the next record to work on.
3. Reason: `activate` to load the record’s context into the current session.
4. Write: `create_record`, `update_record`, `transition` (updates require activation; `update_record` may return a conflict unless `force=true`).
5. Sync/finish: `sync_session` as needed, then `save_session` and `close_session`.
6. Branch: `branch_session` to continue a focused subtopic in a new chat session.

References: `docs/0116-core-requirements.md`, `docs/0117-scenario-analysis.md`.

## MCP Endpoint

- **MCP URL:** `http://127.0.0.1:8080/mcp`
- **Health:** `http://127.0.0.1:8080/health`
- **Auth:** enabled by default; send `Authorization: Bearer <api-key>` (or use `make dev` to disable auth locally).

## Make Targets

```text
make help             - Show this help message
make dev              - Run dev server (auth disabled, localhost:8080)
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

- `THREDS_CONFIG_PATH`: path to YAML config file (optional)
- `THREDS_SERVER_HOST`: server host (default `0.0.0.0`)
- `THREDS_SERVER_PORT`: server port (default `8080`)
- `THREDS_DB_PATH`: SQLite database path (default `threds.db`)
- `THREDS_LOG_LEVEL`: `debug`, `info`, `warn`, `error` (default `info`)
- `THREDS_AUTH_ENABLED`: `true` or `false` (default `true`)

Sample YAML:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
db:
  path: "threds.db"
log:
  level: "info"
auth:
  enabled: true
```

## Run

```bash
make dev   # local dev (auth disabled)
make run   # go run ./cmd/server
```

Build a binary:

```bash
make build
./bin/threds-mcp
```

## Tests

```bash
make test
```

## MCP Tools (Current)

- Projects: `create_project`, `list_projects`, `get_project`
- Orientation: `get_project_overview`, `search_records`, `list_records`, `get_record_ref`
- Sessions: `activate`, `sync_session`, `save_session`, `close_session`, `branch_session`
- Mutations: `create_record`, `update_record`, `transition`
- History: `get_record_history`, `get_recent_activity`, `get_active_sessions`, `get_record_diff` (placeholder)
- Utility: `ping`
