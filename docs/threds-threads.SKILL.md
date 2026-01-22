---
name: threds-threads
description: Optional threading layer for threds MCP. Threads group conversational records (questions, explorations) that belong to a single reasoning flow. Use threads when you want session-contextual work separated from project-level artifacts.
---

# Threads Skill for Threds MCP

Threads are an optional organizational layer. A thread is a record with `type="thread"` that groups conversational work belonging to a single reasoning flow. Threads help separate session-contextual exploration from project-level artifacts that outlive any conversation.

## When to Use Threads

Use threads when:
- The user is exploring a topic that will generate multiple related records
- You want conversational artifacts (questions mid-exploration, working notes) grouped together
- The work is session-contextual rather than a standalone project artifact

Skip threads when:
- Creating project-level records (decisions, principles, discoveries)
- The user explicitly wants flat organization
- Single-record interactions that don't need grouping

## Thread Lifecycle

### Creating a Thread

Create a thread when first saving conversational content and no thread is active:

```
Agent: create_record(
  type="thread",
  title="<descriptive title>",
  body="<context: what prompted this exploration, current focus>"
)
```

Threads are always root records (no parent). They exist at the project level.

### Activating a Thread

At session start, query for open threads:

```
Agent: list_records(types=["thread"], states=["OPEN"])
```

Present open threads to user. When user chooses one:

```
Agent: activate(thread_id)
```

If the thread is active in another session, the system returns a conflict warning. Advise the user to close the other session or explicitly override.

### Working Within a Thread

Conversational records are children of the active thread:

```
Agent: create_record(
  parent=thread_id,
  type="question",
  title="...",
  body="..."
)
```

The thread provides context for its children. When activated, the thread's body and open children are loaded.

### Updating Thread Body

The thread body is a living document. Update it to reflect current state:

```
Agent: update_record(
  id=thread_id,
  body="<current focus, open questions, recent progress>"
)
```

Don't append session logs. Rewrite to reflect current understanding.

### Closing a Thread

When exploration is complete:

```
Agent: transition(thread_id, to_state=RESOLVED, reason="<summary of outcome>")
```

Child records persist. The thread becomes a historical container—visible in queries but not loaded as active context.

## Record Classification

### Conversational Records (require thread parent)

These record types are session-contextual and should be children of a thread:

- `question` — active exploration, options being weighed
- `working-note` — scratchpad, intermediate thinking
- `exploration` — open-ended investigation

If creating these types with no active thread, create a thread first.

### Project Records (can be roots)

These record types are project-level artifacts that outlive threads:

- `decision` — binding choice with rationale
- `principle` — standing rule or guideline
- `discovery` — factual finding that persists
- `conclusion` — resolved position (may also live under threads)

Project records can be roots or children of other project records. They don't require a thread.

## Session Start Workflow

1. Call `get_project_overview()` for orientation
2. Call `list_records(types=["thread"], states=["OPEN"])` to find open threads
3. Present open threads to user with summaries
4. If user wants to continue a thread: `activate(thread_id)`
5. If user starts new work: create thread when first conversational record is needed
6. If user wants project-level work only: skip threads, create root records

## Conflict Handling

Threads follow standard record conflict rules:

- `activate` warns if thread is active in another session
- User can override (their choice to accept risk)
- `update_record` may return conflict if modified elsewhere

Advise users to close stale sessions before starting new work on the same thread.

## Thread Body Template

```markdown
## Current Focus

[What we're currently exploring or deciding]

## Open Questions

[Active questions under this thread]

## Recent Progress

[Key developments, decisions made, paths eliminated]

## Next Steps

[What needs to happen to move forward]
```

Keep it concise. This is orientation material, not comprehensive documentation.

## Example Flow

```
User opens chat
  ↓
Agent: get_project_overview()
Agent: list_records(types=["thread"], states=["OPEN"])
  → Returns: [{ id: "r_abc", title: "API versioning exploration", state: OPEN }]
  ↓
Agent: "You have an open thread: 'API versioning exploration'. Continue, or start something new?"
  ↓
User: "Continue"
  ↓
Agent: activate("r_abc")
  → Returns thread with open children loaded
  ↓
[Conversation proceeds, new questions created as children of thread]
  ↓
User: "I think we've resolved this"
  ↓
Agent: transition("r_abc", to_state=RESOLVED, reason="Decided on URL-path versioning")
Agent: close_session()
```

## Relationship to Schemes

Threads work with any record scheme (question/conclusion, custom types, etc.). The scheme defines what record types exist; the threads skill defines how to organize session-contextual work.

A project might use:
- Threads skill for organization
- Question/conclusion scheme for record content
- Custom types for domain-specific artifacts
