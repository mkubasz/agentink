# Self-check of flow.sh's guarded vault writers, over a directory standing in for
# the vault: `bash flow.test.sh` and `zsh flow.test.sh`. Needs no Obsidian. About
# fifteen seconds, most of it the two reads that never recover.

here=$(cd "$(dirname "$0")" && pwd); tmp=$(mktemp -d); FAKE="$tmp/vault"
mkdir -p "$tmp/specs" "$FAKE/Boards"
SPEC_FLOW_ROOT="$tmp/specs"; SPEC_VAULT_DIR="$tmp/no-vault-on-disk"; . "$here/flow.sh"

# read, create and append over $FAKE. A number in $FAKE/.empty is how many of the
# next reads come back empty, as the CLI's do under load.
obs() {
  local cmd=$1 p='' content='' a n; shift
  for a in "$@"; do case $a in path=*) p=${a#path=};; content=*) content=${a#content=};; esac; done
  case $cmd in
    read) n=$(cat "$FAKE/.empty" 2>/dev/null || echo 0)
      if [ "$n" -gt 0 ]; then echo $((n-1)) > "$FAKE/.empty"; return 0; fi
      if [ -f "$FAKE/$p" ]; then cat "$FAKE/$p"; else echo "Error: File \"$p\" not found."; fi ;;
    create) mkdir -p "$(dirname "$FAKE/$p")"; printf '%s\n' "$content" > "$FAKE/$p"; echo "Created: $p" ;;
    append) printf '%s\n' "$content" >> "$FAKE/$p" ;;
  esac
}
empty_reads() { echo "$1" > "$FAKE/.empty"; }
fails=0; check() { if eval "$2"; then echo "ok   $1"; else echo "FAIL $1"; fails=$((fails+1)); fi; }

printf -- '---\nkanban-plugin: basic\n---\n\n## Backlog\n\n## Developer\n\n## QA\n\n## Done\n' > "$FAKE/$BOARD"
SPEC="$tmp/specs/ZZ-1-a-test.md"
printf -- '---\nid: ZZ-1\ntitle: A test\nstep: developer\nstatus: doing\nbranch: main\nrepo: x\nupdated: 2000-01-01\ntags: []\n---\n<!-- spec-card -->\n\n## Log\n' > "$SPEC"

resolve_spec ZZ-1 2>/dev/null
check "a spec that names no folder gets one, and then names it" \
  '[ -f "$FAKE/$IDX" ] && grep -q "^vault: $TDIR\$" "$SPEC"'
rm "$FAKE/$IDX"; empty_reads 0; resolve_spec ZZ-1 2>/dev/null
check "a spec that names its folder never gets a second index" '[ ! -f "$FAKE/$IDX" ]'
sed '/^vault: /d' "$SPEC" > "$SPEC.t" && mv "$SPEC.t" "$SPEC"; resolve_spec ZZ-1 2>/dev/null   # the folder back, for the rest
open_note qa none "QA"; N=$(note qa); body="$tmp/body"; echo "the result" > "$body"

empty_reads 3; set_section "$N" Work "$body"
check "a section is written through three empty reads" \
  'grep -qx "the result" "$FAKE/$N" && grep -qx "## Feedback" "$FAKE/$N" && grep -q "^- .* — opened\$" "$FAKE/$N"'

cp "$FAKE/$N" "$tmp/before"; echo "another" > "$body"; empty_reads 99; set_section "$N" Work "$body" 2>/dev/null
check "a read that never recovers writes nothing and says so" '[ $? -ne 0 ] && empty_reads 0 && cmp -s "$FAKE/$N" "$tmp/before"'

: > "$body"; empty_reads 1; set_section "$N" Work "$body" 2>/dev/null
check "an empty body is refused before the note is read" '[ $? -ne 0 ] && [ "$(cat "$FAKE/.empty")" = 1 ] && cmp -s "$FAKE/$N" "$tmp/before"'
empty_reads 0; set_section "$N" Work "$tmp/missing" 2>/dev/null
check "a missing body is refused and writes nothing" '[ $? -ne 0 ] && cmp -s "$FAKE/$N" "$tmp/before"'

printf '\n  \n' > "$body"; empty_reads 0; set_section "$N" Work "$body" 2>/dev/null
check "a body of blank lines is refused and writes nothing" '[ $? -ne 0 ] && cmp -s "$FAKE/$N" "$tmp/before"'

had=$TDIR; TDIR=''; open_note qa none "QA" 2>/dev/null
check "no ticket folder opens no note at the vault's root" '[ $? -ne 0 ] && [ ! -f "$FAKE/05-qa.md" ]'
empty_reads 1; note_log "$(note qa 2>/dev/null)" "work written" 2>/dev/null
check "no ticket folder logs to no note, before anything is read" '[ $? -ne 0 ] && [ "$(cat "$FAKE/.empty")" = 1 ] && [ ! -f "$FAKE/05-qa.md" ]'
empty_reads 0; TDIR=$had

open_note developer balanced "Developer"; D=$(note developer)
empty_reads 2; section_line "$D" Critique "round 1: one should"; section_line "$D" Critique "round 2: green"
check "a section line lands at the end of its section, dated by the clock" \
  'awk "/^## Critique\$/{c=1;next} /^## /{c=0} c" "$FAKE/$D" | grep -v "^\$" | tr "\n" "|" | grep -qE "^- [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2} — round 1: one should\|- [^|]* — round 2: green\|\$" && grep -qx "## Feedback" "$FAKE/$D"'

VAULT_DIR="$FAKE"; empty_reads 99; c=$(vault_read "$N" '^## Log$' 2>/dev/null)
check "a note the CLI reads as nothing is read from the vault's folder on disk" '[ -n "$c" ] && printf "%s\n" "$c" | grep -qx "the result"'
empty_reads 0; VAULT_DIR="$tmp/no-vault-on-disk"

printf -- '- 2000-01-01 00:00 — %s\n' "work written" "feedback written" >> "$FAKE/$D"; empty_reads 2
check "a single cycle is finished, through empty reads" 'finished developer'

open_note architect hard "Architect"; for l in "work written" "feedback written" "work written"; do note_log "$(note architect)" "$l"; done
check "a cycle reopened after its feedback is not finished" '! finished architect'
for l in "work started: qa 1" "work written: qa 1" "feedback written: qa 1 red" "work started: qa 2" "work written: qa 2" "feedback written: qa 2 green"; do note_log "$N" "$l"; done
check "a second qa round whose feedback comes last is finished" 'finished qa'

cp "$SPEC" "$tmp/spec-before"; empty_reads 99; bump qa doing "gate started" 2>/dev/null
check "a refused bump leaves the spec as it was" '[ $? -ne 0 ] && empty_reads 0 && cmp -s "$SPEC" "$tmp/spec-before"'

bump qa doing "gate started"
check "a bump writes the spec, the index and where it stands" \
  'grep -qx "step: qa" "$SPEC" && grep -qx "step: qa" "$FAKE/$IDX" && grep -q " — gate started\$" "$FAKE/$IDX" && grep -q "^qa · doing — gate started" "$FAKE/$TDIR/summary.md" && grep -qx "## Decisions" "$FAKE/$TDIR/summary.md"'

rm "$FAKE/$TDIR/summary.md"; bump done done "closed"
check "a bump after the summary is deleted still closes" '[ $? -eq 0 ] && grep -qx "status: done" "$SPEC" && grep -q " — closed\$" "$FAKE/$IDX"'

move_card QA; move_card Done x
check "a card moves to one column and the board keeps its columns" \
  '[ "$(grep -c "ZZ-1" "$FAKE/$BOARD")" = 1 ] && grep -q "^- \[x\] .*ZZ-1" "$FAKE/$BOARD" && grep -qx "## Backlog" "$FAKE/$BOARD"'

rm -rf "$tmp"; [ "$fails" = 0 ] && echo "flow.sh: all checks passed" || { echo "flow.sh: $fails failed"; exit 1; }

