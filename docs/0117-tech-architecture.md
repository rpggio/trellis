# Record-Sync MCP: Technical Architecture

*Technical selection, system architecture, and implementation guidance*

**Draft v0.2 — January 2025**

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [MCP Protocol Decisions](#2-mcp-protocol-decisions)
3. [Database Selection](#3-database-selection)
4. [Technology Stack](#4-technology-stack)
5. [System Architecture](#5-system-architecture)
6. [Package Structure](#6-package-structure)
7. [Coding Guidelines](#7-coding-guidelines)
8. [Testing Strategy](#8-testing-strategy)
9. [Decision Summary](#9-decision-summary)

---

## 1. Executive Summary

This document provides technical guidance for building an HTTP-based MCP server for the Record-Sync system. The architecture prioritizes simplicity for v1 while establishing foundations for future concurrent multi-agent usage.

### Key Decisions

| Area | v1 Choice | Rationale |
|------|-----------|-----------|
| Transport | Plain HTTP POST (JSON-RPC) | SSE deprecated; all operations are request-response |
| Database | SQLite with MySQL-compatible SQL | Simple ops; clean migration path to Dolt |
| Language | Go 1.22+ | Excellent HTTP performance, simple deployment |
| Auth | Bearer tokens (API keys) | Standard, works with MCP clients |

### What This Document Covers

- Protocol and transport decisions
- Database selection and migration strategy
- Complete package structure with interface definitions
- Coding patterns and conventions
- Testing strategy at all levels (unit, integration, functional)

---

## 2. MCP Protocol Decisions

### 2.1 Transport: Plain HTTP (No SSE Needed)

The MCP specification (2025-03-26) defines "Streamable HTTP" as the standard transport. Despite the name, **streaming is optional**—it's only needed when a server must send multiple messages for a single request.

**Our API is entirely request-response.** Every command (`activate`, `create_record`, `sync_session`, etc.) follows the pattern:

```
Client sends request → Server returns response → Done
```

There is no scenario where the server needs to push unsolicited data to the client.

#### Why Not SSE?

| Consideration | Analysis |
|---------------|----------|
| Chat-based clients | Everything is user-triggered; Claude waits for tool responses |
| Server push | No mechanism for MCP server to "interrupt" Claude mid-thought |
| MCP spec direction | SSE was deprecated in favor of Streamable HTTP |
| Complexity | SSE adds connection management overhead for zero benefit |

#### Implementation

Single endpoint, standard HTTP POST:

```
POST /mcp
Content-Type: application/json
Authorization: Bearer <api-key>

{"jsonrpc": "2.0", "method": "activate", "params": {"id": "R7"}, "id": 1}
```

Response:

```json
{"jsonrpc": "2.0", "result": {"session_id": "...", "context": {...}}, "id": 1}
```

#### Future: Standalone Agents

When autonomous agents become a requirement, we may need:
- Server-to-agent push notifications
- WebSocket or long-polling for real-time sync

That's a different architecture than MCP tool calls. Defer until needed.

### 2.2 Session Management

Sessions are **logical constructs tracked in the database**, not tied to HTTP connections.

```
POST /mcp  →  {"method": "activate", "params": {"id": "R7"}}
            ←  {"result": {"session_id": "sess_abc123", ...}}

POST /mcp  →  {"method": "update_record", "params": {"id": "R7", ...}}
Header: Mcp-Session-Id: sess_abc123
```

- Client includes `Mcp-Session-Id` header on subsequent requests
- Server looks up session state from database
- Enables horizontal scaling (any server instance can handle any request)

### 2.3 Authentication & Multi-tenancy

**Bearer token authentication** via Authorization header:

```
Authorization: Bearer <api-key>
```

- API keys are issued per-user and map to a `tenant_id`
- All data queries are scoped by `tenant_id`
- Standard, secure, compatible with MCP clients

#### Tenant Isolation

```go
// Every repository method takes tenantID as first parameter
type RecordRepository interface {
    Get(ctx context.Context, tenantID, id string) (*Record, error)
    // ...
}
```

### 2.4 Conflict Detection

Combine session-based tracking with tick comparison:

1. **On activation**: Check if record is active in another session → warn
2. **On update**: Compare tick at activation vs current → detect conflicts
3. **Philosophy**: Warn but don't prevent (matches spec)

---

## 3. Database Selection

### 3.1 Requirements Analysis

| Requirement | v1 (MVP) | Future |
|-------------|----------|--------|
| Single-user operation | Required | Required |
| Concurrent agent sessions | Nice-to-have | Required |
| Automatic session sync | Not needed | Required |
| Record history/versioning | Nice-to-have | Required |
| Branching/merging records | Not needed | Desired |
| Hosted multi-tenant | Required | Required |

### 3.2 Decision: SQLite for v1, Dolt Later

**SQLite advantages for v1:**
- Zero operational complexity
- Excellent Go support via `modernc.org/sqlite` (pure Go, no CGO)
- WAL mode supports concurrent reads
- Per-tenant database files provide natural isolation
- Fast iteration during development

**Dolt advantages (future):**
- Git-like versioning built into database layer
- Branch/merge semantics for concurrent agent sessions
- Time-travel queries (`AS OF`) for history
- MySQL wire protocol compatibility

### 3.3 Migration Strategy

Dolt supports **only MySQL wire protocol** (not Postgres). Build on MySQL-compatible patterns from the start:

1. Use `database/sql` interface with MySQL-compatible syntax
2. Avoid SQLite-specific features (e.g., `AUTOINCREMENT` → use application-generated IDs)
3. Implement manual history tracking (audit tables) for v1
4. When migrating to Dolt, replace audit tables with native versioning

#### SQL Compatibility Notes

```sql
-- Use this (MySQL-compatible)
CREATE TABLE records (
    id VARCHAR(36) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Avoid this (SQLite-specific)
CREATE TABLE records (
    id INTEGER PRIMARY KEY AUTOINCREMENT
);
```

### 3.4 Full-Text Search

- **v1**: SQLite FTS5 (built-in, zero dependencies)
- **Future**: MySQL FULLTEXT (available in Dolt)

Abstract search behind an interface to enable future replacement:

```go
type SearchService interface {
    Search(ctx context.Context, tenantID, query string, opts SearchOptions) ([]SearchResult, error)
}
```

---

## 4. Technology Stack

### 4.1 Core Dependencies

| Component | Selection | Rationale |
|-----------|-----------|-----------|
| Language | Go 1.22+ | Performance, simple deployment, strong concurrency |
| HTTP | `net/http` + `chi` router | Standard library + routing ergonomics |
| Database | `modernc.org/sqlite` | Pure Go SQLite, no CGO required |
| JSON | `encoding/json` | Standard library sufficient |
| Config | `envconfig` + YAML | 12-factor app compliance |
| Logging | `slog` (Go 1.21+) | Standard library structured logging |
| Testing | `testing` + `testify` | Standard + assertion helpers |
| Migrations | `golang-migrate/migrate` | Database schema migrations |

### 4.2 Why These Choices

**Chi router over alternatives:**
- Lightweight, idiomatic Go
- Middleware support
- No framework lock-in

**Pure Go SQLite (`modernc.org/sqlite`):**
- No CGO means simpler cross-compilation
- Single binary deployment
- Slight performance tradeoff is acceptable for our workload

**Standard library preference:**
- `slog` over logrus/zap: Now in stdlib, good enough
- `encoding/json` over easyjson: Simplicity over micro-optimization

---

## 5. System Architecture

### 5.1 Layered Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    MCP Clients (Claude, etc.)                       │
└─────────────────────────────────────────────────────────────────────┘
                              │  HTTPS + Bearer Auth
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Transport Layer                                │
│              (HTTP Server, JSON-RPC handling)                       │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      MCP Handler Layer                              │
│           (Tool dispatch, request/response mapping)                 │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Domain Services                                │
│          (Projects, Records, Sessions, Activities)                  │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Repository Interface                             │
│                 (Abstracts storage backend)                         │
└─────────────────────────────────────────────────────────────────────┘
                    │                       │
                    ▼                       ▼
          ┌─────────────────┐     ┌─────────────────┐
          │  SQLite (v1)    │     │  Dolt (future)  │
          └─────────────────┘     └─────────────────┘
```

### 5.2 Request Flow

```
1. HTTP Request arrives at /mcp
2. Auth middleware extracts tenant_id from Bearer token
3. JSON-RPC envelope parsed, method extracted
4. Handler dispatches to appropriate domain service
5. Domain service uses repository interface for data access
6. Response serialized as JSON-RPC and returned
```

### 5.3 Key Interfaces

```go
// Repository interfaces - storage abstraction

type RecordRepository interface {
    Create(ctx context.Context, tenantID string, r *Record) error
    Get(ctx context.Context, tenantID, id string) (*Record, error)
    Update(ctx context.Context, tenantID string, r *Record, expectedTick int64) error
    Delete(ctx context.Context, tenantID, id string) error
    List(ctx context.Context, tenantID string, opts ListOptions) ([]RecordRef, error)
    GetChildren(ctx context.Context, tenantID, parentID string) ([]Record, error)
    Search(ctx context.Context, tenantID, query string, opts SearchOptions) ([]SearchResult, error)
}

type SessionRepository interface {
    Create(ctx context.Context, tenantID string, s *Session) error
    Get(ctx context.Context, tenantID, id string) (*Session, error)
    Update(ctx context.Context, tenantID string, s *Session) error
    ListActive(ctx context.Context, tenantID string) ([]Session, error)
    GetByRecordID(ctx context.Context, tenantID, recordID string) ([]Session, error)
    Close(ctx context.Context, tenantID, id string) error
}

type ProjectRepository interface {
    Create(ctx context.Context, tenantID string, p *Project) error
    Get(ctx context.Context, tenantID, id string) (*Project, error)
    GetDefault(ctx context.Context, tenantID string) (*Project, error)
    IncrementTick(ctx context.Context, tenantID, projectID string) (int64, error)
}

type ActivityRepository interface {
    Log(ctx context.Context, tenantID string, entry *ActivityEntry) error
    List(ctx context.Context, tenantID string, opts ActivityListOptions) ([]ActivityEntry, error)
}
```

---

## 6. Package Structure

```
record-sync-mcp/
├── cmd/
│   └── server/
│       └── main.go              # Entry point, wires everything together
│
├── internal/
│   ├── transport/
│   │   ├── http.go              # HTTP server setup, middleware
│   │   ├── jsonrpc.go           # JSON-RPC envelope handling
│   │   └── auth.go              # Bearer token validation, tenant extraction
│   │
│   ├── mcp/
│   │   ├── handler.go           # Method dispatch (activate, create_record, etc.)
│   │   ├── types.go             # Request/response types for MCP methods
│   │   └── errors.go            # MCP error codes and formatting
│   │
│   ├── domain/
│   │   ├── record/
│   │   │   ├── service.go       # Record business logic
│   │   │   ├── model.go         # Record, RecordRef types
│   │   │   └── validation.go    # Record validation rules
│   │   ├── session/
│   │   │   ├── service.go       # Session lifecycle, activation
│   │   │   ├── model.go         # Session type
│   │   │   └── conflict.go      # Conflict detection logic
│   │   ├── project/
│   │   │   ├── service.go       # Project and tick management
│   │   │   └── model.go         # Project type
│   │   └── activity/
│   │       ├── service.go       # Activity logging
│   │       └── model.go         # ActivityEntry type
│   │
│   ├── repository/
│   │   ├── interfaces.go        # Repository interface definitions
│   │   └── errors.go            # Repository-level errors (NotFound, etc.)
│   │
│   ├── sqlite/
│   │   ├── db.go                # Database connection, migrations
│   │   ├── record.go            # RecordRepository implementation
│   │   ├── session.go           # SessionRepository implementation
│   │   ├── project.go           # ProjectRepository implementation
│   │   ├── activity.go          # ActivityRepository implementation
│   │   └── search.go            # FTS5 search implementation
│   │
│   └── config/
│       └── config.go            # Configuration loading
│
├── migrations/
│   ├── 001_initial_schema.up.sql
│   ├── 001_initial_schema.down.sql
│   └── ...
│
├── testdata/
│   └── fixtures/                # Test fixtures
│
├── go.mod
├── go.sum
└── README.md
```

### 6.1 Package Responsibilities

| Package | Responsibility | Dependencies |
|---------|----------------|--------------|
| `cmd/server` | Wire dependencies, start server | All internal packages |
| `transport` | HTTP handling, auth middleware | `mcp` |
| `mcp` | MCP protocol, method dispatch | `domain/*` |
| `domain/*` | Business logic, validation | `repository` |
| `repository` | Interface definitions | None (interfaces only) |
| `sqlite` | SQLite implementation | `repository` |
| `config` | Configuration loading | None |

### 6.2 Dependency Direction

```
cmd/server
    ↓
transport → mcp → domain/* → repository (interfaces)
                                  ↑
                              sqlite (implementation)
```

- Domain services depend on repository **interfaces**, not implementations
- SQLite package implements repository interfaces
- `cmd/server` wires concrete implementations to interfaces

---

## 7. Coding Guidelines

### 7.1 General Principles

1. **Explicit over implicit**: Pass dependencies explicitly, avoid globals
2. **Interfaces at boundaries**: Define interfaces where packages meet
3. **Errors are values**: Return errors, don't panic
4. **Context everywhere**: All operations take `context.Context`
5. **Tenant-scoped**: Every data operation includes `tenantID`

### 7.2 Error Handling

```go
// Define domain-specific errors
var (
    ErrRecordNotFound    = errors.New("record not found")
    ErrNotActivated      = errors.New("record not activated in session")
    ErrInvalidTransition = errors.New("invalid state transition")
    ErrConflict          = errors.New("record modified by another session")
)

// Wrap with context
func (s *RecordService) Get(ctx context.Context, tenantID, id string) (*Record, error) {
    record, err := s.repo.Get(ctx, tenantID, id)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            return nil, fmt.Errorf("%w: %s", ErrRecordNotFound, id)
        }
        return nil, fmt.Errorf("getting record %s: %w", id, err)
    }
    return record, nil
}
```

### 7.3 Service Pattern

```go
// Service struct with dependencies injected
type RecordService struct {
    records    repository.RecordRepository
    sessions   repository.SessionRepository
    projects   repository.ProjectRepository
    activities repository.ActivityRepository
    logger     *slog.Logger
}

// Constructor validates dependencies
func NewRecordService(
    records repository.RecordRepository,
    sessions repository.SessionRepository,
    projects repository.ProjectRepository,
    activities repository.ActivityRepository,
    logger *slog.Logger,
) *RecordService {
    return &RecordService{
        records:    records,
        sessions:   sessions,
        projects:   projects,
        activities: activities,
        logger:     logger,
    }
}

// Methods are thin orchestration over repositories
func (s *RecordService) Create(ctx context.Context, tenantID string, req CreateRecordRequest) (*Record, error) {
    // 1. Validate
    if err := req.Validate(); err != nil {
        return nil, err
    }

    // 2. Check preconditions (parent activated if specified)
    if req.ParentID != "" {
        // ... check activation
    }

    // 3. Create record
    record := &Record{
        ID:        generateID(),
        TenantID:  tenantID,
        Type:      req.Type,
        // ...
    }

    if err := s.records.Create(ctx, tenantID, record); err != nil {
        return nil, err
    }

    // 4. Increment project tick
    if _, err := s.projects.IncrementTick(ctx, tenantID, record.ProjectID); err != nil {
        return nil, err
    }

    // 5. Log activity
    s.activities.Log(ctx, tenantID, &ActivityEntry{
        Type:     "record_created",
        RecordID: record.ID,
        // ...
    })

    return record, nil
}
```

### 7.4 Repository Implementation Pattern

```go
// SQLite implementation of RecordRepository
type recordRepo struct {
    db *sql.DB
}

func NewRecordRepository(db *sql.DB) repository.RecordRepository {
    return &recordRepo{db: db}
}

func (r *recordRepo) Get(ctx context.Context, tenantID, id string) (*Record, error) {
    const query = `
        SELECT id, type, title, summary, body, state, parent_id, created_at, modified_at
        FROM records
        WHERE tenant_id = ? AND id = ?
    `

    var rec Record
    err := r.db.QueryRowContext(ctx, query, tenantID, id).Scan(
        &rec.ID, &rec.Type, &rec.Title, &rec.Summary, &rec.Body,
        &rec.State, &rec.ParentID, &rec.CreatedAt, &rec.ModifiedAt,
    )
    if err == sql.ErrNoRows {
        return nil, repository.ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("querying record: %w", err)
    }

    return &rec, nil
}
```

### 7.5 HTTP Handler Pattern

```go
// Handler wraps domain services
type Handler struct {
    records  *record.Service
    sessions *session.Service
    projects *project.Service
    logger   *slog.Logger
}

// MCP method dispatch
func (h *Handler) HandleMCP(w http.ResponseWriter, r *http.Request) {
    // Extract tenant from context (set by auth middleware)
    tenantID := auth.TenantFromContext(r.Context())

    // Parse JSON-RPC request
    var req jsonrpc.Request
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        jsonrpc.WriteError(w, req.ID, jsonrpc.ErrParse, "invalid JSON")
        return
    }

    // Dispatch by method
    var result any
    var err error

    switch req.Method {
    case "activate":
        result, err = h.handleActivate(r.Context(), tenantID, req.Params)
    case "create_record":
        result, err = h.handleCreateRecord(r.Context(), tenantID, req.Params)
    // ... other methods
    default:
        jsonrpc.WriteError(w, req.ID, jsonrpc.ErrMethodNotFound, req.Method)
        return
    }

    if err != nil {
        h.writeError(w, req.ID, err)
        return
    }

    jsonrpc.WriteResult(w, req.ID, result)
}
```

---

## 8. Testing Strategy

### 8.1 Test Pyramid

```
                    ┌───────────────┐
                    │  Functional   │  ← Few, slow, high confidence
                    │   (HTTP)      │
                    └───────────────┘
               ┌─────────────────────────┐
               │      Integration        │  ← Moderate, test boundaries
               │   (Service + SQLite)    │
               └─────────────────────────┘
          ┌───────────────────────────────────┐
          │            Unit Tests             │  ← Many, fast, isolated
          │  (Services with mock repos)       │
          └───────────────────────────────────┘
```

### 8.2 Unit Tests

Test domain logic in isolation using mock repositories.

**Location**: `internal/domain/*/service_test.go`

```go
// internal/domain/record/service_test.go
package record_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"

    "record-sync-mcp/internal/domain/record"
    "record-sync-mcp/internal/repository/mocks"
)

func TestRecordService_Create(t *testing.T) {
    ctx := context.Background()
    tenantID := "tenant-1"

    t.Run("creates record with valid input", func(t *testing.T) {
        // Arrange
        mockRecords := mocks.NewMockRecordRepository(t)
        mockSessions := mocks.NewMockSessionRepository(t)
        mockProjects := mocks.NewMockProjectRepository(t)
        mockActivities := mocks.NewMockActivityRepository(t)

        mockRecords.On("Create", mock.Anything, tenantID, mock.AnythingOfType("*record.Record")).
            Return(nil)
        mockProjects.On("IncrementTick", mock.Anything, tenantID, "proj-1").
            Return(int64(42), nil)
        mockActivities.On("Log", mock.Anything, tenantID, mock.Anything).
            Return(nil)

        svc := record.NewService(mockRecords, mockSessions, mockProjects, mockActivities, nil)

        // Act
        rec, err := svc.Create(ctx, tenantID, record.CreateRequest{
            ProjectID: "proj-1",
            Type:      "question",
            Title:     "Test Question",
            Summary:   "A test question",
            Body:      "Full body content",
        })

        // Assert
        assert.NoError(t, err)
        assert.NotEmpty(t, rec.ID)
        assert.Equal(t, "question", rec.Type)
        mockRecords.AssertExpectations(t)
    })

    t.Run("fails when parent not activated", func(t *testing.T) {
        mockRecords := mocks.NewMockRecordRepository(t)
        mockSessions := mocks.NewMockSessionRepository(t)

        // Session exists but parent not in active_records
        mockSessions.On("Get", mock.Anything, tenantID, "sess-1").
            Return(&session.Session{ActiveRecords: []string{}}, nil)

        svc := record.NewService(mockRecords, mockSessions, nil, nil, nil)

        _, err := svc.Create(ctx, tenantID, record.CreateRequest{
            SessionID: "sess-1",
            ParentID:  "R1",  // Parent specified but not activated
            Type:      "question",
            Title:     "Test",
        })

        assert.ErrorIs(t, err, record.ErrParentNotActivated)
    })
}

func TestRecordService_Transition(t *testing.T) {
    tests := []struct {
        name        string
        fromState   string
        toState     string
        shouldError bool
    }{
        {"OPEN to RESOLVED", "OPEN", "RESOLVED", false},
        {"OPEN to LATER", "OPEN", "LATER", false},
        {"LATER to OPEN", "LATER", "OPEN", false},
        {"RESOLVED to LATER", "RESOLVED", "LATER", true},  // Invalid
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            // ... test transition validity
        })
    }
}
```

**What to unit test:**
- State transition validation
- Business rule enforcement (parent must be activated, etc.)
- Error handling paths
- Input validation

### 8.3 Integration Tests

Test service + repository together with a real SQLite database.

**Location**: `internal/sqlite/*_test.go` and `internal/integration_test.go`

```go
// internal/sqlite/record_test.go
package sqlite_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "record-sync-mcp/internal/sqlite"
    "record-sync-mcp/internal/domain/record"
)

func TestRecordRepository_Integration(t *testing.T) {
    // Setup: create in-memory database
    db := sqlite.NewTestDB(t)  // Helper that runs migrations
    repo := sqlite.NewRecordRepository(db)

    ctx := context.Background()
    tenantID := "test-tenant"

    t.Run("create and retrieve record", func(t *testing.T) {
        rec := &record.Record{
            ID:       "R001",
            Type:     "question",
            Title:    "Test Question",
            Summary:  "Summary",
            Body:     "Body content",
            State:    "OPEN",
        }

        err := repo.Create(ctx, tenantID, rec)
        require.NoError(t, err)

        retrieved, err := repo.Get(ctx, tenantID, "R001")
        require.NoError(t, err)
        assert.Equal(t, "Test Question", retrieved.Title)
    })

    t.Run("tenant isolation", func(t *testing.T) {
        // Create record for tenant-1
        err := repo.Create(ctx, "tenant-1", &record.Record{ID: "R100", Title: "T1 Record"})
        require.NoError(t, err)

        // Attempt to retrieve as tenant-2
        _, err = repo.Get(ctx, "tenant-2", "R100")
        assert.ErrorIs(t, err, repository.ErrNotFound)
    })

    t.Run("full-text search", func(t *testing.T) {
        // Create searchable records
        repo.Create(ctx, tenantID, &record.Record{
            ID:    "R200",
            Title: "Redis Caching Strategy",
            Body:  "Using Redis for distributed caching",
        })
        repo.Create(ctx, tenantID, &record.Record{
            ID:    "R201",
            Title: "PostgreSQL Optimization",
            Body:  "Database query optimization",
        })

        results, err := repo.Search(ctx, tenantID, "caching", record.SearchOptions{})
        require.NoError(t, err)
        assert.Len(t, results, 1)
        assert.Equal(t, "R200", results[0].ID)
    })
}

// Test helper: creates in-memory SQLite with migrations applied
func NewTestDB(t *testing.T) *sql.DB {
    t.Helper()

    db, err := sql.Open("sqlite", ":memory:")
    require.NoError(t, err)

    t.Cleanup(func() { db.Close() })

    // Run migrations
    err = sqlite.Migrate(db)
    require.NoError(t, err)

    return db
}
```

**What to integration test:**
- SQL query correctness
- Transaction behavior
- Constraint enforcement
- Full-text search behavior
- Migration correctness

### 8.4 Functional Tests (HTTP)

Test the complete system via HTTP requests.

**Location**: `internal/functional_test.go` or `test/functional/`

```go
// test/functional/mcp_test.go
package functional_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "record-sync-mcp/cmd/server"
)

func TestMCP_Functional(t *testing.T) {
    // Setup: create full server with in-memory DB
    srv := server.NewTestServer(t)
    ts := httptest.NewServer(srv.Handler())
    defer ts.Close()

    apiKey := "test-api-key"

    t.Run("complete workflow: create project, record, activate, update", func(t *testing.T) {
        // 1. Create project
        resp := mcpCall(t, ts, apiKey, "create_project", map[string]any{
            "name": "Test Project",
        })
        projectID := resp.Result.(map[string]any)["id"].(string)

        // 2. Create root record
        resp = mcpCall(t, ts, apiKey, "create_record", map[string]any{
            "project_id": projectID,
            "parent_id":  nil,
            "type":       "question",
            "title":      "Main Question",
            "summary":    "The main question",
            "body":       "Full question body",
        })
        recordID := resp.Result.(map[string]any)["record"].(map[string]any)["id"].(string)

        // 3. Activate record
        resp = mcpCall(t, ts, apiKey, "activate", map[string]any{
            "id": recordID,
        })
        sessionID := resp.Result.(map[string]any)["session_id"].(string)
        assert.NotEmpty(t, sessionID)

        // 4. Update record (with session)
        resp = mcpCallWithSession(t, ts, apiKey, sessionID, "update_record", map[string]any{
            "id":   recordID,
            "body": "Updated body content",
        })
        assert.Nil(t, resp.Error)

        // 5. Verify update persisted
        resp = mcpCall(t, ts, apiKey, "get_record_ref", map[string]any{
            "id": recordID,
        })
        // Record should reflect update
    })

    t.Run("conflict detection", func(t *testing.T) {
        // Setup: create record
        resp := mcpCall(t, ts, apiKey, "create_record", map[string]any{
            "type":  "note",
            "title": "Conflict Test",
        })
        recordID := resp.Result.(map[string]any)["record"].(map[string]any)["id"].(string)

        // Session 1 activates
        resp1 := mcpCall(t, ts, apiKey, "activate", map[string]any{"id": recordID})
        session1 := resp1.Result.(map[string]any)["session_id"].(string)

        // Session 2 activates (should get warning)
        resp2 := mcpCall(t, ts, apiKey, "activate", map[string]any{"id": recordID})
        session2 := resp2.Result.(map[string]any)["session_id"].(string)
        conflict := resp2.Result.(map[string]any)["conflict"]
        assert.NotNil(t, conflict, "should warn about concurrent activation")

        // Session 1 updates
        mcpCallWithSession(t, ts, apiKey, session1, "update_record", map[string]any{
            "id":   recordID,
            "body": "Session 1 update",
        })

        // Session 2 updates (should get conflict)
        resp = mcpCallWithSession(t, ts, apiKey, session2, "update_record", map[string]any{
            "id":   recordID,
            "body": "Session 2 update",
        })
        assert.NotNil(t, resp.Result.(map[string]any)["conflict"])
    })

    t.Run("authentication required", func(t *testing.T) {
        req, _ := http.NewRequest("POST", ts.URL+"/mcp", bytes.NewReader([]byte(`{}`)))
        // No Authorization header

        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
    })

    t.Run("tenant isolation", func(t *testing.T) {
        // Tenant 1 creates record
        resp := mcpCall(t, ts, "tenant1-key", "create_record", map[string]any{
            "type":  "note",
            "title": "Tenant 1 Note",
        })
        recordID := resp.Result.(map[string]any)["record"].(map[string]any)["id"].(string)

        // Tenant 2 cannot access
        resp = mcpCall(t, ts, "tenant2-key", "get_record_ref", map[string]any{
            "id": recordID,
        })
        assert.NotNil(t, resp.Error)
        assert.Equal(t, "RECORD_NOT_FOUND", resp.Error.Code)
    })
}

// Helper functions

type JSONRPCResponse struct {
    Result any              `json:"result"`
    Error  *JSONRPCError    `json:"error"`
    ID     int              `json:"id"`
}

type JSONRPCError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

func mcpCall(t *testing.T, ts *httptest.Server, apiKey, method string, params map[string]any) JSONRPCResponse {
    return mcpCallWithSession(t, ts, apiKey, "", method, params)
}

func mcpCallWithSession(t *testing.T, ts *httptest.Server, apiKey, sessionID, method string, params map[string]any) JSONRPCResponse {
    t.Helper()

    body, _ := json.Marshal(map[string]any{
        "jsonrpc": "2.0",
        "method":  method,
        "params":  params,
        "id":      1,
    })

    req, err := http.NewRequest("POST", ts.URL+"/mcp", bytes.NewReader(body))
    require.NoError(t, err)

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+apiKey)
    if sessionID != "" {
        req.Header.Set("Mcp-Session-Id", sessionID)
    }

    resp, err := http.DefaultClient.Do(req)
    require.NoError(t, err)
    defer resp.Body.Close()

    var result JSONRPCResponse
    err = json.NewDecoder(resp.Body).Decode(&result)
    require.NoError(t, err)

    return result
}
```

**What to functional test:**
- Complete user workflows
- Authentication and authorization
- Error responses match spec
- Session management across requests
- Conflict detection end-to-end

### 8.5 Test Organization

```
record-sync-mcp/
├── internal/
│   ├── domain/
│   │   └── record/
│   │       ├── service.go
│   │       └── service_test.go      # Unit tests
│   ├── sqlite/
│   │   ├── record.go
│   │   └── record_test.go           # Integration tests
│   └── repository/
│       └── mocks/
│           └── mocks.go             # Generated mocks
│
├── test/
│   └── functional/
│       ├── mcp_test.go              # HTTP functional tests
│       └── testdata/
│           └── fixtures.json
│
└── Makefile
    # test-unit: go test ./internal/domain/...
    # test-integration: go test ./internal/sqlite/...
    # test-functional: go test ./test/functional/...
    # test: test-unit test-integration test-functional
```

### 8.6 Mock Generation

Use `mockery` to generate mocks from interfaces:

```bash
# Install mockery
go install github.com/vektra/mockery/v2@latest

# Generate mocks for repository interfaces
mockery --dir=internal/repository --name=RecordRepository --output=internal/repository/mocks
mockery --dir=internal/repository --name=SessionRepository --output=internal/repository/mocks
```

---

## 9. Decision Summary

| Decision Area | v1 Choice | Migration Path |
|---------------|-----------|----------------|
| Transport | Plain HTTP POST (JSON-RPC) | Add streaming if standalone agents need it |
| Database | SQLite (MySQL-compatible SQL) | Migrate to Dolt when concurrent agents needed |
| Full-text search | SQLite FTS5 | Migrate to MySQL FULLTEXT with Dolt |
| History tracking | Manual audit tables | Replace with Dolt versioning |
| Authentication | Bearer tokens (API keys) | Extend to OAuth when needed |
| Conflict handling | Session tracking + tick comparison | Dolt branches for advanced merging |

### Why These Choices Work for v1

1. **Simplicity**: Single binary, zero external dependencies for dev/test
2. **Fast iteration**: SQLite enables rapid prototyping
3. **Clear boundaries**: Interface-based design enables future replacement
4. **Testability**: Every layer independently testable

### Next Steps

1. ~~Finalize API spec~~ (done: `0117-mcp-api-spec-draft.md`)
2. Create database schema (MySQL-compatible)
3. Implement repository layer (SQLite)
4. Build domain services
5. Add MCP transport layer
6. Deploy hosted prototype

---

*End of document*
