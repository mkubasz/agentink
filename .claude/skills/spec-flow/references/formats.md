# Formats — the note, the index, the summary, the spec

What each file in the two places looks like. The shell that writes them is `scripts/flow.sh`; the loop that fills them is in `SKILL.md`.

## An actor note

```markdown
---
ticket: PRJ-28
actor: architect              # analyst|researcher|architect|developer|qa
critic: hard                  # hard|balanced|none (qa)
---
`#spec/PRJ-28` · `#actor/architect`

# Architect — how

## Work
## Clarify
## Critique
## Feedback
## Log
```

The four sections are the loop, in order; qa's note has no `## Critique`. The note's `## Log` says which steps
ended (*work written*, *clarify answered*, *critique settled*, *feedback written*);
a re-run continues at the first step whose line is missing, so a stop at clarify
resumes at clarify.

## The index

```markdown
---
id: PRJ-28
title: Retry failed webhooks
step: architect               # analyst|researcher|architect|developer|qa|done
status: blocked               # todo|doing|blocked|done
mode: session                 # session|subagents
branch: feature/webhook-retry
repo: myapp
spec: docs/specs/PRJ-28-retry-failed-webhooks.md
created: 2026-10-12
updated: 2026-10-13
tags: [spec/PRJ-28, step/architect, branch/feature-webhook-retry, status/blocked]
---
<!-- spec-card -->
`#spec/PRJ-28` · `#step/architect` · `#branch/feature-webhook-retry` · repo `myapp`

# PRJ-28 — Retry failed webhooks

- [[summary]] · [[01-analyst]] · [[02-researcher]] · [[03-architect]] · [[04-developer]] · [[05-qa]] · [[postmortem]]
- spec: `docs/specs/PRJ-28-retry-failed-webhooks.md`

## Log
```

## The summary

The owner's page. Every section is short and in plain words; the long form is in
the actor's note. Each actor fills its own sections and leaves the others; `bump`
rewrites *Where it stands*. `## Decisions` takes every answered question, whichever
actor asked it (a developer's QD, qa's QA-Q), a line each, written when the answer
is recorded.

````markdown
# PRJ-28 — Retry failed webhooks

## Where it stands
architect · blocked — 3 decisions waiting on the owner (2026-10-13 14:02)

## Why
<analyst: the problem, for whom, what done looks like — three lines>

## What
<analyst: what is delivered, what is out — then the researcher's *Constraints*:
what binds it, what already exists>

## How
<architect: one paragraph>

```mermaid
flowchart LR
  <the process this change implements — the flow of the thing, not of the work>
```

## Decisions
- **D1 — <the question>** → **A, <option>** — <owner, 2026-10-13>
  - for: <…> · against: <…>
  - lost: B — <why>
- ADR: [ADR_0012](../ADR/ADR_0012.md) — proposed
````

## The repository layout

```
docs/specs/
├── README.md                  # what this folder is + the project hook
├── <ID>-<slug>.md             # the spec: the plan and the log while open, the final record after
└── <ID>-<slug>.contract.md    # optional, and only while the change is being built
```

The **extra file** is one of two, never both: `.contract.md` when the change has a
surface somebody else calls, or `.architecture.md` when it moves a boundary. The
architect writes it, the developer builds against it, close deletes it.

## The spec

Frontmatter is written once by `/spec-new`; `step`, `status`, `updated` and `tags`
are the fields a later actor rewrites, and `mode` the one `/spec <ID> subagents`
or `/spec <ID> session` rewrites.

```markdown
---
id: PRJ-28
title: Retry failed webhooks
step: architect
status: blocked
mode: session
branch: feature/webhook-retry
repo: myapp
vault: Tickets/PRJ-28-retry-failed-webhooks
created: 2026-10-12
updated: 2026-10-13
tags: [spec/PRJ-28, step/architect, branch/feature-webhook-retry, status/blocked]
---
<!-- spec-card -->

# PRJ-28 — Retry failed webhooks

## Summary
<analyst: one or two lines — what this delivers, and why>

## Plan
**Delivers:** `docs/plans/<plan>.md` §<item> · **Built by:** <one implementer | orchestrator over n> · **Extra file:** <contract | architecture | none> · **ADR:** [ADR_0012](../ADR/ADR_0012.md) proposed | none
**Mechanism:** <who retries, who buffers failures, what state outlives a request — the contract the change leans on outside its own lines> | none

- [ ] 1. <task> — D1 · proved by `<test>`
- [ ] 2. <task> — D2 · proved by `<test>`

| Behaviour | Where | Command | Result |
|---|---|---|---|
| <what a user of the product sees> | <a place from the hook> | `<command>` | — |

## Log
- 2026-10-12 10:00 — spec opened on `feature/webhook-retry`
```

The architect writes `## Plan`, the developer ticks the tasks, qa fills *Result*
with what the command printed. At close `## Plan` becomes `## Shipped` and the
decisions come in as a line each: that is the final record, about forty lines.

