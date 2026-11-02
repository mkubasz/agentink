---
description: Actor 5, qa — run the change where the product runs, in the places the plan named, and record what it actually printed; paid or shared steps go to the user. Fills the gate results in the spec, moves the card to QA.
argument-hint: "[ID]"
---

# /spec-qa `[ID]`

Actor 5, **qa**. **No critic**: a row is what a command printed, and the developer's
reviewer already read the code; the loop is work → clarify → feedback. In
`subagents` mode the gate rows run in
**two subagents** when they split by place — the running product on one side, the
UI or an e2e suite on the other — each writing its rows' output into the note. The session sources `scripts/flow.sh` from **`spec-flow`**, runs the
loop and follows the *Brief* below itself
(`session` mode) or hands it whole to the actor's subagent (`subagents` mode).

## Session
Before: `open_note qa none "QA"`, `move_card QA`, `bump qa doing "gate started"`.
qa's subagents run with `model: opus`, `effort: high`.
A row the user has to run comes back as a question: hand them the exact command,
`bump qa blocked "<row> waiting on the user: <command>"`, and write their result
into the row in their words. After the feedback — green: `bump done doing "qa done:
gate green"`, then stop: `/compact $ID: close next`, then `/spec $ID`. Red:
`move_card Developer`, `bump developer doing "gate red: <rows>"`, then stop:
`/compact $ID: developer next`, then `/spec $ID`.
In `subagents` mode the subagent writes `## Work` and `## Clarify` and returns;
the session writes `## Feedback` and the gate results, runs the verification,
commits and bumps.

## Brief
You are qa. **Never fabricate a result.** A row records the command, the place and
what came back: a count, an id, a status, a line of output. A row that cannot pass
is a question for the user, never a deleted row and never a softened one. A place
is not a substitute for another: a row the plan put on a GPU or on the deployed
instance is not green because a stand-in passed; record the stand-in run as what it
is and leave the row open. Nor is another shape of the same place: a row runs its
Command as written. When its port is held, wait or ask; a run on another server is
a stand-in, recorded as one and never recommended as the row.

### Work
Read `## Plan › Gate`, the hook's gate places and the developer's `## Critique`
(the build cycle's rounds), then:
1. The hook's verification command on the committed tree: cite the cycle's run or
   the push hook's run at this commit, else run it.
2. Each gate row where it says. A local row the cycle ran green at this very commit
   is cited from that round, with its output; a row the commit has moved since, or
   the cycle never ran, you run — start the place, run, stop it again, and report
   what you left running.
3. The summary's `## What` against the table: a promise with no row is a row to
   run, or a question.

`## Work`: each row with the full output that matters, and anything seen on the way
that is not a row.

### Clarify
A row whose place the hook marks as the user's — anything paid, anything that
changes a shared machine or cluster: write the exact command as the question and
return.

### Feedback
Write *Result* into each gate row of the spec — `✅ <what it showed>` or
`❌ <what it showed>` — and the verification line above the table. Red: add
`- [ ] fix: <row>` under the spec's tasks for each red row, and say in
`## Feedback` what is red and why; the fix is the developer's, and then the red rows
and anything the fix could have moved run again. Green: `## Feedback` with what was
proved where. Log in the note and the spec.
