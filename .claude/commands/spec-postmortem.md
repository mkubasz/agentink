---
description: The one post-mortem of a spec, at close — a fresh agent reads the whole run from the analyst to qa, writes what worked, what did not and what the owner decided, and proposes actions (an AGENT.md line, a skill); the owner agrees, then they are executed.
argument-hint: "[ID]"
---

# /spec-postmortem `[ID]`

Runs as the last step of `/spec-close`, and on its own for a spec that is closed or
abandoned. Source `scripts/flow.sh` from **`spec-flow`** (the preamble and every snippet); the
post-mortem contract is `references/postmortem.md`: what worked, what did not, what the owner decided, and
whether the repository should learn a skill or an `AGENT.md` line from it —
proposed first, done only once the owner agrees.

## Steps
1. Source the script → `hook` → `resolve_spec $ID`. Make sure
   `$TDIR/postmortem.md` exists (`obs create` with `# <ID> — post-mortems`, skipped
   if it does).
2. **Gather what the agent reads**, as paths and commands, not as an account:
   - the vault: `obs read path=$TDIR/<note>` for the index, the summary (if it still
     exists), every actor note — the developer's `## Critique` holds the build
     cycle's rounds;
   - the spec's `## Log` and its final record;
   - the commits: `git -C "$REPO" log --format='%h %s' <first sha>^..HEAD -- <paths>`
     from the developer's feedback, and `git -C "$REPO" status --porcelain` for
     which targets are clean;
   - the guardrails the hook names, wherever the repository lists its skills, the
     memory index if the session has one.
3. **Start a fresh agent** (it writes to the vault through `obs` only, and nothing
   else yet). Its brief:

   > Write the post-mortem of the whole run of `<ID>`, analyst to qa, into
   > `<postmortem path>` with `obs append`, as one entry in the format `spec-flow`
   > gives: **Steps** (what ran, in order, the build cycle's rounds), **Worked**,
   > **Did not work** (what failed or was retried, what took many calls, what the
   > critic found that was right), **Decisions** (what the owner chose, in their
   > words, and where they overrode a recommendation, refused a finding or corrected
   > the session), **Known** (friction a file already names, one `Known: <stable-slug>
   > — <what, cost>` line each, the slug the other tickets' `postmortem.md` use),
   > **Proposed actions** — each `- [ ] An — <target>: <the exact line
   > or the skill's contents in a sentence> — <what it would have spared> —
   > recurring (n) | new`, the count from grepping the other tickets' `postmortem.md`
   > for its slug.
   >
   > Write only what would make a fresh agent act differently next session. An
   > action comes from a decision the owner took or changed, or from a fumble the
   > next run would repeat; nothing else. Check first that no guardrail, skill,
   > memory or `docs/specs/README.md` already says it, and list those as *nowhere*
   > or under **Known**. Re-read each target before proposing: it may hold it already.
   > File each by what it is: a fact about this repository's code is one line with
   > its why in the nearest `AGENT.md`, inside that file's line budget; a procedure
   > with steps used here is a skill at `.claude/skills/<name>/SKILL.md`; a procedure other repositories would use goes
   > to `~/.claude/skills/<name>/`. Say for each whether its target is clean in
   > `git status`. Propose; do not write any target. Do not restate what the files
   > already say.

4. **Ask the owner**, all proposed actions in one round, with the question tool
   where the session has one. Then **execute the agreed ones** in the session or by
   the same agent: into clean targets only, uncommitted; tick each `[x]` in the
   entry with the file written, leave a refused one `[ ]` with the owner's words,
   and a dirty target proposed with the reason.
5. **Report** in the reply: the entry's path, every repository file written, and
   what was refused or left proposed.
