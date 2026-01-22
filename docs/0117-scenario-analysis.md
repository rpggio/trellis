# Scenario Analysis: Record-Sync MCP for Design Reasoning

*Modeling chat workflows for generalized record synchronization*

## Design Context

This document models the 90% most significant chat scenarios for a system that maintains design reasoning across chat sessions. The system uses generic **records** (which can be typed as question, conclusion, or whatever suits the situation) organized in parent-child hierarchies within **projects**.

### Core Concepts

- **Project**: A container for related design work, with a logical clock (tick) for tracking activity
- **Record**: A self-explaining unit of design reasoning with ID, title, summary, body, type, and workflow state
- **Record reference**: Lightweight pointer containing ID, title, and summary (no body)
- **Session**: A chat's connection to the record system, tracking which records are active
- **Activation**: Loading a record into a session's working context for reasoning
- **Workflow states**: OPEN | LATER | RESOLVED | DISCARDED
- **Tick**: Monotonic counter incremented on each write operation; the project's logical clock

### Key Constraints

1. Every session belongs to a project (default project used when not specified)
2. Reasoning on a record requires its parent context to be loaded
3. One chat at a time may reason against any given record
4. Sessions may branch into new chats (e.g., "dive deeper on this in a new thread")
5. Open sessions are assumed to contain unsaved information
6. Staleness is measured by tick gap (project writes since last sync), not wall-clock time
7. The agent handles thought content; we model workflow and operations

---

## Variable Notation

Throughout these scenarios, we use variables to represent thoughts and records:

- `R1, R2, R3...` — Records (generic)
- `P` — Parent record
- `C1, C2...` — Child records
- `S1, S2...` — Sessions
- `T1, T2...` — Thoughts (content in chat, not yet persisted)
- `ctx(R)` — The context loaded for record R

---

## Scenario 1: Cold Start — New Chat, Existing Project

**Situation**: User opens a new chat. Project has existing records from previous work.

### Flow

```
Chat opens
  ↓
Agent calls: get_project_overview()
  → Returns: project summary, open sessions warning, record tree (references only)
  ↓
User: "Let's continue working on the caching design"
  ↓
Agent calls: search_records("caching")
  → Returns: [R7: "Caching strategy", state=OPEN, children=[R12, R13]]
  ↓
Agent calls: activate(R7)
  → Returns: {
      record: R7 (full content),
      parent: P (full content, since R7 needs context),
      children: {
        R12: reference (state=RESOLVED),
        R13: full content (state=OPEN),
        R14: grandchild of R13, reference only
      }
    }
  ↓
Session S1 now tracks: active_records = [R7, R13]
```

### Context Loaded

| Record | What's loaded | Why |
|--------|---------------|-----|
| P (parent of R7) | Full content | Required for reasoning on R7 |
| R7 | Full content | Target of activation |
| R12 | Reference only | State is RESOLVED |
| R13 | Full content | State is OPEN |
| R14 | Reference only | Grandchild (depth limit) |

### Commands Available

- `create_record(parent=R7, type, title, body)` — Create child of active record
- `update_record(R7, body)` — Modify active record
- `transition(R7, new_state)` — Change workflow state
- `activate(R12)` — Load resolved child for deeper inspection
- `save_session()` — Persist changes, mark session clean
- `close_session()` — Sync-up and end session

---

## Scenario 2: Deep Dive — New Chat for Subtopic

**Situation**: During discussion in S1, a subtopic warrants focused exploration in a separate chat.

### Flow

```
In S1, working on R7 "Caching strategy"
  ↓
Discussion reveals: "Redis vs Memcached needs deep analysis"
  ↓
User: "Let's explore this in a new thread"
  ↓
Agent calls: create_record(parent=R7, type="question", title="Redis vs Memcached", summary="Should we use Redis or Memcached for caching?", body=T1)
  → Returns: R15 (new record, state=OPEN)
  ↓
User opens new chat (S2)
  ↓
Agent calls: activate(R15)
  → Context includes:
      - R15 (full content)
      - R7 loaded (parent, full content)
      - P loaded (grandparent, reference only)
```

### State After Branch

| Session | Active Records | Status |
|---------|----------------|--------|
| S1 | R7, R13 | Open (may have unsaved T) |
| S2 | R15 | Open (fresh) |

### Key Insight

Opening a new chat and activating the subtopic record loads the same context. The parent session remains open—user may return to it.

---

## Scenario 3: Convergence — Resolving Child Affects Parent

**Situation**: In S2, the Redis vs Memcached question is resolved. This affects the parent caching strategy.

### Flow

```
In S2, working on R15 "Redis vs Memcached"
  ↓
Discussion concludes: "Redis for our use case because X, Y, Z"
  ↓
Agent calls: create_record(parent=R15, type="conclusion", title="Redis selected", summary="Redis chosen for caching due to persistence and data structure support", body=T2)
  → Returns: R16 (state=OPEN)
  ↓
Agent calls: transition(R15, to_state=RESOLVED, resolved_by=R16)
  → R15 now RESOLVED
  ↓
Agent calls: save_session()
  → S2 synced
  ↓
[Later, user returns to S1]
  ↓
Agent calls: sync_session()
  → Returns: {
      changes: [
        { record: R15, old_state: OPEN, new_state: RESOLVED },
        { record: R16, event: "created" }
      ]
    }
  ↓
Agent incorporates: "The Redis vs Memcached question has been resolved in a parallel thread. Redis was selected."
```

### Commands for Cross-Session Awareness

- `sync_session()` — Pull changes made by other sessions
- `get_record_history(R15)` — See what happened to a record

---

## Scenario 4: Parallel Chats — Conflict Warning

**Situation**: User has S1 open with R7 active. In a different browser tab, opens new chat S3 and tries to work on R7.

### Flow

```
S3 opens
  ↓
Agent calls: activate(R7)
  → Returns: {
      warning: "R7 is currently active in session S1 (last activity: 10 min ago)",
      record: R7 (full content),
      conflict_mode: true
    }
  ↓
Agent to user: "This record is being worked on in another chat. Changes may conflict. Continue anyway?"
  ↓
User: "Yes, continue"
  ↓
Agent calls: activate(R7, override=true)
  → S3 now has R7 active, with conflict flag
  ↓
[User makes changes in S3]
  ↓
Agent calls: update_record(R7, body=T3)
  → Returns: {
      warning: "Record was also modified in S1 at [timestamp]",
      merge_required: true,
      versions: { S1: body_v1, S3: body_v2 }
    }
```

### Conflict Resolution Options

1. **Last-write-wins**: Simpler, may lose information
2. **Merge prompt**: Agent shows both versions, asks user to reconcile
3. **Block**: Refuse update, force user to close other session first

### Recommended Approach

Warn on activation, require override acknowledgment, present merge conflicts on save.

---

## Scenario 5: Returning to Stale Session

**Situation**: User had S1 open, closed browser without saving. Later, opens the same chat. Meanwhile, other sessions made writes to the project.

### Flow

```
Chat reopens (same conversation, S1 still "open" in system)
  ↓
Agent calls: sync_session()
  → Returns: {
      project_tick: 147,
      session_tick_before: 98,
      tick_gap: 49,
      session_status: "stale",
      warning: "49 writes have occurred since your last sync",
      changes: [
        { record_id: R13, change_type: "modified", by_session: S2, at_tick: 112 },
        { record_id: R18, change_type: "created", by_session: S2, at_tick: 130 }
      ]
    }
  ↓
Agent to user: "49 writes have occurred in this project since your last sync. Some records you were working on have been modified. Would you like to review changes before continuing?"
  ↓
User: "Show me what changed in R13"
  ↓
Agent calls: get_record_diff(R13, since_tick=98)
  → Returns: diff of R13
```

### Session Lifecycle and Staleness

Session staleness is measured by **tick gap**, not wall-clock time:
- **Active**: `tick_gap` is small (session recently synced relative to project activity)
- **Stale**: `tick_gap` is large (many writes occurred while session was idle)
- **Closed**: Properly ended via `close_session()`, all changes synced

A session open for 6 months is not stale if the project had no activity. A session open for 5 minutes is stale if 50 writes occurred in that time.

---

## Scenario 6: Review Mode — Surveying Open Work

**Situation**: User wants to see all open items across the project before deciding what to work on.

### Flow

```
New chat opens
  ↓
User: "What's open across the project?"
  ↓
Agent calls: get_project_overview()
  → Returns: {
      open_sessions: [
        { id: S1, focus: R7, last_activity: "2h ago", warning: "may have unsaved work" }
      ],
      open_records: [
        { ref: R7, title: "Caching strategy", children_open: 2 },
        { ref: R13, title: "Cache invalidation", parent: R7 },
        { ref: R22, title: "API versioning", children_open: 0 }
      ]
    }
  ↓
Agent presents summary without activating anything
  ↓
User: "Let's close out the caching work"
  ↓
Agent calls: activate(R7)
  → Loads context per Scenario 1 rules
```

### Key Insight

Review mode (browsing, searching, surveying) doesn't require activation. Activation is only needed when reasoning on a record—when the agent might create, modify, or transition records.

---

## Scenario 7: Backwards Reference — Revisiting Ancestor

**Situation**: While working on R15 (child of R7), user wants to reconsider something in the grandparent record.

### Flow

```
In S2, R15 active, R7 loaded as parent
  ↓
User: "Actually, I want to revisit the project scope that led to R7"
  ↓
Agent calls: activate(P)
  → Returns: {
      record: P (full content),
      parent: null (P is root),
      children: [R7, R8, R9] as references (only R7 is OPEN)
      note: "R15 remains active; P is now also active"
    }
  ↓
Session S2 now tracks: active_records = [R15, P]
  ↓
Agent can now reason about P with full context
```

### Multiple Activations

A session can have multiple records active simultaneously. This enables:
- Comparing related records
- Navigating up and down the hierarchy
- Working on siblings

---

## Scenario 8: Creating Root Record

**Situation**: Starting a completely new line of reasoning with no parent.

### Flow

```
Chat opens
  ↓
User: "I want to explore authentication approaches—this is a new topic"
  ↓
Agent calls: get_project_overview()
  → No existing records match "authentication"
  ↓
Agent calls: create_record(parent=null, type="question", title="Authentication approach", summary="Exploring authentication methods for the application", body=T1)
  → Returns: R23 (new root record, state=OPEN)
  ↓
R23 automatically activated in current session
```

### Root Records

Records with no parent are entry points—top-level topics or themes. The project can have multiple root records representing different areas of exploration.

---

## Scenario 9: Discarding Abandoned Work

**Situation**: A line of reasoning turned out to be a dead end.

### Flow

```
Working on R15, realize this approach won't work
  ↓
User: "This whole Redis exploration was misguided—let's abandon it"
  ↓
Agent calls: transition(R15, to_state=DISCARDED, reason="Approach was misguided—Redis vs Memcached is irrelevant to our actual needs")
  → R15 state = DISCARDED
  → Children of R15 (if any) flagged for review
  ↓
Agent to user: "R15 discarded. It had 2 child records. Should they also be discarded?"
  ↓
User: "Yes"
  ↓
Agent calls: transition(R16, to_state=DISCARDED, reason="Parent discarded")
```

### DISCARDED vs RESOLVED

- **RESOLVED**: Work completed successfully, conclusion reached
- **DISCARDED**: Work abandoned, not useful, but preserved for history

Both states reduce context loading (only reference returned, not full content).

---

## Scenario 10: Deferring for Later

**Situation**: Question is valid but not actionable now.

### Flow

```
Working on R15 "Redis vs Memcached"
  ↓
User: "We can't decide this until we know our scale requirements. Park it."
  ↓
Agent calls: transition(R15, to_state=LATER, reason="Blocked on scale requirements analysis")
  → R15 state = LATER
  ↓
[Future session]
  ↓
Agent calls: list_records(state=LATER)
  → Returns: [R15, R20, ...]
  ↓
User: "We now know our scale. Let's revisit R15"
  ↓
Agent calls: transition(R15, to_state=OPEN)
Agent calls: activate(R15)
```

### LATER State

- Indicates intentional deferral, not abandonment
- Record body should capture why it was deferred
- System can remind user of LATER records periodically

---

## Context Loading Summary

### On Activation of Record R

| Relationship | State | What's Loaded |
|--------------|-------|---------------|
| Parent of R | any | Full content (required for reasoning) |
| R itself | any | Full content (target of activation) |
| Child of R | OPEN | Full content |
| Child of R | LATER | Reference only |
| Child of R | RESOLVED | Reference only |
| Child of R | DISCARDED | Reference only |
| Grandchild | any | Reference only |

### Within a Session

- Content for a record is not loaded twice
- If record changed since last load, system returns diff or updated record
- Session tracks which records are active

---

## Command Summary by Phase

### Orientation Phase (no activation needed)

| Command | Purpose |
|---------|---------|
| `get_project_overview()` | Bootstrap: project state, open sessions, record tree |
| `search_records(query)` | Find records by content |
| `list_records(filters)` | List by state, type, etc. |
| `get_record_ref(id)` | Get reference without activation |

### Reasoning Phase (requires activation)

| Command | Purpose |
|---------|---------|
| `activate(id)` | Load record context for reasoning |
| `create_record(parent, type, title, body)` | Create child record |
| `update_record(id, body)` | Modify record content |
| `transition(id, state)` | Change workflow state |

### Session Lifecycle

| Command | Purpose |
|---------|---------|
| `sync_session()` | Pull external changes |
| `save_session()` | Persist current state |
| `close_session()` | End session, sync-up |

### Conflict Handling

| Command | Purpose |
|---------|---------|
| `get_active_sessions(record_id)` | Check who else is working on record |
| `get_record_diff(id, since)` | See changes since timestamp |
| `update_record(id, body, force=true)` | Resolve conflicting edits by forcing update |

---

## Information Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Record Store                               │
│  ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐             │
│  │   R1    │──▶│   R7    │──▶│  R15    │──▶│  R16    │             │
│  │ (root)  │   │ (open)  │   │(resolved)│   │ (open)  │             │
│  └─────────┘   └─────────┘   └─────────┘   └─────────┘             │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
          ┌─────────────────────┼─────────────────────┐
          │                     │                     │
          ▼                     ▼                     ▼
    ┌───────────┐         ┌───────────┐         ┌───────────┐
    │ Session 1 │         │ Session 2 │         │ Session 3 │
    │ active: R7│         │active: R15│         │  (new)    │
    │ stale     │         │          │         │           │
    └───────────┘         └───────────┘         └───────────┘
          │                     │                     │
          ▼                     ▼                     ▼
    ┌───────────┐         ┌───────────┐         ┌───────────┐
    │  Chat 1   │         │  Chat 2   │         │  Chat 3   │
    │ (browser) │         │ (browser) │         │ (browser) │
    └───────────┘         └───────────┘         └───────────┘
```

---

## Scenario 11: Activity Log Review

**Situation**: User wants to understand what happened across multiple sessions.

### Flow

```
New chat opens
  ↓
User: "What's been happening on this project recently?"
  ↓
Agent calls: get_recent_activity(limit=20)
  → Returns: [
      { type: "state_transition", record_id: R15, to_state: RESOLVED, session: S2, ts: "1h ago" },
      { type: "record_created", record_id: R16, session: S2, ts: "2h ago" },
      { type: "session_saved", session: S1, summary: "Explored caching options", ts: "3h ago" },
      { type: "record_updated", record_id: R7, session: S1, ts: "3h ago" },
      ...
    ]
  ↓
Agent summarizes: "In the last 3 hours: The Redis vs Memcached question was resolved (R15→RESOLVED).
A conclusion record was created (R16). Session S1 saved progress on caching exploration."
  ↓
User: "What specifically changed in R7?"
  ↓
Agent calls: get_record_history(R7, limit=5)
  → Returns: history of changes to R7
```

### Key Insight

Activity log provides cross-session awareness without activating records. Useful for orientation and understanding project state before deciding where to focus.

---

## Edge Cases

### Orphaned Records

If a parent is DISCARDED, what happens to its open children?
- **Option A**: Cascade discard (aggressive cleanup)
- **Option B**: Prompt user to re-parent or discard (safer)
- **Recommendation**: Option B—don't lose work silently

### Circular References

Records form a tree, not a graph. Parent-child is strict hierarchy. Related records can be linked without creating cycles.

### Very Deep Hierarchies

If user creates R → R1 → R2 → R3 → R4 → R5..., context loading becomes expensive.
- **Mitigation**: Warn at depth > 5, suggest flattening
- **Hard limit**: Depth 10, refuse deeper nesting

### Session Staleness Thresholds

Staleness is measured by tick gap, not wall-clock time:
- **Mild** (tick_gap 10-50): "Some activity occurred elsewhere"
- **Moderate** (tick_gap 50-200): "Significant work has happened since your last sync"
- **Severe** (tick_gap 200+): "This session is very out of date"

Sessions are never auto-closed (may have valuable unsaved thoughts in the chat). The system warns on resume via `sync_session()` response.
