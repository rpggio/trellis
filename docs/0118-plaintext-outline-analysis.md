# Plaintext Outline Format for Record Trees

*Analysis of token-efficient API representations for chat agent consumption*

## Executive Summary

This document analyzes how to reduce token cost when returning record references to chat agents. The existing API already separates full records (with bodies) from lightweight references (id, title, summary, state). This analysis proposes a **plaintext outline format** for returning those references, cutting token usage by ~50% compared to JSON.

---

## Record Content Model

Records are self-contained markdown documents of 1-7 paragraphs. They contain substantive reasoning, not simple statements.

### Example: A Question Record

**Title**: Cache invalidation approach

**Summary**: Leaning toward event-driven invalidation with TTL fallback. Need to resolve pub/sub infrastructure question.

**Body**:
```markdown
We need to invalidate cached API responses when underlying data changes. The current approach (fixed 5-minute TTL) causes unacceptable staleness for inventory data.

## Options Evaluated

**TTL-only**: Simple but can't meet the 30-second freshness requirement without destroying cache hit rates.

**Event-driven**: Database triggers publish invalidation events. Accurate but requires Redis pub/sub, which we don't currently run.

**Hybrid**: Short TTL (60s) plus event-driven early invalidation. Gracefully degrades if pub/sub fails.

## Current Thinking

The hybrid approach seems right. It gives us the accuracy of event-driven invalidation while failing safe to TTL behavior. Main blocker: we need to decide whether to add Redis pub/sub to the stack (see child question).

## Open Threads

- Child question on Redis pub/sub infrastructure is still open
- Haven't yet estimated implementation effort for the hybrid approach
```

### Example: A Conclusion Record

**Title**: Use hybrid cache invalidation

**Summary**: Decided on 60-second TTL with event-driven early invalidation. Redis pub/sub approved in R015.

**Body**:
```markdown
We're implementing hybrid cache invalidation: 60-second TTL as the baseline, with event-driven invalidation for early expiry when data changes.

## Decision Rationale

The hybrid approach won because:

1. **Meets freshness requirements**: Event-driven invalidation typically fires within 2 seconds of a write
2. **Fails gracefully**: If pub/sub goes down, we degrade to 60-second staleness rather than serving indefinitely stale data
3. **Infrastructure approved**: Redis pub/sub was approved as part of R015 (session caching decision)

## Implementation Notes

- Database triggers on `products`, `inventory`, and `pricing` tables
- Cache keys follow pattern `api:v1:{endpoint}:{params_hash}`
- Invalidation publishes to channel `cache:invalidate` with key pattern

## What This Supersedes

This replaces the fixed 5-minute TTL approach. The TTL-only option was rejected because it couldn't meet freshness requirements. Pure event-driven was rejected due to brittleness concerns.
```

---

## Current State: JSON References

The `RecordRef` structure:

```typescript
interface RecordRef {
  id: string;
  type: string;
  title: string;
  summary: string;
  state: WorkflowState;
  parent_id: string | null;
  children_count: number;
  open_children_count: number;
}
```

Example JSON for `list_records()`:

```json
{
  "results": [
    {
      "id": "R001",
      "type": "question",
      "title": "Cache invalidation approach",
      "summary": "Leaning toward event-driven invalidation with TTL fallback. Need to resolve pub/sub infrastructure question.",
      "state": "OPEN",
      "parent_id": "R000",
      "children_count": 2,
      "open_children_count": 1
    },
    {
      "id": "R002",
      "type": "question",
      "title": "Redis pub/sub for cache invalidation",
      "summary": "Should we add Redis pub/sub to support event-driven cache invalidation? Evaluating ops complexity vs benefits.",
      "state": "OPEN",
      "parent_id": "R001",
      "children_count": 0,
      "open_children_count": 0
    },
    {
      "id": "R003",
      "type": "conclusion",
      "title": "Use hybrid cache invalidation",
      "summary": "Decided on 60-second TTL with event-driven early invalidation. Redis pub/sub approved in R015.",
      "state": "RESOLVED",
      "parent_id": "R001",
      "children_count": 0,
      "open_children_count": 0
    }
  ]
}
```

**Token cost**: ~250 tokens

---

## Proposed: Plaintext Outline Format

### Format

```
[id] (state) Title
  Summary text

  [child-id] (state) Child Title
    Child summary
```

### Encoding Rules

1. **ID**: Bracketed at line start
2. **State**: Single letter in parens — `(O)` `(L)` `(R)` `(D)`
3. **Title**: Remainder of header line
4. **Summary**: Indented on following line(s)
5. **Hierarchy**: 2-space indentation per level
6. **Separator**: Blank line between siblings

### Same Data in Plaintext

```
[R001] (O) Cache invalidation approach
  Leaning toward event-driven invalidation with TTL fallback.
  Need to resolve pub/sub infrastructure question.

  [R002] (O) Redis pub/sub for cache invalidation
    Should we add Redis pub/sub to support event-driven cache
    invalidation? Evaluating ops complexity vs benefits.

  [R003] (R) Use hybrid cache invalidation
    Decided on 60-second TTL with event-driven early invalidation.
    Redis pub/sub approved in R015.
```

**Token cost**: ~110 tokens (~56% reduction)

---

## What the Agent Needs in References

For navigation and selection, the agent needs to identify records and decide which to activate:

| Attribute | Include | Notes |
|-----------|---------|-------|
| **id** | ✅ | Required for `activate(id)` |
| **title** | ✅ | Primary identification |
| **summary** | ✅ | Enables selection without loading body |
| **state** | ✅ | OPEN vs RESOLVED affects relevance |
| **parent_id** | ❌ | Implicit via indentation |
| **type** | ❌ | Inferrable; rarely affects decisions |
| **children_count** | ❌ | Visible in outline structure |
| **open_children_count** | ⚠️ | Useful as suffix: `(O+2)` |

---

## Which Commands Benefit

### High Impact — Use Plaintext

| Command | Notes |
|---------|-------|
| `list_records` | Primary listing; multiple refs |
| `get_project_overview` | Returns categorized ref lists |
| `search_records` | Returns matching refs + snippets |

### Medium Impact — Use Plaintext

| Command | Notes |
|---------|-------|
| `activate` → non-OPEN children | Refs only |
| `activate` → grandchildren | Refs only |

### Low Impact — Keep JSON

| Command | Notes |
|---------|-------|
| `activate` → target, parent, OPEN children | Full bodies needed anyway |
| `create_record` / `update_record` | Single record response |
| `sync_session` | Structured change objects |

---

## Token Savings Estimates

### Per-Reference Cost

| Format | Tokens | Overhead |
|--------|--------|----------|
| JSON RecordRef | ~70 | ~30 (keys, braces, quotes) |
| Plaintext | ~35 | ~8 (brackets, parens, newlines) |

### Scenario Estimates

| Scenario | Refs | JSON | Plaintext | Savings |
|----------|------|------|-----------|---------|
| Small project overview | 10 | ~700 | ~350 | 50% |
| Medium project overview | 25 | ~1750 | ~875 | 50% |
| Search results | 15 | ~1050 | ~525 | 50% |
| Context bundle children | 8 | ~560 | ~280 | 50% |

---

## Format Options

### Basic

```
[R001] (O) Cache invalidation approach
  Leaning toward event-driven invalidation with TTL fallback.
```

### With Open Children Count

```
[R001] (O+2) Cache invalidation approach
  Leaning toward event-driven invalidation with TTL fallback.
```

### Search Results with Snippet

```
[R001] (O) Cache invalidation approach
  Leaning toward event-driven invalidation with TTL fallback.
  > ...considering **Redis pub/sub** for event delivery...
```

### Minimal (Titles Only)

For maximum compression when summaries aren't needed:

```
[R001] (O) Cache invalidation approach
  [R002] (O) Redis pub/sub for cache invalidation
  [R003] (R) Use hybrid cache invalidation
```

---

## Additional Optimizations

### 1. Omit Summaries for Closed Records

RESOLVED and DISCARDED records are less likely to be activated:

```
[R003] (R) Use hybrid cache invalidation
[R004] (D) TTL-only approach — rejected
```

### 2. Section Headers for Overview

```
## OPEN (3)

[R001] (O+1) Cache invalidation approach
  Leaning toward event-driven with TTL fallback.

[R005] (O) API rate limiting
  Evaluating token bucket vs sliding window.

## LATER (1)

[R010] (L) Performance benchmarks
  Blocked on staging environment.

## RESOLVED (2)

[R003] (R) Use hybrid cache invalidation
[R015] (R) Redis infrastructure approved
```

### 3. Depth Truncation

For deep trees, collapse beyond depth 2:

```
[R001] (O) Cache invalidation approach
  ...

  [R002] (O) Redis pub/sub for cache invalidation
    ...
    [+2 children]
```

### 4. Response Envelope

```
session: sess_abc | tick: 147

[R001] (O) Cache invalidation approach
  ...
```

---

## Grammar

```
outline   := record*
record    := header summary? children?
header    := indent "[" id "]" " " state " " title "\n"
state     := "(" ("O"|"L"|"R"|"D") ("+" digits)? ")"
summary   := (indent+2 text "\n")+
children  := record+
indent    := ("  ")*
```

---

## Implementation

### Phase 1

Apply plaintext to listing commands:
- `list_records`
- `get_project_overview`
- `search_records`

### Phase 2

Apply to reference sections of `activate()`:
- `children.other`
- `grandchildren`

### Phase 3 (Optional)

Add format parameter for backward compatibility:

```typescript
list_records({ format?: "json" | "outline" })
```

---

## Risks

| Risk | Mitigation |
|------|------------|
| Title contains `]` or `(` | Accept as-is; unambiguous in context |
| Multi-line summary | Continue at same indent |
| Agent expects JSON | Document format; offer `format=json` fallback |

---

## Conclusion

The plaintext outline format reduces reference payload size by ~50% compared to JSON. The format is:

- **Human-readable**: No parsing required for LLM consumption
- **Parseable**: Simple regex for programmatic use
- **Hierarchical**: Indentation replaces explicit `parent_id`
- **Compact**: Single-letter states, minimal punctuation

Primary use: listing commands (`list_records`, `get_project_overview`, `search_records`) where multiple references are returned.
