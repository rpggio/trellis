# Plaintext Outline Format for Record Trees

*Analysis of token-efficient API representations for chat agent consumption*

## Executive Summary

This document analyzes how to reduce token cost when returning record trees to chat agents. The key insight is that agents primarily need just enough information to **identify** and **navigate** records—not full structural metadata. A plaintext outline format can cut token usage by 60-80% for listing/orientation commands while remaining highly parseable.

---

## Current State: JSON Record References

The current `RecordRef` structure (from the API spec):

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

Example JSON output for `list_records()` with 3 records:

```json
{
  "results": [
    {
      "id": "R001",
      "type": "question",
      "title": "Caching strategy",
      "summary": "How should we implement caching for the API layer?",
      "state": "OPEN",
      "parent_id": null,
      "children_count": 3,
      "open_children_count": 2
    },
    {
      "id": "R007",
      "type": "question",
      "title": "Redis vs Memcached",
      "summary": "Should we use Redis or Memcached for our caching needs?",
      "state": "OPEN",
      "parent_id": "R001",
      "children_count": 1,
      "open_children_count": 0
    },
    {
      "id": "R012",
      "type": "conclusion",
      "title": "Redis selected",
      "summary": "Redis chosen for persistence and data structure support.",
      "state": "RESOLVED",
      "parent_id": "R007",
      "children_count": 0,
      "open_children_count": 0
    }
  ]
}
```

**Token cost**: ~180 tokens for this small example.

---

## Proposed: Plaintext Outline Format

### Core Format

```
[id] (state) Title
  Summary text

  [child-id] (state) Child Title
    Child summary
```

### Encoding Rules

1. **ID**: Bracketed at line start: `[R001]`
2. **State**: Parenthesized after ID: `(OPEN)`, `(LATER)`, `(RESOLVED)`, `(DISCARDED)`
3. **Title**: Remainder of the ID line
4. **Summary**: Indented on the following line(s)
5. **Hierarchy**: Indentation (2 spaces per level)
6. **Type**: Omitted by default (see "What the Agent Needs" below)

### Same Data in Plaintext

```
[R001] (OPEN) Caching strategy
  How should we implement caching for the API layer?

  [R007] (OPEN) Redis vs Memcached
    Should we use Redis or Memcached for our caching needs?

    [R012] (RESOLVED) Redis selected
      Redis chosen for persistence and data structure support.
```

**Token cost**: ~65 tokens (~64% reduction)

---

## Which API Commands Benefit?

### High Impact (Use Plaintext Outline)

| Command | Current Output | Benefit |
|---------|---------------|---------|
| `get_project_overview` | Arrays of RecordRef | Returns root_records, open_records, later_records—all ideal for outline |
| `list_records` | Array of RecordRef | Primary listing command |
| `search_records` | Array of RecordRef + snippets | Can embed snippet in summary line |

### Medium Impact (Use Simplified JSON or Plaintext)

| Command | Notes |
|---------|-------|
| `activate` → `ContextBundle.children.other` | Non-OPEN children returned as refs |
| `activate` → `ContextBundle.grandchildren` | All grandchildren as refs |

### Low Impact (Keep JSON)

| Command | Why |
|---------|-----|
| `activate` → target, parent, open children | Full content needed—no shortcut |
| `create_record` / `update_record` responses | Returns single record, JSON overhead minimal |
| `sync_session` | Change objects need structured diffs |
| `get_record_diff` | Diff format is inherently structured |

---

## What Record Attributes Does the Agent Actually Need?

### For Navigation/Selection (Listing)

| Attribute | Needed? | Rationale |
|-----------|---------|-----------|
| **id** | ✅ Essential | Required to activate/reference |
| **title** | ✅ Essential | Primary identification |
| **summary** | ✅ Essential | Context for selection |
| **state** | ✅ Essential | Determines relevance (OPEN vs RESOLVED) |
| **parent_id** | ⚠️ Implicit | Hierarchy shown via indentation |
| **type** | ❌ Optional | Rarely affects agent decision-making |
| **children_count** | ❌ Optional | Visible from outline structure |
| **open_children_count** | ⚠️ Useful | Could add as suffix: `(OPEN, 2 open)` |

### For Reasoning (Full Record)

When actually working with a record, the agent needs the full `Record` object including `body`. This is already handled by `activate()` returning full content for target + parent + OPEN children.

### Recommendation: Minimal Reference Format

For listings, include only:
- **id**: Always
- **state**: Always
- **title**: Always
- **summary**: Always
- **open_children_count**: Only if > 0, as suffix

Omit:
- **type**: Agent can infer from context or title
- **parent_id**: Shown by indentation
- **children_count**: Total count rarely useful

---

## Token Savings Estimates

### Assumptions

- Average title: 5 tokens
- Average summary: 15 tokens
- JSON structural overhead per record: ~25 tokens (keys, braces, quotes, colons)
- Plaintext overhead per record: ~5 tokens (brackets, parens, newlines)

### Per-Record Savings

| Format | Tokens/Record | Overhead |
|--------|--------------|----------|
| Full JSON RecordRef | ~50 tokens | ~25 tokens |
| Plaintext outline | ~25 tokens | ~5 tokens |

**Savings**: ~50% per record

### Realistic Scenarios

| Scenario | Records | JSON Tokens | Plaintext Tokens | Savings |
|----------|---------|-------------|------------------|---------|
| Project overview (10 open, 5 later, 3 root) | 18 refs | ~900 | ~450 | 50% |
| Large search result | 20 refs | ~1000 | ~500 | 50% |
| Deep tree listing (3 levels) | 30 refs | ~1500 | ~600 | 60% |
| Context bundle (grandchildren) | 15 refs | ~750 | ~375 | 50% |

### Compound Savings

A typical session might call `get_project_overview` + `list_records` + `search_records` before activating. With 50+ refs returned across these calls, plaintext could save 500-1000 tokens per orientation phase.

---

## Extended Format Options

### Option A: Minimal (Recommended)

```
[R001] (OPEN) Caching strategy
  How should we implement caching for the API layer?
```

### Option B: With Open Children Count

```
[R001] (OPEN+2) Caching strategy
  How should we implement caching for the API layer?
```

The `+2` suffix indicates 2 open children—useful for deciding where to dive deeper.

### Option C: With Type

```
[R001] question (OPEN) Caching strategy
  How should we implement caching for the API layer?
```

Only if type is genuinely useful. In practice, titles like "Caching strategy" (a question) vs "Redis selected" (a conclusion) are self-evident.

### Option D: Search Results with Snippets

```
[R007] (OPEN) Redis vs Memcached
  Should we use Redis or Memcached for caching?
  > ...considering **Redis** for its persistence features...
```

The `>` prefix marks the matching snippet, with highlights in bold.

---

## Additional Token-Saving Techniques

### 1. State Abbreviations

```
(O) = OPEN
(L) = LATER
(R) = RESOLVED
(D) = DISCARDED
```

Savings: ~3 tokens per record

Plaintext becomes:
```
[R001] (O) Caching strategy
  How should we implement caching?
```

### 2. ID Shortening

If IDs are always `Rn` format, drop the prefix in listings:

```
[1] (O) Caching strategy
```

Or use implicit numbering within the response (risky if agent needs to reference later).

### 3. Summary Truncation

Limit summaries to ~50 characters in listings. Full summary available on activation.

```
[R001] (O) Caching strategy
  How should we implement caching for the...
```

### 4. Deferred Loading Annotations

Mark records that have more detail available:

```
[R001] (O) Caching strategy [+body]
  How should we implement caching?
```

The `[+body]` signals that activating will load substantial content.

### 5. Flat vs Tree Responses

For `get_project_overview`, the current spec returns three separate arrays (`root_records`, `open_records`, `later_records`). These could be combined into a single annotated outline:

```
# OPEN
[R001] (O) Caching strategy
  How should we implement caching?

  [R007] (O) Redis vs Memcached
    Should we use Redis or Memcached?

[R022] (O) API versioning
  How should we version the API?

# LATER
[R015] (L) Scale requirements
  Blocked on traffic projections.
```

### 6. Contextual Omission

In `ContextBundle`, the agent doesn't need summaries for records it already activated. The API could return:

```
children:
  [R012] (R) Redis selected
  [R013] (O) Cache invalidation strategy
    When and how should we invalidate cache entries?
```

Only OPEN children (which the agent will reason about) get summaries.

### 7. Response Envelope Minimization

Current JSON responses include metadata:

```json
{
  "success": true,
  "session_id": "sess_abc123",
  "results": [...]
}
```

For listing commands, the envelope could be minimal or omitted:

```
session: sess_abc123

[R001] (O) Caching strategy
...
```

### 8. Tick/Timestamp Omission in Listings

The agent rarely needs `modified` timestamps in listings—only when reviewing history. Omit from standard responses.

---

## Implementation Recommendations

### Phase 1: Core Plaintext Format

Implement plaintext outline for:
- `list_records`
- `get_project_overview` (for record arrays)
- `search_records`

Use state abbreviations `(O/L/R/D)` and minimal format.

### Phase 2: Context Bundle Optimization

Apply plaintext to:
- `ContextBundle.children.other` (non-OPEN children)
- `ContextBundle.grandchildren`

Keep full JSON for target, parent, and OPEN children.

### Phase 3: Response Format Negotiation

Add optional parameter to control verbosity:

```typescript
list_records(params: {
  format?: "full" | "compact" | "minimal"  // default: compact
})
```

- `full`: Current JSON
- `compact`: Plaintext outline with summaries
- `minimal`: Plaintext with IDs, states, titles only (no summaries)

---

## Parsing Considerations

The plaintext format is trivially parseable:

```javascript
// Regex patterns
const RECORD_LINE = /^\s*\[(\w+)\]\s*\(([OLRD])\)\s*(.+)$/;
const SUMMARY_LINE = /^\s{2,}([^[\n].*)$/;

// Parse yields:
// { id: "R001", state: "O", title: "Caching strategy", indent: 0 }
// { summary: "How should we implement caching?" }
```

For LLM consumption, parsing isn't even necessary—the format is human-readable and can be processed as natural language context.

---

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Format ambiguity | Strict grammar; no user content in structural positions |
| Title containing `]` or `(` | Titles are user content—escape or accept as-is |
| Summary containing newlines | Indent continuation lines; or truncate to single line |
| State abbreviation confusion | Document clearly; O/L/R/D are unambiguous |
| Loss of information | Keep full JSON as opt-in fallback |

---

## Conclusion

A plaintext outline format for record trees can reduce token costs by 50-60% for listing and orientation commands. The key principles are:

1. **Hierarchy through indentation** replaces explicit `parent_id`
2. **State abbreviations** (`O/L/R/D`) minimize per-record overhead
3. **Omit rarely-used fields** (`type`, `children_count`) from listings
4. **Keep JSON for mutations and full content** where structure matters

The format is simple enough that agents can parse it programmatically or consume it as natural language context—both work.

---

## Appendix: Full Plaintext Grammar

```
outline      := record*
record       := header summary? children?
header       := indent "[" id "]" state title "\n"
summary      := indent{+2} text "\n"
children     := record+
indent       := ("  ")*
id           := [A-Za-z0-9]+
state        := "(" ("O"|"L"|"R"|"D") ("+" digit+)? ")"
title        := [^\n]+
text         := [^\n]+
```

Example matching the grammar:

```
[R001] (O+2) Caching strategy
  How should we implement caching for the API layer?

  [R007] (O) Redis vs Memcached
    Should we use Redis or Memcached?

    [R012] (R) Redis selected
      Chosen for persistence and data structure support.

  [R013] (O) Cache invalidation
    When and how should we invalidate cache entries?
```
