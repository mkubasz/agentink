# Installing the spec flow on a machine

Everything the flow needs, in the order to do it. Done once per machine; a
repository that ships the flow in its own `.claude/` needs only steps 1, 2, 5 and 6.

## 1. Claude Code

```bash
curl -fsSL https://claude.ai/install.sh | bash     # or: brew install --cask claude-code
claude --version                                   # 2.1 or later
claude                                             # log in once
```

## 2. Obsidian with its CLI and the `Manager` vault

1. Install Obsidian **1.14 or later** from <https://obsidian.md/download> and open
   it. The CLI talks to the running app; it does not launch it.
2. *Settings → General → Command line interface*: enable it.
3. Create a vault named **`Manager`** (*Open another vault → Create new vault*).
   Any name works with `SPEC_VAULT=<name>` in the environment, but `Manager` is
   the default every command assumes.
4. Put the binary on `PATH` so `command -v obsidian` finds it. macOS falls back to
   the app path on its own; Linux and Windows do not.

   ```bash
   mkdir -p ~/.local/bin
   ln -sf "/Applications/Obsidian.app/Contents/MacOS/obsidian" ~/.local/bin/obsidian   # macOS
   # Linux: ln -sf /usr/bin/obsidian ~/.local/bin/obsidian, or the AppImage / flatpak wrapper
   obsidian version                                # 1.14.x (installer 1.14.x)
   obsidian vaults                                 # lists Manager
   ```

5. Optional, when the vault syncs (iCloud, Syncthing, Obsidian Sync): `export
   SPEC_VAULT_DIR=<the vault's folder on disk>`. A synced vault sometimes reads as
   nothing through the CLI; with this set, reads fall back to the file on disk.
6. Optional, for the drag-and-drop view of the board: *Settings → Community
   plugins → Browse → Kanban* (mgmeyers). The board file is Kanban-compatible
   either way.

## 3. The flow itself

The gist is flat; the skill is not. Clone it and put each file where Claude Code
loads it from. The blocks on this page are bash and zsh: in fish, start `zsh`
first, since fish refuses a bare `D=…`.

```bash
git clone https://gist.github.com/1a70b623c89863c8b83e42d99252d33d.git spec-flow-gist
cd spec-flow-gist
D=~/.claude/skills/spec-flow                       # or <repo>/.claude/skills/spec-flow for one repository
C=~/.claude/commands                               # or <repo>/.claude/commands
mkdir -p "$D/scripts" "$D/references" "$C"
cp SKILL.md INSTALL.md DESCRIPTION.md "$D/"
cp flow.sh flow.test.sh "$D/scripts/"
cp build-cycle.md critic.md formats.md postmortem.md shell.md "$D/references/"
cp spec.md spec-*.md "$C/"
bash "$D/scripts/flow.test.sh"                     # flow.sh: all checks passed (no Obsidian needed)
```

Every command looks for `flow.sh` in the repository's `.claude/` first, then in
`~/.claude/`.

## 4. Tools the commands call

Standard on macOS and Linux: `git`, `perl` (the CLI timeout), `awk`, `sed`,
`find`, `grep`, `mktemp`. For the build cycle's CI tier: the forge's CLI the
project hook names (`gh`, `glab`, …), logged in. Everything else — the test
runner, the dev server, the e2e suite, the UI driver — is whatever the
repository's hook names.

```bash
for t in git perl awk sed find grep mktemp; do command -v $t >/dev/null && echo "ok  $t" || echo "MISSING $t"; done
```

## 5. The repository's side (once per repository)

Install the repository's own git hooks if it has any — the flow commits with the
hooks on — then, inside Claude Code in the repository:

```
/spec-board init
```

`init` scaffolds `Boards/Pipeline.md` in the vault with the seven columns, and —
for a repository that has none — writes `docs/specs/README.md` with the **project
hook**, asking you for what it cannot guess: ID prefix, plans, decision records,
guardrails, verification and single-test commands, gate places and which of them
you start, commit hooks, CI and the deploying branch, contract regeneration. The
hook points at files; it restates nothing.

## 6. Check

With Obsidian open, in a Bash block of a Claude Code session in the repository:

```bash
FLOW=$(git rev-parse --show-toplevel 2>/dev/null)/.claude/skills/spec-flow/scripts/flow.sh
[ -f "$FLOW" ] || FLOW=~/.claude/skills/spec-flow/scripts/flow.sh; . "$FLOW"
spec_check && echo "vault $VAULT, board ok" && hook | head -3 && board_sync
```

Expected: `vault Manager, board ok`, the hook's first lines, and a count of cards
per column. Then the real thing:

```
/spec "A first change"
```

It opens the spec file under `docs/specs/`, the ticket folder with `summary.md` in
the vault, the card in Backlog, and runs the analyst. In `session` mode, after
each actor it stops and tells you to `/compact`; `/spec <ID>` continues.
`/spec <ID> subagents` persists that mode: the manager delegates each actor to a
fresh subagent and continues without a compact until a question, required approval
or owner-started gate, unresolved blocker, or close. The whole process — actors,
modes, the loop, where you decide — is `SKILL.md`; the formats, the critic, the
build cycle and the post-mortem are under `references/`.

## When it does not work

| Symptom | Cause | Fix |
|---|---|---|
| every `obs` call prints nothing, then the block fails after 15 s | Obsidian is not running, or the CLI is off | open Obsidian; enable the CLI in Settings |
| `Obsidian is not running, or vault Manager has no board` | no vault of that name, or `init` not run | create the vault, `/spec-board init`; or `export SPEC_VAULT=<name>` |
| `command not found: obsidian` on Linux | not on `PATH` | the symlink in step 2 |
| `vault read of … lacks '…' after 5 tries` | a synced vault, or the CLI under load | run it again; set `SPEC_VAULT_DIR` (step 2.5) |
| `No one spec for 'main': …` | several open specs on the branch | pass the ID |
| a note shows a newline where code had `\n` | the CLI's limit | cite `path:line` instead of pasting such code |

