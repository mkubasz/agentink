---
description: The one command — find the spec, work out which actor it is with, run that actor to its feedback (in the session, or as a subagent with the subagents mode), and stop so the context can be compacted (session) or go on to the next actor (subagents); the last run closes the spec and runs the post-mortem.
argument-hint: "[fix|continue] [ID | title] [session|subagents]"
---

# /spec `[fix|continue] [ID | title] [session|subagents]`

Finds the spec and detects which actor it is with. In `session` mode it runs
that actor and stops; in `subagents` mode the session manages fresh actor subagents
and continues until a question, required approval or owner-started gate, unresolved
blocker, or close. The last run closes the spec and runs the one post-mortem.
Source `scripts/flow.sh` from **`spec-flow`** (the preamble and every snippet)
and follow the actor loop. The actor commands stay for running one actor on
purpose; `/spec-critic` for a critic alone.

## Steps

1. **Source the script, read the hook.** `. "$FLOW"`, then `hook`.
2. **Find the spec.**
   - An ID: resolve it. An ID that resolves to no spec but names an item in the hook's plans (`PL-01`)
     is a brief: `/spec-new`, kind `change`, with the item's title, and the item as
     the Plan's *Delivers*.
   - No argument: the open spec on this branch. If there are several, list them and
     ask which. Listing changes nothing:

     ```bash
     grep -rlx '<!-- spec-card -->' --include='*.md' "$SPEC_ROOT" \
       | xargs grep -l "^branch: $BRANCH\$" | xargs grep -L '^status: done$' | while IFS= read -r f; do
       printf '%s  %s/%s  %s\n' "$(sed -n 's/^id: *//p' "$f" | head -1)" \
         "$(sed -n 's/^step: *//p' "$f" | head -1)" "$(sed -n 's/^status: *//p' "$f" | head -1)" \
         "$(grep '^- [0-9]' "$f" | tail -1 | cut -c1-100)"
     done
     ```

   - Words that are not an ID: a brief. **Classify it** as `spec-flow`'s *Where the
     flow enters* says — `fix` when it names a wrong behaviour with a known cause
     and one sensible fix, `continue` when it names an existing spec, `change`
     otherwise — unless a leading `fix` or `continue` says so. Say the kind in the
     first line. Then `/spec-new [fix] <title>` and carry on; `continue` of a closed
     spec is `/spec-new` with the old record's *Why* and *What* and `step:
     researcher`; `continue` of an open one is the resolve below.
   - The word `session` or `subagents` anywhere in the arguments sets the mode for
     this spec from now on: `set_mode subagents`. Without it, the spec's `mode:`
     holds, `session` by default.
3. **Detect the actor** from the files, never from memory of the session:

   ```bash
   next_actor   # flow.sh: done → closed, blocked → its step, todo → analyst, step done → close,
                # a note whose log says "feedback written" → the next actor, else the step
   ```

   It is the script's function, not a copy here: a command file's `$1` is replaced
   by the command's argument when it is rendered.

4. **Say what was found** in one line before doing anything: the spec, the actor,
   and what it is waiting for if it is waiting.
5. **Run that actor** as the loop in `spec-flow` says, in the spec's mode: the
   session's part of `/spec-$NEXT` (open the note, move the card, bump), then the
   brief — followed by the session itself in `session` mode, or handed to a
   subagent in `subagents` mode, on the model and effort `spec-flow`'s table gives
   the actor, with the questions carried to the user, the critic, the findings
   carried back — then the feedback and the bump. It resumes at the
   first step its note's log does not record as finished. In `session` mode,
   **one actor per run**: when its feedback is written, do not detect again. In
   `subagents` mode, verify the actor's Feedback, detect again and run the next
   actor, until the spec waits on the user (a question, required approval or gate
   step they start), has an unresolved blocker, or is closed. Do not ask for
   `/compact` or another `/spec` invocation merely because an actor finished.
   The session-only stop/compact lines in actor briefs do not override this
   manager loop; delegated actors return to the manager, which starts successors.
   Direct `/spec-<actor>` actions remain single-actor commands. `/spec-close` runs
   in the session.
6. **Close** runs in the session when `$NEXT` is `close`: `/spec-close $ID`, whose
   last step is the one post-mortem of the whole run.
7. **Stop and say so**, in two lines: what this run did and what it waits for, if
   anything; then, in `session` mode, `/compact $ID: $NEXT_ACTOR next`, then
   `/spec $ID` (in `subagents` mode the run stops only where step 5 says). In `session`
   mode, when `spec-flow`'s table gives the next actor a model other than the
   session's, say so: `/model` to `opus` at `high` first. The notes, the summary
   and the spec hold everything the next actor reads, so the compact loses nothing.
   A run that ends with the same actor it started with, nothing blocked and no
   feedback written, means the actor did not leave what it should have: say what is
   missing instead of running it again.

## A blocked spec
`blocked` is not an actor to redo. Read the last log line. If it waits on an answer
the user has now given, write the answer where the note's `## Clarify` wants it,
`note_log` *clarify answered*, `bump $STEP doing "answered"` and carry on at
critique. In `subagents` mode resume the delegated actor from its note's log, then
continue the manager loop after Feedback; do not add a compact or handoff stop.
If it waits on a gate step the user had to run, write the user's result into the
row and carry on at qa's critique. Otherwise put the same question again
and stop.

