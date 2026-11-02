# spec-flow — the shell contract. Source it at the top of every Bash block that
# touches the flow (shell state does not persist between tool calls):
#
#   FLOW=$(git rev-parse --show-toplevel 2>/dev/null)/.claude/skills/spec-flow/scripts/flow.sh
#   [ -f "$FLOW" ] || FLOW=~/.claude/skills/spec-flow/scripts/flow.sh; . "$FLOW"
#
# Sourcing runs the preamble; the functions are the contract. bash and zsh.
# zsh traps: `status` is read-only, a `case` inside $( ) does not parse, an
# unmatched glob fails the command — so nothing here globs, and nothing is
# written with `sed -i` (BSD and GNU disagree about its argument).

REPO=$(git rev-parse --show-toplevel 2>/dev/null || pwd); REPO_NAME=$(basename "$REPO")
SPEC_ROOT="${SPEC_FLOW_ROOT:-$REPO/docs/specs}"
VAULT="${SPEC_VAULT:-Manager}"; BOARD=Boards/Pipeline.md
VAULT_DIR="${SPEC_VAULT_DIR:-}"                 # the vault's folder on disk, read when the CLI is not; empty: no fallback
OBS_BIN="$(command -v obsidian || echo /Applications/Obsidian.app/Contents/MacOS/obsidian)"
BRANCH=$(git -C "$REPO" branch --show-current 2>/dev/null || echo '-')
BTAG=$(printf '%s' "$BRANCH" | tr '/' '-')       # feature/x -> feature-x
NOW=$(date '+%Y-%m-%d %H:%M'); TODAY=$(date '+%Y-%m-%d')

# --- the vault -----------------------------------------------------------------

# The CLI prints a banner on stdout and hangs rather than fails when Obsidian is
# closed, hence the grep and the alarm. It turns a literal \n or \t in content=
# into the character, with no escape: a note never holds code with them.
obs() { perl -e 'alarm 15; exec @ARGV' -- "$OBS_BIN" vault="$VAULT" "$@" 2>/dev/null \
  | grep -vaE 'Loading updated app package|Your Obsidian installer is out of date'; }

# A read sometimes comes back empty for a minute (a vault synced through iCloud or
# the like, or the CLI under load, answers nothing), and a writer that overwrites with what it
# read would empty the note. So a read is tried until it holds its marker, and a
# write is refused without the marker and read back. A note the CLI keeps reading as
# nothing (an 87 KB note was) is read from the file on disk; writes stay the CLI's.
vault_read() {                                   # $1 = vault path, $2 = marker (ERE) the content must hold -> the content
  # An empty path is a `"$(note …)"` without a ticket folder: every writer reads first, so none reaches the vault's root.
  [ -n "$1" ] || { echo "vault read of an empty path refused" >&2; return 1; }
  local c n=0
  while :; do
    c=$(obs read path="$1"); printf '%s\n' "$c" | grep -qE -- "$2" && break
    n=$((n+1))
    if [ "$n" -ge 5 ]; then
      [ -n "$VAULT_DIR" ] && [ -f "$VAULT_DIR/$1" ] && grep -qE -- "$2" "$VAULT_DIR/$1" && { cat "$VAULT_DIR/$1"; return 0; }
      echo "vault read of $1 lacks '$2' after $n tries" >&2; return 1
    fi
    perl -e 'select(undef,undef,undef,1.5)'
  done
  printf '%s\n' "$c"
}

vault_write() {                                  # $1 = vault path, $2 = file with the new content, $3 = marker (ERE) it must hold
  [ -s "$2" ] && grep -qE -- "$3" "$2" || { echo "new content for $1 lacks '$3': nothing written" >&2; return 1; }
  obs create path="$1" content="$(cat "$2")" overwrite >/dev/null
  vault_read "$1" "$3" >/dev/null
}

spec_check() {                                   # Obsidian open, vault there, board scaffolded
  obs read path=$BOARD | grep -q '^## Backlog' && return 0
  echo "Obsidian is not running, or vault $VAULT has no board: open Obsidian, then /spec-board init"; return 1
}

hook() { sed -n '/^## Project hook/,$p' "$SPEC_ROOT/README.md"; }

note() {                                         # $1 = actor -> the note's vault path; none without a ticket folder
  [ -n "$TDIR" ] || { echo "no ticket folder: TDIR is empty, so no note path for $1 (resolve_spec first)" >&2; return 1; }
  local n; case $1 in analyst) n=01-analyst;; researcher) n=02-researcher;; architect) n=03-architect;;
    developer) n=04-developer;; qa) n=05-qa;; esac; printf '%s/%s.md' "$TDIR" "$n"; }

note_log() {                                     # $1 = note, $2 = work written | clarify answered | critique settled | feedback written
  vault_read "$1" '^## Log$' >/dev/null || return 1
  obs append path="$1" content="- $NOW — $2" >/dev/null
}

set_section() {                                  # $1 = note path, $2 = section, $3 = file holding the new body
  grep -q '[^[:space:]]' "$3" 2>/dev/null || { echo "body for $1 › $2 is missing or blank: nothing written" >&2; return 1; }
  local t c rc; c=$(vault_read "$1" "^## $2\$") || return 1; t=$(mktemp)
  printf '%s\n' "$c" | awk -v h="## $2" -v f="$3" '
    $0==h {print; print ""; while ((getline l < f) > 0) print l; print ""; skip=1; next}
    /^## / {skip=0}  !skip' > "$t"
  vault_write "$1" "$t" "^## $2\$"; rc=$?; rm -f "$t"; return $rc
}

# A dated line at the end of one section — a build round under `## Critique`. The time
# is the clock's when it is written, never typed: rounds logged from memory drift.
section_line() {                                 # $1 = note path, $2 = section, $3 = the line's text
  local t c rc; c=$(vault_read "$1" "^## $2\$") || return 1; t=$(mktemp)
  printf '%s\n' "$c" | H="## $2" L="- $(date '+%Y-%m-%d %H:%M') — $3" awk '
    function add() { if (!items) print ""; print ENVIRON["L"] }
    $0 == ENVIRON["H"] { print; ins = 1; items = 0; blanks = 0; next }
    ins && /^## / { add(); print ""; ins = 0; print; next }
    ins && /^$/ { blanks++; next }
    ins { while (blanks > 0) { print ""; blanks-- } print; items++; next }
    { print }
    END { if (ins) add() }' > "$t"
  vault_write "$1" "$t" "^## $2\$"; rc=$?; rm -f "$t"; return $rc
}

open_note() {                                    # $1 = actor, $2 = hard|balanced|none, $3 = title
  local p c; p=$(note "$1") || return 1; obs read path="$p" | grep -q '^actor: ' && return 0
  c='## Critique\n\n'; [ "$2" = none ] && c=''  # qa has no critic
  obs create path="$p" content="$(printf -- '---\nticket: %s\nactor: %s\ncritic: %s\n---\n`#spec/%s` · `#actor/%s`\n\n# %s\n\n## Work\n\n## Clarify\n\n'"$c"'## Feedback\n\n## Log\n- %s — opened' "$ID" "$1" "$2" "$ID" "$1" "$3" "$NOW")" >/dev/null
}

vault_open() {                                   # the ticket folder for a spec that has none
  local step st; step=$(sed -n 's/^step: *//p' "$SPEC" | head -1); st=$(sed -n 's/^status: *//p' "$SPEC" | head -1)
  obs create path="$IDX" content="$(cat <<EOF2
---
id: $ID
title: $TITLE
step: $step
status: $st
mode: $MODE
branch: $BRANCH
repo: $REPO_NAME
spec: ${SPEC#"$REPO"/}
created: $TODAY
updated: $TODAY
tags: [spec/$ID, step/$step, branch/$BTAG, status/$st]
---
<!-- spec-card -->
\`#spec/$ID\` · \`#step/$step\` · \`#branch/$BTAG\` · repo \`$REPO_NAME\`

# $ID — $TITLE

- [[summary]] · [[01-analyst]] · [[02-researcher]] · [[03-architect]] · [[04-developer]] · [[05-qa]] · [[postmortem]]
- spec: \`${SPEC#"$REPO"/}\`

## Log
- $NOW — ticket folder opened for the spec
EOF2
)" | grep -q '^Created: ' || { echo "could not create $IDX in vault $VAULT"; return 1; }
  obs read path="$TDIR/summary.md" | grep -q '^# ' || obs create path="$TDIR/summary.md" content="$(printf -- '# %s — %s\n\n## Where it stands\n\n%s · %s — ticket folder opened (%s)\n\n## Why\n\n## What\n\n## How\n\n## Decisions\n' "$ID" "$TITLE" "$step" "$st" "$NOW")" >/dev/null
  grep -q '^vault: ' "$SPEC" || { awk -v v="vault: $TDIR" '/^---$/{fm++} fm==1 && /^repo: /{print; print v; next} {print}' "$SPEC" > "$SPEC.tmp" && mv "$SPEC.tmp" "$SPEC"; }
}

# --- the board and the frontmatter -------------------------------------------

squeeze() { awk '/^$/{blank=1; next} {if (NR>1 && (blank || /^## /)) print ""; blank=0; print}'; }

move_card() {                                    # $1 = column, $2 = " " or "x"
  local card="- [${2:- }] [[$TDIR/_index|$ID · $TITLE]]" t c rc
  c=$(vault_read "$BOARD" '^## Backlog$') || return 1; t=$(mktemp)
  printf '%s\n' "$c" | grep -vF "[[$TDIR/_index" \
    | awk -v c="$card" -v h="## $1" '$0==h{print; print ""; print c; next} {print}' | squeeze > "$t"
  vault_write "$BOARD" "$t" '^## Done$'; rc=$?; rm -f "$t"; return $rc
}

bump() {                                         # $1 = step, $2 = status, $3 = log line — the spec, the index, the summary
  local t i s='' rc                              # both reads first: a refused bump leaves the spec as it was
  i=$(vault_read "$IDX" '^(id|ticket): ') || return 1
  obs read path="$TDIR/summary.md" | grep -q '^Error: .*not found' \
    || { s=$(vault_read "$TDIR/summary.md" '^## Where it stands$') || return 1; }   # close deletes the summary
  awk -v s="$1" -v st="$2" -v d="$TODAY" -v t="[spec/$ID, step/$1, branch/$BTAG, status/$2]" '
    /^---$/ {fm++}
    fm==1 && /^step: /    {$0="step: " s}
    fm==1 && /^status: /  {$0="status: " st}
    fm==1 && /^updated: / {$0="updated: " d}
    fm==1 && /^tags: /    {$0="tags: " t}
    {print}' "$SPEC" > "$SPEC.tmp" && mv "$SPEC.tmp" "$SPEC"
  printf -- '- %s — %s\n' "$NOW" "$3" >> "$SPEC"
  t=$(mktemp)
  { printf '%s\n' "$i" \
    | sed -E "s/^step: .*/step: $1/; s/^status: .*/status: $2/; s#^updated: .*#updated: $TODAY#; s#^tags: .*#tags: [spec/$ID, step/$1, branch/$BTAG, status/$2]#; s#^\`\#spec/$ID\` · \`\#step/[a-z]*\`#\`\#spec/$ID\` · \`\#step/$1\`#"
    printf -- '- %s — %s\n' "$NOW" "$3"; } > "$t"
  vault_write "$IDX" "$t" '^(id|ticket): ' || { rm -f "$t"; return 1; }
  [ -z "$s" ] || { printf '%s\n' "$s" | awk -v l="$1 · $2 — $3 ($NOW)" \
      '/^## Where it stands$/{print; print ""; print l; print ""; skip=1; next} /^## /{skip=0} !skip' > "$t"
    vault_write "$TDIR/summary.md" "$t" '^## Where it stands$'; }
  rc=$?; rm -f "$t"; return $rc
}

set_mode() {                                     # $1 = session|subagents — kept in the spec's frontmatter
  awk -v m="$1" '/^---$/{fm++} fm==1 && /^mode: /{$0="mode: " m; seen=1} fm==1 && /^branch: / && !seen {print "mode: " m; seen=1} {print}' "$SPEC" > "$SPEC.tmp" && mv "$SPEC.tmp" "$SPEC"; MODE=$1
}

# --- resolve ------------------------------------------------------------------

resolve_spec() {                                 # $1 = ID, or empty for the open spec on this branch
  spec_check || return 1
  ID="$1"
  if [ -n "$ID" ]; then
    SPEC=$(find "$SPEC_ROOT" -maxdepth 1 -type f -name "$ID-*.md" ! -name '*.contract.md' ! -name '*.architecture.md' | head -1)
  else
    SPEC=$(grep -rlx '<!-- spec-card -->' --include='*.md' "$SPEC_ROOT" \
             | xargs grep -l "^branch: $BRANCH\$" | xargs grep -L '^status: done$' | head -2)
  fi
  [ -n "$SPEC" ] && [ -f "$SPEC" ] || { echo "No one spec for '${ID:-$BRANCH}': $SPEC"; return 1; }
  ID=$(sed -n 's/^id: *//p' "$SPEC" | head -1)
  TITLE=$(sed -n 's/^title: *//p' "$SPEC" | head -1)
  NAME=$(basename "$SPEC" .md)
  EXTRA=$(find "$SPEC_ROOT" -maxdepth 1 -type f -name "$NAME.*.md" | head -1)
  TDIR="Tickets/$NAME"; IDX="$TDIR/_index.md"
  MODE=$(sed -n 's/^mode: *//p' "$SPEC" | head -1); MODE=${MODE:-session}
  # An empty read is not a missing folder: a spec that names its folder never gets a second index.
  grep -q '^vault: ' "$SPEC" || vault_read "$IDX" '^(id|ticket): ' >/dev/null 2>&1 || vault_open
}

# The last of work started / work written / feedback written decides: a reopened round is not finished until its own feedback.
finished() { vault_read "$(note "$1")" '^## Log$' 2>/dev/null \
  | sed -nE 's/^- [0-9-]+ [0-9:]+ — (work started|work written|feedback written).*/\1/p' | tail -1 | grep -qx 'feedback written'; }   # $1 = actor

after() { case $1 in analyst) echo researcher;; researcher) echo architect;; architect) echo developer;; developer) echo qa;; qa) echo close;; esac; }

next_actor() {                                   # sets NEXT from the files, never from memory of the session
  local step st; step=$(sed -n 's/^step: *//p' "$SPEC" | head -1); st=$(sed -n 's/^status: *//p' "$SPEC" | head -1)
  if   [ "$st" = done ];    then NEXT=closed
  elif [ "$st" = blocked ]; then NEXT=$step      # waiting; the last log line says on whom
  elif [ "$st" = todo ];    then NEXT=analyst
  elif [ "$step" = done ];  then NEXT=close
  elif finished "$step";    then NEXT=$(after "$step")   # the note is finished, the bump was not
  else NEXT=$step; fi
  echo "$ID: $step/$st -> $NEXT"; grep '^- [0-9]' "$SPEC" | tail -1
}

# --- the board ------------------------------------------------------------------

col_of() { case $1 in analyst) echo Analyst;; researcher) echo Researcher;; architect) echo Architect;;
  developer) echo Developer;; qa) echo QA;; done) echo Done;; *) echo Backlog;; esac; }

board_init() {                                   # the seven columns; never over a board that has cards
  obs read path=$BOARD | grep -q '^- \[' && { echo "the board has cards; left as it is"; return 0; }
  obs create path=$BOARD overwrite content="$(printf -- '---\nkanban-plugin: basic\n---\n\n## Backlog\n\n## Analyst\n\n## Researcher\n\n## Architect\n\n## Developer\n\n## QA\n\n## Done\n\n%%%% kanban:settings\n{"kanban-plugin":"basic"}\n%%%%\n')" >/dev/null
  echo "board scaffolded in vault $VAULT"
}

board_sync() {                                   # rebuild this repository's cards from its spec files; other repositories' cards stay
  local t f id title step st name col box card; t=$(mktemp)
  vault_read "$BOARD" '^## Backlog$' > "$t" || { rm -f "$t"; return 1; }
  grep -rlx '<!-- spec-card -->' --include='*.md' "$SPEC_ROOT" | sort -V | while IFS= read -r f; do
    id=$(sed -n 's/^id: *//p' "$f" | head -1); title=$(sed -n 's/^title: *//p' "$f" | head -1)
    step=$(sed -n 's/^step: *//p' "$f" | head -1); st=$(sed -n 's/^status: *//p' "$f" | head -1)
    name=$(basename "$f" .md)
    case $st in done) col=Done; box=x;; todo) col=Backlog; box=' ';; *) col=$(col_of "$step"); box=' ';; esac
    card="- [$box] [[Tickets/$name/_index|$id · $title]]"
    grep -vF "[[Tickets/$name/_index" "$t" \
      | awk -v c="$card" -v h="## $col" '$0==h{print; print ""; print c; next} {print}' > "$t.n" && mv "$t.n" "$t"
  done
  squeeze < "$t" > "$t.n" && mv "$t.n" "$t"
  vault_write "$BOARD" "$t" '^## Done$' || { rm -f "$t"; return 1; }; rm -f "$t"
  obs read path=$BOARD | awk '/^## /{c=$2} /^- \[/{n[c]++} END{for (k in n) printf "%s %d · ", k, n[k]; print ""}'
}

