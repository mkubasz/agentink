# The post-mortem

One per spec, at close. `/spec-postmortem [ID]` runs it; `/spec-close` runs it last.


**One per spec, at close**, over the whole run from the analyst to qa. A fresh
agent reads every actor note, the summary, the spec's log, the build cycle's
rounds and the commits, and judges the run: what worked, what did not, what the
owner decided, and what the repository should learn. It appends one entry to
`postmortem.md` with its actions **proposed, not done**:

```markdown
## 2026-10-13 — PRJ-28, analyst → qa

### Steps
- <what ran, in order, one line each; the rounds of the build cycle>
### Worked
- <what went straight through, and why>
### Did not work
- <what failed, was retried, or took many calls; what the critic found that was right>
### Decisions
- <what the owner chose, in their words; where they overrode a recommendation, refused a finding or corrected the session>
### Known
- Known: <stable-slug> — <what happened this time, what it cost>
### Proposed actions
- [ ] A1 — `src/x/AGENT.md`: "<the line, with its why>" — would have spared <what> — recurring (3) | new
- [ ] A2 — skill `.claude/skills/<name>/`: <what it holds> — because <what was fumbled>
- [ ] A3 — `~/.claude/skills/<name>/`: <a procedure other repositories would use>
- nowhere: <a lesson the files already state, and where>
```

Then the session puts the proposed actions to the owner, all in one round, and
**executes the agreed ones**: into files that are clean in `git status`,
uncommitted, ticked `[x]` in the entry with the file written; a refused one stays
`[ ]` with the owner's words; a target another session has dirty stays proposed
with the reason, because a revert of that file would take the other work with it.

The bar for anything written here: *would a fresh agent do something differently
next session because of it?* Three clean critic rounds are the flow working, not a
finding; session length is not a signal. Friction a file already names is one
`Known:` line under its stable slug, the same slug every time, never a full entry
and never an action; the agent greps the other tickets' `postmortem.md` for each
slug and each action's slug, and marks the action *recurring (n)* or *new* — a
high count is the strongest fix-this-first signal. Before writing an action it
re-reads the target: a later entry or the file as it stands may already hold it.

An action comes from two places, and from nowhere else:

- **the owner took or changed an approach** — overrode a recommendation, refused a
  finding, corrected the session — and the rule would have changed what the run did;
- **the run fumbled** — a command found by trial, a fact looked up twice, a
  procedure that took many calls where a snippet would take one, a build-cycle
  round that repeated — and the next run would fumble the same way.

Where it goes, decided by what it is:

| The lesson is | It goes to |
|---|---|
| a fact or rule about this repository's code | one line with its why in the nearest `AGENT.md` to that code, within that file's own line budget — root `AGENT.md` states its own, so a full file gets the lesson in the area file instead |
| a procedure with steps, used in this repository | `.claude/skills/<name>/SKILL.md`, listed wherever the repository lists its skills |
| a procedure other repositories would use | `~/.claude/skills/<name>/SKILL.md` |
| already said somewhere | nowhere; the entry names the file |

