APP_NAME=threds-mcp
DEFAULT_PORT=8080

.PHONY: help test-unit test-integration test-functional test test-stdio build run dev clean

## help: Show this help message
help:
	@echo "Available targets:"
	@echo "  make dev              - Run dev server (stdio, auth disabled)"
	@echo "  make dev-http         - Run dev server (HTTP, auth disabled, localhost:8080)"
	@echo "  make test             - Run all tests"
	@echo "  make test-unit        - Run unit tests"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-functional  - Run functional tests"
	@echo "  make test-stdio       - Test stdio protocol compliance"
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
	THREDS_AUTH_ENABLED=false \
	THREDS_LOG_LEVEL=debug \
	go run ./cmd/server

## dev-http: Run development server with HTTP transport (auth disabled)
dev-http:
	@echo "Starting dev server (HTTP transport)..."
	@echo "Auth: DISABLED"
	@echo "URL: http://localhost:$(DEFAULT_PORT)"
	@echo ""
	THREDS_TRANSPORT=http \
	THREDS_AUTH_ENABLED=false \
	THREDS_SERVER_HOST=127.0.0.1 \
	THREDS_SERVER_PORT=$(DEFAULT_PORT) \
	THREDS_LOG_LEVEL=debug \
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

## clean: Remove build artifacts
clean:
	rm -rf bin/
