# 0001 — Bounded contexts, an event-driven core and vertical slices

- **Status:** proposed
- **Date:** 2026-11-01
- **Spec:** docs/specs/INK-1-ink-hub-foundation.md · **Decision point:** settled by the owner's brief; details D3–D5

## Context

Ink Hub is written from scratch (ADR 0003), taking its concepts from
dark-factory, whose shape — one flat `package main` of 128 files, a 17k-line
domain package, agents reached through tmux from 30 files — is the lesson in
what to avoid (`docs/architecture/inventory.md`). The owner has decided the
target shape: bounded contexts, an event-driven core, vertical slices, kept
light (an in-process bus and a JSONL log, no event-sourcing framework), files
first, a disposable index.

The product moves one question through a fixed pipeline: an agent asks, a page
is rendered and published to an e-ink reader, ink comes back, the answer is
recognized, confirmed when uncertain, written into the spec and delivered to
the agent. Each step has a different vocabulary and a different failure mode,
and each step is owned by exactly one process (the Go hub, the Python worker,
the reader).

## Decision

**Contexts.** Six bounded contexts and one platform layer:

| Context | Owns | Core types |
|---|---|---|
| `projects` | a repository, its `docs/specs/`, its orchestrator binding | Project, OrchestratorBinding |
| `inbox` (core) | the lifecycle of a question | Question, Answer, Decision, Revision |
| `pages` | what a question looks like on paper | PageType, Template, Zone, DeviceProfile, PageRevision |
| `ink` | what comes back from the reader | Device, SyncRoot, Artifact (PDF, `.mark`, PNG, strokes), OCRRun |
| `agents` | how an answer reaches an agent | Adapter (herdr, rex, headless), AgentRef, Delivery |
| `specflow` | the spec files on disk | SpecFile, ClarifyRound, LogLine, Summary |
| `platform` | plumbing every context uses | eventlog, bus, lock, config, webdav, auth, mdfile, contracts |

A context never imports another context's internals. Shared vocabulary crosses
only as events (below) and as JSON Schema contracts in `contracts/`.

**Events.** The main flow is
`QuestionAsked → PageRendered → PagePublished → InkReceived → AnswerRecognized → (ConfirmationRequested →) AnswerConfirmed → AnswerDelivered`,
plus `SpecChanged`, `AgentBlocked`, `ProjectRegistered`, and the failure
events each slice emits (`DeliveryFailed`, `AnswerRejected` for a stale
revision). Every event is one envelope:

```json
{"id":"…","type":"QuestionAsked","at":"2026-11-01T10:00:00.000Z","project":"shop",
 "subject":"question/01J…","rev":1,"data":{…},"causation_id":"…","correlation_id":"…"}
```

The log is the source of truth: one append-only JSONL file per project under
the hub's data directory (format in ADR 0005). The inbox and the project board
are **projections** held in memory by `inkd` and rebuilt from the log at start
(a project's log is kilobytes to a few megabytes; a replay is milliseconds), so
there is no database inside the daemon. Analytics over the same JSONL files —
the CER harness, the handwriting dataset, reports — run in DuckDB, which reads
JSONL natively, from the worker and the CLI (D9, pending the owner's
confirmation; ADR 0005). A test asserts for every projection that a replay
rebuilds it.

**Bus.** One in-process bus in `inkd`. `Publish` appends to the log first, then
fans out synchronously to subscribers in registration order; a subscriber's
error is logged and isolated, never propagated to the publisher. Handlers are
idempotent, because replay on start-up delivers every event again. There is no
broker, no retry queue and no at-least-once promise beyond the log itself.

**Slices.** One folder per use case under its context,
`internal/<context>/<slice>/`, holding `command.go` (the input), `handler.go`,
`events.go` (what it emits) and tests. The MVP slices are `registerproject`,
`askquestion`, `renderpage`, `publishpage`, `ingestink`, `recognizeanswer`,
`confirmanswer`, `deliveranswer`, `syncspec`, `watchspec` (a `## Clarify`
section with open answers in a watched spec file becomes `QuestionAsked`, the
brief's second question source), `detectblocked`, `renderinbox`.
A slice may import `internal/<its context>/domain`, `internal/platform/...`
and `contracts`; it never imports another slice or another context. A test in
`internal/platform/arch` lists the import graph with `go list -deps` and fails
on a violation, so the rule holds without a linter plugin.

**Revisions.** A Question's `rev` is the hash of its text and options; a Page
is one render of one question revision with its own `rev` (ADR 0005). An
answer names `page_id` and the page `rev`; an answer for a page whose question
revision is no longer current is recorded as `AnswerRejected` and never
delivered.

**Handwriting is data (brief §7).** Recognized text lives in `ocr.json` and,
once confirmed, in the answer event and the spec file. It never becomes an
instruction: the prompt `deliveranswer` sends through an adapter is the
question id and the `ink answer <id> --json` command plus one line of context,
and a test keeps it so. A question flagged `critical: true` (payment, deploy,
anything irreversible) is answered only from a `stamp` zone; free text on such
a page is recorded but is not an approval.

**Partial failure.** The worker (a local service `inkd` supervises, ADR
0007) writes into `<page-dir>/.run-<ulid>/` and renames into place on
success; a request that `inkd` abandons after `worker.timeout` leaves no
partial `ocr.json` and emits `RecognitionFailed`. The daemon's own
appends are `write` + `fsync` under one mutex; a crash mid-write leaves a
truncated last line that the reader skips (ADR 0005). `ingestink` deduplicates
on `page_id` + `rev` + the file's sha256, because readers and sync tools upload
the same bytes more than once.

## Alternatives

- **An event-sourcing framework** (aggregates, snapshots, a store with
  optimistic concurrency). Rejected by the brief: the hub has one writer and a
  handful of event types; a framework would be most of the code.
- **A layered architecture with shared services** (`service/`, `repository/`).
  Rejected: it is how `internal/tracker` grew to 17k lines with every feature
  touching the same package.
- **One context per process.** Rejected for the Go side: one daemon is simpler
  to run on a laptop; the Python worker is a separate process only because the
  models are Python.

## Consequences

- Some code is duplicated across slices on purpose (a struct here, a helper
  there). The rule is: extract into `platform` only after the third copy.
- Nothing here is ported: every slice is written from its contract and its
  tests (ADR 0003); the inventory names dark-factory's behaviour only as a
  reference for what the concept must do.
- Replay is the test strategy for projections: a slice test feeds events, the
  projection is compared with a golden.
- Anything the hub knows that is not in the log is a bug: every projection
  can be dropped and replayed at any time.
- The import-boundary test is the only enforcement; it has to exist from the
  first slice (Phase 1).
