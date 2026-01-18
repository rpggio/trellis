# threds

An MCP server providing record-based reasoning memory for design conversations.

## Quick Start (Mac)

For the fastest setup experience on macOS:

```bash
./scripts/quick-start.sh
```

This will deploy, start, populate with sample data, and register the server with Claude Desktop in one go.

Or follow individual steps:

```bash
# 1. Deploy a standalone server
./scripts/deploy-standalone.sh ~/my-threds-server

# 2. Start the server
cd ~/my-threds-server && ./start.sh

# 3. Generate sample data (optional)
./scripts/generate-sample-data.sh ~/my-threds-server

# 4. Register with Claude Desktop
./scripts/register-claude-desktop.sh ~/my-threds-server
```

See [scripts/README.md](scripts/README.md) for detailed documentation.

## Configuration

Configuration can be provided via environment variables or a YAML file.

Environment variables:

- `THREDS_CONFIG_PATH`: path to YAML config file (optional)
- `THREDS_SERVER_HOST`: server host (default `0.0.0.0`)
- `THREDS_SERVER_PORT`: server port (default `8080`)
- `THREDS_DB_PATH`: SQLite database path (default `threds.db`)
- `THREDS_LOG_LEVEL`: `debug`, `info`, `warn`, `error` (default `info`)

Sample YAML:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
db:
  path: "threds.db"
log:
  level: "info"
```

## Run

```bash
go run ./cmd/server
```

## Tests

Run unit tests:

```bash
go test ./internal/domain/... -v
```

Run SQLite repository tests:

```bash
go test ./internal/sqlite -v -cover
```

Run integration tests (services + SQLite):

```bash
go test ./test/integration -v
```

Run functional HTTP tests:

```bash
go test ./test/functional -v
```
