APP_NAME=trellis
DEFAULT_PORT=8080

.PHONY: help test-unit test-integration test-functional test test-stdio build run dev clean lint-stdout validate-mcp ci

## help: Show this help message
help:
	@echo "Available targets:"
	@echo "  make dev              - Run dev server (stdio, auth disabled)"
	@echo "  make dev-http         - Run dev server (HTTP, auth disabled, localhost:8080)"
	@echo "  make test             - Run all tests"
	@echo "  make test-unit        - Run unit tests"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-functional  - Run functional tests"
	@echo "  make test-stdio       - Test stdio protocol compliance (shell script)"
	@echo "  make lint-stdout      - Check for stdout pollution in server code"
	@echo "  make validate-mcp     - Run MCP protocol compliance tests"
	@echo "  make ci               - Full CI pipeline (build + test + validate)"
	@echo "  make build            - Build the binary"
	@echo "  make run              - Run the server with default config"
	@echo "  make clean            - Remove build artifacts"

## dev: Run development server with stdio transport (auth disabled)
dev:
	@echo "Starting dev server (stdio transport)..."
	@echo "Auth: DISABLED"
	@echo "Transport: stdio (default)"
	@echo "Connect via MCP client using stdio"
	@echo ""
	TRELLIS_AUTH_ENABLED=false \
	TRELLIS_LOG_LEVEL=debug \
	go run ./cmd/server

## dev-http: Run development server with HTTP transport (auth disabled)
dev-http:
	@echo "Starting dev server (HTTP transport)..."
	@echo "Auth: DISABLED"
	@echo "URL: http://localhost:$(DEFAULT_PORT)"
	@echo ""
	TRELLIS_TRANSPORT=http \
	TRELLIS_AUTH_ENABLED=false \
	TRELLIS_SERVER_HOST=127.0.0.1 \
	TRELLIS_SERVER_PORT=$(DEFAULT_PORT) \
	TRELLIS_LOG_LEVEL=debug \
	go run ./cmd/server

## test-unit: Run unit tests
test-unit:
	go test ./internal/domain/... -v

## test-integration: Run integration tests
test-integration:
	go test ./test/integration -v

## test-functional: Run functional tests
test-functional:
	go test ./test/functional -v

## test: Run all tests
test:
	go test ./internal/... ./test/...

## test-stdio: Test stdio protocol compliance
test-stdio:
	@./test/stdio_test.sh

## build: Build the server binary
build:
	go build -o bin/$(APP_NAME) ./cmd/server

## run: Run the server with default configuration
run:
	go run ./cmd/server

## lint-stdout: Verify no fmt.Print statements in server code (pollutes stdio protocol)
lint-stdout:
	@echo "Checking for stdout pollution..."
	@if grep -rn --include='*.go' 'fmt\.Print\|println(' ./cmd ./internal/mcp 2>/dev/null | grep -v '_test.go'; then \
		echo "ERROR: Found fmt.Print/println in server code. Use slog to stderr instead."; \
		exit 1; \
	fi
	@echo "✓ No stdout pollution detected"

## validate-mcp: Run MCP protocol compliance checks
validate-mcp: build lint-stdout
	@echo "Running protocol compliance tests..."
	TRELLIS_DB_PATH=:memory: go test ./test/integration -run TestStdioProtocol -v

## ci: Full CI pipeline
ci: build test validate-mcp
	@echo "✓ CI pipeline complete"

## clean: Remove build artifacts
clean:
	rm -rf bin/
