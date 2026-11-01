# 0005 — Event log format and identifiers

- **Status:** proposed
- **Date:** 2026-11-01 (revised after the critique the same day)
- **Spec:** docs/specs/INK-1-ink-hub-foundation.md · **Decision points:** D3, D4 (both taken by the architect under brief §9, recorded as assumed)

## Context

The brief fixes the envelope
`{id, type, at, project, subject, rev, data, causation_id, correlation_id}`,
one JSONL file per project in the hub's data directory, and projections that
can be rebuilt from the log. It leaves open who may write the log, what an id
looks like and where projections live. dark-factory's six JSONL logs are the
lesson, not the seed (ADR 0003): no ids, no revisions, three time formats and
three bad-line policies (`docs/architecture/inventory.md` §6.1). The owner
added on 2026-11-01: "zamiast chrome qdrant albo duckdb" — read as "no
SQLite; DuckDB or Qdrant for the index", to be confirmed.

Identifiers appear in three places that constrain them: the log, the file
system (`pages/<page-id>/`) and the QR code printed on every page
(`question_id` + `rev`), which a phone-less e-ink round trip relies on.

## Decision

**File.** `<data>/projects/<project>/events.jsonl`, append-only, one JSON
object per line, no blank lines, UTF-8, `\n` terminated. A truncated last line
(a crash mid-write) is skipped on read and reported once; everything before it
is valid. No rotation in the MVP; a project's log is expected to stay under a
few MB.

**Envelope.** Exactly the brief's fields. `id` is the event's ULID; `at` is
RFC 3339 with milliseconds in UTC; `subject` is `<kind>/<id>` (`question/01J…`,
`page/01J…`); `rev` is the subject's revision the event applies to;
`causation_id` is the event that directly caused this one, `correlation_id`
the first event of the chain (the `QuestionAsked`). `data` is validated
against `contracts/event.schema.json` per `type`.

**Revisions.** A *Question* revision is the content hash of its text and
options: an agent that rewrites a question bumps `rev`; an agent that edits
another part of the spec file does not. A *Page* is one render of one question
revision and has its own `rev`; a confirmation page is a new page of the same
question. The QR carries the **page** id and revision; an answer names them;
the hub derives the question and its revision from the page. An answer to a
page whose question revision is no longer current is `AnswerRejected`. The hash
of the raw spec file at read time is a third thing, `spec_hash`, used only by
`syncspec` as the compare-and-swap token when it writes the file (ADR 0003).

**Identifiers (D4, decided by the architect; reversible).** ULIDs
(`github.com/oklog/ulid/v2`, Apache-2.0) for events, questions, pages, devices
and OCR runs: 26 characters, Crockford base32, uppercase, time-ordered. The QR
payload is `INK:<page_ulid>:<rev>`, uppercase throughout, so it fits QR
alphanumeric mode (digits, `A–Z`, space and `$%*+-./:`); a lowercase prefix
would force byte mode. Honest sizes: ULID gives a 33-character payload,
UUIDv7 printed as 32 uppercase hex digits would give 39, one QR version apart.
That is a taste difference, so the choice was not put to the owner.

**Single writer (D3, decided by the architect; forced by the brief's in-process bus).** Only `inkd` appends. The `ink` CLI talks
to `inkd` over a Unix socket (`$XDG_RUNTIME_DIR/inkhub.sock`, JSON over HTTP;
`--socket` / `INKHUB_SOCKET` to change) and never opens the log. When `inkd`
is not running, `ink ask` fails fast with exit code 3 and the line
`inkd is not running: start it with 'inkd' (or 'ink status')`. Appends are
`write` + `fsync` under one mutex in the daemon; the bus fans out only after
the fsync returned.

**Projections (D9, recommended, pending the owner's confirmation).** `inkd`
holds its projections (the inbox, the project board, the open deliveries) in
memory and rebuilds them from the logs at start; `ink inbox` asks the daemon.
No SQLite. **DuckDB** is the query engine over the same JSONL files for
everything analytical — the CER harness and the handwriting dataset in the
worker (`duckdb` Python package), reports and ad-hoc questions from the CLI
(`duckdb` CLI, or `github.com/duckdb/duckdb-go` later if `ink` needs it in
process; it is CGO). **Qdrant** is deferred: the one use in sight is
similarity search over handwriting crops to feed the OCR engine with the
owner's past corrections (Phase 4); if that proves useful, DuckDB's `vss`
extension is tried first, a Qdrant server only if it does not suffice.

## Alternatives

- **UUIDv7** (`github.com/google/uuid`): an RFC and time-ordered; printed as
  32 uppercase hex digits it is QR-alphanumeric too and six characters longer.
  Lost on taste and on the one dependency either way.
- **KSUID**: 27 characters, base62 (mixed case): byte mode unless re-encoded.
- **The spec file's hash as the question's `rev`**: considered and rejected,
  because any agent edit elsewhere in the file would reject every answer
  written minutes later on the reader.
- **Every process appends with `flock`** (D3 B): `ink` would work while
  `inkd` is down, but two writers need a lock protocol, the in-process bus
  loses its meaning, and `--wait` needs the daemon anyway.
- **A spool directory for `ink ask` while the daemon is down** (D3 C): more
  state to reconcile; deferred until there is evidence the daemon is often
  down.
- **SQLite as a disposable index** (the brief's first wording). Set aside by
  the owner on 2026-11-01 in favour of DuckDB or Qdrant; and once the index is
  rebuilt from the log at every start, an embedded database inside the daemon
  buys nothing at this scale.
- **Qdrant for the projections.** A vector store is the wrong tool for an
  inbox; kept only as a candidate for crop similarity in the worker.

## Consequences

- One new Go dependency (`oklog/ulid/v2`); the daemon has no database
  dependency at all. DuckDB enters through the worker's `pyproject.toml`
  first; a Go binding is a later ADR if `ink` needs it in process.
- The CLI depends on the daemon; `ink status` is the first thing a skill tells
  an agent to run when `ink ask` fails.
- Replaying a log is cheap enough to be the test oracle for every projection.
