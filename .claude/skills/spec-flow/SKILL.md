---
name: spec-flow
type: reference
description: >
  The spec flow — five actors, analyst (why) → researcher (what) → architect (how)
  → developer → qa, each in its own note in the Obsidian "Manager" vault through
  work → clarify → critique → feedback, with the owner's summary beside them, while
  the record — one spec file with the plan and the log, and the decision record —
  stays in the repository under `docs/specs/`. Auto-load this whenever a /spec*
  command runs, or whenever the user talks about specs, the actors, the board, a
  ticket folder in the vault, the summary, the post-mortem, or spec-driven
  development. Holds the shape, the two modes, the loop, where the user decides,
  the command table, the project hook and the rules; the shell contract is
  scripts/flow.sh and the long form is under references/.
---

# Spec flow

`DESCRIPTION.md` beside this file says what the flow is and why, in a page;
`INSTALL.md` puts it on a machine. This file is the contract.

Every unit of work is **one spec**, and five **actors** take it in turn:

> **analyst** (why) → **researcher** (what) → **architect** (how) → **developer** → **qa** → close

Each actor runs the same loop in its own note, and the loop is where the user
decides:

> **work → clarify → critique → feedback**

| Actor | Answers | Critic | Writes | Asks the user about |
|---|---|---|---|---|
| analyst | why: the problem, for whom, what done looks like, what is out | hard | summary `## Why`, `## What`; spec `## Summary` | scope, success, what to leave out |
| researcher | what is true now: plans, decisions, structure, what exists; no opinion | hard | summary `## What › Constraints` | nothing, usually |
| architect | how: decision points, the ADR as proposed, the plan, the gate rows | hard | summary `## How`, `## Decisions`; spec `## Plan`; the ADR | **every decision point**, the plan, who builds |
| developer | the plan's tasks, each with its proof; the build cycle; the commit | the reviewer (balanced) | ticks in the spec's `## Plan` | a decision nobody took |
| qa | the change where the product runs | — | results in the spec's gate table | **paid or shared gate steps** |

The analyst, the researcher and the architect **build the ADR** between them: the
analyst's *why* and the researcher's constraints are its Context; the architect
writes the Decision, the Alternatives and the Consequences and files it as
*proposed*. Close accepts it.

## Two modes

Chosen per spec and kept in its frontmatter as `mode:`.

| Mode | Who plays the actor |
|---|---|
| `session` (default) | the session itself, from the actor's brief; it asks the user directly, and the compact after each actor is what clears it |
| `subagents` (`/spec <ID> subagents`) | a subagent per actor; the session is the manager — it opens the note, starts the actor with its brief, carries questions to the user and answers back, starts the critic, hands the findings to the actor, bumps and starts the next actor, with no stop for a compact. The **developer defaults to two subagents** in worktrees when the tasks split into slices with disjoint files, **qa to two** when the gate rows split by place |

In both modes **the brief is the actor's command file** (`/spec-<actor>`), handed
whole with the paths — never the session's reasoning, never a summary of what the
predecessors wrote: the actor reads their `## Feedback`. The critic is always a
fresh agent (`references/critic.md`): *hard* for the three who think; the
developer's is the **reviewer**, balanced, the build cycle's review step; **qa has
none** — its rows are what a command printed, and the reviewer already read the code.

**Who runs on what.** The session passes these to every agent it starts (`model`,
`effort`); in `session` mode the actor is the session, so the stop line names the
model the next actor wants and the owner switches with `/model` before `/spec`.

| Agent | Model | Effort |
|---|---|---|
| analyst, researcher, architect, the hard critic, the post-mortem | the session's | the session's |
| developer and its agents, the reviewer, qa and its agents | `opus` | `high` |

## Where the user decides

**Inside the loop, at clarify, and nowhere else.** An actor asks only what the
files could not settle, in one round, a recommended answer first; an actor with
nothing to ask says so in a line and goes on. The flow stops at a question, at a
gate step the user has to start, and, in `session` mode, **after every actor's
feedback so the context can be compacted**: the notes, the summary and the spec are
the memory, and the next actor reads them, not the conversation. In `subagents`
mode each actor is a fresh agent, so the next one starts at once. One post-mortem per spec, at close,
proposes what the repository should learn; the user agrees, then it acts.

## Where the flow enters

`/spec` reads the brief and names the **kind** in its first line; a word in the
arguments overrides it (`/spec fix "…"`, `/spec continue PRJ-26`).

| Kind | It is | Enters at | What the skipped actors leave |
|---|---|---|---|
| `change` (default) | anything with a decision in it, or whose cause is not known — a bug with no known cause has the researcher find it by three blind finders (`/spec-researcher`) | analyst | — |
| `fix` | a wrong behaviour with a known cause and one sensible fix: a failing test, an error with its line, a typo | developer, with the plan written by `/spec-new fix`: the symptom as `## Summary`, one task proved by the test that fails today, one gate row where the bug shows; no ADR, no plan item | the summary's *Why* is the symptom, *How* the cause |
| `continue` | more of a spec that exists | where that spec stands (`next_actor`); a closed one gets a new spec whose *Why* and *What* come from the old record and enters at the researcher | — |

A fix that turns out to need a decision is not a fix: the developer says so,
`bump architect doing "not a fix: <why>"`, and the flow goes on from the architect,
who reads the developer's note. The build cycle, qa, close and the post-mortem run
for every kind.

## Two places

| | The vault — the workshop | The repository — the record |
|---|---|---|
| Where | Obsidian vault `Manager`, `Tickets/<ID>-<slug>/` | `docs/specs/<ID>-<slug>.md`, `docs/ADR/` |
| Holds | one note per actor; the index; **`summary.md`**, the owner's page — where it stands, why, what, how with a flow diagram, the decisions with their for and against; the post-mortem | **the spec, one file**: a line of summary, the plan with its gate while open, the log; at close the final record; the ADR; at most one extra file (`.contract.md` or `.architecture.md`) while building |
| Written through | the `obsidian` CLI only; Obsidian has to be open | the shell, committed by path |

The summary is temporary: close deletes it once the decisions are in the ADR and
what shipped is in the spec's final record. The ticket folder stays as history.
Formats: `references/formats.md`.

## The actor loop

A step writes its section with `set_section` and ends with its line in the note's
`## Log` — *work written*, *clarify answered*, *critique settled*, *feedback
written*. That log is the resume point: an actor started again continues at the
first step whose line is missing. A step run again replaces its section; the log
keeps every run's lines.

1. **Work.** Read the predecessors' `## Feedback`, the summary, then the files the
   question needs. `## Work` is the long form: every finding cited `path:line`,
   every option with what it costs; no citation → *Unknowns*.
2. **Clarify.** `## Clarify`: only what the files could not settle, each a question
   with two to four options, what each costs, a recommendation that cites the work,
   `**Answer:** —`. An option is one decision: a choice its costs leave open
   ("kept beside, or removed"), or a reading it names two ways, is a question of
   its own in the same round. Ask, all of it in one round, four to a call, the recommended
   option first, after `bump <actor> blocked "<n> questions waiting on the user"`.
   Write each answer in the user's own words, dated; `bump <actor> doing "answered"`.
   The analyst and the architect start the hard critic on `## Work` and the
   drafted questions before they ask; a finding that needs the owner joins the
   same round, and the critique after the answers reads only what an answer
   changed. The developer's questions wait for that round's review; an option
   the reviewer shows is costed wrong is restated before the owner sees it.
   Nothing to ask: one line, *nothing to ask: <why>*, no stop. An answer taken
   alone is `assumed — <reason>`, and the critic reads every *assumed* as open.
   When the previous actor's round was taken whole as recommended, this round's
   highest-stakes question carries the strongest honest case for the option not
   recommended, so a yes is a choice and not a habit. "You pick" is an answer
   written `assumed — user delegated, <date>`, open to the critic like any other.
3. **Critique.** Start the critic at the note's level and settle each finding in
   `## Critique`: it changes an answer the user gave → reopen and **ask again**,
   nobody overrules the user; right and changes no answer → fix the work; a risk
   worth taking → the user says so, recorded as `accepted risk: <their words>`;
   wrong → refused, as one of `not real: <the guard, caller, config or test that
   prevents it, path:line>`, `spec-conflict: D<n>` or `follow-up: <where it is
   logged>`. "Seems unlikely" refuses nothing; a `blocks` is refused only as *not
   real* or *spec-conflict*. Each refusal stays under `## Critique` with its kind
   and evidence, so the next critic reads it and does not raise it again. A
   `blocks` open or a question reopened → `bump <actor> blocked`. qa has
   no critic and no `## Critique`: its loop goes from clarify to feedback.
4. **Feedback.** `## Feedback`: what was produced (three lines), what the critic
   found and what became of it, what is handed on, what stays open. Fill the
   actor's summary and spec sections from it; one log line in the spec. Then
   `move_card <Next>`, `bump <next-actor> doing "<actor> done: <one line>"`.
5. **Stop and say so** (`session` mode). What was done, who is next, and
   `/compact <ID>: <next actor> next`, then `/spec <ID>`. Never start the next
   actor in the same context. In `subagents` mode say what was done in a line and
   start the next actor.

The developer's critique and qa's work are one **build cycle** — review → tests →
fixes in rounds, local until everything builds and runs, then CI, until the code
is reviewed clean, the proofs and the pre-commit pass and the product runs where
the plan says; three red rounds on the same item, or anything unfixable, go to the
user. `references/build-cycle.md`. The **post-mortem** at close judges the whole
run and proposes an `AGENT.md` line or a skill. `references/postmortem.md`.

## The shell contract

Every `/spec*` command starts by sourcing the script; shell state does not persist
between tool calls:

```bash
FLOW=$(git rev-parse --show-toplevel 2>/dev/null)/.claude/skills/spec-flow/scripts/flow.sh
[ -f "$FLOW" ] || FLOW=~/.claude/skills/spec-flow/scripts/flow.sh; . "$FLOW"
```

Then `hook`, `resolve_spec [ID]` (sets `ID TITLE NAME SPEC EXTRA TDIR IDX MODE`),
`open_note`, `set_section`, `note_log`, `move_card`, `bump`, `set_mode`,
`next_actor`, `board_init`, `board_sync`. The table and the traps — the CLI's
literal `\n`, zsh, `sed -i` — are in `references/shell.md`. A section body comes
from a file written with a quoted heredoc, never from an expanded argument.

## Actor → column → command

| Actor | Note | Column | Command |
|---|---|---|---|
| — | `_index.md`, `summary.md` | Backlog | `/spec-new [ID] <title> [subagents]` |
| analyst | `01-analyst.md` | Analyst | `/spec-analyst [ID]` |
| researcher | `02-researcher.md` | Researcher | `/spec-researcher [ID]` |
| architect | `03-architect.md` | Architect | `/spec-architect [ID]` |
| developer | `04-developer.md` | Developer | `/spec-developer [ID]` |
| qa | `05-qa.md` | QA | `/spec-qa [ID]` |
| close | `postmortem.md` | Done | `/spec-close [ID]`, then `/spec-postmortem [ID]` |
| any but qa | the current actor's note | — | `/spec-critic [ID] [hard\|balanced]` |

`/spec [ID | title] [session|subagents]` is the one to reach for: it finds the
spec, detects the actor from the files (`next_actor`), runs that actor to its
feedback and stops for the compact (`session`), or goes straight on to the next
actor (`subagents`); the last run closes. The actor commands are
for running one actor on purpose. `/spec-board [init|sync|show]` scaffolds,
rebuilds and prints the board; a column is the capitalised `step` (`qa` → `QA`),
`status: todo` is Backlog, `status: done` is Done.

## The project hook

`docs/specs/README.md` is written **once, by each repository**, and it is the only
place repo-specific facts live; `hook` prints it. It **points at** the repository's
own files and never restates them: the ID prefix; where the work is planned; the
decision record and its template; the guardrails; the verification and single-test
commands; the gate places and which of them the user starts; the commit hooks and
message rule; CI and which branch deploys; contract regeneration; the vault name if
not `Manager`. `/spec-board init` writes a starting one.

## Rules

- **One spec is one file in the repository and one ticket folder.** An actor
  writes its own note and its own summary and spec sections, nothing beside them.
- **Never rewrite a `## Log`.** Append. The log is what says when something was
  decided.
- **Work holds no answer the user did not give.** A choice taken alone is
  `assumed`, with its reason, and the critic reads it as open. Nor does the code:
  an `assumed` choice (a rider of an answer, or the fix a finding asks for when
  that fix is itself a choice) is asked before it is built, and the code stays as
  it was until the answer.
- **Ask once, ask everything.** A stop with no question in it is a stop people
  learn to skip.
- **A finding is outside the spec only when no commit of the spec made it
  reachable.** Before saying so, find the commit that opened the path to it (the
  route, the credential, the caller; `git log -S`), not only the blame of the line
  that fails. A finding the spec opened is fixed in it, or asked as its own.
- **Never fabricate a result.** A gate row records what a command printed, where
  it ran. A row that cannot pass is a question, never a deleted or softened row.
- **The gate is not optional and not last-minute.** The architect names the places
  before any code exists; `/spec-close` refuses a red gate.
- **At most one extra file, and it does not outlive the close.** The summary does
  not either.
- **The flow commits, and the build cycle pushes the branch it is on** to run CI,
  and nothing else; a push to a deploying branch, a tag, a merge or a pull request
  happens only when the user asks. A failing hook is fixed, never bypassed.
- **The hook names the commands; the actors run them.** No actor invents a test
  command from the file tree.
- **The vault is written through the CLI only.** A note edited on disk under a
  running Obsidian is a note Obsidian may overwrite.
