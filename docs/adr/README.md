# Architecture Decision Records

One ADR per decision that is expensive to reverse, and one per new dependency.
Files are `NNNN-<slug>.md`, written in English, and never rewritten once
*accepted*: a later ADR supersedes an earlier one and both say so.

| # | Title | Status |
|---|---|---|
| [0001](0001-bounded-contexts-events-and-vertical-slices.md) | Bounded contexts, an event-driven core and vertical slices | proposed |
| [0002](0002-monorepo-layout.md) | Monorepo layout | proposed |
| [0003](0003-clean-room-rebuild-and-provenance.md) | Clean-room rebuild and provenance rules | proposed |
| [0004](0004-go-module-path.md) | Go module path | proposed |
| [0005](0005-event-log-format-and-identifiers.md) | Event log format and identifiers | proposed |
| [0006](0006-page-rendering-with-typst.md) | Page rendering with Typst, answer zones from the template | proposed |
| [0007](0007-worker-as-a-local-service.md) | The Python worker is a local service that `inkd` supervises | proposed |
| [0008](0008-contracts-validated-not-generated.md) | Contracts: JSON Schema as the source of truth, validated on both sides | proposed |
| [0009](0009-just-as-the-task-runner.md) | `just` is the task runner | accepted |

## Template

```markdown
# NNNN — <title>

- **Status:** proposed | accepted | superseded by NNNN
- **Date:** YYYY-MM-DD
- **Spec:** docs/specs/<ID>-<slug>.md · **Decision point:** D<n>

## Context
<what is true that forces a choice; cite files as path:line>

## Decision
<what we do, in the present tense>

## Alternatives
<each option that lost, and why>

## Consequences
<what gets easier, what gets harder, what must be verified>
```

Status moves from *proposed* to *accepted* at the spec's close, after the gate
is green (see `docs/specs/README.md`).
