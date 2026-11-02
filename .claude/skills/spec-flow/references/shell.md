# The shell contract — `scripts/flow.sh`

Source it at the top of every Bash block that touches the flow; shell state does
not persist between tool calls:

```bash
FLOW=$(git rev-parse --show-toplevel 2>/dev/null)/.claude/skills/spec-flow/scripts/flow.sh
[ -f "$FLOW" ] || FLOW=~/.claude/skills/spec-flow/scripts/flow.sh; . "$FLOW"
```

Sourcing runs the preamble: `REPO`, `REPO_NAME`, `SPEC_ROOT` (`SPEC_FLOW_ROOT`
overrides), `VAULT` (`SPEC_VAULT` overrides, default `Manager`), `BOARD`,
`BRANCH`, `BTAG`, `NOW`, `TODAY`, `VAULT_DIR` (`SPEC_VAULT_DIR`; the vault's folder on disk, empty by default). Then the functions:

| Function | Does | Sets |
|---|---|---|
| `obs <cli args>` | the Obsidian CLI with the banner stripped and a 15 s alarm | — |
| `vault_read <path> <marker>` | a note's content, read up to five times, 1.5 s apart, until it holds the marker (an ERE); then the file in `VAULT_DIR` if it holds the marker; else names the path and returns 1 | — |
| `vault_write <path> <file> <marker>` | overwrites a note with the file's content; refuses content without the marker, and reads it back | — |
| `spec_check` | Obsidian open, the vault there, the board scaffolded; else says what to do | — |
| `hook` | prints the project hook from `docs/specs/README.md` | — |
| `resolve_spec [ID]` | the spec by ID, or the open one on this branch; opens the ticket folder for a spec that names none (no `vault:` line) and whose index does not read | `ID TITLE NAME SPEC EXTRA TDIR IDX MODE` |
| `note <actor>` | the actor's note path in the vault | — |
| `open_note <actor> <hard\|balanced\|none> <title>` | the note, if it does not exist | — |
| `set_section <note> <Section> <file>` | replaces one section of a note with the file's body; refuses a body with nothing but blanks | — |
| `section_line <note> <Section> <text>` | appends `- <date> — <text>` at the end of one section, the date from the clock when it is written | — |
| `note_log <note> <line>` | appends a dated line to the note's `## Log` | — |
| `move_card <Column> [x]` | the card under that column, blank runs squeezed | — |
| `bump <step> <status> <line>` | the spec's and the index's frontmatter, a log line in both, *Where it stands* in the summary | — |
| `set_mode <session\|subagents>` | the spec's `mode:` | `MODE` |
| `finished <actor>` | true when the last of the note's *work started*, *work written* and *feedback written* log lines is *feedback written*, read through `vault_read` | — |
| `after <actor>` | the next actor, `close` after qa | — |
| `next_actor` | which actor `/spec` runs, from the files | `NEXT` |
| `board_init` | the seven columns, never over cards | — |
| `board_sync` | this repository's cards rebuilt from its spec files | — |
| `vault_open` | the index and an empty summary for a spec without a folder | — |

## What to know

- **A section body comes from a file**, never from an argument the shell has
  expanded: `cat > "$t" <<'EOF' … EOF`, then `set_section "$(note architect)" Work "$t"`.
  The body may hold `$`, quotes and fences; a line that is exactly `## <name>`
  ends the section, so a fence never holds one.
- **The CLI turns a literal `\n` or `\t` inside `content=` into the character**,
  with no escape. A note never holds code with them: cite `path:line`. The spec
  is written by the shell and has no such limit.
- `create` without `overwrite` refuses an existing note; `append` refuses a
  missing one; multi-line `append` works.
- **A vault read sometimes comes back empty for a minute**: a vault synced
  through iCloud or the like, or the CLI under load, answers nothing. `set_section`,
  `note_log`, `bump`, `move_card` and `board_sync` go through `vault_read` and
  `vault_write`, so an empty read is tried again and never written back over
  the note. One that returns 1 wrote nothing: run it again. `bump` reads the
  index and the summary before it touches the spec. The CLI has read an 87 KB
  note as nothing on every try: with `SPEC_VAULT_DIR` set, reads fall back to
  the file in `VAULT_DIR`, writes stay the CLI's. `scripts/flow.test.sh`
  checks this over a directory, with no Obsidian.
- The Bash tool is zsh on macOS: `status` is read-only, a `case` inside `$( )`
  does not parse, an unmatched glob fails the command, and `echo ====` fails: a
  word starting with `=` is a command lookup. Quote a tool's glob:
  `--include='*.rs'`. The script globs nothing.
- A command handed to the owner runs in the owner's shell, not the tool's: one
  line, with no heredoc, no `for … do` loop and no bare `VAR=value`, which fish
  refuses. Fish cannot source `flow.sh`, so its functions are handed
  over as `zsh -c '. <path>/flow.sh && <function>'`.
- Never `sed -i`: BSD and GNU disagree about its argument. Write a temporary
  file and `mv`.
- `bump` rewrites the spec's frontmatter with `awk` and only there: a spec quotes
  YAML often enough that a `sed` over the whole file would rewrite an example.

