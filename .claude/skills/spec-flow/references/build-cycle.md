# The build cycle

The developer's critique and qa's work, run in rounds until everything holds.


The developer's critique and qa's work are one **cycle**, run by the session in
rounds until everything holds, and the user is asked only when a round is stuck:

> **review → tests → fixes → review → tests → fixes → …**

| Step | Who | What it is |
|---|---|---|
| review | the **reviewer**: a fresh balanced critic, new each round, on `opus` at `high` | the diff against the summary, the spec and the contract: correctness, the guardrails, modularity and the practices the area's `AGENT.md` names, a test that would pass with the change removed, anything generated and stale, anything secret; `/code-review` folded in where the session offers it |
| tests | the session, by the hook's commands | the proof of every touched task (the single-test command), the pre-commit command (the linter and the hygiene the hook names), and the plan's gate rows whose place the session can start — the product running: the dev server, the e2e suite, the UI. A proof counts once it has been **seen red**: a new test red before the code landed, or else one **mutation** — the smallest compiling break of the task's own code, the proof run, the break reverted — and the proof must fail on that task's assertion, not on a missing symbol. The round's line carries each as `broke: <file:line> → FAILED` or `→ SURVIVED → rewritten`; a proof that survives its rewritten mutation is a `blocks`. A forecast from a judged grid (acceptance, precision) is the fresh reviewer's blind count, the builder's beside it |
| fixes | the developer — the session in `session` mode; in `subagents` mode its subagent, continued or started again from its note, or two in worktrees when the fixes split by file | every `blocks` and `should`, every failing proof, every red row; it writes what it changed under `## Work`. A finding the review has made before at another place is fixed for its kind: the developer searches every place the fault can occur, holds the rule at the boundary where there is one (one handler around a function's body, not one per call), and the round's line names the search |

A round is **green** when the review returns no `blocks` and no `should`, every
proof and the pre-commit pass, and every local gate row shows what the plan said
it must. The green round is followed by the hook's verification command whole, on
the tree as it stands, and then the developer's commit by path with the hooks on;
the commit itself proves the pre-commit. That run counts at the commit only if
every new file was in the index when it started (`git add -N <files>`: a check
that lists tracked files passes over the rest; in `subagents` mode the session runs
it, before its verification run, and a developer subagent leaves the shared index
alone) and nothing but the spec was
edited after it ended. A late fix, a `note` included, means the run again. The
round's line names the log and when it ended. A row whose place is the user's —
paid, shared — is not in the cycle: it is qa's question after the commit.

The cycle has **two tiers, in order**. *Local* first, as above, until green and
committed. Then *CI/CD*: before each push the developer runs `git merge main` into the spec
branch, a conflict being a round; then it pushes the branch it is on and watches the
pipeline the hook names; a red job is a round — fix, commit, push — under the same
cap. A push to a branch that deploys (the hook says which) is the owner's: the
cycle prepares it and asks. A finding, a proof, a row or a job still red after
three rounds, or anything the developer cannot fix, is a question for the user with
the attempts: the cycle never loops on what it cannot fix.
Each round is one dated line in the developer's note under `## Critique`, written
with `section_line "$(note developer)" Critique "round n: …"` so the time is the
script's, never typed —
*round n: <findings> · <proofs> · <rows>* — so the post-mortem can read what was
hard. A finding refused in a round stays in that line with its kind and evidence
(`SKILL.md`, *Critique*), so the next reviewer, fresh, reads it and does not raise
it again.

qa, after the commit, records the gate rows in the spec. A local row the cycle ran
green at this very commit is cited from the round, not run again; a row the commit
has moved since runs again; the user's rows are handed over as the loop says.


