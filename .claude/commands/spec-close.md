---
description: Close — accept the ADR, update the plan item, delete the extra file and the summary, cut the spec to its final record, commit, move the card to Done, then the one post-mortem of the whole run.
argument-hint: "[ID]"
---

# /spec-close `[ID]`

The last step, run in the session. Source `scripts/flow.sh` from **`spec-flow`** (the preamble and every snippet).

## What this leaves
Three things that last, and nothing else: the **ADR**, accepted; the **plan item**,
saying where the work now stands; and the spec cut to its **final record**. The
summary in the vault is deleted; the actor notes and the post-mortem stay.

## Steps
1. Source the script → `hook` → `resolve_spec $ID`. **Refuse** unless every row of the
   spec's gate table is ✅ or ⏭. ⏭ is a row whose place the owner starts (a
   deployed instance, a paid GPU), left unrun because the owner closes without it: its Result reads
   `⏭ not run — <the owner's words>, <date>`, and `## Open` and the plan item name
   the row that runs it next. A ❌ row passes too when its Result ends `accepted risk:
   <the owner's words>, <date>` from a clarify, and `## Open` names the spec that takes
   it. Say which row is none of these.
2. **Keep the working spec in history.** If the spec has changes since its last
   commit, commit it as it is. Take that sha:
   `NOTES=$(git -C "$REPO" log -1 --format=%h -- "$SPEC")`.
3. **ADR:** *proposed* → *accepted*, dated, brought in line with what was built — a
   consequence the build or the gate found goes in now. The summary's flow diagram
   goes into the ADR's *Decision* if the ADR has none and the process is what was
   decided. A condition the ADR's own Status names and no gate row holds — a deploy —
   does not hold the close: accept on the gate and say in the Status what accepted it;
   a deploy the plan wants is a row on the deployed instance, set at the architect.
4. **Plan item:** in the plan the hook names, what is delivered, the date, a link to
   the spec, what stays open.
5. **Extra file:** move what lasts out of it — a boundary into the ADR or the area's
   own guide — and delete it. A contract's scenarios last as the tests that check them.
6. **Cut the spec.** Keep the frontmatter and the marker; `## Summary` grows to
   three lines from the summary's *Why* and *What*; `## Plan` becomes:

   ```markdown
   ## Shipped
   - `<sha>` <subject>
   - <what is live, and where>
   - CI: pipeline <id> <result>, holds `<last sha>` · <newer id> running · pushed <by whom, when>

   | Behaviour | Where | Command | Result |
   |---|---|---|---|
   | <…> | <…> | `<…>` | ✅ <what it showed> |

   ## Decisions
   - D1 <the question> → <the answer, in a line> · [<ADR>](<path>)

   ## Open
   - <what was left out or deferred> → <the plan item or spec that holds it>

   Working notes: `git show <NOTES>:<path of this spec>` · vault `Tickets/<name>/`
   ```

   About forty lines. The gate table is copied as it stands, results included.
   Under `## Shipped`, name the CI pipeline: the newest finished one whose commit
   holds the spec's last commit (`git merge-base --is-ancestor <last> <pipeline sha>`),
   with its id and result, a newer one still running by its id, and the push that
   carried the commit. Every product fault qa wrote under *seen on the way* goes
   under `## Open` with the plan item that holds it, or is named there as dropped.
   An `## Open` item handed to another open spec is also appended, dated, to that
   spec's `## Log` as `from <ID>: <item>`.
7. **Delete the summary:** `obs delete path="$TDIR/summary.md"` — the decisions are
   in the ADR and the record, the diagram in the ADR; nothing else in it lasts.
8. `move_card Done x`, then `bump done done "closed"`.
9. **Commit** the close by path — the spec, the deleted extra file, the ADR, the
   plan — with the hooks on. Before it, `git worktree list`: remove each worktree of
   the spec whose HEAD the spec branch holds, after showing the owner its uncommitted
   files, and the scratch files the spec left in the tree.
10. **Post-mortem**, the one of the whole run: `/spec-postmortem $ID`. Put its
    proposed actions to the owner and execute the agreed ones, as that command says.
    Name in the reply what graduated, what stays open, and which actions were taken.

