# The spec flow — how it works and why

A reader's page. `SKILL.md` is the contract, `INSTALL.md` puts it on a machine;
this says what the thing is.

## The approach

One change is one spec. Five actors carry it, each answering one question:
the **analyst** (why), the **researcher** (what is true today), the
**architect** (how), the **developer**, **qa**. Every actor runs the same loop —
**work → clarify → critique → feedback** — and the owner decides only inside
*clarify*: one round of questions, the recommended answer first; an actor with
nothing to ask says so in a line and goes on. The critic is always a fresh agent
handed paths, never a summary: *hard* for the three who think, a balanced
*reviewer* for the developer; qa has none, a gate row being what a command printed. The first three build the ADR between them.

Two places. The Obsidian vault is the workshop: a ticket folder with one note per
actor (the long form, every finding cited `path:line`), `summary.md` for the
owner (where it stands, why, what, how with a flow diagram, the decisions with
their for and against) and the post-mortem. The repository is the record: one
spec file (a line of summary, the plan with its gate, the log; the final record
after close), the ADR, at most one contract file while building. The summary is
temporary; close deletes it. The board lives in the vault.

## The flow

```
 owner ─ brief ─▶ /spec "…"          kind: change | fix | continue
                    │
                    ▼
   ┌──────── session mode: one run = one actor ────────────────────────┐
   │  resolve_spec  → the state comes from FILES, not the conversation: │
   │                  spec.md (step/status/mode) + the note's log       │
   │  open_note, move_card, bump                                        │
   │                                                                    │
   │   ┌────── actor: work → clarify → critique → feedback ─────────┐   │
   │   │  work      long form → vault: 0N-actor.md ## Work          │   │
   │   │  clarify   questions → OWNER (one round) → answers in note │   │
   │   │  critique  a FRESH critic agent (paths, never a summary)   │   │
   │   │  feedback  short form → summary.md + spec.md + log         │   │
   │   └────────────────────────────────────────────────────────────┘   │
   │  bump <next>  →  STOP:  "/compact ID: <next> next"                 │
   └────────────────────────────────────────────────────────────────────┘
                    │  owner: /compact, then /spec ID
                    ▼
   analyst → researcher → architect → developer ⇄ qa → close
                                        └─ build cycle ─┘     │
                                  review → tests → fixes       └─ post-mortem
                                  (local first, then CI)          proposals → owner's yes → actions
```

The stop/compact boundary above is for `session` mode. In `subagents` mode the
manager starts a fresh actor after each Feedback without a compact; it stops for
owner questions, required approvals or owner-started gates, unresolved blockers,
or close.

`/spec "brief"` names the kind of work first and enters where it belongs: a
`change` at the analyst; a `fix` — a wrong behaviour with a known cause — at the
developer with a one-task plan; a `continue` where the spec stands. A fix that
turns out to need a decision goes back to the architect. The developer and qa
share the **build cycle**: review → tests → fixes in rounds, local until the
critic has nothing left, the proofs and the pre-commit pass and the product runs,
then the commit and CI; three red rounds on the same item, or anything unfixable,
go to the owner. Close accepts the ADR, updates the plan item, cuts the spec to
its record, deletes the summary and runs the one post-mortem of the whole run.

## Memory and delegation

- The conversation is not the memory. The files are: the actor's note (long
  form), `summary.md` (the owner's page), the spec (plan and log). The next actor
  reads its predecessors' `## Feedback` and the summary, never their whole work.
- State is two fields and a log: `step/status` in the frontmatter, and the lines
  *work written / clarify answered / critique settled / feedback written* in the
  note. `next_actor` computes what runs next from them; an interrupted step
  resumes at the first missing line.
- In `session` mode, after every actor `/spec` stops and asks for `/compact`.
  After the compact the session knows the spec's ID and nothing else; it reads
  the rest. In `subagents` mode a fresh actor provides the context boundary, so
  the manager continues automatically without asking for a compact or another run.
- The critic and the post-mortem never inherit the session's context: they get
  paths and read for themselves, which is why they cannot confirm its conclusions.

## Delegation

| Who | `session` mode (default) | `subagents` mode |
|---|---|---|
| an actor | the session itself, from the command file as its brief | a subagent with the same file as its brief; the session is the manager: it starts, carries questions to the owner and answers back, hands over the critic's findings, verifies feedback, bumps and starts the next actor |
| researcher | four questions to four read-only agents in parallel | the same, inside the subagent |
| critic | always a fresh agent: hard, or the developer's reviewer; qa has none | the same |
| developer | one implementer, or an orchestrator over agents in worktrees when the plan splits | two in worktrees by default when the tasks have disjoint files |
| qa | the session; paid or shared steps go to the owner as a ready command | two by default when the gate rows split by place (the product / the UI or e2e) |
| post-mortem | a fresh agent at close, reading the whole ticket folder | the same |

The brief is always the command file plus paths, never a summary from the
session. A subagent cannot ask the owner, so it writes its questions under
`## Clarify` and returns; the session asks. The developer, its agents, the
reviewer and qa run on `opus` at `high`; everyone else on the session's model.

## By example — the architect's run on PRJ-26

One `/spec PRJ-26` in `subagents` mode, read top to bottom:

1. **Start + work.** The session resolves the spec from the files (step
   architect, mode subagents), opens the note, moves the card, bumps, and starts
   the architect with its command file and paths — never a summary. The architect
   reads the analyst's and the researcher's feedback and writes `## Work`.
2. **Clarify.** The architect cannot ask the owner, so it writes three decision
   points under `## Clarify` and returns. The session asks the owner once, the
   recommended option first, writes the answers into the note in the owner's
   words and logs *clarify answered*.
3. **Critique.** The session starts a fresh hard critic with the note, the
   summary and the spec paths. The critic reads and returns findings as blocks,
   should or note. The session hands them, as written, to the same architect
   agent, which settles each one and writes `## Critique`.
4. **Feedback + continue.** The architect writes `## Feedback`, the summary's
   *How* and *Decisions*, the spec's *Plan* and the ADR as proposed, and returns.
   The session verifies the feedback, moves the card to Developer, bumps, detects
   the next actor from the files and starts a fresh developer subagent. It does
   not stop for `/compact` or another `/spec PRJ-26`.

In `session` mode the same four steps happen with the session following the
architect's brief itself; the questions go to the owner directly, and the critic
is still a fresh agent. After Feedback it stops for `/compact`, then `/spec PRJ-26`.
A direct `/spec-architect PRJ-26` runs only that actor in either mode; automatic
continuation belongs to the `/spec` manager loop.

## Where the owner decides

In an actor's clarify (scope at the analyst, decisions and who builds at the
architect, paid steps at qa); when the build cycle is stuck after three red
rounds; and at the post-mortem, whose lessons are proposed and executed only on
a yes.

## Mechanics

`SKILL.md` is the shape; `scripts/flow.sh` is the whole shell, sourced by every
command (`resolve_spec`, `bump`, `set_section`, `next_actor`, `board_sync`);
`references/` holds the formats, the critic, the build cycle, the post-mortem and
the traps; `INSTALL.md` reproduces it on a new machine. Facts about a repository
live in one place, the project hook in `docs/specs/README.md`, which points at
files and restates no rule.

