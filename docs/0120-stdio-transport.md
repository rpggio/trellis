# Stdio Transport Implementation

## Overview

This document describes the stdio transport implementation for threds-mcp, which enables local development and integration with MCP clients like Claude Desktop.

## Architecture

### Transport Modes

1. **Stdio Transport** (`THREDS_TRANSPORT=stdio`)
   - Uses stdin/stdout for JSON-RPC communication
   - Newline-delimited JSON messages
   - Blocking operation (server.Run() blocks until stdin closes)
   - No HTTP server
   - Auth disabled (single-tenant, local dev only)
   - Session IDs via `_meta.session_id` in request params

2. **HTTP Transport** (`THREDS_TRANSPORT=http`)
   - Uses Server-Sent Events (SSE) over HTTP
   - Optional stateful sessions with timeout
   - Supports auth via bearer tokens
   - Session IDs via `Mcp-Session-Id` HTTP header

### Middleware Behavior

#### Auth Middleware
- **HTTP mode**: Validates bearer token from `Authorization` header
- **Stdio mode**: Skipped entirely (uses `noAuthMiddleware` with default tenant)

#### Session Middleware
- **HTTP mode**: Extracts session ID from `Mcp-Session-Id` header
- **Stdio mode**: Extracts session ID from `params._meta.session_id`
- Both: Injects session ID into request context via `sessionIDKey`

## Implementation Details

### Transport Selection

Transport is selected based on:
1. Environment variable `THREDS_TRANSPORT`
2. YAML config `transport.mode`
3. Default: `stdio`

### Code Structure

**Config** (`internal/config/config.go:20-22`):
```go
type TransportConfig struct {
    Mode string `yaml:"mode"` // "stdio" or "http"
}
```

**MCP Server** (`internal/mcp/server.go:58-63`):
```go
type Config struct {
    Services      Services
    Resolver      TenantResolver
    AuthEnabled   bool
    TransportMode string // "stdio" or "http"
    Logger        *slog.Logger
}
```

**Main Entry Point** (`cmd/server/main.go:82-86`):
```go
// Branch based on transport mode
if cfg.Transport.Mode == "stdio" {
    runStdioMode(logger, mcpServer, cfg.Log.Level)
} else {
    runHTTPMode(logger, mcpServer, cfg.Server.Host, cfg.Server.Port)
}
```

### Graceful Shutdown

- **HTTP mode**: Signal handler shuts down HTTP server with 5s timeout
- **Stdio mode**: Signal handler cancels context, causing server.Run() to exit

## Usage Examples

### Local Development (Stdio)

```bash
make dev
# Server runs on stdio, connect via MCP client
```

### Local Development (HTTP)

```bash
make dev-http
# Server runs at http://localhost:8080/mcp
```

### Production (HTTP)

```bash
THREDS_TRANSPORT=http THREDS_AUTH_ENABLED=true ./threds-mcp
```

### Claude Desktop Integration

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS):

```json
{
  "mcpServers": {
    "threds": {
      "command": "/usr/local/bin/threds-mcp",
      "env": {
        "THREDS_DB_PATH": "/Users/you/.threds/threds.db",
        "THREDS_LOG_LEVEL": "info"
      }
    }
  }
}
```

No need to specify `THREDS_TRANSPORT` since stdio is the default.

## Security Considerations

1. **Stdio mode is single-tenant**: No multi-tenancy support, uses default tenant "default"
2. **Stdio mode disables auth**: Appropriate for local dev only
3. **Session IDs in metadata**: Less discoverable than headers, but standard for stdio MCP
4. **Log output**: Stdio mode logs to stderr to avoid corrupting stdin/stdout JSON-RPC

## Testing

### Critical Requirement: stdout Must Be Clean JSON

The MCP stdio transport requires **stdout to contain only JSON-RPC messages**. All logging and diagnostic output must go to stderr.

**Implementation**: The server detects stdio mode and routes logs to stderr (see `cmd/server/main.go:34-40`).

### Quick Sanity Test (Built-in)

```bash
./test/stdio_test.sh
```

Our basic test suite validates:
- Protocol initialization handshake
- stdout contains only valid JSON (no log contamination)
- Server capabilities are properly advertised
- Logs are separated to stderr

### Comprehensive Compliance Testing (Recommended for CI/CD)

Use the official `mcp-validation` tool for thorough protocol compliance testing:

```bash
# Install
npm install -g @modelcontextprotocol/mcp-validation

# Run full validation with JSON report
mcp-validate \
  --transport stdio \
  --env THREDS_DB_PATH=:memory: \
  --json-report compliance-report.json \
  --debug \
  -- ./bin/threds-mcp
```

Features:
- ✅ Protocol compliance validation (JSON-RPC 2.0, MCP handshake)
- ✅ Security analysis via mcp-scan integration
- ✅ JSON report output for CI/CD pipelines
- ✅ Supports stdio, HTTP, and SSE transports

**References**:
- [mcp-validation](https://github.com/RHEcosystemAppEng/mcp-validation) - Automated compliance testing
- [MCP Server Testing Tools](https://testomat.io/blog/mcp-server-testing-tools/)

### Interactive Testing (Development)

Use MCP Inspector for manual testing and debugging:

```bash
# Launch interactive inspector
npx @modelcontextprotocol/inspector ./bin/threds-mcp

# Then open http://localhost:6274 in your browser
```

**Note**: Inspector runs a web UI and is **not suitable for automated testing**.

**Reference**: [MCP Inspector](https://modelcontextprotocol.io/docs/tools/inspector)

### Scripted Tool Testing

Use `@wong2/mcp-cli` for automated tool invocation in scripts:

```bash
# Install
npm install -g @wong2/mcp-cli

# Create config
cat > test-config.json <<EOF
{
  "mcpServers": {
    "threds": {
      "command": "./bin/threds-mcp",
      "env": {
        "THREDS_DB_PATH": ":memory:"
      }
    }
  }
}
EOF

# Run commands non-interactively
npx @wong2/mcp-cli --config test-config.json call-tool threds:create_record --args '{...}'
```

**Reference**: [@wong2/mcp-cli](https://github.com/wong2/mcp-cli)

### HTTP Transport Tests

All existing functional tests use HTTP transport and verify:
- Authentication with bearer tokens
- Session management via headers
- Full MCP protocol compliance

```bash
make test-functional  # HTTP transport
```

### Common Issues

**Issue**: "is not valid JSON" error from MCP Inspector

**Cause**: Logs are being written to stdout instead of stderr.

**Solution**: Ensure `THREDS_TRANSPORT=stdio` is set, which triggers stderr logging.

## Migration Path

New default is stdio for simplicity. For HTTP deployments:

1. Set `THREDS_TRANSPORT=http` in environment or config
2. Configure HTTP-specific settings (host, port, auth)
3. Deploy with proper auth enabled

## Design Decisions

### Why Auth is Disabled in Stdio

- Stdio is inherently process-to-process communication
- No HTTP headers means no standard auth mechanism
- Local development doesn't require multi-tenant auth
- Production deployments should use HTTP transport with proper auth

### Why Session IDs Use Metadata

- HTTP headers aren't available in stdio
- MCP SDK supports metadata via `params._meta`
- Consistent pattern across other MCP implementations
- Middleware abstracts the difference transparently

### Why Stdio is the Default

- Simplicity: MCP clients primarily use stdio
- Claude Desktop integration out of the box
- No network configuration required
- Perfect for local development and personal use
- HTTP requires explicit opt-in for production deployments

## Future Enhancements

Potential improvements for stdio support:

1. **Parameterized functional tests**: Run all tests for both transports
2. **Stdio test client**: Helper for testing stdio mode programmatically
3. **Better logging**: Structured logging to stderr with transport mode indicator
4. **Health check**: Stdio equivalent of HTTP /health endpoint

## References

- MCP Specification: https://modelcontextprotocol.io
- Go SDK: github.com/modelcontextprotocol/go-sdk
- Core requirements: `docs/0116-core-requirements.md`
- Scenario analysis: `docs/0122-scenario-analysis.md`
