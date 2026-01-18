# threds

An MCP server providing record-based reasoning memory for design conversations.

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
