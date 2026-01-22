# Scenario Analysis: Threds MCP

*Modeling information flow from chat to record system*

## Design Context

This document models how information moves from ephemeral chat conversation into persistent records. The system maintains design reasoning across sessions using generic **records** organized in parent-child hierarchies within **projects**.

### Core Concepts

- **Project**: Container for records with a logical clock (tick)
- **Record**: Persistent unit with ID, title, summary, body, type, and workflow state
- **Session**: Tracks which records a chat has activated
- **Activation**: Loading a record into session context
- **Workflow states**: OPEN | LATER | RESOLVED | DISCARDED

### Key Principle: Save on Request

The agent only persists information when the user requests it. Rationale:

- Only the user knows their confidence level at any moment
- Positions may shift during conversation; early saves create noise
- Retroactive modeling captures the user's final conviction
- Simple concept, known to produce good output

### Notation

- `T1, T2, T3...` — Thoughts (information in chat, not yet persisted)
- `R1, R2, R3...` — Records (persisted)
- `S1, S2...` — Sessions
- `→` — Information flow direction

---

## Scenario 1: Cold Start — Orientation

**Situation**: User opens new chat with existing project.

### Flow

```
User: "What's the state of the project?"
  ↓
Agent: get_project_overview()
  → Returns: project tick, open sessions, root record refs
  ↓
Agent: [summarizes project state to user]
  ↓
User: "Show me the open questions"
  ↓
Agent: list_records(states=["OPEN"], types=["question"])
  → Returns: record references
  ↓
Agent: [presents list to user]
```

### Information State

| Location | Content |
|----------|---------|
| Chat | Project summary, list of open questions |
| Records | Unchanged |

No activation yet—browsing doesn't require it.

---

## Scenario 2: Resuming Work

**Situation**: User wants to continue previous work.

### Flow

```
User: "Let's continue the caching discussion"
  ↓
Agent: search_records("caching")
  → Returns: [R7: "Caching strategy", state=OPEN]
  ↓
Agent: activate(R7)
  → Returns: R7 (full), parent P (full), open children (full), others as refs
  ↓
Agent: [presents R7 context to user]
  ↓
Session S1 now tracks: active_records = [R7]
```

### Information State

| Location | Content |
|----------|---------|
| Chat | R7 body, parent context, child summaries |
| Records | Unchanged |
| Session | R7 activated |

---

## Scenario 3: Conversation Without Saving

**Situation**: User explores options, changes position, reaches conclusion. No save requested yet.

### Flow

```
[R7 "Caching strategy" activated]
  ↓
User: "Should we use Redis or Memcached?"
  ↓
Agent: [discusses trade-offs] → T1 (Redis pros/cons)
  ↓
User: "Redis seems better for persistence"
  ↓
Agent: [elaborates] → T2 (Redis persistence benefits)
  ↓
User: "Actually, we don't need persistence. Memcached is simpler."
  ↓
Agent: [adjusts] → T3 (Memcached simplicity argument)
  ↓
User: "Let's go with Memcached"
  ↓
Agent: [confirms] → T4 (Memcached decision)
```

### Information State

| Location | Content |
|----------|---------|
| Chat | T1, T2, T3, T4 (full exploration, position changes) |
| Records | Unchanged |

Key: T1 and T2 (Redis direction) are superseded by T3 and T4. If we had saved eagerly, we'd have noise. Waiting for user request means we capture the final position.

---

## Scenario 4: Save on Request

**Situation**: User asks to save the session's conclusions.

### Flow

```
[Continuing from Scenario 3]
  ↓
User: "Save this"
  ↓
Agent: [models conversation retroactively]
  - T4 represents the decision (Memcached)
  - T3 is the rationale (simplicity, no persistence needed)
  - T1, T2 are rejected alternatives (Redis path)
  ↓
Agent: create_record(
  parent=R7,
  type="conclusion",
  title="Memcached selected",
  body="[synthesized from T3, T4; notes T1, T2 as rejected path]"
)
  → Returns: R15
  ↓
Agent: transition(R7, to_state=RESOLVED, resolved_by=R15)
  ↓
Agent: save_session()
```

### Information Flow

```
T3, T4 (final position) → R15 (conclusion)
T1, T2 (rejected) → mentioned in R15 as alternatives considered
```

### Information State

| Location | Content |
|----------|---------|
| Chat | T1-T4, confirmation of save |
| Records | R15 created, R7 resolved |

---

## Scenario 5: Partial Save

**Situation**: User wants to save progress but work isn't complete.

### Flow

```
[Working on R7, discussed T1-T5, no clear conclusion yet]
  ↓
User: "Save where we are"
  ↓
Agent: [models current state]
  - Open question: Redis vs Memcached (T1-T3)
  - Leaning toward: Redis for data structures (T4)
  - Blocker: need to confirm persistence requirements (T5)
  ↓
Agent: update_record(R7, body="[updated with T1-T5 synthesis]")
  ↓
Agent: create_record(
  parent=R7,
  type="question",
  title="Redis vs Memcached",
  body="[T1-T3 options, T4 current leaning, T5 blocker]"
)
  → Returns: R15
  ↓
Agent: save_session()
```

### Information Flow

```
T1-T5 (exploration state) → R7 updated, R15 created
```

R7 remains OPEN. R15 captures the specific question for future sessions.

---

## Scenario 6: Multiple Save Points

**Situation**: Long conversation with multiple logical save points.

### Flow

```
[Session on R7 "API Design"]
  ↓
[Discussion about versioning: T1-T4]
  ↓
User: "Save the versioning decision"
  ↓
Agent: create_record(parent=R7, type="conclusion", title="URL path versioning", body="[T1-T4]")
  → R15
  ↓
[Discussion continues about authentication: T5-T8]
  ↓
User: "Save this too"
  ↓
Agent: create_record(parent=R7, type="question", title="Auth approach", body="[T5-T8]")
  → R16
  ↓
[More discussion: T9-T12]
  ↓
User: "Let's wrap up"
  ↓
Agent: update_record(R7, body="[current state summary]")
Agent: save_session()
Agent: close_session()
```

### Information Flow

```
T1-T4 → R15 (versioning conclusion)
T5-T8 → R16 (auth question)
T9-T12 + overall state → R7 update
```

---

## Scenario 7: Returning to Stale Session

**Situation**: User returns to chat after time away. Other work happened.

### Flow

```
User: "Where were we?"
  ↓
Agent: sync_session()
  → Returns: { tick_gap: 23, status: "stale" }
  ↓
Agent: "23 writes have occurred since we last synced. Want me to check what changed?"
  ↓
User: "Yes"
  ↓
Agent: get_recent_activity(limit=10)
  → Returns: activity entries
  ↓
Agent: [summarizes relevant changes]
  ↓
User: "OK, let's continue"
  ↓
[Work proceeds]
```

### Information State

Session baseline refreshed. Agent has informed user of external changes.

---

## Scenario 8: Conflict on Activation

**Situation**: Record is active in another session.

### Flow

```
User: "Let's work on the caching strategy"
  ↓
Agent: activate(R7)
  → Returns: { warning: "R7 active in S1", conflict: true, record: R7 }
  ↓
Agent: "This record is active in another session. Work on it anyway, or choose something else?"
  ↓
User: "Go ahead"
  ↓
[Work proceeds with conflict awareness]
```

If user later saves and S1 also saved:

```
Agent: update_record(R7, body="...")
  → Returns: { conflict: true, other_version: "..." }
  ↓
Agent: "R7 was modified in another session. Here's their version: [...].
        Keep yours, keep theirs, or merge?"
  ↓
User: "Merge these points..."
  ↓
Agent: update_record(R7, body="[merged]", force=true)
```

---

## Scenario 9: Creating New Work

**Situation**: User starts fresh topic.

### Flow

```
User: "I want to think through our deployment strategy"
  ↓
Agent: search_records("deployment")
  → Returns: [] (nothing exists)
  ↓
Agent: "No existing records on deployment. Want me to create a starting point when we're ready to save?"
  ↓
User: "Sure. Let's discuss..."
  ↓
[Discussion: T1-T6]
  ↓
User: "Save this"
  ↓
Agent: create_record(
  parent=null,
  type="question",
  title="Deployment strategy",
  body="[T1-T6 synthesis]"
)
  → R20 (root record)
```

### Information Flow

```
T1-T6 → R20 (new root)
```

---

## Scenario 10: Discarding Work

**Situation**: Exploration was a dead end.

### Flow

```
[Working on R15 "Redis evaluation"]
  ↓
[Discussion T1-T4 reveals Redis won't work]
  ↓
User: "This whole direction is wrong. Let's abandon it."
  ↓
Agent: transition(R15, to_state=DISCARDED, reason="[T4 summary: why it won't work]")
  ↓
Agent: "R15 has child records. Discard them too?"
  ↓
User: "Yes"
  ↓
Agent: transition(R16, to_state=DISCARDED, reason="Parent discarded")
```

### Information Flow

```
T4 (reason for abandonment) → R15 transition reason
```

DISCARDED records persist for history but won't load as active context.

---

## Scenario 11: Deferring Work

**Situation**: Can't proceed without external input.

### Flow

```
[Working on R15 "Cache sizing"]
  ↓
User: "We can't decide this until we have production metrics"
  ↓
Agent: transition(R15, to_state=LATER, reason="Blocked on production metrics")
  ↓
Agent: update_record(R15, body="[current state + what's needed to unblock]")
  ↓
Agent: save_session()
```

LATER records surface in `list_records(states=["LATER"])` for future review.

---

## Scenario 12: Using Threads (Opt-in)

**Situation**: User wants conversational work grouped.

### Flow

```
User: "Start a thread for API exploration"
  ↓
Agent: create_record(type="thread", title="API exploration", body="[initial context]")
  → R30 (thread)
  ↓
[Discussion: T1-T5]
  ↓
User: "Save the versioning question"
  ↓
Agent: create_record(parent=R30, type="question", title="API versioning", body="[T1-T5]")
  → R31 (under thread)
  ↓
[More discussion: T6-T10]
  ↓
User: "Save this conclusion"
  ↓
Agent: create_record(parent=R30, type="conclusion", title="REST over GraphQL", body="[T6-T10]")
  → R32 (under thread)
  ↓
User: "Close the thread"
  ↓
Agent: transition(R30, to_state=RESOLVED, reason="API decisions complete")
```

### Information Flow

```
T1-T5 → R31 (question under thread)
T6-T10 → R32 (conclusion under thread)
Thread R30 groups the conversational artifacts
```

---

## Command Summary

### Orientation (no activation)

| Command | Purpose |
|---------|---------|
| `get_project_overview()` | Project state, sessions, roots |
| `search_records(query)` | Find by content |
| `list_records(filters)` | Browse by state/type |
| `get_record_ref(id)` | Lightweight lookup |
| `get_recent_activity()` | Cross-session awareness |

### Reasoning (requires activation)

| Command | Purpose |
|---------|---------|
| `activate(id)` | Load context |
| `create_record(...)` | Create record |
| `update_record(id, ...)` | Modify content |
| `transition(id, state)` | Change workflow state |

### Session Lifecycle

| Command | Purpose |
|---------|---------|
| `sync_session()` | Check staleness |
| `save_session()` | Checkpoint |
| `close_session()` | End session |

---

## Information Flow Principles

1. **Chat is ephemeral**: T1, T2, T3... exist only in conversation until saved
2. **Records are durable**: R1, R2, R3... persist across sessions
3. **Save on request**: Agent waits for user to indicate readiness
4. **Retroactive modeling**: Final position captured, not every position change
5. **Synthesis over transcript**: Records distill conversation, don't reproduce it
6. **Self-explaining records**: Each record comprehensible without originating chat
