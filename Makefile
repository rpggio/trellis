APP_NAME=threds-mcp

.PHONY: test-unit test-integration test-functional test build run

test-unit:
	go test ./internal/domain/... -v

test-integration:
	go test ./test/integration -v

test-functional:
	go test ./test/functional -v

test:
	go test ./internal/... ./test/...

build:
	go build -o bin/$(APP_NAME) ./cmd/server

run:
	go run ./cmd/server
