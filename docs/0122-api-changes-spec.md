# Functional Spec: API Simplification

*Removing branch_session and focus_record*

## Summary

This spec describes changes to simplify the session model by removing `branch_session` and the `focus_record` concept. The simplified model treats sessions as lightweight tracking state with no special "focus" record.

---

## Changes

### Remove: `branch_session` Command

**Current behavior**: Creates a new session pre-loaded with context from a specified record.

```typescript
// REMOVE THIS
branch_session(params: {
  session_id?: string;  // parent session
  focus_record: string; // record to center new session on
}): {
  session_id: string;
  inherited_context: RecordRef[];
  launch_url?: string;
}
```

**Rationale**:
- The same result is achieved by opening a new chat and calling `activate(record_id)`
- Removes complexity around `launch_url` generation (MCP server shouldn't create chats)
- Eliminates `focus_record` concept from the model

**Migration**: Users open new chat manually, then call `activate` on the desired record.

### Remove: `focus_record` from Session Model

**Current behavior**: Sessions could have a `focus_record` indicating primary working target.

**Change**: Sessions track only `active_records[]` (list of activated record IDs) and `last_sync_tick`.

**Rationale**:
- Multiple records can be active; no single record is special
- Focus was only used for branching, which is removed
- Simpler mental model: session = set of active records

### Update: `get_project_overview` Response

**Current response** includes:
```typescript
{
  open_sessions: [{
    id: string;
    focus: string;      // REMOVE
    last_activity: string;
  }]
}
```

**New response**:
```typescript
{
  open_sessions: [{
    id: string;
    active_records: string[];  // list of record IDs
    last_sync_tick: number;
    tick_gap: number;          // staleness indicator
  }]
}
```

### Update: Core Requirements Document

Remove this line from `0116-core-requirements.md`:

> A session may be branched into a new session (typically in a new chat). The branched session retains the active context from original session. The branch operation also performs sync-down.

Replace with:

> To work on a record in a new chat, the user opens the chat and the agent calls `activate(record_id)`. Context is loaded per standard activation rules.

---

## Unchanged

### Session Commands

These remain as-is:

| Command | Purpose |
|---------|---------|
| `activate(id)` | Load record into session |
| `sync_session()` | Refresh staleness, pull changes |
| `save_session()` | Checkpoint |
| `close_session()` | End session |

### Activation Behavior

Activation continues to:
- Load target record (full)
- Load parent (full)
- Load OPEN children (full)
- Load other children as references
- Track record in `session.active_records`
- Warn on conflict with other sessions

### Conflict Handling

No changes to conflict detection or resolution:
- `activate` returns warning if record active elsewhere
- `update_record` supports `force=true` for conflict override
- `get_active_sessions(record_id)` shows who's working on a record

---

## Agent Documentation Updates

### Session Start Guidance

Update agent docs to describe the standard session start flow:

1. Call `get_project_overview()` for orientation
2. Present open sessions with staleness warnings
3. Query for open records of interest (e.g., `list_records(types=["thread"], states=["OPEN"])`)
4. When user chooses work target, call `activate(record_id)`

### Multi-Record Sessions

Clarify that sessions naturally support multiple active records:

```
Agent: activate(R7)
  → R7 now active
Agent: activate(R15)
  → R15 now active (R7 remains active)
```

This enables comparison, navigation, and working across related records without special "branching."

### New Chat Workflow

When user wants to explore a subtopic in new chat:

1. **Current session**: Create record for subtopic if needed
2. **User**: Opens new chat
3. **New session**: Agent calls `activate(subtopic_record_id)`
4. Context loads automatically per activation rules

No special handoff mechanism needed.

---

## Implementation Notes

### Database Schema

If `focus_record` exists in session table, it can be:
- Dropped (if no valuable data)
- Kept as nullable, ignored by API (if migration is complex)

### API Versioning

This is a breaking change if clients use `branch_session`. Options:
- Remove in next major version
- Return error with migration message if called
- Since system is pre-release, clean removal is acceptable

---

## Testing Checklist

- [ ] `branch_session` returns error or is removed from tool list
- [ ] `get_project_overview` returns `active_records` instead of `focus`
- [ ] Sessions can have multiple active records
- [ ] Activation in new session loads correct context without prior branching
- [ ] All scenario flows work without branching
