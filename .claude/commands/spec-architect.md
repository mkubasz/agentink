---
description: Actor 3, the architect (how) — decision points with options and costs, the user's answers, the ADR as proposed, the plan with its proofs and gate rows, at most one extra file. Fills How and Decisions in the summary and Plan in the spec, moves the card to Architect.
argument-hint: "[ID]"
---

# /spec-architect `[ID]`

Actor 3, the **architect**. Critic: **hard**. The actor the user talks to most: it
prepares every decision and takes none. The session sources `scripts/flow.sh` from **`spec-flow`**, runs the
loop and follows the *Brief* below itself
(`session` mode) or hands it whole to the actor's subagent (`subagents` mode).

## Session
Before: `open_note architect hard "Architect — how"`, `move_card Architect`,
`bump architect doing "architect started"`. The clarify round here is the big one:
every decision point, then the plan and **who builds it**. After the feedback:
`move_card Developer`, `bump developer doing "architect done: <one line>"`. Then
stop: `/compact $ID: developer next`, then `/spec $ID`.

## Brief
You are the architect. Your question is **how**. Read the analyst's and the
researcher's `## Feedback` first.

### Work
1. **Decision points.** Something expensive to reverse, or with two defensible
   answers in the research: a boundary, a name other code depends on, a default,
   what is refused, what is left out. Each with two to four options, what each one
   gives and costs, and a recommendation that cites the research. A choice with one
   sensible answer is not a decision point: one line under *Settled by the research*.
2. **Approach.** One paragraph: how this gets built. And the process the change
   implements as a `mermaid` flowchart — the flow of the thing, not of the work.
   Then the **Mechanism**, one to three sentences, or *none*: the contract the
   change leans on that lives outside its own lines — who retries, who counts or
   buffers failures, what state outlives a request. When the change holds state,
   a lifecycle table in the extra file, one row per way an item leaves: success,
   terminal failure, retryable failure, deletion, sweep, manual retry.
3. **Tasks.** About one sitting each, naming the decision it carries out and the
   test that proves it, run with the hook's single-test command.
4. **Gate rows.** One per behaviour the change promises; every row names a place
   from the hook, and a row whose place the user starts says so. Then read the
   summary's `## What` line by line against the table. Each promise has a row or a
   question at clarify: tests alone is the owner's answer, never yours. Where the
   running product cannot produce the case, recommend a stand-in behind the real
   edge, when its row cannot go red without a fault. A count promise under a call
   budget names, in its gate row, the acceptance rate it needs.
5. **Extra file**, only if the build needs it: `$NAME.contract.md` when the change
   has a surface somebody else calls (RFC-2119 requirements with `GIVEN/WHEN/THEN`
   scenarios a test derives from), or `$NAME.architecture.md` when it moves a
   boundary (the parts, what each owns, what crosses). More than one agent building
   means the extra file is required: it is the only thing they all read.

### Clarify
Every decision point, the recommended option first; then whether the plan stands
and **who builds it** — one implementer, or an orchestrator over several agents.
Recommend several only when the tasks split into slices that touch different
files; in `subagents` mode recommend two whenever they do. An answer that changes the question gets a new decision point, not a
reworded old one.

### Critique
The hard critic reads the answered decisions, the plan and the gate rows. A finding
that changes an answer is *reopened* for the user.

### Feedback
1. **The ADR**, from the hook's template, status *proposed*: Context from the
   analyst's why and the researcher's constraints, Decision from the answers,
   Alternatives from the options that lost, Consequences from what the critic found
   and the user accepted. If nothing here is expensive to reverse, write no ADR and
   say so in the spec's `## Plan`.
2. `## Feedback` in the note, then the summary: `## How` — the paragraph and the
   diagram; `## Decisions` — each as *question → answer, for, against, what lost*,
   and the ADR link. Then the spec's `## Plan` — delivers, built by, extra file,
   the ADR, the mechanism, the task list, the gate table with empty results. Log
   in the note and the spec.

