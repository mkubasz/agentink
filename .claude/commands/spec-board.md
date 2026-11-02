---
description: Manage the board in the vault — init scaffolds Boards/Pipeline.md and the repository's project hook, sync rebuilds this repository's cards from its spec files, show prints the board.
argument-hint: "[init|sync|show]"
---

# /spec-board `[init|sync|show]`

Manages the one board, `Boards/Pipeline.md` in the vault, which tracks every spec
of every repository that uses the flow. Load **`spec-flow`** for the layout and the
card format. Sub-command from the argument; default `show`.

Start by sourcing `scripts/flow.sh` from **`spec-flow`**; `init` skips `spec_check`.

## init — scaffold the board, the folder and the project hook

Idempotent, and it **never** overwrites a board that already has cards.

1. If `obs read path=$BOARD` holds no `## Backlog`, write the seven columns in the
   Kanban plugin's format:

   ```bash
   obs create path=$BOARD overwrite content="$(cat <<'EOF'
   ---
   kanban-plugin: basic
   ---

   ## Backlog

   ## Analyst

   ## Researcher

   ## Architect

   ## Developer

   ## QA

   ## Done

   %% kanban:settings
   {"kanban-plugin":"basic"}
   %%
   EOF
   )"
   ```

2. `mkdir -p "$SPEC_ROOT"`.
3. If `$SPEC_ROOT/README.md` does not exist, write it: a short paragraph on what the
   folder is, then a `## Project hook` section holding the facts the flow cannot
   guess. Ask the user for anything the repository does not answer plainly:

   - **ID prefix** — two to five capitals, e.g. `PRJ`.
   - **Plans** — where the work is planned, or *none*.
   - **Decision record** — where a decision is written (`docs/ADR/` and its
     template), or *none*.
   - **Guardrails** — the files a critic reads (`CLAUDE.md`, `AGENT.md`,
     `CONTRIBUTING.md`).
   - **Verification** — the command that says a change is good, the command for one
     test by name, and the extra gates that apply to part of the tree.
   - **Gate places** — where the product runs, and which places the user starts
     because they cost money or are shared.
   - **Commit** — where the hooks and the message rule are.
   - **Contract regeneration** — the command that regenerates anything generated.
   - **Vault** — only if it is not `Manager`.

   The hook **points at** those files; it never restates them.
4. Report what it created and what was already there.

## sync — rebuild this repository's cards from its spec files

Authoritative repair for this repository's cards; cards of other repositories are
left where they are.

1. Enumerate: `grep -rlx '<!-- spec-card -->' --include='*.md' "$SPEC_ROOT"`.
2. For each, read `id`, `title`, `step`, `status`.
3. Bucket: **Done** when `status: done`, **Backlog** when `status: todo`, otherwise
   the column of the step (`qa` → `QA`).
4. Read the board, drop every card whose link names one of the enumerated specs,
   insert each card under its column, squeeze blank runs as `move_card` does, write
   the board back whole with `overwrite`.
5. Report what moved ("PRJ-12 Architect→Developer, PRJ-7 QA→Done"), not the board.

## show — print it

`obs read path=$BOARD`, then summarise: which specs sit where, and which are
`blocked` — for those, the spec's last log line says on whom and for what.
