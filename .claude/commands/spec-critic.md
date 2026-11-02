---
description: Run the critic alone on the current actor's note — a fresh agent at hard or balanced — and settle its findings; for a decision that changed, or a critique worth a second pair of eyes.
argument-hint: "[ID] [hard|balanced]"
---

# /spec-critic `[ID] [hard|balanced]`

Runs the **critique** step of the loop on purpose, on the note of the actor the
spec is with. Source `scripts/flow.sh` from **`spec-flow`** (the preamble and every snippet); the two critic
briefs and how findings are settled are in `references/critic.md`.

## Steps
1. Source the script → `hook` → `resolve_spec $ID`. The actor is `step` in the spec;
   the note is `note "$STEP"`. The level is the argument, else the note's `critic:`;
   `none` (qa) has no critic: say so and stop.
2. **Start a fresh read-only agent** with the brief for that level from
   `spec-flow`, on the model its table gives (the reviewer: `opus`, `high`), given
   `vault_read` from `flow.sh`, which falls back to the note's file under
   `$VAULT_DIR`, or the notes' paths on disk, read-only; the vault paths of the note
   and the summary; the spec's path; and the hook's paths for plans, decision
   records and guardrails. Nothing else: not
   the session's reasoning, not which option it prefers, not a summary.
3. Settle every finding in the session as the loop says: a changed answer goes back
   to the user; a correct finding fixes the work; a risk is the user's to accept; a
   wrong one is refused with the reason.
4. Append the findings to the note's `## Critique` under a dated line
   (`- re-run $NOW, $LEVEL:`), so the first critique stays readable. While a `blocks`
   is open: `bump $STEP blocked "<what is open, and on whom>"`.
5. Say in the reply what was found and what changed in the summary or the spec.
