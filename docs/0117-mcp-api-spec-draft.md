# Record-Sync MCP API Specification (Draft)

*API for maintaining design reasoning across chat sessions*

## Overview

This document specifies the MCP (Model Context Protocol) API for a generalized record-sync system. The system enables chat agents to persist design reasoning across sessions, with support for hierarchical records, session management, and conflict handling.

### Design Principles

1. **Generic records**: Records are not pre-typed as "question" or "conclusion"—type is a field that can be any value suitable to the user or situation
2. **Self-explaining content**: Record bodies must stand alone; the agent handles thought quality
3. **Session-scoped activation**: Records must be activated to reason with them; activation loads minimum required context
4. **Single-writer model**: One chat at a time per record; conflicts warned, not prevented
5. **Explicit sync points**: Changes are synced on activate (down) and save/close (up)

---

## Data Model

### Project

```typescript
interface Project {
  id: string;                    // User-provided or auto-generated
  name: string;                  // Display name
  description?: string;          // Optional description
  created: string;               // ISO timestamp
  tick: number;                  // Monotonic counter, increments on every write
}
```

The `tick` is the project's logical clock. Every write operation (create, update, transition, save) increments this counter. Sessions track the tick at which they last synced, enabling staleness detection independent of wall-clock time.

### Record

```typescript
interface Record {
  id: string;                    // Unique identifier (e.g., "R001")
  type: string;                  // User-defined: "question", "conclusion", "note", etc.
  title: string;                 // Short descriptive title
  summary: string;               // 1-2 sentence summary for listings/references
  body: string;                  // Full self-explaining content
  state: WorkflowState;          // OPEN | LATER | RESOLVED | DISCARDED
  parent_id: string | null;      // Parent record ID, null for roots
  created: string;               // ISO timestamp
  modified: string;              // ISO timestamp
  resolved_by?: string;          // Record ID that resolved this (if RESOLVED)
  related?: string[];            // IDs of related records (non-hierarchical)
  metadata?: Record<string, any>; // Extension point
}

type WorkflowState = "OPEN" | "LATER" | "RESOLVED" | "DISCARDED";
```

### Record Reference

Lightweight pointer used when full content isn't needed.

```typescript
interface RecordRef {
  id: string;
  type: string;
  title: string;
  summary: string;
  state: WorkflowState;
  parent_id: string | null;
  children_count: number;        // How many direct children
  open_children_count: number;   // How many children are OPEN
}
```

### Session

```typescript
interface Session {
  id: string;                    // Unique session identifier
  project_id: string;            // Project this session belongs to
  created: string;               // When session started
  last_activity: string;         // ISO timestamp of last activity
  last_sync_tick: number;        // Project tick at last sync
  active_records: string[];      // Record IDs currently activated
  status: "active" | "stale" | "closed";  // Lifecycle state
  parent_session?: string;       // If branched from another session
}

// Session status meanings:
// - active: Recently synced (last_sync_tick close to project tick)
// - stale: Many project writes have occurred since last sync
// - closed: Properly ended via close_session(), all changes synced

// Staleness is measured by tick gap, not wall-clock time:
// If (project.tick - session.last_sync_tick) > threshold, session is stale
```

### Context Bundle

Returned on activation, contains everything needed to reason with a record.

```typescript
interface ContextBundle {
  target: Record;                           // The activated record (full)
  parent: Record | null;                    // Parent record (full, required for reasoning)
  children: {
    open: Record[];                         // OPEN children (full content)
    other: RecordRef[];                     // LATER/RESOLVED/DISCARDED children (refs only)
  };
  grandchildren: RecordRef[];               // All grandchildren as refs
  warnings?: Warning[];                     // Conflict or staleness warnings
}

interface Warning {
  type: "conflict" | "stale" | "orphan";
  message: string;
  details: any;
}
```

---

## API Commands

### Project Commands

#### create_project

Create a new project.

```typescript
create_project(params: {
  id?: string;                   // Optional: user-provided ID (e.g., "my-app-design")
  name: string;                  // Display name
  description?: string;
}): Project
```

**Notes**:
- If `id` is omitted, system generates a unique ID
- If `id` is provided and already exists, returns error
- User-provided IDs allow pre-configuration in agent context

---

#### list_projects

List all accessible projects.

```typescript
list_projects(): Array<{
  id: string;
  name: string;
  description?: string;
  tick: number;
  open_sessions: number;         // Count of non-closed sessions
  open_records: number;          // Count of OPEN records
}>
```

---

#### get_project

Get project details.

```typescript
get_project(params: {
  id?: string;                   // Optional: defaults to user's default project
}): Project
```

---

### Orientation Commands

These commands do not require activation and are safe for browsing/searching.

#### get_project_overview

Bootstrap command for new sessions. Returns project state and identifies work to continue.

```typescript
get_project_overview(params?: {
  project_id?: string;           // Optional: defaults to user's default project
}): {
  project: {
    id: string;
    name: string;
    description: string;
    tick: number;                // Current project tick (logical clock)
  };
  open_sessions: Array<{
    id: string;
    active_records: string[];
    last_sync_tick: number;
    tick_gap: number;            // project.tick - session.last_sync_tick
  }>;
  root_records: RecordRef[];     // Top-level records
  open_records: RecordRef[];     // All OPEN records across project
  later_records: RecordRef[];    // All LATER records (deferred work)
  recent_activity: ActivityEntry[];
}
```

**Side effects**: None.

**When to use**: Start of every new chat session.

---

#### search_records

Find records by content or metadata.

```typescript
search_records(params: {
  query: string;                           // Full-text search
  types?: string[];                        // Filter by record type
  states?: WorkflowState[];                // Filter by state
  parent_id?: string;                      // Scope to subtree
  limit?: number;                          // Default: 20
}): {
  results: Array<RecordRef & {
    relevance: number;                     // 0-1 score
    snippet: string;                       // Matching text excerpt
  }>;
  total: number;
}
```

**Side effects**: None.

---

#### list_records

List records with filters.

```typescript
list_records(params?: {
  states?: WorkflowState[];
  types?: string[];
  parent_id?: string;                      // null for roots only
  depth?: number;                          // How deep to traverse (default: 1)
}): RecordRef[]
```

**Side effects**: None.

---

#### get_record_ref

Get reference for a specific record without activating.

```typescript
get_record_ref(params: {
  id: string;
}): RecordRef
```

**Side effects**: None.

---

### Activation Commands

These commands load context for reasoning.

#### activate

Load a record and its required context for reasoning. This is the primary "sync-down" operation.

```typescript
activate(params: {
  id: string;
  override?: boolean;                      // Proceed despite conflict warning
}): {
  session_id: string;                      // Current session (created if needed)
  context: ContextBundle;
  already_loaded: boolean;                 // True if re-activating in same session
  conflict?: {
    session_id: string;
    last_activity: string;
    message: string;
  };
}
```

**Context loading rules**:
- Parent: Full content (required for reasoning)
- Target: Full content
- OPEN children: Full content
- LATER/RESOLVED/DISCARDED children: Reference only
- Grandchildren: Reference only

**Side effects**:
- Creates session if none exists
- Adds record to session's active_records
- Logs activation event

**Conflict handling**:
- If record is active in another session, returns `conflict` object
- If `override: false` (default), still returns context but flags the conflict
- Agent should warn user and ask for confirmation before proceeding

---

#### sync_session

Pull changes made by other sessions. Called when resuming work or periodically.

```typescript
sync_session(params?: {
  session_id?: string;                     // Default: current session
}): {
  project_tick: number;                    // Current project tick
  session_tick_before: number;             // Session's tick before sync
  tick_gap: number;                        // How many writes occurred while session was idle
  changes: Array<{
    record_id: string;
    change_type: "modified" | "state_changed" | "created" | "deleted";
    old_value?: any;
    new_value?: any;
    by_session: string;
    at_tick: number;                       // Project tick when change occurred
  }>;
  session_status: "active" | "stale";
  warning?: string;                        // e.g., "47 writes occurred since your last sync"
}
```

**Side effects**:
- Updates session's last_sync_tick to current project tick
- Updates session's last_activity timestamp

---

### Mutation Commands

These commands modify records and require activation.

#### create_record

Create a new record as child of an active record (or as root).

```typescript
create_record(params: {
  parent_id: string | null;                // null for root record
  type: string;
  title: string;
  summary: string;
  body: string;
  state?: WorkflowState;                   // Default: OPEN
  related?: string[];
}): {
  record: Record;                          // Full created record
  auto_activated: boolean;                 // New records are auto-activated
}
```

**Preconditions**:
- If parent_id is provided, parent must be activated in current session
- If parent_id is null, no precondition (creating root)

**Side effects**:
- Assigns sequential ID
- Adds to session's active_records
- Logs record_created event

---

#### update_record

Modify an active record's content.

```typescript
update_record(params: {
  id: string;
  title?: string;
  summary?: string;
  body?: string;
  related?: string[];
  force?: boolean;                         // Override conflict, use this version
}): {
  record: Record;
  conflict?: {
    message: string;
    other_version: Record;                 // Version from other session
  };
}
```

**Preconditions**: Record must be activated in current session.

**Conflict handling**:
- If record was modified by another session since activation, returns `conflict`
- Agent should present both versions to user for resolution
- Use `force: true` to override conflict and apply this update regardless

**Side effects**:
- Updates modified timestamp
- Logs record_updated event

---

#### transition

Change a record's workflow state.

```typescript
transition(params: {
  id: string;
  to_state: WorkflowState;
  reason?: string;                         // Required for LATER, DISCARDED
  resolved_by?: string;                    // Required for RESOLVED
}): {
  record: Record;
  cascade_warning?: {                      // If transitioning affects children
    open_children: RecordRef[];
    message: string;
  };
}
```

**Valid transitions**:
- OPEN → LATER (with reason)
- OPEN → RESOLVED (with resolved_by)
- OPEN → DISCARDED (with reason)
- LATER → OPEN
- LATER → DISCARDED (with reason)
- RESOLVED → OPEN (reopening)
- DISCARDED → OPEN (recovery)

**Preconditions**: Record must be activated in current session.

**Side effects**:
- Updates state and modified timestamp
- If RESOLVED, sets resolved_by field
- Logs state_transition event
- If has OPEN children, returns cascade_warning (doesn't auto-cascade)

---

### Session Lifecycle Commands

#### save_session

Persist current state without ending session. This is a "sync-up" operation.

```typescript
save_session(params?: {
  summary?: string;                        // For activity log
}): {
  success: boolean;
  saved_records: string[];                 // IDs of records with changes
  last_save: string;                       // Timestamp
}
```

**Side effects**:
- Updates session's last_save
- Logs session_saved event with summary
- Commits to storage (implementation-dependent)

---

#### close_session

End session cleanly. Performs final sync-up.

```typescript
close_session(params?: {
  summary?: string;
}): {
  success: boolean;
  deactivated_records: string[];
  unsaved_warning?: string;                // If changes since last save
}
```

**Side effects**:
- Clears session's active_records
- Logs session_closed event
- Session marked as ended (can be resumed but starts fresh)

---

### History and Conflict Commands

#### get_record_history

View change history for a record.

```typescript
get_record_history(params: {
  id: string;
  since?: string;                          // ISO timestamp
  limit?: number;
}): Array<{
  timestamp: string;
  session_id: string;
  change_type: string;
  summary: string;
  diff?: string;                           // For body changes
}>
```

---

#### get_record_diff

Compare record versions.

```typescript
get_record_diff(params: {
  id: string;
  from: string;                            // Timestamp or "last_save"
  to?: string;                             // Default: current
}): {
  from_version: Record;
  to_version: Record;
  diff: {
    title?: { old: string; new: string };
    summary?: { old: string; new: string };
    body?: string;                         // Unified diff format
    state?: { old: WorkflowState; new: WorkflowState };
  };
}
```

---

#### get_active_sessions

Check what sessions are working on a record.

```typescript
get_active_sessions(params: {
  record_id: string;
}): Array<{
  session_id: string;
  last_activity: string;
  is_current: boolean;
}>
```

---

### Activity Log

#### get_recent_activity

View recent events across the project.

```typescript
get_recent_activity(params?: {
  limit?: number;                          // Default: 50
  since?: string;                          // ISO timestamp
  types?: string[];                        // Filter by event type
  record_id?: string;                      // Filter to specific record
}): ActivityEntry[]

interface ActivityEntry {
  timestamp: string;
  type: ActivityType;
  session_id: string;
  record_id?: string;
  summary: string;
  details?: any;
}

type ActivityType =
  | "record_created"
  | "record_updated"
  | "state_transition"
  | "session_started"
  | "session_saved"
  | "session_closed"
  | "session_branched"
  | "activation"
  | "conflict_detected"
  | "conflict_resolved";
```

---

## Command Summary Table

| Category | Command | Requires Activation | Mutates Data | Increments Tick |
|----------|---------|---------------------|--------------|-----------------|
| Project | create_project | No | Yes | No (new project) |
| Project | list_projects | No | No | No |
| Project | get_project | No | No | No |
| Orientation | get_project_overview | No | No | No |
| Orientation | search_records | No | No | No |
| Orientation | list_records | No | No | No |
| Orientation | get_record_ref | No | No | No |
| Activation | activate | No* | Yes (session) | No |
| Activation | sync_session | No | Yes (session) | No |
| Mutation | create_record | Yes (parent) | Yes | **Yes** |
| Mutation | update_record | Yes | Yes | **Yes** |
| Mutation | transition | Yes | Yes | **Yes** |
| Session | save_session | No | Yes (session) | **Yes** |
| Session | close_session | No | Yes (session) | No |
| History | get_record_history | No | No | No |
| History | get_record_diff | No | No | No |
| History | get_active_sessions | No | No | No |
| Activity | get_recent_activity | No | No | No |

*`activate` doesn't require prior activation but creates activation state.

**Tick-incrementing operations**: create_record, update_record, transition, save_session. These are the "writes" that advance the project's logical clock.

---

## Workflow Patterns

### Pattern 1: Continue Existing Work

```
get_project_overview()
→ identify record to continue
activate(R7)
→ reason with context
update_record(R7, ...)
create_record(parent=R7, ...)
save_session()
... more work ...
close_session()
```

### Pattern 2: Start New Topic

```
get_project_overview()
→ confirm topic doesn't exist
create_record(parent=null, type="question", ...)
→ auto-activated
... reason and create children ...
save_session()
close_session()
```

### Pattern 3: New Chat for a Subtopic

```
[In session S1, working on R7]
create_record(parent=R7, type="question", title="Subtopic", ...)
→ returns R15
[User opens new chat]
activate(R15)
→ context includes R7 as parent
... deep work on R15 ...
close_session()
[Return to S1]
sync_session()
→ shows R15 was resolved
```

### Pattern 4: Conflict Resolution

```
[S1 and S2 both have R7 active]
[S1]: update_record(R7, body="version A")
[S2]: update_record(R7, body="version B")
→ S2 receives conflict response
→ agent presents both versions
→ user chooses resolution
[S2]: update_record(R7, body="merged version", force=true)
```

---

## Error Handling

### Error Response Format

```typescript
interface ErrorResponse {
  error: {
    code: string;
    message: string;
    details?: any;
    recovery_hint?: string;
  };
}
```

### Error Codes

| Code | Meaning | Recovery Hint |
|------|---------|---------------|
| `RECORD_NOT_FOUND` | Record ID doesn't exist | Check ID spelling |
| `NOT_ACTIVATED` | Operation requires activation | Call activate() first |
| `INVALID_TRANSITION` | State transition not allowed | Check valid transitions |
| `PARENT_NOT_ACTIVATED` | Creating child without parent active | Activate parent first |
| `CONFLICT` | Record modified by another session | Sync and resolve |
| `SESSION_NOT_FOUND` | Session expired or invalid | Start new session |
| `DEPTH_EXCEEDED` | Hierarchy too deep | Consider flattening |

---

## Implementation Notes

### Default Project

Every user has a default project used when `project_id` is not specified:
- Created automatically on first use
- Commands that take `project_id` parameter default to this project
- Agent context can be pre-configured with a specific `project_id` to target a non-default project

User-provided project IDs (via `create_project`) enable:
- Pre-configuring agent context to work with a specific project
- Meaningful, memorable project identifiers (e.g., "myapp-v2-design")
- Multiple agents/contexts working on different projects

### Session Persistence

Sessions should persist across chat restarts within the same conversation. Implementation options:
- Store session ID in chat metadata
- Use conversation ID as session ID
- Persist to same storage as records

### Activation Caching

Within a session, record content should be cached after first load. On subsequent access:
- If record unchanged, return cached version
- If record changed externally, return updated version with diff indicator

### Conflict Windows

Conflicts are detected by comparing `modified` timestamps. Acceptable conflict window depends on implementation (typically seconds to minutes).

### Storage Backend

This spec is storage-agnostic. Implementations may use:
- Filesystem (like existing ganot-fs)
- Database (SQL or document)
- Cloud storage with change tracking

---

## Migration from Previous Design

### Key Changes from ganot-fs

| Previous | New |
|----------|-----|
| Separate types: Thread, Question, Conclusion | Single Record type with `type` field |
| Status: open, active, resolved, deferred | State: OPEN, LATER, RESOLVED, DISCARDED |
| Threads own records | Parent-child hierarchy, no separate "thread" |
| Implicit sessions | Explicit Session objects |
| activate_thread | activate (any record) |
| Thread body = scratchpad | Every record's body is persistent |

### What "Thread" Becomes

The previous "thread" concept (ephemeral scratchpad for session continuity) can be modeled as:
- A record with type="thread" at the session focus
- Or simply use the root record of a reasoning line as the entry point

The choice is user/agent preference. The system doesn't enforce thread semantics.

---

## Version

Draft 0.1 — January 2025
