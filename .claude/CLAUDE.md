# Project Instructions

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
