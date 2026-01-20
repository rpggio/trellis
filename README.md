# threds

An MCP server providing record-based reasoning memory for design conversations.

> NOTE: Threds is pre-release software, under active development

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
