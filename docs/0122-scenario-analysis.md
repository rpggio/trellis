# Scenario Analysis: Threds MCP

*Modeling chat workflows for record synchronization*

## Design Context

This document models the most significant chat scenarios for a system that maintains design reasoning across chat sessions. The system uses generic **records** organized in parent-child hierarchies within **projects**.

### Core Concepts

- **Project**: Container for related work, with a logical clock (tick) for tracking activity
- **Record**: Self-explaining unit with ID, title, summary, body, type, and workflow state
- **Record reference**: Lightweight pointer (ID, title, summary) without body
- **Session**: A chat's connection to the record system, tracking active records
- **Activation**: Loading a record into session context for reasoning
- **Workflow states**: OPEN | LATER | RESOLVED | DISCARDED
- **Tick**: Monotonic counter incremented on writes; the project's logical clock

### Key Constraints

1. Every session belongs to a project (default project when not specified)
2. Reasoning on a record requires parent context loaded
3. One chat at a time should reason against any given record (warning on conflict)
4. User can override default behavior where reasonable
5. Open sessions are assumed to contain unsaved information
6. Staleness is measured by tick gap, not wall-clock time

---

## Variable Notation

- `R1, R2, R3...` — Records
- `P` — Parent record
- `C1, C2...` — Child records
- `S1, S2...` — Sessions

---

## Scenario 1: Cold Start — New Chat, Existing Project

**Situation**: User opens new chat. Project has existing records.

### Flow

```
Chat opens
  ↓
Agent: get_project_overview()
  → Returns: project tick, open sessions, root record refs
  ↓
User: "Let's continue the caching design"
  ↓
Agent: search_records("caching")
  → Returns: [R7: "Caching strategy", state=OPEN]
  ↓
Agent: activate(R7)
  → Returns: {
      record: R7 (full),
      parent: P (full),
      children: {
        R12: ref (RESOLVED),
        R13: full (OPEN)
      }
    }
  ↓
Session S1 tracks: active_records = [R7, R13]
```

### Context Loading Rules

| Relationship | State | Loaded |
|--------------|-------|--------|
| Parent | any | Full content |
| Target | any | Full content |
| Child | OPEN | Full content |
| Child | LATER/RESOLVED/DISCARDED | Reference only |
| Grandchild | any | Reference only |

---

## Scenario 2: Creating Child Records

**Situation**: While working on R7, a subtopic needs exploration.

### Flow

```
Working on R7 "Caching strategy"
  ↓
Discussion: "Redis vs Memcached needs analysis"
  ↓
Agent: create_record(
  parent=R7,
  type="question",
  title="Redis vs Memcached",
  body="..."
)
  → Returns: R15 (new, state=OPEN, auto-activated)
  ↓
S1 now tracks: active_records = [R7, R13, R15]
```

### Key Point

New records are automatically activated in the creating session. Parent context (R7) is already loaded.

---

## Scenario 3: Resolving Work

**Situation**: Question R15 is answered.

### Flow

```
Working on R15 "Redis vs Memcached"
  ↓
Decision reached: "Redis for persistence support"
  ↓
Agent: create_record(
  parent=R15,
  type="conclusion",
  title="Redis selected",
  body="..."
)
  → Returns: R16
  ↓
Agent: transition(R15, to_state=RESOLVED, resolved_by=R16)
  → R15.state = RESOLVED
  ↓
Agent: save_session()
```

---

## Scenario 4: Returning to Stale Session

**Situation**: User had S1 open, closed browser. Later reopens. Other sessions made writes.

### Flow

```
Chat reopens (S1 still "open" in system)
  ↓
Agent: sync_session()
  → Returns: {
      project_tick: 147,
      session_tick: 98,
      tick_gap: 49,
      warning: "49 writes since last sync",
      changes: [
        { record: R13, type: "modified", at_tick: 112 },
        { record: R18, type: "created", at_tick: 130 }
      ]
    }
  ↓
Agent: "49 writes occurred. Some active records modified. Review changes?"
```

### Staleness Thresholds

- **Mild** (10-50 tick gap): "Some activity elsewhere"
- **Moderate** (50-200): "Significant work happened"
- **Severe** (200+): "Very out of date"

Sessions are never auto-closed (may have unsaved thoughts).

---

## Scenario 5: Parallel Sessions — Conflict Warning

**Situation**: S1 has R7 active. User opens S2, tries to work on R7.

### Flow

```
S2 opens
  ↓
Agent: activate(R7)
  → Returns: {
      warning: "R7 active in session S1 (10 min ago)",
      conflict: true,
      record: R7 (full)
    }
  ↓
Agent: "This record is active elsewhere. Close other session, or continue anyway?"
  ↓
User: "Continue"
  ↓
[Work proceeds; update_record may return merge conflicts]
```

### Conflict Resolution

- Warn on activation, require acknowledgment
- Present merge conflicts on save if concurrent edits
- User can force update with `force=true`

---

## Scenario 6: Review Mode — Surveying Open Work

**Situation**: User wants overview before deciding what to work on.

### Flow

```
New chat
  ↓
User: "What's open?"
  ↓
Agent: get_project_overview()
  → Returns: {
      open_sessions: [S1],
      root_records: [R1, R7, R22]
    }
  ↓
Agent: list_records(states=["OPEN"])
  → Returns all open records
  ↓
Agent presents summary (no activation yet)
  ↓
User: "Let's work on the caching stuff"
  ↓
Agent: activate(R7)
```

### Key Point

Browsing (search, list, overview) doesn't require activation. Activation is for reasoning—when creating, modifying, or transitioning records.

---

## Scenario 7: Navigating Hierarchy

**Situation**: While on R15, user wants to revisit grandparent.

### Flow

```
R15 active, R7 loaded as parent
  ↓
User: "Let's look at the project scope"
  ↓
Agent: activate(P)
  → Returns P with its children as refs
  ↓
S1 now tracks: active_records = [R15, P]
```

A session can have multiple records active for comparison or navigation.

---

## Scenario 8: Creating Root Records

**Situation**: Starting new line of reasoning.

### Flow

```
User: "New topic: authentication approaches"
  ↓
Agent: create_record(
  parent=null,
  type="question",
  title="Authentication approach",
  body="..."
)
  → Returns: R23 (root, auto-activated)
```

Root records have no parent. They're entry points for distinct topics.

---

## Scenario 9: Discarding Work

**Situation**: Line of reasoning was a dead end.

### Flow

```
Working on R15
  ↓
User: "This was misguided, abandon it"
  ↓
Agent: transition(R15, to_state=DISCARDED, reason="Approach invalid")
  ↓
Agent: "R15 has 2 children. Discard them too?"
  ↓
User: "Yes"
  ↓
Agent: transition(R16, to_state=DISCARDED, reason="Parent discarded")
```

DISCARDED records persist for history but don't load as full context.

---

## Scenario 10: Deferring Work

**Situation**: Question valid but not actionable now.

### Flow

```
Working on R15
  ↓
User: "Can't decide until we know scale requirements"
  ↓
Agent: transition(R15, to_state=LATER, reason="Blocked on scale analysis")
  ↓
[Future session]
  ↓
Agent: list_records(states=["LATER"])
  → Returns deferred records
```

LATER indicates intentional deferral. Record body should capture why.

---

## Scenario 11: Activity Review

**Situation**: Understanding recent project activity.

### Flow

```
User: "What happened recently?"
  ↓
Agent: get_recent_activity(limit=20)
  → Returns activity entries
  ↓
Agent summarizes without activating records
  ↓
User: "What changed in R7?"
  ↓
Agent: get_record_history(R7)
  → Returns change history
```

Activity log provides cross-session awareness for orientation.

---

## Command Summary

### Orientation (no activation needed)

| Command | Purpose |
|---------|---------|
| `get_project_overview()` | Project state, sessions, roots |
| `search_records(query)` | Find by content |
| `list_records(filters)` | List by state/type |
| `get_record_ref(id)` | Reference without activation |
| `get_recent_activity()` | Cross-session activity |

### Reasoning (requires activation)

| Command | Purpose |
|---------|---------|
| `activate(id)` | Load context for reasoning |
| `create_record(...)` | Create record |
| `update_record(id, ...)` | Modify content |
| `transition(id, state)` | Change workflow state |

### Session Lifecycle

| Command | Purpose |
|---------|---------|
| `sync_session()` | Pull changes, check staleness |
| `save_session()` | Checkpoint |
| `close_session()` | End session |

---

## Edge Cases

### Orphaned Records

If parent is DISCARDED, prompt user about children. Don't cascade silently.

### Deep Hierarchies

Warn at depth > 5. Hard limit at 10.

### Circular References

Records form a tree. Use `related` field for cross-references without cycles.
