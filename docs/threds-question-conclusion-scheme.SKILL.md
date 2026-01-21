---
name: threds-question-conclusion-scheme
description: A scheme for modeling conversations using threds MCP based on questions and conclusions.
---

# Question/Conclusion Reasoning Scheme

Questions and conclusions are the only elements of the reasoning model. When saving a conversation, model as these record types. Fill in inferred records as needed to assemble a coherent reasoning model.

## When to create a question

Create a new question record when:
- There are 2+ plausible options with real trade-offs.
- The user is uncertain or needs a decision to proceed.
- The answer will constrain downstream work (architecture, API shape, deadlines, cost, risk).
- You’re about to “handwave” an assumption that should be made explicit.

## When to create a conclusion

Create a conclusion record when:
- A decision is made with rationale.
- A question is resolved (even if the answer is “defer” or “don’t do this”).
- Constraints eliminate all but one option, and the reasoning matters later.

## Question writing template (content)

Include, in roughly this order:

1) **Context (situation)**: what prompted the question and what system/problem space it belongs to.
2) **The question (scope)**: the exact decision/uncertainty, scoped to what is actually being decided.
3) **Stakes**: why resolving it matters; what depends on it.
4) **Constraints**: non-negotiables and key trade-offs.
5) **Options considered**:
   - List every serious option.
   - For each option: one strong argument **for** and one strong argument **against**.
   - Call out unknowns explicitly (what would we need to learn to decide?).
6) **Decision criteria**: what evidence or principle would make one option win.
7) **Current status**: what’s next (investigation, prototype, user input, revisit date).
8) **If resolved**: point to the resolving conclusion and summarize the resolution in 1–2 sentences.

Rigor rules:
- Don’t strawman: represent rejected options fairly.
- Prefer crisp claims over vibes (“X is risky because …”).
- If an option list grows large, split into child questions rather than bloating one record.

## Conclusion writing template (content)

Suggested sections. Use your own judgement:

1) **Decision**: what we chose (and what we explicitly did *not* choose).
2) **Rationale**: the few key reasons and trade-offs that made it win.
3) **Alternatives rejected**: the top alternatives and why they were rejected.
4) **Implications / next steps**: what this changes and what work it unlocks.
5) **Conditions for reconsideration**: what new information would make us revisit (tests failing, perf budget, new requirements, user feedback, time horizon).
6) **Scope and exceptions**: where it applies, and known boundaries.

## Superseding a conclusion (continuity)

When new work contradicts a prior conclusion:
- Surface the contradiction explicitly.
- Create a new conclusion that references the old one and explains what changed.
- Preserve the chain so a future reader can follow “old → new” without guessing.

## “Files” as custom records (artifacts)

When you have reference material or outputs that should remain literal (specs, prompts, tables, checklists, snippets):
- Store them as custom records and link them from the relevant question/conclusion.
- Keep artifacts small and targeted so future activations stay cheap.
