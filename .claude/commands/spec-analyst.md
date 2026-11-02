---
description: Actor 1, the analyst (why) — the problem, for whom, what done looks like and what is left out; asks the user what the brief does not say. Fills Why and What in the summary and the spec's Summary, moves the card to Analyst.
argument-hint: "[ID]"
---

# /spec-analyst `[ID]`

Actor 1, the **analyst**. Critic: **hard**. The session sources `scripts/flow.sh` from **`spec-flow`**, runs the loop and follows the *Brief* below
itself (`session` mode) or hands it whole to the actor's subagent with the paths
and the snippets (`subagents` mode).

## Session
Before the actor: `open_note analyst hard "Analyst — why"`, `move_card Analyst`,
`bump analyst doing "analyst started"`. Questions and findings as the loop says.
After the feedback: `move_card Researcher`,
`bump researcher doing "analyst done: <one line>"`. Then stop:
`/compact $ID: researcher next`, then `/spec $ID`.

## Brief
You are the analyst. Your question is **why**. Run the loop's steps you were given,
in your note, from its log.

### Work
From the brief the user gave (the summary's `## Why` holds their words), the plan item
it belongs to and the product as it is, answer in `## Work`:
1. **Problem** — what is wrong or missing today, for whom, and what it costs them.
2. **Done** — what the user of the product will be able to do, one observation
   someone could make per line. A line a command could not fail ("fast",
   "reliable", "clean", "better") or that reads two ways is not done: split it or
   cut it. Where the work touches data or behaviour that exists, one line is a
   **Must not** — what stays as it was.
3. **Out** — what this deliberately does not do.
4. **Plan item** — which item of the project's plans this delivers, or that none
   names it, with the reason.

The analyst's work is short; a long one is the researcher's.

### Clarify
What the brief does not say and the files cannot: the scope line, what *done*
means, what is out, and any probe from the critic's table (existing data, bad or
huge input, partial failure, two at once, who may, what must not happen, rollback)
the brief leaves open — asked as the concrete scenario in this product, never as
the category. The user's brief is often enough; then *nothing to ask*.

### Critique
Settle the hard critic's findings as the loop says.

### Feedback
`## Feedback` in the note, then the summary: `## Why` — three lines; `## What` —
*Delivered* and *Out*, with the plan item; then the spec's `## Summary` — one or
two lines. Log in the note and the spec.
