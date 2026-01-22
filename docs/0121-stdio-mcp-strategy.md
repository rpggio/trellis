# Stdio MCP Strategy: Making Every Build Compliant

## Executive Summary

The "SSE connection not established" error in MCP Inspector reveals a fundamental issue: the inspector expects SSE-based HTTP communication but receives stdio. This isn't a server bug—it's a **tooling mismatch**. The server appears to implement stdio correctly per the Go SDK, but we lack proper validation infrastructure to catch protocol issues before deployment.

This plan addresses three goals:

1. Confirm the Go SDK is used correctly (it mostly is)
2. Build validation into every build cycle
3. Create a Claude Code skill for MCP protocol awareness

---

## Part 1: Go SDK Usage Audit

### Current State Assessment

**What's Working:**
- Server uses `sdkmcp.NewServer()` with proper `Implementation` and `ServerOptions`
- Tools registered via `sdkmcp.AddTool()` with typed input structs (generic pattern)
- Stdio mode uses `sdkmcp.StdioTransport{}` directly
- HTTP mode uses `sdkmcp.NewStreamableHTTPHandler()`
- Logging correctly routes to stderr in stdio mode

**SDK Version:** `github.com/modelcontextprotocol/go-sdk v1.2.0` — current as of Jan 2026

**Potential Issues to Verify:**

| Area | Current Code | Concern | Action |
|------|--------------|---------|--------|
| go.mod Go version | `go 1.25.5` | Go 1.25.5 doesn't exist (typo?) | Verify/fix to `go 1.22.x` or `1.23.x` |
| Tool handler returns | Returns `(*sdkmcp.CallToolResult, T, error)` | SDK expects specific return patterns | Verify against SDK examples |
| Middleware ordering | Auth → Session | Order matters for context population | Appears correct |
| Context propagation | Uses `getTenantID(ctx)` etc. | Middleware must inject values | Verify middleware actually runs |

### Recommended SDK Verification Tasks

```
[ ] 1. Fix go.mod version (likely typo)
[ ] 2. Compare tool registration against SDK examples at:
       https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp
[ ] 3. Add a "golden path" integration test using CommandTransport
[ ] 4. Verify server responds to `tools/list` and `tools/call` via SDK client
```

---

## Part 2: Build-Time Validation Pipeline

### The Core Problem

The coding agent writes MCP server code without immediate feedback on protocol compliance. By the time you run MCP Inspector manually, you've already committed non-compliant code.

### Proposed Validation Stack

```
┌─────────────────────────────────────────────────────────────┐
│                    Makefile Targets                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  make build          → build binary                         │
│  make test           → unit + integration tests             │
│  make validate-mcp   → NEW: protocol compliance check       │
│  make ci             → build + test + validate-mcp          │
│                                                             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│               Protocol Validation Layer                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Stdout Hygiene Check (grep-based lint)                  │
│     - Fails if fmt.Print* appears in server code            │
│     - Ensures all logging goes to stderr                    │
│                                                             │
│  2. SDK Client Integration Test (Go)                        │
│     - Spawns server via CommandTransport                    │
│     - Calls initialize, tools/list, tools/call              │
│     - Validates JSON-RPC response structure                 │
│                                                             │
│  3. External Validation (optional, for CI)                  │
│     - mcp-validation tool for comprehensive check           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Implementation Details

#### 2.1 Stdout Hygiene Lint Rule

Add to Makefile:

```makefile
## lint-stdout: Verify no fmt.Print statements in server code
lint-stdout:
	@echo "Checking for stdout pollution..."
	@if grep -rn --include='*.go' 'fmt\.Print\|println(' ./cmd ./internal/mcp 2>/dev/null | grep -v '_test.go'; then \
		echo "ERROR: Found fmt.Print/println in server code. Use slog to stderr instead."; \
		exit 1; \
	fi
	@echo "✓ No stdout pollution detected"
```

#### 2.2 SDK Client Integration Test

Create `test/integration/stdio_protocol_test.go`:

```go
package integration_test

import (
    "context"
    "testing"
    "time"

    sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/stretchr/testify/require"
)

func TestStdioProtocolCompliance(t *testing.T) {
    // Build the server first
    // (or use pre-built binary)

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Spawn server as subprocess
    transport, err := sdkmcp.NewCommandTransport(ctx, sdkmcp.CommandTransportConfig{
        Command: "./bin/trellis",
        Env: []string{
            "TRELLIS_TRANSPORT=stdio",
            "TRELLIS_DB_PATH=:memory:",
        },
    })
    require.NoError(t, err)
    defer transport.Close()

    // Create client
    client := sdkmcp.NewClient(&sdkmcp.Implementation{
        Name:    "test-client",
        Version: "1.0.0",
    }, nil)

    // Connect and initialize
    session, err := client.Connect(ctx, transport)
    require.NoError(t, err)
    defer session.Close()

    // Verify initialize response
    require.NotNil(t, session.ServerInfo())
    require.Equal(t, "trellis", session.ServerInfo().Name)

    // Test tools/list
    tools, err := session.ListTools(ctx, nil)
    require.NoError(t, err)
    require.Greater(t, len(tools.Tools), 15)

    // Find and call a specific tool
    var hasPing bool
    for _, tool := range tools.Tools {
        if tool.Name == "ping" {
            hasPing = true
            break
        }
    }
    require.True(t, hasPing, "ping tool should exist")

    // Test tools/call
    result, err := session.CallTool(ctx, "ping", nil)
    require.NoError(t, err)
    require.False(t, result.IsError)
    require.Contains(t, result.Content[0].Text, "pong")
}
```

#### 2.3 Updated Makefile

```makefile
## validate-mcp: Run MCP protocol compliance checks
validate-mcp: build lint-stdout
	@echo "Running protocol compliance tests..."
	TRELLIS_DB_PATH=:memory: go test ./test/integration -run TestStdioProtocol -v

## ci: Full CI pipeline
ci: build test validate-mcp
	@echo "✓ CI pipeline complete"
```

#### 2.4 Pre-Commit Hook (Optional)

Create `.claude/hooks/pre-commit.sh`:

```bash
#!/bin/bash
# Quick protocol sanity check before commit

make lint-stdout || exit 1

echo "Running quick stdio test..."
./test/stdio_test.sh || exit 1

echo "✓ Pre-commit checks passed"
```

---

## Part 3: Claude Code MCP Skill

### Why a Skill?

Claude Code agents repeatedly make the same MCP protocol mistakes because they lack persistent knowledge of:

- MCP transport modes and their requirements
- stdout/stderr hygiene for stdio transport
- SDK patterns vs. "invented" JSON-RPC
- Validation workflow expectations

A skill embeds this knowledge into every Claude Code session working on MCP servers.

### Skill Design

**Location:** `.claude/skills/mcp-protocol/`

**Structure:**
```
mcp-protocol/
├── SKILL.md           # Main skill document
├── examples/
│   ├── tool-registration.go
│   ├── stdio-transport.go
│   └── integration-test.go
└── checklists/
    ├── before-implementing.md
    └── before-committing.md
```

### SKILL.md Content

```markdown
# MCP Protocol Development Skill

## Core Rules (Non-Negotiable)

### 1. SDK-First Development
ALWAYS use the official Go SDK (`github.com/modelcontextprotocol/go-sdk/mcp`).

**DO:**
- `sdkmcp.NewServer()` for server creation
- `sdkmcp.AddTool()` for tool registration
- `sdkmcp.StdioTransport{}` for stdio mode
- `sdkmcp.CommandTransport` for testing

**NEVER:**
- Manual JSON-RPC parsing
- Custom stdin/stdout framing
- Inventing methods beyond MCP spec (tools/list, tools/call, etc.)

### 2. Stdout is Sacred in Stdio Mode
In stdio transport, stdout IS the wire protocol.

**Rule:** Server MUST NOT write ANYTHING to stdout except valid JSON-RPC messages.

**Implementation:**
```go
// In main(), BEFORE any logging:
if transportMode == "stdio" {
    log.SetOutput(os.Stderr)
    // Or for slog:
    slog.New(slog.NewTextHandler(os.Stderr, nil))
}
```

**Banned in server code (non-test):**
- `fmt.Print()`, `fmt.Println()`, `fmt.Printf()`
- `println()`
- Any direct `os.Stdout.Write()`

### 3. Validate Before Commit
Every change to MCP server code requires:

1. `make lint-stdout` passes
2. `make validate-mcp` passes
3. Manual test with one of:
   - `./test/stdio_test.sh`
   - MCP Inspector (for interactive debugging only)

### 4. Tool Registration Pattern
Use the generic `AddTool` pattern with typed structs:

```go
type MyToolInput struct {
    Name string `json:"name" jsonschema:"required"`
    Age  int    `json:"age,omitempty"`
}

sdkmcp.AddTool(server, &sdkmcp.Tool{
    Name:        "my_tool",
    Description: "Does something useful",
}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input MyToolInput) (*sdkmcp.CallToolResult, *MyOutput, error) {
    // Handler
})
```

### 5. Testing Strategy

| Level | What | How |
|-------|------|-----|
| Unit | Tool handlers | Mock services, test in isolation |
| Integration | Full server | `CommandTransport` + SDK client |
| Functional | HTTP transport | httptest server + real requests |
| Compliance | Protocol | `mcp-validation` CLI tool |

## Debugging Checklist

When stdio doesn't work:

1. **Build fresh:** `make build`
2. **Check stdout pollution:** `make lint-stdout`
3. **Test manually:**
   ```bash
   echo '{"jsonrpc":"2.0","method":"initialize","params":{...},"id":1}' | ./bin/trellis 2>/dev/null
   ```
4. **Check for startup errors:** `./bin/trellis 2>&1 | head -5`
5. **Verify JSON output:** First line of stdout must be `{"jsonrpc":"2.0",...}`

## Common Mistakes

### ❌ Wrong: Logging to stdout
```go
fmt.Println("Server starting...")  // BREAKS PROTOCOL
```

### ✅ Right: Logging to stderr
```go
slog.Info("Server starting...")  // Goes to stderr in stdio mode
```

### ❌ Wrong: Custom JSON-RPC handling
```go
// Reading stdin manually and parsing JSON
scanner := bufio.NewScanner(os.Stdin)
for scanner.Scan() {
    var req map[string]any
    json.Unmarshal(scanner.Bytes(), &req)
    // Custom dispatch...
}
```

### ✅ Right: Using SDK transport
```go
transport := &sdkmcp.StdioTransport{}
server.Run(ctx, transport)  // SDK handles everything
```

## References

- Go SDK: https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp
- MCP Spec: https://modelcontextprotocol.io/specification/2025-03-26
- Transport Spec: https://modelcontextprotocol.io/specification/2025-03-26/basic/transports
```

### Installing the Skill

**Option A: Project-local (recommended for this repo)**

```bash
mkdir -p .claude/skills/mcp-protocol
# Create SKILL.md as above
```

Then add to `.claude/settings.local.json`:
```json
{
  "skills": [".claude/skills/mcp-protocol"]
}
```

**Option B: User-global skill**

Create in `~/.claude/skills/mcp-protocol/` for use across all MCP projects.

---

## Part 4: MCP Inspector Error Analysis

### The Actual Error

```
Error in /message route: Error: SSE connection not established
```

This error is **not from your server**. It's from the MCP Inspector's internal bridge, which:

1. Starts your stdio server as a subprocess
2. Creates an SSE bridge to proxy stdio ↔ HTTP
3. Expects the SSE connection to be established before accepting POST messages

### Root Causes

1. **Server crashes on startup** before any messages exchange
2. **Server emits non-JSON to stdout** (logs), breaking the framing
3. **Initialization handshake fails** due to missing/wrong response
4. **Inspector timing issue** (race condition in bridge setup)

### Diagnostic Steps

```bash
# 1. Check server starts at all
./bin/trellis 2>&1 | head -10

# 2. Check stdout is clean JSON
echo '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}},"id":1}' | \
  TRELLIS_DB_PATH=:memory: ./bin/trellis 2>/dev/null | head -1

# 3. Validate the response
# Should see: {"jsonrpc":"2.0","result":{...},"id":1}

# 4. Check for stderr output (should have logs here)
echo '...' | TRELLIS_DB_PATH=:memory: ./bin/trellis 2>&1 1>/dev/null | head -5
```

---

## Appendix A: MCP Inspector vs. Automated Testing

| Feature | MCP Inspector | SDK Integration Test |
|---------|---------------|---------------------|
| Interactive | ✓ | ✗ |
| CI-friendly | ✗ | ✓ |
| Visual debugging | ✓ | ✗ |
| Automated | ✗ | ✓ |
| Catches stdout pollution | Sometimes | Always |
| Speed | Slow (manual) | Fast (automated) |

**Recommendation:** Use Inspector for debugging. Use SDK tests for validation.

---

## Appendix B: Go SDK Quick Reference

```go
// Server creation
server := sdkmcp.NewServer(&sdkmcp.Implementation{
    Name:    "my-server",
    Version: "1.0.0",
}, &sdkmcp.ServerOptions{
    Instructions: "Server description for clients",
    Logger:       slog.Default(),
})

// Tool registration (generic pattern)
sdkmcp.AddTool(server, &sdkmcp.Tool{
    Name:        "tool_name",
    Description: "What it does",
}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input InputType) (*sdkmcp.CallToolResult, OutputType, error) {
    // Return nil, output, nil for success
    // Return nil, nil, err for errors
})

// Stdio transport
transport := &sdkmcp.StdioTransport{}
server.Run(ctx, transport)

// HTTP transport
handler := sdkmcp.NewStreamableHTTPHandler(
    func(r *http.Request) *sdkmcp.Server { return server },
    nil,
)
http.Handle("/mcp", handler)

// Client testing
transport, _ := sdkmcp.NewCommandTransport(ctx, sdkmcp.CommandTransportConfig{
    Command: "./server",
    Env:     []string{"KEY=value"},
})
client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "1.0"}, nil)
session, _ := client.Connect(ctx, transport)
tools, _ := session.ListTools(ctx, nil)
result, _ := session.CallTool(ctx, "ping", nil)
```

---

## References

- `0117-mcp-api-spec-draft.md` — May contain outdated patterns
- `0118-mcp-protocol-reqmts.md` — Verify against current spec
- `0120-stdio-transport.md` — Good but verify tool recommendations
