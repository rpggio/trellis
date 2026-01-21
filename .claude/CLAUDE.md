# Project Instructions

## Principles

### Defensive Coding with External Libraries
When implementing callbacks, handlers, or middleware that receive data from external libraries (like the MCP SDK), always handle nil/edge cases defensively. Library code may pass nil values in cases that aren't obvious from the type signatures—particularly for optional params in notifications or protocol messages.

### Validation Before Completion
Always run `make ci` before considering any coding work complete. The CI pipeline catches issues that unit tests miss, particularly protocol compliance problems. Don't rely on manual testing alone.

### Test Both Transports
This is an MCP server supporting both HTTP and stdio transports. Any functional change should be verified against both transports. The stdio transport has stricter requirements (stdout hygiene, proper notification handling) that HTTP doesn't expose.

## Documentation Format
- Always use Markdown for documentation unless otherwise specified
- Docs are stored in `docs/`, filename prefixed with date as `MMDD-`

## Build Artifacts and Scripts
- NEVER create compiled executables or build artifacts in the project root
- Build outputs must go in `bin/` directory (already configured in Makefile)
- Test scripts belong in `test/` directory, NOT in project root
- The project root should remain clean of temporary files, executables, and test scripts
- Use `bin/` for compiled binaries (e.g., `bin/threds-mcp`)
- Use `test/` for test scripts (e.g., `test/stdio_compliance.sh`)
- Temporary files should use `/tmp/` or be placed in gitignored directories

## MCP Server Development
- Use the official Go SDK (`github.com/modelcontextprotocol/go-sdk/mcp`) for all MCP functionality
- Never implement custom JSON-RPC handling; let the SDK handle protocol details
- The stdio protocol tests in `test/integration/stdio_protocol_test.go` use the SDK client to verify compliance
