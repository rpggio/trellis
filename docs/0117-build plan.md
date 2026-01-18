Implementation Plan: threds-mcp Server

Overview
Building a Model Context Protocol (MCP) server for managing design reasoning across chat sessions. The system tracks records (questions, conclusions, notes) in parent-child hierarchies with session-based conflict detection using logical clocks (ticks).
Current State: Complete architectural specifications, zero production code.
Technology: Go 1.22+, SQLite (pure Go driver), Chi router, standard library testing.
Implementation Strategy
Build bottom-up in 8 phases, testing each layer before proceeding. Each phase must pass its tests before moving forward.

Phase 1: Project Foundation & Database Schema
Goal: Bootstrap Go project with dependencies and create MySQL-compatible SQLite schema.
Tasks

Initialize Go module

Create go.mod with module path
Add dependencies: modernc.org/sqlite, github.com/go-chi/chi/v5, github.com/golang-migrate/migrate/v4, github.com/stretchr/testify


Create package structure

   cmd/server/main.go (placeholder)
   internal/
   ├── repository/
   │   ├── interfaces.go
   │   └── errors.go
   ├── domain/
   │   ├── project/model.go
   │   ├── record/model.go
   │   ├── session/model.go
   │   └── activity/model.go
   ├── sqlite/db.go
   └── config/config.go
   migrations/
   ├── 001_initial_schema.up.sql
   └── 001_initial_schema.down.sql

Define database schema (MySQL-compatible, 8 tables)

projects: Container with monotonic tick counter
records: Hierarchical records with state (OPEN/LATER/RESOLVED/DISCARDED)
record_relations: Non-hierarchical relationships
sessions: Chat sessions tracking activations
session_activations: Many-to-many session↔record
activity_log: Audit trail with tick tracking
records_fts: SQLite FTS5 full-text search (v1)
api_keys: Bearer token authentication


Define domain models

Go structs matching database schema in internal/domain/*/model.go


Define repository interfaces

ProjectRepository, RecordRepository, SessionRepository, ActivityRepository
Standard errors: ErrNotFound, ErrConflict, ErrForeignKeyViolation



Verification

Create internal/sqlite/db_test.go with NewTestDB(t) helper for in-memory SQLite
Test migrations run successfully
Test tables created with correct structure
Test foreign keys enforce constraints
Command: go test ./internal/sqlite -v

Critical Files

migrations/001_initial_schema.up.sql - Database foundation
internal/repository/interfaces.go - Repository contracts


Phase 2: SQLite Repository Implementation
Goal: Implement all repository interfaces with full integration test coverage.
Notes:
- Source of truth for schema is migrations/001_initial_schema.up.sql; test helpers should read that file.
- Keep FTS5 triggers aligned with migrations to avoid drift between runtime and tests.
Tasks

Implement ProjectRepository (internal/sqlite/project.go)

Methods: Create, Get, GetDefault, IncrementTick, List
Atomic tick increment via transaction


Implement RecordRepository (internal/sqlite/record.go)

Methods: Create, Get, Update, List, GetChildren, Delete
Track modification tick on updates
Handle parent-child relationships


Implement SessionRepository (internal/sqlite/session.go)

Methods: Create, Get, Update, ListActive, GetByRecordID, Close
Manage session_activations join table
Track activation ticks


Implement ActivityRepository (internal/sqlite/activity.go)

Methods: Log, List with filters (type, record, time range)


Implement Search (internal/sqlite/search.go)

Full-text search using SQLite FTS5
Synchronize FTS index with records table



Testing Requirements
Integration tests for each repository (internal/sqlite/*_test.go):

CRUD operations work correctly
Tenant isolation enforced (tenant A cannot see tenant B's data)
Foreign key constraints prevent orphaned records
Tick tracking accurate
Filters and pagination work
Coverage target: >90%

Verification

Command: go test ./internal/sqlite -v -cover
All repository operations succeed
Tenant isolation verified for every repository
Tick increments are atomic


Phase 3: Domain Services (Business Logic)
Goal: Implement business logic services with unit tests using mock repositories.
Tasks

Generate repository mocks

Use mockery to generate mocks from interfaces
Output to internal/repository/mocks/


Implement ProjectService (internal/domain/project/service.go)

Methods: Create, Get, GetDefault, List
Auto-create default project if none exists


Implement RecordService (internal/domain/record/service.go)

Methods: Create, Update, Transition, Get, List, Search
Business rules:

Validate state transitions (OPEN→LATER/RESOLVED/DISCARDED, etc.)
Check parent activation for create child
Check record activation for update/transition
Increment project tick on all mutations
Log activity for all mutations




Implement SessionService (internal/domain/session/service.go)

Methods: Activate, SyncSession, SaveSession, CloseSession, BranchSession, GetActiveSessionsForRecord
Complex business rules:

Context loading: parent (full) + target (full) + OPEN children (full) + other children (ref only) + grandchildren (ref only)
Conflict detection: Check if record active in another session, compare activation tick vs current tick
Create session on first activation
Track active records in session_activations




Implement ActivityService (internal/domain/activity/service.go)

Methods: LogActivity, GetRecentActivity
Ensure activity logged with current project tick


Implement validation (internal/domain/record/validation.go)

State transition rules enforcement
Required field validation
RESOLVED requires resolved_by, LATER/DISCARDED require reason



Testing Requirements
Unit tests for each service (internal/domain/*/service_test.go):

Use mock repositories (no I/O)
Test business logic in isolation
Verify repository method calls using mocks
Test error paths
Coverage target: >85%
Performance: Tests must run in <1 second

Critical test cases:

Session activation loads correct context (parent full, OPEN children full, others ref)
Conflict detection when record active in multiple sessions
State transition validation (all valid and invalid transitions)
Precondition checks (parent activated, record activated)
Tick increments on mutations

Verification

Command: go test ./internal/domain/... -v
All business rules enforced
Mock assertions pass
Fast execution (<1s)

Critical Files

internal/domain/session/service.go - Most complex business logic
internal/domain/record/service.go - Second most complex


Phase 4: Integration Testing (Services + SQLite)
Goal: Verify domain services work with real SQLite repositories end-to-end.
Tasks

Create integration test suite (test/integration/integration_test.go)

Wire real SQLite (in-memory) + real repositories + real services
Test complete workflows


Test workflows:

Cold start: Create project → Create root record → Activate → Create child
Conflict detection: Two sessions activate same record → Both update → Verify conflict
Session branching: Activate parent → Create child → Branch to child session
State transitions: Test all valid transitions, verify invalid transitions fail
Context loading: Create hierarchy (parent → target → OPEN child + RESOLVED child + grandchild) → Activate target → Verify context contains parent (full), target (full), OPEN child (full), RESOLVED child (ref), grandchild (ref)
Staleness detection: Session activates record → Another session updates → First session syncs → Verify tick gap reported



Verification

Command: go test ./test/integration -v
All workflows execute correctly
Conflict detection works across sessions
Context loading follows spec exactly
Activity log captures all mutations


Phase 5: JSON-RPC Transport Layer
Goal: Implement HTTP server with JSON-RPC envelope handling and authentication.
Tasks

Implement JSON-RPC handler (internal/transport/jsonrpc.go)

Types: Request, Response, Error (JSON-RPC 2.0 spec)
Functions: ParseRequest, WriteResponse, WriteError


Implement authentication middleware (internal/transport/auth.go)

Extract Bearer token from Authorization header
Look up tenant_id from api_keys table (hash token)
Store tenant_id in request context
Return 401 if missing/invalid


Implement session header middleware

Extract Mcp-Session-Id header
Store in request context (optional, created on first activation)


Implement HTTP server (internal/transport/http.go)

Use Chi router
Single endpoint: POST /mcp
Middleware chain: logging → auth → session → handler
Health check: GET /health



Testing Requirements
Unit tests:

internal/transport/jsonrpc_test.go: Parse requests, write responses, error mapping
internal/transport/auth_test.go: Valid/invalid tokens, tenant extraction
internal/transport/http_test.go: Endpoint routing, auth enforcement

Verification

Command: go test ./internal/transport -v
Can parse JSON-RPC 2.0 requests
Authentication works
HTTP server handles requests


Phase 6: MCP Handler Layer (API Implementation)
Goal: Implement all 23 MCP commands.
Tasks

Define MCP types (internal/mcp/types.go)

Request/response structs for all 23 commands
Match API specification exactly


Implement MCP handler (internal/mcp/handler.go)

Handler struct with domain service dependencies
Handle(ctx, tenantID, sessionID, method, params) method
Switch on method, unmarshal params, call domain service, return result


Implement all 23 commands:

Project (3): create_project, list_projects, get_project
Orientation (4): get_project_overview, search_records, list_records, get_record_ref
Activation (2): activate, sync_session
Mutation (3): create_record, update_record, transition
Session lifecycle (3): save_session, close_session, branch_session
History & conflict (3): get_record_history, get_record_diff, get_active_sessions
Activity (1): get_recent_activity


Implement error mapping (internal/mcp/errors.go)

Map domain errors to MCP error codes
Examples: ErrNotFound → RECORD_NOT_FOUND, ErrNotActivated → NOT_ACTIVATED



Testing Requirements
Unit tests (internal/mcp/handler_test.go):

Test each command with valid/invalid params
Test error conditions
Verify correct domain service called
Use mock domain services

Verification

Command: go test ./internal/mcp -v
All 23 commands implemented
Error codes mapped correctly

Critical Files

internal/mcp/handler.go - Single dispatch point for all API commands


Phase 7: End-to-End Functional Tests
Goal: Test complete HTTP→MCP→Domain→SQLite flow with real HTTP requests.
Tasks

Create test server helper (cmd/server/test_server.go)

Wire all components with in-memory SQLite
Insert test API key


Implement functional tests (test/functional/mcp_test.go)

Use httptest.Server for real HTTP
Make real JSON-RPC POST requests


Test scenarios:

Authentication (401 without token, 200 with token)
Project creation
Complete activation workflow (create project → create record → activate → create child → update → transition → save → close)
Conflict detection (two sessions, concurrent updates)
Tenant isolation (tenant A cannot access tenant B's data)
Context loading (verify parent full, OPEN children full, others ref)
Session branching
State transitions (all valid and invalid)
Activity log
Full-text search



Verification

Command: go test ./test/functional -v
All 23 MCP commands work end-to-end via HTTP
Authentication enforced
Tenant isolation verified
Conflict detection works
Context loading correct


Phase 8: Main Entry Point & Configuration
Goal: Create production-ready server with configuration and deployment.
Tasks

Implement configuration (internal/config/config.go)

Load from environment variables + YAML
Config: server port/host, database path, log level


Implement main entry point (cmd/server/main.go)

Load config
Initialize logger (slog)
Connect to SQLite (file-based for production)
Run migrations
Wire all dependencies
Start HTTP server
Graceful shutdown on SIGINT/SIGTERM


Embed migrations

Use embed.FS for migration files
Run on startup


Create Makefile

Targets: test-unit, test-integration, test-functional, test, build, run, mock


Create README

Installation instructions
Configuration guide
Running tests
Development setup



Manual Verification
bash# Start server
make run

# Test with curl
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-key" \
  -d '{
    "jsonrpc": "2.0",
    "method": "list_projects",
    "id": 1
  }'
Verification

Server starts without errors
Responds to HTTP requests
Migrations run on startup
Graceful shutdown works
Configuration loaded correctly


Testing Strategy
Test Pyramid
     Functional (10-20 tests)
     HTTP + JSON-RPC + Full Stack

   Integration (30-50 tests)
   Services + Real SQLite

Unit (100-200 tests)
Services + Mocks
Coverage Targets

Unit tests: >85%
Integration tests: >90% (repository layer)
Functional tests: Cover all 23 MCP commands

Running Tests
bash# Fast (run frequently during development)
make test-unit

# Integration (run before commit)
make test-integration

# Functional (run before PR)
make test-functional

# All tests (run in CI)
make test

Critical Business Rules to Verify

Session Management

Sessions created on first activation
Records activated in session tracked in session_activations
Multiple records can be active in one session


Context Loading (on activation)

Parent: Full content
Target: Full content
OPEN children: Full content
LATER/RESOLVED/DISCARDED children: Reference only
Grandchildren: Reference only (always)


Tick & Staleness

Every write increments project tick
Session tracks last_sync_tick
Staleness = project.tick - session.last_sync_tick


Conflict Handling

Activation conflict: Record active in another session → warn but allow
Update conflict: Record modified since activation → return both versions


Valid State Transitions

OPEN → LATER (with reason)
OPEN → RESOLVED (with resolved_by)
OPEN → DISCARDED (with reason)
LATER → OPEN
LATER → DISCARDED
RESOLVED → OPEN
DISCARDED → OPEN


Preconditions

Create child: Parent must be activated
Update/Transition: Record must be activated
Activation: No precondition


Tenant Isolation

Every query scoped by tenant_id
API keys map to tenant_id




Success Criteria
Implementation is complete when:

✅ All 23 MCP commands work via HTTP POST /mcp
✅ All tests pass: unit (>85% coverage), integration (>90%), functional (all commands)
✅ Tenant isolation enforced at every layer
✅ Conflict detection works across sessions
✅ Context loading follows spec (parent full, OPEN children full, etc.)
✅ Activity log captures all mutations
✅ Server starts/stops cleanly with graceful shutdown
✅ Manual E2E test succeeds with curl
✅ No SQLite-specific SQL (MySQL-compatible for future Dolt migration)
✅ Documentation complete (README, development setup)


Phase Dependencies
Phase 1 (Foundation)
    ↓
Phase 2 (SQLite Repositories) ←─────┐
    ↓                                │
Phase 3 (Domain Services)            │ Integration tests
    ↓                                │ require both
Phase 4 (Integration Tests) ─────────┘
    ↓
Phase 5 (Transport Layer)
    ↓
Phase 6 (MCP Handlers)
    ↓
Phase 7 (Functional Tests)
    ↓
Phase 8 (Main + Config)
Note: Phase 3 can use mocks to start before Phase 2 completes.
