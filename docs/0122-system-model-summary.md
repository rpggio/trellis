# Threds: System Model Summary

*A compact reference for the core model, principles, and workflows*

---

## Core Model

### Records

A record is a self-explaining unit of design reasoning:

| Field | Purpose |
|-------|---------|
| `id` | Unique identifier |
| `title` | Short name |
| `summary` | One-line description |
| `body` | Full content |
| `type` | User-defined (question, conclusion, thread, etc.) |
| `state` | Workflow state |
| `parent_id` | Hierarchical parent (null for roots) |
| `related` | Cross-references (non-hierarchical) |

Records form a tree via `parent_id`. Cross-references via `related` don't create hierarchy.

### Workflow States

| State | Meaning | Context Loading |
|-------|---------|-----------------|
| OPEN | Active work | Full content |
| LATER | Intentionally deferred | Reference only |
| RESOLVED | Completed | Reference only |
| DISCARDED | Abandoned | Reference only |

### Projects

A project is a container for records with:
- Logical clock (`tick`) incremented on writes
- Default project used when none specified

### Sessions

A session tracks a chat's interaction with records:
- `active_records[]` — records loaded for reasoning
- `last_sync_tick` — staleness baseline

Sessions are lightweight. No special "focus" record.

---

## Context Loading

When activating a record:

| Relationship | Loaded |
|--------------|--------|
| Parent | Full content |
| Target | Full content |
| OPEN children | Full content |
| Other children | Reference only |
| Grandchildren | Reference only |

This provides sufficient context for reasoning without loading entire tree.

---

## Commands

### Orientation (no activation needed)

- `get_project_overview()` — project tick, open sessions, root refs
- `list_records(filters)` — browse by state/type
- `search_records(query)` — find by content
- `get_record_ref(id)` — lightweight lookup
- `get_recent_activity()` — cross-session awareness

### Reasoning (requires activation)

- `activate(id)` — load record context
- `create_record(...)` — create record (auto-activates if a session is active)
- `update_record(id, ...)` — modify content
- `transition(id, state)` — change workflow state

### Session Lifecycle

- `sync_session()` — check staleness (tick gap), refresh session baseline
- `save_session()` — checkpoint
- `close_session()` — end session

---

## Principles

### User Control Over Schema

The API is type-agnostic. Record types are user-defined strings.

- System doesn't enforce what types exist
- System doesn't enforce type-specific rules
- Schemes (skill files) define types and their semantics
- Users can create custom schemes or use none

**Examples**:
- Question/conclusion scheme: `type="question"`, `type="conclusion"`
- Threads scheme: `type="thread"` for grouping conversational work
- Custom: `type="spec"`, `type="decision"`, `type="risk"`

### Opt-in Threads

Threads are an optional organizational pattern, not a core concept.

- A thread is a record with `type="thread"`
- Threads group conversational/session-contextual work
- Project-level artifacts don't require threads
- Users who don't want threads simply don't create them

See `threds-threads.SKILL.md` for the threads pattern.

### Override by Default

Users can override system defaults:
- Activate records despite conflict warnings
- Force updates over concurrent edits
- Create any record type without validation
- Organize hierarchy however they want

System warns; user decides.

### Records are Self-Explaining

Each record must be comprehensible without the originating conversation. The chat that created it won't be available to future readers.

**The Stranger Test**: Would someone with domain expertise but no conversation access understand this document?

---

## Workflows

### Starting a Session

```
Chat opens
  ↓
Agent: get_project_overview()
  → Orientation: tick, open sessions, roots
  ↓
Agent: list_records(...) or search_records(...)
  → Find relevant records
  ↓
User chooses target
  ↓
Agent: activate(record_id)
  → Context loaded, session tracking begins
```

### Creating Records

```
Agent: create_record(parent, type, title, body)
  → Record created, auto-activated if a session is active
  ↓
Work continues on new record
```

### Resolving Work

```
Agent: transition(record_id, to_state=RESOLVED, resolved_by=conclusion_id)
  → Record closed, children may need review
```

### Ending a Session

```
User: "Let's wrap up"
  ↓
Agent: save_session()  // checkpoint
Agent: close_session() // end tracking
```

### Handling Staleness

```
Returning to old session
  ↓
Agent: sync_session()
  → Returns tick gap and session status
  ↓
Agent: "X writes occurred. Review changes?"
  ↓
[User decides how to proceed]
```

If the user wants details, follow with `get_recent_activity()` or `list_records(...)` since `sync_session()` only reports staleness.

### Handling Conflicts

```
Agent: activate(record_id)
  → Warning: "Active in session S1"
  ↓
Agent: "This record is being worked on elsewhere. Continue?"
  ↓
User: "Yes" or "No"
  ↓
[If yes, proceed with awareness of potential merge conflicts]
```

---

## Threads (Opt-in)

When using the threads pattern:

### Session Start with Threads

```
Agent: get_project_overview()
Agent: list_records(types=["thread"], states=["OPEN"])
  → Open threads
  ↓
Agent: "Open threads: [list]. Continue one, or start new?"
  ↓
[User chooses]
  ↓
Agent: activate(thread_id)  // or create new thread
```

### Record Classification

| Category | Types | Parent |
|----------|-------|--------|
| Conversational | question, working-note | Thread |
| Project-level | decision, principle | Root or other project record |

### Creating Conversational Records

```
No thread active
  ↓
Agent: create_record(type="thread", title="...", body="...")
  → Thread created
  ↓
Agent: create_record(parent=thread_id, type="question", ...)
  → Question under thread
```

### Closing a Thread

```
Agent: transition(thread_id, to_state=RESOLVED, reason="...")
  → Thread closed, children persist
```

---

## Key Constraints

1. One chat at a time per record (warning on conflict, user can override)
2. Parent context required for reasoning
3. Open sessions assumed to have unsaved work
4. Staleness = tick gap, not wall-clock time
5. Records form tree (no cycles)
6. Depth limit is agent guidance for now (warn at 5, avoid >10)
