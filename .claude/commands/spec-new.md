---
description: Open a spec — the spec file under docs/specs/, the ticket folder with its summary in the vault, the card in Backlog.
argument-hint: "[fix] [ID] <title> [session|subagents]"
---

# /spec-new `[fix] [ID] <title> [session|subagents]`

Opens a spec in both places. Load **`spec-flow`** for the layout and
source its `scripts/flow.sh`. The ID is optional:
`/spec-new "Retry failed webhooks"` allocates the next free one from the
project hook's prefix.

## Steps

1. **Source the script, read the hook**: `. "$FLOW"` as `spec-flow` shows, then
   `spec_check` and `hook`. If `$SPEC_ROOT/README.md` is missing or there is no
   board, run `/spec-board init` first.

2. **Resolve the kind, the ID and the mode.** A leading `fix` is the kind (`KIND`,
   default `change`); the word `subagents` or `session` is the mode (`MODE`),
   default `session`. If the rest starts with something matching
   `^[A-Z]{2,5}-[0-9]+$`, that is the ID and the rest is the title. Otherwise allocate:

   ```bash
   PREFIX=$(sed -n 's/^- \*\*ID prefix:\*\* *//p' "$SPEC_ROOT/README.md" | head -1)
   PREFIX=${PREFIX:-$(printf '%s' "$REPO_NAME" | tr -cd '[:alnum:]' | cut -c1-2 | tr '[:lower:]' '[:upper:]')}
   N=$( { find "$SPEC_ROOT" -maxdepth 1 -name "$PREFIX-[0-9]*"; obs folders | grep "^Tickets/$PREFIX-[0-9]"; } \
         | sed -E "s#.*/$PREFIX-([0-9]+).*#\1#" | sort -n | tail -1)   # the vault sees every branch; the checkout sees one
   ID="$PREFIX-$(( ${N:-0} + 1 ))"
   TITLE="Retry failed webhooks"          # <- from the command args
   SLUG=$(printf '%s' "$TITLE" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-' | sed 's/^-//;s/-$//')
   NAME="$ID-$SLUG"; SPEC="$SPEC_ROOT/$NAME.md"; TDIR="Tickets/$NAME"; IDX="$TDIR/_index.md"
   ```

3. **Guard against duplicates** — never overwrite a spec:

   ```bash
   { find "$SPEC_ROOT" -maxdepth 1 -name "$ID-*"; obs folders | grep "^Tickets/$ID-"; } | grep . \
     && { echo "$ID exists"; exit 1; }
   ```

4. **Write the spec** (the `<!-- spec-card -->` line is what board sync enumerates):

   ```bash
   cat > "$SPEC" <<EOF
   ---
   id: $ID
   title: $TITLE
   step: analyst
   status: todo
   mode: ${MODE:-session}
   branch: $BRANCH
   repo: $REPO_NAME
   vault: $TDIR
   created: $TODAY
   updated: $TODAY
   tags: [spec/$ID, step/analyst, branch/$BTAG, status/todo]
   ---
   <!-- spec-card -->

   # $ID — $TITLE

   ## Summary
   <what the user said, in their words — the analyst makes it one or two lines>

   ## Plan

   ## Log
   - $NOW — spec opened on \`$BRANCH\`
   EOF
   ```

5. **Open the ticket folder** with `vault_open` from `spec-flow` — it writes the
   index and an empty `summary.md` — then put the user's words under the summary's
   `## Why` with `set_section`, and the card: `move_card Backlog`.

   **A `fix`** skips the three who think. Write the plan now, from the brief:
   `## Summary` is the symptom in one line; `## Plan` has `**Delivers:** none ·
   **Built by:** one implementer · **Extra file:** none · **ADR:** none`, one task
   `- [ ] 1. fix: <symptom> — proved by \`<the test that fails today, or the one to
   write>\``, and one gate row at the place where the bug shows. In the summary,
   `## Why` is the symptom, `## How` the cause as far as it is known, `## What` the
   fix in a line. Then `move_card Developer` and `bump developer doing "fix: entered
   at the developer"`.

6. **Confirm**: the spec, the folder, the card, the kind, and the next step —
   `/spec $ID` (the analyst, or the developer for a fix). Mention the branch it recorded; if the
   user is on `main`, say so, because the flow records the branch and does not
   create one.
