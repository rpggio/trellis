package mcp

import (
	"context"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverInstructions = `threds-mcp stores durable reasoning as Projects → Records → Sessions.

Core concepts (keep this mental model small):
- Project: a container with a monotonic tick (logical clock). “Stale” means tick gap, not wall time.
- Record: self-explaining reasoning unit in a parent/child tree (+ optional related[] links).
- RecordRef: lightweight summary (no body) for browsing/search.
- Session: a chat’s working context; tracks last synced tick and activated records.
- Activation boundary: call activate(id) before doing serious reasoning or mutation; it returns a minimal ContextBundle.

Rules of engagement (default workflow):
1) Orient: call get_project_overview (default project unless project_id provided).
2) Browse cheaply: use search_records / list_records / get_recent_activity / get_record_ref (prefer RecordRef over full bodies).
3) Reason/mutate: call activate(record_id) to load Target + Parent + OPEN children (full), and refs for the rest.
4) Write safely: create_record / update_record / transition.
   - If update_record returns a conflict, reconcile explicitly; only retry with force=true after merging.
   - If activate returns warnings about other sessions, proceed cautiously.
5) Staleness: if tick_gap > 0 (overview) or staleness > 0 (sync_session), call sync_session before significant edits.
6) Close the loop: save_session for an explicit sync point; close_session when done.

Transport notes:
- HTTP: pass session id via Mcp-Session-Id header.
- Stdio: pass session id via _meta.session_id when supported; otherwise some tools accept session_id arguments.

Docs (progressive disclosure):
- threds://docs/index (what to read when)
- threds://docs/concepts (glossary + invariants)
- threds://docs/workflows/cold-start
- threds://docs/workflows/activation-and-writing
- threds://docs/workflows/conflicts
- threds://docs/record-writing
`

type docResource struct {
	URI         string
	Name        string
	Title       string
	Description string
	Content     string
}

var docResources = []docResource{
	{
		URI:         "threds://docs/index",
		Name:        "docs_index",
		Title:       "threds-mcp docs index",
		Description: "Entry point for agent-facing docs: what exists, what to read, and known limitations.",
		Content: `# threds-mcp: Agent Docs Index

This server is designed for **progressive disclosure**: keep your baseline context small and load deeper docs only when needed.

## Quick start (no deep docs)

1. ` + "`get_project_overview`" + ` to orient (project tick, open sessions, root records).
2. ` + "`search_records`" + ` / ` + "`list_records`" + ` to find a target record (prefer refs).
3. ` + "`activate`" + ` before serious reasoning or mutation.
4. Mutate via ` + "`create_record`" + ` / ` + "`update_record`" + ` / ` + "`transition`" + `.
5. Use ` + "`sync_session`" + ` when tick gap indicates staleness.
6. End with ` + "`save_session`" + ` and ` + "`close_session`" + `.

## Docs (read on demand)

- ` + "`threds://docs/concepts`" + ` — glossary + invariants (activation boundary, tick-gap staleness, concurrency).
- ` + "`threds://docs/record-writing`" + ` — how to write records that remain self-explaining and cheap to load.
- ` + "`threds://docs/workflows/cold-start`" + ` — “new chat / resume work” playbook.
- ` + "`threds://docs/workflows/activation-and-writing`" + ` — the normal reasoning + mutation loop.
- ` + "`threds://docs/workflows/conflicts`" + ` — conflict handling + safe use of ` + "`force`" + `.

## Capabilities & intentional limitations

- ` + "`get_record_diff`" + ` is currently a placeholder (returns the current version for both sides; no computed diff yet).
- Browse tools can return large result sets if you omit ` + "`limit`" + `; use limits to control token usage.

## Where sizes live

` + "`resources/list`" + ` returns each doc resource with a ` + "`size`" + ` (bytes) estimate so clients can budget context.
`,
	},
	{
		URI:         "threds://docs/concepts",
		Name:        "docs_concepts",
		Title:       "Concepts and invariants",
		Description: "Mental model + invariant rules: tick-gap staleness, activation boundary, record tree, and sessions.",
		Content: `# Concepts and invariants

## Glossary

- **Project**: container for records. Has a monotonic **tick** that increments on every write.
- **Tick gap**: ` + "`project.tick - session.last_sync_tick`" + ` (or similar). This is “staleness”.
- **Record**: unit of reasoning with ` + "`type`" + `, ` + "`title`" + `, ` + "`summary`" + `, ` + "`body`" + `, and workflow ` + "`state`" + `.
- **Record tree**: records can have ` + "`parent_id`" + ` and children. Use this to keep reasoning localized.
- **RecordRef**: lightweight record pointer used for browsing/search (no body).
- **Session**: a chat’s working context. Sessions help detect staleness and concurrent activity.
- **Activation**: the boundary between browsing and deep reasoning/mutation.

## Activation boundary (why it exists)

Record bodies are the expensive part of context. ` + "`activate(record_id)`" + ` returns a **minimal bundle** you can reason over:

- target record (full)
- parent record (full, if present)
- OPEN children (full)
- other children + grandchildren (refs)

This keeps routine navigation cheap and makes “now I’m going to think/change things” explicit.

## Workflow states

Records use ` + "`OPEN | LATER | RESOLVED | DISCARDED`" + `. State controls what gets loaded (OPEN children are prioritized).

## Staleness is tick-gap, not time

- Prefer tick-based checks over wall-clock heuristics.
- If you’re resuming work or see a tick gap, call ` + "`sync_session`" + ` before doing significant edits.

## Concurrency and conflicts

This server is “single-writer intent”: concurrent edits aren’t forbidden, but they must be handled explicitly.

- ` + "`activate`" + ` may return warnings if other sessions are active on the same record.
- ` + "`update_record`" + ` may return a **conflict** instead of a record. Treat this as “stop and reconcile”.
- Use ` + "`force=true`" + ` only after you’ve compared versions and intentionally merged.
`,
	},
	{
		URI:         "threds://docs/workflows/cold-start",
		Name:        "docs_workflow_cold_start",
		Title:       "Workflow: cold start / resume",
		Description: "Playbook for new chats or resuming work without pulling too much context.",
		Content: `# Workflow: cold start / resume

Goal: orient yourself with **one call**, then decide what to activate.

## 1) Orient (one call)

Call ` + "`get_project_overview`" + ` (optionally with ` + "`project_id`" + `).

Use it to answer:
- Which project am I in?
- What’s the project tick?
- Are there open sessions with a tick gap warning?
- What are the root records?

## 2) Find a target cheaply

Use one of:
- ` + "`search_records`" + ` (recommended; supply a query and a ` + "`limit`" + `)
- ` + "`list_records`" + ` (e.g., list root records or children under a parent)
- ` + "`get_recent_activity`" + ` (to see what changed without activating)
- ` + "`get_record_ref`" + ` (when you already have an id)

Avoid loading record bodies until you’ve picked a target.

## 3) Enter reasoning mode

Call ` + "`activate({id})`" + ` once you know where to work.

If activation returns warnings (other sessions), prefer non-destructive changes until you’ve checked for conflicts.
`,
	},
	{
		URI:         "threds://docs/workflows/activation-and-writing",
		Name:        "docs_workflow_activation_and_writing",
		Title:       "Workflow: activation and writing",
		Description: "Playbook for the normal loop: activate → write → sync/save/close.",
		Content: `# Workflow: activation and writing

## Normal loop

1) ` + "`activate(record_id)`" + ` to load a ContextBundle.

2) Make changes:
- Add a child record: ` + "`create_record(parent_id=target.id, type, title, summary, body)`" + `.
- Update current record: ` + "`update_record(id, title/summary/body, related[])`" + `.
- Transition state: ` + "`transition(id, to_state, reason)`" + `.

3) Watch for conflicts:
- If ` + "`update_record`" + ` returns ` + "`conflict`" + `, reconcile and retry (only then consider ` + "`force=true`" + `).

4) Keep sessions fresh:
- If you see tick gap warnings (overview) or ` + "`sync_session`" + ` reports staleness > 0, sync before further edits.

5) Close the loop:
- ` + "`save_session`" + ` at a meaningful checkpoint.
- ` + "`close_session`" + ` when the user is done with that thread.

## Session ID passing

- HTTP: use the ` + "`Mcp-Session-Id`" + ` header.
- Stdio: prefer ` + "`_meta.session_id`" + `. If your client can’t set it, pass ` + "`session_id`" + ` arguments where supported (e.g. ` + "`update_record`" + `, ` + "`save_session`" + `, ` + "`close_session`" + `).
`,
		},
	{
		URI:         "threds://docs/workflows/conflicts",
		Name:        "docs_workflow_conflicts",
		Title:       "Workflow: conflicts and force",
		Description: "How to handle update conflicts and use force safely.",
		Content: `# Workflow: conflicts and force

## What a conflict means

` + "`update_record`" + ` can return a ` + "`conflict`" + ` instead of an updated record. This usually means another session wrote to the record since your session’s last synced view.

Treat this as: **stop, compare, and reconcile**.

## Safe reconciliation protocol

1) Show the user the conflict summary (or describe it succinctly).
2) Compare your intended changes against ` + "`conflict.other_version`" + `.
3) Create a merged update intentionally (title/summary/body).
4) Retry ` + "`update_record`" + ` with the merged fields.
5) Only if you *must* override and you understand the loss, retry with ` + "`force=true`" + `.

## When to use force

Use ` + "`force=true`" + ` only when:
- you’ve compared both versions, and
- you’re intentionally choosing one side (or have merged manually).

Avoid “blind force”; it can discard someone else’s work.
`,
	},
	{
		URI:         "threds://docs/record-writing",
		Name:        "docs_record_writing",
		Title:       "Record writing guide",
		Description: "How to write records that remain self-explaining and cheap to load in future activations.",
		Content: `# Record writing guide

Records should be self-explaining without rereading chat transcripts.

## Field guidance

- Title: short and specific (“Decision: …”, “Question: …”, “Plan: …”).
- Summary: 1–3 sentences. Enough to decide whether to activate.
- Body: structured, skimmable, and bounded in length.

## Recommended body structure

Use headings/bullets so the content compresses well:

- Context: what problem this record addresses.
- Constraints: non-negotiables and trade-offs.
- Options considered: 2–5 bullets max.
- Decision / next steps: what we chose and what remains.
- Links: related record ids in ` + "`related[]`" + ` (keep references explicit).

## Keep activations cheap

- Prefer creating multiple small records over one massive one.
- Keep OPEN children count manageable; resolve/discard stale branches.
- Use RecordRefs (search/list) until you’re ready to reason deeply.
`,
	},
}

func registerDocResources(server *sdkmcp.Server) {
	for _, doc := range docResources {
		doc := doc

		server.AddResource(&sdkmcp.Resource{
			URI:         doc.URI,
			Name:        doc.Name,
			Title:       doc.Title,
			Description: doc.Description,
			MIMEType:    "text/markdown",
			Size:        int64(len(doc.Content)),
		}, func(_ context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
			uri := doc.URI
			if req != nil && req.Params != nil && req.Params.URI != "" {
				uri = req.Params.URI
			}
			return &sdkmcp.ReadResourceResult{
				Contents: []*sdkmcp.ResourceContents{{
					URI:      uri,
					MIMEType: "text/markdown",
					Text:     doc.Content,
				}},
			}, nil
		})
	}
}
