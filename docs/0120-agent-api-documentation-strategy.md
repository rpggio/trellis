# Strategy: Agent-Facing Documentation for `threds-mcp`

This doc explores options for documenting the `threds-mcp` MCP server for LLM agents in a way that:

- minimizes context bloat (keeps the agent’s working set small)
- minimizes round trips (avoids “doc fetch” + “do work” ping-pong)
- still gives the agent the right info in each situation:
  - project knowledge (dynamic state)
  - record modeling mental model and principles (static model)
  - tool workflows and constraints (procedural rules)

It builds on:
- `docs/0122-scenario-analysis.md` (what agents *need to do*)
- `docs/0119-patterns-in-mcp-guidance.md` (how mature MCP servers guide agents)

---

## Background: What the agent must internalize (without loading the whole spec)

The core “working model” from the scenario analysis is:

- **Projects** are containers with a monotonic **tick** (logical clock). Staleness is **tick gap**, not wall-clock time.
- **Records** are self-explaining units of reasoning in a **tree** (parent/children). State drives how much content gets loaded.
- **Sessions** track what records are active and provide the coordination surface for multi-chat work.
- **Activation** is the boundary between *browsing* and *reasoning/mutating*: it loads the minimum required context bundle.
- **Single-writer intent**: concurrent edits aren’t forbidden, but must be surfaced (warnings/conflicts) and handled explicitly.

The documentation strategy should make this model “stick” with minimal tokens, and then let the agent pull deeper detail only when needed.

---

## The three documentation payload types (and where they should live)

### 1) Project knowledge (dynamic, per-tenant/per-project)

This is runtime state, not documentation. The agent should retrieve it via tools designed for orientation and review, and avoid pulling full record bodies unless it’s about to reason or mutate.

**Doc needs** (what the agent must know):
- how to bootstrap into the right project quickly
- how to discover “what’s open” without activating half the tree
- how to detect staleness and cross-session activity cheaply

**Mechanism options**
- **A. Orientation tools as the primary surface (preferred):**
  - `get_project_overview` as the single “cold start” call
  - `search_records` / `list_records` for browsing
  - `get_recent_activity` for “what changed” without activation
- **B. Read-only resources for project state (optional):**
  - e.g. `resource://projects/{id}/overview` to standardize “overview” across clients
  - use when clients prefer resource fetch over tool calls, but beware: it’s still a round trip
- **C. A single “brief” tool to reduce multi-call bootstraps (optional):**
  - if agents commonly do `list_projects` → `get_project_overview` → `search_records`,
    consider a combined `get_bootstrap_context` with flags to keep payload small

**Round-trip heuristic**
- Optimize for **one call** to identify “where we are” (`get_project_overview`) and **one call** to load working context (`activate`) after the user indicates a target.

---

### 2) Record modeling mental model & principles (static, cross-tool)

This is the minimum shared vocabulary and invariant rules that prevent shallow tool usage.

**Doc needs** (must be easily accessible):
- canonical glossary (project/record/record ref/session/activation/tick/states)
- the “activation boundary” and why it exists
- how state affects context loading and therefore token budget
- guidance on writing records so they remain self-explaining and cheap to load

**Mechanism options**
- **A. Server Instructions as a token-budgeted “rules of engagement” (preferred):**
  - small, stable, always-present
  - link out to deeper resources by URI (do not inline the whole spec)
- **B. A `concepts.md` documentation resource (recommended):**
  - a deeper reference the agent can fetch when uncertain or when building new workflows
- **C. “Record writing guide” resource (recommended):**
  - keeps record bodies consistent and compressible (structure + length guidance)

**Key principle to document explicitly**
- Record bodies must be **self-explaining** and structured so an agent can reason from the record alone, without rereading chat transcripts.

---

### 3) Tool workflows and constraints (procedural, cross-tool + tool-specific)

This is the “when/why/how” guidance that prevents unnecessary calls and prevents incorrect mutation patterns.

**Doc needs**
- phase separation: orientation vs activation vs mutation vs lifecycle
- staleness + sync expectations (tick-gap semantics)
- concurrency and conflict handling (warnings vs hard conflicts; `force` semantics)
- session header constraints (when `session_id` is required vs inferred)

**Mechanism options**
- **A. Tight tool descriptions + schema constraints (baseline):**
  - keep descriptions short but include side effects and expected follow-ups
  - enforce “deterministic” inputs with enums (e.g., record state), limits, and required fields
- **B. Workflow playbooks as resources (recommended):**
  - scenario-focused docs the agent loads only when needed:
    - cold start / resume stale chat
    - “continue work on X”
    - branch into a new thread
    - resolve/close out work
    - conflict resolution + merge protocol
- **C. A `help(topic)` tool (optional):**
  - reduces round trips versus resource discovery + fetch
  - returns a short, topic-scoped snippet (e.g., “conflicts”, “activation”, “staleness”)

---

## Documentation architecture options (trade-offs)

### Option 1: “Fat Server Instructions”

Put most of the mental model + workflows into `initialize.instructions`.

**Pros**
- zero extra doc round trips
- consistent across clients

**Cons**
- high risk of context bloat (instructions crowd out user intent and record context)
- harder to evolve; agents may overfit to stale instructions

When it works: very small APIs or when tool semantics are trivial.

---

### Option 2: “Layered + Progressive Disclosure” (recommended)

Use a small always-on manual plus modular on-demand resources.

**Always-on (keep small)**
- Server Instructions: glossary + rules of engagement + minimal decision tree
- Tool descriptions: “when to call”, “what you get back”, “what to do next”

**On-demand (resources / templates)**
- `docs/index`: what docs exist, sizes, and when to load them
- `docs/concepts`: deeper mental model + invariants
- `docs/workflows/*`: scenario playbooks (maps directly to `0122-scenario-analysis`)
- `docs/toolcards/*`: tool-by-tool “gotchas” and follow-ups (beyond schema)
- `docs/record-writing`: how to keep records self-explaining and compressible

**Pros**
- predictable context footprint in normal operation
- deep guidance is available without inflating every session

**Cons**
- can add round trips if the agent routinely fetches docs
- requires discipline: instructions must *point* to resources, not duplicate them

Mitigation: design the always-on layer so agents rarely need to fetch deep docs.

---

### Option 3: “Skill Catalog” (advanced)

Expose a compact catalog of workflows (“skills”), each with its own short instruction payload, loaded only when invoked.

**Pros**
- very low baseline context
- strong guardrails per workflow (“do these calls in this order”)

**Cons**
- extra protocol surface (skills discovery/acquisition)
- can become a second system to maintain if not generated from source

When it works: complex servers with many workflows and frequent misuse by agents.

---

## Recommended baseline: Option 2 with explicit “budgets”

Set explicit size expectations for each layer to prevent drift.

**Suggested budgets (tune empirically)**
- Server Instructions: ~300–800 tokens
- Per-tool description: 1–3 short sentences
- `docs/index`: ~100–300 tokens
- Each workflow doc: ~300–1200 tokens, scoped to a single scenario

---

## Proposed content: what the agent should be able to do without fetching deep docs

### A. Cold start / orientation (1 round trip)

Agent can decide:
- which project it’s in (or that it should ask)
- whether there are open/stale sessions that might conflict
- what the top-level record landscape looks like

**Target call pattern**
1) `get_project_overview` (default project unless specified)
2) (optional) `search_records` or `list_records` to find the target

### B. Begin “reasoning mode” (1 round trip)

**Target call pattern**
- `activate(record_id)` to load: target + parent + open children + refs for others

Document the intent clearly: activation is the “I’m about to think/change things” boundary.

### C. Mutate safely (0–2 extra round trips)

**Target call pattern**
- `create_record` / `update_record` / `transition`
- handle warnings/conflicts:
  - if `update_record` returns `conflict`, do a user-visible merge, then retry with `force` only after reconciliation
  - if `activate` returns warnings about other sessions, proceed cautiously and prefer non-destructive changes first

### D. Close the loop (1 round trip)

**Target call pattern**
- `save_session` for an explicit sync point
- `close_session` when the user is done with that thread of work

---

## Drift risks to call out in docs (so agents don’t learn the wrong thing)

- If a scenario doc describes parameters the server doesn’t support (e.g., “override=true”), agents will hallucinate them.
- If “diff” endpoints are placeholders, agents must be told what they actually return today.

Mitigation:
- maintain a small “capabilities” section (in Server Instructions or `docs/index`) that lists any intentional limitations.

---

## Implementation checklist (if/when you implement this strategy)

1) Rewrite `initialize.instructions` as a real “rules of engagement” manual (budgeted).
2) Tighten tool descriptions to include:
   - “use when…” + “side effects…” + “common follow-up…”
3) Add doc resources:
   - `docs/index`, `docs/concepts`, `docs/workflows/*`, `docs/toolcards/*`
4) Add lightweight schema constraints where they reduce ambiguity (enums, min/max limits).
5) Add one round-trip “help” escape hatch (`help(topic)`) *or* ensure resource discovery is trivial via `docs/index`.

---

## Appendix: Draft “Rules of Engagement” (example structure)

This is the shape of what should live in Server Instructions (not necessarily the exact wording):

1) **Start with orientation**: call `get_project_overview` on new chats or when unsure.
2) **Browse without activation**: use `search_records` / `list_records` / `get_recent_activity` to decide where to work.
3) **Activate before reasoning or mutation**: `activate(id)` loads the minimum context bundle.
4) **Prefer references over bodies**: only load full record bodies when needed; keep OPEN children count manageable.
5) **Treat tick gap as staleness**: if warned or resuming an old thread, call `sync_session`.
6) **Handle concurrency explicitly**: warnings mean “someone else may be editing”; conflicts require reconciliation before `force`.
7) **Close the loop**: `save_session` for sync points; `close_session` when done.
