---
description: Actor 4, the developer — build the plan's tasks, alone or over agents in worktrees; the reviewer, a fresh balanced critic, is its critic each round; then the commit by path with the hooks on. Ticks the Plan in the spec, moves the card to Developer.
argument-hint: "[ID]"
---

# /spec-developer `[ID]`

Actor 4, the **developer**. Critic: the **reviewer** — a fresh balanced agent, the
build cycle's review step. Who builds is
the **Built by** line the user answered in `## Plan`: one implementer, or an
orchestrator over agents in worktrees. In `subagents` mode the default is **two
agents** whenever the tasks split into two slices with disjoint files; the
architect recommends that split, and the user confirms it. The session sources `scripts/flow.sh` from **`spec-flow`**, runs the
loop and follows the *Brief* below itself (`session` mode) or hands it to
the subagent (`subagents` mode).

## Session
Before: `open_note developer balanced "Developer"`, `move_card Developer`,
`bump developer doing "build started"`. Every agent this actor starts — the
developer in `subagents` mode, its slices in worktrees, the reviewer each round —
runs with `model: opus`, `effort: high`. After the feedback: `move_card QA`,
`bump qa doing "developer done: <sha> <subject>"`. Then stop:
`/compact $ID: qa next`, then `/spec $ID`.
In `subagents` mode the subagent writes `## Work` and `## Clarify` and returns;
the session writes `## Critique` and `## Feedback`, runs the verification,
commits and bumps.

## Brief
You are the developer. Read `## Plan` and the extra file.

### Work
**One implementer.** Work the tasks in order. Each task lands with the test that
proves it and is ticked in the spec when that test passes — the hook's
single-test command, not the whole suite — and has been seen red once: a new
test run before the code, or the cycle's mutation of the task's own code
(`references/build-cycle.md`). A proof never seen red proves nothing and the
task stays unticked.

**Orchestrator.** Prepare and control, and write no feature code yourself.
*Prepare:* cut the tasks into slices whose files do not overlap; brief each agent
with its tasks, the files it owns and must not touch, the extra file, the guardrails
for its paths, the proof of each task, that it does not commit, and that before it
returns it stops every background command it started — a server, a `tail -f` — by
its PID and lists them (a port check does not see a `tail`); one worktree per agent. *Control:* read each agent's diff, not its summary; bring the slices in
one at a time, run the tests after each, send a slice back with what is wrong. A
slice that needs another slice's file is a planning mistake: stop and say so.

Tick each task as it lands and log it; the checklist is the progress record, kept
honest rather than tidy. Regenerate what the hook says is generated when the change
touches its source. `## Work` is the build log: what landed, what was hard, what
moved from the plan. An answer that changes a number the spec states is built with a
search for the old number over the gate rows and the harness, each hit changed or
named in the round line. A re-run — after a red gate, or a reopened decision — works the
unticked tasks only. A **fix** (the plan has one `fix:` task and no ADR) is proved
by the test that failed before and passes after; a fix that turns out to need a
decision is not a fix — write why under `## Work`, return, and the session runs
`bump architect doing "not a fix: <why>"`.

### Clarify
A task that needs a decision nobody took is not yours to take: write the question
and return.

### Critique
The **build cycle** from `spec-flow`: the session runs review → tests → fixes in
rounds. You are the *fixes* step: each round you get the review's findings, the
failing proofs and the red rows, work them, and write what changed under `## Work`;
the session logs the round under `## Critique`. A finding that says the plan is
wrong is *for the user*. The local tier ends green, then the hook's verification
runs whole, then you commit; the CI tier pushes the branch and watches the
pipeline the hook names, a red job being a round — unless the branch deploys,
which is the user's push.

### Feedback
1. **Commit**, once the local tier is green. Stage by path — the files this spec
   changed, the spec, the extra file, the ADR — never the whole tree. Hooks on; the
   hook names the message rule; a failing hook is fixed and the commit made again,
   never bypassed. Then the CI tier, as the cycle says.
2. `## Feedback` in the note: what shipped, the commit sha, the pipeline's result,
   what the critic found and what became of it, what qa should look at first. Log
   in the note. The spec's line is the session's `bump qa doing "developer done: …"`,
   so add none of your own.

