# The critic

A fresh agent, never the author, at one of two levels. `/spec-critic [ID] [hard|balanced]` runs it on purpose.


A fresh **read-only agent**, given: `vault_read` from `flow.sh`, which falls back to
the note's file under `$VAULT_DIR`, or the notes' paths on disk, read-only; the vault
paths of the note and the summary; the spec's path; and the hook's paths for plans,
decision records and guardrails.
Never the session's reasoning or a summary. It reads, it does not edit, and it
returns findings as `blocks`, `should` or `note`, each with evidence as `path:line`
and what would settle it. It ends, findings or none, with what it covered:
`Coverage: <read>/<in scope>` and one `Not read: <path> — <why>` per file left out —
the diff's files for the reviewer, the paths it was given for the hard critic. A
file only grepped was not read.

The refusals already under `## Critique` (`SKILL.md`, *Critique*) are read first: a
finding settled there with its evidence is not raised again unless the evidence is
wrong.

**hard** — analyst, researcher, architect:

> Read `<note>`, `<summary>` and `<spec>`. You did not write them and you are not here to
> approve them. Assume they are wrong and find the three things most likely to
> make them wrong: a claim with no citation; a decision that contradicts a
> decision record, a guardrail or a plan — cite the clause; a case nobody
> covered, from this table, each as the concrete scenario in this product: data
> that already exists; input empty, huge or malformed; partial failure and retry;
> two callers at once; who is allowed; what must not happen; rollout and rollback;
> how anyone would know it works; a limit; what it costs to run. A *Done* line no
> command could fail, or that reads two ways; a *Must not* missing where existing
> data or behaviour is touched; an *assumed* answer; a simpler way nobody
> listed, including not doing this; behaviour that cannot be shown at any gate
> place the hook lists. Do not list what holds. Do not propose a redesign.

**balanced** — the developer's **reviewer**, on `opus` at `high`, new each round:

> Read `<note>`, `<summary>` and `<spec>`. Find what is contested and what is missing, and
> say in one line each what holds. For the developer, read the diff
> (`git diff HEAD` plus untracked, or `git diff $(git merge-base HEAD main)...HEAD`
> on a branch) against the summary's `## Decisions`, the spec's `## Plan` and the
> contract's scenarios:
> behaviour nobody specified is a finding even when it is good; which test would
> still pass with the change removed; error paths; the guardrails for every path
> touched; every caller and consumer of the plan's **Mechanism** still honouring
> it; anything generated and not regenerated; anything secret.

qa has no critic: a gate row is what a command printed, and the reviewer already
read the code. The hard critic runs on the session's model.

The reviewer is the developer's critic and the build cycle's review step; where the session offers `/code-review`,
run it too and fold in what it finds. `/spec-critic [ID] [hard|balanced]` runs the
critic on purpose on the current actor's note, for instance after a decision changed.

