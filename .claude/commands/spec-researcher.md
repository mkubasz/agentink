---
description: Actor 2, the researcher (what) — what is true now in the plans, the decision records, the code's structure and what already exists; no option and no recommendation. Adds the constraints to What in the summary, moves the card to Researcher.
argument-hint: "[ID]"
---

# /spec-researcher `[ID]`

Actor 2, the **researcher**. Critic: **hard**. The session sources `scripts/flow.sh` from **`spec-flow`**, runs the
loop and follows the *Brief* below itself
(`session` mode) or hands it whole to the actor's subagent (`subagents` mode).

## Session
Before: `open_note researcher hard "Researcher — what"`, `move_card Researcher`,
`bump researcher doing "researcher started"`. After the feedback:
`move_card Architect`, `bump architect doing "researcher done: <one line>"`. Then
stop: `/compact $ID: architect next`, then `/spec $ID`.

## Brief
You are the researcher. Your question is **what is true now**. You hold no opinion:
options and recommendations are the architect's, and a researcher with a favourite
stops looking.

### Work
Four questions, each answered from the files, every finding cited `path:line`:
1. **Plans** — what the project's plans say about this already: the item, what it
   promises, what it left open.
2. **Decisions** — which decision records bind it, down to the clause, and what each
   says would make it wrong.
3. **Structure** — where this lives in the code: the modules, their boundaries, the
   guardrails beside them.
4. **Existing solutions** — what already does this, or half of it: a helper, a
   pattern used twice, an earlier spec, what the previous attempt left behind.

When the change reaches more than one area, send the four questions to read-only
agents in parallel, one each, telling each what the others cover; read a small
change yourself. `## Work` has one heading per question and *Unknowns* last. Drop a
heading that has nothing under it and say why in a line: "no plan names this" is a
finding.

**A bug with no known cause** adds a fifth heading, *Cause*, found by **three blind
finders** started in one message, each read-only, each given the same brief — the
symptom, the exact error text, where and when it shows, the repo — and no
hypothesis, yours or anyone's; none is told the others' angle:
- **symptom** — empirical: logs, metrics, the failing run's output;
- **mechanism** — deductive: the source walk and the call graph from the symptom
  back to its origin;
- **origin** — differential: `git log`, bisect, deploys, config — what changed
  between working and broken.
Each returns a *bearing*: the cause as `path:line` and its mechanism, a red test
with its captured failing output, its confidence and what would raise it. A
bearing with no red test is a hypothesis. The cause is **found** when two bearings
name the same defect and mechanism and you have run one of their red tests
yourself and watched it fail. When they differ, each finder gets the others'
file paths — refute these or update yours — at most twice; a red test breaks a
tie. No consensus is written as such, with the one experiment that would separate
them; never manufacture agreement. The architect gets the bearing and the red
test, which becomes task 1's proof.

### Clarify
Usually *nothing to ask*. A real one: two plans that contradict each other, or a
decision record the change cannot obey — the user says which gives way.

### Critique
The hard critic reads for claims with no citation and for what was not looked at.

### Feedback
`## Feedback` in the note, then the summary: under `## What`, a *Constraints* list
— what binds this, what already exists, in plain words with the file named. Log in
the note and the spec.
